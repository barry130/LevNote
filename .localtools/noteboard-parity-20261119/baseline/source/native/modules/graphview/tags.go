package graphview

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"levnote.local/native/core/text"
	"levnote.local/native/formats/mindmap"
	"levnote.local/native/ui/editor"
)

const (
	maxTagCount    = 64
	maxTagBytes    = 8192
	maxNewTagBytes = 512
	tagAdd         = -1
	tagClear       = -2
	tagDiscard     = -3
	tagRecover     = -4
)

var (
	errTagPending = errors.New("新标签尚未添加；请添加或放弃输入")
	errTagStale   = errors.New("标签来源或操作目标已改变，请重新操作；输入仍保留")
	errTagBudget  = errors.New("标签超过64项或8KiB内容预算；完整内容仍保留，请在源码中修改")
	errTagInput   = errors.New("新标签须为非空单行UTF-8文字，最多512字节；输入仍保留")
	errTagConfirm = errors.New("请先确认正在输入的文字，再移动到属性操作")
)

type tagDraft struct {
	buffer *text.Buffer
	nodeID string
	input  *editor.Editor
}

type tagPresentation struct {
	buffer          *text.Buffer
	version         uint64
	nodeID          string
	loaded, present bool
	count           int
	summary         string
	rows            []string
	err             error
}

type tagInputIdentity struct {
	draft     *tagDraft
	input     *editor.Editor
	buffer    *text.Buffer
	version   uint64
	selection text.Range
	owner     *text.Buffer
	nodeID    string
}

type tagGesture struct {
	buffer                    *text.Buffer
	version                   uint64
	selection                 text.Range
	nodeID                    string
	epoch                     uint64
	action                    int
	tags                      []mindmap.Tag
	present, active, released bool
	open                      bool
	input                     tagInputIdentity
	field                     fieldIdentity
	err                       error
}

func tagValues(node *mindmap.Node) ([]mindmap.Tag, bool) {
	if node == nil || node.Tags == nil {
		return nil, false
	}
	return *node.Tags, true
}

// Include retained Raw AND decoded fields, with a minimum per tag/member.
// Check each component against remaining bytes before adding; no marshal/clone.
func tagBudget(values []mindmap.Tag) error {
	return tagProspectiveBudget(values, nil)
}

func tagProspectiveBudget(values []mindmap.Tag, added *string) error {
	count := len(values)
	if added != nil {
		count++
	}
	if count > maxTagCount {
		return errTagBudget
	}
	remaining := maxTagBytes
	take := func(size int) bool {
		if size > remaining {
			return false
		}
		remaining -= size
		return true
	}
	for _, tag := range values {
		if !take(1) || !take(len(tag.Raw)) || !take(len(tag.Text)) || !take(len(tag.ClassName)) {
			return errTagBudget
		}
		for _, object := range []mindmap.Object{tag.Style, tag.Extra} {
			for _, member := range object.Members {
				if !take(1) || !take(len(member.Key)) || !take(len(member.Raw)) {
					return errTagBudget
				}
			}
		}
	}
	if added != nil && (!take(1) || !take(len(*added))) {
		return errTagBudget
	}
	return nil
}

func tagListsEqual(a, b []mindmap.Tag, ownFieldWrite bool) bool {
	if len(a) != len(b) {
		return false
	}
	objectsEqual := func(a, b mindmap.Object) bool {
		if len(a.Members) != len(b.Members) {
			return false
		}
		for i, x := range a.Members {
			y := b.Members[i]
			if x.Key != y.Key || !bytes.Equal(x.Raw, y.Raw) {
				return false
			}
		}
		return true
	}
	for i, x := range a {
		y := b[i]
		if x.Object != y.Object || x.Text != y.Text || x.ClassName != y.ClassName || x.ClassNameSet != y.ClassNameSet || x.StyleSet != y.StyleSet || !objectsEqual(x.Style, y.Style) || !objectsEqual(x.Extra, y.Extra) {
			return false
		}
		// Only a proven exact ordinary-field receipt may canonicalize the
		// redundant whole-tag envelope. All modeled/unknown fields stay exact.
		if !ownFieldWrite && !bytes.Equal(x.Raw, y.Raw) {
			return false
		}
	}
	return true
}

func (v *view) pendingTagDraft() bool {
	if v.tagDraft == nil {
		return false
	}
	_, composing := v.tagDraft.input.MarkedRange()
	return composing || v.tagDraft.input.Buffer.LenBytes() != 0
}

func (v *view) pendingGraphDraft() bool { return v.pendingDraft() || v.pendingTagDraft() }

func (v *view) invalidateTagActions() {
	v.tagEpoch++
	v.tagPress = tagGesture{}
}

func (v *view) clearTagDraft() {
	v.invalidateTagActions()
	d := v.tagDraft
	v.tagDraft = nil
	if d != nil {
		d.input.ReleaseView()
	}
}

func (v *view) refreshTagDraft() {
	if v.kind != kindMindmap || !v.tagOpen || v.pendingTagDraft() || v.flushing {
		return
	}
	if v.syncFailed || v.selectedMindNode() == nil {
		v.clearTagDraft()
		return
	}
	if d := v.tagDraft; d != nil && d.buffer == v.host.Buffer && d.nodeID == v.selectedID {
		return
	}
	v.clearTagDraft()
	input := editor.New(text.New(""))
	input.Label, input.ShapeLimit, input.MinHeight, input.FontSize = "新标签（单行，最多512字节）", 512, 48, 13
	input.LineNumbers = false
	d := &tagDraft{buffer: v.host.Buffer, nodeID: v.selectedID, input: input}
	v.tagDraft = d
	input.OnError = v.setTagError
	input.OnChange = func() {
		if v.tagDraft != d {
			return
		}
		v.invalidateTagActions()
		v.clearTagFocusWarning()
		v.clearTagPendingWarning()
	}
}

func (v *view) clearTagPendingWarning() {
	if v.pendingTagDraft() {
		return
	}
	if v.tagError == errTagPending {
		v.setTagError(nil)
	}
	if v.draftError == errTagPending {
		v.setDraftError(nil)
	}
}

// A confirmed input removes only the focus-move warning. Validation/refusal
// errors remain until their own action succeeds; another live composition must
// still keep its guidance, including one in the ordinary node field.
func (v *view) clearTagFocusWarning() {
	if v.tagDraft != nil {
		if _, composing := v.tagDraft.input.MarkedRange(); composing {
			return
		}
	}
	if v.draft != nil {
		if _, composing := v.draft.input.MarkedRange(); composing {
			return
		}
	}
	if v.tagError == errTagConfirm {
		v.setTagError(nil)
	}
	if v.draftError == errTagConfirm {
		v.setDraftError(nil)
	}
}

func (v *view) tagInputIdentity() tagInputIdentity {
	if v.tagDraft == nil {
		return tagInputIdentity{}
	}
	d := v.tagDraft
	return tagInputIdentity{d, d.input, d.input.Buffer, d.input.Buffer.Version(), d.input.Buffer.Selection(), d.buffer, d.nodeID}
}

func (v *view) sameTagInput(id tagInputIdentity) bool {
	return v.tagDraft == id.draft && (id.draft == nil || id.draft.input == id.input && id.input.Buffer == id.buffer && id.buffer.Version() == id.version && id.buffer.Selection() == id.selection && id.draft.buffer == id.owner && id.draft.nodeID == id.nodeID)
}

func (v *view) setTagError(err error) {
	previous := v.tagError
	v.tagError = err
	if err != nil && (previous == nil || previous.Error() != err.Error()) {
		v.report(err)
	}
}

func tagLabel(tag mindmap.Tag, ordinal int) string {
	value := tag.Text
	if value == "" {
		value = "（空字符串）"
	}
	return fmt.Sprintf("%d. %s", ordinal+1, markerText(value, 192))
}

func (v *view) tagPresentation() tagPresentation {
	buffer := v.host.Buffer
	if buffer == nil || v.syncFailed || v.mind == nil {
		v.tagInfo = tagPresentation{summary: "完整标签请在源码中查看。", err: errTagStale}
		return v.tagInfo
	}
	if info := v.tagInfo; info.loaded && info.buffer == buffer && info.version == buffer.Version() && info.nodeID == v.selectedID {
		return info
	}
	info := tagPresentation{buffer: buffer, version: buffer.Version(), nodeID: v.selectedID, loaded: true}
	node := v.selectedMindNode()
	if node == nil {
		info.summary, info.err = "完整标签请在源码中查看。", errTagStale
	} else {
		values, present := tagValues(node)
		info.present, info.count = present, len(values)
		info.err = tagBudget(values)
		var summary strings.Builder
		fmt.Fprintf(&summary, "%d 个标签", len(values))
		if len(values) == 0 {
			if present {
				summary.WriteString("（已清空）")
			} else {
				summary.WriteString("（尚未设置）")
			}
		}
		for i, tag := range values {
			if i == maxTagCount || summary.Len() >= 450 {
				break
			}
			summary.WriteString("；")
			summary.WriteString(markerText(tagLabel(tag, i), min(192, 480-summary.Len())))
			if info.err == nil {
				info.rows = append(info.rows, tagLabel(tag, i))
			}
		}
		info.summary = markerText(summary.String()+"；完整列表见源码。", 512)
		// Rows have their own bounded labels; summary exhaustion must not
		// hide the remaining editable ordinals.
		if info.err == nil {
			for i := len(info.rows); i < len(values); i++ {
				info.rows = append(info.rows, tagLabel(values[i], i))
			}
		}
	}
	v.tagInfo = info
	return info
}

func (v *view) captureTag(action int, info tagPresentation) tagGesture {
	g := tagGesture{buffer: info.buffer, version: info.version, nodeID: info.nodeID, epoch: v.tagEpoch, action: action, active: true, open: v.tagOpen, input: v.tagInputIdentity(), field: v.fieldIdentity()}
	if info.buffer == nil || info.buffer != v.host.Buffer || info.version != info.buffer.Version() || v.syncedBuffer != info.buffer || v.lastSynced != info.version || info.nodeID != v.selectedID || v.syncFailed || v.mind == nil || !v.tagOpen {
		g.err = errTagStale
		return g
	}
	g.selection = info.buffer.Selection()
	node, ok := v.mind.FindNode(g.nodeID)
	if !ok {
		g.err = errTagStale
		return g
	}
	values, present := tagValues(node)
	if err := tagBudget(values); err != nil {
		g.err = err
		return g
	}
	g.present = present
	g.tags, _, g.err = v.mind.Tags(g.nodeID)
	return g
}

func (v *view) sameTagGesture(g tagGesture) bool {
	return g.active && g.epoch == v.tagEpoch && g.open == v.tagOpen && (g.open || g.action == tagRecover) && g.buffer != nil && g.buffer == v.host.Buffer && g.nodeID == v.selectedID && v.sameTagInput(g.input) && v.sameFieldIdentity(g.field)
}

func (v *view) activateTag(g tagGesture) (err error) {
	fieldWritten := false
	defer func() {
		if err != nil && fieldWritten {
			err = fmt.Errorf("节点字段已写回，标签未修改：%w", err)
		}
	}()
	if g.err != nil {
		return g.err
	}
	if !v.sameTagGesture(g) || g.buffer.Version() != g.version || g.buffer.Selection() != g.selection {
		return errTagStale
	}
	if v.host.ReadOnly {
		return errors.New("当前文档只读，不能修改标签")
	}
	if g.buffer.InGroup() {
		return text.ErrGroup
	}
	if err := v.previewSizeError(); err != nil {
		return err
	}
	if g.action != tagAdd && v.pendingTagDraft() {
		return errTagPending
	}
	newValue := ""
	if g.action == tagAdd {
		if g.input.draft == nil || g.input.draft.buffer != g.buffer || g.input.draft.nodeID != g.nodeID || g.input.buffer.LenBytes() == 0 || g.input.buffer.LenBytes() > maxNewTagBytes {
			return errTagInput
		}
		newValue = g.input.buffer.String()
		if !utf8.ValidString(newValue) || strings.ContainsAny(newValue, "\r\n") {
			return errTagInput
		}
		// Confirm a staged composition explicitly. Flush may close only its
		// own input group; bytes/version/selection and owners must stay exact.
		if err = g.input.draft.input.Flush(); err != nil {
			return err
		}
		if !v.sameTagGesture(g) || g.input.buffer.InGroup() || g.input.buffer.String() != newValue || g.buffer.Version() != g.version || g.buffer.Selection() != g.selection {
			return errTagStale
		}
	}
	// Both source reads happen only on a validated activation under16MiB.
	before, _, err := mindmap.Parse([]byte(g.buffer.String()), mindmap.ParseOptions{})
	if err != nil {
		return err
	}
	node, ok := before.FindNode(g.nodeID)
	values, present := tagValues(node)
	if !ok || present != g.present || !tagListsEqual(values, g.tags, false) {
		return errTagStale
	}
	if err = tagBudget(values); err != nil {
		return err
	}
	if g.action == tagAdd {
		if err = tagProspectiveBudget(values, &newValue); err != nil {
			return err
		}
	} else if g.action != tagClear && (g.action < 0 || g.action >= len(values)) {
		return errTagStale
	}
	node, before = nil, nil // No parsed node or document crosses Flush/Changed.
	values = nil
	var receipt fieldFlushReceipt
	if g.action == tagAdd {
		receipt, err = v.flushOrdinaryField()
	} else {
		if err = v.Flush(); err == nil {
			receipt = v.lastFieldReceipt
		}
	}
	fieldWritten = receipt.written
	if err != nil {
		return err
	}
	if !v.sameTagGesture(g) || g.buffer.Version() != receipt.version || g.buffer.Selection() != receipt.selection {
		return errTagStale
	}
	candidate, _, err := mindmap.Parse([]byte(g.buffer.String()), mindmap.ParseOptions{})
	if err != nil {
		return err
	}
	node, ok = candidate.FindNode(g.nodeID)
	values, present = tagValues(node)
	if !ok || present != g.present || !tagListsEqual(values, g.tags, receipt.written) {
		return errTagStale
	}
	if err = tagBudget(values); err != nil {
		return err
	}
	switch g.action {
	case tagAdd:
		// A field write can change redundant Raw envelope size. Recheck the
		// actual post-Flush prospective list before the pure API clones it.
		if err = tagProspectiveBudget(values, &newValue); err != nil {
			return err
		}
		err = candidate.AddTag(g.nodeID, newValue)
	case tagClear:
		if len(values) == 0 {
			return nil
		}
		err = candidate.ClearTags(g.nodeID)
	default:
		err = candidate.RemoveTag(g.nodeID, g.action)
	}
	if err != nil {
		return err
	}
	serialized, err := candidate.Serialize(mindmap.SerializeOptions{})
	if err != nil {
		return err
	}
	if len(serialized) > mindmap.DefaultMaxBytes {
		return errMindmapPreviewTooLarge
	}
	checked, _, err := mindmap.Parse(serialized, mindmap.ParseOptions{})
	if err != nil {
		return err
	}
	node, _ = checked.FindNode(g.nodeID)
	values, _ = tagValues(node)
	if err = tagBudget(values); err != nil {
		return err
	}
	releaseEpoch, epoch := v.releaseEpoch, v.tagEpoch
	field := v.fieldIdentity()
	clamped, err := v.commitBufferSelection(string(serialized), true)
	if err != nil {
		return err
	}
	// Changed may release/switch/replace inputs. The source edit succeeded;
	// clear only exactly the captured input, never a replacement or new edit.
	sameInput := v.sameTagInput(g.input)
	if g.action == tagAdd && sameInput {
		v.clearTagDraft()
	}
	if sameInput && v.sameFieldIdentity(field) && v.releaseEpoch == releaseEpoch && v.host.Buffer == g.buffer && v.selectedID == g.nodeID && (v.tagEpoch == epoch || g.action == tagAdd && v.tagEpoch == epoch+1) {
		v.tagInfo = tagPresentation{}
		v.ensureSynced(nil)
		v.refreshTagDraft()
		if clamped {
			v.addWarning("修改标签后源码已重新序列化；原选区已夹取到有效字符边界，撤销可恢复原选区")
		}
	}
	return nil
}
