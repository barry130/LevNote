# Next reading increment review: bounded local image opening

ACTIVE task result. This is a source-grounded product/app plan only; no
implementation was started.

## Current behavior

- Markdown image syntax is parsed as a warning and skipped. `native/formats/markdown/inline.go:175-186`
  recognizes `![`, emits `unsupported-image`, advances through a complete
  destination when present, and produces no `InlineSpan`. `InlineKind` has no
  image value (`native/formats/markdown/types.go:31-45`).
- The preview recognizes only `InlineLink`. `native/modules/markdownview/links.go:44-65`
  counts only `InlineLink`; `links.go:118-133` constructs an interactive target
  only for that kind. `native/modules/markdownview/blocks.go:60-83` styles only
  link spans.
- Existing local links already use a safe application route:
  `View.activateLink` flushes and revalidates buffer/range/parser identity
  (`native/modules/markdownview/links.go:189-226`), then the host callback resolves
  the target (`native/ui/editor/contract.go:78-91`). The app rejects empty,
  remote, shell-like and drive-relative references and resolves relatives only
  against an absolute saved source path (`native/app/reference.go:31-80`).
  `openReference` checks source-tab ownership/activity, then calls `App.Open`
  (`native/app/reference.go:202-220`).
- `App.Open` canonicalizes paths and selects an existing tab case-insensitively
  (`native/app/app.go:625-668`). Image candidates go through the regular binary
  route (`native/app/app.go:679-683`), whose eight-byte probe is only a hint;
  `OpenBinary` supplies stable bytes and the decoder independently admits the
  real format (`native/app/binary.go:45-78`, `native/core/document/binary.go:67-87`).
- The image reader has one bounded lane and ownership/epoch checks
  (`native/app/binary.go:207-288`). `DecodeImage` is a neutral composition seam
  that opens no file and does no caching (`native/modules/modules.go:29-55`).
- Existing tests cover local `OpenReference` barriers and image opening, but no
  test starts from Markdown `![...]` syntax (`native/app/reference_test.go:71-127`,
  `native/app/binary_test.go:83-...`). Therefore today the user must rewrite or
  manually copy the destination into a link/open path; the reader cannot reach it
  from the Markdown surface.

## Recommendation

The strongest next increment is **explicit local image opening from Markdown
image syntax**, not inline image rendering.

User-visible behavior:

- A supported local Markdown image target is presented as an explicit,
  keyboard-focusable “image reference” control with its alt text and bounded
  target. Ctrl/Cmd+click or Tab+Enter routes it through the existing
  `OpenReference` path; the destination opens in the existing image tab.
- Unsupported/missing/oversized/non-local destinations remain text and are
  reported visibly. The source is never modified and no bytes are rewritten.
- Already-open destinations are selected, not duplicated.

This uses the accepted PNG/JPEG reader as the display surface and reuses its
existing lifecycle, async lane, snapshot ownership and decoder admission. It
adds no inline decoder, image cache, network I/O, plugin filesystem access or
automatic source rewrite. Inline display would introduce a second decoded-image
lifecycle inside Markdown, plus row sizing, virtualization and stale-row
concerns; it is a later stage after the simplest reading path works.

## Narrow implementation scope

DeepSeek owns the parser/export contract; this plan assumes a minimal new inline
span such as `InlineImage` carrying `Text` and `Href`, with exact source range.
The app/product stage should then own only:

- `native/modules/markdownview/links.go` and focused tests: count/render/activate
  image references as a separate bounded control, preserving current link budgets
  and no shared limit with ordinary links unless source review requires one.
- `native/modules/markdownview/blocks.go`: neutral non-link styling for fallback
  text if needed; no visual image representation in this stage.
- `native/app/reference_test.go` or a focused app test: Markdown source saved in
  a directory, local PNG/JPEG targets, existing-tab selection, unsaved-source
  rejection and remote/oversized/missing-target reporting.

No edits are proposed to imageview, raster, document, session, plugin manager,
app open/barrier code, or product docs in this stage.

## Invariants and bounds

- Image references use the existing 4096-byte destination bound and visible
  “not opened” reporting; do not create an unbounded URL field.
- Alt text and target are bounded in UI labels. No closure may retain a source
  slice; clone the bounded destination exactly as links do.
- Activation must preserve the existing flush → same-buffer → range mapping →
  reparse → exact syntax identity sequence. A stale draft or edit rejects without
  opening.
- App-side resolution remains the sole filesystem authority. The Markdown view
  only passes the captured literal destination to `OpenReference`.
- Existing image limits remain authoritative: regular file, 16 MiB stable source,
  supported PNG/JPEG admission, one bounded image lane, active-owner/generation
  publication checks. No inline bitmap is held by Markdown rows.
- No network, remote UNC, system shell, current-working-directory resolution or
  automatic path rewriting is introduced.

## Focused validation

- Parser: exact image span/range/alt/destination; nested/escaped/malformed
  syntax; export warning behavior and no source mutation.
- Markdown view: image control rendering, pointer/keyboard activation, stale
  source, edit/reparse rejection, bounded target, fallback text, and existing
  link behavior unchanged.
- App: saved relative image opens an image tab; already-open image is selected
  case-insensitively; unsaved source, remote URL, oversized target and missing
  file report without opening or modifying the source.
- Keep validation to focused package/app tests; no package, product launch or
  broad suite is part of this plan.

## Alternative considered

The other everyday-reading candidate is source-navigation/file-workflow
improvement: an explicit “copy/open resolved path” affordance for selected
references. It is smaller, but it mainly serves editing and does not improve the
reading surface. It also adds a second way to interpret references. Local image
opening gives immediate reading value, reuses all mature barriers and leaves
inline rendering for a later bounded stage.

Remaining unknowns for implementation ownership:

- Whether DeepSeek’s parser/export review chooses `InlineImage`, a reference-only
  span, or a narrower metadata shape. The app stage must not assume unsupported
  image warnings disappear until that contract lands.
- Whether image and link controls should share the current dense-row budget or
  have a separate small image-control budget. Either is bounded; the choice should
  follow actual row-building cost and existing accessibility behavior.
- Exact warning/message wording after the parser/export review.
