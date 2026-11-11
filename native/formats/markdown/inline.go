package markdown

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type inlineParser struct {
	source    string
	offset    int
	spans     []InlineSpan
	warnings  []Warning
	scanSpent int
	exhausted bool
	limit     *inlineLimit
}

const maxInlineScanBytes = 1 << 20
const maxInlineScanSuffix = 64 << 10
const maxInlineDelimiterRun = 64
const maxInlineSpansPerBlock = 4096
const maxInlineSpans = 16384
const maxInlineImageDestinationBytes = 4096

const inlineImageReferenceMessage = "image syntax is preserved as an explicit picture reference; inline images are not rendered"

type inlineLimit struct {
	scanSpent int
	spans     int
	exhausted bool
	warning   string
	message   string
}

func parseInline(source string, offset int, limit *inlineLimit) ([]InlineSpan, []Warning) {
	p := inlineParser{
		source: source, offset: offset, spans: []InlineSpan{}, warnings: []Warning{},
		scanSpent: limit.scanSpent, exhausted: limit.exhausted, limit: limit,
	}
	p.warnings = p.parse()
	limit.spans += len(p.spans)
	return p.spans, p.warnings
}

func (p *inlineParser) parse() []Warning {
	i := 0
	for i < len(p.source) {
		if p.limit.exhausted {
			p.exhausted = true
			break
		}
		if p.limit.spans+len(p.spans) >= maxInlineSpans {
			p.markExhausted("inline-limit", "document inline syntax element limit reached; source remains visible")
			break
		}
		if len(p.spans) >= maxInlineSpansPerBlock {
			p.markExhausted("inline-limit", "block inline syntax element limit reached; source remains visible")
			break
		}
		_, width := utf8.DecodeRuneInString(p.source[i:])
		if width == 0 {
			width = 1
		}
		p.detectUnsupportedAt(i)
		switch p.source[i] {
		case '\\':
			i += 2
			if i > len(p.source) {
				i = len(p.source)
			}
		case '*':
			if i+1 < len(p.source) && p.source[i+1] == '*' {
				if end := p.findMarker(i+2, "**"); end >= 0 {
					p.spans = append(p.spans, InlineSpan{
						Kind: InlineBold, Range: rangeAt(p.offset+i, p.offset+end+2),
						Text: p.source[i+2 : end],
					})
					i = end + 2
				} else {
					p.warn("unmatched-bold", i, i+2)
					i++
				}
			} else if end := p.findMarker(i+1, "*"); end >= 0 {
				p.spans = append(p.spans, InlineSpan{
					Kind: InlineItalic, Range: rangeAt(p.offset+i, p.offset+end+1),
					Text: p.source[i+1 : end],
				})
				i = end + 1
			} else {
				p.warn("unmatched-italic", i, i+1)
				i++
			}
		case '_':
			if i+1 < len(p.source) && p.source[i+1] == '_' {
				if end := p.findMarker(i+2, "__"); end >= 0 {
					p.spans = append(p.spans, InlineSpan{
						Kind: InlineBold, Range: rangeAt(p.offset+i, p.offset+end+2),
						Text: p.source[i+2 : end],
					})
					i = end + 2
				} else {
					p.warn("unmatched-bold", i, i+2)
					i++
				}
			} else if end := p.findMarker(i+1, "_"); end >= 0 {
				p.spans = append(p.spans, InlineSpan{
					Kind: InlineItalic, Range: rangeAt(p.offset+i, p.offset+end+1),
					Text: p.source[i+1 : end],
				})
				i = end + 1
			} else {
				p.warn("unmatched-italic", i, i+1)
				i++
			}
		case '`':
			marker := 1
			for i+marker < len(p.source) && p.source[i+marker] == '`' {
				if marker >= maxInlineDelimiterRun {
					break
				}
				marker++
			}
			if !p.spendScan(min(marker, len(p.source)-i)) {
				break
			}
			closing := p.findMarker(i+marker, strings.Repeat("`", marker))
			if closing >= 0 {
				text := p.source[i+marker : closing]
				trimmed := strings.Trim(text, " ")
				if len(text) > 1 && text[0] == ' ' && text[len(text)-1] == ' ' {
					text = trimmed
				}
				p.spans = append(p.spans, InlineSpan{
					Kind: InlineCode, Range: rangeAt(p.offset+i, p.offset+closing+marker),
					Text: text,
				})
				i = closing + marker
			} else {
				p.warn("unmatched-code", i, i+marker)
				i += marker
			}
		case '[':
			if i+1 < len(p.source) && p.source[i+1] == '^' {
				if closing := p.indexByte(i, ']'); closing > i {
					p.warn("footnote", i, closing+1)
					i = closing + 1
					continue
				}
			}
			label, closing, ok := p.readBracketLabel(i)
			if !ok {
				if p.limit.exhausted {
					break
				}
				p.warn("unmatched-link", i, i+1)
				i++
				continue
			}
			rest := closing + 1
			if rest < len(p.source) && p.source[rest] == '(' {
				urlStart := rest + 1
				urlEnd := p.findLinkEnd(urlStart)
				if urlEnd >= 0 && urlEnd < len(p.source) && p.source[urlEnd] == ')' {
					p.spans = append(p.spans, InlineSpan{
						Kind: InlineLink, Range: rangeAt(p.offset+i, p.offset+urlEnd+1),
						Text: label, Href: p.source[urlStart:urlEnd],
					})
					i = urlEnd + 1
					continue
				}
				p.warn("unmatched-link-url", i, rest)
				i = rest
				continue
			}
			p.warn("unsupported-reference-link", i, closing+1)
			i = closing + 1
		case '!':
			if strings.HasPrefix(p.source[i:], "![") {
				label, closing, ok := p.readBracketLabel(i + 1)
				if !ok {
					if p.limit.exhausted {
						break
					}
					end := p.rejectedImageLineEnd(i)
					p.warn("unsupported-image", i, end)
					i = end
					continue
				}
				rest := closing + 1
				if rest >= len(p.source) || p.source[rest] != '(' {
					end := closing + 1
					if rest < len(p.source) && p.source[rest] == '[' {
						if _, referenceEnd, found := p.readBracketLabel(rest); found {
							end = referenceEnd + 1
						} else {
							if p.limit.exhausted {
								break
							}
							end = p.rejectedImageLineEnd(i)
						}
					}
					p.warn("unsupported-image", i, end)
					i = end
					continue
				}
				urlStart := rest + 1
				urlEnd := p.findLinkEnd(urlStart)
				if urlEnd < 0 || urlEnd >= len(p.source) || p.source[urlEnd] != ')' {
					end := p.rejectedImageEnvelopeEnd(rest)
					p.warn("unsupported-image", i, end)
					i = end
					continue
				}
				href := p.source[urlStart:urlEnd]
				end := urlEnd + 1
				if validInlineImage(label, href) {
					p.spans = append(p.spans, InlineSpan{
						Kind: InlineImage, Range: rangeAt(p.offset+i, p.offset+end),
						Text: label, Href: href,
					})
					p.warnings = appendSyntaxWarning(p.warnings, "image-reference",
						inlineImageReferenceMessage, rangeAt(p.offset+i, p.offset+end))
				} else {
					p.warn("unsupported-image", i, end)
				}
				i = end
				continue
			}
			i++
		case '$':
			if strings.HasPrefix(p.source[i:], "$$") {
				if closing := p.findMarker(i+2, "$$"); closing >= 0 {
					p.warn("unsupported-math", i, closing+2)
					i = closing + 2
					continue
				}
			}
			if closing := p.indexByte(i+1, '$'); closing > i {
				p.warn("unsupported-math", i, closing+1)
				i = closing + 1
				continue
			}
			p.warn("unmatched-math", i, i+1)
			i++
		default:
			i += width
		}
	}
	return p.warnings
}

func (p *inlineParser) detectUnsupportedAt(index int) {
	if !isWordBoundary(p.source, index) {
		return
	}
	for _, phrase := range unsupportedPhrases {
		end := index + len(phrase)
		if end <= len(p.source) && strings.EqualFold(p.source[index:end], phrase) {
			p.warnings = appendSyntaxWarning(p.warnings, "unsupported-syntax", "syntax is not rendered; source text remains visible", rangeAt(p.offset+index, p.offset+end))
			break
		}
	}
}

func (p *inlineParser) spendScan(length int) bool {
	if p.exhausted {
		return false
	}
	if p.limit.exhausted {
		p.exhausted = true
		return false
	}
	if !p.spendDocumentScan(length) {
		return false
	}
	return true
}

func (p *inlineParser) spendDocumentScan(length int) bool {
	p.scanSpent += length
	p.limit.scanSpent = p.scanSpent
	if p.limit.scanSpent > maxInlineScanBytes {
		p.markExhausted("scan-budget",
			"inline syntax scan exceeded the bounded work limit; source remains visible")
		return false
	}
	return true
}

func (p *inlineParser) markExhausted(code, message string) {
	if !p.limit.exhausted {
		p.limit.scanSpent = p.scanSpent
		p.limit.warning = code
		p.limit.message = message
		p.warnings = appendSyntaxWarning(p.warnings, code, message,
			rangeAt(p.offset, p.offset+len(p.source)))
	}
	p.limit.exhausted = true
	p.exhausted = true
}

func (p *inlineParser) indexByte(start int, value byte) int {
	suffix := p.suffix(start)
	if suffix == "" {
		return -1
	}
	relative := strings.IndexByte(suffix, value)
	if relative < 0 {
		p.spendScan(len(suffix))
		return -1
	}
	if !p.spendScan(relative + 1) {
		return -1
	}
	return start + relative
}

func (p *inlineParser) findLinkEnd(start int) int {
	suffix := p.suffix(start)
	if suffix == "" {
		return -1
	}
	relative := findLinkEnd(suffix, 0)
	if relative < 0 {
		p.spendScan(len(suffix))
		return -1
	}
	if !p.spendScan(relative + 1) {
		return -1
	}
	return start + relative
}

func (p *inlineParser) findMarker(start int, marker string) int {
	suffix := p.suffix(start)
	if suffix == "" {
		return -1
	}
	relative := strings.Index(suffix, marker)
	if relative < 0 {
		p.spendScan(len(suffix))
		return -1
	}
	if !p.spendScan(relative + len(marker)) {
		return -1
	}
	return start + relative
}

func (p *inlineParser) readBracketLabel(open int) (string, int, bool) {
	suffix := p.suffix(open)
	if suffix == "" {
		if !p.limit.exhausted && len(p.source)-open > maxInlineScanSuffix {
			p.markExhausted("scan-budget",
				"inline syntax scan exceeded the bounded work limit; source remains visible")
		}
		return "", -1, false
	}
	label, relative, ok := readBracketLabel(suffix, 0)
	if !ok {
		if !p.spendScan(len(suffix)) && p.limit.exhausted {
			return "", -1, false
		}
		if len(p.source)-open > maxInlineScanSuffix {
			p.markExhausted("scan-budget",
				"inline syntax scan exceeded the bounded work limit; source remains visible")
		}
		return "", -1, false
	}
	if !p.spendScan(relative + 1) {
		return "", -1, false
	}
	return label, open + relative, true
}

func (p *inlineParser) suffix(start int) string {
	if p.exhausted || p.limit.exhausted || start < 0 || start >= len(p.source) {
		return ""
	}
	remaining := maxInlineScanBytes - p.scanSpent
	if remaining <= 0 {
		p.markExhausted("scan-budget",
			"inline syntax scan exceeded the bounded work limit; source remains visible")
		return ""
	}
	length := len(p.source) - start
	length = min(length, maxInlineScanSuffix)
	length = min(length, remaining)
	return p.source[start : start+length]
}

var unsupportedPhrases = []string{
	"```mermaid", "```math", "task:", "foo:",
}

func isWordBoundary(source string, index int) bool {
	if index == 0 {
		return true
	}
	previous, _ := utf8.DecodeLastRuneInString(source[:index])
	return !unicode.IsLetter(previous) && !unicode.IsDigit(previous)
}

func (p *inlineParser) warn(code string, start, end int) {
	p.warnings = appendSyntaxWarning(p.warnings, code, "inline syntax is not fully supported; source text remains visible", rangeAt(p.offset+start, p.offset+end))
}

func appendSyntaxWarning(warnings []Warning, code, message string, rng SourceRange) []Warning {
	if len(warnings) >= warningLimit {
		return warnings
	}
	return append(warnings, Warning{Code: code, Message: message, Range: rng})
}

func readBracketLabel(source string, open int) (string, int, bool) {
	depth := 0
	for i := open; i < len(source); i++ {
		if source[i] == '\\' {
			i++
			continue
		}
		if source[i] == '[' {
			depth++
			continue
		}
		if source[i] == ']' {
			depth--
			if depth == 0 {
				return source[open+1 : i], i, true
			}
		}
	}
	return "", -1, false
}

func findLinkEnd(source string, start int) int {
	for i := start; i < len(source); i++ {
		if source[i] == '\\' {
			i++
			continue
		}
		if source[i] == '(' || source[i] == ')' {
			return i
		}
	}
	return -1
}

func validInlineImage(label, destination string) bool {
	if strings.ContainsAny(label, "\r\n") ||
		len(destination) == 0 || len(destination) > maxInlineImageDestinationBytes ||
		!utf8.ValidString(destination) {
		return false
	}
	for _, value := range destination {
		if unicode.IsControl(value) || value == ' ' ||
			value == '<' || value == '>' || value == '(' || value == ')' {
			return false
		}
	}
	for i := 0; i < len(destination); i++ {
		value := destination[i]
		if value == '\\' && i+1 < len(destination) &&
			(destination[i+1] == '(' || destination[i+1] == ')') {
			// The format layer intentionally has no Markdown unescape pass.
			// Passing these bytes through would change their filesystem meaning.
			return false
		}
	}
	return true
}

// rejectedImageLineEnd bounds an unprovable rejected image at its physical
// line. The outer construct owns the remaining label-looking fragments on that
// line, but parser recovery must resume for later lines and complete envelopes.
func (p *inlineParser) rejectedImageLineEnd(start int) int {
	suffix := p.suffix(start)
	if suffix == "" {
		return start
	}
	if relative := strings.IndexAny(suffix, "\r\n"); relative >= 0 {
		if !p.spendScan(relative + 1) {
			return start
		}
		return start + relative
	}
	if !p.spendScan(len(suffix)) {
		return start
	}
	if len(p.source)-start > len(suffix) {
		p.markExhausted("scan-budget",
			"inline syntax scan exceeded the bounded work limit; source remains visible")
	}
	return min(start+len(suffix), len(p.source))
}

// rejectedImageEnvelopeEnd finds the matching end of a rejected parenthesized
// image candidate for skip purposes only. A rejected envelope may cross line
// endings when its matching close is provable; accepted destinations remain
// single-line in validInlineImage. Escape handling mirrors the existing inline
// scanner because the raw backslash bytes remain part of the source.
func (p *inlineParser) rejectedImageEnvelopeEnd(open int) int {
	suffix := p.suffix(open)
	if suffix == "" {
		return open
	}
	depth := 0
	for i := 0; i < len(suffix); i++ {
		switch suffix[i] {
		case '\\':
			i++
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
			if depth == 0 {
				if !p.spendScan(i + 1) {
					return open
				}
				return open + i + 1
			}
		}
	}
	if !p.spendScan(len(suffix)) {
		return open
	}
	if len(p.source)-open > len(suffix) {
		p.markExhausted("scan-budget",
			"inline syntax scan exceeded the bounded work limit; source remains visible")
	}
	return min(open+len(suffix), len(p.source))
}
