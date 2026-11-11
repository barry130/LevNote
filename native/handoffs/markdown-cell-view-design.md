# Markdown table cell editing — finite view design

Status: SUPERSEDED by the final active
`native/handoffs/markdown-cell-contract.md`. Implementation now lives in
`native/modules/markdownview/table_cell.go`.

## Contract selection

Use a specialized mode of the existing independent `editor.Editor` draft. Add
no editor, parser, format, app, session, or vendor API.

- Keep `编辑表格源码` as the whole-table action with its current immediate-sync
  draft lifecycle.
- Add a staged cell mode to the same draft panel and same single `v.draft`.
- Cell activation is a native Tab/Enter or pointer double-click action.
- Cell content is the exact `TableCellRef.TextRange` byte span. No Markdown
  rendering, decoding, escaping, or normalization occurs while editing.
- The only host mutation is one `ReplaceUTF16` at that span after complete
  revalidation. No-op makes no mutation.
- Explicit `放弃草稿` cancels. Escape attempts the same finish/Flush operation
  as `完成编辑`, preserving the current finish behavior.

This is the smallest current-MyGo design and does not depend on roles becoming
focusable automatically.

## Native UX and focus

Update the table guidance to state that `编辑表格源码` edits the complete table,
while double-clicking a cell edits only that cell's literal source. The
whole-table button remains the always-visible, top-of-table escape hatch.

### Grid

Each visible cell box is the single focusable action surface:

```go
box.
    Role(role).                 // RoleColumnHeader for the header row, else RoleCell
    TabStop(true).
    Label(boundedCellLabel).
    Description("按 Enter 编辑此单元格的 Markdown 原文").
    HandleInput(cellInput)
```

Keep the existing visible cell text and selectable copy behavior inside it.
The label contains row/column identity and bounded preview text, using the
existing 512-byte tooltip bound.

### Stacked layout

The value box, not the decorative column-label box, gets the same role,
`TabStop(true)`, label, description, focus ring, and input handler. The
existing row label, column label, visible ellipsis, and complete-source route
stay. The whole-table source button remains above the first record.

Only rows built by the existing virtual `ui.List` create these focus stops.
Each physical source row creates at most eight cell action boxes; no table-wide
focus framework or hidden per-cell widgets are introduced.

### Gesture and row precedence

MyGo delivers pointer input to the innermost element with `HandleInput`; if it
returns true, the row handler does not receive that press or its release.
Therefore:

1. On primary `InputPointerDown` with `Clicks == 1`, the cell handler returns
   false. Ordinary text selection and copy still work, and the row handler
   ignores that single click.
2. On primary `InputPointerDown` with `Clicks >= 2`, the cell captures its
   immutable target and returns true. The row-level whole-table double click
   cannot also run for that gesture.
3. On a matching non-canceled `InputPointerUp`, activate from the captured
   target. On `Canceled`, clear without opening. A disappearing pressed widget
   may therefore receive MyGo's synthetic canceled release.
4. A focused cell handles `InputKeyDown` for unmodified Enter, captures the
   target, and returns true. Tab itself remains MyGo's normal focus movement.
   Shared keyboard guidance says `Tab 聚焦单元格，Enter 编辑原文`.
5. Gesture state is keyed by Buffer, preview row ID, table span, row, and
   column. It cannot be consumed by another table or another cell.

For a disappeared cell, report a stale refusal only when current source
identity no longer maps or the complete table/cell bytes changed. A normal
cancel, a source-equivalent virtualization pass, scroll, or focus movement is
quiet and clears the gesture.

Readonly cells remain recognizably available for guidance. Activation still
captures source identity, flushes an existing draft, then visibly refuses with
the existing readonly message. It creates no cell editor and changes no bytes,
selection, history, or version. The whole-table button keeps its current
readonly behavior.

## Captured target

Per-frame rendering carries only numeric source descriptors. Clone sources only
when an actual pointer/key gesture starts:

```go
type tableCellTarget struct {
    buffer      *text.Buffer
    version     uint64
    rowID       uint64
    tableSpan   text.Range // complete table, UTF-16
    cellSpan    text.Range // exact TextRange, UTF-16
    tableRange  markdown.SourceRange
    cellRange   markdown.SourceRange
    delimiter   markdown.SourceRange
    row, column int
    rows        int
    columns     int
    alignments  [8]markdown.TableAlignment
    tableSource string // clone only at gesture start, <=32 KiB
    cellSource  string // clone only at gesture start, <=4096 bytes
}
```

Capture validates every byte boundary before slicing or cloning. Duplicate
tables, identical labels, and duplicate cell values remain distinct through the
row ID and exact table/row/column source identity.

Before opening the editor:

1. If any draft exists, call `Flush`. A failure keeps it and stops activation.
2. Recheck readonly after Flush.
3. Require the same non-nil Buffer, no foreign group, a non-Partial current
   source, and a parseable current table context.
4. Map table and cell spans with `MapRangeSince`; an unrelated insertion before
   the table may rebase. Require the mapped complete table bytes to equal
   `tableSource` and the mapped cell bytes to equal `cellSource`.
5. Rebuild and validate the current table metadata before opening the draft.

Any edit inside the captured table refuses rather than merging. If activation
first flushed an earlier cell draft in that same table, the flush may
legitimately invalidate the newly captured target. In that known case say that
the previous cell draft was written and ask the user to reselect; otherwise say
the table changed and ask for reselection. Do not report that nothing was
saved when the preceding Flush succeeded.

## One staged cell draft

Add only view-private mode state tied to the existing `v.draft`, for example:

```go
type tableCellEdit struct {
    target tableCellTarget
}
```

`openTableCellDraft` uses the existing editor, outer table guard, cell
`TextRange`, and panel. It sets `ShapeLimit` to 4096 bytes but does not add a
new input cap. The editor may hold an invalid intermediate value, including a
newline, without touching the host.

`clearDraft` clears `cellEdit` along with the existing draft fields.
`dropPreviewCache` and `Release` clear all pending cell gesture state. A failed
Release Flush must not claim crash recovery for this in-memory draft.

### `OnChange`

When `cellEdit != nil`, `OnChange` does not call `syncDraft` and performs no
host `ReplaceUTF16` or `Changed`. It updates only the existing editor's local
text/history. Ordinary block and whole-table drafts keep their current
immediate-sync callback unchanged.

### `pendingDraft`

Composition remains pending. For a staged cell, compare `Buffer.LenBytes()` to
`len(editBase)` first. A mismatch proves pending without `Buffer.String()`, so
an oversized dirty value is never materialized merely to paint state. Only an
equal byte length is compared with the bounded base string.

### `Flush`

`View.Flush` still calls `draft.Flush()` first. It then dispatches:

- ordinary/whole-table draft: existing `syncDraft`;
- staged cell draft: `syncCellDraft`.

`syncCellDraft`:

1. Reject more than 4096 draft bytes using `LenBytes` before calling `String`.
2. Rebase/revalidate the table and cell as above; reject readonly, group,
   stale/replaced Buffer, changed same-table bytes, Partial/uncertain source, or
   oversize target visibly.
3. Read the now-bounded proposed value and perform raw validation.
4. No-op means exact equality with the captured cell bytes. Do not write, call
   `Changed`, dirty the document, or add history.
5. Build one bounded candidate table and parse it with the oracle below.
6. On success perform exactly one `ReplaceUTF16(cellSpan.Start,
   cellSpan.End, value)` and call `Changed` once.
7. On success, including no-op, clear the cell draft and invalidate the preview.
   The next physical row build captures a fresh parser `TextRange`.
8. On failure, retain the same View, editor, draft text, local selection and
   local history, host source/version/history, and recovery controls.

Clearing after successful Flush also means Save, export, tab/mode switch, and
close commit at most once. `完成编辑` calls `Flush`; its following `clearDraft`
is idempotent. Escape goes through that same finish path. `放弃草稿` remains an
explicit cancel at every staged cell state.

The existing save/switch/close barriers already call `editor.FlushView`, so no
app or session change is required. Show
`单元格草稿尚未写回；保存或完成编辑时写回` throughout staged mode, then show a
bounded validation reason when present. Always show `放弃草稿` for a staged
cell. On failure also retain `打开未同步副本`. This local draft is not captured
by app session persistence, so no crash-recovery claim is made.

## Empty cells and padding

Replace only `TextRange`, never `Range`.

- Existing surrounding ASCII padding and structural separators remain byte
  identical.
- A nonempty cell replaced with empty removes only its trimmed text and leaves
  its padding.
- An all-padding cell has the parser's zero-width `TextRange` at the padding's
  right edge. Insertion writes there; no synthetic marker or space is added.
- Exact no-op, including empty draft over empty source, writes nothing.
- Leading/trailing ASCII whitespace typed by the user becomes ordinary cell
  padding after commit. Because success clears the draft, a later edit
  re-captures a freshly trimmed `TextRange`; the editor never continues with a
  stale span.

## Raw and candidate validation

The following are refused before host mutation:

- draft length over 4096 UTF-8 bytes;
- `\r` or `\n`, which would change physical rows;
- an unescaped structural `|` under the parser's odd/even immediate-backslash
  rule; backticks never protect a pipe;
- a value ending in an odd immediate backslash when the unchanged byte after
  the cell `TextRange` is a structural `|`;
- projected complete-table source over 32 KiB, checked from lengths before
  concatenation;
- a changed/replaced/stale Buffer, group, readonly state, Partial/uncertain
  context, or captured table/cell byte mismatch.

An escaped pipe such as `\|` remains literal source. Validation follows the
existing one-pass backslash parity rule instead of changing parser code.

After local checks, build:

```text
captured table bytes before TextRange
+ proposed cell bytes
+ captured table bytes after TextRange
```

Parse that one bounded candidate table with existing `markdown.Parse`. Require:

- `Partial == false` and exactly one block;
- `BlockTable`, empty `Problem`, and no unexpected warning;
- block Range/Raw exactly the complete candidate;
- the same header/body row count and every row's column count;
- delimiter bytes and alignments exactly unchanged;
- all real row/cell ranges valid and contained.

This candidate full-table oracle prevents an edit from deleting compact
one-column/no-outer essentials or turning the intended table into text, while
CR/LF/pipe refusal makes the unchanged surrounding block boundaries stable.
It does not require reparsing or copying the complete document, so independent
cell Flush can still validate when the overall preview is outside its separate
whole-document limit.

The old stale `tableCurrent` clone remains the whole-table path. Cell rendering
does not call it or clone the table each frame.

## Focused implementation acceptance

When root activates implementation, the owned view tests should prove:

- Tab reaches visible `RoleCell`/`RoleColumnHeader` boxes; Enter opens the exact
  cell; pointer double-click is captured on down and cannot trigger row-level
  whole-table editing.
- Grid and stacked layouts provide the same action; empty, long, Unicode,
  astral, duplicate-value, duplicate-table, and second-visible-table targets
  use exact source identity.
- No host bytes/history/dirty signal change before Flush; invalid intermediate
  raw text remains recoverable; explicit discard cancels; Escape follows the
  fixed finish behavior.
- A valid Flush is one undo entry; no-op has none; successful Flush closes the
  cell draft and refreshes preview; failed Flush retains the same draft,
  selection, recovery route, and source.
- Insertion before the table rebases; any same-table edit, replacement,
  deletion, reorder, delimiter/shape change, group, Partial, expired mapping,
  or readonly state visibly refuses.
- 4096-byte cells are valid, 4097-byte drafts refuse without host mutation;
  candidate growth over 32 KiB refuses; CR/LF, unescaped/backtick-protected
  pipes, and an odd trailing backslash before a separator refuse; escaped pipes
  and CRLF bytes outside the edit remain exact.
- Existing whole-table, reference, task, inline-link, list, input, Flush, and
  draft behavior remains unchanged.

No further product/API decision is needed for this slice. The remaining root
gate is authorization to implement this finite design.
