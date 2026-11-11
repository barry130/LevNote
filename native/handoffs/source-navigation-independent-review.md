# Native source navigation: independent final review

2026-11-09. Final read-only review of the frozen source, tests and tester
evidence. No test, build, vet or launch was run by this reviewer.

## Disposition

No material defect remains within the bounded Ctrl+G/source-navigation scope.
The current hashes match `source-navigation-result.md`, the previous findings
are resolved in the current source/tests, and both final 760x480 captures show
the input, error, compact actions, complete help text and status row without
crossing the main-pane bounds.

## Verified corrections

- Panel generations are scoped through the input key, ancestor observer,
  submit/cancel callbacks and button shortcuts (`source_navigation.go:106-127`,
  `154-210`, `223-278`). A stale callback fails the generation check, and a
  reopened input cannot reuse the closed field's keyed TextInput state.
- Escape works while either button owns keyboard focus through scoped button
  shortcuts (`source_navigation.go:267`, `275`). The focused-button keyboard
  test covers both buttons (`source_navigation_test.go:150-188`).
- Active query composition suppresses Escape/Enter submission and pointer
  submission while keeping focus in the field; composition cancel, commit and
  the complete post-commit query are tested (`source_navigation_test.go:236-323`).
- Submission checks the captured tab, document, editor and both buffer
  identities, Flushes once, rechecks them, validates final-buffer coordinates,
  then persists source/split/preview navigation (`source_navigation.go:130-199`).
- The real Markdown view is wrapped rather than replaced, and source navigation
  invokes that actual view's `Flush` exactly once; a deterministic Flusher
  separately proves final coordinates are resolved after source-changing
  synchronization (`source_navigation_test.go:325-362`, `503-539`).
- Leading zeroes are an explicit, documented rejection. The help says that
  omitting `:column` defaults to 1 and states the 32-UTF-16-unit limit
  (`source_navigation.go:219-221`).
- The invalid-syntax test now expects rejection without Flush and preserves the
  raw query and local panel state (`source_navigation_test.go:466-481`).

The updated result records the final focused pass, captures, one combined
affected suite and scoped vet under the exact source/test hashes, in that order.
The two capture files were inspected directly:

- `source-navigation-evidence/source-navigation-error-760x480-light.png`
- `source-navigation-evidence/source-navigation-error-760x480-dark.png`

Remaining limits are correctly disclosed: synthetic tester IME coverage is not
physical OS candidate-window evidence, tester captures are not physical UI
automation, and no process-resource claim is made. No package build, product-doc
change, `.11` artifact change or additional review framework is justified.
