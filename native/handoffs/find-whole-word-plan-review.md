# Whole-word Find plan review

2026-11-10. Read-only review during the `.16` source freeze. I inspected the
staged plan, native coordination rules, current app Find/search wiring, the
shared core search implementation and current cache/layout tests. I edited no
product source, tests, docs or captures and ran no tests, builds or launches.

## Verdict

The proposed slice is correct and ready for activation after `.16` is accepted.
The missing capability is native runtime/UI wiring, not a search algorithm.
Keep this as one bounded implementation pass; do not change `core/search`,
replacement semantics, the cache design or persisted session state.

## Confirmed current behavior

- `native/core/search/find.go` already defines `Options.WholeWord` and applies
  it to literal and regex matches. `whole()` treats
  `unicode.IsLetter`, `IsNumber`, `IsMark` and `_` as word characters.
- `TestUnicodeCaseAndWholeWords` already proves case-insensitive `CAT`/`cat`,
  rejection inside `category`, `_cat` and `cat_`, acceptance of standalone
  `猫`, rejection inside `猫咪`, and rejection of `e` before a combining mark.
- `native/app/app.go` stores only `CaseSensitive` and `Regex`.
- `native/app/state.go:findOptions()` copies only Query, Regex and
  CaseSensitive, so document navigation and replacement cannot currently pass
  whole-word matching.
- `native/app/workspace.go` passes that same `findOptions()` result into
  `search.Workspace`; fixing the shared method wires workspace search too.
- Document Replace and Replace All also call `findOptions()` directly, so they
  need no whole-word-specific branch.
- `findNavigationCache` stores the complete `search.Options`, and both
  `expireFindCache` and `searchMatchesCurrent` compare it. Changing `WholeWord`
  therefore causes a cold scan on the next Find action. Do not add another
  invalidation path.
- Native Find currently renders Checkbox controls for case and regex but has
  no whole-word control.

## Smallest implementation

Production changes are limited to two app files:

1. `native/app/app.go`: add one runtime `WholeWord bool` beside
   `CaseSensitive` and `Regex`.
2. `native/app/state.go`: add `WholeWord: a.WholeWord` to `findOptions()`.
3. In the existing first Find row in `state.go`, add one short checkbox beside
   case and regex:

   ```go
   ui.Checkbox(c, &a.WholeWord, "全词").KeepFocus().
       OnShortcut(0, ui.KeyEscape, cancel)
   ```

This mirrors the established control lifecycle. A checkbox has no text
composition state of its own, so no generation/remount work or new focus
framework is needed. Keep the existing query/replacement composition handling
unchanged. If a description is added, state only that it matches whole Unicode
word boundaries and does not perform Chinese word segmentation.

`WholeWord` remains runtime-only like case and regex. Do not alter session
schema, per-tab preferences, `core/search`, workspace traversal, replacement
algorithms, or cache structure.

## Test and evidence scope

Use a small feature test file, `native/app/find_whole_word_test.go`, plus the
existing cache/layout tests:

- Extend `TestFindCacheReusesColdOffsetsAndTracksSourceVersion` in
  `native/app/search_navigation_test.go`: warm the cache, toggle `WholeWord`,
  run Find, and prove one additional cold scan and the filtered match set.
- In the feature test, use `ui.Tester` to click the actual `全词` checkbox and
  exercise next, previous and wrap. Cover `cat`, `category`, `_cat`, `cat_`,
  a combining-mark case, and `猫` versus `猫咪`. Include one regex + case
  interaction using the same checkbox without changing the query text.
- In the feature test, cover Replace Current and Replace All through the shared
  option. Assert Replace All is one grouped undo. Reuse one readonly and one
  failed-Flush fixture to prove the option does not bypass existing lifecycle
  guards; failed Flush must leave text, version and selection unchanged.
- Add one local workspace fixture through the app's actual workspace-search
  entry point, with both an in-boundary and an in-identifier candidate, proving
  the same checkbox reaches `search.Workspace`.
- In `native/app/findbar_layout_test.go`, add `"全词"` to
  `findbarControlLabels`, action/tab-order coverage, and keyboard Escape after
  focusing the checkbox. At `760x480`, retain bounds, overlap, editor and
  status assertions. Because visible markup changes, refresh the existing light
  and dark `760x480` screenshots and record their hashes in the result.
- Run focused failures first, focused regressions after correction, then one
  final affected `app` + `core/search` test/vet pass using the native harness.
  Preserve raw commands and output. No build is part of source acceptance.

Existing core whole-word tests need no duplicate expansion unless implementation
uncovers a concrete uncovered boundary. Existing focus, composition, stale
callback, rich-Flush and workspace suites should be reused selectively rather
than retested wholesale.

## User-visible semantic limit

The option means the existing Unicode boundary rule above. It is not Chinese
word segmentation, identifier-aware matching or syntax parsing. Regex query
text and replacement expansion remain unchanged; filtering only decides which
compiled matches satisfy the boundary.
