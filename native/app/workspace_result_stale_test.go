package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

// Exercise real disk search and public editor/list input, not synthetic results
// or a direct call to the private activation method.
func workspaceDiskResultFixture(t *testing.T) (*App, *Tab, *ui.Tester, chan func(), string) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "search.txt")
	if err := os.WriteFile(path, []byte("zero\nneedle\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, t.TempDir())
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Doc.Buffer.String() != "zero\nneedle\n" {
		t.Fatal("fixture did not open the literal disk document")
	}
	a.Root = root
	queue := make(chan func(), 64)
	a.uiDispatch = func(fn func()) { queue <- fn }
	tester := ui.NewTester(a.View, 1180, 780)
	if err := tester.Click("查找"); err != nil {
		t.Fatal(err)
	}
	if err := tester.Click("查找内容"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	tester.Type("needle")
	if err := tester.Click("搜索工作区"); err != nil {
		t.Fatal(err)
	}
	drainListUIUntil(t, queue, func() bool { return !a.Searching })
	tester.Frame()
	tester.Frame()
	if len(a.SearchResults) != 1 || a.SearchResults[0].Start != 5 || a.SearchResults[0].End != 11 ||
		!sameNativeFilePath(a.SearchResults[0].Path, tab.Doc.Path) {
		t.Fatalf("real search must find the literal disk range 5..11: %#v", a.SearchResults)
	}
	return a, tab, tester, queue, path
}

func submitWorkspaceDiskResult(t *testing.T, a *App, tab *Tab, tester *ui.Tester, queue chan func()) {
	t.Helper()
	clickTesterText(t, tester, "search.txt:2  needle")
	before, message, active := tab.Doc.Buffer.Selection(), a.Message, a.Current()
	tester.Key(0, ui.KeyEnter)
	// Drain until any visible outcome or navigation, not until the expected
	// GREEN selection. The old asynchronous wrong jump must finish this driver.
	drainListUIUntil(t, queue, func() bool {
		return a.Message != message || tab.Doc.Buffer.Selection() != before || a.Current() != active
	})
	tester.Frame()
}

func TestWorkspaceDiskResultEditedTabEnterKeepsDraft(t *testing.T) {
	a, tab, tester, queue, path := workspaceDiskResultFixture(t)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	tester.Type("xxxxx\nzero\nneedle\n")
	tester.Frame()
	doc, input, buffer := tab.Doc, tab.Editor, tab.Doc.Buffer
	version, history := buffer.Version(), buffer.History()
	encoding, bom, readOnly := doc.Encoding, doc.BOM, doc.ReadOnly
	if buffer.String() != "xxxxx\nzero\nneedle\n" || buffer.Selection() != (text.Range{Anchor: 18, Head: 18}) || !doc.Dirty() {
		t.Fatalf("public typing did not prepare the literal draft: text=%q selection=%+v dirty=%v", buffer.String(), buffer.Selection(), doc.Dirty())
	}
	submitWorkspaceDiskResult(t, a, tab, tester, queue)
	if buffer.Selection() != (text.Range{Anchor: 18, Head: 18}) ||
		!tester.HasText("工作区结果来自磁盘；文档已有未保存修改，未跳转") {
		selected, _ := buffer.SliceUTF16(buffer.Selection().Start(), buffer.Selection().End())
		t.Fatalf("stale disk range must fail visibly and retain 18..18: selection=%+v selected=%q message=%q", buffer.Selection(), selected, a.Message)
	}
	if a.Current() != tab || tab.Doc != doc || tab.Editor != input || doc.Buffer != buffer || input.Buffer != buffer ||
		buffer.String() != "xxxxx\nzero\nneedle\n" || buffer.Version() != version || buffer.History() != history ||
		doc.Encoding != encoding || doc.BOM != bom || doc.ReadOnly != readOnly || !doc.Dirty() || tab.Mode != "source" {
		t.Fatal("stale navigation changed the live draft, owner, history or metadata")
	}
	bytes, err := os.ReadFile(path)
	if err != nil || string(bytes) != "zero\nneedle\n" {
		t.Fatalf("stale navigation changed disk bytes: %q, %v", bytes, err)
	}
}

// A successful public Flusher can synchronize a draft. This is a host contract
// fixture, not a claim about built-in concurrency or physical input methods.
func TestWorkspaceDiskResultPendingFlushKeepsSynchronizedDraft(t *testing.T) {
	a, tab, tester, queue, path := workspaceDiskResultFixture(t)
	doc, input, buffer := tab.Doc, tab.Editor, tab.Doc.Buffer
	encoding, bom := doc.Encoding, doc.BOM
	var flushedHistory text.HistoryStatus
	view := &admissionView{onFlush: func() error {
		if err := buffer.ReplaceUTF16(0, 0, "xxxxx\n"); err != nil {
			return err
		}
		if err := buffer.SetSelection(18, 18); err != nil {
			return err
		}
		flushedHistory = buffer.History()
		return nil
	}}
	tab.View, tab.Mode = view, "preview"
	tester.Frame()
	if doc.Dirty() || buffer.Version() != 0 {
		t.Fatal("pending view must start with clean disk-authoritative source at version zero")
	}
	submitWorkspaceDiskResult(t, a, tab, tester, queue)
	if buffer.String() != "xxxxx\nzero\nneedle\n" || buffer.Selection() != (text.Range{Anchor: 18, Head: 18}) ||
		buffer.Version() != 1 || buffer.History() != flushedHistory || flushedHistory.UndoDepth != 1 || !doc.Dirty() ||
		!tester.HasText("工作区结果来自磁盘；文档已有未保存修改，未跳转") {
		t.Fatalf("pending Flush must retain its synchronized edit without old-offset reveal: text=%q selection=%+v history=%+v message=%q",
			buffer.String(), buffer.Selection(), buffer.History(), a.Message)
	}
	if a.Current() != tab || tab.Doc != doc || tab.Editor != input || doc.Buffer != buffer || input.Buffer != buffer ||
		view.flushes != 1 || view.releases != 0 || tab.View != view || tab.Mode != "preview" || doc.Encoding != encoding || doc.BOM != bom {
		t.Fatalf("pending admission released or replaced the live presentation: flush=%d releases=%d mode=%q", view.flushes, view.releases, tab.Mode)
	}
	bytes, err := os.ReadFile(path)
	if err != nil || string(bytes) != "zero\nneedle\n" {
		t.Fatalf("pending navigation changed disk bytes: %q, %v", bytes, err)
	}
	if !buffer.Undo() || buffer.String() != "zero\nneedle\n" || !buffer.Redo() || buffer.String() != "xxxxx\nzero\nneedle\n" {
		t.Fatal("synchronized draft lost its undo/redo contents")
	}
}

func TestWorkspaceDiskResultCleanOpenTabRevealsNeedle(t *testing.T) {
	a, tab, tester, queue, _ := workspaceDiskResultFixture(t)
	doc, input, buffer := tab.Doc, tab.Editor, tab.Doc.Buffer
	version, history := buffer.Version(), buffer.History()
	view := &admissionView{}
	tab.View, tab.Mode = view, "preview"
	tester.Frame()
	submitWorkspaceDiskResult(t, a, tab, tester, queue)
	selected, err := buffer.SliceUTF16(5, 11)
	if err != nil || selected != "needle" || buffer.Selection() != (text.Range{Anchor: 5, Head: 11}) ||
		a.Current() != tab || tab.Doc != doc || tab.Editor != input || doc.Buffer != buffer || input.Buffer != buffer ||
		buffer.String() != "zero\nneedle\n" || buffer.Version() != version || buffer.History() != history || doc.Dirty() ||
		view.flushes != 1 || view.releases != 1 || tab.View != view || tab.Mode != "source" {
		t.Fatalf("clean opened target should reveal exact needle once: selection=%+v flush=%d release=%d mode=%q message=%q",
			buffer.Selection(), view.flushes, view.releases, tab.Mode, a.Message)
	}
}

func openWorkspaceOrigin(t *testing.T, a *App, tester *ui.Tester, queue chan func()) *Tab {
	t.Helper()
	path := filepath.Join(t.TempDir(), "origin.txt")
	if err := os.WriteFile(path, []byte("origin\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a.Open(path)
	drainListUIUntil(t, queue, func() bool {
		current := a.Current()
		return current != nil && sameNativeFilePath(canonical(current.Doc.Path), canonical(path))
	})
	tester.Frame()
	origin := a.Current()
	if origin == nil || !sameNativeFilePath(canonical(origin.Doc.Path), canonical(path)) {
		t.Fatal("public Open did not activate the unrelated origin")
	}
	return origin
}

func TestWorkspaceDiskResultCleanInactiveTabFlushesOnce(t *testing.T) {
	a, tab, tester, queue, _ := workspaceDiskResultFixture(t)
	origin := openWorkspaceOrigin(t, a, tester, queue)
	originView, targetView := &admissionView{}, &admissionView{}
	origin.View, origin.Mode = originView, "preview"
	tab.View, tab.Mode = targetView, "preview"
	tester.Frame()
	submitWorkspaceDiskResult(t, a, tab, tester, queue)
	if a.Current() != tab || tab.Doc.Buffer.Selection() != (text.Range{Anchor: 5, Head: 11}) ||
		tab.Doc.Buffer.String() != "zero\nneedle\n" || tab.Doc.Dirty() || origin.Doc.Buffer.String() != "origin\n" || origin.Doc.Dirty() ||
		targetView.flushes != 1 || targetView.releases != 1 || originView.flushes != 1 || originView.releases != 1 || tab.Mode != "source" {
		t.Fatalf("clean inactive target/source must flush once before synchronous reveal: target=%d/%d origin=%d/%d selection=%+v message=%q",
			targetView.flushes, targetView.releases, originView.flushes, originView.releases, tab.Doc.Buffer.Selection(), a.Message)
	}
}

func TestWorkspaceDiskResultSourceFlushMakesTargetDirtyBeforeSelect(t *testing.T) {
	a, tab, tester, queue, path := workspaceDiskResultFixture(t)
	origin := openWorkspaceOrigin(t, a, tester, queue)
	buffer := tab.Doc.Buffer
	targetView := &admissionView{}
	originView := &admissionView{onFlush: func() error {
		if err := buffer.ReplaceUTF16(0, 0, "xxxxx\n"); err != nil {
			return err
		}
		return buffer.SetSelection(18, 18)
	}}
	origin.View, origin.Mode = originView, "preview"
	tab.View, tab.Mode = targetView, "preview"
	tester.Frame()
	submitWorkspaceDiskResult(t, a, tab, tester, queue)
	if a.Current() != origin || buffer.String() != "xxxxx\nzero\nneedle\n" || buffer.Selection() != (text.Range{Anchor: 18, Head: 18}) ||
		buffer.Version() != 1 || buffer.History().UndoDepth != 1 || !tab.Doc.Dirty() ||
		originView.flushes != 1 || targetView.flushes != 1 || originView.releases != 0 || targetView.releases != 0 ||
		origin.View != originView || tab.View != targetView || !tester.HasText("工作区结果来自磁盘；文档已有未保存修改，未跳转") {
		t.Fatalf("source Flush boundary must recheck before switching/releasing: current=%v selection=%+v target=%d/%d origin=%d/%d message=%q",
			a.Current() == origin, buffer.Selection(), targetView.flushes, targetView.releases, originView.flushes, originView.releases, a.Message)
	}
	bytes, err := os.ReadFile(path)
	if err != nil || string(bytes) != "zero\nneedle\n" {
		t.Fatalf("source Flush contract changed disk: %q, %v", bytes, err)
	}
}

func TestWorkspaceDiskResultFlushFailureKeepsPendingView(t *testing.T) {
	a, tab, tester, queue, _ := workspaceDiskResultFixture(t)
	buffer := tab.Doc.Buffer
	selection, version, history := buffer.Selection(), buffer.Version(), buffer.History()
	view := &admissionView{onFlush: func() error { return os.ErrPermission }}
	tab.View, tab.Mode = view, "preview"
	tester.Frame()
	submitWorkspaceDiskResult(t, a, tab, tester, queue)
	if a.Current() != tab || tab.View != view || tab.Mode != "preview" || view.flushes != 1 || view.releases != 0 ||
		buffer.Selection() != selection || buffer.Version() != version || buffer.History() != history ||
		buffer.String() != "zero\nneedle\n" || tab.Doc.Dirty() || !tester.HasText("工作区搜索定位前未能同步格式编辑：permission denied") {
		t.Fatalf("failed Flush must keep draft and disclose its error: selection=%+v flush=%d release=%d message=%q",
			buffer.Selection(), view.flushes, view.releases, a.Message)
	}
}

func TestWorkspaceDiskResultDirtyInactiveTabStopsBeforeFlush(t *testing.T) {
	a, tab, tester, queue, _ := workspaceDiskResultFixture(t)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	tester.Type("xxxxx\nzero\nneedle\n")
	tester.Frame()
	origin := openWorkspaceOrigin(t, a, tester, queue)
	originView, targetView := &admissionView{}, &admissionView{}
	origin.View, origin.Mode = originView, "preview"
	tab.View, tab.Mode = targetView, "preview"
	buffer := tab.Doc.Buffer
	version, history := buffer.Version(), buffer.History()
	tester.Frame()
	submitWorkspaceDiskResult(t, a, tab, tester, queue)
	if a.Current() != origin || buffer.Selection() != (text.Range{Anchor: 18, Head: 18}) ||
		buffer.String() != "xxxxx\nzero\nneedle\n" || buffer.Version() != version || buffer.History() != history || !tab.Doc.Dirty() ||
		targetView.flushes != 0 || targetView.releases != 0 || originView.flushes != 0 || originView.releases != 0 ||
		tab.View != targetView || origin.View != originView || !tester.HasText("工作区结果来自磁盘；文档已有未保存修改，未跳转") {
		t.Fatalf("already-dirty inactive target should not Flush or Select either presentation: target=%d/%d origin=%d/%d message=%q",
			targetView.flushes, targetView.releases, originView.flushes, originView.releases, a.Message)
	}
}

func TestWorkspaceDiskResultUnopenedFileKeepsOpenAtPath(t *testing.T) {
	a, old, tester, queue, path := workspaceDiskResultFixture(t)
	if !a.CloseTab(0) || len(a.Tabs) != 0 {
		t.Fatal("public CloseTab did not prepare an unopened result target")
	}
	tester.Frame()
	submitWorkspaceDiskResult(t, a, old, tester, queue)
	tab := a.Current()
	if tab == nil || tab == old || len(a.Tabs) != 1 || !sameNativeFilePath(canonical(tab.Doc.Path), canonical(path)) ||
		tab.Doc.Buffer.String() != "zero\nneedle\n" || tab.Doc.Buffer.Selection() != (text.Range{Anchor: 5, Head: 11}) || tab.Doc.Dirty() || tab.Mode != "source" {
		t.Fatalf("unopened result must retain the existing OpenAt load/reveal path: current=%v message=%q", tab != nil, a.Message)
	}
}

func TestWorkspaceDiskResultSuccessfulFlushRechecksCapturedTarget(t *testing.T) {
	cases := []struct {
		name    string
		change  func(*App, *Tab) error
		message string
	}{
		{"search-generation", func(a *App, _ *Tab) error { a.searchID++; return nil }, "工作区搜索结果已变化；未跳转"},
		{"root", func(a *App, _ *Tab) error { a.Root = filepath.Join(a.Root, "other"); return nil }, "工作区搜索结果已变化；未跳转"},
		{"buffer-owner", func(_ *App, tab *Tab) error {
			tab.Doc.Buffer = text.New("zero\nneedle\n")
			tab.Editor.Buffer = tab.Doc.Buffer
			return nil
		}, "工作区搜索目标已变化；未跳转"},
		{"clean-new-version", func(_ *App, tab *Tab) error {
			if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, "x"); err != nil {
				return err
			}
			return tab.Doc.Buffer.ReplaceUTF16(0, 1, "")
		}, "同步输入期间文档已变化；工作区结果未跳转"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, tab, tester, queue, path := workspaceDiskResultFixture(t)
			var selection text.Range
			var version uint64
			var history text.HistoryStatus
			view := &admissionView{onFlush: func() error {
				if err := tc.change(a, tab); err != nil {
					return err
				}
				selection, version, history = tab.Doc.Buffer.Selection(), tab.Doc.Buffer.Version(), tab.Doc.Buffer.History()
				return nil
			}}
			tab.View, tab.Mode = view, "preview"
			tester.Frame()
			submitWorkspaceDiskResult(t, a, tab, tester, queue)
			if a.Current() != tab || tab.View != view || tab.Mode != "preview" || view.flushes != 1 || view.releases != 0 ||
				tab.Doc.Buffer.String() != "zero\nneedle\n" || tab.Doc.Buffer.Selection() != selection ||
				tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.History() != history || tab.Doc.Dirty() || !tester.HasText(tc.message) {
				t.Fatalf("successful Flush must recheck %s and preserve its effective state: selection=%+v version=%d history=%+v message=%q",
					tc.name, tab.Doc.Buffer.Selection(), tab.Doc.Buffer.Version(), tab.Doc.Buffer.History(), a.Message)
			}
			if tc.name == "clean-new-version" && (version != 2 || history.UndoDepth != 2) {
				t.Fatalf("net-zero edits must still expose the independent two-edit version/history premise: version=%d history=%+v", version, history)
			}
			bytes, err := os.ReadFile(path)
			if err != nil || string(bytes) != "zero\nneedle\n" {
				t.Fatalf("successful Flush contract changed disk: %q, %v", bytes, err)
			}
		})
	}
}
