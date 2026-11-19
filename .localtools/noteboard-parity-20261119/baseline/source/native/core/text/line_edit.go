package text

// LineEditAction edits logical lines, independently of visual wrapping.
type LineEditAction uint8

const (
	MoveLinesUp LineEditAction = iota
	MoveLinesDown
	CopyLinesUp
	CopyLinesDown
)

// LineEditPlan is a read-only plan for the current buffer and selection. It is
// valid only until either changes. Changed is false for a boundary no-op or an
// exchange of identical text; the latter can still move Selection.
type LineEditPlan struct {
	Start, End int
	Text       string
	Selection  Range
	Changed    bool
}

// PlanLineEdit keeps the selected block's internal separators verbatim. A move
// swaps that block with one adjacent line using the existing joining separator.
// A copy uses the block's last separator, the preceding one at EOF, or LF for a
// single-line document. No operation normalizes the rest of the document.
// Capacity is checked before copying source text and without committing an IME
// group. Callers own read-only checks, composition settlement and undo grouping.
func (b *Buffer) PlanLineEdit(action LineEditAction) (LineEditPlan, error) {
	plan := LineEditPlan{Selection: b.selection}
	if action > CopyLinesDown {
		return plan, ErrRange
	}
	first, _, err := b.LineColumn(b.selection.Start())
	if err != nil {
		return plan, err
	}
	last, column, err := b.LineColumn(b.selection.End())
	if err != nil {
		return plan, err
	}
	if first != last && column == 0 {
		last-- // the endpoint at the next line's start does not select that line
	}
	if action == MoveLinesUp && first == 0 || action == MoveLinesDown && last == b.LineCount()-1 {
		return plan, nil
	}
	start, err := b.OffsetOfLine(first)
	if err != nil {
		return plan, err
	}
	lastSpan, err := b.LineRange(last)
	if err != nil {
		return plan, err
	}
	end := lastSpan.End()
	tail, err := b.lineEditTail(last)
	if err != nil {
		return plan, err
	}
	newStart, newTail := start, tail-end
	plan.Start, plan.End = start, end

	if action == CopyLinesUp || action == CopyLinesDown {
		separator, err := b.lineEditSeparator(last, end, tail)
		if err != nil {
			return plan, err
		}
		left, err := b.byteOffset(start)
		if err != nil {
			return plan, err
		}
		right, err := b.byteOffset(end)
		if err != nil {
			return plan, err
		}
		// Two checked additions avoid overflow and allocate nothing on rejection.
		size, err := b.CheckReplaceSize(b.bytes, 0, right-left)
		if err != nil {
			return plan, err
		}
		if _, err = b.CheckReplaceSize(size, 0, len(separator)); err != nil {
			return plan, err
		}
		block := b.sliceBytes(left, right)
		plan.Start, plan.End = start, start
		plan.Text = block + separator
		newTail = UTF16Len(separator)
		if action == CopyLinesDown {
			plan.Start, plan.End = tail, tail
			newStart = tail
			if tail == end {
				plan.Text = separator + block
				newStart += UTF16Len(separator)
				newTail = 0
			}
		}
		plan.Changed = true
	} else {
		neighborLine := first - 1
		if action == MoveLinesDown {
			neighborLine = last + 1
		}
		neighbor, err := b.LineRange(neighborLine)
		if err != nil {
			return plan, err
		}
		gapStart, gapEnd := neighbor.End(), start
		plan.Start = neighbor.Start()
		newStart = neighbor.Start()
		if action == MoveLinesDown {
			gapStart, gapEnd = end, neighbor.Start()
			plan.Start, plan.End = start, neighbor.End()
			newStart = start + neighbor.End() - neighbor.Start() + gapEnd - gapStart
			neighborTail, err := b.lineEditTail(neighborLine)
			if err != nil {
				return plan, err
			}
			newTail = neighborTail - neighbor.End()
		} else {
			newTail = gapEnd - gapStart
		}
		// Reordering has exactly the same byte size. Apply the buffer's admission
		// contract before materializing even an exceptionally long logical line.
		left, err := b.byteOffset(plan.Start)
		if err != nil {
			return plan, err
		}
		right, err := b.byteOffset(plan.End)
		if err != nil {
			return plan, err
		}
		if _, err = b.CheckReplaceSize(b.bytes, right-left, right-left); err != nil {
			return plan, err
		}
		block, err := b.SliceUTF16(start, end)
		if err != nil {
			return plan, err
		}
		other, err := b.SliceUTF16(neighbor.Start(), neighbor.End())
		if err != nil {
			return plan, err
		}
		separator, err := b.SliceUTF16(gapStart, gapEnd)
		if err != nil {
			return plan, err
		}
		plan.Text = block + separator + other
		if action == MoveLinesDown {
			plan.Text = other + separator + block
		}
		plan.Changed = plan.Text != b.sliceBytes(left, right)
	}

	mapOffset := func(offset int) int {
		if offset <= end {
			return newStart + offset - start
		}
		// A full trailing separator maps to the full new separator (or EOF).
		// This also keeps CRLF interior endpoints out of following surrogate pairs.
		extra := min(offset-end, newTail)
		if offset == tail {
			extra = newTail
		}
		return newStart + end - start + extra
	}
	plan.Selection = Range{Anchor: mapOffset(b.selection.Anchor), Head: mapOffset(b.selection.Head)}
	return plan, nil
}

func (b *Buffer) lineEditTail(line int) (int, error) {
	if line+1 < b.LineCount() {
		return b.OffsetOfLine(line + 1)
	}
	return b.LenUTF16(), nil
}

func (b *Buffer) lineEditSeparator(line, end, tail int) (string, error) {
	if tail > end {
		return b.SliceUTF16(end, tail)
	}
	if line > 0 {
		previous, err := b.LineRange(line - 1)
		if err != nil {
			return "", err
		}
		start, err := b.OffsetOfLine(line)
		if err != nil {
			return "", err
		}
		return b.SliceUTF16(previous.End(), start)
	}
	return "\n", nil
}
