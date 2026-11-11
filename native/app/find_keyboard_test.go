package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
)

func TestFindKeyboardCtrlFFocusesQueryOnceAndPreservesFieldState(t *testing.T) {
	a, tab := newSearchTextApp(t, "needle needle")
	a.Query = "needle"
	a.Replacement = "match"
	a.CaseSensitive = true
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()

	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Key(ui.Cmd, ui.KeyF)
	if !a.ShowFind || !tester.Focused("查找内容") {
		t.Fatalf("Ctrl+F did not open and focus query: show=%v texts=%q", a.ShowFind, tester.Texts())
	}
	if a.Query != "needle" || a.Replacement != "match" || !a.CaseSensitive || a.Regex {
		t.Fatalf("Ctrl+F changed field/options state: query=%q replacement=%q case=%v regex=%v",
			a.Query, a.Replacement, a.CaseSensitive, a.Regex)
	}

	if err := tester.Click("替换内容"); err != nil {
		t.Fatal(err)
	}
	tester.Type("/draft")
	tester.Frame()
	if a.Replacement != "match/draft" {
		t.Fatalf("replacement typing = %q", a.Replacement)
	}
	tester.Key(ui.Cmd, ui.KeyF)
	if !tester.Focused("查找内容") || a.Replacement != "match/draft" {
		t.Fatalf("repeated Ctrl+F did not focus existing query without clearing replacement: focused=%v replacement=%q",
			tester.Focused("查找内容"), a.Replacement)
	}
	if a.Query != "needle" || !a.CaseSensitive || a.Regex {
		t.Fatalf("repeated Ctrl+F changed fields/options: query=%q case=%v regex=%v",
			a.Query, a.CaseSensitive, a.Regex)
	}

	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	tester.Frame()
	if !tester.Focused("文档编辑区") || tester.Focused("查找内容") {
		t.Fatal("an ordinary frame stole focus back to the Find query")
	}
	if err := tab.Doc.Buffer.SetSelection(0, 0); err != nil {
		t.Fatal(err)
	}
	tester.Type("x")
	tester.Frame()
	if !strings.HasPrefix(tab.Doc.Buffer.String(), "xneedle") {
		t.Fatalf("source typing was not routed to the focused editor: text=%q selection=%+v",
			tab.Doc.Buffer.String(), tab.Doc.Buffer.Selection())
	}

	if err := tester.Click("查找"); err != nil {
		t.Fatal(err)
	}
	if a.ShowFind {
		t.Fatal("toolbar Find toggle did not close the visible Find bar")
	}
	if err := tester.Click("查找"); err != nil {
		t.Fatal(err)
	}
	if !a.ShowFind || !tester.Focused("查找内容") {
		t.Fatalf("toolbar open did not use the one-shot query focus: show=%v focused=%v",
			a.ShowFind, tester.Focused("查找内容"))
	}
}

func TestFindKeyboardEnterNavigationUsesRepeatedQueryFocus(t *testing.T) {
	a, tab := newSearchTextApp(t, "needle needle")
	a.Query = "needle"
	a.ShowFind = true
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if err := tester.Click("查找内容"); err != nil {
		t.Fatal(err)
	}

	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	assertSearchSelection(t, tab, 0, 6)
	if !tester.Focused("查找内容") {
		t.Fatal("first Enter did not retain query focus")
	}
	tester.Key(0, ui.KeyEnter)
	assertSearchSelection(t, tab, 7, 13)
	if !tester.Focused("查找内容") {
		t.Fatal("repeated Enter did not retain query focus")
	}
	tester.Key(0, ui.KeyEnter)
	assertSearchSelection(t, tab, 0, 6)
	tester.Key(ui.Shift, ui.KeyEnter)
	assertSearchSelection(t, tab, 7, 13)
	if !tester.Focused("查找内容") {
		t.Fatal("Shift+Enter did not retain query focus")
	}
}

func TestFindKeyboardEnterRetainsFocusAfterFailedFlush(t *testing.T) {
	a, tab := newSearchTextApp(t, "needle needle")
	a.Query = "needle"
	a.Replacement = "match"
	a.ShowFind = true
	setSearchSelection(t, tab, 0, 0)
	a.find(true)
	beforeText := tab.Doc.Buffer.String()
	beforeSelection := tab.Doc.Buffer.Selection()
	beforeVersion := tab.Doc.Buffer.Version()
	beforeHistory := tab.Doc.Buffer.History()
	view := &failingView{flushErr: errors.New("draft conflict")}
	tab.View = view

	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if err := tester.Click("替换内容"); err != nil {
		t.Fatal(err)
	}
	tester.Key(0, ui.KeyEnter)
	if view.flushes != 1 || tab.Doc.Buffer.String() != beforeText ||
		tab.Doc.Buffer.Selection() != beforeSelection ||
		tab.Doc.Buffer.Version() != beforeVersion ||
		tab.Doc.Buffer.History() != beforeHistory {
		t.Fatalf("failed Flush replacement changed source: flushes=%d text=%q selection=%+v version=%d history=%+v",
			view.flushes, tab.Doc.Buffer.String(), tab.Doc.Buffer.Selection(),
			tab.Doc.Buffer.Version(), tab.Doc.Buffer.History())
	}
	if !a.ShowFind || !tester.Focused("替换内容") {
		t.Fatalf("failed Flush lost Find focus: show=%v focus=%v", a.ShowFind, tester.Focused("替换内容"))
	}
}

func TestFindKeyboardStaleEditorBufferCallbackRejected(t *testing.T) {
	a, tab := newSearchTextApp(t, "needle needle")
	a.ShowFind = true
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if err := tester.Click("查找内容"); err != nil {
		t.Fatal(err)
	}
	tester.Type("needle")
	before := tab.Doc.Buffer.Selection()
	other := document.New("needle needle")
	otherBefore := other.Buffer.Selection()
	tab.Editor.Buffer = other.Buffer
	tester.Key(0, ui.KeyEnter)
	if tab.Doc.Buffer.Selection() != before || other.Buffer.Selection() != otherBefore {
		t.Fatalf("stale editor-buffer callback navigated: current=%+v other=%+v",
			tab.Doc.Buffer.Selection(), other.Buffer.Selection())
	}
	tab.Editor.Buffer = tab.Doc.Buffer
	tester.Frame()
	if !tester.Focused("查找内容") {
		t.Fatal("fresh editor-buffer frame did not retain query focus")
	}
	tester.Key(0, ui.KeyEnter)
	assertSearchSelection(t, tab, 0, 6)
}

func TestFindKeyboardReplacementEnterAndEscape(t *testing.T) {
	a, tab := newSearchTextApp(t, "needle needle")
	a.Query = "needle"
	a.Replacement = "match"
	a.ShowFind = true
	if err := tab.Doc.Buffer.SetSelection(0, 0); err != nil {
		t.Fatal(err)
	}
	a.find(true)
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if err := tester.Click("替换内容"); err != nil {
		t.Fatal(err)
	}
	history := tab.Doc.Buffer.History().UndoDepth
	tester.Key(0, ui.KeyEnter)
	if tab.Doc.Buffer.String() != "match needle" || tab.Doc.Buffer.History().UndoDepth != history+1 ||
		!tester.Focused("替换内容") || !a.ShowFind {
		t.Fatalf("replacement Enter = text=%q history=%+v focus=%v show=%v",
			tab.Doc.Buffer.String(), tab.Doc.Buffer.History(), tester.Focused("替换内容"), a.ShowFind)
	}

	if err := tester.Click("查找内容"); err != nil {
		t.Fatal(err)
	}
	tester.Key(0, ui.KeyEscape)
	if a.ShowFind || !tester.Focused("文档编辑区") {
		t.Fatalf("Escape did not close Find and focus the live source: show=%v focused=%v",
			a.ShowFind, tester.Focused("文档编辑区"))
	}
	tester.Type("!")
	tester.Frame()
	if tab.Doc.Buffer.String() != "match !" ||
		a.Query != "needle" || a.Replacement != "match" {
		t.Fatalf("post-Escape source typing/state = text=%q query=%q replacement=%q",
			tab.Doc.Buffer.String(), a.Query, a.Replacement)
	}
}

func TestFindKeyboardCompositionAndSourceCompositionBarriers(t *testing.T) {
	t.Run("Find Escape", func(t *testing.T) {
		a, tab := newSearchTextApp(t, "needle needle")
		a.ShowFind = true
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		if err := tester.Click("查找内容"); err != nil {
			t.Fatal(err)
		}
		tester.Compose("needle", 6)
		selection := tab.Doc.Buffer.Selection()
		tester.Key(0, ui.KeyEscape)
		if !a.ShowFind || !tester.Focused("查找内容") ||
			a.Query != "" || tab.Doc.Buffer.Selection() != selection {
			t.Fatalf("Escape crossed Find composition: show=%v focus=%v query=%q selection=%+v/%+v",
				a.ShowFind, tester.Focused("查找内容"), a.Query, selection, tab.Doc.Buffer.Selection())
		}
	})

	t.Run("source", func(t *testing.T) {
		a, tab := newSearchTextApp(t, "")
		mark := ui.TextInputRange{Start: 0, End: 0}
		tab.Editor.SetMarkedText(&mark, "中", ui.TextInputRange{Start: 1, End: 1})
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		tester.Key(ui.Cmd, ui.KeyF)
		if a.ShowFind || !strings.Contains(a.Message, "输入法正在组合") {
			t.Fatalf("source composition did not block Find opening: show=%v message=%q",
				a.ShowFind, a.Message)
		}
		if err := tester.Click("查找"); err != nil {
			t.Fatal(err)
		}
		if a.ShowFind {
			t.Fatal("toolbar Find crossed source composition")
		}
		if _, composing := tab.Editor.MarkedRange(); !composing {
			t.Fatal("toolbar Find discarded source composition")
		}
		tab.Editor.UnmarkText()
	})

	t.Run("Ctrl+G", func(t *testing.T) {
		a, _ := newSearchTextApp(t, "needle")
		a.ShowFind = true
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		if err := tester.Click("查找内容"); err != nil {
			t.Fatal(err)
		}
		tester.Compose("needle", 6)
		tester.Key(ui.Cmd, ui.KeyG)
		if a.ShowGoToLine || !a.ShowFind || !a.FindQueryComposing || a.Query != "" ||
			!tester.Focused("查找内容") || !strings.Contains(a.Message, "输入法正在组合") {
			t.Fatalf("Ctrl+G crossed Find composition: goto=%v find=%v composing=%v query=%q focus=%v message=%q",
				a.ShowGoToLine, a.ShowFind, a.FindQueryComposing, a.Query, tester.Focused("查找内容"), a.Message)
		}
		tester.Compose("", 0)
		tester.Key(0, ui.KeyEscape)
		tester.Key(ui.Cmd, ui.KeyG)
		if !a.ShowGoToLine || !tester.Focused("转到行号") {
			t.Fatalf("Ctrl+G did not work after Find composition ended: show=%v focus=%v",
				a.ShowGoToLine, tester.Focused("转到行号"))
		}
	})
}

func TestFindKeyboardTypedQueryUsesCurrentFieldAndStaleTabRejects(t *testing.T) {
	a, first := newSearchTextApp(t, "needle")
	a.NewDocument("text")
	second := a.Current()
	if err := second.Doc.Buffer.ReplaceUTF16(0, 0, "needle needle"); err != nil {
		t.Fatal(err)
	}
	if !a.Select(0) {
		t.Fatal("could not return to first tab")
	}
	a.ShowFind = true
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if err := tester.Click("查找内容"); err != nil {
		t.Fatal(err)
	}
	tester.Type("needle")
	if a.Query != "needle" {
		t.Fatalf("typed query routing = %q", a.Query)
	}
	tester.Key(0, ui.KeyEnter)
	assertSearchSelection(t, first, 0, 6)
	if !tester.Focused("查找内容") {
		t.Fatal("typed query Enter did not retain query focus")
	}

	if !a.Select(1) {
		t.Fatal("could not switch to second tab")
	}
	before := second.Doc.Buffer.Selection()
	tester.Key(0, ui.KeyEnter)
	if second.Doc.Buffer.Selection() != before {
		t.Fatalf("stale first-tab callback navigated second tab: got %#v, want %#v",
			second.Doc.Buffer.Selection(), before)
	}
	tester.Frame()
	tester.Key(ui.Cmd, ui.KeyF)
	if !tester.Focused("查找内容") {
		t.Fatal("repeated Ctrl+F did not focus the rebound query")
	}
	tester.Key(0, ui.KeyEnter)
	assertSearchSelection(t, second, 0, 6)
	if !tester.Focused("查找内容") {
		t.Fatal("rebound second-tab query did not retain focus")
	}

	beforeCloseReopen := second.Doc.Buffer.Selection()
	a.closeFind()
	if !a.openFind() {
		t.Fatal("could not reopen Find after closing it")
	}
	tester.Key(0, ui.KeyEnter)
	if second.Doc.Buffer.Selection() != beforeCloseReopen {
		t.Fatalf("stale close/reopen callback navigated: got %+v, want %+v",
			second.Doc.Buffer.Selection(), beforeCloseReopen)
	}
	tester.Frame()
	if !tester.Focused("查找内容") {
		t.Fatal("reopened Find did not focus its current-generation query")
	}
	tester.Key(0, ui.KeyEnter)
	assertSearchSelection(t, second, 7, 13)
}

func TestFindKeyboardReplacementFreshTextShiftReadonlyAndComposition(t *testing.T) {
	t.Run("fresh text and shift", func(t *testing.T) {
		a, tab := newSearchTextApp(t, "needle needle")
		a.Query = "needle"
		a.ShowFind = true
		setSearchSelection(t, tab, 0, 0)
		a.find(true)
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		if err := tester.Click("替换内容"); err != nil {
			t.Fatal(err)
		}
		history := tab.Doc.Buffer.History().UndoDepth
		tester.Type("match")
		tester.Key(0, ui.KeyEnter)
		if tab.Doc.Buffer.String() != "match needle" || tab.Doc.Buffer.History().UndoDepth != history+1 ||
			a.Replacement != "match" || !tester.Focused("替换内容") {
			t.Fatalf("fresh replacement Enter: text=%q history=%+v replacement=%q focus=%v",
				tab.Doc.Buffer.String(), tab.Doc.Buffer.History(), a.Replacement, tester.Focused("替换内容"))
		}
		afterText := tab.Doc.Buffer.String()
		afterHistory := tab.Doc.Buffer.History()
		tester.Key(ui.Shift, ui.KeyEnter)
		if tab.Doc.Buffer.String() != afterText || tab.Doc.Buffer.History() != afterHistory || !a.ShowFind {
			t.Fatalf("Shift+Enter replaced: text=%q history=%+v show=%v",
				tab.Doc.Buffer.String(), tab.Doc.Buffer.History(), a.ShowFind)
		}
	})

	t.Run("readonly", func(t *testing.T) {
		a, tab := newSearchTextApp(t, "needle needle")
		tab.Doc.ReadOnly = true
		tab.Editor.ReadOnly = true
		a.Query = "needle"
		a.Replacement = "match"
		a.ShowFind = true
		setSearchSelection(t, tab, 0, 0)
		a.find(true)
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		if err := tester.Click("替换内容"); err != nil {
			t.Fatal(err)
		}
		before := tab.Doc.Buffer.String()
		beforeSelection := tab.Doc.Buffer.Selection()
		history := tab.Doc.Buffer.History()
		tester.Key(0, ui.KeyEnter)
		if tab.Doc.Buffer.String() != before || tab.Doc.Buffer.History() != history ||
			tab.Doc.Buffer.Selection() != beforeSelection {
			t.Fatalf("readonly replacement changed source: text=%q history=%+v selection=%+v want=%+v",
				tab.Doc.Buffer.String(), tab.Doc.Buffer.History(), tab.Doc.Buffer.Selection(), beforeSelection)
		}
	})

	t.Run("composition", func(t *testing.T) {
		a, tab := newSearchTextApp(t, "needle needle")
		a.Query = "needle"
		a.ShowFind = true
		setSearchSelection(t, tab, 0, 0)
		a.find(true)
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		if err := tester.Click("替换内容"); err != nil {
			t.Fatal(err)
		}
		before := tab.Doc.Buffer.String()
		tester.Compose("match", 5)
		if !a.FindReplacementComposing {
			t.Fatal("replacement composition was not routed to Find")
		}
		composingAccess, ok := tester.AccessByRole(ui.RoleTextField, "替换内容")
		if !ok || composingAccess.Value != "" {
			t.Fatalf("replacement composition committed early: %#v", composingAccess)
		}
		tester.Key(0, ui.KeyEnter)
		tester.Key(0, ui.KeyEscape)
		if !a.ShowFind || !a.FindReplacementComposing || tab.Doc.Buffer.String() != before ||
			!tester.Focused("替换内容") {
			t.Fatalf("composing keys crossed replacement: show=%v composing=%v text=%q focus=%v",
				a.ShowFind, a.FindReplacementComposing, tab.Doc.Buffer.String(), tester.Focused("替换内容"))
		}
		afterKeys, ok := tester.AccessByRole(ui.RoleTextField, "替换内容")
		if !ok || afterKeys.Value != composingAccess.Value {
			t.Fatalf("composing replacement keys changed committed text: before=%#v after=%#v", composingAccess, afterKeys)
		}
		if err := tester.Click("替换当前"); err != nil {
			t.Fatal(err)
		}
		if !a.ShowFind || !a.FindReplacementComposing || tab.Doc.Buffer.String() != before ||
			!tester.Focused("替换内容") {
			t.Fatalf("pointer replacement crossed composition: show=%v composing=%v text=%q focus=%v",
				a.ShowFind, a.FindReplacementComposing, tab.Doc.Buffer.String(), tester.Focused("替换内容"))
		}
		tester.Type("match")
		tester.Frame()
		if a.FindReplacementComposing || a.Replacement != "match" {
			t.Fatalf("replacement composition commit: composing=%v value=%q", a.FindReplacementComposing, a.Replacement)
		}
		tester.Key(0, ui.KeyEnter)
		if tab.Doc.Buffer.String() != "match needle" || !tester.Focused("替换内容") {
			t.Fatalf("post-composition replacement = %q focus=%v", tab.Doc.Buffer.String(), tester.Focused("替换内容"))
		}
	})
}

func TestFindKeyboardQueryCompositionAndPointerActionsDoNotCross(t *testing.T) {
	a, tab := newSearchTextApp(t, "needle needle")
	a.ShowFind = true
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if err := tester.Click("查找内容"); err != nil {
		t.Fatal(err)
	}
	beforeSelection := tab.Doc.Buffer.Selection()
	tester.Compose("needle", 6)
	if !a.FindQueryComposing {
		t.Fatal("query composition was not routed to Find")
	}
	composingAccess, ok := tester.AccessByRole(ui.RoleTextField, "查找内容")
	if !ok || composingAccess.Value != "" {
		t.Fatalf("query composition committed early: %#v", composingAccess)
	}
	tester.Key(0, ui.KeyEnter)
	if tab.Doc.Buffer.Selection() != beforeSelection {
		t.Fatalf("Enter crossed query composition: %+v", tab.Doc.Buffer.Selection())
	}
	afterEnter, ok := tester.AccessByRole(ui.RoleTextField, "查找内容")
	if !ok || afterEnter.Value != composingAccess.Value {
		t.Fatalf("composing Enter changed committed query: before=%#v after=%#v", composingAccess, afterEnter)
	}
	if err := tester.Click("下一个"); err != nil {
		t.Fatal(err)
	}
	if !a.ShowFind || !a.FindQueryComposing || tab.Doc.Buffer.Selection() != beforeSelection ||
		!tester.Focused("查找内容") {
		t.Fatalf("pointer action crossed query composition: show=%v composing=%v selection=%+v focus=%v",
			a.ShowFind, a.FindQueryComposing, tab.Doc.Buffer.Selection(), tester.Focused("查找内容"))
	}
	if err := tester.Click("查找"); err != nil {
		t.Fatal(err)
	}
	if !a.ShowFind || !a.FindQueryComposing || !tester.Focused("查找内容") {
		t.Fatalf("toolbar close crossed query composition: show=%v composing=%v focus=%v",
			a.ShowFind, a.FindQueryComposing, tester.Focused("查找内容"))
	}
	if err := tester.Click("关闭查找"); err != nil {
		t.Fatal(err)
	}
	if !a.ShowFind || !a.FindQueryComposing || !tester.Focused("查找内容") {
		t.Fatalf("pointer close crossed query composition: show=%v composing=%v focus=%v",
			a.ShowFind, a.FindQueryComposing, tester.Focused("查找内容"))
	}
	tester.Type("needle")
	tester.Frame()
	if a.FindQueryComposing || a.Query != "needle" {
		t.Fatalf("query composition commit: composing=%v value=%q", a.FindQueryComposing, a.Query)
	}
	tester.Key(0, ui.KeyEnter)
	assertSearchSelection(t, tab, 0, 6)
}

func TestFindKeyboardEscapeClosesFromFindButtonAndPreviewStaysUnfocused(t *testing.T) {
	t.Run("button", func(t *testing.T) {
		a, _ := newSearchTextApp(t, "needle")
		a.Query = "needle"
		a.ShowFind = true
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		if err := tester.Click("查找内容"); err != nil {
			t.Fatal(err)
		}
		tester.Key(0, ui.KeyTab)
		if !tester.Focused("上一个") {
			t.Fatalf("Tab did not focus Find action: texts=%q", tester.Texts())
		}
		tester.Key(0, ui.KeyEscape)
		if a.ShowFind || !tester.Focused("文档编辑区") {
			t.Fatalf("button Escape: show=%v source-focused=%v", a.ShowFind, tester.Focused("文档编辑区"))
		}
	})

	t.Run("preview", func(t *testing.T) {
		a := newTestApp(t, t.TempDir())
		a.NewDocument("markdown")
		tab := a.Current()
		if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), "# 标题\n\n正文 needle\n"); err != nil {
			t.Fatal(err)
		}
		a.Query = "needle"
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		tester.Key(ui.Cmd, ui.KeyF)
		if !a.ShowFind || !tester.Focused("查找内容") {
			t.Fatalf("preview Find did not focus query: show=%v focus=%v", a.ShowFind, tester.Focused("查找内容"))
		}
		before := tab.Doc.Buffer.String()
		tester.Key(0, ui.KeyEscape)
		if a.ShowFind || tab.Mode != "preview" || tester.Focused("文档编辑区") {
			t.Fatalf("preview Escape: show=%v mode=%q source-focused=%v",
				a.ShowFind, tab.Mode, tester.Focused("文档编辑区"))
		}
		tester.Type("!")
		tester.Frame()
		if tab.Doc.Buffer.String() != before {
			t.Fatalf("preview Escape activated an invisible source: %q", tab.Doc.Buffer.String())
		}
	})
}

func TestFindKeyboardRealMarkdownCompositionBlocksOpen(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.NewDocument("markdown")
	tab := a.Current()
	if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), "# 标题\n\n原生写作\n"); err != nil {
		t.Fatal(err)
	}
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	beginRealMarkdownComposition(t, tab, tester)
	tester.Key(ui.Cmd, ui.KeyF)
	if a.ShowFind || !strings.Contains(a.Message, "输入法正在组合") {
		t.Fatalf("real Markdown composition did not block Find: show=%v message=%q", a.ShowFind, a.Message)
	}
	tab.Editor.UnmarkText()
}

func TestReplaceCurrentRejectsIdentityChangingFlush(t *testing.T) {
	a, tab := newSearchTextApp(t, "needle needle")
	oldBuffer := tab.Doc.Buffer
	nextDoc := document.New("match needle")
	a.Query = "needle"
	a.Replacement = "match"
	a.ShowFind = true
	setSearchSelection(t, tab, 0, 0)
	a.find(true)
	beforeSelection := oldBuffer.Selection()
	nextSelection := nextDoc.Buffer.Selection()
	view := &searchIdentityChangingView{tab: tab, nextDoc: nextDoc}
	tab.View = view
	tab.Mode = "preview"
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if err := tester.Click("替换内容"); err != nil {
		t.Fatal(err)
	}
	tester.Key(0, ui.KeyEnter)
	if view.flushes != 1 || tab.Doc != nextDoc || oldBuffer.String() != "needle needle" ||
		nextDoc.Buffer.String() != "match needle" || oldBuffer.Selection() != beforeSelection ||
		tab.Doc.Buffer.Selection() != nextSelection {
		t.Fatalf("identity-changing replace acted on stale target: flushes=%d old=%q new=%q current=%p",
			view.flushes, oldBuffer.String(), nextDoc.Buffer.String(), tab.Doc)
	}
	if a.findNavigation.tab != nil || a.FindIndex != -1 {
		t.Fatalf("identity-changing replace retained stale Find cache: %#v index=%d", a.findNavigation, a.FindIndex)
	}
	if a.findFocus != findFocusNone || a.findFocusTab != nil || a.findFocusDoc != nil ||
		a.findFocusEditor != nil || a.findFocusBuffer != nil {
		t.Fatalf("identity-changing replace retained stale focus request: target=%v tab=%p doc=%p editor=%p buffer=%p",
			a.findFocus, a.findFocusTab, a.findFocusDoc, a.findFocusEditor, a.findFocusBuffer)
	}
}
