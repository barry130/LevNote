# .19 harness review correction

ACTIVE 2026-11-10 for DeepSeek
`/root/read_native_handoffs_release10_ui_feasibility_task_md`.
Root found two runtime blockers by static inspection of the prepared harness:
`run-package.ps1:203` and `aggregate.ps1:20` assign `$host`, which is PowerShell's
read-only automatic `$Host` variable. Rename to task-specific variables and
update uses; also rename local `$matches` to avoid automatic-variable reuse.

Preserve the frozen original preparation directory/result. Copy its scripts to
fresh sibling `drawio-worker-release19-preparation-corrected/`, fix there, and
produce fresh static-parser evidence and manifest. This is a harness-only
correction; no .19 package/build/test/worker/host execution yet. Do not touch
product files, original preparation, accepted gates or .18.

Also protect BOTH Stage2 evidence directories in the corrected initializer:
`handoffs/drawio-worker-stage2-evidence` and
`handoffs/drawio-worker-stage2-test-followup-evidence`. De-duplicate the supplied
final evidence path so each directory is inventoried once. The final followup
and independent review now exist and are frozen. Retain all previous gates.

Inspect only these changes plus basic runtime variable/shape issues in the
prepared scripts. Static parse is allowed; do not execute delivery entrypoints.
Publish `drawio-worker-release19-harness-correction-result.md` with the new
directory, changed hashes, static result and exact activation command shape.
No child agents; compact before 200K. Freeze when done.
