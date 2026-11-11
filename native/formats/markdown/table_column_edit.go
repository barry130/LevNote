package markdown

import (
	"errors"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"
)

type TableColumnAction uint8

const (
	TableColumnInsertBefore TableColumnAction = iota + 1
	TableColumnInsertAfter
	TableColumnDelete
)

// TableColumnEdit replaces an original-source byte range with Text. Edits in a
// successful patch are nonoverlapping and ordered by their original offsets.
type TableColumnEdit struct {
	Range SourceRange
	Text  string
}

// TableColumnPatch is applied by the caller in descending edit order. A failed
// plan always returns its zero value.
type TableColumnPatch struct {
	Edits []TableColumnEdit
}

var (
	ErrTableColumnAction  = errors.New("unknown table column action")
	ErrTableColumnTarget  = errors.New("table, row, or column no longer matches supported source")
	ErrTableColumnLimit   = errors.New("table column edit exceeds a source or table limit")
	ErrTableColumnContext = errors.New("table column edit cannot preserve the complete source structure")
	ErrTableColumnMinimum = errors.New("the table must retain at least one column")
)

type tableColumnRow struct {
	ref               TableRowRef
	indent            int
	pipes             int
	leading, trailing bool
	degenerateOnePipe bool
}

// PlanTableColumnEdit inserts or deletes one complete column in a supported
// pipe table. Caller ranges are identities checked against a fresh parse; they
// are never used as trusted descriptors. column is zero based.
func PlanTableColumnEdit(
	source string,
	tableRange, anchorRowRange SourceRange,
	column int,
	action TableColumnAction,
) (TableColumnPatch, error) {
	if action != TableColumnInsertBefore && action != TableColumnInsertAfter && action != TableColumnDelete {
		return TableColumnPatch{}, ErrTableColumnAction
	}
	if len(source) > MaxParseBytes {
		return TableColumnPatch{}, ErrTableColumnLimit
	}
	if !utf8.ValidString(source) || !validColumnSourceRange(tableRange, len(source)) ||
		!validColumnSourceRange(anchorRowRange, len(source)) ||
		anchorRowRange.Start < tableRange.Start || anchorRowRange.End > tableRange.End {
		return TableColumnPatch{}, ErrTableColumnTarget
	}
	if tableRange.End-tableRange.Start > maxTableRawBytes {
		return TableColumnPatch{}, ErrTableColumnLimit
	}

	doc := Parse(source)
	if doc.Partial {
		return TableColumnPatch{}, ErrTableColumnLimit
	}
	blockIndex := -1
	for i, block := range doc.Blocks {
		if block.Kind == BlockTable && block.Range == tableRange {
			blockIndex = i
			break
		}
	}
	if blockIndex < 0 {
		return TableColumnPatch{}, ErrTableColumnTarget
	}
	before := doc.Blocks[blockIndex]
	if before.Table.Problem != "" {
		return TableColumnPatch{}, ErrTableColumnTarget
	}

	beforeRows, err := columnPhysicalRows(source, before)
	if err != nil {
		return TableColumnPatch{}, err
	}
	columns := len(before.Table.Alignments)
	if columns < 1 || columns > maxTableColumns || column < 0 || column >= columns {
		return TableColumnPatch{}, ErrTableColumnTarget
	}
	foundAnchor := false
	for _, row := range before.Table.Rows {
		if row.Range == anchorRowRange {
			foundAnchor = true
			break
		}
	}
	if !foundAnchor {
		return TableColumnPatch{}, ErrTableColumnTarget
	}
	if action == TableColumnDelete && columns == 1 {
		return TableColumnPatch{}, ErrTableColumnMinimum
	}
	if action != TableColumnDelete && columns == maxTableColumns {
		return TableColumnPatch{}, ErrTableColumnLimit
	}

	edits := make([]TableColumnEdit, 0, len(beforeRows))
	for i, row := range beforeRows {
		var rowEdits []TableColumnEdit
		if action == TableColumnDelete {
			rowEdits, err = planDeleteTableColumnCell(row, column)
		} else {
			rowEdits, err = planInsertTableColumnCellInSource(source, row, i == 1, column, action)
		}
		if err != nil {
			return TableColumnPatch{}, err
		}
		edits = append(edits, rowEdits...)
	}
	edits, err = mergeTableColumnEdits(edits)
	if err != nil {
		return TableColumnPatch{}, err
	}

	candidate, err := applyTableColumnEdits(source, edits)
	if err != nil {
		return TableColumnPatch{}, err
	}
	if len(candidate) > MaxParseBytes {
		return TableColumnPatch{}, ErrTableColumnLimit
	}
	candidateTableBytes := before.Range.End - before.Range.Start + len(candidate) - len(source)
	if candidateTableBytes > maxTableRawBytes {
		return TableColumnPatch{}, ErrTableColumnLimit
	}
	afterDoc := Parse(candidate)
	if afterDoc.Partial {
		return TableColumnPatch{}, ErrTableColumnLimit
	}
	if err := validateTableColumnCandidate(
		source, candidate, doc, afterDoc, blockIndex, beforeRows, column, action,
	); err != nil {
		return TableColumnPatch{}, err
	}
	return TableColumnPatch{Edits: edits}, nil
}

func validColumnSourceRange(r SourceRange, limit int) bool {
	return r.Start >= 0 && r.End > r.Start && r.End <= limit
}

func columnPhysicalRows(source string, block Block) ([]tableColumnRow, error) {
	if block.Kind != BlockTable || block.Table.Problem != "" || len(block.Table.Rows) == 0 {
		return nil, ErrTableColumnTarget
	}
	columns := len(block.Table.Alignments)
	if columns < 1 || columns > maxTableColumns {
		return nil, ErrTableColumnTarget
	}

	rows := make([]tableColumnRow, 0, len(block.Table.Rows)+1)
	header, err := scanColumnPhysicalRow(source, block.Table.Rows[0].Range, columns)
	if err != nil || !sameTableRowRef(header.ref, block.Table.Rows[0]) {
		return nil, ErrTableColumnTarget
	}
	rows = append(rows, header)
	delimiter, err := scanColumnPhysicalRow(source, block.Table.DelimiterRange, columns)
	if err != nil {
		return nil, err
	}
	rows = append(rows, delimiter)
	for i := 1; i < len(block.Table.Rows); i++ {
		row, rowErr := scanColumnPhysicalRow(source, block.Table.Rows[i].Range, columns)
		if rowErr != nil || !sameTableRowRef(row.ref, block.Table.Rows[i]) {
			return nil, ErrTableColumnTarget
		}
		rows = append(rows, row)
	}
	if rows[1].ref.Range != block.Table.DelimiterRange {
		return nil, ErrTableColumnTarget
	}
	return rows, nil
}

func scanColumnPhysicalRow(source string, rowRange SourceRange, columns int) (tableColumnRow, error) {
	if !validColumnSourceRange(rowRange, len(source)) || rowRange.End > len(source) {
		return tableColumnRow{}, ErrTableColumnTarget
	}
	raw := source[rowRange.Start:rowRange.End]
	content := strings.TrimSuffix(raw, tableRowEnding(raw))
	scan := scanTableRow(source, line{
		content: content,
		start:   rowRange.Start,
		end:     rowRange.End,
	})
	if !scan.valid || len(scan.row.Cells) != columns || !validRows(source, []TableRowRef{scan.row}) {
		return tableColumnRow{}, ErrTableColumnTarget
	}
	leading, trailing, pipes, err := inspectTableColumnPipes(source, scan)
	if err != nil {
		return tableColumnRow{}, err
	}
	return tableColumnRow{
		ref:               scan.row,
		indent:            scan.indent,
		pipes:             pipes,
		leading:           leading,
		trailing:          trailing,
		degenerateOnePipe: columns == 1 && pipes == 1 &&
			scan.row.Cells[0].Range.Start == scan.row.Cells[0].Range.End,
	}, nil
}

func inspectTableColumnPipes(source string, scan tableScan) (leading, trailing bool, pipes int, err error) {
	raw := source[scan.row.Range.Start:scan.row.Range.End]
	content := strings.TrimSuffix(raw, tableRowEnding(raw))
	first := scan.indent
	last := len(content)
	for last > first && isTablePadding(content[last-1]) {
		last--
	}
	if first >= last {
		return false, false, 0, ErrTableColumnTarget
	}
	firstPipe, lastPipe := -1, -1
	for i := first; i < last; i++ {
		if content[i] == '|' && structuralAt(source, scan.row.Range.Start+i) {
			if firstPipe < 0 {
				firstPipe = i
			}
			lastPipe = i
			pipes++
		}
	}
	if pipes == 0 || pipes != scan.pipes {
		return false, false, 0, ErrTableColumnTarget
	}
	return firstPipe == first, lastPipe == last-1 && last-1 != first, pipes, nil
}

func sameTableRowRef(a, b TableRowRef) bool {
	if a.Range != b.Range || len(a.Cells) != len(b.Cells) {
		return false
	}
	for i := range a.Cells {
		if a.Cells[i] != b.Cells[i] {
			return false
		}
	}
	return true
}

func planInsertTableColumnCell(
	row tableColumnRow,
	delimiter bool,
	column int,
	action TableColumnAction,
) (TableColumnEdit, error) {
	cell := row.ref.Cells[column]
	last := column == len(row.ref.Cells)-1
	position := cell.Range.Start
	text := "|"
	if action == TableColumnInsertAfter {
		position = cell.Range.End
	}

	if delimiter {
		switch {
		case action == TableColumnInsertBefore && column == 0 && !row.leading:
			text = "|---|"
		case action == TableColumnInsertBefore:
			text = "---|"
		case last && !row.trailing:
			text = "|---|"
		default:
			text = "|---"
		}
	} else {
		switch {
		case row.degenerateOnePipe:
			text = "||"
		case action == TableColumnInsertBefore && column == 0 && !row.leading:
			text = "||"
		case action == TableColumnInsertAfter && last && !row.trailing:
			text = "||"
		}
	}
	return TableColumnEdit{Range: rangeAt(position, position), Text: text}, nil
}

func planInsertTableColumnCellInSource(
	source string,
	row tableColumnRow,
	delimiter bool,
	column int,
	action TableColumnAction,
) ([]TableColumnEdit, error) {
	edit, err := planInsertTableColumnCell(row, delimiter, column, action)
	if err != nil {
		return nil, err
	}
	if edit.Text[0] == '|' && oddBackslashRunBefore(source, edit.Range.Start) {
		return nil, ErrTableColumnContext
	}
	cell := row.ref.Cells[column]
	if !delimiter && action == TableColumnInsertBefore && len(row.ref.Cells) == 1 &&
		columnInsertBeforeAddsTrailingBoundary(row, column) &&
		!row.degenerateOnePipe {
		// The sole surviving cell contains only padding. Without a right outer
		// pipe the newly inserted separator would be mistaken for the trailing
		// outer marker and the original padding would fall outside the table.
		return []TableColumnEdit{
			edit,
			{Range: rangeAt(cell.Range.End, cell.Range.End), Text: "|"},
		}, nil
	}
	return []TableColumnEdit{edit}, nil
}

func columnInsertBeforeAddsTrailingBoundary(row tableColumnRow, column int) bool {
	if column != 0 || len(row.ref.Cells) != 1 || !row.leading || row.trailing {
		return false
	}
	if row.degenerateOnePipe {
		return true
	}
	cell := row.ref.Cells[column]
	return cell.TextRange.Start == cell.TextRange.End && cell.Range.Start < cell.Range.End
}

func oddBackslashRunBefore(source string, position int) bool {
	if position < 0 || position > len(source) {
		return false
	}
	count := 0
	for i := position - 1; i >= 0 && source[i] == '\\'; i-- {
		count++
	}
	return count%2 != 0
}

func planDeleteTableColumnCell(row tableColumnRow, column int) ([]TableColumnEdit, error) {
	cell := row.ref.Cells[column]
	if column > 0 {
		previous := row.ref.Cells[column-1]
		if previous.Range.End >= cell.Range.End || cell.Range.End > row.ref.Range.End {
			return nil, ErrTableColumnTarget
		}
		if !row.leading && len(row.ref.Cells) == 2 {
			start := row.ref.Cells[0].Range.Start
			return []TableColumnEdit{
				{Range: rangeAt(start, start), Text: "|"},
				{Range: rangeAt(previous.Range.End, cell.Range.End)},
			}, nil
		}
		if columnDeleteLastAddsTrailingBoundary(row, column) {
			return []TableColumnEdit{
				{Range: rangeAt(previous.Range.End, cell.Range.End)},
				{Range: rangeAt(cell.Range.End, cell.Range.End), Text: "|"},
			}, nil
		}
		return []TableColumnEdit{{Range: rangeAt(previous.Range.End, cell.Range.End)}}, nil
	}

	end := cell.Range.End + 1
	if end > row.ref.Range.End || cell.Range.End >= row.ref.Range.End {
		return nil, ErrTableColumnTarget
	}
	text := ""
	if !row.leading {
		text = "|"
	}
	return []TableColumnEdit{{Range: rangeAt(cell.Range.Start, end), Text: text}}, nil
}

func columnDeleteLastAddsTrailingBoundary(row tableColumnRow, column int) bool {
	if column < 2 || column != len(row.ref.Cells)-1 || row.trailing {
		return false
	}
	previous := row.ref.Cells[column-1]
	return previous.TextRange.Start == previous.TextRange.End
}

func mergeTableColumnEdits(edits []TableColumnEdit) ([]TableColumnEdit, error) {
	ordered := slices.Clone(edits)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Range.Start != ordered[j].Range.Start {
			return ordered[i].Range.Start < ordered[j].Range.Start
		}
		return ordered[i].Range.End < ordered[j].Range.End
	})
	merged := make([]TableColumnEdit, 0, len(ordered))
	for _, edit := range ordered {
		if edit.Range.Start < 0 || edit.Range.End < edit.Range.Start || !utf8.ValidString(edit.Text) {
			return nil, ErrTableColumnContext
		}
		if len(merged) == 0 {
			merged = append(merged, edit)
			continue
		}
		last := &merged[len(merged)-1]
		if last.Range == edit.Range && last.Range.Start == last.Range.End {
			last.Text += edit.Text
			continue
		}
		if edit.Range.Start < last.Range.End {
			return nil, ErrTableColumnContext
		}
		merged = append(merged, edit)
	}
	return merged, nil
}

func applyTableColumnEdits(source string, edits []TableColumnEdit) (string, error) {
	var candidate strings.Builder
	cursor := 0
	for _, edit := range edits {
		if edit.Range.Start < cursor || edit.Range.End < edit.Range.Start || edit.Range.End > len(source) {
			return "", ErrTableColumnContext
		}
		candidate.WriteString(source[cursor:edit.Range.Start])
		candidate.WriteString(edit.Text)
		cursor = edit.Range.End
	}
	candidate.WriteString(source[cursor:])
	return candidate.String(), nil
}

func validateTableColumnCandidate(
	source, candidate string,
	before, after Document,
	blockIndex int,
	beforeRows []tableColumnRow,
	column int,
	action TableColumnAction,
) error {
	if len(after.Blocks) != len(before.Blocks) || blockIndex < 0 || blockIndex >= len(after.Blocks) ||
		len(after.Warnings) != len(before.Warnings) {
		return ErrTableColumnContext
	}
	oldTable := before.Blocks[blockIndex]
	nextTable := after.Blocks[blockIndex]
	if nextTable.Kind != BlockTable || nextTable.Table.Problem != "" {
		return ErrTableColumnLimit
	}
	delta := len(candidate) - len(source)
	if nextTable.Range.Start != oldTable.Range.Start || nextTable.Range.End != oldTable.Range.End+delta ||
		nextTable.TextRange != nextTable.Range ||
		nextTable.Raw != candidate[nextTable.Range.Start:nextTable.Range.End] ||
		len(nextTable.Table.Rows) != len(oldTable.Table.Rows) {
		return ErrTableColumnContext
	}
	if len(nextTable.Raw) > maxTableRawBytes {
		return ErrTableColumnLimit
	}
	afterRows, err := columnPhysicalRows(candidate, nextTable)
	if err != nil || len(afterRows) != len(beforeRows) {
		return ErrTableColumnContext
	}
	columns := len(oldTable.Table.Alignments)
	wantColumns := columns
	insertIndex := column
	if action == TableColumnDelete {
		wantColumns--
	} else {
		if action == TableColumnInsertAfter {
			insertIndex++
		}
		wantColumns++
	}
	if wantColumns < 1 || wantColumns > maxTableColumns || len(nextTable.Table.Alignments) != wantColumns {
		return ErrTableColumnLimit
	}
	wantAlignments := make([]TableAlignment, 0, wantColumns)
	for i, alignment := range oldTable.Table.Alignments {
		if action == TableColumnDelete && i == column {
			continue
		}
		if action != TableColumnDelete && i == insertIndex {
			wantAlignments = append(wantAlignments, TableAlignDefault)
		}
		wantAlignments = append(wantAlignments, alignment)
	}
	if action != TableColumnDelete && insertIndex == columns {
		wantAlignments = append(wantAlignments, TableAlignDefault)
	}
	if !slices.Equal(nextTable.Table.Alignments, wantAlignments) {
		return ErrTableColumnContext
	}

	for i := range beforeRows {
		if err := validateTableColumnRow(
			source, candidate, beforeRows[i], afterRows[i], i == 1, column,
			action, insertIndex,
		); err != nil {
			return err
		}
	}

	for i, old := range before.Blocks {
		if i == blockIndex {
			continue
		}
		shift := 0
		if old.Range.Start >= oldTable.Range.End {
			shift = delta
		} else if old.Range.End > oldTable.Range.Start {
			return ErrTableColumnContext
		}
		if !alignmentBlockPreserved(old, after.Blocks[i], shift) {
			return ErrTableColumnContext
		}
	}
	for i, warning := range before.Warnings {
		shift := 0
		switch {
		case warning.Range.End <= oldTable.Range.Start:
		case warning.Range.Start >= oldTable.Range.End:
			shift = delta
		default:
			return ErrTableColumnContext
		}
		next := after.Warnings[i]
		if warning.Code != next.Code || warning.Message != next.Message ||
			shiftTableRange(warning.Range, shift) != next.Range {
			return ErrTableColumnContext
		}
	}
	return nil
}

func validateTableColumnRow(
	source, candidate string,
	before, after tableColumnRow,
	delimiter bool,
	column int,
	action TableColumnAction,
	insertIndex int,
) error {
	wantCells := len(before.ref.Cells)
	if action == TableColumnDelete {
		wantCells--
	} else {
		wantCells++
	}
	leadingGainedAllowed := action != TableColumnDelete && column == 0 ||
		action == TableColumnDelete && (column == 0 || len(before.ref.Cells) == 2)
	trailingGainedAllowed := action == TableColumnInsertAfter && column == len(before.ref.Cells)-1 ||
		!delimiter && (columnInsertBeforeAddsTrailingBoundary(before, column) ||
			action == TableColumnDelete && columnDeleteLastAddsTrailingBoundary(before, column))
	if len(after.ref.Cells) != wantCells ||
		after.indent != before.indent ||
		after.leading != before.leading && !(after.leading && leadingGainedAllowed) ||
		after.trailing != before.trailing && !(after.trailing && trailingGainedAllowed) {
		return ErrTableColumnContext
	}
	oldEnding := tableRowEnding(source[before.ref.Range.Start:before.ref.Range.End])
	newEnding := tableRowEnding(candidate[after.ref.Range.Start:after.ref.Range.End])
	if oldEnding != newEnding {
		return ErrTableColumnContext
	}
	if action == TableColumnDelete {
		for oldIndex, oldCell := range before.ref.Cells {
			if oldIndex == column {
				continue
			}
			newIndex := oldIndex
			if oldIndex > column {
				newIndex--
			}
			if !sameSurvivingColumnCell(source, candidate, oldCell, after.ref.Cells[newIndex]) {
				return ErrTableColumnContext
			}
		}
		return nil
	}
	for oldIndex, oldCell := range before.ref.Cells {
		newIndex := oldIndex
		if oldIndex >= insertIndex {
			newIndex++
		}
		if !sameSurvivingColumnCell(source, candidate, oldCell, after.ref.Cells[newIndex]) {
			return ErrTableColumnContext
		}
	}
	inserted := after.ref.Cells[insertIndex]
	raw := candidate[inserted.Range.Start:inserted.Range.End]
	text := candidate[inserted.TextRange.Start:inserted.TextRange.End]
	if delimiter {
		if raw != "---" || text != "---" {
			return ErrTableColumnContext
		}
	} else if raw != "" || text != "" {
		return ErrTableColumnContext
	}
	return nil
}

func sameSurvivingColumnCell(source, candidate string, before, after TableCellRef) bool {
	if before.Range.Start < 0 || before.Range.End > len(source) ||
		after.Range.Start < 0 || after.Range.End > len(candidate) ||
		before.TextRange.Start < before.Range.Start || before.TextRange.End > before.Range.End ||
		after.TextRange.Start < after.Range.Start || after.TextRange.End > after.Range.End {
		return false
	}
	return source[before.Range.Start:before.Range.End] == candidate[after.Range.Start:after.Range.End] &&
		source[before.TextRange.Start:before.TextRange.End] == candidate[after.TextRange.Start:after.TextRange.End]
}

