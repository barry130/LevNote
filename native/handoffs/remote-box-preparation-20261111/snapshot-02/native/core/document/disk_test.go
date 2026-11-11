package document

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDiskTokenDetectsChangeMissingAndCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(path, []byte("baseline"), 0600); err != nil {
		t.Fatal(err)
	}
	doc, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	token := doc.DiskToken()
	if !token.Valid() || token.Path() != path {
		t.Fatal("token missing baseline", token)
	}
	result, err := token.CheckDisk(context.Background())
	if err != nil || result.Changed || result.Missing || result.Invalid {
		t.Fatal(result, err)
	}
	if !doc.SameBaseline(token) {
		t.Fatal("fresh token was considered stale")
	}

	if err = os.WriteFile(path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err = token.CheckDisk(context.Background())
	if err != nil || !result.Changed || result.Missing || result.Invalid {
		t.Fatal(result, err)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	result, err = token.CheckDisk(context.Background())
	if err != nil || !result.Changed || !result.Missing || result.Invalid {
		t.Fatal(result, err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = token.CheckDisk(cancelled)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation not explicit: %v", err)
	}
}

func TestDiskTokenRejectsSameByteReplacementAndUnchangedSaveRevertStale(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(path, []byte("same"), 0600); err != nil {
		t.Fatal(err)
	}
	doc, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	token := doc.DiskToken()
	replacement := filepath.Join(dir, "replacement")
	if err = os.WriteFile(replacement, []byte("same"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	result, err := token.CheckDisk(context.Background())
	if err != nil || !result.Changed {
		t.Fatal("same-byte replacement accepted", result, err)
	}

	// An unchanged Save must reject, not adopt, externally replaced content.
	doc, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	oldToken := doc.DiskToken()
	if err = os.WriteFile(replacement, []byte("DIFF"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	if err = doc.Save(); !errors.Is(err, ErrConflict) {
		t.Fatalf("unchanged Save adopted changed external content: %v", err)
	}
	if doc.SameBaseline(oldToken) != true {
		t.Fatal("failed Save changed baseline freshness")
	}

	// Reopen the replacement, then prove an unchanged successful Save advances
	// the baseline generation even though bytes and timestamps may remain equal.
	doc, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	oldToken = doc.DiskToken()
	if err = doc.Save(); err != nil {
		t.Fatal(err)
	}
	if doc.SameBaseline(oldToken) {
		t.Fatal("unchanged Save did not invalidate the old token")
	}
	if !doc.SameBaseline(doc.DiskToken()) {
		t.Fatal("fresh Save token was rejected")
	}
	if err = os.WriteFile(replacement, []byte("same"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	if err = doc.Revert(); err != nil {
		t.Fatal(err)
	}
	freshBeforeRevert := doc.DiskToken()
	if !doc.SameBaseline(freshBeforeRevert) {
		t.Fatal("fresh token after open was stale")
	}
	if err = doc.Revert(); err != nil {
		t.Fatal(err)
	}
	if doc.SameBaseline(freshBeforeRevert) {
		t.Fatal("Revert did not invalidate a fresh pre-revert token")
	}
}

func TestDiskTokenDetectsSameMtimeReplacementAndSaveAsInvalidatesBaseline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(path, []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}
	doc, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	token := doc.DiskToken()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, []byte("other"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	changed, statErr := os.Stat(path)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if changed.Size() != info.Size() || !changed.ModTime().Equal(info.ModTime()) {
		t.Fatalf("same-mtime fixture changed size or mtime: before=%#v after=%#v", info, changed)
	}
	result, err := token.CheckDisk(context.Background())
	if err != nil || !result.Changed {
		t.Fatal(result, err)
	}
	if err = doc.Revert(); err != nil {
		t.Fatal(err)
	}
	freshBeforeRevert := doc.DiskToken()
	if doc.SameBaseline(token) {
		t.Fatal("revert did not invalidate the old token")
	}
	if !doc.SameBaseline(freshBeforeRevert) {
		t.Fatal("fresh revert baseline was rejected")
	}
	copyPath := filepath.Join(dir, "copy.txt")
	if err = doc.SaveAs(copyPath); err != nil {
		t.Fatal(err)
	}
	saved := doc.DiskToken()
	if doc.SameBaseline(freshBeforeRevert) || !doc.SameBaseline(saved) {
		t.Fatal("save-as did not invalidate the old token")
	}
	result, err = freshBeforeRevert.CheckDisk(context.Background())
	if err != nil || result.Changed {
		t.Fatal("old baseline check should still describe its immutable path", result, err)
	}
}

func TestDiskTokenForNewDocumentIsInvalidAndReadOnlyHasBaseline(t *testing.T) {
	newDoc := New("draft")
	token := newDoc.DiskToken()
	if token.Valid() {
		t.Fatal("new document acquired a disk baseline")
	}
	result, err := token.CheckDisk(context.Background())
	if err != nil || !result.Invalid || result.Changed || result.Missing {
		t.Fatal(result, err)
	}
	if newDoc.SameBaseline(token) {
		t.Fatal("invalid token was accepted")
	}

	path := filepath.Join(t.TempDir(), "read-only.txt")
	if err = os.WriteFile(path, []byte("read-only"), 0400); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(path, 0600)
	readOnly, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	readOnlyToken := readOnly.DiskToken()
	if !readOnlyToken.Valid() {
		t.Fatal("read-only document lacks disk baseline")
	}
	result, err = readOnlyToken.CheckDisk(context.Background())
	if err != nil || result.Changed {
		t.Fatal(result, err)
	}
}

func TestDiskTokenDetectsUnstableReadDuringHashing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(path, []byte("stable baseline"), 0600); err != nil {
		t.Fatal(err)
	}
	doc, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	token := doc.DiskToken()
	oldHook := diskCheckHook
	diskCheckHook = func() error {
		diskCheckHook = func() error { return nil }
		return os.WriteFile(path, []byte("changed later"), 0600)
	}
	defer func() {
		diskCheckHook = oldHook
	}()
	result, err := token.CheckDisk(context.Background())
	if !errors.Is(err, ErrConflict) || !result.Invalid {
		t.Fatalf("unstable read was not reported as conflict: %#v %v", result, err)
	}
}

func TestOpenLimitedReadEnforcesActualReadLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "limited.txt")
	if err := os.WriteFile(path, []byte("find"), 0600); err != nil {
		t.Fatal(err)
	}
	doc, err := OpenLimitedRead(path, "", 8)
	if err != nil || doc.Buffer.String() != "find" {
		t.Fatalf("limited read rejected a small file: %#v %v", doc, err)
	}
	if err = os.WriteFile(path, []byte("xxxxxxxxx"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = OpenLimitedRead(path, "", 8)
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("limited read did not enforce its limit: %v", err)
	}
}
