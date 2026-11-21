package text

import (
	"errors"
	"math"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
)

func mustSetComposition(t *testing.T, b *Buffer) CompositionToken {
	t.Helper()
	token, err := b.BeginCompositionSetGroup()
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func compositionSet(t *testing.T, b *Buffer, ranges []Range, primary int) SelectionSet {
	t.Helper()
	if err := b.SetSelectionSet(SelectionSet{Ranges: ranges, Primary: primary}); err != nil {
		t.Fatal(err)
	}
	return b.SelectionSet()
}

func TestMultiCompositionExplicitBeginPreservesLegacyAndOwner(t *testing.T) {
	b := New("abcdef")
	before := compositionSet(t, b, []Range{{1, 0}, {4, 2}, {6, 5}}, 1)
	legacy := mustComposition(t, b)
	if err := b.ReplaceCompositionUTF16(legacy, 2, 4, "Q"); err != nil {
		t.Fatal(err)
	}
	if err := b.EndCompositionGroup(legacy); err != nil || b.String() != "abQef" || !b.Undo() || b.String() != "abcdef" {
		t.Fatal("legacy multi before-set End contract", err)
	}
	checkCompositionSet(t, b, before)
	token := mustSetComposition(t, b)
	unchanged := multiStateGuard(t, b)
	if other, err := b.BeginCompositionSetGroup(); !errors.Is(err, ErrGroup) || other != (CompositionToken{}) {
		t.Fatal("explicit begin stole active owner", err)
	}
	unchanged()
	if err := b.EndCompositionGroup(token); !errors.Is(err, ErrCompositionPending) {
		t.Fatal("explicit begin did not require native final", err)
	}
	unchanged()
	if err := b.CancelCompositionGroup(token); err != nil {
		t.Fatal(err)
	}
	if err := b.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	unchanged = multiStateGuard(t, b)
	if _, err := b.BeginCompositionSetGroup(); !errors.Is(err, ErrGroup) {
		t.Fatal("explicit begin stole ordinary group", err)
	}
	unchanged()
	if !b.CancelGroup() {
		t.Fatal("ordinary group lost")
	}
}

func TestMultiCompositionBeginRangeMustBeOriginalPrimary(t *testing.T) {
	b := New("abcdef")
	compositionSet(t, b, []Range{{0, 1}, {4, 2}, {5, 6}}, 1)
	token := mustSetComposition(t, b)
	unchanged := multiStateGuard(t, b)
	if err := b.ReplaceCompositionUTF16(token, 0, 1, "错"); err == nil {
		t.Fatal("multi preedit outside the original primary was accepted")
	}
	unchanged()
}

func TestMultiCompositionEndRequiresNativeFinalText(t *testing.T) {
	b := New("abcdef")
	compositionSet(t, b, []Range{{0, 1}, {4, 2}, {5, 6}}, 1)
	token := mustSetComposition(t, b)
	for _, preview := range []bool{false, true} {
		if preview {
			if err := b.ReplaceCompositionUTF16(token, 2, 4, "预览"); err != nil {
				t.Fatal(err)
			}
		}
		unchanged := multiStateGuard(t, b)
		if err := b.EndCompositionGroup(token); !errors.Is(err, ErrCompositionPending) {
			t.Fatal("multi End must keep the unconfirmed owner pending", err)
		}
		unchanged()
	}
	if err := b.CommitCompositionSetUTF16(token, "终"); err != nil || b.String() != "终b终e终" {
		t.Fatal("native final after pending End", err, b.String())
	}
}

// The behavior oracle uses independent UTF-16 slices and builds a new output;
// it never calls the production mapper, normalizer, batch or composition code.
func compositionOracle(source string, set SelectionSet, final string) (string, SelectionSet) {
	input, value := utf16.Encode([]rune(source)), utf16.Encode([]rune(final))
	var output []uint16
	after := SelectionSet{}
	previous := 0
	for i, r := range set.Ranges {
		start, end := r.Anchor, r.Head
		if start > end {
			start, end = end, start
		}
		output = append(output, input[previous:start]...)
		output = append(output, value...)
		caret := Range{len(output), len(output)}
		if len(after.Ranges) == 0 || after.Ranges[len(after.Ranges)-1] != caret {
			after.Ranges = append(after.Ranges, caret)
		}
		if i == set.Primary {
			after.Primary = len(after.Ranges) - 1
		}
		previous = end
	}
	output = append(output, input[previous:]...)
	return string(utf16.Decode(output)), after
}

func checkCompositionSet(t *testing.T, b *Buffer, want SelectionSet) {
	t.Helper()
	if got := b.SelectionSet(); !reflect.DeepEqual(got, want) || b.Selection() != want.Ranges[want.Primary] {
		t.Fatalf("selection got=%+v primary=%+v want=%+v", got, b.Selection(), want)
	}
}

func TestMultiCompositionIndependentUnicodeOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(25020261119))
	words := []string{"", "x", "汉字", "🙂", "e\u0301", "👨‍👩‍👧‍👦", "\r\n", "🇨🇳"}
	for trial := 0; trial < 210; trial++ {
		n := 2 + rng.Intn(7)
		var source strings.Builder
		before := SelectionSet{Primary: trial % n}
		for i := 0; i < n; i++ {
			source.WriteString("|")
			start := len(utf16.Encode([]rune(source.String())))
			if trial%4 != 0 || i%2 == 0 {
				source.WriteString(words[1+rng.Intn(len(words)-1)])
			}
			end := len(utf16.Encode([]rune(source.String())))
			r := Range{start, end}
			if (trial+i)%2 == 0 {
				r = Range{end, start}
			}
			before.Ranges = append(before.Ranges, r)
		}
		source.WriteString("尾🙂")
		seed := source.String()
		b := New(seed)
		compositionSet(t, b, before.Ranges, before.Primary)
		checkCompositionSet(t, b, before)
		token := mustSetComposition(t, b)
		beforeStorage := &b.group.beforeSelections.secondary[0]
		primary := before.Ranges[before.Primary]
		start, end := primary.Start(), primary.End()
		// Zero, one and repeated short/long/empty/same previews all confirm
		// against the original selection authority, not live primary carets.
		for round := 0; round < trial%6; round++ {
			next := words[(trial+round)%len(words)]
			if err := b.ReplaceCompositionUTF16(token, start, end, next); err != nil {
				t.Fatalf("trial%d round%d: %v", trial, round, err)
			}
			wantPreview, _ := compositionOracle(seed, SelectionSet{Ranges: []Range{primary}}, next)
			if b.String() != wantPreview || &b.group.beforeSelections.secondary[0] != beforeStorage || len(b.group.edits) > 1 || cap(b.group.edits) > 1 {
				t.Fatalf("trial%d preview changed secondary/storage/coalescing", trial)
			}
			end = start + len(utf16.Encode([]rune(next)))
			// Match native selected-in-marked updates; this must not replace
			// or copy the token's complete starting selection authority.
			if err := b.SetSelection(end, start); err != nil {
				t.Fatal(err)
			}
		}
		final := words[(trial*3+1)%len(words)]
		want, after := compositionOracle(seed, before, final)
		if err := b.CommitCompositionSetUTF16(token, final); err != nil {
			t.Fatalf("trial%d commit: %v", trial, err)
		}
		if b.String() != want || b.LenBytes() != len(want) || b.LenUTF16() != len(utf16.Encode([]rune(want))) || b.InGroup() || b.OwnsComposition(token) {
			t.Fatalf("trial%d got=%q want=%q", trial, b.String(), want)
		}
		checkCompositionSet(t, b, after)
		if want == seed {
			if b.CanUndo() || b.History().UndoDepth != 0 {
				t.Fatal("net-zero invented history", trial)
			}
			continue
		}
		if b.History().UndoDepth != 1 || len(b.undo[0].edits) > n || cap(b.undo[0].edits) > n {
			t.Fatal("confirmation not one bounded history item", trial)
		}
		if !b.Undo() || b.String() != seed || b.CanUndo() {
			t.Fatal("whole composition undo", trial)
		}
		checkCompositionSet(t, b, before)
		if !b.Redo() || b.String() != want || b.CanRedo() {
			t.Fatal("whole composition redo", trial)
		}
		checkCompositionSet(t, b, after)
	}
}

func TestMultiCompositionAdjacentBoundaryCaretsAndPrimaryMapping(t *testing.T) {
	for _, tc := range []struct {
		name    string
		ranges  []Range
		primary int
		before  SelectionSet
	}{
		{"adjacent-left", []Range{{2, 0}, {4, 2}, {6, 4}}, 0, SelectionSet{[]Range{{2, 0}, {4, 2}, {6, 4}}, 0}},
		{"adjacent-middle", []Range{{2, 0}, {2, 4}, {6, 4}}, 1, SelectionSet{[]Range{{2, 0}, {2, 4}, {6, 4}}, 1}},
		{"adjacent-right", []Range{{2, 0}, {4, 2}, {6, 4}}, 2, SelectionSet{[]Range{{2, 0}, {4, 2}, {6, 4}}, 2}},
		// The shared-boundary primary caret belongs to the right interval.
		{"boundary-caret", []Range{{2, 0}, {2, 2}, {4, 2}, {5, 6}}, 1, SelectionSet{[]Range{{2, 0}, {4, 2}, {5, 6}}, 1}},
		{"primary-caret", []Range{{0, 1}, {3, 3}, {6, 4}}, 1, SelectionSet{[]Range{{0, 1}, {3, 3}, {6, 4}}, 1}},
	} {
		for _, final := range []string{"", "🙂", "longer"} {
			t.Run(tc.name+"/"+final, func(t *testing.T) {
				b := New("abcdef")
				compositionSet(t, b, tc.ranges, tc.primary)
				checkCompositionSet(t, b, tc.before)
				token := mustSetComposition(t, b)
				p := tc.before.Ranges[tc.before.Primary]
				if err := b.ReplaceCompositionUTF16(token, p.Start(), p.End(), "预🙂览"); err != nil {
					t.Fatal(err)
				}
				want, after := compositionOracle("abcdef", tc.before, final)
				if err := b.CommitCompositionSetUTF16(token, final); err != nil || b.String() != want {
					t.Fatal("confirm", err, b.String(), want)
				}
				checkCompositionSet(t, b, after)
				if !b.Undo() || b.String() != "abcdef" {
					t.Fatal("undo")
				}
				checkCompositionSet(t, b, tc.before)
				if !b.Redo() || b.String() != want {
					t.Fatal("redo")
				}
				checkCompositionSet(t, b, after)
			})
		}
	}
}

func TestMultiCompositionInvalidPreeditAndFinalPreserveEveryField(t *testing.T) {
	b := NewWithLimit("a🙂bc", 32)
	compositionSet(t, b, []Range{{0, 1}, {3, 1}, {4, 5}}, 1)
	token := mustSetComposition(t, b)
	for _, accepted := range []bool{false, true} {
		if accepted {
			if err := b.ReplaceCompositionUTF16(token, 1, 3, "汉🙂"); err != nil {
				t.Fatal(err)
			}
		}
		for _, candidate := range []struct {
			start, end int
			value      string
			want       error
		}{
			{-1, 1, "x", ErrRange}, {0, math.MaxInt, "x", ErrRange},
			{3, 1, "x", ErrRange}, {0, 1, "x", ErrRange},
			{2, 3, "x", map[bool]error{false: ErrSurrogate, true: ErrSurrogate}[accepted]},
			{1, 2, "x", map[bool]error{false: ErrSurrogate, true: ErrRange}[accepted]},
			{1, 3, "\xff", ErrUTF8},
		} {
			unchanged := multiStateGuard(t, b)
			if err := b.ReplaceCompositionUTF16(token, candidate.start, candidate.end, candidate.value); !errors.Is(err, candidate.want) {
				t.Fatalf("accepted=%v range=%d..%d: %v want %v", accepted, candidate.start, candidate.end, err, candidate.want)
			}
			unchanged()
		}
		unchanged := multiStateGuard(t, b)
		if err := b.CommitCompositionSetUTF16(token, "\xff"); !errors.Is(err, ErrUTF8) {
			t.Fatal(err)
		}
		unchanged()
		if err := b.EndCompositionGroup(token); !errors.Is(err, ErrCompositionPending) {
			t.Fatal(err)
		}
		unchanged()
	}
	if err := b.CancelCompositionGroup(token); err != nil || b.String() != "a🙂bc" || b.CanUndo() {
		t.Fatal("cancel", err)
	}
	checkCompositionSet(t, b, SelectionSet{[]Range{{0, 1}, {3, 1}, {4, 5}}, 1})
}

func TestMultiCompositionBudgetFinalIntermediateRetryAndCancel(t *testing.T) {
	for _, tc := range []struct {
		name, source, preedit, final, retry string
		ranges                              []Range
		primary, limit, exact               int
	}{
		{"aggregate-final", "a|b", "x", "中", "zz", []Range{{1, 0}, {2, 3}}, 0, 6, 7},
		{"primary-first-intermediate", "long|x", "", "yy", "y", []Range{{4, 0}, {6, 5}}, 1, 6, 7},
		{"secondary-descending-intermediate", "a|long|x", "", "zz", "z", []Range{{1, 0}, {6, 2}, {7, 8}}, 0, 9, 10},
		{"net-zero-intermediate", "aaaa", "", "aa", "a", []Range{{3, 0}, {4, 3}}, 1, 4, 5},
	} {
		for _, finish := range []string{"cancel", "retry", "exact", "above-exact"} {
			t.Run(tc.name+"/"+finish, func(t *testing.T) {
				limit := tc.limit
				if finish == "exact" {
					limit = tc.exact
				}
				if finish == "above-exact" {
					limit = tc.exact + 1
				}
				b := NewWithLimit(tc.source, limit)
				// Keep an existing redo endpoint throughout every refusal and
				// cancellation. The fixture deletes before undo, so it fits.
				if err := b.ReplaceUTF16(0, 1, ""); err != nil || !b.Undo() {
					t.Fatal("redo fixture", err)
				}
				before := compositionSet(t, b, tc.ranges, tc.primary)
				history := b.History()
				token := mustSetComposition(t, b)
				p := before.Ranges[before.Primary]
				if err := b.ReplaceCompositionUTF16(token, p.Start(), p.End(), tc.preedit); err != nil {
					t.Fatal(err)
				}
				if finish == "cancel" || finish == "retry" {
					for _, tooLarge := range []string{tc.final, strings.Repeat("🙂", 20), tc.final} {
						unchanged := multiStateGuard(t, b)
						if err := b.CommitCompositionSetUTF16(token, tooLarge); !errors.Is(err, ErrSizeLimit) {
							t.Fatal("budget accepted", err)
						}
						unchanged()
					}
					if finish == "cancel" {
						if err := b.CancelCompositionGroup(token); err != nil || b.String() != tc.source || b.History() != history || !b.CanRedo() {
							t.Fatal("cancel after refusal", err)
						}
						checkCompositionSet(t, b, before)
						return
					}
					// A valid shorter preedit may follow a failed confirmation.
					span := b.composition.span
					if err := b.ReplaceCompositionUTF16(token, span.Start(), span.End(), ""); err != nil {
						t.Fatal("preedit retry", err)
					}
				}
				final := tc.final
				if finish == "retry" {
					final = tc.retry
				}
				want, after := compositionOracle(tc.source, before, final)
				if err := b.CommitCompositionSetUTF16(token, final); err != nil || b.String() != want {
					t.Fatal("legal retry/exact", err, b.String(), want)
				}
				checkCompositionSet(t, b, after)
				if want == tc.source {
					if b.History() != history || !b.CanRedo() {
						t.Fatal("net-zero erased redo")
					}
				} else {
					if b.History().UndoDepth != 1 || b.CanRedo() || !b.Undo() || b.String() != tc.source {
						t.Fatal("one undo after retry")
					}
					checkCompositionSet(t, b, before)
					if !b.Redo() || b.String() != want {
						t.Fatal("redo after retry")
					}
					checkCompositionSet(t, b, after)
				}
			})
		}
	}
}

func TestMultiCompositionTokensForeignStaleShallowAndReset(t *testing.T) {
	makeBuffer := func() (*Buffer, CompositionToken) {
		b := New("a|b")
		compositionSet(t, b, []Range{{1, 0}, {2, 3}}, 1)
		return b, mustSetComposition(t, b)
	}
	a, ta := makeBuffer()
	b, tb := makeBuffer()
	assertRefused := func(target *Buffer, token CompositionToken) {
		t.Helper()
		unchanged := multiStateGuard(t, target)
		if err := target.CommitCompositionSetUTF16(token, "x"); !errors.Is(err, ErrComposition) {
			t.Fatal("foreign/stale confirm", err)
		}
		unchanged()
	}
	assertRefused(a, tb)
	assertRefused(b, ta)
	assertRefused(a, CompositionToken{})
	shallow := *a
	assertRefused(&shallow, ta)
	for _, reset := range []string{"reset", "restore", "cancel", "commit"} {
		t.Run(reset, func(t *testing.T) {
			b, old := makeBuffer()
			if err := b.ReplaceCompositionUTF16(old, 2, 3, "🙂"); err != nil {
				t.Fatal(err)
			}
			switch reset {
			case "reset":
				b.ResetHistory()
			case "restore":
				b.Restore(New("a|b").Snapshot())
			case "cancel":
				if err := b.CancelCompositionGroup(old); err != nil {
					t.Fatal(err)
				}
			case "commit":
				if err := b.CommitCompositionSetUTF16(old, "x"); err != nil {
					t.Fatal(err)
				}
			}
			assertRefused(b, old)
			if err := b.BeginGroup(); err != nil {
				t.Fatal(err)
			}
			assertRefused(b, old)
			if !b.CancelGroup() {
				t.Fatal("ordinary group lost")
			}
			fresh := mustSetComposition(t, b)
			assertRefused(b, old)
			if err := b.CancelCompositionGroup(fresh); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, tokenless := range []func() error{
		func() error { return a.ReplaceUTF16(0, 1, "x") },
		func() error {
			return a.ReplaceBatchUTF16(a.Version(), a.SelectionGeneration(), []Replacement{{0, 1, "x"}}, 0)
		},
		func() error { return a.SetSelectionSet(SelectionSet{[]Range{{0, 0}}, 0}) },
	} {
		unchanged := multiStateGuard(t, a)
		if err := tokenless(); !errors.Is(err, ErrGroup) {
			t.Fatal("tokenless hijack", err)
		}
		unchanged()
	}
	if a.EndGroup() || a.CancelGroup() {
		t.Fatal("tokenless end/cancel")
	}
	_ = a.CancelCompositionGroup(ta)
	_ = b.CancelCompositionGroup(tb)
}

func TestMultiCompositionSameTextAndNetZeroPreserveRedo(t *testing.T) {
	for _, tc := range []struct {
		name, source, final string
		ranges              []Range
	}{
		{"same-content", "a|a", "a", []Range{{1, 0}, {2, 3}}},
		{"redistributed", "aaaa", "aa", []Range{{1, 0}, {4, 1}}},
		{"piece-stream", strings.Repeat("a", 13000), strings.Repeat("a", 6500), []Range{{1, 0}, {13000, 1}}},
	} {
		for primary := 0; primary < 2; primary++ {
			for rounds := 0; rounds < 3; rounds++ {
				t.Run(tc.name+string(rune('0'+primary))+string(rune('0'+rounds)), func(t *testing.T) {
					b := New(tc.source)
					_ = b.ReplaceUTF16(0, 1, "")
					if !b.Undo() {
						t.Fatal("redo fixture")
					}
					before := compositionSet(t, b, tc.ranges, primary)
					history := b.History()
					token := mustSetComposition(t, b)
					p := before.Ranges[primary]
					start, end := p.Start(), p.End()
					for i := 0; i < rounds; i++ {
						value := "预🙂览"
						if i == 1 {
							value = string(utf16.Decode(utf16.Encode([]rune(tc.source))[p.Start():p.End()]))
						}
						if err := b.ReplaceCompositionUTF16(token, start, end, value); err != nil {
							t.Fatal(err)
						}
						end = start + len(utf16.Encode([]rune(value)))
					}
					version, journal, lines := b.Version(), b.rangeChanges, b.changedLines
					want, after := compositionOracle(tc.source, before, tc.final)
					if want != tc.source {
						t.Fatal("invalid net-zero oracle fixture")
					}
					if err := b.CommitCompositionSetUTF16(token, tc.final); err != nil || b.String() != tc.source || b.History() != history || !b.CanRedo() {
						t.Fatal("net-zero commit", err, b.History(), history)
					}
					if rounds != 1 && (b.Version() != version || b.rangeChanges != journal || b.changedLines != lines) {
						t.Fatal("same live bytes invented revision/journal")
					}
					checkCompositionSet(t, b, after)
					if !b.Redo() || b.String() != tc.source[1:] {
						t.Fatal("redo destroyed")
					}
				})
			}
		}
	}
}

func TestMultiCompositionFinalSameAsLiveKeepsOnePreviewHistory(t *testing.T) {
	b := New("a|b|a")
	before := compositionSet(t, b, []Range{{1, 0}, {3, 2}, {5, 4}}, 1)
	token := mustSetComposition(t, b)
	if err := b.ReplaceCompositionUTF16(token, 2, 3, "a"); err != nil {
		t.Fatal(err)
	}
	version, journal, lines := b.Version(), b.rangeChanges, b.changedLines
	if err := b.CommitCompositionSetUTF16(token, "a"); err != nil || b.String() != "a|a|a" {
		t.Fatal(err)
	}
	if b.Version() != version || b.rangeChanges != journal || b.changedLines != lines || b.History().UndoDepth != 1 || len(b.undo[0].edits) != 1 {
		t.Fatal("same-live confirmation rewrote bytes or lost original history")
	}
	checkCompositionSet(t, b, SelectionSet{[]Range{{1, 1}, {3, 3}, {5, 5}}, 1})
	if !b.Undo() || b.String() != "a|b|a" {
		t.Fatal("preview endpoint undo")
	}
	checkCompositionSet(t, b, before)
	if !b.Redo() || b.String() != "a|a|a" {
		t.Fatal("preview endpoint redo")
	}
	checkCompositionSet(t, b, SelectionSet{[]Range{{1, 1}, {3, 3}, {5, 5}}, 1})
}

func TestMultiCompositionNoOpPreeditDoesNotCopySelectionSlice(t *testing.T) {
	b := New(strings.Repeat("a|", 256))
	ranges := make([]Range, 256)
	for i := range ranges {
		ranges[i] = Range{2 * i, 2*i + 1}
	}
	before := compositionSet(t, b, ranges, 128)
	token := mustSetComposition(t, b)
	if err := b.ReplaceCompositionUTF16(token, 256, 257, "a"); err != nil {
		t.Fatal(err)
	}
	storage := &b.group.beforeSelections.secondary[0]
	if allocs := testing.AllocsPerRun(1000, func() {
		if err := b.ReplaceCompositionUTF16(token, 256, 257, "a"); err != nil {
			panic(err)
		}
	}); allocs != 0 {
		t.Fatal("no-op multi preedit copied retained selections", allocs)
	}
	if &b.group.beforeSelections.secondary[0] != storage {
		t.Fatal("before-set storage replaced")
	}
	if err := b.CancelCompositionGroup(token); err != nil {
		t.Fatal(err)
	}
	checkCompositionSet(t, b, before)
}

func TestMultiCompositionHistoryAdmissionAtomicRetryAndClear(t *testing.T) {
	// Three old/new one-byte fragments plus four retained Range values. Use
	// independent byte/struct-size arithmetic, not production accounting.
	retained := 6 + 4*int(reflect.TypeOf(Range{}).Size())
	for _, tc := range []struct {
		name            string
		entries, budget int
	}{
		{"zero-entries", 0, retained}, {"zero-bytes", 1, 0},
		{"below", 1, retained - 1}, {"exact", 1, retained}, {"plus-one", 1, retained + 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := New("a|b|c")
			if err := b.ReplaceUTF16(0, 1, ""); err != nil || !b.Undo() {
				t.Fatal("redo fixture", err)
			}
			b.SetHistoryLimits(tc.entries, tc.budget)
			before := compositionSet(t, b, []Range{{1, 0}, {2, 3}, {5, 4}}, 1)
			history := b.History()
			token := mustSetComposition(t, b)
			if err := b.ReplaceCompositionUTF16(token, 2, 3, "preview"); err != nil {
				t.Fatal(err)
			}
			oldEdits := b.group.edits[:cap(b.group.edits)]
			final, expectedRetained := "x", retained
			if tc.entries == 0 || tc.budget < retained {
				unchanged := multiStateGuard(t, b)
				if err := b.CommitCompositionSetUTF16(token, final); !errors.Is(err, ErrCompositionHistoryLimit) {
					t.Fatal("multi confirmation did not refuse its unretainable undo transaction", err)
				}
				unchanged()
				if err := b.EndCompositionGroup(token); !errors.Is(err, ErrCompositionPending) {
					t.Fatal("history refusal bypassed by End", err)
				}
				unchanged()
				if tc.entries == 0 || tc.budget == 0 {
					if err := b.CancelCompositionGroup(token); err != nil || b.String() != "a|b|c" || b.History() != history {
						t.Fatal("history refusal cancel", err)
					}
					checkCompositionSet(t, b, before)
					return
				}
				// The same unchanged small history budget admits a shorter final
				// confirmation. Do not increase it to hide the original refusal.
				final, expectedRetained = "", retained-3
			}
			want, after := compositionOracle("a|b|c", before, final)
			if err := b.CommitCompositionSetUTF16(token, final); err != nil || b.String() != want {
				t.Fatal("history exact/retry", err)
			}
			for _, dead := range oldEdits {
				if dead != (change{}) {
					t.Fatal("accepted preedit retained in dead group slots")
				}
			}
			if b.History().UndoDepth != 1 || b.History().RetainedBytes != expectedRetained || !b.Undo() || b.String() != "a|b|c" {
				t.Fatal("retained accounting/undo", b.History(), expectedRetained)
			}
			checkCompositionSet(t, b, before)
			if !b.Redo() || b.String() != want {
				t.Fatal("retained redo")
			}
			checkCompositionSet(t, b, after)
			backing := b.undo[:cap(b.undo)]
			b.SetHistoryLimits(0, 0)
			for _, dead := range backing {
				if !reflect.DeepEqual(dead, transaction{}) {
					t.Fatal("trim retained removed transaction endpoints")
				}
			}
		})
	}
}

func TestMultiCompositionDefaultHistoryRejectsBeforeWritingAndShorterRetry(t *testing.T) {
	originalPrimary := strings.Repeat("a", 17<<20)
	source := originalPrimary + "|b"
	b := New(source)
	before := compositionSet(t, b, []Range{{17 << 20, 0}, {(17 << 20) + 1, (17 << 20) + 2}}, 0)
	token := mustSetComposition(t, b)
	if err := b.ReplaceCompositionUTF16(token, 0, 17<<20, "p"); err != nil {
		t.Fatal(err)
	}
	unchanged := multiStateGuard(t, b)
	if err := b.CommitCompositionSetUTF16(token, strings.Repeat("z", 17<<20)); !errors.Is(err, ErrCompositionHistoryLimit) {
		t.Fatal("default 32MiB history must atomically refuse 51MiB multi undo", err)
	}
	unchanged()
	if err := b.CommitCompositionSetUTF16(token, "z"); err != nil || b.String() != "z|z" || !b.CanUndo() {
		t.Fatal("default-budget shorter retry", err)
	}
	if !b.Undo() || b.String() != source {
		t.Fatal("default-budget complete undo")
	}
	checkCompositionSet(t, b, before)
	if !b.Redo() || b.String() != "z|z" {
		t.Fatal("default-budget redo")
	}
	checkCompositionSet(t, b, SelectionSet{[]Range{{1, 1}, {3, 3}}, 0})
}

func TestMultiCompositionNetZeroNeedsNoHistoryBudget(t *testing.T) {
	for _, tc := range []struct {
		source, final string
		ranges        []Range
	}{
		{"a|a", "a", []Range{{1, 0}, {2, 3}}},
		{"aaaa", "aa", []Range{{1, 0}, {4, 1}}},
	} {
		b := New(tc.source)
		b.SetHistoryLimits(0, 0)
		before := compositionSet(t, b, tc.ranges, 0)
		history := b.History()
		token := mustSetComposition(t, b)
		if err := b.ReplaceCompositionUTF16(token, before.Ranges[0].Start(), before.Ranges[0].End(), "预览"); err != nil {
			t.Fatal(err)
		}
		want, after := compositionOracle(tc.source, before, tc.final)
		if err := b.CommitCompositionSetUTF16(token, tc.final); err != nil || b.String() != want || b.History() != history || b.CanUndo() {
			t.Fatal("net-zero should need no undo budget", err)
		}
		checkCompositionSet(t, b, after)
	}
}

func TestMultiCompositionHistoryIntegerAdmission(t *testing.T) {
	b := New("")
	b.SetHistoryLimits(1, math.MaxInt)
	oneRange := int(reflect.TypeOf(Range{}).Size())
	for _, tc := range []struct {
		name  string
		value transaction
		want  bool
	}{
		{"exact-max", transaction{bytes: math.MaxInt}, true},
		{"saturated-edits", transaction{bytes: math.MaxInt, bytesOverflow: true}, false},
		{"selection-overflow", transaction{bytes: math.MaxInt, beforeSelections: selectionExtra{secondary: make([]Range, 1)}}, false},
		{"exact-with-selection", transaction{bytes: math.MaxInt - oneRange, afterSelections: selectionExtra{secondary: make([]Range, 1)}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := b.admitsCompositionHistory(tc.value); got != tc.want {
				t.Fatal("history integer admission", got, tc.want)
			}
		})
	}
	b.SetHistoryLimits(0, math.MaxInt)
	if b.admitsCompositionHistory(transaction{bytes: 1}) {
		t.Fatal("zero entries admitted history")
	}
}

func TestMultiCompositionHistoryMayTrimOlderItemsButKeepsAcceptedItem(t *testing.T) {
	b := New("a|b")
	if err := b.ReplaceUTF16(1, 2, "-"); err != nil {
		t.Fatal(err)
	}
	if err := b.ReplaceUTF16(1, 2, "="); err != nil || !b.Undo() {
		t.Fatal("old undo/redo fixture", err)
	}
	before := compositionSet(t, b, []Range{{1, 0}, {2, 3}}, 1)
	retained := 4 + 2*int(reflect.TypeOf(Range{}).Size())
	b.SetHistoryLimits(2, retained)
	if len(b.undo) != 1 || cap(b.undo) < 2 || len(b.redo) != 1 {
		t.Fatal("history backing fixture")
	}
	oldSlot, oldRedo := &b.undo[0], b.redo[:cap(b.redo)]
	token := mustSetComposition(t, b)
	if err := b.CommitCompositionSetUTF16(token, "x"); err != nil || b.String() != "x-x" {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*oldSlot, transaction{}) {
		t.Fatal("trim retained older item in backing capacity")
	}
	for _, dead := range oldRedo {
		if !reflect.DeepEqual(dead, transaction{}) {
			t.Fatal("accepted commit retained old redo item")
		}
	}
	if !b.History().Truncated || b.History().UndoDepth != 1 || b.History().RetainedBytes != retained || !b.Undo() || b.String() != "a-b" {
		t.Fatal("accepted item did not survive older trim", b.History())
	}
	checkCompositionSet(t, b, before)
	if !b.Redo() || b.String() != "x-x" {
		t.Fatal("accepted item redo")
	}
}

func TestMultiCompositionSingleSetStillUsesLegacyHistoryTrim(t *testing.T) {
	for _, budget := range []int{1, 2} {
		b := New("x")
		b.SetHistoryLimits(1, budget)
		if err := b.SetSelection(1, 0); err != nil {
			t.Fatal(err)
		}
		token := mustSetComposition(t, b)
		if err := b.CommitCompositionSetUTF16(token, "y"); err != nil || b.String() != "y" {
			t.Fatal("single set commit", err)
		}
		if b.CanUndo() != (budget == 2) || b.History().Truncated != (budget == 1) {
			t.Fatal("single legacy trimming changed", budget, b.History())
		}
	}
}

func TestMultiComposition256AndNormalized257Refusal(t *testing.T) {
	b := New(strings.Repeat("a|", 256))
	ranges := make([]Range, 256)
	for i := range ranges {
		ranges[i] = Range{2*i + 1, 2 * i}
	}
	before := compositionSet(t, b, ranges, 127)
	token := mustSetComposition(t, b)
	if err := b.ReplaceCompositionUTF16(token, 254, 255, "预🙂览"); err != nil {
		t.Fatal(err)
	}
	want, after := compositionOracle(strings.Repeat("a|", 256), before, "🙂")
	if err := b.CommitCompositionSetUTF16(token, "🙂"); err != nil || b.String() != want || len(b.SelectionSet().Ranges) != 256 || cap(b.undo[0].edits) > 256 {
		t.Fatal("256 commit", err)
	}
	checkCompositionSet(t, b, after)
	if !b.Undo() || b.String() != strings.Repeat("a|", 256) {
		t.Fatal("256 undo")
	}
	checkCompositionSet(t, b, before)
	if !b.Redo() || b.String() != want {
		t.Fatal("256 redo")
	}
	checkCompositionSet(t, b, after)
	x := New(strings.Repeat("a|", 257))
	tooMany := make([]Range, 257)
	for i := range tooMany {
		tooMany[i] = Range{2 * i, 2*i + 1}
	}
	unchanged := multiStateGuard(t, x)
	if err := x.SetSelectionSet(SelectionSet{tooMany, 128}); !errors.Is(err, ErrSelectionLimit) {
		t.Fatal("already ordered 257 accepted", err)
	}
	unchanged()
}

// Independently enumerate byte budgets from lengths, including every primary
// position and accepted empty/long preedit. This does not reuse the production
// plan, mapped offsets, size checker or history accounting as an oracle.
func TestMultiCompositionIndependentBudgetSequenceOracle(t *testing.T) {
	trials, refused := 0, 0
	for _, left := range []int{0, 1, 3} {
		for _, middle := range []int{0, 1, 3} {
			for _, right := range []int{0, 1, 3} {
				lengths := []int{left, middle, right}
				source := strings.Repeat("a", left) + "|" + strings.Repeat("b", middle) + "|" + strings.Repeat("c", right)
				ranges := []Range{{left, 0}, {left + 1, left + 1 + middle}, {len(source), left + middle + 2}}
				for primary := 0; primary < 3; primary++ {
					for _, preview := range []string{"", "p", "预🙂"} {
						for _, final := range []string{"", "z", "xy", "🙂"} {
							for _, allowance := range []int{0, 1, 5} {
								limit := len(source) + allowance
								liveBytes := len(source) - lengths[primary] + len(preview)
								if liveBytes > limit {
									continue
								}
								b := NewWithLimit(source, limit)
								before := compositionSet(t, b, ranges, primary)
								token := mustSetComposition(t, b)
								p := ranges[primary]
								if err := b.ReplaceCompositionUTF16(token, p.Start(), p.End(), preview); err != nil {
									t.Fatal("admitted oracle preview", err)
								}
								peak := len(source) - lengths[primary] + len(final)
								current := peak
								for i := 2; i >= 0; i-- {
									if i != primary {
										current += len(final) - lengths[i]
										if current > peak {
											peak = current
										}
									}
								}
								unchanged := multiStateGuard(t, b)
								err := b.CommitCompositionSetUTF16(token, final)
								trials++
								if peak > limit {
									refused++
									if !errors.Is(err, ErrSizeLimit) {
										t.Fatalf("budget oracle lengths=%v primary=%d preview=%q final=%q limit=%d peak=%d: %v", lengths, primary, preview, final, limit, peak, err)
									}
									unchanged()
									if err := b.CancelCompositionGroup(token); err != nil || b.String() != source {
										t.Fatal("refused oracle cancel", err)
									}
									checkCompositionSet(t, b, before)
									continue
								}
								want, after := compositionOracle(source, before, final)
								if err != nil || b.String() != want {
									t.Fatal("budget oracle valid final", err, b.String(), want)
								}
								checkCompositionSet(t, b, after)
								if want != source {
									if !b.Undo() || b.String() != source {
										t.Fatal("budget oracle undo")
									}
									checkCompositionSet(t, b, before)
								} else if b.CanUndo() {
									t.Fatal("budget oracle no-op history")
								}
							}
						}
					}
				}
			}
		}
	}
	t.Logf("independent admitted-preedit budget cases=%d refusals=%d", trials, refused)
}

func TestMultiCompositionSinglePointOldContractsAndZeroAllocSelection(t *testing.T) {
	b := New("a🙂z")
	if err := b.SetSelection(3, 1); err != nil {
		t.Fatal(err)
	}
	token := mustSetComposition(t, b)
	if err := b.ReplaceCompositionUTF16(token, 1, 3, "x"); err != nil {
		t.Fatal(err)
	}
	// Existing single-point irregular spans remain accepted and grouped.
	if err := b.ReplaceCompositionUTF16(token, 0, 1, "A"); err != nil {
		t.Fatal(err)
	}
	if err := b.CommitCompositionSetUTF16(token, "B"); err != nil || b.String() != "Bxz" {
		t.Fatal("single irregular final", err)
	}
	if !b.Undo() || b.String() != "a🙂z" || b.Selection() != (Range{3, 1}) || !b.Redo() || b.String() != "Bxz" {
		t.Fatal("single old undo/redo")
	}
	token = mustSetComposition(t, b)
	if err := b.EndCompositionGroup(token); err != nil {
		t.Fatal("legacy single End", err)
	}
	if allocs := testing.AllocsPerRun(1000, func() {
		r := b.Selection()
		if err := b.SetSelection(r.Anchor, r.Head); err != nil {
			panic(err)
		}
	}); allocs != 0 {
		t.Fatal("legacy selection allocated", allocs)
	}
	for _, tc := range []struct {
		base, offset, boundary int
		ok                     bool
	}{
		{math.MaxInt - 1, 1, 0, true}, {math.MaxInt, 1, 0, false}, {1, 0, 1, false}, {-1, 0, 0, false},
	} {
		if _, ok := compositionOffset(tc.base, tc.offset, tc.boundary); ok != tc.ok {
			t.Fatal("integer guard", tc, ok)
		}
	}
}
