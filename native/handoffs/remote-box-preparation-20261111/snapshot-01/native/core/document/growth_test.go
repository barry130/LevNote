package document

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levnote.local/native/core/text"
)

func TestGrowthDecodedSeedSaveAndRevertFidelity(t *testing.T) {
	for _, enc := range []Encoding{UTF8, UTF16LE, UTF16BE, GBK} {
		t.Run(string(enc), func(t *testing.T) {
			const value = "中文中文\r\n"
			bom := enc != GBK
			original, err := encode(value, enc, bom)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "seed.txt")
			put(t, path, original)
			doc, err := OpenWithEncoding(path, enc)
			if err != nil || doc.Buffer.String() != value || doc.Encoding != enc || doc.BOM != bom {
				t.Fatal("decode fidelity", err)
			}
			// Substitute the public tiny-policy constructor after the real codec
			// path. Disk limit remains distinct; no huge decoder fixture needed.
			doc.Buffer = text.NewWithLimit(doc.Buffer.String(), 4)
			buffer := doc.Buffer
			if buffer.MaxBytes() != len(value) || doc.Dirty() {
				t.Fatal("oversized decoded seed not preserved")
			}
			before, _ := os.Stat(path)
			if err = doc.Save(); err != nil || !bytes.Equal(read(t, path), original) {
				t.Fatal("untouched byte save", err)
			}
			after, _ := os.Stat(path)
			if !before.ModTime().Equal(after.ModTime()) {
				t.Fatal("untouched save rewrote file")
			}
			if err = buffer.ReplaceUTF16(0, buffer.LenUTF16(), "中"); err != nil || !doc.Dirty() {
				t.Fatal("shrink", err)
			}
			if !buffer.Undo() || doc.Dirty() || !buffer.Redo() || !doc.Dirty() {
				t.Fatal("decoded seed history")
			}
			if err = doc.Revert(); err != nil || doc.Buffer != buffer || buffer.String() != value || buffer.MaxBytes() != len(value) || doc.Dirty() || buffer.CanUndo() {
				t.Fatal("Revert lost identity/ceiling/fidelity", err)
			}
			if err = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), "x"); !errors.Is(err, text.ErrSizeLimit) || doc.Dirty() {
				t.Fatal("reverted seed refused growth unsafely", err)
			}
			short, _ := encode("中", enc, bom)
			put(t, path, short)
			if err = doc.Revert(); err != nil || doc.Buffer != buffer || buffer.MaxBytes() != 4 || doc.Encoding != enc || doc.BOM != bom {
				t.Fatal("Revert did not use destination base", err)
			}
			if err = buffer.ReplaceUTF16(1, 1, "x"); err != nil || doc.Save() != nil {
				t.Fatal("post-Revert legal edit/save", err)
			}
			want, _ := encode("中x", enc, bom)
			if !bytes.Equal(read(t, path), want) {
				t.Fatal("encoding/BOM changed on legal edit")
			}
		})
	}
}

func TestGrowthDiskReadBoundRemainsDistinct(t *testing.T) {
	const value = "中文中文"
	for _, enc := range []Encoding{UTF16LE, GBK} {
		raw, _ := encode(value, enc, enc == UTF16LE)
		path := filepath.Join(t.TempDir(), "decoded.txt")
		put(t, path, raw)
		doc, err := OpenLimitedRead(path, enc, int64(len(raw)))
		if err != nil || doc.Buffer.String() != value || doc.Buffer.LenBytes() <= len(raw) || doc.Buffer.MaxBytes() != text.DefaultMaxBytes {
			t.Fatal("logical default conflated with disk read cap", enc, err)
		}
		if _, err = OpenLimitedRead(path, enc, int64(len(raw)-1)); !errors.Is(err, ErrTooLarge) || errors.Is(err, text.ErrSizeLimit) {
			t.Fatal("stable disk error changed", err)
		}
	}
	var diskLimit int64 = MaxEditableBytes
	if diskLimit != text.DefaultMaxBytes {
		t.Fatal("shared lower-layer default drift")
	}
}
