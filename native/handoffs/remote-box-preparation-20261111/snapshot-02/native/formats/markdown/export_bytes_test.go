package markdown

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

const exportTestPrefix = "<!doctype html>\n<meta charset=\"utf-8\">\n<body>\n"
const exportTestSuffix = "</body>\n"

func requireExactExport(t *testing.T, doc Document, body string, wantWarnings []Warning) {
	t.Helper()
	want := exportTestPrefix + body + exportTestSuffix
	for _, limit := range []int{len(want), len(want) + 127} {
		got, warnings, err := ExportHTMLBytes(doc, limit)
		if err != nil || string(got) != want || len(got) != cap(got) || cap(got) > limit {
			t.Fatalf("byte export cap=%d: len/cap=%d/%d err=%v\ngot  %q\nwant %q", limit, len(got), cap(got), err, got, want)
		}
		if !reflect.DeepEqual(warnings, wantWarnings) {
			t.Fatalf("warnings=%#v want=%#v", warnings, wantWarnings)
		}
	}
	got, warnings, err := ExportHTMLBytes(doc, len(want)-1)
	if !errors.Is(err, ErrOutputLimit) || got != nil || warnings != nil {
		t.Fatalf("one byte short returned output: len=%d warnings=%#v err=%v", len(got), warnings, err)
	}
	text, warnings, err := ExportHTMLChecked(doc)
	if err != nil || text != want || !reflect.DeepEqual(warnings, wantWarnings) {
		t.Fatalf("legacy string API changed: output=%q warnings=%#v err=%v", text, warnings, err)
	}
}

func TestExportBytesExactSupportedBlocksAndEscapes(t *testing.T) {
	fixtures := []struct {
		name, source, body string
	}{
		{"empty", "", ""},
		{"UnicodeLiteralBOMCRLF", "\ufeff字<&>'\"\r\nx\r\n", "<p>\ufeff字&lt;&amp;&gt;&#39;&#34;\r\nx</p>\n"},
		{"headings", "# 字 Title\r\n\r\n## 字 Title\r\n", "<h1 id=\"字-title\">字 Title</h1>\n<h2 id=\"字-title-2\">字 Title</h2>\n"},
		{"inlines", "**&'<>\"** *字* `&'<>\"` plain&\n", "<p><strong>&amp;&#39;&lt;&gt;&#34;</strong> <em>字</em> <code>&amp;&#39;&lt;&gt;&#34;</code> plain&amp;</p>\n"},
		{"orderedTasks", "3. [x] &done\n4. [ ] later\n", "<ol start=\"3\">\n<li><input type=\"checkbox\" disabled checked> &amp;done</li>\n<li><input type=\"checkbox\" disabled> later</li>\n</ol>\n"},
		{"unorderedAndEmptyTask", "- plain&\n- [X]\n", "<ul>\n<li>plain&amp;</li>\n<li><input type=\"checkbox\" disabled checked></li>\n</ul>\n"},
		{"orderedOne", "1. first\n", "<ol>\n<li>first</li>\n</ol>\n"},
		{"quoteCRLF", "> 字&'<>\"\r\n", "<pre>&gt; 字&amp;&#39;&lt;&gt;&#34;\r\n</pre>\n"},
		{"codeCRLF", "```go\r\n字&'<>\"\r\n```\r\n", "<pre><code>字&amp;&#39;&lt;&gt;&#34;\r\n</code></pre>\n"},
		{"divider", "---\n", "<hr>\n"},
		{"literalTable", "| <&> | \"' | plain |\n| :--- | :---: | ---: |\n| **literal** | 字 | [x](y) |\n", "<table>\n<thead>\n<tr><th style=\"text-align:left\">&lt;&amp;&gt;</th><th style=\"text-align:center\">&#34;&#39;</th><th style=\"text-align:right\">plain</th></tr>\n</thead>\n<tbody>\n<tr><td style=\"text-align:left\">**literal**</td><td style=\"text-align:center\">字</td><td style=\"text-align:right\">[x](y)</td></tr>\n</tbody>\n</table>\n"},
		{"headerOnlyTable", "| A |\n| --- |\n", "<table>\n<thead>\n<tr><th>A</th></tr>\n</thead>\n</table>\n"},
		{"docBlockRaw", "```doc-block\nid: a\nkind: markdown\nmode: inline\n---\n<&>\n```\n", "<pre>```doc-block\nid: a\nkind: markdown\nmode: inline\n---\n&lt;&amp;&gt;\n```\n</pre>\n"},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			doc := Parse(fixture.source)
			if doc.Partial || len(doc.Warnings) != 0 || doc.Raw() != fixture.source {
				t.Fatalf("unexpected fixture parse: partial=%v warnings=%#v", doc.Partial, doc.Warnings)
			}
			requireExactExport(t, doc, fixture.body, []Warning{})
		})
	}
	// BlockHTML is a supported descriptor even though the current parser often
	// keeps raw HTML in paragraphs. Exercise its own raw-text branch explicitly.
	requireExactExport(t, Document{Blocks: []Block{{Kind: BlockHTML, Text: "<&>'\"字"}}}, "<pre>&lt;&amp;&gt;&#39;&#34;字</pre>\n", []Warning{})
}

func TestExportBytesSafeURLsAndWarningOrder(t *testing.T) {
	for _, fixture := range []struct {
		href, escaped string
		safe          bool
	}{
		{"", "", true}, {"#字&'<>\"", "#字&amp;&#39;&lt;&gt;&#34;", true},
		{"mailto:a@b.test?subject=<&>", "mailto:a@b.test?subject=&lt;&amp;&gt;", true},
		{" ./file.md?a=1&b=\"x\" ", "./file.md?a=1&amp;b=&#34;x&#34;", true},
		{"https://example.com", "", false}, {"//example.com", "", false},
		{"javascript:alert(1)", "", false}, {"C:\\file.md", "", false}, {"%zz", "", false},
	} {
		t.Run(fixture.href, func(t *testing.T) {
			const text = "prefix& LINK tail<"
			doc := Document{Blocks: []Block{{Kind: BlockParagraph, Text: text, TextRange: rangeAt(10, 10+len(text)), Inline: []InlineSpan{{Kind: InlineLink, Range: rangeAt(18, 22), Text: "<&>'\"字", Href: fixture.href}}}}}
			body := "<p>prefix&amp; "
			warnings := []Warning{}
			if fixture.safe {
				body += "<a href=\"" + fixture.escaped + "\">&lt;&amp;&gt;&#39;&#34;字</a>"
			} else {
				body += "&lt;&amp;&gt;&#39;&#34;字"
				warnings = append(warnings, Warning{Code: "unsafe-link", Message: "link was not emitted because its URL is unsafe; label text remains", Range: rangeAt(18, 22)})
			}
			requireExactExport(t, doc, body+" tail&lt;</p>\n", warnings)
		})
	}

	const raw = "| <&> |\n"
	parsedWarning := Warning{Code: "already-parsed", Message: "retained", Range: rangeAt(0, 1)}
	doc := Document{source: raw, Warnings: []Warning{parsedWarning}, Blocks: []Block{
		{Kind: BlockParagraph, Text: "<&>", TextRange: rangeAt(4, 7), Inline: []InlineSpan{{Kind: InlineBold, Range: rangeAt(3, 6), Text: "bad"}}},
		{Kind: BlockParagraph, Text: "link", TextRange: rangeAt(0, 4), Inline: []InlineSpan{{Kind: InlineLink, Range: rangeAt(0, 4), Text: "bad&", Href: "https://x"}}},
		{Kind: BlockTable, Range: rangeAt(0, len(raw)), Raw: raw},
	}}
	wantWarnings := []Warning{parsedWarning,
		{Code: "export-range", Message: "inline span does not match source; text was exported without formatting", Range: rangeAt(4, 7)},
		{Code: "unsafe-link", Message: "link was not emitted because its URL is unsafe; label text remains", Range: rangeAt(0, 4)},
		{Code: "export-table", Message: "table descriptor is invalid; complete table source was exported as raw text", Range: rangeAt(0, len(raw))},
	}
	requireExactExport(t, doc, "<p>&lt;&amp;&gt;</p>\n<p>bad&amp;</p>\n<pre>| &lt;&amp;&gt; |\n</pre>\n", wantWarnings)
	if !reflect.DeepEqual(doc.Warnings, []Warning{parsedWarning}) {
		t.Fatal("export mutated parser warnings")
	}
	// An explicitly diagnosed table has already reported its parser warning.
	doc.Blocks = []Block{{Kind: BlockTable, Raw: raw, Table: TableRef{Problem: "retained literal"}}}
	requireExactExport(t, doc, "<pre>| &lt;&amp;&gt; |\n</pre>\n", []Warning{parsedWarning})
	// Unknown inline descriptors preserve their original source range literally.
	requireExactExport(t, Document{Blocks: []Block{{Kind: BlockParagraph, Text: "<&>", Inline: []InlineSpan{{Kind: "future", Range: rangeAt(0, 3), Text: "ignored"}}}}}, "<p>&lt;&amp;&gt;</p>\n", []Warning{})
}

func TestExportBytesBudgetOwnershipAndPartialPrecedence(t *testing.T) {
	for _, limit := range []int{-1, 0, len(exportTestPrefix+exportTestSuffix) - 1} {
		data, warnings, err := ExportHTMLBytes(Document{}, limit)
		if !errors.Is(err, ErrOutputLimit) || data != nil || warnings != nil {
			t.Fatalf("empty envelope budget%d: data=%v warnings=%v err=%v", limit, data, warnings, err)
		}
	}
	doc := Parse("# same\n\n# same\n")
	first, _, err := ExportHTMLBytes(doc, 4096)
	if err != nil {
		t.Fatal(err)
	}
	second, _, _ := ExportHTMLBytes(doc, 4096)
	first[0] = '!'
	if second[0] != '<' || doc.Raw() != "# same\n\n# same\n" {
		t.Fatal("result storage aliases another export or the source")
	}
	partial := Document{Partial: true, Warnings: []Warning{{Code: "parse-limit", Message: "bounded", Range: rangeAt(0, 1)}}}
	data, warnings, err := ExportHTMLBytes(partial, -1)
	if data != nil || !errors.Is(err, ErrPartialDocument) || !reflect.DeepEqual(warnings, partial.Warnings) {
		t.Fatal("partial refusal or warning copy changed")
	}
	warnings[0].Code = "mutated"
	if partial.Warnings[0].Code != "parse-limit" {
		t.Fatal("partial warnings alias document")
	}
}

func TestExportSinkRefusesWithoutResultGrowthAndStops(t *testing.T) {
	for _, text := range []string{strings.Repeat("plain", 1<<16), strings.Repeat("&", 1<<18)} {
		count := htmlSink{limit: 4}
		count.escaped(text)
		if !errors.Is(count.err, ErrOutputLimit) || count.written != 0 || count.bytes != nil || count.text != nil {
			t.Fatal("obvious oversized raw fragment advanced or allocated output")
		}
		count.WriteString("x")
		if count.written != 0 {
			t.Fatal("failed sink continued")
		}
	}
	// Raw bytes fit but entity expansion does not. No output allocation is made
	// in count mode; failed fragment counting commits no partial count.
	count := htmlSink{limit: 6}
	count.escaped("&&&&&&")
	if !errors.Is(count.err, ErrOutputLimit) || count.written != 0 || count.bytes != nil {
		t.Fatalf("expansion refusal count=%d err=%v", count.written, count.err)
	}
	// Remaining-budget subtraction must not overflow even at the int boundary.
	count = htmlSink{limit: int(^uint(0) >> 1), written: int(^uint(0)>>1) - 1}
	count.WriteString("xx")
	if !errors.Is(count.err, ErrOutputLimit) || count.written != int(^uint(0)>>1)-1 {
		t.Fatal("counter overflowed")
	}
	// A later heading with large text would require anchor work. The rejected
	// first block stops traversal; its partial count contains only the envelope
	// and opening pre/code tag, no partial result allocation.
	doc := Document{Blocks: []Block{{Kind: BlockCode, Text: strings.Repeat("&", 1<<18)}, {Kind: BlockHeading, Text: "later", Level: 1}}}
	count = htmlSink{limit: len(exportTestPrefix) + 20}
	renderHTML(&count, doc, nil)
	if !errors.Is(count.err, ErrOutputLimit) || count.bytes != nil || count.written != len(exportTestPrefix)+len("<pre><code>") {
		t.Fatalf("render continued after limit: written=%d err=%v", count.written, count.err)
	}
	data, warnings, err := ExportHTMLBytes(doc, len(exportTestPrefix)+20)
	if !errors.Is(err, ErrOutputLimit) || data != nil || warnings != nil {
		t.Fatal("failed byte export exposed partial result")
	}
}
