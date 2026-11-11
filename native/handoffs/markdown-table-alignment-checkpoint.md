# Selected-column alignment — API and lifecycle checkpoint

Status: DESIGN ONLY, 2026-11-09. Root accepted tag `.4` and authorized this
checkpoint under [the controlling plan](markdown-table-alignment-next-stage.md).
No product/test changes or test/build/process execution in this design step.

## Pure API and exact result

New `formats/markdown/table_alignment.go`:

```go
type TableAlignmentPatch struct {
    Range SourceRange // byte interval of the selected delimiter marker only
    Text  string
    Noop  bool
}

func PlanTableAlignmentEdit(source string, tableRange SourceRange,
    column int, alignment TableAlignment) (TableAlignmentPatch, error)

var (
    ErrTableAlignmentValue   error // not one of the existing four enum values
    ErrTableAlignmentTarget  error // invalid UTF8/range/column or unsupported target
    ErrTableAlignmentLimit   error // source/parse/table/cell bound would be exceeded
    ErrTableAlignmentContext error // unexpected candidate structure/other-table change
)
```

`column` is zero-based. The caller supplies authoritative UTF8 source and an
exact table range, never a trusted descriptor or a table-text search key.
Failure returns the zero patch and an `errors.Is`-compatible sentinel. After
validating the current complete document/table/column, equal semantic alignment
returns `{Noop:true}` with zero Range/empty Text: no candidate copy, normalization
or replacement. A real patch has `Noop:false`, nonempty marker range/Text and
changes only that column. Readonly/history are caller concerns, not pure inputs.

Reject unknown alignment first; enforce source size before parsing, UTF8/range
and exact supported target identity before reading descriptors. Existing fallback
or missing target is Target; oversized/Partial source is Limit. Candidate bound
or newly unsupported selected table is Limit; other unexpected shape/context
change is Context. No parser/type/schema/renderer behavior changes.

Use `scanTableRow` on the exact `DelimiterRange` with its existing `line` shape,
removing only the recognized terminal CR/LF/CRLF via `tableRowEnding`. Reuse
`parseTableAlignments` and require agreement with parsed `TableRef`. The selected
scanner cell's `TextRange` supplies the marker. Borrow its validated hyphen run,
retain its length, add/remove edge colons; retain all surrounding padding/pipes,
indentation, other markers, content and endings. No second pipe scanner.

Before constructing marker/candidate, check positive delta against remaining
4096-byte delimiter-cell,32KiB table and8MiB source capacity; source parsing also
enforces8columns/256body rows/16384global cell descriptors and existing parse caps.
For changes, parse the full candidate once. Require identical block count/order;
selected table rows/cells/text and all other alignments unchanged; only delimiter
marker, table end, delimiter end and subsequent offsets may differ. Header
descriptors stay fixed; body descriptors shift by delta. Reuse
`unchangedTable`/`unchangedTableRow`/`shiftTableRange` where applicable. Compare
unaffected blocks' raw/semantic fields and descriptors, and warnings' code/message
with expected range translation. Other fallback transitions are Context, even
without `Partial`. These checks live in the new helper, not parser modifications.

## Shared native action seam and completion

Extract the existing pipeline from `table_rows.go` into one private
`table_actions.go`: action discriminator, gesture capture/current check,
composition finish, `prepareTableCellDraft` + ordinary Flush receipt, patch and
UTF16 selection commit, input cancellation, and operation-specific error wording.
`table_rows.go` keeps row button definitions/row planner dispatch;
`table_alignment.go` adds the four alignment definitions/planner dispatch.
Both call the same button input handler and mutation pipeline. No separate
alignment epoch, receipt, selection mapper or repeat/multiclick implementation.
View/draft seams rename the existing shared state only as necessary; existing
row tests retain their behavioral oracles (mechanical private-name updates only).

The captured action includes requested alignment and the selected column/current
alignment from the exact cell target, in addition to the existing draft/source
buffer identities, versions, selections and lifecycle. Add verified `column` to
the bounded `tableCellPlan`; no parsed pointer survives Flush. Preserve all current
row guards, including raw ancestor continuation-click detection and scroll handling
that returns false. Source changes before a gesture may rebase via existing exact
cell validation; changes during a held gesture cancel, including unrelated edits.

After finishing composition, validate the captured draft and predicted cell result,
then call existing Flush. Accept only its exact own receipt: same source owner,
release epoch unchanged, clear epoch advanced once, no replacement draft/cell,
readonly/group absent, version unchanged or +1 as predicted, expected selection,
and exact predicted table bytes. Resolve the verified column on this current
source and run the pure planner. No global bypass or callback-dependent cleanup.

**Success and semantic no-op both complete and close the cell panel.** A clean
same-alignment request changes no source/version/history/dirty state and sends no
Changed notification. If cell text changed first, that one edit remains, with its
own undo/Changed; alignment no-op adds none. Do not reopen the old cell after
Changed. A later explicit cell activation opens a new panel normally.

Failed cell Flush retains the exact recoverable draft and makes no alignment
edit. A later alignment refusal leaves a successfully written cell intact and
reports “单元格已写回，对齐未修改”; a clean completed cell may also leave the
panel closed on late refusal. Do not promise a failed draft after successful
Flush. Error text differentiates row versus alignment; local error assignment
must still belong to the captured lifecycle/source/draft so a synchronous
Changed/Report replacement is not overwritten. Real failures remain reportable.

Only a non-noop patch starts a new owned group. Preserve the existing endpoint
rule: before marker unchanged, inside marker clamps to its start, at/after end
shifts by UTF16 delta (right affinity at insert). Map both oriented endpoints,
using source byte boundaries converted to legal UTF16 boundaries. Expose possible
source-caret clamping in the control tooltip. Replace+selection are one undo;
on either failure cancel only this owned group. Never cancel foreign groups or
undo an already successful cell write. Complete local bookkeeping before Changed;
after it, never restore captured draft, caret, preview or selection.

## Compact cell-panel layout

Keep48px input and one fixed30px/two-line F6 hint through composition/focus loss.
Use the existing title line for `第N列 · 当前：默认/左/中/右` plus compact row kind;
full editor identity remains in accessibility/tooltip. Read the bounded existing
cell/preview descriptor. If its provenance is stale, show “待重验” until ordinary
refresh/gesture validation; no whole-source parse solely for idle label painting.

At roughly265px preview width, keep each action row unwrapped: cell finish/
discard/recovery faces `完成`/`放弃`/`恢复` with existing full accessible names;
row faces `插入行`/`删此行` with existing full names; alignment faces
`默认`/`左`/`中`/`右`, each46px ×26px and4px gaps (196px total, below the
approximately229px inner width). Accessibility names identify “第N列…对齐”,
and tooltips include completion/no-op/undo semantics. Existing ordinary block
editor sizing/text is unchanged. Fold header-delete guidance into existing hint/
tooltip instead of adding height. Error detail remains one compact block plus
full tooltip/Warnings; recovery stays before structure actions. No required
control is hidden to make the layout fit.

F6 still enters the first row action; Tab/ShiftTab traverse row then alignment
controls. Current alignment remains activatable to complete a semantic no-op.
During marked composition F6 retains input/focus and shows the fixed-slot confirm
guidance; pointer confirmation must leave button bounds unchanged. Native/app
checks must assert real button bounds above status at760×480 split, not just text
presence. Narrower unsupported layouts are not claimed as fully verified.

## Finite checks and file ownership after activation

- Pure `table_alignment_test.go`: independent expected marker/source bytes for
  all four transitions/no-op; long runs/padding/no outer pipes/one and eight
  columns/header-only; Unicode/escaped pipes/BOM prefix/mixed or absent final
  endings; adjacent prose/tables; invalid enum/column/range/fallback/UTF8. Exact
  4096-cell,32KiB-table,8MiB-document boundaries with colon growth refusal and
  shrinking success; full global-descriptor boundary with later fallback intact.
  Failed/no-op patches have their promised zero fields. No fabricated “readonly”
  pure test; readonly is a native oracle.
- Native `table_alignment_test.go`: real pointer/F6+Tab/Enter/Space, held repeat,
  marked composition + fixed bounds, separately undoable cell/alignment and
  same-alignment closed-panel no-op, exact selection/redo, recoverable invalid
  draft and post-cell limit refusal. Cover one alignment case for shared stale/
  callback/scroll/rapid-click routing, with non-vacuous intersecting hit bounds;
  existing row tests cover the full shared safety matrix. Settled frames retain
  parsed preview identity and schedule no extra frames.
- Scope: new format helper/tests; new native alignment/shared-action files/tests;
  narrow `table_rows.go`, `table_cell.go`, `draft.go`, `view.go` wiring and
  mechanical existing row-test private names. Parser/exporter/core/app production,
  runtime/protocol/scripts/vendor and accepted packages remain frozen. Run focused
  oracles while implementing, then full changed format/view packages + scoped vet
  once stable. Preserve failed attempts. App consumer/captures and a new package
  remain later gates, with no resource/runtime/full-repository replay.
