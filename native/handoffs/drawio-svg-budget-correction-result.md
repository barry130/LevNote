# Draw.io SVG budget: narrow correction result

Status: correction implementation and scoped verification COMPLETE 2026-11-10.
`svg.go` and `svg_budget_test.go` are frozen below pending independent
reconciliation. No build, launch, UI action, capture, benchmark, protocol/catalog
change, product documentation, or graphview execution was performed.

## Exact corrections

`native/formats/drawio/svg.go`:

- `warnings` is initialized with `make([]Issue, 0)`, so a successful
  warning-free export returns a non-nil empty issue slice. Every error path
  still returns nil output and nil issues through `ExportSVG`.
- `emitSVG` rejects an already-failed writer before header work and checks the
  writer after the header, each edge path and label, each node shape and label,
  and the final closing fragment.
- After edge-path or node-shape output fails, `emitSVG` returns before optional
  midpoint/text work or issue collection.
- `writeShape` and `writeSVGText` return before shape formatting or scanning a
  label's newlines when the writer is already failed.
- `writeSVGPath` checks the writer at the start of every point iteration, so a
  failed first write stops the remaining path traversal.

`native/formats/drawio/svg_budget_test.go`:

- `TestSVGWarningFreeSuccessPreservesNonNilIssues` exports a plain rectangle
  fixture and requires successful non-empty SVG, `issues != nil`, and
  `len(issues) == 0`.
- `TestSVGEscapingProductionChunkBoundary` places a two-byte `é` immediately
  after 4095 ASCII bytes, then adds `<`, `&`, invalid byte `0xff`, and `中`.
  It proves the first production chunk is 4095 bytes, straddling the rune at the
  4096-byte boundary, and compares the complete chunked output byte-for-byte
  with the existing `escapeAttr(value)` oracle.

The exact SVG oracle, output-limit behavior, warning order, selected-page
precedence, graphview caller, and unchanged error-return contract were not
modified.

## Commands and actual results

Formatting:

```powershell
.\native\scripts\go.ps1 -GoArguments @('fmt','./formats/drawio')
<no stdout or stderr>
exit=0
```

One scoped affected package run after the corrections:

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./formats/drawio','-count=1','-timeout','180s')
ok  	levnote.local/native/formats/drawio	0.641s
exit=0
```

Raw log:
`native/handoffs/evidence/drawio-svg-budget-correction-test.log`.

One drawio-only vet:

```powershell
.\native\scripts\go.ps1 -GoArguments @('vet','./formats/drawio')
<no stdout or stderr>
exit=0
```

Raw log:
`native/handoffs/evidence/drawio-svg-budget-correction-vet.log`.

## Frozen source and evidence hashes

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/formats/drawio/svg.go` | 15,572 | `47F3E584EEF6E2AD981FAE539EE6E2ECA3E93DDBD16F22B7DEC3D5FA086944CF` |
| `native/formats/drawio/svg_budget_test.go` | 10,737 | `79C1F18B12D35C238FA76E5F919BCA65FEE734AA62FC40BEB48FE43797261801` |
| `native/handoffs/evidence/drawio-svg-budget-correction-test.log` | 153 | `1B862566161A1E0BE16A7FC763E120262421DF17FF5DBBED7AB8119C2E236541` |
| `native/handoffs/evidence/drawio-svg-budget-correction-vet.log` | 97 | `F5AF051BEC22BCC210182D93E9B17B65E18A8ACFD7B039D0ABE21D390A5429E9` |

The four protected identities remain unchanged:

| File | SHA-256 |
|---|---|
| `native/formats/drawio/types.go` | `233025C35D1A6C33832E47D8782A07A367954F1EEF70754FB83A5BE436C7D845` |
| `native/formats/drawio/drawio_test.go` | `A1A777227507637928A1920FF9B253F75C6E4A1EFEA697BB087F74CB8D18F189` |
| `native/formats/drawio/testdata/svg-prechange-oracle.json` | `14D7C7166E047979ACDBCBE670EE6918CF44892D54680AAFAC7B0B81944A6611` |
| `native/modules/graphview/graphview_test.go` | `3F8B264305F77F1D6EF2D1468742E8F62DB9248631EB7AB675A1EE213CDE18A0` |

## Evidence history

Both prior `native/handoffs/evidence/drawio-svg-budget-final-test.log` and
`drawio-svg-budget-final-test-2.log` passed. The second was a duplicate run with
only timing differences; it was not a recovery from a failed first final suite.
Earlier genuine red and compile-failure evidence remains unchanged.

No next stage or self-acceptance was performed. The corrected source and tests
remain frozen for independent reconciliation.
