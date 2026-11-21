package editor

import (
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

// Every positive transport case uses the installed adapter and the production
// controller. These checks run before any subsequent Frame can repair a client.
type nativeCancelFixture struct {
	e      *Editor
	tester *ui.Tester
	views  int
}

func newNativeCancelFixture(t *testing.T, b *text.Buffer) *nativeCancelFixture {
	t.Helper()
	f := &nativeCancelFixture{e: New(b)}
	f.tester = ui.NewTester(func(c *ui.Context) { f.views++; f.e.View(c) }, 600, 280)
	if err := f.tester.Click(f.e.Label); err != nil {
		t.Fatal(err)
	}
	if active, noIME := f.tester.TextInputActive(); !active || noIME || f.e.nativeClient == nil {
		t.Fatal("missing actual installed native client")
	}
	return f
}

func (f *nativeCancelFixture) batch(t *testing.T, fn func(*ui.TesterTextInput)) {
	t.Helper()
	views, client := f.views, f.e.nativeClient
	err := f.tester.TextInputNoFrame(func(n *ui.TesterTextInput) {
		fn(n)
		if f.views != views || f.e.nativeClient != client {
			t.Fatal("batch rebuilt View or replaced the installed Editor client")
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.views != views || f.e.nativeClient != client {
		t.Fatal("native batch needed a later View to repair its attachment")
	}
}

func requireNativeCancelReady(t *testing.T, c *nativeInputClient, target bool) ui.CompositionCancelStamp {
	t.Helper()
	stamp, availability := c.CompositionCancelState()
	if availability != ui.CancelReady || stamp.HasTarget != target {
		t.Fatalf("cancel state = %+v/%v, want target %t", stamp, availability, target)
	}
	return stamp
}

func requireNativeCancelRefused(t *testing.T, e *Editor, c *nativeInputClient, expected ui.CompositionCancelStamp) {
	t.Helper()
	before, admission, receipt := terminalState(e), currentAdmissionState(e), c.compositionReceipt
	if _, handled := c.CancelComposition(expected); handled {
		t.Fatal("stale/foreign cancellation was handled")
	}
	requireTerminalState(t, e, before)
	if !reflect.DeepEqual(currentAdmissionState(e), admission) || c.compositionReceipt != receipt {
		t.Fatal("failed cancellation changed retained state or receipt")
	}
}

func TestNativeCancelRollbackThenTextNoFrame(t *testing.T) {
	for _, multi := range []bool{false, true} {
		t.Run("multi-"+strconv.FormatBool(multi), func(t *testing.T) {
			const source = "🙂a\r\n中🙂z"
			b := text.New(source)
			admissionRedoFixture(t, b)
			f := newNativeCancelFixture(t, b)
			e, client := f.e, f.e.nativeClient
			ranges, primary := []text.Range{{Anchor: 8, Head: 5}}, 0
			if multi {
				ranges, primary = []text.Range{{Anchor: 2, Head: 0}, {Anchor: 8, Head: 5}, {Anchor: 9, Head: 9}}, 1
			}
			start := setSelections(t, b, ranges, primary)
			history, confirmed := b.History(), b.ChangeStamp()
			version, selectionGen := b.Version(), b.SelectionGeneration()
			changes, terminals := 0, 0
			var deliveries []InputDelivery
			e.OnError = func(err error) { t.Fatal(err) }
			e.OnChange = func() {
				changes++
				d, ok := e.LastInputDelivery()
				if !ok {
					t.Fatal("missing complete delivery")
				}
				deliveries = append(deliveries, d)
				if d.Reason == InputPreedit {
					requireNativeCancelReady(t, client, true)
					if client.compositionReceipt != e.compositionStamp {
						t.Fatal("receipt was not installed before OnChange")
					}
				}
			}
			e.BindInputTerminal(func(InputDelivery) { terminals++ })
			f.batch(t, func(n *ui.TesterTextInput) {
				if !n.Start() || !n.Preedit(nil, "拼🙂", ui.TextInputRange{Start: 3, End: 3}) {
					t.Fatal("preedit failed")
				}
				old := n.Capture()
				token := e.compositionToken
				stamp := requireNativeCancelReady(t, client, true)
				wrong := stamp
				wrong.Generation++
				requireNativeCancelRefused(t, e, client, wrong)
				if !n.Cancel() {
					t.Fatal("native cancel failed")
				}
				if b.String() != source || b.InGroup() || b.OwnsComposition(token) || e.compositionBuffer != nil || e.compositionError != nil {
					t.Fatal("cancel did not retire precisely the native owner")
				}
				requireSelections(t, b, start)
				if b.History() != history || !b.CanRedo() || b.ChangeStamp() != confirmed || b.Version() <= version || b.SelectionGeneration() <= selectionGen {
					t.Fatal("rollback lost redo, changed confirmed identity, or rewound counters")
				}
				after := requireNativeCancelReady(t, client, false)
				if after.Generation != stamp.Generation+1 || changes != 2 || terminals != 0 || len(deliveries) != 2 ||
					deliveries[0].Recovery != InputRecoveryProvisional || deliveries[1].Reason != InputOwnCancel ||
					deliveries[1].Recovery != InputRecoveryRollback || deliveries[1].Disposition != InputSame {
					t.Fatal("cancel publication/recovery count differs from actual rollback")
				}
				beforeLate := terminalState(e)
				if n.Result(nil, "late") || n.ResultFor(old, nil, "late") || n.End() || n.Result(nil, "later") || n.Cancel() {
					t.Fatal("cancelled session accepted a late result/End/repeated cancel")
				}
				requireTerminalState(t, e, beforeLate)
				if !n.Reset() || !n.Text("Q") || b.String() != "🙂a\r\nQz" || changes != 3 || b.ChangeStamp() == confirmed {
					t.Fatal("ordinary text did not work immediately on the same client")
				}
				if !n.Start() || !n.Preedit(nil, "r", ui.TextInputRange{Start: 1, End: 1}) || !n.Result(nil, "終") {
					t.Fatal("fresh explicit composition failed")
				}
				if b.String() != "🙂a\r\nQ終z" || b.InGroup() || changes != 5 || terminals != 0 || n.ResultFor(old, nil, "old") {
					t.Fatal("fresh composition inherited old cancellation isolation")
				}
			})
		})
	}
	t.Run("equal-preedit-restores-directed-set-and-real-redo", func(t *testing.T) {
		b := text.New("a a")
		admissionRedoFixture(t, b)
		f := newNativeCancelFixture(t, b)
		e, client := f.e, f.e.nativeClient
		start := setSelections(t, b, []text.Range{{Anchor: 1, Head: 0}, {Anchor: 3, Head: 2}}, 1)
		version, confirmed, history := b.Version(), b.ChangeStamp(), b.History()
		changes := 0
		e.OnChange = func() {
			changes++
			d, _ := e.LastInputDelivery()
			if d.Recovery != InputRecoveryNone || d.Disposition != InputSame {
				t.Fatal("equal preedit/cancel invented recovery or a commit")
			}
		}
		f.batch(t, func(n *ui.TesterTextInput) {
			if !n.Start() || !n.Preedit(nil, "a", ui.TextInputRange{}) || !n.Cancel() {
				t.Fatal("equal preedit cancellation failed")
			}
			requireSelections(t, b, start)
			if b.String() != "a a" || b.Version() != version || b.ChangeStamp() != confirmed || b.History() != history || changes != 2 {
				t.Fatal("equal preedit/cancel changed bytes, history, or callback policy")
			}
			requireNativeCancelReady(t, client, false)
		})
		if !b.Redo() || b.String() != "a a!" || !b.Undo() || b.String() != "a a" {
			t.Fatal("native cancellation damaged the actual saved redo entry")
		}
	})
}

func TestNativeCancelFirstRefusalOwnsExactTokenNoFrame(t *testing.T) {
	for _, mode := range []string{"wrong-range", "size", "history-no-mark", "history-live-mark"} {
		t.Run(mode, func(t *testing.T) {
			b := text.NewWithLimit("ab cd", 9)
			admissionRedoFixture(t, b)
			if mode == "history-no-mark" || mode == "history-live-mark" {
				b.SetHistoryLimits(10, 2*(strconv.IntSize/4)+2)
			}
			f := newNativeCancelFixture(t, b)
			e, client := f.e, f.e.nativeClient
			start := setSelections(t, b, []text.Range{{Anchor: 1, Head: 1}, {Anchor: 4, Head: 4}}, 0)
			history, confirmed, version := b.History(), b.ChangeStamp(), b.Version()
			changes, terminals, errorsSeen := 0, 0, 0
			var issue error
			e.OnChange = func() { changes++ }
			e.OnError = func(err error) {
				errorsSeen++
				issue = err
				requireNativeCancelReady(t, client, true)
				if !b.OwnsComposition(client.compositionReceipt.token) || client.compositionReceipt != e.compositionStamp {
					t.Fatal("first refusal published before its exact native receipt")
				}
			}
			e.BindInputTerminal(func(d InputDelivery) {
				terminals++
				if d.Reason != InputOwnCancel || d.Recovery != InputRecoveryNone || d.Disposition != InputSame || d.Failed || e.compositionBuffer != nil || b.InGroup() {
					t.Fatal("markless retirement delivered an intermediate/fake change")
				}
			})
			f.batch(t, func(n *ui.TesterTextInput) {
				var replacement *ui.TextInputRange
				value, wantIssue := "拼🙂", text.ErrSizeLimit
				if mode == "wrong-range" {
					replacement = &ui.TextInputRange{Start: 4, End: 4}
					value, wantIssue = "x", text.ErrRange
				}
				if mode == "history-live-mark" {
					value = "x"
				}
				if !n.Start() || !n.Preedit(replacement, value, ui.TextInputRange{}) {
					t.Fatal("refusal did not retain a tracked native token")
				}
				if mode == "history-no-mark" || mode == "history-live-mark" {
					if !n.Result(nil, "xy") {
						t.Fatal("history refusal lost the native retry route")
					}
					wantIssue = text.ErrCompositionHistoryLimit
				}
				live := mode == "history-live-mark"
				if !errors.Is(issue, wantIssue) || e.compositionMarked != live || !b.OwnsComposition(e.compositionToken) {
					t.Fatalf("wrong refusal/mark/owner: %v", issue)
				}
				stamp := requireNativeCancelReady(t, client, true)
				wrong := stamp
				wrong.HasTarget = false
				requireNativeCancelRefused(t, e, client, wrong)
				if !n.Cancel() {
					t.Fatal("first refusal could not cancel its own token")
				}
				if b.String() != "ab cd" || b.InGroup() || b.History() != history || b.ChangeStamp() != confirmed {
					t.Fatal("refusal cancellation modified baseline/history/confirmed state")
				}
				requireSelections(t, b, start)
				wantChanges, wantTerminals, wantErrors := 0, 1, 1
				if mode == "history-no-mark" {
					wantErrors = 2
				}
				if live {
					wantChanges, wantTerminals = 2, 0
				}
				if changes != wantChanges || terminals != wantTerminals || errorsSeen != wantErrors || !live && b.Version() != version {
					t.Fatalf("fake callback/rollback: changes %d, terminals %d, errors %d", changes, terminals, errorsSeen)
				}
				requireNativeCancelRefused(t, e, client, stamp)
			})
		})
	}
	t.Run("short-retry-without-cancel", func(t *testing.T) {
		b := text.NewWithLimit("ab cd", 9)
		f := newNativeCancelFixture(t, b)
		e := f.e
		setSelections(t, b, []text.Range{{Anchor: 1, Head: 1}, {Anchor: 4, Head: 4}}, 0)
		var issue error
		e.OnError = func(err error) { issue = err }
		f.batch(t, func(n *ui.TesterTextInput) {
			if !n.Start() || !n.Preedit(nil, "拼🙂", ui.TextInputRange{}) || !errors.Is(issue, text.ErrSizeLimit) {
				t.Fatal("initial refusal missing")
			}
			token := e.compositionToken
			if !n.Result(nil, "你") || !errors.Is(issue, text.ErrSizeLimit) || !b.OwnsComposition(token) {
				t.Fatal("aggregate refusal lost retry token")
			}
			if !n.Result(nil, "xy") || b.String() != "axyb cxyd" || b.InGroup() {
				t.Fatal("shorter retry failed")
			}
		})
	})
	t.Run("tokenless-single-refusal-keeps-ordinary-retry", func(t *testing.T) {
		b := text.NewWithLimit("ab", 3)
		f := newNativeCancelFixture(t, b)
		e, client := f.e, f.e.nativeClient
		if err := b.SetSelection(2, 2); err != nil {
			t.Fatal(err)
		}
		var issue error
		changes, errorsSeen := 0, 0
		e.OnChange = func() { changes++ }
		e.OnError = func(err error) {
			errorsSeen++
			issue = err
			stamp, availability := client.CompositionCancelState()
			if !errors.Is(err, text.ErrSizeLimit) || availability != ui.CancelReady || !stamp.HasTarget ||
				client.compositionReceipt != e.compositionStamp || client.compositionReceipt.token != (text.CompositionToken{}) {
				t.Fatal("native refusal receipt was not bound before OnError")
			}
		}
		f.batch(t, func(n *ui.TesterTextInput) {
			if !n.Start() {
				t.Fatal("start failed")
			}
			if !n.Preedit(nil, "拼🙂", ui.TextInputRange{}) || !errors.Is(issue, text.ErrSizeLimit) || e.composing || b.InGroup() {
				t.Fatal("tokenless refusal was not retained for native retry")
			}
			beforeEdit, beforeTerminal, beforeAdmission := currentEditState(b), terminalState(e), currentAdmissionState(e)
			beforeError, beforeStamp, beforeReceipt := issue, e.compositionStamp, client.compositionReceipt
			if e.compositionBuffer != b || e.compositionToken != (text.CompositionToken{}) || !errors.Is(e.InputActionError(), text.ErrSizeLimit) {
				t.Fatal("tokenless budget error/owner was not retained")
			}
			stamp := requireNativeCancelReady(t, client, true)
			if n.Cancel() {
				t.Fatal("tokenless refusal was reported as a handled cancellation")
			}
			requireNativeCancelRefused(t, e, client, stamp)
			requireEditState(t, b, beforeEdit)
			requireTerminalState(t, e, beforeTerminal)
			if !reflect.DeepEqual(currentAdmissionState(e), beforeAdmission) || e.compositionStamp != beforeStamp ||
				client.compositionReceipt != beforeReceipt || issue != beforeError || e.compositionError != beforeError ||
				!errors.Is(e.InputActionError(), text.ErrSizeLimit) || errorsSeen != 1 || changes != 0 {
				t.Fatal("refused Cancel changed the retained error, history, token, or callbacks")
			}
			if !n.Text("x") || b.String() != "abx" || e.compositionBuffer != nil || errorsSeen != 1 || changes != 1 {
				t.Fatal("ordinary Text retry no longer works beside native transport retry")
			}
		})
	})
	for _, mode := range []string{"preedit-final", "direct-result", "end-late-final"} {
		t.Run("tokenless-native-"+mode, func(t *testing.T) {
			b := text.NewWithLimit("ab", 3)
			f := newNativeCancelFixture(t, b)
			e, client := f.e, f.e.nativeClient
			if err := b.SetSelection(2, 2); err != nil {
				t.Fatal(err)
			}
			var issue error
			changes, terminals, errorsSeen := 0, 0, 0
			e.OnChange = func() { changes++ }
			e.OnError = func(err error) { errorsSeen++; issue = err }
			e.BindInputTerminal(func(d InputDelivery) {
				terminals++
				if d.Reason != InputSettled || e.composing || e.compositionBuffer != nil || b.InGroup() {
					t.Fatal("normal End did not publish a complete settlement")
				}
			})
			f.batch(t, func(n *ui.TesterTextInput) {
				if !n.Start() || !n.Preedit(nil, "拼🙂", ui.TextInputRange{}) || !errors.Is(issue, text.ErrSizeLimit) {
					t.Fatal("first native budget refusal was not reported")
				}
				beforeEdit, beforeTerminal := currentEditState(b), terminalState(e)
				receipt := client.compositionReceipt
				stamp := requireNativeCancelReady(t, client, true)
				if n.Cancel() {
					t.Fatal("native controller treated refused Cancel as handled")
				}
				requireNativeCancelRefused(t, e, client, stamp)
				requireEditState(t, b, beforeEdit)
				requireTerminalState(t, e, beforeTerminal)
				if client.compositionReceipt != receipt || e.compositionToken != (text.CompositionToken{}) || errorsSeen != 1 {
					t.Fatal("refused Cancel rebound the tokenless receipt")
				}
				switch mode {
				case "preedit-final", "end-late-final":
					if !n.Preedit(nil, "x", ui.TextInputRange{Start: 1, End: 1}) || !e.composing || !e.compositionMarked ||
						e.compositionToken == (text.CompositionToken{}) || !b.OwnsComposition(e.compositionToken) {
						t.Fatal("same-attachment short preedit could not acquire its own native token")
					}
					if requireNativeCancelReady(t, client, true) != stamp || client.compositionReceipt != e.compositionStamp {
						t.Fatal("short preedit changed the transport stamp or failed to bind the newly owned token")
					}
					if mode == "end-late-final" && (!n.End() || terminals != 1 || e.composing || e.compositionBuffer != nil) {
						t.Fatal("normal End did not retain the final-text route")
					}
					if !n.Result(nil, "y") {
						t.Fatal("final text after short native preedit was blocked")
					}
				case "direct-result":
					if !n.Result(nil, "y") {
						t.Fatal("legal short Result after tokenless refusal was blocked")
					}
				}
				wantChanges, wantTerminals := 2, 0
				if mode == "direct-result" {
					wantChanges = 1 // the refused first preedit reports only OnError
				}
				if mode == "end-late-final" {
					wantTerminals = 1
				}
				if b.String() != "aby" || b.InGroup() || e.compositionBuffer != nil || e.compositionError != nil ||
					e.InputActionError() != nil || errorsSeen != 1 || changes != wantChanges ||
					terminals != wantTerminals {
					t.Fatalf("native retry/final state mismatch: text=%q changes=%d terminals=%d errors=%d", b.String(), changes, terminals, errorsSeen)
				}
				requireNativeCancelReady(t, client, false)
			})
		})
	}
	for _, mode := range []string{"selection-ABA", "programmatic"} {
		t.Run("tokenless-native-rejects-"+mode, func(t *testing.T) {
			b := text.NewWithLimit("ab", 3)
			f := newNativeCancelFixture(t, b)
			e, client := f.e, f.e.nativeClient
			if err := b.SetSelection(2, 2); err != nil {
				t.Fatal(err)
			}
			changes, errorsSeen, terminals := 0, 0, 0
			e.OnChange = func() { changes++ }
			e.OnError = func(err error) {
				if !errors.Is(err, text.ErrSizeLimit) {
					t.Fatal(err)
				}
				errorsSeen++
			}
			e.BindInputTerminal(func(InputDelivery) { terminals++ })
			f.batch(t, func(n *ui.TesterTextInput) {
				if !n.Start() {
					t.Fatal("native start failed")
				}
				if mode == "programmatic" {
					e.SetMarkedText(nil, "拼🙂", ui.TextInputRange{})
				} else {
					// Both runtimes retain the error. Only the repaired one tracks
					// the receipt, which the raw selection ABA must invalidate.
					n.Preedit(nil, "拼🙂", ui.TextInputRange{})
					if err := b.SetSelection(0, 0); err != nil {
						t.Fatal(err)
					}
					if err := b.SetSelection(2, 2); err != nil {
						t.Fatal(err)
					}
				}
				stamp, availability := client.CompositionCancelState()
				if !stamp.HasTarget || availability != ui.CancelUnavailable || !errors.Is(e.InputActionError(), text.ErrSizeLimit) {
					t.Fatal("foreign or stale refusal appeared current")
				}
				beforeEdit, beforeTerminal := currentEditState(b), terminalState(e)
				receipt := client.compositionReceipt
				if n.Preedit(nil, "x", ui.TextInputRange{}) || n.Result(nil, "y") || n.End() || n.Cancel() {
					t.Fatal("native transport acquired a foreign or stale refusal")
				}
				requireEditState(t, b, beforeEdit)
				requireTerminalState(t, e, beforeTerminal)
				if client.compositionReceipt != receipt || errorsSeen != 1 || changes != 0 || terminals != 0 {
					t.Fatal("rejected native transport changed the receipt or called back")
				}
			})
		})
	}
}

func TestNativeCancelRejectsStaleReceipt(t *testing.T) {
	for _, mode := range []string{"selection", "selection-ABA", "replacement-token", "native-successor", "buffer-ABA", "readonly-ABA", "view-release", "replacement-client"} {
		t.Run(mode, func(t *testing.T) {
			b := text.New("a b")
			f := newNativeCancelFixture(t, b)
			e, client := f.e, f.e.nativeClient
			setSelections(t, b, []text.Range{{Anchor: 1, Head: 0}, {Anchor: 3, Head: 2}}, 1)
			var old ui.TesterTextInputCapture
			f.batch(t, func(n *ui.TesterTextInput) {
				if !n.Start() || !n.Preedit(nil, "p", ui.TextInputRange{}) {
					t.Fatal("preedit failed")
				}
				old = n.Capture()
			})
			expected := requireNativeCancelReady(t, client, true)
			var foreignToken text.CompositionToken
			foreign := text.New("foreign")
			switch mode {
			case "selection", "selection-ABA":
				primary := b.Selection()
				if err := b.SetSelection(0, 0); err != nil {
					t.Fatal(err)
				}
				if mode == "selection-ABA" {
					if err := b.SetSelection(primary.Anchor, primary.Head); err != nil {
						t.Fatal(err)
					}
				}
			case "replacement-token":
				if err := b.CancelCompositionGroup(e.compositionToken); err != nil {
					t.Fatal(err)
				}
				var err error
				foreignToken, err = b.BeginCompositionGroup()
				if err != nil {
					t.Fatal(err)
				}
			case "native-successor":
				f.batch(t, func(n *ui.TesterTextInput) {
					if !n.Cancel() || !n.Start() || !n.Preedit(nil, "q", ui.TextInputRange{}) {
						t.Fatal("new native owner failed")
					}
				})
			case "buffer-ABA", "replacement-client":
				var err error
				foreignToken, err = foreign.BeginCompositionGroup()
				if err != nil {
					t.Fatal(err)
				}
				e.SetBuffer(foreign)
				e.SetBuffer(b)
			case "readonly-ABA":
				e.SetReadOnly(true)
				e.SetReadOnly(false)
			case "view-release":
				e.ReleaseView()
			}
			// Lifecycle operations occur outside a stable no-frame batch.
			if mode == "replacement-client" {
				f.tester.Frame()
				if e.nativeClient == client {
					t.Fatal("attachment ABA reused its client")
				}
				before := terminalState(e)
				if _, availability := e.nativeClient.CompositionCancelState(); availability != ui.CancelUnavailable {
					t.Fatal("replacement client acquired old token")
				}
				requireTerminalState(t, e, before)
			}
			before, foreignBefore, receipt := terminalState(e), currentEditState(foreign), client.compositionReceipt
			for i := 0; i < 3; i++ {
				_, availability := client.CompositionCancelState()
				if mode != "native-successor" && availability != ui.CancelUnavailable {
					t.Fatal("stale receipt remained available")
				}
			}
			requireTerminalState(t, e, before)
			requireNativeCancelRefused(t, e, client, expected)
			if mode == "selection" || mode == "selection-ABA" || mode == "replacement-token" || mode == "native-successor" {
				f.batch(t, func(n *ui.TesterTextInput) {
					if n.CancelFor(old) || n.ResultFor(old, nil, "old") || n.EndFor(old) {
						t.Fatal("old native handle reached a stale/replacement token")
					}
				})
			}
			requireTerminalState(t, e, before)
			requireEditState(t, foreign, foreignBefore)
			if client.compositionReceipt != receipt {
				t.Fatal("query/refusal rebound a stale native receipt")
			}
			if mode == "replacement-token" && !b.OwnsComposition(foreignToken) || mode == "buffer-ABA" && !foreign.OwnsComposition(foreignToken) {
				t.Fatal("foreign token was cancelled")
			}
		})
	}
	for _, finish := range []string{"result", "end-late-result", "preedit-result"} {
		t.Run("editor-select-all-refreshes-owned-receipt/"+finish, func(t *testing.T) {
			b := text.New("ab")
			f := newNativeCancelFixture(t, b)
			e, client := f.e, f.e.nativeClient
			if err := b.SetSelection(2, 2); err != nil {
				t.Fatal(err)
			}
			start := b.SelectionSet()
			changes, terminals := 0, 0
			e.OnChange = func() { changes++ }
			e.OnError = func(err error) { t.Fatal(err) }
			e.BindInputTerminal(func(d InputDelivery) {
				terminals++
				if d.Reason != InputSettled || e.composing || e.compositionBuffer != nil || b.InGroup() {
					t.Fatal("End published before completing the owned composition")
				}
			})
			f.batch(t, func(n *ui.TesterTextInput) {
				if !n.Start() || !n.Preedit(nil, "p", ui.TextInputRange{}) {
					t.Fatal("native preedit failed")
				}
				before := e.compositionStamp
				if before != client.compositionReceipt || before != e.inputStampForOwner(client.owner) ||
					before.token != e.compositionToken || !b.OwnsComposition(before.token) {
					t.Fatal("fixture did not start from the exact owned native receipt")
				}
				if !n.Reset() || !client.handleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyA, Mods: ui.Cmd}) {
					t.Fatal("Editor Ctrl+A route was not handled")
				}
				after := before
				after.selectionGen, after.primary = b.SelectionGeneration(), b.Selection()
				if after.selectionGen == before.selectionGen || after != e.compositionStamp || client.compositionReceipt != after ||
					!b.OwnsComposition(before.token) || after.primary != (text.Range{Anchor: 0, Head: b.LenUTF16()}) ||
					e.marked != (ui.TextInputRange{Start: 2, End: 3}) || changes != 1 || terminals != 0 {
					t.Fatal("legal Editor selectAll did not transfer the exact still-owned receipt")
				}
				requireNativeCancelReady(t, client, true)
				wantChanges, wantTerminals := 2, 0
				if finish == "preedit-result" {
					if !n.Preedit(nil, "q", ui.TextInputRange{Start: 1, End: 1}) || !b.OwnsComposition(before.token) {
						t.Fatal("same-token preedit after Editor selectAll was blocked")
					}
					wantChanges++
				}
				if finish == "end-late-result" {
					if !n.End() || terminals != 1 || b.String() != "abp" || b.InGroup() || b.OwnsComposition(before.token) {
						t.Fatal("normal End after Editor selectAll lost its late-final route")
					}
					wantTerminals++
				}
				if !n.Result(nil, "Z") || b.String() != "abZ" || b.InGroup() || e.compositionBuffer != nil ||
					e.compositionError != nil || b.OwnsComposition(before.token) || changes != wantChanges || terminals != wantTerminals {
					t.Fatalf("selectAll native completion mismatch: %q changes=%d terminals=%d", b.String(), changes, terminals)
				}
				requireNativeCancelReady(t, client, false)
			})
			if !b.Undo() {
				t.Fatal("selectAll completion lost Undo")
			}
			if finish == "end-late-result" {
				if b.String() != "abp" || b.Selection() != (text.Range{Anchor: 0, Head: 3}) || !b.Undo() {
					t.Fatal("normal End lost the independent late-final or preedit Undo")
				}
			}
			if b.String() != "ab" || b.CanUndo() {
				t.Fatal("selectAll completion changed the original history")
			}
			requireSelections(t, b, start)
		})
	}
	for _, mode := range []string{"raw-selection-then-select-all", "raw-selection-ABA-then-select-all"} {
		t.Run("raw-selection-cannot-borrow-select-all-refresh/"+mode, func(t *testing.T) {
			b := text.New("ab")
			f := newNativeCancelFixture(t, b)
			e, client := f.e, f.e.nativeClient
			if err := b.SetSelection(2, 2); err != nil {
				t.Fatal(err)
			}
			f.batch(t, func(n *ui.TesterTextInput) {
				if !n.Start() || !n.Preedit(nil, "p", ui.TextInputRange{}) {
					t.Fatal("native preedit failed")
				}
			})
			staleReceipt, token := client.compositionReceipt, e.compositionToken
			primary := b.Selection()
			if err := b.SetSelection(0, 0); err != nil {
				t.Fatal(err)
			}
			if mode == "raw-selection-ABA-then-select-all" {
				if err := b.SetSelection(primary.Anchor, primary.Head); err != nil {
					t.Fatal(err)
				}
			}
			if !client.handleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyA, Mods: ui.Cmd}) {
				t.Fatal("Editor Ctrl+A route was not handled")
			}
			before, admission := terminalState(e), currentAdmissionState(e)
			if client.compositionReceipt != staleReceipt || e.compositionStamp == staleReceipt ||
				!e.compositionBuffer.OwnsComposition(token) {
				t.Fatal("stale/raw receipt was refreshed by selectAll")
			}
			if _, availability := client.CompositionCancelState(); availability != ui.CancelUnavailable {
				t.Fatal("raw selection or its ABA gained native receipt authority")
			}
			f.batch(t, func(n *ui.TesterTextInput) {
				if n.Preedit(nil, "q", ui.TextInputRange{}) || n.Result(nil, "Z") || n.End() || n.Cancel() {
					t.Fatal("stale native callback was accepted after raw selection/selectAll")
				}
			})
			requireTerminalState(t, e, before)
			if !reflect.DeepEqual(currentAdmissionState(e), admission) || client.compositionReceipt != staleReceipt ||
				!e.compositionBuffer.OwnsComposition(token) {
				t.Fatal("rejected stale callback changed its retained owner or exact receipt")
			}
		})
	}
	for _, mode := range []string{"ordinary-group", "foreign-composition", "programmatic-preedit"} {
		t.Run(mode, func(t *testing.T) {
			b := text.New("ab")
			f := newNativeCancelFixture(t, b)
			e, client := f.e, f.e.nativeClient
			var token text.CompositionToken
			var err error
			switch mode {
			case "ordinary-group":
				err = b.BeginGroup()
			case "foreign-composition":
				token, err = b.BeginCompositionGroup()
			case "programmatic-preedit":
				e.SetMarkedText(nil, "program", ui.TextInputRange{})
				token = e.compositionToken
			}
			if err != nil {
				t.Fatal(err)
			}
			before, receipt := terminalState(e), client.compositionReceipt
			stamp, availability := client.CompositionCancelState()
			if availability != ui.CancelUnavailable {
				t.Fatal("getter acquired foreign/programmatic ownership")
			}
			requireNativeCancelRefused(t, e, client, stamp)
			f.batch(t, func(n *ui.TesterTextInput) {
				if !n.Start() {
					t.Fatal("fixture session failed")
				}
				if n.Preedit(nil, "native", ui.TextInputRange{}) || n.Cancel() || n.Result(nil, "native") || n.End() {
					t.Fatal("transport acquired a foreign owner")
				}
			})
			requireTerminalState(t, e, before)
			if client.compositionReceipt != receipt || !b.InGroup() || token != (text.CompositionToken{}) && !b.OwnsComposition(token) {
				t.Fatal("foreign retained owner changed")
			}
		})
	}
	t.Run("generation-exhaustion-retires-before-reuse", func(t *testing.T) {
		b := text.New("ab")
		f := newNativeCancelFixture(t, b)
		e, client := f.e, f.e.nativeClient
		e.compositionGeneration = ^uint64(0) - 1
		f.batch(t, func(n *ui.TesterTextInput) {
			if !n.Start() || !n.Preedit(nil, "p", ui.TextInputRange{}) {
				t.Fatal("last generation failed")
			}
		})
		stamp := requireNativeCancelReady(t, client, true)
		if stamp.Generation != ^uint64(0) {
			t.Fatal("exhaustion fixture did not reach the last generation")
		}
		// Supplemental client-boundary control outside the stable-attachment
		// batch: exhaustion intentionally retires that attachment.
		after, handled := client.CancelComposition(stamp)
		if !handled || after.HasTarget || after.Generation != 0 || e.nativeClient != nil {
			t.Fatal("generation wrapped on a reusable attachment")
		}
		if _, availability := client.CompositionCancelState(); availability != ui.CancelUnavailable {
			t.Fatal("exhausted client revived")
		}
		f.tester.Frame()
		if e.nativeClient == nil || e.nativeClient == client {
			t.Fatal("fresh attachment was not installed")
		}
		requireNativeCancelReady(t, e.nativeClient, false)
		requireNativeCancelRefused(t, e, client, stamp)
	})
}

func TestNativeCancelCallbackSuccessorSurvives(t *testing.T) {
	for _, mode := range []string{"change-successor", "terminal-successor", "observer-successor", "buffer", "readonly-ABA"} {
		t.Run(mode, func(t *testing.T) {
			b := text.New("a b")
			f := newNativeCancelFixture(t, b)
			e, client := f.e, f.e.nativeClient
			setSelections(t, b, []text.Range{{Anchor: 1, Head: 0}, {Anchor: 3, Head: 2}}, 1)
			armed, called, terminals := false, 0, 0
			var exit terminalTestState
			var exitAdmission admissionState
			var exitReceipt inputStamp
			var successor text.CompositionToken
			var boundary func()
			e.OnChange = func() {
				d, _ := e.LastInputDelivery()
				if armed && d.Reason == InputOwnCancel {
					boundary()
				}
			}
			e.OnError = func(err error) {
				if mode != "terminal-successor" || !errors.Is(err, text.ErrRange) {
					t.Fatal(err)
				}
				requireNativeCancelReady(t, client, true)
			}
			e.BindInputTerminal(func(d InputDelivery) {
				terminals++
				if armed && d.Reason == InputOwnCancel {
					boundary()
				}
			})
			f.batch(t, func(n *ui.TesterTextInput) {
				boundary = func() {
					armed = false
					called++
					switch mode {
					case "buffer":
						e.SetBuffer(text.New("replacement"))
					case "readonly-ABA":
						e.SetReadOnly(true)
						e.SetReadOnly(false)
					default:
						if mode == "observer-successor" {
							e.BindInputTerminal(func(InputDelivery) { t.Fatal("unexpected terminal tail") })
						}
						if !n.Start() || !n.Preedit(nil, "s", ui.TextInputRange{}) {
							t.Fatal("callback could not start its native successor")
						}
						successor = e.compositionToken
					}
					exit, exitAdmission, exitReceipt = terminalState(e), currentAdmissionState(e), client.compositionReceipt
				}
				var replacement *ui.TextInputRange
				if mode == "terminal-successor" {
					replacement = &ui.TextInputRange{Start: 0, End: 1}
				}
				if !n.Start() || !n.Preedit(replacement, "p", ui.TextInputRange{}) {
					t.Fatal("initial preedit/refusal failed")
				}
				old := n.Capture()
				armed = true
				if n.Cancel() {
					t.Fatal("toolkit accepted an old cancellation after callback replacement")
				}
				if called != 1 {
					t.Fatal("cancel callback did not run exactly once")
				}
				requireTerminalState(t, e, exit)
				if !reflect.DeepEqual(currentAdmissionState(e), exitAdmission) || client.compositionReceipt != exitReceipt {
					t.Fatal("old cancel overwrote callback-exit input state")
				}
				if n.ResultFor(old, nil, "old") || n.EndFor(old) || n.CancelFor(old) {
					t.Fatal("old session survived callback replacement")
				}
				requireTerminalState(t, e, exit)
				if successor != (text.CompositionToken{}) {
					if !b.OwnsComposition(successor) || !n.Result(nil, "Z") || b.String() != "Z Z" || b.InGroup() {
						t.Fatal("old suppression or cleanup consumed the callback's successor")
					}
				}
				wantTerminals := 0
				if mode == "terminal-successor" {
					wantTerminals = 1
				}
				if terminals != wantTerminals {
					t.Fatal("stale terminal callback followed the final boundary")
				}
			})
		})
	}
	t.Run("local-after-is-captured-before-final-callback", func(t *testing.T) {
		b := text.New("ab")
		f := newNativeCancelFixture(t, b)
		e, client := f.e, f.e.nativeClient
		f.batch(t, func(n *ui.TesterTextInput) {
			if !n.Start() || !n.Preedit(nil, "p", ui.TextInputRange{}) {
				t.Fatal("preedit failed")
			}
		})
		expected := requireNativeCancelReady(t, client, true)
		var exit terminalTestState
		var receipt inputStamp
		e.OnChange = func() {
			d, _ := e.LastInputDelivery()
			if d.Reason != InputOwnCancel {
				return
			}
			client.SetMarkedText(nil, "successor", ui.TextInputRange{})
			exit, receipt = terminalState(e), client.compositionReceipt
		}
		// Supplemental return-value oracle; production revalidation and nested
		// controller dispatch are exercised by the cases immediately above.
		after, handled := client.CancelComposition(expected)
		if !handled || after.HasTarget || after.Generation != expected.Generation+1 {
			t.Fatal("CancelComposition returned a reloaded successor stamp")
		}
		current := requireNativeCancelReady(t, client, true)
		if current.Generation != after.Generation+1 || !b.OwnsComposition(receipt.token) {
			t.Fatal("successor did not retain its distinct token")
		}
		requireTerminalState(t, e, exit)
		if client.compositionReceipt != receipt {
			t.Fatal("old native return path changed the new receipt")
		}
	})
	t.Run("first-refusal-error-is-last", func(t *testing.T) {
		b := text.New("a b")
		f := newNativeCancelFixture(t, b)
		e, client := f.e, f.e.nativeClient
		setSelections(t, b, []text.Range{{Anchor: 1, Head: 0}, {Anchor: 3, Head: 2}}, 1)
		calls := 0
		var exit terminalTestState
		var receipt inputStamp
		f.batch(t, func(n *ui.TesterTextInput) {
			e.OnError = func(err error) {
				calls++
				if calls != 1 || !errors.Is(err, text.ErrRange) {
					t.Fatal("unexpected error tail", err)
				}
				stamp := requireNativeCancelReady(t, client, true)
				if _, handled := client.CancelComposition(stamp); !handled {
					t.Fatal("error callback did not see the pre-captured receipt")
				}
				if !n.Start() || !n.Preedit(nil, "s", ui.TextInputRange{}) {
					t.Fatal("error callback successor failed")
				}
				exit, receipt = terminalState(e), client.compositionReceipt
			}
			if !n.Start() {
				t.Fatal("start failed")
			}
			old := n.Capture()
			if n.Preedit(&ui.TextInputRange{Start: 0, End: 1}, "bad", ui.TextInputRange{}) {
				t.Fatal("reentered old preedit retained its transport authority")
			}
			requireTerminalState(t, e, exit)
			if calls != 1 || client.compositionReceipt != receipt || !b.OwnsComposition(receipt.token) || n.ResultFor(old, nil, "old") {
				t.Fatal("old error action changed its callback successor")
			}
			requireTerminalState(t, e, exit)
			if !n.Result(nil, "Z") || b.String() != "Z Z" {
				t.Fatal("error callback successor could not finish")
			}
		})
	})
	for _, multi := range []bool{false, true} {
		for _, empty := range []bool{false, true} {
			for _, end := range []bool{false, true} {
				name := "normal/multi-" + strconv.FormatBool(multi) + "/empty-" + strconv.FormatBool(empty) + "/end-" + strconv.FormatBool(end)
				t.Run(name, func(t *testing.T) {
					b := text.New("a b")
					f := newNativeCancelFixture(t, b)
					e := f.e
					ranges, primary := []text.Range{{Anchor: 3, Head: 2}}, 0
					if multi {
						ranges, primary = []text.Range{{Anchor: 1, Head: 0}, {Anchor: 3, Head: 2}}, 1
					}
					start := setSelections(t, b, ranges, primary)
					changes, terminals, failures := 0, 0, 0
					e.OnChange = func() { changes++ }
					e.OnError = func(err error) {
						failures++
						if !multi || !end || !errors.Is(err, text.ErrCompositionPending) {
							t.Fatal(err)
						}
					}
					e.BindInputTerminal(func(d InputDelivery) {
						terminals++
						if d.Reason != InputSettled {
							t.Fatal("normal End became cancellation")
						}
					})
					f.batch(t, func(n *ui.TesterTextInput) {
						value := "p"
						if empty {
							value = ""
						}
						if !n.Start() || !n.Preedit(nil, value, ui.TextInputRange{}) {
							t.Fatal("normal preedit failed")
						}
						token := e.compositionToken
						if end {
							if !n.End() {
								t.Fatal("normal End lost its late-final span")
							}
							if multi != b.OwnsComposition(token) {
								t.Fatal("normal End changed pending/settlement policy")
							}
						}
						if !n.Result(nil, "Z") {
							t.Fatal("legitimate final was suppressed")
						}
						want, wantTerminals, wantFailures := "a Z", 0, 0
						if multi {
							want = "Z Z"
						}
						if end && !multi {
							wantTerminals = 1
						}
						if end && multi {
							wantFailures = 1
						}
						if b.String() != want || b.InGroup() || changes != 2 || terminals != wantTerminals || failures != wantFailures {
							t.Fatalf("normal final/End changed: %q, callbacks %d/%d/%d", b.String(), changes, terminals, failures)
						}
					})
					if !b.Undo() {
						t.Fatal("normal final lost Undo")
					}
					if end && !multi && !b.Undo() {
						t.Fatal("single End lost its independent preedit Undo")
					}
					if b.String() != "a b" {
						t.Fatal("normal completion changed rollback history")
					}
					requireSelections(t, b, start)
				})
			}
		}
	}
}
