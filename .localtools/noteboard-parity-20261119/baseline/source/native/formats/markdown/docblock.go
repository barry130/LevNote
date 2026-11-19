package markdown

import (
	"regexp"
	"strings"
)

const docBlockInfo = "doc-block"

const maxReferenceProblemBytes = 256

var (
	docBlockOpenRe = regexp.MustCompile(`^( {0,3})(` + "`{3,}" + `)[ \t]*doc-block[ \t]*$`)
	headerLineRe   = regexp.MustCompile(`^([a-z][a-z0-9_-]*)[ \t]*:[ \t]*(.*)$`)
	docIDRe        = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)
)

var docBlockRecognizedKeys = map[string]struct{}{
	"id": {}, "kind": {}, "mode": {}, "src": {}, "title": {},
}

func parseDocBlock(source string, lines []line, index int) (Block, int, []Warning, bool) {
	open := docBlockOpenRe.FindStringSubmatch(lines[index].content)
	if open == nil {
		return Block{}, index, nil, false
	}

	fenceLength := len(open[2])
	terminator := -1
	headerEnd := -1
	for i := index + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i].content) == "---" {
			terminator = i
			headerEnd = i
			break
		}
		if match := fenceOnlyRe.FindStringSubmatch(lines[i].content); match != nil && len(match[2]) >= fenceLength {
			break
		}
	}
	if terminator < 0 {
		headerEnd = index
	}

	closeIndex := -1
	for i := headerEnd + 1; i < len(lines); i++ {
		if match := fenceOnlyRe.FindStringSubmatch(lines[i].content); match != nil && len(match[2]) >= fenceLength {
			closeIndex = i
			break
		}
	}

	rawStart := lines[index].start
	rawEnd := len(source)
	if closeIndex >= 0 {
		rawEnd = lines[closeIndex].end
	}
	raw := source[rawStart:rawEnd]

	if terminator < 0 {
		warnings := []Warning{}
		warnings = addWarning(warnings,
			"doc-block separator is missing; source is retained as raw code",
			rangeAt(rawStart, rawEnd))
		if closeIndex < 0 {
			warnings = addWarning(warnings,
				"unterminated doc-block left visible", rangeAt(rawStart, rawEnd))
		}
		next := len(lines)
		if closeIndex >= 0 {
			next = closeIndex + 1
		}
		return Block{
			Kind: BlockCode, Range: rangeAt(rawStart, rawEnd),
			TextRange: rangeAt(rawStart, rawEnd), Text: raw, Language: docBlockInfo,
			Raw: raw,
		}, next, warnings, true
	}

	headerRange := rangeAt(lines[index].end, lines[headerEnd].start)
	ref := DocBlockRef{FenceLength: fenceLength, HeaderRange: headerRange}
	payloadStart := lines[terminator].end
	payloadEnd := rawEnd
	if closeIndex >= 0 {
		payloadEnd = lines[closeIndex].start
	}
	ref.PayloadRange = rangeAt(payloadStart, payloadEnd)
	ref.Payload = source[payloadStart:payloadEnd]

	var warnings []Warning
	headerLines := lines[index+1 : headerEnd]
	rawAttrs := map[string]string{}
	duplicates := map[string]struct{}{}
	srcPresent := false
	for i := range headerLines {
		header := headerLines[i]
		if isBlank(header.content) {
			continue
		}
		match := headerLineRe.FindStringSubmatch(header.content)
		if match == nil {
			warnings = addWarning(warnings, "doc-block header line ignored", lineRange(header))
			continue
		}
		key := strings.ToLower(match[1])
		if key == "src" {
			srcPresent = true
		}
		if _, recognized := docBlockRecognizedKeys[key]; recognized {
			if _, exists := rawAttrs[key]; exists {
				duplicates[key] = struct{}{}
			}
		}
		rawAttrs[key] = strings.TrimSpace(match[2])
	}
	ref.Attrs = DocBlockAttrs{
		ID: rawAttrs["id"], Kind: rawAttrs["kind"], Mode: rawAttrs["mode"],
		Src: rawAttrs["src"], Title: rawAttrs["title"], Extra: map[string]string{},
	}
	for key, value := range rawAttrs {
		switch key {
		case "id", "kind", "mode", "src", "title":
		default:
			ref.Attrs.Extra[key] = value
		}
	}
	if id, seen := rawAttrs["id"]; seen {
		if !docIDRe.MatchString(id) {
			warnings = addSyntaxWarning(warnings, "doc-block-bad-id",
				"doc-block id is invalid; reference is preserved", rangeAt(rawStart, rawEnd))
		}
	} else {
		warnings = addWarning(warnings, "doc-block id is missing or invalid", rangeAt(rawStart, rawEnd))
	}
	if ref.Attrs.Kind == "" {
		warnings = addWarning(warnings, "doc-block kind is missing; retained as text", rangeAt(rawStart, rawEnd))
	}
	if closeIndex < 0 {
		warnings = addWarning(warnings, "unterminated doc-block left visible", rangeAt(rawStart, rawEnd))
	}
	ref.ReferenceProblem = referenceProblem(ref, duplicates, srcPresent, closeIndex < 0)

	next := len(lines)
	if closeIndex >= 0 {
		next = closeIndex + 1
	}
	return Block{
		Kind: BlockDocBlock, Range: rangeAt(rawStart, rawEnd), Raw: raw, DocBlock: ref,
	}, next, warnings, true
}

func referenceProblem(ref DocBlockRef, duplicates map[string]struct{}, srcPresent, unterminated bool) string {
	problems := make([]string, 0, 8)
	if unterminated {
		problems = append(problems, "unterminated fence")
	}
	for _, key := range []string{"id", "kind", "mode", "src", "title"} {
		if _, exists := duplicates[key]; exists {
			problems = append(problems, "duplicate "+key)
		}
	}
	if ref.Attrs.ID == "" || !docIDRe.MatchString(ref.Attrs.ID) {
		problems = append(problems, "id missing or invalid")
	}
	if ref.Attrs.Kind == "" {
		problems = append(problems, "kind missing")
	}
	switch ref.Attrs.Mode {
	case "reference":
		if !srcPresent || ref.Attrs.Src == "" {
			problems = append(problems, "src missing or empty in reference mode")
		}
	case "inline":
		if srcPresent {
			problems = append(problems, "src is not opened in inline mode")
		}
	default:
		problems = append(problems, "mode missing or unknown")
	}
	return boundedReferenceProblem(problems)
}

func boundedReferenceProblem(problems []string) string {
	var builder strings.Builder
	for _, problem := range problems {
		if builder.Len() > 0 {
			builder.WriteString("; ")
		}
		if builder.Len()+len(problem)+len("; …") > maxReferenceProblemBytes {
			if builder.Len() > 0 {
				if builder.Len()+len("…") <= maxReferenceProblemBytes {
					builder.WriteString("…")
				}
			}
			break
		}
		builder.WriteString(problem)
	}
	return builder.String()
}
