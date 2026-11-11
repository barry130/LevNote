# Drawio worker stage2 preflight

## Definite missing guards

1. **Source mode cannot be inferred from `View == nil`.** `releasePresentation`
   calls `View.Release()` but retains `t.View`; `disposePresentation` is the only
   variant that clears it. The toolbar draws the export button before lazy view
   creation and before the `t.Mode == "source"` early return. Stage 2 must gate
   the button on `capability.RequiresView && t.Mode == "source"` separately from
   non-view engines, and `startEngineExport` must repeat the same mode check
   because `exportCurrent` is callable from methods/tests, not only the button.
   A retained, previously valid page-0 view must not authorize export.

2. **The current identity snapshot has no view or mode identity.** `exportJob`
   checks tab, buffer, adapter, capability, and buffer version, but page changes
   do not alter the buffer version. Add the captured view pointer, exported
   epoch token, mode, and copied page `{Index,ID}` to the job. Keep existing
   generation/lane/engine-generation checks unchanged.

3. **Preparation can invalidate a pre-captured page.** `flushTab` can leave graph
   cache stale when there is no pending draft, so preparation must run after the
   source-size and group gates. Graphview's cache synchronization may report or
   recover and therefore reenter. Save the post-Flush version before
   `PrepareExportRequest`; after preparation, require unchanged tab, doc,
   buffer, editor, adapter, capability, view pointer, mode, version, group, and
   token. Only then snapshot and enqueue. Do not use a second `Flush`.

4. **Reparse and release are not merely page-index changes.** Increment the
   export epoch in `Release` and whenever graphview actually clears/replaces
   its parsed cache, even when the selected index remains the same. Increment on
   every actual index assignment, including draft restoration and A→B→A. Do not
   increment on `ensureSynced` cache hits or unchanged `Draw`/`Warnings`.
   Existing direct `pageIndex` assignments are in `sync.go`, `draft.go`, and
   `changePage`; inventory all before implementation.

5. **A released view object can later become live again.** Existing tests call
   `Release`, then `Warnings`/`Export`/`Draw` on the same pointer. A dead flag is
   the wrong model. Epoch is monotonic; a later prepared request may use the new
   epoch. The job only accepts the exact captured token.

6. **Final revalidation must be side-effect-free.** `recheckExport` currently
   flushes. Preserve that behavior, but after the flush check the captured
   pointer/token/page plus existing identity and version. Do not call
   `PrepareExportRequest` again. Apply the same check before publication and
   after the native save dialog.

7. **Native-dialog reentrancy must not bypass the reservation.** Keep the
   existing `deferExportDelivery`/`modalDepth` structure: nested modal callbacks
   complete the job, delivery is deferred, and exactly one save dialog runs at
   modal depth 1. Add page/epoch revalidation to the existing post-dialog path
   rather than creating a second delivery route.

8. **The button path must not lazily mount a graph merely to export.** `View`
   is created after the toolbar in `preview`/`split`. A nil view for a
   view-required engine must visibly refuse with the select-page instruction.
   Normal preview/split rendering may still lazily create the view; do not make
   Markdown/Mindmap exports depend on a mounted view.

9. **Size refusal must happen before preparation.** `PrepareExportRequest` may
   parse and should not run on a source already above the 32 MiB engine limit.
   The app source-size check must precede preparation and use Drawio-specific
   error text.

## Reuse and add tests

Existing fixtures to reuse:

- `app/export_worker_test.go`: `controlledExporter`, `awaitExportCall`,
  stale-identity tests, mutating-Flush tests, nested-modal deferral, shutdown and
  cleanup joins.
- `app/export_mindmap_worker_test.go`: mode matrix, readonly publication,
  callback/reentrant Flush (`callbackExportFlushView`), source-mode no-view
  refusal, exact source/dirty/history/selection assertions.
- `modules/graphview/graphview_test.go`: second-page selection and page
  navigation.
- `modules/graphview/safety_test.go`: pending draft blocks page change/export,
  missing draft page never falls back, `Release` rebuilds and retains view state.
- `app/acceptance_test.go`: real Drawio fixture and 760×480 capture patterns.

Add focused cases rather than a new matrix:

- Graphview: epoch unchanged on cache hit/Draw; changes on A→B→A, same-index
  reparse, and Release; preparation returns copied second-page index/ID/token;
  missing buffer/parser/page, overlong ID, invalid index, and group refusal;
  failed Flush retains draft/history.
- App: exact post-Flush source and page metadata after size/group gates;
  preparation reentrancy changes version/view/tab and refuses; source mode and
  nil-view make no runner call; A→B→A, retained released pointer, same-index
  reparse, and post-dialog page/version/view changes refuse and free payload;
  successful exact publication leaves original disk/source/BOM/dirty state
  unchanged and, when Flush commits a draft, asserts that one intended undo
  transaction rather than falsely requiring history unchanged.
- UI: source-mode label/tooltip at 760×480 light/dark; fourth engine package
  panel layout. No physical-IME or resource-peak claim.

## Minimal implementation map

1. `ui/editor/export_request.go`: copied `ExportPage{Index,ID}`,
   `ExportToken`, `ViewExportRequest`, optional `PrepareExportRequest`, and
   side-effect-free `ExportRequestToken`. No engineipc/engineruntime import.
2. `ui/editor/contract.go`: add `ExportEngine.RequiresView bool` only.
3. `modules/graphview`: add monotonic export epoch and one preparation method for
   Drawio. Centralize actual page-index changes and real cache invalidation.
   The method may call `ensureSynced` and returns a copied index/ID/token after
   parser limits and ID-bound checks. It does not Flush, publish, or discard
   drafts. Advertise the fourth worker only in Stage 2.
4. `app/view.go`: source-mode/nil-view button state and visible instruction for
   view-required Drawio while preserving the current button for old engines.
5. `app/export_worker.go`: extend start/recheck identity with view, mode, token,
   and page; preparation after flush/group/size; owned page copy in
   `Source.Page`; snapshot after final revalidation; post-dialog revalidation.
   Leave lane/cancel/timeout/modal-depth/poison behavior intact.
6. `app/engine_packages*.go`: list the fourth catalog entry and use count-neutral
   wording. No runtime-manager edits unless Stage 1 reports a frozen defect.

Stage 2 must wait for accepted Stage 1 output and must not rework lower-layer
half-edits. The synchronous `View.Export("svg")` API remains for regressions, but
the app's Drawio route must use the worker exclusively with no silent first-page
fallback.
