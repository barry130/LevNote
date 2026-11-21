package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

const sourcePasteIdentityText = "abc\r\ndef\r\n"
const sourcePasteIdentityStaleClipboardMessage = "文档、选区或输入状态已改变，已停止剪贴板操作"

func requireSourcePasteIdentityFocusedAttachment(t *testing.T, tester *ui.Tester, readOnly bool) {
	t.Helper()
	active, noInputMethod := tester.TextInputActive()
	if !tester.Focused("文档编辑区") || active == readOnly || noInputMethod {
		t.Fatal("source attachment did not install the expected focused input route", readOnly, active, noInputMethod)
	}
}

func newSourcePasteIdentityTester(t *testing.T) (*App, *Tab, *ui.Tester) {
	t.Helper()
	a := newTestApp(t, t.TempDir())
	// Clipboard identity tests exercise only issues and status from the new
	// input transaction. Start that history after unrelated startup reporting.
	a.Issues, a.IssuesDropped, a.Message = nil, 0, ""
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
	if a.Message != sourcePasteIdentityStaleClipboardMessage || len(a.Issues) != 1 || a.Issues[0] != a.Message {
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
	// Replacing the attachment retires the captured error handler as well as
	// the paste target. The old action has no authority to report on its tail.
	if a.Message != "" || len(a.Issues) != 0 {
		t.Fatal("retired buffer callback reported into the replacement owner", a.Message, a.Issues)
	}
	requireSourcePasteIdentityFocusedAttachment(t, tester, false)
	tab.Editor.Paste = func() (string, error) { calls++; return "P", nil }
	tester.Command("paste")
	if calls != 2 || original.String() != sourcePasteIdentityText || original.Selection() != (text.Range{Anchor: 1, Head: 1}) || original.Version() != version || original.History() != history ||
		other.String() != "aPbc\r\ndef\r\n" || other.Selection() != (text.Range{Anchor: 2, Head: 2}) || other.History().UndoDepth != 1 || other.History().RedoDepth != 0 || a.Message != "" || len(a.Issues) != 0 {
		t.Fatal("replacement attachment did not recover an isolated ordinary paste", calls, original.String(), other.String(), other.History())
	}
	tester.Command("undo")
	if other.String() != sourcePasteIdentityText || other.Selection() != (text.Range{Anchor: 1, Head: 1}) || other.History().UndoDepth != 0 || other.History().RedoDepth != 1 || original.Version() != version || original.History() != history {
		t.Fatal("replacement paste undo changed either document's original ownership")
	}
	tester.Command("redo")
	if other.String() != "aPbc\r\ndef\r\n" || other.Selection() != (text.Range{Anchor: 2, Head: 2}) || other.History().UndoDepth != 1 || other.History().RedoDepth != 0 || original.String() != sourcePasteIdentityText || original.Version() != version || original.History() != history {
		t.Fatal("replacement paste redo did not retain isolated history")
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
	if a.Message != sourcePasteIdentityStaleClipboardMessage || len(a.Issues) != 1 || a.Issues[0] != a.Message {
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
	requireSourcePasteIdentityFocusedAttachment(t, tester, false)
	captureNative := func() ui.TesterTextInputCapture {
		t.Helper()
		var capture ui.TesterTextInputCapture
		if err := tester.TextInputNoFrame(func(input *ui.TesterTextInput) { capture = input.Capture() }); err != nil {
			t.Fatal(err)
		}
		return capture
	}
	originalNative := captureNative()
	originalRoute := a.sourceLanguage.shortcut
	if !a.sourceLanguageShortcutCurrent(originalRoute) {
		t.Fatal("fixture did not establish the live App source route")
	}
	calls := 0
	tab.Editor.Paste = func() (string, error) {
		calls++
		if !tester.Focused("文档编辑区") || captureNative() != originalNative || !a.sourceLanguageShortcutCurrent(originalRoute) {
			t.Fatal("native paste did not start on the captured source attachment")
		}
		tab.Editor.ReleaseView()
		// ReleaseView invalidates the Editor attachment immediately. MyGo's
		// installed client and focused element ID still describe the last frame.
		if tab.Editor.SourceActionBindingCurrent(originalRoute.lease) || captureNative() != originalNative {
			t.Fatal("release did not retire the source seal before the native frame changed")
		}
		return "P", nil
	}
	tester.Command("paste")
	if calls != 1 || tab.Editor.Buffer != buffer || buffer.String() != sourcePasteIdentityText || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) || buffer.Version() != version || buffer.History() != history {
		t.Fatal("view release did not preserve unchanged source identity")
	}
	if a.Message != "" || len(a.Issues) != 0 || tab.Editor.SourceActionBindingCurrent(originalRoute.lease) {
		t.Fatal("released callback reported or regained its source seal", a.Message, a.Issues)
	}
	// Command settles frames and the App still shows the same source route.
	// If the surface kept focus, its focused native client must be the one
	// installed by the fresh App frame. A matching label proves no identity.
	if tester.Focused("文档编辑区") {
		route := a.sourceLanguage.shortcut
		if captureNative() == originalNative || route == originalRoute || !a.sourceLanguageShortcutCurrent(route) {
			t.Fatal("the settled frame focused a source route without a new native attachment")
		}
	}
	tab.Editor.Paste = func() (string, error) { calls++; return "P", nil }
	tab.Editor.Reveal(1, 1)
	tester.Frame()
	requireSourcePasteIdentityFocusedAttachment(t, tester, false)
	if route := a.sourceLanguage.shortcut; route == originalRoute || !a.sourceLanguageShortcutCurrent(route) {
		t.Fatal("explicit recovery did not establish a new live App source route")
	}
	if err := tester.TextInputNoFrame(func(input *ui.TesterTextInput) {
		if input.Capture() == originalNative || input.ResultFor(originalNative, nil, "stale") {
			t.Fatal("explicit recovery accepted a delivery from the retired native attachment")
		}
	}); err != nil {
		t.Fatal(err)
	}
	if buffer.String() != sourcePasteIdentityText || buffer.Version() != version || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) || buffer.History() != history || a.Message != "" || len(a.Issues) != 0 {
		t.Fatal("retired native delivery mutated or reported into the recovery attachment")
	}
	tester.Command("paste")
	if calls != 2 || buffer.String() != "aPbc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 2, Head: 2}) || buffer.History().UndoDepth != 1 || buffer.History().RedoDepth != 0 || a.Message != "" || len(a.Issues) != 0 {
		t.Fatal("new live view did not recover ordinary paste", buffer.String(), buffer.Selection())
	}
	tester.Command("undo")
	if buffer.String() != sourcePasteIdentityText || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) || buffer.History().UndoDepth != 0 || buffer.History().RedoDepth != 1 {
		t.Fatal("released-view recovery undo lost original source or selection")
	}
	tester.Command("redo")
	if buffer.String() != "aPbc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 2, Head: 2}) || buffer.History().UndoDepth != 1 || buffer.History().RedoDepth != 0 {
		t.Fatal("released-view recovery redo lost the sole accepted paste")
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
				tab.Doc.ReadOnly = true
				tab.Editor.SetReadOnly(true)
				return "P", nil
			}
			tab.Doc.ReadOnly = true
			tab.Editor.SetReadOnly(true)
			tester.Frame()
			requireSourcePasteIdentityFocusedAttachment(t, tester, true)
			tester.Command(command)
			if calls != 0 || buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) || buffer.Version() != version || buffer.History() != history || len(a.Issues) != 0 {
				t.Fatal("initial readonly paste read the clipboard or mutated the source")
			}
			tab.Doc.ReadOnly = false
			tab.Editor.SetReadOnly(false)
			// The readonly frame retired the writable native client. Render the
			// new attachment before asking its focused handler to paste.
			tester.Frame()
			requireSourcePasteIdentityFocusedAttachment(t, tester, false)
			tester.Command(command)
			if calls != 1 || !tab.Doc.ReadOnly || !tab.Editor.ReadOnly || buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) || buffer.Version() != version || buffer.History() != history {
				t.Fatal("paste mutated readonly callback-owned state")
			}
			// A final readonly attachment revokes the old callback's report
			// authority; the never-bound same-attachment ABA exception does not apply.
			if a.Message != "" || len(a.Issues) != 0 {
				t.Fatal("retired paste callback reported into the readonly attachment", a.Message, a.Issues)
			}
			requireSourcePasteIdentityFocusedAttachment(t, tester, true)
			tab.Doc.ReadOnly = false
			tab.Editor.SetReadOnly(false)
			tab.Editor.Paste = func() (string, error) {
				calls++
				return "P", nil
			}
			tester.Frame()
			requireSourcePasteIdentityFocusedAttachment(t, tester, false)
			tester.Command(command)
			if calls != 2 || buffer.String() != "aPbc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 2, Head: 2}) || buffer.History().UndoDepth != 1 || buffer.History().RedoDepth != 0 || a.Message != "" || len(a.Issues) != 0 {
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
