package graphview

import (
	"math"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/formats/drawio"
)

func (v *view) worldBounds() (x, y, width, height float32) {
	if v.kind == kindMindmap {
		r := v.layout.Bounds
		return float32(r.X), float32(r.Y), float32(r.Width), float32(r.Height)
	}
	page := v.currentPage()
	if page == nil {
		return 0, 0, 320, 200
	}
	r := page.Model.Bounds
	if len(page.Model.Nodes) == 0 {
		return 0, 0, 320, 200
	}
	return float32(r.X), float32(r.Y), max32(float32(r.Width), 80), max32(float32(r.Height), 80)
}

func (v *view) fitCanvas() {
	if v.canvasRect.W <= 1 || v.canvasRect.H <= 1 {
		v.fitted = false
		return
	}
	x, y, width, height := v.worldBounds()
	if width <= 0 || height <= 0 {
		width, height = 320, 200
	}
	scale := min32((v.canvasRect.W-64)/width, (v.canvasRect.H-64)/height)
	if scale <= 0 {
		scale = 0.2
	}
	v.zoom = clamp32(scale, 0.18, 2.4)
	v.panX = (v.canvasRect.W-width*v.zoom)/2 - x*v.zoom
	v.panY = (v.canvasRect.H-height*v.zoom)/2 - y*v.zoom
	v.fitted = true
}

func (v *view) zoomAroundCenter(factor float32) {
	v.zoomAt(v.canvasRect.W/2, v.canvasRect.H/2, factor)
}

func (v *view) zoomAt(localX, localY, factor float32) {
	worldX, worldY := v.screenToWorld(localX, localY)
	v.zoom = clamp32(v.zoom*factor, 0.15, 4)
	v.panX = localX - worldX*v.zoom
	v.panY = localY - worldY*v.zoom
}

func (v *view) screenToWorld(localX, localY float32) (float32, float32) {
	return (localX - v.panX) / v.zoom, (localY - v.panY) / v.zoom
}

func (v *view) worldToScreen(x, y float32) (float32, float32) {
	return x*v.zoom + v.panX, y*v.zoom + v.panY
}

func (v *view) screenNodeRect(x, y, width, height float32) ui.Rect {
	sx, sy := v.worldToScreen(x, y)
	return ui.Rect{X: sx, Y: sy, W: width * v.zoom, H: height * v.zoom}
}

func (v *view) mindmapNodeAt(localX, localY float32) (string, bool) {
	worldX, worldY := v.screenToWorld(localX, localY)
	for index := len(v.layout.Nodes) - 1; index >= 0; index-- {
		node := v.layout.Nodes[index]
		if worldX >= float32(node.X) && worldX <= float32(node.X+node.Width) &&
			worldY >= float32(node.Y) && worldY <= float32(node.Y+node.Height) {
			return node.ID, true
		}
	}
	return "", false
}

func (v *view) drawioNodeAt(localX, localY float32) (string, bool) {
	page := v.currentPage()
	if page == nil {
		return "", false
	}
	worldX, worldY := v.screenToWorld(localX, localY)
	for index := len(page.Model.Nodes) - 1; index >= 0; index-- {
		node := page.Model.Nodes[index]
		x, y := node.X, node.Y
		if v.draggingID == node.ID {
			x += float64(v.dragWorldX)
			y += float64(v.dragWorldY)
		}
		width := max(node.Width, 8)
		height := max(node.Height, 8)
		if float64(worldX) >= x && float64(worldX) <= x+width &&
			float64(worldY) >= y && float64(worldY) <= y+height {
			return node.ID, true
		}
	}
	return "", false
}

func (v *view) displayNodePosition(node drawio.Node) (float64, float64) {
	if v.draggingID == node.ID {
		return node.X + float64(v.dragWorldX), node.Y + float64(v.dragWorldY)
	}
	return node.X, node.Y
}

func clamp32(value, low, high float32) float32 {
	if math.IsNaN(float64(value)) {
		return low
	}
	return min32(max32(value, low), high)
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
