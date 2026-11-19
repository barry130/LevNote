package mindmap

import (
	"fmt"
	"unicode/utf8"
)

// FieldKind identifies an optional node field that the native inspector can
// read and edit without treating it as topic text.
type FieldKind string

const (
	FieldNote      FieldKind = "note"
	FieldHyperLink FieldKind = "hyperLink"
)

func validFieldKind(field FieldKind) bool {
	switch field {
	case FieldNote, FieldHyperLink:
		return true
	default:
		return false
	}
}

func validFieldValue(field FieldKind, value string) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%w: %s is not valid UTF-8", ErrInvalidDocument, field)
	}
	if field == FieldHyperLink && value == "" {
		return fmt.Errorf("%w: hyperLink is empty", ErrInvalidDocument)
	}
	return nil
}

func (d *Document) findEditableNode(id string, field FieldKind) (*Node, error) {
	node, ok := d.FindNode(id)
	if !ok {
		return nil, fmt.Errorf("%w: %s field for %q", ErrNodeNotFound, field, id)
	}
	return node, nil
}

// Field returns the literal value and presence of an optional node field.
// Reading never changes the document or its serialization.
func (d *Document) Field(id string, field FieldKind) (string, bool, error) {
	if !validFieldKind(field) {
		return "", false, fmt.Errorf("%w: unknown field kind %q", ErrInvalidDocument, field)
	}
	node, err := d.findEditableNode(id, field)
	if err != nil {
		return "", false, err
	}
	switch field {
	case FieldNote:
		if node.Note == nil {
			return "", false, nil
		}
		return *node.Note, true, nil
	case FieldHyperLink:
		if node.HyperLink == nil {
			return "", false, nil
		}
		return *node.HyperLink, true, nil
	default:
		return "", false, fmt.Errorf("%w: unknown field kind %q", ErrInvalidDocument, field)
	}
}

// SetNote stores an exact, present note. An empty string is meaningful and is
// distinct from an absent note.
func (d *Document) SetNote(id string, value string) error {
	node, err := d.findEditableNode(id, FieldNote)
	if err != nil {
		return err
	}
	if err := validFieldValue(FieldNote, value); err != nil {
		return err
	}
	if node.Note != nil && *node.Note == value {
		return nil
	}

	old := node.Note
	node.Note = &value
	if err := d.validate(); err != nil {
		node.Note = old
		return err
	}
	d.dirty = true
	return nil
}

// SetHyperLink stores an exact, present, nonempty local-link value.
func (d *Document) SetHyperLink(id string, value string) error {
	node, err := d.findEditableNode(id, FieldHyperLink)
	if err != nil {
		return err
	}
	if err := validFieldValue(FieldHyperLink, value); err != nil {
		return err
	}
	if node.HyperLink != nil && *node.HyperLink == value {
		return nil
	}

	old := node.HyperLink
	node.HyperLink = &value
	if err := d.validate(); err != nil {
		node.HyperLink = old
		return err
	}
	d.dirty = true
	return nil
}

// ClearField removes an optional field. Clearing an absent field is a no-op.
func (d *Document) ClearField(id string, field FieldKind) error {
	if !validFieldKind(field) {
		return fmt.Errorf("%w: unknown field kind %q", ErrInvalidDocument, field)
	}
	node, err := d.findEditableNode(id, field)
	if err != nil {
		return err
	}

	var old **string
	switch field {
	case FieldNote:
		old = &node.Note
	case FieldHyperLink:
		old = &node.HyperLink
	}
	if *old == nil {
		return nil
	}

	previous := *old
	*old = nil
	if err := d.validate(); err != nil {
		*old = previous
		return err
	}
	d.dirty = true
	return nil
}

// Icons returns an exact ordered copy of a node's optional glyph list. An
// absent field returns nil and false; present empty returns a non-nil empty
// copy and true.
func (d *Document) Icons(id string) ([]string, bool, error) {
	node, ok := d.FindNode(id)
	if !ok {
		return nil, false, fmt.Errorf("%w: icons for %q", ErrNodeNotFound, id)
	}
	if node.Icons == nil {
		return nil, false, nil
	}
	return append(make([]string, 0, len(*node.Icons)), *node.Icons...), true, nil
}

// SetIcons stores an exact, present icon list. Nil means present empty, which
// is distinct from an absent field. Arbitrary glyphs, order, and duplicates
// are preserved; the format layer imposes no marker vocabulary or size limit.
func (d *Document) SetIcons(id string, values []string) error {
	node, ok := d.FindNode(id)
	if !ok {
		return fmt.Errorf("%w: icons for %q", ErrNodeNotFound, id)
	}
	for _, value := range values {
		if !utf8.ValidString(value) {
			return fmt.Errorf("%w: icons contain invalid UTF-8", ErrInvalidDocument)
		}
	}
	if node.Icons != nil && iconListsEqual(*node.Icons, values) {
		return nil
	}

	candidate := make([]string, len(values))
	copy(candidate, values)
	old := node.Icons
	node.Icons = &candidate
	if err := d.validate(); err != nil {
		node.Icons = old
		return err
	}
	d.dirty = true
	return nil
}

func iconListsEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
