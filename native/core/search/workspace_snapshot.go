package search

import (
	"crypto/sha256"

	"levnote.local/native/core/text"
)

// workspaceSnapshot describes the decoded UTF-8 source used for UTF-16 match
// coordinates. It is shared by a file's results without retaining its contents.
// Only Workspace creates it, from the same source passed to findCompiled.
type workspaceSnapshot struct {
	bytes  int
	digest [sha256.Size]byte
}

func snapshotSource(source string) *workspaceSnapshot {
	return &workspaceSnapshot{bytes: len(source), digest: sha256.Sum256([]byte(source))}
}

// HasSnapshot reports whether Workspace supplied this result's source identity.
// A manually constructed FileMatch has no evidence for its stored coordinates.
func (m FileMatch) HasSnapshot() bool { return m.snapshot != nil }

// MatchesBuffer compares the current decoded buffer with the search source.
// Saved/dirty state, versions, timestamps and the current query are not content
// identity. Reading the buffer leaves its selection and history untouched.
func (m FileMatch) MatchesBuffer(buffer *text.Buffer) bool {
	if m.snapshot == nil || buffer == nil || buffer.LenBytes() != m.snapshot.bytes {
		return false
	}
	hash := sha256.New()
	n, err := buffer.WriteTo(hash)
	if err != nil || n != int64(m.snapshot.bytes) {
		return false
	}
	return [sha256.Size]byte(hash.Sum(nil)) == m.snapshot.digest
}

// MatchesBufferWithoutSnapshot performs the same content check without going
// through Buffer.WriteTo, so a read-only lost-observation caller cannot
// populate a dirty Buffer's snapshot cache. The source remains m.snapshot's
// recorded identity; a nil snapshot still has no evidence for matching.
func (m FileMatch) MatchesBufferWithoutSnapshot(buffer *text.Buffer) bool {
	if m.snapshot == nil || buffer == nil || buffer.LenBytes() != m.snapshot.bytes {
		return false
	}
	source := buffer.String()
	if len(source) != m.snapshot.bytes {
		return false
	}
	digest := sha256.Sum256([]byte(source))
	return digest == m.snapshot.digest
}
