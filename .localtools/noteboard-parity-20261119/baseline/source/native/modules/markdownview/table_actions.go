package markdownview

import (
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/ui/editor"
)

const tableActionHint = "F6 表格操作 · 先写回单元格；两步可分别撤销。"
const tableActionComposingHint = "请先确认输入法组合，再按 F6 进入表格操作。"
const tableHeaderHint = "表头不能删除；请选中数据行。"

var errTableActionStale = errors.New("单元格或源码已变化，请重新选择后操作")

type tableActionKind uint8

const (
	tableActionRow tableActionKind = iota
	tableActionAlignment
	tableActionColumn
)

type tableAction struct {
	kind      tableActionKind
	row       markdown.TableRowAction
	alignment markdown.TableAlignment
	column    markdown.TableColumnAction
}

func (a tableAction) refusal() string {
	if a.kind == tableActionAlignment {
		return "对齐未修改"
	}
	if a.kind == tableActionColumn {
		return "列未修改"
	}
	return "行未修改"
}

func (a tableAction) failure(err error, written bool) error {
	if err == nil {
		return nil
	}
	if written {
		return fmt.Errorf("单元格已写回，%s：%w", a.refusal(), err)
	}
	return fmt.Errorf("%s：%w", a.refusal(), err)
}

type tableActionGesture struct {
	action                                 tableAction
	edit                                   *tableCellEdit
	draft                                  *editor.Editor
	draftBuffer, buffer                    *text.Buffer
	draftVersion, version, epoch, released uint64
	draftSelection, selection              text.Range
	column                                 int
	alignment                              markdown.TableAlignment
	active, up                             bool
	err                                    error
}

type tableActionPatch struct {
	edits []tableActionEdit
	noop  bool
}

type tableActionEdit struct {
	rng   markdown.SourceRange
	value string
}

type tableActionUTF16Edit struct {
	start, end int
	value      string
}

type tableActionReceipt struct {
	completed bool
	version   uint64
	selection text.Range
}

func (v *View) captureTableAction(action tableAction, edit *tableCellEdit, draft *editor.Editor) tableActionGesture {
	g := tableActionGesture{action: action, edit: edit, draft: draft, buffer: v.host.Buffer,
		active: true, epoch: v.tableActionEpoch, released: v.tableActionReleased}
	if edit == nil || draft == nil || v.cellEdit != edit || v.draft != draft || g.buffer == nil ||
		v.host.ReadOnly || edit.target.column < 0 || edit.target.column >= edit.target.columns {
		g.err = errTableActionStale
		return g
	}
	g.column, g.alignment = edit.target.column, edit.target.alignments[edit.target.column]
	g.draftBuffer = draft.Buffer
	g.draftVersion, g.version = draft.Buffer.Version(), g.buffer.Version()
	g.draftSelection, g.selection = draft.Buffer.Selection(), g.buffer.Selection()
	_, g.err = v.currentTableCellContext(edit.target)
	return g
}

func (v *View) tableActionGestureCurrent(g tableActionGesture) bool {
	return g.active && g.err == nil && g.edit != nil && g.draft != nil &&
		v.cellEdit == g.edit && v.draft == g.draft && v.host.Buffer == g.buffer &&
		v.tableActionEpoch == g.epoch && v.tableActionReleased == g.released &&
		!v.host.ReadOnly && g.buffer != nil && !g.buffer.InGroup() &&
		g.buffer.Version() == g.version && g.buffer.Selection() == g.selection &&
		g.draft.Buffer == g.draftBuffer && g.draftBuffer.Version() == g.draftVersion &&
		g.draftBuffer.Selection() == g.draftSelection && g.edit.target.column == g.column &&
		g.column >= 0 && g.column < g.edit.target.columns && g.edit.target.alignments[g.column] == g.alignment
}

func (v *View) activateTableAction(g tableActionGesture) (err error) {
	written := false
	receipt := tableActionReceipt{version: g.version, selection: g.selection}
	defer func() {
		err = g.action.failure(err, written)
		v.finishTableAction(g, err, receipt)
	}()
	if g.err != nil {
		return g.err
	}
	if !v.tableActionGestureCurrent(g) {
		return errTableActionStale
	}
	if _, err := v.currentTableCellContext(g.edit.target); err != nil {
		return err
	}
	// Clear this owner's old action error before any callback can replace it.
	v.tableActionError = nil
	if err := g.draft.Flush(); err != nil {
		v.setDraftError(err)
		return err
	}
	if !v.tableActionGestureCurrent(g) {
		return errTableActionStale
	}
	plan, err := v.prepareTableCellDraft()
	if err != nil {
		v.setDraftError(err)
		return err
	}
	expectedVersion, expectedSelection := g.version, g.selection
	if plan.changed {
		expectedVersion++
		caret := plan.cellSpan.Start() + text.UTF16Len(plan.value)
		expectedSelection = text.Range{Anchor: caret, Head: caret}
	}
	if err := v.Flush(); err != nil {
		return err
	}
	receipt = tableActionReceipt{completed: true, version: expectedVersion, selection: expectedSelection}
	written = plan.changed
	// Exactly our normal Flush clears the panel once. A Changed callback may
	// release, replace the source/draft, open a group, edit or move selection.
	if v.host.Buffer != g.buffer || v.tableActionReleased != g.released || v.tableActionEpoch != g.epoch+1 ||
		v.draft != nil || v.cellEdit != nil || v.host.ReadOnly || g.buffer.InGroup() ||
		g.buffer.Version() != expectedVersion || g.buffer.Selection() != expectedSelection || plan.column != g.column {
		return errTableActionStale
	}
	if g.buffer.LenBytes() > markdown.MaxParseBytes {
		return errTableCellCurrentTooLarge
	}
	source := g.buffer.String()
	if plan.tableRange.Start < 0 || plan.tableRange.End > len(source) ||
		source[plan.tableRange.Start:plan.tableRange.End] != plan.tableSource {
		return errTableActionStale
	}
	var patch tableActionPatch
	switch g.action.kind {
	case tableActionAlignment:
		patch, err = planTableAlignmentAction(source, plan, g.action.alignment)
	case tableActionColumn:
		patch, err = planTableColumnAction(source, plan, g.action.column)
	default:
		patch, err = planTableRowAction(source, plan, g.action.row)
	}
	if err != nil {
		return err
	}
	if patch.noop {
		return nil // normal Flush completed the panel; no second undo/Changed
	}
	edits, selection, err := prepareTableActionEdits(source, patch, expectedSelection)
	if err != nil {
		return err
	}
	if v.host.Buffer != g.buffer || v.host.ReadOnly || v.tableActionReleased != g.released ||
		v.tableActionEpoch != g.epoch+1 || v.draft != nil || v.cellEdit != nil ||
		g.buffer.Version() != expectedVersion || g.buffer.Selection() != expectedSelection || g.buffer.InGroup() {
		return errTableActionStale
	}
	if err := applyTableActionEdits(g.buffer, edits, selection); err != nil {
		return err
	}
	v.tableActionPress = tableActionGesture{}
	v.list.HoldPlace()
	v.loaded = false
	if v.host.Changed != nil {
		v.host.Changed()
	}
	return nil // never restore captured UI/selection after Changed
}

// Plans use ascending, nonoverlapping original byte ranges. Convert every range
// before editing, rejecting split UTF-8 and ambiguous same-position edits. The
// planner must merge those itself. Both endpoints use the existing affinity in
// exactly the descending order used by the buffer transaction.
func prepareTableActionEdits(source string, patch tableActionPatch, selection text.Range) ([]tableActionUTF16Edit, text.Range, error) {
	if !utf8.ValidString(source) || len(patch.edits) == 0 {
		return nil, selection, errTableActionStale
	}
	edits := make([]tableActionUTF16Edit, len(patch.edits))
	byteCursor, units := 0, 0
	for i, edit := range patch.edits {
		r := edit.rng
		if r.Start < byteCursor || r.End < r.Start || r.End > len(source) ||
			i > 0 && r.Start == patch.edits[i-1].rng.Start || !utf8.ValidString(edit.value) ||
			r.Start < len(source) && !utf8.RuneStart(source[r.Start]) ||
			r.End < len(source) && !utf8.RuneStart(source[r.End]) {
			return nil, selection, errTableActionStale
		}
		start := units + text.UTF16Len(source[byteCursor:r.Start])
		end := start + text.UTF16Len(source[r.Start:r.End])
		edits[i] = tableActionUTF16Edit{start: start, end: end, value: edit.value}
		byteCursor, units = r.End, end
	}
	for i := len(edits) - 1; i >= 0; i-- {
		edit := edits[i]
		selection = mapTableActionSelection(selection, edit.start, edit.end, text.UTF16Len(edit.value))
	}
	return edits, selection, nil
}

func applyTableActionEdits(buffer *text.Buffer, edits []tableActionUTF16Edit, selection text.Range) error {
	if err := buffer.BeginGroup(); err != nil {
		return err // a foreign group stays owned by its caller
	}
	for i := len(edits) - 1; i >= 0; i-- {
		edit := edits[i]
		if err := buffer.ReplaceUTF16(edit.start, edit.end, edit.value); err != nil {
			buffer.CancelGroup() // restores bytes/selection/history, not revision
			return err
		}
	}
	if err := buffer.SetSelection(selection.Anchor, selection.Head); err != nil {
		buffer.CancelGroup()
		return err
	}
	buffer.EndGroup()
	return nil // no external callback may run inside the owned group
}

func mapTableActionSelection(selection text.Range, start, end, inserted int) text.Range {
	mapPoint := func(point int) int {
		if point < start {
			return point
		}
		if point >= end {
			return point + inserted - (end - start)
		}
		return start
	}
	return text.Range{Anchor: mapPoint(selection.Anchor), Head: mapPoint(selection.Head)}
}

// A real refusal is still reported when the old view was released/replaced, but
// its local error must not overwrite replacement UI. No writes follow Report.
func (v *View) finishTableAction(g tableActionGesture, err error, receipt tableActionReceipt) {
	if err == nil {
		return
	}
	owned := v.host.Buffer == g.buffer && g.buffer != nil && v.tableActionReleased == g.released &&
		g.buffer.Version() == receipt.version && g.buffer.Selection() == receipt.selection &&
		(v.tableActionEpoch == g.epoch && v.draft == g.draft && v.cellEdit == g.edit &&
			g.draft != nil && g.draft.Buffer == g.draftBuffer && g.draftBuffer.Version() == g.draftVersion &&
			g.draftBuffer.Selection() == g.draftSelection ||
			receipt.completed && v.tableActionEpoch == g.epoch+1 && v.draft == nil && v.cellEdit == nil)
	previous := v.tableActionError
	if owned {
		v.tableActionError, v.tableActionErrorKind = err, g.action.kind
	}
	if !owned || previous == nil || previous.Error() != err.Error() {
		v.report(err)
	}
}

func (v *View) observeTableActionInput(event ui.InputEvent) {
	if event.Kind == ui.InputPointerDown {
		// Ancestor raw click count precedes MyGo's count reset for a newly hit
		// widget. A continuation cannot click through a newly opened panel.
		v.tableActionPointerContinuation = event.Clicks > 1
	} else if event.Kind == ui.InputPointerUp || event.Kind == ui.InputKeyDown {
		v.tableActionPointerContinuation = false
	}
	if event.Kind == ui.InputPointerDown || event.Kind == ui.InputScroll || event.Kind == ui.InputZoom ||
		event.Kind == ui.InputPointerUp && event.Canceled ||
		event.Kind == ui.InputKeyDown && (event.Mods != 0 || event.Key != ui.KeyEnter && event.Key != ui.KeySpace) {
		v.tableActionPress = tableActionGesture{}
	}
}

func (v *View) handleTableActionNavigation(event ui.InputEvent) bool {
	if event.Kind == ui.InputScroll || event.Kind == ui.InputZoom {
		v.tableActionPress = tableActionGesture{}
		v.tableActionPointerContinuation = false
	}
	return false // allow the native scroll container to consume it
}

func (v *View) bindTableAction(c *ui.Context, button ui.Element, action tableAction, edit *tableCellEdit, draft *editor.Editor) {
	button.HandleInput(func(event ui.InputEvent) bool {
		v.handleTableActionNavigation(event)
		if event.Kind == ui.InputKeyDown && event.Mods == 0 && (event.Key == ui.KeyEnter || event.Key == ui.KeySpace) {
			if event.Repeat {
				return true
			}
			v.tableActionPress = v.captureTableAction(action, edit, draft)
			v.tableActionPress.up = true
		} else if event.Kind == ui.InputPointerDown && event.Button == 0 && event.Mods == 0 {
			if v.tableActionPointerContinuation || event.Clicks > 1 {
				v.tableActionPress = tableActionGesture{}
				return true
			}
			v.tableActionPress = v.captureTableAction(action, edit, draft)
		} else if event.Kind == ui.InputPointerUp && event.Button == 0 {
			if event.Canceled {
				v.tableActionPress = tableActionGesture{}
			} else {
				v.tableActionPress.up = true
			}
		}
		return false
	})
	if button.Clicked() {
		captured := v.tableActionPress
		v.tableActionPress = tableActionGesture{}
		if captured.active && captured.action == action {
			_ = v.activateTableAction(captured)
			c.Invalidate()
		}
	}
}

func (v *View) drawTableActions(c *ui.Context) {
	edit, draft := v.cellEdit, v.draft
	if edit == nil || draft == nil {
		return
	}
	hint := tableActionHint
	if edit.target.row == 0 {
		hint = "F6 表格操作 · " + tableHeaderHint
	}
	if _, composing := draft.MarkedRange(); composing {
		hint = tableActionComposingHint
	}
	ui.Text(c, hint).Height(30).FontSize(11).MaxLines(2).Tooltip(hint).TextColor(c.Theme().TextMuted)
	v.drawTableRowButtons(c, edit, draft)
	v.drawTableColumnButtons(c, edit, draft)
	v.drawTableAlignmentButtons(c, edit, draft)
}
