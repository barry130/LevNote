# Markdown image reference contract review

Status: READ-ONLY REVIEW, 2026-11-10. No product source, tests, builds,
formatting, app runs, or product operations were changed or executed. This
handoff is the only file written. `.25` packaging remains active.

## Recommendation

Add one distinct parsed span kind for an explicit image reference, but keep
pixels and interpretation outside the format:

```go
const InlineImage InlineKind = "image"
```

Reuse the existing `InlineSpan` fields without adding a second destination
model:

- `Range`: exact original half-open byte range from `!` through the closing
  `)`;
- `Text`: the raw label returned by the existing bracket scanner, with escapes
  retained;
- `Href`: the exact raw bytes between the destination parentheses, with no
  trimming, unescaping, percent decoding, title removal, or path normalization.

The view should expose this span as a native explicit reference action and pass
the raw `Href` to the existing `Host.OpenReference`. It must not read image
bytes, decode pixels, infer an image type, open the source itself, or emit an
HTML image. The app's existing reference resolver remains authoritative over
local/remote/unsafe targets and relative-path resolution.

The format and view changes must land as one reviewed contract. Format-only
merge is unsafe: current `richText` enters native-child mode only when it finds
an `InlineLink`, while the non-native span renderer otherwise uses
`InlineSpan.Text`. An unrecognized `InlineImage` would therefore lose the
visible Markdown delimiters and expose no action.

## Existing source facts

- `native/formats/markdown/types.go:31-45` defines four inline kinds and the
  `InlineSpan{Kind, Range, Text, Href}` shape. It already has the exact fields
  needed for a bounded image reference.
- `native/formats/markdown/inline.go:43-205` parses inline syntax. At
  `inline.go:175-191`, `![` currently reports `unsupported-image`, uses the
  first `]`, and emits no span.
- `readBracketLabel` at `inline.go:375-394` handles escaped bytes and balanced
  nested brackets. Unlike the current image branch, it does not stop at the
  first `]`.
- `findLinkEnd` at `inline.go:396-407` skips an escaped byte and returns the
  first unescaped `(` or `)`. Therefore existing inline links do not support
  balanced unescaped destination parentheses, angle destinations, titles, or
  reference-style syntax. The URL is raw; it is not decoded.
- Inline work is bounded by a 64 KiB suffix, 1 MiB document scan, 4096 spans
  per block, and 16384 spans per document (`inline.go:19-23, 278-350`).
  Warnings are capped at 256 (`parser.go:17-18`, `inline.go:365-372`).
- `parser.go:278-316` permits paragraph source to contain multiple physical
  lines. New image recognition therefore needs an explicit single-line rule;
  relying only on bracket/parenthesis search could cross a newline.
- `parser.go:208-214` creates `BlockTable` descriptors without parsing cell
  inline spans. `native/formats/markdown/table.go:43-124` preserves table cell
  byte ranges instead.
- `native/formats/markdown/export.go:139-194` renders known inline formatting
  but sends unknown inline kinds through the exact original range. A new
  `InlineImage` case should explicitly escape and emit that same original range,
  never `span.Href` or `span.Text`.
- `export.go:328-446` renders tables from exact cell text and HTML-escapes it.
  An image-looking table cell therefore remains literal.
- `native/ui/editor/contract.go:84-100` already provides
  `OpenReference func(string)`. No new host API is needed.
- `native/app/reference.go:31-145` resolves references as raw strings. Relative
  paths are joined literally to the saved source directory; no percent
  decoding is applied. Remote URLs, UNC paths, drive-relative paths and unsafe
  schemes are rejected. `reference.go:175` wires `OpenReference`, and
  `reference.go:202-220` routes accepted local targets through the normal
  app-open barrier.

## Supported syntax and fallback

Accept only this single-line, inline subset:

```text
![label](destination)
```

The opening `!` must be unescaped. The label uses the existing
`readBracketLabel` scanner. The destination must:

- be non-empty and at most 4096 bytes;
- contain no CR or LF;
- end at the first unescaped `)`;
- contain no unescaped `(` before that close;
- contain no ASCII space, tab, `<`, or `>`, so title and angle destinations
  cannot be misread as literal filenames;
- reject `\(` and `\)` escape forms because there is no format unescape step
  and passing their backslashes into a filesystem resolver would change their
  meaning. Other raw backslashes remain valid Windows path bytes.

Do not parse or classify URL schemes in the format. For example,
`https://...` can remain a syntactically recognized image reference, but the
existing app resolver rejects it without a network request. Likewise, `%20`
stays the three literal bytes `%20`; this increment must not decode it.

Fallback behavior should be explicit:

| Input case | Contract |
| --- | --- |
| `![图](dir/a.png)` | Emit one `InlineImage`; exact full `Range`, raw `Text`, raw `Href`; no `unsupported-image` warning. |
| Escaped or nested label brackets such as `![a\\[b\\]c](x.png)` | Accept through the existing label scanner; retain the source spelling in `Text`. |
| `\\![a](x.png)` | Do not emit `InlineImage`. This is an escaped exclamation, although the existing link scanner may still recognize the remaining `[a](x.png)`. |
| Unbalanced/nested unescaped destination parentheses | No image span; raw source and `unsupported-image` warning. Do not enable balanced-parenthesis parsing. |
| Escaped destination parentheses | No image span; raw source and warning. Do not add an unescape layer. |
| Angle destination or title form | No image span; raw source and warning. |
| Reference-style `![a][id]` or collapsed/shortcut form | No image span; raw source and warning. |
| Multiline label or destination | No image span; raw source and warning. This applies even though a paragraph block may contain newlines. |
| Malformed `![...` or missing closing delimiter | No image span; keep the full raw source visible and warn on a safely bounded exact range. |
| Image inside code span or fenced code | No image span; code parsing consumes/owns those bytes. |
| Image inside table cell | No image span; table cell displays/exports literal bytes. |
| Image inside bold/italic/code or another existing outer inline span | No independently actionable image; the outer span remains authoritative and displays raw image syntax. |
| Nested image-in-link `[![a](img.png)](target.md)` | Do not emit `InlineImage`. To keep the rejected image syntax raw and non-actionable, do not emit the outer `InlineLink` for a label containing an unescaped image opener. Emit a bounded `unsupported-image` warning. This is a deliberate narrowing of the outer link only for this unsupported composition, not a link-parser expansion. |

Accepted image spans must count against the existing per-block/document span and
scan limits. All candidate searches should reuse `readBracketLabel`,
`findLinkEnd`, and their budget-aware helpers; do not add a second unbounded or
quadratic scanner.

## Exact action identity

The view should reuse the discipline already implemented for Markdown links,
but extend its target identity with the inline kind:

```go
type referenceTarget struct {
    buffer      *text.Buffer
    version     uint64
    rowID       uint64
    enclosing   text.Range // visible row/list-item span
    syntax      text.Range // exact ![label](destination) span
    blockKind   markdown.BlockKind
    inlineKind  markdown.InlineKind
    item        bool
    href        string
}
```

At activation:

1. Flush any pending same-view edit and preserve existing conflict/draft
   behavior.
2. Map both the enclosing and syntax ranges through the buffer version.
3. Reparse if the view snapshot is stale, as links already do.
4. Find only a span at the same exact syntax start/end with
   `inlineKind == markdown.InlineImage` and byte-equal raw `Href`.
5. If the source has become an `InlineLink`, another inline kind, or a different
   image target, report stale and activate nothing.
6. Clone the bounded raw destination before the action, then call
   `Host.OpenReference(href)`. Do not retain a block, document, raw-header slice,
   or view object across the host call.

This kind check is mandatory. Matching only endpoints and `Href` is not enough:
an image syntax edited into a link, or a same-location duplicate introduced by
an edit, must never inherit the captured gesture.

## View integration requirements

The Markdown-view owner should make image actionability visible before enabling
the parsed kind:

- update row capability detection so image-only rows take the native-child
  path instead of `spans()`;
- count image controls under the existing interactive-span safety budget;
- render an explicit image-reference control with distinct accessibility text
  and a tooltip that exposes a bounded raw target;
- keep empty alt text discoverable;
- disable the action visibly when the host is absent or the destination exceeds
  4096 bytes;
- preserve keyboard, pointer-cancel, scroll-cancel, focus, readonly, and
  synchronous-release behavior already covered for links;
- do not derive or display image dimensions and do not request `BeginImage` from
  a Markdown view.

The app should retain the existing `OpenReference` path. Opening a local image
through that path can use the already accepted binary image tab; the Markdown
view neither needs a path nor receives decoded pixels.

## Ownership split

- **Format owner:** add `InlineImage`; implement only the accepted scanner
  branch and exact raw range; preserve fallback warnings; make HTML export
  explicitly emit original image syntax; add parser/export tests.
- **Markdown-view owner:** recognize image spans as native reference controls;
  capture and validate image kind plus exact ranges; preserve all existing
  gesture/draft/readonly semantics; add view and app-action tests.
- **App host owner:** no new API or runtime behavior. Add resolver/action tests
  only if the view integration needs them.
- **GLM product plan:** remains the owner of the visible reading/app-side
  experience in `native-next-reading-review.md`; this review intentionally does
  not prescribe its UI layout or wording.

Do not edit app routing, binary/image decoding, core modules, vendor code, or
export engines for this increment.

## Highest-risk cases

1. Format-only merge makes an image span disappear into alt text with no
   action because `linkCount` and `spans()` do not know `InlineImage`.
2. A range-only stale check can activate after image-to-link mutation; inline
   kind must be part of the captured identity and reparse check.
3. Reusing the current first-`]` image scan is wrong for escaped or nested label
   brackets, while broadening parentheses/titles would disagree with the raw
   resolver. Keep the narrow matrix above.
4. A parser that searches past newlines can turn malformed multiline text into
   an actionable target.
5. Export must not emit `<img>` or an `src` value. Original Markdown bytes must
   survive in HTML as escaped text.
6. Table and code-looking syntax must stay literal because those owners do not
   create inline image descriptors.
7. Dense image rows must use the existing work/control fallbacks rather than
   building one native child per unbounded span.

## Focused expected outcomes for the later implementation

- Parser: exact source/range/text/href for plain, nested-label, escaped-label,
  heading, quote, paragraph, and list-item images; offsets include the base
  source position.
- Parser fallback: no `InlineImage` for escaped exclamation, unescaped/escaped
  destination parentheses beyond the subset, angle/title, reference-style,
  multiline, malformed, code, table, and nested-image-in-link fixtures; raw
  source remains byte-identical and warnings use exact bounded ranges.
- Export: accepted and rejected image syntax appears as literal escaped
  Markdown in HTML; there is no `<img`, `src=`, pixel/type inference, or remote
  fetch. Table cells and code blocks remain literal.
- View: an image action passes the exact raw local path to `OpenReference`;
  missing host and over-4096-byte target are visibly unavailable; readonly mode
  does not edit source; pointer/keyboard cancellation opens nothing.
- Stale identity: prefix edits that map the same image may still activate;
  duplicate-before, label/target edits, delete/restore, image-to-link mutation,
  foreign buffer/group, partial parse, and oversize source do not.
- Host: a raw relative image path resolves against the saved document path; a
  remote/UNC/unsafe target is rejected with no network access; `%20` is not
  decoded for a relative local path.
- Budgets: many candidates stop through the existing scan/span/control limits
  while complete raw text remains visible and the view settles without
  unbounded native controls.

