package markdown

import (
	"reflect"
	"strings"
	"testing"
)

func TestNestedListDescriptorsPreserveSourceAndTaskRanges(t *testing.T) {
	source := "标题🙂\r\n- parent\r\n    3. [ ] 子🙂\r\n         - [X] 孙\r\n- tail\r\n"
	doc := Parse(source)
	if doc.Partial || doc.Raw() != source || len(doc.Blocks) != 2 || doc.Blocks[1].Kind != BlockList {
		t.Fatalf("nested list parse=%#v", doc)
	}
	list := doc.Blocks[1]
	if len(list.Items) != 4 {
		t.Fatalf("flattened descriptors=%#v", list.Items)
	}
	wantDepth := []int{0, 1, 2, 0}
	wantOrdered := []bool{false, true, false, false}
	wantRanges := []string{"- parent\r\n", "    3. [ ] 子🙂\r\n", "         - [X] 孙\r\n", "- tail\r\n"}
	for index, item := range list.Items {
		if item.Depth != wantDepth[index] || item.Ordered != wantOrdered[index] ||
			item.Range.Start < list.Range.Start || item.Range.End > list.Range.End ||
			source[item.Range.Start:item.Range.End] != wantRanges[index] {
			t.Fatalf("item %d descriptor=%#v source=%q want=%q", index, item, source[item.Range.Start:item.Range.End], wantRanges[index])
		}
	}
	child := list.Items[1]
	if child.Task == nil || source[child.Task.Range.Start:child.Task.Range.End] != "[ ]" ||
		source[child.TextRange.Start:child.TextRange.End] != "子🙂" {
		t.Fatalf("nested task source ranges=%#v", child)
	}
	grandchild := list.Items[2]
	if grandchild.Task == nil || !grandchild.Task.Checked ||
		source[grandchild.Task.Range.Start:grandchild.Task.Range.End] != "[X]" {
		t.Fatalf("grandchild task=%#v", grandchild)
	}
}

func TestNestedListExportKeepsMixedListStructureAndInertTasks(t *testing.T) {
	doc := Parse("- parent\n    3. [ ] child\n         - [x] grandchild\n- tail\n")
	html, warnings, err := ExportHTMLChecked(doc)
	if err != nil || len(warnings) != 0 {
		t.Fatalf("export error=%v warnings=%#v", err, warnings)
	}
	want := "<!doctype html>\n<meta charset=\"utf-8\">\n<body>\n" +
		"<ul>\n<li>parent<ol start=\"3\">\n<li><input type=\"checkbox\" disabled> child<ul>\n" +
		"<li><input type=\"checkbox\" disabled checked> grandchild</li>\n</ul>\n</li>\n</ol>\n" +
		"</li>\n<li>tail</li>\n</ul>\n</body>\n"
	if html != want {
		t.Fatalf("nested output=%q want=%q", html, want)
	}
	bounded, boundedWarnings, err := ExportHTMLBytes(doc, len(html))
	if err != nil || len(boundedWarnings) != 0 || string(bounded) != html || cap(bounded) != len(html) {
		t.Fatalf("nested two-pass export differs: len=%d cap=%d err=%v warnings=%#v", len(bounded), cap(bounded), err, boundedWarnings)
	}
	if _, _, err := ExportHTMLBytes(doc, len(html)-1); err != ErrOutputLimit {
		t.Fatalf("nested output budget accepted undersized result: %v", err)
	}
}

func TestListColumnsPreserveRealNestingReturnsTasksAndOrderedStarts(t *testing.T) {
	source := "- root\n    123. [ ] child\n          - grand\n  999. [x] peer\n- tail\n"
	doc := Parse(source)
	if doc.Partial || doc.Raw() != source || len(doc.Blocks) != 1 || doc.Blocks[0].Kind != BlockList {
		t.Fatalf("list parse=%#v", doc)
	}
	items := doc.Blocks[0].Items
	wantDepth := []int{0, 1, 2, 1, 0}
	wantOrdered := []bool{false, true, false, true, false}
	wantStarts := []int{0, 123, 0, 999, 0}
	wantRaw := []string{"- root\n", "    123. [ ] child\n", "          - grand\n", "  999. [x] peer\n", "- tail\n"}
	if len(items) != len(wantDepth) {
		t.Fatalf("item count=%d want=%d items=%#v", len(items), len(wantDepth), items)
	}
	for index, item := range items {
		if item.Depth != wantDepth[index] || item.Ordered != wantOrdered[index] || item.Start != wantStarts[index] ||
			source[item.Range.Start:item.Range.End] != wantRaw[index] {
			t.Fatalf("item %d=%#v raw=%q want depth=%d ordered=%v start=%d raw=%q", index, item,
				source[item.Range.Start:item.Range.End], wantDepth[index], wantOrdered[index], wantStarts[index], wantRaw[index])
		}
	}
	if child := items[1]; child.Task == nil || source[child.Task.Range.Start:child.Task.Range.End] != "[ ]" ||
		source[child.TextRange.Start:child.TextRange.End] != "child" {
		t.Fatalf("child task ranges=%#v", child)
	}
	if peer := items[3]; peer.Task == nil || !peer.Task.Checked || source[peer.Task.Range.Start:peer.Task.Range.End] != "[x]" {
		t.Fatalf("peer task ranges=%#v", peer)
	}

	tabSource := "- root\n\t+ tab child\n- tail\n"
	tabDoc := Parse(tabSource)
	if tabDoc.Partial || len(tabDoc.Blocks) != 1 || len(tabDoc.Blocks[0].Items) != 3 {
		t.Fatalf("tab list parse=%#v", tabDoc)
	}
	if got := []int{tabDoc.Blocks[0].Items[0].Depth, tabDoc.Blocks[0].Items[1].Depth, tabDoc.Blocks[0].Items[2].Depth};
		!reflect.DeepEqual(got, []int{0, 1, 0}) {
		t.Fatalf("tab depths=%v", got)
	}
}

func TestNestedSameDepthTypeSwitchStaysInsideParentListItem(t *testing.T) {
	source := "- parent\n  2. ordered\n  - bullet\n- tail\n"
	doc := Parse(source)
	if doc.Partial || len(doc.Blocks) != 1 || len(doc.Blocks[0].Items) != 4 {
		t.Fatalf("same-depth nested type switch parse=%#v", doc)
	}
	items := doc.Blocks[0].Items
	wantDepth := []int{0, 1, 1, 0}
	wantOrdered := []bool{false, true, false, false}
	for index := range items {
		if items[index].Depth != wantDepth[index] || items[index].Ordered != wantOrdered[index] {
			t.Fatalf("item %d=%#v", index, items[index])
		}
	}
	html, warnings, err := ExportHTMLChecked(doc)
	if err != nil || len(warnings) != 0 {
		t.Fatalf("same-depth switch export err=%v warnings=%#v", err, warnings)
	}
	want := "<!doctype html>\n<meta charset=\"utf-8\">\n<body>\n" +
		"<ul>\n<li>parent<ol start=\"2\">\n<li>ordered</li>\n</ol>\n" +
		"<ul>\n<li>bullet</li>\n</ul>\n</li>\n<li>tail</li>\n</ul>\n</body>\n"
	if html != want {
		t.Fatalf("same-depth type switch HTML=%q want=%q", html, want)
	}
}

func TestUnsupportedListContinuationAndInnerFenceWarnAndStayVisible(t *testing.T) {
	for _, source := range []string{
		"- item\n  continuation text\n",
		"- item\n  ```text\n  - visible source\n  ```\n",
	} {
		t.Run(strings.ReplaceAll(source, "\n", "/"), func(t *testing.T) {
			doc := Parse(source)
			if doc.Partial || doc.Raw() != source || len(doc.Blocks) < 2 {
				t.Fatalf("unsupported continuation did not remain complete and visible: %#v", doc)
			}
			warned := false
			for _, warning := range doc.Warnings {
				if warning.Code == "unsupported-list-continuation" {
					warned = true
					if source[warning.Range.Start:warning.Range.End] == "" {
						t.Fatal("continuation warning has no source range")
					}
				}
			}
			if !warned {
				t.Fatalf("unsupported continuation was silently accepted: %#v", doc.Warnings)
			}
			html, _, err := ExportHTMLChecked(doc)
			if err != nil || !strings.Contains(html, "continuation text") && !strings.Contains(html, "visible source") {
				t.Fatalf("continuation source not visible in export: err=%v html=%s", err, html)
			}
		})
	}
}

func TestNestedListDepthLimitPreservesRemainingSource(t *testing.T) {
	var source strings.Builder
	source.WriteString("- root\n")
	for depth := 1; depth <= maxListDepth+1; depth++ {
		source.WriteString(strings.Repeat(" ", depth*2))
		source.WriteString("- item\n")
	}
	input := source.String()
	doc := Parse(input)
	if !doc.Partial || doc.Raw() != input || len(doc.Warnings) == 0 ||
		doc.Warnings[len(doc.Warnings)-1].Code != "parse-limit" {
		t.Fatalf("depth limit did not preserve source and warn: partial=%v warnings=%#v", doc.Partial, doc.Warnings)
	}
}
