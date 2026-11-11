package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/egoist/mygo/internal/scene"
)

type noConversion struct{}

func (noConversion) Bounds() image.Rectangle { panic("disposed Update converted input") }
func (noConversion) ColorModel() color.Model { panic("disposed Update converted input") }
func (noConversion) At(int, int) color.Color { panic("disposed Update converted input") }

func TestBitmapDisposeOwnsBaseMipsAndDecoder(t *testing.T) {
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 16, 16))); err != nil {
		t.Fatal(err)
	}
	b, err := DecodeBitmap(data.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	b.smaller(8)
	base, levels := b.img, append([]*scene.Image(nil), b.levels...)
	b.ReleasePixels()
	if b.remake == nil || base.Restore == nil || len(levels) != 3 {
		t.Fatal("missing restoration setup")
	}
	// If any stale callback is invoked, fail before a real decode can occur.
	b.remake = func() (*scene.Image, error) { t.Fatal("decoded after Dispose"); return nil, nil }
	if !b.Dispose() || !b.Dispose() || !(*Bitmap)(nil).Dispose() {
		t.Fatal("Dispose failed")
	}
	b.Update(noConversion{})
	b.ReleasePixels()
	if b.Pixelated() != b || b.pixelated || (*Bitmap)(nil).Pixelated() != nil {
		t.Fatal("Pixelated revived bitmap")
	}
	if w, h := b.Size(); w != 0 || h != 0 {
		t.Fatal("disposed size")
	}
	if b.img != nil || b.levels != nil || b.remake != nil || b.source != nil || b.released || b.fresh != 0 || b.smaller(8) != nil {
		t.Fatal("bitmap still owns source data")
	}
	for _, img := range append(levels, base) {
		img.Uploaded()
		if !img.Retired() || img.Pixels() != nil || img.Restore != nil || img.Dropped != nil {
			t.Fatal("stale image restored")
		}
	}
	p := &Painter{} // safe before any painter/window state is needed
	p.drawBitmap(b, Rect{W: 5, H: 5}, 0, [2]float32{}, [4]float32{}, false)
	p.ImageRegion(b, Rect{}, Rect{W: 5, H: 5}, ImageOptions{})
	in := ShaderInput{Images: [4]*Bitmap{b}}
	if in.op(nil).Images != [4]*scene.Image{} {
		t.Fatal("disposed shader input")
	}
}

func TestBitmapUpdateRetiresOnlyReplacedSize(t *testing.T) {
	b := NewBitmap(image.NewRGBA(image.Rect(0, 0, 8, 8)))
	base, mip := b.img, b.smaller(2)
	b.Update(image.NewRGBA(image.Rect(0, 0, 8, 8)))
	if b.img != base || base.Retired() || b.smaller(2) != mip || mip.Retired() {
		t.Fatal("same-size update lost identities")
	}
	b.Update(image.NewRGBA(image.Rect(0, 0, 9, 7)))
	if !base.Retired() || !mip.Retired() || base.Pixels() != nil || mip.Pixels() != nil || b.img == base || b.img.Retired() || len(b.levels) != 0 {
		t.Fatal("size replacement kept old pixels")
	}
	b.Dispose()
}

func TestBitmapCanvasBorrowedDisposeRefused(t *testing.T) {
	c := NewCanvas(8, 8)
	for _, b := range []*Bitmap{c.Bitmap(), c.region(4, 4)} {
		img, w, h := b.img, b.w, b.h
		if b.Dispose() || b.disposed || b.img != img || b.w != w || b.h != h || img.Retired() {
			t.Fatal("borrow changed")
		}
		b.Update(noConversion{})
		b.ReleasePixels()
	}
	c.resize(3, 4)
	if c.Bitmap().img != c.img || c.img.Retired() {
		t.Fatal("canvas unusable after rejected Dispose")
	}
}
