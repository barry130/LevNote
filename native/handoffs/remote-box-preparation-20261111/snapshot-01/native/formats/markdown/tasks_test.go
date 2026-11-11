package markdown

import (
	"errors"
	"strings"
	"testing"
)

func singleTask(t *testing.T, source string) ListItem {
	t.Helper()
	doc := Parse(source)
	if doc.Raw() != source {
		t.Fatal("Raw source changed")
	}
	if len(doc.Blocks) != 1 || doc.Blocks[0].Kind != BlockList || len(doc.Blocks[0].Items) != 1 {
		t.Fatalf("expected one list item, got %#v", doc.Blocks)
	}
	return doc.Blocks[0].Items[0]
}

func TestTaskMarkersHaveExactAbsoluteRanges(t *testing.T) {
	cases := []struct {
		name       string
		source     string
		checked    bool
		text       string
		wantMarker string
	}{
		{"unchecked", "- [ ] label", false, "label", "[ ]"},
		{"lower", "- [x] label", true, "label", "[x]"},
		{"upper", "* [X] label", true, "label", "[X]"},
		{"ordered-space", "1. [ ] label", false, "label", "[ ]"},
		{"ordered-paren", "1) [X] label", true, "label", "[X]"},
		{"ordered-zero", "0. [x] label", true, "label", "[x]"},
		{"ordered-start3", "3. [ ] label", false, "label", "[ ]"},
		{"tab-separator", "- [x]\tlabel", true, "label", "[x]"},
		{"unicode", "- [ ] 中文🙂 label", false, "中文🙂 label", "[ ]"},
		{"inline", "0. [X] `code` **bold**", true, "`code` **bold**", "[X]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			item := singleTask(t, tc.source)
			if item.Task == nil || item.Task.Checked != tc.checked {
				t.Fatalf("task metadata = %#v", item.Task)
			}
			if got := tc.source[item.Task.Range.Start:item.Task.Range.End]; got != tc.wantMarker {
				t.Fatalf("marker source range = %q", got)
			}
			if item.Text != tc.text {
				t.Fatalf("label text = %q", item.Text)
			}
			if got := tc.source[item.TextRange.Start:item.TextRange.End]; got != tc.text {
				t.Fatalf("label source range = %q", got)
			}
			if item.Task.Range.Start < item.Range.Start || item.Task.Range.End > item.Range.End {
				t.Fatalf("marker outside item: marker=%#v item=%#v", item.Task.Range, item.Range)
			}
		})
	}
}

func TestEmptyTaskLabelsUseBodyEndBoundedRange(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		checked bool
		marker  string
	}{
		{"no-label", "- [ ]", false, "[ ]"},
		{"lower-empty", "- [x]", true, "[x]"},
		{"upper-empty", "1. [X]", true, "[X]"},
		{"tab-only", "- [ ]\t", false, "[ ]"},
		{"spaces-only", "- [ ]   ", false, "[ ]"},
		{"crlf", "0. [ ]\r\n", false, "[ ]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			item := singleTask(t, tc.source)
			if item.Task == nil || item.Task.Checked != tc.checked {
				t.Fatalf("task metadata = %#v", item.Task)
			}
			markerEnd := item.Task.Range.End
			if item.TextRange.Start != markerEnd || item.TextRange.End != markerEnd {
				t.Fatalf("empty label range = %#v, want [%d,%d)", item.TextRange, markerEnd, markerEnd)
			}
			if item.Text != "" || len(item.Inline) != 0 {
				t.Fatalf("empty label content = %q %#v", item.Text, item.Inline)
			}
			if got := tc.source[item.Task.Range.Start:item.Task.Range.End]; got != tc.marker {
				t.Fatalf("marker bytes = %q, want %q", got, tc.marker)
			}
		})
	}
}

func TestInvalidTaskLookalikesRemainOrdinaryListItems(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		wantText string
	}{
		{"no-space", "- [ ]label", "[ ]label"},
		{"no-space-x", "- [x]word", "[x]word"},
		{"wrong-inner", "- [y] label", "[y] label"},
		{"inner-space", "- [ x] label", "[ x] label"},
		{"space-before-close", "- [X ] label", "[X ] label"},
		{"escaped", `- \[ ] label`, `\[ ] label`},
		{"escaped-x", `- \\[x] label`, `\\[x] label`},
		{"inline-code", "- `[ ]` label", "`[ ]` label"},
		{"quoted", `- "[ ]" label`, `"[ ]" label`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			item := singleTask(t, tc.source)
			if item.Task != nil {
				t.Fatalf("lookalike became task: %#v", item.Task)
			}
			if item.Text != tc.wantText {
				t.Fatalf("ordinary text changed = %q, want %q", item.Text, tc.wantText)
			}
			if got := tc.source[item.TextRange.Start:item.TextRange.End]; got != tc.wantText {
				t.Fatalf("ordinary text range changed = %q, want %q", got, tc.wantText)
			}
		})
	}
}

func TestTaskLookalikesOutsideCurrentListItemRemainNonTasks(t *testing.T) {
	cases := []struct {
		name       string
		source     string
		blockCount int
	}{
		{"fenced", "```text\n- [ ] fenced\n```\n", 1},
		{"unterminated-fence", "```text\n- [ ] fenced\n", 1},
		{"quote", "> - [ ] quoted\n", 1},
		{"paragraph", "text - [ ] paragraph\n", 1},
		{"indented", "  - [ ] indented\n", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := Parse(tc.source)
			if doc.Raw() != tc.source || len(doc.Blocks) != tc.blockCount {
				t.Fatalf("blocks/raw changed: %#v", doc.Blocks)
			}
			for _, block := range doc.Blocks {
				if block.Kind != BlockList {
					continue
				}
				for _, item := range block.Items {
					if item.Task != nil {
						t.Fatalf("outside-body marker became task: %#v", item.Task)
					}
				}
			}
		})
	}
}

func TestTaskContinuationKeepsRealTaskAndOrdinaryParagraph(t *testing.T) {
	source := "- [ ] task\n  [ ] continuation\n"
	doc := Parse(source)
	if doc.Raw() != source || len(doc.Blocks) != 2 {
		t.Fatalf("blocks/raw changed: %#v", doc.Blocks)
	}
	list := doc.Blocks[0]
	if list.Kind != BlockList || len(list.Items) != 1 || list.Items[0].Task == nil {
		t.Fatalf("first list item lost its real task: %#v", list)
	}
	paragraph := doc.Blocks[1]
	if paragraph.Kind != BlockParagraph || paragraph.Text != "  [ ] continuation" {
		t.Fatalf("continuation lookalike was not retained as paragraph: %#v", paragraph)
	}
}

func TestTaskRangesUseExplicitUnicodeByteOffsets(t *testing.T) {
	source := "标题🙂\r\n- [x] 中文🙂 label\r\n"
	doc := Parse(source)
	if doc.Raw() != source || len(doc.Blocks) != 2 {
		t.Fatalf("blocks/raw changed: %#v", doc.Blocks)
	}
	item := doc.Blocks[1].Items[0]
	if item.Task == nil || !item.Task.Checked {
		t.Fatalf("task metadata = %#v", item.Task)
	}
	if item.Range != (SourceRange{Start: 12, End: 36}) {
		t.Fatalf("item range = %#v", item.Range)
	}
	if item.Task.Range != (SourceRange{Start: 14, End: 17}) {
		t.Fatalf("marker range = %#v", item.Task.Range)
	}
	if item.TextRange != (SourceRange{Start: 18, End: 34}) {
		t.Fatalf("label range = %#v", item.TextRange)
	}
	if item.Text != "中文🙂 label" {
		t.Fatalf("label text = %q", item.Text)
	}
}

func TestTaskExportIsInertEscapedAndPartialRefused(t *testing.T) {
	source := "- [ ] <plain> `code`\n- [x] **完成** [x](tasks.md)\n- [X]\n0. [ ] 0 item\n1. [ ] plain\n"
	doc := Parse(source)
	if doc.Raw() != source || doc.Partial {
		t.Fatalf("source/partial state = %#v", doc)
	}
	html, warnings, err := ExportHTMLChecked(doc)
	if err != nil || len(warnings) != 0 {
		t.Fatal(err, warnings)
	}
	for _, expected := range []string{
		`<input type="checkbox" disabled> &lt;plain&gt; <code>code</code>`,
		`<input type="checkbox" disabled checked> <strong>完成</strong> <a href="tasks.md">x</a>`,
		`<input type="checkbox" disabled checked></li>`,
		`<ol start="0">`,
		`<input type="checkbox" disabled> 0 item`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("missing task output %q in %s", expected, html)
		}
	}
	if strings.Contains(html, "onclick") || strings.Contains(html, "script") {
		t.Fatal("task export emitted active content")
	}

	partial := Parse("- [ ] task\n" + strings.Repeat("\n", maxLines+1))
	if !partial.Partial {
		t.Fatal("fixture did not become partial")
	}
	output, _, err := ExportHTMLChecked(partial)
	if len(output) != 0 || !errors.Is(err, ErrPartialDocument) {
		t.Fatalf("partial task export = %q %v", output, err)
	}
}

func TestUnsafeTaskLinkLabelIsEscapedWithoutHref(t *testing.T) {
	source := "- [x] <plain> [alert](https://example.com)\n"
	doc := Parse(source)
	if doc.Raw() != source || doc.Partial {
		t.Fatalf("source/partial state = %#v", doc)
	}
	output, warnings, err := ExportHTMLChecked(doc)
	if err != nil {
		t.Fatal(err)
	}
	want := "<!doctype html>\n<meta charset=\"utf-8\">\n<body>\n<ul>\n<li><input type=\"checkbox\" disabled checked> &lt;plain&gt; alert</li>\n</ul>\n</body>\n"
	if output != want {
		t.Fatalf("unsafe task export = %q, want %q", output, want)
	}
	if len(warnings) != 1 || warnings[0].Code != "unsafe-link" {
		t.Fatalf("unsafe-link warnings = %#v", warnings)
	}
	if strings.Contains(output, "href") || strings.Contains(output, "javascript:") {
		t.Fatal("unsafe task link was emitted")
	}
}

func TestNonTaskListExportRemainsExact(t *testing.T) {
	doc := Parse("- plain <b>\n* item\n")
	output, warnings, err := ExportHTMLChecked(doc)
	if err != nil {
		t.Fatal(err)
	}
	want := "<!doctype html>\n<meta charset=\"utf-8\">\n<body>\n<ul>\n<li>plain &lt;b&gt;</li>\n<li>item</li>\n</ul>\n</body>\n"
	if output != want {
		t.Fatalf("non-task export changed = %q, want %q", output, want)
	}
	if len(warnings) != 0 {
		t.Fatalf("non-task export warnings = %#v", warnings)
	}
}

func TestTaskMarkerPreservesCRLFAndTrailingWhitespace(t *testing.T) {
	source := "3. [X] label   \r\n- [ ]\t  \r\n"
	doc := Parse(source)
	if doc.Raw() != source || len(doc.Blocks) != 2 {
		t.Fatalf("blocks/raw changed: %#v", doc.Blocks)
	}
	ordered := doc.Blocks[0].Items[0]
	if ordered.Task == nil || !ordered.Task.Checked || ordered.Text != "label" {
		t.Fatalf("ordered task = %#v", ordered)
	}
	if source[ordered.Task.Range.Start:ordered.Task.Range.End] != "[X]" {
		t.Fatalf("ordered marker range = %q", source[ordered.Task.Range.Start:ordered.Task.Range.End])
	}
	bullet := doc.Blocks[1].Items[0]
	if bullet.Task == nil || bullet.Task.Checked || bullet.TextRange.Start != bullet.Task.Range.End || bullet.TextRange.End != bullet.Task.Range.End {
		t.Fatalf("empty task range = %#v", bullet)
	}
	if source[bullet.Range.Start:bullet.Range.End] != "- [ ]\t  \r\n" {
		t.Fatalf("item raw bytes changed: %q", source[bullet.Range.Start:bullet.Range.End])
	}
}
