# Binary core implementation result

2026-11-10 GLM completed the authorized first increment in `core/document`.
No app, session, plugin, view, product-doc, or vendor files were changed.

## API

```go
const MaxBinaryBytes int64 = 16 << 20

type BinaryKind string
const (
    BinaryKindUnknown BinaryKind = ""
    BinaryKindPNG     BinaryKind = "png"
    BinaryKindJPEG    BinaryKind = "jpeg"
)

func OpenBinary(ctx context.Context, path string, kind BinaryKind) (*BinaryFile, *BinarySnapshot, error)
func ReadBinarySnapshot(ctx context.Context, token BinaryToken) (*BinarySnapshot, error)

type BinaryFile struct{ /* private mutable same-goroutine state */ }
func (f *BinaryFile) Path() string
func (f *BinaryFile) Kind() BinaryKind
func (f *BinaryFile) SourceReadOnly() bool
func (f *BinaryFile) Size() int64
func (f *BinaryFile) Revision() uint64
func (f *BinaryFile) Token() BinaryToken
func (f *BinaryFile) SameBaseline(token BinaryToken) bool
func (f *BinaryFile) Snapshot(ctx context.Context) (*BinarySnapshot, error)
func (f *BinaryFile) Reload(ctx context.Context) (*BinarySnapshot, error)
func (f *BinaryFile) SaveAs(ctx context.Context, path string, snapshot *BinarySnapshot) (BinarySaveResult, error)

type BinaryToken struct{ /* private immutable identity */ }
func (t BinaryToken) Valid() bool
func (t BinaryToken) Path() string
func (t BinaryToken) Kind() BinaryKind
func (t BinaryToken) Size() int64
func (t BinaryToken) CheckDisk(ctx context.Context) (DiskCheckResult, error)

type BinarySnapshot struct{ /* private bytes + immutable token */ }
func (s *BinarySnapshot) Size() int64
func (s *BinarySnapshot) Token() BinaryToken
func (s *BinarySnapshot) Reader() io.Reader
func (s *BinarySnapshot) ReaderAt() io.ReaderAt
func (s *BinarySnapshot) WriteTo(w io.Writer) (int64, error)

type BinarySaveResult struct {
    Committed bool
    Verified  bool
    Adopted   bool
    NoRewrite bool
}
```

`OpenBinary` and `Reload` return the just-accepted snapshot, avoiding a second
full read. `BinaryFile` retains no bytes or OS handle and is documented as the
same-goroutine mutable owner; its token and snapshot are immutable and safe to
capture for worker use. `ReadBinarySnapshot` verifies an immutable token without
copying a live `BinaryFile`.

All stable reads are bounded by `MaxBinaryBytes`, honor context at the loop and
final boundary, verify metadata before/after, and require the actual byte count
to equal the accepted file size. No known raw-byte mismatch establishes a
baseline. Same-size/restored-mtime replacement is rejected by SHA-256.

Snapshot read methods copy into caller storage. `WriteTo` streams through a
32 KiB scratch buffer, so a custom writer never receives the private backing
slice. SaveAs uses a separate context-aware stream and checks context before
atomic publication. Cancellation before publication leaves no destination;
cancellation or durability failure after publication reports `Committed:true`
while `Verified` and `Adopted` remain false and leaves the committed file intact.

SaveAs accepts only a snapshot belonging to the current path, identity, hash,
revision, and authority. It copies the accepted snapshot even when the original
later changes or disappears. An explicit Reload or successful adoption advances
the revision; the old snapshot remains readable but cannot be adopted by the
current authority. A same-path request re-verifies the source and writes no
bytes. A new destination uses the existing non-overwriting create-if-absent
publisher and is verified by digest before adoption.

The private atomic outcome records `Committed` at the publication boundary.
Existing text wrappers preserve their signatures and errors, and directory sync
is post-commit.

## Files

- `native/core/document/binary.go`: binary authority, immutable snapshot/token,
  context-aware stable reads, and exact non-overwriting SaveAs.
- `native/core/document/binary_test.go`: focused regressions for exact bytes,
  alias safety, source changes, reload/adoption generations, cancellation,
  read bounds, same-path no-rewrite, exclusive races, and post-commit state.
- `native/core/document/atomic.go`: private `atomicOutcome` extraction; no
  public text API changes.
- `native/handoffs/binary-image-prerequisite-design.md`: initial design record.

## Final review corrections

The final focused review corrections are included:

- Mid-copy cancellation now wraps the real atomic writer and cancels after the
  first 32 KiB write; it verifies a nonzero write, no destination, no temp leak,
  and unchanged baseline.
- A low-level per-call stable-read seam injects a short EOF, actual read error,
  and final-boundary cancellation into the real helper. No helper emulation or
  global hook is used.
- Same-size/restored-mtime replacement uses a different five-byte value. A
  separate replace-by-rename fixture checks identity independently.
- Empty-file copy, copy-after-source-deletion, and cross-authority rejection are
  tested. A separately opened authority rejects a foreign snapshot before
  creating a destination, while that second authority accepts its own token.
- Same-path SaveAs now uses the bounded streaming digest helper and does not
  retain a second source-size byte buffer.
- Stable reads reject growth beyond the accepted initial stat size immediately,
  avoiding a redundant expanded backing array.
- The 10:40 review corrections use one context for post-commit cancellation,
  retain the real digest verifier, record exactly one successful 32 KiB write,
  preserve the old baseline token, and prove same-size/same-timestamp plus
  identity-isolation before the replacement checks.
- The Windows rename fixture was corrected after a false-positive review finding.
  It now uses the eager baseline handle identity from `file.disk.info`, obtains
  the replacement identity from an open file handle, and proves identical
  bytes/size/timestamp with different identities. `CheckDisk` and `Snapshot`
  reject the replacement.

## Verification

All commands used `native/scripts/go.ps1` from the repository root. Final
streams and exit codes are frozen in
`native/handoffs/binary-core-final-evidence-20261110/`.

- `go fmt ./core/document` — exit 0.
- `go test -run ^TestBinary ./core/document -count=1 -v` — 17/17 PASS,
  exit 0 (`focused-binary-2.txt`, `.exit`), followed by the corrected rename-only
  fixture pass (`rename-identity-3.txt`, `.exit`).
- `go test ./core/document -count=1` — pass, exit 0
  (`document-suite-3.txt`, `.exit`).
- `go vet ./core/document` — no output, exit 0
  (`document-vet-3.txt`, `.exit`).

During correction, real focused runs exposed the wrapper argument order, a
stale-snapshot expectation, unused imports/variables, and a committed-then-
verification-cancel result. Each was fixed and the final focused, affected
package, and vet runs passed. No broad build, app/session suite, launch, or
resource run was performed.

## Frozen source identities

- `core/document/binary.go`: SHA256
  `E3D56794354B3CE2B1A9CBF93BD68A67E3F333A482CCA3EC5D8753DE18FF759D`,
  14,495 bytes.
- `core/document/binary_test.go`: SHA256
  `60637AD62D744E9331F584B6E60F95DBA268C2E2487DAB8D7F2E1B07D0CCFB19`,
  22,341 bytes.
- `core/document/atomic.go`: SHA256
  `33D45A65253972D8547A5DC7ACD1AC9FA04C0540EF11DD3DE207676801A232CC`,
  3,516 bytes.
