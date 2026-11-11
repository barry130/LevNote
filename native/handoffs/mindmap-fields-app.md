# Mindmap fields — app acceptance

## Final integration checkpoint — 2026-11-08 18:16–18:19

Root accepted the focused gate and froze product source. The authorized single
full integration sequence then passed without a later product edit or rerun:
full `go test ./...` 9.738s, full `go vet ./...` 2.358s, and the stripped GUI
build 3.207s, all with exit code 0.

Current artifact:

- `native\artifacts\acceptance\mindmap-fields-20261108-181628\`;
- `levnote-native.exe`, 13,274,624 bytes, SHA-256
  `F06ABEF11653EF93F03D3017A62215CACD8364536E9EDF900242DC9283950A6C`,
  UTC mtime `2026-11-08T10:17:14.9536389Z`;
- root-accepted integrated images
  `mindmap-fields-app-normal-760x480-dark.png` and
  `mindmap-fields-app-conflict-760x480-dark.png`, copied with their exact
  hashes into `result.json`;
- the previously accepted Markdown-links executable and all command logs,
  metadata and provenance retained in the same directory.

The isolated hidden startup smoke is
`native\artifacts\acceptance\startup-smoke-mindmap-fields-20261108-181739\`.
PID 21964 passed hidden launch, PID-scoped `EnumWindows` discovery and
`WM_CLOSE`, exited 0, preserved both unrelated sentinels, and left no
`levnote*` process. The app window remained non-visible after hidden launch.
This is hidden-process acceptance, not visible rendering coverage.

The final aggregate reports `FinalStatus=PASS`. Its package-summary derivation
and one environment field were corrected from already captured logs and
current machine facts only, with that evidence-only change recorded in the
JSON. Corrected `result.json` SHA-256 is
`C15825065372F36735D68CA5E609F9829D22336E73B72A3EF7858222E340F47F`.
`evidence-validation.json` rechecked the untouched metadata/logs and all
product/smoke hashes and reports 9/9 checks PASS; its SHA-256 is
`74D39C33E9FA4BE1EC0146B7FD15EEDFB3AD4A61FB9C307B491060B07B4D4EB4`.
No full test, vet, build, product test or resource sample was repeated. Root
artifact acceptance remains pending.

## Current focused checkpoint — 2026-11-08 18:12

The real-adapter focused suite is green. `app/mindmap_fields_app_test.go` uses
native editor `selectAll` + `copy` to verify the reopened Unicode/CRLF note;
`Tester.Texts()` is a paint-label inventory and intentionally cannot observe
the independent editor's `Painter.TextLayout` content. Astra confirmed the
same limitation, so this was a test-observation correction rather than a
product defect.

Exact lifecycle command:

```powershell
.\native\scripts\go.ps1 -GoArguments @(
  'test','./app',
  '-run','^TestMindmap(FieldRealAdapterDiskSaveAndReopen|PendingLinkWriteOpenSurvivesSynchronousTabRelease|PendingNoteAcrossModeTabAndClose|MissingAndRemoteLinksPreserveSource|ReadonlySaveAsCreatesEditableFieldView)$',
  '-count=1','-timeout=120s','-v'
)
```

Result: PASS in `1.933s`, covering disk save/reopen; pending write-and-open
through synchronous tab release; pending note through mode/tab/close with one
commit and unknown metadata retained; missing/remote refusal with source and
tab identity preserved; and readonly Save As followed by real field editing.

Exact screenshot command:

```powershell
$env:LEVNOTE_MINDMAP_FIELDS_APP_SCREENSHOT_DIR = 'F:\AIproject\LevNote\native\artifacts\acceptance\mindmap-fields-20261108-180356'
.\native\scripts\go.ps1 -GoArguments @(
  'test','./app','-run','^TestMindmapFieldsAppNarrowScreenshot$',
  '-count=1','-timeout=120s','-v'
)
```

Result: PASS in `1.556s` for both subtests. `App.Dark` is set before the first
frame, so the filenames and images agree.

- `mindmap-fields-app-normal-760x480-dark.png`: 37,022 B,
  2026-11-08 18:11:56. Warning-free fixture; field selector/editor visible at
  the normal inspector position; source and `阅读 / 编辑` mode controls are
  visible.
- `mindmap-fields-app-conflict-760x480-dark.png`: 47,825 B,
  2026-11-08 18:11:56. Retained scrolled conflict/recovery evidence; selector
  and editor are checked before scrolling, then both recovery actions are
  reachable inside the content area.

`.\native\scripts\go.ps1 -GoArguments @('vet','./app')` completed successfully
with empty output. No app product source changed in this checkpoint. Full
integration test/vet/build remains gated on root review; no full suite or build
has run for this stage.

## Acknowledgement — 2026-11-08

DeepSeek acknowledges the active `mindmap_fields_stage_plan.txt` and final
`handoffs/mindmap-fields-contract.md`. App ownership is limited to real
registered-adapter acceptance in `app/**`. Astra owns the single-editor
topic/note/link inspector and all `modules/graphview/**` implementation; GLM
owns `formats/mindmap/**`. No peer product files will be edited. A shell change
will be made only if the real app tests demonstrate a concrete composition
defect.

The pure-format seam is now visible in `formats/mindmap/fields.go`:
`FieldKind`, `FieldNote`, `FieldHyperLink`, `Field`, `SetNote`,
`SetHyperLink`, and `ClearField`. Astra has acknowledged the active view stage
and is implementing the native controls. Resource arithmetic was reconciled
first from saved JSON; no resource execution was performed during this stage.

## App test plan

Tests will use `newTestApp`, real `.mindmap` filesystem fixtures, registered
`MindmapAdapter` creation, and `ui.Tester` input against the exact proposed
native labels:

- field selectors: `字段：主题文本`, `字段：纯文本备注`, `字段：本地链接`;
- one active field editor: existing `节点文本`, plus `节点备注（纯文本）` and
  `节点链接（本地）`;
- action: `打开链接`, or `写回并打开链接` when the selected node's link field
  has a pending draft.

The focused app file will cover these workflows once Astra publishes a
compiling native-control checkpoint:

1. Open through the app registry and assert adapter ID `mindmap`, preview
   mode, a selected node, and the three field selectors. Selecting a clean
   note/link field must not dirty or reserialize the source.
2. Select `纯文本备注`, drive actual native `Compose` with Chinese and emoji,
   then ordinary `App.Save`. Assert the note is written exactly, preview mode
   is retained, unknown source metadata remains present, and the source buffer
   is no longer dirty.
3. Reopen the saved fixture through a fresh app. Assert the committed Unicode
   note is present through the native field editor and ordinary unchanged Save
   preserves the saved bytes.
4. Repeat the pending-note composition across `setMode(source)` and a real tab
   switch. Each lifecycle barrier must run the existing fallible Flush route;
   the note commits once and the source contains the full composed value.
5. Use a new writable document configured readonly before view creation.
   Readonly field selection and clean local-link activation remain available;
   native edits are refused and existing source bytes/history stay unchanged.
6. With a real relative link fixture, select `本地链接`, enter the exact target
   through native Compose, activate `写回并打开链接`, and assert the existing
   `Host.OpenReference` route opens the expected file exactly once. Verify that
   activation for a clean link does not mutate source, selection, history, or
   dirty state.
7. Exercise refusal through the app route using a network or remote target.
   The target must not open, the source field stays retained, and the app
   reports the existing local-only refusal without adding another classifier.
8. Narrow 760x480 acceptance will show a long note or pending link error,
   selected field, reachable editor/link action, and recovery controls.
   Light/dark and readonly coverage follows Astra's final component captures;
   the app will capture one representative integrated state rather than repeat
   the owner's full field matrix.

No app test was added before the native control checkpoint because guessing
the selector/editor identity would make the test seam speculative. After the
checkpoint, only a focused `./app` regex will run. Full tests/vet/build remain
gated on focused app evidence and root review.

## Prepared app integration — 2026-11-08

Root approved preparing the independent real-adapter tests while Astra finishes
the native control checkpoint. `app/mindmap_fields_app_test.go` now contains
four integration cases:

1. real registered mindmap preview; actual note-field Compose; ordinary Save
   to disk; exact Unicode/emoji/CRLF/trailing spaces; unknown top-level and node
   metadata; preview retained and dirty cleared; fresh-app reopen shows the
   Unicode in the note editor; clean selection/save keeps the reopened bytes.
2. pending local-link Compose -> `写回并打开链接` while the app dispatcher path
   is synchronous; the destination opens once and the graph tab/view survives
   Release; the committed link is exact; graph Save persists the link and
   unknown metadata.
3. clean missing-local and remote-link refusals open no tab and leave source,
   history version, dirty state and active tab unchanged while surfacing the
   existing refusal.
4. a mindmap made readonly before preview creation rejects direct editor input,
   Save As creates a writable copy, and a real field Compose/Save succeeds on
   that copy while preserving unknown metadata.

Selector lookup accepts the root-confirmed active suffix `（当前）` on either
side of the base accessible name, while editors keep the published exact names.
These tests are prepared and gofmt-clean only. They have not run yet because the
native inspector is under active edit; execution waits for Astra's compiling
checkpoint. No app product source changed and no full/integration suite ran.
