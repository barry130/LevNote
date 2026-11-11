# Next product increment: data-only draw.io format route

ACTIVE 2026-11-10 for GLM
`/root/read_native_handoffs_drawio_worker_next_plan_task_md`. Root permits this
read-only plan in parallel with Stage2 implementation. Inspect stable plugins/
composition and accepted view contract, do not assess in-flight app edits as
final. No implementation/tests/build/product docs yet, no children.

## Loop recovery

The prior attempt repeatedly read `drawio-worker-stage2-task.md` eight times and
was stopped by the tool loop guard. Do not repeat that read. You already have
Stage2's relevant contract: drawio advertises `ExportEngine{ID:"levnote.drawio-svg",
Format:"svg",RequiresView:true}`, page metadata from mounted preview/split only.
Read at most once each: core/plugins/native_route.go, declarative.go validation,
modules/modules.go, app/plugins.go and relevant existing format_routing tests.
Then write the short concrete plan directly. Do not poll for Stage2 completion;
this is an independent read-only plan and can finish now. A tool error is not a
reason to rerun an identical request: use already-read source/context, describe
any material unknown, and finish named handoff. No more than two adaptive reads
before the first plan draft; refine only if a new fact requires it.

User wants supported file formats managed as plugins. Schema3 currently maps
local custom extensions only to builtin.markdown or builtin.mindmap. Fourth-worker
integration gives drawio a neutral view-dependent export contract. Plan a small
next increment permitting exact builtin.drawio in the same closed data-only
schema3 route, preserving plaintext fallback, registry order and explicit enable
state. Don't invent a new schema, arbitrary code execution, format converter,
unbounded plugin dependency graph or global routing subsystem.

Read actual plugins/native_route.go and declarative validation, modules composition
root, app plugin matching/template/lifecycle, format_routing tests and new drawio
export integration. Identify exact files and minimal tests: strict route decode,
unknown route rejection kept, custom extension gets drawio template/native view/
RequiresView export, source-mode refusal, new/SaveAs route rematch, disable returns
source safely, failed draft flush blocks install/disable/new/rematch as existing
routes. Confirm duplicate/fallback policy and no unnecessary parser duplication.
Default generated drawio template and all unknown data preserved. Existing
nativeRoute label comes from factory, avoid format-ID conditions in app.

Write `drawio-route-next-plan.md` with concrete design and focused verification,
drawio-specific meaningful risks/limits, minimal portable/example/user docs and
stage sequencing. Read-only; no change to .19 accepted evidence or delivery.
