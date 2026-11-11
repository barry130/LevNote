# Open-document picker WIP source review

2026-11-10. Static read-only review of the latest draft source and the
owner-saved evidence. This reviewer ran no test, build, product instance, or
physical-input check and edited only this handoff. This is not the final frozen
identity review: the implementation and its tests changed after the latest
passing evidence.

## Snapshot reviewed

The final source read was the `12:04:42Z` draft:

```text
native/app/open_documents.go       EB91BB1DF9D70B96D1ACB25FD7FDF493A115B218884A882DA4F53BD5AFAE0A5D
native/app/open_documents_test.go  D1556BB4E070052B7FD0980A577FE2629F0F7B8D5D83CAD094617BFAFB26B5CF
native/app/app.go                  FAD0EDE4C480E0C8F5C56F4E81CF2499BEF0007BA6EAE877FC516623BFFEA050
native/app/view.go                 64DCDC194510C6426112A7B6AAFE57A84DDE68E630A20D53C994D77CC34102C0
native/app/workspace.go            7A9ACA4F8E807D7A0563B5595976839AA7DED706EFDC5171B259079D68A16DA4
```

No reachable lifecycle, retained-reference, stale-action, or real-draft data
loss blocker was found in this draft. The remaining findings are one bounded
work concern and one test-coverage gap. The latest source also corrects the
narrow Tab traversal concern, but that correction has no current evidence
run.

## Actionable findings

1. `refreshOpenDocuments` still scans every bounded source title/path and
   recomputes each row's flags and metadata on every dialog frame. It reuses
   unchanged owned strings and the allocation regression is useful, but the
   source work is not actually skipped when the frame is unchanged. This
   satisfies bounded ownership and zero-allocation reuse; it only partly meets
   root's request to avoid rebuilding unchanged frames. Separating metadata
   reconciliation from filtering, or caching only bounded metadata plus
   source-change signals, would close that remaining performance gap without
   retaining a `Tab`. This is not presented as a whole-process memory result.

2. The `same/remove` and `same/reorder` reentry cases do not perturb their
   current-tab target. `remove` truncates after index zero, leaving `first`
   unchanged at index zero; `reorder` swaps indices one and two. The cross-tab
   removal/reordering cases are real and the source checks the exact pointer at
   the captured index, but these two subcases do not prove same-tab
   reentry/removal/reorder refusal. A same-target mutation regression would
   make the requested invariant direct rather than inferred.

## Tab during query preedit

This concern is source-grounded, not only a hypothesis. MyGo's public
`OnTextKey` contract says a key taken by the callback does not perform its
normal action, including Tab focus movement. Its engine offers a non-text key
to `OnTextKey` before the generic Tab/Shift+Tab focus path. The reviewed
`openDocumentsKey` now consumes unmodified Tab and Shift+Tab while
`p.composing` is true, so it preserves the focused native query and its
preedit by the public contract. Ordinary Tab traversal remains available once
composition ends.

The latest test now sends both Tab and Shift+Tab during real Tester composition
and checks same query focus, native composition, caret, preedit text, active
tab and document bytes. That is the right regression. It changed after the
last saved passing run, so this review does not claim that the latest source
passes it. Physical Windows IME delivery remains unobserved and is not implied.

## Source checks that passed static review

- Picker rows retain only stable ID, bounded sanitized/clipped title/path and
  folded copies, flags, and owned list metadata. No `Tab`, document, buffer,
  view, content snapshot, or worker is retained.
- `closeOpenDocuments` cancels query focus, clears row/match storage, resets
  the picker and list callbacks, and advances the generation. `Shutdown` calls
  it. A stale generation or changed origin refuses an action.
- Pointer and submitted actions are deferred. They store a row ID, then the
  next View consumes the action before rebuilding rows.
- Activation resolves the destination by stable ID, captures origin/target
  index and pointer synchronously, and validates generation, current origin,
  exact destination pointer/index/ID after Flush. The same-tab path performs
  the explicit Flush barrier that the ordinary same-index Select path omits.
- Reentrant origin/destination changes, stale async active-tab changes,
  source/rich/utility composition, 100-row virtualization/selection, and
  refusal/restoration all have direct focused regressions.
- The failed-rich-Flush regressions retain the real visible field and verify
  focus, unchanged buffer/history, message, and refusal to focus a hidden
  source field. The separate real growth-refusal test exercises the actual
  host ceiling and checks the exact editable draft recovers.

## Saved evidence boundary

`04-focused` was an actual pre-correction failure: started
`2026-11-10T12:01:26.6696839Z`, finished `12:01:32.3687074Z`, Go and shell exit
`1`, empty stderr. Its four real Markdown draft/focus subtests failed. This is
historical WIP evidence, not a hidden pass.

After the owner's real-view correction, `05-rich` passed: command
`go test ./app -run '^TestOpenDocumentsReal' -count=1 -v`, started
`2026-11-10T12:03:33.2720835Z`, finished `12:03:38.9368658Z`, Go exit `0`;
shell exit `0` at `12:03:39.1724262Z`; stderr empty. The Tab guard and its test
were added after this run, so `05-rich` does not certify the reviewed
`12:04:42Z` files.

No fresh reviewer-run identity or physical IME, RSS/heap, or whole-app memory
claim is made. Root should require a final focused pass over the frozen source
union before acceptance.
