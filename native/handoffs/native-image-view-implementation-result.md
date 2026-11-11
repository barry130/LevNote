# Native image view implementation result

2026-11-10. Implementation complete for root review. Work stayed within new
`modules/imageview/**`, new `ui/editor/image.go`, the additive `Host.BeginImage`
field in `ui/editor/contract.go`, and new checkpoint/evidence/result handoffs.
No app, core, `modules.go`, raster, plugin, document, session, vendor, script or
product-doc file was edited. No commit, push, branch, child agent, product
build, host, worker or resource run occurred.

## Neutral contract

`native/ui/editor/image.go` defines:

```go
type ImageRequest struct {
    Epoch uint64
}

type ImageResult struct {
    Format          string
    Width, Height   int
    SourceWidth     int
    SourceHeight    int
    Orientation     uint8
    CompressedBytes int64
    Pixels          *image.RGBA
}
```

`native/ui/editor/contract.go` adds, beside `BeginScene`:

```go
BeginImage func(
    owner View,
    request ImageRequest,
    complete func(ImageResult, error),
) (cancel func(), err error)
```

The callback is UI-only and nonblocking. The host captures current binary
authority, token, active owner and generation; the view supplies no path or
source bytes. Completion runs on the UI thread, at most once unless canceled.
Cancel is idempotent and UI-only. Successful pixel ownership transfers to the
view, while the host drops its reference after callback. A stale callback drops
only its pixels and cannot construct a bitmap or revive a released view.

The exact contract definitions were sent to root as soon as they were saved.

## Adapter and presentation

`imageview.Adapter()` is intentionally compile-ready but unregistered:

- ID `image`, name `原生图片阅读器`
- case-insensitive `.png`, `.jpg`, `.jpeg`
- `Create` returns an `editor.View` from `editor.Host`
- no Template, Outline, Exports, SceneEngine or ExportEngine

The single clipped viewport uses one `ui.Box`, one public
`Painter.Image`, and at most one `*ui.Bitmap`. Public Tester input covers wheel,
pinch, drag, toolbar clicks and resize. There is no widget-tree inspection,
internal renderer import, source parsing, filesystem access, remote link,
animation, thumbnail generation, export or idle invalidation.

The toolbar exposes `适合窗口`, `100%`, `减小`, current percentage, `增大`,
truthful format/dimension/compressed-size metadata, and a `查看只读源码` action
that calls only `Host.ShowSource`. A refusal remains visible as a warning and
does not clear or replace the image view.

## Exact geometry

> This section records the first frozen implementation. The root review
> corrections and the authoritative current geometry are documented below.

All values are finite floats in DIPs. Invalid viewport size, image size, zoom or
pan is rejected before drawing. Constants are:

- `FitPadding = 24 DIP` on each side
- `MinZoom = 0.02` (2%)
- `MaxZoom = 8` (800%)

Fit uses:

```text
availableW = max(viewW - 48, 0.5)
availableH = max(viewH - 48, 0.5)
scale      = min(1, availableW/imageW, availableH/imageH)
fitZoom    = clamp(scale, 0.02, 8)
panX       = (viewW - imageW*fitZoom)/2
panY       = (viewH - imageH*fitZoom)/2
```

The `min(1, ...)` policy deliberately forbids unbounded enlargement of tiny
images. Manual controls can still reach 800%. Fit keeps aspect ratio, centers
the image, and recomputes whenever the viewport size changes.

100% sets one image pixel per DIP. Every displayed rectangle is:

```text
screenX = viewportX + panX
screenY = viewportY + panY
screenW = imageW * zoom
screenH = imageH * zoom
```

For either axis, `scaled <= view` centers the image with
`(view-scaled)/2`; a larger image clamps pan to `[view-scaled, 0]`.
Pointer zoom preserves the image point under the pointer:

```text
imageX = (pointerX - panX)/oldZoom
imageY = (pointerY - panY)/oldZoom
newPanX = pointerX - imageX*newZoom
newPanY = pointerY - imageY*newZoom
```

The new pan is then clamped as above. A conventional wheel delta is negated,
capped to four steps per event, and applied as `1.12^steps`. Drag uses the
pointer origin plus the current displacement, so multiple move events do not
accumulate. The neutral checkerboard is clipped to the visible image rectangle,
uses 12-DIP cells, and moves with the image.

## Ownership and lifecycle

The first `Draw` of an activation requests exactly once. Loading, decode failure,
missing host service and source refusal all have visible state. A result is
validated before bitmap creation: final dimensions must agree with source
dimensions and EXIF rotation, and pixels must be one origin-zero, tightly packed
RGBA surface of the declared size.

`Release` synchronously increments the generation, revokes completion authority,
calls the returned cancel once, clears every image/result/metadata reference,
and calls public `Bitmap.Dispose` on the UI thread. It retains only small
viewport geometry and an explicit disposal error if disposal fails. A false
disposal result is reported and remains visible; it is not ignored. A released
view can be reused by the next `Draw`, which starts one fresh request.

A late completion from an earlier epoch, including after release and redraw,
only nils its local `Pixels` reference. It cannot allocate a bitmap, overwrite a
fresh result, clear loading state or resurrect the released view. After release,
the disposed bitmap reports `(0,0)`.

The view never calls `DecodeBitmap`, `ReleasePixels`, opens the document or
retains source bytes/readers. Successful RGBA is passed once to `ui.NewBitmap`
and the view's pointer to it is immediately cleared.

## Verification

There are ten substantive top-level image-view tests. Focused coverage includes
pure fit/aspect/resize geometry, 100%, pointer-anchored zoom, pan bounds, public
drag input, finite guards, one request per activation, deferred and synchronous
completion, loading/error/source states, stale A after release/redraw versus
fresh B, zero-size after disposal, explicit disposal failure, and real
`formats/raster.Decode` integration for an EXIF orientation-6 JPEG and a
transparent palette PNG.

| Step | Exit | Result |
|---|---:|---|
| 01 focused imageview | 1 | Compile errors in test source; corrected |
| 02 focused imageview | 1 | Loading/error text was painter-only; made accessible UI text |
| 03 focused imageview | 0 | PASS |
| 04 focused imageview | 0 | PASS after drag-origin/finite corrections |
| 05 final imageview + editor suite | 0 | Both packages PASS |
| 06 affected-package vet | 0 | Empty stdout and stderr |

Exact commands, stdout, stderr and exits are in
`handoffs/native-image-view-evidence-20261110/`. The required checkpoint is
`handoffs/native-image-view-checkpoint.md`.

Two 760x480 synthetic Tester captures show the toolbar and alpha checkerboard:

- light: `imageview-alpha-toolbar-light-760x480.png`, SHA256
  `f42a9161b09c7c709698fc680a7d5dece825ff4999bab23352ec941477844ad9`
- dark: `imageview-alpha-toolbar-dark-760x480.png`, SHA256
  `43718399e2becd68c0bbf1e296cb8c3eea60a27c98671e469aa88a34a6e3e8bf`

They establish software-renderer composition only, not physical OS/font/driver/
GPU/IME behavior.

## Scoped SHA256 identities

```text
cc7d5510f81abbc10704cf0ef32f4ff9fc6f96de66937d1af19ded312b4fca8d  ui/editor/contract.go
775ddfb9c1477dd357232cc5cb242c1a15e9b19a3349217ad3e86f1460c8c073  ui/editor/image.go
e24ee00325b412de2d67571808e103c7c96b0e35acfe8c5e6c28c90d1ac0dfba  modules/imageview/adapter.go
68673dac188e5d162f41ad8f69d2f7b6fb68f55d85705959de86179f31d844ad  modules/imageview/geometry.go
5f6d04645cb0d1464cb587e4d28174b6f7b82c583e388fda04ff81639551debe  modules/imageview/view.go
e3e9efb80df0b1c44de47a281528a8ba25bb9e52b23e2b7a471b25f27b4b8996  modules/imageview/ui.go
eae500a2acdd90d65efae2ef92117eb92e62353e4a048cd56b98a649783cbc0a  modules/imageview/input.go
56a6a3e87fa8debc8b1bdfbf280664afdceaa2cd48a0d69625cb80105ca52af8  modules/imageview/imageview_test.go
b12230d0c9e1871e6e058144415419b7610dc7afca17f199f976042f55433526  handoffs/native-image-view-checkpoint.md
f42a9161b09c7c709698fc680a7d5dece825ff4999bab23352ec941477844ad9  handoffs/native-image-view-evidence-20261110/imageview-alpha-toolbar-light-760x480.png
43718399e2becd68c0bbf1e296cb8c3eea60a27c98671e469aa88a34a6e3e8bf  handoffs/native-image-view-evidence-20261110/imageview-alpha-toolbar-dark-760x480.png
```

Paths above are relative to `native/`.

These identities record the first frozen implementation. The reviewed source
identities are listed in the correction section below.

## Pending integration

The application side still needs to implement `Host.BeginImage`. It must capture
the current binary snapshot/authority, active token/owner and generation, decode
through the accepted bounded `formats/raster.Decode`, map truthful metadata and
compressed size into `ImageResult`, deliver once on the UI thread, invalidate the
window after delivery, and cancel/ignore stale work without retaining pixels.

The composition owner must then register `imageview.Adapter()` for the evolving
binary document route. This change does not register it in `modules.Builtins`.

## Limits

- No app wiring, binary host route, product package, launch or resource
  measurement was performed.
- The accepted raster subset and EXIF behavior are inherited; this view supports
  no additional codecs.
- Fit caps tiny images at 100%; explicit zoom range is 2% through 800%.
- Codec CPU between guarded reads remains non-preemptible, as documented by the
  accepted decode contract.
- The captures are synthetic Tester output, not a physical-window/driver claim.
- One bitmap and no idle invalidation are source-level behavior; no process RSS
  or VRAM reduction was measured.

## Root review corrections

Root reviewed the first frozen implementation and found four narrow defects.
The correction stayed within the same imageview/editor ownership and preserved
all original source history, evidence runs 01 through 06, result text and
captures.

Fit no longer uses the manual zoom floor. It returns the positive finite scale
required to show the whole image at up to 100%, even when that scale is below
2%. Explicit zoom actions still clamp to the manual range:

```text
availableW = max(viewW - 48, 0.5)
availableH = max(viewH - 48, 0.5)
fitZoom    = min(1, availableW/imageW, availableH/imageH)  // > 0
manualZoom = clamp(requestedZoom, 0.02, 8)
```

For the reviewed case, `fitViewport(120, 120, 1, 8192)` produces
`fitZoom = 72/8192 = 0.0087890625`, `panY = 24`, and a 72-DIP-high image
fully inside the 120-DIP viewport. Decreasing from a sub-floor fit is a no-op
instead of snapping upward to `MinZoom`; increasing explicitly enters the
manual range at `MinZoom`.

Manual resize preserves the image coordinate at the old viewport center before
clamping:

```text
oldCenterImageX = (oldViewW/2 - oldPanX)/zoom
oldCenterImageY = (oldViewH/2 - oldPanY)/zoom
candidatePanX   = newViewW/2 - oldCenterImageX*zoom
candidatePanY   = newViewH/2 - oldCenterImageY*zoom
newPan          = clamp(candidatePan, newViewport, scaledImage)
```

Thus the exact 1000x800, zoom-1 case from a 400x300 viewport with pan
`(-300,-250)` translates to `(-200,-200)` at 600x400. If a dimension becomes
smaller than the viewport, clamp then centers that axis. Both cases are covered
through the actual Tester resize lifecycle.

`MaxZoom` now states its real direction: eight DIPs per image pixel, or 800%.
The binary inspection button reads `查看字节`, its accessible label remains
`查看只读字节`, and refusal prose says `只读字节视图`. `imageRectangle` checks
the converted `float32` X and Y positions as well as width and height, so a
finite float64 coordinate that overflows during conversion cannot reach
drawing.

### Review verification

The focused regression command passed, followed by the corrected captures,
the final affected suite and vet:

| Step | Exit | Result |
|---|---:|---|
| 07 focused review regressions | 0 | PASS |
| 08 corrected light/dark captures | 0 | PASS; original captures retained |
| 09 final imageview + editor suite | 0 | Both packages PASS |
| 10 final affected-package vet | 0 | Empty stdout and stderr |

Exact review commands, streams and exits are runs 07 through 10 in
`handoffs/native-image-view-evidence-20261110/`.

The corrected captures are:

- `imageview-alpha-toolbar-light-760x480-review-fix.png`, SHA256
  `c00aa0141252f585691226d0ac4092ce67c2425b5175fbd0536ba78e633ba92f`
- `imageview-alpha-toolbar-dark-760x480-review-fix.png`, SHA256
  `37ea85ba9bdd51a20cc43b23d11e5813ae56133a476cf71aadb92d1079d6298e`

The original light/dark captures remain byte-identical:

- light `f42a9161b09c7c709698fc680a7d5dece825ff4999bab23352ec941477844ad9`
- dark `43718399e2becd68c0bbf1e296cb8c3eea60a27c98671e469aa88a34a6e3e8bf`

### Reviewed scoped SHA256 identities

Paths are relative to `native/`.

```text
cc7d5510f81abbc10704cf0ef32f4ff9fc6f96de66937d1af19ded312b4fca8d  ui/editor/contract.go
775ddfb9c1477dd357232cc5cb242c1a15e9b19a3349217ad3e86f1460c8c073  ui/editor/image.go
e24ee00325b412de2d67571808e103c7c96b0e35acfe8c5e6c28c90d1ac0dfba  modules/imageview/adapter.go
23b1dd60acb7c6c76e823428dc2201931a868d4c37c5092cc7b6a8d1e5f52856  modules/imageview/geometry.go
aa1eb6a586c38579eff6f65ef9276b9e4fe35c75fd87bfc0f89821e06943b346  modules/imageview/view.go
a76bda4747c17f3efce8220f0c98a1e32a5fc41ade19d1f66fb83e3c10593ed7  modules/imageview/ui.go
eae500a2acdd90d65efae2ef92117eb92e62353e4a048cd56b98a649783cbc0a  modules/imageview/input.go
002fe4ec058d7fd936deaa49f88965e0af1694fce7139bb5db1d2d11bbfee92f  modules/imageview/imageview_test.go
7c252006688d76fd7de1522e1d93a848fcc82103debafb8dee6e69f5d560393a  handoffs/native-image-view-checkpoint.md
c00aa0141252f585691226d0ac4092ce67c2425b5175fbd0536ba78e633ba92f  handoffs/native-image-view-evidence-20261110/imageview-alpha-toolbar-light-760x480-review-fix.png
37ea85ba9bdd51a20cc43b23d11e5813ae56133a476cf71aadb92d1079d6298e  handoffs/native-image-view-evidence-20261110/imageview-alpha-toolbar-dark-760x480-review-fix.png
```
