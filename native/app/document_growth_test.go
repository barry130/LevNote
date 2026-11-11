package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

func TestGrowthAppRealMarkdownSaveSwitchAndRecover(t *testing.T) {
	const source = "# 标题\r\n\r\n正文\r\n"
	root := t.TempDir()
	path := filepath.Join(root, "growth.md")
	disk := append([]byte{0xef, 0xbb, 0xbf}, []byte(source)...)
	if err := os.WriteFile(path, disk, 0600); err != nil {
		t.Fatal(err)
	}
	doc, err := document.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	doc.Buffer = text.NewWithLimit(doc.Buffer.String(), len(source))
	a := newTestApp(t, filepath.Join(root, "config"))
	origin := a.add(doc, "growth.md", "")
	if origin == nil || origin.Adapter == nil || origin.Adapter.ID != "markdown" || origin.Doc.Dirty() {
		t.Fatal("real adapter fixture did not retain file baseline")
	}
	other := a.appendTab(document.New("other"), "other.txt", "", false, false)
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	beginRealMarkdownComposition(t, origin, tester)
	view := origin.View
	version, selection, history, snapshot := doc.Buffer.Version(), doc.Buffer.Selection(), doc.Buffer.History(), doc.Buffer.Snapshot()
	if a.Save(origin, false) || !strings.Contains(a.Message, "内容上限") || !strings.Contains(a.Message, "缩短") {
		t.Fatal("save did not visibly stop on refused rich flush", a.Message)
	}
	if err = editor.FlushView(view); !errors.Is(err, text.ErrSizeLimit) {
		t.Fatal("wrapped rich error lost identity", err)
	}
	if a.setMode(origin, "source") || a.Select(1) || a.CloseTab(0) || a.Current() != origin || origin.View != view || origin.Mode != "preview" {
		t.Fatal("mode/tab switch discarded refused draft", a.Message)
	}
	tester.Frame()
	if !tester.HasText("打开未同步副本") || doc.Buffer.Version() != version || doc.Buffer.Selection() != selection || doc.Buffer.History() != history ||
		!doc.Buffer.EqualSnapshot(snapshot) || doc.Dirty() || !bytes.Equal(mustReadFile(t, path), disk) || !doc.BOM || doc.Encoding != document.UTF8 {
		t.Fatal("refused save/switch changed source/disk or removed recovery")
	}
	recoverer, ok := view.(interface{ RecoverDraft() error })
	if !ok {
		t.Fatal("real view lacks recovery")
	}
	if err = recoverer.RecoverDraft(); err != nil {
		t.Fatal(err)
	}
	recovered := a.Current()
	if recovered == origin || recovered == other || recovered.Doc.Buffer.String() != "中文🙂" || !recovered.Doc.IsNew || recovered.Doc.Path != "" ||
		!bytes.Equal(mustReadFile(t, path), disk) || doc.Buffer.Version() != version {
		t.Fatal("real app recovery lost data or overwrote original")
	}
	copyPath := filepath.Join(root, "recovered.txt")
	a.saveDialog = func(*Tab) (string, error) { return copyPath, nil }
	if !a.Save(recovered, true) || string(mustReadFile(t, copyPath)) != "中文🙂" {
		t.Fatal("recovered full draft could not be saved", a.Message)
	}
	if !a.Select(0) || !a.Save(origin, false) || !bytes.Equal(mustReadFile(t, path), disk) {
		t.Fatal("recovered owner did not return to intact untouched save", a.Message)
	}
}

func TestGrowthAppSourceReportsRefusedPaste(t *testing.T) {
	a := newTestApp(t, filepath.Join(t.TempDir(), "config"))
	doc := document.New("甲")
	doc.Buffer = text.NewWithLimit("甲", 4)
	tab := a.add(doc, "growth.txt", "")
	tab.Editor.Paste = func() (string, error) { return "ab", nil }
	tab.Editor.Reveal(1, 1)
	version, selection := doc.Buffer.Version(), doc.Buffer.Selection()
	tab.Editor.Command("paste")
	if doc.Buffer.String() != "甲" || doc.Buffer.Version() != version || doc.Buffer.Selection() != selection ||
		!strings.Contains(a.Message, "4 字节") || !strings.Contains(a.Message, "缩短") || len(a.Issues) == 0 {
		t.Fatal("source size refusal was silent or destructive", a.Message)
	}
}

func TestGrowthAppSourceCompositionBlocksLifecycleUntilRetryOrCancel(t *testing.T) {
	for _, started := range []bool{false, true} {
		for _, finish := range []string{"retry", "cancel"} {
			t.Run(map[bool]string{false: "first", true: "ongoing"}[started]+"/"+finish, func(t *testing.T) {
				root := t.TempDir()
				path := filepath.Join(root, "composition.txt")
				if err := os.WriteFile(path, []byte("a"), 0600); err != nil {
					t.Fatal(err)
				}
				doc, err := document.Open(path)
				if err != nil {
					t.Fatal(err)
				}
				doc.Buffer = text.NewWithLimit("a", 4)
				a := newTestApp(t, filepath.Join(root, "config"))
				origin := a.add(doc, "composition.txt", "")
				other := a.appendTab(document.New("other"), "other.txt", "", false, false)
				closeCalls := 0
				a.closeDialog = func(*Tab) (closeDecision, error) { closeCalls++; return closeDiscard, nil }
				tester := ui.NewTester(a.View, 760, 480)
				if err := tester.Click("文档编辑区"); err != nil {
					t.Fatal(err)
				}
				input, view, buffer := origin.Editor, origin.View, doc.Buffer
				input.Reveal(1, 1)
				if started {
					tester.Compose("ni", 2)
				}
				tester.Compose("🙂", 1)
				mark, composing := input.MarkedRange()
				version, selection, history, snapshot := buffer.Version(), buffer.Selection(), buffer.History(), buffer.Snapshot()
				check := func() {
					t.Helper()
					after, active := input.MarkedRange()
					if a.Current() != origin || len(a.Tabs) != 2 || a.Tabs[1] != other || origin.Doc != doc || doc.Buffer != buffer || origin.Editor != input || origin.View != view ||
						after != mark || active != composing || active != started || buffer.InGroup() != started || buffer.Version() != version || buffer.Selection() != selection || buffer.History() != history || !buffer.EqualSnapshot(snapshot) ||
						!errors.Is(input.Flush(), text.ErrSizeLimit) || string(mustReadFile(t, path)) != "a" || closeCalls != 0 {
						t.Fatal("refused lifecycle lost composition/identity/disk", a.Message)
					}
				}
				for _, action := range []func() bool{func() bool { return a.Save(origin, false) }, func() bool { return a.Select(1) }, func() bool { return a.CloseTab(0) }, a.CanClose} {
					if action() || !strings.Contains(a.Message, "内容上限") || !strings.Contains(a.Message, "Esc") {
						t.Fatal("host accepted refused IME", a.Message)
					}
					check()
				}
				tester.SetFocused(false)
				tester.SetFocused(true)
				input.UnmarkText()
				input.ReleaseView()
				tester.Frame()
				check()
				if err := tester.Click("文档编辑区"); err != nil {
					t.Fatal(err)
				}
				tester.Key(0, ui.KeyLeft)
				check()
				want := "a"
				if finish == "retry" {
					tester.Compose("你", 1)
					want = "a你"
				} else {
					tester.Key(0, ui.KeyEscape)
				}
				if !a.Save(origin, false) || buffer.String() != want || buffer.InGroup() || string(mustReadFile(t, path)) != want || doc.Dirty() {
					t.Fatal("resolved input could not save", a.Message)
				}
				if !a.Select(1) || a.Current() != other || !a.Select(0) || !a.CloseTab(0) || a.Current() != other || closeCalls != 0 {
					t.Fatal("resolved input still blocked lifecycle", a.Message)
				}
			})
		}
	}
}
