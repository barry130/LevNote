# One combined app validation after reopen and live-tab admission

ACTIVE2026-11-10. Root accepted cap source/focused results and independent
review in live-tab-admission-root-acceptance.md. All app writers are frozen.
Owner is /root/read_native_handoffs_live_tab_admission_implementation_task_md;
root remains planning/review only. No child delegation. Execute now.

Use native/scripts/go.ps1 -GoArguments @('test','./app','-count=1') once and
@('vet','./app') once, sequentially. Capture exact commands, numeric exit,
stdout/stderr and UTC times. Use absolute evidence paths because the wrapper
changes working directory. Do not use pipelines that hide the Go exit code.
No new benchmarks/captures/GUI launch/worker requests or package work.

Fresh evidence: native/handoffs/file-workflow-final-evidence-20261110.
Before running, inventory the union of final reopen/cap changed source and
tests. After the commands, require identical identities for those files.
Write source-frozen.json with [{Path,Length,Sha256}], Path native-relative
(e.g. app/app.go), actual final bytes/hash. Do not insert placeholders. The
manifest is the .27 packaging dependency, together with root's future
native/handoffs/file-workflow-source-root-acceptance.md.

If tests fail, retain failure and publish exact cause rather than building.
Only owner fixes or separately assigned test fixture corrections may justify
a rerun; do not silently weaken assertions or run broad repeated checks.
Publish file-workflow-final-validation-result.md with actual results, final
union identities, inherited focused evidence and independent review boundary.
Do not claim physical keyboard/IME behavior,100-tab RSS, or binary delivery.
