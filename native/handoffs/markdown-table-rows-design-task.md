# Native Markdown table rows — read-only next-feature design

Assigned to a new GLM-5.3/crouter design agent. Root plans/coordinates/reviews;
implementation/tests/builds belong to child agents. This task is DESIGN ONLY.
Do not edit product code, tests or existing documentation, run commands that
execute code/build/test/launch, or spawn children. You may read source and write
only `native/handoffs/markdown-table-rows-design.md`. Reply with a brief ACK and
an incremental finding within a few minutes. Compact before200K context.

Context: MyGo native Surface, self-written text/format cores, local-only,
source authority and low idle memory. Markdown has bounded literal pipe-table
preview, whole-table source draft and individual cell source draft. There is
no row/column toolbar. We want the next small user-visible improvement after
the active memory/worker acceptance, rather than an unbounded formatting suite.
Existing accepted .2 product remains frozen. Your proposal does NOT authorize
implementation, packaging or any runtime test.

Read the root/native AGENTS ownership plus current table/cell format and native
view code, actual app draft barriers, the accepted table/cell contract/handoffs,
and README capability boundary. Propose just inserting/deleting BODY ROWS in
already supported tables, preferably directly from a selected cell's existing
native draft area so no always-visible toolbar or per-cell permanent control
tree is needed. Header deletion/replacement, columns, alignment, formula,
sorting, rich inline interpretation, paste-grid, arbitrary GFM and new package
dependencies are out of this slice.

Specify:

1. Exact current source files/functions that can be reused and a minimal pure
   format edit seam using existing byte/UTF16 ranges. Preserve bytes outside the
   intended row edit, BOM/CRLF/LF, neighboring whitespace/delimiters/column text.
   Define new empty row syntax and newline behavior including unterminated last
   rows. Do not rewrite all existing rows simply to insert one. Specify what
   happens for header-only table, one-column table, zero-body rows and last row.
2. Interaction/semantics: how row is selected, below/above insertion if needed,
   delete action wording/discovery, limits/read-only and recovery. Prefer one
   consistent below/selected-row operation over several redundant controls.
3. Captured identity and Flush ordering: existing cell draft may be pending or
   conflict. An action cannot follow a changed row by same text/index. Source
   version/range/log history invalidation, same-version Buffer replacement,
   inactive view and gesture held through redraw require explicit treatment.
   Header-row selection must not imply deleting the header. A rejected action
   leaves source/history/selection and failed pending draft intact.
4. Source patch + one undo transaction for a successful row action; a prior
   pending field Flush may legitimately create its own transaction. Define
   selection mapping after insert/delete and avoid stale local cell draft
   references. Noop navigation/repaint creates no source change.
5. Existing8-column/256-body-row/32KiB/4096-byte-cell and global document caps;
   candidate validation before commit; no per-frame whole table/source copy.
   Keep source fallback for unsupported input; no silent normalization.
6. Meaningful format/view/app acceptance cases (not just mirrored helpers),
   pointer/keyboard/narrow-light-dark checks, save/reopen/undo and metadata/
   untouched source bytes. Recommend the smallest test set warranted by risk.

Cite current implementation locations and identify any actual constraint that
makes the UI proposal unsuitable. You may simplify further if the current
contract naturally supports a smaller useful slice. Do not invent results,
claim independent review of the memory stage, or assume product code changed.
