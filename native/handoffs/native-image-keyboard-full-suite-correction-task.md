# Explicit 100% mode: full-suite regression correction

ACTIVE2026-11-10. DeepSeek image keyboard owner only. Root reviewed the failure
in native-autosave-evidence-20261110/04-final-test.output.txt. The app suite passed;
imageview failed TestOneRequestAndPublicToolbarInput at line360: after clicking
100%, zoom=1 but fitMode remained true. Vet was correctly not run.

Current ui.go zoomToCenter returns before changing mode when zoom/pan equal the
current viewport. For a small image fitted at1 this ignores an explicit100%
action, leaving future resizes in fit mode. Explicit100% (button or key1) must
enter manual mode even when current pixels coincide. Preserve fit for genuinely
clamped no-op relative zoom/pan; do not globally remove no-op checks or alter
admission/geometry. Inspect and implement the narrow correction.

Ownership ui.go and imageview_test.go only, plus handoff/evidence. Add or extend
a meaningful public-button/key1 regression for a small fit-at1 image, including
resize proving mode semantics. Do not weaken existing toolbar assertions.
Preserve all earlier source identities and failed combined evidence.

Run one affected focused regression through go.ps1, recording exact command,
output, exit and source changes. Stop/freeze when it passes. Do not run full
suite/vet: Astra remains sole combined runner after corrected freeze. No package,
host, worker, captures, performance runs or child delegation.

Update native-image-keyboard-result.md with the correction and new owned hashes,
and report in native-image-keyboard-full-suite-correction-result.md.
