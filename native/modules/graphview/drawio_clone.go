package graphview

import (
	"errors"
	"fmt"
	"slices"

	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
)

// A history selection is tied to an exact buffer/page and the selection epoch
// produced by the clone itself (or its previous undo). Replacing the buffer,
// navigating elsewhere or selecting another shape retires that restoration.
const (
	drawioCloneHistoryMaxEntries = 256
	drawioCloneHistoryMaxBytes = 32 << 20
)

type drawioCloneHistory struct {
	buffer     *text.Buffer
	page       int
	pageEpoch  uint64
	release    uint64
	version    uint64
	beforeText string
	afterText  string
	beforeIDs  []string
	afterIDs   []string
	afterEpoch uint64
	undoEpoch  uint64
	pending     bool
	pendingUndo bool
}

func drawioCloneHistoryRecordBytes(record drawioCloneHistory) int {
	if len(record.beforeText) > drawioCloneHistoryMaxBytes-len(record.afterText) {
		return drawioCloneHistoryMaxBytes + 1
	}
	return len(record.beforeText) + len(record.afterText)
}

func drawioCloneHistoryBytes(history []drawioCloneHistory) int {
	const limit = drawioCloneHistoryMaxBytes + 1
	bytes := 0
	for _, record := range history {
		size := drawioCloneHistoryRecordBytes(record)
		if size >= limit-bytes { return limit }
		bytes += size
	}
	return bytes
}

func appendDrawioCloneHistory(history []drawioCloneHistory, record drawioCloneHistory) []drawioCloneHistory {
	incoming := drawioCloneHistoryRecordBytes(record)
	if incoming > drawioCloneHistoryMaxBytes { return history }
	for len(history) > 0 && (len(history) >= drawioCloneHistoryMaxEntries ||
		drawioCloneHistoryBytes(history) > drawioCloneHistoryMaxBytes-incoming) {
		// Retire the physical head before advancing the logical slice. The append
		// below then writes past every physical slot retained in survivors.
		clear(history[:1])
		history = history[1:]
	}
	return append(history, record)
}

func drawioClonedSource(raw string, page int, selected ...string) ([]byte, []string, error) {
	candidate, _, err := drawio.Parse([]byte(raw), drawio.ParseOptions{})
	if err != nil { return nil, nil, err }
	vertexIDs, err := candidate.CloneCells(page, selected, 24, 24)
	if err != nil { return nil, nil, err }
	serialized, err := candidate.Serialize()
	return serialized, vertexIDs, err
}

func (v *view) cloneDrawioCells() bool {
	selectionEpoch := v.drawSelection.epoch
	buffer, released := v.host.Buffer, v.releaseEpoch
	pageEpoch, tagEpoch, page := v.pageEpoch, v.tagEpoch, v.pageIndex
	targets, primary, field := slices.Clone(v.drawSelection.ids), v.selectedID, v.field
	input := v.fieldIdentity()
	sameOwner := func() bool {
		return v.host.Buffer == buffer && v.releaseEpoch == released && v.pageEpoch == pageEpoch &&
			v.tagEpoch == tagEpoch && v.pageIndex == page && v.drawSelection.epoch == selectionEpoch &&
			v.selectedID == primary && v.field == field && v.sameFieldIdentity(input) &&
			slices.Equal(v.drawSelection.ids, targets)
	}
	refuse := func(err error) bool {
		if sameOwner() {
			v.addWarning("未克隆：" + err.Error() + "；原文保持不变。")
			v.report(err)
		}
		return true
	}
	if buffer == nil { return refuse(errors.New("没有可写回的文档缓冲区")) }
	if v.host.ReadOnly { return refuse(errors.New("文档当前只读")) }
	if len(targets) < 2 { return refuse(errors.New("请先用 Ctrl/Cmd 点击选择至少两个要克隆的图形")) }
	if buffer.InGroup() { return refuse(text.ErrGroup) }
	if v.flushing { return refuse(ErrDraftConflict) }
	if v.CompositionActive() { return refuse(errors.New("请先确认正在输入的文字，再克隆图形")) }
	version, caret := buffer.Version(), buffer.Selection()
	sameSource := func() bool {
		return sameOwner() && !v.host.ReadOnly && !buffer.InGroup() && !v.CompositionActive() &&
			buffer.Version() == version && buffer.Selection() == caret
	}
	v.ensureSynced(nil)
	if !sameSource() { return refuse(ErrDraftConflict) }
	if v.syncFailed || v.syncedBuffer != buffer || v.lastSynced != version { return refuse(ErrDraftConflict) }
	if err := v.writable(); err != nil { return refuse(err) }
	raw := buffer.String()
	serialized, ids, err := drawioClonedSource(raw, page, targets...)
	if err != nil { return refuse(err) }
	receipt, err := v.flushOrdinaryField()
	if err != nil { return true }
	expectedVersion := version
	if receipt.written { expectedVersion++ }
	if !sameOwner() || receipt.version != expectedVersion { return refuse(ErrDraftConflict) }
	version, caret = receipt.version, receipt.selection
	if !sameSource() || v.flushing { return refuse(ErrDraftConflict) }
	if receipt.written {
		raw = buffer.String()
		serialized, ids, err = drawioClonedSource(raw, page, targets...)
		if err != nil { return refuse(err) }
	}
	next := string(serialized)
	if next == raw || len(ids) != len(targets) { return refuse(errors.New("克隆没有产生可确认的节点副本")) }
	if err := buffer.CheckReplaceUTF16(0, buffer.LenUTF16(), next); err != nil { return refuse(err) }
	if !sameSource() { return refuse(ErrDraftConflict) }
	beforeText := raw
	clamped, err := func() (bool, error) {
		v.flushing = true
		defer func() { v.flushing = false }()
		return v.commitBufferSelection(next, true)
	}()
	if err != nil { return refuse(err) }
	clampedCaret := text.Range{Anchor: caret.Anchor, Head: caret.Head}
	_, anchorErr := buffer.SliceUTF16(clampedCaret.Anchor, clampedCaret.Anchor)
	_, headErr := buffer.SliceUTF16(clampedCaret.Head, clampedCaret.Head)
	clamped = errors.Is(anchorErr, text.ErrSurrogate) || errors.Is(headErr, text.ErrSurrogate) || clampedCaret != buffer.Selection()
	if clamped {
		if errors.Is(anchorErr, text.ErrSurrogate) { clampedCaret.Anchor-- }
		if errors.Is(headErr, text.ErrSurrogate) { clampedCaret.Head-- }
	}
	clamped = clamped || buffer.Selection() != caret
	if !sameOwner() || v.host.ReadOnly || buffer.InGroup() || v.CompositionActive() ||
		buffer.Version() != version+1 || buffer.String() != next || buffer.Selection() != clampedCaret { return true }
	v.cancelGraphGesture()
	v.setDrawioSelection(ids, ids[len(ids)-1])
	v.lastSynced = ^uint64(0)
	v.ensureSynced(nil)
	if v.host.Buffer != buffer || v.releaseEpoch != released || v.pageEpoch != pageEpoch || v.pageIndex != page ||
		v.syncedBuffer != buffer || v.lastSynced != buffer.Version() || v.syncFailed ||
		!slices.Equal(v.drawSelection.ids, ids) || v.selectedID != ids[len(ids)-1] || buffer.String() != next { return true }
	v.drawCloneHistory = appendDrawioCloneHistory(v.drawCloneHistory, drawioCloneHistory{
		buffer:buffer, page:page, pageEpoch:pageEpoch, release:released, version:version,
		beforeText:beforeText, afterText:next, beforeIDs:slices.Clone(targets), afterIDs:slices.Clone(ids),
		afterEpoch:v.drawSelection.epoch,
	})
	message := fmt.Sprintf("已克隆 %d 个图形及内部连线到当前页；可撤销。", len(ids))
	if clamped { message += "源码光标已移到前一个有效的 UTF-16 边界。" }
	v.warnings = append([]string{message}, v.warnings...)
	v.canvasHandle.Focus()
	return true
}

// Capture the exact previous current selection, then restore only after the
// clone history replay yields the expected original/cloned XML. Any unrelated
// user edit or selection change retires this clone record.
func (v *view) beginDrawioCloneHistory(buffer *text.Buffer, release, pageEpoch uint64, page int, undo bool) *drawioCloneHistory {
	if v.kind != kindDrawio || buffer == nil { return nil }
	for i := len(v.drawCloneHistory)-1; i >= 0; i-- {
		record := &v.drawCloneHistory[i]
		if record.buffer != buffer || record.release != release || record.pageEpoch != pageEpoch || record.page != page { continue }
		epoch, ids, raw := record.afterEpoch, record.afterIDs, record.beforeText
		if !undo { epoch, ids, raw = record.undoEpoch, record.beforeIDs, record.afterText }
		if buffer.String() != raw || !slices.Equal(v.drawSelection.ids, ids) { continue }
		// Buffer history has already moved to raw, while the canvas still has
		// the selection from the other side of this transaction. Witness that
		// pre-replay boundary, not the source we are about to restore.
		boundary := record.afterText
		if !undo { boundary = record.beforeText }
		if v.drawSelection.epoch != epoch && !v.knownDrawioCloneBoundary(buffer, release, pageEpoch, page, boundary) { continue }
		record.pendingUndo = undo
		record.pending = true
		return record
	}
	return nil
}

// A later clone can retire the preceding record's epoch while undoing to an
// intermediate snapshot. Accept only the matching epoch already witnessed for
// the same XML and ordered IDs by a record with the same owner. An explicit
// user selection has a new, unwitnessed epoch and keeps its restoration veto.
func (v *view) knownDrawioCloneBoundary(buffer *text.Buffer, release, pageEpoch uint64, page int, raw string) bool {
	for i, record := range v.drawCloneHistory {
		if record.buffer != buffer || record.release != release || record.pageEpoch != pageEpoch || record.page != page { continue }
		if record.afterText == raw && record.afterEpoch == v.drawSelection.epoch &&
			slices.Equal(record.afterIDs, v.drawSelection.ids) { return true }
		if record.beforeText == raw && record.undoEpoch == v.drawSelection.epoch &&
			slices.Equal(record.beforeIDs, v.drawSelection.ids) { return true }
		if i+1 == len(v.drawCloneHistory) || record.afterText != raw || record.afterText != v.drawCloneHistory[i+1].beforeText { continue }
		later := &v.drawCloneHistory[i+1]
		if later.buffer == buffer && later.release == release && later.pageEpoch == pageEpoch && later.page == page &&
			later.buffer == record.buffer && later.release == record.release && later.pageEpoch == record.pageEpoch && later.page == record.page && later.undoEpoch == v.drawSelection.epoch &&
			slices.Equal(record.afterIDs, v.drawSelection.ids) && slices.Equal(later.beforeIDs, v.drawSelection.ids) { return true }
	}
	return false
}

func (v *view) finishDrawioCloneHistory(record *drawioCloneHistory, buffer *text.Buffer, release, pageEpoch uint64, page int) {
	if record == nil || !record.pending { return }
	record.pending = false
	ids, raw := record.afterIDs, record.afterText
	if record.pendingUndo { ids, raw = record.beforeIDs, record.beforeText }
	if record.buffer != buffer || record.release != release || record.pageEpoch != pageEpoch || record.page != page ||
		buffer.String() != raw || v.pageIndex != page { return }
	if v.currentPage() == nil { return }
	for _, id := range ids {
		found := false
		for _, node := range v.currentPage().Model.Nodes { if node.ID == id { found = true; break } }
		if !found { return }
	}
	v.setDrawioSelection(ids, ids[len(ids)-1])
	if record.pendingUndo { record.undoEpoch = v.drawSelection.epoch } else { record.afterEpoch = v.drawSelection.epoch }
}
