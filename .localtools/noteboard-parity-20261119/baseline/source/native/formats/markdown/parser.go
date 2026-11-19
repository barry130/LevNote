package markdown

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	maxInputBytes = 8 << 20 // 8 MiB parse guard
	maxLines      = 65536
	maxBlocks     = 8192
	maxListItems  = 4096
	warningLimit  = 256
	maxLineSlice  = 128
)

type line struct {
	content string
	start   int
	end     int // includes the line ending
}

func splitLines(source string) ([]line, bool) {
	complete := true
	start := 0
	var lines []line
	allocLineSlice := func() {
		if lines == nil {
			lines = make([]line, 0, maxLineSlice)
		}
	}
	for i := 0; i < len(source); i++ {
		if source[i] != '\n' && source[i] != '\r' {
			continue
		}
		end := i + 1
		if source[i] == '\r' && i+1 < len(source) && source[i+1] == '\n' {
			i++
			end++
		}
		contentEnd := end
		if source[end-1] == '\n' {
			contentEnd--
		}
		if contentEnd > start && source[contentEnd-1] == '\r' {
			contentEnd--
		}
		allocLineSlice()
		lines = append(lines, line{source[start:contentEnd], start, end})
		if len(lines) > maxLines {
			complete = false
			break
		}
		start = end
	}
	if start < len(source) {
		allocLineSlice()
		lines = append(lines, line{source[start:], start, len(source)})
		if len(lines) > maxLines {
			lines = lines[:maxLines]
			complete = false
		}
	}
	return lines, complete
}

var (
	headingRe   = regexp.MustCompile(`^(#{1,6})([ \t]+)(.*)$`)
	orderedRe   = regexp.MustCompile(`^(\d{1,9})[.)][ \t]+(.*)$`)
	bulletRe    = regexp.MustCompile(`^[-*+][ \t]+(.*)$`)
	openFenceRe = regexp.MustCompile("^(`{3,})[ \\t]*([^`\\s]*)[^\\n]*$")
	fenceOnlyRe = regexp.MustCompile("^( {0,3})(`{3,})[ \\t]*$")
)

func isTableRow(content string) bool {
	start := 0
	for start < len(content) && (content[start] == ' ' || content[start] == '\t') {
		start++
	}
	return start < len(content) && content[start] == '|'
}

func isBlank(s string) bool { return strings.Trim(s, " \t\r\n") == "" }

func isThematic(s string) bool {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	if end-start < 3 {
		return false
	}
	if s[start] != '-' && s[start] != '*' {
		return false
	}
	first := s[start]
	count := 0
	for i := start; i < end; i++ {
		if s[i] == first {
			count++
		} else if s[i] != ' ' && s[i] != '\t' {
			return false
		}
	}
	return count >= 3
}

func rangeAt(start, end int) SourceRange { return SourceRange{Start: start, End: end} }
func lineRange(line line) SourceRange    { return rangeAt(line.start, line.end) }
func lineContentEnd(line line) int       { return line.start + len(line.content) }

func addWarning(warnings []Warning, message string, rng SourceRange) []Warning {
	if len(warnings) >= warningLimit {
		return warnings
	}
	return append(warnings, Warning{Code: "parse", Message: message, Range: rng})
}

func startsList(s string) (bool, bool, string) {
	if match := orderedRe.FindStringSubmatch(s); match != nil {
		return true, true, match[2]
	}
	if match := bulletRe.FindStringSubmatch(s); match != nil {
		return true, false, match[1]
	}
	return false, false, ""
}

func isBlockStart(s string) bool {
	if isBlank(s) || headingRe.MatchString(s) {
		return true
	}
	list, _, _ := startsList(s)
	return list || openFenceRe.MatchString(s) || strings.HasPrefix(s, ">")
}

func Parse(source string) Document {
	if len(source) > maxInputBytes {
		return Document{source: source, Partial: true, Warnings: []Warning{{
			Code:    "too-large",
			Message: "Markdown over 8 MiB is shown as raw text",
			Range:   rangeAt(0, len(source)),
		}}}
	}

	doc := Document{source: source}
	lines, complete := splitLines(source)
	if !complete {
		doc.Partial = true
		doc.Warnings = append(doc.Warnings, Warning{
			Code: "parse-limit", Message: "Markdown line limit reached; remaining source is preserved as raw text",
			Range: rangeAt(limitStart(lines), len(source)),
		})
	}
	inlineBudget := &inlineLimit{}
	tableCellsUsed := 0
	syncInlineExhaustion := func(warnings []Warning) []Warning {
		if inlineBudget.exhausted && inlineBudget.warning != "" {
			warnings = appendSyntaxWarning(warnings, inlineBudget.warning, inlineBudget.message,
				rangeAt(0, len(source)))
			inlineBudget.warning = ""
			inlineBudget.message = ""
			inlineBudget.exhausted = true
		}
		return warnings
	}
	for i := 0; i < len(lines); {
		line := lines[i]
		if len(doc.Blocks) >= maxBlocks {
			doc.Partial = true
			doc.Warnings = appendSyntaxWarning(doc.Warnings, "parse-limit",
				"Markdown block limit reached; remaining source is preserved as raw text",
				rangeAt(line.start, len(source)))
			break
		}
		if isBlank(line.content) {
			if inlineBudget.exhausted {
				doc.Partial = true
				doc.Warnings = syncInlineExhaustion(doc.Warnings)
				break
			}
			i++
			continue
		}

		if match := headingRe.FindStringSubmatch(line.content); match != nil {
			textStart := line.start + len(match[1]) + len(match[2])
			textEnd := line.start + len(strings.TrimRight(line.content, " \t"))
			if textEnd < textStart {
				textEnd = textStart
			}
			text := source[textStart:textEnd]
			inline, inlineWarnings := parseInline(text, textStart, inlineBudget)
			doc.Warnings = appendWarningsCapped(doc.Warnings, inlineWarnings)
			doc.Blocks = append(doc.Blocks, Block{
				Kind: BlockHeading, Range: lineRange(line), TextRange: rangeAt(textStart, textEnd),
				Text: text, Inline: inline, Level: len(match[1]), Raw: source[line.start:line.end],
			})
			i++
			continue
		}

		if block, next, warnings, cells, handled := parseTable(source, lines, i, tableCellsUsed); handled {
			doc.Warnings = appendWarningsCapped(doc.Warnings, warnings)
			doc.Blocks = append(doc.Blocks, block)
			tableCellsUsed = cells
			i = next
			continue
		}

		if isThematic(line.content) {
			doc.Blocks = append(doc.Blocks, Block{
				Kind: BlockDivider, Range: lineRange(line), Raw: source[line.start:line.end],
			})
			i++
			continue
		}

		if isTableRow(line.content) {
			end := i
			for end < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[end].content), "|") {
				end++
			}
			rng := rangeAt(lines[i].start, lines[end-1].end)
			doc.Warnings = appendSyntaxWarning(doc.Warnings, "unsupported-table", "table syntax left visible; rendering tables is unsupported", rng)
			doc.Blocks = append(doc.Blocks, Block{
				Kind: BlockParagraph, Range: rng, TextRange: rng,
				Text: source[rng.Start:rng.End], Inline: nil, Raw: source[rng.Start:rng.End],
			})
			i = end
			continue
		}

		if docBlock, next, blockWarnings, handled := parseDocBlock(source, lines, i); handled {
			doc.Warnings = appendWarningsCapped(doc.Warnings, blockWarnings)
			doc.Blocks = append(doc.Blocks, docBlock)
			i = next
			continue
		}

		if match := openFenceRe.FindStringSubmatch(line.content); match != nil {
			language := strings.TrimSpace(match[2])
			if language == "mermaid" || language == "math" {
				doc.Warnings = appendSyntaxWarning(doc.Warnings, "unsupported-syntax", "diagram/math fence is left visible and is not rendered", lineRange(line))
			}
			block, next, blockWarnings := parseFence(source, lines, i, match[1], language)
			doc.Warnings = appendWarningsCapped(doc.Warnings, blockWarnings)
			if block != nil {
				doc.Blocks = append(doc.Blocks, *block)
			}
			i = next
			continue
		}

		list, ordered, _ := startsList(line.content)
		if list {
			block, next, blockWarnings, complete := parseList(source, lines, i, ordered, inlineBudget)
			doc.Warnings = appendWarningsCapped(doc.Warnings, blockWarnings)
			doc.Blocks = append(doc.Blocks, block)
			if !complete {
				doc.Partial = true
				break
			}
			i = next
			continue
		}

		if strings.HasPrefix(line.content, ">") {
			end := i
			for end < len(lines) && strings.HasPrefix(lines[end].content, ">") {
				end++
			}
			quoteText := source[lines[i].start:lines[end-1].end]
			quoteInline, quoteWarnings := parseInline(quoteText, lines[i].start, inlineBudget)
			doc.Warnings = appendWarningsCapped(doc.Warnings, quoteWarnings)
			doc.Blocks = append(doc.Blocks, Block{
				Kind: BlockQuote, Range: rangeAt(lines[i].start, lines[end-1].end),
				TextRange: rangeAt(lines[i].start, lines[end-1].end), Text: quoteText,
				Inline: quoteInline, Raw: quoteText,
			})
			i = end
			continue
		}

		end := i
		if len(doc.Blocks) >= maxBlocks {
			doc.Partial = true
			doc.Warnings = appendSyntaxWarning(doc.Warnings, "parse-limit",
				"Markdown block limit reached; remaining source is preserved as raw text",
				rangeAt(line.start, len(source)))
			break
		}
		for end < len(lines) && !isBlank(lines[end].content) && !isBlockStart(lines[end].content) {
			if end > i && end+1 < len(lines) &&
				tableCandidate(lines[end].content) &&
				delimiterShapedContent(lines[end+1].content) {
				break
			}
			end++
		}
		if end == i {
			end++
		}
		textStart := lines[i].start
		textEnd := lineContentEnd(lines[end-1])
		text := source[textStart:textEnd]
		inline, paragraphWarnings := parseInline(text, textStart, inlineBudget)
		doc.Warnings = appendWarningsCapped(doc.Warnings, paragraphWarnings)
		doc.Blocks = append(doc.Blocks, Block{
			Kind: BlockParagraph, Range: rangeAt(textStart, textEnd), TextRange: rangeAt(textStart, textEnd),
			Text: text, Inline: inline, Raw: text,
		})
		i = end
	}
	if inlineBudget.exhausted {
		doc.Partial = true
	}

	return doc
}

func limitStart(lines []line) int {
	if len(lines) == 0 {
		return 0
	}
	return lines[len(lines)-1].end
}

func parseFence(source string, lines []line, index int, ticks, language string) (*Block, int, []Warning) {
	fenceLength := len(ticks)
	bodyStart := lines[index].end
	end := -1
	for i := index + 1; i < len(lines); i++ {
		trimmed := strings.TrimLeft(lines[i].content, " \t")
		if isFenceClose(trimmed, fenceLength) {
			end = i
			break
		}
	}

	bodyEnd := lines[len(lines)-1].end
	rawEnd := len(source)
	if end >= 0 {
		bodyEnd = lines[end].start
		rawEnd = lines[end].end
	}
	body := source[bodyStart:bodyEnd]
	block := Block{
		Kind: BlockCode, Range: rangeAt(lines[index].start, rawEnd),
		TextRange: rangeAt(bodyStart, bodyEnd), Text: body, Language: language,
		Raw: source[lines[index].start:rawEnd],
	}

	var warnings []Warning
	next := len(lines)
	if end >= 0 {
		next = end + 1
	} else {
		warnings = addWarning(warnings, "unterminated code fence left visible", rangeAt(lines[index].start, rawEnd))
	}
	return &block, next, warnings
}

func isFenceClose(s string, fence int) bool {
	count := 0
	for len(s) > 0 && s[0] == '`' {
		count++
		s = s[1:]
	}
	return count >= fence && strings.TrimLeft(s, " \t") == ""
}

// parseTaskMarker recognizes the approved task subset at an already recognized
// list item body start. bodyEnd is the parser's trimmed body end, so separator
// scanning never advances beyond it; a trailing-whitespace item therefore gets
// a valid empty label at markerEnd.
func parseTaskMarker(source string, bodyStart, bodyEnd int) (*TaskMarker, int, bool) {
	if bodyEnd-bodyStart < 3 || source[bodyStart] != '[' {
		return nil, bodyStart, false
	}
	middle := source[bodyStart+1]
	checked := middle == 'x' || middle == 'X'
	if middle != ' ' && !checked || source[bodyStart+2] != ']' {
		return nil, bodyStart, false
	}
	markerEnd := bodyStart + 3
	if markerEnd != bodyEnd && (source[markerEnd] != ' ' && source[markerEnd] != '\t') {
		return nil, bodyStart, false
	}
	labelStart := markerEnd
	for labelStart < bodyEnd && (source[labelStart] == ' ' || source[labelStart] == '\t') {
		labelStart++
	}
	return &TaskMarker{
		Range:   rangeAt(bodyStart, markerEnd),
		Checked: checked,
	}, labelStart, true
}

func parseList(source string, lines []line, index int, ordered bool, inlineBudget *inlineLimit) (Block, int, []Warning, bool) {
	items := []ListItem{}
	var warnings []Warning
	i := index
	start := 1
	if ordered {
		orderedMatch := orderedRe.FindStringSubmatchIndex(lines[index].content)
		if orderedMatch != nil && len(orderedMatch) >= 4 {
			if number, err := strconv.Atoi(lines[index].content[orderedMatch[2]:orderedMatch[3]]); err == nil {
				start = number
			}
		}
	}
	for i < len(lines) {
		current := lines[i]
		orderedMatch := orderedRe.FindStringSubmatchIndex(current.content)
		bulletMatch := bulletRe.FindStringSubmatchIndex(current.content)
		if i == index && ordered {
			bulletMatch = nil
		}
		var textStart int
		var kindOrdered bool
		if orderedMatch != nil {
			textStart = orderedMatch[4]
			kindOrdered = true
		} else if bulletMatch != nil {
			textStart = bulletMatch[2]
			kindOrdered = false
		}
		if orderedMatch == nil && bulletMatch == nil {
			break
		}
		if kindOrdered != ordered {
			break
		}
		if len(items) >= maxListItems {
			warnings = appendSyntaxWarning(warnings, "parse-limit",
				"Markdown list item limit reached; remaining list source is preserved as raw text",
				rangeAt(current.start, len(source)))
			return Block{Kind: BlockList, Ordered: ordered, Start: start, Items: items,
				Range: rangeAt(lines[index].start, current.start), Raw: source[lines[index].start:current.start],
			}, i, warnings, false
		}
		if textStart >= len(current.content) {
			textEnd := current.start + len(strings.TrimRight(current.content, " \t"))
			textStart = min(current.start+textStart, textEnd)
			task, labelStart, isTask := parseTaskMarker(source, textStart, textEnd)
			if isTask {
				textStart = labelStart
			}
			itemInline, inlineWarnings := parseInline("", textStart, inlineBudget)
			warnings = appendWarningsCapped(warnings, inlineWarnings)
			items = append(items, ListItem{
				Range: lineRange(current), TextRange: rangeAt(textStart, textEnd),
				Text: "", Inline: itemInline,
				Task: task,
			})
			i++
			break
		}
		textEnd := current.start + len(strings.TrimRight(current.content, " \t"))
		textStart = current.start + textStart
		textStart = min(textStart, textEnd)
		task, labelStart, isTask := parseTaskMarker(source, textStart, textEnd)
		if isTask {
			textStart = labelStart
		}
		visible := source[textStart:textEnd]
		itemInline, inlineWarnings := parseInline(visible, textStart, inlineBudget)
		warnings = appendWarningsCapped(warnings, inlineWarnings)
		items = append(items, ListItem{
			Range: lineRange(current), TextRange: rangeAt(textStart, textEnd),
			Text: visible, Inline: itemInline,
			Task: task,
		})
		i++
		if i < len(lines) && !isBlank(lines[i].content) &&
			(strings.HasPrefix(lines[i].content, " ") || strings.HasPrefix(lines[i].content, "\t")) {
			continue
		}
	}

	raw := source[lines[index].start:lines[i-1].end]
	return Block{
		Kind: BlockList, Range: rangeAt(lines[index].start, lines[i-1].end),
		Ordered: ordered, Start: start, Items: items, Raw: raw,
	}, i, warnings, true
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func appendWarningsCapped(target, source []Warning) []Warning {
	for _, warning := range source {
		target = appendSyntaxWarning(target, warning.Code, warning.Message, warning.Range)
	}
	return target
}
