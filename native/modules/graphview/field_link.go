package graphview

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/mindmap"
)

const maxFieldLinkBytes = 4096

var errFieldLinkStale = errors.New("节点链接或草稿已改变，请重新操作；未打开链接")
var errFieldLinkLong = errors.New("链接超过4096字节，请在源码中查看或修改目标；未打开链接")
var errFieldLinkMissing = errors.New("链接已移除，未打开链接")
var errFieldLinkHost = errors.New("当前宿主不能打开本地引用；未打开链接")

// Captured by the actual native press/key, not from the next frame's current
// selectedID. Ordinary destinations are complete and bounded. A pending
// write/open retains the exact independent draft plus its input/commit tokens,
// not another full copy of an arbitrarily long original field.
type fieldLinkTarget struct {
	buffer       *text.Buffer
	nodeID       string
	present      bool
	href         string
	draft        *fieldDraft
	inputVersion uint64
	commits      uint64
	released     bool
}

func fieldLinkText(value string, limit int) string {
	if len(value) <= limit {
		return strings.Clone(value)
	}
	end := limit
	for !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end] + "…"
}

func fieldInputLinkTip(buffer *text.Buffer) string {
	end := min(buffer.LenUTF16(), 512)
	value, err := buffer.SliceUTF16(0, end)
	if errors.Is(err, text.ErrSurrogate) {
		value, _ = buffer.SliceUTF16(0, end-1)
	}
	value = fieldLinkText(value, 512)
	if end < buffer.LenUTF16() && !strings.HasSuffix(value, "…") {
		value += "…"
	}
	return value
}

func (v *view) drawFieldLink(c *ui.Context, node *mindmap.Node) bool {
	href, present := mindmapField(node, fieldHyperLink)
	target := fieldLinkTarget{buffer: v.host.Buffer, nodeID: node.ID, present: present}
	label, tip, reason := "打开链接", "", ""
	if d := v.draft; d != nil && d.nodeID == node.ID && d.field == fieldHyperLink && v.pendingDraft() {
		label = "写回并打开链接"
		target.draft, target.inputVersion, target.commits = d, d.input.Buffer.Version(), d.commits
		if d.input.Buffer.LenBytes() > maxFieldLinkBytes {
			reason = errFieldLinkLong.Error()
		} else {
			tip = fieldInputLinkTip(d.input.Buffer)
		}
	} else if len(href) > maxFieldLinkBytes {
		reason = errFieldLinkLong.Error()
	} else if !present {
		if v.field != fieldHyperLink {
			v.linkEnabled = false
			return false
		}
		reason = "尚无链接；选择链接字段可添加本地引用"
	} else {
		target.href = strings.Clone(href)
		tip = fieldLinkText(href, 512)
	}
	if v.host.OpenReference == nil {
		reason = errFieldLinkHost.Error()
	}
	if reason != "" {
		tip = reason
		v.linkGesture = fieldLinkTarget{}
	}
	v.linkEnabled = reason == ""
	button := ui.Button(c.Key("mindmap-open-link"), label).Label(label).FillWidth().Bind(&v.linkHandle).
		Tooltip(tip).Disabled(reason != "").HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind == ui.InputPointerDown && ev.Button == 0 ||
			ev.Kind == ui.InputKeyDown && ev.Mods == 0 && (ev.Key == ui.KeyEnter || ev.Key == ui.KeySpace) {
			v.linkGesture = target
			v.linkGesture.released = ev.Kind == ui.InputKeyDown
		}
		if ev.Kind == ui.InputPointerUp {
			if ev.Canceled {
				v.linkGesture = fieldLinkTarget{}
			} else {
				v.linkGesture.released = true
			}
		}
		return false
	})
	if button.Clicked() {
		captured := v.linkGesture
		v.linkGesture = fieldLinkTarget{}
		if captured.buffer == nil || captured.nodeID != target.nodeID || captured.buffer != target.buffer {
			v.report(errFieldLinkStale)
		} else {
			v.report(v.activateFieldLink(captured))
		}
		return true
	}
	if reason != "" && (present || target.draft != nil || v.field == fieldHyperLink) {
		ui.Text(c, reason).FontSize(11).TextColor(c.Theme().Warning)
	}
	return false
}

func (v *view) activateFieldLink(target fieldLinkTarget) error {
	// A pending action has to belong to the original, unchanged input, even
	// if a redraw reused the fixed-key button for a different field/node.
	if target.draft != nil && (v.draft != target.draft || v.draft.field != fieldHyperLink ||
		v.draft.input.Buffer.Version() != target.inputVersion || v.draft.commits != target.commits) {
		return errFieldLinkStale
	}
	if err := v.Flush(); err != nil {
		return err
	}
	if v.host.Buffer == nil || v.host.Buffer != target.buffer || v.selectedID != target.nodeID {
		return errFieldLinkStale
	}
	if v.host.Buffer.InGroup() {
		return text.ErrGroup
	}
	if err := v.previewSizeError(); err != nil {
		return err
	}
	v.ensureSynced(nil)
	if v.syncError != nil {
		return v.syncError
	}
	if v.mind == nil {
		return errFieldLinkStale
	}
	node, ok := v.mind.FindNode(target.nodeID)
	if !ok {
		return errFieldLinkStale
	}
	href, present := mindmapField(node, fieldHyperLink)
	if d := target.draft; d != nil {
		if v.draft != d || d.buffer != target.buffer || d.nodeID != target.nodeID ||
			d.field != fieldHyperLink || d.input.Buffer.Version() != target.inputVersion ||
			(d.commits != target.commits && d.commits != target.commits+1) || v.pendingDraft() ||
			href != d.sourceBase || present != d.sourcePresent {
			return errFieldLinkStale
		}
	} else if href != target.href || present != target.present {
		return errFieldLinkStale
	}
	if !present || href == "" {
		return errFieldLinkMissing
	}
	if len(href) > maxFieldLinkBytes {
		return errFieldLinkLong
	}
	if v.host.OpenReference == nil {
		return errFieldLinkHost
	}
	v.host.OpenReference(href)
	return nil
}
