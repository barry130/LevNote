package imageview

import (
	"math"

	"github.com/egoist/mygo/ui"
)

const (
	// MinZoom is the lower bound for explicit manual zoom actions. Fit has its
	// own validity rule and may need a smaller scale to show a very tall image.
	MinZoom = 0.02
	// MaxZoom bounds manual magnification at eight DIPs per image pixel (800%).
	MaxZoom = 8.0
	// FitPadding leaves this many DIPs on each side of a fitted image.
	FitPadding = 24.0
)

type viewport struct {
	zoom    float64
	panX    float64
	panY    float64
	fitMode bool
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func validBox(width, height float64) bool {
	return finite(width) && finite(height) && width > 0 && height > 0
}

func validImage(width, height int) bool {
	return width > 0 && height > 0
}

func validDisplayZoom(zoom float64) bool {
	return finite(zoom) && zoom > 0 && zoom <= MaxZoom
}

func validManualZoom(zoom float64) bool {
	return finite(zoom) && zoom >= MinZoom && zoom <= MaxZoom
}

// fitScale returns a finite positive display scale that preserves aspect ratio
// and never enlarges a tiny image beyond 100%. Fit is for reading the whole
// image, so it is deliberately not subject to the explicit manual zoom floor.
func fitScale(viewWidth, viewHeight float64, imageWidth, imageHeight int) (float64, bool) {
	if !validBox(viewWidth, viewHeight) || !validImage(imageWidth, imageHeight) {
		return 0, false
	}
	availableWidth := math.Max(viewWidth-2*FitPadding, 0.5)
	availableHeight := math.Max(viewHeight-2*FitPadding, 0.5)
	scale := math.Min(1,
		math.Min(
			availableWidth/float64(imageWidth),
			availableHeight/float64(imageHeight),
		),
	)
	if !validDisplayZoom(scale) {
		return 0, false
	}
	return scale, true
}

func clampZoom(zoom float64) float64 {
	if !finite(zoom) {
		return 1
	}
	return math.Max(MinZoom, math.Min(zoom, MaxZoom))
}

func centeredPan(viewSize, imageSize float64, zoom float64) float64 {
	return (viewSize - imageSize*zoom) / 2
}

// clampPan centers an image smaller than the viewport. A larger image may move
// until either edge reaches its viewport edge, so it cannot be lost offscreen.
func clampPan(viewSize, scaledSize, pan float64) float64 {
	if !finite(viewSize) || !finite(scaledSize) || !finite(pan) || viewSize <= 0 || scaledSize <= 0 {
		return 0
	}
	if scaledSize <= viewSize {
		return (viewSize - scaledSize) / 2
	}
	return math.Max(viewSize-scaledSize, math.Min(pan, 0))
}

func clampPanPair(viewWidth, viewHeight float64, imageWidth, imageHeight int, zoom, panX, panY float64) (float64, float64, bool) {
	if !validBox(viewWidth, viewHeight) || !validImage(imageWidth, imageHeight) ||
		!validDisplayZoom(zoom) || !finite(panX) || !finite(panY) {
		return 0, 0, false
	}
	return clampPan(viewWidth, float64(imageWidth)*zoom, panX),
		clampPan(viewHeight, float64(imageHeight)*zoom, panY), true
}

// preserveCenterOnResize keeps the image point at the old viewport center at
// the new viewport center. Clamping then centers dimensions that became
// smaller than the viewport; it must not happen before this translation.
func preserveCenterOnResize(current viewport, oldWidth, oldHeight, newWidth, newHeight float64, imageWidth, imageHeight int) (viewport, bool) {
	if !validBox(oldWidth, oldHeight) || !validBox(newWidth, newHeight) ||
		!validImage(imageWidth, imageHeight) || !validDisplayZoom(current.zoom) ||
		!finite(current.panX) || !finite(current.panY) {
		return current, false
	}
	imageCenterX := (oldWidth/2 - current.panX) / current.zoom
	imageCenterY := (oldHeight/2 - current.panY) / current.zoom
	if !finite(imageCenterX) || !finite(imageCenterY) {
		return current, false
	}
	panX, panY, ok := clampPanPair(
		newWidth, newHeight, imageWidth, imageHeight,
		current.zoom,
		newWidth/2-imageCenterX*current.zoom,
		newHeight/2-imageCenterY*current.zoom,
	)
	if !ok {
		return current, false
	}
	current.panX, current.panY = panX, panY
	return current, true
}

func fitViewport(viewWidth, viewHeight float64, imageWidth, imageHeight int) (viewport, bool) {
	scale, ok := fitScale(viewWidth, viewHeight, imageWidth, imageHeight)
	if !ok {
		return viewport{}, false
	}
	return viewport{
		zoom:    scale,
		panX:    centeredPan(viewWidth, float64(imageWidth), scale),
		panY:    centeredPan(viewHeight, float64(imageHeight), scale),
		fitMode: true,
	}, true
}

// zoomAround keeps the image point under (x, y) under it when the pan can move
// there, then applies the same viewport clamp used for drag and resize.
func zoomAround(current viewport, viewWidth, viewHeight float64, imageWidth, imageHeight int, x, y, factor float64) (viewport, bool) {
	if !validBox(viewWidth, viewHeight) || !validImage(imageWidth, imageHeight) ||
		!validDisplayZoom(current.zoom) ||
		!finite(current.panX) || !finite(current.panY) || !finite(x) || !finite(y) || !finite(factor) || factor <= 0 {
		return current, false
	}
	rawZoom := current.zoom * factor
	if !finite(rawZoom) || rawZoom <= 0 {
		return current, false
	}
	nextZoom := clampZoom(rawZoom)
	if !validManualZoom(nextZoom) {
		return current, false
	}
	// A fit below the manual floor is already the most zoomed-out valid view.
	// A smaller gesture must not snap upward to MinZoom and crop the image.
	if factor <= 1 && nextZoom > current.zoom {
		return current, false
	}
	imageX := (x - current.panX) / current.zoom
	imageY := (y - current.panY) / current.zoom
	if !finite(imageX) || !finite(imageY) {
		return current, false
	}
	panX := x - imageX*nextZoom
	panY := y - imageY*nextZoom
	panX, panY, ok := clampPanPair(viewWidth, viewHeight, imageWidth, imageHeight, nextZoom, panX, panY)
	if !ok {
		return current, false
	}
	return viewport{zoom: nextZoom, panX: panX, panY: panY}, true
}

func zoomTo(current viewport, viewWidth, viewHeight float64, imageWidth, imageHeight int, x, y, target float64) (viewport, bool) {
	if !finite(target) || target <= 0 || !finite(current.zoom) || current.zoom <= 0 {
		return current, false
	}
	return zoomAround(current, viewWidth, viewHeight, imageWidth, imageHeight, x, y, target/current.zoom)
}

func panBy(current viewport, viewWidth, viewHeight float64, imageWidth, imageHeight int, dx, dy float64) (viewport, bool) {
	if !finite(dx) || !finite(dy) {
		return current, false
	}
	panX, panY, ok := clampPanPair(
		viewWidth, viewHeight, imageWidth, imageHeight,
		current.zoom, current.panX+dx, current.panY+dy,
	)
	if !ok {
		return current, false
	}
	current.panX, current.panY = panX, panY
	return current, true
}

func imageRectangle(canvas ui.Rect, imageWidth, imageHeight int, view viewport) (ui.Rect, bool) {
	if !validBox(float64(canvas.W), float64(canvas.H)) || !validImage(imageWidth, imageHeight) ||
		!validDisplayZoom(view.zoom) ||
		!finite(view.panX) || !finite(view.panY) {
		return ui.Rect{}, false
	}
	x := float64(canvas.X) + view.panX
	y := float64(canvas.Y) + view.panY
	width := float64(imageWidth) * view.zoom
	height := float64(imageHeight) * view.zoom
	castX := float64(float32(x))
	castY := float64(float32(y))
	castWidth := float64(float32(width))
	castHeight := float64(float32(height))
	if !finite(castX) || !finite(castY) || !finite(castWidth) || !finite(castHeight) ||
		castWidth <= 0 || castHeight <= 0 {
		return ui.Rect{}, false
	}
	return ui.Rect{X: float32(castX), Y: float32(castY), W: float32(castWidth), H: float32(castHeight)}, true
}

func intersectRect(a, b ui.Rect) ui.Rect {
	x0, y0 := max32(a.X, b.X), max32(a.Y, b.Y)
	x1, y1 := min32(a.X+a.W, b.X+b.W), min32(a.Y+a.H, b.Y+b.H)
	if x1 <= x0 || y1 <= y0 {
		return ui.Rect{}
	}
	return ui.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
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
