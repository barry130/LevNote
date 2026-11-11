# Native Markdown tables — view feasibility proposal

> 2026-11-08 root decision: the first-slice design below naming
> `ui.List.RowWidth` is superseded. The active contract
> `native/handoffs/markdown-tables-contract.md` requires an actual-width
> responsive choice: an equal grid only when the committed usable row width is
> at least `columns * 112` DIP, otherwise a stacked labelled record. Unknown
> first-frame and resized widths default to stacked, settle in one measured
> frame, and never enable document-wide horizontal panning.

Status: superseded for the active first slice by
`handoffs/markdown-tables-contract.md` and the implementation checkpoint in
`handoffs/markdown-tables-view.md`. The older `RowWidth` proposal and later
cell-editor sketch below are retained as design history only.

## Recommended first slice

Extend the existing `previewRow` flattening instead of building one tall table
widget. A valid table becomes one `previewRow` for its header and one for each
body row. Each row has a line-level `text.Range` for stable List identity and
passes its mapped numeric row ID through the existing
`ItemKey(func(int) any)` callback. The document's existing
`ui.List(...).Rows(...)` stays the sole vertical virtualizer.

Use the current palette and a plain reading surface: a distinct header row,
one-pixel column separators, monospace literal cell Markdown, and two or three
wrapped lines per cell with a rune-safe ellipsis. Cell text needs no inline
Markdown, escape decoding, or link controls in this slice.

The concrete narrow container is
`ui.List(...).Grow(1).MinHeight(0).Clip().TrackScroll(&v.scroll).RowWidth(w)`.
`RowWidth` turns on the List's horizontal axis and keeps the header and body
rows at the same offset. That offset is shared by ordinary document rows; this
avoids coupling several independently measured scroll states. A `128`-DIP
fixed cell plus `6`-DIP gap makes an
8-column row `1,066` DIP wide; a 12-column row is `1,602` DIP wide. The app's
minimum window is 760 DIP. With its default 228-DIP sidebar and the documented
50/50 split, each pane is about 266 DIP before the preview's 36 DIP horizontal
padding, leaving about 230 DIP in the table viewport. With the existing 265-DIP
sidebar fixture, it is about 211 DIP. Eight 40-DIP cells would expose only a
few characters, so this design intentionally shows about one and a half
readable columns and one honest horizontal scrollbar instead of squeezing all
columns into the viewport. Their width never falls below 128 DIP.

Build a source row with concrete MyGo layout calls:

```go
row := ui.Row(c).MinHeight(44).AlignItems(ui.Stretch).ClipX()
for _, cell := range cells {
	box := ui.Box(c.Key(cellRange)).Width(128).Shrink(0).Clip().
		Padding(6, 8).BorderEndWidth(1).BorderColor(c.Theme().Border)
	box.Children(func() {
		ui.Text(c, literalMarkdownSource).Font("monospace").FontSize(13).
			LineHeight(1.35).MaxLines(3).Ellipsis("…").Tooltip(boundedSource)
	})
}
```

MyGo measures each built List row from its tallest cell. `MinHeight(44)` gives
short and empty rows a consistent floor, `AlignItems(ui.Stretch)` makes every
cell share that measured height, `Clip()` prevents drawing outside a cell,
and `MaxLines(3).Ellipsis("…")` makes every cut visible. `Tooltip` carries a
rune-safe source copy of at most 512 bytes plus an explicit ellipsis; the
whole-table source editor remains the route to all bytes. Empty cells show a
muted `（空单元格）` marker. No manual shaping or height callback is needed.

The first slice needs a visible whole-table action in addition to the existing
double-click path. Make the header preview row a `ui.Column` containing a
small `ui.Row` toolbar and the header grid. In writable documents the toolbar
uses a keyed ordinary
`ui.Button(c.Key(rowID), "编辑表格源码").Height(26).Label("编辑表格源码").Clicked()`;
click passes the immutable block snapshot to the existing `begin`, which maps
and validates its complete `BlockTable.Range` before opening the sole
`editor.Editor`. Readonly documents show `只读表格` and no editor control.
Double-clicking any table row keeps the same `begin` behavior.

## Parser seam

One additive metadata shape is enough for rendering, safe HTML export, and the
later cell editor. `markdown.Parse` remains the entry point; the view and app do
not need a table-specific document or parser registry.

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
	// Range is the exact byte interval between structural pipes, including
	// source padding. It is empty when the cell is empty.
	Range SourceRange
	// TextRange is Range after trimming only ASCII space/tab. Replacing this
	// range is the later cell-edit operation; the untouched padding remains.
	TextRange SourceRange
}

type TableRowRef struct {
	// Range includes the complete source line and its LF or CRLF.
	Range SourceRange
	Cells []TableCellRef
}

type TableRef struct {
	// Rows[0] is the header and Rows[1:] are body rows in source order.
	// The delimiter remains metadata and is never rendered or edited as a row.
	Rows []TableRowRef
	DelimiterRange SourceRange
	Alignments []TableAlignment
	// Empty means the recognized table is rectangular and renderable.
	// A recognized but malformed/over-limit shape keeps Block.Raw authoritative
	// and carries one bounded explanation here.
	Problem string
}

type Block struct {
	// Existing fields unchanged; add:
	Table TableRef
}
```

Recognition should produce `BlockTable` only when a header and delimiter are
present. A leading pipe run with no delimiter remains the current paragraph
with its existing unsupported warning. Once a table attempt is recognized,
`TableRef.Problem` is the single per-block fallback seam, analogous to
`DocBlockRef.ReferenceProblem`; the view shows the complete raw block and the
bounded reason. Header plus delimiter with zero body rows is valid.

Cell boundaries require the agreed one-pass backslash-parity splitter. The
parser, view, and export must share that decision; code spans do not make an
unescaped pipe safe. `TextRange` lets a later edit preserve the original outer
pipes, spacing, CRLF, and every neighboring byte.

Recommended finite parser limits for root selection are 128 header-plus-body
rows, 12 columns, and 2,048 UTF-8 bytes per cell. Exceeding any limit marks the
whole recognized table with a bounded problem so the view falls back to raw
source. The existing 32 KiB `maxPreviewBlockBytes` check remains a separate
view fallback before any grid widgets are built.

## Later cell-edit seam

Stage 2 can reuse `View.draft`, `draft.ShapeLimit = 2048`, Flush/rebase and the
draft recovery controls. There must still be exactly one editor instance.
The cell target should capture Buffer, version, preview row ID, mapped table
and cell UTF-16 ranges, row/column index, and the complete bounded original
cell source. The enclosing table range and delimiter range identify the
construct; text equality and table `id` do not.

Before committing a replacement, the view needs one pure parser-owned
candidate check with this behavior:

```go
func ValidateTableReplacement(
	source string,
	table SourceRange,
	cell SourceRange,
	replacement string,
) error
```

It should return nil only when replacing the captured cell keeps the recognized
table rectangular with the same row count, column count, alignment vector, and
target coordinate. It must reject candidate newlines and structural unescaped
pipes under the final grammar. It copies at most the bounded table fragment,
retains no source, and does not mutate or serialize the document. On success,
the existing Buffer replacement runs once; on failure, the same draft,
selection, history, and recovery controls remain and the refusal is visible.

An unrelated insertion before the table maps both table and cell ranges. An
edit inside the table or captured cell, a replaced Buffer, partial/oversize
parse, deleted/reordered row, changed delimiter, or expired range history
refuses visibly without a same-coordinate fallback. Pointer and keyboard cell
activation capture on the actual gesture start and follow the reference/link
lifetime discipline already established in `markdownview`.

## Export and acceptance framing

Safe HTML export belongs with the parser/format checkpoint: emit a semantic
`table`/`thead`/`tbody` structure, escaped literal cell text, and alignment
derived from `TableRef.Alignments`. A zero-body table exports its header.
`Problem` tables export their complete source as escaped preformatted text.
The view applies no competing export interpretation.

Focused activation should prove: valid zero-body and body tables; BOM + CRLF
and optional outer pipes; escaped and structural pipes; raw fallback for
malformed, non-rectangular, and each limit; an untouched byte-identical save;
narrow light/dark geometry; virtualization with a far table; and, only in
stage 2, equal-text cell identity, insertion-ahead rebase, overlap refusal,
candidate pipe/newline refusal, Flush/save/switch/close barriers, readonly
preview without an editor, and same-View draft recovery.
