package markdown

import "testing"

func TestTableMinimumLeadingOuterEOF(t *testing.T) {
	source := "| A | B |\n| --- | --- |"
	doc := Parse(source)
	if doc.Partial || len(doc.Blocks) != 1 || doc.Blocks[0].Kind != BlockTable {
		t.Fatalf("parse = %#v", doc)
	}
	block := doc.Blocks[0]
	if block.Table.Problem != "" || len(block.Table.Rows) != 1 ||
		len(block.Table.Alignments) != 2 || block.Raw != source {
		t.Fatalf("table = %#v raw=%q", block.Table, block.Raw)
	}
	header := block.Table.Rows[0]
	if got := source[header.Cells[0].Range.Start:header.Cells[0].Range.End]; got != " A " {
		t.Fatalf("first cell range = %q", got)
	}
	if got := source[header.Cells[0].TextRange.Start:header.Cells[0].TextRange.End]; got != "A" {
		t.Fatalf("first cell text = %q", got)
	}
	if got := source[header.Cells[1].Range.Start:header.Cells[1].Range.End]; got != " B " {
		t.Fatalf("second cell range = %q", got)
	}
	if got := source[header.Cells[1].TextRange.Start:header.Cells[1].TextRange.End]; got != "B" {
		t.Fatalf("second cell text = %q", got)
	}
	if block.Table.DelimiterRange.End != len(source) {
		t.Fatalf("delimiter range = %#v, source len = %d", block.Table.DelimiterRange, len(source))
	}
}

func TestTableMinimumNoOuter(t *testing.T) {
	source := "A | B\n--- | ---\none | two\n"
	doc := Parse(source)
	if doc.Partial || len(doc.Blocks) != 1 || doc.Blocks[0].Kind != BlockTable {
		t.Fatalf("parse = %#v", doc)
	}
	block := doc.Blocks[0]
	if block.Table.Problem != "" || len(block.Table.Rows) != 2 {
		t.Fatalf("table = %#v", block.Table)
	}
	for rowIndex, want := range [][2]string{{"A", "B"}, {"one", "two"}} {
		row := block.Table.Rows[rowIndex]
		if len(row.Cells) != 2 {
			t.Fatalf("row %d cells = %#v", rowIndex, row.Cells)
		}
		for cellIndex, wantText := range want {
			cell := row.Cells[cellIndex]
			if got := source[cell.TextRange.Start:cell.TextRange.End]; got != wantText {
				t.Fatalf("row %d cell %d text = %q", rowIndex, cellIndex, got)
			}
		}
	}
	if got := source[block.Table.Rows[0].Cells[0].Range.Start:block.Table.Rows[0].Cells[0].Range.End]; got != "A " {
		t.Fatalf("first cell range = %q", got)
	}
	if got := source[block.Table.Rows[0].Cells[1].Range.Start:block.Table.Rows[0].Cells[1].Range.End]; got != " B" {
		t.Fatalf("second cell range = %q", got)
	}
}

func TestTableMinimumOneColumnAndEmptyPipes(t *testing.T) {
	source := "| |\n|---|\n||\n| value |\n"
	doc := Parse(source)
	if len(doc.Blocks) != 1 || doc.Blocks[0].Kind != BlockTable {
		t.Fatalf("blocks = %#v", doc.Blocks)
	}
	table := doc.Blocks[0].Table
	if table.Problem != "" || len(table.Rows) != 3 {
		t.Fatalf("table = %#v", table)
	}
	for rowIndex := 0; rowIndex < 2; rowIndex++ {
		cell := table.Rows[rowIndex].Cells[0]
		if cell.TextRange.Start != cell.TextRange.End {
			t.Fatalf("row %d empty text range = %#v", rowIndex, cell.TextRange)
		}
	}
	value := table.Rows[2].Cells[0]
	if got := source[value.TextRange.Start:value.TextRange.End]; got != "value" {
		t.Fatalf("value text = %q", got)
	}

	for _, tc := range []struct {
		name   string
		source string
		cells  int
	}{
		{name: "bare-leading-pipe", source: "|\n|---|\n|", cells: 1},
		{name: "three-pipes", source: "|||\n|---|---|\n|||", cells: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed := Parse(tc.source)
			if len(parsed.Blocks) != 1 || parsed.Blocks[0].Kind != BlockTable {
				t.Fatalf("blocks = %#v", parsed.Blocks)
			}
			table := parsed.Blocks[0].Table
			if table.Problem != "" || len(table.Rows) != 2 || len(table.Rows[0].Cells) != tc.cells {
				t.Fatalf("table = %#v", table)
			}
		})
	}
}

func TestTableMinimumRejectsTrailingOnlyOneColumn(t *testing.T) {
	source := "a |\n--- |\n"
	doc := Parse(source)
	if len(doc.Blocks) != 1 || doc.Blocks[0].Kind != BlockTable {
		t.Fatalf("blocks = %#v", doc.Blocks)
	}
	if doc.Blocks[0].Table.Problem == "" ||
		doc.Blocks[0].Table.Problem != tableOneColumnMessage ||
		doc.Blocks[0].Raw != source ||
		len(doc.Warnings) != 1 {
		t.Fatalf("trailing-only one-column candidate = %#v warnings=%#v", doc.Blocks[0], doc.Warnings)
	}
}
