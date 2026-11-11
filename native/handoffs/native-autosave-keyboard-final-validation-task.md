# Autosave and image keyboard: final affected validation

ACTIVE 2026-11-10. Sole runner: Astra
`/root/read_native_handoffs_tab_workflow_resume_task_md`.
Both owners have frozen source. Root reviewed the current implementations and
the corrected keyboard tests and matched their reported key source hashes.

Confirm all11 entries in native-autosave-evidence-20261110/source-freeze.json
and the four keyboard entries in native-image-keyboard-result.md still match.
Record the pre/post source identities without changing source. Then run once,
sequentially, from the repository root through the wrapper:

1. native/scripts/go.ps1 -GoArguments @('test','./app','./modules/imageview','-count=1')
2. native/scripts/go.ps1 -GoArguments @('vet','./app','./modules/imageview')

Preserve exact commands, complete output, exit status and any failure in the
existing native-autosave-evidence-20261110 directory with fresh numbered files.
Do not execute step2 if step1 exposes a compile/runtime blocker without first
reporting it. No duplicate focused/full suites, captures, benchmarks, package,
host or worker execution. Do not change the keyboard owner's files.

Write native-autosave-keyboard-final-validation-result.md with actual outcomes,
immutable source identities or their verified inventory link, and clear limits:
synthetic/fake-time/UI-thread queue coverage is not physical Windows IME or
driver/resource evidence. Report and freeze; root handles acceptance and .24.

## Corrected final run activation

ACTIVE after the first combined failure: root reviewed the narrow explicit100%
mode correction and the strengthened public regression. New test uses400x200 at
760x480 fit1, shrinks to300x240 and distinguishes cropped100% from public0fit.
Focused corrected command passed1.204s/exit0. All prior evidence stays intact.

Current keyboard ui.go6456B SHA256
BEF3A43EA6A028791A623728E9899CF481A1BB46BC12D91D61AFD67FB95AA368;
imageview_test.go42807B SHA256
824645F0115D8E4DF7851696E78B33AAC941523C9C2D5929B0EB3B1019094231.
input.go/binary_keyboard_test.go and11autosave files remain frozen unchanged.
Use current result for all expectations; do not rewrite old evidence manifests.

Astra is now authorized to run ONE corrected combined test then vet as above,
with fresh numbered records and pre/post15file identities. No new source edits
or product execution in this validation action. .24 helper preparation can resume
afterward; package/host still requires root source acceptance.
