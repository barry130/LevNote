# Fixed engine app source review — findings

2026-11-09. Read-only app source inspection against
`fixed-engine-package-app-activation.md`. Sources read: `native/app/app.go`,
`native/app/view.go`, `native/app/engine_packages.go`,
`native/app/engine_packages_ui.go`, `native/app/preview_worker.go`,
`native/app/export_worker.go`, `native/ui/editor/scene.go`, and
`native/modules/graphview/scene_preview.go`. No tests or builds were run.

## Findings

### 1. Package actions still start after persistent engine poison

**File/function:** `native/app/engine_packages.go`, `startPackageWork`

**Trigger:** An old lane or runner reports `FailureCleanup`; delivery stores it
in `a.enginePoison`. Import, list, activate, reset, and repair buttons can still
start a `packageJob`. `packageJob.run` captures the poison, sets
`err = "旧引擎清理失败，选择未切换"`, and skips the actual manager call.

**Expected:** New package work must not start when the service is permanently
poisoned. The request should be refused at the UI entry point, and the user
should see one durable recovery path rather than a queued action guaranteed to
fail.

**Minimal correction:** In `startPackageWork`, refuse non-repair actions when
`a.enginePoison != nil`, with the same visible message used for the poisoned
lane. Disable their buttons when poison is set. Keep repair reachable only if
it is deliberately implemented to run without the failed runner, which the
current implementation does not do.

### 2. “重置全部三个选择” is a dead-end while poisoned

**File/function:** `native/app/engine_packages_ui.go`,
`enginePackagePanel`; `native/app/engine_packages.go`, `packageJob.run`

**Trigger:** After a lane cleanup failure, the repair button remains enabled
because its `Disabled` condition checks only `busy`. Clicking it starts
`packageRepairAction`, but `run` sees the captured poison, records
“选择未切换”, and returns without calling `RepairResetAll`.

**Expected:** The repair action should either be able to clear the relevant
selection state independently of the failed runner, or be disabled while that
recovery is impossible. A permanently enabled action that can only fail is
misleading and provides no repair path.

**Minimal correction:** Disable repair when `a.enginePoison != nil` unless its
worker path is changed to perform the store repair before consulting poison.
The latter would need an explicit contract review because “service poison”
currently gates every package action.

### 3. Engine transition clears a user’s selected preview node

**File/function:** `native/modules/graphview/scene_preview.go`,
`(*sceneView).RefreshScene`; called by `native/app/engine_packages.go`,
`deliverPackageWork`

**Trigger:** A user selects a node in a readonly mindmap preview, then activates
or resets an engine. `deliverPackageWork` calls `RefreshScene` for every
readonly scene tab. `RefreshScene` preserves the stale scene but clears
`selectedID` unless that exact node remains in the old index. For a still-owned
tab whose source did not change, this drops the user’s selection solely because
the engine selection changed.

**Expected:** A package transition may briefly mark the preview stale, but it
should preserve the selected node and viewport when the buffer identity and
version are unchanged. The old request must be invalidated; the UI selection
need not be.

**Minimal correction:** Do not clear `selectedID` in `RefreshScene` merely for
the epoch change. Retain it, then validate it against the replacement scene
after the next result; clear it only when the node is actually absent or the
buffer/version changed.

## No other concrete defects

The remaining focus areas look correctly ordered in source: manager creation
uses `os.Executable` and `configDir` once at startup; package state is read
before any list scan; one bounded package result slot is used with a done
barrier; transitions capture the old lane/runner/jobs and advance generations;
old owners are joined before persistence; postcommit cleanup remains visible;
managed preview/export runners are used on later requests; bundled fallback is
explicit in status text; SceneRefresher is limited to readonly scene views;
CanClose preserves its existing save/persistence refusal flow; and the engine
panel uses scrollable containers with distinct import, activate, reset, and
repair controls. No tests/builds were run, so this is source inspection rather
than verification.
