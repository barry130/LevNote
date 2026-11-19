package document

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"
	"testing"

	"levnote.local/native/core/text"
)

func benchmarkUTF8Fixture(size int) string {
	var out strings.Builder
	out.Grow(size)
	patterns := []string{"ASCII line\r\n", "中文行\n", "emoji🙂tail\r\n", "mixed 字🙂\n"}
	for index := 0; out.Len() < size; index++ {
		pattern := patterns[index%len(patterns)]
		if out.Len()+len(pattern) > size {
			break
		}
		out.WriteString(pattern)
	}
	for out.Len() < size {
		out.WriteByte('x')
	}
	return out.String()
}

func fullBufferUTF8Work(snapshot text.Snapshot, bom bool, w io.Writer, digest *sha256Digest) error {
	encoded, err := encode(snapshot.String(), UTF8, bom)
	if err != nil {
		return err
	}
	*digest = sha256.Sum256(encoded)
	_, err = w.Write(encoded)
	return err
}

func streamingUTF8Work(snapshot text.Snapshot, bom bool, w io.Writer, digest *sha256Digest) error {
	return streamUTF8(snapshot, bom, w, digest)
}

func BenchmarkUTF8Output1MiB(b *testing.B) { benchmarkUTF8Output(b, 1<<20) }

func BenchmarkUTF8Output8MiB(b *testing.B) { benchmarkUTF8Output(b, 8<<20) }

func benchmarkUTF8Output(b *testing.B, size int) {
	source := benchmarkUTF8Fixture(size)
	snapshot := text.New(source).Snapshot()
	oracle, err := encode(source, UTF8, true)
	if err != nil {
		b.Fatal(err)
	}
	expectedDigest := sha256.Sum256(oracle)
	fixtureDigest := sha256.Sum256([]byte(source))
	b.Logf("fixture_bytes=%d fixture_sha256=%x", len(source), fixtureDigest)
	b.ReportAllocs()
	for _, mode := range []struct {
		name string
		run  func(w io.Writer, digest *sha256Digest) error
	}{
		{"full-buffer", func(w io.Writer, digest *sha256Digest) error {
			return fullBufferUTF8Work(snapshot, true, w, digest)
		}},
		{"streaming", func(w io.Writer, digest *sha256Digest) error {
			return streamingUTF8Work(snapshot, true, w, digest)
		}},
	} {
		b.Run(fmt.Sprintf("%s/size=%d", mode.name, size), func(b *testing.B) {
			var digest sha256Digest
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := mode.run(io.Discard, &digest); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			if digest != expectedDigest {
				b.Fatal("streamed digest does not match the encoded fixture")
			}
		})
	}
}
