package markdownview

import (
	"errors"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/ui/editor"
)

const maxPreviewBlockBytes = 32 << 10

const partialSourceGuidance = "Markdown 内容超过预览处理上限。完整原文仍保留，请切到源码视图查看和编辑；HTML 导出已暂停。"

type View struct {
	host                           editor.Host
	doc                            markdown.Document
	version                        uint64
	docBuffer                      *text.Buffer
	loaded                         bool
	previewError                   error // cached size refusal; never owns source text
	historyError                   error
	historyReleased                uint64
	historyControls                ui.Handle
	editing                        int
	draft                          *editor.Editor
	editBuffer                     *text.Buffer
	editVersion                    uint64
	editSpan                       text.Range // UTF-16 range of the latest synchronized field
	editGuard                      text.Range // enclosing syntax cannot change under the draft
	editBase                       string
	editPanelHeight                float32
	cellEdit                       *tableCellEdit
	draftError                     error
	scroll                         ui.ScrollState
	frameRows                      int // rows built in the most recent pass; resource acceptance
	list                           *ui.ListState
	rows                           []previewRow
	nextRowID                      uint64
	taskActivation                 taskTarget   // numeric identity captured by native input before rebuild
	taskRow                        taskTarget   // exact own-toggle row identity, retained until reparse
	linkModifiers                  ui.Modifiers // ObserveInput updates before native hit testing
	linkActivation                 linkTarget   // one captured gesture; no parser/source retention
	linkReleased                   uint64
	linkPointerUp                  bool
	linkGestureBuilt               bool
	linkPointerContinuation        bool
	referenceActivation            referenceTarget
	referencePointerUpBuffer       *text.Buffer
	referencePointerUpRowID        uint64
	referenceGestureBuiltRowID     uint64
	tableEditActivation            tableEditTarget
	tablePointerUpBuffer           *text.Buffer
	tablePointerUpRowID            uint64
	tableGestureBuiltRowID         uint64
	tableCellActivation            tableCellTarget
	tableCellGestureBuilt          bool
	tableActionPress               tableActionGesture
	tableActionEpoch               uint64
	tableActionReleased            uint64
	tableActionError               error
	tableActionErrorKind           tableActionKind
	tableActionHandle              ui.Handle
	tableActionPointerContinuation bool
	tableViewport                  float32
	tableViewportKnown             bool
	tableWindowWidth               float32
	tableWindowHeight              float32
	tableWindowKnown               bool
	hasLinks                       bool
	hasTables                      bool
	outline                        *outlineSource
	outlineCurrent                 string
	outlineCaret                   int
}

var errDraftConflict = errors.New("正在编辑的块已在源码中改变或无法定位；草稿仍保留，可打开未同步副本")
var errDraftReadOnly = errors.New("文档当前只读，未打开块编辑器；当前草稿仍保留")
var errTableReadOnly = errors.New("文档当前只读，未打开表格源码；当前草稿仍保留")

func Adapter() editor.Adapter {
	return editor.Adapter{ID: "markdown", Name: "Markdown", Matches: func(path string) bool {
		for _, suffix := range []string{".md", ".markdown", ".mdown", ".mkd", ".mkdn", ".mdwn", ".mdtxt", ".mdtext"} {
			if strings.HasSuffix(strings.ToLower(path), suffix) {
				return true
			}
		}
		return false
	}, Create: func(h editor.Host) editor.View { return &View{host: h, editing: -1, list: &ui.ListState{}} }, Outline: newOutlineSource,
		Template: func() string { return "# 新文档\n\n" }, Exports: []string{"html"},
		ExportEngine: &editor.ExportEngine{ID: "levnote.markdown-export", Format: "html"}}
}

func (v *View) refresh() {
	if v.host.Buffer == nil {
		return
	}
	if v.draft != nil {
		// Reconcile external undo/source edits even when no local key arrives.
		// A clean stale editor yields to source, a dirty one remains recoverable.
		if v.editBuffer != v.host.Buffer || v.editVersion != v.host.Buffer.Version() {
			if mapped, err := v.rebaseDraft(); err != nil {
				v.setDraftError(err)
			} else if mapped && !v.pendingDraft() {
				// A clean independent field must not freeze unrelated source
				// changes (for example undoing a task toggle). The parse below
				// keeps that field/range but detaches its old block index.
				v.loaded = false
			}
		}
	}
	// A local field may stay open, but obsolete parsed source must not survive
	// an oversized host buffer merely because that field has a pending draft.
	if v.checkPreviewSize() != nil {
		return
	}
	if v.draft != nil && v.loaded {
		return // keep the rendered block identity stable while editing
	}
	if v.linkActivation.buffer != nil && v.docBuffer == v.host.Buffer &&
		v.version != v.host.Buffer.Version() {
		// Preserve the pressed row only through release; activation revalidates
		// exact source ranges and inline kind after mapping and any draft Flush.
		return
	}
	if v.referenceActivation.buffer != nil && v.docBuffer == v.host.Buffer &&
		v.version != v.host.Buffer.Version() {
		// Keep the pressed reference row alive through the physical release.
		// MapRangeSince validates its source identity before any later click.
		return
	}
	if v.tableEditActivation.buffer != nil && v.docBuffer == v.host.Buffer &&
		v.version != v.host.Buffer.Version() {
		// Table activation uses a bounded source snapshot captured at the
		// physical gesture start. Hold the current rows only through release so
		// that MyGo can deliver the old handler's pointer-up/click consistently.
		return
	}
	if v.loaded && v.docBuffer == v.host.Buffer && v.version == v.host.Buffer.Version() {
		return
	}
	if v.draft != nil {
		// A released/refused preview can be rebuilt without rebinding its
		// independent draft to a new row index. The fixed draft panel remains.
		v.editing = -1
	}
	v.setPreview(markdown.Parse(v.host.Buffer.String()))
	v.version, v.docBuffer, v.loaded = v.host.Buffer.Version(), v.host.Buffer, true
}

func (v *View) Release() {
	v.historyReleased++
	v.historyError = nil
	v.list.Handle.CancelFocus()
	v.historyControls.CancelFocus()
	v.linkReleased++
	v.linkPointerUp, v.linkGestureBuilt, v.linkPointerContinuation = false, false, false
	v.tableActionReleased++
	v.tableActionPress = tableActionGesture{}
	v.tableActionPointerContinuation = false
	v.tableActionError = nil
	v.tableActionHandle.CancelFocus()
	if err := v.Flush(); err != nil {
		v.setDraftError(err)
		v.draft.ReleaseView()
	} else {
		v.clearDraft()
	}
	v.doc = markdown.Document{}
	v.previewError = nil
	v.docBuffer, v.loaded = nil, false
	v.rows = nil
	v.taskActivation, v.taskRow = taskTarget{}, taskTarget{}
	v.linkActivation, v.hasLinks = linkTarget{}, false
	v.referenceActivation = referenceTarget{}
	v.referencePointerUpBuffer, v.referencePointerUpRowID = nil, 0
	v.tableEditActivation = tableEditTarget{}
	v.tablePointerUpBuffer, v.tablePointerUpRowID = nil, 0
	v.tableCellActivation, v.tableCellGestureBuilt = tableCellTarget{}, false
	v.tableViewport, v.tableViewportKnown = 0, false
	v.tableWindowWidth, v.tableWindowHeight, v.tableWindowKnown = 0, 0, false
	v.hasTables = false
	// Navigation may synchronously release us during a List row callback.
	// Swap ownership; the active immutable build keeps its old state pointer
	// until MyGo finishes that pass, rather than zeroing it underneath MyGo.
	v.resetPreviewList()
	if v.outline != nil {
		v.outline.Release()
		v.outline = nil
	}
}

func (v *View) Warnings() []string {
	v.refresh()
	var warnings []string
	if v.previewError != nil {
		warnings = append(warnings, oversizeSourceGuidance)
	} else if v.doc.Partial {
		warnings = append(warnings, partialSourceGuidance)
	}
	for _, warning := range v.doc.Warnings {
		warnings = append(warnings, warning.Message)
	}
	for _, block := range v.doc.Blocks {
		if len(block.Raw) > maxPreviewBlockBytes {
			warnings = append(warnings, "较大的内容块已使用占位显示；双击可分段编辑，完整内容仍保留")
			break
		}
	}
	if v.draftError != nil {
		prefix := "块草稿尚未写回："
		if v.cellEdit != nil {
			prefix = "单元格草稿尚未写回："
		}
		warnings = append(warnings, prefix+v.draftError.Error())
	}
	if v.tableActionError != nil {
		warnings = append(warnings, v.tableActionError.Error())
	}
	if v.historyError != nil {
		warnings = append(warnings, v.historyError.Error())
	}
	return warnings
}

func (v *View) CompositionActive() bool {
	if v.draft == nil {
		return false
	}
	_, composing := v.draft.MarkedRange()
	return composing
}

func (v *View) Export(format string) ([]byte, error) {
	if strings.ToLower(format) != "html" {
		return nil, errors.New("Markdown 支持导出 HTML")
	}
	// Invalidate an obsolete preview even when the draft Flush below fails.
	_ = v.checkPreviewSize()
	if err := v.Flush(); err != nil {
		return nil, err
	}
	if v.host.Buffer == nil {
		return nil, errors.New("没有可导出的文档")
	}
	if err := v.checkPreviewSize(); err != nil {
		return nil, err
	}
	// Export owns a temporary parse. It never replaces the active preview's
	// block identities or the independent draft range/base.
	doc := markdown.Parse(v.host.Buffer.String())
	html, warnings, err := markdown.ExportHTMLChecked(doc)
	for _, warning := range warnings {
		v.report(errors.New(warning.Message))
	}
	if err != nil {
		if errors.Is(err, markdown.ErrPartialDocument) {
			err = fmt.Errorf("%s：%w", partialSourceGuidance, err)
		}
		v.report(err)
		return nil, err
	}
	return []byte(html), nil
}

func (v *View) Draw(c *ui.Context) {
	history := v.historyTarget()
	heldInlineGesture := v.linkActivation.buffer != nil
	defer func() {
		if v.linkPointerUp || v.linkActivation.buffer != nil && !v.linkGestureBuilt {
			v.linkActivation = linkTarget{}
		}
		v.linkPointerUp = false
		if heldInlineGesture && v.linkActivation.buffer == nil && v.loaded && v.host.Buffer != nil &&
			v.docBuffer == v.host.Buffer && v.version != v.host.Buffer.Version() && v.draft == nil {
			c.Invalidate() // settle the one snapshot retained through a reference gesture
		}
		if v.tableActionPress.up {
			v.tableActionPress = tableActionGesture{}
		}
	}()
	if v.tableActionPress.active && (v.tableActionPress.err != nil || !v.tableActionGestureCurrent(v.tableActionPress)) {
		captured := v.tableActionPress
		err := captured.err
		if err == nil {
			err = errTableActionStale
		}
		v.tableActionPress = tableActionGesture{}
		v.finishTableAction(captured, captured.action.failure(err, false),
			tableActionReceipt{version: captured.version, selection: captured.selection})
		if v.tableActionReleased != captured.released || v.host.Buffer != captured.buffer {
			return
		}
	}
	v.frameRows = 0
	v.linkGestureBuilt = false
	v.referenceGestureBuiltRowID = 0
	v.tableGestureBuiltRowID = 0
	v.tableCellGestureBuilt = false
	windowWidth, windowHeight := c.Size()
	if !v.tableWindowKnown || windowWidth != v.tableWindowWidth || windowHeight != v.tableWindowHeight {
		// A changed window or an unknown first frame starts conservatively.
		// The List's committed rectangle supplies the next frame's real width.
		v.tableWindowWidth, v.tableWindowHeight, v.tableWindowKnown = windowWidth, windowHeight, true
		v.tableViewport, v.tableViewportKnown = 0, false
	}
	v.linkModifiers = c.Modifiers()
	v.refresh()
	ui.Column(c).Grow(1).MinHeight(0).Clip().Gap(6).ObserveInput(func(event ui.InputEvent) {
		v.observeInlineReferenceInput(event)
		v.observeTableActionInput(event)
	}).
		HandleInput(v.handleTableActionNavigation).Children(func() {
		hint := "双击内容编辑 · Ctrl+Enter 完成"
		if v.host.ReadOnly {
			hint = "只读预览"
		}
		ui.Text(c, hint).Shrink(0).Padding(4, 18).FontSize(11).TextColor(c.Theme().TextMuted)
		if v.hasLinks {
			ui.Text(c, "链接 / 图片引用：Ctrl/Cmd+点击 · Tab 后 Enter").Shrink(0).Padding(0, 18).FontSize(11).TextColor(c.Theme().TextMuted)
		}
		if v.host.Buffer == nil {
			ui.Text(c, "没有可用的文档缓冲区").TextColor(c.Theme().Warning)
			return
		}
		v.drawHistoryControls(c, history)
		if v.historyError != nil {
			ui.Text(c, v.historyError.Error()).MaxLines(2).Ellipsis("…").
				Tooltip(v.historyError.Error()).FontSize(11).TextColor(c.Theme().Warning).Shrink(0)
		}
		if v.tableActionError != nil && v.cellEdit == nil {
			ui.Text(c, v.tableActionError.Error()).Label("表格操作问题").MaxLines(2).Ellipsis("…").
				Tooltip(v.tableActionError.Error()).FontSize(11).TextColor(c.Theme().Warning).Shrink(0)
		}
		// A draft has its own stable position outside the recycled preview
		// rows. Scrolling cannot hide its IME client or move it to another block.
		if v.draft != nil {
			ui.Box(c.Key("active-draft")).Shrink(0).Padding(0, 10).Children(func() { v.drawDraft(c) })
		}
		if v.doc.Partial {
			message := partialSourceGuidance
			if v.previewError != nil {
				message = oversizeSourceGuidance
			}
			ui.Text(c, message).TextColor(c.Theme().Warning)
			return
		}
		if len(v.doc.Blocks) == 0 && v.draft == nil {
			message := "文档为空。切到源码开始书写。"
			if v.host.Buffer.LenBytes() > 0 {
				message = "没有可预览的内容块。完整原文仍保留，请切到源码视图查看。"
			}
			ui.Text(c, message).TextColor(c.Theme().TextMuted)
		}
		// Input is applied during widget construction. A task activation may
		// flush a draft and invalidate the next preview, but this pass continues
		// to use one immutable parser/row snapshot and its original identities.
		preview := previewSnapshot{v.doc, v.rows, v.docBuffer, v.version}
		list := ui.List(c.Key("preview-list"), v.list, len(preview.rows)).Grow(1).MinHeight(0).Clip().TrackScroll(&v.scroll).Padding(8, 18).Gap(6).
			ItemKey(func(i int) any { return preview.rows[i].id }).Rows(func(row ui.ListRow) { v.drawPreviewRow(row.Context, preview, row.Index) })
		list.ObserveInput(func(event ui.InputEvent) { v.historyCommand(history, event) })
		list.Draw(func(_ *ui.Painter, r ui.Rect) {
			if !v.hasTables {
				return
			}
			usable := r.W - previewListHorizontalPadding - previewRowHorizontalPadding
			if usable < 0 {
				usable = 0
			}
			if !v.tableViewportKnown || usable != v.tableViewport {
				v.tableViewport, v.tableViewportKnown = usable, true
				c.Invalidate() // one settle frame after a first layout or resize
			}
		})
		// Resolve shortcuts only after bindings/construction. The draft editor
		// is outside the preview subtree and consumes its own history/IME keys.
		// A row callback can Release during this build; its old receipt then
		// cannot mutate or refocus a newly mounted preview.
		history.list.Handle.OnShortcut(c, ui.Cmd, ui.KeyZ, func() { v.changeHistory(history, false) })
		history.list.Handle.OnShortcut(c, ui.Cmd, ui.KeyY, func() { v.changeHistory(history, true) })
		history.list.Handle.OnShortcut(c, ui.Cmd|ui.Shift, ui.KeyZ, func() { v.changeHistory(history, true) })
		if !v.host.ReadOnly && ui.Button(c, "添加段落").Height(28).Shrink(0).Margin(0, 18).Clicked() {
			v.addParagraph()
		}
	})
	// A release whose pressed row disappeared before Clicked is finished after
	// this pass. A visible matching button consumes it above when MyGo queues
	// its click; no retained gesture can keep an obsolete preview forever.
	if v.referencePointerUpBuffer != nil {
		v.referenceActivation = referenceTarget{}
		v.referencePointerUpBuffer, v.referencePointerUpRowID = nil, 0
	}
	if v.referenceActivation.buffer != nil && v.referenceGestureBuiltRowID != v.referenceActivation.rowID {
		captured := v.referenceActivation
		v.referenceActivation = referenceTarget{}
		if v.host.Buffer == nil || captured.buffer != v.host.Buffer ||
			captured.version != captured.buffer.Version() || v.doc.Partial {
			v.report(errReferenceStale)
		}
		// A merely virtualized-out row from a source-equivalent pass is an
		// ordinary canceled gesture and needs no visible refusal.
	}
	if v.tablePointerUpBuffer != nil {
		v.tableEditActivation = tableEditTarget{}
		v.tablePointerUpBuffer, v.tablePointerUpRowID = nil, 0
	}
	if v.tableEditActivation.buffer != nil &&
		v.tableGestureBuiltRowID != v.tableEditActivation.rowID {
		captured := v.tableEditActivation
		v.tableEditActivation = tableEditTarget{}
		if v.host.ReadOnly {
			v.report(errTableReadOnly)
		} else if !v.tableIdentityCurrent(captured) {
			v.report(errTableStale)
		}
	}
	if v.tableCellActivation.buffer != nil && !v.tableCellGestureBuilt {
		captured := v.tableCellActivation
		v.tableCellActivation = tableCellTarget{}
		if !v.tableCellIdentityCurrent(captured) {
			v.report(errTableCellStale)
		}
	}
}

func (v *View) drawDraft(c *ui.Context) {
	editorHeight := v.editPanelHeight
	if editorHeight <= 0 {
		editorHeight = max(48, min(80, float32(v.draft.Buffer.LineCount())*22+24))
	}
	ui.Column(c.Key("markdown-draft")).Border(1, c.Theme().Accent).Radius(5).Padding(8).Gap(6).Children(func() {
		if v.cellEdit != nil {
			ui.Text(c, v.tableAlignmentPanelTitle(v.cellEdit)).FontSize(11).Bold().TextColor(c.Theme().TextMuted).
				MaxLines(1).Ellipsis("…").Tooltip(v.cellEdit.label)
		}
		v.draft.ReadOnly = v.host.ReadOnly && !v.pendingDraft()
		draftBox := ui.Box(c).Height(editorHeight)
		if v.cellEdit != nil && !v.host.ReadOnly && draftBox.Shortcut(0, ui.KeyF6) {
			// A local focus escape leaves Tab's text-indent behavior intact in
			// the shared editor. MyGo unmarks on blur, so a live composition
			// keeps focus until the user confirms it through the normal seam.
			if _, composing := v.draft.MarkedRange(); !composing {
				v.tableActionHandle.Focus()
			}
		}
		draftBox.Children(func() { v.draft.View(c) })
		if v.cellEdit != nil {
			ui.Text(c, tableCellStagedNotice).FontSize(11).TextColor(c.Theme().TextMuted).
				Tooltip(tableCellSessionHint)
			if v.draftError != nil || v.tableActionError != nil {
				message, detail := "单元格未写回；草稿仍保留。", ""
				if v.draftError != nil {
					detail = v.draftError.Error()
				}
				if v.tableActionError != nil {
					message = tableAction{kind: v.tableActionErrorKind}.refusal() + "；单元格草稿仍保留。"
					if detail != "" {
						detail += "\n"
					}
					detail += v.tableActionError.Error()
				}
				ui.Text(c, message).Label("单元格操作问题").MaxLines(2).Ellipsis("…").Tooltip(detail).FontSize(11).TextColor(c.Theme().Warning)
			}
		} else if v.draftError != nil {
			message := "块草稿尚未写回：" + v.draftError.Error()
			if errors.Is(v.draftError, errDraftConflict) {
				message = "块草稿尚未写回；源码已变，可打开副本。"
			}
			ui.Text(c, message).MaxLines(2).Ellipsis("…").Tooltip(v.draftError.Error()).FontSize(11).TextColor(c.Theme().Warning)
		} else if v.pendingDraft() {
			ui.Text(c, "输入仍在草稿中；保存或完成编辑时写回").FontSize(11).TextColor(c.Theme().TextMuted)
		} else {
			ui.Text(c, "修改已同步到文档").FontSize(11).TextColor(c.Theme().TextMuted)
		}
		ui.Row(c).Wrap().Gap(8).Children(func() {
			finishFace, discardFace, recoverFace := "完成编辑", "放弃草稿", "打开未同步副本"
			if v.cellEdit != nil {
				finishFace, discardFace, recoverFace = "完成", "放弃", "恢复"
			}
			if ui.PrimaryButton(c, finishFace).Label("完成编辑").Height(26).Clicked() || c.Shortcut(ui.Cmd, ui.KeyEnter) || c.Shortcut(0, ui.KeyEscape) {
				v.commit()
			}
			if v.draftError != nil && ui.Button(c, recoverFace).Label("打开未同步副本").Height(26).Clicked() {
				_ = v.RecoverDraft()
			}
			if (v.draftError != nil || v.cellEdit != nil) && ui.Button(c, discardFace).Label("放弃草稿").Height(26).Clicked() {
				v.DiscardDraft()
			}
		})
		v.drawTableActions(c)
	})
}

func (v *View) report(err error) {
	if err != nil && v.host.Report != nil {
		v.host.Report(err)
	}
}
