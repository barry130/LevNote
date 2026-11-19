package markdown

import (
	"errors"
	"strings"
	"testing"
)

func tableAt(t *testing.T, doc Document, index int) Block {
	t.Helper()
	if index >= len(doc.Blocks) {
		t.Fatalf("block %d missing: %#v", index, doc.Blocks)
	}
	block := doc.Blocks[index]
	if block.Kind != BlockTable {
		t.Fatalf("block %d kind = %s", index, block.Kind)
	}
	return block
}

func TestTableExactRangesMixedPipesCRLFAndUnicode(t *testing.T) {
	source := "\r\n  | H🙂 | 普通 |\r\n  |:---|---:|\r\n  | a | 中文 |\r\n\r\n"
	doc := Parse(source)
	block := tableAt(t, doc, 0)
	if block.Table.Problem != "" || len(block.Table.Rows) != 2 ||
		len(block.Table.Alignments) != 2 ||
		block.Table.Alignments[0] != TableAlignLeft ||
		block.Table.Alignments[1] != TableAlignRight {
		t.Fatalf("table descriptor = %#v", block.Table)
	}
	bodyLine := "  | a | 中文 |\r\n"
	tableEnd := sourceIndex(source, bodyLine) + len(bodyLine)
	if block.Range != rangeAt(2, tableEnd) || block.Raw != source[2:tableEnd] {
		t.Fatalf("table range = %#v raw=%q", block.Range, block.Raw)
	}
	delimiterStart := sourceIndex(source, "  |:---|---:|")
	bodyStart := sourceIndex(source, bodyLine)
	if block.Table.DelimiterRange != rangeAt(delimiterStart, bodyStart) {
		t.Fatalf("delimiter range = %#v", block.Table.DelimiterRange)
	}
	header := block.Table.Rows[0]
	if header.Range.End-header.Range.Start != len("  | H🙂 | 普通 |\r\n") {
		t.Fatalf("header row range = %#v", header.Range)
	}
	got := [][2]string{}
	for _, cell := range header.Cells {
		got = append(got, [2]string{
			source[cell.Range.Start:cell.Range.End],
			source[cell.TextRange.Start:cell.TextRange.End],
		})
	}
	if got[0] != [2]string{" H🙂 ", "H🙂"} ||
		got[1] != [2]string{" 普通 ", "普通"} {
		t.Fatalf("header cells = %#v", got)
	}
	if doc.Raw() != source || doc.Partial || len(doc.Blocks) != 1 {
		t.Fatalf("document = %#v", doc)
	}
}

func TestTableOneColumnEmptyRowsAndEscapedOnlyProse(t *testing.T) {
	source := "| |\n|---|\n||\na \\| b \\| c\n"
	doc := Parse(source)
	if len(doc.Blocks) != 2 || doc.Blocks[0].Kind != BlockTable ||
		doc.Blocks[1].Kind != BlockParagraph {
		t.Fatalf("blocks = %#v", doc.Blocks)
	}
	table := doc.Blocks[0].Table
	if table.Problem != "" || len(table.Rows) != 2 ||
		len(table.Rows[0].Cells) != 1 || len(table.Rows[1].Cells) != 1 {
		t.Fatalf("one-column table = %#v", table)
	}
	for rowIndex := 0; rowIndex < 2; rowIndex++ {
		cell := table.Rows[rowIndex].Cells[0]
		if cell.TextRange.Start != cell.TextRange.End {
			t.Fatalf("row %d empty text range = %#v", rowIndex, cell.TextRange)
		}
	}
	if doc.Blocks[1].Text != "a \\| b \\| c" {
		t.Fatalf("escaped-only row = %q", doc.Blocks[1].Text)
	}
}

func TestTableNoOuterParagraphInterruptionAndBlockBoundaries(t *testing.T) {
	source := "intro\nA | B\n--- | ---\n| one | two |\n# Heading\n| x |\n"
	doc := Parse(source)
	if len(doc.Blocks) != 4 || doc.Blocks[0].Kind != BlockParagraph ||
		doc.Blocks[1].Kind != BlockTable ||
		doc.Blocks[2].Kind != BlockHeading ||
		doc.Blocks[3].Kind != BlockParagraph {
		t.Fatalf("blocks = %#v", doc.Blocks)
	}
	table := doc.Blocks[1].Table
	if table.Problem != "" || len(table.Rows) != 2 ||
		len(table.Rows[0].Cells) != 2 ||
		doc.Raw()[table.Rows[0].Cells[0].TextRange.Start:table.Rows[0].Cells[0].TextRange.End] != "A" {
		t.Fatalf("table = %#v", table)
	}
	if len(doc.Warnings) != 1 || doc.Warnings[0].Code != "unsupported-table" ||
		doc.Warnings[0].Range != doc.Blocks[3].Range {
		t.Fatalf("legacy pipe-run warning = %#v", doc.Warnings)
	}

	boundaries := Parse("| h |\n|---|\n# heading\n- item\n> quote\n```go\nx\n```\n---\n")
	if len(boundaries.Blocks) != 6 ||
		boundaries.Blocks[0].Kind != BlockTable ||
		boundaries.Blocks[1].Kind != BlockHeading ||
		boundaries.Blocks[2].Kind != BlockList ||
		boundaries.Blocks[3].Kind != BlockQuote ||
		boundaries.Blocks[4].Kind != BlockCode ||
		boundaries.Blocks[5].Kind != BlockDivider {
		t.Fatalf("boundary blocks = %#v", boundaries.Blocks)
	}
	if boundaries.Blocks[0].Raw != "| h |\n|---|\n" {
		t.Fatalf("boundary table raw = %q", boundaries.Blocks[0].Raw)
	}
}

func TestTableFailedCandidatesRemainOneCompleteRawBlock(t *testing.T) {
	cases := []struct {
		name   string
		source string
		reason string
	}{
		{
			name:   "body-column-mismatch",
			source: "| a | b |\n|---|---|\n| c | d | e |\n| later | two |\n\nprose\n",
			reason: tableBodyCountMessage,
		},
		{
			name:   "short-delimiter-dash",
			source: "| a | b |\n|--|---|\n| c | d |\n\nprose\n",
			reason: tableDelimiterMessage,
		},
		{
			name:   "too-indented-header",
			source: "    | a | b |\n|---|---|\n| c | d |\n",
			reason: tableIndentMessage,
		},
		{
			name:   "too-indented-body",
			source: "| a | b |\n|---|---|\n    | c | d |\n",
			reason: tableIndentMessage,
		},
		{
			name:   "nine-columns",
			source: "|a|b|c|d|e|f|g|h|i|\n|---|---|---|---|---|---|---|---|---|\n",
			reason: tableColumnMessage,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := Parse(tc.source)
			block := tableAt(t, doc, 0)
			wantRaw := tc.source
			if proseStart := strings.Index(tc.source, "\n\nprose\n"); proseStart >= 0 {
				wantRaw = tc.source[:proseStart+1]
			}
			if block.Table.Problem != tc.reason ||
				block.Table.Rows != nil || block.Table.Alignments != nil ||
				block.Raw != wantRaw {
				t.Fatalf("fallback = %#v raw=%q", block.Table, block.Raw)
			}
			if len(doc.Warnings) != 1 ||
				doc.Warnings[0].Code != "unsupported-table" ||
				doc.Warnings[0].Message != tc.reason {
				t.Fatalf("warnings = %#v", doc.Warnings)
			}
			if strings.Contains(tc.source, "\n\nprose\n") {
				if len(doc.Blocks) != 2 || doc.Blocks[1].Text != "prose" {
					t.Fatalf("post-fallback blocks = %#v", doc.Blocks)
				}
			}
		})
	}
}

func TestTableBackslashParityAndBackticksAreLiteral(t *testing.T) {
	valid := Parse("| a \\| b | `c\\|d` |\n|---|---|\n| e | f |\n")
	table := tableAt(t, valid, 0).Table
	if table.Problem != "" || len(table.Rows) != 2 ||
		len(table.Rows[0].Cells) != 2 {
		t.Fatalf("escaped-pipe table = %#v", table)
	}
	first := table.Rows[0].Cells[0]
	if got := valid.Raw()[first.TextRange.Start:first.TextRange.End]; got != `a \| b` {
		t.Fatalf("odd-backslash cell = %q", got)
	}
	second := table.Rows[0].Cells[1]
	if got := valid.Raw()[second.TextRange.Start:second.TextRange.End]; got != "`c\\|d`" {
		t.Fatalf("backtick escaped cell = %q", got)
	}

	backtick := Parse("| a | `b|c` |\n|---|---|\n")
	if backtick.Blocks[0].Kind != BlockTable ||
		backtick.Blocks[0].Table.Problem == "" {
		t.Fatalf("unescaped backtick pipe did not fall back: %#v", backtick.Blocks)
	}

	even := Parse("| a \\\\| b | c |\n|---|---|---|\n")
	evenTable := tableAt(t, even, 0).Table
	if evenTable.Problem != "" || len(evenTable.Rows[0].Cells) != 3 {
		t.Fatalf("even-backslash pipe should be structural = %#v", evenTable)
	}
}

func TestTableCapsInclusiveAndCapPlusOne(t *testing.T) {
	t.Run("columns", func(t *testing.T) {
		header8 := "|a|b|c|d|e|f|g|h|\n"
		delimiter8 := "|---|---|---|---|---|---|---|---|\n"
		doc := Parse(header8 + delimiter8)
		if table := tableAt(t, doc, 0).Table; table.Problem != "" ||
			len(table.Rows[0].Cells) != 8 {
			t.Fatalf("eight columns = %#v", table)
		}
		nine := "|a|b|c|d|e|f|g|h|i|\n"
		delimiter9 := "|---|---|---|---|---|---|---|---|---|\n"
		doc = Parse(nine + delimiter9)
		if table := tableAt(t, doc, 0).Table; table.Problem != tableColumnMessage {
			t.Fatalf("nine columns = %#v", table)
		}
	})

	t.Run("body-rows", func(t *testing.T) {
		makeRows := func(body int) string {
			var out strings.Builder
			out.WriteString("| a | b |\n|---|---|\n")
			for i := 0; i < body; i++ {
				out.WriteString("| x | y |\n")
			}
			return out.String()
		}
		doc := Parse(makeRows(256))
		table := tableAt(t, doc, 0).Table
		if table.Problem != "" || len(table.Rows) != 257 {
			t.Fatalf("256 body rows = %#v", table)
		}
		doc = Parse(makeRows(257))
		block := tableAt(t, doc, 0)
		table = block.Table
		if table.Problem != tableRowsMessage || table.Rows != nil ||
			block.Raw != makeRows(257) {
			t.Fatalf("257 body rows = %#v", table)
		}
	})

	t.Run("cell-text", func(t *testing.T) {
		makeCell := func(size int) string {
			return "| value | other |\n|---|---|\n| " +
				strings.Repeat("a", size) + " | x |\n"
		}
		doc := Parse(makeCell(4096))
		if table := tableAt(t, doc, 0).Table; table.Problem != "" {
			t.Fatalf("4096-byte cell = %#v", table)
		}
		doc = Parse(makeCell(4097))
		if table := tableAt(t, doc, 0).Table; table.Problem != tableCellMessage {
			t.Fatalf("4097-byte cell = %#v", table)
		}
	})

	t.Run("raw-table", func(t *testing.T) {
		header := "|a|b|c|d|e|f|g|h|\n"
		delimiter := "|---|---|---|---|---|---|---|---|\n"
		parts := []string{
			strings.Repeat("a", 4095),
			strings.Repeat("b", 4095),
			strings.Repeat("c", 4095),
			strings.Repeat("d", 4095),
			strings.Repeat("e", 4095),
			strings.Repeat("f", 4095),
			strings.Repeat("g", 4095),
			strings.Repeat("h", 4041),
		}
		body := "|" + strings.Join(parts, "|") + "|\n"
		exact := header + delimiter + body
		if len(exact) != maxTableRawBytes {
			t.Fatalf("raw fixture = %d, want %d", len(exact), maxTableRawBytes)
		}
		doc := Parse(exact)
		if table := tableAt(t, doc, 0).Table; table.Problem != "" {
			t.Fatalf("exact raw cap = %#v", table)
		}
		over := header + delimiter + "|" + strings.Join(parts[:7], "|") +
			"|" + parts[7] + "x|\n"
		doc = Parse(over)
		block := tableAt(t, doc, 0)
		if table := block.Table; table.Problem != tableRawMessage ||
			table.Rows != nil || block.Raw != over {
			t.Fatalf("raw cap plus one = %#v", table)
		}
	})
}

func TestTableAggregateDescriptorBudget(t *testing.T) {
	table := "| a|b|c|d|e|f|g|h|\n|---|---|---|---|---|---|---|---|\n\n"
	var source strings.Builder
	for i := 0; i < 2048; i++ {
		source.WriteString(table)
	}
	source.WriteString(table)
	doc := Parse(source.String())
	valid, fallback := 0, 0
	for _, block := range doc.Blocks {
		if block.Kind != BlockTable {
			t.Fatalf("unexpected block kind = %s", block.Kind)
		}
		if block.Table.Problem == "" {
			valid++
		} else {
			fallback++
		}
	}
	if valid != 2048 || fallback != 1 || len(doc.Blocks) != 2049 {
		t.Fatalf("aggregate tables valid=%d fallback=%d blocks=%d", valid, fallback, len(doc.Blocks))
	}
	if doc.Partial {
		t.Fatal("aggregate table fallback marked document partial")
	}
}

func TestTableLiteralHTMLExportAndFallbacks(t *testing.T) {
	source := "| <b> & \"x\" | [link](https://x) | `code\\|` |\n" +
		"| :--- | ---: | --- |\n" +
		"| <img onerror=x> | a\\|b | c |\n"
	doc := Parse(source)
	htmlText, warnings, err := ExportHTMLChecked(doc)
	if err != nil || len(warnings) != 0 {
		t.Fatalf("export err=%v warnings=%#v", err, warnings)
	}
	for _, want := range []string{
		"<table>\n<thead>\n<tr>",
		`<th style="text-align:left">&lt;b&gt; &amp; &#34;x&#34;</th>`,
		`<th style="text-align:right">[link](https://x)</th>`,
		"`code\\|`",
		"&lt;img onerror=x&gt;",
		"a\\|b",
	} {
		if !strings.Contains(htmlText, want) {
			t.Fatalf("literal export missing %q:\n%s", want, htmlText)
		}
	}
	if strings.Contains(htmlText, "<a href") || strings.Contains(htmlText, "<img onerror") {
		t.Fatalf("literal cells activated markup:\n%s", htmlText)
	}

	zeroBody := Parse("| A | B |\n|---|---|\n")
	zeroHTML, zeroWarnings, err := ExportHTMLChecked(zeroBody)
	if err != nil || len(zeroWarnings) != 0 ||
		!strings.Contains(zeroHTML, "<thead>") ||
		strings.Contains(zeroHTML, "<tbody>") {
		t.Fatalf("zero-body export err=%v warnings=%#v\n%s", err, zeroWarnings, zeroHTML)
	}

	fallback := Parse("| A | B |\n|--|---|\n| <script>x</script> | [x](y) |\n")
	fallbackBlock := tableAt(t, fallback, 0)
	fallbackHTML, fallbackWarnings, err := ExportHTMLChecked(fallback)
	if err != nil || len(fallbackWarnings) != 1 {
		t.Fatalf("fallback export err=%v warnings=%#v", err, fallbackWarnings)
	}
	if !strings.Contains(fallbackHTML, "<pre>"+htmlEscape(fallbackBlock.Raw)+"</pre>") ||
		strings.Contains(fallbackHTML, "<table>") ||
		strings.Contains(fallbackHTML, "<script>") {
		t.Fatalf("fallback export:\n%s", fallbackHTML)
	}

	corruptSource := "prose\n| a |\n"
	corrupt := Document{
		source: corruptSource,
		Blocks: []Block{
			{
				Kind: BlockParagraph, Range: rangeAt(0, 6), TextRange: rangeAt(0, 6),
				Text: "prose",
			},
			{
				Kind: BlockTable, Range: rangeAt(6, 12), TextRange: rangeAt(6, 12),
				Raw: corruptSource[6:12],
				Table: TableRef{
					Rows:           []TableRowRef{{Cells: []TableCellRef{}}},
					DelimiterRange: rangeAt(6, 6),
				},
			},
		},
	}
	corruptHTML, corruptWarnings, err := ExportHTMLChecked(corrupt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(corruptHTML, "<p>prose</p>") ||
		!strings.Contains(corruptHTML, "<pre>"+htmlEscape(corruptSource[6:12])+"</pre>") ||
		len(corruptWarnings) != 1 ||
		corruptWarnings[0].Code != "export-table" {
		t.Fatalf("corrupt descriptor export:\n%s warnings=%#v", corruptHTML, corruptWarnings)
	}

	partial := Parse(strings.Repeat("a\n", maxLines+1))
	if _, _, err := ExportHTMLChecked(partial); !errors.Is(err, ErrPartialDocument) {
		t.Fatalf("partial export error = %v", err)
	}
}

func htmlEscape(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"'", "&#39;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&#34;",
	)
	return replacer.Replace(value)
}

func sourceIndex(source, needle string) int {
	index := strings.Index(source, needle)
	if index < 0 {
		panic("missing fixture needle")
	}
	return index
}
