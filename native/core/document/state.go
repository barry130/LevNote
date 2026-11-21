package document

import (
	"errors"
	"io"
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
	prepared, err := PreparePrivateStateTo(path, write)
	if err != nil {
		return err
	}
	commitErr := prepared.Commit(nil)
	cleanupErr := prepared.Abort()
	return finishPrivateStateWrite(prepared.path, prepared.committed, commitErr, cleanupErr)
}

func finishPrivateStateWrite(path string, committed bool, commitErr, cleanupErr error) error {
	var publication *StatePublicationError
	if errors.As(commitErr, &publication) {
		return &StatePublicationError{
			Path: publication.Path, Committed: committed, Err: publication.Err,
			CleanupErr: errors.Join(publication.CleanupErr, cleanupErr),
		}
	}
	if committed {
		if commitErr == nil && cleanupErr == nil {
			return nil
		}
		return &StatePublicationError{Path: path, Committed: true, Err: commitErr, CleanupErr: cleanupErr}
	}
	if commitErr == nil && cleanupErr == nil {
		return nil
	}
	return &StatePublicationError{Path: path, Err: commitErr, CleanupErr: cleanupErr}
}
