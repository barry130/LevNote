package code

import (
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

func LanguageForPath(path string) Language {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return LanguageGo
	case ".ts", ".tsx":
		return LanguageTypeScript
	case ".js", ".jsx", ".mjs", ".cjs":
		return LanguageJavaScript
	case ".py", ".pyw":
		return LanguagePython
	case ".json", ".jsonc", ".mindmap":
		return LanguageJSON
	case ".rs":
		return LanguageRust
	case ".xml", ".svg", ".drawio", ".dio":
		return LanguageXML
	}
	return ""
}

func languageFor(language Language) string {
	switch language {
	case LanguageGo, LanguageTypeScript, LanguageJavaScript, LanguagePython, LanguageJSON, LanguageRust, LanguageXML:
		return string(language)
	case "ts":
		return string(LanguageTypeScript)
	case "js":
		return string(LanguageJavaScript)
	}
	return ""
}

func Highlight(source string, language Language) Result {
	result := Result{Spans: []Span{}}
	normalized := languageFor(language)
	if normalized == "" {
		result.Warnings = append(result.Warnings, Warning{
			Code: "language", Message: "unsupported language", Range: SourceRange{Start: 0, End: len(source)},
		})
		return result
	}
	if len(source) > maxHighlightBytes {
		result.Warnings = append(result.Warnings, Warning{
			Code: "large-file", Message: "file exceeds 2 MiB highlight limit", Range: SourceRange{Start: 0, End: len(source)},
		})
		return result
	}
	if normalized == string(LanguageXML) {
		result = highlightXML(source)
	} else {
		result = highlightTokenized(source, keywordSet(normalized), normalized)
	}
	if workOf(len(source), len(result.Spans)) > maxHighlightWork {
		return Result{Spans: []Span{}, Warnings: []Warning{{
			Code: "complexity", Message: "highlight stopped because the file exceeded the work limit",
			Range: SourceRange{Start: 0, End: len(source)},
		}}}
	}
	return result
}

// HighlightDeclarative applies line comments, block comments, and keywords
// from a plugin manifest. Unknown words remain plain text.
func HighlightDeclarative(source string, syntax DeclarativeSyntax) Result {
	result := Result{Spans: []Span{}}
	if len(source) > maxHighlightBytes {
		result.Warnings = append(result.Warnings, Warning{
			Code: "large-file", Message: "file exceeds 2 MiB highlight limit", Range: SourceRange{Start: 0, End: len(source)},
		})
		return result
	}
	keywords := map[string]bool{}
	for _, keyword := range syntax.Keywords {
		if keyword != "" {
			keywords[keyword] = true
		}
	}
	var tokens []Span
	i := 0
	work := 0
	for i < len(source) {
		work++
		if work > maxHighlightWork {
			break
		}
		switch {
		case syntax.BlockStart != "" && strings.HasPrefix(source[i:], syntax.BlockStart):
			end := strings.Index(source[i+len(syntax.BlockStart):], syntax.BlockEnd)
			if end >= 0 {
				end = i + len(syntax.BlockStart) + end + len(syntax.BlockEnd)
			} else {
				end = len(source)
			}
			tokens = append(tokens, Span{Start: i, End: end, Token: TokenComment, Kind: "comment"})
			i = end
			continue
		case syntax.LineComment != "" && strings.HasPrefix(source[i:], syntax.LineComment):
			end := indexOfNewline(source, i)
			if end == i {
				_, width := decodeRune(source[i:])
				end = i + max(width, 1)
			}
			tokens = append(tokens, Span{Start: i, End: end, Token: TokenComment, Kind: "comment"})
			i = end
			continue
		case source[i] == '"':
			end := readString(source, i, '"', true)
			tokens = append(tokens, Span{Start: i, End: end, Token: TokenString, Kind: "string"})
			i = end
			continue
		case source[i] == '\'':
			end := readString(source, i, '\'', true)
			tokens = append(tokens, Span{Start: i, End: end, Token: TokenString, Kind: "string"})
			i = end
			continue
		case source[i] == '`':
			end := strings.IndexByte(source[i+1:], '`')
			if end >= 0 {
				end = i + end + 2
			} else {
				end = len(source)
			}
			tokens = append(tokens, Span{Start: i, End: end, Token: TokenString, Kind: "string"})
			i = end
			continue
		}

		r, width := decodeRune(source[i:])
		if isDigit(byte(r)) && r < 0x80 {
			end := readNumber(source, i)
			tokens = append(tokens, Span{Start: i, End: end, Token: TokenNumber, Kind: "number"})
			i = end
			continue
		}
		if isIdentStart(r) {
			start := i
			i += width
			for i < len(source) {
				next, nextWidth := decodeRune(source[i:])
				if !isIdentPart(next) {
					break
				}
				i += nextWidth
				work++
			}
			token := TokenName
			if keywords[source[start:i]] {
				token = TokenKeyword
			}
			tokens = append(tokens, Span{Start: start, End: i, Token: token, Kind: "word"})
			continue
		}
		tokens = append(tokens, Span{Start: i, End: i + width, Token: TokenPunct, Kind: "punct"})
		i += width
	}
	result = spansFrom(source, tokens)
	if workOf(len(source), len(result.Spans)) > maxHighlightWork {
		return Result{Spans: []Span{}, Warnings: []Warning{{
			Code: "complexity", Message: "highlight stopped because the file exceeded the work limit",
			Range: SourceRange{Start: 0, End: len(source)},
		}}}
	}
	return result
}

func workOf(length, spans int) int { return length + spans*64 }

func keywordSet(language string) map[string]bool {
	set := map[string]bool{}
	for _, keyword := range keywords(language) {
		set[keyword] = true
	}
	return set
}

func keywords(language string) []string {
	switch language {
	case string(LanguageGo):
		return []string{"break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough", "for", "func", "go", "goto", "if", "import", "interface", "map", "package", "range", "return", "select", "struct", "switch", "type", "var"}
	case string(LanguageTypeScript):
		return []string{"abstract", "any", "as", "boolean", "break", "case", "catch", "class", "const", "continue", "declare", "default", "do", "else", "enum", "export", "extends", "false", "finally", "for", "from", "function", "if", "implements", "import", "in", "instanceof", "interface", "let", "new", "null", "number", "of", "private", "protected", "public", "readonly", "return", "static", "string", "super", "switch", "this", "throw", "true", "try", "type", "typeof", "undefined", "var", "void", "while"}
	case string(LanguageJavaScript):
		return []string{"await", "break", "case", "catch", "class", "const", "continue", "default", "do", "else", "export", "extends", "false", "finally", "for", "from", "function", "if", "import", "in", "instanceof", "let", "new", "null", "of", "return", "static", "super", "switch", "this", "throw", "true", "try", "typeof", "undefined", "var", "void", "while", "yield"}
	case string(LanguagePython):
		return []string{"and", "as", "assert", "async", "await", "break", "class", "continue", "def", "del", "elif", "else", "except", "False", "finally", "for", "from", "global", "if", "import", "in", "is", "lambda", "None", "nonlocal", "not", "or", "pass", "raise", "return", "True", "try", "while", "with", "yield"}
	case string(LanguageJSON):
		return []string{"false", "null", "true"}
	case string(LanguageRust):
		return []string{"as", "async", "await", "break", "const", "continue", "crate", "dyn", "else", "enum", "extern", "false", "fn", "for", "if", "impl", "in", "let", "loop", "match", "mod", "move", "mut", "pub", "ref", "return", "self", "static", "struct", "super", "trait", "true", "type", "unsafe", "use", "where", "while"}
	}
	return nil
}

func spansFrom(source string, tokens []Span) Result {
	spans := make([]Span, 0, len(tokens)+4)
	cursor := 0
	for _, token := range tokens {
		if token.Start < cursor || token.End <= token.Start || token.End > len(source) {
			continue
		}
		if token.Start > cursor {
			spans = append(spans, Span{Start: cursor, End: token.Start, Token: TokenText, Kind: "plain"})
		}
		spans = append(spans, token)
		cursor = token.End
	}
	if cursor < len(source) {
		spans = append(spans, Span{Start: cursor, End: len(source), Token: TokenText, Kind: "plain"})
	}
	return Result{Spans: spans}
}

func isIdentStart(r rune) bool { return unicode.IsLetter(r) || r == '_' || r == '$' || r == '@' }
func isIdentPart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '$' || r == '@'
}

func highlightTokenized(source string, keywords map[string]bool, language string) Result {
	var tokens []Span
	i := 0
	work := 0
	for i < len(source) {
		work++
		if work > maxHighlightWork {
			break
		}
		r, width := decodeRune(source[i:])
		width = max(width, 1)

		switch {
		case language == string(LanguagePython) && strings.HasPrefix(source[i:], `"""`):
			if end := strings.Index(source[i+3:], `"""`); end >= 0 {
				tokens = append(tokens, Span{Start: i, End: i + 6 + end, Token: TokenString, Kind: "string"})
				i += 6 + end
				continue
			}
			tokens = append(tokens, Span{Start: i, End: len(source), Token: TokenString, Kind: "string"})
			i = len(source)
		case language == string(LanguagePython) && strings.HasPrefix(source[i:], `'''`):
			if end := strings.Index(source[i+3:], `'''`); end >= 0 {
				tokens = append(tokens, Span{Start: i, End: i + 6 + end, Token: TokenString, Kind: "string"})
				i += 6 + end
				continue
			}
			tokens = append(tokens, Span{Start: i, End: len(source), Token: TokenString, Kind: "string"})
			i = len(source)
		case source[i] == '"':
			end := readString(source, i, '"', true)
			tokens = append(tokens, Span{Start: i, End: end, Token: TokenString, Kind: "string"})
			i = end
		case source[i] == '\'':
			end := readString(source, i, '\'', language == string(LanguageRust))
			tokens = append(tokens, Span{Start: i, End: end, Token: TokenString, Kind: "string"})
			i = end
		case language == string(LanguageGo) || language == string(LanguageRust) || language == string(LanguageTypeScript) || language == string(LanguageJavaScript):
			if strings.HasPrefix(source[i:], "//") {
				end := indexOfNewline(source, i)
				tokens = append(tokens, Span{Start: i, End: end, Token: TokenComment, Kind: "comment"})
				i = end
				continue
			}
			if strings.HasPrefix(source[i:], "/*") {
				end := strings.Index(source[i+2:], "*/")
				if end >= 0 {
					end = i + 4 + end
				} else {
					end = len(source)
				}
				tokens = append(tokens, Span{Start: i, End: end, Token: TokenComment, Kind: "comment"})
				i = end
				continue
			}
			if strings.HasPrefix(source[i:], "`") {
				end := strings.IndexByte(source[i+1:], '`')
				if end >= 0 {
					end = i + 2 + end
				} else {
					end = len(source)
				}
				tokens = append(tokens, Span{Start: i, End: end, Token: TokenString, Kind: "raw"})
				i = end
				continue
			}
		case language == string(LanguagePython):
			if strings.HasPrefix(source[i:], "#") {
				end := indexOfNewline(source, i)
				tokens = append(tokens, Span{Start: i, End: end, Token: TokenComment, Kind: "comment"})
				i = end
				continue
			}
		}

		if isDigit(byte(r)) && !(language == string(LanguageRust) && i+1 < len(source) && source[i+1] == '\'') {
			end := readNumber(source, i)
			tokens = append(tokens, Span{Start: i, End: end, Token: TokenNumber, Kind: "number"})
			i = end
			continue
		}

		if isIdentStart(r) {
			start := i
			i += width
			for i < len(source) {
				decoded, width := decodeRune(source[i:])
				if !isIdentPart(decoded) {
					break
				}
				i += width
				work++
			}
			word := source[start:i]
			token := TokenName
			if keywords[word] {
				token = TokenKeyword
			}
			tokens = append(tokens, Span{Start: start, End: i, Token: token, Kind: "word"})
			continue
		}

		tokens = append(tokens, Span{Start: i, End: i + width, Token: TokenPunct, Kind: "punct"})
		i += width
	}
	return spansFrom(source, tokens)
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

func decodeRune(source string) (rune, int) {
	r, width := utf8.DecodeRuneInString(source)
	if width == 0 {
		return r, 1
	}
	return r, width
}

func indexOfNewline(source string, start int) int {
	for i := start; i < len(source); i++ {
		if source[i] == '\n' {
			return i
		}
		if source[i] == '\r' {
			return i
		}
	}
	return len(source)
}

func readString(source string, start int, quote byte, escapes bool) int {
	for i := start + 1; i < len(source); i++ {
		if escapes && source[i] == '\\' {
			i++
			continue
		}
		if source[i] == quote {
			return i + 1
		}
		if source[i] == '\n' {
			return i
		}
	}
	return len(source)
}

func readNumber(source string, start int) int {
	i := start + 1
	for i < len(source) {
		b := source[i]
		if isDigit(b) || b == '.' || b == 'x' || b == 'X' || b == 'e' || b == 'E' || b == '+' || b == '-' || b == '_' {
			if (b == '+' || b == '-') && !(i > start+1 && (source[i-1] == 'e' || source[i-1] == 'E')) {
				break
			}
			i++
			continue
		}
		break
	}
	return i
}

func highlightXML(source string) Result {
	var tokens []Span
	i := 0
	for i < len(source) {
		start := strings.IndexByte(source[i:], '<')
		if start < 0 {
			tokens = append(tokens, Span{Start: i, End: len(source), Token: TokenText, Kind: "plain"})
			break
		}
		if start > 0 {
			tokens = append(tokens, Span{Start: i, End: i + start, Token: TokenText, Kind: "plain"})
		}
		i += start
		end := strings.IndexByte(source[i:], '>')
		if end < 0 {
			tokens = append(tokens, Span{Start: i, End: len(source), Token: TokenTag, Kind: "tag"})
			break
		}
		tokens = append(tokens, Span{Start: i, End: i + end + 1, Token: TokenTag, Kind: "tag"})
		i += end + 1
	}
	return spansFrom(source, tokens)
}
