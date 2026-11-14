package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestTreeListHandleRootSwitchKeepsFreshKeyboardTarget(t *testing.T) {
	roots := []string{t.TempDir(), t.TempDir()}
	for rootIndex, root := range roots {
		for i := 0; i < 48; i++ {
			path := filepath.Join(root, fmt.Sprintf("root-%d-%02d.txt", rootIndex, i))
			if err := os.WriteFile(path, []byte(fmt.Sprintf("root %d row %d", rootIndex, i)), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	a := newTestApp(t, t.TempDir())
	queue := make(chan func(), 64)
	a.uiDispatch = func(fn func()) { queue <- fn }
	tester := ui.NewTester(a.View, 760, 480)
	var handle ui.Handle
	for pass := 0; pass < 4; pass++ {
		rootIndex := pass % len(roots)
		root := roots[rootIndex]
		a.openDialog = func(opts mygo.OpenDialogOptions) ([]string, error) {
			if !opts.Directory {
				t.Fatal("folder entry did not request a directory")
			}
			return []string{root}, nil
		}
		if err := tester.Click("文件夹"); err != nil {
			t.Fatal(err)
		}
		// Adopt the real scanner's newest root on the test/UI owner. No OS
		// watcher or live product window participates in this regression.
		drainListUIUntil(t, queue, func() bool { return len(a.Entries) == 48 })
		tester.Frame()
		tester.Frame()
		if pass == 0 {
			handle = a.TreeList.Handle
			if handle == (ui.Handle{}) {
				t.Fatal("nonempty tree did not bind its public handle")
			}
		} else if a.TreeList.Handle != handle {
			t.Fatal("root replacement abandoned the tree's bound handle")
		}
		first, _ := a.TreeList.Visible()
		if a.TreeSelected != -1 || first != 0 {
			t.Fatalf("new root retained selection/place: selected=%d first=%d", a.TreeSelected, first)
		}
		clickTesterText(t, tester, fmt.Sprintf("root-%d-00.txt", rootIndex))
		tester.Key(0, ui.KeyEnd)
		tester.Frame()
		tester.Frame()
		if a.TreeSelected != 47 {
			t.Fatalf("End selected %d, want the current root's last row", a.TreeSelected)
		}
		tester.Key(0, ui.KeyEnter)
		tester.Frame()
		want := filepath.Join(root, fmt.Sprintf("root-%d-47.txt", rootIndex))
		drainListUIUntil(t, queue, func() bool {
			return a.Current() != nil && strings.EqualFold(a.Current().Doc.Path, want)
		})
		if current := a.Current(); current == nil || !strings.EqualFold(current.Doc.Path, want) {
			t.Fatalf("Enter did not open the newest root's row: current=%#v want=%q", current, want)
		}
		selection := a.TreeSelected
		a.refreshTreeNow()
		tester.Frame()
		if a.TreeList.Handle != handle || a.TreeSelected != selection {
			t.Fatal("same-root scan changed the handle or selected item")
		}
	}
}

// This timer is only a deadlock guard; successful outcomes depend on UI events,
// never a delay or the wall-clock speed of the scanner/search/open worker.
func nextListUI(t *testing.T, queue <-chan func()) func() {
	t.Helper()
	guard := time.NewTimer(time.Minute)
	defer guard.Stop()
	select {
	case fn := <-queue:
		return fn
	case <-guard.C:
		t.Fatal("list workflow stopped delivering its next UI event")
		return nil
	}
}

func drainListUIUntil(t *testing.T, queue <-chan func(), done func() bool) {
	t.Helper()
	for !done() {
		nextListUI(t, queue)()
	}
}

func TestSearchListHandleNewSearchRejectsStaleDelivery(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "matches.txt")
	var content strings.Builder
	for _, prefix := range []string{"old", "new"} {
		for i := 0; i < 32; i++ {
			fmt.Fprintf(&content, "%s-%02d\n", prefix, i)
		}
	}
	original := content.String()
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, t.TempDir())
	a.Open(path)
	tab := a.Current()
	version := tab.Doc.Buffer.Version()
	a.Root = root
	queue := make(chan func(), 64)
	a.uiDispatch = func(fn func()) { queue <- fn }
	tester := ui.NewTester(a.View, 1180, 780)
	if err := tester.Click("查找"); err != nil {
		t.Fatal(err)
	}
	start := func(query string) {
		t.Helper()
		if err := tester.Click("查找内容"); err != nil {
			t.Fatal(err)
		}
		tester.Command("selectAll")
		tester.Type(query)
		if err := tester.Click("搜索工作区"); err != nil {
			t.Fatal(err)
		}
		if !a.Searching {
			t.Fatal("public Find action did not admit the search")
		}
	}
	finish := func() {
		t.Helper()
		drainListUIUntil(t, queue, func() bool { return !a.Searching })
		tester.Frame()
		tester.Frame()
	}
	start("old")
	finish()
	if len(a.SearchResults) != 32 {
		t.Fatalf("real old search returned %d results", len(a.SearchResults))
	}
	handle := a.SearchList.Handle
	if handle == (ui.Handle{}) {
		t.Fatal("nonempty workspace results did not bind a handle")
	}
	a.Root = t.TempDir()
	a.refreshTreeNow()
	tester.Frame()
	if !tester.HasText("工作区已变化，请重新搜索。") || a.SearchList.Label(0) != "" || len(a.SearchResults) != 32 {
		t.Fatal("old-root results stayed actionable or were silently erased")
	}
	a.Root = root
	a.refreshTreeNow()
	tester.Frame()
	tester.Frame()
	clickTesterText(t, tester, "matches.txt:1  old-00")
	tester.Key(0, ui.KeyEnd)
	tester.Frame()
	if a.SearchSelected != 31 {
		t.Fatalf("old search End selected %d", a.SearchSelected)
	}

	start("old")
	stale := nextListUI(t, queue) // Keep a real previous search's batch undelivered.
	oldID := a.searchID
	start("new")
	finish()
	if a.searchID <= oldID || len(a.SearchResults) != 32 {
		t.Fatal("replacement search did not own the current result set")
	}
	currentSummary := a.SearchSummary
	stale()
	if len(a.SearchResults) != 32 || a.SearchSummary != currentSummary {
		t.Fatal("stale search delivery changed the replacement results/summary")
	}
	for _, result := range a.SearchResults {
		if !strings.HasPrefix(result.Preview, "new-") {
			t.Fatalf("stale result survived query replacement: %#v", result)
		}
	}
	first, _ := a.SearchList.Visible()
	if a.SearchList.Handle != handle || a.SearchSelected != -1 || first != 0 {
		t.Fatalf("new search did not reuse/reset: selected=%d first=%d", a.SearchSelected, first)
	}
	clickTesterText(t, tester, "matches.txt:33  new-00")
	tester.Key(0, ui.KeyEnd)
	tester.Frame()
	tester.Frame()
	if a.SearchSelected != 31 {
		t.Fatalf("replacement End selected %d", a.SearchSelected)
	}
	result := a.SearchResults[a.SearchSelected]
	tester.Key(0, ui.KeyEnter)
	drainListUIUntil(t, queue, func() bool {
		selection := tab.Doc.Buffer.Selection()
		return selection.Start() == result.Start && selection.End() == result.End
	})
	tester.Frame()
	if a.Current() != tab || tab.Doc.Buffer.String() != original || tab.Doc.Buffer.Version() != version {
		t.Fatal("result navigation changed owner, bytes or edit history")
	}
	beforeFirst, beforeLast := a.SearchList.Visible()
	if err := tester.Click("收起结果"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if a.ShowWorkspaceSearch || len(a.SearchResults) != 32 || a.SearchSelected != 31 {
		t.Fatal("collapse cleared live search results/selection")
	}
	a.ShowWorkspaceSearch = true // Existing panel visibility seam, not a new search.
	tester.Frame()
	tester.Frame()
	afterFirst, afterLast := a.SearchList.Visible()
	if a.SearchList.Handle != handle || afterFirst != beforeFirst || afterLast != beforeLast {
		t.Fatal("collapse/reopen changed the live results' handle or scroll place")
	}

	start("no-such-match")
	finish()
	if len(a.SearchResults) != 0 || a.SearchSelected != -1 || a.SearchList.Handle != handle ||
		a.SearchList.Key != nil || a.SearchList.Label != nil || a.SearchList.Selected != nil {
		t.Fatal("empty replacement retained old callbacks/choice or abandoned handle")
	}
	start("new")
	finish()
	if a.SearchList.Handle != handle || len(a.SearchResults) != 32 || a.SearchSelected != -1 {
		t.Fatal("nonempty/empty/nonempty replacement lost its stable fresh state")
	}
	if tab.Doc.Buffer.String() != original || tab.Doc.Buffer.Version() != version {
		t.Fatal("search/replacement/collapse changed the source document")
	}
}
