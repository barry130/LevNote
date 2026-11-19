package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func TestBuiltinSourcePasteCompositionNewEmptyPreeditRetainsOwnership(t *testing.T) {
	_, tab, tester := newSourcePasteIdentityTester(t)
	buffer := tab.Doc.Buffer
	version, history := buffer.Version(), buffer.History()
	reads := 0
	var reported error
	tab.Editor.OnError = func(err error) { reported = err }
	tab.Editor.Paste = func() (string, error) {
		reads++
		tab.Editor.SetMarkedText(nil, "", ui.TextInputRange{})
		if buffer.String() != "abc\r\ndef\r\n" || buffer.Version() != version || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) {
			t.Fatal("empty host preedit changed text, version or selection")
		}
		if mark, active := tab.Editor.MarkedRange(); !active || mark != (ui.TextInputRange{Start: 1, End: 1}) {
			t.Fatal("empty host preedit was not active", mark, active)
		}
		return "P", nil
	}
	tester.Command("paste")
	if got := buffer.String(); got != "abc\r\ndef\r\n" {
		t.Fatalf("source = %q, want literal unchanged source %q", got, "abc\r\ndef\r\n")
	}
	if mark, active := tab.Editor.MarkedRange(); !active || mark != (ui.TextInputRange{Start: 1, End: 1}) {
		t.Fatal("old paste settled the new host preedit", mark, active)
	}
	if reads != 1 || reported == nil || buffer.Version() != version || buffer.History() != history || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) {
		t.Fatal("stale paste changed source state or omitted visible failure", reads, reported)
	}
	tester.Type("中")
	if got := buffer.String(); got != "a中bc\r\ndef\r\n" {
		t.Fatalf("host composition commit = %q, want literal host edit", got)
	}
	if _, active := tab.Editor.MarkedRange(); active || buffer.History().UndoDepth != 1 {
		t.Fatal("host commit retained a preedit or invented paste history", buffer.History())
	}
	tester.Command("undo")
	if got := buffer.String(); got != "abc\r\ndef\r\n" {
		t.Fatalf("undo host composition = %q", got)
	}
	tester.Command("redo")
	if got := buffer.String(); got != "a中bc\r\ndef\r\n" {
		t.Fatalf("redo host composition = %q", got)
	}
}

func TestBuiltinSourcePasteCompositionNewRefusalKeepsOriginalIssue(t *testing.T) {
	_, tab, tester := newSourcePasteIdentityTester(t)
	buffer := text.NewWithLimit("abc\r\ndef\r\n", 11)
	tab.Doc.Buffer, tab.Editor.Buffer = buffer, buffer
	tester.Frame()
	tab.Editor.Reveal(1, 1)
	version, history := buffer.Version(), buffer.History()
	var reported, issue error
	tab.Editor.OnError = func(err error) { reported = err }
	tab.Editor.Paste = func() (string, error) {
		tab.Editor.SetMarkedText(nil, "中", ui.TextInputRange{Start: 1, End: 1})
		issue = tab.Editor.Flush()
		if issue == nil || reported != issue || buffer.InGroup() {
			t.Fatal("host refused preedit did not retain its original issue", issue, reported)
		}
		return "P", nil
	}
	tester.Command("paste")
	if got := buffer.String(); got != "abc\r\ndef\r\n" {
		t.Fatalf("source after refused host preedit = %q, want literal unchanged source", got)
	}
	if reported != issue || tab.Editor.Flush() != issue || buffer.Version() != version || buffer.History() != history || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) {
		t.Fatal("old paste cleared or replaced the host composition issue", reported, issue)
	}
	tester.Key(0, ui.KeyEscape)
	if err := tab.Editor.Flush(); err != nil {
		t.Fatal("explicit Escape did not recover the host issue", err)
	}
	tab.Editor.Paste = func() (string, error) { return "P", nil }
	tester.Command("paste")
	if got := buffer.String(); got != "aPbc\r\ndef\r\n" {
		t.Fatalf("paste after explicit recovery = %q", got)
	}
	tester.Command("undo")
	if buffer.String() != "abc\r\ndef\r\n" {
		t.Fatal("recovered paste undo changed the original CRLF source")
	}
}

func TestBuiltinSourcePasteCompositionSameTextPreeditKeepsSelectionAndUndo(t *testing.T) {
	for _, command := range []string{"paste", "pasteAndMatchStyle"} {
		t.Run(command, func(t *testing.T) {
			_, tab, tester := newSourcePasteIdentityTester(t)
			buffer := tab.Doc.Buffer
			tab.Editor.Reveal(1, 2)
			version, history := buffer.Version(), buffer.History()
			var reported error
			tab.Editor.OnError = func(err error) { reported = err }
			tab.Editor.Paste = func() (string, error) {
				tab.Editor.SetMarkedText(nil, "b", ui.TextInputRange{Start: 0, End: 1})
				if buffer.Version() != version || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) {
					t.Fatal("same-text preedit changed version or selection")
				}
				return "P", nil
			}
			tester.Command(command)
			mark, active := tab.Editor.MarkedRange()
			if buffer.String() != "abc\r\ndef\r\n" || buffer.Version() != version || buffer.History() != history || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) || !active || mark != (ui.TextInputRange{Start: 1, End: 2}) || reported == nil {
				t.Fatal("same-text host preedit was overwritten or settled", buffer.String(), mark, active, reported)
			}
			tester.Type("中🙂")
			if buffer.String() != "a中🙂c\r\ndef\r\n" || buffer.History().UndoDepth != 1 {
				t.Fatal("host Unicode commit did not own one history edit", buffer.String(), buffer.History())
			}
			tester.Command("undo")
			if buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) {
				t.Fatal("host commit undo lost literal source or original selection")
			}
			tester.Command("redo")
			if buffer.String() != "a中🙂c\r\ndef\r\n" {
				t.Fatal("host Unicode redo failed", buffer.String())
			}
		})
	}
}

func TestBuiltinSourcePasteCompositionReadErrorKeepsPriorityAndPendingIssue(t *testing.T) {
	_, tab, tester := newSourcePasteIdentityTester(t)
	buffer := text.NewWithLimit("abc\r\ndef\r\n", 11)
	tab.Doc.Buffer, tab.Editor.Buffer = buffer, buffer
	tester.Frame()
	tab.Editor.Reveal(1, 1)
	version, history := buffer.Version(), buffer.History()
	var reported, issue error
	tab.Editor.OnError = func(err error) { reported = err }
	tab.Editor.Paste = func() (string, error) {
		tab.Editor.SetMarkedText(nil, "中", ui.TextInputRange{Start: 1, End: 1})
		issue = tab.Editor.Flush()
		if issue == nil {
			t.Fatal("host preedit should retain a size issue")
		}
		return "P", text.ErrRange
	}
	tester.Command("pasteAndMatchStyle")
	if reported != text.ErrRange || tab.Editor.Flush() != issue || buffer.String() != "abc\r\ndef\r\n" || buffer.Version() != version || buffer.History() != history || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) {
		t.Fatal("read error priority or pending host issue changed", reported, issue)
	}
	tester.Key(0, ui.KeyEscape)
	if err := tab.Editor.Flush(); err != nil || buffer.String() != "abc\r\ndef\r\n" {
		t.Fatal("host issue did not recover without changing text", err)
	}
}

func TestBuiltinSourcePasteCompositionUnchangedEntryOwnerAllowsPaste(t *testing.T) {
	for _, command := range []string{"paste", "pasteAndMatchStyle"} {
		t.Run(command, func(t *testing.T) {
			for _, preedit := range []string{"", "n"} {
				t.Run("preedit="+preedit, func(t *testing.T) {
					_, tab, tester := newSourcePasteIdentityTester(t)
					buffer := tab.Doc.Buffer
					tab.Editor.SetMarkedText(nil, preedit, ui.TextInputRange{Start: len(preedit), End: len(preedit)})
					if _, active := tab.Editor.MarkedRange(); !active {
						t.Fatal("entry owner fixture is not composing")
					}
					var reported error
					reads := 0
					tab.Editor.OnError = func(err error) { reported = err }
					tab.Editor.Paste = func() (string, error) { reads++; return "中\r\n🙂", nil }
					tester.Command(command)
					if buffer.String() != "a中\r\n🙂bc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 6, Head: 6}) || buffer.History().UndoDepth != 1 || reads != 1 || reported != nil {
						t.Fatal("unchanged entry owner did not accept literal paste", buffer.String(), buffer.Selection(), buffer.History(), reported)
					}
					if _, active := tab.Editor.MarkedRange(); active {
						t.Fatal("ordinary entry paste did not settle the owner")
					}
					tester.Command("undo")
					if buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) {
						t.Fatal("entry paste undo did not restore literal original source")
					}
					tester.Command("redo")
					if buffer.String() != "a中\r\n🙂bc\r\ndef\r\n" {
						t.Fatal("entry paste redo changed Unicode or CRLF", buffer.String())
					}
				})
			}
		})
	}
}

func TestBuiltinSourcePasteCompositionCallbackSettlesEntryOwnerWithoutPaste(t *testing.T) {
	for _, command := range []string{"paste", "pasteAndMatchStyle"} {
		t.Run(command, func(t *testing.T) {
			_, tab, tester := newSourcePasteIdentityTester(t)
			buffer := tab.Doc.Buffer
			tab.Editor.SetMarkedText(nil, "", ui.TextInputRange{})
			version, history := buffer.Version(), buffer.History()
			var reported error
			tab.Editor.OnError = func(err error) { reported = err }
			tab.Editor.Paste = func() (string, error) {
				if err := tab.Editor.Flush(); err != nil {
					t.Fatal("host failed to settle its entry owner", err)
				}
				if _, active := tab.Editor.MarkedRange(); active {
					t.Fatal("host Flush retained its empty preedit")
				}
				return "P", nil
			}
			tester.Command(command)
			if buffer.String() != "abc\r\ndef\r\n" || buffer.Version() != version || buffer.History() != history || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) || reported == nil {
				t.Fatal("old paste continued after the host settled its owner", buffer.String(), reported)
			}
			if _, active := tab.Editor.MarkedRange(); active {
				t.Fatal("old paste restored a settled owner")
			}
			tab.Editor.Paste = func() (string, error) { return "P", nil }
			tester.Command(command)
			if buffer.String() != "aPbc\r\ndef\r\n" || buffer.History().UndoDepth != 1 {
				t.Fatal("subsequent new paste failed", buffer.String(), buffer.History())
			}
			tester.Command("undo")
			if buffer.String() != "abc\r\ndef\r\n" {
				t.Fatal("subsequent new paste undo changed original CRLF source")
			}
		})
	}
}
