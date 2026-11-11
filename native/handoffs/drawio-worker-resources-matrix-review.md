# Root review of matrix wrapper

2026-11-10. Do not execute the matrix yet. Accepted source/tester remain frozen.
Root read both wrappers and the actual tester/preflight. Concrete corrections:

1. wrapperRoot is native/handoffs/<directory>. Four parent traversals resolve
   F:/AIproject, not the repository; use the actual three-parent root and check
   known repository file. Current precondition cannot admit the requested cwd.
2. TestDrawioWorkerResourceMeasurement explicitly requires its output directory
   to be ABSENT, but wrapper creates sampleRoot and passes it as HARNESS_OUTPUT.
   Keep sampleRoot for stdout/stderr; pass an absent child (e.g. measurement),
   then use that child's measurement.json and config. No live trial needed.
3. Successful preflight and measurement failures serialize as JSON null. In
   PowerShell @($null).Count is one; normalize null to an empty array before
   count. Same discipline for optional arrays/partial records.
4. Summarizer uses Byte labels for CPU100ns and lifetimeNS. Use explicit unit
   or unit-neutral min/median/max values. LifetimeNS is process wall lifetime,
   not CPU. Preserve quantization disclosure and CPU kernel/user separately.
5. Set-StrictMode plus raw failed rows without MeasurementSummary currently
   makes summary throw. Partial/failed/malformed measurement must preserve raw
   rows and a visible diagnostic; missing counters are not zeros. Each row must
   get an explicit FAIL on exception, with available logs/identities, and stop.
6. There are two same-scope trap declarations, including before runRecord exists.
   Replace with one bounded explicit try/catch/finally flow so precondition
   failure and mid-row failure do not fail while writing the failure record.
7. Sort inventory deterministically. Timeout cleanup currently Kill(true) is a
   process-tree kill, not the exact-only claim in prose. Use the exact owned
   tester handle/object and the Go runner's existing Job cleanup; bound cleanup
   wait and record failure instead of an unbounded WaitForExit. Retain exact PID,
   creation time/image where available. No process-name scan.
8. Read the actual Go-produced record shapes; do not reject a passing row due
   only to null failures or assume failed probes always have FinalTimes.Times.

Fix these concrete defects in the existing small wrappers. Preserve old files'
identities/parse record and all failed logs. Then one fresh final static parse
with real command/stdout/stderr/exit, and a small NO-PROCESS PowerShell check of
path arithmetic, successful null-array preflight admission, absent child output
layout, plus summary with a passing sample and an early failure with missing
measurement. Synthetic samples must be unmistakably labeled fixture-only and
kept separate; no fabricated measurement claims. These wrapper-only checks are
now authorized; no tester/build/preflight/worker/host execution. No new generic
test framework. Update the preparation report and freeze for review.
