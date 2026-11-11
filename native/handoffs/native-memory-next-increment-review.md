# Next memory-oriented increment: enforce the live-tab ceiling

2026-11-10 read-only source review. No source/test/build/process/package work
was performed. `.26` remains accepted and frozen.

## One recommended increment

Apply the already accepted `session.MaxTabs == 100` ceiling when the running
workspace creates a tab. Today that ceiling exists only in recovery
serialization/loading. A user can open a text or image file repeatedly, drag in
an arbitrary path list, create documents, or recover drafts beyond 100 live
tabs. Those tabs retain documents and histories, while the next recovery save
fails the 100-tab preflight. This is both an unbounded live-retention path and a
recovery-capacity mismatch.

The recommended user-facing behavior is:

> `最多可同时打开 100 个标签。请先保存并关闭一个标签后再打开新文件。`

Refuse only the new tab. Keep every existing tab, selection, draft, undo/redo
history, binary result and the previous recovery file intact. Do not silently
close, evict or truncate a tab. Opening a path that is already in the workspace
at the limit must still select/reveal that existing tab.

## Source-grounded ownership

- `core/session/session.go` defines `MaxTabs = 100`.
- `core/session/builder.go` preflight rejects addition beyond that count, and
  `core/session/load.go` decodes at most that many entries and reports omitted
  records.
- `app/app.go` `appendTab` and `app/binary.go` `appendBinary` append without a
  live count check. `openRequests` iterates every supplied path; `NewDocument`,
  `NewFromPlugin`, `recoverDraft`, binary open completion and session restore
  also create tabs through these seams.
- `app/state.go` therefore can hold more tabs than its own
  `SnapshotBuilder.AddTab`/`AddBinaryTab` calls can persist. Existing recovery
  remains protected by the builder's failure behavior, but drafts opened after
  the last successful snapshot are not added to recovery.

The app already imports `core/session`, so no new package dependency or format
version is needed.

## Minimal source change

Add one small app-owned admission helper, tentatively
`canCreateTab() error`, which compares `len(a.Tabs)` with
`session.MaxTabs` and returns the visible refusal above. Use that same helper
at every production tab-creation origin:

1. `addDocumentChecked`, before the current tab is flushed or released. This
   covers ordinary text opens, new built-in documents, plugin templates,
   byte-inspector fallback and restored text if the invariant is ever reached
   through another path.
2. `openRequests`, after the same-path lookup and before expensive file/image
   loading. Keep the check again in `addDocumentChecked` because the load/UI
   handoff is asynchronous.
3. `recoverDraft`, before changing the current presentation or appending the
   in-memory recovery document.
4. `openBinaryRequest` before scheduling image IO, and
   `completeBinaryExplicit` after its same-path lookup and before flushing the
   current tab. The second check handles a workspace reaching the cap while
   image IO is in flight; a refused result must be dropped through the existing
   binary-job cleanup.

Keep the count comparison centralized so the app cannot drift from
`session.MaxTabs`. No session schema, budget, view, binary or text-buffer change
is required. Direct internal test helpers may continue appending tiny fixtures,
but production callers should all pass through admission.

This app work is currently behind Astra's active closed-file ownership. It
should not be implemented until that owner freezes/releases `app/app.go` and the
binary creation seams.

## Data-safety constraints

- Check before `flushTab` or `releasePresentation` when possible. A rejected
  open must not make an unrelated format draft commit or discard a view.
- Check after same-path deduplication. At the ceiling, an already-open path is a
  successful select/reveal, not a new-tab request.
- Never evict the least-recent tab, reset history, clear an unsaved recovery
  draft or alter the active document to make room.
- Preserve existing close/save/discard prompts and failed-persistence behavior.
  This admission check does not claim to make more than 100 tabs recoverable.
- Leave the current 100 limit unchanged in this increment. A smaller live-tab
  policy would be a separate product decision and needs its own UX/retention
  analysis.

## Deterministic validation

One focused `app` test is sufficient for the proposed implementation:

- Build exactly `session.MaxTabs` in-memory tabs with tiny content, recording
  pointers, active index and one document's draft/history.
- Prove the 100th creation succeeds and each of `NewDocument`, a new text open,
  a new image completion and `recoverDraft` refuses tab 101 with the visible
  message while all recorded state and the previous session bytes remain
  unchanged.
- Prove an existing-path open at the ceiling still selects/reveals that tab and
  does not report the limit.
- Close one tab and prove the next creation succeeds, covering the off-by-one
  boundary.

This is an in-memory ownership/admission check. It does not measure process RSS,
working set or GPU memory.

## Existing fixes ruled out

- Text undo/redo is already bounded per buffer to 256 entries and 32 MiB, and
  closed-tab backing slots are explicitly cleared. The residual issue is the
  absence of a live-tab count ceiling, not stale closed-tab references.
- Inactive image paths retain only path/metadata; the held source snapshot is
  cleared and presentation release/disposal are already owned.
- Rich-format parsed caches have explicit `Release`; draw.io's private/current/
  public source representation was deliberately reviewed and reduced.
- Workspace search is bounded by file/entry/result limits; active-document Find
  is bounded to 200,000 matches.
- Session JSON no longer creates Marshal's extra returned full-output copy.
  The standard encoder still owns one complete encoded value and dirty draft
  strings remain, but changing that exact-wire encoder is wider and more
  compatibility-sensitive than this tab-admission gap.

The live-text ceiling bounds legal source/history retention by tab count. Each
tab still has substantial independent limits: editable text up to 128 MiB and
undo/redo up to 32 MiB. A 100-tab cap is therefore a consistency and
unbounded-growth fix, not a claim that the resulting source-allocation ceiling
is small and not evidence of an RSS or whole-process improvement.

## Tradeoff and evidence limits

A user attempting to open more than 100 files in one action will receive a
visible refusal instead of an ever-growing workspace. They must close a tab and
retry. That is preferable to silently losing recovery coverage beyond the
accepted format limit, but it changes a currently permissive UI path and should
be implemented as one assigned app increment after Astra's freeze.

This review establishes source ownership and a bounded user-trigger scenario.
It does not establish current process memory, expected RSS reduction, GC timing,
physical UI behavior or performance.
