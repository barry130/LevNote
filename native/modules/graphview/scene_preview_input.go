package graphview

import "github.com/egoist/mygo/ui"

func (v *sceneView) sceneInput(ev ui.InputEvent) bool {
	if !finiteSceneInput(ev.X, ev.Y, ev.DX, ev.DY, ev.Scale) {
		return false
	}
	switch ev.Kind {
	case ui.InputPointerDown:
		v.canvasHandle.Focus()
		if ev.Button == 0 {
			x, y := (float64(ev.X)-float64(v.panX))/float64(v.zoom), (float64(ev.Y)-float64(v.panY))/float64(v.zoom)
			for i := len(v.scene.Nodes) - 1; i >= 0; i-- {
				node := v.scene.Nodes[i]
				if x >= node.X && x <= node.X+node.Width && y >= node.Y && y <= node.Y+node.Height {
					v.selectedID = node.ID
					return true
				}
			}
			v.selectedID = ""
		}
		v.panning = true
		v.panStartX, v.panStartY, v.panOriginX, v.panOriginY = ev.X, ev.Y, v.panX, v.panY
		return true
	case ui.InputPointerMove:
		if v.panning {
			v.panX = sceneFinitePan(float64(v.panOriginX) + float64(ev.X) - float64(v.panStartX))
			v.panY = sceneFinitePan(float64(v.panOriginY) + float64(ev.Y) - float64(v.panStartY))
			return true
		}
	case ui.InputPointerUp:
		was := v.panning
		v.panning = false
		return was
	case ui.InputScroll:
		if ev.Mods&ui.Cmd != 0 {
			factor := float32(1)
			if ev.DY < 0 {
				factor = 1.12
			} else if ev.DY > 0 {
				factor = 0.89
			}
			v.zoomAt(ev.X, ev.Y, factor)
		} else {
			v.panX = sceneFinitePan(float64(v.panX) - float64(ev.DX))
			v.panY = sceneFinitePan(float64(v.panY) - float64(ev.DY))
		}
		return true
	case ui.InputZoom:
		v.zoomAt(ev.X, ev.Y, ev.Scale)
		return true
	case ui.InputKeyDown:
		switch ev.Key {
		case ui.KeyEscape:
			v.panning = false
		case ui.KeyLeft:
			v.panX = sceneFinitePan(float64(v.panX) + 32)
		case ui.KeyRight:
			v.panX = sceneFinitePan(float64(v.panX) - 32)
		case ui.KeyUp:
			v.panY = sceneFinitePan(float64(v.panY) + 32)
		case ui.KeyDown:
			v.panY = sceneFinitePan(float64(v.panY) - 32)
		default:
			return false
		}
		return true
	}
	return false
}
