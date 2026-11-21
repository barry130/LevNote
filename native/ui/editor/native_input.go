package editor

import (
	"fmt"
	"slices"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

// inputOwner is an attachment identity. Observe legacy field assignments on
// entry to the editor. Hosts that change either field between observations
// must use the setters to preserve every transition, including A→B→A.
type inputOwner struct {
	buffer        *text.Buffer
	bufferEpoch   uint64
	viewEpoch     uint64
	readOnlyEpoch uint64
	readOnly      bool
}

func (e *Editor) observeInputOwner() inputOwner {
	e.observeTerminalAttachment()
	if e.inputOwner.buffer != e.Buffer {
		e.inputOwner.buffer = e.Buffer
		e.inputOwner.bufferEpoch++
	}
	if e.inputOwner.readOnly != e.ReadOnly {
		e.inputOwner.readOnly = e.ReadOnly
		e.inputOwner.readOnlyEpoch++
	}
	e.inputOwner.viewEpoch = e.viewGeneration
	return e.inputOwner
}

// SetBuffer records buffer identity transitions even before the next frame.
// A prior composition retains its original buffer for explicit cancellation.
func (e *Editor) SetBuffer(buffer *text.Buffer) {
	e.observeInputOwner()
	if e.Buffer != buffer {
		e.Buffer = buffer
		e.observeInputOwner()
		e.services.Invalidate()
	}
}

// SetReadOnly records editability transitions even before the next frame.
// The compatibility field cannot detect direct assignments that cycle back
// to their prior value between observations.
func (e *Editor) SetReadOnly(readOnly bool) {
	e.observeInputOwner()
	if e.ReadOnly != readOnly {
		e.ReadOnly = readOnly
		e.observeInputOwner()
		e.services.Invalidate()
	}
}

// SelectionGeneration covers every normalized selection, including direction,
// primary index and secondary-only changes. It also rejects selection ABA.
type inputStamp struct {
	owner                                 inputOwner
	version, selectionGen, compositionGen uint64
	primary                               text.Range
	token                                 text.CompositionToken
	group                                 bool
}

func (e *Editor) inputStamp() inputStamp {
	owner := e.observeInputOwner()
	s := inputStamp{owner: owner, token: e.compositionToken, compositionGen: e.compositionGeneration}
	if b := owner.buffer; b != nil {
		s.version, s.selectionGen = b.Version(), b.SelectionGeneration()
		s.primary, s.group = b.Selection(), b.InGroup()
	}
	return s
}

type clipboardTarget struct {
	stamp      inputStamp
	selections text.SelectionSet
}

func (e *Editor) clipboardTarget() clipboardTarget {
	s := e.inputStamp()
	return clipboardTarget{stamp: s, selections: s.owner.buffer.SelectionSet()}
}

func (t clipboardTarget) matches(e *Editor) bool {
	if t.stamp != e.inputStamp() {
		return false
	}
	current := t.stamp.owner.buffer.SelectionSet()
	return current.Primary == t.selections.Primary && slices.Equal(current.Ranges, t.selections.Ranges)
}

// Keep one client per attachment, with a stable pointer across frames. MyGo
// invalidates its previous native adapter when the pointer changes. An older
// client can neither use the current editor buffer nor claim a new token.
type nativeInputClient struct {
	editor *Editor
	owner  inputOwner
}

var _ ui.TextInputClient = (*nativeInputClient)(nil)

func (e *Editor) nativeTextClient() *nativeInputClient {
	owner := e.observeInputOwner()
	if e.nativeClient == nil || e.nativeClient.owner != owner {
		e.nativeClient = &nativeInputClient{editor: e, owner: owner}
	}
	return e.nativeClient
}

func (c *nativeInputClient) current() bool {
	return c.owner == c.editor.observeInputOwner() && c.editor.nativeClient == c && c.owner.buffer != nil
}

func (c *nativeInputClient) canRead() bool {
	return c.current() && !c.owner.readOnly
}

func (c *nativeInputClient) canWrite() bool {
	if !c.current() {
		c.editor.fail(text.ErrComposition)
		return false
	}
	return !c.owner.readOnly
}

func (c *nativeInputClient) handleInput(ev ui.InputEvent) bool {
	return !c.current() || c.editor.HandleInput(ev)
}

func (c *nativeInputClient) TextForRange(r ui.TextInputRange) (string, ui.TextInputRange) {
	if c.canRead() {
		return c.editor.TextForRange(r)
	}
	return "", ui.TextInputRange{}
}

func (c *nativeInputClient) Selection() ui.TextInputSelection {
	if c.canRead() {
		return c.editor.Selection()
	}
	return ui.TextInputSelection{}
}

func (c *nativeInputClient) MarkedRange() (ui.TextInputRange, bool) {
	if c.canRead() && c.editor.compositionOwnerIssue(true) == nil {
		return c.editor.MarkedRange()
	}
	return ui.TextInputRange{}, false
}

func (c *nativeInputClient) ReplaceText(r *ui.TextInputRange, value string) {
	if c.canWrite() {
		c.editor.replaceNativeText(r, value)
	}
}

func (c *nativeInputClient) SetMarkedText(r *ui.TextInputRange, value string, selected ui.TextInputRange) {
	if c.canWrite() {
		c.editor.setMarkedText(r, value, selected, true)
	}
}

func (c *nativeInputClient) UnmarkText() {
	if c.canRead() {
		c.editor.UnmarkText()
	}
}

func (c *nativeInputClient) BoundsForRange(r ui.TextInputRange) (ui.Rect, ui.TextInputRange, bool) {
	if c.canRead() {
		return c.editor.BoundsForRange(r)
	}
	return ui.Rect{}, ui.TextInputRange{}, false
}

func (c *nativeInputClient) IndexForPoint(p ui.Point) (int, bool) {
	if c.canRead() {
		return c.editor.IndexForPoint(p)
	}
	return 0, false
}

// Only the toolkit's text-client and InputText routes reach this method.
// Native clients expose one primary marked span regardless of the start set.
func (e *Editor) replaceNativeText(r *ui.TextInputRange, value string) {
	a := e.beginInputAction()
	e.nativeInputAction(a, r, value)
	a.finish()
}
func (e *Editor) nativeInputAction(a *inputAction, r *ui.TextInputRange, value string) {
	if err := e.compositionOwnerIssue(true); err != nil {
		a.fail(err)
		return
	}
	if e.ReadOnly {
		return
	}
	if e.compositionMulti {
		if !e.compositionTargetAdmitted {
			a.fail(e.retainCompositionError(fmt.Errorf("%w：原生预编辑尚未通过主选区范围校验", text.ErrRange)))
			return
		}
		span := ui.TextInputRange{Start: e.compositionPrimary.Start(), End: e.compositionPrimary.End()}
		if e.compositionMarked {
			span = e.marked
		}
		if r != nil && *r != span {
			a.fail(e.retainCompositionError(fmt.Errorf("%w：原生替换范围与主选区或完整标记范围不符", text.ErrRange)))
			return
		}
		if err := e.compositionBuffer.CommitCompositionSetUTF16(e.compositionToken, value); err != nil {
			a.fail(e.retainCompositionError(err))
			return
		}
		e.clearComposition()
		a.retired, a.reason = true, InputFinal
		a.preparedChange()
		return
	}
	if !e.composing && e.Buffer.InGroup() {
		a.fail(text.ErrGroup)
		return
	}
	e.replaceInputAction(a, r, value)
}

func (e *Editor) compositionOwnerIssue(checkRevision bool) error {
	if e.compositionBuffer == nil {
		return nil
	}
	now := e.inputStamp()
	if now.owner != e.compositionStamp.owner || e.composing && !e.compositionBuffer.OwnsComposition(e.compositionToken) {
		return text.ErrComposition
	}
	if checkRevision && now != e.compositionStamp {
		return text.ErrComposition
	}
	return nil
}

func (e *Editor) beginComposition(multi bool) error {
	var token text.CompositionToken
	var err error
	if multi {
		token, err = e.Buffer.BeginCompositionSetGroup()
	} else {
		token, err = e.Buffer.BeginCompositionGroup()
	}
	if err != nil {
		return err
	}
	e.composing, e.compositionMulti = true, multi
	e.compositionBuffer, e.compositionToken = e.Buffer, token
	e.compositionPrimary = e.Buffer.Selection()
	e.compositionTargetAdmitted = false
	e.compositionGeneration++
	e.compositionStamp = e.inputStamp()
	e.compositionChangeStamp = e.Buffer.ChangeStamp()
	if multi {
		// Stop a text drag or arrow press. Toolbar presses retain their
		// captured composition generation so a later release can reject them.
		e.dragging = false
		if e.folds != nil {
			e.folds.pointer = nil
		}
	}
	return nil
}

func (e *Editor) clearComposition() {
	if e.compositionBuffer != nil {
		e.compositionGeneration++
	}
	e.composing, e.compositionMulti, e.compositionMarked = false, false, false
	e.marked = ui.TextInputRange{}
	e.compositionBuffer, e.compositionError = nil, nil
	e.compositionToken = text.CompositionToken{}
	e.compositionStamp = inputStamp{}
	e.compositionPrimary = text.Range{}
	e.compositionTargetAdmitted = false
	e.compositionChangeStamp = text.ChangeStamp{}
}

func (e *Editor) pendingMultiIssue() error {
	if e.compositionMulti {
		e.observeInputOwner()
		return text.ErrCompositionPending
	}
	return nil
}

func (e *Editor) rejectPendingMulti() bool {
	if err := e.pendingMultiIssue(); err != nil {
		e.fail(err)
		return true
	}
	return false
}
