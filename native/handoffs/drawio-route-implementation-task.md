# Draw.io custom-suffix native route implementation

ACTIVE 2026-11-10 after `.19` binary and aggregate acceptance by root.
Implementation owner:
DeepSeek `/root/read_native_handoffs_resource_workload_next_task_md`.
No child delegation; compact before200K. Root plans/reviews only.

Implement the small product increment in `drawio-route-next-plan.md`: an
installed schema3 JSON format configuration may use exact `builtin.drawio` to
open a custom suffix (example `.diagramnote`) with the shipped native draw.io
editor and selected-page SVG worker. Preserve schema3/API1 and closed routes.

Own only `core/plugins/native_route.go`, `core/plugins/declarative.go`, relevant
plugin tests, `modules/modules.go`, `modules/native_route_test.go`,
`app/plugins.go` and focused app routing tests/capture test. Do not edit general
docs, examples, portable assets, accepted packages/evidence or export/graphview
production unless an actual blocker is identified and coordinated first.
`.19` documentation has a separate owner, so changed workspace source must not
be described as the accepted `.19` binary.

Keep all existing validation and registration-order/enable/persistence rules.
Existing unknown-route test uses `builtin.drawio`; change it to a truly unknown
route so refusal stays covered. Modules return a fresh adapter/capability set,
including RequiresView=true, and no invented Outline/Scene capability.
`declarativeCapability` currently promises outline for every route; use truthful
generic native capability wording (or real capability checks). Schema3 outline
rejection wording must not imply every routed format supplies an outline.
Also update the installation dialog's format list in `app/plugins.go` to include
draw.io; root observed it currently mentions only Markdown / mindmap. Audit
other touched route labels narrowly so the capability is discoverable.

Do not weaken helpers used by existing Markdown/mindmap tests: add a draw.io
helper or assert optional capability explicitly. Cover the following substantive
paths by reusing current test utilities, with independent content assertions:

1. Plugin decode/normalized extension and manager install/enable; exact route
   rejection remains strict. Module template is valid drawio from the existing
   sole template source. Fresh mutable capabilities and builtin enable state
   remain independent of installed configuration state.
2. New/Open and preview/source/split for custom suffix. BOM/CRLF/unknown XML
   metadata remain exact without edits. An unchanged SaveAs to another mapped
   suffix changes routing/path but not content. A readonly original is still
   readonly and can SaveAs a writable exact copy. Do not use mutable-source
   helper output as the expected disk bytes.
3. Selected second-page worker request carries exact index/ID after a real
   pending field Flush, with the declared single undo transaction. Source/no
   view visibly refuses. No need to repeat lower-layer actual worker execution.
4. Disable/rematch or same-ID refresh rejects a late old export and clears
   payload. Failed real conflicting graph draft Flush blocks the relevant
   disable/new/install action without losing the draft; reuse lower-layer
   coverage instead of duplicating a broad cross-product matrix.
5. Builtin drawio independently disabled does not disable the installed route;
   disabling that installed route returns safe editable source (or readonly
   inspection). An unavailable compiled adapter reports a visible fallback.

One focused development check as necessary, then one final affected suite and
vet after corrections: plugins, modules and app only. Preserve actual failed
attempts/raw streams/exit codes. All Go via native/scripts/go.ps1. Generate at
most one760x480Tester capture of actual installed drawio route panel if needed
for new capability wording, no physical Windows input/IME claim. No package,
host/worker launch or resource benchmark in this stage.

Publish `drawio-route-implementation-result.md` with exact changed files/hashes,
final evidence, stated inherited coverage and limitations. Freeze for independent
review. This task does not update docs to an unbuilt `.20` or activate packaging.
