# Drawio worker resource matrix preparation

2026-11-10. The reviewed fixed12 sequential wrapper and summary generator are
corrected, wrapper-only checked, statically parsed, and frozen for root review.
No wrapper execution, tester launch, worker request, host request, compile,
preflight, product rebuild, or harness source change occurred. Actual execution
remains blocked until root explicitly activates it after the .22
package/build/hidden-host activity finishes.

## Owned files

The wrapper files are in
`native/handoffs/drawio-worker-resources-matrix-wrapper-20261110/`:

- `run-matrix.ps1`: fixed12 sequential wrapper.
- `summarize-matrix.ps1`: raw-row and deterministic summary generator.
- `final-static-parse.ps1`: the small static-parse helper.
- `final-static-parse.json`: final parse record.
- `wrapper-fixture-check-20261110/`: unmistakably fixture-only no-process checks
  and all preserved failures.

This preparation record is
`native/handoffs/drawio-worker-resources-matrix-preparation.md`.

## Review corrections applied

- Repository root arithmetic now uses the actual three-parent path and checks the
  known `AGENTS.md` and `native/scripts/go.ps1` markers. It resolves to
  `F:\AIproject\LevNote`, not `F:\AIproject`.
- Each sample keeps `sample-NN/` for wrapper-owned stdout/stderr and passes the
  absent child `sample-NN/measurement` to
  `LEVNOTE_DRAWIO_WORKER_RESOURCES_HARNESS_OUTPUT`. The child must not exist
  before launch; the wrapper uses its `measurement.json` and `config/` after
  the tester creates it.
- JSON `null` arrays are normalized before counting, including successful
  preflight `failures`, actual/expected warnings, record failures, limitations,
  and rows. Optional partial-row fields are not treated as zeros.
- Unit labels are explicit. Memory uses `bytes`; request and process lifetime
  use `ns`; CPU kernel/user use `100ns`. `ProcessLifetimeNS` is identified as
  creation-to-exit wall lifetime, not CPU. Quantization is included per metric.
- The summary preserves all raw rows and available artifacts under strict mode,
  emits a visible failure diagnostic for a failed row with no
  `measurementSummary`, and does not substitute zero for missing counters.
- The two same-scope `trap` blocks were removed. The wrapper now uses one bounded
  try/catch/finally flow. Precondition failure cannot write a fabricated run;
  a mid-row failure marks the current row and record FAIL, preserves available
  logs/identities, and stops.
- Package inventories are sorted deterministically. Timeout cleanup calls
  `Process.Kill()` only for the exact tester process object returned by
  `Start-Process`, waits at most ten seconds, records cleanup failure, and does
  not kill a process tree or scan process names. The tester PID, process start
  time, and image path are retained where available. The Go runner retains its
  existing worker Job ownership.
- Measurement handling follows the actual Go JSON shape. Null failure arrays,
  missing `finalTimes.times`, and partial probe records are handled without
  rejecting a passing row merely because optional arrays serialized as null.

## Wrapper behavior

The wrapper requires both `Activation=RUN_FIXED_12_DRAWIO_WORKER_MATRIX` and
invocation from the repository root. It verifies the exact accepted tester and
preflight identities before launch and reads the fixture source pins from that
preflight. It performs one fresh tester launch per row, in this exact order:

1. NormalMultiPage `.20`, ordinal1
2. NormalMultiPage `.21`, ordinal1
3. NormalMultiPage `.21`, ordinal2
4. NormalMultiPage `.20`, ordinal2
5. NormalMultiPage `.20`, ordinal3
6. NormalMultiPage `.21`, ordinal3
7. MetadataHeavy `.21`, ordinal1
8. MetadataHeavy `.20`, ordinal1
9. MetadataHeavy `.20`, ordinal2
10. MetadataHeavy `.21`, ordinal2
11. MetadataHeavy `.21`, ordinal3
12. MetadataHeavy `.20`, ordinal3

There is no warmup, retry, discarded sample, or parallelism. Each launch uses an
argument array, hidden window, absolute paths, a 120-second bounded wait, and a
fresh absent measurement child. The wrapper preserves stdout, stderr, exit code,
`measurement.json`, config inventory, command/environment pins, process
identity, and timestamps. It validates schema/status, sample identity, matrix
position, source pins, tester identity, actual SVG, ordered warnings, operation
counts, probe reads, and scoped package inventories. It stops at the first actual
failure and never reruns a slot.

## Summary behavior

The summarizer emits every raw row before any descriptive statistics. It reports
fixture/version groups for request elapsed, sampled maximum working-set, sampled
maximum private commit, observed OS working-set peak, observed OS pagefile/commit
peak, CPU kernel, CPU user, and process wall lifetime as separate measures. It
records unit and quantization, retains limitations, keeps complete and limited
lifetime peak rows separate, and includes failure diagnostics with available
artifacts. It makes no host/editor/Tauri/global memory or speed claim.

## Authorized no-process fixture checks

`wrapper-fixture-check-20261110/` is fixture-only and separate from all real
evidence. It checks repository-root arithmetic and markers, admission of the
successful preflight whose `failures` array is JSON null, the absent
`sample-01/measurement` child layout, and summary behavior with one synthetic
passing row plus one synthetic early-failure row lacking a measurement record.
The synthetic rows are explicitly marked fixture-only and make no measurement
claim.

The final fixture check returned exit0 and all four checks passed:

- Repository root resolved to `F:\AIproject\LevNote` with both markers present.
- JSON null preflight failures normalized to zero and admitted the clean PASS
  record.
- `sample-01/measurement` was absent before layout creation and produced the
  expected `measurement.json`/`config` locations.
- The partial summary retained one passing row, one failed row, one limitation,
  explicit byte/ns/100ns units, and a diagnostic proving `measurementSummary`
  was absent on the failed row.

Five earlier fixture-check attempts failed and are preserved with stdout,
stderr, exit, and a correction note. They found strict-mode summary property
access issues, an idempotent fixture-directory issue, and optional diagnostic
field access. They involved no tester, worker, host, build, preflight, or
product execution.

## Final static parse

The final parse invocation returned exit0. Windows PowerShell 5.1.26100.9168
parsed both wrapper scripts with zero errors and wrote
`final-static-parse.json` with `Pass: true`. Final wrapper identities:

- `run-matrix.ps1`: 21,487 bytes, SHA256
  `0f722793e4a1168c0f9730b983a9da93732d5c423ed1db1b44aa5b6a7536eba7`.
- `summarize-matrix.ps1`: 7,155 bytes, SHA256
  `83c88813f32bc7f492df8ac4be200192f672e0c1478f04786815e87bd842dc24`.

Two earlier final-parse helper attempts failed before writing a parse result:
one strict-mode `Measure-Object` issue and one missing array parenthesis. Both
commands, streams, exit records, and correction notes are preserved. An earlier
pre-review static parse remains in `static-parse.json`; it described the prior
wrapper and is superseded by the final parse above.

## Frozen input pins

- Harness source:
  `native/core/engineruntime/drawio_worker_resources_test.go`,
  SHA256 `6b546efc1f19a38dec603ed9040c183d0ed6f4f69f3b14b3d470fedc05cc69a6`,
  40,272 bytes.
- Exact tester:
  `drawio-worker-resources-harness-final-validation-20261110/drawio-resources.test.exe`,
  SHA256 `e737a93cb11331b09b83113984b9d279aef6cab129ff590063fd21fdee845398`,
  7,528,448 bytes.
- Exact preflight:
  `drawio-worker-resources-harness-final-validation-20261110/preflight-output/preflight.json`,
  SHA256 `1722314a3c99f4f70eb42f9e8b13078a3230a8f9642d2359d1884e9a9688911c`,
  8,797 bytes.
- NormalMultiPage: 28,976 bytes, SHA256
  `ab822c28db64279a793810c45bbff7f6d7ecfd1cd44aaa1f59c48e00fedcaea2`.
- MetadataHeavy: 93,639 bytes, SHA256
  `504b19ffcf126353b79b34bcb5735f6a1cff21d5d515b23bc294342c7a61bcb1`.
- Expected SVG: 598 bytes, SHA256
  `3f990a4aac3209bafad94f44026bf14f3cec27ea44bdbd3169555947936472bf`.
