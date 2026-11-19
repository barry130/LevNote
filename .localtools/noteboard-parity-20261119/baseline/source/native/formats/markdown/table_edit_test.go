package markdown

import (
	"errors"
	"strings"
	"testing"
)

func rowEditTable(t *testing.T, source string, number int) Block {
	t.Helper()
	doc := Parse(source)
	if doc.Partial {
		t.Fatal("fixture must be complete")
	}
	for _, block := range doc.Blocks {
		if block.Kind == BlockTable {
			if number == 0 {
				if block.Table.Problem != "" {
					t.Fatalf("fixture table is fallback: %s", block.Table.Problem)
				}
				return block
			}
			number--
		}
	}
	t.Fatal("fixture table missing")
	return Block{}
}

func TestTableRowPlanExactBytes(t *testing.T) {
	for _, test := range []struct {
		name, source string
		row          int
		action       TableRowAction
		patchStart   int
		old, added   string
		want         string
	}{
		{"header-only", "| H | B |\n|---|---|\n", 0, TableRowInsertAfter, len("| H | B |\n|---|---|\n"), "", "| | |\n", "| H | B |\n|---|---|\n| | |\n"},
		{"CRLF-unicode-escaped-padding", "\ufeffintro\r\n\r\n  H🙂 | B\r\n  :--- | ---:\r\n\t甲\\|乙 | **literal** \r\n\r\nafter\r\n", 1, TableRowInsertAfter, len("\ufeffintro\r\n\r\n  H🙂 | B\r\n  :--- | ---:\r\n\t甲\\|乙 | **literal** \r\n"), "", "\t| | |\r\n", "\ufeffintro\r\n\r\n  H🙂 | B\r\n  :--- | ---:\r\n\t甲\\|乙 | **literal** \r\n\t| | |\r\n\r\nafter\r\n"},
		{"header-indent-delimiter-ending", "  | H |\n\t|---|\r\n| a |\r", 0, TableRowInsertAfter, len("  | H |\n\t|---|\r\n"), "", "  | |\r\n", "  | H |\n\t|---|\r\n  | |\r\n| a |\r"},
		{"CR", "| H |\r|---|\r| a |\r", 1, TableRowInsertAfter, len("| H |\r|---|\r| a |\r"), "", "| |\r", "| H |\r|---|\r| a |\r| |\r"},
		{"unterminated-body-nearest-ending", "| H |\n|---|\r\n| a |\r| last |", 2, TableRowInsertAfter, len("| H |\n|---|\r\n| a |\r| last |"), "", "\r| |", "| H |\n|---|\r\n| a |\r| last |\r| |"},
		{"unterminated-delimiter", "| H |\r\n|---|", 0, TableRowInsertAfter, len("| H |\r\n|---|"), "", "\r\n| |", "| H |\r\n|---|\r\n| |"},
		{"middle-delete-own-ending", "prefix\n\n| H |\n|---|\n| a |\r\n| b |\r| c |\n\npost", 2, TableRowDelete, len("prefix\n\n| H |\n|---|\n| a |\r\n"), "| b |\r", "", "prefix\n\n| H |\n|---|\n| a |\r\n| c |\n\npost"},
		{"delete-final-preserve-previous-ending", "| H |\n|---|\r\n| a |", 1, TableRowDelete, len("| H |\n|---|\r\n"), "| a |", "", "| H |\n|---|\r\n"},
		{"delete-only-body", "| H |\n|---|\n| a |\n\nnext", 1, TableRowDelete, len("| H |\n|---|\n"), "| a |\n", "", "| H |\n|---|\n\nnext"},
		{"one-column-single-pipe", "|\n|---|\n|\n", 1, TableRowInsertAfter, len("|\n|---|\n|\n"), "", "| |\n", "|\n|---|\n|\n| |\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			table := rowEditTable(t, test.source, 0)
			patch, err := PlanTableRowEdit(test.source, table.Range, table.Table.Rows[test.row].Range, test.action)
			if err != nil {
				t.Fatal(err)
			}
			if patch.Range != (SourceRange{test.patchStart, test.patchStart + len(test.old)}) || patch.Text != test.added {
				t.Fatalf("patch=%+v text=%q", patch.Range, patch.Text)
			}
			got := test.source[:patch.Range.Start] + patch.Text + test.source[patch.Range.End:]
			if got != test.want || test.source[patch.Range.Start:patch.Range.End] != test.old {
				t.Fatalf("exact source\ngot %q\nwant%q", got, test.want)
			}
		})
	}
}

func TestTableRowPlanRefusesForgedAndUnsupportedTargets(t *testing.T) {
	source := "pre🙂\n\n| H |\n|---|\n| a |\n"
	table := rowEditTable(t, source, 0)
	body := table.Table.Rows[1].Range
	for _, test := range []struct {
		name, source string
		table, row   SourceRange
		action       TableRowAction
		want         error
	}{
		{"unknown-action", source, table.Range, body, 0, ErrTableRowAction},
		{"header-delete", source, table.Range, table.Table.Rows[0].Range, TableRowDelete, ErrTableRowHeader},
		{"truncated-table", source, rangeAt(table.Range.Start, table.Range.End-1), body, TableRowDelete, ErrTableRowTarget},
		{"row-without-ending", source, table.Range, rangeAt(body.Start, body.End-1), TableRowDelete, ErrTableRowTarget},
		{"delimiter-not-row", source, table.Range, table.Table.DelimiterRange, TableRowInsertAfter, ErrTableRowTarget},
		{"negative", source, rangeAt(-1, table.Range.End), body, TableRowDelete, ErrTableRowTarget},
		{"utf8", "\xff" + source[1:], table.Range, body, TableRowDelete, ErrTableRowTarget},
		{"malformed", strings.Replace(source, "|---|", "|-- |", 1), table.Range, body, TableRowDelete, ErrTableRowTarget},
		{"fenced", "```\n" + source + "```\n", shiftTableRange(table.Range, 4), shiftTableRange(body, 4), TableRowDelete, ErrTableRowTarget},
	} {
		t.Run(test.name, func(t *testing.T) {
			patch, err := PlanTableRowEdit(test.source, test.table, test.row, test.action)
			if !errors.Is(err, test.want) || patch != (TableRowPatch{}) {
				t.Fatalf("patch=%+v err=%v want=%v", patch, err, test.want)
			}
		})
	}
}

func TestTableRowPlanEightColumnsAndBodyLimit(t *testing.T) {
	const header = "|a|b|c|d|e|f|g|h|\n|---|---|---|---|---|---|---|---|\n"
	const body = "|1|2|3|4|5|6|7|8|\n"
	source := header + strings.Repeat(body, 255)
	table := rowEditTable(t, source, 0)
	patch, err := PlanTableRowEdit(source, table.Range, table.Table.Rows[255].Range, TableRowInsertAfter)
	if err != nil || patch.Range != (SourceRange{len(source), len(source)}) || patch.Text != "| | | | | | | | |\n" {
		t.Fatalf("255->256 patch=%+v err=%v", patch, err)
	}
	source += patch.Text
	table = rowEditTable(t, source, 0)
	patch, err = PlanTableRowEdit(source, table.Range, table.Table.Rows[0].Range, TableRowInsertAfter)
	if !errors.Is(err, ErrTableRowLimit) || patch != (TableRowPatch{}) {
		t.Fatalf("257 rows allowed: %+v %v", patch, err)
	}
	patch, err = PlanTableRowEdit(source, table.Range, table.Table.Rows[256].Range, TableRowDelete)
	if err != nil || patch.Text != "" || source[patch.Range.Start:patch.Range.End] != "| | | | | | | | |\n" {
		t.Fatalf("limit deletion failed: %+v %v", patch, err)
	}
}

func TestTableRowPlanLocalAndDocumentLimits(t *testing.T) {
	t.Run("table-bytes", func(t *testing.T) {
		prefix := "| H |\n|---|\n|"
		source := prefix + strings.Repeat(" ", (32<<10)-len(prefix)-2) + "|\n"
		table := rowEditTable(t, source, 0)
		patch, err := PlanTableRowEdit(source, table.Range, table.Table.Rows[0].Range, TableRowInsertAfter)
		if !errors.Is(err, ErrTableRowLimit) || patch != (TableRowPatch{}) {
			t.Fatalf("32KiB insertion=%+v %v", patch, err)
		}
	})
	for _, bad := range []string{
		"| H |\n|---|\n|" + strings.Repeat("x", 4097) + "|\n",
		"|a|b|c|d|e|f|g|h|i|\n|---|---|---|---|---|---|---|---|---|\n",
	} {
		patch, err := PlanTableRowEdit(bad, rangeAt(0, len(bad)), rangeAt(0, strings.IndexByte(bad, '\n')+1), TableRowInsertAfter)
		if !errors.Is(err, ErrTableRowTarget) || patch != (TableRowPatch{}) {
			t.Fatalf("unsupported source accepted: %+v %v", patch, err)
		}
	}
	t.Run("full-document-bytes", func(t *testing.T) {
		const table = "| H |\n|---|\n"
		const fence = "\n```\n"
		const end = "\n```\n"
		source := table + fence + strings.Repeat("x", MaxParseBytes-len(table+fence+end)) + end
		patch, err := PlanTableRowEdit(source, rangeAt(0, len(table)), rangeAt(0, 6), TableRowInsertAfter)
		if !errors.Is(err, ErrTableRowLimit) || patch != (TableRowPatch{}) {
			t.Fatalf("8MiB insertion=%+v %v", patch, err)
		}
	})
	t.Run("candidate-line-limit", func(t *testing.T) {
		const table = "| H |\n|---|\n"
		source := table + strings.Repeat("\n", maxLines-2)
		if Parse(source).Partial {
			t.Fatal("exact line-limit fixture incomplete")
		}
		patch, err := PlanTableRowEdit(source, rangeAt(0, len(table)), rangeAt(0, 6), TableRowInsertAfter)
		if !errors.Is(err, ErrTableRowLimit) || patch != (TableRowPatch{}) {
			t.Fatalf("candidate line-limit=%+v %v", patch, err)
		}
	})
	t.Run("already-partial", func(t *testing.T) {
		const table = "| H |\n|---|\n"
		source := table + strings.Repeat("# h\n", maxBlocks)
		patch, err := PlanTableRowEdit(source, rangeAt(0, len(table)), rangeAt(0, 6), TableRowInsertAfter)
		if !errors.Is(err, ErrTableRowLimit) || patch != (TableRowPatch{}) {
			t.Fatalf("partial source=%+v %v", patch, err)
		}
	})
}

func TestTableRowPlanProtectsLaterDescriptorBudget(t *testing.T) {
	const header = "|a|b|c|d|e|f|g|h|\n|---|---|---|---|---|---|---|---|\n"
	const body = "|1|2|3|4|5|6|7|8|\n"
	oneTable := header + strings.Repeat(body, 255) //256*8cells
	source := strings.Repeat(oneTable+"\n", 8)     //16384cells, all tables valid
	doc := Parse(source)
	if doc.Partial || len(doc.Blocks) != 8 {
		t.Fatal("budget fixture incomplete")
	}
	for _, block := range doc.Blocks {
		if block.Table.Problem != "" || len(block.Table.Rows) != 256 {
			t.Fatal("budget fixture already fallback")
		}
	}
	first := doc.Blocks[0]
	point := first.Table.DelimiterRange.End
	naive := Parse(source[:point] + "| | | | | | | | |\n" + source[point:])
	if naive.Partial || naive.Blocks[7].Table.Problem == "" || naive.Blocks[0].Table.Problem != "" {
		t.Fatal("oracle must demonstrate later fallback without Partial")
	}
	patch, err := PlanTableRowEdit(source, first.Range, first.Table.Rows[0].Range, TableRowInsertAfter)
	if !errors.Is(err, ErrTableRowContext) || patch != (TableRowPatch{}) {
		t.Fatalf("later-table silent fallback accepted: %+v %v", patch, err)
	}
	patch, err = PlanTableRowEdit(source, first.Range, first.Table.Rows[1].Range, TableRowDelete)
	if err != nil || patch.Text != "" || source[patch.Range.Start:patch.Range.End] != body {
		t.Fatalf("budget deletion=%+v %v", patch, err)
	}
	// Existing malformed tables retain their exact raw fallback when a nearby
	// legal table changes. A budget fallback becoming valid is also a context
	// change and is deliberately refused by this precise local command.
	malformed := "\n| A |\n|--|\n| B |\n"
	short := "| H |\n|---|\n| x |\n"
	table := rowEditTable(t, short+malformed, 0)
	if _, err := PlanTableRowEdit(short+malformed, table.Range, table.Table.Rows[1].Range, TableRowDelete); err != nil {
		t.Fatal("unrelated malformed fallback changed:", err)
	}
	over := source + "\n| H |\n|---|\n"
	if Parse(over).Blocks[8].Table.Problem == "" {
		t.Fatal("expected initial budget fallback")
	}
	if patch, err := PlanTableRowEdit(over, first.Range, first.Table.Rows[1].Range, TableRowDelete); !errors.Is(err, ErrTableRowContext) || patch != (TableRowPatch{}) {
		t.Fatalf("unrelated fallback promotion accepted: %+v %v", patch, err)
	}
}
