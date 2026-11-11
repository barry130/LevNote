# Native Markdown tables — approved first-slice contract

Status: ACTIVE first slice after root acceptance of the19:20 doc-block reference
release. This contract supersedes conflicting proposal text. Root plans and
reviews; child agents implement, test and build. Cell editing remains a later
second slice and is not authorized yet.

## Product scope and ownership

Current ownership override: root interrupted GLM after repeated unresolved
minimum scanner regressions. DeepSeek APP owner now exclusively finishes
formats/markdown/** parser/export and focused tests, then returns to actual app
acceptance. GLM is read-only reviewer with no product write ownership. View
DeepSeek remains exclusive markdownview owner. Earlier GLM implementation
assignment below is history; final API/grammar remain unchanged.

First slice: independently parsed, source-preserving pipe tables; a readable
native preview; safe literal HTML export; the existing single whole-block raw
editor. No new renderer dependency, inline-cell links, cell editor, structural
toolbar, formulas, file I/O in format/view, or process-resource matrix.

- GLM formats owner: formats/markdown/** table metadata/parser/export and
  focused tests. Publish the exact additive API before view/app depend on it;
  then independent read-only view review. Do not edit peer files or docs.
- DeepSeek reference-view owner: modules/markdownview/** first-slice rendering,
  source-safe whole-table editor entry, focused native tests and images.
  Preserve accepted task/link/reference behavior. No shell or format edits.
- DeepSeek app owner: finish accepted reference artifact docs, then app/**
  actual registered-adapter table acceptance and final product docs. Product
  shell changes only for a demonstrated relevant issue. Sole final integrator
  after root's next focused review gate.

No child delegation or peer owner restarts. Compact GLM/DeepSeek before200K.

## Authoritative grammar

This is a declared rectangular pipe-table subset. Source is never normalized.
Header plus delimiter with zero body rows is valid.

1. Each physical row uses a bounded linear structural-separator scan with
   local checks of immediate backslash runs: an odd run escapes a pipe; an even
   run leaves it structural. Do not rescan a table suffix after a failure.
   Backticks never protect an unescaped pipe. Display/export preserve escapes
   literally. This does not require literally one physical pass over each byte.
2. Optional outer pipes are independent on each row. A pipe at the first
   non-padding byte is a leading outer pipe, and a structural pipe at the last
   non-padding byte is a trailing outer pipe. An escaped final pipe is content:
   `a | b\|` is valid two-column content. No final outer pipe is required.
3. Accept1..8 columns. A one-column row requires a leading structural outer
   pipe; this applies to header, delimiter and body. Bare `header` / `---`
   is never a table. Internal consecutive pipes represent empty cells. `|`,
   `| |` and `||` each represent one empty cell; `|||` represents two.
4. Up to3 leading ASCII spaces/tabs count as row indentation. Preserve them
   outside cell ranges. More indentation uses visible raw fallback when a
   table-shaped candidate is recognized; do not silently reinterpret it.
5. Cell Range covers the original segment between structural boundaries,
   including its ASCII space/tab padding. TextRange trims the entire ASCII
   space/tab run at each side. An all-padding segment has a stable empty
   TextRange at its right edge. Nothing is removed from the source.
6. A delimiter cell is `:?-{3,}:?` after ASCII trimming. Alignment is default,
   left, center or right. Counts must match the header. A delimiter-shaped line
   has at least a hyphen in every cell and otherwise only colon/hyphen/padding;
   malformed dash counts, colon placement or column counts are recognizable
   failed candidates with a bounded reason. Do not classify arbitrary prose
   containing a pipe as a table without a delimiter-shaped following line.
7. Stop before blank/non-pipe lines or another recognized block start (heading,
   list, fence/doc-block, quote, divider). A non-leading row requires a valid
   lookahead for a table to start, including paragraph interruption. Leading
   pipe runs without a delimiter retain the existing raw fallback behavior.
8. Once a header/delimiter candidate is recognized, consume its contiguous pipe
   body rows. If any row/limit is invalid, represent the whole contiguous
   candidate ONCE as raw fallback, including later pipe rows, until the normal
   boundary. After invalidation collect no more descriptors. Keep one cursor;
   do not repeatedly rescan the remaining document. Resume at the first boundary.

The current document decoder owns BOM/encoding. Format fixtures retain any BOM
bytes in Raw; app acceptance proves actual BOM+CRLF file round trips. No new
encoding policy is introduced.

## Exact additive API

Add BlockTable and Block.Table. Keep these names to avoid peer API drift:

```go
const BlockTable BlockKind = "table"
type TableAlignment uint8
const (
    TableAlignDefault TableAlignment = iota
    TableAlignLeft
    TableAlignCenter
    TableAlignRight
)
type TableCellRef struct {
    Range SourceRange
    TextRange SourceRange
}
type TableRowRef struct {
    Range SourceRange // complete physical row including LF/CRLF
    Cells []TableCellRef
}
type TableRef struct {
    Rows []TableRowRef // header first; delimiter is not a display row
    DelimiterRange SourceRange
    Alignments []TableAlignment
    Problem string
}
```

All ranges are absolute, half-open UTF8 byte offsets into Document.Raw. No
per-cell Text/Inline/Raw copies. Block.Range/Raw cover the complete table and
Block.TextRange may equal Range for the whole-table editor. A recognized failed
candidate is BlockTable with nonempty source-free Problem (<=256UTF8bytes),
nil Rows/Alignments, complete Raw and one capped unsupported-table warning.
A valid table has empty Problem and >=1 header row. DelimiterRange is exact
when present. Missing-delimiter legacy pipe runs may remain raw paragraphs.

Caps:8columns;256body rows;4096UTF8bytes per TextRange;32KiB complete raw table,
inclusive;16384 total table cell descriptors per document including headers.
Check budgets before allocation; after invalidation retain only raw range and
reason. Bound temporary descriptors to one table. Aggregate fallback is a
feature limit, not Document.Partial. Existing overall parse limits still apply.
No speculative cell replacement API in this first slice.

## Preview and whole-table editing

Flatten every valid header/body row into the existing virtual ui.List. Stable
row identity includes source range/kind/table-row distinction; no stale index
or text matching. Skipping delimiter bytes must keep UTF16 conversion monotonic.
One built row has <=8 cells and bounded text/widgets. The first row exposes a
brief literal-source hint and the whole-table action; no per-cell controls.

Use a responsive layout with the current palette:
- Grid only when actual usable row width permits >=112DIP per column (including
  cell padding). Equal-width columns, visibly distinct header, fixed enum
  alignment. Values literal, monospace13, MaxLines3 and visible ellipsis.
- At narrower widths, each physical data row becomes a stacked record with
  bounded header/column labels and literal values; show row/column identity
  even for duplicate/empty headings. The header is visibly distinct. Values
  and labels wrap and truncate visibly, with a route to complete source.
- Use actual native layout width; unknown first-frame width defaults stacked.
  Resize must converge without resetting source identity, losing drafts, or
  causing perpetual frames. No document-wide horizontal RowWidth/panning.
- Tooltips, including ellipsis, are cloned rune-safe strings <=512bytes. Empty
  cells show a muted placeholder. Complete source remains available in the
  existing source view or whole-table editor. Clearly explain literal cell
  display and that ellipsis omits preview text only.

The `编辑表格源码` button uses ordinary pointer/Enter/Space semantics and opens
the one existing independent editor. Capture source Buffer/version/range/rowID
at native gesture start, then Flush and map/revalidate exact current table
identity. Do not read a newly rendered table from a fixed-key click. An unrelated
insertion before may rebase; changed/deleted/reordered target, replaced Buffer,
expired journal, group, Partial or oversize must refuse visibly. Failed Flush
retains draft/recovery. Existing double-click entry must use immutable source
snapshot identity for table rows. No generic input framework is requested.

This is WHOLE-TABLE RAW editing: preserve all original bytes on open/no-op/save;
allow intentional structural source edits, then rebuild preview or show truthful
raw fallback. Do not impose later single-cell shape restrictions here. Readonly
renders preview and complete-source guidance but never creates an editor.
Navigation/resize/preview alone never marks dirty or adds undo history.

Raw-fallback tables show the specific reason and complete bounded raw preview;
over32KiB uses the existing explicit source/editor placeholder. No partial grid.
This is a rendering limit: Block.Raw deliberately retains the complete
authoritative source, including over-limit tables, so source editing and escaped
fallback export remain lossless. Retain a source slice where possible; do not
copy or shape the complete over-limit table merely to render its placeholder.
Release drops table metadata/layout caches alongside current preview; preserve
unsynced drafts according to existing Flush/recovery contract.

## Literal HTML export

Valid table: semantic table/thead/tbody/tr/th/td, HTML-escaped TextRange bytes,
no Markdown/escape decoding, link, image or HTML activation. Alignments map only
to fixed text-align style values. A zero-body table exports its header. Invalid
or over-limit table exports the complete escaped Raw inside pre and retains its
warning. Validate source ranges before slicing; no panic on malformed descriptors.
Keep checked-export Partial refusal. UI and export both describe literal cells.

## Focused gate and integration

GLM: exact independent byte-range oracles (Unicode, indentation, padding, empty
cells, CRLF), optional/mixed outer pipes, single-column, zero-body, escaped final
pipe, backslash parity/backticks, count/delimiter failures, valid block boundaries
and paragraph interruption, every cap and cap+1, many tiny tables aggregate
budget, complete one-block raw fallback, Partial export refusal, escaped hostile
HTML/link/image literal cells and fallback. Test bounded linear work using
existing instrumentation if appropriate; no new general benchmark matrix.

View: native whole-source button pointer/Enter/Space and double-click; duplicate
tables/second target, insertion-before mapping, overlap/stale/replaced/expired
refusal, Compose Flush/conflict/recovery/readonly. Grid/stacked resizing, long
and empty Unicode values, far-table virtualization, bounded active widgets and
cache Release. Native tester light/dark narrow images; record scope honestly.

App: real registered adapter, BOM+CRLF untouched Save, intentional whole-table
source edit Save and undo/redo, source/tab/save/close Flush barriers and conflict
retention, readonly, malformed/overlimit source visibility, literal HTML export,
760x480 split dark/light with sidebar, ordinary prose unaffected by far tables.
Reuse accepted app host behavior; no unrelated shell changes.

Root reviews focused evidence/current source/images and independent findings
before authorizing one full test/vet/release-build/hidden-startup sequence. No
matched Tauri benchmark, Windows IME/GPU acceptance claim or resource resample.
After acceptance, define and activate the second single-cell editing slice.
