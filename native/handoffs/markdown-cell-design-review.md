# Markdown single-cell editing — second-slice design review

Status: DESIGN ONLY. No implementation, tests, product changes, or executions
were performed. This proposal assumes the first table slice remains frozen and
uses its current byte-range API and the existing one-draft/Flush lifecycle.

## Recommendation

Implement single-cell editing as a specialized mode of the existing independent
draft, not as a new editor framework. The draft buffer contains the exact
`TableCellRef.TextRange` bytes. Before the existing host `ReplaceUTF16`, the
view validates that the proposed bytes cannot change the table's physical row,
column count, delimiter, or raw-table cap. Invalid content remains in the
draft with a visible reason; the authoritative host source and history remain
unchanged.

No exported format API is needed. `TableCellRef`, `TableRowRef`, `TableRef`,
absolute byte ranges, and the existing whole-table action remain unchanged.

## Cell target and source identity

Use a view-private target captured at the native gesture start, alongside the
existing row identity:

```go
type tableCellTarget struct {
    buffer      *text.Buffer
    version     uint64
    rowID       uint64
    tableSpan   text.Range // complete table, UTF-16
    cellSpan    text.Range // exact TextRange, UTF-16
    tableSource string     // bounded clone, <=32 KiB
    cellSource  string     // bounded clone, <=4096 bytes
    row, column int
}
```

Derivation from an immutable preview:

1. Validate the table block, selected row, cell, alignment count, and all byte
   ranges against `preview.doc.Raw()`.
2. Convert the complete table and selected `TextRange` to UTF-16 spans from the
   block start with `text.UTF16Len`. For a zero-length `TextRange`, the cell
   span is a stable anchor with equal start/end.
3. Clone only the complete table source (already capped at 32 KiB for a valid
   table) and the selected cell source (already capped at 4096 bytes).
4. Store buffer, version, numeric row ID, row/column indices, and both spans.

Activation should reuse the whole-table lifecycle:

1. Flush any existing draft first. A failed Flush returns without clearing it.
2. Refuse readonly, nil/replaced buffer, group transaction, Partial, oversize,
   or stale mapping.
3. Map both spans with `MapRangeSince`; require the current complete table and
   current cell slices to equal the captured clones.
4. Open the existing draft with the whole table as the guard and the cell
   `TextRange` as the edit field. Keep one draft, one `editSpan`, and one
   `editGuard`; add only the private cell-target state needed for validation.

An insertion before the unchanged table can map both spans and remain valid.
Any edit inside the same table changes `tableSource` and must be refused, even
if the selected cell itself still maps. This is stricter than trying to rebase
around a changed table and avoids silently merging concurrent edits.

## Staged candidate and authoritative shape validation

Perform validation immediately before any host replacement, after the draft
Flush and range rebase. The proposed value is the exact raw cell source; it is
not Markdown-decoded or normalized.

Do not rely on character-rule checks alone. Build a bounded staged candidate
from the current authoritative host source:

1. Map the captured complete-table and cell spans through the existing journal.
2. Require current slices to equal the captured table and cell sources.
3. Require the current full parse at activation and at commit to be non-Partial,
   and require `host.LenBytes()+len(value)-len(cellSource)` to stay no larger
   than `markdown.MaxParseBytes`.
4. Build the prospective complete table by replacing exactly the current cell
   `TextRange` bytes with the proposed value. Padding, outer pipes, escaped
   pipes, line endings, and every byte outside that range remain authoritative.
5. Parse the staged table standalone with `markdown.Parse(stagedTable)`.
   Require exactly one valid `BlockTable` consuming the whole candidate: no
   `Problem`, `Rows != nil`, `Range == [0,len(stagedTable))`, and `Raw ==
   stagedTable`.
6. Compare shape with the captured table: the same number of display rows,
   the same cell count in every row and the target row, and the same alignment
   sequence.
7. Compare every display row's `Range` and every cell's `Range` to the captured
   table, except that the target cell's `TextRange` is allowed to change.
   Padding and all other cell boundaries must not move.
8. Require the target cell's new `TextRange` to contain at most 4096 UTF-8
   bytes and the staged table to remain at most 32 KiB.

This parser-based oracle catches cases that character rules miss:

- a one-column compact row such as `|a` edited to empty or padding can
  reinterpret its leading pipe as a trailing outer marker and lose the
  one-cell shape;
- deleting the last non-padding byte of a no-outer row can reinterpret a
  neighboring structural pipe as an outer marker;
- changing a compact header or body cell can make that row classify as a
  heading, list, quote, fence, or another recognized block;
- a value ending in an odd backslash immediately before a structural pipe can
  absorb the separator into content.

The standalone parse also enforces the same line/block budget and aggregate
table-descriptor budget for the candidate; because the candidate consumes its
full raw range and has the same shape, those counts remain constant relative
to the captured table. This is bounded: clone and parse at most the current
32 KiB table, and create no host transaction before acceptance.

Backslash parity remains useful for an early explanation, but acceptance must
come from the parsed candidate: count the immediate backslash run before each
pipe; odd escapes it, even leaves it structural. Backticks never protect a
pipe. Escaped pipes remain literal raw content and are preserved exactly.

Do not add syntax normalization or a second parser. `markdown.Parse` is the
authoritative classifier. On refusal, report the first concrete invariant that
failed (source identity, whole-candidate consumption, row/column shape,
alignment, boundary movement, cell size, raw size, line/block/descriptor
budget, or document size), not a speculative normalization.

## Draft lifecycle correction

The current editor's `OnChange` callback invokes `syncDraft` immediately in the
existing draft path. Therefore the earlier implication of one host undo per
edit session and Escape-as-discard is not true without a staged cell mode.

For single-cell editing, open the existing editor with `OnChange` disabled or
routed to staging only. Keep edits local until the existing complete/commit
action or a host Flush barrier. At that boundary:

1. commit the native editor composition (`draft.Flush`);
2. map and validate source identity;
3. build and parse the staged candidate;
4. perform one host `ReplaceUTF16`, producing one undo transaction.

Do not keep a long-lived host `BeginGroup`. A failed validation returns to the
same local draft with its text, selection, and recovery controls intact and
does not call `host.Changed`. A no-op (`value == cellSource`) clears normally
without a host write, dirty flag, undo entry, or `Changed`.

Preserve normal draft semantics: Escape attempts commit/finish, not hidden
discard. Add an explicit `放弃草稿` control if a discard affordance is needed.
This is the safest consistency with the existing editor and avoids a special
Escape branch. Release/save/tab switching still uses the same Flush barrier;
if validation fails there, the draft remains recoverable.

On success, call the existing `ReplaceUTF16(cellSpan.Start, cellSpan.End,
value)`. This gives one undo transaction. On no-op (`value == cellSource`),
do not write, dirty the document, or add history. On refusal, retain the draft
and reuse the existing conflict/recovery controls and “open unsynchronized
copy” route; do not clear or overwrite the user's draft.

## Native action and UX

Keep the header's existing `编辑表格源码` button for whole-table raw editing.
For single-cell editing, use the cell itself as the only additional affordance:

- double-click a body or header cell opens the one cell draft;
- readonly cells never open an editor and retain existing guidance;
- Escape attempts the same commit/validation path as the existing draft;
- an explicit `放弃草稿` control discards locally staged content without writing;
- the existing complete/commit control performs Flush and validation.

This adds no per-cell buttons, menus, toolbars, or separate editors. Update the
table hint to say that a cell double-click edits literal cell source, while the
header button edits the complete table. If the current accessibility runtime
already exposes focusable `RoleCell` targets, Enter should use the same
captured identity path as double-click. Keyboard activation is part of this
slice; no new focus framework is needed.

Use a bounded one-line-looking draft panel, but do not add a new editor widget
type. `ShapeLimit` can remain bounded to the cell limit; Enter may still create
a local newline in the draft, but validation refuses it on Flush and preserves
the draft. If owners later want live rejection, that should be a separate
editor API change, not a prerequisite for source safety.

## Acceptance matrix

Parser/range behavior:

- edit each of header, first/last body, empty, Unicode/astral, and padded
  cells; assert only `TextRange` changes and all padding/pipes/CRLF remain;
- empty `TextRange` anchor: insert text, delete it again, and replace nonempty
  with empty without changing row/column count or history on no-op;
- CRLF/LF and EOF rows remain byte-identical outside the cell.

Structural safety:

- typing an unescaped pipe is refused and leaves source/history unchanged;
- editing compact `|a` to empty/padding is refused when it would reinterpret
  the pipe and lose the one-cell shape;
- deleting the last non-padding byte of a no-outer row is refused when it would
  reinterpret a neighboring pipe as an outer marker;
- changing only a compact cell is refused if the staged row becomes another
  recognized block, including heading/list;
- `\|`, `\\\|`, and existing escaped pipe content round-trip exactly;
- backtick content containing an unescaped pipe is refused;
- a value ending with an odd backslash immediately before a separator is
  refused, while the same value followed by padding remains valid;
- CR and LF are refused on commit and the draft remains recoverable.

Identity/lifecycle:

- insertion before the table maps and opens the same cell;
- edit in another cell, row reorder, delimiter change, table replacement,
  expired journal, group, Partial, oversize, and readonly are visibly refused;
- duplicate identical tables select the exact captured row/table;
- failed Flush retains draft, selection, and host history;
- successful staged edit session is one host undo; undo restores the exact
  previous cell bytes;
- no-op does not call `Changed`, dirty the document, or add undo.
- Escape commits and validates; explicit discard leaves the host unchanged.

Bounds:

- 4096-byte cell remains valid, 4097 is refused before host write;
- replacing a small cell with a value that would push the table over 32 KiB,
  the document over `markdown.MaxParseBytes`, or the staged parse over a
  line/block/descriptor budget is refused before host write;
- activation and commit require the current full parse to be non-Partial;
- tooltips/preview remain bounded and rune-safe.

This slice should not add cell replacement APIs, multi-cell transactions,
toolbar operations, or Markdown inline decoding. Those can be considered only
after the raw single-cell lifecycle is accepted.
