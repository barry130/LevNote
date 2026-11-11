package graphview

import (
	"errors"
	"fmt"
	"path/filepath"

	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
	"levnote.local/native/formats/mindmap"
	"levnote.local/native/ui/editor"
)

var ErrDraftConflict = errors.New("节点的源字段已改变，草稿仍保留；可恢复为新文档")

// A field draft is independent of the parsed graph cache. It remembers the
// exact authoritative field, not a document version: unrelated source changes
// can be preserved while only this field is applied. Stable page+node IDs are
// required when a diagram's page ordering changes.
type fieldDraft struct {
	nodeID, pageID, pageName string
	pageIndex, pageCount     int
	version                  uint64
	commits                  uint64
	buffer                   *text.Buffer
	field                    nodeField
	sourcePresent            bool
	base, sourceBase         string
	input                    *editor.Editor
	pendingVersion           uint64
	pendingKnown, pending    bool
}

// Only input identity crosses a host callback. The ordinary helper may update
// its own source base/version after proving the receipt; those are not identity.
type fieldIdentity struct {
	draft          *fieldDraft
	input          *editor.Editor
	buffer         *text.Buffer
	version        uint64
	selection      text.Range
	nodeID, pageID string
	field          nodeField
	owner          *text.Buffer
}

type fieldFlushReceipt struct {
	written   bool
	version   uint64
	selection text.Range
}

func (v *view) fieldIdentity() fieldIdentity {
	d := v.draft
	if d == nil {
		return fieldIdentity{}
	}
	return fieldIdentity{d, d.input, d.input.Buffer, d.input.Buffer.Version(), d.input.Buffer.Selection(), d.nodeID, d.pageID, d.field, d.buffer}
}

func (v *view) sameFieldIdentity(id fieldIdentity) bool {
	return v.draft == id.draft && (id.draft == nil || id.draft.input == id.input && id.input.Buffer == id.buffer && id.buffer.Version() == id.version && id.buffer.Selection() == id.selection && id.draft.nodeID == id.nodeID && id.draft.pageID == id.pageID && id.draft.field == id.field && id.draft.buffer == id.owner)
}

func (v *view) pendingDraft() bool {
	if v.draft == nil {
		return false
	}
	d := v.draft
	_, composing := d.input.MarkedRange()
	if composing {
		return true
	}
	version := d.input.Buffer.Version()
	if !d.pendingKnown || d.pendingVersion != version {
		d.pending = d.input.Buffer.String() != d.base
		d.pendingVersion, d.pendingKnown = version, true
	}
	return d.pending
}

func (v *view) setDraftError(err error) {
	if err == nil {
		v.draftError = nil
		return
	}
	changed := v.draftError == nil || v.draftError.Error() != err.Error()
	v.draftError = err
	if changed {
		v.report(err)
	}
}

func (v *view) refreshDraft() {
	if v.pendingGraphDraft() || v.flushing {
		return
	}
	value, source, present := "", "", true
	field := v.field
	if v.kind == kindMindmap {
		node := v.selectedMindNode()
		if node == nil {
			v.clearDraft()
			return
		}
		value, present = mindmapField(node, field)
		source = value
	} else {
		node := v.selectedDrawioNode()
		if node == nil {
			v.clearDraft()
			return
		}
		value, source = node.PlainText, node.Text
		field = fieldDrawioLabel
	}
	pageID, pageName, pageCount := "", "", 0
	if page := v.currentPage(); page != nil {
		pageID, pageName, pageCount = page.ID, page.Name, len(v.diagram.Pages)
	}
	if d := v.draft; d != nil && d.buffer == v.host.Buffer && d.nodeID == v.selectedID && d.pageID == pageID && d.field == field && d.sourcePresent == present && d.base == value && d.sourceBase == source {
		d.version = v.host.Buffer.Version()
		d.pageIndex, d.pageName, d.pageCount = v.pageIndex, pageName, pageCount
		return
	}
	v.clearDraft()
	input := editor.New(text.New(value))
	input.ShapeLimit = 512
	input.LineNumbers = false
	input.Label, input.MinHeight, input.FontSize = field.inputLabel(), 48, 13
	input.OnError = v.setDraftError
	d := &fieldDraft{nodeID: v.selectedID, pageID: pageID, pageName: pageName, pageIndex: v.pageIndex, pageCount: pageCount, version: v.host.Buffer.Version(), buffer: v.host.Buffer, field: field, sourcePresent: present, base: value, sourceBase: source, input: input, pendingKnown: true, pendingVersion: input.Buffer.Version()}
	v.draft = d
	input.OnChange = func() {
		if v.draft != d {
			return
		}
		v.invalidateTagActions()
		if _, composing := input.MarkedRange(); composing {
			return
		}
		v.clearTagFocusWarning()
		v.setDraftError(v.Flush())
	}
}

func (v *view) clearDraft() {
	v.linkGesture = fieldLinkTarget{}
	v.invalidateTagActions()
	d := v.draft
	v.draft = nil
	if d != nil {
		d.input.ReleaseView()
	}
	v.draftError = nil
}

func (v *view) focusLabel() {
	if v.draft != nil {
		v.draft.input.Focus()
	}
}

func (v *view) focusTopic() {
	if v.kind == kindMindmap && !v.selectField(fieldTopic) {
		return
	}
	v.focusLabel()
}

func (v *view) draftPage(document *drawio.Document, draft *fieldDraft) (int, error) {
	if draft.pageID != "" {
		found := -1
		for i, page := range document.Pages {
			if page.ID == draft.pageID {
				if found >= 0 {
					return 0, ErrDraftConflict
				}
				found = i
			}
		}
		if found < 0 {
			return 0, ErrDraftConflict
		}
		return found, nil
	}
	// A standalone model is its own sole page. Multiple id-less pages cannot
	// be safely associated after any source edit; do not guess by index/name.
	if len(document.Pages) == 1 && draft.pageCount == 1 && document.Pages[0].ID == "" {
		return 0, nil
	}
	if v.host.Buffer.Version() == draft.version && draft.pageIndex < len(document.Pages) && document.Pages[draft.pageIndex].ID == "" && document.Pages[draft.pageIndex].Name == draft.pageName {
		return draft.pageIndex, nil
	}
	return 0, ErrDraftConflict
}

func (v *view) Flush() error {
	// This must precede the reentrancy guard: synchronous Changed callbacks
	// cannot save/switch through the private Add -> ordinary-field path.
	if v.pendingTagDraft() {
		v.setTagError(errTagPending)
		return errTagPending
	}
	if v.flushing {
		return nil
	}
	receipt, err := v.flushOrdinaryField()
	v.lastFieldReceipt = receipt
	return err
}

func (v *view) flushOrdinaryField() (receipt fieldFlushReceipt, result error) {
	if v.flushing {
		return receipt, ErrDraftConflict
	}
	v.flushing = true
	defer func() { v.flushing = false }()
	buffer, released, epoch := v.host.Buffer, v.releaseEpoch, v.tagEpoch
	selected, selectedField, selectedPage := v.selectedID, v.field, v.pageIndex
	id := v.fieldIdentity()
	var draftState fieldDraft
	if id.draft != nil {
		draftState = *id.draft
	}
	stillSame := func() bool {
		d := id.draft
		return v.host.Buffer == buffer && v.releaseEpoch == released && v.tagEpoch == epoch && v.selectedID == selected && v.field == selectedField && v.pageIndex == selectedPage && v.sameFieldIdentity(id) && (d == nil || d.base == draftState.base && d.sourceBase == draftState.sourceBase && d.sourcePresent == draftState.sourcePresent && d.version == draftState.version && d.commits == draftState.commits && d.pageIndex == draftState.pageIndex && d.pageCount == draftState.pageCount && d.pageName == draftState.pageName)
	}
	defer func() {
		if stillSame() {
			v.setDraftError(result)
		}
	}()
	if buffer != nil {
		receipt.version, receipt.selection = buffer.Version(), buffer.Selection()
	}
	if v.previewSizeError() != nil {
		// Discard obsolete render state even if readonly/identity/group checks
		// below reject a pending field. A clean draft may be released here.
		v.ensureSynced(nil)
	}
	d := v.draft
	if d == nil {
		return receipt, nil
	}
	if err := d.input.Flush(); err != nil {
		return receipt, err
	}
	if !stillSame() || buffer != nil && (buffer.Version() != receipt.version || buffer.Selection() != receipt.selection) {
		return receipt, ErrDraftConflict
	}
	value := d.input.Buffer.String()
	if value == d.base {
		return receipt, nil
	}
	if v.host.Buffer == nil {
		err := errors.New("没有可写回的文档缓冲区")
		return receipt, err
	}
	if v.host.ReadOnly {
		err := errors.New("文档当前只读，节点草稿仍保留")
		return receipt, err
	}
	if v.host.Buffer != d.buffer {
		return receipt, ErrDraftConflict
	}
	if v.host.Buffer.InGroup() {
		return receipt, text.ErrGroup
	}
	if err := v.previewSizeError(); err != nil {
		v.ensureSynced(nil)
		return receipt, err
	}
	raw := v.host.Buffer.String()
	var serialized []byte
	var err error
	page := d.pageIndex
	switch v.kind {
	case kindMindmap:
		var candidate *mindmap.Document
		candidate, _, err = mindmap.Parse([]byte(raw), mindmap.ParseOptions{})
		if err == nil {
			node, ok := candidate.FindNode(d.nodeID)
			current, present := mindmapField(node, d.field)
			if !ok || current != d.sourceBase || present != d.sourcePresent {
				err = ErrDraftConflict
			} else {
				err = applyMindmapField(candidate, d.nodeID, d.field, value)
				if err == nil {
					serialized, err = candidate.Serialize(mindmap.SerializeOptions{})
				}
			}
		}
	case kindDrawio:
		var candidate *drawio.Document
		candidate, _, err = drawio.Parse([]byte(raw), drawio.ParseOptions{})
		if err == nil {
			page, err = v.draftPage(candidate, d)
			if err == nil {
				p := candidate.Pages[page]
				found := false
				for _, node := range p.Model.Nodes {
					if node.ID == d.nodeID {
						found = true
						if node.Text != d.sourceBase {
							err = ErrDraftConflict
						}
						break
					}
				}
				if !found {
					err = ErrDraftConflict
				}
				if err == nil {
					err = candidate.UpdateText(page, d.nodeID, value)
				}
				if err == nil {
					serialized, err = candidate.Serialize()
				}
			}
		}
	}
	if err == nil {
		value := string(serialized)
		expectedVersion, expectedSelection := receipt.version, receipt.selection
		written := value != raw
		if written {
			expectedVersion++
			end := text.UTF16Len(value)
			expectedSelection = text.Range{Anchor: end, Head: end}
		}
		err = v.commitBuffer(value)
		if err == nil {
			receipt.written, receipt.version, receipt.selection = written, expectedVersion, expectedSelection
			// Nothing may restore old selectedID/base/cache until Changed has
			// returned and this exact 0/1-version source receipt is proved.
			if !stillSame() || v.host.ReadOnly || buffer.InGroup() || buffer.Version() != expectedVersion || buffer.Selection() != expectedSelection || buffer.String() != value {
				return receipt, ErrDraftConflict
			}
		}
	}
	if err != nil {
		return receipt, err
	}
	d.base, d.sourceBase, d.version = value, value, v.host.Buffer.Version()
	d.pendingKnown, d.pending, d.pendingVersion = true, false, d.input.Buffer.Version()
	d.commits++
	d.sourcePresent = d.field != fieldHyperLink || value != ""
	d.pageIndex = page
	v.selectedID = d.nodeID
	v.setPageIndex(page)
	v.lastSynced = ^uint64(0)
	v.setDraftError(nil)
	v.ensureSynced(nil)
	return receipt, nil
}

// RecoverDraft never tries the conflicting mutation again and never writes the
// authoritative source. The host opens this plain text as a separate document.
func (v *view) RecoverDraft() error {
	fieldPending, tagPending := v.pendingDraft(), v.pendingTagDraft()
	if !fieldPending && !tagPending {
		return nil
	}
	d, tag := v.draft, v.tagDraft
	fieldID, tagID := v.fieldIdentity(), v.tagInputIdentity()
	buffer, released := v.host.Buffer, v.releaseEpoch
	if v.host.RecoverDraft == nil {
		err := errors.New("当前工作区不能创建草稿副本；节点草稿仍保留")
		v.setDraftError(err)
		return err
	}
	name := filepath.Base(v.host.Path)
	if name == "" || name == "." {
		name = "图形"
	}
	content := ""
	if fieldPending {
		// Preserve the established field-only recovery envelope byte for byte.
		content = fmt.Sprintf("%s\n节点 ID: %s\n页面 ID: %s\n字段: %s\n原字段存在: %t\n\n%s", name, d.nodeID, d.pageID, d.field.inputLabel(), d.sourcePresent, d.input.Buffer.String())
	}
	if tagPending {
		if content != "" {
			content += "\n\n--- 未添加的标签输入 ---\n"
		} else {
			content = name + "\n未添加的标签输入\n"
		}
		content += fmt.Sprintf("节点 ID: %s\n字段: 新标签（尚未添加）\n\n%s", tag.nodeID, tag.input.Buffer.String())
	}
	if err := v.host.RecoverDraft(name+"-节点草稿.txt", content); err != nil {
		v.setDraftError(err)
		return err
	}
	sameField, sameTag := v.sameFieldIdentity(fieldID), v.sameTagInput(tagID)
	if fieldPending && sameField {
		v.clearDraft()
	}
	if tagPending && sameTag {
		v.clearTagDraft()
	}
	// Recovery activates another document. Release the old graph's heavy
	// caches now; the retained view will rebuild if the user returns.
	if sameField && sameTag && v.host.Buffer == buffer && v.releaseEpoch == released {
		v.Release()
	}
	return nil
}

func (v *view) changePage(index int) bool {
	if err := v.Flush(); err != nil {
		return false
	}
	v.ensureSynced(nil)
	if v.diagram == nil || index < 0 || index >= len(v.diagram.Pages) {
		return false
	}
	v.setPageIndex(index)
	v.selectedID = ""
	v.cancelGraphGesture()
	v.clearDraft()
	v.fitted = false
	v.lastSynced = ^uint64(0)
	v.ensureSynced(nil)
	return true
}
