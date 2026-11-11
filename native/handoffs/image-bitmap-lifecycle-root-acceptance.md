# Terminal bitmap disposal source accepted

2026-11-10 root review. Accepted as the pinned dependency prerequisite for the
native reader. Root read actual UI/scene/raster/GPU retirement paths, backend
checkpoints and fake regressions, provenance/guard/asset wiring, saved test and
compile evidence, corrected pristine replay. All35 frozen-file size/hash entries
match current files. No root rerun of checks.

Windows focused five-package tests/vet pass. Windows amd64, Darwin arm64,
Linux amd64 and js/wasm package compile checks pass; Darwin/Linux backend fake
tests compile only. Final Metal empty-pending behavior has its own successful
compile. Build guard42/42 and portable asset fixture25/25 pass. Corrected
isolated pristine replay includes actual upstream .gitattributes and matches
all24 source files exactly after both patches. Original parameter-binding and
CRLF fixture failures remain recorded; active vendor was never regenerated.

Public Dispose is UI-thread-only, terminal and idempotent for owned bitmaps;
borrowed Canvas returns false. No background disposal from decoder workers.
Bitmap-owned CPU/source/remake references end immediately; independent renderers
drop retired textures/derived caches at next drawable opportunity or Release,
before no-damage skip. Idle/minimized and in-flight driver lifetime remains a
real limitation, not an immediate RSS/VRAM promise. Existing pixels held by
external aliases/captures are outside disposal ownership.

Patch47176B SHA256
995d5505c3b5b1bd869547df3f1ad5954c2dd57f1c9f32e53554851b5db0d4d1
Manifest4463B SHA256
9113120bd2ebcd9a86842153239cddff84d1f1bd3f46f5d4f0f152eb4560e16a
Full identity authority:image-bitmap-lifecycle-frozen-files.json
2b1d185cb3302427e692b0fe955ce2d0aac0bab8f559775b8e522b56bca10e8a

No product package was rebuilt, no real GPU/window/resource run occurred, and
the accepted .22 package is unchanged. Do not repeat these suites absent a new
source concern. Future integrated reader validation is separate.
