# App list handle lifetime: bounded follow-up recommendation

2026-11-10 Astra, read-only source trace. No implementation, tests, process launch,
measurement or modification of frozen picker/runtime inputs. Future scope is
only app TreeList, SearchList and OutlineList.

The concern is real but the triggers are narrower than the initial task wording:

| List | Actual trigger | What does not allocate another bound identity |
|---|---|---|
|TreeList|app/app.go:909 resetTreeIdentity replaces ListState only when treeRoot differs from requested root.|Same-root watcher refresh, toolbar refresh, directory expansion and applyTreeScan preserve it.|
|SearchList|app/workspace.go:81 searchWorkspace replaces ListState at98 after valid root/query admission, for every new search.|Each result batch only appends data; collapse at156 only hides results. There is no production clearSearchResults helper.|
|OutlineList|app/outline.go:16 show transition, :33 releaseOutline, and ensureOutlineSnapshot error/stale branches at165/184/193 replace state.|A successful same-controller/version reparse updates items and selects/reveals by ID without replacing ListState.|

Tree flow: workspace.go:45 watcher UI delivery calls refreshTree; app.go:889
increments request generation and invokes resetTreeIdentity; workers.go:292
applyTreeScan checks root/generation then replaces Entries only. Changing roots
therefore causes fresh bindings, ordinary watched-folder updates do not.

Search flow: state.go:449 public Search Workspace button calls searchWorkspace;
it cancels earlier work, increments searchID, clears results/selection and state,
requests scroll to0, then receives batches through a.update with ID checks.
Only a search that eventually renders nonempty results binds that fresh handle.

Outline release also occurs on presentation/tab/mode change through
app.go:1099 releaseTabPresentation, on controller identity replacement through
outline.go:69 outlineSourceFor, hide/retry, and Shutdown. Repeated unsupported
error frames may assign fresh zero state but bind no list: this alone does not
grow engine.refs. Bind requires a later nonempty displayed list.

Pinned public toolkit is github.com/egoist/mygo v0.8.3 (native/go.mod:5), using
the existing vendored patches. Actual vendor/github.com/egoist/mygo/ui sources:

- ref.go:34 Handle.binding creates one refState per handle/window, appends it to
  engine.refs at53, and reuses it when the same handle binds again in that window.
- list.go:518–523 embeds a per-pass frame in ListState and automatically binds
  its Handle. Ref/Handle have Focus, CancelFocus, Focused, FocusWithin and Bounds;
  there is no public handle-disposal operation. CancelFocus only clears requests.
- ref.go:181 applyFocusRequests scans all refs each build pass; runtime.go:1229
  marks them closed and clears refs on window close. Abandoning a handle does
  not remove its already-created engine reference during that window lifetime.
- frame.go:66 Element is a checked owner/slot/generation handle, not a retained
  old node or row closure. Each pass expires it. Old ref records establish small
  reference bookkeeping growth, not evidence of accumulated document snapshots,
  heap/RSS growth magnitude or a measured performance regression.

Small future implementation: preserve exactly one public Handle per app-owned
list/window through semantic resets. Capture its Handle, CancelFocus, replace
the ListState value with only that Handle, then restore only intentional new
selection/scroll requests. This clears Key/Label/Selected and private list frame,
row/height/selection/scroll internals through a public zero value, without private
field access or retaining whole old ListState. Keep current tree root generation,
searchID checks and outline controller/version/ID authority unchanged. No new
global cache, vendor edit or per-row handles. Public handle equality after first
binding is sufficient to test identity reuse.

Do not blindly perform every full reset at the call site. list.go:416 listFrame
holds *ListState and row closure; node.list points directly to ListState.frame.
buildLate at805 invokes that frame again during layout, after App.View returns
(runtime.go:768). Replacing ListState after its list has already been built can
zero the still-referenced frame. App.View builds sidebar before tabs/main
(view.go:75); a subsequent tab switch/search-result activation can releaseOutline
after that outline was already constructed. Outline submission can itself
reenter Flush and release its controller. Those are concrete routes to cover;
this trace did not execute or claim a reproduced panic.

Use safe pre-build reset boundaries: tree root selection from toolbar and search
start from Find occur before their respective lists; outline snapshot error
occurs before building its list. For release/hide after a list was built, cancel
pending focus immediately and defer only the full ListState reset to the next
App.View start, before any lists are constructed. A small per-list pending-reset
flag/bit is sufficient if needed; no generic scheduler. Retain existing captured
bounded row metadata only through that current pass. Consume a pending reset
before building a new list and before any deliberate new Focus/ScrollIntoView
request, so the reset does not erase valid new navigation. Shutdown is after
window close in cmd/levnote/main.go:28; clear state/callbacks and cancel requests
there. Collapsing a still-live result panel need only cancel pending focus unless
existing UX deliberately calls for a semantic reset; do not silently remove its
current result/scroll behavior merely to change handle lifetime.

Suggested narrow ownership: app/app.go resetTreeIdentity; workspace.go search
start/collapse/Shutdown; outline.go resets; view.go only a safe pre-build reset
drain/tree-hide cancellation if needed; a focused app/list_handle_lifetime_test.go.
Leave picker, workers' scheduling, core/modules and vendor unchanged. Any tiny
helper should only preserve/reset these app-owned states at the safe boundary.

Meaningful focused public Tester regressions, after separately activated source work:

1. Tree: repeatedly choose two real temporary roots through the folder entry
   with the existing dialog seam; compare Handle after rendered nonempty roots.
   Verify stale root selection does not survive, latest root End/Enter activates
   the right path, and same-root expansion/watch-result refresh retains anchor.
   Reuse real scanner/adoption seams; no live OS watcher or product host needed.
2. Search: use the public Find/Search Workspace route with real temp files and a
   deterministic queued UI dispatcher; drain callbacks on test/UI owner. Repeat
   query replacement/nonempty→empty→nonempty, stale prior searchID delivery and
   collapse/reopen; assert same Handle, reset selection/scroll on new search,
   far-row End/Enter reveals the current result, no old callback/navigation.
3. Outline: public file/outline toggle, actual tab/mode switch after sidebar
   construction, failure/retry and refreshed headings. Compare Handle through
   releases; assert old controller release once, fresh stable item ID navigation,
   no stale selection/callback, no late-layout failure and unchanged document
   bytes/history when only navigating. Existing outline stub enables exact
   failed-Flush/reentrant release checks alongside a real Markdown route.
4. Queue public Focus on a hidden/released list, cancel through production hide/
   reset, reopen and type into the actual intended editor/query to prove no
   focus theft; separately request deliberate focus after rebuilding and verify
   keyboard navigation works. Compare public handles, not private engine refs.

Use existing TestWorkspaceTreeVirtualizesRowsAndKeepsExpansionAnchor,
TestWorkspaceSearchVirtualizesFarResultAndReplacement and outline virtual-list/
Flush lifecycle tests as integration patterns, avoiding broad duplicate suites.
No forced GC, internal ref count inspection or whole-process memory assertion.
Markdown module v.list replacement on Release is a separate future scope only;
this recommendation does not authorize a module/editor lifecycle audit or fix.
