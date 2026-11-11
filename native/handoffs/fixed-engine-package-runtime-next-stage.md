# Next runtime/app stage — queued pending core acceptance

2026-11-09. Planning by root after source inspection; NOT active implementation.
GLM currently owns the package store. Astra completes .8 source docs first.
Use actual accepted core APIs, not the older conceptual design's signatures.
Accepted .8/.7 directories and existing evidence remain immutable.

## Outcome and ownership

Let the user import one of the three fixed existing engines from a local
complete release, inspect retained packages and activate an exact digest or
restore bundled. Root will activate finite runtime then app slices after core
source/tests acceptance. No new engine, generic executable path, wire format,
network, automatic update, editable worker or resource claim in this stage.

Runtime should consume the manager, with a constructor limited to its private
validated roots, and capture immutable selection per request. Reject blocked
state/invalid selected package; never silently select bundled. All installed
files must pass the accepted receipt/manifest verification while held through
launch. A private adapter may reuse existing verify/locked handle helpers; do
not validate then drop handles and reopen via a caller-supplied path. HostDir
comes from os.Executable at production composition. NewBundled remains valid
for current callers. No selection mutation during a request retargets it.
Keep engine ID, opaque version, executable hash and package digest distinct.

## UI and lifecycle constraints found in current source

Existing ensureEngineLane is lazy, returning a lane failure or NewBundled.
App owns one exporter and lane; lane close joins without UI dispatch. Export
publication is reserved until its UI delivery; preview keeps only latest. New
package transitions must join these same boundaries, not create a parallel lane.

All potentially large disk hashing/import/list verification runs off UI using
captured immutable input and a bounded result slot + Window.Invalidate. Do not
list/re-hash packages during every frame. Show one operation pending and allow
cancel. Opening/closing the panel must not block rendering. Startup should not
enumerate all installed bytes; bounded selection state only. Filesystem or
corrupt-selection errors are visible in this panel and in failed operations.

UI flow: choose one fixed engine, import from directory, show receipt source/
version/short digest and retained packages; activation is a distinct button with
plain execution disclosure. Restore bundled is separate. Corrupt entire state
offers clearly labeled reset ALL3 selections, preserving installed packages.
No implicit reset and no arbitrary path/editor control in the panel. Keep the
small window scrollable, text wrapped, button actions keyboard accessible.

Switching starts on UI: capture engine/action/manager identity, block new jobs,
increment engine generation, invalidate preview/export result owners, capture
old lane/runner and cancel. Background closes/joins old work BEFORE activation/
reset persistence. A failed cleanup poisons the application engine service and
must not be erased by making a new lane. A persistence error leaves old state;
once joined, recreate from that old valid state. Successful imported selection
affects subsequent requests only. Late background callbacks never read tabs,
buffers or views and never resurrect an app that is shutting down.

Specific current-code hazards to address:
- beginScene returns a cancel closure that calls a.engineLane dynamically.
  Capture the owning lane, so releasing an old scene can't cancel on a new lane.
- deliverEngineExport's deferred releaseExportReservation uses current lane.
  Old delivery must not release a new reservation; capture identity/generation.
- closeEngineExport reads/mutates UI-owned fields, so don't run it wholesale in
  a background goroutine. UI invalidates/captures, worker joins only captured
  objects, UI consumes completion. Shutdown joins pending store/switch work;
  no callback may hang waiting for UI dispatch after shutdown.
- Existing sceneView started flag prevents retries at unchanged Buffer version.
  After transition explicitly invalidate/refresh readonly scene views only,
  preserving view position where feasible. Do not reconfigure writable format
  views, reset undo or discard pending cell/node/tag drafts.
- CanClose modal persistence has specific discard-recovery semantics. Pending
  package work must not bypass its refusal/cancellation or revive state later.
- Opening dialogs can pump UI callbacks. Revalidate action owner after modal;
  cancel/changed owner leaves selection and files unchanged.

## Proportional validation

First runtime: filesystem fixture tests of selection/refusal/snapshot and
focused existing bundle tests; use private seams only. Do not run imported
arbitrary code, rebuild packages or replay the full process resource matrix.
If a real process seam needs checking, use the existing controlled worker test
helper in a fresh fixture and justify the one relevant case.

Then app: controlled runners and real package-store filesystem fixtures for
import-only, activate/reset, failed persistence, corrupt selection and repair,
switch while queued/running/reserved, cleanup poison, late results, shutdown,
owner change around modal, draft/undo preservation, immutable input state and
cached panel rendering. Native 760x480 light/dark workflow capture; not physical
IME. Full app test once after changes and scoped vet, affected runtime tests.
Preserve actual streams/exits/failures. Only after source/UI acceptance prepare
one fresh candidate package and narrow actual import/activation smoke. Reuse
existing packet/format/worker coverage instead of amplifying validator count.
