# Session output: final independent reconciliation

2026-11-10. Final read-only reconciliation of the corrected session-output
source, tests, benchmark and new evidence. No tests, benchmarks, builds,
launches, source edits, or evidence changes were made. The earlier
[initial review](session-output-independent-review.md) remains the historical
interim review.

## Conclusion

No material defect or evidence mismatch was found in the corrected freeze. The
four initial-review items are closed by source and preserved focused evidence:

1. Adapter errors are sticky and later `Write`/`Finish` calls return the first
   error without another sink write.
2. Sentinel and short-write failures now exercise the real
   `writeSnapshotJSON` adapter, injected failing destination and atomic temp
   writer, including old-recovery byte/time preservation and temp cleanup.
3. Both benchmark fixtures are encoded through the new helper and byte-compared
   with the Marshal oracle outside timed loops.
4. The byte oracle contains a saved-only tab with no draft and an explicit true
   `LineNumbers` case.

The valid production path remains consistent with the bounded plan: full
preflight precedes filesystem effects, existing `[]byte` private-state callers
retain the shared atomic lifecycle and short-write behavior, and only
`json.Marshal`'s extra returned encoded copy is removed. `json.Encoder` still
uses a complete internal JSON buffer; no token-streaming, fixed-memory,
whole-Save, process-memory or application-speed claim is supported.

## Corrected behavior

`native/core/session/output.go:17-68` stores the first adapter error. A size,
destination-write or finalization failure makes every later `Write` and
`Finish` return that same error, and the error check precedes any pending-byte
flush. This closes the previous sequence where an excess write could return an
error and a following `Finish` could return nil.

`native/core/session/output_test.go:212-269` verifies sticky excess, sentinel
and short-write failures, including repeated `Write`/`Finish`, exact error
identity through `errors.Is`, and no additional sink calls after failure.

`output_test.go:348-428` sends the real `writeSnapshotJSON` output through
`prefixFailureWriter` to the actual temp writer supplied by
`document.WritePrivateStateTo`. Both short-count/no-error and
nil-or-injected-error paths forward a nonzero prefix, must return
`io.ErrShortWrite` or the exact sentinel, and are followed by checks that the
old recovery bytes and timestamp remain unchanged and no temp file remains.

`native/core/session/output_benchmark_test.go:33-55` constructs the preflight
length and `json.Marshal` oracle, runs the new helper into a buffer, and requires
byte equality and no final LF before timing starts. The 10x timed paths then
perform equivalent output-to-`io.Discard` work. Raw allocation samples remain
scoped observations affected by the standard library's pooled encoder state.

`output_test.go:42-87` now contains a valid saved-only tab (`Draft == nil` plus
absolute `Path`) and explicit true `LineNumbers`; the existing nil/false,
dirty-tab, escaping, Unicode, selection, BOM/encoding, float and bound cases
remain. `Save` still calls `preflightSnapshot` before the private-state
callback, and the existing nil-callback and invalid/over-limit no-side-effect
tests remain intact.

## Evidence reconciliation

All supplied source, test, result and evidence identities match the actual
files:

| Item | SHA-256 |
|---|---|
| `native/core/document/state.go` | `979DFD1E0945EBC57341712C2CE9CF8D5A55B279447382ACCD1F03748C9FB368` |
| `native/core/session/session.go` | `BA3580DBDA6B92984EA9E1844B211FC5FF7ED690CF0E19D30F799620E5402132` |
| `native/core/session/output.go` | `AA06E8E3C5C85BAE8940FABA92673582DF41B8E9F7FBD64AAD1B47DED45407EA` |
| `native/core/document/state_test.go` | `940C51F8ABD1CF731C196276C6A77B4F5EBADE570DE44DD42A56E25AAAB35E24` |
| `native/core/session/output_test.go` | `43739380C6C5E9208FE18D94A9194DA2D812465D678702A2371A0EA33D7DDFA7` |
| `native/core/session/output_benchmark_test.go` | `D3C6DD70CFF82D20F129BC3A9BF273DEA5E111F6CEF1EAF5125B27AD7850B7B0` |
| `session-output-implementation-result.md` | `BE517D20BB9E127FFA791EB5EC264B41584FF03132BBAD1DA04E27497C8E208E` |

The six new evidence hashes also match:

| Evidence | SHA-256 |
|---|---|
| `review-focused-red-1.log` | `F970B73C3D09D505A5692515E43F00652E8C8AAB4F52823B19EF503EE7AE5E23` |
| `review-focused-red-2.log` | `964CAE1AAC6B7AFC07181CB8C0337938742EBD2E1EA5F9987FBE3918F34C4BFF` |
| `review-focused-green-1.log` | `A35C416D36A39D6C5A32CFB8CC7F9955A5DEC2B1F93C061AD1E2EE61187D79AD` |
| `review-session-output-benchmark-1.log` | `4457C3757892F8BCD9B3AB5B8122B028B49008519DE1296E2C1ECC72B49FE8D3` |
| `review-final-affected-suite-1.log` | `B2268E1D62703561D65604CD2F519F035AD660C944A3F291E42321893269F524` |
| `review-final-affected-vet-1.log` | `EAFAF8D4E06C5830A9BD826C69B9E66867C2C8A4AE7D257361E9738061B88CBC` |

The first red log correctly records the non-sticky excess failure and the fixture
mistake that reached the length guard before the intended sink failure. The
second red records excess plus sentinel/short errors being replaced by the
finalization mismatch. The green focused log, corrected benchmark log, final
affected suite and vet are later than the frozen production/test files.

The final affected log records exit 0 with `core/document` 0.894s,
`core/session` 1.646s and `app` 17.571s. The final vet log records exit 0 with
empty output. The benchmark raw values match the result: 1,048,701 bytes with
1,164,120 B/op and 5 allocs/op versus 106,213 B/op and 4 allocs/op; 1,049,114
bytes with 1,469,992 B/op and 5 allocs/op versus 412,825 B/op and 4 allocs/op.

## Scope and remaining limits

The reviewed change does not alter the recovery format, draft ownership or
limits, app retry/discard logic, atomic publication semantics, or the accepted
`.15` package and evidence. It was not built, packaged, launched, or measured
as a process/resource change in this review.

The source is ready for root source acceptance. Package construction remains a
separate future step, and the benchmark supports only the stated
output-boundary allocation observations.
