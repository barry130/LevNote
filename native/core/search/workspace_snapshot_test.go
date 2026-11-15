package search_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levnote.local/native/core/search"
	"levnote.local/native/core/text"
)

func TestWorkspaceSnapshotBatchesRetainSearchedBytes(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "batch.txt")
	original := strings.Repeat("needle\n", 70)
	changed := strings.Repeat("xxxxxx\n", 70)
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	var matches []search.FileMatch
	batches := 0
	summary, err := search.Workspace(t.Context(), root, search.WorkspaceOptions{Options: search.Options{Query: "needle"}}, func(batch search.Batch) {
		batches++
		matches = append(matches, batch.Matches...)
		if batches == 1 {
			// Disk changes between emitted batches. Later matches must still
			// describe the exact bytes that produced the earlier coordinates.
			if err := os.WriteFile(path, []byte(changed), 0600); err != nil {
				t.Fatal(err)
			}
		}
	})
	if err != nil || batches != 2 || len(matches) != 70 || summary.Matches != 70 {
		t.Fatalf("stream premise: batches=%d matches=%d summary=%+v error=%v", batches, len(matches), summary, err)
	}
	before, after := text.New(original), text.New(changed)
	for _, match := range matches {
		if !match.HasSnapshot() || !match.MatchesBuffer(before) || match.MatchesBuffer(after) {
			t.Fatalf("batch result must retain search-source identity: %+v", match)
		}
	}
	if before.Version() != 0 || before.Selection() != (text.Range{}) || before.History().UndoDepth != 0 {
		t.Fatal("identity comparison mutated the public buffer")
	}
}

func TestWorkspaceSnapshotDecodedUnicodeBOMAndCRLF(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"utf8-bom", append([]byte{0xef, 0xbb, 0xbf}, []byte("😀\r\nneedle\r\n")...)},
		{"utf16-le", []byte{0xff, 0xfe, 0x3d, 0xd8, 0x00, 0xde, 13, 0, 10, 0, 'n', 0, 'e', 0, 'e', 0, 'd', 0, 'l', 0, 'e', 0, 13, 0, 10, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "unicode.txt")
			if err := os.WriteFile(path, tc.data, 0600); err != nil {
				t.Fatal(err)
			}
			var matches []search.FileMatch
			_, err := search.Workspace(t.Context(), root, search.WorkspaceOptions{Options: search.Options{Query: "needle"}}, func(batch search.Batch) {
				matches = append(matches, batch.Matches...)
			})
			if err != nil || len(matches) != 1 {
				t.Fatalf("Unicode search: matches=%+v error=%v", matches, err)
			}
			match := matches[0]
			if match.Start != 4 || match.End != 10 || match.Line != 2 || match.Column != 1 ||
				!match.MatchesBuffer(text.New("😀\r\nneedle\r\n")) ||
				match.MatchesBuffer(text.New("😁\r\nneedle\r\n")) || match.MatchesBuffer(text.New("😀\nneedle\n")) {
				t.Fatalf("identity must describe decoded bytes including Unicode and line endings: %+v", match)
			}
		})
	}
}

func TestWorkspaceSnapshotMissingIdentityDoesNotAuthenticate(t *testing.T) {
	match := search.FileMatch{Path: "manual.txt", Start: 0, End: 6, Preview: "needle"}
	if match.HasSnapshot() || match.MatchesBuffer(text.New("needle")) || match.MatchesBuffer(text.New("")) || match.MatchesBuffer(nil) {
		t.Fatal("a manually fabricated result must not authenticate any buffer")
	}
}
