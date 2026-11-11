//go:build darwin

package metal

import (
	"runtime"
	"testing"
	"weak"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/egoist/mygo/internal/scene"
)

func TestRetirementDropsPendingBeforeRelease(t *testing.T) {
	oldSend, oldSelectors := msgSend, selectors
	defer func() { msgSend, selectors = oldSend, oldSelectors }()
	selectors = map[string]objc.SEL{"commit": 1, "retain": 2, "release": 3}
	var order []uintptr
	msgSend = purego.NewCallback(func(obj, selector uintptr) uintptr { order = append(order, selector); return obj })
	img := scene.NewImageRGBA(1, 1, nil)
	r := Renderer{pending: pendingFrame{cb: 101}, images: map[uint64]*texture{img.ID(): {tex: 102, image: weak.Make(img)}}}
	r.retirement.Check(nil, func() {})
	woken := 0
	r.wake = func() {
		woken++
		if len(r.images) != 0 || r.pending.cb != 0 {
			t.Error("woke content before completing purge")
		}
	}
	img.Retire()
	if err := r.finishPending(false); err != nil {
		t.Fatal(err)
	}
	if len(order) != 4 || order[0] != 1 || order[1] != 2 || order[2] != 3 || order[3] != 3 || r.pending.cb != 0 || r.inflight[0] != 101 || len(r.images) != 0 || woken != 1 {
		t.Fatalf("pending/cache order: %v", order)
	}
	if r.imageTexture(img) != 0 || r.retireImages() {
		t.Fatal("retired texture revived")
	}
	// Fake encoded buffer retains its own resource ownership until completion.
	release(&r.inflight[0])
	runtime.KeepAlive(img)
}
