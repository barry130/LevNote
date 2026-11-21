//go:build windows

package document

import (
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// lockExclusive opens path with no sharing, so ReplaceFileW cannot replace it.
// It returns a release function; callers decide when the external holder goes
// away. This models an antivirus scanner or indexer holding the destination.
func lockExclusive(t *testing.T, path string) (release func()) {
	t.Helper()
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ, 0, nil,
		syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	closed := false
	return func() {
		if !closed {
			closed = true
			_ = syscall.CloseHandle(h)
		}
	}
}

func TestPreparedPrivateStateWaitsOutTransientWindowsReplaceLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	if err := os.WriteFile(path, []byte("old recovery"), 0600); err != nil {
		t.Fatal(err)
	}
	prepared, err := PreparePrivateStateTo(path, func(w io.Writer) error {
		_, err := io.WriteString(w, "new recovery")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := prepared.Abort(); err != nil {
			t.Error(err)
		}
	}()
	release := lockExclusive(t, path)
	defer release()
	wait := replaceRetryWait
	defer func() { replaceRetryWait = wait }()
	waits := 0
	replaceRetryWait = func(delay time.Duration) {
		wait(delay)
		waits++
		if waits == 1 {
			release()
		}
	}
	if err := prepared.Commit(nil); err != nil {
		t.Fatalf("a transient Windows ReplaceFile lock failed prepared state: %v", err)
	}
	if waits != 1 {
		t.Fatalf("prepared state did not use the bounded ReplaceFile retry: waits=%d", waits)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "new recovery" {
		t.Fatalf("prepared state bytes = %q, err=%v", got, err)
	}
}

func TestSaveWaitsOutTransientWindowsReplaceLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doc.md")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	release := lockExclusive(t, path)
	go func() {
		time.Sleep(80 * time.Millisecond)
		release()
	}()
	defer release()

	if err := writeAtomic(path, []byte("new"), 0600, false, nil); err != nil {
		t.Fatalf("a transient external lock failed the save: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("saved bytes = %q, want %q", got, "new")
	}
}

// A destination that stays locked must fail loudly, and the original bytes must
// survive: no delete-and-move fallback, no silent success.
func TestSaveFailsExplicitlyWhenWindowsReplaceStaysLocked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doc.md")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	release := lockExclusive(t, path)
	defer release()

	err := writeAtomic(path, []byte("new"), 0600, false, nil)
	if err == nil {
		t.Fatal("a locked destination reported a successful save")
	}
	// The original must survive intact. Read it only after the external holder
	// releases it: the lock this test creates denies reads too, so reading while
	// held would prove nothing about the file's contents.
	release()
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("original document became unreadable: %v", readErr)
	}
	if string(got) != "old" {
		t.Fatalf("original bytes were destroyed: %q", got)
	}
	// The failed attempt must not leave temp files behind.
	entries, dirErr := os.ReadDir(filepath.Dir(path))
	if dirErr != nil {
		t.Fatal(dirErr)
	}
	for _, entry := range entries {
		if entry.Name() != "doc.md" {
			t.Fatalf("failed save left %q behind", entry.Name())
		}
	}
}
