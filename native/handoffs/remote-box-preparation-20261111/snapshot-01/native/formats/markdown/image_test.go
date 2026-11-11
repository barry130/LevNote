package markdown

import (
	"fmt"
	"strings"
	"testing"
)

func inlineSpansOfKind(doc Document, kind InlineKind) []InlineSpan {
	var result []InlineSpan
	for _, block := range doc.Blocks {
		for _, span := range block.Inline {
			if span.Kind == kind {
				result = append(result, span)
			}
		}
		for _, item := range block.Items {
			for _, span := range item.Inline {
				if span.Kind == kind {
					result = append(result, span)
				}
			}
		}
	}
	return result
}

func requireOneInlineImage(t *testing.T, source, label, href string) InlineSpan {
	t.Helper()
	doc := Parse(source)
	images := inlineSpansOfKind(doc, InlineImage)
	if doc.Raw() != source || len(images) != 1 {
		t.Fatalf("image spans=%#v warnings=%#v", images, doc.Warnings)
	}
	image := images[0]
	if image.Text != label || image.Href != href ||
		image.Range.Start < 0 || image.Range.End > len(source) {
		t.Fatalf("image=%#v source=%q", image, source)
	}
	warnings := 0
	for _, warning := range doc.Warnings {
		if warning.Code == "unsupported-image" {
			t.Fatalf("accepted image retained rejection warning: %#v", warning)
		}
		if warning.Code == "image-reference" {
			warnings++
			if warning.Message != inlineImageReferenceMessage || warning.Range != image.Range {
				t.Fatalf("reference warning=%#v image=%#v", warning, image)
			}
		}
	}
	if warnings != 1 {
		t.Fatalf("reference warning count=%d warnings=%#v", warnings, doc.Warnings)
	}
	return image
}

func TestInlineImageExactFieldsRangesAndContexts(t *testing.T) {
	fixtures := []struct {
		name, source, label, href, syntax string
	}{
		{"paragraph", "prefix ![图](dir/a.png) tail", "图", "dir/a.png", "![图](dir/a.png)"},
		{"heading", "# ![标题图](a.png)", "标题图", "a.png", "![标题图](a.png)"},
		{"quote", "> ![引用图](q.png)\r\n", "引用图", "q.png", "![引用图](q.png)"},
		{"list", "- ![列表图](l.png)\n", "列表图", "l.png", "![列表图](l.png)"},
		{"nested-label", "![a[b]c](n.png)", "a[b]c", "n.png", "![a[b]c](n.png)"},
		{"escaped-label", "![a\\]b](e.png)", "a\\]b", "e.png", "![a\\]b](e.png)"},
		{"windows-path", "![win](C:\\图\\a.png)", "win", `C:\图\a.png`, `![win](C:\图\a.png)`},
		{"literal-percent-and-url", "![url](https://example.invalid/a%20b.png)", "url", "https://example.invalid/a%20b.png", "![url](https://example.invalid/a%20b.png)"},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			image := requireOneInlineImage(t, fixture.source, fixture.label, fixture.href)
			got := fixture.source[image.Range.Start:image.Range.End]
			if got != fixture.syntax {
				t.Fatalf("range text=%q want=%q range=%#v", got, fixture.syntax, image.Range)
			}
		})
	}

	const multiLine = "before\n![line](a.png)\nafter"
	image := requireOneInlineImage(t, multiLine, "line", "a.png")
	if multiLine[image.Range.Start:image.Range.End] != "![line](a.png)" {
		t.Fatalf("multiline paragraph offset=%#v", image.Range)
	}
}

func TestInlineImageFallbackIsLiteralAndWarned(t *testing.T) {
	fixtures := []struct {
		name, source string
	}{
		{"empty", "![x]()"},
		{"space", "![x](a b.png)"},
		{"tab", "![x](a\tb.png)"},
		{"control", "![x](a\x00b.png)"},
		{"unicode-control", "![x](a\u0085b.png)"},
		{"invalid-utf8", "![x](a" + string([]byte{0xff}) + "b.png)"},
		{"angle", "![x](<a.png>)"},
		{"title", `![x](a.png "title")`},
		{"nested-unescaped", "![x](a(b).png)"},
		{"escaped-open", `![x](a\(b.png)`},
		{"escaped-close", `![x](a\)b.png)`},
		{"reference-style", "![x][id]"},
		{"malformed-label", "![x(a.png)"},
		{"malformed-destination", "![x](a.png"},
		{"multiline-label", "![x\ny](a.png)"},
		{"multiline-destination", "![x](a\nb.png)"},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			doc := Parse(fixture.source)
			if doc.Raw() != fixture.source || len(inlineSpansOfKind(doc, InlineImage)) != 0 {
				t.Fatalf("fallback produced image: spans=%#v warnings=%#v",
					inlineSpansOfKind(doc, InlineImage), doc.Warnings)
			}
			if !warningCodePresent(doc.Warnings, "unsupported-image") {
				t.Fatalf("missing unsupported-image warning: %#v", doc.Warnings)
			}
			for _, warning := range doc.Warnings {
				if warning.Code == "image-reference" {
					t.Fatalf("fallback produced reference warning: %#v", warning)
				}
			}
		})
	}
}

func TestInlineImageRejectedEnvelopeOwnsNestedFragments(t *testing.T) {
	fixtures := []struct {
		name, source string
		links        int
	}{
		{"nested-destination", "![bad](folder(![inner](x.png)))", 0},
		{"unmatched-destination", "![bad](folder(![inner](x.png))", 0},
		{"unmatched-label", "![outer ![inner](x.png)", 0},
		{"reference-label", "![outer][id ![inner](x.png)]", 0},
		{"complete-then-link", "![bad](folder(![inner](x.png))) [ordinary](ok.md)", 1},
		{"unmatched-then-next-line-link", "![outer ![inner](x.png)\n[ordinary](ok.md)", 1},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			doc := Parse(fixture.source)
			images := inlineSpansOfKind(doc, InlineImage)
			links := inlineSpansOfKind(doc, InlineLink)
			if doc.Raw() != fixture.source || len(images) != 0 || len(links) != fixture.links ||
				!warningCodePresent(doc.Warnings, "unsupported-image") {
				t.Fatalf("images=%#v links=%#v warnings=%#v", images, links, doc.Warnings)
			}
			for _, link := range links {
				if link.Href != "ok.md" {
					t.Fatalf("unrelated link changed: %#v", link)
				}
			}
		})
	}

	longOuter := "![outer" + strings.Repeat("a", maxInlineScanSuffix+1) + "![inner](x.png)"
	doc := Parse(longOuter)
	if doc.Raw() != longOuter || !doc.Partial || len(inlineSpansOfKind(doc, InlineImage)) != 0 ||
		!warningCodePresent(doc.Warnings, "scan-budget") {
		t.Fatalf("long rejected envelope: partial=%v warnings=%#v", doc.Partial, doc.Warnings)
	}
}

func TestInlineImageRejectedEnvelopeConsumesMultilineDestination(t *testing.T) {
	for _, fixture := range []struct {
		name, newline string
	}{
		{"lf", "\n"},
		{"crlf", "\r\n"},
		{"cr", "\r"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			envelope := "![bad](a" + fixture.newline + "![inner](x.png))"
			const suffix = " [ordinary](ok.md)"
			source := envelope + suffix

			doc := Parse(source)
			images := inlineSpansOfKind(doc, InlineImage)
			links := inlineSpansOfKind(doc, InlineLink)
			if doc.Raw() != source || doc.Partial || len(images) != 0 ||
				len(links) != 1 || links[0].Href != "ok.md" ||
				links[0].Range.Start != len(envelope)+1 {
				t.Fatalf("images=%#v links=%#v partial=%v warnings=%#v",
					images, links, doc.Partial, doc.Warnings)
			}
			if len(doc.Warnings) != 1 ||
				doc.Warnings[0].Code != "unsupported-image" ||
				doc.Warnings[0].Range != rangeAt(0, len(envelope)) {
				t.Fatalf("warnings=%#v envelopeEnd=%d", doc.Warnings, len(envelope))
			}
		})
	}
}

func TestInlineImageDestinationBoundaryAndWarningRange(t *testing.T) {
	for _, fixture := range []struct {
		size int
		want bool
	}{
		{1, true},
		{maxInlineImageDestinationBytes, true},
		{maxInlineImageDestinationBytes + 1, false},
	} {
		t.Run(fmt.Sprint(fixture.size), func(t *testing.T) {
			destination := strings.Repeat("x", fixture.size)
			source := "![bounded](" + destination + ")"
			doc := Parse(source)
			images := inlineSpansOfKind(doc, InlineImage)
			if doc.Raw() != source || (len(images) == 1) != fixture.want {
				t.Fatalf("size=%d images=%#v warnings=%#v", fixture.size, images, doc.Warnings)
			}
			if fixture.want {
				if images[0].Href != destination {
					t.Fatalf("destination changed: got=%d want=%d", len(images[0].Href), len(destination))
				}
				return
			}
			if doc.Partial || len(doc.Warnings) != 1 ||
				doc.Warnings[0].Code != "unsupported-image" ||
				doc.Warnings[0].Range != rangeAt(0, len(source)) {
				t.Fatalf("oversized fallback warnings=%#v partial=%v", doc.Warnings, doc.Partial)
			}
		})
	}
}

func TestInlineImagePreservesOwnershipAndOrdinaryLinks(t *testing.T) {
	fixtures := []struct {
		name, source string
		kind         InlineKind
	}{
		{"outer-link", "[![image](a.png)](target.md)", InlineLink},
		{"escaped-exclamation", `\![ordinary](target.md)`, InlineLink},
		{"bold", "**![image](a.png)**", InlineBold},
		{"italic", "*![image](a.png)*", InlineItalic},
		{"code-span", "`![image](a.png)`", InlineCode},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			doc := Parse(fixture.source)
			if doc.Raw() != fixture.source || len(inlineSpansOfKind(doc, InlineImage)) != 0 ||
				len(inlineSpansOfKind(doc, fixture.kind)) != 1 {
				t.Fatalf("ownership changed: blocks=%#v warnings=%#v", doc.Blocks, doc.Warnings)
			}
		})
	}

	outer := Parse(fixtures[0].source)
	link := inlineSpansOfKind(outer, InlineLink)[0]
	if link.Text != "![image](a.png)" || link.Href != "target.md" {
		t.Fatalf("ordinary outer link changed: %#v", link)
	}

	escaped := Parse(fixtures[1].source)
	link = inlineSpansOfKind(escaped, InlineLink)[0]
	if link.Text != "ordinary" || link.Href != "target.md" {
		t.Fatalf("escaped exclamation link changed: %#v", link)
	}

	code := Parse("```\n![image](a.png)\n```\n")
	if len(inlineSpansOfKind(code, InlineImage)) != 0 ||
		len(code.Blocks) != 1 || code.Blocks[0].Kind != BlockCode {
		t.Fatalf("code fence changed: %#v", code.Blocks)
	}
	table := Parse("| image |\n| --- |\n| ![x](a.png) |\n")
	if len(inlineSpansOfKind(table, InlineImage)) != 0 ||
		len(table.Blocks) != 1 || table.Blocks[0].Kind != BlockTable ||
		table.Blocks[0].Raw != "| image |\n| --- |\n| ![x](a.png) |\n" {
		t.Fatalf("table cell gained image action: %#v", table.Blocks)
	}
}

func TestInlineImageHTMLIsExactLiteral(t *testing.T) {
	const source = "![<&>](https://example.invalid/a%20b.png \"ignored\")\n"
	doc := Parse(source)
	if len(inlineSpansOfKind(doc, InlineImage)) != 0 {
		t.Fatalf("title form became actionable: %#v", doc.Warnings)
	}
	body := "<p>![&lt;&amp;&gt;](https://example.invalid/a%20b.png &#34;ignored&#34;)</p>\n"
	requireExactExport(t, doc, body, doc.Warnings)

	const accepted = "![<&>](https://example.invalid/a%20b.png)\n"
	doc = Parse(accepted)
	images := inlineSpansOfKind(doc, InlineImage)
	if len(images) != 1 || len(doc.Warnings) != 1 ||
		doc.Warnings[0].Code != "image-reference" {
		t.Fatalf("accepted fixture=%#v warnings=%#v", images, doc.Warnings)
	}
	body = "<p>![&lt;&amp;&gt;](https://example.invalid/a%20b.png)</p>\n"
	requireExactExport(t, doc, body, doc.Warnings)
	html, _, err := ExportHTMLChecked(doc)
	if err != nil || strings.Contains(html, "<img") || strings.Contains(html, " src=") {
		t.Fatalf("image syntax became HTML resource: err=%v html=%q", err, html)
	}
}

func TestInlineImageBudgetsAndForwardProgress(t *testing.T) {
	const image = "![x](a.png) "
	source := strings.Repeat(image, maxInlineSpansPerBlock+1)
	doc := Parse(source)
	images := inlineSpansOfKind(doc, InlineImage)
	if doc.Raw() != source || !doc.Partial || len(images) != maxInlineSpansPerBlock ||
		images[len(images)-1].Range.End > len(source) {
		t.Fatalf("span boundary: partial=%v images=%d warnings=%d", doc.Partial, len(images), len(doc.Warnings))
	}

	tooLongLabel := "![" + strings.Repeat("a", maxInlineScanSuffix+1) + "](a.png)"
	doc = Parse(tooLongLabel)
	if doc.Raw() != tooLongLabel || len(inlineSpansOfKind(doc, InlineImage)) != 0 ||
		!doc.Partial || !warningCodePresent(doc.Warnings, "scan-budget") {
		t.Fatalf("work boundary: partial=%v warnings=%#v", doc.Partial, doc.Warnings)
	}

	limit := &inlineLimit{spans: maxInlineSpans}
	spans, warnings := parseInline("![x](a.png)", 0, limit)
	if len(spans) != 0 || !limit.exhausted ||
		!warningCodePresent(warnings, "inline-limit") {
		t.Fatalf("document span boundary: spans=%#v limit=%#v warnings=%#v", spans, limit, warnings)
	}
}
