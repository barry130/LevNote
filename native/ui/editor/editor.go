// Package editor implements LevNote's text editing independently of a widget
// engine. MyGo provides native input transport and text geometry, never history.
package editor

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

const (
	lineCacheLimit = 256
	maxShapedLine  = 4 * 1024 // keep mixed-script shaping bounded in source/split
	segmentUnits   = 2 * 1024 // bounded editable window of a longer paragraph
	tabWidth       = "    "
	caretBlink     = 500 * time.Millisecond
)

var errLineTooLong = errors.New("超长行已按段显示；全部内容仍可编辑和保存。使用编辑区上方的上一段 / 下一段浏览")

type Editor struct {
	Buffer      *text.Buffer
	ReadOnly    bool
	SoftWrap    bool
	FontSize    float32
	LineNumbers bool
	// Label and MinHeight allow this same native input engine to serve a
	// bounded property field without using a toolkit-owned editing model.
	Label     string
	MinHeight float32
	// ShapeLimit lowers the per-line window for small property fields. Zero
	// uses the document default; it never raises the global work limit.
	ShapeLimit       int
	ScrollX, ScrollY float32
	OnChange         func()
	OnError          func(error)
	Copy             func(string) error
	// CopyThen, when supplied by a host, must call done on the UI goroutine.
	// The default native path also waits for confirmed clipboard success.
	CopyThen   func(string, func(error))
	Paste      func() (string, error)
	Highlight  func(line string) []ui.Span
	CaretBlink bool

	handle                       ui.Handle
	services                     ui.Services
	width, height                float32
	version                      uint64
	cache                        map[int]lineLayout
	cachedShapeLimit             int
	cachedBuffer                 *text.Buffer
	rowCounts                    map[int]int
	rowTree                      map[int]int
	indexedLines                 int
	affinity                     ui.TextAffinity
	desiredX                     float32
	hasDesiredX                  bool
	marked                       ui.TextInputRange
	composing, dragging, focused bool
	compositionBuffer            *text.Buffer
	compositionToken             text.CompositionToken
	compositionError             error
	longLineWarned               bool
	theme                        *ui.Theme
	frameNow                     time.Time
	blinkStart                   time.Time
	revealCaret                  bool
	viewGeneration               uint64
	pendingClipboard             *clipboardRequest
}

type clipboardRequest struct {
	value      string
	cut        bool
	dispatched bool
	buffer     *text.Buffer
	version    uint64
	generation uint64
	selection  text.Range
}

func New(buffer *text.Buffer) *Editor {
	return &Editor{
		Buffer: buffer, FontSize: 16, LineNumbers: true,
		Label: "文档编辑区", MinHeight: 80,
		cache: map[int]lineLayout{},
	}
}

func (e *Editor) View(c *ui.Context) {
	e.services, e.theme = c.Services(), c.Theme()
	e.frameNow = c.Now()
	if request := e.pendingClipboard; request != nil && !request.dispatched {
		request.dispatched = true
		c.WriteClipboardThen(request.value, func(err error) { e.finishClipboard(request, err) })
	}
	line, column, _ := e.Buffer.LineColumn(e.Buffer.Selection().Head)
	span, _ := e.Buffer.LineRange(line)
	if span.End()-span.Start() > e.shapingLimit() {
		l := e.lineAt(line, column)
		ui.Row(c.Key("segment-controls")).Wrap().Padding(4, 8).Gap(6).Children(func() {
			label := fmt.Sprintf("UTF-16 位置 %d–%d / %d", l.windowStart+1, l.windowStart+ui.UTF16Len(l.value), l.totalUnits)
			if e.ShapeLimit > 0 {
				label = fmt.Sprintf("位置 %d–%d / %d", l.windowStart+1, l.windowStart+ui.UTF16Len(l.value), l.totalUnits)
			}
			ui.Text(c, label).FontSize(11)
			if ui.Button(c, "上一段").Disabled(l.windowStart == 0).Clicked() {
				e.selectTo(span.Start()+max(0, l.windowStart-e.segmentSize()), false)
				e.Focus()
			}
			if ui.Button(c, "下一段").Disabled(l.windowStart+ui.UTF16Len(l.value) >= l.totalUnits).Clicked() {
				e.selectTo(l.start+ui.UTF16Len(l.value), false)
				e.lineAt(line, min(l.totalUnits, l.windowStart+ui.UTF16Len(l.value)+1))
				e.ensureCaret()
				e.Focus()
			}
		})
	}
	box := ui.Box(c.Key("editor-input")).Grow(1).MinHeight(e.MinHeight).MinWidth(80).Clip().
		Label(e.Label).Role(ui.RoleTextField).Focusable().
		Bind(&e.handle).HandleInput(e.HandleInput).Draw(e.Draw)
	if !e.ReadOnly {
		box.HandleTextInput(e)
	}
	// A focused editor owns cancellation while input is pending. Register on
	// the element so a host's window Escape shortcut cannot swallow it, even
	// when the first refused preedit has no marked range yet.
	if (e.composing || e.compositionIssue() != nil) && box.Shortcut(0, ui.KeyEscape) {
		e.cancelComposition()
	}
	e.focused = box.Focused()
	if e.focused {
		if e.blinkStart.IsZero() {
			e.blinkStart = c.Now()
		}
	} else {
		e.blinkStart = time.Time{}
	}
}

func (e *Editor) Focus() { e.handle.Focus() }

func (e *Editor) CancelFocus() { e.handle.CancelFocus() }

func (e *Editor) ReleaseView() {
	if e.composing {
		e.UnmarkText()
	}
	e.releaseGeometry()
	e.viewGeneration++
	e.pendingClipboard = nil
	e.focused = false
	e.dragging = false
	e.handle.CancelFocus()
}

func (e *Editor) SetFontSize(size float32) {
	e.FontSize = max(10, min(32, size))
	e.invalidateStyle()
}

func (e *Editor) SetSoftWrap(enabled bool) {
	if e.SoftWrap == enabled {
		return
	}
	e.SoftWrap = enabled
	e.invalidateStyle()
}

func (e *Editor) SetCaretBlink(enabled bool) {
	if e.CaretBlink == enabled {
		return
	}
	e.CaretBlink = enabled
	e.invalidateStyle()
}

func (e *Editor) invalidateStyle() {
	e.releaseGeometry()
	e.revealCaret = true
	e.blinkStart = time.Time{}
	e.services.Invalidate()
}

func (e *Editor) GetPosition() (line, column, visualLine int) {
	line, column, err := e.Buffer.LineColumn(e.Buffer.Selection().Head)
	if err != nil {
		return 0, 0, 0
	}
	e.lineAt(line, column)
	return line, column, e.visualLineOf(line, column)
}

func (e *Editor) SetPosition(line, column int) error {
	if err := e.compositionIssue(); err != nil {
		return err
	}
	e.UnmarkText()
	e.hasDesiredX = false
	e.affinity = ui.TextDownstream
	line = max(0, min(line, e.Buffer.LineCount()-1))
	span, _ := e.Buffer.LineRange(line)
	column = max(0, min(column, span.End()-span.Start()))
	r := e.normalized(ui.TextInputRange{Start: span.Start() + column, End: span.Start() + column})
	if err := e.Buffer.SetSelection(r.Start, r.Start); err != nil {
		return err
	}
	e.ensureCaret()
	e.services.Invalidate()
	return nil
}

func (e *Editor) rowHeight() float32 { return e.FontSize * 1.65 }
func (e *Editor) gutter() float32 {
	if e.LineNumbers {
		return 64
	}
	return 24
}

func (e *Editor) fail(err error) {
	if err != nil && e.OnError != nil {
		e.OnError(err)
	}
}

func (e *Editor) changed() {
	e.invalidateCacheIfChanged()
	e.hasDesiredX = false
	e.affinity = ui.TextDownstream
	e.blinkStart = e.frameNow
	e.ensureCaret()
	if e.OnChange != nil {
		e.OnChange()
	}
	e.services.Invalidate()
}

func (e *Editor) normalized(r ui.TextInputRange) ui.TextInputRange {
	r.Start = max(0, min(r.Start, e.Buffer.LenUTF16()))
	r.End = max(r.Start, min(r.End, e.Buffer.LenUTF16()))
	if _, err := e.Buffer.SliceUTF16(r.Start, r.Start); err != nil && r.Start > 0 {
		r.Start--
	}
	if _, err := e.Buffer.SliceUTF16(r.End, r.End); err != nil && r.End < e.Buffer.LenUTF16() {
		r.End++
	}
	return r
}

func (e *Editor) TextForRange(r ui.TextInputRange) (string, ui.TextInputRange) {
	r = e.normalized(r)
	value, err := e.Buffer.SliceUTF16(r.Start, r.End)
	e.fail(err)
	return value, r
}

func (e *Editor) Selection() ui.TextInputSelection {
	selection := e.Buffer.Selection()
	return ui.TextInputSelection{
		Range:    ui.TextInputRange{Start: selection.Start(), End: selection.End()},
		Reversed: selection.Head < selection.Anchor,
	}
}

func (e *Editor) MarkedRange() (ui.TextInputRange, bool) {
	return e.marked, e.composing && e.compositionBuffer == e.Buffer && e.Buffer.OwnsComposition(e.compositionToken)
}

// A refused IME update is not permission to commit the preceding preedit.
// Retain only its small error and owner, never the rejected incoming string.
func (e *Editor) compositionIssue() error {
	if e.compositionError != nil {
		return e.compositionError
	}
	if e.compositionBuffer != nil && e.compositionBuffer != e.Buffer {
		return errors.New("文档已变化；请按 Esc 取消未完成的输入后重试")
	}
	if e.composing && !e.Buffer.OwnsComposition(e.compositionToken) {
		return text.ErrComposition
	}
	return nil
}

func (e *Editor) refuseComposition(err error) {
	if errors.Is(err, text.ErrSizeLimit) {
		e.compositionBuffer = e.Buffer
		e.compositionError = fmt.Errorf("%w；输入法内容尚未接受，请缩短后重试，或按 Esc 取消本次输入", err)
		e.dragging = false
		err = e.compositionError
	}
	e.fail(err)
}

func (e *Editor) cancelComposition() bool {
	if !e.composing && e.compositionError == nil && e.compositionBuffer == nil {
		return false
	}
	owner, composed := e.compositionBuffer, e.composing
	var err error
	if composed {
		err = owner.CancelCompositionGroup(e.compositionToken)
	}
	e.composing = false
	e.marked = ui.TextInputRange{}
	e.compositionBuffer, e.compositionError = nil, nil
	e.compositionToken = text.CompositionToken{}
	e.fail(err)
	if composed && err == nil && owner == e.Buffer {
		e.changed()
	} else {
		e.services.Invalidate()
	}
	return true
}

func (e *Editor) replacement(r *ui.TextInputRange) ui.TextInputRange {
	if r != nil {
		return e.normalized(*r)
	}
	if e.composing {
		return e.marked
	}
	return e.Selection().Range
}

func (e *Editor) ReplaceText(r *ui.TextInputRange, value string) {
	if e.ReadOnly {
		return
	}
	if e.compositionBuffer != nil && e.compositionBuffer != e.Buffer {
		e.fail(e.compositionIssue())
		return
	}
	rr := e.replacement(r)
	before := e.Buffer.Version()
	wasComposing := e.composing
	var err error
	if e.composing {
		err = e.Buffer.ReplaceCompositionUTF16(e.compositionToken, rr.Start, rr.End, value)
	} else if e.compositionError != nil && e.Buffer.InGroup() {
		err = text.ErrGroup
	} else {
		err = e.Buffer.ReplaceUTF16(rr.Start, rr.End, value)
	}
	if err != nil {
		if e.composing || e.compositionError != nil {
			e.refuseComposition(err)
		} else {
			e.fail(err)
		}
		return
	}
	pos := rr.Start + ui.UTF16Len(value)
	e.fail(e.Buffer.SetSelection(pos, pos))
	if e.composing {
		if err = e.Buffer.EndCompositionGroup(e.compositionToken); err != nil {
			e.fail(err)
			return
		}
		e.composing = false
		e.marked = ui.TextInputRange{}
	}
	e.compositionBuffer, e.compositionError = nil, nil
	e.compositionToken = text.CompositionToken{}
	if e.Buffer.Version() != before || wasComposing {
		e.changed()
	} else {
		e.ensureCaret()
		e.services.Invalidate()
	}
}

func (e *Editor) SetMarkedText(r *ui.TextInputRange, value string, selected ui.TextInputRange) {
	if e.ReadOnly {
		return
	}
	if e.compositionBuffer != nil && e.compositionBuffer != e.Buffer {
		e.fail(e.compositionIssue())
		return
	}
	rr := e.replacement(r)
	started := !e.composing
	if started {
		if e.Buffer.InGroup() {
			e.fail(text.ErrGroup)
			return
		}
		if err := e.Buffer.CheckReplaceUTF16(rr.Start, rr.End, value); err != nil {
			e.refuseComposition(err)
			return
		}
		token, err := e.Buffer.BeginCompositionGroup()
		if err != nil {
			e.fail(err)
			return
		}
		e.composing = true
		e.compositionBuffer = e.Buffer
		e.compositionToken = token
	}
	if err := e.Buffer.ReplaceCompositionUTF16(e.compositionToken, rr.Start, rr.End, value); err != nil {
		e.refuseComposition(err)
		if started {
			e.fail(e.Buffer.CancelCompositionGroup(e.compositionToken))
			e.composing = false
			e.compositionToken = text.CompositionToken{}
			if e.compositionError == nil {
				e.compositionBuffer = nil
			}
		}
		return
	}
	e.compositionError = nil
	e.marked = ui.TextInputRange{Start: rr.Start, End: rr.Start + ui.UTF16Len(value)}
	selectedRange := e.normalized(ui.TextInputRange{
		Start: e.marked.Start + max(0, min(selected.Start, ui.UTF16Len(value))),
		End:   e.marked.Start + max(0, min(selected.End, ui.UTF16Len(value))),
	})
	e.fail(e.Buffer.SetSelection(selectedRange.Start, selectedRange.End))
	e.changed()
}

func (e *Editor) UnmarkText() {
	// Toolkit focus loss and host geometry release also call this method;
	// neither is an explicit cancellation of a refused input.
	if e.compositionIssue() != nil {
		return
	}
	if e.composing {
		if err := e.Buffer.EndCompositionGroup(e.compositionToken); err != nil {
			e.fail(err)
			return
		}
		e.composing = false
		e.marked = ui.TextInputRange{}
		e.compositionBuffer = nil
		e.compositionToken = text.CompositionToken{}
		e.services.Invalidate()
	}
}

// Flush commits the current native composition before a host saves or switches
// views. It never closes a transaction owned by another editor operation.
func (e *Editor) Flush() error {
	if err := e.compositionIssue(); err != nil {
		return err
	}
	e.UnmarkText()
	return nil
}

func (e *Editor) selectTo(pos int, extend bool) {
	if err := e.compositionIssue(); err != nil {
		e.fail(err)
		return
	}
	r := e.normalized(ui.TextInputRange{Start: pos, End: pos})
	pos = r.Start
	anchor := pos
	if extend {
		anchor = e.Buffer.Selection().Anchor
	}
	e.fail(e.Buffer.SetSelection(anchor, pos))
	e.blinkStart = e.frameNow
	e.ensureCaret()
	e.services.Invalidate()
}

func (e *Editor) boundary(pos, dir int) int {
	next, err := e.Buffer.GraphemeBoundary(pos, dir)
	e.fail(err)
	if err != nil {
		return pos
	}
	return next
}

func (e *Editor) word(pos, dir int) int {
	for n := 0; n < 4096; n++ {
		next := e.boundary(pos, dir)
		if next == pos {
			break
		}
		value, _ := e.Buffer.SliceUTF16(min(pos, next), max(pos, next))
		pos = next
		if strings.ContainsAny(value, " \t\r\n.,;:()[]{}") {
			break
		}
	}
	return pos
}

// selectionWordBoundary stops before the delimiter; navigation word deliberately
// consumes it. Keep the same grapheme stepping and bounded work as navigation.
func (e *Editor) selectionWordBoundary(pos, dir int) int {
	for n := 0; n < 4096; n++ {
		next := e.boundary(pos, dir)
		if next == pos {
			break
		}
		value, _ := e.Buffer.SliceUTF16(min(pos, next), max(pos, next))
		if strings.ContainsAny(value, " \t\r\n.,;:()[]{}") {
			break
		}
		pos = next
	}
	return pos
}

// A pointer query reports an insertion point, not the glyph side clicked.
// Limit the new selection rule to points surrounded by word graphemes; retain
// the existing interpretation at delimiters, word edges and document edges.
func (e *Editor) insideSelectionWord(pos int) bool {
	for _, dir := range []int{-1, 1} {
		next := e.boundary(pos, dir)
		if next == pos {
			return false
		}
		value, _ := e.Buffer.SliceUTF16(min(pos, next), max(pos, next))
		if strings.ContainsAny(value, " \t\r\n.,;:()[]{}") {
			return false
		}
	}
	return true
}

func (e *Editor) selectedLines() (int, int) {
	selection := e.Buffer.Selection()
	startLine, _, err := e.Buffer.LineColumn(selection.Start())
	if err != nil {
		return 0, -1
	}
	endLine, _, err := e.Buffer.LineColumn(selection.End())
	if err != nil {
		return startLine, e.Buffer.LineCount() - 1
	}
	if selection.Start() == selection.End() {
		return startLine, startLine
	}
	endStart, _ := e.Buffer.OffsetOfLine(endLine)
	if endLine > startLine && selection.End() == endStart {
		endLine--
	}
	return startLine, min(endLine, e.Buffer.LineCount()-1)
}

func (e *Editor) indentSelected(direction int) {
	startLine, endLine := e.selectedLines()
	if endLine < startLine {
		return
	}
	if e.Buffer.InGroup() && !e.composing {
		e.fail(text.ErrGroup)
		return
	}
	// Comparing the zero-based line span avoids overflowing a line count or
	// multiplying that count by the four-byte indentation.
	if direction > 0 && endLine-startLine >= (e.Buffer.MaxBytes()-e.Buffer.LenBytes())/len(tabWidth) {
		e.fail(&text.SizeLimitError{Limit: e.Buffer.MaxBytes()})
		return
	}
	e.UnmarkText()
	selection := e.Buffer.Selection()
	anchor, head := selection.Anchor, selection.Head
	changed := false
	if err := e.Buffer.BeginGroup(); err != nil {
		e.fail(err)
		return
	}
	mapOffset := func(offset, start, removed, added int) int {
		if offset < start {
			return offset
		}
		return max(start, offset-removed) + added
	}
	for line := startLine; line <= endLine; line++ {
		span, lineErr := e.Buffer.LineRange(line)
		if lineErr != nil {
			e.Buffer.CancelGroup()
			e.fail(lineErr)
			return
		}
		pos := span.Start()
		remove, insert := 0, tabWidth
		if direction < 0 {
			insert = ""
			prefix := e.normalized(ui.TextInputRange{Start: pos, End: min(pos+4, span.End())})
			value, _ := e.Buffer.SliceUTF16(prefix.Start, prefix.End)
			if strings.HasPrefix(value, "\t") {
				remove = 1
			} else {
				for remove < len(value) && remove < 4 && value[remove] == ' ' {
					remove++
				}
			}
		}
		if remove == 0 && insert == "" {
			continue
		}
		if err := e.Buffer.ReplaceUTF16(pos, pos+remove, insert); err != nil {
			e.Buffer.CancelGroup()
			e.fail(err)
			return
		}
		changed = true
		anchor = mapOffset(anchor, pos, remove, len(insert))
		head = mapOffset(head, pos, remove, len(insert))
	}
	e.fail(e.Buffer.SetSelection(anchor, head))
	// EndGroup reports whether there were edits, not whether closing succeeded.
	e.Buffer.EndGroup()
	if changed {
		e.changed()
	}
}

func (e *Editor) insertNewlineWithIndent() {
	e.UnmarkText()
	position := e.Buffer.Selection().Start()
	line, _, _ := e.Buffer.LineColumn(position)
	span, _ := e.Buffer.LineRange(line)
	// Copy only leading spaces/tabs before the insertion point. Inspect bounded
	// chunks so pressing Enter near the start of a huge line never copies it.
	var indentation strings.Builder
	for at := span.Start(); at < min(position, span.End()); {
		end := min(at+256, min(position, span.End()))
		r := e.normalized(ui.TextInputRange{Start: at, End: end})
		value, _ := e.Buffer.SliceUTF16(r.Start, r.End)
		n := 0
		for n < len(value) && (value[n] == ' ' || value[n] == '\t') {
			n++
		}
		indentation.WriteString(value[:n])
		if n != len(value) {
			break
		}
		at = r.End
	}
	newline := "\n"
	if line+1 < e.Buffer.LineCount() {
		next, _ := e.Buffer.OffsetOfLine(line + 1)
		newline, _ = e.Buffer.SliceUTF16(span.End(), next)
	} else if line > 0 {
		previous, _ := e.Buffer.LineRange(line - 1)
		newline, _ = e.Buffer.SliceUTF16(previous.End(), span.Start())
	}
	e.ReplaceText(nil, newline+indentation.String())
}

func (e *Editor) finishClipboard(request *clipboardRequest, err error) {
	if e.pendingClipboard != request {
		return
	}
	e.pendingClipboard = nil
	if err != nil {
		e.fail(err)
		return
	}
	if !request.cut {
		return
	}
	if err := e.compositionIssue(); err != nil {
		e.fail(err)
		return
	}
	if e.ReadOnly || e.Buffer != request.buffer || e.Buffer.Version() != request.version || e.Buffer.Selection() != request.selection || e.viewGeneration != request.generation {
		e.fail(errors.New("复制已完成；文档或选区已改变，未删除原内容"))
		return
	}
	e.ReplaceText(nil, "")
}

func (e *Editor) Command(command string) bool {
	if err := e.compositionIssue(); err != nil {
		e.fail(err)
		return true
	}
	selection := e.Buffer.Selection()
	switch command {
	case "selectAll":
		e.fail(e.Buffer.SetSelection(0, e.Buffer.LenUTF16()))
		e.ensureCaret()
		e.services.Invalidate()
		return true
	case "copy", "cut":
		if selection.Start() == selection.End() {
			return true
		}
		value, err := e.Buffer.SliceUTF16(selection.Start(), selection.End())
		if err != nil {
			e.fail(err)
			return true
		}
		if e.pendingClipboard != nil {
			e.fail(errors.New("上一次复制尚未完成，请稍后重试"))
			return true
		}
		e.UnmarkText()
		request := &clipboardRequest{value: value, cut: command == "cut" && !e.ReadOnly, buffer: e.Buffer, version: e.Buffer.Version(), generation: e.viewGeneration, selection: selection}
		e.pendingClipboard = request
		if e.Copy != nil {
			e.finishClipboard(request, e.Copy(value))
		} else if e.CopyThen != nil {
			request.dispatched = true
			e.CopyThen(value, func(err error) { e.finishClipboard(request, err) })
		} else {
			e.services.Invalidate()
		}
		return true
	case "paste", "pasteAndMatchStyle":
		if !e.ReadOnly {
			var value string
			if e.Paste != nil {
				var err error
				value, err = e.Paste()
				if err != nil {
					e.fail(err)
					return true
				}
			} else {
				value = e.services.ReadClipboard()
			}
			e.ReplaceText(nil, value)
		}
		return true
	case "undo":
		if !e.ReadOnly {
			e.UnmarkText()
			if e.Buffer.Undo() {
				e.changed()
			}
		}
		return true
	case "redo":
		if !e.ReadOnly {
			e.UnmarkText()
			if e.Buffer.Redo() {
				e.changed()
			}
		}
		return true
	case "delete":
		if !e.ReadOnly {
			e.delete(1)
		}
		return true
	case "indent":
		if !e.ReadOnly {
			e.indentSelected(1)
		}
		return true
	case "outdent":
		if !e.ReadOnly {
			e.indentSelected(-1)
		}
		return true
	}
	return false
}

func (e *Editor) delete(dir int) {
	e.UnmarkText()
	selection := e.Buffer.Selection()
	r := ui.TextInputRange{Start: selection.Start(), End: selection.End()}
	if r.Start == r.End {
		next := e.boundary(selection.Head, dir)
		r.Start = min(selection.Head, next)
		r.End = max(selection.Head, next)
	}
	e.ReplaceText(&r, "")
}

func (e *Editor) HandleInput(ev ui.InputEvent) bool {
	selection := e.Buffer.Selection()
	switch ev.Kind {
	case ui.InputText:
		e.ReplaceText(nil, ev.Text)
		return true
	case ui.InputCompose:
		e.SetMarkedText(nil, ev.Text, ui.TextInputRange{Start: ui.UTF16Len(ev.Text), End: ui.UTF16Len(ev.Text)})
		return true
	case ui.InputCommand:
		return e.Command(ev.Text)
	case ui.InputPointerDown:
		if ev.Button != 0 {
			return false
		}
		if err := e.compositionIssue(); err != nil {
			e.fail(err)
			return true
		}
		e.UnmarkText()
		position, ok := e.positionForPoint(ui.Point{X: ev.X, Y: ev.Y})
		if !ok {
			return false
		}
		e.hasDesiredX = false
		e.affinity = position.Affinity
		pos := position.Index
		e.selectTo(pos, ev.Mods&ui.Shift != 0)
		e.dragging = true
		if ev.Clicks == 2 && e.insideSelectionWord(pos) {
			e.fail(e.Buffer.SetSelection(e.selectionWordBoundary(pos, -1), e.selectionWordBoundary(pos, 1)))
		} else if ev.Clicks >= 2 {
			e.fail(e.Buffer.SetSelection(e.word(pos, -1), e.word(pos, 1)))
		}
		return true
	case ui.InputPointerMove:
		if e.compositionIssue() != nil {
			return true
		}
		if e.dragging {
			position, ok := e.positionForPoint(ui.Point{X: ev.X, Y: ev.Y})
			if ok {
				e.affinity = position.Affinity
				e.selectTo(position.Index, true)
			}
			return true
		}
	case ui.InputPointerUp:
		e.dragging = false
		return ev.Button == 0
	case ui.InputScroll:
		if e.SoftWrap {
			e.ScrollY += ev.DY
		} else if ev.Mods&ui.Shift != 0 {
			e.ScrollX += ev.DY
		} else {
			e.ScrollY += ev.DY
			e.ScrollX += ev.DX
		}
		e.clampScroll()
		return true
	case ui.InputKeyDown:
		if ev.Key == ui.KeyEscape && ev.Mods == 0 && e.cancelComposition() {
			return true
		}
		if err := e.compositionIssue(); err != nil {
			e.fail(err)
			return true
		}
		cmd, extend := ev.Mods&ui.Cmd != 0, ev.Mods&ui.Shift != 0
		if cmd {
			switch ev.Key {
			case ui.KeyA:
				return e.Command("selectAll")
			case ui.KeyC:
				return e.Command("copy")
			case ui.KeyX:
				return e.Command("cut")
			case ui.KeyV:
				return e.Command("paste")
			case ui.KeyZ:
				if extend {
					return e.Command("redo")
				}
				return e.Command("undo")
			case ui.KeyY:
				return e.Command("redo")
			}
		}
		switch ev.Key {
		case ui.KeyLeft, ui.KeyRight:
			e.UnmarkText()
			e.hasDesiredX = false
			e.affinity = ui.TextDownstream
			dir := 1
			if ev.Key == ui.KeyLeft {
				dir = -1
			}
			next := e.boundary(selection.Head, dir)
			if !extend && selection.Start() != selection.End() {
				if dir < 0 {
					next = selection.Start()
				} else {
					next = selection.End()
				}
			} else if cmd {
				next = e.word(selection.Head, dir)
			}
			e.selectTo(next, extend)
			return true
		case ui.KeyUp, ui.KeyDown, ui.KeyPageUp, ui.KeyPageDown:
			e.UnmarkText()
			delta := 1
			if ev.Key == ui.KeyPageUp || ev.Key == ui.KeyPageDown {
				delta = max(1, int(e.height/e.rowHeight())-1)
			}
			if ev.Key == ui.KeyUp || ev.Key == ui.KeyPageUp {
				delta = -delta
			}
			e.moveVertical(delta, extend)
			return true
		case ui.KeyHome, ui.KeyEnd:
			e.UnmarkText()
			e.hasDesiredX = false
			line, column, _ := e.Buffer.LineColumn(selection.Head)
			l := e.lineAt(line, column)
			row := l.caretRow(column-l.windowStart, e.affinity)
			next := l.start + l.lines[row].Start
			e.affinity = ui.TextDownstream
			if ev.Key == ui.KeyEnd {
				next = l.start + l.lines[row].End
				e.affinity = ui.TextUpstream
			}
			if cmd {
				next = 0
				if ev.Key == ui.KeyEnd {
					next = e.Buffer.LenUTF16()
				}
			}
			e.selectTo(next, extend)
			return true
		case ui.KeyBackspace:
			if !e.ReadOnly {
				e.delete(-1)
			}
			return true
		case ui.KeyDelete:
			if !e.ReadOnly {
				e.delete(1)
			}
			return true
		case ui.KeyEnter:
			if !e.ReadOnly {
				e.insertNewlineWithIndent()
			}
			return true
		case ui.KeyTab:
			if e.ReadOnly {
				return false
			}
			if extend {
				e.Command("outdent")
			} else if selection.Start() != selection.End() {
				e.Command("indent")
			} else {
				e.ReplaceText(nil, tabWidth)
			}
			return true
		}
	}
	return false
}

func (e *Editor) Reveal(start, end int) {
	if err := e.compositionIssue(); err != nil {
		e.fail(err)
		e.Focus()
		return
	}
	e.fail(e.Buffer.SetSelection(start, end))
	e.ensureCaret()
	e.Focus()
}
