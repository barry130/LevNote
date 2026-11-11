package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"levnote.local/native/core/document"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/formats/markdown"
)

func callWorker(t *testing.T, source []byte) (engineipc.ResponseMeta, []byte) {
	t.Helper()
	prefix, err := engineipc.EncodeRequestPrefix(engineipc.RequestMeta{Engine: engineipc.MarkdownHTML, Format: "html"}, uint64(len(source)))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if exit := serve(bytes.NewReader(append(prefix, source...)), &output); exit != 0 {
		t.Fatalf("serve exit%d", exit)
	}
	d := engineipc.NewResponseDecoder(engineipc.MarkdownHTML, engineVersion, uint64(len(source)))
	if _, err := d.Write(output.Bytes()); err != nil {
		t.Fatal(err)
	}
	meta, html, err := d.Finish()
	if err != nil {
		t.Fatal(err)
	}
	return meta, html
}

func TestWorkerParserCapAndParity(t *testing.T) {
	if engineipc.SourceMaxBytes != markdown.MaxParseBytes {
		t.Fatal("protocol/parser source cap drift")
	}
	fixtures := []string{"", "plain", "# 字😊\r\n\r\n- [x] done\r\n", "\ufeffliteral BOM content\n", "| A | B |\n| --- | --- |\n| **字面** | <tag> |\n", "<script>alert(1)</script>\n", "[safe](local.md) [bad](https://example.com)", "```doc-block\nid: abc\nkind: markdown\nmode: inline\n---\nhello\n```\n"}
	for _, source := range fixtures {
		meta, html := callWorker(t, []byte(source))
		want, warnings, err := markdown.ExportHTMLChecked(markdown.Parse(source))
		if err != nil {
			t.Fatal(err)
		}
		converted, err := wireWarnings(warnings, len(source))
		if err != nil {
			t.Fatal(err)
		}
		if meta.Status != engineipc.StatusOK || string(html) != want || !reflect.DeepEqual(meta.Warnings, converted) {
			t.Fatalf("parity source=%q meta=%+v html=%q", source, meta, html)
		}
		if strings.Contains(source, "<script>") && bytes.Contains(html, []byte("<script>")) {
			t.Fatal("raw HTML executed")
		}
	}
	meta, html := callWorker(t, []byte("hello <&"))
	if meta.Status != engineipc.StatusOK || !bytes.Contains(html, []byte("<p>hello &lt;&amp;</p>")) {
		t.Fatalf("independent HTML oracle: %s", html)
	}
}

func TestWorkerPartialAndRequestFailureMapping(t *testing.T) {
	meta, html := callWorker(t, []byte(strings.Repeat("x\n", 65537)))
	if meta.Status != engineipc.StatusError || meta.Error.Code != "partial-document" || len(html) != 0 {
		t.Fatalf("partial accepted %+v", meta)
	}
	valid, _ := engineipc.EncodeRequestPrefix(engineipc.RequestMeta{Engine: engineipc.MarkdownHTML, Format: "html"}, 1)
	for _, test := range []struct {
		name    string
		request []byte
		code    string
	}{
		{"invalidUTF8", append(append([]byte{}, valid...), 0xff), "invalid-utf8"},
		{"truncated", valid, "invalid-request"},
		{"trailing", append(append([]byte{}, valid...), 'x', 'y'), "invalid-request"},
	} {
		t.Run(test.name, func(t *testing.T) { checkError(t, test.request, test.code) })
	}
	oversize := append([]byte{}, valid...)
	binary.LittleEndian.PutUint64(oversize[16:24], engineipc.SourceMaxBytes+1)
	checkError(t, oversize, "source-limit")
}

func TestWorkerExactSourceCapWithMaximumHTMLEscaping(t *testing.T) {
	// A complete, single fence avoids the separate inline/line/block limits.
	// Ampersands each expand to five HTML bytes, the maximum for EscapeString.
	const envelope = "```\n\n```\n"
	count := engineipc.SourceMaxBytes - len(envelope)
	source := []byte("```\n" + strings.Repeat("&", count) + "\n```\n")
	if len(source) != engineipc.SourceMaxBytes {
		t.Fatal("fixture cap drift")
	}
	meta, html := callWorker(t, source)
	prefix := []byte("<!doctype html>\n<meta charset=\"utf-8\">\n<body>\n<pre><code>")
	suffix := []byte("\n</code></pre>\n</body>\n")
	if meta.Status != engineipc.StatusOK || len(meta.Warnings) != 0 || len(html) != len(prefix)+5*count+len(suffix) ||
		!bytes.HasPrefix(html, prefix) || !bytes.HasSuffix(html, suffix) || bytes.Count(html, []byte("&amp;")) != count {
		t.Fatalf("exact cap export: status=%s warnings=%d bytes=%d", meta.Status, len(meta.Warnings), len(html))
	}
}

func TestWorkerDecodedFileBOMAndLiteralBOMContent(t *testing.T) {
	for _, test := range []struct {
		name string
		raw  []byte
	}{
		{"UTF8", append([]byte{0xef, 0xbb, 0xbf}, []byte("hello <&\r\n")...)},
		{"UTF16LE", []byte{0xff, 0xfe, 'h', 0, 'e', 0, 'l', 0, 'l', 0, 'o', 0, ' ', 0, '<', 0, '&', 0, '\r', 0, '\n', 0}},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "decoded.md")
			if err := os.WriteFile(path, test.raw, 0600); err != nil {
				t.Fatal(err)
			}
			doc, err := document.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			var decoded bytes.Buffer
			if _, err := doc.Buffer.Snapshot().WriteTo(&decoded); err != nil {
				t.Fatal(err)
			}
			if !doc.BOM || decoded.String() != "hello <&\r\n" {
				t.Fatalf("document decoding: BOM=%v bytes=%q", doc.BOM, decoded.Bytes())
			}
			meta, html := callWorker(t, decoded.Bytes())
			if meta.Status != engineipc.StatusOK || !bytes.Contains(html, []byte("<p>hello &lt;&amp;</p>")) || bytes.Contains(html, []byte("\ufeff")) {
				t.Fatal("encoding BOM entered worker source")
			}
		})
	}
	meta, html := callWorker(t, []byte("\ufeffliteral content"))
	if meta.Status != engineipc.StatusOK || !bytes.Contains(html, []byte("<p>\ufeffliteral content</p>")) {
		t.Fatal("literal buffer BOM content was stripped")
	}
}

func checkError(t *testing.T, request []byte, code string) {
	t.Helper()
	var output bytes.Buffer
	if exit := serve(bytes.NewReader(request), &output); exit != 0 {
		t.Fatalf("coherent error exit%d", exit)
	}
	d := engineipc.NewResponseDecoder(engineipc.MarkdownHTML, engineVersion, 0)
	if _, err := d.Write(output.Bytes()); err != nil {
		t.Fatal(err)
	}
	m, html, err := d.Finish()
	if err != nil || m.Status != engineipc.StatusError || m.Error.Code != code || len(html) != 0 {
		t.Fatalf("error mapping %+v %v", m, err)
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func TestWorkerWriteFailureAndWarningValidation(t *testing.T) {
	if serve(strings.NewReader("bad"), brokenWriter{}) == 0 {
		t.Fatal("write failure exited0")
	}
	for _, warning := range []markdown.Warning{
		{Code: "x", Range: markdown.SourceRange{Start: -1, End: 0}},
		{Code: "x", Range: markdown.SourceRange{Start: 0, End: 2}},
		{Code: "x", Message: strings.Repeat("x", 513)},
		{Code: strings.Repeat("x", 65)},
	} {
		if _, err := wireWarnings([]markdown.Warning{warning}, 1); err == nil {
			t.Fatal("bad warning accepted")
		}
	}
	if _, _, err := render([]byte{0xff}); err == nil || err.Code != "invalid-utf8" {
		t.Fatal("invalid payload")
	}
}

func TestWorkerTinyOutputBudgetFraming(t *testing.T) {
	const source = "字 <&>"
	const expected = "<!doctype html>\n<meta charset=\"utf-8\">\n<body>\n<p>字 &lt;&amp;&gt;</p>\n</body>\n"
	prefix, err := engineipc.EncodeRequestPrefix(engineipc.RequestMeta{Engine: engineipc.MarkdownHTML, Format: "html"}, uint64(len(source)))
	if err != nil {
		t.Fatal(err)
	}
	for _, limit := range []int{len(expected), len(expected) - 1, 0, -1} {
		var output bytes.Buffer
		if exit := serveWithLimit(bytes.NewReader(append(append([]byte{}, prefix...), source...)), &output, limit); exit != 0 {
			t.Fatalf("coherent response exit%d cap%d", exit, limit)
		}
		decoder := engineipc.NewResponseDecoder(engineipc.MarkdownHTML, engineVersion, uint64(len(source)))
		if _, err := decoder.Write(output.Bytes()); err != nil {
			t.Fatal(err)
		}
		meta, payload, err := decoder.Finish()
		if err != nil {
			t.Fatal(err)
		}
		if limit == len(expected) {
			if meta.Status != engineipc.StatusOK || meta.Error != nil || len(meta.Warnings) != 0 || string(payload) != expected {
				t.Fatalf("exact cap failed: meta=%+v payload=%q", meta, payload)
			}
		} else if meta.Status != engineipc.StatusError || meta.Error == nil || meta.Error.Code != "output-limit" || len(payload) != 0 || len(meta.Warnings) != 0 || binary.LittleEndian.Uint64(output.Bytes()[16:24]) != 0 {
			t.Fatalf("cap%d partial output: meta=%+v bytes=%d", limit, meta, len(payload))
		}
	}
	// The production render result is the owned renderer slice, not a copied
	// string conversion or capacity-grown buffer.
	payload, _, failed := render([]byte(source))
	if failed != nil || string(payload) != expected || cap(payload) != len(payload) {
		t.Fatalf("owned production result: cap/len=%d/%d error=%v", cap(payload), len(payload), failed)
	}
	// Parser refusal retains precedence over a tiny output budget.
	payload, warnings, failed := renderWithLimit([]byte(strings.Repeat("x\n", 65537)), 0)
	if failed == nil || failed.Code != "partial-document" || payload != nil || len(warnings) == 0 {
		t.Fatal("partial document was hidden by output budget")
	}
}
