package graphview

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
)

const (
	drawioCloneHistoryMaxEntries = 256
	drawioCloneHistoryMaxBytes   = 32 << 20
	drawioCloneIDHeaderBytes     = 2 * (strconv.IntSize / 8)
)

// These IDs contain the binary SHA-256 of the exact XML bytes. The empty value
// means no identity. They keep no reference to the source document.
type drawioCloneTextID string

func drawioCloneTextIdentity(source []byte) drawioCloneTextID {
	digest := sha256.Sum256(source)
	return drawioCloneTextID(digest[:])
}

// A history selection is tied to the buffer, page and selection epoch produced
// by the clone or its replay. beforeText/afterText identify the two sources.
type drawioCloneHistory struct {
	buffer                *text.Buffer
	page                  int
	pageEpoch, release    uint64
	beforeText, afterText drawioCloneTextID
	beforeIDs, afterIDs   []string
	beforePrimary         int
	afterEpoch, undoEpoch uint64
}

func drawioCloneHistoryRecordBytes(record drawioCloneHistory) int {
	size := len(record.beforeText) + len(record.afterText)
	for _, ids := range [...][]string{record.beforeIDs, record.afterIDs} {
		for _, id := range ids {
			if len(id) > drawioCloneHistoryMaxBytes-size-drawioCloneIDHeaderBytes {
				return drawioCloneHistoryMaxBytes + 1
			}
			size += drawioCloneIDHeaderBytes + len(id)
		}
	}
	return size
}

func drawioCloneHistoryBytes(history []drawioCloneHistory) int {
	const limit = drawioCloneHistoryMaxBytes + 1
	size := 0
	for _, record := range history {
		size += drawioCloneHistoryRecordBytes(record)
		if size >= limit {
			return limit
		}
	}
	return size
}

func appendDrawioCloneHistory(history []drawioCloneHistory, record drawioCloneHistory) ([]drawioCloneHistory, bool) {
	incoming := drawioCloneHistoryRecordBytes(record)
	if incoming > drawioCloneHistoryMaxBytes {
		return history, false
	}
	size := drawioCloneHistoryBytes(history)
	for len(history) > 0 && (len(history) >= drawioCloneHistoryMaxEntries || size > drawioCloneHistoryMaxBytes-incoming) {
		size -= drawioCloneHistoryRecordBytes(history[0])
		clear(history[:1])
		history = history[1:]
	}
	// Own the ID bytes as well as the slices. Parsed IDs may otherwise retain
	// backing storage larger than their length.
	cloneIDs := func(ids []string) []string {
		owned := make([]string, len(ids))
		for i, id := range ids {
			owned[i] = strings.Clone(id)
		}
		return owned
	}
	record.beforeIDs, record.afterIDs = cloneIDs(record.beforeIDs), cloneIDs(record.afterIDs)
	return append(history, record), true
}

func drawioClonedSource(raw string, page int, selected ...string) ([]byte, []string, error) {
	candidate, _, err := drawio.Parse([]byte(raw), drawio.ParseOptions{})
	if err != nil {
		return nil, nil, err
	}
	vertexIDs, err := candidate.CloneCells(page, selected, 24, 24)
	if err != nil {
		return nil, nil, err
	}
	serialized, err := candidate.Serialize()
	return serialized, vertexIDs, err
}

type drawioCloneEdit struct {
	at                    int
	inserted              string
	ids                   []string
	beforeText, afterText drawioCloneTextID
}

func planDrawioClone(buffer *text.Buffer, page int, targets []string) (drawioCloneEdit, error) {
	raw := buffer.String()
	serialized, ids, err := drawioClonedSource(raw, page, targets...)
	if err != nil {
		return drawioCloneEdit{}, err
	}
	added := len(serialized) - len(raw)
	if added <= 0 || len(ids) != len(targets) {
		return drawioCloneEdit{}, errors.New("克隆没有产生可确认的节点副本")
	}

	// CloneCells inserts one span at the root close. Find a split where the
	// prefix and suffix exactly match, even if some bytes repeat at the seam.
	cut := len(raw)
	for cut > 0 && raw[cut-1] == serialized[cut+added-1] {
		cut--
	}
	for cut < len(raw) && !utf8.RuneStart(raw[cut]) {
		cut++
	}
	if !bytes.Equal([]byte(raw[:cut]), serialized[:cut]) {
		return drawioCloneEdit{}, errors.New("克隆结果改动了原有图纸内容")
	}
	at := text.UTF16Len(raw[:cut])
	// Copy only the insertion so text history cannot retain the output XML.
	inserted := string(serialized[cut : cut+added])
	if err := buffer.CheckReplaceUTF16(at, at, inserted); err != nil {
		return drawioCloneEdit{}, err
	}
	return drawioCloneEdit{at: at, inserted: inserted, ids: ids,
		beforeText: drawioCloneTextIdentity([]byte(raw)), afterText: drawioCloneTextIdentity(serialized)}, nil
}

func (v *view) commitDrawioClone(buffer *text.Buffer, edit drawioCloneEdit, caret text.Range) (text.Range, error) {
	v.flushing = true
	defer func() { v.flushing = false }()
	if err := buffer.BeginGroup(); err != nil {
		return text.Range{}, err
	}
	if err := buffer.ReplaceUTF16(edit.at, edit.at, edit.inserted); err != nil {
		buffer.CancelGroup()
		return text.Range{}, err
	}
	// Keep numerical UTF-16 endpoints, with the same surrogate clamp as the
	// other graph edits. Set them before EndGroup so redo reapplies them.
	boundary := func(offset int) int {
		if _, err := buffer.SliceUTF16(offset, offset); errors.Is(err, text.ErrSurrogate) {
			return offset - 1
		}
		return offset
	}
	selection := text.Range{Anchor: boundary(caret.Anchor), Head: boundary(caret.Head)}
	if err := buffer.SetSelection(selection.Anchor, selection.Head); err != nil {
		buffer.CancelGroup()
		return text.Range{}, err
	}
	if !buffer.EndGroup() {
		return text.Range{}, errors.New("图形克隆没有生成编辑事务")
	}
	v.lastSynced = ^uint64(0)
	if v.host.Changed != nil {
		v.host.Changed()
	}
	return selection, nil
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
	ordinaryFieldWritten := false
	refuse := func(err error) bool {
		if sameOwner() {
			outcome := "原文保持不变。"
			if ordinaryFieldWritten {
				outcome = "普通字段已写回，可单独撤销；克隆未完成。"
				if !buffer.CanUndo() {
					outcome = "普通字段已写回，未保留该编辑的撤销记录。克隆未完成。"
				}
			}
			v.addWarning("未克隆：" + err.Error() + "；" + outcome)
			v.report(err)
		}
		return true
	}
	if buffer == nil {
		return refuse(errors.New("没有可写回的文档缓冲区"))
	}
	if v.host.ReadOnly {
		return refuse(errors.New("文档当前只读"))
	}
	if len(targets) < 2 {
		return refuse(errors.New("请先用 Ctrl/Cmd 点击选择至少两个要克隆的图形"))
	}
	if buffer.InGroup() {
		return refuse(text.ErrGroup)
	}
	if v.flushing {
		return refuse(ErrDraftConflict)
	}
	if v.CompositionActive() {
		return refuse(errors.New("请先确认正在输入的文字，再克隆图形"))
	}
	version, caret := buffer.Version(), buffer.Selection()
	sameSource := func() bool {
		return sameOwner() && !v.host.ReadOnly && !buffer.InGroup() && !v.CompositionActive() &&
			buffer.Version() == version && buffer.Selection() == caret
	}
	v.ensureSynced(nil)
	if !sameSource() || v.syncFailed || v.syncedBuffer != buffer || v.lastSynced != version {
		return refuse(ErrDraftConflict)
	}
	if err := v.writable(); err != nil {
		return refuse(err)
	}
	edit, err := planDrawioClone(buffer, page, targets)
	if err != nil {
		return refuse(err)
	}
	receipt, err := v.flushOrdinaryField()
	if err != nil {
		return true
	}
	ordinaryFieldWritten = receipt.written
	expectedVersion := version
	if receipt.written {
		expectedVersion++
	}
	if !sameOwner() || receipt.version != expectedVersion {
		return refuse(ErrDraftConflict)
	}
	version, caret = receipt.version, receipt.selection
	if !sameSource() || v.flushing {
		return refuse(ErrDraftConflict)
	}
	if receipt.written {
		edit, err = planDrawioClone(buffer, page, targets)
		if err != nil {
			return refuse(err)
		}
	}
	if !sameSource() {
		return refuse(ErrDraftConflict)
	}
	selection, err := v.commitDrawioClone(buffer, edit, caret)
	if err != nil {
		return refuse(err)
	}
	if !sameOwner() || v.host.ReadOnly || buffer.InGroup() || v.CompositionActive() ||
		buffer.Version() != version+1 || buffer.Selection() != selection {
		return true
	}
	ids := edit.ids
	v.cancelGraphGesture()
	v.setDrawioSelection(ids, ids[len(ids)-1])
	publishedEpoch := v.drawSelection.epoch
	v.lastSynced = ^uint64(0)
	v.ensureSynced(nil)
	if v.host.Buffer != buffer || v.releaseEpoch != released || v.pageEpoch != pageEpoch || v.pageIndex != page ||
		v.host.ReadOnly || buffer.InGroup() || v.CompositionActive() || buffer.Version() != version+1 || buffer.Selection() != selection ||
		!v.drawioCloneSourceCurrent(buffer) || v.drawSelection.epoch != publishedEpoch ||
		!slices.Equal(v.drawSelection.ids, ids) || v.selectedID != ids[len(ids)-1] ||
		drawioCloneTextIdentity(v.diagram.Source) != edit.afterText {
		return true
	}

	// Buffer trims oldest first. At this unchanged commit version, any undo
	// depth implies that this newest transaction survived, even after a trim.
	message := fmt.Sprintf("已克隆 %d 个图形及内部连线到当前页", len(ids))
	if !buffer.CanUndo() {
		clear(v.drawCloneHistory)
		v.drawCloneHistory = nil
		message += "。未保留此次撤销记录，克隆不可撤销。"
	} else {
		if n := len(v.drawCloneHistory); n > 0 && !v.drawCloneHistory[n-1].owns(buffer, released, pageEpoch, page) {
			clear(v.drawCloneHistory)
			v.drawCloneHistory = nil
		}
		var recorded bool
		previous := len(v.drawCloneHistory)
		v.drawCloneHistory, recorded = appendDrawioCloneHistory(v.drawCloneHistory, drawioCloneHistory{
			buffer: buffer, page: page, pageEpoch: pageEpoch, release: released,
			beforeText: edit.beforeText, afterText: edit.afterText, beforeIDs: targets, afterIDs: ids,
			beforePrimary: slices.Index(targets, primary), afterEpoch: publishedEpoch,
		})
		message += "；可撤销。"
		if !recorded {
			message += "此次图形选区未保留历史。"
		} else if len(v.drawCloneHistory) <= previous {
			message += "部分较早的图形选区记录已因历史上限移除。"
		}
	}
	if selection != caret {
		message += "源码光标已移到前一个有效的 UTF-16 边界。"
	}
	v.warnings = append([]string{message}, v.warnings...)
	v.canvasHandle.Focus()
	return true
}

func (record drawioCloneHistory) owns(buffer *text.Buffer, release, pageEpoch uint64, page int) bool {
	return record.buffer == buffer && record.release == release && record.pageEpoch == pageEpoch && record.page == page
}

func (record drawioCloneHistory) beforePrimaryID() string {
	if record.beforePrimary >= 0 && record.beforePrimary < len(record.beforeIDs) {
		return record.beforeIDs[record.beforePrimary]
	}
	return ""
}

func (v *view) drawioCloneSourceCurrent(buffer *text.Buffer) bool {
	return !v.syncFailed && v.diagram != nil && v.syncedBuffer == buffer && v.lastSynced == buffer.Version()
}

type drawioCloneReplay struct {
	record    *drawioCloneHistory
	undo      bool
	version   uint64
	epoch     uint64
	selection text.Range
}

// Buffer history has moved one version, while the canvas still has the prior
// parsed XML and selection. Prove that boundary before calling Host.Changed.
func (v *view) beginDrawioCloneHistory(buffer *text.Buffer, release, pageEpoch uint64, page int, undo bool) *drawioCloneReplay {
	if len(v.drawCloneHistory) == 0 || v.kind != kindDrawio || buffer == nil || v.syncFailed || v.diagram == nil ||
		v.syncedBuffer != buffer || v.lastSynced == ^uint64(0) || v.lastSynced+1 != buffer.Version() {
		return nil
	}
	var prior drawioCloneTextID
	for i := len(v.drawCloneHistory) - 1; i >= 0; i-- {
		record := &v.drawCloneHistory[i]
		if !record.owns(buffer, release, pageEpoch, page) {
			continue
		}
		epoch, ids, primary, expected := record.undoEpoch, record.beforeIDs, record.beforePrimaryID(), record.beforeText
		if undo {
			epoch, ids, expected = record.afterEpoch, record.afterIDs, record.afterText
			primary = ids[len(ids)-1]
		}
		if !slices.Equal(v.drawSelection.ids, ids) || v.selectedID != primary {
			continue
		}
		if prior == "" {
			prior = drawioCloneTextIdentity(v.diagram.Source)
		}
		if prior != expected || (v.drawSelection.epoch != epoch && !v.knownDrawioCloneBoundary(buffer, release, pageEpoch, page, prior)) {
			continue
		}
		return &drawioCloneReplay{record: record, undo: undo, version: buffer.Version(), epoch: v.drawSelection.epoch, selection: buffer.Selection()}
	}
	return nil
}

// Replaying a later clone may produce a new epoch for an intermediate XML
// state. Accept an epoch already witnessed at that exact source with the same
// owner and ordered IDs. An explicit selection change has no such witness.
func (v *view) knownDrawioCloneBoundary(buffer *text.Buffer, release, pageEpoch uint64, page int, source drawioCloneTextID) bool {
	for _, record := range v.drawCloneHistory {
		if !record.owns(buffer, release, pageEpoch, page) {
			continue
		}
		if record.afterText == source && record.afterEpoch == v.drawSelection.epoch &&
			slices.Equal(record.afterIDs, v.drawSelection.ids) {
			return true
		}
		if record.beforeText == source && record.undoEpoch == v.drawSelection.epoch &&
			slices.Equal(record.beforeIDs, v.drawSelection.ids) {
			return true
		}
	}
	return false
}

func (v *view) finishDrawioCloneHistory(replay *drawioCloneReplay, buffer *text.Buffer, release, pageEpoch uint64, page int) {
	if replay == nil || v.host.Buffer != buffer || v.releaseEpoch != release || v.pageEpoch != pageEpoch || v.pageIndex != page ||
		v.host.ReadOnly || buffer.InGroup() || v.CompositionActive() || buffer.Version() != replay.version ||
		buffer.Selection() != replay.selection || !v.drawioCloneSourceCurrent(buffer) {
		return
	}
	record := replay.record
	if !record.owns(buffer, release, pageEpoch, page) {
		return
	}
	ids, primary, expected := record.afterIDs, record.afterIDs[len(record.afterIDs)-1], record.afterText
	if replay.undo {
		// Sync prunes the now-absent cloned IDs in one selection change.
		if v.drawSelection.epoch != replay.epoch+1 || len(v.drawSelection.ids) != 0 || v.selectedID != "" {
			return
		}
		ids, primary, expected = record.beforeIDs, record.beforePrimaryID(), record.beforeText
	} else if v.drawSelection.epoch != replay.epoch {
		return
	}
	current := v.currentPage()
	if current == nil || drawioCloneTextIdentity(v.diagram.Source) != expected {
		return
	}
	missing := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		missing[id] = struct{}{}
	}
	for _, node := range current.Model.Nodes {
		delete(missing, node.ID)
		if len(missing) == 0 {
			break
		}
	}
	if len(missing) != 0 {
		return
	}
	v.setDrawioSelection(ids, primary)
	if replay.undo {
		record.undoEpoch = v.drawSelection.epoch
	} else {
		record.afterEpoch = v.drawSelection.epoch
	}
}
