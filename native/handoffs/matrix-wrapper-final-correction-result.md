# Matrix wrapper final correction

2026-11-10 Astra. Corrected existing run-matrix.ps1 and summarize-matrix.ps1;
frozen for root review. No real tester/worker/host, Go test/build, preflight or
resource measurement was executed. Accepted harness source/tester unchanged.

Both files remain in drawio-worker-resources-matrix-wrapper-20261110. Prior
versions and preparation are saved in final-correction-20261110/*.before.*;
all earlier failed parse/fixture evidence remains untouched.

Corrections:
- Null/empty/singleton arrays retain actual array objects under StrictMode,
  including warning comparison and successful null preflight failures.
- Path conditions parenthesize separate commands; sample parent is created,
  measurement child remains absent for tester. Existing output/wrong cwd and
  output beneath protected packages fail before launch/write.
- Per-row saves retain RUNNING. Terminal PASS requires all12 PASS rows, no
  stop/error and successful package protection. Any row/protection failure
  yields wrapper exit1. Empty final error text is explicitly accepted.
- Current row exists before sample setup; catch marks FAIL and collects all
  available stdout/stderr/measurement/config. Package-after inventory runs in
  finally whenever before-inventory exists, including exceptional paths.
- Exact returned tester SafeHandle is held before optional identity reads.
  Optional identity failures are disclosed. Errors after start and timeout
  clean up that process object only, bounded10s, then dispose. No tree kill or
  process-name search; worker cleanup remains Go Job ownership.
- Missing finalTimes/Times/summary/counters stay null. Missing CPU prevents a
  passing measurement. Summary accesses omitted fields safely, retains raw
  rows and diagnostics, and never fills unknown counters with zero.
- ExportElapsedNS is consistent end-to-end; memory bytes, elapsed/lifetime ns
  and CPU100ns remain distinct. CPU quantization/zero and limited/unknown peak
  coverage stay visible; even-count median retains fractional values.

Verification: final-correction-20261110/fixtures-01, PowerShell7.6.9, exit0,
fixture-results.json128/128 checks PASS. Actual helpers loaded from AST; actual
wrapper and summarizer invoked with a mock Start-Process returning fake process
objects. Scenarios: two complete12-row success cases (normal and optional
identity error), malformed JSON, start error, wait error, timeout, cleanup
timeout, null FinalTimes, truly omitted summary/exit/timedOut/violations,
existing output, wrong cwd, protected-package output, null/empty/singleton
arrays/warnings, completion gating, exact bounded cleanup and statistics.

All synthetic evidence is fixture-only; no fabricated real measurement claim.
Each failure case stopped after one mocked start, returned nonzero, kept
available artifacts and both after-package inventories. Successful per-row
snapshots stayed RUNNING; elapsed and CPU stats were non-null with correct
units, including genuine numeric zero in the synthetic CPU fixture.

One final static parse under the same PowerShell7.6.9 returned exit0, zero
errors. Command/streams/exit are preserved alongside final-parse.json:
- run-matrix.ps125076B SHA256
  7947980b6bdc00fd5fe0171ca0b681726cbbbd55706f88286516c212433eae0e
- summarize-matrix.ps18391B SHA256
  1bd9455d41ba17c49b055dc4a30bbc72eeb387ce351278bf4af3b8178bc29929

Fixture stdout's final dictionary projection initially printed nulls; saved
result JSON was correct. Only the fixture display cast was corrected afterward,
with no wrapper change or rerun. verification-notes.md preserves that detail.

Next: root accepts/activates fixed12 once. Until then scripts/evidence frozen;
GPU design checkpoint remains image-bitmap-lifecycle-review-checkpoint.md.
