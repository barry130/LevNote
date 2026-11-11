# UTF-8 streaming save: independent review

2026-11-09. Active for DeepSeek
`read_native_handoffs_resource_workload_next_task_md`. Read-only review, no
product/test/docs/build/launch/measurement changes and no children. Root only
coordinates and reviews; DeepSeek read_native_handoffs_release10_ui_feasibility_task_md owns the frozen core/document implementation/tests. Read
active utf8-save-implementation-task.md including root's in-progress digest
finding. Owner is still implementing; report concrete interim blockers promptly
but reserve final frozen reconciliation until its result exists. Publish
`native/handoffs/utf8-save-independent-review.md`.

Scope is exact Save/SaveAs behavior and allocation evidence, not generic
filesystem redesign. Review shared callback atomic lifecycle, cleanup and
short writes, bounded StringWriter buffering before exact SHA256, BOM count,
UTF8-only routing, nonUTF8 old path, untouched saves, readonly/composition,
permissions and final conflict/exclusive publication. Final readback MUST
compare to expected written digest before saved baseline/dirty state changes;
snapshot saved/selection/history semantics unchanged. A stable hash of arbitrary
readback is not proof that intended bytes were saved. Failure before publication
preserves current target; failure after publication has no guaranteed rollback.

Review meaningful exact-byte/chunk/Unicode tests and sink failure/conflict/
exclusive creation/digest mismatch tests. Benchmark must compare equal output
work for deterministic1/8MiB local fixtures: setup outside timing, consistent
snapshot policy, digest and sink/flush boundary, bounded iteration count. No
RSS/private-byte/whole-app speed claim. An io.Discard microbenchmark can prove
output-generation allocations only; don't call it a full Save measurement.

Freeze verification: owner's actual changed-file identities/raw logs match,
focused tests and final document/session/app suite+vet passed; benchmark raw
output and fixture identities are provided. Do not rerun tests. Read enough to
identify real problems, avoid speculative redesign or duplicate acceptance
framework. Report verified file/line/trigger/impact for material defects; once
fixed and frozen, reconcile only changed identities and conclude in scope.

## Final reconciliation active

The takeover owner has frozen utf8-save-implementation-result.md, SHA256
142EEE3090FA6A3D2CC950915D35CCAE49C34EE565186B87F8DC0BE7A90209A8.
Read the actual final source/tests/raw logs and benchmark. Independently check
the five source/test and five final evidence identities, preserved SaveAs .14
metadata timing, exact expected digest, failure cleanup, and benchmark scope.
Do not rerun tests or write product code. Publish a new
utf8-save-independent-reconciliation.md; leave the historical interim report
intact. Report any material defect before delivery. No .15 build is authorized
until root accepts this freeze.
