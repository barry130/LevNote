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
	e.cache = map[int]lineLayout{}
	e.rowCounts = map[int]int{}
	e.rowTree = map[int]int{}
	e.hasDesiredX = false
	e.cachedBuffer = e.Buffer
	e.cachedShapeLimit = e.shapingLimit()
	e.indexedLines = e.Buffer.LineCount()
	e.version = e.Buffer.Version()
}

func (e *Editor) releaseGeometry() {
	e.cache = map[int]lineLayout{}
	e.rowCounts = map[int]int{}
	e.rowTree = map[int]int{}
	e.indexedLines = e.Buffer.LineCount()
	e.version = e.Buffer.Version()
	e.hasDesiredX = false
}

func (e *Editor) invalidateCacheIfChanged() {
	if e.cachedBuffer != e.Buffer || e.cachedShapeLimit != e.shapingLimit() {
		e.resetGeometry()
		return
	}
	if e.version == e.Buffer.Version() {
		return
	}
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
}

// Unvisited lines count as one row until they reach the viewport. Only integer
// row counts survive layout eviction; scrolling never shapes the whole file.
// The sparse Fenwick index also keeps jumping to a far line logarithmic.
func (e *Editor) addRows(line, delta int) {
	for i := line + 1; i <= e.indexedLines; i += i & -i {
		e.rowTree[i] += delta
	}
}
func (e *Editor) setRows(line, count int) {
	old := e.rowCounts[line]
	if old == 0 {
		old = 1
	}
	if count != old {
		e.addRows(line, count-old)
		if count == 1 {
			delete(e.rowCounts, line)
		} else {
			e.rowCounts[line] = count
		}
	}
}
func (e *Editor) visualBase(line int) int {
	e.invalidateCacheIfChanged()
	result := line
	for i := line; i > 0; i -= i & -i {
		result += e.rowTree[i]
	}
	return result
}
func (e *Editor) totalVisualRows() int { return e.visualBase(e.Buffer.LineCount()) }
func (e *Editor) lineAndRowForVisual(visual int) (int, int) {
	visual = max(0, visual)
	count := e.Buffer.LineCount()
	line := sort.Search(count, func(i int) bool { return e.visualBase(i+1) > visual })
	line = min(line, count-1)
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
	font := ui.Font{Family: "monospace", Size: e.FontSize, LineHeight: 1.65}
	var layout *ui.TextLayout
	if e.Highlight != nil {
		spans := e.Highlight(value)
		var content strings.Builder
		for _, s := range spans {
			content.WriteString(s.Text)
		}
		// A plugin may style text, but it cannot replace what the user sees.
		if content.String() == value {
			layout = ui.ShapeRichText(spans, font, e.lineWidth())
		}
	}
	if layout == nil {
		layout = ui.ShapeText(value, font, e.lineWidth())
	}
	lines := layout.Lines()
	if len(lines) == 0 {
		lines = []ui.TextLine{{Rect: ui.Rect{H: e.rowHeight()}}}
	}
	starts := make([]int, len(lines))
	for i, row := range lines {
		starts[i] = row.Start
	}
	out := lineLayout{value: value, start: r.Start, windowStart: r.Start - span.Start(), totalUnits: total, rows: max(1, len(lines)), rowStart: starts, lines: lines, layout: layout}
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
	e.cache[index] = out
	e.setRows(index, out.rows)
	return out
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
	l := e.lineAt(line, column)
	return e.visualBase(line) + l.caretRow(column-l.windowStart, e.affinity)
}

func (e *Editor) Draw(p *ui.Painter, r ui.Rect) {
	if e.width != r.W && e.SoftWrap {
		e.resetGeometry()
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
	e.lineAt(caretLine, caretColumn)
	if e.revealCaret {
		e.ensureCaret()
		e.revealCaret = false
	}
	e.clampScroll()
	first, _ := e.lineAndRowForVisual(max(0, int((e.ScrollY-12)/e.rowHeight())))
	p.Clip(r, 0, func() {
		for line := first; line < e.Buffer.LineCount(); line++ {
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
				if e.composing {
					left, right = max(0, e.marked.Start-l.start), min(text.UTF16Len(l.value), e.marked.End-l.start)
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
		}
	})
	height := float32(e.totalVisualRows())*e.rowHeight() + 24
	if height > r.H {
		thumb := max(24, r.H*r.H/height)
		y := r.Y + e.ScrollY/max(1, height-r.H)*(r.H-thumb)
		p.Fill(ui.Rect{X: r.X + r.W - 7, Y: y, W: 4, H: thumb}, theme.Scrollbar.Alpha(.7), 2)
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
	if err != nil {
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
	line, column, _ := e.Buffer.LineColumn(e.Buffer.Selection().Head)
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
			line = next
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
