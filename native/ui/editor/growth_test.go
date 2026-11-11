package editor

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func TestGrowthSourcePasteRefusalAndRetry(t *testing.T) {
	b := text.NewWithLimit("甲🙂", 8)
	e := New(b)
	tester := ui.NewTester(e.View, 480, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	e.Reveal(b.LenUTF16(), b.LenUTF16())
	var reported error
	changes, reads := 0, 0
	e.OnError = func(err error) { reported = err }
	e.OnChange = func() { changes++ }
	e.Paste = func() (string, error) { reads++; return "ab", nil }
	version, selection, history, snapshot := b.Version(), b.Selection(), b.History(), b.Snapshot()
	tester.Command("paste")
	if !errors.Is(reported, text.ErrSizeLimit) || reads != 1 || changes != 0 || b.Version() != version || b.Selection() != selection || b.History() != history || !b.EqualSnapshot(snapshot) {
		t.Fatal("paste refusal changed state or read twice", reported, reads)
	}
	if err := e.Flush(); err != nil {
		t.Fatal("ordinary paste refusal incorrectly blocked host", err)
	}
	e.Reveal(1, 3)
	tester.Command("pasteAndMatchStyle")
	if b.String() != "甲ab" || changes != 1 || !b.Undo() || b.String() != "甲🙂" {
		t.Fatal("replacing selection after refusal failed")
	}
}

func TestGrowthSourceCompositionRefusalOwnership(t *testing.T) {
	b := text.NewWithLimit("a", 4)
	e := New(b)
	tester := ui.NewTester(e.View, 480, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	e.Reveal(1, 1)
	var reported error
	e.OnError = func(err error) { reported = err }
	version, selection, history := b.Version(), b.Selection(), b.History()
	tester.Compose("🙂", 1)
	if _, composing := e.MarkedRange(); composing || b.InGroup() || b.String() != "a" || b.Version() != version || b.Selection() != selection || b.History() != history || !errors.Is(reported, text.ErrSizeLimit) {
		t.Fatal("first refused composition opened/changed state", reported)
	}
	if !errors.Is(e.Flush(), text.ErrSizeLimit) {
		t.Fatal("first refused preedit did not block Flush")
	}
	tester.Compose("ni", 2)
	mark, composing := e.MarkedRange()
	version, selection, history = b.Version(), b.Selection(), b.History()
	tester.Compose("🙂", 1)
	after, active := e.MarkedRange()
	if !composing || !active || mark != after || !b.InGroup() || b.String() != "ani" || b.Version() != version || b.Selection() != selection || b.History() != history {
		t.Fatal("ongoing refused composition lost accepted preedit")
	}
	e.UnmarkText()
	if !errors.Is(e.Flush(), text.ErrSizeLimit) || !b.InGroup() {
		t.Fatal("incidental unmark cleared ongoing refusal")
	}
	tester.Type("🙂")
	if after, active = e.MarkedRange(); !active || after != mark || b.String() != "ani" || !b.InGroup() {
		t.Fatal("refused final commit discarded preedit")
	}
	tester.Compose("你", 1)
	tester.Type("你")
	if b.String() != "a你" || b.InGroup() || b.History().UndoDepth != 1 || !b.Undo() || b.String() != "a" || !b.Redo() || b.String() != "a你" {
		t.Fatal("retry did not preserve one composition undo")
	}
	e.Reveal(1, 2)
	tester.Compose("b", 1)
	tester.Compose("huge", 4)
	tester.Key(0, ui.KeyEscape)
	if b.String() != "a你" || b.InGroup() {
		t.Fatal("cancel could not restore larger accepted preedit source")
	}
	if err := e.Flush(); err != nil {
		t.Fatal("explicit cancel did not clear refusal", err)
	}
	if err := b.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	e.SetMarkedText(nil, "x", ui.TextInputRange{})
	if !errors.Is(reported, text.ErrGroup) || !b.InGroup() {
		t.Fatal("foreign group was stolen", reported)
	}
	if err := e.Flush(); err != nil || !b.InGroup() || !b.CancelGroup() {
		t.Fatal("Flush closed foreign group", err)
	}
}

func TestGrowthSourceRefusalGuardsIncidentalNavigation(t *testing.T) {
	for _, started := range []bool{false, true} {
		t.Run(map[bool]string{false: "first", true: "ongoing"}[started], func(t *testing.T) {
			b := text.NewWithLimit("a", 4)
			e := New(b)
			globalEscapes := 0
			tester := ui.NewTester(func(c *ui.Context) {
				if c.Shortcut(0, ui.KeyEscape) {
					globalEscapes++
				}
				ui.Button(c, "其他控件")
				e.View(c)
			}, 480, 240)
			if err := tester.Click("文档编辑区"); err != nil {
				t.Fatal(err)
			}
			e.Reveal(1, 1)
			if started {
				tester.Compose("ni", 2)
			}
			tester.Compose("🙂", 1)
			mark, active := e.MarkedRange()
			version, selection, history, snapshot := b.Version(), b.Selection(), b.History(), b.Snapshot()
			check := func() {
				t.Helper()
				got, composing := e.MarkedRange()
				if got != mark || composing != active || b.InGroup() != started || b.Version() != version || b.Selection() != selection || b.History() != history || !b.EqualSnapshot(snapshot) || !errors.Is(e.Flush(), text.ErrSizeLimit) {
					t.Fatal("incidental operation changed refused composition")
				}
			}
			if err := tester.Click("其他控件"); err != nil {
				t.Fatal(err)
			}
			check()
			if err := tester.Click("文档编辑区"); err != nil {
				t.Fatal(err)
			}
			check()
			for _, key := range []ui.Key{ui.KeyLeft, ui.KeyHome, ui.KeyBackspace, ui.KeyEnter, ui.KeyTab} {
				tester.Key(0, key)
				check()
			}
			for _, command := range []string{"selectAll", "copy", "cut", "paste", "delete", "indent", "outdent", "undo", "redo"} {
				tester.Command(command)
				check()
			}
			if err := e.SetPosition(0, 0); !errors.Is(err, text.ErrSizeLimit) {
				t.Fatal(err)
			}
			e.Reveal(0, 0)
			e.ReleaseView()
			tester.Frame()
			check()
			if err := tester.Click("文档编辑区"); err != nil {
				t.Fatal(err)
			}
			tester.Key(0, ui.KeyEscape)
			if err := e.Flush(); err != nil || b.InGroup() || b.String() != "a" || globalEscapes != 0 {
				t.Fatal("Escape did not explicitly cancel", err)
			}
			if !started && (b.Version() != version || b.Selection() != selection || b.History() != history) {
				t.Fatal("first refusal cancellation edited source")
			}
			tester.Type("你")
			if b.String() != "a你" {
				t.Fatal("input did not resume after cancel", b.String())
			}
			tester.Key(0, ui.KeyEscape)
			if globalEscapes != 1 {
				t.Fatal("settled editor retained window Escape", globalEscapes)
			}
		})
	}
}

func TestGrowthSourceRefusalDoesNotFollowReplacedBuffer(t *testing.T) {
	old := text.NewWithLimit("a", 4)
	e := New(old)
	_ = old.SetSelection(1, 1)
	e.SetMarkedText(nil, "ni", ui.TextInputRange{Start: 2, End: 2})
	e.SetMarkedText(nil, "🙂", ui.TextInputRange{})
	replacement := text.New("replacement")
	if err := replacement.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	e.Buffer = replacement
	version, selection, history, snapshot := replacement.Version(), replacement.Selection(), replacement.History(), replacement.Snapshot()
	e.UnmarkText()
	e.ReleaseView()
	e.SetMarkedText(nil, "x", ui.TextInputRange{})
	e.ReplaceText(nil, "x")
	if !errors.Is(e.Flush(), text.ErrSizeLimit) || !old.InGroup() || !replacement.InGroup() || replacement.Version() != version || replacement.Selection() != selection || replacement.History() != history || !replacement.EqualSnapshot(snapshot) {
		t.Fatal("stale composition touched replacement buffer")
	}
	e.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape})
	if e.Flush() != nil || old.InGroup() || old.String() != "a" || !replacement.InGroup() || replacement.Version() != version || !replacement.EqualSnapshot(snapshot) {
		t.Fatal("cancel did not use original composition owner")
	}
	replacement.CancelGroup()
}

func TestGrowthIndentPreflightDirectionCRLFAndComposition(t *testing.T) {
	const source = "甲\r\n🙂\r\n"
	// The first line alone would fit; both selected lines do not.
	b := text.NewWithLimit(source, len(source)+7)
	e := New(b)
	_ = ui.NewTester(e.View, 480, 240)
	_ = b.SetSelection(b.LenUTF16(), 0)
	var reported error
	e.OnError = func(err error) { reported = err }
	version, selection, history, snapshot := b.Version(), b.Selection(), b.History(), b.Snapshot()
	e.Command("indent")
	if !errors.Is(reported, text.ErrSizeLimit) || b.Version() != version || b.Selection() != selection || b.History() != history || !b.EqualSnapshot(snapshot) || b.InGroup() {
		t.Fatal("indent partially edited before refusal", reported)
	}
	// Select one line exactly; reversed selection and CRLF survive undo/redo.
	_ = b.SetSelection(3, 0)
	selection = b.Selection()
	e.Command("indent")
	if b.String() != "    甲\r\n🙂\r\n" || b.Selection().Anchor <= b.Selection().Head || !b.Undo() || b.Selection() != selection || b.String() != source || !b.Redo() {
		t.Fatal("legal indent selection/history")
	}
	e.Command("outdent")
	if b.String() != source {
		t.Fatal("outdent blocked or normalized CRLF")
	}
	b = text.NewWithLimit("a", 3)
	e = New(b)
	e.OnError = func(err error) { reported = err }
	_ = b.SetSelection(1, 1)
	e.SetMarkedText(nil, "b", ui.TextInputRange{Start: 1, End: 1})
	marked, _ := e.MarkedRange()
	version = b.Version()
	e.Command("indent")
	if after, active := e.MarkedRange(); !active || after != marked || !b.InGroup() || b.Version() != version || b.String() != "ab" || !errors.Is(reported, text.ErrSizeLimit) {
		t.Fatal("refused indent committed active composition")
	}
}
