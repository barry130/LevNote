package gpu

import (
	"runtime"
	"testing"
	"weak"

	"github.com/egoist/mygo/internal/scene"
)

func TestIndependentRetirementAndReleaseOrdering(t *testing.T) {
	type entry struct {
		owner    weak.Pointer[scene.Image]
		releases int
	}
	dead, live := scene.NewImageRGBA(1, 1, nil), scene.NewImageRGBA(1, 1, nil)
	a, b := &entry{owner: weak.Make(dead)}, &entry{owner: weak.Make(dead)}
	l := &entry{owner: weak.Make(live)}
	maps := []map[uint64]*entry{{dead.ID(): a, live.ID(): l}, {dead.ID(): b}}
	var cursors [2]ImageRetirement
	for i := range cursors {
		cursors[i].Check(nil, func() {})
	}
	dead.Retire()
	for i := range cursors {
		pending := true
		purge := func() {
			if pending {
				t.Fatal("released before pending batches/bindings dropped")
			}
			SweepRetired(maps[i], func(e *entry) weak.Pointer[scene.Image] { return e.owner }, func(e *entry) { e.releases++ })
		}
		if !cursors[i].Check(func() { pending = false }, purge) || cursors[i].Check(nil, purge) {
			t.Fatal("renderer missed/repeated retirement")
		}
		SweepRetired(maps[i], func(e *entry) weak.Pointer[scene.Image] { return e.owner }, func(e *entry) { e.releases++ })
	}
	if a.releases != 1 || b.releases != 1 || l.releases != 0 || len(maps[0]) != 1 || len(maps[1]) != 0 {
		t.Fatal("cache ownership incorrect")
	}
	// A dead weak owner is also eligible, without any GC timing assumption.
	maps[0][99] = &entry{}
	SweepRetired(maps[0], func(e *entry) weak.Pointer[scene.Image] { return e.owner }, func(e *entry) { e.releases++ })
	if len(maps[0]) != 1 {
		t.Fatal("unreachable entry retained")
	}
	runtime.KeepAlive(live)
}

func TestForgetImagesDropsDerivedMasksAndKeepsOptions(t *testing.T) {
	m := scene.NewImageRGBA(1, 1, make([]byte, 4))
	b := Builder{Classes: true, Opaque: true, NoFold: true, GeneralClasses: 7,
		maskSlots: []clipMaskSlot{{image: m}}, Batches: []Batch{{Image: 42}}}
	var dropped []uint64
	b.ForgetImages(func(id uint64) { dropped = append(dropped, id) })
	b.ForgetImages(func(id uint64) { dropped = append(dropped, id) })
	if len(dropped) != 1 || dropped[0] != m.ID() || b.maskSlots != nil || b.Batches != nil || !b.Classes || !b.Opaque || !b.NoFold || b.GeneralClasses != 7 {
		t.Fatal("lost options or retained/double-freed masks")
	}
}

func TestRetiredImageAndShaderInputsDoNotRequestTextures(t *testing.T) {
	img := scene.NewImageRGBA(1, 1, []byte{255, 0, 0, 255})
	img.Retire()
	f := &scene.Effect{Shader: &scene.ShaderLayout{}}
	s := scene.Scene{Width: 8, Height: 8, Ops: []scene.Op{
		{Kind: scene.OpImage, Image: img, Rect: scene.Rect{W: 8, H: 8}, Src: scene.Rect{W: 1, H: 1}, Blur: 2},
		{Kind: scene.OpEffect, Rect: scene.Rect{W: 8, H: 8}},
	}, Effects: []scene.EffectOp{{Effect: f, Images: [4]*scene.Image{img}}}}
	b := Builder{BlurredImage: func(*scene.Image, scene.Backdrop) uintptr { t.Fatal("blurred retired input"); return 0 }}
	b.Build(&s, func(*scene.Image) uintptr { t.Fatal("uploaded retired input"); return 0 })
	if len(b.Batches) != 1 || b.Batches[0].Textures != [4]uintptr{} {
		t.Fatal("retired inputs emitted handles")
	}
}
