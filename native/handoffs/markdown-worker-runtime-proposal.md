# Markdown export worker — proposed protocol and Windows host runtime

Status: **DESIGN ONLY / NOT ACTIVATED**. Owner `/root/astra_marker_view`.
This file proposes the smallest repository-owned one-shot HTML export seam.
No worker product/test/script implementation, test execution, build, process
launch or resource measurement was performed. Graphview is frozen again after
the separately authorized SVG correction recorded in `mindmap-markers-view.md`.

This proposal replaces the multi-message handshake/chunk sequence and the
per-request ten-sample idle measurement suggestion in `next-checkpoint-design.md`
for this slice. GLM owns that older document's cleanup. Root approval is required
to activate any implementation ownership below; the document itself is not an
activation.

## Observed repository seams

- `app/export.go:exportCurrent` currently Flushes, synchronously calls
  `View.Export`, then calls native SaveDialog and `writeExportFile`.
- `modules/markdownview/view.go:Export` creates a temporary parse of the current
  buffer and calls `formats/markdown.ExportHTMLChecked`. It must not replace the
  view's active parse/draft identities. Its checked exporter rejects `Partial`.
- `formats/markdown.MaxParseBytes` is the existing8MiB parser cap. The parser
  also bounds lines65,536, blocks8,192, list items, inline spans16,384, table
  descriptor cells16,384 and warnings256. No cap enlargement is proposed.
- `ExportHTMLChecked` uses a `strings.Builder` and constructs the full HTML.
  This proposal does **not** describe it as an incremental/capped-allocation
  renderer. A wire/output cap can reject the complete result; it does not cap
  the builder's peak memory. The existing finite parser/source limits still
  constrain the current producer.
- `text.Buffer` is UI-owned. `text.Snapshot` shares immutable chunks, and its
  `WriteTo(io.Writer)` is explicitly safe for background readers. Capture Size
  and Snapshot on the UI thread; do not call `Buffer.String` in the export UI
  handler or read a live Buffer from the process runner.
- `app.update` rechecks shutdown inside its queued callback. `updateWait` can
  wait for an already-started callback; it must not be used to make process
  cleanup depend on a SaveDialog callback. `Shutdown` already cancels and joins
  tree/disk workers. `CanClose` only removes tabs after all decisions and quit
  persistence succeed. A canceled close leaves tabs intact.
- `document.WriteNewFile` already publishes via a same-directory temporary
  file, sync/close and a Windows no-replace move. It rejects an existing target
  including a target created during the dialog race. Keep this host boundary.
- Bundled Go1.27.3's `os.Process.WithHandle` exposes the exact live Windows
  process handle until its callback returns. `exec.Cmd` supports
  `CommandContext`, `Cancel`, `WaitDelay` and hidden Windows startup.
  `Cmd.Wait` must run exactly once after every successful Start and joins its
  own stdout/stderr copy loops. `WaitDelay` cannot rescue a custom Writer that
  blocks inside Write, so all capture writers below are nonblocking.

## Finite product decision

Ship only `levnote.markdown-export`, repository-built alongside the host. It
accepts UTF8 Markdown bytes and returns HTML bytes plus typed warnings. It does
not load files, resolve image/link/doc-block targets, accept destinations, spawn
other programs, open sockets or host UI. It imports the existing pure Markdown
parser/exporter, not MyGo, the app or plugin installation. The worker may import
`os` solely for inherited `Stdin`/`Stdout`/`Stderr` and `Exit`; it does not call
path-based file APIs, inspect its working directory or read a manifest. Only the
host reads the bundle manifest and executable. References remain whatever the
existing safe HTML renderer emits; no new remote rendering, CSS asset retrieval
or file embedding is added.

The native app continues editing, drawing and selecting files. The actual
worker-enabled UI export path never calls `View.Export` as a fallback after a
missing/mismatched/crashed/refused worker. Keep the existing in-process
`View.Export` as the pure reference/test API and for adapters without an engine
capability (the graph SVG path remains unchanged). This is not an installed
executable plugin SDK or a new rich-view engine.

Only one export is admitted per App/Runner at a time. There is no hidden queue,
coalescing, restart loop, warm process, heartbeat, polling timer or resident
worker. Export shows `正在导出…` and a `取消导出` action; a second request is visibly
busy until the first request has been canceled/completed **and joined**.

## Stable package/API proposal for independent owners

### `core/engineipc` — GLM pure owner

Stdlib-only protocol package; no imports of UI, runtime, process, files, plugins
or the Markdown implementation. Own every wire constant/struct/validator in
one place, including `ProtocolVersion=1` and the fixed engine identifier.

```go
type EngineID string
const MarkdownHTML EngineID = "levnote.markdown-export"

type RequestMeta struct {
    Engine EngineID `json:"engine"`
    Format string   `json:"format"` // exactly "html"
}
type Warning struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Start   uint32 `json:"start"` // source UTF8 byte offset, inclusive
    End     uint32 `json:"end"`   // exclusive
}
type EngineError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}
type ResponseMeta struct {
    Engine        EngineID     `json:"engine"`
    EngineVersion string       `json:"engineVersion"`
    Format        string       `json:"format"`
    Status        string       `json:"status"` // "ok" or "error"
    Warnings      []Warning    `json:"warnings"`
    Error         *EngineError `json:"error,omitempty"`
}
```

Proposed callable surface (implementation may use private helpers):

```go
// Construct bounded request prefix before process launch; source follows it.
func EncodeRequestPrefix(meta RequestMeta, sourceBytes uint64) ([]byte, error)

// Worker: validate header before allocating, then metadata/source and exact EOF.
func ReadRequest(r io.Reader) (RequestMeta, []byte, error)
func WriteResponse(w io.Writer, meta ResponseMeta, html []byte) error

// Host's os/exec stdout writer: accepts arbitrarily divided transport writes,
// validates header/meta before payload allocation, never blocks inside Write.
type ResponseDecoder struct { /* private bounded state */ }
func NewResponseDecoder(engine EngineID, version string, sourceBytes uint64) *ResponseDecoder
func (d *ResponseDecoder) Write(p []byte) (int, error)
// Call only after stdout EOF has been established by the runner's completed Wait.
func (d *ResponseDecoder) Finish() (ResponseMeta, []byte, error)
```

`Finish` transfers the owned HTML slice; do not duplicate the whole response.
An error leaves no publishable HTML. The decoder only stores one prefix,
bounded metadata and at most the declared bounded HTML, not a second complete
framed response. GLM's focused package tests assert that the protocol's8MiB
cap equals `markdown.MaxParseBytes`; that equality is a test, not a runtime
dependency from the pure protocol to one format.

### `core/engineruntime` — Astra host runtime owner

No app/MyGo/live Buffer imports. No public executable-path/argument/env factory.

```go
type Source struct {
    Size int64
    Data io.WriterTo // immutable snapshot; WriteTo must not block except on dst
}
type EngineInfo struct {
    ID      engineipc.EngineID
    Version string
    SHA256  string
}
type Result struct {
    HTML     []byte
    Warnings []engineipc.Warning
    Engine   EngineInfo
}
type FailureKind string
type Failure struct {
    Kind   FailureKind
    Code   string // bounded worker code when Kind == "engine"
    Detail string // bounded local diagnostic, never raw document content
}
func (*Failure) Error() string
func (*Failure) Unwrap() error // preserves cancellation/deadline cause as applicable

type Runner struct { /* private admission, cancellation and lifecycle */ }
func NewBundled() (*Runner, error)
func (r *Runner) Export(ctx context.Context, id engineipc.EngineID, src Source) (Result, error)
func (r *Runner) Close() error
```

`NewBundled` resolves the fixed bundle location without starting a process.
It does not require the worker to be present at App startup; missing/integrity
errors belong to an attempted export/status refresh, so the native editor stays
usable. Unknown engine IDs are refused, never interpreted as paths. `Export`
admits at most one call, rejects busy/closed, and returns only after process
Wait, request-writer join, pipe cleanup and Job cleanup. A coherent engine
failure may return validated Warnings with an error; its HTML is always nil.

Failure kinds: `busy`, `closed`, `unavailable`, `integrity`, `source-limit`,
`source-size`, `protocol`, `stdout-limit`, `stderr-limit`, `worker-exit`,
`unexpected-stderr`, `engine`, `canceled`, `timeout`, `cleanup`.
Host-local failures do not invent worker warnings. A public Limit field or
caller-specified timeout is unnecessary for v1:30s whole-run timeout, a shorter
caller context wins, and a2s exec WaitDelay. A private test seam can shorten
durations and inject fixed repository test children; no production setting
admits an arbitrary command.

`Close` atomically prevents admission, cancels an active export, and waits for
its done signal. Done means cleanup completed independently of UI dispatch.
It is idempotent; it never abandons a started child simply because a UI callback
or a second cleanup timer expired. A cleanup error is visible, not a claim that
the child was reaped. Windows termination normally makes the2s drain bound
effective; no Go timeout guarantees recovery from a broken/unresponsive OS API.

### `editor.Adapter` metadata and app — app owner

Small additive metadata only; no UI contract import of the engine runtime.

```go
type ExportEngine struct {
    ID     string
    Format string
}
// Existing Adapter fields remain; only Markdown assigns this optional value.
ExportEngine *ExportEngine
```

Markdown's built-in Adapter assigns `{ID:"levnote.markdown-export",Format:"html"}`.
App routes an advertised format through this capability and the fixed runtime
catalog, not by importing `modules/markdownview` or checking extension strings.
The format must also be in the adapter's existing Exports list. A descriptor
mismatch is a visible configuration failure. Existing disabled-built-in plugin
selection still removes that rich capability; installed schema1/2 plugins
remain declarative and cannot create this descriptor or a process catalog entry.
If the plugin/capability is disabled while a request is active, invalidate and
cancel the request; a late result cannot restore an export capability.

## One request frame / one response frame

Both directions use exactly this24byte little-endian header:

| Offset | Bytes | Meaning |
| --- | ---: | --- |
| 0 | 8 | request magic `LVEXREQ\0` or response magic `LVEXRES\0` |
| 8 | 2 | protocol version1 |
| 10 | 2 | header byte count24; other values refused |
| 12 | 4 | UTF8 JSON metadata byte length |
| 16 | 8 | raw payload byte length |

The frame is `header || metadata || raw payload || EOF`. Request payload is
UTF8 Markdown; response payload is UTF8 HTML on success and zero bytes on error.
There is no Initialize/Begin/Chunk/End session, ID negotiation, second request,
compression, base64, JSON-escaped source or handshake. The response's engine ID,
version, format and the header protocol must match the validated manifest and
request. App request generation/Tab identity remains host-local because a child
serves exactly one request.

Limits are checked before length-to-int conversion, addition or allocation:

| Item | Maximum / rule |
| --- | --- |
| Request metadata | 4KiB, exactly one JSON object, known fields/types only |
| Source payload | existing8MiB inclusive; valid UTF8, zero-length source legal |
| Response metadata | 1MiB, exactly one JSON object, known fields/types only |
| HTML payload | 64MiB inclusive; valid UTF8; never partial success |
| Warnings | 256 entries, code64bytes, message512UTF8bytes each |
| Warning ranges | `0 <= Start <= End <= submitted Source.Size` |
| EngineVersion | 64 printable UTF8bytes; exact manifest match |
| Error message | 1,024UTF8bytes; error code is a fixed bounded enum |
| Captured stderr | 16KiB total; crossing the cap cancels and fails the request |

JSON decoding rejects duplicate keys at every object level, unknown or
case-mismatched field names, missing required fields, trailing JSON, invalid
UTF8 and wrong scalar types. Validate the bounded metadata bytes with
`utf8.Valid` before decoding; `encoding/json` alone would replace invalid bytes
and is insufficient. Reject unpaired escaped UTF16 surrogates instead of
accepting replacement characters. Required key presence, exact allowed keys
and duplicate detection must be explicit; `DisallowUnknownFields` alone is
insufficient. The protocol does not use unbounded `io.ReadAll`, Scanner defaults,
maps holding arbitrary extra fields or a caller-provided limit.
Metadata's1MiB ceiling accommodates the worst JSON escaping of256warnings with
the stated field limits, rather than silently truncating warning messages/count.
All cap failures are explicit; valid warning/source text is not shortened on
the wire. The UI can show a clearly abbreviated diagnostic while retaining a
bounded error record.

64MiB is a deliberately conservative result refusal cap. Current escaping
expands a source byte by at most5x;8MiB contributes at most40MiB of escaped
source. Existing bounded block/list/table/inline wrappers and at most8,192
heading anchors (bounded256byte base plus uniqueness suffix) have substantial
remaining headroom. This is a sizing rationale, not a substitute for the actual
post-render cap check or a promise about future renderer changes. Test against
the largest accepted escaping/wrapper fixtures, and refuse if output exceeds
64MiB. Do not rewrite the renderer as streaming in this slice merely to claim a
stronger bound. A later streaming writer would be a separate parser/export API
change.

Worker `main` reads the bounded request, verifies exact EOF, validates UTF8,
parses and calls `ExportHTMLChecked`. `Partial` returns status error/code
`partial-document`, validated existing warnings and no HTML. Other error codes
are `invalid-request`, `source-limit`, `invalid-utf8`, `output-limit`,
`warning-limit`, `render-failed`. Known coherent error responses exit0; crashes
or incomplete/protocol-write failures exit nonzero. Successful responses must
be complete and exit0. Any malformed/truncated/extra stdout, invalid warning
metadata, unexpected stderr on an otherwise successful run, or nonzero exit
refuses the entire result even if an earlier prefix looked like valid HTML.
Protocol output is stdout only; diagnostic stderr is separate and bounded.

The worker does not serialize its source or echo it in error messages. It exits
after its one response. EOF before a complete request ends it without parsing;
this also makes a host dying before transmission a finite child lifetime.

## Fixed bundle manifest and integrity

Proposed release layout, relative to the actual running host executable:

```text
levnote-native.exe
engines/markdown-export/manifest.json
engines/markdown-export/levnote-markdown-export.exe
```

The runtime catalog contains the literal ID, directory, filename, protocol,
format and expected host GOOS/GOARCH. Manifest fields are schema1, engine ID,
engineVersion, protocol1, format`html`, GOOS/GOARCH, fixed filename, executable
byte size and64hex SHA256. Manifest maximum16KiB; executable maximum64MiB.
Do not let JSON contribute an executable path, arguments, working directory,
environment, DLL or capability list beyond those fixed allowed values.

Before every Start, read manifest within its cap and hash the regular executable
incrementally using a small buffer. Check exact size, digest, ID, protocol,
architecture, version and capability. No PATH lookup, shell, working-directory
search, user plugin directory, network update or arbitrary executable picker.
Missing manifest/executable, malformed metadata, incompatible protocol and
digest mismatch have distinct visible failure details. Manifest/status reading
does not happen in every UI frame.

Windows verification opens the executable at the fixed absolute bundle path
with a read handle denying write/delete sharing, validates the regular/non-
reparse file and resolved expected location, computes SHA256 from that handle,
and holds it through Start (or preferably the run). This prevents ordinary
replacement between the integrity check and launch. Resolve/check the bundle
directory too; reject unexpected symlink/junction/reparse redirection rather
than accepting an executable from a workspace. An install location that cannot
provide those guarantees yields an explicit launch refusal; no unlocked retry.

An adjacent mutable manifest plus digest proves **consistency of the bundled
pair**, not authenticity against someone who can replace both. It is not code
signing, a trusted publisher chain or a sandbox. The worker still has the user's
normal OS privileges. The no-file/no-network/no-child behavior is repository
code and protocol scope, not a claim that the OS prevents arbitrary user code
from doing those things. The Windows Job below is lifetime management.

Packaging is a separate explicit script: build the worker first with a fixed
build-version value, hash it, generate its manifest, then stage the host and
worker bundle together under a new artifact directory. The current accepted
binary/bundle stays preserved. Do not modify the normal `go.ps1` verification
wrapper or silently add build side effects to it. No signing/update system is
introduced.

## Windows run sequence and cancellation

Use the existing stdlib process facilities, plus a small Windows-only Job helper;
no vendor or dependency patch is needed. The important Go1.27.3 capability is
`cmd.Process.WithHandle`, avoiding an OpenProcess-by-PID race.

1. Atomically acquire the Runner's single admission. Derive a30s timeout from
   the caller context. Reject missing source, negative/over-limit Size before
   reading the source or launching anything. Encode the small request prefix.
   Validate the bundle while honoring cancellation between hash reads.
2. Create a non-inherited Windows Job with
   `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`; no breakaway flags. Failure refuses
   before Start. Prepare `exec.CommandContext(ctx, absoluteFixedExe,
   "--stdio-v1")`, fixed working directory at the bundle engine directory,
   `HideWindow:true`, `CREATE_NO_WINDOW`, `WaitDelay:2*time.Second` and a small
   explicit Windows environment (SystemRoot/WINDIR as needed, no inherited
   GODEBUG/profiling/plugin configuration or unrelated secrets).
3. Obtain `StdinPipe`; install the nonblocking bounded ResponseDecoder as Stdout
   and a nonblocking16KiB bounded stderr sink. The stdout/stderr adapters cancel
   the run on malformed data/overflow. They never wait for UI, channels or disk
   in Write. `Cmd.Wait` owns joining their os/exec copy goroutines.
4. Start. On failure, close stdin, Job and verification handles; Wait is not
   called because no child started. On success, record owned PID only for
   evidence and use `Process.WithHandle` immediately to
   `AssignProcessToJobObject`. Do not start Wait or send request bytes until
   assignment succeeds. If WithHandle/assignment fails: terminate the exact
   started process, close stdin, call Wait exactly once and release everything
   before returning the visible error. No uncontained retry.
5. After successful assignment, one request writer goroutine writes prefix,
   then `Snapshot.WriteTo` through an exact-size counting writer. Refuse a write
   crossing declared Size before forwarding its excess bytes; a short source
   or WriterTo failure cancels. Close stdin exactly once on completion so the
   worker can verify request EOF. No full source flatten/string copy is needed
   in the host. The immutable snapshot remains owned only until this writer
   exits.
6. One coordinator calls `Cmd.Wait` exactly once. Cancellation/user cancel/
   timeout first terminates the Job (and uses the precise Process.Kill as a
   fallback), closes stdin to release a blocked source write, and lets Wait
   finish. The exec Cancel callback must not call Wait. `WaitDelay` bounds pipe
   draining after cancellation/exit; it is not a timer to abandon a goroutine.
   Coordinate close/terminate with `sync.Once` or equivalent; do not close a
   Job handle while another goroutine can still use that raw handle.
7. Join the request writer as well as Wait. A valid decoder Finish requires
   complete header/meta/payload and stdout EOF (the successful completed copy
   loop). Success additionally requires exit0, no stderr, no cancellation and
   no writer/cleanup error. Check context after Wait to give an accepted cancel
   precedence over a completion racing it. Only then transfer result ownership.
8. In every started path, close/terminate the per-run Job and close every owned
   pipe/hash handle. The Job kills any accidental descendants, including ones
   retaining inherited pipes. Release the admission only after all owned work
   joined. Report a cleanup failure explicitly; never publish on that path.

If the host dies after Job assignment, non-inheritance plus KILL_ON_JOB_CLOSE
ends the child tree. Before assignment there is a small Start/attach interval:
the known repository worker only blocks on the unopened request and exits on
EOF when the parent closes/dies. It cannot begin parsing before the gate.
This closes the ordinary one-shot lifecycle without a suspended-thread/custom
CreateProcess implementation. It is **not** a guarantee against a malicious
replacement executable escaping during that interval; arbitrary/untrusted
executable execution remains out of scope. No name-based process killing or
machine-wide cleanup is permitted.

## App snapshot, late completion, dialogs and close decisions

Proposed local job record: monotonically increasing generation, Tab ID,
originating Buffer pointer, source Version/byte count, copied capability and
display name, cancellation, done and a mutex-protected optional bounded result.
Background code consumes only the captured Snapshot/Size and runner, not Tab,
Buffer, View, App slices or settings. It stores the bounded result and schedules
`a.update` with only the job identity; do not trap the full output in a queued
closure that shutdown cannot clear. Job done closes after runner cleanup,
without waiting for that callback.

The concrete minimal UI policy is conservative: **export must still describe
the current originating tab's captured source when SaveDialog is opened and
when the host publishes it**. Editing is allowed while the worker runs, but an
older result is discarded with `导出期间内容已变化，请重新导出`; do not silently
export an earlier revision or restart a new worker behind the user's back.

1. On click, validate the current enabled adapter/capability, run existing
   `flushTab(t,"导出")`, and reject a still-open Buffer group. Check LenBytes
   before Snapshot. Capture the identities and immutable Snapshot on the UI
   thread, then run Export in the single job goroutine. Readonly documents may
   export. Existing conflict behavior is preserved: this is an in-memory export
   to a new file, not an overwrite of the conflicted source.
2. Tab switch cancels/invalidates the pending export so completion cannot open
   a modal dialog for an inactive tab. Source/preview/split changes may continue
   if the same Buffer/capability remains; ordinary mode-switch Flush still
   applies. Buffer replacement/reload, capability/plugin disable and actual
   origin-tab removal invalidate and cancel. A requested tab close that the
   user cancels does not remove the tab or force an export cancellation.
3. On completion, reject shutdown, invalid generation, removed/replaced tab,
   different active tab, Buffer identity or enabled capability before any
   dialog. A source version change, including edit-then-undo, deliberately
   refuses; no full content comparison is necessary. Before SaveDialog run
   `flushTab(t,"完成导出")` again to catch a new pending native composition/
   field draft that has not changed the host Buffer yet, then recheck Version.
   A failed Flush retains exact draft/recovery state, drops this export result
   and visibly explains the failure. It never asks a background goroutine to
   flush a view.
4. Only a completed/joined successful worker can reach native SaveDialog. Its
   default name uses the captured display name and `.html`; no destination was
   passed to the worker. Keep the job marked busy while the dialog is open.
   Dialog cancel releases result bytes and reports a benign canceled status.
5. Native dialogs can run nested event loops. After SaveDialog returns, recheck
   generation, active Tab/Buffer/capability, shutdown/closing, Flush and Version
   again before `writeExportFile`. An accepted cancel/close during the dialog
   prevents publication. Do not put SaveDialog inside an `updateWait` handshake
   that Shutdown must await.
6. Publication remains the existing synchronous host `WriteNewFile` boundary
   in this slice. Once that final write starts on the UI thread it completes or
   fails before another UI close/cancel is accepted; do not claim cancellation
   of an in-progress filesystem write. This slice removes parse/render CPU from
   UI; it does not solve arbitrary filesystem latency. Export never marks the
   source saved, changes source selection/undo or silently overwrites a file.
7. On successful publication, report exact output path and validated warning
   count/details through the host's bounded issue UI. Warnings are not hidden
   because the child exited0. On renderer/protocol/timeout/integrity failure,
   report the typed reason and do not show SaveDialog or call the publisher.
   User cancel/tab-switch cancel are status outcomes, not crash diagnostics.

Close handling needs a small `closing`/export-delivery deferral flag during
`CanClose`/native close prompts, not early `shuttingDown=true`. If any tab's
close/Flush/save/persistence fails or user chooses Cancel, clear that flag and
keep normal app operation; a completed export can be reconsidered afterward
with all guards. Once all close decisions and quit persistence succeed, cancel
and join export before final disposal/accepted close. `removeTab` cancels only
an actually removed origin tab. `Shutdown` idempotently calls Runner.Close
as a defensive last barrier and clears any pending result slot before queued
callbacks can run. Closing never waits for UI result delivery and never leaves
a child registered only with a disappearing view.

## Exact proposed ownership after root activation

| Owner | Exclusive files / responsibility |
| --- | --- |
| GLM protocol/format | new `native/core/engineipc/{protocol.go,decode.go,protocol_test.go}` and focused protocol fixtures; new `native/cmd/levnote-markdown-export/{main.go,main_test.go}`; worker maps existing Markdown warnings/errors. Existing parser/export stays unchanged unless root approves a demonstrated defect. |
| Astra runtime/packaging | new `native/core/engineruntime/{runner.go,bundle.go,process_windows.go,process_other.go,*_test.go}` and repository helper child fixtures; new explicit `native/scripts/package-markdown-worker.ps1`; per-run Job/stdin/stdout/stderr/Wait cleanup. No change to `go.ps1`, vendor, app or protocol-owned files. |
| GLM app/integration | `native/app/export.go`, a new `native/app/export_worker.go` plus focused app tests, necessary cancellation hooks in app lifecycle/UI/plugin changes; additive `native/ui/editor/contract.go` ExportEngine metadata; exactly one descriptor assignment in `native/modules/markdownview/view.go`; final product docs and sole full integration after root's focused gate. |
| Root | approve stable protocol/API/limits, independently inspect source/evidence and authorize integration; no implementation or test/build work. |

Names above are intended as the shared contract. GLM must publish protocol
constants/signatures first; Astra must publish Runner lifecycle/API before app
integration. An owner discovering a required peer change sends a request to
root, not an edit. Initial Windows-only runtime can have an explicit unsupported
platform stub for other GOOS values; it must not silently use an unverified
generic launch on another platform.

## Validation plan — commands are proposed, not executed

1. **Pure protocol/worker owner:** independent header byte oracles; segmented
   reads/writes including1byte fragments; bad magic/version/header length,
   duplicate/unknown JSON keys, truncated/extra bytes, overflow lengths before
   allocation, wrong identity/version, invalidUTF8, illegal status/payload and
   warning limits/ranges. Exact writer Size mismatch. Valid empty document.
   Worker HTML and warnings compared to independent expected fixtures and the
   existing checked exporter for BOM-decoded source, CRLF, Unicode, tasks,
   tables, raw HTML, safe/unsafe links, doc-blocks, Partial and8MiB/8MiB+1.
2. **Runtime owner:** fixed catalog/path/manifest/hash/missing/wrong-size/version
   failures before Start; exact raw Snapshot WriterTo and no UI flatten;
   one active call/no queue; stdout fragmentation/extra/flood/truncation;
   nonzero exit with plausible HTML; stderr cap/exit0stderr; worker not reading
   stdin; timeout while parsing; cancel before Start, during writing and racing
   completion; attach failure forces kill+Wait; Wait once; Close joins without
   UI; no retained result/snapshot/pipe/Job handles after finish.
3. **Windows lifecycle acceptance:** only repository-built helper processes in
   isolated fixture directories. Record exact executable digest/owned PID,
   exit, elapsed run and cleanup outcomes. Verify normal, explicit cancel,
   timeout, malformed-output and parent-close paths leave no owned child;
   job-descendant/helper tests use exact owned handles/PIDs and preserve an
   unrelated sentinel. Include parent death during pre-assignment gate and
   after assignment. No broad process-name termination, shell subprocess,
   existing user's file or installed third-party executable.
4. **App owner:** actual registered Markdown descriptor; disabled builtin and
   installed declaration refusal; normal native preview/source/split export;
   pending cell/field/IME Flush; failed draft recovery retained; readonly and
   conflict behavior; source change, source undo, Buffer replacement, tab switch,
   plugin disable and queued late result; native SaveDialog cancel/failure and
   change during nested dialog; accepted/canceled window close; no publish on
   any canceled/stale/failed path; WriteNewFile no-replace race; no source dirty,
   selection, undo or save-baseline changes. Verify no in-process fallback call.
5. **Focused gates then sole integration:** owners run only their packages via
   `native/scripts/go.ps1 test ...` / `vet ...`, save actual outputs, then root
   reviews before the app owner alone packages/builds/full-tests and exercises
   the isolated release. No command in this plan is permission to execute now.

Use one-shot timing evidence: validation/start, request sent, response complete,
Wait/cleanup complete, worker CPU from ProcessState where available, bytes and
actual success/cancel/timeout outcomes. Do not insert sleeps or force ten idle
samples into every request; there is intentionally no persistent worker to
sample at idle. A later explicitly authorized resource comparison may sample
host/worker separately using an appropriate real workload. Do not reuse earlier
binary resource numbers, claim reduced CPU/memory, a Tauri comparison, sandbox
security, OS IME coverage or GPU proof from this protocol slice.
