package editor

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func TestNativeInputUndoAndSurrogateBoundaries(t *testing.T) {
	buffer := text.New("你好🙂\nsecond line")
	editor := New(buffer)
	tester := ui.NewTester(editor.View, 600, 300)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Key(ui.Cmd, ui.KeyEnd)
	tester.Type("世界")
	if buffer.String() != "你好🙂\nsecond line世界" {
		t.Fatalf("typed %q", buffer.String())
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != "你好🙂\nsecond line" {
		t.Fatalf("undo %q", buffer.String())
	}
	editor.Reveal(4, 4)
	tester.Frame()
	tester.Key(0, ui.KeyBackspace)
	if buffer.String() != "你好\nsecond line" {
		t.Fatalf("grapheme delete %q", buffer.String())
	}
}

func TestCompositionAndReadOnly(t *testing.T) {
	buffer := text.New("")
	editor := New(buffer)
	_ = ui.NewTester(editor.View, 600, 300)
	editor.SetMarkedText(nil, "ni", ui.TextInputRange{Start: 2, End: 2})
	editor.SetMarkedText(nil, "你", ui.TextInputRange{Start: 1, End: 1})
	editor.ReplaceText(nil, "你")
	if buffer.String() != "你" {
		t.Fatal(buffer.String())
	}
	if !buffer.Undo() || buffer.String() != "" {
		t.Fatal("IME must be one undo transaction")
	}
	editor.ReadOnly = true
	editor.ReplaceText(nil, "forbidden")
	if buffer.String() != "" {
		t.Fatal("read-only mutation")
	}
}

func TestIMEThroughNativeTransportDoesNotDoubleInsert(t *testing.T) {
	buffer := text.New("")
	editor := New(buffer)
	tester := ui.NewTester(editor.View, 400, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Compose("ni", 2)
	if _, composing := editor.MarkedRange(); !composing {
		t.Fatal("composition was not marked")
	}
	tester.Compose("你", 1)
	tester.Type("你")
	if _, composing := editor.MarkedRange(); composing {
		t.Fatal("composition remained active")
	}
	if buffer.String() != "你" {
		t.Fatalf("IME committed %q", buffer.String())
	}
	if buffer.History().UndoDepth != 1 {
		t.Fatalf("IME undo depth = %d", buffer.History().UndoDepth)
	}
}

func TestEnterUsesCRLFAndCurrentIndentation(t *testing.T) {
	buffer := text.New("    alpha\r\n    next")
	editor := New(buffer)
	tester := ui.NewTester(editor.View, 500, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	if err := editor.SetPosition(0, 9); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	tester.Key(0, ui.KeyEnter)
	want := "    alpha\r\n    \r\n    next"
	if buffer.String() != want {
		t.Fatalf("CRLF enter got %q", buffer.String())
	}
}

func TestSelectionUndoAndIndentCommands(t *testing.T) {
	buffer := text.New("alpha\nbeta\ngamma")
	editor := New(buffer)
	tester := ui.NewTester(editor.View, 500, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	if err := buffer.SetSelection(5, 15); err != nil {
		t.Fatal(err)
	}
	tester.Key(0, ui.KeyTab)
	if buffer.String() != "    alpha\n    beta\n    gamma" {
		t.Fatalf("indent got %q", buffer.String())
	}
	tester.Key(ui.Shift, ui.KeyTab)
	if buffer.String() != "alpha\nbeta\ngamma" {
		t.Fatalf("outdent got %q", buffer.String())
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != "alpha\nbeta\ngamma" {
		t.Fatalf("undo selection edits got %q", buffer.String())
	}
}

func TestReadOnlyClipboardAndEditing(t *testing.T) {
	buffer := text.New("retain me")
	editor := New(buffer)
	editor.ReadOnly = true
	tester := ui.NewTester(editor.View, 400, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Key(ui.Cmd, ui.KeyA)
	tester.Command("copy")
	if tester.Clipboard() != "retain me" {
		t.Fatalf("clipboard = %q", tester.Clipboard())
	}
	tester.Command("cut")
	tester.Key(0, ui.KeyEnter)
	tester.Key(0, ui.KeyBackspace)
	if buffer.String() != "retain me" {
		t.Fatalf("read-only buffer changed to %q", buffer.String())
	}
}

func TestClipboardFailureDoesNotCut(t *testing.T) {
	buffer := text.New("retain me")
	editor := New(buffer)
	editor.Copy = func(string) error { return errors.New("clipboard locked") }
	_ = buffer.SetSelection(0, buffer.LenUTF16())
	editor.Command("cut")
	if buffer.String() != "retain me" {
		t.Fatal("failed clipboard write deleted content")
	}
}

func TestPasteFailureIsReported(t *testing.T) {
	buffer := text.New("")
	editor := New(buffer)
	editor.Paste = func() (string, error) { return "", errors.New("clipboard read failed") }
	var reported error
	editor.OnError = func(err error) { reported = err }
	editor.Command("paste")
	if reported == nil {
		t.Fatal("paste failure was silently ignored")
	}
	if buffer.String() != "" {
		t.Fatalf("failed paste inserted %q", buffer.String())
	}
}

func TestReleaseViewRetainsUndo(t *testing.T) {
	buffer := text.New("a")
	editor := New(buffer)
	editor.ReplaceText(&ui.TextInputRange{Start: 1, End: 1}, "b")
	editor.ReleaseView()
	if !buffer.Undo() || buffer.String() != "a" {
		t.Fatal("history lost when view released")
	}
}

func TestOnChangeKeepsDirtySnapshot(t *testing.T) {
	buffer := text.New("before")
	editor := New(buffer)
	before := buffer.Snapshot()
	editor.ReplaceText(&ui.TextInputRange{Start: 6, End: 6}, " after")
	if buffer.EqualSnapshot(before) {
		t.Fatal("edit left buffer equal to saved snapshot")
	}
}

func TestSoftWrapRowsAndIMEGeometry(t *testing.T) {
	buffer := text.New("the quick brown fox jumps over the lazy dog")
	editor := New(buffer)
	editor.SoftWrap = true
	tester := ui.NewTester(editor.View, 180, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if err := editor.SetPosition(0, buffer.LenUTF16()); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	_, column, visualLine := editor.GetPosition()
	if visualLine == 0 {
		t.Fatal("wrapped final caret was not assigned a visual row")
	}
	if _, ok := tester.TextCaret(); !ok {
		t.Fatal("text input geometry unavailable after soft wrap")
	}
	if err := editor.SetPosition(0, 10); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if _, column, visualLine = editor.GetPosition(); column != 10 || visualLine != editor.line(0).rowAt(10) {
		t.Fatalf("wrapped position = %d:%d", column, visualLine)
	}
	tester.Type("!")
	if !strings.HasPrefix(buffer.String(), "the quick !") {
		t.Fatalf("wrapped edit got %q", buffer.String())
	}
}

func TestHighlightHookUsesSpansWithoutPluginDependency(t *testing.T) {
	buffer := text.New("let value")
	editor := New(buffer)
	editor.Highlight = func(line string) []ui.Span {
		if strings.HasPrefix(line, "let ") {
			return []ui.Span{{Text: "let", Weight: 700}, {Text: " value"}}
		}
		return []ui.Span{{Text: line}}
	}
	tester := ui.NewTester(editor.View, 400, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Type("x")
	if buffer.String() != "let valuex" {
		t.Fatalf("highlighted edit got %q", buffer.String())
	}
}

func TestLongLineWarnsOnceAndStaysEditable(t *testing.T) {
	buffer := text.New("x" + strings.Repeat("y", 40_000))
	editor := New(buffer)
	editor.OnError = func(error) {}
	tester := ui.NewTester(editor.View, 400, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	editor.OnError = nil
	var warnings int
	editor.OnError = func(error) { warnings++ }
	tester.Frame()
	editor.SetPosition(0, 40_001)
	tester.Type("z")
	if warnings != 0 {
		t.Fatalf("long-line warning repeated %d times", warnings)
	}
	if buffer.String() != "x"+strings.Repeat("y", 40_000)+"z" {
		t.Fatal("long line edit was lost")
	}
}

func TestFocusAndCaretAreIdleWithoutInput(t *testing.T) {
	buffer := text.New("focus")
	editor := New(buffer)
	tester := ui.NewTester(editor.View, 400, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if tester.FramePending() {
		t.Fatal("editor requested animation frames while idle")
	}
	tester.Advance(time.Second)
	tester.Frame()
	if tester.FramePending() {
		t.Fatal("editor requested frames after idle clock advance")
	}
}
