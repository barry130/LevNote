# Native image reference action and app integration

PLANNED until markdown-table-columns-release25-root-acceptance.md exists and
your .25 delivery task is finished. Then ACTIVE for Astra
/root/read_native_handoffs_tab_workflow_resume_task_md. No child delegation.
Read markdown-image-reference-root-decision.md and DeepSeek's format handoff.

Own modules/markdownview/links.go,blocks.go,rows.go,view.go as needed, related
focused tests and new app/markdown_image_reference_app_test.go. No format/app
runtime/editor/core/imageview/raster/vendor/plugin/session or product docs edits.
Do not add a second image route. Existing Host.OpenReference is sufficient.

Implement clear explicit native image-reference controls using InlineImage
exact fields. Shared links+images budgets16controls/64spans; raw fallback retains
syntax. Distinct bounded accessibility label/tooltip; raw Markdown may remain
the visible control. Activation Ctrl/Cmd+click and Tab+Enter, source identity
includes inline kind. Preserve link selection/double-click/read-only behavior,
draft Flush/recovery, range mapping/reparse, source ownership, cancellation,
repeat/scroll/release safety. Do not perform I/O, decode, cache or automatic open
while drawing. Existing ordinary links remain behaviorally unchanged.

Meaningful view tests: image-only/mixed/dense rows, empty alt, Unicode/wrapping,
265DIP controls, rejected oversized literal, absent host, pointer/Enter path,
plain selection/double-click, readonly, composition/Flush conflict, stale target
and image->link mutation, prefix mapping, cancel/repeat/scroll/release/owner change.
Test actual behaviors rather than asserting helper output against itself.

Real app test from saved Markdown through registered native view action to
PNG/JPEG reader tab; test existing-tab selection (no duplicate), exact unchanged
source bytes/BOM/CRLF/undo, missing/unsaved/remote rejection, image plugin disabled
byte fallback. Use existing synthetic test interfaces and bounded generated
fixture. No physical OS input assertion, network, worker or product host runs.

One optional screenshot at265/300DIP only if needed to assess visible wrapping;
avoid duplicate captures. Save exact command/stdout/stderr/exit from first run.
After focused correction freeze and write
native/handoffs/markdown-image-reference-ui-result.md plus final source inventory.
Coordinate one final affected formats/markdown+modules/markdownview+app test/vet
after root reviews frozen implementations; do not independently repeat full
passing suites. Package delivery is a separate later task.
