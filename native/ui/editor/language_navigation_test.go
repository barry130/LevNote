package editor

import (
	"errors"
	"maps"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func captureLanguageTarget(t *testing.T, e *Editor) SourceTarget {
	t.Helper()
	snapshot, err := e.CaptureSourceSnapshot(2 << 20)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot.Target
}

type sourceRejectState struct {
	edit             editState
	snapshot         text.Snapshot
	canUndo, canRedo bool
	changed          int
	mapped           text.Range
	mappedOK         bool
}

func captureSourceRejectState(b *text.Buffer) sourceRejectState {
	span := b.Selection()
	mapped, ok := b.MapRangeSince(b.Version(), span)
	return sourceRejectState{
		edit:     currentEditState(b),
		snapshot: b.Snapshot(),
		canUndo:  b.CanUndo(),
		canRedo:  b.CanRedo(),
		changed:  b.ChangedLineSince(b.Version()),
		mapped:   mapped,
		mappedOK: ok,
	}
}

func requireSourceRejectState(t *testing.T, b *text.Buffer, want sourceRejectState) {
	t.Helper()
	if got := captureSourceRejectState(b); !reflect.DeepEqual(got, want) {
		t.Fatalf("source refusal changed observable Buffer state: before=%+v after=%+v", want, got)
	}
}

// Read raw fields here: the refusal oracle must not itself observe an owner,
// sync folds, query geometry or run a frame that could hide an admission write.
func captureSourceEditorRejectState(e *Editor) []any {
	var folds any
	if e.folds != nil {
		copy := *e.folds
		copy.provider = nil // function identity is covered by the retained foldState
		copy.reported = maps.Clone(e.folds.reported)
		copy.collapsed = maps.Clone(e.folds.collapsed)
		copy.ranges = append([]FoldRange(nil), e.folds.ranges...)
		copy.intervals = append([]foldInterval(nil), e.folds.intervals...)
		folds = copy
	}
	return []any{
		e.Buffer, e.ReadOnly, e.inputOwner, e.viewGeneration, e.nativeClient,
		e.compositionBuffer, e.compositionToken, e.compositionError, e.compositionStamp,
		e.compositionGeneration, e.composing, e.compositionMulti, e.compositionMarked,
		e.compositionPrimary, e.compositionTargetAdmitted, e.marked,
		e.focused, e.dragging, e.ScrollX, e.ScrollY, e.revealCaret, e.longLineWarned,
		e.affinity, e.desiredX, e.hasDesiredX, e.blinkStart,
		e.width, e.height, e.version, e.cachedBuffer, e.cachedShapeLimit, e.cachedGutter,
		e.indexedLines, maps.Clone(e.cache), maps.Clone(e.rowCounts), maps.Clone(e.rowTree),
		e.folds, folds, e.foldControl, e.pendingClipboard,
		e.geometryGeneration, e.navigationGeneration,
	}
}

func requirePureSourceRefusal(t *testing.T, e *Editor, action func() error, want error, buffers ...*text.Buffer) {
	t.Helper()
	beforeEditor := captureSourceEditorRejectState(e)
	beforeBuffers := make([]sourceRejectState, len(buffers))
	for i, b := range buffers {
		beforeBuffers[i] = captureSourceRejectState(b)
	}
	if err := action(); !errors.Is(err, want) {
		t.Fatalf("source refusal = %v, want %v", err, want)
	}
	if after := captureSourceEditorRejectState(e); !reflect.DeepEqual(beforeEditor, after) {
		t.Fatalf("source refusal changed owner/input/navigation state: before=%+v after=%+v", beforeEditor, after)
	}
	for i, b := range buffers {
		requireSourceRejectState(t, b, beforeBuffers[i])
	}
}

func TestLanguageTargetRefusalDoesNotObserveInputOwner(t *testing.T) {
	for _, name := range []string{"direct-buffer", "direct-readonly", "direct-view", "buffer-setter-ABA", "readonly-setter-ABA", "view-release", "foreign-editor", "zero-target"} {
		for _, operation := range []string{"check-read", "check-selection", "reveal"} {
			t.Run(name+"/"+operation, func(t *testing.T) {
				const source = "{\n🙂 inside\n}\nafter"
				b := text.New(source)
				if err := b.ReplaceUTF16(b.LenUTF16(), b.LenUTF16(), "!"); err != nil || !b.Undo() {
					t.Fatal("redo fixture", err)
				}
				e := New(b)
				if err := installBraceProvider(e); err != nil {
					t.Fatal(err)
				}
				tester := ui.NewTester(func(c *ui.Context) {
					ui.Button(c, "outside")
					e.View(c)
				}, 420, 220)
				if err := tester.Click("outside"); err != nil {
					t.Fatal(err)
				}
				if err := e.ToggleFoldAt(0); err != nil {
					t.Fatal(err)
				}
				setSelections(t, b, []text.Range{{Anchor: 0, Head: 0}, {Anchor: 5, Head: 2}}, 1)
				target := captureLanguageTarget(t, e)
				other := text.New(source)
				switch name {
				case "direct-buffer":
					e.Buffer = other
				case "direct-readonly":
					e.ReadOnly = true
				case "direct-view":
					e.viewGeneration++
				case "buffer-setter-ABA":
					e.SetBuffer(other)
					e.SetBuffer(b)
				case "readonly-setter-ABA":
					e.SetReadOnly(true)
					e.SetReadOnly(false)
				case "view-release":
					e.ReleaseView()
				case "foreign-editor":
					target = captureLanguageTarget(t, New(b))
				case "zero-target":
					target = SourceTarget{}
				}
				changes, reports, clipboard := 0, 0, 0
				e.OnChange = func() { changes++ }
				e.OnError = func(error) { reports++ }
				e.Copy = func(string) error { clipboard++; return nil }
				e.Paste = func() (string, error) { clipboard++; return "", nil }
				action := func() error { return e.CheckSourceTarget(target, operation == "check-selection") }
				if operation == "reveal" {
					// Valid UTF-16 endpoints inside a collapsed fold: an admitted
					// Reveal would expand it, replace the full selection and focus.
					action = func() error { return e.RevealSourceTarget(target, 5, 2) }
				}
				requirePureSourceRefusal(t, e, action, errSourceTargetStale, b, other)
				if changes != 0 || reports != 0 || clipboard != 0 || tester.Focused(e.Label) || !tester.Focused("outside") {
					t.Fatal("refusal changed callbacks, clipboard or actual focus")
				}
			})
		}
	}
}

func TestLanguageTargetRejectsAttachmentAndVersionABA(t *testing.T) {
	for _, name := range []string{"foreign-editor", "buffer-ABA", "readonly-ABA", "view-release", "restore-same-bytes", "version-ABA"} {
		t.Run(name, func(t *testing.T) {
			b := text.New("source")
			e := New(b)
			target := captureLanguageTarget(t, e)
			other := text.New("other")
			switch name {
			case "foreign-editor":
				foreign := New(b)
				if err := foreign.CheckSourceTarget(target, false); err == nil {
					t.Fatal("target crossed editor issuer")
				}
				return
			case "buffer-ABA":
				e.SetBuffer(other)
				e.SetBuffer(b)
			case "readonly-ABA":
				e.SetReadOnly(true)
				e.SetReadOnly(false)
			case "view-release":
				e.ReleaseView()
			case "restore-same-bytes":
				seed := b.Snapshot()
				b.Restore(seed)
			case "version-ABA":
				if err := b.ReplaceUTF16(b.LenUTF16(), b.LenUTF16(), "!"); err != nil || !b.Undo() {
					t.Fatal("version ABA fixture", err)
				}
			}
			if b.String() != "source" {
				t.Fatal("ABA fixture changed visible source")
			}
			if err := e.CheckSourceTarget(target, false); err == nil {
				t.Fatal("stale target accepted after " + name)
			}
		})
	}
}

func TestLanguageTargetSelectionModes(t *testing.T) {
	base := text.SelectionSet{Ranges: []text.Range{{Anchor: 3, Head: 1}, {Anchor: 7, Head: 5}, {Anchor: 10, Head: 10}}, Primary: 1}
	for _, tc := range []struct {
		name   string
		mutate func(*text.Buffer)
	}{
		{"secondary-only", func(b *text.Buffer) {
			_ = b.SetSelectionSet(text.SelectionSet{Ranges: []text.Range{{Anchor: 3, Head: 1}, {Anchor: 7, Head: 5}, {Anchor: 9, Head: 9}}, Primary: 1})
		}},
		{"primary-index", func(b *text.Buffer) {
			_ = b.SetSelectionSet(text.SelectionSet{Ranges: append([]text.Range(nil), base.Ranges...), Primary: 0})
		}},
		{"direction", func(b *text.Buffer) {
			_ = b.SetSelectionSet(text.SelectionSet{Ranges: []text.Range{{Anchor: 1, Head: 3}, {Anchor: 7, Head: 5}, {Anchor: 10, Head: 10}}, Primary: 1})
		}},
		{"selection-ABA", func(b *text.Buffer) {
			_ = b.SetSelectionSet(text.SelectionSet{Ranges: []text.Range{{Anchor: 3, Head: 2}, {Anchor: 7, Head: 5}, {Anchor: 10, Head: 10}}, Primary: 1})
			_ = b.SetSelectionSet(base)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := text.New("0123456789")
			if err := b.SetSelectionSet(base); err != nil {
				t.Fatal(err)
			}
			e := New(b)
			target := captureLanguageTarget(t, e)
			version := b.Version()
			tc.mutate(b)
			if b.Version() != version {
				t.Fatal("selection-only fixture changed text version")
			}
			if err := e.CheckSourceTarget(target, false); err != nil {
				t.Fatalf("read/navigation target rejected ordinary selection change: %v", err)
			}
			if err := e.CheckSourceTarget(target, true); err == nil {
				t.Fatal("selection-sensitive target accepted changed full selection identity")
			}
		})
	}
	b := text.New("0123456789")
	if err := b.SetSelectionSet(base); err != nil {
		t.Fatal(err)
	}
	e := New(b)
	target := captureLanguageTarget(t, e)
	if err := e.CheckSourceTarget(target, true); err != nil {
		t.Fatalf("unchanged complete selection rejected: %v", err)
	}
}

func TestLanguageRevealUsesUtf16AndFoldGeometry(t *testing.T) {
	const source = "{\r\n🙂 inside\n}\r\ntail"
	b := text.New(source)
	e := New(b)
	tester := ui.NewTester(e.View, 520, 280)
	if err := installBraceProvider(e); err != nil {
		t.Fatal(err)
	}
	if err := e.ToggleFoldAt(0); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	hidden, err := b.LineRange(1)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, visible := e.BoundsForRange(ui.TextInputRange{Start: hidden.Start(), End: hidden.Start()}); visible {
		t.Fatal("folded source unexpectedly had visible geometry")
	}
	tail, err := b.LineRange(3)
	if err != nil {
		t.Fatal(err)
	}
	beforeSelections := setSelections(t, b, []text.Range{{Anchor: 0, Head: 0}, {Anchor: tail.Start(), Head: tail.Start()}}, 1)
	e.SetReadOnly(true)
	changes, reports := 0, 0
	e.OnChange = func() { changes++ }
	e.OnError = func(error) { reports++ }
	target := captureLanguageTarget(t, e)
	if !reflect.DeepEqual(target.selections, beforeSelections) {
		t.Fatalf("target did not retain complete before-selection set: %+v", target.selections)
	}
	beforeSource, beforeVersion, beforeHistory := b.String(), b.Version(), b.History()
	beforeGeneration := b.SelectionGeneration()
	if err := e.RevealSourceTarget(target, hidden.End(), hidden.Start()); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if e.FoldCandidates()[0].Collapsed || b.Selection() != (text.Range{Anchor: hidden.End(), Head: hidden.Start()}) {
		t.Fatalf("reveal did not expand and preserve direction: folds=%+v selection=%+v", e.FoldCandidates(), b.Selection())
	}
	if b.String() != beforeSource || b.Version() != beforeVersion || b.History() != beforeHistory || changes != 0 || reports != 0 {
		t.Fatal("read-only reveal changed document/history/callbacks")
	}
	if !tester.Focused(e.Label) {
		t.Fatal("successful Reveal did not focus the real editor")
	}
	if got := b.SelectionSet(); !reflect.DeepEqual(got, text.SelectionSet{Ranges: []text.Range{{Anchor: hidden.End(), Head: hidden.Start()}}, Primary: 0}) {
		t.Fatalf("navigation after-set did not match the existing collapse semantics: before=%+v after=%+v", beforeSelections, got)
	}
	if _, _, visible := e.BoundsForRange(ui.TextInputRange{Start: hidden.Start(), End: hidden.Start()}); !visible {
		t.Fatal("revealed folded target has no visible geometry")
	}
	if b.SelectionGeneration() == beforeGeneration {
		t.Fatal("Reveal did not publish its existing selection navigation")
	}
	if err := e.RevealSourceTarget(target, b.LenUTF16(), b.LenUTF16()); err != nil {
		t.Fatalf("EOF zero-width reveal failed: %v", err)
	}
}

func TestLanguageRevealRejectsInvalidUTF16BeforeAnyNavigation(t *testing.T) {
	b := text.New("{\n🙂x\n}\n")
	e := New(b)
	if err := installBraceProvider(e); err != nil {
		t.Fatal(err)
	}
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Button(c, "outside")
		e.View(c)
	}, 420, 220)
	if err := tester.Click("outside"); err != nil {
		t.Fatal(err)
	}
	if err := e.ToggleFoldAt(0); err != nil {
		t.Fatal(err)
	}
	if err := b.SetSelectionSet(text.SelectionSet{Ranges: []text.Range{{Anchor: 0, Head: 0}, {Anchor: 8, Head: 8}}, Primary: 1}); err != nil {
		t.Fatal(err)
	}
	target := captureLanguageTarget(t, e)
	before := captureSourceRejectState(b)
	foldsBefore := e.FoldCandidates()
	projectionBefore, scrollX, scrollY := e.folds.projection, e.ScrollX, e.ScrollY
	changes, reports := 0, 0
	e.OnChange = func() { changes++ }
	e.OnError = func(error) { reports++ }
	for _, tc := range []struct {
		start, end int
		want       error
	}{
		{3, 3, text.ErrSurrogate},
		{0, 3, text.ErrSurrogate},
		{-1, 1, text.ErrRange},
		{0, 100, text.ErrRange},
	} {
		requirePureSourceRefusal(t, e, func() error { return e.RevealSourceTarget(target, tc.start, tc.end) }, tc.want, b)
		requireSourceRejectState(t, b, before)
		if changes != 0 || reports != 0 || tester.Focused(e.Label) || !reflect.DeepEqual(e.FoldCandidates(), foldsBefore) ||
			e.folds.projection != projectionBefore || e.ScrollX != scrollX || e.ScrollY != scrollY {
			t.Fatal("invalid endpoint changed callbacks, folds, scroll, focus or selection")
		}
	}
}

func TestLanguageRevealDoesNotSettleForeignInput(t *testing.T) {
	b := text.New("{\ninside\n}\nafter")
	e := New(b)
	if err := installBraceProvider(e); err != nil {
		t.Fatal(err)
	}
	if err := e.ToggleFoldAt(0); err != nil {
		t.Fatal(err)
	}
	target := captureLanguageTarget(t, e)
	foldsBefore := e.FoldCandidates()
	projectionBefore, scrollBefore, focusedBefore := e.folds.projection, e.ScrollY, e.focused
	if err := b.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	before := captureSourceRejectState(b)
	changes, reports := 0, 0
	e.OnChange = func() { changes++ }
	e.OnError = func(error) { reports++ }
	if err := e.RevealSourceTarget(target, 0, 0); !errors.Is(err, text.ErrGroup) {
		t.Fatalf("foreign group was not rejected first: %v", err)
	}
	requireSourceRejectState(t, b, before)
	if changes != 0 || reports != 0 || !b.InGroup() || !reflect.DeepEqual(e.FoldCandidates(), foldsBefore) ||
		e.folds.projection != projectionBefore || e.ScrollY != scrollBefore || e.focused != focusedBefore {
		t.Fatal("foreign group was settled or navigation/fold state changed")
	}
	if !b.CancelGroup() {
		t.Fatal("foreign group owner could not cancel after rejection")
	}

	// A first-refused multi preedit has no MarkedRange, but it still owns a
	// pending group and must block navigation without settling that owner.
	b = text.NewWithLimit("a b", 5)
	e = New(b)
	tester := multiTester(t, e)
	start := setSelections(t, b, []text.Range{{Anchor: 1, Head: 1}, {Anchor: 3, Head: 3}}, 1)
	target = captureLanguageTarget(t, e)
	tester.Compose("拼🙂", 2)
	changes, reports = 0, 0
	e.OnChange = func() { changes++ }
	e.OnError = func(error) { reports++ }
	beforePending := captureSourceRejectState(b)
	token := e.compositionToken
	if _, marked := e.MarkedRange(); marked {
		t.Fatal("pending fixture unexpectedly has marked range")
	}
	if err := e.RevealSourceTarget(target, 0, 0); !errors.Is(err, text.ErrCompositionPending) {
		t.Fatalf("markless pending navigation was not rejected: %v", err)
	}
	requireSourceRejectState(t, b, beforePending)
	if !b.OwnsComposition(token) || !reflect.DeepEqual(b.SelectionSet(), start) {
		t.Fatal("pending owner or complete selection changed")
	}
	tester.Key(0, ui.KeyEscape)
}

func TestLanguageTargetSelectionABAAndCrossBufferNeverUsesEqualText(t *testing.T) {
	b := text.New("same")
	e := New(b)
	target := captureLanguageTarget(t, e)
	other := text.New("same")
	e.SetBuffer(other)
	e.SetBuffer(b)
	if err := e.CheckSourceTarget(target, false); err == nil {
		t.Fatal("buffer ABA through setters revived equal text target")
	}
	b = text.New(strings.Repeat("x", 8))
	e = New(b)
	if err := b.SetSelection(2, 1); err != nil {
		t.Fatal(err)
	}
	target = captureLanguageTarget(t, e)
	if err := b.SetSelection(4, 3); err != nil {
		t.Fatal(err)
	}
	if err := b.SetSelection(2, 1); err != nil {
		t.Fatal(err)
	}
	if err := e.CheckSourceTarget(target, false); err != nil {
		t.Fatalf("read mode rejected ordinary selection ABA: %v", err)
	}
	if err := e.CheckSourceTarget(target, true); err == nil {
		t.Fatal("selection-generation ABA revived sensitive target")
	}
}
