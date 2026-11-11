# Focused picker review

ACTIVE2026-11-10. Owner DeepSeek
/root/read_native_handoffs_live_tab_admission_implementation_task_md. No children,
no edits except this review's handoff, no tests/builds/launches. Astra solely
owns app implementation. Review open-document-picker-next-plan.md and current
open-document-picker-implementation-task.md against actual public MyGo APIs and
app lifecycle. Do not create another broad audit or blocking preparation stage.

Focus on concrete reachable blockers: modal dismissal before child callback;
query composition/focus loss; KeepFocus on backdrop and row actions; opener
restoration when failed Flush; post-Flush stable ID validation/reentry; bounded
metadata without document retention; virtual List callbacks/close cleanup.
Read actual source as it appears. Report precise paths/conditions, distinguish
static concern from observed test result. No invented behavior and no speculative
requirements outside the task. Send root concise findings and save
open-document-picker-independent-review.md. If implementation is not yet ready,
save a brief contract checkpoint and stop; root will activate final source review
after implementation is frozen. No repeated polling or testing by reviewer.
