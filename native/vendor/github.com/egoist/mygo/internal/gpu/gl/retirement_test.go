//go:build linux && (amd64 || arm64)

package gl

import (
	"runtime"
	"testing"
	"weak"

	"github.com/egoist/mygo/internal/scene"
)

func TestRetirementOnlyDeletesOwningContext(t *testing.T) {
	oldContext, oldDelete, oldFB := currentGDKContext, glDeleteTextures, glDeleteFramebuffers
	defer func() { currentGDKContext, glDeleteTextures, glDeleteFramebuffers = oldContext, oldDelete, oldFB }()
	current := uintptr(2)
	currentGDKContext = func() uintptr { return current }
	var deleted []uint32
	glDeleteTextures = func(_ int32, p *uint32) { deleted = append(deleted, *p) }
	glDeleteFramebuffers = func(_ int32, p *uint32) { deleted = append(deleted, *p) }
	img, live := scene.NewImageRGBA(1, 1, nil), scene.NewImageRGBA(1, 1, nil)
	r := Renderer{context: 1, held: true, images: map[uint64]*texture{img.ID(): {tex: 7, image: weak.Make(img)}, live.ID(): {tex: 8, image: weak.Make(live)}},
		canvases: map[uint64]*canvasTexture{99: {texture: texture{tex: 9}, fb: 10}}, canvasTex: map[uint32]bool{9: true}, mask: texture{tex: 11}, stops: texture{tex: 12}, grab: texture{tex: 13}}
	r.retirement.Check(nil, func() {})
	img.Retire()
	if r.retireImages() || len(deleted) != 0 {
		t.Fatal("deleted on wrong context")
	}
	current = 1
	if !r.retireImages() || r.retireImages() || len(deleted) != 4 || len(r.images) != 1 || len(r.canvases) != 0 || len(r.canvasTex) != 0 || r.held || r.mask.tex != 11 || r.stops.tex != 12 {
		t.Fatal("incorrect retirement")
	}
	if r.imageTexture(img) != 0 {
		t.Fatal("retired cached hit")
	}
	current = 2
	r.Release()
	if len(deleted) != 4 || len(r.images) != 0 {
		t.Fatal("wrong-context teardown deleted handles")
	}
	runtime.KeepAlive(live)
}
