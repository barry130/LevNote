package text

import (
	"errors"
	"sort"
	"strings"
	"unicode/utf8"
)

// MaxPatchEdits bounds patch work independently of the complete selection set.
const MaxPatchEdits = 1024

var (
	ErrPatchLimit        = errors.New("too many patch edits")
	ErrPatchOldText      = errors.New("patch old text does not match the original range")
	ErrPatchHistoryLimit = errors.New("patch exceeds the capacity for one complete undo record")
)

// PatchEdit replaces exact OldText at [Start, End) in the original UTF-16
// coordinate space. Ranges follow Replacement's ordering and no-touch rules.
type PatchEdit struct {
	Start, End       int
	OldText, NewText string
}

// ApplyPatchUTF16 applies the complete patch and normalized afterSelections as
// one undoable transaction. All predictable refusals are pure, including foreign
// ordinary/composition ownership and invalid output endpoints. Empty and final
// byte-equivalent patches validate their inputs but preserve every Buffer field,
// the original complete selections and redo. A changing success retains one
// complete undo item; older history may be trimmed to admit it.
func (b *Buffer) ApplyPatchUTF16(version, selectionGeneration uint64, edits []PatchEdit, afterSelections SelectionSet) (bool, error) {
	if b.group != nil || b.composition != nil {
		return false, ErrGroup
	}
	if version != b.version || selectionGeneration != b.selectionGeneration {
		return false, ErrStaleEdit
	}
	if len(edits) > MaxPatchEdits {
		return false, ErrPatchLimit
	}
	plan := make([]batchReplacement, len(edits))
	for i, edit := range edits {
		if !utf8.ValidString(edit.OldText) || !utf8.ValidString(edit.NewText) {
			return false, ErrUTF8
		}
		if edit.Start > edit.End {
			return false, ErrRange
		}
		left, err := b.byteOffset(edit.Start)
		if err != nil {
			return false, err
		}
		right, err := b.byteOffset(edit.End)
		if err != nil {
			return false, err
		}
		if i > 0 {
			previous := edits[i-1]
			if previous.Start >= edit.Start || previous.End > edit.Start ||
				previous.End == edit.Start && (previous.Start == previous.End || edit.Start == edit.End) {
				return false, ErrEditOrder
			}
		}
		if len(edit.OldText) != right-left || !b.equalBytesAt(left, edit.OldText) {
			return false, ErrPatchOldText
		}
		plan[i] = batchReplacement{Replacement{edit.Start, edit.End, edit.NewText}, left, right}
	}
	// Subtract before adding, then separately admit every descending write.
	// CheckReplaceSize bounds every addition without constructing signed deltas.
	finalBytes := b.bytes
	for _, edit := range plan {
		finalBytes -= edit.right - edit.left
	}
	for _, edit := range plan {
		var err error
		finalBytes, err = b.CheckReplaceSize(finalBytes, 0, len(edit.Text))
		if err != nil {
			return false, err
		}
	}
	current := b.bytes
	for i := len(plan) - 1; i >= 0; i-- {
		edit := plan[i]
		var err error
		current, err = b.CheckReplaceSize(current, edit.right-edit.left, len(edit.Text))
		if err != nil {
			return false, err
		}
	}
	// The normalizer first checks cardinality/primary, then calls the supplied
	// validator on EVERY original endpoint, even ones a merge would absorb.
	var endpoints map[int]error
	after, err := normalizeSelectionSet(afterSelections, func(offset int) error {
		if endpoints == nil {
			endpoints = b.patchOutputEndpoints(plan, afterSelections)
		}
		return endpoints[offset]
	})
	if err != nil {
		return false, err
	}
	if finalBytes == b.bytes && b.batchTextUnchanged(plan) {
		return false, nil
	}
	// Fully own the text and complete history endpoints before the first write.
	t := transaction{before: b.selection, beforeSelections: cloneExtra(b.selections),
		after: after.Ranges[after.Primary], afterSelections: extraOf(after),
		edits: make([]change, 0, len(plan))}
	for i := len(plan) - 1; i >= 0; i-- {
		edit := plan[i]
		old := b.sliceBytes(edit.left, edit.right)
		if old != edit.Text {
			t.edits = append(t.edits, change{edit.Start, old, strings.Clone(edit.Text)})
		}
	}
	t.recountBytes()
	if !b.admitsStandaloneHistory(t) {
		return false, ErrPatchHistoryLimit
	}
	// Admission is complete: the writes, selection publication and commit have
	// no callbacks or error exits. Undo reverses this same admitted sequence.
	for _, edit := range t.edits {
		b.replace(edit.start, edit.start+UTF16Len(edit.old), edit.next)
	}
	b.setSelections(t.after, t.afterSelections, false)
	b.commit(t)
	return true, nil
}

// Validate at most 512 requested endpoints in one forward virtual-output scan.
// Immutable piece fragments and inserted strings are never joined into a second
// document. Every stream fragment ends on a rune boundary. The byte admission
// above bounds its UTF-16 count too; additions are checked explicitly here.
func (b *Buffer) patchOutputEndpoints(plan []batchReplacement, set SelectionSet) map[int]error {
	positions := make([]int, 0, 2*len(set.Ranges))
	result := make(map[int]error, 2*len(set.Ranges))
	for _, r := range set.Ranges {
		positions = append(positions, r.Anchor, r.Head)
		result[r.Anchor], result[r.Head] = ErrRange, ErrRange
	}
	sort.Ints(positions)
	i := 0
	for i < len(positions) && positions[i] <= 0 {
		if positions[i] == 0 {
			result[0] = nil
		}
		i++
	}
	stream := compositionStream{buffer: b, plan: plan}
	units := 0
	for i < len(positions) {
		fragment := stream.next()
		if fragment == "" {
			break
		}
		for _, r := range fragment {
			width := 1
			if r > 0xffff {
				width = 2
			}
			const maxInt = int(^uint(0) >> 1)
			if width > maxInt-units {
				return result
			}
			end := units + width
			for i < len(positions) && positions[i] <= end {
				if positions[i] == end {
					result[positions[i]] = nil
				} else {
					result[positions[i]] = ErrSurrogate
				}
				i++
			}
			units = end
			if i == len(positions) {
				break
			}
		}
	}
	return result
}
