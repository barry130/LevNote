package markdownview

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/ui/editor"
)

const (
	tableCellMaxDraftBytes = 4096
	tableCellEditorHeight  = 48
	tableCellActionHint    = "按 Enter 编辑此单元格的 Markdown 原文；双击也可以编辑"
	tableCellStagedNotice  = "单元格草稿尚未写回；保存或完成编辑时写回"
	tableCellSessionHint   = "暂存于当前窗口，尚不进入恢复快照"
)

var (
	errTableCellStale           = errors.New("单元格位置或内容已改变，无法确认；请重新选择单元格")
	errTableCellReadOnly        = errors.New("文档当前只读，未打开单元格编辑器；当前草稿仍保留")
	errTableCellTooLarge        = errors.New("单元格草稿超过 4096 字节；未写回，草稿仍保留")
	errTableCellRaw             = errors.New("单元格源码只能包含一行；未写回，草稿仍保留。结构变化请编辑完整表格源码")
	errTableCellPipe            = errors.New("单元格源码含有未转义的表格分隔符；未写回，草稿仍保留。请用 \\| 表示字面竖线")
	errTableCellShape           = errors.New("修改会改变表格行、列或对齐结构；未写回，草稿仍保留。结构变化请编辑完整表格源码")
	errTableCellTableSize       = errors.New("修改会让表格超过 32 KiB；未写回，草稿仍保留")
	errTableCellDocumentSize    = errors.New("修改会让 Markdown 超过解析上限；未写回，草稿仍保留")
	errTableCellCurrentTooLarge = errors.New("当前 Markdown 超过解析上限，无法安全定位单元格")
	errTableCellAfterFlush      = errors.New("上一份草稿已处理；表格位置或内容已变化，请重新选择单元格")
)

type tableCellIdentity struct {
	rng, field markdown.SourceRange
}

type tableCellRowIdentity struct {
	rng   markdown.SourceRange
	cells []tableCellIdentity
}

// tableCellTarget carries only numeric descriptors until a native gesture
// starts. The two source strings are cloned by captureTableCellTarget.
type tableCellTarget struct {
	buffer      *text.Buffer
	version     uint64
	rowID       uint64
	tableSpan   text.Range
	cellSpan    text.Range
	tableRange  markdown.SourceRange
	cellRange   markdown.SourceRange
	delimiter   markdown.SourceRange
	row, column int
	columns     int
	alignments  [8]markdown.TableAlignment
	rows        []tableCellRowIdentity
	tableSource string
	cellSource  string
}

type tableCellEdit struct {
	target tableCellTarget
	label  string
}

type tableCellContext struct {
	doc               markdown.Document
	index             int
	block             markdown.Block
	tableSpan         text.Range
	cellSpan          text.Range
	row, column, cols int
}

func tableCellName(row previewRow, column int, value string, ok bool) string {
	rowKind, rowNumber := "数据", row.tableRow+1
	if row.tableRow == 0 {
		rowKind, rowNumber = "表头", 1
	}
	display := tableEmptyCell
	if !ok {
		display = "（范围不可用）"
	} else if value != "" {
		display = value
	}
	prefix := fmt.Sprintf("%s，第 %d 行，第 %d 列：", rowKind, rowNumber, column+1)
	return boundedTableTooltip(prefix, display, tableTooltipBytes)
}

func tableCellPanelLabel(row previewRow, column int) string {
	if row.tableRow == 0 {
		return fmt.Sprintf("编辑单元格 · 表头 · 第 1 行 · 第 %d 列", column+1)
	}
	return fmt.Sprintf("编辑单元格 · 数据 · 第 %d 行 · 第 %d 列", row.tableRow+1, column+1)
}

func tableCellTargetOf(preview previewSnapshot, row previewRow, block markdown.Block, column int) (tableCellTarget, bool) {
	raw := preview.doc.Raw()
	if preview.buffer == nil || block.Kind != markdown.BlockTable || block.Table.Problem != "" ||
		row.item >= 0 || row.tableRow < 0 || row.tableRow >= len(block.Table.Rows) ||
		column < 0 || column >= len(block.Table.Alignments) || len(block.Table.Alignments) > 8 ||
		block.Range.Start < 0 || block.Range.End < block.Range.Start || block.Range.End > len(raw) ||
		len(block.Raw) != block.Range.End-block.Range.Start ||
		len(block.Table.Rows) == 0 || block.Table.Rows[0].Range.Start != block.Range.Start {
		return tableCellTarget{}, false
	}
	sourceRow := block.Table.Rows[row.tableRow]
	if sourceRow.Range.Start < block.Range.Start || sourceRow.Range.End > block.Range.End ||
		sourceRow.Range.Start > sourceRow.Range.End || column >= len(sourceRow.Cells) ||
		row.span.End()-row.span.Start() != text.UTF16Len(raw[sourceRow.Range.Start:sourceRow.Range.End]) ||
		row.tableSpan.Start() < 0 || row.tableSpan.End() < row.tableSpan.Start() ||
		row.tableOffsetUnits < 0 || row.tableOffsetUnits > row.span.Start() ||
		row.span.Start()-row.tableOffsetUnits != row.tableSpan.Start() {
		return tableCellTarget{}, false
	}
	tableSpan := row.tableSpan
	cell := sourceRow.Cells[column]
	if cell.Range.Start < sourceRow.Range.Start || cell.Range.End > sourceRow.Range.End ||
		cell.Range.Start > cell.Range.End || cell.TextRange.Start < cell.Range.Start ||
		cell.TextRange.End < cell.TextRange.Start || cell.TextRange.End > cell.Range.End {
		return tableCellTarget{}, false
	}
	cellStart := row.span.Start() + text.UTF16Len(raw[sourceRow.Range.Start:cell.TextRange.Start])
	cellSpan := text.Range{
		Anchor: cellStart,
		Head:   cellStart + text.UTF16Len(raw[cell.TextRange.Start:cell.TextRange.End]),
	}

	var alignments [8]markdown.TableAlignment
	copy(alignments[:], block.Table.Alignments)
	return tableCellTarget{
		buffer: preview.buffer, version: preview.version, rowID: row.id,
		tableSpan: tableSpan, cellSpan: cellSpan, tableRange: block.Range,
		cellRange: cell.TextRange, delimiter: block.Table.DelimiterRange,
		row: row.tableRow, column: column, columns: len(block.Table.Alignments),
		alignments: alignments,
	}, true
}

func captureTableCellTarget(preview previewSnapshot, target tableCellTarget) (tableCellTarget, bool) {
	raw := preview.doc.Raw()
	if target.buffer == nil || target.buffer != preview.buffer ||
		target.tableRange.Start < 0 || target.tableRange.End < target.tableRange.Start ||
		target.tableRange.End > len(raw) ||
		target.tableRange.End-target.tableRange.Start > maxPreviewBlockBytes ||
		target.cellRange.Start < target.tableRange.Start || target.cellRange.End > target.tableRange.End ||
		target.cellRange.Start > target.cellRange.End {
		return tableCellTarget{}, false
	}
	var block markdown.Block
	found := false
	for _, candidate := range preview.doc.Blocks {
		if candidate.Kind == markdown.BlockTable && candidate.Range == target.tableRange {
			block, found = candidate, true
			break
		}
	}
	if !found || block.Table.Problem != "" || target.row < 0 || target.row >= len(block.Table.Rows) ||
		target.column < 0 || target.column >= len(block.Table.Rows[target.row].Cells) ||
		len(block.Table.Alignments) != target.columns || target.columns > 8 {
		return tableCellTarget{}, false
	}
	rows := make([]tableCellRowIdentity, len(block.Table.Rows))
	for rowIndex, tableRow := range block.Table.Rows {
		cells := make([]tableCellIdentity, len(tableRow.Cells))
		for cellIndex, tableCell := range tableRow.Cells {
			cells[cellIndex] = tableCellIdentity{rng: tableCell.Range, field: tableCell.TextRange}
		}
		rows[rowIndex] = tableCellRowIdentity{rng: tableRow.Range, cells: cells}
	}
	target.tableSource = strings.Clone(raw[target.tableRange.Start:target.tableRange.End])
	target.cellSource = strings.Clone(raw[target.cellRange.Start:target.cellRange.End])
	target.rows = rows
	return target, true
}

func sameTableCellIdentity(a, b tableCellTarget) bool {
	return a.buffer == b.buffer && a.rowID == b.rowID && a.row == b.row && a.column == b.column
}

func (v *View) noteTableCellBuilt(target tableCellTarget) {
	if sameTableCellIdentity(v.tableCellActivation, target) {
		v.tableCellGestureBuilt = true
	}
}

func (v *View) tableCellInput(preview previewSnapshot, target tableCellTarget) func(ui.InputEvent) bool {
	return func(event ui.InputEvent) bool {
		if event.Kind == ui.InputKeyDown && event.Mods == 0 && event.Key == ui.KeyEnter && !event.Repeat {
			captured, ok := captureTableCellTarget(preview, target)
			if !ok {
				v.report(errTableCellStale)
				return true
			}
			v.tableCellActivation = tableCellTarget{}
			if err := v.activateTableCell(captured); err != nil {
				v.report(err)
			}
			return true
		}
		if event.Kind == ui.InputPointerDown && event.Button == 0 && event.Mods == 0 && event.Clicks >= 2 {
			captured, ok := captureTableCellTarget(preview, target)
			if !ok {
				v.tableCellActivation = tableCellTarget{}
				v.report(errTableCellStale)
				return true
			}
			v.tableCellActivation = captured
			return true
		}
		owns := v.tableCellActivation.buffer != nil && sameTableCellIdentity(v.tableCellActivation, target)
		if event.Kind == ui.InputScroll && owns {
			v.tableCellActivation = tableCellTarget{}
			return false
		}
		if event.Kind != ui.InputPointerUp || event.Button != 0 || !owns {
			return false
		}
		captured := v.tableCellActivation
		v.tableCellActivation = tableCellTarget{}
		if event.Canceled {
			if !v.tableCellIdentityCurrent(captured) {
				v.report(errTableCellStale)
			}
			return true
		}
		if err := v.activateTableCell(captured); err != nil {
			v.report(err)
		}
		return true
	}
}

func (v *View) activateTableCell(target tableCellTarget) error {
	if target.buffer == nil || target.tableSource == "" {
		return errTableCellStale
	}
	hadDraft := v.draft != nil
	if err := v.Flush(); err != nil {
		return err
	}
	if v.host.ReadOnly {
		return errTableCellReadOnly
	}
	context, err := v.currentTableCellContext(target)
	if err != nil {
		if hadDraft {
			return errTableCellAfterFlush
		}
		return err
	}
	return v.openTableCellDraft(context, target)
}

func (v *View) currentCompleteDocument() (markdown.Document, error) {
	buffer := v.host.Buffer
	if buffer == nil {
		return markdown.Document{}, errTableCellStale
	}
	if buffer.LenBytes() > markdown.MaxParseBytes {
		return markdown.Document{}, errTableCellCurrentTooLarge
	}
	version := buffer.Version()
	if v.loaded && v.docBuffer == buffer && v.version == version && !v.doc.Partial {
		return v.doc, nil
	}
	doc := markdown.Parse(buffer.String())
	if doc.Partial {
		return markdown.Document{}, errors.New("当前 Markdown 解析不完整；单元格操作需要完整解析")
	}
	return doc, nil
}

func (v *View) currentTableCellContext(target tableCellTarget) (tableCellContext, error) {
	buffer := v.host.Buffer
	if target.buffer == nil || buffer == nil || buffer != target.buffer || buffer.InGroup() {
		return tableCellContext{}, errTableCellStale
	}
	tableSpan, tableOK := buffer.MapRangeSince(target.version, target.tableSpan)
	cellSpan, cellOK := buffer.MapRangeSince(target.version, target.cellSpan)
	if !tableOK || !cellOK {
		return tableCellContext{}, errTableCellStale
	}
	currentTable, err := buffer.SliceUTF16(tableSpan.Start(), tableSpan.End())
	if err != nil || currentTable != target.tableSource {
		return tableCellContext{}, errTableCellStale
	}
	currentCell, err := buffer.SliceUTF16(cellSpan.Start(), cellSpan.End())
	if err != nil || currentCell != target.cellSource {
		return tableCellContext{}, errTableCellStale
	}
	doc, err := v.currentCompleteDocument()
	if err != nil {
		return tableCellContext{}, err
	}
	raw := doc.Raw()
	byteCursor, unitCursor := 0, 0
	for index, block := range doc.Blocks {
		unitCursor += text.UTF16Len(raw[byteCursor:block.Range.Start])
		start := unitCursor
		unitCursor += text.UTF16Len(raw[block.Range.Start:block.Range.End])
		byteCursor = block.Range.End
		if block.Kind != markdown.BlockTable || block.Table.Problem != "" ||
			start != tableSpan.Start() || unitCursor != tableSpan.End() || block.Raw != target.tableSource {
			continue
		}
		if len(block.Table.Rows) != len(target.rows) ||
			len(block.Table.Alignments) != target.columns || target.columns == 0 || target.columns > 8 ||
			target.row < 0 || target.row >= len(block.Table.Rows) ||
			target.column < 0 || target.column >= len(block.Table.Rows[target.row].Cells) {
			return tableCellContext{}, errTableCellStale
		}
		for rowIndex, tableRow := range block.Table.Rows {
			if len(tableRow.Cells) != len(target.rows[rowIndex].cells) {
				return tableCellContext{}, errTableCellStale
			}
		}
		for index := range block.Table.Alignments {
			if block.Table.Alignments[index] != target.alignments[index] {
				return tableCellContext{}, errTableCellStale
			}
		}
		cell := block.Table.Rows[target.row].Cells[target.column]
		if cell.TextRange.Start < block.Range.Start || cell.TextRange.End > block.Range.End ||
			cell.TextRange.Start > cell.TextRange.End {
			return tableCellContext{}, errTableCellStale
		}
		startUnits := tableSpan.Start() + text.UTF16Len(raw[block.Range.Start:cell.TextRange.Start])
		endUnits := startUnits + text.UTF16Len(raw[cell.TextRange.Start:cell.TextRange.End])
		if startUnits != cellSpan.Start() || endUnits != cellSpan.End() ||
			raw[cell.TextRange.Start:cell.TextRange.End] != target.cellSource {
			return tableCellContext{}, errTableCellStale
		}
		return tableCellContext{
			doc: doc, index: index, block: block, tableSpan: tableSpan, cellSpan: cellSpan,
			row: target.row, column: target.column, cols: target.columns,
		}, nil
	}
	return tableCellContext{}, errTableCellStale
}

func (v *View) tableCellIdentityCurrent(target tableCellTarget) bool {
	_, err := v.currentTableCellContext(target)
	return err == nil
}

func (v *View) openTableCellDraft(context tableCellContext, target tableCellTarget) error {
	buffer := v.host.Buffer
	if buffer == nil || buffer != target.buffer || v.host.ReadOnly || context.doc.Partial {
		return errTableCellStale
	}
	v.clearDraft()
	v.tableActionError = nil
	v.setPreview(context.doc)
	v.version, v.docBuffer, v.loaded = buffer.Version(), buffer, true
	v.editing = context.index
	v.editVersion, v.editBuffer = buffer.Version(), buffer
	v.editSpan, v.editGuard = context.cellSpan, context.tableSpan
	v.editBase = strings.Clone(target.cellSource)
	v.editPanelHeight = tableCellEditorHeight
	v.cellEdit = &tableCellEdit{
		target: target,
		label:  tableCellPanelLabel(v.previewTableRow(context.index, target.row), target.column),
	}

	draft := editor.New(text.New(v.editBase))
	draft.ShapeLimit = tableCellMaxDraftBytes
	draft.MinHeight = 48
	draft.LineNumbers = false
	draft.Label = v.cellEdit.label
	draft.OnError = v.setDraftError
	draft.OnChange = func() {
		if v.draft != draft || v.cellEdit != nil {
			return
		}
		_ = v.syncDraft()
	}
	v.draft = draft
	draft.Focus()
	return nil
}

func (v *View) previewTableRow(block, tableRow int) previewRow {
	for _, row := range v.rows {
		if row.block == block && row.tableRow == tableRow {
			return row
		}
	}
	return previewRow{tableRow: tableRow}
}

func (v *View) syncCellDraft() error {
	if v.draft == nil || v.cellEdit == nil {
		return nil
	}
	plan, err := v.prepareTableCellDraft()
	if err != nil {
		v.setDraftError(err)
		return err
	}
	if !plan.changed {
		v.finishTableCellDraft()
		return nil
	}
	if err := v.host.Buffer.ReplaceUTF16(plan.cellSpan.Start(), plan.cellSpan.End(), plan.value); err != nil {
		v.setDraftError(err)
		return err
	}
	v.finishTableCellDraft()
	if v.host.Changed != nil {
		v.host.Changed()
	}
	return nil
}

// The preparation shared by normal Flush and a row action owns only bounded
// strings and coordinates. No parsed document/block pointer crosses Flush.
type tableCellPlan struct {
	cellSpan    text.Range
	tableRange  markdown.SourceRange
	rowRange    markdown.SourceRange
	tableSource string
	value       string
	column      int
	changed     bool
}

func (v *View) prepareTableCellDraft() (tableCellPlan, error) {
	if v.draft == nil || v.cellEdit == nil {
		return tableCellPlan{}, errTableCellStale
	}
	target := v.cellEdit.target
	if v.host.ReadOnly {
		return tableCellPlan{}, errTableCellReadOnly
	}
	if v.draft.Buffer.LenBytes() > tableCellMaxDraftBytes {
		return tableCellPlan{}, errTableCellTooLarge
	}
	context, err := v.currentTableCellContext(target)
	if err != nil {
		return tableCellPlan{}, err
	}
	value := v.draft.Buffer.String()
	if !utf8.ValidString(value) || strings.ContainsAny(value, "\r\n") {
		return tableCellPlan{}, errTableCellRaw
	}
	for index := 0; index < len(value); index++ {
		if value[index] == '|' && tableCellStructuralAt(value, index) {
			return tableCellPlan{}, errTableCellPipe
		}
	}
	plan := tableCellPlan{cellSpan: context.cellSpan, tableRange: context.block.Range,
		rowRange: context.block.Table.Rows[context.row].Range, tableSource: target.tableSource, value: value, column: context.column}
	if value == target.cellSource {
		return plan, nil
	}
	delta := len(value) - len(target.cellSource)
	if len(target.tableSource)+delta > maxPreviewBlockBytes {
		return tableCellPlan{}, errTableCellTableSize
	}
	if delta > 0 && v.host.Buffer.LenBytes() > markdown.MaxParseBytes-delta {
		return tableCellPlan{}, errTableCellDocumentSize
	}
	if err := validateTableCellCandidate(target, value, delta); err != nil {
		return tableCellPlan{}, err
	}
	if v.host.Buffer.InGroup() {
		return tableCellPlan{}, text.ErrGroup
	}
	fieldStart, fieldEnd := target.cellRange.Start-target.tableRange.Start, target.cellRange.End-target.tableRange.Start
	plan.tableSource = target.tableSource[:fieldStart] + value + target.tableSource[fieldEnd:]
	plan.tableRange.End += delta
	plan.rowRange.End += delta
	plan.changed = true
	return plan, nil
}

func tableCellStructuralAt(value string, index int) bool {
	backslashes := 0
	for cursor := index - 1; cursor >= 0 && value[cursor] == '\\'; cursor-- {
		backslashes++
	}
	return backslashes%2 == 0
}

func validateTableCellCandidate(target tableCellTarget, value string, delta int) error {
	fieldStart := target.cellRange.Start - target.tableRange.Start
	fieldEnd := target.cellRange.End - target.tableRange.Start
	if fieldStart < 0 || fieldEnd < fieldStart || fieldEnd > len(target.tableSource) {
		return errTableCellStale
	}
	candidate := target.tableSource[:fieldStart] + value + target.tableSource[fieldEnd:]
	doc := markdown.Parse(candidate)
	if doc.Partial || len(doc.Blocks) != 1 {
		return errTableCellShape
	}
	block := doc.Blocks[0]
	if block.Kind != markdown.BlockTable || block.Table.Problem != "" || block.Range.Start != 0 ||
		block.Range.End != len(candidate) || block.Raw != candidate ||
		len(block.Table.Rows) != len(target.rows) ||
		len(block.Table.Alignments) != target.columns {
		return errTableCellShape
	}
	for index, alignment := range target.alignments[:target.columns] {
		if block.Table.Alignments[index] != alignment {
			return errTableCellShape
		}
	}
	localCell := markdown.SourceRange{
		Start: fieldStart,
		End:   fieldEnd,
	}
	wantRange := func(value markdown.SourceRange) markdown.SourceRange {
		value.Start -= target.tableRange.Start
		value.End -= target.tableRange.Start
		ownsField := value.Start <= localCell.Start && value.End >= localCell.End
		start := tableCellTranslatedStart(value.Start, localCell, delta, ownsField)
		end := tableCellTranslatedEnd(value.End, localCell, delta, ownsField)
		return markdown.SourceRange{Start: start, End: end}
	}
	if block.Table.DelimiterRange != wantRange(target.delimiter) {
		return errTableCellShape
	}
	for rowIndex, tableRow := range block.Table.Rows {
		original := target.rows[rowIndex]
		if len(tableRow.Cells) != len(original.cells) || tableRow.Range != wantRange(original.rng) {
			return errTableCellShape
		}
		for cellIndex, cell := range tableRow.Cells {
			originalCell := original.cells[cellIndex]
			wantCellRange := wantRange(originalCell.rng)
			targetCell := rowIndex == target.row && cellIndex == target.column
			if targetCell {
				// Leading ASCII padding on the first no-outer cell becomes row
				// indentation, so the parser may omit it from Range. Keep the
				// candidate inside the translated original cell envelope; the
				// envelope check still rejects an outer pipe absorbed by an odd
				// trailing backslash or any other delimiter reinterpretation.
				if cell.Range.Start < wantCellRange.Start || cell.Range.End > wantCellRange.End ||
					cell.Range.Start > cell.Range.End {
					return errTableCellShape
				}
			} else if cell.Range != wantCellRange {
				return errTableCellShape
			}
			if !targetCell {
				if cell.TextRange != wantRange(originalCell.field) {
					return errTableCellShape
				}
			}
		}
	}
	return nil
}

func tableCellTranslatedStart(point int, field markdown.SourceRange, delta int, ownsField bool) int {
	if point > field.End || point == field.Start && field.Start == field.End && !ownsField {
		return point + delta
	}
	return point
}

func tableCellTranslatedEnd(point int, field markdown.SourceRange, delta int, ownsField bool) int {
	if point > field.End || point == field.End && ownsField {
		return point + delta
	}
	return point
}

func (v *View) finishTableCellDraft() {
	v.clearDraft()
	v.list.HoldPlace()
	v.loaded = false
}
