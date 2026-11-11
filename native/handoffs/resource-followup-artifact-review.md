# Resource follow-up tooling — current independent review

Status: **APPROVED FOR THE SINGLE AUTHORIZED MATRIX.**
This is a read-only review of the final sampler revision
`8004E715418E653018413B36A4BAAF5C6FA293C9064720597A460CDC2E332481`
(2026-11-08 17:27:48). No generator, sampler, product process, benchmark,
product test, or product build was run by this reviewer. The matrix had not
started when this review was written; `raw/` and `appdata/` were empty before
DeepSeek's authorized launch.

## Preserved preflight failure — 2026-11-08 17:30 +08

The first authorized invocation stopped before any app launch. The precompiled
validator's product-side fixture output was PASS, but the sampler wrapper
recorded `Process.status=exited` with `Process.ExitCode=null`, so the script
classified fixture validation as failed and stopped. `result.json` correctly
reports overall `FAIL`, `processResourceSamplingPerformed=false`,
`completedSampleScenarioCount=0`, zero scenarios, zero `levnote-native`
processes, zero live sampler-owned processes, and no process-observation
failure. `raw/` and `appdata/` remained empty.

This is a sampler preflight/wrapper lifecycle failure, not a product parser
failure and not a completed or partial resource sample. The record is preserved
as evidence; DeepSeek's focused wrapper correction and the root-authorized
actual matrix remain the next step.

## First matrix result — six completed rows, clean-restore stop

The first actual matrix launched the product and completed the first six
scenarios. Their raw evidence is consistent: each has ten observations, a
completed CPU window, working/private summaries, passing session validation,
graceful `WM_CLOSE` exit code 0, unchanged fixture hashes, and a passing
external-process check after owned work. The aggregate remains `FAIL`, which is
correct because the seventh scenario is incomplete and must not be presented as
a completed matrix.

The clean-restore failure is specific. The hidden seed launch found its window
but produced no `session.json` during the 20-second pre-close wait; the
sampler recorded `failed-session-write`, then closed the owned seed with
`WM_CLOSE`. The close itself passed with exit code 0 and produced a 382-byte
session containing the expected Markdown fixture, `preview` view, active index
0, UTF-8/BOM metadata, and the recorded preferences. Therefore:

- **Automatic hidden-live persistence was not observed within the declared
  20-second bound.** This does not prove the product never persists while
  hidden/idle; it only bounds this no-input scenario.
- **Graceful-shutdown persistence was observed.** `WM_CLOSE` exited 0 and wrote
  the correct session before the process ended.
- The remaining restore measurement is whether relaunching with the existing
  session restores that Markdown tab, performs the ten-observation window, and
  again exits cleanly. This is a continuation from the produced session, not a
  repeat of the six completed rows.

The original matrix `result.json` and raw files must be preserved unchanged.
The continuation should be recorded as separate evidence and must not overwrite
or relabel the original failed matrix.

### Completed six-row resource values

These values are process-scope observations from hidden startup windows, not a
visible rendering or interaction claim. Every row reports zero CPU seconds over
an approximately 10.1-second window.

| Scenario | Working set | Private bytes |
| --- | ---: | ---: |
| blank | 61,292,544 B | 90,255,360 B |
| plain | 64,827,392 B | 93,097,984 B |
| markdown | 66,883,584 B | 94,773,248 B |
| mindmap | 62,128,128 B | 90,771,456 B |
| drawio | 61,902,848 B | 91,312,128 B |
| ordered four | 67,637,248 B | 96,284,672 B |

In these hidden/no-input samples the ten observations within each row were
identical, so min/average/peak/final are the same. That should be reported
honestly as stable observed values, not as a claim that memory cannot vary in
normal interactive use.

## Clean-restore followup and final evidence review

The authorized continuation produced a real PASS at 17:40. It validated the
session produced by the seed's graceful close, relaunched with no CLI arguments
from the same app-data, restored the expected Markdown tab and `preview` view,
took ten observations over about 10.14 seconds, closed with `WM_CLOSE` and
exit code 0, and validated the unchanged session again. Both fixture checks
passed and the 382-byte session hash remained
`FDAC9EBEFCCEB693D6AC95AC3BF6ED43459EDD382F85A4CED42D64F7CE6355D4` before and
after the reload.

DeepSeek then repeated the same 17:40 followup at 17:42 solely because the
first record omitted `systemLogicalProcessors`. The repeat also passed and
recorded 16 logical processors, but it was unnecessary. Treat the 17:40 PASS
as the primary restore observation; treat 17:42 only as a supplemental repeat
that fixes missing metadata. Do not count it as a second independent scenario
or average it with the first. Root has ordered no further runs.

The original matrix evidence is preserved unchanged; its `result.json` and all
seven raw files hash-match the
`matrix-original-fail-20261108-171800` preservation copy.

### Primary user-facing resource table

These are means of the saved `samples.observations[]` records, one value per
observation, from the `workingSet64` and `privateMemorySize64` fields. The
six matrix rows and the first 17:40 clean-restore continuation each contribute
10 observations. MiB values are exactly `mean(raw bytes) / 1,048,576` and are
shown to nine decimal places; they were not copied from rounded summaries or
assumed equal. This is main-process memory during hidden, no-input
startup/restore windows; it does not claim visible rendering, GPU paint, or
interactive behavior.

| Scenario | n | Mean working set bytes | Mean private bytes | Working set MiB | Private MiB | CPU delta |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| blank | 10 | 61,292,544 | 90,255,360 | 58.453125000 | 86.074218750 | 0 s |
| plain text | 10 | 64,827,392 | 93,097,984 | 61.824218750 | 88.785156250 | 0 s |
| Markdown | 10 | 66,883,584 | 94,773,248 | 63.785156250 | 90.382812500 | 0 s |
| mindmap | 10 | 62,128,128 | 90,771,456 | 59.250000000 | 86.566406250 | 0 s |
| draw.io | 10 | 61,902,848 | 91,312,128 | 59.035156250 | 87.082031250 | 0 s |
| ordered four tabs | 10 | 67,637,248 | 96,284,672 | 64.503906250 | 91.824218750 | 0 s |
| restored Markdown, 17:40 | 10 | 73,801,728 | 101,376,000 | 70.382812500 | 96.679687500 | 0 s |

The unplanned 17:42 repeat is supplemental metadata evidence, not an eighth
scenario and not averaged with the primary restore. Its 10 observations have
mean working set 75,214,438.4 bytes (`71.730078125 MiB`) and mean private bytes
101,138,432 (`96.453125000 MiB`), with zero CPU delta over 10.182 seconds.

## Final scope statement

- The original matrix aggregate remains `FAIL`, because its clean-restore row
  stopped after the hidden seed produced no session within the declared
  20-second pre-close wait.
- Graceful shutdown did write the correct session, and the authorized followup
  proved that the app can reload that session into the expected Markdown
  preview tab and close cleanly.
- Automatic hidden-live persistence remains unobserved within this bounded
  scenario; the evidence does not establish whether it occurs under other
  timing or interaction.
- The resource table is valid for hidden/no-input process startup and restore
  only. No Tauri comparison, visible rendering, GPU, or interactive-memory
  claim is supported.

## Current-source findings

The five process-lifecycle findings are now fixed in the named sampler revision:

1. Owned app and validator PIDs are registered immediately, and the top-level
   `finally` calls `Cleanup-OwnedProcesses` even after a scenario exception.
   Cleanup status and per-PID history are retained in the aggregate evidence.
2. Empty argument lists omit `Start-Process -ArgumentList`; non-empty Windows
   arguments are explicitly quoted. The no-launch self-test covers the empty,
   space, quote, and mixed-argument cases.
3. `Start-AppProcess` performs an external-process check before every launch;
   `Complete-ScenarioRecord` checks again after owned work, and the aggregate
   checks again after final cleanup.
4. Fixture and session validators are bounded child processes with captured
   logs, timeout cleanup, strict UTF-8 JSON reads, and a bounded polling
   deadline.
5. Aggregate PASS is derived from the exact seven-scenario set, completed
   sample rows, session/raw evidence, fixture checks, executable identity,
   cleanup, and final process checks. A synthetic preflight abort is now
   tested to report `processResourceSamplingPerformed=false` and FAIL, while a
   complete synthetic evidence set reports PASS with seven sampled scenarios.

The no-launch self-test and PowerShell parser checks pass on Windows PowerShell
5.1 Desktop. The final reviewed identities are:

- sampler `run-samples.ps1`: 57,022 bytes,
  `8004E715418E653018413B36A4BAAF5C6FA293C9064720597A460CDC2E332481`;
- tooling self-test `test-sampler-tooling.ps1`: 13,037 bytes,
  `32930D64ACC957D9B41DAC4A892D5714D1C157A00F921BE4DD10ED1F99CCFF49`;
- self-test result: 3,478 bytes,
  `67BFBCEF055CCA71774780ED4F8928F1D4F1722BAD7FAEE45D580D51442F5E99`.

## Corrected P2: process exit during observation

The final sampler now guards observation probes with `HasExited` and records
either `failed-process-exited` or `failed-process-observation`. It retains the
owned PID, exit code, label, observation index, partial observations, and the
active scenario record in the aggregate. The no-launch self-test verifies an
already-exited synthetic process produces the `failed-process-exited` scenario
status and PID/exit-code evidence. This closes the prior evidence-quality gap
without expanding the framework.

It is accepted as P2 diagnostic-quality work: the previous version already
failed truthfully and could not promote a crash to PASS. No additional gate is
required unless final results expose a concrete false-success, file-safety, or
process-safety defect.
completion path; no broader redesign is needed.

## Other evidence and non-blockers

- Fixture and session records remain internally consistent and use the real
  application loader. The two drawio `UNSUPPORTED_ELEMENT` warnings are the
  expected mandatory structural cells `0` and `1`, not a fixture failure.
- CPU percentages use the recorded Stopwatch delta between baseline and end
  observations, with UTC kept as provenance. The result explicitly limits its
  scope to the hidden main process and makes no visible GPU/rendering claim.
- Successful `Close-AppGracefully` and the timeout fallback unregister the
  owned PID before the next pre-launch external-process check, so normal
  closure does not hide a still-live owned process from the next scenario.
- An early `Start-Process` failure still propagates to the aggregate failure
  path rather than producing a per-scenario `failed-launch` row. Global
  cleanup and final FAIL evidence cover this, so I do not consider it a
  blocker; add a structured launch status only if root wants richer
  per-scenario diagnostics.
- `Assert-Fixtures` and manifest reads can still throw to the aggregate
  path. This is truthful and cannot promote a failure to PASS; I do not require
  another wrapper before the matrix.

## Review conclusion

The fixture, session, executable, UTF-8, argument-quoting, Stopwatch,
process-ownership, and aggregate-truth requirements are satisfied. After the
mid-sample process-exit status is implemented and its focused no-launch or
controlled failure check passes, the package is suitable for the sampling
gate. The existing ordinary drawio structural warnings should not block
resource sampling.
