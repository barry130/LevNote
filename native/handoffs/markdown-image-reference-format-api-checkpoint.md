# Markdown image-reference format API checkpoint

2026-11-10. Additive API is now present for Astra to consume:

```go
const InlineImage InlineKind = "image"
```

`InlineSpan` is unchanged. An accepted reference carries its exact source byte
`Range`, the raw bracket label in `Text`, and the exact raw destination in
`Href`. Accepted references will emit the stable warning code
`image-reference` because inline image rendering remains unsupported.
Unsupported or oversized (over 4096 destination bytes) forms emit no
`InlineImage`; their complete source remains visible through the ordinary raw
fallback. Ordinary outer `InlineLink` parsing, including an image-looking link
label, remains unchanged. A complete rejected image envelope is consumed for
recovery without activating nested image-looking fragments, including when a
provable matching envelope close crosses LF, CRLF, or CR line endings; parsing
resumes after that envelope. An unmatched label is bounded to its physical
line, so recovery resumes on the next line.

The format scanner/export implementation and focused tests are complete. The
frozen source and exact identities are in
`markdown-image-reference-format-result.md`.
