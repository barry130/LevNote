# .16 Session-output documentation

ACTIVE2026-11-10: .16 accepted in session-output-release16-root-acceptance.md. Owner DeepSeek
read_native_handoffs_resource_workload_next_task_md. No children. Same six docs:
root README/ROADMAP/CHANGELOG; native README/ARCHITECTURE/VALIDATION, own
session-output-release16-docs-result.md. No source/tests/build/launch/evidence edits.

Current path/version must use native-worker.20261110.16 from actual acceptance.
Explain session recovery writing avoids Marshal's extra full returned encoded
copy by writing Encoder output to existing atomic temp. JSON format is exactly
preserved, including no trailing newline; validation/budgets before filesystem
effects, failure preserves old recovery, current retry/discard behavior intact.
Standard Encoder still buffers a complete JSON value and dirty draft strings
remain; don't say token-streaming/fixed memory/all allocations eliminated.

Use final corrected source/reconciliation/result and actual benchmark numbers,
not first uncorrected run. Pooled encodeState affects bounded samples; only
output-boundary B/op/alloc observations, no process/RSS/UI/CPU/full-Save claim.
Helper sticky errors are hardening, not a demonstrated data-loss fix in prior
production caller. Test and build details belong VALIDATION, contracts belong
ARCHITECTURE. Keep root README concise; link evidence instead of copying tables.

Record current .16 accepted package identities/smoke and .15 UTF8 output history.
Clean close-time session in hidden smoke is not dirty recovery/physical input
verification. Existing .11 worker/.13 visual evidence stay historical. Preserve
all older package/evidence records. Read-only link check and current host/manifest
identity once, freeze result. No rerun tests/builds for docs-only changes.
