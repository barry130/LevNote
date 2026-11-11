# One-shot native Markdown export engine — root checkpoint contract

Status: **FULL INTEGRATION ACCEPTED; FINAL DOCS AND LAUNCH ASSETS ACTIVE**.
Root accepted exact final aggregate FB0468DADF9F526AFD7BAE72BD1E3893CAD7CEE78176870DDDAE5BB59B12A83A
and20/20validator2026-11-09 01:55local after actual source/log/oracle and
corrected metadata review. Accepted payload is host4B5DCECA.../workerDB8F02E6...;
real bundled export/refusals and hidden native startup/owned shutdown passed.
Only final docs, separate acceptance record and clean launch copy are now active;
keep binary/source/evidence immutable. Earlier activation history follows.
Root accepted the
app focused gate2026-11-09 01:32local after source/oracle review, actual
TestWorkerExport PASS1.773s, app-wide PASS5.411s, scoped vet1.525383s exit0,
four760x480 native images and final opt-in bundled test source. Product/tests
are frozen. Astra alone now executes `markdown-worker-integration-task.md`;
release acceptance, actual bundle results and startup remain pending. Root
accepted the exact marker release
17E014EB6F61E671F6FCCA439719E8831B7591042A6F29EE5354960C120C0466.
The protocol review is resolved in this controlling contract. The initial
protocol owner published the seam but delivered no source or status after
repeated requests and was interrupted at00:37local. Astra now owns both pure
protocol/worker and runtime/packaging; keep agreed APIs unchanged. Astra already
finished marker final docs. No peer edits by interrupted owners.
The first app design reviewer produced no ACK/file after repeated requests and
was interrupted. Root reviewed the current app seams and explicitly activates
replacement read_native_worker_app_implementation_task_txt for app integration.
That replacement later produced no implementation/handoff after25minutes and
repeated checkpoint requests; root interrupted it at00:50local. App ownership is
now transferred to Astra. Root accepted current protocol/worker/runtime focused
checkpoint EAA53F06003BD871D442178F7781CB5AE2425269ED8C2E273E0BF1E7BFCBEB50
after actual source/log/oracle review and activates app integration01:09local.
Bottom layers stay frozen unless a demonstrated defect requires assigned fix.
The later app focused acceptance above activates the release/package plan;
no results are implied by activation. Root coordinates and reviews only; children do
all implementation, validation, packaging and launches. No child delegation.

This finite checkpoint addresses the difference between the existing installed
JSON syntax/outline configurations and an independently packaged format engine.
It ships one repository-owned Markdown HTML engine. Native UI, preview and
editing stay in MyGo. Arbitrary executable installation, rich-view IPC, an SDK,
background service, network access and new third-party dependencies are outside
this checkpoint. The existing pure parser/exporter remains our implementation.

## Ownership and shared API

- `astra_marker_view` (protocol takeover): new `core/engineipc/**` and
  `cmd/levnote-markdown-export/**`. Publish protocol constants/signatures first;
  implement bounded framing, validation, worker and meaningful focused tests.
- `astra_marker_view` (original ownership): new `core/engineruntime/**`, Windows job/handle lifecycle,
  fixed bundle verification and a new explicit packaging script. Do not edit
  the existing Go wrapper, vendor or app files.
- `astra_marker_view` (app takeover ACTIVE after focused bottom-layer acceptance):
  app export job/UI/lifecycle and its tests; additive
  `editor.Adapter.ExportEngine` metadata in `ui/editor/contract.go`; exactly
  the Markdown descriptor assignment in `modules/markdownview/view.go`; final
  product docs and sole full integration after root's focused gate.
- All other product code stays frozen. A demonstrated peer defect is reported
  before changing ownership. GLM must compact before200K context tokens.

Use the callable API and exact frame layout in
`markdown-worker-runtime-proposal.md`. This shorter contract controls scope and
resolves policy choices; the design proposal is supporting rationale.

Protocol package: `EngineID`, `MarkdownHTML`, `RequestMeta`, `Warning`,
`EngineError`, `ResponseMeta`, `EncodeRequestPrefix`, `ReadRequest`,
`WriteResponse`, `NewResponseDecoder`, decoder `Write` and `Finish`.
ProtocolVersion1; fixed ID `levnote.markdown-export`, format `html`.

Runtime: `Source{Size int64, Data io.WriterTo}`, `EngineInfo`,
`Result{HTML,Warnings,Engine}`, typed `Failure`, `Runner`, `NewBundled`,
`Export(context.Context, EngineID, Source)` and idempotent `Close`.
One active call; busy rejects, no queue. Return only after cleanup and joins.
No public arbitrary command/path/env injection. Private repository test seams
are allowed. App owns its injectable exporter interface for deterministic tests.
Result.Engine is populated after successful per-run bundle verification, even
if a later launch/protocol/engine step fails; HTML remains nil on all failures.
Before successful verification Engine is zero. A cached previous verified
version/hash must be labeled historical alongside the latest failure.

Adapter metadata is optional `*ExportEngine{ID string, Format string}`; the
shell routes by that capability, without importing a concrete format module.
Schema1/2 installed plugins cannot supply executable capabilities. Disabling
the existing Markdown plugin removes its rich capability as before.

## Wire and resource boundaries

Exactly one24-byte little-endian header in each direction:8-byte magic
`LVEXREQ\0` / `LVEXRES\0`, uint16version1, uint16headerSize24,
uint32metadata bytes, uint64payload bytes. Then exact metadata, exact raw
payload and EOF. No base64, whole-source JSON, multi-message session or socket.

Limits: request metadata4KiB; source8MiB inclusive matching the existing
`markdown.MaxParseBytes`; response metadata1MiB; HTML64MiB inclusive;
256warnings with code64bytes/message512bytes and valid source byte ranges;
error message1024bytes; engine version1..64ASCIIbytes in0x21..0x7E; stderr16KiB.
All lengths validated before conversion/allocation/summing. UTF8 is validated,
not silently repaired. JSON requires exact allowed key case/presence/types,
no duplicate keys at any object level, valid surrogate pairs, no unknown fields
or trailing JSON. Empty source is legal. Required arrays are arrays, not null.

Protocol review resolutions (root,2026-11-09):
- Source is the exact UTF8 snapshot of the decoded editor buffer. The document
  layer already removes file-encoding BOMs and tracks encoding separately;
  neither the app export path nor worker strips/transcodes any further bytes.
  A literal U+FEFF remaining/inserted in the buffer is content and is accepted
  unchanged. Source.Size and warning ranges refer to exactly the submitted
  bytes, excluding any encoding BOM already removed when opening the file.
  Validate payload UTF8 before Parse. Protocol-to-parser cap equality is tested
  in the worker tests, not an import dependency in core/engineipc.
- EngineVersion must be nonempty,1..64bytes of ASCII0x21..0x7E, with bytewise
  equality to the verified manifest. Whitespace/control/nonASCIIversions refuse.
- Worker error codes are exactly invalid-request, source-limit, invalid-utf8,
  output-limit, warning-limit, render-failed, partial-document. ReadRequest
  failures return a coherent bounded status:error response where possible:
  an oversized declared source maps to source-limit, invalid source UTF8 to
  invalid-utf8, all other invalid header/frame/meta/trailing/truncated requests
  to invalid-request. No invalid request reaches Parse. Those error responses
  have zero HTML and exit0 after a successful write; transport/write failure or
  an unexpected crash exits nonzero. Early error response does not waive the
  host's exact input-writer/exit/EOF checks; any writer failure also refuses.
  Non-request renderer/limit failures use the other fixed codes. Diagnostics
  are bounded constants/local descriptions and do not quote document bytes.
- A ResponseDecoder.Write error is sticky and terminal immediately, with no
  publishable HTML. The runtime stdout wrapper cancels the request as soon as
  it receives that error, returning it to stop the exec copy loop. Do not wait
  for Finish to discover a known overflow/malformed frame. Invalid UTF8 split
  across writes must be handled incrementally or checked at complete payload;
  no malformed bytes can ever become a successful result.

No partial success: malformed/truncated/extra output, nonzero exit, unexpected
stderr, invalid status/identity/version/warnings or exceeded limits fail the
entire result. Worker-declared coherent error has no HTML and may include valid
bounded warnings. Decoder owns one result buffer and transfers it without
another full response copy. Success needs complete payload, EOF and exit0.

The worker imports `os` only for inherited stdio and Exit. It receives no paths,
reads no manifest/files, resolves no links and starts no child/network request.
It validates the request, calls existing Parse/ExportHTMLChecked and exits.
Partial parses refuse export. Map signed parser warning offsets only after
checking their range. Errors must not echo raw document content.

The current exporter creates its full HTML with strings.Builder;64MiB is a
post-render output refusal cap, not a peak-memory guarantee or streaming claim.
Snapshot/LenBytes capture happens on UI; background writing consumes immutable
Snapshot.WriteTo through an exact-size writer. No live Buffer access or full
Buffer.String in the UI export click path. Do not expand parser limits.

## Fixed bundle and process lifecycle

Relative to the actual host executable:

```
engines/markdown-export/manifest.json
engines/markdown-export/levnote-markdown-export.exe
```

Fixed catalog includes ID/path/protocol/format/GOOS/GOARCH. Manifest schema1 is
bounded16KiB and records exact engine version, filename, executable size and
SHA256. Executable cap64MiB. No PATH lookup, shell, user-selected executable,
user-plugin-directory search or unverified fallback. Verify before each launch,
using bounded reads and a locked regular-file handle held through launch/run;
reject symlink/junction/reparse redirection in the bundle path. Hash adjacent
manifest agreement proves pair consistency, not publisher authenticity or
protection against someone who can replace both.
Hold/check the fixed bundle directory handles as well as the executable when
claiming ordinary check-to-launch replacement protection; a pathname-only
directory check followed by launch leaves an avoidable rename/reparse race.

Create a non-inherited KILL_ON_JOB_CLOSE job; start only the fixed worker with
hidden/no-console Windows flags and minimal explicit environment. Assign its
exact os.Process.WithHandle to the job before sending any request bytes.
Assignment failure kills the exact child and calls Wait. No uncontained retry.
Known worker waits for its framed input/EOF during the small Start/assign gap;
do not claim this is a security boundary for hostile executable code.

Default whole request timeout30s, earlier caller cancellation wins; exec pipe
drain WaitDelay2s. Bounded stdout/stderr writers never block waiting for UI or
disk; cancel on malformed/overflow output. Every successful Start gets exactly
one Wait. Join source writer, exec copy loops and owned cleanup before freeing
admission or publishing. Close cancels, prevents new admission and joins without
depending on UI callbacks. Use exact owned handles/PIDs only; no name-based kill.
Job ownership/termination/closing must be race-safe. Report cleanup failures.
Non-Windows builds may explicitly return unsupported; no unverified generic
launch or cross-platform acceptance claim.

Process boundary and job lifetime management are not a sandbox: worker still
has ordinary user OS privileges. No integrity/sandbox/copyright immunity claim.

## App decisions

One app export job with generation, originating Tab/Buffer, captured source
version, capability/name and immutable snapshot. Result bytes belong to a
clearable job slot; queued UI callbacks capture job identity, not a huge result.
Worker completion/cleanup never waits for UI dispatch or a modal dialog.

1. Existing Flush first; refuse a foreign text group or oversized source.
   Readonly text may export. Capture identities and launch off UI.
2. Show plain `正在导出…` and `取消导出`; busy rejects duplicate requests.
   Tab switch, successful originating-tab removal, Buffer replacement/reload,
   capability disable/reconfiguration or accepted window close cancels and
   invalidates. Same-buffer mode switch may continue after its ordinary Flush.
3. Joined completion rechecks generation, active Tab, Buffer, enabled capability
   and source version. Flush again before SaveDialog to catch pending fields/
   composition; recheck version afterward. Changed source including edit+undo
   refuses visibly. Failed Flush retains exact draft/recovery state.
4. Native SaveDialog only after successful joined worker. Keep busy during it.
   Recheck all identities/version and Flush after dialog because native dialogs
   can run nested events. Cancel/error/stale must never publish.
5. Use existing synchronous `document.WriteNewFile` only after final checks.
   This exclusive atomic publication completes/fails before another UI action;
   cancellation after publication starts is unavailable, and disk latency still
   blocks UI in this finite slice. No async publication framework here.
6. Report result path/warnings; never change source dirty/saved baseline,
   selection or history. Failure has no hidden in-process export fallback.
   Existing View.Export stays as reference API and for non-worker adapters.
7. Close prompts defer result dialogs. Canceled close restores operation; only
   accepted close cancels/joins before disposal. Shutdown idempotently joins
   runner and clears pending result slots, with no UI-delivery dependency.

Concurrent/idempotent Runner.Close callers all wait for the same cleanup-done
barrier and return its stored outcome; setting closed does not justify an early
return. Admission and Close serialize so no accepted Export escapes the join.

Root app-seam review resolutions before app activation:
- An advertised engine works from source mode even before a View is mounted;
  the old exportCurrent early View-nil guard must not suppress this route.
  Keep non-engine adapter behavior unchanged except for demonstrated defects.
- Successful active-tab changes include Select, addDocument and recoverDraft.
  Successful removal, SaveAs format reconfiguration, Reload/Revert/reinterpret
  and plugin reconfiguration are covered too. Failed Flush/failed mutation
  cannot cancel a still-valid operation merely because it was attempted.
  Completion guards remain necessary even with these cancellation hooks.
- Both CloseTab and CanClose prompts defer export delivery. Modal nesting must
  not open a competing export SaveDialog while a close decision is outstanding.
  A canceled close resumes pending delivery after the prompt unwinds. A close
  accepted only after persistence succeeds cancels/joins before disposal.
- Export slot/result/done coordination must be race-safe. Background worker
  may store/take/clear only its protected result slot; App/Tab/UI fields stay on
  UI thread. Shutdown clears retained HTML without waiting for UI dispatch.
  Do not use updateWait for process cleanup. Deterministic tests control queued
  completion and nested dialog events without arbitrary timing sleeps.

Plugin panel adds a small cached status under the Markdown format: bundled
external HTML export, native in-process editing/preview. Before first use say
not checked/validated on export; afterward show last verified version/hash prefix
and result or error. Do not hash/start/scan every frame. No separate install
button, new persisted engine toggle or unnecessary status-refresh API. Existing
Markdown enable/disable governs this capability. Technical details belong in
this panel/tooltips, not the normal writing flow.

## Acceptance

Pure owner checks independent frame bytes, fragmented transport, truncation,
extra bytes, huge declared lengths, JSON/UTF8/identity/status/error/warning
limits and HTML/warning parity against independent fixtures plus current export
for empty/CRLF/Unicode/BOM-decoded/tasks/tables/links/doc-block/Partial/size cases.

Runtime owner checks manifest/path/hash failures before Start, exact WriterTo
size, one admission, malformed/flood output, stderr, nonzero exit, no-input child,
cancel/timeout/Close, attach failure, descendant cleanup and no leaked owned
work. Repository-built hidden helper children only. Save actual evidence and
owned process identity. An unrelated sentinel must survive. Do not fake delays
or ten-sample idle measurements for a process intended to exit immediately.

App owner checks actual descriptor routing, no fallback, Flush/failure recovery,
readonly/conflict, source/preview/split, late completion and nested dialog races,
tab/plugin/close behavior and exclusive publishing with deterministic seams.
Use final narrow light/dark screenshots for native progress/status where useful.

After focused source/tests/review, root freezes product and authorizes one sole
integrator full test/vet/package plus actual bundled end-to-end export and
isolated startup/owned shutdown. Each attempt gets a fresh evidence directory;
preserve prior accepted executable and failure streams. No benchmark/resource
matrix unless separately planned. One-shot elapsed/CPU/byte/cleanup observations
are scoped evidence, not proof of a memory reduction or a Tauri comparison.

The explicit packaging script builds BOTH current host and worker into a fresh
artifact bundle, then hashes/generates the manifest. It must not stage an
unspecified older host executable. Ordinary host-only go build remains valid:
its cached status starts unverified and its first export visibly refuses a
missing bundle, with no silent fallback or build side effect.
