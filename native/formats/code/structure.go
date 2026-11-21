package code

import (
	"errors"
	"fmt"
)

// StructureProfile is a syntax choice, independent of Language and its
// highlighting aliases. In particular, JSONC is not StructureJSON.
type StructureProfile string

const (
	StructureGo   StructureProfile = "go"
	StructureJSON StructureProfile = "json"
)

// StructureLimits bounds a single analysis. Each field must be positive;
// fields above their hard limits are clamped individually.
//
// Lines counts one plus the number of LF bytes, including a final empty line.
// Tokens counts Go lexical tokens (including comments and implicit semicolons)
// or JSON lexemes and punctuation, excluding whitespace and EOF. Depth counts
// simultaneously open delimiters. Folds and Diagnostics count produced entries.
type StructureLimits struct {
	Bytes, Lines, Tokens, Depth, Folds, Diagnostics int
}

// DefaultStructureLimits returns the hard ceilings. Callers with a lazy source
// snapshot should check its UTF-8 byte length before materializing a string.
func DefaultStructureLimits() StructureLimits {
	return StructureLimits{
		Bytes: 2 << 20, Lines: 65_536, Tokens: 262_144,
		Depth: 256, Folds: 4_096, Diagnostics: 256,
	}
}

// StructureByteRange is an absolute, zero-based, half-open UTF-8 byte range.
type StructureByteRange struct{ Start, End int }

// StructureUTF16Range is an absolute, zero-based, half-open UTF-16 unit range.
type StructureUTF16Range struct{ Start, End int }

// StructureLineRange is a half-open range of zero-based physical lines.
type StructureLineRange struct{ Start, End int }

// StructureSpan identifies the original text and its zero-based starting line.
// CRLF occupies two byte/UTF-16 units but ends exactly one physical line.
type StructureSpan struct {
	Bytes StructureByteRange
	UTF16 StructureUTF16Range
	Line  int
}

// StructureFold preserves both delimiter lines. HiddenLines is nonempty and
// contains exactly the fully enclosed lines, not either delimiter line.
type StructureFold struct {
	Open, Close StructureSpan
	HiddenLines StructureLineRange
}

// StructureDiagnostic identifies a source rejection (not a budget rejection).
// A diagnostic at EOF or at the first invalid UTF-8 byte is zero-width.
type StructureDiagnostic struct {
	Code, Message string
	Span          StructureSpan
}

// StructureResult owns its slices. Folds are in source closing-delimiter order
// (nested children before parents) and are present only for a complete result.
type StructureResult struct {
	Folds       []StructureFold
	Diagnostics []StructureDiagnostic
	Complete    bool
	Unsupported bool
}

// StructureResource identifies the exhausted analysis budget.
type StructureResource string

const (
	StructureBytes       StructureResource = "bytes"
	StructureLines       StructureResource = "lines"
	StructureTokens      StructureResource = "tokens"
	StructureDepth       StructureResource = "depth"
	StructureFolds       StructureResource = "folds"
	StructureDiagnostics StructureResource = "diagnostics"
)

// StructureBudgetError reports the effective (possibly clamped) limit.
type StructureBudgetError struct {
	Resource StructureResource
	Limit    int
}

func (e *StructureBudgetError) Error() string {
	return fmt.Sprintf("structure analysis exceeds %s limit (%d)", e.Resource, e.Limit)
}

// ErrInvalidStructureLimits indicates that at least one limit is nonpositive.
var ErrInvalidStructureLimits = errors.New("invalid structure limits")

// ErrStructureValidationLimit indicates that the Go grammar validator reached
// its own recursion guard, distinct from the structural delimiter depth limit.
var ErrStructureValidationLimit = errors.New("Go syntax validator reached its nesting limit")

// AnalyzeStructure validates and scans source without changing it or accessing
// files. Go uses lexical, delimiter, and single-file go/parser validation, not
// type checking. JSON uses the RFC 8259 grammar. An unsupported profile returns
// Unsupported without scanning source. A source error returns diagnostics, while
// a limit error returns a zero result and a non-nil error. No partial folds are
// returned. Limits are validated before dispatch, including for unknown profiles.
func AnalyzeStructure(source string, profile StructureProfile, limits StructureLimits) (StructureResult, error) {
	limits, err := normalizeStructureLimits(limits)
	if err != nil {
		return StructureResult{}, err
	}
	if profile != StructureGo && profile != StructureJSON {
		return StructureResult{Unsupported: true}, nil
	}
	if len(source) > limits.Bytes {
		return StructureResult{}, &StructureBudgetError{StructureBytes, limits.Bytes}
	}
	if problem, err := checkStructureSource(source, limits.Lines); err != nil {
		return StructureResult{}, err
	} else if problem != nil {
		return StructureResult{Diagnostics: []StructureDiagnostic{*problem}}, nil
	}

	scan := structureScan{source: source, limits: limits}
	if profile == StructureGo {
		err = scan.scanGo()
	} else {
		err = scan.scanJSON()
	}
	if err != nil {
		return StructureResult{}, err
	}
	if len(scan.problems) > 0 {
		return StructureResult{Diagnostics: scan.diagnostics()}, nil
	}
	return StructureResult{Complete: true, Folds: scan.folds}, nil
}

func normalizeStructureLimits(limits StructureLimits) (StructureLimits, error) {
	maxima := DefaultStructureLimits()
	for _, field := range [...]struct {
		value *int
		max   int
		name  StructureResource
	}{
		{&limits.Bytes, maxima.Bytes, StructureBytes},
		{&limits.Lines, maxima.Lines, StructureLines},
		{&limits.Tokens, maxima.Tokens, StructureTokens},
		{&limits.Depth, maxima.Depth, StructureDepth},
		{&limits.Folds, maxima.Folds, StructureFolds},
		{&limits.Diagnostics, maxima.Diagnostics, StructureDiagnostics},
	} {
		if *field.value <= 0 {
			return StructureLimits{}, fmt.Errorf("%w: %s must be positive", ErrInvalidStructureLimits, field.name)
		}
		*field.value = min(*field.value, field.max)
	}
	return limits, nil
}
