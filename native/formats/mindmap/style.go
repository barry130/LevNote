package mindmap

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

// StyleProperty names the three colors shared by native drawing and SVG.
// Other style members remain opaque and are never rewritten by these edits.
type StyleProperty string

const (
	StyleTextColor   StyleProperty = "color"
	StyleFillColor   StyleProperty = "background-color"
	StyleBorderColor StyleProperty = "border"
)

// StyleTextFallback supplies readable ink only for an explicit opaque fill with
// no text-color member. Native drawing and SVG share this decision. Transparent
// fills depend on their containing surface; existing explicit ink is never
// overridden, even if it is unsupported. This function never changes content.
func StyleTextFallback(node *Node) (string, bool) {
	if node == nil || node.Style == nil {
		return "", false
	}
	if _, explicit := node.Style.Get(string(StyleTextColor)); explicit {
		return "", false
	}
	fill, ok := presentationObjectString(*node.Style, string(StyleFillColor))
	if !ok {
		return "", false
	}
	rgba, ok := literalStyleRGBA(fill)
	if !ok || rgba&255 != 255 {
		return "", false
	}
	luminance := func(rgba uint32) float64 {
		channel := func(value uint32) float64 {
			c := float64(value&255) / 255
			if c <= 0.04045 {
				return c / 12.92
			}
			return math.Pow((c+0.055)/1.055, 2.4)
		}
		return 0.2126*channel(rgba>>24) + 0.7152*channel(rgba>>16) + 0.0722*channel(rgba>>8)
	}
	contrast := func(a, b float64) float64 {
		if a < b {
			a, b = b, a
		}
		return (a + 0.05) / (b + 0.05)
	}
	background := luminance(rgba)
	if contrast(background, luminance(0x17211eff)) >= contrast(background, luminance(0xf8fafcff)) {
		return "#17211e", true
	}
	return "#f8fafc", true
}

func literalStyleRGBA(value string) (uint32, bool) {
	literal, ok := LiteralColor(value)
	if !ok {
		return 0, false
	}
	digits := literal[1:]
	if len(digits) == 3 || len(digits) == 4 {
		var expanded [8]byte
		for i := range digits {
			expanded[i*2], expanded[i*2+1] = digits[i], digits[i]
		}
		digits = string(expanded[:len(digits)*2])
	}
	number, err := strconv.ParseUint(digits, 16, 32)
	if err != nil {
		return 0, false
	}
	if len(digits) == 6 {
		number = number<<8 | 255
	}
	return uint32(number), true
}

func validStyleProperty(property StyleProperty) bool {
	return property == StyleTextColor || property == StyleFillColor || property == StyleBorderColor
}

// SetStyleProperty changes one color. Setting an equivalent literal leaves the
// original JSON spelling, optional presence and document history untouched.
func (d *Document) SetStyleProperty(id string, property StyleProperty, value string) error {
	if !validStyleProperty(property) {
		return fmt.Errorf("%w: unknown style property %q", ErrInvalidDocument, property)
	}
	color, ok := literalStyleRGBA(value)
	if !ok {
		return fmt.Errorf("%w: unsupported literal color", ErrInvalidDocument)
	}
	node, ok := d.FindNode(id)
	if !ok {
		return fmt.Errorf("%w: style for %q", ErrNodeNotFound, id)
	}
	if node.Style != nil {
		if raw, exists := node.Style.Get(string(property)); exists {
			var previous string
			if json.Unmarshal(raw, &previous) == nil {
				if normalized, valid := literalStyleRGBA(previous); valid && normalized == color {
					return nil
				}
			}
		}
	}
	candidate := Object{}
	if node.Style != nil {
		candidate = node.Style.Clone()
	}
	raw, _ := json.Marshal(value)
	if err := candidate.Set(string(property), raw); err != nil {
		return err
	}
	return d.replaceNodeStyle(node, &candidate)
}

// ClearStyleProperty removes only the named member. Removing the final member
// removes the style object; an already absent key preserves any empty object.
func (d *Document) ClearStyleProperty(id string, property StyleProperty) error {
	if !validStyleProperty(property) {
		return fmt.Errorf("%w: unknown style property %q", ErrInvalidDocument, property)
	}
	node, ok := d.FindNode(id)
	if !ok {
		return fmt.Errorf("%w: style for %q", ErrNodeNotFound, id)
	}
	if node.Style == nil {
		return nil
	}
	candidate := node.Style.Clone()
	if !candidate.Delete(string(property)) {
		return nil
	}
	if candidate.Len() == 0 {
		return d.replaceNodeStyle(node, nil)
	}
	return d.replaceNodeStyle(node, &candidate)
}

func (d *Document) replaceNodeStyle(node *Node, candidate *Object) error {
	old := node.Style
	node.Style = candidate
	if err := d.validate(); err != nil {
		node.Style = old
		return err
	}
	d.dirty = true
	return nil
}
