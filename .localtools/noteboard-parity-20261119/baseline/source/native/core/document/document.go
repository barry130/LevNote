// Package document owns file safety and document sessions. Format plugins edit
// a Buffer; they never receive permission to bypass this save/conflict boundary.
package document

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"levnote.local/native/core/text"
)

var (
	ErrConflict                = errors.New("file changed outside LevNote; reload or save a copy")
	ErrReadOnly                = errors.New("file is read-only")
	ErrNoPath                  = errors.New("choose a file name before saving")
	ErrExists                  = errors.New("destination already exists; choose a different file name")
	ErrDirty                   = errors.New("document has unsaved changes")
	ErrBinary                  = errors.New("binary content cannot be opened as editable text")
	ErrUnsupportedEncoding     = errors.New("encoding is unknown or invalid; original bytes are preserved")
	ErrUnrepresentableEncoding = errors.New("text contains characters that the selected encoding cannot represent")
	ErrTooLarge                = errors.New("file exceeds the editable text size limit")
	ErrNotRegular              = errors.New("path is not a regular file")
	ErrComposition             = errors.New("finish the current text composition before saving")
)

type Encoding string

const (
	UTF8    Encoding = "UTF-8"
	UTF16LE Encoding = "UTF-16LE"
	UTF16BE Encoding = "UTF-16BE"
	GBK     Encoding = "GBK"
	GB18030 Encoding = "GB18030"
	// Stable disk-read bound; decoded UTF-8 seeds may be larger and are preserved.
	MaxEditableBytes int64 = text.DefaultMaxBytes
)

// Error keeps the actual path and OS error intact. The UI translates the Kind,
// but must not translate or replace the diagnostic detail supplied by the OS.
type Error struct {
	Op, Path string
	Kind     error
	Detail   error
}

func (e *Error) Error() string {
	if e.Detail != nil {
		return fmt.Sprintf("%s %s: %v: %v", e.Op, e.Path, e.Kind, e.Detail)
	}
	return fmt.Sprintf("%s %s: %v", e.Op, e.Path, e.Kind)
}
func (e *Error) Unwrap() error { return e.Kind }

// PublicationError means atomic publication completed, but durability or
// read-back verification failed. The target cannot safely be rolled back and
// no clean saved baseline was established. MetadataAdopted identifies SaveAs
// failures after Path/IsNew/ReadOnly changed; Cause remains available to Is/As.
type PublicationError struct {
	Path            string
	MetadataAdopted bool
	Cause           error
}

func (e *PublicationError) Error() string {
	return fmt.Sprintf("save incomplete; bytes may already be on target %s (metadata adopted: %t): %v", e.Path, e.MetadataAdopted, e.Cause)
}
func (e *PublicationError) Unwrap() error { return e.Cause }

type identity struct {
	hash [32]byte
	info os.FileInfo
}
type Document struct {
	Path             string
	Buffer           *text.Buffer
	Encoding         Encoding
	BOM              bool
	ReadOnly         bool
	IsNew            bool
	saved            text.Snapshot
	savedEncoding    Encoding
	savedBOM         bool
	disk             identity
	baselineRevision uint64
	dirtyVersion     uint64
	dirtyValue       bool
	dirtyCached      bool
	unverifiedSave   bool
}

func New(value string) *Document {
	b := text.New(value)
	return &Document{Buffer: b, Encoding: UTF8, IsNew: true, saved: b.Snapshot(), savedEncoding: UTF8}
}

// WriteNewFile publishes an export or new file only if no target exists.
// Existing files, including one created after a caller's preflight, stay intact.
func WriteNewFile(path string, data []byte) error {
	path, err := canonical(path)
	if err != nil {
		return err
	}
	if _, err = os.Lstat(path); err == nil {
		return &Error{"write new", path, ErrExists, nil}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return writeAtomic(path, data, 0600, true, nil)
}

func canonical(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	// Resolve existing aliases: saving through a symbolic link must update its
	// target, not replace the link itself with an unrelated regular file.
	if actual, err := filepath.EvalSymlinks(abs); err == nil {
		return actual, nil
	}
	return filepath.Clean(abs), nil
}

func Open(path string) (*Document, error) {
	return open(path, "")
}

// OpenWithEncoding is an explicit user choice, never an automatic guess. A
// failed or non-round-trippable decode leaves the file available to Inspect.
func OpenWithEncoding(path string, encoding Encoding) (*Document, error) {
	return open(path, encoding)
}

func open(path string, forced Encoding) (*Document, error) {
	path, err := canonical(path)
	if err != nil {
		return nil, err
	}
	bytes, id, err := readStable(path, MaxEditableBytes)
	if err != nil {
		return nil, err
	}
	value, enc, bom, err := decodeWith(bytes, forced)
	if err != nil {
		return nil, &Error{"open", path, err, nil}
	}
	b := text.New(value)
	return &Document{Path: path, Buffer: b, Encoding: enc, BOM: bom, ReadOnly: id.info.Mode().Perm()&0200 == 0,
		saved: b.Snapshot(), savedEncoding: enc, savedBOM: bom, disk: id}, nil
}

func (d *Document) Dirty() bool {
	if d.IsNew || d.unverifiedSave || d.Encoding != d.savedEncoding || d.BOM != d.savedBOM {
		return true
	}
	if !d.dirtyCached || d.dirtyVersion != d.Buffer.Version() {
		d.dirtyValue = !d.Buffer.EqualSnapshot(d.saved)
		d.dirtyVersion = d.Buffer.Version()
		d.dirtyCached = true
	}
	return d.dirtyValue
}

func (d *Document) CheckExternalChange() (bool, error) {
	if d.IsNew || d.Path == "" {
		return false, nil
	}
	if d.disk.info == nil {
		return true, nil
	}
	id, err := fingerprint(d.Path, d.disk.info.Size(), d.disk.info)
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, ErrConflict) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return !sameIdentity(d.disk, id), nil
}

// OpenLimitedRead decodes with the normal stable-read semantics but enforces
// the supplied read limit during the actual read, not merely after Stat. A
// file growing between stat and open therefore cannot allocate past limit.
func OpenLimitedRead(path string, encoding Encoding, limit int64) (*Document, error) {
	path, err := canonical(path)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > MaxEditableBytes {
		return nil, ErrTooLarge
	}
	bytes, id, err := readStable(path, limit)
	if err != nil {
		if errors.Is(err, ErrTooLarge) {
			return nil, ErrTooLarge
		}
		return nil, err
	}
	value, enc, bom, err := decodeWith(bytes, encoding)
	if err != nil {
		return nil, &Error{"open", path, err, nil}
	}
	b := text.New(value)
	return &Document{Path: path, Buffer: b, Encoding: enc, BOM: bom,
		ReadOnly: id.info.Mode().Perm()&0200 == 0,
		saved:    b.Snapshot(), savedEncoding: enc, savedBOM: bom, disk: id}, nil
}

func (d *Document) Save() error {
	if d.Path == "" || d.IsNew {
		return ErrNoPath
	}
	if d.Buffer.InGroup() {
		return ErrComposition
	}
	if d.ReadOnly {
		return &Error{"save", d.Path, ErrReadOnly, nil}
	}
	path, buffer, version := d.Path, d.Buffer, d.Buffer.Version()
	encoding, bom, disk, revision := d.Encoding, d.BOM, d.disk, d.baselineRevision
	check := func() error {
		if d.ReadOnly {
			return &Error{"save", path, ErrReadOnly, nil}
		}
		if d.Buffer != buffer || d.Path != path || d.IsNew || d.Encoding != encoding || d.BOM != bom ||
			buffer.Version() != version || d.baselineRevision != revision || disk.info != nil && !sameIdentity(d.disk, disk) {
			return &Error{"save", path, ErrDirty, nil}
		}
		if buffer.InGroup() {
			return ErrComposition
		}
		if disk.info == nil {
			return &Error{"save", d.Path, ErrConflict, nil}
		}
		id, err := fingerprint(path, disk.info.Size(), disk.info)
		if errors.Is(err, os.ErrNotExist) {
			return &Error{"save", d.Path, ErrConflict, err}
		}
		if err != nil {
			return err
		}
		if !sameIdentity(disk, id) {
			return &Error{"save", d.Path, ErrConflict, nil}
		}
		if id.info.Mode().Perm()&0200 == 0 {
			return &Error{"save", d.Path, ErrReadOnly, nil}
		}
		return nil
	}
	if err := check(); err != nil {
		return err
	}
	// Opening and saving an untouched file does not run any serializer, change
	// its timestamps or normalize line endings, whitespace, BOM or encoding.
	if !d.Dirty() {
		d.baselineRevision++
		d.dirtyCached = false
		return nil
	}
	if d.Encoding == UTF8 {
		digest, encodedLength, outcome, err := writeUTF8AtomicOutcome(d.Path, d.disk.info.Mode(), false, check, d.Buffer.Snapshot(), d.BOM)
		if err != nil {
			return publicationFailure(d.Path, outcome, false, err)
		}
		return d.finishSave(digest, encodedLength, outcome.info, false)
	}
	bytes, err := encode(d.Buffer.String(), d.Encoding, d.BOM)
	if err != nil {
		return err
	}
	outcome, err := writeAtomicOutcome(d.Path, bytes, d.disk.info.Mode(), false, check)
	if err != nil {
		return publicationFailure(d.Path, outcome, false, err)
	}
	return d.finishSave(sha256.Sum256(bytes), int64(len(bytes)), outcome.info, false)
}

// SaveAs is deliberately non-overwriting. The host may offer a separate
// explicit replace workflow; it must not assume a file-dialog answer is a lock.
func (d *Document) SaveAs(path string) error {
	if d.Buffer.InGroup() {
		return ErrComposition
	}
	path, err := canonical(path)
	if err != nil {
		return err
	}
	if d.Path != "" && path == d.Path && !d.IsNew {
		return d.Save()
	}
	if _, err = os.Lstat(path); err == nil {
		return &Error{"save as", path, ErrExists, nil}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if d.Encoding == UTF8 {
		digest, encodedLength, outcome, err := writeUTF8AtomicOutcome(path, 0600, true, nil, d.Buffer.Snapshot(), d.BOM)
		if err != nil {
			return publicationFailure(path, outcome, false, err)
		}
		d.Path = path
		d.IsNew = false
		d.ReadOnly = false
		return d.finishSave(digest, encodedLength, outcome.info, true)
	}
	bytes, err := encode(d.Buffer.String(), d.Encoding, d.BOM)
	if err != nil {
		return err
	}
	outcome, err := writeAtomicOutcome(path, bytes, 0600, true, nil)
	if err != nil {
		return publicationFailure(path, outcome, false, err)
	}
	d.Path = path
	d.IsNew = false
	d.ReadOnly = false
	return d.finishSave(sha256.Sum256(bytes), int64(len(bytes)), outcome.info, true)
}

func publicationFailure(path string, outcome atomicOutcome, metadataAdopted bool, err error) error {
	if outcome.Committed {
		return &PublicationError{Path: path, MetadataAdopted: metadataAdopted, Cause: err}
	}
	return err
}

func (d *Document) finishSave(expected [32]byte, encodedLength int64, expectedInfo os.FileInfo, metadataAdopted bool) error {
	if err := d.didSave(expected, encodedLength, expectedInfo); err != nil {
		// SaveAs already adopted the target metadata, but even a new buffer
		// equal to its initial snapshot must remain an unverified dirty draft.
		d.unverifiedSave = true
		return &PublicationError{Path: d.Path, MetadataAdopted: metadataAdopted, Cause: err}
	}
	return nil
}

func (d *Document) didSave(expected [32]byte, encodedLength int64, expectedInfo os.FileInfo) error {
	// Read back at most the exact published encoded length plus one. The real
	// intended temp-file identity, not another writer's post-publication Stat,
	// anchors verification. Published bytes cannot be rolled back here; refuse
	// visibly without adopting the live buffer or saved/disk state. SaveAs has
	// already changed Path/IsNew/ReadOnly after successful publication.
	id, err := fingerprint(d.Path, encodedLength, expectedInfo)
	if err != nil {
		return &Error{"verify save", d.Path, err, nil}
	}
	if id.hash != expected {
		return &Error{"verify save", d.Path, ErrConflict, nil}
	}
	d.disk = id
	d.unverifiedSave = false
	d.ReadOnly = id.info.Mode().Perm()&0200 == 0
	d.baselineRevision++
	d.saved = d.Buffer.Snapshot()
	d.savedEncoding, d.savedBOM = d.Encoding, d.BOM
	d.IsNew = false
	d.dirtyCached = false
	return nil
}

func (d *Document) Reload() error {
	if d.Dirty() {
		return ErrDirty
	}
	return d.Revert()
}

// Revert is the explicit discard operation; the UI must name that consequence.
func (d *Document) Revert() error {
	if d.Path == "" {
		return ErrNoPath
	}
	next, err := OpenWithEncoding(d.Path, d.savedEncoding)
	if err != nil {
		return err
	}
	// Preserve Buffer identity: existing native input clients keep their pointer.
	d.Buffer.Restore(next.Buffer.Snapshot())
	next.Buffer = d.Buffer
	revision := d.baselineRevision + 1
	*d = *next
	d.baselineRevision = revision
	return nil
}

func sameIdentity(a, b identity) bool {
	return a.info != nil && b.info != nil && a.hash == b.hash && a.info.Size() == b.info.Size() && os.SameFile(a.info, b.info)
}

func readStable(path string, limit int64) ([]byte, identity, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, identity{}, err
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return nil, identity{}, err
	}
	if !before.Mode().IsRegular() {
		return nil, identity{}, &Error{"open", path, ErrNotRegular, nil}
	}
	if before.Size() > limit {
		return nil, identity{}, &Error{"open", path, ErrTooLarge, nil}
	}
	bytes, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, identity{}, err
	}
	if int64(len(bytes)) > limit {
		return nil, identity{}, &Error{"open", path, ErrTooLarge, nil}
	}
	after, err := f.Stat()
	if err != nil {
		return nil, identity{}, err
	}
	current, err := os.Stat(path)
	if err != nil {
		return nil, identity{}, err
	}
	if before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) ||
		after.Size() != current.Size() || !after.ModTime().Equal(current.ModTime()) || !os.SameFile(after, current) {
		return nil, identity{}, &Error{"open", path, ErrConflict, nil}
	}
	return bytes, identity{sha256.Sum256(bytes), after}, nil
}
func fingerprint(path string, expectedLength int64, expectedInfo os.FileInfo) (identity, error) {
	current, err := os.Stat(path)
	if err != nil {
		return identity{}, err
	}
	if !current.Mode().IsRegular() {
		return identity{}, ErrNotRegular
	}
	if current.Size() != expectedLength || expectedInfo != nil && !os.SameFile(expectedInfo, current) {
		return identity{}, ErrConflict
	}
	f, err := os.Open(path)
	if err != nil {
		return identity{}, err
	}
	defer f.Close()
	return fingerprintOpened(path, f, expectedLength, expectedInfo)
}

type fingerprintFile interface {
	io.Reader
	Stat() (os.FileInfo, error)
}

// fingerprintOpened hashes at most the expected encoded bytes plus one. Expected
// info establishes size and file identity, never baseline mtime: a same-file,
// same-content touch is not a conflict. Only observations during this read must
// agree on mtime, so concurrent writes cannot establish a successful identity.
func fingerprintOpened(path string, f fingerprintFile, expectedLength int64, expectedInfo os.FileInfo) (identity, error) {
	before, err := f.Stat()
	if err != nil {
		return identity{}, err
	}
	if !before.Mode().IsRegular() {
		return identity{}, ErrNotRegular
	}
	if expectedLength < 0 || expectedLength == 1<<63-1 || before.Size() != expectedLength ||
		expectedInfo != nil && (expectedInfo.Size() != expectedLength || !os.SameFile(expectedInfo, before)) {
		return identity{}, ErrConflict
	}
	h := sha256.New()
	count, err := io.Copy(h, io.LimitReader(f, expectedLength+1))
	if err != nil {
		return identity{}, err
	}
	after, err := f.Stat()
	if err != nil {
		return identity{}, err
	}
	current, err := os.Stat(path)
	if err != nil {
		return identity{}, err
	}
	if count != expectedLength || !after.Mode().IsRegular() || !current.Mode().IsRegular() ||
		before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || !os.SameFile(before, after) ||
		after.Size() != current.Size() || !after.ModTime().Equal(current.ModTime()) || !os.SameFile(after, current) {
		return identity{}, ErrConflict
	}
	var sum [32]byte
	copy(sum[:], h.Sum(nil))
	return identity{sum, after}, nil
}

type Inspection struct {
	Path      string
	Size      int64
	Bytes     []byte
	Truncated bool
	ReadOnly  bool
}

// Inspect reads at most limit bytes for the always-available read-only fallback.
// Binary data is never passed to the text decoder or written back from this view.
func Inspect(path string, limit int) (Inspection, error) {
	if limit <= 0 || limit > 1<<20 {
		limit = 64 << 10
	}
	path, err := canonical(path)
	if err != nil {
		return Inspection{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return Inspection{}, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return Inspection{}, err
	}
	if !info.Mode().IsRegular() {
		return Inspection{}, ErrNotRegular
	}
	bytes, err := io.ReadAll(io.LimitReader(f, int64(limit)))
	if err != nil {
		return Inspection{}, err
	}
	return Inspection{path, info.Size(), bytes, info.Size() > int64(len(bytes)), true}, nil
}
