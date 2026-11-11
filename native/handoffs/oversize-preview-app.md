# Oversized rich-preview app acceptance

## CURRENT — focused app checkpoint accepted, final integration authorized

Root visually accepted the regenerated `20261108-150938` graph image after
Astra's inspector clipping/scroll correction. The app focused gate is green and
product source is frozen for this stage. The app change is test-only, in
`app/oversize_preview_app_test.go`; no production shell/view guard was added.

Exact focused commands and results:

- `native/scripts/go.ps1 -GoArguments @('test','./app','-run','TestOversize','-count=1','-timeout','180s')`
  -> PASS (`ok levnote.local/native/app 2.545s`).
- `native/scripts/go.ps1 -GoArguments @('test','./app','-count=1','-timeout','240s')`
  -> PASS (`ok levnote.local/native/app 4.179s`).
- `native/scripts/go.ps1 -GoArguments @('vet','./app')` -> clean.
- Screenshot gate with
  `LEVNOTE_OVERSIZE_SCREENSHOT_DIR=F:\AIproject\LevNote\native\artifacts\acceptance\oversize-preview-20261108-150938`
  and `@('test','./app','-run','^TestAcceptanceOversizePreviewScreenshots$','-count=1','-timeout','180s','-v')`
  -> PASS (`ok levnote.local/native/app 1.384s`).

The app tests exercise real source -> preview/split transitions for Markdown,
mindmap and drawio, writable versus readonly actual source input, real Markdown
Compose above the preview limit, both real graph kinds with a retained Compose
draft and failed mode switch, recovery after shrinking, and graph-draft
recovery into a new document. They assert the authoritative buffer
identity/version/selection/history/source survive, stale old rich content is not
visible, repeated frames retain the mounted view, warnings expose the source
route, and shrinking resumes export/rendering.

The oversized export assertion calls `exportCurrent()` with an empty status and
requires the preview-limit error to be visible. It does not fabricate
`Document.Path`, create a sibling export directory, or claim `os.ReadDir` proves
global no-write behavior. `app/export.go` was source-reviewed: `View.Export` is
checked before `SaveDialog`, and export publication is unavailable on the error
path; the existing core tests retain the no-output guarantee.

Narrow `760x480` artifacts:

- `native/artifacts/acceptance/oversize-preview-20261108-150938/oversize-markdown-source-route-760x480.png`
  — `39,260` bytes, SHA-256
  `28983E9737746F5A3DFBAFD5DBE4BB7D6FAE7D7574CC424F32E040FAFC5B9D9B`.
- `native/artifacts/acceptance/oversize-preview-20261108-150938/oversize-graph-draft-recovery-760x480.png`
  — `55,355` bytes, SHA-256
  `7E24D8F29F7D734EDD527636D898CA54D8E00D8EC49494723CC106364157C293`.

Positive-size, on-screen geometry assertions (status row begins at y=451):

- Markdown `源码`: `ui.Rect{X:579, Y:96.74487, W:26, H:16.510254}`.
- Markdown `分屏`: `ui.Rect{X:641, Y:96.74487, W:26, H:16.510254}`.
- Graph `重试写回`: `ui.Rect{X:575, Y:403.37622, W:44, H:13.970215}`.
- Graph `恢复为新文档`: `ui.Rect{X:666, Y:403.37622, W:66, H:13.970215}`.

Root accepted the two 150938 images. No focused app failure remains. The
accepted schema2 14:43 binary/screenshots/smoke remain historical for this
stage.

## CURRENT — final oversized-preview integration checkpoint

After the focused checkpoint above was green and root authorized one final
integration run, the exact commands completed as follows:

```powershell
native\scripts\go.ps1 -GoArguments @('test','./...','-count=1','-timeout','300s')
native\scripts\go.ps1 -GoArguments @('vet','./...')
native\scripts\go.ps1 -GoArguments @('build','-trimpath','-buildvcs=false','-ldflags','-s -w -H=windowsgui','-o','bin\levnote-native.exe','./cmd/levnote')
```

- Full test: PASS (7.165s command wall time). `app` passed in5.104s;
  all listed core, format, module, editor and dependency-patch packages passed.
- Full vet: PASS with no diagnostics (1.681s command wall time).
- Release GUI build: PASS (3.573s command wall time).

Final executable:

- `native\bin\levnote-native.exe`;
- bytes `13,209,600`;
- SHA-256
  `36CDC562D0E3C7F1793EDC935154B3AD7FF726716FCC409A736B04BD0FCAF3C6`;
- UTC mtime `2026-11-08T07:12:01.8261018Z`
  (`2026-11-08T15:12:01.8261018+08:00`);
- `go1.27.3`, Windows amd64/v1, CGO enabled, `-trimpath=true`,
  `MyGo v0.8.3`.

Hidden isolated startup/graceful-close evidence:
`native\artifacts\acceptance\startup-smoke-oversize-preview-20261108-151201\`.
The process was launched with `Start-Process -WindowStyle Hidden`; PID-scoped
EnumWindows found title `轻笺 · 原生版` while `IsWindowVisible=false`; the
script posted `WM_CLOSE`; the app exited `0`. Both updater-shaped file and
directory sentinels beside the copied executable survived with contents
`preserve-file` and `preserve-directory`. The copied binary SHA-256 equals the
release binary SHA-256. Exact fields are in the smoke `result.json`.

No material failure remains. Product source remained frozen for this final run;
the app change was test-only.

## ACTIVE — app lifecycle checkpoint

Acknowledged 2026-11-08. Root accepted the schema2 source-plugin checkpoint at
14:43 and activated `oversize_preview_stage_plan.txt`. That accepted
binary/screenshots/smoke evidence remains frozen and will not be rerun or
relabelled for this stage.

Owned app scope:

- Consume Astra's authoritative parser-bound constants and view refusal/error
  behavior without importing concrete parser internals into `app/**`.
- Add focused app-level lifecycle tests for Markdown, mindmap and drawio:
  source -> preview/split refusal, preservation of source bytes, buffer
  identity/version, selection, undo history, writable editing and readonly
  source navigation.
- Verify that a previously valid rich view is not displayed or exported as
  current content after the same buffer becomes oversized, and that a refusal
  never publishes a partial/empty export target.
- Verify a failed `Flush` retains the tab, view/controller, pending draft,
  selection and source. A successful Markdown range-only flush may remain
  available above its preview limit, while an oversized graph flush must fail
  before materializing/copying the source.
- Verify repeated unchanged frames remain bounded; shrinking or replacing the
  buffer with within-limit content recovers; reload/new-buffer changes cannot
  reuse a stale refusal or stale rich view.
- Keep installed source-only schema2 tabs outside all rich preview/split/export
  paths.

Execution gate:

1. Wait for Astra's exact compile/UI/error contract checkpoint and inspect the
   resulting owned view changes.
2. Implement only demonstrated app lifecycle/composition behavior in
   `app/**`; do not edit `modules/markdownview/**`,
   `modules/graphview/**`, parser packages or the neutral editor contract.
3. Run focused `./app` tests and vet, then publish exact commands, failures,
   API assumptions and remaining risks here for root review.
4. Run one full native test/vet/build and isolated startup smoke only after
   root freezes the focused source checkpoint.

At this acknowledgement, Astra's implementation is active and no app product
edit has been made speculatively. The 14:43 artifact is historical for this
stage and is not current final-build evidence.
