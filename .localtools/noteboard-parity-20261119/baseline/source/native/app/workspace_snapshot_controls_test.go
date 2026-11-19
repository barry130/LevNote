package app

import (
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/search"
	"levnote.local/native/core/text"
)

// The virtualization fixture still controls its row count and labels, but its
// actionable row now carries the same evidence as a real workspace result.
func workspaceSnapshotMatch(t *testing.T, root, query, path string) search.FileMatch {
	t.Helper()
	var matches []search.FileMatch
	_, err := search.Workspace(t.Context(), root, search.WorkspaceOptions{Options: search.Options{Query: query}}, func(batch search.Batch) {
		for _, match := range batch.Matches {
			if sameNativeFilePath(match.Path, path) {
				matches = append(matches, match)
			}
		}
	})
	if err != nil || len(matches) != 1 {
		t.Fatalf("real workspace fixture: matches=%+v error=%v", matches, err)
	}
	return matches[0]
}

func TestWorkspaceSnapshotSameLengthSavedContentAndResearch(t *testing.T) {
	a, tab, tester, queue, path := workspaceDiskResultFixture(t)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	tester.Type("xxxx\nneedle\n")
	if !a.Save(tab, false) || tab.Doc.Dirty() {
		t.Fatalf("save changed same-length source: %q", a.Message)
	}
	buffer := tab.Doc.Buffer
	version, history := buffer.Version(), buffer.History()
	submitWorkspaceDiskResult(t, a, tab, tester, queue)
	if buffer.Selection() != (text.Range{Anchor: 12, Head: 12}) ||
		!tester.HasText("工作区搜索结果对应的内容已变化；请重新搜索，未跳转") ||
		buffer.String() != "xxxx\nneedle\n" || buffer.Version() != version || buffer.History() != history || tab.Doc.Dirty() {
		t.Fatalf("same query and offsets must not authenticate different surrounding content: selection=%+v message=%q", buffer.Selection(), a.Message)
	}
	if err := tester.Click("搜索工作区"); err != nil {
		t.Fatal(err)
	}
	drainListUIUntil(t, queue, func() bool { return !a.Searching })
	tester.Frame()
	submitWorkspaceDiskResult(t, a, tab, tester, queue)
	if buffer.Selection() != (text.Range{Anchor: 5, Head: 11}) || buffer.String() != "xxxx\nneedle\n" ||
		buffer.Version() != version || buffer.History() != history || tab.Doc.Dirty() {
		t.Fatalf("a new real search should reveal unchanged content: selection=%+v message=%q", buffer.Selection(), a.Message)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "xxxx\nneedle\n" {
		t.Fatalf("navigation changed saved bytes: %q, %v", data, err)
	}
}

func TestWorkspaceSnapshotUnchangedResultIgnoresLaterQuery(t *testing.T) {
	a, tab, tester, queue, _ := workspaceDiskResultFixture(t)
	if err := tester.Click("查找内容"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	tester.Type("not-in-this-file")
	if a.Query != "not-in-this-file" {
		t.Fatal("public query edit was not applied")
	}
	buffer := tab.Doc.Buffer
	version, history := buffer.Version(), buffer.History()
	submitWorkspaceDiskResult(t, a, tab, tester, queue)
	if buffer.Selection() != (text.Range{Anchor: 5, Head: 11}) || buffer.String() != "zero\nneedle\n" ||
		buffer.Version() != version || buffer.History() != history || tab.Doc.Dirty() {
		t.Fatalf("captured unchanged result should navigate independently of current query: selection=%+v message=%q", buffer.Selection(), a.Message)
	}
}

func TestWorkspaceSnapshotMissingIdentityNeverNavigates(t *testing.T) {
	for _, unopened := range []bool{false, true} {
		name := "opened"
		if unopened {
			name = "unopened"
		}
		t.Run(name, func(t *testing.T) {
			a, tab, tester, queue, _ := workspaceDiskResultFixture(t)
			if unopened && !a.CloseTab(0) {
				t.Fatal("CloseTab failed")
			}
			current, count := a.Current(), len(a.Tabs)
			buffer := tab.Doc.Buffer
			selection, version, history := buffer.Selection(), buffer.Version(), buffer.History()
			match := a.SearchResults[0]
			a.SearchResults[0] = search.FileMatch{Path: match.Path, Line: match.Line, Column: match.Column,
				Start: match.Start, End: match.End, Preview: match.Preview}
			tester.Frame()
			submitWorkspaceDiskResult(t, a, tab, tester, queue)
			if a.Current() != current || len(a.Tabs) != count || buffer.Selection() != selection ||
				buffer.Version() != version || buffer.History() != history || buffer.String() != "zero\nneedle\n" ||
				!tester.HasText("工作区搜索结果缺少内容快照；请重新搜索，未跳转") {
				t.Fatalf("missing identity must not borrow fresh content: current=%v tabs=%d message=%q", a.Current() == current, len(a.Tabs), a.Message)
			}
		})
	}
}

func TestWorkspaceSnapshotSavedDuringSourceFlushKeepsTarget(t *testing.T) {
	a, tab, tester, queue, path := workspaceDiskResultFixture(t)
	origin := openWorkspaceOrigin(t, a, tester, queue)
	buffer := tab.Doc.Buffer
	var savedVersion uint64
	var savedHistory text.HistoryStatus
	originView := &admissionView{onFlush: func() error {
		if err := buffer.ReplaceUTF16(0, 4, "xxxx"); err != nil {
			return err
		}
		if err := buffer.SetSelection(12, 12); err != nil {
			return err
		}
		if err := tab.Doc.Save(); err != nil {
			return err
		}
		savedVersion, savedHistory = buffer.Version(), buffer.History()
		return nil
	}}
	targetView := &admissionView{}
	origin.View, origin.Mode = originView, "preview"
	tab.View, tab.Mode = targetView, "preview"
	tester.Frame()
	submitWorkspaceDiskResult(t, a, tab, tester, queue)
	if a.Current() != origin || tab.Doc.Dirty() || buffer.Selection() != (text.Range{Anchor: 12, Head: 12}) ||
		buffer.String() != "xxxx\nneedle\n" || buffer.Version() != savedVersion || buffer.History() != savedHistory ||
		originView.flushes != 1 || originView.releases != 0 || targetView.flushes != 1 || targetView.releases != 0 ||
		!tester.HasText("工作区搜索结果对应的内容已变化；请重新搜索，未跳转") {
		t.Fatalf("saved-clean change across source Flush must stop before Select: selection=%+v origin=%v message=%q", buffer.Selection(), a.Current() == origin, a.Message)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "xxxx\nneedle\n" {
		t.Fatalf("synchronized save bytes changed: %q, %v", data, err)
	}
}

func TestWorkspaceSnapshotUnopenedSearchChangesDuringOriginFlush(t *testing.T) {
	a, old, tester, queue, _ := workspaceDiskResultFixture(t)
	if !a.CloseTab(0) {
		t.Fatal("CloseTab failed")
	}
	origin := openWorkspaceOrigin(t, a, tester, queue)
	buffer := origin.Doc.Buffer
	selection, version, history := buffer.Selection(), buffer.Version(), buffer.History()
	view := &admissionView{onFlush: func() error {
		// Start a real newer search through the public UI, while the loaded
		// old request is about to adopt. The old search generation must lose.
		return tester.Click("搜索工作区")
	}}
	origin.View, origin.Mode = view, "preview"
	tester.Frame()
	clickTesterText(t, tester, "search.txt:2  needle")
	tester.Key(0, ui.KeyEnter)
	drainListUIUntil(t, queue, func() bool { return a.Message == "工作区搜索结果已变化；未跳转" })
	// Join the new search before finishing the fixture.
	drainListUIUntil(t, queue, func() bool { return !a.Searching })
	tester.Frame()
	if a.Current() != origin || len(a.Tabs) != 1 || view.flushes != 1 || view.releases != 0 ||
		buffer.String() != "origin\n" || buffer.Selection() != selection || buffer.Version() != version || buffer.History() != history ||
		!tester.HasText("工作区搜索结果已变化；未跳转") {
		t.Fatalf("obsolete async result must not adopt after Flush: tabs=%d message=%q", len(a.Tabs), a.Message)
	}
	_ = old
}

func TestWorkspaceSnapshotUnopenedUnchangedReportsOpen(t *testing.T) {
	a, old, tester, queue, path := workspaceDiskResultFixture(t)
	if !a.CloseTab(0) {
		t.Fatal("CloseTab failed")
	}
	origin := openWorkspaceOrigin(t, a, tester, queue)
	originBuffer := origin.Doc.Buffer
	selection, version, history := originBuffer.Selection(), originBuffer.Version(), originBuffer.History()
	submitWorkspaceDiskResult(t, a, old, tester, queue)
	current := a.Current()
	if current == nil || current == origin || current.Doc.Path != path || len(a.Tabs) != 2 ||
		current.Doc.Buffer.String() != "zero\nneedle\n" || current.Doc.Buffer.Selection() != (text.Range{Anchor: 5, Head: 11}) ||
		current.Doc.Dirty() || current.Doc.Buffer.Version() != 0 || current.Doc.Buffer.History().UndoDepth != 0 ||
		!tester.HasText("已打开 search.txt") {
		t.Fatalf("unchanged file should open, reveal and report its name: message=%q", a.Message)
	}
	if originBuffer.String() != "origin\n" || originBuffer.Selection() != selection || originBuffer.Version() != version || originBuffer.History() != history {
		t.Fatal("successful navigation edited the origin")
	}
}
