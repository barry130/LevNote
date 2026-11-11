# Binary copy receipt source acceptance

ACCEPTED2026-11-10 by root. Read actual publication/adoption implementation,
existing SaveAs delegation and meaningful new tests, then saved exact commands,
outputs/exits and frozen identities. No tests/builds rerun by root.

Publication works from immutable snapshot only. Receipt carries source token,
verified eager destination identity, outcome and monotonic serial; no mutable
owner or full bytes. Owner adoption checks source authority/current baseline,
snapshot token and verified receipt before mutation; new descriptor shares bytes
and advances revision. Same-path verification retains revision and is replay
protected. Already committed destination outcome survives failed/stale adoption.

Tests include real worker/owner reload barrier, same-path identity/time checks,
postcommit cancellation using production digest verification, destination races,
actual destination mutation and subsequent conflict detection, deleted source
held bytes and rebound second save. Focused9 tests/12subtests PASS0.553s; full
document suite PASS0.973s; vet exit0. Atomic helper unchanged. No repeat needed.

All four frozen hashes match current files:
- binary.go: b60483712068daa13e0293dd70873b5e132df183e59c05599680efa18ecc427e
- binary_copy.go: 51117335de9a7c5d95b85f2c1e603b05d3414f79e4a22987ebeef63c6ab85fa2
- binary_test.go: 4a462ccd2c0f0611c9928347fee9de928d3e2d91a5db16f1ce6c4cdf5e6ebcb9
- atomic.go: 33d45a65253972d8547a5dc7acd1ac9fa04c0540ef11dd3de207676801a232cc

Receipt identity is publication-time evidence, not filesystem CAS. No race
detector or app/UI/driver/process-resource claim. Root now activates app-only
integration; session and view remain separate owners until their review closes.
