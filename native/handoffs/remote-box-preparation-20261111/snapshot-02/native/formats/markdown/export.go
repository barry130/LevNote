package markdown

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
)

var ErrPartialDocument = errors.New("Markdown parse was bounded; export would be incomplete")

// ErrOutputLimit reports that the complete HTML would not fit the supplied
// byte budget. No partial output or export warnings are returned with it.
var ErrOutputLimit = errors.New("Markdown HTML output exceeds limit")

// htmlSink shares the emitter between a counting pass, exact-sized bytes and
// the legacy one-pass string API. Only the byte API has a caller-supplied cap.
type htmlSink struct {
	limit   int
	written int
	bytes   []byte
	text    *strings.Builder
	err     error
}

func (out *htmlSink) WriteString(value string) {
	if out.err != nil {
		return
	}
	if len(value) > out.limit-out.written {
		out.err = ErrOutputLimit
		return
	}
	if out.bytes != nil {
		copy(out.bytes[out.written:], value)
	} else if out.text != nil {
		out.text.WriteString(value)
	}
	out.written += len(value)
}

func (out *htmlSink) escaped(value string) {
	if out.err != nil {
		return
	}
	// Escaping cannot shorten a fragment. Refuse obviously oversized raw runs
	// without scanning them or materializing an escaped intermediate string.
	if len(value) > out.limit-out.written {
		out.err = ErrOutputLimit
		return
	}
	if out.bytes == nil && out.text == nil {
		// Counting needs only entity expansion deltas, not simulated writes of
		// every run/entity. The same entity function serves the fill path.
		remaining := out.limit - out.written
		size := len(value)
		for index := 0; index < len(value); index++ {
			if entity := htmlEscapeEntity(value[index]); entity != "" {
				extra := len(entity) - 1
				if extra > remaining-size {
					out.err = ErrOutputLimit
					return
				}
				size += extra
			}
		}
		out.written += size
		return
	}
	start := 0
	for index := 0; index < len(value); index++ {
		entity := htmlEscapeEntity(value[index])
		if entity == "" {
			continue
		}
		if start < index {
			out.WriteString(value[start:index])
		}
		out.WriteString(entity)
		if out.err != nil {
			return
		}
		start = index + 1
	}
	out.WriteString(value[start:])
}

func htmlEscapeEntity(char byte) string {
	switch char {
	case '&':
		return "&amp;"
	case '\'':
		return "&#39;"
	case '<':
		return "&lt;"
	case '>':
		return "&gt;"
	case '"':
		return "&#34;"
	default:
		return ""
	}
}

func exportWarning(warnings *[]Warning, code, message string, rng SourceRange) {
	if warnings != nil {
		*warnings = appendSyntaxWarning(*warnings, code, message, rng)
	}
}

func safeURL(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" || strings.HasPrefix(value, "#") || strings.HasPrefix(value, "mailto:") {
		return value, true
	}
	if strings.HasPrefix(value, "//") {
		return "", false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "" || strings.ContainsRune(value, '\\') {
		return "", false
	}
	return value, true
}

func validateInlineRanges(spans []InlineSpan, text string, base int) bool {
	cursor := 0
	for _, span := range spans {
		start := int(span.Range.Start) - base
		end := int(span.Range.End) - base
		if start < cursor || start < 0 || start > end || end > len(text) {
			return false
		}
		cursor = end
	}
	return true
}

func renderInline(out *htmlSink, spans []InlineSpan, text string, base int, warnings *[]Warning) {
	if out.err != nil {
		return
	}
	if !validateInlineRanges(spans, text, base) {
		exportWarning(warnings, "export-range",
			"inline span does not match source; text was exported without formatting",
			rangeAt(base, base+len(text)))
		out.escaped(text)
		return
	}
	if len(spans) == 0 {
		out.escaped(text)
		return
	}
	cursor := 0
	for _, span := range spans {
		start := int(span.Range.Start) - base
		end := int(span.Range.End) - base
		out.escaped(text[cursor:start])
		if out.err != nil {
			return
		}
		switch span.Kind {
		case InlineBold:
			out.WriteString("<strong>")
			out.escaped(span.Text)
			out.WriteString("</strong>")
		case InlineItalic:
			out.WriteString("<em>")
			out.escaped(span.Text)
			out.WriteString("</em>")
		case InlineCode:
			out.WriteString("<code>")
			out.escaped(span.Text)
			out.WriteString("</code>")
		case InlineLink:
			if href, ok := safeURL(span.Href); ok {
				out.WriteString("<a href=\"")
				out.escaped(href)
				out.WriteString("\">")
				out.escaped(span.Text)
				out.WriteString("</a>")
			} else {
				exportWarning(warnings, "unsafe-link",
					"link was not emitted because its URL is unsafe; label text remains",
					rangeAt(base+start, base+end))
				out.escaped(span.Text)
			}
		case InlineImage:
			// Image references are explicitly reference-only in the native
			// reader. HTML preserves the complete original syntax literally.
			out.escaped(text[start:end])
		default:
			out.escaped(text[start:end])
		}
		if out.err != nil {
			return
		}
		cursor = end
	}
	if cursor < len(text) {
		out.escaped(text[cursor:])
	}
}

func ExportHTML(doc Document) (string, []Warning) {
	html, warnings, err := ExportHTMLChecked(doc)
	if err != nil {
		// Retain the old raw diagnostic surface for non-view callers. A partial
		// document is always signalled by Document.Partial and parse-limit.
		_ = err
	}
	return html, warnings
}

func ExportHTMLChecked(doc Document) (string, []Warning, error) {
	if doc.Partial {
		return "", append([]Warning(nil), doc.Warnings...), ErrPartialDocument
	}
	var builder strings.Builder
	out := htmlSink{limit: int(^uint(0) >> 1), text: &builder}
	warnings := make([]Warning, len(doc.Warnings))
	copy(warnings, doc.Warnings)
	renderHTML(&out, doc, &warnings)
	if out.err != nil {
		return "", nil, out.err
	}
	return builder.String(), warnings, nil
}

// ExportHTMLBytes returns owned HTML with len == cap <= maxBytes. It counts
// using the same renderer before allocating the result, then fills exactly
// once. This bounds result capacity, not source/AST memory or process RSS.
// A partial document takes precedence over an invalid or insufficient budget.
func ExportHTMLBytes(doc Document, maxBytes int) ([]byte, []Warning, error) {
	if doc.Partial {
		return nil, append([]Warning(nil), doc.Warnings...), ErrPartialDocument
	}
	if maxBytes < 0 {
		return nil, nil, ErrOutputLimit
	}
	count := htmlSink{limit: maxBytes}
	renderHTML(&count, doc, nil)
	if count.err != nil {
		return nil, nil, count.err
	}
	out := htmlSink{limit: count.written, bytes: make([]byte, count.written)}
	warnings := make([]Warning, len(doc.Warnings))
	copy(warnings, doc.Warnings)
	renderHTML(&out, doc, &warnings)
	if out.err != nil {
		return nil, nil, out.err
	}
	return out.bytes, warnings, nil
}

func renderHTML(out *htmlSink, doc Document, warnings *[]Warning) {
	out.WriteString("<!doctype html>\n<meta charset=\"utf-8\">\n<body>\n")
	var anchors HeadingAnchors
	for _, block := range doc.Blocks {
		if out.err != nil {
			return
		}
		switch block.Kind {
		case BlockHeading:
			level := clampHeading(block.Level)
			anchor, _ := anchors.Next(block.Text)
			out.WriteString("<h" + strconv.Itoa(level) + " id=\"")
			out.escaped(anchor)
			out.WriteString("\">")
			renderInline(out, block.Inline, block.Text, block.TextRange.Start, warnings)
			out.WriteString("</h" + strconv.Itoa(level) + ">\n")
		case BlockParagraph:
			out.WriteString("<p>")
			renderInline(out, block.Inline, block.Text, block.TextRange.Start, warnings)
			out.WriteString("</p>\n")
		case BlockList:
			tag := "ul"
			if block.Ordered {
				tag = "ol"
				if block.Start != 1 {
					out.WriteString("<" + tag + " start=\"" + strconv.Itoa(block.Start) + "\">\n")
				} else {
					out.WriteString("<" + tag + ">\n")
				}
			} else {
				out.WriteString("<" + tag + ">\n")
			}
			for _, item := range block.Items {
				if out.err != nil {
					return
				}
				out.WriteString("<li>")
				if item.Task != nil {
					out.WriteString(`<input type="checkbox" disabled`)
					if item.Task.Checked {
						out.WriteString(" checked")
					}
					out.WriteString(">")
					if item.Text != "" {
						out.WriteString(" ")
					}
				}
				renderInline(out, item.Inline, item.Text, item.TextRange.Start, warnings)
				out.WriteString("</li>\n")
			}
			out.WriteString("</" + tag + ">\n")
		case BlockQuote:
			renderPre(out, block.Text)
		case BlockCode:
			out.WriteString("<pre><code>")
			out.escaped(block.Text)
			out.WriteString("</code></pre>\n")
		case BlockDivider:
			out.WriteString("<hr>\n")
		case BlockDocBlock:
			renderPre(out, block.Raw)
		case BlockHTML:
			renderPre(out, block.Text)
		case BlockTable:
			renderTable(out, doc, block, warnings)
		}
	}
	out.WriteString("</body>\n")
}

func renderPre(out *htmlSink, text string) {
	out.WriteString("<pre>")
	out.escaped(text)
	out.WriteString("</pre>\n")
}

func renderTable(out *htmlSink, doc Document, block Block, warnings *[]Warning) {
	table := block.Table
	if table.Problem != "" {
		renderPre(out, block.Raw)
		return
	}
	if !validExportTable(doc.Raw(), block, table) {
		exportWarning(warnings, "export-table",
			"table descriptor is invalid; complete table source was exported as raw text",
			block.Range)
		renderPre(out, block.Raw)
		return
	}
	out.WriteString("<table>\n<thead>\n<tr>")
	for i, cell := range table.Rows[0].Cells {
		if out.err != nil {
			return
		}
		writeTableCell(out, "th", table.Alignments[i], doc.Raw()[cell.TextRange.Start:cell.TextRange.End])
	}
	out.WriteString("</tr>\n</thead>\n")
	if len(table.Rows) > 1 {
		out.WriteString("<tbody>\n")
		for _, row := range table.Rows[1:] {
			if out.err != nil {
				return
			}
			out.WriteString("<tr>")
			for i, cell := range row.Cells {
				if out.err != nil {
					return
				}
				writeTableCell(out, "td", table.Alignments[i], doc.Raw()[cell.TextRange.Start:cell.TextRange.End])
			}
			out.WriteString("</tr>\n")
		}
		out.WriteString("</tbody>\n")
	}
	out.WriteString("</table>\n")
}

func validExportTable(raw string, block Block, table TableRef) bool {
	if block.Range.Start < 0 || block.Range.End < block.Range.Start ||
		block.Range.End > len(raw) || block.Range.Start > block.Range.End ||
		len(block.Raw) != block.Range.End-block.Range.Start ||
		block.Raw != raw[block.Range.Start:block.Range.End] ||
		len(block.Raw) > maxTableRawBytes ||
		table.Problem != "" ||
		len(table.Rows) == 0 || len(table.Rows) > maxTableBodyRows+1 {
		return false
	}
	delimiter := table.DelimiterRange
	if delimiter.Start < block.Range.Start || delimiter.End < delimiter.Start ||
		delimiter.End > block.Range.End ||
		delimiter.Start < table.Rows[0].Range.End {
		return false
	}
	descriptorCells := 0
	previousRowEnd := block.Range.Start
	for rowIndex, row := range table.Rows {
		if row.Range.Start < block.Range.Start || row.Range.End > block.Range.End ||
			row.Range.Start > row.Range.End ||
			row.Range.Start < previousRowEnd ||
			len(row.Cells) == 0 || len(row.Cells) > maxTableColumns {
			return false
		}
		if rowIndex == 0 {
			if row.Range.Start != block.Range.Start || row.Range.End != delimiter.Start {
				return false
			}
		} else if row.Range.Start < delimiter.End {
			return false
		}
		previousCellEnd := row.Range.Start
		for _, cell := range row.Cells {
			if cell.Range.Start < previousCellEnd ||
				cell.Range.Start < row.Range.Start || cell.Range.End > row.Range.End ||
				cell.TextRange.Start < cell.Range.Start || cell.TextRange.End > cell.Range.End ||
				cell.TextRange.Start > cell.TextRange.End {
				return false
			}
			previousCellEnd = cell.Range.End
		}
		previousRowEnd = row.Range.End
		descriptorCells += len(row.Cells)
		if descriptorCells > maxTableDescriptorCells {
			return false
		}
	}
	if len(table.Alignments) != len(table.Rows[0].Cells) {
		return false
	}
	for _, row := range table.Rows {
		if len(row.Cells) != len(table.Alignments) {
			return false
		}
	}
	for _, alignment := range table.Alignments {
		if alignment > TableAlignRight {
			return false
		}
	}
	return true
}

func writeTableCell(out *htmlSink, tag string, alignment TableAlignment, text string) {
	switch alignment {
	case TableAlignLeft:
		out.WriteString("<" + tag + ` style="text-align:left">`)
	case TableAlignCenter:
		out.WriteString("<" + tag + ` style="text-align:center">`)
	case TableAlignRight:
		out.WriteString("<" + tag + ` style="text-align:right">`)
	default:
		out.WriteString("<" + tag + ">")
	}
	out.escaped(text)
	out.WriteString("</" + tag + ">")
}

func clampHeading(level int) int {
	if level < 1 {
		return 1
	}
	if level > 6 {
		return 6
	}
	return level
}
