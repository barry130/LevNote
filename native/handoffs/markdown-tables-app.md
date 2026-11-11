# Markdown tables — app focused acceptance

## Current status — focused and final integration ACCEPTED

The actual registered-Markdown-adapter first-slice workflow now passes. Root
accepted the parser/export focused checkpoint, and the independent read-only
review reported no actionable parser/export/view-lifecycle finding. The view
owner's focused checkpoint is also green.

Root then authorized one final full native integration. The test, vet, release
build, isolated hidden startup and owned-PID `WM_CLOSE` all completed
successfully, and independent evidence validation passed 12/12 checks. Root
reviewed the raw20-package test stdout, aggregate result, executable metadata,
provenance, validation and smoke evidence, then accepted the exact20:49
executable. The 19:20 doc-block-reference release is now the previous accepted
binary. No resource sample was run.

Final integration commands, executed once from the repository root:

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./...','-count=1','-timeout','300s')
.\native\scripts\go.ps1 -GoArguments @('vet','./...')
.\native\scripts\go.ps1 -GoArguments @('build','-trimpath','-buildvcs=false','-ldflags','-s -w -H=windowsgui','-o','bin\levnote-native.exe','./cmd/levnote')
```

Actual integration result:

- full test exit0 in10.453s wall time; all20 package summaries parsed, with no
  failed package;
- app7.700s, `core/plugins`4.541s, `core/search`2.597s,
  `modules/graphview`4.146s, `modules/markdownview`3.910s and
  `formats/markdown`2.288s;
- vet exit0 in2.050s, with both live-captured streams empty;
- release build exit0 in2.441s, with both live-captured streams empty;
- the isolated hidden-startup harness exit0 in3.804s; owned PID2732 was found
  without a visible window, accepted `WM_CLOSE`, exited0, performed no forced
  cleanup and left no LevNote process;
- the fixture and target hashes, updater-shaped file/directory sentinels and
  source/copy executable hashes all remained exact;
- no process-resource sample or benchmark ran, and no product source was edited
  during integration.

Accepted executable:

- artifact path
  `native/artifacts/acceptance/markdown-tables-20261108-204724/levnote-native.exe`;
- 13,384,192 bytes;
- SHA-256
  `7632B1C50D9536B25A12CF9FA7E288B62F903A3B4ABA9896EF8849356864BBCD`;
- UTC mtime `2026-11-08T12:49:19.7290537Z`; PE32+ Windows GUI
  `0x8664`/`0x020B`/subsystem2.

Integration evidence is under
`native/artifacts/acceptance/markdown-tables-20261108-204724/`; the smoke
evidence is under
`native/artifacts/acceptance/startup-smoke-markdown-tables-20261108-204724/`.
The aggregate `result.json` SHA-256 at validation time is
`823205D739CDA036110ECADD1200BA8AD35F2449B68F8B9883CCAB265296466F`;
the smoke `result.json` SHA-256 is
`F2AD86A9C95708C28774017604A138F59F476082EC53B2CA2A117C2EFFC2E2CC`.
`evidence-validation.json` records PASS,12/12. The prior accepted binary is
preserved byte-for-byte as `previous-accepted-docblock-reference.exe`.
Root's separate acceptance receipt is `root-acceptance.json`; the accepted
`result.json` and `evidence-validation.json` snapshots were not rewritten.

## Final focused app command

Executed from `native`:

```powershell
$env:LEVNOTE_MARKDOWN_TABLES_SCREENSHOT_DIR =
  'F:\AIproject\LevNote\native\artifacts\acceptance\markdown-tables-app-20261108-200223'
.\scripts\go.ps1 -GoArguments @(
  'test','./app','-run','^TestMarkdownTable','-count=1','-timeout=120s','-v'
)
```

Actual result:

```text
PASS
ok  levnote.local/native/app  1.646s
exit 0
```

Focused `vet ./app`, `gofmt -d` over the touched app test files, and
`git diff --check` all exited `0`. The empty vet log is retained explicitly.

## Workflow coverage

`native/app/markdown_tables_app_test.go` now passes these actual-app cases:

- Opening through the real registered `markdown` adapter; BOM+CRLF unchanged
  Save and reopen preserve exact source bytes and history, with neighboring
  prose and literal table cells visible and no active link inside a cell.
- Native pointer, focused Enter and focused Space activation of exact
  `编辑表格源码`; pending native Compose is flushed once immediately before
  opening the whole-table draft and is not written to disk by that activation.
- Intentional whole-table raw replacement, one undo/redo transaction, Save and
  reopen. A committed structural row error intentionally rebuilds as complete
  visible raw fallback rather than being blocked as a later cell-edit shape.
- Save, source-mode, tab-switch and close barriers each flush the whole-table
  draft and preserve the edited source.
- A conflicting pending Compose fails activation without replacing the View,
  source, selection, version or history, and retains the `完成编辑` /
  `打开未同步副本` recovery path.
- Readonly rendering uses a wide-enough three-column fixture to force the
  stacked app layout. It exposes `只读表格` and complete-source guidance, has no
  editor action, and repeated cell clicks create neither a draft nor history.
- Recognized malformed and over-32KiB tables preserve complete source, expose
  warnings and retain the source route. Root's clarified resource rule is used
  truthfully: complete `Block.Raw` may be retained, while preview avoids full
  table layout materialization and shows the bounded source placeholder.
- Valid literal HTML export escapes hostile HTML and Markdown-link syntax and
  emits no active `<img>`/`href`; malformed fallback exports complete escaped
  source in `<pre>`. Export changes neither source bytes, version nor history.
- Ordinary prose before and after a distant table stays inside the content
  bounds at wide size.
- Final 760x480 light/dark split captures include the sidebar, active tab,
  source and responsive stacked preview. Geometry assertions keep the
  `编辑表格源码` action and a real body cell (`**字面**`) above the fixed status
  bar. The table is the first preview block in the screenshot fixture so one
  body row is visibly exposed; the separate wide fixture still covers prose on
  both sides.

## Test-harness corrections retained as provenance

The first execution was correctly unsuccessful; no product defect was inferred
from it. It exposed four invalid test assumptions:

- the shared Compose helper looked for headings from another fixture, so the
  table fixture now passes its `表格工作区` heading explicitly;
- source editing was incorrectly required to report `Multiline`; this role
  description is not part of the app acceptance contract;
- a two-column grid was assumed to become stacked at 360px even while the app's
  300px main-pane minimum kept its available width above the 2x112DIP grid
  breakpoint; readonly acceptance now uses a three-column fixture at that
  width;
- `| --- | x |` is not a delimiter-shaped failed candidate. The malformed
  tests now use `| --- | -- |`, which is recognized and must remain raw
  fallback.

An intermediate screenshot check also caught that the original tall fixture
left its body row clipped, even though the text remained in the accessibility
tree. That observation produced a strict visible-cell geometry assertion and a
compact screenshot fixture; it did not change product code.

No app product source changed in this focused stage.

## Evidence

Artifact directory:

`native/artifacts/acceptance/markdown-tables-app-20261108-200223/`

| File | Bytes | SHA-256 |
|---|---:|---|
| `focused-test.stdout.log` | 2,934 | `032D87E4F9784DD94405F7F0C41A0A0E42F19BA9CBDCCCE23B2DFAD1A2DAF834` |
| `focused-vet.stdout.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `markdown-tables-app-split-light-760x480.png` | 46,219 | `ACE45FD41128DBEB268257CD481D2A01A09CD2DF40824F8EE52BFAEF35C69B78` |
| `markdown-tables-app-split-dark-760x480.png` | 46,639 | `5321F8B3324A2A2A417F67B6E40C0EE5D9D0011BEAA875E48D72D43403B83ABD` |

Both final PNGs are 760x480 and were inspected directly after capture.

Parser/export focused evidence remains in
[`markdown-tables-format.md`](markdown-tables-format.md); native view focused
evidence and six component images are in
[`markdown-tables-view.md`](markdown-tables-view.md).

## Documentation and final review package

The owned native/root documentation describes the first slice on all nine
surfaces: `native/README.md`, `native/formats/API.md`,
`native/ARCHITECTURE.md`, `native/VALIDATION.md`, root `README.md`,
`ROADMAP.md`, `CHANGELOG.md`, `docs/decisions/0063-native-migration.md`, and
`docs/decisions/README.md`. The wording keeps the accepted 19:20
doc-block-reference release as the previous accepted binary and marks the
20:49 table first slice as current.

Documentation validation after the edit:

- strict UTF-8 decoding: PASS for all9 edited files;
- balanced Markdown fences: PASS for all9 files;
- 214 relative Markdown links across the nine product documents: all resolved;
- `git diff --check`: PASS.

The final app images were independently re-opened after the documentation
edit. Both are 760x480; the light and dark split views show the Markdown source,
the visible `编辑表格源码` action and a real body cell, and the fixed status bar
remains below the content. A fresh hash/size check exactly matched the table
above. The view owner's exact focused pattern is `-run Table` (without `^`);
the current
[`markdown-tables-view.md`](markdown-tables-view.md) already records that
literal command.

