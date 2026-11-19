package markdown

import (
	"errors"
	"maps"
	"strings"
	"unicode/utf8"
)

// TableAlignmentPatch addresses source in bytes. Noop has zero Range and Text;
// errors return the zero value. Only the caller applies a successful patch.
type TableAlignmentPatch struct {
	Range SourceRange
	Text  string
	Noop  bool
}

var (
	ErrTableAlignmentValue   = errors.New("unknown table alignment")
	ErrTableAlignmentTarget  = errors.New("table or column no longer matches supported source")
	ErrTableAlignmentLimit   = errors.New("table alignment exceeds a source or table limit")
	ErrTableAlignmentContext = errors.New("table alignment changes another source structure")
)

// PlanTableAlignmentEdit changes only the non-padding delimiter marker of one
// column. It reparses authoritative source and the complete candidate, including
// other tables' global descriptor/fallback state. No descriptors are trusted
// from the caller, and an equal semantic alignment preserves every source byte.
func PlanTableAlignmentEdit(source string, tableRange SourceRange, column int, alignment TableAlignment) (TableAlignmentPatch, error) {
	if alignment > TableAlignRight {
		return TableAlignmentPatch{}, ErrTableAlignmentValue
	}
	if len(source) > MaxParseBytes {
		return TableAlignmentPatch{}, ErrTableAlignmentLimit
	}
	if !utf8.ValidString(source) || tableRange.Start < 0 || tableRange.End > len(source) ||
		tableRange.Start >= tableRange.End || column < 0 || column >= maxTableColumns {
		return TableAlignmentPatch{}, ErrTableAlignmentTarget
	}
	if tableRange.End-tableRange.Start > maxTableRawBytes {
		return TableAlignmentPatch{}, ErrTableAlignmentLimit
	}
	doc := Parse(source)
	if doc.Partial {
		return TableAlignmentPatch{}, ErrTableAlignmentLimit
	}
	index := -1
	for i, block := range doc.Blocks {
		if block.Kind == BlockTable && block.Range == tableRange && block.Table.Problem == "" {
			index = i
			break
		}
	}
	if index < 0 || column >= len(doc.Blocks[index].Table.Alignments) {
		return TableAlignmentPatch{}, ErrTableAlignmentTarget
	}
	before := doc.Blocks[index]
	delimiter := before.Table.DelimiterRange
	rawLine := source[delimiter.Start:delimiter.End]
	scan := scanTableRow(source, line{
		content: strings.TrimSuffix(rawLine, tableRowEnding(rawLine)),
		start:   delimiter.Start, end: delimiter.End,
	})
	alignments, valid := parseTableAlignments(source, scan.row)
	if !scan.valid || !valid || len(alignments) != len(before.Table.Alignments) {
		return TableAlignmentPatch{}, ErrTableAlignmentTarget
	}
	for i, value := range alignments {
		if value != before.Table.Alignments[i] {
			return TableAlignmentPatch{}, ErrTableAlignmentTarget
		}
	}
	if alignments[column] == alignment {
		return TableAlignmentPatch{Noop: true}, nil
	}
	markerRange := scan.row.Cells[column].TextRange
	marker := source[markerRange.Start:markerRange.End]
	hyphens := strings.Trim(marker, ":") // the existing parser validated this run
	left, right := "", ""
	if alignment == TableAlignLeft || alignment == TableAlignCenter {
		left = ":"
	}
	if alignment == TableAlignRight || alignment == TableAlignCenter {
		right = ":"
	}
	delta := len(hyphens) + len(left) + len(right) - len(marker)
	if delta > maxTableCellTextBytes-len(marker) || delta > maxTableRawBytes-len(before.Raw) ||
		delta > MaxParseBytes-len(source) {
		return TableAlignmentPatch{}, ErrTableAlignmentLimit
	}
	patch := TableAlignmentPatch{Range: markerRange, Text: left + hyphens + right}
	candidate := source[:markerRange.Start] + patch.Text + source[markerRange.End:]
	next := Parse(candidate)
	if next.Partial {
		return TableAlignmentPatch{}, ErrTableAlignmentLimit
	}
	if len(next.Blocks) != len(doc.Blocks) {
		return TableAlignmentPatch{}, ErrTableAlignmentContext
	}
	for i, block := range doc.Blocks {
		if i == index {
			if next.Blocks[i].Kind != BlockTable || next.Blocks[i].Table.Problem != "" {
				return TableAlignmentPatch{}, ErrTableAlignmentLimit
			}
			if !alignmentTablePreserved(source, candidate, block, next.Blocks[i], column, alignment, delta) {
				return TableAlignmentPatch{}, ErrTableAlignmentContext
			}
			continue
		}
		shift := 0
		if i > index {
			shift = delta
		}
		if !alignmentBlockPreserved(block, next.Blocks[i], shift) {
			return TableAlignmentPatch{}, ErrTableAlignmentContext
		}
	}
	if len(doc.Warnings) != len(next.Warnings) {
		return TableAlignmentPatch{}, ErrTableAlignmentContext
	}
	for i, warning := range doc.Warnings {
		shift := 0
		if warning.Range.Start >= markerRange.End {
			shift = delta
		}
		if warning.Code != next.Warnings[i].Code || warning.Message != next.Warnings[i].Message ||
			shiftTableRange(warning.Range, shift) != next.Warnings[i].Range {
			return TableAlignmentPatch{}, ErrTableAlignmentContext
		}
	}
	return patch, nil
}

func alignmentTablePreserved(source, candidate string, old, next Block, column int, alignment TableAlignment, delta int) bool {
	if next.Range != (SourceRange{old.Range.Start, old.Range.End + delta}) || next.TextRange != next.Range ||
		next.Table.DelimiterRange != (SourceRange{old.Table.DelimiterRange.Start, old.Table.DelimiterRange.End + delta}) ||
		len(old.Table.Rows) != len(next.Table.Rows) || len(old.Table.Alignments) != len(next.Table.Alignments) {
		return false
	}
	for i, value := range old.Table.Alignments {
		if i == column {
			value = alignment
		}
		if value != next.Table.Alignments[i] {
			return false
		}
	}
	for i, row := range old.Table.Rows {
		shift := 0
		if i > 0 {
			shift = delta
		}
		if !unchangedTableRow(row, next.Table.Rows[i], shift, 0) {
			return false
		}
		for j, cell := range row.Cells {
			nextCell := next.Table.Rows[i].Cells[j]
			if source[cell.TextRange.Start:cell.TextRange.End] != candidate[nextCell.TextRange.Start:nextCell.TextRange.End] {
				return false
			}
		}
	}
	return true
}

// Unaffected parsed structures must retain their meaning, not just raw bytes.
// Absent optional ranges stay zero; present ranges shift with their block.
func alignmentBlockPreserved(old, next Block, shift int) bool {
	optionalRange := func(r SourceRange) SourceRange {
		if r == (SourceRange{}) {
			return r
		}
		return shiftTableRange(r, shift)
	}
	if old.Kind != next.Kind || old.Raw != next.Raw || old.Text != next.Text ||
		old.Level != next.Level || old.Language != next.Language || old.Ordered != next.Ordered || old.Start != next.Start ||
		shiftTableRange(old.Range, shift) != next.Range || optionalRange(old.TextRange) != next.TextRange ||
		!alignmentInlinePreserved(old.Inline, next.Inline, shift) || len(old.Items) != len(next.Items) ||
		!unchangedTable(old.Table, next.Table, shift) {
		return false
	}
	for i, item := range old.Items {
		after := next.Items[i]
		if item.Text != after.Text || shiftTableRange(item.Range, shift) != after.Range ||
			shiftTableRange(item.TextRange, shift) != after.TextRange ||
			!alignmentInlinePreserved(item.Inline, after.Inline, shift) || (item.Task == nil) != (after.Task == nil) {
			return false
		}
		if item.Task != nil && (item.Task.Checked != after.Task.Checked || shiftTableRange(item.Task.Range, shift) != after.Task.Range) {
			return false
		}
	}
	if old.Kind == BlockDocBlock {
		a, b := old.DocBlock, next.DocBlock
		if a.Payload != b.Payload || a.FenceLength != b.FenceLength || a.ReferenceProblem != b.ReferenceProblem ||
			shiftTableRange(a.HeaderRange, shift) != b.HeaderRange || shiftTableRange(a.PayloadRange, shift) != b.PayloadRange ||
			a.Attrs.ID != b.Attrs.ID || a.Attrs.Kind != b.Attrs.Kind || a.Attrs.Mode != b.Attrs.Mode ||
			a.Attrs.Src != b.Attrs.Src || a.Attrs.Title != b.Attrs.Title || !maps.Equal(a.Attrs.Extra, b.Attrs.Extra) {
			return false
		}
	}
	return true
}

func alignmentInlinePreserved(old, next []InlineSpan, shift int) bool {
	if len(old) != len(next) {
		return false
	}
	for i, span := range old {
		if span.Kind != next[i].Kind || span.Text != next[i].Text || span.Href != next[i].Href ||
			shiftTableRange(span.Range, shift) != next[i].Range {
			return false
		}
	}
	return true
}
