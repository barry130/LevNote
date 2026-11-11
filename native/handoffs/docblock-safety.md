# Doc-block safety checkpoint (2026-11-08)

Owner: GLM (`formats/markdown/docblock.go` and focused tests only)

## Change

- A `doc-block` fence without the required `---` separator no longer slices
  header lines backwards and panics.
- Malformed/missing-separator content is now emitted as a `BlockCode` block
  containing the complete original source. This uses the existing readable
  code rendering path instead of an empty doc-block tile, so malformed source
  remains visible and source-equivalent.
- A warning states that the separator is missing and the source is retained as
  raw code. Unterminated input adds the existing unterminated warning.
- A closed malformed block consumes the closing fence line (`closeIndex + 1`),
  so following prose is parsed normally and the fence is not reparsed.
- An unterminated malformed block consumes the remaining line range once and
  does not return an overlapping index.
- Valid `---` doc-block parsing keeps its metadata, payload, CRLF handling, and
  raw-source semantics. Its cursor is deliberately corrected as well: a closed
  valid doc-block consumes its closing fence with `closeIndex+1`, and an
  unterminated valid doc-block consumes the remaining line range once. This
  fixes continuation after a valid closing fence; it is not an unchanged
  continuation behavior.

## Focused evidence

Commands:

```powershell
native/scripts/go.ps1 -GoArguments @('test','-count=1','./formats/markdown')
native/scripts/go.ps1 -GoArguments @('vet','./formats/markdown')
```

Results:

- `ok levnote.local/native/formats/markdown 0.657s`
- vet completed with no diagnostics.

Regression coverage:

- Missing separator at EOF, empty open fence, malformed closed fence, LF
  headers at EOF, and CRLF malformed headers at EOF.
- Closed malformed fence followed by prose; verifies the closing line is
  consumed exactly once.
- Valid CRLF doc-block with payload and following prose; verifies attributes,
  exact payload/range, and continuation behavior.

Current hashes:

- `native/formats/markdown/docblock.go`
  `F37CDB498F5FF781A0E6FD3BB926DCDA0CC06E4D7DBDBB316460519736514E8A`
- `native/formats/markdown/docblock_test.go`
  `D89B4CF1D3037AD1CDA0B469D330B8D4A3E60E6D60AE9DB680FC2E1205E01EAC`

This is a narrow safety correction, not a grammar expansion. View/link code was
not changed.
