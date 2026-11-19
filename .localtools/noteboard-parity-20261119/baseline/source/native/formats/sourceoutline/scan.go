// Package sourceoutline scans literal physical-line rules without a language
// runtime, regex engine, UI, or whole-source/whole-line materialization.
package sourceoutline

import (
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

const (
	MaxScanBytes   = 2 << 20
	MaxScanLines   = 65536
	MaxItems       = 2048
	MaxLabelBytes  = 512
	MaxRules       = 8
	MaxMarkerBytes = 16
)

type Rule struct {
	Prefix, Suffix string
	Level          int
}

type Config struct {
	PluginID string
	Schema   int
	Rules    []Rule
}

type Entry struct {
	Start, End int // half-open UTF16 physical-line range, including terminator
	Level      int
	Text       string // owned bounded display copy
}

type Result struct {
	Entries   []Entry
	Truncated bool
	Bytes     int
	Lines     int
}

var ErrUTF8 = errors.New("source outline input is not valid UTF-8")

// ValidateRules is defensive validation for non-JSON callers. The plugin
// loader separately owns strict manifest field and schema validation.
func ValidateRules(rules []Rule) error {
	if len(rules) > MaxRules {
		return fmt.Errorf("outline has more than %d rules", MaxRules)
	}
	for i, rule := range rules {
		if len(rule.Prefix) == 0 || len(rule.Prefix) > MaxMarkerBytes || len(rule.Suffix) > MaxMarkerBytes || rule.Level < 1 || rule.Level > 6 {
			return fmt.Errorf("outline rule %d has invalid marker length or level", i)
		}
		for _, marker := range []string{rule.Prefix, rule.Suffix} {
			if !utf8.ValidString(marker) {
				return fmt.Errorf("outline rule %d marker is not UTF-8", i)
			}
			for _, char := range marker {
				if char < 0x20 || char == 0x7f {
					return fmt.Errorf("outline rule %d marker contains a control byte", i)
				}
			}
		}
		for j := 0; j < i; j++ {
			if rules[j].Prefix == rule.Prefix && rules[j].Suffix == rule.Suffix {
				return fmt.Errorf("outline rule %d duplicates prefix and suffix", i)
			}
		}
	}
	return nil
}

// Scan interprets at most MaxScanBytes and consumes at most MaxScanBytes+1.
// Fixed 4096-byte read-ahead can extend beyond a line/item limit, but is never
// interpreted after that limit. No incomplete budget-cut line becomes an item.
func Scan(reader io.Reader, rules []Rule) (Result, error) {
	var result Result
	if reader == nil {
		return result, errors.New("source outline reader is nil")
	}
	if err := ValidateRules(rules); err != nil {
		return result, err
	}
	if len(rules) == 0 {
		return result, nil
	}
	input := boundedInput{reader: reader}
	line := newLine(rules)
	lineStart, units := 0, 0
	var runeBytes [utf8.UTFMax]byte
	runeSize := 0
	activeLine := false
	finishLine := func() {
		result.Lines++
		if entry, matched, clipped := line.finish(lineStart, units); matched {
			result.Entries = append(result.Entries, entry)
			result.Truncated = result.Truncated || clipped
		}
		line.reset()
		lineStart, activeLine = units, false
	}
	finishEOF := func() (Result, error) {
		if runeSize != 0 {
			return result, ErrUTF8
		}
		if activeLine {
			finishLine()
		}
		return result, nil
	}
	for {
		if result.Bytes == MaxScanBytes || result.Lines == MaxScanLines || len(result.Entries) == MaxItems {
			more, err := input.more()
			if err != nil {
				return result, err
			}
			if more {
				result.Truncated = true
				return result, nil
			}
			return finishEOF()
		}
		value, err := input.next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return finishEOF()
			}
			return result, err
		}
		result.Bytes++
		activeLine = true
		if runeSize == 0 && value < utf8.RuneSelf {
			units++
		} else {
			runeBytes[runeSize] = value
			runeSize++
			if utf8.FullRune(runeBytes[:runeSize]) {
				char, size := utf8.DecodeRune(runeBytes[:runeSize])
				if char == utf8.RuneError && size == 1 {
					return result, ErrUTF8
				}
				units++
				if char > 0xffff {
					units++
				}
				runeSize = 0
			}
		}
		if value == '\n' {
			finishLine() // pending terminal CR deliberately remains unconsumed
		} else {
			line.push(value)
		}
	}
}

type boundedInput struct {
	reader     io.Reader
	buffer     [4096]byte
	pos, count int
	read       int
	err        error
}

func (r *boundedInput) next() (byte, error) {
	if r.pos == r.count {
		if r.err != nil {
			return 0, r.err
		}
		limit := min(len(r.buffer), MaxScanBytes+1-r.read)
		if limit <= 0 {
			return 0, io.EOF
		}
		for attempts := 0; ; attempts++ {
			n, err := r.reader.Read(r.buffer[:limit])
			if n < 0 || n > limit {
				return 0, errors.New("source outline reader returned an invalid byte count")
			}
			r.pos, r.count, r.err = 0, n, err
			r.read += n
			if n > 0 {
				break
			}
			if err != nil {
				return 0, err
			}
			if attempts == 7 {
				return 0, io.ErrNoProgress
			}
		}
	}
	value := r.buffer[r.pos]
	r.pos++
	return value, nil
}

func (r *boundedInput) more() (bool, error) {
	if r.err != nil && !errors.Is(r.err, io.EOF) {
		return false, r.err
	}
	if r.pos < r.count {
		return true, nil // prefetched content already proves truncation
	}
	_, err := r.next()
	if errors.Is(err, io.EOF) {
		return false, nil
	}
	return err == nil, err
}
