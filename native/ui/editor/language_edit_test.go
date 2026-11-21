package editor

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func requireApplyRefusal(t *testing.T, e *Editor, target SourceTarget, edits []text.PatchEdit, policy SourceSelectionPolicy, want error, buffers ...*text.Buffer) {
	t.Helper()
	requirePureSourceRefusal(t, e, func() error {
		changed, err := e.ApplySourceEdits(target, edits, policy)
		if changed {
			t.Fatal("refused source edit reported a change")
		}
		return err
	}, want, buffers...)
}

func requireApplyRoundTrip(t *testing.T, e *Editor, edits []text.PatchEdit, policy SourceSelectionPolicy, expected string, after text.SelectionSet) {
	t.Helper()
	b := e.Buffer
	source, before, history := b.String(), b.SelectionSet(), b.History()
	calls := 0
	e.OnChange = func() { calls++ }
	changed, err := e.ApplySourceEdits(captureLanguageTarget(t, e), edits, policy)
	if err != nil || !changed || calls != 1 || b.String() != expected || b.History().UndoDepth != history.UndoDepth+1 || b.CanRedo() {
		t.Fatalf("source apply changed=%t err=%v calls=%d text=%q history=%+v", changed, err, calls, b.String(), b.History())
	}
	requireSelections(t, b, after)
	if !b.Undo() || b.String() != source || b.History().UndoDepth != history.UndoDepth {
		t.Fatal("source patch did not Undo in one step")
	}
	requireSelections(t, b, before)
	if !b.Redo() || b.String() != expected || b.CanRedo() || calls != 1 {
		t.Fatal("source patch did not Redo in one step")
	}
	requireSelections(t, b, after)
}

func TestLanguageApplyEditsRevalidatesFullSelectionAndComposition(t *testing.T) {
	base := text.SelectionSet{Ranges: []text.Range{{Anchor: 3, Head: 1}, {Anchor: 7, Head: 5}, {Anchor: 10, Head: 10}}, Primary: 1}
	for _, name := range []string{"secondary", "primary", "direction", "selection-ABA", "direct-buffer", "buffer-ABA", "direct-readonly", "readonly-ABA", "readonly-live", "direct-view", "release", "undo-journal-ABA", "restore", "foreign-issuer", "zero"} {
		t.Run(name, func(t *testing.T) {
			b := text.New("0123456789")
			e := New(b)
			tester := ui.NewTester(func(c *ui.Context) { ui.Button(c, "outside"); e.View(c) }, 420, 220)
			if err := tester.Click("outside"); err != nil {
				t.Fatal(err)
			}
			setSelections(t, b, base.Ranges, base.Primary)
			target := captureLanguageTarget(t, e)
			other := text.New("0123456789")
			want := errSourceTargetStale
			switch name {
			case "secondary":
				setSelections(t, b, []text.Range{{Anchor: 3, Head: 1}, {Anchor: 7, Head: 5}, {Anchor: 9, Head: 9}}, 1)
			case "primary":
				setSelections(t, b, base.Ranges, 0)
			case "direction":
				setSelections(t, b, []text.Range{{Anchor: 1, Head: 3}, {Anchor: 7, Head: 5}, {Anchor: 10, Head: 10}}, 1)
			case "selection-ABA":
				setSelections(t, b, []text.Range{{Anchor: 3, Head: 2}, {Anchor: 7, Head: 5}, {Anchor: 10, Head: 10}}, 1)
				setSelections(t, b, base.Ranges, base.Primary)
			case "direct-buffer":
				e.Buffer = other
			case "buffer-ABA":
				e.SetBuffer(other)
				e.SetBuffer(b)
			case "direct-readonly":
				e.ReadOnly = true
			case "readonly-ABA":
				e.SetReadOnly(true)
				e.SetReadOnly(false)
			case "readonly-live":
				e.SetReadOnly(true)
				target = captureLanguageTarget(t, e)
				want = ErrSourceReadOnly
			case "direct-view":
				e.viewGeneration++
			case "release":
				e.ReleaseView()
			case "undo-journal-ABA":
				for i := 0; i < 65; i++ {
					if err := b.ReplaceUTF16(0, 1, "X"); err != nil || !b.Undo() {
						t.Fatal("journal ABA fixture", err)
					}
				}
				if b.String() != "0123456789" || !reflect.DeepEqual(b.SelectionSet(), base) {
					t.Fatal("journal ABA did not return to original bytes/selections")
				}
			case "restore":
				b.Restore(b.Snapshot())
			case "foreign-issuer":
				target = captureLanguageTarget(t, New(b))
			case "zero":
				target = SourceTarget{}
			}
			changes, highlights, reports := 0, 0, 0
			e.OnChange = func() { changes++ }
			e.Highlight = func(string) []ui.Span { highlights++; return nil }
			e.OnError = func(error) { reports++ }
			for _, edits := range [][]text.PatchEdit{nil, {{Start: 1, End: 2, OldText: "1", NewText: "word"}}} {
				requireApplyRefusal(t, e, target, edits, PreserveMappedSelections, want, b, other)
			}
			if changes != 0 || highlights != 0 || reports != 0 || !tester.Focused("outside") {
				t.Fatal("refusal called host code or changed actual focus")
			}
		})
	}
	for _, name := range []string{"single-marked", "multi-marked", "single-markless", "multi-markless", "composition-ABA"} {
		t.Run(name, func(t *testing.T) {
			limit := 64
			if strings.Contains(name, "markless") {
				limit = 5
			}
			b := text.NewWithLimit("a b", limit)
			e := New(b)
			tester := multiTester(t, e)
			ranges := []text.Range{{Anchor: 1, Head: 1}}
			multi := strings.HasPrefix(name, "multi")
			if multi {
				ranges = append(ranges, text.Range{Anchor: 3, Head: 3})
			}
			before := setSelections(t, b, ranges, len(ranges)-1)
			target := captureLanguageTarget(t, e)
			tester.Compose("拼🙂", 2)
			token := e.compositionToken
			if strings.Contains(name, "markless") {
				if _, marked := e.MarkedRange(); marked {
					t.Fatal("first-refused preedit unexpectedly marked")
				}
			}
			var want error = text.ErrComposition
			if multi {
				want = text.ErrCompositionPending
			} else if name == "single-markless" {
				want = text.ErrSizeLimit
			} else if name == "composition-ABA" {
				tester.Key(0, ui.KeyEscape)
				want = errSourceTargetStale
			}
			calls, reports := 0, 0
			e.OnChange = func() { calls++ }
			e.OnError = func(error) { reports++ }
			requireApplyRefusal(t, e, target, []text.PatchEdit{{Start: 0, End: 1, OldText: "a", NewText: "A"}}, PreserveMappedSelections, want, b)
			requireApplyRefusal(t, e, SourceTarget{}, nil, SourceSelectionPolicy(255), want, b)
			if calls != 0 || reports != 0 || !tester.Focused(e.Label) {
				t.Fatal("composition refusal had host/focus effects")
			}
			if multi {
				if !b.OwnsComposition(token) {
					t.Fatal("refusal lost live multi token")
				}
				tester.Compose("x", 1)
				tester.Type("Y")
				if b.String() != "aY bY" || b.InGroup() || !b.Undo() || b.String() != "a b" {
					t.Fatal("original native owner could not continue/Undo", b.String())
				}
				requireSelections(t, b, before)
				if !b.Redo() || b.String() != "aY bY" {
					t.Fatal("native continuation Redo")
				}
			} else {
				tester.Key(0, ui.KeyEscape)
				if b.String() != "a b" || b.InGroup() || e.CompositionActive() {
					t.Fatal("original single owner could not cancel")
				}
				requireSelections(t, b, before)
			}
		})
	}
}

func TestLanguageApplyEditsForeignGroupsAndPurePlanRefusals(t *testing.T) {
	for _, composition := range []bool{false, true} {
		b := text.New("abc")
		e := New(b)
		setSelections(t, b, []text.Range{{Anchor: 1, Head: 0}, {Anchor: 3, Head: 3}}, 0)
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
		for _, edits := range [][]text.PatchEdit{nil, {{Start: 0, End: 1, OldText: "a", NewText: "a"}}, {{Start: 0, End: 1, OldText: "a", NewText: "A"}}} {
			requireApplyRefusal(t, e, target, edits, PreserveMappedSelections, text.ErrGroup, b)
		}
		if composition {
			if !b.OwnsComposition(token) || b.ReplaceCompositionUTF16(token, 0, 1, "A") != nil || b.EndCompositionGroup(token) != nil {
				t.Fatal("foreign composition could not continue")
			}
		} else if b.ReplaceUTF16(0, 1, "A") != nil || !b.EndGroup() {
			t.Fatal("foreign ordinary group could not continue")
		}
		if !b.Undo() || b.String() != "abc" || !b.Redo() || b.String() != "Abc" {
			t.Fatal("foreign history continuation")
		}
	}
	b := text.New("🙂abcd")
	e := New(b)
	target := captureLanguageTarget(t, e)
	for _, tc := range []struct {
		edits  []text.PatchEdit
		policy SourceSelectionPolicy
		want   error
	}{
		{nil, SourceSelectionPolicy(255), ErrSourceSelectionPolicy},
		{nil, SingleInsertionCaret, ErrSourceSelectionPolicy},
		{[]text.PatchEdit{{}, {}}, SingleInsertionCaret, ErrSourceSelectionPolicy},
		{make([]text.PatchEdit, 1025), PreserveMappedSelections, text.ErrPatchLimit},
		{[]text.PatchEdit{{Start: 2, End: 3, OldText: "a", NewText: "X"}, {Start: 4, End: 5, OldText: "wrong", NewText: "Y"}}, PreserveMappedSelections, text.ErrPatchOldText},
		{[]text.PatchEdit{{Start: 0, End: 1, NewText: "x"}}, PreserveMappedSelections, text.ErrSurrogate},
		{[]text.PatchEdit{{Start: 0, End: math.MaxInt, NewText: "x"}}, PreserveMappedSelections, text.ErrRange},
		{[]text.PatchEdit{{Start: 2, End: 3, OldText: "a", NewText: "\xff"}}, PreserveMappedSelections, text.ErrUTF8},
		{[]text.PatchEdit{{Start: 2, End: 3, OldText: "a", NewText: "A"}, {Start: 3, End: 3, NewText: "B"}}, PreserveMappedSelections, text.ErrEditOrder},
	} {
		requireApplyRefusal(t, e, target, tc.edits, tc.policy, tc.want, b)
	}
	// Directly exercise checked arithmetic without allocating impossible strings.
	for _, values := range [][3]int{{math.MaxInt, 1, 0}, {-1, 1, 0}, {1, 0, 1}, {1, 1, -1}} {
		if _, err := sourcePatchOffset(values[0], values[1], values[2]); !errors.Is(err, text.ErrRange) {
			t.Fatal("unchecked source mapping arithmetic", values, err)
		}
	}
	b.SetHistoryLimits(0, 0)
	target = captureLanguageTarget(t, e)
	requireApplyRefusal(t, e, target, []text.PatchEdit{{Start: 2, End: 3, OldText: "a", NewText: "A"}}, PreserveMappedSelections, text.ErrPatchHistoryLimit, b)
}

func TestLanguageApplyEditsMapsCompleteSelectionsOnce(t *testing.T) {
	b := text.New(strings.Repeat("a ", 1024))
	e := New(b)
	ranges := make([]text.Range, 256)
	after := text.SelectionSet{Ranges: make([]text.Range, 256), Primary: 190}
	edits := make([]text.PatchEdit, 1024)
	for i := range edits {
		edits[i] = text.PatchEdit{Start: 2 * i, End: 2*i + 1, OldText: "a", NewText: "🙂"}
	}
	for i := range ranges {
		ranges[i] = text.Range{Anchor: 8*i + 1, Head: 8 * i}
		after.Ranges[i] = text.Range{Anchor: 12*i + 2, Head: 12 * i}
	}
	setSelections(t, b, ranges, 190)
	requireApplyRoundTrip(t, e, edits, PreserveMappedSelections, strings.Repeat("🙂 ", 1024), after)
	for _, tc := range []struct {
		name, source, expected string
		before, after          text.SelectionSet
		edits                  []text.PatchEdit
		policy                 SourceSelectionPolicy
	}{
		{"interior", "abcdef", "a🙂f", text.SelectionSet{Ranges: []text.Range{{Anchor: 0, Head: 2}, {Anchor: 4, Head: 6}}, Primary: 1}, text.SelectionSet{Ranges: []text.Range{{Anchor: 0, Head: 3}, {Anchor: 3, Head: 4}}, Primary: 1}, []text.PatchEdit{{Start: 1, End: 5, OldText: "bcde", NewText: "🙂"}}, PreserveMappedSelections},
		{"reverse", "abcdef", "a🙂f", text.SelectionSet{Ranges: []text.Range{{Anchor: 2, Head: 0}, {Anchor: 6, Head: 4}}, Primary: 0}, text.SelectionSet{Ranges: []text.Range{{Anchor: 3, Head: 0}, {Anchor: 4, Head: 3}}, Primary: 0}, []text.PatchEdit{{Start: 1, End: 5, OldText: "bcde", NewText: "🙂"}}, PreserveMappedSelections},
		{"collision-primary-caret", "abcdef", "a🙂f", text.SelectionSet{Ranges: []text.Range{{Anchor: 2, Head: 0}, {Anchor: 3, Head: 3}, {Anchor: 6, Head: 4}}, Primary: 1}, text.SelectionSet{Ranges: []text.Range{{Anchor: 3, Head: 0}, {Anchor: 4, Head: 3}}, Primary: 1}, []text.PatchEdit{{Start: 1, End: 5, OldText: "bcde", NewText: "🙂"}}, PreserveMappedSelections},
		{"insertion-right-affinity", "ab", "a🙂b", text.SelectionSet{Ranges: []text.Range{{Anchor: 1, Head: 1}}}, text.SelectionSet{Ranges: []text.Range{{Anchor: 3, Head: 3}}}, []text.PatchEdit{{Start: 1, End: 1, NewText: "🙂"}}, PreserveMappedSelections},
		{"adjacent-boundary", "ab", "XYZ", text.SelectionSet{Ranges: []text.Range{{Anchor: 1, Head: 1}}}, text.SelectionSet{Ranges: []text.Range{{Anchor: 2, Head: 2}}}, []text.PatchEdit{{Start: 0, End: 1, OldText: "a", NewText: "XY"}, {Start: 1, End: 2, OldText: "b", NewText: "Z"}}, PreserveMappedSelections},
		{"collapse", "abcdef", "aXYf", text.SelectionSet{Ranges: []text.Range{{Anchor: 2, Head: 2}, {Anchor: 4, Head: 3}}, Primary: 1}, text.SelectionSet{Ranges: []text.Range{{Anchor: 3, Head: 3}}}, []text.PatchEdit{{Start: 1, End: 5, OldText: "bcde", NewText: "XY"}}, PreserveMappedSelections},
		{"single-policy", "ab\r\ncd\nef", "ab\r\n🙂é\nef", text.SelectionSet{Ranges: []text.Range{{Anchor: 1, Head: 0}, {Anchor: 6, Head: 4}}, Primary: 1}, text.SelectionSet{Ranges: []text.Range{{Anchor: 8, Head: 8}}}, []text.PatchEdit{{Start: 4, End: 6, OldText: "cd", NewText: "🙂é"}}, SingleInsertionCaret},
		{"eof", "a\r\nb\n", "a\r\nb\n🙂", text.SelectionSet{Ranges: []text.Range{{Anchor: 5, Head: 5}}}, text.SelectionSet{Ranges: []text.Range{{Anchor: 7, Head: 7}}}, []text.PatchEdit{{Start: 5, End: 5, NewText: "🙂"}}, SingleInsertionCaret},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := text.New(tc.source)
			e := New(b)
			setSelections(t, b, tc.before.Ranges, tc.before.Primary)
			requireApplyRoundTrip(t, e, tc.edits, tc.policy, tc.expected, tc.after)
		})
	}
}

func TestLanguageApplyEditsNoOpPreservesCompleteStateAndFocus(t *testing.T) {
	b := text.New("🙂x")
	e := New(b)
	tester := ui.NewTester(func(c *ui.Context) { ui.Button(c, "outside"); e.View(c) }, 420, 220)
	if err := tester.Click("outside"); err != nil {
		t.Fatal(err)
	}
	setSelections(t, b, []text.Range{{Anchor: 2, Head: 0}, {Anchor: 3, Head: 3}}, 1)
	if err := b.ReplaceUTF16(3, 3, "!"); err != nil || !b.Undo() {
		t.Fatal("redo fixture", err)
	}
	target := captureLanguageTarget(t, e)
	calls, reports, highlights := 0, 0, 0
	e.OnChange = func() { calls++ }
	e.OnError = func(error) { reports++ }
	e.Highlight = func(string) []ui.Span { highlights++; return nil }
	for _, tc := range []struct {
		edits  []text.PatchEdit
		policy SourceSelectionPolicy
	}{
		{nil, PreserveMappedSelections},
		{[]text.PatchEdit{{Start: 0, End: 2, OldText: "🙂", NewText: "🙂"}}, SingleInsertionCaret},
		{[]text.PatchEdit{{Start: 0, End: 2, OldText: "🙂"}, {Start: 2, End: 3, OldText: "x", NewText: "🙂x"}}, PreserveMappedSelections},
	} {
		before, bufferBefore := captureSourceEditorRejectState(e), captureSourceRejectState(b)
		if changed, err := e.ApplySourceEdits(target, tc.edits, tc.policy); changed || err != nil {
			t.Fatal("no-op", changed, err)
		}
		if !reflect.DeepEqual(before, captureSourceEditorRejectState(e)) {
			t.Fatal("no-op changed editor state")
		}
		requireSourceRejectState(t, b, bufferBefore)
	}
	if calls != 0 || reports != 0 || highlights != 0 || !tester.Focused("outside") || !b.CanRedo() || !b.Redo() || b.String() != "🙂x!" {
		t.Fatal("no-op caused callbacks/focus or lost redo")
	}
}

func TestLanguageApplyEditsOnChangeAttachmentReplacementPreservesExit(t *testing.T) {
	f := newLanguageReentryFixture(t, true)
	e, b := f.e, f.b
	beforeText, beforeSelections := b.String(), b.SelectionSet()
	start := f.span.Start()
	other := text.New("callback replacement\r\n🙂 new attachment")
	var exit []any
	var exitBuffers [2]sourceRejectState
	calls, highlights, reports := 0, 0, 0
	e.Highlight = func(string) []ui.Span { highlights++; return nil }
	e.OnError = func(error) { reports++ }
	e.OnChange = func() {
		calls++
		if !e.revealCaret || e.hasDesiredX || e.affinity != ui.TextDownstream || e.version != b.Version() || len(e.folds.ranges) != 0 || e.folds.ready {
			t.Fatal("callback ran before pure bookkeeping and fold invalidation")
		}
		e.SetBuffer(other)
		e.SetReadOnly(true)
		e.ReleaseView()
		e.SetFontSize(21)
		e.ScrollX, e.ScrollY, e.desiredX, e.hasDesiredX = 11, 17, 23, true
		e.revealCaret, e.affinity = false, ui.TextUpstream
		e.blinkStart = time.Unix(55, 0)
		if err := other.ReplaceUTF16(0, 8, "host"); err != nil {
			t.Fatal(err)
		}
		e.OnChange = func() { t.Fatal("old apply invoked replacement callback") }
		exit = captureSourceEditorRejectState(e)
		exitBuffers = [2]sourceRejectState{captureSourceRejectState(b), captureSourceRejectState(other)}
	}
	changed, err := e.ApplySourceEdits(f.target, []text.PatchEdit{{Start: start, End: start + 2, OldText: "🙂", NewText: "new🙂"}}, SingleInsertionCaret)
	if !changed || err != nil || calls != 1 || highlights != 0 || reports != 0 || exit == nil {
		t.Fatal("apply callbacks", changed, err, calls, highlights, reports)
	}
	if !reflect.DeepEqual(exit, captureSourceEditorRejectState(e)) {
		t.Fatal("apply overwrote synchronous callback's attachment exit state")
	}
	requireSourceRejectState(t, b, exitBuffers[0])
	requireSourceRejectState(t, other, exitBuffers[1])
	if !f.tester.Focused("outside") || b.History().UndoDepth != 1 {
		t.Fatal("apply changed focus or did not keep old buffer's committed item")
	}
	patched := b.String()
	after := text.SelectionSet{Ranges: []text.Range{{Anchor: start + 5, Head: start + 5}}}
	requireSelections(t, b, after)
	if !b.Undo() || b.String() != beforeText {
		t.Fatal("reentrant apply Undo")
	}
	requireSelections(t, b, beforeSelections)
	if !b.Redo() || b.String() != patched {
		t.Fatal("reentrant apply Redo")
	}
	requireSelections(t, b, after)
}

func TestLanguageApplyEditsDeferredRevealOnActualTesterFrame(t *testing.T) {
	const source = "first\r\nsecond\n"
	b := text.New(source)
	e := New(b)
	tester := multiTester(t, e)
	setSelections(t, b, []text.Range{{Anchor: 2, Head: 0}, {Anchor: b.LenUTF16(), Head: b.LenUTF16()}}, 1)
	before := b.SelectionSet()
	target := captureLanguageTarget(t, e)
	calls, highlights, reports := 0, 0, 0
	e.OnChange = func() { calls++ }
	e.Highlight = func(string) []ui.Span { highlights++; return nil }
	e.OnError = func(error) { reports++ }
	insert := strings.Repeat("new line\r\n", 28) + "🙂end"
	end := b.LenUTF16() + text.UTF16Len(insert)
	changed, err := e.ApplySourceEdits(target, []text.PatchEdit{{Start: b.LenUTF16(), End: b.LenUTF16(), NewText: insert}}, SingleInsertionCaret)
	if !changed || err != nil || calls != 1 || highlights != 0 || reports != 0 || !e.revealCaret {
		t.Fatal("apply synchronously queried geometry", changed, err, calls, highlights, reports)
	}
	afterBuffer := captureSourceRejectState(b)
	tester.Frame()
	requireSourceRejectState(t, b, afterBuffer)
	if e.revealCaret || e.ScrollY <= 0 || highlights == 0 || reports != 0 || calls != 1 || !tester.Focused(e.Label) {
		t.Fatal("ordinary frame did not reveal new caret", e.revealCaret, e.ScrollY, highlights, reports, calls)
	}
	bounds, actual, ok := e.BoundsForRange(ui.TextInputRange{Start: end, End: end})
	if !ok || actual.Start != end || actual.End != end || bounds.H <= 0 || bounds.Y < 0 || bounds.Y > e.height {
		t.Fatal("new caret geometry", bounds, actual, ok, e.height)
	}
	if b.String() != source+insert || b.History().UndoDepth != 1 || !b.Undo() || b.String() != source {
		t.Fatal("frame changed patch/history")
	}
	requireSelections(t, b, before)
	if !b.Redo() || b.String() != source+insert {
		t.Fatal("frame patch Redo")
	}
	requireSelections(t, b, text.SelectionSet{Ranges: []text.Range{{Anchor: end, Head: end}}})
}
