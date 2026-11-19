package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

var (
	_ editor.View    = (*admissionView)(nil)
	_ editor.Flusher = (*admissionView)(nil)
)

// The public Flusher callback is a synchronous permission-change fixture, not
// evidence of a built-in user workflow, concurrent mutation or disk corruption.
func TestReplaceReadOnlyAfterFlushCurrentEnter(t *testing.T) {
	a, tab := newSearchTextApp(t, "needle needle")
	doc, input, buffer := tab.Doc, tab.Editor, tab.Doc.Buffer
	buffer.ResetHistory()
	if err := buffer.ReplaceUTF16(13, 13, "!"); err != nil {
		t.Fatal(err)
	}
	if err := buffer.ReplaceUTF16(14, 14, "?"); err != nil {
		t.Fatal(err)
	}
	if !buffer.Undo() {
		t.Fatal("could not prepare a retained redo entry")
	}
	if err := buffer.SetSelection(6, 0); err != nil {
		t.Fatal(err)
	}
	history := buffer.History()
	if history.UndoDepth != 1 || history.RedoDepth != 1 || buffer.Version() != 4 {
		t.Fatalf("history fixture: version=%d history=%+v", buffer.Version(), history)
	}
	a.Query, a.Replacement, a.ShowFind = "needle", "match", true
	view := &admissionView{onFlush: func() error {
		doc.ReadOnly = true
		return nil
	}}
	tab.View, tab.Mode = view, "preview"
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if err := tester.Click("替换内容"); err != nil {
		t.Fatal(err)
	}
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	if buffer.String() != "needle needle!" || buffer.Version() != 4 ||
		buffer.Selection() != (text.Range{Anchor: 6, Head: 0}) || buffer.History() != history {
		t.Fatalf("readonly Flush must stop replacement without changing source/history/selection: text=%q version=%d selection=%+v history=%+v",
			buffer.String(), buffer.Version(), buffer.Selection(), buffer.History())
	}
	if view.flushes != 1 || view.releases != 0 || tab.View != view || tab.Mode != "preview" ||
		a.Current() != tab || tab.Doc != doc || tab.Editor != input || doc.Buffer != buffer || input.Buffer != buffer || !doc.ReadOnly {
		t.Fatalf("readonly Flush changed the live presentation: flush=%d release=%d mode=%q view=%p", view.flushes, view.releases, tab.Mode, tab.View)
	}
	if !a.ShowFind || a.Query != "needle" || a.Replacement != "match" || !tester.Focused("替换内容") {
		t.Fatalf("readonly replacement lost Find state: show=%v query=%q replacement=%q focus=%v", a.ShowFind, a.Query, a.Replacement, tester.Focused("替换内容"))
	}
	if _, visible := tester.Find("文档已变为只读；未替换"); !visible {
		t.Fatalf("readonly replacement did not report its outcome: %q", tester.Texts())
	}
}

func TestReplaceReadOnlyAfterFlushAllKeepsSynchronizedEdit(t *testing.T) {
	a, tab := newSearchTextApp(t, "needle needle")
	doc, input, buffer := tab.Doc, tab.Editor, tab.Doc.Buffer
	buffer.ResetHistory()
	if err := buffer.SetSelection(6, 0); err != nil {
		t.Fatal(err)
	}
	a.Query, a.Replacement, a.ShowFind = "needle", "match", true
	var flushedHistory text.HistoryStatus
	view := &admissionView{onFlush: func() error {
		if err := buffer.ReplaceUTF16(13, 13, " draft"); err != nil {
			return err
		}
		if err := buffer.SetSelection(6, 0); err != nil {
			return err
		}
		flushedHistory = buffer.History()
		doc.ReadOnly = true
		return nil
	}}
	tab.View, tab.Mode = view, "preview"
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if err := tester.Click("全部替换"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if buffer.String() != "needle needle draft" || buffer.Version() != 2 ||
		buffer.Selection() != (text.Range{Anchor: 6, Head: 0}) || buffer.History() != flushedHistory ||
		flushedHistory.UndoDepth != 1 || flushedHistory.RedoDepth != 0 {
		t.Fatalf("readonly replace-all must retain exactly the synchronized Flush edit: text=%q version=%d selection=%+v history=%+v flushed=%+v",
			buffer.String(), buffer.Version(), buffer.Selection(), buffer.History(), flushedHistory)
	}
	if view.flushes != 1 || view.releases != 0 || tab.View != view || tab.Mode != "preview" ||
		a.Current() != tab || tab.Doc != doc || tab.Editor != input || doc.Buffer != buffer || input.Buffer != buffer || !doc.ReadOnly {
		t.Fatalf("readonly replace-all changed presentation: flush=%d release=%d mode=%q view=%p", view.flushes, view.releases, tab.Mode, tab.View)
	}
	if !a.ShowFind || a.Query != "needle" || a.Replacement != "match" {
		t.Fatalf("readonly replace-all lost Find fields: show=%v query=%q replacement=%q", a.ShowFind, a.Query, a.Replacement)
	}
	if _, visible := tester.Find("文档已变为只读；未替换"); !visible {
		t.Fatalf("readonly replace-all did not report its outcome: %q", tester.Texts())
	}
	if !buffer.Undo() || buffer.String() != "needle needle" || !buffer.Redo() || buffer.String() != "needle needle draft" {
		t.Fatalf("the legal Flush edit lost its undo/redo contents: %q", buffer.String())
	}
}

func TestReplaceReadOnlyAfterFlushWritableAllSucceeds(t *testing.T) {
	a, tab := newSearchTextApp(t, "needle needle")
	doc, input, buffer := tab.Doc, tab.Editor, tab.Doc.Buffer
	buffer.ResetHistory()
	a.Query, a.Replacement, a.ShowFind = "needle", "match", true
	view := &admissionView{onFlush: func() error {
		return buffer.ReplaceUTF16(13, 13, " draft")
	}}
	tab.View, tab.Mode = view, "preview"
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if err := tester.Click("全部替换"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if buffer.String() != "match match draft" || buffer.Version() != 4 ||
		buffer.History().UndoDepth != 2 || buffer.History().RedoDepth != 0 {
		t.Fatalf("writable Flush must keep ordinary replace-all behavior: text=%q version=%d history=%+v",
			buffer.String(), buffer.Version(), buffer.History())
	}
	if view.flushes != 1 || view.releases == 0 || a.Current() != tab || tab.Doc != doc || tab.Editor != input ||
		doc.Buffer != buffer || input.Buffer != buffer || doc.ReadOnly || input.ReadOnly || !a.ShowFind {
		t.Fatalf("writable replacement failed its live target: flush=%d release=%d readonly=%v show=%v",
			view.flushes, view.releases, doc.ReadOnly, a.ShowFind)
	}
	if _, visible := tester.Find("已替换 2 处"); !visible {
		t.Fatalf("writable replace-all did not report success: %q", tester.Texts())
	}
	if !buffer.Undo() || buffer.String() != "needle needle draft" || !buffer.Undo() || buffer.String() != "needle needle" {
		t.Fatalf("replacement and Flush must remain separate undo steps: %q", buffer.String())
	}
	if !buffer.Redo() || buffer.String() != "needle needle draft" || !buffer.Redo() || buffer.String() != "match match draft" {
		t.Fatalf("replacement and Flush lost redo contents: %q", buffer.String())
	}
}

func TestReplaceReadOnlyAfterFlushCurrentButtonKeepsPendingInput(t *testing.T) {
	a, tab := newSearchTextApp(t, "needle needle")
	doc, input, buffer := tab.Doc, tab.Editor, tab.Doc.Buffer
	buffer.ResetHistory()
	if err := buffer.SetSelection(6, 0); err != nil {
		t.Fatal(err)
	}
	a.Query, a.Replacement, a.ShowFind = "needle", "match", true
	var flushedHistory text.HistoryStatus
	view := &admissionView{onFlush: func() error {
		// The callback exercises the editor's public input ownership. App must
		// not unmark a newly pending input after write permission has changed.
		pending := ui.TextInputRange{Start: 13, End: 13}
		input.SetMarkedText(&pending, " draft", ui.TextInputRange{Start: 6, End: 6})
		if err := buffer.SetSelection(6, 0); err != nil {
			return err
		}
		flushedHistory = buffer.History()
		doc.ReadOnly = true
		return nil
	}}
	tab.View, tab.Mode = view, "preview"
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if err := tester.Click("替换当前"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	marked, pending := input.MarkedRange()
	if !pending || marked != (ui.TextInputRange{Start: 13, End: 19}) || !buffer.InGroup() ||
		buffer.String() != "needle needle draft" || buffer.Version() != 2 ||
		buffer.Selection() != (text.Range{Anchor: 6, Head: 0}) || buffer.History() != flushedHistory ||
		flushedHistory.UndoDepth != 0 || flushedHistory.RedoDepth != 0 {
		t.Fatalf("readonly current button committed or discarded pending input: pending=%v marked=%+v group=%v text=%q version=%d selection=%+v history=%+v",
			pending, marked, buffer.InGroup(), buffer.String(), buffer.Version(), buffer.Selection(), buffer.History())
	}
	if view.flushes != 1 || view.releases != 0 || tab.View != view || tab.Mode != "preview" ||
		a.Current() != tab || tab.Doc != doc || tab.Editor != input || doc.Buffer != buffer || input.Buffer != buffer || !doc.ReadOnly {
		t.Fatalf("readonly current button changed presentation: flush=%d release=%d mode=%q view=%p", view.flushes, view.releases, tab.Mode, tab.View)
	}
	if _, visible := tester.Find("文档已变为只读；未替换"); !visible {
		t.Fatalf("readonly current button did not report its outcome: %q", tester.Texts())
	}
	// Explicit test cleanup happens only after all pending-input assertions.
	input.UnmarkText()
}
