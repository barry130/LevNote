package code

import (
	"cmp"
	"slices"
	"strings"
	"unicode/utf8"
)

type structurePoint struct {
	offset, units, line int
}

func (p *structurePoint) step(r rune, width int) {
	p.offset += width
	p.units++
	if r > 0xffff {
		p.units++
	}
	if r == '\n' {
		p.line++
	}
}

// advance requires a valid, forward rune boundary. Each pass has its own
// cursor; a span never rescans an arbitrary prefix to compute its coordinates.
func (p *structurePoint) advance(source string, to int) {
	for p.offset < to {
		r, width := utf8.DecodeRuneInString(source[p.offset:])
		p.step(r, width)
	}
}

func (p structurePoint) span(end structurePoint) StructureSpan {
	return StructureSpan{
		Bytes: StructureByteRange{p.offset, end.offset},
		UTF16: StructureUTF16Range{p.units, end.units},
		Line:  p.line,
	}
}

// Check bytes before entering this preflight. It allocates no line or rune
// index and never hands invalid UTF-8 to either language scanner.
func checkStructureSource(source string, lines int) (*StructureDiagnostic, error) {
	var p structurePoint
	for p.offset < len(source) {
		r, width := utf8.DecodeRuneInString(source[p.offset:])
		if r == utf8.RuneError && width == 1 {
			return &StructureDiagnostic{
				Code: "invalid-utf8", Message: "invalid UTF-8 byte",
				Span: p.span(p),
			}, nil
		}
		if r == '\n' && p.line+1 == lines {
			return nil, &StructureBudgetError{StructureLines, lines}
		}
		p.step(r, width)
	}
	return nil, nil
}

type structureFrame struct {
	open      StructureSpan
	close     byte
	jsonState structureJSONState // unused by Go
}

type structureProblem struct {
	offset        int
	code, message string
}

type structureScan struct {
	source   string
	limits   StructureLimits
	tokens   int
	point    structurePoint // only visits structural ASCII delimiters, in order
	stack    []structureFrame
	folds    []StructureFold
	problems []structureProblem
}

func (s *structureScan) consumeToken() error {
	if s.tokens == s.limits.Tokens {
		return &StructureBudgetError{StructureTokens, s.limits.Tokens}
	}
	s.tokens++
	return nil
}

func (s *structureScan) delimiter(offset int) StructureSpan {
	s.point.advance(s.source, offset)
	start := s.point
	s.point.step(rune(s.source[offset]), 1)
	return start.span(s.point)
}

func (s *structureScan) push(offset int, close byte, state structureJSONState) error {
	if len(s.stack) == s.limits.Depth {
		return &StructureBudgetError{StructureDepth, s.limits.Depth}
	}
	s.stack = append(s.stack, structureFrame{s.delimiter(offset), close, state})
	return nil
}

// pop is called only for a matching closer. Check candidate capacity before
// extending the output. Same- or adjacent-line pairs use no candidate budget.
func (s *structureScan) pop(offset int) error {
	frame := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	close := s.delimiter(offset)
	if frame.open.Line+1 < close.Line {
		if len(s.folds) == s.limits.Folds {
			return &StructureBudgetError{StructureFolds, s.limits.Folds}
		}
		s.folds = append(s.folds, StructureFold{
			Open: frame.open, Close: close,
			HiddenLines: StructureLineRange{frame.open.Line + 1, close.Line},
		})
	}
	return nil
}

func (s *structureScan) problem(offset int, code, message string) error {
	if len(s.problems) == s.limits.Diagnostics {
		return &StructureBudgetError{StructureDiagnostics, s.limits.Diagnostics}
	}
	// The Go frontend can include literal or directive text in errors. Bound
	// the retained message and don't retain its backing string.
	const messageLimit = 160
	if len(message) > messageLimit {
		end := messageLimit - len("…")
		for !utf8.RuneStart(message[end]) {
			end--
		}
		message = message[:end] + "…"
	}
	offset = min(max(offset, 0), len(s.source))
	for offset < len(s.source) && !utf8.RuneStart(s.source[offset]) {
		offset--
	}
	s.problems = append(s.problems, structureProblem{offset, code, strings.Clone(message)})
	return nil
}

func (s *structureScan) diagnostics() []StructureDiagnostic {
	// The Go scanner can report errors out of offset order within a token.
	// Sort at most 256 entries, then translate all offsets in one forward pass.
	slices.SortStableFunc(s.problems, func(a, b structureProblem) int {
		return cmp.Compare(a.offset, b.offset)
	})
	result := make([]StructureDiagnostic, 0, len(s.problems))
	var point structurePoint
	for _, problem := range s.problems {
		point.advance(s.source, problem.offset)
		end := point
		if end.offset < len(s.source) {
			r, size := utf8.DecodeRuneInString(s.source[end.offset:])
			end.step(r, size)
		}
		result = append(result, StructureDiagnostic{problem.code, problem.message, point.span(end)})
	}
	return result
}
