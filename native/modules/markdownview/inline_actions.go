package markdownview

import (
	"errors"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/ui/editor"
)

var errInlineStale = errors.New("编辑位置已变化，请重新选择文字后操作")
var errInlineComposing = errors.New("请先确认输入法文字，再设置行内格式")

var inlineActions = []struct {
	kind  markdown.InlineKind
	label string
	key   ui.Key
	tip   string
}{
	{markdown.InlineBold, "粗体", ui.KeyB, "粗体 / 取消粗体 · Ctrl+B"},
	{markdown.InlineItalic, "斜体", ui.KeyI, "斜体 / 取消斜体 · Ctrl+I"},
	{markdown.InlineCode, "行内代码", ui.KeyE, "行内代码 / 取消代码 · Ctrl+E"},
}

type inlineActionGesture struct {
	kind                                   markdown.InlineKind
	draft                                  *editor.Editor
	buffer, draftBuffer                    *text.Buffer
	version, draftVersion, epoch, released uint64
	selection                              text.Range
	active, up                             bool
}

func (v *View) inlineDraftEligible() bool {
	if v.draft == nil || v.cellEdit != nil || v.fencedDraft != nil || v.editing < 0 || v.editing >= len(v.doc.Blocks) {
		return false
	}
	switch v.doc.Blocks[v.editing].Kind {
	case markdown.BlockHeading, markdown.BlockParagraph, markdown.BlockQuote:
		return true
	}
	return false
}

func (v *View) captureInlineAction(kind markdown.InlineKind, draft *editor.Editor) inlineActionGesture {
	g := inlineActionGesture{kind: kind, draft: draft, buffer: v.host.Buffer,
		epoch: v.tableActionEpoch, released: v.tableActionReleased}
	if !v.inlineDraftEligible() || draft != v.draft || draft == nil || g.buffer == nil {
		return g
	}
	g.active = true
	g.draftBuffer, g.version = draft.Buffer, g.buffer.Version()
	g.draftVersion, g.selection = draft.Buffer.Version(), draft.Buffer.Selection()
	return g
}

func (v *View) inlineActionCurrent(g inlineActionGesture) bool {
	return g.active && v.inlineDraftEligible() && v.draft == g.draft && v.host.Buffer == g.buffer &&
		g.draft.Buffer == g.draftBuffer && g.buffer.Version() == g.version &&
		g.draftBuffer.Version() == g.draftVersion && g.draftBuffer.Selection() == g.selection &&
		g.epoch == v.tableActionEpoch && g.released == v.tableActionReleased
}

func (v *View) activateInlineAction(g inlineActionGesture) (err error) {
	defer func() {
		// A callback may replace/release the view. Do not put an error or focus
		// into that new owner, nor retain the old editor through the panel.
		if v.draft == g.draft && v.tableActionReleased == g.released {
			v.inlineActionDraft, v.inlineActionError = g.draft, err
			if err != nil {
				v.report(err)
			}
		}
	}()
	if !v.inlineActionCurrent(g) {
		return errInlineStale
	}
	if v.host.ReadOnly || g.draft.ReadOnly {
		return errDraftReadOnly
	}
	if _, composing := g.draft.MarkedRange(); composing {
		return errInlineComposing
	}
	if g.buffer.InGroup() || g.draftBuffer.InGroup() {
		return text.ErrGroup
	}
	if err := g.draft.Flush(); err != nil {
		return err
	}
	if !v.inlineActionCurrent(g) {
		return errInlineStale
	}
	if ok, err := v.rebaseDraft(); err != nil {
		return err
	} else if !ok {
		return errInlineStale
	}
	if v.pendingDraft() {
		return errors.New("草稿尚未写回；请先完成编辑或解决当前草稿问题，再设置格式")
	}
	source := g.draftBuffer.String()
	left, err := g.draftBuffer.SliceUTF16(0, g.selection.Start())
	if err != nil {
		return err
	}
	selected, err := g.draftBuffer.SliceUTF16(g.selection.Start(), g.selection.End())
	if err != nil {
		return err
	}
	plan, err := markdown.PlanInlineEdit(source, markdown.SourceRange{Start: len(left), End: len(left) + len(selected)}, g.kind)
	if err != nil {
		return err
	}
	candidate := source[:plan.Range.Start] + plan.Text + source[plan.Range.End:]
	start, end := text.UTF16Len(source[:plan.Range.Start]), text.UTF16Len(source[:plan.Range.End])
	if err := g.draftBuffer.CheckReplaceUTF16(start, end, plan.Text); err != nil {
		return err
	}
	// Admission only: the existing draft OnChange/Flush seam is the sole host writer.
	if err := g.buffer.CheckReplaceUTF16(v.editSpan.Start(), v.editSpan.End(), candidate); err != nil {
		return err
	}
	anchor, head := text.UTF16Len(candidate[:plan.Selection.Start]), text.UTF16Len(candidate[:plan.Selection.End])
	if g.selection.Anchor > g.selection.Head {
		anchor, head = head, anchor
	}
	if err := g.draftBuffer.BeginGroup(); err != nil {
		return err
	}
	// Finish the local transaction (including caret) before syncing once. The
	// local group never crosses native input events or a host Changed callback.
	onChange := g.draft.OnChange
	g.draft.OnChange = nil
	r := ui.TextInputRange{Start: start, End: end}
	g.draft.ReplaceText(&r, plan.Text)
	g.draft.OnChange = onChange
	if g.draftBuffer.Version() == g.draftVersion {
		g.draftBuffer.CancelGroup()
		return errors.New("格式操作未写入；当前草稿仍保留")
	}
	g.draft.Reveal(anchor, head)
	g.draftBuffer.EndGroup()
	if onChange != nil {
		onChange()
	}
	if v.draft == g.draft && v.host.Buffer == g.buffer {
		return v.draftError
	}
	return nil
}

func (v *View) bindInlineDraftKeys(c *ui.Context, box ui.Element, draft *editor.Editor) {
	if !v.inlineDraftEligible() {
		return
	}
	box.ObserveInput(func(event ui.InputEvent) {
		if event.Kind != ui.InputKeyDown || event.Mods != ui.Cmd {
			return
		}
		for _, action := range inlineActions {
			if event.Key == action.key && !event.Repeat {
				v.inlineActionPress = v.captureInlineAction(action.kind, draft)
				v.inlineActionPress.up = true
			}
		}
	})
	for _, action := range inlineActions {
		if box.Shortcut(ui.Cmd, action.key) {
			g := v.inlineActionPress
			v.inlineActionPress = inlineActionGesture{}
			if !c.ShortcutRepeated() && g.active && g.draft == draft && g.kind == action.kind {
				_ = v.activateInlineAction(g)
				c.Invalidate()
			}
		}
	}
}

func (v *View) drawInlineActions(c *ui.Context) {
	if v.inlineActionDraft != v.draft {
		v.inlineActionDraft, v.inlineActionError = v.draft, nil
	}
	if !v.inlineDraftEligible() {
		return
	}
	draft := v.draft
	for _, action := range inlineActions {
		button := ui.Button(c.Key(string(action.kind)), action.label).Height(26).Padding(4, 8).KeepFocus().Tooltip(action.tip).
			Disabled(v.host.ReadOnly || draft.ReadOnly)
		button.HandleInput(func(event ui.InputEvent) bool {
			if event.Kind == ui.InputPointerDown && event.Button == 0 && event.Mods == 0 {
				v.inlineActionPress = inlineActionGesture{}
				if event.Clicks > 1 || v.tableActionPointerContinuation {
					return true
				}
				v.inlineActionPress = v.captureInlineAction(action.kind, draft)
			} else if event.Kind == ui.InputPointerUp && event.Button == 0 {
				if event.Canceled {
					v.inlineActionPress = inlineActionGesture{}
				} else {
					v.inlineActionPress.up = true
				}
			} else if event.Kind == ui.InputKeyDown && event.Mods == 0 && (event.Key == ui.KeyEnter || event.Key == ui.KeySpace) {
				if event.Repeat {
					return true
				}
				v.inlineActionPress = v.captureInlineAction(action.kind, draft)
				v.inlineActionPress.up = true
			}
			return false
		})
		if button.Clicked() {
			g := v.inlineActionPress
			v.inlineActionPress = inlineActionGesture{}
			if g.active && g.draft == draft && g.kind == action.kind {
				_ = v.activateInlineAction(g)
				c.Invalidate()
			}
		}
	}
}
