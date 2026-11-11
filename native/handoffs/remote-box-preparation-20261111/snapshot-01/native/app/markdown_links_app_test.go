package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

const markdownLinksBody = "# 标题\r\n\r\n- [ ] 邻居任务\r\n\r\n[本地链接](子目录/target.txt) [锚点](#%E9%87%8D%E5%A4%8D-2) [外部链接](https://example.com/file)\r\n\r\n## 🙂重复\r\n\r\n第一处\r\n\r\n## 🙂重复\r\n\r\n第二处\r\n"

func markdownLinksFixture() []byte {
	return append([]byte{0xef, 0xbb, 0xbf}, []byte(markdownLinksBody)...)
}

type realMarkdownLinksFixture struct {
	app        *App
	tab        *Tab
	tester     *ui.Tester
	sourcePath string
	targetPath string
	before     []byte
}

func openRealMarkdownLinks(t *testing.T, content []byte, width, height int, readonly bool) realMarkdownLinksFixture {
	t.Helper()
	root := t.TempDir()
	sourcePath := filepath.Join(root, "links.md")
	targetDir := filepath.Join(root, "子目录")
	targetPath := filepath.Join(targetDir, "target.txt")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, content, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte("目标内容"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(sourcePath)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "markdown" ||
		tab.Mode != "preview" || tab.Doc == nil || tab.Doc.Buffer == nil {
		t.Fatalf("real Markdown link fixture did not open in preview: %#v", tab)
	}
	if readonly {
		tab.Doc.ReadOnly = true
		tab.Editor.ReadOnly = true
	}
	tester := ui.NewTester(a.View, width, height)
	tester.Frame()
	return realMarkdownLinksFixture{
		app: a, tab: tab, tester: tester, sourcePath: sourcePath,
		targetPath: targetPath, before: append([]byte(nil), content...),
	}
}

func requireMarkdownLinkAccess(t *testing.T, tester *ui.Tester, label, href string) ui.AccessInfo {
	t.Helper()
	name := "链接：" + boundedMarkdownLinkLabel(label)
	info, ok := tester.AccessByRole(ui.RoleLink, name)
	if !ok {
		t.Fatalf("link %q is missing from the accessibility tree: %#v", name, tester.Texts())
	}
	if info.Name != name || !info.Focusable || info.Disabled || info.Offscreen ||
		info.Bounds.W <= 0 || info.Bounds.H <= 0 {
		t.Fatalf("link navigation state for %q = %#v", name, info)
	}
	wantDescription := "Ctrl/Cmd+点击或 Tab 后 Enter 打开：" + href
	if info.Description != wantDescription {
		t.Fatalf("link %q tooltip = %q, want %q", name, info.Description, wantDescription)
	}
	return info
}

func boundedMarkdownLinkLabel(label string) string {
	const limit = 128
	if len(label) <= limit {
		return label
	}
	end := limit
	for end > 0 && !utf8.RuneStart(label[end]) {
		end--
	}
	return label[:end] + "…"
}

func clickMarkdownLink(t *testing.T, tester *ui.Tester, name string) {
	t.Helper()
	rect, ok := tester.Find(name)
	if !ok {
		t.Fatalf("first native fragment for %q is missing: %#v", name, tester.Texts())
	}
	tester.ClickAtWith(ui.Cmd, rect.X+4, rect.Y+rect.H/2)
	tester.Frame()
}

func linkOutlineAnchorIndex(a *App, anchor string) int {
	for i, item := range a.OutlineItems {
		if item.AnchorID == anchor {
			return i
		}
	}
	return -1
}

func TestMarkdownLinkRealAdapterLocalPointerKeyboardSaveReopen(t *testing.T) {
	for _, gesture := range []string{"pointer", "keyboard"} {
		t.Run(gesture, func(t *testing.T) {
			f := openRealMarkdownLinks(t, markdownLinksFixture(), 1180, 780, false)
			if gesture == "pointer" {
				if !f.app.setMode(f.tab, "source") {
					t.Fatalf("Markdown source mode failed: %s", f.app.Message)
				}
				f.tester.Frame()
				if _, ok := f.tester.AccessByRole(ui.RoleLink, ""); ok {
					t.Fatal("Markdown source mode exposed an authored preview RoleLink")
				}
				if !f.app.setMode(f.tab, "preview") {
					t.Fatalf("Markdown preview mode after source failed: %s", f.app.Message)
				}
				f.tester.Frame()
			}
			const name = "链接：本地链接"
			info := requireMarkdownLinkAccess(t, f.tester, "本地链接", "子目录/target.txt")
			version := f.tab.Doc.Buffer.Version()
			selection := f.tab.Doc.Buffer.Selection()
			history := f.tab.Doc.Buffer.History()
			if f.tab.Doc.Dirty() {
				t.Fatal("opening link fixture made the source dirty")
			}

			switch gesture {
			case "pointer":
				clickMarkdownLink(t, f.tester, name)
			case "keyboard":
				focused := false
				for attempt := 0; attempt < 80; attempt++ {
					current, _ := f.tester.AccessByRole(ui.RoleLink, name)
					if current.Focused {
						focused = true
						break
					}
					f.tester.Key(0, ui.KeyTab)
					f.tester.Frame()
				}
				if !focused {
					t.Fatalf("Tab did not focus the real Markdown link; initial=%#v", info)
				}
				f.tester.Key(0, ui.KeySpace)
				f.tester.Frame()
				if len(f.app.Tabs) != 1 {
					t.Fatal("Space activated the focused RoleLink")
				}
				f.tester.Key(0, ui.KeyEnter)
				f.tester.Frame()
			}

			if len(f.app.Tabs) != 2 || f.app.Current() == nil ||
				f.app.Current().Doc.Path != f.targetPath {
				t.Fatalf("%s did not open the local target exactly once: tabs=%d current=%#v",
					gesture, len(f.app.Tabs), f.app.Current())
			}
			if f.tab.Doc.Buffer.Version() != version ||
				f.tab.Doc.Buffer.Selection() != selection ||
				f.tab.Doc.Buffer.History() != history ||
				f.tab.Doc.Buffer.String() != markdownLinksBody ||
				f.tab.Doc.Dirty() {
				t.Fatal("link navigation changed source bytes, selection, history or dirty state")
			}
			if got, err := os.ReadFile(f.sourcePath); err != nil || !bytes.Equal(got, f.before) {
				t.Fatalf("link navigation changed on-disk bytes: len=%d err=%v", len(got), err)
			}

			if gesture == "pointer" {
				if !f.app.Save(f.tab, false) {
					t.Fatalf("saving unchanged linked source failed: %s", f.app.Message)
				}
				reopened := newTestApp(t, filepath.Join(t.TempDir(), "config"))
				reopened.Open(f.sourcePath)
				reopenedTab := reopened.Current()
				if reopenedTab == nil || reopenedTab.Adapter == nil ||
					reopenedTab.Adapter.ID != "markdown" || reopenedTab.Doc.Dirty() {
					t.Fatalf("linked source did not reopen unchanged: %#v", reopenedTab)
				}
				reopenedTester := ui.NewTester(reopened.View, 1180, 780)
				reopenedTester.Frame()
				requireMarkdownLinkAccess(t, reopenedTester, "本地链接", "子目录/target.txt")
				if got, err := os.ReadFile(f.sourcePath); err != nil || !bytes.Equal(got, f.before) {
					t.Fatalf("save/reopen changed BOM or CRLF bytes: len=%d err=%v", len(got), err)
				}
			}
		})
	}
}

func TestMarkdownLinkRealAdapterReadonlyAnchorsAndRejection(t *testing.T) {
	f := openRealMarkdownLinks(t, markdownLinksFixture(), 1180, 780, true)
	anchorInfo := requireMarkdownLinkAccess(t, f.tester, "锚点", "#%E9%87%8D%E5%A4%8D-2")
	if anchorInfo.Disabled {
		t.Fatal("readonly Markdown link was disabled")
	}
	version := f.tab.Doc.Buffer.Version()
	selection := f.tab.Doc.Buffer.Selection()
	history := f.tab.Doc.Buffer.History()
	view := f.tab.View

	clickMarkdownLink(t, f.tester, "链接：锚点")
	if f.tab.Mode != "preview" || f.tab.View != view ||
		f.tab.Doc.Buffer.Version() != version ||
		f.tab.Doc.Buffer.Selection() != selection ||
		f.tab.Doc.Buffer.History() != history ||
		f.tab.Doc.Dirty() {
		t.Fatalf("readonly preview anchor changed source/view state: mode=%q view=%p/%p message=%q",
			f.tab.Mode, f.tab.View, view, f.app.Message)
	}
	if !strings.Contains(f.app.Message, "已定位：") {
		t.Fatalf("readonly preview anchor did not navigate visibly: %q", f.app.Message)
	}
	if source, ok := f.tester.Access("文档编辑区"); ok && source.Focused {
		t.Fatal("preview anchor activated the hidden source editor")
	}

	if !f.app.setMode(f.tab, "split") {
		t.Fatalf("readonly Markdown split mode failed: %s", f.app.Message)
	}
	f.app.ShowOutline = true
	f.tester.Frame()
	f.tester.Frame()
	targetIndex := linkOutlineAnchorIndex(f.app, "重复-2")
	if targetIndex < 0 {
		t.Fatalf("duplicate Unicode anchor was not exposed by the outline provider: %#v", f.app.OutlineItems)
	}
	targetOffset := f.app.OutlineItems[targetIndex].Offset
	requireMarkdownLinkAccess(t, f.tester, "锚点", "#%E9%87%8D%E5%A4%8D-2")
	clickMarkdownLink(t, f.tester, "链接：锚点")
	if f.tab.Mode != "split" ||
		f.tab.Doc.Buffer.Selection().Start() != targetOffset ||
		f.tab.Doc.Buffer.Selection().End() != targetOffset {
		t.Fatalf("readonly split anchor = mode %q selection %#v, want UTF-16 offset %d; message=%q",
			f.tab.Mode, f.tab.Doc.Buffer.Selection(), targetOffset, f.app.Message)
	}
	if !strings.Contains(f.app.Message, "已定位：") {
		t.Fatalf("readonly split anchor did not navigate visibly: %q", f.app.Message)
	}

	requireMarkdownLinkAccess(t, f.tester, "外部链接", "https://example.com/file")
	version = f.tab.Doc.Buffer.Version()
	selection = f.tab.Doc.Buffer.Selection()
	history = f.tab.Doc.Buffer.History()
	sourceBefore := f.tab.Doc.Buffer.String()
	tabsBefore := len(f.app.Tabs)
	clickMarkdownLink(t, f.tester, "链接：外部链接")
	if len(f.app.Tabs) != tabsBefore ||
		f.tab.Doc.Buffer.Version() != version ||
		f.tab.Doc.Buffer.Selection() != selection ||
		f.tab.Doc.Buffer.History() != history ||
		f.tab.Doc.Buffer.String() != sourceBefore ||
		f.tab.Doc.Dirty() {
		t.Fatal("unsupported Markdown URL changed tabs, source, selection or history")
	}
	if !strings.Contains(f.app.Message, "不会打开网络地址") {
		t.Fatalf("unsupported Markdown URL refusal is not visible: %q", f.app.Message)
	}
	if source, ok := f.tester.Access("文档编辑区"); ok && source.Focused {
		t.Fatal("unsupported Markdown URL moved focus into the source editor")
	}
	if got, err := os.ReadFile(f.sourcePath); err != nil || !bytes.Equal(got, f.before) {
		t.Fatalf("readonly/rejection navigation changed file bytes: len=%d err=%v", len(got), err)
	}
}

func TestMarkdownLinkRealAdapterFlushBarriers(t *testing.T) {
	t.Run("successful", func(t *testing.T) {
		f := openRealMarkdownLinks(t, markdownLinksFixture(), 1180, 780, false)
		beginRealMarkdownComposition(t, f.tab, f.tester)
		f.tester.Frame()
		if !strings.Contains(f.tab.Doc.Buffer.String(), "# 标题") ||
			strings.Contains(f.tab.Doc.Buffer.String(), "# 中文🙂") {
			t.Fatalf("composition was not pending in another block: %q", f.tab.Doc.Buffer.String())
		}
		requireMarkdownLinkAccess(t, f.tester, "本地链接", "子目录/target.txt")
		clickMarkdownLink(t, f.tester, "链接：本地链接")
		if len(f.app.Tabs) != 2 || f.app.Current() == nil ||
			f.app.Current().Doc.Path != f.targetPath {
			t.Fatalf("successful Flush did not open local target once: tabs=%d message=%q",
				len(f.app.Tabs), f.app.Message)
		}
		if !strings.Contains(f.tab.Doc.Buffer.String(), "# 中文🙂\r\n") {
			t.Fatalf("link navigation lost the successful Markdown Compose: %q", f.tab.Doc.Buffer.String())
		}
	})

	t.Run("failed", func(t *testing.T) {
		f := openRealMarkdownLinks(t, markdownLinksFixture(), 1180, 780, false)
		beginRealMarkdownComposition(t, f.tab, f.tester)
		f.tester.Frame()
		if err := f.tab.Doc.Buffer.ReplaceUTF16(2, 4, "外部"); err != nil {
			t.Fatal(err)
		}
		f.tester.Frame()
		version := f.tab.Doc.Buffer.Version()
		selection := f.tab.Doc.Buffer.Selection()
		history := f.tab.Doc.Buffer.History()
		sourceBefore := f.tab.Doc.Buffer.String()
		viewBefore := f.tab.View
		requireMarkdownLinkAccess(t, f.tester, "本地链接", "子目录/target.txt")
		clickMarkdownLink(t, f.tester, "链接：本地链接")
		if len(f.app.Tabs) != 1 || f.app.Current() != f.tab ||
			f.tab.Doc.Buffer.Version() != version ||
			f.tab.Doc.Buffer.Selection() != selection ||
			f.tab.Doc.Buffer.History() != history ||
			f.tab.Doc.Buffer.String() != sourceBefore {
			t.Fatal("failed link Flush opened a target or changed source/selection/history")
		}
		if f.tab.View != viewBefore || f.tab.Mode != "preview" || f.tab.Editor == nil {
			t.Fatalf("failed link Flush replaced the active draft view: mode=%q", f.tab.Mode)
		}
		if !strings.Contains(f.app.Message, "草稿仍保留") {
			t.Fatalf("failed link Flush refusal is not visible: %q", f.app.Message)
		}
		if !f.tester.HasText("完成编辑") || !f.tester.HasText("打开未同步副本") {
			t.Fatalf("failed Flush discarded draft recovery controls: %#v", f.tester.Texts())
		}
		if got, err := os.ReadFile(f.sourcePath); err != nil || !bytes.Equal(got, f.before) {
			t.Fatalf("failed link Flush changed file bytes on disk: len=%d err=%v", len(got), err)
		}
	})
}

func TestMarkdownLinksNarrowScreenshot(t *testing.T) {
	out := os.Getenv("LEVNOTE_MARKDOWN_LINKS_SCREENSHOT_DIR")
	if out == "" {
		t.Skip("set LEVNOTE_MARKDOWN_LINKS_SCREENSHOT_DIR for link visual acceptance")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	label := strings.Repeat("本地链接与换行🙂", 7)
	body := "# 链接验收\r\n\r\n- [ ] 邻居任务\r\n\r\n[" + label + "](https://example.com/local-only-wrapped-link)\r\n"
	content := append([]byte{0xef, 0xbb, 0xbf}, []byte(body)...)
	f := openRealMarkdownLinks(t, content, 760, 480, false)
	f.app.Dark = true
	f.tester.SetDark(true)
	f.tester.Frame()
	info := requireMarkdownLinkAccess(t, f.tester, label, "https://example.com/local-only-wrapped-link")
	if info.Bounds.H < 32 {
		t.Fatalf("narrow fixture did not wrap the native link: %#v", info.Bounds)
	}
	task, ok := f.tester.AccessByRole(ui.RoleCheckBox, "未完成任务：邻居任务")
	if !ok || task.Bounds.W <= 0 || task.Bounds.H <= 0 || task.Offscreen {
		t.Fatalf("narrow link fixture lost its neighboring task control: %#v", task)
	}
	clickMarkdownLink(t, f.tester, "链接："+boundedMarkdownLinkLabel(label))
	if !strings.Contains(f.app.Message, "不会打开网络地址") {
		t.Fatalf("narrow wrapped-link refusal is not visible: %q", f.app.Message)
	}
	f.tester.Advance(time.Second)
	f.tester.Frame()
	const statusTop = float32(480 - 29)
	if info.Bounds.X < 0 || info.Bounds.Y < 0 ||
		info.Bounds.X+info.Bounds.W > 760 ||
		info.Bounds.Y+info.Bounds.H > statusTop ||
		task.Bounds.Y+task.Bounds.H > statusTop {
		t.Fatalf("narrow wrapped link or task overlaps chrome/status: link=%#v task=%#v",
			info.Bounds, task.Bounds)
	}
	captureAcceptanceTester(t, filepath.Join(out, "markdown-links-wrapped-760x480-dark.png"), f.tester)
}

func TestEditorHostStaleLocalReferenceIsRefused(t *testing.T) {
	type fixture struct {
		app        *App
		sourcePath string
		targetPath string
		captured   func(string)
	}
	openFixture := func(t *testing.T) fixture {
		t.Helper()
		root := t.TempDir()
		sourcePath := filepath.Join(root, "source.md")
		targetPath := filepath.Join(root, "target.txt")
		if err := os.WriteFile(sourcePath, []byte("[目标](target.txt)\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(targetPath, []byte("target"), 0600); err != nil {
			t.Fatal(err)
		}
		a := newTestApp(t, filepath.Join(root, "config"))
		a.Open(sourcePath)
		source := a.Current()
		if source == nil || source.Doc.Path != sourcePath {
			t.Fatalf("source tab = %#v", source)
		}
		return fixture{
			app:        a,
			sourcePath: sourcePath,
			targetPath: targetPath,
			captured:   a.editorHost(source).OpenReference,
		}
	}

	t.Run("closed", func(t *testing.T) {
		f := openFixture(t)
		f.app.removeTab(0)
		if len(f.app.Tabs) != 0 || f.app.Current() != nil {
			t.Fatalf("source tab was not removed: tabs=%#v current=%#v", f.app.Tabs, f.app.Current())
		}
		f.app.Message = ""
		f.captured("target.txt")
		if len(f.app.Tabs) != 0 || f.app.Current() != nil {
			t.Fatalf("stale local reference opened %d tab(s); current=%#v", len(f.app.Tabs), f.app.Current())
		}
		if !strings.Contains(f.app.Message, "已关闭") {
			t.Fatalf("stale reference did not report a closed source: %q", f.app.Message)
		}
	})

	t.Run("inactive", func(t *testing.T) {
		f := openFixture(t)
		otherPath := filepath.Join(filepath.Dir(f.sourcePath), "other.txt")
		if err := os.WriteFile(otherPath, []byte("other"), 0600); err != nil {
			t.Fatal(err)
		}
		f.app.Open(otherPath)
		if len(f.app.Tabs) != 2 || f.app.Current() == nil || f.app.Current().Doc.Path != otherPath {
			t.Fatalf("could not make source inactive: tabs=%#v current=%#v", f.app.Tabs, f.app.Current())
		}
		f.app.Message = ""
		f.captured("target.txt")
		if len(f.app.Tabs) != 2 || f.app.Current() == nil || f.app.Current().Doc.Path != otherPath {
			t.Fatalf("inactive-source reference changed tabs=%d current=%#v", len(f.app.Tabs), f.app.Current())
		}
		if !strings.Contains(f.app.Message, "不再活动") {
			t.Fatalf("inactive-source refusal was not visible: %q", f.app.Message)
		}
	})
}
