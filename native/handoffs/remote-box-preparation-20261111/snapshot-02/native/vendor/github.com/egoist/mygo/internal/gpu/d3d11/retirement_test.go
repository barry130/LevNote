//go:build windows

package d3d11

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"
	"weak"

	"github.com/egoist/mygo/internal/scene"
)

func TestRetirementUnbindsBeforeCOMRelease(t *testing.T) {
	var order []string
	var table [112]uintptr
	table[release] = syscall.NewCallback(func(uintptr) uintptr { order = append(order, "release"); return 0 })
	table[ctxPSSetShaderResources] = syscall.NewCallback(func(_, start, count, views uintptr) uintptr {
		if start != 0 || count != 10 {
			t.Error("wrong SRV unbind range")
		}
		for _, v := range unsafe.Slice((*uintptr)(ptr(views)), 10) {
			if v != 0 {
				t.Error("bound SRV survived")
			}
		}
		order = append(order, "srvs")
		return 0
	})
	table[ctxOMSetRenderTargets] = syscall.NewCallback(func(_, count, _, _ uintptr) uintptr {
		if count != 0 {
			t.Error("render target stayed bound")
		}
		order = append(order, "targets")
		return 0
	})
	table[devCreateTexture2D] = syscall.NewCallback(func(_, _, _, _ uintptr) uintptr { return 0x80004005 })
	obj := uintptr(unsafe.Pointer(&table[0]))
	handle := uintptr(unsafe.Pointer(&obj))
	dead, live := scene.NewImageRGBA(1, 1, nil), scene.NewImageRGBA(1, 1, nil)
	r := Renderer{ctx: handle, ctx1: handle, device: handle, swapChain: handle, w: 1, h: 1, bgSet: true, held: true, images: map[uint64]*texture{
		dead.ID(): {tex: handle, srv: handle, image: weak.Make(dead)},
		live.ID(): {tex: handle, srv: handle, image: weak.Make(live)},
	}, canvases: map[uint64]*canvasTexture{99: {texture: texture{tex: handle, srv: handle, rtv: handle}}}, grab: texture{tex: handle, srv: handle}, mask: texture{tex: handle}, stops: texture{tex: handle}}
	r.retirement.Check(nil, func() {})
	s := scene.Scene{Width: 1, Height: 1, MaskAtlas: &scene.Atlas{W: 1, H: 1, BPP: 1, Pix: []byte{255}}}
	r.diff.Diff(&s)
	r.diff.Remember(&s)
	dead.Retire()
	if r.imageView(dead) != 0 {
		t.Fatal("cached retired image returned")
	}
	// Same remembered scene would return unchanged before touching the fake
	// device. Retirement must run first and force the next upload attempt,
	// whose intentional failure stops before any real drawing API is needed.
	if drew, err := r.drawFrame(&s); drew || err == nil {
		t.Fatal("retirement was skipped by no-damage return")
	}
	if r.retireImages() {
		t.Fatal("repeated checkpoint")
	}
	if len(order) != 10 || order[0] != "srvs" || order[1] != "targets" {
		t.Fatalf("unbind/release order %v", order)
	}
	if len(r.images) != 1 || r.images[live.ID()] == nil || len(r.canvases) != 0 || r.grab.tex != 0 || r.held || r.color.tex != 0 || r.stops.tex != handle {
		t.Fatal("wrong cache invalidation")
	}
	if _, known := r.diff.Diff(&s); known {
		t.Fatal("retirement could take no-damage return")
	}
	// No native renderer/device exists: final Release only frees fake objects.
	r.ctx = 0
	r.Release()
	n := len(order)
	r.Release()
	if len(order) != n {
		t.Fatal("double release after retirement/teardown")
	}
	runtime.KeepAlive(obj)
	runtime.KeepAlive(table)
	runtime.KeepAlive(live)
}
