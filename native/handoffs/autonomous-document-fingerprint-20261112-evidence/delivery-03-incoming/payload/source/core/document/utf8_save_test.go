package document

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"levnote.local/native/core/text"
)

func TestUTF8StreamingSaveExactBytesAndReopen(t *testing.T) {
	cases := []struct {
		name string
		bom  bool
		text string
	}{
		{name: "plain-lf", text: strings.Repeat("ascii\n", 1200)},
		{name: "bom-crlf-unicode", bom: true, text: "甲🙂\r\n乙\r\n" + strings.Repeat("字🙂\r\nx\n", 900)},
		{name: "multi-piece-boundary", bom: true, text: strings.Repeat("🙂中x\r\n", 1500)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "note.txt")
			expected, err := encode(tc.text, UTF8, tc.bom)
			if err != nil {
				t.Fatal(err)
			}
			doc := New(tc.text)
			doc.BOM = tc.bom
			if err = doc.SaveAs(path); err != nil {
				t.Fatal(err)
			}
			if got := read(t, path); !bytes.Equal(got, expected) {
				t.Fatalf("saved bytes differ: got %d want %d", len(got), len(expected))
			}
			if doc.Dirty() || doc.IsNew || doc.ReadOnly {
				t.Fatal("SaveAs did not establish a clean, writable baseline")
			}
			reopened, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			if reopened.Buffer.String() != tc.text || reopened.BOM != tc.bom {
				t.Fatal("reopen changed text or BOM state")
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 {
				t.Fatalf("temporary file leaked: %v", entries)
			}
		})
	}
}

func TestUTF8StreamingSavePreservesHistorySelectionAndBaseline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.txt")
	doc := New("甲乙🙂丙")
	if err := doc.Buffer.ReplaceUTF16(0, 1, "新"); err != nil {
		t.Fatal(err)
	}
	if err := doc.Buffer.SetSelection(2, 4); err != nil {
		t.Fatal(err)
	}
	history := doc.Buffer.History()
	selection := doc.Buffer.Selection()
	if err := doc.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	firstToken := doc.DiskToken()
	if !firstToken.Valid() || doc.Dirty() {
		t.Fatal("SaveAs did not establish a valid clean baseline")
	}
	if got := doc.Buffer.History(); got != history || doc.Buffer.Selection() != selection {
		t.Fatal("SaveAs changed history or selection")
	}
	if err := doc.Buffer.ReplaceUTF16(1, 2, "文"); err != nil {
		t.Fatal(err)
	}
	if !doc.Dirty() {
		t.Fatal("edit did not dirty document")
	}
	oldSelection := doc.Buffer.Selection()
	if err := doc.Save(); err != nil {
		t.Fatal(err)
	}
	if doc.Dirty() || !doc.Buffer.CanUndo() || doc.Buffer.Selection() != oldSelection {
		t.Fatal("Save changed dirty, undo, or selection state")
	}
	if doc.SameBaseline(firstToken) || !doc.DiskToken().Valid() {
		t.Fatal("Save did not advance the disk baseline")
	}
	if !doc.Buffer.Undo() || !doc.Dirty() {
		t.Fatal("undo after Save lost edit history")
	}
	if !doc.Buffer.Redo() || doc.Dirty() {
		t.Fatal("redo did not restore the saved text")
	}
}

func TestUTF8StreamingSaveUntouchedDoesNotRewrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.txt")
	original := []byte("\xef\xbb\xbf甲🙂\r\nline\n")
	put(t, path, original)
	stamp := time.Unix(1600000000, 0)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	doc, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = doc.Save(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(stamp) || !bytes.Equal(read(t, path), original) {
		t.Fatal("untouched Save rewrote bytes or timestamp")
	}
}

func TestUTF8StreamingSaveRejectsWrongDigestWithoutStateChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.txt")
	put(t, path, []byte("baseline"))
	doc, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = doc.Buffer.ReplaceUTF16(0, len("baseline"), "live"); err != nil {
		t.Fatal(err)
	}
	if !doc.Dirty() {
		t.Fatal("fixture did not become dirty")
	}
	before := *doc
	beforeSaved := doc.saved
	beforeDisk := doc.disk
	token := doc.DiskToken()
	wrong := sha256.Sum256([]byte("wrong expected"))
	if err = doc.didSave(wrong, int64(len("baseline")), beforeDisk.info); !errors.Is(err, ErrConflict) {
		t.Fatalf("wrong digest accepted: %v", err)
	}
	if doc.Path != before.Path || doc.Encoding != before.Encoding || doc.BOM != before.BOM ||
		doc.ReadOnly != before.ReadOnly || doc.IsNew != before.IsNew ||
		doc.baselineRevision != before.baselineRevision ||
		doc.dirtyVersion != before.dirtyVersion || doc.dirtyValue != before.dirtyValue ||
		doc.dirtyCached != before.dirtyCached ||
		!reflect.DeepEqual(doc.saved, beforeSaved) ||
		doc.disk.hash != beforeDisk.hash || !os.SameFile(doc.disk.info, beforeDisk.info) ||
		!doc.SameBaseline(token) {
		t.Fatal("wrong digest changed saved or disk state")
	}
	if doc.Buffer.String() != "live" || doc.Buffer.EqualSnapshot(doc.saved) {
		t.Fatal("wrong digest adopted the live buffer as baseline")
	}
}

func TestUTF8AtomicDigestOnlyAfterSuccessfulPublication(t *testing.T) {
	const source = "甲🙂\r\nline\n"
	expected, err := encode(source, UTF8, true)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "new.txt")
	digest, err := writeUTF8Atomic(path, 0600, true, nil, text.New(source).Snapshot(), true)
	if err != nil {
		t.Fatal(err)
	}
	if digest != sha256.Sum256(expected) || !bytes.Equal(read(t, path), expected) {
		t.Fatal("successful publication returned the wrong digest or bytes")
	}

	original := read(t, path)
	digest, err = writeUTF8Atomic(path, 0600, false, func() error {
		return ErrConflict
	}, text.New("replacement").Snapshot(), false)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("publication check failure not preserved: %v", err)
	}
	if digest != (sha256Digest{}) || !bytes.Equal(read(t, path), original) {
		t.Fatal("failed publication exposed a digest or changed the target")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("failed publication leaked a temporary file: %v", entries)
	}
}

func TestUTF8StreamingSaveTempFailureKeepsTargetAndCleansPartialFile(t *testing.T) {
	sentinel := errors.New("injected output failure")
	cases := []struct {
		name    string
		source  string
		accept  int
		wantErr error
	}{
		{name: "short-final-flush", source: "partial", accept: 3, wantErr: io.ErrShortWrite},
		{name: "short-full-buffer", source: strings.Repeat("a", 40<<10), accept: (32 << 10) - 1, wantErr: io.ErrShortWrite},
		{name: "failed-final-flush", source: "partial", accept: 3, wantErr: sentinel},
		{name: "failed-full-buffer", source: strings.Repeat("a", 40<<10), accept: (32 << 10) - 1, wantErr: sentinel},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "note.txt")
			put(t, path, []byte("before"))
			written := 0
			writer := &interruptWriter{accept: tc.accept, written: &written}
			if tc.wantErr != io.ErrShortWrite {
				writer.err = tc.wantErr
			}
			var digest sha256Digest
			err := writeAtomicWriter(path, 0600, false, nil, func(temp io.Writer) error {
				writer.dst = temp
				return streamUTF8(text.New(tc.source).Snapshot(), false, writer, &digest)
			})
			if tc.wantErr == io.ErrShortWrite {
				if err != io.ErrShortWrite {
					t.Fatalf("want exact ErrShortWrite, got %v", err)
				}
			} else if !errors.Is(err, tc.wantErr) {
				t.Fatalf("injected error not forwarded: %v", err)
			}
			if written <= 0 {
				t.Fatal("failure occurred before any bytes reached the temporary file")
			}
			if string(read(t, path)) != "before" {
				t.Fatal("failed streaming writer changed the target")
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 {
				t.Fatalf("partial temporary file leaked: %v", entries)
			}
		})
	}
}

type interruptWriter struct {
	dst     io.Writer
	accept  int
	written *int
	err     error
}

func (w *interruptWriter) Write(p []byte) (int, error) {
	n := min(len(p), w.accept)
	if n > 0 {
		written, err := w.dst.Write(p[:n])
		*w.written += written
		if err != nil {
			return written, err
		}
		if written != n {
			return written, io.ErrShortWrite
		}
	}
	if w.err != nil {
		return n, w.err
	}
	if n != len(p) {
		return n, nil
	}
	return n, nil
}
