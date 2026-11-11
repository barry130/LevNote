# Finite independent source review: shared engine lane and preview ownership

2026-11-09. GLM transport owner is now read-only reviewer. Root accepted minimal
transport source/tests; do not modify/retest that stage. Astra is implementing
the readonly scene consumer and owns all product/test files. DeepSeek owns
packaging sources. No child delegation or build/test/launch execution here.

Read `native-preview-app-source-checkpoint.md`, `native-preview-app-implementation.md`
and actual current files: app/engine_lane.go, preview_worker.go, export_worker.go,
reference.go, app.go, view.go, workspace.go and ui/editor/{contract,scene}.go.
Review this finite slice now while Astra continues rendering. Publish findings
promptly to `native/handoffs/native-preview-consumer-independent-review.md`;
do not wait for every scene view file or polish an extensive report.

Find concrete correctness defects with exact source location and reproducible
interleaving. Focus at most on shared one-runner exclusion, preview preemption
and resumption, export reservation through modal/publication, latest snapshot
ownership/cancellation, done/Close independence of UI dispatch, poisoned cleanup
outcome, identity recheck, and callback/large-result lifetime. Confirm the actual
MyGo Invalidate/Update queue behavior if relevant. A concern requiring unobserved
behavior must be labeled a question, not claimed a bug. Direct findings go to
root; no edits or assignments to other agents.

Astra's files are actively written. Record source size/hash or read time; distinguish
incomplete wiring from a defect in completed code. No unrelated style feedback,
generic recommendations or repeated frozen-code review. Context compact before200K.

If no actionable defect, explicitly say this limited reviewed slice has no finding;
do not claim final integration/visual/process approval. Then remain idle.
