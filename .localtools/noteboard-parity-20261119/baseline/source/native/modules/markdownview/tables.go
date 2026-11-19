package markdownview

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
)

const (
	tableGridCellMinWidth         = 112
	previewListHorizontalPadding  = 36
	previewRowHorizontalPadding   = 16
	tableTooltipBytes             = 512
	tableColumnLabelBytes         = 96
	tableCellLines                = 3
	tableLiteralHint              = "双击单元格或 Tab 聚焦后按 Enter 可编辑单元格原文；“编辑表格源码”保留用于整体结构修改。… 仅省略预览。"
	tableReadonlyHint             = "只读表格；完整源码仍在文档中。单元格可聚焦，按 Enter 或双击会说明只读状态。"
	tableEditButtonLabel          = "编辑表格源码"
	tableReadonlyLabel            = "只读表格"
	tableEmptyCell                = "（空单元格）"
	tableUnavailableRange         = "表格源码不可用：无法确认完整表格范围。"
	tableUnavailableNonRenderable = "表格源码不可用：当前表格没有可用的行元数据。"
)

var errTableStale = errors.New("表格位置已改变或无法确认；未打开表格源码")

type tableEditTarget struct {
	buffer  *text.Buffer
	version uint64
	rowID   uint64
	// allowProblem keeps a recognized raw-fallback table on the same immutable
	// double-click path. The ordinary source action remains valid-table only.
	allowProblem bool
	// span is the complete table in the source snapshot's UTF-16
	// coordinates. source stays empty until an actual pointer or key gesture
	// starts, so rendering a header row does not clone the complete table.
	span        text.Range
	sourceRange markdown.SourceRange
	source      string
}

func boundedTableText(value string, limit int) string {
	if len(value) <= limit {
		return strings.Clone(value)
	}
	end := max(0, limit-len("…"))
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return strings.Clone(value[:end] + "…")
}

func boundedTableTooltip(prefix, value string, limit int) string {
	if len(prefix)+len(value) <= limit {
		return strings.Clone(prefix + value)
	}
	end := max(0, limit-len(prefix)-len("…"))
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return strings.Clone(prefix + value[:end] + "…")
}

func tableCellLiteral(preview previewSnapshot, cell markdown.TableCellRef) (string, bool) {
	raw := preview.doc.Raw()
	if cell.TextRange.Start < cell.Range.Start || cell.TextRange.End < cell.TextRange.Start ||
		cell.TextRange.End > cell.Range.End || cell.Range.Start < 0 || cell.Range.End > len(raw) {
		return "", false
	}
	return raw[cell.TextRange.Start:cell.TextRange.End], true
}

func tableAlignment(a markdown.TableAlignment) ui.Align {
	switch a {
	case markdown.TableAlignCenter:
		return ui.Center
	case markdown.TableAlignRight:
		return ui.End
	default:
		return ui.Start
	}
}

func tableRowUsableForGrid(v *View, columns int) bool {
	return columns > 0 && columns <= 8 && v.tableViewportKnown &&
		v.tableViewport >= float32(columns*tableGridCellMinWidth)
}

func tableEditTargetOf(preview previewSnapshot, row previewRow, block markdown.Block, allowProblem bool) (tableEditTarget, bool) {
	if block.Kind != markdown.BlockTable || (!allowProblem && block.Table.Problem != "") ||
		row.item >= 0 || block.Range.Start < 0 || block.Range.End < block.Range.Start ||
		block.Range.End > len(preview.doc.Raw()) || len(block.Raw) != block.Range.End-block.Range.Start ||
		len(block.Raw) > maxPreviewBlockBytes {
		return tableEditTarget{}, false
	}
	if block.Table.Problem != "" {
		if row.tableRow != -1 ||
			row.span.End()-row.span.Start() != text.UTF16Len(block.Raw) {
			return tableEditTarget{}, false
		}
		return tableEditTarget{
			buffer: preview.buffer, version: preview.version, rowID: row.id,
			allowProblem: true, span: row.span, sourceRange: block.Range,
		}, true
	}
	if row.tableRow < 0 || row.tableRow >= len(block.Table.Rows) ||
		len(block.Table.Rows) == 0 || len(block.Table.Alignments) == 0 ||
		block.Table.Rows[0].Range.Start != block.Range.Start {
		return tableEditTarget{}, false
	}
	sourceRow := block.Table.Rows[row.tableRow]
	if sourceRow.Range.Start < block.Range.Start || sourceRow.Range.End > block.Range.End ||
		sourceRow.Range.Start > sourceRow.Range.End {
		return tableEditTarget{}, false
	}
	raw := preview.doc.Raw()
	prefixUnits := text.UTF16Len(raw[block.Range.Start:sourceRow.Range.Start])
	if prefixUnits > row.span.Start() ||
		row.span.End()-row.span.Start() != text.UTF16Len(raw[sourceRow.Range.Start:sourceRow.Range.End]) {
		return tableEditTarget{}, false
	}
	start := row.span.Start() - prefixUnits
	span := text.Range{Anchor: start, Head: start + text.UTF16Len(block.Raw)}
	return tableEditTarget{
		buffer: preview.buffer, version: preview.version, rowID: row.id,
		span: span, sourceRange: block.Range,
	}, true
}

func captureTableTarget(preview previewSnapshot, target tableEditTarget) (tableEditTarget, bool) {
	rng := target.sourceRange
	raw := preview.doc.Raw()
	if target.buffer == nil || target.buffer != preview.buffer ||
		rng.Start < 0 || rng.End < rng.Start || rng.End > len(raw) ||
		rng.End-rng.Start > maxPreviewBlockBytes {
		return tableEditTarget{}, false
	}
	target.source = strings.Clone(raw[rng.Start:rng.End])
	return target, true
}

func (v *View) tableRow(c *ui.Context, b markdown.Block, preview previewSnapshot, row previewRow) {
	if b.Table.Problem != "" || len(b.Table.Rows) == 0 || len(b.Table.Alignments) == 0 {
		v.tableFallback(c, b)
		return
	}
	columns := len(b.Table.Alignments)
	if row.tableRow < 0 || row.tableRow >= len(b.Table.Rows) ||
		columns <= 0 || columns > 8 ||
		len(b.Table.Rows[row.tableRow].Cells) != columns {
		v.tableFallback(c, b)
		return
	}
	tableRow := b.Table.Rows[row.tableRow]

	ui.Column(c).Gap(6).Children(func() {
		if row.tableRow == 0 {
			v.tableHeaderControls(c, b, preview, row)
		}
		if tableRowUsableForGrid(v, columns) {
			v.drawTableGrid(c, preview, row, tableRow, columns)
		} else {
			v.drawStackedTableRow(c, preview, row, tableRow, columns)
		}
	})
}

func (v *View) tableHeaderControls(c *ui.Context, b markdown.Block, preview previewSnapshot, row previewRow) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		if v.host.ReadOnly {
			ui.Text(c, tableReadonlyLabel).FontSize(11).TextColor(c.Theme().TextMuted)
		} else {
			v.tableEditButton(c, preview, row, b)
		}
		ui.Text(c, "表格").FontSize(11).TextColor(c.Theme().TextMuted)
	})
	ui.Text(c, tableLiteralHint).FontSize(11).TextColor(c.Theme().TextMuted).
		MaxLines(2).Ellipsis("…").Tooltip(boundedTableText(tableLiteralHint, tableTooltipBytes))
	if v.host.ReadOnly {
		ui.Text(c, tableReadonlyHint).FontSize(11).TextColor(c.Theme().TextMuted).
			MaxLines(2).Ellipsis("…").Tooltip(boundedTableText(tableReadonlyHint, tableTooltipBytes))
	}
}

func (v *View) tableEditButton(c *ui.Context, preview previewSnapshot, row previewRow, b markdown.Block) {
	target, ok := tableEditTargetOf(preview, row, b, false)
	ownsGesture := v.tableEditActivation.buffer == target.buffer &&
		v.tableEditActivation.rowID == target.rowID
	if ownsGesture {
		v.tableGestureBuiltRowID = target.rowID
	}
	reason := ""
	if !ok {
		reason = tableUnavailableRange
		if b.Kind == markdown.BlockTable && b.Table.Problem == "" &&
			(len(b.Table.Rows) == 0 || len(b.Table.Alignments) == 0) {
			reason = tableUnavailableNonRenderable
		}
	}
	if ownsGesture && reason != "" {
		v.tableEditActivation = tableEditTarget{}
		v.tablePointerUpBuffer, v.tablePointerUpRowID = nil, 0
	}
	button := ui.Button(c, tableEditButtonLabel).Label(tableEditButtonLabel).
		Tooltip("打开完整表格源码；保留原始字节。").Disabled(reason != "")
	if reason == "" {
		button.HandleInput(func(event ui.InputEvent) bool {
			if (event.Kind == ui.InputPointerDown && event.Button == 0 && event.Mods == 0) ||
				(event.Kind == ui.InputKeyDown && event.Mods == 0 &&
					(event.Key == ui.KeyEnter || event.Key == ui.KeySpace)) {
				captured, capturedOK := captureTableTarget(preview, target)
				if !capturedOK {
					v.tableEditActivation = tableEditTarget{}
					v.tablePointerUpBuffer, v.tablePointerUpRowID = nil, 0
					v.report(errTableStale)
					return false
				}
				v.tableEditActivation = captured
				v.tablePointerUpBuffer, v.tablePointerUpRowID = nil, 0
			}
			if event.Kind == ui.InputScroll && v.tableEditActivation.buffer != nil {
				v.tableEditActivation = tableEditTarget{}
				v.tablePointerUpBuffer, v.tablePointerUpRowID = nil, 0
			}
			if event.Kind == ui.InputPointerUp {
				captured := v.tableEditActivation
				if captured.buffer == nil {
					return false
				}
				if v.host.ReadOnly {
					v.tableEditActivation = tableEditTarget{}
					v.tablePointerUpBuffer, v.tablePointerUpRowID = nil, 0
					v.report(errTableReadOnly)
				} else if event.Canceled {
					current := v.tableIdentityCurrent(captured)
					v.tableEditActivation = tableEditTarget{}
					v.tablePointerUpBuffer, v.tablePointerUpRowID = nil, 0
					if !current {
						v.report(errTableStale)
					}
				} else {
					v.tablePointerUpBuffer, v.tablePointerUpRowID = captured.buffer, captured.rowID
				}
			}
			return false
		})
	}
	clicked := button.Clicked()
	pendingOwnsGesture := v.tablePointerUpBuffer == target.buffer &&
		v.tablePointerUpRowID == target.rowID
	if clicked {
		captured := v.tableEditActivation
		v.tableEditActivation = tableEditTarget{}
		v.tablePointerUpBuffer, v.tablePointerUpRowID = nil, 0
		if captured.buffer == nil || captured.buffer != target.buffer ||
			captured.rowID != target.rowID {
			v.report(errTableStale)
			return
		}
		if err := v.activateTable(captured); err != nil {
			v.report(err)
		}
	} else if pendingOwnsGesture {
		v.tableEditActivation = tableEditTarget{}
		v.tablePointerUpBuffer, v.tablePointerUpRowID = nil, 0
	}
	if reason != "" {
		ui.Text(c, reason).FontSize(11).TextColor(c.Theme().Warning).Tooltip(reason)
	}
}

func (v *View) drawTableGrid(c *ui.Context, preview previewSnapshot, row previewRow, tableRow markdown.TableRowRef, columns int) {
	header := row.tableRow == 0
	ui.Grid(c).Columns(columns).Gap(0).Border(1, c.Theme().Border).Clip().Children(func() {
		for index, cell := range tableRow.Cells {
			index, cell := index, cell
			value, ok := tableCellLiteral(preview, cell)
			tip := tableEmptyCell
			if ok && value != "" {
				tip = boundedTableText(value, tableTooltipBytes)
			}
			box := ui.Box(c).Key(index).FillWidth().MinWidth(0).Clip().Padding(6, 8)
			if index+1 < len(tableRow.Cells) {
				box = box.BorderEndWidth(1).BorderColor(c.Theme().Border)
			}
			if header {
				box = box.Background(c.Theme().Surface)
			}
			role := ui.RoleCell
			if header {
				role = ui.RoleColumnHeader
			}
			box = box.Role(role).Selectable()
			if target, actionOK := tableCellTargetOf(preview, row, preview.doc.Blocks[row.block], index); actionOK {
				v.noteTableCellBuilt(target)
				box = box.Focusable().TabStop(true).Label(tableCellName(row, index, value, ok)).
					Description(tableCellActionHint).HandleInput(v.tableCellInput(preview, target))
			}
			box.Children(func() {
				text := ui.Text(c, tableVisibleCell(value, ok)).Font("monospace").FontSize(13).
					LineHeight(1.35).MaxLines(tableCellLines).Ellipsis("…").
					Tooltip(tip).TextAlign(tableAlignment(alignmentAt(preview, row, index)))
				if !ok || value == "" {
					text.TextColor(c.Theme().TextMuted)
				}
				if header {
					text.Bold()
				}
			})
		}
	})
}

func (v *View) drawStackedTableRow(c *ui.Context, preview previewSnapshot, row previewRow, tableRow markdown.TableRowRef, columns int) {
	header := row.tableRow == 0
	record := ui.Column(c).Border(1, c.Theme().Border).Radius(5).Clip()
	if header {
		record = record.Background(c.Theme().Surface)
	}
	record.Children(func() {
		rowLabel := fmt.Sprintf("数据 · 第 %d 行", row.tableRow+1)
		if header {
			rowLabel = "表头 · 第 1 行"
		}
		ui.Text(c, rowLabel).FontSize(11).Bold().TextColor(c.Theme().TextMuted).
			Padding(5, 7).MaxLines(1).Ellipsis("…")
		for index, cell := range tableRow.Cells {
			index, cell := index, cell
			value, ok := tableCellLiteral(preview, cell)
			label := tableColumnLabel(preview, row, index)
			tip := tableEmptyCell
			if ok && value != "" {
				tip = boundedTableText(value, tableTooltipBytes)
			}
			ui.Grid(c).ColumnTracks(ui.Fixed(76), ui.Fr(1)).Gap(0).Children(func() {
				ui.Box(c).Padding(5, 7).BorderEndWidth(1).BorderColor(c.Theme().Border).Children(func() {
					ui.Text(c, label).FontSize(11).TextColor(c.Theme().TextMuted).
						MaxLines(2).Ellipsis("…").Tooltip(tableColumnTooltip(preview, row, index))
				})
				valueBox := ui.Box(c).Padding(5, 7).Selectable()
				if target, actionOK := tableCellTargetOf(preview, row, preview.doc.Blocks[row.block], index); actionOK {
					v.noteTableCellBuilt(target)
					role := ui.RoleCell
					if header {
						role = ui.RoleColumnHeader
					}
					valueBox = valueBox.Role(role).Focusable().TabStop(true).
						Label(tableCellName(row, index, value, ok)).Description(tableCellActionHint).
						HandleInput(v.tableCellInput(preview, target))
				}
				valueBox.Children(func() {
					text := ui.Text(c, tableVisibleCell(value, ok)).Font("monospace").FontSize(13).
						LineHeight(1.35).MaxLines(tableCellLines).Ellipsis("…").Tooltip(tip).
						TextAlign(tableAlignment(alignmentAt(preview, row, index)))
					if !ok || value == "" {
						text.TextColor(c.Theme().TextMuted)
					}
					if header {
						text.Bold()
					}
				})
			})
		}
	})
}

func tableVisibleCell(value string, ok bool) string {
	if !ok {
		return "（单元格范围不可用）"
	}
	if value == "" {
		return tableEmptyCell
	}
	return value
}

func alignmentAt(preview previewSnapshot, row previewRow, column int) markdown.TableAlignment {
	block := preview.doc.Blocks[row.block]
	if column < 0 || column >= len(block.Table.Alignments) {
		return markdown.TableAlignDefault
	}
	return block.Table.Alignments[column]
}

func tableHeaderLiteral(preview previewSnapshot, row previewRow, column int) string {
	block := preview.doc.Blocks[row.block]
	if len(block.Table.Rows) == 0 || column < 0 || column >= len(block.Table.Rows[0].Cells) {
		return ""
	}
	value, ok := tableCellLiteral(preview, block.Table.Rows[0].Cells[column])
	if !ok {
		return ""
	}
	return value
}

func tableColumnLabel(preview previewSnapshot, row previewRow, column int) string {
	if row.tableRow == 0 {
		return fmt.Sprintf("第 %d 列", column+1)
	}
	header := tableHeaderLiteral(preview, row, column)
	if header == "" {
		return fmt.Sprintf("第 %d 列", column+1)
	}
	return fmt.Sprintf("%d · %s", column+1, boundedTableText(header, tableColumnLabelBytes))
}

func tableColumnTooltip(preview previewSnapshot, row previewRow, column int) string {
	header := tableHeaderLiteral(preview, row, column)
	prefix := fmt.Sprintf("第 %d 列", column+1)
	if header == "" {
		return prefix
	}
	return boundedTableTooltip(prefix+"：", header, tableTooltipBytes)
}

func (v *View) tableFallback(c *ui.Context, b markdown.Block) {
	reason := boundedTableText(b.Table.Problem, 256)
	if reason == "" {
		reason = "表格结构无法用于预览；完整原文仍保留。"
	}
	ui.Text(c, "表格未展开："+reason).FontSize(11).TextColor(c.Theme().Warning).
		MaxLines(3).Ellipsis("…").Tooltip(boundedTableText("表格未展开："+reason, tableTooltipBytes))
	ui.Text(c, b.Raw).Font("monospace").FontSize(13).LineHeight(1.45).
		MaxLines(20).Ellipsis("…").Tooltip(boundedTableText(b.Raw, tableTooltipBytes)).Selectable()
	guidance := "双击此表格可编辑完整源码。"
	if v.host.ReadOnly {
		guidance = "只读表格；完整原文仍保留，可在源码视图查看。"
	}
	ui.Text(c, guidance).FontSize(11).TextColor(c.Theme().TextMuted)
}

func (v *View) activateTable(target tableEditTarget) error {
	if target.buffer == nil || len(target.source) > maxPreviewBlockBytes {
		return errTableStale
	}
	if err := v.Flush(); err != nil {
		return err
	}
	if v.host.ReadOnly {
		return errTableReadOnly
	}
	doc, index, span, ok := v.tableCurrent(target)
	if !ok {
		return errTableStale
	}
	block := doc.Blocks[index]
	return v.beginSource(index, block, doc, span, 0, 240)
}

func (v *View) tableIdentityCurrent(target tableEditTarget) bool {
	_, _, _, ok := v.tableCurrent(target)
	return ok
}

func (v *View) tableCurrent(target tableEditTarget) (markdown.Document, int, text.Range, bool) {
	buffer := v.host.Buffer
	if target.buffer == nil || target.source == "" || len(target.source) > maxPreviewBlockBytes ||
		buffer == nil || buffer != target.buffer || buffer.InGroup() ||
		buffer.LenBytes() > markdown.MaxParseBytes {
		return markdown.Document{}, -1, text.Range{}, false
	}
	span, ok := buffer.MapRangeSince(target.version, target.span)
	if !ok {
		return markdown.Document{}, -1, text.Range{}, false
	}
	current, err := buffer.SliceUTF16(span.Start(), span.End())
	if err != nil || current != target.source {
		return markdown.Document{}, -1, text.Range{}, false
	}
	doc := v.doc
	if v.docBuffer != buffer || v.version != buffer.Version() {
		doc = markdown.Parse(buffer.String())
	}
	if doc.Partial {
		return markdown.Document{}, -1, text.Range{}, false
	}
	raw := doc.Raw()
	byteCursor, unitCursor := 0, 0
	for index, block := range doc.Blocks {
		unitCursor += text.UTF16Len(raw[byteCursor:block.Range.Start])
		start := unitCursor
		unitCursor += text.UTF16Len(raw[block.Range.Start:block.Range.End])
		byteCursor = block.Range.End
		valid := block.Table.Problem == "" &&
			len(block.Table.Rows) > 0 && len(block.Table.Alignments) > 0 &&
			len(block.Table.Rows[0].Cells) == len(block.Table.Alignments)
		fallback := target.allowProblem && block.Table.Problem != ""
		if block.Kind == markdown.BlockTable && (valid || fallback) &&
			start == span.Start() && unitCursor == span.End() &&
			block.Raw == target.source {
			return doc, index, span, true
		}
	}
	return markdown.Document{}, -1, text.Range{}, false
}
