# Draw.io source-retention independent review

Status: completed read-only review, 2026-11-10. This review covers the frozen
source, tests, benchmark evidence, final affected test/vet logs, and the restored
author result. No product source was changed and no test, benchmark, build, or
other execution was performed.

## Verdict

**Source review passes.** The ownership change satisfies the planned constraints
and creates no ownership, rollback, mutation-isolation, serialization, warning,
limit, or BOM blocker.

The restored author result is technically reconciled with the frozen evidence.
Its remaining issues are four copyedit typos assigned to the author; they do not
change any technical claim or source/test verdict.

## Source findings

- Public `Parse` normalizes `ParseOptions` and checks `MaxBytes` before cloning
  caller input. The oversized-input test witnesses rejection before the clone.
- The private owned path takes exclusive ownership, aliases the immutable owned
  buffer as pristine `original` and initial private `source`, and separately
  clones the exported mutable `Document.Source`.
- `rewriteAndReparse` parses into a staged document, preserves the old
  `original`, transfers the freshly allocated private `source`, and assigns
  `*d` only after successful parse. Failed reparse returns before assignment.
- `Serialize()` and `OriginalBytes()` still return defensive clones.
- The only `rewriteAndReparse` callers, `patchStartTag` and `insertAt`,
  construct fresh owned buffers before invoking it.
- Tests mutate every externally reachable byte slice—caller input, exported
  `Source`, `OriginalBytes()`, and `Serialize()`—and then perform a real later
  edit/reparse. Failed reparse rollback is compared against a deep full-document
  snapshot and followed by a successful edit.
- Multiple edits preserve the pristine original bytes, unknown XML, CRLF source,
  offsets, compressed read-only payload and model, dirty state, and warning
  behavior.
- No public API, decoder, source-copy shortcut, serialization behavior, warning
  logic, byte-cap behavior, or raw BOM parser support changed. Document-layer
  BOM handling remains outside and unchanged by this claim.

## Verified identities

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

The deterministic benchmark fixture is 33,985 bytes, SHA-256
`04E533B0DA533713F51E106F820A7F4A62D0E0F06CA910EB6FEA9857A3D3E9F`, with 8
pages, 7 uncompressed pages, 169 nodes, and the final page compressed and
read-only. Fixture and benchmark identities were unchanged pre/post.

## Benchmark and gates

The pre/post benchmark used identical source and fixture with
`-benchtime=20x -benchmem -count=3`. Median cumulative allocations moved:

- `ParseRetained`: 764,244 to 722,520 B/op; 8,335 to 8,333 allocs/op.
- `UpdateSerialize`: 1,733,705 to 1,487,174 B/op; 16,688 to 16,681 allocs/op.

Timing samples varied and is not a speed claim. `B/op` is cumulative allocation
per operation, not retained heap, working set, RSS, or whole-process memory.

The final affected `test` command exited 0 for drawio (0.587s), graphview
(9.423s), and the drawio SVG worker (0.661s). Final `vet` exited 0 with empty
stdout/stderr. The disclosed focused-test and harness corrections are test or
invocation corrections, not product failures.

## Closure

No source or test defect remains. The `.20` package and accepted warning slice
remain immutable. The four author-result copyedits are non-technical, do not
require source changes or reruns, and do not block independent source review.

## Handoff-repair addendum

The later 3,953-byte author result was corrupted beyond the four copyedits
and removed unsupported SVG/warning numbers. It was repaired from the already
verified frozen facts only. The corrected result restores the five changed-file
identities, benchmark table and limits, final test/vet facts, disclosure
language, and raw-evidence links; it removes the unsupported SVG paragraph and
states that `types.go` is unchanged. This was document repair, not new
independent evidence, and does not alter the source-review verdict.
