package document

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

var errFingerprintReadBoundary = errors.New("fingerprint delivered beyond expected encoded bytes plus one")

// A non-embedded real file avoids exposing os.File.WriteTo: every byte delivered
// to the production hash must pass this Reader. Stat always returns real OS data.
type growingFingerprintFile struct {
	file        *os.File
	path        string
	limit       int64
	delivered   int64
	before      os.FileInfo
	statCalls   int
	grew        bool
	crossed     bool
}

func (f *growingFingerprintFile) Stat() (os.FileInfo, error) {
	info, err := f.file.Stat()
	if err == nil {
		f.statCalls++
		if f.before == nil {
			f.before = info
		}
	}
	return info, err
}

func (f *growingFingerprintFile) Read(buffer []byte) (int, error) {
	if !f.grew {
		if f.before == nil || f.statCalls == 0 {
			return 0, errors.New("growth fixture requires the production before-Stat")
		}
		writer, err := os.OpenFile(f.path, os.O_WRONLY|os.O_APPEND, 0)
		if err != nil {
			return 0, err
		}
		growth := []byte("external growth after the legitimate stat and before the first read")
		n, writeErr := writer.Write(growth)
		closeErr := writer.Close()
		if writeErr != nil {
			return 0, writeErr
		}
		if n != len(growth) {
			return 0, io.ErrShortWrite
		}
		if closeErr != nil {
			return 0, closeErr
		}
		current, err := os.Stat(f.path)
		if err != nil {
			return 0, err
		}
		if !os.SameFile(f.before, current) || current.Size() <= f.before.Size() {
			return 0, errors.New("growth fixture did not grow the same real file")
		}
		f.grew = true
	}
	n, err := f.file.Read(buffer)
	f.delivered += int64(n)
	if f.delivered > f.limit {
		f.crossed = true
		return n, errFingerprintReadBoundary
	}
	return n, err
}

// Internal byte-work-boundary evidence, not a public business seam or a claim
// that ordinary filesystem APIs provide compare-and-swap publication.
func TestFingerprintGrowthReadsAtMostExpectedEncodedBytesPlusOne(t *testing.T) {
	// Independent literal UTF-8 BOM + "甲🙂\r\n": twelve encoded disk bytes,
	// distinct from both the UTF-16 character count and the editable-size cap.
	original := []byte{0xef, 0xbb, 0xbf, 0xe7, 0x94, 0xb2, 0xf0, 0x9f, 0x99, 0x82, 0x0d, 0x0a}
	path := filepath.Join(t.TempDir(), "growing.txt")
	put(t, path, original)
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	baseline, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	expectedLength := int64(len(original))
	reader := &growingFingerprintFile{file: file, path: path, limit: expectedLength + 1}
	id, err := fingerprintOpened(path, reader, expectedLength, baseline)
	if reader.crossed || reader.delivered > expectedLength+1 {
		t.Fatalf("encoded-byte work boundary crossed: delivered=%d allowed=%d grew=%t error=%v", reader.delivered, expectedLength+1, reader.grew, err)
	}
	if !reader.grew || reader.before == nil || reader.before.Size() != expectedLength || reader.delivered != expectedLength+1 {
		t.Fatalf("deterministic same-file growth was not observed at the byte boundary: delivered=%d before=%v grew=%t", reader.delivered, reader.before, reader.grew)
	}
	if !errors.Is(err, ErrConflict) || errors.Is(err, errFingerprintReadBoundary) || id.info != nil || id.hash != [32]byte{} {
		t.Fatalf("growth established an identity or hid the conflict: identity=%#v error=%v", id, err)
	}
}
