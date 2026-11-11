package markdownview

import (
	"errors"
	"fmt"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/ui/editor"
)

var tableAlignmentFaces = [...]string{"默认", "左", "中", "右"}
var tableAlignmentNames = [...]string{"默认", "左", "居中", "右"}

func tableAlignmentName(column int, alignment markdown.TableAlignment) string {
	return fmt.Sprintf("第%d列%s对齐", column+1, tableAlignmentNames[alignment])
}

func planTableAlignmentAction(source string, plan tableCellPlan, alignment markdown.TableAlignment) (tableActionPatch, error) {
	patch, err := markdown.PlanTableAlignmentEdit(source, plan.tableRange, plan.column, alignment)
	if err != nil {
		switch {
		case errors.Is(err, markdown.ErrTableAlignmentLimit):
			return tableActionPatch{}, errors.New("操作会超过表格或文档上限；可编辑完整表格源码")
		case errors.Is(err, markdown.ErrTableAlignmentContext):
			return tableActionPatch{}, errors.New("操作会改变其他内容的解析状态；可编辑完整表格源码")
		default:
			return tableActionPatch{}, errTableActionStale
		}
	}
	return tableActionPatch{edits: []tableActionEdit{{rng: patch.Range, value: patch.Text}}, noop: patch.Noop}, nil
}

// Painting reads only the captured bounded descriptor; gestures revalidate its
// authoritative source. An old version is labeled pending instead of reparsed.
func (v *View) tableAlignmentPanelTitle(edit *tableCellEdit) string {
	target := edit.target
	current := "待重验"
	if target.buffer == v.host.Buffer && target.buffer != nil && target.version == target.buffer.Version() &&
		target.column >= 0 && target.column < target.columns && target.alignments[target.column] <= markdown.TableAlignRight {
		current = tableAlignmentNames[target.alignments[target.column]]
	}
	row := fmt.Sprintf("数据第%d行", target.row+1)
	if target.row == 0 {
		row = "表头"
	}
	return fmt.Sprintf("%s · 第%d列 · 当前：%s", row, target.column+1, current)
}

func (v *View) drawTableAlignmentButtons(c *ui.Context, edit *tableCellEdit, draft *editor.Editor) {
	ui.Row(c.Key("table-alignment-actions")).Gap(4).Children(func() {
		for value, face := range tableAlignmentFaces {
			alignment := markdown.TableAlignment(value)
			name := tableAlignmentName(edit.target.column, alignment)
			tip := name + "；先写回并完成单元格，再修改整列。相同对齐只完成编辑，不增加对齐撤销。源码选区落在分隔标记内时移到标记起点。"
			button := ui.Button(c.Key(value), face).Label(name).Width(46).Height(26).Padding(0, 4).
				FontSize(11).Tooltip(tip).Disabled(v.host.ReadOnly)
			v.bindTableAction(c, button, tableAction{kind: tableActionAlignment, alignment: alignment}, edit, draft)
		}
	})
}
