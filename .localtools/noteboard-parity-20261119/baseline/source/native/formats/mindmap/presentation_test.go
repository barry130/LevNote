package mindmap

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestNativeLayoutOptionsAreValues(t *testing.T) {
	options := NativeLayoutOptions()
	if !reflect.DeepEqual(options, LayoutOptions{
		NodeHeight: 40, HorizontalGap: 72, VerticalGap: 24, Padding: 28,
		MinWidth: 92, MaxWidth: 380, CharWidth: 8, LineHeight: 20,
	}) {
		t.Fatal("native dimensions changed", options)
	}
	options.NodeHeight, options.Padding = 900, 901
	options.Measure = func(string) (float64, float64) { return 902, 903 }
	if next := NativeLayoutOptions(); next.NodeHeight != 40 || next.Padding != 28 || next.Measure != nil {
		t.Fatal("caller mutated shared native defaults")
	}
	// Generic Layout retains the earlier non-native defaults.
	doc := &Document{Version: Version, Layout: LayoutBoth, Root: NewNode("r", "Root")}
	layout := Layout(doc, LayoutOptions{})
	if len(layout.Nodes) != 1 || layout.Nodes[0].Height != 38 || layout.Nodes[0].Width != 88 || layout.Nodes[0].X != 24 {
		t.Fatal("generic defaults changed", layout)
	}
}

func TestDisplayLabelExactBoundaryAndPresence(t *testing.T) {
	for _, size := range []int{511, 512, 513} {
		node := NewNode("r", strings.Repeat("x", size))
		want := node.Topic
		if size > 512 {
			want = strings.Repeat("x", 512) + "…"
		}
		if got := DisplayLabel(node, node.Topic); got != want || DisplayLabelTruncated(node) != (size > 512) {
			t.Fatal("topic boundary mismatch", size, len(got))
		}
	}
	icons := []string{"⭐", "custom<&", "⭐"}
	tags := []Tag{{Text: ""}, {Text: "甲<&"}, {Text: "甲<&", Object: true}}
	node := &Node{Topic: "中心", Icons: &icons, Tags: &tags}
	if got := DisplayLabel(node, node.Topic); got != "⭐ custom<& ⭐ 中心 [ · 甲<& · 甲<&]" || DisplayLabelTruncated(node) {
		t.Fatal("ordered/duplicate/empty metadata changed", got)
	}
	emptyIcons, emptyTags := []string{}, []Tag{}
	node.Icons, node.Tags = &emptyIcons, &emptyTags
	if DisplayLabel(node, node.Topic) != "中心" || DisplayLabelTruncated(node) {
		t.Fatal("present empty arrays add display delimiters")
	}
	// Combined icon/topic/tag punctuation crosses the cap even when each field
	// fits individually. This checks the diagnostic without joining the label.
	oneIcon, oneTag := []string{""}, []Tag{{Text: ""}}
	node = &Node{Topic: strings.Repeat("x", 508), Icons: &oneIcon, Tags: &oneTag}
	if DisplayLabelTruncated(node) || len(DisplayLabel(node, node.Topic)) != 512 {
		t.Fatal("exact combined metadata boundary refused")
	}
	node.Topic += "x"
	if !DisplayLabelTruncated(node) || DisplayLabel(node, node.Topic) != " "+strings.Repeat("x", 509)+" […" {
		t.Fatal("combined punctuation boundary not truncated")
	}
	value := strings.Repeat("中🙂", 300)
	got := DisplayLabel(nil, value)
	if got != strings.Repeat("中🙂", 73)+"…" || !utf8.ValidString(got) || DisplayLabelTruncated(nil) {
		t.Fatal("nil fallback/rune boundary changed", got)
	}
}

func TestLiteralColorPurePolicy(t *testing.T) {
	for value, want := range map[string]string{
		"#ABC": "#abc", "#ABCD": "#abcd", "#123456": "#123456", "#12345678": "#12345678",
		" ReBeccAPurple ": "#663399", "red": "#ff0000", "transparent": "#00000000", "none": "#00000000",
		strings.Repeat(" ", 61) + "red": "#ff0000",
	} {
		if got, ok := LiteralColor(value); !ok || got != want {
			t.Fatalf("literal %q: got %q/%v want %q", value, got, ok, want)
		}
	}
	for _, value := range []string{"", "#badhex", "inherit", "url(#x)", "URL(//invalid)", "var(--ink)", "1px solid red", strings.Repeat(" ", 62) + "red"} {
		if got, ok := LiteralColor(value); ok || got != "" || svgColor(value, "#123456") != "#123456" {
			t.Fatal("unsupported paint/fallback changed", value, got)
		}
	}
}

func TestPresentationIssuesOrderCountsAndNil(t *testing.T) {
	for _, doc := range []*Document{nil, {}} {
		if got := PresentationIssues(doc); got != nil {
			t.Fatal("nil presentation issues", got)
		}
	}
	collapsed, branch := true, "url(https://invalid)"
	root := &Node{Topic: strings.Repeat("x", 513), Collapsed: &collapsed, Image: &Image{}, BranchColor: &branch,
		Style:    &Object{Members: []Member{{Key: "color", Raw: json.RawMessage(`4`)}, {Key: "background-color", Raw: json.RawMessage(`"inherit"`)}, {Key: "border", Raw: json.RawMessage(`"red"`)}}},
		Children: []*Node{nil, {Topic: "Hidden", Image: &Image{}, Style: &Object{Members: []Member{{Key: "color", Raw: json.RawMessage(`null`)}}}}},
	}
	doc := &Document{Root: root, Issues: []Issue{{Severity: SeverityWarning, Code: IssueUnknownKey, Message: "leave parse issues separate"}}}
	issues := PresentationIssues(doc)
	want := []Issue{
		{Severity: SeverityWarning, Code: IssueImagePlaceholder, Message: "2 个节点图片已保留；原生画布不访问文件或网络，显示为“图”占位"},
		{Severity: SeverityWarning, Code: IssueLabelTruncated, Message: "画布与 SVG 的较长节点标签仅显示前 512 字节；完整文本仍保留，主题、标记与标签的完整内容请在源码中查看和编辑"},
		{Severity: SeverityWarning, Code: IssueUnsupportedColor, Message: "4 个颜色值不受支持，画布与 SVG 使用默认颜色；原始字段已保留（支持十六进制与命名纯色）"},
	}
	if !reflect.DeepEqual(issues, want) || len(doc.Issues) != 1 || root.Style.Members[0].Key != "color" {
		t.Fatalf("presentation issues order/count/nonmutation: %+v", issues)
	}
	issues[0].Message = "caller-local change"
	if !reflect.DeepEqual(PresentationIssues(doc), want) {
		t.Fatal("presentation shared mutable issue state")
	}
}
