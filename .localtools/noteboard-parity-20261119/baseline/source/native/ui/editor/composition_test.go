package editor

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func TestCoalescingTesterUpdatesPreserveCallbacksRevisionAndUndo(t *testing.T) {
	for _, finish := range []string{"commit", "cancel"} {
		t.Run(finish, func(t *testing.T) {
			b := text.NewWithLimit("L原🙂R", 32)
			e := New(b)
			calls, globalEscapes := 0, 0
			e.OnChange = func() { calls++ }
			tester := ui.NewTester(func(c *ui.Context) {
				if c.Shortcut(0, ui.KeyEscape) {
					globalEscapes++
				}
				e.View(c)
			}, 480, 240)
			if err := tester.Click("文档编辑区"); err != nil {
				t.Fatal(err)
			}
			e.Reveal(4, 1)
			selection := b.Selection()
			current := "原🙂"
			values := []string{"原🙂", "a", "中🙂", "中🙂", "abc", "🙂"}
			for i := 0; i < 240; i++ {
				next := values[i%len(values)]
				version := b.Version()
				tester.Compose(next, len([]rune(next)))
				if next != current {
					version++
				}
				mark, active := e.MarkedRange()
				if calls != i+1 || b.Version() != version || b.String() != "L"+next+"R" || !active || mark != (ui.TextInputRange{Start: 1, End: 1 + text.UTF16Len(next)}) || !b.InGroup() || !b.OwnsComposition(e.compositionToken) || b.History().UndoDepth != 0 {
					t.Fatal("accepted update changed callback/version/group contract", i, calls, b.Version(), version, b.String())
				}
				current = next
			}
			version := b.Version()
			token := e.compositionToken
			if finish == "commit" {
				tester.Type(current) // same-text final commit still closes and calls changed
				if calls != 241 || b.Version() != version || b.InGroup() || b.History().UndoDepth != 1 || b.History().RetainedBytes != len("原🙂")+len(current) || b.OwnsComposition(token) {
					t.Fatal("final commit was not coalesced", calls, b.History())
				}
				if !b.Undo() || b.String() != "L原🙂R" || b.Selection() != selection || !b.Redo() || b.String() != "L"+current+"R" {
					t.Fatal("final one-step undo/redo")
				}
			} else {
				tester.Key(0, ui.KeyEscape)
				if calls != 241 || b.Version() != version+1 || b.InGroup() || b.String() != "L原🙂R" || b.Selection() != selection || b.CanUndo() || globalEscapes != 0 {
					t.Fatal("Escape replayed intermediates or missed exact owner")
				}
			}
			if e.compositionBuffer != nil || e.compositionToken != (text.CompositionToken{}) {
				t.Fatal("settled editor retained owner/token")
			}
			tester.Key(0, ui.KeyEscape)
			if globalEscapes != 1 {
				t.Fatal("ordinary Escape did not return to global host")
			}
		})
	}
}

func TestCoalescingTesterSameTextAndNetZeroPreserveRedo(t *testing.T) {
	b := text.New("L原🙂R")
	_ = b.ReplaceUTF16(5, 5, "!")
	if !b.Undo() {
		t.Fatal("redo fixture")
	}
	e := New(b)
	calls := 0
	e.OnChange = func() { calls++ }
	tester := ui.NewTester(e.View, 480, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	e.Reveal(4, 1)
	selection, history, version := b.Selection(), b.History(), b.Version()
	tester.Compose("原🙂", 2)
	if calls != 1 || b.Version() != version || !b.InGroup() {
		t.Fatal("same-text first preedit callback or ownership lost")
	}
	tester.Compose("中", 1)
	tester.Compose("原🙂", 2)
	tester.Type("原🙂")
	if calls != 4 || b.Version() != version+2 || b.InGroup() || b.History() != history || !b.CanRedo() {
		t.Fatal("net-zero commit invented history or suppressed callbacks", calls, b.History())
	}
	if !b.Redo() || b.String() != "L原🙂R!" || !b.Undo() {
		t.Fatal("prior redo broken")
	}
	e.Reveal(selection.Anchor, selection.Head)
	version = b.Version()
	tester.Compose("原🙂", 2)
	tester.Key(0, ui.KeyEscape)
	if calls != 6 || b.Version() != version || b.Selection() != selection || b.InGroup() || !b.CanRedo() {
		t.Fatal("same-text Escape lost original selection or callback")
	}
}

func TestCoalescingEditorExplicitRangeFallbackAndFinalCommit(t *testing.T) {
	b := text.New("abc🙂")
	e := New(b)
	_ = b.SetSelection(2, 1)
	selection := b.Selection()
	calls := 0
	e.OnChange = func() { calls++ }
	e.SetMarkedText(nil, "xy", ui.TextInputRange{Start: 2, End: 2})
	e.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "甲", ui.TextInputRange{Start: 1, End: 1})
	e.SetMarkedText(nil, "乙🙂", ui.TextInputRange{Start: 3, End: 3})
	e.ReplaceText(nil, "尾")
	if b.String() != "尾xyc🙂" || calls != 4 || b.InGroup() || b.History().UndoDepth != 1 || b.History().RetainedBytes != len("b")+len("xy")+len("a")+len("尾") {
		t.Fatal("irregular transition/final suffix not retained correctly", b.String(), calls, b.History())
	}
	if !b.Undo() || b.String() != "abc🙂" || b.Selection() != selection || !b.Redo() || b.String() != "尾xyc🙂" {
		t.Fatal("irregular native range history")
	}
}

func TestCoalescingEditorStaleTokenEscapeDoesNotCancelReplacementGroup(t *testing.T) {
	for _, reset := range []string{"restore-ordinary", "reset-ordinary", "restore-composition"} {
		t.Run(reset, func(t *testing.T) {
			b := text.New("a")
			e := New(b)
			var reported error
			e.OnError = func(err error) { reported = err }
			calls, globalEscapes := 0, 0
			e.OnChange = func() { calls++ }
			tester := ui.NewTester(func(c *ui.Context) {
				if c.Shortcut(0, ui.KeyEscape) {
					globalEscapes++
				}
				e.View(c)
			}, 480, 240)
			if err := tester.Click("文档编辑区"); err != nil {
				t.Fatal(err)
			}
			e.Reveal(1, 1)
			tester.Compose("中", 1)
			old := e.compositionToken
			if reset == "reset-ordinary" {
				b.ResetHistory()
			} else {
				b.Restore(text.New("restored").Snapshot())
			}
			var foreign text.CompositionToken
			if reset == "restore-composition" {
				var err error
				foreign, err = b.BeginCompositionGroup()
				if err != nil {
					t.Fatal(err)
				}
				if err = b.ReplaceCompositionUTF16(foreign, 0, 0, "!"); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := b.BeginGroup(); err != nil {
					t.Fatal(err)
				}
				if err := b.ReplaceUTF16(0, 0, "!"); err != nil {
					t.Fatal(err)
				}
			}
			version, selection, history, snapshot := b.Version(), b.Selection(), b.History(), b.Snapshot()
			if b.OwnsComposition(old) || !errors.Is(e.Flush(), text.ErrComposition) {
				t.Fatal("stale ownership passed Flush")
			}
			e.UnmarkText()
			tester.Compose("new", 3)
			tester.Type("new")
			if !errors.Is(reported, text.ErrComposition) || b.Version() != version || b.Selection() != selection || b.History() != history || !b.EqualSnapshot(snapshot) || !b.InGroup() {
				t.Fatal("stale update touched replacement group")
			}
			tester.Key(0, ui.KeyEscape)
			if e.composing || e.compositionBuffer != nil || e.compositionToken != (text.CompositionToken{}) || e.Flush() != nil || calls != 1 || globalEscapes != 0 || !b.InGroup() || b.Version() != version || b.Selection() != selection || b.History() != history || !b.EqualSnapshot(snapshot) {
				t.Fatal("stale Escape cancelled foreign group or kept old owner")
			}
			if reset == "restore-composition" {
				if !b.OwnsComposition(foreign) || b.CancelCompositionGroup(foreign) != nil {
					t.Fatal("replacement typed owner lost")
				}
			} else if !b.CancelGroup() {
				t.Fatal("ordinary replacement owner lost")
			}
		})
	}
}

func TestCoalescingFirstRefusalDoesNotAdoptLaterForeignGroup(t *testing.T) {
	b := text.NewWithLimit("a", 2)
	e := New(b)
	_ = b.SetSelection(1, 1)
	e.SetMarkedText(nil, "中", ui.TextInputRange{})
	if !errors.Is(e.Flush(), text.ErrSizeLimit) || b.InGroup() {
		t.Fatal("first refusal fixture")
	}
	if err := b.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	e.ReplaceText(nil, "x")
	e.SetMarkedText(nil, "x", ui.TextInputRange{})
	if b.String() != "a" || !b.InGroup() || !errors.Is(e.Flush(), text.ErrSizeLimit) {
		t.Fatal("retry adopted foreign ordinary group")
	}
	e.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape})
	if !b.InGroup() || e.Flush() != nil || e.compositionBuffer != nil || !b.CancelGroup() {
		t.Fatal("first-refusal Escape cancelled foreign group")
	}
}
