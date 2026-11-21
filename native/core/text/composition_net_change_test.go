package text

import (
	"errors"
	"math/bits"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"
)

func netTestSet(t *testing.T, b *Buffer, set SelectionSet) SelectionSet {
	t.Helper()
	if err := b.SetSelectionSet(set); err != nil {
		t.Fatal(err)
	}
	return b.SelectionSet()
}
func netTestReplace(t *testing.T, b *Buffer, start, end int, value string) {
	t.Helper()
	if err := b.ReplaceUTF16(start, end, value); err != nil {
		t.Fatal(err)
	}
}
func netTestUnits(value string) int { return len(utf16.Encode([]rune(value))) }

func requireOneConfirmedAdvance(t *testing.T, before, after ChangeStamp) {
	t.Helper()
	if after.BaselineID != before.BaselineID || after.ConfirmedChangeID.epoch != before.ConfirmedChangeID.epoch || after.ConfirmedChangeID.generation != before.ConfirmedChangeID.generation+1 {
		t.Fatal("settled transaction/replay must advance confirmed identity exactly once", before, after)
	}
}

func TestIntegratedExplicitBatchConfirmedStampAndHistoryIsolation(t *testing.T) {
	b := New("a🙂|b🙂")
	before := SelectionSet{[]Range{{3, 0}, {7, 4}}, 1}
	mustSetSelections(t, b, before.Ranges, before.Primary)
	wantBefore := SelectionSet{append([]Range(nil), before.Ranges...), before.Primary}
	after := SelectionSet{[]Range{{3, 0}, {5, 4}}, 1}
	wantAfter := SelectionSet{append([]Range(nil), after.Ranges...), after.Primary}
	initial := b.ChangeStamp()
	changed, err := b.ReplaceBatchWithSelectionUTF16(b.Version(), b.SelectionGeneration(), []Replacement{{0, 3, "甲🙂"}, {4, 7, "乙"}}, after)
	if err != nil || !changed || b.String() != "甲🙂|乙" || b.History().UndoDepth != 1 || !b.undo[0].confirmedChange {
		t.Fatal("explicit-after batch did not reach central confirmed commit", changed, err, b.String())
	}
	requireOneConfirmedAdvance(t, initial, b.ChangeStamp())
	requireSelections(t, b, wantAfter)
	retained := b.History().RetainedBytes
	// Mutate both callers' input slices and independent read results, then
	// replay twice: no live or historical endpoint may share their storage.
	before.Ranges[0], after.Ranges[1] = Range{}, Range{}
	read := b.SelectionSet()
	read.Ranges[0] = Range{}
	for i := 0; i < 2; i++ {
		stamp := b.ChangeStamp()
		if !b.Undo() || b.String() != "a🙂|b🙂" || b.History().RetainedBytes != retained {
			t.Fatal("explicit-after whole undo or retained cost")
		}
		requireOneConfirmedAdvance(t, stamp, b.ChangeStamp())
		requireSelections(t, b, wantBefore)
		stamp = b.ChangeStamp()
		if !b.Redo() || b.String() != "甲🙂|乙" || b.History().RetainedBytes != retained {
			t.Fatal("explicit-after whole redo or retained cost")
		}
		requireOneConfirmedAdvance(t, stamp, b.ChangeStamp())
		requireSelections(t, b, wantAfter)
	}
}

func TestIntegratedExplicitBatchAndLineSelectionOnlyPreserveStamp(t *testing.T) {
	for _, line := range []bool{false, true} {
		b := New("abcd")
		if line {
			b = New("a\na")
		}
		source := b.String()
		netTestReplace(t, b, b.LenUTF16(), b.LenUTF16(), "!")
		if !b.Undo() {
			t.Fatal("redo fixture")
		}
		before := mustSetSelections(t, b, []Range{{1, 0}}, 0)
		edits := []Replacement{{0, 2, "a"}, {2, 4, "bcd"}}
		after := SelectionSet{[]Range{{0, 0}, {4, 2}}, 1}
		stamp, version, generation, history := b.ChangeStamp(), b.Version(), b.SelectionGeneration(), b.History()
		journal, lines := b.rangeChanges, b.changedLines
		if line {
			unchanged := multiStateGuard(t, b)
			plan, err := b.PlanLineEditSet(MoveLinesDown, before)
			if err != nil || plan.Changed || len(plan.Edits) != 1 {
				t.Fatal("identical line move fixture", plan, err)
			}
			unchanged()
			edits, after = plan.Edits, plan.After
		}
		changed, err := b.ReplaceBatchWithSelectionUTF16(version, generation, edits, after)
		if err != nil || changed || b.String() != source || b.ChangeStamp() != stamp || b.Version() != version || b.History() != history || b.rangeChanges != journal || b.changedLines != lines || !b.CanRedo() || b.SelectionGeneration() != generation+1 {
			t.Fatal("selection-only result changed settled bytes/stamp/history", line, changed, err)
		}
		requireSelections(t, b, after)
		unchanged := multiStateGuard(t, b)
		if changed, err = b.ReplaceBatchWithSelectionUTF16(b.Version(), b.SelectionGeneration(), edits, after); err != nil || changed {
			t.Fatal("exact selection no-op", changed, err)
		}
		unchanged()
		if !b.Redo() || b.String() != source+"!" {
			t.Fatal("selection-only batch lost original redo")
		}
		requireOneConfirmedAdvance(t, stamp, b.ChangeStamp())
	}
}

func TestIntegratedExplicitBatchTrimKeepsOneConfirmedAdvance(t *testing.T) {
	for _, limits := range [][2]int{{0, 0}, {0, 1 << 20}, {256, 1}, {256, 1 << 20}} {
		b := New("a|b")
		set := mustSetSelections(t, b, []Range{{1, 0}, {3, 2}}, 1)
		b.SetHistoryLimits(limits[0], limits[1])
		before := b.ChangeStamp()
		changed, err := b.ReplaceBatchWithSelectionUTF16(b.Version(), b.SelectionGeneration(), []Replacement{{0, 1, "A"}, {2, 3, "B"}}, set)
		if err != nil || !changed || b.String() != "A|B" {
			t.Fatal("ordinary explicit batch refused normal trim", limits, changed, err)
		}
		requireOneConfirmedAdvance(t, before, b.ChangeStamp())
		keep := limits[0] > 0 && limits[1] > 1
		if b.CanUndo() != keep || b.History().Truncated == keep {
			t.Fatal("ordinary history policy changed", limits, b.History())
		}
		stamp := b.ChangeStamp()
		b.SetHistoryLimits(0, 0)
		b.ResetHistory()
		if b.ChangeStamp() != stamp || b.CanUndo() || b.CanRedo() {
			t.Fatal("idle reset/trim fabricated another confirmed action")
		}
	}
}

func TestIntegratedLineCommandsAdvanceStampOncePerActionAndReplay(t *testing.T) {
	for action := MoveLinesUp; action <= OutdentLines; action++ {
		b := New("a\nb\nc\nd\ne")
		ranges := []Range{{3, 2}, {7, 6}}
		if action == OutdentLines {
			b = New("a\n b\nc\n\td\ne")
			ranges = []Range{{4, 3}, {9, 8}}
		}
		before := mustSetSelections(t, b, ranges, 1)
		source, initial := b.String(), b.ChangeStamp()
		unchanged := multiStateGuard(t, b)
		plan, err := b.PlanLineEditSet(action, before)
		if err != nil || !plan.Changed || len(plan.Edits) != 2 {
			t.Fatal("changing multi-line fixture", action, plan, err)
		}
		unchanged()
		changed, err := b.ReplaceBatchWithSelectionUTF16(b.Version(), b.SelectionGeneration(), plan.Edits, plan.After)
		if err != nil || !changed || b.History().UndoDepth != 1 || !b.undo[0].confirmedChange {
			t.Fatal("line action did not reach one ordinary transaction", action, changed, err)
		}
		requireOneConfirmedAdvance(t, initial, b.ChangeStamp())
		output, committed := b.String(), b.ChangeStamp()
		requireSelections(t, b, plan.After)
		if !b.Undo() || b.String() != source {
			t.Fatal("line whole undo", action)
		}
		requireOneConfirmedAdvance(t, committed, b.ChangeStamp())
		requireSelections(t, b, before)
		undone := b.ChangeStamp()
		if !b.Redo() || b.String() != output {
			t.Fatal("line whole redo", action)
		}
		requireOneConfirmedAdvance(t, undone, b.ChangeStamp())
		requireSelections(t, b, plan.After)
	}
}

func TestChangeStampConstructorsPurityAndConfirmedRollover(t *testing.T) {
	a, b := New("a"), NewWithLimit("a", 10)
	initial := a.ChangeStamp()
	if initial.BaselineID == b.ChangeStamp().BaselineID || initial.ConfirmedChangeID == b.ChangeStamp().ConfirmedChangeID || initial == (ChangeStamp{}) {
		t.Fatal("constructors reused identity epochs")
	}
	var observed ChangeStamp
	if allocs := testing.AllocsPerRun(100, func() { observed = a.ChangeStamp() }); allocs != 0 || observed != initial || a.Version() != 0 || a.InGroup() || a.History().UndoDepth != 0 {
		t.Fatal("stamp accessor is not pure", allocs, observed)
	}
	a.changeStamp.ConfirmedChangeID.generation = ^uint64(0) - 1
	before := a.ChangeStamp()
	netTestReplace(t, a, 0, 1, "b")
	maximum := a.ChangeStamp()
	if maximum.BaselineID != before.BaselineID || maximum.ConfirmedChangeID.epoch != before.ConfirmedChangeID.epoch || maximum.ConfirmedChangeID.generation != ^uint64(0) {
		t.Fatal("counter did not advance to maximum")
	}
	netTestReplace(t, a, 0, 1, "a")
	rolled := a.ChangeStamp()
	if rolled.BaselineID != initial.BaselineID || rolled.ConfirmedChangeID.epoch == maximum.ConfirmedChangeID.epoch || rolled.ConfirmedChangeID.generation != 0 || rolled.ConfirmedChangeID == initial.ConfirmedChangeID {
		t.Fatal("rollover aliased an earlier token or invalidated baseline")
	}
	if !a.Undo() || a.ChangeStamp().ConfirmedChangeID.generation != 1 || a.ChangeStamp().BaselineID != initial.BaselineID {
		t.Fatal("replay after rollover")
	}
	// Held epochs must never be recycled through reset/restore or same bytes.
	held := []ChangeStamp{initial, before, maximum, rolled}
	for i := 0; i < 100; i++ {
		old := a.ChangeStamp()
		if err := a.BeginGroup(); err != nil {
			t.Fatal(err)
		}
		a.ResetHistory()
		next := a.ChangeStamp()
		if next.BaselineID == old.BaselineID || next.ConfirmedChangeID != old.ConfirmedChangeID {
			t.Fatal("active reset identity")
		}
		for _, prior := range held {
			if next.BaselineID == prior.BaselineID {
				t.Fatal("baseline ABA")
			}
		}
		held = append(held, next)
	}
}

func TestChangeStampActiveResetAndRestoreInvalidateOnlyBaseline(t *testing.T) {
	for _, kind := range []string{"ordinary", "single", "multi", "empty"} {
		t.Run(kind, func(t *testing.T) {
			b := New("ab")
			if kind == "multi" {
				netTestSet(t, b, SelectionSet{[]Range{{0, 1}, {2, 2}}, 0})
			}
			before := b.ChangeStamp()
			var token CompositionToken
			var err error
			if kind == "single" {
				token, err = b.BeginCompositionGroup()
			} else if kind == "multi" {
				token, err = b.BeginCompositionSetGroup()
			} else {
				err = b.BeginGroup()
			}
			if err != nil {
				t.Fatal(err)
			}
			if kind == "single" || kind == "multi" {
				err = b.ReplaceCompositionUTF16(token, 0, 1, "x")
			} else if kind != "empty" {
				err = b.ReplaceUTF16(0, 1, "x")
			}
			if err != nil || b.ChangeStamp() != before {
				t.Fatal("provisional bytes advanced confirmed stamp", err)
			}
			value, version := b.String(), b.Version()
			b.ResetHistory()
			after := b.ChangeStamp()
			if after.BaselineID == before.BaselineID || after.ConfirmedChangeID != before.ConfirmedChangeID || b.String() != value || b.Version() != version || b.InGroup() || b.CanUndo() || b.CanRedo() {
				t.Fatal("reset did not separately invalidate baseline")
			}
			if kind != "empty" && b.String() != "xb" {
				t.Fatal("reset must retain provisional xb")
			}
			if token != (CompositionToken{}) && (!errors.Is(b.CancelCompositionGroup(token), ErrComposition) || b.OwnsComposition(token)) {
				t.Fatal("stale cancellation was proved")
			}
			b.ResetHistory()
			b.SetHistoryLimits(0, 0)
			if b.ChangeStamp() != after {
				t.Fatal("idle reset/trim changed stamp")
			}
			b.Restore(New("ab").Snapshot())
			restored := b.ChangeStamp()
			if restored.BaselineID == after.BaselineID || restored.ConfirmedChangeID != after.ConfirmedChangeID || b.String() != "ab" {
				t.Fatal("restore invented confirmed edit")
			}
			b.Restore(New("ab").Snapshot())
			if b.ChangeStamp().BaselineID == restored.BaselineID || b.ChangeStamp().ConfirmedChangeID != restored.ConfirmedChangeID {
				t.Fatal("same-byte restore did not rebase")
			}
		})
	}
	b := New("ab")
	before := b.ChangeStamp()
	token, _ := b.BeginCompositionGroup()
	if err := b.ReplaceCompositionUTF16(token, 0, 1, "x"); err != nil {
		t.Fatal(err)
	}
	b.Restore(New("ab").Snapshot())
	if b.ChangeStamp().BaselineID == before.BaselineID || b.ChangeStamp().ConfirmedChangeID != before.ConfirmedChangeID || b.OwnsComposition(token) {
		t.Fatal("active restore identity")
	}
}

func TestConfirmedNetZeroIrregularHistoryAndReplayRemainLegacy(t *testing.T) {
	for _, composition := range []bool{false, true} {
		b := New("ab")
		netTestReplace(t, b, 2, 2, "!")
		if !b.Undo() {
			t.Fatal("redo fixture")
		}
		if err := b.SetSelection(2, 0); err != nil {
			t.Fatal(err)
		}
		before, history := b.ChangeStamp(), b.History()
		var token CompositionToken
		var err error
		if composition {
			token, err = b.BeginCompositionGroup()
		} else {
			err = b.BeginGroup()
		}
		if err != nil {
			t.Fatal(err)
		}
		for _, step := range []struct {
			start int
			next  string
		}{{0, "x"}, {1, "y"}, {0, "a"}, {1, "b"}} {
			if composition {
				err = b.ReplaceCompositionUTF16(token, step.start, step.start+1, step.next)
			} else {
				err = b.ReplaceUTF16(step.start, step.start+1, step.next)
			}
			if err != nil || b.ChangeStamp() != before {
				t.Fatal("preedit changed confirmed stamp", err)
			}
		}
		if len(b.group.edits) != 4 {
			t.Fatal("irregular history was coalesced")
		}
		if composition {
			err = b.EndCompositionGroup(token)
		} else if !b.EndGroup() {
			t.Fatal("nonempty EndGroup must return true")
		}
		if err != nil || b.String() != "ab" || b.ChangeStamp() != before || b.History().UndoDepth != 1 || b.History().RedoDepth != 0 || b.History().RetainedBytes != 8 || history.RedoDepth != 1 || b.undo[0].confirmedChange {
			t.Fatal("net-zero changed old history/redo/accounting", err, b.History())
		}
		if !b.Undo() || b.String() != "ab" || b.Selection() != (Range{2, 0}) || b.ChangeStamp() != before {
			t.Fatal("net-zero undo advanced identity")
		}
		if !b.Redo() || b.String() != "ab" || b.Selection() != (Range{2, 2}) || b.ChangeStamp() != before {
			t.Fatal("net-zero redo advanced identity")
		}
	}
}

func TestConfirmedChangesOrdinarySingleMultiPatchAndBatch(t *testing.T) {
	for _, kind := range []string{"ordinary", "group", "single", "multi", "patch", "batch"} {
		t.Run(kind, func(t *testing.T) {
			b := New("a|b")
			beforeSet := netTestSet(t, b, SelectionSet{[]Range{{1, 0}, {3, 2}}, 1})
			before := b.ChangeStamp()
			var err error
			switch kind {
			case "ordinary":
				err = b.ReplaceUTF16(0, 1, "A")
			case "group":
				err = b.BeginGroup()
				if err == nil {
					err = b.ReplaceUTF16(0, 1, "A")
				}
				if err == nil {
					err = b.ReplaceUTF16(2, 3, "B")
				}
				if b.ChangeStamp() != before {
					t.Fatal("ordinary group published early")
				}
				if err == nil && !b.EndGroup() {
					t.Fatal("group end")
				}
			case "single":
				var token CompositionToken
				token, err = b.BeginCompositionGroup()
				if err == nil {
					err = b.ReplaceCompositionUTF16(token, 2, 3, "B")
				}
				if b.ChangeStamp() != before {
					t.Fatal("single preview published early")
				}
				if err == nil {
					err = b.EndCompositionGroup(token)
				}
			case "multi":
				var token CompositionToken
				token, err = b.BeginCompositionSetGroup()
				if err == nil {
					err = b.ReplaceCompositionUTF16(token, 2, 3, "预🙂")
				}
				if b.ChangeStamp() != before {
					t.Fatal("multi preview published early")
				}
				if err == nil {
					err = b.CommitCompositionSetUTF16(token, "X")
				}
			case "patch":
				var changed bool
				changed, err = b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), []PatchEdit{{0, 1, "a", "A"}, {2, 3, "b", "B"}}, SelectionSet{[]Range{{1, 0}, {3, 2}}, 1})
				if err == nil && !changed {
					t.Fatal("patch no change")
				}
			case "batch":
				err = b.ReplaceBatchUTF16(b.Version(), b.SelectionGeneration(), []Replacement{{0, 1, "A"}, {2, 3, "B"}}, 1)
			}
			if err != nil {
				t.Fatal(err)
			}
			after, afterSet := b.ChangeStamp(), b.SelectionSet()
			if after.BaselineID != before.BaselineID || after.ConfirmedChangeID.epoch != before.ConfirmedChangeID.epoch || after.ConfirmedChangeID.generation != before.ConfirmedChangeID.generation+1 || b.History().UndoDepth != 1 || !b.undo[0].confirmedChange {
				t.Fatal("commit count/metadata", after, before)
			}
			value := b.String()
			if !b.Undo() || b.String() != "a|b" || !reflect.DeepEqual(b.SelectionSet(), beforeSet) {
				t.Fatal("whole undo")
			}
			undone := b.ChangeStamp()
			if undone.BaselineID != before.BaselineID || undone.ConfirmedChangeID.generation != after.ConfirmedChangeID.generation+1 {
				t.Fatal("undo did not advance exactly once")
			}
			if !b.Redo() || b.String() != value || !reflect.DeepEqual(b.SelectionSet(), afterSet) || b.ChangeStamp().ConfirmedChangeID.generation != undone.ConfirmedChangeID.generation+1 || b.ChangeStamp().BaselineID != before.BaselineID {
				t.Fatal("whole redo stamp/selections")
			}
		})
	}
}

func TestConfirmedNoOpCancelRefusalAndZeroHistory(t *testing.T) {
	for _, kind := range []string{"same", "cancel", "single-zero", "multi-zero", "patch-zero", "batch-zero"} {
		t.Run(kind, func(t *testing.T) {
			b := New("aaaa")
			netTestReplace(t, b, 4, 4, "!")
			if !b.Undo() {
				t.Fatal("redo fixture")
			}
			before, history := b.ChangeStamp(), b.History()
			var err error
			switch kind {
			case "same":
				err = b.ReplaceUTF16(0, 1, "a")
			case "cancel", "single-zero":
				var token CompositionToken
				token, err = b.BeginCompositionGroup()
				if err == nil {
					err = b.ReplaceCompositionUTF16(token, 0, 1, "🙂")
				}
				if err == nil && kind == "single-zero" {
					err = b.ReplaceCompositionUTF16(token, 0, 2, "a")
				}
				if err == nil {
					if kind == "cancel" {
						err = b.CancelCompositionGroup(token)
					} else {
						err = b.EndCompositionGroup(token)
					}
				}
			case "multi-zero":
				netTestSet(t, b, SelectionSet{[]Range{{1, 0}, {4, 1}}, 0})
				var token CompositionToken
				token, err = b.BeginCompositionSetGroup()
				if err == nil {
					err = b.ReplaceCompositionUTF16(token, 0, 1, "预")
				}
				if err == nil {
					err = b.CommitCompositionSetUTF16(token, "aa")
				}
			case "patch-zero":
				var changed bool
				changed, err = b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), []PatchEdit{{0, 1, "a", ""}, {1, 4, "aaa", "aaaa"}}, b.SelectionSet())
				if changed {
					t.Fatal("net-zero patch changed")
				}
			case "batch-zero":
				err = b.ReplaceBatchUTF16(b.Version(), b.SelectionGeneration(), []Replacement{{0, 1, ""}, {1, 4, "aaaa"}}, 1)
			}
			if err != nil || b.String() != "aaaa" || b.ChangeStamp() != before || b.History() != history || !b.CanRedo() {
				t.Fatal("no-op/cancel changed confirmed state", kind, err, b.History())
			}
		})
	}
	b := NewWithLimit("a|b", 8)
	netTestSet(t, b, SelectionSet{[]Range{{0, 1}, {2, 3}}, 0})
	token, err := b.BeginCompositionSetGroup()
	if err != nil {
		t.Fatal(err)
	}
	before := b.ChangeStamp()
	if !errors.Is(b.EndCompositionGroup(token), ErrCompositionPending) {
		t.Fatal("pending final")
	}
	if !errors.Is(b.CommitCompositionSetUTF16(token, strings.Repeat("x", 20)), ErrSizeLimit) || b.ChangeStamp() != before || !b.OwnsComposition(token) {
		t.Fatal("refusal advanced identity/retired owner")
	}
	b.SetHistoryLimits(0, 0)
	if !errors.Is(b.CommitCompositionSetUTF16(token, "X"), ErrCompositionHistoryLimit) || b.ChangeStamp() != before {
		t.Fatal("history refusal advanced identity")
	}
	if b.CancelCompositionGroup(token) != nil || b.ChangeStamp() != before {
		t.Fatal("cancel advanced identity")
	}
	for _, limits := range [][2]int{{0, 0}, {1, 1}, {1, 2}} {
		b = New("a")
		b.SetHistoryLimits(limits[0], limits[1])
		before = b.ChangeStamp()
		netTestReplace(t, b, 0, 1, "b")
		if b.ChangeStamp().ConfirmedChangeID.generation != before.ConfirmedChangeID.generation+1 || b.ChangeStamp().BaselineID != before.BaselineID {
			t.Fatal("trim lost confirmed edit")
		}
		if limits[1] < 2 && (b.CanUndo() || !b.History().Truncated) {
			t.Fatal("old trim policy changed")
		}
		stamp := b.ChangeStamp()
		b.SetHistoryLimits(0, 0)
		if b.ChangeStamp() != stamp {
			t.Fatal("trim advanced stamp")
		}
	}
}

// Independent test-only sequential replacement oracle. It does not call the
// production UTF16 mapper, inverse sequence, classifier or history coalescer.
func netOracleEdit(t *testing.T, source string, start, end int, value string) (string, string) {
	t.Helper()
	positions := map[int]int{0: 0}
	units := 0
	for at, r := range source {
		positions[units] = at
		units += len(utf16.Encode([]rune{r}))
	}
	positions[units] = len(source)
	left, lok := positions[start]
	right, rok := positions[end]
	if !lok || !rok || left > right {
		t.Fatal("invalid independent fixture", start, end)
	}
	return source[:left] + value + source[right:], source[left:right]
}

func netAssertTree(t *testing.T, root *netNode, current string) string {
	t.Helper()
	seen := map[*netNode]bool{}
	var inspect func(*netNode) (int, int, int)
	inspect = func(n *netNode) (int, int, int) {
		if n == nil {
			return 0, 0, 0
		}
		if seen[n] {
			t.Fatal("cycle/aliased virtual node")
		}
		seen[n] = true
		lu, lb, lh := inspect(n.left)
		ru, rb, rh := inspect(n.right)
		f := n.fragment
		value := current
		if f.old != nil {
			value = f.old.value
		}
		if f.byteStart < 0 || f.byteEnd > len(value) || f.byteStart >= f.byteEnd {
			t.Fatal("bad fragment byte bounds")
		}
		text := value[f.byteStart:f.byteEnd]
		own := netTestUnits(text)
		if !utf8.ValidString(text) || !utf8.ValidString(value[:f.byteStart]) || own != f.units || netTestUnits(value[:f.byteStart]) != f.unitStart {
			t.Fatal("fragment byte/UTF16 disagreement")
		}
		if n.units != lu+own+ru || n.bytes != lb+len(text)+rb || n.height != 1+max(lh, rh) || lh-rh > 1 || rh-lh > 1 {
			t.Fatal("AVL/weight invariant", n.units, n.bytes, n.height, lh, rh)
		}
		return n.units, n.bytes, n.height
	}
	inspect(root)
	it := netFragmentIterator{node: root}
	last := map[*netOldText]int{}
	var out strings.Builder
	for {
		f, ok := it.next()
		if !ok {
			break
		}
		if prior, seen := last[f.old]; seen && f.byteStart < prior {
			t.Fatal("occurrence fragments reordered/duplicated")
		}
		last[f.old] = f.byteEnd
		value := current
		if f.old != nil {
			value = f.old.value
		}
		out.WriteString(value[f.byteStart:f.byteEnd])
	}
	return out.String()
}

func TestNetClassifierIndependentSequentialUnicodeOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(87020261120))
	words := []string{"", "a", "中", "🙂", "e\u0301", "👨‍👩‍👧‍👦", "\r\n", "🇨🇳"}
	for trial := 0; trial < 240; trial++ {
		seed := "L🙂中e\u0301\r\nR"
		value := seed
		states := []string{seed}
		var edits []change
		for step := 0; step < 20+trial%31; step++ {
			runes := []rune(value)
			left, right := rng.Intn(len(runes)+1), rng.Intn(len(runes)+1)
			if left > right {
				left, right = right, left
			}
			start, end := netTestUnits(string(runes[:left])), netTestUnits(string(runes[:right]))
			next := words[rng.Intn(len(words))]
			var old string
			value, old = netOracleEdit(t, value, start, end, next)
			edits = append(edits, change{start, old, next})
			states = append(states, value)
		}
		// In alternating trials append exact inverses of every operation. This
		// creates irregular, overlapping net-zero logs with changing lengths.
		if trial%2 == 0 {
			for i := len(edits) - 1; i >= 0; i-- {
				c := edits[i]
				var removed string
				value, removed = netOracleEdit(t, value, c.start, c.start+netTestUnits(c.next), c.old)
				edits = append(edits, change{c.start, removed, c.old})
				states = append(states, value)
			}
		}
		b := New(value)
		before, version := b.ChangeStamp(), b.Version()
		var work netChangeWork
		if got := b.transactionChangesBytes(transaction{edits: edits}, &work); got != (seed != value) {
			t.Fatal("exact byte oracle", trial, got, seed, value)
		}
		ctx := netChangeContext{buffer: b}
		if reconstructed := netAssertTree(t, ctx.inverse(transaction{edits: edits}), value); reconstructed != seed {
			t.Fatal("inverse stream oracle", trial, reconstructed, seed)
		}
		// Check every reconstructed intermediate against the independently
		// retained forward state, including tree balance and occurrence order.
		root := ctx.node(netFragment{byteEnd: len(value), units: netTestUnits(value)})
		for i := len(edits) - 1; i >= 0; i-- {
			edit := edits[i]
			left, rest := ctx.split(root, edit.start)
			_, right := ctx.split(rest, netTestUnits(edit.next))
			var old *netNode
			if edit.old != "" {
				x := indexNetOld(edit.old, nil)
				old = ctx.node(netFragment{old: x, byteEnd: len(edit.old), units: netTestUnits(edit.old)})
			}
			root = ctx.join(ctx.join(left, old), right)
			if got := netAssertTree(t, root, value); got != states[i] {
				t.Fatal("intermediate inverse oracle", trial, i, got, states[i])
			}
		}
		if b.ChangeStamp() != before || b.Version() != version || b.String() != value {
			t.Fatal("classification mutated buffer")
		}
		if work.nodes > 3*len(edits)+1 || work.oldScanBytes > work.oldLookups*chunkBytes || work.currentLookups > 2*len(edits) {
			t.Fatal("input-bounded allocation/index work", trial, work)
		}
	}
	b := New("x")
	var work netChangeWork
	if b.transactionChangesBytes(transaction{}, &work) || !b.transactionChangesBytes(transaction{edits: []change{{0, "a", "x"}}}, &work) || b.transactionChangesBytes(transaction{edits: []change{{0, "x", "x"}}}, &work) || work != (netChangeWork{}) {
		t.Fatal("empty/single exact fast path")
	}
}

func TestNetOldIndexBoundsSurrogatesAndNonmonotoneQueries(t *testing.T) {
	value := strings.Repeat("a", chunkBytes-1) + "🙂" + strings.Repeat("中e\u0301🇨🇳", 3000)
	var work netChangeWork
	x := indexNetOld(value, &work)
	if len(x.points) < 2 || x.points[1] != (netCheckpoint{chunkBytes - 1, chunkBytes - 1}) || work.indexBuildBytes != len(value) {
		t.Fatal("checkpoint must precede crossing rune", x.points[:min(2, len(x.points))])
	}
	previous := 0
	for _, p := range x.points {
		if p.bytes-previous > chunkBytes || netTestUnits(value[:p.bytes]) != p.units {
			t.Fatal("checkpoint boundary")
		}
		previous = p.bytes
	}
	if len(value)-previous > chunkBytes {
		t.Fatal("unbounded final checkpoint window")
	}
	units := 0
	positions, bytesAt := []int{0}, []int{0}
	for at, r := range value {
		if at != 0 {
			positions = append(positions, units)
			bytesAt = append(bytesAt, at)
		}
		units += len(utf16.Encode([]rune{r}))
	}
	positions, bytesAt = append(positions, units), append(bytesAt, len(value))
	rng := rand.New(rand.NewSource(874096))
	for i := 0; i < 1200; i++ {
		at := rng.Intn(len(positions))
		before := work.oldScanBytes
		if got := x.byteOffset(positions[at], &work); got != bytesAt[at] || work.oldScanBytes-before > chunkBytes {
			t.Fatal("indexed old lookup", at, got, bytesAt[at])
		}
	}
	netWantPanic(t, func() { x.byteOffset(chunkBytes, nil) }) // middle of the emoji pair
	netWantPanic(t, func() { x.byteOffset(-1, nil) })
	netWantPanic(t, func() { netCheckedSum(int(^uint(0)>>1), 1) })
	b := New("🙂")
	ctx := netChangeContext{buffer: b}
	root := ctx.node(netFragment{byteEnd: 4, units: 2})
	netWantPanic(t, func() { ctx.split(root, 1) })
}

func netWantPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("internal invariant failure was silently ignored")
		}
	}()
	fn()
}

func TestNetAVLDestructiveSplitJoinAndFragmentOrder(t *testing.T) {
	b := New("")
	ctx := netChangeContext{buffer: b, work: &netChangeWork{}}
	var root *netNode
	value := ""
	rng := rand.New(rand.NewSource(87800))
	for i := 0; i < 800; i++ {
		at := rng.Intn(len(value) + 1)
		x := indexNetOld(string(rune('a'+i%26)), nil)
		left, right := ctx.split(root, at)
		root = ctx.join(ctx.join(left, ctx.node(netFragment{old: x, byteEnd: 1, units: 1})), right)
		value = value[:at] + x.value + value[at:]
		if i%17 == 0 && netAssertTree(t, root, "") != value {
			t.Fatal("destructive insertion order")
		}
	}
	if netHeight(root) > 2*bits.Len(uint(len(value)+1)) || ctx.work.nodes > 3*800+1 {
		t.Fatal("unbalanced/path-copy tree", ctx.work, netHeight(root))
	}
	for len(value) > 0 {
		at := rng.Intn(len(value))
		left, rest := ctx.split(root, at)
		_, right := ctx.split(rest, 1)
		root = ctx.join(left, right)
		value = value[:at] + value[at+1:]
		if len(value)%19 == 0 && netAssertTree(t, root, "") != value {
			t.Fatal("destructive deletion order")
		}
	}
	if root != nil {
		t.Fatal("empty AVL retained fragments")
	}
}

func TestNetLargeOldOccurrenceSplitsReuseSparseIndex(t *testing.T) {
	seed := strings.Repeat("a", 80<<10)
	value := seed
	var edits []change
	for i := 0; i < 600; i++ {
		at := (i * 7907) % len(seed)
		var old string
		value, old = netOracleEdit(t, value, at, at+1, "x")
		edits = append(edits, change{at, old, "x"})
	}
	edits = append(edits, change{0, value, "z"})
	b := New("z")
	log := transaction{edits: edits}
	var work netChangeWork
	if !b.transactionChangesBytes(log, &work) || work.indexBuildBytes != len(seed)+600 || work.oldScanBytes > work.oldLookups*chunkBytes || work.oldLookups > 2*len(edits) || work.nodes > 3*len(edits)+1 {
		t.Fatal("large old source was reindexed/prefix-scanned", work)
	}
	// These old entries are deliberately not valid replay coordinates. The
	// classifier must not inspect or classify prior transactions at all.
	b.undo = make([]transaction, 1000)
	for i := range b.undo {
		b.undo[i] = transaction{edits: []change{{-1, "older", "history"}}}
	}
	var withOlder netChangeWork
	if !b.transactionChangesBytes(log, &withOlder) || withOlder != work {
		t.Fatal("classification scanned older history", work, withOlder)
	}
	c := netChangeContext{buffer: b}
	it := netFragmentIterator{node: c.inverse(log)}
	var out strings.Builder
	for {
		f, ok := it.next()
		if !ok {
			break
		}
		if f.old == nil {
			t.Fatal("final z unexpectedly survives inverse")
		}
		out.WriteString(f.old.value[f.byteStart:f.byteEnd])
	}
	if out.String() != seed {
		t.Fatal("large indexed occurrence inverse")
	}
}

func TestNetSharedCurrentSpansAndAdversarialRetainedEditsWork(t *testing.T) {
	seed := strings.Repeat("a", 256<<10)
	b := New(seed)
	token, _ := b.BeginCompositionGroup()
	for _, step := range []struct {
		at    int
		value string
	}{{0, "x"}, {len(seed) - 1, "y"}, {0, "a"}, {len(seed) - 1, "a"}} {
		if err := b.ReplaceCompositionUTF16(token, step.at, step.at+1, step.value); err != nil {
			t.Fatal(err)
		}
	}
	var work netChangeWork
	if b.transactionChangesBytes(*b.group, &work) || work.skippedCurrentBytes < len(seed)-2 || work.comparedBytes > 4 || work.currentLookups > 8 {
		t.Fatal("shared gap copied/rescanned", work)
	}
	if b.EndCompositionGroup(token) != nil {
		t.Fatal("end")
	}
	// An earlier deletion shifts a long identical current span. Its bytes must
	// actually be checked, but at most once, rather than once per retained edit.
	b = New(seed)
	if b.BeginGroup() != nil {
		t.Fatal("begin")
	}
	netTestReplace(t, b, 0, 1, "")
	netTestReplace(t, b, len(seed)-1, len(seed)-1, "a")
	work = netChangeWork{}
	if b.transactionChangesBytes(*b.group, &work) || work.comparedBytes > len(seed) || work.nodes > 7 {
		t.Fatal("shifted equal gap work", work)
	}
	b.CancelGroup()
	// More than one thousand retained overlapping edits: no artificial E cap.
	value := "ab🙂cd"
	var edits []change
	for i := 0; i < 1200; i++ {
		next := "x🙂y"
		if i%2 != 0 {
			next = "ab🙂cd"
		}
		var old string
		value, old = netOracleEdit(t, value, 0, netTestUnits(value), next)
		edits = append(edits, change{0, old, next})
	}
	b = New(value)
	work = netChangeWork{}
	if got := b.transactionChangesBytes(transaction{edits: edits}, &work); got != (value != "ab🙂cd") || work.nodes > 3*len(edits)+1 || work.joinVisits+work.splitVisits > 80*len(edits)*bits.Len(uint(len(edits)+1)) {
		t.Fatal("adversarial retained edit bound", got, work)
	}
}
