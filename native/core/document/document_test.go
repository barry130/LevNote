package document

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func put(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}
func read(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestEncodingRoundTripAndUntouchedSave(t *testing.T) {
	for _, enc := range []Encoding{UTF8, UTF16LE, UTF16BE} {
		for _, bom := range []bool{false, true} {
			if enc != UTF8 && !bom {
				continue
			}
			t.Run(string(enc)+map[bool]string{true: "-bom", false: "-plain"}[bom], func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "原文.txt")
				original, err := encode("甲🙂\r\n乙\n丙\r末尾", enc, bom)
				if err != nil {
					t.Fatal(err)
				}
				put(t, path, original)
				stamp := time.Unix(1600000000, 0)
				if err = os.Chtimes(path, stamp, stamp); err != nil {
					t.Fatal(err)
				}
				doc, err := Open(path)
				if err != nil {
					t.Fatal(err)
				}
				if doc.Encoding != enc || doc.BOM != bom || doc.Dirty() {
					t.Fatal(doc.Encoding, doc.BOM, doc.Dirty())
				}
				if err = doc.Save(); err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(read(t, path), original) {
					t.Fatal("untouched save rewrote bytes")
				}
				info, _ := os.Stat(path)
				if !info.ModTime().Equal(stamp) {
					t.Fatal("untouched save changed timestamp")
				}
				if err = doc.Buffer.ReplaceUTF16(0, 1, "新"); err != nil {
					t.Fatal(err)
				}
				if !doc.Dirty() {
					t.Fatal("edit did not dirty")
				}
				doc.Buffer.Undo()
				if doc.Dirty() {
					t.Fatal("undo did not restore clean state")
				}
				doc.Buffer.Redo()
				if err = doc.Save(); err != nil {
					t.Fatal(err)
				}
				if doc.Dirty() {
					t.Fatal("save still dirty")
				}
				expected, _ := encode("新🙂\r\n乙\n丙\r末尾", enc, bom)
				if !bytes.Equal(read(t, path), expected) {
					t.Fatal("encoding/ending changed")
				}
				reopened, err := Open(path)
				if err != nil || reopened.Buffer.String() != doc.Buffer.String() {
					t.Fatal("reopen", err)
				}
			})
		}
	}
}

func TestConflictIncludesSameTimestampAndMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.md")
	put(t, path, []byte("original"))
	doc, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	put(t, path, []byte("external"))
	_ = os.Chtimes(path, info.ModTime(), info.ModTime())
	_ = doc.Buffer.ReplaceUTF16(0, 8, "our edit")
	if changed, err := doc.CheckExternalChange(); err != nil || !changed {
		t.Fatal(changed, err)
	}
	if err = doc.Save(); !errors.Is(err, ErrConflict) {
		t.Fatal("overwrote external edit", err)
	}
	if string(read(t, path)) != "external" {
		t.Fatal("conflicting content lost")
	}
	if err = doc.Reload(); !errors.Is(err, ErrDirty) {
		t.Fatal(err)
	}
	buf := doc.Buffer
	if err = doc.Revert(); err != nil {
		t.Fatal(err)
	}
	if doc.Buffer != buf || doc.Buffer.String() != "external" || doc.Dirty() {
		t.Fatal("revert")
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if changed, err := doc.CheckExternalChange(); err != nil || !changed {
		t.Fatal(changed, err)
	}
	if err = doc.Save(); !errors.Is(err, ErrConflict) {
		t.Fatal("recreated deleted file", err)
	}
}

func TestSaveAsNeverOverwritesAndNewEmptyIsUnsaved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "copy.txt")
	doc := New("")
	if !doc.Dirty() || !doc.IsNew || !errors.Is(doc.Save(), ErrNoPath) {
		t.Fatal("new document claims saved")
	}
	put(t, path, []byte("someone else's file"))
	if err := doc.SaveAs(path); !errors.Is(err, ErrExists) {
		t.Fatal(err)
	}
	if string(read(t, path)) != "someone else's file" || doc.Path != "" || !doc.IsNew {
		t.Fatal("save as side effect")
	}
	path = filepath.Join(dir, "new.txt")
	if err := doc.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	if doc.Dirty() || doc.IsNew || len(read(t, path)) != 0 {
		t.Fatal("new save")
	}
	// The publisher must also reject a target created after SaveAs's preflight.
	tmp := filepath.Join(dir, "ready.tmp")
	put(t, tmp, []byte("replacement"))
	if err := publishNew(tmp, path); err == nil {
		t.Fatal("atomic publisher overwrote target")
	}
	if len(read(t, path)) != 0 {
		t.Fatal("target changed")
	}
}

func TestWriteNewFileNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "export.html")
	if err := WriteNewFile(path, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := WriteNewFile(path, []byte("second")); !errors.Is(err, ErrExists) {
		t.Fatal(err)
	}
	if string(read(t, path)) != "first" {
		t.Fatal("existing export was overwritten")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatal("temporary file leaked", entries)
	}
}

func TestReadOnlyAndCompositionNeverWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "read-only.txt")
	put(t, path, []byte("original"))
	if err := os.Chmod(path, 0400); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(path, 0600)
	doc, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if !doc.ReadOnly {
		t.Fatal("readonly not detected")
	}
	_ = doc.Buffer.ReplaceUTF16(0, 8, "edited")
	if err = doc.Save(); !errors.Is(err, ErrReadOnly) {
		t.Fatal(err)
	}
	if string(read(t, path)) != "original" {
		t.Fatal("read-only file changed")
	}
	copyPath := filepath.Join(filepath.Dir(path), "writable-copy.txt")
	if err = doc.SaveAs(copyPath); err != nil {
		t.Fatal(err)
	}
	if doc.ReadOnly {
		t.Fatal("new copy is not writable")
	}
	_ = doc.Buffer.BeginGroup()
	_ = doc.Buffer.ReplaceUTF16(0, 0, "preedit")
	if err = doc.Save(); !errors.Is(err, ErrComposition) {
		t.Fatal(err)
	}
	doc.Buffer.CancelGroup()
	if doc.Dirty() {
		t.Fatal("cancel composition changed document")
	}
}

func TestInvalidEncodingAndBinaryUseBoundedInspector(t *testing.T) {
	cases := []struct {
		data []byte
		kind error
	}{
		{[]byte{0xc3, 0x28}, ErrUnsupportedEncoding},
		{[]byte{0xff, 0xfe, 0x00, 0xd8}, ErrUnsupportedEncoding},
		{[]byte{0xfe, 0xff, 0xdc, 0x00}, ErrUnsupportedEncoding},
		{[]byte{0xff, 0xfe, 1}, ErrUnsupportedEncoding},
		{[]byte{0xff, 0xfe, 0, 0, 0x41, 0, 0, 0}, ErrUnsupportedEncoding},
		{[]byte{0xef, 0xbb, 0xbf, 0xff, 0xfe, 0x41, 0}, ErrUnsupportedEncoding},
		{[]byte("a\x00b"), ErrBinary}, {[]byte("%PDF-1.7\ntext"), ErrBinary},
	}
	for i, c := range cases {
		path := filepath.Join(t.TempDir(), "file")
		put(t, path, c.data)
		if _, err := Open(path); !errors.Is(err, c.kind) {
			t.Fatalf("%d: %v", i, err)
		}
		view, err := Inspect(path, 2)
		if err != nil || len(view.Bytes) != 2 || view.Truncated != (len(c.data) > 2) || !view.ReadOnly {
			t.Fatal(view, err)
		}
		if !bytes.Equal(read(t, path), c.data) {
			t.Fatal("open/inspect altered file")
		}
	}
}

func TestAtomicFailureLeavesOriginalAndNoTemporaryFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")
	put(t, path, []byte("before"))
	err := writeAtomic(path, []byte("after"), 0600, false, func() error { return ErrConflict })
	if !errors.Is(err, ErrConflict) || string(read(t, path)) != "before" {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatal("temporary file leaked", entries)
	}
	if err = writeAtomic(path, []byte(strings.Repeat("new", 1000)), 0600, false, nil); err != nil {
		t.Fatal(err)
	}
	entries, _ = os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatal("successful save left temporary file")
	}
}
