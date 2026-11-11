package markdownview

import (
	"errors"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/ui/editor"
)

func (v *View) captureTableRow(action markdown.TableRowAction, edit *tableCellEdit, draft *editor.Editor) tableActionGesture {
	return v.captureTableAction(tableAction{kind: tableActionRow, row: action}, edit, draft)
}

func (v *View) activateTableRow(g tableActionGesture) error { return v.activateTableAction(g) }

func planTableRowAction(source string, plan tableCellPlan, action markdown.TableRowAction) (tableActionPatch, error) {
	patch, err := markdown.PlanTableRowEdit(source, plan.tableRange, plan.rowRange, action)
	if err != nil {
		switch {
		case errors.Is(err, markdown.ErrTableRowHeader):
			return tableActionPatch{}, errors.New("不能删除表头；请选中数据行")
		case errors.Is(err, markdown.ErrTableRowLimit):
			return tableActionPatch{}, errors.New("操作会超过表格或文档上限；可编辑完整表格源码")
		case errors.Is(err, markdown.ErrTableRowContext):
			return tableActionPatch{}, errors.New("操作会改变其他内容的解析状态；可编辑完整表格源码")
		default:
			return tableActionPatch{}, errTableActionStale
		}
	}
	return tableActionPatch{edits: []tableActionEdit{{rng: patch.Range, value: patch.Text}}}, nil
}

func (v *View) drawTableRowButtons(c *ui.Context, edit *tableCellEdit, draft *editor.Editor) {
	ui.Row(c.Key("table-row-actions")).Gap(8).Children(func() {
		for _, action := range []markdown.TableRowAction{markdown.TableRowInsertAfter, markdown.TableRowDelete} {
			face, name, tip := "插入行", "在下方插入行", "表头会在分隔行后插入第一条数据行；其他行在当前行后插入。"
			if action == markdown.TableRowDelete {
				face, name, tip = "删此行", "删除此数据行", "删除当前数据行，保留其他行的原始内容和行尾。"
			}
			disabled := v.host.ReadOnly || action == markdown.TableRowDelete && edit.target.row == 0
			button := ui.Button(c.Key(action), face).Label(name).Height(26).FontSize(11).Tooltip(tip).Disabled(disabled)
			if action == markdown.TableRowInsertAfter {
				button.Bind(&v.tableActionHandle)
			}
			v.bindTableAction(c, button, tableAction{kind: tableActionRow, row: action}, edit, draft)
		}
	})
}
