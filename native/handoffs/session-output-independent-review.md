# Session output: initial independent review

2026-11-10. Read-only review of the frozen session-output implementation and
evidence. No tests, benchmarks, builds, launches, source edits, or evidence
changes were made. This is the interim review; final reconciliation waits for
the owner's corrected freeze.

## Conclusion

No actual production correctness defect was found in the current valid-save
path. The bundled Go encoder returns immediately when the adapter reports a
write error, so the current implementation does not demonstrate data loss,
partial publication, or false saved-state advancement on that path.

The freeze is not ready for package acceptance because one helper-state defect
and three verification gaps remain:

1. `exactSessionWriter` does not retain an earlier error. A write can fail and a
   later `Finish` can return nil if the byte count happens to satisfy its final
   test. This is a bounded helper-contract defect, not a demonstrated current
   production failure.
2. The real-temp failure tests do not exercise
   `writeSnapshotJSON -> injected failing destination -> actual temp writer`.
   They cover adapter errors against an in-memory failure writer and atomic
   cleanup with a callback that manually returns an error after writing a
   prefix.
3. The benchmark computes a Marshal oracle and length outside timing, but never
   encodes and byte-compares each actual benchmark snapshot through the new
   helper outside timing.
4. The output oracle's nominal saved-tab case still has a non-nil `Draft`, and
   the output-oracle table does not include explicit true `LineNumbers`.

These are narrow corrections. They do not justify a filesystem, retry, or
encoder redesign.

## Findings

### 1. Adapter errors are not sticky; current production caller remains safe

`native/core/session/output.go:29-56` enforces the expected length only on each
individual `Write` and in `Finish`. It does not store a prior excess or
destination-write error. Thus this sequence is possible:

```text
Write(valid expected JSON bytes plus final LF) -> nil
Write(extra bytes)                             -> errEncodedSessionTooLarge
Finish()                                       -> nil
```

The same issue applies if a destination write returns an error and a caller
ignores it, then supplies another successful write whose final count and LF
shape appear valid.

Current production impact is bounded. Bundled Go
`native/tools/go/src/encoding/json/stream.go:245-248` records and returns the
first `Write` error from `Encoder.Encode`; this code ignores the returned count,
which is why the adapter correctly performs its own short-write check.
`writeSnapshotJSON` immediately returns that error at `output.go:72`, so its
normal caller never proceeds to `Finish`.

The original task explicitly required sticky errors, and a truthful helper
contract should not make a failed write look successful. Add a sticky `err`
field, return it from subsequent `Write` and `Finish`, and do not flush another
pending byte after a failure. A focused sequence test should require the exact
original error from both the failed excess write and finalization.

### 2. Real atomic-pipeline sink failures are still unproven

`native/core/session/output_test.go:281-357` confirms that `Save`-style output
with a wrong expected length fails in the actual atomic temp lifecycle and
preserves the old recovery file. For the sentinel and short-write cases,
however, lines 314-327 manually write a prefix to the callback's temp writer
and then return the error. This validates atomic cleanup, but bypasses
`exactSessionWriter`.

`output_test.go:177-202` exercises adapter short/sentinel errors, but its
destination is only `failureWriter`; no real temp file or private-state
lifecycle is involved.

The implementation result therefore overstates the sentence that sentinel and
short-write failures were exercised through a real temp writer. Correct
evidence should wrap the writer supplied by `document.WritePrivateStateTo`,
forward a nonzero prefix to that real temp file, then return either a short
count without an error or the injected sentinel. The test should call the real
`writeSnapshotJSON` composition and require the exact error, byte/time
preservation of the previous recovery, and no leaked temp file. The existing
callback-only cleanup cases can remain identified as such.

### 3. Benchmark has no actual new-output byte oracle

`native/core/session/output_benchmark_test.go:33-44` validates and hashes the
old Marshal output outside timing, but the actual helper path at lines 63-70
writes only to `io.Discard`. A sink that discards bytes cannot prove the new
helper emitted the same fixture bytes.

Before the fixed-10x timed sub-benchmarks, encode the same actual snapshot
through `writeSnapshotJSON` into a buffer and require `bytes.Equal` with the
Marshal oracle, no terminal LF, and the expected byte count. Fixture hashes,
Marshal, helper oracle output, preflight, and comparison must remain outside
timed loops. The current allocation observations can be evidence only after
that setup is corrected; one rerun is justified because benchmark setup
changes.

### 4. Two requested byte-shape oracle cases are absent

In `output_test.go:42-60`, the tab called `saved` still sets
`Draft: &saved`, so it is a draft-bearing tab rather than the required
saved-only `Draft == nil` shape with a valid absolute `Path`. The same table
covers nil and explicit false `LineNumbers`, but not explicit true.

Add one valid saved-only tab and explicit true `LineNumbers` to the existing
Marshal comparison. No redundant fixture matrix is needed.

## Reconciled parts

- `Save` sets `Version` and fully invokes `preflightSnapshot` before calling the
  callback writer. Invalid and just-over fixtures assert that no destination
  directory is created.
- A nil private-state callback is rejected before directory creation.
- Existing `[]byte` `WritePrivateState` callers remain on the same atomic
  writer and preserve the short-write check.
- Valid focused output cases compare helper bytes with `json.Marshal`, including
  nil versus empty tabs, controls and HTML escaping, Unicode/U+2028/U+2029,
  draft-bearing tabs, selections, BOM/encodings, omitted fields, false
  `LineNumbers`, float32 scroll extremes, successful Save/Load, and representative
  bounds.
- The helper removes only the encoder's final LF and does not claim token
  streaming or fixed-memory encoding. The encoder still owns the complete
  encoded JSON buffer.
- Actual source/test hashes match the implementation result. Raw final suite
  (`document` 0.761s, `session` 1.441s, `app` 15.753s) and vet logs match the
  report; no reviewer rerun was used.

## Frozen identities reviewed

| File | SHA-256 |
|---|---|
| `native/core/document/state.go` | `979DFD1E0945EBC57341712C2CE9CF8D5A55B279447382ACCD1F03748C9FB368` |
| `native/core/session/session.go` | `BA3580DBDA6B92984EA9E1844B211FC5FF7ED690CF0E19D30F799620E5402132` |
| `native/core/session/output.go` | `14B81AA3F0A113A2ACFD184D56B4D1B6ECEB1170CC8041B3EF9D2AE8FF3648C3` |
| `native/core/document/state_test.go` | `940C51F8ABD1CF731C196276C6A77B4F5EBADE570DE44DD42A56E25AAAB35E24` |
| `native/core/session/output_test.go` | `67A89F8AE33EC233FD3E2ADE6EDA156EE038C29DA95B7F241CD82BC430DA12F1` |
| `native/core/session/output_benchmark_test.go` | `F7CCEF528424C6E08B8392BAD85D0C465741D5496D6621FCF0F0244B96C3D694` |

The next review should reconcile only corrected source/tests, new correction
failures and final raw evidence; the green logs above must not be reused for
changed files.
