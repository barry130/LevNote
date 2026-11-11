# Markdown worker process measurements — queued finite stage

Status: MATRIX ACCEPTED, final documentation only. Root read actual raw results,
streams and aggregate script; aggregate
1DE4E8D3682B11FFA6200993C7B186C71F3E0367D2B6A697A3F764F26EE18F2C passes8/8.
All12 real requests completed,106memory/24time queries succeeded, final OS peaks
available throughout, source/HTML/ordered warnings exact, cleanup complete;
275protected files and6copies unchanged. No further process requests or product
rebuild. Preserve raw results and prior failures. Final docs must distinguish
worker peaks from renderer allocation and total app footprint, retain initial
timing ranges/CPU quantization and three-local-sample limitations.

Earlier activation: root accepted the
one-character manifest-hash correction, no-process preflight and actual
compile/defaultSKIP checkpoint
0661A416E7F5F0EE04C0D6FD466E274C726E19158EFE8219A1E1559E60156180.
Corrected source10622514C92B6174E863AA6AC6EF7F2824C2C2339FE53444F0A49270A54ED19C
and tester873EE96C3348E7DE1BD658286169A96D4E5A82FD1C4596BFAB4BD61D019D9705
are frozen. The initial tester invocation failed before any worker launch;
preserve its evidence and old tester. At this activation zero real worker
requests have run. Stop at the first failed request; no unplanned repeat.
Count TesterInvocations separately from RealWorkerStarts. `.2` package and
final docs were accepted earlier (acceptanceC058E887...).
Root plans/reviews only. `astra_marker_view` is the sole execution
owner; no additional agent or child delegation. Existing product, parser,
runtime, protocol, app, package scripts, vendor and all accepted releases stay
unchanged. The objective is to answer the remaining process-memory question
after the accepted allocation optimization, rather than infer RSS from B/op.

## Design checkpoint first

Publish a concise method in `handoffs/markdown-worker-resources.md` before
launching measurements. Scope test-only Windows probe code in
`core/engineruntime/*resources*_test.go`, disposable evidence runners/bundle
copies, and later documentation. Reuse existing private runtime hooks; do not
add public hooks or change production process behavior to make sampling easier.
The test must opt in and skip ordinary tests. All builds/launches are delegated.

Use the identical test executable and fixture bytes against preserved accepted
`.1` and `.2` worker/manifest copies. `NewBundled` still resolves beside that
test executable. The tester uses the actual fixed runtime (verification, Job,
bounded transport and cleanup), not an alternative launcher. It is explicitly
a runtime host-path tester, not the packaged application/UI or a Tauri baseline.
No mock worker, sleeps inside worker, new worker flags, preloading, forced GC,
compiler profiling instrumentation or artificially delayed stdout consumption.

The useful output is per-request elapsed time, process CPU, working set,
private commit and trustworthy peak counters for the real one-shot worker.
Retain exact process objects/handles at the existing afterStart hook before
input starts. Never reopen a recycled PID. A lightweight bounded sampler may
query the held process while it lives. Record actual sample times, failures,
intervals, sample counts and the observation window. Independent process maxima
must never be added and called simultaneous peak. Distinguish sampled maxima
from OS peak working set / peak pagefile usage (private commit) and name those
Windows counters correctly. Do not call either of them cumulative allocations.
If counters cannot be read after exit, preserve that outcome and report only
observed values; do not slow, hold or repeat the worker to fabricate ten samples.
The short request may have few samples. Process CPU/time can still be reported.

For tester-side memory, if measured, label it as the instrumented runtime host
including Go tests, source fixture and sampler, not actual LevNote UI overhead.
The worker's true OS counter is the primary target. Record platform/toolchain,
logical CPU count, artifact hashes, sampler source and tester hashes. No claim
about GPU, interactive app memory, process tree isolation or system-wide memory.

## Fixed, small execution matrix

Reuse the already accepted deterministic fixtures from
`markdown-export-memory-20261109-023250-baseline` benchmark harness: one
1MiB ampersand fence (1,048,585 source bytes) and Mixed32 (9,728 source bytes).
Their exact source/output/warning hashes are in `markdown-export-memory.md`
and focused-checkpoint.json. Keep full byte parity, expected warning order,
version/hash identity and clean runtime completion as independent oracles.

Plan three requests per fixture/version: 2 fixtures × 2 accepted versions ×
3 samples = 12 actual one-shot workers. Alternate versions within each fixture
and record the exact execution order; preserve raw rows rather than select a
best run. One request per tester process gives comparable clean parent setup.
No extra warmup or hidden full-app matrix by default. Tests/builds must not run
concurrently with resource samples. The runner's ordinary 30s deadline remains.

Compile the single test harness once into a new evidence directory, verify it
has the same hash in each disposable bundle, and verify all production payload
copies against accepted `.1`/`.2` release records before and after. No files may
be added to either user-facing release directory. Close retained handles and
join the sampler and Runner on success/error. If any request fails, preserve
its evidence, stop the remaining matrix and report the failure for root review;
do not silently retry or substitute another workload.

## Acceptance and follow-up

After source/method review, root activates the finite matrix. Present raw
records, comparison medians and observed variation, exact source/HTML/warning
identities, actual commands/streams/exit codes, process joins and unchanged
product/release fingerprints. Root reads the evidence before accepting the
measurement. No fixed resource threshold invented after seeing the numbers.

This stage does not rebuild or repackage product code. If the process benefit
is smaller than the renderer allocation reduction, state that plainly; parser,
source and transport copies remain. If a real memory hotspot is demonstrated,
root chooses a separate bounded optimization later. Otherwise advance to the
next product feature rather than repeat measurements for prettier results.
