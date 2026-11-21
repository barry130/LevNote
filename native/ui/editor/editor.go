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
	// Use SetBuffer and SetReadOnly when changing an attached editor. Direct
	// field assignments are supported for compatibility but can miss ABA.
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
	cachedGutter                 float32
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
	compositionStamp             inputStamp
	compositionGeneration        uint64
	compositionMulti             bool
	compositionMarked            bool
	compositionPrimary           text.Range
	compositionTargetAdmitted    bool
	inputOwner                   inputOwner
	nativeClient                 *nativeInputClient
	longLineWarned               bool
	theme                        *ui.Theme
	frameNow                     time.Time
	blinkStart                   time.Time
	revealCaret                  bool
	viewGeneration               uint64
	pendingClipboard             *clipboardRequest
	folds                        *foldState
	foldControl                  foldControlPress
	// UI-thread mutations that can invalidate a detached source navigation.
	geometryGeneration   uint64
	navigationGeneration uint64

	terminalObservedBuffer                                        *text.Buffer
	terminalObservedReadonly                                      bool
	terminalObservedView                                          uint64
	terminalBufferEpoch, terminalReadonlyEpoch, terminalViewEpoch *inputTerminalEpoch
	terminalRegistration                                          *inputTerminalEpoch
	inputTerminal                                                 func(InputDelivery)
	inputDelivery                                                 InputDelivery
	inputDeliveryValid                                            bool
	inputActionID, inputDeliveryID                                InputNotificationID
	compositionChangeStamp                                        text.ChangeStamp
}

type clipboardRequest struct {
	value      string
	cut        bool
	dispatched bool
	completed  bool
	target     clipboardTarget
	fold       FoldTarget
	origin     InputBinding
	onChange   func()
	onError    func(error)
}

// Both clipboard transports report completion on the UI thread. Claim the
// first result before its callback can change a shared initiating action.
func (r *clipboardRequest) claimCompletion() bool {
	if r.completed {
		return false
	}
	r.completed = true
	return true
}

func New(buffer *text.Buffer) *Editor {
	e := &Editor{
		Buffer: buffer, FontSize: 16, LineNumbers: true,
		Label: "文档编辑区", MinHeight: 80,
		cache: map[int]lineLayout{},
	}
	e.observeTerminalAttachment()
	return e
}

func (e *Editor) View(c *ui.Context) {
	a := e.beginInputAction()
	e.viewInputAction(a, c)
	a.finish()
}
func (e *Editor) viewInputAction(a *inputAction, c *ui.Context) {
	e.services, e.theme = c.Services(), c.Theme()
	e.frameNow = c.Now()
	if request := e.pendingClipboard; request != nil && !request.dispatched {
		// This frame admits the queued write. Any pre-dispatch failure belongs
		// to the current View action. A transport result retains request origin.
		if err := e.clipboardIssue(request.target); err != nil {
			e.pendingClipboard = nil
			a.fail(err)
			return
		}
		request.dispatched = true
		returned, delivery := false, e.inputDeliveryID
		c.WriteClipboardThen(request.value, func(err error) {
			if returned {
				e.finishClipboard(request, err)
				return
			}
			if !request.claimCompletion() {
				return
			}
			if e.inputDeliveryID != delivery {
				a.suppressPublish = true
			}
			e.finishClipboardInput(a, request, err)
			if a.changed || a.errorCount != 0 {
				a.finish()
			}
		})
		returned = true
		if a.finished {
			return
		}
		if a.abandoned || !a.attachmentCurrent() || e.inputDeliveryID != delivery {
			a.abandoned = true
			return
		}
	}
	e.syncFolds()
	if e.folds != nil && e.folds.provider != nil {
		binding, delivery, buffer := e.terminalBinding(), e.inputDeliveryID, e.Buffer
		version, selection := buffer.Version(), buffer.SelectionGeneration()
		e.foldControls(c)
		if e.terminalBinding() != binding || e.inputDeliveryID != delivery || e.Buffer != buffer || buffer.Version() != version || buffer.SelectionGeneration() != selection {
			a.abandoned = true
			return
		}
	}
	if e.compositionMulti {
		status := text.ErrCompositionPending
		if e.compositionError != nil {
			status = e.compositionError
		}
		ui.Text(c.Key("multi-input-status"), status.Error()).FontSize(11)
	}
	line, column, _ := e.Buffer.LineColumn(e.Buffer.Selection().Head)
	span, _ := e.Buffer.LineRange(line)
	if !e.hiddenLine(line) && span.End()-span.Start() > e.shapingLimit() {
		l, ok := a.line(line, column)
		if !ok {
			return
		}
		ui.Row(c.Key("segment-controls")).Wrap().Padding(4, 8).Gap(6).Children(func() {
			label := fmt.Sprintf("UTF-16 位置 %d–%d / %d", l.windowStart+1, l.windowStart+ui.UTF16Len(l.value), l.totalUnits)
			if e.ShapeLimit > 0 {
				label = fmt.Sprintf("位置 %d–%d / %d", l.windowStart+1, l.windowStart+ui.UTF16Len(l.value), l.totalUnits)
			}
			ui.Text(c, label).FontSize(11)
			if ui.Button(c, "上一段").KeepFocus().Disabled(l.windowStart == 0).Clicked() && !a.rejectedMulti() {
				e.segmentInputAction(a, span.Start()+max(0, l.windowStart-e.segmentSize()), line)
			}
			if ui.Button(c, "下一段").KeepFocus().Disabled(l.windowStart+ui.UTF16Len(l.value) >= l.totalUnits).Clicked() && !a.rejectedMulti() {
				e.segmentInputAction(a, l.start+ui.UTF16Len(l.value), line)
			}
		})
	}
	client := e.nativeTextClient()
	box := ui.Box(c.Key("editor-input")).Grow(1).MinHeight(e.MinHeight).MinWidth(80).Clip().
		Label(e.Label).Role(ui.RoleTextField).Focusable().
		Bind(&e.handle).HandleInput(client.handleInput).Draw(e.Draw)
	if !e.ReadOnly {
		box.HandleTextInput(client)
	}
	// A focused editor owns cancellation while input is pending. Register on
	// the element so a host's window Escape shortcut cannot swallow it, even
	// when the first refused preedit has no marked range yet.
	if (e.composing || e.compositionIssue() != nil) && box.Shortcut(0, ui.KeyEscape) {
		e.cancelInputAction(a)
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

func (e *Editor) Focus() {
	if !e.rejectPendingMulti() {
		e.navigationGeneration++
		e.handle.Focus()
	}
}

func (e *Editor) CancelFocus() {
	if !e.rejectPendingMulti() {
		e.navigationGeneration++
		e.handle.CancelFocus()
	}
}

func (e *Editor) ReleaseView() {
	a := e.beginInputAction()
	if e.composing && !e.compositionMulti && e.compositionIssue() == nil {
		a.fail(e.settleSingleInput(a))
	}
	e.finishInputRelease(a)
	a.finish()
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
	return line, column, e.visualLineOf(line, column)
}

func (e *Editor) SetPosition(line, column int) error {
	if err := e.compositionIssue(); err != nil {
		return err
	}
	a := e.beginInputAction()
	if err := e.settleSingleInput(a); err != nil {
		return err
	}
	e.hasDesiredX = false
	e.affinity = ui.TextDownstream
	line = max(0, min(line, e.Buffer.LineCount()-1))
	span, _ := e.Buffer.LineRange(line)
	column = max(0, min(column, span.End()-span.Start()))
	e.revealFolds(span.Start()+column, span.Start()+column, false)
	r := e.normalized(ui.TextInputRange{Start: span.Start() + column, End: span.Start() + column})
	if err := e.Buffer.SetSelection(r.Start, r.Start); err != nil {
		return err
	}
	e.prepareInputSelection()
	a.finish()
	return nil
}

func (e *Editor) rowHeight() float32 { return e.FontSize * 1.65 }
func (e *Editor) gutter() float32 {
	folding := e.folds != nil && e.folds.provider != nil
	if e.LineNumbers {
		if folding {
			return 114
		}
		return 64
	}
	if folding {
		return 74
	}
	return 24
}

func (e *Editor) fail(err error) {
	if err != nil && e.OnError != nil {
		e.OnError(err)
	}
}

func (e *Editor) changed() {
	a := e.beginInputAction()
	a.preparedChange()
	a.finish()
}

func (e *Editor) normalized(r ui.TextInputRange) ui.TextInputRange {
	return normalizedBufferRange(e.Buffer, r)
}

// Geometry preparation uses the fixed, admitted Buffer rather than a live
// Editor attachment, which a synchronous host callback may replace.
func normalizedBufferRange(buffer *text.Buffer, r ui.TextInputRange) ui.TextInputRange {
	r.Start = max(0, min(r.Start, buffer.LenUTF16()))
	r.End = max(r.Start, min(r.End, buffer.LenUTF16()))
	if _, err := buffer.SliceUTF16(r.Start, r.Start); err != nil && r.Start > 0 {
		r.Start--
	}
	if _, err := buffer.SliceUTF16(r.End, r.End); err != nil && r.End < buffer.LenUTF16() {
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
	return e.marked, e.compositionMarked && e.compositionBuffer == e.Buffer && e.Buffer.OwnsComposition(e.compositionToken)
}

// CompositionActive includes pending multi-selection input without a live mark.
func (e *Editor) CompositionActive() bool {
	if e == nil {
		return false
	}
	if e.compositionMulti {
		return true
	}
	_, active := e.MarkedRange()
	return active
}

// MultiCompositionPending reports retained input awaiting native final text or Esc.
func (e *Editor) MultiCompositionPending() bool {
	return e != nil && e.compositionMulti
}

// A refused IME update is not permission to commit the preceding preedit.
// Retain only its small error and owner, never the rejected incoming string.
func (e *Editor) compositionIssue() error {
	if err := e.pendingMultiIssue(); err != nil {
		return err
	}
	if e.compositionError != nil {
		return e.compositionError
	}
	return e.compositionOwnerIssue(false)
}

func (e *Editor) retainCompositionError(err error) error {
	if errors.Is(err, text.ErrSizeLimit) {
		if e.compositionBuffer == nil {
			e.compositionBuffer = e.Buffer
			e.compositionGeneration++
			e.compositionStamp = e.inputStamp()
			e.compositionChangeStamp = e.Buffer.ChangeStamp()
		}
		err = fmt.Errorf("%w。输入法内容尚未接受，请缩短后重试，或按 Esc 取消本次输入", err)
		e.compositionError = err
		e.dragging = false
	}
	if e.compositionMulti {
		e.compositionError = err
		e.services.Invalidate()
	}
	return err
}
func (e *Editor) refuseComposition(err error) {
	a := e.beginInputAction()
	a.fail(e.retainCompositionError(err))
	a.finish()
}

func (e *Editor) cancelComposition() bool {
	if e.compositionBuffer == nil {
		return false
	}
	a := e.beginInputAction()
	handled := e.cancelInputAction(a)
	a.finish()
	return handled
}
func (e *Editor) cancelInputAction(a *inputAction) bool {
	if e.compositionBuffer == nil {
		return false
	}
	owner, composed := e.compositionBuffer, e.composing
	version := owner.Version()
	proved := owner == a.origin.buffer && a.attachmentCurrent() && e.compositionChangeStamp.BaselineID == owner.ChangeStamp().BaselineID
	if !composed {
		proved = proved && e.compositionStamp == e.inputStamp()
	}
	var err error
	if composed {
		err = owner.CancelCompositionGroup(e.compositionToken)
	}
	proved = proved && err == nil
	if e.compositionMulti {
		e.nativeClient = nil
	}
	e.clearComposition()
	a.reason = InputOwnCancel
	a.retired, a.unproved = proved, !proved
	if owner.Version() != version {
		a.recovery = InputRecoveryRollback
	}
	if composed && err == nil && owner == e.Buffer {
		a.preparedChange()
	} else {
		e.services.Invalidate()
	}
	a.fail(err)
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

// ReplaceText is the programmatic editing entry point. It retains the single
// selection contract, including edits inside an ordinary BeginGroup/EndGroup.
func (e *Editor) ReplaceText(r *ui.TextInputRange, value string) {
	a := e.beginInputAction()
	e.replaceInputAction(a, r, value)
	a.finish()
}
func (e *Editor) replaceText(r *ui.TextInputRange, value string) {
	e.ReplaceText(r, value)
}
func (e *Editor) replaceInputAction(a *inputAction, r *ui.TextInputRange, value string) {
	if a.rejectedMulti() {
		return
	}
	if err := e.compositionOwnerIssue(false); err != nil {
		a.fail(err)
		return
	}
	if e.ReadOnly {
		return
	}
	rr := e.replacement(r)
	if !e.composing && e.Buffer.InGroup() && (e.compositionError != nil || e.hasCollapsedFolds()) {
		a.fail(text.ErrGroup)
		return
	}
	if e.hasCollapsedFolds() {
		if err := e.Buffer.CheckReplaceUTF16(rr.Start, rr.End, value); err != nil {
			if e.composing || e.compositionError != nil {
				err = e.retainCompositionError(err)
			}
			a.fail(err)
			return
		}
		e.revealFolds(rr.Start, rr.End, true)
	}
	before, wasComposing := e.Buffer.Version(), e.composing
	var err error
	if e.composing {
		err = e.Buffer.ReplaceCompositionUTF16(e.compositionToken, rr.Start, rr.End, value)
	} else {
		err = e.Buffer.ReplaceUTF16(rr.Start, rr.End, value)
	}
	if err != nil {
		if e.composing || e.compositionError != nil {
			err = e.retainCompositionError(err)
		}
		a.fail(err)
		return
	}
	if e.composing {
		if err = e.Buffer.EndCompositionGroup(e.compositionToken); err != nil {
			a.fail(err)
			return
		}
		a.retired, a.reason = true, InputFinal
	}
	e.clearComposition()
	if e.Buffer.Version() != before || wasComposing {
		a.preparedChange()
	} else {
		e.prepareInputSelection()
		a.changed = true
	}
}

// SetMarkedText retains the direct single-selection API. Multi-selection
// preedit and confirmation both belong to the native input path.
func (e *Editor) SetMarkedText(r *ui.TextInputRange, value string, selected ui.TextInputRange) {
	a := e.beginInputAction()
	if !a.rejectedMulti() {
		e.markedInputAction(a, r, value, selected, false)
	}
	a.finish()
}
func (e *Editor) setMarkedText(r *ui.TextInputRange, value string, selected ui.TextInputRange, native bool) {
	a := e.beginInputAction()
	e.markedInputAction(a, r, value, selected, native)
	a.finish()
}
func (e *Editor) markedInputAction(a *inputAction, r *ui.TextInputRange, value string, selected ui.TextInputRange, native bool) {
	if err := e.compositionOwnerIssue(native); err != nil {
		a.fail(err)
		return
	}
	if e.ReadOnly {
		return
	}
	if !e.composing {
		if e.Buffer.InGroup() {
			a.fail(text.ErrGroup)
			return
		}
		if len(e.Buffer.SelectionSet().Ranges) > 1 {
			if !native {
				a.fail(text.ErrComposition)
				return
			}
			if err := e.beginComposition(true); err != nil {
				a.fail(err)
				return
			}
		}
	}
	var rr ui.TextInputRange
	if e.compositionMulti {
		rr = ui.TextInputRange{Start: e.compositionPrimary.Start(), End: e.compositionPrimary.End()}
		if e.compositionMarked {
			rr = e.marked
		}
		if r != nil && *r != rr {
			a.fail(e.retainCompositionError(fmt.Errorf("%w：原生预编辑必须替换完整的主选区或标记范围", text.ErrRange)))
			return
		}
		e.compositionTargetAdmitted = true
	} else {
		rr = e.replacement(r)
	}
	if err := e.Buffer.CheckReplaceUTF16(rr.Start, rr.End, value); err != nil {
		a.fail(e.retainCompositionError(err))
		return
	}
	started := !e.composing
	if started {
		if err := e.beginComposition(false); err != nil {
			a.fail(err)
			return
		}
	}
	e.revealFolds(rr.Start, rr.End, true)
	version := e.Buffer.Version()
	if err := e.Buffer.ReplaceCompositionUTF16(e.compositionToken, rr.Start, rr.End, value); err != nil {
		var cancelErr error
		if started {
			cancelErr = e.compositionBuffer.CancelCompositionGroup(e.compositionToken)
			e.clearComposition()
		}
		refusal := e.retainCompositionError(err)
		a.fail(cancelErr)
		a.fail(refusal)
		return
	}
	e.compositionError, e.compositionMarked = nil, true
	length := ui.UTF16Len(value)
	e.marked = ui.TextInputRange{Start: rr.Start, End: rr.Start + length}
	selectedRange := e.normalized(ui.TextInputRange{Start: e.marked.Start + max(0, min(selected.Start, length)), End: e.marked.Start + max(0, min(selected.End, length))})
	err := e.Buffer.SetSelection(selectedRange.Start, selectedRange.End)
	e.compositionStamp = e.inputStamp()
	a.reason = InputPreedit
	if version != e.Buffer.Version() {
		a.recovery = InputRecoveryProvisional
	}
	a.preparedChange()
	a.fail(err)
}

func (e *Editor) UnmarkText() {
	if e.rejectPendingMulti() || e.compositionIssue() != nil || !e.composing {
		return
	}
	a := e.beginInputAction()
	a.fail(e.settleSingleInput(a))
	a.finish()
}

// Preserve the historical private entry as a callback-free helper. Complete
// public actions pass their own record to settleSingleInput instead.
func (e *Editor) endSingleComposition() error {
	return e.settleSingleInput(e.captureInputAction())
}

// Flush never settles a foreign group or pending/refused composition. An idle
// repeat is observational and produces neither a delivery nor metadata writes.
func (e *Editor) Flush() error {
	if err := e.compositionIssue(); err != nil {
		return err
	}
	if !e.composing {
		return nil
	}
	a := e.beginInputAction()
	err := e.settleSingleInput(a)
	a.finish()
	return err
}

func (e *Editor) selectTo(pos int, extend bool) {
	a := e.beginInputAction()
	e.selectInputAction(a, pos, extend)
	a.finish()
}
func (e *Editor) selectInputAction(a *inputAction, pos int, extend bool) {
	if err := e.compositionIssue(); err != nil {
		a.fail(err)
		return
	}
	r := e.normalized(ui.TextInputRange{Start: pos, End: pos})
	pos = r.Start
	anchor := pos
	if extend {
		anchor = e.Buffer.Selection().Anchor
	}
	e.revealFolds(pos, pos, false)
	if err := e.Buffer.SetSelection(anchor, pos); err != nil {
		a.fail(err)
		return
	}
	e.prepareInputSelection()
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
	a := e.beginInputAction()
	e.indentInputAction(a, direction)
	a.finish()
}
func (e *Editor) indentInputAction(a *inputAction, direction int) {
	if a.rejectedMulti() {
		return
	}
	startLine, endLine := e.selectedLines()
	if endLine < startLine {
		return
	}
	if e.Buffer.InGroup() && !e.composing {
		a.fail(text.ErrGroup)
		return
	}
	if direction > 0 && endLine-startLine >= (e.Buffer.MaxBytes()-e.Buffer.LenBytes())/len(tabWidth) {
		a.fail(&text.SizeLimitError{Limit: e.Buffer.MaxBytes()})
		return
	}
	if err := e.settleSingleInput(a); err != nil {
		a.fail(err)
		return
	}
	selection := e.Buffer.Selection()
	anchor, head, changed := selection.Anchor, selection.Head, false
	if err := e.Buffer.BeginGroup(); err != nil {
		a.fail(err)
		return
	}
	mapOffset := func(offset, start, removed, added int) int {
		if offset < start {
			return offset
		}
		return max(start, offset-removed) + added
	}
	for line := startLine; line <= endLine; line++ {
		span, err := e.Buffer.LineRange(line)
		if err != nil {
			e.Buffer.CancelGroup()
			a.fail(err)
			return
		}
		pos, remove, insert := span.Start(), 0, tabWidth
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
		if !changed && e.hasCollapsedFolds() {
			first, _ := e.Buffer.OffsetOfLine(startLine)
			last, _ := e.Buffer.LineRange(endLine)
			e.revealFolds(first, last.End(), true)
		}
		if err := e.Buffer.ReplaceUTF16(pos, pos+remove, insert); err != nil {
			e.Buffer.CancelGroup()
			a.fail(err)
			return
		}
		changed = true
		anchor, head = mapOffset(anchor, pos, remove, len(insert)), mapOffset(head, pos, remove, len(insert))
	}
	if err := e.Buffer.SetSelection(anchor, head); err != nil {
		e.Buffer.CancelGroup()
		a.fail(err)
		return
	}
	e.Buffer.EndGroup()
	if changed {
		a.preparedChange()
	}
}

func (e *Editor) insertNewlineWithIndent() {
	a := e.beginInputAction()
	e.newlineInputAction(a)
	a.finish()
}
func (e *Editor) newlineInputAction(a *inputAction) {
	if a.rejectedMulti() {
		return
	}
	selection := e.Buffer.Selection()
	position := selection.Start()
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
	value := newline + indentation.String()
	if e.composing {
		if err := e.Buffer.CheckReplaceUTF16(selection.Start(), selection.End(), value); err != nil {
			a.fail(err)
			return
		}
	}
	if err := e.settleSingleInput(a); err != nil {
		a.fail(err)
		return
	}
	e.replaceInputAction(a, nil, value)
}

func (e *Editor) clipboardIssue(target clipboardTarget) error {
	if err := e.compositionIssue(); err != nil {
		return err
	}
	if !target.matches(e) {
		return errors.New("文档、选区或输入状态已改变，已停止剪贴板操作")
	}
	return nil
}

func (e *Editor) finishClipboard(request *clipboardRequest, err error) {
	if e.pendingClipboard != request || !request.claimCompletion() {
		return
	}
	a := e.beginInputAction()
	e.finishClipboardInput(a, request, err)
	a.finish()
}

func (e *Editor) finishClipboardInput(a *inputAction, request *clipboardRequest, err error) {
	if e.pendingClipboard != request {
		a.abandoned = true
		return
	}
	e.pendingClipboard = nil
	if err == nil && !request.cut {
		return
	}
	// A delayed result owns a fresh action but retains the originating
	// request's notification seal and handlers.
	a.origin, a.onChange, a.onError = request.origin, request.onChange, request.onError
	if err != nil {
		a.failClipboard(err)
		return
	}
	if err := e.clipboardIssue(request.target); err != nil {
		a.failClipboard(err)
		return
	}
	if request.fold.buffer != nil && !sameFoldSource(request.fold, e.foldIdentity()) {
		a.failClipboard(ErrFoldStale)
		return
	}
	e.replaceInputAction(a, nil, "")
}

func (e *Editor) Command(command string) bool {
	a := e.beginInputAction()
	handled := e.commandInputAction(a, command)
	a.finish()
	return handled
}
func (e *Editor) commandInputAction(a *inputAction, command string) bool {
	if err := e.compositionIssue(); err != nil {
		a.fail(err)
		return true
	}
	selection := e.Buffer.Selection()
	switch command {
	case "selectAll":
		if err := e.Buffer.SetSelection(0, e.Buffer.LenUTF16()); err != nil {
			a.fail(err)
		} else if e.composing {
			e.compositionStamp = e.inputStamp()
		}
		e.prepareInputSelection()
		return true
	case "copy", "cut":
		if selection.Start() == selection.End() {
			return true
		}
		if e.pendingClipboard != nil {
			a.fail(errors.New("上一次复制尚未完成，请稍后重试"))
			return true
		}
		if e.Buffer.InGroup() && !e.composing {
			a.fail(text.ErrGroup)
			return true
		}
		value, err := e.Buffer.SliceUTF16(selection.Start(), selection.End())
		if err != nil {
			a.fail(err)
			return true
		}
		if err := e.settleSingleInput(a); err != nil {
			a.fail(err)
			return true
		}
		request := &clipboardRequest{
			value: value, cut: command == "cut" && !e.ReadOnly, target: e.clipboardTarget(),
			origin: a.origin, onChange: a.onChange, onError: a.onError,
		}
		if e.folds != nil && e.folds.provider != nil {
			request.fold = e.foldIdentity()
		}
		e.pendingClipboard = request
		copy, copyThen := e.Copy, e.CopyThen
		delivery := e.inputDeliveryID
		if copy != nil {
			err := copy(value)
			if e.inputDeliveryID != delivery {
				a.suppressPublish = true
			}
			e.finishClipboardInput(a, request, err)
		} else if copyThen != nil {
			request.dispatched = true
			copyThen(value, func(err error) {
				if a.returned {
					e.finishClipboard(request, err)
					return
				}
				if !request.claimCompletion() {
					return
				}
				if e.inputDeliveryID != delivery {
					a.suppressPublish = true
				}
				e.finishClipboardInput(a, request, err)
				if a.changed || a.errorCount != 0 {
					a.finish()
				}
			})
			a.returned = true
			if a.finished {
				return true
			}
			if e.inputDeliveryID != delivery {
				a.suppressPublish = true
			}
			if e.pendingClipboard != nil && e.pendingClipboard != request {
				a.abandoned = true
			}
		} else {
			e.services.Invalidate()
		}
		return true
	case "paste", "pasteAndMatchStyle":
		if e.ReadOnly {
			return true
		}
		if e.Buffer.InGroup() && !e.composing {
			a.fail(text.ErrGroup)
			return true
		}
		target, delivery := e.clipboardTarget(), e.inputDeliveryID
		var value string
		var err error
		if paste := e.Paste; paste != nil {
			value, err = paste()
		} else {
			value = e.services.ReadClipboard()
		}
		if e.inputDeliveryID != delivery {
			a.suppressPublish = true
		}
		if err != nil {
			a.failClipboard(err)
			return true
		}
		if err := e.clipboardIssue(target); err != nil {
			a.failClipboard(err)
			return true
		}
		e.replaceInputAction(a, nil, value)
		return true
	case "undo", "redo":
		if e.ReadOnly {
			return true
		}
		if err := e.settleSingleInput(a); err != nil {
			a.fail(err)
			return true
		}
		changed := false
		if command == "undo" {
			changed = e.Buffer.Undo()
		} else {
			changed = e.Buffer.Redo()
		}
		if changed {
			a.preparedChange()
		}
		return true
	case "delete":
		if !e.ReadOnly {
			e.deleteInputAction(a, 1)
		}
		return true
	case "moveLinesUp":
		e.editLinesInputAction(a, text.MoveLinesUp)
		return true
	case "moveLinesDown":
		e.editLinesInputAction(a, text.MoveLinesDown)
		return true
	case "copyLinesUp":
		e.editLinesInputAction(a, text.CopyLinesUp)
		return true
	case "copyLinesDown":
		e.editLinesInputAction(a, text.CopyLinesDown)
		return true
	case "indent":
		if !e.ReadOnly {
			e.indentInputAction(a, 1)
		}
		return true
	case "outdent":
		if !e.ReadOnly {
			e.indentInputAction(a, -1)
		}
		return true
	}
	return false
}

func (e *Editor) delete(dir int) {
	a := e.beginInputAction()
	e.deleteInputAction(a, dir)
	a.finish()
}
func (e *Editor) deleteInputAction(a *inputAction, dir int) {
	if a.rejectedMulti() {
		return
	}
	if err := e.settleSingleInput(a); err != nil {
		a.fail(err)
		return
	}
	selection := e.Buffer.Selection()
	r := ui.TextInputRange{Start: selection.Start(), End: selection.End()}
	if r.Start == r.End {
		next := a.boundary(selection.Head, dir)
		r.Start, r.End = min(selection.Head, next), max(selection.Head, next)
	}
	if a.errorCount == 0 {
		e.replaceInputAction(a, &r, "")
	}
}

func (e *Editor) HandleInput(ev ui.InputEvent) bool {
	a := e.beginInputAction()
	handled := e.handleInputAction(a, ev)
	a.finish()
	return handled
}
func (e *Editor) handleInputAction(a *inputAction, ev ui.InputEvent) bool {
	switch ev.Kind {
	case ui.InputText:
		e.nativeInputAction(a, nil, ev.Text)
		return true
	case ui.InputCompose:
		length := ui.UTF16Len(ev.Text)
		e.markedInputAction(a, nil, ev.Text, ui.TextInputRange{Start: length, End: length}, true)
		return true
	case ui.InputCommand:
		return e.commandInputAction(a, ev.Text)
	case ui.InputKeyDown:
		if ev.Key == ui.KeyEscape && ev.Mods == 0 && e.cancelInputAction(a) {
			return true
		}
		if a.rejectedMulti() {
			return true
		}
	case ui.InputPointerDown, ui.InputPointerUp, ui.InputPointerMove, ui.InputSelect:
		if a.rejectedMulti() {
			return true
		}
	}
	if e.foldPointer(ev) {
		return true
	}
	selection := e.Buffer.Selection()
	switch ev.Kind {
	case ui.InputPointerDown:
		if ev.Button != 0 {
			return false
		}
		if err := e.compositionIssue(); err != nil {
			a.fail(err)
			return true
		}
		if err := e.settleSingleInput(a); err != nil {
			a.fail(err)
			return true
		}
		position, ok := a.point(ui.Point{X: ev.X, Y: ev.Y})
		if !ok {
			return false
		}
		e.hasDesiredX = false
		e.affinity = position.Affinity
		pos := position.Index
		e.selectInputAction(a, pos, ev.Mods&ui.Shift != 0)
		e.dragging = true
		if ev.Clicks == 2 && a.insideWord(pos) {
			a.fail(e.Buffer.SetSelection(a.word(pos, -1, true), a.word(pos, 1, true)))
		} else if ev.Clicks >= 2 {
			a.fail(e.Buffer.SetSelection(a.word(pos, -1, false), a.word(pos, 1, false)))
		}
		return true
	case ui.InputPointerMove:
		if e.compositionIssue() != nil {
			return true
		}
		if e.dragging {
			position, ok := a.point(ui.Point{X: ev.X, Y: ev.Y})
			if ok {
				e.affinity = position.Affinity
				e.selectInputAction(a, position.Index, true)
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
		if err := e.compositionIssue(); err != nil {
			a.fail(err)
			return true
		}
		if ev.Mods == ui.Alt || ev.Mods == ui.Alt|ui.Shift {
			switch ev.Key {
			case ui.KeyUp:
				if ev.Mods&ui.Shift != 0 {
					return e.commandInputAction(a, "copyLinesUp")
				}
				return e.commandInputAction(a, "moveLinesUp")
			case ui.KeyDown:
				if ev.Mods&ui.Shift != 0 {
					return e.commandInputAction(a, "copyLinesDown")
				}
				return e.commandInputAction(a, "moveLinesDown")
			}
		}
		cmd, extend := ev.Mods&ui.Cmd != 0, ev.Mods&ui.Shift != 0
		if cmd {
			switch ev.Key {
			case ui.KeyA:
				return e.commandInputAction(a, "selectAll")
			case ui.KeyC:
				return e.commandInputAction(a, "copy")
			case ui.KeyX:
				return e.commandInputAction(a, "cut")
			case ui.KeyV:
				return e.commandInputAction(a, "paste")
			case ui.KeyZ:
				if extend {
					return e.commandInputAction(a, "redo")
				}
				return e.commandInputAction(a, "undo")
			case ui.KeyY:
				return e.commandInputAction(a, "redo")
			}
		}
		switch ev.Key {
		case ui.KeyLeft, ui.KeyRight:
			if err := e.settleSingleInput(a); err != nil {
				a.fail(err)
				return true
			}
			e.hasDesiredX, e.affinity = false, ui.TextDownstream
			dir := 1
			if ev.Key == ui.KeyLeft {
				dir = -1
			}
			next := a.boundary(selection.Head, dir)
			if !extend && selection.Start() != selection.End() {
				if dir < 0 {
					next = selection.Start()
				} else {
					next = selection.End()
				}
			} else if cmd {
				next = a.word(selection.Head, dir, false)
			}
			if a.errorCount == 0 {
				e.selectInputAction(a, next, extend)
			}
			return true
		case ui.KeyUp, ui.KeyDown, ui.KeyPageUp, ui.KeyPageDown:
			if err := e.settleSingleInput(a); err != nil {
				a.fail(err)
				return true
			}
			delta := 1
			if ev.Key == ui.KeyPageUp || ev.Key == ui.KeyPageDown {
				delta = max(1, int(e.height/e.rowHeight())-1)
			}
			if ev.Key == ui.KeyUp || ev.Key == ui.KeyPageUp {
				delta = -delta
			}
			a.vertical(delta, extend)
			return true
		case ui.KeyHome, ui.KeyEnd:
			if err := e.settleSingleInput(a); err != nil {
				a.fail(err)
				return true
			}
			a.homeEnd(ev.Key == ui.KeyEnd, cmd, extend)
			return true
		case ui.KeyBackspace:
			if !e.ReadOnly {
				e.deleteInputAction(a, -1)
			}
			return true
		case ui.KeyDelete:
			if !e.ReadOnly {
				e.deleteInputAction(a, 1)
			}
			return true
		case ui.KeyEnter:
			if !e.ReadOnly {
				e.newlineInputAction(a)
			}
			return true
		case ui.KeyTab:
			if e.ReadOnly {
				return false
			}
			if extend {
				e.commandInputAction(a, "outdent")
			} else if selection.Start() != selection.End() {
				e.commandInputAction(a, "indent")
			} else {
				e.replaceInputAction(a, nil, tabWidth)
			}
			return true
		}
	}
	return false
}

func (e *Editor) Reveal(start, end int) {
	a := e.beginInputAction()
	e.revealInputAction(a, start, end)
	a.finish()
}
func (e *Editor) revealInputAction(a *inputAction, start, end int) {
	if a.rejectedMulti() {
		return
	}
	if err := e.compositionIssue(); err != nil {
		e.navigationGeneration++
		e.handle.Focus()
		a.fail(err)
		return
	}
	if _, err := e.Buffer.SliceUTF16(start, start); err != nil {
		e.navigationGeneration++
		e.handle.Focus()
		a.fail(err)
		return
	}
	if _, err := e.Buffer.SliceUTF16(end, end); err != nil {
		e.navigationGeneration++
		e.handle.Focus()
		a.fail(err)
		return
	}
	e.revealFolds(start, end, false)
	e.revealFolds(end, end, false)
	if err := e.Buffer.SetSelection(start, end); err != nil {
		a.fail(err)
	} else {
		e.hasDesiredX = false
	}
	e.prepareInputSelection()
	e.navigationGeneration++
	e.handle.Focus()
}
