package graphview

import (
	"math"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/enginescene"
)

func sceneFinitePan(value float64) float32 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return float32(max(-8e9, min(8e9, value)))
}

func sceneDimension(value float64) float32 {
	// A legal positive subnormal can round to float32 zero. Apply a floor
	// before conversion; otherwise fitting can produce Inf or divide by zero.
	return float32(max(1e-4, min(float64(enginescene.MaxDimension), value)))
}

func (v *sceneView) fitScene() {
	if v.canvasRect.W <= 1 || v.canvasRect.H <= 1 || len(v.scene.Nodes) == 0 {
		return
	}
	b := v.scene.Bounds
	w, h := sceneDimension(b.Width), sceneDimension(b.Height)
	v.zoom = clamp32(min32((v.canvasRect.W-64)/w, (v.canvasRect.H-64)/h), 0.18, 2.4)
	v.panX = sceneFinitePan(float64(v.canvasRect.W-w*v.zoom)/2 - b.X*float64(v.zoom))
	v.panY = sceneFinitePan(float64(v.canvasRect.H-h*v.zoom)/2 - b.Y*float64(v.zoom))
	v.fitted = true
}

func (v *sceneView) zoomAt(x, y, factor float32) {
	if !finiteSceneInput(x, y, factor) || factor <= 0 {
		return
	}
	old := clamp32(v.zoom, 0.15, 4)
	wx, wy := (float64(x)-float64(v.panX))/float64(old), (float64(y)-float64(v.panY))/float64(old)
	v.zoom = clamp32(old*factor, 0.15, 4)
	v.panX = sceneFinitePan(float64(x) - wx*float64(v.zoom))
	v.panY = sceneFinitePan(float64(y) - wy*float64(v.zoom))
}

func finiteSceneInput(values ...float32) bool {
	for _, value := range values {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return false
		}
	}
	return true
}

func (v *sceneView) point(point enginescene.Point, r ui.Rect) [2]float32 {
	return [2]float32{r.X + float32(point.X)*v.zoom + v.panX, r.Y + float32(point.Y)*v.zoom + v.panY}
}

func (v *sceneView) drawScene(p *ui.Painter, r ui.Rect) {
	v.frameNodes, v.frameEdges, v.frameLabels, v.frameLabelUses, v.frameHiddenLabels = 0, 0, 0, 0, 0
	v.canvasRect = r
	if !v.fitted {
		v.fitScene()
	}
	if index, ok := v.index[v.pendingCenter]; ok {
		v.centerPendingScene(index)
	}
	theme := p.Theme()
	p.Fill(r, theme.Background, 0)
	spacing := 40 * v.zoom
	if spacing >= 18 && finiteSceneInput(r.W, r.H) {
		// Iteration is bounded by viewport dimensions, never scene coordinates.
		for x, count := float32(math.Mod(float64(v.panX), float64(spacing))), 0; x < r.W && count < 4096; x, count = x+spacing, count+1 {
			if x >= 0 {
				p.Line(r.X+x, r.Y, r.X+x, r.Y+r.H, 1, theme.Border.Alpha(0.35))
			}
		}
		for y, count := float32(math.Mod(float64(v.panY), float64(spacing))), 0; y < r.H && count < 4096; y, count = y+spacing, count+1 {
			if y >= 0 {
				p.Line(r.X, r.Y+y, r.X+r.W, r.Y+y, 1, theme.Border.Alpha(0.35))
			}
		}
	}
	p.Clip(r, 0, func() {
		for _, edge := range v.scene.Edges {
			color := theme.Border
			if edge.Stroke != "" {
				color = ui.Hex(edge.Stroke)
			}
			from, to := v.point(edge.From, r), v.point(edge.To, r)
			a, b, ok := clipSegment(from, to, r)
			if !ok {
				continue
			}
			v.frameEdges++
			points := [2][2]float32{a, b}
			p.StrokePolyline(points[:], max32(1.2, 1.5*v.zoom), color)
			if intersects(ui.Rect{X: to[0], Y: to[1]}, r) {
				drawArrow(p, from, to, color, v.zoom)
			}
		}
		for _, node := range v.scene.Nodes {
			point := v.point(enginescene.Point{X: node.X, Y: node.Y}, r)
			box := ui.Rect{X: point[0], Y: point[1], W: sceneDimension(node.Width) * v.zoom, H: sceneDimension(node.Height) * v.zoom}
			if !intersects(box, r) {
				continue
			}
			v.frameNodes++
			fill, stroke, textColor := sceneNodeColors(node, v.selectedID, theme)
			radius := max32(4, min32(box.W, box.H)*0.16)
			p.Fill(box, fill, radius)
			p.Stroke(box, stroke, radius, max32(1, 1.25*v.zoom))
			v.drawSceneLabel(p, node, box, theme.Font, textColor)
			if node.Collapsed {
				badge := ui.Rect{X: box.X + box.W - 13, Y: box.Y + box.H/2 - 7, W: 14, H: 14}
				p.Fill(badge, theme.Accent, 7)
				p.Text(badge.X+4, badge.Y+1, "+", 10, theme.Background)
			}
		}
	})
}

func sceneNodeColors(node enginescene.Node, selected string, theme *ui.Theme) (ui.Color, ui.Color, ui.Color) {
	fill, stroke, textColor := theme.Surface, theme.Border, theme.Text
	if node.Root {
		fill, stroke = theme.Accent.Alpha(0.16), theme.Accent
	}
	if node.Fill != "" {
		fill = ui.Hex(node.Fill)
	}
	if node.Stroke != "" {
		stroke = ui.Hex(node.Stroke)
	}
	if node.Text != "" {
		textColor = ui.Hex(node.Text)
	}
	if node.ID == selected {
		stroke = theme.Accent
	}
	return fill, stroke, textColor
}

func (v *sceneView) drawSceneLabel(p *ui.Painter, node enginescene.Node, box ui.Rect, font string, color ui.Color) {
	if node.ID != v.selectedID && (v.zoom < 0.45 || v.frameLabelUses >= maxGraphLabelLayouts-1) {
		v.frameHiddenLabels++
		return
	}
	v.frameLabelUses++
	key := labelKey{node.ID, font, max32(10, min32(18, 14*v.zoom)), min32(max32(box.W-16, 24), 1024)}
	layout := v.labels.get(key)
	if layout == nil {
		layout = ui.ShapeText(node.Label, ui.Font{Family: font, Size: key.size}, key.width)
		v.labels.put(key, layout)
		v.frameLabels++
	}
	w, h := layout.Size()
	p.Clip(box, 4, func() { p.TextLayout(layout, box.X+(box.W-w)/2, box.Y+(box.H-h)/2, color) })
}
