package sourceoutline

import "unicode/utf8"

type lineState struct {
	rules     []Rule
	states    [MaxRules]ruleState
	started   bool
	pendingCR bool
}

func newLine(rules []Rule) lineState {
	line := lineState{rules: rules}
	line.reset()
	return line
}

func (line *lineState) reset() {
	line.started, line.pendingCR = false, false
	for i := range line.rules {
		state := &line.states[i]
		state.prefix, state.failed = 0, false
		body := &state.body
		body.bytes, body.first, body.spaces = 0, -1, 0
		body.headLen, body.tailCount, body.nonspaceCount = 0, 0, 0
	}
}

func horizontal(value byte) bool { return value == ' ' || value == '\t' }

func (line *lineState) push(value byte) {
	if line.pendingCR {
		line.consume('\r')
		line.pendingCR = false
	}
	if value == '\r' {
		line.pendingCR = true // discard exactly one terminal CR at LF/EOF
		return
	}
	line.consume(value)
}

func (line *lineState) consume(value byte) {
	if !line.started {
		if horizontal(value) {
			return
		}
		line.started = true
	}
	for i, rule := range line.rules {
		state := &line.states[i]
		if state.failed {
			continue
		}
		if state.prefix < len(rule.Prefix) {
			if value != rule.Prefix[state.prefix] {
				state.failed = true
			} else {
				state.prefix++
			}
			continue
		}
		state.body.push(value)
	}
}

func (line *lineState) finish(start, end int) (Entry, bool, bool) {
	for i, rule := range line.rules {
		state := &line.states[i]
		if state.failed || state.prefix != len(rule.Prefix) || !state.body.endsWith(rule.Suffix) {
			continue
		}
		label, clipped := state.body.label(len(rule.Suffix))
		if label == "" {
			return Entry{}, false, false // first marker match owns even empty line
		}
		return Entry{Start: start, End: end, Level: rule.Level, Text: label}, true, clipped
	}
	return Entry{}, false, false
}

type ruleState struct {
	prefix int
	failed bool
	body   bodyState
}

// The last17 non-horizontal byte positions locate the final label byte even
// after removing a16-byte suffix preceded by arbitrarily much whitespace.
// Ring tails and a512-byte prefix bound memory independently of line length.
type bodyState struct {
	bytes, first, spaces int
	head                 [MaxLabelBytes]byte
	headLen              int
	tail                 [MaxMarkerBytes]byte
	tailCount            int
	nonspace             [MaxMarkerBytes + 1]int
	nonspaceCount        int
	pendingSpaces        [MaxMarkerBytes]byte
}

func (body *bodyState) push(value byte) {
	pos := body.bytes
	body.bytes++
	space := horizontal(value)
	if !space && body.first < 0 {
		body.first = pos
	}
	if body.first >= 0 && body.headLen < len(body.head) {
		body.head[body.headLen] = value
		body.headLen++
	}
	if space {
		body.pendingSpaces[body.spaces%len(body.pendingSpaces)] = value
		body.spaces++
		return
	}
	// Preserve the exact space/tab run's ending in tail as it arrives, but
	// defer committing a trailing run until a nonspace byte follows.
	body.flushSpaces()
	body.tail[body.tailCount%len(body.tail)] = value
	body.tailCount++
	body.nonspace[body.nonspaceCount%len(body.nonspace)] = pos + 1
	body.nonspaceCount++
}

func (body *bodyState) flushSpaces() {
	// The actual horizontal bytes are also present in the head only for short
	// labels. A separate pending tail preserves tabs for literal suffixes.
	for i := min(body.spaces, len(body.pendingSpaces)) - 1; i >= 0; i-- {
		index := (body.spaces - 1 - i) % len(body.pendingSpaces)
		body.tail[body.tailCount%len(body.tail)] = body.pendingSpaces[index]
		body.tailCount++
	}
	body.spaces = 0
}

func (body *bodyState) endsWith(suffix string) bool {
	if len(suffix) > body.tailCount {
		return false
	}
	for i := range len(suffix) {
		if body.tail[(body.tailCount-len(suffix)+i)%len(body.tail)] != suffix[i] {
			return false
		}
	}
	return true
}

func (body *bodyState) label(suffixBytes int) (string, bool) {
	if body.first < 0 || body.nonspaceCount == 0 {
		return "", false
	}
	trimmedEnd := body.nonspace[(body.nonspaceCount-1)%len(body.nonspace)]
	end := 0
	for i := 1; i <= min(body.nonspaceCount, len(body.nonspace)); i++ {
		pos := body.nonspace[(body.nonspaceCount-i)%len(body.nonspace)]
		if pos <= trimmedEnd-suffixBytes {
			end = pos
			break
		}
	}
	length := end - body.first
	if length <= 0 {
		return "", false
	}
	if length <= len(body.head) {
		return string(body.head[:length]), false
	}
	limit := MaxLabelBytes - len("…")
	for limit > 0 && !utf8.RuneStart(body.head[limit]) {
		limit--
	}
	return string(body.head[:limit]) + "…", true
}
