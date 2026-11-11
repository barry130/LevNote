# Binary core implementation activated

ACTIVE2026-11-10 for GLM read_native_handoffs_binary_image_glm_task_md. Root accepts
the narrow design direction with the clarifications below. Implement now; hold
test execution until root gives a window (resource matrix will run shortly).
No children; context cap as before. Own core/document binary.go/new tests and
minimal atomic.go internal outcome extraction only. No app/session/view/vendor/
plugins/product docs changes. No product package build or launch.

Read binary-image-prerequisite-design.md and binary-image-api-review-guidance.md.
Clarifications before coding:
- Immutable snapshot copy after original changes/deletion is allowed. Adoption
  checks snapshot belongs to current file baseline/revision, not current disk;
  only reacquisition and same-path no-rewrite verify original disk again.
- Use private BinaryFile path/state with read-only accessors, not a caller-
  writable exported Path that can invalidate the baseline. Binary authority is
  always non-editable; distinguish source filesystem permissions if exposing
  them, so ReadOnly=false never implies pixel editing or ordinary Save.
- Avoid double full reads when opening/reloading then immediately displaying.
  OpenBinary and explicit Reload should return the accepted BinarySnapshot with
  the baseline (exact ergonomic API at your discretion, update the design).
  The file itself retains no bytes; caller decides when to drop snapshot.
- Snapshot Reader/ReaderAt copy into consumer storage. Prefer one public
  streaming method WriteTo; omit redundant CopyTo unless required. It must not
  expose private backing storage through concrete type assertions or writer.
- Context cancellation should bound stable read/copy work. No known raw file
  byte mismatch may establish a baseline, even if metadata happens to match.
  Reject unknown kind values; Unknown/PNG/JPEG are explicit metadata only.
- atomicOutcome needs Committed; do not add unused Digest. Snapshot already has
  expected digest. Preserve existing text wrapper signatures/error behavior.
  Directory sync errors are post-commit. Do not delete committed destination or
  pretend it was never created. Destination verification failure leaves old
  BinaryFile baseline/path and reports Committed=true, Adopted=false.
- Clearly document same-goroutine ownership for mutable BinaryFile. Immutable
  token/snapshot are safe across goroutines; future UI work must capture work
  without races. If a small token-based snapshot read function is needed for
  async host use, add it now rather than copying live mutable file into workers.
  Keep it narrowly binary-specific and avoid a generic document framework.

Substantive tests per design: exact original bytes, source changed/replaced/
missing, explicit reload generation, aliasing writer, size/cancel/read errors,
exclusive destination race, same-path no rewrite, post-commit error and baseline
adoption. Use private per-call seams for deterministic failure tests rather than
global mutable hooks. Note Go io.Writer may retain/mutate scratch but never gets
snapshot's private data. Old snapshots remain readable after baseline changes;
old adoption authority must be rejected before writing.

Write implementation/test files, gofmt through approved wrapper when scheduled.
Publish code-ready checkpoint and request root's internal execution window.
Then run substantive focused regressions and one final affected document suite/
vet after corrections, preserving real logs/exits. No unrelated suites. Freeze
binary-core-implementation-result.md with API/file identities and limits.
