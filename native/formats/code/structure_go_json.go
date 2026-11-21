package code

import (
	"go/parser"
	"go/scanner"
	"go/token"
	"strings"
)

// The scanner callback has no stop return value. Abort it immediately on a
// rejected diagnostic, even when one long malformed literal emits many errors.
type stopStructureGoScan struct{ err error }

func (s *structureScan) scanGo() (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			stop, ok := recovered.(stopStructureGoScan)
			if !ok {
				panic(recovered)
			}
			err = stop.err
		}
	}()

	// Byte/line/UTF-8 admission happened before this bounded copy. The same
	// bytes are used by the scanner and, only after the structural scan, parser.
	source := []byte(s.source)
	file := token.NewFileSet().AddFile("", -1, len(source))
	var lex scanner.Scanner
	lex.Init(file, source, func(pos token.Position, message string) {
		// Offset is physical even for tokens affected by a //line directive.
		if err := s.problem(pos.Offset, "go-lexical", message); err != nil {
			panic(stopStructureGoScan{err})
		}
	}, scanner.ScanComments)

	for {
		pos, item, _ := lex.Scan()
		if item == token.EOF {
			break
		}
		if err := s.consumeToken(); err != nil {
			return err
		}
		offset := file.Offset(pos)
		switch item {
		case token.LPAREN:
			err = s.push(offset, ')', 0)
		case token.LBRACK:
			err = s.push(offset, ']', 0)
		case token.LBRACE:
			err = s.push(offset, '}', 0)
		case token.RPAREN, token.RBRACK, token.RBRACE:
			if len(s.stack) == 0 || s.stack[len(s.stack)-1].close != source[offset] {
				err = s.problem(offset, "go-delimiter", "unexpected closing delimiter")
			} else {
				err = s.pop(offset)
			}
		}
		if err != nil {
			return err
		}
	}
	if len(s.stack) != 0 {
		frame := s.stack[len(s.stack)-1]
		if err := s.problem(frame.open.Bytes.Start, "go-delimiter", "unclosed delimiter"); err != nil {
			return err
		}
	}
	if len(s.problems) != 0 {
		return nil
	}
	return s.validateGoGrammar(source)
}

func (s *structureScan) validateGoGrammar(source []byte) error {
	// The non-nil source selects the in-memory path. No imports, name
	// resolution, comments AST, or source-file reads. Do not request AllErrors:
	// syntax recovery is bounded by the parser's default diagnostic limit.
	_, err := parser.ParseFile(token.NewFileSet(), "", source, parser.SkipObjectResolution)
	if err == nil {
		return nil
	}
	problems, ok := err.(scanner.ErrorList)
	if !ok || len(problems) == 0 {
		return err // fail closed on any unexpected validator failure
	}
	first := problems[0]
	for _, problem := range problems {
		// The pinned Go parser can independently bound deep expressions whose
		// nesting is not represented by delimiters (e.g. long unary chains).
		if problem.Msg == "exceeded max nesting depth" {
			return ErrStructureValidationLimit
		}
		if problem.Pos.Offset < first.Pos.Offset {
			first = problem
		}
	}
	return s.problem(first.Pos.Offset, "go-syntax", first.Msg)
}

type structureJSONState uint8

const (
	jsonArrayFirst structureJSONState = iota
	jsonArrayValue
	jsonArrayAfterValue
	jsonObjectFirst
	jsonObjectKey
	jsonObjectColon
	jsonObjectValue
	jsonObjectAfterValue
)

// scanJSON recognizes exactly one RFC 8259 value, including a scalar. Grammar
// state resides in the explicit delimiter stack, not recursive parser calls.
func (s *structureScan) scanJSON() error {
	rootStarted := false
	for i := 0; ; {
		for i < len(s.source) && jsonStructureWhitespace(s.source[i]) {
			i++
		}
		if i == len(s.source) {
			if !rootStarted || len(s.stack) != 0 {
				return s.problem(i, "json-syntax", "unexpected end of JSON input")
			}
			return nil
		}
		if err := s.consumeToken(); err != nil {
			return err
		}
		var err error
		if len(s.stack) == 0 {
			if rootStarted {
				return s.problem(i, "json-syntax", "unexpected text after JSON value")
			}
			rootStarted = true
			i, err = s.jsonValue(i)
		} else {
			i, err = s.jsonInContainer(i)
		}
		if err != nil || len(s.problems) != 0 {
			return err
		}
	}
}

func (s *structureScan) jsonInContainer(i int) (int, error) {
	frame := &s.stack[len(s.stack)-1]
	b := s.source[i]
	switch frame.jsonState {
	case jsonArrayFirst:
		if b == ']' {
			return i + 1, s.pop(i)
		}
		fallthrough
	case jsonArrayValue:
		frame.jsonState = jsonArrayAfterValue
		return s.jsonValue(i)
	case jsonArrayAfterValue:
		switch b {
		case ',':
			frame.jsonState = jsonArrayValue
			return i + 1, nil
		case ']':
			return i + 1, s.pop(i)
		}
		return i, s.problem(i, "json-syntax", "expected ',' or ']'")
	case jsonObjectFirst:
		if b == '}' {
			return i + 1, s.pop(i)
		}
		fallthrough
	case jsonObjectKey:
		if b != '"' {
			return i, s.problem(i, "json-syntax", "expected a quoted object key")
		}
		frame.jsonState = jsonObjectColon
		return s.jsonString(i)
	case jsonObjectColon:
		if b != ':' {
			return i, s.problem(i, "json-syntax", "expected ':' after object key")
		}
		frame.jsonState = jsonObjectValue
		return i + 1, nil
	case jsonObjectValue:
		frame.jsonState = jsonObjectAfterValue
		return s.jsonValue(i)
	case jsonObjectAfterValue:
		switch b {
		case ',':
			frame.jsonState = jsonObjectKey
			return i + 1, nil
		case '}':
			return i + 1, s.pop(i)
		}
		return i, s.problem(i, "json-syntax", "expected ',' or '}'")
	default:
		panic("invalid JSON structure scan state")
	}
}

func (s *structureScan) jsonValue(i int) (int, error) {
	var word string
	switch b := s.source[i]; b {
	case '{':
		return i + 1, s.push(i, '}', jsonObjectFirst)
	case '[':
		return i + 1, s.push(i, ']', jsonArrayFirst)
	case '"':
		return s.jsonString(i)
	case 't':
		word = "true"
	case 'f':
		word = "false"
	case 'n':
		word = "null"
	default:
		if b == '-' || jsonStructureDigit(b) {
			return s.jsonNumber(i)
		}
		return i, s.problem(i, "json-syntax", "expected a JSON value")
	}
	if !strings.HasPrefix(s.source[i:], word) {
		return i, s.problem(i, "json-syntax", "invalid JSON literal")
	}
	return i + len(word), nil
}

func (s *structureScan) jsonString(i int) (int, error) {
	source := s.source
	for i++; i < len(source); i++ {
		switch b := source[i]; {
		case b == '"':
			return i + 1, nil
		case b == '\\':
			i++
			if i == len(source) {
				return i, s.problem(i, "json-syntax", "unfinished string escape")
			}
			switch source[i] {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
			case 'u':
				// RFC 8259's grammar admits any four hex digits, even an
				// unpaired escaped surrogate. No decoding or rewriting here.
				for range 4 {
					i++
					if i == len(source) || !jsonStructureHex(source[i]) {
						return i, s.problem(i, "json-syntax", "expected four hex digits in Unicode escape")
					}
				}
			default:
				return i, s.problem(i, "json-syntax", "invalid string escape")
			}
		case b < 0x20:
			return i, s.problem(i, "json-syntax", "unescaped control character in string")
		}
	}
	return i, s.problem(i, "json-syntax", "unclosed JSON string")
}

func (s *structureScan) jsonNumber(i int) (int, error) {
	source := s.source
	if source[i] == '-' {
		i++
	}
	if i == len(source) || !jsonStructureDigit(source[i]) {
		return i, s.problem(i, "json-syntax", "expected an integer digit")
	}
	if source[i] == '0' {
		i++
		if i < len(source) && jsonStructureDigit(source[i]) {
			return i, s.problem(i, "json-syntax", "leading zero in JSON number")
		}
	} else {
		i = jsonStructureSkipDigits(source, i)
	}
	if i < len(source) && source[i] == '.' {
		start := i + 1
		i = jsonStructureSkipDigits(source, start)
		if i == start {
			return i, s.problem(i, "json-syntax", "expected a fraction digit")
		}
	}
	if i < len(source) && (source[i] == 'e' || source[i] == 'E') {
		i++
		if i < len(source) && (source[i] == '+' || source[i] == '-') {
			i++
		}
		start := i
		i = jsonStructureSkipDigits(source, i)
		if i == start {
			return i, s.problem(i, "json-syntax", "expected an exponent digit")
		}
	}
	return i, nil
}

func jsonStructureWhitespace(b byte) bool { return b == ' ' || b == '\t' || b == '\r' || b == '\n' }

func jsonStructureDigit(b byte) bool { return '0' <= b && b <= '9' }

func jsonStructureHex(b byte) bool {
	return jsonStructureDigit(b) || 'a' <= b && b <= 'f' || 'A' <= b && b <= 'F'
}

func jsonStructureSkipDigits(source string, i int) int {
	for i < len(source) && jsonStructureDigit(source[i]) {
		i++
	}
	return i
}
