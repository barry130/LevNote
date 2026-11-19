package markdownview

import (
	"errors"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/ui/editor"
)

func planTableColumnAction(source string, plan tableCellPlan, action markdown.TableColumnAction) (tableActionPatch, error) {
	patch, err := markdown.PlanTableColumnEdit(source, plan.tableRange, plan.rowRange, plan.column, action)
	if err != nil {
		switch {
		case errors.Is(err, markdown.ErrTableColumnMinimum):
			return tableActionPatch{}, errors.New("表格至少保留一列；可编辑完整表格源码")
		case errors.Is(err, markdown.ErrTableColumnLimit):
			return tableActionPatch{}, errors.New("操作会超过表格或文档上限；可编辑完整表格源码")
		case errors.Is(err, markdown.ErrTableColumnContext):
			return tableActionPatch{}, errors.New("操作无法保留原始单元格或其他内容的解析状态；可编辑完整表格源码")
		default:
			return tableActionPatch{}, errTableActionStale
		}
	}
	edits := make([]tableActionEdit, len(patch.Edits))
	for i, edit := range patch.Edits {
		edits[i] = tableActionEdit{rng: edit.Range, value: edit.Text}
	}
	return tableActionPatch{edits: edits}, nil
}

func (v *View) drawTableColumnButtons(c *ui.Context, edit *tableCellEdit, draft *editor.Editor) {
	ui.Row(c.Key("table-column-actions")).Wrap().Gap(4).Children(func() {
		for _, action := range []markdown.TableColumnAction{markdown.TableColumnInsertBefore, markdown.TableColumnInsertAfter, markdown.TableColumnDelete} {
			face, name := "前插列", "在此列前插入列"
			tip := "先写回单元格，再插入一个空列；表格最多八列。"
			if action == markdown.TableColumnInsertAfter {
				face, name = "后插列", "在此列后插入列"
			} else if action == markdown.TableColumnDelete {
				face, name = "删此列", "删除此列"
				tip = "先写回单元格，再删除当前整列；表格至少保留一列。"
			}
			disabled := v.host.ReadOnly || action == markdown.TableColumnDelete && edit.target.columns <= 1 ||
				action != markdown.TableColumnDelete && edit.target.columns >= 8
			button := ui.Button(c.Key(action), face).Label(name).Width(62).Height(26).Padding(0, 4).
				FontSize(11).Tooltip(tip).Disabled(disabled)
			v.bindTableAction(c, button, tableAction{kind: tableActionColumn, column: action}, edit, draft)
		}
	})
}
