# Finish bounded raster admission

ACTIVE2026-11-10 for fresh DeepSeek owner. Previous GLM raster owner interrupted
and has no ownership. Root plans/reviews; you implement and run checks. No
children; maximum context256K, compact before200K. Workspace F:\AIproject\LevNote.
PowerShell, unrestricted filesystem/network but runtime product purely local.
All Go MUST use native/scripts/go.ps1 -GoArguments @(...), never system Go.

Own only native/formats/raster/** and new evidence/handoffs. No app/document/
session/vendor/scripts/product-doc files. Concurrent owners: GLM binary core,
Astra pinned vendor disposal patch. No product build/UI/worker/resource launch.
Focused package tests/vet authorized; prior fixed12 resource run ended.

Complete the existing pure ReaderAt PNG/JPEG header admission helper now.
Read raster-admission-task.md, raster-admission-design.md, and ALL of
raster-admission-initial-review.md. They describe the exact subset/budget and
known correctness problems. Read actual source and tests before changing.
Design direction is accepted; no new design-only phase needed.

Production has partially incorporated review. Verify these facts carefully:
- PNG<=8bit/noninterlaced,16MiB compressed, side<=8192,pixels<=8388608;
  walk chunk headers through IEND, bounded4096 chunks, reject acTL/fcTL/fdAT,
  TIFF eXIf and tRNS accounting. Require valid small envelope/order or reject
  visibly; no whole IDAT input copy. This is preflight, not CRC/decode proof.
- JPEG SOF0 only, grayscale/standard3components H/V1/2 tracked separately,
  equal chroma, luma maximal; header walk stops at SOS, bounded total work;
  EXIF is six-byte Exif\0\0 prefix then TIFF, bounded IFD0. SOS small envelope.
- Budget exact padded planes.4:2:0 16x16 MCU=256+64+64=384bytes; recent source
  fixed earlier missing-Y-plane bug but independently verify. Keep codec RGBA+
  viewer/EXIF conservative allowances as documented, exact ceil mip sum/caps.
  Read pinned native/tools/go/src/image/{jpeg,png} allocation paths if needed.

Main current blocker is INVALID TEST FIXTURES, not permission/tools:
- insertJPEGSegment currently locates marker then inserts AFTER its2bytes,
  includes only new4byte header and forgets actual payload. Insert a complete
  marker+length+data BEFORE target marker using a checked marker walk (before
  SOS, never entropy). Do not change preflight to scan entropy to satisfy tests.
- insertPNGChunk doesn't compute CRC despite claiming valid fixture; add actual
  CRC32 and preserve all original chunks/IEND. replaceIHDR must update intended
  depth[8]/color[9]/interlace[12] and CRC if validity claimed. Header-only bad CRC
  fixtures can be deliberate but label them, not accidental.
- Tests still contain wrong payload formulas/row sizes/mip boundary, malformed
  minimal SOS{0}, big-endian IFD count high-byte bug, append-after-IEND metadata,
  encoded variants with unverified real IHDR color/depth. Fix them using known
  independent expectations and assert actual relevant header before test.
- Do not weaken assertions or widen subset just to get green. Test real standard
  encoder fixtures and adversarial focused cases. Explicit independent MCU plane
  expectations for each ratio, odd size/common factors; orientation public Admit
  through real PNG/JPEG metadata, short boundary/no panic, reader failure,
  compressed/dimensions/overflow,APNG,unsupported modes,declared size/traversal.

Previous iterations ran several failing checks without durable output capture;
raster-admission-checkpoint.md lists history. Preserve existing files/results;
do not fabricate old logs. State gap once. New evidence folder:
native/handoffs/raster-admission-final-evidence-20261110 (create if absent;
if exists inspect first and add numbered files, never overwrite failures).
Use full absolute output paths since wrapper changes cwd. Capture exact wrapper
arguments, stdout/stderr and actual exit for every new run. First fix the known
fixture/code issues by inspection, then run focused package tests and correct
actual failures; one final full raster package test/vet after corrections.
No repeated unrelated suites/benchmarks. Do not create large check framework.

Freeze raster-admission-result.md with actual API/limits/formulas, files SHA256,
commands/exits/log locations, retained failures and limitations. Update design to
final behavior. After root accepts, next task is a bounded decoder in same
package (raster-decode-next-plan.md), but do not start it yet.
