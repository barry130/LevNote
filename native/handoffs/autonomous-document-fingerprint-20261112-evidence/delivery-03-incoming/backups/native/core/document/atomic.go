package document

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
)

type sha256Digest = [32]byte

// The replacement is atomic on supported local filesystems. The final check
// narrows, but cannot eliminate, a race with an uncooperative external writer
// between checking and replacing; ordinary filesystem APIs provide no CAS.
func writeAtomic(path string, data []byte, mode os.FileMode, create bool, check func() error) error {
	return writeAtomicWriter(path, mode, create, check, func(w io.Writer) error {
		n, err := w.Write(data)
		if err == nil && n != len(data) {
			err = io.ErrShortWrite
		}
		return err
	})
}

type atomicOutcome struct {
	Committed bool
}

// writeAtomicWriter owns the complete temp-file lifecycle. The callback sees a
// non-nil writer and returns only after all intended bytes have been written.
// The final check runs after the new bytes are durable and before publication;
// publication remains the commit point.
func writeAtomicWriter(path string, mode os.FileMode, create bool, check func() error, write func(io.Writer) error) error {
	_, err := writeAtomicWriterOutcome(path, mode, create, check, write, nil)
	return err
}

func writeAtomicWriterOutcome(path string, mode os.FileMode, create bool, check func() error, write func(io.Writer) error, syncDir func(string) error) (atomicOutcome, error) {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".levnote-save-*")
	if err != nil {
		return atomicOutcome{}, err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err = temp.Chmod(mode.Perm()); err == nil {
		err = write(temp)
	}
	if err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err != nil {
		return atomicOutcome{}, err
	}
	if closeErr != nil {
		return atomicOutcome{}, closeErr
	}
	if check != nil {
		if err = check(); err != nil {
			return atomicOutcome{}, err
		}
	}
	if create {
		err = publishNew(tempPath, path)
	} else {
		err = replaceExisting(tempPath, path)
	}
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return atomicOutcome{}, &Error{"save as", path, ErrExists, err}
		}
		return atomicOutcome{}, err
	}
	outcome := atomicOutcome{Committed: true}
	if syncDir == nil {
		syncDir = syncDirectory
	}
	return outcome, syncDir(dir)
}

// writeUTF8Atomic writes the BOM and snapshot bytes without materializing the
// full text or encoded output. The bounded buffer precedes the hash so each
// Snapshot piece can use io.WriteString without a fresh byte-slice conversion.
func writeUTF8Atomic(path string, mode os.FileMode, create bool, check func() error, snapshot io.WriterTo, bom bool) (sha256Digest, error) {
	var digest sha256Digest
	_, err := writeAtomicWriterOutcome(path, mode, create, check, func(w io.Writer) error {
		return streamUTF8(snapshot, bom, w, &digest)
	}, nil)
	if err != nil {
		return sha256Digest{}, err
	}
	return digest, nil
}

// streamUTF8 writes a UTF-8 snapshot and returns the digest of the exact
// accepted bytes. It is shared by saves, failure tests, and the bounded
// allocation comparison.
func streamUTF8(snapshot io.WriterTo, bom bool, w io.Writer, digest *sha256Digest) error {
	hash := sha256.New()
	buffered := bufio.NewWriterSize(io.MultiWriter(w, hash), 32*1024)
	if bom {
		if _, err := buffered.Write([]byte{0xef, 0xbb, 0xbf}); err != nil {
			return err
		}
	}
	if _, err := snapshot.WriteTo(buffered); err != nil {
		return err
	}
	if err := buffered.Flush(); err != nil {
		return err
	}
	copy(digest[:], hash.Sum(nil))
	return nil
}
