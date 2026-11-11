# Markdown worker host runtime — active implementation seam

Owner `/root/astra_marker_view`. Marker final docs completed and accepted before
activation. This handoff follows the current `markdown-worker-contract.md`;
older queued proposal text does not activate extra scope. Runtime ownership is
`core/engineruntime/**` and `scripts/package-markdown-worker.ps1`; root also
transferred pure protocol/worker ownership at00:37local. Focused checks are now
complete below. Root subsequently accepted full package execution, actual
host-path exports/refusals and hidden host startup; see
[full integration](markdown-worker-full-integration.md). Focused records below
retain their historical scope; no additional runtime execution was performed
for final asset delivery.

## Stable exported API

```go
type Source struct {
    Size int64
    Data io.WriterTo
}
type EngineInfo struct {
    ID engineipc.EngineID
    Version string
    SHA256 string
}
type Result struct {
    HTML []byte
    Warnings []engineipc.Warning
    Engine EngineInfo
}
type FailureKind string
type Failure struct {
    Kind FailureKind
    Code string
    Detail string
    // private cause for errors.Is cancellation/deadline checks
}
func (*Failure) Error() string
func (*Failure) Unwrap() error
func NewBundled() (*Runner, error)
func (*Runner) Export(context.Context, engineipc.EngineID, Source) (Result, error)
func (*Runner) Close() error
```

FailureKind values: `busy`, `closed`, `unavailable`, `integrity`, `source-limit`,
`source-size`, `protocol`, `stdout-limit`, `stderr-limit`, `worker-exit`,
`unexpected-stderr`, `engine`, `canceled`, `timeout`, `cleanup`. Exported constant
names are `FailureBusy`, `FailureClosed`, etc. EngineInfo is zero before bundle
verification and retained on any later error; error results always have nil
HTML. Worker-declared coherent errors may retain their validated warnings.
Details are bounded and never include raw source/stderr text.

`NewBundled` resolves the running host directory only; it does not require or
inspect a worker bundle. Export verifies the fixed bundle anew. It admits one
operation (no queue); every return follows process Wait, request-writer join,
pipe/Job/file/directory-handle cleanup. Concurrent Close callers all join the
same completed barrier and return the stored cleanup outcome. Close does not
wait for UI callbacks. Source.Data must be an immutable WriterTo that does not
block independently of its destination; `text.Snapshot.WriteTo` satisfies it.

Manifest keys: `schema`, `engine`, `engineVersion`, `protocol`, `format`, `goos`,
`goarch`, `filename`, `size`, `sha256`. Values are fixed schema1/engine ID/
protocol1/html/windows/amd64/fixed worker filename plus bounded version/size/
digest. The release script builds current host AND worker in a new artifact
directory, then writes the manifest from the built worker's actual size/hash.
The worker version is injected with `-X main.engineVersion=<version>`.

## Implementation / validation status

Focused source checkpoint completed2026-11-09local. Runtime package PASS21.868s,
command wall23.016s, exit0, stderr0 in
`artifacts/acceptance/markdown-worker-runtime-20261109-010405-attempt03`.
All three owned packages' vet passed exit0 in1.148s with empty stdout/stderr:
`artifacts/acceptance/markdown-worker-owned-20261109-010550-vet`.
Pure protocol/worker package passes and their bounds are recorded in the
protocol handoff. Exact source and actual evidence hashes are in
`artifacts/acceptance/markdown-worker-focused-checkpoint-20261109-010644/checkpoint.json`
(SHA256`EAA53F06003BD871D442178F7781CB5AE2425269ED8C2E273E0BF1E7BFCBEB50`).

Runtime coverage includes strict bounded manifest/identity/hash checks before
Start; locked executable/manifest writes and directory replacement; coherent
hash/size but non-PE Start refusal retaining EngineInfo; exact snapshot WriterTo
length; single admission/concurrent Close; canceled completion linearization;
malformed/truncated/extra/flood stdout, stderr, nonzero exit; timeout/no-input
child; attach failure with exactly one Wait; natural and canceled descendant
cleanup with inherited pipes; both parent-death gates and an unrelated sentinel.
File symlink rejection was skipped because the environment lacks the required
Windows symlink privilege. The code rejects reparse points, but this focused run
does not establish that skipped branch with a real symlink.

Actual helper was the repository-built test executable,4,527,616bytes,
SHA256`91616dc3aec491549405f223c3168d4ce751b082177d1c333bff94362b3b3394`.
The oracle retained exact process handles and verified membership in the owned
Job before cleanup: descendant PIDs13644/4180/35192 all signaled before Export
returned. Parent-death before attach was parent26012/child32752; after attach
parent28188/child11300. Unrelated sentinel28460 survived both checks, then the
test explicitly killed/waited its own sentinel. This is helper lifecycle
evidence, not production worker bundle or sandbox acceptance.

Failure history is preserved. Attempt01 at005243 failed Go's built-in vet on a
redundant GOOS/GOARCH OR expression; fixed by separating current platform
support from fixed manifest target validation. Attempt02 at005510 passed the
pure/worker packages but failed descendant cleanup: Job ActiveProcesses reached
zero while a known process object was not yet signaled. The runtime now obtains
at most256 Job member handles before first termination (also in Cmd.Cancel),
checks membership, retains them, terminates, observes empty Job and waits those
objects before completion. Enumeration/limit/wait failures report cleanup and
close the runner. Its followup record found recorded PID29956 already absent;
no kill was issued. Attempt03's stronger held-handle assertions all passed.

Package script syntax was parsed successfully; **it has not been executed**.
It builds both current host and worker in a fresh directory with explicit
windows/amd64/v1 target, verifies PE target/subsystem and generates the manifest.
It never overwrites `native/bin`. Accepted marker executable remains13,461,504
bytes, SHA256`17E014EB6F61E671F6FCCA439719E8831B7591042A6F29EE5354960C120C0466`,
UTC2026-11-08T16:04:08.3209774Z. No full test/release bundle, app integration,
resource sampling or independent peer review is claimed. Root later accepted
the focused and full integration checkpoints linked above.

Normal local Windows drive paths are supported; UNC/device paths and reparse
components refuse. Directory/file handles deny ordinary path replacement
through the run. Adjacent manifest agreement is integrity consistency only.
The source WriterTo must not block independently of the destination. Current
renderer builds full HTML before its64MiB output check; no streaming or peak
memory guarantee is made. Cleanup uses a finite process-count and wait bound,
not a security boundary for hostile programs; the production worker spawns none.
