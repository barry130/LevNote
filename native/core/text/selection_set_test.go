package text

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func mustSetSelections(t *testing.T, b *Buffer, ranges []Range, primary int) SelectionSet {
	t.Helper()
	if err := b.SetSelectionSet(SelectionSet{ranges, primary}); err != nil {
		t.Fatal(err)
	}
	return b.SelectionSet()
}

func requireSelections(t *testing.T, b *Buffer, want SelectionSet) {
	t.Helper()
	if got := b.SelectionSet(); !reflect.DeepEqual(got, want) || b.Selection() != want.Ranges[want.Primary] {
		t.Fatalf("selections=%+v primary=%+v; want=%+v", got, b.Selection(), want)
	}
}

func TestSelectionSetNormalizationDirectionsPrimaryAndBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name    string
		input   []Range
		primary int
		want    SelectionSet
	}{
		{"document-order", []Range{{9, 7}, {1, 3}}, 0, SelectionSet{[]Range{{1, 3}, {9, 7}}, 1}},
		{"overlap-primary-forward", []Range{{5, 1}, {3, 7}}, 1, SelectionSet{[]Range{{1, 7}}, 0}},
		{"overlap-primary-backward", []Range{{1, 5}, {7, 3}}, 1, SelectionSet{[]Range{{7, 1}}, 0}},
		{"chain-primary-first", []Range{{4, 1}, {3, 7}, {6, 9}}, 0, SelectionSet{[]Range{{9, 1}}, 0}},
		{"chain-primary-last", []Range{{4, 1}, {3, 7}, {6, 9}}, 2, SelectionSet{[]Range{{1, 9}}, 0}},
		{"adjacent-independent", []Range{{1, 3}, {5, 3}}, 0, SelectionSet{[]Range{{1, 3}, {5, 3}}, 0}},
		{"shared-caret-right", []Range{{1, 3}, {5, 3}, {3, 3}}, 2, SelectionSet{[]Range{{1, 3}, {5, 3}}, 1}},
		{"start-caret", []Range{{5, 1}, {1, 1}}, 1, SelectionSet{[]Range{{5, 1}}, 0}},
		{"end-caret", []Range{{1, 5}, {5, 5}}, 1, SelectionSet{[]Range{{1, 5}}, 0}},
		{"inside-primary-caret-first-direction", []Range{{4, 4}, {5, 1}, {3, 7}}, 0, SelectionSet{[]Range{{7, 1}}, 0}},
		{"equal-start-stable-direction", []Range{{4, 4}, {1, 7}, {5, 1}}, 0, SelectionSet{[]Range{{1, 7}}, 0}},
		{"duplicate-caret", []Range{{8, 8}, {2, 2}, {8, 8}}, 2, SelectionSet{[]Range{{2, 2}, {8, 8}}, 1}},
		{"duplicate-range-primary-direction", []Range{{1, 5}, {5, 1}}, 1, SelectionSet{[]Range{{5, 1}}, 0}},
		{"carets-do-not-bridge", []Range{{2, 2}, {4, 2}, {0, 2}, {2, 2}}, 0, SelectionSet{[]Range{{0, 2}, {4, 2}}, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := New("0123456789")
			version, history := b.Version(), b.History()
			input := append([]Range(nil), tc.input...)
			got, err := b.NormalizeSelectionSet(SelectionSet{input, tc.primary})
			if err != nil || !reflect.DeepEqual(got, tc.want) || !reflect.DeepEqual(input, tc.input) || b.SelectionGeneration() != 0 {
				t.Fatalf("normalize=%+v err=%v want=%+v", got, err, tc.want)
			}
			if err := b.SetSelectionSet(SelectionSet{input, tc.primary}); err != nil {
				t.Fatal(err)
			}
			requireSelections(t, b, tc.want)
			if b.Version() != version || b.History() != history || b.String() != "0123456789" {
				t.Fatal("selection normalization mutated text/history")
			}
			generation := b.SelectionGeneration()
			if err := b.SetSelectionSet(tc.want); err != nil || b.SelectionGeneration() != generation {
				t.Fatal("same normalized set changed generation", err)
			}
		})
	}
}

func TestSelectionSetValidationAnd256Limit(t *testing.T) {
	b := New("甲🙂e\u0301\r\n乙" + strings.Repeat("x", 300))
	mustSetSelections(t, b, []Range{{3, 1}, {7, 7}}, 1)
	for _, tc := range []struct {
		set  SelectionSet
		want error
	}{
		{SelectionSet{}, ErrSelectionSet},
		{SelectionSet{[]Range{{0, 0}}, -1}, ErrSelectionSet},
		{SelectionSet{[]Range{{0, 0}}, 1}, ErrSelectionSet},
		{SelectionSet{[]Range{{0, 0}, {2, 3}}, 0}, ErrSurrogate},
		{SelectionSet{[]Range{{3, 2}}, 0}, ErrSurrogate},
		{SelectionSet{[]Range{{-1, 0}}, 0}, ErrRange},
		{SelectionSet{[]Range{{0, 9999}}, 0}, ErrRange},
		{SelectionSet{make([]Range, 257), 0}, ErrSelectionLimit},
	} {
		unchanged := multiStateGuard(t, b)
		if _, err := b.NormalizeSelectionSet(tc.set); !errors.Is(err, tc.want) {
			t.Fatal("normalize validation", err, tc.want)
		}
		if err := b.SetSelectionSet(tc.set); !errors.Is(err, tc.want) {
			t.Fatal("set validation", err, tc.want)
		}
		unchanged()
	}
	ranges := make([]Range, MaxSelections)
	for i := range ranges {
		ranges[i] = Range{8 + i, 8 + i}
	}
	mustSetSelections(t, b, ranges, 255)
	if len(b.SelectionSet().Ranges) != 256 || b.SelectionSet().Primary != 255 {
		t.Fatal("exact limit rejected")
	}
	unchanged := multiStateGuard(t, b)
	if err := b.SetSelectionSet(SelectionSet{append(ranges, Range{}), 256}); !errors.Is(err, ErrSelectionLimit) {
		t.Fatal("257 selections accepted", err)
	}
	unchanged()
}

func TestSelectionSetGenerationCompatibilityAndNoSingleSelectionAllocation(t *testing.T) {
	b := New("abcd")
	seed, version, history := b.Snapshot(), b.Version(), b.History()
	_ = b.SetSelection(1, 2)
	generation := b.SelectionGeneration()
	_ = b.SetSelection(1, 2)
	if b.SelectionGeneration() != generation {
		t.Fatal("identical legacy selection changed generation")
	}
	mustSetSelections(t, b, []Range{{1, 2}, {3, 4}}, 0)
	if b.SelectionGeneration() != generation+1 {
		t.Fatal("secondary did not change identity")
	}
	generation++
	mustSetSelections(t, b, []Range{{1, 2}, {3, 4}}, 1)
	if b.SelectionGeneration() != generation+1 || b.Selection() != (Range{3, 4}) {
		t.Fatal("primary-only change did not change identity")
	}
	generation++
	mustSetSelections(t, b, []Range{{1, 2}, {4, 3}}, 1)
	if b.SelectionGeneration() != generation+1 {
		t.Fatal("direction-only change did not change identity")
	}
	generation++
	_ = b.SetSelection(4, 3)
	requireSelections(t, b, SelectionSet{[]Range{{4, 3}}, 0})
	if b.SelectionGeneration() != generation+1 || b.Version() != version || b.History() != history || !b.EqualSnapshot(seed) {
		t.Fatal("legacy collapse affected text/history or missed generation")
	}
	if n := testing.AllocsPerRun(100, func() {
		_ = b.SetSelection(1, 1)
		_ = b.Selection()
		_ = b.SetSelection(2, 2)
	}); n != 0 {
		t.Fatal("legacy selection path allocates", n)
	}
}

func TestSelectionSetSnapshotAndHistoryOwnTheirSlices(t *testing.T) {
	b := New("abcdef")
	input := []Range{{1, 0}, {3, 4}, {6, 5}}
	want := mustSetSelections(t, b, input, 1)
	snapshot := b.Snapshot()
	input[0] = Range{6, 6}
	read := b.SelectionSet()
	read.Ranges[0] = Range{}
	read = snapshot.SelectionSet()
	read.Ranges[2] = Range{}
	requireSelections(t, b, want)
	if !reflect.DeepEqual(snapshot.SelectionSet(), want) {
		t.Fatal("snapshot leaked write access")
	}
	if err := b.ReplaceUTF16(3, 4, "XYZ"); err != nil {
		t.Fatal(err)
	}
	requireSelections(t, b, SelectionSet{[]Range{{6, 6}}, 0})
	if len(b.undo[0].beforeSelections.secondary) != 2 {
		t.Fatal("legacy replace lost complete before state")
	}
	_ = b.SetSelection(0, 0)
	generation := b.SelectionGeneration()
	if !b.Undo() || b.SelectionGeneration() <= generation {
		t.Fatal("undo reused old selection identity")
	}
	requireSelections(t, b, want)
	// Current storage must not be the historical or snapshot storage.
	b.selections.secondary[0] = Range{2, 2}
	if !reflect.DeepEqual(snapshot.SelectionSet(), want) || b.redo[0].beforeSelections.secondary[0] != want.Ranges[0] {
		t.Fatal("live set shares snapshot or history storage")
	}
	generation = b.SelectionGeneration()
	b.Restore(snapshot)
	requireSelections(t, b, want)
	if b.SelectionGeneration() <= generation || b.CanUndo() || b.CanRedo() || b.InGroup() {
		t.Fatal("restore did not invalidate identity/reset history")
	}
	b.selections.secondary[0] = Range{}
	if !reflect.DeepEqual(snapshot.SelectionSet(), want) {
		t.Fatal("Restore adopted snapshot write ownership")
	}
}

func TestSelectionSetOrdinaryAndCompositionGroupsRestoreEverything(t *testing.T) {
	for _, mode := range []string{"ordinary-cancel", "ordinary-end", "composition-cancel", "composition-end", "reset", "restore"} {
		t.Run(mode, func(t *testing.T) {
			b := New("abcdef")
			before := mustSetSelections(t, b, []Range{{1, 0}, {4, 2}, {6, 5}}, 1)
			seed := b.Snapshot()
			var token CompositionToken
			if strings.HasPrefix(mode, "composition") || mode == "reset" || mode == "restore" {
				var err error
				token, err = b.BeginCompositionGroup()
				if err != nil || b.ReplaceCompositionUTF16(token, 2, 4, "Q") != nil {
					t.Fatal("composition fixture", err)
				}
				unchanged := multiStateGuard(t, b)
				if err := b.SetSelectionSet(before); !errors.Is(err, ErrGroup) {
					t.Fatal("multi-set took IME ownership", err)
				}
				unchanged()
			} else {
				if b.BeginGroup() != nil || b.ReplaceUTF16(2, 4, "Q") != nil {
					t.Fatal("ordinary fixture")
				}
			}
			switch mode {
			case "ordinary-cancel":
				if !b.CancelGroup() {
					t.Fatal("cancel")
				}
			case "composition-cancel":
				if err := b.CancelCompositionGroup(token); err != nil {
					t.Fatal(err)
				}
			case "ordinary-end":
				after := mustSetSelections(t, b, []Range{{0, 0}, {5, 4}}, 1)
				if !b.EndGroup() || !b.Undo() {
					t.Fatal("end/undo")
				}
				requireSelections(t, b, before)
				if !b.Redo() || b.String() != "abQef" {
					t.Fatal("redo")
				}
				requireSelections(t, b, after)
				if !b.Undo() {
					t.Fatal("second undo")
				}
			case "composition-end":
				if b.EndCompositionGroup(token) != nil || !b.Undo() {
					t.Fatal("composition end/undo")
				}
			case "restore":
				b.Restore(seed)
			case "reset":
				after := b.SelectionSet()
				b.ResetHistory()
				requireSelections(t, b, after)
				if b.OwnsComposition(token) || b.InGroup() || b.CanUndo() || b.CanRedo() || b.String() != "abQef" {
					t.Fatal("reset did not keep current complete state")
				}
				return
			}
			requireSelections(t, b, before)
			if b.String() != "abcdef" || b.OwnsComposition(token) || b.InGroup() {
				t.Fatal("group did not restore full state")
			}
		})
	}
}
