# Draw.io source-retention plan

Status: READ-ONLY PLAN, 2026-11-10. No product/test edit, Go command, build,
launch, capture, or measurement was performed. `.20` inputs and evidence remain
frozen. Implementation is a later increment after the structural-warning work
and an explicit ownership assignment.

## Finding and safe boundary

`native/formats/drawio/parse.go` currently retains three source-sized slices
after a successful initial `Parse`: private `original`, private `source`, and the
independent public `Document.Source`. The caller's input is not retained: the
parser aliases it only while parsing.

The private pair is only read. `OriginalBytes()` and `Serialize()` clone their
results; all edits build a fresh `next` buffer (`patchStartTag`/`insertAt`), and
`setTagAttributes` copies its tag before changing attributes. No edit mutates
`original` or `source` in place. A failed reparse leaves the current document
unchanged because `rewriteAndReparse` assigns only after success.

The public mutable `Source` cannot share private backing storage. Current scoped
production callers do not read it, but preserving its exported API and mutation
isolation is mandatory.

## One bounded optimization

Add one private ownership-aware parse path and use it at the two existing parse
seams:

1. Public `Parse(data, options)` clones the caller input exactly once and passes
   that owned slice to the private parser. On success, the owned slice backs both
   private `original` and initial private `source`. They alias only while that
   buffer is immutable. `Document.Source` remains a separate clone.
2. `rewriteAndReparse(next)` keeps the old `original` pointer without cloning it.
   After `next` reparses successfully, transfer ownership of `next` to private
   `source`; the private parser must already have produced one separate clone for
   `Source`. Remove the current redundant `original`, `source`, and `Source`
   clones in this path.
3. Document clearly that `rewriteAndReparse` accepts a freshly allocated,
   exclusively owned `next`; its only callers already construct such a buffer.
   Do not expose this helper or alter the public API.

Nominal retained source-sized arrays therefore fall from three to two after a
fresh parse: one shared private immutable buffer plus public `Source`. After the
first edit, three arrays remain semantically required: pristine original,
current private source, and independently mutable public source. The edit path
also stops discarding a full set of clones made by `parseDocumentBytes`.

This does not change input ownership, `Document.Source` mutability, return-value
isolation, exact no-edit bytes, unknown XML, compressed-page read-only behavior,
or transactional rollback. `OriginalBytes()` and `Serialize()` continue to
allocate a defensive copy on every call. Caller-level conversions such as
`string(raw)`/`[]byte(raw)`, serialization validation, and buffer commit copies
are outside this slice.

`graphview` retains the parsed document while the buffer version is unchanged
and drops `v.diagram` on cache invalidation or `Release`; that lifecycle remains
unchanged. The graph edit, field draft, and SVG-worker callers also benefit from
the same parser allocation reduction, but no process-memory claim follows
without measurement.

## Required regressions

Add focused tests under `native/formats/drawio`:

- Parse a caller-owned byte slice, mutate every caller byte after return, and
  prove model data, `Serialize()`, `OriginalBytes()`, a later edit, and a reparse
  are unchanged.
- Mutate `Document.Source`, a returned `OriginalBytes()` slice, and a returned
  `Serialize()` slice; prove each mutation cannot affect later serialization or
  edits. An exact backing-slice identity assertion may additionally pin the
  intended two-array/three-array ownership states, but is not the only proof.
- Parse a BOM/CRLF multi-page fixture containing unknown XML and a compressed
  page. No-edit serialization is exact; a real text or geometry edit preserves
  unknown XML and the compressed payload; the compressed page remains readable,
  read-only, and unchanged.
- After more than one successful edit, `OriginalBytes()` still equals the initial
  source, `Dirty()` is true, and reparse of `Serialize()` has the expected model.
- Exercise the private reparse seam with deliberately invalid XML. It must return
  an error with source, original bytes, dirty state, pages, nodes, and edges
  unchanged. This is a narrow ownership/rollback guard because public setters
  reject invalid XML before reparse.

Reuse the existing `TestUncompressedParseModelAndExactRoundTrip`,
`TestTextAndGeometryMutationsPreserveUnknownXML`, and
`TestCompressedPageRendersReadOnlyAndRoundTrips` coverage rather than copying it.

## Fair allocation check

After the later implementation gate opens, add the same test-only benchmark to
the unchanged parser first and record a baseline, then rerun it unchanged after
the source edit. Use a representative multi-page fixture with unknown XML and a
compressed page:

- `BenchmarkDrawioSourceRetention/ParseRetained`: parse on every iteration and
  keep the resulting `Document` alive through `runtime.KeepAlive`.
- `BenchmarkDrawioSourceRetention/UpdateSerialize`: parse, apply one real text
  edit, call `Serialize`, and keep the required results alive.
- Set `b.ReportAllocs()` and `b.SetBytes(int64(len(source)))`.

Use the project wrapper and identical fixed iterations for both checkpoints:

```powershell
.\native\scripts\go.ps1 -GoArguments @("test","./formats/drawio","-run","^$","-bench","^BenchmarkDrawioSourceRetention$","-benchtime=20x","-benchmem","-count=3")
```

Record source hashes, full command, exit status, and raw logs. Compare
`ns/op`, `B/op`, and `allocs/op`; the expected evidence is one fewer
source-sized retained/allocation copy and fewer redundant edit-path clones.
These parser/fixture numbers do not establish whole-application RSS, idle
working set, CPU use, or behavior under physical UI input.

## Sequence

1. Finish and accept the queued structural-warning increment and `.20` delivery.
2. Obtain explicit ownership for this source-retention slice.
3. Add the benchmark fixture while product source is unchanged and capture the
   baseline.
4. Apply only the ownership-aware parse/reparse change and focused regressions.
5. Run focused tests and the final affected `./formats/drawio`,
   `./modules/graphview`, and draw.io-worker suites plus vet; rerun the same
   benchmark.
6. Publish a result with exact hashes and raw evidence. Do not modify `.20` or
   prior accepted artifacts, and make no physical-input or process-memory claim.
