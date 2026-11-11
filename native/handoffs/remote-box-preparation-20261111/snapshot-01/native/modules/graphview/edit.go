package graphview

import (
	"errors"
	"fmt"
	"strconv"

	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
	"levnote.local/native/formats/mindmap"
)

func (v *view) commitBuffer(value string) error {
	_, err := v.commitBufferSelection(value, false)
	return err
}

// Selection restoration belongs inside the transaction so redo does not move
// a split source caret to EOF again. A whole-document serialization cannot
// promise semantic source mapping; preserve numerical UTF-16 endpoints where
// valid, otherwise clamp to the preceding legal boundary and disclose it.
func (v *view) commitBufferSelection(value string, preserveSelection bool) (bool, error) {
	if v.host.Buffer == nil {
		return false, errors.New("no host buffer")
	}
	if v.host.ReadOnly {
		return false, errors.New("document is read-only")
	}
	if value == v.host.Buffer.String() {
		return false, nil
	}
	selection := v.host.Buffer.Selection()
	if err := v.host.Buffer.BeginGroup(); err != nil {
		return false, err
	}
	if err := v.host.Buffer.ReplaceUTF16(0, v.host.Buffer.LenUTF16(), value); err != nil {
		v.host.Buffer.CancelGroup()
		return false, err
	}
	clamped := false
	if preserveSelection {
		boundary := func(offset int) int {
			offset = max(0, min(offset, v.host.Buffer.LenUTF16()))
			if _, err := v.host.Buffer.SliceUTF16(offset, offset); errors.Is(err, text.ErrSurrogate) {
				offset--
			}
			return offset
		}
		anchor, head := boundary(selection.Anchor), boundary(selection.Head)
		if err := v.host.Buffer.SetSelection(anchor, head); err != nil {
			v.host.Buffer.CancelGroup()
			return false, err
		}
		clamped = anchor != selection.Anchor || head != selection.Head
	}
	if !v.host.Buffer.EndGroup() {
		return false, errors.New("graph edit produced no buffer transaction")
	}
	v.lastSynced = ^uint64(0)
	if v.host.Changed != nil {
		v.host.Changed()
	}
	return clamped, nil
}

func (v *view) finishCommitted(selected string) {
	if selected != "" {
		v.selectedID = selected
	}
	v.lastSynced = ^uint64(0)
	v.ensureSynced(nil)
}

func (v *view) applyMindmapMutation(selected string, mutate func(*mindmap.Document) error) error {
	return v.applyMindmapMutationSelection(selected, false, mutate)
}

func (v *view) applyMindmapMutationSelection(selected string, preserveSelection bool, mutate func(*mindmap.Document) error) error {
	if err := v.Flush(); err != nil {
		return err
	}
	v.ensureSynced(nil)
	if err := v.requireWritable(); err != nil {
		return err
	}
	if v.host.Buffer == nil {
		return errors.New("no host buffer")
	}
	candidate, _, err := mindmap.Parse([]byte(v.host.Buffer.String()), mindmap.ParseOptions{})
	if err != nil {
		v.report(err)
		return err
	}
	if err := mutate(candidate); err != nil {
		v.report(err)
		return err
	}
	serialized, err := candidate.Serialize(mindmap.SerializeOptions{})
	if err != nil {
		v.report(err)
		return err
	}
	if _, _, err := mindmap.Parse(serialized, mindmap.ParseOptions{}); err != nil {
		v.report(err)
		return err
	}
	clamped, err := v.commitBufferSelection(string(serialized), preserveSelection)
	if err != nil {
		v.report(err)
		return err
	}
	v.finishCommitted(selected)
	if clamped {
		v.addWarning("展开节点后源码已重新序列化；原选区已夹取到有效字符边界，撤销可恢复原选区")
	}
	return nil
}

func (v *view) addMindmapChild() {
	if err := v.Flush(); err != nil {
		return
	}
	v.ensureSynced(nil)
	node := v.selectedMindNode()
	if node == nil {
		v.invalidAction("add child")
		return
	}
	id := v.nextMindmapID()
	err := v.applyMindmapMutation(id, func(document *mindmap.Document) error {
		_, err := document.AddNode(node.ID, -1, mindmap.NewNode(id, "新主题"))
		return err
	})
	if err == nil {
		v.focusTopic()
	}
}

func (v *view) addMindmapSibling() {
	if err := v.Flush(); err != nil {
		return
	}
	v.ensureSynced(nil)
	node := v.selectedMindNode()
	if node == nil {
		v.invalidAction("add sibling")
		return
	}
	parent, index, ok := v.mindmapParent(node.ID)
	if !ok {
		v.invalidAction("add sibling to root")
		return
	}
	id := v.nextMindmapID()
	err := v.applyMindmapMutation(id, func(document *mindmap.Document) error {
		_, err := document.AddNode(parent.ID, index+1, mindmap.NewNode(id, "新主题"))
		return err
	})
	if err == nil {
		v.focusTopic()
	}
}

func (v *view) toggleMindmapCollapse() {
	if err := v.Flush(); err != nil {
		return
	}
	v.ensureSynced(nil)
	node := v.selectedMindNode()
	if node == nil {
		v.invalidAction("collapse")
		return
	}
	value := node.Collapsed == nil || !*node.Collapsed
	_ = v.applyMindmapMutation(node.ID, func(document *mindmap.Document) error {
		return document.SetCollapsed(node.ID, value)
	})
}

func (v *view) deleteMindmapNode() {
	if err := v.Flush(); err != nil {
		return
	}
	v.ensureSynced(nil)
	node := v.selectedMindNode()
	if node == nil {
		v.invalidAction("delete node")
		return
	}
	if node.ID == v.mind.RootID() {
		v.report(mindmap.ErrRootImmutable)
		return
	}
	parent, _, ok := v.mindmapParent(node.ID)
	if !ok {
		v.report(mindmap.ErrRootImmutable)
		return
	}
	_ = v.applyMindmapMutation(parent.ID, func(document *mindmap.Document) error {
		_, err := document.DeleteNode(node.ID)
		return err
	})
}

func (v *view) promoteMindmapNode() {
	if err := v.Flush(); err != nil {
		return
	}
	v.ensureSynced(nil)
	node := v.selectedMindNode()
	if node == nil || node.ID == v.mind.RootID() {
		v.invalidAction("promote node")
		return
	}
	parent, _, ok := v.mindmapParent(node.ID)
	if !ok {
		v.invalidAction("promote node")
		return
	}
	grandparent, _, ok := v.mindmapParent(parent.ID)
	if !ok {
		// Parent is root, so promotion makes the node a sibling of its former
		// parent under the root.
		grandparent = v.mind.Root
	}
	_ = v.applyMindmapMutation(node.ID, func(document *mindmap.Document) error {
		return document.MoveNode(node.ID, grandparent.ID, -1)
	})
}

func (v *view) moveMindmapNode(id, parentID string) {
	_ = v.applyMindmapMutation(id, func(document *mindmap.Document) error {
		return document.MoveNode(id, parentID, -1)
	})
}

func (v *view) cycleMindmapLayout() {
	if err := v.Flush(); err != nil {
		return
	}
	v.ensureSynced(nil)
	next := mindmap.LayoutLeft
	if v.mind != nil {
		switch v.mind.Layout {
		case mindmap.LayoutLeft:
			next = mindmap.LayoutRight
		case mindmap.LayoutRight:
			next = mindmap.LayoutBoth
		case mindmap.LayoutBoth:
			next = mindmap.LayoutOrg
		default:
			next = mindmap.LayoutLeft
		}
	}
	_ = v.applyMindmapMutation(v.selectedID, func(document *mindmap.Document) error {
		return document.SetLayout(next)
	})
	v.fitted = false
}

func (v *view) mindmapParent(id string) (*mindmap.Node, int, bool) {
	if v.mind == nil || v.mind.Root == nil {
		return nil, 0, false
	}
	var walk func(*mindmap.Node) (*mindmap.Node, int, bool)
	walk = func(parent *mindmap.Node) (*mindmap.Node, int, bool) {
		for index, child := range parent.Children {
			if child.ID == id {
				return parent, index, true
			}
			if found, childIndex, ok := walk(child); ok {
				return found, childIndex, true
			}
		}
		return nil, 0, false
	}
	return walk(v.mind.Root)
}

func (v *view) nextMindmapID() string {
	var exists func(*mindmap.Node, string) bool
	exists = func(node *mindmap.Node, id string) bool {
		if node.ID == id {
			return true
		}
		for _, child := range node.Children {
			if exists(child, id) {
				return true
			}
		}
		return false
	}
	for index := 1; ; index++ {
		candidate := "node-" + strconv.Itoa(index)
		if v.mind != nil && v.mind.Root != nil && exists(v.mind.Root, candidate) {
			continue
		}
		return candidate
	}
}

func (v *view) applyDrawioMutation(page int, selected string, mutate func(*drawio.Document) (string, error)) error {
	if err := v.Flush(); err != nil {
		return err
	}
	v.ensureSynced(nil)
	if err := v.requireWritable(); err != nil {
		return err
	}
	if v.host.Buffer == nil {
		return errors.New("no host buffer")
	}
	candidate, _, err := drawio.Parse([]byte(v.host.Buffer.String()), drawio.ParseOptions{})
	if err != nil {
		v.report(err)
		return err
	}
	if page < 0 || page >= len(candidate.Pages) {
		err = fmt.Errorf("%w: %d", drawio.ErrPageNotFound, page)
		v.report(err)
		return err
	}
	selected, err = mutate(candidate)
	if err != nil {
		v.report(err)
		return err
	}
	serialized, err := candidate.Serialize()
	if err != nil {
		v.report(err)
		return err
	}
	if _, _, err := drawio.Parse(serialized, drawio.ParseOptions{}); err != nil {
		v.report(err)
		return err
	}
	if err := v.commitBuffer(string(serialized)); err != nil {
		v.report(err)
		return err
	}
	v.finishCommitted(selected)
	return nil
}

func (v *view) moveDiagramNode(id string, x, y float64) {
	if err := v.Flush(); err != nil {
		return
	}
	v.ensureSynced(nil)
	node := v.selectedDrawioNode()
	if node == nil || node.ID != id {
		v.invalidAction("move node")
		return
	}
	_ = v.applyDrawioMutation(v.pageIndex, id, func(document *drawio.Document) (string, error) {
		return id, document.UpdateGeometry(v.pageIndex, id, x, y, node.Width, node.Height)
	})
}

func (v *view) addDiagramShape(shape string) {
	if err := v.Flush(); err != nil {
		return
	}
	v.ensureSynced(nil)
	page := v.currentPage()
	if page == nil {
		v.invalidAction("add shape")
		return
	}
	if v.canvasRect.W <= 1 || v.canvasRect.H <= 1 {
		v.invalidAction("add shape before the canvas is visible")
		return
	}
	if v.kind == kindDrawio {
		worldX, worldY := v.screenToWorld(v.canvasRect.W/2, v.canvasRect.H/2)
		_ = v.applyDrawioMutation(v.pageIndex, "", func(document *drawio.Document) (string, error) {
			return document.AddShape(v.pageIndex, drawio.ShapeSpec{
				Text:   "新" + shapeLabel(shape),
				Shape:  shape,
				Parent: "1",
				X:      float64(worldX - 70),
				Y:      float64(worldY - 35),
				Width:  140,
				Height: 70,
			})
		})
	}
}

func (v *view) addDiagramConnector(source, target string) {
	_ = v.applyDrawioMutation(v.pageIndex, source, func(document *drawio.Document) (string, error) {
		id, err := document.AddConnector(v.pageIndex, drawio.ConnectorSpec{
			Text:     "",
			Parent:   "1",
			SourceID: source,
			TargetID: target,
		})
		return source, func() error {
			if err != nil {
				return err
			}
			_ = id
			return nil
		}()
	})
}

func shapeLabel(shape string) string {
	switch shape {
	case drawio.ShapeEllipse:
		return "椭圆"
	case drawio.ShapeDiamond:
		return "菱形"
	case drawio.ShapeRounded:
		return "圆角框"
	default:
		return "矩形"
	}
}
