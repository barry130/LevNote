package drawio

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
)

func TestSourceRetentionExportedViewsAreIsolated(t *testing.T) {
	t.Run("caller input", func(t *testing.T) {
		source := sourceRetentionBenchmarkSource()
		original := cloneBytes(source)
		document, _, err := Parse(source, ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		for index := range source {
			source[index] ^= 0xff
		}
		assertRetentionEdit(t, document, original)
	})

	t.Run("document source", func(t *testing.T) {
		source := sourceRetentionBenchmarkSource()
		original := cloneBytes(source)
		document, _, err := Parse(source, ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		for index := range document.Source {
			document.Source[index] ^= 0xff
		}
		assertRetentionEdit(t, document, original)
	})

	t.Run("original bytes result", func(t *testing.T) {
		source := sourceRetentionBenchmarkSource()
		original := cloneBytes(source)
		document, _, err := Parse(source, ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		copy := document.OriginalBytes()
		for index := range copy {
			copy[index] ^= 0xff
		}
		assertRetentionEdit(t, document, original)
	})

	t.Run("serialize result", func(t *testing.T) {
		source := sourceRetentionBenchmarkSource()
		original := cloneBytes(source)
		document, _, err := Parse(source, ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		copy, err := document.Serialize()
		if err != nil {
			t.Fatal(err)
		}
		for index := range copy {
			copy[index] ^= 0xff
		}
		assertRetentionEdit(t, document, original)
	})
}

func assertRetentionEdit(t *testing.T, document *Document, original []byte) {
	t.Helper()
	serialized, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(serialized, original) {
		t.Fatal("mutating an exported view changed serialization")
	}
	if !bytes.Equal(document.OriginalBytes(), original) {
		t.Fatal("mutating an exported view changed original bytes")
	}
	if err := document.UpdateText(0, "p0-n0", "after isolation"); err != nil {
		t.Fatal(err)
	}
	edited, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	reparsed, _, err := Parse(edited, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	node := findNode(reparsed.Pages[0].Model.Nodes, "p0-n0")
	if node == nil || node.Text != "after isolation" {
		t.Fatalf("reparsed node = %#v", node)
	}
}

func TestSourceRetentionMultipleEditsPreserveCompressedPage(t *testing.T) {
	source := sourceRetentionBenchmarkSource()
	start := bytes.Index(source, []byte(`name="Compressed">`))
	if start < 0 {
		t.Fatal("compressed diagram marker missing")
	}
	start += len(`name="Compressed">`)
	end := bytes.Index(source[start:], []byte(`</diagram>`))
	if end < 0 {
		t.Fatal("compressed diagram end missing")
	}
	payload := append([]byte(nil), source[start:start+end]...)

	document, _, err := Parse(source, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	updates := []struct {
		page  int
		cell  string
		value string
	}{
		{0, "p0-n0", "first edit"},
		{0, "p0-n1", "second edit"},
		{6, "p6-n23", "third edit"},
	}
	for _, update := range updates {
		if err := document.UpdateText(update.page, update.cell, update.value); err != nil {
			t.Fatal(err)
		}
	}
	if err := document.UpdateGeometry(6, "p6-n23", 95, 26, 120, 45); err != nil {
		t.Fatal(err)
	}

	serialized, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !document.Dirty() {
		t.Fatal("document is not dirty after multiple edits")
	}
	if !bytes.Equal(document.OriginalBytes(), source) {
		t.Fatal("multiple edits changed OriginalBytes")
	}
	if !bytes.Contains(serialized, payload) {
		t.Fatal("edit changed the compressed payload")
	}
	if !bytes.Contains(serialized, []byte("file-level=\"keep-me\"")) ||
		!bytes.Contains(serialized, []byte("<nested huge=\"123456789012345678901234567890\"/>")) {
		t.Fatal("edit changed unknown XML")
	}
	reparsed, _, err := Parse(serialized, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(reparsed.Pages) != sourceRetentionBenchmarkPages {
		t.Fatalf("reparsed pages = %d", len(reparsed.Pages))
	}
	last := reparsed.Pages[len(reparsed.Pages)-1]
	if !last.Compressed || !last.ReadOnly || findNode(last.Model.Nodes, "compressed-node") == nil {
		t.Fatalf("compressed page lost: %#v", last)
	}
	for _, update := range updates {
		node := findNode(reparsed.Pages[update.page].Model.Nodes, update.cell)
		if node == nil || node.Text != update.value {
			t.Fatalf("%s reparse = %#v", update.cell, node)
		}
	}
	node := findNode(reparsed.Pages[6].Model.Nodes, "p6-n23")
	if node == nil || node.X != 95 || node.Y != 26 || node.Width != 120 || node.Height != 45 {
		t.Fatalf("geometry reparse = %#v", node)
	}
	for index, value := range serialized {
		if value == '\n' && (index == 0 || serialized[index-1] != '\r') {
			t.Fatalf("CRLF changed at byte %d", index)
		}
	}
}

func TestSourceRetentionFailedReparseIsAtomic(t *testing.T) {
	document, issues, err := Parse(sourceRetentionBenchmarkSource(), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := document.UpdateText(0, "p0-n0", "before failure"); err != nil {
		t.Fatal(err)
	}
	before := cloneRetentionDocument(document)
	beforeBytes, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := findRetentionIssue(issues, IssueUnsupportedElement); !ok {
		t.Fatal("fixture has no expected parser warning")
	}

	err = document.rewriteAndReparse([]byte(`<mxfile><diagram></mxfile>`))
	if !errors.Is(err, ErrInvalidXML) {
		t.Fatalf("invalid reparse error = %v", err)
	}
	if !reflect.DeepEqual(before, *document) {
		t.Fatalf("failed reparse changed live state:\nbefore=%#v\nafter=%#v", before, *document)
	}

	if err := document.UpdateGeometry(0, "p0-n0", 41, 52, 130, 70); err != nil {
		t.Fatalf("edit after failed reparse: %v", err)
	}
	if !document.Dirty() {
		t.Fatal("document is not dirty after edit following failed reparse")
	}
	serialized, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(serialized, beforeBytes) {
		t.Fatal("successful edit after failed reparse did not change source")
	}
	reparsed, _, err := Parse(serialized, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	node := findNode(reparsed.Pages[0].Model.Nodes, "p0-n0")
	if node == nil || node.Text != "before failure" ||
		node.X != 41 || node.Y != 52 || node.Width != 130 || node.Height != 70 {
		t.Fatalf("reparsed node after failure = %#v", node)
	}
}

func TestSourceRetentionInputCapBeforeClone(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 8<<20)
	options := ParseOptions{MaxBytes: 1024}
	failed := false
	allocs := testing.AllocsPerRun(20, func() {
		document, issues, err := Parse(data, options)
		if document != nil || issues != nil || !errors.Is(err, ErrLimitExceeded) {
			failed = true
		}
	})
	if failed {
		t.Fatal("oversized input did not return the expected error")
	}
	if allocs != 3 {
		t.Fatalf("oversized input allocated %.2f times; want the three error-construction allocations", allocs)
	}
}

func cloneRetentionDocument(document *Document) Document {
	clone := *document
	clone.Pages = append([]Page(nil), document.Pages...)
	for pageIndex := range clone.Pages {
		clone.Pages[pageIndex].Model.Nodes = append([]Node(nil), clone.Pages[pageIndex].Model.Nodes...)
		for nodeIndex := range clone.Pages[pageIndex].Model.Nodes {
			clone.Pages[pageIndex].Model.Nodes[nodeIndex].Unsupported = append(
				[]string(nil),
				clone.Pages[pageIndex].Model.Nodes[nodeIndex].Unsupported...,
			)
		}
		clone.Pages[pageIndex].Model.Edges = append([]Edge(nil), clone.Pages[pageIndex].Model.Edges...)
		for edgeIndex := range clone.Pages[pageIndex].Model.Edges {
			edge := &clone.Pages[pageIndex].Model.Edges[edgeIndex]
			edge.Points = append([]Point(nil), edge.Points...)
			edge.Unsupported = append([]string(nil), edge.Unsupported...)
			if edge.SourcePoint != nil {
				point := *edge.SourcePoint
				edge.SourcePoint = &point
			}
			if edge.TargetPoint != nil {
				point := *edge.TargetPoint
				edge.TargetPoint = &point
			}
		}
	}
	clone.Issues = append([]Issue(nil), document.Issues...)
	clone.Source = cloneBytes(document.Source)
	clone.original = cloneBytes(document.original)
	clone.source = cloneBytes(document.source)
	return clone
}

func TestSourceRetentionPrivateOwnership(t *testing.T) {
	input := sourceRetentionBenchmarkSource()
	document, _, err := Parse(input, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if &document.original[0] != &document.source[0] {
		t.Fatal("fresh private sources do not share the owned buffer")
	}
	if &document.source[0] == &document.Source[0] {
		t.Fatal("public Source aliases private source")
	}
	if err := document.UpdateText(0, "p0-n0", "edited"); err != nil {
		t.Fatal(err)
	}
	if &document.original[0] == &document.source[0] {
		t.Fatal("edit discarded the pristine private buffer")
	}
	if &document.source[0] == &document.Source[0] {
		t.Fatal("edited public Source aliases private source")
	}
}

func findRetentionIssue(issues []Issue, code IssueCode) (Issue, bool) {
	for _, issue := range issues {
		if issue.Code == code {
			return issue, true
		}
	}
	return Issue{}, false
}
