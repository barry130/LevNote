# Retention draft test review

2026-11-10. Root read the in-progress source/test draft, not a frozen result.
Ownership transfer in parse/edit follows the agreed narrow design. Resolve the
normal compile/vet issues through your existing focused cycle; no new gate.

One substantive test gap to close before final affected suite:
`snapshotRetentionDocument` currently compares page/node/edge/issue COUNTS,
not their values. A failed reparse could corrupt labels, geometry, readonly
metadata, offsets, options or warnings while leaving counts unchanged and that
test would pass. Capture/compare the actual complete state (deep value snapshot
plus source/array identities where useful), then perform a successful later
edit/reparse after the failure. This is the required atomic failure contract.
Avoid assertions that share mutable backing arrays with the object under test.

Also ensure multiple edits assert Dirty, exact untouched source sections/CRLF,
and expected geometry, not only successful calls. Existing exact-byte tests can
cover the complete splice result; cite them rather than inventing a new suite.
Continue only the planned focused and final affected checks.
