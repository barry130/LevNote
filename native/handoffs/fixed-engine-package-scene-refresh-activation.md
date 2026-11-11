# Readonly scene refresh — independent native view slice

2026-11-09. ACTIVE for Astra astra_marker_view. Root read and accepts runtime
preflight direction. Core store is still GLM-owned; avoid waiting idle by
implementing this independent required view contract now. This overrides the
previous preflight-only scope ONLY for the files below. No app/runtime access.

Own native/ui/editor/scene.go and native/modules/graphview/scene_preview.go,
plus a narrowly named new scene-refresh test file there. Add the optional
neutral `editor.SceneRefresher interface { RefreshScene() }`; implement only on
readonly sceneView. No interface additions to mandatory View/Host, no format/
worker/DTO/parser changes and no app wiring yet. No child delegation.

Refresh cancels the current request via its existing cancel callback, advances
epoch to reject its late completion, marks next Draw/Warnings to request current
source at unchanged Buffer/version, and clears loading/error/warnings and
transient navigation/gesture/label state. Preserve source, readonly flag,
selection/undo/dirty, and scene zoom/pan/current selected ID. Keep the previous
scene only as explicitly stale until a successful replacement; failed refresh
must show stale/failure and reject stale node navigation. New scene may drop a
selected ID that no longer exists. Do not implement on writable graph views or
release/reconfigure them. Repeated refresh, Release-before-complete and stale
callback must not resurrect state or duplicate active requests.

Meaningful focused tests through the real view/controlled Scene provider:
same-version refresh issues exactly one request on next read/draw, cancels old,
late old success/error rejected, source/undo/zoom/pan selection untouched,
new absent selectedID cleared, error retains explicitly stale content, readonly
navigation refused while stale, repeat refresh and Release. Use existing view
test seams; don't add app simulation or new generic framework. Native tester
only if needed for actual draw behavior, no additional screenshots required.

Run only new focused TestNativeSceneRefresh tests and graphview/editor vet with
actual streams/exit retained in one fresh evidence directory, scoped source
fingerprints before/after. Wrapper must exit LASTEXITCODE. No build/app/worker,
full repo suite, resource replay or package bytes changed. Publish source/API
and results in fixed-engine-package-scene-refresh.md, then root reviews. This
contract is preparatory; it is not activated by current .8 app or shipped yet.
