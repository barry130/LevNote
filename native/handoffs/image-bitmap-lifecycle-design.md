# Terminal Bitmap retirement: proposed pinned MyGo patch

2026-11-10 Astra. Read-only design for root activation; no vendor/product code,
test/build/launch/network work performed for this review. Binary API is GLM's
separate ownership. Recommend explicit retirement, not an active-only cache.

## Public contract

Add only `func (b *Bitmap) Dispose() bool` to MyGo v0.8.3 `ui/bitmap.go`.
Call on the UI/main thread, including through existing Window.Update, as
Update/ReleasePixels already require; never while a Render/CPU band is running.
Return true for a retired ordinary bitmap, already retired bitmap or nil;
return false without mutation for Canvas.Bitmap/region wrappers. Those borrow
a Canvas's shared Image; disposing one must not destroy the still-usable Canvas.
Document that Bitmap values must not be copied; share the pointer instead.

Successful Dispose is terminal: clear base/mip CPU pixels and Restore/Dropped,
remake/source closures (including compressed bytes), levels/fresh, and bitmap's
image reference; Size becomes0,0. Image/region/shader use draws transparent;
Update and ReleasePixels do nothing, without allocating/converting/decoding.
Pixelated remains chainable/no-op. Retained old scenes/transition references
remain safe; caller must create a new Bitmap to show another picture.
This drops toolkit ownership, not external aliases to an input RGBA's pixels,
independent captures/copies, or the screen's already-presented framebuffer.

Disposal schedules no redraw itself: dispose inside the ordinary state update
that removes/replaces the view, using the existing one normal redraw. Each
renderer releases its ownership at its next drawable render opportunity or
Release. An idle/minimized renderer retains its GPU cache until that opportunity;
there is no timer/forced frame loop or guarantee of immediate OS/VRAM decrease.

## Actual source findings and minimal mechanism

Image ID is stable across same-size Bitmap.Update; size-changing Update replaces
base and drops level IDs. ReleasePixels installs callbacks back to Bitmap and
re-decodes lazily, so it cannot be terminal. D3D11/Metal/GL ordinary image caches
use ID keys and >240 unused-frame aging; sweep runs every120frames. Scene.Reset
clears refs, but Differ and Canvas scenes can keep old Images intentionally.

Add internal Image retired state and idempotent Retire/Retired; preserve ID and
dimensions for stale geometry, bump Version once, nil Pix/Restore/Dropped.
Pixels returns nil and Uploaded/Changed/Resize cannot revive a retired Image.
Increment one global monotonic retirement epoch; no global ID queue or retained
bitmap registry. Each GPU image entry holds only weak.Pointer[scene.Image]
beside existing handles. Each renderer independently remembers the epoch;
on change, remove entries whose Image is retired or already unreachable.
Weakness avoids cache→image→bitmap/remake retention. Explicit retired state,
not GC timing, triggers disposal; multiple renderers never consume each other's
notification. Epoch reads may be atomic, but CPU pixel mutation remains UI-only.

Same-size Update keeps existing texture/version behavior. Size-changing Update
retires the old base and every old mip before replacement. Dispose retires all
current levels. Thus no abandoned image generation relies on aging, and stale
references to replaced dimensions become transparent. ReleasePixels followed
by Dispose must not call remake; neither device recreation nor a new renderer
may upload or decode a retired Image. Keep small stale tombstones until their
scene refs end; don't mutate/reuse their IDs for another picture.

## Derived caches, pending frames and backend-specific retirement

Canvas is a retained scene, not a durable independent snapshot. Choose a small
conservative invalidation on any retirement epoch change instead of a reverse
dependency graph: free each renderer's canvas textures/builders and bitmap
blur outputs/scratch, invalidate frame damage, then rebuild any still-used
canvas from its scene with retired inputs transparent. Include nested canvases,
EffectOp.Images and internal Cache layers. Other canvas content remains usable.
This can rebuild unrelated canvases once; that bounded, infrequent cost is the
tradeoff. Main target, font/gradient atlases and unrelated live images stay.
Renderers' grab/backdrop scratch is also released at this checkpoint, preventing
large obsolete image/effect scratch from remaining at its previous high-water.

CPU canvas Pix/PixVersion needs a retirement-epoch stamp: on next raster use,
invalidate/rebuild even when canvas Version is unchanged. Raster.Render also
invalidates its Differ on epoch change so copied framebuffer pixels refresh.
This is lazy derived-cache invalidation; only bitmap-owned CPU pixels are
cleared immediately. Existing independent CPU canvas copies may live until
their own next use/owner release. Never promise secure pixel erasure.

| Path | Required checkpoint and release behavior |
|---|---|
| D3D11 | Before drawFrame's no-damage early return (covers offscreen draw too), retire cache handles and invalidate held/Differ. Unbind image/effect SRV slots0–9 and affected render targets before releasing SRV/texture/RTV; context bindings otherwise retain COM refs. Use existing free/freeCanvas; no tree/global device destruction. Already submitted commands may finish; normal frame submission retires runtime refs safely. |
| Metal | In render/prepare before damage/upload and at finishPending entry before extracting its pendingFrame. On changed epoch, dropPending before freeing caches: already encoded work commits via existing track/wait ownership; unencoded main batches must never use freed IDs. Invalidate frames, wake normal content, purge; finishPending then returns without drawing that dropped main frame. Normal commandBuffer retains encoded resources; no unretained buffers. Release cache refs; in-flight refs end on completion. Existing trim/Release waits/drains remain. |
| GL/WebGL | After verifying/reinitializing current context, before draw/no-damage. Delete textures and derived targets only in owning current context; reset bindings/state through existing draw path. Context mismatch forgets dead handles and rebuilds empty cache; no deletes against a different context. Existing Release mismatch semantics leave objects to context destruction. |
| Raster | Before whole/Diff; restorePixels runs before joined parallel bands. Pixels and transformed sample/blur paths treat retired/empty pixels as transparent; never run Restore. Invalidate CPU canvas epoch and redraw affected frame. No GPU lifetime claim. |

Every backend image lookup checks Retired BEFORE a cached-hit return, and no
texture creation/update occurs for it. Cache retirement must execute before
unchanged-frame returns. A zero-size/no-render surface is not such an
opportunity. Remove retired entries so subsequent Release/sweep cannot double
free; renderer recreation starts empty. Windows windowHost.render already
releases a failed GPU and retries/falls back; detach releases GPU/software and
last scene. CPU bands finish synchronously before UI can dispose. GL context
is current only during surface events; Metal presenter fetches drawables off
thread but finish callbacks/cache mutations run on the main thread.

## Exact implementation and regression scope after activation

Vendor paths below are relative to native/vendor/github.com/egoist/mygo/:
- ui/bitmap.go; ui/paint.go (drawBitmap/ImageRegion nil-retired guards);
  ui/material.go (omit retired shader input); ui/canvas.go API ownership docs.
- internal/scene/scene.go plus a small image-retirement helper if useful;
  Image/epoch/Pix-epoch state. Existing damage.go version checks already detect
  direct stale Image; effects with images already compare as changed.
- internal/gpu/{d3d11/d3d11.go,metal/metal.go,metal/present.go,gl/gl.go};
  weak cache refs, retirement checkpoint, derived cleanup and state reset.
- internal/raster/{damage.go,canvas.go,raster.go,blur.go} only where guards/
  epoch invalidation require changes. Avoid a public renderer/internal import
  in LevNote, generic eviction framework, finalizers or new heavy dependencies.

Meaningful focused tests: bitmap_retirement_test.go in ui; image_retirement_test.go
in internal/scene; retirement_test.go in raster and each GPU package. Cover
nil/double disposal, ReleasePixels callbacks never invoked after retirement,
CPU/mip/remake cleared, Update same/different size and post-dispose no work,
stale base/mip and transformed/blur/shader inputs, Canvas alias refusal, nested
canvas invalidation, two renderers/independent epochs, device recreation,
no-damage first frame, close-before/after cleanup and exactly-once release.
Factor only a small actual cache-sweep helper with a fake release callback if
needed to exercise handles without drivers; don't test copied pseudo-logic.
Metal pending-drop ordering and GL wrong-context no-delete need fake checks.
Add explicit targeted vendored-package test invocation (normal ./... skips
vendor). Later cross-compile affected packages for Windows, Darwin, Linux and
js/wasm using native/scripts/go.ps1; actual target runners where available.
Fake tests prove ownership/call ordering only. Real Windows driver texture
retirement/memory and no-feature binary-size validation need separate activation.

## Patch provenance and alternative

Add dependency-patches/mygo-v0.8.3-bitmap-dispose.patch; retain existing startup
safety patch. refresh.ps1 is the actual applier (no apply.ps1 exists): apply
both in order after vendor, check exact v0.8.3 and then verify. Extend actual
verify_test.go, dependency-patches/README.md, scripts/go.ps1 guard and
check-build-guard.ps1 fixtures with marker/API/backend coverage or pinned-file
digests, not one cosmetic marker that admits a partially applied patch.
Update THIRD_PARTY_NOTICES.md, portable-package-lib.ps1 Get-PortableAssetCopies
and notice link rewrite, plus check-portable-package.ps1 expected assets.
Future package attribution gains one patch file; do not modify immutable .22.
Record patch provenance/tests without claiming upstream approval. Vendor AGENTS
references docs/architecture.md absent from vendor; inspected actual paths above.

Active-only GPU eviction is insufficient: current scene/canvas refs can still
upload old pixels, mips/remake remain, and windows/canvases need independent
ownership. Dependency traversal every frame also risks churn. Recommend the
explicit Dispose + epoch/weak-entry invalidation above, with the reader keeping
one active bitmap and disposing it on replacement/close. Root activates this
vendor prerequisite separately; no image-view integration is authorized here.
