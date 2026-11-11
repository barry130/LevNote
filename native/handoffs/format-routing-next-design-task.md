# Next slice design: installable format routes to fixed native capabilities

2026-11-09. GLM read_native_preview_ipc_task owns one design handoff only:
native/handoffs/format-routing-design-checkpoint.md. Current packaging source is
accepted/relinquished; do not edit it again. Astra owns opt-in preview delivery.
No product/test/script changes, execution, builds, network or child delegation.

Read current core/plugins, app/plugins, modules/contract and prior
plugin-next-slice-review.md. Refine that recommendation against the actual
three fixed worker capabilities, new readonly scene and existing lifecycle.
The next user-visible slice should allow a local JSON format declaration such
as .taskmd to use the shipped Markdown native view, or .thoughts to use the
shipped Mindmap editor/readonly scene, without rebuilding the host.

Keep it explicitly a format route to a shipped engine, not installation of new
executable code. Choose the smallest strict schema3 design. Do not duplicate
an independently configurable capability-bit list that can contradict the
selected adapter; derive fixed capabilities from the composition root. Select
one manifest route property with a small closed set, distinguish manifest ID
from renderer/export/scene engine IDs, reject arbitrary paths/commands/URLs and
unsupported fields. Schema1/2 bytes/semantics remain backward compatible.

Checkpoint should cover:

- Concrete JSON example, exact allowed fields/schema validation and source
  highlighting/outline behavior when a native route is declared. Keep native
  semantic outline authoritative; do not combine two incompatible outlines.
- Strict top-level key/casing/duplicate/null handling and bounded reads;
  registration tie order, installed-vs-builtin matching, disabling a route or
  builtin registration without destroying the fixed capability catalog.
- The minimal registry/composition/app API seams, specific file ownership,
  clear product-facing plugin label and missing capability/source fallback.
- Flush and reconfigure safety: no draft/undo/source loss, failed install or
  persistence leaves existing selection unchanged, scene/export cancellation
  on route changes; same manifest ID after refresh is a new generation.
- Templates/new files, extension-sensitive links, save-as/reopen/recovery,
  source/split/readonly/export eligibility and outline ID identity. Identify
  concrete path-based checks that would break custom suffixes before selecting
  implementation scope. Do not handwave this as only one configureTab change.
- Finite meaningful tests: strict admission/install rollback, real native
  adapter routing from a custom suffix, disabled fallback, round-trip
  BOM/CRLF/unknown metadata, Flush conflict, writable Save As, stale workers.

This is design only while the .7 consumer/delivery inputs are frozen. Publish
the concise source-grounded checkpoint; root will review and activate a finite
implementation after the accepted new bundle/doc stage. No broad redesign or
new engine process in this slice. Retain the honest remaining gap toward fully
independent editable document-engine packages.
