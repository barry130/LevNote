package editor

import (
	"strings"
	"unicode/utf8"
)

// OutlineLabel copies only the display prefix, so a small cached label cannot
// retain an entire Markdown source or graph topic through a substring.
func OutlineLabel(value string) (string, bool) {
	if len(value) <= MaxOutlineLabelBytes {
		return strings.Clone(value), false
	}
	end := MaxOutlineLabelBytes - len("…")
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end] + "…", true
}
