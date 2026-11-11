package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/session"
)

const (
	markdownCellStagedNotice = "单元格草稿尚未写回；保存或完成编辑时写回"
	markdownCellFinishLabel  = "完成编辑"
	markdownCellDiscardLabel = "放弃草稿"
	markdownCellRecoverLabel = "打开未同步副本"
	markdownCellHeaderName   = "表头，第 1 行，第 1 列：名称"
	markdownCellBodyName     = "数据，第 2 行，第 1 列：甲"
	markdownCellHeaderEditor = "编辑单元格 · 表头 · 第 1 行 · 第 1 列"
	markdownCellBodyEditor   = "编辑单元格 · 数据 · 第 2 行 · 第 1 列"
)

func requireMarkdownCellAccess(t *testing.T, tester *ui.Tester, role ui.Role, name string) ui.AccessInfo {
	t.Helper()
	info, ok := tester.AccessByRole(role, name)
	if !ok {
		t.Fatalf("Markdown table role %v named %q is missing: %#v", role, name, tester.Texts())
	}
	if info.Name != name || !info.Focusable || info.Offscreen ||
		info.Bounds.W <= 0 || info.Bounds.H <= 0 {
		t.Fatalf("Markdown table cell role/state/geometry = %#v", info)
	}
	return info
}

func focusMarkdownCell(t *testing.T, tester *ui.Tester, info ui.AccessInfo) {
	t.Helper()
	x, y := info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2
	tester.Press(x, y)
	tester.CancelPress()
	tester.Frame()
	if tester.Focused(info.Name) {
		return
	}
	for attempt := 0; attempt < 200; attempt++ {
		tester.Key(0, ui.KeyTab)
		tester.Frame()
		if tester.Focused(info.Name) {
			return
		}
	}
	t.Fatalf("Markdown table cell %q could not receive keyboard focus", info.Name)
}

func requireMarkdownCellEditor(t *testing.T, tester *ui.Tester, label string) ui.AccessInfo {
	t.Helper()
	info, ok := tester.AccessByRole(ui.RoleTextField, label)
	if !ok {
		t.Fatalf("cell source editor %q is missing: %#v", label, tester.Texts())
	}
	if info.Disabled || info.ReadOnly || info.Offscreen ||
		info.Bounds.W <= 0 || info.Bounds.H <= 0 {
		t.Fatalf("cell source editor state/geometry = %#v", info)
	}
	return info
}

func openMarkdownCellKeyboard(t *testing.T, tester *ui.Tester, info ui.AccessInfo, editorLabel string) {
	t.Helper()
	focusMarkdownCell(t, tester, info)
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	tester.Frame()
	requireMarkdownCellEditor(t, tester, editorLabel)
}

func openMarkdownCellPointer(t *testing.T, tester *ui.Tester, info ui.AccessInfo, editorLabel string) {
	t.Helper()
	x, y := info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2
	tester.ClickAt(x, y)
	tester.Frame()
	tester.ClickAt(x, y)
	tester.Frame()
	tester.Frame()
	requireMarkdownCellEditor(t, tester, editorLabel)
}

func replaceMarkdownCellDraft(t *testing.T, tester *ui.Tester, editorLabel, value string) {
	t.Helper()
	info, ok := tester.AccessByRole(ui.RoleTextField, editorLabel)
	if !ok {
		t.Fatalf("cell editor %q disappeared before typing", editorLabel)
	}
	x := info.Bounds.X + min(float32(24), info.Bounds.W/2)
	y := info.Bounds.Y + min(float32(20), info.Bounds.H/2)
	tester.ClickAt(x, y)
	tester.Frame()
	tester.Frame()
	info, ok = tester.AccessByRole(ui.RoleTextField, editorLabel)
	if !ok {
		t.Fatalf("cell editor %q disappeared after pointer focus", editorLabel)
	}
	if !info.Focused {
		t.Fatalf("cell editor %q did not receive native pointer focus: %#v", editorLabel, info)
	}
	tester.Command("selectAll")
	tester.Compose(value, utf8.RuneCountInString(value))
	tester.Frame()
}

func markdownCellExpectedSource(value string) string {
	return strings.Replace(markdownTablesBody, "| 甲 | **字面** |", "| "+value+" | **字面** |", 1)
}

func TestMarkdownCellsRealAdapterStagedSessionHeaderBodyUndoRedoSaveReopen(t *testing.T) {
	a, tab, tester, path := openRealMarkdownTables(t, 1180, 780)
	buffer := tab.Doc.Buffer
	if tab.Adapter == nil || tab.Adapter.ID != "markdown" || tab.Mode != "preview" {
		t.Fatalf("cell fixture did not use the registered Markdown adapter: %#v", tab)
	}
	if history := buffer.History(); history.UndoDepth != 0 || history.RedoDepth != 0 {
		t.Fatalf("fresh cell fixture has history: %#v", history)
	}

	header := requireMarkdownCellAccess(t, tester, ui.RoleColumnHeader, markdownCellHeaderName)
	openMarkdownCellKeyboard(t, tester, header, markdownCellHeaderEditor)
	version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
	replaceMarkdownCellDraft(t, tester, markdownCellHeaderEditor, "称谓")
	if buffer.Version() != version || buffer.Selection() != selection ||
		buffer.History() != history || buffer.String() != markdownTablesBody || tab.Doc.Dirty() {
		t.Fatal("local header typing changed host source, selection, version, history or dirty state")
	}
	if !tester.HasText(markdownCellStagedNotice) ||
		!tester.HasText(markdownCellDiscardLabel) ||
		!tester.HasText(markdownCellFinishLabel) {
		t.Fatalf("staged header state is not fully visible: %#v", tester.Texts())
	}

	// Session persistence captures the host document. It must neither commit
	// nor claim to recover this view-local staged value.
	if err := a.persistTabs(a.Tabs, a.Active, "接受单元格草稿持久化边界测试"); err != nil {
		t.Fatalf("session snapshot failed while a cell draft was staged: %v", err)
	}
	snapshot, issues := session.Load(filepath.Join(a.configDir, "session.json"))
	if len(issues) != 0 || len(snapshot.Tabs) != 1 {
		t.Fatalf("session snapshot while staged = %#v issues=%#v", snapshot, issues)
	}
	if snapshot.Tabs[0].Draft != nil || snapshot.Tabs[0].Path != path {
		t.Fatalf("staged cell value was incorrectly captured: %#v", snapshot.Tabs[0])
	}
	sessionBytes := mustReadFile(t, filepath.Join(a.configDir, "session.json"))
	if bytes.Contains(sessionBytes, []byte("称谓")) {
		t.Fatal("session file silently captured the uncommitted cell value")
	}
	if buffer.String() != markdownTablesBody || tab.Doc.Dirty() {
		t.Fatal("session persistence committed the staged cell value")
	}

	if err := tester.Click(markdownCellFinishLabel); err != nil {
		t.Fatalf("could not finish the header cell: %v", err)
	}
	tester.Frame()
	tester.Frame()
	headerSource := strings.Replace(markdownTablesBody, "| 名称 | 说明 |", "| 称谓 | 说明 |", 1)
	if buffer.String() != headerSource || !tab.Doc.Dirty() {
		t.Fatalf("one header finish did not commit exact source: %q message=%q", buffer.String(), a.Message)
	}
	if got := buffer.History(); got.UndoDepth != 1 || got.RedoDepth != 0 {
		t.Fatalf("header finish was not one history transaction: %#v", got)
	}
	if _, ok := tester.AccessByRole(ui.RoleTextField, markdownCellHeaderEditor); ok {
		t.Fatal("successful header finish left the cell draft open")
	}

	tab.Editor.Command("undo")
	tester.Frame()
	tester.Frame()
	if buffer.String() != markdownTablesBody {
		t.Fatal("one undo did not restore the original header source")
	}
	tab.Editor.Command("redo")
	tester.Frame()
	tester.Frame()
	if buffer.String() != headerSource {
		t.Fatal("one redo did not restore the header source")
	}

	body := requireMarkdownCellAccess(t, tester, ui.RoleCell, markdownCellBodyName)
	openMarkdownCellPointer(t, tester, body, markdownCellBodyEditor)
	replaceMarkdownCellDraft(t, tester, markdownCellBodyEditor, "乙🙂")
	if buffer.String() != headerSource {
		t.Fatal("local body typing reached the host before finish")
	}
	if err := tester.Click(markdownCellFinishLabel); err != nil {
		t.Fatalf("could not finish the body cell: %v", err)
	}
	tester.Frame()
	tester.Frame()
	finalSource := strings.Replace(headerSource, "| 甲 | **字面** |", "| 乙🙂 | **字面** |", 1)
	if buffer.String() != finalSource {
		t.Fatalf("one body finish did not commit exact source: %q", buffer.String())
	}
	if got := buffer.History(); got.UndoDepth != 2 || got.RedoDepth != 0 {
		t.Fatalf("body finish was not one additional history transaction: %#v", got)
	}

	tab.Editor.Command("undo")
	tester.Frame()
	if buffer.String() != headerSource {
		t.Fatal("one undo did not isolate the body-cell transaction")
	}
	tab.Editor.Command("redo")
	tester.Frame()
	if buffer.String() != finalSource {
		t.Fatal("one redo did not restore the body-cell transaction")
	}

	if !a.Save(tab, false) {
		t.Fatalf("saving the committed cell value failed: %s", a.Message)
	}
	if tab.Mode != "preview" || tab.Doc.Buffer.History().UndoDepth != 2 {
		t.Fatalf("Save changed preview/history unexpectedly: mode=%q history=%#v", tab.Mode, tab.Doc.Buffer.History())
	}
	wantDisk := append([]byte{0xef, 0xbb, 0xbf}, []byte(finalSource)...)
	if got := mustReadFile(t, path); !bytes.Equal(got, wantDisk) {
		t.Fatalf("cell Save did not preserve BOM/CRLF and exact source:\n got %q\nwant %q", got, wantDisk)
	}

	reopened := newTestApp(t, filepath.Join(t.TempDir(), "config"))
	reopened.Open(path)
	reopenedTab := reopened.Current()
	reopenedTester := ui.NewTester(reopened.View, 1180, 780)
	reopenedTester.Frame()
	reopenedTester.Frame()
	if reopenedTab == nil || reopenedTab.Adapter == nil || reopenedTab.Adapter.ID != "markdown" ||
		reopenedTab.Doc.Buffer.String() != finalSource {
		t.Fatalf("saved cell source did not reopen through Markdown: %#v", reopenedTab)
	}
	requireMarkdownCellAccess(t, reopenedTester, ui.RoleColumnHeader, "表头，第 1 行，第 1 列：称谓")
	requireMarkdownCellAccess(t, reopenedTester, ui.RoleCell, "数据，第 2 行，第 1 列：乙🙂")
	if got := mustReadFile(t, path); !bytes.Equal(got, wantDisk) {
		t.Fatal("reopening the cell-edited table changed disk bytes")
	}
}

func TestMarkdownCellsRealAdapterBarriers(t *testing.T) {
	for _, tc := range []struct {
		name   string
		action func(*testing.T, *App, *Tab, *ui.Tester, string)
	}{
		{
			name: "save",
			action: func(t *testing.T, a *App, tab *Tab, tester *ui.Tester, _ string) {
				if !a.Save(tab, false) {
					t.Fatalf("Save barrier failed: %s", a.Message)
				}
				if tab.Doc.Buffer.String() != markdownCellExpectedSource("保存🙂") {
					t.Fatal("Save barrier did not commit the staged cell")
				}
			},
		},
		{
			name: "export",
			action: func(t *testing.T, a *App, tab *Tab, tester *ui.Tester, _ string) {
				if !a.flushTab(tab, "导出") {
					t.Fatalf("export barrier failed: %s", a.Message)
				}
				if _, err := tab.View.Export(tab.Adapter.Exports[0]); err != nil {
					t.Fatalf("HTML export after barrier failed: %v", err)
				}
				if tab.Doc.Buffer.String() != markdownCellExpectedSource("导出🙂") {
					t.Fatalf("export barrier did not commit the staged cell: %q %s", tab.Doc.Buffer.String(), a.Message)
				}
			},
		},
		{
			name: "source-mode",
			action: func(t *testing.T, a *App, tab *Tab, tester *ui.Tester, _ string) {
				if !a.setMode(tab, "source") {
					t.Fatalf("source-mode barrier failed: %s", a.Message)
				}
				if tab.Mode != "source" || tab.Doc.Buffer.String() != markdownCellExpectedSource("源码🙂") {
					t.Fatalf("source-mode barrier source/mode = %q/%q", tab.Doc.Buffer.String(), tab.Mode)
				}
			},
		},
		{
			name: "tab-switch",
			action: func(t *testing.T, a *App, tab *Tab, tester *ui.Tester, _ string) {
				a.NewDocument("text")
				if len(a.Tabs) != 2 {
					t.Fatalf("tab-switch fixture has %d tabs", len(a.Tabs))
				}
				if !a.Select(0) {
					t.Fatal("could not return to the staged table tab")
				}
				tester.Frame()
				body := requireMarkdownCellAccess(t, tester, ui.RoleCell, markdownCellBodyName)
				openMarkdownCellPointer(t, tester, body, markdownCellBodyEditor)
				replaceMarkdownCellDraft(t, tester, markdownCellBodyEditor, "切页🙂")
				if !a.Select(1) {
					t.Fatalf("tab-switch barrier failed: %s", a.Message)
				}
				if tab.Doc.Buffer.String() != markdownCellExpectedSource("切页🙂") {
					t.Fatal("tab-switch barrier did not commit the staged cell")
				}
			},
		},
		{
			name: "close",
			action: func(t *testing.T, a *App, tab *Tab, tester *ui.Tester, path string) {
				a.closeDialog = func(*Tab) (closeDecision, error) { return closeCancel, nil }
				if a.CloseTab(0) {
					t.Fatal("cancelled close unexpectedly removed the tab")
				}
				if tab.Doc.Buffer.String() != markdownCellExpectedSource("关闭🙂") {
					t.Fatal("close barrier did not commit the staged cell before asking")
				}
				if got := mustReadFile(t, path); !bytes.Equal(got, markdownTablesFixture()) {
					t.Fatal("cancelled close wrote the committed cell to disk")
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, tab, tester, path := openRealMarkdownTables(t, 1180, 780)
			value := map[string]string{
				"save": "保存🙂", "export": "导出🙂", "source-mode": "源码🙂",
				"tab-switch": "切页🙂", "close": "关闭🙂",
			}[tc.name]
			version, history := tab.Doc.Buffer.Version(), tab.Doc.Buffer.History()
			if tc.name != "tab-switch" {
				body := requireMarkdownCellAccess(t, tester, ui.RoleCell, markdownCellBodyName)
				openMarkdownCellPointer(t, tester, body, markdownCellBodyEditor)
				replaceMarkdownCellDraft(t, tester, markdownCellBodyEditor, value)
			}
			if tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.History() != history ||
				tab.Doc.Buffer.String() != markdownTablesBody {
				t.Fatal("typing before the barrier changed host state")
			}
			tc.action(t, a, tab, tester, path)
			tester.Frame()
			tester.Frame()
			if got := tab.Doc.Buffer.History(); got.UndoDepth != 1 || got.RedoDepth != 0 {
				t.Fatalf("%s barrier did not commit exactly one history transaction: %#v", tc.name, got)
			}
			if _, ok := tester.AccessByRole(ui.RoleTextField, markdownCellBodyEditor); ok {
				t.Fatalf("%s barrier left the successful cell draft open", tc.name)
			}
		})
	}
}

func TestMarkdownCellsRealAdapterPendingComposeFlushesBeforeCellActivation(t *testing.T) {
	_, tab, tester, _ := openRealMarkdownTables(t, 1180, 780)
	buffer := tab.Doc.Buffer
	beginRealMarkdownCompositionAt(t, tab, tester, "表格工作区")
	if buffer.String() != markdownTablesBody {
		t.Fatal("pending Markdown composition reached the host before cell activation")
	}

	body := requireMarkdownCellAccess(t, tester, ui.RoleCell, markdownCellBodyName)
	openMarkdownCellPointer(t, tester, body, markdownCellBodyEditor)
	flushed := strings.Replace(markdownTablesBody, "# 表格工作区", "# 中文🙂", 1)
	if buffer.String() != flushed {
		t.Fatalf("cell activation did not flush the existing Markdown composition: %q", buffer.String())
	}
	if got := buffer.History(); got.UndoDepth != 1 || got.RedoDepth != 0 {
		t.Fatalf("pending composition plus cell activation was not one host edit: %#v", got)
	}
	if !tester.HasText(markdownCellStagedNotice) {
		t.Fatalf("cell editor did not open over the flushed preview: %#v", tester.Texts())
	}
	if err := tester.Click(markdownCellDiscardLabel); err != nil {
		t.Fatalf("could not close the post-Flush cell editor: %v", err)
	}
	tester.Frame()
}

func TestMarkdownCellsRealAdapterInvalidDiscardNoopAndReadonly(t *testing.T) {
	t.Run("invalid-barriers-and-discard", func(t *testing.T) {
		a, tab, tester, _ := openRealMarkdownTables(t, 1180, 780)
		a.NewDocument("text")
		if !a.Select(0) {
			t.Fatal("could not return to the table before staging an invalid cell")
		}
		tester.Frame()
		body := requireMarkdownCellAccess(t, tester, ui.RoleCell, markdownCellBodyName)
		openMarkdownCellPointer(t, tester, body, markdownCellBodyEditor)
		replaceMarkdownCellDraft(t, tester, markdownCellBodyEditor, "非法|值")
		version, selection, history := tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection(), tab.Doc.Buffer.History()
		view := tab.View

		if a.Save(tab, false) {
			t.Fatal("Save accepted an invalid structural-pipe cell")
		}
		a.exportCurrent()
		if a.setMode(tab, "source") {
			t.Fatal("source mode accepted an invalid structural-pipe cell")
		}
		if a.Select(1) {
			t.Fatal("tab switch accepted an invalid structural-pipe cell")
		}
		a.closeDialog = func(*Tab) (closeDecision, error) {
			t.Fatal("invalid close reached the save/discard dialog")
			return closeCancel, nil
		}
		if a.CloseTab(0) {
			t.Fatal("tab close accepted an invalid structural-pipe cell")
		}
		tester.Frame()
		if tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.Selection() != selection ||
			tab.Doc.Buffer.History() != history || tab.Doc.Buffer.String() != markdownTablesBody ||
			tab.View != view || tab.Mode != "preview" || a.Active != 0 || len(a.Tabs) != 2 {
			t.Fatal("a failed cell barrier changed source, history, view, mode or tabs")
		}
		requireMarkdownCellEditor(t, tester, markdownCellBodyEditor)
		if !strings.Contains(a.Message, "单元格") || !strings.Contains(a.Message, "未写回") ||
			!tester.HasText(markdownCellDiscardLabel) ||
			!tester.HasText(markdownCellRecoverLabel) {
			t.Fatalf("invalid staged cell lost visible recovery: message=%q texts=%#v", a.Message, tester.Texts())
		}

		if err := tester.Click(markdownCellRecoverLabel); err != nil {
			t.Fatalf("could not recover the invalid cell draft: %v", err)
		}
		tester.Frame()
		tester.Frame()
		if len(a.Tabs) != 3 || a.Active != 2 {
			t.Fatalf("cell recovery tab state = tabs %d active %d", len(a.Tabs), a.Active)
		}
		recovered := a.Current()
		if recovered == nil || recovered == tab || !recovered.Doc.IsNew || recovered.Doc.Path != "" ||
			recovered.Doc.Buffer.String() != "非法|值" {
			t.Fatalf("recovered cell draft = %#v", recovered)
		}
		if tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.History() != history ||
			tab.Doc.Buffer.String() != markdownTablesBody {
			t.Fatal("cell recovery changed the original host document")
		}
		if _, ok := tester.AccessByRole(ui.RoleTextField, markdownCellBodyEditor); ok {
			t.Fatal("cell recovery left the failed editor open")
		}
	})

	t.Run("discard", func(t *testing.T) {
		_, tab, tester, _ := openRealMarkdownTables(t, 1180, 780)
		version, selection, history := tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection(), tab.Doc.Buffer.History()
		body := requireMarkdownCellAccess(t, tester, ui.RoleCell, markdownCellBodyName)
		openMarkdownCellPointer(t, tester, body, markdownCellBodyEditor)
		replaceMarkdownCellDraft(t, tester, markdownCellBodyEditor, "非法|值")
		if err := tester.Click(markdownCellDiscardLabel); err != nil {
			t.Fatalf("could not discard the invalid cell draft: %v", err)
		}
		tester.Frame()
		if tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.Selection() != selection ||
			tab.Doc.Buffer.History() != history || tab.Doc.Buffer.String() != markdownTablesBody {
			t.Fatal("explicit cell discard changed host state")
		}
		if _, ok := tester.AccessByRole(ui.RoleTextField, markdownCellBodyEditor); ok {
			t.Fatal("explicit cell discard left the editor open")
		}
		requireMarkdownTableButton(t, tester)
	})

	t.Run("no-op", func(t *testing.T) {
		_, tab, tester, _ := openRealMarkdownTables(t, 1180, 780)
		version, history := tab.Doc.Buffer.Version(), tab.Doc.Buffer.History()
		body := requireMarkdownCellAccess(t, tester, ui.RoleCell, markdownCellBodyName)
		openMarkdownCellPointer(t, tester, body, markdownCellBodyEditor)
		replaceMarkdownCellDraft(t, tester, markdownCellBodyEditor, "甲")
		if err := tester.Click(markdownCellFinishLabel); err != nil {
			t.Fatalf("could not finish the no-op cell: %v", err)
		}
		tester.Frame()
		if tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.History() != history ||
			tab.Doc.Buffer.String() != markdownTablesBody || tab.Doc.Dirty() {
			t.Fatal("no-op cell finish changed source, version, history or dirty state")
		}
		if _, ok := tester.AccessByRole(ui.RoleTextField, markdownCellBodyEditor); ok {
			t.Fatal("no-op cell finish left the editor open")
		}
	})

	t.Run("readonly", func(t *testing.T) {
		a := newTestApp(t, t.TempDir())
		doc := document.New(markdownTablesBody)
		doc.ReadOnly = true
		tab := a.add(doc, "readonly-cell.md", "")
		if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "markdown" || tab.Mode != "preview" {
			t.Fatalf("readonly cell fixture did not mount Markdown: %#v", tab)
		}
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		tester.Frame()
		version, selection, history := tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection(), tab.Doc.Buffer.History()
		dirty := tab.Doc.Dirty()
		openMarkdownCellPointerWithoutEditor(t, tester)
		if tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.Selection() != selection ||
			tab.Doc.Buffer.History() != history || tab.Doc.Buffer.String() != markdownTablesBody ||
			tab.Doc.Dirty() != dirty {
			t.Fatalf("readonly cell activation changed host state: version %d->%d selection %#v->%#v history %#v->%#v dirty=%v",
				version, tab.Doc.Buffer.Version(), selection, tab.Doc.Buffer.Selection(),
				history, tab.Doc.Buffer.History(), tab.Doc.Dirty())
		}
		for _, label := range []string{markdownCellHeaderEditor, markdownCellBodyEditor} {
			if _, ok := tester.AccessByRole(ui.RoleTextField, label); ok {
				t.Fatalf("readonly cell activation created %q", label)
			}
		}
		if !strings.Contains(a.Message, "只读") {
			t.Fatalf("readonly cell refusal is not visible: %q", a.Message)
		}
	})
}

func openMarkdownCellPointerWithoutEditor(t *testing.T, tester *ui.Tester) {
	t.Helper()
	info := requireMarkdownCellAccess(t, tester, ui.RoleCell, markdownCellBodyName)
	x, y := info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2
	tester.ClickAt(x, y)
	tester.Frame()
	tester.ClickAt(x, y)
	tester.Frame()
	tester.Frame()
}

func TestMarkdownCellsAppNarrowScreenshots(t *testing.T) {
	out := os.Getenv("LEVNOTE_MARKDOWN_CELLS_SCREENSHOT_DIR")
	if out == "" {
		t.Skip("set LEVNOTE_MARKDOWN_CELLS_SCREENSHOT_DIR for integrated cell screenshots")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(workspace, "cells.md")
	screenshotSource := append([]byte{0xef, 0xbb, 0xbf}, []byte(markdownTablesScreenshotBody)...)
	if err := os.WriteFile(path, screenshotSource, 0600); err != nil {
		t.Fatal(err)
	}

	openSplit := func(t *testing.T, dark bool) (*App, *Tab, *ui.Tester) {
		t.Helper()
		a := newTestApp(t, filepath.Join(root, "config-"+map[bool]string{true: "dark", false: "light"}[dark]))
		a.Root = workspace
		a.refreshTreeNow()
		a.Open(path)
		tab := a.Current()
		if tab == nil || !a.setMode(tab, "split") {
			t.Fatalf("could not open the cell fixture in split mode: %#v %s", tab, a.Message)
		}
		a.Dark = dark
		tester := ui.NewTester(a.View, 760, 480)
		tester.SetDark(dark)
		tester.Frame()
		tester.Frame()
		return a, tab, tester
	}
	assertAboveStatus := func(t *testing.T, tester *ui.Tester, labels ...string) {
		t.Helper()
		const statusTop = float32(480 - 29)
		for _, label := range labels {
			rect, ok := tester.Find(label)
			if !ok || rect.W <= 0 || rect.H <= 0 || rect.X < 0 || rect.Y < 0 ||
				rect.X+rect.W > 760 || rect.Y+rect.H > statusTop {
				t.Fatalf("narrow cell control %q is clipped: %#v texts=%#v", label, rect, tester.Texts())
			}
		}
	}

	t.Run("staged-light", func(t *testing.T) {
		_, tab, tester := openSplit(t, false)
		body := requireMarkdownCellAccess(t, tester, ui.RoleCell, markdownCellBodyName)
		openMarkdownCellPointer(t, tester, body, markdownCellBodyEditor)
		replaceMarkdownCellDraft(t, tester, markdownCellBodyEditor, "阶段🙂")
		tester.Frame()
		if tab.Mode != "split" || !tester.HasText("cells.md") {
			t.Fatalf("staged screenshot lost split chrome: mode=%q texts=%#v", tab.Mode, tester.Texts())
		}
		assertAboveStatus(t, tester, markdownCellStagedNotice, markdownCellFinishLabel, markdownCellDiscardLabel)
		captureAcceptanceTester(t, filepath.Join(out, "markdown-cells-staged-light-760x480.png"), tester)
	})

	t.Run("refused-dark", func(t *testing.T) {
		a, tab, tester := openSplit(t, true)
		body := requireMarkdownCellAccess(t, tester, ui.RoleCell, markdownCellBodyName)
		openMarkdownCellPointer(t, tester, body, markdownCellBodyEditor)
		replaceMarkdownCellDraft(t, tester, markdownCellBodyEditor, "坏|值")
		if err := tester.Click(markdownCellFinishLabel); err != nil {
			t.Fatalf("could not attempt the invalid cell finish: %v", err)
		}
		tester.Frame()
		tester.Frame()
		if tab.Mode != "split" || tab.Doc.Buffer.String() != markdownTablesScreenshotBody {
			t.Fatalf("invalid staged screenshot unexpectedly changed source or mode: mode=%q source=%q message=%q",
				tab.Mode, tab.Doc.Buffer.String(), a.Message)
		}
		if !tester.HasText(markdownCellRecoverLabel) {
			t.Fatalf("invalid cell recovery route is missing: %#v", tester.Texts())
		}
		assertAboveStatus(t, tester, markdownCellFinishLabel, markdownCellDiscardLabel, markdownCellRecoverLabel)
		captureAcceptanceTester(t, filepath.Join(out, "markdown-cells-refused-dark-760x480.png"), tester)
	})
}
