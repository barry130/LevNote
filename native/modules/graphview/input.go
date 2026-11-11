package graphview

import (
	"errors"
	"fmt"

	"github.com/egoist/mygo/ui"
)

func (v *view) handleCanvasInput(ev ui.InputEvent) bool {
	v.ensureSynced(nil)
	switch ev.Kind {
	case ui.InputPointerDown:
		if err := v.Flush(); err != nil {
			return true
		}
		v.canvasHandle.Focus()
		return v.pointerDown(ev)
	case ui.InputPointerMove:
		return v.pointerMove(ev)
	case ui.InputPointerUp:
		return v.pointerUp(ev)
	case ui.InputScroll:
		v.markerPress = markerGesture{}
		v.invalidateTagActions()
		if ev.Mods&ui.Cmd != 0 {
			factor := float32(1)
			if ev.DY < 0 {
				factor = 1.12
			} else if ev.DY > 0 {
				factor = 0.89
			}
			v.zoomAt(ev.X, ev.Y, factor)
		} else {
			v.panX -= ev.DX
			v.panY -= ev.DY
		}
		return true
	case ui.InputZoom:
		v.markerPress = markerGesture{}
		v.invalidateTagActions()
		if ev.Scale > 0 {
			v.zoomAt(ev.X, ev.Y, ev.Scale)
		}
		return true
	case ui.InputCommand:
		switch ev.Text {
		case "undo":
			return v.undo()
		case "redo":
			return v.redo()
		case "delete":
			return v.deleteKey()
		}
	case ui.InputKeyDown:
		if ev.Mods&ui.Cmd != 0 {
			switch ev.Key {
			case ui.KeyZ:
				if ev.Mods&ui.Shift != 0 {
					return v.redo()
				}
				return v.undo()
			case ui.KeyY:
				return v.redo()
			}
		}
		switch ev.Key {
		case ui.KeyEscape:
			v.panning = false
			v.cancelGraphGesture()
			return true
		case ui.KeyDelete, ui.KeyBackspace:
			return v.deleteKey()
		case ui.KeyUp:
			v.panY += 32
			return true
		case ui.KeyDown:
			v.panY -= 32
			return true
		case ui.KeyLeft:
			v.panX += 32
			return true
		case ui.KeyRight:
			v.panX -= 32
			return true
		}
	}
	return false
}

func (v *view) cancelGraphGesture() {
	v.draggingID, v.reparentSource, v.connectSource = "", "", ""
	v.dragWorldX, v.dragWorldY = 0, 0
	v.dragMoved = false
}

func (v *view) pointerDown(ev ui.InputEvent) bool {
	if err := v.Flush(); err != nil {
		return true
	}
	if ev.Button != 0 {
		v.beginPan(ev)
		return true
	}
	var id string
	var hit bool
	if v.kind == kindMindmap {
		id, hit = v.mindmapNodeAt(ev.X, ev.Y)
	} else {
		id, hit = v.drawioNodeAt(ev.X, ev.Y)
	}
	if !hit {
		v.selectNode("")
		v.beginPan(ev)
		return true
	}
	if v.reparentSource != "" && v.kind == kindMindmap {
		source := v.reparentSource
		v.reparentSource = ""
		if source != id {
			v.moveMindmapNode(source, id)
		}
		v.selectNode(source)
		return true
	}
	if v.connectSource != "" && v.kind == kindDrawio {
		source := v.connectSource
		v.connectSource = ""
		if source != id {
			v.addDiagramConnector(source, id)
		}
		return true
	}
	if !v.selectNode(id) {
		return true
	}
	if ev.Clicks >= 2 {
		v.focusTopic()
	}
	if v.kind == kindDrawio {
		node := v.selectedDrawioNode()
		if node != nil {
			v.draggingID = node.ID
			v.dragStartX, v.dragStartY = ev.X, ev.Y
			v.dragWorldX, v.dragWorldY = 0, 0
			v.dragBaseX, v.dragBaseY = node.X, node.Y
			v.dragMoved = false
		}
	}
	return true
}

func (v *view) beginPan(ev ui.InputEvent) {
	v.panning = true
	v.panStartX, v.panStartY = ev.X, ev.Y
	v.panOriginX, v.panOriginY = v.panX, v.panY
}

func (v *view) pointerMove(ev ui.InputEvent) bool {
	if v.panning {
		v.panX = v.panOriginX + ev.X - v.panStartX
		v.panY = v.panOriginY + ev.Y - v.panStartY
		return true
	}
	if v.draggingID != "" {
		dx := (ev.X - v.dragStartX) / v.zoom
		dy := (ev.Y - v.dragStartY) / v.zoom
		v.dragWorldX, v.dragWorldY = dx, dy
		v.dragMoved = v.dragMoved || abs32(dx)+abs32(dy) >= 1
		return true
	}
	return false
}

func (v *view) pointerUp(ev ui.InputEvent) bool {
	if v.panning {
		v.panning = false
		return true
	}
	if v.draggingID != "" {
		id := v.draggingID
		v.draggingID = ""
		if ev.Canceled || !v.dragMoved {
			v.dragWorldX, v.dragWorldY = 0, 0
			return true
		}
		x := v.dragBaseX + float64(v.dragWorldX)
		y := v.dragBaseY + float64(v.dragWorldY)
		x = float64(roundTo(float32(x), 4))
		y = float64(roundTo(float32(y), 4))
		v.moveDiagramNode(id, x, y)
		v.dragWorldX, v.dragWorldY = 0, 0
		return true
	}
	return false
}

func (v *view) selectNode(id string) bool {
	if err := v.Flush(); err != nil {
		return false
	}
	v.ensureSynced(nil)
	v.invalidateTagActions()
	v.selectedID = id
	v.refreshLabelDraft()
	v.refreshTagDraft()
	return true
}

func (v *view) undo() bool {
	if err := v.Flush(); err != nil {
		return true
	}
	if v.host.Buffer == nil || v.host.ReadOnly || !v.host.Buffer.CanUndo() {
		return true
	}
	if v.host.Buffer.Undo() {
		v.lastSynced = ^uint64(0)
		if v.host.Changed != nil {
			v.host.Changed()
		}
		v.ensureSynced(nil)
	}
	return true
}

func (v *view) redo() bool {
	if err := v.Flush(); err != nil {
		return true
	}
	if v.host.Buffer == nil || v.host.ReadOnly || !v.host.Buffer.CanRedo() {
		return true
	}
	if v.host.Buffer.Redo() {
		v.lastSynced = ^uint64(0)
		if v.host.Changed != nil {
			v.host.Changed()
		}
		v.ensureSynced(nil)
	}
	return true
}

func (v *view) deleteKey() bool {
	if v.kind == kindMindmap {
		v.deleteMindmapNode()
	}
	return true
}

func (v *view) requireWritable() error {
	if err := v.writable(); err != nil {
		v.report(err)
		return err
	}
	return nil
}

func (v *view) requireSelection() error {
	if v.selectedID == "" {
		err := errors.New("no graph node is selected")
		v.report(err)
		return err
	}
	return nil
}

func (v *view) invalidAction(action string) error {
	err := fmt.Errorf("%s is unavailable in the current graph state", action)
	v.report(err)
	return err
}

func abs32(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}

func roundTo(value float32, step float32) float32 {
	return float32(int((value+step/2)/step)) * step
}
