# .19 delivery harness preparation (no product execution)

ACTIVE 2026-11-10 for DeepSeek
`/root/read_native_handoffs_release10_ui_feasibility_task_md`. Portable wording
and smoke fixture are frozen. Stage2 tests/review still pending. No children.

Read actual `.18` delivery task/result and existing accepted run/validation/
aggregation scripts from its evidence. Prepare the smallest .19 adaptation in
a new `native/handoffs/drawio-worker-release19-preparation/` directory, using
parameters for fresh candidate/evidence/config roots. No output directory
creation for .19 yet, no build, test, capture, worker/host launch or full old
inventory. Root will activate delivery after Stage2 source accepted.

Target per drawio-worker-release19-delivery-task.md: one package invocation,
5GUI builds, actual manifest/count/notice checks; protect .18 package/direct
evidence plus scoped current source and accepted stage gates. Use exact final
frozen Stage2 filenames/hashes later; don't record half-implemented files as
final. Reuse reviewed real smoke TestDrawioManagedRealBundleSmoke fixture and
its env contract, one enabled invocation in evidence bundle copy. No second
fixture/custom executable path or argument override. One hidden host startup/
owned WM_CLOSE separate from worker. Preserve real output/error/attempts.

Avoid hardcoded3worker/23file/4build assertions inherited from .18. Validate
four fixed catalog identities and expected worker artifact set, actual candidate
manifest schema1; record actual counts, expected25only as assertion needing
inspection. Schema2 belongs selections, not release manifest. Include new
managed worker smoke in aggregate while retaining separate hidden-host gate.
Don't expand unneeded resource checks or repeat all source tests. Copy/mutate
only fresh evidence scripts; never edit accepted .18 scripts/evidence.

Use native PowerShell filesystem ops only, validate absolute directories before
recursive copy/move/delete; no delete needed. All Start-Process Hidden.
Parse new PowerShell scripts once if needed; static syntax check permitted,
not product execution. Publish `drawio-worker-release19-preparation-result.md`
with prepared script paths, invocation schedule, deferred gates and assumptions
awaiting stage2. Freeze. This task does NOT authorize actual .19 delivery.
