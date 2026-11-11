package markdown

import "strings"

const (
	maxTableColumns         = 8
	maxTableBodyRows        = 256
	maxTableCellTextBytes   = 4096
	maxTableRawBytes        = 32 << 10
	maxTableProblemBytes    = 256
	maxTableDescriptorCells = 16384
)

const (
	tableFallbackMessage  = "table syntax is not supported; complete source remains visible"
	tableIndentMessage    = "table rows may use at most three leading spaces or tabs"
	tableColumnMessage    = "table has more than eight columns"
	tableCellMessage      = "table cell exceeds the 4096-byte preview limit"
	tableOneColumnMessage = "one-column table rows require a leading pipe"
	tableCountMessage     = "table header and delimiter have different column counts"
	tableDelimiterMessage = "table delimiter is malformed"
	tableBodyCountMessage = "table body row has a different column count"
	tableRowsMessage      = "table exceeds the 256-body-row or descriptor limit"
	tableRawMessage       = "table exceeds the 32 KiB preview limit; complete source remains visible"
	tableRangeMessage     = "table source ranges are invalid"
)

type tableScan struct {
	row                TableRowRef
	indent             int
	pipes              int
	valid              bool
	tooIndented        bool
	overColumns        bool
	overCell           bool
	oneColumnNoLeading bool
}

// parseTable recognizes only the declared pipe-table subset. A pair is a
// candidate when the first row has a structural pipe and the second line has a
// delimiter shape, even if either is malformed. Candidate failures become one
// complete raw BlockTable instead of falling back through other block scanners.
func parseTable(source string, lines []line, index, usedCells int) (Block, int, []Warning, int, bool) {
	if index+1 >= len(lines) ||
		!tableHeaderEligible(lines[index].content) ||
		!tableCandidate(lines[index].content) ||
		!delimiterShapedContent(lines[index+1].content) {
		return Block{}, index, nil, usedCells, false
	}

	header := scanTableRow(source, lines[index])
	delimiter := scanTableRow(source, lines[index+1])
	if !header.valid {
		return tableFailed(source, lines, index, usedCells, scanFailure(header))
	}
	if !delimiter.valid {
		return tableFailed(source, lines, index, usedCells, scanFailure(delimiter))
	}
	if len(header.row.Cells) != len(delimiter.row.Cells) {
		return tableFailed(source, lines, index, usedCells, tableCountMessage)
	}
	alignments, ok := parseTableAlignments(source, delimiter.row)
	if !ok {
		return tableFailed(source, lines, index, usedCells, tableDelimiterMessage)
	}
	if usedCells+len(header.row.Cells) > maxTableDescriptorCells {
		return tableFailed(source, lines, index, usedCells,
			"Markdown table descriptor limit reached; table source remains visible")
	}

	rows := []TableRowRef{header.row}
	projectedCells := usedCells + len(header.row.Cells)
	next := index + 2
	problem := ""
	for next < len(lines) {
		if tableBoundary(lines[next].content) || !tableCandidate(lines[next].content) {
			break
		}
		current := scanTableRow(source, lines[next])
		if !current.valid {
			problem = scanFailure(current)
			break
		}
		if len(current.row.Cells) != len(header.row.Cells) {
			problem = tableBodyCountMessage
			break
		}
		if len(rows)-1 >= maxTableBodyRows ||
			projectedCells+len(current.row.Cells) > maxTableDescriptorCells {
			problem = tableRowsMessage
			break
		}
		projectedCells += len(current.row.Cells)
		rows = append(rows, current.row)
		next++
	}
	if problem != "" {
		return tableFailed(source, lines, index, usedCells, problem)
	}

	rawStart := lines[index].start
	rawEnd := lines[next-1].end
	if rawEnd < rawStart || rawEnd > len(source) || !validRows(source, rows) {
		return tableFailed(source, lines, index, usedCells, tableRangeMessage)
	}
	if rawEnd-rawStart > maxTableRawBytes {
		return tableFailed(source, lines, index, usedCells, tableRawMessage)
	}
	return Block{
		Kind:      BlockTable,
		Range:     rangeAt(rawStart, rawEnd),
		TextRange: rangeAt(rawStart, rawEnd),
		Raw:       source[rawStart:rawEnd],
		Table: TableRef{
			Rows:           rows,
			DelimiterRange: lineRange(lines[index+1]),
			Alignments:     alignments,
		},
	}, next, nil, projectedCells, true
}

func scanFailure(scan tableScan) string {
	switch {
	case scan.tooIndented:
		return tableIndentMessage
	case scan.overColumns:
		return tableColumnMessage
	case scan.overCell:
		return tableCellMessage
	case scan.oneColumnNoLeading:
		return tableOneColumnMessage
	default:
		return tableFallbackMessage
	}
}

func tableFailed(source string, lines []line, first, usedCells int, reason string) (Block, int, []Warning, int, bool) {
	// The header and delimiter are always part of a recognized candidate.
	// Continue through every contiguous pipe row until a normal boundary so an
	// invalid row cannot split one physical table into repeated weak warnings.
	end := first + 2
	if end > len(lines) {
		end = len(lines)
	}
	for end < len(lines) &&
		!tableBoundary(lines[end].content) &&
		tableCandidate(lines[end].content) {
		end++
	}
	rawStart := lines[first].start
	rawEnd := lines[end-1].end
	problem := boundedProblem(reason, maxTableProblemBytes)
	rng := rangeAt(rawStart, rawEnd)
	return Block{
			Kind:      BlockTable,
			Range:     rng,
			TextRange: rng,
			Raw:       source[rawStart:rawEnd],
			Table:     TableRef{Problem: problem},
		}, end, []Warning{{
			Code: "unsupported-table", Message: problem, Range: rng,
		}}, usedCells, true
}

func boundedProblem(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	end := limit - len("…")
	for end > 0 && !isRuneStart(value, end) {
		end--
	}
	return value[:end] + "…"
}

func isRuneStart(value string, index int) bool {
	if index <= 0 || index >= len(value) {
		return true
	}
	return value[index]&0xC0 != 0x80
}

func scanTableRow(source string, current line) tableScan {
	content := current.content
	first := 0
	for first < len(content) && isTablePadding(content[first]) {
		first++
	}
	if first >= len(content) {
		return tableScan{}
	}

	lastNonPadding := len(content)
	for lastNonPadding > first && isTablePadding(content[lastNonPadding-1]) {
		lastNonPadding--
	}
	if lastNonPadding <= first {
		return tableScan{}
	}

	var positions [maxTableColumns + 2]int
	pipes := 0
	for i := first; i < lastNonPadding; i++ {
		if content[i] == '|' && structuralAt(source, current.start+i) {
			if pipes < len(positions) {
				positions[pipes] = i
			}
			pipes++
		}
	}
	if pipes == 0 {
		return tableScan{}
	}

	scan := tableScan{
		indent:      first,
		pipes:       pipes,
		tooIndented: first > 3,
		overColumns: pipes > maxTableColumns+1,
	}
	if scan.tooIndented || scan.overColumns {
		return scan
	}

	leading := positions[0] == first
	trailing := positions[pipes-1] == lastNonPadding-1 && lastNonPadding-1 != first
	cells := make([]TableCellRef, 0, maxTableColumns)
	cursor := first
	if leading {
		cursor = positions[0] + 1
	}
	appendCell := func(start, end int) bool {
		if start < first || end < start || end > len(content) {
			scan.overCell = true
			return false
		}
		textStart, textEnd := start, end
		for textStart < textEnd && isTablePadding(content[textStart]) {
			textStart++
		}
		for textEnd > textStart && isTablePadding(content[textEnd-1]) {
			textEnd--
		}
		if textEnd-textStart > maxTableCellTextBytes {
			scan.overCell = true
			return false
		}
		cells = append(cells, TableCellRef{
			Range:     rangeAt(current.start+start, current.start+end),
			TextRange: rangeAt(current.start+textStart, current.start+textEnd),
		})
		if len(cells) > maxTableColumns {
			scan.overColumns = true
			return false
		}
		return true
	}

	for positionIndex := 0; positionIndex < pipes; positionIndex++ {
		position := positions[positionIndex]
		if leading && positionIndex == 0 {
			continue
		}
		if trailing && positionIndex == pipes-1 {
			break
		}
		if !appendCell(cursor, position) {
			return scan
		}
		cursor = position + 1
	}
	segmentEnd := len(content)
	if trailing {
		segmentEnd = positions[pipes-1]
	}
	if segmentEnd < cursor {
		// A single structural pipe is both an empty row's leading and trailing
		// outer marker. It still contains one stable empty cell.
		segmentEnd = cursor
	}
	if !appendCell(cursor, segmentEnd) {
		return scan
	}

	scan.row = TableRowRef{Range: lineRange(current), Cells: cells}
	scan.oneColumnNoLeading = len(cells) == 1 && !leading
	scan.valid = !scan.tooIndented && !scan.overColumns && !scan.overCell &&
		!scan.oneColumnNoLeading && len(cells) > 0
	return scan
}

func isTablePadding(value byte) bool { return value == ' ' || value == '\t' }

func structuralAt(source string, index int) bool {
	backslashes := 0
	for i := index - 1; i >= 0 && source[i] == '\\'; i-- {
		backslashes++
	}
	return backslashes%2 == 0
}

// tableCandidate deliberately ignores the indentation cap. A too-indented row
// below a recognized header remains part of the one fallback block.
func tableCandidate(content string) bool {
	start := 0
	for start < len(content) && isTablePadding(content[start]) {
		start++
	}
	last := len(content)
	for last > start && isTablePadding(content[last-1]) {
		last--
	}
	if start >= last {
		return false
	}
	backslashes := 0
	for i := start; i < last; i++ {
		switch content[i] {
		case '\\':
			backslashes++
		case '|':
			if backslashes%2 == 0 {
				return true
			}
			backslashes = 0
		default:
			backslashes = 0
		}
	}
	return false
}

// delimiterShapedContent accepts the full colon/hyphen delimiter shape,
// including malformed dash counts and colon placement. This lets parseTable
// retain a complete visible fallback instead of trying to reinterpret a broken
// delimiter as prose. Arbitrary pipe-bearing prose is not a delimiter.
func delimiterShapedContent(content string) bool {
	first := 0
	for first < len(content) && isTablePadding(content[first]) {
		first++
	}
	last := len(content)
	for last > first && isTablePadding(content[last-1]) {
		last--
	}
	if last <= first {
		return false
	}

	hasStructuralPipe := false
	for i := first; i < last; i++ {
		if content[i] == '|' && structuralAt(content, i) {
			hasStructuralPipe = true
			break
		}
	}
	if !hasStructuralPipe {
		return false
	}

	logicalStart := first
	logicalEnd := last
	if content[logicalStart] == '|' && structuralAt(content, logicalStart) {
		logicalStart++
	}
	if logicalEnd > logicalStart &&
		content[logicalEnd-1] == '|' &&
		structuralAt(content, logicalEnd-1) {
		logicalEnd--
	}

	cellHasHyphen := false
	flushCell := func() bool {
		value := cellHasHyphen
		cellHasHyphen = false
		return value
	}
	for i := logicalStart; i <= logicalEnd; i++ {
		if i == logicalEnd {
			if !flushCell() {
				return false
			}
			break
		}
		switch content[i] {
		case '\\':
			return false
		case '|':
			if !flushCell() {
				return false
			}
			continue
		case '-':
			cellHasHyphen = true
		case ':':
		case ' ', '\t':
		default:
			return false
		}
	}
	return true
}

func tableBoundary(content string) bool {
	return isBlank(content) || isBlockStart(content) || isThematic(content)
}

func tableHeaderEligible(content string) bool {
	list, _, _ := startsList(content)
	return !isBlank(content) &&
		!headingRe.MatchString(content) &&
		!list &&
		!openFenceRe.MatchString(content) &&
		!strings.HasPrefix(content, ">") &&
		!isThematic(content)
}

func parseTableAlignments(source string, delimiter TableRowRef) ([]TableAlignment, bool) {
	alignments := make([]TableAlignment, len(delimiter.Cells))
	for i, cell := range delimiter.Cells {
		value := source[cell.TextRange.Start:cell.TextRange.End]
		left := len(value) > 0 && value[0] == ':'
		right := len(value) > 0 && value[len(value)-1] == ':'
		start, end := 0, len(value)
		if left {
			start++
		}
		if right {
			end--
		}
		if end-start < 3 {
			return nil, false
		}
		for i := start; i < end; i++ {
			if value[i] != '-' {
				return nil, false
			}
		}
		switch {
		case left && right:
			alignments[i] = TableAlignCenter
		case left:
			alignments[i] = TableAlignLeft
		case right:
			alignments[i] = TableAlignRight
		default:
			alignments[i] = TableAlignDefault
		}
	}
	return alignments, true
}

func validRows(source string, rows []TableRowRef) bool {
	for _, row := range rows {
		if row.Range.Start < 0 || row.Range.End < row.Range.Start || row.Range.End > len(source) {
			return false
		}
		previousEnd := row.Range.Start
		for _, cell := range row.Cells {
			if cell.Range.Start < previousEnd ||
				cell.Range.End < cell.Range.Start ||
				cell.Range.End > row.Range.End ||
				cell.TextRange.Start < cell.Range.Start ||
				cell.TextRange.End > cell.Range.End ||
				cell.TextRange.Start > cell.TextRange.End {
				return false
			}
			previousEnd = cell.Range.End
		}
	}
	return true
}
