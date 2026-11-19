package text

import (
	"errors"
	"math/rand"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestUnicodeBoundaryAndSelection(t *testing.T) {
	b := New("甲🙂e\u0301\r\n乙")
	if b.LenUTF16() != 8 || b.Lines() != 2 {
		t.Fatalf("units=%d lines=%d", b.LenUTF16(), b.Lines())
	}
	if err := b.ReplaceUTF16(2, 3, "x"); !errors.Is(err, ErrSurrogate) {
		t.Fatalf("split accepted: %v", err)
	}
	if err := b.SetSelection(3, 1); err != nil {
		t.Fatal(err)
	}
	if err := b.ReplaceUTF16(1, 3, "中文"); err != nil {
		t.Fatal(err)
	}
	if b.String() != "甲中文e\u0301\r\n乙" {
		t.Fatal(b.String())
	}
	if !b.Undo() || b.Selection() != (Range{3, 1}) || b.String() != "甲🙂e\u0301\r\n乙" {
		t.Fatal("undo lost text/selection")
	}
	if !b.Redo() || b.String() != "甲中文e\u0301\r\n乙" {
		t.Fatal("redo")
	}
	line, err := b.Line(0)
	if err != nil || line != "甲中文e\u0301" {
		t.Fatalf("line=%q %v", line, err)
	}
	start, _ := b.OffsetOfLine(1)
	if start != 7 {
		t.Fatal(start)
	}
}

func TestCompositionIsOneTransactionAndCancelKeepsRedo(t *testing.T) {
	b := New("before after")
	_ = b.SetSelection(7, 7)
	_ = b.BeginGroup()
	_ = b.ReplaceUTF16(7, 7, "z")
	_ = b.ReplaceUTF16(7, 8, "zhong")
	_ = b.ReplaceUTF16(7, 12, "中")
	b.EndGroup()
	if b.History().UndoDepth != 1 || b.String() != "before 中after" {
		t.Fatal(b.String(), b.History())
	}
	b.Undo()
	if b.String() != "before after" || b.Selection() != (Range{7, 7}) {
		t.Fatal("composition undo")
	}
	_ = b.BeginGroup()
	_ = b.ReplaceUTF16(7, 7, "cancelled")
	b.CancelGroup()
	if !b.CanRedo() || b.String() != "before after" {
		t.Fatal("cancel invalidated redo")
	}
	b.Redo()
	if b.String() != "before 中after" {
		t.Fatal("composition redo")
	}
}

func TestRandomEditsAndUndoMatchReference(t *testing.T) {
	rng := rand.New(rand.NewSource(19))
	b := New(strings.Repeat("甲🙂z\n", 1300))
	b.SetHistoryLimits(2000, 64<<20)
	reference := b.String()
	before := []string{reference}
	words := []string{"", "🙂", "汉字", "\r\n", "e\u0301", strings.Repeat("y", 5000)}
	for i := 0; i < 350; i++ {
		runes := []rune(reference)
		a := rng.Intn(len(runes) + 1)
		z := a + rng.Intn(len(runes)-a+1)
		value := words[rng.Intn(len(words))]
		left, right := len(utf16.Encode(runes[:a])), len(utf16.Encode(runes[:z]))
		if err := b.ReplaceUTF16(left, right, value); err != nil {
			t.Fatal(err)
		}
		next := string(runes[:a]) + value + string(runes[z:])
		if next != reference {
			before = append(before, next)
		}
		reference = next
		if b.String() != reference || b.LenUTF16() != UTF16Len(reference) {
			t.Fatalf("edit %d differs", i)
		}
		if !b.EqualSnapshot(New(reference).Snapshot()) {
			t.Fatalf("fingerprint %d differs", i)
		}
	}
	for i := len(before) - 2; i >= 0; i-- {
		if !b.Undo() || b.String() != before[i] {
			t.Fatalf("undo %d differs", i)
		}
	}
	if b.CanUndo() {
		t.Fatal("extra undo")
	}
	for i := 1; i < len(before); i++ {
		if !b.Redo() || b.String() != before[i] {
			t.Fatalf("redo %d differs", i)
		}
	}
}

func TestSnapshotsBudgetAndLineQueries(t *testing.T) {
	b := New("a\n🙂\n")
	saved := b.Snapshot()
	if b.LineCount() != 3 {
		t.Fatal(b.LineCount())
	}
	if value, _ := b.Line(2); value != "" {
		t.Fatal(value)
	}
	if line, column, err := b.LineColumn(4); err != nil || line != 1 || column != 2 {
		t.Fatal(line, column, err)
	}
	b.SetHistoryLimits(2, 100)
	for _, s := range []string{"x", "y", "z"} {
		_ = b.ReplaceUTF16(0, 0, s)
	}
	if saved.String() != "a\n🙂\n" || b.EqualSnapshot(saved) {
		t.Fatal("mutable snapshot")
	}
	if b.History().UndoDepth != 2 || !b.History().Truncated {
		t.Fatal(b.History())
	}
	b.Restore(saved)
	if b.String() != saved.String() || b.CanUndo() || !b.EqualSnapshot(saved) {
		t.Fatal("restore")
	}
	_ = b.ReplaceUTF16(0, 1, "q")
	if b.EqualSnapshot(saved) {
		t.Fatal("same-size edit hidden")
	}
	b.Undo()
	if !b.EqualSnapshot(saved) {
		t.Fatal("undo not equal")
	}
}

func BenchmarkMiddleEdit(b *testing.B) {
	doc := New(strings.Repeat("A line of 中文 content.\n", 50000))
	doc.SetHistoryLimits(128, 1<<20)
	pos := doc.LenUTF16() / 2
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = doc.ReplaceUTF16(pos, pos, "x")
		doc.Undo()
	}
}
