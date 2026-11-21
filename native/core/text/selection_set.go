package text

import (
	"errors"
	"sort"
	"strconv"
)

// MaxSelections bounds selection normalization, batch edits and retained ranges.
const MaxSelections = 256

var (
	ErrSelectionLimit = errors.New("too many text selections")
	ErrSelectionSet   = errors.New("invalid text selection set or primary index")
)

// SelectionSet owns its Ranges. Buffer and Snapshot accessors return independent
// copies, in document order, with the primary range identified by its index.
type SelectionSet struct {
	Ranges  []Range
	Primary int
}

// Only multi-selection states allocate a secondary slice. The primary remains
// inline so legacy Selection/SetSelection and single-selection history allocate
// no selection slices. secondary is in document order, omitting the primary.
type selectionExtra struct {
	secondary []Range
	primary   int
}

func cloneExtra(s selectionExtra) selectionExtra {
	if len(s.secondary) == 0 {
		return selectionExtra{}
	}
	out := make([]Range, len(s.secondary))
	copy(out, s.secondary)
	return selectionExtra{secondary: out, primary: s.primary}
}

func selectionSet(primary Range, extra selectionExtra) SelectionSet {
	out := SelectionSet{Ranges: make([]Range, len(extra.secondary)+1), Primary: extra.primary}
	copy(out.Ranges, extra.secondary[:extra.primary])
	out.Ranges[extra.primary] = primary
	copy(out.Ranges[extra.primary+1:], extra.secondary[extra.primary:])
	return out
}

func extraOf(set SelectionSet) selectionExtra {
	if len(set.Ranges) == 1 {
		return selectionExtra{}
	}
	extra := selectionExtra{secondary: make([]Range, len(set.Ranges)-1), primary: set.Primary}
	copy(extra.secondary, set.Ranges[:set.Primary])
	copy(extra.secondary[set.Primary:], set.Ranges[set.Primary+1:])
	return extra
}

func equalExtra(a, b selectionExtra) bool {
	if a.primary != b.primary || len(a.secondary) != len(b.secondary) {
		return false
	}
	for i, r := range a.secondary {
		if r != b.secondary[i] {
			return false
		}
	}
	return true
}

func (b *Buffer) setSelections(primary Range, extra selectionExtra, invalidate bool) {
	if invalidate || b.selection != primary || !equalExtra(b.selections, extra) {
		b.selectionGeneration++
	}
	b.selection, b.selections = primary, cloneExtra(extra)
}

// SelectionSet returns a deep copy. Selection remains the allocation-free way
// for legacy callers to read only the primary range.
func (b *Buffer) SelectionSet() SelectionSet  { return selectionSet(b.selection, b.selections) }
func (s Snapshot) SelectionSet() SelectionSet { return selectionSet(s.selection, s.selections) }

// SelectionGeneration changes independently of the text revision. Undo, Redo,
// CancelGroup and Restore invalidate earlier selection identities, even when
// the restored coordinates happen to equal the current ones.
func (b *Buffer) SelectionGeneration() uint64 { return b.selectionGeneration }

// SetSelectionSet validates all input before changing anything. Ordinary open
// groups may set their eventual selections, but an IME owner must settle its
// own composition before using this tokenless multi-selection API.
func (b *Buffer) SetSelectionSet(set SelectionSet) error {
	if b.composition != nil {
		return ErrGroup
	}
	normalized, err := b.NormalizeSelectionSet(set)
	if err != nil {
		return err
	}
	b.setSelections(normalized.Ranges[normalized.Primary], extraOf(normalized), false)
	return nil
}

type selectionComponent struct {
	span    Range
	primary bool
}

// NormalizeSelectionSet is read-only. Nonempty intervals merge only on true
// overlap. Their primary member supplies direction; absent one, the first
// nonempty member in stable start order supplies it. Carets on or inside a
// range are absorbed. At a shared boundary the right-hand range wins; an
// absorbed primary caret does not reverse its containing nonempty range.
func (b *Buffer) NormalizeSelectionSet(set SelectionSet) (SelectionSet, error) {
	return normalizeSelectionSet(set, func(offset int) error {
		_, err := b.byteOffset(offset)
		return err
	})
}

// normalizeSelectionSet shares stable direction/primary/collision semantics with
// transactions whose endpoints belong to a proposed output rather than b.
func normalizeSelectionSet(set SelectionSet, validateEndpoint func(int) error) (SelectionSet, error) {
	if len(set.Ranges) > MaxSelections {
		return SelectionSet{}, ErrSelectionLimit
	}
	if len(set.Ranges) == 0 || set.Primary < 0 || set.Primary >= len(set.Ranges) {
		return SelectionSet{}, ErrSelectionSet
	}
	for _, r := range set.Ranges {
		if err := validateEndpoint(r.Anchor); err != nil {
			return SelectionSet{}, err
		}
		if err := validateEndpoint(r.Head); err != nil {
			return SelectionSet{}, err
		}
	}
	parts := make([]selectionComponent, 0, len(set.Ranges))
	for i, r := range set.Ranges {
		if r.Start() != r.End() {
			parts = append(parts, selectionComponent{r, i == set.Primary})
		}
	}
	sort.SliceStable(parts, func(i, j int) bool { return parts[i].span.Start() < parts[j].span.Start() })
	merged := parts[:0]
	for _, part := range parts {
		if len(merged) == 0 || part.span.Start() >= merged[len(merged)-1].span.End() {
			merged = append(merged, part)
			continue
		}
		last := &merged[len(merged)-1]
		start, end := last.span.Start(), max(last.span.End(), part.span.End())
		backward := last.span.Anchor > last.span.Head
		if part.primary {
			backward = part.span.Anchor > part.span.Head
		}
		last.span = Range{start, end}
		if backward {
			last.span = Range{end, start}
		}
		last.primary = last.primary || part.primary
	}
	// Keep the nonempty prefix fixed while assigning carets, so absorbing a
	// boundary caret cannot accidentally bridge two adjacent nonempty ranges.
	nonempty := len(merged)
	for i, r := range set.Ranges {
		if r.Start() != r.End() {
			continue
		}
		p := r.Head
		at := sort.Search(nonempty, func(j int) bool { return merged[j].span.End() >= p })
		if at < nonempty && merged[at].span.End() == p && at+1 < nonempty && merged[at+1].span.Start() == p {
			at++
		}
		if at < nonempty && merged[at].span.Start() <= p {
			merged[at].primary = merged[at].primary || i == set.Primary
			continue
		}
		merged = append(merged, selectionComponent{r, i == set.Primary})
	}
	sort.SliceStable(merged, func(i, j int) bool { return merged[i].span.Start() < merged[j].span.Start() })
	out := SelectionSet{Ranges: make([]Range, 0, len(merged))}
	for _, part := range merged {
		if len(out.Ranges) == 0 || out.Ranges[len(out.Ranges)-1] != part.span {
			out.Ranges = append(out.Ranges, part.span)
		}
		if part.primary {
			out.Primary = len(out.Ranges) - 1
		}
	}
	return out, nil
}

func (t *transaction) selectionBytes() int {
	// cloneExtra always allocates exact lengths, so this counts every retained
	// secondary range while preserving historical single-selection accounting.
	return (cap(t.beforeSelections.secondary) + cap(t.afterSelections.secondary)) * (strconv.IntSize / 4)
}
