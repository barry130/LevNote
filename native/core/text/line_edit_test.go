package text

import (
	"errors"
	"strings"
	"testing"
)

func TestPlanLineEditTextSelectionAndHistory(t *testing.T) {
	cases := []struct {
		name      string
		source    string
		selection Range
		action    LineEditAction
		want      string
		selected  Range
	}{
		{"up", "one\ntwo\nthree", Range{5, 6}, MoveLinesUp, "two\none\nthree", Range{1, 2}},
		{"down", "one\ntwo\nthree", Range{5, 6}, MoveLinesDown, "one\nthree\ntwo", Range{11, 12}},
		{"block-end-next-start", "a\nb\nc\nd", Range{2, 6}, MoveLinesUp, "b\nc\na\nd", Range{0, 4}},
		{"reversed-block-down", "a\nb\nc\nd", Range{6, 2}, MoveLinesDown, "a\nd\nb\nc", Range{7, 4}},
		{"last-line-up", "a\nb", Range{2, 3}, MoveLinesUp, "b\na", Range{0, 1}},
		{"first-line-down-to-eof", "a\nb", Range{0, 2}, MoveLinesDown, "b\na", Range{2, 3}},
		{"empty-line-up", "a\n\nb", Range{2, 2}, MoveLinesUp, "\na\nb", Range{0, 0}},
		{"trailing-empty-up", "a\n", Range{2, 2}, MoveLinesUp, "\na", Range{0, 0}},
		{"crlf-unicode-up", "甲\r\n🙂\r\n尾", Range{5, 3}, MoveLinesUp, "🙂\r\n甲\r\n尾", Range{2, 0}},
		{"mixed-block-up", "A\r\nB\nC\r\nD", Range{3, 8}, MoveLinesUp, "B\nC\r\nA\r\nD", Range{0, 5}},
		{"mixed-endpoint-down", "A\r\nB\nC", Range{0, 3}, MoveLinesDown, "B\r\nA\nC", Range{3, 5}},
		{"crlf-interior-endpoint", "A\r\nB\nC", Range{0, 2}, MoveLinesDown, "B\r\nA\nC", Range{3, 5}},
		{"copy-up", "one\ntwo\nthree", Range{5, 6}, CopyLinesUp, "one\ntwo\ntwo\nthree", Range{5, 6}},
		{"copy-down", "one\ntwo\nthree", Range{5, 6}, CopyLinesDown, "one\ntwo\ntwo\nthree", Range{9, 10}},
		{"copy-reversed-block", "a\nb\nc\nd", Range{6, 2}, CopyLinesDown, "a\nb\nc\nb\nc\nd", Range{10, 6}},
		{"copy-mixed-block", "A\r\nB\nC\r\nD", Range{3, 8}, CopyLinesDown, "A\r\nB\nC\r\nB\nC\r\nD", Range{8, 13}},
		{"copy-eof-up", "a\r\n🙂", Range{5, 3}, CopyLinesUp, "a\r\n🙂\r\n🙂", Range{5, 3}},
		{"copy-eof-down", "a\r\n🙂", Range{5, 3}, CopyLinesDown, "a\r\n🙂\r\n🙂", Range{9, 7}},
		{"copy-empty-document", "", Range{0, 0}, CopyLinesDown, "\n", Range{1, 1}},
		{"copy-trailing-empty", "a\r\n", Range{3, 3}, CopyLinesDown, "a\r\n\r\n", Range{5, 5}},
		{"copy-single-line", "🙂", Range{2, 2}, CopyLinesDown, "🙂\n🙂", Range{5, 5}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			b := New(tt.source)
			if err := b.SetSelection(tt.selection.Anchor, tt.selection.Head); err != nil {
				t.Fatal(err)
			}
			version, history, snapshot := b.Version(), b.History(), b.Snapshot()
			plan, err := b.PlanLineEdit(tt.action)
			if err != nil {
				t.Fatal(err)
			}
			if b.Version() != version || b.Selection() != tt.selection || b.History() != history || !b.EqualSnapshot(snapshot) || b.InGroup() {
				t.Fatal("planning mutated the source")
			}
			if !plan.Changed || plan.Selection != tt.selected {
				t.Fatalf("plan=%+v; want selection=%+v", plan, tt.selected)
			}
			if err := b.CheckReplaceUTF16(plan.Start, plan.End, plan.Text); err != nil {
				t.Fatal(err)
			}
			if err := b.BeginGroup(); err != nil {
				t.Fatal(err)
			}
			if err := b.ReplaceUTF16(plan.Start, plan.End, plan.Text); err != nil {
				t.Fatal(err)
			}
			if err := b.SetSelection(plan.Selection.Anchor, plan.Selection.Head); err != nil {
				t.Fatal(err)
			}
			if !b.EndGroup() || b.History().UndoDepth != 1 || b.String() != tt.want {
				t.Fatalf("result=%q history=%+v; want %q", b.String(), b.History(), tt.want)
			}
			if !b.Undo() || b.String() != tt.source || b.Selection() != tt.selection || b.CanUndo() {
				t.Fatal("undo did not restore source and selection")
			}
			if !b.Redo() || b.String() != tt.want || b.Selection() != tt.selected || b.CanRedo() {
				t.Fatal("redo did not restore planned result")
			}
		})
	}
}

func TestPlanLineEditBoundaryAndIdenticalNoHistory(t *testing.T) {
	for _, tt := range []struct {
		source    string
		selection Range
		action    LineEditAction
		selected  Range
	}{
		{"", Range{}, MoveLinesUp, Range{}},
		{"", Range{}, MoveLinesDown, Range{}},
		{"a\nb", Range{0, 1}, MoveLinesUp, Range{0, 1}},
		{"a\nb", Range{3, 2}, MoveLinesDown, Range{3, 2}},
		{"a\nb", Range{0, 3}, MoveLinesDown, Range{0, 3}},
		{"a\na", Range{0, 1}, MoveLinesDown, Range{2, 3}},
		{"a\na", Range{2, 3}, MoveLinesUp, Range{0, 1}},
	} {
		b := New(tt.source)
		_ = b.SetSelection(tt.selection.Anchor, tt.selection.Head)
		version, history := b.Version(), b.History()
		plan, err := b.PlanLineEdit(tt.action)
		if err != nil || plan.Changed || plan.Selection != tt.selected || b.Version() != version || b.History() != history || b.Selection() != tt.selection {
			t.Fatalf("%q %v plan=%+v err=%v", tt.source, tt.action, plan, err)
		}
	}
}

func TestPlanLineEditCopyCapacityKeepsRedoAndComposition(t *testing.T) {
	for _, action := range []LineEditAction{CopyLinesUp, CopyLinesDown} {
		b := NewWithLimit("甲\r\n🙂", len("甲\r\n🙂")+1)
		if err := b.ReplaceUTF16(0, 1, "乙"); err != nil {
			t.Fatal(err)
		}
		if !b.Undo() {
			t.Fatal("seed redo")
		}
		_ = b.SetSelection(3, 5)
		version, history, selection, snapshot := b.Version(), b.History(), b.Selection(), b.Snapshot()
		if _, err := b.PlanLineEdit(action); !errors.Is(err, ErrSizeLimit) {
			t.Fatal("copy should exceed byte capacity", err)
		}
		if b.Version() != version || b.History() != history || b.Selection() != selection || !b.EqualSnapshot(snapshot) {
			t.Fatal("rejected plan damaged redo/source")
		}

		token, err := b.BeginCompositionGroup()
		if err != nil {
			t.Fatal(err)
		}
		if err := b.ReplaceCompositionUTF16(token, 3, 5, "好"); err != nil {
			t.Fatal(err)
		}
		_ = b.SetSelection(4, 4)
		version, history, selection, snapshot = b.Version(), b.History(), b.Selection(), b.Snapshot()
		if _, err := b.PlanLineEdit(action); !errors.Is(err, ErrSizeLimit) {
			t.Fatal("copy should retain active composition", err)
		}
		if !b.OwnsComposition(token) || b.Version() != version || b.History() != history || b.Selection() != selection || !b.EqualSnapshot(snapshot) {
			t.Fatal("plan settled or changed composition")
		}
		if err := b.CancelCompositionGroup(token); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPlanLineEditCapacityExactAndLongLogicalLine(t *testing.T) {
	line := strings.Repeat("🙂", 3000)
	for _, action := range []LineEditAction{MoveLinesUp, CopyLinesDown} {
		source := "first\r\n" + line + "\r\nlast"
		limit := len(source)
		if action == CopyLinesDown {
			limit += len(line) + 2
		}
		b := NewWithLimit(source, limit)
		_ = b.SetSelection(9, 9)
		plan, err := b.PlanLineEdit(action)
		if err != nil || !plan.Changed {
			t.Fatal("whole logical line should fit", err)
		}
		if err := b.ReplaceUTF16(plan.Start, plan.End, plan.Text); err != nil {
			t.Fatal(err)
		}
		want := line + "\r\nfirst\r\nlast"
		if action == CopyLinesDown {
			want = "first\r\n" + line + "\r\n" + line + "\r\nlast"
		}
		if b.String() != want || b.LenBytes() != limit {
			t.Fatal("long line was truncated or limit accounting changed")
		}
	}
}

func TestPlanLineEditInvalidActionIsReadOnly(t *testing.T) {
	b := New("a\nb")
	version, history, selection := b.Version(), b.History(), b.Selection()
	if _, err := b.PlanLineEdit(LineEditAction(255)); !errors.Is(err, ErrRange) {
		t.Fatal(err)
	}
	if b.Version() != version || b.History() != history || b.Selection() != selection || b.String() != "a\nb" {
		t.Fatal("invalid action mutated buffer")
	}
}
