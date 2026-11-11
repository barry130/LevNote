# Read-only preview application integration checkpoint

2026-11-09. DeepSeek owns only the new handoff
native/handoffs/native-preview-app-checkpoint.md. Read native/AGENTS.md active
scope and the dto-next-stage/contract/design-review. Root plans/reviews; Astra
alone implements core/enginescene, GLM only plans worker mapping. No overlapping
edits, tests, builds, launches, product changes or child delegation.

Read current ui/editor/contract.go, app/reference.go/editorHost, app/view.go,
app/app.go release/dispose/close paths, app/export_worker.go and core/engineruntime.
Write one concise actual-source integration checkpoint for later root review.

Decide a small optional neutral scene capability that keeps runtime/catalog/IPC
dependencies out of ui/editor. ID/version/hash and warning details can be copied
into a neutral bounded response. View Release can be followed by Draw on the
SAME existing view after tab/mode switch, so cancellation must support clean
reactivation without callback into an old epoch. Do not treat Release as permanent
destruction or leave retained resources until next user action.

Application owns preview admission/cancel/join and immutable buffer snapshots;
UI-owned tab/document/buffer/adapter/view epoch captured/rechecked on delivery.
Use existing ONE shared Runner. Export priority must cancel preview and await
its join off UI, reserve admission against next-preview races, then run export.
At most one active worker plus one newest pending snapshot; no busy loop or
unbounded completed-result retention. Close cancels/joins all jobs and blocks
no required dispatch. Nested modal callback deferral must remain safe.

Read-only pilot only; existing editable view/outline parse remain. Missing or
failed provider has visible error and reachable source path, no silent host
parse. Theme changes repaint without new request. First requested view scene
loading, stale replacement, selection, pan/zoom, source-mode/re-entry and
plugin/tab/buffer disposal must have exact transitions. Neutral DTO canvas may
be a separate type but avoid duplicating unrelated inspector/edit code.

Identify smallest source boundaries and meaningful fake-runner/real-native
tester oracles for scheduling/lifecycle/identity/outline, and later one actual
bundled request. Do not write executable code or a large abstraction proposal.
Report any conflict with current source. No need to audit every historic test.
Context cap200K; compact if approaching. Complete the finite handoff then stop.
