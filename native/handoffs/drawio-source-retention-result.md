# Draw.io source-retention result

Status: implementation, corrections, fixed pre/post benchmark, and final affected
test/vet gates are frozen for independent review, 2026-11-10. No package, build,
launch, physical-UI, or whole-process resource claim is made. `.20` and the
accepted warning slice remain immutable. This handoff does not self-accept the
slice or authorize delivery.

## Change

`Parse` normalizes `ParseOptions` and checks `MaxBytes` before its one
caller-input clone. `parseOwnedDocumentBytes` then takes exclusive ownership of
that clone; its private pristine `original` and initial private `source` may
alias the immutable owned buffer, while exported mutable `Document.Source`
receives a separate clone.

`rewriteAndReparse` parses a freshly allocated `next` buffer into a staged
document, carries the existing pristine `original` pointer forward, transfers
the fresh buffer as private current `source`, and commits `*d` only after a
successful parse. A failed reparse returns before assignment, leaving the live
document and its backing arrays untouched. `Serialize()` and `OriginalBytes()`
still return defensive copies.

The buffer accounting is explicit: a fresh parse owns two distinct cloned byte
arrays—the private owned buffer aliased by `original`/initial `source`, plus
the separate public `Source` clone. After a committed edit, the live document
owns three distinct arrays—pristine `original`, fresh private current `source`,
and separate public `Source`—with exact expected bytes in each role. No
caller-visible slice aliases a private buffer.

No public API, decoder, serialization, warning, byte-limit, or BOM behavior
changed.

## Final identities

Five files changed:

- `native/formats/drawio/parse.go` — 36,257 bytes —
  `98A0620AF1F87ECEF1D761EB6AEDD507269DEF34C5530D412D7CEBFBE3E1DA34`
- `native/formats/drawio/edit.go` — 11,233 bytes —
  `9BD482BEAE8088D340BC8070B48610E841B2E602A11328470196E1B1B7C81DEA`
- `native/formats/drawio/source_retention_test.go` — 9,126 bytes —
  `4810CC6B079F278577EAB5859EE84D437828DE2009D34389AF824DD44B2DACAD`
- `native/formats/drawio/source_retention_fixture_identity_test.go` — 558 bytes —
  `2C88FB51F55D6DBE879398B9AD153991B87338EE39C5AD9D210CE2940B3205AD`
- `native/formats/drawio/source_retention_benchmark_test.go` — 3,397 bytes —
  `58CE35C69B79FFADBC1BCBF619F63516D7BECD5ABA2A5F47D1AF5E0033D93493`

`native/formats/drawio/types.go` is unchanged and is not a changed-file identity.

The deterministic benchmark fixture is 33,985 bytes, SHA-256
`04E533B0DA533713F51E106F820A7F4A62D0E0F06CA910EB6FEA9857A3D3E9F`, with 8
pages, 7 uncompressed pages, 169 model nodes, and compressed page 7 verified
`Compressed && ReadOnly`. Fixture and benchmark identities were unchanged
pre/post.

## Tests

All externally reachable byte slices—caller input, public `Source`,
`OriginalBytes()`, and `Serialize()`—are mutated and followed by real later
edits/reparses. Malformed owned reparse is checked against a deep full-document
snapshot and followed by a successful geometry edit. Multiple edits preserve
pristine original bytes, unknown XML, CRLF source, source offsets, the exact
compressed read-only payload and model, dirty state, and warning behavior.
Oversized input is rejected before the caller clone; the corrected focused test
witnesses the three error-construction allocations on that path.

Final affected commands, through the required wrapper, both exited 0:

```powershell
.\native\scripts\go.ps1 -GoArguments @("test","./formats/drawio","./modules/graphview","./cmd/levnote-drawio-svg","-count=1","-timeout=180s")
.\native\scripts\go.ps1 -GoArguments @("vet","./formats/drawio","./modules/graphview","./cmd/levnote-drawio-svg")
```

Package test times were 0.587s, 9.423s, and 0.661s; vet stdout/stderr was empty.

## Benchmark

Identical fixture and benchmark source were used pre/post with:

```powershell
-run ^$ -bench ^BenchmarkDrawioSourceRetention$ -benchtime=20x -benchmem -count=3
```

Medians:

| subbenchmark | pre ns/op | post ns/op | pre B/op | post B/op | pre allocs/op | post allocs/op |
|---|---:|---:|---:|---:|---:|---:|
| `ParseRetained` | 1,082,480 | 1,244,525 | 764,244 | 722,520 | 8,335 | 8,333 |
| `UpdateSerialize` | 2,291,545 | 2,478,320 | 1,733,705 | 1,487,174 | 16,688 | 16,681 |

`B/op` and `allocs/op` are cumulative allocation measurements, not retained
heap, working set, RSS, or whole-process memory. Timing varied across samples
and is not a speed or performance claim.

## Disclosures and limits

The initial `go.ps1 fmt` invocation failed PowerShell argument binding; the
corrected invocation passed. One focused cap test initially expected fewer than
three allocations, but the no-clone error path performs exactly three
error-construction allocations; its oracle was corrected. These were test or
harness corrections, not product-source retries.

The initial benchmark helper populated `sourceSha256Before` after that command
exited and before the production edit; the root evidence note records that
capture defect, so it is not represented as a pre-process timestamp. Later
captures record genuine before/after identities. Raw invocation JSON,
stdout/stderr, fixture probes, and pre/post logs remain in
[drawio-source-retention-evidence](drawio-source-retention-evidence/); the
capture note is [drawio-source-retention-root-evidence-note.md](drawio-source-retention-root-evidence-note.md).

The synthetic fixture does not establish physical UI, input, or process-wide
resource behavior. The parser-level CRLF fixture neither adds nor claims BOM
support; document-layer BOM handling remains unchanged.
