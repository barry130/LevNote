package mindmap

import (
	"fmt"
)

func (d *Document) FindNode(id string) (*Node, bool) {
	if d == nil || d.Root == nil {
		return nil, false
	}
	var find func(*Node) *Node
	find = func(node *Node) *Node {
		if node.ID == id {
			return node
		}
		for _, child := range node.Children {
			if found := find(child); found != nil {
				return found
			}
		}
		return nil
	}
	node := find(d.Root)
	return node, node != nil
}

type nodeLocation struct {
	parent *Node
	index  int
	node   *Node
}

func (d *Document) locate(id string) (nodeLocation, bool) {
	if d == nil || d.Root == nil {
		return nodeLocation{}, false
	}
	if d.Root.ID == id {
		return nodeLocation{node: d.Root}, true
	}
	var find func(*Node) (nodeLocation, bool)
	find = func(parent *Node) (nodeLocation, bool) {
		for index, child := range parent.Children {
			if child.ID == id {
				return nodeLocation{parent: parent, index: index, node: child}, true
			}
			if location, ok := find(child); ok {
				return location, true
			}
		}
		return nodeLocation{}, false
	}
	return find(d.Root)
}

// AddNode inserts a cloned node under parentID. index == -1 appends; otherwise
// index must be between zero and the current child count, inclusive.
func (d *Document) AddNode(parentID string, index int, node *Node) (*Node, error) {
	if d == nil || d.Root == nil {
		return nil, ErrInvalidDocument
	}
	parent, ok := d.FindNode(parentID)
	if !ok {
		return nil, fmt.Errorf("%w: parent %q", ErrNodeNotFound, parentID)
	}
	if index < -1 || index > len(parent.Children) {
		return nil, fmt.Errorf("%w: child index %d", ErrInvalidDocument, index)
	}
	if node == nil {
		return nil, fmt.Errorf("%w: nil node", ErrInvalidDocument)
	}
	clone := node.Clone()
	temp := &Document{
		Root:    clone,
		Layout:  LayoutBoth,
		Version: Version,
	}
	if err := temp.validate(); err != nil {
		return nil, err
	}
	for _, existing := range d.eachNode() {
		if _, found := findInSubtree(clone, existing.ID); found {
			return nil, fmt.Errorf("%w: %q", ErrDuplicateID, existing.ID)
		}
	}
	if index == -1 {
		index = len(parent.Children)
	}
	parent.Children = append(parent.Children, nil)
	copy(parent.Children[index+1:], parent.Children[index:])
	parent.Children[index] = clone
	if err := d.validate(); err != nil {
		parent.Children = append(parent.Children[:index], parent.Children[index+1:]...)
		return nil, err
	}
	d.dirty = true
	return clone, nil
}

// DeleteNode removes a non-root node and its complete subtree.
func (d *Document) DeleteNode(id string) (*Node, error) {
	location, ok := d.locate(id)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrNodeNotFound, id)
	}
	if location.parent == nil {
		return nil, fmt.Errorf("%w: %q", ErrRootImmutable, id)
	}
	parent := location.parent
	parent.Children = append(parent.Children[:location.index], parent.Children[location.index+1:]...)
	d.dirty = true
	return location.node, nil
}

// RenameNode changes a node's topic. It is valid for the root.
func (d *Document) RenameNode(id, topic string) error {
	node, ok := d.FindNode(id)
	if !ok {
		return fmt.Errorf("%w: %q", ErrNodeNotFound, id)
	}
	if node.Topic == topic {
		return nil
	}
	node.Topic = topic
	d.dirty = true
	return nil
}

// MoveNode moves a non-root node under newParentID. index is an insertion
// position in the destination list before the move; -1 appends.
func (d *Document) MoveNode(id, newParentID string, index int) error {
	location, ok := d.locate(id)
	if !ok {
		return fmt.Errorf("%w: %q", ErrNodeNotFound, id)
	}
	if location.parent == nil {
		return fmt.Errorf("%w: %q", ErrRootImmutable, id)
	}
	newParent, ok := d.FindNode(newParentID)
	if !ok {
		return fmt.Errorf("%w: parent %q", ErrNodeNotFound, newParentID)
	}
	if newParent == location.node {
		return ErrCycle
	}
	if _, found := findInSubtree(location.node, newParent.ID); found {
		return ErrCycle
	}
	oldParent := location.parent
	oldIndex := location.index
	if index < -1 || index > len(newParent.Children) {
		return fmt.Errorf("%w: child index %d", ErrInvalidDocument, index)
	}

	oldParent.Children = append(oldParent.Children[:oldIndex], oldParent.Children[oldIndex+1:]...)
	if newParent == oldParent && index > oldIndex {
		index--
	}
	if index == -1 {
		index = len(newParent.Children)
	}
	if index < 0 || index > len(newParent.Children) {
		index = len(newParent.Children)
	}
	newParent.Children = append(newParent.Children, nil)
	copy(newParent.Children[index+1:], newParent.Children[index:])
	newParent.Children[index] = location.node
	d.dirty = true
	return nil
}

func (d *Document) SetCollapsed(id string, collapsed bool) error {
	node, ok := d.FindNode(id)
	if !ok {
		return fmt.Errorf("%w: %q", ErrNodeNotFound, id)
	}
	if node.Collapsed != nil && *node.Collapsed == collapsed {
		return nil
	}
	value := collapsed
	node.Collapsed = &value
	d.dirty = true
	return nil
}

// SetLayout accepts the four formats this native editor can lay out. Unknown
// layouts that were parsed from a newer file remain readable and preserved.
func (d *Document) SetLayout(layout string) error {
	if !KnownLayout(layout) {
		return fmt.Errorf("%w: %q", ErrUnknownLayoutKind, layout)
	}
	if d.Layout == layout {
		return nil
	}
	d.Layout = layout
	d.dirty = true
	return nil
}

func (d *Document) eachNode() []*Node {
	if d == nil || d.Root == nil {
		return nil
	}
	nodes := make([]*Node, 0)
	var walk func(*Node)
	walk = func(node *Node) {
		nodes = append(nodes, node)
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(d.Root)
	return nodes
}

func findInSubtree(root *Node, id string) (*Node, bool) {
	if root == nil {
		return nil, false
	}
	if root.ID == id {
		return root, true
	}
	for _, child := range root.Children {
		if found, ok := findInSubtree(child, id); ok {
			return found, true
		}
	}
	return nil, false
}
