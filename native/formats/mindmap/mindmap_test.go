package mindmap

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const richMindmap = "\r\n" + `{
  "futureTop": {"huge": 123456789012345678901234567890123456789, "decimal": 1.00},
  "root" : {
    "topic":"根",
    "unknownNode":123456789012345678901234567890,
    "style": {"opacity": 0.50, "custom": {"n": 9007199254740993}},
    "icons": ["1️⃣", "⭐"],
    "image": {"url":"a.png","width":10.00,"height":20,"fit":"cover","futureImage":3.1400},
    "hyperLink":"notes/a.mindmap",
    "tags":["重要",{"text":"风险","className":"x","style":{"tone":"warn","n":1.50},"futureTag":2.000}],
    "branchColor":"#209fb5",
    "children":[
      {"id":"left","topic":"左","collapsed":true,"children":[{"id":"hidden","topic":"隐藏","children":[]}]},
      {"id":"right","topic":"右","children":[]}
    ],
    "id":"root"
  },
  "layout":"logicalStructure",
  "version":1,
  "format":"levnote.mindmap"
}
`

func TestUntouchedRoundTripIsExact(t *testing.T) {
	document, _, err := Parse([]byte(richMindmap), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	serialized, err := document.Serialize(SerializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(serialized, []byte(richMindmap)) {
		t.Fatal("untouched serialization changed bytes")
	}
	if document.Dirty() {
		t.Fatal("fresh document reported dirty")
	}
}

func TestRenamePreservesUnknownValuesAndNumberLexemes(t *testing.T) {
	document, _, err := Parse([]byte(richMindmap), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := document.RenameNode("root", "新根"); err != nil {
		t.Fatal(err)
	}
	serialized, err := document.Serialize(SerializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	text := string(serialized)
	for _, exact := range []string{
		"123456789012345678901234567890123456789",
		"123456789012345678901234567890",
		"9007199254740993",
		"3.1400",
		"1.50",
		"2.000",
		"futureTop",
		"unknownNode",
		"futureImage",
		"futureTag",
	} {
		if !strings.Contains(text, exact) {
			t.Fatalf("serialized output lost %q:\n%s", exact, text)
		}
	}
	again, _, err := Parse(serialized, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if again.Root.Topic != "新根" {
		t.Fatalf("topic = %q", again.Root.Topic)
	}
	if again.Root.Image == nil || again.Root.Image.Width.String() != "10.00" {
		t.Fatalf("image width lexeme changed: %#v", again.Root.Image)
	}
}

func TestMetadataIsExposed(t *testing.T) {
	document, _, err := Parse([]byte(richMindmap), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	root := document.Root
	if root.Style == nil || root.Icons == nil || len(*root.Icons) != 2 {
		t.Fatal("style or icons missing")
	}
	if root.Image == nil || root.Image.Fit != "cover" {
		t.Fatal("image missing")
	}
	if root.HyperLink == nil || *root.HyperLink != "notes/a.mindmap" {
		t.Fatal("hyperlink missing")
	}
	if root.Tags == nil || len(*root.Tags) != 2 || !(*root.Tags)[1].Object {
		t.Fatal("tags missing")
	}
	if root.BranchColor == nil || *root.BranchColor != "#209fb5" {
		t.Fatal("branch color missing")
	}
}

func TestEmptyTagObjectMembersStayPresent(t *testing.T) {
	input := `{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"R","tags":[{"text":"T","className":"","style":{}}],"children":[]}}`
	document, _, err := Parse([]byte(input), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := document.RenameNode("r", "R2"); err != nil {
		t.Fatal(err)
	}
	serialized, err := document.Serialize(SerializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(serialized), `"className": ""`) ||
		!strings.Contains(string(serialized), `"style": {}`) {
		t.Fatalf("empty tag members disappeared:\n%s", serialized)
	}
}

func TestStructuralEditsAndRootException(t *testing.T) {
	document, _, err := Parse([]byte(richMindmap), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	added := NewNode("new", "新增")
	if _, err := document.AddNode("root", 1, added); err != nil {
		t.Fatal(err)
	}
	if err := document.MoveNode("new", "right", -1); err != nil {
		t.Fatal(err)
	}
	if err := document.MoveNode("right", "new", 0); !errors.Is(err, ErrCycle) {
		t.Fatalf("expected cycle error, got %v", err)
	}
	if err := document.MoveNode("root", "right", 0); !errors.Is(err, ErrRootImmutable) {
		t.Fatalf("expected root immovable, got %v", err)
	}
	if _, err := document.DeleteNode("root"); !errors.Is(err, ErrRootImmutable) {
		t.Fatalf("expected root undeletable, got %v", err)
	}
	if _, err := document.AddNode("root", -1, NewNode("right", "duplicate")); !errors.Is(err, ErrDuplicateID) {
		t.Fatalf("expected duplicate id, got %v", err)
	}
	if _, err := document.DeleteNode("new"); err != nil {
		t.Fatal(err)
	}
}

func TestBadShapesAndDuplicateIDsRejectWholeFile(t *testing.T) {
	tests := []string{
		`{"format":"levnote.mindmap","version":1,"root":{"id":"a","topic":"A","children":[]},"root":{"id":"b","topic":"B","children":[]}}`,
		`{"format":"levnote.mindmap","version":1,"root":{"id":"a","topic":"A","children":[]},"root":{"id":"a","topic":"A","children":[]}}`,
		`{"format":"levnote.mindmap","version":1,"root":{"id":"a","topic":1,"children":[]}}`,
		`{"format":"levnote.mindmap","version":1,"root":{"id":"a","topic":"A","children":"bad"}}`,
		`{"format":"levnote.mindmap","version":1,"root":{"id":"a","topic":"A","icons":"bad"}}`,
		`{"format":"levnote.mindmap","version":1,"root":{"id":"a","topic":"A","image":{"url":"x","width":0,"height":1}}}`,
	}
	for _, input := range tests {
		if _, _, err := Parse([]byte(input), ParseOptions{}); err == nil {
			t.Fatalf("accepted invalid input: %s", input)
		}
	}

	duplicate := `{"format":"levnote.mindmap","version":1,"root":{"id":"same","topic":"A","children":[{"id":"same","topic":"B","children":[]}]}}`
	if _, _, err := Parse([]byte(duplicate), ParseOptions{}); !errors.Is(err, ErrDuplicateID) {
		t.Fatalf("expected duplicate id error, got %v", err)
	}
}

func TestLegacyQingmoIsRejected(t *testing.T) {
	input := `{"format":"qingmo.mindmap","version":1,"root":{"id":"a","topic":"A","children":[]}}`
	_, issues, err := Parse([]byte(input), ParseOptions{})
	if err == nil {
		t.Fatal("qingmo format was accepted")
	}
	if len(issues) != 1 || issues[0].Code != IssueLegacyFormat {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestLayoutModesAndCollapse(t *testing.T) {
	document, _, err := Parse([]byte(richMindmap), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{LayoutLeft, LayoutRight, LayoutBoth, LayoutOrg} {
		if err := document.SetLayout(kind); err != nil {
			t.Fatal(err)
		}
		result := Layout(document, LayoutOptions{})
		if result.Kind != kind || len(result.Nodes) == 0 || result.Bounds.Width <= 0 || result.Bounds.Height <= 0 {
			t.Fatalf("%s layout invalid: %#v", kind, result)
		}
		for _, node := range result.Nodes {
			if node.ID == "hidden" {
				t.Fatalf("%s rendered a child under a collapsed node", kind)
			}
		}
	}
}

func TestParseLimits(t *testing.T) {
	input := []byte(`{"format":"levnote.mindmap","version":1,"root":{"id":"a","topic":"A","children":[]}}`)
	if _, _, err := Parse(input, ParseOptions{MaxBytes: 10}); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("expected max bytes, got %v", err)
	}
	if _, _, err := Parse(input, ParseOptions{MaxNodes: 1, MaxDepth: 2}); err == nil {
		// One node fits, so this should succeed. Keep the check paired with a
		// second child below.
		nested := []byte(`{"format":"levnote.mindmap","version":1,"root":{"id":"a","topic":"A","children":[{"id":"b","topic":"B","children":[]}]}}`)
		if _, _, err := Parse(nested, ParseOptions{MaxNodes: 1}); !errors.Is(err, ErrLimitExceeded) {
			t.Fatalf("expected node limit, got %v", err)
		}
	}
}

func TestDirectModelChangeIsDetected(t *testing.T) {
	document, _, err := Parse([]byte(richMindmap), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	document.Root.Topic = "changed outside a method"
	serialized, err := document.Serialize(SerializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(serialized, []byte(richMindmap)) || !strings.Contains(string(serialized), "changed outside a method") {
		t.Fatal("direct model change was silently ignored")
	}
}
