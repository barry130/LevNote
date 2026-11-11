# Native scene consumer: API and state checkpoint

2026-11-09 — DESIGN ONLY. Command source gate is accepted; app implementation
waits root review of this checkpoint and the separate transport evidence.
No product/test edits, builds or process requests are part of this checkpoint.

## Exact optional editor boundary

Add `ui/editor/scene.go`; it imports only `core/enginescene` and standard types,
with no catalog/IPC/runtime dependency:

```go
type SceneCapability struct { ID, Format string }
type SceneRequest struct { Capability SceneCapability; Epoch uint64 }
type SceneWarning struct { Code, Message string; Start, End uint32 }
type SceneEngineInfo struct { ID, Version, SHA256 string }
type SceneResult struct {
    Scene enginescene.Scene
    Warnings []SceneWarning
    Engine SceneEngineInfo
}
// Additive Adapter field:
SceneEngine *SceneCapability
// Additive Host fields:
SceneEngine *SceneCapability
BeginScene func(owner View, request SceneRequest,
    complete func(SceneResult, error)) (cancel func(), err error)
ShowSource func(owner View) bool
```

`owner` is the actual mounted pointer View; host preflight/delivery compares it
with `t.View`. Callback identity is the private request object, never Go function
equality. Begin/returned cancel are UI-only, cancel is nonblocking/idempotent;
accepted completion is asynchronous, at most once, suppressed after cancellation.
Immediate admission errors return no handle/callback. Scene/warning ownership
transfers once to the view; no source or buffer enters the result.

MindmapAdapter advertises `levnote.mindmap-preview` / `mindmap-scene-v1`.
`editorHost` always copies an enabled adapter's capability and supplies the generic
provider. `Create` chooses a separate `sceneView` when readonly and capability is
configured; absent provider/bundle is a visible unavailable state in that view,
never fallback parsing. Editable Mindmap and drawio retain their existing view.
Disabled plugins retain the existing source-only adapter behavior. ShowSource
checks the same owner/current tab then calls the existing `setMode` barrier.

## One shared lane and ownership

App adds a private `engineLane` around the existing `a.exporter` / `exportRunner`;
the Runner API and process implementation need no change. Its one background
loop owns execution; a mutex and capacity-one wake signal protect active task,
one latest pending preview, one export reservation, shutdown and cleanup outcome.
Only UI admits/cancels jobs. Tasks carry immutable Snapshot/Size, context, bounded
completion slot and dispatcher; the loop never reads App/Tab/Buffer state.

`beginScene` checks current open readonly tab, preview/split mode, matching view/
adapter/capability, catalog pair/platform, buffer not in a group and <=16 MiB.
It captures document/buffer/version, adapter/capability, tab ID/name/path, mounted
view, request epoch and private callback token, then takes `Buffer.Snapshot()`.
No `String()`/parse/large copy on UI. New requests cancel older matching work and
replace the pending snapshot; canceled callbacks and pending snapshots are dropped.
An active canceled request remains the active slot until Runner.Export returns.

The loop calls existing `Runner.Export` for fixed ID. After joined return, it
decodes at most32 MiB payload via `enginescene.Decode` off UI, drops raw bytes,
copies bounded warning/engine metadata, fills one result slot and queues an
identity-only delivery. Decode failure is visible and never reaches the view as
partial scene. Delivery rechecks all captured identities/version/epoch and modal
depth before invoking the callback. Clear/take its slot before invoking any
callback so synchronous Release/reconfiguration cannot revive or overwrite it.
Cleanup errors remain recorded even when their view callback was invalidated.

`startEngineExport` retains current Flush/identity/version/group/source guards.
After admission, `exportJob` becomes the priority reservation: cancel active
preview, enqueue export ahead of pending preview, return immediately to UI.
The lane waits for active preview's joined return before running export; no busy
retry or UI wait. Reservation remains through export delivery, SaveDialog and
exclusive publication, even after the process is done. Only existing delivery's
final cleanup releases it; then revalidate and wake the one latest preview.
No goroutine per pending request or discarded completion; no timer retry loop.

Extend the existing depth-safe modal deferral to drain scene delivery as well.
Pair existing invalidations with scene cancellation; `releasePresentation` also
cancels nonblockingly, and source/Buffer replacement and recoverDraft paths must
invalidate even when they do not release the old view. Failed save/close actions
leave still-valid work intact. Source-version detection in Warnings covers direct
buffer edits without a callback. No stale result installs during a modal.

`closeEngineExport` can retain its private name for existing callers/tests but
closes the shared lane: drop pending work, cancel active, call Runner.Close outside
the lane lock, join the loop/result handoff, and return the stored outcome
idempotently without UI dispatch. Shutdown joins it; accepted CanClose cleanup
failure keeps the existing visible refusal and recovery-snapshot restoration.

## Scene-only view and rendering

New graphview `scene_preview*.go` keeps DTO, current generation/version, cancel,
status/error, selected ID, pan/zoom and bounded labels; no mindmap model/parser.
Warnings runs before Draw: it may initiate one generation; repeated idle calls
only inspect buffer identity/version. Release cancels/drops scene/cache/callback,
increments epoch and allows later Warnings/Draw reactivation of the same view.
States: inactive, loading, ready, stale-loading, failed. One previous scene may
remain visibly stale while its replacement runs. Failure stays visible without
retry at that same version; reentry or changed source creates a new generation.

Reuse package-local labelCache/clipping/arrow helpers where applicable, without
extracting editable inspector logic. Render DTO theme roles, literal overrides,
root then selected stroke precedence, collapsed badge, culling, fit/pan/zoom and
visible-ID selection. Theme change does not request a worker. Clamp float64
dimensions to a positive safe floor before float32 conversion/fit; bound zoom
and pan and reject nonfinite input transforms. No NaN/Inf or unbounded grid loop.
Labels retain producer515-byte maximum without a second truncation.

Show loading/stale/unavailable text plus a keyboard-reachable source action and
source-only inspector explanation. No edit controls or image badge promise.
View.Export returns an explicit host-export-only error, never parses; the adapter
already routes actual SVG export to its fixed capability. NavigateOutlineItem
only looks up current scene IDs; stale/unavailable => ErrOutlineStale, absent
resolved target => ErrOutlineReadOnlyHidden, visible => select/center without
source mutation. App's existing OutlineSource still resolves source identities;
this slice makes no claim of removing all host parsing.

## Files and finite evidence

New: editor/scene.go; app/preview_worker.go, engine_lane.go plus focused tests;
graphview/scene_preview*.go plus native tests. Narrow wiring: editor/contract.go;
graphview/adapter.go; app/app.go, reference.go, export_worker.go, view.go,
workspace.go, plugins.go and existing reload/invalidation seams only as required.
No format/codec/transport/runtime/vendor/script/package edits. Existing runtime
failure names may still say export; app wraps them truthfully as preview errors.

Tests first exercise fake runner concurrency<=1, priority+off-UI cancel/join,
latest-only pending, snapshot/version/owner races, bounded result retention,
modal/reentrant Release, malformed DTO, failure/no retry and Close without dispatch.
Native literal-scene tests cover Release/reactivation, theme without requests,
selection/fit/pan/zoom/outline/source access, stale replacement and tiny dimensions.
Use actual registered readonly adapter and keep editable/disabled-plugin/export
regressions. Final changed editor/graphview/app packages and scoped vet follow
root source review; fresh three-worker packaging/real app path/captures are a
separate gate. No physical OSIME or resource-advantage claim in this slice.
