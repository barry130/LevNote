package graphview

import (
	"fmt"
	"strings"

	"levnote.local/native/core/text"
	"levnote.local/native/formats/mindmap"
	"levnote.local/native/ui/editor"
)

type mindmapOutlineSource struct {
	buffer       *text.Buffer
	cachedBuffer *text.Buffer
	version      uint64
	loaded       bool
	items        []editor.OutlineItem
	truncated    bool
	err          error
}

func newMindmapOutlineSource(host editor.Host) editor.OutlineSource {
	return &mindmapOutlineSource{buffer: host.Buffer}
}

func (s *mindmapOutlineSource) Release() {
	s.cachedBuffer, s.loaded, s.items, s.err, s.truncated = nil, false, nil, nil, false
}

func (s *mindmapOutlineSource) refresh() error {
	if s.buffer == nil {
		return fmt.Errorf("没有可用的文档缓冲区：%w", editor.ErrOutlineUnsupported)
	}
	if s.loaded && s.cachedBuffer == s.buffer && s.version == s.buffer.Version() {
		return s.err
	}
	s.Release()
	s.cachedBuffer, s.version, s.loaded = s.buffer, s.buffer.Version(), true
	if s.buffer.LenBytes() > mindmap.DefaultMaxBytes {
		s.err = fmt.Errorf("思维导图超过大纲处理上限，请在源码中查找：%w", editor.ErrOutlineUnsupported)
		return s.err
	}
	doc, _, err := mindmap.Parse([]byte(s.buffer.String()), mindmap.ParseOptions{})
	if err != nil {
		s.err = fmt.Errorf("思维导图大纲无法解析：%w", err)
		return s.err
	}
	// Walk the parser's tree, including folded descendants. No layout, view,
	// metadata or full label survives this bounded descriptor collection.
	var walk func(*mindmap.Node, int) bool
	walk = func(node *mindmap.Node, level int) bool {
		if node == nil {
			return true
		}
		if len(s.items) == editor.MaxOutlineItems {
			s.truncated = true
			return false
		}
		if len(node.ID) > editor.MaxOutlineIDBytes {
			s.truncated = true // never clip an identity into another node's ID
		} else {
			label, clipped := editor.OutlineLabel(node.Topic)
			s.truncated = s.truncated || clipped
			s.items = append(s.items, editor.OutlineItem{ID: strings.Clone(node.ID), Text: label, Level: level, Offset: -1})
		}
		for _, child := range node.Children {
			if !walk(child, level+1) {
				return false
			}
		}
		return true
	}
	walk(doc.Root, 1)
	return nil
}

func (s *mindmapOutlineSource) OutlineSnapshot() (editor.OutlineSnapshot, error) {
	if err := s.refresh(); err != nil {
		return editor.OutlineSnapshot{Version: s.version}, err
	}
	return editor.OutlineSnapshot{Version: s.version, Items: append([]editor.OutlineItem(nil), s.items...), Truncated: s.truncated}, nil
}

func (s *mindmapOutlineSource) ResolveOutlineItem(id string) (editor.OutlineItem, error) {
	if len(id) > editor.MaxOutlineIDBytes || id == "" {
		return editor.OutlineItem{}, editor.ErrOutlineStale
	}
	if err := s.refresh(); err != nil {
		return editor.OutlineItem{}, err
	}
	for _, item := range s.items {
		if item.ID == id {
			return item, nil
		}
	}
	return editor.OutlineItem{}, editor.ErrOutlineStale
}

func (*mindmapOutlineSource) ResolveOutlineAnchor(string) (editor.OutlineItem, error) {
	return editor.OutlineItem{}, fmt.Errorf("思维导图不支持 Markdown 标题锚点：%w", editor.ErrOutlineUnsupported)
}

func (v *view) outlineSource() *mindmapOutlineSource {
	if v.outline == nil || v.outline.buffer != v.host.Buffer {
		v.outline = &mindmapOutlineSource{buffer: v.host.Buffer}
	}
	return v.outline
}

// Only ancestors hide a target. Selecting a folded root itself does not
// expand it, but a folded root must be expanded when locating a descendant.
func collapsedAncestors(root *mindmap.Node, id string) ([]string, bool) {
	var path, found []string
	var walk func(*mindmap.Node) bool
	walk = func(node *mindmap.Node) bool {
		if node == nil {
			return false
		}
		if node.ID == id {
			found = append([]string(nil), path...)
			return true
		}
		length := len(path)
		if node.Collapsed != nil && *node.Collapsed {
			path = append(path, node.ID)
		}
		for _, child := range node.Children {
			if walk(child) {
				return true
			}
		}
		path = path[:length]
		return false
	}
	ok := walk(root)
	return found, ok
}

func (v *view) NavigateOutlineItem(id string) error {
	if v.kind != kindMindmap {
		return editor.ErrOutlineUnsupported
	}
	if err := v.Flush(); err != nil {
		return err
	}
	item, err := v.outlineSource().ResolveOutlineItem(id)
	if err != nil {
		return err
	}
	// Inspect before ensureSynced, which may establish an initial selection.
	// Readonly hidden-node failure must leave selection and canvas untouched.
	doc := v.mind
	if doc == nil || v.syncedBuffer != v.host.Buffer || v.lastSynced != v.host.Buffer.Version() {
		doc, _, err = mindmap.Parse([]byte(v.host.Buffer.String()), mindmap.ParseOptions{})
		if err != nil {
			return err
		}
	}
	ancestors, found := collapsedAncestors(doc.Root, item.ID)
	if !found {
		return editor.ErrOutlineStale
	}
	if len(ancestors) > 0 && v.host.ReadOnly {
		return editor.ErrOutlineReadOnlyHidden
	}
	if len(ancestors) > 0 && v.host.Buffer.InGroup() {
		return text.ErrGroup
	}
	if len(ancestors) > 0 {
		err = v.applyMindmapMutationSelection(item.ID, true, func(candidate *mindmap.Document) error {
			for _, ancestor := range ancestors {
				if err := candidate.SetCollapsed(ancestor, false); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	} else {
		v.ensureSynced(nil)
	}
	v.invalidateTagActions()
	v.selectedID = item.ID
	v.refreshLabelDraft()
	v.refreshTagDraft()
	v.pendingOutlineCenter = item.ID
	v.fitted = true // a deferred first paint must keep the requested zoom too
	v.centerOutlineTarget()
	return nil
}

func (v *view) centerOutlineTarget() {
	if v.pendingOutlineCenter == "" || v.canvasRect.W <= 0 || v.canvasRect.H <= 0 {
		return
	}
	for _, node := range v.layout.Nodes {
		if node.ID == v.pendingOutlineCenter {
			v.panX = v.canvasRect.W/2 - float32(node.X+node.Width/2)*v.zoom
			v.panY = v.canvasRect.H/2 - float32(node.Y+node.Height/2)*v.zoom
			v.fitted = true
			break
		}
	}
	v.pendingOutlineCenter = ""
}

func (v *view) CurrentOutlineItem() (string, bool) {
	if v.kind != kindMindmap || v.selectedID == "" {
		return "", false
	}
	item, err := v.outlineSource().ResolveOutlineItem(v.selectedID)
	return item.ID, err == nil
}
