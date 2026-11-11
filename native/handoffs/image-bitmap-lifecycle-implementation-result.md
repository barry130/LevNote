# Bitmap retirement implementation — frozen

2026-11-10 Astra. Authorized vendor prerequisite completed; source frozen for
root review. No app/document/session/formats edits, product package, UI/driver
launch or resource measurement. No active vendor regeneration; no accepted .22
or fixed12 evidence changed. All Go commands used native/scripts/go.ps1.

## Behavior and ownership

- Public UI-thread `(*ui.Bitmap).Dispose() bool`: nil/repeat succeeds; borrowed
  Canvas.Bitmap/region returns false with no mutation. Ordinary bitmap disposal
  immediately clears base/mip Pix and Restore/Dropped, source/remake closures,
  retained compressed bytes and Bitmap references/size. Update/ReleasePixels/
  Pixelated/draw/shader paths cannot restore or revive it. Values must not be
  copied. Same-size Update preserves image/texture identity; size replacement
  retires old base and mips. Old scenes retain ID/geometry tombstones.
- Each renderer independently observes scene.RetirementEpoch. Ordinary GPU
  entries hold weak image references. Actual gpu.ImageRetirement.Check and
  SweepRetired helpers drop retired/unreachable ownership exactly once.
- Before no-damage return, release derived canvas builders/textures, image
  blur/scratch, backdrop/grab, derived clip masks and remembered frame refs;
  preserve unrelated live image textures and font/gradient atlases. CPU canvases
  use PixEpoch and rebuild nested image/effect/clip inputs lazily. Clip masks
  were included because their copied alpha can otherwise outlive retirement.
- D3D unbinds PS SRV0–9 and render targets before COM releases. Metal discards
  pending main batches, commits/tracks already encoded work, purges, then wakes
  content; finishPending checks before extracting pending. GL checks the owning
  current context; wrong-context teardown forgets rather than deleting names.
- Dispose requests no frame. The caller's ordinary state update removes the
  view; backend ownership ends on the next drawable render or Release. Idle/
  minimized renderers and in-flight native commands may retain resources until
  then. External pixel aliases and independent captures are not erased.

## Frozen files and provenance

`image-bitmap-lifecycle-frozen-files.json` records all35 exact byte/SHA256 entries:
33 changed/added plus unchanged startup app.go/patch references. It includes23
changed/new vendor source/test files, listed in the pre-check implementation
checkpoint; dependency-patches refresh/README/verify/new patch/manifest;
scripts go/check-build-guard/portable-package-lib/check-portable-package; notices.

- New disposal patch:47,176bytes, SHA256
  `995d5505c3b5b1bd869547df3f1ad5954c2dd57f1c9f32e53554851b5db0d4d1`.
-26-entry patch/source manifest:4,463bytes, SHA256
  `9113120bd2ebcd9a86842153239cddff84d1f1bd3f46f5d4f0f152eb4560e16a`.
- go.ps1:6,449bytes, SHA256
  `a3efcd7dc4079641f1767341446168e6778ab07e1c4dd09a968c768d1cc1b178`.
- Frozen identities JSON:7,823bytes, SHA256
  `2b1d185cb3302427e692b0fe955ce2d0aac0bab8f559775b8e522b56bca10e8a`.

Pristine source is the cached pinned MyGo v0.8.3 distribution; ziphash matches
go.sum: `h1:aTLEhS221YhsX+gKGNLQ5xEKMLCcYYr3pH9AkTwIRs4=`. Original affected
file hashes and upstream .gitattributes are in source-provenance.json. Both
patches remain attributed MIT modifications, with no upstream acceptance claim.
Refresh applies startup then disposal and verifies exact identities. Wrapper
pins the manifest itself and hashes all26 entries before guarded commands,
retaining startup and alternate-context guards. Future portable assets now
include both patches,14 independently listed attribution/license files.

## Preserved verification

Evidence root: `native/artifacts/acceptance/image-bitmap-lifecycle-20261110/`.
The saved runner scripts contain exact array arguments; result JSON records
commands, target environment, exit, times and stdout/stderr paths.

| Check | Actual result |
| --- | --- |
| Windows focused tests, count1: ui/scene/raster/gpu/d3d11 | All5 PASS, exit0;0.751/0.118/0.180/0.130/0.124s |
| Vet same5 packages | exit0, empty stderr |
| Target package builds, CGO_ENABLED0 | Windows amd64, Darwin arm64, Linux amd64, js wasm all exit0 |
| Darwin Metal and Linux GL test cross-compilation | Both exit0; no target runtime executed |
| Final Metal test compilation after preserving empty-pending no-op | exit0, final present.go included |
| Dependency provenance tests | PASS, exit0,0.633s |
| Build-guard fixture, pwsh7 |42/42 PASS; source/copy guard hashes identical |
| Portable path/asset/manifest fixture |25/25 PASS; no executable product payload or launch |
| Corrected pristine replay | Both check/apply commands exit0;24/24 exact vendor identities match |

Tests exercise actual Dispose/Retire/renderer/cache functions: no decode after
retirement, mips and size replacement, borrowed canvas refusal, nil/repeat,
stale direct/transformed/blurred scenes, nested/effect/clip canvases, independent
CPU renderers/recreation, independent epoch cursors, mask disposal and option
preservation. D3D fake COM callbacks exercise real drawFrame: an unchanged
remembered scene must purge, unbind, and reach an intentional fake upload
failure instead of taking no-damage return; teardown must not double release.
Metal fake messages verify encoded-buffer commit/track and pending main drop
before cache release/wake. Linux GL fake calls verify wrong-context no-delete.

Preserved failures/corrections:
1. Initial propagation runner passed an array through pwsh -File incorrectly;
   `-w` bound ambiguously before Go started.05-format-verifier logs retained;
   literal-array runner fixed invocation;10-format-verifier then exit0.
2. Initial pristine check/apply all exited0, but exact hashes differed solely
   by CRLF because the fixture omitted upstream `.gitattributes` while host
   core.autocrlf=true. Original scratch and result retained; normalized contents
   matched. Root authorized one fresh corrected replay including actual upstream
   `.gitattributes`; no global Git configuration changed. Corrected command and
   comparison JSON separately preserve24/24 exact matches.

These are pure/software and fake ownership results. Darwin/Linux backend fake
tests are compiled only. No real GPU driver, pending drawable, physical window,
VRAM/RSS, no-feature binary-size or product integration claim. Those require
later activation; implementation and dependency propagation are now frozen.
