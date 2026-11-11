package app

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

func newSourceNavigationFixture(t *testing.T, value string) (*App, *Tab, string) {
	t.Helper()
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))
	a.NewDocument("text")
	tab := a.Current()
	if tab == nil {
		t.Fatal("source navigation fixture has no tab")
	}
	if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), value); err != nil {
		t.Fatal(err)
	}
	tab.Doc.Buffer.ResetHistory()
	if err := tab.Doc.Buffer.SetSelection(0, 0); err != nil {
		t.Fatal(err)
	}
	return a, tab, root
}

func selectionPoint(t *testing.T, tab *Tab) int {
	t.Helper()
	selection := tab.Doc.Buffer.Selection()
	if selection.Anchor != selection.Head {
		t.Fatalf("navigation selection is not a caret: %+v", selection)
	}
	return selection.Head
}

type navigationFlushView struct {
	failingView
	flush func() error
}

type countingFlushView struct {
	editor.View
	flushes int
}

func (v *countingFlushView) Flush() error {
	v.flushes++
	return editor.FlushView(v.View)
}

func (v *navigationFlushView) Flush() error {
	v.flushes++
	if v.flush != nil {
		return v.flush()
	}
	return v.flushErr
}

func TestParseSourceLineTargetStrictGrammarAndOverflow(t *testing.T) {
	valid := map[string]sourceLineTarget{
		"1":       {line: 1, column: 1},
		" 42 ":    {line: 42, column: 1},
		"12:5":    {line: 12, column: 5},
		"\t7:19 ": {line: 7, column: 19},
	}
	for query, want := range valid {
		got, err := parseSourceLineTarget(query)
		if err != nil || got != want {
			t.Errorf("parse %q = %+v, %v; want %+v", query, got, err, want)
		}
	}
	for _, query := range []string{
		"", " ", "0", "01", ":1", "1:", "1::2", "+1", "-1", "1:+2", "1.2",
		"１", "1:２", "a", "1:a", strings.Repeat("9", 100),
	} {
		if got, err := parseSourceLineTarget(query); err == nil {
			t.Errorf("parse %q accepted as %+v", query, got)
		}
	}
}

func TestValidateSourceLineTargetCRLFEmptyFinalSurrogateAndCombining(t *testing.T) {
	buffer := text.New("a\r\n\r\n🙂\r\ne\u0301\n")
	tests := []struct {
		name   string
		line   int
		column int
		ok     bool
	}{
		{name: "CR excluded at EOL", line: 1, column: 2, ok: true},
		{name: "CR is not a column", line: 1, column: 3},
		{name: "middle empty line", line: 2, column: 1, ok: true},
		{name: "surrogate start", line: 3, column: 1, ok: true},
		{name: "surrogate interior", line: 3, column: 2},
		{name: "surrogate end", line: 3, column: 3, ok: true},
		{name: "combining interior addressable", line: 4, column: 2, ok: true},
		{name: "combining EOL", line: 4, column: 3, ok: true},
		{name: "empty final line", line: 5, column: 1, ok: true},
		{name: "outside empty final line", line: 5, column: 2},
		{name: "line overflow", line: 6, column: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSourceLineTarget(buffer, sourceLineTarget{line: tc.line, column: tc.column})
			if tc.ok && err != nil || !tc.ok && err == nil {
				t.Fatalf("validate %d:%d = %v, want ok=%v", tc.line, tc.column, err, tc.ok)
			}
		})
	}
}

func TestSourceNavigationActualShortcutFocusEnterAndStatusAction(t *testing.T) {
	a, tab, _ := newSourceNavigationFixture(t, "first\r\nsecond")
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()

	tester.Key(ui.Cmd, ui.KeyG)
	if !a.ShowGoToLine || !tester.Focused("转到行号") {
		t.Fatalf("Ctrl+G did not open and focus the source-line field: show=%v texts=%q", a.ShowGoToLine, tester.Texts())
	}
	tester.Type("2:4")
	tester.Key(0, ui.KeyEnter)
	if a.ShowGoToLine {
		t.Fatalf("Enter did not close source-line navigation: %q", a.GoToLineError)
	}
	line, column, err := tab.Doc.Buffer.LineColumn(selectionPoint(t, tab))
	if err != nil || line != 1 || column != 3 {
		t.Fatalf("Enter selection = line %d column %d err=%v", line, column, err)
	}
	tester.Frame()
	status := "第 2 行，第 4 列"
	if err := tester.Click(status); err != nil {
		t.Fatalf("status action is not discoverable as %q: %v; texts=%q", status, err, tester.Texts())
	}
	if !a.ShowGoToLine || !tester.Focused("转到行号") {
		t.Fatal("status action did not open and focus source-line navigation")
	}
}

func TestSourceNavigationKeyboardTabPointerCancelAndEscapeAreScoped(t *testing.T) {
	a, tab, _ := newSourceNavigationFixture(t, "alpha\nbeta")
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if !a.openGoToLine() {
		t.Fatal("could not open source-line panel")
	}
	tester.Frame()

	tester.Key(0, ui.KeyTab)
	if !tester.Focused("转到源码行") {
		t.Fatalf("Tab did not reach submit: texts=%q", tester.Texts())
	}
	tester.Key(0, ui.KeyTab)
	if !tester.Focused("取消转到行号") {
		t.Fatalf("second Tab did not reach cancel: texts=%q", tester.Texts())
	}
	tester.Key(ui.Shift, ui.KeyTab)
	if !tester.Focused("转到源码行") {
		t.Fatalf("Shift+Tab did not return to submit: texts=%q", tester.Texts())
	}
	tester.Key(0, ui.KeyEscape)
	if a.ShowGoToLine {
		t.Fatal("Escape did not close the panel while the submit button was focused")
	}

	if !a.openGoToLine() {
		t.Fatal("could not reopen source-line panel")
	}
	tester.Frame()
	tester.Key(0, ui.KeyTab)
	tester.Key(0, ui.KeyTab)
	if !tester.Focused("取消转到行号") {
		t.Fatalf("second Tab did not focus cancel for Escape: texts=%q", tester.Texts())
	}
	tester.Key(0, ui.KeyEscape)
	if a.ShowGoToLine {
		t.Fatal("Escape did not close the panel while the cancel button was focused")
	}

	if !a.openGoToLine() {
		t.Fatal("could not reopen source-line panel")
	}
	a.ShowFind = true
	tester.Frame()
	tester.Key(0, ui.KeyEscape)
	if a.ShowGoToLine || !a.ShowFind {
		t.Fatalf("input Escape was stolen by a global branch: gotoline=%v find=%v", a.ShowGoToLine, a.ShowFind)
	}
	a.ShowFind = false

	if !a.openGoToLine() {
		t.Fatal("could not reopen source-line panel")
	}
	tester.Frame()
	tester.Type("2:2")
	tester.Key(0, ui.KeyEscape)
	if a.ShowGoToLine || selectionPoint(t, tab) != 0 {
		t.Fatal("Escape changed the document selection or retained a stale panel")
	}

	if !a.openGoToLine() {
		t.Fatal("could not reopen source-line panel")
	}
	tester.Frame()
	if err := tester.Click("取消转到行号"); err != nil {
		t.Fatal(err)
	}
	if a.ShowGoToLine || selectionPoint(t, tab) != 0 {
		t.Fatal("pointer cancel changed selection or retained a stale panel")
	}

	if !a.openGoToLine() {
		t.Fatal("could not reopen source-line panel")
	}
	tester.Frame()
	tester.Type("2:3")
	if err := tester.Click("转到源码行"); err != nil {
		t.Fatal(err)
	}
	line, column, err := tab.Doc.Buffer.LineColumn(selectionPoint(t, tab))
	if err != nil || line != 1 || column != 2 || a.ShowGoToLine {
		t.Fatalf("pointer submit = line %d column %d err=%v panel=%v", line, column, err, a.ShowGoToLine)
	}
}

func TestSourceNavigationCompositionEscapeAndPointerSubmit(t *testing.T) {
	value := strings.Repeat("line\n", 122) + "tail"
	a, tab, _ := newSourceNavigationFixture(t, value)
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if !a.openGoToLine() {
		t.Fatal("could not open source-line panel")
	}
	tester.Frame()

	tester.Compose("2", 1)
	if !a.GoToLineComposing || !tester.Focused("转到行号") {
		t.Fatalf("synthetic IME composition was not routed to the field: composing=%v", a.GoToLineComposing)
	}
	tester.Key(0, ui.KeyEscape)
	if !a.ShowGoToLine || !a.GoToLineComposing {
		t.Fatalf("Escape stole the active field composition: panel=%v composing=%v", a.ShowGoToLine, a.GoToLineComposing)
	}
	tester.Compose("", 0)
	if a.GoToLineComposing {
		t.Fatal("ending the composition left the routed state active")
	}
	tester.Key(0, ui.KeyEscape)
	if a.ShowGoToLine {
		t.Fatal("Escape did not close the panel after composition ended")
	}

	history := tab.Doc.Buffer.History()
	version := tab.Doc.Buffer.Version()
	if !a.openGoToLine() {
		t.Fatal("could not reopen source-line panel for pointer cancel")
	}
	tester.Frame()
	tester.Type("12")
	tester.Compose("3", 1)
	if err := tester.Click("取消转到行号"); err != nil {
		t.Fatal(err)
	}
	if a.ShowGoToLine || a.GoToLineComposing || tab.Doc.Buffer.History() != history ||
		tab.Doc.Buffer.Version() != version || selectionPoint(t, tab) != 0 {
		t.Fatalf("pointer cancel during composition changed state: panel=%v composing=%v history=%+v version=%d selection=%d",
			a.ShowGoToLine, a.GoToLineComposing, tab.Doc.Buffer.History(), tab.Doc.Buffer.Version(), selectionPoint(t, tab))
	}
	if !a.openGoToLine() {
		t.Fatal("could not reopen source-line panel after canceling composition")
	}
	tester.Frame()
	if a.GoToLineQuery != "" || a.GoToLineComposing || !tester.Focused("转到行号") {
		t.Fatalf("generation-scoped field retained stale state: query=%q composing=%v texts=%q",
			a.GoToLineQuery, a.GoToLineComposing, tester.Texts())
	}
	tester.Type("2:1")
	if a.GoToLineQuery != "2:1" {
		t.Fatalf("fresh generation field retained stale text: %q", a.GoToLineQuery)
	}
	tester.Key(0, ui.KeyEscape)

	if !a.openGoToLine() {
		t.Fatal("could not reopen source-line panel for pointer submit")
	}
	tester.Frame()
	tester.Type("12")
	tester.Compose("3", 1)
	if err := tester.Click("转到源码行"); err != nil {
		t.Fatal(err)
	}
	if !a.ShowGoToLine || !a.GoToLineComposing || !tester.Focused("转到行号") ||
		!strings.Contains(a.GoToLineError, "输入法正在组合") {
		t.Fatalf("pointer submit crossed active composition: panel=%v composing=%v error=%q",
			a.ShowGoToLine, a.GoToLineComposing, a.GoToLineError)
	}
	if selectionPoint(t, tab) != 0 {
		t.Fatal("pointer submit during composition changed the document selection")
	}
	tester.Type("3")
	if a.GoToLineComposing || a.GoToLineQuery != "123" {
		t.Fatalf("composition did not commit the complete query: composing=%v query=%q",
			a.GoToLineComposing, a.GoToLineQuery)
	}
	if err := tester.Click("转到源码行"); err != nil {
		t.Fatal(err)
	}
	line, column, err := tab.Doc.Buffer.LineColumn(selectionPoint(t, tab))
	if err != nil || line != 122 || column != 0 || a.ShowGoToLine {
		t.Fatalf("post-composition submit = line %d column %d err=%v panel=%v",
			line, column, err, a.ShowGoToLine)
	}
}

func TestSourceNavigationFlushesRealMarkdownDraft(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.md")
	if err := os.WriteFile(path, []byte("# 标题\n\n原生写作\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "markdown" {
		t.Fatal("real Markdown adapter did not open")
	}
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	beginRealMarkdownComposition(t, tab, tester)
	tester.Type("中文🙂")
	if !strings.Contains(tab.Doc.Buffer.String(), "中文🙂") {
		t.Fatalf("real Markdown draft did not synchronize its committed edit: %q", tab.Doc.Buffer.String())
	}
	counted := &countingFlushView{View: tab.View}
	tab.View = counted
	if !a.openGoToLine() {
		t.Fatal("could not open source-line panel over a real Markdown draft")
	}
	tester.Frame()
	if !a.submitGoToLine("2:1") {
		t.Fatalf("real Markdown draft navigation failed: %s", a.GoToLineError)
	}
	if !strings.Contains(tab.Doc.Buffer.String(), "# 中文🙂") {
		t.Fatalf("source navigation did not flush the real Markdown draft: %q", tab.Doc.Buffer.String())
	}
	if counted.flushes != 1 {
		t.Fatalf("source navigation called real Markdown Flush %d times", counted.flushes)
	}
	line, column, err := tab.Doc.Buffer.LineColumn(selectionPoint(t, tab))
	if err != nil || line != 1 || column != 0 || tab.Mode != "source" {
		t.Fatalf("real draft target = mode=%q line=%d column=%d err=%v", tab.Mode, line, column, err)
	}
}

func TestSourceNavigationCancelPreservesHistoryAndSelection(t *testing.T) {
	a, tab, _ := newSourceNavigationFixture(t, "abc")
	if err := tab.Doc.Buffer.ReplaceUTF16(3, 3, "d"); err != nil {
		t.Fatal(err)
	}
	if err := tab.Doc.Buffer.SetSelection(2, 2); err != nil {
		t.Fatal(err)
	}
	history := tab.Doc.Buffer.History()
	version := tab.Doc.Buffer.Version()
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if !a.openGoToLine() {
		t.Fatal("could not open source-line panel")
	}
	tester.Frame()
	tester.Type("not valid")
	tester.Key(0, ui.KeyEscape)
	if tab.Doc.Buffer.History() != history || tab.Doc.Buffer.Version() != version ||
		tab.Doc.Buffer.Selection() != (text.Range{Anchor: 2, Head: 2}) || a.goToLineTab != nil {
		t.Fatalf("cancel changed document state: history=%+v version=%d selection=%+v captured=%v",
			tab.Doc.Buffer.History(), tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection(), a.goToLineTab)
	}
}

func TestSourceNavigationOldPanelCallbacksCannotUseReopenedTarget(t *testing.T) {
	a, tab, _ := newSourceNavigationFixture(t, "alpha\nbeta\ngamma")
	if !a.openGoToLine() {
		t.Fatal("could not open source-line panel")
	}
	oldGeneration := a.goToLineGeneration
	oldCallback := func() {
		a.submitGoToLineAt(oldGeneration, "3:1")
	}
	if !a.openGoToLine() {
		t.Fatal("could not refresh source-line panel")
	}
	oldCallback()
	if !a.ShowGoToLine || a.GoToLineQuery != "" || selectionPoint(t, tab) != 0 {
		t.Fatalf("old callback affected the reopened panel: panel=%v query=%q selection=%d",
			a.ShowGoToLine, a.GoToLineQuery, selectionPoint(t, tab))
	}
}

func TestSourceNavigationInvalidTargetKeepsCaretAndDisclosesSurrogate(t *testing.T) {
	a, tab, _ := newSourceNavigationFixture(t, "🙂x")
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if !a.openGoToLine() {
		t.Fatal("could not open source-line panel")
	}
	tester.Frame()
	tester.Type("1:2")
	tester.Key(0, ui.KeyEnter)
	if !a.ShowGoToLine || !strings.Contains(a.GoToLineError, "代理项") {
		t.Fatalf("surrogate interior was not rejected locally: panel=%v error=%q", a.ShowGoToLine, a.GoToLineError)
	}
	if selectionPoint(t, tab) != 0 {
		t.Fatalf("invalid surrogate target moved selection to %d", selectionPoint(t, tab))
	}
}

func TestSourceNavigationCompositionBarrierSourceAndRealMarkdown(t *testing.T) {
	t.Run("source", func(t *testing.T) {
		a, tab, _ := newSourceNavigationFixture(t, "")
		mark := ui.TextInputRange{Start: 0, End: 0}
		tab.Editor.SetMarkedText(&mark, "中", ui.TextInputRange{Start: 1, End: 1})
		if a.openGoToLine() || !strings.Contains(a.Message, "输入法正在组合") {
			t.Fatalf("source composition barrier failed: show=%v message=%q", a.ShowGoToLine, a.Message)
		}
		tab.Editor.UnmarkText()
	})

	t.Run("real Markdown", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "note.md")
		if err := os.WriteFile(path, []byte("# 标题\n\n原生写作\n"), 0600); err != nil {
			t.Fatal(err)
		}
		a := newTestApp(t, filepath.Join(root, "config"))
		a.Open(path)
		tab := a.Current()
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		beginRealMarkdownComposition(t, tab, tester)
		composer, ok := tab.View.(editor.Composer)
		if !ok || !composer.CompositionActive() {
			t.Fatal("real Markdown composition was not exposed through the optional contract")
		}
		line, column, _ := tab.Doc.Buffer.LineColumn(tab.Doc.Buffer.Selection().Head)
		status := fmt.Sprintf("第 %d 行，第 %d 列", line+1, column+1)
		if err := tester.Click(status); err != nil {
			t.Fatalf("status action unavailable during composition: %v; texts=%q", err, tester.Texts())
		}
		if a.ShowGoToLine || !composer.CompositionActive() || !strings.Contains(a.Message, "输入法正在组合") {
			t.Fatalf("status action stole Markdown composition: show=%v active=%v message=%q",
				a.ShowGoToLine, composer.CompositionActive(), a.Message)
		}
	})
}

func TestSourceNavigationFlushRefusalAndPostFlushInvalidTarget(t *testing.T) {
	t.Run("invalid syntax does not flush", func(t *testing.T) {
		a, tab, _ := newSourceNavigationFixture(t, "one\ntwo")
		view := &navigationFlushView{flushErr: errors.New("should not run")}
		tab.View = view
		if !a.openGoToLine() {
			t.Fatal("could not open source-line panel")
		}
		if a.submitGoToLine("bad") {
			t.Fatal("invalid syntax unexpectedly submitted")
		}
		if view.flushes != 0 || a.GoToLineError == "" || a.GoToLineQuery != "bad" || !a.ShowGoToLine {
			t.Fatalf("invalid syntax changed state: flushes=%d panel=%v query=%q error=%q",
				view.flushes, a.ShowGoToLine, a.GoToLineQuery, a.GoToLineError)
		}
	})

	t.Run("flush refusal retains target", func(t *testing.T) {
		a, tab, _ := newSourceNavigationFixture(t, "one\ntwo")
		if err := tab.Doc.Buffer.SetSelection(2, 2); err != nil {
			t.Fatal(err)
		}
		view := &navigationFlushView{flushErr: errors.New("draft conflict")}
		tab.View = view
		if !a.openGoToLine() {
			t.Fatal("could not open source-line panel")
		}
		if a.submitGoToLine("2:2") {
			t.Fatal("submit succeeded after Flush refusal")
		}
		if view.flushes != 1 || view.releases != 0 || !a.ShowGoToLine || a.GoToLineQuery != "2:2" ||
			selectionPoint(t, tab) != 2 || a.goToLineBuffer != tab.Doc.Buffer {
			t.Fatalf("Flush refusal lost state: view=%+v panel=%v error=%q selection=%d",
				view, a.ShowGoToLine, a.GoToLineError, selectionPoint(t, tab))
		}
	})

	t.Run("successful flush can invalidate target", func(t *testing.T) {
		a, tab, _ := newSourceNavigationFixture(t, "one\ntwo")
		view := &navigationFlushView{flush: func() error {
			return tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), "short")
		}}
		tab.View = view
		if !a.openGoToLine() {
			t.Fatal("could not open source-line panel")
		}
		if a.submitGoToLine("2:1") {
			t.Fatal("post-flush invalid target unexpectedly navigated")
		}
		if view.flushes != 1 || view.releases != 0 || !a.ShowGoToLine ||
			!strings.Contains(a.GoToLineError, "文本输入已同步") ||
			strings.Contains(a.GoToLineError, "尚未同步") {
			t.Fatalf("post-flush failure disclosure is untruthful: flushes=%d releases=%d panel=%v error=%q",
				view.flushes, view.releases, a.ShowGoToLine, a.GoToLineError)
		}
	})

	t.Run("final coordinates use the flushed buffer", func(t *testing.T) {
		a, tab, _ := newSourceNavigationFixture(t, "old\nsecond")
		view := &navigationFlushView{flush: func() error {
			return tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), "inserted\none\ntwo")
		}}
		tab.View = view
		if !a.openGoToLine() {
			t.Fatal("could not open source-line panel")
		}
		if !a.submitGoToLine("2:3") {
			t.Fatalf("final-coordinate navigation failed: %s", a.GoToLineError)
		}
		line, column, err := tab.Doc.Buffer.LineColumn(selectionPoint(t, tab))
		if err != nil || line != 1 || column != 2 {
			t.Fatalf("final-coordinate target = line %d column %d err=%v", line, column, err)
		}
	})

	t.Run("replaced editor buffer invalidates target before flush", func(t *testing.T) {
		a, tab, _ := newSourceNavigationFixture(t, "one\ntwo")
		view := &navigationFlushView{flushErr: errors.New("should not run")}
		tab.View = view
		if !a.openGoToLine() {
			t.Fatal("could not open source-line panel")
		}
		tab.Editor.Buffer = text.New("replacement")
		if a.submitGoToLine("2:1") {
			t.Fatal("navigation accepted an editor that no longer owned the captured buffer")
		}
		if view.flushes != 0 || a.ShowGoToLine || a.goToLineBuffer != nil ||
			!strings.Contains(a.Message, "文档已变化") {
			t.Fatalf("replaced editor target = flushes=%d panel=%v captured=%v message=%q",
				view.flushes, a.ShowGoToLine, a.goToLineBuffer, a.Message)
		}
	})
}

func TestSourceNavigationPreservesSourceSplitAndConvertsPreviewAfterValidation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.md")
	if err := os.WriteFile(path, []byte("# 标题\n\n正文\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"source", "split", "preview"} {
		t.Run(mode, func(t *testing.T) {
			a := newTestApp(t, filepath.Join(root, "config-"+mode))
			a.Open(path)
			tab := a.Current()
			if mode != "preview" && !a.setMode(tab, mode) {
				t.Fatalf("could not enter %s: %s", mode, a.Message)
			}
			tester := ui.NewTester(a.View, 1180, 780)
			tester.Frame()
			view := tab.View
			a.persistPending = false
			if !a.openGoToLine() || !a.submitGoToLine("3:2") {
				t.Fatalf("%s navigation failed: %s", mode, a.GoToLineError)
			}
			if !a.persistPending {
				t.Fatalf("%s navigation did not request position persistence", mode)
			}
			line, column, err := tab.Doc.Buffer.LineColumn(tab.Doc.Buffer.Selection().Head)
			if err != nil || line != 2 || column != 1 {
				t.Fatalf("%s target = line %d column %d err=%v", mode, line, column, err)
			}
			if mode == "preview" {
				if tab.Mode != "source" {
					t.Fatalf("successful preview navigation left mode %q", tab.Mode)
				}
			} else if tab.Mode != mode || mode == "split" && tab.View != view {
				t.Fatalf("%s navigation changed mode or presentation: mode=%q same-view=%v",
					mode, tab.Mode, tab.View == view)
			}
		})
	}
}

func TestSourceNavigationReadonlyAndDisplayedByteView(t *testing.T) {
	t.Run("readonly source", func(t *testing.T) {
		a := newTestApp(t, filepath.Join(t.TempDir(), "config"))
		d := document.New("甲🙂\n乙")
		d.ReadOnly = true
		tab := a.appendTab(d, "readonly.txt", "", false, false)
		a.Active = 0
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		if !a.openGoToLine() || !a.submitGoToLine("2:2") {
			t.Fatalf("readonly navigation failed: %s", a.GoToLineError)
		}
		line, column, _ := d.Buffer.LineColumn(d.Buffer.Selection().Head)
		if tab.Mode != "source" || line != 1 || column != 1 {
			t.Fatalf("readonly target = mode %q line %d column %d", tab.Mode, line, column)
		}
	})

	t.Run("byte view", func(t *testing.T) {
		a := newTestApp(t, filepath.Join(t.TempDir(), "config"))
		d := document.New(hex.Dump([]byte("0123456789abcdef")))
		d.Path = filepath.Join(t.TempDir(), "sample.bin")
		d.ReadOnly = true
		d.IsNew = false
		tab := a.appendTab(d, "sample.bin", "只读字节视图", true, false)
		a.Active = 0
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		if !a.openGoToLine() {
			t.Fatal("could not open byte-view navigation")
		}
		tester.Frame()
		if !tester.HasText("十六进制字节视图的显示行") {
			t.Fatalf("byte-view help is not explicit: %q", tester.Texts())
		}
		if !tester.HasText("省略 :列 时默认为第 1 列") || !tester.HasText("最多 32 个 UTF-16 代码单元") {
			t.Fatalf("byte-view help omits the grammar or query limit: %q", tester.Texts())
		}
		if !tester.HasText("不接受前导 0") {
			t.Fatalf("byte-view help omits the leading-zero rule: %q", tester.Texts())
		}
		if !a.submitGoToLine("1:1") || tab.Mode != "source" || selectionPoint(t, tab) != 0 {
			t.Fatalf("byte-view navigation failed: mode=%q error=%q", tab.Mode, a.GoToLineError)
		}
	})
}

func TestSourceNavigationLifecycleInvalidationAndFailedSwitchRetention(t *testing.T) {
	a, tab, _ := newSourceNavigationFixture(t, "first")
	a.NewDocument("text")
	second := a.Current()
	if !a.Select(0) {
		t.Fatal("could not return to first tab")
	}
	failing := &failingView{flushErr: errors.New("switch refused")}
	tab.View = failing
	if !a.openGoToLine() {
		t.Fatal("could not open source-line panel")
	}
	if a.Select(1) || a.Active != 0 || !a.ShowGoToLine || a.goToLineTab != tab {
		t.Fatal("failed tab switch lost the source-line panel or changed the active tab")
	}
	if a.CloseTab(0) || !a.ShowGoToLine || a.goToLineTab != tab {
		t.Fatal("failed close lost the source-line panel or removed the tab")
	}

	tab.View = nil
	if !a.Select(1) || a.ShowGoToLine || a.goToLineTab != nil || a.goToLineBuffer != nil {
		t.Fatal("successful tab switch did not clear source-line target references")
	}
	if !a.Select(0) || !a.openGoToLine() {
		t.Fatal("could not reopen for committed removal")
	}
	a.removeTab(1)
	if a.ShowGoToLine || a.goToLineTab != nil || a.goToLineDoc != nil ||
		a.goToLineEditor != nil || a.goToLineBuffer != nil {
		t.Fatal("committed removal retained source-line target references")
	}

	if !a.openGoToLine() {
		t.Fatal("could not reopen for reconfiguration")
	}
	a.configureTab(tab)
	if a.ShowGoToLine || a.goToLineBuffer != nil {
		t.Fatal("tab reconfiguration retained source-line target references")
	}
	_ = second
}

func TestSourceNavigationReloadAndPluginReconfigureInvalidateCapturedTarget(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.txt")
	if err := os.WriteFile(path, []byte("one\ntwo\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if !a.openGoToLine() {
		t.Fatal("could not open source-line panel")
	}
	a.Reload()
	if a.ShowGoToLine || a.goToLineTab != nil || a.goToLineBuffer != nil || tab.Doc.Buffer == nil {
		t.Fatal("reload retained stale source-line target")
	}
	if !a.openGoToLine() {
		t.Fatal("could not reopen after reload")
	}
	a.reconfigureTabs()
	if a.ShowGoToLine || a.goToLineBuffer != nil {
		t.Fatal("plugin reconfiguration retained stale source-line target")
	}
}

func TestSourceNavigationLateLineAndSegmentedLongLineTail(t *testing.T) {
	a, tab, _ := newSourceNavigationFixture(t, strings.Repeat("x\n", 12000)+"tail")
	if !a.openGoToLine() || !a.submitGoToLine("12001:5") {
		t.Fatalf("late-line navigation failed: %s", a.GoToLineError)
	}
	line, column, err := tab.Doc.Buffer.LineColumn(tab.Doc.Buffer.Selection().Head)
	if err != nil || line != 12000 || column != 4 {
		t.Fatalf("late-line target = line %d column %d err=%v", line, column, err)
	}

	long := strings.Repeat("y", 12000)
	if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), long); err != nil {
		t.Fatal(err)
	}
	tab.Editor.ShapeLimit = 64
	if !a.openGoToLine() || !a.submitGoToLine("1:12001") {
		t.Fatalf("segmented long-line tail navigation failed: %s", a.GoToLineError)
	}
	line, column, err = tab.Doc.Buffer.LineColumn(tab.Doc.Buffer.Selection().Head)
	if err != nil || line != 0 || column != 12000 {
		t.Fatalf("segmented long-line target = line %d column %d err=%v", line, column, err)
	}
}

func TestSourceNavigationNarrowErrorCapture(t *testing.T) {
	out := os.Getenv("LEVNOTE_SOURCE_NAVIGATION_SCREENSHOT_DIR")
	if out == "" {
		t.Skip("set LEVNOTE_SOURCE_NAVIGATION_SCREENSHOT_DIR for visual acceptance")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	for _, dark := range []bool{false, true} {
		name := "source-navigation-error-760x480-light.png"
		if dark {
			name = "source-navigation-error-760x480-dark.png"
		}
		t.Run(name, func(t *testing.T) {
			a, _, _ := newSourceNavigationFixture(t, "first\nsecond")
			a.Dark = dark
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			if !a.openGoToLine() {
				t.Fatal("could not open source-line panel")
			}
			tester.Frame()
			tester.Type("0")
			tester.Frame()
			if a.submitGoToLine("0") {
				t.Fatal("invalid capture target unexpectedly navigated")
			}
			tester.Frame()
			if !strings.Contains(a.GoToLineError, "正整数") {
				t.Fatalf("capture fixture has no local error: %q", a.GoToLineError)
			}
			if !tester.HasText("转到行号错误") {
				t.Fatalf("capture has no accessible local-error label: %q", tester.Texts())
			}
			const statusTop = float32(480 - 29)
			for _, label := range []string{"转到行号", "转到源码行", "取消转到行号", "转到行号错误"} {
				rect, ok := tester.Find(label)
				if !ok || rect.X < a.SidebarWidth || rect.X+rect.W > 760 ||
					rect.Y < 0 || rect.Y+rect.H > statusTop {
					t.Fatalf("narrow source-line control %q is missing or crosses the main/status bounds: rect=%+v ok=%v main-left=%v",
						label, rect, ok, a.SidebarWidth)
				}
			}
			captureAcceptanceTester(t, filepath.Join(out, name), tester)
		})
	}
}
