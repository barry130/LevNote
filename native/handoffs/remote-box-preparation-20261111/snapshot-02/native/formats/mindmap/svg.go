package mindmap

import (
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// MaxSVGBytes is the default and hard SVG output limit. It does not bound
// document/layout memory or total process memory.
const MaxSVGBytes = 64 << 20

var ErrSVGOutputLimit = errors.New("mindmap SVG output exceeds byte limit")

// ExportSVG exports the supplied layout without changing the document. A zero
// maxBytes uses MaxSVGBytes; positive values are clamped to MaxSVGBytes, and
// negative values are refused. Any error returns nil, never a partial SVG.
// The shared emitter first counts within the budget, then fills an exact-size
// buffer. Callers must not mutate document/layout while the call is running.
func ExportSVG(document *Document, layout LayoutResult, maxBytes int) ([]byte, error) {
	if document == nil || document.Root == nil {
		return nil, errors.New("mindmap is not readable")
	}
	if maxBytes < 0 {
		return nil, ErrSVGOutputLimit
	}
	if maxBytes == 0 || maxBytes > MaxSVGBytes {
		maxBytes = MaxSVGBytes
	}
	count := svgWriter{limit: maxBytes}
	if err := emitSVG(&count, document, layout); err != nil {
		return nil, err
	}
	fill := svgWriter{limit: count.written, data: make([]byte, count.written)}
	if err := emitSVG(&fill, document, layout); err != nil {
		return nil, err
	}
	if fill.written != len(fill.data) {
		return nil, ErrInvalidDocument
	}
	return fill.data, nil
}

// A nil data slice selects count mode. Both passes use the same write checks;
// an over-budget fragment is refused before copying or increasing the count.
type svgWriter struct {
	data           []byte
	limit, written int
	err            error
}

func (w *svgWriter) reserve(size int) bool {
	if w.err != nil {
		return false
	}
	if size > w.limit-w.written {
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
	if w.err == nil {
		// Fragments contain only fixed syntax, numbers and admitted short colors.
		// User labels are escaped separately and bounded before reaching here.
		_, w.err = fmt.Fprintf(w, format, args...)
	}
}

func emitSVG(svg *svgWriter, document *Document, layout LayoutResult) error {
	bounds := layout.Bounds
	margin := 24.0
	if bounds.Width <= 0 {
		bounds.Width = 320
	}
	if bounds.Height <= 0 {
		bounds.Height = 200
	}
	minX, minY := bounds.X-margin, bounds.Y-margin
	width, height := bounds.Width+2*margin, bounds.Height+2*margin
	svg.printf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="%s %s %s %s" width="%s" height="%s">`,
		svgNumber(minX), svgNumber(minY), svgNumber(width), svgNumber(height), svgNumber(width), svgNumber(height))
	svg.text(`<defs><marker id="mm-arrow" markerWidth="8" markerHeight="6" refX="7" refY="3" orient="auto"><path d="M0,0 L8,3 L0,6 z" fill="context-stroke"/></marker></defs>`)
	svg.printf(`<rect x="%s" y="%s" width="%s" height="%s" fill="#ffffff"/>`,
		svgNumber(minX), svgNumber(minY), svgNumber(width), svgNumber(height))
	for _, edge := range layout.Edges {
		if svg.err != nil {
			return svg.err
		}
		stroke := svgColor(edge.Color, "#78909c")
		svg.printf(`<path d="M%s %s L%s %s" fill="none" stroke="%s" stroke-width="2" marker-end="url(#mm-arrow)"/>`,
			svgNumber(edge.FromPoint.X), svgNumber(edge.FromPoint.Y), svgNumber(edge.ToPoint.X), svgNumber(edge.ToPoint.Y), stroke)
	}
	for _, node := range layout.Nodes {
		if svg.err != nil {
			return svg.err
		}
		fill, stroke, text := "#eef5f3", "#5f8178", "#17211e"
		if node.ID == document.RootID() {
			fill, stroke = "#d7ebe5", "#33766c"
		}
		fill, stroke, text = svgColors(node.Node, fill, stroke, text)
		radius := math.Min(node.Width, node.Height) * 0.16
		// Colors have already become a literal #hex value (or a fixed fallback),
		// so XML attribute escaping would leave these exact bytes unchanged.
		svg.printf(`<rect x="%s" y="%s" width="%s" height="%s" rx="%s" ry="%s" fill="%s" stroke="%s" stroke-width="1.5"/>`,
			svgNumber(node.X), svgNumber(node.Y), svgNumber(node.Width), svgNumber(node.Height),
			svgNumber(radius), svgNumber(radius), fill, stroke)
		if svg.err != nil {
			return svg.err
		}
		writeSVGMultilineText(svg, node.X+node.Width/2, node.Y+node.Height/2, DisplayLabel(node.Node, node.Text), text, 14, true)
		if svg.err != nil {
			return svg.err
		}
		if node.Collapsed {
			cx, cy := node.X+node.Width-7, node.Y+node.Height/2
			svg.printf(`<circle cx="%s" cy="%s" r="7" fill="#33766c"/><text x="%s" y="%s" text-anchor="middle" fill="#fff" font-size="11">+</text>`,
				svgNumber(cx), svgNumber(cy), svgNumber(cx), svgNumber(cy+4))
		}
		if node.Node != nil && node.Node.Image != nil {
			svg.printf(`<text x="%s" y="%s" text-anchor="end" font-size="10" fill="#9b6b00">图</text>`,
				svgNumber(node.X+node.Width-8), svgNumber(node.Y+14))
		}
	}
	svg.text("</svg>\n")
	return svg.err
}

func svgColor(value, fallback string) string {
	if literal, ok := LiteralColor(value); ok {
		return literal
	}
	return fallback
}

func svgColors(node *Node, fill, stroke, text string) (string, string, string) {
	if node == nil || node.Style == nil {
		return fill, stroke, text
	}
	if value, ok := presentationObjectString(*node.Style, "background-color"); ok {
		fill = svgColor(value, fill)
	}
	if value, ok := presentationObjectString(*node.Style, "color"); ok {
		text = svgColor(value, text)
	}
	if value, ok := presentationObjectString(*node.Style, "border"); ok {
		stroke = svgColor(value, stroke)
	}
	return fill, stroke, text
}

func writeSVGMultilineText(dst *svgWriter, x, y float64, value, color string, size float64, centered bool) {
	if dst.err != nil {
		return
	}
	lines := splitLines(value)
	if len(lines) == 0 {
		return
	}
	anchor := "start"
	if centered {
		anchor = "middle"
	}
	lineHeight := size * 1.25
	startY := y - float64(len(lines)-1)*lineHeight/2 + size*0.35
	dst.printf(`<text x="%s" y="%s" text-anchor="%s" fill="%s" font-family="Arial, sans-serif" font-size="%s">`,
		svgNumber(x), svgNumber(startY), anchor, color, svgNumber(size))
	for index, line := range lines {
		if dst.err != nil {
			return
		}
		dy := 0.0
		if index > 0 {
			dy = lineHeight
		}
		dst.printf(`<tspan x="%s" dy="%s">`, svgNumber(x), svgNumber(dy))
		if dst.err != nil {
			return
		}
		dst.err = xml.EscapeText(dst, []byte(line))
		dst.text(`</tspan>`)
	}
	dst.text(`</text>`)
}

func splitLines(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, "\n")
}

func svgNumber(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}
