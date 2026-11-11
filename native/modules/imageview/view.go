package imageview

import (
	"errors"
	"fmt"
	"image"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/ui/editor"
)

type view struct {
	host editor.Host

	released       bool
	epoch          uint64
	requested      bool
	completionOpen bool
	cancel         func()
	loading        bool

	format          string
	sourceWidth     int
	sourceHeight    int
	orientation     uint8
	compressedBytes int64
	imageWidth      int
	imageHeight     int
	bitmap          *ui.Bitmap

	canvas       ui.Handle
	canvasRect   ui.Rect
	viewport     viewport
	dragging     bool
	dragStartX   float64
	dragStartY   float64
	dragOriginX  float64
	dragOriginY  float64
	err          error
	sourceNotice error

	disposeBitmap func(*ui.Bitmap) bool
}

func newView(host editor.Host) *view {
	return &view{host: host, viewport: viewport{zoom: 1, fitMode: true}}
}

func (v *view) Draw(c *ui.Context) {
	if v.released {
		v.released = false
		v.requested = false
		v.completionOpen = false
		v.loading = false
		v.err = nil
		v.sourceNotice = nil
		v.viewport.fitMode = true
		v.dragging = false
	}
	v.requestImage()

	theme := c.Theme()
	ui.Column(c).Fill().MinHeight(0).Clip().Background(theme.Background).Children(func() {
		v.drawToolbar(c)
		if v.sourceNotice != nil {
			ui.Text(c, "只读字节视图未打开："+v.sourceNotice.Error()).
				FontSize(11).Padding(5, 12).TextColor(theme.Warning)
		}
		canvas := ui.Box(c).
			Key("image-viewport").
			Grow(1).
			MinHeight(120).
			Clip().
			Focusable().
			Label("图片查看区域").
			Role(ui.RoleGroup).
			Bind(&v.canvas).
			HandleInput(v.handleInput).
			Draw(v.drawViewport)
		if message := v.status(); message != "" {
			canvas.Children(func() {
				color := theme.TextMuted
				if v.err != nil {
					color = theme.Warning
				}
				ui.Column(c).Fill().Center().Padding(18).Children(func() {
					ui.Text(c, message).FontSize(13).TextColor(color).MaxLines(4).Ellipsis("…")
				})
			})
		}
	})
}

func (v *view) Release() {
	if v.released {
		return
	}
	v.released = true
	v.epoch++
	v.completionOpen = false
	v.requested = false
	v.loading = false
	cancel := v.cancel
	v.cancel = nil
	if cancel != nil {
		cancel()
	}
	v.format = ""
	v.sourceWidth, v.sourceHeight = 0, 0
	v.orientation = 0
	v.compressedBytes = 0
	v.imageWidth, v.imageHeight = 0, 0
	v.err = nil
	v.sourceNotice = nil
	v.dragging = false
	v.dragStartX, v.dragStartY, v.dragOriginX, v.dragOriginY = 0, 0, 0, 0
	v.canvas.CancelFocus()
	// Keep an explicit disposal failure visible after all image state is gone.
	v.dispose()
}

func (v *view) Warnings() []string {
	var warnings []string
	if v.err != nil {
		warnings = append(warnings, "图片无法显示："+v.err.Error())
	}
	if v.sourceNotice != nil {
		warnings = append(warnings, "只读字节视图未打开："+v.sourceNotice.Error())
	}
	return warnings
}

func (*view) Export(string) ([]byte, error) {
	return nil, errors.New("图片查看器不提供文档导出")
}

func (v *view) requestImage() {
	if v.requested || v.released || v.bitmap != nil {
		return
	}
	v.requested = true
	v.loading = true
	v.err = nil
	v.sourceNotice = nil
	if v.host.BeginImage == nil {
		v.loading = false
		v.err = errors.New("宿主未提供图片解码服务")
		return
	}
	epoch := v.epoch
	v.completionOpen = true
	cancel, err := v.host.BeginImage(v, editor.ImageRequest{Epoch: epoch}, func(result editor.ImageResult, decodeErr error) {
		v.complete(epoch, result, decodeErr)
	})
	if err != nil {
		if v.completionOpen && v.epoch == epoch && !v.released {
			v.completionOpen = false
			v.loading = false
			v.err = err
		}
		if cancel != nil {
			cancel()
		}
		return
	}
	if v.completionOpen && v.epoch == epoch && !v.released {
		v.cancel = cancel
		return
	}
	// A host may complete synchronously despite the asynchronous contract.
	// Canceling the returned handle is then harmless and keeps cleanup local.
	if cancel != nil {
		cancel()
	}
}

func (v *view) complete(epoch uint64, result editor.ImageResult, err error) {
	if !v.completionOpen || v.released || v.epoch != epoch {
		result.Pixels = nil
		return
	}
	v.completionOpen = false
	v.cancel = nil
	v.loading = false
	if err != nil {
		result.Pixels = nil
		v.err = err
		return
	}
	if validationErr := validateResult(result); validationErr != nil {
		result.Pixels = nil
		v.err = validationErr
		return
	}
	bitmap := ui.NewBitmap(result.Pixels)
	result.Pixels = nil
	if bitmap == nil {
		v.err = errors.New("位图创建失败")
		return
	}
	v.bitmap = bitmap
	v.format = result.Format
	v.imageWidth, v.imageHeight = result.Width, result.Height
	v.sourceWidth, v.sourceHeight = result.SourceWidth, result.SourceHeight
	v.orientation = result.Orientation
	v.compressedBytes = result.CompressedBytes
	v.err = nil
	if v.viewport.fitMode && v.canvasRect.W > 0 && v.canvasRect.H > 0 {
		v.applyFit()
	}
}

func validateResult(result editor.ImageResult) error {
	if result.Format == "" {
		return errors.New("解码结果缺少格式名称")
	}
	if !validImage(result.Width, result.Height) || !validImage(result.SourceWidth, result.SourceHeight) {
		return errors.New("解码结果包含无效尺寸")
	}
	if result.Orientation > 8 {
		return errors.New("解码结果的 EXIF 方向无效")
	}
	if result.CompressedBytes < 0 {
		return errors.New("解码结果包含无效压缩大小")
	}
	expectedWidth, expectedHeight := result.SourceWidth, result.SourceHeight
	if result.Orientation >= 5 {
		expectedWidth, expectedHeight = expectedHeight, expectedWidth
	}
	if result.Width != expectedWidth || result.Height != expectedHeight {
		return errors.New("解码结果的方向后尺寸与源尺寸不一致")
	}
	if result.Pixels == nil {
		return errors.New("解码成功但未提供像素")
	}
	bounds := result.Pixels.Bounds()
	if bounds.Min != (image.Point{}) || bounds.Dx() != result.Width || bounds.Dy() != result.Height ||
		result.Pixels.Stride != 4*result.Width ||
		len(result.Pixels.Pix) != 4*result.Width*result.Height {
		return errors.New("解码像素不是紧密排列的零起点 RGBA")
	}
	return nil
}

func (v *view) dispose() {
	bitmap := v.bitmap
	v.bitmap = nil
	if bitmap == nil {
		return
	}
	dispose := v.disposeBitmap
	v.disposeBitmap = nil
	ok := bitmap.Dispose()
	if dispose != nil {
		ok = dispose(bitmap)
	}
	if !ok {
		err := errors.New("图片位图释放失败；宿主应释放的是本视图拥有的位图")
		v.err = err
		if v.host.Report != nil {
			v.host.Report(err)
		}
	}
}

func (v *view) status() string {
	if v.err != nil {
		return "图片无法显示：" + v.err.Error()
	}
	if v.loading || v.bitmap == nil {
		return "正在加载图片…"
	}
	return ""
}

func (v *view) metadata() string {
	if v.bitmap == nil {
		return ""
	}
	size := float64(v.compressedBytes)
	unit := "B"
	if size >= 1024 {
		size /= 1024
		unit = "KiB"
	}
	if size >= 1024 {
		size /= 1024
		unit = "MiB"
	}
	return fmt.Sprintf("%s · %d × %d · %.1f %s", v.format, v.imageWidth, v.imageHeight, size, unit)
}
