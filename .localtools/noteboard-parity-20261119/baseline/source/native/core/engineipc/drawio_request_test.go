package engineipc

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestLegacyRequestBytesRemainExactAndRejectPageBeforeSource(t *testing.T) {
	source := []byte("source")
	for _, test := range []struct {
		name   string
		engine EngineID
		format string
		meta   string
	}{
		{"markdown", MarkdownHTML, "html", `{"engine":"levnote.markdown-export","format":"html"}`},
		{"mindmap", MindmapSVG, "svg", `{"engine":"levnote.mindmap-svg","format":"svg"}`},
		{"preview", MindmapPreview, "mindmap-scene-v1", `{"engine":"levnote.mindmap-preview","format":"mindmap-scene-v1"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			prefix, err := EncodeRequestPrefix(RequestMeta{Engine: test.engine, Format: test.format}, uint64(len(source)))
			if err != nil {
				t.Fatal(err)
			}
			want := frame(test.meta, source, false)
			if !bytes.Equal(append(prefix, source...), want) {
				t.Fatalf("legacy bytes changed: %q", prefix[:len(prefix)-len(source)])
			}

			foreign := frame(strings.TrimSuffix(test.meta, "}")+`,"page":{"index":0,"id":""}}`, source, false)
			reader := &countingReader{data: foreign}
			_, payload, err := ReadRequestFor(test.engine, reader)
			var failure *Failure
			if !errors.As(err, &failure) || failure.Code != "invalid-request" || payload != nil || reader.reads != 2 || string(reader.data) != "source" {
				t.Fatalf("legacy page rejection reads=%d remaining=%q payload=%v err=%v", reader.reads, reader.data, payload, err)
			}
		})
	}
}

func TestDrawioRequestPageExactBytesAndBounds(t *testing.T) {
	source := []byte("source")
	page := &RequestPage{Index: 1, ID: "second"}
	prefix, err := EncodeRequestPrefix(RequestMeta{Engine: DrawioSVG, Format: "svg", Page: page}, uint64(len(source)))
	if err != nil {
		t.Fatal(err)
	}
	const metadata = `{"engine":"levnote.drawio-svg","format":"svg","page":{"index":1,"id":"second"}}`
	want := frame(metadata, source, false)
	if !bytes.Equal(append(prefix, source...), want) {
		t.Fatal("draw.io request bytes differ from independent literal")
	}
	got, payload, err := ReadRequestFor(DrawioSVG, &oneByteReader{data: want})
	if err != nil || got.Page == nil || *got.Page != *page || !bytes.Equal(payload, source) {
		t.Fatalf("draw.io round trip meta=%+v payload=%q err=%v", got, payload, err)
	}

	for _, index := range []int{0, PageIndexMax} {
		if _, err := EncodeRequestPrefix(RequestMeta{Engine: DrawioSVG, Format: "svg", Page: &RequestPage{Index: index, ID: ""}}, 0); err != nil {
			t.Fatalf("valid page index %d refused: %v", index, err)
		}
	}
	if _, err := EncodeRequestPrefix(RequestMeta{Engine: DrawioSVG, Format: "svg", Page: &RequestPage{Index: 0, ID: strings.Repeat("x", PageIDMaxBytes)}}, 0); err != nil {
		t.Fatalf("exact page ID cap refused: %v", err)
	}
	for name, meta := range map[string]RequestMeta{
		"old-engine-page": {Engine: MarkdownHTML, Format: "html", Page: &RequestPage{}},
		"drawio-missing":  {Engine: DrawioSVG, Format: "svg"},
		"negative":        {Engine: DrawioSVG, Format: "svg", Page: &RequestPage{Index: -1}},
		"over-index":      {Engine: DrawioSVG, Format: "svg", Page: &RequestPage{Index: PageIndexMax + 1}},
		"over-id":         {Engine: DrawioSVG, Format: "svg", Page: &RequestPage{ID: strings.Repeat("x", PageIDMaxBytes+1)}},
		"invalid-utf8":    {Engine: DrawioSVG, Format: "svg", Page: &RequestPage{ID: string([]byte{0xff})}},
	} {
		t.Run("encode-"+name, func(t *testing.T) {
			if data, err := EncodeRequestPrefix(meta, 0); err == nil || data != nil || !errors.Is(err, ErrProtocol) {
				t.Fatalf("invalid page encoded: data=%v err=%v", data != nil, err)
			}
		})
	}
}

func TestDrawioStrictPageMetadataRejectedBeforePayload(t *testing.T) {
	longID := strings.Repeat("x", PageIDMaxBytes+1)
	for name, meta := range map[string]string{
		"absent":                 `{"engine":"levnote.drawio-svg","format":"svg"}`,
		"null-page":              `{"engine":"levnote.drawio-svg","format":"svg","page":null}`,
		"unknown-top":            `{"engine":"levnote.drawio-svg","format":"svg","page":{"index":0,"id":""},"path":"x"}`,
		"duplicate-page":         `{"engine":"levnote.drawio-svg","format":"svg","page":{"index":0,"id":""},"page":{"index":0,"id":""}}`,
		"case-page":              `{"engine":"levnote.drawio-svg","format":"svg","Page":{"index":0,"id":""}}`,
		"page-array":             `{"engine":"levnote.drawio-svg","format":"svg","page":["0",""]}`,
		"duplicate-index":        `{"engine":"levnote.drawio-svg","format":"svg","page":{"index":0,"index":1,"id":""}}`,
		"duplicate-id":           `{"engine":"levnote.drawio-svg","format":"svg","page":{"index":0,"id":"","id":"x"}}`,
		"unknown-page-key":       `{"engine":"levnote.drawio-svg","format":"svg","page":{"index":0,"id":"","name":"x"}}`,
		"missing-index":          `{"engine":"levnote.drawio-svg","format":"svg","page":{"id":""}}`,
		"missing-id":             `{"engine":"levnote.drawio-svg","format":"svg","page":{"index":0}}`,
		"null-index":             `{"engine":"levnote.drawio-svg","format":"svg","page":{"index":null,"id":""}}`,
		"null-id":                `{"engine":"levnote.drawio-svg","format":"svg","page":{"index":0,"id":null}}`,
		"index-string":           `{"engine":"levnote.drawio-svg","format":"svg","page":{"index":"0","id":""}}`,
		"id-number":              `{"engine":"levnote.drawio-svg","format":"svg","page":{"index":0,"id":7}}`,
		"negative":               `{"engine":"levnote.drawio-svg","format":"svg","page":{"index":-1,"id":""}}`,
		"over-index":             `{"engine":"levnote.drawio-svg","format":"svg","page":{"index":250000,"id":""}}`,
		"float-index":            `{"engine":"levnote.drawio-svg","format":"svg","page":{"index":1.0,"id":""}}`,
		"surrogate-id":           `{"engine":"levnote.drawio-svg","format":"svg","page":{"index":0,"id":"\ud800"}}`,
		"over-id":                `{"engine":"levnote.drawio-svg","format":"svg","page":{"index":0,"id":"` + longID + `"}}`,
		"trailing-page-metadata": `{"engine":"levnote.drawio-svg","format":"svg","page":{"index":0,"id":""}}x`,
	} {
		t.Run(name, func(t *testing.T) {
			reader := &countingReader{data: frame(meta, []byte("source"), false)}
			_, payload, err := ReadRequestFor(DrawioSVG, reader)
			var failure *Failure
			if !errors.As(err, &failure) || failure.Code != "invalid-request" || payload != nil || reader.reads != 2 || string(reader.data) != "source" {
				t.Fatalf("bad page metadata reads=%d remaining=%q payload=%v err=%v", reader.reads, reader.data, payload, err)
			}
		})
	}

	const validEmptyID = `{"engine":"levnote.drawio-svg","format":"svg","page":{"index":0,"id":""}}`
	if meta, _, err := ReadRequestFor(DrawioSVG, bytes.NewReader(frame(validEmptyID, []byte("x"), false))); err != nil || meta.Page == nil || meta.Page.ID != "" {
		t.Fatalf("empty parsed-page identity rejected: meta=%+v err=%v", meta, err)
	}
}
