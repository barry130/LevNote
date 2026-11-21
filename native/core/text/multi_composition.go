package text

import (
	"errors"
	"unicode/utf8"
)

// ErrCompositionPending means an explicit selection-set composition needs the
// native final text (or an explicit CancelCompositionGroup). End/unmark/flush
// cannot promote an intermediate primary preedit to a confirmed broadcast.
var ErrCompositionPending = errors.New("多选输入尚未收到系统最终确认文本；请先确认输入或按 Esc 取消")

// ErrCompositionHistoryLimit refuses a changing multi-point confirmation whose
// complete undo item cannot fit the current history entry/byte limits. The live
// owner and preedit remain available for shorter retry or explicit cancellation.
var ErrCompositionHistoryLimit = errors.New("多选输入超过完整撤销记录的容量上限；请缩短确认文本或按 Esc 取消")

// BeginCompositionSetGroup captures the full starting selection set and opts
// multi-point owners into final-text-only completion. Preedit still updates
// only the primary. A multi-point EndCompositionGroup returns
// ErrCompositionPending, without committing, broadcasting or cancelling; the
// caller must deliver native final text or explicitly cancel. Single-point
// groups retain legacy End and irregular-preedit behavior.
//
// Native multi-selection UI must use this entrypoint rather than legacy
// BeginCompositionGroup, whose primary-only End contract remains supported.
func (b *Buffer) BeginCompositionSetGroup() (CompositionToken, error) {
	token, err := b.BeginCompositionGroup()
	if err != nil {
		return CompositionToken{}, err
	}
	b.composition.requireFinal = len(b.group.beforeSelections.secondary) != 0
	return token, nil
}

// CommitCompositionSetUTF16 confirms finalText at every selection captured by
// the composition group, in one owned history transaction. The token's live
// group is the sole authority; no caller-supplied secondary offsets are used.
// With no preedit it replaces the original primary; otherwise it replaces the
// current accepted primary span. Only native final-text delivery should call
// this API. It ends the token on success, including a same-text confirmation.
//
// All refusals precede the first mutation and preserve the accepted preedit,
// token, selections, revision, journals, history and redo for retry or explicit
// cancellation. Changing multi-confirmations must retain their full undo item
// within current history limits. Multi preedit is primary-only and full-span;
// single-point irregular/coalescing behavior and history trimming are unchanged.
func (b *Buffer) CommitCompositionSetUTF16(token CompositionToken, finalText string) error {
	if !b.OwnsComposition(token) {
		return ErrComposition
	}
	c, group := b.composition, b.group
	span := group.before
	if c.ready {
		span = c.span
	}
	if len(group.beforeSelections.secondary) == 0 {
		if err := b.ReplaceCompositionUTF16(token, span.Start(), span.End(), finalText); err != nil {
			return err
		}
		return b.EndCompositionGroup(token)
	}
	if !utf8.ValidString(finalText) {
		return ErrUTF8
	}
	n, primary := len(group.beforeSelections.secondary)+1, group.beforeSelections.primary
	if n > MaxSelections {
		return ErrSelectionLimit
	}
	if primary < 0 || primary >= n || span.Start() != group.before.Start() {
		return ErrComposition
	}
	// The multi preedit contract permits exactly one coalesced suffix, never
	// the irregular single-point path. Check this before indexing its endpoint.
	if c.suffix && (!c.ready || len(group.edits) != 1) || !c.suffix && len(group.edits) != 0 {
		return ErrComposition
	}
	original := selectionSet(group.before, group.beforeSelections)
	plan := make([]batchReplacement, n)
	for i, r := range original.Ranges {
		start, end := r.Start(), r.End()
		if i == primary {
			start, end = span.Start(), span.End()
		} else if i > primary {
			var ok bool
			start, ok = compositionOffset(span.End(), r.Start(), group.before.End())
			if !ok {
				return ErrRange
			}
			end, ok = compositionOffset(span.End(), r.End(), group.before.End())
			if !ok {
				return ErrRange
			}
		}
		left, err := b.byteOffset(start)
		if err != nil {
			return err
		}
		right, err := b.byteOffset(end)
		if err != nil {
			return err
		}
		if start > end || i > 0 && plan[i-1].End > start {
			return ErrRange
		}
		plan[i] = batchReplacement{Replacement{start, end, finalText}, left, right}
	}
	// Final admission counts all replacements together, before checking the
	// actual primary-first, then descending-secondary intermediate sequence.
	finalBytes := b.bytes
	for _, edit := range plan {
		finalBytes -= edit.right - edit.left
	}
	for range plan {
		var err error
		finalBytes, err = b.CheckReplaceSize(finalBytes, 0, len(finalText))
		if err != nil {
			return err
		}
	}
	p := plan[primary]
	currentBytes, err := b.CheckReplaceSize(b.bytes, p.right-p.left, len(finalText))
	if err != nil {
		return err
	}
	for i := n - 1; i >= 0; i-- {
		if i == primary {
			continue
		}
		edit := plan[i]
		currentBytes, err = b.CheckReplaceSize(currentBytes, edit.right-edit.left, len(finalText))
		if err != nil {
			return err
		}
	}
	inserted := UTF16Len(finalText)
	finalPrimaryEnd, ok := compositionOffset(p.Start, inserted, 0)
	if !ok {
		return ErrRange
	}
	carets := SelectionSet{Ranges: make([]Range, 0, n)}
	previousEnd, outputEnd := 0, 0
	for i, r := range original.Ranges {
		var ok bool
		outputEnd, ok = compositionOffset(outputEnd, r.Start(), previousEnd)
		if !ok {
			return ErrRange
		}
		outputEnd, ok = compositionOffset(outputEnd, inserted, 0)
		if !ok {
			return ErrRange
		}
		caret := Range{outputEnd, outputEnd}
		if len(carets.Ranges) == 0 || carets.Ranges[len(carets.Ranges)-1] != caret {
			carets.Ranges = append(carets.Ranges, caret)
		}
		if i == primary {
			carets.Primary = len(carets.Ranges) - 1
		}
		previousEnd = r.End()
	}
	// Prepare every removed fragment, write coordinate and history endpoint
	// before mutation. Current and original primary strings are already owned
	// by the suffix, except when there has not yet been a primary update.
	oldPrimary, livePrimary := c.baseline, c.baseline
	if c.suffix {
		oldPrimary, livePrimary = group.edits[0].old, group.edits[0].next
	} else if !c.ready {
		oldPrimary = b.sliceBytes(p.left, p.right)
		livePrimary = oldPrimary
	}
	afterExtra := extraOf(carets)
	t := transaction{before: group.before, beforeSelections: group.beforeSelections,
		after: carets.Ranges[carets.Primary], afterSelections: afterExtra,
		edits: make([]change, 0, n)}
	writes := make([]change, 0, n)
	if oldPrimary != finalText {
		t.edits = append(t.edits, change{p.Start, oldPrimary, finalText})
	}
	if livePrimary != finalText {
		writes = append(writes, change{p.Start, livePrimary, finalText})
	}
	for i := n - 1; i >= 0; i-- {
		if i == primary {
			continue
		}
		edit := plan[i]
		start := edit.Start
		if i > primary {
			var ok bool
			start, ok = compositionOffset(finalPrimaryEnd, original.Ranges[i].Start(), group.before.End())
			if !ok {
				return ErrRange
			}
		}
		old := b.sliceBytes(edit.left, edit.right)
		if old != finalText {
			edit := change{start, old, finalText}
			t.edits = append(t.edits, edit)
			writes = append(writes, edit)
		}
	}
	// Compare virtual piece streams, not a whole-buffer materialization or a
	// hash. Adjacent ranges can redistribute identical final bytes even when
	// individual replacements differ. Such a net-zero confirmation keeps redo.
	baseline := [1]batchReplacement{{Replacement{p.Start, p.End, oldPrimary}, p.left, p.right}}
	originalBytes, overflow := saturatedBytes(b.bytes-(p.right-p.left), len(oldPrimary))
	if overflow {
		return ErrRange
	}
	if finalBytes == originalBytes && b.equalCompositionStreams(plan, baseline[:]) {
		clear(t.edits)
		t.edits = t.edits[:0]
	}
	t.recountBytes()
	// A successful changing multi-confirmation promises one complete undo.
	// Admit its standalone cost before writing, so normal trim may remove
	// older history but can never immediately remove this accepted item.
	if len(t.edits) != 0 && !b.admitsCompositionHistory(t) {
		return ErrCompositionHistoryLimit
	}
	if finalBytes == b.bytes && b.batchTextUnchanged(plan) {
		writes = writes[:0]
	}
	// No operation below can return an error or relinquish ownership midway.
	for _, edit := range writes {
		b.replace(edit.start, edit.start+UTF16Len(edit.old), edit.next)
	}
	b.setSelections(t.after, t.afterSelections, false)
	clear(group.edits)
	b.group, b.composition = nil, nil
	if len(t.edits) != 0 {
		b.commit(t)
	}
	return nil
}

// Keep the composition entry point for its existing package contract/tests.
func (b *Buffer) admitsCompositionHistory(t transaction) bool {
	return b.admitsStandaloneHistory(t)
}

func (b *Buffer) admitsStandaloneHistory(t transaction) bool {
	retained, overflow := saturatedBytes(t.bytes, t.selectionBytes())
	return b.historyEntries >= 1 && !t.bytesOverflow && !overflow && retained <= b.historyBytes
}

// Map an offset after an old boundary onto the corresponding new boundary
// without signed deltas or overflowing intermediate sums.
func compositionOffset(base, offset, oldBoundary int) (int, bool) {
	const maxInt = int(^uint(0) >> 1)
	if base < 0 || offset < oldBoundary || oldBoundary < 0 || offset-oldBoundary > maxInt-base {
		return 0, false
	}
	return base + (offset - oldBoundary), true
}

// compositionStream emits immutable piece fragments and inserted strings. Its
// work is bounded by current piece count plus the planned replacements;
// it never joins the document or creates one string for an untouched gap.
type compositionStream struct {
	buffer       *Buffer
	plan         []batchReplacement
	offset, edit int
	inserted     bool
}

func (s *compositionStream) next() string {
	for {
		end := s.buffer.bytes
		if s.edit < len(s.plan) {
			edit := s.plan[s.edit]
			end = edit.left
			if s.offset == end {
				if !s.inserted {
					s.inserted, s.offset = true, edit.right
					if edit.Text != "" {
						return edit.Text
					}
				}
				s.edit++
				s.inserted = false
				continue
			}
			if s.inserted {
				s.edit++
				s.inserted = false
				continue
			}
		}
		if s.offset == s.buffer.bytes {
			return ""
		}
		i := s.buffer.pieceAtByte(s.offset)
		p := s.buffer.pieces[i]
		start := p.start + s.offset - s.buffer.prefix[i].bytes
		n := min(end-s.offset, p.end-start)
		s.offset += n
		return p.data.value[start : start+n]
	}
}

func (b *Buffer) equalCompositionStreams(a, z []batchReplacement) bool {
	left, right := compositionStream{buffer: b, plan: a}, compositionStream{buffer: b, plan: z}
	var x, y string
	for {
		if x == "" {
			x = left.next()
		}
		if y == "" {
			y = right.next()
		}
		if x == "" || y == "" {
			return x == y
		}
		n := min(len(x), len(y))
		if x[:n] != y[:n] {
			return false
		}
		x, y = x[n:], y[n:]
	}
}
