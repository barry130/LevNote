# Draw.io SVG budget: independent reconciliation

Status: **RECONCILED; NO REVIEW BLOCKER** 2026-11-10. This was a read-only
reconciliation of the frozen correction. I ran no tests, vet, builds, or
launches.

The original review remains immutable at 5,919 bytes, SHA-256
`34D5B7CFD93B544F56EBD5016FD4432F6B2E1FAF215247C665E7F6FCCCC5A0F2`.

## Finding 1: resolved

The successful warning-free API shape is restored.
`native/formats/drawio/svg.go:110` now initializes
`warnings := make([]Issue, 0)`, so the count pass returns a non-nil empty
slice when no issue is appended. The existing outer failure paths still discard
that slice and return nil output and nil issues.

`native/formats/drawio/svg_budget_test.go:217-226` provides a plain-rectangle
export and directly requires non-empty SVG, `issues != nil`, and
`len(issues) == 0`. This is substantive and closes the gap left by the exact
oracle, whose fixture has four warnings, and by the older length-only checks.

## Finding 2: resolved within the accepted bounded-fragment scope

Unbounded continuation after writer failure is stopped:

- `emitSVG` checks an already failed writer before header work and after each
  header, edge-path, edge-label, node-shape, node-label, and closing stage.
- After an edge path fails, it returns before midpoint calculation or optional
  edge-label handling (`svg.go:132-147`).
- After a node shape fails, it returns before optional node-label handling
  (`svg.go:197-206`).
- `writeSVGText` returns before counting newlines when the destination is
  already failed (`svg.go:400-404`).
- `writeShape` returns before shape formatting when the destination is already
  failed (`svg.go:357-360`).
- `writeSVGPath` checks at the start of every point iteration, so it no longer
  traverses later points after a failure (`svg.go:433-444`).

As requested by the active review note, this reconciliation does not claim
zero instructions after the first failed writer call. If the separator write
for the current path point fails, that iteration can still evaluate the bounded
numeric arguments for that same point. Similar bounded continuation can occur
inside the current shape/text fragment. Crucially, the implementation no longer
walks remaining path points, computes optional label geometry after a failed
path, or scans a label after its shape/path stage has failed. That satisfies the
material intent of the correction without adding a framework.

## Production escaping boundary

`TestSVGEscapingProductionChunkBoundary` is substantive. At
`svg_budget_test.go:201-215`, it places the two-byte rune `é` immediately after
4,095 ASCII bytes, asserts that the first production-sized chunk ends at byte
4,095, and then compares the complete chunked result byte-for-byte with the
independent `escapeAttr` wrapper around `xml.EscapeText`. The fixture also
crosses escapable `<`/`&`, invalid byte `0xff`, and another multibyte rune.
This directly exercises the production `svgEscapeChunkSize` boundary without a
large matrix.

## Identities and evidence

The corrected files match `drawio-svg-budget-correction-result.md`:

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/formats/drawio/svg.go` | 15,572 | `47F3E584EEF6E2AD981FAE539EE6E2ECA3E93DDBD16F22B7DEC3D5FA086944CF` |
| `native/formats/drawio/svg_budget_test.go` | 10,737 | `79C1F18B12D35C238FA76E5F919BCA65FEE734AA62FC40BEB48FE43797261801` |

The four protected identities still match the original review:

| File | SHA-256 |
|---|---|
| `native/formats/drawio/types.go` | `233025C35D1A6C33832E47D8782A07A367954F1EEF70754FB83A5BE436C7D845` |
| `native/formats/drawio/drawio_test.go` | `A1A777227507637928A1920FF9B253F75C6E4A1EFEA697BB087F74CB8D18F189` |
| `native/formats/drawio/testdata/svg-prechange-oracle.json` | `14D7C7166E047979ACDBCBE670EE6918CF44892D54680AAFAC7B0B81944A6611` |
| `native/modules/graphview/graphview_test.go` | `3F8B264305F77F1D6EF2D1468742E8F62DB9248631EB7AB675A1EE213CDE18A0` |

The correction evidence is internally consistent:

- `drawio-svg-budget-correction-test.log`: drawio package passed in 0.641s,
  exit 0, SHA-256
  `1B862566161A1E0BE16A7FC763E120262421DF17FF5DBBED7AB8119C2E236541`.
- `drawio-svg-budget-correction-vet.log`: vet produced no stdout or stderr and
  exited 0, SHA-256
  `F5AF051BEC22BCC210182D93E9B17B65E18A8ACFD7B039D0ABE21D390A5429E9`.

The correction result now also records that both original final-suite logs
passed and the second was a timing-only duplicate. The earlier red, compile
failure, and overly narrow selection evidence remain retained rather than being
replaced by the final green run.

The correction is ready for root source acceptance. Review scope needs no
additional test, graphview repeat, 64 MiB matrix, benchmark, build, or package
gate. Release `.18` remains inactive.
