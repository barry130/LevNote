# Image references: final affected validation

ACTIVE 2026-11-10. Owner Astra /root/read_native_handoffs_tab_workflow_resume_task_md.
Root has read the format/UI implementation, new regressions and focused evidence,
and matched all twelve source identities against the two final result handoffs.
No format/UI source correction is currently outstanding. No child delegation.

Run one full affected check through native/scripts/go.ps1:

    test ./modules/markdownview ./app -count=1
    vet ./modules/markdownview ./app

Forward actual LASTEXITCODE explicitly in the command process. Capture exact
commands, streams, timestamps and numeric exits. Before and after, record the
twelve identities from markdown-image-reference-format-result.md and
markdown-image-reference-ui-result.md and require exact equality. Use new files
05 onward in the existing UI evidence directory; preserve all old failure and
interpretation files. Ensure the optional capture environment variable is unset.

Inherit final format/worker test+vet runs14/15 after the multiline-envelope
correction; do not repeat those passing packages. No broad all-packages suite,
build, package, OS worker, host, capture or resource work in this task. If a
substantive failure appears, preserve it, diagnose within owned UI/app tests and
runtime scope, and report any cross-owner correction before touching that code.

Write native/handoffs/markdown-image-reference-final-validation-result.md with
actual evidence and final frozen identities. On success remain available for
the separate .26 delivery task; do not launch it implicitly.
