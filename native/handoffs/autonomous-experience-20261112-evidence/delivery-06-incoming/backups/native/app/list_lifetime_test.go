package app

import (
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/search"
	"levnote.local/native/ui/editor"
)

func TestAppListHandlesSurviveSemanticResets(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	lists := []struct {
		name     string
		state    *ui.ListState
		lifetime *appListLifetime
		selected *int
	}{
		{"tree", &a.TreeList, &a.treeListLifetime, &a.TreeSelected},
		{"search", &a.SearchList, &a.searchListLifetime, &a.SearchSelected},
		{"outline", &a.OutlineList, &a.outlineListLifetime, &a.OutlineSelected},
	}
	var handles [3]ui.Handle
	expectDrain := false
	tester := ui.NewTester(func(c *ui.Context) {
		a.beginAppListFrame()
		ui.Row(c).Fill().Children(func() {
			for _, list := range lists {
				if expectDrain && (list.state.Key != nil || list.state.Label != nil || list.state.Selected != nil) {
					t.Fatalf("%s retained obsolete public callbacks at the pre-build boundary", list.name)
				}
				list.state.Selected = list.selected
				list.state.Key = func(i int) any { return i }
				list.state.Label = func(i int) string { return fmt.Sprintf("%s-row-%03d", list.name, i) }
				list.lifetime.built = true
				ui.List(c.Key(list.name), list.state, 100, func(i int) {
					ui.Row(c).Height(28).Children(func() { ui.Text(c, fmt.Sprintf("%s-row-%03d", list.name, i)) })
				}).Grow(1)
			}
		})
		expectDrain = false
	}, 760, 480)
	tester.Frame()
	for i, list := range lists {
		handles[i] = list.state.Handle
		if handles[i] == (ui.Handle{}) {
			t.Fatalf("%s did not bind", list.name)
		}
	}
	for pass := 0; pass < 12; pass++ {
		for i, list := range lists {
			*list.selected = -1
			list.lifetime.reset(list.state)
			if list.state.Handle != handles[i] || list.state.Key == nil || list.state.Label == nil || list.state.Selected == nil {
				t.Fatalf("%s destroyed its already-built state before the next boundary", list.name)
			}
			list.lifetime.reveal(list.state, 99)
		}
		// The new deliberate request follows cancellation, while the reset
		// is still pending. Draining must preserve this exact public Handle.
		target := pass % len(lists)
		lists[target].state.Handle.Focus()
		expectDrain = true
		tester.Frame()
		tester.Key(0, ui.KeyEnd)
		tester.Frame()
		for i, list := range lists {
			want := -1
			if i == target {
				want = 99
			}
			if list.state.Handle != handles[i] || *list.selected != want {
				t.Fatalf("%s pass %d: handle replaced or new focus lost, selected=%d want=%d", list.name, pass, *list.selected, want)
			}
		}
	}
	a.Shutdown()
	for i, list := range lists {
		if list.state.Handle != handles[i] || list.state.Key != nil || list.state.Label != nil || list.state.Selected != nil || *list.selected != -1 {
			t.Fatalf("%s shutdown retained application-owned callbacks or abandoned the bound handle", list.name)
		}
	}
}

func listOutlineSource(tab *Tab, prefix string) *outlineSourceStub {
	source := &outlineSourceStub{}
	source.snapshotFn = func() (editor.OutlineSnapshot, error) {
		items := make([]editor.OutlineItem, 64)
		for i := range items {
			items[i] = editor.OutlineItem{ID: fmt.Sprintf("%s-%02d", prefix, i), Text: fmt.Sprintf("%s-%02d", prefix, i), Level: 1, Offset: i}
		}
		return editor.OutlineSnapshot{Version: tab.Doc.Buffer.Version(), Items: items}, nil
	}
	source.resolveItemFn = func(id string) (editor.OutlineItem, error) {
		var row int
		if _, err := fmt.Sscanf(id, prefix+"-%02d", &row); err != nil || row < 0 || row >= 64 {
			return editor.OutlineItem{}, editor.ErrOutlineStale
		}
		return editor.OutlineItem{ID: id, Text: id, Level: 1, Offset: row}, nil
	}
	return source
}

func TestOutlineListHandleReleaseAfterSidebarBuild(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := newOutlineTab(a, strings.Repeat("x", 128), "late.md")
	tab.Mode = "source"
	old := listOutlineSource(tab, "old")
	fresh := listOutlineSource(tab, "fresh")
	calls := 0
	installSourceOutlineFactory(tab, old, &calls)
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	handle := a.OutlineList.Handle
	if handle == (ui.Handle{}) || len(a.OutlineItems) != 64 {
		t.Fatal("outline did not bind before the late-layout transition")
	}
	before, version, history := tab.Doc.Buffer.String(), tab.Doc.Buffer.Version(), tab.Doc.Buffer.History()
	replaced := false
	a.outlineRowBuilt = func(int) {
		if replaced {
			return
		}
		replaced = true
		a.releaseOutline()
		installSourceOutlineFactory(tab, fresh, &calls)
		tab.OutlineFactoryGeneration++
		// This request follows the release while the old list is still built.
		a.OutlineList.Handle.Focus()
	}
	tester.Frame()
	tester.Frame()
	if !replaced || old.releases != 1 || len(old.itemInputs) != 0 || a.OutlineList.Handle != handle ||
		len(a.OutlineItems) != 64 || a.OutlineItems[0].ID != "fresh-00" || a.OutlineSelected != -1 {
		t.Fatalf("obsolete outline survived the after-build transition: old releases=%d resolves=%v selected=%d items=%v",
			old.releases, old.itemInputs, a.OutlineSelected, a.OutlineItems)
	}
	tester.Key(0, ui.KeyEnd)
	tester.Frame()
	if a.OutlineSelected != 63 {
		t.Fatal("new deliberate focus was lost while draining the released list")
	}
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	if len(fresh.itemInputs) != 1 || fresh.itemInputs[0] != "fresh-63" || tab.Doc.Buffer.Selection().Head != 63 {
		t.Fatalf("Enter did not target the replacement controller: resolves=%v selection=%v", fresh.itemInputs, tab.Doc.Buffer.Selection())
	}
	if tab.Doc.Buffer.String() != before || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.History() != history {
		t.Fatal("outline lifecycle/navigation changed source bytes or history")
	}
}

func TestOutlineListHandleReentrantFlushAndRetry(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := newOutlineTab(a, strings.Repeat("x", 128), "retry.md")
	tab.Mode = "preview"
	source := listOutlineSource(tab, "heading")
	_, view := installOutlineStub(a, tab, source)
	tab.View = view
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	handle := a.OutlineList.Handle
	before, version, history := tab.Doc.Buffer.String(), tab.Doc.Buffer.Version(), tab.Doc.Buffer.History()
	clickTesterText(t, tester, "heading-00")
	view.onFlush = func() {
		a.releaseOutline()
		view.onFlush = nil
	}
	tester.Key(0, ui.KeyEnter)
	if len(source.itemInputs) != 0 || len(view.navigations) != 0 || !strings.Contains(a.Message, editor.ErrOutlineStale.Error()) {
		t.Fatal("a released controller received navigation after reentrant Flush")
	}
	tester.Frame()
	goodSnapshot := source.snapshotFn
	source.snapshotFn = func() (editor.OutlineSnapshot, error) { return editor.OutlineSnapshot{}, errors.New("retry fixture") }
	if err := a.ensureOutlineSnapshot(tab, "", true); err == nil {
		t.Fatal("snapshot failure was not surfaced")
	}
	tester.Frame()
	if a.OutlineList.Handle != handle || a.OutlineSelected != -1 || a.OutlineList.Key != nil ||
		a.OutlineList.Label != nil || a.OutlineList.Selected != nil || !tester.HasText("读取大纲失败：retry fixture") {
		t.Fatal("error retained obsolete row callbacks/selection or abandoned its handle")
	}
	source.snapshotFn = goodSnapshot
	if err := tester.Click("重试大纲"); err != nil {
		t.Fatal(err)
	}
	a.OutlineList.Handle.Focus()
	tester.Frame()
	tester.Key(0, ui.KeyEnd)
	tester.Frame()
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	if a.OutlineList.Handle != handle || a.OutlineSelected != 63 || len(view.navigations) != 1 || view.navigations[0] != "heading-63" {
		t.Fatalf("retry did not restore fresh keyboard navigation: selected=%d navigation=%v", a.OutlineSelected, view.navigations)
	}
	// A successful same-controller reparse keeps the selected stable ID,
	// even when its row moves. This is not a semantic reset.
	source.snapshotFn = func() (editor.OutlineSnapshot, error) {
		snapshot, err := goodSnapshot()
		for i, j := 0, len(snapshot.Items)-1; i < j; i, j = i+1, j-1 {
			snapshot.Items[i], snapshot.Items[j] = snapshot.Items[j], snapshot.Items[i]
		}
		return snapshot, err
	}
	if err := a.ensureOutlineSnapshot(tab, "", true); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if a.OutlineList.Handle != handle || a.OutlineSelected != 0 || a.OutlineItems[0].ID != "heading-63" {
		t.Fatal("same-controller reparse lost the selected stable ID")
	}
	source.snapshotFn = func() (editor.OutlineSnapshot, error) { return editor.OutlineSnapshot{Version: tab.Doc.Buffer.Version()}, nil }
	if err := a.ensureOutlineSnapshot(tab, "", true); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if a.OutlineList.Handle != handle || a.OutlineList.Key != nil || a.OutlineList.Label != nil || a.OutlineList.Selected != nil {
		t.Fatal("empty outline retained obsolete callbacks or lost its bound handle")
	}
	source.snapshotFn = goodSnapshot
	if err := a.ensureOutlineSnapshot(tab, "", true); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if a.OutlineList.Handle != handle || len(a.OutlineItems) != 64 || a.OutlineSelected != -1 {
		t.Fatal("empty/nonempty outline did not restore fresh stable state")
	}
	if tab.Doc.Buffer.String() != before || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.History() != history {
		t.Fatal("error/retry/reparse changed source bytes or committed history")
	}
}

func TestHiddenListFocusDoesNotStealEditorInput(t *testing.T) {
	for _, kind := range []string{"tree", "search", "outline"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			for i := 0; i < 64; i++ {
				if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("file-%02d.txt", i)), []byte("needle"), 0600); err != nil { t.Fatal(err) }
			}
			a, tab := newSearchTextApp(t, "needle " + strings.Repeat("x", 128))
			a.Root = root
			a.refreshTreeNow()
			factoryCalls := 0
			installSourceOutlineFactory(tab, listOutlineSource(tab, "focus"), &factoryCalls)
			a.ShowWorkspaceSearch, a.searchRoot = true, root
			for i := 0; i < 32; i++ {
				a.SearchResults = append(a.SearchResults, search.FileMatch{Path: filepath.Join(root, "file-00.txt"), Line: i+1, Start: i, End: i+1, Preview: fmt.Sprintf("result-%02d", i)})
			}
			tester := ui.NewTester(a.View, 1180, 780)
			a.setOutlineVisible(true)
			tester.Frame()
			if kind != "outline" { a.setOutlineVisible(false) }
			tester.Frame()
			state, selected, want := &a.OutlineList, &a.OutlineSelected, 63
			if kind == "tree" { state, selected = &a.TreeList, &a.TreeSelected }
			if kind == "search" { state, selected, want = &a.SearchList, &a.SearchSelected, 31 }
			handle := state.Handle
			if handle == (ui.Handle{}) { t.Fatal("focus fixture did not bind") }
			if err := tester.Click("文档编辑区"); err != nil { t.Fatal(err) }
			before, version := tab.Doc.Buffer.String(), tab.Doc.Buffer.Version()
			handle.Focus()
			switch kind {
			case "tree": a.setOutlineVisible(true)
			case "search": a.ShowWorkspaceSearch = false
			case "outline": a.setOutlineVisible(false)
			}
			tester.Frame()
			tester.Frame()
			if !tester.Focused("文档编辑区") { t.Fatal("a hidden list stole the source input focus") }
			tester.Type("猫")
			if strings.Count(tab.Doc.Buffer.String(), "猫") != 1 || strings.ReplaceAll(tab.Doc.Buffer.String(), "猫", "") != before ||
				tab.Doc.Buffer.Version() == version || a.Query != "" {
				t.Fatal("typed source input was stolen, duplicated or routed into Find")
			}
			switch kind {
			case "tree": a.setOutlineVisible(false)
			case "search": a.ShowWorkspaceSearch = true
			case "outline": a.setOutlineVisible(true)
			}
			handle.Focus()
			tester.Frame()
			tester.Key(0, ui.KeyEnd)
			tester.Frame()
			if state.Handle != handle || *selected != want { t.Fatalf("later deliberate %s focus did not target the fresh list: selected=%d", kind, *selected) }
		})
	}
}

func TestTreeRootSwitchKeepsComposingQueryInput(t *testing.T) {
	roots := []string{t.TempDir(), t.TempDir()}
	for group, root := range roots {
		for i := 0; i < 2; i++ {
			if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("root-%d-%d.txt", group, i)), []byte("needle"), 0600); err != nil { t.Fatal(err) }
		}
	}
	a, tab := newSearchTextApp(t, "needle needle")
	a.Root, a.Query, a.ShowFind = roots[0], "needle", true
	a.refreshTreeNow()
	queue := make(chan func(), 64)
	a.uiDispatch = func(fn func()) { queue <- fn }
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	handle := a.TreeList.Handle
	if handle == (ui.Handle{}) { t.Fatal("root fixture did not bind the tree") }
	if err := tester.Click("查找内容"); err != nil { t.Fatal(err) }
	tester.Compose("猫", 1)
	preedit, ok := tester.AccessByRole(ui.RoleTextField, "查找内容")
	if !ok || !preedit.Focused || !a.FindQueryComposing { t.Fatal("public query preedit did not initialize") }
	before, version, history, selection := tab.Doc.Buffer.String(), tab.Doc.Buffer.Version(), tab.Doc.Buffer.History(), tab.Doc.Buffer.Selection()
	handle.Focus()
	a.openDialog = func(opts mygo.OpenDialogOptions) ([]string, error) {
		if !opts.Directory { t.Fatal("root replacement did not use the folder contract") }
		return []string{roots[1]}, nil
	}
	// ChooseFolder is the app-owned root-transition boundary; a native OS
	// dialog and physical IME are intentionally outside public Tester evidence.
	a.ChooseFolder()
	drainListUIUntil(t, queue, func() bool { return len(a.Entries) == 2 })
	tester.Frame()
	tester.Frame()
	after, ok := tester.AccessByRole(ui.RoleTextField, "查找内容")
	if !ok || !after.Focused || after.Value != preedit.Value || a.Query != "needle" || !a.FindQueryComposing ||
		a.TreeList.Handle != handle || a.Root != roots[1] || a.TreeSelected != -1 {
		t.Fatalf("root transition stole focus, committed preedit or reused obsolete choice: input=%#v query=%q composing=%t selected=%d", after, a.Query, a.FindQueryComposing, a.TreeSelected)
	}
	tester.Key(0, ui.KeyEnter)
	if tab.Doc.Buffer.Selection() != selection || a.Searching || a.searchID != 0 { t.Fatal("uncommitted query input submitted a document/workspace action") }
	tester.Command("selectAll")
	tester.Type("updated")
	if a.Query != "updated" || a.FindQueryComposing { t.Fatal("query did not accept the later deliberate committed input") }
	handle.Focus()
	tester.Frame()
	tester.Key(0, ui.KeyEnd)
	tester.Frame()
	if a.TreeSelected != 1 || a.TreeList.Handle != handle || !strings.HasPrefix(a.Entries[1].Name, "root-1-") { t.Fatal("later tree focus did not select the new root's row") }
	if tab.Doc.Buffer.String() != before || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.History() != history { t.Fatal("root/query focus flow modified document bytes or history") }
}

func TestWorkspaceResultsMinimumWindowControls(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprintf("dark=%t", dark), func(t *testing.T) {
			a := newTestApp(t, t.TempDir())
			a.Dark = dark
			a.NewDocument("text")
			a.ShowFind, a.ShowWorkspaceSearch = true, true
			a.SearchSummary = strings.Repeat("123456 个匹配 · 长路径详情 ", 30)
			a.SearchIssues = []search.FileIssue{{Path: strings.Repeat("long-path/", 20), Reason: "fixture skip"}}
			a.SearchResults = []search.FileMatch{{Path: strings.Repeat("very-long-directory/", 20)+"match.txt", Line: 1, Start: 0, End: 1, Preview: strings.Repeat("内容 ", 200)}}
			cancelled := 0
			a.searchCancel = func() { cancelled++ }
			a.Searching = true
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			var rects []ui.Rect
			for _, label := range []string{"取消搜索", "查看跳过详情", "收起结果"} {
				info, ok := tester.Access(label)
				if !ok || info.Bounds.W <= 0 || info.Bounds.H <= 0 || info.Bounds.X < 1+a.SidebarWidth ||
					info.Bounds.X+info.Bounds.W > 760 || info.Bounds.Y < 0 || info.Bounds.Y+info.Bounds.H > 451 {
					t.Fatalf("minimum-window control %q is missing or clipped: %#v", label, info)
				}
				for _, other := range rects {
					if other.X < info.Bounds.X+info.Bounds.W && info.Bounds.X < other.X+other.W && other.Y < info.Bounds.Y+info.Bounds.H && info.Bounds.Y < other.Y+other.H { t.Fatalf("minimum-window control %q overlaps: %#v %#v", label, info.Bounds, other) }
				}
				rects = append(rects, info.Bounds)
			}
			if source, ok := tester.Find("文档编辑区"); !ok || source.W < 200 || source.H < 64 { t.Fatalf("results+Find leave no usable source input: %#v", source) }
			if err := tester.Click("取消搜索"); err != nil { t.Fatal(err) }
			if cancelled != 1 { t.Fatal("visible cancel control did not invoke the current cancellation") }
			a.Searching = false
			tester.Frame()
			if err := tester.Click("查看跳过详情"); err != nil { t.Fatal(err) }
			if !a.ShowIssues { t.Fatal("visible details control did not expose skip diagnostics") }
			a.ShowIssues = false
			tester.Frame()
			if err := tester.Click("收起结果"); err != nil { t.Fatal(err) }
			if a.ShowWorkspaceSearch { t.Fatal("visible collapse control did not hide results") }
			// The public render is development evidence, not physical OS UI/IME.
			a.ShowWorkspaceSearch = true
			tester.Frame()
			image := tester.Image()
			path := filepath.Join(t.TempDir(), "workspace-results-minimum.png")
			file, err := os.Create(path)
			if err != nil { t.Fatal(err) }
			if err := png.Encode(file, image); err != nil { _ = file.Close(); t.Fatal(err) }
			if err := file.Close(); err != nil { t.Fatal(err) }
			t.Logf("public Tester development render: %s", path)
		})
	}
}
