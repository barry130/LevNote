package drawio

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// CloneCells atomically copies ordered layer-level vertices plus edges wholly
// internal to that selection. Existing XML bytes are retained; no grouping is
// implied and outer endpoints remain attached to their original cells.
func (d *Document) CloneCells(pageIndex int, cellIDs []string, dx, dy float64) ([]string, error) {
	page, err := d.writablePage(pageIndex)
	if err != nil {
		return nil, err
	}
	if len(cellIDs) == 0 {
		return nil, fmt.Errorf("%w: empty clone selection", ErrCellNotFound)
	}
	if !finite(dx) || !finite(dy) {
		return nil, fmt.Errorf("%w: nonfinite clone offset", ErrInvalidGeometry)
	}
	selected := make(map[string]bool, len(cellIDs))
	for _, id := range cellIDs {
		if id == "" {
			return nil, fmt.Errorf("%w: empty clone cell ID", ErrCellNotFound)
		}
		if selected[id] {
			return nil, fmt.Errorf("%w: requested clone cell %q", ErrDuplicateID, id)
		}
		selected[id] = true
	}
	cells, occupied, err := d.scanClonePage(page, selected, dx, dy)
	if err != nil {
		return nil, err
	}
	used := make(map[string]bool, len(cells)+2)
	for id := range occupied {
		used[id] = true
	}
	remap := make(map[string]string, len(cells))
	vertices := make([]string, 0, len(cellIDs))
	nextID := func() string {
		for index := 1; ; index++ {
			id := "levnote-" + strconv.Itoa(index)
			if !used[id] {
				used[id] = true
				return id
			}
		}
	}
	for _, id := range cellIDs {
		remap[id] = nextID()
		vertices = append(vertices, remap[id])
	}
	for index := range cells {
		cell := &cells[index]
		if !cell.vertex && cell.emit {
			remap[cell.id] = nextID()
		}
	}
	emitOrder := make([]int, 0, len(cellIDs)+len(cells))
	byID := make(map[string]int, len(cells))
	for index := range cells {
		if cells[index].vertex {
			byID[cells[index].id] = index
		}
	}
	for _, id := range cellIDs {
		emitOrder = append(emitOrder, byID[id])
	}
	for index := range cells {
		if !cells[index].vertex {
			emitOrder = append(emitOrder, index)
		}
	}
	var insertion []byte
	for _, orderedIndex := range emitOrder {
		cell := &cells[orderedIndex]
		if !cell.emit {
			continue
		}
		updates := map[string]string{"id": remap[cell.id]}
		if !cell.vertex {
			updates["source"], updates["target"] = remap[cell.source], remap[cell.target]
		}
		startTag, err := setTagAttributes(d.source[cell.start:cell.tagEnd], updates)
		if err != nil {
			return nil, err
		}
		insertion = append(insertion, startTag...)
		if !cell.vertex {
			insertion = append(insertion, d.source[cell.tagEnd:cell.end]...)
			continue
		}
		geometry, err := setTagAttributes(d.source[cell.geometryStart:cell.geometryEnd], map[string]string{
			"x": formatFloat(cell.x + dx), "y": formatFloat(cell.y + dy),
		})
		if err != nil {
			return nil, err
		}
		insertion = append(insertion, d.source[cell.tagEnd:cell.geometryStart]...)
		insertion = append(insertion, geometry...)
		insertion = append(insertion, d.source[cell.geometryEnd:cell.end]...)
	}
	if page.rootClose < 0 || page.rootClose > len(d.source) {
		return nil, fmt.Errorf("%w: no editable clone root", ErrUnsupported)
	}
	if int64(len(d.source))+int64(len(insertion)) > int64(d.options.MaxBytes) {
		return nil, fmt.Errorf("%w: clone exceeds %d bytes", ErrLimitExceeded, d.options.MaxBytes)
	}
	next := make([]byte, 0, len(d.source)+len(insertion))
	next = append(next, d.source[:page.rootClose]...)
	next = append(next, insertion...)
	next = append(next, d.source[page.rootClose:]...)
	if err := d.rewriteAndReparse(next); err != nil {
		return nil, err
	}
	return vertices, nil
}

type cloneCell struct {
	id, source, target         string
	start, end, tagEnd         int
	geometryStart, geometryEnd int
	x, y                       float64
	vertex, emit               bool
}

type cloneScan struct {
	page                       *Page
	selected                   map[string]bool
	source                     []byte
	dx, dy                     float64
	model                      map[string]Node
	ordered                    []cloneCell
	active                     *cloneCell
	activeDepth                int
	geometryDepth, arrayDepth  int
	pointDepth                 int
	modelDepth, rootDepth      int
	rootCount                  int
	modelDone                  bool
}

func (s *cloneScan) start(value xml.StartElement, start, end, depth int) error {
	if start == s.page.modelStart {
		if value.Name != (xml.Name{Local: "mxGraphModel"}) || s.modelDepth != 0 {
			return fmt.Errorf("%w: unsupported model XML", ErrUnsupported)
		}
		s.modelDepth = depth
	}
	if s.modelDepth == 0 || s.modelDone {
		return nil
	}
	if err := cloneReferenceGuard(value, s.selected, s.active != nil && depth > s.activeDepth); err != nil {
		return err
	}
	if depth == s.modelDepth+1 {
		if value.Name != (xml.Name{Local: "root"}) {
			return fmt.Errorf("%w: model root", ErrUnsupported)
		}
		s.rootCount++
		if s.rootCount != 1 {
			return fmt.Errorf("%w: ambiguous root", ErrUnsupported)
		}
		s.rootDepth = depth
	}
	if s.rootDepth > 0 && depth == s.rootDepth+1 && value.Name == (xml.Name{Local: "mxCell"}) {
		attrs, err := positionAttributes(value, "id parent vertex edge source target")
		if err != nil {
			return err
		}
		id := attrs["id"]
		if id == "0" || id == "1" {
			if !isCanonicalContainerStart(value, id == "0") {
				return fmt.Errorf("%w: canonical root/layer %q", ErrUnsupported, id)
			}
		} else if attrs["parent"] != "1" {
			return fmt.Errorf("%w: noncanonical parent cell %q", ErrUnsupported, id)
		}
		if id != "0" && id != "1" && (s.selected[id] || (isTruthy(attrs["edge"]) && s.selected[attrs["source"]] && s.selected[attrs["target"]])) {
			if s.selected[id] {
				node, ok := s.model[id]
				if !ok || !isTruthy(attrs["vertex"]) || node.Parent != "1" || len(node.Unsupported) != 0 {
					return fmt.Errorf("%w: cell %q is not an understood vertex", ErrUnsupported, id)
				}
			}
			if err := cloneReferenceGuard(value, s.selected, true); err != nil {
				return err
			}
			tagEnd := cloneStartTagEnd(s.source[start:end])
			if tagEnd < 0 {
				return fmt.Errorf("%w: start tag of cell %q", ErrInvalidXML, id)
			}
			if s.active != nil {
				return fmt.Errorf("%w: nested clone cell", ErrUnsupported)
			}
			s.ordered = append(s.ordered, cloneCell{id: id, source: attrs["source"], target: attrs["target"], start: start, tagEnd: start + tagEnd, geometryStart: -1, vertex: s.selected[id], emit: true})
			s.active = &s.ordered[len(s.ordered)-1]
			s.activeDepth = depth
		}
	} else if s.active != nil && depth > s.activeDepth {
		if err := cloneReferenceGuard(value, s.selected, true); err != nil {
			return err
		}
		if depth == s.activeDepth+1 {
			if value.Name != (xml.Name{Local: "mxGeometry"}) || s.active.geometryStart >= 0 {
				return fmt.Errorf("%w: clone cell %q has unsupported XML", ErrUnsupported, s.active.id)
			}
			if s.active.vertex {
				target := positionTarget{id: s.active.id}
				if err := readPositionGeometry(value, &target); err != nil {
					return err
				}
				if !validPositionBounds(target.x+s.dx, target.y+s.dy, target.width, target.height) {
					return fmt.Errorf("%w: cloned bounds overflow for %q", ErrInvalidGeometry, s.active.id)
				}
				s.active.x, s.active.y = target.x, target.y
			} else if attrValue(value, "as") != "geometry" {
				return fmt.Errorf("%w: edge geometry role", ErrUnsupported)
			}
			tagEnd := cloneStartTagEnd(s.source[start:end])
			if tagEnd < 0 {
				return fmt.Errorf("%w: geometry tag of cell %q", ErrInvalidXML, s.active.id)
			}
			s.active.geometryStart, s.active.geometryEnd = start, start+tagEnd
			s.geometryDepth = depth
		} else if depth == s.geometryDepth+1 || depth == s.arrayDepth+1 {
			switch {
			case depth == s.geometryDepth+1 && s.arrayDepth == 0 && s.pointDepth == 0 && value.Name == (xml.Name{Local: "Array"}):
				if attrValue(value, "as") != "points" {
					return fmt.Errorf("%w: invalid clone waypoint Array", ErrUnsupported)
				}
				s.arrayDepth = depth
			case value.Name == (xml.Name{Local: "mxPoint"}):
				if s.pointDepth != 0 || (depth != s.geometryDepth+1 && (s.arrayDepth == 0 || depth != s.arrayDepth+1)) {
					return fmt.Errorf("%w: nested or misplaced clone point", ErrUnsupported)
				}
				if _, err := clonePointRole(value); err != nil {
					return err
				}
				s.pointDepth = depth
			default:
				return fmt.Errorf("%w: clone cell %q has unknown geometry child", ErrUnsupported, s.active.id)
			}
		} else {
			return fmt.Errorf("%w: clone cell %q has unsupported geometry subtree", ErrUnsupported, s.active.id)
		}
	}
	return nil
}

func (s *cloneScan) end(value xml.EndElement, start, end, depth int) error {
	if s.modelDepth != 0 && !s.modelDone {
		if s.activeDepth == depth {
			if s.active == nil || s.active.geometryStart < 0 {
				return fmt.Errorf("%w: clone cell", ErrGeometryMissing)
			}
			s.active.end = end
			s.active, s.activeDepth = nil, 0
		}
		if s.pointDepth == depth {
			s.pointDepth = 0
		}
		if s.arrayDepth == depth {
			s.arrayDepth = 0
		}
		if s.geometryDepth == depth {
			s.geometryDepth = 0
		}
		if value.Name == (xml.Name{Local: "root"}) && s.rootDepth == depth {
			if start != s.page.rootClose {
				return fmt.Errorf("%w: root source identity", ErrUnsupported)
			}
			s.rootDepth = 0
		}
		if value.Name == (xml.Name{Local: "mxGraphModel"}) && s.modelDepth == depth {
			if s.rootCount != 1 || s.rootDepth != 0 {
				return fmt.Errorf("%w: missing clone root", ErrUnsupported)
			}
			s.modelDone = true
		}
	}
	return nil
}

func (s *cloneScan) finish() ([]cloneCell, map[string]bool, error) {
	if !s.modelDone {
		return nil, nil, fmt.Errorf("%w: missing clone root", ErrUnsupported)
	}
	seen := map[string]bool{}
	for _, cell := range s.ordered {
		if cell.vertex && cell.emit {
			seen[cell.id] = true
		}
	}
	for id := range s.selected {
		if !seen[id] {
			return nil, nil, fmt.Errorf("%w: clone cell %q", ErrCellNotFound, id)
		}
	}
	return s.ordered, nil, nil
}

// scanClonePage scans exactly the selected raw diagram. Other diagrams and
// mxfile-level vendor IDs stay outside its ID allocation namespace. It never
// reconstructs a Page by choosing a same-named or similarly-shaped diagram.
func (d *Document) scanClonePage(page *Page, selected map[string]bool, dx, dy float64) ([]cloneCell, map[string]bool, error) {
	if page.modelStart < 0 || page.modelStart >= len(d.source) || page.rootClose <= page.modelStart || page.Index < 0 || page.Index >= len(d.Pages) {
		return nil, nil, fmt.Errorf("%w: invalid clone model source", ErrUnsupported)
	}
	s := cloneScan{page: page, selected: selected, source: d.source, dx: dx, dy: dy, model: map[string]Node{}}
	for _, node := range page.Model.Nodes {
		s.model[node.ID] = node
	}
	occupied := map[string]bool{}
	parser := newXMLParser(d.source, d.options)
	var names []xml.Name
	pageNumber, pageDepth := 0, 0
	if !d.IsMxFile {
		pageDepth = 1
	}
	for {
		token, start, end, err := parser.next()
		if err != nil {
			return nil, nil, err
		}
		switch value := token.(type) {
		case xml.StartElement:
			depth := len(names)+1
			if pageDepth == 0 && depth == 2 && value.Name.Local == "diagram" && len(names) == 1 && names[0].Local == "mxfile" {
				if pageNumber == page.Index {
					pageDepth = depth
				}
				pageNumber++
			}
			if pageDepth > 0 {
				if err := parser.acceptStart(value, len(names)); err != nil {
					return nil, nil, err
				}
				for _, attr := range value.Attr {
					if attr.Name.Local == "id" {
						occupied[attr.Value] = true
					}
				}
				if d.IsMxFile && depth > 2 && start < page.modelStart && names[len(names)-1].Local != "diagram" {
					return nil, nil, fmt.Errorf("%w: no raw diagram content before model", ErrUnsupported)
				}
				if start == page.modelStart && ((d.IsMxFile && (depth != 3 || names[depth-2].Local != "diagram")) || (!d.IsMxFile && depth != 1)) {
					return nil, nil, fmt.Errorf("%w: model parent", ErrUnsupported)
				}
				if err := s.start(value, start, end, depth); err != nil {
					return nil, nil, err
				}
			}
			names = append(names, value.Name)
			if !isSelfClosingRaw(d.source[start:end]) {
				break
			}
			next, syntheticStart, _, err := parser.next()
			if err != nil {
				return nil, nil, err
			}
			closeElement, ok := next.(xml.EndElement)
			if !ok || closeElement.Name.Local != value.Name.Local || syntheticStart != end {
				return nil, nil, fmt.Errorf("%w: synthetic end of clone element", ErrInvalidXML)
			}
			if pageDepth > 0 {
				if value.Name == (xml.Name{Local: "root"}) && s.rootDepth == depth {
					return nil, nil, fmt.Errorf("%w: empty root has no insertion anchor", ErrUnsupported)
				}
				if err := s.end(closeElement, end, end, depth); err != nil {
					return nil, nil, err
				}
			}
			names = names[:depth-1]
			if pageDepth == depth {
				if closeElement.Name.Local != "diagram" || !s.modelDone {
					return nil, nil, fmt.Errorf("%w: selected page closed before model", ErrUnsupported)
				}
				cells, _, err := s.finish()
				return cells, occupied, err
			}
		case xml.EndElement:
			depth := len(names)
			if depth == 0 || names[depth-1] != value.Name {
				return nil, nil, fmt.Errorf("%w: mismatched clone XML", ErrInvalidXML)
			}
			if pageDepth > 0 {
				if err := s.end(value, start, end, depth); err != nil {
					return nil, nil, err
				}
			}
			names = names[:depth-1]
			if pageDepth == depth {
				if (d.IsMxFile && value.Name.Local != "diagram") || (!d.IsMxFile && value.Name.Local != "mxGraphModel") {
					return nil, nil, fmt.Errorf("%w: clone page source boundary", ErrUnsupported)
				}
				cells, _, err := s.finish()
				return cells, occupied, err
			}
		case xml.CharData:
			if pageDepth > 0 {
				if err := parser.accountText(value); err != nil {
					return nil, nil, err
				}
				if s.active != nil && !isXMLWhitespace(value) {
					return nil, nil, fmt.Errorf("%w: mixed content in clone cell", ErrUnsupported)
				}
			}
		case xml.Directive:
		if pageDepth > 0 {
				return nil, nil, fmt.Errorf("%w: XML directives in clone source", ErrUnsupported)
			}
		}
	}
}

func cloneStartTagEnd(raw []byte) int {
	quote := byte(0)
	for i, char := range raw {
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			continue
		}
		if char == '>' {
			return i+1
		}
	}
	return -1
}

func clonePointRole(value xml.StartElement) (string, error) {
	for _, name := range []string{"x", "y"} {
		raw := attrValue(value, name)
		if raw != "" {
			if _, err := parseFinite(raw); err != nil {
				return "", fmt.Errorf("%w: clone point %s", ErrInvalidGeometry, name)
			}
		}
	}
	return attrValue(value, "as"), nil
}

func cloneReferenceGuard(value xml.StartElement, selected map[string]bool, insideSelected bool) error {
	for _, attr := range value.Attr {
		if insideSelected && (attr.Name.Local == "sourcePort" || attr.Name.Local == "targetPort") {
			return fmt.Errorf("%w: port on copied cell", ErrUnsupported)
		}
		if !insideSelected && selected[attr.Value] {
			switch attr.Name.Local {
			case "parent", "defaultParent", "currentRoot", "sourcePort", "targetPort":
				return fmt.Errorf("%w: raw reference to clone cell %q", ErrUnsupported, attr.Value)
			case "source", "target":
				if value.Name.Local != "mxCell" {
					return fmt.Errorf("%w: raw endpoint reference to %q", ErrUnsupported, attr.Value)
				}
			}
		}
		if attr.Name.Local != "style" {
			continue
		}
		for _, entry := range strings.Split(attr.Value, ";") {
			key, target, found := strings.Cut(entry, "=")
			if strings.TrimSpace(key) == "sourcePort" || strings.TrimSpace(key) == "targetPort" {
				if insideSelected || (found && selected[strings.TrimSpace(target)]) {
					return fmt.Errorf("%w: raw port reference to selected cell", ErrUnsupported)
				}
			}
		}
	}
	return nil
}
