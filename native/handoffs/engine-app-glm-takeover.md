# App engine packages — GLM takeover and completion

Root current source read (continuation): document import and captured Shutdown
job are already present; ignore any stale message requesting those fixes.
Actual remaining draft-test issues at this read: transition deliberately sets
engineLane nil, so do not dereference it to test lazy ownership; readonly refresh
must activate the readonly tab after appendTab and inject/wait on a NEW managed
runner, not the closed old f; currently the test edits readonly tab rather than
the writable pending draft; stale generation must suppress stale receipts after
modal unwind. Please finish against actual APIs, not these temporary drafts.
Use meaningful pending view draft/unchanged undo evidence. DeepSeek performs
source-only independent review; no ownership conflict and no need wait for it.

2026-11-09. Astra astra_marker_view failed with model capacity, no longer owns
implementation. You are the sole application implementation owner. Root plans/
reads/accepts only. Read this once and current app sources; implement and finish
without waiting for another plan. No child delegation; compact before200K.

CURRENT: store, managed runtime and scene refresh are accepted; no edit needed.
Actual runtime final3cases PASS1.970s/vet0 under artifacts/acceptance/fixed-engine-
package-runtime-20261109-151125-followup04. .8 runnable release stays immutable.
New app source exists and compiles: app.go, engine_packages.go/UI, preview_worker,
export_worker, view, ui/editor/scene.go metadata. New engine_packages_app_test.go
only covers narrow poison refusal and hasn't been accepted. .9 not built.

OWN: app production, app/engine_packages_app_test.go plus a new
app/engine_packages_lifecycle_test.go and optional captures test file; narrowly
required neutral editor/scene status copy. DON'T edit store/runner/worker/wire,
managed_bundle_test.go, portable README or peer test files. DeepSeek owns ONLY
new app/engine_packages_store_test.go (real-store fixture helper + import/list/
activation/reset/repair tests), and cannot edit app production. Coordinate via
handoffs and ask root for a demonstrated production issue outside your scope.

Product requirements already in fixed-engine-package-app-activation.md and
app-finish-sequence.md; use actual source rather than replan. Root source review
already corrected two false findings: RefreshScene preserves selected node,
list/import can operate while enginePoison blocks execution. Existing transition
guard/ALL3 disabled is correct. DO NOT block import/list or rework graph refresh.

Finish meaningful integration tests for captured old lane cancel/reservation,
running and ready/reserved export transition, readonly refresh+writable pending
draft/undo unchanged, persistence refusal/corrupt state (peer store test covers
basics), cleanup poison retained, stale modal/generation, shutdown join without
UI dispatcher, close refusal doesn't cancel package work. Reuse controlled
runner helpers in export_worker_test/preview_worker_test. Prefer5-7 focused
tests covering related paths; no giant matrix or fake implementation-copy tests.
Private narrow test seam permitted only where real failure cannot be induced.

Run the relevant focused app tests with exact arguments, real raw stdout/stderr/
exit and small before-after scope under fresh artifacts/acceptance. First make
source compile, then test behavior. Preserve failures. When focused checks pass,
run complete ./app once and vet ./app ./ui/editor ./modules/graphview. No full
repository/process/resource replay. Peer tests must be frozen first; poll its
handoff or message to coordinate, don't create compile races.

Capture actual MyGo tester760x480 engine panel light/dark with installed package,
blocked state and pending operation while panel hidden. Use inert fixture package
and controlled runner; no physical app launch now. Root inspects PNG paths.
Verify scroll reaches activate controls and final status; no clipped essential
actions. Keyboard activation should use current ui tester API, no Web UI.

Publish handoffs/fixed-engine-package-app-result.md with exact tests/images and
remaining limitations; include any actual production changes. Then prepare
the already queued fixed-engine-package-delivery-plan.md succinctly; .9 builds
wait for root's result/image review. Keep source moving, don't spend long writing
reports or rereading the same task. User needs a usable tested new release.
