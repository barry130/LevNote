# UTF-8 streaming save: independent review

2026-11-09 interim review. This is a read-only source review of the in-progress
GLM implementation. The owner has not frozen source or published its result,
tests, benchmark or evidence, so this document does not constitute final
reconciliation or acceptance.

## Material findings

No material blocker was found in the current digest-correction draft.

The previously reported digest-verification deletion is no longer present in
the reviewed snapshot:

- `native/core/document/atomic.go:78-98` streams the optional UTF-8 BOM and
  `Snapshot.WriteTo` through a 32 KiB `bufio.Writer` into the temp file and
  SHA-256 sink. It hashes exactly the bytes accepted at that boundary, returns
  only after a successful flush, and propagates BOM, snapshot and flush errors.
- `native/core/document/atomic.go:35-73` keeps creation, write plus flush,
  `Sync`, close, the pre-publication conflict check, exclusive/atomic
  publication and temp-file removal in one shared lifecycle. A failure before
  publication leaves the existing target untouched; `syncDirectory` remains
  the post-publication error boundary.
- `native/core/document/document.go:229-243` passes the streaming helper's
  digest to `didSave`, while the legacy format path passes
  `sha256.Sum256(encodedBytes)` to the same verifier.
- `native/core/document/document.go:288-311` reads the committed file back,
  compares its SHA-256 with the exact intended digest, and only then advances
  `disk`, `saved`, encoding/BOM, baseline and dirty state. A digest mismatch
  returns `ErrConflict` before those saved-state updates.

## Outstanding freeze gates

The reviewed tree contains no new UTF-8 save test source, benchmark or owner
result/raw evidence yet. Final reconciliation therefore remains open. At the
owner's freeze, the evidence must substantively cover exact BOM/UTF-8 bytes,
LF/CRLF, Unicode and buffer boundaries, short/sink failure and temp cleanup,
wrong-expected-digest refusal without baseline advancement, legacy-encoding
behavior, conflict/exclusive publication, and the bounded equivalent-work
benchmark required by the task.

## Interim source identities

- `native/core/document/atomic.go`: 2,913 bytes, SHA-256
  `69001B1150521748D6C923C716A4AF09DE364658225473D4A4FDD77E4E8D265A`
- `native/core/document/document.go`: 12,603 bytes, SHA-256
  `2CDEA7D8088E21C6F80B8232CFBE69A2786DBEBAED8678A56B995CC6EB2AF9D1`

These identities are observations of the implementing draft, not frozen input
identities. No tests, builds, launches, benchmarks, product/source edits or
evidence changes were performed by this review.
