# Resource follow-up — tooling review-ready checkpoint

## FINAL RESOURCE CHECKPOINT — 2026-11-08 17:42 +08:00

Status: **ROOT/GLM REVIEW READY. Original matrix remains FAIL; the separately
authorized clean-shutdown restore continuation passed.** This is evidence for a
hidden startup process, not a claim about visible rendering, GPU paint, or
automatic hidden-live persistence within the declared observation bound.

The accepted resource executable remained frozen at 13,252,608 bytes, SHA-256
`5BFB5A445CC619FEA9380F0363D3DF26EC4F3FF944C8549B1992F5B462EF3F10`.

### Original reviewed matrix

The single authorized matrix started in a hidden outer process at
`2026-11-08T09:35:00.5225236Z` (outer PID 31364) with sampler SHA-256
`0AE9E4E27E5E10FA3C0D0E5F7F1C34BEB86D719AB3591FB9D4816B90B70F42BD`.
The six substantive rows passed with ten one-second observations each and
`WM_CLOSE` exit code 0. Means below are computed from the saved
`samples.observations[]` `workingSet64` and `privateMemorySize64` fields; MiB is
`mean(raw bytes) / 1,048,576`:

| Scenario | n | Mean working set bytes | Mean private bytes | Working set MiB | Private MiB |
| --- | ---: | ---: | ---: | ---: | ---: |
| blank | 10 | 61,292,544 | 90,255,360 | 58.453125000 | 86.074218750 |
| plain | 10 | 64,827,392 | 93,097,984 | 61.824218750 | 88.785156250 |
| Markdown | 10 | 66,883,584 | 94,773,248 | 63.785156250 | 90.382812500 |
| mindmap | 10 | 62,128,128 | 90,771,456 | 59.250000000 | 86.566406250 |
| draw.io | 10 | 61,902,848 | 91,312,128 | 59.035156250 | 87.082031250 |
| ordered four | 10 | 67,637,248 | 96,284,672 | 64.503906250 | 91.824218750 |

CPU delta was zero in every ten-second window. The blank extension reused the
same PID and collected ten three-second observations. The `/...` matrix still
ended in **FAIL**, intentionally and without being rewritten to success:
`clean-restore` expected `session.json` before closing the hidden seed window,
but no session appeared within the declared 20-second pre-close observation
bound. The evidence does not establish whether persistence occurs under other
timing or interaction. The seed then closed gracefully with exit code 0 and
wrote a valid session containing the Markdown path, `preview`, and active
index 0.

Exact evidence:

- `native/artifacts/acceptance/resource-followup-20261108-165230/result.json`
  — 212,222 bytes, SHA-256
  `6EA9BC9370318AFEB82CF406E2CDFC5B639F58F918C4E10A5261EFEF6C78935E`;
- `raw/` contains seven records: six passed samples plus the preserved failed
  clean-restore seed record;
- the immutable copy is
  `matrix-original-fail-20261108-171800/`; its 65-file manifest is 11,887
  bytes, SHA-256
  `3B21BACC4A6073C472DA4A401D19551CE82EF27E95351D8E6B386C9D2E7E0341`;
- the earlier preflight-only attempt, where the wrapped fixture validator wrote
  PASS but the helper observed a transient null exit code and failed closed,
  remains separately preserved in `attempt-01-preflight-exit-null/`.

### Same-app-data clean-restore continuation

The primary continuation at 17:40 did not reseed or repeat any of the six
matrix rows. It validated the session produced by graceful seed close through
the precompiled product loader, launched the same copied executable with no
arguments against the same isolated app-data directory, settled 10 seconds,
collected 10 one-second observations, closed with PID-scoped `WM_CLOSE`,
validated the session again, and rechecked fixture hashes.

Its primary resource means are computed from the ten saved observations:
73,801,728 bytes working set (`70.382812500 MiB`) and 101,376,000 bytes private
(`96.679687500 MiB`). The actual CPU window was 10.142 seconds with zero CPU
delta. The logical-processor denominator was not recorded in that first run;
the same-machine matrix recorded 16 logical processors, but that value is not
retroactively inserted into the 17:40 JSON.

The 17:42 rerun is supplemental, explicitly unplanned, and not another
independent scenario. Its 10 observations averaged 75,214,438.4 working-set
bytes (`71.730078125 MiB`; one of ten values was 75,214,848 while the rest were
75,210,752) and 101,138,432 private bytes (`96.453125000 MiB`). It recorded a
zero CPU delta over 10.182 seconds with an explicit 16-logical-processor
denominator.

- Reload PID 18728; hidden PID-scoped window discovery found
  `轻笺 · 原生版` with `Visible=false`.
- Graceful close: accepted, exit code 0, no forced cleanup.
- Product-loader validation passed both before and after reload with expected
  path, `preview`, active 0, and no load issues.
- `session.json` stayed 382 bytes with SHA-256
  `FDAC9EBEFCCEB693D6AC95AC3BF6ED43459EDD382F85A4CED42D64F7CE6355D4`
  before and after.
- The copied executable matched 13,252,608 bytes and SHA-256
  `5BFB5A445CC619FEA9380F0363D3DF26EC4F3FF944C8549B1992F5B462EF3F10`
  both before and after the continuation.
- All four fixture hashes were unchanged. Zero sampler-owned or current
  `levnote-native` processes remained.

Exact continuation evidence:

- `followup-clean-restore/clean-restore-followup.json` — 25,318 bytes,
  SHA-256
  `ED7F414F4BEFB8C0BD8A6C54AC65EFB34029147E2B8A85B767DCDD95CF733C27`;
- `scripts/run-clean-restore-followup.ps1` — 13,483 bytes, SHA-256
  `3ACD846728BDF13985900688606F99379293AB6B04DE8231B14EC80A9D7FDDBE`;
- `session-before-reload.json` and `session-after-reload.json` are retained
  beside the reload stdout/stderr logs.

The 17:42 rerun was an **unplanned repeat**: the first authorized continuation
had already run and passed at 17:40 with valid observations, but left
`systemLogicalProcessors` null. The delayed review correction arrived after
that launch. Before overwriting anything, the first result and logs were
preserved unchanged under
`followup-clean-restore-attempt-01-no-logical-cpu/`. That preserved result is
24,340 bytes, SHA-256
`1740A7D6548E42C96CA77D390F55C3DB8EC4AD7716B91A297BEE4FD0954601AE`;
its preservation manifest is SHA-256
`E7DEC8C6D30743B4C2ED9953CFF39238080667B480ED68F375938A054FD829C0`.
The corrected run initialized the denominator from
`[Environment]::ProcessorCount` and added executable identity checks before and
after. No launch or sample followed the corrected run.

The two conclusion boundaries are:

1. the original full matrix is a truthful **FAIL** because no pre-close
   session appeared within its declared hidden-window observation bound;
2. clean session persistence on graceful close, product-loader acceptance,
   same-app-data relaunch, measurement, graceful close, and post-close session
   validation passed in the bounded continuation.

No product code, test, vet, build, or compiler invocation was run for this
resource stage.

## Current checkpoint — 2026-11-08 17:12 +08:00

Status: **REVISED TOOLING REVIEW-READY. PROCESS MATRIX NOT RUN.**

The root/GLM review corrections were applied without launching or changing the
product:

- Empty argument lists now omit `Start-Process -ArgumentList`; nonempty
  Windows paths are quoted explicitly for spaces and quotes.
- Every sampler-created app or validator PID is registered immediately.
  Timeouts and exceptions run bounded owned-process cleanup in `finally`.
  Cleanup status, live-owned count and per-process history enter `result.json`.
- External `levnote-native` processes are checked before every launch, after
  owned scenario work, and at final cleanup. Only tracked sampler-owned PIDs
  may be stopped.
- Fixture and session validators use `Start-Process -WindowStyle Hidden`,
  `WaitForExit(timeout)`, captured logs and a 20-second maximum; session polling
  consumes the remaining declared deadline.
- `result.json` no longer derives success from a nominal status or any sample.
  Sampling activity comes from completed sample records. PASS requires the
  exact seven scenarios, all rows/evidence, fixture checks, executable identity,
  successful cleanup and zero external/live `levnote-native` processes.
- JSON reads explicitly use UTF-8. CPU percentages use the recorded
  `Stopwatch` baseline/end delta; UTC remains provenance.

Exact revised tooling identities:

- `scripts/run-samples.ps1` — 57,022 bytes,
  `8004E715418E653018413B36A4BAAF5C6FA293C9064720597A460CDC2E332481`
- `scripts/test-sampler-tooling.ps1` — 13,037 bytes,
  `32930D64ACC957D9B41DAC4A892D5714D1C157A00F921BE4DD10ED1F99CCFF49`
- `logs/sampler-tooling-self-test.json` — 3,801 bytes,
  `3C90A01D711BAB59B1F50E504E8D18331D9C6BC526853B3DE15C6492CF434947`

The no-launch self-test passed under Windows PowerShell `5.1.26100.9168`
Desktop and records `launchedProductProcess=false`. Parser checks are green.
It also invokes only the extracted aggregate function against synthetic
evidence: a preflight abort remains `FAIL` with
`processResourceSamplingPerformed=false`, while a complete seven-scenario
fixture/session/raw set with clean cleanup derives `PASS`.
The UTF-8 check uses explicit Unicode code points in the test source and
confirms the decoded value remains `中文标题·预览`; it does not compare two
misdecoded ANSI literals.
The focused fake-process check detects `HasExited=true`, records
`failed-process-exited` with owned PID `424242`, exit code `23`, and marks the
current partial scenario failed without launching any process.
There are zero `levnote-native` processes, zero `raw/*.json` files and zero
files under `appdata/`. The accepted executable remains
13,252,608 bytes / SHA-256
`5BFB5A445CC619FEA9380F0363D3DF26EC4F3FF944C8549B1992F5B462EF3F10`.
The sampling gate remains closed pending root and GLM read-only review.

## Current checkpoint — 2026-11-08 17:00 +08:00

Root accepted the 16:45 Markdown-links artifact. Resource follow-up is now
active. DeepSeek implemented and fixture-validated the isolated acceptance
tooling, but **has not launched the app, started the process matrix or collected
any resource sample**. The sampling gate remains closed pending root and GLM
read-only review.

Artifact root:

`native/artifacts/acceptance/resource-followup-20261108-165230/`

Review entry point:

`native/artifacts/acceptance/resource-followup-20261108-165230/README-REVIEW.md`

Concrete files:

- `scripts/generate-fixtures.ps1`
  (`63C4DF369B52587975625B2A7BCD73E76565D3168D55B565C53E2A6040C1312A`)
- `scripts/validate-fixtures.go`
  (`F145945A942498A40D4D8869F3CD22CF0865CEABD8236226B3CA262749951D78`)
- `scripts/run-samples.ps1`
  (`ABB61A3EF6BAFEE9A2DAB231645D09B765A9ADCFA61E260C0210B7C88E3E859B`)
- `bin/load-validator.exe`, precompiled once:
  5,797,376 bytes,
  `B7ADBBB94FF11C8FB43413C88D23900912BE87270E063F6D336D5A1CFC8C2990`
- `fixture-validation.json`
- `sessions/validator-smoke.json`
- `tooling-review.json`
- `powershell-parse-check.json`

Explicit runtime is Windows PowerShell `5.1.26100.9168`, Desktop edition:
`powershell.exe -NoProfile -ExecutionPolicy Bypass -File ...`. Scripts avoid
`Get-FileHash`, use `System.Security.Cryptography.SHA256`, and use
`Start-Process -WindowStyle Hidden` plus PID-scoped C# `EnumWindows` and
`WM_CLOSE`. The sampling script contains no Go/compiler/test/build invocation.

The precompiled validator passed all real format paths:

- `plain.txt`: 565,054 bytes, valid UTF-8/no BOM, 8,192 unique CRLF lines,
  `document.Open` clean.
- `markdown.md`: 994,815 bytes, UTF-8 BOM, `document.Open` decoded 994,812
  bytes, parser `Partial=false`, zero warnings, and exact 2,048 blocks /
  512 headings / 512 lists / 1,024 items / 512 tasks / 512 links.
- `ideas.mindmap`: 20,683 bytes, valid v1 logical structure, 500 nodes.
- `flow.drawio`: 39,444 bytes, one uncompressed page, 100 nodes and 180 edges.
  The parser reports only the two normal warnings for mandatory structural
  cells `0` and `1`; there is no stencil warning and no parser error.

The session mode uses the product `core/session.Load`. A smoke validation
against an existing product-created four-document session reproduced the four
ordered paths, views and active index 3 with no load issues.

Preserved precompile-path/log-directory harness failures:

- `validator-build.failure.json`
- `validator-build.failure-2.json`
- `validator-build.initial-success.json`
- `validator-build.pre-gofmt.json`
- `fixture-validation.initial.json`
- `fixture-validation.initial-meta.json`

They were tooling bootstrap failures, not product failures. The final
validator build and fixture validation both exited 0.

Accepted executable identity rechecked without launching it:
13,252,608 bytes and SHA-256
`5BFB5A445CC619FEA9380F0363D3DF26EC4F3FF944C8549B1992F5B462EF3F10`.
There are currently zero `levnote-native` processes. Neither `raw/` nor
`appdata/` contains a generated file.

After approval, the single matrix command is:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File `
  native\artifacts\acceptance\resource-followup-20261108-165230\scripts\run-samples.ps1 `
  -SourceExe native\artifacts\acceptance\markdown-links-20261108-164403\levnote-native.exe `
  -ArtifactRoot native\artifacts\acceptance\resource-followup-20261108-165230
```

Product source and the accepted binary remain frozen. Await review approval
before executing the sampler.

---

# Original prepared app measurement plan

Status: **PREPARED ONLY. STAGE NOT ACTIVE.** No build, process launch,
resource sample or product change was performed while preparing this plan.

This follows `resource_followup_stage_plan.txt`, including its corrections to
the earlier proposal. Measurement must start only after root accepts the
Markdown-links artifact and supplies its exact release executable.

## Fixed artifact and isolation

Use one artifact root per run:

```text
native/artifacts/acceptance/resource-followup-<UTC timestamp>/
  bin/levnote-native.exe
  fixtures/{plain.txt,markdown.md,ideas.mindmap,flow.drawio}
  fixtures/manifest.json
  appdata/<scenario>/{Roaming,Local}
  raw/<scenario>.json
  scripts/{generate-fixtures.ps1,validate-fixtures.go,run-samples.ps1}
  result.json
```

Copy the accepted executable once into `bin`, then require equal source/copied
SHA-256 values for the whole matrix. Every measured launch gets a fresh
`APPDATA` and `LOCALAPPDATA` under its own `appdata/<scenario>` directory,
except the seed/restore pair, which intentionally share one isolated directory.
Never change `HOME`, `CODEX_HOME`, the user's profile or an unrelated path.

Before any launch, enumerate running processes whose name is
`levnote-native`. Record the count and PID/path metadata, and abort the
checkpoint if any pre-existing LevNote process is present. Do not stop or kill
an unrelated/pre-existing process. Confirm absence again between scenarios;
only the sampler's PID may be closed by its PID-scoped `WM_CLOSE`.

Launch every process with `Start-Process -WindowStyle Hidden`. Find its window
using PID-scoped `EnumWindows`, then close it with `WM_CLOSE`; permit a forced
kill only as a recorded failed checkpoint. Record the returned visibility flag
and call the scope **hidden startup process**, never a visible rendered-editor
workload. Process measurement covers the main process only.

All waits are finite and produce explicit failure records:

- window discovery: 20s, otherwise `failed-window-discovery`;
- session write and product-loader validation (including the clean-restore
  seed): 20s, otherwise `failed-session-write`;
- `WM_CLOSE` acceptance and process exit: 15s, otherwise
  `failed-graceful-close` (then clean up only the sampler-owned PID);
- forced cleanup after an earlier failure: 5s, otherwise
  `failed-process-cleanup`.

Failure stops the matrix before the next scenario. A partial sample set is
retained as diagnostic raw data but never presented as a completed result.
Each scenario result records one status: `passed`, `failed-window-discovery`,
`failed-session-write`, `failed-graceful-close`, `failed-process-cleanup`,
`failed-fixture-validation`, `failed-fixture-immutability`, `failed-nonzero-exit`
or an explicit sampler exception. Only a matrix with every row `passed` gets an
aggregate success status.

## Deterministic fixtures

Generation is byte-for-byte deterministic and uses UTC only in artifact
metadata, never in fixture content. The generator writes
`fixtures/manifest.json` with name, byte count and SHA-256 for every file.
Generate once per matrix and record the actual manifest; do not duplicate a
regeneration/replay pass unless the evidence demonstrates nondeterminism.

### `plain.txt`

- UTF-8, CRLF, approximately 512 KiB.
- 8,192 distinct lines. Most lines use deterministic ASCII padding; every
  128th line contains Chinese and emoji. Do not add a BOM.
- Validation reads the file back and checks UTF-8 validity, CRLF count, unique
  sequence count and the recorded byte/hash values.

### `markdown.md`

- UTF-8 with BOM and CRLF throughout, approximately 0.95 MiB.
- Exactly 512 repeated sections. Each section contains one heading, one
  ordinary paragraph whose body is padded to 1,800 UTF-8 bytes, one unordered
  list with one task item and one ordinary item, and one paragraph containing
  an inline link. Seed bold/italic/code tokens and Unicode/emoji in the prose.
- Validation opens the file through `core/document.Open`, as the application
  loader does, and records raw bytes/BOM separately from the decoded parsed
  text. It must observe UTF-8 plus `BOM == true`; it must not feed the BOM bytes
  directly into a parser that expects decoded text.
- Parser validation must assert `!doc.Partial`, 2,048 blocks, 512 heading
  blocks, 512 list blocks with 1,024 items, 512 task markers, 512 inline links,
  and no `parse-limit` warning. This avoids treating a parser-limited fixture
  as an active-document measurement.
- The long prose is split across bounded blocks; all authored links are in
  separate rows. This fixture is not a dense-one-row stress case.

### `ideas.mindmap`

- Valid LevNote v1 logical-structure JSON with exactly 500 nodes: one root,
  seven branches, bounded deterministic leaves and one extra grandchild.
- Every id is unique and every topic is short. Validation calls the current
  parser with zero-valued limits and requires no error, no issue and 500 nodes.

### `flow.drawio`

- Valid uncompressed `.drawio` XML with a fixed 10x10 grid of 100 nodes
  (`140x64`, 180-pixel horizontal and 100-pixel vertical spacing).
- Connect each adjacent horizontal and vertical pair for 180 edges, using
  stable ids and no stencil library, compression, remote image or external
  resource. Validation requires 100 nodes, 180 edges, no error and no missing
  stencil warning.

Before sampling, `validate-fixtures.go` must validate all four fixtures through
their current real parsers and emit the expected counts plus actual bytes and
SHA-256. A validation failure stops the checkpoint before any process starts.
Keep the fixtures normally writable. Do not set read-only attributes or ACL
denials, because that would change the app's readonly workload. The sampler
performs no input or edit action and treats the fixtures as immutable by
discipline. Re-hash every fixture immediately before the matrix and again after
every process has exited. A changed or missing fixture fails the checkpoint
even when all process samples were collected.

The same helper also validates session snapshots through the current product
`core/session.Load` loader. Its `session` mode receives the expected tab paths
in order and active index; it requires no load errors and records the actual
tab order, active index and mode. The result records the validator path and
these expected/actual values. A schema-specific replacement loader is not
used.

## Sampling matrix

Use the same copied executable for every row. Sampling begins only after
PID-scoped window discovery. `Process.Refresh()` precedes every sample.

| Scenario | Arguments after executable | Settle | Observation window |
| --- | --- | ---: | --- |
| blank | none, fresh app-data | 10s | CPU baseline + 10 samples after ten 1s intervals |
| blank extension | same blank process/PID | none | extension baseline + 10 samples after ten 3s intervals (~30s) |
| plain | `fixtures/plain.txt` | 10s | CPU baseline + 10 samples after ten 1s intervals |
| markdown | `fixtures/markdown.md` | 10s | CPU baseline + 10 samples after ten 1s intervals |
| mindmap | `fixtures/ideas.mindmap` | 10s | CPU baseline + 10 samples after ten 1s intervals |
| drawio | `fixtures/flow.drawio` | 10s | CPU baseline + 10 samples after ten 1s intervals |
| ordered four | plain, markdown, mindmap, drawio in that order | 10s | CPU baseline + 10 samples after ten 1s intervals |
| clean restore | seed with `markdown.md`, close, relaunch with no args | 10s | CPU baseline + 10 samples after ten 1s intervals |

The blank extension is not a restarted process and must not be presented as
long-idle growth from a second baseline. Its extension baseline observation is
taken immediately after the initial blank window, followed by ten observations
at three-second intervals. The result records both windows independently and
identifies them as the same PID.

The clean-restore seed launch waits until its isolated
`Roaming/LevNoteNative/session.json` exists and validates, then closes
gracefully. The second launch uses the same app-data directory and no CLI
arguments. After each launch closes, validate the resulting session snapshot:
tab paths/order, active index, and mode must match the scenario. This proves
metadata state, not painted pixels in a hidden window.

The no-input hidden route cannot drive real tab clicks, close buttons or
reopen controls. Those process-level scenarios remain unmeasured rather than
adding product hooks. Existing native tester tests remain separate evidence
for cache release/rebuild behavior.

## Raw values and calculations

Each raw memory observation records UTC timestamp, elapsed milliseconds since
the previous observation, `WorkingSet64`, `PrivateMemorySize64`, and cumulative
`TotalProcessorTime.TotalSeconds`. A CPU window separately records its actual
baseline and end observations (cumulative CPU and UTC timestamps) after the
last memory sample. Ten memory samples therefore mean ten observations across
ten one-second intervals, not eleven observations or a ten-second label
applied to an interval of a different duration. The blank extension records
its immediate baseline observation plus ten three-second interval samples.
Each scenario also records:

- executable and fixture hashes/bytes;
- sample visibility state and that launch used `-WindowStyle Hidden`;
- process id, logical processor count, argument list, settle/sample cadence;
- first/last cumulative CPU, actual elapsed interval and exit code;
- session snapshot summary and fixture hashes after launch.

Per scenario emit min/average/peak/final working set and private bytes, CPU
seconds delta, one-core percent (`delta / elapsed * 100`) and system percent
(`one-core percent / logical processors`). Report raw arrays as well as the
small summary table. Do not require private or working-set values to fall, and
do not compare with Tauri.

Take one repeat only for an explicit anomaly and label both runs. Do not trim
working sets, force GC, add instrumentation or run a broad stress matrix.
After the fixture validator has completed and the first process launches, do
not invoke `go build`, `go test`, `go run`, another compiler or any source
edit until the matrix and evidence capture finish. The sampler script itself
performs no compilation.

## Planned commands at activation

```powershell
# Generate and validate fixtures before starting any process.
native\scripts\go.ps1 -GoArguments @('run','./artifacts/acceptance/resource-followup-<timestamp>/scripts/validate-fixtures.go')

# The sampler itself performs the bounded matrix and writes raw JSON + result.
powershell -NoProfile -ExecutionPolicy Bypass -File native\artifacts\acceptance\resource-followup-<timestamp>\scripts\run-samples.ps1 -SourceExe <accepted-exe> -ArtifactRoot <artifact-root>
```

The exact relative path of the temporary validator will be fixed when the
artifact directory is created; it is acceptance tooling, not product code.
