package editor

import (
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func requireSelections(t *testing.T, b *text.Buffer, want text.SelectionSet) {
	t.Helper()
	if got := b.SelectionSet(); got.Primary != want.Primary || !slices.Equal(got.Ranges, want.Ranges) {
		t.Fatalf("selection set = %+v, want %+v", got, want)
	}
}

func setSelections(t *testing.T, b *text.Buffer, ranges []text.Range, primary int) text.SelectionSet {
	t.Helper()
	want := text.SelectionSet{Ranges: ranges, Primary: primary}
	if err := b.SetSelectionSet(want); err != nil {
		t.Fatal(err)
	}
	requireSelections(t, b, want)
	return want
}

func multiTester(t *testing.T, e *Editor) *ui.Tester {
	t.Helper()
	tester := ui.NewTester(e.View, 600, 280)
	if err := tester.Click(e.Label); err != nil {
		t.Fatal(err)
	}
	if active, noIME := tester.TextInputActive(); !active || noIME {
		t.Fatal("native text client is not active")
	}
	return tester
}

type editState struct {
	source                 string
	selections             text.SelectionSet
	version, selectionGen uint64
	history                text.HistoryStatus
	group                  bool
}

func currentEditState(b *text.Buffer) editState {
	return editState{b.String(), b.SelectionSet(), b.Version(), b.SelectionGeneration(), b.History(), b.InGroup()}
}

func requireEditState(t *testing.T, b *text.Buffer, before editState) {
	t.Helper()
	if after := currentEditState(b); !reflect.DeepEqual(before, after) {
		t.Fatalf("state changed: before %+v, after %+v", before, after)
	}
}

func TestMultiNativeCompositionRequiresFinal(t *testing.T) {
	for _, primary := range []text.Range{{Anchor: 5, Head: 8}, {Anchor: 8, Head: 5}} {
		name := "forward"
		if primary.Head < primary.Anchor {
			name = "reverse"
		}
		t.Run(name, func(t *testing.T) {
			const source = "🙂 / 中🙂 / xy"
			b := text.New(source)
			e := New(b)
			tester := multiTester(t, e)
			before := setSelections(t, b, []text.Range{{Anchor: 2, Head: 0}, primary, {Anchor: 12, Head: 13}}, 1)
			calls := 0
			var reported error
			e.OnChange = func() { calls++ }
			e.OnError = func(err error) { reported = err }
			tester.Compose("拼🙂", 2)
			mark, active := e.MarkedRange()
			token := e.compositionToken
			if b.String() != "🙂 / 拼🙂 / xy" || !active || mark != (ui.TextInputRange{Start: 5, End: 8}) || calls != 1 || !b.OwnsComposition(token) || b.CanUndo() {
				t.Fatalf("unexpected primary preedit %q, mark %+v/%t, callbacks %d", b.String(), mark, active, calls)
			}
			if err := e.Flush(); !errors.Is(err, text.ErrCompositionPending) || !b.OwnsComposition(token) {
				t.Fatalf("Flush settled a multi owner: %v", err)
			}
			e.ReplaceText(&mark, "program")
			if !errors.Is(reported, text.ErrCompositionPending) || b.String() != "🙂 / 拼🙂 / xy" || calls != 1 || !b.OwnsComposition(token) {
				t.Fatalf("program ReplaceText consumed a native owner: %v %q", reported, b.String())
			}
			tester.Type("終🙂")
			const result = "終🙂 / 終🙂 / x終🙂"
			after := text.SelectionSet{Ranges: []text.Range{{Anchor: 3, Head: 3}, {Anchor: 9, Head: 9}, {Anchor: 16, Head: 16}}, Primary: 1}
			if b.String() != result || calls != 2 || b.InGroup() || b.History().UndoDepth != 1 {
				t.Fatalf("native Type did not settle once: %q, callbacks %d, history %+v", b.String(), calls, b.History())
			}
			requireSelections(t, b, after)
			if !b.Undo() || b.String() != source || b.CanUndo() {
				t.Fatalf("one Undo = %q", b.String())
			}
			requireSelections(t, b, before)
			if !b.Redo() || b.String() != result {
				t.Fatalf("one Redo = %q", b.String())
			}
			requireSelections(t, b, after)
		})
	}
}

func TestMultiNativeCompositionExactMarkedRange(t *testing.T) {
	b := text.New("aa🙂bc")
	e := New(b)
	tester := multiTester(t, e)
	setSelections(t, b, []text.Range{{Anchor: 1, Head: 0}, {Anchor: 2, Head: 4}, {Anchor: 6, Head: 6}}, 1)
	var reported error
	changes := 0
	e.OnError = func(err error) { reported = err }
	e.OnChange = func() { changes++ }
	var native ui.TextInputClient = e.nativeClient
	wrongStart := ui.TextInputRange{Start: 0, End: 1}
	native.SetMarkedText(&wrongStart, "X", ui.TextInputRange{})
	if !errors.Is(reported, text.ErrRange) || changes != 0 || b.String() != "aa🙂bc" || !b.OwnsComposition(e.compositionToken) {
		t.Fatalf("invalid first replacement = %q, %v", b.String(), reported)
	}
	if _, marked := e.MarkedRange(); marked {
		t.Fatal("refused initial range became marked")
	}
	tester.Compose("拼🙂", 2)
	mark, active := native.MarkedRange()
	token := e.compositionToken
	if !active || mark != (ui.TextInputRange{Start: 2, End: 5}) || b.String() != "aa拼🙂bc" {
		t.Fatalf("live mark = %+v/%t, %q", mark, active, b.String())
	}
	before := currentEditState(b)
	for _, wrong := range []ui.TextInputRange{{Start: 2, End: 4}, {Start: 3, End: 5}, {Start: -1, End: 5}, {Start: 5, End: 2}, {Start: 2, End: 6}} {
		reported = nil
		native.ReplaceText(&wrong, "尾")
		if !errors.Is(reported, text.ErrRange) || !b.OwnsComposition(token) || changes != 1 {
			t.Fatalf("invalid final span %+v: %v, callbacks %d", wrong, reported, changes)
		}
		requireEditState(t, b, before)
		if got, active := native.MarkedRange(); got != mark || !active {
			t.Fatal("invalid final discarded the accepted mark")
		}
	}
	native.SetMarkedText(&wrongStart, "X", ui.TextInputRange{})
	requireEditState(t, b, before)
	native.UnmarkText()
	if !errors.Is(reported, text.ErrCompositionPending) || !b.OwnsComposition(token) {
		t.Fatalf("OS unmark before final did not remain pending: %v", reported)
	}
	native.ReplaceText(&mark, "尾")
	native.UnmarkText()
	if b.String() != "尾a尾bc尾" || b.InGroup() || changes != 2 || b.History().UndoDepth != 1 {
		t.Fatalf("valid explicit final = %q, %d callbacks, %+v", b.String(), changes, b.History())
	}
}

func TestMultiNativeCompositionSizeRefusalAndShortRetry(t *testing.T) {
	for _, firstRefused := range []bool{false, true} {
		t.Run(strconv.FormatBool(firstRefused), func(t *testing.T) {
			b := text.NewWithLimit("ab cd", 9)
			e := New(b)
			tester := multiTester(t, e)
			start := setSelections(t, b, []text.Range{{Anchor: 1, Head: 1}, {Anchor: 4, Head: 4}}, 0)
			changes := 0
			var reported error
			e.OnError = func(err error) { reported = err }
			e.OnChange = func() { changes++ }
			if firstRefused {
				tester.Compose("拼🙂", 2)
				if _, marked := e.MarkedRange(); marked || changes != 0 || !errors.Is(reported, text.ErrSizeLimit) || b.String() != "ab cd" {
					t.Fatal("refused first preedit modified the source")
				}
				requireSelections(t, b, start)
			} else {
				tester.Compose("x", 1)
				if b.String() != "axb cd" || changes != 1 {
					t.Fatal("primary-only preedit fixture")
				}
			}
			before, token, changeCount := currentEditState(b), e.compositionToken, changes
			tester.Type("你") // fits the primary replacement; the broadcast exceeds the cap
			if !errors.Is(reported, text.ErrSizeLimit) || !errors.Is(e.Flush(), text.ErrCompositionPending) || !b.OwnsComposition(token) || changes != changeCount {
				t.Fatalf("aggregate refusal = %v, %d callbacks", reported, changes)
			}
			requireEditState(t, b, before)
			tester.Type("xy")
			if b.String() != "axyb cxyd" || changes != changeCount+1 || b.InGroup() || !b.Undo() || b.String() != "ab cd" {
				t.Fatalf("short retry or Undo failed: %q, %d callbacks", b.String(), changes)
			}
			requireSelections(t, b, start)
			if !b.Redo() || b.String() != "axyb cxyd" {
				t.Fatal("short retry Redo lost the broadcast")
			}
			requireSelections(t, b, text.SelectionSet{Ranges: []text.Range{{Anchor: 3, Head: 3}, {Anchor: 8, Head: 8}}, Primary: 0})
		})
	}
}

func TestMultiNativeCompositionHistoryRefusalAndRetry(t *testing.T) {
	for _, tc := range []struct {
		name            string
		entries, bytes  int
		retry, expected string
	}{
		{"zero-entry", 0, 1024, "a", "a-a"},
		{"zero-byte", 10, 0, "a", "a-a"},
		{"tiny", 10, 2*(strconv.IntSize/4) + 4, "q", "q-q"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := text.New("a-a")
			b.SetHistoryLimits(tc.entries, tc.bytes)
			e := New(b)
			tester := multiTester(t, e)
			start := setSelections(t, b, []text.Range{{Anchor: 0, Head: 1}, {Anchor: 3, Head: 2}}, 1)
			changes := 0
			var reported error
			e.OnChange = func() { changes++ }
			e.OnError = func(err error) { reported = err }
			tester.Compose("p", 1)
			before, token := currentEditState(b), e.compositionToken
			tester.Type("zz")
			if !errors.Is(reported, text.ErrCompositionHistoryLimit) || !b.OwnsComposition(token) || changes != 1 || before.source != "a-p" {
				t.Fatalf("history refusal = %v, callbacks %d", reported, changes)
			}
			requireEditState(t, b, before)
			tester.Type(tc.retry)
			if b.String() != tc.expected || b.InGroup() || changes != 2 || e.Flush() != nil {
				t.Fatalf("retry after history refusal = %q, callbacks %d", b.String(), changes)
			}
			if tc.expected == "a-a" {
				if b.CanUndo() {
					t.Fatal("net-zero retry invented Undo")
				}
			} else {
				if !b.Undo() || b.String() != "a-a" || b.CanUndo() {
					t.Fatal("short broadcast missed its only complete Undo")
				}
				requireSelections(t, b, start)
				if !b.Redo() || b.String() != tc.expected {
					t.Fatal("short broadcast Redo failed")
				}
			}
			requireSelections(t, b, text.SelectionSet{Ranges: []text.Range{{Anchor: 1, Head: 1}, {Anchor: 3, Head: 3}}, Primary: 1})
		})
	}
}

func TestMultiNativeCompositionSameTextPreservesRedo(t *testing.T) {
	b := text.New("a-a")
	if err := b.ReplaceUTF16(3, 3, "!"); err != nil || !b.Undo() {
		t.Fatal("redo fixture", err)
	}
	e := New(b)
	tester := multiTester(t, e)
	setSelections(t, b, []text.Range{{Anchor: 1, Head: 0}, {Anchor: 3, Head: 2}}, 0)
	start, history := b.Version(), b.History()
	changes := 0
	e.OnChange = func() { changes++ }
	tester.Compose("a", 1)
	tester.Type("a")
	if b.Version() != start || b.History() != history || b.InGroup() || changes != 2 {
		t.Fatalf("same-text commit changed version/history or callback count: %+v, %d", b.History(), changes)
	}
	requireSelections(t, b, text.SelectionSet{Ranges: []text.Range{{Anchor: 1, Head: 1}, {Anchor: 3, Head: 3}}, Primary: 0})
	if !b.Redo() || b.String() != "a-a!" {
		t.Fatal("same-text confirmation destroyed redo")
	}
}

func TestNativeSingleProgramGroupAndRefusalRetry(t *testing.T) {
	b := text.NewWithLimit("ab", 5)
	e := New(b)
	tester := multiTester(t, e)
	if err := b.SetSelection(2, 1); err != nil {
		t.Fatal(err)
	}
	if err := b.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	e.ReplaceText(&ui.TextInputRange{Start: 1, End: 2}, "β")
	if err := b.SetSelection(0, 0); err != nil {
		t.Fatal(err)
	}
	if b.String() != "aβ" || !b.EndGroup() || !b.Undo() || b.String() != "ab" || b.Selection() != (text.Range{Anchor: 2, Head: 1}) {
		t.Fatal("legacy grouped program edit failed")
	}
	if err := b.SetSelection(2, 2); err != nil {
		t.Fatal(err)
	}
	tester.Compose(strings.Repeat("界", 2048), 2048)
	if !errors.Is(e.Flush(), text.ErrSizeLimit) || b.String() != "ab" || b.InGroup() {
		t.Fatal("legacy first refusal changed the buffer")
	}
	tester.Type("你") // retry without another Compose after a refused first preedit
	if b.String() != "ab你" || e.Flush() != nil || !b.Undo() || b.String() != "ab" {
		t.Fatalf("valid single Tester.Type retry failed: %q", b.String())
	}
}

// admissionState also observes the changed-line journal and redo availability;
// refusal checks intentionally exclude the editor's newly reported error.
type admissionRange struct {
	span text.Range
	valid bool
}

type admissionState struct {
	edit editState
	journal []int
	rangeJournal []admissionRange
	canUndo, canRedo bool
	mark ui.TextInputRange
	marked, composing, multi bool
	token text.CompositionToken
	stamp inputStamp
}

func currentAdmissionState(e *Editor) admissionState {
	b := e.Buffer
	journal := make([]int, b.Version()+2)
	var rangeJournal []admissionRange
	for version := range journal {
		journal[version] = b.ChangedLineSince(uint64(version))
		for anchor := 0; anchor <= b.LenUTF16(); anchor++ {
			for head := 0; head <= b.LenUTF16(); head++ {
				span, valid := b.MapRangeSince(uint64(version), text.Range{Anchor: anchor, Head: head})
				rangeJournal = append(rangeJournal, admissionRange{span, valid})
			}
		}
	}
	return admissionState{currentEditState(b), journal, rangeJournal, b.CanUndo(), b.CanRedo(), e.marked, e.compositionMarked, e.composing, e.compositionMulti, e.compositionToken, e.compositionStamp}
}

func requireAdmissionState(t *testing.T, e *Editor, want admissionState) {
	t.Helper()
	if got := currentAdmissionState(e); !reflect.DeepEqual(got, want) {
		t.Fatalf("refusal changed input state: before %+v, after %+v", want, got)
	}
	if !e.Buffer.OwnsComposition(want.token) {
		t.Fatal("refusal relinquished the original composition token")
	}
}

func requireInitialAdmissionRefusal(t *testing.T, e *Editor, before admissionState) {
	t.Helper()
	before.edit.group = true // Beginning the owner is the only permitted core change.
	requireEditState(t, e.Buffer, before.edit)
	after := currentAdmissionState(e)
	if !reflect.DeepEqual(before.journal, after.journal) || !reflect.DeepEqual(before.rangeJournal, after.rangeJournal) {
		t.Fatal("first refusal changed the line or range journal")
	}
}

func admissionRedoFixture(t *testing.T, b *text.Buffer) {
	t.Helper()
	end := b.LenUTF16()
	if err := b.ReplaceUTF16(end, end, "!"); err != nil || !b.Undo() || !b.CanRedo() {
		t.Fatal("redo fixture", err)
	}
}

func TestMultiAdmissionRejectsUnanchoredFinal(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(strconv.FormatBool(reverse), func(t *testing.T) {
			const source = "aa🙂bc"
			b := text.New(source)
			admissionRedoFixture(t, b)
			e := New(b)
			native := e.nativeTextClient()
			primary := text.Range{Anchor: 2, Head: 4}
			if reverse { primary.Anchor, primary.Head = primary.Head, primary.Anchor }
			start := setSelections(t, b, []text.Range{{Anchor: 1, Head: 0}, primary, {Anchor: 6, Head: 6}}, 1)
			baseline := currentAdmissionState(e)
			changes := 0
			var reported error
			e.OnChange = func() { changes++ }
			e.OnError = func(err error) { reported = err }
			wrong := ui.TextInputRange{Start: 0, End: 1}
			native.SetMarkedText(&wrong, "X", ui.TextInputRange{})
			if !errors.Is(reported, text.ErrRange) || changes != 0 || e.compositionToken == (text.CompositionToken{}) || !b.OwnsComposition(e.compositionToken) {
				t.Fatalf("first range refusal lost pending/token: %v, callbacks %d", reported, changes)
			}
			requireInitialAdmissionRefusal(t, e, baseline)
			if _, marked := native.MarkedRange(); marked { t.Fatal("refused range became marked") }
			before := currentAdmissionState(e)
			original := ui.TextInputRange{Start: 2, End: 4}
			for _, finalRange := range []*ui.TextInputRange{nil, &original, &wrong} {
				reported = nil
				native.ReplaceText(finalRange, "尾")
				if !errors.Is(reported, text.ErrRange) || changes != 0 {
					t.Fatalf("unanchored final broadcast %q: %v, callbacks %d", b.String(), reported, changes)
				}
				requireAdmissionState(t, e, before)
			}
			native.UnmarkText()
			if !errors.Is(e.Flush(), text.ErrCompositionPending) { t.Fatal("unanchored owner was settled") }
			requireAdmissionState(t, e, before)
			// Only a legal SetMarkedText can anchor this originally refused target.
			native.SetMarkedText(&original, "拼🙂", ui.TextInputRange{Start: 1, End: 1})
			mark, active := native.MarkedRange()
			if !active || mark != (ui.TextInputRange{Start: 2, End: 5}) || b.String() != "aa拼🙂bc" || changes != 1 || e.compositionToken != before.token {
				t.Fatalf("legal target retry failed: %q, %+v/%t, %v", b.String(), mark, active, reported)
			}
			accepted := currentAdmissionState(e)
			for _, invalid := range []ui.TextInputRange{original, {Start: 3, End: 5}, {Start: 5, End: 2}} {
				reported = nil
				native.ReplaceText(&invalid, "尾")
				if !errors.Is(reported, text.ErrRange) || changes != 1 { t.Fatal("accepted mark allowed a partial final", reported) }
				requireAdmissionState(t, e, accepted)
			}
			if reverse { native.ReplaceText(&mark, "尾") } else { native.ReplaceText(nil, "尾") }
			if b.String() != "尾a尾bc尾" || changes != 2 || b.InGroup() || b.History().UndoDepth != 1 || b.CanRedo() {
				t.Fatalf("anchored final = %q, callbacks %d, history %+v", b.String(), changes, b.History())
			}
			if !b.Undo() || b.String() != source || b.CanUndo() { t.Fatal("anchored final missed one-step Undo") }
			requireSelections(t, b, start)
			if !b.Redo() || b.String() != "尾a尾bc尾" { t.Fatal("anchored final missed Redo") }
		})
	}
}

func TestMultiAdmissionContentRefusalAllowsOriginalPrimaryFinal(t *testing.T) {
	for _, first := range []struct{name, value string; issue error}{
		{"size", "拼🙂", text.ErrSizeLimit}, {"utf8", "\xff", text.ErrUTF8},
	} {
		for _, explicit := range []bool{false, true} {
			t.Run(first.name+"/explicit-"+strconv.FormatBool(explicit), func(t *testing.T) {
				b := text.NewWithLimit("ab cd", 9)
				admissionRedoFixture(t, b)
				e := New(b)
				native := e.nativeTextClient()
				start := setSelections(t, b, []text.Range{{Anchor: 1, Head: 1}, {Anchor: 4, Head: 4}}, 0)
				baseline := currentAdmissionState(e)
				changes := 0
				var reported error
				e.OnChange = func(){ changes++ }
				e.OnError = func(err error){ reported = err }
				original := ui.TextInputRange{Start: 1, End: 1}
				native.SetMarkedText(&original, first.value, ui.TextInputRange{})
				if !errors.Is(reported, first.issue) || changes != 0 || !b.OwnsComposition(e.compositionToken) {
					t.Fatalf("first content refusal: %v", reported)
				}
				requireInitialAdmissionRefusal(t, e, baseline)
				before := currentAdmissionState(e)
				if before.marked { t.Fatal("refused content acquired a mark") }
				wrong := ui.TextInputRange{Start: 4, End: 4}
				native.SetMarkedText(&wrong, "x", ui.TextInputRange{})
				if !errors.Is(reported, text.ErrRange) { t.Fatal("wrong retry range was accepted") }
				requireAdmissionState(t, e, before)
				var finalRange *ui.TextInputRange
				if explicit { finalRange = &original }
				reported = nil
				native.ReplaceText(finalRange, "你") // aggregate bytes exceed nine.
				if !errors.Is(reported, text.ErrSizeLimit) || changes != 0 { t.Fatal("aggregate content refusal", reported) }
				requireAdmissionState(t, e, before)
				native.ReplaceText(finalRange, "xy")
				if b.String() != "axyb cxyd" || changes != 1 || b.InGroup() || b.History().UndoDepth != 1 || b.CanRedo() {
					t.Fatalf("short final after correct range refusal = %q, callbacks %d, %v", b.String(), changes, reported)
				}
				if !b.Undo() || b.String() != "ab cd" || b.CanUndo() { t.Fatal("short final missed one-step Undo") }
				requireSelections(t, b, start)
				if !b.Redo() || b.String() != "axyb cxyd" { t.Fatal("short final missed Redo") }
			})
		}
	}
}

func TestMultiAdmissionEmptyPreeditAndUnmarkRemainPending(t *testing.T) {
	b := text.New("a a")
	e := New(b)
	native := e.nativeTextClient()
	start := setSelections(t, b, []text.Range{{Anchor: 1, Head: 0}, {Anchor: 3, Head: 2}}, 1)
	native.SetMarkedText(nil, "", ui.TextInputRange{})
	mark, marked := native.MarkedRange()
	if !marked || mark != (ui.TextInputRange{Start: 2, End: 2}) || b.String() != "a " { t.Fatal("empty preedit was not accepted") }
	before := currentAdmissionState(e)
	native.UnmarkText()
	if !errors.Is(e.Flush(), text.ErrCompositionPending) { t.Fatal("Unmark inferred cancellation") }
	requireAdmissionState(t, e, before)
	native.ReplaceText(&mark, "尾")
	if b.String() != "尾 尾" || b.InGroup() || b.History().UndoDepth != 1 || !b.Undo() || b.String() != "a a" { t.Fatal("empty preedit lost its valid final or Undo") }
	requireSelections(t, b, start)
}

func TestMultiAdmissionHistoryRefusalKeepsTargetAndLiveSpan(t *testing.T) {
	for _, acceptedMark := range []bool{false, true} {
		t.Run(strconv.FormatBool(acceptedMark), func(t *testing.T) {
			b := text.NewWithLimit("ab cd", 9)
			admissionRedoFixture(t, b)
			// Full directed before/after sets plus two x insertions fit exactly.
			b.SetHistoryLimits(10, 2*(strconv.IntSize/4)+2)
			e := New(b)
			native := e.nativeTextClient()
			start := setSelections(t, b, []text.Range{{Anchor: 1, Head: 1}, {Anchor: 4, Head: 4}}, 0)
			changes := 0
			var reported error
			e.OnChange = func(){ changes++ }
			e.OnError = func(err error){ reported = err }
			original := ui.TextInputRange{Start: 1, End: 1}
			value := "拼🙂"
			if acceptedMark { value = "x" }
			native.SetMarkedText(&original, value, ui.TextInputRange{})
			before := currentAdmissionState(e)
			if !acceptedMark && !errors.Is(reported, text.ErrSizeLimit) { t.Fatal("missing initial size refusal") }
			span := original
			if acceptedMark { span = e.marked }
			reported = nil
			native.ReplaceText(&span, "xy")
			if !errors.Is(reported, text.ErrCompositionHistoryLimit) { t.Fatalf("history refusal became range refusal: %v", reported) }
			requireAdmissionState(t, e, before)
			if b.History().RedoDepth != 1 { t.Fatal("history refusal destroyed redo") }
			if acceptedMark {
				native.ReplaceText(&original, "x")
				if !errors.Is(reported, text.ErrRange) { t.Fatal("accepted mark fell back to the original caret") }
				requireAdmissionState(t, e, before)
			}
			native.ReplaceText(&span, "x")
			wantCalls := 1
			if acceptedMark { wantCalls++ }
			if b.String() != "axb cxd" || changes != wantCalls || b.InGroup() || b.History().UndoDepth != 1 { t.Fatalf("history retry failed: %q, %v", b.String(), reported) }
			if !b.Undo() || b.String() != "ab cd" || b.CanUndo() { t.Fatal("history retry missed complete Undo") }
			requireSelections(t, b, start)
			if !b.Redo() || b.String() != "axb cxd" { t.Fatal("history retry missed Redo") }
		})
	}
}
