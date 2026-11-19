package graphview

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/mindmap"
	"levnote.local/native/ui/editor"
)

const collapsedOutlineMindmap = `{"format":"levnote.mindmap","version":1,"layout":"logicalStructure","vendor":{"number":1.00},"root":{"id":"root","topic":"中心","collapsed":true,"children":[{"id":"branch","topic":"分支","collapsed":true,"extraVendor":{"raw":[1,"二"]},"children":[{"id":"leaf","topic":"叶子🙂","children":[]}]}]}}`

func TestMindmapOutlineInspectionNoViewNoMutationIncludesFoldedNodes(t *testing.T) {
	buffer := text.New(collapsedOutlineMindmap)
	before, selection := buffer.Version(), buffer.Selection()
	source := MindmapAdapter().Outline(editor.Host{Buffer: buffer, ReadOnly: true, Changed: func() { t.Fatal("inspection Changed") }}).(*mindmapOutlineSource)
	snapshot, err := source.OutlineSnapshot()
	if err != nil || snapshot.Truncated || len(snapshot.Items) != 3 || snapshot.Version != before {
		t.Fatal(snapshot, err)
	}
	for index, id := range []string{"root", "branch", "leaf"} {
		item := snapshot.Items[index]
		if item.ID != id || item.Level != index+1 || item.Offset != -1 || item.AnchorID != "" {
			t.Fatal("folded preorder/source coordinates wrong", item)
		}
	}
	if buffer.Version() != before || buffer.Selection() != selection || buffer.String() != collapsedOutlineMindmap || buffer.CanUndo() {
		t.Fatal("observational source changed document")
	}
	snapshot.Items[0].Text = "caller-owned"
	again, _ := source.OutlineSnapshot()
	if again.Items[0].Text == "caller-owned" {
		t.Fatal("cached descriptor slice leaked")
	}
	if _, err := source.ResolveOutlineAnchor("root"); !errors.Is(err, editor.ErrOutlineUnsupported) {
		t.Fatal("mindmap invented anchor", err)
	}
	source.Release()
	if source.items != nil || source.loaded || source.cachedBuffer != nil {
		t.Fatal("source release retained cache")
	}
	if item, err := source.ResolveOutlineItem("leaf"); err != nil || item.Offset != -1 {
		t.Fatal("release broke stable file identity", err)
	}
	if DrawioAdapter().Outline != nil {
		t.Fatal("out-of-scope format gained invented outline")
	}
}

func TestMindmapOutlineHiddenNavigationSingleUndoPreservesMetadataAndZoom(t *testing.T) {
	buffer := text.New(collapsedOutlineMindmap)
	v := newView(kindMindmap, editor.Host{Buffer: buffer})
	v.zoom = 0.73
	// Navigate before the first Draw: centering is deferred, zoom is retained.
	if err := v.NavigateOutlineItem("leaf"); err != nil {
		t.Fatal(err)
	}
	if v.selectedID != "leaf" || buffer.History().UndoDepth != 1 {
		t.Fatal("hidden navigation did not create one undo group", buffer.History())
	}
	doc := parseMind(t, buffer)
	branch, _ := doc.FindNode("branch")
	if *doc.Root.Collapsed || *branch.Collapsed || !strings.Contains(buffer.String(), `"number":1.00`) || !strings.Contains(buffer.String(), `"raw":[1,"二"]`) {
		t.Fatal("collapsed flags/unknown metadata corrupted", buffer.String())
	}
	tester := ui.NewTester(v.Draw, 960, 640)
	if v.zoom != 0.73 || v.pendingOutlineCenter != "" || len(v.layout.Nodes) != 3 {
		t.Fatal("first frame discarded navigation preferences")
	}
	for _, node := range v.layout.Nodes {
		if node.ID == "leaf" {
			x, y := v.worldToScreen(float32(node.X+node.Width/2), float32(node.Y+node.Height/2))
			if abs32(x-v.canvasRect.W/2) > 0.1 || abs32(y-v.canvasRect.H/2) > 0.1 {
				t.Fatal("target not centered", x, y)
			}
		}
	}
	if !buffer.Undo() || buffer.String() != collapsedOutlineMindmap {
		t.Fatal("one undo did not restore byte-exact folded metadata")
	}
	tester.Frame()
	if len(v.layout.Nodes) != 1 {
		t.Fatal("undo left expanded layout")
	}
	if err := v.NavigateOutlineItem("root"); err != nil || buffer.CanUndo() {
		t.Fatal("selecting folded root expanded document", err)
	}
}

func TestMindmapOutlineReadonlyHiddenFailureIsObservational(t *testing.T) {
	buffer := text.New(collapsedOutlineMindmap)
	v := newView(kindMindmap, editor.Host{Buffer: buffer, ReadOnly: true})
	v.zoom, v.panX, v.panY = 1.2, 37, 19
	if err := v.NavigateOutlineItem("leaf"); !errors.Is(err, editor.ErrOutlineReadOnlyHidden) {
		t.Fatal(err)
	}
	if v.mind != nil || len(v.layout.Nodes) != 0 || v.selectedID != "" || v.zoom != 1.2 || v.panX != 37 || v.panY != 19 || buffer.String() != collapsedOutlineMindmap || buffer.CanUndo() {
		t.Fatal("readonly failure changed selection/layout/preferences/source")
	}
	if err := v.NavigateOutlineItem("root"); err != nil || v.selectedID != "root" || buffer.String() != collapsedOutlineMindmap {
		t.Fatal("visible readonly root cannot navigate", err)
	}
}

func TestMindmapOutlineRootSelectionRevealAndUndoAcrossLayouts(t *testing.T) {
	for _, layout := range []string{mindmap.LayoutBoth, mindmap.LayoutLeft, mindmap.LayoutRight, mindmap.LayoutOrg} {
		t.Run(layout, func(t *testing.T) {
			original := strings.Replace(collapsedOutlineMindmap, mindmap.LayoutBoth, layout, 1)
			buffer := text.New(original)
			v := newView(kindMindmap, editor.Host{Buffer: buffer})
			if err := v.NavigateOutlineItem("root"); err != nil || len(v.layout.Nodes) != 1 || buffer.CanUndo() || buffer.String() != original {
				t.Fatal("root navigation expanded folded content", err)
			}
			if err := v.NavigateOutlineItem("leaf"); err != nil || len(v.layout.Nodes) != 3 || buffer.History().UndoDepth != 1 {
				t.Fatal("descendant not revealed with one undo transaction", err)
			}
			if !buffer.Undo() || buffer.String() != original {
				t.Fatal("undo did not restore complete original metadata")
			}
			v.ensureSynced(nil)
			if len(v.layout.Nodes) != 1 || v.layout.Nodes[0].ID != "root" {
				t.Fatal("undo did not restore folded visible set")
			}
		})
	}
}

func TestMindmapOutlineExpansionPreservesSourceSelectionAndHistory(t *testing.T) {
	for _, selection := range []text.Range{{Anchor: 5, Head: 17}, {Anchor: 17, Head: 5}} {
		t.Run(fmt.Sprint(selection), func(t *testing.T) {
			buffer := text.New(collapsedOutlineMindmap)
			if err := buffer.SetSelection(selection.Anchor, selection.Head); err != nil {
				t.Fatal(err)
			}
			v := newView(kindMindmap, editor.Host{Buffer: buffer})
			if err := v.NavigateOutlineItem("leaf"); err != nil || buffer.Selection() != selection {
				t.Fatal("navigation moved source selection", err, buffer.Selection())
			}
			if !buffer.Undo() || buffer.Selection() != selection || buffer.String() != collapsedOutlineMindmap {
				t.Fatal("undo did not restore source selection/text", buffer.Selection())
			}
			if !buffer.Redo() || buffer.Selection() != selection {
				t.Fatal("redo moved source selection", buffer.Selection())
			}
		})
	}
}

func TestMindmapOutlineExpansionClampsShorterSerializationAndSurrogate(t *testing.T) {
	// Make original end coordinates exceed the canonical output after expansion.
	original := strings.Repeat(" ", 1200) + collapsedOutlineMindmap
	buffer := text.New(original)
	before := text.Range{Anchor: buffer.LenUTF16(), Head: buffer.LenUTF16() - 2}
	_ = buffer.SetSelection(before.Anchor, before.Head)
	v := newView(kindMindmap, editor.Host{Buffer: buffer})
	if err := v.NavigateOutlineItem("leaf"); err != nil {
		t.Fatal(err)
	}
	after := buffer.Selection()
	if after.Anchor != buffer.LenUTF16() || after.Head != buffer.LenUTF16() || !strings.Contains(strings.Join(v.Warnings(), " "), "夹取") {
		t.Fatal("shorter serialization lacked safe/disclosed clamping", after)
	}
	if !buffer.Undo() || buffer.Selection() != before || buffer.String() != original || !buffer.Redo() || buffer.Selection() != after {
		t.Fatal("clamped selection not part of same undo transaction")
	}

	// Find the output emoji's middle UTF-16 unit, then use the same numerical
	// coordinate in a whitespace-prefixed input where it is a legal boundary.
	canonical := buffer.String()
	emojiByte := strings.Index(canonical, "🙂")
	middle := text.UTF16Len(canonical[:emojiByte]) + 1
	input := text.New(strings.Repeat(" ", middle+2) + collapsedOutlineMindmap)
	if err := input.SetSelection(middle, middle); err != nil {
		t.Fatal("fixture pre-selection must be a legal boundary", err)
	}
	v = newView(kindMindmap, editor.Host{Buffer: input})
	if err := v.NavigateOutlineItem("leaf"); err != nil {
		t.Fatal(err)
	}
	if input.Selection().Head != middle-1 || !strings.Contains(strings.Join(v.Warnings(), " "), "夹取") {
		t.Fatal("selection landed inside surrogate", input.Selection(), middle)
	}
	if !input.Undo() || input.Selection().Head != middle || !input.Redo() || input.Selection().Head != middle-1 {
		t.Fatal("surrogate-clamped selection history inconsistent")
	}
}

func TestMindmapOutlineNativeComposeThenNavigateAndConflict(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		t.Run(fmt.Sprint(conflict), func(t *testing.T) {
			buffer := text.New(testMindmap)
			v := newView(kindMindmap, editor.Host{Buffer: buffer})
			tester := ui.NewTester(v.Draw, 960, 640)
			focusDraft(t, tester)
			tester.Compose("中文🙂", 3)
			draft, x, y := v.draft, v.panX, v.panY
			if conflict {
				replaceSource(t, buffer, `"topic": "中心"`, `"topic": "外部"`)
			}
			err := v.NavigateOutlineItem("left")
			if conflict {
				if !errors.Is(err, ErrDraftConflict) || v.draft != draft || !v.pendingDraft() || v.selectedID != "root" || v.panX != x || v.panY != y {
					t.Fatal("failed flush navigated or discarded draft", err)
				}
				return
			}
			if err != nil || parseMind(t, buffer).Root.Topic != "中文🙂" || v.selectedID != "left" || buffer.History().UndoDepth != 1 {
				t.Fatal("composition navigation lost pending text", err)
			}
			v.Release()
			if v.outline != nil || v.mind != nil || v.layout.Nodes != nil || v.selectedID != "left" {
				t.Fatal("release retained heavy state or lost lightweight selection")
			}
		})
	}
}

func largeOutlineMindmap(nodes int) string {
	var source strings.Builder
	source.WriteString(`{"format":"levnote.mindmap","version":1,"root":{"id":"root","topic":"中心","children":[`)
	for i := 1; i < nodes; i++ {
		if i > 1 {
			source.WriteByte(',')
		}
		fmt.Fprintf(&source, `{"id":"n%d","topic":"节点中文🙂 %d"}`, i, i)
	}
	source.WriteString(`]}}`)
	return source.String()
}

func TestMindmapOutlineCapsInvalidSourceAndBudget(t *testing.T) {
	buffer := text.New(largeOutlineMindmap(2200))
	source := newMindmapOutlineSource(editor.Host{Buffer: buffer}).(*mindmapOutlineSource)
	snapshot, err := source.OutlineSnapshot()
	if err != nil || len(snapshot.Items) != editor.MaxOutlineItems || !snapshot.Truncated {
		t.Fatal("count cap not disclosed", len(snapshot.Items), err)
	}
	source.buffer = text.New(strings.Replace(testMindmap, "中心", strings.Repeat("中🙂", 300), 1))
	snapshot, err = source.OutlineSnapshot()
	if err != nil || !snapshot.Truncated || len(snapshot.Items[0].Text) > editor.MaxOutlineLabelBytes || !utf8.ValidString(snapshot.Items[0].Text) {
		t.Fatal("label cap not disclosed", err)
	}
	source.buffer = text.New(strings.Replace(testMindmap, `"id": "root"`, `"id": "`+strings.Repeat("r", 513)+`"`, 1))
	snapshot, err = source.OutlineSnapshot()
	if err != nil || !snapshot.Truncated || len(snapshot.Items) != 2 || snapshot.Items[0].ID != "left" || snapshot.Items[0].Level != 2 {
		t.Fatal("huge identity clipped/aliased", snapshot, err)
	}
	source.buffer = text.New("invalid")
	if _, err := source.OutlineSnapshot(); err == nil {
		t.Fatal("parse failure presented as valid empty")
	}
	source.buffer = text.New(strings.Repeat("x", mindmap.DefaultMaxBytes+1))
	if _, err := source.OutlineSnapshot(); !errors.Is(err, editor.ErrOutlineUnsupported) || source.items != nil {
		t.Fatal("byte guard not applied before parse", err)
	}
}

func BenchmarkMindmapOutline(b *testing.B) {
	buffer := text.New(largeOutlineMindmap(2000))
	for _, cold := range []bool{true, false} {
		b.Run(fmt.Sprint("cold=", cold), func(b *testing.B) {
			source := MindmapAdapter().Outline(editor.Host{Buffer: buffer})
			source.OutlineSnapshot()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if cold {
					source.Release()
				}
				source.OutlineSnapshot()
			}
		})
	}
}
