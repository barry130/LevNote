package editor

import (
	"errors"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func TestMultiPendingRejectsFoldOperationsAndKeepsFocus(t *testing.T) {
	const source = "{\nhidden\n}\nend"
	b := text.New(source)
	e := New(b)
	calls := 0
	provider := func(snapshot FoldSnapshot) (FoldResult, error) {
		calls++
		return FoldResult{Complete: true, Folds: braceFolds(snapshot.Source)}, nil
	}
	if err := e.SetFoldProvider(provider, 1, MaxFoldBytes); err != nil {
		t.Fatal(err)
	}
	if err := e.ToggleFoldAt(0); err != nil {
		t.Fatal(err)
	}
	tester := multiTester(t, e)
	start := setSelections(t, b, []text.Range{{Anchor: 0, Head: 0}, {Anchor: 12, Head: 11}}, 1)
	if !e.FoldCandidates()[0].Collapsed {
		t.Fatal("fixture lost collapsed projection before preedit")
	}
	tester.Compose("e", 1) // same bytes preserve the old fold identity and hidden lines
	candidates := e.FoldCandidates()
	if len(candidates) != 1 || !candidates[0].Collapsed || !tester.Focused(e.Label) {
		t.Fatal("same-text primary preedit did not preserve the folded block or focus")
	}
	before, folds := currentEditState(b), e.folds
	projection, scroll := e.folds.projection, e.ScrollY
	for _, op := range []struct {
		name string
		run  func() error
	}{
		{"enable", func() error { return e.SetFoldProvider(provider, 2, MaxFoldBytes) }},
		{"disable", func() error { return e.SetFoldProvider(nil, 2, 0) }},
		{"refresh", e.RefreshFolds},
		{"toggle-target", func() error { return e.ToggleFold(candidates[0].Target) }},
		{"toggle-line", func() error { return e.ToggleFoldAt(0) }},
		{"fold-position", func() error { return e.FoldAtPosition(0) }},
		{"unfold-position", func() error { return e.UnfoldAtPosition(0) }},
		{"unfold-all", e.UnfoldAll},
	} {
		t.Run(op.name, func(t *testing.T) {
			if err := op.run(); !errors.Is(err, text.ErrCompositionPending) {
				t.Fatalf("fold operation = %v", err)
			}
			if e.folds != folds || calls != 1 || folds.projection != projection || e.ScrollY != scroll || !reflect.DeepEqual(e.FoldCandidates(), candidates) {
				t.Fatal("fold operation changed provider, projection or scrolling")
			}
			requireEditState(t, b, before)
		})
	}
	var reported error
	e.OnError = func(err error) { reported = err }
	button, ok := tester.Find("展开全部")
	if !ok {
		t.Fatal("fold toolbar is missing during composition")
	}
	tester.ClickAt(button.X+button.W/2, button.Y+button.H/2)
	if !tester.Focused(e.Label) || !e.FoldCandidates()[0].Collapsed {
		t.Fatal("fold button changed focus or projection while pending")
	}
	box, ok := tester.Find(e.Label)
	if !ok {
		t.Fatal("editor box is missing")
	}
	tester.ClickAt(box.X+8, box.Y+12+e.rowHeight()/2)
	if !errors.Is(reported, text.ErrCompositionPending) || !e.FoldCandidates()[0].Collapsed {
		t.Fatalf("fold arrow was not refused: %v", reported)
	}
	requireEditState(t, b, before)
	tester.Key(0, ui.KeyEscape)
	requireSelections(t, b, start)
	if b.InGroup() || b.String() != source || b.CanUndo() || e.Flush() != nil {
		t.Fatal("Escape did not restore the folded document's start set")
	}
	if err := e.UnfoldAll(); err != nil || e.FoldCandidates()[0].Collapsed {
		t.Fatalf("settled editor could not unfold: %v", err)
	}
}
