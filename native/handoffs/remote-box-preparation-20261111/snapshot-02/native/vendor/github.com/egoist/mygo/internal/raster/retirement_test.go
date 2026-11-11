package raster

import (
	"testing"

	"github.com/egoist/mygo/internal/scene"
)

func retirementPicture() *scene.Image {
	pix := make([]byte, 4*8*8)
	for i := 0; i < len(pix); i += 4 {
		pix[i], pix[i+3] = 255, 255
	}
	return scene.NewImageRGBA(8, 8, pix)
}

func retirementOp(img *scene.Image) scene.Op {
	return scene.Op{Kind: scene.OpImage, Image: img, Rect: scene.Rect{X: 2, Y: 2, W: 8, H: 8}, Src: scene.Rect{W: float32(img.W), H: float32(img.H)}}
}

func hasRetirementPixels(pix []byte) bool {
	for _, b := range pix {
		if b != 0 {
			return true
		}
	}
	return false
}

func TestRetirementStaleScenesAndIndependentRenderers(t *testing.T) {
	for _, kind := range []string{"direct", "transform", "blur", "nested-canvas", "effect-canvas", "clip-canvas"} {
		t.Run(kind, func(t *testing.T) {
			img := retirementPicture()
			op := retirementOp(img)
			s := scene.Scene{Width: 16, Height: 16, Scale: 1}
			var inner, outer *scene.Image
			switch kind {
			case "transform":
				op.Transform = scene.Affine{Set: true, A: 1, B: .12, C: .1, D: 1}
			case "blur":
				op.Blur = 1
			case "nested-canvas", "effect-canvas", "clip-canvas":
				inner, outer = scene.NewCanvas(12, 12), scene.NewCanvas(12, 12)
				inner.Scene.Ops = []scene.Op{op}
				outer.Scene.Ops = []scene.Op{retirementOp(inner)}
				op = retirementOp(outer)
			}
			if kind == "effect-canvas" {
				fx := &scene.Effect{Pixels: func() scene.EffectPixels { return &retirementEffect{} }}
				s.Effects = []scene.EffectOp{{Effect: fx, Images: [4]*scene.Image{outer}}}
				s.Ops = []scene.Op{{Kind: scene.OpEffect, Rect: scene.Rect{W: 16, H: 16}}}
			} else if kind == "clip-canvas" {
				s.Ops = []scene.Op{
					{Kind: scene.OpPushClip, Rect: scene.Rect{W: 16, H: 16}, Image: outer, Src: scene.Rect{W: 12, H: 12}},
					{Kind: scene.OpFill, Rect: scene.Rect{W: 16, H: 16}, Color: scene.Color{R: 255, A: 255}},
					{Kind: scene.OpPopClip},
				}
			} else {
				s.Ops = []scene.Op{op}
			}
			var a, b Renderer
			defer a.Release()
			defer b.Release()
			for _, r := range []*Renderer{&a, &b} {
				r.Render(&s)
				if !hasRetirementPixels(r.Image.Pix) {
					t.Fatal("fixture drew no image")
				}
				if kind != "effect-canvas" && len(r.Render(&s)) != 0 {
					t.Fatal("fixture not unchanged before retirement")
				}
			}
			img.Retire()
			img.Restore = func() []byte { t.Fatal("retired pixels restored"); return nil }
			for _, r := range []*Renderer{&a, &b} {
				if len(r.Render(&s)) == 0 || hasRetirementPixels(r.Image.Pix) {
					t.Fatal("retired stale scene was not cleared")
				}
			}
			if inner != nil && (hasRetirementPixels(inner.Pix) || hasRetirementPixels(outer.Pix)) {
				t.Fatal("nested derived pixels survived")
			}
			a.Release()
			a.Render(&s)
			if hasRetirementPixels(a.Image.Pix) {
				t.Fatal("recreated renderer revived image")
			}
		})
	}
}

type retirementEffect struct{ pixels []byte }

func (e *retirementEffect) Begin(op *scene.EffectOp, _ scene.Rect, _ [4]float32) {
	e.pixels = op.Images[0].Pixels()
}
func (e *retirementEffect) Color(_, _ float32, _ *scene.BackdropImage) [4]float32 {
	if hasRetirementPixels(e.pixels) {
		return [4]float32{1, 0, 0, 1}
	}
	return [4]float32{}
}
