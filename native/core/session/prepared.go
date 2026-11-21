package session

import (
	"io"

	"levnote.local/native/core/document"
)

// PrepareSave validates the same full budget and writes the same exact JSON
// stream as Save, without publishing it. The prepared file retains no mutable
// caller data. Commit/Abort ownership belongs to the UI coordinator.
func PrepareSave(path string, snapshot Snapshot) (*document.PreparedPrivateState, error) {
	snapshot.Version = Version
	expected, err := preflightSnapshot(snapshot)
	if err != nil {
		return nil, err
	}
	return document.PreparePrivateStateTo(path, func(w io.Writer) error {
		return writeSnapshotJSON(w, snapshot, expected)
	})
}
