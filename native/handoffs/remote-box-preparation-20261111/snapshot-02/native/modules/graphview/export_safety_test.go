package graphview

import (
	"bytes"
	"encoding/xml"
	"io"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/mindmap"
	"levnote.local/native/ui/editor"
)

func TestGraphColorsAreLiteralAndMatchNativeRendering(t *testing.T) {
	for _, value := range []string{"#123", "#1234", "#123456", "#12345678", " ReBeccAPurple ", "red", "transparent", "none"} {
		literal, ok := literalGraphColor(value)
		color, nativeOK := graphColor(value)
		if !ok || !nativeOK || color != ui.Hex(literal) {
			t.Fatalf("supported color mismatch %q: %q %v %v", value, literal, ok, nativeOK)
		}
	}
	for _, value := range []string{"", "url(https://invalid/paint.svg#x)", "URL(//invalid/paint)", "url(#paint)", "var(--paint)", "inherit", "#badhex", "1px solid red", strings.Repeat(" ", 1024*1024) + "red"} {
		if _, ok := literalGraphColor(value); ok {
			t.Fatalf("nonliteral accepted: %.64q", value)
		}
		if _, ok := graphColor(value); ok {
			t.Fatalf("native rejected color mismatch: %.64q", value)
		}
	}
}

func TestMindmapExportRejectsExternalPaintWithoutChangingSource(t *testing.T) {
	source := strings.Replace(testMindmap, `"topic": "中心"`, `"topic": "中心", "branchColor": "url(https://invalid/branch.svg#paint)", "style": {"background-color":"url(//invalid/fill)", "color":"var(--ink)", "border":"red", "vendor":"keep"}`, 1)
	buffer := text.New(source)
	v := newView(kindMindmap, editor.Host{Buffer: buffer})
	svg, err := v.Export("svg")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(svg, []byte("invalid")) || bytes.Contains(svg, []byte("var(")) || !bytes.Contains(svg, []byte(`stroke="#ff0000"`)) {
		t.Fatalf("unsafe or mismatched SVG paints: %s", svg)
	}
	if !strings.Contains(strings.Join(v.Warnings(), " "), "3 个颜色值不受支持") {
		t.Fatal("paint fallback was silent", v.Warnings())
	}
	if buffer.String() != source {
		t.Fatal("export changed unsupported source fields")
	}
	tester := ui.NewTester(v.Draw, 900, 620)
	focusDraft(t, tester)
	tester.Type("改了标签")
	parsed := parseMind(t, buffer)
	if parsed.Root.BranchColor == nil || *parsed.Root.BranchColor != "url(https://invalid/branch.svg#paint)" {
		t.Fatal("editing removed raw branch color")
	}
	for key, want := range map[string]string{"background-color": "url(//invalid/fill)", "color": "var(--ink)", "border": "red", "vendor": "keep"} {
		if got, ok := objectString(*parsed.Root.Style, key); !ok || got != want {
			t.Fatalf("editing lost style metadata %q: %q", key, got)
		}
	}
}

func TestMindmapSVGBackgroundCoversNegativeViewBox(t *testing.T) {
	doc := parseMind(t, text.New(testMindmap))
	layout := mindmap.Layout(doc, mindmap.LayoutOptions{})
	layout.Bounds = mindmap.Rect{X: -210, Y: -150, Width: 520, Height: 340}
	svg, err := exportMindmapSVG(doc, layout)
	if err != nil {
		t.Fatal(err)
	}
	decoder := xml.NewDecoder(bytes.NewReader(svg))
	var bounds []string
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		attrs := make(map[string]string)
		for _, attr := range start.Attr {
			attrs[attr.Name.Local] = attr.Value
		}
		if start.Name.Local == "svg" {
			bounds = strings.Fields(attrs["viewBox"])
		}
		if start.Name.Local == "rect" {
			if len(bounds) != 4 || attrs["x"] != bounds[0] || attrs["y"] != bounds[1] || attrs["width"] != bounds[2] || attrs["height"] != bounds[3] || attrs["fill"] != "#ffffff" {
				t.Fatalf("background does not cover viewBox: bounds=%v rect=%v", bounds, attrs)
			}
			return
		}
	}
	t.Fatal("background rect missing")
}

func mindmapSVGText(t *testing.T, data []byte) []string {
	t.Helper()
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var labels []string
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return labels
		}
		if err != nil {
			t.Fatalf("invalid SVG XML: %v", err)
		}
		if start, ok := token.(xml.StartElement); ok && start.Name.Local == "tspan" {
			var label string
			if err := decoder.DecodeElement(&label, &start); err != nil {
				t.Fatal(err)
			}
			labels = append(labels, label)
		}
	}
}

func TestMindmapSVGMarkersUnicodeEscapingToggleAndClear(t *testing.T) {
	source := markerFixture([]string{"2️⃣", "custom<&🙂", "⭐", "custom<&🙂"}, true)
	buffer := text.New(source)
	v := newView(kindMindmap, editor.Host{Buffer: buffer})
	for _, step := range []struct {
		action int
		want   string
	}{
		{-1, "2️⃣ custom<&🙂 ⭐ custom<&🙂 中心 [标签]"},
		{15, "2️⃣ custom<&🙂 custom<&🙂 中心 [标签]"},
		{2, "custom<&🙂 custom<&🙂 3️⃣ 中心 [标签]"},
		{markerClear, "中心 [标签]"},
	} {
		if step.action >= 0 {
			if err := v.activateMarker(markerTarget(v, step.action)); err != nil {
				t.Fatal(err)
			}
		}
		before, history, selection := buffer.String(), buffer.History(), buffer.Selection()
		svg, err := v.Export("svg")
		if err != nil {
			t.Fatal(err)
		}
		labels := mindmapSVGText(t, svg)
		if !slices.Contains(labels, step.want) || !slices.Contains(labels, "❓ 子主题") {
			t.Fatalf("SVG omitted exact markers/topic/tags: labels=%q want=%q", labels, step.want)
		}
		if strings.Contains(step.want, "custom<&🙂") && !bytes.Contains(svg, []byte("custom&lt;&amp;🙂")) {
			t.Fatal("custom marker not escaped at XML boundary")
		}
		if buffer.String() != before || buffer.History() != history || buffer.Selection() != selection {
			t.Fatal("SVG export mutated source/history/selection")
		}
	}
}

func TestMindmapSVGMarkersAndTagsUseBoundedSharedPresentation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values []string
		want   string
	}{
		{"large-glyph", []string{strings.Repeat("中🙂", 30000)}, strings.Repeat("中🙂", 73) + "…"},
		{"many-custom", slices.Repeat([]string{"custom<&"}, 30000), strings.Repeat("custom<& ", 56) + "custom<&…"},
		{"many-empty", slices.Repeat([]string{""}, 30000), strings.Repeat(" ", 512) + "…"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := markerFixture(tc.values, true)
			buffer := text.New(source)
			v := newView(kindMindmap, editor.Host{Buffer: buffer})
			svg, err := v.Export("svg")
			if err != nil {
				t.Fatal(err)
			}
			labels := mindmapSVGText(t, svg)
			if !slices.Contains(labels, tc.want) || len(svg) > 12<<10 {
				t.Fatalf("SVG label prefix/output not bounded: len=%d labels=%q", len(svg), labels)
			}
			for _, label := range labels {
				if len(label) > maxGraphLabelBytes+len("…") || !utf8.ValidString(label) {
					t.Fatal("SVG label exceeded bounded rune-safe prefix")
				}
			}
			warning := strings.Join(v.Warnings(), " ")
			if !strings.Contains(warning, "画布与 SVG") || !strings.Contains(warning, "完整内容请在源码中") || strings.Contains(warning, "可在节点文本中编辑") {
				t.Fatal("marker/tag truncation guidance is false or missing", warning)
			}
			if buffer.String() != source || buffer.CanUndo() {
				t.Fatal("bounded SVG altered oversized marker content")
			}
		})
	}
	// Tags share the same source-only guidance and bounded presentation.
	source := strings.Replace(markerFixture(nil, false), `"text":"标签"`, `"text":"`+strings.Repeat("标", 30000)+`"`, 1)
	buffer := text.New(source)
	v := newView(kindMindmap, editor.Host{Buffer: buffer})
	svg, err := v.Export("svg")
	want := "中心 [" + strings.Repeat("标", 168) + "…"
	if err != nil || !slices.Contains(mindmapSVGText(t, svg), want) || !strings.Contains(strings.Join(v.Warnings(), " "), "完整内容请在源码中") || buffer.String() != source {
		t.Fatal("large tags lost bounded SVG/source guidance", err)
	}
}
