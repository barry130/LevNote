# Markdown worker app integration — focused checkpoint

Owner `/root/astra_marker_view`, activated after root accepted bottom-layer
checkpoint EAA53F06003BD871D442178F7781CB5AE2425269ED8C2E273E0BF1E7BFCBEB50.
Protocol/runtime/worker/package script remain frozen. Prior owners stay stopped.

Implementation complete; root accepted the app focused gate at01:32local.
Full actual integration passed and root accepted its exact aggregate; see
[markdown-worker-full-integration.md](markdown-worker-full-integration.md).
Optional `editor.Adapter.ExportEngine`
descriptor added; only Markdown advertises the fixed HTML engine. App routes
that capability even from source mode without a mounted View. Snapshot and size
are captured after Flush, with explicit foreign transaction and8MiB refusal.
No UI snapshot flattening or silent in-process fallback.

`app/export_worker.go` owns the injected private `exportRunner` interface, one
UI job identity, mutex-protected clearable result slot and identity-only queued
callback. Runtime joins before completion; shutdown cleanup never waits for UI
delivery. Completion rechecks captured Tab/Buffer/version/adapter/capability,
Flushes before and after the export SaveDialog and uses existing exclusive
synchronous `document.WriteNewFile` only after final guards.

Lifecycle invalidations run after successful tab switch/new tab/
recovery/removal, reload/reinterpret and adapter reconfiguration. Accepted
CanClose joins before disposing tabs; cleanup failure refuses close visibly and
rewrites recovery from the still-open live tabs, undoing prior discard-only quit
snapshot decisions. A second recovery write failure is independently reported
and leaves an explicit retry pending. FailureCleanup stays visible even when
the user previously canceled/switched tabs. Ordinary canceled completion stays
quiet. All concurrent Shutdown callers join the same completion barrier.
Depth-counted modal deferral covers Save (including nested SaveAs), CloseTab,
CanClose, Open/folder/plugin-install and reload confirmation, plus export's own
SaveDialog. Failed operations retain an otherwise unchanged job.

UI has a small progress/cancel row and cached plugin status. The panel performs
no hashing/inspection/launch. Status distinguishes engine processing from final
file publication; stale/draft/cancel/dialog/write failure cannot say a file was
exported. Last verified version/hash is labeled historical after a later
verification failure. Disabled Markdown still falls back to editable source
without an executable capability. JSON plugin semantics are unchanged.

## Actual focused validation

- `artifacts/acceptance/markdown-worker-app-20261109-012428-attempt01`:
  `test ./app -run TestWorkerExport -count=1 -timeout120s -v` PASS1.773s,
  command wall4.470s, exit0 and stderr0.
- `artifacts/acceptance/markdown-worker-app-20261109-012803-package-attempt02`:
  full owned `app` package PASS5.411s, command wall7.617s, exit0 and stderr0.
  Existing app regressions stayed green; opt-in legacy image/resource tests and
  production bundled worker acceptance were intentionally skipped.
- `artifacts/acceptance/markdown-worker-app-20261109-013057-vet`:
  `vet ./app ./ui/editor ./modules/markdownview` exit0, wall1.525s,
  stdout/stderr empty. All Go commands used `scripts/go.ps1`.

No app command failed. Root's three source-review findings were fixed before
attempt01: canceled cleanup failure visibility, recovery restoration on refused
close, and truthful cached final outcomes. Twelve test families cover actual
registered source/noView/preview/split routing; source/version/selection/history/
dirty/baseline preservation; readonly/conflict; foreign group/8MiB refusal;
pending actual Markdown IME Flush; postcompletion draft and dialog-time draft;
edit+undo stale refusal; busy/new job and immutable snapshot after live editing;
successful/failed lifecycle actions; nested close-save/Open/folder/install
dialogs; canceled cleanup errors; dirty existing+new draft session.Load and
RestoreTab after cleanup-refused close, including secondary persistence failure;
Shutdown of running/queued results without UI delivery; exclusive publishing
and surfaced warnings. These are app seam tests, not OS IME certification or
production process acceptance.

Four final760x480native MyGo tester images were rendered under attempt02/images:
`worker-progress-false-760x480.png`, `worker-progress-true-760x480.png`,
`worker-status-false-760x480.png`, `worker-status-true-760x480.png`.
Owner viewed all four. Full RoleButton bounds for Cancel were asserted and the
native click canceled the job. Historical status is a deterministic fixture;
the image does not claim a bundled worker ran. No rerender after final opt-in
test-only additions was necessary because rendered product states were unchanged.

## Gated production acceptance seam

`export_worker_bundle_test.go` remains opt-in. After explicit integration gate,
compile the app test executable beside the freshly packaged host/engines and
run `TestWorkerExportBundledHostPathAcceptance` with
`LEVNOTE_WORKER_BUNDLE_ACCEPTANCE=1`. It uses real NewBundled, fixed worker,
registered source/noView app route and injected SaveDialog only. Oracles include
independent heading ID/task disabled checkbox/table literal/raw HTML escaping/
unsafe-link behavior, HTML+warning parity, original disk BOM/source bytes and
encoding/selection/history/dirty/baseline after two exports, and refusal to
overwrite the first target. This is a host-path tester, not physical dialogs.

`TestWorkerExportBundledHostPathRefusal` with
`LEVNOTE_WORKER_BUNDLE_REFUSAL=missing` or `integrity` runs only in a disposable
copy prepared by the integrator. It never alters bundle files or accepts an
arbitrary engine path. It checks typed failure before verified identity, no
dialog, no fallback, no target and unchanged document. Those final independent
assertions/refusal test were added after the focused app package pass and compiled
by the final vet. Subsequently the full suite, checked-in bundle build, real
host-path acceptance and both disposable refusal cases passed under the activated
integration gate. Exact evidence and preserved harness corrections are in the
full integration handoff linked above; focused records remain unchanged.

Publication remains the synchronous existing WriteNewFile: cancellation is not
available after its commit call starts, and disk latency still blocks UI. No new
async publication framework, streaming renderer, sandbox or resource comparison
claim is introduced. Bottom16source/script fingerprints and accepted marker
executable remain unchanged from the accepted checkpoints.
