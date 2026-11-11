package gpu

import (
	"weak"

	"github.com/egoist/mygo/internal/scene"
)

// ImageRetirement is one renderer's cursor, not a globally consumed event.
// Check runs before an unchanged-frame return. before relinquishes pending
// batches/bindings before purge frees the handles those batches reference.
type ImageRetirement struct{ epoch uint64 }

func (r *ImageRetirement) Check(before, purge func()) bool {
	epoch := scene.RetirementEpoch()
	if r.epoch == epoch {
		return false
	}
	r.epoch = epoch
	if before != nil {
		before()
	}
	purge()
	return true
}

// SweepRetired removes retired or unreachable image owners, releasing each
// cache entry once. Texture caches must not keep pixels or decoder closures.
func SweepRetired[T any](images map[uint64]*T, owner func(*T) weak.Pointer[scene.Image], release func(*T)) {
	for key, t := range images {
		if owner(t).Value().Retired() {
			delete(images, key)
			release(t)
		}
	}
}

// ForgetImages drops retained scene/batch references and derived clip masks,
// preserving builder options and unrelated font/gradient texture caches.
// drop removes each mask's texture from this renderer's ordinary image cache;
// these private masks need no further global retirement notification.
func (b *Builder) ForgetImages(drop func(uint64)) {
	for _, m := range b.maskSlots {
		if m.image != nil {
			drop(m.image.ID())
		}
	}
	b.maskSlots, b.clipShapes, b.stack, b.planStack = nil, nil, nil, nil
	b.Batches = nil
	b.fold = scene.TranslationFolder{}
	b.corners = scene.ClipCorners{}
	b.imageTexture, b.BlurredImage = nil, nil
	b.preparedMask = 0
}
