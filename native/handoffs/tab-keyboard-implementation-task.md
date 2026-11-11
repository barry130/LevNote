# Visual-order tab keyboard navigation

ACTIVE2026-11-10 after .21 package/evidence freeze acceptance. DeepSeek app owner.
The plan
`tab-keyboard-next-plan.md` is accepted as a narrow implementation direction.
No child delegation. GLM/DeepSeek compact before200K. Root plans/reviews only.

Implement two exact shortcut bindings Ctrl+Tab / Ctrl+Shift+Tab and a small
adjacent-tab helper using existing Select lifecycle. Wrap in visible tab order;
zero/one tabs no-op before Select, preserving panels, drafts and focus. Do not
add MRU, persistent history, UI popup/animation/dependency or generic Select
refactor. Keep Find/GoToLine/composition/failed-Flush semantics from the plan.
Offscreen tab auto-reveal follows as a second small increment using only public
APIs; `tab-workflow-stage-plan.md` allows one combined final affected suite and
one later package rather than a package for each increment.

Ownership after activation: native/app/app.go, native/app/view.go, one focused
app test file. Product README update is reserved to docs owner if .21 docs are
still running; otherwise add only the planned shortcut operations-table row.
No package/build/launch, unrelated format/runtime/source changes.

Use substantive tests for no-op, both directions/wrap, failed Flush preserving
active view/draft, existing success cancellation/release/persistence behavior,
and a synthetic Tester key routing case. One final affected ./app test/vet via
native/scripts/go.ps1 -GoArguments after focused corrections. Preserve actual
commands/logs/exits and failures. No physical OS/IME claim.

Publish tab-keyboard-implementation-result.md with final file identities and
validation evidence; freeze for root review. Report any concrete production
reentry issue before expanding this small slice.
