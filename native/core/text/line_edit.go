package text

import (
	"sort"
	"strings"
)

// LineEditAction edits logical lines, independently of visual wrapping.
type LineEditAction uint8

const (
	MoveLinesUp LineEditAction = iota
	MoveLinesDown
	CopyLinesUp
	CopyLinesDown
	IndentLines
	OutdentLines
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
	return b.planLineEditBlock(action, b.selection, first, last)
}

// The explicit line interval also represents a final empty logical line, which
// cannot be encoded by a nonempty half-open range ending at that line's start.
func (b *Buffer) planLineEditBlock(action LineEditAction, selection Range, first, last int) (LineEditPlan, error) {
	plan := LineEditPlan{Selection: selection}
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
			newStart = start + (neighbor.End() - neighbor.Start()) + (gapEnd - gapStart)
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
		return newStart + (end - start) + extra
	}
	plan.Selection = Range{Anchor: mapOffset(selection.Anchor), Head: mapOffset(selection.Head)}
	return plan, nil
}

// LineEditPlanSet describes one ordinary transaction. Edits use original source
// coordinates; After uses final output coordinates. Changed reports byte change,
// so an identical-text move can still have a distinct After. No Edits means an
// exact whole-action edge no-op; callers must not publish a new selection then.
// Apply nonempty Edits with ReplaceBatchWithSelectionUTF16 at the captured text
// and selection generations. Planning itself never settles or changes an owner.
type LineEditPlanSet struct {
	Edits   []Replacement
	After   SelectionSet
	Changed bool
}

type lineEditSetBlock struct {
	first, last      int
	from, to         int
	start, end, tail int
}

// PlanLineEditSet unions overlapping/adjacent selected logical line blocks.
// Moves swap each block with one neighbor; if any block is at the requested
// edge the entire action is an exact no-op. Copies select the inserted duplicate.
// Indent inserts four spaces per line; outdent removes one tab or up to four
// spaces. Separators and all unedited source bytes remain verbatim. At most one
// replacement is produced per merged block, not per selected logical line.
func (b *Buffer) PlanLineEditSet(action LineEditAction, set SelectionSet) (LineEditPlanSet, error) {
	var result LineEditPlanSet
	if action > OutdentLines {
		return result, ErrRange
	}
	before, err := b.NormalizeSelectionSet(set)
	if err != nil {
		return result, err
	}
	result.After = before
	blocks := make([]lineEditSetBlock, 0, len(before.Ranges))
	for i, r := range before.Ranges {
		first, _, err := b.LineColumn(r.Start())
		if err != nil {
			return LineEditPlanSet{}, err
		}
		last, column, err := b.LineColumn(r.End())
		if err != nil {
			return LineEditPlanSet{}, err
		}
		if first != last && column == 0 {
			last--
		}
		if len(blocks) != 0 && first-blocks[len(blocks)-1].last <= 1 {
			block := &blocks[len(blocks)-1]
			block.last, block.to = max(block.last, last), i+1
		} else {
			blocks = append(blocks, lineEditSetBlock{first: first, last: last, from: i, to: i + 1})
		}
	}
	for _, block := range blocks {
		if action == MoveLinesUp && block.first == 0 || action == MoveLinesDown && block.last == b.LineCount()-1 {
			return result, nil
		}
	}
	// Admit aggregate copy/indent growth before materializing selected blocks.
	// Moves/outdent cannot grow. This is content admission, not a new history cap.
	currentBytes := b.bytes
	for i := range blocks {
		block := &blocks[i]
		block.start, err = b.OffsetOfLine(block.first)
		if err != nil {
			return LineEditPlanSet{}, err
		}
		lastRange, err := b.LineRange(block.last)
		if err != nil {
			return LineEditPlanSet{}, err
		}
		block.end = lastRange.End()
		block.tail, err = b.lineEditTail(block.last)
		if err != nil {
			return LineEditPlanSet{}, err
		}
		if action == CopyLinesUp || action == CopyLinesDown {
			left, err := b.byteOffset(block.start)
			if err != nil {
				return LineEditPlanSet{}, err
			}
			right, err := b.byteOffset(block.end)
			if err != nil {
				return LineEditPlanSet{}, err
			}
			separator, err := b.lineEditSeparator(block.last, block.end, block.tail)
			if err != nil {
				return LineEditPlanSet{}, err
			}
			currentBytes, err = b.CheckReplaceSize(currentBytes, 0, right-left)
			if err == nil {
				currentBytes, err = b.CheckReplaceSize(currentBytes, 0, len(separator))
			}
			if err != nil {
				return LineEditPlanSet{}, err
			}
		} else if action == IndentLines {
			lines := block.last - block.first + 1
			if lines > (b.maxBytes-currentBytes)/4 {
				return LineEditPlanSet{}, &SizeLimitError{Limit: b.maxBytes}
			}
			currentBytes += 4 * lines
		}
	}
	after := SelectionSet{Ranges: make([]Range, len(before.Ranges)), Primary: before.Primary}
	result.Edits = make([]Replacement, 0, len(blocks))
	inputBase, outputBase := 0, 0
	for _, block := range blocks {
		var edit Replacement
		local := make([]Range, block.to-block.from)
		if action == IndentLines || action == OutdentLines {
			edit, local, err = b.planLineIndentBlock(action, block, before.Ranges[block.from:block.to])
			if err != nil {
				return LineEditPlanSet{}, err
			}
		} else {
			plan, err := b.planLineEditBlock(action, Range{block.start, block.tail}, block.first, block.last)
			if err != nil {
				return LineEditPlanSet{}, err
			}
			edit = Replacement{plan.Start, plan.End, plan.Text}
			newStart := plan.Selection.Anchor
			newTail := plan.Selection.Head - newStart - (block.end - block.start)
			mapOffset := func(offset int) int {
				if offset <= block.end {
					return newStart + (offset - block.start)
				}
				extra := min(offset-block.end, newTail)
				if offset == block.tail {
					extra = newTail
				}
				return newStart + (block.end - block.start) + extra
			}
			for i, r := range before.Ranges[block.from:block.to] {
				local[i] = Range{mapOffset(r.Anchor), mapOffset(r.Head)}
			}
		}
		outputStart, ok := ordinaryBatchOffset(outputBase, edit.Start-inputBase)
		if !ok {
			return LineEditPlanSet{}, ErrRange
		}
		for i, r := range local {
			anchor, aok := ordinaryBatchOffset(outputStart, r.Anchor-edit.Start)
			head, hok := ordinaryBatchOffset(outputStart, r.Head-edit.Start)
			if !aok || !hok {
				return LineEditPlanSet{}, ErrRange
			}
			after.Ranges[block.from+i] = Range{anchor, head}
		}
		outputBase, ok = ordinaryBatchOffset(outputStart, UTF16Len(edit.Text))
		if !ok {
			return LineEditPlanSet{}, ErrRange
		}
		inputBase = edit.End
		result.Edits = append(result.Edits, edit)
	}
	plan, finalBytes, err := b.prepareOrdinaryBatch(result.Edits)
	if err != nil {
		return LineEditPlanSet{}, err
	}
	result.After, err = b.normalizeBatchOutput(plan, after)
	if err != nil {
		return LineEditPlanSet{}, err
	}
	result.Changed = finalBytes != b.bytes || !b.batchTextUnchanged(plan)
	return result, nil
}

type lineIndentEndpoint struct {
	offset, index int
	anchor, left  bool
}

// This single forward block scan constructs edited text and maps at most 512
// endpoints. It keeps no per-line slice/map and never copies an unrelated gap.
func (b *Buffer) planLineIndentBlock(action LineEditAction, block lineEditSetBlock, ranges []Range) (Replacement, []Range, error) {
	value, err := b.SliceUTF16(block.start, block.end)
	if err != nil {
		return Replacement{}, nil, err
	}
	points := make([]lineIndentEndpoint, 0, 2*len(ranges))
	for i, r := range ranges {
		points = append(points, lineIndentEndpoint{r.Anchor, i, true, r.Anchor == r.End() && r.Anchor != r.Head},
			lineIndentEndpoint{r.Head, i, false, r.Head == r.End() && r.Anchor != r.Head})
	}
	sort.SliceStable(points, func(i, j int) bool { return points[i].offset < points[j].offset })
	after := make([]Range, len(ranges))
	writePoint := func(p lineIndentEndpoint, offset int) {
		if p.anchor {
			after[p.index].Anchor = offset
		} else {
			after[p.index].Head = offset
		}
	}
	var out strings.Builder
	at, inputUnits, outputUnits, point := 0, block.start, block.start, 0
	for {
		lineEnd := len(value)
		if newline := strings.IndexByte(value[at:], '\n'); newline >= 0 {
			lineEnd = at + newline + 1
		}
		line := value[at:lineEnd]
		removed, added := 0, 0
		if action == IndentLines {
			out.WriteString("    ")
			added = 4
		} else if len(line) > 0 && line[0] == '\t' {
			removed = 1
		} else {
			for removed < min(4, len(line)) && line[removed] == ' ' {
				removed++
			}
		}
		out.WriteString(line[removed:])
		units := UTF16Len(line)
		nextInput, nextOutput := inputUnits+units, outputUnits+added+units-removed
		last := lineEnd == len(value) && (len(line) == 0 || line[len(line)-1] != '\n')
		for point < len(points) && (points[point].offset < nextInput || last && points[point].offset <= nextInput) {
			p := points[point]
			mapped := outputUnits + added + max(0, p.offset-inputUnits-removed)
			// A nonempty high endpoint at a following line's start excludes
			// that line, even when another range makes it part of this union.
			if p.left && p.offset == inputUnits && inputUnits != block.start {
				mapped = outputUnits
			}
			writePoint(p, mapped)
			point++
		}
		inputUnits, outputUnits = nextInput, nextOutput
		if last {
			break
		}
		at = lineEnd
	}
	// Endpoints within the final separator retain their exact relative units;
	// the next logical line was excluded by the nonempty half-open range rule.
	for ; point < len(points); point++ {
		p := points[point]
		writePoint(p, outputUnits+(p.offset-block.end))
	}
	return Replacement{block.start, block.end, out.String()}, after, nil
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
