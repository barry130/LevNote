# Second-engine runtime and app integration

Status: IMPLEMENTED, owned gate PASS and source frozen for root review,
2026-11-09. No bundle, production worker, resource matrix or physical UI launch
was performed. The separate catalog/IPC and Mindmap command owners retain their
files; existing `.5` delivery remains unchanged/current.

## Final behavior and source scope

Runtime resolves the compiled catalog before source writing/launch. It admits
only the exact ID/format/platform pair, uses the operation's source limit and
fixed bundle path, and checks each manifest field against the requested engine.
A coherent Markdown manifest cannot satisfy a Mindmap request. Mindmap accepts
up to16MiB source; Markdown remains8MiB. Both keep the existing64MiB wire output
limit. `Result.Payload` is the sole output slice; HTML-specific field names in
historical resource JSON remain unchanged.

The shared slot, locked verification handles, hidden child, request writer,
Job/descendant cleanup, context priority, Start/Wait, Close linearization and
cleanup poisoning remain intact. `process_windows.go`, `bundle_windows.go` and
platform implementations were not changed. Repository helper executables used
by runtime tests are not the production Mindmap worker or a sandbox proof.

App validates the advertised pair, captures owners around synchronous Flush,
then exports an immutable snapshot. Ordinary Mindmap field Flush may write its
existing edit transaction; pending tag input refuses before worker/dialog and
remains recoverable. Source mode keeps View nil. Status and limit messages name
SVG/Mindmap correctly; missing/corrupt results never fall back to View.Export.
The existing modal, late-result, version, exclusive publication and shutdown
guards are reused. Publication still uses synchronous WriteNewFile; once begun,
it cannot be canceled. Only MindmapAdapter gains the SVG capability; Drawio is
unchanged. Disabling Mindmap retains the source fallback and cancels its job.

Changed implementation: `core/engineruntime/{types,run,runner,bundle}.go`,
`app/export_worker.go`, `modules/graphview/adapter.go`.
New tests: `core/engineruntime/engines_test.go`,
`app/export_mindmap_worker_test.go`.
Adapted existing tests: runtime `runner_test.go` adds fixed Mindmap helper modes;
`bundle_test.go` adds the expected ID argument; runtime
`worker_resources_test.go` and app `export_worker_test.go` /
`export_worker_bundle_test.go` mechanically use Payload. The resource helper
retains its literal Markdown filename and original historical JSON schema.

## Meaningful independent oracles

Four new runtime tests:

- `TestRuntimeMindmapIdentityPayloadAndCrossResponse`: exact request identity,
  independently constructed escaped SVG, wrong response engine/format, one
  Start/Wait per request.
- `TestRuntimeOperationLimitsBeforeSourceOrLaunch`: unknown/negative/oversize
  refusal without source reads; allowed Mindmap8MiB+1/exact16MiB advances only
  to missing-bundle verification. This is not a large real-worker measurement.
- `TestRuntimeMindmapFixedManifestPairAndPath`: mismatched identity, format,
  filename, platform, traversal and coherent Markdown manifest; no redirection.
- `TestRuntimeSingleAdmissionAcrossEnginesAndJoinedReuse`: one slot across IDs,
  Busy without reading the second source, then successful joined reuse.

Six new app tests:

- `TestMindmapWorkerModesReadonlyAndExactPublication`: three modes × readonly,
  literal controlled SVG, exact source/BOM/encoding/baseline/version/history/
  selection, SaveDialog options and exclusive-target refusal.
- `TestMindmapWorkerFieldFlushAndPendingTagRecovery`: actual native Compose,
  successful field snapshot and undo, combined pending drafts blocked before
  worker/dialog, exact recovery envelope through the registered controls/host.
- `TestMindmapWorkerPreflightPairSourceCapAndFlushOwners`: pair/cap boundaries
  and synchronous buffer/adapter/capability/recovery owner changes.
- `TestMindmapWorkerStaleSnapshotNoFallbackAndStatus`: immutable old snapshot,
  live source/descriptor changes, missing/corrupt results and per-ID status.
- `TestMindmapWorkerAndMarkdownShareAppAdmission`: tab cancellation retains
  busy admission until the prior completion joins, then accepts the other ID.
- `TestMindmapWorkerDisableRetainsSourceFallback`: disable cancels without
  publication, preserves source/history, and re-enable restores capability.

App tests use a controlled exporter to prove orchestration/publication, not
renderer parity or production worker execution. Existing shared Markdown/modal/
close/recovery tests and graphview native marker/tag tests passed unchanged.

## Actual checks and retained failure

All commands use `scripts/go.ps1 -GoArguments` with explicit arrays; child
scripts clear LEVNOTE opt-ins and GOFLAGS. Exact runners and streams are retained.

1. [focused-attempt01](../artifacts/acceptance/mindmap-engine-20261109-063459-focused-attempt01/result.json):
   runtime PASS4.097s, app FAIL1.959s solely on the extra space in
   “思维导图 源码超过…”. All other new behavior passed. The failed record remains
   SHA256 `0277E1FB0C6807C474307984786F4CCB3758B6F5278CD0ADB8A01EB3B7424E79`.
2. [preflight-attempt02](../artifacts/acceptance/mindmap-engine-20261109-063721-preflight-attempt02/result.json):
   corrected only that Chinese spacing while preserving the Markdown phrase;
   affected preflight PASS1.801s. Record SHA256
   `D64B4A1491BADD35016061E6772CB74FE521BF2D389AABB12891729A2AAD0DC0`.
3. [owned-final-attempt03](../artifacts/acceptance/mindmap-engine-20261109-063755-owned-final-attempt03/result.json):
   complete runtime PASS26.148s, graphview PASS7.005s, app PASS7.389s; scoped vet
   exit0. Wrapper elapsed27.159/8.843/8.828/1.301s respectively. All stderr empty.
   All92 inventoried sources unchanged during execution, including imported
   catalog/IPC production sources and accepted pure Mindmap sources.

Final commands were `test ./core/engineruntime -count=1 -v`,
`test ./modules/graphview -count=1 -v`, `test ./app -count=1 -v`, then
`vet ./core/engineruntime ./modules/graphview ./app`.

Runtime's held descendant/inherited-pipe/cancellation/parent-death/Close oracles
passed. `TestBundleRejectsReparse` SKIP: Windows symlink privilege unavailable.
Resource and app bundled-worker/screenshot opt-ins SKIP as intended. No resource
numbers, physical OSIME claims or production bundle claims derive from this gate.

## Frozen fingerprints

Final [result](../artifacts/acceptance/mindmap-engine-20261109-063755-owned-final-attempt03/result.json):
`0D8117AE0AF34A7F01AAC7564FD18580895315D86AEF1E1A1A1509883A7EF8D0`.
Identical [before/after source inventory](../artifacts/acceptance/mindmap-engine-20261109-063755-owned-final-attempt03/source-after.json):
`DAD757EC9BEF9AB04844D4D7B40CDFC0E35AF29C0073E86613AEA65E12847464`.
That inventory provides all92 full hashes; key new/current seams are:

| File | SHA256 |
|---|---|
| core/engineruntime/bundle.go | `9ABCC9EF3D7CF4DC2649171BA543726E6C8682CAC47E9D3E7F67888C80EFDDAE` |
| core/engineruntime/run.go | `2F7901206AD42D6A7F361931655E645361A9E27C3CAEAACF300BE4C7B4E0F960` |
| core/engineruntime/engines_test.go | `6F152F912951D7AC3DB89CB58A1835639CFE52B61C7291ADAD92DCBF33680FA9` |
| app/export_worker.go | `DFDE688656A15B3671780872E9D69CEBC1FB84C42B9607405ED31BF25A3E457E` |
| app/export_mindmap_worker_test.go | `D010ADB9E9DBADC8FFBC136B176669DCF74E1689483E73ECF5393736286E6414` |
| modules/graphview/adapter.go | `06BCAF60C3ADAB8DB30C5943DFDB75FCCBA2FBA0F032A3BE8D0A861A99364915` |

Imported production hashes: catalog `B064E016…`; IPC decode `663A6B44…`,
json `FE829AE2…`, protocol `E1DC708E…` (full values in inventory). IPC tests are
excluded from the freeze inventory because their separate owner was finishing
evidence; no dependency production file changed during these checks.

Next step requires root source/evidence acceptance and separate delivery-plan
activation. Packaging script and production host-path opt-in remain queued in
[mindmap-engine-delivery-next-stage.md](mindmap-engine-delivery-next-stage.md).
