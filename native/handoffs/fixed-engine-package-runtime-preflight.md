# Fixed-engine package integration preflight

2026-11-09. Read-only preparation under the queued runtime plan; no Go/test/script
edits or execution. Core checkpoint signatures are the proposed dependency, not
accepted implementation yet. No API objection; its private receipt verifier and
atomic selection-read seam must be checked when the actual source lands.

**Constructor and request selection.** Keep `NewBundled()` and existing test
composition. Add `NewManaged(manager *PackageManager) (*Runner, error)`; it
accepts no executable/root override and does no package enumeration/hashing.
Production constructs the manager once with `filepath.Dir(os.Executable())`
(absolute normalized) and the existing app config dir. Startup reads bounded
selection state only, never `Installed`. Constructor/load errors remain visible;
there is no fallback to NewBundled if manager creation/state resolution fails.
Within `Runner.Export` admission, obtain one private immutable selection under
the manager's state lock: `{engine, source, digest, private roots, loadErr}`.
Read selection and blocked status atomically, not separate public Snapshot and
LoadError calls. Copy values; do not retain a mutable status pointer. Later
Activate/Reset cannot retarget this request. Public Snapshot is for cached UI.
`run` derives the fixed path from that captured selection and verifies the full
installed receipt/manifest using held directory/file handles through launch and
Wait; never verify, close, then reopen an unchecked caller path. Keep version
opaque, executable SHA256 separate from package digest; add source/digest to
verified EngineInfo and corresponding UI metadata without changing wire format.

**Small app transition.** Reuse the single engineLane and one bounded package
operation/result slot. Import/list/hash run in background with immutable inputs;
View displays a cache and drains completion, not filesystem enumeration. An
import is not activation. Activate/Reset/Repair capture manager/action/engine/
digest identity after the directory dialog, block new jobs, increment a service
generation and invalidate/cancel old result owners on UI. Capture the old lane,
runner and job done handles; a worker joins only these objects, then persists
selection. Store completion + nonblocking Invalidate; it never reads tabs/views.
On UI, matching completion creates a fresh managed lane lazily. Persistence
failure recreates from the unchanged valid selection; corrupt state stays
blocked. Cleanup failure is permanent service poison and remains reported even
if the operation became stale; a new lane must not erase it. All-selection
repair is explicit and retains installed packages. No version ordering.

**Concrete current seams and scene refresh.** `beginScene`'s cancel closure and
`cancelPreview` currently call dynamic `a.engineLane`; store the owning lane in
previewJob and cancel that exact lane. `deliverEngineExport`'s deferred
`releaseExportReservation` likewise needs the job's captured lane/generation;
an old modal delivery cannot release a new reservation. `sceneView.ensureScene`
short-circuits at unchanged Buffer/version, so add the neutral optional
`editor.SceneRefresher interface { RefreshScene() }`. Only readonly scene views
implement it: cancel the old callback, increment epoch, clear started/error/
warning/cache state, retain scene as explicitly stale plus zoom/pan/selection,
and clear pending navigation; next normal Draw/Warnings starts a new request.
Call it on UI after a completed transition (including persistence refusal with
old state), only for still-owned readonly SceneEngine views. Do not call
configureTab or Release on writable views; cell/node/tag drafts, source,
selection and undo remain intact. Scene apply already drops a selectedID absent
from the new scene. Switching status remains visible while old work joins.

**Close races that matter.** `closeEngineExport` mutates UI fields and cannot be
moved wholesale to a goroutine. Keep UI invalidation separate from captured
lane/job joins. Existing CanClose first finishes all prepareClose decisions and
persists discard-aware recovery: a refused save/close must not cancel unchanged
engine/package work. Only after those succeed cancel/join the package operation
and lane; cleanup refusal must retain tabs and restore live recovery as today.
Shutdown marks shuttingDown before cancellation and joins both barriers without
UI dispatch; repeat calls wait the same completion. A store operation that
already committed may retain its durable selection—cancellation after commit is
not rollback—but its late result cannot recreate a lane/view after shutdown.
Serialize the persistence step and terminal state so cancel-before-commit and
commit-before-cancel are explicit; no UI callback is required to close `done`.

**Finite implementation/checks after activation.** Runtime runner/run and narrow
bundle/store-private seam; app package panel/operation, preview/export/lane/
shutdown wiring; additive editor refresh and sceneView implementation. Reuse
core filesystem fixtures for selected digest/refusal/snapshot (no imported
execution), current controlled runners for queued/running/reserved transitions,
old-lane cancel/release, persistence refusal, cleanup poison and shutdown. Add
unchanged-version scene refresh/position and writable pending-draft preservation,
modal owner replacement, import-only/cached-idle panel, bounded result ownership,
and 760×480 light/dark captures. Run affected runtime checks, complete app once
and scoped vet after source acceptance; no protocol/resource replay or package
build at this preparation gate. Any required controlled child case must be
justified separately under the root plan.
