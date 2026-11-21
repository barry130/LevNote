package editor

import (
	"errors"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

type languageHostGuardState struct {
	boundary                  sourceRevealBoundary
	editor                    []any
	buffer                    sourceRejectState
	shapes                    map[int]lineLayout
	outsideFocus, editorFocus bool
}

// Capture raw state before any frame can consume a focus request or recompute
// geometry. The Buffer snapshot includes the complete directed selections.
func captureLanguageHostGuardState(f languageReentryFixture) languageHostGuardState {
	return languageHostGuardState{
		boundary:     f.e.sourceRevealBoundary(),
		editor:       captureSourceEditorRejectState(f.e),
		buffer:       captureSourceRejectState(f.b),
		shapes:       cloneSourceLineCache(f.e.cache),
		outsideFocus: f.tester.Focused("outside"),
		editorFocus:  f.tester.Focused(f.e.Label),
	}
}

func TestLanguageHostGuardRejectsAtEntry(t *testing.T) {
	f := newLanguageReentryFixture(t, true)
	e := f.e
	generation := uint64(1)
	admitted := generation
	generation++
	stillCurrent := func() bool { return generation == admitted }
	callbacks := 0
	e.Highlight = func(string) []ui.Span { callbacks++; return nil }
	e.OnError = func(error) { callbacks++ }
	e.OnChange = func() { callbacks++ }
	before := captureLanguageHostGuardState(f)
	for _, start := range []int{f.span.Head, -1} {
		if err := e.RevealSourceTargetChecked(f.target, start, f.span.Anchor, stillCurrent); !errors.Is(err, errSourceTargetStale) {
			t.Fatalf("false entry predicate at start %d returned %v, want stale", start, err)
		}
		if !reflect.DeepEqual(captureLanguageHostGuardState(f), before) || callbacks != 0 {
			t.Fatal("entry refusal changed state or invoked a navigation callback")
		}
	}
	f.tester.Frame()
	if !f.tester.Focused("outside") {
		t.Fatal("entry refusal scheduled focus")
	}
}

func TestLanguageHostGuardRejectsInvalidationDuringPreparation(t *testing.T) {
	for _, trigger := range []string{"highlight", "long-error"} {
		t.Run(trigger, func(t *testing.T) {
			f := newLanguageReentryFixture(t, trigger == "long-error")
			e := f.e
			generation := uint64(1)
			admitted := generation
			stillCurrent := func() bool { return generation == admitted }
			var callbackEntry, callbackExit languageHostGuardState
			highlights, warnings, changes := 0, 0, 0
			invalidate := func() {
				callbackEntry = captureLanguageHostGuardState(f)
				generation++
				callbackExit = captureLanguageHostGuardState(f)
			}
			e.Highlight = func(value string) []ui.Span {
				highlights++
				if trigger == "highlight" {
					invalidate()
				}
				return []ui.Span{{Text: value}}
			}
			e.OnError = func(err error) {
				if !errors.Is(err, errLineTooLong) {
					t.Errorf("unexpected warning: %v", err)
				}
				warnings++
				if trigger == "long-error" {
					invalidate()
				}
			}
			e.OnChange = func() { changes++ }
			before := captureLanguageHostGuardState(f)

			err := e.RevealSourceTargetChecked(f.target, f.span.Head, f.span.Anchor, stillCurrent)
			if !errors.Is(err, errSourceTargetStale) || generation != admitted+1 {
				t.Errorf("Host invalidation returned %v at request generation %d", err, generation)
			}
			wantHighlights, wantWarnings := 1, 0
			if trigger == "long-error" {
				wantHighlights, wantWarnings = 0, 1
			}
			if highlights != wantHighlights || warnings != wantWarnings || changes != 0 {
				t.Errorf("callback counts highlight=%d warning=%d change=%d, want %d/%d/0", highlights, warnings, changes, wantHighlights, wantWarnings)
			}
			if !reflect.DeepEqual(before, callbackEntry) || !reflect.DeepEqual(callbackEntry, callbackExit) {
				t.Error("expected unchanged Editor and Buffer at callback entry and exit")
			}
			if !reflect.DeepEqual(captureLanguageHostGuardState(f), callbackExit) {
				t.Error("stale navigation overwrote callback exit state")
			}
			if err := e.CheckSourceTarget(f.target, true); err != nil {
				t.Errorf("Host-only invalidation changed the Editor target: %v", err)
			}
			e.Highlight, e.OnError = nil, nil
			f.tester.Frame()
			if !f.tester.Focused("outside") {
				t.Fatal("rejected navigation scheduled focus")
			}
		})
	}
}

func TestLanguageHostGuardCurrentAndLegacy(t *testing.T) {
	for _, mode := range []string{"checked", "legacy"} {
		t.Run(mode, func(t *testing.T) {
			f := newLanguageReentryFixture(t, true)
			e, b := f.e, f.b
			generation := uint64(1)
			admitted := generation
			stillCurrent := func() bool { return generation == admitted }
			var callbacks []string
			changes := 0
			e.Highlight = func(value string) []ui.Span {
				callbacks = append(callbacks, "highlight")
				if mode == "legacy" {
					generation++
				}
				return []ui.Span{{Text: value}}
			}
			e.OnError = func(err error) {
				if !errors.Is(err, errLineTooLong) {
					t.Errorf("unexpected warning: %v", err)
				}
				callbacks = append(callbacks, "warning")
			}
			e.OnChange = func() { changes++ }
			before := captureLanguageHostGuardState(f)
			var err error
			if mode == "checked" {
				err = e.RevealSourceTargetChecked(f.target, f.span.Head, f.span.Anchor, stillCurrent)
			} else {
				err = e.RevealSourceTarget(f.target, f.span.Head, f.span.Anchor)
			}
			if err != nil {
				t.Fatal(err)
			}
			want := text.SelectionSet{Ranges: []text.Range{{Anchor: f.span.Head, Head: f.span.Anchor}}, Primary: 0}
			if !reflect.DeepEqual(b.SelectionSet(), want) || b.SelectionGeneration() != before.buffer.edit.selectionGen+1 || e.navigationGeneration != before.boundary.navigation+1 {
				t.Fatal("expected one directed navigation commit")
			}
			if b.String() != before.buffer.edit.source || b.Version() != before.buffer.edit.version || b.History() != before.buffer.edit.history || !b.CanRedo() || changes != 0 {
				t.Fatal("navigation changed source, history or edit callbacks")
			}
			if !e.longLineWarned || e.cache[f.line].layout == nil || len(e.folds.collapsed) != 0 || e.ScrollY <= 0 || e.hasDesiredX {
				t.Fatal("navigation did not commit prepared layout, folds and scroll")
			}
			if !reflect.DeepEqual(callbacks, []string{"warning", "highlight"}) || stillCurrent() != (mode == "checked") {
				t.Fatalf("wrong callback order or request generation: %v, %d", callbacks, generation)
			}
			e.Highlight, e.OnError = nil, nil
			f.tester.Frame()
			if !f.tester.Focused(e.Label) {
				t.Fatal("committed navigation did not focus the Editor")
			}
		})
	}
}

func TestLanguageHostGuardWarmCache(t *testing.T) {
	f := newLanguageReentryFixture(t, false)
	e, b := f.e, f.b
	warm := e.line(0)
	if warm.layout == nil {
		t.Fatal("expected warm layout for the first line")
	}
	callbacks := 0
	e.Highlight = func(string) []ui.Span { callbacks++; return nil }
	e.OnError = func(error) { callbacks++ }
	e.OnChange = func() { callbacks++ }
	generation := uint64(2)
	const start, end = 6, 2
	before := captureLanguageHostGuardState(f)
	if err := e.RevealSourceTargetChecked(f.target, start, end, func() bool { return generation == 1 }); !errors.Is(err, errSourceTargetStale) {
		t.Fatalf("stale request for cached layout returned %v", err)
	}
	if !reflect.DeepEqual(before, captureLanguageHostGuardState(f)) {
		t.Fatal("stale warm reveal changed raw state")
	}
	admitted := generation
	checks := 0
	if err := e.RevealSourceTargetChecked(f.target, start, end, func() bool {
		// Test instrumentation counts evaluations without affecting the result.
		checks++
		return generation == admitted
	}); err != nil {
		t.Fatal(err)
	}
	if b.Selection() != (text.Range{Anchor: start, Head: end}) || b.SelectionGeneration() != before.buffer.edit.selectionGen+1 || e.cache[0].layout != warm.layout || callbacks != 0 {
		t.Fatal("current request failed to navigate through warm geometry without callbacks")
	}
	if checks < 2 {
		t.Fatalf("warm reveal checked the request %d times, want entry and precommit", checks)
	}
}

func TestLanguageHostGuardPreservesNestedReveal(t *testing.T) {
	f := newLanguageReentryFixture(t, false)
	e, b := f.e, f.b
	e.line(0)
	generation := uint64(1)
	original := generation
	var callbackEntry, callbackExit languageHostGuardState
	highlights, changes := 0, 0
	const start, end = 6, 2
	e.OnChange = func() { changes++ }
	e.Highlight = func(value string) []ui.Span {
		highlights++
		if highlights != 1 {
			t.Fatal("expected a single cold Highlight and a warm nested reveal")
		}
		callbackEntry = captureLanguageHostGuardState(f)
		generation++
		fresh := generation
		if err := e.RevealSourceTargetChecked(captureLanguageTarget(t, e), start, end, func() bool { return generation == fresh }); err != nil {
			t.Fatal("fresh nested navigation", err)
		}
		callbackExit = captureLanguageHostGuardState(f)
		return []ui.Span{{Text: value}}
	}
	before := captureLanguageHostGuardState(f)
	if err := e.RevealSourceTargetChecked(f.target, f.span.Head, f.span.Anchor, func() bool { return generation == original }); !errors.Is(err, errSourceTargetStale) {
		t.Fatalf("old request returned %v after fresh nested navigation", err)
	}
	if highlights != 1 || changes != 0 || !reflect.DeepEqual(before, callbackEntry) || !reflect.DeepEqual(callbackExit, captureLanguageHostGuardState(f)) {
		t.Fatal("outer reveal changed callback entry or discarded callback exit")
	}
	if b.Selection() != (text.Range{Anchor: start, Head: end}) || len(b.SelectionSet().Ranges) != 1 || e.navigationGeneration != before.boundary.navigation+1 || len(e.folds.collapsed) != 2 {
		t.Fatal("expected only the nested selection and geometry commit")
	}
	e.Highlight = nil
	f.tester.Frame()
	if !f.tester.Focused(e.Label) {
		t.Fatal("fresh nested navigation lost its focus request")
	}
}
