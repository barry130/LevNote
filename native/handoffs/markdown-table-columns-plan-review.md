# Markdown table-column plan review

Status: read-only review, 2026-11-10. No source, tests, builds, launches, or
product documentation were changed.

## Verdict

The plan is directionally sound and reuses the right parser/action seams, but it
needs five concrete corrections before implementation:

1. Use table width, not the zero-based column value, for the one/eight-column
   limits. Delete is invalid when `len(alignments)==1`, and insertion is invalid
   when `len(alignments)==8`; `column` itself must only be checked as
   `0 <= column < len(alignments)`. The plan’s `column == 1` / `column == 8`
   wording is an off-by-one error.
2. Derive insert bytes from the scanner’s leading/trailing form. For
   insert-before on a row without a leading pipe, insert the new outer `|` at
   the original first-cell start after the existing indent, then one padding
   byte; do not add padding before that pipe, because an existing 3-space
   indent would become invalid 4-space indentation. For a row with a leading
   pipe, insert the new separator at `Cells[0].Range.Start` after that outer
   marker.
3. Protect insert-after-last from an odd trailing-backslash cell. A valid
   non-last cell ending in an odd number of backslashes necessarily has padding
   before its structural separator, but a last cell without a trailing pipe can
   end directly in backslashes. When inserting after the last cell, start the
   patch with a padding byte before the new `|` (for example ` | `) so the new
   separator remains structural and the unchanged cell bytes are preserved.
4. Handle no-leading first-column deletion explicitly. After deleting the first
   cell and its following separator, the next cell’s inherited padding can become
   row indentation, or a one-column remainder can become a block opener. The
   candidate reparse will catch this, but the planner should not silently fail
   only at reparse: either refuse that row form or insert the required outer `|`
   at the original first-cell start, preserving every surviving cell byte.
   Document which choice is used; do not let the deletion normalize the
   surviving cell.
5. Keep the supported matrix finite. Replace “all permutations are supported”
   with the actual scanner matrix: 1–8 equal-width rows, optional
   leading/trailing pipes except that a one-column row requires a leading pipe,
   indent ≤3, valid delimiter cells, and the existing cell/table/document limits.
   Unsupported or fallback tables remain refused with source editing available.

## Transaction and verification corrections

- Represent a successful column plan as one ordered list of non-overlapping
  original-source byte edits, one per physical row plus any required outer-boundary
  edit. Apply those edits to the buffer in descending byte order inside one
  `BeginGroup`/`EndGroup`; compute every UTF-16 coordinate from the same original
  source before the group starts.
- Map the captured selection through all edits, not just one. Compose the
  existing point-mapping rule over non-overlapping edits sorted by source start,
  then set the final selection once before `EndGroup`. Any `ReplaceUTF16` or
  `SetSelection` failure must `CancelGroup`; no callback may run inside the group.
- Candidate verification must compare the exact source bytes of every unaffected
  cell, the surviving outer markers/indent, and translated ranges—not merely the
  parsed cell count or text. It must also require the expected target-table
  column count/alignment sequence and unchanged warnings and non-target blocks.
  This is the guard against silently losing an outer marker or changing
  untouched cell bytes.
- Keep the existing cell-draft behavior: flush is one undo step, the column edit
  is a second grouped undo step, and a refused column plan after a successful
  flush must report that the cell was written back but the column was not
  changed.

No parser redesign is needed. The narrow planner can be built on the existing
`scanTableRow`, `structuralAt`, candidate-parse, and grouped-action contracts.
