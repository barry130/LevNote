// Package search supplies format-independent matching and transactional edits.
// Match offsets are UTF-16, exactly like the native text input boundary.
package search

import (
	"crypto/sha256"
	"errors"
	"regexp"
	"unicode"
	"unicode/utf8"

	"levnote.local/native/core/text"
)

var ErrStale = errors.New("document changed after the search; search again")
var ErrTooManyMatches = errors.New("search result limit reached; narrow the query")

const MaxMatches = 200000

type Options struct {
	Query                           string
	Regex, CaseSensitive, WholeWord bool
}
type Match struct {
	Start, End int
	Value      string
	// Match identity is tied to the exact snapshot, not a stale document offset.
	hash    [32]byte
	indices []int
	options Options
}

func compile(opts Options) (*regexp.Regexp, error) {
	query := opts.Query
	if !opts.Regex {
		query = regexp.QuoteMeta(query)
	}
	if !opts.CaseSensitive {
		query = "(?i)" + query
	}
	return regexp.Compile(query)
}

func Find(snapshot text.Snapshot, opts Options) ([]Match, error) {
	return FindString(snapshot.String(), opts)
}
func FindString(source string, opts Options) ([]Match, error) {
	if opts.Query == "" {
		return nil, nil
	}
	re, err := compile(opts)
	if err != nil {
		return nil, err
	}
	return findCompiled(source, opts, re, MaxMatches)
}
func findCompiled(source string, opts Options, re *regexp.Regexp, limit int) ([]Match, error) {
	indices := re.FindAllStringSubmatchIndex(source, limit+1)
	hash := sha256.Sum256([]byte(source))
	matches := make([]Match, 0, len(indices))
	lastByte, lastUnits := 0, 0
	for _, pair := range indices {
		if opts.WholeWord && !whole(source, pair[0], pair[1]) {
			continue
		}
		if len(matches) == limit {
			return matches, ErrTooManyMatches
		}
		lastUnits += text.UTF16Len(source[lastByte:pair[0]])
		start := lastUnits
		lastUnits += text.UTF16Len(source[pair[0]:pair[1]])
		lastByte = pair[1]
		matches = append(matches, Match{start, lastUnits, source[pair[0]:pair[1]], hash, pair, opts})
	}
	// Regex scanning is bounded before filtering. Disclose unvisited candidates
	// even when whole-word filtering discarded some of the initial matches.
	if len(indices) > limit {
		return matches, ErrTooManyMatches
	}
	return matches, nil
}
func word(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsMark(r) || r == '_'
}
func whole(source string, start, end int) bool {
	if start > 0 {
		r, _ := utf8.DecodeLastRuneInString(source[:start])
		if word(r) {
			return false
		}
	}
	if end < len(source) {
		r, _ := utf8.DecodeRuneInString(source[end:])
		if word(r) {
			return false
		}
	}
	return true
}

func Replace(buffer *text.Buffer, match Match, opts Options, replacement string) error {
	source := buffer.String()
	if sha256.Sum256([]byte(source)) != match.hash || opts != match.options {
		return ErrStale
	}
	re, err := compile(opts)
	if err != nil {
		return err
	}
	next := replacement
	if opts.Regex {
		next = string(re.ExpandString(nil, replacement, source, match.indices))
	}
	return buffer.ReplaceUTF16(match.Start, match.End, next)
}
func ReplaceAll(buffer *text.Buffer, opts Options, replacement string) (int, error) {
	source := buffer.String()
	matches, err := FindString(source, opts)
	if err != nil {
		return 0, err
	}
	if len(matches) == 0 {
		return 0, nil
	}
	re, err := compile(opts)
	if err != nil {
		return 0, err
	}
	if buffer.InGroup() {
		return 0, text.ErrGroup
	}
	// Preflight the actual reverse application order, including intermediate
	// sizes. A later shrink does not license an earlier over-limit expansion.
	// Literal replacements share one string. Regex plans retain at most
	// MaxMatches strings whose accepted total bytes cannot exceed the ceiling;
	// the current ExpandString candidate may already allocate before admission.
	var expanded []string
	if opts.Regex {
		expanded = make([]string, len(matches))
	}
	size := buffer.LenBytes()
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		next := replacement
		if opts.Regex {
			next = string(re.ExpandString(nil, replacement, source, m.indices))
		}
		if !utf8.ValidString(next) {
			return 0, text.ErrUTF8
		}
		size, err = buffer.CheckReplaceSize(size, m.indices[1]-m.indices[0], len(next))
		if err != nil {
			return 0, err
		}
		if opts.Regex {
			expanded[i] = next
		}
	}
	if err = buffer.BeginGroup(); err != nil {
		return 0, err
	}
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		next := replacement
		if opts.Regex {
			next = expanded[i]
		}
		if err = buffer.ReplaceUTF16(m.Start, m.End, next); err != nil {
			buffer.CancelGroup()
			return 0, err
		}
	}
	buffer.EndGroup()
	return len(matches), nil
}
