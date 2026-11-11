# Readonly scene refresh — focused implementation complete

2026-11-09. Source and the authorized focused checks are complete; root review
pending. This preparatory contract is not wired into the app or shipped in .8.
No app/runtime, parser/DTO/worker, packaging script or accepted artifact changed.

## Contract and implementation

`ui/editor/scene.go` adds the optional UI-thread interface
`type SceneRefresher interface { RefreshScene() }`. Mandatory View/Host remain
unchanged. Only the configured readonly `sceneView` implements it; editable
Mindmap, standalone parser-backed Mindmap and drawio views do not.

`modules/graphview/scene_preview.go` advances the epoch and clears the captured
cancel callback, loading/error/warnings, pending centering, held pan and label
state before invoking cancellation. Nothing writes view state after that
external call, including when cancellation synchronously completes or releases
the view. The next Draw/Warnings (or existing lazy navigation entry) requests
the current buffer even at the same version. Repeated refreshes coalesce.

The previous scene stays explicitly stale while loading or after failure;
outline navigation/current-item reporting cannot treat it as current. Zoom,
pan and selected ID remain until replacement, which drops an ID absent from
the new scene. Source bytes, readonly state, selection, history and dirty state
remain untouched. Successful idle frames settle without further requests.

## Actual validation

Evidence directory:
`native/artifacts/acceptance/fixed-engine-package-scene-refresh-20261109-124417-attempt01/`.
Its `run.ps1` records actual hidden PowerShell child exit codes and redirects
raw streams; `go-check.ps1` uses explicit Go argument arrays and exits
`$LASTEXITCODE`.

| Command through `native/scripts/go.ps1` | Actual result |
| --- | --- |
| `test ./modules/graphview -run ^TestNativeSceneRefresh -count=1 -v` | PASS; 5 top-level cases, clean/dirty subcases; package 1.189s, command 4.0259231s, exit 0 |
| `vet ./modules/graphview ./ui/editor` | PASS; empty stdout/stderr; command 1.5977629s, exit 0 |

New independent oracles in `scene_refresh_test.go`:

- `TestNativeSceneRefreshSameVersionDrawPreservesDocument`: real Draw and held
  secondary-button pan; one lazy same-version request, explicit stale notice,
  preserved clean/dirty/history/selection/viewport, canceled pan and settled idle.
- `TestNativeSceneRefreshCoalescesAndRejectsOldCallbacks`: synchronous cancel
  completion, repeated refresh, old success/error, Release and lazy reentry.
- `TestNativeSceneRefreshFailureRetainsStaleThenDropsAbsentSelection`: failed
  replacement keeps disclosed old content and refuses navigation; explicit retry
  removes a missing selected ID without altering source or viewport.
- `TestNativeSceneRefreshCancelMayRelease`: cancellation can Release; no later
  state write or old completion resurrects it.
- `TestNativeSceneRefreshOptionalReadonlyContract`: optional interface is confined
  to the configured readonly scene consumer.

All 13 scoped inputs are unchanged during execution. `source-before.json` and
`source-after.json` share SHA256
`2FC95F146C47571A5DB0BF24927ADC24FB6EBA4BD751E774AEF775CBBAFF04C3`.
No failed test attempt, full module suite, screenshot, real worker or app launch
was needed in this slice. Tests use the existing controlled Scene provider;
this is not an installed-package integration claim.

## Frozen checkpoint

| Path | SHA256 |
| --- | --- |
| `ui/editor/scene.go` | `3B3DD7057A8505A3B56C789B6F1094EFE129F80BDE098940163005801CBB8662` |
| `modules/graphview/scene_preview.go` | `E87E342C2137B8BC9D1D05DB96F9B5FA0F526291C90AD879B73C600047D74980` |
| `modules/graphview/scene_refresh_test.go` | `53E5ED54EFA00DC76AB5D3E4AB6F1677370210954F36860849608F1476920C3E` |
| evidence `result.json` | `626AB085ED7F23F5F55DAE09C250ECA54221BECDB0594AB38E12D98EA1622835` |
| evidence `test.stdout` | `4B0F4B84BC22E3874828E5335EB691DDA4D437C22E454E531B4E4E4A27C8F4E1` |

The result, raw streams and before/after inventory are final. Future package
selection wiring must invoke this optional contract only for applicable readonly
views and arrange the next UI read/draw; that app/runtime work is a separate gate.
