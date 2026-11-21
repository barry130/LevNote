package editor

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

type terminalTestState struct {
	legacy               []any
	binding              InputBinding
	delivery             InputDelivery
	valid                bool
	actionID, deliveryID InputNotificationID
	compositionStamp     text.ChangeStamp
	frame                time.Time
	stamp                text.ChangeStamp
}

func terminalState(e *Editor) terminalTestState {
	d, ok := e.LastInputDelivery()
	stamp := text.ChangeStamp{}
	if e.Buffer != nil {
		stamp = e.Buffer.ChangeStamp()
	}
	return terminalTestState{captureSourceEditorRejectState(e), e.terminalBinding(), d, ok, e.inputActionID, e.inputDeliveryID, e.compositionChangeStamp, e.frameNow, stamp}
}
func terminalSameDelivery(a, b InputDelivery) bool {
	return a.Binding == b.Binding && a.Editor == b.Editor && a.Buffer == b.Buffer && a.Stamp == b.Stamp &&
		a.Action == b.Action && a.Delivery == b.Delivery && a.Reason == b.Reason && a.Recovery == b.Recovery &&
		a.Disposition == b.Disposition && a.Failed == b.Failed
}
func requireTerminalState(t *testing.T, e *Editor, want terminalTestState) {
	t.Helper()
	got := terminalState(e)
	if !reflect.DeepEqual(got.legacy, want.legacy) || got.binding != want.binding || got.valid != want.valid ||
		!terminalSameDelivery(got.delivery, want.delivery) || got.actionID != want.actionID || got.deliveryID != want.deliveryID ||
		got.compositionStamp != want.compositionStamp || got.frame != want.frame || got.stamp != want.stamp {
		t.Fatal("old action changed callback-exit or observational state")
	}
}
func terminalClient(t *testing.T, e *Editor) (*ui.Tester, *nativeInputClient) {
	t.Helper()
	tester := multiTester(t, e)
	client := e.nativeClient
	if client == nil || client != e.nativeTextClient() || !client.current() || client.editor != e || client.owner.buffer != e.Buffer {
		t.Fatal("not the actual installed current native client")
	}
	return tester, client
}
func terminalPreview(t *testing.T, client *nativeInputClient, start, end int, value string) {
	t.Helper()
	r := ui.TextInputRange{Start: start, End: end}
	client.SetMarkedText(&r, value, ui.TextInputRange{Start: ui.UTF16Len(value), End: ui.UTF16Len(value)})
	if client.editor.compositionError != nil {
		t.Fatal(client.editor.compositionError)
	}
}

func TestConfirmedInputInstalledClientUnmarkAndDirectFlushImmediate(t *testing.T) {
	for _, finish := range []string{"unmark", "flush"} {
		for _, mode := range []string{"changed", "same", "suffix-zero", "irregular-zero"} {
			t.Run(finish+"/"+mode, func(t *testing.T) {
				b := text.New("ab")
				e := New(b)
				_, client := terminalClient(t, e)
				before := b.ChangeStamp()
				changes, terminals := 0, 0
				e.OnChange = func() { changes++ }
				var delivered InputDelivery
				binding := e.BindInputTerminal(func(d InputDelivery) {
					terminals++
					delivered = d
					last, ok := e.LastInputDelivery()
					if !ok || !terminalSameDelivery(last, d) || e.InputActionError() != nil || e.composing || e.compositionBuffer != nil || b.InGroup() || d.Buffer != b || d.Stamp != b.ChangeStamp() {
						t.Fatal("terminal observed intermediate settlement")
					}
				})
				steps := 1
				switch mode {
				case "changed":
					terminalPreview(t, client, 0, 1, "x")
				case "same":
					terminalPreview(t, client, 0, 1, "a")
				case "suffix-zero":
					terminalPreview(t, client, 0, 1, "x")
					terminalPreview(t, client, 0, 1, "a")
					steps = 2
				case "irregular-zero":
					for _, s := range []struct {
						at    int
						value string
					}{{0, "x"}, {1, "y"}, {0, "a"}, {1, "b"}} {
						terminalPreview(t, client, s.at, s.at+1, s.value)
					}
					steps = 4
				}
				if b.ChangeStamp() != before || changes != steps || terminals != 0 || client != e.nativeClient {
					t.Fatal("preedit confirmed early or changed old callback count")
				}
				// No Tester.Frame/Type/Advance occurs between the native calls and
				// these assertions. This uses the installed client directly; it is
				// not the future toolkit no-frame controller/Windows IME fixture.
				if finish == "unmark" {
					client.UnmarkText()
				} else if err := e.Flush(); err != nil {
					t.Fatal(err)
				}
				if terminals != 1 || changes != steps || delivered.Binding != binding || !e.InputBindingCurrent(binding) {
					t.Fatal("terminal multiplicity/origin")
				}
				want := "ab"
				if mode == "changed" {
					want = "xb"
				}
				if b.String() != want || (b.ChangeStamp().ConfirmedChangeID != before.ConfirmedChangeID) != (mode == "changed") || b.ChangeStamp().BaselineID != before.BaselineID {
					t.Fatal("wrong confirmed identity or bytes")
				}
				state := terminalState(e)
				client.UnmarkText()
				if err := e.Flush(); err != nil {
					t.Fatal(err)
				}
				requireTerminalState(t, e, state)
			})
		}
	}
}

func TestConfirmedInputMarklessOwnEscapeAndMultiCompatibility(t *testing.T) {
	b := text.NewWithLimit("a", 1)
	if err := b.ReplaceUTF16(0, 1, "b"); err != nil || !b.Undo() {
		t.Fatal("redo fixture", err)
	}
	if err := b.SetSelection(1, 0); err != nil {
		t.Fatal(err)
	}
	e := New(b)
	_, client := terminalClient(t, e)
	// Click installation may reposition the caret; set the directed fixture
	// before issuing the actual native preedit, with no frame afterward.
	if err := b.SetSelection(1, 0); err != nil {
		t.Fatal(err)
	}
	before, stamp := currentEditState(b), b.ChangeStamp()
	changes, terms, reports := 0, 0, 0
	e.OnChange = func() { changes++ }
	e.OnError = func(error) { reports++ }
	e.BindInputTerminal(func(d InputDelivery) {
		terms++
		if d.Disposition != InputSame || e.InputActionError() != nil {
			t.Fatal("markless cancel not complete")
		}
	})
	r := ui.TextInputRange{Start: 0, End: 1}
	client.SetMarkedText(&r, "🙂", ui.TextInputRange{})
	if e.compositionError == nil || e.composing || b.InGroup() || changes != 0 || reports != 1 {
		t.Fatal("first refusal fixture")
	}
	if !client.handleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape}) || terms != 1 || changes != 0 || reports != 1 || b.ChangeStamp() != stamp || client != e.nativeClient || !client.current() {
		t.Fatal("markless Escape failed")
	}
	requireEditState(t, b, before)
	state := terminalState(e)
	if client.handleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape}) {
		t.Fatal("idle Escape was consumed")
	}
	requireTerminalState(t, e, state)

	b = text.NewWithLimit("a b", 5)
	e = New(b)
	_, client = terminalClient(t, e)
	setSelections(t, b, []text.Range{{Anchor: 1, Head: 1}, {Anchor: 3, Head: 3}}, 1)
	changes, terms = 0, 0
	e.OnChange = func() { changes++ }
	e.OnError = func(error) {}
	e.BindInputTerminal(func(InputDelivery) { terms++ })
	client.SetMarkedText(nil, "拼🙂", ui.TextInputRange{})
	if !e.compositionMulti || e.compositionMarked || !b.InGroup() {
		t.Fatal("markless multi fixture")
	}
	client.handleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape})
	if changes != 1 || terms != 0 || e.nativeClient != nil || client.current() || b.InGroup() || e.InputActionError() != nil {
		t.Fatal("multi Escape callback/client regression")
	}
}

func TestConfirmedInputAcceptedSitesPureGettersAndB3NoOp(t *testing.T) {
	b := text.New("a")
	e := New(b)
	terminalClient(t, e)
	terms, changes, reports := 0, 0, 0
	e.OnChange = func() { changes++ }
	e.OnError = func(error) { reports++ }
	fn := func(InputDelivery) { terms++ }
	old := e.BindInputTerminal(fn)
	newer := e.BindInputTerminal(fn)
	if e.InputBindingCurrent(old) || !e.InputBindingCurrent(newer) {
		t.Fatal("observer registration ABA")
	}
	state := terminalState(e)
	for i := 0; i < 4; i++ {
		e.LastInputDelivery()
		if !e.InputBindingCurrent(newer) || e.InputActionError() != nil {
			t.Fatal("pure observation")
		}
	}
	requireTerminalState(t, e, state)
	r := ui.TextInputRange{Start: 0, End: 1}
	before := b.ChangeStamp()
	e.ReplaceText(&r, "a")
	if changes != 1 || terms != 0 || b.ChangeStamp() != before {
		t.Fatal("accepted same bytes callback count")
	}
	e.ReplaceText(nil, "")
	if changes != 2 || terms != 0 || b.ChangeStamp() != before {
		t.Fatal("accepted empty callback count")
	}
	state = terminalState(e)
	e.ReplaceText(nil, "\xff")
	if reports != 1 || changes != 2 || terms != 0 {
		t.Fatal("pure refusal notified")
	}
	requireTerminalState(t, e, state)
	target := captureLanguageTarget(t, e)
	state = terminalState(e)
	if changed, err := e.ApplySourceEdits(target, []text.PatchEdit{{Start: 0, End: 1, OldText: "a", NewText: "a"}}, SingleInsertionCaret); err != nil || changed {
		t.Fatal(changed, err)
	}
	requireTerminalState(t, e, state)
	if changed, err := e.ApplySourceEdits(target, []text.PatchEdit{{Start: 0, End: 1, OldText: "wrong", NewText: "b"}}, SingleInsertionCaret); changed || !errors.Is(err, text.ErrPatchOldText) {
		t.Fatal(changed, err)
	}
	requireTerminalState(t, e, state)
	// Existing OC site suppresses terminal delivery even with OnChange nil.
	e.OnChange = nil
	terminalPreview(t, e.nativeClient, 0, 1, "x")
	e.nativeClient.ReplaceText(nil, "x")
	if terms != 0 || e.composing {
		t.Fatal("nil OC subscriber caused extra terminal")
	}
	other := text.New("").ChangeStamp()
	other.BaselineID = before.BaselineID
	if inputDisposition(before, other, false) != InputRealCommit {
		t.Fatal("opaque confirmed-epoch change was projected to a counter")
	}
	// Force only Editor notification serial rollover. The Core counter remains
	// opaque here; its actual overflow regression belongs to the Core increment.
	oldAction, oldDelivery, stamp := e.inputActionID, e.inputDeliveryID, b.ChangeStamp()
	e.inputActionID.serial, e.inputDeliveryID.serial = ^uint64(0), ^uint64(0)
	e.ReplaceText(&r, "x")
	if e.inputActionID.epoch == oldAction.epoch || e.inputDeliveryID.epoch == oldDelivery.epoch || b.ChangeStamp() != stamp || terms != 0 {
		t.Fatal("notification rollover changed text or aliased old IDs")
	}
}

func TestConfirmedInputCompleteCombinedPublicActions(t *testing.T) {
	for _, action := range []string{"copy", "cut", "undo", "undo-zero", "line-identical", "indent", "outdent", "delete", "moveLinesDown", "moveLinesUp", "position", "left", "enter"} {
		t.Run(action, func(t *testing.T) {
			seed := "a\nb"
			if action == "line-identical" {
				seed = "a\na"
			}
			b := text.New(seed)
			e := New(b)
			_, client := terminalClient(t, e)
			changes, terms := 0, 0
			checking := false
			e.OnChange = func() {
				changes++
				if checking && (e.compositionBuffer != nil || b.InGroup() || e.pendingClipboard != nil) {
					t.Fatal("OC observed an intermediate combined action")
				}
			}
			e.BindInputTerminal(func(InputDelivery) {
				terms++
				if e.compositionBuffer != nil || b.InGroup() || e.pendingClipboard != nil {
					t.Fatal("terminal observed intermediate command")
				}
			})
			preview := "X"
			if action == "undo-zero" || action == "line-identical" {
				preview = "a"
			}
			terminalPreview(t, client, 0, 1, preview)
			changes, checking = 0, true
			before := b.ChangeStamp()
			if action == "copy" || action == "cut" {
				if err := b.SetSelection(0, 1); err != nil {
					t.Fatal(err)
				}
				e.Copy = func(s string) error {
					if s != "X" {
						t.Fatal(s)
					}
					return nil
				}
			}
			switch action {
			case "undo-zero":
				e.Command("undo")
			case "line-identical":
				e.Command("moveLinesDown")
			case "position":
				if err := e.SetPosition(1, 0); err != nil {
					t.Fatal(err)
				}
			case "left":
				client.handleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyLeft})
			case "enter":
				client.handleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnter})
			default:
				if !e.Command(action) {
					t.Fatal("unhandled")
				}
			}
			if action == "moveLinesUp" {
				if changes != 0 || terms != 0 || !e.composing || b.ChangeStamp() != before {
					t.Fatal("boundary action settled input")
				}
				return
			}
			wantChanges, wantTerms := 1, 0
			if action == "copy" || action == "outdent" || action == "position" || action == "left" || action == "undo-zero" || action == "line-identical" {
				wantChanges, wantTerms = 0, 1
			}
			if changes != wantChanges || terms != wantTerms || e.InputActionError() != nil || b.ChangeStamp().BaselineID != before.BaselineID {
				t.Fatal("combined callback multiplicity", changes, terms)
			}
			if action == "undo" && (b.String() != "a\nb" || b.History().RedoDepth != 1) {
				t.Fatal("settlement then Undo was dropped")
			}
			if action == "indent" || action == "cut" || action == "delete" || action == "moveLinesDown" || action == "enter" {
				if b.History().UndoDepth != 2 {
					t.Fatal("combined commits not separate history", b.History())
				}
			}
		})
	}
}

func TestConfirmedInputClipboardInlineLaterDuplicateAndObserverReplacement(t *testing.T) {
	for _, mode := range []string{"inline", "later", "rebind"} {
		b := text.New("abc")
		e := New(b)
		terminalClient(t, e)
		if err := b.SetSelection(3, 0); err != nil {
			t.Fatal(err)
		}
		changes, oldTerms, newTerms := 0, 0, 0
		e.OnChange = func() { changes++ }
		e.BindInputTerminal(func(InputDelivery) { oldTerms++ })
		var done func(error)
		e.CopyThen = func(value string, complete func(error)) {
			if value != "abc" {
				t.Fatal(value)
			}
			done = complete
			if mode == "rebind" {
				e.BindInputTerminal(func(InputDelivery) { newTerms++ })
			}
			if mode != "later" {
				complete(nil)
			}
		}
		if !e.Command("cut") || done == nil {
			t.Fatal("cut request")
		}
		if mode == "later" {
			if b.String() != "abc" || e.pendingClipboard == nil {
				t.Fatal("cut did not wait")
			}
			done(nil)
		}
		if b.String() != "" || changes != 1 || oldTerms != 0 || newTerms != 0 || e.pendingClipboard != nil {
			t.Fatal("clipboard callback/observer contract", mode)
		}
		state := terminalState(e)
		done(nil)
		requireTerminalState(t, e, state)
	}
}

func TestConfirmedInputSettlementErrorAndSuccessorClipboardExit(t *testing.T) {
	b := text.New("a")
	e := New(b)
	_, client := terminalClient(t, e)
	terms := 0
	e.BindInputTerminal(func(InputDelivery) { terms++ })
	terminalPreview(t, client, 0, 1, "x")
	if err := b.SetSelection(1, 0); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("clipboard sentinel")
	e.Copy = func(string) error { return failure }
	var exit terminalTestState
	var oldBuffer sourceRejectState
	errorCalls := 0
	e.OnError = func(err error) {
		errorCalls++
		d, ok := e.LastInputDelivery()
		if !errors.Is(err, failure) || !ok || !d.Failed || d.Buffer != b || d.Stamp != b.ChangeStamp() || d.Disposition != InputRealCommit || e.pendingClipboard != nil || e.InputActionError() != nil {
			t.Fatal("settlement error record missing or late")
		}
		e.Copy = nil
		e.CopyThen = func(string, func(error)) {}
		if !e.Command("copy") || e.pendingClipboard == nil {
			t.Fatal("successor clipboard")
		}
		exit, oldBuffer = terminalState(e), captureSourceRejectState(b)
	}
	if !e.Command("copy") || errorCalls != 1 || terms != 0 {
		t.Fatal("error notification count")
	}
	requireTerminalState(t, e, exit)
	requireSourceRejectState(t, b, oldBuffer)
}

func TestConfirmedInputNestedB3AndTerminalCallbacksPreserveExit(t *testing.T) {
	for _, entry := range []string{"b3", "ordinary", "terminal"} {
		t.Run(entry, func(t *testing.T) {
			b := text.New("a")
			e := New(b)
			_, client := terminalClient(t, e)
			var exit terminalTestState
			var source sourceRejectState
			calls, terms := 0, 0
			entered := false
			e.OnChange = func() {
				calls++
				if (entry == "b3" || entry == "ordinary") && !entered {
					entered = true
					if entry == "b3" {
						e.ReplaceText(nil, "!")
					} else {
						target := captureLanguageTarget(t, e)
						if changed, err := e.ApplySourceEdits(target, []text.PatchEdit{{Start: 0, End: 1, OldText: "b", NewText: "c"}}, SingleInsertionCaret); err != nil || !changed {
							t.Fatal(changed, err)
						}
					}
					exit, source = terminalState(e), captureSourceRejectState(b)
				}
			}
			e.BindInputTerminal(func(InputDelivery) {
				terms++
				if entry != "terminal" {
					t.Fatal("B3 invoked terminal observer")
				}
				target := captureLanguageTarget(t, e)
				if changed, err := e.ApplySourceEdits(target, []text.PatchEdit{{Start: 0, End: 1, OldText: "x", NewText: "y"}}, SingleInsertionCaret); err != nil || !changed {
					t.Fatal(changed, err)
				}
				exit, source = terminalState(e), captureSourceRejectState(b)
			})
			if entry == "b3" {
				target := captureLanguageTarget(t, e)
				if changed, err := e.ApplySourceEdits(target, []text.PatchEdit{{Start: 0, End: 1, OldText: "a", NewText: "b"}}, SingleInsertionCaret); err != nil || !changed {
					t.Fatal(changed, err)
				}
				if b.String() != "b!" || calls != 2 || terms != 0 {
					t.Fatal("nested B3 trace", b.String(), calls, terms)
				}
			} else if entry == "ordinary" {
				e.ReplaceText(&ui.TextInputRange{Start: 0, End: 1}, "b")
				if b.String() != "c" || calls != 2 || terms != 0 {
					t.Fatal("ordinary/B3 trace", b.String(), calls, terms)
				}
			} else {
				terminalPreview(t, client, 0, 1, "x")
				if err := e.Flush(); err != nil {
					t.Fatal(err)
				}
				if b.String() != "y" || calls != 2 || terms != 1 {
					t.Fatal("terminal/B3 trace")
				}
			}
			requireTerminalState(t, e, exit)
			requireSourceRejectState(t, b, source)
		})
	}
}

func TestConfirmedInputTerminalReentryAndBaselineInvalidation(t *testing.T) {
	for _, mutation := range []string{"buffer", "readonly-ABA", "release", "observer-ABA", "preedit"} {
		t.Run(mutation, func(t *testing.T) {
			b := text.New("a")
			e := New(b)
			_, client := terminalClient(t, e)
			var exit terminalTestState
			var original sourceRejectState
			e.BindInputTerminal(func(InputDelivery) {
				switch mutation {
				case "buffer":
					e.SetBuffer(text.New("replacement"))
				case "readonly-ABA":
					e.SetReadOnly(true)
					e.SetReadOnly(false)
				case "release":
					e.ReleaseView()
				case "observer-ABA":
					fn := func(InputDelivery) {}
					e.BindInputTerminal(fn)
					e.BindInputTerminal(nil)
					e.BindInputTerminal(fn)
				case "preedit":
					e.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "y", ui.TextInputRange{Start: 1, End: 1})
				}
				e.focused = false
				e.blinkStart = time.Unix(17, 0)
				exit, original = terminalState(e), captureSourceRejectState(b)
			})
			terminalPreview(t, client, 0, 1, "x")
			if err := e.Flush(); err != nil {
				t.Fatal(err)
			}
			requireTerminalState(t, e, exit)
			requireSourceRejectState(t, b, original)
		})
	}
	for _, revoke := range []bool{false, true} {
		b := text.New("a")
		e := New(b)
		_, client := terminalClient(t, e)
		terms := 0
		var binding InputBinding
		binding = e.BindInputTerminal(func(d InputDelivery) {
			terms++
			if d.Reason != InputReleased || d.Binding != binding || e.InputBindingCurrent(binding) || e.nativeClient != nil || e.pendingClipboard != nil || e.focused || e.dragging || e.compositionBuffer != nil || b.InGroup() {
				t.Fatal("release notified before complete retirement")
			}
		})
		terminalPreview(t, client, 0, 1, "x")
		if revoke {
			e.BindInputTerminal(nil)
		}
		e.ReleaseView()
		want := 1
		if revoke {
			want = 0
		}
		if terms != want || b.String() != "x" || b.InGroup() {
			t.Fatal("standalone/revoked release", terms, want)
		}
	}
	b := text.New("ab")
	e := New(b)
	_, client := terminalClient(t, e)
	terms, reports := 0, 0
	e.BindInputTerminal(func(InputDelivery) { terms++ })
	terminalPreview(t, client, 0, 1, "x")
	before := b.ChangeStamp()
	b.ResetHistory() // retains xb, invalidates the settled baseline
	if b.String() != "xb" || b.ChangeStamp().BaselineID == before.BaselineID {
		t.Fatal("baseline fixture")
	}
	e.OnError = func(err error) {
		reports++
		d, ok := e.LastInputDelivery()
		if !errors.Is(err, text.ErrComposition) || !ok || d.Stamp != b.ChangeStamp() || d.Disposition != InputUnprovedRetirement {
			t.Fatal("unproved retirement was mislabeled readiness")
		}
	}
	client.handleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape})
	if terms != 0 || reports != 1 || e.InputActionError() != nil || b.String() != "xb" {
		t.Fatal("reset cancellation semantics")
	}
}

func TestConfirmedInputActualTesterSegmentsNotifyAfterWholeView(t *testing.T) {
	for _, label := range []string{"上一段", "下一段"} {
		t.Run(label, func(t *testing.T) {
			b := text.New(strings.Repeat("a", 10000))
			e := New(b)
			var exit terminalTestState
			checkExit := false
			tester := ui.NewTester(func(c *ui.Context) {
				e.View(c)
				if checkExit {
					requireTerminalState(t, e, exit)
					checkExit = false
				}
			}, 600, 280)
			if err := tester.Click(e.Label); err != nil {
				t.Fatal(err)
			}
			at := 0
			if label == "上一段" {
				at = 3500
			}
			if err := b.SetSelection(at, at); err != nil {
				t.Fatal(err)
			}
			client := e.nativeClient
			terminalPreview(t, client, at, at, "x")
			terms := 0
			e.BindInputTerminal(func(d InputDelivery) {
				terms++
				if d.Buffer != b || b.InGroup() || e.compositionBuffer != nil || e.nativeClient == nil || e.InputActionError() != nil {
					t.Fatal("segment callback not at complete View boundary")
				}
				e.focused = false
				e.blinkStart = time.Unix(23, 0)
				exit, checkExit = terminalState(e), true
			})
			// A normal frame installs the actual current segment controls. This
			// test makes no zero-frame toolkit-controller or physical IME claim.
			tester.Frame()
			if err := tester.Click(label); err != nil {
				t.Fatal(err)
			}
			if terms != 1 || checkExit {
				t.Fatal("segment event/boundary not exercised", terms, checkExit)
			}
		})
	}
}

func TestConfirmedInputLargeNavigationHasNoLanguageSnapshotBudget(t *testing.T) {
	b := text.New(strings.Repeat("a", (2<<20)+4096))
	e := New(b)
	_, client := terminalClient(t, e)
	changes, terms, shapes := 0, 0, 0
	var reported error
	e.OnChange = func() { changes++ }
	e.OnError = func(err error) { reported = err }
	e.BindInputTerminal(func(InputDelivery) { terms++ })
	e.Highlight = func(s string) []ui.Span {
		shapes++
		if len(s) > maxShapedLine {
			t.Fatal("navigation shaped a whole large source")
		}
		return nil
	}
	stamp := b.ChangeStamp()
	if err := e.SetPosition(0, b.LenUTF16()-1); err != nil {
		t.Fatal(err)
	}
	if !client.handleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnd}) || reported != nil || changes != 0 || terms != 0 || shapes == 0 || b.ChangeStamp() != stamp || b.LenBytes() != (2<<20)+4096 {
		t.Fatal("ordinary large navigation gained a language limit or text event", reported, shapes)
	}
}

func TestConfirmedInputObserverCallbackSequenceStopsOnSuccessor(t *testing.T) {
	b := text.New("a")
	e := New(b)
	terminalClient(t, e)
	if err := b.SetSelection(0, 0); err != nil {
		t.Fatal(err)
	}
	trace := []string{}
	e.BindInputTerminal(func(InputDelivery) { t.Fatal("error action sent terminal") })
	e.OnError = func(error) {
		trace = append(trace, "error")
		e.ReplaceText(nil, "x")
	}
	// The fixed cleanup/refusal result is exercised directly in addition to
	// installed-client error cases: the first error starts a successor action.
	a := e.beginInputAction()
	a.fail(errors.New("cleanup"))
	a.fail(errors.New("refusal"))
	a.finish()
	if len(trace) != 1 || b.String() != "xa" {
		t.Fatal("stale second error survived reentry", trace, b.String())
	}
}

func TestConfirmedInputPasteErrorOriginBeforeFirstReport(t *testing.T) {
	cases := []struct {
		name, mutation                             string
		bound, replaceHandler, returnError, report bool
	}{
		{"bound-readonly-aba-replacement", "readonly", true, true, true, false},
		{"bound-buffer-aba-replacement", "buffer", true, true, true, false},
		{"bound-readonly-aba-captured-handler", "readonly", true, false, true, false},
		{"bound-buffer-aba-valid-value", "buffer", true, false, false, false},
		{"bound-current-captured-handler", "same", true, true, true, true},
		{"legacy-readonly-aba-captured-handler", "readonly", false, true, true, true},
		{"legacy-buffer-aba-stale-target", "buffer", false, true, false, true},
		{"legacy-registration-revoked", "register", false, true, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const source = "a🙂b\r\n"
			b, other := text.New(source), text.New(source)
			e := New(b)
			_, client := terminalClient(t, e)
			if err := b.SetSelection(3, 1); err != nil {
				t.Fatal(err)
			}
			changes, terminals, calls := 0, 0, 0
			var originalReports, replacementReports []error
			e.OnChange = func() { changes++ }
			e.OnError = func(err error) { originalReports = append(originalReports, err) }
			var binding InputBinding
			if tc.bound {
				binding = e.BindInputTerminal(func(InputDelivery) { terminals++ })
			}
			failure := errors.New("clipboard read refused")
			original, alternate := captureSourceRejectState(b), captureSourceRejectState(other)
			var exit terminalTestState
			e.Paste = func() (string, error) {
				calls++
				switch tc.mutation {
				case "readonly":
					e.SetReadOnly(true)
					e.SetReadOnly(false)
				case "buffer":
					e.SetBuffer(other)
					e.SetBuffer(b)
				case "register":
					e.BindInputTerminal(nil)
				}
				if tc.replaceHandler {
					e.OnError = func(err error) { replacementReports = append(replacementReports, err) }
				}
				exit = terminalState(e)
				if tc.returnError {
					return "", failure
				}
				return "不应黏贴", nil
			}
			if !e.Command("paste") || calls != 1 || changes != 0 || terminals != 0 || len(replacementReports) != 0 || e.nativeClient != client {
				t.Fatalf("clipboard escaped its origin: calls=%d changes=%d terminals=%d new errors=%v", calls, changes, terminals, replacementReports)
			}
			if tc.bound && e.InputBindingCurrent(binding) != (tc.mutation == "same") {
				t.Fatal("fixture failed to observe the setter round trip")
			}
			if tc.report {
				if len(originalReports) != 1 || originalReports[0] == nil || (originalReports[0] == failure) != tc.returnError {
					t.Fatalf("captured handler got the wrong result: %v", originalReports)
				}
			} else if len(originalReports) != 0 {
				t.Fatalf("retired bound owner received an error: %v", originalReports)
			}
			requireTerminalState(t, e, exit)
			requireSourceRejectState(t, b, original)
			requireSourceRejectState(t, other, alternate)
		})
	}
}

func TestConfirmedInputObserverRebindKeepsPasteAndCapturedChange(t *testing.T) {
	b := text.New("a🙂b\r\n")
	e := New(b)
	_, client := terminalClient(t, e)
	if err := b.SetSelection(3, 1); err != nil {
		t.Fatal(err)
	}
	changes, replacementChanges, terminals, reports := 0, 0, 0, 0
	before, serials := b.ChangeStamp(), [2]InputNotificationID{e.inputActionID, e.inputDeliveryID}
	e.OnError = func(error) { reports++ }
	observer := func(InputDelivery) { terminals++ }
	first := e.BindInputTerminal(observer)
	var next InputBinding
	var exit terminalTestState
	var exitSource sourceRejectState
	e.OnChange = func() {
		changes++
		if _, valid := e.LastInputDelivery(); valid {
			t.Fatal("paste published an old receipt after observer replacement")
		}
		exit, exitSource = terminalState(e), captureSourceRejectState(b)
	}
	e.Paste = func() (string, error) {
		next = e.BindInputTerminal(observer)
		e.OnChange = func() { replacementChanges++ }
		return "中🙂", nil
	}
	if !e.Command("paste") || b.String() != "a中🙂b\r\n" || b.Selection() != (text.Range{Anchor: 4, Head: 4}) ||
		b.History().UndoDepth != 1 || b.ChangeStamp().BaselineID != before.BaselineID || b.ChangeStamp().ConfirmedChangeID == before.ConfirmedChangeID ||
		changes != 1 || replacementChanges != 0 || reports != 0 || terminals != 0 || e.nativeClient != client ||
		e.InputBindingCurrent(first) || !e.InputBindingCurrent(next) || [2]InputNotificationID{e.inputActionID, e.inputDeliveryID} != serials {
		t.Fatalf("observer rebind changed paste or notification semantics: %q, changes=%d replacement=%d reports=%d terminals=%d", b.String(), changes, replacementChanges, reports, terminals)
	}
	requireTerminalState(t, e, exit)
	requireSourceRejectState(t, b, exitSource)
}

func TestConfirmedInputDelayedCutErrorUsesRequestAuthority(t *testing.T) {
	for _, mode := range []string{"current", "bound-aba", "legacy-aba", "observer-replaced"} {
		t.Run(mode, func(t *testing.T) {
			b := text.New("abc")
			e := New(b)
			terminalClient(t, e)
			if err := b.SetSelection(3, 0); err != nil {
				t.Fatal(err)
			}
			original, replacement, changes := 0, 0, 0
			failure := errors.New("clipboard write refused")
			var exit terminalTestState
			e.OnError = func(err error) {
				if err != failure || e.pendingClipboard != nil {
					t.Fatalf("old request completed with %v and pending=%p", err, e.pendingClipboard)
				}
				original++
				exit = terminalState(e)
			}
			e.OnChange = func() { changes++ }
			if mode != "legacy-aba" {
				e.BindInputTerminal(func(InputDelivery) { t.Fatal("idle cut emitted terminal delivery") })
			}
			var done func(error)
			e.CopyThen = func(value string, complete func(error)) {
				if value != "abc" {
					t.Fatalf("copied %q", value)
				}
				done = complete
			}
			before := captureSourceRejectState(b)
			if !e.Command("cut") || done == nil || e.pendingClipboard == nil {
				t.Fatal("cut did not queue its request")
			}
			if mode == "bound-aba" || mode == "legacy-aba" {
				e.SetReadOnly(true)
				e.SetReadOnly(false)
			} else if mode == "observer-replaced" {
				e.BindInputTerminal(nil)
			}
			e.OnError = func(error) { replacement++ }
			done(failure)
			want := 0
			if mode == "current" || mode == "legacy-aba" {
				want = 1
			}
			if original != want || replacement != 0 || changes != 0 || e.pendingClipboard != nil {
				t.Fatalf("delayed request retargeted or cut after failure: old=%d new=%d changes=%d pending=%p", original, replacement, changes, e.pendingClipboard)
			}
			if want == 1 {
				requireTerminalState(t, e, exit)
			}
			settled := terminalState(e)
			done(nil)
			requireTerminalState(t, e, settled)
			requireSourceRejectState(t, b, before)
		})
	}
}

func TestConfirmedInputCopyOnlyInlineDuplicateKeepsSettlement(t *testing.T) {
	for _, mode := range []string{"nil-duplicate", "error-duplicate"} {
		t.Run(mode, func(t *testing.T) {
			b := text.New("ab\r\n")
			e := New(b)
			_, client := terminalClient(t, e)
			changes, terminals, reports, copies := 0, 0, 0, 0
			before := b.ChangeStamp()
			var preedit, delivered InputDelivery
			var exit terminalTestState
			var exitSource sourceRejectState
			e.OnChange = func() { changes++ }
			e.OnError = func(error) { reports++ }
			binding := e.BindInputTerminal(func(d InputDelivery) {
				terminals++
				delivered = d
				last, valid := e.LastInputDelivery()
				if !valid || !terminalSameDelivery(last, d) || d.Action == preedit.Action || d.Delivery == preedit.Delivery ||
					d.Action != e.inputActionID || d.Delivery != e.inputDeliveryID || d.Reason != InputSettled || d.Disposition != InputRealCommit || d.Failed ||
					d.Buffer != b || d.Stamp != b.ChangeStamp() || d.Stamp.BaselineID != before.BaselineID || d.Stamp.ConfirmedChangeID == before.ConfirmedChangeID ||
					e.InputActionError() != nil || e.pendingClipboard != nil || e.compositionBuffer != nil || b.InGroup() || e.nativeClient != client {
					t.Fatalf("copy settlement was missing or published before its action ended: %+v", d)
				}
				exit, exitSource = terminalState(e), captureSourceRejectState(b)
			})
			client.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "x🙂", ui.TextInputRange{Start: 1, End: 3})
			var valid bool
			preedit, valid = e.LastInputDelivery()
			selection := text.Range{Anchor: 1, Head: 3}
			if mark, active := client.MarkedRange(); !active || mark != (ui.TextInputRange{Start: 0, End: 3}) || b.Selection() != selection || !valid || preedit.Reason != InputPreedit || changes != 1 || b.ChangeStamp() != before {
				t.Fatalf("fixture needs a nonempty subselection of installed preedit: %v %v %+v", mark, active, b.Selection())
			}
			duplicateError := errors.New("duplicate completion must be ignored")
			var duplicate error
			if mode == "error-duplicate" {
				duplicate = duplicateError
			}
			var late func(error)
			var afterSettle sourceRejectState
			e.CopyThen = func(value string, done func(error)) {
				copies++
				if value != "🙂" || e.pendingClipboard == nil || e.pendingClipboard.cut || b.InGroup() || e.compositionBuffer != nil || b.String() != "x🙂b\r\n" || b.Selection() != selection {
					t.Fatalf("copy hook saw wrong or incomplete operation: %q", value)
				}
				afterSettle = captureSourceRejectState(b)
				late = done
				done(nil)
				if e.pendingClipboard != nil || terminals != 0 || reports != 0 {
					t.Fatal("successful copy completed before its dispatch returned")
				}
				firstExit := terminalState(e)
				done(duplicate)
				done(nil)
				requireTerminalState(t, e, firstExit)
				requireSourceRejectState(t, b, afterSettle)
			}
			if !e.Command("copy") || copies != 1 || changes != 1 || terminals != 1 || reports != 0 || delivered.Binding != binding {
				t.Fatalf("duplicate discarded or repeated copy settlement: copies=%d changes=%d terminals=%d reports=%d", copies, changes, terminals, reports)
			}
			if b.History().UndoDepth != 1 || b.History().RedoDepth != 0 || !e.InputBindingCurrent(binding) {
				t.Fatalf("copy changed the completed edit's history or owner: %+v", b.History())
			}
			requireTerminalState(t, e, exit)
			requireSourceRejectState(t, b, exitSource)
			requireSourceRejectState(t, b, afterSettle)

			var finishNext func(error)
			e.CopyThen = func(_ string, done func(error)) { finishNext = done }
			if !e.Command("copy") || finishNext == nil || e.pendingClipboard == nil {
				t.Fatal("successor copy did not queue")
			}
			next, nextState := e.pendingClipboard, terminalState(e)
			late(nil)
			late(duplicateError)
			requireTerminalState(t, e, nextState)
			if e.pendingClipboard != next || next.value != "🙂" || !next.dispatched || changes != 1 || terminals != 1 || reports != 0 {
				t.Fatal("old completion touched the successor request")
			}
			finishNext(nil)
			if e.pendingClipboard != nil || changes != 1 || terminals != 1 || reports != 0 {
				t.Fatal("new copy failed to complete quietly")
			}
			requireTerminalState(t, e, exit)
			requireSourceRejectState(t, b, exitSource)
		})
	}
}

func TestConfirmedInputInlineCutErrorCompletesOnceWithSuccessor(t *testing.T) {
	b := text.New("ab")
	e := New(b)
	_, client := terminalClient(t, e)
	changes, reports := 0, 0
	e.OnChange = func() { changes++ }
	binding := e.BindInputTerminal(func(InputDelivery) { t.Fatal("failed cut sent terminal") })
	before := b.ChangeStamp()
	client.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "xy", ui.TextInputRange{Start: 0, End: 1})
	selection := text.Range{Anchor: 0, Head: 1}
	if !e.CompositionActive() || b.Selection() != selection || changes != 1 {
		t.Fatal("fixture has no selected installed preedit")
	}
	failure := errors.New("first cut completion failed")
	var next *clipboardRequest
	var nextDone func(error)
	var exit terminalTestState
	var exitSource sourceRejectState
	e.OnError = func(err error) {
		reports++
		d, valid := e.LastInputDelivery()
		if err != failure || !valid || d.Binding != binding || !d.Failed || d.Reason != InputSettled || d.Disposition != InputRealCommit ||
			d.Stamp != b.ChangeStamp() || d.Stamp.BaselineID != before.BaselineID || d.Stamp.ConfirmedChangeID == before.ConfirmedChangeID ||
			e.pendingClipboard != nil || e.compositionBuffer != nil || b.InGroup() || b.String() != "xyb" || b.Selection() != selection {
			t.Fatalf("cut error lost its completed settlement or deleted text: %v %+v", err, d)
		}
		e.CopyThen = func(_ string, done func(error)) { nextDone = done }
		if !e.Command("copy") || nextDone == nil || e.pendingClipboard == nil {
			t.Fatal("error callback could not open a successor copy")
		}
		next = e.pendingClipboard
		exit, exitSource = terminalState(e), captureSourceRejectState(b)
	}
	e.CopyThen = func(value string, done func(error)) {
		if value != "x" {
			t.Fatalf("cut captured %q", value)
		}
		done(failure)
		done(nil)
		done(failure)
	}
	if !e.Command("cut") || changes != 1 || reports != 1 || next == nil || e.pendingClipboard != next || b.History().UndoDepth != 1 {
		t.Fatalf("duplicate cut altered the successor or emitted another event: changes=%d reports=%d history=%+v", changes, reports, b.History())
	}
	requireTerminalState(t, e, exit)
	requireSourceRejectState(t, b, exitSource)
	nextDone(nil)
	if e.pendingClipboard != nil || changes != 1 || reports != 1 {
		t.Fatal("successor copy was not independently completable")
	}
}

func TestConfirmedInputHighlightPublishedSuccessorStopsSettledNavigation(t *testing.T) {
	for _, mode := range []string{"empty", "equal", "nil-on-change", "warning-before-empty", "benign", "benign-warning", "b3-noop"} {
		t.Run(mode, func(t *testing.T) {
			warns := mode == "warning-before-empty" || mode == "benign-warning"
			publishes := mode == "empty" || mode == "equal" || mode == "nil-on-change" || mode == "warning-before-empty"
			original, changed := "ab", "xb"
			if warns {
				original += strings.Repeat("z", 190)
				changed += strings.Repeat("z", 190)
			}
			b := text.New(original)
			e := New(b)
			_, client := terminalClient(t, e)
			before := b.ChangeStamp()
			var exit terminalTestState
			var exitSource sourceRejectState
			var delivery InputDelivery
			var seen bool
			remember := func(d InputDelivery) {
				seen = true
				delivery = d
				exit, exitSource = terminalState(e), captureSourceRejectState(b)
			}
			changes, reports, terminals, highlights := 0, 0, 0, 0
			insideHighlight := false
			e.OnChange = func() {
				changes++
				if insideHighlight {
					d, ok := e.LastInputDelivery()
					if !ok {
						t.Fatal("nested OnChange had no sticky receipt")
					}
					remember(d)
				}
			}
			e.OnError = func(err error) {
				reports++
				if mode != "benign-warning" || err != errLineTooLong {
					t.Fatalf("stale or unexpected navigation error: %v", err)
				}
				d, ok := e.LastInputDelivery()
				if !ok {
					t.Fatal("valid warning lost its settlement receipt")
				}
				remember(d)
			}
			binding := e.BindInputTerminal(func(d InputDelivery) {
				terminals++
				remember(d)
			})
			terminalPreview(t, client, 0, 1, "x")
			preedit, valid := e.LastInputDelivery()
			caret := text.Range{Anchor: 1, Head: 1}
			if !valid || preedit.Reason != InputPreedit || changes != 1 || !e.CompositionActive() || b.Selection() != caret || b.ChangeStamp() != before {
				t.Fatal("fixture needs installed changing preedit at an interior caret")
			}
			if _, hit := e.cache[0]; hit {
				t.Fatal("fixture must enter the real Highlight callback through a cache miss")
			}
			if warns {
				e.ShapeLimit = 128 // produce a local line warning before Highlight
			}
			if mode == "nil-on-change" {
				e.OnChange = nil
			}
			e.Highlight = func(string) []ui.Span {
				highlights++
				if highlights != 1 || e.compositionBuffer != nil || b.InGroup() || b.String() != changed || b.Selection() != caret ||
					b.ChangeStamp().BaselineID != before.BaselineID || b.ChangeStamp().ConfirmedChangeID == before.ConfirmedChangeID || b.History().UndoDepth != 1 {
					t.Fatal("Highlight did not see the already-settled original input")
				}
				entry, entrySource := e.sourceRevealBoundary(), captureSourceRejectState(b)
				if publishes {
					insideHighlight = true
					if mode == "equal" {
						e.ReplaceText(&ui.TextInputRange{Start: 0, End: 1}, "x")
					} else {
						e.ReplaceText(nil, "")
					}
					insideHighlight = false
					if mode == "nil-on-change" {
						d, ok := e.LastInputDelivery()
						if !ok {
							t.Fatal("accepted nil-OnChange site did not publish")
						}
						remember(d)
					}
					if !seen || delivery.Reason != InputAccepted || delivery.Disposition != InputSame || delivery.Failed || delivery.Action == preedit.Action || delivery.Delivery == preedit.Delivery {
						t.Fatalf("one accepted no-byte change should publish a successor: %+v", delivery)
					}
				} else if mode == "b3-noop" {
					unchanged := terminalState(e)
					target := captureLanguageTarget(t, e)
					if changed, err := e.ApplySourceEdits(target, []text.PatchEdit{{Start: 0, End: 1, OldText: "x", NewText: "x"}}, SingleInsertionCaret); err != nil || changed {
						t.Fatalf("B3 equal patch changed inside Highlight: %v %v", changed, err)
					}
					requireTerminalState(t, e, unchanged)
				}
				if e.sourceRevealBoundary() != entry {
					t.Fatal("nested fixture changed the preexisting geometry/attachment guard")
				}
				requireSourceRejectState(t, b, entrySource)
				return nil
			}
			// No frame between the real native preedit and this public action.
			mods := ui.Modifiers(0)
			if warns {
				mods = ui.Cmd
			}
			if !e.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEnd, Mods: mods}) || highlights != 1 || !seen {
				t.Fatalf("did not exercise complete Highlight boundary: highlights=%d seen=%v", highlights, seen)
			}
			last, ok := e.LastInputDelivery()
			if !ok || !terminalSameDelivery(last, delivery) || delivery.Binding != binding || delivery.Buffer != b ||
				delivery.Action != e.inputActionID || delivery.Delivery != e.inputDeliveryID || delivery.Stamp != b.ChangeStamp() ||
				e.nativeClient != client || !e.InputBindingCurrent(binding) || b.String() != changed || b.InGroup() || b.History().UndoDepth != 1 {
				t.Fatalf("outer navigation overwrote the latest action, buffer or native state: %+v", last)
			}
			wantChanges, wantTerminals, wantReports, wantSelection := 1, 0, 0, caret
			if publishes {
				if mode != "nil-on-change" {
					wantChanges++
				}
			} else {
				wantSelection = text.Range{Anchor: b.LenUTF16(), Head: b.LenUTF16()}
				if warns {
					wantReports = 1
				} else {
					wantTerminals = 1
				}
				if delivery.Reason != InputSettled || delivery.Disposition != InputRealCommit || delivery.Failed != warns {
					t.Fatalf("benign Highlight blocked a valid settlement: %+v", delivery)
				}
			}
			if b.Selection() != wantSelection || changes != wantChanges || terminals != wantTerminals || reports != wantReports {
				t.Fatalf("selection or delivery counts: selection=%+v changes=%d terminals=%d reports=%d", b.Selection(), changes, terminals, reports)
			}
			requireTerminalState(t, e, exit)
			requireSourceRejectState(t, b, exitSource)
		})
	}
}
