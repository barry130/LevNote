# Next reading increment: inspect and propose a bounded implementation

ACTIVE2026-11-10. Owner GLM /root/read_native_handoffs_binary_image_glm_task_md.
Read-only source/design review plus this task's handoff only. No runtime/docs
edits or product operations while Astra builds .25. No child delegation;
compact before200Kcontext. Prior .24 docs are accepted and finished.

Review current native reading flows against the user's central need: one small
local reader/editor with plugins. Focus on the next useful missing capability
after image reading and table columns. Prefer a bounded next step improving
everyday document reading rather than adding visual effects. Inspect actual
Markdown image/link handling, existing imageview/raster/document contracts and
open/reveal route. Mature-software conventions may guide your reasoning; do not
claim that you inspected external software unless you actually did so, and do
not copy an external editor engine.

Decide whether the strongest safe next increment is opening a local Markdown
image/reference through the current app reader (if missing), or bounded inline
local image display using existing project decoder (if opening is already
implemented). Also compare one alternative from source navigation/file workflow
or engine-plugin usability. Trace current code first; no duplicate image decode
stack, network I/O, unbounded cache, unsafe path rewrites or automatic source
modification. Examine memory ownership and stale draft/async publication risks.

Write native/handoffs/native-next-reading-review.md: actual current behavior
with file/line references, chosen narrow next stage, user-visible behavior,
owned files/contracts, invariants/bounds, focused meaningful validation and
remaining unknowns. Be concrete and short enough to implement. This is planning,
not permission-seeking. Do not start implementation until root assigns scope.
