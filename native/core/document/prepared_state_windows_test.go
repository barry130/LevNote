//go:build windows

package document

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

const privateStateDeleteAccess uint32 = 0x00010000 // Windows DELETE access right.

// denyWindowsReadAllowReplace denies new read opens but shares write/delete.
// The caller's Read and ReplaceFileW permissions are intentionally different.
func denyWindowsReadAllowReplace(t *testing.T, path string) (release func()) {
	t.Helper()
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_WRITE|privateStateDeleteAccess,
		syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE, nil,
		syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	closed := false
	return func() {
		if !closed {
			closed = true
			if err := syscall.CloseHandle(h); err != nil {
				t.Errorf("close deny-read handle: %v", err)
			}
		}
	}
}

func TestPrivateStateRetryClassifierOnlyAcceptsDirectOpenSharingErrors(t *testing.T) {
	if !privateStateTargetReadRetryable(&privateStateTargetReadRetryError{
		err: &privateStateFileOpenError{err: syscall.Errno(32)},
	}) {
		t.Fatal("direct target-open sharing violation was not retryable")
	}
	for _, err := range []error{
		&privateStateTargetReadRetryError{err: errors.Join(
			&privateStateFileOpenError{err: syscall.Errno(32)}, errors.New("close failed"))},
		&privateStateTargetReadRetryError{err: errors.New("target metadata failed")},
		&privateStateTargetReadRetryError{err: &privateStateFileOpenError{err: syscall.Errno(5)}},
	} {
		if privateStateTargetReadRetryable(err) {
			t.Fatalf("non-direct-open or non-sharing error was retried: %v", err)
		}
	}
}

func TestDenyReadAllowDeleteStillAllowsRealReplaceFileW(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "session.json")
	temp := filepath.Join(dir, ".candidate")
	if err := os.WriteFile(target, []byte("old recovery"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(temp, []byte("new recovery"), 0600); err != nil {
		t.Fatal(err)
	}
	release := denyWindowsReadAllowReplace(t, target)
	readErr := func() error {
		_, err := os.ReadFile(target)
		return err
	}()
	if !privateStateSharingViolation(readErr) {
		release()
		t.Fatalf("fixture failed to deny target read with Windows sharing violation: %v", readErr)
	}
	// This is an actual ReplaceFileW operation, not a mocked permission check.
	replaceErr := replaceExisting(temp, target, nil)
	release()
	if replaceErr != nil {
		t.Fatalf("deny-read/allow-delete handle unexpectedly blocked ReplaceFileW: %v", replaceErr)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "new recovery" {
		t.Fatalf("ReplaceFileW did not publish candidate: %q %v", got, err)
	}
}

func prepareWindowsPrivateState(t *testing.T, target, candidate string) *PreparedPrivateState {
	t.Helper()
	prepared, err := PreparePrivateStateTo(target, func(writer io.Writer) error {
		_, err := io.WriteString(writer, candidate)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return prepared
}

func TestPreparedPrivateStateWaitsForTargetIdentityReadThenRechecks(t *testing.T) {
	target := filepath.Join(t.TempDir(), "session.json")
	if err := os.WriteFile(target, []byte("old recovery"), 0600); err != nil {
		t.Fatal(err)
	}
	prepared := prepareWindowsPrivateState(t, target, "new recovery")
	release := denyWindowsReadAllowReplace(t, target)
	wait := replaceRetryWait
	defer func() { replaceRetryWait = wait }()
	waits := 0
	replaceRetryWait = func(delay time.Duration) {
		if delay != replaceRetryDelay {
			t.Errorf("identity retry delay=%v, want %v", delay, replaceRetryDelay)
		}
		waits++
		release()
	}
	if err := prepared.Commit(nil); err != nil {
		release()
		t.Fatalf("transient target-read lock prevented publication: %v", err)
	}
	release()
	if waits != 1 {
		t.Fatalf("target identity read did not use one existing retry wait: waits=%d", waits)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "new recovery" {
		t.Fatalf("committed target=%q err=%v", got, err)
	}
	if err := prepared.Abort(); err != nil {
		t.Fatalf("cleanup after commit: %v", err)
	}
}

func TestPreparedPrivateStatePermanentTargetReadLockFailsClosed(t *testing.T) {
	target := filepath.Join(t.TempDir(), "session.json")
	old := []byte("old recovery")
	if err := os.WriteFile(target, old, 0600); err != nil {
		t.Fatal(err)
	}
	prepared := prepareWindowsPrivateState(t, target, "new recovery")
	release := denyWindowsReadAllowReplace(t, target)
	wait := replaceRetryWait
	defer func() { replaceRetryWait = wait }()
	waits := 0
	replaceRetryWait = func(time.Duration) { waits++ }
	err := prepared.Commit(nil)
	if err == nil || waits != replaceRetryAttempts-1 {
		release()
		t.Fatalf("permanent unreadable target was not bounded/fail-closed: waits=%d err=%v", waits, err)
	}
	var publication *StatePublicationError
	if !errors.As(err, &publication) || publication.Committed {
		release()
		t.Fatalf("permanent read lock outcome=%#v err=%v", publication, err)
	}
	release()
	if got, readErr := os.ReadFile(target); readErr != nil || string(got) != string(old) {
		t.Fatalf("failed identity guard changed original bytes: %q %v", got, readErr)
	}
	if err := prepared.Abort(); err != nil {
		t.Fatalf("abort after permanent read-lock refusal: %v", err)
	}
}

func TestPreparedPrivateStateRejectsIdentityOrOwnerChangeDuringReadWait(t *testing.T) {
	for _, mode := range []string{"target", "temp", "directory", "owner"} {
		t.Run(mode, func(t *testing.T) {
			parent := t.TempDir()
			dir := filepath.Join(parent, "state")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(dir, "session.json")
			old := []byte("old recovery")
			if err := os.WriteFile(target, old, 0600); err != nil {
				t.Fatal(err)
			}
			prepared := prepareWindowsPrivateState(t, target, "new recovery")
			release := denyWindowsReadAllowReplace(t, target)
			wait := replaceRetryWait
			defer func() { replaceRetryWait = wait }()
			waits := 0
			ownerErr := errors.New("owner changed during retry wait")
			ownerChecks := 0
			check := func() error {
				ownerChecks++
				if mode == "owner" && waits > 0 {
					return ownerErr
				}
				return nil
			}
			replaceRetryWait = func(time.Duration) {
				waits++
				release()
				switch mode {
				case "target":
					if err := os.Rename(target, target+".old"); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(target, old, 0600); err != nil {
						t.Fatal(err)
					}
					if after := privateStateTestInfo(t, target); os.SameFile(prepared.targetInfo, after) {
					t.Fatal("wait fixture did not replace target file identity")
					}
				case "temp":
					if err := os.Rename(prepared.tempPath, prepared.tempPath+".old"); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(prepared.tempPath, []byte("changed candidate"), 0600); err != nil {
						t.Fatal(err)
					}
				case "directory":
					if err := os.Rename(dir, dir+".old"); err != nil {
						t.Fatal(err)
					}
					if err := os.Mkdir(dir, 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(target, old, 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			err := prepared.Commit(check)
			release()
			var publication *StatePublicationError
			if !errors.As(err, &publication) || publication.Committed || waits != 1 {
				t.Fatalf("%s mutation crossed publication guard: waits=%d result=%#v err=%v", mode, waits, publication, err)
			}
			if mode == "owner" && !errors.Is(err, ownerErr) {
				t.Fatalf("owner invalidation cause was lost: %v", err)
			}
			if mode == "target" {
				if got, readErr := os.ReadFile(target); readErr != nil || string(got) != string(old) {
					t.Fatalf("replacement target bytes were changed: %q %v", got, readErr)
				}
			} else if mode != "directory" {
				if got, readErr := os.ReadFile(target); readErr != nil || string(got) != string(old) {
					t.Fatalf("precommit failure changed target: %q %v", got, readErr)
				}
			}
			if mode == "temp" {
				if err := prepared.Abort(); err == nil {
					t.Fatal("Abort silently removed a replacement temp with unknown identity")
				}
				if got, readErr := os.ReadFile(prepared.tempPath); readErr != nil || string(got) != "changed candidate" {
					t.Fatalf("replacement temp changed during refusal: %q %v", got, readErr)
				}
				for _, path := range []string{prepared.tempPath, prepared.tempPath + ".old"} {
					if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
						t.Fatal(err)
					}
				}
			} else {
				if err := prepared.Abort(); err != nil {
					t.Fatal(err)
				}
				if mode == "directory" {
					oldTemp := filepath.Join(dir+".old", filepath.Base(prepared.tempPath))
					if err := os.Remove(oldTemp); err != nil && !errors.Is(err, os.ErrNotExist) {
						t.Fatal(err)
					}
				}
			}
		})
	}
}
