package text

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestGrowthAdmissionRejectsBeforeEditingState(t *testing.T) {
	b := NewWithLimit("甲🙂x", 9)
	if err := b.ReplaceUTF16(4, 4, "z"); err != nil || !b.Undo() {
		t.Fatal("redo fixture", err)
	}
	if err := b.SetSelection(4, 1); err != nil {
		t.Fatal(err)
	}
	snapshot, version, selection, history := b.Snapshot(), b.Version(), b.Selection(), b.History()
	err := b.ReplaceUTF16(1, 3, "中文") // 8 - 4 + 6 = 10: one byte over.
	var limit *SizeLimitError
	if !errors.Is(err, ErrSizeLimit) || !errors.As(err, &limit) || limit.Limit != 9 || !strings.Contains(err.Error(), "缩短") {
		t.Fatal("unusable admission error", err)
	}
	if b.String() != "甲🙂x" || !b.EqualSnapshot(snapshot) || snapshot.String() != "甲🙂x" ||
		b.Version() != version || b.Selection() != selection || b.History() != history || b.InGroup() {
		t.Fatal("rejection mutated source/snapshot/version/selection/history")
	}
	if err := b.ReplaceUTF16(1, 3, "abcde"); err != nil || b.LenBytes() != 9 {
		t.Fatal("exact-limit replacement rejected", err)
	}
	version, history = b.Version(), b.History()
	if err := b.ReplaceUTF16(1, 6, "abcde"); err != nil || b.Version() != version || b.History() != history || b.Selection() != (Range{6, 6}) {
		t.Fatal("no-op selection compatibility", err)
	}
	if err := b.ReplaceUTF16(1, 6, "中"); err != nil || b.String() != "甲中x" {
		t.Fatal("shrink failed", err)
	}
	if !b.Undo() || b.String() != "甲abcdex" || !b.Redo() || b.String() != "甲中x" {
		t.Fatal("accepted states cannot be replayed")
	}
}

func TestGrowthAdmissionValidationAndOpenGroupOwnership(t *testing.T) {
	b := NewWithLimit("🙂x", 5)
	for _, tc := range []struct {
		start, end int
		value      string
		want       error
	}{
		{-1, 999, "\xff", ErrUTF8},
		{3, 0, "overflow", ErrRange},
		{-1, 0, "overflow", ErrRange},
		{1, 2, "overflow", ErrSurrogate},
		{0, 1, "overflow", ErrSurrogate},
		{0, 999, "overflow", ErrRange},
	} {
		if err := b.ReplaceUTF16(tc.start, tc.end, tc.value); !errors.Is(err, tc.want) {
			t.Fatalf("validation precedence %+v: %v", tc, err)
		}
	}
	if err := b.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	if err := b.ReplaceUTF16(0, 2, "a"); err != nil {
		t.Fatal(err)
	}
	group := b.group
	edits := append([]change(nil), group.edits...)
	before, after, bytes := group.before, group.after, group.bytes
	version, snapshot, selection, history := b.Version(), b.Snapshot(), b.Selection(), b.History()
	if err := b.ReplaceUTF16(1, 1, "🙂"); !errors.Is(err, ErrSizeLimit) ||
		b.group != group || !reflect.DeepEqual(group.edits, edits) || group.before != before || group.after != after || group.bytes != bytes ||
		b.Version() != version || !b.EqualSnapshot(snapshot) || b.Selection() != selection || b.History() != history {
		t.Fatal("rejection damaged owned group", err)
	}
	if !b.CancelGroup() || b.String() != "🙂x" || b.InGroup() || b.CanUndo() {
		t.Fatal("cancel must restore larger accepted seed")
	}
	// Existing rollback replays an edit, so revision is deliberately different.
	if b.Version() <= version {
		t.Fatal("cancel should journal replay")
	}
	if _, err := b.CheckReplaceSize(b.LenBytes(), 0, math.MaxInt); !errors.Is(err, ErrSizeLimit) {
		t.Fatal("huge caller count overflowed admission", err)
	}
}

func TestGrowthAdmissionOversizedSeedHistoryAndRestore(t *testing.T) {
	b := NewWithLimit("中文🙂", 4)
	if b.MaxBytes() != 10 || b.String() != "中文🙂" {
		t.Fatal("oversized seed truncated")
	}
	seed := b.Snapshot()
	_ = b.SetSelection(4, 0)
	if err := b.ReplaceUTF16(0, 4, "a"); err != nil || b.MaxBytes() != 10 {
		t.Fatal("shrinking lowered seed ceiling", err)
	}
	if err := b.ReplaceUTF16(1, 1, "123456789"); err != nil || b.LenBytes() != 10 {
		t.Fatal("regrowth to original seed ceiling", err)
	}
	if err := b.ReplaceUTF16(10, 10, "x"); !errors.Is(err, ErrSizeLimit) {
		t.Fatal("seed ceiling grew", err)
	}
	if !b.Undo() || b.String() != "a" || !b.Undo() || !b.EqualSnapshot(seed) || b.Selection() != (Range{4, 0}) || !b.Redo() || b.String() != "a" {
		t.Fatal("oversized seed history failed")
	}
	if err := b.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	_ = b.ReplaceUTF16(0, 1, "")
	if !b.CancelGroup() || b.String() != "a" || !b.CanRedo() || !b.Redo() || b.LenBytes() != 10 {
		t.Fatal("cancel destroyed redo")
	}
	identity := b
	b.Restore(NewWithLimit("中", 1000).Snapshot())
	if b != identity || b.MaxBytes() != 4 || b.String() != "中" || b.CanUndo() || b.CanRedo() || b.InGroup() {
		t.Fatal("restore adopted source policy or changed buffer identity")
	}
	if err := b.ReplaceUTF16(1, 1, "a"); err != nil {
		t.Fatal(err)
	}
	if err := b.ReplaceUTF16(2, 2, "b"); !errors.Is(err, ErrSizeLimit) {
		t.Fatal("destination base not retained", err)
	}
	b.Restore(seed)
	if b.MaxBytes() != 10 || !b.EqualSnapshot(seed) || b.CanUndo() {
		t.Fatal("restore did not reseed from larger snapshot")
	}
	if err := b.ReplaceUTF16(0, 4, "x"); err != nil || !b.Undo() || !b.EqualSnapshot(seed) {
		t.Fatal("restored seed cannot undo", err)
	}
}

func TestGrowthAdmissionConstructors(t *testing.T) {
	if New("a").MaxBytes() != DefaultMaxBytes || DefaultMaxBytes != 128<<20 {
		t.Fatal("ordinary default changed")
	}
	for _, tc := range []struct {
		name, value string
		limit       int
		wantUTF8    bool
	}{{"zero", "a", 0, false}, {"negative", "a", -1, false}, {"utf8", "\xff", 10, true}} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				err := recover()
				if err == nil || tc.wantUTF8 && err != ErrUTF8 {
					t.Fatal("constructor panic compatibility", err)
				}
			}()
			NewWithLimit(tc.value, tc.limit)
		})
	}
}
