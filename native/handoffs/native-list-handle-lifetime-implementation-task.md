# Reuse the three app-owned list handles

PLANNED2026-11-10, implementation only after `.30` package/evidence closure and
explicit root activation. Owner Astra, no children. Root accepts the bounded
source finding in native-list-handle-lifetime-next-review.md. No whole-process
memory or performance improvement is established.

Scope: app TreeList, SearchList and OutlineList identities only. Own small
changes in app/app.go, workspace.go, outline.go, view.go as needed plus a focused
app/list_handle_lifetime_test.go. No picker change, module/editor lifecycle,
vendor change, global cache, worker scheduling rewrite or extra feature.

Preserve one public ui.Handle per list/window across semantic state resets;
CancelFocus on hide/release and clear callbacks/selection/row/scroll internals
at a safe boundary. Tree resets only when workspace root changes, search only
for an admitted new search, and outline on show/release/error; same-root watcher
refresh and successful same-controller outline reparse preserve current UX.
No new source retention or retained old full ListState merely to save its Handle.

The public List keeps a frame referencing ListState for late layout. A callback
after sidebar construction can release/rebuild outline in the same pass. Use
a small deferred-reset flag/boundary where required; consume it before next list
build and before deliberate new focus/reveal. Avoid resetting a live frame in
its row callback. Clear bounded row metadata/callback references after their
last live build and on Shutdown. Preserve generations/searchID/controller
revalidation, failed-Flush behavior, selection/scroll and deliberate focus.

Meaningful focused tests: real bound public handle equality through repeated
root/search/outline release+reopen; nonempty/empty/nonempty and error/retry;
stale prior result rejection; far row keyboard navigation to correct fresh ID;
same-root anchor preserved; actual sidebar-before-main release and late layout;
hidden pending focus cancelled without stealing intended source/query input.
Use existing public Tester, dialog seam, real temporary files and UI dispatch
queue patterns. No private ref counters, forced GC, sleeps or host processes.
Do not duplicate all prior suites; extend relevant existing behavior fixtures.

Save a progress checkpoint and narrow pre/post identities. One focused affected
union after implementation, preserving raw outputs/exits and any real failures.
Root then assigns exact-source independent review and one final app test/vet.
Do not build/package or execute those final checks preemptively. Handoff:
native-list-handle-lifetime-implementation-result.md plus fresh evidence directory.
