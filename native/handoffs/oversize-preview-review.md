# Oversized-preview boundary review — current code

Reviewed baseline: **2026-11-08 15:00 local file timestamps**. This replaces
the earlier review of the pre-14:53 source; those findings are obsolete.

Verdict: **ready for Astra's focused test result and integration review**. I
found no reachable defect in the current guarded paths. I did not run tests;
this is an independent static review of the current implementation and tests.

## Current-code findings

### No defect: Markdown whole-source paths are guarded

- `checkPreviewSize` rejects on `Buffer.LenBytes()` before materialization and
  caches the refusal by buffer identity and version:
  `native/modules/markdownview/preview_limit.go:16`.
- `refresh` checks before `Buffer.String()`:
  `native/modules/markdownview/view.go:72`, parse at `:81`.
- `Export` invalidates before `Flush`, then checks again before its temporary
  parse: `native/modules/markdownview/view.go:131`, `:138`, parse at `:143`.
- `begin` checks before `Flush` and again before prefix/source mapping:
  `native/modules/markdownview/draft.go:113`, `:117`.
- A pending draft is retained, while obsolete rows, parsed source and outline
  cache are dropped. The fixed draft panel remains visible with guidance.
- `dropPreviewCache` owns no source text:
  `native/modules/markdownview/preview_limit.go:35`.

The first `Export` check intentionally runs before `Flush`. If a conflict stops
the flush, export fails without output while the draft remains recoverable; if a
safe range flush succeeds, the second check decides from the authoritative
buffer. That order is correct.

### No defect: graph whole-source paths are guarded

- `previewSizeError` checks `LenBytes()` against the exported parser defaults:
  `native/modules/graphview/preview_limit.go:15`.
- `ensureSynced` clears parsed/render state, caches the refusal by buffer
  identity/version, and returns before `Buffer.String()`:
  `native/modules/graphview/sync.go:30-39`, copy at `:45`.
- Draft `Flush` checks before copying and parsing:
  `native/modules/graphview/draft.go:144`, `:180`, copy at `:185`.
- Mutation helpers do **not** bypass the guard. Both call `ensureSynced`, then
  `requireWritable()`, which calls `writable()`, which calls
  `previewSizeError()`:
  - mindmap helper: `native/modules/graphview/edit.go:78-89`
  - drawio helper: `native/modules/graphview/edit.go:302-313`
  - writable check: `native/modules/graphview/sync.go:248-256`
  - requireWritable: `native/modules/graphview/input.go:245`
  Thus an oversize document reaches an early `ErrLimitExceeded` before either
  helper's `Buffer.String()`.
- `Export` propagates the cached sync error before SVG generation:
  `native/modules/graphview/adapter.go:203-210`.
- The mindmap outline uses the exported default, not a duplicate local value:
  `native/modules/graphview/outline.go:39`.
- The outline navigation's fallback parse is unreachable for an oversize
  buffer because `ResolveOutlineItem` first returns the outline size error.

The earlier predicted mutation bypass was based on stale source and is withdrawn.

### Constants are single-source

- Markdown: `markdown.MaxParseBytes`
  (`native/formats/markdown/outline.go:12`).
- Mindmap: `mindmap.DefaultMaxBytes`
  (`native/formats/mindmap/parse.go:15`), used by normalization and views.
- Drawio: `drawio.DefaultMaxBytes`
  (`native/formats/drawio/parse.go:19`), used by normalization and views.

No changed parser budget was found.

## Test coverage assessment

The focused tests appear to cover the requested acceptance, with fixtures
constructed outside measured regions:

- Markdown:
  `native/modules/markdownview/preview_limit_test.go:30`
  measures first refusal and 100 cached warning/export refusals, checks no
  output/source/selection/history mutation, same-version buffer replacement,
  shrink and undo recovery.
  `:98` measures stale `begin` refusal and checks no large prefix/source copy.
  `:116` covers safe-range flush versus conflict, draft retention, export
  refusal, release and recovery.
  `:173` covers shrink while an independent composition remains open.
- Graph:
  `native/modules/graphview/preview_limit_test.go:43`
  measures both format bounds, repeated warnings/export refusals, cache
  clearing, state preservation, same-version replacement and shrink.
  `:103` proves valid cached graph state is cleared, mutations are blocked
  without invoking their callbacks or copying source, and undo recovers.
  `:145` proves a pending composition survives guarded flush, release and
  recovery, and export emits no output.

These are meaningful allocation and lifecycle assertions, not merely parser
limit tests. Astra should publish the actual focused command/result; this review
does not treat test execution as already independently verified.

No product code was changed by this review.
