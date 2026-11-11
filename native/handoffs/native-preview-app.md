# Readonly scene consumer — source and focused checkpoint

2026-11-09. Implementation, focused checks and the combined final changed-package
test/vet gate are complete; source is frozen for root acceptance. No packaging,
real worker, physical UI, screenshot or resource execution is part of this step.

## Implemented boundary

- Additive `ui/editor/scene.go` / `contract.go` expose neutral capability,
  request/result, asynchronous BeginScene cancellation and owner-checked source
  navigation. App snapshots the readonly Buffer; views supply no source/path.
- `app/engine_lane.go` owns one execution loop, one newest pending preview and an
  export reservation through dialog/publication. Export cancels the execution
  attempt of a valid preview, waits its joined return, then resumes that generation
  after publication unless a newer one supersedes it. Cleanup poison survives
  view cancellation; Close joins without UI delivery and retains its outcome.
- `app/preview_worker.go` captures identity/version and decodes off UI into one
  result slot. Cancel immediately drops callback/result/tab/doc/buffer/view
  ownership; the pending task drops its snapshot, while active execution retains
  only its immutable input until join. Completed tasks drop source and closures.
  Production uses nonblocking Window.Invalidate; View/modal unwind takes results
  before calling a view. A headless export-only dispatch seam preserves existing
  deterministic export tests without becoming a cleanup dependency.
- Narrow app wiring covers release/dispose/mode/tab/reconfigure/reload/recovery,
  existing export lifecycle and depth-safe modal delivery. The scene adapter is
  always selected for configured production readonly Mindmap, even when provider
  is unavailable. Editable Mindmap/drawio and disabled-plugin fallback remain.
- Separate `graphview/scene_preview*.go` consumes only the neutral DTO; no parser
  or editable inspector. Loading/stale/unavailable and source-only limitations,
  theme/literal colors, bounded labels/culling, finite transforms, zoom/pan/fit,
  selection and visible-ID outline navigation are retained. Lazy outline centers
  after positive canvas geometry. Hidden-label disclosure is an absolute semantic
  text overlay; only its visibility transition requests one additional frame.

## Actual focused evidence

`artifacts/acceptance/native-preview-app-20261109-080612-focused-attempt01`:
8 new app tests PASS2.306s, three native tests PASS; fourth fails because its
HasText oracle cannot observe raw Painter.Text. All97 captured inputs unchanged.
Original sources/logs/result are retained. This led to the semantic overlay above.

`artifacts/acceptance/native-preview-app-20261109-080927-focused-attempt02`:
12 app top-level tests PASS1.716s (8 new plus4 affected readonly consumers),
4 native tests PASS1.011s, exit0, stderr empty,97 inputs unchanged.
Exact command: `scripts/go.ps1 -GoArguments @('test','./app',
'./modules/graphview','-run','^(TestScenePreview|TestNativeScene|TestReadonlyMindmapAtMinimumSize|TestMindmapReadonlySaveAs|TestMindmapWorkerModesReadonly|TestBuiltinMindmapReadonlyHidden)',
'-count=1','-p=1','-timeout','120s','-v')`.
Actual wrapper arguments, timestamps, stream fingerprints and before/after are
stored in `go-check.ps1`, `test-command.json`, raw logs and `result.json`.

Substantive new oracles include immutable exact snapshots and invalid-source DTO
display; single active fake runner; priority/join/dialog reservation/resume;
latest-only pending and cancellation dropping its snapshot; mode/tab/dispose
owner races/reentry; nested modal and invalidated cleanup poison; Close without
dispatch and canceled-before-run export completion; malformed DTO/unavailable
no retry; scene release/late callbacks, first-paint center, theme/keyboard source,
finite subnormal geometry and settled hidden-label disclosure. Ignored-cancel
fakes unblock on Close even when a test fails before answering a request.

Readonly legacy app assertions now use a neutral fake result, retain source/BOM/
history/selection/baseline/exclusive export and writable SaveAs editing checks,
and verify the source-only inspector limitation. The existing opt-in readonly
outline capture was adapted to an independent literal scene and remains unrun.

Independent source-only scheduler review:
[`native-preview-consumer-independent-review.md`](native-preview-consumer-independent-review.md).
Root accepted its corrected no-actionable-finding conclusion; no process/OS claim.

## Final actual gate and fingerprints

`artifacts/acceptance/native-preview-app-20261109-081147-owned-final-attempt03`
ran full `./app ./modules/graphview ./ui/editor -count=1 -p=1 -timeout 180s -v`.
Graphview PASS9.785s and editor PASS1.874s. App failed only two legacy readonly
marker/tag clauses that expected editing panels; no new scene case failed.
Vet was skipped after that exit1. Raw failures and all97 unchanged inputs remain.

Only those two app test clauses were changed: neutral fake scene plus visible
source-only limitation, absence of mutation controls, actual canvas Type and
unchanged source/version/selection/history/dirty checks. All editable marker/tag
oracles remain. Production, graphview and editor bytes are identical to attempt03.

`artifacts/acceptance/native-preview-app-20261109-081403-owned-followup-attempt04`
ran full app PASS8.571s and `vet ./app ./modules/graphview ./ui/editor` exit0,
empty stderr and97 unchanged inputs. The earlier graph/editor full passes are
reused, not rerun. Opt-in real-worker/capture/resource tests remained SKIP.

Combined record `checkpoint.json` SHA256:
`B0D41D89376E4D2D98DA4F97329743ADF78686F6D0552EE48A262B28E3D95A2D`.
It links actual results and the cross-attempt two-test-only delta. Final
`result.json` SHA256:
`F9184121B2545758C7D98299937364D2919861C6D5901109019F723062C144E2`;
`source-after.json` SHA256:
`A15AD52664CB414C8A2D1A49A039C7B5E356DC63C87AA61CBF0E1BD60C699173`.
The inventory contains all source/test and imported catalog/IPC/runtime/DTO
production fingerprints; complete exact values live there, not in prose copies.

Final owned source list: `ui/editor/{scene,contract}.go`;
`app/{engine_lane,preview_worker,export_worker,app,reference,view}.go`;
`modules/graphview/{adapter,scene_preview,scene_preview_draw,scene_preview_input}.go`.
New tests: `app/preview_worker_test.go`, `modules/graphview/scene_preview_test.go`.
Narrow existing test adaptations: app readonly size/SaveAs/outline, Mindmap worker
modes, marker/tag readonly tails and opt-in readonly outline capture.
