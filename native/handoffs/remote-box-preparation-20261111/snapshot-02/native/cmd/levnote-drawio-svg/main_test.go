package main

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"testing"

	"levnote.local/native/core/engineipc"
	"levnote.local/native/formats/drawio"
)

func drawioRequest(t *testing.T, source []byte, page engineipc.RequestPage) []byte {
	t.Helper()
	prefix, err := engineipc.EncodeRequestPrefix(engineipc.RequestMeta{
		Engine: engineipc.DrawioSVG, Format: "svg", Page: &page,
	}, uint64(len(source)))
	if err != nil {
		t.Fatal(err)
	}
	return append(prefix, source...)
}

func decodeDrawioResponse(t *testing.T, response []byte, sourceBytes int) (engineipc.ResponseMeta, []byte) {
	t.Helper()
	decoder := engineipc.NewResponseDecoder(engineipc.DrawioSVG, engineVersion, uint64(sourceBytes))
	if _, err := decoder.Write(response); err != nil {
		t.Fatal(err)
	}
	meta, payload, err := decoder.Finish()
	if err != nil {
		t.Fatal(err)
	}
	return meta, payload
}

func callDrawioWorker(t *testing.T, source []byte, page engineipc.RequestPage) (engineipc.ResponseMeta, []byte) {
	t.Helper()
	var output bytes.Buffer
	if exit := serve(bytes.NewReader(drawioRequest(t, source, page)), &output); exit != 0 {
		t.Fatalf("serve exit%d", exit)
	}
	return decodeDrawioResponse(t, output.Bytes(), len(source))
}

func TestDrawioWorkerExportsExactSecondPageAndPreservesSource(t *testing.T) {
	source := []byte(`<mxfile><diagram id="first" name="One"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="one" value="one" vertex="1" parent="1"><mxGeometry x="0" y="0" width="40" height="20" as="geometry"/></mxCell></root></mxGraphModel></diagram><diagram id="second" name="Two"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="two" value="two" vertex="1" parent="1"><mxGeometry x="10" y="20" width="80" height="40" as="geometry"/></mxCell></root></mxGraphModel></diagram></mxfile>`)
	before := append([]byte(nil), source...)
	meta, svg := callDrawioWorker(t, source, engineipc.RequestPage{Index: 1, ID: "second"})
	if meta.Status != engineipc.StatusOK || meta.Error != nil || len(meta.Warnings) != 0 {
		t.Fatalf("success metadata: %+v error=%s", meta, workerError(meta))
	}
	const expected = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="-14 -4 128 88" width="128" height="88"><defs><marker id="levnote-arrow" markerWidth="10" markerHeight="7" refX="9" refY="3.5" orient="auto" markerUnits="strokeWidth"><path d="M0,0 L10,3.5 L0,7 z" fill="context-stroke"/></marker></defs><rect x="-14" y="-4" width="128" height="88" fill="#ffffff"/><g font-family="Arial, sans-serif" font-size="14"><rect x="10" y="20" width="80" height="40" fill="#dae8fc" stroke="#6c8ebf" stroke-width="1.5"/><text x="50" y="44.9" text-anchor="middle" fill="#111111"><tspan x="50" dy="0">two</tspan></text></g></svg>` + "\n"
	if string(svg) != expected {
		t.Fatalf("exact second-page SVG:\n%s\nwant:\n%s", svg, expected)
	}
	if !bytes.Equal(source, before) {
		t.Fatal("worker mutated source bytes")
	}
}

func TestDrawioWorkerWarningOrderAndScope(t *testing.T) {
	source := []byte(`<mxfile>junk<future/>` +
		`<diagram id="p0" name="Zero"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><future/><mxCell id="zero" value="&lt;b&gt;zero&lt;/b&gt;" style="html=1;" vertex="1" parent="1"><mxGeometry x="0" y="0" width="40" height="20" as="geometry"/></mxCell></root></mxGraphModel></diagram>` +
		`<diagram id="p1" name="One"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><future/>` +
		`<mxCell id="html" value="&lt;b&gt;bold&lt;/b&gt;" style="html=1;" vertex="1" parent="1"><mxGeometry x="10" y="10" width="80" height="40" as="geometry"/></mxCell>` +
		`<mxCell id="stencil" value="stencil" style="shape=mxgraph.flowchart.process;" vertex="1" parent="1"><mxGeometry x="110" y="10" width="80" height="40" as="geometry"/></mxCell>` +
		`</root></mxGraphModel></diagram></mxfile>`)
	meta, svg := callDrawioWorker(t, source, engineipc.RequestPage{Index: 1, ID: "p1"})
	if meta.Status != engineipc.StatusOK || meta.Error != nil || len(svg) == 0 {
		t.Fatalf("warning success metadata=%+v error=%s svg=%d", meta, workerError(meta), len(svg))
	}
	wantCodes := []string{
		"UNSUPPORTED_ELEMENT",
		"UNSUPPORTED_ELEMENT",
		"UNSUPPORTED_ELEMENT",
		"HTML_LABEL",
		"STENCIL",
		"HTML_LABEL",
		"STENCIL",
	}
	if len(meta.Warnings) != len(wantCodes) {
		t.Fatalf("warning count=%d warnings=%+v", len(meta.Warnings), meta.Warnings)
	}
	for index, code := range wantCodes {
		if meta.Warnings[index].Code != code || meta.Warnings[index].Start != 0 || meta.Warnings[index].End != uint32(len(source)) {
			t.Fatalf("warning %d=%+v want code=%s whole source", index, meta.Warnings[index], code)
		}
	}
	wantPrefixes := []string{
		"document: non-whitespace",
		"document: unknown mxfile child",
		"page 1: cell wrapper <future>",
		"page 1 cell html:",
		"page 1 cell stencil:",
		"page 1 cell html:",
		"page 1 cell stencil:",
	}
	for index, prefix := range wantPrefixes {
		if !strings.HasPrefix(meta.Warnings[index].Message, prefix) {
			t.Fatalf("warning %d message=%q want prefix %q", index, meta.Warnings[index].Message, prefix)
		}
	}
	if strings.Contains(fmt.Sprint(meta.Warnings), "page 0") || strings.Contains(fmt.Sprint(meta.Warnings), "page 0 cell zero") {
		t.Fatal("unselected page warnings leaked")
	}
}

func TestDrawioWorkerInvalidPageIDAndIndexRefuseWithoutPayload(t *testing.T) {
	source := []byte(`<mxfile><diagram id="only" name="Only"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="n" value="x" vertex="1" parent="1"><mxGeometry x="1" y="2" width="3" height="4" as="geometry"/></mxCell></root></mxGraphModel></diagram></mxfile>`)
	for _, page := range []engineipc.RequestPage{
		{Index: 1, ID: "only"},
		{Index: 0, ID: "wrong"},
		{Index: 0, ID: ""},
	} {
		meta, payload := callDrawioWorker(t, source, page)
		if meta.Status != engineipc.StatusError || meta.Error == nil || meta.Error.Code != "invalid-page" ||
			len(meta.Warnings) != 0 || payload != nil {
			t.Fatalf("page=%+v meta=%+v error=%s payload=%v", page, meta, workerError(meta), payload)
		}
	}
}

func TestDrawioWorkerCompressedReadOnlySuccess(t *testing.T) {
	model := `<mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="zip" value="压缩" vertex="1" parent="1"><mxGeometry x="5" y="6" width="100" height="40" as="geometry"/></mxCell></root></mxGraphModel>`
	var compressed bytes.Buffer
	writer, err := flate.NewWriter(&compressed, flate.DefaultCompression)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte(model)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	payload := base64.StdEncoding.EncodeToString(compressed.Bytes())
	source := []byte(`<mxfile><diagram id="zip" name="Zip">` + payload + `</diagram></mxfile>`)
	meta, svg := callDrawioWorker(t, source, engineipc.RequestPage{Index: 0, ID: "zip"})
	if meta.Status != engineipc.StatusOK || meta.Error != nil || !bytes.Contains(svg, []byte("压缩")) {
		t.Fatalf("compressed read-only export: meta=%+v error=%s svg=%s", meta, workerError(meta), svg)
	}
	if len(meta.Warnings) != 1 || meta.Warnings[0].Code != string(drawio.IssueCompressedReadOnly) ||
		meta.Warnings[0].Message != "page 0: compressed diagram is rendered read-only because its outer base64/deflate payload is preserved verbatim" {
		t.Fatalf("compressed warning=%+v", meta.Warnings)
	}
}

func TestDrawioWorkerOutputAndWarningLimitsReturnNilPayload(t *testing.T) {
	source := []byte(`<mxfile><diagram id="p" name="P"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="n" value="x" vertex="1" parent="1"><mxGeometry x="1" y="2" width="30" height="40" as="geometry"/></mxCell></root></mxGraphModel></diagram></mxfile>`)
	var output bytes.Buffer
	if exit := serveWithLimit(bytes.NewReader(drawioRequest(t, source, engineipc.RequestPage{Index: 0, ID: "p"})), &output, 1); exit != 0 {
		t.Fatalf("output-limit exit=%d", exit)
	}
	meta, payload := decodeDrawioResponse(t, output.Bytes(), len(source))
	if meta.Error == nil || meta.Error.Code != "output-limit" || len(meta.Warnings) != 0 || payload != nil {
		t.Fatalf("output-limit meta=%+v error=%s payload=%v", meta, workerError(meta), payload)
	}

	excess := []byte(`<mxfile>` + strings.Repeat("<future/>", engineipc.WarningsMaxCount+1) +
		`<diagram id="p" name="P"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/></root></mxGraphModel></diagram></mxfile>`)
	meta, payload = callDrawioWorker(t, excess, engineipc.RequestPage{Index: 0, ID: "p"})
	if meta.Error == nil || meta.Error.Code != "warning-limit" || len(meta.Warnings) != 0 || payload != nil {
		t.Fatalf("warning-limit meta=%+v error=%s payload=%v", meta, workerError(meta), payload)
	}
}

func TestDrawioWorkerWarningComponentBounds(t *testing.T) {
	base := drawio.Issue{Severity: drawio.SeverityWarning, Code: "CODE", Message: "message", Page: 0, CellID: "id"}
	if got, err := wireWarnings(10, engineipc.WarningsMaxCount, base); err != nil || got[0].Message != "page 0 cell id: message" {
		t.Fatalf("base warning=%+v err=%v", got, err)
	}
	for name, issue := range map[string]drawio.Issue{
		"long-code":    {Severity: drawio.SeverityWarning, Code: drawio.IssueCode(strings.Repeat("c", engineipc.WarningCodeMaxBytes+1)), Message: "x", Page: 0},
		"long-message": {Severity: drawio.SeverityWarning, Code: "C", Message: strings.Repeat("m", engineipc.WarningMessageMaxBytes+1), Page: 0},
		"long-cell":    {Severity: drawio.SeverityWarning, Code: "C", Message: "x", Page: 0, CellID: strings.Repeat("i", engineipc.WarningMessageMaxBytes+1)},
		"prefix-total": {Severity: drawio.SeverityWarning, Code: "C", Message: strings.Repeat("m", 500), Page: 0, CellID: strings.Repeat("i", 20)},
		"non-warning":  {Severity: drawio.SeverityError, Code: "C", Message: "x", Page: 0},
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := wireWarnings(10, engineipc.WarningsMaxCount, issue); err == nil || got != nil {
				t.Fatalf("bad warning accepted: %+v %v", got, err)
			}
		})
	}
	exact := make([]drawio.Issue, engineipc.WarningsMaxCount, engineipc.WarningsMaxCount+1)
	for index := range exact {
		exact[index] = base
	}
	if got, err := wireWarnings(10, engineipc.WarningsMaxCount, exact...); err != nil || len(got) != engineipc.WarningsMaxCount {
		t.Fatalf("exact warning count refused: %d %v", len(got), err)
	}
	if got, err := wireWarnings(10, engineipc.WarningsMaxCount, append(exact, base)...); err == nil || got != nil {
		t.Fatalf("excess warning count accepted: %d %v", len(got), err)
	}
}

func TestDrawioWorkerFiltersLargeUnselectedWarningPopulation(t *testing.T) {
	source := []byte(`<mxfile><diagram id="zero"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/>` +
		strings.Repeat("<future/>", 4096) +
		`</root></mxGraphModel></diagram><diagram id="one"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/>` +
		`<mxCell id="selected" value="selected" vertex="1" parent="1"><mxGeometry x="1" y="2" width="30" height="40" as="geometry"/></mxCell>` +
		`</root></mxGraphModel></diagram></mxfile>`)
	svg, warnings, engineErr := render(source, engineipc.RequestPage{Index: 1, ID: "one"})
	if engineErr != nil || len(svg) == 0 || len(warnings) != 0 {
		t.Fatalf("filtered export err=%v svg=%d warnings=%d", engineErr, len(svg), len(warnings))
	}
	if strings.Contains(fmt.Sprint(warnings), "page 0") {
		t.Fatalf("page zero warnings leaked: %+v", warnings)
	}
}

func TestDrawioWorkerCombinedWarningBound(t *testing.T) {
	source := func(extra int) []byte {
		return []byte(`<mxfile><diagram id="p"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/>` +
			strings.Repeat("<future/>", 252+extra) +
			`<mxCell id="html" value="&lt;b&gt;bold&lt;/b&gt;" style="html=1;" vertex="1" parent="1"><mxGeometry x="10" y="10" width="80" height="40" as="geometry"/></mxCell>` +
			`<mxCell id="stencil" value="stencil" style="shape=mxgraph.flowchart.process;" vertex="1" parent="1"><mxGeometry x="110" y="10" width="80" height="40" as="geometry"/></mxCell>` +
			`</root></mxGraphModel></diagram></mxfile>`)
	}
	svg, warnings, engineErr := render(source(0), engineipc.RequestPage{Index: 0, ID: "p"})
	if engineErr != nil || len(svg) == 0 || len(warnings) != engineipc.WarningsMaxCount {
		t.Fatalf("exact combined warning bound err=%v svg=%d warnings=%d", engineErr, len(svg), len(warnings))
	}
	if warnings[len(warnings)-2].Code != string(drawio.IssueHTMLLabel) ||
		warnings[len(warnings)-1].Code != string(drawio.IssueStencil) {
		t.Fatalf("emitter warning order changed: %+v", warnings[len(warnings)-2:])
	}

	svg, warnings, engineErr = render(source(1), engineipc.RequestPage{Index: 0, ID: "p"})
	if engineErr == nil || engineErr.Code != "warning-limit" || svg != nil || warnings == nil || len(warnings) != 0 {
		t.Fatalf("combined excess err=%+v svg=%v warnings=%+v", engineErr, svg != nil, warnings)
	}
}

func TestDrawioWorkerRelevantWarningCountCheckedBeforeConversion(t *testing.T) {
	issues := make([]drawio.Issue, engineipc.WarningsMaxCount+1)
	for index := range issues {
		issues[index] = drawio.Issue{Severity: drawio.SeverityWarning, Code: "C", Message: "x", Page: 0}
	}
	if got, err := wireParseWarnings(10, 0, engineipc.WarningsMaxCount, issues); err == nil || got != nil {
		t.Fatalf("257 relevant warnings converted: %d %v", len(got), err)
	}
	if got, err := wireWarnings(10, 0, issues[0]); err == nil || got != nil {
		t.Fatalf("standalone helper ignored zero capacity: %d %v", len(got), err)
	}
}

func TestDrawioWorkerProtocolErrorsRemainCoherent(t *testing.T) {
	source := []byte(`<mxfile><diagram id="p"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/></root></mxGraphModel></diagram></mxfile>`)
	valid := drawioRequest(t, source, engineipc.RequestPage{Index: 0, ID: "p"})
	for name, request := range map[string][]byte{
		"truncated": valid[:len(valid)-1],
		"utf8":      append(append([]byte(nil), valid[:len(valid)-1]...), 0xff),
		"extra":     append(append([]byte(nil), valid...), 0),
	} {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			if exit := serve(bytes.NewReader(request), &output); exit != 0 {
				t.Fatalf("coherent protocol failure exit=%d", exit)
			}
			meta, payload := decodeDrawioResponse(t, output.Bytes(), len(source))
			if meta.Status != engineipc.StatusError || meta.Error == nil || meta.Error.Code != "invalid-request" && meta.Error.Code != "invalid-utf8" ||
				len(meta.Warnings) != 0 || payload != nil {
				t.Fatalf("protocol error meta=%+v error=%s payload=%v", meta, workerError(meta), payload)
			}
		})
	}
}

type countingDrawioReader struct {
	data  []byte
	reads int
}

func (reader *countingDrawioReader) Read(buffer []byte) (int, error) {
	reader.reads++
	if len(reader.data) == 0 {
		return 0, io.EOF
	}
	count := copy(buffer, reader.data)
	reader.data = reader.data[count:]
	return count, nil
}

func TestDrawioWorkerForeignRequestDoesNotReadPayload(t *testing.T) {
	const metadata = `{"engine":"levnote.markdown-export","format":"html"}`
	request := frameForWorker(metadata, []byte("source"))
	reader := &countingDrawioReader{data: request}
	var output bytes.Buffer
	if exit := serve(reader, &output); exit != 0 {
		t.Fatalf("foreign request exit=%d", exit)
	}
	meta, payload := decodeDrawioResponse(t, output.Bytes(), 0)
	if meta.Error == nil || meta.Error.Code != "invalid-request" || len(meta.Warnings) != 0 || payload != nil ||
		reader.reads != 2 || string(reader.data) != "source" {
		t.Fatalf("foreign request meta=%+v error=%s payload=%v reads=%d remaining=%q", meta, workerError(meta), payload, reader.reads, reader.data)
	}
}

func workerError(meta engineipc.ResponseMeta) string {
	if meta.Error == nil {
		return "<nil>"
	}
	return meta.Error.Code + ": " + meta.Error.Message
}

func frameForWorker(metadata string, payload []byte) []byte {
	prefix := []byte("LVEXREQ\x00")
	prefix = append(prefix, 1, 0, 24, 0)
	value := uint32(len(metadata))
	for range 4 {
		prefix = append(prefix, byte(value))
		value >>= 8
	}
	size := uint64(len(payload))
	for range 8 {
		prefix = append(prefix, byte(size))
		size >>= 8
	}
	return append(append(prefix, metadata...), payload...)
}
