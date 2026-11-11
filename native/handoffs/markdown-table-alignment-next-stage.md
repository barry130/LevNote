# Markdown selected-column alignment — active design plan

Current status: FOCUSED ACCEPTED; APP CONSUMER ACTIVE2026-11-09. Root read actual
source/oracles/logs and handoff5FF1C9EF...; final format0.809s/view2.499s/vet0,
10inputs unchanged, result0A96BE7B3A5BBD6FEC902655BC9773D10B16AC3ECD114397BBC705DA2C07D917.
Proceed under markdown-table-alignment-consumer-next-stage.md. Product sources
freeze pending demonstrated consumer/review defects. No package execution yet.

Status: FOCUSED IMPLEMENTATION ACTIVE2026-11-09. Root read and accepted
markdown-table-alignment-checkpoint.md against current parser/cell/row code.
Astra alone implements its pure/helper/shared native action scope and finite
focused checks, then changed format/view packages and vet. Publish pure and
native progress while proceeding. App consumer/captures/package remain queued
until root reviews actual source/oracles/logs. Preserve all prior acceptance.
When sharing row actions, retain existing behavior assertions; short visible
button labels keep full accessible names and independent disabled explanations.

Prior status: DESIGN ACTIVE2026-11-09. Mindmap tags.4 delivery and final documents
are accepted, separate acceptanceF85856EF258080747528B6FA4B4720B500C4324BBEDE5F6D1C2543B407128D06.
Astra alone owns a new markdown-table-alignment-checkpoint.md describing exact
pure API/no-op/errors, existing scanner reuse, shared row/column action lifecycle,
compact layout and finite meaningful checks. Root plans/reviews only. Publish
the checkpoint for root review before product/test changes or execution. Keep
accepted bundles and evidence immutable; no further tag-package work is needed.

## Product purpose and boundaries

The existing native parser, responsive table preview and HTML exporter already
support default/left/center/right column alignment. Provide a compact control
for that existing meaning in the selected-cell editing panel, completing a
common table editing action without adding another renderer or dependency.
The old product's TableToolbar has the same column-oriented behavior; use the
interaction requirement as a reference, not its third-party editor implementation.

Only the selected column changes, across header and body rows. No column
insertion/deletion, sorting, merging, width manipulation or grammar expansion.
Show the selected column number and current alignment. Four concise native
actions may use default/left/center/right labels with explicit accessibility
names. Keep the current row controls and recovery route usable at760x480,
including split mode. Do not add per-cell permanent toolbars or animation.
Reuse F6 entry to existing table actions and normal Tab traversal. Composition
must be confirmed before keyboard focus escape; pointer focus loss must not
move the pressed controls. Publish compact layout/API choices first.

## Existing source paths

- formats/markdown/table.go: scanTableRow and parseTableAlignments already own
  the recognition and bounds. TableRef exposes exact DelimiterRange, while its
  delimiter cells are intentionally not in Table.Rows.
- formats/markdown/table_edit.go: PlanTableRowEdit reparses authoritative source
  and the complete candidate, including other tables' descriptor budgets.
- modules/markdownview/table_cell.go: tableCellTarget and prepareTableCellDraft
  prove the exact table/cell and predict the owned cell-write result.
- modules/markdownview/table_rows.go: existing capture, composition guard,
  own-Flush receipt, separate undo, selection mapping and raw continuation-click
  protection. Do not duplicate a divergent version of this lifecycle.
- formats/markdown/export.go: writeTableCell already emits literal escaped
  cells with the recognized text-align; renderer changes are not expected.

## Pure, source-preserving edit

Propose a pure alignment patch helper in a new table_alignment.go. Caller
passes authoritative source, exact table range, zero-based selected column and
one existing TableAlignment value. Publish its exact return/no-op/error contract
before implementation. Validate the table and column with the existing parser;
never trust fabricated descriptors, locate a same-label table, or introduce a
second pipe scanner.

Change only the selected delimiter cell's non-padding marker. Preserve its
existing hyphen run count, adding/removing edge colons for the requested
alignment. All spaces/tabs, pipe presence, other delimiters, row contents,
indentation and exact CR/LF/CRLF endings remain unchanged. Example: a padded
` :-----: ` switched to right becomes ` -----: `; to default becomes ` ----- `.
If its semantic alignment already matches, return an explicit no-op and keep
every byte; do not normalize a hyphen run or whitespace for a no-op.

Use the existing8column/256body/4096cell/32KiBtable/8MiBdocument limits. A colon
can cross a limit, so validate the bounded full candidate before offering a
patch. Prove table row/column counts and cell text are unchanged, exactly one
alignment changed, and other parsed blocks/tables remain equivalent with only
expected byte-offset translation. Shared descriptor-budget or fallback changes
must fail visibly. A failed/no-op plan cannot mutate source or allocate an
unbounded snapshot. No parser or serializer behavior change is needed.

## Draft, identity and undo contract

Design checkpoint must explain the interaction after an alignment action:
the existing row-action Flush closes the cell panel, so do not accidentally
promise it stays open or recreate a stale draft after callbacks. Prefer a
clear existing-panel completion rule and reuse the tested lifecycle. A semantic
no-op may complete the current cell edit but must not add an alignment undo,
mark an unchanged document dirty or lose pending input on refusal. State this
in the checkpoint and test the visible result rather than only a helper return.

The current panel uses48pxinput plus status/recovery/row controls. At760x480
split width four long buttons can wrap enough to push recovery below the
viewport. Propose short visible labels and explicit accessible names using a
bounded row/layout, with fixed hint geometry through composition focus loss.
Show current selected column/alignment from the already bounded descriptor;
idle frames must not parse the document merely to label the controls.

Extend the existing selected-cell action lifecycle narrowly, with explicit
action kind and selected column/alignment if a shared private seam is useful.
Do not weaken current row safety. Capture the exact cell draft/input Buffer,
version/selection, source Buffer/version/selection, displayed table and column,
and view lifecycle at native down. Cancel on scroll, switch, Release, repeated
keys and continuation clicks; a stale gesture must not follow another cell.

Finish input composition, validate the exact draft and predicted cell result,
then use ordinary Flush. Re-resolve the column against that proven current
table only. A successful cell write is its own undo; alignment is another.
If the cell was written but the alignment is later refused, say so explicitly.
Failed cell Flush retains its draft and recovery. Foreign groups stay intact.
A semantic alignment no-op does not add a history entry; any legitimate prior
cell write remains separately undoable.

Map both source selection endpoints through the delimiter patch in UTF16 with
the existing explicit affinity rule and legal surrogate boundaries. Apply patch
and selection in one owned group. Synchronous Changed callbacks may release,
replace or edit the view; never restore captured old drafts or selections after
such callbacks. Idle frames only read bounded presentation; no repeated full
table cloning/parse. Document any unavoidable source-caret clamping.

## Finite acceptance and later delivery

Pure independent exact-byte fixtures cover all four alignments, semantic no-op,
long hyphen runs, padding/no-outer pipes, one/eight columns, header-only table,
Unicode/escaped pipes in cells, BOM prefix and mixed/unterminated endings,
adjacent prose and another table, forged identity, readonly caller boundary and
limits. Expected strings must not be generated with the new helper itself.

Native tests cover pointer and keyboard, pending cell/composition plus separate
undo/redo, same-alignment no-op, same-table conflict and unrelated-source edits,
stale held target, callback Release/replacement, scroll and rapid continuation
cancellation, foreign group, selection mapping and bounded idle behavior.
Reuse existing tested safety when its implementation is unchanged rather than
replaying unrelated process/editor suites. Finish changed format/view packages
and scoped vet after demonstrated fixes; root reviews actual source/oracles/logs.

Then real registered-adapter app checks prove BOM/CRLF save/reopen/no-op,
source/preview/split and literal HTML column styles; capture light/dark narrow
controls and a recoverable refusal. Preserve failures and state that simulated
input is not OSIME/GPU acceptance. Package/version assignment waits consumer
acceptance. No resource comparison or arbitrary executable-plugin scope.
