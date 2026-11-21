package editor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

type lineLayout struct {
	value                          string
	start, windowStart, totalUnits int
	rows                           int
	rowStart                       []int
	lines                          []ui.TextLine
	layout                         *ui.TextLayout
}

func (e *Editor) resetGeometry() {
	e.geometryGeneration++
	e.cache = map[int]lineLayout{}
	e.rowCounts = map[int]int{}
	e.rowTree = map[int]int{}
	e.hasDesiredX = false
	e.cachedBuffer = e.Buffer
	e.cachedShapeLimit = e.shapingLimit()
	e.cachedGutter = e.gutter()
	e.indexedLines = e.Buffer.LineCount()
	e.version = e.Buffer.Version()
	e.syncFolds()
	e.bumpFoldProjection()
	e.rebuildFoldExtras()
}

func (e *Editor) releaseGeometry() {
	e.geometryGeneration++
	e.cache = map[int]lineLayout{}
	e.rowCounts = map[int]int{}
	e.rowTree = map[int]int{}
	e.indexedLines = e.Buffer.LineCount()
	e.version = e.Buffer.Version()
	e.hasDesiredX = false
	e.bumpFoldProjection()
	e.rebuildFoldExtras()
}

func (e *Editor) invalidateCacheIfChanged() {
	e.syncFolds()
	if e.cachedBuffer != e.Buffer || e.cachedShapeLimit != e.shapingLimit() || (e.SoftWrap && e.cachedGutter != e.gutter()) {
		e.resetGeometry()
		return
	}
	if e.version == e.Buffer.Version() {
		return
	}
	e.geometryGeneration++
	first := e.Buffer.ChangedLineSince(e.version)
	for line := range e.cache {
		if line >= first {
			delete(e.cache, line)
		}
	}
	for line := range e.rowCounts {
		if line >= first {
			delete(e.rowCounts, line)
		}
	}
	e.indexedLines = e.Buffer.LineCount()
	e.rowTree = map[int]int{}
	for line, count := range e.rowCounts {
		e.addRows(line, count-1)
	}
	e.version = e.Buffer.Version()
	e.syncFolds()
	e.rebuildFoldExtras()
}

// Unvisited lines count as one row until they reach the viewport. Only integer
// row counts survive layout eviction; scrolling never shapes the whole file.
// The sparse Fenwick index also keeps jumping to a far line logarithmic.
func (e *Editor) addRows(line, delta int) {
	// Include direct Fenwick writes in the navigation boundary epoch as well
	// as their callers' count/cache changes, including write-and-restore ABA.
	e.geometryGeneration++
	addRowDelta(e.rowTree, e.indexedLines, line, delta)
}
func addRowDelta(tree map[int]int, indexed, line, delta int) {
	for i := line + 1; i <= indexed; i += i & -i {
		tree[i] += delta
	}
}
func (e *Editor) setRows(line, count int) {
	old := e.rowCounts[line]
	if old == 0 {
		old = 1
	}
	if count != old {
		e.geometryGeneration++
		e.addRows(line, count-old)
		e.bumpFoldProjection()
		if count == 1 {
			delete(e.rowCounts, line)
		} else {
			e.rowCounts[line] = count
		}
	}
}
func (e *Editor) wrapPrefix(line int) int { return rowPrefix(e.rowTree, line) }

func rowPrefix(tree map[int]int, line int) int {
	result := 0
	for i := line; i > 0; i -= i & -i {
		result += tree[i]
	}
	return result
}
func (e *Editor) visualBase(line int) int {
	e.invalidateCacheIfChanged()
	wrap := e.wrapPrefix(line)
	hidden, extra := e.foldDeductions(line)
	return line - hidden + wrap - extra
}
func (e *Editor) totalVisualRows() int { return e.visualBase(e.Buffer.LineCount()) }
func (e *Editor) lineAndRowForVisual(visual int) (int, int) {
	visual = max(0, visual)
	count := e.Buffer.LineCount()
	line := sort.Search(count, func(i int) bool { return e.visualBase(i+1) > visual })
	line = min(line, count-1)
	for e.hiddenLine(line) && line < count-1 {
		line = e.nextVisible(line, 1)
	}
	return line, max(0, min(visual-e.visualBase(line), max(1, e.rowCounts[line])-1))
}
func (e *Editor) lineWidth() float32 {
	if !e.SoftWrap || e.width <= e.gutter()+16 {
		return 0
	}
	return e.width - e.gutter() - 16
}
func (e *Editor) shapingLimit() int {
	if e.ShapeLimit > 0 {
		return max(128, min(e.ShapeLimit, maxShapedLine))
	}
	return maxShapedLine
}
func (e *Editor) segmentSize() int          { return min(segmentUnits, e.shapingLimit()) }
func (e *Editor) line(index int) lineLayout { return e.lineAt(index, -1) }
func (e *Editor) lineAtVisualRow(index, row int) lineLayout {
	return e.line(index)
}
func (e *Editor) lineAt(index, column int) lineLayout {
	e.invalidateCacheIfChanged()
	if cached, ok := e.cache[index]; ok && (column < 0 || column >= cached.windowStart && column <= cached.windowStart+text.UTF16Len(cached.value)) {
		return cached
	}
	span, err := e.Buffer.LineRange(index)
	if err != nil {
		e.fail(err)
		span = text.Range{}
	}
	total := span.End() - span.Start()
	window, windowUnits := 0, e.shapingLimit()
	if total > e.shapingLimit() {
		windowUnits = e.segmentSize()
		window = max(0, min(column, total-1)) / windowUnits * windowUnits
		if !e.longLineWarned {
			e.longLineWarned = true
			e.fail(errLineTooLong)
		}
	}
	start, end := span.Start()+window, min(span.End(), span.Start()+window+windowUnits)
	r := e.normalized(ui.TextInputRange{Start: start, End: end})
	value, err := e.Buffer.SliceUTF16(r.Start, r.End)
	e.fail(err)
	fontSize, rich := e.FontSize, e.Highlight != nil
	var spans []ui.Span
	if rich {
		spans = e.Highlight(value)
	}
	out := shapeEditorLine(value, fontSize, e.lineWidth(), spans, rich)
	out.start, out.windowStart, out.totalUnits = r.Start, r.Start-span.Start(), total
	if len(e.cache) >= lineCacheLimit {
		farthest, distance := index, -1
		for key := range e.cache {
			d := key - index
			if d < 0 {
				d = -d
			}
			if d > distance {
				farthest, distance = key, d
			}
		}
		delete(e.cache, farthest)
	}
	e.geometryGeneration++
	e.cache[index] = out
	e.setRows(index, out.rows)
	return out
}

// shapeEditorLine has no Editor or host callbacks. Both ordinary drawing and
// detached source navigation use the same rich/plain text and row metrics.
func shapeEditorLine(value string, fontSize, width float32, spans []ui.Span, rich bool) lineLayout {
	font := ui.Font{Family: "monospace", Size: fontSize, LineHeight: 1.65}
	var layout *ui.TextLayout
	if rich {
		var content strings.Builder
		for _, span := range spans {
			content.WriteString(span.Text)
		}
		// A plugin may style text, but it cannot replace what the user sees.
		if content.String() == value {
			layout = ui.ShapeRichText(spans, font, width)
		}
	}
	if layout == nil {
		layout = ui.ShapeText(value, font, width)
	}
	lines := layout.Lines()
	if len(lines) == 0 {
		lines = []ui.TextLine{{Rect: ui.Rect{H: fontSize * 1.65}}}
	}
	starts := make([]int, len(lines))
	for i, row := range lines {
		starts[i] = row.Start
	}
	return lineLayout{value: value, rows: max(1, len(lines)), rowStart: starts, lines: lines, layout: layout}
}

func (l lineLayout) rowAt(local int) int {
	return max(0, sort.SearchInts(l.rowStart, local+1)-1)
}
func (l lineLayout) caretRow(local int, affinity ui.TextAffinity) int {
	row := l.rowAt(local)
	if row > 0 && local == l.rowStart[row] && affinity == ui.TextUpstream {
		row--
	}
	return row
}
func (e *Editor) visualLineOf(line, column int) int {
	if e.hiddenLine(line) {
		return e.visualBase(line)
	}
	l := e.lineAt(line, column)
	return e.visualBase(line) + l.caretRow(column-l.windowStart, e.affinity)
}

// sourceRevealLayout prepares navigation geometry for one fixed Buffer. Shape
// and fold state are detached. Compatible row maps are read-only bases until
// callback-free commit; only the revealed line and O(log N) Fenwick delta nodes
// are private writes. Host callbacks are followed immediately by a pure guard.
type sourceRevealLayout struct {
	buffer                          *text.Buffer
	cache                           map[int]lineLayout
	counts, tree                    map[int]int
	countChanges, treeChanges       map[int]int
	folds                           *foldState
	indexed, limit, segment         int
	version                         uint64
	softWrap                        bool
	fontSize, width, height, gutter float32
	scrollX, scrollY                float32
	affinity                        ui.TextAffinity
	longWarned                      bool
}

func newSourceRevealLayout(e *Editor) *sourceRevealLayout {
	p := &sourceRevealLayout{
		buffer: e.Buffer, folds: cloneSourceFolds(e.folds), indexed: e.Buffer.LineCount(),
		limit: e.shapingLimit(), segment: e.segmentSize(), version: e.Buffer.Version(),
		softWrap: e.SoftWrap, fontSize: e.FontSize, width: e.width, height: e.height, gutter: e.gutter(),
		scrollX: e.ScrollX, scrollY: e.ScrollY, affinity: e.affinity, longWarned: e.longLineWarned,
		countChanges: map[int]int{}, treeChanges: map[int]int{},
	}
	if f := p.folds; f != nil && f.provider != nil && (f.buffer != p.buffer || f.version != p.version || f.viewGeneration != e.viewGeneration) {
		f.buffer, f.version, f.viewGeneration = p.buffer, p.version, e.viewGeneration
		f.candidates++
		f.ready = false
		f.ranges, f.intervals, f.collapsed, f.reported = nil, nil, nil, nil
		f.status = "文档已变化；暂时显示全文，请重新分析折叠"
	}
	if e.cachedBuffer != p.buffer || e.cachedShapeLimit != p.limit || p.softWrap && e.cachedGutter != p.gutter {
		p.cache, p.counts, p.tree = map[int]lineLayout{}, map[int]int{}, map[int]int{}
		if p.folds != nil {
			p.folds.projection++
		}
	} else {
		p.cache = cloneSourceLineCache(e.cache)
		if p.cache == nil {
			p.cache = map[int]lineLayout{}
		}
		if e.version == p.version {
			p.counts, p.tree = e.rowCounts, e.rowTree
		} else {
			// Decide validity before touching retained row data. Probe only the
			// still-valid CURRENT prefix, never iterate the obsolete map domain
			// (a same-buffer shrink may leave an arbitrarily larger old suffix).
			first := min(p.indexed, p.buffer.ChangedLineSince(e.version))
			for line := range p.cache {
				if line >= first {
					delete(p.cache, line)
				}
			}
			p.counts, p.tree = map[int]int{}, map[int]int{}
			if len(e.rowCounts) > 0 {
				for line := 0; line < first; line++ {
					if count := e.rowCounts[line]; count > 1 {
						p.counts[line] = count
						addRowDelta(p.tree, p.indexed, line, count-1)
					}
				}
			}
		}
	}
	p.updateFoldExtras()
	return p
}

func (p *sourceRevealLayout) addRows(line, delta int) {
	addRowDelta(p.treeChanges, p.indexed, line, delta)
}
func (p *sourceRevealLayout) rowCount(line int) int {
	if count, ok := p.countChanges[line]; ok {
		return count
	}
	return max(1, p.counts[line])
}
func (p *sourceRevealLayout) wrapPrefix(line int) int {
	return rowPrefix(p.tree, line) + rowPrefix(p.treeChanges, line)
}
func (p *sourceRevealLayout) updateFoldExtras() {
	if p.folds == nil {
		return
	}
	hidden, extra := 0, 0
	for i := range p.folds.intervals {
		s := &p.folds.intervals[i]
		s.hiddenBefore, s.extraBefore = hidden, extra
		hidden += s.end - s.start
		extra += p.wrapPrefix(s.end) - p.wrapPrefix(s.start)
	}
}
func (p *sourceRevealLayout) visualBase(line int) int {
	hidden, extra := 0, 0
	if p.folds != nil && len(p.folds.intervals) > 0 {
		intervals := p.folds.intervals
		i := sort.Search(len(intervals), func(i int) bool { return intervals[i].end >= line })
		if i == len(intervals) {
			i--
		}
		s := intervals[i]
		end := max(s.start, min(line, s.end))
		hidden = s.hiddenBefore + end - s.start
		extra = s.extraBefore + p.wrapPrefix(end) - p.wrapPrefix(s.start)
	}
	return line - hidden + p.wrapPrefix(line) - extra
}
func (p *sourceRevealLayout) hiddenLine(line int) bool {
	if p.folds == nil {
		return false
	}
	intervals := p.folds.intervals
	i := sort.Search(len(intervals), func(i int) bool { return intervals[i].end > line })
	return i < len(intervals) && intervals[i].start <= line
}
func (p *sourceRevealLayout) visibleLine(line int) int {
	if p.folds != nil {
		intervals := p.folds.intervals
		i := sort.Search(len(intervals), func(i int) bool { return intervals[i].end > line })
		if i < len(intervals) && intervals[i].start <= line {
			return intervals[i].end
		}
	}
	return line
}
func (p *sourceRevealLayout) lineAndRow(visual int) (int, int) {
	visual = max(0, visual)
	line := sort.Search(p.indexed, func(i int) bool { return p.visualBase(i+1) > visual })
	line = min(line, p.indexed-1)
	line = p.visibleLine(line)
	return line, max(0, min(visual-p.visualBase(line), p.rowCount(line)-1))
}
func (p *sourceRevealLayout) clampScroll() {
	if p.softWrap {
		p.scrollX = 0
	} else {
		p.scrollX = max(0, p.scrollX)
	}
	if p.width > 0 && p.height > 0 {
		total := float32(p.visualBase(p.indexed))*p.fontSize*1.65 + 24
		p.scrollY = max(0, min(p.scrollY, total-p.height))
	}
}
func (p *sourceRevealLayout) revealFolds(start, end int) {
	f := p.folds
	if f == nil || len(f.collapsed) == 0 {
		return
	}
	rowHeight := p.fontSize * 1.65
	visual := max(0, int((p.scrollY-12)/rowHeight))
	line, row := p.lineAndRow(visual)
	fraction := p.scrollY - float32(visual)*rowHeight
	changed := false
	left, right := min(start, end), max(start, end)
	for r := range f.collapsed {
		first, _ := p.buffer.OffsetOfLine(r.OpenLine + 1)
		last, _ := p.buffer.OffsetOfLine(r.CloseLine)
		// The directed head may be exactly at the nonempty range's far
		// boundary. It must be visible as in legacy Reveal's second fold pass.
		if intersectsFold(left, right, first, last) || intersectsFold(end, end, first, last) {
			delete(f.collapsed, r)
			changed = true
		}
	}
	if !changed {
		return
	}
	f.projection++
	f.intervals = sourceFoldIntervals(f.collapsed)
	p.updateFoldExtras()
	for index := range p.cache {
		if p.hiddenLine(index) {
			delete(p.cache, index)
		}
	}
	if visible := p.visibleLine(line); visible != line {
		line, row = visible, 0
	}
	p.scrollY = float32(p.visualBase(line)+row)*rowHeight + fraction
	p.clampScroll()
}

func (p *sourceRevealLayout) prepareLine(e *Editor, guard sourceNavigationGuard, index, column int) (lineLayout, error) {
	highlight, report := e.Highlight, e.OnError
	if line, ok := p.cache[index]; ok && column >= line.windowStart && column <= line.windowStart+text.UTF16Len(line.value) {
		return line, nil
	}
	span, err := p.buffer.LineRange(index)
	if err != nil {
		return lineLayout{}, err
	}
	total, window, units := span.End()-span.Start(), 0, p.limit
	if total > p.limit {
		units = p.segment
		window = max(0, min(column, total-1)) / units * units
		if !p.longWarned {
			p.longWarned = true
			if report != nil {
				report(errLineTooLong)
				if !guard.matches(e) {
					return lineLayout{}, errSourceTargetStale
				}
			}
		}
	}
	r := normalizedBufferRange(p.buffer, ui.TextInputRange{Start: span.Start() + window, End: min(span.End(), span.Start()+window+units)})
	value, err := p.buffer.SliceUTF16(r.Start, r.End)
	if err != nil {
		return lineLayout{}, err
	}
	var spans []ui.Span
	if highlight != nil {
		spans = highlight(value)
		if !guard.matches(e) {
			return lineLayout{}, errSourceTargetStale
		}
	}
	width := float32(0)
	if p.softWrap && p.width > p.gutter+16 {
		width = p.width - p.gutter - 16
	}
	out := shapeEditorLine(value, p.fontSize, width, spans, highlight != nil)
	out.start, out.windowStart, out.totalUnits = r.Start, r.Start-span.Start(), total
	if len(p.cache) >= lineCacheLimit {
		farthest, distance := index, -1
		for key := range p.cache {
			d := key - index
			if d < 0 {
				d = -d
			}
			if d > distance {
				farthest, distance = key, d
			}
		}
		delete(p.cache, farthest)
	}
	p.cache[index] = out
	old := p.rowCount(index)
	if old != out.rows {
		p.addRows(index, out.rows-old)
		p.countChanges[index] = out.rows
		if p.folds != nil {
			p.folds.projection++
		}
		p.updateFoldExtras()
	}
	return out, nil
}

func (p *sourceRevealLayout) prepare(e *Editor, guard sourceNavigationGuard, start, end int) error {
	p.revealFolds(start, end)
	line, column, err := p.buffer.LineColumn(end)
	if err != nil {
		return err
	}
	l, err := p.prepareLine(e, guard, line, column)
	if err != nil {
		return err
	}
	caret := l.layout.CaretAt(ui.TextCaretPosition{Index: column - l.windowStart, Affinity: p.affinity})
	rowHeight := p.fontSize * 1.65
	y := 12 + float32(p.visualBase(line))*rowHeight + caret.Y
	if y < p.scrollY {
		p.scrollY = y
	}
	if p.height > 0 && y+rowHeight+12 > p.scrollY+p.height {
		p.scrollY = max(0, y+rowHeight+12-p.height)
	}
	if !p.softWrap && p.width > 0 {
		if caret.X < p.scrollX {
			p.scrollX = caret.X
		}
		if caret.X+12 > p.scrollX+p.width-p.gutter {
			p.scrollX = max(0, caret.X+12-p.width+p.gutter)
		}
	}
	p.clampScroll()
	return nil
}

func (p *sourceRevealLayout) commit(e *Editor) {
	// The final guard has proved that these bases still belong to this call.
	// No host code runs while the small delta is installed into compatible
	// live maps (or into the newly rebuilt current-prefix maps).
	e.geometryGeneration++
	if p.counts == nil {
		p.counts = map[int]int{}
	}
	if p.tree == nil {
		p.tree = map[int]int{}
	}
	for line, count := range p.countChanges {
		if count == 1 {
			delete(p.counts, line)
		} else {
			p.counts[line] = count
		}
	}
	for node, delta := range p.treeChanges {
		p.tree[node] += delta
	}
	e.cache, e.rowCounts, e.rowTree = p.cache, p.counts, p.tree
	e.cachedBuffer, e.cachedShapeLimit, e.cachedGutter = p.buffer, p.limit, p.gutter
	e.indexedLines, e.version, e.folds = p.indexed, p.version, p.folds
	e.ScrollX, e.ScrollY, e.longLineWarned = p.scrollX, p.scrollY, p.longWarned
}

func (e *Editor) Draw(p *ui.Painter, r ui.Rect) {
	marked, drawMarked := e.MarkedRange()
	// Observe attachment/revision/token invalidation without settling any owner.
	if e.compositionOwnerIssue(true) != nil {
		drawMarked = false
	}
	if e.width != r.W && e.SoftWrap {
		e.resetGeometry()
		e.revealCaret = true
	}
	if e.height != r.H && e.hasCollapsedFolds() {
		e.revealCaret = true
	}
	e.width, e.height = r.W, r.H
	theme := e.theme
	if theme == nil {
		theme = ui.LightTheme()
	}
	p.Fill(r, theme.Background, 0)
	selection := e.Buffer.Selection()
	caretLine, caretColumn, _ := e.Buffer.LineColumn(selection.Head)
	if !e.hiddenLine(caretLine) {
		e.lineAt(caretLine, caretColumn)
	}
	if e.revealCaret {
		e.ensureCaret()
		e.revealCaret = false
	}
	e.clampScroll()
	first, _ := e.lineAndRowForVisual(max(0, int((e.ScrollY-12)/e.rowHeight())))
	p.Clip(r, 0, func() {
		for line := first; line < e.Buffer.LineCount(); {
			if e.hiddenLine(line) {
				line = e.nextVisible(line, 1)
				continue
			}
			y := r.Y + 12 + float32(e.visualBase(line))*e.rowHeight() - e.ScrollY
			if y > r.Y+r.H {
				break
			}
			l := e.line(line)
			if line == caretLine {
				caret := l.layout.CaretAt(ui.TextCaretPosition{Index: caretColumn - l.windowStart, Affinity: e.affinity})
				p.Fill(ui.Rect{X: r.X, Y: y + caret.Y, W: r.W, H: e.rowHeight()}, theme.SurfaceHover.Alpha(.35), 0)
			}
			if e.LineNumbers {
				p.Text(r.X+16, y+2, fmt.Sprint(line+1), e.FontSize-2, theme.TextMuted)
			}
			if candidate, ok := e.foldCandidateAtLine(line); ok {
				glyph := "▾"
				if candidate.Collapsed {
					glyph = "▸"
				}
				p.Text(r.X+2, y+2, glyph, e.FontSize-2, theme.Accent)
				if candidate.Collapsed {
					count, _ := e.foldDeductions(candidate.Range.CloseLine)
					before, _ := e.foldDeductions(candidate.Range.OpenLine + 1)
					p.Text(r.X+34, y+2, fmt.Sprintf("…%d", count-before), e.FontSize-2, theme.TextMuted)
				}
			}
			x := r.X + e.gutter() - e.ScrollX
			p.Clip(ui.Rect{X: r.X + e.gutter() - 4, Y: r.Y, W: max(0, r.W-e.gutter()+4), H: r.H}, 0, func() {
				left, right := max(0, selection.Start()-l.start), min(text.UTF16Len(l.value), selection.End()-l.start)
				if right > left {
					for _, rect := range l.layout.SelectionRects(ui.TextInputRange{Start: left, End: right}) {
						rect.X += x
						rect.Y += y
						p.Fill(rect, theme.Selection, 1)
					}
				}
				// Paint a paragraph once. Its retained layout already contains
				// the vertical offsets of all its wrapped rows.
				p.TextLayout(l.layout, x, y, theme.Text)
				if drawMarked {
					left, right = max(0, marked.Start-l.start), min(text.UTF16Len(l.value), marked.End-l.start)
					if right > left {
						for _, rect := range l.layout.SelectionRects(ui.TextInputRange{Start: left, End: right}) {
							p.Fill(ui.Rect{X: x + rect.X, Y: y + rect.Y + rect.H - 1, W: rect.W, H: 1}, theme.Accent, 0)
						}
					}
				}
				if e.focused && line == caretLine {
					phase := max(0, e.frameNow.Sub(e.blinkStart))
					if !e.CaretBlink || (phase/caretBlink)%2 == 0 || e.dragging {
						caret := l.layout.CaretAt(ui.TextCaretPosition{Index: caretColumn - l.windowStart, Affinity: e.affinity})
						p.Fill(ui.Rect{X: x + caret.X, Y: y + caret.Y, W: 1.5, H: max(e.FontSize, caret.H)}, theme.Accent, 0)
					}
					if e.CaretBlink {
						p.After(caretBlink - phase%caretBlink)
					}
				}
			})
			line++
		}
	})
	height := float32(e.totalVisualRows())*e.rowHeight() + 24
	if height > r.H {
		thumb := max(24, r.H*r.H/height)
		y := r.Y + e.ScrollY/max(1, height-r.H)*(r.H-thumb)
		p.Fill(ui.Rect{X: r.X + r.W - 7, Y: y, W: 4, H: thumb}, theme.Scrollbar.Alpha(.7), 2)
	}
	if e.folds != nil {
		e.folds.painted = e.foldIdentity()
	}
}
func (e *Editor) clampScroll() {
	if e.SoftWrap {
		e.ScrollX = 0
	} else {
		e.ScrollX = max(0, e.ScrollX)
	}
	if e.width <= 0 || e.height <= 0 {
		return
	}
	total := float32(e.totalVisualRows())*e.rowHeight() + 24
	e.ScrollY = max(0, min(e.ScrollY, total-e.height))
}
func (e *Editor) ensureCaret() {
	line, column, err := e.Buffer.LineColumn(e.Buffer.Selection().Head)
	if err != nil || e.hiddenLine(line) {
		return
	}
	l := e.lineAt(line, column)
	caret := l.layout.CaretAt(ui.TextCaretPosition{Index: column - l.windowStart, Affinity: e.affinity})
	y := 12 + float32(e.visualBase(line))*e.rowHeight() + caret.Y
	if y < e.ScrollY {
		e.ScrollY = y
	}
	if e.height > 0 && y+e.rowHeight()+12 > e.ScrollY+e.height {
		e.ScrollY = max(0, y+e.rowHeight()+12-e.height)
	}
	if e.SoftWrap || e.width <= 0 {
		return
	}
	if caret.X < e.ScrollX {
		e.ScrollX = caret.X
	}
	if caret.X+12 > e.ScrollX+e.width-e.gutter() {
		e.ScrollX = max(0, caret.X+12-e.width+e.gutter())
	}
}

// BoundsForRange returns the first visible row of a native request and its
// actual source range. Native IME callers must not receive a rectangle claiming
// to represent additional wrapped/physical rows which it does not cover.
func (e *Editor) BoundsForRange(r ui.TextInputRange) (ui.Rect, ui.TextInputRange, bool) {
	r = e.normalized(r)
	if e.hiddenLineMustReject(r.Start) || r.End > r.Start && e.rangeTouchesHidden(r.Start, r.End) {
		return ui.Rect{}, r, false
	}
	line, column, err := e.Buffer.LineColumn(r.Start)
	if err != nil {
		return ui.Rect{}, r, false
	}
	l := e.lineAt(line, column)
	local := max(0, min(column-l.windowStart, text.UTF16Len(l.value)))
	row := l.caretRow(local, e.affinity)
	actual := ui.TextInputRange{Start: l.start + local, End: min(r.End, l.start+l.lines[row].End)}
	actual.End = max(actual.Start, actual.End)
	rect := l.layout.CaretAt(ui.TextCaretPosition{Index: local, Affinity: e.affinity})
	if actual.End > actual.Start {
		if rectangles := l.layout.SelectionRects(ui.TextInputRange{Start: local, End: actual.End - l.start}); len(rectangles) > 0 {
			rect = rectangles[0]
			for _, other := range rectangles[1:] {
				left, top := min(rect.X, other.X), min(rect.Y, other.Y)
				rect = ui.Rect{X: left, Y: top, W: max(rect.X+rect.W, other.X+other.W) - left, H: max(rect.Y+rect.H, other.Y+other.H) - top}
			}
		}
	}
	rect.X += e.gutter() - e.ScrollX
	rect.Y += 12 + float32(e.visualBase(line))*e.rowHeight() - e.ScrollY
	return rect, actual, rect.Y+rect.H >= 0 && rect.Y < e.height && rect.X+rect.W >= e.gutter()-4 && rect.X < e.width
}

func (e *Editor) positionForPoint(p ui.Point) (ui.TextCaretPosition, bool) {
	y := max(0, p.Y-12+e.ScrollY)
	line, _ := e.lineAndRowForVisual(int(y / e.rowHeight()))
	if line < 0 {
		return ui.TextCaretPosition{}, false
	}
	// An unvisited line initially occupies one row. Measure only the candidate
	// and subsequent visible lines, then use their actual paragraph geometry.
	l := e.line(line)
	localY := y - float32(e.visualBase(line))*e.rowHeight()
	position := l.layout.PositionAt(ui.Point{X: p.X - e.gutter() + e.ScrollX, Y: localY})
	position.Index += l.start
	return position, true
}

func (e *Editor) IndexForPoint(p ui.Point) (int, bool) {
	position, ok := e.positionForPoint(p)
	return position.Index, ok
}

func (e *Editor) moveVertical(delta int, extend bool) {
	head := e.Buffer.Selection().Head
	if e.hiddenLineMustReject(head) {
		e.revealFolds(head, head, false)
	}
	line, column, _ := e.Buffer.LineColumn(head)
	l := e.lineAt(line, column)
	row := l.caretRow(column-l.windowStart, e.affinity)
	if !e.hasDesiredX {
		e.desiredX = l.layout.CaretAt(ui.TextCaretPosition{Index: column - l.windowStart, Affinity: e.affinity}).X
		e.hasDesiredX = true
	}
	direction := 1
	if delta < 0 {
		direction = -1
		delta = -delta
	}
	for ; delta > 0; delta-- {
		row += direction
		if row >= 0 && row < l.rows {
			continue
		}
		// Treat a long-line segment as a visible portion of that same line.
		// Crossing its edge measures the adjacent bounded segment, never the
		// whole logical line or all preceding paragraphs.
		if direction > 0 && l.windowStart+text.UTF16Len(l.value) < l.totalUnits {
			l = e.lineAt(line, l.windowStart+text.UTF16Len(l.value)+1)
			row = 0
		} else if direction < 0 && l.windowStart > 0 {
			l = e.lineAt(line, l.windowStart-1)
			row = l.rows - 1
		} else if next := line + direction; next >= 0 && next < e.Buffer.LineCount() {
			line = e.nextVisible(next, direction)
			column := 0
			if direction < 0 {
				span, _ := e.Buffer.LineRange(line)
				column = span.End() - span.Start()
			}
			l = e.lineAt(line, column)
			row = 0
			if direction < 0 {
				row = l.rows - 1
			}
		} else {
			row = max(0, min(row, l.rows-1))
			break
		}
	}
	rowBox := l.lines[row].Rect
	position := l.layout.PositionAt(ui.Point{X: e.desiredX, Y: rowBox.Y + rowBox.H/2})
	e.affinity = position.Affinity
	e.selectTo(l.start+position.Index, extend)
}
