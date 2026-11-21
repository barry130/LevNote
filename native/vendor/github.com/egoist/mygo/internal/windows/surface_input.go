//go:build windows && (amd64 || arm64)

package windows

import (
	"unicode/utf16"
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

// Input methods ask for the text around the caret with WM_IME_REQUEST:
// IMR_DOCUMENTFEED to convert in context, and IMR_RECONVERTSTRING to
// convert again what was typed, whose range IMR_CONFIRMRECONVERTSTRING
// then settles. The composition that follows replaces that range.

const (
	wmImeRequest              = 0x0288
	imrReconvertString        = 0x0004
	imrConfirmReconvertString = 0x0005
	imrDocumentFeed           = 0x0007
)

// reconvertString is RECONVERTSTRING, which the string follows in memory.
// Lengths count UTF-16 units; offsets, bytes from the string's start.
type reconvertString struct {
	Size, Version, StrLen, StrOffset, CompStrLen, CompStrOffset, TargetStrLen, TargetStrOffset uint32
}

// imeRequest answers WM_IME_REQUEST.
func (s *surface) imeRequest(wp, lp uintptr) uintptr {
	if !s.input.Active || s.inputTransition {
		return 0
	}
	epoch, client := s.inputEpoch, s.input.Client
	units := utf16Units(s.input.Text)
	var context platform.TextContext
	start, end := unitOffset(s.input.Text, s.input.Start), unitOffset(s.input.Text, s.input.End)
	if client != nil {
		session := s.clientComposition.Capture()
		stamp, availability := nativeCancelState(client)
		if epoch != s.inputEpoch || s.input.Client != client || s.inputTransition || !s.clientComposition.Current(session, client) || availability == platform.CancelUnavailable {
			return 0
		}
		context = platform.ClientTextContext(client)
		after, afterAvailability := nativeCancelState(client)
		if epoch != s.inputEpoch || s.input.Client != client || s.inputTransition || s.w.closed || !s.clientComposition.Current(session, client) || availability != afterAvailability || availability == platform.CancelReady && stamp != after {
			return 0
		}
		units = utf16Units(context.Text)
		start, end = context.Start, context.End
	}
	switch wp {
	case imrDocumentFeed, imrReconvertString:
		head := uint32(unsafe.Sizeof(reconvertString{}))
		size := head + uint32(len(units))*2
		if lp == 0 {
			return uintptr(size)
		}
		rs := (*reconvertString)(native(lp))
		if rs.Size < size {
			return 0
		}
		*rs = reconvertString{Size: size, StrLen: uint32(len(units)), StrOffset: head,
			CompStrLen: uint32(end - start), CompStrOffset: uint32(start) * 2,
			TargetStrLen: uint32(end - start), TargetStrOffset: uint32(start) * 2}
		copy(unsafe.Slice((*uint16)(unsafe.Add(native(lp), head)), len(units)), units)
		return uintptr(size)
	case imrConfirmReconvertString:
		rs := (*reconvertString)(native(lp))
		from := int(rs.CompStrOffset / 2)
		to := from + int(rs.CompStrLen)
		if from < 0 || to > len(units) {
			return 0
		}
		if s.input.Client != nil {
			r := context.Range(platform.TextRange{Start: from, End: to})
			s.reconvert = &[2]int{r.Start, r.End}
		} else {
			s.reconvert = &[2]int{runeOffset(units, from), runeOffset(units, to)}
		}
		return 1
	}
	return 0
}

// composed sends what an input method composed or committed, in place of
// the text it reconverts, if any.
func (s *surface) composed(ev platform.SurfaceEvent) bool {
	if s.inputTransition {
		return false
	}
	epoch, client, reconvert := s.inputEpoch, s.input.Client, s.reconvert
	if client == nil {
		// Preserve the no-custom-client SurfaceEvent route and consumption time.
		if reconvert != nil {
			ev.Replace, ev.From, ev.To = true, reconvert[0], reconvert[1]
			s.reconvert = nil
		}
		s.send(ev)
		return epoch == s.inputEpoch && !s.inputTransition && !s.w.closed
	}
	session := s.clientComposition.Capture()
	if !s.clientComposition.Current(session, client) {
		return false
	}
	if ev.Kind == platform.TextInput && !s.clientComposition.CanReplace(client) {
		// A quarantined result is consumed without touching reconvert. A new
		// flagged COMP in this same message is still allowed to start a session.
		return s.clientComposition.SuppressesResult(client)
	}
	if epoch != s.inputEpoch || s.input.Client != client || s.inputTransition {
		return false
	}
	var wanted *platform.TextRange
	if reconvert != nil {
		ev.Replace, ev.From, ev.To = true, reconvert[0], reconvert[1]
	}
	if ev.Replace {
		r := platform.TextRange{Start: ev.From, End: ev.To}
		wanted = &r
	}
	_, availability := nativeCancelState(client)
	if epoch != s.inputEpoch || s.input.Client != client || s.inputTransition || !s.clientComposition.Current(session, client) {
		return false
	}
	if availability == platform.CancelLegacy && s.reconvert == reconvert {
		s.reconvert = nil // original legacy pre-callback consumption timing
	}
	var delivered bool
	if ev.Kind == platform.TextInput {
		delivered = s.clientComposition.ReplaceFor(session, client, wanted, ev.Text)
	} else {
		caret := platform.UTF16Len(string([]rune(ev.Text)[:max(0, min(ev.Caret, len([]rune(ev.Text))))]))
		delivered = s.clientComposition.PreeditFor(session, client, wanted, ev.Text, platform.TextRange{Start: caret, End: caret})
	}
	if !delivered || epoch != s.inputEpoch || s.input.Client != client || s.inputTransition || s.w.closed {
		return false
	}
	if s.reconvert == reconvert {
		s.reconvert = nil
	}
	return true
}

func nativeCancelState(client platform.TextInputClient) (platform.CompositionCancelStamp, platform.CompositionCancelAvailability) {
	if optional, ok := client.(platform.TextInputCompositionCanceler); ok {
		return optional.CompositionCancelState()
	}
	return platform.CompositionCancelStamp{}, platform.CancelLegacy
}

func (s *surface) cancelNativeComposition(client platform.TextInputClient) {
	epoch, reconvert := s.inputEpoch, s.reconvert
	session := s.clientComposition.Capture()
	_, availability := nativeCancelState(client)
	if epoch != s.inputEpoch || s.input.Client != client || s.inputTransition || !s.clientComposition.Current(session, client) {
		return
	}
	if availability == platform.CancelLegacy {
		// Legacy Cancel never consumed reconvert; retain the old route.
		s.clientComposition.CancelFor(session, client)
		return
	}
	if availability != platform.CancelReady || !s.clientComposition.CancelFor(session, client) {
		return
	}
	// Controller has already checked the application's local after stamp.
	// Do not clear a reconvert record installed by a cancellation callback.
	if epoch == s.inputEpoch && s.input.Client == client && !s.inputTransition && s.clientComposition.Current(session, client) && s.reconvert == reconvert {
		s.reconvert = nil
	}
}

func (s *surface) endNativeComposition(client platform.TextInputClient) {
	epoch, reconvert := s.inputEpoch, s.reconvert
	session := s.clientComposition.Capture()
	_, availability := nativeCancelState(client)
	if epoch != s.inputEpoch || s.input.Client != client || s.inputTransition || !s.clientComposition.Current(session, client) {
		return
	}
	if availability == platform.CancelLegacy {
		s.reconvert = nil // original legacy End timing
		s.clientComposition.EndFor(session, client)
		return
	}
	if availability != platform.CancelReady || !s.clientComposition.EndFor(session, client) {
		return
	}
	if epoch == s.inputEpoch && s.input.Client == client && !s.inputTransition && s.clientComposition.Current(session, client) && s.reconvert == reconvert {
		s.reconvert = nil
	}
}

// unitOffset returns the UTF-16 offset of rune i of s.
func unitOffset(s string, i int) int {
	n := 0
	for _, r := range s {
		if i == 0 {
			break
		}
		n += utf16.RuneLen(r)
		i--
	}
	return n
}

// runeOffset returns the rune offset of UTF-16 unit i.
func runeOffset(units []uint16, i int) int {
	return len(utf16.Decode(units[:i]))
}
