package graphview

import (
	"encoding/json"
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/formats/drawio"
	"levnote.local/native/formats/mindmap"
)

func (v *view) drawCanvas(p *ui.Painter, r ui.Rect) {
	v.frameNodes, v.frameLabels, v.frameLabelUses, v.frameHiddenLabels, v.frameEdges = 0, 0, 0, 0, 0
	v.canvasRect = r
	theme := p.Theme()
	p.Fill(r, theme.Background, 0)
	if !v.fitted {
		v.fitCanvas()
	}
	v.centerOutlineTarget()
	v.drawGrid(p, r, theme)
	p.Clip(r, 0, func() {
		switch v.kind {
		case kindMindmap:
			v.drawMindmap(p, r, theme)
		case kindDrawio:
			v.drawDiagram(p, r, theme)
		}
	})
	if v.frameHiddenLabels > 0 {
		p.Text(r.X+8, r.Y+r.H-20, "概览中部分标签已隐藏；放大或选中节点查看", 11, theme.TextMuted)
	}
	if v.syncFailed || v.mind == nil && v.diagram == nil {
		message := "内容暂时无法以原生画布显示"
		if len(v.warnings) > 0 {
			message = v.warnings[0]
		}
		uiLayout := ui.ShapeText(message, ui.Font{Family: theme.Font, Size: 13}, max32(r.W-40, 80))
		width, height := uiLayout.Size()
		p.TextLayout(uiLayout, r.X+(r.W-width)/2, r.Y+(r.H-height)/2, theme.TextMuted)
	}
}

func (v *view) drawGrid(p *ui.Painter, r ui.Rect, theme *ui.Theme) {
	spacing := 40 * v.zoom
	if spacing < 18 {
		return
	}
	startX := math.Mod(float64(v.panX), float64(spacing))
	startY := math.Mod(float64(v.panY), float64(spacing))
	line := theme.Border.Alpha(0.35)
	for x := float32(startX); x < r.W; x += spacing {
		if x >= 0 {
			p.Line(r.X+x, r.Y, r.X+x, r.Y+r.H, 1, line)
		}
	}
	for y := float32(startY); y < r.H; y += spacing {
		if y >= 0 {
			p.Line(r.X, r.Y+y, r.X+r.W, r.Y+y, 1, line)
		}
	}
}

func (v *view) drawMindmap(p *ui.Painter, r ui.Rect, theme *ui.Theme) {
	if v.mind == nil {
		return
	}
	for _, edge := range v.layout.Edges {
		color := theme.Border
		if edge.Color != "" {
			if parsed, ok := graphColor(edge.Color); ok {
				color = parsed
			}
		}
		fromX, fromY := v.worldToScreen(float32(edge.FromPoint.X), float32(edge.FromPoint.Y))
		toX, toY := v.worldToScreen(float32(edge.ToPoint.X), float32(edge.ToPoint.Y))
		from, to := [2]float32{r.X + fromX, r.Y + fromY}, [2]float32{r.X + toX, r.Y + toY}
		clippedFrom, clippedTo, visible := clipSegment(from, to, r)
		if !visible {
			continue
		}
		v.frameEdges++
		points := [2][2]float32{clippedFrom, clippedTo}
		p.StrokePolyline(points[:], max32(1.2, 1.5*v.zoom), color)
		if intersects(ui.Rect{X: to[0], Y: to[1]}, r) {
			drawArrow(p, from, to, color, v.zoom)
		}
	}
	for _, node := range v.layout.Nodes {
		box := ui.Rect{
			X: r.X + (float32(node.X)*v.zoom + v.panX),
			Y: r.Y + (float32(node.Y)*v.zoom + v.panY),
			W: float32(node.Width) * v.zoom,
			H: float32(node.Height) * v.zoom,
		}
		if !intersects(box, r) {
			continue
		}
		fill := theme.Surface
		stroke := theme.Border
		textColor := theme.Text
		if node.ID == v.mind.RootID() {
			fill = theme.Accent.Alpha(0.16)
			stroke = theme.Accent
		}
		if source := node.Node; source != nil {
			fill, stroke, textColor = mindmapNodeColors(source, fill, stroke, textColor)
		}
		if node.ID == v.selectedID {
			stroke = theme.Accent
		}
		v.drawNodeBox(p, node.ID, box, "rounded", fill, stroke, theme.Font, textColor, v.zoom, func() string { return boundedMindmapLabel(node.Node, node.Text) })
		if node.Collapsed {
			badge := ui.Rect{X: box.X + box.W - 13, Y: box.Y + box.H/2 - 7, W: 14, H: 14}
			p.Fill(badge, theme.Accent, 7)
			p.Text(badge.X+4, badge.Y+1, "+", 10, theme.Background)
		}
	}
}

func (v *view) drawDiagram(p *ui.Painter, r ui.Rect, theme *ui.Theme) {
	page := v.currentPage()
	if page == nil {
		return
	}
	v.ensureNodeIndex(page)
	for _, edge := range page.Model.Edges {
		points, ok := v.diagramEdgePoints(edge, r)
		if !ok || !polylineVisible(points, r) {
			continue
		}
		v.frameEdges++
		color := theme.TextMuted
		if value := graphStyleValue(edge.Style, "strokeColor"); value != "" {
			if parsed, ok := graphColor(value); ok {
				color = parsed
			}
		}
		p.StrokePolyline(points, max32(1.1, 1.5*v.zoom), color)
		drawArrow(p, points[len(points)-2], points[len(points)-1], color, v.zoom)
	}
	for _, node := range page.Model.Nodes {
		box := v.diagramNodeBox(node, r)
		if !intersects(box, r) {
			continue
		}
		fill := theme.SurfaceHover
		stroke := theme.Border
		textColor := theme.Text
		if value := graphStyleValue(node.Style, "fillColor"); value != "" {
			if parsed, ok := graphColor(value); ok {
				fill = parsed
			}
		}
		if value := graphStyleValue(node.Style, "strokeColor"); value != "" {
			if parsed, ok := graphColor(value); ok {
				stroke = parsed
			}
		}
		if value := graphStyleValue(node.Style, "fontColor"); value != "" {
			if parsed, ok := graphColor(value); ok {
				textColor = parsed
			}
		}
		if node.ID == v.selectedID {
			stroke = theme.Accent
		}
		v.drawNodeBox(p, node.ID, box, node.Shape, fill, stroke, theme.Font, textColor, v.zoom, func() string { return boundedLabel(node.PlainText) })
	}
}

func (v *view) drawNodeBox(
	p *ui.Painter,
	id string,
	box ui.Rect,
	shape string,
	fill, stroke ui.Color,
	fontFamily string,
	textColor ui.Color,
	zoom float32,
	text func() string,
) {
	v.frameNodes++
	strokeWidth := max32(1, 1.25*zoom)
	switch shape {
	case drawio.ShapeEllipse:
		path := ellipsePath(box)
		p.FillPath(path, fill)
		p.StrokePath(path, strokeWidth, stroke)
	case drawio.ShapeDiamond:
		path := &ui.Path{}
		path.MoveTo(box.X+box.W/2, box.Y).
			LineTo(box.X+box.W, box.Y+box.H/2).
			LineTo(box.X+box.W/2, box.Y+box.H).
			LineTo(box.X, box.Y+box.H/2).
			Close()
		p.FillPath(path, fill)
		p.StrokePath(path, strokeWidth, stroke)
	case drawio.ShapeRounded:
		radius := max32(4, min32(box.W, box.H)*0.16)
		p.Fill(box, fill, radius)
		p.Stroke(box, stroke, radius, strokeWidth)
	default:
		p.Fill(box, fill, 0)
		p.Stroke(box, stroke, 0, strokeWidth)
	}
	fontSize := max32(10, min32(18, 14*zoom))
	layout := v.labelLayout(id, fontFamily, fontSize, max32(box.W-16, 24), text)
	if layout == nil {
		return
	}
	width, height := layout.Size()
	p.Clip(box, 4, func() {
		p.TextLayout(layout, box.X+(box.W-width)/2, box.Y+(box.H-height)/2, textColor)
	})
}

func ellipsePath(box ui.Rect) *ui.Path {
	const kappa = float32(0.55228475)
	rx := box.W / 2
	ry := box.H / 2
	cx, cy := box.X+rx, box.Y+ry
	kx, ky := rx*kappa, ry*kappa
	path := &ui.Path{}
	path.MoveTo(cx, box.Y)
	path.CubeTo(cx+kx, box.Y, box.X+box.W, cy-ky, box.X+box.W, cy)
	path.CubeTo(box.X+box.W, cy+ky, cx+kx, box.Y+box.H, cx, box.Y+box.H)
	path.CubeTo(cx-kx, box.Y+box.H, box.X, cy+ky, box.X, cy)
	path.CubeTo(box.X, cy-ky, cx-kx, box.Y, cx, box.Y)
	path.Close()
	return path
}

func drawArrow(p *ui.Painter, from, to [2]float32, color ui.Color, zoom float32) {
	dx, dy := to[0]-from[0], to[1]-from[1]
	length := float32(math.Hypot(float64(dx), float64(dy)))
	if length < 0.01 {
		return
	}
	dx, dy = dx/length, dy/length
	size := max32(5, 7*zoom)
	backX, backY := to[0]-dx*size, to[1]-dy*size
	sideX, sideY := -dy*size*0.45, dx*size*0.45
	path := &ui.Path{}
	path.MoveTo(to[0], to[1]).
		LineTo(backX+sideX, backY+sideY).
		LineTo(backX-sideX, backY-sideY).
		Close()
	p.FillPath(path, color)
}

func (v *view) diagramEdgePoints(edge drawio.Edge, canvas ui.Rect) ([][2]float32, bool) {
	page := v.currentPage()
	var source, target ui.Rect
	sourceIndex, hasSource := v.nodeIndex[edge.SourceID]
	targetIndex, hasTarget := v.nodeIndex[edge.TargetID]
	if hasSource {
		source = v.diagramNodeBox(page.Model.Nodes[sourceIndex], canvas)
	}
	if hasTarget {
		target = v.diagramNodeBox(page.Model.Nodes[targetIndex], canvas)
	}
	points := append(v.edgeScratch[:0], [2]float32{}) // first endpoint filled below
	for _, point := range edge.Points {
		x, y := v.worldToScreen(float32(point.X), float32(point.Y))
		points = append(points, [2]float32{canvas.X + x, canvas.Y + y})
	}
	v.edgeScratch = points
	var start, end ui.Point
	startSet, endSet := false, false
	if hasSource {
		start = ui.Point{X: source.X + source.W/2, Y: source.Y + source.H/2}
		startSet = true
	} else if edge.SourcePoint != nil {
		x, y := v.worldToScreen(float32(edge.SourcePoint.X), float32(edge.SourcePoint.Y))
		start = ui.Point{X: canvas.X + x, Y: canvas.Y + y}
		startSet = true
	}
	if hasTarget {
		end = ui.Point{X: target.X + target.W/2, Y: target.Y + target.H/2}
		endSet = true
	} else if edge.TargetPoint != nil {
		x, y := v.worldToScreen(float32(edge.TargetPoint.X), float32(edge.TargetPoint.Y))
		end = ui.Point{X: canvas.X + x, Y: canvas.Y + y}
		endSet = true
	}
	if !startSet || !endSet {
		return nil, false
	}
	next := end
	if len(points) > 1 {
		next = ui.Point{X: points[1][0], Y: points[1][1]}
	}
	if hasSource {
		start = rectBoundary(source, next)
	}
	previous := start
	if len(points) > 1 {
		last := points[len(points)-1]
		previous = ui.Point{X: last[0], Y: last[1]}
	}
	if hasTarget {
		end = rectBoundary(target, previous)
	}
	points[0] = [2]float32{start.X, start.Y}
	points = append(points, [2]float32{end.X, end.Y})
	v.edgeScratch = points
	return points, true
}

func rectBoundary(box ui.Rect, toward ui.Point) ui.Point {
	cx, cy := box.X+box.W/2, box.Y+box.H/2
	dx, dy := toward.X-cx, toward.Y-cy
	if dx == 0 && dy == 0 {
		return ui.Point{X: cx, Y: cy}
	}
	scaleX, scaleY := float32(math.Inf(1)), float32(math.Inf(1))
	if dx != 0 {
		scaleX = (box.W / 2) / abs32(dx)
	}
	if dy != 0 {
		scaleY = (box.H / 2) / abs32(dy)
	}
	scale := min32(scaleX, scaleY)
	return ui.Point{X: cx + dx*scale, Y: cy + dy*scale}
}

func mindmapNodeColors(node *mindmap.Node, fill, stroke, text ui.Color) (ui.Color, ui.Color, ui.Color) {
	if node.Style != nil {
		if value, ok := objectString(*node.Style, "color"); ok {
			if parsed, ok := graphColor(value); ok {
				text = parsed
			}
		}
		if value, ok := objectString(*node.Style, "background-color"); ok {
			if parsed, ok := graphColor(value); ok {
				fill = parsed
			}
		}
		if value, ok := objectString(*node.Style, "border"); ok {
			if parsed, ok := graphColor(value); ok {
				stroke = parsed
			}
		}
	}
	return fill, stroke, text
}

func mindmapDisplayText(node *mindmap.Node, fallback string) string {
	if node == nil {
		return fallback
	}
	var parts []string
	if node.Icons != nil && len(*node.Icons) > 0 {
		parts = append(parts, strings.Join(*node.Icons, " "))
	}
	parts = append(parts, fallback)
	if node.Tags != nil && len(*node.Tags) > 0 {
		tags := make([]string, 0, len(*node.Tags))
		for _, tag := range *node.Tags {
			tags = append(tags, tag.Text)
		}
		parts = append(parts, "["+strings.Join(tags, " · ")+"]")
	}
	return strings.Join(parts, " ")
}

func objectString(object mindmap.Object, key string) (string, bool) {
	raw, ok := object.Get(key)
	if !ok {
		return "", false
	}
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return "", false
	}
	return value, true
}

func graphStyleValue(style, key string) string {
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
