package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// These scenes use the real workbench and file-loading path. Set the output
// directory to retain the same frames used by the layout regression check.
func TestWorkbenchVisualScenarios(t *testing.T) {
	out := os.Getenv("LEVNOTE_WORKBENCH_SCREENSHOT_DIR")
	if out != "" {
		if err := os.MkdirAll(out, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name          string
		dark          bool
		width, height int
		mode          string
	}{
		{"dark-source", true, 1180, 780, "source"},
		{"light-source", false, 1180, 780, "source"},
		{"dark-reading", true, 1180, 780, "preview"},
		{"light-reading", false, 1180, 780, "preview"},
		{"dark-narrow", true, 760, 480, "source"},
		{"dark-welcome", true, 1180, 780, "welcome"},
		{"light-welcome-narrow", false, 760, 480, "welcome"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := t.TempDir()
			root := filepath.Join(base, "LevNote")
			if err := os.MkdirAll(filepath.Join(root, "notes"), 0755); err != nil {
				t.Fatal(err)
			}
			files := map[string]string{
				"README.md":      "# 让想法，有处安放。\n\n轻笺是你的本地工作台。阅读、记录和整理，都在这里。\n\n## 今天的计划\n\n- [x] 整理项目资料\n- [x] 写下新的想法\n- [ ] 把笔记变成下一步行动\n\n## 一个专注的工作空间\n\n文件留在本地，内容由你掌握。\n用标签页切换文档，用大纲找到重点。\n\n> 从一句话开始，慢慢写成一篇。\n\n## 常用快捷键\n\n- `Ctrl + N` 新建文档\n- `Ctrl + S` 保存文件\n- `Ctrl + P` 切换已打开文档\n- `Ctrl + F` 查找与替换\n",
				"project.json":   "{\n  \"name\": \"LevNote\",\n  \"workspace\": \"local\",\n  \"formats\": [\"markdown\", \"text\", \"mindmap\", \"drawio\"]\n}\n",
				"notes/ideas.md": "# 灵感收集\n\n把零散想法写下来，再逐步整理。\n",
			}
			for name, body := range files {
				if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			a := newTestApp(t, filepath.Join(base, "config"))
			a.Dark = tc.dark
			if tc.mode != "welcome" {
				a.Root = root
				a.refreshTreeNow()
				a.Open(filepath.Join(root, "project.json"))
				a.Open(filepath.Join(root, "notes/ideas.md"))
				a.Open(filepath.Join(root, "README.md"))
				if a.Current() == nil || len(a.Tabs) != 3 {
					t.Fatalf("open scene: %s", a.Message)
				}
				a.Current().Mode = tc.mode
			}
			tester := ui.NewTester(a.View, tc.width, tc.height)
			tester.Frame()
			if tc.mode == "source" {
				r, ok := tester.Find("文档编辑区")
				if !ok || r.W < 280 || r.H < 180 || r.X < 0 || r.Y < 0 || r.X+r.W > float32(tc.width) || r.Y+r.H > float32(tc.height) {
					t.Fatalf("unusable source viewport: %#v", r)
				}
			}
			if out != "" {
				captureAcceptanceTester(t, filepath.Join(out, tc.name+".png"), tester)
			}
			if tc.mode == "source" && tc.width >= 900 {
				r, ok := tester.Find("转到源码行；当前第 1 行，第 1 列")
				if !ok {
					t.Fatal("source position action is missing")
				}
				x, y := r.X+r.W/2, r.Y+r.H/2
				for _, state := range []string{"hover", "pressed"} {
					if state == "hover" {
						tester.Move(x, y)
					} else {
						tester.Press(x, y)
					}
					tester.Frame()
					pixel := tester.Image().RGBAAt(int(r.X+3), int(y))
					background := ui.Color{R: pixel.R, G: pixel.G, B: pixel.B, A: pixel.A}
					if ratio := workbenchContrast(workbenchAppearance(tc.dark).StatusText, background); ratio < 4.5 {
						t.Fatalf("%s status label contrast = %.2f on rendered background %+v", state, ratio, pixel)
					}
					if out != "" {
						captureAcceptanceTester(t, filepath.Join(out, tc.name+"-status-"+state+".png"), tester)
					}
				}
				tester.Release(0, 0)
			}
			for _, tab := range a.Tabs {
				if tab.Doc.Dirty() {
					t.Fatalf("render changed %s", tab.Name)
				}
			}
			for name, body := range files {
				got, err := os.ReadFile(filepath.Join(root, name))
				if err != nil || !bytes.Equal(got, []byte(body)) {
					t.Fatalf("render changed file %s: %v", name, err)
				}
			}
		})
	}
}
