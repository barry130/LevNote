package text

import (
	"errors"
	"math"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unicode/utf16"
)

// Retain independent copies of every mutable model input, including journals,
// all history endpoints and active owner pointers. Refusals must preserve this
// complete state, not merely the publicly visible string or undo depth.
func multiStateGuard(t *testing.T, b *Buffer) func() {
	t.Helper()
	before := *b
	before.pieces = append([]piece(nil), b.pieces...)
	before.prefix = append([]piecePrefix(nil), b.prefix...)
	before.selections = cloneExtra(b.selections)
	cloneTransaction := func(source transaction) transaction {
		out := source
		out.edits = append([]change(nil), source.edits...)
		out.beforeSelections = cloneExtra(source.beforeSelections)
		out.afterSelections = cloneExtra(source.afterSelections)
		return out
	}
	cloneHistory := func(source []transaction) []transaction {
		if source == nil {
			return nil
		}
		out := make([]transaction, len(source))
		for i, item := range source {
			out[i] = cloneTransaction(item)
		}
		return out
	}
	before.undo, before.redo = cloneHistory(b.undo), cloneHistory(b.redo)
	group, composition := b.group, b.composition
	if group != nil {
		copy := cloneTransaction(*group)
		before.group = &copy
	}
	if composition != nil {
		copy := *composition
		before.composition = &copy
	}
	return func() {
		t.Helper()
		if b.group != group || b.composition != composition || !reflect.DeepEqual(*b, before) {
			t.Fatalf("refusal mutated complete buffer state: text=%q version=%d generation=%d history=%+v", b.String(), b.Version(), b.SelectionGeneration(), b.History())
		}
	}
}

func applyMulti(t *testing.T, b *Buffer, edits []Replacement, primary int) {
	t.Helper()
	if err := b.ReplaceBatchUTF16(b.Version(), b.SelectionGeneration(), edits, primary); err != nil {
		t.Fatal(err)
	}
}

func TestMultiEditRandomStringOracleAndWholeSelectionUndoRedo(t *testing.T) {
	rng := rand.New(rand.NewSource(20261119))
	words := []string{"", "a", "汉字", "🙂", "e\u0301", "👨‍👩‍👧‍👦", "\r\n", "\n", "🇨🇳"}
	units := func(value string) int { return len(utf16.Encode([]rune(value))) }
	for trial := 0; trial < 450; trial++ {
		var source strings.Builder
		wordCount := 35
		if trial%17 == 0 {
			wordCount = 2500 // exercise batches spanning several immutable chunks
		}
		for i := 0; i < wordCount; i++ {
			source.WriteString(words[1+rng.Intn(len(words)-1)])
		}
		original := source.String()
		runes := []rune(original)
		n := 1 + rng.Intn(8)
		positions := rng.Perm(len(runes) + 1)[:2*n]
		sort.Ints(positions)
		edits := make([]Replacement, n)
		before := SelectionSet{Ranges: make([]Range, n), Primary: rng.Intn(n)}
		var expected strings.Builder
		after := SelectionSet{Ranges: make([]Range, 0, n)}
		previous := 0
		primary := rng.Intn(n)
		for i := range edits {
			left, right := positions[2*i], positions[2*i+1]
			if rng.Intn(3) == 0 {
				right = left
			}
			next := words[rng.Intn(len(words))]
			edits[i] = Replacement{units(string(runes[:left])), units(string(runes[:right])), next}
			before.Ranges[i] = Range{edits[i].Start, edits[i].End}
			if rng.Intn(2) == 0 {
				before.Ranges[i] = Range{edits[i].End, edits[i].Start}
			}
			expected.WriteString(string(runes[previous:left]))
			expected.WriteString(next)
			caret := units(expected.String())
			if len(after.Ranges) == 0 || after.Ranges[len(after.Ranges)-1].Head != caret {
				after.Ranges = append(after.Ranges, Range{caret, caret})
			}
			if i == primary {
				after.Primary = len(after.Ranges) - 1
			}
			previous = right
		}
		expected.WriteString(string(runes[previous:]))
		want := expected.String()
		b := New(original)
		mustSetSelections(t, b, before.Ranges, before.Primary)
		requireSelections(t, b, before)
		applyMulti(t, b, edits, primary)
		if b.String() != want || b.LenUTF16() != units(want) || b.LenBytes() != len(want) {
			t.Fatalf("trial %d source=%q edits=%+v got=%q want=%q", trial, original, edits, b.String(), want)
		}
		requireSelections(t, b, after)
		if original == want {
			if b.Version() != 0 || b.CanUndo() {
				t.Fatal("oracle no-op created history")
			}
			continue
		}
		if b.History().UndoDepth != 1 || len(b.undo[0].edits) > len(edits) {
			t.Fatal("batch did not commit exactly one transaction")
		}
		generation := b.SelectionGeneration()
		if !b.Undo() || b.CanUndo() || b.String() != original || b.SelectionGeneration() <= generation {
			t.Fatal("batch undo", trial)
		}
		requireSelections(t, b, before)
		generation = b.SelectionGeneration()
		if !b.Redo() || b.CanRedo() || b.String() != want || b.SelectionGeneration() <= generation {
			t.Fatal("batch redo", trial)
		}
		requireSelections(t, b, after)
	}
}

func TestMultiEditAdjacentReplacementsAndCollapsedCaretMapping(t *testing.T) {
	for _, tc := range []struct {
		name    string
		edits   []Replacement
		primary int
		want    string
		after   SelectionSet
	}{
		{"adjacent-independent", []Replacement{{0, 2, "中"}, {2, 4, "🙂"}, {4, 6, "!"}}, 1, "中🙂!", SelectionSet{[]Range{{1, 1}, {3, 3}, {4, 4}}, 1}},
		{"delete-dedup-left-primary", []Replacement{{0, 2, ""}, {2, 4, ""}, {4, 6, ""}}, 0, "", SelectionSet{[]Range{{0, 0}}, 0}},
		{"delete-dedup-right-primary", []Replacement{{0, 2, ""}, {2, 4, ""}, {4, 6, ""}}, 2, "", SelectionSet{[]Range{{0, 0}}, 0}},
		{"shift-primary", []Replacement{{0, 0, "🙂"}, {2, 4, "甲乙"}, {6, 6, "Z"}}, 2, "🙂ab甲乙efZ", SelectionSet{[]Range{{2, 2}, {6, 6}, {9, 9}}, 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := New("abcdef")
			before := mustSetSelections(t, b, []Range{{2, 0}, {4, 2}, {6, 4}}, 2)
			applyMulti(t, b, tc.edits, tc.primary)
			if b.String() != tc.want {
				t.Fatal(b.String(), tc.want)
			}
			requireSelections(t, b, tc.after)
			if !b.Undo() || b.String() != "abcdef" {
				t.Fatal("undo")
			}
			requireSelections(t, b, before)
			if !b.Redo() || b.String() != tc.want {
				t.Fatal("redo")
			}
			requireSelections(t, b, tc.after)
		})
	}
}

func TestMultiEditUnicodeGraphemeDeletesAndCRLFPreserveOriginalBytes(t *testing.T) {
	clusters := []string{"🙂", "e\u0301", "👨‍👩‍👧‍👦", "🇨🇳", "\r\n"}
	var source strings.Builder
	carets, edits := []Range{}, []Replacement{}
	for _, cluster := range clusters {
		source.WriteString("|")
		start := UTF16Len(source.String())
		source.WriteString(cluster)
		end := UTF16Len(source.String())
		carets = append(carets, Range{end, end})
		edits = append(edits, Replacement{start, end, ""})
	}
	source.WriteString("|")
	b := New(source.String())
	before := mustSetSelections(t, b, carets, 3)
	for i, caret := range carets {
		start, err := b.GraphemeBoundary(caret.Head, -1)
		if err != nil || start != edits[i].Start {
			t.Fatal("cluster split", i, start, err)
		}
	}
	applyMulti(t, b, edits, 3)
	if b.String() != "||||||" {
		t.Fatal("delete changed untouched bytes", b.String())
	}
	if !b.Undo() || b.String() != source.String() {
		t.Fatal("Unicode undo lost original bytes")
	}
	requireSelections(t, b, before)
}

func TestMultiEditValidationRefusalsPreserveEveryField(t *testing.T) {
	b := NewWithLimit("甲🙂abcdef", 20)
	_ = b.ReplaceUTF16(b.LenUTF16(), b.LenUTF16(), "!")
	if !b.Undo() {
		t.Fatal("redo fixture")
	}
	mustSetSelections(t, b, []Range{{3, 1}, {4, 6}}, 0)
	for _, tc := range []struct {
		name    string
		edits   []Replacement
		primary int
		want    error
	}{
		{"empty", nil, 0, ErrSelectionSet},
		{"primary-negative", []Replacement{{0, 0, "x"}}, -1, ErrSelectionSet},
		{"primary-high", []Replacement{{0, 0, "x"}}, 1, ErrSelectionSet},
		{"utf8-later", []Replacement{{0, 1, "x"}, {4, 5, "\xff"}}, 0, ErrUTF8},
		{"negative", []Replacement{{-1, 1, "x"}}, 0, ErrRange},
		{"huge", []Replacement{{0, math.MaxInt, "x"}}, 0, ErrRange},
		{"reversed", []Replacement{{4, 3, "x"}}, 0, ErrRange},
		{"surrogate-start", []Replacement{{2, 3, "x"}}, 0, ErrSurrogate},
		{"surrogate-end", []Replacement{{1, 2, "x"}}, 0, ErrSurrogate},
		{"order", []Replacement{{4, 5, "x"}, {0, 1, "x"}}, 0, ErrEditOrder},
		{"overlap", []Replacement{{0, 4, "x"}, {3, 5, "x"}}, 0, ErrEditOrder},
		{"duplicate-empty", []Replacement{{3, 3, "x"}, {3, 3, "y"}}, 1, ErrEditOrder},
		{"same-start", []Replacement{{3, 3, "x"}, {3, 5, "y"}}, 0, ErrEditOrder},
		{"touch-empty", []Replacement{{0, 3, "x"}, {3, 3, "y"}}, 0, ErrEditOrder},
		{"limit", []Replacement{{0, 1, strings.Repeat("x", 21)}}, 0, ErrSizeLimit},
		{"257", make([]Replacement, 257), 0, ErrSelectionLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			unchanged := multiStateGuard(t, b)
			if err := b.ReplaceBatchUTF16(b.Version(), b.SelectionGeneration(), tc.edits, tc.primary); !errors.Is(err, tc.want) {
				t.Fatal(err, tc.want)
			}
			unchanged()
		})
	}
	for _, stale := range []struct{ version, generation uint64 }{
		{b.Version() - 1, b.SelectionGeneration()},
		{b.Version(), b.SelectionGeneration() - 1},
		{b.Version() + 1, b.SelectionGeneration()},
		{b.Version(), b.SelectionGeneration() + 1},
	} {
		unchanged := multiStateGuard(t, b)
		if err := b.ReplaceBatchUTF16(stale.version, stale.generation, []Replacement{{0, 1, "X"}}, 0); !errors.Is(err, ErrStaleEdit) {
			t.Fatal("stale edit accepted", err)
		}
		unchanged()
	}
}

func TestMultiEditFinalAndIntermediateByteBudgetExactAndPlusOne(t *testing.T) {
	for _, limit := range []int{9, 10} {
		b := NewWithLimit("甲🙂x", limit)
		before := mustSetSelections(t, b, []Range{{1, 0}, {3, 1}}, 1)
		edits := []Replacement{{0, 1, "乙"}, {1, 3, "中文"}}
		if limit == 9 {
			unchanged := multiStateGuard(t, b)
			if err := b.ReplaceBatchUTF16(b.Version(), b.SelectionGeneration(), edits, 1); !errors.Is(err, ErrSizeLimit) {
				t.Fatal("one byte over accepted", err)
			}
			unchanged()
			continue
		}
		applyMulti(t, b, edits, 1)
		if b.String() != "乙中文x" || b.LenBytes() != 10 || !b.Undo() {
			t.Fatal("exact byte budget")
		}
		requireSelections(t, b, before)
	}
	// Descending inserts right first: final 6 fits, intermediate 10 does not.
	b := NewWithLimit("12345", 6)
	_ = b.ReplaceUTF16(0, 1, "x")
	_ = b.Undo()
	mustSetSelections(t, b, []Range{{5, 0}}, 0)
	unchanged := multiStateGuard(t, b)
	if err := b.ReplaceBatchUTF16(b.Version(), b.SelectionGeneration(), []Replacement{{0, 4, ""}, {5, 5, "abcde"}}, 1); !errors.Is(err, ErrSizeLimit) {
		t.Fatal("intermediate overflow accepted", err)
	}
	unchanged()
	// Same final text also obeys conservative intermediate admission.
	if err := b.ReplaceBatchUTF16(b.Version(), b.SelectionGeneration(), []Replacement{{0, 4, ""}, {4, 5, "12345"}}, 1); !errors.Is(err, ErrSizeLimit) {
		t.Fatal("net-zero bypassed conservative limit", err)
	}
	unchanged()
}

func TestMultiEdit256ExactBoundAnd257Rejected(t *testing.T) {
	b := NewWithLimit(strings.Repeat("a", 511), 767)
	edits, ranges := make([]Replacement, 256), make([]Range, 256)
	for i := range edits {
		edits[i] = Replacement{2 * i, 2 * i, "🙂"}
		ranges[i] = Range{2 * i, 2 * i}
	}
	before := mustSetSelections(t, b, ranges, 255)
	unchanged := multiStateGuard(t, b)
	if err := b.ReplaceBatchUTF16(b.Version(), b.SelectionGeneration(), edits, 255); !errors.Is(err, ErrSizeLimit) {
		t.Fatal("byte accounting used UTF16 instead of UTF8", err)
	}
	unchanged()
	for i := range edits {
		edits[i].Text = "x"
	}
	applyMulti(t, b, edits, 255)
	if b.LenBytes() != 767 || len(b.SelectionSet().Ranges) != 256 || b.SelectionSet().Primary != 255 || b.Selection() != (Range{766, 766}) {
		t.Fatal("256 exact result", b.LenBytes(), b.Selection())
	}
	if !b.Undo() {
		t.Fatal("undo")
	}
	requireSelections(t, b, before)
	unchanged = multiStateGuard(t, b)
	if err := b.ReplaceBatchUTF16(b.Version(), b.SelectionGeneration(), append(edits, Replacement{}), 0); !errors.Is(err, ErrSelectionLimit) {
		t.Fatal("257 accepted", err)
	}
	unchanged()
}

func TestMultiEditNoOpPreservesRedoIncludingRedistributedBytes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		edits  []Replacement
	}{
		{"individual", "abcd", []Replacement{{0, 1, "a"}, {2, 4, "cd"}}},
		{"redistributed", "abcd", []Replacement{{0, 2, "a"}, {2, 4, "bcd"}}},
		{"deletion-insertion", "abcd", []Replacement{{0, 1, ""}, {1, 4, "abcd"}}},
		{"piece-stream", strings.Repeat("ab", 5000), []Replacement{{0, 2, "a"}, {2, 10000, "b" + strings.Repeat("ab", 4999)}}},
		{"shifted-gap-stream", strings.Repeat("a", 13000), []Replacement{{0, 1, ""}, {12999, 13000, "aa"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := New(tc.source)
			_ = b.ReplaceUTF16(b.LenUTF16(), b.LenUTF16(), "!")
			_ = b.Undo()
			mustSetSelections(t, b, []Range{{1, 0}, {b.LenUTF16(), b.LenUTF16()}}, 0)
			version, history, journal, changed := b.Version(), b.History(), b.rangeChanges, b.changedLines
			applyMulti(t, b, tc.edits, 1)
			if b.String() != tc.source || b.Version() != version || b.History() != history || b.rangeChanges != journal || b.changedLines != changed || !b.CanRedo() {
				t.Fatal("no-op changed text identity/history/journal")
			}
			if !b.Redo() || b.String() != tc.source+"!" {
				t.Fatal("no-op erased redo")
			}
		})
	}
}

func TestMultiEditStaleSecondaryPrimaryUndoRestoreAndLegacyNoOp(t *testing.T) {
	b := New("abcdef")
	initial := mustSetSelections(t, b, []Range{{1, 0}, {4, 3}}, 0)
	version, generation := b.Version(), b.SelectionGeneration()
	mustSetSelections(t, b, []Range{{1, 0}, {5, 4}}, 0)
	if b.Selection() != initial.Ranges[0] || b.Version() != version {
		t.Fatal("secondary-only fixture changed primary/text")
	}
	unchanged := multiStateGuard(t, b)
	if err := b.ReplaceBatchUTF16(version, generation, []Replacement{{0, 1, "X"}}, 0); !errors.Is(err, ErrStaleEdit) {
		t.Fatal("secondary-only movement accepted old edit", err)
	}
	unchanged()
	generation = b.SelectionGeneration()
	mustSetSelections(t, b, []Range{{1, 0}, {5, 4}}, 1)
	unchanged = multiStateGuard(t, b)
	if err := b.ReplaceBatchUTF16(version, generation, []Replacement{{0, 1, "X"}}, 0); !errors.Is(err, ErrStaleEdit) {
		t.Fatal("primary-only movement accepted old edit", err)
	}
	unchanged()
	generation = b.SelectionGeneration()
	seed := b.Snapshot()
	applyMulti(t, b, []Replacement{{0, 1, "X"}, {4, 5, "Y"}}, 0)
	if !b.Undo() || b.String() != "abcdef" || b.SelectionGeneration() <= generation {
		t.Fatal("undo did not retire the original identity")
	}
	unchanged = multiStateGuard(t, b)
	if err := b.ReplaceBatchUTF16(version, generation, []Replacement{{0, 1, "X"}}, 0); !errors.Is(err, ErrStaleEdit) {
		t.Fatal("same text after undo accepted old edit", err)
	}
	unchanged()
	version, generation = b.Version(), b.SelectionGeneration()
	history := b.History()
	// Legacy no-op still means one primary caret, without erasing redo.
	if err := b.ReplaceUTF16(4, 5, "e"); err != nil || b.Version() != version || b.History() != history || !b.CanRedo() {
		t.Fatal("legacy no-op changed history/text", err)
	}
	requireSelections(t, b, SelectionSet{[]Range{{5, 5}}, 0})
	if b.SelectionGeneration() <= generation {
		t.Fatal("legacy no-op did not retire secondary set")
	}
	b.Restore(seed)
	version, generation = b.Version(), b.SelectionGeneration()
	b.Restore(seed)
	unchanged = multiStateGuard(t, b)
	if err := b.ReplaceBatchUTF16(version, generation, []Replacement{{0, 1, "X"}}, 0); !errors.Is(err, ErrStaleEdit) {
		t.Fatal("same snapshot restore accepted old identity", err)
	}
	unchanged()
}

func TestMultiEditForeignGroupsAndTokensNeverLoseOwnership(t *testing.T) {
	for _, composition := range []bool{false, true} {
		b := New("abcdef")
		mustSetSelections(t, b, []Range{{1, 0}, {5, 3}}, 1)
		var token CompositionToken
		if composition {
			token = mustComposition(t, b)
			if err := b.ReplaceCompositionUTF16(token, 3, 5, "Z"); err != nil {
				t.Fatal(err)
			}
		} else if b.BeginGroup() != nil || b.ReplaceUTF16(3, 5, "Z") != nil {
			t.Fatal("group fixture")
		}
		unchanged := multiStateGuard(t, b)
		for _, next := range []string{"", "x"} {
			if err := b.ReplaceBatchUTF16(b.Version(), b.SelectionGeneration(), []Replacement{{0, 0, next}}, 0); !errors.Is(err, ErrGroup) {
				t.Fatal("foreign group accepted", err)
			}
			unchanged()
		}
		other := New("foreign")
		foreign := mustComposition(t, other)
		if err := b.CancelCompositionGroup(foreign); !errors.Is(err, ErrComposition) {
			t.Fatal("foreign token accepted", err)
		}
		unchanged()
		if composition {
			if !b.OwnsComposition(token) || b.CancelCompositionGroup(token) != nil {
				t.Fatal("batch swallowed composition token")
			}
		} else if !b.CancelGroup() {
			t.Fatal("batch swallowed ordinary group")
		}
		_ = other.CancelCompositionGroup(foreign)
	}
}

// explicitBatchOracle computes UTF-16-to-byte boundaries directly from runes,
// independently of Buffer slicing, virtual endpoint mapping and normalization.
func explicitBatchOracle(t *testing.T, source string, edits []Replacement) string {
	t.Helper()
	boundaries := map[int]int{0: 0}
	units := 0
	for at, r := range source {
		boundaries[units] = at
		units += len(utf16.Encode([]rune{r}))
	}
	boundaries[units] = len(source)
	var out strings.Builder
	previous := 0
	for _, edit := range edits {
		left, lok := boundaries[edit.Start]
		right, rok := boundaries[edit.End]
		if !lok || !rok || left < previous || left > right {
			t.Fatal("bad independent oracle fixture", edit)
		}
		out.WriteString(source[previous:left])
		out.WriteString(edit.Text)
		previous = right
	}
	out.WriteString(source[previous:])
	return out.String()
}

func TestExplicitBatchIndependentUnicodeOracleAndDirectedHistory(t *testing.T) {
	rng := rand.New(rand.NewSource(85020261120))
	words := []string{"", "a", "汉", "🙂", "e\u0301", "👨‍👩‍👧‍👦", "\r\n", "🇨🇳"}
	units := func(s string) int { return len(utf16.Encode([]rune(s))) }
	for trial := 0; trial < 320; trial++ {
		var text strings.Builder
		for i := 0; i < 40; i++ {
			text.WriteString(words[1+rng.Intn(len(words)-1)])
		}
		source := text.String()
		runes := []rune(source)
		n := 1 + rng.Intn(8)
		positions := rng.Perm(len(runes) + 1)[:2*n]
		sort.Ints(positions)
		edits := make([]Replacement, n)
		for i := range edits {
			left, right := positions[2*i], positions[2*i+1]
			if trial%3 == 0 {
				right = left
			}
			edits[i] = Replacement{units(string(runes[:left])), units(string(runes[:right])), words[rng.Intn(len(words))]}
		}
		want := explicitBatchOracle(t, source, edits)
		outRunes := []rune(want)
		after := SelectionSet{Ranges: []Range{{0, 0}}}
		if len(outRunes) >= 8 {
			points := rng.Perm(len(outRunes) + 1)[:8]
			sort.Ints(points)
			after = SelectionSet{Ranges: make([]Range, 4), Primary: trial % 4}
			for i := range after.Ranges {
				left, right := units(string(outRunes[:points[2*i]])), units(string(outRunes[:points[2*i+1]]))
				after.Ranges[i] = Range{left, right}
				if (trial+i)%2 == 0 {
					after.Ranges[i] = Range{right, left}
				}
			}
		}
		b := New(source)
		before := mustSetSelections(t, b, []Range{{units(string(runes[:1])), 0}, {units(source), units(source)}}, 1)
		supplied := SelectionSet{Ranges: append([]Range(nil), after.Ranges...), Primary: after.Primary}
		changed, err := b.ReplaceBatchWithSelectionUTF16(b.Version(), b.SelectionGeneration(), edits, supplied)
		if err != nil || changed != (want != source) || b.String() != want || !reflect.DeepEqual(supplied, after) {
			t.Fatal("explicit oracle", trial, changed, err, b.String(), want)
		}
		requireSelections(t, b, after)
		if !changed {
			if b.CanUndo() {
				t.Fatal("no-op invented history")
			}
			continue
		}
		if b.History().UndoDepth != 1 || !b.Undo() || b.String() != source {
			t.Fatal("whole explicit undo", trial)
		}
		requireSelections(t, b, before)
		if !b.Redo() || b.String() != want {
			t.Fatal("whole explicit redo", trial)
		}
		requireSelections(t, b, after)
		// Supplied slices cannot mutate either live or retained selections.
		history := b.History()
		supplied.Ranges[0] = Range{}
		requireSelections(t, b, after)
		if !b.Undo() || b.String() != source {
			t.Fatal("post-mutation whole explicit undo", trial)
		}
		requireSelections(t, b, before)
		if !b.Redo() || b.String() != want || b.History() != history {
			t.Fatal("post-mutation whole explicit redo/history", trial)
		}
		requireSelections(t, b, after)
	}
}

func TestExplicitBatchNoOpSelectionOnlyAndExactStorage(t *testing.T) {
	b := New("abcd")
	_ = b.ReplaceUTF16(4, 4, "!")
	_ = b.Undo()
	before := mustSetSelections(t, b, []Range{{1, 0}, {4, 3}}, 1)
	version, generation, history, journal, lines := b.Version(), b.SelectionGeneration(), b.History(), b.rangeChanges, b.changedLines
	edits := []Replacement{{0, 2, "a"}, {2, 4, "bcd"}}
	after := SelectionSet{[]Range{{0, 0}, {4, 2}}, 1}
	changed, err := b.ReplaceBatchWithSelectionUTF16(version, generation, edits, after)
	if err != nil || changed || b.String() != "abcd" || b.Version() != version || b.History() != history || b.rangeChanges != journal || b.changedLines != lines || !b.CanRedo() || b.SelectionGeneration() != generation+1 {
		t.Fatal("selection-only touched text/history", changed, err)
	}
	requireSelections(t, b, after)
	storage := &b.selections.secondary[0]
	unchanged := multiStateGuard(t, b)
	if changed, err = b.ReplaceBatchWithSelectionUTF16(b.Version(), b.SelectionGeneration(), edits, after); err != nil || changed {
		t.Fatal(changed, err)
	}
	unchanged()
	if &b.selections.secondary[0] != storage {
		t.Fatal("exact no-op replaced selection storage")
	}
	if !b.Redo() || b.String() != "abcd!" || len(before.Ranges) != 2 {
		t.Fatal("selection-only lost redo")
	}
}

func TestExplicitBatchRefusalsAndForeignOwnerRemainAtomic(t *testing.T) {
	b := NewWithLimit("甲🙂abcd", 20)
	_ = b.ReplaceUTF16(b.LenUTF16(), b.LenUTF16(), "!")
	_ = b.Undo()
	mustSetSelections(t, b, []Range{{3, 1}, {7, 7}}, 1)
	valid := SelectionSet{Ranges: []Range{{0, 0}}}
	for _, tc := range []struct {
		name  string
		edits []Replacement
		after SelectionSet
		want  error
	}{
		{"empty", nil, valid, ErrSelectionSet},
		{"257-edits", make([]Replacement, 257), valid, ErrSelectionLimit},
		{"utf8", []Replacement{{0, 1, "\xff"}}, valid, ErrUTF8},
		{"negative", []Replacement{{-1, 0, "x"}}, valid, ErrRange},
		{"reverse", []Replacement{{3, 1, "x"}}, valid, ErrRange},
		{"surrogate-edit", []Replacement{{2, 3, "x"}}, valid, ErrSurrogate},
		{"order", []Replacement{{3, 4, "x"}, {0, 1, "y"}}, valid, ErrEditOrder},
		{"touch-empty", []Replacement{{0, 1, "x"}, {1, 1, "y"}}, valid, ErrEditOrder},
		{"too-large", []Replacement{{0, 1, strings.Repeat("x", 21)}}, valid, ErrSizeLimit},
		{"no-after", []Replacement{{0, 1, "x"}}, SelectionSet{}, ErrSelectionSet},
		{"257-after", []Replacement{{0, 1, "x"}}, SelectionSet{Ranges: make([]Range, 257)}, ErrSelectionLimit},
		{"negative-primary", []Replacement{{0, 1, "x"}}, SelectionSet{[]Range{{0, 0}}, -1}, ErrSelectionSet},
		{"high-primary", []Replacement{{0, 1, "x"}}, SelectionSet{[]Range{{0, 0}}, 1}, ErrSelectionSet},
		{"negative-after", []Replacement{{0, 1, "x"}}, SelectionSet{Ranges: []Range{{0, -1}}}, ErrRange},
		{"huge-after", []Replacement{{0, 1, "x"}}, SelectionSet{Ranges: []Range{{0, math.MaxInt}}}, ErrRange},
		{"absorbed-surrogate-source", []Replacement{{0, 1, "甲"}}, SelectionSet{[]Range{{0, 3}, {2, 2}}, 0}, ErrSurrogate},
		{"absorbed-surrogate-insert", []Replacement{{0, 1, "🙂"}}, SelectionSet{[]Range{{0, 4}, {1, 1}}, 0}, ErrSurrogate},
	} {
		t.Run(tc.name, func(t *testing.T) {
			unchanged := multiStateGuard(t, b)
			if changed, err := b.ReplaceBatchWithSelectionUTF16(b.Version(), b.SelectionGeneration(), tc.edits, tc.after); changed || !errors.Is(err, tc.want) {
				t.Fatal(changed, err, tc.want)
			}
			unchanged()
		})
	}
	for _, stale := range [][2]uint64{{b.Version() + 1, b.SelectionGeneration()}, {b.Version(), b.SelectionGeneration() + 1}} {
		unchanged := multiStateGuard(t, b)
		if changed, err := b.ReplaceBatchWithSelectionUTF16(stale[0], stale[1], []Replacement{{0, 1, "x"}}, valid); changed || !errors.Is(err, ErrStaleEdit) {
			t.Fatal("stale accepted", changed, err)
		}
		unchanged()
	}
	for _, composition := range []bool{false, true} {
		owner := New("ab")
		var token CompositionToken
		if composition {
			token = mustComposition(t, owner)
		} else if err := owner.BeginGroup(); err != nil {
			t.Fatal(err)
		}
		unchanged := multiStateGuard(t, owner)
		if changed, err := owner.ReplaceBatchWithSelectionUTF16(owner.Version(), owner.SelectionGeneration(), []Replacement{{0, 1, "a"}}, valid); changed || !errors.Is(err, ErrGroup) {
			t.Fatal("owner hijacked", changed, err)
		}
		unchanged()
		if composition {
			if owner.ReplaceCompositionUTF16(token, 0, 1, "x") != nil || owner.EndCompositionGroup(token) != nil {
				t.Fatal("composition owner unusable")
			}
		} else if owner.ReplaceUTF16(0, 1, "x") != nil || !owner.EndGroup() {
			t.Fatal("ordinary owner unusable")
		}
		if !owner.Undo() || owner.String() != "ab" || !owner.Redo() || owner.String() != "xb" {
			t.Fatal("owner roundtrip")
		}
	}
}

func TestExplicitBatch256DirectedSelectionsAndOrdinaryHistoryLimits(t *testing.T) {
	for _, budget := range []int{0, 1, 1 << 20} {
		b := New(strings.Repeat("a ", 256))
		b.SetHistoryLimits(256, budget)
		edits := make([]Replacement, 256)
		after := SelectionSet{Ranges: make([]Range, 256), Primary: 173}
		for i := range edits {
			edits[i] = Replacement{2 * i, 2*i + 1, "🙂"}
			after.Ranges[i] = Range{3*i + 2, 3 * i}
		}
		before := b.SelectionSet()
		changed, err := b.ReplaceBatchWithSelectionUTF16(b.Version(), b.SelectionGeneration(), edits, after)
		if err != nil || !changed || b.String() != strings.Repeat("🙂 ", 256) {
			t.Fatal("256 explicit ordinary rejected", budget, changed, err)
		}
		requireSelections(t, b, after)
		if budget == 1<<20 {
			if !b.Undo() {
				t.Fatal("undo")
			}
			requireSelections(t, b, before)
			if !b.Redo() {
				t.Fatal("redo")
			}
			requireSelections(t, b, after)
		} else if b.CanUndo() || !b.History().Truncated {
			t.Fatal("ordinary history not trimmed", b.History())
		}
	}
	b := New("a")
	b.SetHistoryLimits(0, 0)
	if changed, err := b.ReplaceBatchWithSelectionUTF16(b.Version(), b.SelectionGeneration(), []Replacement{{0, 1, "b"}}, SelectionSet{Ranges: []Range{{1, 0}}}); err != nil || !changed || b.CanUndo() {
		t.Fatal("zero-entry ordinary refusal", changed, err)
	}
	unchanged := multiStateGuard(t, b)
	if changed, err := b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), []PatchEdit{{0, 1, "b", "c"}}, SelectionSet{Ranges: []Range{{1, 1}}}); changed || !errors.Is(err, ErrPatchHistoryLimit) {
		t.Fatal("strict patch policy changed", changed, err)
	}
	unchanged()
}

func TestExplicitBatchGraphemeDeleteAndCutKeepUneditedCarets(t *testing.T) {
	for _, cut := range []bool{false, true} {
		b := New("x🙂y\r\nz")
		var before SelectionSet
		var edits []Replacement
		if cut {
			before = mustSetSelections(t, b, []Range{{0, 0}, {3, 1}, {7, 6}}, 2)
			// Only the nonempty copied selections are cut; caret zero does
			// not expand to or delete an adjacent grapheme.
			edits = []Replacement{{1, 3, ""}, {6, 7, ""}}
		} else {
			before = mustSetSelections(t, b, []Range{{0, 0}, {3, 3}, {6, 6}}, 2)
			for _, caret := range before.Ranges {
				start, err := b.GraphemeBoundary(caret.Head, -1)
				if err != nil {
					t.Fatal(err)
				}
				if start != caret.Head {
					edits = append(edits, Replacement{start, caret.Head, ""})
				}
			}
		}
		want, after := "xyz", SelectionSet{[]Range{{0, 0}, {1, 1}, {2, 2}}, 2}
		if cut {
			want, after = "xy\r\n", SelectionSet{[]Range{{0, 0}, {1, 1}, {4, 4}}, 2}
		}
		if changed, err := b.ReplaceBatchWithSelectionUTF16(b.Version(), b.SelectionGeneration(), edits, after); err != nil || !changed || b.String() != want {
			t.Fatal("mixed no-op caret mapping", cut, changed, err, b.String())
		}
		requireSelections(t, b, after)
		if !b.Undo() || b.String() != "x🙂y\r\nz" {
			t.Fatal("undo")
		}
		requireSelections(t, b, before)
		if !b.Redo() {
			t.Fatal("redo")
		}
		requireSelections(t, b, after)
	}
}

func TestExplicitBatchIntermediateCeilingAndOutputPrimaryNormalization(t *testing.T) {
	b := NewWithLimit("12345", 6)
	unchanged := multiStateGuard(t, b)
	for _, edits := range [][]Replacement{
		{{0, 4, ""}, {5, 5, "abcde"}},
		{{0, 4, ""}, {4, 5, "12345"}},
	} {
		if changed, err := b.ReplaceBatchWithSelectionUTF16(b.Version(), b.SelectionGeneration(), edits, SelectionSet{Ranges: []Range{{0, 0}}}); changed || !errors.Is(err, ErrSizeLimit) {
			t.Fatal("intermediate byte ceiling", changed, err)
		}
		unchanged()
	}
	b = New("abcdef")
	after := SelectionSet{[]Range{{3, 0}, {5, 2}, {3, 3}}, 1}
	if changed, err := b.ReplaceBatchWithSelectionUTF16(b.Version(), b.SelectionGeneration(), []Replacement{{0, 1, "A"}}, after); !changed || err != nil {
		t.Fatal(changed, err)
	}
	// Primary {5,2} is backward, so its merged range must remain backward.
	requireSelections(t, b, SelectionSet{[]Range{{5, 0}}, 0})
	if !b.Undo() || !b.Redo() {
		t.Fatal("normalized history")
	}
	requireSelections(t, b, SelectionSet{[]Range{{5, 0}}, 0})
	// The same overlap with forward primary must instead preserve that direction.
	after.Ranges[1] = Range{2, 5}
	if changed, err := b.ReplaceBatchWithSelectionUTF16(b.Version(), b.SelectionGeneration(), []Replacement{{0, 1, "a"}}, after); !changed || err != nil {
		t.Fatal("forward primary", changed, err)
	}
	requireSelections(t, b, SelectionSet{[]Range{{0, 5}}, 0})
	if !b.Undo() {
		t.Fatal("forward primary undo")
	}
	requireSelections(t, b, SelectionSet{[]Range{{5, 0}}, 0})
	if !b.Redo() {
		t.Fatal("forward primary redo")
	}
	requireSelections(t, b, SelectionSet{[]Range{{0, 5}}, 0})
}

func TestOrdinaryBatchOutputEndpointsIndexedGapsAnd512Boundaries(t *testing.T) {
	const sourceUnits = 1 << 28
	editStart := sourceUnits - 1024
	plan := []batchReplacement{{Replacement: Replacement{editStart, editStart, "🙂"}}}
	set := SelectionSet{Ranges: make([]Range, 256)}
	for i := range set.Ranges {
		set.Ranges[i] = Range{sourceUnits - 511 + 2*i + 2, sourceUnits - 512 + 2*i + 2}
	}
	calls := 0
	got := ordinaryBatchEndpoints(plan, sourceUnits, set, func(offset int) error {
		calls++
		if offset < sourceUnits-512 || offset > sourceUnits {
			t.Fatal("read unrelated source prefix", offset)
		}
		return nil
	})
	if calls != 512 || len(got) != 512 {
		t.Fatal("endpoint work is not bounded", calls, len(got))
	}
	for _, err := range got {
		if err != nil {
			t.Fatal(err)
		}
	}
	set = SelectionSet{Ranges: []Range{{editStart, editStart + 1}, {editStart + 2, sourceUnits + 2}}}
	got = ordinaryBatchEndpoints(plan, sourceUnits, set, func(int) error { return nil })
	if got[editStart] != nil || !errors.Is(got[editStart+1], ErrSurrogate) || got[editStart+2] != nil || got[sourceUnits+2] != nil {
		t.Fatal("insert seam/surrogate/EOF", got)
	}
	if _, ok := ordinaryBatchOffset(math.MaxInt, 1); ok {
		t.Fatal("overflow accepted")
	}
}

func TestMultiEditHistoryBudgetCountsSecondaryAndClearsOldSlots(t *testing.T) {
	b := New("abcdef")
	before := mustSetSelections(t, b, []Range{{1, 0}, {4, 3}, {6, 5}}, 1)
	applyMulti(t, b, []Replacement{{0, 1, "X"}, {3, 4, "Y"}, {5, 6, "Z"}}, 1)
	if len(b.undo) != 1 {
		t.Fatal("missing history")
	}
	want := 6 + b.undo[0].selectionBytes()
	if b.History().RetainedBytes != want || want <= 6 {
		t.Fatal("secondary memory not charged", b.History(), want)
	}
	backing := b.undo[:cap(b.undo)]
	b.SetHistoryLimits(10, want)
	if !b.CanUndo() || b.History().Truncated {
		t.Fatal("exact retained byte budget rejected")
	}
	if !b.Undo() {
		t.Fatal("undo")
	}
	requireSelections(t, b, before)
	for _, slot := range backing {
		if !reflect.DeepEqual(slot, transaction{}) {
			t.Fatal("undo move retained old slot")
		}
	}
	redoBacking := b.redo[:cap(b.redo)]
	b.SetHistoryLimits(10, want-1)
	if b.CanRedo() || !b.History().Truncated || b.History().RetainedBytes != 0 {
		t.Fatal("secondary budget did not trim", b.History())
	}
	for _, slot := range redoBacking {
		if !reflect.DeepEqual(slot, transaction{}) {
			t.Fatal("trim retained secondary slices")
		}
	}
	b.ResetHistory()
	for i := 0; i < 3; i++ {
		applyMulti(t, b, []Replacement{{0, 1, string(rune('K' + i))}, {5, 6, string(rune('P' + i))}}, 0)
	}
	// Previous tiny budget may trim these entries; lift it and create a pair.
	b.SetHistoryLimits(10, 1<<20)
	applyMulti(t, b, []Replacement{{0, 1, "u"}, {5, 6, "v"}}, 0)
	applyMulti(t, b, []Replacement{{0, 1, "w"}, {5, 6, "z"}}, 0)
	backing = b.undo[:cap(b.undo)]
	b.SetHistoryLimits(1, 1<<20)
	if b.History().UndoDepth != 1 || !reflect.DeepEqual(backing[0], transaction{}) {
		t.Fatal("entry trim retained old selection state")
	}
	backing = b.undo[:cap(b.undo)]
	b.ResetHistory()
	for _, slot := range backing {
		if !reflect.DeepEqual(slot, transaction{}) {
			t.Fatal("reset retained history slots")
		}
	}
}
