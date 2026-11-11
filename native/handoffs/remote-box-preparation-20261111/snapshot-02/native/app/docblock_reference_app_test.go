package app

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	coretext "levnote.local/native/core/text"
)

const docBlockReferenceBody = "# 标题\r\n\r\n```doc-block\r\nid: shared\r\nkind: note\r\ntitle: 同名\r\nmode: inline\r\n---\r\n第一个内联块\r\n```\r\n\r\n```doc-block\r\nid: shared\r\nkind: text\r\ntitle: 同名\r\nmode: reference\r\nsrc: targets/a.txt\r\n---\r\n引用载荷\r\n```\r\n\r\n尾段\r\n"

type realDocBlockReferenceFixture struct {
	app        *App
	tab        *Tab
	tester     *ui.Tester
	sourcePath string
	targetA    string
	targetB    string
	before     []byte
}

func openRealDocBlockReference(t *testing.T, body string, width, height int, readonly bool) realDocBlockReferenceFixture {
	t.Helper()
	root := t.TempDir()
	sourcePath := filepath.Join(root, "references.md")
	targetDir := filepath.Join(root, "targets")
	targetA := filepath.Join(targetDir, "a.txt")
	targetB := filepath.Join(targetDir, "b.txt")
	content := append([]byte{0xef, 0xbb, 0xbf}, []byte(body)...)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, content, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetA, []byte("目标 A"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetB, []byte("目标 B"), 0600); err != nil {
		t.Fatal(err)
	}

	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(sourcePath)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "markdown" ||
		tab.Mode != "preview" || tab.Doc == nil || tab.Doc.Buffer == nil || tab.View != nil {
		t.Fatalf("real Markdown reference fixture did not open before view creation: %#v", tab)
	}
	if readonly {
		tab.Doc.ReadOnly = true
		tab.Editor.ReadOnly = true
	}
	tester := ui.NewTester(a.View, width, height)
	tester.Frame()
	if tab.View == nil {
		t.Fatal("registered Markdown adapter did not create a preview view")
	}
	return realDocBlockReferenceFixture{
		app: a, tab: tab, tester: tester, sourcePath: sourcePath,
		targetA: targetA, targetB: targetB, before: append([]byte(nil), content...),
	}
}

func requireDocBlockReferenceButton(t *testing.T, tester *ui.Tester, enabled bool) ui.AccessInfo {
	t.Helper()
	info, ok := tester.AccessByRole(ui.RoleButton, "打开引用")
	if !ok {
		t.Fatalf("doc-block reference button is missing: %#v", tester.Texts())
	}
	if info.Name != "打开引用" || info.Offscreen ||
		info.Bounds.W <= 0 || info.Bounds.H <= 0 {
		t.Fatalf("doc-block reference button geometry = %#v", info)
	}
	if enabled {
		if info.Disabled || !info.Focusable {
			t.Fatalf("enabled doc-block reference button state = %#v", info)
		}
	} else if !info.Disabled || info.Focusable {
		t.Fatalf("disabled doc-block reference button state = %#v", info)
	}
	if !utf8.ValidString(info.Description) || len(info.Description) > 515 ||
		(len(info.Description) > 512 && !strings.HasSuffix(info.Description, "…")) {
		t.Fatalf("doc-block reference tooltip is not bounded to 512 bytes plus ellipsis: bytes=%d text=%q",
			len(info.Description), info.Description)
	}
	return info
}

func clickDocBlockReference(t *testing.T, tester *ui.Tester, info ui.AccessInfo) {
	t.Helper()
	tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tester.Frame()
}

func focusDocBlockReference(t *testing.T, tester *ui.Tester, info ui.AccessInfo) {
	t.Helper()
	tester.Press(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tester.CancelPress()
	tester.Frame()
	if tester.Focused("打开引用") {
		return
	}
	for attempt := 0; attempt < 120; attempt++ {
		tester.Key(0, ui.KeyTab)
		tester.Frame()
		if tester.Focused("打开引用") {
			return
		}
	}
	t.Fatalf("doc-block reference button could not receive keyboard focus: %#v", tester.Texts())
}

func replaceDocBlockBufferText(t *testing.T, tab *Tab, old, replacement string) {
	t.Helper()
	source := tab.Doc.Buffer.String()
	index := strings.Index(source, old)
	if index < 0 {
		t.Fatalf("fixture text %q is absent from %q", old, source)
	}
	start := coretext.UTF16Len(source[:index])
	end := start + coretext.UTF16Len(old)
	if err := tab.Doc.Buffer.ReplaceUTF16(start, end, replacement); err != nil {
		t.Fatal(err)
	}
}

func TestDocBlockReferenceRealAdapterLocalGesturesFlushAndRelease(t *testing.T) {
	for _, gesture := range []string{"pointer", "enter", "space"} {
		t.Run(gesture, func(t *testing.T) {
			f := openRealDocBlockReference(t, docBlockReferenceBody, 1180, 780, false)
			historyBefore := f.tab.Doc.Buffer.History()
			beginRealMarkdownComposition(t, f.tab, f.tester)
			f.tester.Frame()
			if !strings.Contains(f.tab.Doc.Buffer.String(), "# 标题") ||
				strings.Contains(f.tab.Doc.Buffer.String(), "# 中文🙂") {
				t.Fatalf("Markdown composition was not pending in a different block: %q", f.tab.Doc.Buffer.String())
			}

			button := requireDocBlockReferenceButton(t, f.tester, true)
			switch gesture {
			case "pointer":
				clickDocBlockReference(t, f.tester, button)
			case "enter", "space":
				focusDocBlockReference(t, f.tester, button)
				if len(f.app.Tabs) != 1 || f.app.Current() != f.tab {
					t.Fatalf("%s canceled press activated the reference: tabs=%d current=%#v message=%q",
						gesture, len(f.app.Tabs), f.app.Current(), f.app.Message)
				}
				if gesture == "enter" {
					f.tester.Key(0, ui.KeyEnter)
				} else {
					f.tester.Key(0, ui.KeySpace)
				}
				f.tester.Frame()
			}

			if len(f.app.Tabs) != 2 || f.app.Current() == nil ||
				f.app.Current().Doc.Path != f.targetA {
				t.Fatalf("%s did not open the local target once: tabs=%d current=%#v message=%q",
					gesture, len(f.app.Tabs), f.app.Current(), f.app.Message)
			}
			if !strings.Contains(f.tab.Doc.Buffer.String(), "# 中文🙂\r\n") {
				t.Fatalf("%s did not Flush the pending Markdown composition: %q", gesture, f.tab.Doc.Buffer.String())
			}
			historyAfter := f.tab.Doc.Buffer.History()
			if historyAfter.UndoDepth != historyBefore.UndoDepth+1 ||
				historyAfter.RedoDepth != 0 {
				t.Fatalf("%s Flush committed %d undo entries (want exactly one) history=%#v",
					gesture, historyAfter.UndoDepth-historyBefore.UndoDepth, historyAfter)
			}
			if got, err := os.ReadFile(f.sourcePath); err != nil || !bytes.Equal(got, f.before) {
				t.Fatalf("%s reference navigation changed unsaved source bytes on disk: len=%d err=%v", gesture, len(got), err)
			}
		})
	}
}

func TestDocBlockReferenceRealAdapterStaleGestureAndFreshTargetIdentity(t *testing.T) {
	f := openRealDocBlockReference(t, docBlockReferenceBody, 1180, 780, false)
	button := requireDocBlockReferenceButton(t, f.tester, true)

	// Hold the pointer on the rendered reference, then replace its target before
	// releasing. A capture made from the release-time frame would silently open
	// B; the gesture must instead be refused because the pressed source identity
	// changed. The first and reference blocks deliberately share id/title, so a
	// same-ID/title fallback would also be caught.
	x, y := button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2
	f.tester.Press(x, y)
	replaceDocBlockBufferText(t, f.tab, "src: targets/a.txt", "src: targets/b.txt")
	f.tester.Release(x, y)
	f.tester.Frame()
	if len(f.app.Tabs) != 1 || f.app.Current() != f.tab {
		t.Fatalf("stale reference gesture opened a target: tabs=%d current=%#v message=%q",
			len(f.app.Tabs), f.app.Current(), f.app.Message)
	}
	if !strings.Contains(f.app.Message, "引用") {
		t.Fatalf("stale reference gesture has no visible refusal: %q", f.app.Message)
	}
	if got, err := os.ReadFile(f.sourcePath); err != nil || !bytes.Equal(got, f.before) {
		t.Fatalf("stale reference gesture changed disk bytes: len=%d err=%v", len(got), err)
	}

	button = requireDocBlockReferenceButton(t, f.tester, true)
	clickDocBlockReference(t, f.tester, button)
	if len(f.app.Tabs) != 2 || f.app.Current() == nil ||
		f.app.Current().Doc.Path != f.targetB {
		t.Fatalf("fresh gesture did not resolve the current exact target: tabs=%d current=%#v message=%q",
			len(f.app.Tabs), f.app.Current(), f.app.Message)
	}
	if strings.Contains(f.tab.Doc.Buffer.String(), "targets/a.txt") ||
		!strings.Contains(f.tab.Doc.Buffer.String(), "targets/b.txt") {
		t.Fatalf("fresh target identity did not follow the current source: %q", f.tab.Doc.Buffer.String())
	}
}

func TestDocBlockReferenceRealAdapterUnrelatedInsertionRebases(t *testing.T) {
	f := openRealDocBlockReference(t, docBlockReferenceBody, 1180, 780, false)
	button := requireDocBlockReferenceButton(t, f.tester, true)
	x, y := button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2
	f.tester.Press(x, y)
	if err := f.tab.Doc.Buffer.ReplaceUTF16(0, 0, "前插\r\n"); err != nil {
		t.Fatal(err)
	}
	f.tester.Release(x, y)
	f.tester.Frame()

	if len(f.app.Tabs) != 2 || f.app.Current() == nil ||
		f.app.Current().Doc.Path != f.targetA {
		t.Fatalf("insertion ahead did not rebase the captured reference: tabs=%d current=%#v message=%q",
			len(f.app.Tabs), f.app.Current(), f.app.Message)
	}
	if !strings.Contains(f.tab.Doc.Buffer.String(), "前插\r\n# 标题") {
		t.Fatalf("insertion ahead was not retained in the source: %q", f.tab.Doc.Buffer.String())
	}
}

func TestDocBlockReferenceRealAdapterAnchorNavigation(t *testing.T) {
	body := "# 目标\r\n\r\n```doc-block\r\nid: anchor\r\nkind: heading\r\nmode: reference\r\ntitle: 跳到目标\r\nsrc: #目标\r\n---\r\n```\r\n"
	f := openRealDocBlockReference(t, body, 1180, 780, false)
	version := f.tab.Doc.Buffer.Version()
	selection := f.tab.Doc.Buffer.Selection()
	history := f.tab.Doc.Buffer.History()
	sourceBefore := f.tab.Doc.Buffer.String()
	button := requireDocBlockReferenceButton(t, f.tester, true)
	clickDocBlockReference(t, f.tester, button)

	if len(f.app.Tabs) != 1 || f.app.Current() != f.tab {
		t.Fatalf("anchor reference opened another tab: tabs=%d current=%#v message=%q",
			len(f.app.Tabs), f.app.Current(), f.app.Message)
	}
	if !strings.Contains(f.app.Message, "已定位：目标") {
		t.Fatalf("anchor reference did not report its navigation: %q", f.app.Message)
	}
	if f.tab.Doc.Buffer.Version() != version ||
		f.tab.Doc.Buffer.Selection() != selection ||
		f.tab.Doc.Buffer.History() != history ||
		f.tab.Doc.Buffer.String() != sourceBefore || f.tab.Doc.Dirty() {
		t.Fatal("anchor reference navigation mutated source, selection, history or dirty state")
	}
}

func TestDocBlockReferenceRealAdapterReadonlyFileURLAndRefusal(t *testing.T) {
	t.Run("readonly-local", func(t *testing.T) {
		f := openRealDocBlockReference(t, docBlockReferenceBody, 1180, 780, true)
		button := requireDocBlockReferenceButton(t, f.tester, true)
		version := f.tab.Doc.Buffer.Version()
		selection := f.tab.Doc.Buffer.Selection()
		history := f.tab.Doc.Buffer.History()
		sourceBefore := f.tab.Doc.Buffer.String()
		clickDocBlockReference(t, f.tester, button)
		if len(f.app.Tabs) != 2 || f.app.Current() == nil ||
			f.app.Current().Doc.Path != f.targetA {
			t.Fatalf("readonly reference did not navigate: tabs=%d current=%#v message=%q",
				len(f.app.Tabs), f.app.Current(), f.app.Message)
		}
		if f.tab.Doc.Buffer.Version() != version ||
			f.tab.Doc.Buffer.Selection() != selection ||
			f.tab.Doc.Buffer.History() != history ||
			f.tab.Doc.Buffer.String() != sourceBefore || f.tab.Doc.Dirty() {
			t.Fatal("readonly reference navigation mutated source, selection, history or dirty state")
		}
		if got, err := os.ReadFile(f.sourcePath); err != nil || !bytes.Equal(got, f.before) {
			t.Fatalf("readonly reference navigation changed disk bytes: len=%d err=%v", len(got), err)
		}
	})

	t.Run("file-url", func(t *testing.T) {
		root := t.TempDir()
		target := filepath.Join(root, "file url target.txt")
		fileURL := (&url.URL{Scheme: "file", Path: "/" + filepath.ToSlash(target)}).String()
		body := "# 标题\r\n\r\n```doc-block\r\nid: file-url\r\nkind: text\r\nmode: reference\r\ntitle: 文件 URL\r\nsrc: " + fileURL + "\r\n---\r\n```\r\n"
		f := openRealDocBlockReference(t, body, 1180, 780, false)
		if err := os.WriteFile(target, []byte("URL 目标"), 0600); err != nil {
			t.Fatal(err)
		}
		button := requireDocBlockReferenceButton(t, f.tester, true)
		clickDocBlockReference(t, f.tester, button)
		if len(f.app.Tabs) != 2 || f.app.Current() == nil ||
			!strings.EqualFold(filepath.Clean(f.app.Current().Doc.Path), filepath.Clean(target)) {
			t.Fatalf("file URL reference did not open its local target: tabs=%d current=%#v message=%q",
				len(f.app.Tabs), f.app.Current(), f.app.Message)
		}
	})

	t.Run("remote-refusal", func(t *testing.T) {
		body := "# 标题\r\n\r\n```doc-block\r\nid: remote\r\nkind: text\r\nmode: reference\r\ntitle: 远程\r\nsrc: https://example.com/no-open.txt\r\n---\r\n```\r\n"
		f := openRealDocBlockReference(t, body, 1180, 780, false)
		button := requireDocBlockReferenceButton(t, f.tester, true)
		version := f.tab.Doc.Buffer.Version()
		selection := f.tab.Doc.Buffer.Selection()
		history := f.tab.Doc.Buffer.History()
		sourceBefore := f.tab.Doc.Buffer.String()
		clickDocBlockReference(t, f.tester, button)
		if len(f.app.Tabs) != 1 || f.app.Current() != f.tab ||
			f.tab.Doc.Buffer.Version() != version ||
			f.tab.Doc.Buffer.Selection() != selection ||
			f.tab.Doc.Buffer.History() != history ||
			f.tab.Doc.Buffer.String() != sourceBefore || f.tab.Doc.Dirty() {
			t.Fatal("remote reference refusal opened a target or mutated source")
		}
		if !strings.Contains(f.app.Message, "不会打开网络地址") {
			t.Fatalf("remote reference refusal is not visible: %q", f.app.Message)
		}
		if got, err := os.ReadFile(f.sourcePath); err != nil || !bytes.Equal(got, f.before) {
			t.Fatalf("remote reference refusal changed disk bytes: len=%d err=%v", len(got), err)
		}
	})
}

func TestDocBlockReferenceRealAdapterFailedFlushRetainsDraft(t *testing.T) {
	f := openRealDocBlockReference(t, docBlockReferenceBody, 1180, 780, false)
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
	button := requireDocBlockReferenceButton(t, f.tester, true)
	clickDocBlockReference(t, f.tester, button)

	if len(f.app.Tabs) != 1 || f.app.Current() != f.tab ||
		f.tab.Doc.Buffer.Version() != version ||
		f.tab.Doc.Buffer.Selection() != selection ||
		f.tab.Doc.Buffer.History() != history ||
		f.tab.Doc.Buffer.String() != sourceBefore {
		t.Fatal("failed reference Flush opened a target or changed source/selection/history")
	}
	if f.tab.View != viewBefore || f.tab.Mode != "preview" || f.tab.Editor == nil {
		t.Fatalf("failed reference Flush replaced the active draft view: mode=%q", f.tab.Mode)
	}
	if !strings.Contains(f.app.Message, "草稿仍保留") {
		t.Fatalf("failed reference Flush refusal is not visible: %q", f.app.Message)
	}
	if !f.tester.HasText("完成编辑") || !f.tester.HasText("打开未同步副本") {
		t.Fatalf("failed reference Flush discarded draft recovery controls: %#v", f.tester.Texts())
	}
	if got, err := os.ReadFile(f.sourcePath); err != nil || !bytes.Equal(got, f.before) {
		t.Fatalf("failed reference Flush changed file bytes on disk: len=%d err=%v", len(got), err)
	}
}

func TestDocBlockReferenceOver4096TargetDisabled(t *testing.T) {
	target := strings.Repeat("a", 4097)
	body := fmt.Sprintf("# 标题\r\n\r\n```doc-block\r\nid: long\r\nkind: text\r\nmode: reference\r\ntitle: 过长\r\nsrc: %s\r\n---\r\n```\r\n", target)
	f := openRealDocBlockReference(t, body, 1180, 780, false)
	button := requireDocBlockReferenceButton(t, f.tester, false)
	version := f.tab.Doc.Buffer.Version()
	selection := f.tab.Doc.Buffer.Selection()
	history := f.tab.Doc.Buffer.History()
	sourceBefore := f.tab.Doc.Buffer.String()
	tabsBefore := len(f.app.Tabs)
	// Tester.Click reports lookup success independently of activation. A raw
	// pointer event at the disabled control's center is the actual no-op check.
	f.tester.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
	f.tester.Frame()
	if len(f.app.Tabs) != tabsBefore ||
		f.tab.Doc.Buffer.Version() != version ||
		f.tab.Doc.Buffer.Selection() != selection ||
		f.tab.Doc.Buffer.History() != history ||
		f.tab.Doc.Buffer.String() != sourceBefore || f.tab.Doc.Dirty() {
		t.Fatal("disabled over-bound reference changed tabs, source, selection or history")
	}
	if !f.tester.HasText("引用不可用") {
		t.Fatalf("disabled over-bound reference has no visible reason: %#v", f.tester.Texts())
	}
}

func TestDocBlockReferenceNarrowScreenshot(t *testing.T) {
	out := os.Getenv("LEVNOTE_DOCBLOCK_REFERENCE_SCREENSHOT_DIR")
	if out == "" {
		t.Skip("set LEVNOTE_DOCBLOCK_REFERENCE_SCREENSHOT_DIR for doc-block reference visual acceptance")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	for _, dark := range []bool{false, true} {
		theme := "light"
		if dark {
			theme = "dark"
		}
		t.Run("normal-"+theme, func(t *testing.T) {
			body := "# 引用验收\r\n\r\n- [ ] 邻居任务\r\n\r\n```doc-block\r\nid: visual\r\nkind: document\r\ntitle: 本地链接与远程引用\r\nmode: reference\r\nsrc: targets/a.txt\r\n---\r\n引用载荷保持可见\r\n```\r\n\r\n尾段\r\n"
			f := openRealDocBlockReference(t, body, 760, 480, false)
			f.app.Dark = dark
			f.tester.SetDark(dark)
			f.tester.Frame()
			button := requireDocBlockReferenceButton(t, f.tester, true)
			requireDocBlockReferenceControlWithinStatus(t, button)
			captureAcceptanceTester(t, filepath.Join(out, "docblock-reference-normal-760x480-"+theme+".png"), f.tester)
		})
		t.Run("refused-"+theme, func(t *testing.T) {
			body := "# 引用验收\r\n\r\n- [ ] 邻居任务\r\n\r\n```doc-block\r\nid: visual\r\nkind: document\r\ntitle: 本地链接与远程引用\r\nmode: reference\r\nsrc: https://example.com/local-only-wrapped-reference\r\n---\r\n引用载荷保持可见\r\n```\r\n\r\n尾段\r\n"
			f := openRealDocBlockReference(t, body, 760, 480, false)
			f.app.Dark = dark
			f.tester.SetDark(dark)
			f.tester.Frame()
			button := requireDocBlockReferenceButton(t, f.tester, true)
			clickDocBlockReference(t, f.tester, button)
			if !strings.Contains(f.app.Message, "不会打开网络地址") {
				t.Fatalf("narrow reference refusal is not visible: %q", f.app.Message)
			}
			button = requireDocBlockReferenceButton(t, f.tester, true)
			requireDocBlockReferenceControlWithinStatus(t, button)
			captureAcceptanceTester(t, filepath.Join(out, "docblock-reference-refused-760x480-"+theme+".png"), f.tester)
		})
	}
}

func requireDocBlockReferenceControlWithinStatus(t *testing.T, button ui.AccessInfo) {
	t.Helper()
	const statusTop = float32(480 - 29)
	if button.Bounds.X < 0 || button.Bounds.Y < 0 ||
		button.Bounds.X+button.Bounds.W > 760 ||
		button.Bounds.Y+button.Bounds.H > statusTop {
		t.Fatalf("narrow doc-block reference control overlaps chrome/status: %#v", button.Bounds)
	}
}
