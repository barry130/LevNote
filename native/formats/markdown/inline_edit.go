package markdown

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// InlineEdit is one source-byte replacement, plus the selection in the result.
// Neither parsing nor planning normalizes line endings or untouched source.
type InlineEdit struct {
	Range     SourceRange
	Text      string
	Selection SourceRange
}

var (
	errInlineSelection    = errors.New("请选择完整文字，避免把选区放在字符或格式标记中间")
	errInlineMultiline    = errors.New("行内格式仅处理单行文字；请分行操作，或在源码中编辑多行格式")
	errInlineNested       = errors.New("当前预览仅支持单层行内格式；请先取消已有格式，再选择另一种格式")
	errInlineBoundary     = errors.New("所选文字与相邻格式标记相连；请调整选区，或在源码中编辑")
	errInlineCodeBoundary = errors.New("代码边界的反引号或两端空格会改变当前解析结果；请调整选区，或在源码中编辑")
	errInlineBudget       = errors.New("当前块超过行内格式处理上限；请在源码中编辑")
)

// PlanInlineEdit toggles a complete existing mark, or wraps a plain selection.
// A caret inside a mark removes that whole mark without moving within its text.
// An empty plain selection inserts a delimiter pair with the caret between it.
// Nested/partial marks and ambiguous delimiter boundaries are left unchanged.
func PlanInlineEdit(source string, selection SourceRange, kind InlineKind) (InlineEdit, error) {
	marker := ""
	switch kind {
	case InlineBold:
		marker = "**"
	case InlineItalic:
		marker = "*"
	case InlineCode:
		marker = "`"
	default:
		return InlineEdit{}, errInlineSelection
	}
	if !utf8.ValidString(source) || selection.Start < 0 || selection.End < selection.Start || selection.End > len(source) ||
		!inlineByteBoundary(source, selection.Start) || !inlineByteBoundary(source, selection.End) {
		return InlineEdit{}, errInlineSelection
	}
	if len(source) > maxInlineScanSuffix {
		return InlineEdit{}, errInlineBudget
	}
	if strings.ContainsAny(source[selection.Start:selection.End], "\r\n") {
		return InlineEdit{}, errInlineMultiline
	}
	before, err := inlineEditSpans(source)
	if err != nil {
		return InlineEdit{}, err
	}
	// Empty pairs produced by this command may be incomplete syntax until the
	// next character is typed (notably ** for italic and two backticks).
	if selection.Start == selection.End {
		p, n := selection.Start, len(marker)
		if p >= n && p+n <= len(source) && source[p-n:p] == marker && source[p:p+n] == marker &&
			(p == n || source[p-n-1] != marker[0]) && (p+n == len(source) || source[p+n] != marker[0]) {
			plan := InlineEdit{Range: rangeAt(p-n, p+n), Selection: rangeAt(p-n, p-n)}
			if err := validateInlineEdit(source, before, plan, nil); err != nil {
				return InlineEdit{}, err
			}
			return plan, nil
		}
	}
	for _, span := range before {
		content, ok := inlineContentRange(source, span)
		caret := selection.Start == selection.End && ok && selection.Start >= content.Start && selection.Start <= content.End
		whole := selection == span.Range || ok && selection == content
		if caret || whole {
			if span.Kind != kind {
				return InlineEdit{}, errInlineNested
			}
			if !ok || strings.ContainsAny(source[content.Start:content.End], "\r\n") {
				return InlineEdit{}, errInlineMultiline
			}
			value := source[content.Start:content.End]
			if value != span.Text {
				return InlineEdit{}, errInlineCodeBoundary
			}
			selected := rangeAt(span.Range.Start, span.Range.Start+len(value))
			if caret {
				pos := span.Range.Start + selection.Start - content.Start
				selected = rangeAt(pos, pos)
			}
			plan := InlineEdit{Range: span.Range, Text: value, Selection: selected}
			if err := validateInlineEdit(source, before, plan, nil); err != nil {
				return InlineEdit{}, err
			}
			return plan, nil
		}
		if selection.Start < span.Range.End && selection.End > span.Range.Start ||
			selection.Start == selection.End && selection.Start > span.Range.Start && selection.Start < span.Range.End {
			return InlineEdit{}, errInlineNested
		}
	}
	value := source[selection.Start:selection.End]
	if kind == InlineCode && value != "" {
		if value[0] == '`' || value[len(value)-1] == '`' || value[0] == ' ' && value[len(value)-1] == ' ' {
			return InlineEdit{}, errInlineCodeBoundary
		}
		run, longest := 0, 0
		for i := range value {
			if value[i] == '`' {
				run++
				longest = max(longest, run)
			} else {
				run = 0
			}
		}
		if longest >= maxInlineDelimiterRun {
			return InlineEdit{}, errInlineCodeBoundary
		}
		marker = strings.Repeat("`", longest+1)
	}
	plan := InlineEdit{Range: selection, Text: marker + value + marker,
		Selection: rangeAt(selection.Start+len(marker), selection.Start+len(marker)+len(value))}
	// Validate an empty insertion using a temporary payload; the actual plan
	// still inserts only the pair. No placeholder enters the user's buffer.
	probe, payload := plan, value
	if value == "" {
		payload = "文"
		probe.Text = marker + payload + marker
	}
	expected := InlineSpan{Kind: kind, Range: rangeAt(selection.Start, selection.Start+len(probe.Text)), Text: payload}
	if err := validateInlineEdit(source, before, probe, &expected); err != nil {
		return InlineEdit{}, err
	}
	if value == "" {
		// An empty pair can parse differently from the payload probe and consume
		// a later mark. Validate the exact bytes that will reach the buffer too.
		empty := InlineSpan{Kind: kind, Range: rangeAt(selection.Start, selection.Start+len(plan.Text))}
		if err := validateInlineEdit(source, before, plan, &empty); err != nil {
			return InlineEdit{}, err
		}
	}
	return plan, nil
}

func inlineByteBoundary(source string, index int) bool {
	return index == len(source) || utf8.RuneStart(source[index])
}

func inlineEditSpans(source string) ([]InlineSpan, error) {
	limit := &inlineLimit{}
	spans, _ := parseInline(source, 0, limit)
	if limit.exhausted {
		return nil, errInlineBudget
	}
	return spans, nil
}

func inlineContentRange(source string, span InlineSpan) (SourceRange, bool) {
	r := span.Range
	n := 0
	switch span.Kind {
	case InlineBold:
		n = 2
	case InlineItalic:
		n = 1
	case InlineCode:
		for r.Start+n < r.End && source[r.Start+n] == '`' {
			n++
		}
	default:
		return SourceRange{}, false
	}
	return rangeAt(r.Start+n, r.End-n), n > 0 && r.Start+n <= r.End-n
}

// Reparse the candidate through the production scanner. In addition to the
// requested span, every unrelated mark must retain its exact kind/text/href.
func validateInlineEdit(source string, before []InlineSpan, plan InlineEdit, added *InlineSpan) error {
	candidate := source[:plan.Range.Start] + plan.Text + source[plan.Range.End:]
	after, err := inlineEditSpans(candidate)
	if err != nil {
		return err
	}
	remaining := make([]InlineSpan, 0, len(after))
	// Empty delimiter pairs need not form a span until text is typed. Any
	// span they do form must match this exact range, kind and empty content.
	found := added == nil || added.Text == ""
	for _, span := range after {
		if added != nil && span == *added {
			found = true
		} else {
			remaining = append(remaining, span)
		}
	}
	if !found {
		return errInlineBoundary
	}
	delta, index := len(plan.Text)-(plan.Range.End-plan.Range.Start), 0
	for _, span := range before {
		if added == nil && span.Range == plan.Range {
			continue
		}
		if span.Range.Start >= plan.Range.End {
			span.Range.Start += delta
			span.Range.End += delta
		}
		if index >= len(remaining) || span != remaining[index] {
			return errInlineBoundary
		}
		index++
	}
	if index != len(remaining) {
		return errInlineBoundary
	}
	return nil
}
