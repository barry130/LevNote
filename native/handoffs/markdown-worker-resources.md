# Markdown worker process measurements — accepted

Status: COMPLETE, root accepted the actual12request matrix on2026-11-09 after
reading all raw records, RUN/PASS streams, cleanup and aggregate source.
Aggregate `1DE4E8D3682B11FFA6200993C7B186C71F3E0367D2B6A697A3F764F26EE18F2C`
passes8/8. Product, existing tests, packaging scripts and accepted releases
remain unchanged; no new product version or product rebuild. The new Windows
test is opt-in. Primary raw records, aggregate and observation summary are frozen.

## Accepted observations

Evidence directory:
`artifacts/acceptance/markdown-worker-resources-20261109-031432-matrix-attempt02/`.
`aggregate.json` is40203B; `execution-result.json` is21682B SHA256
`DA451511EF75CA4A1C944AA4BFCE2048B727225B1A734FE40BC8D75A58ED9162`.
`summary.md` SHA256 is
`9A27D38E07A09F4815A7D3960A7F175A78CEA33C2945517517FCC660A8B04DC0`.
The summary's original “pending root acceptance” heading records its creation
state; this handoff and the separate `root-acceptance.json` record later acceptance.

Windows10.0 build26200 / amd64, Go1.27.3,16logical CPUs. Fixed alternating order,
three observations per fixture/version,12tester invocations and12real worker
starts. All106memory queries and24GetProcessTimes reads succeeded; final OS peak
counters remained available after exit in every row. Samples per request were
6–12, with actual times/intervals preserved. Every exact held process signaled
exit0; sampler, Export and Close joined, duplicates closed. All275protected
paths and6copies retained size/hash/UTCmtime. Full source/HTML/ordered-warning
identities matched the independent baseline.

| Fixture / version | OS peak working set MiB median [min,max] | OS peak process commit MiB median [min,max] | Runtime elapsed ms median [min,max] |
| --- | --- | --- | --- |
| EscapedFence1MiB .1 | 25.273 [25.250,25.875] | 33.918 [31.570,36.422] | 32.615 [32.113,437.402] |
| EscapedFence1MiB .2 | 15.199 [15.031,15.234] | 21.406 [21.320,21.781] | 33.517 [31.110,579.734] |
| Mixed32 .1 | 7.699 [7.688,7.773] | 13.457 [13.445,13.484] | 22.743 [22.283,26.587] |
| Mixed32 .2 | 7.527 [7.504,7.727] | 13.133 [13.129,13.297] | 21.837 [21.128,23.110] |

Median worker peak working set/commit decreased39.86%/36.89% for the escaped
fixture and2.23%/2.41% for Mixed32. These are local three-sample observations,
not guarantees. First escaped requests437.4022/579.734ms remain in the range;
their causes were not measured. CPU reads were0/15.625/31.25ms: this quantization
does not prove zero work or a useful CPU speed difference. Earlier80%/66.8%B/op
reductions describe cumulative renderer allocation, whereas these process
peaks include source/parser/transport/runtime. No app/UI/tester memory measured.

## Probe and lifetime

`TestBundledWorkerResources` in `core/engineruntime/worker_resources_test.go`
skips before bundle access unless `LEVNOTE_WORKER_RESOURCES=1`. Opt-in also
requires a fixed fixture name (`EscapedFence1MiB` or `Mixed32`), accepted version
(`native-worker.20261109.1` or `.2`), sample ordinal1–3, and a fresh JSON output
path. These inputs never choose a command/executable. The test calls the actual
`NewBundled()` / `Export()` / `Close()` and keeps the ordinary30s timeout.

At the existing `afterStart` hook, before source writing starts, `WithHandle`
duplicates that exact process handle with `DuplicateHandle`. No PID lookup is
used. The probe reads identity and an immediate memory observation, then starts
a ticker with a nominal2ms interval. It neither waits for a sample quota nor
changes worker input, stdout consumption, arguments, environment or lifetime.
At `afterWait`, it stops/joins the sampler and attempts final memory and process
time reads using the duplicate, which remains valid after `os.Process.Wait`.
After `Export` and `Close`, a zero-time wait on the held handle verifies it is
signaled; the handle is closed on every path. Missing hooks/duplication/identity/
join/close failures make the request fail. Final counter-query failures remain
explicit observations, never fabricated zero measurements.

Every memory attempt records its UTC time, monotonic offset and duration,
phase, success/error and raw `PROCESS_MEMORY_COUNTERS_EX` fields. Sampling is
bounded to20,000attempts (40s at nominal cadence); actual scheduling may yield
fewer samples. Reaching the bound is recorded as a probe failure. The observation
window is first-to-last actual attempt, not a promised interval. The report
distinguishes sampled maximum `WorkingSetSize` / `PrivateUsage` from Windows
`PeakWorkingSetSize` / `PeakPagefileUsage`. `PrivateUsage` is current private
commit; peak pagefile usage is the OS peak process commit counter, not disk I/O
or cumulative Go allocations. After-exit memory reads may fail or report a zero
current working set: keep the raw result and use only successful observations.
CPU comes from `GetProcessTimes` kernel/user100ns counts, with process creation/
exit timestamps and lifetime; request elapsed includes verification, transport
and runtime cleanup. Neither tester memory nor UI memory is measured. Do not
sum independently observed maxima.

## Fixed fixtures and independent checks

Reproduce the exact fixture constructors from the accepted baseline harness
`artifacts/acceptance/markdown-export-memory-20261109-023250-baseline/export_memory_bench_test.go`.

| Fixture | Source bytes / SHA256 | HTML bytes / SHA256 | Warnings / SHA256 |
| --- | --- | --- | --- |
| EscapedFence1MiB | 1048585 / `0371420aad5aa78a8f68488a92db63f7ce3ad24cd1c713f507879bbf5ba9dbec` | 5242960 / `3042906ac2b1f579e799b92f5dc9b8a180a28e114177b60852a1cc1dd35fc0d9` | 0 / `4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945` |
| Mixed32 | 9728 / `40ba0b8857ca84e06a5968b7129ffa9eec2875bd1f57c831a4d6680f5c47e221` | 24203 / `30969c1a616107e23cafaf3ba572993be2ea3dad3d5c9f41d7d9350398b9d0a4` | 32 / `60c451822ff9073813ed711901c448f5111d1966416583672ea95dd719ee8017` |

Warning identity normalizes the wire warnings, in original order, to the
baseline JSON shape `Code,Message,Range{Start,End}` using a local test-only
structure; no renderer/parser is used as the oracle. The report also retains
ordered wire warnings. Test fixed accepted worker+manifest hashes before/after,
returned engine version/hash, complete HTML/warning identities, nil Export/
Close errors and signaled held process. Record tester hash, Go/platform/Windows
build/logical CPUs and request metadata; capture probe/source hashes externally.

## Execution gate and finite order

The execution gate required one compiled tester and only the default SKIP path.
Root reviewed method/source and actual compile/SKIP logs before real requests.
Then that exact tester and accepted worker+manifest were copied into two disposable
bundles, leaving user launch directories unchanged. The same tester hash must
appear in both. Each invocation is a fresh tester process with exactly one
request. No concurrent tests/builds, extra warmup or hidden UI launches.

Fixed order: EscapedFence1MiB `.1/1,.2/1,.1/2,.2/2,.1/3,.2/3`, then Mixed32
in the same version/ordinal order. Twelve requests total; stop on the first
request failure, preserve JSON/stdout/stderr/exit status, and report for root
review without retry. Memory-query errors are included in their observation
records; a successful export with an unreadable final memory counter does not
justify an extra request. Raw records precede medians/ranges and conclusions.

This observes the real bundled worker through an instrumented runtime tester,
not the packaged app, UI idle memory, GPU, Tauri baseline or a security sandbox.
Three samples describe local variation only. The earlier80%/66.8%B/op reduction
is a different metric; source/parser/transport allocations remain in this run.

## Probe checkpoint, before matrix

Source reviewed by root; one successful tester compilation and explicit default
SKIP completed; at that checkpoint no real worker request had run. Evidence:
`artifacts/acceptance/markdown-worker-resources-20261109-030917-probe-attempt02/probe-checkpoint.json`
(3240B, SHA256 `B35C9FE2667FD0B94EB7FBC3E8BCABD7B217551CEBD318BA6276B33B14D4294A`).
Compile exit0 wall1.5189532s; SKIP exit0 wall0.6752647s, actual RUN/SKIP/PASS lines
retained, both stderr streams empty. The253frozen source/previous-output files
match their captured fingerprints. Source test20792B SHA256
`2AE77415587CE3D806832637FA03CF77A53F7115271A53FA8BA95B8FDA11FFC9`.
The initial tester was6678016B SHA256
`5AF85C2633AB4E5797993D3A7A401EFA7EB8A82354E97E3DE5100C5219510E4C`;
it is preserved but was superseded by the narrow correction below.

The original attempt in `markdown-worker-resources-20261109-030814-probe`
failed before Go because PowerShell bound `-o` as an ambiguous public parameter.
Its streams remain unchanged. Attempt02 uses an evidence-only helper passing
the explicit `-GoArguments` string array; product scripts were not changed.
The checkpoint records the method's pre-result hash; this appended results
section is not part of that earlier method fingerprint.

The first opted-in invocation (`031108-matrix`) stopped before NewBundled/Export:
the probe's `.1`manifest expectation omitted one `b`, leaving63hex characters.
The actual accepted file and copied file were correct. TesterInvocations1,
RealWorkerStarts0; original FAIL/notes remain unchanged, no measurement inferred.
Root authorized only that constant correction and a no-process actual-file/hash
length preflight. The new preflight passed5/5 (all10literal hashes64hex and all4
accepted payload hashes matched); one new compile/defaultSKIP passed, checkpoint
`0661A416E7F5F0EE04C0D6FD466E274C726E19158EFE8219A1E1559E60156180`.
Corrected source20793B SHA256
`10622514C92B6174E863AA6AC6EF7F2824C2C2339FE53444F0A49270A54ED19C`;
the one tester used for all12real requests is6677504B SHA256
`873EE96C3348E7DE1BD658286169A96D4E5A82FD1C4596BFAB4BD61D019D9705`.
Root then separately activated the corrected full12matrix. No worker retry,
warmup, additional process request or product rebuild followed these12results.
