# UTF-8 streaming save implementation result

2026-11-09. Implemented and frozen pending independent review. Scope is
`native/core/document` plus `native/handoffs/utf8-save-evidence/`. No product
build, package, worker, UI launch, documentation edit, or resource matrix was
run. `.14` remains immutable.

## Result

UTF-8/BOM `Save` and `SaveAs` now stream the existing `Buffer.Snapshot()` through
a 32 KiB buffer into the temporary file and a SHA-256 writer. The shared
`writeAtomicWriter` lifecycle owns temp-file mode, callback write, sync, close,
pre-publication check, exclusive/atomic publication, and deferred temp cleanup.
`writeUTF8Atomic` returns the digest only when the whole publication lifecycle
succeeds; every failure returns the zero digest and the preserved error.

The existing full-buffer encoding path remains for UTF-16, GBK, and GB18030.
`WriteNewFile`, platform publishing, permissions, optimistic identity checks,
no-op save behavior, composition/readonly refusal, and exclusive target
creation were not redesigned.

`didSave` still compares the committed file hash with the exact digest of the
bytes intended for publication before changing disk, encoding/BOM, saved
snapshot, baseline revision, or dirty-cache state. SaveAs retains the original
.14 metadata timing: after successful publication it updates `Path`, `IsNew`,
and `ReadOnly`, then calls `didSave`. A post-publication verification failure
therefore still cannot promise rollback and retains the existing metadata
timing, but the expected-digest mismatch gates all saved-state advancement.

## Focused coverage

- Exact UTF-8 bytes with and without BOM, LF and CRLF, Unicode surrogate pairs,
  and multiple 4 KiB snapshot pieces; reopen and temp cleanup included.
- Save/SaveAs preserve selection, undo/redo history, dirty state, and disk-token
  baseline advancement.
- Untouched BOM UTF-8 Save leaves bytes and timestamp unchanged.
- Wrong expected digest returns `ErrConflict` before changing saved snapshot,
  token, disk identity, baseline, encoding/BOM metadata, or dirty-cache fields;
  the live edited buffer remains dirty.
- `writeUTF8Atomic` returns the exact encoded digest only after publication;
  a post-write check failure returns the zero digest, preserves the target, and
  removes the temp file.
- A real temp destination is wrapped by injected short/failing sinks. Exact
  `io.ErrShortWrite` is covered both below 32 KiB and at a full-buffer flush;
  the injected sentinel is covered at both boundaries. Each case writes partial
  bytes to the temporary file, preserves the original target, and leaves no
  temporary file.
- Existing package tests continue to cover the legacy encoding paths,
  unrepresentable characters, readonly/composition refusal, missing/externally
  changed targets, exclusive creation, and final atomic checks.

## Final commands

Focused compile:

```powershell
.\native\scripts\go.ps1 -GoArguments @("test","./core/document","-run","^$","-count=1")
```

Result: exit 0, `ok levnote.local/native/core/document ... [no tests to run]`.

Focused tests:

```powershell
.\native\scripts\go.ps1 -GoArguments @("test","./core/document","-run","^TestUTF8","-count=1","-v")
```

Result: exit 0, all exact-byte, state, digest, atomic-publication, and injected
writer cases passed in 0.563s.

Benchmark:

```powershell
.\native\scripts\go.ps1 -GoArguments @("test","./core/document","-run","^$","-bench","^BenchmarkUTF8Output","-benchmem","-benchtime=10x","-count=1","-v")
```

Result: exit 0. Both output paths consumed the same preset snapshot and wrote
the same BOM-prefixed UTF-8 bytes to `io.Discard`; fixture creation and oracle
setup were outside the timed loops, and each sub-benchmark ran the requested
10 iterations through `b.N`.

| Fixture | Source bytes | Fixture SHA-256 | Full-buffer allocation | Streaming allocation |
|---|---:|---|---:|---:|
| 1 MiB | 1,048,576 | `79dc3c40b1efac3f8af6dc91cafbed8f4a5afd307594194f9be07a9ba98b9db7` | 2,105,398 B/op, 4 allocs/op | 33,131 B/op, 8 allocs/op |
| 8 MiB | 8,388,608 | `008cb89935cb613c5939e6b6fa7631711cd3c38eec57eb06dc57e30c1057c266` | 16,785,478 B/op, 4 allocs/op | 33,139 B/op, 8 allocs/op |

These numbers measure encoded-output generation/allocation only. They do not
measure `Document.Save`, process RSS, working set, private bytes, total save
latency, UI speed, or GC behavior, and no such application-level claim is made.

Final affected suite:

```powershell
.\native\scripts\go.ps1 -GoArguments @("test","./core/document","./core/session","./app","-count=1")
```

Result after restoring .14 SaveAs metadata timing: exit 0.

```text
ok  levnote.local/native/core/document  0.818s
ok  levnote.local/native/core/session   1.040s
ok  levnote.local/native/app            16.036s
```

Final vet:

```powershell
.\native\scripts\go.ps1 -GoArguments @("vet","./core/document","./core/session","./app")
```

Result: exit 0 with no output.

## Frozen source and tests

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/core/document/atomic.go` | 2,990 | `872E1467775737A341AB202563276F48A4EF255C278312CB006403C1716281C0` |
| `native/core/document/document.go` | 12,493 | `C405449A2BD981EEE09DEFB89F7C113CF40E0254D6673DC99E2945171C08E1F4` |
| `native/core/document/document_test.go` | 7,489 | `CF6B0A37B7874FC6070375F59C4244A219C40B44FD452A9FE4D8E4E4036F1B38` |
| `native/core/document/utf8_save_test.go` | 8,197 | `446F7542B0447FEA4F29B92796F6CB4B32B3AEB249E914BFB279731F78302863` |
| `native/core/document/utf8_save_benchmark_test.go` | 2,223 | `3BE4BC99538EFF5EAEF03E118BFBD12A1B6510C2B65806B24A810FF6EF2FB4EE` |

## Raw evidence

Final evidence:

| Log | Bytes | SHA-256 |
|---|---:|---|
| `native/handoffs/utf8-save-evidence/focused-compile-3-after-import-fix.log` | 177 | `2EEEE23F172465018ECAAF04E7C553512A0958377F83C635C8CF4344F490584E` |
| `native/handoffs/utf8-save-evidence/focused-utf8-save-tests-6.log` | 2,229 | `2AE05B43F97B8C37E1B4EF1609F824218454C343ECBB1BE8E1E478A1ECC9233D` |
| `native/handoffs/utf8-save-evidence/benchmark-utf8-output-10x-final.log` | 1,364 | `BE3903AC749E581B4CDA5F8BEB9EB5FA412803AFA04E1318F9ECD1AA94A1AF77` |
| `native/handoffs/utf8-save-evidence/final-affected-suite-2.log` | 258 | `35106B42C7E4E59F45CDE6347649796FCC509B2551905180B8FA181DFB7FA04F` |
| `native/handoffs/utf8-save-evidence/final-vet.log` | 112 | `CE9ED7AF2DCFC0B15B632E6F4C30750C1C8889D61A50340E632C8F83F71737F8` |

Preserved intermediate failures:

- Initial GLM compile failure, wrong `writeUTF8Atomic` return arity:
  `focused-document-tests-first.log`, SHA-256
  `5763BD5084A068F11BD5766EF5BB3CC109F261E4C3FB6E271149F289ED852650`.
- Initial wrong-digest fixture failure:
  `focused-document-tests-second.log`, SHA-256
  `572C4B5F59B69CE3FD39ABFF90E96036476A88706855CCC32814A8F119632ED5`.
- Corrective compile failure from a stale moved-test import:
  `focused-compile-2-after-correction.log`, SHA-256
  `799476C94A403491671B6D6D0C5BEB6E1600FD1452063D1E71A8ABFFC01E4B75`.
- Focused test fixture failures before correction:
  `focused-utf8-save-tests-3.log`, SHA-256
  `7E53029562229DC47E6F8BB51C6753D11DB2621D80B07FF910AC2274F02A2754`;
  `focused-utf8-save-tests-4.log`, SHA-256
  `CE834D080338C6CA424F46117D6E1C52545AAA652223F39D07EDA144D5609D2C`.

## Remaining limits

- The streaming path is UTF-8/BOM only. Existing UTF-16 and legacy encoding
  implementations are intentionally unchanged.
- `Save` still captures a `Snapshot` descriptor slice and may retain its
  immutable backing chunks through save; the allocation benchmark starts after
  that shared snapshot capture and measures only encoded-output generation.
- No process-memory or application-speed reduction is inferred from B/op.
- Verification after atomic publication remains unable to roll back a file
  changed concurrently after publication, as before. The intended-versus-readback
  digest comparison prevents that content from becoming the document baseline.
