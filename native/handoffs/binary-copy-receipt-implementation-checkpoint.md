# Binary copy receipt implementation checkpoint

2026-11-10. Source implementation and focused regression cases are on disk;
formatting/tests have not run yet. No app/session/raster/vendor changes or runs.

- `PublishBinaryCopy(ctx, path, snapshot)` accepts immutable data only. It uses
  the existing exclusive atomic writer and bounded streaming digest, returning
  truthful publication flags and an opaque receipt without retaining bytes.
- `BinaryFile.AdoptCopy(receipt, snapshot)` checks current authority, supplied
  snapshot and receipt status/replay serial before mutation. New destinations
  return a descriptor sharing accepted data with the new owner token. Same-path
  verification does not rewrite or increment revision. Existing SaveAs delegates.
- Tests appended to binary_test.go cover worker publication, concurrent owner
  reload, rebound/second SaveAs, stale/foreign/zero/replayed receipts, same-path
  verification, cancellation and real post-publication verification failures,
  exclusive races, held bytes after source deletion, and later disk changes.
- Only binary.go, new binary_copy.go and binary_test.go are implementation scope.
  Original three-file identities and before copies are saved in
  binary-copy-receipt-evidence-20261110. Atomic helper remains unchanged.
- Next: wrapper-only formatting, focused new tests, then one final affected
  core/document suite/vet after any actual corrections. Save all results and
  freeze APIs/source identities in the implementation-result handoff.
