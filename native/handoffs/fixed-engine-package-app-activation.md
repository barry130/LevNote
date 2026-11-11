# Local engine packages in native app — active after runtime focused PASS

2026-11-09. Astra is sole app/runtime owner. Root read the runtime source and
accepted its API. Continue here immediately after the finite runtime gate passes
and its actual result is published; no additional idle approval checkpoint.
Reuse accepted store/runner/SceneRefresher and the existing single engine lane.
No worker/catalog/wire additions, executable-path settings or resource claim.
Root plans/reviews; all source/test/capture work belongs to this child.

## Product flow

Extend the existing native Plugins panel with two clear choices: format
configuration and engine packages. Avoid another permanent toolbar. Engine
packages shows one of the3fixed engines, current bundled/installed selection,
retained verified versions with short digest, import directory, refresh list,
activate exact package and restore bundled. Display concise Chinese disclosure:
import copies a local executable; activation permits this app to run that engine
for its stated preview/export operation; only import a release the user trusts.
The JSON format configuration remains nonexecuting; don't conflate the two.

Import and activation are separate. Never implicitly activate the first/newest
version or sort opaque versions as semver. No delete/update/network feature.
Corrupt selection state exposes a clearly labeled reset ALL THREE selections
action retaining packages, never repurposes normal single-engine reset. Show
healthy receipts alongside any list error. Busy/cancel/outcome stays visible;
absence of a selection verification is not presented as verified. 760x480
content must scroll; wrap copy rather than clipping essential actions/status.
Leave readonly source/SaveAs semantics intact.

## Composition and asynchronous work

Production host dir is resolved once from os.Executable; existing configDir owns
the private store. Failed initialization/state resolution is visible, no silent
fallback to NewBundled. Default startup reads only bounded selection state;
no Installed scan. Keep existing injected controlled-runner test seam private.

All import/list/hash/activation/reset work is background work with captured
immutable input and a bounded result slot, done barrier and nonblocking window
Invalidate. No UI dispatcher is required for completion/join. One package
operation at a time; no goroutine per frame or panel open repeated scan. UI
consumes cached snapshots and receipts. Closing the panel may hide it while the
operation remains visible/reopenable; cancellation must join before replacement.
Directory-dialog callbacks can pump UI: capture/recheck manager+engine+action
identity and shutdown after return. Do not inspect tabs/views from workers.

## Switching and shutdown

Begin a selection transition on UI: block new engine requests, advance engine
generation, invalidate preview/export results, capture/cancel the OLD lane and
jobs. Background joins those captured owners BEFORE attempting persistence.
Cleanup failure becomes persistent application service poison; never clear it
by constructing a new lane. Finish work into its bounded slot even if cancelled.
On matching completion UI takes actual manager snapshot/load error and permits
fresh lazy managed runner only if healthy. A precommit failure retains selection;
postcommit FailureCleanup may accompany a committed new selection. Report the
actual state, never infer rollback solely from err != nil.

Bind preview cancellation and export reservation release to their original
lane and generation. A stale completion/modal unwind must not cancel or release
the new lane or overwrite current selection/outcome. Refresh only still-owned
readonly scene views through SceneRefresher after a joined transition (including
old-selection retention on persistence refusal). Don't configure/release writable
views, flush their drafts or reset their undo history merely to change an engine.
They are host-native editable models; a package selection does not change them.

CanClose still gathers Save/Discard/Cancel decisions and persists recovery
before cancelling work. Failed save/persistence leaves unchanged package/engine
work alive. Accepted close/shutdown cancels and joins package work and both old/
current captured lane barriers without UI dispatch. Existing cleanup-refusal
recovery restoration and repeat shutdown idempotence stay intact. No late
completion may resurrect a closed app. Cancellation after commit is not rollback.

## Verification and deliverable

Publish source checkpoint after API/operation ownership is concrete. Implement
finite behavior tests with real store fixtures + controlled runners: import-only,
activate/reset/repair, failure and actual selection status, cached idle panel,
running/queued/reserved transition, old-lane cancellation/release, late results,
poison and shutdown/no dispatch, modal owner changes, readonly refresh and a
writable pending-draft+undo preservation case. Reuse existing tests; no giant
matrix of every package/protocol/resource combination. Give test seams a narrow
private boundary; no public unchecked resolver.

Run focused new app tests while developing; after final source fix run full app
once and scoped vet of changed packages. Reuse accepted unchanged runtime/core
results. Capture native 760x480 light/dark engine panel incl an installed package
and a meaningful error/pending state; root will inspect actual images. Do not
claim these are physical IME/OS focus tests. Preserve failed runs/raw streams.
Final source result: fixed-engine-package-app-result.md. One fresh release and
narrow actual local import/activation smoke follow root source/UI acceptance;
the .8/.7 packages remain immutable throughout this source slice.
