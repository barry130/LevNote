# Root review of preview worker checkpoint

2026-11-09. Root read worker-checkpoint against actual catalog, IPC, native
draw.go and pure presentation/layout/SVG sources. Minimal mapping is accepted
for the next source stage after codec evidence. Two finite clarifications:

1. The neutral DTO has no image-presence bit. Existing graph drawMindmap itself
   does not draw the SVG exporter’s per-node image badge, even though the shared
   PresentationIssues message says it does. Do not carry that inaccurate promise
   into this pilot. Keep IssueImagePlaceholder code/order/range, but the preview
   command must use a concise truthful message: image references are preserved,
   this read-only preview does not load/display images, full references are in
   source. No filename/URL/image data in DTO. No DTO change or accepted SVG/source
   behavior change required for this bounded pilot; future badge parity can be a
   separate explicit capability. Other warning messages remain unchanged.
2. `enginescene.Encode` has the production cap only. Do not add a public codec
   limit API solely for producer tests. Command output-limit can be verified by
   a valid constructed scene/source exceeding the real wire budget or a private
   injected encoder seam coupled with an actual codec refusal test. A post-hoc
   smaller check after full Encode is not a memory-admission test. Review exact
   oracles before claiming bounded command behavior. Fixed catalog lookup for
   descriptor/budget is allowed; no filesystem/runtime/bundle matching in worker.

Worker retains original parse/layout/presentation warning order and validates
all DTO limits before output publication; errors never contain source content.
No implementation is activated by this review alone. Code/source evidence
activation follows root acceptance of the independent codec package.
