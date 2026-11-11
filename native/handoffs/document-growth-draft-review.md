# Document growth draft/refusal integration review

2026-11-10T10:41:37Z. Static review only; no source, product docs, product
run, benchmark, build, or test execution. Snapshot is the source below; Astra
remains the sole implementation owner. Findings are about reachable state
transitions in the inspected snapshot, not assumptions that the final guard/API
is already frozen.

## Concrete finding

**P1 - source IME refusal is transient, so a later Flush can authorize a host
action with stale or partial text.**

Reachable trigger:

1. A source composition has already accepted a partial preedit. For example,
   a tiny host limit accepts `"abc"` as the current marked text but rejects a
   subsequent `"abcd"` update.
2. `native/ui/editor/editor.go:296-324`, `SetMarkedText`, skips its initial
   preflight because `started` is false. `ReplaceUTF16` fails at317, `fail`
   reports the refusal, but lines319-322 intentionally leave `composing` and
   `marked` pointing at the previous accepted partial preedit.
3. `native/ui/editor/editor.go:343-345`, `Flush`, calls `UnmarkText` and returns
   nil. `UnmarkText` at334-340 closes the composition group and clears the
   marked range. It does not return, preserve, or retry the refused update.
4. `native/app/app.go:1239-1255`, `flushTab`, treats that nil result as success
   and then flushes the view. Save, select/open, close, reload, and other
   lifecycle paths use this gate, for example `Save` at916-1007,
   `selectChecked` at383-425, `CloseTab`/`prepareClose` at1017-1052,
   `addDocumentChecked` at461-470, and `Reload` at1212-1235.

Exact loss on refusal:

- The host buffer still contains the last accepted partial preedit after the
  rejected update, so immediate buffer source/version/selection/history are
  unchanged by that failed `ReplaceUTF16`.
- The user's current refused preedit is not represented by `MarkedRange`.
- A host save can nevertheless succeed, persist the previous partial text, and
  clear the engine's notion of composition. A switch or close can similarly
  proceed. Sealing the group in `Flush` can also make the accepted partial
  composition become an undo/history entry while the refused current text is
  lost.

The first-composition branch at317-323 has the same sticky-state problem if a
post-preflight `ReplaceUTF16` refusal is reachable: it ends the group, clears
`composing`, reports the error, and later `Flush` returns nil. Current
`CheckReplaceUTF16` may catch the normal size case before `BeginGroup`; verify
that after the implementation is frozen and do not manufacture a race to make
the branch reachable. A refused first composition still needs a visible,
lifecycle-blocking retry/cancel state because the preflight path at307-310 also
returns without any persistent refusal state.

Minimal implementation direction for Astra:

- Keep a sticky refusal in `Editor` when either preflight or replacement
  rejects marked text. `Flush` must return that error until a subsequent marked
  update is accepted or the UI receives an explicit IME cancellation.
- A successful retry clears the refusal and proceeds normally. An explicit
  cancel may deliberately settle/discard the draft, but `Flush` from Save,
  switch, close, reload, or open must not turn refusal into success.
- For a first replacement failure after `BeginGroup`, use transaction cleanup
  that guarantees no group/history state from the failed attempt. Preserve the
  last accepted partial composition on an ongoing failure.
- Report visibility alone is insufficient. `OnError` cannot stop `flushTab`.

## Real rich-view paths

No additional draft-release defect was found in the inspected production
writeback paths.

- Markdown block draft: `native/modules/markdownview/draft.go:71-121` calls
  `ReplaceUTF16` at107; refusal stores `draftError` and returns before any
  `editBase`, version, history, or `Changed` update. `commit()` at240-247 clears
  only after a successful `Flush`.
- Markdown table-cell draft: `native/modules/markdownview/table_cell.go:395-417`
  returns at408-410 on host refusal, before `finishTableCellDraft` at412.
- Markdown recovery: `draft.go:249-271` clears at267 only after
  `host.RecoverDraft` succeeds. A real capacity-refused callback retaining its
  draft is already covered by `native/app/live_tab_admission_test.go:247-311`.
- Graph field writeback: `native/modules/graphview/edit.go:22-38` starts its
  transaction, and a `ReplaceUTF16` refusal calls `CancelGroup` before return.
  `graphview/draft.go:198-357` returns the error through `Flush` without
  clearing the field draft. `graphview/draft.go:359-408` clears recovery state
  only after successful recovery and identity checks.

These paths therefore stop the real app action through `flushTab` once the
view `Flush` returns the source error. The source composition defect above
precedes this protection and has no equivalent error return.

## Focused real-seam tests

1. Add one tiny-limit source App test, preferably near
   `native/app/app_test.go:622-652` and the source composition coverage at
   `native/ui/editor/growth_test.go:36-84`:
   start an accepted partial composition, submit a one-byte-over update through
   `ui.Tester`, then assert `Editor.Flush` retains `ErrSizeLimit`. Invoke the
   public App Save, Select, and CloseTab paths; each must refuse. Assert tab and
   document identity, current tab, source, version, selection, history/group
   ownership, exact marked/retry state, and disk are unchanged. Retry with a
   smaller valid marked value; only then should the selected host action
   succeed. Include the first-composition refusal case, asserting the refusal
   remains sticky until an explicit retry/cancel.

2. Add one compact real Markdown or graph draft case using the existing real
   view/Tester helpers. With the host just below its tiny ceiling, edit a real
   field so writeback is refused; then call Save, Select, and CloseTab and
   assert each refuses while the exact `pendingDraft` text and view identity
   remain. Delete enough text to fit and retry or invoke the real
   `RecoverDraft` seam; that success is the only point allowed to release the
   draft. This complements, rather than rewrites, the existing real recovery
   capacity-refusal test.

Tests should exercise public tester/app seams. They do not need injected
callbacks around `ReplaceUTF16`, forced goroutine interleavings, or a second
concurrency harness.

## Inspected snapshot

```text
3568bbae46d0931446ee59a211c23386ad9e68ecc0e2b95f8ed827c729b2bfc8  native/core/text/buffer.go
cc7d5510f81abbc10704cf0ef32f4ff9fc6f96de66937d1af19ded312b4fca8d  native/ui/editor/contract.go
c108fc2107da573f0919a73acd71fdea1bdb5f2b4456b491dba07043609af9c5  native/ui/editor/editor.go
e02a59da5fc713f5ea9db2b8fa656f00b92edf39daed137f29291c8cfffecb22  native/modules/markdownview/draft.go
2ed3e259c91be0f8698b9b56aa22a44d41c63d19f996298d196262cd8853c998  native/modules/markdownview/table_cell.go
b482887442d57103cb0b610d4469882ccde32f2d665cbf96ac444154e19c1da2  native/modules/graphview/draft.go
3b9c2188a36d25eeacc2aa767f7e45afcd4532eccd305102faf9214de83068e3  native/modules/graphview/edit.go
680840af4782e3481bfdfeed3d0b73d1f3f2e623920183aec99bb7981378edf9  native/modules/graphview/adapter.go
0ac7fb907c83dc3c2b7642b197441492274cbb1cbc75e6688264d577d995adb4  native/app/app.go
195afa3d82e7612a0cdbec6a9d0f0432777f492cab4d94596b0c6f2329c83456  native/app/view.go
7f3f8e338e5619241fa3d0acc72ba791778f1ce2a328272cbaed12e40b9400fd  native/app/state.go
c585759102fbcbeb3b14e8d4d61511ce98d189c17fb96f4aff6be98a3ebf475e  native/app/search_navigation.go
c2948cf459ed04772a27c587ee0618b018b44d19d2ee8f07b5005710e2807e55  native/app/source_navigation.go
```
