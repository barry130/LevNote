# Markdown single-cell editing — independent review

Status: FINAL current-source review, 2026-11-08 21:31 local time. GLM did not
run tests or edit product files. Root accepted the focused cell gate and the
app owner is running the sole final integration.

## Reviewed checkpoint

Current files read:

- `native/modules/markdownview/table_cell.go` (19,930 bytes, 21:20:07)
- `native/modules/markdownview/table_cell_test.go` (22,414 bytes, 21:15:03)
- `native/modules/markdownview/view.go` (14,997 bytes, 21:20:54)
- `native/modules/markdownview/draft.go`
- `native/modules/markdownview/tables.go`
- `native/modules/markdownview/preview_limit.go`

The authoritative contract is `native/handoffs/markdown-cell-contract.md`.

## Correction to the earlier interim report

The interim report incorrectly treated `View.Release` after a failed staged
Flush as a data-loss/lifecycle defect. That conclusion was wrong:

- `app.replace` returns at `!flushTab` before its changed branch. A successful
  cell Flush clears the draft before a replace can change the host, and a later
  failed Flush likewise returns before `releasePresentation`.
- `editor.Editor.ReleaseView` only drops geometry/focus and increments the view
  generation; it retains the local buffer, text, selection, and undo state.
- `View.Release` retains `v.draft` and `v.cellEdit` on failed Flush. The next
  `Draw`/`refresh` rebuilds the preview and fixed draft panel while preserving
  the staged text and recovery controls.
- The contract’s “do not clear/release on a failed Flush” barrier is enforced
  by app navigation actions through `flushTab`; a direct cache `Release` may
  drop parsed preview state while retaining the recoverable draft.

No proven data loss or destroyed recovery context results from the current
failed-Flush Release path.

## Current actionable source findings

None. The current implementation has no independently verified product defect.

The bounded post-21:20 view changes are consistent with the contract:
`Warnings()` uses a cell-specific `单元格草稿尚未写回：` prefix for a staged
cell, and the staged notice keeps the one-line notice while moving the
view-local/session-scope explanation to its tooltip. The immutable host table
preview remains drawn below the one staged draft panel.

`TestTableCellHeldBufferGroupExpiryPartialAndReleaseRecovery/failed-release-recovery`
covers the direct failed staged-Release case for both invalid (`坏|值`) and
stale same-table content. It asserts that the same draft and local
selection/history survive, host source/version/selection/history remain
unchanged, parsed preview cache is dropped, recovery controls are visible
again, and host recovery returns the exact local text. The real-adapter test
also blocks Save, export, source mode, tab switch, and close on invalid staged
input while retaining the view, source, selection, and recovery controls. The
required test oracle is therefore present.

## Corrected historical findings

The earlier root findings are present as corrections in current source and are
not repeated as defects:

- `tableCellTargetOf` compares the preview row span with the selected
  `sourceRow.Range`, not the whole table block.
- frame-time target construction retains numeric descriptors only;
  `captureTableCellTarget` clones bounded source and range identity only at an
  actual native gesture.
- candidate translation subtracts `target.tableRange.Start` before applying the
  replacement byte delta.
- separate start/end translation permits empty anchors and length changes.

The 21:20 target-cell envelope change is also consistent with the accepted
first-no-outer-cell padding-reclassification case: only the target cell may
fall inside its translated original envelope, while every non-target row/cell
and text range must match its delta-translated descriptor exactly.

## Verified implementation areas

- Header/body boxes are `RoleColumnHeader`/`RoleCell`, explicitly
  `Focusable()` and `TabStop(true)`, with bounded labels and action hints.
- Tab/Enter and double-click use the same captured target and
  `activateTableCell`; Space does not open a draft.
- Capture happens only at the native gesture, not per visible frame.
- Staged `OnChange` is local and does not call `syncDraft`, mutate the host, or
  add host history.
- `Flush` routes cell drafts through `syncCellDraft`.
- The 4096-byte raw draft cap is checked before `String()`.
- Invalid UTF-8, CR/LF, and unescaped structural pipes are rejected while the
  draft remains editable.
- Table/document size checks precede candidate construction/parse.
- Candidate parsing requires exactly one complete, non-Partial table consuming
  the complete candidate, with unchanged row counts, per-row cell counts, and
  alignments.
- Non-target source descriptors are translated by byte delta; target changes
  are confined by exact replacement of the captured `TextRange`.
- Successful commit performs one `ReplaceUTF16`, one `Changed`, and closes the
  draft; no-op commit performs neither.
- Read-only, insertion-before rebase, same-table conflict, cancellation,
  replaced buffer/group/Partial/oversize paths refuse visibly.
- `Release` and oversize cache cleanup clear table-cell gesture/activation
  state without discarding a live staged editor.

## Test/evidence status

The current test file has meaningful native coverage for pointer/keyboard
activation, header/body/empty/stacked cells, duplicate tables, local staging,
one-undo commit, failed-then-corrected Flush, read-only, insertion rebase,
same-table conflict, cancellation, 4096/4097 bytes, CRLF/EOF, compact
one-column empties, odd/even backslashes, shifted non-target ranges, and
light/dark staged/invalid states.

GLM did not execute tests. The owners published focused evidence:

- `go test ./modules/markdownview -count=1 -timeout=120s -v`: PASS,
  2.404s;
- `go vet ./modules/markdownview`: clean;
- complete focused `^TestMarkdownCells` app suite: PASS, 2.010s;
- `go vet ./app`: clean;
- final component and integrated light/dark images were published and visually
  inspected, including retained table context, staged/refused controls, and the
  local-only recovery-scope tooltip.

Root accepted this focused gate. Final full integration remains the app owner's
authorized sole sequence and is not covered by this review.
