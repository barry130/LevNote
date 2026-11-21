package text

import (
	"errors"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
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

func TestPlanLineEditSetDirectedMappingsAndExactSeparators(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		ranges       []Range
		primary      int
		action       LineEditAction
		want         string
		after        SelectionSet
		blocks       int
	}{
		{"adjacent-up", "A\nB\nC\nD", []Range{{3, 2}, {4, 5}}, 1, MoveLinesUp, "B\nC\nA\nD", SelectionSet{[]Range{{1, 0}, {2, 3}}, 1}, 1},
		{"adjacent-copy-down", "A\nB\nC\nD", []Range{{3, 2}, {4, 5}}, 1, CopyLinesDown, "A\nB\nC\nB\nC\nD", SelectionSet{[]Range{{7, 6}, {8, 9}}, 1}, 1},
		{"separate-up", "a\nb\nc\nd\ne", []Range{{3, 2}, {6, 7}}, 1, MoveLinesUp, "b\na\nd\nc\ne", SelectionSet{[]Range{{1, 0}, {4, 5}}, 1}, 2},
		{"separate-down", "a\nb\nc\nd\ne", []Range{{3, 2}, {6, 7}}, 1, MoveLinesDown, "a\nc\nb\ne\nd", SelectionSet{[]Range{{5, 4}, {8, 9}}, 1}, 2},
		{"multiple-copy-down", "a\nb\nc", []Range{{1, 0}, {4, 5}}, 1, CopyLinesDown, "a\na\nb\nc\nc", SelectionSet{[]Range{{3, 2}, {8, 9}}, 1}, 2},
		{"multiple-copy-up", "a\nb\nc", []Range{{1, 0}, {4, 5}}, 1, CopyLinesUp, "a\na\nb\nc\nc", SelectionSet{[]Range{{1, 0}, {6, 7}}, 1}, 2},
		{"same-line-union", "abc\ndef", []Range{{0, 1}, {3, 2}}, 1, CopyLinesDown, "abc\nabc\ndef", SelectionSet{[]Range{{4, 5}, {7, 6}}, 1}, 1},
		{"outdent-one-through-five", " A\n  B\n   C\n    D\n     E", []Range{{24, 0}}, 0, OutdentLines, "A\nB\nC\nD\n E", SelectionSet{[]Range{{10, 0}}, 0}, 1},
		{"mixed-eol", "A\r\nB\nC\r\nD", []Range{{8, 3}}, 0, MoveLinesUp, "B\nC\r\nA\r\nD", SelectionSet{[]Range{{5, 0}}, 0}, 1},
		{"eof-emoji-copy", "a\r\n🙂", []Range{{5, 3}}, 0, CopyLinesDown, "a\r\n🙂\r\n🙂", SelectionSet{[]Range{{9, 7}}, 0}, 1},
		{"trailing-empty-copy", "a\r\n", []Range{{3, 3}}, 0, CopyLinesDown, "a\r\n\r\n", SelectionSet{[]Range{{5, 5}}, 0}, 1},
		{"mixed-indent", "a\r\nb\nc", []Range{{3, 4}, {6, 5}}, 1, IndentLines, "a\r\n    b\n    c", SelectionSet{[]Range{{7, 8}, {14, 13}}, 1}, 1},
		{"outdent-prefixes", "\tA\n   B\n     C", []Range{{14, 0}}, 0, OutdentLines, "A\nB\n C", SelectionSet{[]Range{{6, 0}}, 0}, 1},
		{"outdent-partial", "\tA\n   B\n     C", []Range{{2, 1}, {7, 3}}, 1, OutdentLines, "A\nB\n     C", SelectionSet{[]Range{{1, 0}, {3, 2}}, 1}, 1},
		{"high-end-left-affinity", "a\nb\nc", []Range{{0, 2}, {2, 3}}, 1, IndentLines, "    a\n    b\nc", SelectionSet{[]Range{{4, 6}, {10, 11}}, 1}, 1},
		{"trailing-empty-indent", "a\n", []Range{{0, 1}, {2, 2}}, 1, IndentLines, "    a\n    ", SelectionSet{[]Range{{4, 5}, {10, 10}}, 1}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := New(tc.source)
			before := mustSetSelections(t, b, tc.ranges, tc.primary)
			unchanged := multiStateGuard(t, b)
			plan, err := b.PlanLineEditSet(tc.action, before)
			if err != nil || len(plan.Edits) != tc.blocks || !reflect.DeepEqual(plan.After, tc.after) || plan.Changed != (tc.source != tc.want) {
				t.Fatal("plan", plan, err, "want", tc.after)
			}
			unchanged()
			if got := explicitBatchOracle(t, tc.source, plan.Edits); got != tc.want {
				t.Fatalf("oracle output %q want %q", got, tc.want)
			}
			changed, err := b.ReplaceBatchWithSelectionUTF16(b.Version(), b.SelectionGeneration(), plan.Edits, plan.After)
			if err != nil || !changed || b.String() != tc.want || b.History().UndoDepth != 1 {
				t.Fatal("apply", changed, err, b.String())
			}
			requireSelections(t, b, tc.after)
			if !b.Undo() || b.String() != tc.source {
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

func TestPlanLineEditSetEdgeAndIdenticalNoOpKeepRedo(t *testing.T) {
	for _, action := range []LineEditAction{MoveLinesUp, MoveLinesDown} {
		b := New("a\nb\nc")
		_ = b.ReplaceUTF16(0, 1, "A")
		_ = b.Undo()
		before := mustSetSelections(t, b, []Range{{1, 0}, {4, 5}}, 1)
		unchanged := multiStateGuard(t, b)
		plan, err := b.PlanLineEditSet(action, before)
		if err != nil || plan.Changed || len(plan.Edits) != 0 || !reflect.DeepEqual(plan.After, before) {
			t.Fatal("partial edge move", plan, err)
		}
		unchanged()
		if !b.Redo() || b.String() != "A\nb\nc" {
			t.Fatal("edge lost redo")
		}
	}
	b := New("a\na")
	_ = b.ReplaceUTF16(0, 1, "A")
	_ = b.Undo()
	before := mustSetSelections(t, b, []Range{{1, 0}}, 0)
	plan, err := b.PlanLineEditSet(MoveLinesDown, before)
	if err != nil || plan.Changed || len(plan.Edits) != 1 {
		t.Fatal("identical plan", plan, err)
	}
	version, history, journal := b.Version(), b.History(), b.rangeChanges
	if changed, err := b.ReplaceBatchWithSelectionUTF16(version, b.SelectionGeneration(), plan.Edits, plan.After); err != nil || changed || b.Version() != version || b.History() != history || b.rangeChanges != journal || !b.CanRedo() {
		t.Fatal("identical selection-only damaged history", changed, err)
	}
	requireSelections(t, b, SelectionSet{[]Range{{3, 2}}, 0})
}

func TestPlanLineEditSetSingleBlockMatchesLegacy(t *testing.T) {
	for _, source := range []string{"", "a\n", "a\nb", "A\r\nB\nC", "甲\r\n🙂\r\n尾", "a\n\nb"} {
		positions := []int{0}
		units := 0
		for _, r := range source {
			units += len(utf16.Encode([]rune{r}))
			positions = append(positions, units)
		}
		for _, anchor := range positions {
			for _, head := range positions {
				for action := MoveLinesUp; action <= CopyLinesDown; action++ {
					b := New(source)
					_ = b.SetSelection(anchor, head)
					old, err := b.PlanLineEdit(action)
					if err != nil {
						t.Fatal(err)
					}
					set, err := b.PlanLineEditSet(action, b.SelectionSet())
					if err != nil || set.Changed != old.Changed || set.After.Ranges[0] != old.Selection {
						t.Fatal("legacy mismatch", source, anchor, head, action, old, set, err)
					}
					if len(set.Edits) > 0 && (len(set.Edits) != 1 || set.Edits[0] != (Replacement{old.Start, old.End, old.Text})) {
						t.Fatal("legacy edit mismatch", old, set)
					}
				}
			}
		}
	}
}

func TestPlanLineEditSetCapacityUnionAndOwnerAreReadOnly(t *testing.T) {
	for _, action := range []LineEditAction{CopyLinesUp, CopyLinesDown, IndentLines} {
		b := NewWithLimit("a\nb\nc", 8)
		before := mustSetSelections(t, b, []Range{{1, 0}, {4, 5}}, 1)
		token := mustComposition(t, b)
		unchanged := multiStateGuard(t, b)
		if _, err := b.PlanLineEditSet(action, before); !errors.Is(err, ErrSizeLimit) {
			t.Fatal("aggregate capacity accepted", action, err)
		}
		unchanged()
		if !b.OwnsComposition(token) || b.CancelCompositionGroup(token) != nil {
			t.Fatal("planner stole owner")
		}
	}
	source := strings.Repeat("x\n", 1999) + "x"
	b := New(source)
	before := mustSetSelections(t, b, []Range{{len(source), 0}}, 0)
	plan, err := b.PlanLineEditSet(IndentLines, before)
	if err != nil || len(plan.Edits) != 1 || explicitBatchOracle(t, source, plan.Edits) != strings.Repeat("    x\n", 1999)+"    x" {
		t.Fatal("one edit per logical line instead of block", len(plan.Edits), err)
	}
	for _, action := range []LineEditAction{IndentLines, OutdentLines, LineEditAction(255)} {
		unchanged := multiStateGuard(t, b)
		if _, err := b.PlanLineEdit(action); !errors.Is(err, ErrRange) {
			t.Fatal("legacy action set was expanded", action, err)
		}
		unchanged()
	}
	for _, set := range []SelectionSet{{}, {Ranges: make([]Range, 257)}, {Ranges: []Range{{-1, 0}}}} {
		unchanged := multiStateGuard(t, b)
		if _, err := b.PlanLineEditSet(IndentLines, set); err == nil {
			t.Fatal("invalid set accepted")
		}
		unchanged()
	}
}

func TestPlanLineEditSet256SeparateBlocksAnd257Refusal(t *testing.T) {
	source := strings.Repeat("a\nx\n", 256) + "tail"
	b := New(source)
	before := SelectionSet{Ranges: make([]Range, 256), Primary: 255}
	after := SelectionSet{Ranges: make([]Range, 256), Primary: 255}
	for i := range before.Ranges {
		before.Ranges[i] = Range{4*i + 1, 4*i + 1}
		after.Ranges[i] = Range{4*i + 3, 4*i + 3}
	}
	mustSetSelections(t, b, before.Ranges, before.Primary)
	unchanged := multiStateGuard(t, b)
	bad := SelectionSet{Ranges: append(append([]Range(nil), before.Ranges...), Range{len(source), len(source)}), Primary: 255}
	if _, err := b.PlanLineEditSet(MoveLinesDown, bad); !errors.Is(err, ErrSelectionLimit) {
		t.Fatal("257 line selections accepted", err)
	}
	unchanged()
	plan, err := b.PlanLineEditSet(MoveLinesDown, before)
	if err != nil || len(plan.Edits) != 256 || !reflect.DeepEqual(plan.After, after) {
		t.Fatal("256 separate line blocks", len(plan.Edits), plan.After, err)
	}
	unchanged()
	if changed, err := b.ReplaceBatchWithSelectionUTF16(b.Version(), b.SelectionGeneration(), plan.Edits, plan.After); err != nil || !changed || b.String() != strings.Repeat("x\na\n", 256)+"tail" {
		t.Fatal("256 blocks apply", changed, err)
	}
	requireSelections(t, b, after)
	if !b.Undo() || b.String() != source {
		t.Fatal("256 blocks undo")
	}
	requireSelections(t, b, before)
	if !b.Redo() {
		t.Fatal("256 blocks redo")
	}
	requireSelections(t, b, after)
}

// lineSetOracle operates on an independent LF-delimited line list. It never
// calls production line coordinates, planners, endpoint mapping or normalizers.
// Mixed separators and separator-interior endpoints have separate exact fixtures.
func lineSetOracle(lines []string, selected []int, primary int, action LineEditAction) (string, SelectionSet) {
	units := func(s string) int { return len(utf16.Encode([]rune(s))) }
	chosen := make([]bool, len(lines))
	for _, line := range selected {
		chosen[line] = true
	}
	type block struct{ first, last int }
	var blocks []block
	for i := 0; i < len(lines); i++ {
		if !chosen[i] {
			continue
		}
		first := i
		for i+1 < len(lines) && chosen[i+1] {
			i++
		}
		blocks = append(blocks, block{first, i})
	}
	order := make([]int, len(lines))
	for i := range order {
		order[i] = i
	}
	edge := action == MoveLinesUp && chosen[0] || action == MoveLinesDown && chosen[len(lines)-1]
	if !edge {
		for _, block := range blocks {
			if action == MoveLinesUp {
				copy(order[block.first-1:block.last], order[block.first:block.last+1])
				order[block.last] = block.first - 1
			} else if action == MoveLinesDown {
				for i := block.last + 1; i > block.first; i-- {
					order[i] = i - 1
				}
				order[block.first] = block.last + 1
			}
		}
	}
	out := make([]string, 0, 2*len(lines))
	dest := make([]int, len(lines))
	appendOriginal := func(line int, target bool) {
		if target {
			dest[line] = len(out)
		}
		body := lines[line]
		if chosen[line] && action == IndentLines {
			body = "    " + body
		} else if chosen[line] && action == OutdentLines {
			if strings.HasPrefix(body, "\t") {
				body = body[1:]
			} else {
				n := 0
				for n < len(body) && n < 4 && body[n] == ' ' {
					n++
				}
				body = body[n:]
			}
		}
		out = append(out, body)
	}
	if action == CopyLinesUp || action == CopyLinesDown {
		at := 0
		for _, block := range blocks {
			for at < block.first {
				appendOriginal(at, false)
				at++
			}
			for repeat := 0; repeat < 2; repeat++ {
				for line := block.first; line <= block.last; line++ {
					appendOriginal(line, repeat == 0 && action == CopyLinesUp || repeat == 1 && action == CopyLinesDown)
				}
			}
			at = block.last + 1
		}
		for at < len(lines) {
			appendOriginal(at, false)
			at++
		}
	} else {
		for _, line := range order {
			appendOriginal(line, true)
		}
	}
	starts := make([]int, len(out))
	for i := 1; i < len(out); i++ {
		starts[i] = starts[i-1] + units(out[i-1]) + 1
	}
	after := SelectionSet{Ranges: make([]Range, len(selected)), Primary: primary}
	for i, line := range selected {
		end := starts[dest[line]] + units(out[dest[line]])
		after.Ranges[i] = Range{end, end}
	}
	return strings.Join(out, "\n"), after
}

func TestPlanLineEditSetRandomIndependentLineListOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(8520261120))
	words := []string{"🙂", "汉", "e\u0301", "abc"}
	prefixes := []string{"", "\t", " ", "   ", "     "}
	for trial := 0; trial < 180; trial++ {
		n := 5 + rng.Intn(12)
		lines := make([]string, n)
		for i := range lines {
			lines[i] = prefixes[rng.Intn(len(prefixes))] + string(rune('A'+i)) + words[rng.Intn(len(words))]
		}
		var selected []int
		for i := range lines {
			if rng.Intn(3) == 0 {
				selected = append(selected, i)
			}
		}
		if len(selected) == 0 {
			selected = []int{trial % n}
		}
		primary := trial % len(selected)
		source := strings.Join(lines, "\n")
		before := SelectionSet{Ranges: make([]Range, len(selected)), Primary: primary}
		at, next := 0, 0
		for line, value := range lines {
			at += len(utf16.Encode([]rune(value)))
			if next < len(selected) && selected[next] == line {
				before.Ranges[next] = Range{at, at}
				next++
			}
			at++
		}
		for action := MoveLinesUp; action <= OutdentLines; action++ {
			b := New(source)
			mustSetSelections(t, b, before.Ranges, before.Primary)
			want, after := lineSetOracle(lines, selected, primary, action)
			unchanged := multiStateGuard(t, b)
			plan, err := b.PlanLineEditSet(action, before)
			if err != nil || plan.Changed != (source != want) || !reflect.DeepEqual(plan.After, after) {
				t.Fatal("line oracle plan", trial, action, plan, err, want, after)
			}
			unchanged()
			if len(plan.Edits) == 0 {
				if source != want || !reflect.DeepEqual(before, after) {
					t.Fatal("bad edge no-op")
				}
				continue
			}
			changed, err := b.ReplaceBatchWithSelectionUTF16(b.Version(), b.SelectionGeneration(), plan.Edits, plan.After)
			if err != nil || changed != (source != want) || b.String() != want {
				t.Fatal("line oracle apply", trial, action, changed, err, b.String(), want)
			}
			requireSelections(t, b, after)
			if changed {
				if !b.Undo() || b.String() != source {
					t.Fatal("line oracle undo")
				}
				requireSelections(t, b, before)
				if !b.Redo() || b.String() != want {
					t.Fatal("line oracle redo")
				}
				requireSelections(t, b, after)
			}
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
