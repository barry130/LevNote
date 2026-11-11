package main

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"levnote.local/native/core/engineipc"
	"levnote.local/native/core/enginescene"
)

func TestPreviewWorkerFourLayoutsIndependentGeometryPaintAndHiddenChild(t *testing.T) {
	// Geometry is authored from the fixed native dimensions, not obtained from
	// Layout or Encode. The hidden child, source-only links and unknown raw
	// numbers must never become extra scene values.
	const sourceTemplate = `{"format":"levnote.mindmap","version":1,"layout":%q,"futureTop":1.00,"root":{"id":"r","topic":"Root","icons":["⭐"],"tags":["甲"],"note":"private note","hyperLink":"https://invalid/private","style":{"background-color":"#aBc","border":"#1357","color":" ReBeccAPurple "},"branchColor":"#0F08","children":[{"id":"a","topic":"A","collapsed":true,"style":{"background-color":"none","color":"red"},"children":[{"id":"hidden","topic":"HIDDEN"}]},{"id":"b","topic":"B","branchColor":"#12345678","futureNode":9007199254740993}]}}`
	cases := []struct {
		layout string
		bounds enginescene.Rect
		nodes  [3]enginescene.Point
		edges  [2][2]enginescene.Point
	}{
		{"logicalStructureLeft", enginescene.Rect{Width: 312, Height: 160},
			[3]enginescene.Point{{X: 192, Y: 60}, {X: 28, Y: 28}, {X: 28, Y: 92}},
			[2][2]enginescene.Point{{{X: 192, Y: 80}, {X: 120, Y: 48}}, {{X: 192, Y: 80}, {X: 120, Y: 112}}}},
		{"logicalStructureRight", enginescene.Rect{Width: 312, Height: 160},
			[3]enginescene.Point{{X: 28, Y: 60}, {X: 192, Y: 28}, {X: 192, Y: 92}},
			[2][2]enginescene.Point{{{X: 120, Y: 80}, {X: 192, Y: 48}}, {{X: 120, Y: 80}, {X: 192, Y: 112}}}},
		{"logicalStructure", enginescene.Rect{Width: 476, Height: 96},
			[3]enginescene.Point{{X: 192, Y: 28}, {X: 28, Y: 28}, {X: 356, Y: 28}},
			[2][2]enginescene.Point{{{X: 192, Y: 48}, {X: 120, Y: 48}}, {{X: 284, Y: 48}, {X: 356, Y: 48}}}},
		{"orgChart", enginescene.Rect{Width: 312, Height: 160},
			[3]enginescene.Point{{X: 110, Y: 28}, {X: 28, Y: 92}, {X: 192, Y: 92}},
			[2][2]enginescene.Point{{{X: 156, Y: 68}, {X: 74, Y: 92}}, {{X: 156, Y: 68}, {X: 238, Y: 92}}}},
	}
	for _, tc := range cases {
		t.Run(tc.layout, func(t *testing.T) {
			source := []byte(fmt.Sprintf(sourceTemplate, tc.layout))
			before := bytes.Clone(source)
			meta, payload := callPreviewWorker(t, source)
			if meta.Status != engineipc.StatusOK || meta.Error != nil {
				t.Fatalf("success metadata: %+v", meta)
			}
			wantWarnings := []engineipc.Warning{
				{Code: "UNKNOWN_KEY", Message: `node "b" has 1 preserved unknown field(s)`, End: uint32(len(source))},
				{Code: "UNKNOWN_KEY", Message: "preserved 1 unknown top-level field(s)", End: uint32(len(source))},
			}
			if !reflect.DeepEqual(meta.Warnings, wantWarnings) {
				t.Fatalf("warning order/ranges: %+v want %+v", meta.Warnings, wantWarnings)
			}
			want := enginescene.Scene{
				Schema: 1, Layout: tc.layout, Bounds: tc.bounds,
				Nodes: []enginescene.Node{
					{ID: "r", Label: "⭐ Root [甲]", Width: 92, Height: 40, Shape: "rounded", Fill: "#aabbccff", Stroke: "#11335577", Text: "#663399ff", Root: true},
					{ID: "a", Label: "A", Width: 92, Height: 40, Shape: "rounded", Fill: "#00000000", Text: "#ff0000ff", Collapsed: true},
					{ID: "b", Label: "B", Width: 92, Height: 40, Shape: "rounded"},
				},
				Edges: []enginescene.Edge{
					{FromID: "r", ToID: "a", Stroke: "#00ff0088"},
					{FromID: "r", ToID: "b", Stroke: "#12345678"},
				},
			}
			for i, point := range tc.nodes {
				want.Nodes[i].X, want.Nodes[i].Y = point.X, point.Y
			}
			for i, points := range tc.edges {
				want.Edges[i].From, want.Edges[i].To = points[0], points[1]
			}
			got, err := enginescene.Decode(payload)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("independent scene=%+v want=%+v err=%v", got, want, err)
			}
			for _, forbidden := range []string{"hidden", "HIDDEN", "private", "futureTop", "futureNode", "9007199254740993", "1.00", "branchColor"} {
				if bytes.Contains(payload, []byte(forbidden)) {
					t.Fatalf("source-only value %q entered scene", forbidden)
				}
			}
			if !bytes.Equal(source, before) {
				t.Fatal("source bytes changed")
			}
		})
	}
}

func TestPreviewWorkerIndependentLiteralEscapingAndNoThemeDefaults(t *testing.T) {
	source := []byte(`{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"中🙂<>&'\"\\\u0000\nX"}}`)
	// Nine runes before the newline: 9*8+28=100; two lines: 2*20+16=56.
	const want = `{"schema":1,"layout":"logicalStructure","nodeCount":1,"edgeCount":0,"bounds":{"x":0,"y":0,"width":156,"height":112},"nodes":[{"id":"r","label":"中🙂<>&'\"\\\u0000\nX","x":28,"y":28,"width":100,"height":56,"shape":"rounded","fill":"","stroke":"","text":"","root":true,"collapsed":false}],"edges":[]}`
	meta, payload := callPreviewWorker(t, source)
	if meta.Status != engineipc.StatusOK || len(meta.Warnings) != 0 || string(payload) != want {
		t.Fatalf("authored scene wire: meta=%+v payload=%s want=%s", meta, payload, want)
	}
	metaAgain, payloadAgain := callPreviewWorker(t, source)
	if !reflect.DeepEqual(metaAgain, meta) || !bytes.Equal(payloadAgain, payload) {
		t.Fatal("identical source did not produce identical frame values")
	}
}

func TestPreviewWorkerColorNormalizationAndUnsupportedFallbacks(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"#aBc", "#aabbccff"}, {"#aBcD", "#aabbccdd"},
		{"#010203", "#010203ff"}, {"#01020304", "#01020304"},
		{"  ReBeccAPurple ", "#663399ff"}, {"RED", "#ff0000ff"},
		{"transparent", "#00000000"}, {"none", "#00000000"},
		{"", ""}, {"inherit", ""}, {"var(--a)", ""},
		{"url(https://invalid)", ""}, {"rgb(1,2,3)", ""},
		{"#12gg", ""}, {strings.Repeat(" ", 65) + "red", ""},
	} {
		if got := normalizedPaint(tc.input); got != tc.want {
			t.Fatalf("normalized paint %q=%q want=%q", tc.input, got, tc.want)
		}
	}
	source := []byte(`{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"R","style":{"background-color":null,"border":42,"color":"url(https://invalid)"},"branchColor":"inherit","children":[{"id":"a","topic":"A"}]}}`)
	meta, payload := callPreviewWorker(t, source)
	wantWarnings := []engineipc.Warning{{Code: "UNSUPPORTED_COLOR", Message: "4 个颜色值不受支持，画布与 SVG 使用默认颜色；原始字段已保留（支持十六进制与命名纯色）", End: uint32(len(source))}}
	if meta.Status != engineipc.StatusOK || !reflect.DeepEqual(meta.Warnings, wantWarnings) {
		t.Fatalf("unsupported paint warnings: %+v", meta)
	}
	scene, err := enginescene.Decode(payload)
	if err != nil || len(scene.Nodes) != 2 || len(scene.Edges) != 1 {
		t.Fatalf("paint fallback scene=%+v err=%v", scene, err)
	}
	for _, node := range scene.Nodes {
		if node.Fill != "" || node.Stroke != "" || node.Text != "" {
			t.Fatalf("unsupported paint became literal override: %+v", node)
		}
	}
	if scene.Edges[0].Stroke != "" || bytes.Contains(payload, []byte("https://")) {
		t.Fatal("unsupported edge paint or URL entered scene")
	}
}

func TestPreviewWorkerWarningOrderHiddenImageAndEffectiveLayout(t *testing.T) {
	source := []byte(`{"format":"levnote.mindmap","version":2,"layout":"future-layout","futureTop":1,"root":{"id":"r","topic":"R","collapsed":true,"children":[{"id":"hidden","topic":"` + strings.Repeat("字", 513) + `","futureChild":1,"image":{"url":"https://private/image.png","width":40,"height":40},"style":{"border":"bad paint"}}]}}`)
	before := bytes.Clone(source)
	meta, payload := callPreviewWorker(t, source)
	wantCodes := []string{"FUTURE_VERSION", "UNKNOWN_LAYOUT", "UNKNOWN_KEY", "UNKNOWN_KEY", "UNKNOWN_LAYOUT", "IMAGE_PLACEHOLDER", "LABEL_TRUNCATED", "UNSUPPORTED_COLOR"}
	var gotCodes []string
	for _, warning := range meta.Warnings {
		gotCodes = append(gotCodes, warning.Code)
		if warning.Start != 0 || warning.End != uint32(len(source)) {
			t.Fatalf("warning range: %+v", warning)
		}
	}
	if meta.Status != engineipc.StatusOK || !reflect.DeepEqual(gotCodes, wantCodes) {
		t.Fatalf("warning order=%q want=%q meta=%+v", gotCodes, wantCodes, meta)
	}
	if meta.Warnings[5].Message != "节点图片引用已保留；此只读预览不加载或显示图片，完整引用请在源码中查看" {
		t.Fatalf("image capability message: %q", meta.Warnings[5].Message)
	}
	if meta.Warnings[1].Message != `unknown layout "future-layout" is preserved and renders as "logicalStructure"` ||
		meta.Warnings[4].Message != `layout "future-layout" is preserved and rendered as "logicalStructure"` {
		t.Fatalf("unknown-layout warning wording/order: %+v", meta.Warnings)
	}
	scene, err := enginescene.Decode(payload)
	if err != nil || scene.Layout != "logicalStructure" || len(scene.Nodes) != 1 || !scene.Nodes[0].Collapsed || len(scene.Edges) != 0 {
		t.Fatalf("collapsed effective scene=%+v err=%v", scene, err)
	}
	for _, forbidden := range []string{"private", "hidden", "字", "image", "future-layout"} {
		if bytes.Contains(payload, []byte(forbidden)) {
			t.Fatalf("hidden/metadata %q entered scene", forbidden)
		}
	}
	if !bytes.Equal(source, before) {
		t.Fatal("read-only future source changed")
	}
}
