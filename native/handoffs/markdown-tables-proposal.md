# Native Markdown pipe tables — parser feasibility proposal

Status: **queued/design-only**. This document does not authorize product code,
tests, builds, or a resource run. The current doc-block reference release has
priority. It refines `markdown_tables_stage_plan.txt`; the app-side usability
notes in `markdown-tables-app-proposal.md` remain non-authoritative for parser
grammar.

## Current source facts

- `formats/markdown/parser.go` recognizes a table line only when its first
  non-space/non-tab byte is `|`.
- Every contiguous run of such lines becomes one `BlockParagraph` with an
  `unsupported-table` warning. `Document.Raw` remains byte-exact.
- There are no table, row, cell, alignment, or escaped-pipe descriptors.
- The native preview already has a 32 KiB per-visible-block guard and a virtual
  list of flattened rows.
- The accepted independent draft lifecycle captures an exact UTF-16 field range
  plus an enclosing guard and uses `Buffer.MapRangeSince`; it is suitable for
  a single-cell editor without per-cell editor instances.

## Recommended smallest safe subset

### 1. Strict two-line minimum and bounded rectangular shape

Recognize only a table whose first two physical lines are:

1. a pipe header row;
2. a delimiter row with exactly one alignment cell per header cell.

A header plus delimiter with **zero body rows is valid**.

### Exact candidate grammar and block boundaries

Use one linear parser branch before the current paragraph branch. When a line
is a table-candidate line, inspect the candidate and the following line:

1. If the candidate is not a recognized header or the next line is not a
   matching delimiter, keep the current raw paragraph behavior. The first
   candidate line must not be consumed merely because it starts with `|`.
2. On a valid header plus delimiter, scan body rows only while each physical
   line is a table-candidate row with exactly the header's cell count and all
   cell/row/table caps pass.
3. Stop before the first blank line, first line recognized as another block
   start (heading, list, fence, quote, divider, or doc-block), EOF, or first
   nonconforming/over-limit table row.
4. If any already-collected body row is invalid, the entire candidate is
   represented once as the current raw fallback block covering all its
   physical lines, including the invalid final row, and parsing resumes at the
   next physical line. A valid table resumes at the next line after its final
   valid body row.
5. Never rescan the entire remaining document per candidate. Consume one line
   at a time and retain one cursor. Malformed rows and subsequent tables are
   charged once to their own candidate.

### Candidate row grammar

Define a row after removing the physical LF/CRLF ending but without changing
source bytes:

- up to three ASCII spaces or tabs may precede the first structural pipe;
- a row must contain at least one unescaped structural pipe; escaped pipes are
  content and never make a non-pipe line into a row;
- the first and final structural pipes may optionally be outer pipes;
- a row with a leading outer pipe and exactly one structural pipe is a
  one-column row (its final outer pipe is optional);
- every other accepted row has at least two cells;
- indentation and each cell's padding ranges are retained exactly; no value is
  rewritten.

Examples:

- `a | b` and `| a | b |`: two-column rows;
- `| a |`, `| a`, and `| a \| b`: valid one-column rows because each has an
  explicit leading structural pipe;
- `a \| b`: not a table row because its only pipe is escaped content;
- `a | b\|`: a valid two-column row whose final cell is literal `b\|`;
  because final outer pipes are optional, a trailing escaped pipe is content,
  not an unterminated row.

### Scanner and cell ranges

The first stage accepts at most:

- 2 through 8 columns;
- 0 through 256 body rows;
- 4096 bytes for any one cell content range;
- 32 KiB total raw table bytes, matching the existing visible-block guard.

The one-column exception above is the only exception to the two-column floor.
The header and delimiter must use the same cell count and compatible outer-pipe
form.

Any mismatch or limit reaches the whole-table raw fallback below. No partial
grid is rendered.

Why these bounds: one virtualized table row then constructs at most eight cell
elements; the parser allocates a bounded, predictable descriptor array; and
large or malformed tables remain readable as source rather than becoming a
resource/UI hazard.

### 2. One-pass pipe splitting with backslash parity

Define an unescaped `|` as structural in every cell, including inside a code
span. Do not implement a second code-span-aware splitter.

Scan each row once:

- an odd number of immediately preceding backslashes escapes a pipe (`\|` is
  content);
- an even number leaves it structural (`\\|` has a structural pipe);
- escaped pipes do not disappear from the source or automatically become `|`
  during display;
- because a final outer pipe is optional, a trailing escaped pipe is final-cell
  content and is valid.

This is a declared native subset, not a promise of complete GFM. In particular,
GFM’s code-span pipe behavior differs and must not be inferred from backticks.

### 3. Exact cell ranges and alignment metadata

Use one compact table descriptor rather than per-cell duplicated strings:

```go
type TableCellRef struct {
    // ContentRange is the exact raw-content byte range after structural pipes
    // and one optional left/right ASCII space or tab padding layer.
    ContentRange SourceRange
    // TextRange can be ContentRange for the literal first slice. It exists so
    // a later shared inline renderer can use one exact text interval without
    // duplicating Text/Inline fields.
    TextRange SourceRange
}

type TableRowRef struct {
    // Range includes the complete physical line and its LF/CRLF ending.
    Range SourceRange
    Cells []TableCellRef
}

type TableAlignment string

const (
    TableAlignNone   TableAlignment = "none"
    TableAlignLeft   TableAlignment = "left"
    TableAlignCenter TableAlignment = "center"
    TableAlignRight  TableAlignment = "right"
)

type TableRef struct {
    Rows          []TableRowRef // header first; no synthetic data rows
    DelimiterRange SourceRange
    Alignments    []TableAlignment
    Problem       string
}
```

The literal first slice does not add per-cell `Text`, `Inline`, or `Raw`
strings; the view slices the authoritative `Document.Raw` ranges. `Problem`
is bounded and absent/empty only for the valid table case.

For each accepted row, store:

- one complete-row `SourceRange` including its original LF/CRLF ending;
- one descriptor per cell with:
  - `ContentRange`: the exact half-open byte range of the raw cell content
    after the structural pipes and after trimming any run of ASCII spaces or
    tabs that pads that side of the cell;
  - `TextRange`: equal to `ContentRange` in the literal slice;
  - no copied padding string. The exact boundary derivation is:
    leading row indentation belongs to the row, not a cell; a leading outer
    pipe, if present, precedes the first cell; each structural separator or
    trailing outer pipe bounds the adjacent cell; and only the trimmed
    ASCII-space/tab runs outside `ContentRange` are padding. A future editor
    can derive those bytes from row/cell ranges or export explicit padding
    fields after a demonstrated need.
- the delimiter alignment per column: `none`, `left`, `center`, or `right`.

All ranges are absolute byte ranges into `Document.Raw`. `Raw` is the complete
table source, including BOM elsewhere, outer pipes, padding, backslash escapes,
and CRLF. No parser path rewrites or canonicalizes source.

### Aggregate metadata budget

In addition to per-table caps, keep a document-wide table descriptor budget of
**16384 cells**. Header and body cells count; delimiter cells count only as
alignment metadata, not cell descriptors.

When the budget is exhausted, later candidate tables remain the existing raw
paragraph fallback with a bounded `unsupported-table` reason such as
`Markdown table descriptor limit reached; table source remains visible`. This
is a per-feature fallback and must not set `Document.Partial`: the complete
source is still parsed by the bounded parser and export can truthfully emit the
raw fallback rather than claiming incomplete document parsing.

The parser should check the projected cell count before allocating each row.
Thus many tiny valid tables cannot allocate unbounded metadata under the 8 MiB
input guard.

### 4. Display semantics: raw source first

Use two display stages:

1. **Stage 1A — literal source display.** Each cell shows its raw
   `ContentRange` bytes as text, with a visible hint that the grid displays
   Markdown source and source view is authoritative. This avoids mixing a
   rendered, decoded rich-cell view with an editor that writes raw source.
2. **Stage 1B (optional later) — shared inline rendering.** Reuse
   `parseInline` and the existing `richText` path only for cells within budget.
   Cells with any backslash escape remain literal or fall back to the whole raw
   table; do not selectively unescape only `\|`.

This proposal recommends shipping 1A first. It is honest, low-resource, and
does not claim inline formatting support until the display and editing story is
tested together.

### 5. Flattened preview rows and narrow windows

Do not render an entire table as one list row.

- Emit one virtual preview row per visible table row, in source order.
- Each row constructs at most eight bounded cell widgets.
- The header row is visually distinct, but still a source row.
- Preserve the existing 32 KiB block fallback wording and list virtualization.
- For 760x480, use a fixed equal-width eight-column layout with per-cell
  wrapping and row clipping/ellipsis plus a tooltip/full-source route. A
  horizontal-scroll API can be adopted only if the current native toolkit
  supports a bounded, tested viewport; do not invent a framework or add global
  scroll coupling.

At 8 columns on a 760px window, each cell receives roughly 80px before
padding. That is intentionally conservative: wrapping plus ellipsis keeps text
visible without building a separate large horizontal scrolling surface in the
first stage.

### 6. Single-cell editing seam (stage 2)

Double-click one bounded cell to open the single existing independent draft
editor with:

- exact `ContentRange` as the editable field;
- complete table row as the enclosing guard;
- captured buffer, version, row identity, and current cell bytes;
- one active editor, never one editor per cell.

Before replacing the buffer, `Flush` must:

1. call the existing draft `Flush`;
2. require the same buffer, non-group mode, non-oversize, and complete parse;
3. map the captured cell and enclosing row through `MapRangeSince`;
4. reparse and require the exact table, row, and cell range;
5. validate the candidate cell shape before mutation.

The candidate content must:

- contain no LF or CR;
- contain no unescaped structural pipe;
- preserve the original cell padding bytes when only the content changes;
- leave all pipes, delimiter row, surrounding rows, CRLF endings, and untouched
  cells byte-identical.

A candidate containing `\|` is accepted literally under the one-pass parity rule
and is not silently rewritten to `|`. Newline and unescaped-pipe input visibly
refuse and retain the draft; they must never split the table or rewrite another
cell.

### 7. Whole-table fallback

If any of the following occur, emit the current raw paragraph fallback with a
bounded, specific reason instead of a partial grid:

- fewer than two qualifying lines;
- delimiter row contains an invalid alignment cell;
- delimiter count differs from header count;
- any body row count differs from header count;
- more than 8 columns;
- more than 256 body rows;
- cell content exceeds 4096 bytes;
- table raw bytes exceed 32 KiB;
- the document table-cell descriptor budget is exhausted;
- any backslash parity edge not covered by the declared scanner.

Use a stable warning code such as `unsupported-table` and keep the current
message pattern truthful. The complete source must remain visible and raw block
editing available.

### HTML export

Export valid tables as complete `<table>` markup:

- one `<tr>` per stored row, header first;
- each valid literal cell is `<td>`/`<th>` with HTML-escaped raw
  `ContentRange` bytes;
- no link, inline, or escape decoding in this slice;
- alignment uses the fixed enum: `none` emits no style/attribute, and
  left/center/right map to the agreed HTML mechanism;
- invalid, over-limit, and aggregate-budget-exhausted tables export the
  complete raw fallback, escaped exactly once, inside `<pre>`, with the
  warning retained.

Because fallback is complete and document parsing remains bounded rather than
`Partial`, export refuses on `Document.Partial` but does not refuse merely
because some individual tables use raw fallback.

### 8. Focused test matrix after activation

The plan document already sets reviewable two-step delivery. The parser-only
step should pin:

- header + delimiter with zero body rows;
- LF and CRLF header/body/ending preservation;
- optional and mixed outer pipes;
- one versus two spaces of padding;
- 2-column and exact 8-column positives;
- all four delimiter alignment forms and mixed forms;
- Unicode in headers/body with byte-range oracles, not only self-consistent slices;
- escaped pipe `a \| b` as one cell;
- literal double backslash before a structural pipe;
- no-code-span exception: an unescaped pipe inside backticks still splits/falls
  back under the declared subset;
- exact 4096-byte and 4097-byte cell boundaries;
- exact 256 and 257 body-row boundaries;
- 8 and 9 column boundaries;
- whole-source and `Document.Raw` byte equality.

The view/edit step should add pointer double-click on the second of two
equal-content cells, Flush success/conflict, unrelated insertion, structural
candidate refusal, readonly, partial/oversize, and 760x480 wrapping evidence.

## Explicit non-goals for the first active slice

Row/column insertion or deletion, resizing, alignment mutation, merging,
formulas, table creation toolbars, canonical pretty-printing, multi-cell
selection, per-cell editor instances, and generic table frameworks. Those add
surface area without improving the first safe read/edit workflow.

## Root decision points before activation

1. Confirm the literal-source display option for stage 1A.
2. Confirm 8-column/256-row/4096-byte-cell/32KiB-table limits.
3. Confirm that a body row with a different pipe count remains whole-table raw
   fallback rather than a partial grid.
4. Confirm the equal-width wrapping/ellipsis behavior for narrow windows unless
   the implementation team identifies an existing bounded horizontal-scroll
   primitive.
5. Confirm whether `PaddingRange` should be exported or kept as internal
   parser fields; the existing draft seam only needs `ContentRange`.
