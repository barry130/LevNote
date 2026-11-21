package editor

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func requireLanguageReadRefusals(t *testing.T, e *Editor, target SourceTarget, want error, buffers ...*text.Buffer) {
	t.Helper()
	for _, action := range []func() error{
		func() error { _, err := e.CaptureSourceSnapshot(64); return err },
		func() error { return e.CheckSourceTarget(target, false) },
		func() error { return e.CheckSourceTarget(target, true) },
		func() error { return e.RevealSourceTarget(target, 0, 0) },
	} {
		requirePureSourceRefusal(t, e, action, want, buffers...)
	}
}

func TestLanguageSnapshotRejectsActiveNativeComposition(t *testing.T) {
	for _, multi := range []bool{false, true} {
		t.Run(map[bool]string{false: "single", true: "multi"}[multi], func(t *testing.T) {
			b := text.New("a b")
			e := New(b)
			tester := multiTester(t, e)
			ranges := []text.Range{{Anchor: 1, Head: 1}}
			if multi {
				ranges = append(ranges, text.Range{Anchor: 3, Head: 3})
			}
			start := setSelections(t, b, ranges, len(ranges)-1)
			target := captureLanguageTarget(t, e)
			tester.Compose("拼🙂", 2)
			token := e.compositionToken
			if _, marked := e.MarkedRange(); !marked || !b.OwnsComposition(token) {
				t.Fatal("accepted preedit did not retain a native marked owner")
			}
			changes, reports := 0, 0
			e.OnChange = func() { changes++ }
			e.OnError = func(error) { reports++ }
			want := text.ErrComposition
			if multi {
				want = text.ErrCompositionPending
			}
			requireLanguageReadRefusals(t, e, target, want, b)
			if changes != 0 || reports != 0 || !b.OwnsComposition(token) || !tester.Focused(e.Label) {
				t.Fatal("native refusal settled input, reported or changed focus")
			}
			tester.Key(0, ui.KeyEscape)
			if b.String() != "a b" || b.InGroup() || !reflect.DeepEqual(b.SelectionSet(), start) {
				t.Fatal("original native owner could not cancel after source refusal")
			}
		})
	}
}

func TestLanguageSnapshotBoundedAndReadonly(t *testing.T) {
	const source = "\ufefffirst\r\n🙂 second\nthird\r\n\ufefftail"
	b := text.NewWithLimit(source, 128)
	e := New(b)
	setSelections(t, b, []text.Range{{Anchor: 4, Head: 1}, {Anchor: 13, Head: 11}, {Anchor: 25, Head: 25}}, 1)
	if err := b.ReplaceUTF16(b.LenUTF16(), b.LenUTF16(), "!"); err != nil || !b.Undo() {
		t.Fatal("redo fixture", err)
	}
	e.SetReadOnly(true)
	changes, errorsSeen := 0, 0
	e.OnChange = func() { changes++ }
	e.OnError = func(error) { errorsSeen++ }

	beforeSource, beforeVersion := b.String(), b.Version()
	beforeSelections, beforeGeneration := b.SelectionSet(), b.SelectionGeneration()
	beforeHistory, beforeRedo := b.History(), b.CanRedo()
	beforeJournal := b.ChangedLineSince(beforeVersion)
	beforeOwner := e.inputOwner

	got, err := e.CaptureSourceSnapshot(len(source))
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != source || got.Version != beforeVersion || !reflect.DeepEqual(got.Selections, beforeSelections) {
		t.Fatalf("snapshot lost raw source or selection: %+v", got)
	}
	if e.inputOwner != beforeOwner || b.String() != beforeSource || b.Version() != beforeVersion ||
		!reflect.DeepEqual(b.SelectionSet(), beforeSelections) || b.SelectionGeneration() != beforeGeneration ||
		b.History() != beforeHistory || b.CanRedo() != beforeRedo || b.ChangedLineSince(beforeVersion) != beforeJournal ||
		changes != 0 || errorsSeen != 0 {
		t.Fatal("capture changed observable editor state")
	}

	// The returned selection and the token-owned selection must not share backing
	// storage with each other or with the live Buffer.
	got.Selections.Ranges[0] = text.Range{Anchor: 0, Head: 0}
	if len(got.Target.selections.Ranges) == 0 {
		t.Fatal("target lost its complete selection set")
	}
	if !reflect.DeepEqual(got.Target.selections, beforeSelections) || !reflect.DeepEqual(b.SelectionSet(), beforeSelections) {
		t.Fatal("snapshot selection mutation leaked into the token or Buffer")
	}
	if err := e.CheckSourceTarget(got.Target, true); err != nil {
		t.Fatalf("token-owned selection was not deep-copied: %v", err)
	}

	for _, maxBytes := range []int{-1, 0} {
		if _, err := e.CaptureSourceSnapshot(maxBytes); err == nil {
			t.Fatalf("invalid maxBytes=%d accepted", maxBytes)
		}
	}
	if _, err := e.CaptureSourceSnapshot(len(source) - 1); err == nil {
		t.Fatal("under-budget source accepted")
	}
}

func TestLanguageSnapshotRawUTF8AndHardCap(t *testing.T) {
	atCap := strings.Repeat("x", maxLanguageSourceBytes)
	b := text.NewWithLimit(atCap, maxLanguageSourceBytes+1)
	e := New(b)
	got, err := e.CaptureSourceSnapshot(maxLanguageSourceBytes)
	if err != nil || got.Source != atCap || len(got.Source) != maxLanguageSourceBytes {
		t.Fatalf("exact hard cap capture: len=%d err=%v", len(got.Source), err)
	}
	if _, err := e.CaptureSourceSnapshot(maxLanguageSourceBytes + 1); err != nil {
		t.Fatalf("max above hard cap should use the 2MiB cap: %v", err)
	}

	over := strings.Repeat("y", maxLanguageSourceBytes+1)
	overEditor := New(text.NewWithLimit(over, maxLanguageSourceBytes+2))
	for _, maxBytes := range []int{maxLanguageSourceBytes + 1, maxLanguageSourceBytes + 1024} {
		if _, err := overEditor.CaptureSourceSnapshot(maxBytes); err == nil {
			t.Fatalf("source above hard cap accepted for maxBytes=%d", maxBytes)
		}
	}
	if _, err := overEditor.CaptureSourceSnapshot(maxLanguageSourceBytes); err == nil {
		t.Fatal("source above explicit cap accepted")
	}
}

func TestLanguageSnapshotRejectsNativeAndMarklessPending(t *testing.T) {
	b := text.NewWithLimit("a b", 5)
	e := New(b)
	tester := multiTester(t, e)
	start := setSelections(t, b, []text.Range{{Anchor: 1, Head: 1}, {Anchor: 3, Head: 3}}, 1)
	target := captureLanguageTarget(t, e)
	changes, reports := 0, 0
	e.OnChange = func() { changes++ }
	tester.Compose("拼🙂", 2)
	if e.compositionToken == (text.CompositionToken{}) || !e.compositionMulti || !b.InGroup() {
		t.Fatal("multi pending owner was not retained")
	}
	if _, marked := e.MarkedRange(); marked {
		t.Fatal("size-refused first preedit unexpectedly has a mark")
	}
	before := currentEditState(b)
	token := e.compositionToken
	e.OnError = func(error) { reports++ }
	requireLanguageReadRefusals(t, e, target, text.ErrCompositionPending, b)
	if err := e.RevealSourceTarget(SourceTarget{}, 0, 0); !errors.Is(err, text.ErrCompositionPending) {
		t.Fatalf("zero target did not reject pending input first: %v", err)
	}
	requireEditState(t, b, before)
	if !b.OwnsComposition(token) || changes != 0 || reports != 0 || !reflect.DeepEqual(b.SelectionSet(), start) {
		t.Fatal("pending rejection settled or changed the native owner")
	}
	tester.Key(0, ui.KeyEscape)
	if b.InGroup() || e.compositionToken != (text.CompositionToken{}) {
		t.Fatal("Escape did not release pending native owner")
	}
	if _, err := e.CaptureSourceSnapshot(64); err != nil {
		t.Fatalf("capture after explicit cancellation failed: %v", err)
	}
}

func TestLanguageSnapshotRejectsSingleCompositionErrorAndCompositionABA(t *testing.T) {
	b := text.NewWithLimit("a", 1)
	e := New(b)
	tester := multiTester(t, e)
	if err := b.SetSelection(1, 1); err != nil {
		t.Fatal(err)
	}
	target := captureLanguageTarget(t, e)
	before := currentEditState(b)
	tester.Compose("🙂", 1)
	if e.compositionError == nil || e.compositionBuffer != b || e.composing || e.compositionMarked {
		t.Fatal("single first-refusal fixture must retain an error owner without a mark")
	}
	if _, marked := e.MarkedRange(); marked {
		t.Fatal("single first refusal unexpectedly created a mark")
	}
	changes, reports := 0, 0
	e.OnChange = func() { changes++ }
	e.OnError = func(error) { reports++ }
	requireLanguageReadRefusals(t, e, target, text.ErrSizeLimit, b)
	requireEditState(t, b, before)
	if changes != 0 || reports != 0 || e.compositionBuffer != b || e.compositionToken != (text.CompositionToken{}) {
		t.Fatal("composition-error refusal settled or reported the retained owner")
	}
	tester.Key(0, ui.KeyEscape)
	if e.compositionBuffer != nil || b.InGroup() {
		t.Fatal("Escape did not clear the no-mark error owner")
	}
	if err := e.CheckSourceTarget(target, false); err == nil {
		t.Fatal("composition-generation ABA revived the earlier target")
	}
}

func TestLanguageSnapshotRejectsForeignBufferGroup(t *testing.T) {
	for _, composition := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "composition"}[composition], func(t *testing.T) {
			b := text.New("source")
			e := New(b)
			target := captureLanguageTarget(t, e)
			var token text.CompositionToken
			if composition {
				var err error
				token, err = b.BeginCompositionGroup()
				if err != nil {
					t.Fatal(err)
				}
			} else if err := b.BeginGroup(); err != nil {
				t.Fatal(err)
			}
			before := currentEditState(b)
			changes, reports := 0, 0
			e.OnChange = func() { changes++ }
			e.OnError = func(error) { reports++ }
			requireLanguageReadRefusals(t, e, target, text.ErrGroup, b)
			requireEditState(t, b, before)
			if changes != 0 || reports != 0 || e.compositionToken != (text.CompositionToken{}) {
				t.Fatal("foreign group refusal changed callbacks or editor token")
			}
			if composition {
				if !b.OwnsComposition(token) || b.CancelCompositionGroup(token) != nil {
					t.Fatal("foreign composition owner was disturbed")
				}
			} else if !b.InGroup() || !b.CancelGroup() {
				t.Fatal("foreign ordinary group owner was disturbed")
			}
		})
	}
}
