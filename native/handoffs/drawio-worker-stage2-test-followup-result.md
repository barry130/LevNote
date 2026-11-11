# Fourth fixed worker, stage 2 test follow-up result

FROZEN-READY 2026-11-10. Production remained frozen for fresh independent
review. This follow-up changes only `app/export_drawio_worker_test.go` and adds
the evidence below. The original Stage2 result, logs, captures and hashes are
preserved. No `.19` build, package, managed worker request, host launch,
physical-UI run or resource measurement was performed.

## Coverage corrections

- The case formerly called `A-B-A` is now truthful `B-A-B`. Starting from page
  B, it uses two real Tester page-button transitions, B to A then A to B, and
  requires both transitions to increase the epoch token.
- `TestDrawioWorkerFlushesRealSecondPageDraftBeforePrepare` navigates to the
  real second page and composes `第二页待导出🙂` in the real `节点文本` field.
  Before Flush, it proves the pending composition has not reached the
  authoritative buffer, version or history.
- Export then asserts `RequestPage{Index:1, ID:"p2"}`, an exact post-Flush
  source snapshot and size, version increment by one, one added undo entry,
  and zero redo entries. The Save callback observes that same exact
  one-transaction source/state, while the original disk file and BOM remain
  unchanged. One Undo restores the exact pre-Flush baseline.
- The stale-publication table now also includes post-dialog view replacement
  and same-buffer version change, alongside editor and tab-name changes and the
  existing page/reparse/release cases. Every stale case checks that no output
  file was published, the export job was released, and the result payload and
  warnings were cleared.
- Conflicting draw.io field-draft refusal is inherited from the accepted
  lower-layer test
  `modules/graphview/safety_test.go:TestDrawioDraftTracksPageIDAcrossReorderAndBlocksChangedField`.
  That test proves the pending draft survives and conflicted Export returns
  `ErrDraftConflict`; no duplicate app matrix was added.

The follow-up also corrects the earlier summary wording:
`PrepareExportRequest` is **not** side-effect-free; it may synchronize stale
cache as part of preparation. Only `ExportRequestToken` is a pure token getter.
The synthetic runner publication proves that the intended post-Flush bytes and
request page are forwarded and that the returned payload is published exactly.
Exact SVG renderer output remains lower-layer/real-worker evidence and is not
claimed here.

## Verification chronology

The test-only first compile attempt failed because the then-current line used
the nonexistent `text.History` type:

```text
app\export_drawio_worker_test.go:145:25: undefined: text.History
FAIL	levnote.local/native/app [build failed]
```

The failure is preserved under both `focused-app.*` and
`focused-app-attempt1.*`; their streams and exits have identical hashes. This
is duplicate preservation, not two additional runtime attempts. After changing
the variable type to `text.HistoryStatus` and running the second formatting
check, the focused app retry passed in `1.878s`.

The single final app suite ran after the correction:

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./app','-count=1')
```

Exit `0`, with raw stdout:

```text
ok  	levnote.local/native/app	17.000s
```

The single final vet command also exited `0` with no stdout or stderr:

```powershell
.\native\scripts\go.ps1 -GoArguments @('vet','./app')
```

No graph/editor suite, capture generation or managed worker was repeated.

## Frozen identities

| Artifact | Bytes | SHA-256 |
|---|---:|---|
| `app/export_drawio_worker_test.go` | 14,855 | `F8FF9ADBF52F1ED68161306FB4D472A91C500FC59B3A90BA24278C579F4C2BD2` |
| `handoffs/drawio-worker-stage2-result.md` (unchanged) | 5,966 | `19DD58AB7589CC1DBDFE7A6B454A1C28F76968D24C2A4D74A30CF3BDC9BEA283` |

Raw follow-up evidence:

| Artifact | Bytes | SHA-256 |
|---|---:|---|
| `drawio-worker-stage2-test-followup-evidence/focused-app-attempt1.stdout.log` | 52 | `C82A3FE1200FD46EE0959310AE9ED8A92AD62A2A35FF9CBC89BA4E4C98D39046` |
| `drawio-worker-stage2-test-followup-evidence/focused-app-attempt1.stderr.log` | 126 | `FCD6FEB28B422FED3F3879A1F89EE710316438748B8BA574B0B7C8531D22AA63` |
| `drawio-worker-stage2-test-followup-evidence/focused-app-attempt1.exit.txt` | 3 | `F1B2F662800122BED0FF255693DF89C4487FBDCF453D3524A42D4EC20C3D9C04` |
| `drawio-worker-stage2-test-followup-evidence/focused-app-attempt2.stdout.log` | 38 | `4D7105F4083224254D818678ABC4F19986AA85F7F698B5D98C30F7F013A8AC91` |
| `drawio-worker-stage2-test-followup-evidence/focused-app-attempt2.stderr.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `drawio-worker-stage2-test-followup-evidence/focused-app-attempt2.exit.txt` | 3 | `13BF7B3039C63BF5A50491FA3CFD8EB4E699D1BA1436315AEF9CBE5711530354` |
| `drawio-worker-stage2-test-followup-evidence/final-app.stdout.log` | 39 | `0A0CD700FF3A28013B2025DE20B2D1F32317D44095457C9E1D80B0897EA6205D` |
| `drawio-worker-stage2-test-followup-evidence/final-app.stderr.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `drawio-worker-stage2-test-followup-evidence/final-app.exit.txt` | 3 | `13BF7B3039C63BF5A50491FA3CFD8EB4E699D1BA1436315AEF9CBE5711530354` |
| `drawio-worker-stage2-test-followup-evidence/final-vet.stdout.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `drawio-worker-stage2-test-followup-evidence/final-vet.stderr.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `drawio-worker-stage2-test-followup-evidence/final-vet.exit.txt` | 3 | `13BF7B3039C63BF5A50491FA3CFD8EB4E699D1BA1436315AEF9CBE5711530354` |

The mirrored `focused-app.*` and two `gofmt*` evidence sets remain in place;
the focused failure hashes above describe both identical failure copies.

## Scope and next gate

This is MyGo Tester and controlled-runner evidence. It does not establish
physical Windows input/IME behavior, whole-process or peak memory, user-observed
latency, or exact renderer output for arbitrary draw.io documents. Fresh GLM
independent review of the frozen production remains the next gate. Delivery is
not authorized or claimed by this follow-up.
