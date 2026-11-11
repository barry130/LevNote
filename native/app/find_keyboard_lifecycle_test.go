package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
)

func TestFindKeyboardFocusLeaveReconcilesComposition(t *testing.T) {
	t.Run("query to source", func(t *testing.T) {
		a, tab := newSearchTextApp(t, "needle needle")
		a.ShowFind = true
		a.Replacement = "match"
		a.CaseSensitive = true
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		if err := tester.Click("查找内容"); err != nil {
			t.Fatal(err)
		}
		tester.Compose("needle", 6)
		if !a.FindQueryComposing {
			t.Fatal("query composition was not routed to Find")
		}

		beforeText := tab.Doc.Buffer.String()
		beforeVersion := tab.Doc.Buffer.Version()
		beforeHistory := tab.Doc.Buffer.History()
		if err := tester.Click("文档编辑区"); err != nil {
			t.Fatal(err)
		}
		if a.FindQueryComposing || a.Query != "needle" {
			t.Fatalf("source focus left stale query composition: composing=%v query=%q focused-query=%v",
				a.FindQueryComposing, a.Query, tester.Focused("查找内容"))
		}
		queryAccess, ok := tester.AccessByRole(ui.RoleTextField, "查找内容")
		if !ok || queryAccess.Value != "needle" {
			t.Fatalf("query commit did not happen exactly once: %#v", queryAccess)
		}
		afterFocus := tab.Doc.Buffer.Selection()
		tester.Frame()
		tester.Frame()
		if tab.Doc.Buffer.String() != beforeText || tab.Doc.Buffer.Version() != beforeVersion ||
			tab.Doc.Buffer.History() != beforeHistory || tab.Doc.Buffer.Selection() != afterFocus {
			t.Fatalf("Find cleanup changed source: text=%q version=%d history=%+v selection=%+v/%+v",
				tab.Doc.Buffer.String(), tab.Doc.Buffer.Version(), tab.Doc.Buffer.History(),
				tab.Doc.Buffer.Selection(), afterFocus)
		}

		tester.Key(ui.Cmd, ui.KeyF)
		if !tester.Focused("查找内容") {
			t.Fatalf("Ctrl+F stayed blocked after query focus leave: composing=%v message=%q",
				a.FindQueryComposing, a.Message)
		}
		setSearchSelection(t, tab, 0, 0)
		tester.Key(0, ui.KeyEnter)
		assertSearchSelection(t, tab, 0, 6)
		if !tester.Focused("查找内容") {
			t.Fatal("Enter did not retain query focus after focus-leave reconciliation")
		}
		tester.Key(0, ui.KeyEscape)
		if a.ShowFind || !tester.Focused("文档编辑区") {
			t.Fatalf("Escape did not return to source: show=%v focused=%v",
				a.ShowFind, tester.Focused("文档编辑区"))
		}
	})

	t.Run("replacement to source", func(t *testing.T) {
		a, tab := newSearchTextApp(t, "needle needle")
		a.Query = "needle"
		a.ShowFind = true
		a.CaseSensitive = true
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		if err := tester.Click("替换内容"); err != nil {
			t.Fatal(err)
		}
		tester.Compose("match", 5)
		if !a.FindReplacementComposing {
			t.Fatal("replacement composition was not routed to Find")
		}

		beforeText := tab.Doc.Buffer.String()
		beforeVersion := tab.Doc.Buffer.Version()
		beforeHistory := tab.Doc.Buffer.History()
		if err := tester.Click("文档编辑区"); err != nil {
			t.Fatal(err)
		}
		if a.FindReplacementComposing || a.Replacement != "match" {
			t.Fatalf("source focus left stale replacement composition: composing=%v replacement=%q focused-replacement=%v",
				a.FindReplacementComposing, a.Replacement, tester.Focused("替换内容"))
		}
		replacementAccess, ok := tester.AccessByRole(ui.RoleTextField, "替换内容")
		if !ok || replacementAccess.Value != "match" {
			t.Fatalf("replacement commit did not happen exactly once: %#v", replacementAccess)
		}
		afterFocus := tab.Doc.Buffer.Selection()
		tester.Frame()
		tester.Frame()
		if tab.Doc.Buffer.String() != beforeText || tab.Doc.Buffer.Version() != beforeVersion ||
			tab.Doc.Buffer.History() != beforeHistory || tab.Doc.Buffer.Selection() != afterFocus {
			t.Fatalf("replacement cleanup changed source: text=%q version=%d history=%+v selection=%+v/%+v",
				tab.Doc.Buffer.String(), tab.Doc.Buffer.Version(), tab.Doc.Buffer.History(),
				tab.Doc.Buffer.Selection(), afterFocus)
		}

		tester.Key(ui.Cmd, ui.KeyF)
		if !tester.Focused("查找内容") {
			t.Fatalf("Ctrl+F stayed blocked after replacement focus leave: composing=%v message=%q",
				a.FindReplacementComposing, a.Message)
		}
		setSearchSelection(t, tab, 0, 6)
		if err := tester.Click("替换内容"); err != nil {
			t.Fatal(err)
		}
		history := tab.Doc.Buffer.History().UndoDepth
		tester.Key(0, ui.KeyEnter)
		if tab.Doc.Buffer.String() != "match needle" || tab.Doc.Buffer.History().UndoDepth != history+1 ||
			!tester.Focused("替换内容") {
			t.Fatalf("replacement Enter after focus leave: text=%q history=%+v focused=%v",
				tab.Doc.Buffer.String(), tab.Doc.Buffer.History(), tester.Focused("替换内容"))
		}
		tester.Key(0, ui.KeyEscape)
		if a.ShowFind || !tester.Focused("文档编辑区") {
			t.Fatalf("replacement Escape did not return to source: show=%v focused=%v",
				a.ShowFind, tester.Focused("文档编辑区"))
		}
	})

	t.Run("query to tab", func(t *testing.T) {
		a, first := newSearchTextApp(t, "needle needle")
		first.Name = "first-source.txt"
		a.add(document.New("needle needle"), "second-source.txt", "")
		second := a.Current()
		if !a.Select(0) {
			t.Fatal("could not return to first tab")
		}
		a.ShowFind = true
		a.Replacement = "match"
		a.CaseSensitive = true
		firstBefore := first.Doc.Buffer.String()
		firstVersion := first.Doc.Buffer.Version()
		firstHistory := first.Doc.Buffer.History()
		secondBefore := second.Doc.Buffer.String()
		secondVersion := second.Doc.Buffer.Version()
		secondHistory := second.Doc.Buffer.History()
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		if err := tester.Click("查找内容"); err != nil {
			t.Fatal(err)
		}
		tester.Compose("needle", 6)
		if !a.FindQueryComposing {
			t.Fatal("query composition was not routed before tab switch")
		}

		if err := tester.Click("• second-source.txt"); err != nil {
			t.Fatal(err)
		}
		if a.Current() != second || a.FindQueryComposing || a.Query != "needle" ||
			tester.Focused("查找内容") {
			t.Fatalf("tab switch left stale query lifecycle: current=%p composing=%v query=%q focused-query=%v",
				a.Current(), a.FindQueryComposing, a.Query, tester.Focused("查找内容"))
		}
		if a.findFocus != findFocusNone || a.findFocusTab != nil || a.findFocusDoc != nil ||
			a.findFocusEditor != nil || a.findFocusBuffer != nil {
			t.Fatalf("tab switch retained stale Find focus request: target=%v tab=%p doc=%p editor=%p buffer=%p",
				a.findFocus, a.findFocusTab, a.findFocusDoc, a.findFocusEditor, a.findFocusBuffer)
		}
		if first.Doc.Buffer.String() != firstBefore || first.Doc.Buffer.Version() != firstVersion ||
			first.Doc.Buffer.History() != firstHistory ||
			second.Doc.Buffer.String() != secondBefore || second.Doc.Buffer.Version() != secondVersion ||
			second.Doc.Buffer.History() != secondHistory {
			t.Fatalf("Find lifecycle crossed source ownership: first=%q/%d/%+v second=%q/%d/%+v",
				first.Doc.Buffer.String(), first.Doc.Buffer.Version(), first.Doc.Buffer.History(),
				second.Doc.Buffer.String(), second.Doc.Buffer.Version(), second.Doc.Buffer.History())
		}

		tester.Frame()
		if tester.Focused("查找内容") {
			t.Fatal("a later frame stole focus back to the abandoned query")
		}
		tester.Key(ui.Cmd, ui.KeyF)
		if !tester.Focused("查找内容") {
			t.Fatalf("Ctrl+F stayed blocked after tab switch: composing=%v message=%q",
				a.FindQueryComposing, a.Message)
		}
		setSearchSelection(t, second, 0, 0)
		tester.Key(0, ui.KeyEnter)
		assertSearchSelection(t, second, 0, 6)
		if first.Doc.Buffer.String() != firstBefore {
			t.Fatalf("Enter after tab switch changed old source: %q", first.Doc.Buffer.String())
		}
		tester.Key(0, ui.KeyEscape)
		if a.ShowFind || !tester.Focused("文档编辑区") {
			t.Fatalf("Escape after tab switch did not restore current source: show=%v focused=%v",
				a.ShowFind, tester.Focused("文档编辑区"))
		}
	})

	t.Run("replacement to tab", func(t *testing.T) {
		a, first := newSearchTextApp(t, "needle needle")
		first.Name = "first-source.txt"
		a.add(document.New("needle needle"), "second-source.txt", "")
		second := a.Current()
		if !a.Select(0) {
			t.Fatal("could not return to first tab")
		}
		a.Query = "needle"
		a.ShowFind = true
		a.CaseSensitive = true
		firstBefore := first.Doc.Buffer.String()
		firstVersion := first.Doc.Buffer.Version()
		firstHistory := first.Doc.Buffer.History()
		secondBefore := second.Doc.Buffer.String()
		secondVersion := second.Doc.Buffer.Version()
		secondHistory := second.Doc.Buffer.History()
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		if err := tester.Click("替换内容"); err != nil {
			t.Fatal(err)
		}
		tester.Compose("match", 5)
		if !a.FindReplacementComposing {
			t.Fatal("replacement composition was not routed before tab switch")
		}

		if err := tester.Click("• second-source.txt"); err != nil {
			t.Fatal(err)
		}
		if a.Current() != second || a.FindReplacementComposing || a.Replacement != "match" ||
			tester.Focused("替换内容") {
			t.Fatalf("tab switch left stale replacement lifecycle: current=%p composing=%v replacement=%q focused-replacement=%v",
				a.Current(), a.FindReplacementComposing, a.Replacement, tester.Focused("替换内容"))
		}
		if a.findFocus != findFocusNone || a.findFocusTab != nil || a.findFocusDoc != nil ||
			a.findFocusEditor != nil || a.findFocusBuffer != nil {
			t.Fatalf("replacement tab switch retained stale Find focus request: target=%v tab=%p doc=%p editor=%p buffer=%p",
				a.findFocus, a.findFocusTab, a.findFocusDoc, a.findFocusEditor, a.findFocusBuffer)
		}
		if first.Doc.Buffer.String() != firstBefore || first.Doc.Buffer.Version() != firstVersion ||
			first.Doc.Buffer.History() != firstHistory ||
			second.Doc.Buffer.String() != secondBefore || second.Doc.Buffer.Version() != secondVersion ||
			second.Doc.Buffer.History() != secondHistory {
			t.Fatalf("replacement lifecycle crossed source ownership: first=%q/%d/%+v second=%q/%d/%+v",
				first.Doc.Buffer.String(), first.Doc.Buffer.Version(), first.Doc.Buffer.History(),
				second.Doc.Buffer.String(), second.Doc.Buffer.Version(), second.Doc.Buffer.History())
		}

		tester.Frame()
		if tester.Focused("替换内容") {
			t.Fatal("a later frame stole focus back to the abandoned replacement")
		}
		tester.Key(ui.Cmd, ui.KeyF)
		if !tester.Focused("查找内容") {
			t.Fatalf("Ctrl+F stayed blocked after replacement tab switch: composing=%v message=%q",
				a.FindReplacementComposing, a.Message)
		}
		setSearchSelection(t, second, 0, 6)
		if err := tester.Click("替换内容"); err != nil {
			t.Fatal(err)
		}
		history := second.Doc.Buffer.History().UndoDepth
		tester.Key(0, ui.KeyEnter)
		if second.Doc.Buffer.String() != "match needle" || second.Doc.Buffer.History().UndoDepth != history+1 ||
			first.Doc.Buffer.String() != firstBefore || !tester.Focused("替换内容") {
			t.Fatalf("replacement Enter after tab switch: second=%q/%+v first=%q focused=%v",
				second.Doc.Buffer.String(), second.Doc.Buffer.History(), first.Doc.Buffer.String(),
				tester.Focused("替换内容"))
		}
		tester.Key(0, ui.KeyEscape)
		if a.ShowFind || !tester.Focused("文档编辑区") {
			t.Fatalf("replacement tab-switch Escape did not restore current source: show=%v focused=%v",
				a.ShowFind, tester.Focused("文档编辑区"))
		}
	})

	t.Run("plugins hide and show", func(t *testing.T) {
		a, tab := newSearchTextApp(t, "needle needle")
		a.ShowFind = true
		a.Replacement = "match"
		a.CaseSensitive = true
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		if err := tester.Click("查找内容"); err != nil {
			t.Fatal(err)
		}
		tester.Compose("needle", 6)
		if !a.FindQueryComposing {
			t.Fatal("query composition was not routed before plugin hide")
		}
		beforeText := tab.Doc.Buffer.String()
		beforeVersion := tab.Doc.Buffer.Version()
		beforeHistory := tab.Doc.Buffer.History()

		if err := tester.Click("插件"); err != nil {
			t.Fatal(err)
		}
		if !a.ShowPlugins || !a.ShowFind || a.FindQueryComposing || a.Query != "needle" ||
			a.Replacement != "match" || !a.CaseSensitive || a.Regex {
			t.Fatalf("plugin hide did not clear only transient query state: plugins=%v find=%v composing=%v query=%q replacement=%q case=%v regex=%v",
				a.ShowPlugins, a.ShowFind, a.FindQueryComposing, a.Query, a.Replacement, a.CaseSensitive, a.Regex)
		}
		if a.findFocus != findFocusNone || a.findFocusTab != nil || a.findFocusDoc != nil ||
			a.findFocusEditor != nil || a.findFocusBuffer != nil {
			t.Fatalf("plugin hide retained stale Find focus request: target=%v tab=%p doc=%p editor=%p buffer=%p",
				a.findFocus, a.findFocusTab, a.findFocusDoc, a.findFocusEditor, a.findFocusBuffer)
		}
		if tab.Doc.Buffer.String() != beforeText || tab.Doc.Buffer.Version() != beforeVersion ||
			tab.Doc.Buffer.History() != beforeHistory {
			t.Fatalf("plugin hide cleanup changed source: text=%q version=%d history=%+v",
				tab.Doc.Buffer.String(), tab.Doc.Buffer.Version(), tab.Doc.Buffer.History())
		}

		if err := tester.Click("插件"); err != nil {
			t.Fatal(err)
		}
		if a.ShowPlugins || !a.ShowFind || a.Query != "needle" || a.Replacement != "match" ||
			!a.CaseSensitive || a.Regex || tester.Focused("查找内容") {
			t.Fatalf("plugin show restored values with an unsolicited focus: plugins=%v find=%v query=%q replacement=%q case=%v regex=%v focused-query=%v",
				a.ShowPlugins, a.ShowFind, a.Query, a.Replacement, a.CaseSensitive, a.Regex,
				tester.Focused("查找内容"))
		}
		if err := tester.Click("文档编辑区"); err != nil {
			t.Fatal(err)
		}
		tester.Frame()
		tester.Frame()
		if tester.Focused("查找内容") {
			t.Fatal("a later frame stole focus after plugin show")
		}
		tester.Key(ui.Cmd, ui.KeyF)
		if !tester.Focused("查找内容") {
			t.Fatalf("Ctrl+F stayed blocked after plugin show: composing=%v message=%q",
				a.FindQueryComposing, a.Message)
		}
		setSearchSelection(t, tab, 0, 0)
		tester.Key(0, ui.KeyEnter)
		assertSearchSelection(t, tab, 0, 6)
		tester.Key(0, ui.KeyEscape)
		if a.ShowFind || !tester.Focused("文档编辑区") {
			t.Fatalf("Escape after plugin show did not restore source: show=%v focused=%v",
				a.ShowFind, tester.Focused("文档编辑区"))
		}
	})

	t.Run("replacement plugins hide and show", func(t *testing.T) {
		a, tab := newSearchTextApp(t, "needle needle")
		a.Query = "needle"
		a.ShowFind = true
		a.CaseSensitive = true
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		if err := tester.Click("替换内容"); err != nil {
			t.Fatal(err)
		}
		tester.Compose("match", 5)
		if !a.FindReplacementComposing {
			t.Fatal("replacement composition was not routed before plugin hide")
		}
		beforeText := tab.Doc.Buffer.String()
		beforeVersion := tab.Doc.Buffer.Version()
		beforeHistory := tab.Doc.Buffer.History()

		if err := tester.Click("插件"); err != nil {
			t.Fatal(err)
		}
		if !a.ShowPlugins || !a.ShowFind || a.FindReplacementComposing || a.Replacement != "match" ||
			a.Query != "needle" || !a.CaseSensitive || a.Regex {
			t.Fatalf("plugin hide did not clear only replacement composition: plugins=%v find=%v composing=%v query=%q replacement=%q case=%v regex=%v",
				a.ShowPlugins, a.ShowFind, a.FindReplacementComposing, a.Query, a.Replacement, a.CaseSensitive, a.Regex)
		}
		if a.findFocus != findFocusNone || a.findFocusTab != nil || a.findFocusDoc != nil ||
			a.findFocusEditor != nil || a.findFocusBuffer != nil {
			t.Fatalf("replacement plugin hide retained stale Find focus request: target=%v tab=%p doc=%p editor=%p buffer=%p",
				a.findFocus, a.findFocusTab, a.findFocusDoc, a.findFocusEditor, a.findFocusBuffer)
		}
		if tab.Doc.Buffer.String() != beforeText || tab.Doc.Buffer.Version() != beforeVersion ||
			tab.Doc.Buffer.History() != beforeHistory {
			t.Fatalf("replacement plugin hide cleanup changed source: text=%q version=%d history=%+v",
				tab.Doc.Buffer.String(), tab.Doc.Buffer.Version(), tab.Doc.Buffer.History())
		}

		if err := tester.Click("插件"); err != nil {
			t.Fatal(err)
		}
		if a.ShowPlugins || !a.ShowFind || a.Query != "needle" || a.Replacement != "match" ||
			!a.CaseSensitive || a.Regex || tester.Focused("替换内容") {
			t.Fatalf("replacement plugin show restored values with unsolicited focus: plugins=%v find=%v query=%q replacement=%q case=%v regex=%v focused-replacement=%v",
				a.ShowPlugins, a.ShowFind, a.Query, a.Replacement, a.CaseSensitive, a.Regex,
				tester.Focused("替换内容"))
		}
		if err := tester.Click("文档编辑区"); err != nil {
			t.Fatal(err)
		}
		tester.Frame()
		tester.Frame()
		if tester.Focused("替换内容") {
			t.Fatal("a later frame stole focus after replacement plugin show")
		}
		tester.Key(ui.Cmd, ui.KeyF)
		if !tester.Focused("查找内容") {
			t.Fatalf("Ctrl+F stayed blocked after replacement plugin show: composing=%v message=%q",
				a.FindReplacementComposing, a.Message)
		}
		setSearchSelection(t, tab, 0, 6)
		if err := tester.Click("替换内容"); err != nil {
			t.Fatal(err)
		}
		history := tab.Doc.Buffer.History().UndoDepth
		tester.Key(0, ui.KeyEnter)
		if tab.Doc.Buffer.String() != "match needle" || tab.Doc.Buffer.History().UndoDepth != history+1 ||
			!tester.Focused("替换内容") {
			t.Fatalf("replacement Enter after plugin show: text=%q history=%+v focused=%v",
				tab.Doc.Buffer.String(), tab.Doc.Buffer.History(), tester.Focused("替换内容"))
		}
		tester.Key(0, ui.KeyEscape)
		if a.ShowFind || !tester.Focused("文档编辑区") {
			t.Fatalf("replacement plugin Escape did not restore source: show=%v focused=%v",
				a.ShowFind, tester.Focused("文档编辑区"))
		}
	})

	t.Run("plugins hide pending focus", func(t *testing.T) {
		a, _ := newSearchTextApp(t, "needle")
		a.ShowFind = true
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		if err := tester.Click("文档编辑区"); err != nil {
			t.Fatal(err)
		}
		if !a.openFind() || a.findFocus != findFocusQuery || a.findFocusTab == nil ||
			a.findFocusDoc == nil || a.findFocusEditor == nil || a.findFocusBuffer == nil {
			t.Fatalf("could not stage pending Find focus: target=%v tab=%p doc=%p editor=%p buffer=%p",
				a.findFocus, a.findFocusTab, a.findFocusDoc, a.findFocusEditor, a.findFocusBuffer)
		}

		if err := tester.Click("插件"); err != nil {
			t.Fatal(err)
		}
		if !a.ShowPlugins || a.findFocus != findFocusNone || a.findFocusTab != nil ||
			a.findFocusDoc != nil || a.findFocusEditor != nil || a.findFocusBuffer != nil {
			t.Fatalf("plugin hide retained pending Find focus: plugins=%v target=%v tab=%p doc=%p editor=%p buffer=%p",
				a.ShowPlugins, a.findFocus, a.findFocusTab, a.findFocusDoc, a.findFocusEditor, a.findFocusBuffer)
		}
		if err := tester.Click("插件"); err != nil {
			t.Fatal(err)
		}
		tester.Frame()
		tester.Frame()
		if tester.Focused("查找内容") {
			t.Fatal("hidden pending Find focus became a delayed focus steal")
		}
	})
}
