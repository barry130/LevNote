package editor

import (
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func TestNativeLineCommandsKeyboardSelectionUndoRedo(t *testing.T) {
	for _, tt := range []struct {
		name      string
		mods      ui.Modifiers
		key       ui.Key
		want      string
		selection text.Range
	}{
		{"move-up", ui.Alt, ui.KeyUp, "two\none\nthree", text.Range{Anchor: 2, Head: 1}},
		{"move-down", ui.Alt, ui.KeyDown, "one\nthree\ntwo", text.Range{Anchor: 12, Head: 11}},
		{"copy-up", ui.Alt | ui.Shift, ui.KeyUp, "one\ntwo\ntwo\nthree", text.Range{Anchor: 6, Head: 5}},
		{"copy-down", ui.Alt | ui.Shift, ui.KeyDown, "one\ntwo\ntwo\nthree", text.Range{Anchor: 10, Head: 9}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			b := text.New("one\ntwo\nthree")
			e := New(b)
			tester := ui.NewTester(e.View, 480, 240)
			if err := tester.Click("文档编辑区"); err != nil {
				t.Fatal(err)
			}
			before := text.Range{Anchor: 6, Head: 5}
			if err := b.SetSelection(before.Anchor, before.Head); err != nil {
				t.Fatal(err)
			}
			calls := 0
			e.OnChange = func() { calls++ }
			tester.Key(tt.mods, tt.key)
			if b.String() != tt.want || b.Selection() != tt.selection || b.History().UndoDepth != 1 || calls != 1 || b.InGroup() {
				t.Fatalf("keyboard result=%q selected=%+v history=%+v calls=%d", b.String(), b.Selection(), b.History(), calls)
			}
			tester.Key(ui.Cmd, ui.KeyZ)
			if b.String() != "one\ntwo\nthree" || b.Selection() != before || b.CanUndo() || calls != 2 {
				t.Fatal("one undo did not restore source and direction")
			}
			tester.Key(ui.Cmd|ui.Shift, ui.KeyZ)
			if b.String() != tt.want || b.Selection() != tt.selection || b.CanRedo() || calls != 3 {
				t.Fatal("one redo did not restore result")
			}
		})
	}
}

func TestNativeLineCommandsBlockEndpointAndLogicalWrapping(t *testing.T) {
	b := text.New("a\r\n甲\n🙂\r\nz")
	e := New(b)
	tester := ui.NewTester(e.View, 480, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	before := text.Range{Anchor: 9, Head: 3}
	_ = b.SetSelection(before.Anchor, before.Head)
	tester.Key(ui.Alt, ui.KeyUp)
	if b.String() != "甲\n🙂\r\na\r\nz" || b.Selection() != (text.Range{Anchor: 6, Head: 0}) {
		t.Fatal("next-line endpoint or mixed EOL changed", b.String(), b.Selection())
	}
	if !e.Command("undo") || b.String() != "a\r\n甲\n🙂\r\nz" || b.Selection() != before {
		t.Fatal("block undo")
	}

	long := strings.Repeat("abc🙂", 1100)
	b = text.New("first\n" + long + "\nlast")
	e = New(b)
	e.SoftWrap = true
	tester = ui.NewTester(e.View, 240, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	_ = b.SetSelection(11, 11)
	tester.Key(ui.Alt, ui.KeyUp)
	if b.String() != long+"\nfirst\nlast" || b.Selection() != (text.Range{Anchor: 5, Head: 5}) {
		t.Fatal("command operated on a visual/segmented row")
	}
}

func TestNativeLineCommandsRepeatedCopyAndEmptyEOF(t *testing.T) {
	b := text.New("a\r\n🙂")
	e := New(b)
	_ = ui.NewTester(e.View, 480, 240)
	_ = b.SetSelection(5, 3)
	if !e.Command("copyLinesDown") || !e.Command("copyLinesDown") {
		t.Fatal("copy command not recognized")
	}
	if b.String() != "a\r\n🙂\r\n🙂\r\n🙂" || b.Selection() != (text.Range{Anchor: 13, Head: 11}) || b.History().UndoDepth != 2 {
		t.Fatal("repeat copy did not follow new block", b.String(), b.Selection())
	}
	e.Command("undo")
	e.Command("undo")
	if b.String() != "a\r\n🙂" || b.Selection() != (text.Range{Anchor: 5, Head: 3}) {
		t.Fatal("repeat copy undo")
	}

	b = text.New("")
	e = New(b)
	_ = ui.NewTester(e.View, 480, 240)
	e.Command("copyLinesDown")
	if b.String() != "\n" || b.Selection() != (text.Range{Anchor: 1, Head: 1}) || b.History().UndoDepth != 1 {
		t.Fatal("empty logical line was not copied")
	}
}

func TestNativeLineCommandsReadonlyAndForeignGroup(t *testing.T) {
	for _, command := range []string{"moveLinesUp", "moveLinesDown", "copyLinesUp", "copyLinesDown"} {
		t.Run(command, func(t *testing.T) {
			b := text.New("a\nb\nc")
			e := New(b)
			_ = ui.NewTester(e.View, 480, 240)
			_ = b.SetSelection(2, 3)
			calls := 0
			var reported error
			e.OnChange = func() { calls++ }
			e.OnError = func(err error) { reported = err }
			e.ReadOnly = true
			version, history, selection := b.Version(), b.History(), b.Selection()
			if !e.Command(command) {
				t.Fatal("read-only command not consumed")
			}
			if b.Version() != version || b.History() != history || b.Selection() != selection || calls != 0 || reported != nil {
				t.Fatal("readonly command mutated state")
			}
			e.ReadOnly = false
			if err := b.BeginGroup(); err != nil {
				t.Fatal(err)
			}
			if !e.Command(command) || !errors.Is(reported, text.ErrGroup) || !b.InGroup() || b.Version() != version || b.History() != history || b.Selection() != selection || calls != 0 {
				t.Fatal("command stole foreign transaction", reported)
			}
			b.CancelGroup()
			token, err := b.BeginCompositionGroup()
			if err != nil {
				t.Fatal(err)
			}
			reported = nil
			e.Command(command)
			if !errors.Is(reported, text.ErrGroup) || !b.OwnsComposition(token) || b.Version() != version || b.Selection() != selection || calls != 0 {
				t.Fatal("command stole foreign composition", reported)
			}
			if err := b.CancelCompositionGroup(token); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNativeLineCommandsExactModifiersAndReadonlyKeyboard(t *testing.T) {
	for _, mods := range []ui.Modifiers{ui.Alt | ui.Cmd, ui.Alt | ui.Shift | ui.Cmd} {
		b := text.New("a\nb\nc")
		e := New(b)
		_ = ui.NewTester(e.View, 480, 240)
		_ = b.SetSelection(2, 2)
		e.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Mods: mods, Key: ui.KeyUp})
		if b.String() != "a\nb\nc" || b.History().UndoDepth != 0 {
			t.Fatal("extra modifier triggered line edit")
		}
	}
	b := text.New("a\nb\nc")
	e := New(b)
	e.ReadOnly = true
	tester := ui.NewTester(e.View, 480, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	_ = b.SetSelection(2, 2)
	for _, mods := range []ui.Modifiers{ui.Alt, ui.Alt | ui.Shift} {
		for _, key := range []ui.Key{ui.KeyUp, ui.KeyDown} {
			tester.Key(mods, key)
		}
	}
	if b.String() != "a\nb\nc" || b.Selection() != (text.Range{Anchor: 2, Head: 2}) || b.History().UndoDepth != 0 {
		t.Fatal("readonly keyboard changed source/selection")
	}
}

func TestNativeLineCommandsNoOpKeepsHistoryAndMarkedInput(t *testing.T) {
	b := text.New("a\na")
	e := New(b)
	_ = ui.NewTester(e.View, 480, 240)
	_ = b.ReplaceUTF16(0, 1, "b")
	b.Undo()
	_ = b.SetSelection(0, 1)
	calls := 0
	e.OnChange = func() { calls++ }
	version, history := b.Version(), b.History()
	e.Command("moveLinesUp")
	if b.Version() != version || b.History() != history || b.Selection() != (text.Range{Anchor: 0, Head: 1}) || calls != 0 {
		t.Fatal("boundary operation changed history")
	}
	e.Command("moveLinesDown")
	if b.String() != "a\na" || b.Version() != version || b.History() != history || b.Selection() != (text.Range{Anchor: 2, Head: 3}) || calls != 0 {
		t.Fatal("identical exchange added history or lost target selection")
	}

	b = text.New("a\nb")
	e = New(b)
	_ = ui.NewTester(e.View, 480, 240)
	_ = b.SetSelection(1, 1)
	e.SetMarkedText(nil, "你", ui.TextInputRange{Start: 1, End: 1})
	marked, active := e.MarkedRange()
	if !active {
		t.Fatal("setup composition")
	}
	token, version, history := e.compositionToken, b.Version(), b.History()
	selection := b.Selection()
	e.Command("moveLinesUp")
	if after, active := e.MarkedRange(); !active || after != marked || e.compositionToken != token || b.Version() != version || b.History() != history || b.Selection() != selection {
		t.Fatal("boundary no-op settled composition")
	}
}

func TestNativeLineCommandsCompositionHasSeparateUndo(t *testing.T) {
	for _, tt := range []struct{ command, want string }{
		{"moveLinesUp", "b你\na\nc"},
		{"moveLinesDown", "a\nc\nb你"},
		{"copyLinesUp", "a\nb你\nb你\nc"},
		{"copyLinesDown", "a\nb你\nb你\nc"},
	} {
		t.Run(tt.command, func(t *testing.T) {
			b := text.New("a\nb\nc")
			e := New(b)
			_ = ui.NewTester(e.View, 480, 240)
			_ = b.SetSelection(3, 3)
			e.SetMarkedText(nil, "你", ui.TextInputRange{Start: 1, End: 1})
			stagedSelection := b.Selection()
			calls := 0
			e.OnChange = func() { calls++ }
			e.Command(tt.command)
			finalSelection := b.Selection()
			if _, active := e.MarkedRange(); active || b.String() != tt.want || b.History().UndoDepth != 2 || calls != 1 || b.InGroup() {
				t.Fatal("line edit did not settle own composition exactly once", b.String(), b.History(), calls)
			}
			e.Command("undo")
			if b.String() != "a\nb你\nc" || b.Selection() != stagedSelection {
				t.Fatal("first undo should restore composed text")
			}
			e.Command("undo")
			if b.String() != "a\nb\nc" || b.Selection() != (text.Range{Anchor: 3, Head: 3}) {
				t.Fatal("second undo should remove composition")
			}
			e.Command("redo")
			e.Command("redo")
			if b.String() != tt.want || b.Selection() != finalSelection {
				t.Fatal("redo composition and line operation")
			}
		})
	}
}

func TestNativeLineCommandsCapacityAndRefusedCompositionAreAtomic(t *testing.T) {
	for _, command := range []string{"copyLinesUp", "copyLinesDown"} {
		t.Run(command, func(t *testing.T) {
			b := text.NewWithLimit("a\nb", 6)
			e := New(b)
			_ = ui.NewTester(e.View, 480, 240)
			_ = b.SetSelection(3, 3)
			e.SetMarkedText(nil, "你", ui.TextInputRange{Start: 1, End: 1})
			var reported error
			calls := 0
			e.OnError = func(err error) { reported = err }
			e.OnChange = func() { calls++ }
			version, history, selection, snapshot := b.Version(), b.History(), b.Selection(), b.Snapshot()
			marked, _ := e.MarkedRange()
			token := e.compositionToken
			e.Command(command)
			if after, active := e.MarkedRange(); !active || after != marked || e.compositionToken != token || !errors.Is(reported, text.ErrSizeLimit) || b.Version() != version || b.History() != history || b.Selection() != selection || !b.EqualSnapshot(snapshot) || calls != 0 {
				t.Fatal("capacity rejection changed accepted preedit", reported)
			}
			e.SetMarkedText(nil, "too long", ui.TextInputRange{Start: 8, End: 8})
			if e.compositionError == nil {
				t.Fatal("setup refused composition")
			}
			for _, next := range []string{"moveLinesUp", "moveLinesDown", "copyLinesUp", "copyLinesDown"} {
				e.Command(next)
			}
			if after, active := e.MarkedRange(); !active || after != marked || e.compositionToken != token || b.Version() != version || b.History() != history || b.Selection() != selection || !b.EqualSnapshot(snapshot) || calls != 0 {
				t.Fatal("commands crossed refused-input guard")
			}
			e.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape})
			if b.String() != "a\nb" || b.InGroup() {
				t.Fatal("cancel no longer restores source")
			}
		})
	}
}

func TestNativeLineCommandsCapacityPreservesRedo(t *testing.T) {
	b := text.NewWithLimit("a\r\n🙂", len("a\r\n🙂"))
	e := New(b)
	_ = ui.NewTester(e.View, 480, 240)
	_ = b.ReplaceUTF16(0, 1, "b")
	b.Undo()
	_ = b.SetSelection(5, 3)
	version, history, selection, snapshot := b.Version(), b.History(), b.Selection(), b.Snapshot()
	calls := 0
	var reported error
	e.OnChange = func() { calls++ }
	e.OnError = func(err error) { reported = err }
	e.Command("copyLinesDown")
	if !errors.Is(reported, text.ErrSizeLimit) || b.Version() != version || b.History() != history || b.Selection() != selection || !b.EqualSnapshot(snapshot) || calls != 0 || b.InGroup() {
		t.Fatal("rejected copy damaged redo/source")
	}
}
