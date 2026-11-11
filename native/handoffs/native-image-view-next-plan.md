# Local PNG/JPEG reader: proposed next slice

2026-11-10. Read-only source inspection; no implementation, tests, builds,
launches, package changes or product-doc edits. Accepted tab source stays frozen.

Recommendation: assign a bounded binary-document prerequisite first. A useful
native reader is feasible, but adding only an image Adapter would bypass saving,
session and lifecycle invariants. Resolve the public GPU-lifecycle gap below
before promising a resource-bounded image release.

## Findings that determine the contract

- `core/document/document.go`, `encoding.go`, `disk.go`: documents own text
  buffers. Open stable-reads up to128MiB before rejecting PNG/JPEG signatures.
  The private stable-read machinery supplies file identity and SHA256; reuse it.
- `app/app.go:loadForOpen` turns rejected files into readonly hex text from
  `Inspect`'s64KiB prefix. That prefix is not a complete stable binary snapshot.
  `app/plugins.go:configureTab` bypasses adapters for ByteInspector; Save refuses
  it, including SaveAs. Never make its synthetic text buffer the image authority.
- Text SaveAs re-encodes Buffer; Reload/Revert and disk checks assume a text
  Document. `app/state.go` skips inspectors; `core/session` requires a text
  buffer and restores paths through OpenWithEncoding. Images need explicit paths.
- `ui/editor/contract.go` exposes Buffer/Path, not binary ownership. A view must
  not reopen Host.Path itself. `modules/modules.go` is the composition root;
  plugins have builtin capabilities and closed schema3 routes for three text
  formats. An image reader must not inherit text templates or syntax routing.
- `releasePresentation` retains the View after Release. Release must clear every
  large reference, not merely stop drawing. Audit Select, mode changes, close,
  restore and recoverDraft's separate activation path.

## Proposed authority and behavior

- Add an explicit readonly BinaryFile alongside text Document, with exclusive
  ownership in Tab; retain path, size, accepted disk identity/digest and kind.
  Do not refactor every document type into a new generic framework.
- A tiny signature probe may select bounded binary admission before text Open's
  128MiB read. Probe data is only a hint; full stable bytes authorize decoding.
  Initially builtin image routing covers .png/.jpg/.jpeg plus PNG/JPEG signature
  verification. Unknown suffixes, disabled reader, unsupported/corrupt/oversized
  images retain byte inspection with a specific reason and no guessed conversion.
- Host owns immutable bounded snapshots, decode scheduling and publication.
  Add a small optional binary capability to the neutral view contract: metadata,
  cancellable request, owner/generation receipt and readonly result ownership.
  Keep filesystem access and SaveAs in host/document code; do not reuse scene
  worker requests or expose executable configuration. No binary bytes in Buffer.
- Ctrl+S cannot alter an image. SaveAs atomically publishes the exact accepted
  bytes to a new destination, refuses an existing destination, and adopts the
  new path/baseline only after publication. Original bytes and metadata are not
  re-encoded. Same-path SaveAs must not rewrite the source. Report actual state
  if publication succeeded but later durability/cleanup reporting failed.
- Inactive tabs keep no compressed snapshot. A later view/copy stable-reads and
  verifies the accepted baseline; changed/missing sources require visible reload
  or reopen, never silently substitute bytes for the previously displayed image.
  Reuse `core/document/atomic.go`'s nonoverwriting publication, not text SaveAs.
- Persist path-only binary records with an explicit validated kind; no Draft,
  Encoding or BOM fiction. Reopen through the binary dispatcher. Keep old text
  sessions compatible and preserve unsupported records/failures visibly.
- Image mode has dimensions/format/size status, no line/column/encoding claims.
  Disable text edit/find/GoToLine/outline/split controls through capabilities;
  Source means bounded readonly byte inspection. No image template/New entry.

## Initial admission and resource policy (proposed constants)

- Compressed input <=16MiB; each side1..8192; pixels <=8,388,608. Check stat and
  actual limit+1 read, exact signature, DecodeConfig, overflow-safe products,
  format/header agreement and final decoded bounds on the same immutable bytes.
- Start with <=8-bit noninterlaced PNG and8-bit baseline SOF0 JPEG, grayscale or
  standard3-component sampling with factors1/2. Reject16-bit/interlaced PNG,
  progressive/CMYK/other JPEG modes explicitly in this first slice. This avoids
  pretending4bytes/pixel bounds PNG16 or JPEG progressive coefficient storage.
- A bounded PNG chunk walk rejects acTL/APNG; bounded JPEG segment parsing checks
  SOF/components/sampling before allocation. Validate lengths/overflow/truncation;
  metadata stays opaque except bounded EXIF orientation. Preflight is not full
  file validation; codecs may still fail, with byte inspection available.
- One app-wide decode/copy admission slot, one current bitmap, no inactive image
  cache or thumbnails. Reserve <=128MiB of modeled decode payload before work:
  compressed input + codec planes/MCU padding/row scratch + RGBA conversion +
  possible EXIF rotation copy. Reject estimates above it. Derive allocation
  formulas from the pinned codecs in implementation, not a guessed multiplier.
- Retained base RGBA plus every possible mip level <=48MiB, computed exactly
  with ceil-halving dimensions before decode; compressed retained input <=16MiB.
  Release previous image payloads before another decode. SaveAs reuses the active
  immutable snapshot; no second source copy/full-resolution export buffer. These
  are app-owned payload budgets, not a measured or hard RSS/driver ceiling.
- Coalesce navigation to one latest request; cancellation invalidates publication
  immediately. One codec call must finish before another begins. Context cannot
  preempt codec CPU loops; canceled work may hold its bounded inputs/output until
  it returns. Do not block the UI to join or start overlapping abandoned decodes.
  Publish only for current tab, view owner, baseline and generation; clear stale
  completion closures/results. Shutdown joins outstanding work.

## Public drawing and lifecycle limit

- MyGo `ui/bitmap.go` offers DecodeBitmap (including JPEG EXIF orientation),
  NewBitmap/Update; `ui.Image`/Painter.Image and clipped Box+HandleInput support
  fit, zoom and drag pan. Prefer a preflighted DecodeBitmap, not DecodeImage
  (which also admits SVG). No canvas, WebView or widget-tree inspection needed.
- App-owned viewport behavior: Fit,100%, bounded pointer-centered zoom and pan,
  finite-value checks and resize clamping.100% means1source pixel per DIP with
  current public APIs; do not label it physical screen-pixel accuracy on HiDPI.
- DecodeBitmap retains compressed data in a remake closure. ReleasePixels only
  drops CPU pixels and can redecode synchronously on the UI thread; it is not
  disposal. Avoid it in the initial reader; clear bitmap/bytes/work references
  on deactivation, retaining only small zoom/pan state. GC reclamation is later.
- There is no public bitmap Dispose. D3D11 currently sweeps ordinary textures
  every120frames after >240unused frames; idle views can retain them. Update
  reuses identity only at the same dimensions. One app bitmap does not bound
  accumulated GPU textures across varied files. Do not import renderer internals,
  force redraws for eviction, or claim deterministic GPU release.
- Before image release, require a public disposal/bounded renderer-cache contract
  with tested retirement semantics, or root explicitly narrows the resource claim
  to app-owned payloads. This is a real dependency decision, not solved by GC.

## Three bounded increments and proposed ownership

1. Binary prerequisite: new `core/document/binary.go` plus focused tests; small
   `app/binary.go` dispatcher and necessary app.go/state.go/workers.go hooks;
   validated path-only records in core/session. Establish stable snapshot, exact
   SaveAs, change/reload, fallback and release authority before adding pixels.
2. Reader: new `formats/image` preflight/budget helpers, `modules/imageview`, a
   neutral binary capability file under ui/editor, modules.go/builtin registry
   wiring and capability UI gates. No schema3 image route in the initial slice.
   Gate this increment on the lifecycle decision above; no implicit vendor patch.
3. Integrate and validate: adversarial header/overflow/limit/APNG/mode cases;
   source SHA256 equality and nonoverwrite/error/disk-change SaveAs cases;
   text-session compatibility and binary reopen; fake delayed-decode A->B->close
   stale-result tests and bounded concurrent work; actual pointer zoom/pan/fit,
   byte fallback and text-command routing; EXIF/alpha/palette fixture captures.
   Then one affected final suite/vet. Later separately authorized host/resource
   evidence must exercise repeated varied images and idle release; Tester images
   cannot establish physical GPU/RSS behavior or native pointer quality.

No editing, re-encoding, animation, SVG, HTML, remote URLs, metadata editor,
color-management/HDR guarantee, background scanning or image export. Go codecs
are licensed BSD-style primitives; MyGo is MIT. App-authored admission, viewport
and lifecycle logic does not make those dependencies copyright-free.

Next assignment: increment1 with a reviewed binary/session contract; resolve
the public GPU lifecycle choice before assigning increment2. This plan is frozen.
