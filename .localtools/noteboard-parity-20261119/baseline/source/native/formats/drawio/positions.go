package drawio

import (
	"encoding/xml"
	"fmt"
	"math"
	"sort"
	"strings"
)

// PositionUpdate changes a vertex location without changing its dimensions.
type PositionUpdate struct {
	CellID string
	X, Y   float64
}

// Alignment uses the bounding box of all selected vertices, without grid snapping.
type Alignment uint8

const (
	AlignLeft Alignment = iota
	AlignCenterX
	AlignRight
	AlignTop
	AlignCenterY
	AlignBottom
)

// ValidatePositionTargets accepts one or more unique, absolute layer-level leaves.
// Qualification reads the private XML, not mutable exported model or Source views.
func (d *Document) ValidatePositionTargets(pageIndex int, cellIDs []string) error {
	_, err := d.positionTargets(pageIndex, cellIDs)
	return err
}

// PlanAlignment returns only changed positions, in request order. It never edits
// the document. A valid, already aligned selection returns an empty plan.
func (d *Document) PlanAlignment(pageIndex int, cellIDs []string, alignment Alignment) ([]PositionUpdate, error) {
	if alignment > AlignBottom {
		return nil, fmt.Errorf("%w: unknown alignment %d", ErrUnsupported, alignment)
	}
	if len(cellIDs) < 2 {
		return nil, fmt.Errorf("%w: alignment needs at least two vertices", ErrUnsupported)
	}
	targets, err := d.positionTargets(pageIndex, cellIDs)
	if err != nil {
		return nil, err
	}
	left, top := targets[0].x, targets[0].y
	right, bottom := left+targets[0].width, top+targets[0].height
	for _, target := range targets[1:] {
		left = math.Min(left, target.x)
		top = math.Min(top, target.y)
		right = math.Max(right, target.x+target.width)
		bottom = math.Max(bottom, target.y+target.height)
	}
	if !finite(right-left) || !finite(bottom-top) {
		return nil, fmt.Errorf("%w: selection bounds overflow", ErrInvalidGeometry)
	}
	centerX, centerY := left+(right-left)/2, top+(bottom-top)/2
	updates := make([]PositionUpdate, 0, len(targets))
	for _, target := range targets {
		x, y := target.x, target.y
		switch alignment {
		case AlignLeft:
			x = left
		case AlignCenterX:
			x = centerX - target.width/2
		case AlignRight:
			x = right - target.width
		case AlignTop:
			y = top
		case AlignCenterY:
			y = centerY - target.height/2
		case AlignBottom:
			y = bottom - target.height
		}
		if !validPositionBounds(x, y, target.width, target.height) {
			return nil, fmt.Errorf("%w: aligned bounds of cell %q overflow", ErrInvalidGeometry, target.id)
		}
		if x != target.x || y != target.y {
			updates = append(updates, PositionUpdate{CellID: target.id, X: x, Y: y})
		}
	}
	return updates, nil
}

// UpdatePositions applies all positions or none. Each changed x/y value is
// patched in place; untouched attributes, quoting and unknown XML keep their
// original bytes. Empty updates on a writable page and numeric no-ops do not
// change Dirty, Source, original bytes or parsed state.
func (d *Document) UpdatePositions(pageIndex int, updates []PositionUpdate) error {
	if _, err := d.writablePage(pageIndex); err != nil {
		return err
	}
	if len(updates) == 0 {
		return nil
	}
	ids := make([]string, len(updates))
	for i, update := range updates {
		ids[i] = update.CellID
		if !finite(update.X) || !finite(update.Y) {
			return fmt.Errorf("%w: position of cell %q is not finite", ErrInvalidGeometry, update.CellID)
		}
	}
	targets, err := d.positionTargets(pageIndex, ids)
	if err != nil {
		return err
	}
	type patch struct {
		start, end int
		text       []byte
	}
	patches := make([]patch, 0, len(updates))
	size := int64(len(d.source))
	for i, target := range targets {
		update := updates[i]
		if !validPositionBounds(update.X, update.Y, target.width, target.height) {
			return fmt.Errorf("%w: destination bounds of cell %q overflow", ErrInvalidGeometry, target.id)
		}
		attributes := make(map[string]string, 2)
		if target.x != update.X {
			attributes["x"] = formatFloat(update.X)
		}
		if target.y != update.Y {
			attributes["y"] = formatFloat(update.Y)
		}
		if len(attributes) == 0 {
			continue
		}
		tag, err := setTagAttributes(d.source[target.start:target.end], attributes)
		if err != nil {
			return err
		}
		size += int64(len(tag)) - int64(target.end-target.start)
		patches = append(patches, patch{target.start, target.end, tag})
	}
	if len(patches) == 0 {
		return nil
	}
	if size > int64(d.options.MaxBytes) {
		return fmt.Errorf("%w: position edits exceed %d bytes", ErrLimitExceeded, d.options.MaxBytes)
	}
	sort.Slice(patches, func(i, j int) bool { return patches[i].start < patches[j].start })
	previous := 0
	for _, patch := range patches {
		if patch.start < previous || patch.end <= patch.start || patch.end > len(d.source) {
			return fmt.Errorf("%w: overlapping geometry ranges", ErrUnsupported)
		}
		previous = patch.end
	}
	next := make([]byte, 0, int(size))
	previous = 0
	for _, patch := range patches {
		next = append(next, d.source[previous:patch.start]...)
		next = append(next, patch.text...)
		previous = patch.end
	}
	next = append(next, d.source[previous:]...)
	return d.rewriteAndReparse(next)
}

type positionTarget struct {
	id                  string
	x, y, width, height float64
	start, end          int
}

func validPositionBounds(x, y, width, height float64) bool {
	return finite(x) && finite(y) && finite(width) && finite(height) &&
		width > 0 && height > 0 && finite(x+width) && finite(y+height)
}

// The scanner starts at the document root to retain namespace bindings inherited
// from mxfile/diagram. Starting at a sliced model would lose default namespaces.
// It stops at the selected model's end and never decompresses another page.
func (d *Document) positionTargets(pageIndex int, ids []string) ([]positionTarget, error) {
	page, err := d.writablePage(pageIndex)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: select at least one vertex", ErrUnsupported)
	}
	requested := make(map[string]*positionTarget, len(ids))
	for _, id := range ids {
		if id == "" {
			return nil, fmt.Errorf("%w: empty cell ID", ErrCellNotFound)
		}
		if id == "0" || id == "1" {
			return nil, fmt.Errorf("%w: root/layer cells have no editable position", ErrUnsupported)
		}
		if _, exists := requested[id]; exists {
			return nil, fmt.Errorf("%w: requested cell %q", ErrDuplicateID, id)
		}
		requested[id] = &positionTarget{id: id, start: -1}
	}
	if page.Compressed {
		return nil, &ReadOnlyError{Page: pageIndex, Reason: "compressed page"}
	}
	if page.modelStart < 0 || page.modelStart >= len(d.source) {
		return nil, fmt.Errorf("%w: invalid model source range", ErrUnsupported)
	}
	parser := newXMLParser(d.source, d.options)
	var names []xml.Name
	modelDepth, rootDepth, rootCount := 0, 0, 0
	targetDepth, geometryDepth, containerDepth := 0, 0, 0
	containerID := ""
	var active *positionTarget
	seen := make(map[string]bool, len(ids)+2)
	containers := make(map[string]bool, 2)
	children := make(map[string]bool)
	for {
		token, start, end, err := parser.next()
		if err != nil {
			return nil, err
		}
		switch typed := token.(type) {
		case xml.StartElement:
			depth := len(names) + 1
			if err := parser.acceptStart(typed, depth-1); err != nil {
				return nil, err
			}
			if start == page.modelStart {
				if typed.Name != (xml.Name{Local: "mxGraphModel"}) {
					return nil, fmt.Errorf("%w: model is not unqualified native XML", ErrUnsupported)
				}
				modelDepth = depth
			}
			if modelDepth != 0 {
				if containerDepth != 0 {
					return nil, fmt.Errorf("%w: canonical container %q has children", ErrUnsupported, containerID)
				}
				if depth == modelDepth+1 && typed.Name.Local == "root" {
					rootCount++
					if rootCount != 1 || typed.Name.Space != "" {
						return nil, fmt.Errorf("%w: ambiguous or namespaced root container", ErrUnsupported)
					}
					rootDepth = depth
				}
				// Only actual mxCell elements declare cell-level parentage.
				// Extension metadata elements with a parent attribute are preserved but
				// must not force the target into a non-leaf classification.
				if typed.Name.Local == "mxCell" {
					for _, attr := range typed.Attr {
						if attr.Name.Local == "parent" && requested[attr.Value] != nil {
							children[attr.Value] = true
						}
					}
				}
				if active != nil && depth > targetDepth {
					if typed.Name.Local == "mxGeometry" {
						if depth != targetDepth+1 || typed.Name.Space != "" || active.start >= 0 {
							return nil, fmt.Errorf("%w: cell %q has ambiguous geometry", ErrUnsupported, active.id)
						}
						if err := readPositionGeometry(typed, active); err != nil {
							return nil, err
						}
						active.start, active.end = start, end
						geometryDepth = depth
					} else if geometryDepth != 0 || typed.Name.Local == "mxCell" {
						return nil, fmt.Errorf("%w: cell %q contains nested geometry or cells", ErrUnsupported, active.id)
					}
				}
				if typed.Name.Local == "mxCell" {
					id := ""
					// Locate any relevant local-name id first; the strict reader
					// then rejects duplicate and namespaced aliases of that id.
					for _, attr := range typed.Attr {
						if attr.Name.Local == "id" && (requested[attr.Value] != nil || attr.Value == "0" || attr.Value == "1") {
							id = attr.Value
							break
						}
					}
					if id != "" {
						attrs, err := positionAttributes(typed, "id parent vertex edge style value source target collapsed visible")
						if err != nil {
							return nil, err
						}
						if seen[id] {
							return nil, fmt.Errorf("%w: source cell %q", ErrDuplicateID, id)
						}
						seen[id] = true
						if rootDepth == 0 || depth != rootDepth+1 || typed.Name.Space != "" {
							return nil, fmt.Errorf("%w: cell %q is not a direct native layer child", ErrUnsupported, id)
						}
						if id == "0" || id == "1" {
							if !isCanonicalContainerStart(typed, id == "0") || (id == "1" && !containers["0"]) {
								return nil, fmt.Errorf("%w: noncanonical root/layer cell %q", ErrUnsupported, id)
							}
							containerDepth, containerID = depth, id
						} else {
							if err := validatePositionCell(id, attrs); err != nil {
								return nil, err
							}
							active, targetDepth = requested[id], depth
						}
					}
				}
			}
			names = append(names, typed.Name)
		case xml.CharData:
			if err := parser.accountText(typed); err != nil {
				return nil, err
			}
			if (containerDepth != 0 || geometryDepth != 0) && !isXMLWhitespace(typed) {
				return nil, fmt.Errorf("%w: text in a geometry/container", ErrUnsupported)
			}
		case xml.ProcInst:
			if containerDepth != 0 {
				return nil, fmt.Errorf("%w: instruction in canonical container", ErrUnsupported)
			}
		case xml.Directive:
			return nil, fmt.Errorf("%w: XML directives are not editable", ErrUnsupported)
		case xml.EndElement:
			depth := len(names)
			if depth == 0 {
				return nil, fmt.Errorf("%w: invalid XML nesting", ErrInvalidXML)
			}
			if geometryDepth == depth {
				geometryDepth = 0
			}
			if targetDepth == depth {
				if active.start < 0 {
					return nil, fmt.Errorf("%w: cell %q", ErrGeometryMissing, active.id)
				}
				active, targetDepth = nil, 0
			}
			if containerDepth == depth {
				containers[containerID] = true
				containerDepth, containerID = 0, ""
			}
			if rootDepth == depth {
				if start != page.rootClose {
					return nil, fmt.Errorf("%w: root source identity changed", ErrUnsupported)
				}
				rootDepth = 0
			}
			names = names[:depth-1]
			if modelDepth != 0 && depth == modelDepth {
				if rootCount != 1 || !containers["0"] || !containers["1"] {
					return nil, fmt.Errorf("%w: canonical root/layer pair is required", ErrUnsupported)
				}
				result := make([]positionTarget, 0, len(ids))
				for _, id := range ids {
					target := requested[id]
					if !seen[id] {
						return nil, fmt.Errorf("%w: %q", ErrCellNotFound, id)
					}
					if children[id] {
						return nil, fmt.Errorf("%w: cell %q is a parent, not a leaf", ErrUnsupported, id)
					}
					if target.start < 0 || target.end <= target.start || target.end > len(d.source) {
						return nil, fmt.Errorf("%w: cell %q source range", ErrGeometryMissing, id)
					}
					result = append(result, *target)
				}
				return result, nil
			}
		}
	}
}

// Local-name collisions are ambiguous to the existing model reader, even when
// XML allows distinct expanded names. Critical attributes must be unqualified;
// namespace declarations with a critical local name are ambiguous too.
func positionAttributes(start xml.StartElement, critical string) (map[string]string, error) {
	result := make(map[string]string, len(start.Attr))
	for _, attr := range start.Attr {
		criticalName := strings.Contains(" "+critical+" ", " "+attr.Name.Local+" ")
		if attr.Name.Space == "xmlns" && criticalName {
			return nil, fmt.Errorf("%w: namespace declaration shadows %s on %s", ErrUnsupported, attr.Name.Local, start.Name.Local)
		}
		if attr.Name.Space == "xmlns" || (attr.Name.Space == "" && attr.Name.Local == "xmlns") {
			continue
		}
		if _, exists := result[attr.Name.Local]; exists {
			return nil, fmt.Errorf("%w: ambiguous %s attribute on %s", ErrUnsupported, attr.Name.Local, start.Name.Local)
		}
		if attr.Name.Space != "" && strings.Contains(" "+critical+" ", " "+attr.Name.Local+" ") {
			return nil, fmt.Errorf("%w: namespaced %s attribute on %s", ErrUnsupported, attr.Name.Local, start.Name.Local)
		}
		result[attr.Name.Local] = attr.Value
	}
	return result, nil
}

func validatePositionCell(id string, attrs map[string]string) error {
	if attrs["id"] != id || attrs["parent"] != "1" || !isTruthy(attrs["vertex"]) {
		return fmt.Errorf("%w: cell %q is not a layer-level vertex", ErrUnsupported, id)
	}
	for _, name := range []string{"edge", "collapsed"} {
		if value, ok := attrs[name]; ok && !positionFalse(value) {
			return fmt.Errorf("%w: cell %q has %s=%q", ErrUnsupported, id, name, value)
		}
	}
	if value, ok := attrs["visible"]; ok && !isTruthy(value) {
		return fmt.Errorf("%w: cell %q is hidden", ErrUnsupported, id)
	}
	for _, name := range []string{"source", "target", "rotation", "transform", "scale", "offset", "relative"} {
		if _, exists := attrs[name]; exists {
			return fmt.Errorf("%w: cell %q has %s", ErrUnsupported, id, name)
		}
	}
	if err := validatePositionStyle(attrs["style"]); err != nil {
		return fmt.Errorf("cell %q: %w", id, err)
	}
	return nil
}

func readPositionGeometry(start xml.StartElement, target *positionTarget) error {
	attrs, err := positionAttributes(start, "x y width height as relative offset rotation transform scale")
	if err != nil {
		return err
	}
	if attrs["as"] != "geometry" {
		return fmt.Errorf("%w: cell %q geometry role", ErrUnsupported, target.id)
	}
	if value, exists := attrs["relative"]; exists && !positionFalse(value) {
		return fmt.Errorf("%w: cell %q uses relative geometry", ErrUnsupported, target.id)
	}
	for _, name := range []string{"offset", "rotation", "transform", "scale", "translate", "alternateBounds"} {
		if _, exists := attrs[name]; exists {
			return fmt.Errorf("%w: cell %q geometry has %s", ErrUnsupported, target.id, name)
		}
	}
	values := []*float64{&target.x, &target.y, &target.width, &target.height}
	for i, name := range []string{"x", "y", "width", "height"} {
		value, exists := attrs[name]
		if !exists && i < 2 {
			continue
		}
		number, err := parseFinite(value)
		if err != nil {
			return fmt.Errorf("%w: cell %q has invalid %s", ErrInvalidGeometry, target.id, name)
		}
		*values[i] = number
	}
	if !validPositionBounds(target.x, target.y, target.width, target.height) {
		return fmt.Errorf("%w: cell %q has nonpositive or overflowing bounds", ErrInvalidGeometry, target.id)
	}
	return nil
}

func positionFalse(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "0", "false", "no":
		return true
	default:
		return false
	}
}

// Unknown named styles can hide a group or shape transform. Accept only the
// native shape selectors and presentation keys whose geometry is understood.
func validatePositionStyle(style string) error {
	if isImageStyle(style) || isStencilStyle(style) {
		return fmt.Errorf("%w: image or stencil geometry", ErrInvalidShape)
	}
	seen := make(map[string]bool)
	for _, part := range strings.Split(style, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, equals := strings.Cut(part, "=")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if seen[key] {
			return fmt.Errorf("%w: duplicate style key %q", ErrUnsupported, key)
		}
		seen[key] = true
		if !equals {
			if key != "ellipse" && key != "rhombus" && key != "rounded" {
				return fmt.Errorf("%w: named style %q", ErrUnsupported, key)
			}
			continue
		}
		switch key {
		case "shape":
			if value != "rectangle" && value != "ellipse" && value != "rhombus" {
				return fmt.Errorf("%w: shape %q", ErrInvalidShape, value)
			}
		case "rounded", "ellipse", "rhombus":
			if !isTruthy(value) && !positionFalse(value) {
				return fmt.Errorf("%w: shape selector %q", ErrUnsupported, key)
			}
		case "rotation":
			angle, err := parseFinite(value)
			if err != nil || angle != 0 {
				return fmt.Errorf("%w: rotated geometry", ErrUnsupported)
			}
		case "flipH", "flipV":
			if !positionFalse(value) {
				return fmt.Errorf("%w: flipped geometry", ErrUnsupported)
			}
		case "whiteSpace", "html", "fillColor", "strokeColor", "strokeWidth", "fontColor", "fontFamily", "fontSize", "fontStyle",
			"align", "verticalAlign", "spacing", "spacingTop", "spacingBottom", "spacingLeft", "spacingRight", "labelPosition",
			"verticalLabelPosition", "labelBackgroundColor", "labelBorderColor", "overflow", "textOpacity", "opacity", "fillOpacity",
			"strokeOpacity", "dashed", "dashPattern", "fixDash", "shadow", "glass", "gradientColor", "gradientDirection", "arcSize", "absoluteArcSize":
		default:
			return fmt.Errorf("%w: style key %q has no position-edit contract", ErrUnsupported, key)
		}
	}
	return nil
}
