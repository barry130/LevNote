package graphview

import (
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
)

func TestDrawioMultiCloneOneTransactionSelectionAndExactUndoRedo(t *testing.T) {
	v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
	selectMultiNodes(t, tester, v, "a", "b")
	if err := buffer.SetSelection(4, 9); err != nil { t.Fatal(err) }
	before, caret, version := buffer.String(), buffer.Selection(), buffer.Version()
	parsed := parseMultiDrawio(t, buffer)
	children := len(parsed.Pages[0].Model.Nodes)
	tester.Key(ui.Cmd, ui.KeyD)
	if *changed != 1 || buffer.Version() != version+1 {
		t.Fatalf("Ctrl+D must be one transaction changed=%d version=%d", *changed, buffer.Version())
	}
	after := buffer.String()
	if !slices.Equal(v.drawSelection.ids, []string{"levnote-1", "levnote-2"}) || v.selectedID != "levnote-2" {
		t.Fatalf("clone nodes not selected in ordered order: %#v", v.drawSelection)
	}
	doc := parseMultiDrawio(t, buffer)
	if len(doc.Pages[0].Model.Nodes) != children+2 || len(doc.Pages[0].Model.Edges) != 2 {
		t.Fatalf("incorrect vertex/internal-edge count: %d/%d", len(doc.Pages[0].Model.Nodes), len(doc.Pages[0].Model.Edges))
	}
	for _, id := range []string{"levnote-1", "levnote-2"} {
		if nodeByID(doc.Pages[0].Model.Nodes, id) == nil { t.Fatal(id, "absent") }
	}
	for _, original := range []string{`<future z='007'/>`, `<mxfile custom="keep-🙂">`, `width="80.0" height="40.00"`, "<!--unknown spelling-->"} {
		if !strings.Contains(after, original) { t.Fatal("untouched XML lost", original) }
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != before || buffer.Selection() != caret || *changed != 2 || !slices.Equal(v.drawSelection.ids, []string{"a", "b"}) {
		t.Fatal("undo must exactly restore original buffer/caret/multi-selection", v.drawSelection.ids)
	}
	for _, direction := range []ui.Key{ui.KeyY, ui.KeyZ, ui.KeyY} {
		tester.Key(ui.Cmd, direction)
	}
	if buffer.String() != after || *changed != 5 || !slices.Equal(v.drawSelection.ids, []string{"levnote-1", "levnote-2"}) {
		t.Fatal("repeated history transitions lost the exact cloned state", v.drawSelection.ids)
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	v.setDrawioSelection([]string{"c"}, "c")
	tester.Key(ui.Cmd, ui.KeyY)
	if buffer.String() != after || !slices.Equal(v.drawSelection.ids, []string{"c"}) {
		t.Fatal("an unrelated user selection was overwritten by history", v.drawSelection.ids)
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != before || !slices.Equal(v.drawSelection.ids, []string{"c"}) {
		t.Fatal("old clone selection revived after an explicit user choice", v.drawSelection.ids)
	}
}

func TestDrawioMultiCloneContinuousHistoryRespectsOwnerAndSelectionEpoch(t *testing.T) {
	v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
	selectMultiNodes(t, tester, v, "a", "b")
	caret := text.Range{Anchor: 4, Head: 9}
	if err := buffer.SetSelection(caret.Anchor, caret.Head); err != nil { t.Fatal(err) }
	before := buffer.String()
	tester.Key(ui.Cmd, ui.KeyD)
	first := buffer.String()
	firstIDs := slices.Clone(v.drawSelection.ids)
	if len(firstIDs) != 2 || v.selectedID != firstIDs[1] || buffer.String() == before || *changed != 1 || !strings.Contains(first, `id="`+firstIDs[1]+`"`) { t.Fatal("first clone must commit and select clones", firstIDs, v.Warnings()) }
	v.ensureSynced(nil)
	tester.Key(ui.Cmd, ui.KeyD)
	second := buffer.String()
	secondIDs := slices.Clone(v.drawSelection.ids)
	if len(secondIDs) != 2 || v.selectedID != secondIDs[1] || buffer.String() == first || *changed != 2 || !strings.Contains(second, `id="` + secondIDs[1] + `"`) { t.Fatal("second clone must commit and select clones", secondIDs, v.Warnings()) }
	tester.Key(ui.Cmd, ui.KeyZ)
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != before || !slices.Equal(v.drawSelection.ids, []string{"a", "b"}) {
		t.Fatal("two undos did not restore the original source and selection", v.drawSelection.ids)
	}
	tester.Key(ui.Cmd, ui.KeyY)
	if buffer.String() != first || buffer.Selection() != caret || !slices.Equal(v.drawSelection.ids, firstIDs) {
		t.Fatal("first redo must restore the first clone selection", v.drawSelection.ids)
	}
	tester.Key(ui.Cmd, ui.KeyY)
	if buffer.String() != second || !slices.Equal(v.drawSelection.ids, secondIDs) {
		t.Fatal("second redo must restore the second clone selection", v.drawSelection.ids)
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	tester.Key(ui.Cmd, ui.KeyZ)
	v.setDrawioSelection([]string{"c"}, "c")
	tester.Key(ui.Cmd, ui.KeyY)
	tester.Key(ui.Cmd, ui.KeyY)
	if buffer.String() != second || !slices.Equal(v.drawSelection.ids, []string{"c"}) || v.selectedID != "c" {
		t.Fatal("history overrode the user's explicit selection", v.drawSelection.ids)
	}
}

func TestDrawioCloneRejectsInvalidAndLifecycleChanges(t *testing.T) {
	for _, tc := range []struct {name string; mutate func(*view, *text.Buffer); readonly bool}{
		{"empty", func(v *view, _ *text.Buffer){ v.setDrawioSelection(nil, "") }, false},
		{"single", func(v *view, _ *text.Buffer){ v.setDrawioSelection([]string{"a"}, "a") }, false},
		{"missing", func(v *view, _ *text.Buffer){ v.setDrawioSelection([]string{"a", "missing"}, "missing") }, false},
		{"replaced-buffer", func(v *view, _ *text.Buffer){ v.host.Buffer = text.New(multiDrawio) }, false},
		{"readonly", func(v *view, _ *text.Buffer){ }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
			selectMultiNodes(t, tester, v, "a", "b")
			before, version, caret := buffer.String(), buffer.Version(), buffer.Selection()
			tc.mutate(v, buffer)
			v.host.ReadOnly = tc.readonly
			tester.Key(ui.Cmd, ui.KeyD)
			if buffer.String() != before || buffer.Version() != version || buffer.Selection() != caret || *changed != 0 {
				t.Fatal("invalid/replaced/readonly clone changed the original buffer")
			}
		})
	}
}

func nodeByID(nodes []drawio.Node, id string) *drawio.Node {
	for index := range nodes { if nodes[index].ID == id { return &nodes[index] } }
	return nil
}
