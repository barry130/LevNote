package graphview

import (
	"errors"
	"math"
	"slices"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/formats/drawio"
)

type drawioGestureKind uint8

const (
	drawioNoGesture drawioGestureKind = iota
	drawioMarquee
	drawioMove
)

type drawioPointerGesture struct {
	kind                               drawioGestureKind
	action                             drawioAction
	canvas                             ui.Rect
	zoom, panX, panY                   float32
	startX, startY, currentX, currentY float32
	extend, moved                      bool
	candidates                         []string
	origins                            map[string]drawio.PositionUpdate
	dx, dy                             float64
}

func (v *view) drawioGestureCurrent(g drawioPointerGesture) bool {
	return g.kind != drawioNoGesture && v.sameDrawioSource(g.action) && !v.flushing && !v.CompositionActive() &&
		v.canvasRect == g.canvas && v.zoom == g.zoom && v.panX == g.panX && v.panY == g.panY && !v.syncFailed
}

func (v *view) drawioPointerDown(ev ui.InputEvent) bool {
	g := v.captureDrawioAction()
	if v.flushing || v.CompositionActive() {
		return true
	}
	v.ensureSynced(nil)
	if !v.sameDrawioSource(g) || v.syncFailed {
		return true
	}
	id, hit := v.drawioNodeAt(ev.X, ev.Y)
	retains := hit && ev.Button == 0 && ev.Mods&ui.Cmd == 0 && ev.Clicks < 2 && v.drawSelection.members[id] && id == v.selectedID && v.connectSource == ""
	if !retains {
		var err error
		g, err = v.flushDrawioAction(g)
		if err != nil || !v.sameDrawioSource(g) {
			return true
		}
	}
	if !v.pendingDraft() {
		v.canvasHandle.Focus()
	}
	v.drawGesture = drawioPointerGesture{}
	if ev.Button != 0 {
		v.cancelGraphGesture()
		v.beginPan(ev)
		return true
	}
	v.panning = false
	if !hit {
		if edge, ok := v.drawioEdgeAt(ev.X, ev.Y); ok {
			v.cancelGraphGesture()
			if ev.Mods&ui.Cmd != 0 {
				v.addWarning("多选仅包含图形")
				return true
			}
			v.setDrawioSingle(edge)
			v.refreshLabelDraft()
			return true
		}
		v.cancelGraphGesture()
		if ev.Mods&ui.Shift != 0 {
			v.drawGesture = drawioPointerGesture{kind: drawioMarquee, action: v.captureDrawioAction(), canvas: v.canvasRect,
				zoom: v.zoom, panX: v.panX, panY: v.panY, startX: ev.X, startY: ev.Y, currentX: ev.X, currentY: ev.Y, extend: ev.Mods&ui.Cmd != 0}
			return true
		}
		v.setDrawioSelection(nil, "")
		v.refreshLabelDraft()
		v.beginPan(ev)
		return true
	}
	if ev.Mods&ui.Cmd != 0 {
		v.cancelGraphGesture()
		v.toggleDrawioSelection(id)
		v.refreshLabelDraft()
		return true
	}
	if v.connectSource != "" && !v.drawioMultiSelected() {
		source := v.connectSource
		v.connectSource = ""
		if source != id {
			v.addDiagramConnector(source, id)
		}
		return true
	}
	if ev.Clicks >= 2 {
		v.cancelGraphGesture()
		v.setDrawioSingle(id)
		v.refreshLabelDraft()
		v.focusTopic()
		return true
	}
	if v.drawSelection.members[id] {
		v.setDrawioSelection(v.drawSelection.ids, id)
	} else {
		v.setDrawioSingle(id)
	}
	v.refreshLabelDraft()
	if v.host.ReadOnly || v.host.Buffer.InGroup() {
		return true
	}
	if err := v.diagram.ValidatePositionTargets(v.pageIndex, v.drawSelection.ids); err != nil {
		v.reportDrawioAction(v.captureDrawioAction(), err)
		return true
	}
	origins := make(map[string]drawio.PositionUpdate, len(v.drawSelection.ids))
	for _, node := range v.currentPage().Model.Nodes {
		if v.drawSelection.members[node.ID] {
			origins[node.ID] = drawio.PositionUpdate{CellID: node.ID, X: node.X, Y: node.Y}
		}
	}
	v.drawGesture = drawioPointerGesture{kind: drawioMove, action: v.captureDrawioAction(), canvas: v.canvasRect, zoom: v.zoom,
		panX: v.panX, panY: v.panY, startX: ev.X, startY: ev.Y, currentX: ev.X, currentY: ev.Y, origins: origins}
	// Maintain legacy single-drag state for consumers and ensureSynced guard.
	if len(v.drawSelection.ids) <= 1 {
		v.draggingID = id
	}
	return true
}

func (v *view) moveDrawioPointer(ev ui.InputEvent) bool {
	g := &v.drawGesture
	if g.kind == drawioNoGesture {
		return false
	}
	if !v.drawioGestureCurrent(*g) {
		v.drawGesture = drawioPointerGesture{}
		v.draggingID = ""
		return true
	}
	g.currentX, g.currentY = ev.X, ev.Y
	distance := max(abs32(ev.X-g.startX), abs32(ev.Y-g.startY))
	if g.kind == drawioMarquee {
		g.moved = distance >= 4
		g.candidates = nil
		if g.moved {
			box := drawioMarqueeBox(*g)
			// Convert screen-space marquee to world-space for containment test.
			// diagramNodeBox enforces a minimum 8px screen size which would exclude
			// small nodes at extreme zoom; use actual node geometry instead.
			wx1, wy1 := v.screenToWorld(box.X-v.canvasRect.X, box.Y-v.canvasRect.Y)
			wx2, wy2 := v.screenToWorld(box.X+box.W-v.canvasRect.X, box.Y+box.H-v.canvasRect.Y)
			wLeft, wTop := min32(wx1, wx2), min32(wy1, wy2)
			wRight, wBottom := max32(wx1, wx2), max32(wy1, wy2)
			for _, node := range v.currentPage().Model.Nodes {
				nx := float32(node.X)
				ny := float32(node.Y)
				nw := float32(node.Width)
				nh := float32(node.Height)
				if nx >= wLeft && ny >= wTop && nx+nw <= wRight && ny+nh <= wBottom {
					g.candidates = append(g.candidates, node.ID)
				}
			}
		}
		return true
	}
	g.moved = distance >= 1
	if !g.moved {
		g.dx, g.dy = 0, 0
		return true
	}
	primary, ok := g.origins[g.action.primary]
	if !ok {
		v.drawGesture = drawioPointerGesture{}
		return true
	}
	// Snap the primary once in float64. Every member receives this same delta.
	g.dx = math.Round((primary.X+float64(ev.X-g.startX)/float64(g.zoom))/4)*4 - primary.X
	g.dy = math.Round((primary.Y+float64(ev.Y-g.startY)/float64(g.zoom))/4)*4 - primary.Y
	return true
}

func (v *view) endDrawioPointer(ev ui.InputEvent) bool {
	if v.drawGesture.kind == drawioNoGesture {
		return false
	}
	if !ev.Canceled {
		v.moveDrawioPointer(ev)
	}
	g := v.drawGesture
	v.drawGesture = drawioPointerGesture{}
	v.draggingID = ""
	if ev.Canceled || !g.moved || !v.drawioGestureCurrent(g) {
		return true
	}
	if g.kind == drawioMarquee {
		ids := slices.Clone(g.candidates)
		primary := ""
		if g.extend {
			ids = slices.Clone(g.action.ids)
			for _, id := range g.candidates {
				if !slices.Contains(ids, id) {
					ids = append(ids, id)
				}
			}
			primary = g.action.primary
		}
		if len(ids) > 0 && !slices.Contains(ids, primary) {
			primary = ids[len(ids)-1]
		}
		v.setDrawioSelection(ids, primary)
		v.refreshLabelDraft()
		return true
	}
	if g.dx == 0 && g.dy == 0 {
		return true
	}
	v.applyDrawioPositions(g.action, func(document *drawio.Document) ([]drawio.PositionUpdate, error) {
		if err := document.ValidatePositionTargets(g.action.page, g.action.ids); err != nil {
			return nil, err
		}
		updates := make([]drawio.PositionUpdate, 0, len(g.action.ids))
		for _, id := range g.action.ids {
			original, ok := g.origins[id]
			if !ok {
				return nil, errors.New("选中图形的坐标已失效")
			}
			updates = append(updates, drawio.PositionUpdate{CellID: id, X: original.X + g.dx, Y: original.Y + g.dy})
		}
		return updates, nil
	})
	return true
}

func drawioMarqueeBox(g drawioPointerGesture) ui.Rect {
	x1, y1 := clamp32(g.startX, 0, g.canvas.W), clamp32(g.startY, 0, g.canvas.H)
	x2, y2 := clamp32(g.currentX, 0, g.canvas.W), clamp32(g.currentY, 0, g.canvas.H)
	return ui.Rect{X: min32(x1, x2), Y: min32(y1, y2), W: abs32(x2 - x1), H: abs32(y2 - y1)}
}

func (v *view) drawioPreviewPosition(node drawio.Node) (float64, float64, bool) {
	g := v.drawGesture
	if g.kind == drawioMove {
		if origin, ok := g.origins[node.ID]; ok {
			return origin.X + g.dx, origin.Y + g.dy, true
		}
	}
	return 0, 0, false
}

func (v *view) drawDrawioGesture(p *ui.Painter, canvas ui.Rect, theme *ui.Theme) {
	g := v.drawGesture
	if g.kind != drawioMarquee || !g.moved {
		return
	}
	box := drawioMarqueeBox(g)
	box.X += canvas.X
	box.Y += canvas.Y
	p.Fill(box, theme.Accent.Alpha(0.12), 0)
	p.Stroke(box, theme.Accent, 0, 1)
	for _, node := range v.currentPage().Model.Nodes {
		if slices.Contains(g.candidates, node.ID) {
			rect := v.diagramNodeBox(node, canvas)
			p.Stroke(rect, theme.Accent, 0, 2)
		}
	}
}
