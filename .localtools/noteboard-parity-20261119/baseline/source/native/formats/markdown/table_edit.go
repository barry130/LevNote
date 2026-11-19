package markdown

import (
	"errors"
	"slices"
	"strings"
	"unicode/utf8"
)

type TableRowAction uint8

const (
	TableRowInsertAfter TableRowAction = iota + 1
	TableRowDelete
)

// TableRowPatch addresses the supplied source in bytes. A failed plan always
// returns its zero value. Only the caller applies a successful patch.
type TableRowPatch struct {
	Range SourceRange
	Text  string
}

var (
	ErrTableRowTarget  = errors.New("table or physical row no longer matches supported source")
	ErrTableRowAction  = errors.New("unknown table row action")
	ErrTableRowHeader  = errors.New("the table header cannot be deleted")
	ErrTableRowLimit   = errors.New("table row edit exceeds a source or table limit")
	ErrTableRowContext = errors.New("table row edit changes another source structure")
)

// PlanTableRowEdit reparses authoritative source; caller-supplied ranges are
// identities to match, never trusted descriptors or a search by row text. It
// checks the complete candidate too: global table-descriptor exhaustion can
// turn a later table into fallback without setting Document.Partial.
func PlanTableRowEdit(source string, tableRange, rowRange SourceRange, action TableRowAction) (TableRowPatch, error) {
	if action != TableRowInsertAfter && action != TableRowDelete {
		return TableRowPatch{}, ErrTableRowAction
	}
	if len(source) > MaxParseBytes {
		return TableRowPatch{}, ErrTableRowLimit
	}
	if !utf8.ValidString(source) || tableRange.Start < 0 || tableRange.End > len(source) ||
		tableRange.Start >= tableRange.End || rowRange.Start < tableRange.Start ||
		rowRange.End > tableRange.End || rowRange.Start >= rowRange.End {
		return TableRowPatch{}, ErrTableRowTarget
	}
	if tableRange.End-tableRange.Start > maxTableRawBytes {
		return TableRowPatch{}, ErrTableRowLimit
	}
	doc := Parse(source)
	if doc.Partial {
		return TableRowPatch{}, ErrTableRowLimit
	}
	blockIndex, rowIndex := -1, -1
	for i, block := range doc.Blocks {
		if block.Kind != BlockTable || block.Range != tableRange || block.Table.Problem != "" {
			continue
		}
		blockIndex = i
		for row, descriptor := range block.Table.Rows {
			if descriptor.Range == rowRange {
				rowIndex = row
				break
			}
		}
		break
	}
	if blockIndex < 0 || rowIndex < 0 {
		return TableRowPatch{}, ErrTableRowTarget
	}
	before := doc.Blocks[blockIndex]
	patch := TableRowPatch{Range: rowRange}
	separatorBytes := 0
	if action == TableRowDelete {
		if rowIndex == 0 {
			return TableRowPatch{}, ErrTableRowHeader
		}
	} else {
		if len(before.Table.Rows)-1 >= maxTableBodyRows {
			return TableRowPatch{}, ErrTableRowLimit
		}
		anchor := rowRange
		if rowIndex == 0 {
			anchor = before.Table.DelimiterRange
		}
		row := source[rowRange.Start:rowRange.End]
		indent := 0
		for indent < len(row) && isTablePadding(row[indent]) {
			indent++
		}
		newRow := row[:indent] + "|" + strings.Repeat(" |", len(before.Table.Alignments))
		ending := tableRowEnding(source[anchor.Start:anchor.End])
		if ending == "" {
			// Only the final physical line can lack an ending. Use the nearest
			// preceding table ending, without changing existing line endings.
			for cursor := anchor.Start - 1; cursor >= tableRange.Start; cursor-- {
				if source[cursor] == '\n' {
					ending = "\n"
					if cursor > tableRange.Start && source[cursor-1] == '\r' {
						ending = "\r\n"
					}
					break
				}
				if source[cursor] == '\r' {
					ending = "\r"
					break
				}
			}
			if ending == "" {
				ending = "\n"
			}
			separatorBytes = len(ending)
			patch.Text = ending + newRow
		} else {
			patch.Text = newRow + ending
		}
		patch.Range = rangeAt(anchor.End, anchor.End)
	}
	delta := len(patch.Text) - (patch.Range.End - patch.Range.Start)
	if delta > maxTableRawBytes-len(before.Raw) || delta > MaxParseBytes-len(source) {
		return TableRowPatch{}, ErrTableRowLimit
	}
	candidate := source[:patch.Range.Start] + patch.Text + source[patch.Range.End:]
	afterDoc := Parse(candidate)
	if afterDoc.Partial {
		return TableRowPatch{}, ErrTableRowLimit
	}
	if len(afterDoc.Blocks) != len(doc.Blocks) {
		return TableRowPatch{}, ErrTableRowContext
	}
	for i, original := range doc.Blocks {
		next := afterDoc.Blocks[i]
		if i == blockIndex {
			if !tableRowShapePreserved(original, next, rowIndex, action, delta, separatorBytes, candidate) {
				return TableRowPatch{}, ErrTableRowLimit
			}
			continue
		}
		shift := 0
		if i > blockIndex {
			shift = delta
		}
		if original.Kind != next.Kind || original.Raw != next.Raw || original.Text != next.Text ||
			shiftTableRange(original.Range, shift) != next.Range ||
			!unchangedTable(original.Table, next.Table, shift) {
			return TableRowPatch{}, ErrTableRowContext
		}
	}
	return patch, nil
}

func tableRowEnding(line string) string {
	if strings.HasSuffix(line, "\r\n") {
		return "\r\n"
	}
	if strings.HasSuffix(line, "\n") {
		return "\n"
	}
	if strings.HasSuffix(line, "\r") {
		return "\r"
	}
	return ""
}

func shiftTableRange(r SourceRange, delta int) SourceRange {
	return rangeAt(r.Start+delta, r.End+delta)
}

func unchangedTable(old, next TableRef, shift int) bool {
	if old.Problem != next.Problem || !slices.Equal(old.Alignments, next.Alignments) || len(old.Rows) != len(next.Rows) {
		return false
	}
	// A fallback/ordinary block has no delimiter descriptor to translate.
	if len(old.Rows) == 0 {
		return old.DelimiterRange == next.DelimiterRange
	}
	if shiftTableRange(old.DelimiterRange, shift) != next.DelimiterRange {
		return false
	}
	for i := range old.Rows {
		if !unchangedTableRow(old.Rows[i], next.Rows[i], shift, 0) {
			return false
		}
	}
	return true
}

func unchangedTableRow(old, next TableRowRef, shift, addedEnding int) bool {
	wantRange := shiftTableRange(old.Range, shift)
	wantRange.End += addedEnding
	if wantRange != next.Range || len(old.Cells) != len(next.Cells) {
		return false
	}
	for i, cell := range old.Cells {
		if shiftTableRange(cell.Range, shift) != next.Cells[i].Range ||
			shiftTableRange(cell.TextRange, shift) != next.Cells[i].TextRange {
			return false
		}
	}
	return true
}

func tableRowShapePreserved(old, next Block, row int, action TableRowAction, delta, separatorBytes int, candidate string) bool {
	if next.Kind != BlockTable || next.Table.Problem != "" || next.Range.Start != old.Range.Start ||
		next.Range.End != old.Range.End+delta || !slices.Equal(old.Table.Alignments, next.Table.Alignments) {
		return false
	}
	wantCount := len(old.Table.Rows) + 1
	if action == TableRowDelete {
		wantCount -= 2
	}
	if len(next.Table.Rows) != wantCount {
		return false
	}
	delimiter := old.Table.DelimiterRange
	if row == 0 {
		delimiter.End += separatorBytes
	}
	if next.Table.DelimiterRange != delimiter {
		return false
	}
	for i, original := range old.Table.Rows {
		if action == TableRowDelete && i == row {
			continue
		}
		newIndex, shift, ending := i, 0, 0
		if i > row {
			shift = delta
			if action == TableRowInsertAfter {
				newIndex++
			} else {
				newIndex--
			}
		} else if i == row && row > 0 {
			ending = separatorBytes
		}
		if !unchangedTableRow(original, next.Table.Rows[newIndex], shift, ending) {
			return false
		}
	}
	if action == TableRowInsertAfter {
		inserted := next.Table.Rows[row+1]
		if len(inserted.Cells) != len(old.Table.Alignments) {
			return false
		}
		for _, cell := range inserted.Cells {
			if cell.TextRange.Start != cell.TextRange.End || candidate[cell.Range.Start:cell.Range.End] != " " {
				return false
			}
		}
	}
	return true
}
