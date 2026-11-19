package text

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

func mustComposition(t *testing.T, b *Buffer) CompositionToken {
	t.Helper()
	token, err := b.BeginCompositionGroup()
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestCoalescingFullSpanHundredsOfUpdatesRetainOnlyEndpoints(t *testing.T) {
	for _, selected := range []bool{false, true} {
		t.Run(map[bool]string{false: "insert", true: "replace-selected"}[selected], func(t *testing.T) {
			original := ""
			if selected {
				original = "原🙂"
			}
			b := NewWithLimit("L"+original+"R", 64)
			_ = b.SetSelection(1+UTF16Len(original), 1)
			selection, seed := b.Selection(), b.Snapshot()
			token := mustComposition(t, b)
			value := original
			values := []string{"a", "中🙂", "abc", "🙂", "🙂", "文", ""}
			for i := 0; i < 600; i++ {
				next := values[i%len(values)]
				version := b.Version()
				if err := b.ReplaceCompositionUTF16(token, 1, 1+UTF16Len(value), next); err != nil {
					t.Fatal(i, err)
				}
				if next != value {
					version++
				}
				if b.Version() != version || b.String() != "L"+next+"R" || b.Selection() != (Range{1 + UTF16Len(next), 1 + UTF16Len(next)}) || b.group.before != selection || b.group.after != b.Selection() {
					t.Fatalf("update%d changed revision/selection/source", i)
				}
				if next == original {
					if len(b.group.edits) != 0 || b.composition.baseline != original || b.group.bytes != 0 {
						t.Fatal("net-zero suffix retained edits", i)
					}
				} else if len(b.group.edits) != 1 || b.group.edits[0] != (change{start: 1, old: original, next: next}) || b.group.bytes != len(original)+len(next) || b.composition.baseline != "" {
					t.Fatal("retained an intermediate instead of original/current", i, b.group)
				}
				if b.group.bytesOverflow || cap(b.group.edits) > 1 {
					t.Fatal("full-span transaction grew with update count")
				}
				for _, dead := range b.group.edits[len(b.group.edits):cap(b.group.edits)] {
					if dead != (change{}) {
						t.Fatal("net-zero retained dead suffix backing")
					}
				}
				value = next
			}
			if err := b.ReplaceCompositionUTF16(token, 1, 1+UTF16Len(value), "最终🙂"); err != nil {
				t.Fatal(err)
			}
			if err := b.EndCompositionGroup(token); err != nil {
				t.Fatal(err)
			}
			if b.InGroup() || b.OwnsComposition(token) || b.History().UndoDepth != 1 || len(b.undo[0].edits) != 1 {
				t.Fatal("final commit not one aggregate transaction")
			}
			if !b.Undo() || !b.EqualSnapshot(seed) || b.Selection() != selection || !b.Redo() || b.String() != "L最终🙂R" {
				t.Fatal("aggregate undo/redo lost source/selection")
			}
		})
	}
}

func TestCoalescingNoOpBaselineNetZeroCancelAndRedo(t *testing.T) {
	for _, finish := range []string{"same-end", "return-end", "cancel", "later-update"} {
		t.Run(finish, func(t *testing.T) {
			b := NewWithLimit("L原🙂R", 32)
			_ = b.ReplaceUTF16(b.LenUTF16(), b.LenUTF16(), "!")
			if !b.Undo() {
				t.Fatal("redo fixture")
			}
			_ = b.SetSelection(4, 1)
			selection, history, seed, version := b.Selection(), b.History(), b.Snapshot(), b.Version()
			token := mustComposition(t, b)
			if err := b.ReplaceCompositionUTF16(token, 1, 4, "原🙂"); err != nil {
				t.Fatal(err)
			}
			if b.Version() != version || len(b.group.edits) != 0 || b.group.before != selection || b.composition.baseline != "原🙂" {
				t.Fatal("first same-text baseline lost")
			}
			if finish != "same-end" {
				for _, next := range []string{"x", "中🙂", "原🙂"} {
					span := b.composition.span
					if err := b.ReplaceCompositionUTF16(token, span.Start(), span.End(), next); err != nil {
						t.Fatal(err)
					}
				}
				if !b.EqualSnapshot(seed) || len(b.group.edits) != 0 || b.History() != history || b.group.bytes != 0 || b.Version() != version+3 {
					t.Fatal("net-zero lost history or invented revision")
				}
			}
			switch finish {
			case "cancel":
				version = b.Version()
				if err := b.CancelCompositionGroup(token); err != nil || b.Selection() != selection || b.Version() != version {
					t.Fatal("empty suffix cancel changed text/revision", err)
				}
			case "later-update":
				if err := b.ReplaceCompositionUTF16(token, 1, 4, "新"); err != nil {
					t.Fatal(err)
				}
				if err := b.CancelCompositionGroup(token); err != nil || b.Selection() != selection || !b.EqualSnapshot(seed) {
					t.Fatal("baseline after net-zero could not cancel", err)
				}
			default:
				if err := b.EndCompositionGroup(token); err != nil {
					t.Fatal(err)
				}
			}
			if b.History() != history || !b.CanRedo() || !b.Redo() || b.String() != "L原🙂R!" {
				t.Fatal("no-change composition erased earlier redo")
			}
		})
	}
}

func TestCoalescingIrregularRangesCompactOnlyNewestSuffix(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		b := NewWithLimit("A中🙂Z", 64)
		_ = b.SetSelection(4, 1)
		selection := b.Selection()
		token := mustComposition(t, b)
		steps := []struct {
			start, end int
			next       string
			edits      int
		}{
			{1, 4, "xy", 1}, {1, 3, "pq", 1}, {0, 1, "B", 2}, {0, 1, "甲", 2},
			{0, 1, "A", 1}, // net-zero newest suffix leaves the preceding one intact
			{3, 4, "Z", 1}, // irregular no-op establishes a new baseline
			{3, 4, "🙂", 2}, {3, 5, "末", 2},
		}
		for i, step := range steps {
			if err := b.ReplaceCompositionUTF16(token, step.start, step.end, step.next); err != nil || len(b.group.edits) != step.edits {
				t.Fatal(i, err, b.group)
			}
			bytes := 0
			for _, edit := range b.group.edits {
				bytes += len(edit.old) + len(edit.next)
			}
			if b.group.bytes != bytes {
				t.Fatal("suffix accounting", i)
			}
		}
		if b.String() != "Apq末" || b.group.edits[0] != (change{start: 1, old: "中🙂", next: "pq"}) || b.group.edits[1] != (change{start: 3, old: "Z", next: "末"}) {
			t.Fatal("fallback altered earlier suffix", b.String(), b.group)
		}
		if cancel {
			if err := b.CancelCompositionGroup(token); err != nil || b.String() != "A中🙂Z" || b.Selection() != selection || b.CanUndo() {
				t.Fatal("irregular cancel", err)
			}
		} else {
			if err := b.EndCompositionGroup(token); err != nil || !b.Undo() || b.String() != "A中🙂Z" || b.Selection() != selection || !b.Redo() || b.String() != "Apq末" {
				t.Fatal("irregular undo/redo", err)
			}
		}
	}
}

func TestCoalescingRefusalsAreAtomicAndOrdinaryGroupsUnchanged(t *testing.T) {
	b := NewWithLimit("a🙂z", 8)
	_ = b.ReplaceUTF16(3, 4, "")
	if !b.Undo() {
		t.Fatal("redo fixture")
	}
	_ = b.SetSelection(3, 1)
	token := mustComposition(t, b)
	if err := b.ReplaceCompositionUTF16(token, 1, 3, "🙂"); err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []struct {
		start, end int
		next       string
		want       error
	}{
		{-1, 999, "\xff", ErrUTF8}, {3, 0, "x", ErrRange}, {1, 2, "x", ErrSurrogate}, {2, 3, "x", ErrSurrogate}, {1, 3, "超限🙂", ErrSizeLimit},
	} {
		version, selection, history, snapshot := b.Version(), b.Selection(), b.History(), b.Snapshot()
		group, composition := *b.group, *b.composition
		edits := append([]change(nil), b.group.edits...)
		if err := b.ReplaceCompositionUTF16(token, candidate.start, candidate.end, candidate.next); !errors.Is(err, candidate.want) {
			t.Fatal(candidate, err)
		}
		if b.Version() != version || b.Selection() != selection || b.History() != history || !b.EqualSnapshot(snapshot) || !reflect.DeepEqual(b.group.edits, edits) || b.group.before != group.before || b.group.after != group.after || b.group.bytes != group.bytes || *b.composition != composition {
			t.Fatal("refusal mutated retained state")
		}
	}
	if err := b.ReplaceCompositionUTF16(token, 1, 3, "中文"); err != nil {
		t.Fatal("exact limit retry", err)
	}
	if err := b.CancelCompositionGroup(token); err != nil || b.String() != "a🙂z" || !b.CanRedo() {
		t.Fatal("refusal/retry cancel lost redo", err)
	}
	if err := b.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"x", "y", "z"} {
		if err := b.ReplaceUTF16(0, 1, value); err != nil {
			t.Fatal(err)
		}
	}
	if len(b.group.edits) != 3 || b.EndGroup() != true || b.History().UndoDepth != 1 || !b.Undo() || b.String() != "a🙂z" {
		t.Fatal("ordinary group was coalesced or changed")
	}
}

func TestCoalescingTokensCannotTouchOtherOrReplacedGroups(t *testing.T) {
	a, b := New("a"), New("b")
	ta, tb := mustComposition(t, a), mustComposition(t, b)
	assertRefused := func(target *Buffer, token CompositionToken) {
		t.Helper()
		group, version, selection, history, snapshot := target.group, target.Version(), target.Selection(), target.History(), target.Snapshot()
		if target.OwnsComposition(token) || !errors.Is(target.ReplaceCompositionUTF16(token, 0, 0, "x"), ErrComposition) || !errors.Is(target.EndCompositionGroup(token), ErrComposition) || !errors.Is(target.CancelCompositionGroup(token), ErrComposition) {
			t.Fatal("foreign/stale token accepted")
		}
		if target.group != group || target.Version() != version || target.Selection() != selection || target.History() != history || !target.EqualSnapshot(snapshot) {
			t.Fatal("foreign/stale token mutated state")
		}
	}
	assertRefused(a, tb)
	assertRefused(b, ta)
	assertRefused(a, CompositionToken{})
	shallow := *a
	assertRefused(&shallow, ta)
	if err := a.ReplaceUTF16(0, 1, "x"); !errors.Is(err, ErrGroup) || a.EndGroup() || a.CancelGroup() || !a.OwnsComposition(ta) {
		t.Fatal("tokenless action hijacked composition", err)
	}
	if _, err := a.BeginCompositionGroup(); !errors.Is(err, ErrGroup) {
		t.Fatal(err)
	}
	if err := a.EndCompositionGroup(ta); err != nil {
		t.Fatal(err)
	}
	assertRefused(a, ta)
	for _, reset := range []string{"reset", "restore", "end", "cancel"} {
		token := mustComposition(t, a)
		_ = a.ReplaceCompositionUTF16(token, 0, 1, "x")
		switch reset {
		case "reset":
			a.ResetHistory()
		case "restore":
			a.Restore(New("seed").Snapshot())
		case "end":
			_ = a.EndCompositionGroup(token)
		case "cancel":
			_ = a.CancelCompositionGroup(token)
		}
		if err := a.BeginGroup(); err != nil {
			t.Fatal(err)
		}
		assertRefused(a, token)
		if !a.CancelGroup() {
			t.Fatal("foreign ordinary group lost")
		}
		fresh := mustComposition(t, a)
		assertRefused(a, token)
		if err := a.CancelCompositionGroup(fresh); err != nil {
			t.Fatal(err)
		}
	}
	_ = b.CancelCompositionGroup(tb)
}

func TestCoalescingAccountingSaturatesAndRecountsAfterShrink(t *testing.T) {
	count := transaction{bytes: math.MaxInt - 1}
	count.addBytes(1)
	if count.bytes != math.MaxInt || count.bytesOverflow {
		t.Fatal("exact max misclassified")
	}
	count.addBytes(1)
	count.addBytes(5)
	if count.bytes != math.MaxInt || !count.bytesOverflow {
		t.Fatal("overflow was lost")
	}
	// Exercise recovery of a saturated active count without enormous strings.
	b := New("abc")
	token := mustComposition(t, b)
	_ = b.ReplaceCompositionUTF16(token, 0, 1, "xy")
	_ = b.ReplaceCompositionUTF16(token, 3, 4, "long")
	b.group.bytes, b.group.bytesOverflow = math.MaxInt, true
	if err := b.ReplaceCompositionUTF16(token, 3, 7, "z"); err != nil || b.group.bytes != 5 || b.group.bytesOverflow {
		t.Fatal("shrink did not recompute all retained suffixes", err, b.group)
	}
	if err := b.ReplaceCompositionUTF16(token, 3, 4, "c"); err != nil || b.group.bytes != 3 {
		t.Fatal("net-zero suffix accounting", err)
	}
	_ = b.CancelCompositionGroup(token)
	// Overflow cannot evade trimming even when configured at the integer max.
	b.historyEntries, b.historyBytes = 10, math.MaxInt
	b.undo = []transaction{{bytes: math.MaxInt, bytesOverflow: true}}
	if b.History().RetainedBytes != math.MaxInt {
		t.Fatal("overflow returned a wrapped total")
	}
	b.trimHistory()
	if len(b.undo) != 0 || !b.History().Truncated {
		t.Fatal("overflow evaded trimming")
	}
	b.undo = []transaction{{bytes: math.MaxInt}, {bytes: 1}}
	if b.History().RetainedBytes != math.MaxInt {
		t.Fatal("cross-transaction overflow")
	}
	b.trimHistory()
	if len(b.undo) != 1 || b.History().RetainedBytes != 1 {
		t.Fatal("cross-transaction overflow trimming")
	}
}

func TestCoalescingCommitUsesExistingHistoryBudget(t *testing.T) {
	for _, budget := range []int{5, 6} {
		b := New("x")
		b.SetHistoryLimits(10, budget)
		token := mustComposition(t, b)
		current := "x"
		for i := 0; i < 200; i++ {
			next := strings.Repeat("a", 1+i%5)
			if err := b.ReplaceCompositionUTF16(token, 0, len(current), next); err != nil {
				t.Fatal(err)
			}
			current = next
		}
		if err := b.EndCompositionGroup(token); err != nil {
			t.Fatal(err)
		}
		if b.CanUndo() != (budget == 6) || b.History().Truncated != (budget == 5) {
			t.Fatal("normal commit trimming changed", budget, b.History())
		}
	}
}
