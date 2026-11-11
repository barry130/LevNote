# Native image app independent review

2026-11-10 · GLM read-only review of Astra's current WIP. Scope:
`native/app/binary.go`, `binary_lane.go`, `binary_actions.go`, `state.go`,
`app.go`, `plugins.go`, `view.go`, `workers.go`, `workspace.go`, and the
document/session contracts those paths call. No source edits and no
test/build/launch execution were performed.

## Result

No additional final blocker beyond the concrete items already recorded in
`native-image-app-root-review-notes.md`. Those items are not repeated here.
The current source addresses the reviewed lane-reservation, modal-headless,
window-close, and awaiting-result ownership concerns coherently.

## Maximum live source ownership

The truthful steady state is one accepted compressed snapshot (`a.binarySnapshot`)
and at most one live decoded bitmap. A second compressed snapshot exists only
during the bounded explicit open/reload transaction:

1. The old tab keeps its accepted snapshot while `OpenBinary` creates a new
   `BinaryFile` plus snapshot in the explicit result slot.
2. The new `awaiting` gate prevents any subsequent full read/decode/copy until
   the UI consumes or abandons that result.
3. Successful adoption replaces the tab owner and `attachBinarySnapshot`
   overwrites `a.binarySnapshot`, releasing the old snapshot. Failure or an
   invalid initiating tab reports/drops the new result and preserves the old
   accepted authority. `completeBinaryExplicit` clears the slot and releases the
   explicit reservation in all outcomes.

Normal A→B navigation does not add a second noncanceled source. Selecting away
cancels the preview job, clears its tab/view/callback references, and calls
`deactivateBinary`, which releases the old snapshot and bitmap. Superseded
pending preview requests are finished as canceled and their output is not made
ready. A new B preview may then read one snapshot; no inactive A snapshot is
retained. If B is already active with its own held snapshot, an explicit open of
C creates the same bounded old-B/new-C transaction described above.

Preview/bytes/copy requests pass only immutable token/snapshot data to the
lane; no worker captures or mutates a UI-owned `BinaryFile`. Copy is transactional
too, but its retained source is the initiating snapshot itself, not a second
independent read.

## Other reviewed paths

- Startup `Restore` is synchronous and intentionally blocks while binary files
  are read one at a time. Each inactive result is consumed and dropped before
  the next read; only small path/kind/title metadata remains. `persistTabs`
  refuses writes while restoring, so a partial assembly cannot overwrite the
  existing session. Text dispatch follows the same mixed order, and active index
  mapping accounts for skipped records.
- Mode switching and plugin reconfiguration preserve the active binary snapshot
  only while that tab remains current; releasing view state retires the bitmap.
  Leaving the tab drops snapshot and pixels, and returning requires token-verified
  reacquisition.
- Duplicate opens route through the canonical-path tab search and activate the
  existing tab; they do not create a second owner. Explicit reload checks tab,
  owner, token, and generation before adoption, and failure preserves the old
  authority.
- External change checks use the current immutable binary token and ignore stale
  tab IDs/baselines. Missing or changed disk state marks conflict without
  replacing accepted bytes.
- Readonly/command gates are based on actual binary authority: binary tabs have
  no text find/replace, GoToLine, outline, split, export, or new-image path.
  Binary status shows format, dimensions, byte size, and readonly rather than
  line/column/encoding.
- `Shutdown` directly cancels/closes and joins the lane, clears explicit and
  preview jobs, releases final bitmaps, and clears the retained snapshot without
  requiring `CanClose`. Committed copy outcomes remain small receipt/accounting
  records; ordinary window close now refuses once while the explicit outcome is
  resolved and displayed.

## Verification recommendation

Proceed with Astra's one combined final affected suite/vet. Preserve focused
logs and add no new API or framework work from this review.
