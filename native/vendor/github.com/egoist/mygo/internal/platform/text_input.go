package platform

import (
	"unicode/utf16"
)

// TextRange is a half-open range in UTF-16 code units of a text client's
// document. Native input protocols use these units, independently of UTF-8
// storage or the client's choice of buffer.
type TextRange struct{ Start, End int }

// TextSelection is the primary selection exposed to input methods. A client
// may own additional selections. Reversed puts the caret at Range.Start.
type TextSelection struct {
	Range    TextRange
	Reversed bool
}

func (s TextSelection) Caret() int {
	if s.Reversed {
		return s.Range.Start
	}
	return s.Range.End
}

// TextInputClient supplies a focused custom element's text to native input
// methods. All calls are synchronous on the main thread. Ranges are absolute
// UTF-16 document offsets, including marked text. Nil replacement ranges mean
// the marked range, if present, otherwise the selection.
type TextInputClient interface {
	TextForRange(TextRange) (string, TextRange)
	Selection() TextSelection
	MarkedRange() (TextRange, bool)
	ReplaceText(*TextRange, string)
	SetMarkedText(*TextRange, string, TextRange)
	UnmarkText()
	BoundsForRange(TextRange) (RectF, TextRange, bool)
	IndexForPoint(x, y float64) (int, bool)
}

// CompositionCancelStamp identifies retained application input, including a
// refusal without a mark or a core token. The client must retire its attachment
// before reusing a Generation (including uint64 wrap).
type CompositionCancelStamp struct {
	Generation uint64
	HasTarget  bool
}

type CompositionCancelAvailability uint8

const (
	CancelLegacy CompositionCancelAvailability = iota
	CancelUnavailable
	CancelReady
)

// TextInputCompositionCanceler is optional; required client methods stay intact.
// Queries are observational. The returned after stamp is local to the completed
// cancellation, before its final application callback; transports revalidate it.
type TextInputCompositionCanceler interface {
	CompositionCancelState() (CompositionCancelStamp, CompositionCancelAvailability)
	CancelComposition(CompositionCancelStamp) (CompositionCancelStamp, bool)
}

// Nonzero identities reject attachment and session ABA without keeping text.
type compositionAttachment struct{ marker byte }
type compositionSession struct{ marker byte }

// CompositionSession is a captured transport identity, not an application token.
// Retained/queued events must use the For methods with their original handle.
type CompositionSession struct {
	attachment *compositionAttachment
	session    *compositionSession
}

// InputComposition owns transport ordering only. Legacy End/Replace/Reset retain
// their original behavior on backends which do not opt into SetClient/Start.
// UI-thread-only; no callback can be assumed non-reentrant.
type InputComposition struct {
	pending                  *TextRange
	client                   TextInputClient
	attachment               *compositionAttachment
	session                  *compositionSession
	revision                 uint64
	stamp                    CompositionCancelStamp
	tracked, idle, cancelled bool
}

func cancelState(client TextInputClient) (CompositionCancelStamp, CompositionCancelAvailability) {
	if optional, ok := client.(TextInputCompositionCanceler); ok {
		return optional.CompositionCancelState()
	}
	return CompositionCancelStamp{}, CancelLegacy
}

// Invalidate is the hard attachment boundary, including focus ABA and IME-off.
// Call it before native notifications that may synchronously reenter.
func (c *InputComposition) Invalidate() {
	c.pending, c.client, c.session = nil, nil, nil
	c.attachment = &compositionAttachment{1}
	c.revision = 0
	c.stamp = CompositionCancelStamp{}
	c.tracked, c.idle, c.cancelled = false, false, false
}

// SetClient requires stable pointer clients, as the native text contract does.
func (c *InputComposition) SetClient(client TextInputClient) {
	if c.attachment != nil && c.client == client {
		return
	}
	c.Invalidate()
	c.client = client
}

func (c *InputComposition) Capture() CompositionSession {
	return CompositionSession{c.attachment, c.session}
}

func (c *InputComposition) Current(h CompositionSession, client TextInputClient) bool {
	return h.attachment != nil && h.attachment == c.attachment && h.session == c.session && client == c.client
}

func (c *InputComposition) touch() {
	c.revision++
	if c.revision == 0 {
		// Do not let an outstanding callback regain authority after wrap.
		c.attachment = &compositionAttachment{1}
		c.session, c.pending = nil, nil
		c.tracked, c.idle, c.cancelled = false, false, false
	}
}

func (c *InputComposition) unchanged(h CompositionSession, client TextInputClient, revision uint64) bool {
	return c.Current(h, client) && c.revision == revision
}

// Legacy continuation authority is attachment/session/client identity only.
// A same-scope pending-only Reset keeps the original callback sequence valid.
// Installing any new attachment (including A->B->A or from an unconfigured
// backend) still rejects the old tail. Opt-in paths use unchanged as well.
func (c *InputComposition) legacyUnchanged(h CompositionSession, client TextInputClient) bool {
	if c.Capture() != h {
		return false
	}
	return h.attachment == nil || c.client == client
}

func nextCancelGeneration(before, after CompositionCancelStamp) bool {
	return before.Generation != ^uint64(0) && after.Generation == before.Generation+1
}

// Reset is deliberately pending-only. It cannot lift cancelled-session isolation.
func (c *InputComposition) Reset() { c.touch(); c.pending = nil }

// Start keeps a live owner or an End->final span. Otherwise it begins a new
// session. A pre-existing target is not acquired merely by observing it here.
func (c *InputComposition) Start(client TextInputClient) CompositionSession {
	entry, revision := c.Capture(), c.revision
	stamp, availability := cancelState(client)
	if entry != c.Capture() || c.revision != revision {
		return CompositionSession{}
	}
	if availability != CancelReady {
		return c.Capture() // legacy START never resets its pending span
	}
	if c.client != client || c.attachment == nil {
		c.SetClient(client)
	}
	c.touch()
	h := c.Capture()
	if c.session != nil && !c.cancelled && stamp == c.stamp && (c.tracked && stamp.HasTarget || c.pending != nil) {
		return h
	}
	c.session = &compositionSession{1}
	c.pending = nil
	c.stamp, c.tracked, c.idle, c.cancelled = stamp, false, !stamp.HasTarget, false
	return c.Capture()
}

// Preedit is a flagged composition boundary, including an empty string. It can
// restart a cancelled session without START. Binding is proved from this call's
// before/after stamps, including a first refusal that retains only an error.
func (c *InputComposition) Preedit(client TextInputClient, r *TextRange, value string, selected TextRange) bool {
	return c.PreeditFor(c.Capture(), client, r, value, selected)
}

func (c *InputComposition) PreeditFor(h CompositionSession, client TextInputClient, r *TextRange, value string, selected TextRange) bool {
	if c.attachment != nil && !c.Current(h, client) {
		return false
	}
	c.touch()
	h, revision := c.Capture(), c.revision
	before, availability := cancelState(client)
	if c.attachment != nil && !c.unchanged(h, client, revision) {
		return false
	}
	if availability == CancelLegacy {
		c.pending = nil
		client.SetMarkedText(r, value, selected)
		return c.legacyUnchanged(h, client)
	}
	if availability != CancelReady || !c.Current(h, client) {
		return false
	}
	if before.HasTarget && (!c.tracked || c.cancelled || before != c.stamp) {
		return false // reject an observed/foreign owner before changing session state
	}
	if c.session == nil || c.cancelled || before != c.stamp || !c.tracked && !c.idle {
		c.session = &compositionSession{1}
		c.tracked, c.idle, c.cancelled = false, false, false
		c.stamp = before
		h = c.Capture()
	}
	wasTracked := c.tracked
	if before.HasTarget && !wasTracked {
		return false // START/query cannot acquire someone else's existing owner
	}
	pending := c.pending
	client.SetMarkedText(r, value, selected)
	if !c.unchanged(h, client, revision) {
		return false
	}
	after, availability := cancelState(client)
	if !c.unchanged(h, client, revision) || availability != CancelReady {
		return false
	}
	if before.HasTarget {
		if !wasTracked || after != before {
			return false
		}
	} else if after.HasTarget {
		if !nextCancelGeneration(before, after) {
			return false
		}
	} else if after != before {
		return false
	}
	if c.pending == pending {
		c.pending = nil
	}
	c.stamp, c.tracked = after, after.HasTarget
	return true
}

// CanReplace is checked before a backend consumes its reconversion span. A
// different ready application stamp retires only old cancellation isolation and
// its old span; it never acquires the new target. The changed session handle
// makes the event which observed that transition stale.
func (c *InputComposition) CanReplace(client TextInputClient) bool {
	if c.attachment == nil {
		return true // untouched backends retain their ordinary Replace route
	}
	h, revision := c.Capture(), c.revision
	stamp, availability := cancelState(client)
	if c.attachment != nil && !c.unchanged(h, client, revision) {
		return false
	}
	if availability == CancelLegacy {
		return true
	}
	if availability != CancelReady || !c.Current(h, client) {
		return false
	}
	if c.cancelled {
		if stamp != c.stamp {
			c.touch()
			c.session = &compositionSession{1}
			c.pending = nil
			c.stamp, c.cancelled, c.tracked, c.idle = stamp, false, false, false
		}
		return false
	}
	if c.session == nil {
		return !stamp.HasTarget // ordinary result-only, never acquire an owner
	}
	return stamp == c.stamp && (c.tracked || c.pending != nil || !stamp.HasTarget)
}

// SuppressesResult distinguishes a proved old cancelled result from a stale or
// unavailable client. It is used for the result-first half of a dual-flag event.
func (c *InputComposition) SuppressesResult(client TextInputClient) bool {
	h, revision := c.Capture(), c.revision
	stamp, availability := cancelState(client)
	return c.unchanged(h, client, revision) && availability == CancelReady && c.cancelled && stamp == c.stamp
}

func (c *InputComposition) Cancel(client TextInputClient) bool {
	return c.CancelFor(c.Capture(), client)
}

func (c *InputComposition) CancelFor(h CompositionSession, client TextInputClient) bool {
	if c.attachment != nil && !c.Current(h, client) {
		return false
	}
	c.touch()
	h, revision := c.Capture(), c.revision
	before, availability := cancelState(client)
	if c.attachment != nil && !c.unchanged(h, client, revision) {
		return false
	}
	if availability == CancelLegacy {
		// Preserve the old early clear and sequence only within the same scope.
		c.pending = nil
		_, marked := client.MarkedRange()
		if !c.legacyUnchanged(h, client) {
			return false
		}
		if marked {
			client.SetMarkedText(nil, "", TextRange{})
			if !c.legacyUnchanged(h, client) {
				return false
			}
			client.UnmarkText()
		}
		return c.legacyUnchanged(h, client)
	}
	if availability != CancelReady || !c.Current(h, client) || c.session == nil || c.cancelled || before != c.stamp || !(c.tracked || c.idle && !before.HasTarget) {
		return false
	}
	optional := client.(TextInputCompositionCanceler)
	after, handled := optional.CancelComposition(before)
	if !handled || !c.unchanged(h, client, revision) {
		return false
	}
	current, availability := cancelState(client)
	if !c.unchanged(h, client, revision) || availability != CancelReady || current != after || after.HasTarget {
		return false
	}
	if before.HasTarget && !nextCancelGeneration(before, after) || !before.HasTarget && after != before && !nextCancelGeneration(before, after) {
		return false
	}
	// The application and every callback finished. Only now clear the old span
	// and install suppression. No application callback follows these writes.
	c.pending = nil
	c.stamp, c.cancelled, c.tracked, c.idle = after, true, false, false
	return true
}

func (c *InputComposition) End(client TextInputClient) bool {
	return c.EndFor(c.Capture(), client)
}

func (c *InputComposition) EndFor(h CompositionSession, client TextInputClient) bool {
	if c.attachment != nil && !c.Current(h, client) {
		return false
	}
	c.touch()
	h, revision := c.Capture(), c.revision
	before, availability := CompositionCancelStamp{}, CancelLegacy
	if c.attachment != nil {
		before, availability = cancelState(client)
	}
	if !c.legacyUnchanged(h, client) {
		return false
	}
	if availability == CancelLegacy {
		mark, marked := client.MarkedRange()
		if !c.legacyUnchanged(h, client) {
			return false
		}
		if marked {
			c.pending = &mark
		}
		client.UnmarkText()
		return c.legacyUnchanged(h, client)
	}
	if availability != CancelReady || !c.unchanged(h, client, revision) || c.session == nil || c.cancelled || before != c.stamp {
		return false
	}
	ownedTarget := c.tracked && before.HasTarget
	ownedIdleOrPending := !before.HasTarget && (c.idle || c.pending != nil)
	if !ownedTarget && !ownedIdleOrPending {
		return false // observing a target at START is never settlement authority
	}
	wasTracked := c.tracked
	pending := c.pending
	mark, marked := client.MarkedRange()
	if !c.unchanged(h, client, revision) {
		return false
	}
	fresh, availability := cancelState(client)
	if !c.unchanged(h, client, revision) || availability != CancelReady || fresh != before {
		return false // a getter may retire the old owner without touching transport
	}
	proposed := pending
	if marked {
		proposed = &mark
	}
	client.UnmarkText()
	if !c.unchanged(h, client, revision) {
		return false
	}
	after, availability := cancelState(client)
	if !c.unchanged(h, client, revision) || availability != CancelReady {
		return false
	}
	if after != before && !(before.HasTarget && !after.HasTarget && nextCancelGeneration(before, after)) {
		return false // unproved retirement must not clear a successor/old span
	}
	if c.pending == pending {
		c.pending = proposed
	}
	c.stamp, c.tracked, c.idle = after, wasTracked && after.HasTarget, false
	return true
}

func (c *InputComposition) Replace(client TextInputClient, r *TextRange, value string) bool {
	return c.ReplaceFor(c.Capture(), client, r, value)
}

func (c *InputComposition) ReplaceFor(h CompositionSession, client TextInputClient, r *TextRange, value string) bool {
	if c.attachment != nil && !c.Current(h, client) || !c.CanReplace(client) {
		return false
	}
	if c.attachment != nil && !c.Current(h, client) {
		return false
	}
	c.touch()
	h, revision := c.Capture(), c.revision
	before, availability := CompositionCancelStamp{}, CancelLegacy
	if c.attachment != nil {
		before, availability = cancelState(client)
	}
	if !c.legacyUnchanged(h, client) {
		return false
	}
	if availability != CancelLegacy && (availability != CancelReady || !c.unchanged(h, client, revision) || c.cancelled || c.session != nil && before != c.stamp || c.session == nil && before.HasTarget) {
		return false
	}
	pending := c.pending
	if r == nil && pending != nil {
		r = pending
	}
	if availability == CancelLegacy {
		c.pending = nil // preserve original legacy consumption timing
		client.ReplaceText(r, value)
		return c.legacyUnchanged(h, client)
	}
	client.ReplaceText(r, value)
	if !c.unchanged(h, client, revision) {
		return false
	}
	after, availability := cancelState(client)
	if !c.unchanged(h, client, revision) || availability != CancelReady {
		return false
	}
	if before.HasTarget && after == before {
		if c.pending == pending {
			c.pending = nil
		}
		return true // refused final; the same owner retains its retry route
	}
	if after.HasTarget || before.HasTarget && !nextCancelGeneration(before, after) || !before.HasTarget && after != before {
		return false
	}
	if c.pending == pending {
		c.pending = nil
	}
	c.session = &compositionSession{1} // retire every completed result handle
	c.stamp, c.tracked, c.idle = after, false, false
	return true
}

// UTF16Len counts native text offsets without allocating an encoded string.
func UTF16Len(s string) (n int) {
	for _, r := range s {
		n += utf16.RuneLen(r)
	}
	return
}

// UTF16ByteOffset rounds an offset inside a surrogate pair down to its rune.
func UTF16ByteOffset(s string, offset int) int {
	for i, r := range s {
		if offset < utf16.RuneLen(r) {
			return i
		}
		offset -= utf16.RuneLen(r)
	}
	return len(s)
}

func NormalizeTextRange(r TextRange) TextRange {
	r.Start, r.End = max(0, min(r.Start, r.End)), max(0, max(r.Start, r.End))
	return r
}

// TextContext is a bounded window of a client document, with preedit removed
// as GTK and IMM32 surrounding-text protocols expect. Positions in Text are
// UTF-16 units; Range maps them back to the client's document.
type TextContext struct {
	Text       string
	Start, End int
	Caret      int
	Base       int
	mark       TextRange
	hasMark    bool
}

func ClientTextContext(client TextInputClient) TextContext {
	return clientTextContext(client, 1024, 1024)
}

func clientTextContext(client TextInputClient, before, after int) TextContext {
	sel := client.Selection()
	r := NormalizeTextRange(sel.Range)
	mark, marked := client.MarkedRange()
	if marked {
		mark = NormalizeTextRange(mark)
		r.Start, r.End = min(r.Start, mark.Start), max(r.End, mark.End)
	}
	// A selection may cover a huge document. Query a bounded window around
	// its active end; full text remains available through TextForRange.
	caret := max(0, sel.Caret())
	if marked {
		caret = mark.Start
	}
	wanted := TextRange{Start: max(0, caret-before), End: caret + after}
	if marked {
		wanted.Start, wanted.End = min(wanted.Start, mark.Start), max(wanted.End, mark.End)
	}
	text, actual := client.TextForRange(wanted)
	actual = NormalizeTextRange(actual)
	c := TextContext{Text: text, Base: actual.Start, mark: mark, hasMark: marked && mark.Start >= actual.Start && mark.End <= actual.End}
	if c.hasMark {
		a, z := UTF16ByteOffset(text, mark.Start-actual.Start), UTF16ByteOffset(text, mark.End-actual.Start)
		c.Text = text[:a] + text[z:]
	}
	local := func(i int) int {
		if c.hasMark {
			if i >= mark.End {
				i -= mark.End - mark.Start
			} else if i > mark.Start {
				i = mark.Start
			}
		}
		return max(0, min(i-c.Base, UTF16Len(c.Text)))
	}
	c.Start, c.End = local(r.Start), local(r.End)
	c.Caret = local(sel.Caret())
	if marked {
		c.Start, c.End = local(mark.Start), local(mark.Start)
		c.Caret = c.Start
	}
	return c
}

// Range maps a range of the surrounding text to the actual document. A
// collapsed position at preedit stays collapsed; an explicit range crossing
// it includes preedit, so replacement remains one client operation.
func (c TextContext) Range(r TextRange) TextRange {
	r = NormalizeTextRange(r)
	n := UTF16Len(c.Text)
	r.Start, r.End = min(r.Start, n)+c.Base, min(r.End, n)+c.Base
	if c.hasMark {
		if r.Start > c.mark.Start {
			r.Start += c.mark.End - c.mark.Start
		}
		if r.End > c.mark.Start {
			r.End += c.mark.End - c.mark.Start
		}
	}
	return r
}

// DeleteRange maps GTK character offsets around the selection's end, rather
// than confusing Unicode characters with UTF-16 or UTF-8 offsets.
func (c TextContext) DeleteRange(offset, count int) TextRange {
	caretByte := UTF16ByteOffset(c.Text, c.Caret)
	caretRune := len([]rune(c.Text[:caretByte]))
	runes := []rune(c.Text)
	a := max(0, min(caretRune+offset, len(runes)))
	z := max(a, min(a+max(count, 0), len(runes)))
	return c.Range(TextRange{Start: UTF16Len(string(runes[:a])), End: UTF16Len(string(runes[:z]))})
}

// ClientDeleteRange can query beyond the usual surrounding window when an
// input method asks to replace more text. GTK offsets count Unicode runes.
func ClientDeleteRange(client TextInputClient, offset, count int) TextRange {
	before, after := max(0, -offset)*2, max(0, offset+max(count, 0))*2
	return clientTextContext(client, before, after).DeleteRange(offset, count)
}
