# Markdown tables independent review

Review time: 2026-11-08 about 20:00 (Asia/Shanghai)

Scope: current read-only inspection after the parser takeover. I did not edit
product/test files and did not run tests. The earlier GLM scaffold is
superseded; findings below apply only to the current files.

## Status

- Current parser implementation is a valid in-progress checkpoint, not an
  accepted release. I saw the owner-reported three-minimum-positive result
  (`0.471s`), but I did not re-run it and this review does not substitute for
  the complete focused parser package/vet gate.
- Current view tests are also an in-progress owner gate. Their source claims
  cover pointer/keyboard/cancel, exact-table rebase/stale refusal, duplicate
  tables, double-click, fallback identity, resize, readonly, failed-Flush
  recovery and Release, but independent execution is not part of this review.

## Parser/export review

No actionable defect found in the current `formats/markdown` implementation.

Verified from source:

- The current scanner accepts the three minimum shapes: leading-outer/zero-body
  at EOF, no-outer two-column with paragraph interruption, and one-column empty
  rows. Bare `|`, `| |`, `||`, and `|||` use the expected one/two empty-cell
  boundary arithmetic.
- A trailing-only one-column candidate is rejected with the specific
  one-column reason. No-leading rows with one structural separator correctly
  produce two cells.
- Leading/trailing padding is retained in `TableCellRef.Range`; `TextRange`
  trims ASCII spaces/tabs. Row ranges include complete LF/CRLF endings.
- Backslash parity and the final escaped pipe are handled as required. Backticks
  do not suppress a structural pipe.
- Header/delimiter mismatch, malformed delimiter, over-indentation, body count
  mismatch and over-column candidates become one bounded complete raw fallback.
  The cursor advances once; no later contiguous pipe row is rescanned.
- Column/body/cell/raw/aggregate caps are checked before or while building the
  current table, with bounded separator metadata. Aggregate fallback leaves
  `Document.Partial` false.
- Block/paragraph interruption uses recognized boundaries and does not absorb
  headings, lists, quotes, fences or dividers.
- Export validates row/cell/alignment ranges before writing a table, preserves
  already-exported blocks on corrupt-descriptor fallback, and emits literal
  escaped cell text with fixed alignment values only. `Problem` fallback does
  not add a false invalid-descriptor warning.

One contract-ambiguity note, not a code defect:

- For a recognized table over the 32 KiB raw cap, `tableFailed` necessarily
  retains the complete `Raw` so export can show it in `<pre>` and whole-source
  editing remains possible. The preview itself does not materialize it again,
  but the contract sentence “over32KiB uses the existing explicit
  source/editor placeholder” does not say whether `Block.Raw` itself may
  exceed the view’s 32 KiB display cap. Owner/root should state this explicitly
  (likely: keep authoritative complete `Raw`, but render only the existing
  bounded raw preview/placeholder and refuse grid).

## View/source-safety review

No actionable defect found in `modules/markdownview/tables.go` or its relevant
`rows.go`, `view.go`, `draft.go`, and `preview_limit.go` integration.

Verified from source:

- Preview rows use immutable numeric row IDs plus source range/kind/table-row
  identity; the delimiter row is not flattened into display rows.
- Valid table action targets derive the whole-table UTF-16 span from the
  captured physical row by subtracting the exact UTF-16 prefix. Fallback action
  targets use the complete fallback row. `captureTableTarget` captures buffer,
  version, row identity and the exact complete source clone before activation.
- `activateTable` Flushes first, refuses readonly and grouped/current/stale
  sources, and only then opens the current exact block via `MapRangeSince`.
  Failed Flush propagates and retains the existing draft/recovery path.
- Pointer-up/click requires the same buffer and row ID as the captured gesture.
  Canceled, scroll, readonly-after-press and virtualized/missing-button paths
  clear the gesture. Read-only renders guidance and never opens an editor.
- Reentrant Release swaps the list state pointer, drops parser/rows/table
  gesture/layout caches, and does not mutate an active immutable snapshot.
- Oversize preview refusal happens before whole-source materialization, clears
  table caches, and still permits the independent draft lifecycle.
- Grid uses actual committed row width and >=112 DIP per column; unknown first
  frame and narrow widths use stacked records with one settle frame. Resize
  does not rebuild source identity. Literal labels, tooltips and ellipses are
  rune-safe and byte-bounded.

## Concrete follow-up recommendations

1. Treat the 32 KiB raw-cap `Block.Raw` note above as a documentation/API
   clarification; no implementation change is required unless root chooses a
   stricter no-authoritative-Raw rule.
2. Keep the current parser takeover gate to the agreed focused markdown package
   tests and vet. Do not cite this review as execution evidence.
3. The current source-specific test matrix is materially aligned with the
   contract and includes the previously reported minimum positives, cap
   boundaries, fallbacks, hostile literal export, readonly, failed-Flush and
   Release. I did not independently run it.
