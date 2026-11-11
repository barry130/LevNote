# Search/disk checkpoint

## 2026-11-08 12:52 +08:00 — takeover checkpoint green

Root transferred the finite `native/core/search/**` and
`native/core/document/**` checkpoint to DeepSeek after interrupting GLM.
Implementation is frozen after the checks below.

The search traversal uses one stateful directory session at a time and keeps
it open after every short non-empty `nil` read. It closes the session only on
explicit `io.EOF`, a real read error, cancellation, or the disclosed `0,nil`
no-progress condition. The focused tests now prove:

- 257 entries are complete even when a reader returns one entry plus `nil` per
  call; explicit EOF is a separate read;
- reads remain capped at 256 entries and complete after more than one batch;
- a controlled `0,nil` reader is reported as `read-stalled` and its session is
  closed;
- root and nested sessions each close exactly once on the normal path, and the
  active session closes on cancellation;
- total-entry cap truncation requires an additional observed entry, so exactly
  hitting the cap without extra data is not falsely labelled truncated;
- root symlinks are rejected, nested symlinks/non-regular entries are surfaced,
  and every skipped path has a batch issue;
- a file that grows before the limited open is reported as `too-large`;
- `readStable` reads through `io.LimitReader(limit+1)` and rejects an over-limit
  actual read, independent of its pre-stat;
- disk-token tests cover change, missing, invalid, cancellation, same-byte and
  same-size/same-mtime replacement, stale tokens after unchanged Save and
  Revert, read-only baselines, and unstable reads.

Exact passing commands:

```powershell
native\scripts\go.ps1 test ./core/search ./core/document -count=1 -timeout=120s
native\scripts\go.ps1 vet ./core/search ./core/document
```

Results: search `0.835s`; document `0.792s`; vet produced no diagnostics.

The first focused run found three test-fixture inconsistencies, not production
defects: the exact-cap fixture had no second entry to prove truncation; a
normal traversal expected two rather than four read calls despite the required
explicit EOF call; and the stall fixture returned `io.EOF` rather than
`nil,nil`. Those fixtures now model the intended contracts directly.
