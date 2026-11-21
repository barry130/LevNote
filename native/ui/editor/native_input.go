package editor

import (
	"errors"
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
	return e.inputStampForOwner(e.observeInputOwner())
}

// inputStampForOwner only reads state; optional cancellation queries must not
// observe attachments or acquire an existing programmatic/foreign owner.
func (e *Editor) inputStampForOwner(owner inputOwner) inputStamp {
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
	// Native preedit binds this receipt. An admitted Editor selection change
	// can migrate it to the new selection generation for the same token.
	compositionReceipt inputStamp
}

var _ ui.TextInputClient = (*nativeInputClient)(nil)
var _ ui.TextInputCompositionCanceler = (*nativeInputClient)(nil)

// Unlike current, this check is observational, including on stale attachments.
func (c *nativeInputClient) cancelAttachmentCurrent() bool {
	e := c.editor
	return e.nativeClient == c && c.owner == e.inputOwner && c.owner.buffer != nil &&
		c.owner.buffer == e.Buffer && c.owner.readOnly == e.ReadOnly &&
		c.owner.viewEpoch == e.viewGeneration && !c.owner.readOnly
}

func (c *nativeInputClient) CompositionCancelState() (ui.CompositionCancelStamp, ui.CompositionCancelAvailability) {
	e := c.editor
	stamp := ui.CompositionCancelStamp{Generation: e.compositionGeneration, HasTarget: e.compositionBuffer != nil}
	if !c.cancelAttachmentCurrent() {
		return stamp, ui.CancelUnavailable
	}
	if stamp.HasTarget {
		receipt := c.compositionReceipt
		if receipt != e.compositionStamp || receipt != e.inputStampForOwner(c.owner) || e.compositionBuffer != c.owner.buffer {
			return stamp, ui.CancelUnavailable
		}
		if !e.composing {
			// A native first refusal can retain an owner without opening a Core
			// group. Its exact caller receipt permits the transport to continue
			// the same session, but never gives Cancel a token to roll back.
			if !c.isTokenlessRefusal(receipt) {
				return stamp, ui.CancelUnavailable
			}
			return stamp, ui.CancelReady
		}
		if receipt.token == (text.CompositionToken{}) || !receipt.owner.buffer.OwnsComposition(receipt.token) {
			return stamp, ui.CancelUnavailable
		}
	} else if c.owner.buffer.InGroup() {
		return stamp, ui.CancelUnavailable
	}
	return stamp, ui.CancelReady
}

// Called at native group creation, before range/content refusal or notification.
// An observational getter or a later call cannot claim an uncaptured token.
func (c *nativeInputClient) captureComposition() {
	if c != nil && c.cancelAttachmentCurrent() {
		c.compositionReceipt = c.editor.compositionStamp
	}
}

// Bind the first budget refusal produced by this exact native caller, before
// publishing the error. The receipt admits continuation without a Core token.
func (c *nativeInputClient) captureTokenlessRefusal() {
	if !c.cancelAttachmentCurrent() {
		return
	}
	e := c.editor
	stamp := e.compositionStamp
	if stamp == e.inputStampForOwner(c.owner) && c.isTokenlessRefusal(stamp) {
		c.compositionReceipt = stamp
	}
}

func (c *nativeInputClient) isTokenlessRefusal(stamp inputStamp) bool {
	e := c.editor
	return e.compositionBuffer == c.owner.buffer && !e.composing && !e.compositionMulti &&
		errors.Is(e.compositionError, text.ErrSizeLimit) && stamp.token == (text.CompositionToken{}) && !stamp.group
}

func (c *nativeInputClient) refreshComposition() {
	if c == nil || !c.cancelAttachmentCurrent() {
		return
	}
	e := c.editor
	if c.compositionReceipt.token == e.compositionToken && c.compositionReceipt.compositionGen == e.compositionGeneration {
		c.compositionReceipt = e.compositionStamp
	}
}

// A selection-changing Editor command may transfer a receipt only after its
// old native stamp was proven current and the same owned Core token survives.
func (c *nativeInputClient) refreshCompositionSelection(before, after inputStamp) {
	if c == nil || !c.cancelAttachmentCurrent() {
		return
	}
	e := c.editor
	migrated := before
	migrated.selectionGen = after.selectionGen
	migrated.primary = after.primary
	if c.compositionReceipt != before || before.owner != c.owner || before.token == (text.CompositionToken{}) ||
		after != migrated || after != e.compositionStamp || after != e.inputStampForOwner(c.owner) ||
		e.compositionBuffer != c.owner.buffer || !e.composing || e.compositionMulti ||
		!c.owner.buffer.OwnsComposition(before.token) {
		return
	}
	c.compositionReceipt = after
}

func (c *nativeInputClient) CancelComposition(expected ui.CompositionCancelStamp) (ui.CompositionCancelStamp, bool) {
	before, availability := c.CompositionCancelState()
	if availability != ui.CancelReady || before != expected {
		return before, false
	}
	if !before.HasTarget {
		// The transport, which owns START/session provenance, may cancel idle.
		return before, true
	}
	if c.compositionReceipt.token == (text.CompositionToken{}) {
		// A tokenless refusal is retryable input state, but it is not a
		// composition group and therefore has no rollback authority.
		return before, false
	}
	e := c.editor
	owner, token := c.compositionReceipt.owner.buffer, c.compositionReceipt.token
	version, marked := owner.Version(), e.compositionMarked
	a := e.beginInputAction()
	if err := owner.CancelCompositionGroup(token); err != nil {
		return before, false
	}
	e.clearComposition()
	a.reason, a.retired = InputOwnCancel, true
	if owner.Version() != version {
		a.recovery = InputRecoveryRollback
	}
	if marked || owner.Version() != version {
		a.preparedChange()
	} else {
		// A markless first refusal retires through the terminal-only channel.
		e.services.Invalidate()
	}
	// Capture before the last callback. A successor must invalidate this result
	// at toolkit revalidation; never reload it or clear state after finish.
	after := ui.CompositionCancelStamp{Generation: e.compositionGeneration}
	a.finish()
	return after, true
}

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
		c.editor.setMarkedText(r, value, selected, c)
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

func (e *Editor) beginComposition(multi bool, caller *nativeInputClient) error {
	// A tracked tokenless refusal already advanced the transport generation.
	// Its exact caller may acquire the first Core token in the same session.
	resumingRefusal := false
	if !multi && caller != nil && caller.isTokenlessRefusal(caller.compositionReceipt) {
		stamp, availability := caller.CompositionCancelState()
		resumingRefusal = stamp.HasTarget && availability == ui.CancelReady
	}
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
	if !resumingRefusal {
		e.advanceCompositionGeneration()
	}
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
	caller.captureComposition()
	return nil
}

// Generation values are never reused by an installed attachment. Retiring the
// client before rollover lets the next View install a distinct native adapter.
func (e *Editor) advanceCompositionGeneration() {
	if e.compositionGeneration == ^uint64(0) {
		e.nativeClient = nil
		e.compositionGeneration = 0
		return
	}
	e.compositionGeneration++
}

func (e *Editor) clearComposition() {
	if e.compositionBuffer != nil {
		if c := e.nativeClient; c != nil && c.compositionReceipt.token == e.compositionToken && c.compositionReceipt.compositionGen == e.compositionGeneration {
			c.compositionReceipt = inputStamp{}
		}
		e.advanceCompositionGeneration()
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
