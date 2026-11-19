package session

import (
	"errors"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
)

var errRecoveryLimit = errors.New("session exceeds recovery storage limit before JSON marshalling; existing recovery file kept")

// jsonStringCounter implements both io.Writer and io.StringWriter. Buffer's
// immutable piece strings reach WriteString directly, without converting every
// piece to []byte. Pending UTF-8 is at most three bytes, never the input string.
// Counts include encoding/json's default HTML escaping. Once the budget is
// exceeded the counter stops; there is no reason to scan the remaining input.
type jsonStringCounter struct {
	count   int
	pending [utf8.UTFMax]byte
	n       int
}

func (c *jsonStringCounter) add(r rune, width int) {
	if r == utf8.RuneError && width == 1 {
		c.count += 3 // Go 1.27 JSON emits one UTF-8 replacement per invalid byte.
	} else {
		c.count += jsonEncodedRuneBytes(r)
	}
}

func (c *jsonStringCounter) byte(b byte) {
	c.pending[c.n] = b
	c.n++
	for c.n > 0 && utf8.FullRune(c.pending[:c.n]) {
		r, width := utf8.DecodeRune(c.pending[:c.n])
		c.add(r, width)
		copy(c.pending[:], c.pending[width:c.n])
		c.n -= width
	}
}

func (c *jsonStringCounter) Write(data []byte) (int, error) {
	i := 0
	for i < len(data) {
		if c.n > 0 {
			c.byte(data[i])
			i++
		} else if !utf8.FullRune(data[i:]) {
			c.n = copy(c.pending[:], data[i:])
			i = len(data)
		} else {
			r, width := utf8.DecodeRune(data[i:])
			c.add(r, width)
			i += width
		}
		if c.count+2 > MaxBytes {
			return i, errRecoveryLimit
		}
	}
	return len(data), nil
}

func (c *jsonStringCounter) WriteString(value string) (int, error) {
	i := 0
	for i < len(value) {
		if c.n > 0 {
			c.byte(value[i])
			i++
		} else if value[i] < utf8.RuneSelf {
			c.count += jsonEncodedRuneBytes(rune(value[i]))
			i++
		} else if !utf8.FullRuneInString(value[i:]) {
			c.n = copy(c.pending[:], value[i:])
			i = len(value)
		} else {
			r, width := utf8.DecodeRuneInString(value[i:])
			c.add(r, width)
			i += width
		}
		if c.count+2 > MaxBytes {
			return i, errRecoveryLimit
		}
	}
	return len(value), nil
}

func (c *jsonStringCounter) Finish() int {
	// A truncated UTF-8 sequence encodes as one replacement per invalid byte.
	c.count += c.n * 3
	c.n = 0
	return c.count + 2
}

func jsonEncodedRuneBytes(r rune) int {
	switch {
	case r == '"' || r == '\\' || r == '\b' || r == '\f' || r == '\n' || r == '\r' || r == '\t':
		return 2
	case r < 0x20 || r == '<' || r == '>' || r == '&' || r == 0x2028 || r == 0x2029:
		return 6
	default:
		return utf8.RuneLen(r)
	}
}

func draftEncodedLength(buffer *text.Buffer) (int, error) {
	if buffer == nil {
		return 0, errors.New("session text buffer is nil")
	}
	counter := &jsonStringCounter{}
	if _, err := buffer.WriteTo(counter); err != nil {
		return 0, err
	}
	return counter.Finish(), nil
}

// draftEncodedBytes returns an exact length up to the session budget, then a
// value above the budget. No input-length allocation occurs, even on rejection.
func draftEncodedBytes(value string) int {
	if len(value) > MaxBytes {
		return MaxBytes + 1
	}
	var counter jsonStringCounter
	_, _ = counter.WriteString(value)
	return counter.Finish()
}

func optionalStringBytes(name, value string) int {
	if value == "" {
		return 0
	}
	return len(name) + 4 + draftEncodedBytes(value) // ,"name":value
}

func boolBytes(value bool) int {
	if value {
		return 4
	}
	return 5
}

func intBytes(value int) int {
	var scratch [24]byte
	return len(strconv.AppendInt(scratch[:0], int64(value), 10))
}

// Match encoding/json's float32 encoder, including its ES6 exponent cutoffs
// and removal of the redundant exponent zero. The scratch space is bounded.
func floatBytes(value float32) int {
	f := float64(value)
	fmt := byte('f')
	abs := math.Abs(f)
	if abs != 0 && (float32(abs) < 1e-6 || float32(abs) >= 1e21) {
		fmt = 'e'
	}
	var scratch [32]byte
	encoded := strconv.AppendFloat(scratch[:0], f, fmt, -1, 32)
	n := len(encoded)
	if fmt == 'e' && n >= 4 && encoded[n-4] == 'e' && encoded[n-3] == '-' && encoded[n-2] == '0' {
		n--
	}
	return n
}

// draftSize is -1 when omitted and includes the JSON quotes otherwise. Field
// punctuation/order matches the public structs; regressions compare to Marshal.
func tabEncodedLength(tab Tab, draftSize int) int {
	if tab.binary != nil {
		return binaryTabEncodedLength(*tab.binary)
	}
	total := len(`{"bom":,"selection":{"Anchor":,"Head":}}`) + boolBytes(tab.BOM) + intBytes(tab.Selection.Anchor) + intBytes(tab.Selection.Head)
	total += optionalStringBytes("path", tab.Path) + optionalStringBytes("originalPath", tab.OriginalPath) + optionalStringBytes("title", tab.Title)
	total += optionalStringBytes("encoding", string(tab.Encoding)) + optionalStringBytes("pluginId", tab.PluginID) + optionalStringBytes("view", tab.View)
	if draftSize >= 0 {
		total += len(`,"draft":`) + draftSize
	}
	if tab.Scroll != 0 {
		total += len(`,"scroll":`) + floatBytes(tab.Scroll)
	}
	return total
}

func preferencesEncodedLength(p Preferences) int {
	total := len(`{"wordWrap":}`) + boolBytes(p.WordWrap) + optionalStringBytes("theme", p.Theme)
	if p.FontSize != 0 {
		total += len(`,"fontSize":`) + floatBytes(p.FontSize)
	}
	if p.SidebarWidth != 0 {
		total += len(`,"sidebarWidth":`) + floatBytes(p.SidebarWidth)
	}
	if p.LineNumbers != nil {
		total += len(`,"lineNumbers":`) + boolBytes(*p.LineNumbers)
	}
	return total
}

func snapshotEnvelopeLength(snapshot Snapshot, tabCount, tabsBytes int, nilTabs bool) (int, error) {
	if err := validMetadataString(snapshot.Root); err != nil {
		return 0, err
	}
	if snapshot.Root != "" && (!filepath.IsAbs(snapshot.Root) || strings.IndexByte(snapshot.Root, 0) >= 0) {
		return 0, errors.New("session root must be absolute and contain no NUL")
	}
	if snapshot.Active < 0 || (tabCount == 0 && snapshot.Active != 0) || (tabCount > 0 && snapshot.Active >= tabCount) {
		return 0, errors.New("active session tab is out of range")
	}
	if err := validatePreferences(snapshot.Preferences); err != nil {
		return 0, err
	}
	total := len(`{"version":,"active":,"tabs":,"preferences":}`) + intBytes(snapshot.Version) + intBytes(snapshot.Active) + optionalStringBytes("root", snapshot.Root) + preferencesEncodedLength(snapshot.Preferences)
	if nilTabs {
		total += 4 // null, preserved for compatibility with direct Save(Snapshot{})
	} else {
		total += 2 + max(0, tabCount-1) + tabsBytes
	}
	if total > MaxBytes {
		return 0, errRecoveryLimit
	}
	return total, nil
}

func preflightSnapshot(snapshot Snapshot) (int, error) {
	if len(snapshot.Tabs) > MaxTabs {
		return 0, errors.New("session has more than 100 tabs; existing recovery file kept")
	}
	// Root/preferences are checked before walking any draft strings.
	if _, err := snapshotEnvelopeLength(snapshot, len(snapshot.Tabs), 0, snapshot.Tabs == nil); err != nil {
		return 0, err
	}
	total, raw := 0, 0
	for _, tab := range snapshot.Tabs {
		if err := validateTab(tab); err != nil {
			return 0, err
		}
		encoded := -1
		if tab.binary == nil && tab.Draft != nil {
			raw += len(*tab.Draft)
			if raw > MaxDraftAggregateBytes {
				return 0, errRecoveryLimit
			}
			encoded = draftEncodedBytes(*tab.Draft)
		}
		total += tabEncodedLength(tab, encoded)
		if total > MaxBytes {
			return 0, errRecoveryLimit
		}
	}
	return snapshotEnvelopeLength(snapshot, len(snapshot.Tabs), total, snapshot.Tabs == nil)
}

func validMetadataString(value string) error {
	if len(value) > MaxBytes {
		return errRecoveryLimit
	}
	if !utf8.ValidString(value) {
		return errors.New("session metadata is not valid UTF-8")
	}
	return nil
}

func finite(value float32) bool {
	return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}

func validatePreferences(p Preferences) error {
	if err := validMetadataString(p.Theme); err != nil {
		return err
	}
	if p.Theme != "" && p.Theme != "light" && p.Theme != "dark" && p.Theme != "system" {
		return errors.New("unknown session theme")
	}
	if !finite(p.FontSize) || (p.FontSize != 0 && (p.FontSize < 9 || p.FontSize > 40)) {
		return errors.New("invalid session font size")
	}
	if !finite(p.SidebarWidth) || (p.SidebarWidth != 0 && (p.SidebarWidth < 160 || p.SidebarWidth > 640)) {
		return errors.New("invalid session sidebar width")
	}
	return nil
}

func validateTabMetadata(tab Tab, hasDraft bool) error {
	for _, value := range []string{tab.Path, tab.OriginalPath, tab.Title, string(tab.Encoding), tab.PluginID, tab.View} {
		if err := validMetadataString(value); err != nil {
			return err
		}
	}
	if tab.Path != "" && (!filepath.IsAbs(tab.Path) || strings.IndexByte(tab.Path, 0) >= 0) {
		return errors.New("tab path must be absolute and contain no NUL")
	}
	if tab.OriginalPath != "" && (!filepath.IsAbs(tab.OriginalPath) || strings.IndexByte(tab.OriginalPath, 0) >= 0) {
		return errors.New("original path must be absolute and contain no NUL")
	}
	if !hasDraft && tab.Path == "" {
		return errors.New("tab has neither a file nor a draft")
	}
	if tab.Encoding != "" && tab.Encoding != document.UTF8 && tab.Encoding != document.UTF16LE && tab.Encoding != document.UTF16BE && tab.Encoding != document.GBK && tab.Encoding != document.GB18030 {
		return errors.New("unknown draft encoding")
	}
	if (tab.Encoding == document.GBK || tab.Encoding == document.GB18030) && tab.BOM {
		return errors.New("legacy encoding draft cannot contain BOM")
	}
	if (tab.Encoding == document.UTF16LE || tab.Encoding == document.UTF16BE) && !tab.BOM {
		return errors.New("UTF-16 draft lacks BOM")
	}
	if tab.Selection.Anchor < 0 || tab.Selection.Head < 0 {
		return errors.New("negative selection offset")
	}
	if !finite(tab.Scroll) || tab.Scroll < 0 {
		return errors.New("invalid scroll position")
	}
	return nil
}

func validateTab(tab Tab) error {
	if tab.binary != nil {
		return validateBinaryTabContainer(tab)
	}
	if err := validateTabMetadata(tab, tab.Draft != nil); err != nil {
		return err
	}
	if tab.Draft != nil && (len(*tab.Draft) > MaxDraftBytes || !utf8.ValidString(*tab.Draft)) {
		return errors.New("draft is invalid or exceeds 4 MiB")
	}
	return nil
}
