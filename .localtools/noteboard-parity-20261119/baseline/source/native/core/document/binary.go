package document

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"sync/atomic"
)

const MaxBinaryBytes int64 = 16 << 20

type BinaryKind string

const (
	BinaryKindUnknown BinaryKind = ""
	BinaryKindPNG     BinaryKind = "png"
	BinaryKindJPEG    BinaryKind = "jpeg"
)

var (
	ErrUnsupportedBinaryKind = errors.New("unsupported binary kind")
	ErrNoBinaryBaseline      = errors.New("binary baseline is not available")
	ErrStaleBinarySnapshot   = errors.New("binary snapshot no longer matches the current baseline")
)

var lastBinaryAuthority atomic.Uint64

// BinaryFile is the mutable, same-goroutine owner of a non-editable binary
// baseline. It retains no file bytes. BinaryToken and BinarySnapshot are
// immutable and may cross goroutines after they are captured on the owner.
type BinaryFile struct {
	path      string
	kind      BinaryKind
	readOnly  bool
	disk      identity
	revision  uint64
	authority uint64
	// lastCopy rejects replay/superseded adoption without retaining receipts.
	lastCopy uint64
}

type BinaryToken struct {
	path      string
	kind      BinaryKind
	hash      [32]byte
	info      os.FileInfo
	revision  uint64
	authority uint64
	valid     bool
}

type BinarySnapshot struct {
	data  []byte
	token BinaryToken
}

type BinarySaveResult struct {
	Committed bool
	Verified  bool
	Adopted   bool
	NoRewrite bool
}

func OpenBinary(ctx context.Context, path string, kind BinaryKind) (*BinaryFile, *BinarySnapshot, error) {
	if !validBinaryKind(kind) {
		return nil, nil, &Error{"open binary", path, ErrUnsupportedBinaryKind, nil}
	}
	path, err := canonical(path)
	if err != nil {
		return nil, nil, err
	}
	data, disk, err := readStableContext(ctx, path, MaxBinaryBytes)
	if err != nil {
		return nil, nil, err
	}
	file := &BinaryFile{
		path: path, kind: kind, readOnly: disk.info.Mode().Perm()&0200 == 0,
		disk: disk, revision: 1, authority: lastBinaryAuthority.Add(1),
	}
	snapshot := &BinarySnapshot{data: data, token: file.token()}
	return file, snapshot, nil
}

func ReadBinarySnapshot(ctx context.Context, token BinaryToken) (*BinarySnapshot, error) {
	return readBinarySnapshotHook(ctx, token, binaryStableReadHook{})
}

type binaryStableReadHook struct {
	reader func(*os.File) io.Reader
	finish func(*os.File) error
}

func readBinarySnapshotHook(ctx context.Context, token BinaryToken, hook binaryStableReadHook) (*BinarySnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !token.Valid() {
		return nil, ErrNoBinaryBaseline
	}
	data, disk, err := readStableContextHook(ctx, token.path, MaxBinaryBytes, hook)
	if err != nil {
		return nil, err
	}
	if !sameIdentity(token.identity(), disk) {
		return nil, &Error{"open binary", token.path, ErrConflict, nil}
	}
	return &BinarySnapshot{data: data, token: token}, ctx.Err()
}

func (f *BinaryFile) Path() string         { return f.path }
func (f *BinaryFile) Kind() BinaryKind     { return f.kind }
func (f *BinaryFile) SourceReadOnly() bool { return f.readOnly }
func (f *BinaryFile) Size() int64 {
	if f == nil || f.disk.info == nil {
		return 0
	}
	return f.disk.info.Size()
}
func (f *BinaryFile) Revision() uint64 { return f.revision }

func (f *BinaryFile) Token() BinaryToken {
	if f == nil || f.disk.info == nil {
		return BinaryToken{}
	}
	return BinaryToken{
		path: f.path, kind: f.kind, hash: f.disk.hash, info: f.disk.info,
		revision: f.revision, authority: f.authority, valid: true,
	}
}

func (f *BinaryFile) SameBaseline(token BinaryToken) bool {
	return f != nil && token.Valid() && f.token().same(token)
}

func (f *BinaryFile) Snapshot(ctx context.Context) (*BinarySnapshot, error) {
	if f == nil {
		return nil, ErrNoBinaryBaseline
	}
	return ReadBinarySnapshot(ctx, f.token())
}

func (f *BinaryFile) Reload(ctx context.Context) (*BinarySnapshot, error) {
	if f == nil {
		return nil, ErrNoBinaryBaseline
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, disk, err := readStableContext(ctx, f.path, MaxBinaryBytes)
	if err != nil {
		return nil, err
	}
	f.disk = disk
	f.readOnly = disk.info.Mode().Perm()&0200 == 0
	f.revision++
	return &BinarySnapshot{data: data, token: f.token()}, nil
}

// SaveAs synchronously publishes and adopts a copy on the owner's goroutine.
// For asynchronous work, PublishBinaryCopy uses only an immutable snapshot;
// return its receipt to AdoptCopy, which also returns the rebound snapshot.
func (f *BinaryFile) SaveAs(ctx context.Context, path string, snapshot *BinarySnapshot) (BinarySaveResult, error) {
	return f.saveAs(ctx, path, snapshot, binarySaveHooks{})
}

type binarySaveHooks struct {
	beforeWrite   func() error
	beforePublish func() error
	syncDir       func(string) error
	verify        func(context.Context, string) (identity, error)
	digest        func(context.Context, string, int64) (identity, error)
	wrapWriter    func(io.Writer) io.Writer
}

func (f *BinaryFile) saveAs(ctx context.Context, path string, snapshot *BinarySnapshot, hooks binarySaveHooks) (BinarySaveResult, error) {
	if f == nil {
		return BinarySaveResult{}, ErrNoBinaryBaseline
	}
	if err := ctx.Err(); err != nil {
		return BinarySaveResult{}, err
	}
	if snapshot == nil {
		return BinarySaveResult{}, &Error{"save as binary", f.path, ErrNoBinaryBaseline, nil}
	}
	if !f.ownsSnapshot(snapshot) {
		return BinarySaveResult{}, &Error{"save as binary", f.path, ErrStaleBinarySnapshot, nil}
	}
	result, receipt, err := publishBinaryCopy(ctx, path, snapshot, hooks)
	if err != nil {
		return result, err
	}
	result, _, err = f.AdoptCopy(receipt, snapshot)
	return result, err
}

func (f *BinaryFile) ownsSnapshot(snapshot *BinarySnapshot) bool {
	return snapshot != nil && f.SameBaseline(snapshot.token)
}

func validBinaryKind(kind BinaryKind) bool {
	switch kind {
	case BinaryKindUnknown, BinaryKindPNG, BinaryKindJPEG:
		return true
	default:
		return false
	}
}

func (f *BinaryFile) token() BinaryToken {
	return f.Token()
}

func (t BinaryToken) identity() identity {
	return identity{hash: t.hash, info: t.info}
}

func (t BinaryToken) diskToken() DiskToken {
	return DiskToken{
		path: t.path, hash: t.hash, info: t.info,
		revision: t.revision, valid: t.valid,
	}
}

func (t BinaryToken) Valid() bool      { return t.valid }
func (t BinaryToken) Path() string     { return t.path }
func (t BinaryToken) Kind() BinaryKind { return t.kind }
func (t BinaryToken) Size() int64 {
	if !t.valid || t.info == nil {
		return 0
	}
	return t.info.Size()
}
func (t BinaryToken) CheckDisk(ctx context.Context) (DiskCheckResult, error) {
	return t.diskToken().CheckDisk(ctx)
}
func (t BinaryToken) same(other BinaryToken) bool {
	return t.valid && other.valid && t.path == other.path && t.kind == other.kind &&
		t.hash == other.hash && sameIdentity(t.identity(), other.identity()) &&
		t.revision == other.revision && t.authority == other.authority
}

func (s *BinarySnapshot) Size() int64 {
	if s == nil {
		return 0
	}
	return int64(len(s.data))
}
func (s *BinarySnapshot) Token() BinaryToken { return s.token }

func (s *BinarySnapshot) Reader() io.Reader {
	if s == nil {
		return nil
	}
	return &binarySnapshotReader{snapshot: s}
}

func (s *BinarySnapshot) ReaderAt() io.ReaderAt {
	if s == nil {
		return nil
	}
	return &binarySnapshotReaderAt{snapshot: s}
}

func (s *BinarySnapshot) WriteTo(w io.Writer) (int64, error) {
	if s == nil {
		return 0, nil
	}
	scratch := make([]byte, 32<<10)
	var total int64
	for total < int64(len(s.data)) {
		chunk := scratch[:copy(scratch, s.data[total:])]
		written, err := w.Write(chunk)
		if written < 0 || int64(written) > int64(len(chunk)) {
			return total, fmt.Errorf("writer reported %d bytes from %d", written, len(chunk))
		}
		total += int64(written)
		if err != nil {
			return total, err
		}
		if written < len(chunk) {
			return total, io.ErrShortWrite
		}
	}
	return total, nil
}

func writeBinarySnapshot(ctx context.Context, w io.Writer, snapshot *BinarySnapshot) error {
	if snapshot == nil {
		return ErrNoBinaryBaseline
	}
	scratch := make([]byte, 32<<10)
	for total := 0; total < len(snapshot.data); {
		if err := ctx.Err(); err != nil {
			return err
		}
		chunk := scratch[:copy(scratch, snapshot.data[total:])]
		written, err := w.Write(chunk)
		if written < 0 || written > len(chunk) {
			return fmt.Errorf("writer reported %d bytes from %d", written, len(chunk))
		}
		total += written
		if err != nil {
			return err
		}
		if written < len(chunk) {
			return io.ErrShortWrite
		}
	}
	return ctx.Err()
}

type binarySnapshotReader struct {
	snapshot *BinarySnapshot
	offset   int64
}

func (r *binarySnapshotReader) Read(p []byte) (int, error) {
	if r.snapshot == nil || r.offset < 0 || r.offset >= int64(len(r.snapshot.data)) {
		return 0, io.EOF
	}
	n := copy(p, r.snapshot.data[r.offset:])
	r.offset += int64(n)
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

type binarySnapshotReaderAt struct {
	snapshot *BinarySnapshot
}

func (r *binarySnapshotReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.snapshot == nil || off < 0 || off >= int64(len(r.snapshot.data)) {
		return 0, io.EOF
	}
	n := copy(p, r.snapshot.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func readStableContext(ctx context.Context, path string, limit int64) ([]byte, identity, error) {
	return readStableContextHook(ctx, path, limit, binaryStableReadHook{})
}

func readStableContextHook(ctx context.Context, path string, limit int64, hook binaryStableReadHook) ([]byte, identity, error) {
	file, before, err := openStable(ctx, path, limit)
	if err != nil {
		return nil, identity{}, err
	}
	defer file.Close()
	reader := io.Reader(file)
	if hook.reader != nil {
		reader = hook.reader(file)
	}
	data := make([]byte, 0, before.Size())
	buffer := make([]byte, 64<<10)
	for {
		if err := ctx.Err(); err != nil {
			return nil, identity{}, err
		}
		read, readErr := reader.Read(buffer)
		if read > 0 {
			if int64(len(data))+int64(read) > before.Size() {
				return nil, identity{}, &Error{"open binary", path, ErrConflict, nil}
			}
			if int64(len(data))+int64(read) > limit {
				return nil, identity{}, &Error{"open binary", path, ErrTooLarge, nil}
			}
			data = append(data, buffer[:read]...)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, identity{}, readErr
		}
	}
	if hook.finish != nil {
		if err := hook.finish(file); err != nil {
			return nil, identity{}, err
		}
	}
	after, err := finishStable(file, path, before)
	if err != nil {
		return nil, identity{}, err
	}
	if err := ctx.Err(); err != nil {
		return nil, identity{}, err
	}
	if int64(len(data)) != after.Size() {
		return nil, identity{}, &Error{"open binary", path, ErrConflict, nil}
	}
	return data, identity{hash: sha256.Sum256(data), info: after}, nil
}

func readStableDigestContext(ctx context.Context, path string, limit int64) (identity, error) {
	file, before, err := openStable(ctx, path, limit)
	if err != nil {
		return identity{}, err
	}
	defer file.Close()
	hash := sha256.New()
	buffer := make([]byte, 64<<10)
	size := int64(0)
	for {
		if err := ctx.Err(); err != nil {
			return identity{}, err
		}
		read, readErr := file.Read(buffer)
		if read > 0 {
			size += int64(read)
			if size > limit {
				return identity{}, &Error{"verify binary save", path, ErrTooLarge, nil}
			}
			if _, err = hash.Write(buffer[:read]); err != nil {
				return identity{}, err
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return identity{}, readErr
		}
	}
	after, err := finishStable(file, path, before)
	if err != nil {
		return identity{}, err
	}
	if err := ctx.Err(); err != nil {
		return identity{}, err
	}
	if size != after.Size() {
		return identity{}, &Error{"verify binary save", path, ErrConflict, nil}
	}
	var digest [32]byte
	copy(digest[:], hash.Sum(nil))
	return identity{hash: digest, info: after}, nil
}

func openStable(ctx context.Context, path string, limit int64) (*os.File, os.FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	before, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	if !before.Mode().IsRegular() {
		file.Close()
		return nil, nil, &Error{"open binary", path, ErrNotRegular, nil}
	}
	if before.Size() > limit {
		file.Close()
		return nil, nil, &Error{"open binary", path, ErrTooLarge, nil}
	}
	return file, before, nil
}

func finishStable(file *os.File, path string, before os.FileInfo) (os.FileInfo, error) {
	after, err := file.Stat()
	if err != nil {
		return nil, err
	}
	current, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) ||
		after.Size() != current.Size() || !after.ModTime().Equal(current.ModTime()) ||
		!os.SameFile(after, current) {
		return nil, &Error{"open binary", path, ErrConflict, nil}
	}
	return after, nil
}
