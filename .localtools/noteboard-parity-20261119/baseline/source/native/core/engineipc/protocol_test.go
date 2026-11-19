package engineipc

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/enginescene"
)

const testMeta = `{"engine":"levnote.markdown-export","engineVersion":"v1","format":"html","status":"ok","warnings":[]}`

// Independent little-endian frame construction, deliberately not makeHeader.
func frame(meta string, payload []byte, response bool) []byte {
	magic := "LVEXREQ\x00"
	if response {
		magic = "LVEXRES\x00"
	}
	h := append([]byte(magic), 1, 0, 24, 0)
	m := uint32(len(meta))
	for range 4 {
		h = append(h, byte(m))
		m >>= 8
	}
	p := uint64(len(payload))
	for range 8 {
		h = append(h, byte(p))
		p >>= 8
	}
	return append(append(h, meta...), payload...)
}

func TestRequestHeaderRawAndExactEOF(t *testing.T) {
	source := []byte("\ufeff字形\r\n<&")
	prefix, err := EncodeRequestPrefix(RequestMeta{Engine: MarkdownHTML, Format: "html"}, uint64(len(source)))
	if err != nil {
		t.Fatal(err)
	}
	want := frame(`{"engine":"levnote.markdown-export","format":"html"}`, source, false)
	if !bytes.Equal(append(prefix, source...), want) {
		t.Fatal("header/raw bytes differ from independent oracle")
	}
	meta, got, err := ReadRequest(&oneByteReader{data: want})
	if err != nil || meta.Engine != MarkdownHTML || !bytes.Equal(got, source) {
		t.Fatalf("request roundtrip: %+v %q %v", meta, got, err)
	}
	for _, bad := range [][]byte{want[:len(want)-1], append(append([]byte{}, want...), 0)} {
		if _, data, err := ReadRequest(bytes.NewReader(bad)); err == nil || data != nil {
			t.Fatal("truncated/extra request accepted")
		}
	}
	if _, _, err := ReadRequest(bytes.NewReader(frame(`{"engine":"levnote.markdown-export","format":"html"}`, []byte{0xff}, false))); err == nil {
		t.Fatal("invalidUTF8 accepted")
	}
}

func TestMindmapOperationWireAndPreflight(t *testing.T) {
	const requestMeta = `{"engine":"levnote.mindmap-svg","format":"svg"}`
	source := []byte(`{"format":"levnote.mindmap"}`)
	prefix, err := EncodeRequestPrefix(RequestMeta{Engine: MindmapSVG, Format: "svg"}, uint64(len(source)))
	if err != nil {
		t.Fatal(err)
	}
	got := append(append([]byte{}, prefix...), source...)
	want := frame(requestMeta, source, false)
	if !bytes.Equal(got, want) {
		t.Fatal("mindmap request bytes differ from independent frame")
	}
	meta, decoded, err := ReadRequestFor(MindmapSVG, &oneByteReader{data: want})
	if err != nil || meta.Engine != MindmapSVG || meta.Format != "svg" || !bytes.Equal(decoded, source) {
		t.Fatalf("mindmap request: %+v %q %v", meta, decoded, err)
	}
	if _, data, err := ReadRequestFor(MindmapSVG, bytes.NewReader(frame(requestMeta, []byte{0xff}, false))); err == nil || data != nil {
		t.Fatal("mindmap invalid UTF8 source accepted")
	}
	if _, data, err := ReadRequestFor(MarkdownHTML, bytes.NewReader(want)); err == nil || data != nil {
		t.Fatal("markdown reader accepted mindmap request")
	}

	counting := &countingReader{}
	if _, data, err := ReadRequestFor(EngineID("unknown"), counting); err == nil || data != nil || counting.reads != 0 {
		t.Fatal("unsupported expected engine read input")
	}
	oversize := frame(requestMeta, nil, false)
	binary.LittleEndian.PutUint64(oversize[16:], uint64(enginecatalog.MindmapSourceMaxBytes)+1)
	counting = &countingReader{data: oversize[:24+len(requestMeta)]}
	if _, data, err := ReadRequestFor(MindmapSVG, counting); err == nil || data != nil || counting.reads != 1 {
		t.Fatal("oversize mindmap source reached payload allocation/read")
	}
}

func TestPreviewOperationWireAndPreflight(t *testing.T) {
	const requestMeta = `{"engine":"levnote.mindmap-preview","format":"mindmap-scene-v1"}`
	source := []byte(`{"format":"levnote.mindmap"}`)
	prefix, err := EncodeRequestPrefix(RequestMeta{Engine: MindmapPreview, Format: "mindmap-scene-v1"}, uint64(len(source)))
	if err != nil {
		t.Fatal(err)
	}
	got := append(append([]byte{}, prefix...), source...)
	want := frame(requestMeta, source, false)
	if !bytes.Equal(got, want) {
		t.Fatal("preview request bytes differ from independent frame")
	}
	meta, decoded, err := ReadRequestFor(MindmapPreview, &oneByteReader{data: want})
	if err != nil || meta.Engine != MindmapPreview || meta.Format != "mindmap-scene-v1" || !bytes.Equal(decoded, source) {
		t.Fatalf("preview request: %+v %q %v", meta, decoded, err)
	}
	if _, data, err := ReadRequestFor(MindmapPreview, bytes.NewReader(frame(requestMeta, []byte{0xff}, false))); err == nil || data != nil {
		t.Fatal("preview invalid UTF8 source accepted")
	}

	crossMeta := `{"engine":"levnote.mindmap-svg","format":"svg"}`
	reader := &countingReader{data: frame(crossMeta, []byte("source"), false)}
	if _, data, err := ReadRequestFor(MindmapPreview, reader); err == nil || data != nil || reader.reads != 2 || len(reader.data) != 6 {
		t.Fatalf("preview reader accepted another operation or read source: reads=%d remaining=%d err=%v", reader.reads, len(reader.data), err)
	}

	oversize := frame(requestMeta, nil, false)
	binary.LittleEndian.PutUint64(oversize[16:], uint64(enginecatalog.MindmapSourceMaxBytes)+1)
	counting := &countingReader{data: oversize[:24+len(requestMeta)]}
	if _, data, err := ReadRequestFor(MindmapPreview, counting); err == nil || data != nil || counting.reads != 1 {
		t.Fatal("oversize preview source reached payload allocation/read")
	}
}

func TestForeignRequestsDoNotReadSource(t *testing.T) {
	const markdownMeta = `{"engine":"levnote.markdown-export","format":"html"}`
	const mindmapMeta = `{"engine":"levnote.mindmap-svg","format":"svg"}`
	const previewMeta = `{"engine":"levnote.mindmap-preview","format":"mindmap-scene-v1"}`
	const wrongFormatMeta = `{"engine":"levnote.mindmap-svg","format":"html"}`
	for _, test := range []struct {
		name     string
		expected EngineID
		meta     string
	}{
		{"markdown-wrong-format", MarkdownHTML, wrongFormatMeta},
		{"markdown-wrong-engine", MarkdownHTML, mindmapMeta},
		{"mindmap-wrong-engine", MindmapSVG, markdownMeta},
		{"expected-wrong-format", MindmapSVG, wrongFormatMeta},
	} {
		t.Run(test.name, func(t *testing.T) {
			reader := &countingReader{data: frame(test.meta, []byte("source"), false)}
			_, source, err := ReadRequestFor(test.expected, reader)
			if err == nil || source != nil || reader.reads != 2 || len(reader.data) != 6 {
				t.Fatalf("foreign request read source: reads=%d remaining=%d source=%v err=%v", reader.reads, len(reader.data), source != nil, err)
			}
		})
	}
	legacy := &countingReader{data: frame(mindmapMeta, []byte("source"), false)}
	if _, source, err := ReadRequest(legacy); err == nil || source != nil || legacy.reads != 2 || len(legacy.data) != 6 {
		t.Fatalf("legacy wrapper read foreign source: reads=%d remaining=%d source=%v err=%v", legacy.reads, len(legacy.data), source != nil, err)
	}
	legacyPreview := &countingReader{data: frame(previewMeta, []byte("source"), false)}
	if _, source, err := ReadRequest(legacyPreview); err == nil || source != nil || legacyPreview.reads != 2 || len(legacyPreview.data) != 6 {
		t.Fatalf("legacy wrapper read preview source: reads=%d remaining=%d source=%v err=%v", legacyPreview.reads, len(legacyPreview.data), source != nil, err)
	}
}

type countingReader struct {
	data  []byte
	reads int
}

func (r *countingReader) Read(p []byte) (int, error) {
	r.reads++
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

type oneByteReader struct{ data []byte }

func (r *oneByteReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	p[0] = r.data[0]
	r.data = r.data[1:]
	return 1, nil
}

func TestDeclaredLimitsBeforeAllocation(t *testing.T) {
	for _, response := range []bool{false, true} {
		for _, offset := range []int{12, 16} {
			h := frame(`{}`, nil, response)[:24]
			for i := offset; i < 24; i++ {
				h[i] = 0xff
			}
			if response {
				d := NewResponseDecoder(MarkdownHTML, "v1", 0)
				if _, err := d.Write(h); err == nil || d.payload != nil {
					t.Fatal("oversize response accepted/allocated")
				}
			} else if _, data, err := ReadRequest(bytes.NewReader(h)); err == nil || data != nil {
				t.Fatal("oversize request accepted/allocated")
			}
		}
	}
	for _, test := range []struct {
		engine EngineID
		format string
		limit  int64
	}{
		{MarkdownHTML, "html", enginecatalog.MarkdownSourceMaxBytes},
		{MindmapSVG, "svg", enginecatalog.MindmapSourceMaxBytes},
		{MindmapPreview, "mindmap-scene-v1", enginecatalog.MindmapSourceMaxBytes},
	} {
		prefix, err := EncodeRequestPrefix(RequestMeta{Engine: test.engine, Format: test.format}, uint64(test.limit))
		if err != nil {
			t.Fatalf("exact source cap rejected for %s: %v", test.engine, err)
		}
		if got := binary.LittleEndian.Uint64(prefix[16:24]); got != uint64(test.limit) {
			t.Fatalf("exact source cap encoded %d for %s", got, test.engine)
		}
		if _, err := EncodeRequestPrefix(RequestMeta{Engine: test.engine, Format: test.format}, uint64(test.limit)+1); err == nil || !errors.Is(err, ErrProtocol) {
			t.Fatalf("source cap +1 accepted for %s: %v", test.engine, err)
		}
	}
}

func TestStrictRequestMetadata(t *testing.T) {
	for name, meta := range map[string]string{
		"duplicate":             `{"engine":"levnote.markdown-export","format":"html","format":"html"}`,
		"unknown":               `{"engine":"levnote.markdown-export","format":"html","path":"local.md"}`,
		"case":                  `{"Engine":"levnote.markdown-export","format":"html"}`,
		"null":                  `{"engine":"levnote.markdown-export","format":null}`,
		"missing":               `{"engine":"levnote.markdown-export"}`,
		"type":                  `{"engine":"levnote.markdown-export","format":7}`,
		"nested":                `{"engine":{"engine":"levnote.markdown-export"},"format":"html"}`,
		"array":                 `["levnote.markdown-export","html"]`,
		"surrogate":             `{"engine":"levnote.markdown-export","format":"\ud800"}`,
		"invalid-key-surrogate": `{"engine":"levnote.markdown-export","\udc00":"html"}`,
		"invalid-utf8":          "{\"engine\":\"levnote.markdown-export\",\"format\":\"\xff\"}",
		"trailing":              `{"engine":"levnote.markdown-export","format":"html"}{}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, source, err := ReadRequest(bytes.NewReader(frame(meta, nil, false)))
			var failure *Failure
			if !errors.As(err, &failure) || failure.Code != "invalid-request" || source != nil {
				t.Fatalf("bad request accepted: %v", err)
			}
		})
	}
}

func TestResponseFragmentationTransferAndStickyErrors(t *testing.T) {
	payload := []byte("<p>\ufeff字😊</p>")
	for _, size := range []int{1, 2, 7, 24, 4096} {
		d := NewResponseDecoder(MarkdownHTML, "v1", 1)
		data := frame(testMeta, payload, true)
		for len(data) > 0 {
			n := min(size, len(data))
			if _, err := d.Write(data[:n]); err != nil {
				t.Fatal(err)
			}
			data = data[n:]
		}
		owned := d.payload
		meta, got, err := d.Finish()
		if err != nil || meta.Status != StatusOK || !bytes.Equal(got, payload) || &got[0] != &owned[0] || d.payload != nil {
			t.Fatal("result not transferred exactly")
		}
		if _, _, err := d.Finish(); err == nil {
			t.Fatal("second Finish accepted")
		}
	}
	d := NewResponseDecoder(MarkdownHTML, "v1", 0)
	bad := frame(testMeta, []byte{0xff}, true)
	_, first := d.Write(bad)
	if first == nil {
		t.Fatal("invalid UTF8 did not fail at complete payload")
	}
	_, next := d.Write(frame(testMeta, nil, true))
	_, data, last := d.Finish()
	if next != first || last != first || data != nil {
		t.Fatal("decoder failure not sticky")
	}
}

func TestStrictResponseMetadataAndFrameFailures(t *testing.T) {
	mutations := map[string]string{
		"duplicate":              strings.Replace(testMeta, `"format":"html"`, `"format":"html","format":"html"`, 1),
		"wrong-case":             strings.Replace(testMeta, `"engine":`, `"Engine":`, 1),
		"unknown":                strings.Replace(testMeta, `"warnings":[]`, `"warnings":[],"other":1`, 1),
		"missing":                strings.Replace(testMeta, `,"warnings":[]`, "", 1),
		"null-array":             strings.Replace(testMeta, `"warnings":[]`, `"warnings":null`, 1),
		"wrong-type":             strings.Replace(testMeta, `"format":"html"`, `"format":1`, 1),
		"version-empty":          strings.Replace(testMeta, `"v1"`, `""`, 1),
		"version-space":          strings.Replace(testMeta, `"v1"`, `"v 1"`, 1),
		"version-nonascii":       strings.Replace(testMeta, `"v1"`, `"字"`, 1),
		"wrong-identity":         strings.Replace(testMeta, "levnote.markdown-export", "untrusted", 1),
		"wrong-version":          strings.Replace(testMeta, `"v1"`, `"v2"`, 1),
		"surrogate":              strings.Replace(testMeta, `"v1"`, `"\ud800"`, 1),
		"trailing-json":          testMeta + `{}`,
		"bad-status":             strings.Replace(testMeta, `"ok"`, `"partial"`, 1),
		"error-on-success":       strings.Replace(testMeta, `"warnings":[]`, `"warnings":[],"error":{"code":"render-failed","message":"x"}`, 1),
		"nested-duplicate":       strings.Replace(testMeta, `"warnings":[]`, `"warnings":[{"code":"x","code":"y","message":"x","start":0,"end":0}]`, 1),
		"negative-range":         strings.Replace(testMeta, `"warnings":[]`, `"warnings":[{"code":"x","message":"x","start":-1,"end":0}]`, 1),
		"out-of-range":           strings.Replace(testMeta, `"warnings":[]`, `"warnings":[{"code":"x","message":"x","start":0,"end":2}]`, 1),
		"nested-error-duplicate": strings.Replace(strings.Replace(testMeta, `"ok"`, `"error"`, 1), `"warnings":[]`, `"warnings":[],"error":{"code":"render-failed","message":"x","message":"y"}`, 1),
		"nested-error-null":      strings.Replace(strings.Replace(testMeta, `"ok"`, `"error"`, 1), `"warnings":[]`, `"warnings":[],"error":{"code":"render-failed","message":null}`, 1),
		"nested-error-case":      strings.Replace(strings.Replace(testMeta, `"ok"`, `"error"`, 1), `"warnings":[]`, `"warnings":[],"error":{"Code":"render-failed","message":"x"}`, 1),
	}
	for name, meta := range mutations {
		t.Run(name, func(t *testing.T) {
			d := NewResponseDecoder(MarkdownHTML, "v1", 1)
			_, err := d.Write(frame(meta, nil, true))
			if err == nil {
				_, _, err = d.Finish()
			}
			if err == nil || !errors.Is(err, ErrProtocol) {
				t.Fatalf("accepted invalid metadata: %s", name)
			}
		})
	}
	for _, kind := range []string{"magic", "version", "header-size", "truncated", "extra", "invalid-meta-utf8"} {
		t.Run(kind, func(t *testing.T) {
			data := frame(testMeta, []byte("x"), true)
			switch kind {
			case "magic":
				data[0] = 'X'
			case "version":
				data[8] = 2
			case "header-size":
				data[10] = 25
			case "truncated":
				data = data[:len(data)-1]
			case "extra":
				data = append(data, 0)
			case "invalid-meta-utf8":
				data[30] = 0xff
			}
			d := NewResponseDecoder(MarkdownHTML, "v1", 0)
			_, err := d.Write(data)
			if err == nil {
				_, _, err = d.Finish()
			}
			if err == nil {
				t.Fatal("invalid frame accepted")
			}
		})
	}
}

func TestDecodedTextAndCountBoundaries(t *testing.T) {
	base := func() ResponseMeta {
		return ResponseMeta{Engine: MarkdownHTML, EngineVersion: "v1", Format: "html", Status: StatusOK, Warnings: []Warning{}}
	}
	check := func(t *testing.T, m ResponseMeta, version string, valid bool) {
		t.Helper()
		metadata, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		d := NewResponseDecoder(MarkdownHTML, version, 1)
		_, err = d.Write(frame(string(metadata), nil, true))
		if err == nil {
			_, _, err = d.Finish()
		}
		if (err == nil) != valid {
			t.Fatalf("valid=%v error=%v", valid, err)
		}
	}
	for _, length := range []int{64, 65} {
		t.Run("version-"+strconv.Itoa(length), func(t *testing.T) {
			m := base()
			m.EngineVersion = strings.Repeat("v", length)
			expected := m.EngineVersion
			if length > 64 {
				expected = strings.Repeat("v", 64)
			}
			check(t, m, expected, length == 64)
		})
	}
	for _, length := range []int{64, 65} {
		t.Run("warning-code-"+strconv.Itoa(length), func(t *testing.T) {
			m := base()
			m.Warnings = []Warning{{Code: strings.Repeat("c", length), Message: "fine", End: 1}}
			check(t, m, "v1", length == 64)
		})
	}
	for _, length := range []int{512, 513} {
		t.Run("warning-message-"+strconv.Itoa(length), func(t *testing.T) {
			m := base()
			m.Warnings = []Warning{{Code: "warning", Message: strings.Repeat("x", length), End: 1}}
			check(t, m, "v1", length == 512)
		})
	}
	for _, length := range []int{1024, 1025} {
		t.Run("error-message-"+strconv.Itoa(length), func(t *testing.T) {
			m := base()
			m.Status = StatusError
			m.Error = &EngineError{Code: "render-failed", Message: strings.Repeat("x", length)}
			check(t, m, "v1", length == 1024)
		})
	}
	for _, count := range []int{256, 257} {
		m := base()
		m.Warnings = make([]Warning, count)
		for i := range m.Warnings {
			m.Warnings[i] = Warning{Code: "w", Message: "valid", End: 1}
		}
		check(t, m, "v1", count == 256)
	}
	// Bounds are UTF8 bytes; surrogate pairs survive as their actual character.
	m := base()
	m.Warnings = []Warning{{Code: "w", Message: strings.Repeat("字", 170) + "ab", End: 1}}
	check(t, m, "v1", true)
	m.Warnings[0].Message += "x"
	check(t, m, "v1", false)
	paired := strings.Replace(testMeta, `"warnings":[]`, `"warnings":[{"code":"w","message":"\ud83d\ude0a","start":0,"end":0}]`, 1)
	d := NewResponseDecoder(MarkdownHTML, "v1", 0)
	if _, err := d.Write(frame(paired, nil, true)); err != nil {
		t.Fatal(err)
	}
	got, _, err := d.Finish()
	if err != nil || got.Warnings[0].Message != "😊" {
		t.Fatal("valid surrogate pair changed")
	}
}

func TestErrorAndWarningBounds(t *testing.T) {
	m := ResponseMeta{Engine: MarkdownHTML, EngineVersion: "v1", Format: "html", Status: StatusError, Warnings: []Warning{}, Error: &EngineError{Code: "partial-document", Message: "refused"}}
	var out bytes.Buffer
	if err := WriteResponse(&out, m, nil); err != nil {
		t.Fatal(err)
	}
	d := NewResponseDecoder(MarkdownHTML, "v1", 0)
	if _, err := d.Write(out.Bytes()); err != nil {
		t.Fatal(err)
	}
	meta, payload, err := d.Finish()
	if err != nil || payload != nil || meta.Error.Code != "partial-document" {
		t.Fatal("coherent error failed")
	}
	m.Error.Code = "arbitrary"
	if WriteResponse(io.Discard, m, nil) == nil {
		t.Fatal("unknown error accepted")
	}
	m.Error.Code = "render-failed"
	if WriteResponse(io.Discard, m, []byte("partial")) == nil {
		t.Fatal("partial error payload accepted")
	}
	m.Status = StatusOK
	m.Error = nil
	for _, count := range []int{256, 257} {
		m.Warnings = make([]Warning, count)
		for i := range m.Warnings {
			m.Warnings[i] = Warning{Code: "x", Message: strings.Repeat("x", 512)}
		}
		err := WriteResponse(io.Discard, m, nil)
		if (err == nil) != (count == 256) {
			t.Fatalf("warning count%d err=%v", count, err)
		}
	}
	h := frame(testMeta, nil, true)[:24]
	binary.LittleEndian.PutUint64(h[16:], HTMLMaxBytes+1)
	d = NewResponseDecoder(MarkdownHTML, "v1", 0)
	_, err = d.Write(h)
	if !errors.Is(err, ErrStdoutLimit) {
		t.Fatalf("limit not classified: %v", err)
	}
}

func TestMindmapResponseIdentityAndWholeDocumentWarning(t *testing.T) {
	const sourceBytes = 9 << 20
	if sourceBytes <= SourceMaxBytes {
		t.Fatal("fixture must exceed markdown source alias")
	}
	meta := ResponseMeta{
		Engine: MindmapSVG, EngineVersion: "v1", Format: "svg", Status: StatusOK,
		Warnings: []Warning{{Code: "mindmap-presentation", Message: "whole document", Start: 0, End: uint32(sourceBytes)}},
	}
	var warningOut bytes.Buffer
	if err := WriteResponse(&warningOut, meta, []byte("<svg/>")); err != nil {
		t.Fatal(err)
	}
	want := frame(`{"engine":"levnote.mindmap-svg","engineVersion":"v1","format":"svg","status":"ok","warnings":[{"code":"mindmap-presentation","message":"whole document","start":0,"end":9437184}]}`, []byte("<svg/>"), true)
	if !bytes.Equal(warningOut.Bytes(), want) {
		t.Fatal("mindmap response bytes differ from independent literal oracle")
	}
	d := NewResponseDecoder(MindmapSVG, "v1", sourceBytes)
	if _, err := d.Write(warningOut.Bytes()); err != nil {
		t.Fatal(err)
	}
	got, payload, err := d.Finish()
	if err != nil || string(payload) != "<svg/>" || len(got.Warnings) != 1 {
		t.Fatalf("mindmap response=%+v payload=%q err=%v", got, payload, err)
	}

	meta.Format = "html"
	if err := WriteResponse(io.Discard, meta, []byte("<svg/>")); err == nil {
		t.Fatal("cross-engine response format accepted")
	}
	meta.Format = "svg"
	meta.Engine = MarkdownHTML
	if err := WriteResponse(io.Discard, meta, []byte("<svg/>")); err == nil {
		t.Fatal("cross-engine response identity accepted")
	}
	meta.Engine = MindmapSVG
	meta.Warnings[0].End = uint32(sourceBytes + 1)
	var invalidWarningOut bytes.Buffer
	if err := WriteResponse(&invalidWarningOut, meta, []byte("<svg/>")); err != nil {
		t.Fatal("operation-bound writer rejected a warning within the operation source cap")
	}
	d = NewResponseDecoder(MindmapSVG, "v1", sourceBytes)
	if _, err := d.Write(invalidWarningOut.Bytes()); err == nil {
		t.Fatal("out-of-source warning accepted by decoder")
	}
}

func TestResponseDecoderRejectsForeignMetadataBeforePayload(t *testing.T) {
	mutations := []struct {
		name     string
		expected EngineID
		meta     string
	}{
		{
			"foreign-known-engine", MindmapSVG,
			`{"engine":"levnote.markdown-export","engineVersion":"v1","format":"html","status":"ok","warnings":[]}`,
		},
		{
			"same-engine-wrong-format", MindmapSVG,
			`{"engine":"levnote.mindmap-svg","engineVersion":"v1","format":"html","status":"ok","warnings":[]}`,
		},
		{
			"same-engine-wrong-version", MindmapSVG,
			`{"engine":"levnote.mindmap-svg","engineVersion":"v2","format":"svg","status":"ok","warnings":[]}`,
		},
	}
	for _, test := range mutations {
		t.Run(test.name, func(t *testing.T) {
			d := NewResponseDecoder(test.expected, "v1", 1)
			data := frame(test.meta, nil, true)[:24+len(test.meta)]
			binary.LittleEndian.PutUint64(data[16:], 6)
			n, err := d.Write(data)
			if err == nil {
				t.Fatal("foreign response metadata accepted")
			}
			if n != len(data) {
				t.Fatalf("foreign response consumed %d bytes, want metadata boundary %d", n, len(data))
			}
			if d.payload != nil || d.complete {
				t.Fatalf("payload allocated after foreign identity: payload=%v complete=%v", d.payload != nil, d.complete)
			}
			if _, next := d.Write(frame(testMeta, nil, true)); next == nil {
				t.Fatal("foreign response error was not terminal")
			}
			if _, payload, err := d.Finish(); err == nil || payload != nil {
				t.Fatal("foreign response produced a payload")
			}
		})
	}
}

func TestPreviewHardBudgetsBeforePayload(t *testing.T) {
	const requestMeta = `{"engine":"levnote.mindmap-preview","format":"mindmap-scene-v1"}`
	requestHeader := frame(requestMeta, nil, false)[:24]
	binary.LittleEndian.PutUint64(requestHeader[16:], uint64(enginecatalog.MindmapSourceMaxBytes)+1)
	counting := &countingReader{data: requestHeader}
	if _, data, err := ReadRequestFor(MindmapPreview, counting); err == nil || data != nil || counting.reads != 1 {
		t.Fatal("preview source cap accepted after allocation/read")
	}
	oversizeRequest := frame(requestMeta, []byte("source"), false)
	binary.LittleEndian.PutUint64(oversizeRequest[16:], uint64(enginecatalog.MindmapSourceMaxBytes)+1)
	counting = &countingReader{data: oversizeRequest}
	_, data, err := ReadRequestFor(MindmapPreview, counting)
	var failure *Failure
	if !errors.As(err, &failure) || failure.Code != "source-limit" || data != nil || counting.reads != 1 || len(counting.data) != len(requestMeta)+len("source") {
		t.Fatalf("preview source cap error=%v data=%v reads=%d remaining=%d", err, data != nil, counting.reads, len(counting.data))
	}

	responseHeader := frame(`{}`, nil, true)[:24]
	preview, ok := enginecatalog.Lookup(MindmapPreview)
	if !ok {
		t.Fatal("preview descriptor missing")
	}
	binary.LittleEndian.PutUint64(responseHeader[16:], uint64(preview.PayloadMaxBytes)+1)
	decoder := NewResponseDecoder(MindmapPreview, "v1", 0)
	if _, err := decoder.Write(responseHeader); err == nil || decoder.payload != nil {
		t.Fatal("preview response cap accepted/allocated")
	}

	if preview.PayloadMaxBytes != enginescene.MaxPayloadBytes {
		t.Fatalf("preview payload limit=%d want scene limit=%d", preview.PayloadMaxBytes, enginescene.MaxPayloadBytes)
	}
}

func TestPreviewResponseFragmentationAndIdentity(t *testing.T) {
	const source = "mindmap"
	payload := []byte(`{"schema":1,"layout":"orgChart"}`)
	meta := ResponseMeta{
		Engine: MindmapPreview, EngineVersion: "v1", Format: "mindmap-scene-v1", Status: StatusOK,
		Warnings: []Warning{{Code: "mindmap-preview", Message: "whole document", Start: 0, End: uint32(len(source))}},
	}
	var out bytes.Buffer
	if err := WriteResponse(&out, meta, payload); err != nil {
		t.Fatal(err)
	}
	want := frame(`{"engine":"levnote.mindmap-preview","engineVersion":"v1","format":"mindmap-scene-v1","status":"ok","warnings":[{"code":"mindmap-preview","message":"whole document","start":0,"end":7}]}`, payload, true)
	if !bytes.Equal(out.Bytes(), want) {
		t.Fatal("preview response bytes differ from independent literal oracle")
	}

	for _, size := range []int{1, 3, 24, 4096} {
		decoder := NewResponseDecoder(MindmapPreview, "v1", uint64(len(source)))
		data := append([]byte{}, want...)
		for len(data) > 0 {
			n := min(size, len(data))
			if _, err := decoder.Write(data[:n]); err != nil {
				t.Fatal(err)
			}
			data = data[n:]
		}
		got, decoded, err := decoder.Finish()
		if err != nil || got.Engine != MindmapPreview || !bytes.Equal(decoded, payload) {
			t.Fatalf("preview response=%+v payload=%q err=%v", got, decoded, err)
		}
	}

	for _, test := range []struct {
		name string
		meta ResponseMeta
	}{
		{"cross-format", ResponseMeta{Engine: MindmapPreview, EngineVersion: "v1", Format: "svg", Status: StatusOK, Warnings: []Warning{}}},
		{"cross-engine", ResponseMeta{Engine: MindmapSVG, EngineVersion: "v1", Format: "mindmap-scene-v1", Status: StatusOK, Warnings: []Warning{}}},
	} {
		if err := WriteResponse(io.Discard, test.meta, payload); err == nil {
			t.Fatalf("preview response %s accepted", test.name)
		}
	}

	foreignMeta := `{"engine":"levnote.mindmap-svg","engineVersion":"v1","format":"svg","status":"ok","warnings":[]}`
	data := frame(foreignMeta, nil, true)[:24+len(foreignMeta)]
	binary.LittleEndian.PutUint64(data[16:], uint64(len(payload)))
	decoder := NewResponseDecoder(MindmapPreview, "v1", uint64(len(source)))
	if _, err := decoder.Write(data); err == nil || decoder.payload != nil {
		t.Fatal("preview decoder allocated payload for foreign identity")
	}

	for _, test := range []struct {
		name, meta string
	}{
		{
			"same-id-wrong-format",
			`{"engine":"levnote.mindmap-preview","engineVersion":"v1","format":"svg","status":"ok","warnings":[]}`,
		},
		{
			"same-id-wrong-version",
			`{"engine":"levnote.mindmap-preview","engineVersion":"v2","format":"mindmap-scene-v1","status":"ok","warnings":[]}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := frame(test.meta, nil, true)[:24+len(test.meta)]
			binary.LittleEndian.PutUint64(data[16:], uint64(len(payload)))
			decoder := NewResponseDecoder(MindmapPreview, "v1", uint64(len(source)))
			if _, err := decoder.Write(data); err == nil || decoder.payload != nil {
				t.Fatalf("preview decoder accepted metadata or allocated payload: err=%v payload=%v", err, decoder.payload != nil)
			}
		})
	}
}
