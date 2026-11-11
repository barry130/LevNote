# Markdown image-reference format result

2026-11-10. Format/source result is frozen after the focused corrections and
the single final affected test/vet pass. No product build, package, host, app,
view, core, vendor, or docs implementation was run or changed by this task.

## Shared API

`formats/markdown` now exports:

```go
const InlineImage InlineKind = "image"
```

`InlineSpan` is unchanged. An accepted reference carries the exact original
byte `Range`, the raw bracket label in `Text`, and the exact raw destination in
`Href`. The native view can reuse the existing neutral Host reference contract
and revalidate kind, source range, buffer/version, row, and target before any
open.

## Accepted behavior

The actionable subset is one-line, unescaped `![label](destination)`.
`label` uses the existing balanced/escaped bracket helper. `destination` is
1..4096 valid UTF-8 bytes and rejects Unicode controls, ASCII whitespace,
`<`, `>`, and parentheses. Escaped destination parentheses are rejected
because this format layer deliberately does not Markdown-unescape them.

Plain Windows backslashes elsewhere, literal `%`, and URL-shaped strings remain
byte-for-byte values in `Href`; there is no trimming, percent decoding, title
extraction, path rewriting, or URL authority decision in the format layer.
The host remains responsible for refusing network, UNC, and other unsafe
targets.

An accepted reference emits exactly one `image-reference` warning covering the
exact `InlineImage` range:

> image syntax is preserved as an explicit picture reference; inline images
> are not rendered

The syntax remains visible instead of being rendered as inline pixels.

Rejected, malformed, multiline, reference-style, titled, angle-delimited,
nested-parenthesis, and over-4096-byte destinations emit no `InlineImage`.
They retain complete source and emit `unsupported-image` with the complete
candidate range. The 4096-byte boundary is covered with 4096 accepted and 4097
rejected. Existing parser warning, span, and scan budgets still apply.

## Rejected-envelope recovery

A rejected parenthesized image candidate normally consumes the matching outer
envelope, including nested parentheses, for recovery only. It does not make a
balanced nested destination actionable and does not leave an inner
image-looking fragment available as a new `InlineImage`. A provable matching
close may be found across LF, CRLF, or CR line endings because the rejected
outer candidate still owns that fragment; accepted destinations themselves
remain single-line. Parsing resumes after the completed envelope, so an
unrelated ordinary link that follows it still parses.

If the outer label cannot be proven complete, recovery is bounded to the
physical line and parsing resumes on the following line. This intentionally
treats the remaining malformed label text on that line as part of the rejected
outer candidate; it does not swallow later lines. Exhausting the existing
64-KiB inline scan suffix or 1-MiB scan budget marks the document partial and
emits `scan-budget`; no nested image action is emitted.

Ordinary `InlineLink` behavior is unchanged. In particular, an outer link with
raw image syntax in its label remains an outer link and gains no independent
image action. Escaped `!` keeps existing link behavior. Code spans, code
fences, tables, bold, and italic retain their existing nonrecursive ownership.

## HTML and worker boundary

`InlineImage` exports the escaped original syntax. It never emits `<img>` or a
`src` resource, in either `ExportHTMLChecked` or bounded byte export.

The focused in-process Markdown worker test verifies the same literal HTML,
exact `image-reference` and `unsupported-image` codes/ranges crossing the
worker response boundary, absence of `<img>`/`src`, and unchanged source bytes.
It does not launch an OS worker or add runtime behavior.

## Verification and identities

All runs used `native/scripts/go.ps1`. Earlier focused runs 01-06, corrected
rejected-envelope runs 07-09, and their final affected runs 10-11 exited 0.
The root review found one multiline-envelope gap, which was corrected with the
superseding final pass below. Prior evidence was preserved:

| Run | Command | Result |
|---|---|---|
| 13 | `gofmt` for the multiline-envelope correction files | exit 0 |
| 14 | `go test ./formats/markdown ./cmd/levnote-markdown-export` | exit 0; both PASS |
| 15 | `go vet ./formats/markdown ./cmd/levnote-markdown-export` | exit 0 |
| 16 | frozen source identity check | exit 0; six matches |

Exact command text, stdout, stderr, and exit files are under
`native/handoffs/evidence/markdown-image-reference-format-20261110/`.
No implementation-test failure occurred. The new rejected-envelope regression
covers nested destination, unmatched destination, unmatched label,
reference-label nesting, a complete rejected envelope followed by an ordinary
link, an unmatched outer candidate followed by a next-line ordinary link,
multiline rejected destinations terminated by LF, CRLF, and CR with an exact
warning range and following ordinary link, and the bounded-work exhaustion
case.

Frozen source identities (byte length and lowercase SHA-256):

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/formats/markdown/types.go` | 3347 | `0fac32eca5cd9e8fa49565610700f57afaf848e16ec05cce6647fae9423e1029` |
| `native/formats/markdown/inline.go` | 13928 | `aa793a782d35baab4f27a62ab000bd4b77e4718c36c68c8721167efbfdb7f070` |
| `native/formats/markdown/export.go` | 12453 | `f9918914b182ec19dcac47bb7533f6a44614d8ff9e84d530c4e012f4e7ea6db6` |
| `native/formats/markdown/image_test.go` | 11686 | `ce11258b0a1e71bb2b7ea15ebdea26d298558ed63a9ff43bd492578e8b6a90f5` |
| `native/formats/markdown/markdown_test.go` | 12783 | `b31ba61545754d39fc80961c3803dbea891792c2d4fbb7c3efa15936b71684c9` |
| `native/cmd/levnote-markdown-export/image_reference_test.go` | 1700 | `8dda439d04322cdcaf742d3be37cc7d8855dd05c066cd59dc1076209d0d3d9cb` |

These identities are also saved as
`16-multiline-envelope-source-identities.stdout.txt`; the earlier run-12
identity file remains as preserved historical evidence. The format source is
frozen for the separate coordinated app/view integration check.
