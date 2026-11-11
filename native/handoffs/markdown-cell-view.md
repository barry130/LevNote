# Markdown single-cell view — compile-ready checkpoint

Status: READY FOR ROOT REVIEW AND APP INTEGRATION (2026-11-08).

`handoffs/markdown-cell-contract.md` is implemented in
`modules/markdownview/**`. Product source is ready to freeze at this checkpoint.
The accepted responsive table and whole-table source behavior remain intact.
No app, parser/format, core, editor, vendor, script, or other owner file was
changed for this slice.

## Owned verification

Go 1.27.3 was run through `native/scripts/go.ps1`.

```text
go test ./modules/markdownview -count=1 -timeout=120s -v
PASS
ok levnote.local/native/modules/markdownview 2.404s
wrapper exit 0, 3.647s

go vet ./modules/markdownview
wrapper exit 0, 0.652s, no output

gofmt -d modules/markdownview
clean, no output
```

The focused table-plus-cell run also passed:

```text
go test ./modules/markdownview -run '^Test(Table|TableCell)' -count=1 -timeout=120s -v
PASS
ok levnote.local/native/modules/markdownview 1.477s
```

The component screenshot test passed with an absolute output directory:

```text
go test ./modules/markdownview -run '^TestTableCellScreenshotsLightDark$' -count=1 -timeout=120s -v
PASS, test 0.09s
ok levnote.local/native/modules/markdownview 1.259s
```

No app test, full native test, build, release, hidden-startup, resource matrix,
commit, or push was run by the view owner.

## Stable app-facing action contract

- Header cells have role `ColumnHeader` and stable accessible-name prefix
  `表头，第 1 行，第 N 列：`.
- Body cells have role `Cell` and stable accessible-name prefix
  `数据，第 R 行，第 N 列：`.
- Empty cells use `（空单元格）`; names are bounded to 512 UTF-8 bytes with a
  rune-safe `…`.
- Cell description is
  `按 Enter 编辑此单元格的 Markdown 原文；双击也可以编辑`.
- Cells are explicitly `Focusable` and `TabStop(true)`. Tab reaches a visible
  cell. Unmodified Enter opens that focused cell. Space remains scrolling.
- Primary-button double-click captures the exact target on pointer-down and
  consumes the gesture, so the row's whole-table double-click does not open
  behind it. A normal single pointer click remains available for selection.
- The staged editor labels are stable:
  `编辑单元格 · 表头 · 第 1 行 · 第 N 列` and
  `编辑单元格 · 数据 · 第 R 行 · 第 N 列`.
- The visible staged line is
  `单元格草稿尚未写回；保存或完成编辑时写回`. Its help tooltip is
  `暂存于当前窗口，尚不进入恢复快照`.
- Stable controls are `完成编辑`, `放弃草稿`, and on failure
  `打开未同步副本`. `放弃草稿` is present for every staged cell state.
- Refusal warnings use the visible prefix `单元格草稿尚未写回：`.
- `编辑表格源码` remains the immediate-sync whole-table path.

Save, export, explicit source/mode switching, tab switching/opening, reload,
close, and format-view switching already enter `App.flushTab`, which calls
`editor.FlushView(View)`. The staged cell therefore uses the same `View.Flush`
barrier as existing drafts. `完成编辑`, `Ctrl+Enter`, and `Escape` call that same
commit path. The application must stop the action when `Flush` returns an
error, as `flushTab` already does. Direct `Release` is the deliberate cache
lifecycle exception described below.

## Capture and rendering

Per-frame table rows retain numeric table/row offsets. A complete table or cell
source string is cloned only at actual pointer-down or focused Enter gesture
start, after validating all byte/UTF-16 bounds. The captured target includes:

- Buffer identity and version;
- native preview row ID;
- complete-table and exact edited-cell UTF-16 ranges;
- table, delimiter, row, cell, and field source ranges;
- physical row and column indices;
- column count and alignment vector;
- bounded complete table source at most 32 KiB;
- bounded edited-cell source.

Rendering uses the existing virtual `List`, one row per physical source table
row, at most eight cells per row. Grid/stacked behavior and the 112-DIP
breakpoint are unchanged. Cell literals remain bounded to three visible lines
with a rune-safe ellipsis and a complete-source tooltip capped at 512 bytes.
No per-frame whole-table clone, row-width widening, global horizontal scroll,
or additional parser/cache was introduced.

While a cell draft is open, the immutable table preview remains visible below
the editor as context. The current cell/panel is identified once. Ordinary and
whole-table draft modes continue to suppress their edited block with
`正在编辑此块`.

## Staged draft and commit

The one existing editor is used at a compact 96-DIP height. A cell OnChange
updates only local editor text and history. It does not mutate the host Buffer,
call `Changed`, or hold a host transaction while typing or composing.

`View.Flush` first completes native composition, then performs:

1. Buffer identity, group, readonly, table/cell mapped-range, complete source,
   row identity, and current non-Partial document checks.
2. A 4096-byte local draft cap using `LenBytes` before materializing dirty
   text. The 4097-byte case is refused without a host write.
3. Single-line, valid UTF-8, and unescaped structural-pipe checks.
4. Projected complete-table <=32 KiB and projected host document <=
   `markdown.MaxParseBytes` checks from lengths before candidate construction.
5. One bounded complete-table candidate parsed by `markdown.Parse`; it must be
   exactly one complete valid table with the same row/cell counts and
   alignments.
6. Translated absolute ranges: all non-target ranges and bytes must retain
   their identity after replacement-byte delta. Only the target padding/field
   classification may change. Empty compact cells and escaped pipes use the
   parser oracle rather than a competing grammar.
7. One `ReplaceUTF16` and one `Changed` callback only after every check passes.

A changed success adds one host undo entry. A successful no-op adds no write,
Changed callback, or undo entry. Success, including no-op, closes the staged
draft and invalidates the preview so later editing recaptures freshly trimmed
padding. Failure keeps the exact same View, draft text, local selection/history,
host bytes/version/selection/history, warning state, and recovery controls.

An unrelated insertion before the captured table rebases through the existing
journal. A replacement Buffer, same-table edit, group, expired journal,
Partial/current parse failure, malformed or structurally changed candidate,
oversize draft/table/document, or live readonly transition refuses visibly.
If a previous draft was successfully flushed and that invalidates the newly
captured table, the refusal says the previous draft was handled and asks the
user to reselect.

Readonly cells remain visible and focusable. Activation flushes an existing
draft, rechecks readonly after Flush, reports the readonly refusal, and creates
no cell draft or host change.

`View.Release` is a direct cache lifecycle operation. A failed staged Flush
retains the local draft, text, local selection/history, cell identity, and the
next-frame recovery path; parsed preview geometry/cache may be dropped. A
successful Release closes the draft. Pending cell pointer/key gesture state is
always cleared and is never allowed to block a later Draw/source refresh.

## Contract coverage in owned tests

The focused suite exercises:

- real pointer, Enter, and Space/cancel behavior;
- grid/stacked header, body, empty, long, nonzero-source-offset, and second
  duplicate table targets;
- staged host immutability, no-op, discard, and existing-draft Flush order;
- successful empty-cell commit and exactly one host write/Changed/undo entry;
- failed Flush retention/correction, readonly live transition, and recovery;
- pointer cancel and View.Release gesture cleanup;
- insertion-before rebase with and without an intervening frame;
- same-table refusal, replaced Buffer, group, expired journal, Partial source,
  and failed Release recovery;
- 4096/4097 draft bytes, 32 KiB table, and exact document parse bound;
- Unicode/astral, CRLF/EOF, optional/combined/mixed outer-pipe fixtures,
  compact one-column empty, no-outer lead-padding changes, escaped pipes,
  odd/even backslashes, and non-target shorter/longer range translation;
- direct-Release invalid and stale drafts rebuilding `打开未同步副本` with the
  exact `Markdown-块草稿.txt` recovery payload;
- accepted table/reference/tasks/inline-link regressions in the complete owned
  package.

The empty stacked-cell fixture intentionally commits to `|  新增🙂|`: the
parser's empty `TextRange` anchor is after the preserved padding, so the view
does not synthesize or normalize padding.

## Component images

Directory:

`native/artifacts/acceptance/markdown-cell-view-20261108-final`

| File | Size | SHA-256 |
| --- | ---: | --- |
| `table-cell-staged-light-760x480.png` | 28,294 B | `5AFCA25A66578BCB5731490A5D6F5100CB1EB883B9ABC317E220619AC31E3F8A` |
| `table-cell-staged-dark-760x480.png` | 28,339 B | `6E2B922646170D652096E5901C8FE5D3AB60BC7AAE2F81E88708AAEE1C2608D1` |
| `table-cell-invalid-light-760x480.png` | 32,254 B | `6A71B658385F544B10BEED9D4CD1AB9A231DCF235FA57CF4A9AA012A97186062` |
| `table-cell-invalid-dark-760x480.png` | 32,127 B | `ED3042A07006E0AE91FDD9EAE91F82ABA87A6669F604BAFD23A7DECFDFA55207` |

All four are exactly 760x480 and were visually inspected. The retained table
preview is visible during cell editing; the staged notice appears once; invalid
light/dark states keep `完成编辑`, `打开未同步副本`, and `放弃草稿` visible and
inside the viewport. These are component images; app integration must recapture
its own paired 760x480 split images against the current view.

The visual fixture semantics changed from the earlier cell images because the
immutable table preview now remains visible during cell editing and the
local-only recovery-scope text moved to the staged-notice tooltip. Earlier
component images must not be reused as current evidence.

## Limitations

- This checkpoint does not claim Windows IME, GPU, or integrated app-shell
  behavior; the app owner runs the registered-adapter suite.
- Direct session persistence does not capture an uncommitted cell draft. The
  visible UI says it is staged only in the current window. Normal save, finish,
  switch, and close use the Flush barrier; no crash-recovery claim is made.
- No resource matrix or full build is part of this owned gate.
