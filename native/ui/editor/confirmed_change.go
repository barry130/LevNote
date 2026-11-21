package editor

import (
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

// InputBinding is an opaque registration and attachment seal. Setter/view and
// registration nonces prevent observed ABA without comparing callback addresses.
type InputBinding struct {
	editor                                              *Editor
	buffer                                              *text.Buffer
	bufferEpoch, readonlyEpoch, viewEpoch, registration *inputTerminalEpoch
	readOnly                                            bool
	view                                                uint64
}
type inputTerminalEpoch struct{ marker byte }

type InputDeliveryReason uint8

const (
	InputAccepted InputDeliveryReason = iota
	InputPreedit
	InputFinal
	InputOwnCancel
	InputSettled
	InputReleased
)

type InputRecoveryEffect uint8

const (
	InputRecoveryNone InputRecoveryEffect = iota
	InputRecoveryProvisional
	InputRecoveryRollback
)

type InputChangeDisposition uint8

const (
	InputSame InputChangeDisposition = iota
	InputRealCommit
	InputRebaseInvalidation
	InputUnprovedRetirement
)

// InputNotificationID is local notification identity, not a text version.
// Its opaque epoch also makes serial rollover safe for retained records.
type InputNotificationID struct {
	epoch  *inputTerminalEpoch
	serial uint64
}

type InputDelivery struct {
	Binding          InputBinding
	Editor           *Editor
	Buffer           *text.Buffer
	Stamp            text.ChangeStamp
	Action, Delivery InputNotificationID
	Reason           InputDeliveryReason
	Recovery         InputRecoveryEffect
	Disposition      InputChangeDisposition
	Failed           bool
}

func (e *Editor) observeTerminalAttachment() {
	if e.terminalBufferEpoch == nil || e.terminalObservedBuffer != e.Buffer {
		e.terminalBufferEpoch = &inputTerminalEpoch{marker: 1}
		e.terminalObservedBuffer = e.Buffer
	}
	if e.terminalReadonlyEpoch == nil || e.terminalObservedReadonly != e.ReadOnly {
		e.terminalReadonlyEpoch = &inputTerminalEpoch{marker: 1}
		e.terminalObservedReadonly = e.ReadOnly
	}
	if e.terminalViewEpoch == nil || e.terminalObservedView != e.viewGeneration {
		e.terminalViewEpoch = &inputTerminalEpoch{marker: 1}
		e.terminalObservedView = e.viewGeneration
	}
}
func (e *Editor) terminalBinding() InputBinding {
	return InputBinding{editor: e, buffer: e.Buffer, readOnly: e.ReadOnly, view: e.viewGeneration,
		bufferEpoch: e.terminalBufferEpoch, readonlyEpoch: e.terminalReadonlyEpoch,
		viewEpoch: e.terminalViewEpoch, registration: e.terminalRegistration}
}

// BindInputTerminal must be called on the UI thread, including for revocation.
// nil revokes the old registration. Priming emits no event or text snapshot.
func (e *Editor) BindInputTerminal(fn func(InputDelivery)) InputBinding {
	e.observeInputOwner()
	e.observeTerminalAttachment()
	e.terminalRegistration = &inputTerminalEpoch{marker: 1}
	e.inputTerminal = fn
	e.inputDelivery = InputDelivery{}
	e.inputDeliveryValid = false
	return e.terminalBinding()
}
func (e *Editor) InputBindingCurrent(binding InputBinding) bool {
	return e != nil && binding.editor == e && binding.registration != nil && binding.buffer != nil &&
		binding == e.terminalBinding()
}
func (e *Editor) LastInputDelivery() (InputDelivery, bool) {
	if e == nil {
		return InputDelivery{}, false
	}
	return e.inputDelivery, e.inputDeliveryValid
}

// inputAction is explicitly shared by internal helpers. Public reentry creates
// a fresh record; there is no Editor action-depth stack or deferred observer list.
// Fixed error slots cover the cleanup/refusal pair and View warning/control
// sites; they retain errors, never a deferred list of observer callbacks.
type inputAction struct {
	editor                                         *Editor
	origin                                         InputBinding
	before                                         text.ChangeStamp
	onChange                                       func()
	onError                                        func(error)
	terminal                                       func(InputDelivery)
	reason                                         InputDeliveryReason
	recovery                                       InputRecoveryEffect
	changed, retired, unproved                     bool
	finished, abandoned, returned, suppressPublish bool
	errors                                         [4]error
	errorCount                                     int
	errorOwned, legacyClipboardError               bool
	action                                         InputNotificationID
	releasedView                                   *inputTerminalEpoch
	releasedGeneration                             uint64
}

func (e *Editor) beginInputAction() *inputAction {
	e.observeTerminalAttachment()
	return e.captureInputAction()
}

// captureInputAction itself is observational, also on B3 refusal/no-op paths.
func (e *Editor) captureInputAction() *inputAction {
	a := &inputAction{editor: e, origin: e.terminalBinding(), onChange: e.OnChange, onError: e.OnError,
		terminal: e.inputTerminal, reason: InputAccepted}
	if e.Buffer != nil {
		a.before = e.Buffer.ChangeStamp()
	}
	return a
}
func (a *inputAction) fail(err error) {
	if err == nil {
		return
	}
	if a.errorCount == 0 {
		a.errorOwned = a.canReport()
	}
	if a.errorCount >= len(a.errors) {
		panic("editor: unexpected error count in one complete input action")
	}
	a.errors[a.errorCount] = err
	a.errorCount++
}

func (a *inputAction) failClipboard(err error) {
	if err != nil && a.errorCount == 0 {
		a.legacyClipboardError = true
	}
	a.fail(err)
}
func (a *inputAction) rejectedMulti() bool {
	if err := a.editor.pendingMultiIssue(); err != nil {
		a.fail(err)
		return true
	}
	return false
}
func (a *inputAction) attachmentCurrent() bool {
	e := a.editor
	now := e.terminalBinding()
	// Observer replacement revokes notification authority, not mutation authority.
	now.registration = a.origin.registration
	return now == a.origin
}
func (a *inputAction) notificationCurrent() bool {
	if a.abandoned {
		return false
	}
	e := a.editor
	current := e.terminalBinding()
	// A complete ReleaseView may deliver to the original binding once. Its
	// captured released view is the only part of that binding we retire here.
	if a.reason == InputReleased {
		if current.viewEpoch != a.releasedView || current.view != a.releasedGeneration {
			return false
		}
		current.view, current.viewEpoch = a.origin.view, a.origin.viewEpoch
	}
	return current == a.origin
}

func (a *inputAction) canReport() bool {
	if a.notificationCurrent() {
		return true
	}
	// Never-bound clipboard callers still receive an explicit error from
	// their captured legacy handler when a setter round trip stales a target.
	// This exception does not authorize a delivery or an accepted change.
	if a.abandoned || !a.legacyClipboardError || a.origin.registration != nil {
		return false
	}
	current := a.editor.terminalBinding()
	return current.registration == nil && current.editor == a.origin.editor && current.buffer == a.origin.buffer &&
		current.readOnly == a.origin.readOnly && current.view == a.origin.view && current.viewEpoch == a.origin.viewEpoch
}
func (a *inputAction) hasCompletion() bool {
	return a.changed || a.retired || a.errorCount != 0 || a.unproved
}
func (a *inputAction) preparedChange() {
	e := a.editor
	e.invalidateCacheIfChanged()
	e.hasDesiredX = false
	e.affinity = ui.TextDownstream
	e.blinkStart = e.frameNow
	e.revealCaret = true
	e.services.Invalidate()
	a.changed = true
}
func (a *inputAction) prepareSelection() {
	a.editor.revealCaret = true
	a.editor.services.Invalidate()
}
func nextInputNotification(id *InputNotificationID) InputNotificationID {
	if id.epoch == nil || id.serial == ^uint64(0) {
		id.epoch = &inputTerminalEpoch{marker: 1}
		id.serial = 0
	}
	id.serial++
	return *id
}
func inputDisposition(before, after text.ChangeStamp, unproved bool) InputChangeDisposition {
	if before.BaselineID != after.BaselineID {
		return InputRebaseInvalidation
	}
	if unproved {
		return InputUnprovedRetirement
	}
	if before.ConfirmedChangeID != after.ConfirmedChangeID {
		return InputRealCommit
	}
	return InputSame
}
func (a *inputAction) publish(err error) (InputDelivery, bool) {
	if a.suppressPublish || !a.notificationCurrent() || a.origin.buffer == nil {
		return InputDelivery{}, false
	}
	e := a.editor
	if a.action == (InputNotificationID{}) {
		a.action = nextInputNotification(&e.inputActionID)
	}
	delivery := InputDelivery{Binding: a.origin, Editor: e, Buffer: a.origin.buffer, Stamp: a.origin.buffer.ChangeStamp(),
		Action: a.action, Delivery: nextInputNotification(&e.inputDeliveryID), Reason: a.reason, Recovery: a.recovery, Failed: err != nil}
	delivery.Disposition = inputDisposition(a.before, delivery.Stamp, a.unproved)
	e.inputDelivery, e.inputDeliveryValid = delivery, true
	return delivery, true
}

// finish is the only notifying tail. Every Editor write precedes it. A caller
// may inspect only action-local values afterward; a repeated finish is inert.
func (a *inputAction) finish() {
	if a.finished {
		return
	}
	a.finished = true
	if a.abandoned {
		return
	}
	reportErrors := a.errorCount != 0 && a.errorOwned && a.canReport()
	var failure error
	if reportErrors {
		failure = a.errors[0]
	}
	var delivery InputDelivery
	var published bool
	if a.changed || reportErrors && (a.retired || a.unproved) {
		delivery, published = a.publish(failure)
	}
	if a.editor.Buffer == nil {
		return
	}
	var callbackBoundary inputGeometryGuard
	if reportErrors {
		callbackBoundary = a.geometryGuard()
	}
	if a.changed {
		// The captured generic callback follows physical attachment ownership.
		// Rebinding the terminal observer alone only revokes its delivery.
		if a.onChange != nil && a.attachmentCurrent() {
			a.onChange()
		}
	} else if a.retired && !a.unproved && a.errorCount == 0 {
		if a.terminal != nil {
			if settled, ok := a.publish(nil); ok {
				a.terminal(settled)
			}
		}
		return
	}
	if !reportErrors || a.onError == nil {
		return
	}
	for i := 0; i < a.errorCount; i++ {
		if !a.canReport() || !callbackBoundary.matches(a.editor) {
			return
		}
		if published {
			latest, ok := a.editor.LastInputDelivery()
			if !ok || latest.Delivery != delivery.Delivery {
				return
			}
		}
		a.onError(a.errors[i])
	}
}

func (e *Editor) settleSingleInput(a *inputAction) error {
	if !e.composing {
		return nil
	}
	owner, token := e.compositionBuffer, e.compositionToken
	if owner == nil || owner != a.origin.buffer {
		a.unproved = true
		return text.ErrComposition
	}
	if err := owner.EndCompositionGroup(token); err != nil {
		a.unproved = true
		return err
	}
	e.clearComposition()
	a.retired, a.reason = true, InputSettled
	e.services.Invalidate()
	return nil
}

// inputGeometryGuard protects actual Highlight callbacks while preparing a
// bounded action-local layout. It does not require idle input: View may render
// its accepted preedit. Notification registration is guarded by nonce, never a
// callback function address. Existing geometry boundaries add their own checks.
type inputGeometryGuard struct {
	binding                        InputBinding
	boundary                       sourceRevealBoundary
	selections                     text.SelectionSet
	fold                           *foldState
	foldGeneration, foldProjection uint64
	actionID, deliveryID           InputNotificationID
}

func (a *inputAction) geometryGuard() inputGeometryGuard {
	e := a.editor
	g := inputGeometryGuard{
		binding: e.terminalBinding(), boundary: e.sourceRevealBoundary(),
		selections: e.Buffer.SelectionSet(), fold: e.folds,
		actionID: e.inputActionID, deliveryID: e.inputDeliveryID,
	}
	if e.folds != nil {
		g.foldGeneration, g.foldProjection = e.folds.generation, e.folds.projection
	}
	return g
}
func (g inputGeometryGuard) matches(e *Editor) bool {
	// Accepted empty/equal input can publish a successor while every text
	// and geometry field still matches. The old prepared plan then expires.
	if g.actionID != e.inputActionID || g.deliveryID != e.inputDeliveryID {
		return false
	}
	if e.Buffer == nil || g.binding != e.terminalBinding() || g.boundary != e.sourceRevealBoundary() || !equalSourceSelectionSet(g.selections, e.Buffer.SelectionSet()) || g.fold != e.folds {
		return false
	}
	return e.folds == nil || g.foldGeneration == e.folds.generation && g.foldProjection == e.folds.projection
}
func (a *inputAction) geometryPlan() *sourceRevealLayout {
	e := a.editor
	// Existing invalidation is callback-free. Never clone or enumerate the
	// historic shape cache; a plan owns only shapes queried by this action.
	e.invalidateCacheIfChanged()
	return &sourceRevealLayout{buffer: e.Buffer, cache: map[int]lineLayout{}, counts: e.rowCounts, tree: e.rowTree,
		countChanges: map[int]int{}, treeChanges: map[int]int{}, folds: cloneSourceFolds(e.folds), indexed: e.Buffer.LineCount(),
		limit: e.shapingLimit(), segment: e.segmentSize(), version: e.Buffer.Version(), softWrap: e.SoftWrap,
		fontSize: e.FontSize, width: e.width, height: e.height, gutter: e.gutter(), scrollX: e.ScrollX, scrollY: e.ScrollY,
		affinity: e.affinity, longWarned: e.longLineWarned}
}
func (a *inputAction) commitGeometry(p *sourceRevealLayout) {
	e := a.editor
	e.geometryGeneration++
	if e.cache == nil {
		e.cache = map[int]lineLayout{}
	}
	for index, line := range p.cache {
		if _, exists := e.cache[index]; exists || len(e.cache) < lineCacheLimit {
			e.cache[index] = line
		}
	}
	if e.rowCounts == nil {
		e.rowCounts = map[int]int{}
	}
	if e.rowTree == nil {
		e.rowTree = map[int]int{}
	}
	for line, count := range p.countChanges {
		if count == 1 {
			delete(e.rowCounts, line)
		} else {
			e.rowCounts[line] = count
		}
	}
	for node, delta := range p.treeChanges {
		e.rowTree[node] += delta
	}
	e.cachedBuffer, e.cachedShapeLimit, e.cachedGutter = p.buffer, p.limit, p.gutter
	e.indexedLines, e.version, e.folds = p.indexed, p.version, p.folds
	e.ScrollX, e.ScrollY, e.longLineWarned = p.scrollX, p.scrollY, p.longWarned
}

// prepareLine reuses accepted sourceRevealLayout/shapeEditorLine storage and
// math, but its guard admits owned preedit and its warning belongs to the action's
// final error boundary. No callback override, copied Editor or live cache tail.
func (a *inputAction) prepareLine(p *sourceRevealLayout, index, column int) (lineLayout, bool) {
	e := a.editor
	if l, ok := p.cache[index]; ok && column >= l.windowStart && column <= l.windowStart+text.UTF16Len(l.value) {
		return l, true
	}
	if l, ok := e.cache[index]; ok && column >= l.windowStart && column <= l.windowStart+text.UTF16Len(l.value) {
		return l, true
	}
	span, err := p.buffer.LineRange(index)
	if err != nil {
		a.fail(err)
		return lineLayout{}, false
	}
	total, window, units := span.End()-span.Start(), 0, p.limit
	if total > p.limit {
		units = p.segment
		window = max(0, min(column, total-1)) / units * units
		if !p.longWarned {
			p.longWarned = true
			a.fail(errLineTooLong)
		}
	}
	r := normalizedBufferRange(p.buffer, ui.TextInputRange{Start: span.Start() + window, End: min(span.End(), span.Start()+window+units)})
	value, err := p.buffer.SliceUTF16(r.Start, r.End)
	if err != nil {
		a.fail(err)
		return lineLayout{}, false
	}
	highlight := e.Highlight
	var spans []ui.Span
	if highlight != nil {
		guard := a.geometryGuard()
		spans = highlight(value)
		if !guard.matches(e) {
			a.abandoned = true
			return lineLayout{}, false
		}
	}
	width := float32(0)
	if p.softWrap && p.width > p.gutter+16 {
		width = p.width - p.gutter - 16
	}
	out := shapeEditorLine(value, p.fontSize, width, spans, highlight != nil)
	out.start, out.windowStart, out.totalUnits = r.Start, r.Start-span.Start(), total
	if len(p.cache) < lineCacheLimit {
		p.cache[index] = out
	}
	if old := p.rowCount(index); old != out.rows {
		p.addRows(index, out.rows-old)
		p.countChanges[index] = out.rows
		if p.folds != nil {
			p.folds.projection++
		}
		p.updateFoldExtras()
	}
	return out, true
}
func (a *inputAction) line(index, column int) (lineLayout, bool) {
	a.editor.invalidateCacheIfChanged()
	if l, ok := a.editor.cache[index]; ok && column >= l.windowStart && column <= l.windowStart+text.UTF16Len(l.value) {
		return l, true
	}
	p := a.geometryPlan()
	guard := a.geometryGuard()
	l, ok := a.prepareLine(p, index, column)
	if !ok {
		return lineLayout{}, false
	}
	if !guard.matches(a.editor) {
		a.abandoned = true
		return lineLayout{}, false
	}
	a.commitGeometry(p)
	return l, true
}

func (a *inputAction) boundary(pos, dir int) int {
	next, err := a.origin.buffer.GraphemeBoundary(pos, dir)
	a.fail(err)
	if err != nil {
		return pos
	}
	return next
}
func (a *inputAction) word(pos, dir int, stopBefore bool) int {
	for n := 0; n < 4096; n++ {
		next := a.boundary(pos, dir)
		if next == pos {
			break
		}
		value, err := a.origin.buffer.SliceUTF16(min(pos, next), max(pos, next))
		if err != nil {
			a.fail(err)
			break
		}
		if strings.ContainsAny(value, " \t\r\n.,;:()[]{}") {
			if !stopBefore {
				pos = next
			}
			break
		}
		pos = next
	}
	return pos
}
func (a *inputAction) insideWord(pos int) bool {
	for _, dir := range []int{-1, 1} {
		next := a.boundary(pos, dir)
		if next == pos {
			return false
		}
		value, err := a.origin.buffer.SliceUTF16(min(pos, next), max(pos, next))
		if err != nil {
			a.fail(err)
			return false
		}
		if strings.ContainsAny(value, " \t\r\n.,;:()[]{}") {
			return false
		}
	}
	return true
}
func (a *inputAction) point(point ui.Point) (ui.TextCaretPosition, bool) {
	p := a.geometryPlan()
	guard := a.geometryGuard()
	y := max(float32(0), point.Y-12+p.scrollY)
	line, _ := p.lineAndRow(int(y / (p.fontSize * 1.65)))
	if line < 0 {
		return ui.TextCaretPosition{}, false
	}
	l, cached := a.editor.cache[line]
	if !cached {
		var ok bool
		l, ok = a.prepareLine(p, line, 0)
		if !ok {
			return ui.TextCaretPosition{}, false
		}
	}
	if !guard.matches(a.editor) {
		a.abandoned = true
		return ui.TextCaretPosition{}, false
	}
	localY := y - float32(p.visualBase(line))*p.fontSize*1.65
	position := l.layout.PositionAt(ui.Point{X: point.X - p.gutter + p.scrollX, Y: localY})
	position.Index += l.start
	a.commitGeometry(p)
	return position, true
}
func inputVisibleLine(p *sourceRevealLayout, line, dir int) int {
	if p.folds == nil {
		return line
	}
	i := sort.Search(len(p.folds.intervals), func(i int) bool { return p.folds.intervals[i].end > line })
	if i < len(p.folds.intervals) && p.folds.intervals[i].start <= line {
		if dir < 0 {
			return p.folds.intervals[i].start - 1
		}
		return p.folds.intervals[i].end
	}
	return line
}
func (a *inputAction) vertical(delta int, extend bool) {
	e := a.editor
	p := a.geometryPlan()
	guard := a.geometryGuard()
	selection := p.buffer.Selection()
	head := selection.Head
	p.revealFolds(head, head)
	line, column, err := p.buffer.LineColumn(head)
	if err != nil {
		a.fail(err)
		return
	}
	l, ok := a.prepareLine(p, line, column)
	if !ok {
		return
	}
	row := l.caretRow(column-l.windowStart, e.affinity)
	x := e.desiredX
	if !e.hasDesiredX {
		x = l.layout.CaretAt(ui.TextCaretPosition{Index: column - l.windowStart, Affinity: e.affinity}).X
	}
	direction := 1
	if delta < 0 {
		direction, delta = -1, -delta
	}
	for ; delta > 0; delta-- {
		row += direction
		if row >= 0 && row < l.rows {
			continue
		}
		if direction > 0 && l.windowStart+text.UTF16Len(l.value) < l.totalUnits {
			l, ok = a.prepareLine(p, line, l.windowStart+text.UTF16Len(l.value)+1)
			row = 0
		} else if direction < 0 && l.windowStart > 0 {
			l, ok = a.prepareLine(p, line, l.windowStart-1)
			row = l.rows - 1
		} else if next := line + direction; next >= 0 && next < p.buffer.LineCount() {
			line = inputVisibleLine(p, next, direction)
			column = 0
			if direction < 0 {
				span, spanErr := p.buffer.LineRange(line)
				if spanErr != nil {
					a.fail(spanErr)
					return
				}
				column = span.End() - span.Start()
			}
			l, ok = a.prepareLine(p, line, column)
			row = 0
			if direction < 0 {
				row = l.rows - 1
			}
		} else {
			row = max(0, min(row, l.rows-1))
			break
		}
		if !ok {
			return
		}
	}
	if !guard.matches(e) {
		a.abandoned = true
		return
	}
	rowBox := l.lines[row].Rect
	position := l.layout.PositionAt(ui.Point{X: x, Y: rowBox.Y + rowBox.H/2})
	pos, anchor := l.start+position.Index, l.start+position.Index
	if extend {
		anchor = selection.Anchor
	}
	if err := p.buffer.SetSelection(anchor, pos); err != nil {
		a.fail(err)
		return
	}
	a.commitGeometry(p)
	e.affinity, e.desiredX, e.hasDesiredX = position.Affinity, x, true
	e.prepareInputSelection()
}
func (a *inputAction) homeEnd(end, whole, extend bool) {
	e := a.editor
	p := a.geometryPlan()
	guard := a.geometryGuard()
	selection := p.buffer.Selection()
	p.revealFolds(selection.Head, selection.Head)
	line, column, err := p.buffer.LineColumn(selection.Head)
	if err != nil {
		a.fail(err)
		return
	}
	l, ok := a.prepareLine(p, line, column)
	if !ok {
		return
	}
	row := l.caretRow(column-l.windowStart, e.affinity)
	pos, affinity := l.start+l.lines[row].Start, ui.TextDownstream
	if end {
		pos, affinity = l.start+l.lines[row].End, ui.TextUpstream
	}
	if whole {
		pos = 0
		if end {
			pos = p.buffer.LenUTF16()
		}
	}
	if !guard.matches(e) {
		a.abandoned = true
		return
	}
	anchor := pos
	if extend {
		anchor = selection.Anchor
	}
	if err := p.buffer.SetSelection(anchor, pos); err != nil {
		a.fail(err)
		return
	}
	a.commitGeometry(p)
	e.hasDesiredX, e.affinity = false, affinity
	e.prepareInputSelection()
}
func (e *Editor) segmentInputAction(a *inputAction, pos, line int) {
	if err := e.compositionIssue(); err != nil {
		a.fail(err)
		return
	}
	if err := e.settleSingleInput(a); err != nil {
		a.fail(err)
		return
	}
	errorsBefore := a.errorCount
	e.selectInputAction(a, pos, false)
	if a.errorCount != errorsBefore || a.abandoned {
		return
	}
	// Drop only this line so the next normal frame shapes the adjacent segment
	// even when the old cached segment includes the shared boundary caret.
	delete(e.cache, line)
	e.geometryGeneration++
	e.revealCaret = true
	e.navigationGeneration++
	e.handle.Focus()
}

func (e *Editor) finishInputRelease(a *inputAction) {
	e.releaseGeometry()
	e.viewGeneration++
	e.terminalViewEpoch = &inputTerminalEpoch{marker: 1}
	e.terminalObservedView = e.viewGeneration
	a.releasedView, a.releasedGeneration = e.terminalViewEpoch, e.viewGeneration
	e.nativeClient = nil
	e.pendingClipboard = nil
	e.foldControl = foldControlPress{}
	if e.folds != nil {
		e.folds.pointer = nil
		e.folds.painted = FoldTarget{}
		e.syncFolds()
	}
	e.focused, e.dragging = false, false
	e.handle.CancelFocus()
	a.reason = InputReleased
	if err := e.pendingMultiIssue(); err != nil {
		a.fail(err)
	}
}

// Kept here so the action's complete bookkeeping uses no synchronous geometry
// callback before publication. The next normal draw performs caret reveal.
func (e *Editor) prepareInputSelection() {
	e.revealCaret = true
	e.blinkStart = e.frameNow
	e.services.Invalidate()
}
