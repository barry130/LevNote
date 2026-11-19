// Package text implements LevNote's text model independently of any UI toolkit.
// Public editing offsets are UTF-16 code units, as required by native input
// methods. Storage remains lossless UTF-8; a surrogate pair is never split.
package text

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"
)

var (
	ErrRange       = errors.New("text range is outside the document")
	ErrSurrogate   = errors.New("text range splits a UTF-16 surrogate pair")
	ErrUTF8        = errors.New("text is not valid UTF-8")
	ErrGroup       = errors.New("an edit transaction is already open")
	ErrSizeLimit   = errors.New("edit exceeds the document content limit")
	ErrComposition = errors.New("文本组合事务已失效；请按 Esc 取消未完成的输入后重试")
)

const chunkBytes = 4096

// DefaultMaxBytes bounds ordinary UTF-8 logical-content growth, not disk reads
// or process memory. An oversized decoded seed remains intact and editable.
const DefaultMaxBytes = 128 << 20

type SizeLimitError struct{ Limit int }

func (e *SizeLimitError) Error() string {
	return fmt.Sprintf("此次编辑超过文档的 UTF-8 内容上限（%d 字节）；请先缩短内容或拆分文档后重试", e.Limit)
}
func (e *SizeLimitError) Unwrap() error { return ErrSizeLimit }

// Range preserves selection direction. Start and End return ordered bounds.
type Range struct{ Anchor, Head int }

func (r Range) Start() int {
	if r.Anchor < r.Head {
		return r.Anchor
	}
	return r.Head
}
func (r Range) End() int {
	if r.Anchor > r.Head {
		return r.Anchor
	}
	return r.Head
}

type chunk struct{ value string }
type piece struct {
	data                     *chunk
	start, end, units, lines int
	hash                     uint64
}
type piecePrefix struct{ bytes, units, lines, column int }
type change struct {
	start     int
	old, next string
}
type transaction struct {
	edits         []change
	before, after Range
	bytes         int
	bytesOverflow bool
}

// CompositionToken identifies one group in its issuing Buffer. It keeps no
// buffer, transaction or text alive. A nonzero-sized private allocation gives
// each live identity a distinct address, including across different buffers.
type CompositionToken struct{ identity *compositionIdentity }
type compositionIdentity struct{ marker byte }
type compositionGroup struct {
	identity      *compositionIdentity
	owner         *Buffer
	span          Range
	ready, suffix bool
	// Only a net-zero/initial no-op suffix needs a separate baseline. A live
	// suffix keeps its original/current strings in the last transaction edit.
	baseline string
}

// Snapshot shares immutable chunks and has no UI or native handles. Background
// workers may read a snapshot while the session continues editing the Buffer.
type Snapshot struct {
	pieces              []piece
	units, bytes, lines int
	selection           Range
	hash                uint64
}

type HistoryStatus struct {
	UndoDepth, RedoDepth, RetainedBytes int
	// The shell can disclose that a history budget was reached, rather than
	// claiming unlimited undo. ResetHistory clears this flag explicitly.
	Truncated bool
}

// Buffer is owned by one document/session goroutine. It uses a chunked piece
// table: edits copy bounded chunks instead of the entire document. Independently
// owned strings let deleted parts of large files be reclaimed with history.
type Buffer struct {
	pieces                       []piece
	prefix                       []piecePrefix
	units, bytes, lines          int
	selection                    Range
	version                      uint64
	hash                         uint64
	undo, redo                   []transaction
	group                        *transaction
	composition                  *compositionGroup
	historyEntries, historyBytes int
	baseLimit, maxBytes          int
	truncated                    bool
	// A bounded invalidation journal lets views keep geometry preceding an
	// edit. It contains coordinates only, never another copy of document text.
	changedLines [64]int
	rangeChanges [64]rangeChange
}

func New(value string) *Buffer {
	return NewWithLimit(value, DefaultMaxBytes)
}

// NewWithLimit fixes a positive base limit for the buffer's lifetime. Each
// initial/Restore seed sets a ceiling of max(base, seed bytes); ordinary edits
// never lower it, so history can replay every previously accepted state.
func NewWithLimit(value string, limit int) *Buffer {
	if !utf8.ValidString(value) {
		panic(ErrUTF8)
	}
	if limit <= 0 {
		panic("text content limit must be positive")
	}
	b := &Buffer{historyEntries: 256, historyBytes: 32 << 20, baseLimit: limit, maxBytes: max(limit, len(value))}
	b.pieces = piecesOf(value)
	b.recount()
	return b
}
func (b *Buffer) Version() uint64  { return b.version }
func (b *Buffer) LenUTF16() int    { return b.units }
func (b *Buffer) LenBytes() int    { return b.bytes }
func (b *Buffer) MaxBytes() int    { return b.maxBytes }
func (b *Buffer) Lines() int       { return b.lines + 1 }
func (b *Buffer) LineCount() int   { return b.Lines() }
func (b *Buffer) Selection() Range { return b.selection }
func (b *Buffer) InGroup() bool    { return b.group != nil }

// ChangedLineSince returns the first logical line whose geometry may have
// changed. An old or reset revision conservatively invalidates the whole view.
func (b *Buffer) ChangedLineSince(version uint64) int {
	if version > b.version || b.version-version > uint64(len(b.changedLines)) {
		return 0
	}
	first := b.LineCount()
	for v := version + 1; v <= b.version; v++ {
		first = min(first, b.changedLines[v%uint64(len(b.changedLines))])
	}
	return first
}

func UTF16Len(value string) int {
	n := 0
	for _, r := range value {
		if r > 0xffff {
			n += 2
		} else {
			n++
		}
	}
	return n
}
func (b *Buffer) String() string {
	var out strings.Builder
	out.Grow(b.bytes)
	for _, p := range b.pieces {
		out.WriteString(p.data.value[p.start:p.end])
	}
	return out.String()
}
func (b *Buffer) WriteTo(w io.Writer) (int64, error) { return b.Snapshot().WriteTo(w) }
func (s Snapshot) WriteTo(w io.Writer) (int64, error) {
	var total int64
	for _, p := range s.pieces {
		n, err := io.WriteString(w, p.data.value[p.start:p.end])
		total += int64(n)
		if err != nil {
			return total, err
		}
		if n != p.end-p.start {
			return total, io.ErrShortWrite
		}
	}
	return total, nil
}
func (s Snapshot) String() string {
	var out strings.Builder
	out.Grow(s.bytes)
	_, _ = s.WriteTo(&out)
	return out.String()
}
func (b *Buffer) Snapshot() Snapshot {
	return Snapshot{append([]piece(nil), b.pieces...), b.units, b.bytes, b.lines, b.selection, b.hash}
}

// EqualSnapshot rejects ordinary changes through a composable fingerprint. Hash
// equality is confirmed byte-for-byte, so collisions can never hide an edit.
func (b *Buffer) EqualSnapshot(s Snapshot) bool {
	if b.bytes != s.bytes || b.hash != s.hash {
		return false
	}
	i, j, bi, bj := 0, 0, 0, 0
	for i < len(b.pieces) && j < len(s.pieces) {
		p, q := b.pieces[i], s.pieces[j]
		n := p.end - p.start - bi
		if m := q.end - q.start - bj; m < n {
			n = m
		}
		if p.data.value[p.start+bi:p.start+bi+n] != q.data.value[q.start+bj:q.start+bj+n] {
			return false
		}
		bi += n
		bj += n
		if bi == p.end-p.start {
			i++
			bi = 0
		}
		if bj == q.end-q.start {
			j++
			bj = 0
		}
	}
	return i == len(b.pieces) && j == len(s.pieces)
}

// Restore is an explicit reload/recovery reset and clears history. Ordinary
// commands should edit the buffer, so the user can still undo them.
func (b *Buffer) Restore(s Snapshot) {
	b.pieces = append([]piece(nil), s.pieces...)
	b.units, b.bytes, b.lines, b.selection, b.hash = s.units, s.bytes, s.lines, s.selection, s.hash
	b.maxBytes = max(b.baseLimit, s.bytes)
	b.recount()
	b.ResetHistory()
	b.version++
	b.changedLines[b.version%uint64(len(b.changedLines))] = 0
	b.rangeChanges[b.version%uint64(len(b.rangeChanges))] = rangeChange{reset: true}
}
func (b *Buffer) SetSelection(anchor, head int) error {
	if _, err := b.byteOffset(anchor); err != nil {
		return err
	}
	if _, err := b.byteOffset(head); err != nil {
		return err
	}
	b.selection = Range{anchor, head}
	return nil
}
func (b *Buffer) SliceUTF16(start, end int) (string, error) {
	if start > end {
		return "", ErrRange
	}
	left, err := b.byteOffset(start)
	if err != nil {
		return "", err
	}
	right, err := b.byteOffset(end)
	if err != nil {
		return "", err
	}
	return b.sliceBytes(left, right), nil
}

// CheckReplaceSize preflights one edit in a caller's planned sequence. The
// counts are bytes, not UTF-16 units; no text or history is copied. Subtraction
// avoids overflow even for caller-supplied lengths near the integer limit.
func (b *Buffer) CheckReplaceSize(current, removed, added int) (int, error) {
	if current < 0 || removed < 0 || removed > current || added < 0 {
		return 0, ErrRange
	}
	remaining := current - removed
	if remaining > b.maxBytes || added > b.maxBytes-remaining {
		return 0, &SizeLimitError{Limit: b.maxBytes}
	}
	return remaining + added, nil
}

func (b *Buffer) replacementBytes(start, end int, value string) (int, int, error) {
	if !utf8.ValidString(value) {
		return 0, 0, ErrUTF8
	}
	if start > end {
		return 0, 0, ErrRange
	}
	left, err := b.byteOffset(start)
	if err != nil {
		return 0, 0, err
	}
	right, err := b.byteOffset(end)
	if err != nil {
		return 0, 0, err
	}
	_, err = b.CheckReplaceSize(b.bytes, right-left, len(value))
	return left, right, err
}

// CheckReplaceUTF16 has the same UTF-8, range, surrogate and size precedence as
// ReplaceUTF16, without opening a group or changing any editing state.
func (b *Buffer) CheckReplaceUTF16(start, end int, value string) error {
	_, _, err := b.replacementBytes(start, end, value)
	return err
}

func (b *Buffer) ReplaceUTF16(start, end int, value string) error {
	if b.composition != nil {
		return ErrGroup
	}
	left, right, err := b.replacementBytes(start, end, value)
	if err != nil {
		return err
	}
	// Admission precedes materializing removed bytes or allocating history.
	old := b.sliceBytes(left, right)
	before := b.selection
	if old == value {
		b.selection = Range{start + UTF16Len(value), start + UTF16Len(value)}
		return nil
	}
	b.replace(start, end, value)
	after := Range{start + UTF16Len(value), start + UTF16Len(value)}
	b.selection = after
	c := change{start: start, old: old, next: value}
	if b.group != nil {
		b.group.edits = append(b.group.edits, c)
		b.group.after = after
		b.group.addBytes(len(old))
		b.group.addBytes(len(value))
	} else {
		t := transaction{edits: []change{c}, before: before, after: after}
		t.recountBytes()
		b.commit(t)
	}
	return nil
}

// BeginCompositionGroup opts into full-span suffix coalescing. Ordinary groups
// retain append semantics; tokenless mutation/end/cancel cannot touch this one.
func (b *Buffer) BeginCompositionGroup() (CompositionToken, error) {
	if err := b.BeginGroup(); err != nil {
		return CompositionToken{}, err
	}
	id := &compositionIdentity{marker: 1}
	b.composition = &compositionGroup{identity: id, owner: b}
	return CompositionToken{identity: id}, nil
}

func (b *Buffer) OwnsComposition(token CompositionToken) bool {
	return token.identity != nil && b.group != nil && b.composition != nil && b.composition.owner == b && b.composition.identity == token.identity
}

// ReplaceCompositionUTF16 compacts only consecutive replacements of exactly
// the previous accepted span. Irregular ranges append an ordinary edit and
// start a new coalescible suffix; arbitrary irregular sequences are not bounded.
func (b *Buffer) ReplaceCompositionUTF16(token CompositionToken, start, end int, value string) error {
	if !b.OwnsComposition(token) {
		return ErrComposition
	}
	left, right, err := b.replacementBytes(start, end, value)
	if err != nil {
		return err
	}
	c, t := b.composition, b.group
	full := c.ready && start == c.span.Start() && end == c.span.End()
	current := c.baseline
	if full && c.suffix {
		current = t.edits[len(t.edits)-1].next
	} else if !full {
		// The exact full-span path already owns the current text; only an
		// irregular or first replacement materializes the removed bytes.
		current = b.sliceBytes(left, right)
	}
	if current != value {
		b.replace(start, end, value)
	}
	after := Range{start + UTF16Len(value), start + UTF16Len(value)}
	b.selection, t.after = after, after
	if full && c.suffix {
		i := len(t.edits) - 1
		last := &t.edits[i]
		oldNextBytes := len(last.next)
		if value == last.old {
			c.baseline = last.old
			oldBytes := len(last.old)
			*last = change{}
			t.edits = t.edits[:i]
			c.suffix = false
			if t.bytesOverflow {
				t.recountBytes()
			} else {
				t.bytes -= oldNextBytes
				t.bytes -= oldBytes
			}
		} else {
			last.next = value
			if t.bytesOverflow {
				t.recountBytes()
			} else {
				t.bytes -= oldNextBytes
				t.addBytes(len(value))
			}
		}
	} else {
		c.baseline, c.suffix = current, false
		if current != value {
			t.edits = append(t.edits, change{start: start, old: current, next: value})
			t.addBytes(len(current))
			t.addBytes(len(value))
			c.baseline, c.suffix = "", true
		}
	}
	c.span, c.ready = Range{start, after.Head}, true
	return nil
}

func (b *Buffer) EndCompositionGroup(token CompositionToken) error {
	if !b.OwnsComposition(token) {
		return ErrComposition
	}
	b.composition = nil
	b.EndGroup()
	return nil
}

func (b *Buffer) CancelCompositionGroup(token CompositionToken) error {
	if !b.OwnsComposition(token) {
		return ErrComposition
	}
	b.composition = nil
	b.CancelGroup()
	return nil
}

func saturatedBytes(total, added int) (int, bool) {
	const maxInt = int(^uint(0) >> 1)
	if added > maxInt-total {
		return maxInt, true
	}
	return total + added, false
}

func (t *transaction) addBytes(n int) {
	if !t.bytesOverflow {
		t.bytes, t.bytesOverflow = saturatedBytes(t.bytes, n)
	}
}

func (t *transaction) recountBytes() {
	t.bytes, t.bytesOverflow = 0, false
	for _, edit := range t.edits {
		t.addBytes(len(edit.old))
		t.addBytes(len(edit.next))
	}
}

// Grouping gives composition updates and multi-step commands a single undo
// action. Cancel restores the original selection as well as the original text.
func (b *Buffer) BeginGroup() error {
	if b.group != nil {
		return ErrGroup
	}
	b.group = &transaction{before: b.selection, after: b.selection}
	return nil
}
func (b *Buffer) EndGroup() bool {
	if b.group == nil || b.composition != nil {
		return false
	}
	t := *b.group
	b.group = nil
	t.after = b.selection
	if len(t.edits) == 0 {
		return false
	}
	b.commit(t)
	return true
}
func (b *Buffer) CancelGroup() bool {
	if b.group == nil || b.composition != nil {
		return false
	}
	t := *b.group
	b.group = nil
	for i := len(t.edits) - 1; i >= 0; i-- {
		c := t.edits[i]
		b.replace(c.start, c.start+UTF16Len(c.next), c.old)
	}
	b.selection = t.before
	return true
}
func (b *Buffer) CanUndo() bool { return b.group == nil && len(b.undo) > 0 }
func (b *Buffer) CanRedo() bool { return b.group == nil && len(b.redo) > 0 }
func (b *Buffer) Undo() bool {
	if !b.CanUndo() {
		return false
	}
	i := len(b.undo) - 1
	t := b.undo[i]
	b.undo[i] = transaction{}
	b.undo = b.undo[:i]
	for i := len(t.edits) - 1; i >= 0; i-- {
		c := t.edits[i]
		b.replace(c.start, c.start+UTF16Len(c.next), c.old)
	}
	b.selection = t.before
	b.redo = append(b.redo, t)
	return true
}
func (b *Buffer) Redo() bool {
	if !b.CanRedo() {
		return false
	}
	i := len(b.redo) - 1
	t := b.redo[i]
	b.redo[i] = transaction{}
	b.redo = b.redo[:i]
	for _, c := range t.edits {
		b.replace(c.start, c.start+UTF16Len(c.old), c.next)
	}
	b.selection = t.after
	b.undo = append(b.undo, t)
	return true
}
func (b *Buffer) ResetHistory() {
	b.undo = nil
	b.redo = nil
	b.group = nil
	b.composition = nil
	b.truncated = false
}
func (b *Buffer) SetHistoryLimits(entries, bytes int) {
	if entries < 0 {
		entries = 0
	}
	if bytes < 0 {
		bytes = 0
	}
	b.historyEntries, b.historyBytes = entries, bytes
	b.trimHistory()
}
func (b *Buffer) History() HistoryStatus {
	n, _ := b.retainedHistoryBytes()
	return HistoryStatus{len(b.undo), len(b.redo), n, b.truncated}
}

func (b *Buffer) retainedHistoryBytes() (int, bool) {
	n := 0
	for _, history := range [][]transaction{b.undo, b.redo} {
		for _, t := range history {
			var overflow bool
			n, overflow = saturatedBytes(n, t.bytes)
			if overflow || t.bytesOverflow {
				return int(^uint(0) >> 1), true
			}
		}
	}
	return n, false
}
func (b *Buffer) commit(t transaction) { b.redo = nil; b.undo = append(b.undo, t); b.trimHistory() }
func (b *Buffer) trimHistory() {
	for {
		n, overflow := b.retainedHistoryBytes()
		if len(b.undo) <= b.historyEntries && len(b.redo) <= b.historyEntries-len(b.undo) && !overflow && n <= b.historyBytes {
			return
		}
		b.truncated = true
		if len(b.undo) > 0 {
			b.undo[0] = transaction{}
			b.undo = b.undo[1:]
		} else if len(b.redo) > 0 {
			b.redo[0] = transaction{}
			b.redo = b.redo[1:]
		} else {
			return
		}
	}
}

// Line is zero-based, excludes LF and a preceding CR, and never changes the
// stored ending. A final newline produces a final empty line.
func (b *Buffer) Line(index int) (string, error) {
	span, err := b.LineRange(index)
	if err != nil {
		return "", err
	}
	return b.SliceUTF16(span.Start(), span.End())
}

// LineRange returns content offsets without materializing the entire line.
// A native view can page a very long line using bounded SliceUTF16 requests.
func (b *Buffer) LineRange(index int) (Range, error) {
	start, err := b.OffsetOfLine(index)
	if err != nil {
		return Range{}, err
	}
	end := b.units
	if index+1 < b.Lines() {
		end, _ = b.OffsetOfLine(index + 1)
		end--
		if end > start {
			last, lastErr := b.SliceUTF16(end-1, end)
			if lastErr == nil && last == "\r" {
				end--
			}
		}
	}
	return Range{start, end}, nil
}
func (b *Buffer) OffsetOfLine(index int) (int, error) {
	if index < 0 || index >= b.Lines() {
		return 0, ErrRange
	}
	if index == 0 {
		return 0, nil
	}
	i := sort.Search(len(b.pieces), func(i int) bool { return b.prefix[i+1].lines >= index })
	p, prefix := b.pieces[i], b.prefix[i]
	offset, line := prefix.units, prefix.lines
	for _, r := range p.data.value[p.start:p.end] {
		offset++
		if r > 0xffff {
			offset++
		}
		if r == '\n' {
			line++
			if line == index {
				return offset, nil
			}
		}
	}
	return b.units, nil
}
func (b *Buffer) LineColumn(offset int) (int, int, error) {
	if _, err := b.byteOffset(offset); err != nil {
		return 0, 0, err
	}
	i := b.pieceAtUnit(offset)
	prefix := b.prefix[i]
	line, column, left := prefix.lines, prefix.column, offset-prefix.units
	if i == len(b.pieces) {
		return line, column, nil
	}
	p := b.pieces[i]
	for _, r := range p.data.value[p.start:p.end] {
		if left == 0 {
			return line, column, nil
		}
		n := 1
		if r > 0xffff {
			n = 2
		}
		left -= n
		if r == '\n' {
			line++
			column = 0
		} else {
			column += n
		}
	}
	return line, column, nil
}

func piecesOf(value string) []piece {
	var out []piece
	for len(value) > 0 {
		n := len(value)
		if n > chunkBytes {
			n = chunkBytes
			for !utf8.RuneStart(value[n]) {
				n--
			}
		}
		owned := strings.Clone(value[:n])
		c := &chunk{owned}
		out = append(out, makePiece(c, 0, n))
		value = value[n:]
	}
	return out
}
func makePiece(data *chunk, start, end int) piece {
	value := data.value[start:end]
	var hash uint64
	for i := 0; i < len(value); i++ {
		hash = hash*257 + uint64(value[i]) + 1
	}
	return piece{data, start, end, UTF16Len(value), strings.Count(value, "\n"), hash}
}
func pow257(n int) uint64 {
	result, base := uint64(1), uint64(257)
	for n > 0 {
		if n&1 != 0 {
			result *= base
		}
		base *= base
		n >>= 1
	}
	return result
}
func (b *Buffer) recount() {
	b.units, b.bytes, b.lines, b.hash = 0, 0, 0, 0
	if cap(b.prefix) < len(b.pieces)+1 {
		b.prefix = make([]piecePrefix, len(b.pieces)+1)
	} else {
		b.prefix = b.prefix[:len(b.pieces)+1]
		b.prefix[0] = piecePrefix{}
	}
	column := 0
	for i, p := range b.pieces {
		n := p.end - p.start
		b.units += p.units
		b.bytes += n
		b.lines += p.lines
		b.hash = b.hash*pow257(n) + p.hash
		if p.lines == 0 {
			column += p.units
		} else {
			value := p.data.value[p.start:p.end]
			column = UTF16Len(value[strings.LastIndexByte(value, '\n')+1:])
		}
		b.prefix[i+1] = piecePrefix{b.bytes, b.units, b.lines, column}
	}
}

func (b *Buffer) pieceAtUnit(offset int) int {
	return sort.Search(len(b.pieces), func(i int) bool { return b.prefix[i+1].units > offset })
}

func (b *Buffer) byteOffset(offset int) (int, error) {
	if offset < 0 || offset > b.units {
		return 0, ErrRange
	}
	i := b.pieceAtUnit(offset)
	bytes := b.prefix[i].bytes
	offset -= b.prefix[i].units
	if i == len(b.pieces) {
		return bytes, nil
	}
	p := b.pieces[i]
	for i, r := range p.data.value[p.start:p.end] {
		if offset == 0 {
			return bytes + i, nil
		}
		n := 1
		if r > 0xffff {
			n = 2
		}
		if offset < n {
			return 0, ErrSurrogate
		}
		offset -= n
	}
	return bytes, nil
}
func (b *Buffer) sliceBytes(start, end int) string {
	var out strings.Builder
	out.Grow(end - start)
	first := sort.Search(len(b.pieces), func(i int) bool { return b.prefix[i+1].bytes > start })
	base := b.prefix[first].bytes
	for _, p := range b.pieces[first:] {
		n := p.end - p.start
		lo, hi := start-base, end-base
		if lo < 0 {
			lo = 0
		}
		if hi > n {
			hi = n
		}
		if lo < hi {
			out.WriteString(p.data.value[p.start+lo : p.start+hi])
		}
		base += n
		if base >= end {
			break
		}
	}
	return out.String()
}
func (b *Buffer) replace(start, end int, value string) {
	firstLine, _, _ := b.LineColumn(start)
	oldUnits := b.units
	left, _ := b.byteOffset(start)
	right, _ := b.byteOffset(end)
	out := make([]piece, 0, len(b.pieces)+4)
	base := 0
	inserted := false
	for _, p := range b.pieces {
		n := p.end - p.start
		if base+n <= left {
			out = append(out, p)
		} else if base >= right {
			if !inserted {
				out = append(out, piecesOf(value)...)
				inserted = true
			}
			out = append(out, p)
		} else {
			if left > base {
				out = append(out, makePiece(p.data, p.start, p.start+left-base))
			}
			if !inserted {
				out = append(out, piecesOf(value)...)
				inserted = true
			}
			if right < base+n {
				out = append(out, makePiece(p.data, p.start+right-base, p.end))
			}
		}
		base += n
	}
	if !inserted {
		out = append(out, piecesOf(value)...)
	}
	// Coalescing only small adjacent pieces bounds fragmentation from typing,
	// while a keystroke never copies more than one 4 KiB backing chunk here.
	merged := out[:0]
	for _, p := range out {
		if len(merged) > 0 {
			last := &merged[len(merged)-1]
			if last.data == p.data && last.end == p.start {
				last.end = p.end
				last.units += p.units
				last.lines += p.lines
				last.hash = last.hash*pow257(p.end-p.start) + p.hash
				continue
			}
			if last.end-last.start+p.end-p.start <= chunkBytes {
				joined := last.data.value[last.start:last.end] + p.data.value[p.start:p.end]
				*last = makePiece(&chunk{joined}, 0, len(joined))
				continue
			}
		}
		merged = append(merged, p)
	}
	// merged reuses out's backing array; clear slots removed by coalescing so
	// the shortened piece slice cannot retain dead chunks through its capacity.
	clear(out[len(merged):])
	b.pieces = merged
	b.recount()
	b.version++
	b.changedLines[b.version%uint64(len(b.changedLines))] = firstLine
	b.rangeChanges[b.version%uint64(len(b.rangeChanges))] = rangeChange{start: start, end: end, inserted: b.units - oldUnits + end - start}
}
