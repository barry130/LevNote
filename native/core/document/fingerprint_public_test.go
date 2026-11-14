package document_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levnote.local/native/core/document"
)

func TestDocumentMtimeOnlyTouchKeepsCheckAndSaveAvailable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "原文.txt")
	original := []byte{0xef, 0xbb, 0xbf, 0xe7, 0x94, 0xb2, 0xf0, 0x9f, 0x99, 0x82, 0x0d, 0x0a}
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
	if doc.Buffer.String() != "甲🙂\r\n" || doc.Encoding != document.UTF8 || !doc.BOM || doc.Dirty() {
		t.Fatal("normal Open did not preserve the literal UTF-8 BOM/CRLF fixture")
	}
	touch := time.Unix(1600086400, 0)
	if err := os.Chtimes(path, touch, touch); err != nil {
		t.Fatal(err)
	}
	afterTouch, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, afterTouch) || before.Size() != afterTouch.Size() || afterTouch.Size() != 12 ||
		before.Mode() != afterTouch.Mode() || before.ModTime().Equal(afterTouch.ModTime()) || !afterTouch.ModTime().Equal(touch) {
		t.Fatal("fixture did not perform a real same-file, same-size, same-mode mtime-only touch")
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, original) {
		t.Fatalf("touch changed the independent literal encoded bytes: %x, %v", data, err)
	}
	if changed, err := doc.CheckExternalChange(); err != nil || changed {
		t.Fatalf("same-file same-bytes touch was reported as a conflict: changed=%v err=%v", changed, err)
	}
	if err := doc.Save(); err != nil {
		t.Fatalf("untouched Save was blocked by mtime-only drift: %v", err)
	}
	afterNoOpSave, err := os.Stat(path)
	if err != nil || !os.SameFile(afterTouch, afterNoOpSave) || !afterNoOpSave.ModTime().Equal(touch) {
		t.Fatalf("untouched Save replaced the real file or its touched timestamp: %v", err)
	}
	if err := doc.Buffer.ReplaceUTF16(0, 1, "乙"); err != nil {
		t.Fatal(err)
	}
	if !doc.Dirty() {
		t.Fatal("the public Buffer edit was not dirty before Save")
	}
	if err := doc.Save(); err != nil {
		t.Fatalf("edited Save was blocked by mtime-only drift: %v", err)
	}
	expected := []byte{0xef, 0xbb, 0xbf, 0xe4, 0xb9, 0x99, 0xf0, 0x9f, 0x99, 0x82, 0x0d, 0x0a}
	data, err = os.ReadFile(path)
	if err != nil || !bytes.Equal(data, expected) {
		t.Fatalf("Save did not write the literal UTF-8 BOM/CRLF output: %x, %v", data, err)
	}
	if doc.Buffer.String() != "乙🙂\r\n" || doc.Dirty() || doc.Encoding != document.UTF8 || !doc.BOM || doc.IsNew || doc.ReadOnly {
		t.Fatal("successful public Save lost content, metadata or clean state")
	}
	if changed, err := doc.CheckExternalChange(); err != nil || changed {
		t.Fatalf("the verified public Save did not establish a usable disk baseline: changed=%v err=%v", changed, err)
	}
}
