# Native Markdown tables — app feasibility proposal

Status: design-only, not active. No table parser, view, app product code, or
table test is implemented or authorized by this document.

## Root decisions on this proposal

- The native format rule does not inherit the old Milkdown engine's
  header-plus-body minimum. A structurally valid header and delimiter with zero
  body rows is a valid table and must display and edit its header cells safely.
  A body-row minimum, if needed later, is a view/product choice rather than a
  parser validity rule.
- Code spans are not an automatic safe-cell exception. In particular, an
  unescaped pipe inside inline code is not GFM pipe-table-safe merely because
  it is inside backticks. The implementation stage must define an explicit
  accepted escape grammar and test it before claiming code-span protection.
  Until then, candidate cells containing structural pipes must be refused
  rather than silently rewritten or accepted under an inferred grammar.

## Current baseline

The native Markdown parser does not model tables. In
`formats/markdown/parser.go`, `isTableRow` recognizes only a line whose first
non-space byte is `|`; contiguous such lines become one paragraph and emit
`unsupported-table`. `Block` has no table/row/cell metadata. The native view
therefore shows the raw pipe source and warns that table rendering is
unsupported.

The accepted view already has a recoverable block draft:

- double-click a preview row starts an independent `core/editor.Editor`;
- the source block is captured by Buffer identity, version and mapped UTF-16
  ranges;
- a clean edited field is rebased through `MapRangeSince`;
- an overlapping source edit returns a conflict while retaining the draft and
  its `完成编辑` / `打开未同步副本` / `放弃草稿` controls;
- `Flush` is the lifecycle barrier used by save, mode changes, tab changes and
  close.

App save is also already the correct barrier. `App.Save` calls `flushTab`
first, then `Document.Save`/`SaveAs`. A table cell editor must therefore write
only to the document Buffer and mark it dirty through the existing change
callback; it must never write files, canonicalize the whole table, or bypass
the app save path.

ADR 0017 supplies the relevant old-product constraints: pipe tables have a
structural header row, alignment is column-wide and comes from the delimiter
row, a practical editable table has at least a header plus one body row, and
row/column operations are a later product layer. The native MVP should read and
edit basic pipe tables only; it should not reproduce the ProseMirror table
toolbar or serializer.

## Proposed MVP

1. Add neutral table metadata in the Markdown format layer, not the app shell:
   one table block with exact source range, header/body row descriptors in
   source order, delimiter/alignment metadata, and each editable cell's exact
   content range and table shape. Keep CRLF, outer-pipe choices, escaping,
   spacing and column padding as source bytes.
2. Render a bounded native grid. Header cells are visually distinct; body rows
   follow the delimiter's left/center/right alignment. Inline formatting may
   use the existing inline spans, but this stage needs no row/column commands,
   add/delete/reorder, resizing, drag selection, or alignment controls.
3. Double-clicking a cell opens one native editor for that cell's raw Markdown
   source slice. Editing two cells at once is not needed. The delimiter line is
   never an editable cell. Readonly documents render the grid without opening
   an editor.
4. A cell target must own Buffer identity, source version, row identity,
   enclosing table/cell UTF-16 ranges, and the original cell bytes. Equal text
   in sibling cells is irrelevant: source coordinates choose the target.
5. Flush must validate before replacing the source. Reparse and require the
   same Buffer, table location, row/column coordinates, delimiter shape and
   cell range after mapping. A replaced Buffer, overlapping source edit,
   deleted/reordered table, expired history or partial parse must retain the
   cell draft and refuse visibly.
6. Commit should validate the complete candidate table before touching the
   host Buffer. An unescaped pipe or embedded newline can change the table's
   row/column structure. Basic safety therefore means rejecting such a candidate
   with a visible reason, or requiring the user to write `\|`, rather than
   silently escaping text or rewriting unrelated cells. Loading and saving
   untouched tables must remain byte-exact.
7. Preserve the existing `Flush`/save contract. A failed cell Flush must make
   Save/mode/tab/close fail without writing the file. A successful edit becomes
   one normal Buffer transaction, participates in undo/redo, and then follows
   the existing BOM/CRLF/encoding-aware `Document.Save` path.

The app should not need a second table-specific API. The existing real
registered Markdown adapter remains the integration seam. Any editor/view
contract addition should be justified by an actual implementation need rather
than added speculatively.

## Parsing and source-safety boundaries

The minimum parser must be stricter than "split every `|`":

- recognize the ordinary GFM header + delimiter + body shape, including
  optional outer pipes and bounded leading indentation;
- recognize the explicitly agreed escaped-pipe grammar when identifying cell
  boundaries; code-span pipe handling requires its own agreed grammar before it
  can be accepted;
- require matching header/delimiter column counts and a rectangular editable
  body;
- retain every cell's exact source range, including surrounding whitespace;
- refuse or visibly fall back for malformed, non-rectangular, unterminated or
  beyond-budget tables;
- retain the current `32 KiB` per-preview-block behavior and define finite
  row/column/cell bounds so one native List row cannot construct an unbounded
  number of widgets.

Alignment is read from each delimiter cell (`---`, `:---`, `:---:`, `---:`)
and is display metadata only in this stage. A header-and-delimiter table with
zero body rows is structurally valid under the native format rule and must
display and edit its header cells safely.

## UX risks

- **Raw versus decoded cell text:** showing decoded rich text but accepting
  edits to raw Markdown can surprise users. The safest MVP label is that the
  editor edits the cell's Markdown source. Full inline editing is a later
  feature.
- **Structural characters:** an unescaped `|` or newline can turn one cell into
  two cells or rows. Silently escaping it rewrites bytes; allowing it breaks the
  grid. Validation plus a visible refusal is the least surprising finite rule.
- **Whitespace and padding:** trimming or re-rendering cells can alter every
  line. Only the target cell's exact range may change.
- **Duplicate content:** matching by text, row number after a reparse, or
  `id`-like content is unsafe because tables have no source IDs. Only mapped
  ranges and exact row/column coordinates can identify a cell.
- **Large tables:** virtualization at the block level is not enough if one
  row builds thousands of cell widgets. Bounded fallback must be visible.
- **Narrow screens:** fixed cell widths can clip content; wrapping can destroy
  row alignment. The MVP needs an explicit horizontal scroll or another bounded
  behavior, verified at `760x480`, before it claims usability.
- **README/source familiarity:** grid rendering must not hide the fact that
  source remains authoritative. Unsupported shapes should stay visible as
  source with a reason, not as an empty or invented table.

## Minimum actual-app acceptance after activation

1. Open a BOM + CRLF Markdown table through the real registered adapter; render
   header/body without the unsupported-table warning. An untouched Save keeps
   the file byte-identical.
2. Double-click the second of two equal-text cells; edit, finish and Save.
   Assert exactly that cell changes, all other bytes and alignment formatting
   remain identical, and the saved file is the expected BOM/CRLF encoding.
3. Edit a cell, then perform Save, source mode, another tab and close. Each
   barrier either commits exactly once or visibly retains the draft. Undo/redo
   sees one normal source transaction.
4. While a cell draft is pending, insert an unrelated heading above the table.
   The draft maps and saves to the intended cell. Overlap-edit the cell/table in
   source instead: the draft remains recoverable and no wrong cell is written.
5. Replace/Restore the Buffer or Reload while a cell draft is pending; stale
   coordinates never reach the new Buffer.
6. A readonly table renders but cannot open/edit a cell and preserves source,
   selection, history and dirty state.
7. Malformed, non-rectangular, partial and over-limit tables remain visible
   source with a clear reason. No empty grid, stale prior table or cell editor
   is shown.
8. Attempting a newline or unescaped pipe shows the chosen structural refusal
   and preserves the draft. Pipes accepted by the final agreed escaped-pipe
   grammar are displayed and saved without reinterpretation. Code-span pipes
   are tested only after that grammar is explicitly defined.
9. Far table in a large document remains virtualized, and a narrow/dark
   `760x480` capture proves header, cells, editor, warnings and status controls
   do not overlap or lose the active filename.

## Explicit non-goals

- adding/removing rows or columns, resizing, alignment changes or cell merging;
- table creation templates, toolbars, drag selection or spreadsheet formulas;
- canonical Markdown table formatting;
- rich inline editors for every cell;
- background workers or a separate table document/session format;
- implementation or testing until root activates this stage after the current
  doc-block reference release.
