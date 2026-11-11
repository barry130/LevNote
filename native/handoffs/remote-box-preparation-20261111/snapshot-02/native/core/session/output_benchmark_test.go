package session

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"io"
	"testing"
)

func BenchmarkSessionOutput(b *testing.B) {
	draft := repeatedFixture('d', 1<<20)
	benchmarkSessionOutput(b, "one-draft-1MiB", Snapshot{
		Version: Version,
		Tabs:    []Tab{{Draft: &draft}},
	})

	const tabCount = 8
	const tabDraftBytes = (1 << 20) / tabCount
	tabs := make([]Tab, tabCount)
	for i := range tabs {
		value := repeatedFixture(byte('a'+i), tabDraftBytes)
		tabs[i] = Tab{Draft: &value}
	}
	benchmarkSessionOutput(b, "eight-tabs-1MiB", Snapshot{
		Version: Version,
		Active:  tabCount - 1,
		Tabs:    tabs,
	})
}

func benchmarkSessionOutput(b *testing.B, name string, snapshot Snapshot) {
	b.Helper()
	expected, err := preflightSnapshot(snapshot)
	if err != nil {
		b.Fatal(err)
	}
	oracle, err := json.Marshal(snapshot)
	if err != nil {
		b.Fatal(err)
	}
	if len(oracle) != expected {
		b.Fatalf("preflight = %d, Marshal = %d", expected, len(oracle))
	}
	digest := sha256.Sum256(oracle)
	var helperOracle bytes.Buffer
	if err := writeSnapshotJSON(&helperOracle, snapshot, expected); err != nil {
		b.Fatal(err)
	}
	if !bytes.Equal(helperOracle.Bytes(), oracle) {
		b.Fatal("new output differs from json.Marshal oracle")
	}
	if bytes.HasSuffix(helperOracle.Bytes(), []byte{'\n'}) {
		b.Fatal("new output retained Encoder's final LF")
	}
	b.Run(name, func(b *testing.B) {
		b.Logf("fixture_sha256=%x emitted_bytes=%d tabs=%d", digest, expected, len(snapshot.Tabs))
		b.Run("marshal-discard", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				encoded, err := json.Marshal(snapshot)
				if err != nil {
					b.Fatal(err)
				}
				if len(encoded) != expected {
					b.Fatalf("Marshal = %d, want %d", len(encoded), expected)
				}
				if n, err := io.Discard.Write(encoded); err != nil || n != len(encoded) {
					b.Fatalf("Discard.Write = %d, %v", n, err)
				}
			}
		})
		b.Run("exact-writer-discard", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if err := writeSnapshotJSON(io.Discard, snapshot, expected); err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}

func repeatedFixture(fill byte, size int) string {
	data := make([]byte, size)
	for i := range data {
		data[i] = fill
	}
	return string(data)
}
