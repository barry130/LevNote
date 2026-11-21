package document

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func preparedLiteral(t *testing.T, target, value string) *PreparedPrivateState {
	t.Helper()
	p, err := PreparePrivateStateTo(target, func(w io.Writer) error {
		_, err := io.WriteString(w, value)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := p.Abort(); err != nil {
			t.Error(err)
		}
	})
	return p
}

func TestPreparedPrivateStateCommitAbortAndCreate(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "create", true: "replace"}[existing], func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "session.json")
			if existing {
				if err := os.WriteFile(target, []byte("original recovery"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			p := preparedLiteral(t, target, "prepared recovery")
			before, err := os.ReadFile(target)
			if existing && (err != nil || string(before) != "original recovery") || !existing && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("Prepare published target: %q %v", before, err)
			}
			if err := p.Commit(nil); err != nil {
				t.Fatal(err)
			}
			if err := p.Abort(); err != nil {
				t.Fatal(err)
			}
			if err := p.Abort(); err != nil {
				t.Fatal(err)
			}
			if got, err := os.ReadFile(target); err != nil || string(got) != "prepared recovery" {
				t.Fatalf("Commit/Abort changed published bytes: %q %v", got, err)
			}
			if files, err := filepath.Glob(filepath.Join(dir, ".levnote-save-*")); err != nil || len(files) != 0 {
				t.Fatalf("temporary files leaked: %v %v", files, err)
			}
		})
	}
}

func TestPreparedPrivateStateAbortAndPrecommitFailureKeepOriginal(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "session.json")
	original := []byte("original recovery")
	if err := os.WriteFile(target, original, 0600); err != nil {
		t.Fatal(err)
	}
	p := preparedLiteral(t, target, "candidate discard")
	failure := errors.New("owner changed before commit")
	err := p.Commit(func() error { return failure })
	var publication *StatePublicationError
	if !errors.As(err, &publication) || publication.Committed || !errors.Is(err, failure) {
		t.Fatalf("precommit outcome=%#v err=%v", publication, err)
	}
	if err := p.Abort(); err != nil {
		t.Fatal(err)
	}
	if err := p.Commit(nil); err == nil {
		t.Fatal("aborted state was published")
	}
	if got, err := os.ReadFile(target); err != nil || !bytes.Equal(got, original) {
		t.Fatalf("failed candidate overwrote original: %q %v", got, err)
	}
	p, err = PreparePrivateStateTo(target, func(w io.Writer) error {
		_, _ = io.WriteString(w, "partial")
		return failure
	})
	if p != nil || !errors.Is(err, failure) {
		t.Fatalf("writer failure=%v p=%v", err, p)
	}
	if files, _ := filepath.Glob(filepath.Join(dir, ".levnote-save-*")); len(files) != 0 {
		t.Fatalf("writer failure leaked temps: %v", files)
	}
}

func TestPreparedPrivateStatePostcommitErrorIsNotRollback(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "session.json")
	if err := os.WriteFile(target, []byte("old recovery"), 0600); err != nil {
		t.Fatal(err)
	}
	p := preparedLiteral(t, target, "new committed recovery")
	originalSync := atomicSyncDirectory
	failure := errors.New("injected directory sync failure")
	atomicSyncDirectory = func(string) error { return failure }
	t.Cleanup(func() { atomicSyncDirectory = originalSync })
	err := p.Commit(nil)
	var publication *StatePublicationError
	if !errors.As(err, &publication) || !publication.Committed || !errors.Is(err, failure) {
		t.Fatalf("postcommit outcome=%#v err=%v", publication, err)
	}
	if err := p.Abort(); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "new committed recovery" {
		t.Fatalf("postcommit error rolled back or removed target: %q %v", got, err)
	}
}

func TestPreparedPrivateStateSyncAndCloseFailureDoNotPublish(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "session.json")
	if err := os.WriteFile(target, []byte("old recovery"), 0600); err != nil {
		t.Fatal(err)
	}
	prepared, err := PreparePrivateStateTo(target, func(w io.Writer) error {
		limited, ok := w.(*privateStateLimitWriter)
		if !ok {
			t.Fatal("writer did not expose the bounded temporary writer")
		}
		file, ok := limited.writer.(*os.File)
		if !ok {
			t.Fatal("bounded writer does not wrap the real temporary file")
		}
		if _, err := io.WriteString(file, "candidate"); err != nil {
			return err
		}
		return file.Close() // A real Sync/Close error follows the successful write.
	})
	if err == nil || prepared != nil {
		t.Fatalf("closed temporary file was accepted: %v %v", prepared, err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "old recovery" {
		t.Fatalf("failed Sync/Close published bytes: %q %v", got, err)
	}
	if files, _ := filepath.Glob(filepath.Join(dir, ".levnote-save-*")); len(files) != 0 {
		t.Fatalf("failed Sync/Close leaked temps: %v", files)
	}
}

func TestPreparedPrivateStateRefusesChangedTarget(t *testing.T) {
	target := filepath.Join(t.TempDir(), "session.json")
	p := preparedLiteral(t, target, "candidate")
	if err := os.WriteFile(target, []byte("new external authority"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := p.Commit(nil); err == nil {
		t.Fatal("concurrently appearing target was overwritten")
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "new external authority" {
		t.Fatalf("external target lost: %q %v", got, err)
	}
}

func privateStateTestInfo(t *testing.T, path string) os.FileInfo {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	info, statErr := file.Stat()
	closeErr := file.Close()
	if statErr != nil || closeErr != nil {
		t.Fatal(errors.Join(statErr, closeErr))
	}
	return info
}

func TestPreparedPrivateStateRejectsSameFileChangedBytes(t *testing.T) {
	for _, targetFile := range []bool{true, false} {
		t.Run(map[bool]string{true: "target", false: "temp"}[targetFile], func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "session.json")
			if err := os.WriteFile(target, []byte("old recovery"), 0600); err != nil {
				t.Fatal(err)
			}
			p := preparedLiteral(t, target, "new recovery")
			path, replacement := target, "bad recovery"
			if !targetFile {
				path, replacement = p.tempPath, "bad recovery"
			}
			before := privateStateTestInfo(t, path)
			if err := os.WriteFile(path, []byte(replacement), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(path, before.ModTime(), before.ModTime()); err != nil {
				t.Fatal(err)
			}
			after := privateStateTestInfo(t, path)
			if !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
				t.Fatalf("fixture did not preserve file identity/size/mtime: before=%v after=%v", before, after)
			}
			err := p.Commit(nil)
			var publication *StatePublicationError
			if !errors.As(err, &publication) || publication.Committed {
				t.Fatalf("same-inode changed bytes were published: %v", err)
			}
			if targetFile {
				if got, readErr := os.ReadFile(target); readErr != nil || string(got) != replacement {
					t.Fatalf("changed target was overwritten: %q %v", got, readErr)
				}
			}
			if err := p.Abort(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPreparedPrivateStateRejectsNewTargetFileIDWithSameMetadata(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "session.json")
	if err := os.WriteFile(target, []byte("old recovery"), 0600); err != nil {
		t.Fatal(err)
	}
	p := preparedLiteral(t, target, "new recovery")
	before := privateStateTestInfo(t, target)
	old := target + ".old"
	if err := os.Rename(target, old); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old recovery"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(target, before.ModTime(), before.ModTime()); err != nil {
		t.Fatal(err)
	}
	after := privateStateTestInfo(t, target)
	if os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatalf("fixture did not create a new file identity with matching metadata: before=%v after=%v", before, after)
	}
	err := p.Commit(nil)
	var publication *StatePublicationError
	if !errors.As(err, &publication) || publication.Committed {
		t.Fatalf("new target file identity was overwritten: %v", err)
	}
	if got, readErr := os.ReadFile(target); readErr != nil || string(got) != "old recovery" {
		t.Fatalf("external target was lost: %q %v", got, readErr)
	}
	if err := p.Abort(); err != nil {
		t.Fatal(err)
	}
}

func TestPreparedPrivateStateRejectsNewDirectoryFileID(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "state")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "session.json")
	if err := os.WriteFile(target, []byte("old recovery"), 0600); err != nil {
		t.Fatal(err)
	}
	p := preparedLiteral(t, target, "new recovery")
	oldDirInfo := p.dirInfo
	oldDir := dir + ".old"
	if err := os.Rename(dir, oldDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old recovery"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(target, p.targetInfo.ModTime(), p.targetInfo.ModTime()); err != nil {
		t.Fatal(err)
	}
	oldTemp := filepath.Join(oldDir, filepath.Base(p.tempPath))
	if err := os.Link(oldTemp, p.tempPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(dir, oldDirInfo.ModTime(), oldDirInfo.ModTime()); err != nil {
		t.Fatal(err)
	}
	newDirInfo, err := inspectPrivateStateDirectory(dir)
	if err != nil || os.SameFile(oldDirInfo, newDirInfo) || !oldDirInfo.ModTime().Equal(newDirInfo.ModTime()) {
		t.Fatalf("fixture did not create a new directory identity with matching mtime: %v", err)
	}
	err = p.Commit(nil)
	var publication *StatePublicationError
	if !errors.As(err, &publication) || publication.Committed {
		t.Fatalf("new directory identity was accepted: %v", err)
	}
	if got, readErr := os.ReadFile(target); readErr != nil || string(got) != "old recovery" {
		t.Fatalf("replacement directory target was changed: %q %v", got, readErr)
	}
	if err := p.Abort(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(oldTemp); err != nil {
		t.Fatal(err)
	}
}

func TestPreparedPrivateStateRefusesOversizedExistingState(t *testing.T) {
	target := filepath.Join(t.TempDir(), "session.json")
	if err := os.WriteFile(target, []byte("small placeholder"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(target, maxPrivateStateBytes+1); err != nil {
		t.Fatal(err)
	}
	if _, err := PreparePrivateStateTo(target, func(w io.Writer) error {
		_, err := io.WriteString(w, "candidate")
		return err
	}); !errors.Is(err, errPrivateStateLimit) {
		t.Fatalf("oversized authority was not refused explicitly: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil || info.Size() != maxPrivateStateBytes+1 {
		t.Fatalf("oversized target changed: %v %v", info, err)
	}
	if files, globErr := filepath.Glob(filepath.Join(filepath.Dir(target), ".levnote-save-*")); globErr != nil || len(files) != 0 {
		t.Fatalf("oversized prepare leaked a candidate: %v %v", files, globErr)
	}
}

func TestWritePrivateStateToClassifiesPublishedTempCleanupFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "first-session.json")
	var tempPath string
	originalSync := atomicSyncDirectory
	atomicSyncDirectory = func(string) error {
		if tempPath == "" {
			t.Fatal("writer did not expose its real temp identity")
		}
		if err := os.Remove(tempPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		if err := os.WriteFile(tempPath, []byte("replacement must not be deleted"), 0600); err != nil {
			t.Fatal(err)
		}
		return nil
	}
	defer func() { atomicSyncDirectory = originalSync }()

	err := WritePrivateStateTo(target, func(w io.Writer) error {
		limited, ok := w.(*privateStateLimitWriter)
		if !ok {
			t.Fatal("writer did not expose the bounded temporary writer")
		}
		file, ok := limited.writer.(*os.File)
		if !ok {
			t.Fatal("bounded writer does not wrap the real temporary file")
		}
		tempPath = file.Name()
		_, err := io.WriteString(w, "new published state")
		return err
	})
	var publication *StatePublicationError
	if !errors.As(err, &publication) || !publication.Committed || publication.Err != nil ||
		publication.CleanupErr == nil || !strings.Contains(err.Error(), "temporary cleanup failed") ||
		strings.Contains(err.Error(), "directory durability is unconfirmed") {
		t.Fatalf("published cleanup error was not accurately classified: %#v %v", publication, err)
	}
	if got, readErr := os.ReadFile(target); readErr != nil || string(got) != "new published state" {
		t.Fatalf("cleanup error removed or rolled back the published target: %q %v", got, readErr)
	}
	files, globErr := filepath.Glob(filepath.Join(dir, ".levnote-save-*"))
	if globErr != nil || len(files) != 1 {
		t.Fatalf("unverified replacement was removed or not retained for inspection: %v %v", files, globErr)
	}
	if err := os.Remove(files[0]); err != nil {
		t.Fatal(err)
	}
}

func TestRetryPrivateStateTargetReadRechecksIdentityAndOwner(t *testing.T) {
	readBlocked := errors.New("target direct open: sharing violation")
	var identityReads, ownerChecks, waits int
	blocked := true
	err := retryPrivateStateTargetRead(
		func() error {
			identityReads++ // Represents the complete original dir/temp/target guard.
			if blocked {
				return readBlocked
			}
			return nil
		},
		func() error { ownerChecks++; return nil },
		func(err error) bool { return errors.Is(err, readBlocked) },
		func() { waits++; blocked = false },
		5,
	)
	if err != nil || waits != 1 || identityReads != 2 || ownerChecks != 1 {
		t.Fatalf("transient read lock result err=%v reads=%d waits=%d owners=%d", err, identityReads, waits, ownerChecks)
	}
}

func TestRetryPrivateStateTargetReadFailsClosedAndRechecksOwner(t *testing.T) {
	readBlocked := errors.New("target direct open: sharing violation")
	ownerChanged := errors.New("prepared owner changed")
	var identityReads, ownerChecks, waits int
	err := retryPrivateStateTargetRead(
		func() error { identityReads++; return readBlocked },
		func() error {
			ownerChecks++
			if ownerChecks == 2 {
				return ownerChanged
			}
			return nil
		},
		func(err error) bool { return errors.Is(err, readBlocked) },
		func() { waits++ },
		5,
	)
	if !errors.Is(err, ownerChanged) || !errors.Is(err, readBlocked) || waits != 2 || identityReads != 3 || ownerChecks != 2 {
		t.Fatalf("owner change during identity retry err=%v reads=%d waits=%d owners=%d", err, identityReads, waits, ownerChecks)
	}
}

func TestRetryPrivateStateTargetReadBoundsWaitsAndRejectsOtherErrors(t *testing.T) {
	readBlocked := errors.New("target direct open: sharing violation")
	metadataFailure := errors.New("target metadata failure")
	t.Run("bounded-sharing-lock", func(t *testing.T) {
		var reads, owners, waits int
		err := retryPrivateStateTargetRead(
			func() error { reads++; return readBlocked },
			func() error { owners++; return nil },
			func(err error) bool { return errors.Is(err, readBlocked) },
			func() { waits++ },
			5,
		)
		if !errors.Is(err, readBlocked) || reads != 6 || waits != 5 || owners != 5 {
			t.Fatalf("permanent sharing lock exceeded bounds err=%v reads=%d waits=%d owners=%d", err, reads, waits, owners)
		}
	})
	t.Run("metadata-error-is-not-retried", func(t *testing.T) {
		var reads, owners, waits int
		err := retryPrivateStateTargetRead(
			func() error { reads++; return metadataFailure },
			func() error { owners++; return nil },
			func(err error) bool { return errors.Is(err, readBlocked) },
			func() { waits++ },
			5,
		)
		if !errors.Is(err, metadataFailure) || reads != 1 || waits != 0 || owners != 0 {
			t.Fatalf("metadata failure was retried: err=%v reads=%d waits=%d owners=%d", err, reads, waits, owners)
		}
	})
}
