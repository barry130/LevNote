package text

import (
	"errors"
	"sort"
	"strings"
	"unicode/utf8"
)

var (
	ErrStaleEdit = errors.New("text or selection changed before the batch edit")
	ErrEditOrder = errors.New("batch edit ranges must be normalized in document order")
)

// Replacement is one replacement in the original, pre-batch UTF-16 space.
// Start <= End, and Text must be valid UTF-8. Adjacent nonempty intervals are
// independent; an empty interval must not touch another edit's closed bounds.
type Replacement struct {
	Start, End int
	Text       string
}

type batchReplacement struct {
	Replacement
	left, right int
}

// ReplaceBatchUTF16 applies an already normalized edit list as one transaction.
// primary indexes edits, not the buffer's current selection set. The complete
// current set is the undo before-state; output selections are insertion-end
// carets, in document order, with coincident final carets deduplicated.
//
// All predictable refusal paths precede text/selection/history mutation. An
// existing ordinary or composition group belongs to its caller and is refused,
// even for a no-op. This API never settles an IME token or cancels a foreign
// group. A batch whose final bytes equal the original only updates selections,
// preserving the text revision, invalidation journal, history and redo.
func (b *Buffer) ReplaceBatchUTF16(version, selectionGeneration uint64, edits []Replacement, primary int) error {
	if b.group != nil || b.composition != nil {
		return ErrGroup
	}
	if version != b.version || selectionGeneration != b.selectionGeneration {
		return ErrStaleEdit
	}
	if len(edits) > MaxSelections {
		return ErrSelectionLimit
	}
	if len(edits) == 0 || primary < 0 || primary >= len(edits) {
		return ErrSelectionSet
	}
	plan := make([]batchReplacement, len(edits))
	for i, edit := range edits {
		if !utf8.ValidString(edit.Text) {
			return ErrUTF8
		}
		if edit.Start > edit.End {
			return ErrRange
		}
		left, err := b.byteOffset(edit.Start)
		if err != nil {
			return err
		}
		right, err := b.byteOffset(edit.End)
		if err != nil {
			return err
		}
		if i > 0 {
			previous := edits[i-1]
			if previous.Start >= edit.Start || previous.End > edit.Start ||
				previous.End == edit.Start && (previous.Start == previous.End || edit.Start == edit.End) {
				return ErrEditOrder
			}
		}
		plan[i] = batchReplacement{edit, left, right}
	}
	// Subtract all removals before adding replacements to validate the final
	// byte count without signed deltas or an overflowing sum of insertions.
	finalBytes := b.bytes
	for _, edit := range plan {
		finalBytes -= edit.right - edit.left
	}
	for _, edit := range plan {
		var err error
		finalBytes, err = b.CheckReplaceSize(finalBytes, 0, len(edit.Text))
		if err != nil {
			return err
		}
	}
	// Preserve the existing conservative policy: even a final result that fits
	// is refused if a descending intermediate state would exceed the ceiling.
	current := b.bytes
	for i := len(plan) - 1; i >= 0; i-- {
		edit := plan[i]
		var err error
		current, err = b.CheckReplaceSize(current, edit.right-edit.left, len(edit.Text))
		if err != nil {
			return err
		}
	}
	// Walk untouched gaps and inserted lengths rather than accumulating signed
	// shifts. Every partial sum is <= the validated final UTF-8 byte count.
	carets := SelectionSet{Ranges: make([]Range, 0, len(plan))}
	previousEnd, outputEnd := 0, 0
	for i, edit := range plan {
		gap, inserted := edit.Start-previousEnd, UTF16Len(edit.Text)
		const maxInt = int(^uint(0) >> 1)
		if gap > maxInt-outputEnd || inserted > maxInt-outputEnd-gap {
			return ErrRange
		}
		outputEnd += gap + inserted
		caret := Range{outputEnd, outputEnd}
		if len(carets.Ranges) == 0 || carets.Ranges[len(carets.Ranges)-1] != caret {
			carets.Ranges = append(carets.Ranges, caret)
		}
		if i == primary {
			carets.Primary = len(carets.Ranges) - 1
		}
		previousEnd = edit.End
	}
	after, afterExtra := carets.Ranges[carets.Primary], extraOf(carets)
	if finalBytes == b.bytes && b.batchTextUnchanged(plan) {
		b.setSelections(after, afterExtra, false)
		return nil
	}
	// Admission is complete. Materialize removed bytes and the entire history
	// transaction before the first write. No fallible operation follows.
	t := transaction{before: b.selection, beforeSelections: cloneExtra(b.selections), after: after, afterSelections: afterExtra}
	t.edits = make([]change, 0, len(plan))
	for i := len(plan) - 1; i >= 0; i-- {
		edit := plan[i]
		old := b.sliceBytes(edit.left, edit.right)
		if old != edit.Text {
			t.edits = append(t.edits, change{start: edit.Start, old: old, next: edit.Text})
		}
	}
	t.recountBytes()
	for _, edit := range t.edits {
		b.replace(edit.start, edit.start+UTF16Len(edit.old), edit.next)
	}
	b.setSelections(after, afterExtra, false)
	b.commit(t)
	return nil
}

// ReplaceBatchWithSelectionUTF16 is the ordinary batch contract with explicit
// output selections. All endpoints are checked against the proposed output before
// normalization, including endpoints that normalization would absorb. The bool
// reports exact byte change; a byte-equivalent batch may update only selections
// while preserving text revision, journals, history and redo. Changing batches
// use ordinary commit/trim, including when no history capacity is configured.
// This method never settles or cancels an existing ordinary/composition owner.
func (b *Buffer) ReplaceBatchWithSelectionUTF16(version, selectionGeneration uint64, edits []Replacement, after SelectionSet) (bool, error) {
	if b.group != nil || b.composition != nil {
		return false, ErrGroup
	}
	if version != b.version || selectionGeneration != b.selectionGeneration {
		return false, ErrStaleEdit
	}
	plan, finalBytes, err := b.prepareOrdinaryBatch(edits)
	if err != nil {
		return false, err
	}
	normalized, err := b.normalizeBatchOutput(plan, after)
	if err != nil {
		return false, err
	}
	primary, extra := normalized.Ranges[normalized.Primary], extraOf(normalized)
	if finalBytes == b.bytes && b.batchTextUnchanged(plan) {
		// Do not replace even selection storage on an exact complete-state no-op.
		if b.selection != primary || !equalExtra(b.selections, extra) {
			b.setSelections(primary, extra, false)
		}
		return false, nil
	}
	t := transaction{before: b.selection, beforeSelections: cloneExtra(b.selections), after: primary, afterSelections: extra,
		edits: make([]change, 0, len(plan))}
	for i := len(plan) - 1; i >= 0; i-- {
		edit := plan[i]
		old := b.sliceBytes(edit.left, edit.right)
		if old != edit.Text {
			t.edits = append(t.edits, change{edit.Start, old, strings.Clone(edit.Text)})
		}
	}
	t.recountBytes()
	for _, edit := range t.edits {
		b.replace(edit.start, edit.start+UTF16Len(edit.old), edit.next)
	}
	b.setSelections(primary, extra, false)
	b.commit(t)
	return true, nil
}

// prepareOrdinaryBatch admits original-coordinate edits without materializing
// removed text. Keep this ordinary policy independent of strict patch admission.
func (b *Buffer) prepareOrdinaryBatch(edits []Replacement) ([]batchReplacement, int, error) {
	if len(edits) > MaxSelections {
		return nil, 0, ErrSelectionLimit
	}
	if len(edits) == 0 {
		return nil, 0, ErrSelectionSet
	}
	plan := make([]batchReplacement, len(edits))
	for i, edit := range edits {
		if !utf8.ValidString(edit.Text) {
			return nil, 0, ErrUTF8
		}
		if edit.Start > edit.End {
			return nil, 0, ErrRange
		}
		left, err := b.byteOffset(edit.Start)
		if err != nil {
			return nil, 0, err
		}
		right, err := b.byteOffset(edit.End)
		if err != nil {
			return nil, 0, err
		}
		if i > 0 {
			previous := edits[i-1]
			if previous.Start >= edit.Start || previous.End > edit.Start ||
				previous.End == edit.Start && (previous.Start == previous.End || edit.Start == edit.End) {
				return nil, 0, ErrEditOrder
			}
		}
		plan[i] = batchReplacement{edit, left, right}
	}
	finalBytes := b.bytes
	for _, edit := range plan {
		finalBytes -= edit.right - edit.left
	}
	for _, edit := range plan {
		var err error
		finalBytes, err = b.CheckReplaceSize(finalBytes, 0, len(edit.Text))
		if err != nil {
			return nil, 0, err
		}
	}
	current := b.bytes
	for i := len(plan) - 1; i >= 0; i-- {
		edit := plan[i]
		var err error
		current, err = b.CheckReplaceSize(current, edit.right-edit.left, len(edit.Text))
		if err != nil {
			return nil, 0, err
		}
	}
	return plan, finalBytes, nil
}

func (b *Buffer) normalizeBatchOutput(plan []batchReplacement, set SelectionSet) (SelectionSet, error) {
	var endpoints map[int]error
	return normalizeSelectionSet(set, func(offset int) error {
		if endpoints == nil {
			endpoints = ordinaryBatchEndpoints(plan, b.units, set, func(original int) error {
				_, err := b.byteOffset(original)
				return err
			})
		}
		return endpoints[offset]
	})
}

// ordinaryBatchEndpoints merge-walks at most 512 sorted requested endpoints and
// the at most 256 admitted edits. Untouched gaps map back to indexed source
// coordinates; they are neither copied nor scanned. Only inserted text is walked
// rune-by-rune, once per edit. sourceEndpoint is an indexed original-boundary
// validator, also allowing tests to check precisely which source offsets are read.
func ordinaryBatchEndpoints(plan []batchReplacement, sourceUnits int, set SelectionSet, sourceEndpoint func(int) error) map[int]error {
	positions := make([]int, 0, 2*len(set.Ranges))
	result := make(map[int]error, 2*len(set.Ranges))
	for _, r := range set.Ranges {
		positions = append(positions, r.Anchor, r.Head)
		result[r.Anchor], result[r.Head] = ErrRange, ErrRange
	}
	sort.Ints(positions)
	i := 0
	for i < len(positions) && positions[i] < 0 {
		i++
	}
	input, output := 0, 0
	gap := func(end int) bool {
		next, ok := ordinaryBatchOffset(output, end-input)
		if !ok {
			return false
		}
		for i < len(positions) && positions[i] <= next {
			p := positions[i]
			result[p] = sourceEndpoint(input + (p - output))
			i++
		}
		input, output = end, next
		return true
	}
	for _, edit := range plan {
		if !gap(edit.Start) {
			return result
		}
		for _, r := range edit.Text {
			width := 1
			if r > 0xffff {
				width = 2
			}
			next, ok := ordinaryBatchOffset(output, width)
			if !ok {
				return result
			}
			for i < len(positions) && positions[i] <= next {
				p := positions[i]
				result[p] = nil
				if p != next {
					result[p] = ErrSurrogate
				}
				i++
			}
			output = next
		}
		input = edit.End
	}
	gap(sourceUnits)
	return result
}

func ordinaryBatchOffset(base, count int) (int, bool) {
	const maxInt = int(^uint(0) >> 1)
	if base < 0 || count < 0 || count > maxInt-base {
		return 0, false
	}
	return base + count, true
}

// Compare the proposed output stream against current immutable pieces. Gaps
// need no allocation, including when earlier edits shifted their byte offset.
func (b *Buffer) batchTextUnchanged(plan []batchReplacement) bool {
	input, output := 0, 0
	for _, edit := range plan {
		gap := edit.left - input
		if !b.equalByteSpans(input, output, gap) || !b.equalBytesAt(output+gap, edit.Text) {
			return false
		}
		output += gap + len(edit.Text)
		input = edit.right
	}
	return b.equalByteSpans(input, output, b.bytes-input)
}

func (b *Buffer) pieceAtByte(offset int) int {
	return sort.Search(len(b.pieces), func(i int) bool { return b.prefix[i+1].bytes > offset })
}

func (b *Buffer) equalBytesAt(offset int, value string) bool {
	if offset < 0 || offset > b.bytes || len(value) > b.bytes-offset {
		return false
	}
	i := b.pieceAtByte(offset)
	for len(value) > 0 {
		p := b.pieces[i]
		start := p.start + offset - b.prefix[i].bytes
		n := min(p.end-start, len(value))
		if p.data.value[start:start+n] != value[:n] {
			return false
		}
		offset += n
		value = value[n:]
		i++
	}
	return true
}

func (b *Buffer) equalByteSpans(a, z, count int) bool {
	if a == z || count == 0 {
		return true
	}
	if a < 0 || z < 0 || a > b.bytes || z > b.bytes || count > b.bytes-a || count > b.bytes-z {
		return false
	}
	i, j := b.pieceAtByte(a), b.pieceAtByte(z)
	for count > 0 {
		p, q := b.pieces[i], b.pieces[j]
		left, right := p.start+a-b.prefix[i].bytes, q.start+z-b.prefix[j].bytes
		n := min(count, min(p.end-left, q.end-right))
		if p.data.value[left:left+n] != q.data.value[right:right+n] {
			return false
		}
		a, z, count = a+n, z+n, count-n
		if left+n == p.end {
			i++
		}
		if right+n == q.end {
			j++
		}
	}
	return true
}
