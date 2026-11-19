package text

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestLineRangesAndColumnsAcrossChunksEditsAndRestore(t *testing.T) {
	value := strings.Repeat("中文🙂e\u0301\r\nshort\n", 600) + strings.Repeat("界", 6000)
	b := New(value)
	saved := b.Snapshot()
	rng := rand.New(rand.NewSource(91))
	verify := func() {
		t.Helper()
		full := b.String()
		lines := strings.Split(full, "\n")
		start := 0
		for i, value := range lines {
			end := start + UTF16Len(value)
			if i+1 < len(lines) && strings.HasSuffix(value, "\r") {
				end--
			}
			span, err := b.LineRange(i)
			if err != nil || span != (Range{start, end}) {
				t.Fatalf("line %d range=%+v want=%d:%d err=%v", i, span, start, end, err)
			}
			start += UTF16Len(value) + 1
		}
		runes := []rune(full)
		for i := 0; i < 250; i++ {
			at := rng.Intn(len(runes) + 1)
			prefix := string(runes[:at])
			offset := len(utf16.Encode(runes[:at]))
			line, column, err := b.LineColumn(offset)
			wantLine := strings.Count(prefix, "\n")
			wantColumn := UTF16Len(prefix[strings.LastIndexByte(prefix, '\n')+1:])
			if err != nil || line != wantLine || column != wantColumn {
				t.Fatalf("offset %d: %d:%d want %d:%d err %v", offset, line, column, wantLine, wantColumn, err)
			}
		}
	}
	verify()
	_ = b.ReplaceUTF16(0, 0, "\n🙂\r\n")
	verify()
	_ = b.ReplaceUTF16(15, 100, "replacement\r\n")
	verify()
	b.Undo()
	verify()
	b.Restore(saved)
	verify()
}

func TestChangedLineJournalIncludesUndoCancelAndOldRevision(t *testing.T) {
	b := New("a\nb\nc\nd")
	if b.ChangedLineSince(b.Version()) != 4 {
		t.Fatal("unchanged revision invalidates lines")
	}
	v := b.Version()
	_ = b.ReplaceUTF16(6, 7, "D")
	if b.ChangedLineSince(v) != 3 {
		t.Fatal("tail invalidation")
	}
	_ = b.ReplaceUTF16(2, 3, "B\nnew")
	if b.ChangedLineSince(v) != 1 {
		t.Fatal("journal did not retain earliest line")
	}
	v = b.Version()
	b.Undo()
	if b.ChangedLineSince(v) != 1 {
		t.Fatal("undo missing from journal")
	}
	v = b.Version()
	_ = b.BeginGroup()
	_ = b.ReplaceUTF16(0, 0, "x")
	b.CancelGroup()
	if b.ChangedLineSince(v) != 0 {
		t.Fatal("cancel missing from journal")
	}
	v = b.Version()
	for i := 0; i < 65; i++ {
		_ = b.ReplaceUTF16(b.LenUTF16(), b.LenUTF16(), "z")
	}
	if b.ChangedLineSince(v) != 0 {
		t.Fatal("expired revision must invalidate all")
	}
}

func TestGraphemeBoundaryUnicodeAndLongContext(t *testing.T) {
	clusters := []string{"A", "e\u0301", "🙂", "👨‍👩‍👧‍👦", "🇨🇳", "\r\n", "क्‍ष", "Z"}
	b := New(strings.Join(clusters, ""))
	position := 0
	for _, cluster := range clusters {
		next, err := b.GraphemeBoundary(position, 1)
		want := position + UTF16Len(cluster)
		if err != nil || next != want {
			t.Fatalf("next %q %d => %d want %d err %v", cluster, position, next, want, err)
		}
		back, err := b.GraphemeBoundary(next, -1)
		if err != nil || back != position {
			t.Fatalf("previous %q %d => %d want %d", cluster, next, back, position)
		}
		position = next
	}
	for _, cluster := range []string{"a" + strings.Repeat("\u0301", 2000), strings.Repeat("🇦", 199) + "🇧"} {
		b := New("x" + cluster + "z")
		end := 1 + UTF16Len(cluster)
		got, err := b.GraphemeBoundary(end, -1)
		want := 1
		if strings.HasPrefix(cluster, "🇦") {
			want = end - 4
		}
		if err != nil || got != want {
			t.Fatalf("long context boundary %d want %d err %v", got, want, err)
		}
	}
}

func BenchmarkLargeTextLookup(b *testing.B) {
	for _, count := range []int{50_000, 500_000} {
		b.Run(fmt.Sprintf("lines_%d", count), func(b *testing.B) {
			doc := New(strings.Repeat("A line of 中文🙂 content.\r\n", count))
			line := count - 3
			offset, _ := doc.OffsetOfLine(line)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, _, err := doc.LineColumn(offset + 14); err != nil {
					b.Fatal(err)
				}
				if _, err := doc.LineRange(line); err != nil {
					b.Fatal(err)
				}
				if _, err := doc.SliceUTF16(offset, offset+10); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkLargeTextInput(b *testing.B) {
	for _, count := range []int{50_000, 500_000} {
		b.Run(fmt.Sprintf("lines_%d", count), func(b *testing.B) {
			doc := New(strings.Repeat("A line of 中文🙂 content.\r\n", count))
			position := doc.LenUTF16()
			doc.SetHistoryLimits(16, 1<<20)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := doc.ReplaceUTF16(position, position, "x"); err != nil {
					b.Fatal(err)
				}
				doc.Undo()
			}
		})
	}
}
