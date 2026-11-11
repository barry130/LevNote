# Session JSON output: root source acceptance

2026-11-10. Root accepts corrected session-output source for `.16` delivery.
Current accepted runnable package remains immutable `.15` until binary acceptance.

Root read actual Save, private-state callback seam, one-byte forwarding adapter,
tests, byte-oracle benchmark and final logs. Independently hashed all six source/
test files plus result/reconciliation; they match the corrected freeze. Final
independent reconciliation found no remaining material defect or evidence gap.

- [Implementation result](session-output-implementation-result.md):
  SHA256 `BE517D20BB9E127FFA791EB5EC264B41584FF03132BBAD1DA04E27497C8E208E`.
- [Independent reconciliation](session-output-independent-reconciliation.md):
  SHA256 `DBDC93638F1E92328551255EEDA0C100C18F69CDE516CA0F79D4F565C0C5DA42`.

Full session validation still precedes directory/temp creation. Encoder output
is forwarded to the existing private-state atomic temp, dropping only its final
LF and enforcing the exact preflight length. JSON bytes/format, public snapshot,
draft ownership/budgets, permissions, atomic publication and app retry/discard
logic are preserved. Existing []byte private-state callers use the same
lifecycle. Nil callback rejects before filesystem effects.

The adapter retains its first failure across later Write/Finish. This is helper
contract hardening; previous production Encoder already returned its first
error, so no current-runtime data-loss fix is claimed. Final real-pipeline
short/sentinel tests forward bytes into the actual temp, then confirm preserved
old recovery bytes/time and cleanup. Exact-byte oracle includes saved-only and
dirty tabs, nil/empty, false/true optional values, valid floats/Unicode/escapes,
and existing budgets. Benchmark fixtures each match Marshal byte-for-byte.

Final affected suite passed: document0.894s, session1.646s, app17.571s; final
vet exit0, empty output. Actual red/helper/fixture failures remain preserved.
Final fixed10x output-to-io.Discard observations:

- 1,048,701 output bytes: 1,164,120B/op,5allocs ->106,213B/op,4allocs.
- 1,049,114 output bytes/eight tabs: 1,469,992B/op,5allocs ->412,825B/op,4allocs.

Both use preset snapshots, complete validation/oracles outside timing. Standard
library pooled encodeState affects these samples. This removes Marshal's extra
returned full copy; Encoder still holds complete output and draft strings
remain. No token-streaming/fixed-memory, whole-Save, disk, process/RSS or UI/CPU
claim. The difference in measured B/op supports this bounded output-copy change.

Product source frozen for .16 packaging. Root ran no tests/builds/benchmarks or
launches. Follow-on UX work remains read-only planning until package acceptance.
