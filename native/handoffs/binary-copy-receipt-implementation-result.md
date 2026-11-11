# Worker-safe binary copy receipts: frozen result

2026-11-10. Implementation complete; focused tests and one final affected suite
and vet passed. Scope is binary.go, new binary_copy.go and binary_test.go only.
Atomic helper, app/session/raster/vendor, accepted packages and resource evidence
were not changed. No app/package/host/worker/driver/resource runs.

## Public API and ownership

```go
func PublishBinaryCopy(ctx context.Context, path string, snapshot *BinarySnapshot) (BinarySaveResult, BinaryCopyReceipt, error)
func (f *BinaryFile) AdoptCopy(receipt BinaryCopyReceipt, snapshot *BinarySnapshot) (BinarySaveResult, *BinarySnapshot, error)
func (r BinaryCopyReceipt) Path() string
func (r BinaryCopyReceipt) Result() BinarySaveResult
```

- Worker publication receives only context, destination and immutable snapshot.
  It never captures or mutates BinaryFile. It uses the existing exclusive atomic
  writer, 32KiB output scratch and 64KiB streaming verification scratch. It does
  not reopen the source or allocate/copy another source-sized byte buffer.
- A receipt has private source token, destination, eagerly captured verified
  destination identity, result flags and serial. It retains no BinaryFile,
  snapshot, byte slice, reader or open handle. Result().Adopted remains false.
- Owner-thread AdoptCopy checks verified receipt validity, current SameBaseline,
  supplied snapshot token, and replay status before mutation. New-destination
  adoption updates path/disk/readonly, increments revision and returns a fresh
  descriptor sharing accepted bytes with its new token. The old descriptor and
  token remain unchanged and stale. The rebound descriptor can be saved again.
- Same-path publication verifies without writing. Its successful adoption
  returns the supplied snapshot, NoRewrite=true, Adopted=false and no revision
  change. A per-owner serial watermark rejects consumed or superseded same-path
  receipts with ErrReplayedBinaryCopyReceipt, without retaining receipt history.
  New-destination replay is already stale after the revision change and returns
  ErrStaleBinarySnapshot. Failed adoption never consumes the valid receipt.
- Zero/unverified receipts return ErrInvalidBinaryCopyReceipt; wrong owner,
  wrong snapshot, reload or another destination adoption reject as stale.
  Publication flags survive rejection, and committed destinations stay intact.
- Existing synchronous SaveAs retains its signature and outcome semantics,
  delegating to the same publication/adoption implementation. Its initial owner
  check still rejects stale snapshots before publication.

## Integration contract and limits

Capture the immutable snapshot on the owner, send only it plus path/context to
the worker, then return receipt and original snapshot to the owner for adoption.
On success replace the active snapshot with the returned rebound descriptor.
If the originating tab closes/changes, report the actual copy result instead of
discarding or rolling back the destination. Committed=true with Verified=false
means publication happened but cancellation/sync/verification prevented adoption.

A receipt describes the destination at verification time. External writers can
change/delete it afterward. Adoption performs no filesystem IO and does not
claim filesystem CAS; retain the verified token and use CheckDisk to observe
later conflict. Held accepted bytes remain available even after source deletion.
Cancellation is checked during writing, before publication and during digest
verification; it cannot revoke a publication that has already completed.

## Preserved validation

All Go commands used native/scripts/go.ps1. Exact command scripts, stdout/stderr,
exit codes and timestamps are in binary-copy-receipt-evidence-20261110.

| Check | Result |
| --- | --- |
| wrapper `run cmd/gofmt -w` on the three scoped files | exit0 |
| `test ./core/document -run ^TestBinaryCopy -count=1 -v` | exit0, package0.553s;9 top-level tests and12 subtests pass |
| final `test ./core/document -count=1` | exit0, package0.973s |
| final `vet ./core/document` | exit0, no diagnostics |

New cases cover worker publication with owner unchanged, rebound storage/token
and second SaveAs, zero/foreign/mismatched/replayed/stale receipts, a real channel
barrier allowing owner reload during publication, same-path no rewrite, actual
pre-publication and post-commit cancellation, directory-sync failure, actual
destination mutation followed by production digest mismatch, source deletion,
exclusive publication race, and external change/deletion after verification.
Existing accepted binary/text tests remain in the final suite. No race-detector,
physical UI or process resource claim is made.

No Go check failed or was repeated. The initial before-snapshot PowerShell
command had a parse error before execution; the corrected metadata command
saved the before copies and identities. Implementation checkpoint is preserved.

## Frozen identities

Machine-readable list: binary-copy-receipt-frozen-files.json. Original files and
identities are preserved under the evidence root for scoped comparison.

| File | Bytes | SHA256 |
| --- | ---: | --- |
| core/document/binary.go |12921|b60483712068daa13e0293dd70873b5e132df183e59c05599680efa18ecc427e|
| core/document/binary_copy.go |7389|51117335de9a7c5d95b85f2c1e603b05d3414f79e4a22987ebeef63c6ab85fa2|
| core/document/binary_test.go |38628|4a462ccd2c0f0611c9928347fee9de928d3e2d91a5db16f1ce6c4cdf5e6ebcb9|
| core/document/atomic.go (unchanged) |3516|33d45a65253972d8547a5dc7acd1ac9fa04c0540ef11dd3de207676801a232cc|

Frozen for root review. App integration awaits its separate activation.
