# Native preview engine: finite design checkpoint

Status: DESIGN ONLY, 2026-11-09. Owner glm_alignment_review. Root has read your
completed independent integration review. The second export bundle is still
awaiting command tests and packaging. This task may proceed independently but
does not activate implementation, test execution, build or launch.

Read native/AGENTS.md, ui/editor/contract.go, current graphview and markdownview
read/paint paths, enginecatalog/engineipc/runtime and their existing contracts.
Write only native/handoffs/native-preview-engine-design.md. No child delegation,
product/test/script/docs edits, network or execution. Compact before 200K.

Recommend one finite read-only parsed/presentation-model operation that makes
an actual native preview path consume a separate engine response rather than
invoke the concrete parser in the host. Mindmap is a plausible first candidate
because its pure layout/presentation just became shared; choose based on actual
source dependencies. Do not imply that export-only or a manifest alias already
fulfills the user's independently packaged document-engine requirement.

Keep the checkpoint concrete and short (approximately 150 lines maximum):

1. User-visible first path, exact before/after import or ownership boundary and
   why it advances the migration. Decide explicitly whether only read-only
   documents or all preview modes enter it; editing/drafts must stay intact.
2. Minimal bounded response DTO (numeric geometry, labels, literal paints,
   node/edge identities, diagnostics), source identity/stale result rules,
   selection/pan/zoom, refusal/fallback/retry and release/cancel ownership.
   No HTML/WebView, renderer code, source duplication or arbitrary executable
   loading in a DTO. State count/string/byte/depth/finite-number validation.
3. Identity/capability transition: present catalog identifies an export pair
   and one executable per pair. Recommend one sustainable extension, preserving
   existing v1 Markdown/Mindmap export bytes. Explain whether the existing
   Mindmap binary serves another operation or why a separate binary is needed.
   Do not design a general scripting framework or duplicate the process runner.
4. Scheduling and memory: on-demand immutable snapshots, one active worker,
   cancellation/coalescing of superseded versions, bounded retained model and
   no worker-per-paint or hidden fallback parse. Distinguish design limits from
   measured memory; input/output caps do not bound RSS.
5. Small implementation stages with nonoverlapping files and meaningful tests:
   independent DTO literal/invalid cases, native consumer/stale/close path and
   one actual bundled request. Existing editing/export coverage should be reused.
6. Remaining gap to independently installed editable format engines and what
   this stage deliberately leaves to later migration. Do not claim completion.

Publish the design and report complete. No wait for the export package is needed
for this read-only planning task. Root selects the next slice after delivery.
