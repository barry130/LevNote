package drawio

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestSourceRetentionFixtureIdentity(t *testing.T) {
	source := sourceRetentionBenchmarkSource()
	document, _, err := Parse(source, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(source)
	nodes := 0
	for _, page := range document.Pages {
		nodes += len(page.Model.Nodes)
	}
	t.Logf("source_bytes=%d sha256=%s pages=%d uncompressed_pages=%d nodes=%d",
		len(source), hex.EncodeToString(sum[:]), len(document.Pages), sourceRetentionBenchmarkPages-1, nodes)
}
