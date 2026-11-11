# Markdown table columns: bounded next increment

Status: READ-ONLY PLAN, 2026-11-10. Implementation is not active. This plan
permits no product edits, tests, builds, launches, package work, or `.24`
changes until root activates a separate implementation task.

## Decision

**GO** for one-column-at-a-time insert-before, insert-after, and delete-current
on supported, equal-width pipe tables in `native/formats/markdown`.

**NO-GO; retain whole-source editing** for ragged/malformed tables, any
`BlockTable` with `Table.Problem`, unsupported syntax such as an unescaped pipe
inside inline code, partial documents, stale or forged source identities, and
any operation that would violate a table, document, column, or descriptor limit.
The planner checks the complete candidate document so it also refuses a change
that would promote, demote, or otherwise reparses another table or block.

The increment adds exactly one column action per gesture. It does not add row
reordering, resizing, multi-column selection, another table model, parser
syntax, a new editor dependency, or normalization of source outside the
necessary structural splice.

## Reuse and ownership

Keep the proven architecture used by `PlanTableRowEdit` and
`PlanTableAlignmentEdit`: reparse the authoritative source, use caller ranges
only as exact identities, produce original-source patches, construct and parse
the full candidate, then reject any unexpected structural or warning change.
Reuse `Parse`, `TableRef`, `TableRowRef`, `TableCellRef`, `SourceRange`,
`tableRowEnding`, `structuralAt`, `text.UTF16Len`, `text.Buffer` grouping, and
the existing `markdownview` selected-cell action lifecycle.

New product files:

- `native/formats/markdown/table_column_edit.go`
- `native/formats/markdown/table_column_edit_test.go`
- `native/modules/markdownview/table_columns.go`
- `native/modules/markdownview/table_columns_test.go`
- `native/app/markdown_columns_app_test.go`

Narrow existing-file changes:

- `native/modules/markdownview/table_actions.go`: add the column action kind,
  represent a successful plan as one or more original-source edits, and apply
  all edits in one buffer group. Extend only the refusal text needed to say
  “列未修改”.
- `native/modules/markdownview/table_rows.go` and `table_alignment.go`: only
  adapt their existing single patch to the common edit list. Their behavior and
  labels remain unchanged.

Do not change the parser/renderer, app runtime, core editor, protocol,
filesystem, package, `view.go`, or introduce a new UI framework.

## Format API

Add a patch-only API shaped like this:

```go
type TableColumnAction uint8

const (
	TableColumnInsertBefore TableColumnAction = iota + 1
	TableColumnInsertAfter
	TableColumnDelete
)

type TableColumnEdit struct {
	Range SourceRange
	Text  string
}

type TableColumnPatch struct {
	Edits []TableColumnEdit
}

func PlanTableColumnEdit(
	source string,
	tableRange, anchorRowRange SourceRange,
	column int,
	action TableColumnAction,
) (TableColumnPatch, error)
```

`tableRange` identifies the table and `anchorRowRange` identifies the selected
physical header/body row. Reparse must find both exact identities and exactly
one targeted cell at `column`. A failed plan returns the zero patch. Define
typed action, target, limit, and context errors like the row planner; wrappers
in `markdownview` translate them into concise user-facing refusals.

Validate unknown actions; invalid UTF-8; source/table/ranges; partial parsing;
unsupported target; column bounds; `column == 1` for delete; `column == 8` for
insert; the 32 KiB table limit; the 8 MiB document limit; and table/global
descriptor growth. Every source row must still have exactly the header width.

## Exact source transformation

Operate on parsed `Cell.Range` and structural pipe locations. Never split a row
with `strings.Split`. This preserves escaped `\|`, backslash parity, Markdown
inside cells, Unicode, indentation, unknown bytes, and the existing distinction
between unsupported and supported inline-code syntax.

Bounds depend on the textual layout because all permutations are supported:

1. Insert before column 0 at `Cells[0].Range.Start`, adding the minimum
   separator plus empty-cell bytes required by the row's leading-pipe form.
2. Insert after a non-last column at that cell's `Range.End`, immediately before
   its existing following separator.
3. Insert after the last column at `Range.End`. With an existing trailing outer
   pipe, one separator suffices. Without one, add both the separator and the
   outer boundary needed to represent the new cell under the current parser.
4. Delete column 0 together with its following structural separator; delete
   another column together with its preceding structural separator. Never
   delete an outer marker or another cell's bytes.

For header and body rows, the inserted cell is empty; for the delimiter row it
is exactly `---`, which has default alignment. Existing delimiter spelling,
alignment markers, cell padding, and indent remain untouched. If a two-column
table has no leading boundary, deletion to one column must add the one leading
pipe required by the existing parser; that structural byte is recorded in the
same grouped patch. A one-column source is never transformed by deletion, and
insertion before/after it is supported.

Do not append or normalize a missing final newline. The patches themselves do
not create line endings; they only splice within each physical row, so LF, CRLF,
CR, a final terminated line, and a final unterminated line all remain as they
were.

Before returning, build the candidate from all edits and parse it. Require:

- no document partial state and no new warning;
- the same block and physical-row counts;
- exact expected column count and alignment sequence;
- empty inserted header/body cell text and `---` inserted delimiter text;
- exact unchanged text/raw bytes for every unaffected cell;
- translated ranges for the target table and otherwise unchanged parsed blocks,
  warnings, and tables;
- no target fallback or change in another table's supported/fallback state.

Applying the original-source edits in descending offset order, or an equivalent
that cannot shift a later patch, is mandatory.

## Existing selected-cell transaction

Use `captureTableAction`, `tableActionGestureCurrent`, `activateTableAction`,
`prepareTableCellDraft`, the post-`Flush` source/table identity receipt,
`BeginGroup`, `ReplaceUTF16`, `SetSelection`, `EndGroup`, and the existing
callback/release/read-only/epoch invalidation. Add column planning at the same
point where row and alignment planning currently occur.

For a changed cell draft:

1. `draft.Flush()` writes the cell as its own undo transaction and may close the
   panel.
2. Reparse and plan the column against the flushed authoritative source.
3. Apply all per-row column edits and the final mapped selection inside one new
   `BeginGroup`/`EndGroup`.

Thus a clean cell plus a column operation has one undo step; a changed cell plus
a column operation has two separately undoable steps. A flush/conflict failure
must leave the draft, source, selection, history, and recovery controls intact.
A successful cell flush followed by a refused column plan reports “单元格已写回，
列未修改” and does not erase the first undo.

Map both selection endpoints through the edits using the existing insertion
affinity rule, set the mapped selection once before `EndGroup`, and emit
`host.Changed` once after the group. Verify the buffer is still the captured
buffer, not readonly, not in a foreign group, and still at the expected
version/selection immediately before grouping. A failure inside our group uses
`CancelGroup`; no callback runs inside the group.

## UI and keyboard

Reuse the selected-cell panel and `bindTableAction`. After the existing row
buttons and before alignment, draw one wrapped action row with these focusable
buttons:

- current column insert-before;
- current column insert-after;
- delete current column.

Use short visible labels and longer accessible labels/tooltips. Disable all
three when `host.ReadOnly`; disable insertion at eight columns; disable delete
at one column. Unsupported/fallback tables continue to expose the full table
source editor and no column buttons.

Keep the existing F6 target unchanged: F6 must still first focus “在下方插入行”.
Tab/Shift+Tab then reach the two row actions, three column actions, and alignment
actions in deterministic order. Enter and Space activate, held/repeated keys do
not duplicate the operation, and mouse press/release uses the same validated
gesture path.

Retain the 30-DIP fixed-height action hint. During live IME composition F6 must
not steal focus, and pointer activation must wait for the normal composition
confirmation seam without moving the pressed button between down and up. The
wrapped column row must fit in the existing panel at the established 760-DIP
and narrow 265/300-DIP test widths.

## Preservation, drafts, and user-visible behavior

- The format layer never reserializes a table. Only inserted structural bytes
  or the deleted target cell plus one separator change.
- Neighboring prose, another table, list/doc-block bytes, BOM handling, CRLF,
  warnings, and all unrelated spans retain their exact source meaning.
- A pending draft uses the existing stable panel position and finite recovery
  path. Conflict, readonly, stale source, limit, foreign group, release, and
  callback replacement all refuse without losing the draft.
- Mode switch, tab switch, Save, or close uses the existing draft barriers; the
  column increment adds no second flush path.
- No disk mutation occurs before normal Save. Save/reopen writes the exact
  candidate bytes without normalizing the rest of the document.

## Focused behavioral tests

`table_column_edit_test.go`:

- exact before/after/delete patches for `| A | B | C |`, no-leading,
  no-trailing, both outer-pipe variants, escaped `\|`, escaped pipe in inline
  code, Unicode, tabs/indent, and all three newline forms;
- one-column insert and delete refusal; eight-column insert refusal and
  seven-to-eight success; two-column no-leading deletion to a valid one-column
  form;
- delimiter `---`, existing `:--`/`--:`/`:-:` preservation, and expected
  alignment insertion/deletion;
- ragged/malformed/unsupported/fallback refusal with zero patch;
- unknown action, bad column/table/row range, stale/forged identities, partial
  document, table/document/descriptor limits, and another-table
  promotion/demotion refusal;
- unaffected adjacent prose, another table, known/unknown bytes, and warnings.

`table_columns_test.go`:

- native pointer and F6/Tab/Enter/Space routes at wide and narrow widths,
  deterministic focus order, disabled one/eight-column and readonly cases;
- live composition geometry and confirmation, plain-cell one undo,
  changed-cell two undos, selection mapping, and exact undo/redo source;
- draft conflict, foreign group, outside edit, limit/refusal after cell flush,
  held/repeated keys, scroll/cancel, release, and callback replacement;
- no duplicate `Changed` callback and no source/history mutation on refusal.

`markdown_columns_app_test.go`:

- one real app route using a BOM/CRLF fixture with leading/trailing prose and a
  supported table: perform insert and delete through the real adapter, verify
  dirty state and unchanged disk bytes before Save, then Save and reopen;
- verify visible column count/content, exact neighboring source, mode-switch
  flush behavior, and byte-identical reopen. This one route is enough; do not
  duplicate the entire pointer/draft matrix at app level.

Run only the packages touched by these new tests during implementation, then
the affected app/view suite and `go vet` when root releases the combined final
test gate. No package build, host launch, resource run, or `.24` execution is
part of this plan.

## Implementation order and stop conditions

1. Add and test the format planner against exact fixture strings.
2. Adapt the common action patch list and add the column UI adapter.
3. Add focused view tests for transaction, focus, IME, and preservation.
4. Add the single real-app save/reopen route.
5. Freeze source identities and run the authorized affected suites/vet.

Stop rather than broaden the parser if any fixture reveals a new syntax class.
Stop and preserve the fixture/evidence rather than rewriting a whole document.
If a candidate cannot prove exact unaffected bytes and parse structure, refuse
the operation and leave complete source editing available.
