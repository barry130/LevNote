package drawio

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

func (d *Document) Serialize() ([]byte, error) {
	if d == nil {
		return nil, ErrEmptyDocument
	}
	return cloneBytes(d.source), nil
}

func (d *Document) Page(index int) (*Page, error) {
	if d == nil || index < 0 || index >= len(d.Pages) {
		return nil, fmt.Errorf("%w: %d", ErrPageNotFound, index)
	}
	return &d.Pages[index], nil
}

// UpdateText changes one vertex or edge label attribute. The rest of the start
// tag and the complete source document remain byte-for-byte untouched.
func (d *Document) UpdateText(pageIndex int, cellID, text string) error {
	page, err := d.writablePage(pageIndex)
	if err != nil {
		return err
	}
	if !validXMLCharacters(text) {
		return fmt.Errorf("%w: label contains a character XML 1.0 cannot represent", ErrInvalidXML)
	}
	for _, node := range page.Model.Nodes {
		if node.ID == cellID {
			return d.patchStartTag(node.startTagStart, node.startTagEnd, map[string]string{"value": text})
		}
	}
	for _, edge := range page.Model.Edges {
		if edge.ID == cellID {
			return d.patchStartTag(edge.startTagStart, edge.startTagEnd, map[string]string{"value": text})
		}
	}
	return fmt.Errorf("%w: %q", ErrCellNotFound, cellID)
}

func (d *Document) UpdateGeometry(pageIndex int, cellID string, x, y, width, height float64) error {
	page, err := d.writablePage(pageIndex)
	if err != nil {
		return err
	}
	if !finite(x) || !finite(y) || !finite(width) || !finite(height) || width < 0 || height < 0 {
		return ErrInvalidGeometry
	}
	for _, node := range page.Model.Nodes {
		if node.ID != cellID {
			continue
		}
		if node.geometryStart < 0 || node.geometryEnd <= node.geometryStart {
			return fmt.Errorf("%w: cell %q", ErrGeometryMissing, cellID)
		}
		return d.patchStartTag(node.geometryStart, node.geometryEnd, map[string]string{
			"x":      formatFloat(x),
			"y":      formatFloat(y),
			"width":  formatFloat(width),
			"height": formatFloat(height),
		})
	}
	return fmt.Errorf("%w: %q", ErrCellNotFound, cellID)
}

func (d *Document) AddShape(pageIndex int, spec ShapeSpec) (string, error) {
	page, err := d.writablePage(pageIndex)
	if err != nil {
		return "", err
	}
	if page.rootClose < 0 || page.rootClose > len(d.source) {
		return "", fmt.Errorf("%w: page has no editable root insertion point", ErrUnsupported)
	}
	switch spec.Shape {
	case ShapeRectangle, ShapeEllipse, ShapeDiamond, ShapeRounded:
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidShape, spec.Shape)
	}
	if !finite(spec.X) || !finite(spec.Y) || !finite(spec.Width) || !finite(spec.Height) ||
		spec.Width <= 0 || spec.Height <= 0 {
		return "", ErrInvalidGeometry
	}
	if !validXMLCharacters(spec.Text) {
		return "", fmt.Errorf("%w: label contains invalid XML characters", ErrInvalidXML)
	}
	id := spec.ID
	if id == "" {
		id = d.nextCellID(page)
	}
	if !validXMLCharacters(id) {
		return "", fmt.Errorf("%w: id contains invalid XML characters", ErrInvalidXML)
	}
	if pageHasCell(page, id) {
		return "", fmt.Errorf("%w: %q", ErrDuplicateID, id)
	}
	parent := spec.Parent
	if parent == "" {
		parent = "1"
	}
	cell := "<mxCell id=\"" + escapeAttr(id) + "\" value=\"" + escapeAttr(spec.Text) +
		"\" style=\"" + escapeAttr(styleForShape(spec.Shape)) + "\" vertex=\"1\" parent=\"" +
		escapeAttr(parent) + "\"><mxGeometry x=\"" + formatFloat(spec.X) + "\" y=\"" +
		formatFloat(spec.Y) + "\" width=\"" + formatFloat(spec.Width) + "\" height=\"" +
		formatFloat(spec.Height) + "\" as=\"geometry\"/></mxCell>"
	return id, d.insertAt(page.rootClose, cell)
}

func (d *Document) AddConnector(pageIndex int, spec ConnectorSpec) (string, error) {
	page, err := d.writablePage(pageIndex)
	if err != nil {
		return "", err
	}
	if page.rootClose < 0 || page.rootClose > len(d.source) {
		return "", fmt.Errorf("%w: page has no editable root insertion point", ErrUnsupported)
	}
	if spec.SourceID == "" || spec.TargetID == "" {
		return "", fmt.Errorf("%w: connector source and target are required", ErrInvalidGeometry)
	}
	if !pageHasCell(page, spec.SourceID) {
		return "", fmt.Errorf("%w: source %q", ErrCellNotFound, spec.SourceID)
	}
	if !pageHasCell(page, spec.TargetID) {
		return "", fmt.Errorf("%w: target %q", ErrCellNotFound, spec.TargetID)
	}
	if !validXMLCharacters(spec.Text) {
		return "", fmt.Errorf("%w: label contains invalid XML characters", ErrInvalidXML)
	}
	id := spec.ID
	if id == "" {
		id = d.nextCellID(page)
	}
	if !validXMLCharacters(id) {
		return "", fmt.Errorf("%w: id contains invalid XML characters", ErrInvalidXML)
	}
	if pageHasCell(page, id) {
		return "", fmt.Errorf("%w: %q", ErrDuplicateID, id)
	}
	parent := spec.Parent
	if parent == "" {
		parent = "1"
	}
	style := "edgeStyle=orthogonalEdgeStyle;rounded=0;orthogonalLoop=1;jettySize=auto;html=0;endArrow=classic;endFill=1;"
	cell := "<mxCell id=\"" + escapeAttr(id) + "\" value=\"" + escapeAttr(spec.Text) +
		"\" style=\"" + escapeAttr(style) + "\" edge=\"1\" parent=\"" + escapeAttr(parent) +
		"\" source=\"" + escapeAttr(spec.SourceID) + "\" target=\"" + escapeAttr(spec.TargetID) +
		"\"><mxGeometry relative=\"1\" as=\"geometry\"/></mxCell>"
	return id, d.insertAt(page.rootClose, cell)
}

func (d *Document) writablePage(index int) (*Page, error) {
	if d == nil || index < 0 || index >= len(d.Pages) {
		return nil, fmt.Errorf("%w: %d", ErrPageNotFound, index)
	}
	page := &d.Pages[index]
	if page.ReadOnly {
		return nil, &ReadOnlyError{Page: index, Reason: page.ReadOnlyReason}
	}
	return page, nil
}

func (d *Document) patchStartTag(start, end int, updates map[string]string) error {
	if start < 0 || end <= start || end > len(d.source) {
		return fmt.Errorf("%w: invalid source range", ErrUnsupported)
	}
	nextTag, err := setTagAttributes(d.source[start:end], updates)
	if err != nil {
		return err
	}
	next := make([]byte, 0, len(d.source)-end+start+len(nextTag))
	next = append(next, d.source[:start]...)
	next = append(next, nextTag...)
	next = append(next, d.source[end:]...)
	return d.rewriteAndReparse(next)
}

func (d *Document) insertAt(offset int, value string) error {
	if offset < 0 || offset > len(d.source) {
		return fmt.Errorf("%w: invalid insertion offset", ErrUnsupported)
	}
	next := make([]byte, 0, len(d.source)+len(value))
	next = append(next, d.source[:offset]...)
	next = append(next, value...)
	next = append(next, d.source[offset:]...)
	return d.rewriteAndReparse(next)
}

func (d *Document) rewriteAndReparse(next []byte) error {
	// Callers construct next with a new backing array. The owned parser transfers
	// that array to the staged document; no caller-visible slice may be mutated.
	stage, issues, err := parseOwnedDocumentBytes(next, d.options)
	if err != nil {
		return err
	}
	stage.original = d.original
	stage.dirty = true
	stage.Issues = append([]Issue(nil), issues...)
	*d = *stage
	return nil
}

func pageHasCell(page *Page, id string) bool {
	if page == nil {
		return false
	}
	for _, node := range page.Model.Nodes {
		if node.ID == id {
			return true
		}
	}
	for _, edge := range page.Model.Edges {
		if edge.ID == id {
			return true
		}
	}
	return false
}

func (d *Document) nextCellID(page *Page) string {
	for index := 1; ; index++ {
		candidate := "levnote-" + strconv.Itoa(index)
		if !pageHasCell(page, candidate) {
			return candidate
		}
	}
}

func styleForShape(shape string) string {
	switch shape {
	case ShapeEllipse:
		return "ellipse;whiteSpace=wrap;html=0;"
	case ShapeDiamond:
		return "rhombus;whiteSpace=wrap;html=0;"
	case ShapeRounded:
		return "rounded=1;whiteSpace=wrap;html=0;"
	default:
		return "rounded=0;whiteSpace=wrap;html=0;"
	}
}

type attrReplacement struct {
	start int
	end   int
	value string
}

func setTagAttributes(tag []byte, updates map[string]string) ([]byte, error) {
	if len(tag) < 3 || tag[0] != '<' || tag[len(tag)-1] != '>' {
		return nil, fmt.Errorf("%w: malformed start tag", ErrInvalidXML)
	}
	limit := len(tag) - 1
	if limit > 0 && tag[limit-1] == '/' {
		limit--
	}
	position := 1
	for position < limit && !isXMLSpace(tag[position]) && tag[position] != '>' && tag[position] != '/' {
		position++
	}
	found := make(map[string]attrReplacement, len(updates))
	for position < limit {
		for position < limit && isXMLSpace(tag[position]) {
			position++
		}
		if position >= limit || tag[position] == '/' {
			break
		}
		nameStart := position
		for position < limit {
			value := tag[position]
			if isXMLSpace(value) || value == '=' || value == '/' || value == '>' {
				break
			}
			position++
		}
		if nameStart == position {
			break
		}
		name := string(tag[nameStart:position])
		for position < limit && isXMLSpace(tag[position]) {
			position++
		}
		if position >= limit || tag[position] != '=' {
			continue
		}
		position++
		for position < limit && isXMLSpace(tag[position]) {
			position++
		}
		if position >= limit || (tag[position] != '"' && tag[position] != '\'') {
			return nil, fmt.Errorf("%w: unquoted attribute %q", ErrInvalidXML, name)
		}
		quote := tag[position]
		position++
		valueStart := position
		for position < limit && tag[position] != quote {
			position++
		}
		if position >= limit {
			return nil, fmt.Errorf("%w: unterminated attribute %q", ErrInvalidXML, name)
		}
		if replacement, ok := updates[name]; ok {
			found[name] = attrReplacement{start: valueStart, end: position, value: escapeAttr(replacement)}
		}
		position++
	}

	replacements := make([]attrReplacement, 0, len(found))
	for _, replacement := range found {
		replacements = append(replacements, replacement)
	}
	sort.Slice(replacements, func(i, j int) bool { return replacements[i].start > replacements[j].start })
	result := append([]byte(nil), tag...)
	for _, replacement := range replacements {
		result = append(result[:replacement.start], append([]byte(replacement.value), result[replacement.end:]...)...)
	}

	missing := make([]string, 0)
	for name := range updates {
		if _, exists := found[name]; !exists {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) == 0 {
		return result, nil
	}
	insertAt := len(result) - 1
	if insertAt > 0 && result[insertAt-1] == '/' {
		insertAt--
	}
	var addition strings.Builder
	for _, name := range missing {
		addition.WriteByte(' ')
		addition.WriteString(name)
		addition.WriteString("=\"")
		addition.WriteString(escapeAttr(updates[name]))
		addition.WriteByte('"')
	}
	result = append(result[:insertAt], append([]byte(addition.String()), result[insertAt:]...)...)
	return result, nil
}

func isXMLSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\n' || value == '\r'
}

func escapeAttr(value string) string {
	var buffer bytes.Buffer
	_ = xml.EscapeText(&buffer, []byte(value))
	return buffer.String()
}

func validXMLCharacters(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if r == '\t' || r == '\n' || r == '\r' ||
			(r >= 0x20 && r <= 0xD7FF) ||
			(r >= 0xE000 && r <= 0xFFFD) ||
			(r >= 0x10000 && r <= 0x10FFFF) {
			continue
		}
		return false
	}
	return true
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
