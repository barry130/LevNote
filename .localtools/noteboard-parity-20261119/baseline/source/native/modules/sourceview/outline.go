// Package sourceview adapts a bounded literal-line outline to the editor
// contract. It creates no rich view and imports no plugin registry model.
package sourceview

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"levnote.local/native/core/text"
	"levnote.local/native/formats/sourceoutline"
	"levnote.local/native/ui/editor"
)

type source struct {
	buffer       *text.Buffer
	config       sourceoutline.Config
	fingerprint  string
	configErr    error
	cachedBuffer *text.Buffer
	version      uint64
	loaded       bool
	items        []editor.OutlineItem
	spans        []text.Range
	truncated    bool
	err          error
	readChunks   int // focused resource evidence; not a UI/API counter
}

// NewOutline copies the small immutable configuration. Invalid configuration
// is reported through the normal fallible snapshot interface, never a panic.
func NewOutline(host editor.Host, config sourceoutline.Config) editor.OutlineSource {
	s := &source{buffer: host.Buffer}
	if config.Schema != 2 || config.PluginID == "" || len(config.PluginID) > 64 || !utf8.ValidString(config.PluginID) {
		s.configErr = errors.New("源码大纲配置需要 schema2 和有效的插件 ID")
		return s
	}
	if err := sourceoutline.ValidateRules(config.Rules); err != nil {
		s.configErr = fmt.Errorf("源码大纲规则无效：%w", err)
		return s
	}
	s.config = sourceoutline.Config{Schema: config.Schema, PluginID: strings.Clone(config.PluginID), Rules: make([]sourceoutline.Rule, len(config.Rules))}
	for i, rule := range config.Rules {
		s.config.Rules[i] = sourceoutline.Rule{Prefix: strings.Clone(rule.Prefix), Suffix: strings.Clone(rule.Suffix), Level: rule.Level}
	}
	hash := sha256.New()
	var encoded [8]byte
	writeInt := func(value int) {
		binary.BigEndian.PutUint64(encoded[:], uint64(value))
		hash.Write(encoded[:])
	}
	writeString := func(value string) {
		writeInt(len(value))
		io.WriteString(hash, value)
	}
	writeString(s.config.PluginID)
	writeInt(s.config.Schema)
	writeInt(len(s.config.Rules))
	for _, rule := range s.config.Rules {
		writeString(rule.Prefix)
		writeString(rule.Suffix)
		writeInt(rule.Level)
	}
	s.fingerprint = hex.EncodeToString(hash.Sum(nil))
	return s
}

func (s *source) Release() {
	s.cachedBuffer, s.loaded, s.items, s.spans, s.err = nil, false, nil, nil, nil
	s.truncated, s.readChunks = false, 0
}

func (s *source) identity(span text.Range) string {
	return fmt.Sprintf("line:%p:%s:%d:%d:%d", s.buffer, s.fingerprint, s.version, span.Start(), span.End())
}

func (s *source) refresh() error {
	if s.configErr != nil {
		return s.configErr
	}
	if s.buffer == nil {
		return fmt.Errorf("没有源码缓冲区：%w", editor.ErrOutlineUnsupported)
	}
	if s.loaded && s.cachedBuffer == s.buffer && s.version == s.buffer.Version() {
		return s.err
	}
	s.Release()
	s.cachedBuffer, s.version, s.loaded = s.buffer, s.buffer.Version(), true
	reader := &bufferReader{buffer: s.buffer, version: s.version}
	result, err := sourceoutline.Scan(reader, s.config.Rules)
	s.readChunks = reader.chunks
	if err != nil {
		s.err = fmt.Errorf("源码大纲读取失败：%w", err)
		return s.err
	}
	if s.buffer.Version() != s.version {
		s.err = editor.ErrOutlineStale
		return s.err
	}
	s.truncated = result.Truncated
	s.items, s.spans = make([]editor.OutlineItem, 0, len(result.Entries)), make([]text.Range, 0, len(result.Entries))
	for _, entry := range result.Entries {
		span := text.Range{Anchor: entry.Start, Head: entry.End}
		s.items = append(s.items, editor.OutlineItem{ID: s.identity(span), Text: entry.Text, Level: entry.Level, Offset: entry.Start})
		s.spans = append(s.spans, span)
	}
	return nil
}

func (s *source) OutlineSnapshot() (editor.OutlineSnapshot, error) {
	if err := s.refresh(); err != nil {
		return editor.OutlineSnapshot{Version: s.version}, err
	}
	return editor.OutlineSnapshot{Version: s.version, Items: append([]editor.OutlineItem(nil), s.items...), Truncated: s.truncated}, nil
}

func (s *source) ResolveOutlineItem(id string) (editor.OutlineItem, error) {
	if s.configErr != nil {
		return editor.OutlineItem{}, s.configErr
	}
	if len(id) > editor.MaxOutlineIDBytes {
		return editor.OutlineItem{}, editor.ErrOutlineStale
	}
	parts := strings.Split(id, ":")
	if len(parts) != 6 || parts[0] != "line" || parts[1] != fmt.Sprintf("%p", s.buffer) || parts[2] != s.fingerprint || s.buffer == nil {
		return editor.OutlineItem{}, editor.ErrOutlineStale
	}
	version, versionErr := strconv.ParseUint(parts[3], 10, 64)
	start, startErr := strconv.Atoi(parts[4])
	end, endErr := strconv.Atoi(parts[5])
	if versionErr != nil || startErr != nil || endErr != nil || start < 0 || end <= start {
		return editor.OutlineItem{}, editor.ErrOutlineStale
	}
	span, ok := s.buffer.MapRangeSince(version, text.Range{Anchor: start, Head: end})
	if !ok {
		return editor.OutlineItem{}, editor.ErrOutlineStale
	}
	if err := s.refresh(); err != nil {
		return editor.OutlineItem{}, err
	}
	for i, current := range s.spans {
		if span == current {
			return s.items[i], nil
		}
	}
	return editor.OutlineItem{}, editor.ErrOutlineStale
}

func (*source) ResolveOutlineAnchor(string) (editor.OutlineItem, error) {
	return editor.OutlineItem{}, fmt.Errorf("字面行大纲没有链接锚点：%w", editor.ErrOutlineUnsupported)
}

// SliceUTF16 uses binary-search piece prefixes, so each bounded window avoids
// scanning preceding source. Windows end on a legal surrogate boundary; the
// io.Reader may still split UTF8 bytes across reads, as the pure scanner allows.
type bufferReader struct {
	buffer  *text.Buffer
	version uint64
	offset  int
	pending string
	chunks  int
}

func (r *bufferReader) Read(target []byte) (int, error) {
	if len(target) == 0 {
		return 0, nil
	}
	if r.buffer.Version() != r.version {
		return 0, editor.ErrOutlineStale
	}
	if len(r.pending) == 0 {
		if r.offset == r.buffer.LenUTF16() {
			return 0, io.EOF
		}
		// At most 2048 UTF16 units. Normal windows materialize <=len(target)
		// UTF8 bytes; a tiny EOF probe may materialize2 units (up to6 bytes)
		// while returning only requested bytes. No full line is copied.
		units := min(2048, max(2, len(target)/3))
		end := min(r.buffer.LenUTF16(), r.offset+units)
		chunk, err := r.buffer.SliceUTF16(r.offset, end)
		if errors.Is(err, text.ErrSurrogate) {
			end--
			chunk, err = r.buffer.SliceUTF16(r.offset, end)
		}
		if err != nil {
			return 0, err
		}
		r.pending, r.offset = chunk, end
		r.chunks++
	}
	n := copy(target, r.pending)
	r.pending = r.pending[n:]
	return n, nil
}
