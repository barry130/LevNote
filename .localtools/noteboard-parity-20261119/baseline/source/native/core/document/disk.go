package document

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// DiskToken is an immutable identity of the document's last verified saved
// baseline. It contains no mutable document references and may be held by a
// worker while the UI continues editing the in-memory buffer.
type DiskToken struct {
	path     string
	hash     [32]byte
	info     os.FileInfo
	revision uint64
	valid    bool
}

// DiskCheckResult reports the state of the file at the token's path relative to
// the captured baseline. It is safe to copy across goroutines.
type DiskCheckResult struct {
	Changed bool
	Missing bool
	Invalid bool
}

// DiskToken captures the immutable baseline of a saved document on the UI
// thread. New documents have no disk baseline and return a zero token.
func (d *Document) DiskToken() DiskToken {
	if d.IsNew || d.Path == "" || d.disk.info == nil {
		return DiskToken{}
	}
	return DiskToken{
		path: d.Path, hash: d.disk.hash, info: d.disk.info,
		revision: d.baselineRevision, valid: true,
	}
}

// Valid reports whether the token came from a saved disk baseline.
func (t DiskToken) Valid() bool { return t.valid }

// Path returns the absolute immutable path captured with the token.
func (t DiskToken) Path() string { return t.path }

// CheckDisk compares the captured path and baseline without touching Document
// or Buffer. It hashes the current regular file with bounded reads and honors
// context cancellation. Same-size, same-mtime replacement remains detectable by
// the content hash.
func (t DiskToken) CheckDisk(ctx context.Context) (DiskCheckResult, error) {
	if err := ctx.Err(); err != nil {
		return DiskCheckResult{}, err
	}
	if !t.valid {
		return DiskCheckResult{Invalid: true}, nil
	}
	observe := diskCheckHook
	if observe == nil {
		observe = func() error { return nil }
	}
	info, err := os.Stat(t.path)
	if errors.Is(err, os.ErrNotExist) {
		return DiskCheckResult{Changed: true, Missing: true}, nil
	}
	if err != nil {
		return DiskCheckResult{Invalid: true}, err
	}
	if err := ctx.Err(); err != nil {
		return DiskCheckResult{}, err
	}
	if !info.Mode().IsRegular() || !sameIdentity(identity{t.hash, t.info}, identity{t.hash, info}) {
		return DiskCheckResult{Changed: true}, nil
	}
	file, err := os.Open(t.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DiskCheckResult{Changed: true, Missing: true}, nil
		}
		return DiskCheckResult{Invalid: true}, err
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil {
		return DiskCheckResult{Invalid: true}, err
	}
	hash := sha256.New()
	buf := make([]byte, 64<<10)
	hashedBytes := 0
	for {
		if err := ctx.Err(); err != nil {
			return DiskCheckResult{}, err
		}
		if hashedBytes > 0 {
			if err := observe(); err != nil {
				return DiskCheckResult{}, err
			}
		}
		read, readErr := file.Read(buf)
		if read > 0 {
			if _, err = hash.Write(buf[:read]); err != nil {
				return DiskCheckResult{Invalid: true}, err
			}
			hashedBytes += read
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return DiskCheckResult{Invalid: true}, readErr
		}
	}
	after, err := file.Stat()
	if err != nil {
		return DiskCheckResult{Invalid: true}, err
	}
	current, err := os.Stat(t.path)
	if err != nil {
		return DiskCheckResult{Invalid: true}, err
	}
	if err := ctx.Err(); err != nil {
		return DiskCheckResult{}, err
	}
	if before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) ||
		!os.SameFile(before, after) || !os.SameFile(after, current) ||
		after.Size() != current.Size() || !after.ModTime().Equal(current.ModTime()) {
		return DiskCheckResult{Invalid: true}, ErrConflict
	}
	var sum [32]byte
	copy(sum[:], hash.Sum(nil))
	return DiskCheckResult{Changed: !sameIdentity(identity{t.hash, t.info}, identity{sum, after})}, nil
}

var diskCheckHook func() error

// SameBaseline reports whether another UI-thread token still identifies this
// document's current saved baseline. Save, SaveAs and Revert must capture a
// new token; an old response compared with the new token is stale even if the
// bytes are the same.
func (d *Document) SameBaseline(token DiskToken) bool {
	if !token.Valid() {
		return false
	}
	if d.IsNew || d.Path == "" || d.disk.info == nil {
		return false
	}
	if d.Path != token.path || d.disk.hash != token.hash ||
		!sameIdentity(d.disk, identity{token.hash, token.info}) ||
		d.baselineRevision != token.revision {
		return false
	}
	return true
}

// normalizeDiskPath exists for tests and callers that need the same canonical
// path rule as document open/save paths without exposing canonical outside the
// package.
func normalizeDiskPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}
