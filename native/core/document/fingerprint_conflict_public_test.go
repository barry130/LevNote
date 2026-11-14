package document_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
)

func TestDocumentSameSizeSameMtimeConflictPreservesUndoAndRetry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "冲突.txt")
	original := []byte{0xef, 0xbb, 0xbf, 0xe7, 0x94, 0xb2, 0xf0, 0x9f, 0x99, 0x82, 0x0d, 0x0a}
	external := []byte{0xef, 0xbb, 0xbf, 0xe4, 0xb8, 0x99, 0xf0, 0x9f, 0x99, 0x82, 0x0d, 0x0a}
	expected := []byte{0xef, 0xbb, 0xbf, 0xe4, 0xb9, 0x99, 0xf0, 0x9f, 0x99, 0x82, 0x0d, 0x0a}
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	stamp := time.Unix(1600000000, 0)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := document.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	buffer, baseline := doc.Buffer, doc.DiskToken()
	if doc.Buffer.String() != "甲🙂\r\n" || doc.Encoding != document.UTF8 || !doc.BOM || doc.Dirty() ||
		doc.IsNew || doc.ReadOnly || !baseline.Valid() || !doc.SameBaseline(baseline) {
		t.Fatal("normal Open did not preserve the literal UTF-8 BOM/CRLF baseline")
	}
	if err := buffer.SetSelection(0, 1); err != nil {
		t.Fatal(err)
	}
	if err := buffer.ReplaceUTF16(0, 1, "乙"); err != nil {
		t.Fatal(err)
	}
	draftVersion, draftSelection, draftHistory := buffer.Version(), buffer.Selection(), buffer.History()
	if buffer.String() != "乙🙂\r\n" || draftSelection != (text.Range{Anchor: 1, Head: 1}) ||
		draftHistory.UndoDepth != 1 || draftHistory.RedoDepth != 0 || !buffer.CanUndo() || buffer.CanRedo() || !doc.Dirty() {
		t.Fatal("public edit did not establish a real dirty draft and Undo transaction")
	}
	assertMetadata := func(stage string) {
		t.Helper()
		if doc.Path != path || doc.Buffer != buffer || doc.Encoding != document.UTF8 || !doc.BOM ||
			doc.ReadOnly || doc.IsNew || !doc.SameBaseline(baseline) {
			t.Fatalf("%s changed the public document metadata, Buffer owner or saved baseline", stage)
		}
	}
	assertDraft := func(stage string) {
		t.Helper()
		assertMetadata(stage)
		if buffer.String() != "乙🙂\r\n" || buffer.Version() != draftVersion || buffer.Selection() != draftSelection ||
			buffer.History() != draftHistory || buffer.InGroup() || !doc.Dirty() || !buffer.CanUndo() || buffer.CanRedo() {
			t.Fatalf("%s changed the unsaved draft, version, caret, history or Dirty", stage)
		}
	}
	// A real external writer changes the encoded bytes, not the file identity,
	// length, mode or observable mtime. A Stat-only comparison cannot catch it.
	if err := os.WriteFile(path, external, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	afterExternal, err := os.Stat(path)
	if err != nil || !os.SameFile(before, afterExternal) || before.Size() != afterExternal.Size() || afterExternal.Size() != 12 ||
		before.Mode() != afterExternal.Mode() || !before.ModTime().Equal(afterExternal.ModTime()) {
		t.Fatalf("fixture did not establish actual SameFile/same-size/same-mode/same-mtime content drift: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, external) {
		t.Fatalf("external bytes do not match the independent encoded literal: %x, %v", data, err)
	}
	if changed, err := doc.CheckExternalChange(); err != nil || !changed {
		t.Fatalf("same-file same-size same-mtime content drift was missed: changed=%v err=%v", changed, err)
	}
	assertDraft("CheckExternalChange")
	if err := doc.Save(); !errors.Is(err, document.ErrConflict) {
		t.Fatalf("Save did not visibly refuse the external content conflict: %v", err)
	}
	assertDraft("refused Save")
	afterRefusal, err := os.Stat(path)
	if err != nil || !os.SameFile(afterExternal, afterRefusal) || afterRefusal.Size() != 12 ||
		afterExternal.Mode() != afterRefusal.Mode() || !afterExternal.ModTime().Equal(afterRefusal.ModTime()) {
		t.Fatalf("refused Save changed the actual external file receipt: %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil || !bytes.Equal(data, external) {
		t.Fatalf("refused Save overwrote the external encoded bytes: %x, %v", data, err)
	}
	if !buffer.Undo() || buffer.String() != "甲🙂\r\n" || buffer.Selection() != (text.Range{Anchor: 0, Head: 1}) ||
		buffer.CanUndo() || !buffer.CanRedo() || buffer.InGroup() || doc.Dirty() {
		t.Fatal("conflict refusal damaged genuine Undo or the unchanged saved snapshot")
	}
	assertMetadata("Undo")
	if changed, err := doc.CheckExternalChange(); err != nil || !changed {
		t.Fatalf("Undo falsely adopted the external file as a saved baseline: changed=%v err=%v", changed, err)
	}
	if !buffer.Redo() || buffer.String() != "乙🙂\r\n" || buffer.Selection() != draftSelection ||
		buffer.History() != draftHistory || buffer.InGroup() || !doc.Dirty() || !buffer.CanUndo() || buffer.CanRedo() {
		t.Fatal("conflict refusal damaged genuine Redo, caret or dirty draft")
	}
	assertMetadata("Redo")
	retryVersion := buffer.Version()
	// Restoring the real original file contents permits retry without Reload,
	// private baseline injection, history reset or loss of the in-memory draft.
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	afterRestore, err := os.Stat(path)
	if err != nil || !os.SameFile(before, afterRestore) || afterRestore.Size() != 12 ||
		before.Mode() != afterRestore.Mode() || !before.ModTime().Equal(afterRestore.ModTime()) {
		t.Fatalf("fixture did not restore the actual same-file encoded baseline receipt: %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil || !bytes.Equal(data, original) {
		t.Fatalf("restore did not retain the independent original encoded bytes: %x, %v", data, err)
	}
	if changed, err := doc.CheckExternalChange(); err != nil || changed {
		t.Fatalf("restored original contents did not permit retry: changed=%v err=%v", changed, err)
	}
	assertMetadata("restored CheckExternalChange")
	if err := doc.Save(); err != nil {
		t.Fatalf("Save retry lost the preserved user draft: %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil || !bytes.Equal(data, expected) {
		t.Fatalf("retry did not publish the independent UTF-8 BOM/CRLF output: %x, %v", data, err)
	}
	if doc.Path != path || doc.Buffer != buffer || buffer.String() != "乙🙂\r\n" || buffer.Version() != retryVersion ||
		buffer.Selection() != draftSelection || buffer.History() != draftHistory || buffer.InGroup() || doc.Dirty() ||
		doc.Encoding != document.UTF8 || !doc.BOM || doc.ReadOnly || doc.IsNew || doc.SameBaseline(baseline) ||
		!doc.DiskToken().Valid() || !doc.SameBaseline(doc.DiskToken()) {
		t.Fatal("successful retry changed the draft receipt or failed to establish a fresh clean baseline")
	}
	if changed, err := doc.CheckExternalChange(); err != nil || changed {
		t.Fatalf("successful retry left an unusable disk baseline: changed=%v err=%v", changed, err)
	}
}
