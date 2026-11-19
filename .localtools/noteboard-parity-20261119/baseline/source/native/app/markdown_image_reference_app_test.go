package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func requireMarkdownImageReference(t *testing.T, tester *ui.Tester, alt, target string) {
	t.Helper()
	info, ok := tester.AccessByRole(ui.RoleLink, "图片引用："+alt)
	if !ok || !info.Focusable || info.Disabled || info.Offscreen || info.Bounds.W <= 0 ||
		info.Description != "图片引用（不内嵌显示）；Ctrl/Cmd+点击或 Tab 后 Enter 打开："+target {
		t.Fatalf("registered image reference unavailable: %+v texts=%q", info, tester.Texts())
	}
}

func settleMarkdownImageReader(t *testing.T, a *App, tester *ui.Tester) {
	t.Helper()
	tester.Frame()
	if a.binaryPreview != nil {
		waitBinaryTask(t, a.binaryPreview.task)
	}
	tester.Frame()
	tester.Frame()
}

func TestMarkdownImageReferenceRealAdapterOpenPreservesSource(t *testing.T) {
	for _, format := range []string{"PNG", "JPEG"} {
		t.Run(format, func(t *testing.T) {
			root := t.TempDir()
			suffix := ".png"
			if format == "JPEG" {
				suffix = ".jpg"
			}
			// %20 is deliberately a literal filename component, not a space.
			name := "原%20图" + suffix
			targetPath := filepath.Join(root, name)
			targetBytes := imageFixture(t, targetPath, format)
			body := "# 原始文档\r\n\r\n![本地图](" + name + ")\r\n"
			before := append([]byte{0xef, 0xbb, 0xbf}, []byte(body)...)
			sourcePath := filepath.Join(root, "images.md")
			if err := os.WriteFile(sourcePath, before, 0600); err != nil {
				t.Fatal(err)
			}
			a := newTestApp(t, filepath.Join(root, "config"))
			a.Open(sourcePath)
			source := a.Current()
			if source == nil || source.Adapter == nil || source.Adapter.ID != "markdown" || source.Mode != "preview" {
				t.Fatal("registered Markdown fixture did not open")
			}
			tester := ui.NewTester(a.View, 1180, 780)
			requireMarkdownImageReference(t, tester, "本地图", name)
			version, history, selection := source.Doc.Buffer.Version(), source.Doc.Buffer.History(), source.Doc.Buffer.Selection()
			if format == "PNG" {
				clickMarkdownLink(t, tester, "图片引用：本地图")
			} else {
				for attempt := 0; attempt < 80; attempt++ {
					info, _ := tester.AccessByRole(ui.RoleLink, "图片引用：本地图")
					if info.Focused {
						break
					}
					tester.Key(0, ui.KeyTab)
				}
				if info, _ := tester.AccessByRole(ui.RoleLink, "图片引用：本地图"); !info.Focused {
					t.Fatal("registered image keyboard route unreachable")
				}
				tester.Key(0, ui.KeyEnter)
			}
			settleMarkdownImageReader(t, a, tester)
			imageTab := a.Current()
			if len(a.Tabs) != 2 || imageTab == source || imageTab.path() != targetPath ||
				imageTab.Adapter == nil || imageTab.Adapter.ID != "image" || imageTab.Doc != nil ||
				imageTab.View == nil || imageTab.ImageFormat != format || imageTab.ImageWidth != 40 || imageTab.ImageHeight != 24 ||
				!tester.HasText("适合窗口") || !tester.HasText("另存副本") {
				t.Fatalf("reference did not reach actual %s reader: tabs=%d current=%+v texts=%q", format, len(a.Tabs), imageTab, tester.Texts())
			}
			if !a.Select(0) {
				t.Fatal(a.Message)
			}
			tester.Frame()
			requireMarkdownImageReference(t, tester, "本地图", name)
			// Selecting the source is programmatic; make the next pointer press
			// a fresh user gesture, not continuation of the original image click.
			tester.Advance(time.Second)
			clickMarkdownLink(t, tester, "图片引用：本地图")
			settleMarkdownImageReader(t, a, tester)
			if len(a.Tabs) != 2 || a.Current() != imageTab {
				t.Fatalf("existing image selection failed: tabs=%d active=%d current=%p source=%p image=%p currentPath=%q message=%q",
					len(a.Tabs), a.Active, a.Current(), source, imageTab, a.Current().path(), a.Message)
			}
			if source.Doc.Buffer.String() != body || source.Doc.Buffer.Version() != version ||
				source.Doc.Buffer.History() != history || source.Doc.Buffer.Selection() != selection || source.Doc.Dirty() {
				t.Fatal("image navigation changed source/history/selection")
			}
			if !a.Select(0) || !a.Save(source, false) {
				t.Fatal(a.Message)
			}
			if got, err := os.ReadFile(sourcePath); err != nil || !bytes.Equal(got, before) {
				t.Fatal("BOM/CRLF source changed", err)
			}
			if got, err := os.ReadFile(targetPath); err != nil || !bytes.Equal(got, targetBytes) {
				t.Fatal("reader changed original image", err)
			}
			reopened := newTestApp(t, filepath.Join(root, "reopened-config"))
			reopened.Open(sourcePath)
			if tab := reopened.Current(); tab == nil || tab.Doc.Buffer.String() != body || tab.Doc.Dirty() || tab.Doc.Buffer.History().UndoDepth != 0 {
				t.Fatal("unchanged Markdown did not reopen")
			}
		})
	}
}

func TestMarkdownImageReferenceRealAdapterRefusalAndDisabledFallback(t *testing.T) {
	root := t.TempDir()
	imagePath := filepath.Join(root, "local.png")
	imageFixture(t, imagePath, "PNG")
	body := "![丢失](missing.png)\n\n![网络](https://example.invalid/image.png)\n\n![共享](\\\\server\\share\\image.png)\n\n![脚本](javascript:alert)\n\n![本地](local.png)\n"
	sourcePath := filepath.Join(root, "references.md")
	if err := os.WriteFile(sourcePath, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(sourcePath)
	source := a.Current()
	tester := ui.NewTester(a.View, 1180, 780)
	version, history, selection := source.Doc.Buffer.Version(), source.Doc.Buffer.History(), source.Doc.Buffer.Selection()
	for _, item := range []struct{ alt, target, reason string }{
		{"丢失", "missing.png", ""},
		{"网络", "https://example.invalid/image.png", "不会打开网络地址"},
		{"共享", `\\server\share\image.png`, "远程 UNC"},
		{"脚本", "javascript:alert", "不会打开网络地址"},
	} {
		requireMarkdownImageReference(t, tester, item.alt, item.target)
		a.Message = ""
		issues := len(a.Issues)
		clickMarkdownLink(t, tester, "图片引用："+item.alt)
		if len(a.Tabs) != 1 || a.Current() != source || a.Message == "" || len(a.Issues) <= issues ||
			item.reason != "" && !strings.Contains(a.Message, item.reason) {
			t.Fatalf("reference refusal %s: tabs=%d message=%q", item.alt, len(a.Tabs), a.Message)
		}
	}
	if source.Doc.Buffer.String() != body || source.Doc.Buffer.Version() != version || source.Doc.Buffer.History() != history || source.Doc.Buffer.Selection() != selection || source.Doc.Dirty() {
		t.Fatal("refusal changed source")
	}
	a.setPluginEnabled("image", false)
	tester.Frame()
	clickMarkdownLink(t, tester, "图片引用：本地")
	settleMarkdownImageReader(t, a, tester)
	tab := a.Current()
	if len(a.Tabs) != 2 || tab.path() != imagePath || tab.Doc != nil || tab.Binary == nil || tab.Adapter != nil ||
		tab.Mode != "source" || !tab.Editor.ReadOnly || tab.Editor.Buffer.LenBytes() == 0 ||
		!strings.Contains(tab.FormatWarning, "未启用") || a.heldBinary(tab) == nil {
		t.Fatalf("disabled image did not retain readonly bytes: %+v", tab)
	}
	if got, err := os.ReadFile(sourcePath); err != nil || string(got) != body {
		t.Fatal("refusal/fallback wrote source", err)
	}
	unsaved := newTestApp(t, filepath.Join(root, "unsaved-config"))
	unsaved.NewDocument("markdown")
	untitled := unsaved.Current()
	if err := untitled.Doc.Buffer.ReplaceUTF16(0, untitled.Doc.Buffer.LenUTF16(), "![未保存](local.png)"); err != nil {
		t.Fatal(err)
	}
	unsavedTester := ui.NewTester(unsaved.View, 1180, 780)
	requireMarkdownImageReference(t, unsavedTester, "未保存", "local.png")
	unsavedVersion, unsavedHistory := untitled.Doc.Buffer.Version(), untitled.Doc.Buffer.History()
	clickMarkdownLink(t, unsavedTester, "图片引用：未保存")
	if len(unsaved.Tabs) != 1 || unsaved.Current() != untitled || !strings.Contains(unsaved.Message, "尚未保存") ||
		untitled.Doc.Buffer.Version() != unsavedVersion || untitled.Doc.Buffer.History() != unsavedHistory {
		t.Fatal("unsaved relative reference silently resolved", unsaved.Message)
	}
}
