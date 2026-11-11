package text

// rangeChange stores coordinates only. It neither extends undo history nor
// retains source text. Undo, redo and canceled groups pass through replace;
// Restore records a reset barrier, even when restoring identical bytes.
type rangeChange struct {
	start, end, inserted int
	reset                bool
}

// MapRangeSince follows an untouched UTF-16 range through the last 64 edits.
// Insertions at its start are before it (both ends move); insertions at its end
// are after it (neither end moves). An insertion into an empty range or inside
// a nonempty range, any overlapping replacement, Restore, an expired journal,
// or invalid current boundaries returns false. Direction is preserved.
// Callers must also check Buffer identity and may compare expected field text.
func (b *Buffer) MapRangeSince(version uint64, span Range) (Range, bool) {
	if version > b.version || b.version-version > uint64(len(b.rangeChanges)) || span.Start() < 0 {
		return Range{}, false
	}
	start, end := span.Start(), span.End()
	for revision := version + 1; revision <= b.version; revision++ {
		change := b.rangeChanges[revision%uint64(len(b.rangeChanges))]
		if change.reset {
			return Range{}, false
		}
		if change.start == change.end && start == end && change.start == start {
			return Range{}, false
		}
		if change.end <= start {
			delta := change.inserted - (change.end - change.start)
			start += delta
			end += delta
		} else if change.start < end {
			return Range{}, false
		}
	}
	if _, err := b.byteOffset(start); err != nil {
		return Range{}, false
	}
	if _, err := b.byteOffset(end); err != nil {
		return Range{}, false
	}
	if span.Anchor > span.Head {
		return Range{end, start}, true
	}
	return Range{start, end}, true
}
