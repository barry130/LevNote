package markdownview

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/ui/editor"
)

// fencedDraft keeps the syntax-required line ending out of the editor's input
// history while tracking the exact source field used for conflict detection.
type fencedDraft struct {
	ending string
	source string
}

func (v *View) pendingDraft() bool {
	if v.draft == nil {
		return false
	}
	_, composing := v.draft.MarkedRange()
	if v.cellEdit != nil {
		if composing || v.draft.Buffer.LenBytes() != len(v.editBase) {
			return true
		}
		return v.draft.Buffer.String() != v.editBase
	}
	return composing || v.draft.Buffer.String() != v.editBase
}

// A clear advances the same epoch that invalidates a held table action.
// A release can also replace the preview while keeping a failed draft alive.
type draftOwner struct {
	draft    *editor.Editor
	buffer   *text.Buffer
	epoch    uint64
	released uint64
}

func (v *View) draftOwner() draftOwner {
	return draftOwner{v.draft, v.host.Buffer, v.tableActionEpoch, v.tableActionReleased}
}

func (v *View) ownsDraft(owner draftOwner) bool {
	return v.draftOwner() == owner
}

func (v *View) ownsFlushedDraft(owner draftOwner) bool {
	// A successful Flush can close its cell or let a clean stale block yield
	// to source. Only that single expected clear belongs to this operation.
	if owner.draft != nil && v.draft == nil {
		owner.draft = nil
		owner.epoch++
	}
	return v.ownsDraft(owner)
}

func (v *View) clearDraft() bool {
	draft := v.draft
	v.tableActionEpoch++
	v.tableActionPress = tableActionGesture{}
	v.tableActionPointerContinuation = false
	v.draft, v.editBuffer = nil, nil
	v.editing, v.editBase, v.editPanelHeight, v.draftError = -1, "", 0, nil
	v.cellEdit, v.fencedDraft = nil, nil
	notify := v.resetCellDraftRecovery()
	owner := v.draftOwner()

	// Detach before touching native input and finish retiring only the
	// captured editor before a host callback can open and focus a new one.
	v.tableActionHandle.CancelFocus()
	var releaseErrors []error
	if draft != nil {
		draft.OnChange = nil
		releaseErrors = releaseDraftView(draft)
	}
	if notify != nil {
		notify(v)
	}
	for _, err := range releaseErrors {
		v.report(err)
	}
	return v.ownsDraft(owner)
}

func releaseDraftView(draft *editor.Editor) []error {
	// A composition error must reach the host after the editor finishes its
	// own release and focus cleanup. Reporting it is itself a reentry point.
	onError := draft.OnError
	var errs []error
	draft.OnError = func(err error) { errs = append(errs, err) }
	draft.ReleaseView()
	draft.OnError = onError
	return errs
}

func (v *View) reportDraftInputError(draft *editor.Editor, err error) {
	if v.draft == draft {
		v.setDraftError(err)
	} else {
		v.report(err)
	}
}

func (v *View) setDraftError(err error) {
	changed := err != nil && (v.draftError == nil || v.draftError.Error() != err.Error())
	v.draftError = err
	if changed {
		v.report(err)
	}
}

// rebaseDraft follows only edits outside the field. Clean external changes
// win, including undo; they must never be silently reapplied by a later Save.
func (v *View) rebaseDraft() (bool, error) {
	if v.draft == nil {
		return false, nil
	}
	base := v.editBase
	if v.fencedDraft != nil {
		base = v.fencedDraft.source
	}
	if v.host.Buffer != nil && v.host.Buffer == v.editBuffer {
		guard, guardOK := v.host.Buffer.MapRangeSince(v.editVersion, v.editGuard)
		if span, ok := v.host.Buffer.MapRangeSince(v.editVersion, v.editSpan); ok && guardOK {
			if current, err := v.host.Buffer.SliceUTF16(span.Start(), span.End()); err == nil && current == base {
				v.editSpan, v.editGuard, v.editVersion = span, guard, v.host.Buffer.Version()
				return true, nil
			}
		}
	}
	if !v.pendingDraft() {
		v.loaded = false
		v.clearDraft()
		return false, nil
	}
	return false, errDraftConflict
}

// draftFlushReceipt records a successful source write before any host callback
// can change the buffer or its owner. It lives only in the synchronous flush.
type draftFlushReceipt struct {
	cellCommitted  bool
	blockCommitted bool
}

func (r draftFlushReceipt) supersededError() error {
	if r.cellCommitted {
		return fmt.Errorf("原单元格已写回：%w", errDraftSuperseded)
	}
	if r.blockCommitted {
		return fmt.Errorf("原块已写回：%w", errDraftSuperseded)
	}
	return errDraftSuperseded
}

func (v *View) Flush() error {
	_, err := v.flushDraft(false)
	return err
}

func (v *View) flushDraft(actionReceipt bool) (draftFlushReceipt, error) {
	var receipt draftFlushReceipt
	owner := v.draftOwner()
	if owner.draft == nil {
		return receipt, nil
	}
	var flushErr error
	if actionReceipt && v.cellEdit == nil {
		// Composition settlement may call OnChange inside Editor.Flush. Route
		// that one callback through this call-local receipt; never keep it on View.
		onChange := owner.draft.OnChange
		owner.draft.OnChange = func() {
			if v.draft == owner.draft {
				_ = v.syncDraftWithReceipt(&receipt)
			}
		}
		flushErr = owner.draft.Flush()
		if v.draft == owner.draft {
			owner.draft.OnChange = onChange
		}
	} else {
		flushErr = owner.draft.Flush()
	}
	if flushErr != nil {
		if v.ownsDraft(owner) {
			v.setDraftError(flushErr)
		} else {
			v.report(flushErr)
		}
		return receipt, flushErr
	}
	if !v.ownsDraft(owner) {
		err := errDraftSuperseded
		if actionReceipt {
			err = receipt.supersededError()
		}
		v.report(err)
		return receipt, err
	}
	if v.cellEdit != nil {
		return v.syncCellDraft()
	}
	var err error
	if actionReceipt {
		err = v.syncDraftWithReceipt(&receipt)
	} else {
		err = v.syncDraft()
	}
	if err != nil {
		return receipt, err
	}
	if !v.ownsFlushedDraft(owner) {
		err := errDraftSuperseded
		if actionReceipt {
			err = receipt.supersededError()
		}
		v.report(err)
		return receipt, err
	}
	return receipt, nil
}

// A successful write can call Changed. Check that its caller still owns the
// action before opening a panel, navigating, or starting a second edit.
func (v *View) flushDraftAction() error {
	owner := v.draftOwner()
	receipt, err := v.flushDraft(true)
	if err != nil {
		return err
	}
	if !v.ownsFlushedDraft(owner) {
		err := receipt.supersededError()
		v.report(err)
		return err
	}
	return nil
}

// draftActionReport belongs to one UI invocation. A failed flush owns its
// notification (including setDraftError's existing unchanged-issue behavior);
// returning that same refusal through an action is not another report event.
// The action must return a flush refusal unchanged and immediately. Target
// errors after a successful flush remain the UI caller's duty. No host callback
// is replaced and no receipt survives this synchronous call.
type draftActionReport struct {
	flushRefused bool
}

func (r *draftActionReport) flush(v *View) error {
	err := v.flushDraftAction()
	if r != nil {
		r.flushRefused = err != nil
	}
	return err
}

func (r draftActionReport) finish(v *View, err error) {
	if !r.flushRefused {
		v.report(err) // terminal: no view-owned state is written after Report
	}
}

func (v *View) syncDraft() error {
	return v.syncDraftWithReceipt(nil)
}

func (v *View) syncDraftWithReceipt(receipt *draftFlushReceipt) error {
	if v.draft == nil {
		return nil
	}
	ok, err := v.rebaseDraft()
	if err != nil {
		v.setDraftError(err)
		return err
	}
	if !ok {
		return nil
	}
	if _, composing := v.draft.MarkedRange(); composing {
		return nil // never hold a host transaction across native IME events
	}
	value := v.draft.Buffer.String()
	if value == v.editBase {
		v.setDraftError(nil)
		return nil
	}
	source := value
	if v.fencedDraft != nil && source != "" {
		// The final terminator separates the body from the closing fence.
		// Keep its original style even when local Enter defaults to LF; the
		// local input and all internal line endings remain unchanged.
		if strings.HasSuffix(source, "\r\n") {
			source = strings.TrimSuffix(source, "\r\n")
		} else {
			source = strings.TrimSuffix(source, "\n")
		}
		source += v.fencedDraft.ending
	}
	previousSource := v.editBase
	if v.fencedDraft != nil {
		previousSource = v.fencedDraft.source
	}
	sourceChanged := source != previousSource
	if v.host.ReadOnly {
		err = errors.New("文档当前只读，块草稿仍保留")
	} else if v.host.Buffer.InGroup() {
		err = text.ErrGroup
	} else {
		err = v.host.Buffer.ReplaceUTF16(v.editSpan.Start(), v.editSpan.End(), source)
	}
	if err != nil {
		v.setDraftError(err)
		return err
	}
	if receipt != nil && sourceChanged {
		receipt.blockCommitted = true
	}
	v.editGuard.Head += text.UTF16Len(source) - (v.editSpan.End() - v.editSpan.Start())
	v.editBase, v.editVersion = value, v.host.Buffer.Version()
	v.editSpan.Head = v.editSpan.Start() + text.UTF16Len(source)
	if v.fencedDraft != nil {
		v.fencedDraft.source = source
	}
	v.setDraftError(nil)
	if v.host.Changed != nil {
		v.host.Changed()
	}
	return nil
}

func (v *View) begin(_ int, block markdown.Block) {
	if v.host.ReadOnly || v.host.Buffer == nil {
		return
	}
	owner := v.draftOwner()
	_ = v.checkPreviewSize()
	if !v.ownsDraft(owner) || v.flushDraftAction() != nil {
		return
	}
	if v.checkPreviewSize() != nil {
		return
	}
	// The clicked row can come from the frozen preview preceding many local
	// edits. Map its whole source block first, then reparse and require that
	// exact block before clearing the former editor. An index alone is unsafe.
	raw, version, buffer := v.doc.Raw(), v.version, v.docBuffer
	r := block.Range
	if buffer != v.host.Buffer || r.Start < 0 || r.End > len(raw) || r.Start > r.End {
		v.report(errDraftConflict)
		return
	}
	span := text.Range{Anchor: text.UTF16Len(raw[:r.Start]), Head: text.UTF16Len(raw[:r.End])}
	span, ok := buffer.MapRangeSince(version, span)
	if !ok {
		v.report(errDraftConflict)
		return
	}
	current, err := buffer.SliceUTF16(span.Start(), span.End())
	if err != nil || current != raw[r.Start:r.End] {
		v.report(errDraftConflict)
		return
	}
	prefix, err := buffer.SliceUTF16(0, span.Start())
	if err != nil {
		v.report(err)
		return
	}
	doc := markdown.Parse(buffer.String())
	if doc.Partial {
		v.report(errors.New(partialSourceGuidance))
		return
	}
	for index, candidate := range doc.Blocks {
		if candidate.Kind != block.Kind || candidate.Range.Start != len(prefix) || candidate.Range.End != len(prefix)+len(current) {
			continue
		}
		field := candidate.TextRange
		if candidate.Kind == markdown.BlockDocBlock || candidate.Kind == markdown.BlockDivider || candidate.Kind == markdown.BlockList {
			field = candidate.Range
		}
		if field.Start < candidate.Range.Start || field.End > candidate.Range.End || field.Start > field.End {
			v.report(errDraftConflict)
			return
		}
		if err := v.openDraft(index, candidate, doc, buffer, span, field, 2048, 0); err != nil {
			v.report(err)
		}
		return
	}
	v.report(errDraftConflict)
}

// beginSource opens one already validated current block through the same
// independent draft lifecycle. Tables pass the complete block as the field;
// ordinary double-click editing passes only its editable TextRange.
func (v *View) beginSource(index int, block markdown.Block, doc markdown.Document, span text.Range, shapeLimit int, panelHeight float32) error {
	buffer := v.host.Buffer
	if v.host.ReadOnly {
		return errDraftReadOnly
	}
	if buffer == nil || doc.Partial {
		return errDraftConflict
	}
	// The table-source caller already flushed. An independent cell still
	// here belongs to a callback and must not be replaced by this open.
	if v.cellEdit != nil || v.pendingDraft() {
		return errDraftSuperseded
	}
	current, err := buffer.SliceUTF16(span.Start(), span.End())
	if err != nil || block.Range.Start < 0 || block.Range.End < block.Range.Start ||
		block.Range.End > len(doc.Raw()) || current != doc.Raw()[block.Range.Start:block.Range.End] {
		return errDraftConflict
	}
	return v.openDraft(index, block, doc, buffer, span, block.Range, shapeLimit, panelHeight)
}

func (v *View) openDraft(index int, block markdown.Block, doc markdown.Document, buffer *text.Buffer, outer text.Range, field markdown.SourceRange, shapeLimit int, panelHeight float32) error {
	raw := doc.Raw()
	if v.host.ReadOnly {
		return errDraftReadOnly
	}
	if buffer == nil || buffer != v.host.Buffer || index < 0 || index >= len(doc.Blocks) || field.Start < block.Range.Start ||
		field.End > block.Range.End || field.Start > field.End || block.Range.End > len(raw) ||
		outer.Start() < 0 || outer.End() > buffer.LenUTF16() {
		return errDraftConflict
	}
	outerValue, err := buffer.SliceUTF16(outer.Start(), outer.End())
	if err != nil || outerValue != raw[block.Range.Start:block.Range.End] {
		return errDraftConflict
	}
	version := buffer.Version()
	if !v.clearDraft() {
		return errDraftSuperseded
	}
	if v.host.ReadOnly {
		return errDraftReadOnly
	}
	if buffer.Version() != version {
		return errDraftConflict
	}
	v.setPreview(doc)
	v.version, v.docBuffer, v.loaded = version, buffer, true
	v.editing, v.editVersion, v.editBuffer = index, version, buffer
	v.editSpan = text.Range{Anchor: text.UTF16Len(raw[:field.Start]), Head: text.UTF16Len(raw[:field.End])}
	v.editGuard = text.Range{Anchor: text.UTF16Len(raw[:block.Range.Start]), Head: text.UTF16Len(raw[:block.Range.End])}
	v.editBase = strings.Clone(raw[field.Start:field.End])
	if block.Kind == markdown.BlockCode && field == block.TextRange && field.End < block.Range.End {
		// TextRange ends at the closing fence, including its preceding line
		// ending. An empty body borrows the opening fence's line ending.
		ending := "\n"
		if strings.HasSuffix(raw[:field.End], "\r\n") {
			ending = "\r\n"
		}
		v.fencedDraft = &fencedDraft{ending: ending, source: v.editBase}
	}
	draft := editor.New(text.New(v.editBase))
	draft.ShapeLimit = shapeLimit
	draft.MinHeight = 48
	draft.LineNumbers = false
	draft.OnError = func(err error) { v.reportDraftInputError(draft, err) }
	draft.OnChange = func() {
		if v.draft == draft {
			_ = v.syncDraft()
		}
	}
	v.editPanelHeight = panelHeight
	v.draft = draft
	draft.Focus()
	return nil
}

func (v *View) commit() {
	if v.flushDraftAction() != nil {
		return
	}
	v.list.HoldPlace()
	v.loaded = false
	v.clearDraft()
}

func (v *View) RecoverDraft() error {
	owner := v.draftOwner()
	draft := owner.draft
	if draft == nil {
		return nil
	}
	_ = draft.Flush() // recovery can still copy accepted text after a refusal
	if !v.ownsDraft(owner) {
		v.report(errDraftSuperseded)
		return errDraftSuperseded
	}
	if v.host.RecoverDraft == nil {
		err := errors.New("宿主无法创建草稿副本；当前草稿仍保留")
		v.setDraftError(err)
		return err
	}
	name := filepath.Base(v.host.Path)
	if name == "" || name == "." {
		name = "Markdown"
	}
	value, version := draft.Buffer.String(), draft.Buffer.Version()
	inputChanged, onChange := false, draft.OnChange
	draft.OnChange = func() {
		inputChanged = true
		if onChange != nil {
			onChange()
		}
	}
	defer func() {
		if v.draft == draft {
			draft.OnChange = onChange
		}
	}()
	if err := v.host.RecoverDraft(name+"-块草稿.txt", value); err != nil {
		if v.ownsDraft(owner) {
			v.setDraftError(err)
		} else {
			v.report(err)
		}
		return err
	}
	if !v.ownsDraft(owner) || draft.Buffer.Version() != version || inputChanged {
		return nil
	}
	v.doc, v.docBuffer, v.loaded = markdown.Document{}, nil, false
	v.rows = nil
	v.clearDraft()
	return nil
}

func (v *View) DiscardDraft() {
	v.loaded = false
	v.clearDraft()
}

func (v *View) addParagraph() {
	if v.flushDraftAction() != nil {
		return
	}
	if v.host.Buffer == nil || v.host.ReadOnly {
		return
	}
	if v.host.Buffer.InGroup() {
		v.report(text.ErrGroup)
		return
	}
	end := v.host.Buffer.LenUTF16()
	ending := "\n"
	// Follow the last actual line break, including a final line without its
	// own newline. Read only separator units, not the preceding line's text.
	if lastLine := v.host.Buffer.LineCount() - 1; lastLine > 0 {
		previous, _ := v.host.Buffer.LineRange(lastLine - 1)
		next, _ := v.host.Buffer.OffsetOfLine(lastLine)
		ending, _ = v.host.Buffer.SliceUTF16(previous.End(), next)
	}
	if err := v.host.Buffer.ReplaceUTF16(end, end, ending+ending+"新段落"+ending); err != nil {
		v.report(err)
		return
	}
	changed := v.host.Changed
	v.loaded = false
	v.clearDraft()
	if changed != nil {
		changed()
	}
}
