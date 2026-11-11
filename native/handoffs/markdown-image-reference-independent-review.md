# Markdown image-reference independent lifecycle review

2026-11-10. Read-only review of the frozen image-reference format/view/app
change. No source, test, build, package, runtime or product-document changes
were made.

## Result

No additional reachable lifecycle defect found.

The review covered `modules/markdownview/links.go`, `blocks.go`, `view.go`,
`rows.go`, `draft.go`, the focused image-reference and existing link tests, and
the app routing through `app/reference.go` and the existing `Open` path. The
final affected app/view test and vet result was read as inherited evidence and
was not repeated.

## Lifecycle trace

- A retained gesture owns only buffer/version/release/row descriptors, exact
  enclosing and syntax ranges, block/list/item kind, inline kind, and a bounded
  cloned destination. It does not retain a parser document or a slice of source.
- Native input cancels the captured gesture on a new pointer press, scroll,
  zoom, canceled release, or non-Enter key movement. Image auto-repeat Enter is
  consumed, and rapid pointer continuation is suppressed before it can become a
  second activation. The pressed preview is held only through release and then
  settled with one invalidation.
- Activation checks release generation and inline kind, flushes a pending draft,
  rechecks the same current buffer, maps both enclosing and syntax ranges from
  the captured version, reparses when the preview is stale, and requires the
  exact block/item shape, syntax range, inline kind, and cloned destination
  before calling `Host.OpenReference`. The host capability check happens after
  source validation, and opening is the final operation.
- Synchronous release from `OpenReference` is safe: activation state is cleared
  before the call, `OpenReference` is last, and `View.Release` replaces list
  ownership rather than mutating the state used by the active build.
- Ordinary links keep the existing route. Image-only branches are keyed to
  `InlineImage`; an outer ordinary link containing raw image syntax remains an
  outer `InlineLink`. Dense and styled fallbacks preserve the complete raw image
  syntax, and links plus images share the stated 16-control, 64-span and
  parser/preview limits.
- App routing remains the existing `OpenReference`/`Open` barrier, including
  current/open-tab checks and the established local-reference safety rules. No
  filesystem access, decoder, bitmap cache, network path, or new host API was
  added to the format/view path.

This is a bounded source review, not a whole-application audit. The inherited
validation remains synthetic and makes no physical OS input/IME, accessibility,
driver, or resource claim.
