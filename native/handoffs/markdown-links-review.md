# Markdown links — final independent review

Reviewer: GLM formats agent. This is the final read-only review of Astra's
16:43 owner checkpoint in `modules/markdownview/links.go`, `rows.go`, `view.go`,
and `preview_limit.go`, plus the resource-followup planning review. The earlier
16:20 review was stale on two facts: it discussed a clicked-target fallback that
has since been removed, and its "immutable snapshot is sufficient during
Release" conclusion was wrong. The Release crash was found by a real application
test, not inferred by this review. This pass made no product edits and reran no
tests.

## Result

The strict activation, reentrant-Release fix, and final dense-row resource
fallback all match the approved policy. No actionable product defect remains in
this review. The two stale review facts are corrected here:

1. The current link implementation does **not** fall back to a current row
   target. On `link.Clicked()`, it captures and clears `v.linkActivation`, then
   reports `errLinkStale` when the capture is absent or its row/relative-syntax
   identity does not match. Only an exact capture proceeds to `activateLink`.
   Canceled releases still do not produce a native click, and a canceled or
   modifier-lost release clears activation. This is the strict behavior root
   requested.

2. The real reentrant Release defect was not caught by the immutable
   `previewSnapshot`. During a local-file link activation, `Host.OpenReference`
   can synchronously call `App.Open`, which releases the source tab's
   presentation view while MyGo is still building the active List row. Zeroing
   the active `ui.ListState` underneath that pass caused the application crash
   found by the real app test. Immutable `doc` and `rows` arrays did not cover
   this because the native list itself retains mutable state through a
   pointer.

The current fix is concrete: `View.list` is now `*ui.ListState`, and both
`View.Release` and `dropPreviewCache` assign `v.list = &ui.ListState{}` rather
than clearing the old state in place. The active row callback keeps the old
pointer until that List pass finishes, while the next view build receives a
fresh empty list. This accurately addresses the proven synchronous
`OpenReference -> App.Open -> View.Release` path. `dropPreviewCache` uses the
same ownership swap, so cache refusal/drop does not reintroduce the mutation.

This correction supersedes the earlier review's removed-fallback discussion and
its claim that immutable `previewSnapshot` alone made synchronous Release safe.

## Current link identity and activation review

`linkTarget` retains the buffer pointer/version, row ID, complete enclosing
UTF-16 range, exact inline syntax range, block kind/item flag, and a cloned
full href. It does not use labels, display text, link ordinal, or an old
parser document as identity.

`activateLink` performs Flush first, then checks buffer identity, the 4096-byte
href budget, foreign transaction state, and oversize input. It maps both
enclosing and syntax ranges through the edit journal, reparses the
authoritative buffer when cached state is stale, rejects partial input, and
requires the current bounded parse to contain the exact mapped enclosing
range, kind/item flag, inline range, and unchanged href before calling
`Host.OpenReference`. An unrelated insertion above can therefore map and
succeed; target overlap, deletion, Restore, journal expiry, buffer
replacement, changed fence/grammar context, partial input, oversize input, and
href changes fail visibly.

Navigation itself creates no source transaction or `Changed` notification. A
preceding successful Flush may have edited source; the stale-link wording does
not claim otherwise. `findLink` advances one cursor through the current parsed
rows and does not rescan the whole source prefix per visible link.

## Resource finding: link-heavy rows

Astra measured one under-32KiB row containing 700 links across five frames at
approximately 2.15GB of allocations. This is a real resource defect, not a
style-only concern: the row remains under the current 32KiB block limit and
the List virtualization cannot help because all 700 links are in the same
visible row. Each interactive link currently contributes a native child text
widget, key, focus target, tooltip/metadata, and retained closure, so the
worst-case interactive-tree cost grows with links in one row and repeats every
frame while it is visible.

Once Astra chooses the exact policy, a bounded-interactive-links fallback must
meet these constraints:

- Bound the number of **native interactive children** per row (and, if needed,
  document-wide), not merely the href length. The observed workload is a single
  under-limit row, so source-byte size alone cannot predict it.
- Render the non-interactive remainder without constructing another native
  focus/input widget for every omitted link. Marking all 700 widgets disabled
  would preserve the allocation problem.
- Keep link styling/source text intact, but make the degradation visible and
  deterministic. A nearby row or view notice should state that interactive
  links were capped and that source mode contains the complete links; it should
  not leave inaccessible later links to be discovered by Tab order.
- Prefer source order for the first bounded interactive links so keyboard and
  pointer behavior are reproducible. The omitted links must not gain a clicked
  or current-row fallback; absent or stale activation must remain `errLinkStale`.
- Preserve the existing 4096-byte href refusal and the exact
  Flush/identity/current-parser validation for every link that remains
  interactive. The fallback must not create a second, weaker link activation
  path.
- Keep source selection/copy and double-click behavior independent from the
  interactive-link count. Users must still be able to reach and copy every
  authored link from the preview/source, even when not all links are
  keyboard-focusable.

The cap threshold and whether the remainder is plain styled text, a bounded
tooltip-only route, or another Astra-chosen presentation are policy decisions
for the markdownview owner. This review does not propose changing parser or
URL semantics.

This section records the pre-fix finding and constraints. The final checkpoint
below resolves it with the approved whole-row fallback and the measured
allocation reductions; it is retained as provenance rather than as a current
blocker.

## Final dense-row policy checkpoint

The final policy is visible in the current source:

- A linked row is interactive only when it contains at most 16 parsed links
  and at most 64 parsed `InlineSpan` values. Exactly 16 links / 64 spans remain
  interactive; 17 links or 65 spans triggers the whole-row fallback.
- The fallback uses the existing bounded readable `spans` path. It preserves
  the theme link foreground, clears the link underline, creates no `RoleLink`
  child, and clones no href into a retained target.
- Because the dense row is one selectable `ui.RichText` rather than native
  inline children, parent-level selection and double-click remain available.
- The row-level reason is rendered above the row: dense-link and
  styled-span workloads get distinct nearby guidance directing the user to the
  full source targets. It is not merely a defined constant.

Astra's owner checkpoint records complete `./modules/markdownview` PASS 2.050s
and vet with no diagnostics, including 16/17 and 64/65 boundary tests, the
long-label and mixed-style probes, selection/double-click behavior, and source
preservation. This review does not duplicate those tests or claim independent
execution evidence.

## Reentrant Release and app liveness

The accepted app guard remains intentionally narrow: `openReference` rejects a
nil or non-member source `*Tab`, and rejects a member tab that is no longer
current before resolving the reference. No broader tab-generation mechanism is
warranted by the reproduced path.

For a same-document anchor, the guard routes through the existing outline path
and does not release this view. For a local file target, the existing ordered
`App.Open` barrier may synchronously release the source view; the current
`*ui.ListState` ownership swap described above preserves the in-flight native
List pass. The app guard then rejects stale source tabs before routing a
reference captured from them.

## Input, readonly, resources, and destination policy

- Cmd/Ctrl pointer down and keyboard Enter capture the immutable row target.
  Canceled pointer release or modifier loss clears activation. The inline
  text is exposed as `RoleLink` but remains a native text child, preserving
  paragraph selection and the enclosing row's double-click route.
- Readonly links remain enabled for navigation. The control's disabled state
  is reserved for a missing host callback or an over-budget href.
- Full hrefs over 4096 UTF-8 bytes are not copied into a target or truncated
  into another destination; the authored source remains and the unavailable
  state is visible.
- Accessible names are bounded to 128 bytes and tooltip detail to 512 bytes
  with rune-safe ellipses. Retained closures hold numeric ranges and bounded
  cloned hrefs, not slices of the full parser document.
- The view does not decode, normalize, split, or classify destinations.
  `app/reference.go` owns local/anchor/URL policy and visibly rejects network,
  UNC, malformed file URLs, unsafe relative targets, and unsupported schemes.
- The current dark link foreground is theme-specific (`#7dd3c4` dark,
  `#24675d` light). That contrast fix belongs to Astra's final styling/test
  evidence and is not a new blocker from this pass.

## Execution evidence and provenance

This reviewer did not rerun tests. The synchronous Release crash was observed
by the application integration test, not inferred here; Astra's focused package
evidence should remain the source of record for the pointer-swap fix. DeepSeek's
app evidence covers reference routing and the real adapter. Astra's 16:43
record is the source for PASS/vet timing and the allocation figures, including
the pre-fix 700-link five-frame sample and the final 4.6967ms / 3,935,736B /
106-allocation sample. This review makes no execution claim that supersedes
that evidence.
