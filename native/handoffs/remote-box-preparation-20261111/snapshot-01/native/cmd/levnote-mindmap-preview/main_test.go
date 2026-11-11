package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/core/enginescene"
	"levnote.local/native/formats/mindmap"
)

const simpleSource = `{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"R"}}`

func previewRequest(t *testing.T, source []byte) []byte {
	t.Helper()
	prefix, err := engineipc.EncodeRequestPrefix(engineipc.RequestMeta{
		Engine: engineipc.MindmapPreview, Format: "mindmap-scene-v1",
	}, uint64(len(source)))
	if err != nil {
		t.Fatal(err)
	}
	return append(prefix, source...)
}

func decodePreviewResponse(t *testing.T, data []byte, sourceBytes int) (engineipc.ResponseMeta, []byte) {
	t.Helper()
	decoder := engineipc.NewResponseDecoder(engineipc.MindmapPreview, engineVersion, uint64(sourceBytes))
	if _, err := decoder.Write(data); err != nil {
		t.Fatal(err)
	}
	meta, payload, err := decoder.Finish()
	if err != nil {
		t.Fatal(err)
	}
	if meta.Engine != engineipc.MindmapPreview || meta.Format != "mindmap-scene-v1" || meta.EngineVersion != engineVersion {
		t.Fatalf("response identity: %+v", meta)
	}
	return meta, payload
}

func callPreviewWorker(t *testing.T, source []byte) (engineipc.ResponseMeta, []byte) {
	t.Helper()
	var output bytes.Buffer
	if exit := serve(bytes.NewReader(previewRequest(t, source)), &output); exit != 0 {
		t.Fatalf("serve exit=%d", exit)
	}
	return decodePreviewResponse(t, output.Bytes(), len(source))
}

func requireRefusal(t *testing.T, source []byte, code string) {
	t.Helper()
	var output bytes.Buffer
	if exit := serve(bytes.NewReader(previewRequest(t, source)), &output); exit != 0 {
		t.Fatalf("coherent refusal exit=%d", exit)
	}
	meta, payload := decodePreviewResponse(t, output.Bytes(), len(source))
	if meta.Status != engineipc.StatusError || meta.Error == nil || meta.Error.Code != code ||
		len(meta.Warnings) != 0 || len(payload) != 0 || binary.LittleEndian.Uint64(output.Bytes()[16:24]) != 0 {
		t.Fatalf("refusal want=%q meta=%+v payload=%d", code, meta, len(payload))
	}
	if strings.Contains(meta.Error.Message, "SECRET_SOURCE") {
		t.Fatal("source text echoed in error")
	}
}

func TestPreviewWorkerLabelAndIDBudgets(t *testing.T) {
	for _, tc := range []struct {
		name, sourceTopic, label string
		warning                  bool
	}{
		{"exact", strings.Repeat("x", 512), strings.Repeat("x", 512), false},
		{"ascii-plus-one", strings.Repeat("x", 513), strings.Repeat("x", 512) + "…", true},
		{"unicode-boundary", strings.Repeat("x", 511) + "中", strings.Repeat("x", 511) + "…", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(`{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"` + tc.sourceTopic + `"}}`)
			before := bytes.Clone(source)
			meta, payload := callPreviewWorker(t, source)
			scene, err := enginescene.Decode(payload)
			if meta.Status != engineipc.StatusOK || err != nil || len(scene.Nodes) != 1 || scene.Nodes[0].Label != tc.label {
				t.Fatalf("label scene=%+v meta=%+v err=%v", scene, meta, err)
			}
			if tc.warning {
				if len(meta.Warnings) != 1 || meta.Warnings[0].Code != "LABEL_TRUNCATED" || meta.Warnings[0].End != uint32(len(source)) {
					t.Fatalf("truncation warning=%+v", meta.Warnings)
				}
			} else if len(meta.Warnings) != 0 {
				t.Fatalf("exact label falsely warned: %+v", meta.Warnings)
			}
			if !bytes.Equal(source, before) {
				t.Fatal("truncated display mutated source")
			}
		})
	}
	for _, size := range []int{512, 513} {
		source := []byte(`{"format":"levnote.mindmap","version":1,"root":{"id":"` + strings.Repeat("i", size) + `","topic":"R"}}`)
		if size == 513 {
			requireRefusal(t, source, "output-limit")
			continue
		}
		meta, payload := callPreviewWorker(t, source)
		scene, err := enginescene.Decode(payload)
		if meta.Status != engineipc.StatusOK || err != nil || len(scene.Nodes) != 1 || scene.Nodes[0].ID != strings.Repeat("i", 512) {
			t.Fatalf("exact ID refused or truncated: meta=%+v scene=%+v err=%v", meta, scene, err)
		}
	}
}

func TestPreviewWorkerInvalidSchemaAndValidDeepSource(t *testing.T) {
	for name, source := range map[string]string{
		"json":         `{SECRET_SOURCE`,
		"format":       `{"format":"SECRET_SOURCE","version":1,"root":{"id":"r","topic":"R"}}`,
		"version":      `{"format":"levnote.mindmap","version":0,"root":{"id":"r","topic":"R"}}`,
		"root":         `{"format":"levnote.mindmap","version":1,"root":null}`,
		"child":        `{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"R","children":[7]}}`,
		"duplicate-id": `{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"R","children":[{"id":"r","topic":"A"}]}}`,
	} {
		t.Run(name, func(t *testing.T) { requireRefusal(t, []byte(source), "render-failed") })
	}
	node := `{"id":"leaf","topic":"R"}`
	for i := 0; i < 257; i++ {
		node = fmt.Sprintf(`{"id":"n%d","topic":"R","children":[%s]}`, i, node)
	}
	deep := []byte(`{"format":"levnote.mindmap","version":1,"root":` + node + `}`)
	if !json.Valid(deep) {
		t.Fatal("deep source fixture is invalid JSON")
	}
	requireRefusal(t, deep, "source-limit")
}

func TestPreviewWorkerWarningAdmission(t *testing.T) {
	base := mindmap.Issue{Severity: mindmap.SeverityWarning, Code: "WARNING", Message: "fine"}
	exact := make([]mindmap.Issue, 256)
	for i := range exact {
		exact[i] = base
	}
	warnings, err := wireWarnings(7, exact)
	if err != nil || len(warnings) != 256 || warnings[255] != (engineipc.Warning{Code: "WARNING", Message: "fine", End: 7}) {
		t.Fatalf("exact warning count=%d err=%v", len(warnings), err)
	}
	for name, issues := range map[string][]mindmap.Issue{
		"count-plus-one":       append(append([]mindmap.Issue{}, exact...), base),
		"code-empty":           {{Severity: mindmap.SeverityWarning}},
		"code-too-long":        {{Severity: mindmap.SeverityWarning, Code: mindmap.IssueCode(strings.Repeat("c", 65))}},
		"message-too-long":     {{Severity: mindmap.SeverityWarning, Code: "W", Message: strings.Repeat("m", 513)}},
		"code-invalid-utf8":    {{Severity: mindmap.SeverityWarning, Code: mindmap.IssueCode(string([]byte{0xff}))}},
		"message-invalid-utf8": {{Severity: mindmap.SeverityWarning, Code: "W", Message: string([]byte{0xff})}},
		"severity":             {{Severity: mindmap.SeverityError, Code: "W"}},
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := wireWarnings(1, issues); err == nil || got != nil {
				t.Fatalf("bad warning accepted: %+v err=%v", got, err)
			}
		})
	}
	for _, size := range []int{-1, int(uint64(^uint32(0)) + 1)} {
		if got, err := wireWarnings(size, []mindmap.Issue{base}); err == nil || got != nil {
			t.Fatalf("bad warning range accepted: size=%d got=%+v err=%v", size, got, err)
		}
	}
	exactText := mindmap.Issue{Severity: mindmap.SeverityWarning, Code: mindmap.IssueCode(strings.Repeat("c", 64)), Message: strings.Repeat("m", 512)}
	if got, err := wireWarnings(1, []mindmap.Issue{exactText}); err != nil || len(got) != 1 || got[0].Code != string(exactText.Code) || got[0].Message != exactText.Message {
		t.Fatalf("exact warning text budgets: %+v err=%v", got, err)
	}
	var source strings.Builder
	source.WriteString(`{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"R","future":1,"children":[`)
	for i := 0; i < 256; i++ {
		if i > 0 {
			source.WriteByte(',')
		}
		fmt.Fprintf(&source, `{"id":"n%d","topic":"N","future":1}`, i)
	}
	source.WriteString(`]}}`)
	requireRefusal(t, []byte(source.String()), "warning-limit")
	longLayout := []byte(`{"format":"levnote.mindmap","version":1,"layout":"` + strings.Repeat("a", 513) + `","root":{"id":"r","topic":"R"}}`)
	requireRefusal(t, longLayout, "warning-limit")
}

type trackedReader struct {
	data            []byte
	reads, consumed int
}

func (r *trackedReader) Read(p []byte) (int, error) {
	r.reads++
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	r.consumed += n
	return n, nil
}

func literalRequest(meta string, length uint64, payload []byte) []byte {
	header := append([]byte("LVEXREQ\x00"), 1, 0, 24, 0)
	header = binary.LittleEndian.AppendUint32(header, uint32(len(meta)))
	header = binary.LittleEndian.AppendUint64(header, length)
	return append(append(header, meta...), payload...)
}

func TestPreviewWorkerRequestRefusalsBeforePayload(t *testing.T) {
	for _, meta := range []string{
		`{"engine":"levnote.markdown-export","format":"html"}`,
		`{"engine":"levnote.mindmap-svg","format":"svg"}`,
		`{"engine":"levnote.mindmap-preview","format":"svg"}`,
		`{"engine":"levnote.mindmap-svg","format":"mindmap-scene-v1"}`,
	} {
		reader := &trackedReader{data: literalRequest(meta, 6, []byte("source"))}
		var output bytes.Buffer
		if exit := serve(reader, &output); exit != 0 {
			t.Fatalf("foreign refusal exit=%d", exit)
		}
		response, payload := decodePreviewResponse(t, output.Bytes(), 0)
		if response.Status != engineipc.StatusError || response.Error == nil || response.Error.Code != "invalid-request" || len(payload) != 0 ||
			reader.consumed != 24+len(meta) || string(reader.data) != "source" {
			t.Fatalf("foreign frame read payload: meta=%+v consumed=%d remaining=%q", response, reader.consumed, reader.data)
		}
	}
	const meta = `{"engine":"levnote.mindmap-preview","format":"mindmap-scene-v1"}`
	reader := &trackedReader{data: literalRequest(meta, 16<<20+1, []byte("not-read"))}
	var output bytes.Buffer
	if exit := serve(reader, &output); exit != 0 {
		t.Fatalf("source limit exit=%d", exit)
	}
	response, payload := decodePreviewResponse(t, output.Bytes(), 0)
	if response.Error == nil || response.Error.Code != "source-limit" || len(payload) != 0 || reader.consumed != 24 || string(reader.data) != meta+"not-read" {
		t.Fatalf("oversize read beyond header: meta=%+v consumed=%d", response, reader.consumed)
	}
}

func TestPreviewWorkerInvalidFramesAndVersion(t *testing.T) {
	valid := previewRequest(t, []byte(simpleSource))
	invalidUTF8 := previewRequest(t, []byte{0xff})
	badVersion := bytes.Clone(valid)
	badVersion[8] = 2
	for _, tc := range []struct {
		name string
		data []byte
		code string
	}{
		{"utf8", invalidUTF8, "invalid-utf8"},
		{"header", valid[:23], "invalid-request"},
		{"short-source", valid[:len(valid)-1], "invalid-request"},
		{"trailing", append(bytes.Clone(valid), 0), "invalid-request"},
		{"protocol-version", badVersion, "invalid-request"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			if exit := serve(bytes.NewReader(tc.data), &output); exit != 0 {
				t.Fatalf("coherent invalid frame exit=%d", exit)
			}
			meta, payload := decodePreviewResponse(t, output.Bytes(), 0)
			if meta.Status != engineipc.StatusError || meta.Error == nil || meta.Error.Code != tc.code || len(meta.Warnings) != 0 || len(payload) != 0 {
				t.Fatalf("bad frame refusal: %+v payload=%d", meta, len(payload))
			}
		})
	}
	old := engineVersion
	t.Cleanup(func() { engineVersion = old })
	for _, version := range []string{"", "bad version", strings.Repeat("v", 65)} {
		engineVersion = version
		reader := &trackedReader{data: valid}
		var output bytes.Buffer
		if exit := serve(reader, &output); exit != 2 || reader.reads != 0 || output.Len() != 0 {
			t.Fatalf("invalid build version read/wrote: exit=%d reads=%d bytes=%d", exit, reader.reads, output.Len())
		}
	}
}

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }

func TestPreviewWorkerResponseWriterFailures(t *testing.T) {
	request := previewRequest(t, []byte(simpleSource))
	for name, writer := range map[string]io.Writer{"failed": failedWriter{}, "short": shortWriter{}} {
		if exit := serve(bytes.NewReader(request), writer); exit != 1 {
			t.Fatalf("%s writer exit=%d want1", name, exit)
		}
	}
}

func TestPreviewWorkerExactSourceLimitAndNonmutation(t *testing.T) {
	const prefix = `{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"R","note":"`
	const suffix = `"}}`
	source := []byte(prefix + strings.Repeat("s", (16<<20)-len(prefix)-len(suffix)) + suffix)
	if len(source) != 16<<20 || !json.Valid(source) {
		t.Fatal("exact source fixture invalid")
	}
	meta, payload := callPreviewWorker(t, source)
	const want = `{"schema":1,"layout":"logicalStructure","nodeCount":1,"edgeCount":0,"bounds":{"x":0,"y":0,"width":148,"height":96},"nodes":[{"id":"r","label":"R","x":28,"y":28,"width":92,"height":40,"shape":"rounded","fill":"","stroke":"","text":"","root":true,"collapsed":false}],"edges":[]}`
	if meta.Status != engineipc.StatusOK || len(meta.Warnings) != 0 || string(payload) != want {
		t.Fatalf("exact source cap: meta=%+v payload=%s", meta, payload)
	}
	if !bytes.Equal(source[:len(prefix)], []byte(prefix)) || !bytes.Equal(source[len(source)-len(suffix):], []byte(suffix)) ||
		bytes.Count(source[len(prefix):len(source)-len(suffix)], []byte{'s'}) != len(source)-len(prefix)-len(suffix) {
		t.Fatal("large input mutated")
	}
	oversize := append(source, ' ')
	data, warnings, failed := render(oversize)
	if failed == nil || failed.Code != "source-limit" || data != nil || len(warnings) != 0 {
		t.Fatalf("direct source cap: error=%+v payload=%d warnings=%+v", failed, len(data), warnings)
	}
}

func TestPreviewWorkerRealEscapedOutputLimit(t *testing.T) {
	// A legal 512-byte root ID is repeated as every edge's parent. Every NUL
	// requires six wire bytes; 11,000 parent IDs alone exceed the real 32MiB
	// codec cap. Source is <1MiB, labels are short, the star is shallow, and
	// native height remains <1e7. No private encoder or artificial cap is used.
	const children = 11_000
	if children*512*6 <= 32<<20 {
		t.Fatal("independent escaped-size lower bound does not exceed cap")
	}
	var source strings.Builder
	source.WriteString(`{"format":"levnote.mindmap","version":1,"layout":"logicalStructureRight","root":{"id":"`)
	source.WriteString(strings.Repeat(`\u0000`, 512))
	source.WriteString(`","topic":"R","children":[`)
	for i := 0; i < children; i++ {
		if i > 0 {
			source.WriteByte(',')
		}
		fmt.Fprintf(&source, `{"id":"n%d","topic":"N"}`, i)
	}
	source.WriteString(`]}}`)
	raw := []byte(source.String())
	if len(raw) >= 1<<20 || !json.Valid(raw) {
		t.Fatalf("output fixture source=%d invalid or unexpectedly large", len(raw))
	}
	doc, issues, err := mindmap.Parse(raw, mindmap.ParseOptions{})
	if err != nil || len(issues) != 0 || len(doc.Root.ID) != 512 || doc.Root.ID != strings.Repeat("\x00", 512) {
		t.Fatalf("output fixture parse err=%v issues=%+v", err, issues)
	}
	layout := mindmap.Layout(doc, mindmap.NativeLayoutOptions())
	if len(layout.Issues) != 0 || len(layout.Nodes) != children+1 || len(layout.Edges) != children ||
		layout.Bounds != (mindmap.Rect{Width: 312, Height: 704032}) || len(mindmap.PresentationIssues(doc)) != 0 {
		t.Fatalf("fixture before encoder: nodes/edges=%d/%d bounds=%+v issues=%+v", len(layout.Nodes), len(layout.Edges), layout.Bounds, layout.Issues)
	}
	for _, node := range layout.Nodes {
		if len(node.ID) > 512 || len(node.Text) != 1 || node.Shape != "rounded" || node.Width != 92 || node.Height != 40 {
			t.Fatalf("fixture violates a different scene bound: %+v", node)
		}
	}
	requireRefusal(t, raw, "output-limit")
	t.Logf("real output-limit source=%d nodes=%d edges=%d escaped-parent-ID lower bound=%d >33554432", len(raw), len(layout.Nodes), len(layout.Edges), children*512*6)
}

func TestPreviewWorkerFixedDescriptor(t *testing.T) {
	got, ok := enginecatalog.Lookup(engineipc.MindmapPreview)
	want := enginecatalog.Descriptor{
		Engine: "levnote.mindmap-preview", Format: "mindmap-scene-v1", Protocol: 1,
		RelativePath: "engines/mindmap-preview", Filename: "levnote-mindmap-preview.exe",
		GOOS: "windows", GOARCH: "amd64", SourceMaxBytes: 16 << 20, PayloadMaxBytes: 32 << 20,
	}
	if !ok || !reflect.DeepEqual(got, want) || got.SourceMaxBytes != mindmap.DefaultMaxBytes || got.PayloadMaxBytes != enginescene.MaxPayloadBytes {
		t.Fatalf("fixed producer descriptor: %+v want=%+v", got, want)
	}
}
