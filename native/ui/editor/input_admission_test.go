package editor

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

var _ HostFindFocusGuard = inputAdmissionGuardFixture{}

type inputAdmissionGuardFixture struct{}

func (inputAdmissionGuardFixture) HostFindFocusError() error { return ErrHostFindFocusDraft }

func TestHostFindFocusGuardDraftSentinelNeutral(t *testing.T) {
	var guard HostFindFocusGuard = inputAdmissionGuardFixture{}
	if err := guard.HostFindFocusError(); err != ErrHostFindFocusDraft {
		t.Fatalf("draft guard error = %v, want exported neutral draft sentinel", err)
	}
	if ErrHostFindFocusDraft == nil || strings.Contains(ErrHostFindFocusDraft.Error(), "来源") || strings.Contains(ErrHostFindFocusDraft.Error(), "冲突") {
		t.Fatalf("independent draft sentinel mislabels a source conflict: %v", ErrHostFindFocusDraft)
	}
}

func requireInputActionErrorPure(t *testing.T, e *Editor, want error, tester *ui.Tester, focusedLabel string) {
	t.Helper()
	if e == nil || e.Buffer == nil {
		t.Fatal("purity helper requires an attached Editor")
	}
	beforeEditor := captureSourceEditorRejectState(e)
	beforeBuffer := captureSourceRejectState(e.Buffer)
	beforeServices, beforeHandle := e.services, e.handle
	focused := false
	if tester != nil {
		focused = tester.Focused(focusedLabel)
	}
	changes, reports := 0, 0
	previousChange, previousError := e.OnChange, e.OnError
	e.OnChange = func() { changes++ }
	e.OnError = func(error) { reports++ }
	for i := 0; i < 3; i++ {
		got := e.InputActionError()
		if got != want || !errors.Is(got, want) {
			t.Fatalf("InputActionError probe %d = %v (%T), want the owned error %v (%T)", i+1, got, got, want, want)
		}
		if after := captureSourceEditorRejectState(e); !reflect.DeepEqual(beforeEditor, after) {
			t.Fatalf("input admission probe mutated Editor state: before=%+v after=%+v", beforeEditor, after)
		}
		if after := captureSourceRejectState(e.Buffer); !reflect.DeepEqual(beforeBuffer, after) {
			t.Fatalf("input admission probe mutated Buffer state: before=%+v after=%+v", beforeBuffer, after)
		}
		if !reflect.DeepEqual(e.services, beforeServices) || !reflect.DeepEqual(e.handle, beforeHandle) {
			t.Fatal("input admission probe changed UI services or focus handle")
		}
		if changes != 0 || reports != 0 {
			t.Fatalf("input admission probe invoked callbacks: change=%d error=%d", changes, reports)
		}
		if tester != nil && tester.Focused(focusedLabel) != focused {
			t.Fatal("input admission probe changed actual focus")
		}
	}
	e.OnChange, e.OnError = previousChange, previousError
}

func TestEditorInputActionAdmissionNilIdleReadonlyAndLarge(t *testing.T) {
	if got := (*Editor)(nil).InputActionError(); got != errSourceTargetStale {
		t.Fatalf("nil Editor admission = %v, want existing stale error", got)
	}
	withoutBuffer := New(nil)
	if got := withoutBuffer.InputActionError(); got != errSourceTargetStale {
		t.Fatalf("nil Buffer admission = %v, want existing stale error", got)
	}

	for _, tc := range []struct {
		name     string
		readonly bool
		source   string
	}{
		{name: "writable", source: "idle"},
		{name: "readonly", readonly: true, source: "idle"},
		{name: "readonly-large-source", readonly: true, source: strings.Repeat("x", (2<<20)+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := text.New(tc.source)
			e := New(b)
			e.ReadOnly = tc.readonly
			beforeEditor := captureSourceEditorRejectState(e)
			version, selection, selectionGeneration := b.Version(), b.SelectionSet(), b.SelectionGeneration()
			history, canUndo, canRedo, size := b.History(), b.CanUndo(), b.CanRedo(), b.LenBytes()
			if size != len(tc.source) {
				t.Fatalf("fixture source bytes=%d, want %d", size, len(tc.source))
			}
			for i := 0; i < 3; i++ {
				if got := e.InputActionError(); got != nil {
					t.Fatalf("idle admission = %v", got)
				}
			}
			if after := captureSourceEditorRejectState(e); !reflect.DeepEqual(beforeEditor, after) ||
				b.LenBytes() != size || b.Version() != version || b.SelectionGeneration() != selectionGeneration ||
				!reflect.DeepEqual(b.SelectionSet(), selection) || b.History() != history || b.CanUndo() != canUndo || b.CanRedo() != canRedo || b.InGroup() {
				t.Fatal("idle admission changed Editor or Buffer state")
			}
		})
	}
}

func TestEditorInputActionAdmissionAcceptedSingleAndEmptyPreedit(t *testing.T) {
	for _, value := range []string{"拼🙂", ""} {
		name := "nonempty"
		if value == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			b := text.New("abc")
			e := New(b)
			tester := multiTester(t, e)
			client := e.nativeClient
			if client == nil || client != e.nativeTextClient() || client.editor != e || client.owner.buffer != b || !client.current() {
				t.Fatal("fixture is not using the current installed project Editor client")
			}
			rangeToReplace := ui.TextInputRange{Start: 1, End: 2}
			selected := ui.TextInputRange{Start: ui.UTF16Len(value), End: ui.UTF16Len(value)}
			client.SetMarkedText(&rangeToReplace, value, selected)
			if !e.composing || !e.compositionMarked || !b.InGroup() {
				t.Fatal("accepted single preedit was not retained")
			}
			if e.nativeClient != client {
				t.Fatal("native client pointer changed during preedit")
			}
			requireInputActionErrorPure(t, e, text.ErrComposition, tester, e.Label)
			if e.nativeClient != client || !client.current() || !b.OwnsComposition(e.compositionToken) {
				t.Fatal("pure admission changed native client or composition ownership")
			}
			client.handleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape})
			if b.String() != "abc" || b.InGroup() || e.compositionBuffer != nil || !tester.Focused(e.Label) {
				t.Fatal("owned Escape cleanup was not available after admission refusal")
			}
		})
	}
}

func TestEditorInputActionAdmissionInstalledClientFirstRefusalNoFrame(t *testing.T) {
	b := text.NewWithLimit("a", 1)
	// Keep an existing redo branch and directed selection in the refusal oracle.
	if err := b.ReplaceUTF16(0, 1, "b"); err != nil || !b.Undo() {
		t.Fatal("could not prepare redo state", err)
	}
	if err := b.SetSelection(1, 0); err != nil {
		t.Fatal(err)
	}
	e := New(b)
	tester := multiTester(t, e)
	client := e.nativeClient
	if client == nil || client != e.nativeTextClient() || client.editor != e || client.owner.buffer != b || !client.current() {
		t.Fatal("fixture is not using the installed, current native client")
	}
	changes, reports := 0, 0
	e.OnChange = func() { changes++ }
	e.OnError = func(error) { reports++ }
	span := ui.TextInputRange{Start: 0, End: 1}
	client.SetMarkedText(&span, "🙂", ui.TextInputRange{})
	// No Tester frame or other toolkit callback occurs between the native input
	// and these readiness checks: the refusal must be visible immediately.
	retained := e.compositionError
	if retained == nil || !errors.Is(retained, text.ErrSizeLimit) || e.composing || e.compositionMulti ||
		e.compositionMarked || e.compositionToken != (text.CompositionToken{}) || b.InGroup() {
		t.Fatalf("first refused single preedit unexpectedly became active: error=%v composing=%v marked=%v group=%v", retained, e.composing, e.compositionMarked, b.InGroup())
	}
	if e.CompositionActive() {
		t.Fatal("boolean CompositionActive approximation hid markless retained input")
	}
	if _, active := e.MarkedRange(); active {
		t.Fatal("first refused single preedit unexpectedly has a mark")
	}
	if changes != 0 || reports != 1 || e.nativeClient != client || !client.current() {
		t.Fatal("native refusal lost its installed client or emitted unexpected callbacks")
	}
	requireInputActionErrorPure(t, e, retained, tester, e.Label)
	if e.nativeClient != client || !client.current() || e.compositionError != retained ||
		changes != 0 || reports != 1 || b.String() != "a" || !b.CanRedo() {
		t.Fatal("admission changed retained refusal, client, or redo-capable source")
	}

	// The original owner's Escape route clears the markless refusal without
	// taking the Buffer group or focus from another owner.
	if !client.handleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape}) {
		t.Fatal("installed client's own Escape route did not consume retained input")
	}
	if e.compositionBuffer != nil || e.compositionError != nil || b.InGroup() || b.String() != "a" || !b.CanRedo() || !tester.Focused(e.Label) {
		t.Fatal("owned Escape did not preserve source and redo state")
	}

	// The same still-installed client can accept a fresh preedit and its final
	// native text after the refusal has been explicitly canceled.
	client.SetMarkedText(&span, "b", ui.TextInputRange{Start: 1, End: 1})
	if !e.composing || !e.compositionMarked || !b.InGroup() || b.String() != "b" || e.nativeClient != client {
		t.Fatal("same installed client could not start accepted input after Escape")
	}
	client.ReplaceText(nil, "b")
	if e.composing || e.compositionBuffer != nil || b.InGroup() || b.String() != "b" || b.CanRedo() || e.nativeClient != client {
		t.Fatal("same installed client could not deliver final text after admission refusal")
	}
}

func TestEditorInputActionAdmissionFirstRefusedMultiNoMark(t *testing.T) {
	b := text.NewWithLimit("a b", 5)
	e := New(b)
	tester := multiTester(t, e)
	start := setSelections(t, b, []text.Range{{Anchor: 1, Head: 1}, {Anchor: 3, Head: 3}}, 1)
	client := e.nativeClient
	client.SetMarkedText(nil, "拼🙂", ui.TextInputRange{})
	if !e.compositionMulti || !e.MultiCompositionPending() || !b.InGroup() || e.compositionToken == (text.CompositionToken{}) {
		t.Fatal("first refused multi preedit did not retain its owner")
	}
	if _, marked := e.MarkedRange(); marked {
		t.Fatal("first refused multi preedit unexpectedly has a mark")
	}
	if e.CompositionActive() != true {
		t.Fatal("multi pending state must remain visible to the legacy boolean")
	}
	token := e.compositionToken
	requireInputActionErrorPure(t, e, text.ErrCompositionPending, tester, e.Label)
	if !b.OwnsComposition(token) || !e.compositionMulti || e.nativeClient != client || !reflect.DeepEqual(b.SelectionSet(), start) {
		t.Fatal("pure admission changed pending multi input or its complete directed selection")
	}
	if !e.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape}) {
		t.Fatal("multi owner's Escape route did not consume pending input")
	}
	if b.InGroup() || e.compositionBuffer != nil || e.compositionToken != (text.CompositionToken{}) || !reflect.DeepEqual(b.SelectionSet(), start) {
		t.Fatal("multi owner could not cancel after admission refusal")
	}
}

func TestEditorInputActionAdmissionForeignGroupsDoNotStealFocus(t *testing.T) {
	for _, composition := range []bool{false, true} {
		name := "ordinary"
		if composition {
			name = "composition"
		}
		t.Run(name, func(t *testing.T) {
			b := text.New("source")
			e := New(b)
			tester := ui.NewTester(func(c *ui.Context) {
				ui.Button(c, "outside")
				e.View(c)
			}, 480, 240)
			if err := tester.Click("outside"); err != nil {
				t.Fatal(err)
			}
			var token text.CompositionToken
			if composition {
				var err error
				token, err = b.BeginCompositionGroup()
				if err != nil {
					t.Fatal(err)
				}
			} else if err := b.BeginGroup(); err != nil {
				t.Fatal(err)
			}
			if !tester.Focused("outside") {
				t.Fatal("fixture must start with focus outside source")
			}
			requireInputActionErrorPure(t, e, text.ErrGroup, tester, "outside")
			if !b.InGroup() || (composition && !b.OwnsComposition(token)) || !tester.Focused("outside") {
				t.Fatal("input admission stole foreign transaction or focus")
			}
			if composition {
				if err := b.CancelCompositionGroup(token); err != nil {
					t.Fatal(err)
				}
			} else if !b.CancelGroup() {
				t.Fatal("foreign ordinary transaction lost its cancellation route")
			}
		})
	}
}

func TestEditorInputActionAdmissionRetainsOwnedErrorsAcrossAttachmentChanges(t *testing.T) {
	t.Run("first-refusal-error", func(t *testing.T) {
		b := text.NewWithLimit("a", 1)
		e := New(b)
		tester := multiTester(t, e)
		client := e.nativeClient
		span := ui.TextInputRange{Start: 0, End: 1}
		client.SetMarkedText(&span, "🙂", ui.TextInputRange{})
		want := e.compositionError
		if want == nil || !errors.Is(want, text.ErrSizeLimit) {
			t.Fatal("missing actual retained size error")
		}
		other := text.New("replacement")
		e.SetBuffer(other)
		requireInputActionErrorPure(t, e, want, tester, e.Label)
		if e.Buffer != other || e.compositionBuffer != b || e.compositionError != want {
			t.Fatal("admission altered the stale retained input owner or its error")
		}
	})

	for _, name := range []string{"buffer-setter", "readonly-setter"} {
		t.Run(name, func(t *testing.T) {
			b := text.New("abc")
			e := New(b)
			tester := multiTester(t, e)
			client := e.nativeClient
			span := ui.TextInputRange{Start: 1, End: 2}
			client.SetMarkedText(&span, "x", ui.TextInputRange{Start: 1, End: 1})
			if !e.composing || !b.InGroup() {
				t.Fatal("accepted composition fixture missing")
			}
			if name == "buffer-setter" {
				e.SetBuffer(text.New("other"))
			} else {
				e.SetReadOnly(true)
			}
			requireInputActionErrorPure(t, e, text.ErrComposition, tester, e.Label)
			if e.nativeClient != client {
				t.Fatal("admission unexpectedly replaced attached native client")
			}
		})
	}
}
