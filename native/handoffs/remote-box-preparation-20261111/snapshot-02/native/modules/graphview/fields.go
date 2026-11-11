package graphview

import (
	"errors"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/formats/mindmap"
)

// The private discriminator covers both adapters. A visible label or selected
// node alone cannot identify the authoritative field of an independent draft.
type nodeField uint8

const (
	fieldTopic nodeField = iota
	fieldNote
	fieldHyperLink
	fieldDrawioLabel
)

func (field nodeField) inputLabel() string {
	switch field {
	case fieldNote:
		return "节点备注（纯文本）"
	case fieldHyperLink:
		return "节点链接（本地）"
	default:
		return "节点文本"
	}
}

func mindmapField(node *mindmap.Node, field nodeField) (string, bool) {
	if node == nil {
		return "", false
	}
	switch field {
	case fieldTopic:
		return node.Topic, true
	case fieldNote:
		if node.Note != nil {
			return *node.Note, true
		}
	case fieldHyperLink:
		if node.HyperLink != nil {
			return *node.HyperLink, true
		}
	}
	return "", false
}

func applyMindmapField(doc *mindmap.Document, id string, field nodeField, value string) error {
	switch field {
	case fieldTopic:
		return doc.RenameNode(id, value)
	case fieldNote:
		return doc.SetNote(id, value)
	case fieldHyperLink:
		if value == "" {
			return doc.ClearField(id, mindmap.FieldHyperLink)
		}
		return doc.SetHyperLink(id, value)
	default:
		return errors.New("未知节点字段，未写回草稿")
	}
}

func (v *view) selectField(field nodeField) bool {
	if v.kind != kindMindmap || field > fieldHyperLink {
		return false
	}
	if err := v.Flush(); err != nil {
		if !v.pendingTagDraft() {
			v.focusLabel()
		}
		return false
	}
	v.ensureSynced(nil)
	if v.selectedMindNode() == nil || v.syncFailed {
		return false
	}
	v.linkGesture = fieldLinkTarget{}
	v.invalidateTagActions()
	v.field = field
	v.refreshDraft()
	return true
}

func (v *view) drawFieldSelector(c *ui.Context) {
	ui.Row(c).FillWidth().Shrink(0).Gap(4).Children(func() {
		for _, option := range []struct {
			field       nodeField
			text, label string
		}{
			{fieldTopic, "文本", "字段：主题文本"},
			{fieldNote, "备注", "字段：纯文本备注"},
			{fieldHyperLink, "链接", "字段：本地链接"},
		} {
			button := ui.Button(c.Key(int(option.field)), option.text).Label(option.label).
				Grow(1).Height(28).Padding(4, 6).FontSize(12)
			if v.field == option.field {
				button.Bind(&v.fieldHandle).Border(1, c.Theme().TextMuted).Background(c.Theme().SurfaceHover).Tooltip("当前字段")
			}
			if button.Clicked() {
				v.selectField(option.field)
			}
		}
	})
}
