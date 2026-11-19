package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

const sourcePasteIdentityText = "abc\r\ndef\r\n"

func newSourcePasteIdentityTester(t *testing.T) (*App, *Tab, *ui.Tester) {
	t.Helper()
	a := newTestApp(t, t.TempDir())
	a.NewDocument("markdown")
	tab := a.Current()
	if tab == nil || tab.Adapter == nil {
		t.Fatal("built-in Markdown document was not created")
	}
	if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), sourcePasteIdentityText); err != nil {
		t.Fatal(err)
	}
	tab.Doc.Buffer.ResetHistory()
	tester := ui.NewTester(a.View, 1180, 780)
	for _, label := range []string{"源码", "文档编辑区"} {
		if err := tester.Click(label); err != nil {
			t.Fatal(err)
		}
		tester.Frame()
	}
	tester.Key(ui.Cmd, ui.KeyHome)
	tester.Key(0, ui.KeyRight)
	if tab.Mode != "source" || tab.Editor.Buffer != tab.Doc.Buffer || tab.Doc.Buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) {
		t.Fatal("source editor did not own literal selection [1,1)")
	}
	return a, tab, tester
}

func TestBuiltinSourcePasteIdentitySelectionChangeKeepsHostNavigation(t *testing.T) {
	a, tab, tester := newSourcePasteIdentityTester(t)
	buffer := tab.Doc.Buffer
	version, history := buffer.Version(), buffer.History()
	calls := 0
	var hostVersion uint64
	var hostHistory text.HistoryStatus
	var hostSelection text.Range
	tab.Editor.Paste = func() (string, error) {
		calls++
		tab.Editor.Reveal(6, 6)
		hostVersion, hostHistory, hostSelection = buffer.Version(), buffer.History(), buffer.Selection()
		return "P", nil
	}
	tester.Command("paste")
	if calls != 1 || hostSelection != (text.Range{Anchor: 6, Head: 6}) || hostVersion != version || hostHistory != history {
		t.Fatal("public Paste hook did not perform selection-only host navigation")
	}
	if buffer.String() != "abc\r\ndef\r\n" {
		t.Fatalf("source = %q, want literal unchanged source %q", buffer.String(), "abc\r\ndef\r\n")
	}
	if tab.Editor.Buffer != buffer || buffer.Selection() != hostSelection || buffer.Version() != hostVersion || buffer.History() != hostHistory {
		t.Fatal("paste changed or rolled back the host's navigation state")
	}
	if a.Message != "剪贴板已读取；文档或选区已改变，未粘贴内容" || len(a.Issues) != 1 || a.Issues[0] != a.Message {
		t.Fatal("stale paste did not report its visible failure", a.Message, a.Issues)
	}
}

func TestBuiltinSourcePasteIdentityBufferChangeKeepsBothDocuments(t *testing.T) {
	a, tab, tester := newSourcePasteIdentityTester(t)
	original := tab.Doc.Buffer
	other := text.New("")
	if err := other.ReplaceUTF16(0, 0, "abc\r\ndef\r\n"); err != nil {
		t.Fatal(err)
	}
	other.ResetHistory()
	if err := other.SetSelection(1, 1); err != nil {
		t.Fatal(err)
	}
	version, history := original.Version(), original.History()
	if other.Version() != version || other.Selection() != original.Selection() {
		t.Fatal("buffer replacement fixture must retain the same version and selection")
	}
	calls := 0
	tab.Editor.Paste = func() (string, error) {
		calls++
		tab.Doc.Buffer, tab.Editor.Buffer = other, other
		return "P", nil
	}
	tester.Command("paste")
	if calls != 1 || tab.Editor.Buffer != other || tab.Doc.Buffer != other {
		t.Fatal("paste rolled back the public host buffer replacement")
	}
	for _, buffer := range []*text.Buffer{original, other} {
		if buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) || buffer.Version() != version || buffer.History() != history {
			t.Fatal("paste modified either document despite stale buffer identity")
		}
	}
	if a.Message != "剪贴板已读取；文档或选区已改变，未粘贴内容" || len(a.Issues) != 1 {
		t.Fatal("buffer identity refusal was silent", a.Message, a.Issues)
	}
}

func TestBuiltinSourcePasteIdentityVersionChangeKeepsHostEditAndHistory(t *testing.T) {
	a, tab, tester := newSourcePasteIdentityTester(t)
	buffer := tab.Doc.Buffer
	version := buffer.Version()
	var hostVersion uint64
	var hostHistory text.HistoryStatus
	calls := 0
	tab.Editor.Paste = func() (string, error) {
		calls++
		tab.Editor.ReplaceText(nil, "H")
		tab.Editor.Reveal(1, 1)
		hostVersion, hostHistory = buffer.Version(), buffer.History()
		return "P", nil
	}
	tester.Command("paste")
	if calls != 1 || hostVersion == version || hostHistory.UndoDepth != 1 || hostHistory.RedoDepth != 0 {
		t.Fatal("host edit fixture did not change only document version at the captured selection")
	}
	if buffer.String() != "aHbc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) || buffer.Version() != hostVersion || buffer.History() != hostHistory {
		t.Fatal("stale paste overwrote, added to or rolled back the host edit", buffer.String(), buffer.Selection(), buffer.History())
	}
	if a.Message != "剪贴板已读取；文档或选区已改变，未粘贴内容" || len(a.Issues) != 1 {
		t.Fatal("version identity refusal was silent", a.Message)
	}
	tester.Command("undo")
	if buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) || buffer.History().RedoDepth != 1 {
		t.Fatal("undo did not restore the literal source before the host edit")
	}
	tester.Command("redo")
	if buffer.String() != "aHbc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 2, Head: 2}) || buffer.History().UndoDepth != 1 {
		t.Fatal("redo did not recover exactly the host edit")
	}
}

func TestBuiltinSourcePasteIdentityReleasedViewKeepsSourceUntilRecovery(t *testing.T) {
	a, tab, tester := newSourcePasteIdentityTester(t)
	buffer := tab.Doc.Buffer
	version, history := buffer.Version(), buffer.History()
	calls := 0
	tab.Editor.Paste = func() (string, error) {
		calls++
		tab.Editor.ReleaseView()
		return "P", nil
	}
	tester.Command("paste")
	if calls != 1 || tab.Editor.Buffer != buffer || buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) || buffer.Version() != version || buffer.History() != history {
		t.Fatal("view release did not preserve unchanged source identity")
	}
	if a.Message != "剪贴板已读取；文档或选区已改变，未粘贴内容" || len(a.Issues) != 1 {
		t.Fatal("released-view refusal was silent", a.Message)
	}
	tab.Editor.Paste = func() (string, error) { return "P", nil }
	tab.Editor.Reveal(1, 1)
	tester.Frame()
	tester.Command("paste")
	if buffer.String() != "aPbc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 2, Head: 2}) || buffer.History().UndoDepth != 1 || len(a.Issues) != 1 {
		t.Fatal("new live view did not recover ordinary paste", buffer.String(), buffer.Selection())
	}
}

func TestBuiltinSourcePasteIdentityReadErrorPrecedesStaleSelection(t *testing.T) {
	a, tab, tester := newSourcePasteIdentityTester(t)
	buffer := tab.Doc.Buffer
	version, history := buffer.Version(), buffer.History()
	calls := 0
	tab.Editor.Paste = func() (string, error) {
		calls++
		tab.Editor.Reveal(6, 6)
		return "P", text.ErrRange
	}
	tester.Command("paste")
	if calls != 1 || buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 6, Head: 6}) || buffer.Version() != version || buffer.History() != history {
		t.Fatal("clipboard error changed source or rolled back host navigation")
	}
	if a.Message != text.ErrRange.Error() || len(a.Issues) != 1 || a.Issues[0] != text.ErrRange.Error() {
		t.Fatal("stale identity masked the original clipboard error", a.Message, a.Issues)
	}
}

func TestBuiltinSourcePasteIdentityOrdinaryCommandsPreserveUnicodeCRLFAndUndo(t *testing.T) {
	for _, command := range []string{"paste", "pasteAndMatchStyle"} {
		for _, clipboard := range []string{"host", "services"} {
			t.Run(command+"/"+clipboard, func(t *testing.T) {
				a, tab, tester := newSourcePasteIdentityTester(t)
				buffer := tab.Doc.Buffer
				tab.Editor.Reveal(1, 3)
				version := buffer.Version()
				calls := 0
				if clipboard == "host" {
					tab.Editor.Paste = func() (string, error) {
						calls++
						return "中\r\n🙂", nil
					}
				} else {
					tab.Editor.Paste = nil
					tester.SetClipboard("中\r\n🙂")
				}
				tester.Command(command)
				if (clipboard == "host" && calls != 1) || buffer.String() != "a中\r\n🙂\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 6, Head: 6}) || buffer.Version() == version || buffer.History().UndoDepth != 1 || buffer.History().RedoDepth != 0 || len(a.Issues) != 0 {
					t.Fatal("ordinary paste lost Unicode, CRLF, selection or history", buffer.String(), buffer.Selection(), a.Issues)
				}
				tester.Command("undo")
				if buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 3}) || buffer.History().UndoDepth != 0 || buffer.History().RedoDepth != 1 {
					t.Fatal("ordinary paste undo did not restore source and selected text")
				}
				tester.Command("redo")
				if buffer.String() != "a中\r\n🙂\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 6, Head: 6}) || buffer.History().UndoDepth != 1 || buffer.History().RedoDepth != 0 || len(a.Issues) != 0 {
					t.Fatal("ordinary paste redo did not restore exact pasted bytes")
				}
			})
		}
	}
}

func TestBuiltinSourcePasteIdentityReadOnlyEntryCallbackAndRecovery(t *testing.T) {
	for _, command := range []string{"paste", "pasteAndMatchStyle"} {
		t.Run(command, func(t *testing.T) {
			a, tab, tester := newSourcePasteIdentityTester(t)
			buffer := tab.Doc.Buffer
			version, history := buffer.Version(), buffer.History()
			calls := 0
			tab.Editor.Paste = func() (string, error) {
				calls++
				tab.Doc.ReadOnly, tab.Editor.ReadOnly = true, true
				return "P", nil
			}
			tab.Doc.ReadOnly, tab.Editor.ReadOnly = true, true
			tester.Command(command)
			if calls != 0 || buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) || buffer.Version() != version || buffer.History() != history || len(a.Issues) != 0 {
				t.Fatal("initial readonly paste read the clipboard or mutated the source")
			}
			tab.Doc.ReadOnly, tab.Editor.ReadOnly = false, false
			tester.Command(command)
			if calls != 1 || !tab.Doc.ReadOnly || !tab.Editor.ReadOnly || buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) || buffer.Version() != version || buffer.History() != history {
				t.Fatal("paste mutated readonly callback-owned state")
			}
			if a.Message != "剪贴板已读取；文档或选区已改变，未粘贴内容" || len(a.Issues) != 1 {
				t.Fatal("readonly transition did not report the interrupted paste", a.Message)
			}
			tab.Doc.ReadOnly, tab.Editor.ReadOnly = false, false
			tab.Editor.Paste = func() (string, error) {
				calls++
				return "P", nil
			}
			tester.Command(command)
			if calls != 2 || buffer.String() != "aPbc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 2, Head: 2}) || buffer.History().UndoDepth != 1 || len(a.Issues) != 1 {
				t.Fatal("readonly recovery blocked ordinary paste", buffer.String(), buffer.Selection())
			}
		})
	}
}

func TestBuiltinSourcePasteIdentityEntryCompositionErrorKeepsPriorityUntilEscape(t *testing.T) {
	a, tab, tester := newSourcePasteIdentityTester(t)
	buffer := text.NewWithLimit("abc\r\ndef\r\n", 11)
	tab.Doc.Buffer, tab.Editor.Buffer = buffer, buffer
	tester.Frame()
	tester.Key(ui.Cmd, ui.KeyHome)
	tester.Key(0, ui.KeyRight)
	if buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) {
		t.Fatal("composition fixture did not own literal source offset one")
	}
	version, history := buffer.Version(), buffer.History()
	calls := 0
	tab.Editor.Paste = func() (string, error) {
		calls++
		return "P", nil
	}
	tester.Compose("中", 1)
	issue := tab.Editor.Flush()
	if issue == nil || buffer.InGroup() {
		t.Fatal("composition fixture did not preserve a rejected oversized input")
	}
	tester.Command("paste")
	if calls != 0 || tab.Editor.Flush() != issue || a.Message != issue.Error() || buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) || buffer.Version() != version || buffer.History() != history {
		t.Fatal("paste masked the pending composition error or read the clipboard", a.Message)
	}
	tester.Key(0, ui.KeyEscape)
	if tab.Editor.Flush() != nil || buffer.InGroup() {
		t.Fatal("Escape did not recover from the rejected composition")
	}
	tester.Command("paste")
	if calls != 1 || buffer.String() != "aPbc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 2, Head: 2}) || buffer.History().UndoDepth != 1 {
		t.Fatal("composition recovery did not accept a bounded ordinary paste", buffer.String(), buffer.Selection())
	}
}
