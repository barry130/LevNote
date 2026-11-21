package app

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
)

func TestLanguageDefinitionCommandUsesPhysicalUTF16Navigation(t *testing.T) {
	const source = "package p\r\n//line virtual.go:900\r\nfunc Solve(变量 int) int {\r\n return 变量\r\n}\r\n"
	a, tab, queue, _ := languageFixture(t, ".go", source)
	definitionStart := strings.Index(source, "Solve")
	caret := text.UTF16Len(source[:definitionStart+len("Sol")])
	if err := tab.Doc.Buffer.SetSelection(caret, caret); err != nil {
		t.Fatal(err)
	}
	before := languageState(tab)
	generationBeforeNavigation := tab.Doc.Buffer.SelectionGeneration()
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	a.runSourceLanguageCommand("definition")
	if !a.sourceLanguage.visible || a.sourceLanguage.task == nil || a.sourceLanguage.intent.operation != code.ServiceDefinition {
		t.Fatalf("definition command did not admit the real request: %+v", a.sourceLanguage)
	}
	task := a.sourceLanguage.task
	languageWait(t, task.done)
	languageReceive(t, queue.calls)()
	tester.Frame()
	result := a.sourceLanguage.result
	if result.Status != code.ServiceComplete || result.Definition == nil || result.Definition.Name != "Solve" {
		t.Fatalf("real Analyzer definition result missing: %+v", result)
	}
	line, column, err := tab.Doc.Buffer.LineColumn(result.Definition.Span.UTF16.Start)
	if err != nil || line != 2 || column != 5 {
		t.Fatalf("virtual //line directive leaked into physical host coordinates: line=%d column=%d err=%v", line, column, err)
	}
	rows := a.sourceLanguageRows()
	if len(rows) != 1 || rows[0].position != (sourceLanguagePosition{line: 3, column: 6}) {
		t.Fatalf("physical definition position was not retained in the accessible row: %+v", rows)
	}
	rowLabel := rows[0].label
	if _, ok := tester.Find("语言结果：" + rowLabel); !ok {
		t.Fatalf("physical definition row is not accessible at %s: %v", rowLabel, tester.Texts())
	}
	if err := tester.Click("语言结果：" + rowLabel); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if a.sourceLanguage.selected != 0 {
		t.Fatalf("pointer selection did not update the virtual result list: selected=%d", a.sourceLanguage.selected)
	}
	if err := tester.Click("转到语言结果"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	selection := tab.Doc.Buffer.Selection()
	if selection.Anchor != result.Definition.Span.UTF16.Start || selection.Head != result.Definition.Span.UTF16.End ||
		a.sourceLanguage.message != "已定位到第 3 行第 6 UTF-16 列" {
		t.Fatalf("checked result activation selected wrong physical UTF-16 span: selection=%+v message=%q span=%+v", selection, a.sourceLanguage.message, result.Definition.Span)
	}
	if tab.Doc.Buffer.SelectionGeneration() != generationBeforeNavigation+1 {
		t.Fatalf("one successful checked navigation must make exactly one selection change: before=%d after=%d", generationBeforeNavigation, tab.Doc.Buffer.SelectionGeneration())
	}
	if !tester.Focused(tab.Editor.Label) || tab.Doc.Buffer.String() != source || tab.Doc.Buffer.Version() != before.version || tab.Doc.Buffer.History() != before.history {
		t.Fatal("read-only navigation changed content/history or failed to focus the source editor")
	}
	if err := tab.Doc.Buffer.SetSelectionSet(before.selections); err != nil {
		t.Fatal(err)
	}
	if tab.Doc.Buffer.SelectionGeneration() != generationBeforeNavigation+2 {
		t.Fatalf("restoring full test selection should account for exactly one additional generation: %d", tab.Doc.Buffer.SelectionGeneration())
	}
	expected := before
	expected.selectionGeneration = generationBeforeNavigation + 2
	languageUnchanged(t, tab, expected)

	// The public list focus/Enter route also activates the checked result row.
	a.sourceLanguage.list.Handle.Focus()
	tester.Frame()
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	if tab.Doc.Buffer.Selection().Anchor != result.Definition.Span.UTF16.Start || tab.Doc.Buffer.Selection().Head != result.Definition.Span.UTF16.End ||
		tab.Doc.Buffer.SelectionGeneration() != generationBeforeNavigation+3 || !tester.Focused(tab.Editor.Label) {
		t.Fatalf("keyboard list submit did not perform one checked navigation: selection=%+v generation=%d", tab.Doc.Buffer.Selection(), tab.Doc.Buffer.SelectionGeneration())
	}
	if err := tab.Doc.Buffer.SetSelectionSet(before.selections); err != nil {
		t.Fatal(err)
	}
	if tab.Doc.Buffer.SelectionGeneration() != generationBeforeNavigation+4 {
		t.Fatalf("second explicit test restoration did not increment selection generation once: %d", tab.Doc.Buffer.SelectionGeneration())
	}
	expected.selectionGeneration = generationBeforeNavigation + 4
	languageUnchanged(t, tab, expected)
}

func TestLanguageCheckedNavigationPreservesReentrantCancellation(t *testing.T) {
	source := "package p\n" + strings.Repeat("// retained filler\n", 120) + "func Solve(变量 int) int { return 变量 }\n"
	a, tab, queue, _ := languageFixture(t, ".go", source)
	byteAt := strings.Index(source, "Solve")
	at := text.UTF16Len(source[:byteAt])
	if err := tab.Doc.Buffer.SetSelection(at, at); err != nil {
		t.Fatal(err)
	}
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	a.runSourceLanguageCommand("definition")
	task := a.sourceLanguage.task
	if task == nil {
		t.Fatal("definition request did not start")
	}
	languageWait(t, task.done)
	languageReceive(t, queue.calls)()
	tester.Frame()
	rows := a.sourceLanguageRows()
	if len(rows) != 1 || rows[0].kind != "definition" {
		t.Fatalf("expected one real definition row, got %+v", rows)
	}
	row := rows[0]
	before := tab.Doc.Buffer.SelectionSet()
	beforeSource, beforeVersion, beforeHistory := tab.Doc.Buffer.String(), tab.Doc.Buffer.Version(), tab.Doc.Buffer.History()
	called := false
	previousHighlight := tab.Editor.Highlight
	if previousHighlight == nil {
		t.Fatal("real Go route did not install its production highlighter")
	}
	tab.Editor.Highlight = func(line string) []ui.Span {
		if !called {
			called = true
			a.cancelSourceLanguage()
		}
		return previousHighlight(line)
	}
	tab.Editor.SetFontSize(tab.Editor.FontSize + 1) // cold line geometry forces the real callback during checked reveal
	a.queueSourceLanguageAction("navigate", row.id)
	a.processSourceLanguageActions()
	if !called {
		t.Fatal("navigation did not exercise the real Editor highlight callback")
	}
	if a.sourceLanguage.status != string(code.ServiceCanceled) || a.sourceLanguage.reason != "canceled-by-host" || a.sourceLanguage.owner != nil {
		t.Fatalf("reentrant successor state was overwritten: %+v", a.sourceLanguage)
	}
	if got := tab.Doc.Buffer.SelectionSet(); got.Primary != before.Primary || len(got.Ranges) != len(before.Ranges) || got.Ranges[0] != before.Ranges[0] {
		t.Fatalf("stale navigation committed a selection after callback cancellation: before=%+v after=%+v", before, got)
	}
	if tab.Doc.Buffer.String() != beforeSource || tab.Doc.Buffer.Version() != beforeVersion || tab.Doc.Buffer.History() != beforeHistory {
		t.Fatal("reentrant read-only navigation changed source/history")
	}
}
