package markdownview

import (
	"errors"
	"path/filepath"
	"strings"

	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/ui/editor"
)

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

func (v *View) clearDraft() {
	v.tableActionEpoch++
	v.tableActionPress = tableActionGesture{}
	v.tableActionPointerContinuation = false
	v.tableActionHandle.CancelFocus()
	if v.draft != nil {
		v.draft.ReleaseView()
	}
	v.draft, v.editBuffer = nil, nil
	v.editing, v.editBase, v.editPanelHeight, v.draftError = -1, "", 0, nil
	v.cellEdit = nil
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
	if v.host.Buffer != nil && v.host.Buffer == v.editBuffer {
		guard, guardOK := v.host.Buffer.MapRangeSince(v.editVersion, v.editGuard)
		if span, ok := v.host.Buffer.MapRangeSince(v.editVersion, v.editSpan); ok && guardOK {
			if current, err := v.host.Buffer.SliceUTF16(span.Start(), span.End()); err == nil && current == v.editBase {
				v.editSpan, v.editGuard, v.editVersion = span, guard, v.host.Buffer.Version()
				return true, nil
			}
		}
	}
	if !v.pendingDraft() {
		v.clearDraft()
		v.loaded = false
		return false, nil
	}
	return false, errDraftConflict
}

func (v *View) Flush() error {
	if v.draft == nil {
		return nil
	}
	if err := v.draft.Flush(); err != nil {
		v.setDraftError(err)
		return err
	}
	if v.cellEdit != nil {
		return v.syncCellDraft()
	}
	return v.syncDraft()
}

func (v *View) syncDraft() error {
	if v.draft == nil {
		return nil
	}
	ok, err := v.rebaseDraft()
	if err != nil || !ok {
		v.setDraftError(err)
		return err
	}
	if _, composing := v.draft.MarkedRange(); composing {
		return nil // never hold a host transaction across native IME events
	}
	value := v.draft.Buffer.String()
	if value == v.editBase {
		v.setDraftError(nil)
		return nil
	}
	if v.host.ReadOnly {
		err = errors.New("文档当前只读，块草稿仍保留")
	} else if v.host.Buffer.InGroup() {
		err = text.ErrGroup
	} else {
		err = v.host.Buffer.ReplaceUTF16(v.editSpan.Start(), v.editSpan.End(), value)
	}
	if err != nil {
		v.setDraftError(err)
		return err
	}
	v.editGuard.Head += text.UTF16Len(value) - (v.editSpan.End() - v.editSpan.Start())
	v.editBase, v.editVersion = value, v.host.Buffer.Version()
	v.editSpan.Head = v.editSpan.Start() + text.UTF16Len(value)
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
	_ = v.checkPreviewSize()
	if err := v.Flush(); err != nil {
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
	if buffer == nil || index < 0 || index >= len(doc.Blocks) || field.Start < block.Range.Start ||
		field.End > block.Range.End || field.Start > field.End || block.Range.End > len(raw) ||
		outer.Start() < 0 || outer.End() > buffer.LenUTF16() {
		return errDraftConflict
	}
	outerValue, err := buffer.SliceUTF16(outer.Start(), outer.End())
	if err != nil || outerValue != raw[block.Range.Start:block.Range.End] {
		return errDraftConflict
	}
	v.clearDraft()
	v.setPreview(doc)
	v.version, v.docBuffer, v.loaded = buffer.Version(), buffer, true
	v.editing, v.editVersion, v.editBuffer = index, buffer.Version(), buffer
	v.editSpan = text.Range{Anchor: text.UTF16Len(raw[:field.Start]), Head: text.UTF16Len(raw[:field.End])}
	v.editGuard = text.Range{Anchor: text.UTF16Len(raw[:block.Range.Start]), Head: text.UTF16Len(raw[:block.Range.End])}
	v.editBase = strings.Clone(raw[field.Start:field.End])
	draft := editor.New(text.New(v.editBase))
	draft.ShapeLimit = shapeLimit
	draft.MinHeight = 48
	draft.LineNumbers = false
	draft.OnError = v.setDraftError
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
	if err := v.Flush(); err != nil {
		return
	}
	v.clearDraft()
	v.list.HoldPlace()
	v.loaded = false
}

func (v *View) RecoverDraft() error {
	if v.draft == nil {
		return nil
	}
	draft := v.draft
	_ = draft.Flush()
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
	defer func() { draft.OnChange = onChange }()
	if err := v.host.RecoverDraft(name+"-块草稿.txt", value); err != nil {
		if v.draft == draft {
			v.setDraftError(err)
		} else {
			v.report(err)
		}
		return err
	}
	if v.draft != draft || draft.Buffer.Version() != version || inputChanged {
		return nil
	}
	v.clearDraft()
	v.doc, v.docBuffer, v.loaded = markdown.Document{}, nil, false
	v.rows = nil
	return nil
}

func (v *View) DiscardDraft() {
	v.clearDraft()
	v.loaded = false
}

func (v *View) addParagraph() {
	if err := v.Flush(); err != nil {
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
	v.clearDraft()
	v.loaded = false
	if v.host.Changed != nil {
		v.host.Changed()
	}
}
