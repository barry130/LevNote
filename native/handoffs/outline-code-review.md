# Independent outline implementation review — corrected

Date: 2026-11-08  
Scope: read-only review of the landed outline implementation and lifecycle
integration. This revision corrects the earlier report after root's source
review. No product, test, or documentation source was modified, and no tests
or builds were run.

## Verdict

The four numbered findings in the earlier review are withdrawn: one was false,
one was unsupported without a reachable edit sequence, and two were either an
approved design scope or an accepted bounded cost. One real regression was
found in writable hidden mindmap navigation: a split source caret moved to EOF
when expanding collapsed ancestors. In the current checkout, that regression is
already addressed by Astra's selection-preserving transaction and covered by a
hidden-descendant split test. I found no remaining material correctness finding
in the reviewed outline paths.

## Corrections to the original findings

1. **Current-item tracking was not a defect.** `CurrentOutlineProvider` is an
   optional snapshot-time hint, not a caret, scroll, or graph-selection
   subscription. The outline list is selected-row-only and does not promise to
   continuously follow source caret movement, preview scrolling, or visible node
   selection. The early version-cache return in
   `native/app/outline.go:145-195` is consistent with that approved scope.

   Documentation should state plainly that the sidebar selection is
   selected-row-only and does not continuously follow caret, scroll, or node
   selection. This is a documentation acceptance item, not a code finding.

2. **The complete-preview/anchor-clipping finding was false.** The current source
   at `native/modules/markdownview/outline.go:57-58` is:

   ```go
   headings, truncated := markdown.Headings(doc, editor.MaxOutlineItems)
   s.truncated, s.partial = truncated, doc.Partial
   ```

   Anchor/outline clipping is therefore separated from the parser's actual
   `Document.Partial` state. Preview navigation at
   `native/modules/markdownview/outline.go:148-150` correctly refuses only
   parser-partial work. The finding is withdrawn.

3. **The duplicate mapped-range finding is withdrawn.** No reproducible sequence
   was supplied under the actual guards. `MapRangeSince` rejects overlapping and
   target edits, and current heading block ranges are disjoint. Replacing the
   source range identity with an ordinal could also break a valid insertion
   before a heading, where the existing range mapping correctly moves the old
   heading and preserves its identity. I am not proposing that redesign.

4. **The duplicate bounded cache is an accepted cost, not a finding.** The app
   and mounted view can each retain a bounded outline cache. Root accepted this
   finite duplicate cost; no shared mutable parsed document is needed at this
   stage.

## Real regression: hidden mindmap navigation moved a split caret to EOF

The concrete pre-fix path was:

1. In split mode, outline navigation for a mindmap descendant hidden by a
   collapsed ancestor reached `applyMindmapMutation` in
   `native/modules/graphview/outline.go`.
2. That path committed the whole re-serialized source through
   `commitBuffer` in `native/modules/graphview/edit.go`.
3. `Buffer.ReplaceUTF16` at `native/core/text/buffer.go:231-249` places the
   selection after a full replacement at `start + UTF16Len(value)`, which is EOF
   for a replacement of the whole document.
4. Mindmap items have `Offset == -1`, so the app's split branch at
   `native/app/outline.go:389-417` navigated the graph pane, left the changed
   source selection untouched, and reported “已在图形中定位；此格式没有源码定位坐标”.

The user-visible sequence was: open a writable mindmap in split mode, put the
source caret at a non-EOF position, and activate a descendant hidden by one or
more collapsed ancestors. The graph expanded correctly, but the source caret
jumped to EOF. A split test that only activates the root does not reach this
path because the root has no collapsed ancestor and no source transaction.

Current source status: fixed. Hidden navigation now calls
`applyMindmapMutationSelection(item.ID, true, ...)` at
`native/modules/graphview/outline.go:171-182`. The `true` argument enables
selection restoration inside `commitBufferSelection`
(`native/modules/graphview/edit.go:18-55`): it clamps the old numerical UTF-16
endpoints to valid boundaries, restores them before `EndGroup`, and reports a
warning only when clamping was required. This also keeps undo/redo from moving
the source caret to EOF.

The current regression at
`native/app/outline_builtin_test.go:372-430` is the appropriate acceptance case:
it uses a hidden descendant, starts from a non-EOF source range, verifies the
split pane and view survive, verifies the selection remains unchanged after the
expansion, checks the graph/source limitation message, and verifies exact
selection preservation across undo and redo. I did not execute it in this
read-only review; it should be included in Astra's focused test acceptance.

## Verified strengths retained

- Source-mode discovery creates no heavy view. `outlineSourceFor` uses
  `Adapter.Outline`; preview/split navigation creates the view only when the
  current mode requires one.
- Activation uses a fallible flush and revalidates tab, adapter, path, mode,
  buffer identity, outline controller generation, and buffer version around
  resolution and navigation.
- A failed flush, stale target, or conflict retains the tab, view, draft, and
  selection instead of discarding content.
- Mindmap outline extraction includes folded descendants without constructing
  layout or canvas.
- Readonly hidden navigation fails before mutation and leaves model, viewport,
  and source state unchanged.
- Writable hidden navigation expands all collapsed ancestors in one undoable
  transaction and, in the current source, restores the split source selection
  within that same transaction.
- Outline caches are released on hidden outline, tab/mode presentation release,
  reload/reconfigure, and view release.
- Empty but truncated snapshots retain the partial notice instead of claiming
  that the document has no outline.
- Plugin reconfiguration releases the active outline source rather than leaving
  it attached to a removed adapter.

## Acceptance items for the owners

- Keep the hidden-descendant split test in the acceptance run. It is the case
  that distinguishes the EOF-selection regression from a root-only test.
- Documentation should explicitly describe selected-row-only outline selection
  and avoid promising continuous caret, scroll, or node-selection following.
- Keep the bounded duplicate app/view cache behavior documented as an accepted
  finite cost; do not add a shared mutable parse state for it.
