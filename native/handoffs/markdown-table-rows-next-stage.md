# Native table body-row actions — focused gate accepted

Status: FOCUSED GATE ACCEPTED2026-11-09. Root read the final native source,
meaningful exact-byte/identity/input oracles and actual successful streams.
Native attempt05 passed1.218s; final format0.647s/view2.518s and vet exit0;
result0AAA6BB85C64239070D2D429FEC6DA082BD205C95E42967DFE1534A7290DF004.
Seven checked source files stayed unchanged. Consumer work now follows the
separate integration task; package delivery is still queued. Scope below is
the completed focused stage, not authorization to keep changing product code.

Historical focused activation2026-11-09:
Root accepted resource final docs and the API/algorithm checkpoint in
markdown-table-rows.md (75F30A1A117899534D2EC9317A3A7EBEA6426E434822256DD57613D5CEBAB141).
Only astra_marker_view owns implementation/execution. New table_edit* and
table_rows* plus narrow shared cell-validator/draft/view lifecycle changes are
authorized; parser/renderer/app production/runtime/protocol/editor core/scripts/
vendor and accepted outputs remain frozen. Publish pure then native focused
checkpoints while proceeding with focused validation. Later app integration and
new package wait root focused source/oracle review; no resource reruns.
The GLM design task produced no ACK or handoff and was interrupted; this is
root's plan from current source, not an independent design endorsement. All
implementation/test/build execution is delegated.

## Useful bounded behavior

Extend only the existing selected-cell draft panel with `在下方插入行` and
`删除此数据行`. No permanent toolbar per row/cell. Header selection inserts the
first body row after the delimiter; deleting the header is disabled with a
short visible explanation. Body selection inserts below or deletes that exact
body row. Header-only/one-column tables remain valid. No columns/alignment,
formula, sorting, rich cell interpretation or full GFM expansion. Existing
whole-table source editing remains available.

Use the same native focusable controls, compact wrapping layout and recovery
panel. One small hint explains that body-row actions first write the current
cell draft and can be undone. Read-only does not create a cell draft and no row
mutation is available. Invalid/oversized/partial table/source remains source
fallback, not a normalized table. Error text separates successful cell writeback
from a later rejected row operation.

## Current reusable paths and actual constraints

- `formats/markdown/table.go` owns table recognition, limits and exact row/cell/
  delimiter byte descriptors. `TableRowRef.Range` includes original line ending.
  Table.Rows[0] is header; delimiter is separate. A header-only table is valid.
- `modules/markdownview/table_cell.go` captures a bounded table source, full
  descriptors, selected row/column and host Buffer identity. `currentTableCellContext`
  maps both table/cell ranges, compares exact source, then reparses actual full
  context. It rejects *any* overlapping external table change, intentionally.
- `syncCellDraft` validates a bounded single-cell patch then calls ReplaceUTF16.
  On success `finishTableCellDraft` clears both `cellEdit` and draft. Therefore a
  row action MUST NOT merely Flush and reuse its old row index or target: the
  old entire-table range will be invalidated by our own cell edit.
- `View.drawDraft` is a fixed native panel. Existing `clearDraft/Release` must
  cancel any new held row-action gesture and release bounded captured state.
- Text `BeginGroup/EndGroup` allows one row patch plus mapped source selection
  to be recorded as one undo. `CancelGroup` cannot be called for a foreign group.

## Pure edit proposal

Add one pure format helper returning a byte-range patch plus replacement text
for an exact supported current table/row/action; it never writes files or a
Buffer. Exact API is the implementing child's choice, published before code.
Validate against source and current parser descriptors, not a caller's guessed
row index alone. For all failures return no patch, no mutated document/model.

New empty row syntax uses leading and trailing pipes with one ASCII space per
empty cell, e.g. `| | |` for two columns, retaining the anchor row's allowed
indentation. Existing rows/cells/delimiter/spacing are untouched. Header inserts
after delimiter. Prefer anchor line's exact LF/CRLF/CR ending; for an unterminated
last line use closest preceding table ending (delimiter/header), default LF only
when none exists. If the old final line is unterminated at EOF, insert a separator
before the new row and leave the new final row unterminated. Otherwise insert
new row plus the selected ending at row.Range.End.

Delete only the selected body physical row including *its* ending, never the
header/delimiter/preceding row's ending. If an unterminated final row is deleted,
the previous surviving line's original newline remains. Explain this exact
patch rule and test it rather than globally adjusting EOF newline style.

Before commit, parse/check the bounded candidate table with existing8column,
256body-row,32KiBtable,4096bytecell constraints and exact expected one-row shape
change. Existing global8MiB/line/block/descriptor limits also must hold for the
actual candidate document. Preserve every byte outside the intended patch.
In particular, the global descriptor-cell budget can make a later valid table
become `Table.Problem` without setting Document.Partial. Adding an earlier row
must not silently degrade that later table. Check the full candidate's affected
table descriptors, or the exact equivalent document-wide budget, and include a
regression at the shared descriptor limit. The action path may do bounded
candidate work; unchanged frames may not clone, parse or serialize full
table/source repeatedly.

## Capture, own-Flush reconciliation and transaction contract

At native pointer/key down capture action, exact current `cellEdit`/draft object,
draft Buffer/version/selection and host Buffer/current version plus bounded table
identity. A held gesture must not follow a new draft/selected cell after redraw,
scroll, mode/tab switch, Release or external edit. Enter/Space repeats cannot
double-activate. All cancellation clears the capture.

Before host Flush, validate the captured target against the current authoritative
source using currentTableCellContext and verify the same draft and input version
still exist. Complete native composition through its own editor seam, enforce
the existing cell candidate checks, and determine the expected exact table
source after only that cell replacement. Reuse/refactor the existing validation
only as narrowly needed; avoid a second incompatible definition of cell edits.

Then Flush using normal rules. A conflict keeps exact pending draft, recovery,
source and row action uncommitted. If Flush succeeds, accept *only* the expected
own cell write (zero changes for no-op, one version step for actual replacement)
to the same Buffer and exact predicted table bytes, anchored at the already
resolved source start and revalidated through current complete parser context.
The current cell ReplaceUTF16 also changes host selection to its own insertion
end. Distinguish this expected own-selection change from callbacks changing
selection; state which original source selection is deliberately mapped through
cell and row patches. Callbacks may synchronously change/release/replace the view or buffer: capture
and recheck lifecycle generation and identity, never retain stale Node/block
pointers. Any additional edit or changed selection rejects visibly. Do not use
same label, same row index or whole-source search to recover an ambiguous row.
An implementation may use a narrow internal commit receipt instead of predicting
the patch, but it must prove the same exact source/identity conditions.

The cell Flush and row patch are separately undoable; no transaction spans
native events. Plan/check row patch on the authoritative candidate; refuse
foreign group before beginning ours. Map both ends of source selection through
the UTF16 patch, retaining orientation and valid rune boundaries: before stays,
after shifts by delta, deleted interior clamps to patch start, insertion at an
endpoint uses an explicit consistent affinity. Apply patch + selection in one
owned group; failures roll back our group only. Clear stale cell/row action and
reparse normally, preserve preview location; do not silently edit a replacement
cell. A successful insertion may focus its new empty cell only if current
descriptors prove identity, otherwise preserve preview and let user select it.

## Staged acceptance

First pure patch/source tests with independent exact-byte expectations: LF,
CRLF/CR, BOM in document prefix, Unicode, escaped pipes, padded/no-outer rows,
0/1/max body rows, one/eight columns, final unterminated row, unchanged adjacent
prose/blank lines, malformed/partial/forged identities and all limits. New row
only is deliberately normalized; existing bytes must be exactly preserved.

Then native view pointer/keyboard tests with pending composition and normal cell
draft, zero/one cell Flush plus one row undo, conflicting same-table/source edits,
outside-source insertion, duplicate rows, held gesture target changes, callback
release/replacement, history expiry, foreign group/read-only and limit refusal.
Ensure failed drafts remain recoverable; successful own Flush does not promise
to undo itself when the later row patch is refused. Selection before/after/inside
deletion and reversed/astral endpoints need real assertions. Idle unchanged
frames must not request continuous redraw or clone the full table.

Finally real app save/reopen with BOM/CRLF and source/preview/split barriers,
Undo/Redo and HTML semantics plus two narrow light/dark screenshots with usable
buttons and recoverable error. Reuse existing export worker acceptance when the
pure renderer remains unchanged; no redundant protocol/process lifecycle suite.
Root reads focused implementation/oracles/evidence before authorizing a fresh
versioned package. No new deps, network access, copy of another editor engine,
arbitrary executable plugin support or unrelated feature work in this slice.
