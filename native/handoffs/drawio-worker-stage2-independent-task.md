# Stage2 independent review: current-page native host export

ACTIVE2026-11-10 for fresh GLM
`/root/read_native_handoffs_drawio_worker_stage2_independent_task_md`. Original
stage2 result/production frozen; narrow test followup active on app owner, see
drawio-worker-stage2-test-followup-task.md. Review production now; reconcile final
test handoff when available. Root reviews independently. No children, product
edits or repeated tests/builds/launches. Review actual final source+test+capture
evidence; source is untracked, use named files not empty git diff.

Accepted lower layers and view contract don't need another broad re-review.
Prioritize new app capture and lifecycle: active tab/doc/buffer/editor/adapter/
capability/name/view/mode immutable identity; Flush may commit one intended draft
transaction but can't retarget; source/group gate before Prepare/snapshot;
post-Flush version fixed across Prepare; page copied before async queue; getter
pure; pageA->B->A, release/reparse/view/mode change refuses final publication.
Check both pre-dialog and post-dialog rechecks; failed Flush preserves view and
draft/history, stale result payload released. Existing lane reservation, modal
deferral, cancellation/Close/poison/cleanup semantics retained. Old3 export and
scene requests have nil page and no mounted-view dependence. No silent
synchronous drawio fallback once adapter advertises worker.

UI requires source/nil view instruction at760x480, valid read/edit or split
current-page export, fourth fixed engine selector with accurate labels/count
neutral reset. Import doesn't activate; malformed/poison state still blocks.
Capture claims match actual images and method tests; no physical input/IME claim.
Check final tests include substantive exact second-page+postFlush source capture,
state reentrancy, stale mode/page/token after dialog, disk/BOM readonly/no unwanted
history mutation, successful exclusive publication and original worker regressions.
Avoid duplicating accepted graphview cases already covered except wiring.

Report only concrete defects/triggers or scoped no-blocker conclusion. Missing
cosmetic coverage isn't a blocker. If fixture/final hashes changed, distinguish
authorized correction from unexpected drift. One narrow repro only if source
can't settle a substantive concern; request owner fix through root. Publish
`drawio-worker-stage2-independent-review.md`. Stage2 source acceptance precedes
portable assets and actual .19 delivery; do not self-build or self-release.
