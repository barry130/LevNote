package search

import (
	"crypto/sha256"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"levnote.local/native/core/text"
)

// ErrReplacementLimit means that the expanded replacement did not fit the
// caller's remaining byte budget. No output or document mutation is produced.
var ErrReplacementLimit = errors.New("expanded replacement exceeds byte budget")

// ErrReplacementWork is distinct from output capacity: even an empty expansion
// must pay for parsing, capture lookup and match validation.
var ErrReplacementWork = errors.New("replacement work budget exhausted")

// DefaultReplacementWork is a finite per-call ceiling when no ledger is supplied.
// Multi-field callers should pass the SAME remaining counter to compilation,
// binding, expansion and stitching. Units are deterministic admission charges,
// not elapsed time: one per operation/index/probe and per byte scanned or copied.
const DefaultReplacementWork = 64 << 20

func replacementWork(work *int) *int {
	if work == nil {
		remaining := DefaultReplacementWork
		return &remaining
	}
	return work
}
func spendReplacementWork(work *int, units int) error {
	if units < 0 || *work < units {
		*work = 0
		return ErrReplacementWork
	}
	*work -= units
	return nil
}

// FindStringBounded shares FindString's matching and UTF-16 coordinates, with a
// caller-owned candidate limit. Nonpositive limits are refused; larger limits
// are capped at MaxMatches before limit+1 arithmetic. WholeWord filtering follows
// candidate admission, so ErrTooManyMatches can accompany fewer than limit hits.
// Such a partial scan must never be presented as a complete replace-all plan.
// Callers must bound source/query bytes before calling this pure function.
func FindStringBounded(source string, opts Options, limit int) ([]Match, error) {
	if limit <= 0 {
		return nil, ErrTooManyMatches
	}
	if !utf8.ValidString(source) || !utf8.ValidString(opts.Query) {
		return nil, text.ErrUTF8
	}
	if opts.Query == "" {
		return nil, nil
	}
	re, err := compile(opts)
	if err != nil {
		return nil, err
	}
	return findCompiled(source, opts, re, min(limit, MaxMatches))
}

type replacementPart struct {
	literal string
	ref     int // -1 for a literal; otherwise an index into references
}
type replacementCapture struct {
	number     int
	candidates []int // duplicate names, in regexp order
}

// ReplacementTemplate parses the regexp replacement grammar once. It is
// immutable after compilation; bindings borrow one immutable field string.
type ReplacementTemplate struct {
	opts       Options
	captures   int
	parts      []replacementPart
	references []replacementCapture
}

// CompileReplacementTemplate preserves regexp.ExpandString's maximal names,
// numeric references (including its 1e8 cutoff), $$ and malformed references.
// Name parsing and candidate comparisons are charged BEFORE doing that work.
func CompileReplacementTemplate(opts Options, replacement string, work *int) (*ReplacementTemplate, error) {
	work = replacementWork(work)
	for _, cost := range []int{1, len(opts.Query), len(replacement), len(opts.Query)} {
		if err := spendReplacementWork(work, cost); err != nil {
			return nil, err
		}
	}
	if !utf8.ValidString(opts.Query) || !utf8.ValidString(replacement) {
		return nil, text.ErrUTF8
	}
	re, err := compile(opts)
	if err != nil {
		return nil, err
	}
	t := &ReplacementTemplate{opts: opts, captures: re.NumSubexp() + 1}
	literal := func(value string) {
		if value != "" {
			t.parts = append(t.parts, replacementPart{literal: value, ref: -1})
		}
	}
	if !opts.Regex {
		literal(replacement)
		return t, nil
	}
	names := re.SubexpNames()
	named := make(map[string]int)
	start := 0
	for i := 0; i < len(replacement); {
		if err := spendReplacementWork(work, 1); err != nil {
			return nil, err
		}
		if replacement[i] != '$' {
			i++
			continue
		}
		literal(replacement[start:i])
		i++
		if i < len(replacement) && replacement[i] == '$' {
			if err := spendReplacementWork(work, 1); err != nil {
				return nil, err
			}
			literal("$")
			i++
			start = i
			continue
		}
		name, number, consumed, err := replacementReference(replacement[i:], work)
		if err != nil {
			return nil, err
		}
		if consumed == 0 {
			literal("$") // leave the malformed suffix to the literal scanner
			start = i
			continue
		}
		i += consumed
		start = i
		ref := len(t.references)
		if number >= 0 {
			t.references = append(t.references, replacementCapture{number: number})
		} else {
			// Pay for hashing the name even on a cached lookup.
			if err := spendReplacementWork(work, len(name)); err != nil {
				return nil, err
			}
			if cached, ok := named[name]; ok {
				ref = cached
			} else {
				capture := replacementCapture{number: -1}
				for index, candidate := range names {
					for _, cost := range []int{1, len(name), len(candidate)} {
						if err := spendReplacementWork(work, cost); err != nil {
							return nil, err
						}
					}
					if candidate == name {
						capture.candidates = append(capture.candidates, index)
					}
				}
				named[name] = ref
				t.references = append(t.references, capture)
			}
		}
		t.parts = append(t.parts, replacementPart{ref: ref})
	}
	literal(replacement[start:])
	return t, nil
}

// replacementReference is regexp's extract grammar, with a charged rune scan and
// a separate charged numeric scan. Invalid braced references consume no suffix.
func replacementReference(value string, work *int) (name string, number, consumed int, err error) {
	if value == "" {
		return
	}
	brace := value[0] == '{'
	begin := 0
	if brace {
		if err = spendReplacementWork(work, 1); err != nil {
			return
		}
		begin = 1
	}
	end := begin
	for end < len(value) {
		_, width := utf8.DecodeRuneInString(value[end:])
		if err = spendReplacementWork(work, width); err != nil {
			return
		}
		r, _ := utf8.DecodeRuneInString(value[end:])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			break
		}
		end += width
	}
	if end == begin || brace && (end == len(value) || value[end] != '}') {
		return
	}
	name = value[begin:end]
	consumed = end
	if brace {
		consumed++
	}
	for i := 0; i < len(name); i++ {
		if err = spendReplacementWork(work, 1); err != nil {
			return
		}
		if name[i] < '0' || name[i] > '9' || number >= 1e8 {
			number = -1
			break
		}
		number = number*10 + int(name[i]-'0')
	}
	if len(name) > 1 && name[0] == '0' {
		number = -1
	}
	return
}

// BoundReplacement validates UTF-8, hashes, and maps byte boundaries to UTF-16
// exactly once per field, rather than rescanning the field for every match.
type BoundReplacement struct {
	template *ReplacementTemplate
	source   string
	hash     [32]byte
	units    []int // UTF-16 offset + 1; zero marks a non-boundary
}

func (t *ReplacementTemplate) Bind(source string, work *int) (*BoundReplacement, error) {
	work = replacementWork(work)
	if err := spendReplacementWork(work, 1); err != nil {
		return nil, err
	}
	// Validation, hashing/copying and the coordinate-table pass are all bounded.
	for i := 0; i < 4; i++ {
		if err := spendReplacementWork(work, len(source)); err != nil {
			return nil, err
		}
	}
	if !utf8.ValidString(source) {
		return nil, text.ErrUTF8
	}
	if t == nil {
		return nil, ErrStale
	}
	b := &BoundReplacement{template: t, source: source, hash: sha256.Sum256([]byte(source)), units: make([]int, len(source)+1)}
	units := 0
	for i, r := range source {
		b.units[i] = units + 1
		units++
		if r > 0xffff {
			units++
		}
	}
	b.units[len(source)] = units + 1
	return b, nil
}

// FieldExpansion carries byte positions verified against the bound original
// field. Descending hit order keeps these positions valid during stitching.
type FieldExpansion struct {
	Text               string
	StartByte, EndByte int
}

func (b *BoundReplacement) Expand(match Match, maxBytes int, work *int) (FieldExpansion, error) {
	work = replacementWork(work)
	fail := func(err error) (FieldExpansion, error) { return FieldExpansion{}, err }
	if maxBytes < 0 {
		return fail(ErrReplacementLimit)
	}
	if err := spendReplacementWork(work, 1); err != nil {
		return fail(err)
	}
	if b == nil || b.template == nil {
		return fail(ErrStale)
	}
	t := b.template
	for _, cost := range []int{len(t.opts.Query), len(match.indices), len(match.Value)} {
		if err := spendReplacementWork(work, cost); err != nil {
			return fail(err)
		}
	}
	if t.opts.Query == "" || t.opts != match.options || b.hash != match.hash || len(match.indices) != 2*t.captures {
		return fail(ErrStale)
	}
	for i := 0; i < len(match.indices); i += 2 {
		a, z := match.indices[i], match.indices[i+1]
		if a == -1 && z == -1 && i > 0 {
			continue
		}
		if a < 0 || z < a || z > len(b.source) || b.units[a] == 0 || b.units[z] == 0 {
			return fail(ErrStale)
		}
	}
	a, z := match.indices[0], match.indices[1]
	if match.Start != b.units[a]-1 || match.End != b.units[z]-1 || match.Value != b.source[a:z] {
		return fail(ErrStale)
	}
	if err := spendReplacementWork(work, len(t.references)); err != nil {
		return fail(err)
	}
	resolved := make([]int, len(t.references))
	for i, ref := range t.references {
		resolved[i] = ref.number
		if ref.number < 0 {
			for _, candidate := range ref.candidates {
				if err := spendReplacementWork(work, 1); err != nil {
					return fail(err)
				}
				if match.indices[2*candidate] >= 0 {
					resolved[i] = candidate
					break
				}
			}
		}
	}
	partValue := func(part replacementPart) string {
		if part.ref < 0 {
			return part.literal
		}
		index := resolved[part.ref]
		if index >= 0 && index < t.captures && match.indices[2*index] >= 0 {
			return b.source[match.indices[2*index]:match.indices[2*index+1]]
		}
		return ""
	}
	size := 0
	for _, part := range t.parts {
		if err := spendReplacementWork(work, 1); err != nil {
			return fail(err)
		}
		n := len(partValue(part))
		if n > maxBytes-size {
			return fail(ErrReplacementLimit)
		}
		size += n
	}
	for _, cost := range []int{len(t.parts), size} {
		if err := spendReplacementWork(work, cost); err != nil {
			return fail(err)
		}
	}
	var out strings.Builder
	out.Grow(size)
	for _, part := range t.parts {
		out.WriteString(partValue(part))
	}
	return FieldExpansion{out.String(), a, z}, nil
}

// StitchReplacement charges the entire new candidate before allocating/copying.
// source and expansion must be validated UTF-8; byte-boundary checks fail closed.
// A failed admission returns no partial output and never mutates either input.
func StitchReplacement(source string, expansion FieldExpansion, maxBytes int, work *int) (string, error) {
	work = replacementWork(work)
	if err := spendReplacementWork(work, 1); err != nil {
		return "", err
	}
	a, z := expansion.StartByte, expansion.EndByte
	if a < 0 || z < a || z > len(source) || a < len(source) && !utf8.RuneStart(source[a]) || z < len(source) && !utf8.RuneStart(source[z]) {
		return "", ErrStale
	}
	base := len(source) - (z - a)
	if maxBytes < base || len(expansion.Text) > maxBytes-base {
		return "", ErrReplacementLimit
	}
	if err := spendReplacementWork(work, base+len(expansion.Text)); err != nil {
		return "", err
	}
	return source[:a] + expansion.Text + source[z:], nil
}

// ExpandReplacement keeps its original signature and regexp semantics. All
// stages share one finite default ledger; repeated field operations should use
// CompileReplacementTemplate, Bind, Expand and StitchReplacement explicitly.
func ExpandReplacement(source string, match Match, opts Options, replacement string, maxBytes int) (string, error) {
	if maxBytes < 0 {
		return "", ErrReplacementLimit
	}
	work := DefaultReplacementWork
	t, err := CompileReplacementTemplate(opts, replacement, &work)
	if err != nil {
		return "", err
	}
	b, err := t.Bind(source, &work)
	if err != nil {
		return "", err
	}
	expansion, err := b.Expand(match, maxBytes, &work)
	return expansion.Text, err
}
