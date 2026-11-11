# Prepare three fixed engine packaging sources

2026-11-09. You are the DeepSeek packaging source owner. Root only plans/reviews.
Read native/AGENTS.md first. Astra alone owns app/editor/graphview implementation;
GLM owns catalog/IPC evidence. Do not edit their files or dispatch child agents.
Context must be compacted before 200K tokens. Accepted .6 and prior evidence are
immutable. The new scene command and codec are source-accepted, not packaged.

Implement this narrow packaging source increment now:

1. Read core/enginecatalog/catalog.go and existing scripts/package-markdown-worker.ps1.
   Extend the fixed definitions with levnote.mindmap-preview, mindmap-scene-v1,
   engines/mindmap-preview, levnote-mindmap-preview.exe, ./cmd/levnote-mindmap-preview.
   All source/path/format inputs remain repository-fixed. Source cap16MiB and
   payload cap32MiB. Existing Markdown/html8MiB/64MiB and Mindmap/svg16MiB/64MiB
   stay exact. PayloadMaxBytes must be per descriptor, not the existing hardcoded
   64MB aggregate entry. Shared build, PE checking and manifest code stays shared.
   Retain script entry/parameters and existing exclusive fresh-directory guards.
2. Update native/portable/README.md to truthfully describe three one-shot workers:
   two export engines and readonly Mindmap scene preview. Native UI draws DTO; no
   WebView. Editable Mindmap and source outline parsing still run in host. Preview
   missing bundle/failure is visible, source mode available. No arbitrary executable
   plugin sandboxing or resource/copyright claims. Preserve version substitution,
   APPDATA path, local links, actual licenses and explicit unaccepted-build wording.
3. Update scripts/check-portable-package.ps1 only as needed for an independent
   nonexecuted fixture with all three fixed payload/manifest paths. Derive real
   inventory counts instead of stale prose claiming18shipped files. No fake test
   output. Do not change portable-package-lib.ps1 unless a concrete defect requires
   it; report before broader repair.

Before running anything, publish native/handoffs/native-preview-packaging-source.md
with exact changed source and static findings. Root will review. No Go builds,
package creation, product/worker launches, tests or evidence execution in this
first task. Read-only source inspection and formatting are allowed. Do not update
current launch/version in source README/VALIDATION or old bundle docs yet.

After handoff, wait for root's finite activation rather than expanding scope.
