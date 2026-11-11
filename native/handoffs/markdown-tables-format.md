# Markdown tables — format parser takeover

2026-11-08 takeover acknowledgement: root interrupted GLM after repeated
minimum-scanner failures. DeepSeek app now exclusively owns
`formats/markdown/**` parsing, export and focused tests under
`table_parser_takeover_task.txt`; GLM is read-only review. The additive API and
grammar above remain frozen.

Next action: replace the conflicting scanner branches with one bounded
separator/count pass, add the three exact minimum positives first, then run the
cap/boundary/fallback/export matrix and full owned package plus vet.

Status: **focused parser/export checkpoint green; previous0.601s result is not
acceptance and is retained only as history**.

Owner: DeepSeek app, temporarily `formats/markdown/**` only. No view, shell,
vendor, app product, docs, or resource work during this correction.

## Exact additive API

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
    Range     SourceRange // original cell segment, including ASCII padding
    TextRange SourceRange // same segment after trimming ASCII space/tab padding
}

type TableRowRef struct {
    Range SourceRange // complete physical row, including LF or CRLF
    Cells []TableCellRef
}

type TableRef struct {
    Rows           []TableRowRef // header first; delimiter is not a display row
    DelimiterRange SourceRange   // complete delimiter physical row
    Alignments     []TableAlignment
    Problem        string         // bounded; nonempty for recognized fallback
}

type Block struct {
    // existing fields omitted
    Table TableRef
}
```

All ranges are absolute, half-open UTF-8 byte offsets into `Document.Raw`.
Valid tables have empty `Table.Problem`, non-nil `Rows`, and one alignment per
cell. A recognized failed table is `BlockTable` with nil rows/alignments, a
bounded nonempty problem, complete `Raw`, and one `unsupported-table` warning.
Legacy pipe runs without a valid delimiter-shaped following line remain the
existing raw paragraph fallback.

## Current parser behavior

- Header plus delimiter with zero body rows is valid.
- One to eight columns are accepted.
- Up to three ASCII spaces/tabs are row indentation and stay outside cell
  ranges; deeper indentation fails the table-shaped candidate visibly.
- Leading/trailing outer pipes are optional and may differ per row.
- One-column rows require a leading structural pipe.
- `|`, `| |`, and `||` are explicit one-cell empty rows; `|||` is two empty
  cells.
- Backslash parity is one-pass: odd backslashes escape a pipe, even backslashes
  leave it structural. Backticks do not protect a pipe.
- `a | b\|` is a valid two-column row; the escaped final pipe is content.
- Cell `Range` includes padding; `TextRange` trims all ASCII spaces/tabs.
- All caps are enforced before descriptor retention: 8 columns, 256 body rows,
  4096-byte text ranges, 32 KiB raw table, and 16,384 aggregate cell
  descriptors.
- A failed contiguous candidate becomes one complete raw fallback block and
  includes later contiguous pipe rows. It does not mark the document partial.
- Valid tables stop before blank lines or other recognized block starts.

## Export behavior

Valid tables export semantic HTML with:

- `<table>/<thead>/<tbody>/<tr>/<th>/<td>`;
- literal `TextRange` bytes, HTML-escaped;
- fixed enum alignment styles only;
- no Markdown decoding, links, images, or activation.

Invalid, over-limit, and aggregate-budget fallback tables export the complete
escaped `Raw` in `<pre>`. Export validates all rows, counts, ranges, and enum
values before writing any table markup, so a corrupt descriptor cannot erase
preceding export output. `ExportHTMLChecked` still refuses `Document.Partial`.

## Prior non-acceptance evidence

Command:

```powershell
native/scripts/go.ps1 -GoArguments @('test','-count=1','./formats/markdown')
```

Result:

```text
ok  	levnote.local/native/formats/markdown	0.601s
```

This is a compile/API checkpoint, not the final focused gate. Byte-range and
cap/export regressions are being added next.

## Minimum-positive unblock checkpoint

The scanner was replaced with one bounded structural-separator/count pass.
Leading and trailing optional outer pipes are removed from logical cell
boundaries, non-leading rows accept one or more structural separators, and
single/triple-pipe empty rows retain the contract's one/two-cell identities.

Executed from `native`:

```powershell
.\scripts\go.ps1 -GoArguments @('test','./formats/markdown','-run','^TestTableMinimum','-count=1','-timeout=120s','-v')
```

Actual output summary (exit0):

```text
--- PASS: TestTableMinimumLeadingOuterEOF (0.00s)
--- PASS: TestTableMinimumNoOuter (0.00s)
--- PASS: TestTableMinimumOneColumnAndEmptyPipes (0.00s)
PASS
ok  levnote.local/native/formats/markdown  0.471s
```

This unblocks the view owner's focused execution. It is not yet the final
parser/export acceptance checkpoint; the cap and export matrix remains in
progress.

## Focused parser/export checkpoint

The final scanner correction rejects trailing-only one-column rows such as
`a |` / `--- |` as a visible raw fallback; two-column `a | b` remains valid.
The bounded row scan records only up to the small fixed separator set needed to
classify a candidate, then performs a separate local left scan through each
pipe's immediate backslash run. It does not retain an unbounded separator slice.

The final table-focused test matrix covers:

- exact mixed-outer, indentation, padding, Unicode and CRLF source ranges;
- leading-outer EOF without a final newline and no-outer paragraph interruption;
- one-column leading-pipe requirements, `|`, `||`, `|||`, and trailing-only
  fallback;
- odd/even backslash parity, backticks remaining literal/unprotected, malformed
  delimiter, body count, indentation, nine-column and contiguous raw fallback;
- 8/9 columns, 256/257 body rows, 4096/4097-byte cell text, exact32 KiB/+1
  complete raw tables and the16,384-cell aggregate budget;
- valid/zero-body/literal-hostile semantic HTML, escaped complete raw fallback,
  corrupt descriptor fallback after prior content, and checked-export refusal
  for a Partial document.

Executed from `native`:

```powershell
.\scripts\go.ps1 -GoArguments @('test','./formats/markdown','-count=1','-timeout=120s')
.\scripts\go.ps1 -GoArguments @('vet','./formats/markdown')
```

Actual results:

```text
ok   levnote.local/native/formats/markdown  0.634s
vet exit0 (no output)
```

`gofmt -d` and `git diff --check` are clean. This is the current parser/view
integration checkpoint; app acceptance has not yet been executed.
