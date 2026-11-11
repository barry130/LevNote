# Closed-tab backing array retention: narrow verification and correction

2026-11-09. DeepSeek read_native_handoffs_resource_workload_next_task_md now owns
app/app.go removeTab and one new focused app test file for this finite issue.
Other DeepSeek owns core/text retention; GLM owns7product docs. No shared writes,
no child agents, compact before200K. Use scripts/go.ps1 -GoArguments for tests.
Accepted .10/older packages are immutable, no build yet.

Root source review found removeTab currently uses
`a.Tabs = append(a.Tabs[:index], a.Tabs[index+1:]...)` without clearing the removed
tail pointer. The app keeps the backing array alive after shortening the slice,
potentially retaining a closed Tab -> Doc/Buffer/history/Editor graph until later
append or application exit. Last-tab removal is especially direct, and closing
all tabs preserves multiple old capacity slots. This is a concrete ownership
hypothesis to prove, not an OS memory measurement.

First add a focused meaningful regression using public CloseTab and ordinary
newTestApp/document fixtures; prove closed tabs are absent from the entire live
backing slice, not just len(Tabs). Cases last tab with a substantial edit/history,
middle removal preserving order/active, closing all, and close refusal preserving
all original pointers/content/session. Existing saved/dirty close decisions and
tests may be reused. No flaky finalizer/GC timing or RSS threshold. Run against
old code and preserve actual failure.

If confirmed, minimal authorized fix: shift remaining pointers with copy, clear
the old last pointer before slicing (or equivalent standard slices.Delete with
documented clearing); optionally nil when empty if justified. Keep all lifecycle
cancel/flush/dispose/watch/persistence/active selection behavior unchanged.
Don't nil fields on tabs still referenced by legitimate in-flight tasks or test
callers; remove only application's obsolete backing-array references.

After new regression green, run focused close/persistence/lifecycle tests and
app vet as relevant; no fullapp yet because core/text owner will do one combined
dependent fullapp after both changes stabilize. Publish
handoffs/tab-retention-fix-result.md with actual red/green raw evidence, final
hashes, proof scope and limitations. Send readiness to core/text owner/root;
no process launch/package build/docs edits. State that no UI resource comparison
is available because Computer Use helper is unavailable, not because this test
proves an application memory percentage.
