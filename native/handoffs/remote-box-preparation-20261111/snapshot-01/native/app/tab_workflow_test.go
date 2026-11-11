package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
)

func TestTabKeyboardNoOpWithFewerThanTwoTabs(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		a := newTestApp(t, t.TempDir())
		a.ShowFind, a.ShowGoToLine = true, true
		a.FindQueryComposing, a.GoToLineComposing = true, true
		a.Query, a.GoToLineQuery = "query", "12:3"
		a.Message = "unchanged"
		a.persistPending = false
		for _, forward := range []bool{true, false} {
			if a.selectAdjacentTab(forward) || a.Active != -1 || a.Current() != nil ||
				!a.ShowFind || !a.ShowGoToLine || a.Query != "query" || a.GoToLineQuery != "12:3" || a.persistPending || a.Message != "unchanged" {
				t.Fatal("empty tab navigation changed state")
			}
		}
	})
	t.Run("single keeps panel and focus", func(t *testing.T) {
		a, tab := newSearchTextApp(t, "draft")
		tester := ui.NewTester(a.View, 1180, 780)
		if !a.openGoToLine() {
			t.Fatal("could not open GoToLine")
		}
		tester.Frame()
		view := &failingView{flushErr: errors.New("must not flush")}
		tab.View = view
		a.persistPending = false
		beforeHistory, beforeSelection := tab.Doc.Buffer.History(), tab.Doc.Buffer.Selection()
		for _, forward := range []bool{true, false} {
			if a.selectAdjacentTab(forward) || a.Active != 0 || a.Current() != tab ||
				!a.ShowGoToLine || a.goToLineTab != tab || a.persistPending || view.flushes != 0 || view.releases != 0 {
				t.Fatal("single tab navigation invoked Select lifecycle")
			}
			tester.Frame()
			if !tester.Focused("转到行号") || tab.Doc.Buffer.String() != "draft" ||
				tab.Doc.Buffer.History() != beforeHistory || tab.Doc.Buffer.Selection() != beforeSelection {
				t.Fatal("single tab navigation changed focus or draft")
			}
		}
	})
}

func TestTabKeyboardVisualOrderAndWrap(t *testing.T) {
	for _, count := range []int{2, 3} {
		t.Run(fmt.Sprintf("%d tabs", count), func(t *testing.T) {
			a := newTestApp(t, t.TempDir())
			for i := 0; i < count; i++ {
				a.add(document.New(fmt.Sprintf("draft %d", i)), fmt.Sprintf("note-%d.txt", i), "")
			}
			original := append([]*Tab(nil), a.Tabs...)
			for _, forward := range []bool{true, false} {
				for start := 0; start < count; start++ {
					if !a.Select(start) {
						t.Fatal("could not select fixture tab")
					}
					want := (start + 1) % count
					if !forward {
						want = (start + count - 1) % count
					}
					if !a.selectAdjacentTab(forward) || a.Active != want || a.Current() != original[want] {
						t.Fatalf("forward=%v start=%d active=%d want=%d", forward, start, a.Active, want)
					}
					if !sameTabPointers(a.Tabs, original) {
						t.Fatal("navigation reordered tabs")
					}
				}
			}
		})
	}
}

func TestTabKeyboardFlushBarrierAndSuccessfulLifecycle(t *testing.T) {
	a, first := newSearchTextApp(t, "needle draft")
	second := a.add(document.New("second draft"), "second.txt", "")
	if !a.Select(0) {
		t.Fatal("could not select first tab")
	}
	a.ShowFind, a.Query = true, "needle"
	a.find(true)
	if a.findNavigation.tab != first || len(a.findNavigation.matches) != 1 || !a.openGoToLine() {
		t.Fatal("missing Find cache or GoToLine fixture")
	}
	view := &failingView{flushErr: errors.New("draft conflict"), cacheLoaded: true}
	first.View = view
	beforeCache := a.findNavigation
	beforeHistory, beforeSelection, beforeVersion := first.Doc.Buffer.History(), first.Doc.Buffer.Selection(), first.Doc.Buffer.Version()
	cancels := 0
	done := make(chan struct{})
	close(done)
	job := &exportJob{tab: first, cancel: func() { cancels++ }, done: done}
	a.exportJob = job
	a.persistPending = false
	for _, forward := range []bool{true, false} {
		if a.selectAdjacentTab(forward) || a.Active != 0 || a.Current() != first || first.View != view ||
			view.releases != 0 || !view.cacheLoaded || cancels != 0 || job.invalid != "" || a.persistPending {
			t.Fatal("failed Flush changed active ownership, export or persistence")
		}
		if first.Doc.Buffer.String() != "needle draft" || first.Doc.Buffer.History() != beforeHistory ||
			first.Doc.Buffer.Selection() != beforeSelection || first.Doc.Buffer.Version() != beforeVersion ||
			!a.ShowFind || !a.ShowGoToLine || a.goToLineTab != first || !reflect.DeepEqual(a.findNavigation, beforeCache) {
			t.Fatal("failed Flush lost draft or navigation panel state")
		}
		if !strings.Contains(a.Message, "切换标签前未能同步格式编辑") || !strings.Contains(a.Message, "draft conflict") {
			t.Fatalf("failed Flush was not reported: %q", a.Message)
		}
	}
	view.flushErr = nil
	if !a.selectAdjacentTab(true) || a.Active != 1 || a.Current() != second ||
		view.flushes != 3 || view.releases != 1 || view.cacheLoaded || first.View != view {
		t.Fatal("successful switch did not flush/release the old presentation once")
	}
	if cancels != 1 || job.invalid != "切换标签，已取消导出" || a.exportGeneration != 1 ||
		!a.persistPending || a.findNavigation.tab != nil || a.FindIndex != -1 || !a.ShowFind ||
		a.ShowGoToLine || a.goToLineTab != nil || a.goToLineBuffer != nil {
		t.Fatal("successful switch did not preserve existing export/panel/persistence lifecycle")
	}
	a.exportJob = nil // The completed stub owns no runner; avoid a second cancellation during cleanup.
	tester := ui.NewTester(a.View, 1180, 780)
	if !tester.Focused("文档编辑区") {
		t.Fatal("successful switch did not focus the current source editor")
	}
	tester.Type("x")
	if second.Doc.Buffer.String() != "xsecond draft" || first.Doc.Buffer.String() != "needle draft" {
		t.Fatal("typing after the switch reached the wrong document")
	}
}

func TestTabKeyboardSyntheticRoutingAndSourceComposition(t *testing.T) {
	a, first := newSearchTextApp(t, "")
	second := a.add(document.New(""), "second.txt", "")
	if !a.Select(0) {
		t.Fatal("could not select first tab")
	}
	tester := ui.NewTester(a.View, 1180, 780)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Compose("中文🙂", 3)
	if _, composing := first.Editor.MarkedRange(); !composing {
		t.Fatal("fixture did not start source composition")
	}
	tester.Key(ui.Cmd, ui.KeyTab)
	if a.Current() != second || first.Doc.Buffer.String() != "中文🙂" || !tester.Focused("文档编辑区") {
		t.Fatalf("Ctrl+Tab did not commit and switch: active=%d source=%q", a.Active, first.Doc.Buffer.String())
	}
	if _, composing := first.Editor.MarkedRange(); composing {
		t.Fatal("tab switch left source composition active")
	}
	tester.Key(ui.Cmd|ui.Shift, ui.KeyTab)
	if a.Current() != first {
		t.Fatal("Ctrl+Shift+Tab did not return to previous tab")
	}
	for _, mods := range []ui.Modifiers{0, ui.Shift, ui.Cmd | ui.Alt, ui.Cmd | ui.Shift | ui.Alt} {
		tester.Key(mods, ui.KeyTab)
		if a.Current() != first {
			t.Fatalf("nonmatching modifiers %v switched tabs", mods)
		}
	}
}

func TestTabKeyboardSyntheticRoutingCommitsRichComposition(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	first := a.add(document.New("# 标题\n\n第二块\n"), "rich.md", "")
	second := a.add(document.New("second"), "second.txt", "")
	if !a.Select(0) || first.Mode != "preview" {
		t.Fatal("could not select rich fixture")
	}
	tester := ui.NewTester(a.View, 1180, 780)
	beginRealMarkdownComposition(t, first, tester)
	view := first.View
	tester.Key(ui.Cmd, ui.KeyTab)
	if a.Current() != second || first.View != view || !strings.Contains(first.Doc.Buffer.String(), "# 中文🙂") {
		t.Fatal("rich composition did not commit through the existing Select barrier")
	}
	tester.Key(ui.Cmd|ui.Shift, ui.KeyTab)
	if a.Current() != first || first.Mode != "preview" || first.View != view {
		t.Fatal("returning to rich tab discarded its retained view state")
	}
}

func TestTabKeyboardSyntheticRoutingFromComposingPanels(t *testing.T) {
	for _, panel := range []string{"query", "replacement", "goto"} {
		t.Run(panel, func(t *testing.T) {
			a, first := newSearchTextApp(t, "first")
			second := a.add(document.New("second"), "second.txt", "")
			if !a.Select(0) {
				t.Fatal("could not select first tab")
			}
			tester := ui.NewTester(a.View, 1180, 780)
			label, preedit := "转到行号", "12"
			if panel != "goto" {
				tester.Key(ui.Cmd, ui.KeyF)
				label, preedit = "查找内容", "needle"
				if panel == "replacement" {
					label = "替换内容"
					if err := tester.Click(label); err != nil {
						t.Fatal(err)
					}
				}
				tester.Compose("needle", 6)
				if !a.findCompositionActive() {
					t.Fatal("Find composition did not start")
				}
			} else {
				tester.Key(ui.Cmd, ui.KeyG)
				tester.Compose("12", 2)
				if !a.GoToLineComposing {
					t.Fatal("GoToLine composition did not start")
				}
			}
			query, replacement, gotoQuery := a.Query, a.Replacement, a.GoToLineQuery
			history, selection, version := first.Doc.Buffer.History(), first.Doc.Buffer.Selection(), first.Doc.Buffer.Version()
			view := &failingView{}
			first.View = view
			cancels := 0
			done := make(chan struct{})
			close(done)
			job := &exportJob{tab: first, cancel: func() { cancels++ }, done: done}
			a.exportJob = job
			tester.Frame() // Lay out the export-progress row before recording caret geometry.
			compositionCaret, compositionActive := tester.TextCaret()
			if !compositionActive {
				t.Fatal("composing field has no native input caret")
			}
			a.persistPending = false
			for _, mods := range []ui.Modifiers{ui.Cmd, ui.Cmd | ui.Shift} {
				tester.Key(mods, ui.KeyTab)
				if a.Current() != first || !tester.Focused(label) || a.persistPending || view.flushes != 0 || view.releases != 0 ||
					cancels != 0 || job.invalid != "" || a.exportGeneration != 0 ||
					(!a.findCompositionActive() && !a.GoToLineComposing) ||
					a.Query != query || a.Replacement != replacement || a.GoToLineQuery != gotoQuery ||
					first.Doc.Buffer.String() != "first" || first.Doc.Buffer.History() != history ||
					first.Doc.Buffer.Selection() != selection || first.Doc.Buffer.Version() != version {
					t.Fatal("composing panel shortcut changed focus, field, draft or lifecycle")
				}
				caret, active := tester.TextCaret()
				if !strings.Contains(a.Message, "请先完成组合后再切换标签") || !active || caret != compositionCaret {
					t.Fatalf("composition refusal moved the preedit caret or hid its explanation: message=%q caret=%+v/%+v", a.Message, caret, compositionCaret)
				}
			}
			a.exportJob = nil
			if panel == "goto" {
				tester.Type(preedit)
			} else {
				// A deliberate pointer focus change commits the actual surviving
				// Find preedit through MyGo's public input lifecycle.
				if err := tester.Click("文档编辑区"); err != nil {
					t.Fatal(err)
				}
			}
			tester.Frame()
			if a.Current() != first || a.findCompositionActive() || a.GoToLineComposing {
				t.Fatal("composition commit triggered an unsolicited switch or kept the guard")
			}
			tester.Key(ui.Cmd, ui.KeyTab)
			if a.Current() != second || !tester.Focused("文档编辑区") || a.ShowGoToLine ||
				first.Doc.Buffer.String() != "first" || second.Doc.Buffer.String() != "second" {
				t.Fatal("explicit retry after commit did not switch cleanly")
			}
			if panel == "query" && (!a.ShowFind || a.Query != preedit) ||
				panel == "replacement" && (!a.ShowFind || a.Replacement != preedit) {
				t.Fatal("switch lost the committed Find field value")
			}
			tester.Key(ui.Cmd|ui.Shift, ui.KeyTab)
			if a.Current() != first {
				t.Fatal("reverse shortcut did not return")
			}
		})
	}
}

func TestTabKeyboardCancelledAndHiddenPanelComposition(t *testing.T) {
	a, first := newSearchTextApp(t, "first")
	second := a.add(document.New("second"), "second.txt", "")
	if !a.Select(0) {
		t.Fatal("could not select first tab")
	}
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Key(ui.Cmd, ui.KeyF)
	tester.Compose("needle", 6)
	tester.Key(ui.Cmd, ui.KeyTab)
	tester.Compose("", 0)
	tester.Frame()
	if a.Current() != first || a.FindQueryComposing {
		t.Fatal("composition cancel queued a switch or left the guard")
	}
	tester.Key(ui.Cmd|ui.Shift, ui.KeyTab)
	if a.Current() != second {
		t.Fatal("explicit retry after cancellation did not switch")
	}
	a.closeFind()
	a.closeGoToLine()
	a.FindQueryComposing, a.FindReplacementComposing, a.GoToLineComposing = true, true, true
	if !a.selectAdjacentTab(true) || a.Current() != first {
		t.Fatal("stale hidden-panel flags blocked navigation")
	}
	a.ShowFind, a.ShowGoToLine, a.ShowPlugins = true, true, true
	if !a.selectAdjacentTab(false) || a.Current() != second {
		t.Fatal("fields hidden by the plugin panel blocked navigation")
	}
}

func TestTabKeyboardGoToLinePointerLeaveClearsComposition(t *testing.T) {
	for _, direction := range []struct {
		name string
		mods ui.Modifiers
		want int
	}{
		{"forward", ui.Cmd, 1},
		{"reverse", ui.Cmd | ui.Shift, 2},
	} {
		t.Run(direction.name, func(t *testing.T) {
			a, first := newSearchTextApp(t, "first draft")
			a.add(document.New("second draft"), "second.txt", "")
			a.add(document.New("third draft"), "third.txt", "")
			if !a.Select(0) {
				t.Fatal("could not select first tab")
			}
			tester := ui.NewTester(a.View, 1180, 780)
			beforeText, beforeHistory, beforeVersion := first.Doc.Buffer.String(), first.Doc.Buffer.History(), first.Doc.Buffer.Version()
			beforeSelection := first.Doc.Buffer.Selection()
			tester.Key(ui.Cmd, ui.KeyG)
			tester.Type("1")
			tester.Compose("2", 1)
			if !a.GoToLineComposing || !tester.Focused("转到行号") || a.GoToLineQuery != "1" {
				t.Fatalf("missing committed value and preedit fixture: query=%q composing=%v", a.GoToLineQuery, a.GoToLineComposing)
			}
			tester.Key(direction.mods, ui.KeyTab)
			if a.Current() != first || !a.ShowGoToLine || !a.GoToLineComposing || !tester.Focused("转到行号") ||
				a.GoToLineQuery != "1" || first.Doc.Buffer.Selection() != beforeSelection {
				t.Fatal("shortcut did not preserve the focused preedit")
			}
			if err := tester.Click("文档编辑区"); err != nil {
				t.Fatal(err)
			}
			tester.Frame()
			if a.Current() != first || !a.ShowGoToLine || a.GoToLineComposing ||
				!tester.Focused("文档编辑区") || a.GoToLineQuery != "12" {
				t.Fatalf("pointer leave did not commit surviving preedit and clear the guard without switching: active=%d query=%q composing=%v", a.Active, a.GoToLineQuery, a.GoToLineComposing)
			}
			pointerSelection := first.Doc.Buffer.Selection()
			tester.Key(direction.mods, ui.KeyTab)
			if a.Active != direction.want || a.ShowGoToLine || a.GoToLineComposing || !tester.Focused("文档编辑区") {
				t.Fatal("explicit shortcut after pointer leave did not switch")
			}
			if first.Doc.Buffer.String() != beforeText || first.Doc.Buffer.History() != beforeHistory ||
				first.Doc.Buffer.Version() != beforeVersion || first.Doc.Buffer.Selection() != pointerSelection ||
				a.Tabs[1].Doc.Buffer.String() != "second draft" || a.Tabs[2].Doc.Buffer.String() != "third draft" {
				t.Fatal("GoToLine preedit lifecycle changed a source document or its undo history")
			}
		})
	}
}

func newTabRevealFixture(t *testing.T, count int) (*App, *ui.Tester) {
	t.Helper()
	a := newTestApp(t, t.TempDir())
	for i := 0; i < count; i++ {
		a.add(document.New(fmt.Sprintf("document %d", i)), fmt.Sprintf("tab-%02d.txt", i), "")
	}
	a.closeDialog = func(*Tab) (closeDecision, error) { return closeDiscard, nil }
	return a, ui.NewTester(a.View, 760, 480)
}

func tabButtonLabel(tab *Tab) string {
	if !tab.Doc.ReadOnly && tab.Doc.Dirty() {
		return "• " + tab.Name
	}
	return tab.Name
}

func tabButtonRect(t *testing.T, tester *ui.Tester, tab *Tab) ui.Rect {
	t.Helper()
	r, ok := tester.Find(tabButtonLabel(tab))
	if !ok {
		t.Fatalf("tab button %q absent: %q", tab.Name, tester.Texts())
	}
	return r
}

func requireVisibleTab(t *testing.T, a *App, tester *ui.Tester, tab *Tab) ui.Rect {
	t.Helper()
	r := tabButtonRect(t, tester, tab)
	if r.W <= 0 || r.H <= 0 || r.X < a.SidebarWidth-1 || r.X+r.W > 761 || r.Y < 46 || r.Y+r.H > 86 {
		t.Fatalf("tab %q outside minimum-window strip: %+v sidebar=%g active=%d", tab.Name, r, a.SidebarWidth, a.Active)
	}
	return r
}

func TestTabRevealKeyboardWrapAndManualScroll(t *testing.T) {
	a, tester := newTabRevealFixture(t, 14)
	first, last := a.Tabs[0], a.Current()
	requireVisibleTab(t, a, tester, last)
	if r := tabButtonRect(t, tester, first); r.X+r.W > a.SidebarWidth {
		t.Fatalf("fixture did not put first tab offscreen: %+v", r)
	}
	tester.Key(ui.Cmd, ui.KeyTab)
	if a.Current() != first {
		t.Fatal("forward wrap did not select first")
	}
	requireVisibleTab(t, a, tester, first)
	tester.Key(ui.Cmd|ui.Shift, ui.KeyTab)
	if a.Current() != last {
		t.Fatal("reverse wrap did not select last")
	}
	r := requireVisibleTab(t, a, tester, last)
	tester.Scroll(a.SidebarWidth+50, r.Y+r.H/2, -10000, 0)
	manual := tabButtonRect(t, tester, last)
	if manual.X < 760 {
		t.Fatalf("manual scroll did not move active tab offscreen: %+v", manual)
	}
	tester.Frame()
	tester.Frame()
	if got := tabButtonRect(t, tester, last); got != manual || a.Current() != last {
		t.Fatalf("ordinary redraw pulled active tab back: before=%+v after=%+v", manual, got)
	}
	if !a.Select(a.Active) {
		t.Fatal("same-tab selection failed")
	}
	tester.Frame()
	if got := tabButtonRect(t, tester, last); got != manual {
		t.Fatalf("same-tab focus pulled the tab bar back: before=%+v after=%+v", manual, got)
	}
	tester.Key(ui.Cmd|ui.Shift, ui.KeyTab)
	requireVisibleTab(t, a, tester, a.Current())
}

func TestTabRevealAddOpenSelectAndRestore(t *testing.T) {
	a, tester := newTabRevealFixture(t, 12)
	first := a.Tabs[0]
	if !a.Select(0) {
		t.Fatal("could not select first")
	}
	tester.Frame()
	requireVisibleTab(t, a, tester, first)
	added := a.add(document.New("added"), "added-tab.txt", "")
	tester.Frame()
	requireVisibleTab(t, a, tester, added)
	path := filepath.Join(t.TempDir(), "opened-tab.txt")
	if err := os.WriteFile(path, []byte("opened"), 0600); err != nil {
		t.Fatal(err)
	}
	if !a.Select(0) {
		t.Fatal("could not select first before open")
	}
	tester.Frame()
	a.Open(path)
	opened := a.Current()
	if opened == first || opened.Doc.Path != path {
		t.Fatal("file did not open")
	}
	tester.Frame()
	requireVisibleTab(t, a, tester, opened)
	a.Persist()
	restored := newTestApp(t, a.configDir)
	restored.Restore()
	if len(restored.Tabs) != len(a.Tabs) || restored.Active != a.Active || restored.Current().Name != opened.Name {
		t.Fatal("session did not restore active tab order")
	}
	restoredTester := ui.NewTester(restored.View, 760, 480)
	requireVisibleTab(t, restored, restoredTester, restored.Current())
}

func TestTabRevealClickOrderingAndCloseNeighbors(t *testing.T) {
	a, tester := newTabRevealFixture(t, 14)
	first, last := a.Tabs[0], a.Current()
	r := requireVisibleTab(t, a, tester, last)
	tester.Scroll(a.SidebarWidth+50, r.Y+r.H/2, -10000, 0)
	requireVisibleTab(t, a, tester, first)
	if err := tester.Click(tabButtonLabel(first)); err != nil {
		t.Fatal(err)
	}
	if a.Current() != first {
		t.Fatal("click on earlier tab did not change active tab")
	}
	requireVisibleTab(t, a, tester, first)
	tester.Scroll(a.SidebarWidth+50, r.Y+r.H/2, 10000, 0)
	if err := tester.Click(tabButtonLabel(last)); err != nil {
		t.Fatal(err)
	}
	if a.Current() != last {
		t.Fatal("click on later tab did not change active tab")
	}
	requireVisibleTab(t, a, tester, last)
	tester.Scroll(a.SidebarWidth+50, r.Y+r.H/2, -10000, 0)
	if err := tester.Click("关闭 " + first.Name); err != nil {
		t.Fatal(err)
	}
	if len(a.Tabs) != 13 || a.Current() != last || a.Active != 12 {
		t.Fatal("inactive close did not preserve the selected tab identity")
	}
	requireVisibleTab(t, a, tester, a.Tabs[0])
	if got := tabButtonRect(t, tester, last); got.X < 760 {
		t.Fatalf("index change revealed the unchanged active identity: %+v", got)
	}
	assertTabAbsentFromBacking(t, a, first)
	// Select an early tab so its real close button is reachable. Closing
	// compacts the ranged backing array and must not render a cleared tail.
	first = a.Tabs[0]
	if !a.Select(0) {
		t.Fatal("could not select first before close")
	}
	tester.Frame()
	neighbor := a.Tabs[1]
	if err := tester.Click("关闭 " + first.Name); err != nil {
		t.Fatal(err)
	}
	if len(a.Tabs) != 12 || a.Current() != neighbor || a.Active != 0 {
		t.Fatal("closing first tab selected the wrong neighbor")
	}
	requireVisibleTab(t, a, tester, neighbor)
	assertTabAbsentFromBacking(t, a, first)
	if !a.Select(len(a.Tabs) - 1) {
		t.Fatal("could not select last before close")
	}
	tester.Frame()
	previous := a.Tabs[len(a.Tabs)-2]
	if !a.CloseTab(a.Active) {
		t.Fatal("closing last tab failed")
	}
	tester.Frame()
	if a.Current() != previous {
		t.Fatal("closing last tab selected wrong previous tab")
	}
	requireVisibleTab(t, a, tester, previous)
}

func TestTabRevealFailedFlushDoesNotMoveStrip(t *testing.T) {
	a, tester := newTabRevealFixture(t, 14)
	last := a.Current()
	r := requireVisibleTab(t, a, tester, last)
	tester.Scroll(a.SidebarWidth+50, r.Y+r.H/2, -10000, 0)
	before := tabButtonRect(t, tester, last)
	view := &failingView{flushErr: errors.New("keep draft"), cacheLoaded: true}
	last.View = view
	tester.Key(ui.Cmd, ui.KeyTab)
	if a.Current() != last || view.releases != 0 || !view.cacheLoaded {
		t.Fatal("failed keyboard switch changed active presentation")
	}
	if got := tabButtonRect(t, tester, last); got != before {
		t.Fatalf("failed switch scrolled the tab strip: before=%+v after=%+v", before, got)
	}
	if err := tester.Click(tabButtonLabel(a.Tabs[0])); err != nil {
		t.Fatal(err)
	}
	if a.Current() != last || tabButtonRect(t, tester, last) != before || !strings.Contains(a.Message, "keep draft") {
		t.Fatal("failed click switch moved the tab strip or hid the refusal")
	}
}

func TestTabRevealEmptyCloseAndReopen(t *testing.T) {
	a, tester := newTabRevealFixture(t, 1)
	old := a.Current()
	if err := tester.Click("关闭 " + old.Name); err != nil {
		t.Fatal(err)
	}
	if a.Current() != nil || a.Active != -1 || a.revealedTabID != 0 {
		t.Fatal("last close left active visibility state")
	}
	a.add(document.New("reopened"), old.Name, "")
	tester.Frame()
	if a.Current().ID == old.ID {
		t.Fatal("reopened tab reused old identity")
	}
	requireVisibleTab(t, a, tester, a.Current())
}

func TestTabWorkflowMinimumWindowCapture(t *testing.T) {
	output := os.Getenv("LEVNOTE_TAB_WORKFLOW_SCREENSHOT_DIR")
	if output == "" {
		t.Skip("set LEVNOTE_TAB_WORKFLOW_SCREENSHOT_DIR for tab workflow geometry capture")
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		t.Fatal(err)
	}
	a, tester := newTabRevealFixture(t, 14)
	tester.Key(ui.Cmd, ui.KeyTab)
	requireVisibleTab(t, a, tester, a.Tabs[0])
	tester.Key(ui.Cmd|ui.Shift, ui.KeyTab)
	requireVisibleTab(t, a, tester, a.Tabs[13])
	captureAcceptanceTester(t, filepath.Join(output, "tab-active-wrap-760x480.png"), tester)
}
