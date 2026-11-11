# Fourth fixed worker, stage 2: native host integration

ACTIVE 2026-11-10 for DeepSeek
`/root/read_native_handoffs_resource_workload_next_task_md`. Root accepted stage1
and view source for integration; final independent view review remains required
before package delivery. Root plans/reviews; no children; compact before200K. No
new release build/launch until root accepts integrated source. Existing .18
package/evidence and accepted SVG emitter/oracle stay immutable.

Read stage1 result, warning correction and root acceptance, original plan/
refinement and stage1 task overrides, plus `drawio-worker-stage2-preflight.md`.
The neutral contract and graphview implementation have ALREADY BEEN implemented
in the isolated view-contract stage, including subsequent reentry correction.
Read their final result/acceptance. Reuse `ExportPage{Index,ID}`, numeric
`ExportToken`, `ViewExportRequest{Page,Token}`, `ExportRequestPreparer` and
`ExportRequestTokenProvider`; `ExportEngine.RequiresView` already exists.
Do not reimplement these layers or broaden their surface. No IPC/runtime import
in editor or concrete graphview inspection in app. Only drawio advertises
RequiresView and fourth fixed capability. Old3 requests/behavior stay unchanged.

Graphview owns epoch. Invalidate on Release, real reparse/cache invalidation,
actual page change (including draft-driven selection and A->B->A). Cache hits,
unchanged Draw/Warnings and selection unrelated to export do not churn epoch.
Inventory actual assignments. Prepare can synchronize stale cache after caller
size check; no second Flush/publication and no draft discard. Reject missing
buffer/parser/page/overlong ID/invalid index/group; return copied exact page
index+ID+token. Reuse parser limits, do not reserialize/isolate a page. Inactive
source-mode retained views cannot authorize an export. View objects are reused
after Release in this app, so don't permanently disable a view with a dead flag.

App export capture:

1. Check capability/mode and current tab before work. Source mode (even nonnil
   retained View) must visibly refuse with instruction to open read/edit or split
   and select a page. Button disabled with meaningful label/tooltip; keyboard or
   method entry must also refuse. Avoid causing all non-graph exports to depend
   on a mounted view.
2. Capture existing tab/doc/buffer/editor/adapter/name/capability/view identities;
   Flush; recheck because callbacks may recover/retarget. Guard group and source
   byte budget before new source materialization. Existing flush may itself
   parse a pending draft; don't falsely claim no allocation before Flush.
3. For view-required export, save post-Flush version, call Prepare, then recheck
   all identities/mode/version/group and token. Never silently accept a version
   change during preparation. Current preview/ split may lazily mount a missing
   view only via normal host lifecycle; don't use a retained source view.
4. Copy page metadata before async enqueue; snapshot captured final buffer.
   exportJob stores doc/buffer/view/token/mode and existing generation/capability.
5. Result recheck and post-save-dialog recheck both validate current identity,
   epoch/version/mode, Flush and validate again. PageA->B->A or release/reparse/
   source transition/inactive tab cannot publish. No side-effectful preparation
   in final pure validation. Failed Flush preserves draft/history/view.
6. Keep lane/cancel/timeout/modal-depth/close/poison semantics. Old requests Page
   nil. Drawio uses32MiB source error text, exact page/index/ID. Exclusive atomic
   export publication unchanged. No new source fallback or silent first-page.

Expose fourth fixed engine in existing package panel and accurate reset labels.
Use four (or count-neutral wording), preserve explicit import/activation, all
selection migration/poison/retention behavior. Update relevant tests/captures,
not product release docs yet. Inspect narrow760x480 light/dark layout including
fourth engine choice and source-mode message; keep controls simple/no animation.

Own modules/graphview/adapter.go engine advertisement and integration tests,
app/** relevant export/engine-panel/tests. Accepted editor contract and graphview
sync/epoch/prepare implementation stay frozen unless you report a concrete
integration defect and root transfers ownership. No core/protocol/worker/script
edits unless specific stage1
integration defect is reported and ownership transferred. No format emitter
changes. Existing synchronous View.Export remains for format API/regressions,
but app drawio route must use advertised worker exclusively; no silent fallback.

Decisive behavioral tests: exact flushed second-page metadata+source after
limits; before-source/launch size refusal; pageA->B->A; same-index reparse;
Release retained pointer; source mode; draft failure; preparation reentrancy;
post-dialog page/version/view change; successful exclusive publication; old3
Page nil and original request behavior; unchanged original disk bytes/BOM,
buffer history/dirty/selection absent intentional draft Flush. If Flush commits
pending edit, assert intended single transaction rather than falsely requiring
history unchanged. Invalidated result frees payload and does not write a file.
One focused graphview/app suite/vet after corrections; no giant matrix or
physical Windows IME claim. Two substantive Tester captures for changed UI.

Publish drawio-worker-stage2-result.md with actual code/test/capture identities,
commands/logs/exits and limits. Freeze for independent review. Fresh .19 delivery
will then build host+4 workers and perform one actual managed drawio second-page
request with joined cleanup, plus normal package/host checks. Do not self-accept
or run delivery early.
