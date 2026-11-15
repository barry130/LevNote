package app

import (
	"os"
	"testing"

	"levnote.local/native/core/text"
)

func TestWorkspaceSnapshotSavedCleanRetainsSelection(t *testing.T) {
	a, tab, tester, queue, path := workspaceDiskResultFixture(t)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	tester.Type("xxxxx\nzero\nneedle\n")
	tester.Frame()
	if !a.Save(tab, false) || tab.Doc.Dirty() {
		t.Fatalf("public Save did not publish a clean changed document: %q", a.Message)
	}
	doc, input, buffer := tab.Doc, tab.Editor, tab.Doc.Buffer
	version, history := buffer.Version(), buffer.History()
	encoding, bom, readOnly := doc.Encoding, doc.BOM, doc.ReadOnly
	if buffer.String() != "xxxxx\nzero\nneedle\n" || buffer.Selection() != (text.Range{Anchor: 18, Head: 18}) {
		t.Fatalf("saved source premise: text=%q selection=%+v", buffer.String(), buffer.Selection())
	}
	submitWorkspaceDiskResult(t, a, tab, tester, queue)
	if buffer.Selection() != (text.Range{Anchor: 18, Head: 18}) ||
		!tester.HasText("工作区搜索结果对应的内容已变化；请重新搜索，未跳转") {
		selected, _ := buffer.SliceUTF16(buffer.Selection().Start(), buffer.Selection().End())
		t.Fatalf("saved-clean stale result must retain 18..18 and report changed content: selection=%+v selected=%q message=%q", buffer.Selection(), selected, a.Message)
	}
	if a.Current() != tab || tab.Doc != doc || tab.Editor != input || doc.Buffer != buffer || input.Buffer != buffer ||
		buffer.String() != "xxxxx\nzero\nneedle\n" || buffer.Version() != version || buffer.History() != history ||
		doc.Encoding != encoding || doc.BOM != bom || doc.ReadOnly != readOnly || doc.Dirty() || tab.Mode != "source" {
		t.Fatal("stale result changed source owner, bytes, history or metadata")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "xxxxx\nzero\nneedle\n" {
		t.Fatalf("saved disk bytes changed: %q, %v", data, err)
	}
	if !buffer.Undo() || buffer.String() != "zero\nneedle\n" ||
		!buffer.Redo() || buffer.String() != "xxxxx\nzero\nneedle\n" {
		t.Fatal("saved document lost the original public editing undo/redo chain")
	}
}

func TestWorkspaceSnapshotUnopenedChangedFileKeepsOrigin(t *testing.T) {
	a, old, tester, queue, path := workspaceDiskResultFixture(t)
	if !a.CloseTab(0) {
		t.Fatal("CloseTab did not close the search target")
	}
	origin := openWorkspaceOrigin(t, a, tester, queue)
	buffer := origin.Doc.Buffer
	selection, version, history := buffer.Selection(), buffer.Version(), buffer.History()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("xxxx\nneedle\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	submitWorkspaceDiskResult(t, a, old, tester, queue)
	if a.Current() != origin || len(a.Tabs) != 1 ||
		!tester.HasText("工作区搜索结果对应的内容已变化；请重新搜索，未跳转") {
		t.Fatalf("unopened changed source must stay unadopted: origin=%v tabs=%d message=%q", a.Current() == origin, len(a.Tabs), a.Message)
	}
	if buffer.String() != "origin\n" || buffer.Selection() != selection || buffer.Version() != version || buffer.History() != history || origin.Doc.Dirty() {
		t.Fatal("stale unopened result changed the original tab")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "xxxx\nneedle\n" {
		t.Fatalf("external bytes changed: %q, %v", data, err)
	}
}
