package mindmap

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Tags returns the exact ordered optional tag list as an owned deep copy.
// Absent is nil/false; present empty is non-nil/true. Reading is not an edit.
func (d *Document) Tags(id string) ([]Tag, bool, error) {
	node, ok := d.FindNode(id)
	if !ok {
		return nil, false, fmt.Errorf("%w: tags for %q", ErrNodeNotFound, id)
	}
	if node.Tags == nil {
		return nil, false, nil
	}
	return cloneTags(*node.Tags), true, nil
}

// AddTag appends one literal nonempty UTF-8 string, without trimming, splitting
// or deduplicating. Existing object/string tags retain their original kind and
// metadata. Input/UI budgets belong to the caller, not the file schema.
func (d *Document) AddTag(id, value string) error {
	node, ok := d.FindNode(id)
	if !ok {
		return fmt.Errorf("%w: tags for %q", ErrNodeNotFound, id)
	}
	if value == "" || !utf8.ValidString(value) {
		return fmt.Errorf("%w: new tag must be a nonempty UTF-8 string", ErrInvalidDocument)
	}
	var tags []Tag
	if node.Tags != nil {
		tags = *node.Tags
	}
	candidate := make([]Tag, len(tags)+1)
	for i, tag := range tags {
		candidate[i] = cloneTag(tag)
	}
	candidate[len(tags)] = Tag{Text: strings.Clone(value)}
	return d.replaceTags(node, candidate)
}

// RemoveTag removes exactly one zero-based ordinal, never every equal text.
// Removing the final entry leaves a present empty list.
func (d *Document) RemoveTag(id string, ordinal int) error {
	node, ok := d.FindNode(id)
	if !ok {
		return fmt.Errorf("%w: tags for %q", ErrNodeNotFound, id)
	}
	if node.Tags == nil || ordinal < 0 || ordinal >= len(*node.Tags) {
		return fmt.Errorf("%w: tag ordinal is out of range", ErrInvalidDocument)
	}
	candidate := make([]Tag, 0, len(*node.Tags)-1)
	for i, tag := range *node.Tags {
		if i != ordinal {
			candidate = append(candidate, cloneTag(tag))
		}
	}
	return d.replaceTags(node, candidate)
}

// ClearTags preserves absence/already-empty as a no-op. A nonempty list is
// replaced with present [], including when it held legacy empty strings.
func (d *Document) ClearTags(id string) error {
	node, ok := d.FindNode(id)
	if !ok {
		return fmt.Errorf("%w: tags for %q", ErrNodeNotFound, id)
	}
	if node.Tags == nil || len(*node.Tags) == 0 {
		return nil
	}
	return d.replaceTags(node, make([]Tag, 0))
}

func (d *Document) replaceTags(node *Node, candidate []Tag) error {
	previous := node.Tags
	node.Tags = &candidate
	if err := d.validate(); err != nil {
		node.Tags = previous
		return err
	}
	d.dirty = true
	return nil
}

func cloneTags(tags []Tag) []Tag {
	copy := make([]Tag, len(tags))
	for i, tag := range tags {
		copy[i] = cloneTag(tag)
	}
	return copy
}

func cloneTag(tag Tag) Tag {
	tag.Text, tag.ClassName = strings.Clone(tag.Text), strings.Clone(tag.ClassName)
	tag.Raw = cloneRaw(tag.Raw)
	tag.Style, tag.Extra = cloneTagObject(tag.Style), cloneTagObject(tag.Extra)
	return tag
}

func cloneTagObject(object Object) Object {
	copy := Object{Members: make([]Member, len(object.Members))}
	for i, member := range object.Members {
		copy.Members[i] = Member{Key: strings.Clone(member.Key), Raw: cloneRaw(member.Raw)}
	}
	return copy
}
