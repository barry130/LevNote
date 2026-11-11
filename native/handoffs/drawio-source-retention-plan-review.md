# Draw.io source-retention plan review

Date: 2026-11-10. Read-only review of
[drawio-source-retention-plan.md](drawio-source-retention-plan.md) against the
current `Parse`, edit/rewrite, serialization, graphview cache, and draw.io worker
call paths. No product/docs edits, tests, benchmarks, builds, launches, or
measurements were run, and `.20` packaging inputs remain untouched.

## Conclusion

The ownership design is safe and sufficiently narrow. There is no structural
blocker, provided implementation adds one mandatory ordering guard:

1. normalize `ParseOptions`;
2. check `len(data) > options.MaxBytes`;
3. only then make the single public `Parse` input clone.

The proposed preclone must never allocate rejected or oversized caller input.
Keep the existing core cap and hard maximum unchanged.

The minimum retained ownership remains two source-sized arrays after a fresh
parse: one private immutable buffer shared by `original` and initial `source`,
plus one independent mutable public `Source`. After an edit, three arrays are
semantically required: pristine `original`, current private `source`, and
mutable public `Source`. Further reduction would require changing the exported
mutable `Source` API or weakening isolation, and is out of scope.

## Production safeguards

- Use one unexported ownership-aware parser helper, not a second parser or
  whole-source decoder. A name such as `parseOwnedDocumentBytes` should make
  the assumption explicit.
- Public `Parse` owns its preclone exclusively. The private helper may alias
  that clone as both `original` and initial `source`, and must create a separate
  clone for `Document.Source`.
- `rewriteAndReparse` must remain private and documented as accepting a freshly
  allocated, exclusively owned `next`. Current audit confirms its only callers,
  `patchStartTag` and `insertAt`, construct such a buffer; do not expose the
  helper or add another caller without a new ownership review.
- On successful rewrite, transfer the existing `original` pointer without
  cloning, transfer `next` as private `source`, and keep the separate public
  `Source` clone produced by the owned parser. Assign the staged document only
  after parse success.
- On parse error, return before assigning `*d`; the current document, private
  arrays, public `Source`, dirty state, pages, nodes, edges, and issues must
  remain unchanged.
- Do not mutate the caller input, private source arrays, original bytes,
  compressed-page payload, unknown XML, or parser warnings. `OriginalBytes()`
  and `Serialize()` must continue returning defensive clones.
- Keep parser issue order/content separate from this ownership-only slice. The
  queued warning work, including root's reconciliation that harmless XML
  comments remain allowed, must not be folded into this change.

## Required external isolation tests

Behavior tests, not only pointer identity, must establish the public contract:

- Parse caller-owned bytes, then mutate every caller byte after return. Later
  model reads, `Serialize()`, `OriginalBytes()`, one edit, and reparse must be
  unchanged.
- Mutate every byte of `Document.Source`, a returned `OriginalBytes()` slice,
  and a returned `Serialize()` slice in separate cases. Later serialization and
  edits must not observe those mutations.
- Exercise the private reparse seam with invalid XML and prove the live
  document and all source arrays/state remain unchanged.
- Use a deterministic BOM/CRLF multi-page fixture containing unknown XML and a
  compressed page. No-edit serialization must be exact; an edit on another page
  must preserve the compressed payload and read-only semantics.
- After multiple successful edits, `OriginalBytes()` must equal the initial
  source, `Dirty()` must be true, and reparsing `Serialize()` must produce the
  expected model.

An in-package pointer-identity test may additionally pin the intended fresh
parse shared private backing and separate public `Source`, and the post-edit
three-array state. It is an internal ownership regression only and cannot
replace the external isolation tests.

## Benchmark semantics

The proposed benchmark method is fair only with these constraints:

- Add the benchmark source and fixture before the production ownership change,
  record the baseline, then rerun the identical benchmark and fixture after the
  change.
- Use the same deterministic moderate/large source for both checkpoints and
  record its hash, full wrapper command, exit status, and raw logs.
- Use fixed iterations (`-benchtime=20x`, `-count=3`) and `b.ReportAllocs()`;
  `b.SetBytes` may report source-byte throughput but must not be described as
  retained-memory savings.
- `B/op` is cumulative allocation per operation. It includes parser models,
  issue slices, XML tokens, compressed-page work, and output clones, so the
  expected reduction is not necessarily exactly one full source size in every
  subbenchmark. It does not measure live heap, retained RSS, whole-application
  memory, CPU, UI behavior, or physical input.
- `runtime.KeepAlive` should keep the result used during its iteration; it
  does not collect heap statistics or prove retention. Do not accumulate every
  parsed document in a growing slice and do not add runtime memory counters,
  unsafe code, or flaky RSS assertions.
- `ParseRetained` means the returned document's ownership state, not a
  retained-heap measurement. `UpdateSerialize` should keep both the edited
  document and serialized result used during the iteration.

## Affected suites and sequencing

Later implementation, only after the warning increment is accepted and root
assigns ownership, should run the focused `native/formats/drawio` tests plus
the final affected `native/modules/graphview` and
`native/cmd/levnote-drawio-svg` suites and vet through the fixed Go wrapper.
Preserve all actual outputs and failures. No result from this review authorizes
execution while `.20` packages.

This review is final design input; no additional broad planning round is needed
once the safeguards above are included.
