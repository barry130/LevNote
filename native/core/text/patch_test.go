package text

import (
	"errors"
	"math"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"
)

func patchUnits(value string) int { return len(utf16.Encode([]rune(value))) }

// Independent rune/byte oracle: no Buffer slicing, patch plan or stream helper.
func patchOracle(t *testing.T, source string, edits []PatchEdit) string {
	t.Helper()
	byteAt := map[int]int{0: 0}
	units := 0
	for index, r := range source {
		byteAt[units] = index
		units += len(utf16.Encode([]rune{r}))
	}
	byteAt[units] = len(source)
	var out strings.Builder
	previous := 0
	for _, edit := range edits {
		left, leftOK := byteAt[edit.Start]
		right, rightOK := byteAt[edit.End]
		if !leftOK || !rightOK || left < previous || source[left:right] != edit.OldText {
			t.Fatal("invalid independent oracle plan", edit)
		}
		out.WriteString(source[previous:left])
		out.WriteString(edit.NewText)
		previous = right
	}
	out.WriteString(source[previous:])
	return out.String()
}

// Every accepted changing fixture proves exactly one transaction and both full
// directed selection endpoints. supplied and want may differ by normalization.
func checkPatchRoundTrip(t *testing.T, b *Buffer, edits []PatchEdit, supplied, want SelectionSet) {
	t.Helper()
	source, before, history := b.String(), b.SelectionSet(), b.History()
	expected := patchOracle(t, source, edits)
	if source == expected {
		t.Fatal("changing fixture is byte-equivalent")
	}
	changed, err := b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), edits, supplied)
	if err != nil || !changed {
		t.Fatalf("patch refused: changed=%t err=%v", changed, err)
	}
	if b.String() != expected || b.LenBytes() != len(expected) || b.LenUTF16() != patchUnits(expected) || b.History().UndoDepth != min(history.UndoDepth+1, b.historyEntries) || b.CanRedo() {
		t.Fatalf("patch result/history: got=%q want=%q history=%+v", b.String(), expected, b.History())
	}
	requireSelections(t, b, want)
	if !b.Undo() || b.String() != source {
		t.Fatal("one patch Undo failed")
	}
	requireSelections(t, b, before)
	if !b.Redo() || b.String() != expected || b.CanRedo() {
		t.Fatal("one patch Redo failed")
	}
	requireSelections(t, b, want)
}

func TestPatchUtf16AtomicOldTextAndBudgetAdmission(t *testing.T) {
	b := NewWithLimit("甲🙂abcdef", 20)
	if err := b.ReplaceUTF16(b.LenUTF16(), b.LenUTF16(), "!"); err != nil || !b.Undo() {
		t.Fatal("redo fixture", err)
	}
	mustSetSelections(t, b, []Range{{3, 1}, {4, 6}}, 1)
	validAfter := SelectionSet{Ranges: []Range{{0, 0}}}
	for _, tc := range []struct {
		name  string
		edits []PatchEdit
		after SelectionSet
		want  error
	}{
		{"late-old-mismatch", []PatchEdit{{0, 1, "甲", "x"}, {4, 5, "wrong", "Y"}}, validAfter, ErrPatchOldText},
		{"old-length", []PatchEdit{{0, 1, "", "x"}}, validAfter, ErrPatchOldText},
		{"utf8-later", []PatchEdit{{0, 1, "甲", "x"}, {4, 5, "b", "\xff"}}, validAfter, ErrUTF8},
		{"utf8-old", []PatchEdit{{0, 1, "\xff", "x"}}, validAfter, ErrUTF8},
		{"negative", []PatchEdit{{-1, 1, "", "x"}}, validAfter, ErrRange},
		{"huge", []PatchEdit{{0, math.MaxInt, "", "x"}}, validAfter, ErrRange},
		{"reverse", []PatchEdit{{4, 3, "", "x"}}, validAfter, ErrRange},
		{"surrogate-start", []PatchEdit{{2, 3, "", "x"}}, validAfter, ErrSurrogate},
		{"surrogate-end", []PatchEdit{{1, 2, "", "x"}}, validAfter, ErrSurrogate},
		{"order", []PatchEdit{{4, 5, "b", "x"}, {0, 1, "甲", "y"}}, validAfter, ErrEditOrder},
		{"overlap", []PatchEdit{{0, 4, "甲🙂a", "x"}, {3, 5, "ab", "y"}}, validAfter, ErrEditOrder},
		{"duplicate-empty", []PatchEdit{{3, 3, "", "x"}, {3, 3, "", "y"}}, validAfter, ErrEditOrder},
		{"empty-left-touch", []PatchEdit{{3, 3, "", "x"}, {3, 5, "ab", "y"}}, validAfter, ErrEditOrder},
		{"empty-right-touch", []PatchEdit{{0, 3, "甲🙂", "x"}, {3, 3, "", "y"}}, validAfter, ErrEditOrder},
		{"final-limit", []PatchEdit{{0, 1, "甲", strings.Repeat("x", 21)}}, validAfter, ErrSizeLimit},
		{"1025", make([]PatchEdit, 1025), validAfter, ErrPatchLimit},
		{"empty-selection", nil, SelectionSet{}, ErrSelectionSet},
		{"negative-primary", nil, SelectionSet{[]Range{{0, 0}}, -1}, ErrSelectionSet},
		{"high-primary", nil, SelectionSet{[]Range{{0, 0}}, 1}, ErrSelectionSet},
		{"257-selections", nil, SelectionSet{Ranges: make([]Range, 257)}, ErrSelectionLimit},
		{"negative-after", nil, SelectionSet{Ranges: []Range{{0, -1}}}, ErrRange},
		{"huge-after", nil, SelectionSet{Ranges: []Range{{math.MaxInt, 0}}}, ErrRange},
		// Even an endpoint a later normalization merge would absorb is validated.
		{"absorbed-surrogate", nil, SelectionSet{Ranges: []Range{{0, 3}, {2, 2}}}, ErrSurrogate},
	} {
		t.Run(tc.name, func(t *testing.T) {
			unchanged := multiStateGuard(t, b)
			changed, err := b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), tc.edits, tc.after)
			if changed || !errors.Is(err, tc.want) {
				t.Fatal(changed, err, tc.want)
			}
			unchanged()
		})
	}
	for _, identity := range [][2]uint64{{b.Version() - 1, b.SelectionGeneration()}, {b.Version() + 1, b.SelectionGeneration()}, {b.Version(), b.SelectionGeneration() - 1}, {b.Version(), b.SelectionGeneration() + 1}} {
		unchanged := multiStateGuard(t, b)
		if changed, err := b.ApplyPatchUTF16(identity[0], identity[1], nil, validAfter); changed || !errors.Is(err, ErrStaleEdit) {
			t.Fatal("stale empty patch", changed, err)
		}
		unchanged()
	}
}

func TestPatchUtf16FinalIntermediateAndOutputCoordinates(t *testing.T) {
	for _, limit := range []int{9, 10} {
		t.Run("final-"+strconv.Itoa(limit), func(t *testing.T) {
			b := NewWithLimit("甲🙂x", limit)
			mustSetSelections(t, b, []Range{{1, 0}, {3, 1}}, 1)
			edits := []PatchEdit{{0, 1, "甲", "乙"}, {1, 3, "🙂", "中文"}}
			after := SelectionSet{[]Range{{3, 1}, {4, 4}}, 1}
			if limit == 10 {
				checkPatchRoundTrip(t, b, edits, after, after)
			} else {
				unchanged := multiStateGuard(t, b)
				if changed, err := b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), edits, after); changed || !errors.Is(err, ErrSizeLimit) {
					t.Fatal(changed, err)
				}
				unchanged()
			}
		})
	}
	for _, limit := range []int{9, 10} {
		t.Run("intermediate-"+strconv.Itoa(limit), func(t *testing.T) {
			b := NewWithLimit("12345", limit)
			edits := []PatchEdit{{0, 4, "1234", ""}, {5, 5, "", "abcde"}}
			after := SelectionSet{Ranges: []Range{{6, 6}}}
			if limit == 10 {
				checkPatchRoundTrip(t, b, edits, after, after)
			} else {
				unchanged := multiStateGuard(t, b)
				if changed, err := b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), edits, after); changed || !errors.Is(err, ErrSizeLimit) {
					t.Fatal(changed, err)
				}
				unchanged()
			}
		})
	}
	for _, tc := range []struct {
		name, source string
		edits        []PatchEdit
		after, want  SelectionSet
		err          error
	}{
		{"growth", "a", []PatchEdit{{0, 0, "", "🙂"}}, SelectionSet{[]Range{{3, 2}}, 0}, SelectionSet{[]Range{{3, 2}}, 0}, nil},
		{"growth-surrogate", "a", []PatchEdit{{0, 0, "", "🙂"}}, SelectionSet{[]Range{{1, 1}}, 0}, SelectionSet{}, ErrSurrogate},
		{"shrink", "🙂x", []PatchEdit{{0, 2, "🙂", ""}}, SelectionSet{[]Range{{1, 0}}, 0}, SelectionSet{[]Range{{1, 0}}, 0}, nil},
		{"shrink-range", "🙂x", []PatchEdit{{0, 2, "🙂", ""}}, SelectionSet{[]Range{{2, 2}}, 0}, SelectionSet{}, ErrRange},
		{"normalization", "ab", []PatchEdit{{0, 1, "a", "🙂"}}, SelectionSet{[]Range{{3, 2}, {0, 2}, {2, 2}}, 2}, SelectionSet{[]Range{{0, 2}, {3, 2}}, 1}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := New(tc.source)
			if tc.err == nil {
				checkPatchRoundTrip(t, b, tc.edits, tc.after, tc.want)
			} else {
				unchanged := multiStateGuard(t, b)
				if changed, err := b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), tc.edits, tc.after); changed || !errors.Is(err, tc.err) {
					t.Fatal(changed, err)
				}
				unchanged()
			}
		})
	}
}

func TestPatchUtf16EditAndSelectionBudgetsAreSeparate(t *testing.T) {
	b := New(strings.Repeat("a ", 1024))
	edits := make([]PatchEdit, 1024)
	before := make([]Range, 256)
	after := SelectionSet{Ranges: make([]Range, 256), Primary: 173}
	for i := range edits {
		edits[i] = PatchEdit{2 * i, 2*i + 1, "a", "🙂"}
	}
	for i := range before {
		before[i] = Range{8*i + 1, 8 * i}
		after.Ranges[i] = Range{12*i + 2, 12 * i}
	}
	mustSetSelections(t, b, before, 173)
	unchanged := multiStateGuard(t, b)
	if changed, err := b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), append(edits, PatchEdit{}), after); changed || !errors.Is(err, ErrPatchLimit) {
		t.Fatal("1025 admitted", changed, err)
	}
	unchanged()
	tooMany := SelectionSet{Ranges: append(append([]Range(nil), after.Ranges...), Range{})}
	if changed, err := b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), edits, tooMany); changed || !errors.Is(err, ErrSelectionLimit) {
		t.Fatal("257 output selections admitted", changed, err)
	}
	unchanged()
	checkPatchRoundTrip(t, b, edits, after, after)
	if len(b.SelectionSet().Ranges) != 256 || len(b.undo[len(b.undo)-1].edits) != 1024 {
		t.Fatal("edit and selection budgets became coupled")
	}
	// Legacy 257 refusal and legacy no-op selection updates remain contractual.
	unchanged = multiStateGuard(t, b)
	if err := b.ReplaceBatchUTF16(b.Version(), b.SelectionGeneration(), make([]Replacement, 257), 0); !errors.Is(err, ErrSelectionLimit) {
		t.Fatal(err)
	}
	unchanged()
	if err := b.ReplaceBatchUTF16(b.Version(), b.SelectionGeneration(), []Replacement{{0, 2, "🙂"}}, 0); err != nil {
		t.Fatal(err)
	}
	requireSelections(t, b, SelectionSet{Ranges: []Range{{2, 2}}})
}

func TestPatchUtf16NoOpPreservesRedoAndSelections(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		edits        []PatchEdit
	}{
		{"empty", "🙂x", nil},
		{"identical", "🙂x", []PatchEdit{{0, 2, "🙂", "🙂"}}},
		{"redistributed", "🙂x", []PatchEdit{{0, 2, "🙂", ""}, {2, 3, "x", "🙂x"}}},
		{"piece-stream", strings.Repeat("ab", 5000), []PatchEdit{{0, 2, "ab", "a"}, {2, 10000, strings.Repeat("ab", 4999), "b" + strings.Repeat("ab", 4999)}}},
		{"shifted-gap", strings.Repeat("a", 13000), []PatchEdit{{0, 1, "a", ""}, {12999, 13000, "a", "aa"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := New(tc.source)
			before := mustSetSelections(t, b, []Range{{2, 0}, {b.LenUTF16(), b.LenUTF16()}}, 1)
			if err := b.ReplaceUTF16(b.LenUTF16(), b.LenUTF16(), "!"); err != nil || !b.Undo() {
				t.Fatal("redo fixture", err)
			}
			unchanged := multiStateGuard(t, b)
			// A valid but different after-set must never replace the original set.
			if changed, err := b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), tc.edits, SelectionSet{Ranges: []Range{{0, 0}}}); err != nil || changed {
				t.Fatal(changed, err)
			}
			unchanged()
			requireSelections(t, b, before)
			if tc.source == "🙂x" {
				if changed, err := b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), tc.edits, SelectionSet{Ranges: []Range{{1, 1}}}); changed || !errors.Is(err, ErrSurrogate) {
					t.Fatal("net-zero output surrogate bypass", changed, err)
				}
				unchanged()
			}
			if !b.Redo() || b.String() != tc.source+"!" {
				t.Fatal("no-op lost original redo")
			}
		})
	}
	b := NewWithLimit("12345", 6)
	unchanged := multiStateGuard(t, b)
	if changed, err := b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), []PatchEdit{{0, 4, "1234", ""}, {4, 5, "5", "12345"}}, SelectionSet{Ranges: []Range{{0, 0}}}); changed || !errors.Is(err, ErrSizeLimit) {
		t.Fatal("net-zero bypassed intermediate ceiling", changed, err)
	}
	unchanged()
}

func TestPatchUtf16StandaloneHistoryAdmissionAndTrim(t *testing.T) {
	// Two before/after secondaries plus a->XYZ: independently computed cost.
	cost := 4 + 4*(strconv.IntSize/4)
	for _, entries := range []int{0, 1, 2} {
		for _, bytes := range []int{0, cost - 1, cost} {
			t.Run(strconv.Itoa(entries)+"-"+strconv.Itoa(bytes), func(t *testing.T) {
				b := New("a b c")
				mustSetSelections(t, b, []Range{{1, 0}, {2, 2}, {5, 4}}, 2)
				b.SetHistoryLimits(entries, bytes)
				after := SelectionSet{[]Range{{3, 0}, {4, 4}, {7, 6}}, 2}
				edits := []PatchEdit{{0, 1, "a", "XYZ"}}
				if entries >= 1 && bytes == cost {
					checkPatchRoundTrip(t, b, edits, after, after)
					if b.History().RetainedBytes != cost {
						t.Fatal("selection/text history cost", b.History(), cost)
					}
				} else {
					unchanged := multiStateGuard(t, b)
					if changed, err := b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), edits, after); changed || !errors.Is(err, ErrPatchHistoryLimit) {
						t.Fatal(changed, err)
					}
					unchanged()
					// No-op requires no new history budget.
					if changed, err := b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), nil, after); changed || !errors.Is(err, ErrRange) {
						t.Fatal("invalid no-op output accepted", changed, err)
					}
					unchanged()
					if changed, err := b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), nil, b.SelectionSet()); changed || err != nil {
						t.Fatal("no-op required history capacity", changed, err)
					}
					unchanged()
				}
			})
		}
	}
	for _, entries := range []int{1, 2} {
		b := New("abc")
		if err := b.ReplaceUTF16(2, 3, "C"); err != nil {
			t.Fatal(err)
		}
		b.SetHistoryLimits(entries, 2)
		before := b.SelectionSet()
		changed, err := b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), []PatchEdit{{0, 1, "a", "A"}}, SelectionSet{Ranges: []Range{{1, 1}}})
		if !changed || err != nil || b.History() != (HistoryStatus{UndoDepth: 1, RetainedBytes: 2, Truncated: true}) || !b.Undo() || b.String() != "abC" || b.CanUndo() {
			t.Fatal("trim removed new undo item", changed, err, b.History())
		}
		requireSelections(t, b, before)
		if !b.Redo() || b.String() != "AbC" {
			t.Fatal("trimmed item redo")
		}
		requireSelections(t, b, SelectionSet{Ranges: []Range{{1, 1}}})
	}
	b := New("x")
	b.SetHistoryLimits(1, math.MaxInt)
	if b.admitsStandaloneHistory(transaction{bytes: math.MaxInt, beforeSelections: selectionExtra{secondary: []Range{{0, 0}}}}) || b.admitsStandaloneHistory(transaction{bytes: math.MaxInt, bytesOverflow: true}) {
		t.Fatal("saturated text/selection history accounting admitted overflow")
	}
}

func TestPatchUtf16ForeignOwnersRemainUsable(t *testing.T) {
	for _, composition := range []bool{false, true} {
		t.Run(strconv.FormatBool(composition), func(t *testing.T) {
			b := New("abc")
			mustSetSelections(t, b, []Range{{1, 0}, {3, 3}}, 0)
			var token CompositionToken
			if composition {
				var err error
				token, err = b.BeginCompositionGroup()
				if err != nil {
					t.Fatal(err)
				}
			} else if err := b.BeginGroup(); err != nil {
				t.Fatal(err)
			}
			unchanged := multiStateGuard(t, b)
			for _, edits := range [][]PatchEdit{nil, {{0, 1, "a", "a"}}, {{0, 1, "a", "A"}}} {
				if changed, err := b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), edits, b.SelectionSet()); changed || !errors.Is(err, ErrGroup) {
					t.Fatal("foreign owner admitted", changed, err)
				}
				unchanged()
			}
			if composition {
				if !b.OwnsComposition(token) || b.ReplaceCompositionUTF16(token, 0, 1, "A") != nil || b.EndCompositionGroup(token) != nil {
					t.Fatal("composition owner could not continue")
				}
			} else if b.ReplaceUTF16(0, 1, "A") != nil || !b.EndGroup() {
				t.Fatal("ordinary owner could not continue")
			}
			if b.String() != "Abc" || !b.Undo() || b.String() != "abc" || !b.Redo() || b.String() != "Abc" {
				t.Fatal("owner continuation history")
			}
		})
	}
}

func TestPatchUtf16IndependentStringOracleAndHistory(t *testing.T) {
	rng := rand.New(rand.NewSource(20261120))
	words := []string{"", "a", "中", "🙂", "e\u0301", "👨‍👩‍👧‍👦", "\r\n", "\n", "\r", "🇨🇳"}
	for trial := 0; trial < 240; trial++ {
		var source strings.Builder
		count := 45
		if trial%20 == 0 {
			count = 2600
		}
		for i := 0; i < count; i++ {
			source.WriteString(words[1+rng.Intn(len(words)-1)])
		}
		original := source.String()
		runes := []rune(original)
		n := 1 + rng.Intn(7)
		positions := rng.Perm(len(runes) + 1)[:2*n]
		sort.Ints(positions)
		edits := make([]PatchEdit, n)
		before := SelectionSet{Ranges: make([]Range, n), Primary: rng.Intn(n)}
		for i := range edits {
			left, right := positions[2*i], positions[2*i+1]
			if rng.Intn(3) == 0 {
				right = left
			}
			edits[i] = PatchEdit{patchUnits(string(runes[:left])), patchUnits(string(runes[:right])), string(runes[left:right]), words[rng.Intn(len(words))]}
			before.Ranges[i] = Range{edits[i].End, edits[i].Start}
		}
		b := New(original)
		mustSetSelections(t, b, before.Ranges, before.Primary)
		wantText := patchOracle(t, original, edits)
		// Generate disjoint output ranges from the independent rune oracle, so
		// internal fragment boundaries, directions and primary membership vary.
		outputRunes := []rune(wantText)
		after := SelectionSet{Ranges: []Range{{0, 0}}}
		if len(outputRunes) > 0 {
			count := min(4, (len(outputRunes)+1)/2)
			points := rng.Perm(len(outputRunes) + 1)[:2*count]
			sort.Ints(points)
			after = SelectionSet{Ranges: make([]Range, count), Primary: rng.Intn(count)}
			for i := range after.Ranges {
				left := patchUnits(string(outputRunes[:points[2*i]]))
				right := patchUnits(string(outputRunes[:points[2*i+1]]))
				after.Ranges[i] = Range{left, right}
				if i%2 == 0 {
					after.Ranges[i] = Range{right, left}
				}
			}
		}
		// Cross a random virtual output with a surrogate interior before any
		// changing call, then compare every Buffer field on refusal.
		units := 0
		for _, r := range outputRunes {
			if r > 0xffff {
				unchanged := multiStateGuard(t, b)
				bad := SelectionSet{Ranges: []Range{{units + 1, units + 1}}}
				if changed, err := b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), edits, bad); changed || !errors.Is(err, ErrSurrogate) {
					t.Fatal("random output surrogate", trial, changed, err)
				}
				unchanged()
				break
			}
			units++
		}
		if wantText == original {
			unchanged := multiStateGuard(t, b)
			if changed, err := b.ApplyPatchUTF16(b.Version(), b.SelectionGeneration(), edits, after); changed || err != nil {
				t.Fatal(trial, changed, err)
			}
			unchanged()
		} else {
			checkPatchRoundTrip(t, b, edits, after, after)
		}
		if !reflect.DeepEqual(b.SelectionSet(), after) && wantText != original {
			t.Fatal("random complete after state", trial)
		}
	}
}
