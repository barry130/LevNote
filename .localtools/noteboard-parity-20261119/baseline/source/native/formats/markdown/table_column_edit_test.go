package markdown

import (
	"errors"
	"strings"
	"testing"
)

func TestTableColumnPlanExactBytes(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		column   int
		action   TableColumnAction
		expected string
	}{
		{
			name: "outer-before-middle",
			source: "| H1 | H2 | H3 |\n" +
				"| :--- | ---: | :---: |\n" +
				"| A | B | C |\n" +
				"| D | E | F |\n",
			column: 1,
			action: TableColumnInsertBefore,
			expected: "| H1 || H2 | H3 |\n" +
				"| :--- |---| ---: | :---: |\n" +
				"| A || B | C |\n" +
				"| D || E | F |\n",
		},
		{
			name: "outer-after-last",
			source: "| H1 | H2 |\n" +
				"| --- | --- |\n" +
				"| A | B |\n",
			column: 1,
			action: TableColumnInsertAfter,
			expected: "| H1 | H2 ||\n" +
				"| --- | --- |---|\n" +
				"| A | B ||\n",
		},
		{
			name: "outer-delete-middle",
			source: "| H1 | H2 | H3 |\n" +
				"| :--- | ---: | :---: |\n" +
				"| A | B | C |\n",
			column: 1,
			action: TableColumnDelete,
			expected: "| H1 | H3 |\n" +
				"| :--- | :---: |\n" +
				"| A | C |\n",
		},
		{
			name: "seven-to-eight-insert-after",
			source: "|a|b|c|d|e|f|g|\n" +
				"|---|---|---|---|---|---|---|\n" +
				"|1|2|3|4|5|6|7|\n",
			column: 6,
			action: TableColumnInsertAfter,
			expected: "|a|b|c|d|e|f|g||\n" +
				"|---|---|---|---|---|---|---|---|\n" +
				"|1|2|3|4|5|6|7||\n",
		},
		{
			name: "even-trailing-backslash-insert-after",
			source: "| A | B\\\\\n" +
				"|--- | ---\n" +
				"| C | D\\\\\n",
			column: 1,
			action: TableColumnInsertAfter,
			expected: "| A | B\\\\||\n" +
				"|--- | ---|---|\n" +
				"| C | D\\\\||\n",
		},
		{
			name: "no-outer-before-first",
			source: "H1 | H2 | H3\n" +
				":--- | ---: | :---:\n" +
				"A | B | C",
			column: 0,
			action: TableColumnInsertBefore,
			expected: "||H1 | H2 | H3\n" +
				"|---|:--- | ---: | :---:\n" +
				"||A | B | C",
		},
		{
			name: "no-outer-after-last",
			source: "H1 | H2 | H3\n" +
				":--- | ---: | :---:\n" +
				"A | B | C",
			column: 2,
			action: TableColumnInsertAfter,
			expected: "H1 | H2 | H3||\n" +
				":--- | ---: | :---:|---|\n" +
				"A | B | C||",
		},
		{
			name: "no-outer-delete-middle",
			source: "H1 | H2 | H3\n" +
				":--- | ---: | :---:\n" +
				"A | B | C",
			column: 1,
			action: TableColumnDelete,
			expected: "H1 | H3\n" +
				":--- | :---:\n" +
				"A | C",
		},
		{
			name: "leading-only-insert-last",
			source: "| H1 | H2\n" +
				"| --- | ---\n" +
				"| A | B\n",
			column: 1,
			action: TableColumnInsertAfter,
			expected: "| H1 | H2||\n" +
				"| --- | ---|---|\n" +
				"| A | B||\n",
		},
		{
			name: "leading-only-append-after",
			source: "| H\n" +
				"|---\n" +
				"| A\n",
			column: 0,
			action: TableColumnInsertAfter,
			expected: "| H||\n" +
				"|---|---|\n" +
				"| A||\n",
		},
		{
			name: "leading-only-delete-last-empty-middle",
			source: "| H1 || H3\n" +
				"| --- | --- | ---\n" +
				"| A1 || A3\n",
			column: 2,
			action: TableColumnDelete,
			expected: "| H1 ||\n" +
				"| --- | --- \n" +
				"| A1 ||\n",
		},
		{
			name: "leading-only-delete-last-padding-middle",
			source: "| H1 |  | H3\n" +
				"| --- | --- | ---\n" +
				"| A1 |  | A3\n",
			column: 2,
			action: TableColumnDelete,
			expected: "| H1 |  |\n" +
				"| --- | --- \n" +
				"| A1 |  |\n",
		},
		{
			name: "trailing-only-insert-first",
			source: "H1 | H2 |\n" +
				"--- | --- |\n" +
				"A | B |\n",
			column: 0,
			action: TableColumnInsertBefore,
			expected: "||H1 | H2 |\n" +
				"|---|--- | --- |\n" +
				"||A | B |\n",
		},
		{
			name: "two-to-one-no-leading-delete-first",
			source: "A | B\n" +
				"--- | ---\n" +
				"C | D",
			column: 0,
			action: TableColumnDelete,
			expected: "| B\n" +
				"| ---\n" +
				"| D",
		},
		{
			name: "two-to-one-no-leading-delete-last",
			source: "A | B\n" +
				"--- | ---\n" +
				"C | D",
			column: 1,
			action: TableColumnDelete,
			expected: "|A \n" +
				"|--- \n" +
				"|C ",
		},
		{
			name: "indented-no-leading-delete-first",
			source: "  A | B\n" +
				"  --- | ---\n" +
				"  C | D\n",
			column: 0,
			action: TableColumnDelete,
			expected: "  | B\n" +
				"  | ---\n" +
				"  | D\n",
		},
		{
			name: "single-column-normal-insert-before",
			source: "| A |\n" +
				"|---|\n" +
				"| C |\n",
			column: 0,
			action: TableColumnInsertBefore,
			expected: "|| A |\n" +
				"|---|---|\n" +
				"|| C |\n",
		},
		{
			name: "single-column-leading-only-insert-before",
			source: "| A\n" +
				"|---\n" +
				"| C\n",
			column: 0,
			action: TableColumnInsertBefore,
			expected: "|| A\n" +
				"|---|---\n" +
				"|| C\n",
		},
		{
			name: "single-column-leading-only-insert-after",
			source: "| A\n" +
				"|---\n" +
				"| C\n",
			column: 0,
			action: TableColumnInsertAfter,
			expected: "| A||\n" +
				"|---|---|\n" +
				"| C||\n",
		},
		{
			name: "single-column-degenerate-insert-after",
			source: "|\n" +
				"|---|\n" +
				"|\n",
			column: 0,
			action: TableColumnInsertAfter,
			expected: "|||\n" +
				"|---|---|\n" +
				"|||\n",
		},
		{
			name: "single-column-degenerate-insert-before",
			source: "|\n" +
				"|---|\n" +
				"|\n",
			column: 0,
			action: TableColumnInsertBefore,
			expected: "|||\n" +
				"|---|---|\n" +
				"|||\n",
		},
		{
			name: "single-column-padding-insert-before",
			source: "|  \n" +
				"|---|\n" +
				"|  \n",
			column: 0,
			action: TableColumnInsertBefore,
			expected: "||  |\n" +
				"|---|---|\n" +
				"||  |\n",
		},
		{
			name: "single-column-padding-insert-after",
			source: "|  \n" +
				"|---|\n" +
				"|  \n",
			column: 0,
			action: TableColumnInsertAfter,
			expected: "|  ||\n" +
				"|---|---|\n" +
				"|  ||\n",
		},
		{
			name: "escaped-pipe-and-unicode-delete-first",
			source: "| 甲 \\| 左 | 😀 `代码\\|值` |\n" +
				"| --- | --- |\n" +
				"| 前 \\| 中 | 后 |\n",
			column: 0,
			action: TableColumnDelete,
			expected: "| 😀 `代码\\|值` |\n" +
				"| --- |\n" +
				"| 后 |\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			block := columnTestTable(t, test.source)
			anchor := block.Table.Rows[len(block.Table.Rows)-1].Range
			patch, err := PlanTableColumnEdit(test.source, block.Range, anchor, test.column, test.action)
			if err != nil {
				t.Fatalf("PlanTableColumnEdit: %v", err)
			}
			if len(patch.Edits) == 0 {
				t.Fatal("successful plan has no edits")
			}
			previousEnd := -1
			for _, edit := range patch.Edits {
				if edit.Range.Start < previousEnd || edit.Range.End < edit.Range.Start || edit.Range.End > len(test.source) {
					t.Fatalf("invalid or unordered edit %+v in %+v", edit, patch.Edits)
				}
				previousEnd = edit.Range.End
			}
			if got := applyColumnTestPatch(t, test.source, patch); got != test.expected {
				t.Fatalf("candidate bytes:\n got %q\nwant %q", got, test.expected)
			}
			gotBlock := columnTestTable(t, test.expected)
			wantColumns := len(block.Table.Alignments)
			if test.action == TableColumnDelete {
				wantColumns--
			} else {
				wantColumns++
			}
			if len(gotBlock.Table.Alignments) != wantColumns {
				t.Fatalf("candidate columns=%d want=%d", len(gotBlock.Table.Alignments), wantColumns)
			}
		})
	}
}

func TestTableColumnCandidateExactRawLimitRefusal(t *testing.T) {
	source := exactColumnLimitTable(t, maxTableRawBytes)
	lines, complete := splitLines(source)
	if !complete || len(lines) != maxTableBodyRows+2 {
		t.Fatalf("exact-limit lines=%d complete=%v", len(lines), complete)
	}
	for i, line := range lines {
		scan := scanTableRow(source, line)
		if !scan.valid || len(scan.row.Cells) != 7 {
			t.Fatalf("exact-limit line=%d bytes=%d pipes=%d cells=%d valid=%v mask=%+v",
				i, len(line.content), scan.pipes, len(scan.row.Cells), scan.valid, scan)
		}
	}
	doc := Parse(source)
	if doc.Partial || len(doc.Blocks) != 1 || doc.Blocks[0].Kind != BlockTable ||
		doc.Blocks[0].Table.Problem != "" || doc.Blocks[0].Range.End-doc.Blocks[0].Range.Start != maxTableRawBytes {
		t.Fatalf("exact-limit fixture partial=%v blocks=%d", doc.Partial, len(doc.Blocks))
	}
	block := doc.Blocks[0]
	patch, err := PlanTableColumnEdit(
		source, block.Range, block.Table.Rows[0].Range, len(block.Table.Alignments)-1,
		TableColumnInsertAfter,
	)
	if !errors.Is(err, ErrTableColumnLimit) || len(patch.Edits) != 0 {
		t.Fatalf("candidate-over-exact-limit patch=%+v err=%v", patch, err)
	}
}

func TestTableColumnPlanNewlineFormsAndNeighbors(t *testing.T) {
	for _, ending := range []string{"\n", "\r\n", "\r"} {
		t.Run(strings.ReplaceAll(ending, "\n", "LF"), func(t *testing.T) {
			source := strings.Join([]string{
				"# 前",
				"",
				"| A | B |",
				"| --- | --- |",
				"| C | D |",
				"",
				"```txt",
				"literal",
				"```",
				"# 后",
			}, ending)
			want := strings.Replace(source, "| A | B |", "| A || B |", 1)
			want = strings.Replace(want, "| --- | --- |", "| --- |---| --- |", 1)
			want = strings.Replace(want, "| C | D |", "| C || D |", 1)
			block := columnTestTable(t, source)
			patch, err := PlanTableColumnEdit(source, block.Range, block.Table.Rows[1].Range, 1, TableColumnInsertBefore)
			if err != nil {
				t.Fatal(err)
			}
			if got := applyColumnTestPatch(t, source, patch); got != want {
				t.Fatalf("line ending %q changed unrelated bytes:\n got %q\nwant %q", ending, got, want)
			}
		})
	}

	source := "# 前\n\n| A | B |\n| --- | --- |\n| C | D |\n\n> 后"
	want := strings.Replace(source, "| C | D |", "| C || D |", 1)
	want = strings.Replace(want, "| --- | --- |", "| --- |---| --- |", 1)
	want = strings.Replace(want, "| A | B |", "| A || B |", 1)
	block := columnTestTable(t, source)
	patch, err := PlanTableColumnEdit(source, block.Range, block.Table.Rows[0].Range, 1, TableColumnInsertBefore)
	if err != nil {
		t.Fatal(err)
	}
	if got := applyColumnTestPatch(t, source, patch); got != want {
		t.Fatalf("final unterminated neighbor source changed:\n got %q\nwant %q", got, want)
	}
}

func TestTableColumnPlanWarningAndDescriptorState(t *testing.T) {
	t.Run("neighbor-warning-remains", func(t *testing.T) {
		source := "| A | B |\n| --- | --- |\n| C | D |\n\n| bad |\n| --- | --- |\n"
		before := Parse(source)
		if len(before.Warnings) != 1 {
			t.Fatalf("fixture warnings=%+v", before.Warnings)
		}
		block := columnTestTable(t, source)
		patch, err := PlanTableColumnEdit(source, block.Range, block.Table.Rows[0].Range, 1, TableColumnInsertAfter)
		if err != nil {
			t.Fatal(err)
		}
		candidate := applyColumnTestPatch(t, source, patch)
		after := Parse(candidate)
		if len(after.Warnings) != 1 || after.Warnings[0].Code != before.Warnings[0].Code ||
			after.Warnings[0].Message != before.Warnings[0].Message ||
			after.Warnings[0].Range.Start-before.Warnings[0].Range.Start != len(candidate)-len(source) {
			t.Fatalf("warning was not translated exactly: before=%+v after=%+v", before.Warnings, after.Warnings)
		}
	})

	t.Run("neighbor-demotion-refused", func(t *testing.T) {
		source := columnDescriptorTables(t, 7, 6, 252)
		before := Parse(source)
		target := before.Blocks[0]
		if target.Table.Problem != "" || len(before.Warnings) != 0 {
			t.Fatalf("demotion fixture shape target=%+v warnings=%+v", target.Table.Problem, before.Warnings)
		}
		patch, err := PlanTableColumnEdit(source, target.Range, target.Table.Rows[0].Range, 0, TableColumnInsertAfter)
		if !errors.Is(err, ErrTableColumnContext) || len(patch.Edits) != 0 {
			t.Fatalf("descriptor demotion patch=%+v err=%v", patch, err)
		}
	})

	t.Run("neighbor-promotion-refused", func(t *testing.T) {
		source := columnDescriptorTables(t, 8, 6, 256)
		before := Parse(source)
		target := before.Blocks[0]
		if target.Table.Problem != "" || len(before.Warnings) != 1 || before.Warnings[0].Code != "unsupported-table" {
			t.Fatalf("promotion fixture shape target=%+v warnings=%+v", target.Table.Problem, before.Warnings)
		}
		patch, err := PlanTableColumnEdit(source, target.Range, target.Table.Rows[0].Range, 0, TableColumnDelete)
		if !errors.Is(err, ErrTableColumnContext) || len(patch.Edits) != 0 {
			t.Fatalf("descriptor promotion patch=%+v err=%v", patch, err)
		}
	})
}

func TestTableColumnPlanRefusesInvalidTargetsAndLimits(t *testing.T) {
	source := "| A | B |\n| --- | --- |\n| C | D |\n"
	block := columnTestTable(t, source)
	header := block.Table.Rows[0].Range

	checkZero := func(t *testing.T, patch TableColumnPatch, err, want error) {
		t.Helper()
		if !errors.Is(err, want) || len(patch.Edits) != 0 {
			t.Fatalf("patch=%+v err=%v want %v", patch, err, want)
		}
	}
	t.Run("unknown-action", func(t *testing.T) {
		patch, err := PlanTableColumnEdit(source, block.Range, header, 0, TableColumnAction(0))
		checkZero(t, patch, err, ErrTableColumnAction)
	})
	t.Run("invalid-utf8", func(t *testing.T) {
		patch, err := PlanTableColumnEdit(source+"\xff", block.Range, header, 0, TableColumnInsertBefore)
		checkZero(t, patch, err, ErrTableColumnTarget)
	})
	t.Run("wrong-table", func(t *testing.T) {
		patch, err := PlanTableColumnEdit(source, rangeAt(0, 5), header, 0, TableColumnInsertBefore)
		checkZero(t, patch, err, ErrTableColumnTarget)
	})
	t.Run("wrong-anchor", func(t *testing.T) {
		patch, err := PlanTableColumnEdit(source, block.Range, rangeAt(0, 1), 0, TableColumnInsertBefore)
		checkZero(t, patch, err, ErrTableColumnTarget)
	})
	t.Run("column-bounds", func(t *testing.T) {
		for _, column := range []int{-1, 2} {
			patch, err := PlanTableColumnEdit(source, block.Range, header, column, TableColumnInsertBefore)
			checkZero(t, patch, err, ErrTableColumnTarget)
		}
	})
	t.Run("one-column-minimum", func(t *testing.T) {
		one := "| A |\n|---|\n| C |\n"
		oneBlock := columnTestTable(t, one)
		patch, err := PlanTableColumnEdit(one, oneBlock.Range, oneBlock.Table.Rows[0].Range, 0, TableColumnDelete)
		checkZero(t, patch, err, ErrTableColumnMinimum)
	})
	t.Run("eight-column-maximum", func(t *testing.T) {
		eight := "|a|b|c|d|e|f|g|h|\n|---|---|---|---|---|---|---|---|\n|1|2|3|4|5|6|7|8|\n"
		eightBlock := columnTestTable(t, eight)
		patch, err := PlanTableColumnEdit(eight, eightBlock.Range, eightBlock.Table.Rows[0].Range, 0, TableColumnInsertAfter)
		checkZero(t, patch, err, ErrTableColumnLimit)
	})
	t.Run("ragged-and-unsupported", func(t *testing.T) {
		for _, bad := range []string{
			"| A | B |\n| --- | --- |\n| C |\n",
			"| A | B |\n| --- | --- |\n| `C | D` | E |\n",
		} {
			doc := Parse(bad)
			var table Block
			for _, candidate := range doc.Blocks {
				if candidate.Kind == BlockTable {
					table = candidate
					break
				}
			}
			patch, err := PlanTableColumnEdit(bad, table.Range, rangeAt(table.Range.Start, table.Range.Start+1), 0, TableColumnInsertBefore)
			checkZero(t, patch, err, ErrTableColumnTarget)
		}
	})
	t.Run("odd-trailing-backslash", func(t *testing.T) {
		odd := "| A | B\\\n| --- | --- |\n| C | D\\\n"
		oddBlock := columnTestTable(t, odd)
		patch, err := PlanTableColumnEdit(odd, oddBlock.Range, oddBlock.Table.Rows[1].Range, 1, TableColumnInsertAfter)
		checkZero(t, patch, err, ErrTableColumnContext)
	})
	t.Run("over-document", func(t *testing.T) {
		patch, err := PlanTableColumnEdit(strings.Repeat("x", MaxParseBytes+1), block.Range, header, 0, TableColumnInsertBefore)
		checkZero(t, patch, err, ErrTableColumnLimit)
	})
	t.Run("candidate-over-document", func(t *testing.T) {
		big := source + "```\n" + strings.Repeat("x", MaxParseBytes-len(source)-9) + "\n```\n"
		bigBlock := columnTestTable(t, big)
		patch, err := PlanTableColumnEdit(big, bigBlock.Range, bigBlock.Table.Rows[0].Range, 0, TableColumnInsertBefore)
		checkZero(t, patch, err, ErrTableColumnLimit)
	})
	t.Run("partial-document", func(t *testing.T) {
		partial := source + strings.Repeat("x\n", maxLines)
		patch, err := PlanTableColumnEdit(partial, block.Range, header, 0, TableColumnInsertBefore)
		checkZero(t, patch, err, ErrTableColumnLimit)
	})
	t.Run("over-table-bytes", func(t *testing.T) {
		cell := strings.Repeat("x", maxTableCellTextBytes)
		lines := []string{
			"|" + cell + "|b|c|d|e|f|g|h|",
			"|---|---|---|---|---|---|---|---|",
		}
		for len(strings.Join(lines, "\n")) <= maxTableRawBytes {
			lines = append(lines, "|1|2|3|4|5|6|7|8|")
		}
		over := strings.Join(lines, "\n")
		doc := Parse(over)
		var table Block
		for _, candidate := range doc.Blocks {
			if candidate.Kind == BlockTable {
				table = candidate
				break
			}
		}
		patch, err := PlanTableColumnEdit(over, table.Range, table.Range, 0, TableColumnInsertBefore)
		checkZero(t, patch, err, ErrTableColumnLimit)
	})
}

func columnTestTable(t *testing.T, source string) Block {
	t.Helper()
	doc := Parse(source)
	if doc.Partial {
		t.Fatalf("fixture is partial: %+v", doc.Warnings)
	}
	for _, block := range doc.Blocks {
		if block.Kind == BlockTable && block.Table.Problem == "" {
			return block
		}
	}
	t.Fatalf("fixture has no supported table: %+v", doc.Warnings)
	return Block{}
}

func exactColumnLimitTable(t *testing.T, targetBytes int) string {
	t.Helper()
	const columns = 7
	header := "|a|b|c|d|e|f|g|"
	delimiter := "|---|---|---|---|---|---|---|"
	rows := make([][]string, maxTableBodyRows)
	lines := make([]string, 0, maxTableBodyRows+2)
	lines = append(lines, header, delimiter)
	for i := range rows {
		rows[i] = make([]string, columns)
		lines = append(lines, strings.Repeat("|", columns+1))
	}
	base := strings.Join(lines, "\n")
	if len(base) > targetBytes {
		t.Fatalf("empty exact-limit fixture is already %d bytes", len(base))
	}
	remaining := targetBytes - len(base)
	for row := range rows {
		for column := range rows[row] {
			if remaining == 0 {
				break
			}
			n := remaining
			if n > maxTableCellTextBytes {
				n = maxTableCellTextBytes
			}
			rows[row][column] = strings.Repeat("x", n)
			remaining -= n
		}
		if remaining == 0 {
			break
		}
	}
	if remaining != 0 {
		t.Fatalf("exact-limit fixture has %d unfilled bytes", remaining)
	}
	lines = lines[:2]
	for _, cells := range rows {
		lines = append(lines, "|"+strings.Join(cells, "|")+"|")
	}
	source := strings.Join(lines, "\n")
	if len(source) != targetBytes {
		t.Fatalf("exact-limit fixture bytes=%d want=%d", len(source), targetBytes)
	}
	return source
}

func applyColumnTestPatch(t *testing.T, source string, patch TableColumnPatch) string {
	t.Helper()
	candidate := source
	for i := len(patch.Edits) - 1; i >= 0; i-- {
		edit := patch.Edits[i]
		if edit.Range.Start < 0 || edit.Range.End < edit.Range.Start || edit.Range.End > len(candidate) {
			t.Fatalf("invalid descending edit %+v for length %d", edit, len(candidate))
		}
		candidate = candidate[:edit.Range.Start] + edit.Text + candidate[edit.Range.End:]
	}
	return candidate
}

func columnDescriptorTables(t *testing.T, targetColumns, fullTables, lastBodyRows int) string {
	t.Helper()
	rowFor := func(columns int) string {
		var row strings.Builder
		row.WriteByte('|')
		for i := 0; i < columns; i++ {
			row.WriteString("a|")
		}
		row.WriteByte('\n')
		return row.String()
	}
	delimiterFor := func(columns int) string {
		var delimiter strings.Builder
		delimiter.WriteByte('|')
		for i := 0; i < columns; i++ {
			delimiter.WriteString("---|")
		}
		delimiter.WriteByte('\n')
		return delimiter.String()
	}
	var source strings.Builder
	source.WriteString(rowFor(targetColumns))
	source.WriteString(delimiterFor(targetColumns))
	for body := 0; body < maxTableBodyRows; body++ {
		source.WriteString(rowFor(targetColumns))
	}
	source.WriteByte('\n')
	for i := 0; i < fullTables; i++ {
		source.WriteString(rowFor(8))
		source.WriteString(delimiterFor(8))
		for body := 0; body < maxTableBodyRows; body++ {
			source.WriteString(rowFor(8))
		}
		source.WriteByte('\n')
	}
	source.WriteString(rowFor(8))
	source.WriteString(delimiterFor(8))
	for body := 0; body < lastBodyRows; body++ {
		source.WriteString(rowFor(8))
	}
	_ = t
	return source.String()
}
