package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestWorkbenchActivityNavigationRetainsOperations(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.NewDocument("text")
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	for _, label := range []string{"新建", "打开", "文件夹", "保存", "最近关闭的文件", "文件树", "文档大纲", "查找", "插件", "设置"} {
		if _, ok := tester.Find(label); !ok {
			t.Fatalf("operation %q is missing", label)
		}
	}
	click := func(label string) {
		t.Helper()
		if err := tester.Click(label); err != nil {
			t.Fatal(err)
		}
		tester.Frame()
	}
	click("文档大纲")
	if !a.ShowOutline {
		t.Fatal("outline activity did not show the existing outline")
	}
	click("文件树")
	if a.ShowOutline {
		t.Fatal("files activity did not restore the tree")
	}
	click("查找")
	if !a.ShowFind {
		t.Fatal("search activity did not open Find")
	}
	if _, ok := tester.Find("搜索工作区"); !ok {
		t.Fatal("workspace search entry is missing")
	}
	click("插件")
	if !a.ShowPlugins {
		t.Fatal("plugin activity did not switch to the plugin panel")
	}
	if _, visible := tester.Find("查找内容"); visible {
		t.Fatal("Find controls remain visible over the plugin panel")
	}
	for _, label := range []string{"查看格式配置", "查看本地引擎包"} {
		if _, ok := tester.Find(label); !ok {
			t.Fatalf("plugin operation %q is missing", label)
		}
	}
	click("查找")
	if !a.ShowFind || a.ShowPlugins {
		t.Fatal("search activity did not reopen Find from plugins")
	}
	click("插件")
	click("设置")
	if !a.ShowSettings || a.ShowPlugins {
		t.Fatal("settings activity did not expose settings from plugins")
	}
	if _, ok := tester.Find("切换明暗"); !ok {
		t.Fatal("theme preference is missing")
	}
	click("插件")
	click("设置")
	if !a.ShowSettings || a.ShowPlugins {
		t.Fatal("settings activity did not reopen settings from plugins")
	}
	if _, ok := tester.Find("切换明暗"); !ok {
		t.Fatal("restored settings panel has no theme action")
	}
}

func TestWorkbenchMinimumWindowKeepsEditorAndStatusMetadata(t *testing.T) {
	for _, dark := range []bool{false, true} {
		name := "light"
		if dark {
			name = "dark"
		}
		t.Run(name, func(t *testing.T) {
			a := newTestApp(t, t.TempDir())
			a.Dark = dark
			a.NewDocument("text")
			a.Current().Doc.ReadOnly = true
			a.Message = strings.Repeat("保存失败：文件正被占用。", 12)
			a.Issues = []string{a.Message}
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			editor, ok := tester.Find("文档编辑区")
			if !ok || editor.W < 480 || editor.H < 200 || editor.X < 48 || editor.X+editor.W > 760 || editor.Y+editor.H > 451 {
				t.Fatalf("minimum window lost its editor area: %+v", editor)
			}
			for _, label := range []string{string(a.Current().Doc.Encoding), "只读", "问题 1", a.Message} {
				r, ok := tester.Find(label)
				if !ok || r.W <= 0 || r.H <= 0 || r.X < 0 || r.Y < 451 || r.X+r.W > 760 || r.Y+r.H > 480 {
					t.Fatalf("status %q is absent or clipped: %+v", label, r)
				}
			}
			for _, label := range []string{"文件树", "查找", "文档大纲", "插件", "设置"} {
				r, ok := tester.Find(label)
				if !ok || r.X != 8 || r.W != 40 || r.Y < 56 || r.Y+r.H > 451 {
					t.Fatalf("activity %q is outside the activity rail: %+v", label, r)
				}
			}
		})
	}
}

func TestWorkbenchTreeIndentAndSelectionSurviveNewChrome(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	root := t.TempDir()
	a.Root, a.treeRoot = root, root
	a.Entries = []Entry{
		{Path: filepath.Join(root, "notes"), Name: "notes", Directory: true},
		{Path: filepath.Join(root, "notes", "guide.md"), Name: "guide.md", Depth: 1},
		{Path: filepath.Join(root, "plain.txt"), Name: "plain.txt"},
	}
	a.Expanded[a.Entries[0].Path] = true
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	parent, parentOK := tester.AccessByRole(ui.RoleText, "⌄ notes")
	child, childOK := tester.AccessByRole(ui.RoleText, "guide.md")
	if !parentOK || !childOK || child.Bounds.X <= parent.Bounds.X || child.Bounds.Y <= parent.Bounds.Y {
		t.Fatalf("tree hierarchy lost indentation: parent=%+v child=%+v", parent.Bounds, child.Bounds)
	}
	if err := tester.Click("guide.md"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if a.TreeSelected != 1 {
		t.Fatalf("tree selected %d instead of the nested document", a.TreeSelected)
	}
	tester.SetSize(1180, 780)
	tester.Frame()
	tester.SetSize(760, 480)
	tester.Frame()
	if a.TreeSelected != 1 {
		t.Fatal("resizing replaced the selected tree identity")
	}
}

func TestWorkbenchWelcomeNewDocumentIsUsableAtMinimumSize(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	for _, label := range []string{"打开文件", "新建 Markdown"} {
		r, ok := tester.Find(label)
		if !ok || r.W <= 0 || r.H <= 0 || r.X < 48 || r.X+r.W > 760 || r.Y+r.H > 451 {
			t.Fatalf("welcome operation %q is clipped: %+v", label, r)
		}
	}
	if err := tester.Click("新建 Markdown"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if a.Current() == nil || len(a.Tabs) != 1 {
		t.Fatal("welcome new action did not create a document")
	}
	for _, label := range []string{"阅读 / 编辑", "源码", "分屏", "已打开文档"} {
		if _, ok := tester.Find(label); !ok {
			t.Fatalf("document operation %q is missing", label)
		}
	}
}

func TestWorkbenchCompactToolbarAndCreationList(t *testing.T) {
	for _, width := range []int{760, 1180} {
		t.Run(fmt.Sprintf("width%d", width), func(t *testing.T) {
			a := newTestApp(t, t.TempDir())
			tester := ui.NewTester(a.View, width, 780)
			tester.Frame()
			for _, label := range []string{"轻笺", "新建", "打开", "最近关闭的文件", "文件夹", "保存", "搜索已打开的文档", "打开命令面板"} {
				r, found := tester.Find(label)
				if !found || r.W <= 0 || r.H <= 0 || r.X < 0 || r.X+r.W > float32(width) || r.Y < 0 || r.Y+r.H > workbenchToolbarHeight {
					t.Fatalf("toolbar action %q is clipped: %+v", label, r)
				}
			}
			for _, item := range []struct{ label, suffix string }{{"Markdown", ".md"}, {"纯文本", ".txt"}, {"思维导图", ".mindmap"}, {"图纸", ".drawio"}} {
				r, found := tester.Find(item.label)
				if !found || r.W <= 0 || r.H != 30 || r.Y+r.H > 780-workbenchStatusHeight {
					t.Fatalf("creation entry %q is clipped: %+v", item.label, r)
				}
				if err := tester.Click(item.label); err != nil {
					t.Fatal(err)
				}
				tester.Frame()
				if a.Current() == nil || !strings.HasSuffix(a.Current().Name, item.suffix) {
					t.Fatalf("creation entry %q produced %+v", item.label, a.Current())
				}
			}
		})
	}
}
