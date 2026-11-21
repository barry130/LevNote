package document

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// StatePublicationError identifies whether private state was published before
// an error. Committed means the new target is visible; durability may still be
// unconfirmed. Aborting never rolls back or deletes that target.
type StatePublicationError struct {
	Path       string
	Committed  bool
	Err        error
	CleanupErr error
}

func (e *StatePublicationError) Error() string {
	if e.Committed {
		if e.Err != nil && e.CleanupErr != nil {
			return fmt.Sprintf("private state %s was published; directory durability is unconfirmed: %v; temporary cleanup failed: %v", e.Path, e.Err, e.CleanupErr)
		}
		if e.CleanupErr != nil {
			return fmt.Sprintf("private state %s was published and directory sync completed; temporary cleanup failed: %v", e.Path, e.CleanupErr)
		}
		return fmt.Sprintf("private state %s was published; directory durability is unconfirmed: %v", e.Path, e.Err)
	}
	if e.CleanupErr != nil {
		if e.Err != nil {
			return fmt.Sprintf("private state %s was not published: %v; temporary cleanup failed: %v", e.Path, e.Err, e.CleanupErr)
		}
		return fmt.Sprintf("private state %s was not published; temporary cleanup failed: %v", e.Path, e.CleanupErr)
	}
	return fmt.Sprintf("private state %s was not published: %v", e.Path, e.Err)
}
func (e *StatePublicationError) Unwrap() []error {
	if e.Err == nil {
		if e.CleanupErr == nil {
			return nil
		}
		return []error{e.CleanupErr}
	}
	if e.CleanupErr == nil {
		return []error{e.Err}
	}
	return []error{e.Err, e.CleanupErr}
}

// PreparedPrivateState owns one fully written, synced and closed temp file.
// The caller serializes Commit/Abort. It cannot redirect this handle to a new
// target, rebase to a changed directory, or publish after Abort.
const maxPrivateStateBytes int64 = 16 << 20

var errPrivateStateLimit = errors.New("private state exceeds 16 MiB limit")

type PreparedPrivateState struct {
	path, tempPath string
	dirInfo        os.FileInfo
	targetInfo     os.FileInfo
	targetDigest   [sha256.Size]byte
	tempInfo       os.FileInfo
	tempDigest     [sha256.Size]byte
	committed      bool
	aborted        bool
}

type privateStateFileOpenError struct{ err error }

func (e *privateStateFileOpenError) Error() string { return e.err.Error() }
func (e *privateStateFileOpenError) Unwrap() error { return e.err }

type privateStateTargetReadRetryError struct{ err error }

func (e *privateStateTargetReadRetryError) Error() string { return e.err.Error() }
func (e *privateStateTargetReadRetryError) Unwrap() error { return e.err }

// retryPrivateStateTargetRead retries only the caller-identified target-open
// sharing error. Each attempt reacquires the original identity checks; owner is
// checked after each wait and again before allowing publication. This narrows
// the window but does not provide an operating-system compare-and-swap.
func retryPrivateStateTargetRead(unchanged, owner func() error, retryable func(error) bool, wait func(), maxWaits int) error {
	checkOwner := func() error {
		if owner == nil {
			return nil
		}
		return owner()
	}
	err := unchanged()
	if err == nil {
		return checkOwner()
	}
	for waits := 0; ; waits++ {
		if waits >= maxWaits || retryable == nil || !retryable(err) {
			return err
		}
		wait()
		// Every delay is followed by a complete identity recheck and an owner
		// recheck. Unreadable identity stays a failure; it is never adopted.
		err = unchanged()
		ownerErr := checkOwner()
		if ownerErr != nil {
			return errors.Join(ownerErr, err)
		}
		if err == nil {
			return nil
		}
	}
}

// PreparePrivateStateTo leaves the existing recovery file unchanged. A writer
// failure is detected before the caller needs to stop an existing service.
func PreparePrivateStateTo(path string, write func(io.Writer) error) (_ *PreparedPrivateState, err error) {
	if write == nil {
		return nil, errors.New("private-state writer is nil")
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	p := &PreparedPrivateState{path: path}
	if p.dirInfo, err = inspectPrivateStateDirectory(dir); err != nil {
		return nil, err
	}
	p.targetInfo, p.targetDigest, err = inspectPrivateStateFile(path)
	if errors.Is(err, os.ErrNotExist) {
		p.targetInfo, err = nil, nil
	}
	if err != nil {
		return nil, err
	}
	temp, err := os.CreateTemp(dir, ".levnote-save-*")
	if err != nil {
		return nil, err
	}
	p.tempPath = temp.Name()
	defer func() {
		if err != nil {
			err = errors.Join(err, p.Abort())
		}
	}()
	p.tempInfo, err = temp.Stat()
	if err == nil {
		err = temp.Chmod(0600)
	}
	limited := &privateStateLimitWriter{writer: temp}
	if err == nil {
		err = write(limited)
		if err == nil && limited.exceeded {
			err = errPrivateStateLimit
		}
	}
	if err == nil {
		err = temp.Sync()
	}
	if err == nil {
		p.tempInfo, err = temp.Stat()
	}
	err = errors.Join(err, temp.Close())
	if err != nil {
		return nil, err
	}
	p.tempInfo, p.tempDigest, err = inspectPrivateStateFile(p.tempPath)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *PreparedPrivateState) unchanged(owner func() error) error {
	return retryPrivateStateTargetRead(p.unchangedOnce, owner, privateStateTargetReadRetryable,
		privateStateTargetReadWait, privateStateTargetReadWaitBudget())
}

func (p *PreparedPrivateState) unchangedOnce() error {
	dirInfo, err := inspectPrivateStateDirectory(filepath.Dir(p.path))
	if err != nil || !os.SameFile(p.dirInfo, dirInfo) {
		return errors.Join(errors.New("private-state directory changed during preparation"), err)
	}
	tempInfo, tempDigest, err := inspectPrivateStateFile(p.tempPath)
	if err != nil || !os.SameFile(p.tempInfo, tempInfo) ||
		tempInfo.Size() != p.tempInfo.Size() || !tempInfo.ModTime().Equal(p.tempInfo.ModTime()) || tempDigest != p.tempDigest {
		return errors.Join(errors.New("prepared private-state temporary file changed"), err)
	}
	info, digest, err := inspectPrivateStateFile(p.path)
	if p.targetInfo != nil {
		var openErr *privateStateFileOpenError
		if errors.As(err, &openErr) && privateStateSharingViolation(openErr.err) {
			// A read-denying handle may still allow ReplaceFileW. Do not treat
			// this as an unchanged target; the bounded caller retries the direct
			// open and rechecks every original identity before publication.
			return &privateStateTargetReadRetryError{err: err}
		}
	}
	if p.targetInfo == nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return errors.Join(errors.New("private-state target appeared during preparation"), err)
	}
	if err != nil || !os.SameFile(p.targetInfo, info) || info.Size() != p.targetInfo.Size() ||
		!info.ModTime().Equal(p.targetInfo.ModTime()) || digest != p.targetDigest {
		return errors.Join(errors.New("private-state target changed during preparation"), err)
	}
	return nil
}

// privateStateLimitWriter bounds candidate bytes before they reach the temp.
type privateStateLimitWriter struct {
	writer   io.Writer
	written  int64
	exceeded bool
}

func (w *privateStateLimitWriter) Write(data []byte) (int, error) {
	if int64(len(data)) > maxPrivateStateBytes-w.written {
		w.exceeded = true
		return 0, errPrivateStateLimit
	}
	n, err := w.writer.Write(data)
	w.written += int64(n)
	return n, err
}

func inspectPrivateStateDirectory(path string) (os.FileInfo, error) {
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	opened, statErr := dir.Stat()
	closeErr := dir.Close()
	if statErr != nil || closeErr != nil {
		return nil, errors.Join(statErr, closeErr)
	}
	if !opened.IsDir() {
		return nil, errors.New("private-state parent is not a directory")
	}
	current, err := os.Open(path)
	if err != nil {
		return nil, errors.Join(errors.New("private-state directory changed while inspected"), err)
	}
	currentInfo, statErr := current.Stat()
	closeErr = current.Close()
	if statErr != nil || closeErr != nil || !currentInfo.IsDir() || !os.SameFile(opened, currentInfo) {
		return nil, errors.Join(errors.New("private-state directory changed while inspected"), statErr, closeErr)
	}
	return opened, nil
}

func inspectPrivateStateFile(path string) (os.FileInfo, [sha256.Size]byte, error) {
	var digest [sha256.Size]byte
	before, err := os.Lstat(path)
	if err != nil {
		return nil, digest, err
	}
	if !before.Mode().IsRegular() {
		return nil, digest, errors.New("private-state file is not a regular file")
	}
	if before.Size() > maxPrivateStateBytes {
		return nil, digest, errPrivateStateLimit
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, digest, &privateStateFileOpenError{err: err}
	}
	opened, err := file.Stat()
	if err == nil && (!opened.Mode().IsRegular() || opened.Size() != before.Size() || !opened.ModTime().Equal(before.ModTime())) {
		err = errors.New("private-state file changed while opening")
	}
	var hash [sha256.Size]byte
	if err == nil {
		_, err = file.Seek(0, io.SeekStart)
		if err == nil {
			h := sha256.New()
			var size int64
			size, err = io.Copy(h, io.LimitReader(file, maxPrivateStateBytes+1))
			if err == nil && size > maxPrivateStateBytes {
				err = errPrivateStateLimit
			}
			if err == nil {
				copy(hash[:], h.Sum(nil))
			}
		}
	}
	afterHandle, statErr := file.Stat()
	if err == nil {
		err = statErr
	}
	if err == nil && (opened.Size() != afterHandle.Size() || !opened.ModTime().Equal(afterHandle.ModTime())) {
		err = errors.New("private-state file changed while reading")
	}
	if err != nil {
		closeErr := file.Close()
		return nil, digest, errors.Join(err, closeErr)
	}
	current, openErr := os.Open(path)
	if openErr != nil {
		_ = file.Close()
		return nil, digest, errors.Join(errors.New("private-state file identity changed while reading"), openErr)
	}
	currentInfo, currentStatErr := current.Stat()
	afterPath, pathErr := os.Lstat(path)
	if pathErr == nil && !afterPath.Mode().IsRegular() {
		pathErr = errors.New("private-state file is not a regular file")
	}
	final, finalOpenErr := os.Open(path)
	var finalInfo os.FileInfo
	var finalStatErr, finalCloseErr error
	if finalOpenErr == nil {
		finalInfo, finalStatErr = final.Stat()
		finalCloseErr = final.Close()
	}
	currentCloseErr := current.Close()
	closeErr := file.Close()
	if currentStatErr != nil || pathErr != nil || finalOpenErr != nil || finalStatErr != nil ||
		finalCloseErr != nil || currentCloseErr != nil || closeErr != nil {
		return nil, digest, errors.Join(errors.New("private-state file identity changed while reading"), currentStatErr,
			pathErr, finalOpenErr, finalStatErr, finalCloseErr, currentCloseErr, closeErr)
	}
	if !currentInfo.Mode().IsRegular() || !finalInfo.Mode().IsRegular() ||
		!os.SameFile(afterHandle, currentInfo) || !os.SameFile(currentInfo, finalInfo) ||
		afterPath.Size() != finalInfo.Size() || !afterPath.ModTime().Equal(finalInfo.ModTime()) {
		return nil, digest, errors.New("private-state file identity changed while reading")
	}
	return afterHandle, hash, nil
}

// Commit repeats the original checks on Windows ReplaceFileW retries. It uses
// the same OS-specific publication primitives as document saves. There is no
// delete-and-move fallback and no ordinary-filesystem CAS guarantee.
func (p *PreparedPrivateState) Commit(check func() error) error {
	if p == nil || p.aborted || p.committed {
		return errors.New("prepared private state is no longer pending")
	}
	guard := func() error { return p.unchanged(check) }
	if err := guard(); err != nil {
		return &StatePublicationError{Path: p.path, Err: err}
	}
	var err error
	if p.targetInfo == nil {
		err = publishNew(p.tempPath, p.path)
	} else {
		err = replaceExisting(p.tempPath, p.path, guard)
	}
	if err != nil {
		return &StatePublicationError{Path: p.path, Err: err}
	}
	p.committed = true
	if err := atomicSyncDirectory(filepath.Dir(p.path)); err != nil {
		return &StatePublicationError{Path: p.path, Committed: true, Err: err}
	}
	return nil
}

// Abort cleans only the exact temp owned by this handle, including an extra
// hard link left by publishNew. A changed temp is never removed. It is safe to
// repeat after success or a failed Commit; published targets remain untouched.
func (p *PreparedPrivateState) Abort() error {
	if p == nil || p.tempPath == "" || p.aborted {
		return nil
	}
	info, err := os.Lstat(p.tempPath)
	if errors.Is(err, os.ErrNotExist) {
		p.aborted = true
		return nil
	}
	if err != nil {
		return err
	}
	if p.tempInfo == nil || !os.SameFile(p.tempInfo, info) {
		return errors.New("prepared private-state temporary file identity is unverified; not removed")
	}
	if err := os.Remove(p.tempPath); err != nil {
		return err
	}
	p.aborted = true
	return nil
}
