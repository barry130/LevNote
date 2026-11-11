# Draw.io SVG budget: narrow correction

ACTIVE 2026-11-10 for DeepSeek implementation owner
`/root/read_native_handoffs_release10_ui_feasibility_task_md`.
Read `drawio-svg-budget-independent-review.md`; root agrees with both definite
findings. Own only svg.go and svg_budget_test.go, fresh correction evidence and
`drawio-svg-budget-correction-result.md`. Original result/oracle/evidence stay
immutable. No builds/launches/product docs/protocol/UI edits or children.

1. Preserve nonnil empty []Issue on successful warning-free export; keep nil
   payload/issues on every error. Add an actual no-warning success assertion.
2. Stop emission helpers on first writer failure: prefailed text/path/shape
   inputs return before scans/formatting, each path iteration exits on failure,
   and emitSVG checks between header/path/shape/label stages before optional
   midpoint/label work and warnings. Avoid an interface/framework for this.
3. Add one compact 4096-byte production escaping-boundary regression with a
   multibyte rune straddling the boundary plus escapable/invalid data, comparing
   full output to existing escapeAttr. No large64MiB matrix or benchmark.

Run meaningful affected drawio tests once after the corrections and vet for
drawio via existing wrapper, retaining outputs/exits. Do not repeat graphview
suite, UI/captures or package gates; those caller paths remain unchanged.
If any check fails, keep the log and rerun only relevant checks after fixing.

Write correction result with exact edits/final hashes, commands/outputs/exits,
and note that both earlier final-test logs passed (the second was duplicate,
not a recovery from a failed first final suite). Freeze for independent
reconciliation; don't self-accept or start the next stage.
