package imageview

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

func (v *view) drawToolbar(c *ui.Context) {
	theme := c.Theme()
	hasImage := v.bitmap != nil
	ui.Column(c).
		Key("image-toolbar").
		Shrink(0).
		MinHeight(46).
		Padding(6, 12).
		Gap(4).
		Background(theme.Surface).
		Border(1, theme.Border).
		Children(func() {
			ui.Row(c).Key("image-zoom-controls").FillWidth().Wrap().Gap(6).
				Tooltip("图片区域获焦后：方向键移动").Children(func() {
				if ui.Button(c, "适合窗口").Shrink(0).Label("适合窗口").Height(30).Disabled(!hasImage).
					Tooltip("适合窗口：0").Clicked() {
					v.fitMode()
					v.canvas.Focus()
				}
				if ui.Button(c, "100%").Shrink(0).Label("按百分之百显示").Height(30).Disabled(!hasImage).
					Tooltip("100%：1").Clicked() {
					v.zoomToCenter(1)
					v.canvas.Focus()
				}
				if ui.Button(c, "减小").Shrink(0).Label("减小图片显示").Height(30).
					Disabled(!hasImage || v.viewport.zoom <= MinZoom).
					Tooltip("减小：-").Clicked() {
					v.zoomAroundCenter(1 / 1.25)
					v.canvas.Focus()
				}
				ui.Text(c, fmt.Sprintf("%d%%", percent(v.viewport.zoom))).
					Label("当前缩放比例").
					FontSize(12).
					Width(48).
					NoWrap()
				if ui.Button(c, "增大").Shrink(0).Label("增大图片显示").Height(30).
					Disabled(!hasImage || v.viewport.zoom >= MaxZoom).
					Tooltip("增大：+ / =").Clicked() {
					v.zoomAroundCenter(1.25)
					v.canvas.Focus()
				}
			})
			ui.Row(c).Key("image-details").FillWidth().Shrink(0).Justify(ui.End).Gap(6).Children(func() {
				if meta := v.metadata(); meta != "" {
					ui.Text(c, meta).
						Label("图片元数据").
						Grow(1).
						MinWidth(0).
						FontSize(12).
						TextColor(theme.TextMuted).
						MaxLines(1).
						Ellipsis("…")
				}
				if v.host.ShowSource != nil && ui.Button(c, "查看字节").Label("查看只读字节").Height(30).Clicked() {
					v.requestSource()
				}
			})
		})
}

func percent(zoom float64) int {
	if !finite(zoom) {
		return 100
	}
	value := math.Round(zoom * 100)
	if value < 1 {
		value = 1
	}
	if value > 9999 {
		value = 9999
	}
	return int(value)
}

func (v *view) drawViewport(p *ui.Painter, r ui.Rect) {
	if !validBox(float64(r.W), float64(r.H)) {
		v.canvasRect = ui.Rect{}
		return
	}
	oldCanvasRect := v.canvasRect
	changed := oldCanvasRect != r
	v.canvasRect = r
	theme := p.Theme()
	p.Fill(r, theme.Background, 0)

	if v.bitmap == nil {
		return
	}
	if v.viewport.fitMode && changed {
		v.applyFit()
	}
	if !v.viewport.fitMode && changed {
		next, ok := preserveCenterOnResize(
			v.viewport,
			float64(oldCanvasRect.W), float64(oldCanvasRect.H),
			float64(r.W), float64(r.H),
			v.imageWidth, v.imageHeight,
		)
		if ok {
			v.viewport = next
		} else {
			panX, panY, clampOK := clampPanPair(
				float64(r.W), float64(r.H), v.imageWidth, v.imageHeight,
				v.viewport.zoom, v.viewport.panX, v.viewport.panY,
			)
			if clampOK {
				v.viewport.panX, v.viewport.panY = panX, panY
			}
		}
	}
	imageRect, ok := imageRectangle(r, v.imageWidth, v.imageHeight, v.viewport)
	if !ok {
		p.Text(r.X+18, r.Y+18, "图片尺寸或缩放状态无效", 13, theme.Warning)
		return
	}
	visible := intersectRect(imageRect, r)
	if visible.W <= 0 || visible.H <= 0 {
		return
	}
	p.Clip(r, 0, func() {
		drawCheckerboard(p, imageRect, visible, theme.Dark)
		p.Image(v.bitmap, imageRect, ui.FillBox)
	})
}

// drawCheckerboard paints only the visible part of the image rectangle. Cell
// indices stay anchored to the image origin, so panning moves the neutral
// backdrop with the pixels rather than making it shimmer independently.
func drawCheckerboard(p *ui.Painter, imageRect, visible ui.Rect, dark bool) {
	const cell = float32(12)
	even, odd := ui.RGB(236, 236, 236), ui.RGB(207, 207, 207)
	if dark {
		even, odd = ui.RGB(74, 74, 78), ui.RGB(49, 49, 53)
	}
	startColumn := int(math.Floor(float64((visible.X - imageRect.X) / cell)))
	endColumn := int(math.Ceil(float64((visible.X + visible.W - imageRect.X) / cell)))
	startRow := int(math.Floor(float64((visible.Y - imageRect.Y) / cell)))
	endRow := int(math.Ceil(float64((visible.Y + visible.H - imageRect.Y) / cell)))
	for row := startRow; row < endRow; row++ {
		top := imageRect.Y + float32(row)*cell
		bottom := min32(top+cell, visible.Y+visible.H)
		top = max32(top, visible.Y)
		if bottom <= top {
			continue
		}
		for column := startColumn; column < endColumn; column++ {
			left := imageRect.X + float32(column)*cell
			right := min32(left+cell, visible.X+visible.W)
			left = max32(left, visible.X)
			if right <= left {
				continue
			}
			color := even
			if (row+column)&1 != 0 {
				color = odd
			}
			p.Fill(ui.Rect{X: left, Y: top, W: right - left, H: bottom - top}, color, 0)
		}
	}
}

func (v *view) applyFit() {
	next, ok := fitViewport(float64(v.canvasRect.W), float64(v.canvasRect.H), v.imageWidth, v.imageHeight)
	if !ok {
		return
	}
	v.viewport = next
}

func (v *view) fitMode() {
	v.viewport.fitMode = true
	v.applyFit()
}

func (v *view) zoomAroundCenter(factor float64) {
	v.zoomAt(float64(v.canvasRect.W)/2, float64(v.canvasRect.H)/2, factor)
}

func (v *view) zoomToCenter(target float64) {
	next, ok := zoomTo(
		v.viewport,
		float64(v.canvasRect.W), float64(v.canvasRect.H),
		v.imageWidth, v.imageHeight,
		float64(v.canvasRect.W)/2, float64(v.canvasRect.H)/2,
		target,
	)
	if !ok {
		return
	}
	if next.zoom == v.viewport.zoom && next.panX == v.viewport.panX && next.panY == v.viewport.panY {
		// An explicit target is a mode selection even when a small image's
		// fitted scale and pan already match it. A later resize must keep 100%
		// manual rather than silently resuming fit behavior.
		v.viewport.fitMode = false
		return
	}
	next.fitMode = false
	v.viewport = next
}

func (v *view) zoomAt(x, y, factor float64) {
	next, ok := zoomAround(
		v.viewport,
		float64(v.canvasRect.W), float64(v.canvasRect.H),
		v.imageWidth, v.imageHeight,
		x, y, factor,
	)
	if !ok {
		return
	}
	if next.zoom == v.viewport.zoom && next.panX == v.viewport.panX && next.panY == v.viewport.panY {
		return
	}
	next.fitMode = false
	v.viewport = next
}

func (v *view) requestSource() {
	v.sourceNotice = nil
	if v.host.ShowSource == nil {
		v.sourceNotice = fmt.Errorf("宿主未提供只读字节检视")
		return
	}
	if !v.host.ShowSource(v) && !v.released {
		v.sourceNotice = fmt.Errorf("当前标签已不能切换到只读字节；图片保持原状")
	}
}
