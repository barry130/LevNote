# Markdown single-cell editing — app acceptance

## Current status

Focused app gate and final view-owner checkpoint PASS. Root accepted the
test-only correction to one older app assertion that used raw cell text rather
than the agreed full `RoleCell` name. A fresh authorized integration passed
test/vet/release build/evidence validation and the isolated hidden
startup/owned-PID `WM_CLOSE` smoke; the complete accepted record is in
`native/handoffs/integration.md` and
`native/artifacts/acceptance/markdown-cells-20261108-213713/root-acceptance.json`.
The13,425,152-byte executable with SHA-256
`34F3F48C98300B053B0BA0E69BE9D9F89CDA15E17759E6A26AD817A1E9C1D14C` is the
current accepted release. The20:49 first-table release remains preserved as
the previous accepted executable.

The view owner published compiling exact labels. The app tests now target the
exact staged panel labels rather than the source editor's generic
`文档编辑区`, which can also be present in split mode.

Focused commands already run:

```powershell
.\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestMarkdownCellsRealAdapterStagedSessionHeaderBodyUndoRedoSaveReopen$','-count=1','-timeout','180s','-v')
.\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestMarkdownCells','-count=1','-timeout','180s','-v')
.\scripts\go.ps1 -GoArguments @('vet','./app')
.\scripts\go.ps1 -GoArguments @('test','./modules/markdownview','-run','^$','-count=1','-timeout','120s')
```

Current result: all PASS. The complete cell-focused app regex passed in 2.010s
and focused app vet produced no diagnostics. The view compile-only command is
an integration prerequisite, not cell-view acceptance.

The primary test proves a real registered Markdown adapter, exact header and
body cells, host unchanged before Finish, exact BOM/CRLF Save/reopen, and
isolated Undo/Redo.

The barrier matrix uses source version/history captured before any staged
typing and covers Save, HTML export, source mode, tab switch, and close. The
pending native composition test types into an ordinary Markdown heading and
then activates a cell; activation flushes that composition exactly once before
opening the cell draft.

The invalid path exercises Save, export, source mode, tab switch, and close,
and verifies the original document, selection, history, view, mode, and tab
list remain unchanged across every failed barrier. It then presses the actual
`打开未同步副本` control. Successful recovery adds and activates a new unsaved
tab containing the exact invalid staged value, while the original document's
source and history remain unchanged. Explicit discard, no-op Finish, and
readonly activation are tested separately.

During diagnosis, a premature accessibility-value assertion was removed:
native cell input is correctly held as pending composition until Flush or
Finish and is not exposed as a committed value before that barrier.

A transient owner-side package compile failure (`undefined: markdown`) was
reported and is no longer present.

The app acceptance will prove the host boundary rather than repeat the view
owner's parser/helper matrix:

- actual native header/body activation and staged typing;
- host source bytes, version, dirty state and history remain unchanged while
  the cell draft is local;
- one successful finish/save barrier performs at most one host edit and one
  history transaction, followed by exact Save/reopen and one Undo/Redo;
- Save, export, mode switch, tab switch and close respect the view barrier;
- invalid values retain the exact source, selection, version and history and
  remain visible/recoverable;
- explicit discard and no-op paths leave source/history unchanged;
- readonly activation creates no draft;
- the existing whole-table editor and literal HTML export continue to work;
- narrow integrated split screenshots in light and dark keep the staged notice,
  recovery action and status bar within 760x480.

## Session persistence boundary

The staged cell draft is view-local by contract. An app session snapshot while
that draft exists captures the host document, so it can never silently commit
the staged value and must not claim to have captured it. The verifier will
produce a session while a staged value is pending, confirm the host document is
unchanged, confirm the staged text is absent from the recovery file, and then
confirm that Finish or Save commits the value through the normal barrier.

The staged UI must continue to say that Finish or Save is required. Product
help and final documentation must state that an uncommitted cell value is not
captured by crash/session recovery. The normal Save/close barriers handle
committed values; no session API change is planned.

## Integrated screenshots

Command:

```powershell
$env:LEVNOTE_MARKDOWN_CELLS_SCREENSHOT_DIR='<artifact directory>'
.\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestMarkdownCellsAppNarrowScreenshots$','-count=1','-timeout','120s','-v')
```

Final recapture PASS after the view owner retained the host table preview and
disclosed the local-only draft boundary. Command result:

```text
PASS, screenshot test 0.10s
ok levnote.local/native/app 1.636s
```

Artifact directory:

`native/artifacts/acceptance/markdown-cells-20261108-213013/`

- `markdown-cells-staged-light-760x480.png`, 45,262 bytes,
  SHA-256 `4B7E44C1F007ED46DF4C8BE82D353051EA2D1081F3B2E8F2B2E3F667B47F9607`.
- `markdown-cells-refused-dark-760x480.png`, 50,747 bytes,
  SHA-256 `D3D30CE6927132143A00505A73737C7E3E9CB2E7F58DCC37FF454465B3ECE461`.

Both are actual app views at 760x480 in split mode. The light image shows a
completed staged native composition with Finish and Discard above the status
bar. The dark image shows the structural-pipe refusal, retained table context,
staged value, Finish, Discard, and `打开未同步副本` within the content area; the
global status remains below it. Both were opened and visually inspected after
the final view-owned visual correction.

## Accepted evidence gate

Root accepted the fresh full native test/vet/release-build/evidence-validation
and isolated-smoke sequence recorded in
`native/artifacts/acceptance/markdown-cells-20261108-213713/`. The immutable
aggregate result is `result.json`, SHA-256
`E3AD9163C07EBA90C6436069BBE751DAACEBB029E8EB8D8FFDBE4DF9112D9BF8`; the
independent validator is `evidence-validation.json`, SHA-256
`952BA483165D6D1181F8C5F3EA82421485B2C0575E6218855D4318242D18D72C`, PASS12/12.
No product source changed after this accepted integration, and no new full
sequence is authorized for the cell slice.
