# Worker-safe exact binary publication and owner adoption

ACTIVE2026-11-10 for Astra read_native_handoffs_tab_workflow_resume_task_md.
Bitmap disposal is accepted; leave vendor/patch/guard evidence frozen. Binary
core accepted too, but its current SaveAs mutates a same-goroutine BinaryFile.
Implement the narrow missing async publication contract now, so the later UI
can copy at most16MiB without blocking or racing mutable authority. No extra
design-only stage. No children. Other owners: GLM core/session, DeepSeek raster.

Own core/document/binary.go, binary_test.go and at most one new binary copy
source/test file plus scoped handoffs/evidence. Minimal reuse of existing private
atomic helper is allowed if necessary; preserve public text APIs and semantics.
No app/session/raster/vendor changes, build/host/worker/resource runs.

Provide an operation accepting only ctx, destination and immutable snapshot,
usable on a worker; it performs exact exclusive publication and streaming digest
verification and returns truthful Committed/Verified/NoRewrite outcome plus an
opaque receipt. No live BinaryFile capture/mutation, no global mutable hooks,
no file reopening into a second source-size buffer, no compressed byte copy.
The receipt is small: source authority/token and verified destination identity,
path and operation status only. Do not retain snapshot/bytes/reader/handle in it.
Private fields prevent callers forging identity/adoption. Zero receipt rejected.

Add an explicit owner-thread adoption method that checks receipt validity,
current SameBaseline/source authority, and supplied snapshot token before any
mutation. On new destination success increment revision and return a fresh
immutable snapshot descriptor sharing the same accepted data with the new token;
the old snapshot stays immutable/stale. This avoids rereading the disk or losing
the active accepted image after SaveAs. Same-path receipt performs verification
without rewrite/revision change. Reject stale/reloaded/foreign/replayed receipt
without mutating current path/baseline; preserve already committed destination
and surface that fact. Failed/committed-but-unverified receipts cannot adopt.
The host must report a completed copy if the originating tab closes/changes
before UI adoption; destination success is not rolled back or hidden.

Make existing synchronous SaveAs use the same publication/adoption machinery
where practical, preserving current signatures/outcome semantics and meaningful
tests. Avoid a second copy implementation. A receipt describes the destination
verified at publication time; external writers may change it afterward, as in
existing saves. Document this limit and use existing disk-conflict observation,
not claims of filesystem CAS or synchronous UI rereads on adoption.

Tests must cover immutable worker publication with owner unchanged, successful
adoption/rebound snapshot and second SaveAs from it, foreign/stale/zero/replayed
receipt, reload between publish/adopt, same-path no rewrite, canceled before and
after publication, source deleted held copy, committed-but-unverified refusal
and existing exclusive-race semantics. Use actual production helpers and per-call
hooks only where needed. Preserve accepted tests; no fixtures that fake result
flags or compare lazy Windows Stat identities after rename.

Use scripts/go.ps1; save exact commands/output/exits in
native/handoffs/binary-copy-receipt-evidence-20261110. Run focused new tests then
one final affected document suite/vet after actual fixes. Publish
binary-copy-receipt-implementation-result.md with APIs, ownership, limitations
and scoped frozen identities. Root reviews, then authorizes host/view integration
using settled session/decode contracts. No package for this helper alone.
