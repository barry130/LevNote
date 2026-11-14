//go:build windows

package document

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// Reading and writing remain shared, but deletion/replacement is denied. Thus
// Document.Save can perform its genuine fingerprint checks while ReplaceFileW
// fails on the real first publication attempt, not on a mocked predicate.
func denyWindowsReplacement(t *testing.T, path string) func() {
	t.Helper()
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil,
		syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	closed := false
	return func() {
		if !closed {
			closed = true
			if err := syscall.CloseHandle(h); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestDocumentSaveWindowsLateRetryAndBudget(t *testing.T) {
	for _, mode := range []string{"late-change", "late-release", "persistent"} {
		t.Run(mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "retry.txt")
			put(t, path, []byte("old"))
			doc, err := Open(path)
			if err != nil { t.Fatal(err) }
			baseline := doc.DiskToken()
			if err := doc.Buffer.ReplaceUTF16(0, 3, "draft"); err != nil { t.Fatal(err) }
			release := denyWindowsReplacement(t, path)
			defer release()
			wait := replaceRetryWait
			defer func() { replaceRetryWait = wait }()
			waits := 0
			replaceRetryWait = func(delay time.Duration) {
				if delay != 25*time.Millisecond { t.Fatalf("retry delay changed: %v", delay) }
				wait(delay)
				waits++
				if waits == 5 && mode != "persistent" {
					release()
					if mode == "late-change" { put(t, path, []byte("out")) }
				}
			}
			err = doc.Save()
			release()
			if waits != 5 { t.Fatalf("real transient OS retries exceeded six-attempt/five-delay budget: waits=%d err=%v", waits, err) }
			switch mode {
			case "late-change":
				if !errors.Is(err, ErrConflict) || string(read(t, path)) != "out" || !doc.Dirty() || !doc.SameBaseline(baseline) { t.Fatalf("last delayed attempt overwrote changed bytes: %v", err) }
			case "late-release":
				if err != nil || string(read(t, path)) != "draft" || doc.Dirty() { t.Fatalf("sixth attempt could not save unchanged target after lock release: %v", err) }
			case "persistent":
				if err == nil || !isTransientReplaceError(err) || string(read(t, path)) != "old" || !doc.Dirty() || !doc.SameBaseline(baseline) { t.Fatalf("persistent real lock lost OS error/original draft: %v", err) }
			}
			matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".levnote-save-*"))
			if err != nil || len(matches) != 0 { t.Fatalf("failed/successful retry left staging temps: %v %v", matches, err) }
		})
	}
}

func TestDocumentSaveWindowsRetryIdentityAndMtime(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		name := "same-file-mtime-touch"
		if replacement {
			name = "same-bytes-new-file-same-mtime"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "retry.txt")
			put(t, path, []byte("old"))
			stamp := time.Unix(1600000000, 0)
			if err := os.Chtimes(path, stamp, stamp); err != nil {
				t.Fatal(err)
			}
			doc, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			baseline := doc.DiskToken()
			if err := doc.Buffer.ReplaceUTF16(0, 3, "draft"); err != nil {
				t.Fatal(err)
			}
			originalFile, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			// File.Stat captures the actual handle identity on Windows. os.Stat
			// may lazily resolve identity later, after this path was replaced.
			before, err := originalFile.Stat()
			_ = originalFile.Close()
			if err != nil {
				t.Fatal(err)
			}
			release := denyWindowsReplacement(t, path)
			defer release()
			wait := replaceRetryWait
			defer func() { replaceRetryWait = wait }()
			waits := 0
			replaceRetryWait = func(delay time.Duration) {
				wait(delay)
				waits++
				if waits != 1 {
					return
				}
				release()
				mtime := stamp.Add(time.Hour)
				if replacement {
					if err := os.Rename(path, path+".external-old"); err != nil {
						t.Fatal(err)
					}
					put(t, path, []byte("old"))
					mtime = stamp
				}
				if err := os.Chtimes(path, mtime, mtime); err != nil {
					t.Fatal(err)
				}
				after, err := os.Stat(path)
				if err != nil || os.SameFile(before, after) == replacement || !after.ModTime().Equal(mtime) {
					t.Fatalf("fixture did not establish intended identity/mtime: %v", err)
				}
			}
			err = doc.Save()
			if waits != 1 {
				t.Fatal("real ReplaceFileW did not enter the delayed attempt")
			}
			if replacement {
				if !errors.Is(err, ErrConflict) || string(read(t, path)) != "old" || !doc.Dirty() || !doc.SameBaseline(baseline) {
					t.Fatalf("same-byte new identity was adopted or overwritten: %v", err)
				}
			} else if err != nil || string(read(t, path)) != "draft" || doc.Dirty() || doc.SameBaseline(baseline) {
				t.Fatalf("same-file same-byte mtime touch falsely refused: %v", err)
			}
		})
	}
}

func TestDocumentSaveRefusesBufferChangeDuringWindowsRetry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "retry.txt")
	put(t, path, []byte("old"))
	doc, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	baseline := doc.DiskToken()
	if err := doc.Buffer.ReplaceUTF16(0, 3, "draft"); err != nil {
		t.Fatal(err)
	}
	release := denyWindowsReplacement(t, path)
	defer release()
	wait := replaceRetryWait
	defer func() { replaceRetryWait = wait }()
	waits := 0
	replaceRetryWait = func(delay time.Duration) {
		wait(delay)
		waits++
		if waits == 1 {
			release()
			if err := doc.Buffer.ReplaceUTF16(0, 5, "newer draft"); err != nil {
				t.Fatal(err)
			}
		}
	}
	err = doc.Save()
	if !errors.Is(err, ErrDirty) || waits != 1 || string(read(t, path)) != "old" ||
		doc.Buffer.String() != "newer draft" || !doc.Dirty() || !doc.SameBaseline(baseline) || doc.Buffer.History().UndoDepth != 2 {
		t.Fatalf("delayed publication accepted a stale snapshot or cleaned newer draft: err=%v target=%q dirty=%v", err, read(t, path), doc.Dirty())
	}
}

func TestDocumentSaveRefusesChangedBytesDuringWindowsRetry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "retry.txt")
	put(t, path, []byte("old"))
	stamp := time.Unix(1600000000, 0)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	doc, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	baseline, buffer := doc.DiskToken(), doc.Buffer
	if err := buffer.ReplaceUTF16(0, 3, "draft"); err != nil {
		t.Fatal(err)
	}
	version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	release := denyWindowsReplacement(t, path)
	defer release()
	wait := replaceRetryWait
	defer func() { replaceRetryWait = wait }()
	waits := 0
	replaceRetryWait = func(delay time.Duration) {
		wait(delay)
		waits++
		if waits == 1 {
			release()
			put(t, path, []byte("out"))
			if err := os.Chtimes(path, stamp, stamp); err != nil {
				t.Fatal(err)
			}
			after, err := os.Stat(path)
			if err != nil || !os.SameFile(before, after) || after.Size() != before.Size() || !after.ModTime().Equal(stamp) {
				t.Fatalf("fixture failed same-file/same-size/same-mtime changed bytes: %v", err)
			}
		}
	}
	err = doc.Save()
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("retry overwrote an external changed-byte winner: err=%v target=%q", err, read(t, path))
	}
	if waits != 1 || string(read(t, path)) != "out" || doc.Path != path || doc.Buffer != buffer ||
		!doc.Dirty() || !doc.SameBaseline(baseline) || buffer.String() != "draft" ||
		buffer.Version() != version || buffer.Selection() != selection || buffer.History() != history {
		t.Fatal("retry conflict changed the target, metadata, baseline or dirty draft")
	}
	if !buffer.Undo() || buffer.String() != "old" || doc.Dirty() || !buffer.Redo() || buffer.String() != "draft" || !doc.Dirty() {
		t.Fatal("retry conflict discarded undo/redo")
	}
}
