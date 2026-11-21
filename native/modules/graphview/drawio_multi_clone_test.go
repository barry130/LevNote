package graphview

import (
	"errors"
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

func TestDrawioMultiClonePostFlushBudgetRefusalRetainsFieldUndo(t *testing.T) {
	const fieldBytes = 256
	before := drawioCloneBudgetFixture(t, fieldBytes/2)
	v, buffer, changed, label := prepareDrawioCloneBudgetCase(t, before, fieldBytes)
	// The normal 32 MiB history budget must hold only the changed label span,
	// not another pair of full-document snapshots near the 32 MiB format cap.
	var reports []error
	v.host.Report = func(err error) { reports = append(reports, err) }
	version, selection := buffer.Version(), buffer.Selection()
	if !v.cloneDrawioCells() {
		t.Fatal("clone action was not consumed")
	}

	postField := buffer.String()
	if postField == before || *changed != 1 || buffer.Version() != version+1 || buffer.InGroup() || buffer.History().UndoDepth != 1 || buffer.History().Truncated {
		t.Fatalf("ordinary field must commit as its own transaction before clone refusal: changed=%d version=%d history=%+v group=%t", *changed, buffer.Version(), buffer.History(), buffer.InGroup())
	}
	updated, _, err := drawio.Parse([]byte(postField), drawio.ParseOptions{})
	if err != nil || len(updated.Pages[0].Model.Nodes) != 3 || len(updated.Pages[0].Model.Edges) != 1 || nodeByID(updated.Pages[0].Model.Nodes, "a").Text != label {
		t.Fatalf("field should be written but no clone should be committed: nodes/edges=%d/%d err=%v", len(updated.Pages[0].Model.Nodes), len(updated.Pages[0].Model.Edges), err)
	}
	if len(reports) != 1 || !errors.Is(reports[0], drawio.ErrLimitExceeded) {
		t.Fatalf("second clone budget check did not report the actual limit failure: %v", reports)
	}
	warning := strings.Join(v.Warnings(), "\n")
	if !strings.Contains(warning, "普通字段已写回") || !strings.Contains(warning, "克隆未完成") || strings.Contains(warning, "原文保持不变") {
		t.Fatalf("refusal must disclose that ordinary field committed and clone did not: %q", warning)
	}

	if !v.undo() {
		t.Fatal("ordinary field undo was unavailable")
	}
	if buffer.String() != before || buffer.Selection() != selection || *changed != 2 || buffer.History().UndoDepth != 0 || buffer.History().RedoDepth != 1 || buffer.InGroup() {
		t.Fatalf("field commit must remain independently undoable: changed=%d history=%+v group=%t", *changed, buffer.History(), buffer.InGroup())
	}
}

func TestDrawioMultiClonePreflightBudgetRefusalDoesNotCommitField(t *testing.T) {
	before := drawioCloneBudgetFixture(t, -1)
	v, buffer, changed, label := prepareDrawioCloneBudgetCase(t, before, 256)
	var reports []error
	v.host.Report = func(err error) { reports = append(reports, err) }
	version, selection := buffer.Version(), buffer.Selection()
	history := buffer.History()
	if !v.cloneDrawioCells() {
		t.Fatal("clone action was not consumed")
	}
	if buffer.String() != before || buffer.Version() != version || buffer.Selection() != selection || *changed != 0 ||
		buffer.History() != history || buffer.InGroup() || !v.pendingDraft() || v.draft.input.Buffer.String() != label {
		t.Fatalf("preflight refusal must leave source, history, and pending field untouched: version=%d changed=%d history=%+v pending=%t", buffer.Version(), *changed, buffer.History(), v.pendingDraft())
	}
	if len(reports) != 1 || !errors.Is(reports[0], drawio.ErrLimitExceeded) {
		t.Fatalf("preflight refusal did not report the clone byte limit: %v", reports)
	}
	warning := strings.Join(v.Warnings(), "\n")
	if !strings.Contains(warning, "原文保持不变") || strings.Contains(warning, "普通字段已写回") {
		t.Fatalf("preflight wording must match the unchanged source stage: %q", warning)
	}
}

func prepareDrawioCloneBudgetCase(t *testing.T, before string, fieldBytes int) (*view, *text.Buffer, *int, string) {
	t.Helper()
	v, _, buffer, changed := newMultiDrawio(t, before, 1180, 780)
	if v.draft == nil || v.draft.field != fieldDrawioLabel || v.draft.nodeID != "a" {
		t.Fatalf("expected editable initial label draft, got %#v", v.draft)
	}
	label := strings.Repeat("Q", fieldBytes)
	if err := v.draft.input.Buffer.ReplaceUTF16(0, v.draft.input.Buffer.LenUTF16(), label); err != nil {
		t.Fatal(err)
	}
	// Preserve the pending draft as the clone action's ordinary-field flush
	// while establishing the exact ordered multi-selection it targets.
	v.setDrawioSelection([]string{"a", "b"}, "b")
	return v, buffer, changed, label
}

func drawioCloneBudgetFixture(t *testing.T, headroom int) string {
	t.Helper()
	baseClone, _, err := drawioClonedSource(multiDrawio, 0, "a", "b")
	if err != nil {
		t.Fatal("control clone must fit before budgeting fixture", err)
	}
	delta := len(baseClone) - len(multiDrawio)
	if delta <= 256 {
		t.Fatalf("clone fixture delta is too small for field growth: %d", delta)
	}
	fillerBytes := drawio.DefaultMaxBytes - len(multiDrawio) - delta - headroom - len("<!---->")
	if fillerBytes <= 0 {
		t.Fatalf("fixture cannot approach the format limit: %d", fillerBytes)
	}
	filler := "<!--" + strings.Repeat("x", fillerBytes) + "-->"
	before := strings.Replace(multiDrawio, "</root>", filler+"</root>", 1)
	clone, _, err := drawioClonedSource(before, 0, "a", "b")
	if headroom < 0 {
		if !errors.Is(err, drawio.ErrLimitExceeded) || len(before) >= drawio.DefaultMaxBytes {
			t.Fatalf("preflight fixture must be a valid source whose clone is one byte over: source=%d output=%d err=%v", len(before), len(clone), err)
		}
	} else if err != nil || len(clone) > drawio.DefaultMaxBytes || drawio.DefaultMaxBytes-len(clone) != headroom {
		t.Fatalf("pre-field clone must fit exact chosen headroom: source=%d output=%d headroom=%d err=%v", len(before), len(clone), headroom, err)
	}
	return before
}
