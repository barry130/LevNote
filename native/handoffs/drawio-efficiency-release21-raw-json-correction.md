# Preparation correction: actual Go JSON uses uppercase Warnings

2026-11-10. Root checked the actual `drawioSmokeRecord` definition at
core/engineruntime/drawio_managed_bundle_test.go:85 and the immutable .19
managed-smoke/record.json:41. The struct has NO JSON tags. Its emitted property
is uppercase `Warnings`, just like the normalized result. The earlier root
draft note assumed lowercase raw `warnings`; that assumption was wrong and is
superseded by this direct source/historical-output evidence.

Change the strict helper calls in BOTH run-managed-smoke and aggregate for the
raw record to uppercase `Warnings`; normalized result already uses uppercase.
Keep exact required-property and null/empty-array semantics. Do not modify
production Go JSON or the immutable old evidence to fit the wrong harness.
Update the tiny no-product sanity cases to actual uppercase raw null/empty
shapes; retain rejection for absent/wrong-case/scalar/object/nonempty shapes.
One corrected sanity/static parse is justified by this discovered defect.
Preserve prior parse/sanity outputs; no product execution.

Rewrite the short preparation result with the actual final identities and
simple readable prose. Current result contains unrelated injected words such
as `Clarence`, `Euler`, `dining`, `talents`, `hesitating`, and `TBD` annotations.
No need to expand it: gates, evidence links and corrected final identity list
are sufficient. Preparation is not accepted until this correction is complete.
