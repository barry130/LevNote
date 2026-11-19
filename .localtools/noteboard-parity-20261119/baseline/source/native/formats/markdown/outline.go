package markdown

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxParseBytes exposes the existing parser guard for callers that must reject
// before materializing a complete text buffer. It does not enlarge the budget.
const MaxParseBytes = maxInputBytes

const maxAnchorBaseBytes = 256

// HeadingAnchors assigns deterministic, unique current-document link slugs.
// It is not a navigation identity across source edits. The per-base suffix
// cursor bounds repeated identical headings without quadratic collision scans.
type HeadingAnchors struct {
	used map[string]bool
	next map[string]int
}

func (a *HeadingAnchors) Next(label string) (anchor string, clipped bool) {
	var base strings.Builder
	base.Grow(min(len(label), maxAnchorBaseBytes))
	separator := false
	for offset, char := range label {
		if unicode.IsSpace(char) || char == '-' {
			separator = base.Len() > 0
			continue
		}
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) && !unicode.IsMark(char) {
			continue
		}
		char = unicode.ToLower(char)
		needed := utf8.RuneLen(char)
		if separator {
			needed++
		}
		if base.Len()+needed > maxAnchorBaseBytes {
			clipped = offset < len(label)
			break
		}
		if separator {
			base.WriteByte('-')
		}
		separator = false
		base.WriteRune(char)
	}
	anchor = base.String()
	if anchor == "" {
		anchor = "section"
	}
	if a.used == nil {
		a.used, a.next = make(map[string]bool), make(map[string]int)
	}
	if !a.used[anchor] {
		a.used[anchor] = true
		return anchor, clipped
	}
	stem := anchor
	n := max(2, a.next[stem])
	for {
		anchor = stem + "-" + strconv.Itoa(n)
		n++
		if !a.used[anchor] {
			a.used[anchor], a.next[stem] = true, n
			return anchor, clipped
		}
	}
}

type Heading struct {
	Range    SourceRange
	Level    int
	Text     string
	AnchorID string
}

// Headings reuses only parser-recognized heading blocks. Text/Range refer to
// doc; callers retaining descriptors must copy a bounded Text prefix.
func Headings(doc Document, limit int) ([]Heading, bool) {
	var headings []Heading
	var anchors HeadingAnchors
	partial := doc.Partial
	for _, block := range doc.Blocks {
		if block.Kind != BlockHeading {
			continue
		}
		if len(headings) >= limit {
			return headings, true
		}
		anchor, clipped := anchors.Next(block.Text)
		partial = partial || clipped
		headings = append(headings, Heading{Range: block.Range, Level: clampHeading(block.Level), Text: block.Text, AnchorID: anchor})
	}
	return headings, partial
}
