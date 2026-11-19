package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func TestBuiltinSourceCutCompositionCopySameTextPreeditRetainsOwnership(t *testing.T) {
	a, tab, tester := newSourcePasteIdentityTester(t)
	buffer := tab.Doc.Buffer
	tab.Editor.Reveal(1, 2)
	version, history := buffer.Version(), buffer.History()
	writes := 0
	tab.Editor.Copy = func(value string) error {
		writes++
		if value != "b" {
			t.Fatalf("clipboard write = %q, want literal starting selection b", value)
		}
		tab.Editor.SetMarkedText(nil, "b", ui.TextInputRange{Start: 0, End: 1})
		mark, active := tab.Editor.MarkedRange()
		if buffer.String() != "abc\r\ndef\r\n" || buffer.Version() != version || buffer.History() != history || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) || !active || mark != (ui.TextInputRange{Start: 1, End: 2}) {
			t.Fatal("same-text host preedit must retain literal source, version, history and nonempty selection", buffer.String(), mark, active)
		}
		return nil
	}
	tester.Command("cut")
	if got := buffer.String(); got != "abc\r\ndef\r\n" {
		t.Fatalf("old cut deleted new host preedit: source = %q, want literal unchanged source", got)
	}
	mark, active := tab.Editor.MarkedRange()
	if writes != 1 || !active || mark != (ui.TextInputRange{Start: 1, End: 2}) || buffer.Version() != version || buffer.History() != history || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) {
		t.Fatal("old cut settled or changed the new host owner", writes, mark, active, buffer.History())
	}
	if a.Message == "" || len(a.Issues) != 1 || a.Issues[0] != a.Message {
		t.Fatal("stale cut omitted its visible failure", a.Message, a.Issues)
	}
	tester.Type("中🙂")
	if buffer.String() != "a中🙂c\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 4, Head: 4}) || buffer.History().UndoDepth != 1 {
		t.Fatal("host Unicode commit did not own one literal history edit", buffer.String(), buffer.Selection(), buffer.History())
	}
	if _, active := tab.Editor.MarkedRange(); active || tab.Editor.Flush() != nil {
		t.Fatal("host commit retained a preedit or issue")
	}
	tester.Command("undo")
	if buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) {
		t.Fatal("host undo lost original CRLF source or selection", buffer.String(), buffer.Selection())
	}
	tester.Command("redo")
	if buffer.String() != "a中🙂c\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 4, Head: 4}) {
		t.Fatal("host redo changed Unicode source or selection", buffer.String(), buffer.Selection())
	}
}

func TestBuiltinSourceCutCompositionCopyThenDelayedOwnerAndDuplicateDone(t *testing.T) {
	a, tab, tester := newSourcePasteIdentityTester(t)
	buffer := tab.Doc.Buffer
	tab.Editor.Reveal(1, 2)
	version, history := buffer.Version(), buffer.History()
	var done func(error)
	writes := 0
	tab.Editor.CopyThen = func(value string, complete func(error)) {
		writes++
		if value != "b" {
			t.Fatalf("delayed clipboard write = %q, want b", value)
		}
		done = complete
	}
	tester.Command("cut")
	if done == nil || writes != 1 || buffer.String() != "abc\r\ndef\r\n" || buffer.Version() != version || buffer.History() != history || a.Message != "" || len(a.Issues) != 0 {
		t.Fatal("cut changed source before UI-thread done", done == nil, writes, buffer.String(), a.Message)
	}
	// This callback returns first; the host creates a new owner on the same
	// test UI thread before completing the clipboard write.
	tab.Editor.SetMarkedText(nil, "b", ui.TextInputRange{Start: 0, End: 1})
	mark, active := tab.Editor.MarkedRange()
	if buffer.Version() != version || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) || !active || mark != (ui.TextInputRange{Start: 1, End: 2}) {
		t.Fatal("delayed same-text owner fixture changed version or selection", mark, active)
	}
	done(nil)
	if buffer.String() != "abc\r\ndef\r\n" || buffer.Version() != version || buffer.History() != history || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) || a.Message == "" || len(a.Issues) != 1 {
		t.Fatal("delayed cut deleted host owner or omitted visible failure", buffer.String(), a.Message, a.Issues)
	}
	mark, active = tab.Editor.MarkedRange()
	if !active || mark != (ui.TextInputRange{Start: 1, End: 2}) {
		t.Fatal("delayed cut settled the host owner", mark, active)
	}
	message := a.Message
	done(nil)
	done(text.ErrRange)
	if buffer.String() != "abc\r\ndef\r\n" || buffer.Version() != version || buffer.History() != history || a.Message != message || len(a.Issues) != 1 {
		t.Fatal("duplicate done repeated deletion or reporting", buffer.String(), a.Message, a.Issues)
	}
	tester.Type("中🙂")
	if buffer.String() != "a中🙂c\r\ndef\r\n" || buffer.History().UndoDepth != 1 {
		t.Fatal("delayed host commit lost Unicode or created cut history", buffer.String(), buffer.History())
	}
	tester.Command("undo")
	if buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) {
		t.Fatal("delayed host undo lost literal source or selection")
	}
	tester.Command("redo")
	if buffer.String() != "a中🙂c\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 4, Head: 4}) {
		t.Fatal("delayed host redo lost Unicode or selection")
	}
}

func TestBuiltinSourceCutCompositionCopyNormalCutOwnsOneUndoRedo(t *testing.T) {
	a, tab, tester := newSourcePasteIdentityTester(t)
	buffer := tab.Doc.Buffer
	tab.Editor.Reveal(1, 2)
	writes := 0
	tab.Editor.Copy = func(value string) error {
		writes++
		if value != "b" {
			t.Fatalf("normal clipboard write = %q, want b", value)
		}
		return nil
	}
	tester.Command("cut")
	if writes != 1 || buffer.String() != "ac\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) || buffer.History().UndoDepth != 1 || a.Message != "" || len(a.Issues) != 0 {
		t.Fatal("normal cut did not own one deletion", writes, buffer.String(), buffer.History(), a.Message)
	}
	if _, active := tab.Editor.MarkedRange(); active || tab.Editor.Flush() != nil {
		t.Fatal("normal cut invented a composition or issue")
	}
	tester.Command("undo")
	if buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) || buffer.History().UndoDepth != 0 || buffer.History().RedoDepth != 1 {
		t.Fatal("normal cut undo lost original literal source or selection")
	}
	tester.Command("redo")
	if buffer.String() != "ac\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) || buffer.History().UndoDepth != 1 {
		t.Fatal("normal cut redo lost literal deletion")
	}
}

func TestBuiltinSourceCutCompositionCopyThenNormalCutOwnsOneUndoRedo(t *testing.T) {
	a, tab, tester := newSourcePasteIdentityTester(t)
	buffer := tab.Doc.Buffer
	tab.Editor.Reveal(1, 2)
	version, history := buffer.Version(), buffer.History()
	var done func(error)
	tab.Editor.CopyThen = func(value string, complete func(error)) {
		if value != "b" {
			t.Fatalf("normal delayed clipboard write = %q, want b", value)
		}
		done = complete
	}
	tester.Command("cut")
	if done == nil || buffer.String() != "abc\r\ndef\r\n" || buffer.Version() != version || buffer.History() != history {
		t.Fatal("normal delayed cut deleted before confirmation")
	}
	done(nil)
	if buffer.String() != "ac\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) || buffer.History().UndoDepth != 1 || a.Message != "" || len(a.Issues) != 0 {
		t.Fatal("confirmed delayed cut lost one literal deletion", buffer.String(), buffer.History(), a.Message)
	}
	cutVersion, cutHistory := buffer.Version(), buffer.History()
	done(nil)
	done(text.ErrRange)
	if buffer.String() != "ac\r\ndef\r\n" || buffer.Version() != cutVersion || buffer.History() != cutHistory || a.Message != "" || len(a.Issues) != 0 {
		t.Fatal("normal duplicate done repeated deletion or reporting")
	}
	tester.Command("undo")
	if buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) || buffer.History().UndoDepth != 0 || buffer.History().RedoDepth != 1 {
		t.Fatal("delayed cut undo lost original source or selection")
	}
	tester.Command("redo")
	if buffer.String() != "ac\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 1}) || buffer.History().UndoDepth != 1 {
		t.Fatal("delayed cut redo lost literal deletion")
	}
}

func TestBuiltinSourceCutCompositionCopyOnlyKeepsNewOwnerWithoutIssue(t *testing.T) {
	a, tab, tester := newSourcePasteIdentityTester(t)
	buffer := tab.Doc.Buffer
	tab.Editor.Reveal(1, 2)
	version, history := buffer.Version(), buffer.History()
	writes := 0
	tab.Editor.Copy = func(value string) error {
		writes++
		if value != "b" {
			t.Fatalf("copy-only clipboard write = %q, want b", value)
		}
		tab.Editor.SetMarkedText(nil, "b", ui.TextInputRange{Start: 0, End: 1})
		return nil
	}
	tester.Command("copy")
	mark, active := tab.Editor.MarkedRange()
	if writes != 1 || buffer.String() != "abc\r\ndef\r\n" || buffer.Version() != version || buffer.History() != history || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) || !active || mark != (ui.TextInputRange{Start: 1, End: 2}) || a.Message != "" || len(a.Issues) != 0 {
		t.Fatal("copy-only settled new owner or reported stale-cut failure", buffer.String(), mark, active, a.Message)
	}
	tester.Type("中🙂")
	if buffer.String() != "a中🙂c\r\ndef\r\n" || buffer.History().UndoDepth != 1 {
		t.Fatal("copy-only host commit lost its literal edit")
	}
	tester.Command("undo")
	if buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) {
		t.Fatal("copy-only host undo lost source or selection")
	}
	tester.Command("redo")
	if buffer.String() != "a中🙂c\r\ndef\r\n" {
		t.Fatal("copy-only host redo lost Unicode source")
	}
}

func TestBuiltinSourceCutCompositionCopyThenOnlyKeepsNewOwnerWithoutIssue(t *testing.T) {
	a, tab, tester := newSourcePasteIdentityTester(t)
	buffer := tab.Doc.Buffer
	tab.Editor.Reveal(1, 2)
	version, history := buffer.Version(), buffer.History()
	var done func(error)
	tab.Editor.CopyThen = func(value string, complete func(error)) {
		if value != "b" {
			t.Fatalf("delayed copy-only write = %q, want b", value)
		}
		done = complete
	}
	tester.Command("copy")
	if done == nil {
		t.Fatal("delayed copy-only host did not receive done")
	}
	tab.Editor.SetMarkedText(nil, "b", ui.TextInputRange{Start: 0, End: 1})
	done(nil)
	done(nil)
	mark, active := tab.Editor.MarkedRange()
	if buffer.String() != "abc\r\ndef\r\n" || buffer.Version() != version || buffer.History() != history || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) || !active || mark != (ui.TextInputRange{Start: 1, End: 2}) || a.Message != "" || len(a.Issues) != 0 {
		t.Fatal("delayed copy-only settled new owner or reported stale-cut failure", buffer.String(), mark, active, a.Message)
	}
	tester.Type("中🙂")
	if buffer.String() != "a中🙂c\r\ndef\r\n" || buffer.History().UndoDepth != 1 {
		t.Fatal("delayed copy-only host commit lost its edit")
	}
	tester.Command("undo")
	if buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) {
		t.Fatal("delayed copy-only undo lost source or selection")
	}
	tester.Command("redo")
	if buffer.String() != "a中🙂c\r\ndef\r\n" {
		t.Fatal("delayed copy-only redo lost Unicode source")
	}
}

func TestBuiltinSourceCutCompositionNewIssueRetainsOriginalErrorAndRecovery(t *testing.T) {
	_, tab, tester := newSourcePasteIdentityTester(t)
	buffer := text.NewWithLimit("abc\r\ndef\r\n", 11)
	tab.Doc.Buffer, tab.Editor.Buffer = buffer, buffer
	tester.Frame()
	tab.Editor.Reveal(1, 2)
	version, history := buffer.Version(), buffer.History()
	var reported, issue error
	tab.Editor.OnError = func(err error) { reported = err }
	tab.Editor.Copy = func(value string) error {
		if value != "b" {
			t.Fatalf("clipboard starting selection = %q, want b", value)
		}
		tab.Editor.SetMarkedText(nil, "中🙂", ui.TextInputRange{Start: 0, End: 3})
		issue = tab.Editor.Flush()
		if issue == nil || reported != issue {
			t.Fatal("refused host preedit did not retain its original error object", issue, reported)
		}
		return nil
	}
	tester.Command("cut")
	if reported != issue || tab.Editor.Flush() != issue || buffer.String() != "abc\r\ndef\r\n" || buffer.Version() != version || buffer.History() != history || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) {
		t.Fatal("successful write cleared or replaced the original composition error", reported, issue, buffer.String())
	}
	tester.Key(0, ui.KeyEscape)
	if err := tab.Editor.Flush(); err != nil {
		t.Fatal("Escape did not recover the pending host issue", err)
	}
	reported = nil
	tab.Editor.Copy = func(value string) error { return nil }
	tester.Command("cut")
	if buffer.String() != "ac\r\ndef\r\n" || buffer.History().UndoDepth != 1 || reported != nil {
		t.Fatal("cut after recovery did not own one deletion", buffer.String(), buffer.History(), reported)
	}
	tester.Command("undo")
	if buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) {
		t.Fatal("recovered cut undo lost original CRLF source or selection")
	}
	tester.Command("redo")
	if buffer.String() != "ac\r\ndef\r\n" {
		t.Fatal("recovered cut redo lost literal deletion")
	}
}

func TestBuiltinSourceCutCompositionCopyWriteErrorKeepsPriorityAndPendingIssue(t *testing.T) {
	_, tab, tester := newSourcePasteIdentityTester(t)
	buffer := text.NewWithLimit("abc\r\ndef\r\n", 11)
	tab.Doc.Buffer, tab.Editor.Buffer = buffer, buffer
	tester.Frame()
	tab.Editor.Reveal(1, 2)
	version, history := buffer.Version(), buffer.History()
	var reported, issue error
	tab.Editor.OnError = func(err error) { reported = err }
	tab.Editor.Copy = func(value string) error {
		if value != "b" {
			t.Fatalf("failed clipboard write = %q, want b", value)
		}
		tab.Editor.SetMarkedText(nil, "中🙂", ui.TextInputRange{Start: 0, End: 3})
		issue = tab.Editor.Flush()
		if issue == nil {
			t.Fatal("host preedit must retain a size issue")
		}
		return text.ErrRange
	}
	tester.Command("cut")
	if reported != text.ErrRange || tab.Editor.Flush() != issue || buffer.String() != "abc\r\ndef\r\n" || buffer.Version() != version || buffer.History() != history || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) {
		t.Fatal("write error lost priority or cleared the pending host issue", reported, issue, buffer.String())
	}
	tester.Key(0, ui.KeyEscape)
	if err := tab.Editor.Flush(); err != nil || buffer.String() != "abc\r\ndef\r\n" {
		t.Fatal("Escape did not recover failed clipboard host issue", err)
	}
	reported = nil
	tab.Editor.Copy = func(value string) error { return nil }
	tester.Command("cut")
	if reported != nil || buffer.String() != "ac\r\ndef\r\n" || buffer.History().UndoDepth != 1 {
		t.Fatal("write-error recovery did not permit one normal cut")
	}
	tester.Command("undo")
	if buffer.String() != "abc\r\ndef\r\n" || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) {
		t.Fatal("write-error recovery undo lost source or selection")
	}
}

func TestBuiltinSourceCutCompositionCopyThenWriteErrorKeepsPriorityAndPendingIssue(t *testing.T) {
	_, tab, tester := newSourcePasteIdentityTester(t)
	buffer := text.NewWithLimit("abc\r\ndef\r\n", 11)
	tab.Doc.Buffer, tab.Editor.Buffer = buffer, buffer
	tester.Frame()
	tab.Editor.Reveal(1, 2)
	version, history := buffer.Version(), buffer.History()
	var reported error
	var done func(error)
	reports := 0
	tab.Editor.OnError = func(err error) {
		reported = err
		reports++
	}
	tab.Editor.CopyThen = func(value string, complete func(error)) {
		if value != "b" {
			t.Fatalf("delayed failed clipboard write = %q, want b", value)
		}
		done = complete
	}
	tester.Command("cut")
	if done == nil {
		t.Fatal("delayed failed clipboard host did not receive done")
	}
	tab.Editor.SetMarkedText(nil, "中🙂", ui.TextInputRange{Start: 0, End: 3})
	issue := tab.Editor.Flush()
	if issue == nil || reported != issue {
		t.Fatal("delayed host preedit must retain its original size issue")
	}
	done(text.ErrRange)
	if reported != text.ErrRange || tab.Editor.Flush() != issue || buffer.String() != "abc\r\ndef\r\n" || buffer.Version() != version || buffer.History() != history || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) {
		t.Fatal("delayed write error lost priority or pending issue", reported, issue)
	}
	count := reports
	done(nil)
	done(text.ErrComposition)
	if reports != count || reported != text.ErrRange || tab.Editor.Flush() != issue || buffer.String() != "abc\r\ndef\r\n" || buffer.History() != history {
		t.Fatal("duplicate failed done changed source or repeated reporting")
	}
	// Direct UI-thread host input must be rendered before its Escape shortcut
	// exists; Command had already returned before this new issue was created.
	tester.Frame()
	tester.Key(0, ui.KeyEscape)
	if err := tab.Editor.Flush(); err != nil || buffer.String() != "abc\r\ndef\r\n" {
		t.Fatal("Escape did not recover delayed write failure", err)
	}
}

func TestBuiltinSourceCutCompositionReadOnlyCutRemainsCopyOnly(t *testing.T) {
	a, tab, tester := newSourcePasteIdentityTester(t)
	buffer := tab.Doc.Buffer
	tab.Editor.Reveal(1, 2)
	tab.Editor.ReadOnly = true
	version, history := buffer.Version(), buffer.History()
	writes := 0
	tab.Editor.Copy = func(value string) error {
		writes++
		if value != "b" {
			t.Fatalf("readonly cut write = %q, want b", value)
		}
		return nil
	}
	tester.Command("cut")
	if writes != 1 || buffer.String() != "abc\r\ndef\r\n" || buffer.Version() != version || buffer.History() != history || buffer.Selection() != (text.Range{Anchor: 1, Head: 2}) || a.Message != "" || len(a.Issues) != 0 {
		t.Fatal("readonly cut lost copy-only semantics", writes, buffer.String(), a.Message)
	}
}
