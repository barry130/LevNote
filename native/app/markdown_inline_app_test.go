package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

const markdownInlineBody = "# 标题🙂\r\n\r\n原文与未知标记保留。\r\n\r\n<!-- unknown-key: 原值 -->\r\n"

func openMarkdownInlineApp(t *testing.T, width, height int, dark bool) (*App, *Tab, *ui.Tester, string) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "inline.md")
	if err := os.WriteFile(path, append([]byte{0xef, 0xbb, 0xbf}, []byte(markdownInlineBody)...), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Dark = dark
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "markdown" || tab.Mode != "preview" {
		t.Fatal("Markdown real adapter missing")
	}
	tester := ui.NewTester(a.View, width, height)
	tester.SetDark(dark)
	tester.Frame()
	r, ok := tester.Find("标题🙂")
	if !ok {
		t.Fatal("heading not visible", tester.Texts())
	}
	tester.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tester.Frame()
	tester.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tester.Frame()
	if !tester.Focused("文档编辑区") {
		t.Fatal("native double click did not focus block draft", tester.Texts())
	}
	tester.Command("selectAll")
	return a, tab, tester, path
}

func TestMarkdownInlineRealAdapterBOMCRLFSaveReopen(t *testing.T) {
	for _, tc := range []struct {
		label, marker, html string
		key                 ui.Key
	}{
		{"粗体", "**", "<strong>标题🙂</strong>", ui.KeyB},
		{"斜体", "*", "<em>标题🙂</em>", ui.KeyI},
		{"行内代码", "`", "<code>标题🙂</code>", ui.KeyE},
	} {
		t.Run(tc.label, func(t *testing.T) {
			a, tab, tester, path := openMarkdownInlineApp(t, 1180, 780, false)
			tester.Advance(time.Second)
			if err := tester.Click(tc.label); err != nil {
				t.Fatal(err)
			}
			tester.Frame()
			want := strings.Replace(markdownInlineBody, "标题🙂", tc.marker+"标题🙂"+tc.marker, 1)
			if tab.Doc.Buffer.String() != want || !tab.Doc.Dirty() || tab.Doc.Buffer.History().UndoDepth != 1 {
				t.Fatal("format did not reach document", tab.Doc.Buffer.String(), a.Message)
			}
			before := append([]byte{0xef, 0xbb, 0xbf}, []byte(markdownInlineBody)...)
			if !bytes.Equal(mustReadFile(t, path), before) {
				t.Fatal("format wrote disk before Save")
			}
			tester.Key(ui.Cmd, ui.KeyZ)
			if tab.Doc.Buffer.String() != markdownInlineBody {
				t.Fatal("draft undo did not restore bytes")
			}
			tester.Key(ui.Cmd, ui.KeyY)
			if tab.Doc.Buffer.String() != want {
				t.Fatal("draft redo failed")
			}
			tester.Key(ui.Cmd, tc.key)
			if tab.Doc.Buffer.String() != markdownInlineBody {
				t.Fatal("second format did not toggle")
			}
			tester.Key(ui.Cmd, tc.key)
			if err := tester.Click("完成编辑"); err != nil {
				t.Fatal(err)
			}
			tester.Frame()
			if !a.Save(tab, false) {
				t.Fatal("save failed", a.Message)
			}
			if !bytes.Equal(mustReadFile(t, path), append([]byte{0xef, 0xbb, 0xbf}, []byte(want)...)) || !tab.Doc.BOM || tab.Doc.Dirty() {
				t.Fatal("Save changed BOM/EOL/unknown bytes")
			}
			reopened := newTestApp(t, filepath.Join(t.TempDir(), "config"))
			reopened.Open(path)
			reopenedTester := ui.NewTester(reopened.View, 1180, 780)
			reopenedTester.Frame()
			other := reopened.Current()
			if other == nil || other.Doc.Buffer.String() != want || !other.Doc.BOM {
				t.Fatal("reopen lost formatting or BOM")
			}
			output, err := other.View.Export("html")
			if err != nil || !strings.Contains(string(output), tc.html) {
				t.Fatal("reopened format not rendered by existing exporter", string(output), err)
			}
		})
	}
}

func TestMarkdownInlineRealAdapterIMEBarriers(t *testing.T) {
	for _, action := range []string{"save", "mode", "tab"} {
		t.Run(action, func(t *testing.T) {
			a, tab, tester, path := openMarkdownInlineApp(t, 1180, 780, true)
			tester.Key(ui.Cmd, ui.KeyB)
			formatted := strings.Replace(markdownInlineBody, "标题🙂", "**标题🙂**", 1)
			if tab.Doc.Buffer.String() != formatted {
				t.Fatal("keyboard format missing", a.Message)
			}
			tester.Compose("输入🙂", 4)
			tester.Key(ui.Cmd, ui.KeyI)
			if tab.Doc.Buffer.String() != formatted {
				t.Fatal("format shortcut committed native preedit")
			}
			switch action {
			case "save":
				if !a.Save(tab, false) {
					t.Fatal(a.Message)
				}
			case "mode":
				if !a.setMode(tab, "source") {
					t.Fatal(a.Message)
				}
			case "tab":
				a.NewDocument("text")
			}
			want := strings.Replace(markdownInlineBody, "标题🙂", "**输入🙂**", 1)
			if tab.Doc.Buffer.String() != want || tab.Doc.Buffer.InGroup() || tab.Doc.Buffer.History().UndoDepth != 2 {
				t.Fatal("lifecycle lost composition or merged format history", tab.Doc.Buffer.String(), tab.Doc.Buffer.History())
			}
			if action == "save" && !bytes.Equal(mustReadFile(t, path), append([]byte{0xef, 0xbb, 0xbf}, []byte(want)...)) {
				t.Fatal("IME save changed bytes")
			}
		})
	}
}

func TestMarkdownInlineLayoutAndCapture(t *testing.T) {
	for _, dark := range []bool{true, false} {
		for _, width := range []int{760, 1180} {
			t.Run(fmt.Sprintf("dark=%v-width=%d", dark, width), func(t *testing.T) {
				_, tab, tester, _ := openMarkdownInlineApp(t, width, 480, dark)
				tester.Key(ui.Cmd, ui.KeyB)
				if !strings.Contains(tab.Doc.Buffer.String(), "**标题🙂**") {
					t.Fatal("layout fixture did not format")
				}
				for _, label := range []string{"粗体", "斜体", "行内代码", "文档编辑区", "完成编辑"} {
					r, ok := tester.Find(label)
					if !ok || r.W <= 0 || r.H <= 0 || r.X < 0 || r.Y < 0 || r.X+r.W > float32(width) || r.Y+r.H > 451 {
						t.Fatalf("clipped format control %s: %+v", label, r)
					}
				}
				if out := os.Getenv("LEVNOTE_MARKDOWN_INLINE_SCREENSHOT_DIR"); out != "" {
					if err := os.MkdirAll(out, 0755); err != nil {
						t.Fatal(err)
					}
					captureAcceptanceTester(t, filepath.Join(out, fmt.Sprintf("markdown-inline-%v-%d.png", dark, width)), tester)
				}
			})
		}
	}
}
