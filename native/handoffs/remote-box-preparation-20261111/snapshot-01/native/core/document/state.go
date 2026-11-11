package document

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

// WritePrivateState persists application-owned recovery/configuration data.
// It is not a document-save API: callers only pass their state directory.
func WritePrivateState(path string, data []byte) error {
	return WritePrivateStateTo(path, func(w io.Writer) error {
		n, err := w.Write(data)
		if err == nil && n != len(data) {
			return io.ErrShortWrite
		}
		return err
	})
}

// WritePrivateStateTo persists application-owned data supplied incrementally.
// The callback writes only to the durable temp file; publication remains the
// commit point after the callback, sync, close and optional checks succeed.
func WritePrivateStateTo(path string, write func(io.Writer) error) error {
	if write == nil {
		return errors.New("private-state writer is nil")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	_, err := os.Stat(path)
	create := errors.Is(err, os.ErrNotExist)
	if err != nil && !create {
		return err
	}
	return writeAtomicWriter(path, 0600, create, nil, write)
}
