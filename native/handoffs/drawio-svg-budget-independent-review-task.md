# Draw.io SVG budget: independent source review

2026-11-10. Active for DeepSeek
`/root/read_native_handoffs_resource_workload_next_task_md` after the .17 docs.
Read this file when message payload is opaque. Root coordinates and reviews;
do not spawn agents. Compact before 200K context.

Review the frozen slice in `drawio-svg-budget-result.md` against
`drawio-svg-budget-implementation-task.md` and its plan, actual source, tests,
prechange oracle and raw logs. Own only this review's planning/evidence files,
not product code, tests or documents. Do not edit the implementer's files or
build/run the application. No automatic repeat of passing tests; if a concrete
unresolved defect needs a reproduction, explain it first in the review and use
one minimal isolated check via the project Go wrapper, retaining actual output.

Write `drawio-svg-budget-independent-review.md` with actionable findings,
locations, evidence, recommended minimal fixes, and precise acceptance scope.
Separate definite defects from optional follow-up. Verify the reported frozen
file identities while reading; no broad repository/toolchain inventories.

Root preliminary concerns to independently decide:

1. Old export constructed `make([]Issue, 0)`; new emitter has `var warnings
   []Issue`. Successful no-warning result may change nonnil empty to nil while
   the exact oracle has four issues. Preserve prior successful API shape if so.
2. `writeSVGPath` appears to traverse/format remaining points after writer
   failure. `writeSVGText` appears to count newlines before checking an already
   failed writer. The plan calls for stop on first error; inspect all emit paths.
3. Tests cover lower requested bounds and limit resolver, but do they connect
   default/hard limit to actual emission? Decide if one bounded meaningful
   default-cap case is needed; avoid large matrices or memory claims.
4. Confirm chunk-boundary equivalence with `xml.EscapeText`, ordered warning
   compatibility, nil payload/issues on every failure, unchanged selected page
   and model/source bytes. Output cap is not a process-memory cap.
5. Read both final-test logs if present and explain why final-test-2 exists;
   preserve actual failed history rather than accepting only the final green.

When done, publish the review and stop pending any correction reconciliation.

## Correction reconciliation now active

Read `drawio-svg-budget-correction-result.md` and the actual changed svg.go /
svg_budget_test.go plus fresh correction logs. Root already read the two source
hunks. Independently confirm both findings fixed and the new production4096
boundary case is substantive; hashes/protected files agree. Note the writer
can still evaluate a bounded numeric argument in the current fragment after a
failure, while the unbounded point/label continuation is stopped; do not claim
zero post-error instructions. No repeat tests/vet/builds/launches are needed.
Write `drawio-svg-budget-independent-reconciliation.md`, keep original review
immutable, then stop pending root source acceptance. .18 delivery not active.
