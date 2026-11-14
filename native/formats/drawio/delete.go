package drawio

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// DeleteCell removes one understood layer-level vertex or edge without
// serializing the page. The result counts incident edges removed with a vertex.
// Unsupported dependencies are refused before the private source is changed.
func (d *Document) DeleteCell(pageIndex int, cellID string) (int, error) {
	page, err := d.writablePage(pageIndex)
	if err != nil {
		return 0, err
	}
	if cellID == "0" || cellID == "1" {
		return 0, fmt.Errorf("%w: root and layer cells cannot be deleted", ErrUnsupported)
	}
	found, vertex := 0, false
	for _, node := range page.Model.Nodes {
		if node.ID == cellID {
			found++
			vertex = true
			if node.Parent != "1" || len(node.Unsupported) != 0 || node.geometryStart < 0 {
				return 0, fmt.Errorf("%w: cell %q is not an understood layer-level vertex", ErrUnsupported, cellID)
			}
		}
	}
	for _, edge := range page.Model.Edges {
		if edge.ID == cellID {
			found++
			if edge.Parent != "1" || len(edge.Unsupported) != 0 {
				return 0, fmt.Errorf("%w: cell %q is not an understood layer-level edge", ErrUnsupported, cellID)
			}
		}
	}
	if found == 0 || cellID == "" {
		return 0, fmt.Errorf("%w: %q", ErrCellNotFound, cellID)
	}
	if found != 1 {
		return 0, fmt.Errorf("%w: cell %q has ambiguous vertex/edge roles", ErrUnsupported, cellID)
	}
	removeIDs := map[string]bool{cellID: true}
	incidentEdges := 0
	if vertex {
		for _, edge := range page.Model.Edges {
			if edge.SourceID != cellID && edge.TargetID != cellID {
				continue
			}
			if edge.Parent != "1" || len(edge.Unsupported) != 0 || edge.ID == "0" || edge.ID == "1" {
				return 0, fmt.Errorf("%w: incident edge %q is not an understood layer-level edge", ErrUnsupported, edge.ID)
			}
			if !removeIDs[edge.ID] {
				removeIDs[edge.ID] = true
				incidentEdges++
			}
		}
	}
	for _, node := range page.Model.Nodes {
		if !removeIDs[node.ID] && removeIDs[node.Parent] {
			return 0, fmt.Errorf("%w: deletion would remove the parent of vertex %q", ErrUnsupported, node.ID)
		}
	}
	for _, edge := range page.Model.Edges {
		if !removeIDs[edge.ID] && (removeIDs[edge.Parent] || removeIDs[edge.SourceID] || removeIDs[edge.TargetID]) {
			return 0, fmt.Errorf("%w: surviving edge %q depends on a removed cell", ErrUnsupported, edge.ID)
		}
	}
	for _, issue := range d.Issues {
		if issue.Page == pageIndex && issue.Code == IssueUnsupportedElement {
			return 0, fmt.Errorf("%w: page contains an unrecognized cell or XML structure", ErrUnsupported)
		}
	}
	spans, err := d.deleteCellRanges(page, removeIDs)
	if err != nil {
		return 0, err
	}
	removedBytes := 0
	for _, span := range spans {
		removedBytes += span.end - span.start
	}
	next := make([]byte, 0, len(d.source)-removedBytes)
	previous := 0
	for _, span := range spans {
		next = append(next, d.source[previous:span.start]...)
		previous = span.end
	}
	next = append(next, d.source[previous:]...)
	if err := d.rewriteAndReparse(next); err != nil {
		return 0, err
	}
	return incidentEdges, nil
}

type deleteSourceRange struct {
	start, end int
}

// Parser model offsets point to end-tag starts, not complete cell endings.
// Token offsets determine complete direct mxCell spans, including end tags,
// and inspect raw references rather than trusting only rendered objects.
func (d *Document) deleteCellRanges(page *Page, removeIDs map[string]bool) ([]deleteSourceRange, error) {
	base := page.modelStart
	if base < 0 || base >= len(d.source) {
		return nil, fmt.Errorf("%w: invalid model source range", ErrUnsupported)
	}
	parser := newXMLParser(d.source[base:], d.options)
	spans := make([]deleteSourceRange, 0, len(removeIDs))
	seen := make(map[string]bool, len(removeIDs))
	var names []xml.Name
	selectedDepth, selectedIndex := 0, -1
	for {
		token, start, end, err := parser.next()
		if err != nil {
			return nil, err
		}
		switch typed := token.(type) {
		case xml.StartElement:
			if err := parser.acceptStart(typed, len(names)); err != nil {
				return nil, err
			}
			if len(names) == 0 && (typed.Name.Local != "mxGraphModel" || typed.Name.Space != "") {
				return nil, fmt.Errorf("%w: model is not an uncompressed native XML model", ErrUnsupported)
			}
			id := attrValue(typed, "id")
			isTarget := len(names) == 2 && names[1].Local == "root" && names[1].Space == "" &&
				typed.Name.Local == "mxCell" && typed.Name.Space == "" && removeIDs[id]
			if isTarget {
				if seen[id] {
					return nil, fmt.Errorf("%w: ambiguous cell source", ErrUnsupported)
				}
				seen[id] = true
				spans = append(spans, deleteSourceRange{start: base + start, end: -1})
				selectedIndex = len(spans) - 1
				selectedDepth = len(names) + 1
			}
			if selectedDepth == 0 {
				for _, attr := range typed.Attr {
					switch attr.Name.Local {
					case "parent", "source", "target", "sourcePort", "targetPort", "defaultParent", "currentRoot":
						if removeIDs[attr.Value] {
							return nil, fmt.Errorf("%w: surviving XML references cell %q", ErrUnsupported, attr.Value)
						}
					case "style":
						// Duplicate port keys must not hide a surviving reference.
						for style := attr.Value; style != ""; {
							part, rest, _ := strings.Cut(style, ";")
							style = rest
							key, value, ok := strings.Cut(part, "=")
							key = strings.TrimSpace(key)
							if ok && (key == "sourcePort" || key == "targetPort") && removeIDs[strings.TrimSpace(value)] {
								return nil, fmt.Errorf("%w: surviving port reference to a removed cell", ErrUnsupported)
							}
						}
					}
				}
			}
			names = append(names, typed.Name)
		case xml.CharData:
			if err := parser.accountText(typed); err != nil {
				return nil, err
			}
		case xml.Directive:
			return nil, fmt.Errorf("%w: XML directives are not editable", ErrUnsupported)
		case xml.EndElement:
			if len(names) == 0 {
				return nil, fmt.Errorf("%w: invalid XML nesting", ErrUnsupported)
			}
			if selectedDepth == len(names) {
				spans[selectedIndex].end = base + end
				selectedDepth, selectedIndex = 0, -1
			}
			names = names[:len(names)-1]
			if len(names) == 0 {
				if len(spans) != len(removeIDs) {
					return nil, fmt.Errorf("%w: no complete direct cell source range", ErrUnsupported)
				}
				previous := 0
				for _, span := range spans {
					if span.start < previous || span.end <= span.start || span.end > len(d.source) {
						return nil, fmt.Errorf("%w: invalid or overlapping cell source ranges", ErrUnsupported)
					}
					previous = span.end
				}
				return spans, nil
			}
		}
	}
}
