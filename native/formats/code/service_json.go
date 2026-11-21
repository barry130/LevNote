package code

import (
	"strconv"
	"strings"
)

type serviceJSONWalk struct {
	run      *serviceRun
	scan     structureScan
	i        int
	path     []string
	pathSize int
	symbols  []ServiceSymbol
	vocab    *serviceVocabulary
}

func (s *serviceRun) jsonResult() ServiceResult {
	walk := serviceJSONWalk{run: s, scan: structureScan{source: s.request.Source}, pathSize: 1}
	if s.request.Operation == ServiceCompletion {
		walk.vocab = &serviceVocabulary{run: s, seen: make(map[string]struct{})}
	} else if !s.loadPoints() {
		return s.failure
	}
	if !walk.value() {
		return s.failure
	}
	if walk.vocab != nil {
		return walk.vocab.result()
	}
	return ServiceResult{Status: ServiceComplete, Symbols: walk.symbols}
}

func (w *serviceJSONWalk) space() bool {
	source := w.scan.source
	for w.i < len(source) && jsonStructureWhitespace(source[w.i]) {
		w.i++
		if w.i%servicePointStride == 0 && w.run.stop() {
			return false
		}
	}
	return !w.run.stop()
}

// value projects the key paths of the validated T1 tree. It relies on T1's
// complete RFC 8259 result for delimiter pairing, punctuation and numbers.
// The recursive depth is therefore bounded by T1's delimiter limit.
func (w *serviceJSONWalk) value() bool {
	if !w.space() {
		return false
	}
	source := w.scan.source
	switch source[w.i] {
	case '{':
		return w.object()
	case '[':
		return w.array()
	case '"':
		w.i, _ = w.scan.jsonString(w.i)
	default:
		for w.i < len(source) && !jsonStructureWhitespace(source[w.i]) && !strings.ContainsRune(",}]", rune(source[w.i])) {
			w.i++
			if w.i%servicePointStride == 0 && w.run.stop() {
				return false
			}
		}
	}
	return !w.run.stop()
}

func (w *serviceJSONWalk) push(part string) {
	w.path = append(w.path, part)
	w.pathSize += len(part) + 2
}

func (w *serviceJSONWalk) pop() {
	last := len(w.path) - 1
	w.pathSize -= len(w.path[last]) + 2
	w.path = w.path[:last]
}

func (w *serviceJSONWalk) key(start int, raw string) bool {
	if w.vocab != nil {
		return w.vocab.add(raw, "json-key")
	}
	if !w.run.count(len(w.symbols), w.run.limits.Symbols, serviceSymbolsResource) ||
		!w.run.keepText(raw) || !w.run.keepBytes(w.pathSize) {
		return false
	}
	var path strings.Builder
	path.Grow(w.pathSize)
	path.WriteByte('$')
	for _, part := range w.path {
		path.WriteByte('[')
		path.WriteString(part)
		path.WriteByte(']')
	}
	w.symbols = append(w.symbols, ServiceSymbol{
		Name: strings.Clone(raw), Kind: "json-key", Path: path.String(), Span: w.run.span(start, w.i),
	})
	return true
}

func (w *serviceJSONWalk) object() bool {
	source := w.scan.source
	w.i++
	for {
		if !w.space() {
			return false
		}
		if source[w.i] == '}' {
			w.i++
			return true
		}
		start := w.i
		w.i, _ = w.scan.jsonString(w.i)
		if w.run.stop() {
			return false
		}
		raw := source[start:w.i]
		if w.vocab == nil {
			w.push(raw)
		}
		if !w.key(start, raw) || !w.space() {
			return false
		}
		w.i++ // colon
		if !w.value() {
			return false
		}
		if w.vocab == nil {
			w.pop()
		}
		if !w.space() {
			return false
		}
		if source[w.i] == ',' {
			w.i++
		}
	}
}

func (w *serviceJSONWalk) array() bool {
	source := w.scan.source
	w.i++
	for index := 0; ; index++ {
		if !w.space() {
			return false
		}
		if source[w.i] == ']' {
			w.i++
			return true
		}
		if w.vocab == nil {
			w.push(strconv.Itoa(index))
		}
		if !w.value() {
			return false
		}
		if w.vocab == nil {
			w.pop()
		}
		if !w.space() {
			return false
		}
		if source[w.i] == ',' {
			w.i++
		}
	}
}
