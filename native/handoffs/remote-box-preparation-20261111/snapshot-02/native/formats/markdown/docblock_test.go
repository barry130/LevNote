package markdown

import (
	"strings"
	"testing"
)

func TestDocBlockMissingSeparatorAndEOFPreserveRawCode(t *testing.T) {
	sources := []string{
		"```doc-block",
		"```doc-block\n",
		"```doc-block\n```",
		"```doc-block\nid: demo\nkind: diagram\n",
		"```doc-block\r\nid: demo\r\nkind: diagram\r\npayload\r\n",
	}
	for _, source := range sources {
		doc := Parse(source)
		if doc.Raw() != source {
			t.Fatalf("raw source changed for %q", source)
		}
		if len(doc.Blocks) != 1 {
			t.Fatalf("blocks for %q = %#v", source, doc.Blocks)
		}
		block := doc.Blocks[0]
		if block.Kind != BlockCode || block.Text != source || block.Raw != source ||
			block.Range.Start != 0 || block.Range.End != len(source) {
			t.Fatalf("fallback block for %q = %#v", source, block)
		}
		if !warningMessageContains(doc.Warnings, "doc-block separator is missing") {
			t.Fatalf("warnings for %q = %#v", source, doc.Warnings)
		}
		if source != "```doc-block\n```" &&
			!warningMessageContains(doc.Warnings, "unterminated doc-block") {
			t.Fatalf("missing unterminated warning for %q: %#v", source, doc.Warnings)
		}
	}
}

func TestDocBlockClosedWithMissingSeparatorConsumesFenceAndContinues(t *testing.T) {
	source := "```doc-block\nid: demo\n```\n后续段落"
	doc := Parse(source)
	if doc.Raw() != source {
		t.Fatal("raw source changed")
	}
	if len(doc.Blocks) != 2 || doc.Blocks[0].Kind != BlockCode ||
		doc.Blocks[1].Kind != BlockParagraph {
		t.Fatalf("blocks = %#v", doc.Blocks)
	}
	block := doc.Blocks[0]
	if block.Text != "```doc-block\nid: demo\n```\n" ||
		block.Range.End != len("```doc-block\nid: demo\n```\n") {
		t.Fatalf("doc-block = %#v", block)
	}
	if doc.Blocks[1].Text != "后续段落" {
		t.Fatalf("prose after doc-block = %#v", doc.Blocks[1])
	}
	if !warningMessageContains(doc.Warnings, "doc-block separator is missing") {
		t.Fatalf("warnings = %#v", doc.Warnings)
	}
}

func TestDocBlockCRLFValidPayloadAndProseContinuation(t *testing.T) {
	source := "```doc-block\r\nid: demo\r\nkind: diagram\r\n---\r\npayload\r\n```\r\n\r\n后续段落\r\n"
	doc := Parse(source)
	if doc.Raw() != source {
		t.Fatal("raw source changed")
	}
	if len(doc.Blocks) != 2 || doc.Blocks[0].Kind != BlockDocBlock ||
		doc.Blocks[1].Kind != BlockParagraph {
		t.Fatalf("blocks = %#v", doc.Blocks)
	}
	block := doc.Blocks[0]
	if block.DocBlock.Attrs.ID != "demo" || block.DocBlock.Attrs.Kind != "diagram" ||
		block.DocBlock.Payload != "payload\r\n" {
		t.Fatalf("doc-block = %#v", block)
	}
	if source[block.Range.Start:block.Range.End] != "```doc-block\r\nid: demo\r\nkind: diagram\r\n---\r\npayload\r\n```\r\n" {
		t.Fatalf("doc-block range = %q", source[block.Range.Start:block.Range.End])
	}
	if doc.Blocks[1].Text != "后续段落" {
		t.Fatalf("prose after doc-block = %#v", doc.Blocks[1])
	}
}

func TestDocBlockHeaderRangeExactLFAndCRLF(t *testing.T) {
	sources := []string{
		"```doc-block\nid: demo\nkind: text\nmode: reference\nsrc: 目标.txt\n\nfuture: kept\n---\npayload\n```",
		"```doc-block\r\nid: demo\r\nkind: text\r\nmode: reference\r\nsrc: 目标.txt\r\n\r\nfuture: kept\r\n---\r\npayload\r\n```\r\n",
	}
	for _, source := range sources {
		doc := Parse(source)
		if len(doc.Blocks) != 1 || doc.Blocks[0].Kind != BlockDocBlock {
			t.Fatalf("blocks for %q = %#v", source, doc.Blocks)
		}
		ref := doc.Blocks[0].DocBlock
		header := source[ref.HeaderRange.Start:ref.HeaderRange.End]
		want := "id: demo\nkind: text\nmode: reference\nsrc: 目标.txt\n\nfuture: kept\n"
		if strings.Contains(source, "\r\n") {
			want = "id: demo\r\nkind: text\r\nmode: reference\r\nsrc: 目标.txt\r\n\r\nfuture: kept\r\n"
		}
		if header != want {
			t.Fatalf("header = %q want %q range=%#v", header, want, ref.HeaderRange)
		}
		if strings.Contains(header, "---") {
			t.Fatalf("header included separator: %q", header)
		}
		if ref.ReferenceProblem != "" {
			t.Fatalf("valid reference has problem %q", ref.ReferenceProblem)
		}
	}
}

func TestDocBlockReferenceProblemMatrix(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		contains   []string
		omits      []string
		problemSet bool
	}{
		{
			name:       "valid reference",
			source:     "```doc-block\nid: a\nkind: text\nmode: reference\nsrc: 目标.txt\n---\n```",
			problemSet: false,
		},
		{
			name:       "valid inline without src",
			source:     "```doc-block\nid: a\nkind: text\nmode: inline\n---\npayload\n```",
			problemSet: false,
		},
		{
			name:       "inline with src is not reference",
			source:     "```doc-block\nid: a\nkind: text\nmode: inline\nsrc: 目标.txt\n---\n```",
			contains:   []string{"src is not opened in inline mode"},
			problemSet: true,
		},
		{
			name:       "inline with explicitly empty src is not reference",
			source:     "```doc-block\nid: a\nkind: text\nmode: inline\nsrc:   \n---\n```",
			contains:   []string{"src is not opened in inline mode"},
			problemSet: true,
		},
		{
			name:       "unknown mode",
			source:     "```doc-block\nid: a\nkind: text\nmode: unknown\nsrc: 目标.txt\n---\n```",
			contains:   []string{"mode missing or unknown"},
			problemSet: true,
		},
		{
			name:       "missing mode",
			source:     "```doc-block\nid: a\nkind: text\nsrc: 目标.txt\n---\n```",
			contains:   []string{"mode missing or unknown"},
			problemSet: true,
		},
		{
			name:       "missing src",
			source:     "```doc-block\nid: a\nkind: text\nmode: reference\n---\n```",
			contains:   []string{"src missing or empty in reference mode"},
			problemSet: true,
		},
		{
			name:       "empty src",
			source:     "```doc-block\nid: a\nkind: text\nmode: reference\nsrc:   \n---\n```",
			contains:   []string{"src missing or empty in reference mode"},
			problemSet: true,
		},
		{
			name:       "duplicate recognized headers",
			source:     "```doc-block\nid: a\nid: b\nkind: text\nkind: text\nmode: reference\nmode: reference\nsrc: a.txt\nsrc: b.txt\ntitle: a\ntitle: b\n---\n```",
			contains:   []string{"duplicate id", "duplicate kind", "duplicate mode", "duplicate src", "duplicate title"},
			problemSet: true,
		},
		{
			name:       "unknown duplicate does not block reference",
			source:     "```doc-block\nid: a\nkind: text\nmode: reference\nsrc: 目标.txt\nfuture: one\nfuture: two\n---\n```",
			problemSet: false,
		},
		{
			name:       "missing id and kind",
			source:     "```doc-block\nmode: reference\nsrc: 目标.txt\n---\n```",
			contains:   []string{"id missing or invalid", "kind missing"},
			problemSet: true,
		},
		{
			name:       "invalid id",
			source:     "```doc-block\nid: -bad\nkind: text\nmode: reference\nsrc: 目标.txt\n---\n```",
			contains:   []string{"id missing or invalid"},
			problemSet: true,
		},
		{
			name:       "unterminated",
			source:     "```doc-block\nid: a\nkind: text\nmode: reference\nsrc: 目标.txt\n---\n",
			contains:   []string{"unterminated fence"},
			problemSet: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc := Parse(test.source)
			if doc.Raw() != test.source {
				t.Fatal("source changed")
			}
			if len(doc.Blocks) != 1 || doc.Blocks[0].Kind != BlockDocBlock {
				t.Fatalf("blocks = %#v", doc.Blocks)
			}
			problem := doc.Blocks[0].DocBlock.ReferenceProblem
			if (problem != "") != test.problemSet {
				t.Fatalf("problem = %q", problem)
			}
			for _, fragment := range test.contains {
				if !strings.Contains(problem, fragment) {
					t.Fatalf("problem %q missing %q", problem, fragment)
				}
			}
			for _, fragment := range test.omits {
				if strings.Contains(problem, fragment) {
					t.Fatalf("problem %q unexpectedly contains %q", problem, fragment)
				}
			}
			if len(problem) > maxReferenceProblemBytes {
				t.Fatalf("problem exceeds bound: %q", problem)
			}
		})
	}
}

func TestDocBlockIDWarningsDoNotDuplicate(t *testing.T) {
	tests := []struct {
		name, source string
		wantCode     string
	}{
		{
			name:     "invalid id uses useful syntax code",
			source:   "```doc-block\nid: -bad\nkind: text\nmode: reference\nsrc: 目标.txt\n---\n```",
			wantCode: "doc-block-bad-id",
		},
		{
			name:     "missing id uses parse warning",
			source:   "```doc-block\nkind: text\nmode: reference\nsrc: 目标.txt\n---\n```",
			wantCode: "parse",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc := Parse(test.source)
			if doc.Raw() != test.source {
				t.Fatal("source changed")
			}
			idWarnings := 0
			for _, warning := range doc.Warnings {
				if strings.Contains(warning.Message, "id is") ||
					warning.Code == "doc-block-bad-id" {
					idWarnings++
				}
			}
			if idWarnings != 1 {
				t.Fatalf("id warning count = %d: %#v", idWarnings, doc.Warnings)
			}
			if doc.Warnings[0].Code != test.wantCode {
				t.Fatalf("warning code = %q, want %q", doc.Warnings[0].Code, test.wantCode)
			}
		})
	}
}

func TestDocBlockReferenceProblemBoundedFromMalformedHeaders(t *testing.T) {
	// Every recognized key is duplicated many times. The real parser cannot
	// add more than five duplicate phrases, but this still proves source
	// values and arbitrary headers are not appended to the status string.
	var source strings.Builder
	source.WriteString("```doc-block\n")
	for i := 0; i < 100; i++ {
		source.WriteString("id: value-" + strings.Repeat("x", i%10) + "\n")
		source.WriteString("kind: text\n")
		source.WriteString("mode: reference\n")
		source.WriteString("src: target-" + strings.Repeat("y", i%10) + ".txt\n")
		source.WriteString("title: title-" + strings.Repeat("z", i%10) + "\n")
	}
	source.WriteString("unknown: one\nunknown: two\n---\n```\n")
	doc := Parse(source.String())
	if doc.Raw() != source.String() {
		t.Fatal("source changed")
	}
	if len(doc.Blocks) != 1 || doc.Blocks[0].Kind != BlockDocBlock {
		t.Fatalf("blocks = %#v", doc.Blocks)
	}
	ref := doc.Blocks[0].DocBlock
	if ref.ReferenceProblem == "" || len(ref.ReferenceProblem) > maxReferenceProblemBytes {
		t.Fatalf("reference problem = %q", ref.ReferenceProblem)
	}
	if strings.Contains(ref.ReferenceProblem, "target-") ||
		strings.Contains(ref.ReferenceProblem, "title-") ||
		strings.Contains(ref.ReferenceProblem, "unknown") {
		t.Fatalf("reference problem embeds source values: %q", ref.ReferenceProblem)
	}
	for _, fragment := range []string{"duplicate id", "duplicate kind", "duplicate mode", "duplicate src", "duplicate title"} {
		if !strings.Contains(ref.ReferenceProblem, fragment) {
			t.Fatalf("reference problem %q missing %q", ref.ReferenceProblem, fragment)
		}
	}
}

func TestDocBlockReferenceProblemBounded(t *testing.T) {
	// Fixed problem phrases only; the bound is deterministic because arbitrary
	// source values are never embedded.
	problems := make([]string, 100)
	for i := range problems {
		problems[i] = "duplicate id"
	}
	problem := boundedReferenceProblem(problems)
	if len(problem) > maxReferenceProblemBytes {
		t.Fatalf("problem exceeds bound: %d", len(problem))
	}
	if !strings.HasSuffix(problem, "…") {
		t.Fatalf("truncated problem has no ellipsis: %q", problem)
	}
}

func warningMessageContains(warnings []Warning, fragment string) bool {
	for _, warning := range warnings {
		if warning.Code == "parse" && strings.Contains(warning.Message, fragment) {
			return true
		}
	}
	return false
}
