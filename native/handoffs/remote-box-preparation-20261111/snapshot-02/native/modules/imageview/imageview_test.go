package imageview

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/formats/raster"
	"levnote.local/native/ui/editor"
)

type fakeCall struct {
	owner       editor.View
	request     editor.ImageRequest
	complete    func(editor.ImageResult, error)
	wasCanceled bool
}

type fakeImages struct {
	calls       []*fakeCall
	beginErr    error
	cancelCount int
	sourceCalls int
	sourceOK    bool
	reports     []error
}

func (f *fakeImages) host() editor.Host {
	return editor.Host{
		BeginImage: f.begin,
		ShowSource: func(editor.View) bool {
			f.sourceCalls++
			return f.sourceOK
		},
		Report: func(err error) {
			f.reports = append(f.reports, err)
		},
	}
}

func (f *fakeImages) begin(owner editor.View, request editor.ImageRequest, complete func(editor.ImageResult, error)) (func(), error) {
	call := &fakeCall{owner: owner, request: request, complete: complete}
	f.calls = append(f.calls, call)
	return func() {
		if call.wasCanceled {
			return
		}
		call.wasCanceled = true
		f.cancelCount++
	}, f.beginErr
}

func solidRGBA(width, height int, alpha uint8, value color.NRGBA) *image.RGBA {
	pixels := image.NewRGBA(image.Rect(0, 0, width, height))
	pixel := color.RGBA{
		R: scaleAlpha(value.R, alpha),
		G: scaleAlpha(value.G, alpha),
		B: scaleAlpha(value.B, alpha),
		A: alpha,
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixels.SetRGBA(x, y, pixel)
		}
	}
	return pixels
}

func scaleAlpha(value, alpha uint8) uint8 {
	return uint8(uint32(value) * uint32(alpha) / 255)
}

func imageResult(pixels *image.RGBA) editor.ImageResult {
	return editor.ImageResult{
		Format:          "PNG",
		Width:           pixels.Rect.Dx(),
		Height:          pixels.Rect.Dy(),
		SourceWidth:     pixels.Rect.Dx(),
		SourceHeight:    pixels.Rect.Dy(),
		CompressedBytes: int64(len(pixels.Pix)),
		Pixels:          pixels,
	}
}

func completeCall(t *testing.T, f *fakeImages, index int, result editor.ImageResult, err error) {
	t.Helper()
	if index < 0 || index >= len(f.calls) {
		t.Fatalf("request %d missing; calls=%d", index, len(f.calls))
	}
	f.calls[index].complete(result, err)
}

func TestAdapterIsNarrowAndReadOnly(t *testing.T) {
	adapter := Adapter()
	if adapter.ID != "image" || adapter.Name != "原生图片阅读器" {
		t.Fatalf("adapter identity = %q/%q", adapter.ID, adapter.Name)
	}
	for _, path := range []string{"photo.png", "PHOTO.JPG", `C:\docs\scan.JPEG`} {
		if !adapter.Matches(path) {
			t.Fatalf("adapter did not match %q", path)
		}
	}
	for _, path := range []string{"photo.png.bak", "photo.webp", "photo.gif", "photo"} {
		if adapter.Matches(path) {
			t.Fatalf("adapter unexpectedly matched %q", path)
		}
	}
	if adapter.Template != nil || adapter.Outline != nil || adapter.SceneEngine != nil || adapter.ExportEngine != nil || len(adapter.Exports) != 0 {
		t.Fatal("readonly image adapter advertises an unowned capability")
	}
	created := adapter.Create(editor.Host{})
	if _, ok := created.(*view); !ok {
		t.Fatalf("Create returned %T", created)
	}
}

func TestGeometryFitAspectResizeAndBounds(t *testing.T) {
	fit, ok := fitViewport(760, 480, 1200, 800)
	if !ok {
		t.Fatal("fit rejected finite geometry")
	}
	if math.Abs(fit.zoom-0.54) > 0.0001 {
		t.Fatalf("fit zoom = %g, want 0.54", fit.zoom)
	}
	if math.Abs(fit.panX-56) > 0.0001 || math.Abs(fit.panY-24) > 0.0001 {
		t.Fatalf("fit pan = %g,%g, want 56,24", fit.panX, fit.panY)
	}
	if math.Abs(fit.panX*2+1200*fit.zoom-760) > 0.0001 ||
		math.Abs(fit.panY*2+800*fit.zoom-480) > 0.0001 {
		t.Fatal("fit did not center the image")
	}

	resized, ok := fitViewport(400, 400, 1200, 800)
	if !ok {
		t.Fatal("resized fit rejected finite geometry")
	}
	if math.Abs(resized.zoom-(352.0/1200.0)) > 0.0001 {
		t.Fatalf("resized fit zoom = %g", resized.zoom)
	}
	if ratio := (1200 * resized.zoom) / (800 * resized.zoom); math.Abs(ratio-1.5) > 0.0001 {
		t.Fatalf("resized aspect ratio = %g", ratio)
	}
	if math.Abs(resized.panX*2+1200*resized.zoom-400) > 0.0001 ||
		math.Abs(resized.panY*2+800*resized.zoom-400) > 0.0001 {
		t.Fatal("resized fit did not recenter")
	}

	tiny, ok := fitViewport(800, 600, 1, 1)
	if !ok || tiny.zoom != 1 {
		t.Fatalf("tiny fit = %#v, ok=%v; want a finite 100%% bound", tiny, ok)
	}

	oneToOne, ok := zoomTo(tiny, 800, 600, 100, 50, 400, 300, 1)
	if !ok || oneToOne.zoom != 1 || oneToOne.fitMode {
		t.Fatalf("100%% viewport = %#v, ok=%v", oneToOne, ok)
	}
	rect, ok := imageRectangle(ui.Rect{W: 800, H: 600}, 100, 50, oneToOne)
	if !ok || rect.W != 100 || rect.H != 50 {
		t.Fatalf("100%% image rect = %#v, ok=%v", rect, ok)
	}
}

func TestTallImageFitBelowManualZoomFloor(t *testing.T) {
	fit, ok := fitViewport(120, 120, 1, 8192)
	if !ok {
		t.Fatal("tall image fit rejected finite geometry")
	}
	wantZoom := 72.0 / 8192.0
	if math.Abs(fit.zoom-wantZoom) > 1e-12 || fit.zoom >= MinZoom {
		t.Fatalf("tall fit zoom = %g, want %g below manual floor %g", fit.zoom, wantZoom, MinZoom)
	}
	rect, ok := imageRectangle(ui.Rect{W: 120, H: 120}, 1, 8192, fit)
	if !ok {
		t.Fatal("drawing rejected finite fit scale below manual floor")
	}
	if rect.Y < FitPadding-0.001 || rect.Y+rect.H > 120-FitPadding+0.001 {
		t.Fatalf("tall fit rect = %#v, want full image inside 24 DIP top/bottom padding", rect)
	}

	farther, ok := zoomAround(fit, 120, 120, 1, 8192, 60, 60, 0.8)
	if ok {
		t.Fatalf("zoom-out from sub-floor fit snapped to crop instead of holding fit: %#v", farther)
	}
	nearer, ok := zoomAround(fit, 120, 120, 1, 8192, 60, 60, 1.25)
	if !ok || nearer.zoom != MinZoom || nearer.fitMode {
		t.Fatalf("zoom-in from sub-floor fit = %#v, ok=%v; want explicit manual MinZoom", nearer, ok)
	}
	if percent(fit.zoom) != 1 {
		t.Fatalf("sub-floor fit percentage = %d, want truthful 1%%", percent(fit.zoom))
	}
}

func TestResizePreservesCenterBeforeClamp(t *testing.T) {
	manual := viewport{zoom: 1, panX: -300, panY: -250}
	resized, ok := preserveCenterOnResize(manual, 400, 300, 600, 400, 1000, 800)
	if !ok || math.Abs(resized.panX+200) > 1e-9 || math.Abs(resized.panY+200) > 1e-9 {
		t.Fatalf("expandable resize pan = %#v, ok=%v; want -200,-200", resized, ok)
	}
	smaller, ok := preserveCenterOnResize(resized, 600, 400, 1200, 900, 1000, 800)
	if !ok || smaller.panX != 100 || smaller.panY != 50 {
		t.Fatalf("smaller-than-viewport resize pan = %#v, ok=%v; want centered 100,50", smaller, ok)
	}
}

func TestManualResizePreservesCenterThroughTester(t *testing.T) {
	// Tester geometry is computed through target/current.zoom multiplication.
	// A millionth of a DIP is far below pixel accuracy and tolerates only the
	// resulting floating-point residue; dimensions, modes, and bounds stay exact.
	const geometryTolerance = 1e-6
	closeEnough := func(got, want float64) bool {
		return math.Abs(got-want) <= geometryTolerance
	}

	fake := &fakeImages{}
	v := newView(fake.host())
	tester := ui.NewTester(v.Draw, 400, 300)
	completeCall(t, fake, 0, imageResult(solidRGBA(1000, 800, 255, color.NRGBA{R: 40, G: 110, B: 190})), nil)
	tester.Frame()
	if err := tester.Click("按百分之百显示"); err != nil {
		t.Fatal(err)
	}
	if v.viewport.fitMode || !closeEnough(v.viewport.zoom, 1) {
		t.Fatalf("manual 100%% state = %#v", v.viewport)
	}

	oldWidth := float64(v.canvasRect.W)
	oldHeight := float64(v.canvasRect.H)
	centerImageX := (oldWidth/2 - v.viewport.panX) / v.viewport.zoom
	centerImageY := (oldHeight/2 - v.viewport.panY) / v.viewport.zoom
	tester.SetSize(600, 400)
	if v.viewport.fitMode {
		t.Fatal("manual resize returned to fit mode")
	}
	afterWidth := float64(v.canvasRect.W)
	afterHeight := float64(v.canvasRect.H)
	gotCenterX := (afterWidth/2 - v.viewport.panX) / v.viewport.zoom
	gotCenterY := (afterHeight/2 - v.viewport.panY) / v.viewport.zoom
	if !closeEnough(gotCenterX, centerImageX) || !closeEnough(gotCenterY, centerImageY) {
		t.Fatalf("expandable resize center moved from %g,%g to %g,%g (canvas %gx%g to %gx%g)",
			centerImageX, centerImageY, gotCenterX, gotCenterY, oldWidth, oldHeight, afterWidth, afterHeight)
	}

	tester.SetSize(1400, 1000)
	if afterWidth := float64(v.canvasRect.W); afterWidth <= 1000 || !closeEnough(v.viewport.panX, (afterWidth-1000)/2) {
		t.Fatalf("width smaller than viewport: canvas=%g pan=%g, want centered %g", afterWidth, v.viewport.panX, (afterWidth-1000)/2)
	}
	if afterHeight := float64(v.canvasRect.H); afterHeight <= 800 || !closeEnough(v.viewport.panY, (afterHeight-800)/2) {
		t.Fatalf("height smaller than viewport: canvas=%g pan=%g, want centered %g", afterHeight, v.viewport.panY, (afterHeight-800)/2)
	}
}

func TestGeometryZoomAnchorPanClampAndFiniteGuards(t *testing.T) {
	start := viewport{zoom: 2, panX: -100, panY: -50}
	got, ok := zoomAround(start, 300, 200, 400, 200, 150, 100, 1.5)
	if !ok {
		t.Fatal("valid anchored zoom rejected")
	}
	beforeX := (150 - start.panX) / start.zoom
	beforeY := (100 - start.panY) / start.zoom
	afterX := (150 - got.panX) / got.zoom
	afterY := (100 - got.panY) / got.zoom
	if math.Abs(beforeX-afterX) > 1e-9 || math.Abs(beforeY-afterY) > 1e-9 {
		t.Fatalf("anchor moved from %g,%g to %g,%g", beforeX, beforeY, afterX, afterY)
	}
	if got.panX > 0 || got.panY > 0 || got.panX < 300-float64(400)*got.zoom || got.panY < 200-float64(200)*got.zoom {
		t.Fatalf("zoomed pan escaped bounds: %#v", got)
	}

	panX, panY, ok := clampPanPair(300, 200, 100, 50, 1, -80, 60)
	if !ok || panX != 100 || panY != 75 {
		t.Fatalf("small-image clamp = %g,%g, ok=%v; want centered 100,75", panX, panY, ok)
	}
	panX, panY, ok = clampPanPair(300, 200, 800, 400, 1, 100, -500)
	if !ok || panX != 0 || panY != -200 {
		t.Fatalf("large-image clamp = %g,%g, ok=%v; want 0,-200", panX, panY, ok)
	}
	dragged, ok := panBy(viewport{zoom: 1, panX: -100, panY: -50}, 300, 200, 800, 400, 40, 20)
	if !ok || dragged.panX != -60 || dragged.panY != -30 {
		t.Fatalf("drag pan = %#v, ok=%v", dragged, ok)
	}

	if _, ok := fitViewport(math.NaN(), 200, 10, 10); ok {
		t.Fatal("NaN viewport fit accepted")
	}
	if _, ok := fitViewport(300, math.Inf(1), 10, 10); ok {
		t.Fatal("infinite viewport fit accepted")
	}
	if _, ok := zoomAround(start, 300, 200, 400, 200, 10, 10, math.NaN()); ok {
		t.Fatal("NaN zoom factor accepted")
	}
	if _, ok := imageRectangle(ui.Rect{W: 300, H: 200}, 10, 10, viewport{zoom: math.Inf(1)}); ok {
		t.Fatal("infinite scale reached image drawing")
	}
	max := float32(math.MaxFloat32)
	for _, tc := range []struct {
		name   string
		canvas ui.Rect
		panX   float64
		panY   float64
	}{
		{name: "x", canvas: ui.Rect{X: max, W: 300, H: 200}, panX: float64(max)},
		{name: "y", canvas: ui.Rect{Y: max, W: 300, H: 200}, panY: float64(max)},
	} {
		if _, ok := imageRectangle(tc.canvas, 10, 10, viewport{zoom: 1, panX: tc.panX, panY: tc.panY}); ok {
			t.Fatalf("float32 position overflow on %s reached image drawing", tc.name)
		}
	}
	if wheelFactor(float32(math.NaN())) != 0 {
		t.Fatal("NaN wheel delta produced a zoom factor")
	}
}

func TestOneRequestAndPublicToolbarInput(t *testing.T) {
	fake := &fakeImages{}
	v := newView(fake.host())
	tester := ui.NewTester(v.Draw, 760, 480)
	if len(fake.calls) != 1 {
		t.Fatalf("first Draw made %d requests, want one", len(fake.calls))
	}
	if !tester.HasText("正在加载图片…") {
		t.Fatalf("loading state missing: %#v", tester.Texts())
	}
	tester.Frame()
	if len(fake.calls) != 1 {
		t.Fatalf("redraw repeated request: %d", len(fake.calls))
	}
	completeCall(t, fake, 0, imageResult(solidRGBA(400, 200, 255, color.NRGBA{R: 30, G: 140, B: 210})), nil)
	tester.Frame()
	if v.bitmap == nil {
		t.Fatalf("decoded bitmap not shown; warnings=%v", v.Warnings())
	}
	if w, h := v.bitmap.Size(); w != 400 || h != 200 {
		t.Fatalf("bitmap size = %dx%d", w, h)
	}
	for _, text := range []string{"适合窗口", "100%", "减小", "增大"} {
		if !tester.HasText(text) {
			t.Fatalf("toolbar text %q missing: %#v", text, tester.Texts())
		}
	}
	if !tester.HasText("查看字节") || tester.HasText("查看只读源码") {
		t.Fatalf("binary byte-view action label is wrong: %#v", tester.Texts())
	}

	if err := tester.Click("按百分之百显示"); err != nil {
		t.Fatal(err)
	}
	if v.viewport.zoom != 1 || v.viewport.fitMode {
		t.Fatalf("100%% action = %#v", v.viewport)
	}
	for range 4 {
		if err := tester.Click("增大图片显示"); err != nil {
			t.Fatal(err)
		}
	}
	if math.Abs(v.viewport.zoom-math.Pow(1.25, 4)) > 1e-9 {
		t.Fatalf("four increases reached %g", v.viewport.zoom)
	}

	centerX := v.canvasRect.X + v.canvasRect.W*0.62
	centerY := v.canvasRect.Y + v.canvasRect.H*0.44
	beforeX := (float64(centerX-v.canvasRect.X) - v.viewport.panX) / v.viewport.zoom
	beforeY := (float64(centerY-v.canvasRect.Y) - v.viewport.panY) / v.viewport.zoom
	zoomBefore := v.viewport.zoom
	tester.Wheel(centerX, centerY, 0, -1)
	if v.viewport.zoom <= zoomBefore {
		t.Fatalf("wheel did not zoom: %g -> %g", zoomBefore, v.viewport.zoom)
	}
	afterX := (float64(centerX-v.canvasRect.X) - v.viewport.panX) / v.viewport.zoom
	afterY := (float64(centerY-v.canvasRect.Y) - v.viewport.panY) / v.viewport.zoom
	if math.Abs(beforeX-afterX) > 1e-5 || math.Abs(beforeY-afterY) > 1e-5 {
		t.Fatalf("wheel anchor moved from %g,%g to %g,%g", beforeX, beforeY, afterX, afterY)
	}

	panBefore := v.viewport.panX
	tester.Press(v.canvasRect.X+v.canvasRect.W/2, v.canvasRect.Y+v.canvasRect.H/2)
	tester.Move(v.canvasRect.X+v.canvasRect.W/2+18, v.canvasRect.Y+v.canvasRect.H/2+4)
	tester.Move(v.canvasRect.X+v.canvasRect.W/2+36, v.canvasRect.Y+v.canvasRect.H/2+8)
	if math.Abs(v.viewport.panX-panBefore-36) > 0.01 {
		t.Fatalf("public pointer drag accumulated offsets: before=%g after=%g", panBefore, v.viewport.panX)
	}
	tester.Release(v.canvasRect.X+v.canvasRect.W/2+36, v.canvasRect.Y+v.canvasRect.H/2+8)

	if err := tester.Click("适合窗口"); err != nil {
		t.Fatal(err)
	}
	oldZoom := v.viewport.zoom
	if err := tester.Click("减小图片显示"); err != nil {
		t.Fatal(err)
	}
	if v.viewport.zoom >= oldZoom {
		t.Fatalf("decrease did not reduce zoom: %g -> %g", oldZoom, v.viewport.zoom)
	}
	if err := tester.Click("适合窗口"); err != nil {
		t.Fatal(err)
	}
	fittedBefore := v.viewport
	tester.SetSize(400, 300)
	if !v.viewport.fitMode || v.viewport.zoom == fittedBefore.zoom {
		t.Fatalf("fit did not recompute after resize: before=%#v after=%#v", fittedBefore, v.viewport)
	}
	if math.Abs(v.viewport.panX*2+400*v.viewport.zoom-float64(v.canvasRect.W)) > 0.01 ||
		math.Abs(v.viewport.panY*2+200*v.viewport.zoom-float64(v.canvasRect.H)) > 0.01 {
		t.Fatal("resized fit did not keep the image centered")
	}
}

func TestExplicit100PercentFitAtOneRemainsManualAcrossResize(t *testing.T) {
	const tolerance = 1e-9
	for _, activation := range []struct {
		name string
		run  func(*testing.T, *ui.Tester)
	}{
		{
			name: "public button",
			run: func(t *testing.T, tester *ui.Tester) {
				if err := tester.Click("按百分之百显示"); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "key 1",
			run: func(t *testing.T, tester *ui.Tester) {
				if err := tester.Click("图片查看区域"); err != nil {
					t.Fatal(err)
				}
				tester.Key(0, ui.Key1)
			},
		},
	} {
		t.Run(activation.name, func(t *testing.T) {
			fake := &fakeImages{}
			v := newView(fake.host())
			tester := ui.NewTester(v.Draw, 760, 480)
			completeCall(t, fake, 0, imageResult(solidRGBA(400, 200, 255, color.NRGBA{R: 70, G: 130, B: 180})), nil)
			tester.Frame()
			if !v.viewport.fitMode || v.viewport.zoom != 1 {
				t.Fatalf("fit-at-one fixture = %#v", v.viewport)
			}

			activation.run(t, tester)
			if v.viewport.fitMode || v.viewport.zoom != 1 {
				t.Fatalf("explicit 100%% did not select manual mode: %#v", v.viewport)
			}

			tester.SetSize(300, 240)
			if v.viewport.fitMode || v.viewport.zoom != 1 {
				t.Fatalf("resize resumed fit after explicit 100%%: %#v", v.viewport)
			}
			width, height := float64(v.canvasRect.W), float64(v.canvasRect.H)
			scaledWidth, scaledHeight := 400*v.viewport.zoom, 200*v.viewport.zoom
			if scaledWidth <= width && scaledHeight <= height {
				t.Fatalf("resized 100%% image did not exceed the viewport: canvas=%gx%g image=%gx%g",
					width, height, scaledWidth, scaledHeight)
			}
			if scaledWidth > width {
				minPanX := width - scaledWidth
				if v.viewport.panX < minPanX-tolerance || v.viewport.panX > tolerance {
					t.Fatalf("100%% horizontal pan escaped crop bounds: pan=%g range=%g..0", v.viewport.panX, minPanX)
				}
			} else if want := (width - scaledWidth) / 2; math.Abs(v.viewport.panX-want) > tolerance {
				t.Fatalf("100%% narrow image is not centered horizontally: pan=%g want=%g", v.viewport.panX, want)
			}
			if scaledHeight > height {
				minPanY := height - scaledHeight
				if v.viewport.panY < minPanY-tolerance || v.viewport.panY > tolerance {
					t.Fatalf("100%% vertical pan escaped crop bounds: pan=%g range=%g..0", v.viewport.panY, minPanY)
				}
			} else if want := (height - scaledHeight) / 2; math.Abs(v.viewport.panY-want) > tolerance {
				t.Fatalf("100%% short image is not centered vertically: pan=%g want=%g", v.viewport.panY, want)
			}

			tester.Key(0, ui.Key0)
			if !v.viewport.fitMode || v.viewport.zoom >= 1 {
				t.Fatalf("public 0 key did not restore fit below 100%%: %#v", v.viewport)
			}
			if math.Abs(v.viewport.panX*2+400*v.viewport.zoom-width) > tolerance ||
				math.Abs(v.viewport.panY*2+200*v.viewport.zoom-height) > tolerance {
				t.Fatalf("public 0 key did not center the fitted image: viewport=%#v canvas=%gx%g",
					v.viewport, width, height)
			}
		})
	}
}

func TestToolbarControlsRemainReachableAtNarrowWidths(t *testing.T) {
	for _, width := range []int{532, 300} {
		t.Run(fmt.Sprintf("%d", width), func(t *testing.T) {
			fake := &fakeImages{sourceOK: true}
			v := newView(fake.host())
			tester := ui.NewTester(v.Draw, width, 320)
			result := imageResult(solidRGBA(320, 180, 255, color.NRGBA{R: 35, G: 105, B: 175}))
			result.Format = strings.Repeat("超长格式元数据-", 32) + "PNG"
			completeCall(t, fake, 0, result, nil)
			tester.Frame()

			for _, label := range []string{
				"适合窗口",
				"按百分之百显示",
				"减小图片显示",
				"当前缩放比例",
				"增大图片显示",
				"图片元数据",
				"查看只读字节",
			} {
				rect, ok := tester.Find(label)
				if !ok {
					t.Fatalf("%q is missing at %d DIP: %#v", label, width, tester.Texts())
				}
				if rect.W <= 0 || rect.H <= 0 ||
					rect.X < 0 || rect.Y < 0 ||
					rect.X+rect.W > float32(width) || rect.Y+rect.H > 320 {
					t.Fatalf("%q bounds %+v escape the %dx320 content pane", label, rect, width)
				}
			}
			if err := tester.Click("查看只读字节"); err != nil {
				t.Fatalf("byte-view action is unreachable at %d DIP: %v", width, err)
			}
			if fake.sourceCalls != 1 {
				t.Fatalf("byte-view click at %d DIP made %d callbacks, want 1", width, fake.sourceCalls)
			}
		})
	}
}

func TestImageKeyboardFocusAndInputContainment(t *testing.T) {
	const tolerance = 1e-9
	fake := &fakeImages{}
	v := newView(fake.host())
	fieldValue := ""
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Gap(8).Children(func() {
			v.Draw(c.Key("keyboard-image"))
			ui.TextInput(c, &fieldValue).Label("隔离文本字段").Height(30)
		})
	}, 760, 480)
	completeCall(t, fake, 0, imageResult(solidRGBA(1000, 800, 255, color.NRGBA{R: 35, G: 120, B: 190})), nil)
	tester.Frame()

	assertZoom := func(name string, want float64, fit bool) {
		t.Helper()
		if math.Abs(v.viewport.zoom-want) > tolerance || v.viewport.fitMode != fit {
			t.Fatalf("%s viewport = %#v, want zoom %g fit=%v", name, v.viewport, want, fit)
		}
	}

	if err := tester.Click("图片查看区域"); err != nil {
		t.Fatal(err)
	}
	if !tester.Focused("图片查看区域") {
		t.Fatal("pointer press did not focus the image canvas")
	}
	tester.TypeKey(0, ui.Key1, "1")
	assertZoom("100% key", 1, false)

	tester.TypeKey(0, ui.KeyEqual, "=")
	assertZoom("equals once", 1.25, false)
	tester.TypeKey(ui.Shift, ui.KeyEqual, "+")
	assertZoom("shift-equals once", 1.5625, false)
	tester.TypeKey(0, ui.KeyMinus, "-")
	assertZoom("minus once", 1.25, false)
	tester.TypeKey(0, ui.Key0, "0")
	if !v.viewport.fitMode {
		t.Fatalf("0 key did not enter fit mode: %#v", v.viewport)
	}
	tester.TypeKey(0, ui.Key1, "1")
	assertZoom("actual-size key", 1, false)

	tester.CodeDown(0, ui.CodeNumpadAdd)
	assertZoom("numpad add once", 1.25, false)
	tester.CodeUp(0, ui.CodeNumpadAdd)
	assertZoom("numpad add key-up ignored", 1.25, false)
	tester.CodeDown(0, ui.CodeNumpadSubtract)
	assertZoom("numpad subtract once", 1, false)
	tester.CodeUp(0, ui.CodeNumpadSubtract)
	assertZoom("numpad subtract key-up ignored", 1, false)

	// A platform event may carry both the layout key and the keypad code.
	// Normalization must still apply exactly one action.
	dual := ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyMinus, Code: ui.CodeNumpadSubtract}
	if !v.handleInput(dual) {
		t.Fatal("dual minus/keypad key was not recognized")
	}
	assertZoom("dual key/code normalized once", 1/1.25, false)

	beforeIgnored := v.viewport
	tester.Press(
		v.canvasRect.X+v.canvasRect.W/2,
		v.canvasRect.Y+v.canvasRect.H/2,
	)
	tester.Key(0, ui.KeyEqual)
	if v.viewport != beforeIgnored {
		t.Fatalf("keyboard zoom changed the viewport during a pointer drag: before=%#v after=%#v", beforeIgnored, v.viewport)
	}
	tester.Release(
		v.canvasRect.X+v.canvasRect.W/2,
		v.canvasRect.Y+v.canvasRect.H/2,
	)

	beforeIgnored = v.viewport
	for _, key := range []ui.Key{ui.Key0, ui.Key1, ui.KeyEqual, ui.KeyMinus, ui.KeyLeft, ui.KeyRight, ui.KeyUp, ui.KeyDown} {
		tester.Key(ui.Ctrl, key)
		tester.Key(ui.Alt, key)
		tester.Key(ui.Super, key)
	}
	tester.Key(ui.Shift, ui.KeyMinus)
	tester.Key(ui.Shift, ui.KeyLeft)
	tester.Type("ignored")
	tester.Compose("组合", 2)
	tester.Command("paste")
	tester.KeyUp(ui.KeyEqual)
	if v.viewport != beforeIgnored {
		t.Fatalf("non-keyboard controls leaked into the canvas: before=%#v after=%#v", beforeIgnored, v.viewport)
	}

	containedViewport := v.viewport
	tester.Key(0, ui.KeyTab)
	if !tester.Focused("隔离文本字段") {
		t.Fatalf("Tab did not leave the image canvas for the text field: %#v", tester.Texts())
	}
	if active, noIME := tester.TextInputActive(); !active || noIME {
		t.Fatalf("text field input state = active:%v noIME:%v", active, noIME)
	}
	tester.TypeKey(0, ui.KeyEqual, "=")
	tester.Type("a")
	tester.Compose("草稿", 2)
	tester.Type("草稿")
	tester.Key(ui.Ctrl, ui.Key0)
	tester.Key(ui.Shift, ui.KeyDown)
	if fieldValue == "" || !strings.Contains(fieldValue, "=") || !strings.Contains(fieldValue, "草稿") {
		t.Fatalf("text/IME input did not stay in the field: %q", fieldValue)
	}
	if v.viewport != containedViewport {
		t.Fatalf("text-field input changed the complete image viewport: before=%#v after=%#v",
			containedViewport, v.viewport)
	}

	if err := tester.Click("图片查看区域"); err != nil {
		t.Fatal(err)
	}
	beforeBlur := v.viewport
	tester.SetFocused(false)
	tester.Key(0, ui.KeyEqual)
	if v.viewport != beforeBlur {
		t.Fatalf("blurred window delivered a canvas zoom: before=%#v after=%#v", beforeBlur, v.viewport)
	}
	tester.SetFocused(true)
	tester.Key(0, ui.KeyEqual)
	if v.viewport.zoom <= beforeBlur.zoom {
		t.Fatalf("restored window did not deliver the canvas key: before=%#v after=%#v", beforeBlur, v.viewport)
	}

	for _, label := range []string{"按百分之百显示", "减小图片显示", "增大图片显示", "适合窗口", "按百分之百显示"} {
		if err := tester.Click(label); err != nil {
			t.Fatalf("click %q: %v", label, err)
		}
		if !tester.Focused("图片查看区域") {
			t.Fatalf("enabled toolbar action %q did not return focus to the canvas", label)
		}
	}
	beforeToolbarKeyboard := v.viewport
	tester.Key(0, ui.KeyEqual)
	if v.viewport.zoom <= beforeToolbarKeyboard.zoom {
		t.Fatalf("keyboard after enabled toolbar focus return did not zoom: before=%#v after=%#v",
			beforeToolbarKeyboard, v.viewport)
	}

	for _, activation := range []ui.Key{ui.KeyEnter, ui.KeySpace} {
		tester.Key(ui.Shift, ui.KeyTab)
		tester.Key(ui.Shift, ui.KeyTab)
		if !tester.Focused("增大图片显示") {
			t.Fatalf("Shift+Tab did not reach the zoom-in button before %v: %#v", activation, tester.Texts())
		}
		before := v.viewport
		tester.Key(0, activation)
		if v.viewport.zoom <= before.zoom {
			t.Fatalf("%v on the zoom-in button did not run its enabled action: before=%#v after=%#v", activation, before, v.viewport)
		}
		if !tester.Focused("图片查看区域") {
			t.Fatalf("%v on the zoom-in button did not return focus to the canvas", activation)
		}
	}
	tester.Key(0, ui.KeyTab)
	if !tester.Focused("隔离文本字段") || tester.Focused("图片查看区域") {
		t.Fatal("Tab did not exit the toolbar-returned canvas focus")
	}
}

func TestImageKeyboardGeometryThroughPublicEvents(t *testing.T) {
	const tolerance = 1e-6
	closeEnough := func(a, b float64) bool { return math.Abs(a-b) <= tolerance }

	t.Run("manual limits pan edges and center anchor", func(t *testing.T) {
		fake := &fakeImages{}
		v := newView(fake.host())
		tester := ui.NewTester(v.Draw, 760, 480)
		completeCall(t, fake, 0, imageResult(solidRGBA(1200, 800, 255, color.NRGBA{R: 20, G: 100, B: 170})), nil)
		tester.Frame()
		if err := tester.Click("按百分之百显示"); err != nil {
			t.Fatal(err)
		}

		width, height := float64(v.canvasRect.W), float64(v.canvasRect.H)
		centerX := (width/2 - v.viewport.panX) / v.viewport.zoom
		centerY := (height/2 - v.viewport.panY) / v.viewport.zoom
		tester.Key(0, ui.KeyEqual)
		gotX := (width/2 - v.viewport.panX) / v.viewport.zoom
		gotY := (height/2 - v.viewport.panY) / v.viewport.zoom
		if !closeEnough(gotX, centerX) || !closeEnough(gotY, centerY) {
			t.Fatalf("keyboard zoom moved the center image point from %g,%g to %g,%g", centerX, centerY, gotX, gotY)
		}

		for range 32 {
			tester.Key(0, ui.KeyMinus)
		}
		if !closeEnough(v.viewport.zoom, MinZoom) || v.viewport.fitMode {
			t.Fatalf("repeated minus did not stop at the manual floor: %#v", v.viewport)
		}
		for range 32 {
			tester.Key(0, ui.KeyEqual)
		}
		if !closeEnough(v.viewport.zoom, MaxZoom) || v.viewport.fitMode {
			t.Fatalf("repeated equals did not stop at the manual ceiling: %#v", v.viewport)
		}

		for i := 0; i < 400 && v.viewport.panX < -tolerance; i++ {
			tester.Key(0, ui.KeyLeft)
		}
		for i := 0; i < 400 && v.viewport.panY < -tolerance; i++ {
			tester.Key(0, ui.KeyUp)
		}
		if !closeEnough(v.viewport.panX, 0) || !closeEnough(v.viewport.panY, 0) {
			t.Fatalf("left/up did not reach the top-left image edge: %#v", v.viewport)
		}

		minX := width - float64(v.imageWidth)*v.viewport.zoom
		minY := height - float64(v.imageHeight)*v.viewport.zoom
		for i := 0; i < 400 && v.viewport.panX > minX+tolerance; i++ {
			tester.Key(0, ui.KeyRight)
		}
		for i := 0; i < 400 && v.viewport.panY > minY+tolerance; i++ {
			tester.Key(0, ui.KeyDown)
		}
		if !closeEnough(v.viewport.panX, minX) || !closeEnough(v.viewport.panY, minY) {
			t.Fatalf("right/down did not reach the bottom-right image edge: got %#v, want pan %.9f,%.9f", v.viewport, minX, minY)
		}
		if !finite(v.viewport.zoom) || !finite(v.viewport.panX) || !finite(v.viewport.panY) ||
			v.viewport.panX > tolerance || v.viewport.panX < minX-tolerance ||
			v.viewport.panY > tolerance || v.viewport.panY < minY-tolerance {
			t.Fatalf("keyboard navigation escaped finite bounds: %#v", v.viewport)
		}

		tester.Key(0, ui.Key0)
		if !v.viewport.fitMode || v.viewport.zoom >= 1 {
			t.Fatalf("0 key did not restore a fitted large image: %#v", v.viewport)
		}
		tester.Key(0, ui.Key1)
		if v.viewport.fitMode || !closeEnough(v.viewport.zoom, 1) {
			t.Fatalf("1 key did not restore actual size: %#v", v.viewport)
		}
		panBefore := v.viewport.panX
		tester.Key(0, ui.KeyRight)
		if closeEnough(v.viewport.panX, panBefore) {
			t.Fatalf("arrow did not pan a cropped image: before=%g after=%g", panBefore, v.viewport.panX)
		}
	})

	t.Run("sub-floor fit stays fitted on no-op zoom and pan", func(t *testing.T) {
		fake := &fakeImages{}
		v := newView(fake.host())
		// 8192 is the admitted raster side limit. The short tester viewport
		// leaves the fitted scale below 2% without injecting a wider font-size
		// image into the public route.
		tester := ui.NewTester(v.Draw, 760, 200)
		completeCall(t, fake, 0, imageResult(solidRGBA(1, 8192, 255, color.NRGBA{R: 120, G: 80, B: 30})), nil)
		tester.Frame()
		if err := tester.Click("图片查看区域"); err != nil {
			t.Fatal(err)
		}
		tester.Key(0, ui.Key0)
		if !v.viewport.fitMode || v.viewport.zoom >= MinZoom {
			t.Fatalf("tall-image fit is not below the manual floor: %#v", v.viewport)
		}
		before := v.viewport
		tester.Key(0, ui.KeyMinus)
		if v.viewport != before {
			t.Fatalf("minus snapped a sub-floor fit into a cropped manual view: before=%#v after=%#v", before, v.viewport)
		}
		tester.Key(0, ui.KeyLeft)
		if v.viewport != before {
			t.Fatalf("an arrow moved a centered fitted tall image: before=%#v after=%#v", before, v.viewport)
		}
		tester.Key(0, ui.KeyEqual)
		if v.viewport.fitMode || !closeEnough(v.viewport.zoom, MinZoom) {
			t.Fatalf("first plus did not enter the manual floor: %#v", v.viewport)
		}
	})
}

func TestImageKeyboardReleasedViewHasNoRoutes(t *testing.T) {
	fake := &fakeImages{}
	v := newView(fake.host())
	draw := true
	tester := ui.NewTester(func(c *ui.Context) {
		if draw {
			v.Draw(c)
		} else {
			ui.Box(c).Key("released-image-placeholder")
		}
	}, 760, 480)
	completeCall(t, fake, 0, imageResult(solidRGBA(320, 180, 255, color.NRGBA{R: 70, G: 40, B: 100})), nil)
	tester.Frame()
	if err := tester.Click("图片查看区域"); err != nil {
		t.Fatal(err)
	}
	before := v.viewport
	requestCount := len(fake.calls)
	v.canvas.Focus()
	v.Release()
	draw = false
	tester.Frame()

	for _, key := range []ui.Key{ui.Key0, ui.Key1, ui.KeyEqual, ui.KeyMinus, ui.KeyLeft, ui.KeyRight, ui.KeyUp, ui.KeyDown} {
		tester.Key(0, key)
	}
	tester.CodeDown(0, ui.CodeNumpadAdd)
	tester.CodeUp(0, ui.CodeNumpadAdd)
	tester.Type("=")
	tester.Compose("组合", 2)
	tester.Command("paste")

	if v.viewport != before {
		t.Fatalf("released viewport changed after input: before=%#v after=%#v", before, v.viewport)
	}
	if len(fake.calls) != requestCount {
		t.Fatalf("released view requested decoding again: calls=%d/%d", len(fake.calls), requestCount)
	}
	if v.bitmap != nil || v.imageWidth != 0 || v.imageHeight != 0 || !v.released {
		t.Fatalf("released view retained pixels or accepted a new activation: %#v", v)
	}
	if tester.Focused("图片查看区域") {
		t.Fatal("released view retained or regained keyboard focus")
	}
}

func TestErrorSourceRefusalAndPixelsStayExplicit(t *testing.T) {
	fake := &fakeImages{sourceOK: false}
	v := newView(fake.host())
	tester := ui.NewTester(v.Draw, 760, 480)
	decodeErr := errors.New("JPEG 数据截断")
	completeCall(t, fake, 0, editor.ImageResult{}, decodeErr)
	tester.Frame()
	if v.bitmap != nil || !tester.HasText("图片无法显示") || !strings.Contains(strings.Join(v.Warnings(), " "), decodeErr.Error()) {
		t.Fatalf("decode error not explicit: bitmap=%v texts=%#v warnings=%v", v.bitmap, tester.Texts(), v.Warnings())
	}
	if len(fake.calls) != 1 {
		t.Fatal("decode error retried without a new activation")
	}

	// A failed decode never closes byte inspection. The source refusal remains
	// visible and does not erase the view or grant retry authority.
	if err := tester.Click("查看只读字节"); err != nil {
		t.Fatal(err)
	}
	if fake.sourceCalls != 1 || !tester.HasText("只读字节视图未打开") {
		t.Fatalf("source refusal hidden: calls=%d texts=%#v", fake.sourceCalls, tester.Texts())
	}
	if v.released || len(fake.calls) != 1 {
		t.Fatal("source refusal dropped or replaced the current view")
	}

	fake2 := &fakeImages{sourceOK: true}
	v2 := newView(fake2.host())
	tester2 := ui.NewTester(v2.Draw, 760, 480)
	completeCall(t, fake2, 0, imageResult(solidRGBA(12, 8, 255, color.NRGBA{R: 100, G: 50, B: 20})), nil)
	tester2.Frame()
	before := v2.bitmap
	if err := tester2.Click("查看只读字节"); err != nil {
		t.Fatal(err)
	}
	if fake2.sourceCalls != 1 || v2.bitmap != before || v2.sourceNotice != nil {
		t.Fatalf("successful source callback changed image/view: calls=%d bitmap=%p/%p notice=%v", fake2.sourceCalls, before, v2.bitmap, v2.sourceNotice)
	}
}

func TestBeginFailureAndSynchronousCompletionRemainVisible(t *testing.T) {
	failed := &fakeImages{beginErr: errors.New("二进制权威已变化")}
	v := newView(failed.host())
	tester := ui.NewTester(v.Draw, 500, 320)
	if len(failed.calls) != 1 || !tester.HasText("二进制权威已变化") {
		t.Fatalf("BeginImage failure not explicit: calls=%d texts=%#v", len(failed.calls), tester.Texts())
	}
	if _, err := v.Export("png"); err == nil {
		t.Fatal("readonly image view silently exported")
	}
	if warnings := v.Warnings(); len(warnings) == 0 {
		t.Fatal("BeginImage failure absent from warnings")
	}
	tester.Frame()
	if len(failed.calls) != 1 {
		t.Fatal("failed request retried inside one activation")
	}

	calls, canceled := 0, 0
	syncHost := editor.Host{}
	syncHost.BeginImage = func(owner editor.View, request editor.ImageRequest, complete func(editor.ImageResult, error)) (func(), error) {
		calls++
		if owner == nil {
			t.Fatal("view did not identify its request owner")
		}
		complete(imageResult(solidRGBA(9, 7, 255, color.NRGBA{R: 8, G: 9, B: 10})), nil)
		return func() { canceled++ }, nil
	}
	v2 := newView(syncHost)
	tester2 := ui.NewTester(v2.Draw, 500, 320)
	if calls != 1 || canceled != 1 || v2.bitmap == nil {
		t.Fatalf("synchronous completion handling: calls=%d canceled=%d bitmap=%v", calls, canceled, v2.bitmap)
	}
	if !tester2.HasText("适合窗口") {
		t.Fatalf("synchronous completion did not render toolbar: %#v", tester2.Texts())
	}
}

func TestStaleCompletionAfterReleaseCannotReviveOrReplace(t *testing.T) {
	fake := &fakeImages{}
	v := newView(fake.host())
	tester := ui.NewTester(v.Draw, 760, 480)
	if len(fake.calls) != 1 {
		t.Fatalf("request count = %d", len(fake.calls))
	}
	firstEpoch := fake.calls[0].request.Epoch

	v.Release()
	if v.epoch <= firstEpoch || !fake.calls[0].wasCanceled || fake.cancelCount != 1 {
		t.Fatalf("Release did not advance/cancel: epoch=%d first=%d canceled=%v count=%d", v.epoch, firstEpoch, fake.calls[0].wasCanceled, fake.cancelCount)
	}
	v.Release()
	if fake.cancelCount != 1 {
		t.Fatalf("repeated Release canceled twice: %d", fake.cancelCount)
	}

	// Reusing the same view starts exactly one new activation.
	tester.Frame()
	if len(fake.calls) != 2 {
		t.Fatalf("reuse request count = %d, want 2", len(fake.calls))
	}
	if fake.calls[1].request.Epoch <= firstEpoch {
		t.Fatalf("new activation epoch = %d, want > %d", fake.calls[1].request.Epoch, firstEpoch)
	}
	stalePixels := solidRGBA(40, 20, 255, color.NRGBA{R: 220, G: 20, B: 20})
	completeCall(t, fake, 0, imageResult(stalePixels), nil)
	if v.bitmap != nil || !v.loading || v.err != nil {
		t.Fatalf("stale A completion changed active B state: bitmap=%v loading=%v err=%v", v.bitmap, v.loading, v.err)
	}

	freshPixels := solidRGBA(31, 17, 255, color.NRGBA{R: 20, G: 40, B: 220})
	freshResult := imageResult(freshPixels)
	freshResult.Format = "JPEG"
	completeCall(t, fake, 1, freshResult, nil)
	tester.Frame()
	if v.bitmap == nil || v.format != "JPEG" || v.imageWidth != 31 || v.imageHeight != 17 {
		t.Fatalf("fresh B result not installed: bitmap=%v format=%q size=%dx%d", v.bitmap, v.format, v.imageWidth, v.imageHeight)
	}
	if len(fake.calls) != 2 {
		t.Fatalf("completion caused another request: %d", len(fake.calls))
	}
	bitmap := v.bitmap
	v.Release()
	if w, h := bitmap.Size(); w != 0 || h != 0 {
		t.Fatalf("released bitmap size = %dx%d, want 0x0", w, h)
	}
	if v.bitmap != nil || v.imageWidth != 0 || v.imageHeight != 0 || v.format != "" || v.cancel != nil || v.completionOpen {
		t.Fatal("Release retained bitmap, metadata, or completion closure state")
	}
}

func TestMalformedResultAndDisposeFailureAreExplicit(t *testing.T) {
	fake := &fakeImages{}
	v := newView(fake.host())
	tester := ui.NewTester(v.Draw, 500, 320)
	bad := imageResult(solidRGBA(10, 10, 255, color.NRGBA{R: 1, G: 2, B: 3}))
	bad.Width = 11
	completeCall(t, fake, 0, bad, nil)
	tester.Frame()
	if v.bitmap != nil || !strings.Contains(strings.Join(v.Warnings(), " "), "方向后尺寸") {
		t.Fatalf("malformed result accepted or silent: bitmap=%v warnings=%v", v.bitmap, v.Warnings())
	}

	fake2 := &fakeImages{}
	v2 := newView(fake2.host())
	tester2 := ui.NewTester(v2.Draw, 500, 320)
	completeCall(t, fake2, 0, imageResult(solidRGBA(10, 10, 255, color.NRGBA{R: 1, G: 2, B: 3})), nil)
	tester2.Frame()
	v2.disposeBitmap = func(*ui.Bitmap) bool { return false }
	v2.Release()
	if v2.err == nil || len(fake2.reports) != 1 || !strings.Contains(strings.Join(v2.Warnings(), " "), "位图释放失败") {
		t.Fatalf("Dispose false was hidden: err=%v reports=%v warnings=%v", v2.err, fake2.reports, v2.Warnings())
	}
}

func TestOrientedJPEGAndTransparentPalettePNGDecodeIntegration(t *testing.T) {
	jpegFixture := orientedJPEGFixture(t, 6)
	jpegResult := decodeForView(t, jpegFixture)
	fake := &fakeImages{}
	v := newView(fake.host())
	tester := ui.NewTester(v.Draw, 760, 480)
	completeCall(t, fake, 0, editor.ImageResult{
		Format:          "JPEG",
		Width:           jpegResult.Width,
		Height:          jpegResult.Height,
		SourceWidth:     jpegResult.SourceWidth,
		SourceHeight:    jpegResult.SourceHeight,
		Orientation:     jpegResult.Orientation,
		CompressedBytes: int64(len(jpegFixture)),
		Pixels:          jpegResult.Pixels,
	}, nil)
	tester.Frame()
	if v.bitmap == nil || v.imageWidth != 16 || v.imageHeight != 32 ||
		v.sourceWidth != 32 || v.sourceHeight != 16 || v.orientation != 6 {
		t.Fatalf("oriented JPEG metadata = bitmap:%v final:%dx%d source:%dx%d orientation:%d", v.bitmap, v.imageWidth, v.imageHeight, v.sourceWidth, v.sourceHeight, v.orientation)
	}
	if w, h := v.bitmap.Size(); w != 16 || h != 32 {
		t.Fatalf("oriented JPEG bitmap = %dx%d", w, h)
	}

	paletteFixture := transparentPalettePNG(t, 96, 64)
	paletteResult := decodeForView(t, paletteFixture)
	fake2 := &fakeImages{}
	v2 := newView(fake2.host())
	tester2 := ui.NewTester(v2.Draw, 760, 480)
	completeCall(t, fake2, 0, editor.ImageResult{
		Format:          "PNG",
		Width:           paletteResult.Width,
		Height:          paletteResult.Height,
		SourceWidth:     paletteResult.SourceWidth,
		SourceHeight:    paletteResult.SourceHeight,
		Orientation:     paletteResult.Orientation,
		CompressedBytes: int64(len(paletteFixture)),
		Pixels:          paletteResult.Pixels,
	}, nil)
	tester2.Frame()
	if v2.bitmap == nil || v2.imageWidth != 96 || v2.imageHeight != 64 {
		t.Fatalf("transparent palette PNG = %v %dx%d", v2.bitmap, v2.imageWidth, v2.imageHeight)
	}
	capture := tester2.Image()
	if capture == nil || capture.Rect.Dx() != 760 || capture.Rect.Dy() != 480 {
		t.Fatalf("Tester image = %v", capture)
	}
}

func TestSyntheticTesterCaptures(t *testing.T) {
	directory := os.Getenv("LEVNOTE_IMAGEVIEW_CAPTURE_DIR")
	if directory == "" {
		t.Skip("set LEVNOTE_IMAGEVIEW_CAPTURE_DIR to write the two synthetic captures")
	}
	suffix := strings.TrimSpace(os.Getenv("LEVNOTE_IMAGEVIEW_CAPTURE_SUFFIX"))
	if suffix != "" && !strings.HasPrefix(suffix, "-") {
		suffix = "-" + suffix
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	fixture := transparentPalettePNG(t, 600, 360)
	for _, theme := range []struct {
		name string
		dark bool
	}{
		{name: "light", dark: false},
		{name: "dark", dark: true},
	} {
		decoded := decodeForView(t, fixture)
		fake := &fakeImages{}
		v := newView(fake.host())
		tester := ui.NewTesterWith(v.Draw, 760, 480, ui.TesterOptions{Dark: theme.dark})
		completeCall(t, fake, 0, editor.ImageResult{
			Format:          "PNG",
			Width:           decoded.Width,
			Height:          decoded.Height,
			SourceWidth:     decoded.SourceWidth,
			SourceHeight:    decoded.SourceHeight,
			Orientation:     decoded.Orientation,
			CompressedBytes: int64(len(fixture)),
			Pixels:          decoded.Pixels,
		}, nil)
		tester.Frame()
		captured := tester.Image()
		path := filepath.Join(directory, "imageview-alpha-toolbar-"+theme.name+"-760x480"+suffix+".png")
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		encodeErr := png.Encode(file, captured)
		closeErr := file.Close()
		if encodeErr != nil || closeErr != nil {
			t.Fatalf("write %s: encode=%v close=%v", path, encodeErr, closeErr)
		}
		v.Release()
	}
}

func decodeForView(t *testing.T, fixture []byte) raster.DecodeResult {
	t.Helper()
	result, err := raster.Decode(context.Background(), bytes.NewReader(fixture), int64(len(fixture)))
	if err != nil {
		t.Fatalf("raster.Decode fixture: %v", err)
	}
	if result.Pixels == nil {
		t.Fatal("raster.Decode returned no pixels")
	}
	return result
}

func orientedJPEGFixture(t *testing.T, orientation uint16) []byte {
	t.Helper()
	source := image.NewNRGBA(image.Rect(0, 0, 32, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 32; x++ {
			value := color.NRGBA{R: 220, G: 30, B: 30, A: 255}
			if x >= 16 {
				value = color.NRGBA{R: 30, G: 40, B: 220, A: 255}
			}
			source.SetNRGBA(x, y, value)
		}
	}
	var base bytes.Buffer
	if err := jpeg.Encode(&base, source, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	tiff := make([]byte, 26)
	copy(tiff, "II")
	binary.LittleEndian.PutUint16(tiff[2:], 42)
	binary.LittleEndian.PutUint32(tiff[4:], 8)
	binary.LittleEndian.PutUint16(tiff[8:], 1)
	binary.LittleEndian.PutUint16(tiff[10:], 0x0112)
	binary.LittleEndian.PutUint16(tiff[12:], 3)
	binary.LittleEndian.PutUint32(tiff[14:], 1)
	binary.LittleEndian.PutUint16(tiff[18:], orientation)
	payload := append([]byte("Exif\x00\x00"), tiff...)
	segmentLength := len(payload) + 2
	fixture := append([]byte(nil), base.Bytes()[:2]...)
	fixture = append(fixture, 0xff, 0xe1, byte(segmentLength>>8), byte(segmentLength))
	fixture = append(fixture, payload...)
	fixture = append(fixture, base.Bytes()[2:]...)
	return fixture
}

func transparentPalettePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	palette := color.Palette{
		color.NRGBA{R: 0, G: 0, B: 0, A: 0},
		color.NRGBA{R: 32, G: 160, B: 210, A: 255},
		color.NRGBA{R: 240, G: 196, B: 64, A: 255},
	}
	pixels := image.NewPaletted(image.Rect(0, 0, width, height), palette)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			index := uint8(0)
			if (x/40+y/40)%2 == 0 {
				index = 1
			}
			if x > width/3 && x < width*2/3 && y > height/4 && y < height*3/4 {
				index = 0
			}
			if (x+y)%97 == 0 {
				index = 2
			}
			pixels.SetColorIndex(x, y, index)
		}
	}
	var output bytes.Buffer
	if err := png.Encode(&output, pixels); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
