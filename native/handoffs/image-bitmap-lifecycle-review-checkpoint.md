# Bitmap lifecycle review checkpoint

Completed2026-11-10: image-bitmap-lifecycle-design.md (145 lines) supersedes
the pending research below. Read-only recommendation delivered; no vendor edits.

2026-11-10 Astra. Read-only review paused for root's matrix-wrapper correction.
This is research progress, not an accepted API design or implementation.

- Task: image-bitmap-lifecycle-task.md. Design deliverable remains
  image-bitmap-lifecycle-design.md; do not resume historical tab/package work.
- Read native/vendor/github.com/egoist/mygo/AGENTS.md, scene/scene.go Image and
  EachCanvas, ui/bitmap.go API, ui/canvas.go, raster/canvas.go and patch README.
- MyGo v0.8.3 Bitmap has no terminal retirement. ReleasePixels installs Restore/
  Dropped closures, retains remake/source and may synchronously decode again.
- DecodeBitmap's remake captures compressed input. Mips are separate Image IDs.
  Update reuses ID for equal size; dimensional replacement loses old base/mip
  IDs to existing backend aging. Need explicitly retire replaced ownership too.
- Scene.Image has id/version/Pix/Scene/PixVersion/Transient/Restore/Dropped,
  without a tombstone. Pixels can invoke Restore; stale scene/mip references
  need terminal no-op semantics and cannot resurrect after device loss.
- Canvas.Bitmap and region wrappers alias one canvas Image. Its disposal rule
  must be deliberate; rendered canvas pixels are retained snapshots unless
  dependency invalidation is explicitly implemented. Scene.EachCanvas visits
  referenced effect images and nested canvases children-first, deduplicated.
- D3D11 uses ID-keyed images, weak canvas references, ordinary texture aging
  >240 unused frames checked every120 presented frames. A global consume-once
  retirement queue would be wrong for multiple renderers.
- Need targeted remaining reads: raster restorePixels/image/damage, GPU shared
  damage/sum, D3D11 imageView/Render/sweep/Release, Metal imageTexture/in-flight
  command ownership/trim/Release, GL per-context image cache and context loss;
  UI/surface normal render threading and idle no-damage early returns.
- Explicit recommendation must cover UI-thread idempotence, immediate CPU/
  remake/source/mip clearing, each backend's next normal redraw (or close),
  stale references, Update, close/device loss, derived caches and multi-renderer.
- Need inspect dependency-patches apply.ps1/refresh.ps1/check tests, scripts/
  go.ps1 build guard fixtures, portable-package-lib explicit attribution list,
  THIRD_PARTY_NOTICES. Recommend a second pinned patch; preserve startup patch
  and immutable .22 package. New attribution changes future package file count.
- Vendor AGENTS references docs/architecture.md but that file is absent from
  vendor. No code changed, tests/build/launch/network performed for this task.
- Binary API belongs to GLM. Accepted immutable active snapshot copy is allowed
  after source changes/deletes; reacquisition verifies baseline. Do not repeat
  older overview's stronger disk-change restriction.

Pending design should stay bounded (~100–150 lines), name exact future files and
pure/fake tests, distinguish separately authorized driver measurement, and
compare active-only cache alternative. No finalizer/GC driver-release claim.
