# Bounded local PNG/JPEG admission

ACTIVE2026-11-10 for GLM read_native_handoffs_drawio_worker_resources_harness_task_md.
Your matrix wrapper is handed to Astra; do not edit or execute it. Accepted
source/tester/package/evidence immutable. Own only new native/formats/raster/**
and this stage's new handoff/evidence. No children; compact before200K context.
Root plans/reviews only. .22 package accepted; source freeze ended.

Implement the small pure image-header admission prerequisite from
native-image-view-next-plan.md. This is not yet a reader/view/decoder integration.
Do not edit app, ui, vendor, document, session, plugins, package/build scripts or
product docs. No full product build/launch. Check for existing files first.

API should accept io.ReaderAt plus declared size (compatible with forthcoming
immutable binary snapshot); no filesystem access and no whole-input copy. Return
small metadata and explicit typed rejection reasons: PNG/JPEG, raw dimensions,
orientation if safely available, supported subset and derived payload budgets.
Hard limits: compressed16MiB, each side1..8192, pixels<=8,388,608, baseRGBA+all
ceil-halved mip levels<=48MiB; modeled decode payload<=128MiB. Checked integer
arithmetic. These are payload/admission limits, not RSS/driver guarantees.

PNG subset: <=8bit legal color/depth pairs, noninterlaced, valid signature/IHDR,
bounded chunk traversal/truncation/length checks; reject acTL/APNG and unsupported
modes explicitly. JPEG subset: 8bit baseline SOF0, grayscale or standard3component
sampling factors1/2; reject progressive/lossless/CMYK/unsupported markers/sampling.
Use a bounded marker walk before decode; never mistake entropy bytes for header
markers. EXIF parse must be bounded, overflow-safe, and orientation1..8; unknown
metadata is retained in source and need not be interpreted. Do not re-encode.
Read pinned Go codec allocation behavior to derive budget rather than guessed
multiplier; include input, decoder planes/MCU padding/rows, RGBA conversion and
orientation temporary. Prefer rejecting a case whose estimate is unknown.

This is preflight, not full image validation. Document that codecs/final bounds
must still validate the same immutable bytes and failures fall back visibly to
byte inspection. No generic image.DecodeImage/SVG/network or extra formats.

Begin with concise raster-admission-design.md listing exact API/limits/budget
formula and files; then implement this authorized pure helper without waiting
for another approval unless a material contract conflict appears. Meaningful
table tests: valid small PNG/JPEG variants, truncation, oversized headers and
length overflow, illegal color/depth, APNG, unsupported modes/components,
bounded EXIF and ReaderAt failures, precise budget boundaries. Standard encoders
may generate fixtures as codec primitives; logic is project authored.

Do not start Go tests while resource matrix runs. For now coding/tests are
authorized, but leave tests PENDING and ask root internally for an execution
window once code is ready. Root will avoid overlap; no user question. All Go
via native/scripts/go.ps1. One focused final package suite/vet after corrections,
save actual commands/output/exit/failures. No mirror tests or giant new framework.
Publish raster-admission-result.md and freeze source for root review.
