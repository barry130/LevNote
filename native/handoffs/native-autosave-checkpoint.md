# Paint-independent autosave checkpoint

2026-11-10. Implementation in progress; no Go check or product process run yet.

`app/autosave.go` now owns one cancellable one-shot request. Real scheduling
requires a nonnil Window and captures its public Update method on the UI thread.
The timer callback only posts that captured callback; it does not inspect App.
Existing uiDispatch is deliberately ignored because several headless tests use
an inline function. Tests opt into a complete fake timer/queued dispatcher pair.

One mutex protects only the timer slot, deadline and request identity across
Shutdown cancellation. Tab/document/modal/restore/deadline checks stay on UI.
Duplicate scheduling at the same lastEdit+900ms deadline coalesces, including a
timer whose UI callback is queued. A later edit cancels and invalidates the old
request. Non-edit mutations retain lastEdit and its remaining delay.

All production pending=true mutations now pass through markPersistPending;
source edits use noteEdit. View no longer drives recovery. Modal exit rearms
after deferred result delivery. Restore invalidates timers before its full
transaction and retains its final pending=false decision. Explicit persistence
consumes/cancels the current request, including failure; a later mutation or
explicit persistence may retry. Shutdown cancels before joining owned work.
Automatic capture calls neither Flush nor composition commit.

Runtime ownership: app/autosave.go, app/app.go, app/state.go, app/view.go,
app/workspace.go, app/export_worker.go, app/binary_actions.go, app/outline.go,
app/plugins.go, app/source_navigation.go. New substantive tests will be only in
app/autosave_test.go. The image owner's app/binary_keyboard_test.go is untouched.

Next: deterministic queued-dispatch tests with actual session bytes, including
no-View text capture, stale/early callbacks, nested modal exit, Restore/Shutdown,
failure suppression, explicit close supersession, and binary path-only metadata.
Focused debugging checks allowed; one combined app/imageview test and vet remains
held until both owners freeze. No package/host/worker/resource/capture execution.

## Source freeze after focused verification

Implementation and all eight new TestRecoveryAutosave tests are now frozen.
The first focused run passed (app1.971s, command exit0,2026-11-10T06:03:06Z).
Formatting also exited0. Exact command/output/result files are under
native-autosave-evidence-20261110 as01-format and02-focused. No failed Go check.
Tests additionally confirm that source preedit is not unmarked/committed and
that an inline uiDispatch never enables a real background autosave timer.

source-freeze.json records all11 owned source identities. Autosave implementation
SHA256 EF68DDEA9ECE3C2B118DB9579065F73672FF1354E8CF577302A63B9732612B40;
autosave tests DD0DDEEB176149E7471AEF6B485AF04AA92745098BB4975639180ED2D4A54F75.
Planned combined commands remain held: test ./app ./modules/imageview -count=1,
then vet ./app ./modules/imageview, both through scripts/go.ps1. Awaiting root's
confirmation that the separately owned keyboard source is also frozen.
