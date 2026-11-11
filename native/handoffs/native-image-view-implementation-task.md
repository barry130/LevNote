# Native image presentation implementation

ACTIVE2026-11-10 for DeepSeek read_native_handoffs_raster_admission_finish_task_md.
Raster admission/decode and public Bitmap.Dispose are accepted. Implement now;
no extra design-only phase. No child agents. Own only new modules/imageview/**,
new ui/editor/image.go and the minimal additive Host field in contract.go, plus
new handoff/evidence. No app/modules.go/core/plugins/document/session/raster/
vendor/scripts/product-doc edits. Those owners will wire the accepted contract.

Read native/AGENTS.md, raster-decode-root-acceptance.md and
image-bitmap-lifecycle-root-acceptance.md. Historical next-plan's suggestion to
use DecodeBitmap is superseded: host Decode produces owned RGBA; NewBitmap only.

## Neutral contract and boundaries

Add an additive BeginImage callback to editor.Host, parallel to BeginScene:
UI-only nonblocking request(owner View, request ImageRequest,
complete func(ImageResult,error)) -> (cancel func(),err error).
ImageRequest has only Epoch uint64; the host captures current binary authority,
token, active owner and generation. No filesystem path/source bytes supplied by
the view. ImageResult contains owned *image.RGBA plus truthful small metadata:
format label, final width/height, source width/height, EXIF orientation and
compressed bytes. Use standard image types, not a concrete format/core import.
Callback is UI-thread, asynchronous, at most once unless canceled. Cancel is
idempotent/UI-only. Ownership of successful result pixels transfers to view;
host releases its reference after callback. Stale callbacks must only drop
pixels, never allocate a bitmap or resurrect a released view.

Adapter() ID image, name native image reader, matches .png/.jpg/.jpeg, Create
from Host. No Template/Outline/Exports/SceneEngine/ExportEngine. It may be
compiled independently but is NOT registered in modules.Builtins yet.

## Behavior

One app-owned viewport and at most one bitmap in this view. Request once on
first Draw; show loading/error explicitly, plus a host ShowSource callback to
bounded readonly byte inspection. Release must cancel, advance generation,
call public Bitmap.Dispose on UI thread, clear bitmap/result/pixel/closure
references, and retain only small geometry. Reusing the same View after Release
may request again on its next Draw; a late completion from the previous epoch
cannot overwrite current state. Dispose false for an ordinary owned bitmap is
an explicit error, not silently ignored. No ReleasePixels or DecodeBitmap.

Minimal toolbar: 适合窗口,100%,减小/增大, current percentage.100% means one image
pixel per DIP, not physical screen-pixel accuracy. Pointer-centered bounded
wheel zoom and drag pan through public clipped Box/Painter.Image/input APIs;
no widget-tree inspection/internal renderer import. Zero/NaN/Inf sizes/scales
must not enter drawing. Fit maintains aspect/centering, recomputes on resize;
manual zoom retains meaningful center then clamps pan. Pick/document small
finite min/max zoom. Prevent tiny images scaling without a bound; fit policy
must be clear. Neutral checkerboard behind alpha, simple light/dark colors.
No continuous idle invalidation, thumbnails, animation, source parsing, editing,
export, remote links or OS calls. Warnings/Export must remain safe with no pixels;
unsupported export returns explicit error, Flush is unnecessary for readonly.

## Verification

Use substantive pure geometry and actual MyGo Tester public input tests:
fit/aspect/resize,100%, cursor anchor on zoom, clamp, drag, finite bounds;
one request per activation, error and source callback; delayed A callback after
Release/re-Draw cannot revive or replace B; bitmap size becomes zero after
Release; result owns no reader/source and no retained inactive pixels. Fake
provider is for lifecycle; feed actual raster.Decode fixtures for at least
oriented JPEG and transparent/palette PNG integration. Test source callback
refusal doesn't silently drop current view.

Run focused new tests then one final affected imageview/editor suite and vet
after real fixes, all through scripts/go.ps1; preserve command/stdout/stderr/
exit and failures in native/handoffs/native-image-view-evidence-20261110.
At most two760x480 Tester captures (light/dark) covering alpha and toolbar,
saved in that evidence directory. They are synthetic, no OS/driver claim.
No product package, host, worker or resource run.

Checkpoint before checks to native-image-view-checkpoint.md, publish frozen
native-image-view-implementation-result.md with exact contract/geometry and
scoped identities. Send contract details to root as soon as additive definitions
exist so app integration can be planned without concurrent edits.
