package document_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
)

func TestDocumentUTF16SaveAndSaveAsUseActualEncodedLength(t *testing.T) {
	dir := t.TempDir()
	path, copyPath := filepath.Join(dir, "原文.txt"), filepath.Join(dir, "另存.txt")
	// Independent UTF-16LE BOM literals: 14 disk bytes, versus 10 decoded
	// UTF-8 bytes or 6 UTF-16 units. The edited SaveAs output has 16 disk bytes.
	original := []byte{0xff, 0xfe, 0x32, 0x75, 0x3d, 0xd8, 0x42, 0xde, 0x0d, 0x00, 0x0a, 0x00, 0x5a, 0x00}
	saved := []byte{0xff, 0xfe, 0x59, 0x4e, 0x3d, 0xd8, 0x42, 0xde, 0x0d, 0x00, 0x0a, 0x00, 0x5a, 0x00}
	copied := []byte{0xff, 0xfe, 0x59, 0x4e, 0x3d, 0xd8, 0x42, 0xde, 0x0d, 0x00, 0x0a, 0x00, 0x5a, 0x00, 0x5a, 0x00}
	if err := os.WriteFile(path, original, 0600); err != nil {
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
	if buffer.String() != "甲🙂\r\nZ" || doc.Encoding != document.UTF16LE || !doc.BOM || doc.Dirty() ||
		doc.ReadOnly || doc.IsNew || !baseline.Valid() || !doc.SameBaseline(baseline) {
		t.Fatal("normal Open did not preserve the independent UTF-16LE BOM/CRLF literal")
	}
	if err := buffer.SetSelection(0, 1); err != nil {
		t.Fatal(err)
	}
	if err := buffer.ReplaceUTF16(0, 1, "乙"); err != nil {
		t.Fatal(err)
	}
	version, caret, history := buffer.Version(), buffer.Selection(), buffer.History()
	if buffer.String() != "乙🙂\r\nZ" || caret != (text.Range{Anchor: 1, Head: 1}) ||
		history.UndoDepth != 1 || history.RedoDepth != 0 || !doc.Dirty() {
		t.Fatal("public edit did not create a genuine dirty UTF-16 draft and history")
	}
	if changed, err := doc.CheckExternalChange(); err != nil || changed {
		t.Fatalf("unchanged 14-byte UTF-16 baseline was confused with decoded length: changed=%v err=%v", changed, err)
	}
	if err := doc.Save(); err != nil {
		t.Fatalf("UTF-16 Save failed to verify the actual 14 encoded bytes: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, saved) {
		t.Fatalf("Save did not preserve the independent UTF-16LE BOM/CRLF output: %x, %v", data, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() != 14 || info.Mode().Perm() != before.Mode().Perm() {
		t.Fatalf("Save lost the actual encoded size or original permissions: %v", err)
	}
	if doc.Path != path || doc.Buffer != buffer || buffer.String() != "乙🙂\r\nZ" || buffer.Version() != version ||
		buffer.Selection() != caret || buffer.History() != history || buffer.InGroup() || doc.Dirty() ||
		doc.Encoding != document.UTF16LE || !doc.BOM || doc.ReadOnly || doc.IsNew || doc.SameBaseline(baseline) ||
		!doc.SameBaseline(doc.DiskToken()) {
		t.Fatal("verified UTF-16 Save changed the draft receipt or failed to establish a clean baseline")
	}
	if changed, err := doc.CheckExternalChange(); err != nil || changed {
		t.Fatalf("verified 14-byte Save did not establish a usable disk baseline: changed=%v err=%v", changed, err)
	}
	beforeCopy := doc.DiskToken()
	if err := buffer.ReplaceUTF16(5, 6, "ZZ"); err != nil {
		t.Fatal(err)
	}
	version, caret, history = buffer.Version(), buffer.Selection(), buffer.History()
	if buffer.String() != "乙🙂\r\nZZ" || caret != (text.Range{Anchor: 7, Head: 7}) ||
		history.UndoDepth != 2 || history.RedoDepth != 0 || !doc.Dirty() {
		t.Fatal("second public edit did not establish a larger encoded draft")
	}
	if err := doc.SaveAs(copyPath); err != nil {
		t.Fatalf("UTF-16 SaveAs failed to verify the actual 16 encoded bytes: %v", err)
	}
	data, err = os.ReadFile(copyPath)
	if err != nil || !bytes.Equal(data, copied) {
		t.Fatalf("SaveAs did not publish the independent larger UTF-16LE BOM/CRLF literal: %x, %v", data, err)
	}
	info, err = os.Stat(copyPath)
	if err != nil || info.Size() != 16 {
		t.Fatalf("SaveAs did not publish the actual larger encoded length: %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil || !bytes.Equal(data, saved) {
		t.Fatalf("SaveAs changed the original document's verified encoded bytes: %x, %v", data, err)
	}
	if doc.Path != copyPath || doc.Buffer != buffer || buffer.String() != "乙🙂\r\nZZ" || buffer.Version() != version ||
		buffer.Selection() != caret || buffer.History() != history || buffer.InGroup() || doc.Dirty() ||
		doc.Encoding != document.UTF16LE || !doc.BOM || doc.ReadOnly || doc.IsNew || doc.SameBaseline(beforeCopy) ||
		!doc.DiskToken().Valid() || !doc.SameBaseline(doc.DiskToken()) {
		t.Fatal("verified SaveAs lost the public metadata, draft receipt or fresh clean baseline")
	}
	if changed, err := doc.CheckExternalChange(); err != nil || changed {
		t.Fatalf("verified 16-byte SaveAs left an unusable disk baseline: changed=%v err=%v", changed, err)
	}
}
