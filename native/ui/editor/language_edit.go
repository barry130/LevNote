package editor

import (
	"errors"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

// SourceSelectionPolicy specifies the complete selection state stored in the
// patch's history item. It carries no language-service operation semantics.
type SourceSelectionPolicy uint8

const (
	PreserveMappedSelections SourceSelectionPolicy = iota
	SingleInsertionCaret
)

var (
	ErrSourceSelectionPolicy = errors.New("invalid source selection policy or patch cardinality")
	ErrSourceReadOnly        = errors.New("source editor is read-only")
)

type sourcePatchBoundary struct {
	start, end int
}

// ApplySourceEdits revalidates the full opaque target and applies exact edits
// with their complete after-selection state in one transaction. Admission never
// observes/repairs input ownership, settles composition or invokes host code.
// On a changing success, OnChange is the final host callback: synchronous host
// attachment replacement or nested edits retain their exact callback exit state.
func (e *Editor) ApplySourceEdits(target SourceTarget, edits []text.PatchEdit, policy SourceSelectionPolicy) (bool, error) {
	if err := e.CheckSourceTarget(target, true); err != nil {
		return false, err
	}
	if e.ReadOnly {
		return false, ErrSourceReadOnly
	}
	if policy != PreserveMappedSelections && policy != SingleInsertionCaret || policy == SingleInsertionCaret && len(edits) != 1 {
		return false, ErrSourceSelectionPolicy
	}
	if len(edits) > text.MaxPatchEdits {
		return false, text.ErrPatchLimit
	}
	// Check bounds/order before mapping, and add only nonnegative gaps/lengths.
	// Core remains the authority for exact old text, surrogate endpoints, byte
	// budgets, proposed-output normalization and standalone history admission.
	boundaries := make([]sourcePatchBoundary, len(edits))
	previous, output := 0, 0
	for i, edit := range edits {
		if !utf8.ValidString(edit.OldText) || !utf8.ValidString(edit.NewText) {
			return false, text.ErrUTF8
		}
		if edit.Start < 0 || edit.Start > edit.End || edit.End > e.Buffer.LenUTF16() {
			return false, text.ErrRange
		}
		if i > 0 {
			prior := edits[i-1]
			if prior.Start >= edit.Start || prior.End > edit.Start ||
				prior.End == edit.Start && (prior.Start == prior.End || edit.Start == edit.End) {
				return false, text.ErrEditOrder
			}
		}
		start, err := sourcePatchOffset(output, edit.Start, previous)
		if err != nil {
			return false, err
		}
		end, err := sourcePatchOffset(start, text.UTF16Len(edit.NewText), 0)
		if err != nil {
			return false, err
		}
		boundaries[i] = sourcePatchBoundary{start, end}
		previous, output = edit.End, end
	}
	var after text.SelectionSet
	if policy == SingleInsertionCaret {
		end := boundaries[0].end
		after = text.SelectionSet{Ranges: []text.Range{{Anchor: end, Head: end}}}
	} else {
		after = cloneSourceSelectionSet(target.selections)
		for i, r := range after.Ranges {
			anchor, err := mapSourcePatchEndpoint(r.Anchor, edits, boundaries)
			if err != nil {
				return false, err
			}
			head, err := mapSourcePatchEndpoint(r.Head, edits, boundaries)
			if err != nil {
				return false, err
			}
			after.Ranges[i] = text.Range{Anchor: anchor, Head: head}
		}
	}
	a := e.captureInputAction()
	changed, err := e.Buffer.ApplyPatchUTF16(target.stamp.version, target.stamp.selectionGen, edits, after)
	if err != nil || !changed {
		return changed, err
	}
	// These helpers invalidate stale shapes/folds without geometry queries,
	// highlighting, error reporting or fold analysis. Ordinary drawing reveals
	// the admitted caret later. Schedule invalidation before notifying the host.
	e.invalidateCacheIfChanged()
	e.hasDesiredX = false
	e.affinity = ui.TextDownstream
	e.blinkStart = e.frameNow
	e.revealCaret = true
	e.services.Invalidate()
	a.changed = true
	a.finish()
	return true, nil
}

func sourcePatchOffset(base, offset, oldBoundary int) (int, error) {
	const maxInt = int(^uint(0) >> 1)
	if base < 0 || oldBoundary < 0 || offset < oldBoundary || offset-oldBoundary > maxInt-base {
		return 0, text.ErrRange
	}
	return base + (offset - oldBoundary), nil
}

func mapSourcePatchEndpoint(offset int, edits []text.PatchEdit, boundaries []sourcePatchBoundary) (int, error) {
	previous, output := 0, 0
	for i, edit := range edits {
		boundary := boundaries[i]
		if offset < edit.Start {
			return sourcePatchOffset(output, offset, previous)
		}
		if offset == edit.Start && edit.Start != edit.End {
			return boundary.start, nil
		}
		if offset <= edit.End {
			// Replacement interiors/end go to the new end; a coincident
			// zero-width insertion has explicit right affinity as well.
			return boundary.end, nil
		}
		previous, output = edit.End, boundary.end
	}
	return sourcePatchOffset(output, offset, previous)
}
