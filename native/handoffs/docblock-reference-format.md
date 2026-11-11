# Doc-block reference format checkpoint (2026-11-08)

Owner: GLM (`formats/markdown/types.go`, `formats/markdown/docblock.go`, focused
parser tests only)

## API seam

`markdown.DocBlockRef` now exposes:

```go
type DocBlockRef struct {
    Attrs DocBlockAttrs
    HeaderRange SourceRange
    Payload string
    FenceLength int
    PayloadRange SourceRange
    ReferenceProblem string
}
```

### HeaderRange

`HeaderRange` is the exact half-open UTF-8 byte interval in the original source:

- Starts at the first byte after the opening fence line’s line ending.
- Ends at the first byte of the `---` separator line.
- Includes all complete LF/CRLF endings belonging to header lines.
- Excludes the opening fence line, separator line, payload, and closing fence.

Example for LF:

```text
```doc-block\nid: a\nkind: text\n---\npayload\n```
```

The range slices exactly `id: a\nkind: text\n`.

For CRLF, the range ends immediately before the first byte of `---` and
therefore includes the header’s final `\r\n`.

### ReferenceProblem

`ReferenceProblem == ""` means the recognized header is structurally usable. A
normal inline block without `src` can also have an empty problem string and
still must not expose an open-reference action. Activation requires all of:
empty `ReferenceProblem`, explicit `mode: reference`, and the view’s own
destination/host/readonly checks. The string never promises host support, file
existence, navigation permission, readonly status, or a destination within the
4096-byte activation limit.

The parser sets a bounded explanatory problem, without embedding arbitrary
source values, for:

- unterminated fence;
- duplicate recognized headers: `id`, `kind`, `mode`, `src`, or `title`;
- missing or invalid `id` (the parser emits exactly one warning: the useful
  `doc-block-bad-id` syntax warning when an `id` is present but invalid, or the
  existing missing/invalid warning only when `id` is absent);
- missing `kind`;
- missing or unknown `mode`;
- missing or empty recognized `src` when mode is `reference`;
- any `src` header in explicit `mode: inline`, including an explicitly empty
  value (it is not opened as a reference).

Existing parsed values/source remain authoritative and displayed. The parser
does not add duplicate warnings for these conditions; view code should use
`ReferenceProblem` as the single bounded status seam. Unknown header keys
remain preserved in `Attrs.Extra`, and duplicate unknown keys do not block a
structurally valid reference.

Existing last-recognized-value behavior is preserved. A duplicate recognized
key sets `ReferenceProblem` even if both values are identical.

## Focused evidence

Commands:

```powershell
native/scripts/go.ps1 -GoArguments @('test','-count=1','./formats/markdown')
native/scripts/go.ps1 -GoArguments @('vet','./formats/markdown')
```

Results after the final duplicate-warning correction:

- `ok levnote.local/native/formats/markdown 0.651s`
- vet completed with no diagnostics.

## Behavior after the final correction

This seam intentionally changes one warning behavior: an invalid present `id`
now emits exactly one warning, the useful `doc-block-bad-id` warning, instead
of also emitting the generic missing/invalid ID warning. A missing `id`
continues to emit exactly one existing missing/invalid warning. No warning is
duplicated for the same ID issue. Raw source remains byte-identical.

No serialization, grammar, view, app, or host-routing behavior was changed.

