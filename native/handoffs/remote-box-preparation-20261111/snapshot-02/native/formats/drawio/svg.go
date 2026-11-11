package drawio

import (
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

const (
	MaxSVGBytes        = 64 << 20
	svgEscapeChunkSize = 4096
)

var errSVGFillMismatch = errors.New("draw.io SVG fill length does not match count")

func (d *Document) ExportSVG(pageIndex int, options SVGOptions) ([]byte, []Issue, error) {
	page, err := d.Page(pageIndex)
	if err != nil {
		return nil, nil, err
	}
	limit, err := resolveSVGMaxBytes(options.MaxBytes)
	if err != nil {
		return nil, nil, err
	}
	options = normalizeSVGOptions(options)
	render := newSVGRender(page, pageIndex, options)

	count := svgWriter{limit: limit}
	issues, err := emitSVG(&count, render, true)
	if err != nil {
		return nil, nil, err
	}

	fill := svgWriter{limit: count.written, data: make([]byte, count.written)}
	if _, err := emitSVG(&fill, render, false); err != nil {
		return nil, nil, err
	}
	if fill.written != len(fill.data) {
		return nil, nil, errSVGFillMismatch
	}
	return fill.data, issues, nil
}

type svgRender struct {
	pageIndex int
	model     *Model
	byID      map[string]Node
	bounds    Rect
	options   SVGOptions
}

func newSVGRender(page *Page, pageIndex int, options SVGOptions) *svgRender {
	model := &page.Model
	byID := make(map[string]Node, len(model.Nodes))
	for _, node := range model.Nodes {
		if finite(node.X) && finite(node.Y) && finite(node.Width) && finite(node.Height) {
			byID[node.ID] = node
		}
	}
	bounds := computeBounds(model.Nodes, model.Edges)
	if bounds.Width <= 0 {
		bounds.Width = 320
	}
	if bounds.Height <= 0 {
		bounds.Height = 200
	}
	return &svgRender{
		pageIndex: pageIndex,
		model:     model,
		byID:      byID,
		bounds:    bounds,
		options:   options,
	}
}

func emitSVG(svg *svgWriter, render *svgRender, collectIssues bool) ([]Issue, error) {
	if svg.err != nil {
		return nil, svg.err
	}
	options := render.options
	bounds := render.bounds
	minX := bounds.X - options.Padding
	minY := bounds.Y - options.Padding
	width := bounds.Width + 2*options.Padding
	height := bounds.Height + 2*options.Padding
	if width <= 0 {
		width = 320
	}
	if height <= 0 {
		height = 200
	}

	svg.printf("<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"%s %s %s %s\" width=\"%s\" height=\"%s\">",
		formatFloat(minX), formatFloat(minY), formatFloat(width), formatFloat(height), formatFloat(width), formatFloat(height))
	svg.text("<defs><marker id=\"levnote-arrow\" markerWidth=\"10\" markerHeight=\"7\" refX=\"9\" refY=\"3.5\" orient=\"auto\" markerUnits=\"strokeWidth\">")
	svg.text("<path d=\"M0,0 L10,3.5 L0,7 z\" fill=\"context-stroke\"/></marker></defs>")
	svg.printf("<rect x=\"%s\" y=\"%s\" width=\"%s\" height=\"%s\" fill=\"",
		formatFloat(minX), formatFloat(minY), formatFloat(width), formatFloat(height))
	writeEscapedXML(svg, options.Background)
	svg.text("\"/><g font-family=\"")
	writeEscapedXML(svg, options.FontFamily)
	svg.printf("\" font-size=\"%s\">", formatFloat(options.FontSize))
	if svg.err != nil {
		return nil, svg.err
	}

	warnings := make([]Issue, 0)
	var warningSink *[]Issue
	if collectIssues {
		warningSink = &warnings
	}
	for _, edge := range render.model.Edges {
		if svg.err != nil {
			return nil, svg.err
		}
		points, ok := edgePathPoints(edge, render.byID)
		if !ok {
			addSVGWarning(warningSink, Issue{
				Severity: SeverityWarning,
				Code:     IssueMissingEndpoint,
				Message:  "edge endpoint is unavailable; native SVG omitted the edge",
				Page:     render.pageIndex,
				CellID:   edge.ID,
			})
			continue
		}
		stroke := safeColor(styleValue(edge.Style, "strokeColor"), options.DefaultStroke)
		svg.text("<path d=\"")
		writeSVGPath(svg, points)
		if svg.err != nil {
			return nil, svg.err
		}
		svg.text("\" fill=\"none\" stroke=\"")
		writeEscapedXML(svg, stroke)
		svg.printf("\" stroke-width=\"%s\" marker-end=\"url(#levnote-arrow)\"/>", formatFloat(options.StrokeWidth))
		if svg.err != nil {
			return nil, svg.err
		}
		if edge.PlainText != "" || edge.Text != "" {
			mid := pathMidpoint(edge, render.byID)
			writeSVGText(svg, mid.X, mid.Y-3, edge.PlainText, options.FontSize, "#111111", true)
			if svg.err != nil {
				return nil, svg.err
			}
		}
		if edge.HTML {
			addSVGWarning(warningSink, Issue{
				Severity: SeverityWarning,
				Code:     IssueHTMLLabel,
				Message:  "HTML edge label was rendered as escaped plain text",
				Page:     render.pageIndex,
				CellID:   edge.ID,
			})
		}
	}

	for _, node := range render.model.Nodes {
		if svg.err != nil {
			return nil, svg.err
		}
		fill := safeColor(styleValue(node.Style, "fillColor"), options.DefaultFill)
		stroke := safeColor(styleValue(node.Style, "strokeColor"), options.DefaultStroke)
		shape := node.Shape
		if node.Image {
			shape = ShapeRounded
			addSVGWarning(warningSink, Issue{
				Severity: SeverityWarning,
				Code:     IssueImage,
				Message:  "image was not loaded; native SVG drew a labeled placeholder",
				Page:     render.pageIndex,
				CellID:   node.ID,
			})
		}
		if node.Stencil || (shape != ShapeRectangle && shape != ShapeEllipse && shape != ShapeDiamond && shape != ShapeRounded) {
			shape = ShapeRounded
			if node.Stencil {
				addSVGWarning(warningSink, Issue{
					Severity: SeverityWarning,
					Code:     IssueStencil,
					Message:  "custom stencil was rendered with a basic-shape fallback",
					Page:     render.pageIndex,
					CellID:   node.ID,
				})
			} else {
				addSVGWarning(warningSink, Issue{
					Severity: SeverityWarning,
					Code:     IssueUnsupportedElement,
					Message:  "unsupported shape was rendered with a rounded-rectangle fallback",
					Page:     render.pageIndex,
					CellID:   node.ID,
				})
			}
		}
		writeShape(svg, node, shape, fill, stroke, options.StrokeWidth)
		if svg.err != nil {
			return nil, svg.err
		}
		if node.Text != "" {
			writeSVGText(svg, node.X+node.Width/2, node.Y+node.Height/2, node.PlainText, options.FontSize, "#111111", true)
			if svg.err != nil {
				return nil, svg.err
			}
		}
		if node.HTML {
			addSVGWarning(warningSink, Issue{
				Severity: SeverityWarning,
				Code:     IssueHTMLLabel,
				Message:  "HTML label was rendered as escaped plain text",
				Page:     render.pageIndex,
				CellID:   node.ID,
			})
		}
	}
	if svg.err != nil {
		return nil, svg.err
	}
	svg.text("</g></svg>\n")
	if svg.err != nil {
		return nil, svg.err
	}
	return warnings, svg.err
}

func addSVGWarning(sink *[]Issue, issue Issue) {
	if sink != nil {
		*sink = append(*sink, issue)
	}
}

func normalizeSVGOptions(options SVGOptions) SVGOptions {
	if options.Padding <= 0 {
		options.Padding = 24
	}
	if options.Background == "" {
		options.Background = "#ffffff"
	}
	if options.DefaultFill == "" {
		options.DefaultFill = "#dae8fc"
	}
	if options.DefaultStroke == "" {
		options.DefaultStroke = "#6c8ebf"
	}
	if options.StrokeWidth <= 0 {
		options.StrokeWidth = 1.5
	}
	if options.FontFamily == "" {
		options.FontFamily = "Arial, sans-serif"
	}
	if options.FontSize <= 0 {
		options.FontSize = 14
	}
	return options
}

func resolveSVGMaxBytes(maxBytes int) (int, error) {
	if maxBytes < 0 {
		return 0, ErrSVGOutputLimit
	}
	if maxBytes == 0 || maxBytes > MaxSVGBytes {
		return MaxSVGBytes, nil
	}
	return maxBytes, nil
}

type svgWriter struct {
	data           []byte
	limit, written int
	err            error
}

func (w *svgWriter) reserve(size int) bool {
	if w.err != nil {
		return false
	}
	if size < 0 || w.written > w.limit || size > w.limit-w.written {
		w.err = ErrSVGOutputLimit
		return false
	}
	return true
}

func (w *svgWriter) Write(value []byte) (int, error) {
	if !w.reserve(len(value)) {
		return 0, w.err
	}
	if w.data != nil {
		copy(w.data[w.written:], value)
	}
	w.written += len(value)
	return len(value), nil
}

func (w *svgWriter) text(value string) {
	if !w.reserve(len(value)) {
		return
	}
	if w.data != nil {
		copy(w.data[w.written:], value)
	}
	w.written += len(value)
}

func (w *svgWriter) printf(format string, args ...any) {
	if w.err != nil {
		return
	}
	if _, err := fmt.Fprintf(w, format, args...); err != nil && w.err == nil {
		w.err = err
	}
}

func writeEscapedXML(dst *svgWriter, value string) {
	writeEscapedXMLChunks(dst, value, svgEscapeChunkSize)
}

func writeEscapedXMLChunks(dst *svgWriter, value string, maxChunk int) {
	for value != "" && dst.err == nil {
		size := svgEscapeChunkLength(value, maxChunk)
		if err := xml.EscapeText(dst, []byte(value[:size])); err != nil {
			if dst.err == nil {
				dst.err = err
			}
			return
		}
		value = value[size:]
	}
}

func svgEscapeChunkLength(value string, maxChunk int) int {
	if maxChunk < utf8.UTFMax {
		maxChunk = utf8.UTFMax
	}
	if len(value) <= maxChunk {
		return len(value)
	}
	end := 0
	for end < len(value) && end < maxChunk {
		_, size := utf8.DecodeRuneInString(value[end:])
		if size <= 0 || end+size > maxChunk {
			break
		}
		end += size
	}
	if end > 0 {
		return end
	}
	_, size := utf8.DecodeRuneInString(value)
	if size <= 0 {
		return 1
	}
	return size
}

func writeShape(dst *svgWriter, node Node, shape, fill, stroke string, strokeWidth float64) {
	if dst.err != nil {
		return
	}
	switch shape {
	case ShapeEllipse:
		dst.printf("<ellipse cx=\"%s\" cy=\"%s\" rx=\"%s\" ry=\"%s\" fill=\"",
			formatFloat(node.X+node.Width/2), formatFloat(node.Y+node.Height/2),
			formatFloat(node.Width/2), formatFloat(node.Height/2))
		writeEscapedXML(dst, fill)
		dst.text("\" stroke=\"")
		writeEscapedXML(dst, stroke)
		dst.printf("\" stroke-width=\"%s\"/>", formatFloat(strokeWidth))
	case ShapeDiamond:
		dst.printf("<polygon points=\"%s,%s %s,%s %s,%s %s,%s\" fill=\"",
			formatFloat(node.X+node.Width/2), formatFloat(node.Y),
			formatFloat(node.X+node.Width), formatFloat(node.Y+node.Height/2),
			formatFloat(node.X+node.Width/2), formatFloat(node.Y+node.Height),
			formatFloat(node.X), formatFloat(node.Y+node.Height/2))
		writeEscapedXML(dst, fill)
		dst.text("\" stroke=\"")
		writeEscapedXML(dst, stroke)
		dst.printf("\" stroke-width=\"%s\"/>", formatFloat(strokeWidth))
	case ShapeRounded:
		radius := math.Min(node.Width, node.Height) * 0.15
		dst.printf("<rect x=\"%s\" y=\"%s\" width=\"%s\" height=\"%s\" rx=\"%s\" ry=\"%s\" fill=\"",
			formatFloat(node.X), formatFloat(node.Y), formatFloat(node.Width), formatFloat(node.Height),
			formatFloat(radius), formatFloat(radius))
		writeEscapedXML(dst, fill)
		dst.text("\" stroke=\"")
		writeEscapedXML(dst, stroke)
		dst.printf("\" stroke-width=\"%s\"/>", formatFloat(strokeWidth))
	default:
		dst.printf("<rect x=\"%s\" y=\"%s\" width=\"%s\" height=\"%s\" fill=\"",
			formatFloat(node.X), formatFloat(node.Y), formatFloat(node.Width), formatFloat(node.Height),
		)
		writeEscapedXML(dst, fill)
		dst.text("\" stroke=\"")
		writeEscapedXML(dst, stroke)
		dst.printf("\" stroke-width=\"%s\"/>", formatFloat(strokeWidth))
	}
}

func writeSVGText(dst *svgWriter, x, y float64, value string, fontSize float64, color string, centered bool) {
	if dst.err != nil || value == "" {
		return
	}
	lineCount := 1 + strings.Count(value, "\n")
	lineHeight := fontSize * 1.25
	startY := y - float64(lineCount-1)*lineHeight/2 + fontSize*0.35
	anchor := "start"
	if centered {
		anchor = "middle"
	}
	dst.printf("<text x=\"%s\" y=\"%s\" text-anchor=\"%s\" fill=\"",
		formatFloat(x), formatFloat(startY), anchor)
	writeEscapedXML(dst, color)
	dst.text("\">")
	lineStart := 0
	for index := 0; index < lineCount && dst.err == nil; index++ {
		dy := "0"
		if index > 0 {
			dy = formatFloat(lineHeight)
		}
		lineEnd := len(value)
		if relative := strings.IndexByte(value[lineStart:], '\n'); relative >= 0 {
			lineEnd = lineStart + relative
		}
		dst.printf("<tspan x=\"%s\" dy=\"%s\">", formatFloat(x), dy)
		writeEscapedXML(dst, value[lineStart:lineEnd])
		dst.text("</tspan>")
		lineStart = lineEnd + 1
	}
	dst.text("</text>")
}

func writeSVGPath(dst *svgWriter, points []Point) {
	for index, point := range points {
		if dst.err != nil {
			return
		}
		if index == 0 {
			dst.text("M")
		} else {
			dst.text(" L")
		}
		dst.printf("%s %s", formatFloat(point.X), formatFloat(point.Y))
	}
}

func edgePathPoints(edge Edge, byID map[string]Node) ([]Point, bool) {
	var start, end *Point
	if edge.SourcePoint != nil {
		value := *edge.SourcePoint
		start = &value
	}
	if edge.TargetPoint != nil {
		value := *edge.TargetPoint
		end = &value
	}
	sourceNode, hasSource := byID[edge.SourceID]
	targetNode, hasTarget := byID[edge.TargetID]
	if start == nil && hasSource {
		value := Point{X: sourceNode.X + sourceNode.Width/2, Y: sourceNode.Y + sourceNode.Height/2}
		start = &value
	}
	if end == nil && hasTarget {
		value := Point{X: targetNode.X + targetNode.Width/2, Y: targetNode.Y + targetNode.Height/2}
		end = &value
	}
	if start == nil || end == nil {
		return nil, false
	}
	points := make([]Point, 0, len(edge.Points)+2)
	next := *end
	if len(edge.Points) > 0 {
		next = edge.Points[0]
	}
	startValue := *start
	if hasSource {
		startValue = boxBoundary(sourceNode, next)
	}
	points = append(points, startValue)
	points = append(points, edge.Points...)
	previous := startValue
	if len(edge.Points) > 0 {
		previous = edge.Points[len(edge.Points)-1]
	}
	endValue := *end
	if hasTarget {
		endValue = boxBoundary(targetNode, previous)
	}
	points = append(points, endValue)
	return points, true
}

func boxBoundary(node Node, toward Point) Point {
	cx := node.X + node.Width/2
	cy := node.Y + node.Height/2
	dx := toward.X - cx
	dy := toward.Y - cy
	if dx == 0 && dy == 0 {
		return Point{X: cx, Y: cy}
	}
	scaleX := math.Inf(1)
	scaleY := math.Inf(1)
	if dx != 0 {
		scaleX = (node.Width / 2) / math.Abs(dx)
	}
	if dy != 0 {
		scaleY = (node.Height / 2) / math.Abs(dy)
	}
	scale := math.Min(scaleX, scaleY)
	return Point{X: cx + dx*scale, Y: cy + dy*scale}
}

func pathMidpoint(edge Edge, byID map[string]Node) Point {
	start, end := Point{}, Point{}
	if edge.SourcePoint != nil {
		start = *edge.SourcePoint
	} else if node, ok := byID[edge.SourceID]; ok {
		start = Point{X: node.X + node.Width/2, Y: node.Y + node.Height/2}
	}
	if edge.TargetPoint != nil {
		end = *edge.TargetPoint
	} else if node, ok := byID[edge.TargetID]; ok {
		end = Point{X: node.X + node.Width/2, Y: node.Y + node.Height/2}
	}
	return Point{X: (start.X + end.X) / 2, Y: (start.Y + end.Y) / 2}
}

func safeColor(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "default" {
		return fallback
	}
	if strings.HasPrefix(value, "#") {
		if len(value) == 4 || len(value) == 5 || len(value) == 7 || len(value) == 9 {
			for _, char := range value[1:] {
				if !isHex(char) {
					return fallback
				}
			}
			return value
		}
		return fallback
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && char != '-' {
			return fallback
		}
	}
	return value
}

func isHex(value rune) bool {
	return value >= '0' && value <= '9' || value >= 'a' && value <= 'f' || value >= 'A' && value <= 'F'
}

func plainLabel(value string) string {
	if !looksLikeHTML(value) {
		return value
	}
	var output strings.Builder
	for index := 0; index < len(value); {
		if value[index] != '<' {
			output.WriteByte(value[index])
			index++
			continue
		}
		end := strings.IndexByte(value[index:], '>')
		if end < 0 {
			output.WriteString(value[index:])
			break
		}
		tag := strings.ToLower(strings.TrimSpace(value[index+1 : index+end]))
		if strings.HasPrefix(tag, "br") || strings.HasPrefix(tag, "/p") || strings.HasPrefix(tag, "/div") {
			if output.Len() > 0 && !strings.HasSuffix(output.String(), "\n") {
				output.WriteByte('\n')
			}
		}
		index += end + 1
	}
	return strings.TrimSpace(output.String())
}
