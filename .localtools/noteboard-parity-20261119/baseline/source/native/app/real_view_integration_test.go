package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func beginRealMarkdownComposition(t *testing.T, tab *Tab, tester *ui.Tester) {
	t.Helper()
	beginRealMarkdownCompositionAt(t, tab, tester, "标题", "原生写作")
}

func beginRealMarkdownCompositionAt(t *testing.T, tab *Tab, tester *ui.Tester, headings ...string) {
	t.Helper()
	var rect ui.Rect
	ok := false
	for _, heading := range headings {
		if rect, ok = tester.Find(heading); ok {
			break
		}
	}
	if !ok {
		t.Fatalf("real Markdown heading did not render: %#v", tester.Texts())
	}
	tester.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	tester.Frame()
	tester.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	tester.Frame()
	tester.Command("selectAll")
	tester.Compose("中文🙂", 3)
	if strings.Contains(tab.Doc.Buffer.String(), "中文🙂") {
		t.Fatal("Markdown preedit reached the host buffer before a lifecycle action")
	}
}

func beginRealGraphComposition(t *testing.T, tab *Tab, tester *ui.Tester) {
	t.Helper()
	if err := tester.Click("节点文本"); err != nil {
		t.Fatalf("real graph node editor did not render: %v", err)
	}
	tester.Command("selectAll")
	tester.Compose("中文🙂", 3)
	if strings.Contains(tab.Doc.Buffer.String(), "中文🙂") {
		t.Fatal("graph preedit reached the host buffer before a lifecycle action")
	}
}

func TestRealMarkdownPendingIMECompositionAcrossSaveModeAndTab(t *testing.T) {
	for _, action := range []string{"save", "mode", "tab"} {
		t.Run(action, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "note.md")
			if err := os.WriteFile(path, []byte("# 标题\n\n第二块\n"), 0600); err != nil {
				t.Fatal(err)
			}
			a := newTestApp(t, filepath.Join(root, "config"))
			a.Open(path)
			tab := a.Current()
			if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "markdown" || tab.Mode != "preview" {
				t.Fatalf("real Markdown adapter did not open: %#v", tab)
			}
			tester := ui.NewTester(a.View, 1180, 780)
			tester.Frame()
			beginRealMarkdownComposition(t, tab, tester)

			switch action {
			case "save":
				if !a.Save(tab, false) {
					t.Fatalf("Markdown save failed: %s", a.Message)
				}
				if tab.Mode != "preview" {
					t.Fatalf("ordinary Markdown save changed mode to %q", tab.Mode)
				}
				saved, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(saved), "# 中文🙂") {
					t.Fatalf("saved Markdown omitted IME composition: %q", saved)
				}
			case "mode":
				if !a.setMode(tab, "split") {
					t.Fatalf("Markdown mode barrier failed: %s", a.Message)
				}
				if tab.Mode != "split" {
					t.Fatalf("Markdown mode = %q, want split", tab.Mode)
				}
			case "tab":
				a.NewDocument("text")
				if len(a.Tabs) != 2 || a.Current() == tab {
					t.Fatalf("Markdown tab barrier did not switch tabs: %d", len(a.Tabs))
				}
			}
			if !strings.Contains(tab.Doc.Buffer.String(), "# 中文🙂") {
				t.Fatalf("Markdown lifecycle action lost composition: %q", tab.Doc.Buffer.String())
			}
		})
	}
}

func TestNarrowMarkdownConflictKeepsViewOutOfStatusRow(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.md")
	if err := os.WriteFile(path, []byte("# 标题\n\n第二块\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || !a.setMode(tab, "split") {
		t.Fatalf("could not enter Markdown split mode: %s", a.Message)
	}
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	beginRealMarkdownComposition(t, tab, tester)
	if err := tab.Doc.Buffer.ReplaceUTF16(2, 4, "外部"); err != nil {
		t.Fatal(err)
	}
	if a.flushTab(tab, "验收冲突") {
		t.Fatal("overlapping Markdown draft unexpectedly flushed")
	}
	tester.Frame()

	const statusTop = float32(480 - 29)
	for _, label := range []string{"完成编辑", "打开未同步副本", "放弃草稿", "正在编辑此块"} {
		rect, ok := tester.Find(label)
		if !ok {
			t.Fatalf("conflict control %q is missing: %#v", label, tester.Texts())
		}
		if rect.Y < 0 || rect.Y+rect.H > statusTop {
			t.Fatalf("conflict control %q overlaps the status row: %#v, status-top=%v", label, rect, statusTop)
		}
	}
	message, ok := tester.Find(a.Message)
	if !ok {
		t.Fatalf("compact status message is missing: %q", a.Message)
	}
	issues, ok := tester.Find(fmt.Sprintf("问题 %d", len(a.Issues)))
	if !ok || len(a.Issues) == 0 {
		t.Fatalf("compact status lost the Issues details entry: %#v", tester.Texts())
	}
	if message.Y < statusTop || issues.Y < statusTop ||
		message.X+message.W > issues.X || message.Y+message.H > 480 ||
		issues.Y+issues.H > 480 {
		t.Fatalf("compact status items overlap or leave the status row: message=%#v issues=%#v", message, issues)
	}
}

func TestRealGraphPendingIMECompositionAcrossSaveModeAndTab(t *testing.T) {
	const graph = `{
  "format": "levnote.mindmap",
  "version": 1,
  "layout": "logicalStructure",
  "root": {
    "id": "root",
    "topic": "中心",
    "children": [
      {"id": "left", "topic": "左分支", "children": []}
    ]
  }
}`

	for _, action := range []string{"save", "mode", "tab"} {
		t.Run(action, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "graph.mindmap")
			if err := os.WriteFile(path, []byte(graph), 0600); err != nil {
				t.Fatal(err)
			}
			a := newTestApp(t, filepath.Join(root, "config"))
			a.Open(path)
			tab := a.Current()
			if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "mindmap" || tab.Mode != "preview" {
				t.Fatalf("real graph adapter did not open: %#v", tab)
			}
			tester := ui.NewTester(a.View, 1180, 780)
			tester.Frame()
			beginRealGraphComposition(t, tab, tester)

			switch action {
			case "save":
				if !a.Save(tab, false) {
					t.Fatalf("graph save failed: %s", a.Message)
				}
				if tab.Mode != "preview" {
					t.Fatalf("ordinary graph save changed mode to %q", tab.Mode)
				}
				saved, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(saved), "中文🙂") {
					t.Fatalf("saved graph omitted IME composition: %q", saved)
				}
			case "mode":
				if !a.setMode(tab, "source") {
					t.Fatalf("graph mode barrier failed: %s", a.Message)
				}
				if tab.Mode != "source" {
					t.Fatalf("graph mode = %q, want source", tab.Mode)
				}
			case "tab":
				a.NewDocument("text")
				if len(a.Tabs) != 2 || a.Current() == tab {
					t.Fatalf("graph tab barrier did not switch tabs: %d", len(a.Tabs))
				}
			}
			if !strings.Contains(tab.Doc.Buffer.String(), "中文🙂") {
				t.Fatalf("graph lifecycle action lost composition: %q", tab.Doc.Buffer.String())
			}
		})
	}
}

func TestRealDrawioPendingIMECompositionAcrossSaveModeAndTab(t *testing.T) {
	const diagram = `<mxfile host="levnote"><diagram id="page-1" name="页面 1"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="shape" value="标题" vertex="1" parent="1" vendor="keep"><mxGeometry x="40" y="50" width="160" height="60" as="geometry"/></mxCell></root></mxGraphModel></diagram></mxfile>`

	for _, action := range []string{"save", "mode", "tab"} {
		t.Run(action, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "diagram.drawio")
			if err := os.WriteFile(path, []byte(diagram), 0600); err != nil {
				t.Fatal(err)
			}
			a := newTestApp(t, filepath.Join(root, "config"))
			a.Open(path)
			tab := a.Current()
			if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "drawio" || tab.Mode != "preview" {
				t.Fatalf("real draw.io adapter did not open: %#v", tab)
			}
			tester := ui.NewTester(a.View, 1180, 780)
			tester.Frame()
			beginRealGraphComposition(t, tab, tester)

			switch action {
			case "save":
				if !a.Save(tab, false) {
					t.Fatalf("draw.io save failed: %s", a.Message)
				}
				if tab.Mode != "preview" {
					t.Fatalf("ordinary draw.io save changed mode to %q", tab.Mode)
				}
				saved, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(saved), `value="中文🙂"`) ||
					!strings.Contains(string(saved), `vendor="keep"`) {
					t.Fatalf("draw.io save lost composition or unknown metadata: %q", saved)
				}
			case "mode":
				if !a.setMode(tab, "source") {
					t.Fatalf("draw.io mode barrier failed: %s", a.Message)
				}
				if tab.Mode != "source" {
					t.Fatalf("draw.io mode = %q, want source", tab.Mode)
				}
			case "tab":
				a.NewDocument("text")
				if len(a.Tabs) != 2 || a.Current() == tab {
					t.Fatalf("draw.io tab barrier did not switch tabs: %d", len(a.Tabs))
				}
			}
			if !strings.Contains(tab.Doc.Buffer.String(), `value="中文🙂"`) ||
				!strings.Contains(tab.Doc.Buffer.String(), `vendor="keep"`) {
				t.Fatalf("draw.io lifecycle action lost composition or metadata: %q",
					tab.Doc.Buffer.String())
			}
		})
	}
}
