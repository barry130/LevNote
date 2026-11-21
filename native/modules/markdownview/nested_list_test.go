package markdownview

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

func TestNestedTaskControlsToggleExactMarkerAndUndo(t *testing.T) {
	source := "前文🙂\r\n\r\n- [ ] parent\r\n  3. [ ] child🙂\r\n     - [X] grandchild\r\n- tail\r\n"
	buffer := text.New(source)
	selection := text.Range{Anchor: buffer.LenUTF16() - 2, Head: 2}
	if err := buffer.SetSelection(selection.Anchor, selection.Head); err != nil {
		t.Fatal(err)
	}
	changes := 0
	v := Adapter().Create(editor.Host{Buffer: buffer, Changed: func() { changes++ }}).(*View)
	tester := ui.NewTester(v.Draw, 720, 480)
	parent := taskAccess(t, tester, "未完成任务：parent")
	child := taskAccess(t, tester, "未完成任务：child🙂")
	grandchild := taskAccess(t, tester, "已完成任务：grandchild")
	if child.Bounds.X <= parent.Bounds.X || grandchild.Bounds.X <= parent.Bounds.X {
		t.Fatalf("nested task indentation parent=%+v child=%+v grandchild=%+v", parent.Bounds, child.Bounds, grandchild.Bounds)
	}
	tester.ClickAt(child.Bounds.X+child.Bounds.W/2, child.Bounds.Y+child.Bounds.H/2)
	want := strings.Replace(source, "3. [ ] child🙂", "3. [x] child🙂", 1)
	if buffer.String() != want || buffer.Selection() != selection || changes != 1 ||
		buffer.History().UndoDepth != 1 || v.draft != nil {
		t.Fatalf("nested click changed unexpected bytes/state: source=%q selection=%v changes=%d", buffer.String(), buffer.Selection(), changes)
	}
	child = taskAccess(t, tester, "已完成任务：child🙂")
	if !child.Focused {
		t.Fatal("nested checkbox lost native focus")
	}
	tester.Key(0, ui.KeySpace)
	if buffer.String() != source || buffer.Selection() != selection || changes != 2 ||
		buffer.History().UndoDepth != 2 {
		t.Fatalf("nested Space did not toggle marker back: %q", buffer.String())
	}
	if !buffer.Undo() || buffer.String() != want || !buffer.Undo() || buffer.String() != source {
		t.Fatal("nested task edits did not remain separately undoable")
	}
}

func TestListPreviewNumberingFollowsParentAndTypeRuns(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		want         []int
	}{
		{
			name:   "nested skipped source markers",
			source: "- root\n  3. child\n  9. peer\n",
			want:   []int{3, 4},
		},
		{
			name:   "parent resumes after descendants",
			source: "3. parent\n   - child\n9. peer\n   6. branch\n   1. branch-peer\n1. last\n",
			want:   []int{3, 4, 6, 7, 5},
		},
		{
			name:   "type switches reopen runs and new parent resets children",
			source: "- root\n  3. first\n  9. second\n  - bullet\n    8. leaf\n    1. leaf-peer\n  7. restarted\n  1. next\n- parent-two\n  11. fresh\n  2. fresh-peer\n",
			want:   []int{3, 4, 8, 9, 7, 8, 11, 12},
		},
		{
			name:   "deep independent runs resume ancestors",
			source: "- root\n  3. a\n     8. deep-a\n        12. leaf-a\n        99. leaf-b\n     1. deep-b\n        6. new-leaf\n        9. new-peer\n  9. b\n     5. fresh-deep\n     1. fresh-peer\n",
			want:   []int{3, 8, 12, 13, 9, 6, 7, 4, 5, 6},
		},
		{
			name:   "multi digit starts use real content columns",
			source: "123. parent\n     98. child\n         1. deep\n         9. peer\n     7. child-peer\n999. root-peer\n     12. fresh\n     90. fresh-peer\n",
			want:   []int{123, 98, 1, 2, 99, 124, 12, 13},
		},
		{
			name:   "zero is a valid ordered start",
			source: "0. root\n   9. child\n   4. next\n7. peer\n",
			want:   []int{0, 9, 10, 1},
		},
		{
			name:   "top level type and block boundaries",
			source: "3. one\n9. two\n- bullet\n8. fresh\n1. peer\n\n5. another\n1. next\n",
			want:   []int{3, 4, 8, 9, 5, 6},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buffer := text.New(tc.source)
			selection := text.Range{Anchor: buffer.LenUTF16(), Head: 1}
			if err := buffer.SetSelection(selection.Anchor, selection.Head); err != nil {
				t.Fatal(err)
			}
			version, history := buffer.Version(), buffer.History()
			v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
			tester := ui.NewTester(v.Draw, 720, 1200)
			if v.doc.Partial || len(v.doc.Warnings) != 0 {
				t.Fatalf("invalid list fixture: partial=%v warnings=%v", v.doc.Partial, v.doc.Warnings)
			}
			var got []int
			for _, label := range tester.Texts() {
				if number, err := strconv.Atoi(strings.TrimSuffix(label, ".")); err == nil && strings.HasSuffix(label, ".") {
					got = append(got, number)
				}
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("rendered ordered prefixes = %v, want %v; source=%q", got, tc.want, tc.source)
			}
			ids := make([]uint64, len(v.rows))
			for index, row := range v.rows {
				ids[index] = row.id
				item := v.doc.Blocks[row.block].Items[row.item]
				wantSpan := text.Range{
					Anchor: text.UTF16Len(tc.source[:item.Range.Start]),
					Head:   text.UTF16Len(tc.source[:item.Range.End]),
				}
				if row.span != wantSpan {
					t.Errorf("item %d span = %v, want %v", index, row.span, wantSpan)
				}
			}
			// Reprojection must retain editing/focus identity and parsed markers.
			beforeItems := fmt.Sprint(v.doc.Blocks)
			v.setPreview(v.doc)
			tester.Frame()
			for index, row := range v.rows {
				if row.id != ids[index] {
					t.Errorf("unchanged row %d identity changed", index)
				}
			}
			if fmt.Sprint(v.doc.Blocks) != beforeItems || buffer.String() != tc.source || buffer.Selection() != selection ||
				buffer.Version() != version || buffer.History() != history {
				t.Fatal("numbering changed parser markers, source, selection or history")
			}
		})
	}
}

func TestListPreviewNumberingLargestStartContinuesBeyondMarkerWidth(t *testing.T) {
	const source = "- root\n  999999999. child\n  1. peer\n"
	buffer := text.New(source)
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 720, 480)
	if v.doc.Partial || len(v.doc.Warnings) != 0 ||
		!slices.Contains(tester.Texts(), "999999999.") || !slices.Contains(tester.Texts(), "1000000000.") {
		t.Fatalf("largest supported source start wrapped or reset: %v", tester.Texts())
	}
	if buffer.String() != source || v.doc.Blocks[0].Items[1].Start != 999999999 || v.doc.Blocks[0].Items[2].Start != 1 {
		t.Fatal("logical numbering rewrote literal markers")
	}
}

func TestListPreviewNumberingVirtualScrollPreservesCheckboxIdentity(t *testing.T) {
	var fixture strings.Builder
	fixture.WriteString("# 导言🙂\r\n\r\n")
	for i := 0; i < 600; i++ {
		fmt.Fprintf(&fixture, "10. parent %04d\r\n    - branch %04d\r\n      3. [ ] child %04d\r\n      9. [ ] peer %04d\r\n", i, i, i, i)
	}
	source := fixture.String()
	buffer := text.New(source)
	selection := text.Range{Anchor: buffer.LenUTF16() - 2, Head: 3}
	if err := buffer.SetSelection(selection.Anchor, selection.Head); err != nil {
		t.Fatal(err)
	}
	changes := 0
	v := Adapter().Create(editor.Host{Buffer: buffer, Changed: func() { changes++ }}).(*View)
	tester := ui.NewTester(v.Draw, 365, 480)
	if v.doc.Partial || len(v.doc.Warnings) != 0 || len(v.rows) != 1+4*600 {
		t.Fatalf("virtual list fixture incomplete: partial=%v warnings=%v rows=%d", v.doc.Partial, v.doc.Warnings, len(v.rows))
	}
	const group, firstRow = 525, 1 + 4*525
	v.list.ScrollTo(firstRow, ui.Start)
	tester.Frame()
	checkPrefixes := func() {
		t.Helper()
		want := map[string]string{"parent 0525": "535.", "child 0525": "3.", "peer 0525": "4."}
		prefix := ""
		for _, label := range tester.Texts() {
			if _, err := strconv.Atoi(strings.TrimSuffix(label, ".")); err == nil && strings.HasSuffix(label, ".") {
				prefix = label
			}
			if expected, ok := want[label]; ok {
				if prefix != expected {
					t.Errorf("visible %q prefix = %q, want %q", label, prefix, expected)
				}
				delete(want, label)
			}
		}
		if len(want) != 0 || v.frameRows == 0 || v.frameRows > 60 || tester.FramePending() {
			t.Fatalf("far rows missing/unbounded/unsettled: missing=%v built=%d pending=%v", want, v.frameRows, tester.FramePending())
		}
	}
	checkPrefixes()
	info := taskAccess(t, tester, fmt.Sprintf("未完成任务：peer %04d", group))
	id := v.rows[firstRow+3].id
	tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	want := strings.Replace(source, "9. [ ] peer 0525", "9. [x] peer 0525", 1)
	if buffer.String() != want || buffer.Selection() != selection || changes != 1 || buffer.History().UndoDepth != 1 ||
		v.rows[firstRow+3].id != id || v.draft != nil {
		t.Fatal("far numbered checkbox changed wrong marker, selection, history or identity")
	}
	if !taskAccess(t, tester, "已完成任务：peer 0525").Focused {
		t.Fatal("far numbered checkbox lost focus")
	}
	checkPrefixes()
	tester.Key(0, ui.KeySpace)
	if buffer.String() != source || buffer.Selection() != selection || changes != 2 || buffer.History().UndoDepth != 2 ||
		v.rows[firstRow+3].id != id {
		t.Fatal("far numbered checkbox Space lost exact source/selection/identity")
	}
	// Exercise native wheel scrolling, then return to the same virtual rows.
	tester.Scroll(200, 300, 0, 240)
	v.list.ScrollTo(firstRow, ui.Start)
	tester.Frame()
	checkPrefixes()
	if v.rows[firstRow+3].id != id || !buffer.Undo() || buffer.String() != want ||
		!buffer.Undo() || buffer.String() != source || !buffer.Redo() || buffer.String() != want ||
		!buffer.Redo() || buffer.String() != source || buffer.Selection() != selection {
		t.Fatal("numbered task scroll/undo/redo did not preserve exact edits and selection")
	}
	t.Logf("projected rows=%d far first row=%d built rows=%d pending frame=%v", len(v.rows), firstRow, v.frameRows, tester.FramePending())
}
