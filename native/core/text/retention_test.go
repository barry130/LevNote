package text

import (
	"strings"
	"testing"
)

func TestReplaceReleasesDeadPieceTailAfterHistoryReset(t *testing.T) {
	source := strings.Repeat("a", chunkBytes)
	b := New(source)
	mid := chunkBytes / 2
	beforeSnapshot := b.Snapshot()

	if err := b.ReplaceUTF16(mid, mid+1, "x"); err != nil {
		t.Fatal(err)
	}
	editedSnapshot := b.Snapshot()
	if !b.Undo() {
		t.Fatal("edit did not enter undo history")
	}
	if got := b.String(); got != source {
		t.Fatalf("undo changed source: got %d bytes, want %d", len(got), len(source))
	}
	if !b.Redo() {
		t.Fatal("edit did not enter redo history")
	}
	edited := source[:mid] + "x" + source[mid+1:]
	if got := b.String(); got != edited {
		t.Fatalf("redo changed source: got %d bytes, want %d", len(got), len(edited))
	}
	b.ResetHistory()
	if got := beforeSnapshot.String(); got != source {
		t.Fatalf("snapshot before edit changed: got %d bytes, want %d", len(got), len(source))
	}
	if got := editedSnapshot.String(); got != edited {
		t.Fatalf("snapshot after edit changed: got %d bytes, want %d", len(got), len(edited))
	}

	full := b.pieces[:cap(b.pieces)]
	logical := make(map[*chunk]struct{}, len(b.pieces))
	for _, p := range b.pieces {
		logical[p.data] = struct{}{}
	}
	retained := make(map[*chunk]struct{})
	retainedBytes := 0
	staleOnlyBytes := 0
	deadSlots := 0
	staleOnlySlots := 0
	duplicateSlots := 0
	for i := len(b.pieces); i < len(full); i++ {
		if full[i].data == nil {
			continue
		}
		deadSlots++
		_, stillLogical := logical[full[i].data]
		if !stillLogical {
			staleOnlySlots++
		} else {
			duplicateSlots++
		}
		if _, seen := retained[full[i].data]; !seen {
			retained[full[i].data] = struct{}{}
			retainedBytes += len(full[i].data.value)
			if !stillLogical {
				staleOnlyBytes += len(full[i].data.value)
			}
		}
	}
	if deadSlots != 0 {
		t.Fatalf("piece backing retains %d dead slots beyond logical length %d (capacity %d): %d stale-only slots / %d stale-only bytes; %d duplicate-logical slots; %d unique referenced bytes total",
			deadSlots, len(b.pieces), cap(b.pieces), staleOnlySlots, staleOnlyBytes, duplicateSlots, retainedBytes)
	}
}
