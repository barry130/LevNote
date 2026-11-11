package drawio

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"net/url"
	"strconv"
	"strings"
)

// DefaultMaxBytes is the input byte limit used by zero-valued ParseOptions.
// Callers can inspect a buffer length before materializing it for Parse.
const DefaultMaxBytes = 32 << 20

const (
	hardMaxBytes                = 128 << 20
	defaultMaxDecompressedBytes = 64 << 20
	hardMaxDecompressedBytes    = 256 << 20
	defaultMaxXMLDepth          = 256
	defaultMaxElements          = 250_000
	defaultMaxAttributes        = 64
	defaultMaxTextBytes         = 16 << 20
)

func normalizedParseOptions(options ParseOptions) ParseOptions {
	if options.MaxBytes <= 0 || options.MaxBytes > hardMaxBytes {
		options.MaxBytes = DefaultMaxBytes
	}
	if options.MaxDecompressedBytes <= 0 || options.MaxDecompressedBytes > hardMaxDecompressedBytes {
		options.MaxDecompressedBytes = defaultMaxDecompressedBytes
	}
	if options.MaxXMLDepth <= 0 || options.MaxXMLDepth > 1024 {
		options.MaxXMLDepth = defaultMaxXMLDepth
	}
	if options.MaxElements <= 0 {
		options.MaxElements = defaultMaxElements
	}
	if options.MaxAttributes <= 0 {
		options.MaxAttributes = defaultMaxAttributes
	}
	if options.MaxTextBytes <= 0 {
		options.MaxTextBytes = defaultMaxTextBytes
	}
	return options
}

type xmlParser struct {
	decoder      *xml.Decoder
	source       []byte
	options      ParseOptions
	elements     int
	textBytes    int
	decompressed int
	pageIndex    int
	issues       []Issue
}

func Parse(data []byte, options ParseOptions) (*Document, []Issue, error) {
	options = normalizedParseOptions(options)
	if len(data) > options.MaxBytes {
		return nil, nil, fmt.Errorf("%w: input is larger than %d bytes", ErrLimitExceeded, options.MaxBytes)
	}
	return parseOwnedDocumentBytes(cloneBytes(data), options)
}

// parseOwnedDocumentBytes takes exclusive ownership of data. The private
// pristine and current-source fields may alias data, while the exported mutable
// Source field always receives a separate clone.
func parseOwnedDocumentBytes(data []byte, options ParseOptions) (*Document, []Issue, error) {
	if len(data) > options.MaxBytes {
		return nil, nil, fmt.Errorf("%w: input is larger than %d bytes", ErrLimitExceeded, options.MaxBytes)
	}
	parser := newXMLParser(data, options)
	token, start, end, err := parser.next()
	if err != nil {
		return nil, nil, err
	}
	for {
		switch typed := token.(type) {
		case xml.CharData:
			if len(bytes.TrimSpace(typed)) != 0 {
				return nil, parser.issues, fmt.Errorf("%w: text before the root element", ErrInvalidXML)
			}
		case xml.Comment, xml.ProcInst:
		case xml.Directive:
			return nil, parser.issues, fmt.Errorf("%w: XML directives and DTDs are not accepted", ErrInvalidXML)
		case xml.StartElement:
			goto root
		case nil:
			return nil, parser.issues, fmt.Errorf("%w: empty document", ErrInvalidXML)
		default:
			return nil, parser.issues, fmt.Errorf("%w: unexpected XML token", ErrInvalidXML)
		}
		token, start, end, err = parser.next()
		if err != nil {
			return nil, parser.issues, err
		}
	}

root:
	root := token.(xml.StartElement)
	if err := parser.acceptStart(root, 0); err != nil {
		return nil, parser.issues, err
	}
	document := &Document{
		original: data,
		source:   data,
		options:  options,
	}
	switch root.Name.Local {
	case "mxfile":
		document.IsMxFile = true
		if err := parser.parseMxFile(root, start, end, document); err != nil {
			return nil, parser.issues, err
		}
	case "mxGraphModel":
		model, _, rootClose, err := parser.parseModel(root, start, end, 1)
		if err != nil {
			return nil, parser.issues, err
		}
		page := Page{
			Index:    0,
			Model:    model,
			ReadOnly: rootClose < 0 || model.ReadOnlyReason != "",
			ReadOnlyReason: func() string {
				if model.ReadOnlyReason != "" {
					return model.ReadOnlyReason
				}
				if rootClose < 0 {
					return "mxGraphModel has no root cell container"
				}
				return ""
			}(),
			modelStart: start,
			rootClose:  rootClose,
		}
		document.Pages = append(document.Pages, page)
	default:
		return nil, parser.issues, fmt.Errorf("%w: root must be mxfile or mxGraphModel", ErrUnsupported)
	}
	if err := parser.finishRoot(); err != nil {
		return nil, parser.issues, err
	}
	if len(document.Pages) == 0 {
		return nil, parser.issues, ErrEmptyDocument
	}
	document.Source = cloneBytes(document.source)
	return document, parser.issues, nil
}

func newXMLParser(data []byte, options ParseOptions) *xmlParser {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = true
	return &xmlParser{
		decoder: decoder,
		source:  data,
		options: options,
		issues:  make([]Issue, 0),
	}
}

func (p *xmlParser) next() (xml.Token, int, int, error) {
	start := p.decoder.InputOffset()
	token, err := p.decoder.Token()
	end := p.decoder.InputOffset()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, int(start), int(end), io.EOF
		}
		return nil, int(start), int(end), fmt.Errorf("%w: %v", ErrInvalidXML, err)
	}
	return token, int(start), int(end), nil
}

func (p *xmlParser) acceptStart(start xml.StartElement, depth int) error {
	if depth > p.options.MaxXMLDepth {
		return fmt.Errorf("%w: XML depth exceeds %d", ErrLimitExceeded, p.options.MaxXMLDepth)
	}
	p.elements++
	if p.elements > p.options.MaxElements {
		return fmt.Errorf("%w: XML element count exceeds %d", ErrLimitExceeded, p.options.MaxElements)
	}
	if len(start.Attr) > p.options.MaxAttributes {
		return fmt.Errorf("%w: element %s has too many attributes", ErrLimitExceeded, start.Name.Local)
	}
	seen := make(map[xml.Name]struct{}, len(start.Attr))
	for _, attr := range start.Attr {
		if _, duplicate := seen[attr.Name]; duplicate {
			return fmt.Errorf("%w: duplicate attribute %q on %s", ErrInvalidXML, attr.Name.Local, start.Name.Local)
		}
		seen[attr.Name] = struct{}{}
	}
	return nil
}

func (p *xmlParser) accountText(text []byte) error {
	p.textBytes += len(text)
	if p.textBytes > p.options.MaxTextBytes {
		return fmt.Errorf("%w: XML text exceeds %d bytes", ErrLimitExceeded, p.options.MaxTextBytes)
	}
	return nil
}

func (p *xmlParser) finishRoot() error {
	for {
		token, _, _, err := p.next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		switch typed := token.(type) {
		case xml.CharData:
			if len(bytes.TrimSpace(typed)) != 0 {
				return fmt.Errorf("%w: text after the root element", ErrInvalidXML)
			}
		case xml.Comment, xml.ProcInst:
		case xml.Directive:
			return fmt.Errorf("%w: XML directives and DTDs are not accepted", ErrInvalidXML)
		case xml.StartElement:
			return fmt.Errorf("%w: multiple root elements", ErrInvalidXML)
		}
	}
}

func (p *xmlParser) parseMxFile(start xml.StartElement, startTag, endTag int, document *Document) error {
	_ = startTag
	selfClosing, err := p.selfClosing(startTag, endTag)
	if err != nil {
		return err
	}
	if selfClosing {
		if err := p.consumeEmptyEnd(start.Name.Local); err != nil {
			return err
		}
		return nil
	}
	for {
		token, tokenStart, tokenEnd, err := p.next()
		if err != nil {
			return err
		}
		switch typed := token.(type) {
		case xml.CharData:
			if err := p.accountText(typed); err != nil {
				return err
			}
			if len(bytes.TrimSpace(typed)) != 0 {
				p.addIssue(IssueUnsupportedElement, "non-whitespace text inside mxfile is preserved", "", DocumentPage)
			}
		case xml.Comment, xml.ProcInst:
		case xml.Directive:
			return fmt.Errorf("%w: XML directives and DTDs are not accepted", ErrInvalidXML)
		case xml.StartElement:
			if typed.Name.Local != "diagram" {
				p.addIssue(IssueUnsupportedElement, "unknown mxfile child <"+typed.Name.Local+"> is preserved", "", DocumentPage)
				if err := p.skipElement(typed, tokenStart, tokenEnd, 2); err != nil {
					return err
				}
				continue
			}
			if err := p.acceptStart(typed, 1); err != nil {
				return err
			}
			if err := p.parseDiagram(typed, tokenStart, tokenEnd, document); err != nil {
				return err
			}
		case xml.EndElement:
			if typed.Name.Local == start.Name.Local {
				p.pageIndex = len(document.Pages)
				return nil
			}
			return fmt.Errorf("%w: mismatched mxfile end element", ErrInvalidXML)
		}
	}
}

func (p *xmlParser) parseDiagram(start xml.StartElement, startTag, endTag int, document *Document) error {
	pageID := attrValue(start, "id")
	pageName := attrValue(start, "name")
	pageIndex := len(document.Pages)
	p.pageIndex = pageIndex
	selfClosing, err := p.selfClosing(startTag, endTag)
	if err != nil {
		return err
	}
	if selfClosing {
		if err := p.consumeEmptyEnd(start.Name.Local); err != nil {
			return err
		}
		document.Pages = append(document.Pages, Page{
			Index:          pageIndex,
			ID:             pageID,
			Name:           pageName,
			ReadOnly:       true,
			ReadOnlyReason: "empty diagram has no editable mxGraphModel",
		})
		p.addIssue(IssueEmptyPage, "empty diagram page is preserved read-only", "", pageIndex)
		return nil
	}

	var compressed strings.Builder
	hasModel := false
	for {
		token, tokenStart, tokenEnd, err := p.next()
		if err != nil {
			return err
		}
		switch typed := token.(type) {
		case xml.CharData:
			if err := p.accountText(typed); err != nil {
				return err
			}
			compressed.Write([]byte(typed))
		case xml.Comment, xml.ProcInst:
		case xml.Directive:
			return fmt.Errorf("%w: XML directives and DTDs are not accepted", ErrInvalidXML)
		case xml.StartElement:
			if typed.Name.Local == "mxGraphModel" && !hasModel {
				if err := p.acceptStart(typed, 2); err != nil {
					return err
				}
				model, _, rootClose, err := p.parseModel(typed, tokenStart, tokenEnd, 2)
				if err != nil {
					return err
				}
				document.Pages = append(document.Pages, Page{
					Index:      pageIndex,
					ID:         pageID,
					Name:       pageName,
					Model:      model,
					modelStart: tokenStart,
					rootClose:  rootClose,
					ReadOnly:   rootClose < 0 || model.ReadOnlyReason != "",
					ReadOnlyReason: func() string {
						if model.ReadOnlyReason != "" {
							return model.ReadOnlyReason
						}
						if rootClose < 0 {
							return "mxGraphModel has no root cell container"
						}
						return ""
					}(),
				})
				hasModel = true
				continue
			}
			p.addIssue(IssueUnsupportedElement, "unsupported <"+typed.Name.Local+"> inside diagram is preserved", "", pageIndex)
			if err := p.skipElement(typed, tokenStart, tokenEnd, 2); err != nil {
				return err
			}
		case xml.EndElement:
			if typed.Name.Local != start.Name.Local {
				return fmt.Errorf("%w: mismatched diagram end element", ErrInvalidXML)
			}
			if hasModel {
				if len(bytes.TrimSpace([]byte(compressed.String()))) != 0 {
					p.addIssue(IssueUnsupportedElement, "mixed compressed text and XML model is preserved without rewriting", "", pageIndex)
				}
				p.pageIndex = pageIndex + 1
				return nil
			}
			payload := strings.TrimSpace(compressed.String())
			if payload == "" {
				document.Pages = append(document.Pages, Page{
					Index:          pageIndex,
					ID:             pageID,
					Name:           pageName,
					ReadOnly:       true,
					ReadOnlyReason: "empty diagram has no editable mxGraphModel",
				})
				p.addIssue(IssueEmptyPage, "empty diagram page is preserved read-only", "", pageIndex)
				p.pageIndex = pageIndex + 1
				return nil
			}
			modelXML, err := p.decompress(payload)
			if err != nil {
				return err
			}
			if p.decompressed+len(modelXML) > p.options.MaxDecompressedBytes {
				return fmt.Errorf("%w: decompressed pages exceed %d bytes", ErrLimitExceeded, p.options.MaxDecompressedBytes)
			}
			p.decompressed += len(modelXML)
			model, issues, err := parseStandaloneModel(modelXML, p.options, pageIndex)
			if err != nil {
				return fmt.Errorf("compressed page %d: %w", pageIndex, err)
			}
			p.issues = append(p.issues, issues...)
			p.addIssue(IssueCompressedReadOnly, "compressed diagram is rendered read-only because its outer base64/deflate payload is preserved verbatim", "", pageIndex)
			document.Pages = append(document.Pages, Page{
				Index:          pageIndex,
				ID:             pageID,
				Name:           pageName,
				Compressed:     true,
				ReadOnly:       true,
				ReadOnlyReason: "compressed pages render read-only until draw.io re-compression can be proven lossless",
				Model:          model,
			})
			p.pageIndex = pageIndex + 1
			return nil
		}
	}
}

func parseStandaloneModel(data []byte, options ParseOptions, pageIndex int) (Model, []Issue, error) {
	parser := newXMLParser(data, options)
	parser.pageIndex = pageIndex
	token, start, end, err := parser.next()
	if err != nil {
		return Model{}, parser.issues, err
	}
	for {
		switch typed := token.(type) {
		case xml.CharData:
			if len(bytes.TrimSpace(typed)) != 0 {
				return Model{}, parser.issues, fmt.Errorf("%w: text before mxGraphModel", ErrInvalidXML)
			}
		case xml.Comment, xml.ProcInst:
		case xml.Directive:
			return Model{}, parser.issues, fmt.Errorf("%w: XML directives and DTDs are not accepted", ErrInvalidXML)
		case xml.StartElement:
			startElement := typed
			if startElement.Name.Local != "mxGraphModel" {
				return Model{}, parser.issues, fmt.Errorf("%w: compressed payload is not mxGraphModel", ErrUnsupported)
			}
			if err := parser.acceptStart(startElement, 0); err != nil {
				return Model{}, parser.issues, err
			}
			model, _, _, err := parser.parseModel(startElement, start, end, 1)
			if err != nil {
				return Model{}, parser.issues, err
			}
			if err := parser.finishRoot(); err != nil {
				return Model{}, parser.issues, err
			}
			return model, parser.issues, nil
		case nil:
			return Model{}, parser.issues, fmt.Errorf("%w: empty compressed payload", ErrInvalidXML)
		}
		token, start, end, err = parser.next()
		if err != nil {
			return Model{}, parser.issues, err
		}
	}
}

func (p *xmlParser) parseModel(start xml.StartElement, startTag, endTag, depth int) (Model, int, int, error) {
	model := Model{ID: attrValue(start, "id")}
	if selfClosing, err := p.selfClosing(startTag, endTag); err != nil {
		return model, endTag, -1, err
	} else if selfClosing {
		if err := p.consumeEmptyEnd(start.Name.Local); err != nil {
			return model, endTag, -1, err
		}
		return model, endTag, -1, nil
	}
	rootClose := -1
	modelEnd := -1
	for {
		token, tokenStart, tokenEnd, err := p.next()
		if err != nil {
			return model, modelEnd, rootClose, err
		}
		switch typed := token.(type) {
		case xml.CharData:
			if err := p.accountText(typed); err != nil {
				return model, modelEnd, rootClose, err
			}
			if len(bytes.TrimSpace(typed)) != 0 {
				p.addIssue(IssueUnsupportedElement, "text inside mxGraphModel is preserved", "", p.pageIndex)
			}
		case xml.Comment, xml.ProcInst:
		case xml.Directive:
			return model, modelEnd, rootClose, fmt.Errorf("%w: XML directives and DTDs are not accepted", ErrInvalidXML)
		case xml.StartElement:
			if typed.Name.Local == "root" {
				if rootClose >= 0 {
					p.addIssue(IssueUnsupportedElement, "multiple root cell containers are preserved read-only", "", p.pageIndex)
					model.ReadOnlyReason = "multiple root cell containers are preserved read-only"
					if err := p.skipElement(typed, tokenStart, tokenEnd, depth+1); err != nil {
						return model, modelEnd, rootClose, err
					}
					continue
				}
				closeOffset, cells, err := p.parseRootCells(typed, tokenStart, tokenEnd, depth+1)
				if err != nil {
					return model, modelEnd, rootClose, err
				}
				rootClose = closeOffset
				model.Nodes = append(model.Nodes, cells.nodes...)
				model.Edges = append(model.Edges, cells.edges...)
				continue
			}
			p.addIssue(IssueUnsupportedElement, "unknown mxGraphModel child <"+typed.Name.Local+"> is preserved", "", p.pageIndex)
			if err := p.skipElement(typed, tokenStart, tokenEnd, depth+1); err != nil {
				return model, modelEnd, rootClose, err
			}
		case xml.EndElement:
			if typed.Name.Local != start.Name.Local {
				return model, modelEnd, rootClose, fmt.Errorf("%w: mismatched mxGraphModel end element", ErrInvalidXML)
			}
			modelEnd = tokenStart
			model.Bounds = computeBounds(model.Nodes, model.Edges)
			p.validateEdgeEndpoints(model)
			return model, modelEnd, rootClose, nil
		}
	}
}

type parsedCells struct {
	nodes []Node
	edges []Edge
}

type parsedCellState struct {
	empty                  bool
	processingInstructions bool
	nonVertexIssueIndex    int
}

func (p *xmlParser) parseRootCells(start xml.StartElement, startTag, endTag, depth int) (int, parsedCells, error) {
	cells := parsedCells{}
	duplicates := make(map[string]struct{})
	rootClose := -1
	lastCellID := ""
	cellIndex := 0
	firstContainerIssue := -1
	selfClosing, err := p.selfClosing(startTag, endTag)
	if err != nil {
		return rootClose, cells, err
	}
	if selfClosing {
		if err := p.consumeEmptyEnd(start.Name.Local); err != nil {
			return rootClose, cells, err
		}
		return rootClose, cells, nil
	}
	for {
		token, tokenStart, tokenEnd, err := p.next()
		if err != nil {
			return rootClose, cells, err
		}
		switch typed := token.(type) {
		case xml.CharData:
			if err := p.accountText(typed); err != nil {
				return rootClose, cells, err
			}
			if len(bytes.TrimSpace(typed)) != 0 {
				p.addIssue(IssueUnsupportedElement, "text inside <root> is preserved", "", p.pageIndex)
			}
			if !isXMLWhitespace(typed) {
				firstContainerIssue = -1
			}
		case xml.Comment:
		case xml.ProcInst:
			firstContainerIssue = -1
		case xml.Directive:
			return rootClose, cells, fmt.Errorf("%w: XML directives and DTDs are not accepted", ErrInvalidXML)
		case xml.StartElement:
			if typed.Name.Local != "mxCell" {
				firstContainerIssue = -1
				p.addIssue(IssueUnsupportedElement, "cell wrapper <"+typed.Name.Local+"> is preserved but not rendered", "", p.pageIndex)
				if err := p.skipElement(typed, tokenStart, tokenEnd, depth+1); err != nil {
					return rootClose, cells, err
				}
				continue
			}
			if err := p.acceptStart(typed, depth+1); err != nil {
				return rootClose, cells, err
			}
			id := attrValue(typed, "id")
			if id == "" {
				return rootClose, cells, fmt.Errorf("%w: mxCell id is required", ErrInvalidXML)
			}
			if _, exists := duplicates[id]; exists {
				return rootClose, cells, fmt.Errorf("%w: %q", ErrDuplicateID, id)
			}
			duplicates[id] = struct{}{}
			lastCellID = id
			secondContainer := firstContainerIssue >= 0 && cellIndex == 1 && isCanonicalContainerStart(typed, false)
			state := parsedCellState{nonVertexIssueIndex: -1}
			node, edge, isNode, isEdge, err := p.parseCell(typed, tokenStart, tokenEnd, depth+1, &state)
			if err != nil {
				return rootClose, cells, err
			}
			if secondContainer && state.empty && !state.processingInstructions && state.nonVertexIssueIndex >= 0 {
				p.removeCanonicalContainerWarnings(firstContainerIssue, state.nonVertexIssueIndex)
			}
			firstContainerIssue = -1
			if cellIndex == 0 && isCanonicalContainerStart(typed, true) &&
				state.empty && !state.processingInstructions && state.nonVertexIssueIndex >= 0 {
				firstContainerIssue = state.nonVertexIssueIndex
			}
			cellIndex++
			if isNode {
				cells.nodes = append(cells.nodes, node)
			}
			if isEdge {
				cells.edges = append(cells.edges, edge)
			}
		case xml.EndElement:
			if typed.Name.Local != start.Name.Local {
				return rootClose, cells, fmt.Errorf("%w: mismatched root end element %q (wanted %q after cell %q)", ErrInvalidXML, typed.Name.Local, start.Name.Local, lastCellID)
			}
			rootClose = tokenStart
			return rootClose, cells, nil
		}
	}
}

func (p *xmlParser) parseCell(start xml.StartElement, startTag, endTag, depth int, state *parsedCellState) (Node, Edge, bool, bool, error) {
	id := attrValue(start, "id")
	parent := attrValue(start, "parent")
	text := attrValue(start, "value")
	style := attrValue(start, "style")
	vertex := isTruthy(attrValue(start, "vertex"))
	edgeCell := isTruthy(attrValue(start, "edge"))
	node := Node{
		ID:            id,
		Text:          text,
		PlainText:     plainLabel(text),
		Shape:         inferShape(style, vertex),
		Parent:        parent,
		Style:         style,
		startTagStart: startTag,
		startTagEnd:   endTag,
		endOffset:     endTag,
		geometryStart: -1,
		geometryEnd:   -1,
	}
	edge := Edge{
		ID:            id,
		Text:          text,
		PlainText:     plainLabel(text),
		SourceID:      attrValue(start, "source"),
		TargetID:      attrValue(start, "target"),
		Parent:        parent,
		Style:         style,
		startTagStart: startTag,
		startTagEnd:   endTag,
		endOffset:     endTag,
		geometryStart: -1,
		geometryEnd:   -1,
	}
	html := isTruthy(styleValue(style, "html")) || looksLikeHTML(text)
	if html {
		node.HTML = true
		edge.HTML = true
		message := "HTML label is preserved verbatim and rendered as escaped plain text"
		p.addIssue(IssueHTMLLabel, message, id, p.pageIndex)
		node.Unsupported = append(node.Unsupported, "html-label")
		edge.Unsupported = append(edge.Unsupported, "html-label")
	}
	if isImageStyle(style) {
		node.Image = true
		node.Unsupported = append(node.Unsupported, "image")
		p.addIssue(IssueImage, "image or remote image style is preserved; native SVG export does not fetch it", id, p.pageIndex)
	}
	if isStencilStyle(style) {
		node.Stencil = true
		node.Unsupported = append(node.Unsupported, "stencil")
		p.addIssue(IssueStencil, "custom stencil is preserved and rendered with a basic-shape fallback", id, p.pageIndex)
	}
	if !vertex && !edgeCell {
		state.nonVertexIssueIndex = len(p.issues)
		p.addIssue(IssueUnsupportedElement, "mxCell is neither a vertex nor an edge and is not rendered", id, p.pageIndex)
	}

	selfClosing, err := p.selfClosing(startTag, endTag)
	if err != nil {
		return Node{}, Edge{}, false, false, err
	}
	if selfClosing {
		state.empty = true
		if err := p.consumeEmptyEnd(start.Name.Local); err != nil {
			return Node{}, Edge{}, false, false, err
		}
	} else {
		state.empty = true
		for {
			token, tokenStart, tokenEnd, nextErr := p.next()
			if nextErr != nil {
				return Node{}, Edge{}, false, false, nextErr
			}
			switch typed := token.(type) {
			case xml.CharData:
				if err := p.accountText(typed); err != nil {
					return Node{}, Edge{}, false, false, err
				}
				if len(bytes.TrimSpace(typed)) != 0 {
					p.addIssue(IssueUnsupportedElement, "text inside mxCell is preserved", id, p.pageIndex)
				}
				if !isXMLWhitespace(typed) {
					state.empty = false
				}
			case xml.Comment:
			case xml.ProcInst:
				state.empty = false
				state.processingInstructions = true
			case xml.Directive:
				return Node{}, Edge{}, false, false, fmt.Errorf("%w: XML directives and DTDs are not accepted", ErrInvalidXML)
			case xml.StartElement:
				state.empty = false
				if typed.Name.Local == "mxGeometry" {
					if err := p.acceptStart(typed, depth+1); err != nil {
						return Node{}, Edge{}, false, false, err
					}
					geometry, err := p.parseGeometry(typed, tokenStart, tokenEnd, depth+1, id)
					if err != nil {
						return Node{}, Edge{}, false, false, err
					}
					node.X, node.Y = geometry.x, geometry.y
					node.Width, node.Height = geometry.width, geometry.height
					node.geometryStart, node.geometryEnd = tokenStart, tokenEnd
					edge.geometryStart, edge.geometryEnd = tokenStart, tokenEnd
					edge.SourcePoint = geometry.sourcePoint
					edge.TargetPoint = geometry.targetPoint
					edge.Points = append([]Point(nil), geometry.points...)
					continue
				}
				p.addIssue(IssueUnsupportedElement, "unknown child of mxCell is preserved", id, p.pageIndex)
				if err := p.skipElement(typed, tokenStart, tokenEnd, depth+1); err != nil {
					return Node{}, Edge{}, false, false, err
				}
			case xml.EndElement:
				if typed.Name.Local != start.Name.Local {
					return Node{}, Edge{}, false, false, fmt.Errorf("%w: mismatched mxCell end element", ErrInvalidXML)
				}
				node.endOffset = tokenStart
				edge.endOffset = tokenStart
				goto cellDone
			}
		}
	}

cellDone:
	if vertex && node.geometryStart < 0 {
		p.addIssue(IssueMissingGeometry, "vertex has no mxGeometry; its bounds are zero", id, p.pageIndex)
	}
	return node, edge, vertex, edgeCell, nil
}

func isCanonicalContainerStart(start xml.StartElement, first bool) bool {
	if start.Name.Local != "mxCell" || start.Name.Space != "" {
		return false
	}
	wantAttrs := 2
	if first {
		wantAttrs = 1
	}
	if len(start.Attr) != wantAttrs {
		return false
	}
	seenID, seenParent := false, false
	for _, attr := range start.Attr {
		if attr.Name.Space != "" {
			return false
		}
		switch attr.Name.Local {
		case "id":
			want := "1"
			if first {
				want = "0"
			}
			if attr.Value != want {
				return false
			}
			seenID = true
		case "parent":
			if first || attr.Value != "0" {
				return false
			}
			seenParent = true
		default:
			return false
		}
	}
	return seenID && (first || seenParent)
}

func isXMLWhitespace(data []byte) bool {
	for _, value := range data {
		switch value {
		case ' ', '\t', '\r', '\n':
		default:
			return false
		}
	}
	return true
}

func (p *xmlParser) removeCanonicalContainerWarnings(first, second int) {
	if first < 0 || second <= first || second >= len(p.issues) ||
		!isCanonicalContainerWarning(p.issues[first], "0") ||
		!isCanonicalContainerWarning(p.issues[second], "1") {
		return
	}
	oldLen := len(p.issues)
	p.issues = append(p.issues[:second], p.issues[second+1:]...)
	p.issues = append(p.issues[:first], p.issues[first+1:]...)
	clear(p.issues[len(p.issues):oldLen])
}

func isCanonicalContainerWarning(issue Issue, id string) bool {
	return issue.Code == IssueUnsupportedElement &&
		issue.CellID == id &&
		issue.Message == "mxCell is neither a vertex nor an edge and is not rendered"
}

type parsedGeometry struct {
	x, y, width, height float64
	sourcePoint         *Point
	targetPoint         *Point
	points              []Point
}

func (p *xmlParser) parseGeometry(start xml.StartElement, startTag, endTag, depth int, cellID string) (parsedGeometry, error) {
	geometry := parsedGeometry{}
	var err error
	if value := attrValue(start, "x"); value != "" {
		geometry.x, err = parseFinite(value)
		if err != nil {
			p.addIssue(IssueMissingGeometry, "invalid geometry x is treated as zero", cellID, p.pageIndex)
			geometry.x = 0
		}
	}
	if value := attrValue(start, "y"); value != "" {
		geometry.y, err = parseFinite(value)
		if err != nil {
			p.addIssue(IssueMissingGeometry, "invalid geometry y is treated as zero", cellID, p.pageIndex)
			geometry.y = 0
		}
	}
	if value := attrValue(start, "width"); value != "" {
		geometry.width, err = parseFinite(value)
		if err != nil {
			p.addIssue(IssueMissingGeometry, "invalid geometry width is treated as zero", cellID, p.pageIndex)
			geometry.width = 0
		}
	}
	if value := attrValue(start, "height"); value != "" {
		geometry.height, err = parseFinite(value)
		if err != nil {
			p.addIssue(IssueMissingGeometry, "invalid geometry height is treated as zero", cellID, p.pageIndex)
			geometry.height = 0
		}
	}
	selfClosing, err := p.selfClosing(startTag, endTag)
	if err != nil {
		return geometry, err
	}
	if selfClosing {
		if err := p.consumeEmptyEnd(start.Name.Local); err != nil {
			return geometry, err
		}
		return geometry, nil
	}
	for {
		token, tokenStart, tokenEnd, nextErr := p.next()
		if nextErr != nil {
			return geometry, nextErr
		}
		switch typed := token.(type) {
		case xml.CharData:
			if err := p.accountText(typed); err != nil {
				return geometry, err
			}
		case xml.Comment, xml.ProcInst:
		case xml.Directive:
			return geometry, fmt.Errorf("%w: XML directives and DTDs are not accepted", ErrInvalidXML)
		case xml.StartElement:
			if typed.Name.Local != "mxPoint" {
				p.addIssue(IssueUnsupportedElement, "unknown mxGeometry child is preserved", cellID, p.pageIndex)
				if err := p.skipElement(typed, tokenStart, tokenEnd, depth+1); err != nil {
					return geometry, err
				}
				continue
			}
			if err := p.acceptStart(typed, depth+1); err != nil {
				return geometry, err
			}
			point := Point{}
			if value := attrValue(typed, "x"); value != "" {
				point.X, _ = parseFinite(value)
			}
			if value := attrValue(typed, "y"); value != "" {
				point.Y, _ = parseFinite(value)
			}
			switch attrValue(typed, "as") {
			case "sourcePoint":
				value := point
				geometry.sourcePoint = &value
			case "targetPoint":
				value := point
				geometry.targetPoint = &value
			default:
				geometry.points = append(geometry.points, point)
			}
			if isSelfClosingRaw(p.source[tokenStart:tokenEnd]) {
				continue
			}
			if err := p.skipElementTail(typed, depth+1); err != nil {
				return geometry, err
			}
		case xml.EndElement:
			if typed.Name.Local != start.Name.Local {
				return geometry, fmt.Errorf("%w: mismatched mxGeometry end element", ErrInvalidXML)
			}
			return geometry, nil
		}
	}
}

func (p *xmlParser) skipElement(start xml.StartElement, startTag, endTag, depth int) error {
	if err := p.acceptStart(start, depth); err != nil {
		return err
	}
	if isSelfClosingRaw(p.source[startTag:endTag]) {
		if err := p.consumeEmptyEnd(start.Name.Local); err != nil {
			return err
		}
		return nil
	}
	return p.skipElementTail(start, depth)
}

func (p *xmlParser) skipElementTail(start xml.StartElement, depth int) error {
	// The caller has consumed and validated the start tag. mxCell children are
	// skipped here only for unknown wrappers, so attribute validation remains
	// best-effort at nested levels while depth and counts stay enforced.
	if depth > p.options.MaxXMLDepth {
		return fmt.Errorf("%w: XML depth exceeds %d", ErrLimitExceeded, p.options.MaxXMLDepth)
	}
	for {
		token, tokenStart, tokenEnd, err := p.next()
		if err != nil {
			return err
		}
		switch typed := token.(type) {
		case xml.StartElement:
			if err := p.skipElement(typed, tokenStart, tokenEnd, depth+1); err != nil {
				return err
			}
		case xml.CharData:
			if err := p.accountText(typed); err != nil {
				return err
			}
		case xml.Directive:
			return fmt.Errorf("%w: XML directives and DTDs are not accepted", ErrInvalidXML)
		case xml.EndElement:
			if typed.Name.Local == start.Name.Local {
				return nil
			}
			return fmt.Errorf("%w: mismatched skipped element", ErrInvalidXML)
		}
	}
}

func (p *xmlParser) selfClosing(startTag, endTag int) (bool, error) {
	if startTag < 0 || endTag < startTag || endTag > len(p.source) {
		return false, fmt.Errorf("%w: invalid element offsets", ErrInvalidXML)
	}
	return isSelfClosingRaw(p.source[startTag:endTag]), nil
}

func (p *xmlParser) consumeEmptyEnd(name string) error {
	token, _, _, err := p.next()
	if err != nil {
		return err
	}
	end, ok := token.(xml.EndElement)
	if !ok || end.Name.Local != name {
		return fmt.Errorf("%w: empty element %s has no matching end token", ErrInvalidXML, name)
	}
	return nil
}

func (p *xmlParser) decompress(payload string) ([]byte, error) {
	compact := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\n', '\r', '\t':
			return -1
		default:
			return r
		}
	}, payload)
	compressed, err := base64.StdEncoding.DecodeString(compact)
	if err != nil {
		compressed, err = base64.RawStdEncoding.DecodeString(compact)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid base64 payload: %v", ErrInvalidXML, err)
		}
	}
	if len(compressed) > p.options.MaxBytes {
		return nil, fmt.Errorf("%w: compressed payload exceeds %d bytes", ErrLimitExceeded, p.options.MaxBytes)
	}
	reader := flate.NewReader(bytes.NewReader(compressed))
	defer reader.Close()
	limit := int64(p.options.MaxDecompressedBytes) + 1
	decompressed, err := io.ReadAll(io.LimitReader(reader, limit))
	if err != nil {
		return nil, fmt.Errorf("%w: raw-deflate payload: %v", ErrInvalidXML, err)
	}
	if len(decompressed) > p.options.MaxDecompressedBytes {
		return nil, fmt.Errorf("%w: decompressed payload exceeds %d bytes", ErrLimitExceeded, p.options.MaxDecompressedBytes)
	}
	xmlData, err := url.PathUnescape(string(decompressed))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid URL-encoded compressed XML: %v", ErrInvalidXML, err)
	}
	if len(xmlData) > p.options.MaxDecompressedBytes {
		return nil, fmt.Errorf("%w: decompressed XML exceeds %d bytes", ErrLimitExceeded, p.options.MaxDecompressedBytes)
	}
	return []byte(xmlData), nil
}

func (p *xmlParser) addIssue(code IssueCode, message, cellID string, page int) {
	p.issues = append(p.issues, Issue{
		Severity: SeverityWarning,
		Code:     code,
		Message:  message,
		Page:     page,
		CellID:   cellID,
	})
}

func (p *xmlParser) validateEdgeEndpoints(model Model) {
	known := make(map[string]struct{}, len(model.Nodes)+len(model.Edges))
	for _, node := range model.Nodes {
		known[node.ID] = struct{}{}
	}
	for _, edge := range model.Edges {
		known[edge.ID] = struct{}{}
	}
	for _, edge := range model.Edges {
		if edge.SourceID != "" {
			if _, ok := known[edge.SourceID]; !ok {
				p.addIssue(IssueMissingEndpoint, "edge source cell is not present in this page", edge.ID, p.pageIndex)
			}
		}
		if edge.TargetID != "" {
			if _, ok := known[edge.TargetID]; !ok {
				p.addIssue(IssueMissingEndpoint, "edge target cell is not present in this page", edge.ID, p.pageIndex)
			}
		}
	}
}

func computeBounds(nodes []Node, edges []Edge) Rect {
	if len(nodes) == 0 && len(edges) == 0 {
		return Rect{}
	}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	include := func(x, y float64) {
		if math.IsNaN(x) || math.IsInf(x, 0) || math.IsNaN(y) || math.IsInf(y, 0) {
			return
		}
		minX = math.Min(minX, x)
		minY = math.Min(minY, y)
		maxX = math.Max(maxX, x)
		maxY = math.Max(maxY, y)
	}
	for _, node := range nodes {
		include(node.X, node.Y)
		include(node.X+node.Width, node.Y+node.Height)
	}
	for _, edge := range edges {
		if edge.SourcePoint != nil {
			include(edge.SourcePoint.X, edge.SourcePoint.Y)
		}
		if edge.TargetPoint != nil {
			include(edge.TargetPoint.X, edge.TargetPoint.Y)
		}
		for _, point := range edge.Points {
			include(point.X, point.Y)
		}
	}
	if math.IsInf(minX, 0) {
		return Rect{}
	}
	return Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}

func attrValue(start xml.StartElement, name string) string {
	for _, attr := range start.Attr {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func parseFinite(value string) (float64, error) {
	number, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, ErrInvalidGeometry
	}
	return number, nil
}

func styleValue(style, key string) string {
	for _, part := range strings.Split(style, ";") {
		name, value, ok := strings.Cut(part, "=")
		if ok && strings.TrimSpace(name) == key {
			return strings.TrimSpace(value)
		}
		if !ok && strings.TrimSpace(part) == key {
			return "1"
		}
	}
	return ""
}

func inferShape(style string, vertex bool) string {
	lower := strings.ToLower(style)
	shape := strings.ToLower(styleValue(style, "shape"))
	if strings.Contains(lower, "shape=image") || strings.Contains(lower, "image=") {
		return "image"
	}
	if strings.Contains(lower, "shape=stencil") || strings.Contains(lower, "shape=mxgraph.") {
		return "stencil"
	}
	if shape != "" && shape != "rectangle" && shape != "ellipse" && shape != "rhombus" && shape != "label" {
		return shape
	}
	if isTruthy(styleValue(style, "ellipse")) || shape == "ellipse" {
		return ShapeEllipse
	}
	if isTruthy(styleValue(style, "rhombus")) || shape == "rhombus" {
		return ShapeDiamond
	}
	if isTruthy(styleValue(style, "rounded")) {
		return ShapeRounded
	}
	if vertex {
		return ShapeRectangle
	}
	return ShapeConnector
}

func isImageStyle(style string) bool {
	lower := strings.ToLower(style)
	return strings.Contains(lower, "shape=image") ||
		strings.Contains(lower, "image=") ||
		strings.Contains(lower, "imagebackground=")
}

func isStencilStyle(style string) bool {
	lower := strings.ToLower(style)
	return strings.Contains(lower, "shape=stencil") ||
		strings.Contains(lower, "shape=mxgraph.") ||
		strings.Contains(lower, "shape=custom")
}

func looksLikeHTML(text string) bool {
	return strings.Contains(text, "<") && strings.Contains(text, ">")
}

func isSelfClosingRaw(raw []byte) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) >= 2 && trimmed[len(trimmed)-2] == '/' && trimmed[len(trimmed)-1] == '>'
}
