# Drawio worker resource harness progress

2026-11-10: source work began with the new Windows-only
`native/core/engineruntime/drawio_worker_resources_test.go`. The active scope
is preparation only. The default path skips before any filesystem/package/
process work. The opt-in no-process preflight records platform and tester
identity, verifies the exact accepted `.20`/`.21` package identities, freezes
two deterministic sub-1MiB fixtures and their accepted 598-byte SVG oracle,
and derives source hash/length plus XML page/node/edge counts. No worker
request, matrix run, GUI launch, product rebuild or full suite has occurred.

Root review requested three fixture/oracle corrections and inclusion of a
default-skipping future managed measurement path. Source now keeps only the
accepted single rectangle on each selected page, places workload on unselected
pages, emits valid `vertex="1"`/`edge="1"` attributes, uses the exact uppercase
wire warning code/message, and predeclares warnings for the selected page only
(the worker protocol filters parser issues to selected page plus document).
The future measurement test is present but refuses execution until root review.

Root checkpoint correction applied: the opt-in measurement function now contains
the real managed path—fixed package metadata verification, fresh config,
`NewPackageManager`, `ImportFromReleaseDir`, explicit `Activate`, `NewManaged`,
one page-selected `Export`, one `Runner.Close`, existing `resourceProbe`
instrumentation and exact image/exit/cleanup checks. Only the separate
measurement environment gate prevents execution. The first preflight also
caught a transposed hex digit in the literal `.21` release-manifest identity; the
actual accepted digest is `c255...ce9a824` with `ce` before `9a`, and the source
constant was corrected. Exact fixture count assertions are now
Normal8/148/42 and Metadata6/41/60, with selected page identity checks.

Root05:30 corrections are applied before any additional compile: every JSON
record field has a unique tag, sentinel marshaling validates required keys, a
one-request atomic guard is added, the installed image path is derived from
`manager.installedPath(..., receipt.Digest)`, operation counts and receipt
digest are recorded, actual SVG bytes/hash plus ordered warnings and post-run
source hash are recorded, probe completion is guaranteed after cleanup on every
path, and creation/exit/final peak/CPU/read-failure coverage is checked.
Source is ready for the single final validation sequence and root review.

Root continuation review corrections are applied and formatted. The saved
source is now nil-safe for initial/final process-time reads, records incomplete
lifetime peak coverage under serialized `limitations` instead of `failures`,
serializes empty actual warnings as JSON `[]`, validates fixture/version names
before source generation, and includes all new record fields in the JSON
sentinel. No new compile, tester run, preflight, worker request or matrix run
was performed after these corrections.

The final root order correction is applied too: `MatrixPosition` now derives
its value from one explicit frozen 12-entry tuple, matching the alternating
request order exactly. The no-process preflight validates all 12 mappings
against that tuple, rather than checking only range or uniqueness.

SOURCE CHECKPOINT for root review (before any new validation):

- `native/core/engineruntime/drawio_worker_resources_test.go`
  SHA256 `6b546efc1f19a38dec603ed9040c183d0ed6f4f69f3b14b3d470fedc05cc69a6`,
  40,272 bytes after `gofmt`.
- Formatting only was run through
  `native/scripts/go.ps1 -GoArguments @('fmt','./core/engineruntime/drawio_worker_resources_test.go')`
  with exit0.
- The existing frozen tester remains
  `native/drawio-resources.test.exe`, SHA256
  `50552a1b3ad8a83a366f910fdda8b9773ae5f36af9ebef4aced517ddcd8514c3`,
  7,519,744 bytes. It is evidence from the prior source and does not represent
  this checkpoint.
- Existing final evidence
  `native/handoffs/drawio-worker-resources-harness-final-20261110/` and earlier
  snapshots remain unchanged.

After root accepts this source checkpoint, the only approved validation sequence
is: one explicit evidence-owned tester compile using `native/scripts/go.ps1`,
one default-SKIP run of that exact tester, one no-process preflight with a fresh
absent output directory, then preservation of all stdout/stderr/exit files and
failures. Measurement remains unactivated.

Root accepted the checkpoint and the approved validation sequence completed
once. The source remained frozen at SHA256
`6b546efc1f19a38dec603ed9040c183d0ed6f4f69f3b14b3d470fedc05cc69a6`
(40,272 bytes). The fresh evidence-owned tester is
`native/handoffs/drawio-worker-resources-harness-final-validation-20261110/drawio-resources.test.exe`,
SHA256 `e737a93cb11331b09b83113984b9d279aef6cab129ff590063fd21fdee845398`,
7,528,448 bytes. Compile, exact-tester default SKIP and no-process preflight all
returned exit0; preflight recorded PASS and preserved all commands, output,
exit codes and fixed fixture/package identities. Measurement, worker requests,
host requests and matrix execution remain unactivated. Final factual results
are in `drawio-worker-resources-harness-result.md`.
