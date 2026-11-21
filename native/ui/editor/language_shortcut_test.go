package editor

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

var sourceShortcutCases = []struct {
	name   string
	action SourceAction
	mods   ui.Modifiers
	key    ui.Key
}{
	{"complete", SourceActionComplete, ui.Ctrl, ui.KeySpace},
	{"references", SourceActionReferences, ui.Shift, ui.KeyF12},
	{"rename", SourceActionRename, 0, ui.KeyF2},
}

func sourceShortcutEvent(mods ui.Modifiers, key ui.Key, repeat bool) ui.InputEvent {
	return ui.InputEvent{Kind: ui.InputKeyDown, Mods: mods, Key: key, Repeat: repeat}
}

// Include the input notification identities: a language intent must never be
// manufactured as an edit or publish a terminal-input delivery.
func sourceShortcutEditorState(e *Editor) []any {
	return append(captureSourceEditorRejectState(e), e.sourceActionBinding, e.sourceActionShortcutBinding,
		e.inputActionID, e.inputDeliveryID, e.inputDelivery, e.inputDeliveryValid,
		e.terminalBufferEpoch, e.terminalReadonlyEpoch, e.terminalViewEpoch,
		e.terminalRegistration, e.handle, e.services)
}

func TestEditorSourceShortcutKeyOnlyTesterBothRoutes(t *testing.T) {
	for _, registered := range []bool{false, true} {
		name := "first-unclaimed-native-client"
		if registered {
			name = "registered-element"
		}
		t.Run(name, func(t *testing.T) {
			b := text.New("source")
			e := New(b)
			var got []SourceAction
			handler := func(action SourceAction) { got = append(got, action) }
			if registered {
				e.SetSourceActionHandler(handler)
			}
			tester := multiTester(t, e)
			client := e.nativeClient
			if !registered {
				// No frame after installation: the first key must use the actual
				// retained native client's raw HandleInput route.
				e.SetSourceActionHandler(handler)
			}
			binding := e.sourceActionBinding
			before := currentEditState(b)
			for _, tc := range sourceShortcutCases {
				tester.Key(tc.mods, tc.key) // Key-only; TypeKey would inject text.
				tester.KeyRepeat(tc.mods, tc.key)
				tester.KeyRepeat(tc.mods, tc.key)
			}
			// Also prove a later distinct keydown works after held-key repeats.
			tester.Key(ui.Ctrl, ui.KeySpace)
			want := []SourceAction{SourceActionComplete, SourceActionReferences, SourceActionRename, SourceActionComplete}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("key-only route dispatched %v, want %v", got, want)
			}
			requireEditState(t, b, before)
			if !e.SourceActionBindingCurrent(binding) || e.nativeClient != client || !client.current() || !tester.Focused(e.Label) {
				t.Fatal("normal frames/repeats changed lease, installed client or actual focus")
			}
		})
	}
}

func TestEditorSourceShortcutRegisteredElementBeatsWindowAndKeepsAllKeys(t *testing.T) {
	b := text.New("source")
	e := New(b)
	var got []SourceAction
	window := 0
	e.SetSourceActionHandler(func(action SourceAction) { got = append(got, action) })
	tester := ui.NewTester(func(c *ui.Context) {
		// Window conflicts are deliberately registered before the editor.
		for _, tc := range sourceShortcutCases {
			if c.Shortcut(tc.mods, tc.key) {
				window++
			}
		}
		e.View(c)
	}, 600, 280)
	if err := tester.Click(e.Label); err != nil {
		t.Fatal(err)
	}
	for round := 0; round < 2; round++ {
		for _, tc := range sourceShortcutCases {
			tester.Key(tc.mods, tc.key)
			tester.KeyRepeat(tc.mods, tc.key)
		}
	}
	want := []SourceAction{SourceActionComplete, SourceActionReferences, SourceActionRename, SourceActionComplete, SourceActionReferences, SourceActionRename}
	if !reflect.DeepEqual(got, want) || window != 0 {
		t.Fatalf("actual input registrations: source=%v window=%d, want %v and zero", got, window, want)
	}
}

func TestEditorSourceShortcutRoutedKeyKeepsItsRegistration(t *testing.T) {
	for _, scenario := range []string{"same-handler-ABA", "buffer-ABA", "no-previous-registration"} {
		t.Run(scenario, func(t *testing.T) {
			e := New(text.New("source"))
			var got []SourceAction
			handler := func(action SourceAction) { got = append(got, action) }
			old := e.SetSourceActionHandler(handler)
			var fresh SourceActionBinding
			armed := false
			tester := ui.NewTester(func(c *ui.Context) {
				if armed {
					armed = false
					if e.sourceActionShortcutBinding != old || !e.focused {
						t.Fatal("fixture lacks a focused previous registration")
					}
					// The real router already queued F2 for the stable editor-input ID.
					switch scenario {
					case "same-handler-ABA":
						e.SetSourceActionHandler(nil)
					case "buffer-ABA":
						buffer := e.Buffer
						e.SetBuffer(text.New("intermediate"))
						e.SetBuffer(buffer)
					case "no-previous-registration":
						e = New(text.New("successor"))
					}
					prior := e.sourceActionShortcutBinding
					fresh = e.SetSourceActionHandler(handler)
					if e.sourceActionShortcutBinding != prior || !e.SourceActionBindingCurrent(fresh) || e.SourceActionBindingCurrent(old) {
						t.Fatal("setter changed the registered receipt or failed to replace the lease")
					}
				}
				e.View(c)
			}, 600, 280)
			if err := tester.Click(e.Label); err != nil {
				t.Fatal(err)
			}
			armed = true
			tester.Key(0, ui.KeyF2)
			if armed || len(got) != 0 || !e.focused || !tester.Focused(e.Label) || e.sourceActionShortcutBinding != fresh {
				t.Fatal("stale routed F2 dispatched, lost actual focus, or missed successor registration")
			}
			tester.Key(0, ui.KeyF2)
			if !reflect.DeepEqual(got, []SourceAction{SourceActionRename}) {
				t.Fatalf("fresh F2 dispatched %v, want exactly one rename", got)
			}
		})
	}
}

func TestEditorSourceShortcutFocusNegativesAndUnboundProjectField(t *testing.T) {
	source, property := New(text.New("source")), New(text.New("name"))
	source.Label, property.Label = "main source", "rename name"
	calls := 0
	source.SetSourceActionHandler(func(SourceAction) { calls++ })
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Gap(8).Children(func() {
			ui.Button(c, "outside")
			ui.Column(c.Key("source")).Grow(1).Children(func() { source.View(c) })
			ui.Column(c.Key("property")).Grow(1).Children(func() { property.View(c) })
		})
	}, 600, 440)
	for _, label := range []string{"outside", property.Label} {
		if err := tester.Click(label); err != nil {
			t.Fatal(err)
		}
		before := currentEditState(property.Buffer)
		for _, tc := range sourceShortcutCases {
			tester.Key(tc.mods, tc.key)
			tester.KeyRepeat(tc.mods, tc.key)
		}
		if calls != 0 || !tester.Focused(label) {
			t.Fatalf("focus on %q invoked source or moved focus: calls=%d", label, calls)
		}
		requireEditState(t, property.Buffer, before)
	}
	if err := tester.Click(source.Label); err != nil {
		t.Fatal(err)
	}
	tester.Key(ui.Shift, ui.KeyF12)
	if calls != 1 || !tester.Focused(source.Label) {
		t.Fatal("actual source focus did not enable its references intent")
	}
	for _, tc := range sourceShortcutCases {
		if property.HandleInput(sourceShortcutEvent(tc.mods, tc.key, false)) {
			t.Fatalf("unbound project field swallowed %s", tc.name)
		}
	}
}

func TestEditorSourceShortcutExactModifiersAndReservedF12(t *testing.T) {
	e := New(text.New("source"))
	calls := 0
	e.SetSourceActionHandler(func(SourceAction) { calls++ })
	for _, ev := range []ui.InputEvent{
		sourceShortcutEvent(0, ui.KeyF12, false),
		sourceShortcutEvent(ui.Ctrl, ui.KeyF12, false),
		sourceShortcutEvent(ui.Alt|ui.Shift, ui.KeyF12, false),
		sourceShortcutEvent(ui.Ctrl|ui.Shift, ui.KeySpace, false),
		sourceShortcutEvent(ui.Ctrl|ui.Alt, ui.KeySpace, false),
		sourceShortcutEvent(ui.Super, ui.KeySpace, false),
		sourceShortcutEvent(ui.Shift, ui.KeyF2, false),
		sourceShortcutEvent(ui.Ctrl, ui.KeyF2, false),
		{Kind: ui.InputKeyUp, Key: ui.KeyF2},
	} {
		if e.HandleInput(ev) || calls != 0 {
			t.Fatalf("nonbinding event was claimed/dispatched: %+v calls=%d", ev, calls)
		}
	}
	// Exercise the toolkit's reserved F12 too, without relying on whether the
	// inspector is built in. Neither production nor debug may dispatch B6.
	tester := multiTester(t, e)
	tester.Key(0, ui.KeyF12)
	if calls != 0 {
		t.Fatal("Inspector-reserved F12 dispatched a source action")
	}
}

func TestEditorSourceShortcutLeaseLifecyclePureAndCallbackFree(t *testing.T) {
	if (*Editor)(nil).SourceActionBindingCurrent(SourceActionBinding{}) {
		t.Fatal("nil editor accepted a lease")
	}
	for _, transition := range []string{"replace-handler", "revoke", "buffer-ABA", "readonly-ABA", "release", "direct-buffer", "direct-readonly"} {
		t.Run(transition, func(t *testing.T) {
			b := text.New("source")
			e := New(b)
			calls, changes, reports := 0, 0, 0
			e.OnChange = func() { changes++ }
			e.OnError = func(error) { reports++ }
			binding := e.SetSourceActionHandler(func(SourceAction) { calls++ })
			if !e.SourceActionBindingCurrent(binding) || New(b).SourceActionBindingCurrent(binding) {
				t.Fatal("initial lease identity is wrong")
			}
			switch transition {
			case "replace-handler":
				e.SetSourceActionHandler(func(SourceAction) { calls++ })
			case "revoke":
				e.SetSourceActionHandler(nil)
			case "buffer-ABA":
				e.SetBuffer(text.New("other"))
				e.SetBuffer(b)
			case "readonly-ABA":
				e.SetReadOnly(true)
				e.SetReadOnly(false)
			case "release":
				e.ReleaseView()
			case "direct-buffer":
				e.Buffer = text.New("legacy")
			case "direct-readonly":
				e.ReadOnly = true
			}
			before, bufferBefore := sourceShortcutEditorState(e), currentEditState(b)
			for i := 0; i < 3; i++ {
				if e.SourceActionBindingCurrent(binding) {
					t.Fatal("old lease survived lifecycle transition")
				}
			}
			if !reflect.DeepEqual(before, sourceShortcutEditorState(e)) || calls != 0 || changes != 0 || reports != 0 {
				t.Fatal("setter or pure lease checks invoked callbacks or mutated observed state")
			}
			requireEditState(t, b, bufferBefore)
			// A stale saved dispatch never resolves against a new registration.
			e.dispatchSourceAction(binding, SourceActionComplete, false)
			if calls != 0 {
				t.Fatal("stale retained dispatch invoked a replacement handler")
			}
			if transition != "replace-handler" && transition != "revoke" {
				if !e.HandleInput(sourceShortcutEvent(ui.Ctrl, ui.KeySpace, false)) || calls != 0 {
					t.Fatal("recognized stale binding was not consumed without callback")
				}
			}
		})
	}
}

func TestEditorSourceShortcutLeaseIgnoresTextSelectionTerminalAndNormalFrames(t *testing.T) {
	b := text.New("source")
	e := New(b)
	binding := e.SetSourceActionHandler(func(SourceAction) {})
	tester := multiTester(t, e)
	client := e.nativeClient
	e.BindInputTerminal(func(InputDelivery) {})
	if err := b.ReplaceUTF16(0, 0, "x"); err != nil {
		t.Fatal(err)
	}
	if err := b.SetSelection(1, 0); err != nil {
		t.Fatal(err)
	}
	e.SetBuffer(b)
	e.SetReadOnly(false)
	for i := 0; i < 3; i++ {
		tester.Frame()
	}
	if !e.SourceActionBindingCurrent(binding) || e.nativeClient != client {
		t.Fatal("ordinary edits, selection, terminal registration or steady frames churned source lease/client")
	}
}

func TestEditorSourceShortcutInstalledClientReadonlyAndLargeSource(t *testing.T) {
	// No source materialization/admission budget belongs to a keyboard intent.
	b := text.New(strings.Repeat("x", (2<<20)+1))
	e := New(b)
	e.SetReadOnly(true)
	var got []SourceAction
	e.SetSourceActionHandler(func(action SourceAction) { got = append(got, action) })
	client := e.nativeTextClient()
	before := currentEditState(b)
	for _, tc := range sourceShortcutCases {
		if !client.handleInput(sourceShortcutEvent(tc.mods, tc.key, false)) {
			t.Fatalf("idle readonly client did not own %s", tc.name)
		}
		client.handleInput(sourceShortcutEvent(tc.mods, tc.key, true))
	}
	if !reflect.DeepEqual(got, []SourceAction{SourceActionComplete, SourceActionReferences, SourceActionRename}) {
		t.Fatalf("readonly host policy was bypassed or snapshot budget imposed: %v", got)
	}
	requireEditState(t, b, before)
}

func TestEditorSourceShortcutRefusesRetainedInputWithoutEffects(t *testing.T) {
	for _, state := range []string{"foreign-group", "foreign-composition", "accepted-single", "first-refused-single", "pending-multi"} {
		t.Run(state, func(t *testing.T) {
			b := text.NewWithLimit("a b", 5)
			e := New(b)
			calls, changes, reports := 0, 0, 0
			var reported error
			e.SetSourceActionHandler(func(SourceAction) { calls++ })
			tester := multiTester(t, e)
			client := e.nativeClient
			var foreign text.CompositionToken
			switch state {
			case "foreign-group":
				if err := b.BeginGroup(); err != nil {
					t.Fatal(err)
				}
			case "foreign-composition":
				var err error
				foreign, err = b.BeginCompositionGroup()
				if err != nil {
					t.Fatal(err)
				}
			case "accepted-single":
				client.SetMarkedText(nil, "x", ui.TextInputRange{Start: 1, End: 1})
			case "first-refused-single":
				client.SetMarkedText(nil, "too long", ui.TextInputRange{})
				if e.composing || e.compositionMarked || b.InGroup() || e.compositionError == nil {
					t.Fatal("fixture did not retain a markless first refusal")
				}
			case "pending-multi":
				setSelections(t, b, []text.Range{{Anchor: 1, Head: 1}, {Anchor: 3, Head: 3}}, 1)
				client.SetMarkedText(nil, "拼🙂", ui.TextInputRange{})
				if !e.MultiCompositionPending() {
					t.Fatal("fixture did not retain pending multi input")
				}
			}
			want := e.InputActionError()
			if want == nil {
				t.Fatal("fixture did not refuse input admission")
			}
			e.OnChange = func() { changes++ }
			e.OnError = func(err error) { reports++; reported = err }
			before, bufferBefore := sourceShortcutEditorState(e), captureSourceRejectState(b)
			for _, tc := range sourceShortcutCases {
				if !client.handleInput(sourceShortcutEvent(tc.mods, tc.key, false)) {
					t.Fatalf("refused %s leaked its key", tc.name)
				}
				client.handleInput(sourceShortcutEvent(tc.mods, tc.key, true))
			}
			if calls != 0 || changes != 0 || reports != 3 || reported != want {
				t.Fatalf("refusal callbacks: host=%d change=%d error=%d last=%v want exact %v", calls, changes, reports, reported, want)
			}
			if !reflect.DeepEqual(before, sourceShortcutEditorState(e)) || e.nativeClient != client || !client.current() || !tester.Focused(e.Label) {
				t.Fatal("refusal settled input, observed owners, moved focus or mutated editor state")
			}
			requireSourceRejectState(t, b, bufferBefore)
			if state == "foreign-group" {
				if !b.CancelGroup() {
					t.Fatal("shortcut stole foreign group")
				}
			} else if state == "foreign-composition" {
				if err := b.CancelCompositionGroup(foreign); err != nil {
					t.Fatal(err)
				}
			} else if !client.handleInput(sourceShortcutEvent(0, ui.KeyEscape, false)) || e.compositionBuffer != nil || b.InGroup() {
				t.Fatal("retained native client's Escape recovery was lost")
			}
		})
	}
}

func TestEditorSourceShortcutTesterRefusalRepeatAndFutureTextIME(t *testing.T) {
	b := text.New("abc")
	e := New(b)
	calls, reports := 0, 0
	e.SetSourceActionHandler(func(SourceAction) { calls++ })
	e.OnError = func(error) { reports++ }
	tester := multiTester(t, e)
	if err := b.SetSelection(3, 3); err != nil {
		t.Fatal(err)
	}
	tester.Key(ui.Ctrl, ui.KeySpace)
	tester.Type(" ")
	if b.String() != "abc " || calls != 1 {
		t.Fatal("handled Ctrl+Space swallowed a subsequent independent space")
	}
	tester.Compose("拼", 1)
	client, token := e.nativeClient, e.compositionToken
	if !e.composing || !b.OwnsComposition(token) {
		t.Fatal("future IME did not begin after shortcut")
	}
	for _, tc := range sourceShortcutCases {
		tester.Key(tc.mods, tc.key)
		tester.KeyRepeat(tc.mods, tc.key)
	}
	if calls != 1 || reports != 3 || e.nativeClient != client || e.compositionToken != token || !b.OwnsComposition(token) {
		t.Fatal("registered shortcut refusal/repeat changed retained native input")
	}
	tester.Type("拼")
	if b.String() != "abc 拼" || e.composing || b.InGroup() || e.compositionBuffer != nil {
		t.Fatal("shortcut refusal swallowed or canceled the original owner's final text")
	}
	tester.Key(0, ui.KeyF2)
	if calls != 2 || reports != 3 {
		t.Fatal("new action did not recover after native final text")
	}
}

func TestEditorSourceShortcutRawCallbackExitState(t *testing.T) {
	for _, refusal := range []bool{false, true} {
		name := "host"
		if refusal {
			name = "OnError"
		}
		t.Run(name, func(t *testing.T) {
			e := New(text.New("source"))
			client := e.nativeTextClient()
			e.BindInputTerminal(func(InputDelivery) { t.Fatal("shortcut published an input terminal") })
			e.SetSourceActionHandler(func(SourceAction) {})
			if refusal {
				if err := e.Buffer.BeginGroup(); err != nil {
					t.Fatal(err)
				}
			}
			var exit []any
			calls := 0
			callback := func() {
				calls++
				e.SetBuffer(text.New("replacement"))
				e.SetReadOnly(true)
				e.ReleaseView()
				e.ScrollX, e.ScrollY = 71, 83
				e.focused, e.dragging, e.hasDesiredX = true, true, true
				e.blinkStart = time.Unix(123, 0)
				e.SetSourceActionHandler(func(SourceAction) { t.Fatal("replacement handler called by old action") })
				exit = sourceShortcutEditorState(e)
			}
			if refusal {
				e.OnError = func(error) { callback() }
			} else {
				e.SetSourceActionHandler(func(SourceAction) { callback() })
			}
			if !client.handleInput(sourceShortcutEvent(ui.Ctrl, ui.KeySpace, false)) || calls != 1 {
				t.Fatal("installed client did not invoke exactly one final callback")
			}
			if !reflect.DeepEqual(exit, sourceShortcutEditorState(e)) {
				t.Fatal("old raw action wrote after callback exit")
			}
		})
	}
}

func TestEditorSourceShortcutElementCallbackExitState(t *testing.T) {
	for _, refusal := range []bool{false, true} {
		name := "host"
		if refusal {
			name = "OnError"
		}
		t.Run(name, func(t *testing.T) {
			e := New(text.New("source"))
			var exit []any
			calls, returned := 0, 0
			callback := func() {
				calls++
				e.SetSourceActionHandler(nil)
				e.focused, e.dragging, e.hasDesiredX = false, true, true
				e.ScrollX, e.ScrollY = 31, 47
				e.blinkStart = time.Unix(456, 0)
				exit = sourceShortcutEditorState(e)
			}
			e.SetSourceActionHandler(func(SourceAction) { callback() })
			e.OnError = func(error) { callback() }
			tester := ui.NewTester(func(c *ui.Context) {
				if calls != 0 {
					return // Do not conflate a successor frame with the old View.
				}
				e.View(c)
				if calls != 0 {
					returned++
					if !reflect.DeepEqual(exit, sourceShortcutEditorState(e)) {
						t.Fatal("View wrote editor state after its final callback")
					}
				}
			}, 600, 280)
			if err := tester.Click(e.Label); err != nil {
				t.Fatal(err)
			}
			if refusal {
				if err := e.Buffer.BeginGroup(); err != nil {
					t.Fatal(err)
				}
			}
			tester.Key(ui.Ctrl, ui.KeySpace)
			if calls != 1 || returned != 1 {
				t.Fatalf("registered element callback/return counts = %d/%d, want 1/1", calls, returned)
			}
		})
	}
}
