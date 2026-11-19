package imageview

import (
	"math"

	"github.com/egoist/mygo/ui"
)

func (v *view) handleInput(ev ui.InputEvent) bool {
	if v.bitmap == nil || v.released {
		return false
	}
	switch ev.Kind {
	case ui.InputKeyDown:
		if v.dragging {
			return false
		}
		switch ev.Key {
		case ui.KeyEqual:
			if ev.Mods == 0 || ev.Mods == ui.Shift {
				v.zoomAroundCenter(1.25)
				return true
			}
		case ui.KeyMinus:
			if ev.Mods == 0 {
				v.zoomAroundCenter(1 / 1.25)
				return true
			}
		case ui.Key0:
			if ev.Mods == 0 {
				v.fitMode()
				return true
			}
		case ui.Key1:
			if ev.Mods == 0 {
				v.zoomToCenter(1)
				return true
			}
		case ui.KeyLeft:
			if ev.Mods == 0 {
				v.panFromKeyboard(32, 0)
				return true
			}
		case ui.KeyRight:
			if ev.Mods == 0 {
				v.panFromKeyboard(-32, 0)
				return true
			}
		case ui.KeyUp:
			if ev.Mods == 0 {
				v.panFromKeyboard(0, 32)
				return true
			}
		case ui.KeyDown:
			if ev.Mods == 0 {
				v.panFromKeyboard(0, -32)
				return true
			}
		}
		if ev.Mods == 0 {
			switch ev.Code {
			case ui.CodeNumpadAdd:
				v.zoomAroundCenter(1.25)
				return true
			case ui.CodeNumpadSubtract:
				v.zoomAroundCenter(1 / 1.25)
				return true
			}
		}
	case ui.InputPointerDown:
		if ev.Button != 0 {
			return false
		}
		v.canvas.Focus()
		v.dragging = true
		v.dragStartX, v.dragStartY = float64(ev.X), float64(ev.Y)
		v.dragOriginX, v.dragOriginY = v.viewport.panX, v.viewport.panY
		return true
	case ui.InputPointerMove:
		if !v.dragging {
			return false
		}
		origin := v.viewport
		origin.panX, origin.panY = v.dragOriginX, v.dragOriginY
		next, ok := panBy(
			origin,
			float64(v.canvasRect.W), float64(v.canvasRect.H),
			v.imageWidth, v.imageHeight,
			float64(ev.X)-v.dragStartX,
			float64(ev.Y)-v.dragStartY,
		)
		if ok {
			next.fitMode = false
			v.viewport = next
		}
		return true
	case ui.InputPointerUp:
		if !v.dragging {
			return false
		}
		v.dragging = false
		return true
	case ui.InputScroll:
		factor := wheelFactor(ev.DY)
		if factor == 0 || !finite(factor) {
			return false
		}
		v.zoomAt(float64(ev.X), float64(ev.Y), factor)
		return true
	case ui.InputZoom:
		if !finite(float64(ev.Scale)) || ev.Scale <= 0 {
			return false
		}
		v.zoomAt(float64(ev.X), float64(ev.Y), float64(ev.Scale))
		return true
	}
	return false
}

func (v *view) panFromKeyboard(dx, dy float64) {
	next, ok := panBy(
		v.viewport,
		float64(v.canvasRect.W), float64(v.canvasRect.H),
		v.imageWidth, v.imageHeight,
		dx, dy,
	)
	if !ok || next.panX == v.viewport.panX && next.panY == v.viewport.panY {
		return
	}
	next.fitMode = false
	v.viewport = next
}

func wheelFactor(delta float32) float64 {
	notches := -float64(delta)
	if !finite(notches) || notches == 0 {
		return 0
	}
	// A conventional 120-unit wheel notch is several steps; touchpad deltas
	// remain proportionally small. Cap one event in either direction.
	notches = math.Max(-4, math.Min(notches, 4))
	return math.Pow(1.12, notches)
}
