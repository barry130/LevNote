package markdown

import (
	"errors"
	"strings"
	"testing"
)

func TestParseInlineOffsetsAndText(t *testing.T) {
	const source = "**bold** [link](https://example.com)"
	doc := Parse(source)
	if len(doc.Blocks) != 1 || doc.Blocks[0].Kind != BlockParagraph {
		t.Fatalf("expected one paragraph, got %#v", doc.Blocks)
	}
	spans := doc.Blocks[0].Inline
	if len(spans) != 2 {
		t.Fatalf("expected two spans, got %#v", spans)
	}
	if spans[0].Kind != InlineBold || spans[0].Text != "bold" {
		t.Fatalf("bold span = %#v", spans[0])
	}
	if source[spans[0].Range.Start:spans[0].Range.End] != "**bold**" {
		t.Fatalf("bold source range = %q", source[spans[0].Range.Start:spans[0].Range.End])
	}
	if spans[1].Kind != InlineLink || spans[1].Text != "link" || spans[1].Href != "https://example.com" {
		t.Fatalf("link span = %#v", spans[1])
	}
	if source[spans[1].Range.Start:spans[1].Range.End] != "[link](https://example.com)" {
		t.Fatalf("link source range = %q", source[spans[1].Range.Start:spans[1].Range.End])
	}
}

func TestParserLimitsPreserveRawSourceAndRefusePartialExport(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		code     string
		blocks   int
		complete bool
	}{
		{name: "line-limit", source: strings.Repeat("a\n", maxLines+1), code: "parse-limit", blocks: 1},
		{name: "newline-line-limit", source: strings.Repeat("\n", maxLines+1), code: "parse-limit"},
		{name: "pipe-dense-limit", source: strings.Repeat("|", 256<<10), code: "unsupported-table", blocks: 1, complete: true},
		{name: "exact-line-limit", source: strings.Repeat("a\n", maxLines), code: ""},
		{name: "block-limit", source: strings.Repeat("# a\n", maxBlocks+1), code: "parse-limit", blocks: maxBlocks},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			doc := Parse(test.source)
			if doc.Raw() != test.source {
				t.Fatal("raw source changed")
			}
			if test.code == "" {
				if doc.Partial || len(doc.Warnings) != 0 {
					t.Fatalf("exact limit marked partial: %#v", doc.Warnings)
				}
				return
			}
			if !doc.Partial && !test.complete {
				t.Fatalf("limit not marked partial: blocks=%d warnings=%#v", len(doc.Blocks), doc.Warnings)
			}
			if test.complete && doc.Partial {
				t.Fatalf("complete source marked partial: %#v", doc.Warnings)
			}
			if len(doc.Blocks) > test.blocks || len(doc.Warnings) > warningLimit {
				t.Fatalf("limits exceeded: blocks=%d warnings=%d", len(doc.Blocks), len(doc.Warnings))
			}
			if !warningCodePresent(doc.Warnings, test.code) {
				t.Fatalf("missing %q warning: %#v", test.code, doc.Warnings)
			}
			_, _, err := ExportHTMLChecked(doc)
			if test.complete {
				if errors.Is(err, ErrPartialDocument) {
					t.Fatalf("complete pipe source rejected: %v", err)
				}
			} else if !errors.Is(err, ErrPartialDocument) {
				t.Fatalf("partial export accepted: %v", err)
			}
		})
	}
}

func warningCodePresent(warnings []Warning, code string) bool {
	for _, warning := range warnings {
		if warning.Code == code {
			return true
		}
	}
	return false
}

func TestUnsupportedSyntaxRemainsVisibleAndWarns(t *testing.T) {
	source := "![image](a.png)\n\n| a | b |\n| - | --- |\n\n$$math$$\n\n```mermaid\ngraph TD\n```"
	doc := Parse(source)
	if doc.Raw() != source {
		t.Fatal("source changed")
	}
	codes := make(map[string]bool)
	for _, warning := range doc.Warnings {
		codes[warning.Code] = true
	}
	for _, code := range []string{"image-reference", "unsupported-table", "unsupported-math", "unsupported-syntax"} {
		if !codes[code] {
			t.Errorf("missing warning %q; got %#v", code, doc.Warnings)
		}
	}
}

func TestFootnoteAndImageOffsetsMakeProgress(t *testing.T) {
	source := "前缀 [^1] 中段 ![图](图.png) 后段 ![未闭合"
	doc := Parse(source)
	if doc.Raw() != source {
		t.Fatal("source changed")
	}
	codes := map[string]bool{}
	for _, warning := range doc.Warnings {
		codes[warning.Code] = true
	}
	if !codes["footnote"] || !codes["unsupported-image"] {
		t.Fatalf("footnote/image warnings: %#v", doc.Warnings)
	}
	// Repeated unmatched markers exercise forward progress without a timeout.
	_ = Parse(strings.Repeat("[^ ![", 10000))
}

func TestCRLFLineContentExcludesCarriageReturn(t *testing.T) {
	source := "# 标题\r\n\r\n```go\r\n中文\r\n```\r\n\r\n后续段落"
	doc := Parse(source)
	if doc.Raw() != source {
		t.Fatal("source changed")
	}
	if len(doc.Blocks) != 3 {
		t.Fatalf("blocks = %#v", doc.Blocks)
	}
	if doc.Blocks[0].Kind != BlockHeading || doc.Blocks[0].Text != "标题" {
		t.Fatalf("CRLF heading = %#v", doc.Blocks[0])
	}
	if doc.Blocks[1].Kind != BlockCode || doc.Blocks[1].Text != "中文\r\n" || doc.Blocks[1].Language != "go" {
		t.Fatalf("CRLF fence = %#v", doc.Blocks[1])
	}
	if doc.Blocks[2].Kind != BlockParagraph || doc.Blocks[2].Text != "后续段落" {
		t.Fatalf("paragraph after CRLF fence = %#v", doc.Blocks[2])
	}
}

func TestListRangesAndEmptyItemsDoNotLoop(t *testing.T) {
	source := "- hello\n- \n\n# 标题\n\n1. 第一项"
	doc := Parse(source)
	if doc.Raw() != source {
		t.Fatal("source changed")
	}
	if len(doc.Blocks) != 3 {
		t.Fatalf("blocks = %#v", doc.Blocks)
	}
	list := doc.Blocks[0]
	if list.Kind != BlockList || len(list.Items) != 2 || list.Items[0].Text != "hello" {
		t.Fatalf("bullet list = %#v", list)
	}
	if source[list.Items[1].TextRange.Start:list.Items[1].TextRange.End] != "" {
		t.Fatalf("empty item range = %#v", list.Items[1])
	}
	if len(list.Items[1].Inline) != 0 || len(doc.Warnings) != 0 {
		t.Fatalf("empty list item = %#v warnings=%#v", list.Items[1], doc.Warnings)
	}
	if doc.Blocks[1].Kind != BlockHeading || doc.Blocks[1].Text != "标题" {
		t.Fatalf("heading after list = %#v", doc.Blocks[1])
	}
	ordered := doc.Blocks[2]
	if ordered.Kind != BlockList || !ordered.Ordered || len(ordered.Items) != 1 || ordered.Items[0].Text != "第一项" {
		t.Fatalf("ordered list = %#v", ordered)
	}
}

func TestExportUsesExactSourceRangesInLaterBlocks(t *testing.T) {
	source := "**粗**前\n\n后面 **again** and *again* `code` [链接](file.md)\n\n__下划线__ ```多``反引号```"
	doc := Parse(source)
	html, warnings := ExportHTML(doc)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v", warnings)
	}
	for _, expected := range []string{"<strong>粗</strong>前", "后面 <strong>again</strong> and <em>again</em>", "<code>code</code>", `<a href="file.md">链接</a>`, "<strong>下划线</strong>", "<code>多``反引号</code>"} {
		if !strings.Contains(html, expected) {
			t.Fatalf("missing %q in %s", expected, html)
		}
	}
}

func TestDocBlockPreservedByteForByte(t *testing.T) {
	source := "# Title\n\n```doc-block\nid: demo\nkind: diagram\nmode: reference\nsrc: demo.drawio\ntitle: Demo\nfuture: kept\n---\npayload\n```"
	doc := Parse(source)
	if doc.Raw() != source {
		t.Fatal("source changed")
	}
	found := false
	for _, block := range doc.Blocks {
		if block.Kind != BlockDocBlock {
			continue
		}
		found = true
		if block.DocBlock.Attrs.ID != "demo" || block.DocBlock.Attrs.Kind != "diagram" {
			t.Fatalf("attrs = %#v", block.DocBlock.Attrs)
		}
		if block.DocBlock.Attrs.Extra["future"] != "kept" {
			t.Fatalf("extra = %#v", block.DocBlock.Attrs.Extra)
		}
		if block.DocBlock.Payload != "payload\n" {
			t.Fatalf("payload = %q", block.DocBlock.Payload)
		}
		if block.Raw != "```doc-block\nid: demo\nkind: diagram\nmode: reference\nsrc: demo.drawio\ntitle: Demo\nfuture: kept\n---\npayload\n```" {
			t.Fatalf("raw = %q", block.Raw)
		}
	}
	if !found {
		t.Fatal("doc-block was not parsed")
	}
}

func TestClosedFenceConsumesClosingLineAndProseContinues(t *testing.T) {
	source := "```go\r\n中文\r\n```\r\n\r\n后续段落\r\n"
	doc := Parse(source)
	if doc.Raw() != source {
		t.Fatal("source changed")
	}
	if len(doc.Blocks) != 2 {
		t.Fatalf("blocks = %#v", doc.Blocks)
	}
	code := doc.Blocks[0]
	if code.Kind != BlockCode || code.Text != "中文\r\n" || code.Language != "go" {
		t.Fatalf("code block = %#v", code)
	}
	if source[code.Range.Start:code.Range.End] != "```go\r\n中文\r\n```\r\n" {
		t.Fatalf("fence range = %q", source[code.Range.Start:code.Range.End])
	}
	paragraph := doc.Blocks[1]
	if paragraph.Kind != BlockParagraph || paragraph.Text != "后续段落" {
		t.Fatalf("paragraph after CRLF fence = %#v", paragraph)
	}
}

func TestUnterminatedFenceConsumesRemainingBlockSyntax(t *testing.T) {
	source := "```text\n中文\n# Heading\n- item\n"
	doc := Parse(source)
	if doc.Raw() != source {
		t.Fatal("source changed")
	}
	if len(doc.Blocks) != 1 {
		t.Fatalf("blocks = %#v", doc.Blocks)
	}
	code := doc.Blocks[0]
	if code.Kind != BlockCode || code.Text != "中文\n# Heading\n- item\n" {
		t.Fatalf("unterminated fence = %#v", code)
	}
	if len(doc.Warnings) == 0 || doc.Warnings[0].Code != "parse" {
		t.Fatalf("warnings = %#v", doc.Warnings)
	}
}

func TestClosingFenceTrailingSpacesAreAcceptedAndPreserved(t *testing.T) {
	source := "```\nbody\n```   \nprose"
	doc := Parse(source)
	if doc.Raw() != source {
		t.Fatal("source changed")
	}
	if len(doc.Blocks) != 2 {
		t.Fatalf("blocks = %#v", doc.Blocks)
	}
	if doc.Blocks[0].Kind != BlockCode || doc.Blocks[0].Text != "body\n" {
		t.Fatalf("fence = %#v", doc.Blocks[0])
	}
	if doc.Blocks[1].Kind != BlockParagraph || doc.Blocks[1].Text != "prose" {
		t.Fatalf("prose = %#v", doc.Blocks[1])
	}
}

func TestOrderedListStartAndMixedKindsStop(t *testing.T) {
	source := "3. third\n4. fourth\n- bullet\n"
	doc := Parse(source)
	if doc.Raw() != source {
		t.Fatal("source changed")
	}
	if len(doc.Blocks) != 2 {
		t.Fatalf("blocks = %#v", doc.Blocks)
	}
	list := doc.Blocks[0]
	if list.Kind != BlockList || !list.Ordered || list.Start != 3 || len(list.Items) != 2 {
		t.Fatalf("ordered list = %#v", list)
	}
	if source[list.Items[0].TextRange.Start:list.Items[0].TextRange.End] != "third" {
		t.Fatalf("ordered item = %#v", list.Items[0])
	}
	next := doc.Blocks[1]
	if next.Kind != BlockList || next.Ordered || len(next.Items) != 1 || next.Items[0].Text != "bullet" {
		t.Fatalf("mixed kind continuation = %#v", next)
	}
	html, warnings, err := ExportHTMLChecked(doc)
	if err != nil || len(warnings) != 0 {
		t.Fatal(html, warnings)
	}
	if !strings.Contains(html, `<ol start="3">`) {
		t.Fatalf("ordered list start was not exported: %s", html)
	}
}

func TestRepeatedInlineScanExhaustionIsBoundedAndDisclosed(t *testing.T) {
	budget := &inlineLimit{}
	parser := &inlineParser{source: strings.Repeat("[", 500000), spans: []InlineSpan{}, warnings: []Warning{}, limit: budget}
	parser.warnings = parser.parse()
	if parser.exhausted != true || parser.scanSpent > maxInlineScanBytes {
		t.Fatalf("scan budget not bounded: spent=%d exhausted=%v warnings=%d", parser.scanSpent, parser.exhausted, len(parser.warnings))
	}
	if count := len(parser.warnings); count == 0 || count > warningLimit {
		t.Fatalf("warnings = %d", count)
	}
	budgetWarnings := 0
	for _, warning := range parser.warnings {
		if warning.Code == "scan-budget" {
			budgetWarnings++
		}
	}
	if budgetWarnings != 1 {
		t.Fatalf("budget warnings = %d", budgetWarnings)
	}

	source := "prefix " + strings.Repeat("[", 500000) + " end"
	doc := Parse(source)
	if doc.Raw() != source {
		t.Fatal("source changed")
	}
	budgetWarnings = 0
	for _, warning := range doc.Warnings {
		if warning.Code == "scan-budget" {
			budgetWarnings++
		}
	}
	if budgetWarnings != 1 || len(doc.Warnings) > warningLimit {
		t.Fatalf("document scan warnings = %#v", doc.Warnings)
	}
	if !doc.Partial {
		t.Fatal("document inline budget exhaustion was not marked partial")
	}
	if _, _, err := ExportHTMLChecked(doc); !errors.Is(err, ErrPartialDocument) {
		t.Fatalf("partial export accepted: %v", err)
	}
}

func TestEmptyInlineScanDoesNotWarnForShortPlainParagraph(t *testing.T) {
	doc := Parse("plain text with no inline syntax")
	if len(doc.Warnings) != 0 {
		t.Fatalf("warnings = %#v", doc.Warnings)
	}
}

func TestExportEscapesHTMLAndBlocksNetworkLinks(t *testing.T) {
	source := "**<script>** [external](https://example.com) [relative](./x.md)"
	doc := Parse(source)
	html, _ := ExportHTML(doc)
	if strings.Contains(html, "<script>") {
		t.Fatal("script was not escaped")
	}
	if strings.Contains(html, "href=\"https://example.com\"") {
		t.Fatal("external network link was emitted")
	}
	if !strings.Contains(html, "href=\"./x.md\"") {
		t.Fatal("relative link was not emitted")
	}
}

func TestExportReportsParserWarningsAndUnsafeLinks(t *testing.T) {
	source := "[外部](//example.com) ![图](图.png)"
	doc := Parse(source)
	exported, warnings := ExportHTML(doc)
	if strings.Contains(exported, "<a ") || strings.Contains(exported, "href=") {
		t.Fatalf("unsafe link was emitted: %s", exported)
	}
	codes := map[string]bool{}
	for _, warning := range warnings {
		codes[warning.Code] = true
	}
	if !codes["unsafe-link"] || !codes["image-reference"] {
		t.Fatalf("warnings = %#v", warnings)
	}
}
