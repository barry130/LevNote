# Doc-block reference readiness proposal (queued stage)

Status: design-only. `native/docblock_reference_stage_plan.txt` is not active;
this proposal does not request implementation or tests yet.

## Current source facts

- A `doc-block` becomes `BlockDocBlock` only when it has the required `---`
  separator. A missing separator now remains visible as `BlockCode` with the
  complete original source and a warning.
- `Block.Range` is byte-oriented, while the native Markdown list stores row
  ranges as UTF-16 `text.Range`. A doc-block action must convert the captured
  byte range to UTF-16 before `Buffer.MapRangeSince`.
- `DocBlockRef.HeaderRange` is currently the empty range at the end of the
  opening fence; it does not cover the attribute section. It is not adequate
  as an activation identity.
- Header keys are lowercased and values are trimmed by the existing grammar.
  Unknown keys are preserved in `Extra`; payload bytes are untouched.
- The current map assignment silently lets a duplicate `id`, `kind`, `mode`,
  `src`, or `title` overwrite earlier values. There is no duplicate warning and
  no raw value range. IDs and titles are not unique source identity.
- The existing native button calls `Host.OpenReference(ref.Src)` directly from
  parsed data. It has no Buffer identity, version, enclosing range, Flush, or
  current-construct recheck.

## Smallest parser readiness seam

Add one small, additive parser result to an otherwise valid `BlockDocBlock`:

```go
type DocBlockIdentity struct {
    // Header is the UTF-8 byte range from after the opening fence line
    // through the end of the last header line, excluding the separator.
    Header SourceRange
    // Src is the existing recognized value: lowercased key, trimmed value.
    // It is not a claim of the raw src bytes.
    Src string
    // SrcPresent records whether a src header appeared.
    SrcPresent bool
    // DuplicateKeys contains lowercased keys that appeared more than once,
    // capped by the existing warning cap, sorted deterministically.
    DuplicateKeys []string
}
```

Place it on `DocBlockRef` (for example `Identity DocBlockIdentity`) without
removing current fields or changing serialization. This is metadata for safe
activation, not a new grammar: headers still lower-case, values still trim, and
unknown headers/payload remain unchanged.

Optionally add a syntax warning for duplicate recognized keys so malformed
headers are visible in warnings. That warning is parser-facing disclosure; it
does not change block kind.

No ID-based lookup or cross-document registry belongs in this stage. Two
blocks with the same ID/title must remain distinguishable by their source range
and exact parsed construct.

## Activation matrix

The native button may be enabled only for a valid `BlockDocBlock` meeting all
of these conditions:

- Header identity is available and the mapped header is still the exact
  current doc-block header.
- The block has no duplicate recognized keys reported by the parser.
- `src` is present and nonempty after the existing recognized-value grammar.
- The recognized `src` is at most 4096 UTF-8 bytes.
- The parsed construct at the mapped block range has the same kind, exact
  complete recognized `src`, and exact header range/content.
- The host provides `OpenReference`.
- The current source is not Partial and is within `markdown.MaxParseBytes`.

These existing constructs remain visible but cannot activate:

- Missing `src`, empty recognized `src`, or `src` over 4096 bytes.
- Duplicate `src`, `id`, `kind`, `mode`, or `title` headers.
- Missing separator (already `BlockCode` fallback).
- Unterminated block, Partial source, or over-limit source.
- Blocks whose header/body changed so range mapping fails.

The UI should show a specific bounded reason next to the action (for example
 duplicate headers, missing `src`, destination too long, source changed, or no
reference handler). It must never synthesize a target by trimming or shortening
an over-budget `src`. The action remains inaccessible rather than falling back
to another same-ID/same-title block.

No existing construct is broadened: `kind` and `mode` are not validated more
strictly by the parser in this stage, but the action should conservatively
require the already documented reference-compatible combination
`mode == "reference"` and a nonempty `src`. Requiring `kind` to a particular
value is not necessary for `Host.OpenReference`; the host remains the only
classifier/opener. If product wants a narrower button for diagram blocks only,
root must explicitly add that product rule.

## Native action contract

Capture at the actual pointer/key gesture:

- Buffer pointer.
- Buffer version.
- Stable preview row ID and block index from the immutable frame snapshot.
- Enclosing doc-block UTF-16 range.
- Header UTF-16 range.
- Recognized full `src` string, cloned only when within 4096 bytes.

Activation must then:

1. Call the existing Markdown `Flush()` first. A failed Flush retains the
   draft and stops navigation.
2. Reject Buffer replacement, foreign group, Partial source, oversize source,
   expired range mapping, split surrogate boundaries, or any mapped overlap.
3. Re-parse the current bounded complete source and find the `BlockDocBlock`
   whose current UTF-16 enclosing and header ranges equal the mapped ranges.
4. Require the exact complete current recognized `src` to equal the captured
   target, with no duplicate-key metadata. ID/title similarity is never enough.
5. Call `Host.OpenReference(target)` at most once and return immediately.
   The callback may synchronously Release the view, so the action must use no
   stale `v.doc` after the call.

Read-only documents can navigate through this path. Pure open creates no
history transaction; an active draft Flush may create its one existing normal
transaction. Ordinary pointer, Enter, and Space semantics should match the
existing native link controls; a newly rendered fixed-key button must not
activate from a stale gesture.

## Display and resource boundaries

- Keep the existing 32 KiB block preview guard. A large doc-block remains a
  placeholder with source guidance.
- Labels and tooltips use bounded, rune-safe strings. A target tooltip should
  show at most 512 bytes; an actionable full target may be retained only while
  within the 4096-byte activation limit.
- Do not retain a substring that keeps the backing source alive.
- Host.OpenReference continues to accept only its existing literal local
  references, anchors, local file URLs, and visible refusal policy. No network,
  embedded viewer, image loading, or URL normalization is added.

## Required focused acceptance once active

- Native pointer and keyboard activation, including stale fixed-key reuse.
- Same-ID and same-title distinct blocks.
- Unrelated insertion before the block maps safely; edits overlapping the
  block, deletion, Buffer replacement, Restore/history expiry, and Partial
  refuse visibly.
- Pending Markdown draft: successful Flush then navigation, and failed Flush
  retaining draft without navigation.
- Missing/empty/over-budget `src`, duplicate recognized headers, missing
  separator, readonly navigation, and no OpenReference handler.
- Exact recognized target passed once; pure open has no source/history change.
- Callback synchronously calls Release without invalidating the active List
  frame.

This proposal intentionally leaves `HeaderRange` compatibility unchanged until
the stage is active; the additive `Identity.Header` field avoids breaking
current callers while making exact verification possible.
