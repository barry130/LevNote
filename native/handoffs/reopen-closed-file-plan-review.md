# Reopen-closed-file plan review

2026-11-10. Read-only review of `reopen-closed-file-root-decision.md` and
`reopen-closed-file-next-plan.md` against the current app close/open/binary/menu
seams. No runtime, test, build, product-doc, or execution changes were made.

## Result

No blocking finding beyond the risks already specified by the plan. The chosen
small prerequisite is source-grounded and sufficient for the first path-only
slice.

## Source seams and confirmed risks

- Close recording must capture and revalidate the exact `*Tab`. Current
  `CloseTab(index)` takes the tab at that index, then `prepareClose` can flush,
  show the close dialog, and perform SaveAs (`app/app.go` around 932–1084). A
  naïve post-dialog `removeTab(index)` could remove a different tab after a
  nested tab change, while `commitClose` only knows that its `remove` callback
  returned. The plan's requirement to check the same target/slot after dialog
  reentry and record only after the exact tab was actually removed is therefore
  concrete, not speculative.
- Close candidacy should be captured after successful `prepareClose`, because
  successful SaveAs updates `t.Doc.Path`; recording before the dialog would
  store the old path. Failed flush/save, cancel, persistence failure, or an
  aborted remove must retain no entry.
- `openRequest` currently has only `path` and `reveal`
  (`app/app.go` around 208). `openRequests`' completed channel only means the
  queue finished (`app/app.go` around 649), and an existing-tab `Select` can
  fail while `existing` is still assigned. A private numeric request/entry
  identity and a terminal receipt at actual `Select`/new-tab adoption are
  required.
- The binary lane confirms the same distinction. `openBinaryRequest` returns
  `true` after a start error, a failed completion, or refused adoption; only
  dispatch/shutdown paths return `false` (`app/binary_actions.go`). Receipts
  must be attached at the actual duplicate `Select` or `appendBinary` points in
  `completeBinaryExplicit`, with the request identity carried into the binary
  job. A late or failed result must retain the entry and clear only its exact
  in-flight ID.
- Path-only reopen correctly reuses the current decode/fallback route:
  `loadForOpen` reports the readonly inspector and unknown-encoding
  limitation, while `reopenWithEncoding` is the explicit user action. Replaced
  bytes should not inherit a remembered code page; this is a visible limitation
  of the first slice, not a blocker.
- MyGo menu identity is safe for this design. `MenuItem.ID` stores a stable
  key, and `Chosen` matches that key rather than place/label
  (`ui/menu.go`; `ui/popovermenu.go`). A missing old ID does not choose a
  replacement item. Reuse is avoidable by giving each reclose a fresh numeric
  entry ID. Labels/tooltips can be bounded from in-memory paths without
  filesystem reads.
- The proposed history storage can remain only `{id, path}` with cloned
  bounded strings. No `Tab`, `Document`, `Buffer`, `View`, bitmap, filesystem
  handle, or persistent session field is needed. Whole-window `CanClose` and
  shutdown must not populate it.

## Boundary

This review does not ask for tab/draft/session restoration, remembered content,
generic job infrastructure, or changes outside the app's close/open/binary/menu
workflow. Those would be separate features rather than defects in this increment.
