package editor

import (
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func TestMultiPendingBlocksEditorActionsAndEscRestoresSet(t *testing.T) {
	const source = "a b\n" // the second selected range is a caret on another line
	b := text.New(source + strings.Repeat("x", maxShapedLine+100))
	e := New(b)
	globalEscapes := 0
	tester := ui.NewTester(func(c *ui.Context) {
		if c.Shortcut(0, ui.KeyEscape) {
			globalEscapes++
		}
		ui.Button(c, "其他控件")
		e.View(c)
	}, 580, 290)
	if err := tester.Click(e.Label); err != nil {
		t.Fatal(err)
	}
	start := setSelections(t, b, []text.Range{{Anchor: 3, Head: 2}, {Anchor: 5, Head: 5}}, 1)
	changes, reads, writes := 0, 0, 0
	var reported error
	e.OnChange = func() { changes++ }
	e.OnError = func(err error) { reported = err }
	e.Paste = func() (string, error) { reads++; return "paste", nil }
	e.Copy = func(string) error { writes++; return nil }
	tester.Compose("拼🙂", 2)
	mark, active := e.MarkedRange()
	token := e.compositionToken
	before := currentEditState(b)
	if !active || changes != 1 || !tester.HasText(text.ErrCompositionPending.Error()) {
		t.Fatal("pending status or primary preedit missing")
	}
	checkPending := func() {
		t.Helper()
		requireEditState(t, b, before)
		if got, active := e.MarkedRange(); !active || got != mark || !b.OwnsComposition(token) || changes != 1 || reads != 0 || writes != 0 || !errors.Is(e.Flush(), text.ErrCompositionPending) {
			t.Fatalf("pending state changed: mark %+v, callbacks %d, clipboard R/W %d/%d", got, changes, reads, writes)
		}
	}
	for _, cmd := range []string{"selectAll", "copy", "cut", "paste", "pasteAndMatchStyle", "delete", "indent", "outdent", "undo", "redo", "moveLinesUp", "moveLinesDown", "copyLinesUp", "copyLinesDown"} {
		reported = nil
		tester.Command(cmd)
		if !errors.Is(reported, text.ErrCompositionPending) {
			t.Fatalf("%s did not report pending: %v", cmd, reported)
		}
		checkPending()
	}
	for _, key := range []ui.Key{ui.KeyLeft, ui.KeyUp, ui.KeyEnd, ui.KeyPageDown, ui.KeyTab, ui.KeyBackspace, ui.KeyDelete, ui.KeyEnter} {
		tester.Key(0, key)
		checkPending()
	}
	e.indentSelected(1)
	e.insertNewlineWithIndent()
	e.delete(1)
	e.editLines(text.CopyLinesDown)
	e.selectTo(0, false)
	e.SetMarkedText(nil, "program", ui.TextInputRange{})
	if err := e.SetPosition(0, 0); !errors.Is(err, text.ErrCompositionPending) {
		t.Fatal(err)
	}
	if err := tester.Click("下一段"); err != nil {
		t.Fatal(err)
	}
	if !tester.Focused(e.Label) || !errors.Is(reported, text.ErrCompositionPending) {
		t.Fatal("segment operation changed focus or concealed pending")
	}
	checkPending()
	if err := tester.Click("其他控件"); err != nil {
		t.Fatal(err)
	}
	checkPending()
	e.Reveal(0, 0)
	e.Focus()
	tester.Frame()
	if tester.Focused(e.Label) || !errors.Is(reported, text.ErrCompositionPending) {
		t.Fatal("program reveal/focus moved focus before admission")
	}
	if err := tester.Click(e.Label); err != nil {
		t.Fatal(err)
	}
	checkPending()
	oldClient := e.nativeClient
	tester.Key(0, ui.KeyEscape)
	if b.InGroup() || b.String() != source+strings.Repeat("x", maxShapedLine+100) || changes != 2 || globalEscapes != 0 || b.CanUndo() || e.Flush() != nil {
		t.Fatalf("Escape did not cancel once on its original buffer, calls %d, global Escape %d", changes, globalEscapes)
	}
	requireSelections(t, b, start)
	settled := currentEditState(b)
	oldClient.ReplaceText(nil, "late")
	requireEditState(t, b, settled)
	if e.nativeClient == oldClient || !errors.Is(reported, text.ErrComposition) {
		t.Fatal("the cancelled native client accepted a late final")
	}
}

func TestMultiNativeAttachmentEpochs(t *testing.T) {
	for _, name := range []string{"foreign-editor", "buffer-swap", "buffer-ABA-setter", "buffer-ABA-observed", "buffer-ABA-flush", "readonly-ABA", "view-release", "selection-ABA", "same-bytes-restore"} {
		t.Run(name, func(t *testing.T) {
			b := text.New("a b c")
			e := New(b)
			tester := multiTester(t, e)
			start := setSelections(t, b, []text.Range{{Anchor: 1, Head: 0}, {Anchor: 2, Head: 3}, {Anchor: 5, Head: 5}}, 1)
			var reported error
			e.OnError = func(err error) { reported = err }
			oldClient := e.nativeClient
			tester.Frame()
			tester.Frame()
			if e.nativeClient != oldClient {
				t.Fatal("unmodified frames replaced the native client")
			}
			tester.Compose("拼🙂", 2)
			token := e.compositionToken
			mark, marked := e.MarkedRange()
			if !marked || !b.OwnsComposition(token) {
				t.Fatal("missing original owner")
			}
			foreign := text.New("replacement")
			var foreignToken text.CompositionToken
			var err error
			switch name {
			case "foreign-editor":
				other := New(b)
				var foreignErr error
				other.OnError = func(err error) { foreignErr = err }
				before := currentEditState(b)
				other.nativeTextClient().ReplaceText(nil, "foreign")
				if !errors.Is(foreignErr, text.ErrGroup) || !b.OwnsComposition(token) {
					t.Fatalf("foreign editor adopted the owner: %v", foreignErr)
				}
				requireEditState(t, b, before)
				tester.Key(0, ui.KeyEscape)
				requireSelections(t, b, start)
				return
			case "buffer-swap":
				foreignToken, err = foreign.BeginCompositionGroup()
				if err != nil {
					t.Fatal(err)
				}
				e.SetBuffer(foreign)
			case "buffer-ABA-setter":
				e.SetBuffer(foreign)
				e.SetBuffer(b)
			case "buffer-ABA-observed", "buffer-ABA-flush":
				e.Buffer = foreign
				if name == "buffer-ABA-observed" {
					tester.Frame()
				} else if err := e.Flush(); !errors.Is(err, text.ErrCompositionPending) {
					t.Fatal(err)
				}
				e.Buffer = b
			case "readonly-ABA":
				e.SetReadOnly(true)
				e.SetReadOnly(false)
			case "view-release":
				e.ReleaseView()
				if len(e.cache) != 0 || len(e.rowCounts) != 0 || len(e.rowTree) != 0 || e.focused {
					t.Fatal("view release retained geometry")
				}
				if got, active := e.MarkedRange(); !active || got != mark {
					t.Fatal("view release discarded local mark")
				}
			case "selection-ABA":
				primary := b.Selection()
				if err := b.SetSelection(0, 0); err != nil {
					t.Fatal(err)
				}
				if err := b.SetSelection(primary.Anchor, primary.Head); err != nil {
					t.Fatal(err)
				}
			case "same-bytes-restore":
				b.Restore(b.Snapshot())
				foreignToken, err = b.BeginCompositionGroup()
				if err != nil {
					t.Fatal(err)
				}
			}
			oldState, foreignState := currentEditState(b), currentEditState(foreign)
			tester.Frame()
			reported = nil
			oldClient.ReplaceText(nil, "stale")
			if !errors.Is(reported, text.ErrComposition) || !errors.Is(e.Flush(), text.ErrCompositionPending) {
				t.Fatalf("old client rejection = %v", reported)
			}
			reported = nil
			e.nativeClient.ReplaceText(nil, "new-client-final")
			if !errors.Is(reported, text.ErrComposition) {
				t.Fatalf("new client claimed a stale owner: %v", reported)
			}
			e.nativeClient.SetMarkedText(nil, "ignored", ui.TextInputRange{})
			requireEditState(t, b, oldState)
			requireEditState(t, foreign, foreignState)
			if name != "same-bytes-restore" && !b.OwnsComposition(token) {
				t.Fatal("stale clients settled the original token")
			}
			if err := tester.Click(e.Label); err != nil {
				t.Fatal(err)
			}
			tester.Key(0, ui.KeyEscape)
			if e.Flush() != nil || e.compositionBuffer != nil || e.compositionToken != (text.CompositionToken{}) {
				t.Fatal("explicit Escape did not retire the original owner")
			}
			if name == "same-bytes-restore" {
				requireEditState(t, b, oldState)
				if !b.OwnsComposition(foreignToken) {
					t.Fatal("Escape cancelled the replacement token")
				}
			} else {
				if b.String() != "a b c" || b.InGroup() || b.CanUndo() {
					t.Fatalf("original buffer was not restored: %q %+v", b.String(), b.History())
				}
				requireSelections(t, b, start)
			}
			requireEditState(t, foreign, foreignState)
			if name == "buffer-swap" && !foreign.OwnsComposition(foreignToken) {
				t.Fatal("Escape cancelled the new buffer's owner")
			}
		})
	}
}

func TestMultiNativeInputEventFallback(t *testing.T) {
	b := text.New("a a")
	e := New(b)
	setSelections(t, b, []text.Range{{Anchor: 0, Head: 1}, {Anchor: 3, Head: 2}}, 1)
	if !e.HandleInput(ui.InputEvent{Kind: ui.InputCompose, Text: "m", Caret: 1}) || b.String() != "a m" {
		t.Fatal("fallback did not create native primary preedit")
	}
	e.ReplaceText(nil, "P")
	if b.String() != "a m" || !b.InGroup() {
		t.Fatal("program ReplaceText settled the fallback owner")
	}
	if !e.HandleInput(ui.InputEvent{Kind: ui.InputText, Text: "Z"}) || b.String() != "Z Z" || b.InGroup() {
		t.Fatal("InputText did not confirm the whole group")
	}
}

// Count actual rasterized accent pixels in the mark's underline strip. The
// caret is unfocused and the selection collapsed, so neither can be the oracle.
func admissionUnderlinePixels(t *testing.T, e *Editor, tester *ui.Tester, mark ui.TextInputRange) int {
	t.Helper()
	box, ok := tester.Find(e.Label)
	if !ok { t.Fatal("missing editor geometry") }
	rect, actual, visible := e.BoundsForRange(mark)
	if !visible || actual != mark || rect.W < 5 { t.Fatalf("missing marked-span geometry: %+v %+v %t", rect, actual, visible) }
	image := tester.Image()
	pixels := 0
	for y := int(box.Y+rect.Y+rect.H)-2; y <= int(box.Y+rect.Y+rect.H)+1; y++ {
		for x := int(box.X+rect.X)+2; x < int(box.X+rect.X+rect.W)-2; x++ {
			pixel := image.RGBAAt(x,y)
			if int(pixel.B) > int(pixel.R)+40 && int(pixel.B) > int(pixel.G)+20 { pixels++ }
		}
	}
	return pixels
}

func TestMultiAdmissionDrawSuppressesStaleMarkWithoutSettling(t *testing.T) {
	for _, name := range []string{"buffer-swap", "buffer-ABA-setter", "buffer-ABA-observed", "readonly-ABA", "token-replaced", "revision-moved", "selection-ABA", "view-release"} {
		t.Run(name, func(t *testing.T) {
			b := text.New("a a")
			e := New(b)
			tester := ui.NewTester(func(c *ui.Context){ ui.Button(c,"other"); e.View(c) },600,280)
			if err := tester.Click(e.Label); err != nil { t.Fatal(err) }
			start := setSelections(t,b,[]text.Range{{Anchor: 1, Head: 0},{Anchor: 3,Head: 2}},1)
			changes := 0
			e.OnChange = func(){ changes++ }
			tester.Compose("m",0)
			mark, active := e.MarkedRange()
			token := e.compositionToken
			if !active || mark != (ui.TextInputRange{Start: 2,End: 3}) || changes != 1 { t.Fatal("missing mark fixture") }
			if err := tester.Click("other"); err != nil { t.Fatal(err) }
			validPixels := admissionUnderlinePixels(t,e,tester,mark)
			t.Logf("valid mark underline pixels=%d", validPixels)
			if e.focused || validPixels == 0 { t.Fatal("valid unfocused mark was not actually painted") }
			foreign := text.New("X Y")
			foreignToken, err := foreign.BeginCompositionGroup()
			if err != nil { t.Fatal(err) }
			var replacementToken text.CompositionToken
			switch name {
			case "buffer-swap":
				e.SetBuffer(foreign)
			case "buffer-ABA-setter":
				for range 3 { e.SetBuffer(foreign); e.SetBuffer(b) }
			case "buffer-ABA-observed":
				for range 3 { e.Buffer = foreign; tester.Frame(); e.Buffer = b; tester.Frame() }
			case "readonly-ABA":
				for range 3 { e.SetReadOnly(true); e.SetReadOnly(false) }
			case "token-replaced":
				b.Restore(b.Snapshot())
				replacementToken, err = b.BeginCompositionGroup()
				if err != nil { t.Fatal(err) }
			case "revision-moved":
				if err := b.ReplaceCompositionUTF16(token, mark.Start, mark.End, "n"); err != nil { t.Fatal(err) }
			case "selection-ABA":
				primary := b.Selection()
				for range 3 {
					if err := b.SetSelection(0,0); err != nil { t.Fatal(err) }
					if err := b.SetSelection(primary.Anchor,primary.Head); err != nil { t.Fatal(err) }
				}
			case "view-release":
				e.ReleaseView()
			}
			originalState, foreignState := currentEditState(b), currentEditState(foreign)
			var renderErrors int
			e.OnError = func(error){ renderErrors++ }
			for range 3 { tester.Frame() }
			if got := admissionUnderlinePixels(t,e,tester,mark); got != 0 { t.Fatalf("stale mark painted %d accent pixels in current buffer",got) }
			if renderErrors != 0 || changes != 1 || e.compositionToken != token || !e.compositionMulti || !e.compositionMarked || e.marked != mark {
				t.Fatal("draw reported an error or settled/discarded the local pending owner")
			}
			requireEditState(t,b,originalState)
			requireEditState(t,foreign,foreignState)
			if !foreign.OwnsComposition(foreignToken) { t.Fatal("draw ended another buffer's token") }
			if name == "token-replaced" {
				if !b.OwnsComposition(replacementToken) { t.Fatal("draw ended the replacement token") }
			} else if !b.OwnsComposition(token) { t.Fatal("draw ended the original token") }
			if !e.cancelComposition() { t.Fatal("lost explicit cancellation path") }
			if name == "token-replaced" {
				requireEditState(t,b,originalState)
				if !b.OwnsComposition(replacementToken) { t.Fatal("cancel ended the replacement token") }
			} else {
				if b.String() != "a a" || b.InGroup() || b.CanUndo() { t.Fatal("cancel failed to restore original owner") }
				requireSelections(t,b,start)
			}
			requireEditState(t,foreign,foreignState)
			if !foreign.OwnsComposition(foreignToken) { t.Fatal("cancel ended another buffer's token") }
		})
	}
}
