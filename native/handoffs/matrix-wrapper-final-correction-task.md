# Matrix wrapper final correction: Astra ownership

ACTIVE2026-11-10 for Astra read_native_handoffs_tab_workflow_resume_task_md.
Temporarily prioritize this bounded correction over the GPU design (keep any
GPU notes). Previous GLM wrapper owner has finished and has no current write
ownership. Root reviewed its revised files; runtime defects still remain.
No tester/worker/preflight/compile/build/host launch authorized yet.

Continue existing run-matrix.ps1 and summarize-matrix.ps1 in
native/handoffs/drawio-worker-resources-matrix-wrapper-20261110, preserving old
snapshots and failed verification logs. Read drawio-worker-resources-matrix-
wrapper-task.md and review.md plus actual Go measurement/preflight schema.
Accepted harness source/tester immutable. No production code changes.

Root concrete second-review findings:
- ConvertTo-Array returns @() through pipeline enumeration: assigned result is
  null, so .Count under StrictMode fails. Preserve an actual array object or
  capture with @(...), consistently at call sites (including warning arrays).
- `if (Test-Path -LiteralPath $sampleRoot -or Test-Path ... )` treats -or as a
  cmdlet parameter; parenthesize the independent command expressions.
- Complete-RunRecord declares Mandatory[string] ErrorText, then passes ''.
  Empty strings need explicit admission or optional parameter.
- Complete is called after every row and marks whole run PASS prematurely.
  Terminal PASS requires all12 rows passed and package protection succeeded;
  per-row saves must keep RUNNING. Row FAIL must produce nonzero wrapper exit.
- Catch does not mark the current row FAIL or collect available artifacts,
  and exceptional paths skip after-package inventory entirely. Keep failure
  records explicit and package protection in finally where inputs initialized.
- Wrapper finalTimes.times is accessed even when finalTimes is null; preserve
  absent values. The actual summary loop accesses missing measurementSummary
  directly despite StrictMode. Synthetic missing-record rows must OMIT fields,
  not just pre-fill null and thereby hide the failure.
- Summary asks RequestElapsedNS but wrapper emits ExportElapsedNS: elapsed
  statistics silently missing. Use one field and unit-correct names.
- Summary failed-row exitCode/timedOut/violations access also assumes presence.
- Owned process cleanup must cover errors after Start-Process as well as timeout;
  read/hold process handle immediately before querying optional identity. Keep
  waits bounded and exact-owned only, Go Job owns child-worker cleanup.

Fix existing small scripts, no new general framework. For verification exercise
the actual functions/scripts, not copied pseudo-logic: no-process checks for
null/empty/singleton arrays under StrictMode, precondition/output path handling,
run completion state and nonzero failure exit, summary of actual-shaped PASS
and truncated FAIL rows (missing properties), correctly non-null elapsed/CPU
stats. A small injection/mock or extracted pure helpers is acceptable; never
launch the accepted tester to debug wrapper. Synthetic evidence stays explicitly
fixture-only in fresh paths. Run one final static parse with command/streams/
exit under PowerShell7 used for execution, then publish final correction result.

No further execution until root accepts. After that run the fixed12 matrix once
if explicitly activated, then return to image-bitmap-lifecycle-task.md design.
