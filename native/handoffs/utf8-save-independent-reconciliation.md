# UTF-8 streaming save: final independent reconciliation

2026-11-09. Final read-only reconciliation of the frozen UTF-8 Save/SaveAs
implementation. The historical
`utf8-save-independent-review.md` remains the interim draft and is not revised.

## Conclusion

No material defect was found in the frozen review scope. The implementation
streams dirty UTF-8/BOM saves through a bounded writer, hashes the exact bytes
accepted at the temporary-file boundary, publishes atomically, then requires
the committed readback digest to equal the intended digest before advancing
any saved baseline or dirty state.

The earlier in-progress problems are resolved in the frozen source:

- `writeUTF8Atomic` now returns `(digest, error)` and returns the zero digest
  on any lifecycle failure.
- `writeAtomic` checks short writes, and the streaming path converts a short
  underlying write at both buffered and full-buffer boundaries into
  `io.ErrShortWrite`.
- The wrong-digest test compares saved snapshots structurally and also checks
  disk identity, baseline generation, metadata and cache fields without
  weakening the requirement that the live edited buffer remains dirty.
- Save maintains undo history after a successful save.
- SaveAs keeps the original `.14` metadata timing: after successful publication
  it sets `Path`, `IsNew` and `ReadOnly`, then invokes `didSave`; a verification
  failure therefore retains that established post-publication metadata behavior
  while preventing saved-state advancement.
- The benchmark uses `b.N`, constrained by `-benchtime=10x`; fixture creation,
  snapshot capture and oracle setup are outside the timed loops, and both paths
  write the same BOM-prefixed UTF-8 bytes to `io.Discard` with the same full-output
  digest oracle.

## Source review

`native/core/document/atomic.go:17-24` preserves the existing byte-slice write
contract while detecting a short write even when the underlying writer returns
nil. `writeAtomicWriter` at lines 31-69 owns temporary creation, permission
setting, callback write, `Sync`, close, pre-publication conflict check,
exclusive/replacing publication, directory sync and deferred cleanup.

`streamUTF8` at lines 88-103 writes the optional three-byte BOM and
`Snapshot.WriteTo` through a 32 KiB `bufio.Writer`. The buffer targets
`io.MultiWriter(temp, sha256)`, so the digest covers the exact accepted output
bytes, including BOM bytes. BOM, snapshot, flush and short-write errors are
propagated. `writeUTF8Atomic` zeroes the digest when the surrounding sync,
close, check, publication or directory-sync lifecycle fails.

`Document.Save` and `SaveAs` route UTF-8 through the streaming helper while
UTF-16, GBK and GB18030 retain the existing full-buffer `encode` plus
`writeAtomic` path. The existing pre-save and pre-publication conflict checks,
read-only and composition checks, no-op timestamp behavior, exclusive SaveAs
creation and platform publishing code remain in place.

`didSave` at `native/core/document/document.go:288-310` first stats the committed
path, fingerprints the complete current file, and compares that SHA-256 with
the helper's expected digest. `ErrConflict` returns before any change to
`disk`, `saved`, saved encoding/BOM, baseline revision or dirty cache. Only an
exact match records the readback identity and advances the saved baseline.
The established no-rollback limitation after publication remains disclosed.

## Test review

The focused tests substantively cover:

- exact UTF-8 bytes with and without BOM, LF/CRLF, Unicode surrogate-pair text,
  multiple 4 KiB pieces and reopen;
- Save/SaveAs history, selection, dirty cache, undo/redo and disk-token
  advancement;
- untouched UTF-8/BOM save preserving both bytes and timestamp;
- wrong expected digest returning `ErrConflict` without changing the saved
  snapshot, disk identity, baseline, encoding/BOM metadata or dirty-cache
  fields while the live edit stays dirty;
- successful publication returning the exact digest, while a failed atomic
  check returns the zero digest, preserves the target and removes the temporary
  file;
- partial underlying writes below 32 KiB and at/to the full-buffer boundary,
  with both nil-error short writes becoming `io.ErrShortWrite` and injected
  sentinels preserved through `errors.Is`;
- actual partial bytes reaching the temporary destination before failure,
  cleanup of that temporary file, and preservation of the original target.

Existing document and legacy tests continue to cover UTF-16, GBK and GB18030,
unrepresentable characters, read-only and composition refusal, actual external
conflict, no-op saves, exclusive creation and the original atomic failure path.

## Frozen identities

The following actual source and result identities match the owner's published
identities:

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/core/document/atomic.go` | 2,990 | `872E1467775737A341AB202563276F48A4EF255C278312CB006403C1716281C0` |
| `native/core/document/document.go` | 12,493 | `C405449A2BD981EEE09DEFB89F7C113CF40E0254D6673DC99E2945171C08E1F4` |
| `native/core/document/document_test.go` | 7,489 | `CF6B0A37B7874FC6070375F59C4244A219C40B44FD452A9FE4D8E4E4036F1B38` |
| `native/core/document/utf8_save_test.go` | 8,197 | `446F7542B0447FEA4F29B92796F6CB4B32B3AEB249E914BFB279731F78302863` |
| `native/core/document/utf8_save_benchmark_test.go` | 2,223 | `3BE4BC99538EFF5EAEF03E118BFBD12A1B6510C2B65806B24A810FF6EF2FB4EE` |
| `native/handoffs/utf8-save-implementation-result.md` | 7,927 | `142EEE3090FA6A3D2CC950915D35CCAE49C34EE565186B87F8DC0BE7A90209A8` |

The five final raw evidence files also match their reported sizes and hashes:

| Evidence | Bytes | SHA-256 |
|---|---:|---|
| `focused-compile-3-after-import-fix.log` | 177 | `2EEEE23F172465018ECAAF04E7C553512A0958377F83C635C8CF4344F490584E` |
| `focused-utf8-save-tests-6.log` | 2,229 | `2AE05B43F97B8C37E1B4EF1609F824218454C343ECBB1BE8E1E478A1ECC9233D` |
| `benchmark-utf8-output-10x-final.log` | 1,364 | `BE3903AC749E581B4CDA5F8BEB9EB5FA412803AFA04E1318F9ECD1AA94A1AF77` |
| `final-affected-suite-2.log` | 258 | `35106B42C7E4E59F45CDE6347649796FCC509B2551905180B8FA181DFB7FA04F` |
| `final-vet.log` | 112 | `CE9ED7AF2DCFC0B15B632E6F4C30750C1C8889D61A50340E632C8F83F71737F8` |

The final focused test log records exit 0 and every `TestUTF8*` case passing.
The final affected log records exit 0 for document, session and app
(`0.818s`, `1.040s`, `16.036s`). The final vet log records exit 0 with no
diagnostics. These logs are later than the final source/test identities.

The final benchmark records the deterministic 1 MiB and 8 MiB fixture hashes,
10 timed iterations per sub-benchmark, equal verified output digests, and
approximately 33 KiB/op streaming allocations versus roughly 2.1 MiB/op and
16.8 MiB/op for full-buffer output. These values establish encoded-output
allocation behavior only. They are not a whole-Save, process-memory, UI-speed or
resource-matrix measurement.

No tests, builds, launches, benchmarks or product/source/evidence edits were
performed during this reconciliation.
