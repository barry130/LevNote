package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/formats/mindmap"
)

func mindmapRequest(t *testing.T, source []byte) []byte {
	t.Helper()
	prefix, err := engineipc.EncodeRequestPrefix(
		engineipc.RequestMeta{Engine: engineipc.MindmapSVG, Format: "svg"},
		uint64(len(source)),
	)
	if err != nil {
		t.Fatal(err)
	}
	return append(append([]byte{}, prefix...), source...)
}

func decodeMindmapResponse(t *testing.T, response []byte, sourceBytes int) (engineipc.ResponseMeta, []byte) {
	t.Helper()
	decoder := engineipc.NewResponseDecoder(engineipc.MindmapSVG, engineVersion, uint64(sourceBytes))
	if _, err := decoder.Write(response); err != nil {
		t.Fatal(err)
	}
	meta, payload, err := decoder.Finish()
	if err != nil {
		t.Fatal(err)
	}
	return meta, payload
}

func callMindmapWorker(t *testing.T, source []byte) (engineipc.ResponseMeta, []byte) {
	t.Helper()
	var output bytes.Buffer
	if exit := serve(bytes.NewReader(mindmapRequest(t, source)), &output); exit != 0 {
		t.Fatalf("serve exit%d", exit)
	}
	return decodeMindmapResponse(t, output.Bytes(), len(source))
}

func TestMindmapWorkerIndependentSVGXMLGeometryEscapingAndNonmutation(t *testing.T) {
	source := []byte(`{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"root <&\"'🙂\nvisible"}}`)
	before := append([]byte{}, source...)
	meta, svg := callMindmapWorker(t, source)

	if meta.Status != engineipc.StatusOK || meta.Error != nil || len(meta.Warnings) != 0 {
		t.Fatalf("success metadata: %+v", meta)
	}
	if !bytes.Equal(source, before) {
		t.Fatalf("source mutated: %q", source)
	}
	if !bytes.Contains(svg, []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="-24 -24 212 160" width="212" height="160">`)) ||
		!bytes.Contains(svg, []byte(`<rect x="28" y="28" width="108" height="56"`)) ||
		!bytes.Contains(svg, []byte(`root &lt;&amp;&#34;&#39;🙂</tspan>`)) ||
		!bytes.Contains(svg, []byte(">visible</tspan>")) {
		t.Fatalf("independent SVG literal oracle: %s", svg)
	}
	if bytes.Contains(svg, []byte("root <")) || bytes.Contains(svg, []byte(`root &amp;`)) {
		t.Fatalf("unescaped label entered SVG: %s", svg)
	}

	var parsed struct {
		ViewBox string    `xml:"viewBox,attr"`
		Rects   []svgRect `xml:"rect"`
		Texts   []svgText `xml:"text"`
	}
	if err := xml.Unmarshal(svg, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.ViewBox != "-24 -24 212 160" || len(parsed.Rects) != 2 {
		t.Fatalf("geometry XML: %+v rects=%d", parsed, len(parsed.Rects))
	}
	root := parsed.Rects[1]
	if root.X != 28 || root.Y != 28 || root.Width != 108 || root.Height != 56 {
		t.Fatalf("root geometry changed: %+v", root)
	}
	if len(parsed.Texts) != 1 || len(parsed.Texts[0].Tspans) != 2 ||
		parsed.Texts[0].Tspans[0].Value != "root <&\"'🙂" || parsed.Texts[0].Tspans[1].Value != "visible" {
		t.Fatalf("decoded SVG label changed: %+v", parsed.Texts)
	}

	document, parsedIssues, err := mindmap.Parse(source, mindmap.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	layout := mindmap.Layout(document, mindmap.NativeLayoutOptions())
	presentation := mindmap.PresentationIssues(document)
	converted, err := wireWarnings(len(source), parsedIssues, layout.Issues, presentation)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := mindmap.ExportSVG(document, layout, mindmap.MaxSVGBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(svg, expected) || !reflect.DeepEqual(meta.Warnings, converted) {
		t.Fatalf("pure parity failed: svg=%d expected=%d warnings=%+v", len(svg), len(expected), meta.Warnings)
	}

	metaTwo, svgTwo := callMindmapWorker(t, source)
	if !reflect.DeepEqual(meta, metaTwo) || !bytes.Equal(svg, svgTwo) {
		t.Fatal("successful response is not deterministic")
	}
}

type svgRect struct {
	X      float64 `xml:"x,attr"`
	Y      float64 `xml:"y,attr"`
	Width  float64 `xml:"width,attr"`
	Height float64 `xml:"height,attr"`
}

type svgText struct {
	Tspans []svgTspan `xml:"tspan"`
}

type svgTspan struct {
	Value string `xml:",chardata"`
}

func TestMindmapWorkerDefaultAndUnknownLayout(t *testing.T) {
	defaultSource := []byte(`{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"root","children":[{"id":"a","topic":"child"}]}}`)
	meta, svg := callMindmapWorker(t, defaultSource)
	if meta.Status != engineipc.StatusOK || len(meta.Warnings) != 0 ||
		!bytes.Contains(svg, []byte(`<path d="M192 48 L120 48"`)) {
		t.Fatalf("default two-sided layout: meta=%+v svg=%s", meta, svg)
	}

	unknownSource := []byte(`{"format":"levnote.mindmap","version":1,"layout":"future-layout","root":{"id":"r","topic":"root"}}`)
	meta, svg = callMindmapWorker(t, unknownSource)
	if meta.Status != engineipc.StatusOK || len(meta.Warnings) != 2 ||
		meta.Warnings[0].Code != string(mindmap.IssueUnknownLayout) ||
		meta.Warnings[1].Code != string(mindmap.IssueUnknownLayout) ||
		!bytes.Contains(svg, []byte(`width="196"`)) {
		t.Fatalf("unknown layout fallback/warnings: meta=%+v svg=%s", meta, svg)
	}
	for _, warning := range meta.Warnings {
		if warning.Start != 0 || warning.End != uint32(len(unknownSource)) {
			t.Fatalf("layout warning range: %+v source=%d", warning, len(unknownSource))
		}
	}
}

func TestMindmapWorkerDisclosureWarningsIncludeHiddenMetadata(t *testing.T) {
	shortSource := []byte(`{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"root","icons":["⭐","custom<&"],"tags":["甲<&"],"style":{"color":" ReBeccAPurple "},"image":{"url":"https://invalid/a.png","width":40,"height":40}}}`)
	shortMeta, shortSVG := callMindmapWorker(t, shortSource)
	if shortMeta.Status != engineipc.StatusOK || len(shortMeta.Warnings) != 1 ||
		shortMeta.Warnings[0].Code != string(mindmap.IssueImagePlaceholder) ||
		!bytes.Contains(shortSVG, []byte(`fill="#663399"`)) {
		t.Fatalf("short disclosure metadata: %+v", shortMeta)
	}
	if !bytes.Contains(shortSVG, []byte("⭐ custom&lt;&amp; root [甲&lt;&amp;]")) ||
		!bytes.Contains(shortSVG, []byte(`fill="#663399"`)) ||
		!bytes.Contains(shortSVG, []byte(">图</text>")) {
		t.Fatalf("exact icon/topic/tag/paint/image disclosure missing: %s", shortSVG)
	}

	hiddenSource := []byte(`{"format":"levnote.mindmap","version":2,"layout":"future-layout","futureTop":1,` +
		`"root":{"id":"r","topic":"root","collapsed":true,"children":[{"id":"hidden","topic":"` +
		strings.Repeat("字", 513) + `","icons":["⭐"],"tags":["甲<&"],` +
		`"style":{"color":" ReBeccAPurple ","border":"url(https://invalid)"},` +
		`"image":{"url":"https://invalid/a.png","width":40,"height":40},"futureChild":1}]}}`)
	meta, svg := callMindmapWorker(t, hiddenSource)

	wantCodes := []string{
		string(mindmap.IssueFutureVersion),
		string(mindmap.IssueUnknownLayout),
		string(mindmap.IssueUnknownKey),
		string(mindmap.IssueUnknownKey),
		string(mindmap.IssueUnknownLayout),
		string(mindmap.IssueImagePlaceholder),
		string(mindmap.IssueLabelTruncated),
		string(mindmap.IssueUnsupportedColor),
	}
	gotCodes := make([]string, 0, len(meta.Warnings))
	for _, warning := range meta.Warnings {
		gotCodes = append(gotCodes, warning.Code)
		if warning.Start != 0 || warning.End != uint32(len(hiddenSource)) {
			t.Fatalf("warning does not cover whole source: %+v source=%d", warning, len(hiddenSource))
		}
	}
	if !reflect.DeepEqual(gotCodes, wantCodes) {
		t.Fatalf("warning order/codes: %q want %q", gotCodes, wantCodes)
	}
	for _, hidden := range []string{"hidden", "⭐", "甲", "#663399", ">图</text>"} {
		if bytes.Contains(svg, []byte(hidden)) {
			t.Fatalf("hidden descendant metadata rendered: %s", svg)
		}
	}
	if bytes.Contains(svg, []byte("https://invalid")) {
		t.Fatalf("unsupported paint/image URL leaked: %s", svg)
	}
}

func TestMindmapWorkerWarningBoundsRefuseInsteadOfTruncating(t *testing.T) {
	base := mindmap.Issue{Severity: mindmap.SeverityWarning, Code: "WARNING", Message: "fine"}
	exactlyMax := make([]mindmap.Issue, engineipc.WarningsMaxCount)
	for index := range exactlyMax {
		exactlyMax[index] = base
	}
	if warnings, err := wireWarnings(1, exactlyMax); err != nil || len(warnings) != engineipc.WarningsMaxCount {
		t.Fatalf("maximum warning count refused: %d %v", len(warnings), err)
	}
	tooMany := append([]mindmap.Issue{}, exactlyMax...)
	tooMany = append(tooMany, base)
	if warnings, err := wireWarnings(1, tooMany); err == nil || warnings != nil {
		t.Fatalf("excessive warning count accepted: %d %v", len(warnings), err)
	}

	longCode := base
	longCode.Code = mindmap.IssueCode(strings.Repeat("c", engineipc.WarningCodeMaxBytes+1))
	longMessage := base
	longMessage.Message = strings.Repeat("m", engineipc.WarningMessageMaxBytes+1)
	nonWarning := base
	nonWarning.Severity = mindmap.SeverityError
	for name, issue := range map[string]mindmap.Issue{
		"long-code":    longCode,
		"long-message": longMessage,
		"non-warning":  nonWarning,
	} {
		if warnings, err := wireWarnings(1, []mindmap.Issue{issue}); err == nil || warnings != nil {
			t.Fatalf("%s accepted: %d %v", name, len(warnings), err)
		}
	}
	if warnings, err := wireWarnings(-1, []mindmap.Issue{base}); err == nil || warnings != nil {
		t.Fatalf("negative source byte count accepted: %d %v", len(warnings), err)
	}

	var source bytes.Buffer
	source.WriteString(`{"format":"levnote.mindmap","version":1,"root":{"id":"root","topic":"root","future":1,"children":[`)
	for index := 0; index < engineipc.WarningsMaxCount; index++ {
		if index > 0 {
			source.WriteByte(',')
		}
		fmt.Fprintf(&source, `{"id":"n%03d","topic":"x","future":1}`, index)
	}
	source.WriteString(`]}}`)
	if !json.Valid(source.Bytes()) {
		t.Fatal("excessive warning fixture is invalid JSON")
	}
	meta, payload := callMindmapWorker(t, source.Bytes())
	if meta.Status != engineipc.StatusError || meta.Error == nil || meta.Error.Code != "warning-limit" ||
		len(meta.Warnings) != 0 || len(payload) != 0 {
		t.Fatalf("excessive warning count through serve: %+v payload=%d", meta, len(payload))
	}

	longLayout := strings.Repeat("x", engineipc.WarningMessageMaxBytes+1)
	longLayoutSource := []byte(`{"format":"levnote.mindmap","version":1,"layout":"` + longLayout + `","root":{"id":"r","topic":"x"}}`)
	if !json.Valid(longLayoutSource) {
		t.Fatal("long layout source is invalid JSON")
	}
	meta, payload = callMindmapWorker(t, longLayoutSource)
	if meta.Status != engineipc.StatusError || meta.Error == nil || meta.Error.Code != "warning-limit" ||
		len(meta.Warnings) != 0 || len(payload) != 0 {
		t.Fatalf("overlong warning through serve: %+v payload=%d", meta, len(payload))
	}
}

func TestMindmapWorkerInvalidDeepAndOversizeRequests(t *testing.T) {
	invalid := []byte(`{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"x","children":[7]}}`)
	meta, payload := callMindmapWorker(t, invalid)
	if meta.Status != engineipc.StatusError || meta.Error == nil || meta.Error.Code != "render-failed" || len(payload) != 0 || len(meta.Warnings) != 0 {
		t.Fatalf("invalid schema mapping: %+v payload=%d", meta, len(payload))
	}

	node := []byte(`{"id":"deep","topic":"x"}`)
	for index := 1; index <= 257; index++ {
		prefix := []byte(fmt.Sprintf(`{"id":"node%03d","topic":"x","children":[`, index))
		node = append(append(prefix, node...), ']', '}')
	}
	deep := append([]byte(`{"format":"levnote.mindmap","version":1,"root":`), node...)
	deep = append(deep, '}')
	if !json.Valid(deep) {
		t.Fatalf("deep fixture is invalid JSON: %s", deep)
	}
	meta, payload = callMindmapWorker(t, deep)
	if meta.Status != engineipc.StatusError || meta.Error == nil || meta.Error.Code != "source-limit" || len(payload) != 0 || len(meta.Warnings) != 0 {
		t.Fatalf("deep source mapping: %+v payload=%d", meta, len(payload))
	}

	valid := mindmapRequest(t, []byte("x"))
	oversize := append([]byte{}, valid...)
	binary.LittleEndian.PutUint64(oversize[16:24], uint64(enginecatalog.MindmapSourceMaxBytes+1))
	checkMindmapRequestError(t, oversize, "source-limit")

	invalidUTF8 := append(append([]byte{}, valid[:len(valid)-1]...), 0xff)
	checkMindmapRequestError(t, invalidUTF8, "invalid-utf8")
	checkMindmapRequestError(t, valid[:len(valid)-1], "invalid-request")
	checkMindmapRequestError(t, append(append([]byte{}, valid...), 0), "invalid-request")
}

func checkMindmapRequestError(t *testing.T, request []byte, code string) {
	t.Helper()
	var output bytes.Buffer
	if exit := serve(bytes.NewReader(request), &output); exit != 0 {
		t.Fatalf("coherent request failure exit%d", exit)
	}
	meta, payload := decodeMindmapResponse(t, output.Bytes(), 0)
	if meta.Status != engineipc.StatusError || meta.Error == nil || meta.Error.Code != code || len(payload) != 0 || len(meta.Warnings) != 0 {
		t.Fatalf("request failure code=%q meta=%+v payload=%d", code, meta, len(payload))
	}
}

type countingMindmapReader struct {
	data  []byte
	reads int
}

func (reader *countingMindmapReader) Read(buffer []byte) (int, error) {
	reader.reads++
	if len(reader.data) == 0 {
		return 0, io.EOF
	}
	count := copy(buffer, reader.data)
	reader.data = reader.data[count:]
	return count, nil
}

func TestMindmapWorkerForeignRequestDoesNotReadPayload(t *testing.T) {
	const metadata = `{"engine":"levnote.markdown-export","format":"html"}`
	frame := append([]byte("LVEXREQ\x00"), 1, 0, 24, 0)
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(metadata)))
	frame = binary.LittleEndian.AppendUint64(frame, 6)
	frame = append(append(frame, metadata...), "source"...)

	reader := &countingMindmapReader{data: frame}
	var output bytes.Buffer
	if exit := serve(reader, &output); exit != 0 {
		t.Fatalf("coherent foreign request exit%d", exit)
	}
	meta, payload := decodeMindmapResponse(t, output.Bytes(), 0)
	if meta.Status != engineipc.StatusError || meta.Error == nil || meta.Error.Code != "invalid-request" || len(payload) != 0 {
		t.Fatalf("foreign request mapping: %+v payload=%d", meta, len(payload))
	}
	if reader.reads != 2 || string(reader.data) != "source" {
		t.Fatalf("foreign request read payload: reads=%d remaining=%q", reader.reads, reader.data)
	}
}

func TestMindmapWorkerTinyOutputCapWithEscapedLabel(t *testing.T) {
	source := []byte(`{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"<&\"'🙂"}}`)
	expected, warnings, failed := renderWithLimit(source, mindmap.MaxSVGBytes)
	if failed != nil || len(warnings) != 0 || !bytes.Contains(expected, []byte("&lt;&amp;&#34;&#39;🙂")) {
		t.Fatalf("escaped fixture failed: %v %s", failed, expected)
	}

	request := mindmapRequest(t, source)
	for _, limit := range []int{len(expected), len(expected) - 1, 1} {
		var output bytes.Buffer
		if exit := serveWithLimit(bytes.NewReader(request), &output, limit); exit != 0 {
			t.Fatalf("coherent output-cap response exit%d limit=%d", exit, limit)
		}
		meta, payload := decodeMindmapResponse(t, output.Bytes(), len(source))
		if limit == len(expected) {
			if meta.Status != engineipc.StatusOK || meta.Error != nil || len(meta.Warnings) != 0 || !bytes.Equal(payload, expected) {
				t.Fatalf("exact SVG budget failed: limit=%d meta=%+v payload=%d", limit, meta, len(payload))
			}
			continue
		}
		if meta.Status != engineipc.StatusError || meta.Error == nil || meta.Error.Code != "output-limit" ||
			len(payload) != 0 || len(meta.Warnings) != 0 ||
			binary.LittleEndian.Uint64(output.Bytes()[16:24]) != 0 {
			t.Fatalf("partial SVG output accepted: limit=%d meta=%+v payload=%d", limit, meta, len(payload))
		}
	}
}

type brokenMindmapWriter struct{}

func (brokenMindmapWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

type shortMindmapWriter struct{}

func (shortMindmapWriter) Write(buffer []byte) (int, error) { return len(buffer) - 1, nil }

func TestMindmapWorkerWriterFailures(t *testing.T) {
	request := mindmapRequest(t, []byte(`{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"x"}}`))
	for name, writer := range map[string]io.Writer{
		"broken": brokenMindmapWriter{},
		"short":  shortMindmapWriter{},
	} {
		if exit := serve(bytes.NewReader(request), writer); exit != 1 {
			t.Fatalf("%s writer failure exit%d", name, exit)
		}
	}
}

func TestMindmapWorkerFrozenLimits(t *testing.T) {
	descriptor, ok := enginecatalog.Lookup(engineipc.MindmapSVG)
	if !ok || descriptor.SourceMaxBytes != mindmap.DefaultMaxBytes || mindmap.MaxSVGBytes != descriptor.PayloadMaxBytes {
		t.Fatalf("catalog/parser/output limit drift: %+v parser=%d svg=%d", descriptor, mindmap.DefaultMaxBytes, mindmap.MaxSVGBytes)
	}
	if engineipc.SourceMaxBytes != enginecatalog.MarkdownSourceMaxBytes {
		t.Fatal("legacy markdown source alias changed")
	}
}
