# Next slice: searchable already-open documents

2026-11-10, Astra. Read-only plan; supersedes the earlier checkpoint's in-content
panel suggestion. Composition source and .29 inputs remain frozen. No runtime
edits, tests, builds, launches or external software audit were performed.
Implementation starts only after .29 closure and explicit root assignment.

## Small product surface

- Add an “已打开文档” button outside the horizontally scrolling tab strip,
  with “Ctrl+P” tooltip; `ui.Cmd, ui.KeyP` is unused in `app/view.go:shortcuts`.
  KeepFocus on its pointer entry preserves the current input until admission.
  Repeated Ctrl+P while open focuses its existing query; it creates no queue.
- One native dialog, about 560 DIP wide and at most 360 DIP high at 760×480,
  clamped to the window with margins. Query, result count, virtual list, one
  short hint/error line and Cancel. No animation or editor creation for rows.
- Preserve tab order. Match title or path by case-insensitive Unicode substring
  (`strings.ToLower` + `strings.Contains`); no fuzzy, accent or normalization
  equivalence promise. Empty query lists all open tabs; no matches is explicit.
- Each two-line row shows title, muted path, active marker and applicable
  “未保存 / 只读 / 二进制” badges. Unsaved tabs say “尚未保存到文件”; duplicate or
  shortened names remain distinct with a small stable tab number. No previews.

## Actual state and public controls

- `app/app.go`: `Tabs`, stable `Tab.ID`, `Current`, `Select`, `selectChecked`,
  `flushTab`; `app/binary.go:Tab.path()` supplies text/binary-neutral paths.
  Reuse the existing live admission maximum100; do not add a second authority.
- New `app/open_documents.go`: transient picker state with generation, origin
  tab ID, selected tab ID, query, bounded row metadata, list state and UI handles.
  No stored Tab/Document/Buffer/View pointers, snapshots, content or workers.
  Indices exist only to satisfy the current list widget and are never identities.
- At most100 rows; clone UTF-8-boundary-clipped title≤256 bytes and path≤4096
  bytes, sanitize control characters, bound folded copies too. Query≤128 UTF-16
  units via public `MaxLength(128)`. Explicit hint when a path/title is shortened:
  filtering searches the retained text only. No full-path tooltip escape hatch.
  Reconcile this small metadata set while visible; nil/clear it and closures on
  close/Shutdown, reset `ListState`, cancel pending focus, increment generation.
- `ui/list.go`: `ListState.Key` = tab ID, `Label` = bounded accessible row name,
  `Selected` = current filtered row index; `ui.List` builds visible rows plus
  overscan2. `ScrollIntoView(index)` is public and resolves during layout.
- `api_gen.go`/`ref.go`: `TextInput`, `OnTextKey`, `TextKey.Text()`, `Bind`,
  `Handle.Focus/CancelFocus`, `Composing`, `Focused`, `KeepFocus` are sufficient.
  Up/Down in the query updates selection and queues ScrollIntoView for the next
  view pass; keep query focus. Enter uses fresh `k.Text()` before resolving ID.
  Query changes retain the selected ID if still matched, otherwise choose the
  first match (initially prefer active ID); clamp arrows at list ends.
- `List.Submitted()` means Enter or double-click, not single-click. Use explicit
  full-row ButtonBase/Clicked+KeepFocus for single-click activation, and the same
  generation/ID action for query Enter and list Submitted. Defer a clicked action
  until row construction ends; never mutate/release tabs from a list builder.
  Arrow selection alone never changes the active document.

## Focus, composition and switching

- Use public `DialogBase` (`api_gen.go`, `base.go`, `scope.go`) rather than an
  extra editor focus API: its modal scope contains Tab/window shortcuts, and
  closing returns focus to the still-existing opener. Keep underlying current
  document mounted; the picker itself retains only origin ID and generation.
  This preserves exact rich/utility opener identity inside MyGo, without app
  knowledge of an editor's widget tree. No vendor or module contract change.
- Opening refuses shutdown/restore/native-modal state. Before any focus move,
  check source `MarkedRange`, rich `editor.Composer`, and visible Find query/
  replacement or GoToLine composition using existing navigation policy. Then
  `flushTab(current, "打开已打开文档")`; this catches first rejected preedit with
  no mark and rich pending drafts. Failure keeps draft/view/focus and reports why;
  no queued opening. Revalidate current origin after this synchronous Flush.
  Filtering thereafter never flushes, edits text/undo, changes order or counts,
  reads disk, mounts an editor, or starts background work.
- Query composition follows `search_navigation.go`/`source_navigation.go`:
  ObserveInput Compose/Text plus actual Focused/Composing reconciliation after
  a native pointer leave. IME owns Enter/arrows/Escape while composing. Do not
  register an ancestor Escape that preempts it. Row/cancel/backdrop pointer
  controls KeepFocus; reject activation/dismissal with a message while preedit
  remains. DialogBase's automatic Dismissed is vetoable by restoring open=true
  before the next frame; after explicit IME commit/cancel, retry works. Opening
  and cancellation never send Escape to the source editor or discard its input.
- On activation resolve selected ID fresh, capture origin identity only for the
  synchronous call, then use `selectChecked(index, check)` (the existing Select
  path). Its callback must verify picker generation, current origin and exact
  destination ID still at that index after Flush. Reentry/removal/reordering
  refuses visibly; do not retry another slot. Also check same-tab activation:
  Select's same-index branch does not Flush, so call the barrier explicitly.
- On successful selection close/clear picker and keep existing Select behavior.
  On Flush failure dismiss picker without switching so the unchanged original
  draft can be corrected with its opener focus restored; retain the app error.
  Cancel similarly returns to the existing opener, without forcing a hidden
  source editor in preview mode. If origin/current identity changed externally,
  invalidate the picker; never enqueue focus for the previous tab. A stale row
  cannot reopen or create a tab. Async adoption is checked on every picker pass.

## Bounded implementation and evidence

Own `app/open_documents.go`, small `App`/view/shortcut/Shutdown wiring in
`app.go`, `view.go`, `workspace.go`, and focused `app/open_documents_test.go`.
No core/session/module/vendor/storage changes or persistent picker history.

Meaningful tests: Unicode/emoji/path/duplicate/untitled/badge/bounds filtering;
100 mixed tabs with virtual row count and final-row arrow reveal; real query
typing+Enter, single pointer click, Escape and one shortcut open; source/rich/
utility composition and first refused preedit keep draft/focus; query preedit
survives blocked click/backdrop/Escape then commit/cancel retry; failed rich Flush
restores exact field; same-tab barrier, synchronous Flush reentry/removal and
async active-tab change cannot select a stale slot. Assert byte/history/order/
tab-count invariants and no new inactive view creation. One optional760×480
Tester capture only if geometry needs proof; later one affected app test/vet.

No public API blocker found. Dialog dismissal veto + KeepFocus and exact opener
restoration on failed Flush need the targeted Tester evidence above; they are
implementation checks, not grounds for a toolkit fork or broad focus redesign.
