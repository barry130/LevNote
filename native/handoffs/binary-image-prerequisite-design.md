# Binary document API contract

2026-11-10 GLM design checkpoint for the active binary prerequisite owner. This
is the narrow first increment: only `native/core/document` changes. Session,
app, plugins, views, formats, and vendor/GPU lifecycle remain separate later
ownership. Astra owns the bitmap lifecycle review.

## Core authority

Add `core/document/binary.go` with an explicit read-only binary authority. It
does not decode text, create a synthetic text `Document`, or claim byte
inspection is an image.

```go
const MaxBinaryBytes int64 = 16 << 20

type BinaryKind string

const (
    BinaryKindUnknown BinaryKind = ""
    BinaryKindPNG     BinaryKind = "png"
    BinaryKindJPEG    BinaryKind = "jpeg"
)

type BinaryFile struct {
    Path     string
    ReadOnly bool
    // private: canonical path, kind, size, identity, baseline revision
}

func OpenBinary(path string, kind BinaryKind) (*BinaryFile, error)
func (f *BinaryFile) Kind() BinaryKind
func (f *BinaryFile) Size() int64
func (f *BinaryFile) DiskToken() DiskToken
func (f *BinaryFile) SameBaseline(token DiskToken) bool
func (f *BinaryFile) Snapshot(ctx context.Context) (*BinarySnapshot, error)
func (f *BinaryFile) Reload() error
```

`OpenBinary` canonicalizes the path, rejects non-regular files and files over
`MaxBinaryBytes`, and reuses the existing stable-read discipline: the complete
read is bounded, the file is checked for mutation during the read, and the
accepted identity contains the SHA-256 and file identity. `kind` is admission
metadata only; media syntax validation remains in the future image admission
code. `Reload` is explicit, establishes a new revision, and therefore makes old
`DiskToken`s stale. It never mutates an already accepted snapshot.

The `BinaryFile` baseline never retains compressed bytes. Dropping the caller's
snapshot drops the only authoritative byte copy owned by this API.

## Immutable snapshot

`Snapshot` verifies the current file against the accepted baseline, including
path, size, mtime/file identity, and SHA-256. A changed or missing source
returns `ErrConflict` (or the actual read error) and never substitutes newly
read bytes. A snapshot already held by the caller remains valid and copyable
after the source changes; it represents the bytes the user had accepted.

```go
type BinarySnapshot struct {
    // private: separately owned bytes and accepted DiskToken
}

func (s *BinarySnapshot) Size() int64
func (s *BinarySnapshot) Token() DiskToken
func (s *BinarySnapshot) Reader() io.Reader
func (s *BinarySnapshot) ReaderAt() io.ReaderAt
func (s *BinarySnapshot) WriteTo(w io.Writer) (int64, error)
func (s *BinarySnapshot) CopyTo(w io.Writer) (int64, error)
```

There is no public `Bytes()`. `Read` and `ReadAt` copy into caller-provided
buffers. `WriteTo`/`CopyTo` stream through a bounded scratch buffer (32 KiB),
so a custom writer never receives the private backing slice. In particular,
returning a `*bytes.Reader` is forbidden: callers can type-assert its private
`WriterTo` path and expose the backing array to a custom writer. This is an
aliasing boundary, not a security sandbox claim.

`ReaderAt` is the decoder interface: random access without duplicating the
whole source merely to hide a slice. The future decoder must treat the snapshot
as immutable and must not retain it on an inactive tab.

## Exact non-overwriting SaveAs

Saving a copy uses the accepted snapshot, not a fresh source read. The public
method verifies that the snapshot still matches the file's current baseline;
after an explicit `Reload`, an old snapshot is not adopted as the new baseline.

```go
type BinarySaveResult struct {
    Committed bool // publication returned success
    Verified  bool // committed bytes and disk identity were verified
    Adopted   bool // BinaryFile now owns the destination baseline
    NoRewrite bool // same accepted path; no bytes were written
}

func (f *BinaryFile) SaveAs(path string, snapshot *BinarySnapshot) (BinarySaveResult, error)
```

`SaveAs` canonicalizes the destination and uses the existing create-if-absent
atomic publisher. It never overwrites a target created before or during the
operation. The exact snapshot bytes are written; no decode, re-encode, BOM, or
text serializer is involved.

For the same canonical path as the current accepted baseline, it verifies the
source still matches, returns `NoRewrite: true`, and writes nothing. If that
source changed, it returns `ErrConflict` without rewriting it.

On success, a new destination is verified by digest and identity, then adopted
as the new path and baseline. The original source remains untouched. A caller
may continue copying the old snapshot through its read-only methods.

## Atomic outcome helper

Refactor the private atomic writer to return a small outcome while preserving
every existing public text API and error behavior:

```go
type atomicOutcome struct {
    Committed bool
    Digest    sha256Digest
}

func writeAtomicWriterOutcome(
    path string,
    mode os.FileMode,
    create bool,
    check func() error,
    write func(io.Writer) error,
) (atomicOutcome, error)
```

`Committed` becomes true only after `publishNew` or `replaceExisting` succeeds;
directory sync and all later verification are post-commit steps. Existing
`writeAtomic` and `writeAtomicWriter` delegate to this helper and discard the
outcome, leaving their signatures and errors unchanged. `writeUTF8Atomic`
continues to return its digest as before.

A post-commit durability or verification failure must return a non-nil error
with `Committed: true`, `Verified/Adopted: false`. It must not delete the
destination, claim the destination never existed, or establish an unverified
baseline. A pre-publication failure returns `Committed: false`. Keep test seams
as private function parameters in the narrow save path, not global hooks.

## Focused tests

Add tests in `core/document` only:

1. Stable binary open: exact bytes, size, kind, readonly mode, non-regular and
   over-limit rejection, mutation during read.
2. Snapshot reacquisition: unchanged source succeeds; same-size replacement,
   append, truncation, missing source, and mtime races are refused.
3. Alias boundary: a malicious `io.Writer` and `io.ReaderAt` consumer cannot
   obtain or mutate the backing slice; `WriteTo` remains bounded and correct.
4. SaveAs: exact-byte copy, existing target and target-created-during-save are
   never overwritten, same path is a verified no-rewrite, old snapshots remain
   readable after source change, and reload invalidates old adoption tokens.
5. Atomic outcomes: pre-publication failure leaves no destination; committed
   sync or verification failure reports committed-but-unverified and does not
   delete or adopt.
6. Existing text save APIs retain current errors and success behavior.

Run the focused document tests first, then the affected `core/document` suite
once after corrections. No package build, app/session test, vendor patch, or
GPU work belongs to this increment.
