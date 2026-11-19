package markdownview

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/ui/editor"
)

type headingEntry struct {
	item editor.OutlineItem
	span text.Range
}

type outlineSource struct {
	buffer       *text.Buffer
	cachedBuffer *text.Buffer
	version      uint64
	loaded       bool
	entries      []headingEntry
	truncated    bool
	partial      bool
	err          error
}

func newOutlineSource(host editor.Host) editor.OutlineSource {
	return &outlineSource{buffer: host.Buffer}
}

func (s *outlineSource) Release() {
	s.cachedBuffer, s.loaded, s.entries, s.err = nil, false, nil, nil
	s.truncated, s.partial = false, false
}

func headingID(buffer *text.Buffer, version uint64, span text.Range) string {
	return fmt.Sprintf("md:%p:%d:%d:%d", buffer, version, span.Start(), span.End())
}

func (s *outlineSource) refresh() error {
	if s.buffer == nil {
		return fmt.Errorf("没有可用的文档缓冲区：%w", editor.ErrOutlineUnsupported)
	}
	if s.loaded && s.cachedBuffer == s.buffer && s.version == s.buffer.Version() {
		return s.err
	}
	s.Release()
	s.cachedBuffer, s.version, s.loaded = s.buffer, s.buffer.Version(), true
	if s.buffer.LenBytes() > markdown.MaxParseBytes {
		s.err = fmt.Errorf("Markdown 超过大纲处理上限，请在源码中查找：%w", editor.ErrOutlineUnsupported)
		return s.err
	}
	doc := markdown.Parse(s.buffer.String())
	headings, truncated := markdown.Headings(doc, editor.MaxOutlineItems)
	s.truncated, s.partial = truncated, doc.Partial
	s.entries = make([]headingEntry, 0, len(headings))
	byteCursor, unitCursor := 0, 0
	for _, heading := range headings {
		unitCursor += text.UTF16Len(doc.Raw()[byteCursor:heading.Range.Start])
		start := unitCursor
		unitCursor += text.UTF16Len(doc.Raw()[heading.Range.Start:heading.Range.End])
		byteCursor = heading.Range.End
		span := text.Range{Anchor: start, Head: unitCursor}
		label, clipped := editor.OutlineLabel(heading.Text)
		s.truncated = s.truncated || clipped
		s.entries = append(s.entries, headingEntry{
			item: editor.OutlineItem{ID: headingID(s.buffer, s.version, span), AnchorID: heading.AnchorID,
				Text: label, Level: heading.Level, Offset: start}, span: span,
		})
	}
	return nil
}

func (s *outlineSource) OutlineSnapshot() (editor.OutlineSnapshot, error) {
	if err := s.refresh(); err != nil {
		return editor.OutlineSnapshot{Version: s.version}, err
	}
	items := make([]editor.OutlineItem, len(s.entries))
	for i, entry := range s.entries {
		items[i] = entry.item
	}
	return editor.OutlineSnapshot{Version: s.version, Items: items, Truncated: s.truncated}, nil
}

func (s *outlineSource) ResolveOutlineItem(id string) (editor.OutlineItem, error) {
	if len(id) > editor.MaxOutlineIDBytes {
		return editor.OutlineItem{}, editor.ErrOutlineStale
	}
	parts := strings.Split(id, ":")
	if len(parts) != 5 || parts[0] != "md" || parts[1] != fmt.Sprintf("%p", s.buffer) {
		return editor.OutlineItem{}, editor.ErrOutlineStale
	}
	version, versionErr := strconv.ParseUint(parts[2], 10, 64)
	start, startErr := strconv.Atoi(parts[3])
	end, endErr := strconv.Atoi(parts[4])
	if versionErr != nil || startErr != nil || endErr != nil || start < 0 || end <= start || s.buffer == nil {
		return editor.OutlineItem{}, editor.ErrOutlineStale
	}
	span, ok := s.buffer.MapRangeSince(version, text.Range{Anchor: start, Head: end})
	if !ok {
		return editor.OutlineItem{}, editor.ErrOutlineStale
	}
	if err := s.refresh(); err != nil {
		return editor.OutlineItem{}, err
	}
	for _, entry := range s.entries {
		if entry.span == span {
			return entry.item, nil
		}
	}
	return editor.OutlineItem{}, editor.ErrOutlineStale
}

func (s *outlineSource) ResolveOutlineAnchor(anchor string) (editor.OutlineItem, error) {
	if len(anchor) > editor.MaxOutlineIDBytes {
		return editor.OutlineItem{}, editor.ErrOutlineStale
	}
	if err := s.refresh(); err != nil {
		return editor.OutlineItem{}, err
	}
	for _, entry := range s.entries {
		if entry.item.AnchorID == anchor {
			return entry.item, nil
		}
	}
	return editor.OutlineItem{}, editor.ErrOutlineStale
}

func (v *View) outlineSource() *outlineSource {
	if v.outline == nil || v.outline.buffer != v.host.Buffer {
		v.outline = &outlineSource{buffer: v.host.Buffer}
	}
	return v.outline
}

func (v *View) NavigateOutlineItem(id string) error {
	if err := v.Flush(); err != nil {
		return err
	}
	source := v.outlineSource()
	item, err := source.ResolveOutlineItem(id)
	if err != nil {
		return err
	}
	if source.partial {
		return fmt.Errorf("当前预览不完整，请切到源码定位：%w", editor.ErrOutlineUnsupported)
	}
	// Only after successful flush and identity resolution may the old editor
	// finish. A stale click/conflict keeps the independent draft recoverable.
	v.clearDraft()
	v.refresh()
	for i, row := range v.rows {
		if row.kind == markdown.BlockHeading && row.span.Start() == item.Offset {
			v.list.ScrollTo(i, ui.Start)
			v.outlineCurrent = item.ID
			v.outlineCaret = v.host.Buffer.Selection().Head
			return nil
		}
	}
	return editor.ErrOutlineStale
}

func (v *View) CurrentOutlineItem() (string, bool) {
	source := v.outlineSource()
	if err := source.refresh(); err != nil {
		return "", false
	}
	caret := v.host.Buffer.Selection().Head
	if v.outlineCurrent != "" && v.outlineCaret == caret {
		if item, err := source.ResolveOutlineItem(v.outlineCurrent); err == nil {
			return item.ID, true
		}
	}
	id := ""
	for _, entry := range source.entries {
		if entry.item.Offset > caret {
			break
		}
		id = entry.item.ID
	}
	return id, id != ""
}
