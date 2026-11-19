package markdownview

import (
	"errors"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

var errHistoryReadOnly = errors.New("文档当前只读，无法撤销或重做")
var errHistoryPending = errors.New("草稿尚未写回；请先完成或放弃草稿，再撤销或重做文档")

// Keep the public Handle, but detach old geometry without changing a ListState
// that MyGo may still be measuring in this pass. Release/cache-drop cancellation
// prevents an obsolete queued focus request surviving omission/rebinding.
func (v *View) resetPreviewList() {
	v.list.Handle.CancelFocus()
	v.list = &ui.ListState{Handle: v.list.Handle}
}

// Two small explicit controls keep document history reachable for plain/empty
// previews with no link, checkbox or cell to Tab to. A pending draft disables
// them before pointer blur can commit IME; preview keyboard refusals stay visible.
func (v *View) drawHistoryControls(c *ui.Context, target historyTarget) {
	if target.buffer == nil {
		return
	}
	target.controls = true
	history := target.buffer.History()
	blocked := v.host.ReadOnly || v.pendingDraft() || target.buffer.InGroup()
	tip := "文档历史：Ctrl/Cmd+Z 撤销；Ctrl/Cmd+Y 或 Ctrl/Cmd+Shift+Z 重做"
	if v.pendingDraft() {
		tip = errHistoryPending.Error()
	} else if v.host.ReadOnly {
		tip = errHistoryReadOnly.Error()
	}
	ui.Row(c.Key("document-history")).Bind(&v.historyControls).Wrap().Gap(6).Padding(0, 18).Shrink(0).
		ObserveInput(func(event ui.InputEvent) { v.historyCommand(target, event) }).Children(func() {
		if ui.Button(c, "撤销文档").Height(26).Disabled(blocked || history.UndoDepth == 0).Tooltip(tip).Clicked() {
			v.changeHistory(target, false)
		}
		if ui.Button(c, "重做文档").Height(26).Disabled(blocked || history.RedoDepth == 0).Tooltip(tip).Clicked() {
			v.changeHistory(target, true)
		}
	})
	v.historyControls.OnShortcut(c, ui.Cmd, ui.KeyZ, func() { v.changeHistory(target, false) })
	v.historyControls.OnShortcut(c, ui.Cmd, ui.KeyY, func() { v.changeHistory(target, true) })
	v.historyControls.OnShortcut(c, ui.Cmd|ui.Shift, ui.KeyZ, func() { v.changeHistory(target, true) })
}

// A deferred shortcut owns only this preview's numeric receipt, not source or
// parsed rows. Flush and host callbacks may synchronously replace its owner.
type historyTarget struct {
	buffer    *text.Buffer
	version   uint64
	selection text.Range
	list      *ui.ListState
	released  uint64
	controls  bool
}

func (v *View) historyTarget() historyTarget {
	target := historyTarget{buffer: v.host.Buffer, list: v.list, released: v.historyReleased}
	if target.buffer != nil {
		target.version, target.selection = target.buffer.Version(), target.buffer.Selection()
	}
	return target
}

func (v *View) historyCurrent(target historyTarget) bool {
	return target.buffer != nil && v.host.Buffer == target.buffer && v.list == target.list &&
		v.historyReleased == target.released && target.buffer.Version() == target.version &&
		target.buffer.Selection() == target.selection
}

func (v *View) refuseHistory(err error) {
	v.historyError = err
	v.report(err)
}

func (v *View) historyCommand(target historyTarget, event ui.InputEvent) bool {
	if event.Kind != ui.InputCommand || event.Text != "undo" && event.Text != "redo" {
		return false
	}
	v.changeHistory(target, event.Text == "redo")
	return true
}

// The independent draft editor consumes its own keys first. Preview history
// refuses a pending draft before Flush; it never commits one merely to undo it.
// Rows remain immutable through this pass, and focus is returned only to the
// still-current preview after Changed, keeping consecutive undo/redo reachable.
func (v *View) changeHistory(target historyTarget, redo bool) {
	if !v.historyCurrent(target) {
		return
	}
	if v.host.ReadOnly {
		v.refuseHistory(errHistoryReadOnly)
		return
	}
	if v.pendingDraft() {
		v.refuseHistory(errHistoryPending)
		return
	}
	if err := v.Flush(); err != nil {
		// Flush already reports the draft failure. Keep it visible even if the
		// draft panel later becomes unavailable; do not report it twice.
		if v.historyCurrent(target) {
			v.historyError = err
		}
		return
	}
	if !v.historyCurrent(target) {
		return
	}
	if v.host.ReadOnly {
		v.refuseHistory(errHistoryReadOnly)
		return
	}
	if v.pendingDraft() {
		v.refuseHistory(errHistoryPending)
		return
	}
	if target.buffer.InGroup() {
		v.refuseHistory(text.ErrGroup)
		return
	}
	changed := false
	if redo {
		changed = target.buffer.Redo()
	} else {
		changed = target.buffer.Undo()
	}
	if !changed {
		return // Empty history is an ordinary quiet no-op.
	}
	target.version, target.selection = target.buffer.Version(), target.buffer.Selection()
	v.historyError = nil
	v.loaded = false
	if v.host.Changed != nil {
		v.host.Changed()
	}
	if v.historyCurrent(target) && !v.host.ReadOnly && !v.pendingDraft() {
		if target.controls {
			v.historyControls.Focus()
		} else {
			target.list.Handle.Focus()
		}
	}
}
