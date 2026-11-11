# View contract reentry correction: result

FROZEN 2026-11-10 for root and independent reconciliation. This correction is
limited to graphview synchronization/preparation and narrow tests. The original
view-contract result and evidence remain unchanged. No app integration,
adapter advertisement, Stage 2 work, build, launch, product documentation, or
`.18` change is included.

## Defect and correction

`ensureSynced` reported a canceled graph gesture before reading and publishing
the stale prepared state. A reentrant `Host.Report` callback could therefore
remove or replace the buffer and make the subsequent `Buffer.String()` call
panic, or it could release the view, change the source/version, or retarget the
page and let the outer synchronization overwrite callback-owned state.

The synchronization path now captures ownership before the callback: buffer
identity and version, release and page-selection epochs, page index, and the
last synchronized buffer/version state. Immediately after `Report`, it returns
if any captured identity differs. This check occurs before the buffer is read
or parsed cache state is published. Parse-error terminal state is assigned
before its callback, and the draft-page conflict callback is followed by the
same ownership check. A released view is only skipped for this stale outer
synchronization; it is not permanently marked unusable. Normal synchronization
and later released-view reuse remain intact.

No panic recovery or error suppression was added. `PrepareExportRequest` still
returns a copied current-page identity only after its unchanged-owner checks,
and it does not add a second `Flush`.

## Tests

`TestDrawioExportRequestStopsSyncAfterReentrantOwnerChange` starts from a stale
source with an active graph gesture and checks narrow callback cases:

- buffer set to nil;
- buffer replaced;
- view released;
- source version changed in the same buffer;
- page A -> B -> A;
- callback-created pending draft retained without source/history changes.

Each changed-owner case requires an empty request and
`ErrViewExportChanged`; the tests also reject stale cache publication or stale
`lastSynced` state. The pending-draft case requires `ErrViewExportPending` and
preserves the draft and buffer text/history. Existing graphview tests remain in
the same package and passed.

## Changed files

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/modules/graphview/sync.go` | 6,834 | `7E7F448F1F07FE086B406DAAF54BA6EC39CDB1A6E23B85F3D6830B36197FF79F` |
| `native/modules/graphview/export_request_test.go` | 9,644 | `1509104A7A80702781FB426926F3FF660FCD191CC339B5488BE83DEC444087C8` |

## Final validation

Graphview test, through the required wrapper:

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./modules/graphview','-count=1')
```

- Exit: `0`
- stdout: `ok  \tlevnote.local/native/modules/graphview\t10.581s`
- stdout evidence:
  `native/handoffs/drawio-worker-view-contract-reentry-correction-evidence/final-graphview-test.stdout.log`,
  53 bytes, SHA-256
  `7D2119FBBF748C80AD2E5DBA87C8960750B5C771767AE0BCEA748854C2D53CC4`
- stderr evidence:
  `native/handoffs/drawio-worker-view-contract-reentry-correction-evidence/final-graphview-test.stderr.log`,
  0 bytes, SHA-256
  `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`
- exit evidence:
  `native/handoffs/drawio-worker-view-contract-reentry-correction-evidence/final-graphview-test.exit.txt`,
  3 bytes, SHA-256
  `13BF7B3039C63BF5A50491FA3CFD8EB4E699D1BA1436315AEF9CBE5711530354`

Graphview vet:

```powershell
.\native\scripts\go.ps1 -GoArguments @('vet','./modules/graphview')
```

- Exit: `0`
- stdout and stderr: empty
- stdout/stderr SHA-256:
  `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`
- exit evidence:
  `native/handoffs/drawio-worker-view-contract-reentry-correction-evidence/final-graphview-vet.exit.txt`,
  3 bytes, SHA-256
  `13BF7B3039C63BF5A50491FA3CFD8EB4E699D1BA1436315AEF9CBE5711530354`

## Preserved original handoff

`native/handoffs/drawio-worker-view-contract-result.md` remains the original
frozen 4,978-byte result, SHA-256
`75EB1B45B6EF59E7A7785697E7B3BA09A6FCCB2B07CAFC2E45647B39546879AF`.
Its evidence directory still contains the original six files and their recorded
identities, including the 96-byte affected-test stdout with SHA-256
`14E027D6AF4554D9D256A0F199CD4301171D862164470609BB80B0D0049E3277`
and exit `0`.

Stage 2 remains inactive pending root acceptance and explicit activation.
