# Fixed preview catalog/IPC implementation task

ACTIVE2026-11-09. Root accepted actual frozen codec source and12PASS/vet0 evidence;
independent review continues but does not block this isolated addition. You own
ONLY native/core/enginecatalog and native/core/engineipc plus new scoped acceptance
evidence and native/handoffs/native-preview-ipc.md. Astra owns preview command;
the other GLM only reviews codec. No child delegation or overlapping edits.

Read native-preview-worker-next-stage.md, worker-checkpoint and root worker-plan-
review. Add only MindmapPreview fixed catalog ID levnote.mindmap-preview, format
mindmap-scene-v1, protocol1, path engines/mindmap-preview, filename
levnote-mindmap-preview.exe, windows/amd64, source16MiB via existing constant and
payload32MiB via enginescene.MaxPayloadBytes. IPC adds alias and reuses generic
request/response logic. Keep old descriptors/ReadRequest Markdown-only wrapper.
No manifest semantics/runtime/process/UI/command/dependencies/script changes.

Meaningful tests: exact literal descriptor and cross-format/platform/id refusals;
preview valid request/response fragmentation and hard budgets; cross-operation
metadata refused BEFORE source bytes read, source header cap refused before
metadata/payload, response cap/identity refusal before payload retention. Check
the old Markdown wrapper refuses preview as expected. Reuse narrow helpers where
available, independent wire literals and counting reader/panic suffix oracles.
Do not just encode→decode same implementation and call it independent acceptance.

Publish source-ready note in handoff promptly so Astra command can compile. Then
run full catalog/IPC -count=1 and scoped vet through native/scripts/go.ps1; only
these packages. Record actual arguments/stdout/stderr/exits/times and small
source/module/wrapper before-after inventory in fresh evidence. Formatting before
inventory. Preserve failures; correct concrete owned defects only, no blanket
reruns. Root reads source/oracles/logs. No packaging/build/worker launches.
Use normal permission profile with no sandbox_permissions. Context cap200K.
