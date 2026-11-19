package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

func paletteOpen(t *testing.T, a *App, tester *ui.Tester) {
	t.Helper()
	tester.Key(ui.Cmd|ui.Shift, ui.KeyP)
	if !a.commandPalette.open || !tester.Focused(commandPaletteQueryLabel) {
		t.Fatalf("palette not open/focused: %s; %v", a.Message, tester.Texts())
	}
}
func paletteRun(t *testing.T, a *App, tester *ui.Tester, query string) {
	t.Helper()
	paletteOpen(t, a, tester)
	tester.Type(query)
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
}

func TestCommandPaletteKeyboardFilterExecuteAndEscape(t *testing.T) {
	a, first, _, tester := openDocumentsFixture(t)
	before, history, selection := first.Doc.Buffer.Snapshot(), first.Doc.Buffer.History(), first.Doc.Buffer.Selection()
	paletteOpen(t, a, tester)
	tester.Type("unmatched command")
	if len(a.commandPalette.matches) != 0 {
		t.Fatal("unexpected filter matches")
	}
	tester.Key(0, ui.KeyEnter)
	if !a.commandPalette.open {
		t.Fatal("empty query executed a command")
	}
	tester.Key(0, ui.KeyEscape)
	tester.Frame()
	if a.commandPalette.open || !tester.Focused("文档编辑区") {
		t.Fatal("Escape did not restore source focus")
	}
	dark := a.Dark
	paletteRun(t, a, tester, "> THEME dark")
	if a.commandPalette.open || a.Dark == dark {
		t.Fatal("multi-token English command did not execute")
	}
	if !first.Doc.Buffer.EqualSnapshot(before) || first.Doc.Buffer.History() != history || first.Doc.Buffer.Selection() != selection {
		t.Fatal("palette changed document content, history or selection")
	}
	if a.commandPalette.origin != nil || a.commandPalette.matches != nil || a.commandPalette.list.Key != nil || a.commandPalette.query != "" {
		t.Fatal("closed palette retained command rows or the document")
	}
	paletteRun(t, a, tester, "新建文本")
	if a.Current() == first || a.Current().Name != "未命名.txt" || len(a.Tabs) != 3 {
		t.Fatal("new-document command did not use the application action")
	}
}

func TestCommandPaletteFilterSelectionAndReset(t *testing.T) {
	a, _, _, tester := openDocumentsFixture(t)
	paletteOpen(t, a, tester)
	tester.Type("视图")
	if a.commandPalette.selected != 0 || a.commandPalette.selectedID != "view.source" {
		t.Fatalf("first filtered row lost: index=%d id=%s", a.commandPalette.selected, a.commandPalette.selectedID)
	}
	tester.Key(0, ui.KeyDown)
	if a.commandPalette.selected != 1 || a.commandPalette.selectedID != "view.preview" {
		t.Fatalf("first Down did not advance: index=%d id=%s", a.commandPalette.selected, a.commandPalette.selectedID)
	}
	tester.Key(ui.Cmd, ui.KeyA)
	tester.Type("theme")
	if a.commandPalette.selected != 0 || a.commandPalette.selectedID != "view.theme" {
		t.Fatal("single filtered row not highlighted")
	}
	tester.Key(ui.Cmd, ui.KeyA)
	tester.Key(0, ui.KeyBackspace)
	tester.Frame()
	if a.commandPalette.selected != 0 || a.commandPalette.selectedID != "file.new.markdown" {
		t.Fatalf("query clear retained old key: index=%d id=%s", a.commandPalette.selected, a.commandPalette.selectedID)
	}
}

func TestCommandPaletteArrowPointerAndLiveAvailability(t *testing.T) {
	a, first, _, tester := openDocumentsFixture(t)
	pickerClick(t, tester, "打开命令面板")
	if !a.commandPalette.open || !tester.Focused(commandPaletteQueryLabel) {
		t.Fatal("toolbar command did not focus query")
	}
	tester.Type("新建")
	tester.Key(0, ui.KeyDown)
	if a.commandPalette.selectedID != "file.new.text" {
		t.Fatalf("selection=%s", a.commandPalette.selectedID)
	}
	tester.Key(0, ui.KeyUp)
	if a.commandPalette.selectedID != "file.new.markdown" {
		t.Fatal("arrow navigation lost stable identity")
	}
	tester.Key(0, ui.KeyEscape)
	tester.Frame()
	paletteOpen(t, a, tester)
	tester.Type("主题")
	dark := a.Dark
	pickerClick(t, tester, "执行命令：外观：切换明暗主题")
	tester.Frame()
	if a.commandPalette.open || a.Dark == dark {
		t.Fatal("pointer command did not execute")
	}

	first.Doc.ReadOnly = true
	paletteOpen(t, a, tester)
	tester.Type("保存")
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	if !a.commandPalette.open || !strings.Contains(a.commandPalette.message, "当前不可用") {
		t.Fatal("readonly Save was executed")
	}
	first.Doc.ReadOnly = false
	saved := false
	a.saveDialog = func(*Tab) (string, error) { saved = true; return "", nil }
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	if a.commandPalette.open || !saved {
		t.Fatal("availability was not rechecked at execution")
	}
}

func TestCommandPaletteQueryAndSourceComposition(t *testing.T) {
	a, first, _, tester := openDocumentsFixture(t)
	before, history := first.Doc.Buffer.Snapshot(), first.Doc.Buffer.History()
	paletteOpen(t, a, tester)
	tester.Compose("主题", 2)
	for _, key := range []ui.Key{ui.KeyLeft, ui.KeyRight, ui.KeyBackspace, ui.KeyDelete, ui.KeyUp, ui.KeyDown, ui.KeyEnter, ui.KeyEscape} {
		tester.Key(0, key)
		if !a.commandPalette.open || !a.commandPalette.composing || a.commandPalette.query != "" || a.commandPalette.pendingID != "" || !tester.Focused(commandPaletteQueryLabel) {
			t.Fatalf("key %v stole query composition", key)
		}
	}
	pickerClick(t, tester, "关闭命令面板")
	if !a.commandPalette.open || !a.commandPalette.composing {
		t.Fatal("close button discarded preedit")
	}
	tester.Compose("", 0)
	tester.Key(0, ui.KeyEscape)
	tester.Frame()
	if a.commandPalette.open || !tester.Focused("文档编辑区") {
		t.Fatal("composition cancellation failed")
	}
	if !first.Doc.Buffer.EqualSnapshot(before) || first.Doc.Buffer.History() != history {
		t.Fatal("query leaked into source")
	}
	tester.Compose("未确认", 3)
	marked, active := first.Editor.MarkedRange()
	current, h := first.Doc.Buffer.Snapshot(), first.Doc.Buffer.History()
	pickerClick(t, tester, "打开命令面板")
	got, composing := first.Editor.MarkedRange()
	if a.commandPalette.open || !active || !composing || marked != got || !tester.Focused("文档编辑区") || !first.Doc.Buffer.EqualSnapshot(current) || first.Doc.Buffer.History() != h {
		t.Fatal("toolbar stole source composition")
	}
	tester.Key(0, ui.KeyF1)
	if a.commandPalette.open {
		t.Fatal("F1 stole source composition")
	}
}

func TestCommandPaletteStaleActionAndModalLifecycle(t *testing.T) {
	a, _, second, tester := openDocumentsFixture(t)
	paletteOpen(t, a, tester)
	a.commandPalette.pendingID = "view.theme"
	dark := a.Dark
	a.Active = 1
	tester.Frame()
	if a.commandPalette.open || a.Dark != dark || a.Current() != second {
		t.Fatal("stale command executed in a different tab")
	}
	paletteOpen(t, a, tester)
	a.commandPalette.pendingID = "file.new.text"
	a.modalDepth++
	tester.Frame()
	if a.commandPalette.open || len(a.Tabs) != 2 {
		t.Fatal("command executed behind another dialog")
	}
	a.modalDepth--
	tester.Key(0, ui.KeyF1)
	if !a.commandPalette.open {
		t.Fatal("F1 did not open palette")
	}
	tester.Key(0, ui.KeyTab)
	tester.Key(0, ui.KeyF1)
	if !tester.Focused(commandPaletteQueryLabel) {
		t.Fatal("F1 did not refocus existing palette")
	}
}

func TestCommandPaletteWelcomeAndDocumentNavigation(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tester := ui.NewTester(a.View, 760, 480)
	paletteRun(t, a, tester, "新建 Markdown")
	if a.Current() == nil || a.Current().Adapter == nil || a.Current().Adapter.ID != "markdown" {
		t.Fatal("welcome palette failed to create Markdown")
	}
	first := a.Current()
	paletteRun(t, a, tester, "视图 分屏")
	if first.Mode != "split" {
		t.Fatal("mode command did not switch the document")
	}
	paletteRun(t, a, tester, "查找")
	if !a.ShowFind || !tester.Focused("查找内容") {
		t.Fatalf("find command failed to focus: %v", tester.Texts())
	}
	tester.Key(0, ui.KeyEscape)
	paletteRun(t, a, tester, "搜索已打开")
	if !a.openDocuments.open || !tester.Focused(openDocumentsQueryLabel) {
		t.Fatal("document picker did not receive focus after command palette")
	}
}

func TestCommandPaletteRichDraftGrowthAndReentrantOpening(t *testing.T) {
	t.Run("growth", func(t *testing.T) {
		const source = "# 标题\n\n正文\n"
		a := newTestApp(t, t.TempDir())
		doc := document.New(source)
		doc.Buffer = text.NewWithLimit(source, len(source))
		origin := a.add(doc, "limited.md", "")
		tester := ui.NewTester(a.View, 760, 480)
		beginRealMarkdownComposition(t, origin, tester)
		tester.Key(ui.Cmd|ui.Shift, ui.KeyP)
		if a.commandPalette.open || !tester.Focused("文档编辑区") {
			t.Fatal("rich IME ownership lost")
		}
		tester.Type("中文🙂")
		if err := editor.FlushView(origin.View); !errors.Is(err, text.ErrSizeLimit) {
			t.Fatalf("fixture missing growth refusal: %v", err)
		}
		history, selection, view := doc.Buffer.History(), doc.Buffer.Selection(), origin.View
		pickerClick(t, tester, "打开命令面板")
		if a.commandPalette.open || origin.View != view || doc.Buffer.String() != source || doc.Buffer.History() != history || doc.Buffer.Selection() != selection || !tester.Focused("文档编辑区") {
			t.Fatal("palette discarded refused rich draft")
		}
		tester.Command("selectAll")
		tester.Type("短")
		paletteOpen(t, a, tester)
		tester.Key(0, ui.KeyEscape)
		tester.Frame()
		tester.Type("!")
		if doc.Buffer.String() != "# 短!\n\n正文\n" {
			t.Fatalf("rich opener not restored: %q", doc.Buffer.String())
		}
	})
	t.Run("reentry", func(t *testing.T) {
		a, first, second, tester := openDocumentsFixture(t)
		first.View = &navigationFlushView{flush: func() error { a.Active = 1; return nil }}
		tester.Key(ui.Cmd|ui.Shift, ui.KeyP)
		if a.commandPalette.open || a.Current() != second {
			t.Fatal("palette retained old origin after Flush reentry")
		}
	})
}

func TestCommandPaletteLayoutAndCapture(t *testing.T) {
	for _, dark := range []bool{true, false} {
		for _, width := range []int{760, 1180} {
			t.Run(fmt.Sprintf("dark=%v-width=%d", dark, width), func(t *testing.T) {
				a := newTestApp(t, t.TempDir())
				a.Dark = dark
				a.add(document.New("# 项目笔记\n\n用命令面板快速找到需要的操作。\n"), "项目笔记.md", "")
				tester := ui.NewTester(a.View, width, 480)
				paletteOpen(t, a, tester)
				for _, label := range []string{"命令面板", commandPaletteQueryLabel, "关闭命令面板", "执行命令：文件：新建 Markdown"} {
					r, ok := tester.Find(label)
					if !ok || r.W <= 0 || r.H <= 0 || r.X < 0 || r.Y < 0 || r.X+r.W > float32(width) || r.Y+r.H > 480 {
						t.Fatalf("clipped %s: %+v", label, r)
					}
				}
				if out := os.Getenv("LEVNOTE_PALETTE_SCREENSHOT_DIR"); out != "" {
					if err := os.MkdirAll(out, 0755); err != nil {
						t.Fatal(err)
					}
					captureAcceptanceTester(t, filepath.Join(out, fmt.Sprintf("command-palette-%v-%d.png", dark, width)), tester)
				}
			})
		}
	}
}
