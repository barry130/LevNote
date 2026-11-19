package document_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
)

func TestDocumentSaveAsDirectorySyncErrorKeepsPublishedBytesAndRetry(t *testing.T) {
	// This Linux-only real-permission fixture must fail, not skip or pretend
	// to exercise EACCES, if the test process can bypass owner permissions.
	if os.Geteuid() == 0 {
		t.Fatal("directory-read refusal requires the actual non-root process")
	}
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "原文.txt")
	publishDir := filepath.Join(dir, "可写不可读")
	if err := os.Mkdir(publishDir, 0700); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chmod(publishDir, 0700); err != nil {
			t.Errorf("restore directory permissions before fixture cleanup: %v", err)
		}
	}()
	publishedPath, retryPath := filepath.Join(publishDir, "已发布.txt"), filepath.Join(publishDir, "重试另存.txt")
	original := []byte{0xef, 0xbb, 0xbf, 0xe7, 0x94, 0xb2, 0xf0, 0x9f, 0x99, 0x82, 0x0d, 0x0a}
	expected := []byte{0xef, 0xbb, 0xbf, 0xe4, 0xb9, 0x99, 0xf0, 0x9f, 0x99, 0x82, 0x0d, 0x0a}
	if err := os.WriteFile(sourcePath, original, 0600); err != nil {
		t.Fatal(err)
	}
	doc, err := document.Open(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	buffer, baseline := doc.Buffer, doc.DiskToken()
	if buffer.String() != "甲🙂\r\n" || doc.Dirty() || !baseline.Valid() || !doc.SameBaseline(baseline) ||
		doc.Encoding != document.UTF8 || !doc.BOM || doc.IsNew || doc.ReadOnly {
		t.Fatal("normal Open did not establish the independent original saved baseline")
	}
	if err := buffer.SetSelection(0, 1); err != nil {
		t.Fatal(err)
	}
	if err := buffer.ReplaceUTF16(0, 1, "乙"); err != nil {
		t.Fatal(err)
	}
	version, caret, history := buffer.Version(), buffer.Selection(), buffer.History()
	if buffer.String() != "乙🙂\r\n" || caret != (text.Range{Anchor: 1, Head: 1}) ||
		history.UndoDepth != 1 || history.RedoDepth != 0 || !doc.Dirty() || buffer.InGroup() {
		t.Fatal("public edit did not create a genuine dirty draft and undo receipt")
	}
	if err := os.Chmod(publishDir, 0300); err != nil {
		t.Fatal(err)
	}
	directoryInfo, err := os.Stat(publishDir)
	if err != nil || directoryInfo.Mode().Perm() != 0300 {
		t.Fatalf("fixture did not establish owner write+execute without read: %v", err)
	}
	// Create/write/Sync/Close and publish are allowed by write+execute. The
	// subsequent real directory Open fails; published bytes cannot be undone.
	saveErr := doc.SaveAs(publishedPath)
	var pathErr *os.PathError
	if !errors.Is(saveErr, os.ErrPermission) || !errors.As(saveErr, &pathErr) || pathErr.Op != "open" || pathErr.Path != publishDir {
		t.Fatalf("SaveAs did not expose the real post-publication directory-open EACCES: %v", saveErr)
	}
	data, err := os.ReadFile(publishedPath)
	if err != nil || !bytes.Equal(data, expected) {
		t.Fatalf("directory-sync error falsely treated the actual published bytes as unpublished: %x, %v", data, err)
	}
	publishedInfo, err := os.Stat(publishedPath)
	if err != nil || !publishedInfo.Mode().IsRegular() || publishedInfo.Size() != 12 || publishedInfo.Mode().Perm() != 0600 {
		t.Fatalf("the genuinely published file lacks its expected independent encoded receipt: %v", err)
	}
	if doc.Path != sourcePath || doc.IsNew || doc.ReadOnly || doc.Encoding != document.UTF8 || !doc.BOM || !doc.Dirty() ||
		doc.Buffer != buffer || buffer.String() != "乙🙂\r\n" || buffer.Version() != version || buffer.Selection() != caret ||
		buffer.History() != history || buffer.InGroup() || !doc.SameBaseline(baseline) {
		t.Fatal("directory-sync error adopted SaveAs metadata/baseline or lost the public dirty draft receipt")
	}
	data, err = os.ReadFile(sourcePath)
	if err != nil || !bytes.Equal(data, original) {
		t.Fatalf("failed SaveAs changed the original document's independent bytes: %x, %v", data, err)
	}
	if !buffer.Undo() || buffer.String() != "甲🙂\r\n" || buffer.Selection() != (text.Range{Anchor: 0, Head: 1}) ||
		doc.Dirty() || buffer.CanUndo() || !buffer.CanRedo() || !doc.SameBaseline(baseline) {
		t.Fatal("post-publication directory-sync error discarded Undo or advanced the original saved snapshot")
	}
	if !buffer.Redo() || buffer.String() != "乙🙂\r\n" || buffer.Selection() != caret || !doc.Dirty() ||
		buffer.History() != history || !doc.SameBaseline(baseline) {
		t.Fatal("post-publication directory-sync error discarded Redo or dirty retry state")
	}
	version = buffer.Version()
	if err := os.Chmod(publishDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := doc.SaveAs(publishedPath); !errors.Is(err, document.ErrExists) {
		t.Fatalf("retry silently replaced the already-published target instead of refusing it: %v", err)
	}
	data, err = os.ReadFile(publishedPath)
	if err != nil || !bytes.Equal(data, expected) {
		t.Fatalf("same-path retry changed already-published independent bytes: %x, %v", data, err)
	}
	afterRetryInfo, err := os.Stat(publishedPath)
	if err != nil || !os.SameFile(publishedInfo, afterRetryInfo) || afterRetryInfo.Size() != publishedInfo.Size() ||
		afterRetryInfo.Mode().Perm() != publishedInfo.Mode().Perm() || !afterRetryInfo.ModTime().Equal(publishedInfo.ModTime()) {
		t.Fatalf("refused same-path retry replaced the published file's actual receipt: %v", err)
	}
	if doc.Path != sourcePath || doc.IsNew || doc.ReadOnly || doc.Encoding != document.UTF8 || !doc.BOM || !doc.Dirty() ||
		doc.Buffer != buffer || buffer.String() != "乙🙂\r\n" || buffer.Version() != version || buffer.Selection() != caret ||
		buffer.History() != history || buffer.InGroup() || !doc.SameBaseline(baseline) {
		t.Fatal("same-path ErrExists retry changed the dirty document or original baseline")
	}
	if err := doc.SaveAs(retryPath); err != nil {
		t.Fatalf("SaveAs to a genuinely unused name failed after directory permissions were restored: %v", err)
	}
	data, err = os.ReadFile(retryPath)
	if err != nil || !bytes.Equal(data, expected) {
		t.Fatalf("successful unused-name retry did not publish the independent draft bytes: %x, %v", data, err)
	}
	if doc.Path != retryPath || doc.IsNew || doc.ReadOnly || doc.Encoding != document.UTF8 || !doc.BOM || doc.Dirty() ||
		doc.Buffer != buffer || buffer.String() != "乙🙂\r\n" || buffer.Version() != version || buffer.Selection() != caret ||
		buffer.History() != history || buffer.InGroup() || doc.SameBaseline(baseline) || !doc.DiskToken().Valid() ||
		!doc.SameBaseline(doc.DiskToken()) {
		t.Fatal("successful unused-name retry lost the draft receipt or failed to adopt a fresh clean baseline")
	}
	if changed, err := doc.CheckExternalChange(); err != nil || changed {
		t.Fatalf("successful retry did not establish a usable verified baseline: changed=%v err=%v", changed, err)
	}
	data, err = os.ReadFile(sourcePath)
	if err != nil || !bytes.Equal(data, original) {
		t.Fatalf("unused-name retry changed the original independent source bytes: %x, %v", data, err)
	}
	data, err = os.ReadFile(publishedPath)
	if err != nil || !bytes.Equal(data, expected) {
		t.Fatalf("unused-name retry changed the earlier genuinely published bytes: %x, %v", data, err)
	}
}
