package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

const sourceOutlineNavigationText = "iiiiiiiiiiii\r\niiiiiiiiiiii\r\n# Target\r\niiiiiiiiiiii\r\n"

func newSourceOutlineNavigationTester(t *testing.T) (*Tab, *ui.Tester) {
	t.Helper()
	a := newTestApp(t, t.TempDir())
	a.NewDocument("markdown")
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.Outline == nil {
		t.Fatal("built-in Markdown outline is unavailable")
	}
	if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), sourceOutlineNavigationText); err != nil {
		t.Fatal(err)
	}
	tab.Doc.Buffer.ResetHistory()
	tester := ui.NewTester(a.View, 1180, 780)
	for _, label := range []string{"源码", "切换文件树与文档大纲", "文档编辑区"} {
		if err := tester.Click(label); err != nil {
			t.Fatal(err)
		}
		tester.Frame()
	}
	tester.Key(ui.Cmd, ui.KeyHome)
	for range 8 {
		tester.Key(0, ui.KeyRight)
	}
	tester.Key(0, ui.KeyDown)
	assertSourceOutlineSelection(t, tab, 22, 22)
	return tab, tester
}

func assertSourceOutlineSelection(t *testing.T, tab *Tab, anchor, head int) {
	t.Helper()
	if got := tab.Doc.Buffer.Selection(); got != (text.Range{Anchor: anchor, Head: head}) {
		t.Fatalf("selection = %+v, want literal [%d,%d)", got, anchor, head)
	}
}

func TestBuiltinSourceOutlineNavigationResetsVerticalTargetColumn(t *testing.T) {
	tab, tester := newSourceOutlineNavigationTester(t)
	before := tab.Doc.Buffer.History()
	if err := tester.Click("Target"); err != nil {
		t.Fatal(err)
	}
	tester.Key(0, ui.KeyEnter)
	assertSourceOutlineSelection(t, tab, 28, 28)
	// Outline relocation starts a new vertical run; an editor click here would mask the defect.
	tester.Key(0, ui.KeyDown)
	assertSourceOutlineSelection(t, tab, 38, 38)
	if got := tab.Doc.Buffer.String(); got != sourceOutlineNavigationText {
		t.Fatalf("navigation changed CRLF source: %q", got)
	}
	if got := tab.Doc.Buffer.History(); got != before {
		t.Fatalf("navigation changed history: %+v, want %+v", got, before)
	}
}

func TestBuiltinSourceOutlineNavigationKeepsColumnAcrossShortLine(t *testing.T) {
	tab, tester := newSourceOutlineNavigationTester(t)
	before := tab.Doc.Buffer.History()
	tester.Key(0, ui.KeyDown)
	assertSourceOutlineSelection(t, tab, 36, 36)
	tester.Key(0, ui.KeyDown)
	assertSourceOutlineSelection(t, tab, 46, 46)
	if tab.Doc.Buffer.String() != sourceOutlineNavigationText || tab.Doc.Buffer.History() != before {
		t.Fatal("ordinary vertical navigation changed source or history")
	}
}

func TestBuiltinSourceOutlineNavigationExtendsFromRevealedColumn(t *testing.T) {
	tab, tester := newSourceOutlineNavigationTester(t)
	before := tab.Doc.Buffer.History()
	if err := tester.Click("Target"); err != nil {
		t.Fatal(err)
	}
	tester.Key(0, ui.KeyEnter)
	assertSourceOutlineSelection(t, tab, 28, 28)
	tester.Key(ui.Shift, ui.KeyDown)
	assertSourceOutlineSelection(t, tab, 28, 38)
	if got, err := tab.Doc.Buffer.SliceUTF16(28, 38); err != nil || got != "# Target\r\n" {
		t.Fatalf("expanded selection = %q, %v, want literal heading and CRLF", got, err)
	}
	if tab.Doc.Buffer.String() != sourceOutlineNavigationText || tab.Doc.Buffer.History() != before {
		t.Fatal("extended outline navigation changed source or history")
	}
}

func TestBuiltinSourceOutlineNavigationFailedRevealKeepsTargetAndHistory(t *testing.T) {
	tab, tester := newSourceOutlineNavigationTester(t)
	tester.Type("x")
	tester.Command("undo")
	assertSourceOutlineSelection(t, tab, 22, 22)
	tester.Key(0, ui.KeyRight)
	tester.Key(0, ui.KeyRight)
	tester.Key(0, ui.KeyDown)
	assertSourceOutlineSelection(t, tab, 36, 36)
	before, version := tab.Doc.Buffer.History(), tab.Doc.Buffer.Version()
	if before.RedoDepth != 1 {
		t.Fatalf("redo fixture = %+v, want one retained edit", before)
	}
	var reported error
	tab.Editor.OnError = func(err error) { reported = err }
	tab.Editor.Reveal(-1, -1)
	if reported != text.ErrRange {
		t.Fatalf("invalid Reveal error = %v, want %v", reported, text.ErrRange)
	}
	assertSourceOutlineSelection(t, tab, 36, 36)
	if tab.Doc.Buffer.String() != sourceOutlineNavigationText || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.History() != before {
		t.Fatal("failed Reveal changed source, version or history")
	}
	// The short heading clipped column ten; a failed reveal must keep that target for the next line.
	tester.Key(0, ui.KeyDown)
	assertSourceOutlineSelection(t, tab, 48, 48)
	tester.Command("redo")
	const edited = "iiiiiiiiiiii\r\niiiiiiiixiiii\r\n# Target\r\niiiiiiiiiiii\r\n"
	if tab.Doc.Buffer.String() != edited {
		t.Fatalf("redo after failed Reveal = %q, want literal edit", tab.Doc.Buffer.String())
	}
	assertSourceOutlineSelection(t, tab, 23, 23)
	tester.Command("undo")
	assertSourceOutlineSelection(t, tab, 22, 22)
	if tab.Doc.Buffer.String() != sourceOutlineNavigationText {
		t.Fatal("undo after failed Reveal did not restore CRLF source")
	}
}

func TestBuiltinSourceOutlineNavigationRejectedCompositionKeepsTargetUntilEscape(t *testing.T) {
	tab, tester := newSourceOutlineNavigationTester(t)
	buffer := text.NewWithLimit(sourceOutlineNavigationText, 52)
	tab.Doc.Buffer, tab.Editor.Buffer = buffer, buffer
	tester.Frame()
	tester.Key(ui.Cmd, ui.KeyHome)
	for range 10 {
		tester.Key(0, ui.KeyRight)
	}
	tester.Key(0, ui.KeyDown)
	assertSourceOutlineSelection(t, tab, 24, 24)
	tester.Key(0, ui.KeyDown)
	assertSourceOutlineSelection(t, tab, 36, 36)
	before, version := buffer.History(), buffer.Version()
	var reported error
	tab.Editor.OnError = func(err error) { reported = err }
	tester.Compose("中", 1)
	issue := tab.Editor.Flush()
	if issue == nil || reported != issue || buffer.InGroup() {
		t.Fatalf("oversized composition fixture: issue=%v, reported=%v, grouped=%v", issue, reported, buffer.InGroup())
	}
	tab.Editor.Reveal(28, 28)
	if reported != issue || tab.Editor.Flush() != issue {
		t.Fatal("Reveal replaced or cleared the pending composition error")
	}
	assertSourceOutlineSelection(t, tab, 36, 36)
	if buffer.String() != sourceOutlineNavigationText || buffer.Version() != version || buffer.History() != before {
		t.Fatal("blocked Reveal changed source, version or history")
	}
	tester.Key(0, ui.KeyEscape)
	if tab.Editor.Flush() != nil || buffer.InGroup() {
		t.Fatal("Escape did not clear the rejected input")
	}
	tester.Key(0, ui.KeyDown)
	assertSourceOutlineSelection(t, tab, 48, 48)
	if buffer.String() != sourceOutlineNavigationText || buffer.Version() != version || buffer.History() != before {
		t.Fatal("recovered vertical navigation changed source, version or history")
	}
	tab.Editor.Reveal(28, 28)
	assertSourceOutlineSelection(t, tab, 28, 28)
	tester.Key(0, ui.KeyDown)
	assertSourceOutlineSelection(t, tab, 38, 38)
}
