# Root plan: bounded read-only native scene consumer

2026-11-09. Planning only, app/UI/runtime implementation remains queued after
command/transport acceptance. The prior DeepSeek design task was interrupted
without a written checkpoint; no product ownership transferred. This root plan
resolves actual source lifecycle requirements for the future implementation owner.

## Native view boundary

Add a separate read-only scene view selected by MindmapAdapter.Create only when
Host.ReadOnly is true and the advertised generic scene capability is configured.
Production must configure that path consistently; provider/bundle failure is a
visible unavailable state, never a hidden fallback into mindmap.Parse. Existing
editable Mindmap/drawio and their inspector/draft paths stay in current view.

Small optional ui/editor contract owns neutral request/result types: ID/format
strings, engine ID/version/hash, bounded warning code/message/source byte range,
enginescene.Scene, callback/cancel. No runtime/catalog/IPC import into editor.
App owns snapshot capture; the view requests its known capability, not an
arbitrary command/path or source. Begin returns immediate preflight error or a
nonblocking idempotent cancel handle. Completion is asynchronous on UI, at most
once and suppressed after cancellation. Exact signature is owner's short source
checkpoint; no separate per-format Host callback or generic plugin framework.

App captures tab/document/buffer/version/adapter/capability/view plus view-request
epoch and callback identity before admission. Recheck them before delivery.
Worker does not echo source identity. Request and result metadata stay bounded;
raw response payload drops after scene decoding. Background code never reads UI
owned App/Tab/Buffer; it uses captured immutable Snapshot and identities only.

Warnings() is called BEFORE Draw in app.documentView; it may lazily initiate a
single request when entering the active view. Release is NOT permanent disposal:
app.releasePresentation leaves t.View alive for tab/mode reentry. Release cancels
the current generation, drops scene/caches/owned callback, does not wait for a
process or close the shared Runner. Next Warnings/Draw may reactivate same View
with a fresh epoch. Preserve small viewport/selection preferences as appropriate.

## Application scheduling

Reuse existing one a.exporter/engineruntime.Runner, not a second runtime. Existing
Export method can execute a fixed preview descriptor without widening input
authority; names/messages can be clarified narrowly by owner if needed.

At most one active job (including cancellation/join), one newest pending preview
snapshot and one old displayed scene. Admission decisions are made on UI; async
workers only publish into bounded protected result slots. A pending snapshot is
replaced on newer request; cancellation drops callbacks/source ownership when
safe, with cleanup completion tracked independently of UI dispatch.

startEngineExport retains its existing flush/owner/version checks. Once accepted,
its exportJob is a priority reservation: cancel current preview, wait its owned
completion off UI, then call shared Runner. Do not admit next preview while this
reservation is live, including export dialog/publication phase. Completion/cancel
of export wakes one latest pending preview after ownership recheck; no timer or
FailureBusy retry loop, no goroutine per stale callback. Existing modal deferral
continues; callbacks into views are likewise deferred until modal operations end.

Tab switch, mode switch, source replacement, plugin reconfiguration/disable,
recovery, disposal and shutdown invalidate matching previews. Existing export
invalidation already marks many transitions; add paired scope where required.
Shutdown and CanClose cancel/join both paths independently of dispatch and
preserve existing cleanup-failure refusal. Release on ordinary UI transitions is
nonblocking. No result may revive a released view or attach to a replacement tab.

## Presentation and retained functionality

Use scene-only drawing for nodes/edges with current theme roles, literal overrides,
root/selection precedence and collapsed badge. Keep existing bounded label cache,
culling/clipping, pan/zoom/fit and selection behavior. Theme change does not make a
worker request. Positive float64 dimensions may underflow when cast to float32:
fit/zoom conversion must guard finite values and use safe lower bounds. No source
parser/model/image data may enter this view. Reuse neutral render helpers when
simple; don't extract editable inspector code just for this pilot.

Keep loading, unavailable, canceled/stale state visible. On source-version update,
one old scene may remain explicitly stale until replacement; preserve viewport
and selected ID only if present. Clear heavy caches when source/model changes.
No automatic retry on failure at same version; explicit reentry or later source
version can request again. Show source-only inspector limitation and reachable
source command. No promise of image badges; producer's truthful image warning
and full-source entry suffice for this pilot.

Existing adapter outline parser remains; no full host parser-removal claim.
Visible IDs navigate by scene identity. Hidden folded read-only target reports
ErrOutlineReadOnlyHidden without expanding/mutating source. Existing source
outline selection must not force synchronous concrete parse in the scene view.

## Acceptance sequence

Owner publishes concise API/state/source checkpoint, then pure fake-runner
scheduling tests and native view tests with independent literal scene. Assert
active-count<=1, export-priority order, join off UI, one latest pending snapshot,
identity races, modal release, immediate refusal/cancel/failure/no retry, Close
without dispatch, malformed payload rejection and no host parser fallback.

Native tester covers theme without requests, selection/pan/zoom/fit, stale/source
update, Release/reactivation, outline visible/hidden behavior, accessible source
entry and truthful errors. Tiny positive dimensions and large finite coordinates
must not yield NaN/Inf or unbounded draw work. Existing editing/fields/tags/outline/
export regressions remain relevant but avoid unrelated full repository reruns.

Only after source tests/vet and root review: fresh fixed three-worker package,
one actual readonly app path with complete bundle and exact source preservation,
focused source/preview visual evidence and truthful docs. No physical OSIME or
resource advantage claim without a separate actual measurement gate.
