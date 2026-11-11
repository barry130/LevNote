# Next bounded resource review: app list handle lifetime

ACTIVE read-only planning2026-11-10 while picker final review runs. Owner Astra
/root/read_native_handoffs_tab_workflow_resume_task_md. No children, source edits,
tests, builds, launch or measurement. Final picker validation takes precedence
when root activates it. Save this small review and stop; do not implement yet.

Root found a related concrete source concern while reviewing picker: public
MyGo ui/ref.go Handle.binding appends refState to engine.refs; there is no public
disposal, and applyFocusRequests scans those references. ui.List binds its
ListState.Handle automatically. App repeatedly assigns fresh ListState in:
- app/app.go refresh-tree result around line915 (TreeList)
- app/workspace.go clearSearchResults around line98 (SearchList)
- app/outline.go show/release/error/reparse paths (OutlineList)

Review whether ordinary watched-folder refresh, repeated workspace search and
outline changes create a new bound handle each time. Read actual production
flow and pinned public toolkit implementation; distinguish growing small
reference bookkeeping from retained documents or measured heap/RSS. Picker has
just been fixed to retain exactly two opaque handles; don't modify it again.

Recommend the smallest follow-up limited to the three app-owned list identities:
reuse one public Handle per list/window through semantic ListState resets,
CancelFocus when hidden/released, clear callbacks/selection/scroll internals,
preserve fresh row generation/identity and valid deliberate focus. Check late
row callbacks before recommending direct value replacement. No vendor/private
API or new global cache. Do not expand into module/editor handle lifecycle in
this stage; note that broader scope separately if found.

Write native-list-handle-lifetime-next-review.md with exact paths/trigger/API
reasoning, minimal ownership/test strategy, and risks. Meaningful tests should
use actual app public Tester flows and compare public handle identity across
repeated rebuild/release/reopen, verify no stale selection/navigation and no
wrong focus. No private refs introspection, no artificial forced GC or whole-app
memory claim. Current `.30` runtime stays frozen; this is future planning only.
