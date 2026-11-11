package graphview

import (
	"errors"
	"fmt"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/ui/editor"
)

func (v *view) observeGraphInspectorInput(ev ui.InputEvent) {
	v.observeMarkerInput(ev)
	if ev.Kind == ui.InputPointerDown {
		// Keep the raw ancestor count: a newly exposed child may receive a
		// reset click count from the framework after a panel layout change.
		v.tagPointerContinuation = ev.Clicks > 1
	} else if ev.Kind == ui.InputPointerUp || ev.Kind == ui.InputKeyDown {
		v.tagPointerContinuation = false
	}
	if ev.Kind == ui.InputPointerDown || ev.Kind == ui.InputPointerUp && ev.Canceled ||
		ev.Kind == ui.InputKeyDown && (ev.Mods != 0 || ev.Key != ui.KeyEnter && ev.Key != ui.KeySpace) {
		v.invalidateTagActions()
	}
}

func (v *view) graphInspectorScrollInput(ev ui.InputEvent) bool {
	v.markerScrollInput(ev)
	if ev.Kind == ui.InputScroll || ev.Kind == ui.InputZoom {
		v.invalidateTagActions()
		v.tagPointerContinuation = false
	}
	return false
}

// Only focus moves; no text is committed by the shortcut. Native composition
// must be confirmed first because losing editor focus would unmark it.
func (v *view) focusTagControl(c *ui.Context, input *editor.Editor, target *ui.Handle) {
	if _, composing := input.MarkedRange(); composing {
		// Transient focus guidance belongs to this input and Warnings, not
		// the host's persistent failure/status history. Confirmation clears
		// it locally; validation and mutation failures still use Report.
		v.tagError = errTagConfirm
		if v.draft != nil && input == v.draft.input {
			v.draftError = errTagConfirm
		}
		c.Invalidate()
		return
	}
	target.Focus()
}

func tagActionLabel(action int) string {
	switch action {
	case tagAdd:
		return "添加标签"
	case tagClear:
		return "清空全部标签"
	case tagDiscard:
		return "放弃标签输入"
	case tagRecover:
		return "恢复标签及节点草稿"
	default:
		return fmt.Sprintf("移除第%d个标签", action+1)
	}
}

func (v *view) drawTagButton(c *ui.Context, info tagPresentation, action int, disabled, compactRecovery bool) {
	label := tagActionLabel(action)
	face := label
	if action >= 0 {
		face = "移除"
	}
	if action == tagDiscard {
		face = "放弃输入"
	}
	if action == tagRecover {
		face = "恢复输入为新文档"
		if compactRecovery {
			label, face = "恢复节点及标签草稿", "恢复草稿"
		}
	}
	button := ui.Button(c.Key(action), face).Label(label).Height(28).FontSize(11).Disabled(disabled)
	if action == tagAdd {
		button.Bind(&v.tagAddHandle).Grow(1)
	}
	if action == tagClear {
		button.FillWidth().Tooltip("移除全部标签，包括对象标签，并写入空列表 []。")
	}
	if action == tagDiscard {
		button.Grow(1)
	}
	if action == tagRecover {
		if compactRecovery {
			button.Grow(1).Tooltip("将节点字段和未添加的标签输入恢复为新文档。")
		} else {
			button.FillWidth()
		}
	}
	capture := func() tagGesture {
		if action == tagDiscard || action == tagRecover {
			g := tagGesture{active: true, buffer: v.host.Buffer, nodeID: v.selectedID, epoch: v.tagEpoch, action: action, open: v.tagOpen, input: v.tagInputIdentity(), field: v.fieldIdentity()}
			if g.buffer != nil {
				g.version, g.selection = g.buffer.Version(), g.buffer.Selection()
			}
			return g
		}
		return v.captureTag(action, info)
	}
	// info contains only bounded presentation; no parsed node crosses input.
	button.HandleInput(func(ev ui.InputEvent) bool {
		v.graphInspectorScrollInput(ev)
		if ev.Kind == ui.InputKeyDown && ev.Mods == 0 && (ev.Key == ui.KeyEnter || ev.Key == ui.KeySpace) {
			if ev.Repeat {
				return true
			}
			v.tagPress = capture()
			v.tagPress.released = true
		} else if ev.Kind == ui.InputPointerDown && ev.Button == 0 && ev.Mods == 0 {
			if v.tagPointerContinuation || ev.Clicks > 1 {
				v.tagPress = tagGesture{}
				return true
			}
			v.tagPress = capture()
		} else if ev.Kind == ui.InputPointerUp {
			if ev.Canceled {
				v.tagPress = tagGesture{}
			} else {
				v.tagPress.released = true
			}
		}
		return false
	})
	if button.Clicked() {
		g := v.tagPress
		v.tagPress = tagGesture{}
		if g.active && g.action == action {
			if action == tagDiscard || action == tagRecover {
				if !v.sameTagGesture(g) || g.buffer.Version() != g.version || g.buffer.Selection() != g.selection {
					v.setTagError(errTagStale)
				} else if action == tagDiscard {
					v.clearTagDraft()
					v.setTagError(nil)
					v.clearTagPendingWarning()
					v.refreshTagDraft()
				} else {
					v.setTagError(v.RecoverDraft())
				}
			} else {
				v.setTagError(v.activateTag(g))
			}
			c.Invalidate()
		}
	}
}

func (v *view) drawTags(c *ui.Context) {
	info := v.tagPresentation()
	theme := c.Theme()
	ui.Column(c).Key("mindmap-tags").FillWidth().Shrink(0).Gap(6).HandleInput(v.graphInspectorScrollInput).Children(func() {
		label, face := "展开标签面板", "▸ 标签"
		if v.tagOpen {
			label, face = "收起标签面板", "▾ 标签"
		}
		if ui.Button(c.Key("tag-toggle"), face).Label(label).FillWidth().Height(30).Bind(&v.tagHandle).Expanded(v.tagOpen).Clicked() {
			v.tagOpen = !v.tagOpen
			v.invalidateTagActions()
			v.refreshTagDraft()
			c.Invalidate()
		}
		ui.Text(c, info.summary).Label("当前节点标签摘要").MaxLines(2).Ellipsis("…").Tooltip(info.summary).FontSize(11).TextColor(theme.TextMuted)
		if v.pendingTagDraft() && !v.tagOpen {
			ui.Text(c, "有尚未添加的输入；展开后添加、放弃或恢复。").FontSize(11).TextColor(theme.Warning)
		}
		if !v.tagOpen {
			return
		}
		v.refreshTagDraft()
		reason := info.err
		if reason == nil && v.host.ReadOnly {
			reason = errors.New("当前文档只读，不能修改标签")
		}
		if reason == nil && v.host.Buffer != nil && v.host.Buffer.InGroup() {
			reason = errors.New("其他文本事务正在进行，暂时不能修改标签")
		}
		if reason != nil {
			ui.Text(c, reason.Error()).FontSize(11).MaxLines(3).Tooltip(reason.Error()).TextColor(theme.Warning)
		}
		if v.tagError != nil {
			ui.Text(c, markerText(v.tagError.Error(), 512)).Label("标签操作问题").MaxLines(2).Ellipsis("…").Tooltip(v.tagError.Error()).FontSize(11).TextColor(theme.Warning)
		}
		if d := v.tagDraft; d != nil {
			d.input.ReadOnly = reason != nil && !v.pendingTagDraft()
			box := ui.Column(c.Key("tag-input")).Height(48).FillWidth().Border(1, theme.Border)
			if box.Shortcut(0, ui.KeyF6) || box.Shortcut(0, ui.KeyTab) {
				v.focusTagControl(c, d.input, &v.tagAddHandle)
			}
			if box.Shortcut(ui.Shift, ui.KeyTab) {
				v.focusTagControl(c, d.input, &v.tagHandle)
			}
			box.Children(func() { d.input.View(c) })
		}
		hint := "F6 进入标签操作；组合输入请先确认。"
		if v.tagDraft != nil {
			if _, composing := v.tagDraft.input.MarkedRange(); composing {
				hint = "请先确认正在输入的文字，再移动到属性操作。"
			}
		}
		ui.Text(c, hint).Height(30).MaxLines(2).FontSize(11).TextColor(theme.TextMuted)
		ui.Row(c).FillWidth().Shrink(0).Gap(4).Children(func() {
			v.drawTagButton(c, info, tagAdd, reason != nil || !v.pendingTagDraft(), false)
			v.drawTagButton(c, info, tagDiscard, !v.pendingTagDraft(), false)
		})
		v.drawTagButton(c, info, tagClear, reason != nil || info.count == 0 || v.pendingTagDraft(), false)
		if v.pendingTagDraft() {
			v.drawTagButton(c, info, tagRecover, false, false)
		}
		for i, row := range info.rows {
			ui.Row(c.Key(i)).FillWidth().Shrink(0).Gap(4).AlignItems(ui.Center).Children(func() {
				ui.Text(c, row).Label(fmt.Sprintf("第%d个标签", i+1)).Grow(1).MinWidth(0).MaxLines(2).Ellipsis("…").Tooltip(row).FontSize(11)
				v.drawTagButton(c, info, i, reason != nil || v.pendingTagDraft(), false)
			})
		}
	})
}
