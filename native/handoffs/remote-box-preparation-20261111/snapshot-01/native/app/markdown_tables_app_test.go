package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
)

const (
	markdownTablesBody = "# 表格工作区\r\n\r\n" +
		"表格前的正文\r\n\r\n" +
		"| 名称 | 说明 |\r\n" +
		"| :--- | ---: |\r\n" +
		"| 甲 | **字面** |\r\n" +
		"| 🙂 | [链接](https://example.com) |\r\n\r\n" +
		"表格后的正文\r\n"
	markdownTablesReplacementBody = "| 名称 | 说明 |\r\n" +
		"| --- | --- |\r\n" +
		"| 乙 | 已修改🙂 |\r\n"
	markdownTablesScreenshotBody = "| 名称 | 说明 |\r\n" +
		"| :--- | ---: |\r\n" +
		"| 甲 | **字面** |\r\n" +
		"| 🙂 | [链接](https://example.com) |\r\n\r\n" +
		"表格后的正文\r\n"
	markdownTableEditLabel = "编辑表格源码"
)

func markdownTablesFixture() []byte {
	return append([]byte{0xef, 0xbb, 0xbf}, []byte(markdownTablesBody)...)
}

func openRealMarkdownTables(t *testing.T, width, height int) (*App, *Tab, *ui.Tester, string) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "tables.md")
	before := markdownTablesFixture()
	if err := os.WriteFile(path, before, 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "markdown" ||
		tab.Mode != "preview" || tab.Doc == nil || tab.Doc.Buffer == nil {
		t.Fatalf("real Markdown table fixture did not open in preview: %#v", tab)
	}
	tester := ui.NewTester(a.View, width, height)
	tester.Frame()
	tester.Frame()
	return a, tab, tester, path
}

func requireMarkdownTableButton(t *testing.T, tester *ui.Tester) ui.AccessInfo {
	t.Helper()
	info, ok := tester.AccessByRole(ui.RoleButton, markdownTableEditLabel)
	if !ok {
		t.Fatalf("table source action %q is missing: %#v", markdownTableEditLabel, tester.Texts())
	}
	if info.Disabled || !info.Focusable || info.Offscreen ||
		info.Bounds.W <= 0 || info.Bounds.H <= 0 {
		t.Fatalf("table source action state/geometry = %#v", info)
	}
	return info
}

func focusMarkdownTableButton(t *testing.T, tester *ui.Tester, info ui.AccessInfo) {
	t.Helper()
	tester.Press(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tester.CancelPress()
	tester.Frame()
	if tester.Focused(markdownTableEditLabel) {
		return
	}
	for attempt := 0; attempt < 120; attempt++ {
		tester.Key(0, ui.KeyTab)
		tester.Frame()
		if tester.Focused(markdownTableEditLabel) {
			return
		}
	}
	t.Fatalf("table source action could not receive keyboard focus: %#v", tester.Texts())
}

func activateMarkdownTableButton(t *testing.T, tester *ui.Tester, gesture string) {
	t.Helper()
	info := requireMarkdownTableButton(t, tester)
	switch gesture {
	case "pointer":
		tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	case "enter":
		focusMarkdownTableButton(t, tester, info)
		tester.Key(0, ui.KeyEnter)
	case "space":
		focusMarkdownTableButton(t, tester, info)
		tester.Key(0, ui.KeySpace)
	default:
		t.Fatalf("unknown table activation gesture %q", gesture)
	}
	tester.Frame()
	tester.Frame()
	editorInfo, ok := tester.AccessByRole(ui.RoleTextField, "文档编辑区")
	if !ok {
		t.Fatalf("%s did not open the whole-table source editor: %#v", gesture, tester.Texts())
	}
	if editorInfo.Bounds.W <= 0 || editorInfo.Bounds.H <= 0 || editorInfo.Offscreen ||
		editorInfo.ReadOnly {
		t.Fatalf("%s table source editor state/geometry = %#v", gesture, editorInfo)
	}
}

func replaceOpenMarkdownTableSource(t *testing.T, tester *ui.Tester, source string) {
	t.Helper()
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatalf("could not focus the table source editor: %v", err)
	}
	tester.Frame()
	tester.Command("selectAll")
	tester.Type(source)
	tester.Frame()
}

func TestMarkdownTablesRealAdapterPreviewBOMCRLFSaveReopen(t *testing.T) {
	before := markdownTablesFixture()
	a, tab, tester, path := openRealMarkdownTables(t, 1180, 780)
	if tab.Doc.Dirty() {
		t.Fatal("opening the table fixture made it dirty")
	}
	if history := tab.Doc.Buffer.History(); history.UndoDepth != 0 || history.RedoDepth != 0 {
		t.Fatalf("opening the table fixture added history: %#v", history)
	}
	requireMarkdownTableButton(t, tester)
	texts := strings.Join(tester.Texts(), "\n")
	if !strings.Contains(texts, "表格前的正文") || !strings.Contains(texts, "表格后的正文") ||
		(!strings.Contains(texts, "表头 · 第 1 行") && !strings.Contains(texts, "**字面**")) {
		t.Fatalf("literal table preview or neighboring prose is missing: %q", texts)
	}
	if _, ok := tester.AccessByRole(ui.RoleLink, ""); ok {
		t.Fatal("a Markdown link inside a literal table cell became an active link")
	}
	if got := mustReadFile(t, path); !bytes.Equal(got, before) {
		t.Fatal("opening the table fixture changed source bytes")
	}

	history := tab.Doc.Buffer.History()
	if !a.Save(tab, false) {
		t.Fatalf("unchanged table Save failed: %s", a.Message)
	}
	if tab.Mode != "preview" {
		t.Fatalf("unchanged table Save changed mode to %q", tab.Mode)
	}
	if tab.Doc.Buffer.History() != history {
		t.Fatalf("unchanged table Save changed history: %#v -> %#v", history, tab.Doc.Buffer.History())
	}
	if got := mustReadFile(t, path); !bytes.Equal(got, before) {
		t.Fatal("unchanged table Save did not preserve BOM and CRLF bytes")
	}

	reopened := newTestApp(t, filepath.Join(t.TempDir(), "config"))
	reopened.Open(path)
	reopenedTab := reopened.Current()
	reopenedTester := ui.NewTester(reopened.View, 1180, 780)
	reopenedTester.Frame()
	reopenedTester.Frame()
	if reopenedTab == nil || reopenedTab.Adapter == nil || reopenedTab.Adapter.ID != "markdown" ||
		reopenedTab.Mode != "preview" {
		t.Fatalf("saved table did not reopen through the registered Markdown adapter: %#v", reopenedTab)
	}
	requireMarkdownTableButton(t, reopenedTester)
	if got := mustReadFile(t, path); !bytes.Equal(got, before) {
		t.Fatal("reopening changed the preserved table bytes")
	}
}

func TestMarkdownTablesNativeGesturesFlushPendingComposeOnce(t *testing.T) {
	for _, gesture := range []string{"pointer", "enter", "space"} {
		t.Run(gesture, func(t *testing.T) {
			a, tab, tester, path := openRealMarkdownTables(t, 1180, 780)
			beginRealMarkdownCompositionAt(t, tab, tester, "表格工作区")
			tester.Frame()
			if strings.Contains(tab.Doc.Buffer.String(), "# 中文🙂") {
				t.Fatal("pending heading composition reached the host before table activation")
			}

			activateMarkdownTableButton(t, tester, gesture)
			want := strings.Replace(markdownTablesBody, "# 表格工作区", "# 中文🙂", 1)
			if got := tab.Doc.Buffer.String(); got != want {
				t.Fatalf("%s table activation did not Flush exactly the pending block: %q", gesture, got)
			}
			if history := tab.Doc.Buffer.History(); history.UndoDepth != 1 || history.RedoDepth != 0 {
				t.Fatalf("%s table activation Flush was not one history transaction: %#v", gesture, history)
			}
			if got := mustReadFile(t, path); !bytes.Equal(got, markdownTablesFixture()) {
				t.Fatalf("%s table activation unexpectedly wrote the Flush to disk", gesture)
			}
			if a.Message != "" && strings.Contains(a.Message, "草稿尚未") {
				t.Fatalf("%s table activation refused its valid Flush: %q", gesture, a.Message)
			}
		})
	}
}

func TestMarkdownTablesWholeSourceEditUndoRedoSaveFallback(t *testing.T) {
	a, tab, tester, path := openRealMarkdownTables(t, 1180, 780)
	activateMarkdownTableButton(t, tester, "pointer")
	brokenTail := markdownTablesReplacementBody +
		"| 额外 |\r\n"
	replaceOpenMarkdownTableSource(t, tester, brokenTail)
	wantBroken := strings.Replace(markdownTablesBody, "| 名称 | 说明 |\r\n| :--- | ---: |\r\n| 甲 | **字面** |\r\n| 🙂 | [链接](https://example.com) |", strings.TrimSuffix(brokenTail, "\r\n"), 1)
	if got := tab.Doc.Buffer.String(); got != wantBroken {
		t.Fatalf("whole-table editing did not preserve neighboring source:\n got %q\nwant %q", got, wantBroken)
	}
	if history := tab.Doc.Buffer.History(); history.UndoDepth != 1 || history.RedoDepth != 0 {
		t.Fatalf("whole-table edit was not one source transaction: %#v", history)
	}

	if err := tester.Click("完成编辑"); err != nil {
		t.Fatalf("could not finish whole-table editing: %v", err)
	}
	tester.Frame()
	tester.Frame()
	if _, ok := tester.AccessByRole(ui.RoleButton, markdownTableEditLabel); ok {
		t.Fatal("structural table edit still exposed a valid-table action")
	}
	if texts := strings.Join(tester.Texts(), "\n"); !strings.Contains(texts, "表格未展开") ||
		!strings.Contains(texts, "完整") {
		t.Fatalf("structural whole-table edit did not expose truthful raw fallback: %q", texts)
	}

	tab.Editor.Command("undo")
	tester.Frame()
	tester.Frame()
	if tab.Doc.Buffer.String() != markdownTablesBody {
		t.Fatal("one undo did not restore the original table source")
	}
	requireMarkdownTableButton(t, tester)
	tab.Editor.Command("redo")
	tester.Frame()
	tester.Frame()
	if tab.Doc.Buffer.String() != wantBroken {
		t.Fatal("redo did not restore the intentional structural source edit")
	}
	if !a.Save(tab, false) {
		t.Fatalf("saving the intentionally invalid table source failed: %s", a.Message)
	}
	wantDisk := append([]byte{0xef, 0xbb, 0xbf}, []byte(wantBroken)...)
	if got := mustReadFile(t, path); !bytes.Equal(got, wantDisk) {
		t.Fatalf("saved fallback table differs from exact source bytes:\n got %q\nwant %q", got, wantDisk)
	}

	reopened := newTestApp(t, filepath.Join(t.TempDir(), "config"))
	reopened.Open(path)
	reopenedTab := reopened.Current()
	reopenedTester := ui.NewTester(reopened.View, 1180, 780)
	reopenedTester.Frame()
	reopenedTester.Frame()
	if reopenedTab == nil || reopenedTab.Adapter == nil || reopenedTab.Adapter.ID != "markdown" {
		t.Fatalf("raw-fallback table did not reopen through Markdown: %#v", reopenedTab)
	}
	if texts := strings.Join(reopenedTester.Texts(), "\n"); !strings.Contains(texts, "表格未展开") {
		t.Fatalf("reopened raw fallback lost its warning: %q", texts)
	}
	if got := mustReadFile(t, path); !bytes.Equal(got, wantDisk) {
		t.Fatal("reopening or rendering the raw fallback changed source bytes")
	}
}

func TestMarkdownTablesDraftFlushBarriers(t *testing.T) {
	t.Run("save", func(t *testing.T) {
		a, tab, tester, path := openRealMarkdownTables(t, 1180, 780)
		activateMarkdownTableButton(t, tester, "pointer")
		replaceOpenMarkdownTableSource(t, tester, markdownTablesReplacementBody)
		if !a.Save(tab, false) {
			t.Fatalf("Save did not flush the table draft: %s", a.Message)
		}
		want := strings.Replace(markdownTablesBody,
			"| 名称 | 说明 |\r\n| :--- | ---: |\r\n| 甲 | **字面** |\r\n| 🙂 | [链接](https://example.com) |",
			strings.TrimSuffix(markdownTablesReplacementBody, "\r\n"), 1)
		wantDisk := append([]byte{0xef, 0xbb, 0xbf}, []byte(want)...)
		if got := mustReadFile(t, path); !bytes.Equal(got, wantDisk) {
			t.Fatalf("Save table bytes:\n got %q\nwant %q", got, wantDisk)
		}
	})

	t.Run("mode", func(t *testing.T) {
		a, tab, tester, _ := openRealMarkdownTables(t, 1180, 780)
		activateMarkdownTableButton(t, tester, "pointer")
		replaceOpenMarkdownTableSource(t, tester, markdownTablesReplacementBody)
		if !a.setMode(tab, "source") || tab.Mode != "source" {
			t.Fatalf("source barrier did not flush the table draft: %s", a.Message)
		}
		if !strings.Contains(tab.Doc.Buffer.String(), markdownTablesReplacementBody) ||
			strings.Contains(tab.Doc.Buffer.String(), "| 甲 | **字面** |") {
			t.Fatal("mode switch replaced or discarded a pending whole-table edit")
		}
	})

	t.Run("tab", func(t *testing.T) {
		a, tab, tester, _ := openRealMarkdownTables(t, 1180, 780)
		a.NewDocument("text")
		if !a.Select(0) {
			t.Fatal("could not return to the table tab")
		}
		tester.Frame()
		activateMarkdownTableButton(t, tester, "pointer")
		replaceOpenMarkdownTableSource(t, tester, markdownTablesReplacementBody)
		if !a.Select(1) {
			t.Fatalf("tab barrier refused a valid table draft flush: %s", a.Message)
		}
		if !strings.Contains(tab.Doc.Buffer.String(), markdownTablesReplacementBody) ||
			strings.Contains(tab.Doc.Buffer.String(), "| 甲 | **字面** |") {
			t.Fatal("tab switch replaced or discarded a pending whole-table edit")
		}
	})

	t.Run("close", func(t *testing.T) {
		a, tab, tester, _ := openRealMarkdownTables(t, 1180, 780)
		activateMarkdownTableButton(t, tester, "pointer")
		replaceOpenMarkdownTableSource(t, tester, markdownTablesReplacementBody)
		buffer := tab.Doc.Buffer
		a.closeDialog = func(*Tab) (closeDecision, error) { return closeDiscard, nil }
		if !a.CloseTab(0) {
			t.Fatalf("close barrier refused after flushing the table draft: %s", a.Message)
		}
		if len(a.Tabs) != 0 {
			t.Fatalf("close barrier left the table tab open: %d", len(a.Tabs))
		}
		if !strings.Contains(buffer.String(), markdownTablesReplacementBody) {
			t.Fatal("close removed the table before its pending draft was flushed")
		}
	})
}

func TestMarkdownTablesFailedFlushReadonlyFallbackAndExport(t *testing.T) {
	t.Run("failed-flush", func(t *testing.T) {
		before := markdownTablesFixture()
		a, tab, tester, path := openRealMarkdownTables(t, 1180, 780)
		beginRealMarkdownCompositionAt(t, tab, tester, "表格工作区")
		tester.Frame()
		if err := tab.Doc.Buffer.ReplaceUTF16(2, 4, "外部"); err != nil {
			t.Fatal(err)
		}
		tester.Frame()
		version, selection, history := tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection(), tab.Doc.Buffer.History()
		source := tab.Doc.Buffer.String()
		view := tab.View
		button := requireMarkdownTableButton(t, tester)
		tester.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
		tester.Frame()
		if tab.View != view || tab.Mode != "preview" ||
			tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.Selection() != selection ||
			tab.Doc.Buffer.History() != history || tab.Doc.Buffer.String() != source {
			t.Fatal("failed table Flush replaced the view or changed source state")
		}
		if got := mustReadFile(t, path); !bytes.Equal(got, before) {
			t.Fatal("failed table Flush changed disk bytes")
		}
		if !strings.Contains(a.Message, "草稿仍保留") ||
			!tester.HasText("完成编辑") || !tester.HasText("打开未同步副本") {
			t.Fatalf("failed table Flush did not expose recoverable state: %q %#v", a.Message, tester.Texts())
		}
	})

	t.Run("readonly", func(t *testing.T) {
		a := newTestApp(t, t.TempDir())
		readonlySource := strings.NewReplacer(
			"| 名称 | 说明 |", "| 名称 | 说明 | 备注 |",
			"| :--- | ---: |", "| :--- | ---: | --- |",
			"| 甲 | **字面** |", "| 甲 | **字面** | 只读 |",
			"| 🙂 | [链接](https://example.com) |", "| 🙂 | [链接](https://example.com) | 第三列 |",
		).Replace(markdownTablesBody)
		doc := document.New(readonlySource)
		doc.ReadOnly = true
		tab := a.add(doc, "readonly.md", "")
		if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "markdown" || tab.Mode != "preview" {
			t.Fatalf("readonly table did not mount the registered Markdown adapter: %#v", tab)
		}
		tester := ui.NewTester(a.View, 360, 780)
		tester.Frame()
		tester.Frame()
		texts := strings.Join(tester.Texts(), "\n")
		if _, ok := tester.AccessByRole(ui.RoleButton, markdownTableEditLabel); ok ||
			!strings.Contains(texts, "只读表格") || !strings.Contains(texts, "完整源码") ||
			!strings.Contains(texts, "表头 · 第 1 行") {
			t.Fatalf("readonly table controls or guidance = %q", texts)
		}
		cell, ok := tester.AccessByRole(ui.RoleCell, "数据，第 2 行，第 2 列：**字面**")
		if !ok || cell.Bounds.W <= 0 || cell.Bounds.H <= 0 {
			t.Fatalf("readonly table cell is unavailable: %#v texts=%q", cell, tester.Texts())
		}
		tester.ClickAt(cell.Bounds.X+cell.Bounds.W/2, cell.Bounds.Y+cell.Bounds.H/2)
		tester.Frame()
		tester.ClickAt(cell.Bounds.X+cell.Bounds.W/2, cell.Bounds.Y+cell.Bounds.H/2)
		tester.Frame()
		if _, ok := tester.AccessByRole(ui.RoleTextField, "文档编辑区"); ok ||
			tab.Doc.Buffer.String() != readonlySource ||
			tab.Doc.Buffer.CanUndo() || tab.Doc.Buffer.CanRedo() {
			t.Fatal("readonly table double-click created an editor or history")
		}
	})

	t.Run("malformed-and-over-limit", func(t *testing.T) {
		malformed := "# 失败候选\r\n\r\n| A | B |\r\n| --- | -- |\r\n| C | D |\r\n"
		var overlarge strings.Builder
		overlarge.WriteString("# 超限表格\r\n\r\n| A | B |\r\n| --- | --- |\r\n")
		for index := 0; index <= 256; index++ {
			fmt.Fprintf(&overlarge, "| %03d | %s |\r\n", index, strings.Repeat("x", 220))
		}
		if overlarge.Len() <= 32<<10 {
			t.Fatalf("over-limit fixture is too small: %d bytes", overlarge.Len())
		}
		for _, tc := range []struct{ name, source string }{
			{name: "malformed", source: malformed},
			{name: "over-limit", source: overlarge.String()},
		} {
			t.Run(tc.name, func(t *testing.T) {
				a := newTestApp(t, t.TempDir())
				tab := a.add(document.New(tc.source), tc.name+".md", "")
				if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "markdown" {
					t.Fatalf("%s did not use the registered Markdown adapter: %#v", tc.name, tab)
				}
				tester := ui.NewTester(a.View, 1180, 780)
				tester.Frame()
				tester.Frame()
				if len(tab.ViewWarnings) == 0 {
					t.Fatalf("%s fallback did not expose a warning", tc.name)
				}
				if tab.Doc.Buffer.String() != tc.source {
					t.Fatalf("%s preview changed complete raw source", tc.name)
				}
				if !a.setMode(tab, "source") || tab.Doc.Buffer.String() != tc.source {
					t.Fatalf("%s source route did not preserve the full document: %s", tc.name, a.Message)
				}
			})
		}
	})

	t.Run("literal-html-export", func(t *testing.T) {
		hostile := "# 导出安全\r\n\r\n" +
			"| A | B |\r\n| --- | --- |\r\n" +
			"| <img src=x onerror=alert(1)> | [链接](https://example.com) |\r\n"
		a := newTestApp(t, t.TempDir())
		tab := a.add(document.New(hostile), "literal-export.md", "")
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		tester.Frame()
		if _, ok := tester.AccessByRole(ui.RoleLink, ""); ok {
			t.Fatal("literal table cell Markdown exposed an active native link")
		}
		version, history := tab.Doc.Buffer.Version(), tab.Doc.Buffer.History()
		html, err := tab.View.Export("html")
		if err != nil {
			t.Fatalf("valid literal table export failed: %v", err)
		}
		if !bytes.Contains(html, []byte("<table>")) ||
			!bytes.Contains(html, []byte("&lt;img src=x onerror=alert(1)&gt;")) ||
			!bytes.Contains(html, []byte("[链接](https://example.com)")) ||
			bytes.Contains(html, []byte("<img")) || bytes.Contains(html, []byte("href=")) {
			t.Fatalf("literal table HTML was not safe/verbatim:\n%s", html)
		}
		if tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.History() != history ||
			tab.Doc.Buffer.String() != hostile {
			t.Fatal("literal table export changed source state")
		}

		fallback := "# 原样回退\r\n\r\n| A | B |\r\n| --- | -- |\r\n| <script>x</script> | [x](y) |\r\n"
		fallbackTab := a.add(document.New(fallback), "fallback-export.md", "")
		a.Current().Mode = "preview"
		tester.Frame()
		tester.Frame()
		html, err = fallbackTab.View.Export("html")
		if err != nil {
			t.Fatalf("fallback table export failed: %v", err)
		}
		if !bytes.Contains(html, []byte("<pre>")) ||
			!bytes.Contains(html, []byte("&lt;script&gt;")) ||
			bytes.Contains(html, []byte("<script>")) ||
			bytes.Contains(html, []byte("<a ")) {
			t.Fatalf("fallback table HTML was unsafe or incomplete:\n%s", html)
		}
		if fallbackTab.Doc.Buffer.String() != fallback {
			t.Fatal("fallback table export changed complete raw source")
		}
	})
}

func TestMarkdownTablesProseLayoutAndNarrowScreenshots(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(workspace, "tables.md")
	if err := os.WriteFile(path, markdownTablesFixture(), 0600); err != nil {
		t.Fatal(err)
	}
	screenshotPath := filepath.Join(workspace, "table-view.md")
	if err := os.WriteFile(screenshotPath, []byte(markdownTablesScreenshotBody), 0600); err != nil {
		t.Fatal(err)
	}

	layoutApp := newTestApp(t, filepath.Join(root, "config-layout"))
	layoutApp.Open(path)
	layoutTester := ui.NewTester(layoutApp.View, 1180, 780)
	layoutTester.Frame()
	layoutTester.Frame()
	for _, prose := range []string{"表格前的正文", "表格后的正文"} {
		rect, ok := layoutTester.Find(prose)
		if !ok || rect.W <= 0 || rect.H <= 0 || rect.X < 0 || rect.X+rect.W > 1180 ||
			rect.Y < 0 || rect.Y+rect.H > 780-29 {
			t.Fatalf("distant table changed ordinary prose geometry for %q: %#v", prose, rect)
		}
	}

	out := os.Getenv("LEVNOTE_MARKDOWN_TABLES_SCREENSHOT_DIR")
	if out == "" {
		t.Skip("set LEVNOTE_MARKDOWN_TABLES_SCREENSHOT_DIR for integrated table screenshots")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	for _, dark := range []bool{false, true} {
		a := newTestApp(t, filepath.Join(root, fmt.Sprintf("config-%t", dark)))
		a.Root = workspace
		a.refreshTreeNow()
		a.Open(screenshotPath)
		tab := a.Current()
		if tab == nil || !a.setMode(tab, "split") {
			t.Fatalf("could not open the table fixture in split mode: %#v %q", tab, a.Message)
		}
		a.Dark = dark
		tester := ui.NewTester(a.View, 760, 480)
		tester.SetDark(dark)
		tester.Frame()
		tester.Frame()
		tester.Frame()
		if !tester.HasText("workspace") || !tester.HasText("table-view.md") ||
			!tester.HasText("表头 · 第 1 行") || !tester.HasText("**字面**") {
			t.Fatalf("theme=%v integrated table fixture lacks sidebar or narrow table: %#v", dark, tester.Texts())
		}
		button := requireMarkdownTableButton(t, tester)
		const statusTop = float32(480 - 29)
		if button.Bounds.X < 0 || button.Bounds.Y < 0 ||
			button.Bounds.X+button.Bounds.W > 760 || button.Bounds.Y+button.Bounds.H > statusTop {
			t.Fatalf("theme=%v table action overlaps app chrome/status: %#v", dark, button.Bounds)
		}
		bodyCell, ok := tester.Find("**字面**")
		if !ok || bodyCell.W <= 0 || bodyCell.H <= 0 || bodyCell.X < 0 || bodyCell.Y < 0 ||
			bodyCell.X+bodyCell.W > 760 || bodyCell.Y+bodyCell.H > statusTop {
			t.Fatalf("theme=%v representative table body cell is clipped: %#v", dark, bodyCell)
		}
		name := "markdown-tables-app-split-light-760x480.png"
		if dark {
			name = "markdown-tables-app-split-dark-760x480.png"
		}
		captureAcceptanceTester(t, filepath.Join(out, name), tester)
	}
}
