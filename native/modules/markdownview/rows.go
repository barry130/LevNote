package markdownview

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
)

type previewRow struct {
	id               uint64
	block, item      int // item -1 means a whole block
	tableRow         int // -1 means this row is not one physical table row
	span             text.Range
	tableSpan        text.Range // complete table span, for a physical table row
	tableOffsetUnits int        // UTF-16 units from the table start to this row
	kind             markdown.BlockKind
}

type rowIdentity struct {
	span     text.Range
	kind     markdown.BlockKind
	item     bool
	tableRow int
}

type previewSnapshot struct {
	doc     markdown.Document
	rows    []previewRow
	buffer  *text.Buffer
	version uint64
}

// Only numeric identity/range descriptors live outside the parser. Map old
// unchanged rows through the edit journal, so source insertions above the
// viewport preserve the list anchor without retaining another source string.
func (v *View) setPreview(doc markdown.Document) {
	previous := make(map[rowIdentity]uint64, len(v.rows))
	if v.docBuffer == v.host.Buffer && v.docBuffer != nil {
		for _, row := range v.rows {
			if span, ok := v.docBuffer.MapRangeSince(v.version, row.span); ok {
				previous[rowIdentity{
					span: span, kind: row.kind, item: row.item >= 0, tableRow: row.tableRow,
				}] = row.id
			}
		}
	}
	// An exact verified own toggle changes only one marker character. Its
	// row may keep keyboard focus; no other overlapping edit gets this path.
	if target := v.taskRow; target.buffer != nil && target.buffer == v.host.Buffer {
		if span, ok := target.buffer.MapRangeSince(target.version, target.item); ok {
			previous[rowIdentity{
				span: span, kind: markdown.BlockList, item: true, tableRow: -1,
			}] = target.rowID
		}
	}
	v.taskRow = taskTarget{}
	v.hasLinks = false
	v.hasTables = false
	rows := make([]previewRow, 0, len(doc.Blocks))
	byteCursor, unitCursor := 0, 0
	appendRow := func(block, item, tableRow int, r markdown.SourceRange, tableSpan text.Range, tableOffsetUnits int) {
		unitCursor += text.UTF16Len(doc.Raw()[byteCursor:r.Start])
		start := unitCursor
		unitCursor += text.UTF16Len(doc.Raw()[r.Start:r.End])
		byteCursor = r.End
		span := text.Range{Anchor: start, Head: unitCursor}
		identity := rowIdentity{span, doc.Blocks[block].Kind, item >= 0, tableRow}
		id, ok := previous[identity]
		if !ok {
			v.nextRowID++
			id = v.nextRowID
		}
		rows = append(rows, previewRow{
			id: id, block: block, item: item, tableRow: tableRow, span: span,
			tableSpan: tableSpan, tableOffsetUnits: tableOffsetUnits, kind: identity.kind,
		})
	}
	if !doc.Partial {
		for index, block := range doc.Blocks {
			if block.Kind != markdown.BlockTable {
				v.hasLinks = v.hasLinks || containsLink(block.Inline)
			}
			if block.Kind == markdown.BlockTable {
				v.hasTables = true
				if block.Table.Problem == "" && len(block.Table.Rows) > 0 {
					tableStart := unitCursor + text.UTF16Len(doc.Raw()[byteCursor:block.Range.Start])
					tableSpan := text.Range{
						Anchor: tableStart,
						Head:   tableStart + text.UTF16Len(doc.Raw()[block.Range.Start:block.Range.End]),
					}
					for tableRow, table := range block.Table.Rows {
						appendRow(index, -1, tableRow, table.Range, tableSpan, 0)
						rows[len(rows)-1].tableOffsetUnits = rows[len(rows)-1].span.Start() - tableSpan.Start()
					}
				} else {
					appendRow(index, -1, -1, block.Range, text.Range{}, 0)
				}
			} else if block.Kind == markdown.BlockList && len(block.Items) > 0 {
				for item, entry := range block.Items {
					v.hasLinks = v.hasLinks || containsLink(entry.Inline)
					appendRow(index, item, -1, entry.Range, text.Range{}, 0)
				}
			} else {
				appendRow(index, -1, -1, block.Range, text.Range{}, 0)
			}
		}
	}
	v.doc, v.rows = doc, rows
}

func (v *View) drawPreviewRow(c *ui.Context, preview previewSnapshot, index int) {
	v.frameRows++
	entry := preview.rows[index]
	block := preview.doc.Blocks[entry.block]
	row := ui.Column(c).Padding(4, 8).Gap(8)
	row.ObserveInput(func(event ui.InputEvent) { v.linkModifiers = event.Mods })
	row.HandleInput(func(event ui.InputEvent) bool {
		if event.Kind == ui.InputPointerDown && event.Button == 0 && event.Clicks >= 2 && !v.host.ReadOnly {
			if entry.tableRow >= 0 && block.Kind == markdown.BlockTable && block.Table.Problem == "" {
				target, ok := tableEditTargetOf(preview, entry, block, false)
				captured, capturedOK := captureTableTarget(preview, target)
				if !ok || !capturedOK {
					v.report(errTableStale)
				} else if err := v.activateTable(captured); err != nil {
					v.report(err)
				}
			} else if block.Kind == markdown.BlockTable && block.Table.Problem != "" &&
				len(block.Raw) <= maxPreviewBlockBytes {
				target, ok := tableEditTargetOf(preview, entry, block, true)
				captured, capturedOK := captureTableTarget(preview, target)
				if !ok || !capturedOK {
					v.report(errTableStale)
				} else if err := v.activateTable(captured); err != nil {
					v.report(err)
				}
			} else {
				v.begin(entry.block, block)
			}
			return true
		}
		return false
	})
	row.Children(func() {
		inline, visibleBytes := block.Inline, len(block.Raw)
		if entry.item >= 0 {
			inline, visibleBytes = block.Items[entry.item].Inline, len(block.Items[entry.item].Text)
		}
		if visibleBytes <= maxPreviewBlockBytes {
			if reason := linkFallbackReason(inline); reason != "" {
				ui.Text(c, reason).FontSize(11).TextColor(c.Theme().TextMuted)
			}
		}
		// A staged cell keeps its immutable table preview visible as editing
		// context. Only ordinary/whole-table drafts replace their block because
		// their one editor already contains the complete source.
		if v.draft != nil && v.cellEdit == nil && v.editing == entry.block && entry.item < 0 {
			ui.Text(c, "正在编辑此块").FontSize(11).TextColor(c.Theme().TextMuted)
			return
		}
		if v.draft != nil && v.editing == entry.block && entry.item == 0 {
			ui.Text(c, "列表草稿在上方；此处保留原文预览").FontSize(11).TextColor(c.Theme().TextMuted)
		}
		if entry.item >= 0 {
			item := block.Items[entry.item]
			if len(item.Text) > maxPreviewBlockBytes && item.Task == nil {
				ui.Text(c, "较大列表项：双击编辑，或切到源码查看完整内容").TextColor(c.Theme().Warning)
			} else {
				v.listItem(c, block, entry.item, preview.taskTarget(entry, item), preview, entry)
			}
		} else if len(block.Raw) > maxPreviewBlockBytes {
			ui.Text(c, fmt.Sprintf("较大内容块（%d 字节）：双击分段编辑，或切到源码查看完整内容", len(block.Raw))).TextColor(c.Theme().Warning)
		} else {
			v.block(c, block, preview, entry)
		}
	})
}
