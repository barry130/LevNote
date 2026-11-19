package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/session"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

func openDocumentsFixture(t *testing.T) (*App, *Tab, *Tab, *ui.Tester) {
	t.Helper()
	a := newTestApp(t, t.TempDir())
	first := a.add(document.New("first untouched"), "alpha🙂.txt", "")
	second := a.appendTab(document.New("second untouched"), "BETA.txt", "", false, false)
	tester := ui.NewTester(a.View, 760, 480)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	return a, first, second, tester
}

func pickerClick(t *testing.T, tester *ui.Tester, name string) {
	t.Helper()
	tester.Advance(time.Second)
	if err := tester.Click(name); err != nil {
		t.Fatal(err)
	}
}

func pickerOpen(t *testing.T, a *App, tester *ui.Tester) {
	t.Helper()
	tester.Key(ui.Cmd, ui.KeyP)
	if !a.openDocuments.open || !tester.Focused(openDocumentsQueryLabel) {
		t.Fatalf("picker not focused: open=%v message=%s texts=%v", a.openDocuments.open, a.Message, tester.Texts())
	}
}

func pickerRowName(tab *Tab) string { return fmt.Sprintf("选择文档 %d：%s", tab.ID, tab.Name) }

func TestOpenDocumentsModalKeyboardAndOpener(t *testing.T) {
	a, first, second, tester := openDocumentsFixture(t)
	history, version, selection := first.Doc.Buffer.History(), first.Doc.Buffer.Version(), first.Doc.Buffer.Selection()
	pickerOpen(t, a, tester)
	tester.Key(0, ui.KeyTab)
	if tester.Focused(openDocumentsQueryLabel) {
		t.Fatal("Tab did not leave query")
	}
	tester.Key(ui.Cmd, ui.KeyP)
	if !tester.Focused(openDocumentsQueryLabel) {
		t.Fatal("repeat Ctrl+P inside modal failed")
	}
	tester.Key(0, ui.KeyEscape)
	if a.openDocuments.open || !tester.Focused("文档编辑区") || a.Current() != first {
		t.Fatalf("Escape opener not restored: open=%v current=%v texts=%v", a.openDocuments.open, a.Current().Name, tester.Texts())
	}
	pickerClick(t, tester, "已打开文档")
	if !a.openDocuments.open || !tester.Focused(openDocumentsQueryLabel) {
		t.Fatal("toolbar entry did not open")
	}
	tester.Type("BeTa")
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	if a.openDocuments.open || a.Current() != second || !tester.Focused("文档编辑区") {
		t.Fatalf("destination focus not delivered: current=%s open=%v", a.Current().Name, a.openDocuments.open)
	}
	tester.Type("!")
	if !strings.Contains(second.Doc.Buffer.String(), "!") || first.Doc.Buffer.String() != "first untouched" || first.Doc.Buffer.History() != history || first.Doc.Buffer.Version() != version || first.Doc.Buffer.Selection() != selection || len(a.Tabs) != 2 {
		t.Fatal("filter or focus changed original document")
	}
	if a.openDocuments.rows != nil || a.openDocuments.matches != nil || a.openDocuments.list.Key != nil || a.openDocuments.query != "" {
		t.Fatal("closed picker retained metadata")
	}
}

func TestOpenDocumentsModalQueryComposition(t *testing.T) {
	for _, finish := range []string{"commit", "cancel"} {
		t.Run(finish, func(t *testing.T) {
			a, first, second, tester := openDocumentsFixture(t)
			var nativeComposing bool
			tester = ui.NewTester(func(c *ui.Context) {
				a.View(c)
				if a.openDocuments.open {
					nativeComposing = c.Resolve(a.openDocuments.queryHandle).Composing()
				}
			}, 760, 480)
			pickerClick(t, tester, "文档编辑区")
			pickerOpen(t, a, tester)
			firstBefore, secondBefore := first.Doc.Buffer.Snapshot(), second.Doc.Buffer.Snapshot()
			firstVersion, secondVersion := first.Doc.Buffer.Version(), second.Doc.Buffer.Version()
			firstHistory, secondHistory := first.Doc.Buffer.History(), second.Doc.Buffer.History()
			firstSelection, secondSelection := first.Doc.Buffer.Selection(), second.Doc.Buffer.Selection()
			tester.SetClipboard("clipboard untouched")
			tester.Compose("B", 1)
			preeditCaret, active := tester.TextCaret()
			if !active {
				t.Fatal("missing query IME caret")
			}
			check := func(action string) {
				t.Helper()
				if !a.openDocuments.open || !a.openDocuments.composing || !tester.Focused(openDocumentsQueryLabel) || a.openDocuments.query != "" || a.openDocuments.filteredQuery != "" || a.openDocuments.selectedID != first.ID || a.openDocuments.pendingID != 0 || a.openDocuments.pendingClose || a.Current() != first || len(a.Tabs) != 2 {
					t.Fatalf("%s: preedit lost: picker=%+v texts=%v", action, a.openDocuments, tester.Texts())
				}
				if !first.Doc.Buffer.EqualSnapshot(firstBefore) || !second.Doc.Buffer.EqualSnapshot(secondBefore) || first.Doc.Buffer.Version() != firstVersion || second.Doc.Buffer.Version() != secondVersion || first.Doc.Buffer.History() != firstHistory || second.Doc.Buffer.History() != secondHistory || first.Doc.Buffer.Selection() != firstSelection || second.Doc.Buffer.Selection() != secondSelection {
					t.Fatalf("%s: query key changed a document", action)
				}
				caret, active := tester.TextCaret()
				if !nativeComposing || !active || caret != preeditCaret {
					t.Fatalf("%s: native preedit/caret changed: composing=%v caret=%+v want=%+v", action, nativeComposing, caret, preeditCaret)
				}
				if tester.Clipboard() != "clipboard untouched" {
					t.Fatalf("%s: preedit edit command changed clipboard", action)
				}
			}
			check("compose")
			pickerClick(t, tester, pickerRowName(second))
			check("row click")
			pickerClick(t, tester, "关闭已打开文档")
			check("cancel click")
			tester.ClickAt(6, 6)
			check("backdrop click")
			for _, key := range []struct {
				name string
				mods ui.Modifiers
				key  ui.Key
			}{
				{"left", 0, ui.KeyLeft}, {"right", 0, ui.KeyRight},
				{"shift-left", ui.Shift, ui.KeyLeft}, {"shift-right", ui.Shift, ui.KeyRight},
				{"ctrl-left", ui.Ctrl, ui.KeyLeft}, {"ctrl-right", ui.Ctrl, ui.KeyRight},
				{"ctrl-shift-left", ui.Ctrl | ui.Shift, ui.KeyLeft}, {"ctrl-shift-right", ui.Ctrl | ui.Shift, ui.KeyRight},
				{"home", 0, ui.KeyHome}, {"end", 0, ui.KeyEnd},
				{"shift-home", ui.Shift, ui.KeyHome}, {"shift-end", ui.Shift, ui.KeyEnd},
				{"up", 0, ui.KeyUp}, {"down", 0, ui.KeyDown},
				{"page-up", 0, ui.KeyPageUp}, {"page-down", 0, ui.KeyPageDown},
				{"backspace", 0, ui.KeyBackspace}, {"delete", 0, ui.KeyDelete},
				{"ctrl-backspace", ui.Ctrl, ui.KeyBackspace}, {"ctrl-delete", ui.Ctrl, ui.KeyDelete},
				{"select-all", ui.Cmd, ui.KeyA}, {"copy", ui.Cmd, ui.KeyC}, {"cut", ui.Cmd, ui.KeyX}, {"paste", ui.Cmd, ui.KeyV},
				{"undo", ui.Cmd, ui.KeyZ}, {"redo", ui.Cmd, ui.KeyY}, {"shift-redo", ui.Cmd | ui.Shift, ui.KeyZ},
				{"enter", 0, ui.KeyEnter}, {"shift-enter", ui.Shift, ui.KeyEnter},
				{"escape", 0, ui.KeyEscape}, {"tab", 0, ui.KeyTab}, {"shift-tab", ui.Shift, ui.KeyTab},
			} {
				tester.Key(key.mods, key.key)
				check(key.name)
			}
			if finish == "commit" {
				tester.Type("BETA")
				if a.openDocuments.composing || nativeComposing || a.openDocuments.query != "BETA" {
					t.Fatal("commit left stale composing")
				}
				tester.Key(0, ui.KeyEnter)
				if a.Current() != second || a.openDocuments.open || !tester.Focused("文档编辑区") {
					t.Fatal("commit retry did not select")
				}
			} else {
				tester.Compose("", 0)
				if a.openDocuments.composing || nativeComposing || a.openDocuments.query != "" {
					t.Fatal("cancel left stale composing")
				}
				tester.Key(0, ui.KeyEscape)
				if a.openDocuments.open || a.Current() != first || !tester.Focused("文档编辑区") {
					t.Fatal("cancel retry did not restore opener")
				}
			}
		})
	}
}

func TestOpenDocumentsQueryOrdinaryCaretAndEditKeys(t *testing.T) {
	a, first, second, tester := openDocumentsFixture(t)
	firstBefore, secondBefore := first.Doc.Buffer.Snapshot(), second.Doc.Buffer.Snapshot()
	firstVersion, secondVersion := first.Doc.Buffer.Version(), second.Doc.Buffer.Version()
	firstHistory, secondHistory := first.Doc.Buffer.History(), second.Doc.Buffer.History()
	pickerOpen(t, a, tester)
	check := func(want string) {
		t.Helper()
		if !a.openDocuments.open || a.openDocuments.composing || !tester.Focused(openDocumentsQueryLabel) || a.openDocuments.query != want || a.openDocuments.filteredQuery != want || a.Current() != first {
			t.Fatalf("ordinary edit query=%q want=%q composing=%v", a.openDocuments.query, want, a.openDocuments.composing)
		}
		if !first.Doc.Buffer.EqualSnapshot(firstBefore) || !second.Doc.Buffer.EqualSnapshot(secondBefore) || first.Doc.Buffer.Version() != firstVersion || second.Doc.Buffer.Version() != secondVersion || first.Doc.Buffer.History() != firstHistory || second.Doc.Buffer.History() != secondHistory {
			t.Fatal("ordinary query editing changed a document")
		}
	}
	tester.Type("abcd")
	check("abcd")
	tester.Key(0, ui.KeyLeft)
	tester.Type("X")
	check("abcXd")
	tester.Key(0, ui.KeyRight)
	tester.Type("Y")
	check("abcXdY")
	tester.Key(ui.Shift, ui.KeyLeft)
	tester.Type("Z")
	check("abcXdZ")
	tester.Key(0, ui.KeyHome)
	tester.Key(0, ui.KeyDelete)
	check("bcXdZ")
	tester.Key(0, ui.KeyEnd)
	tester.Key(0, ui.KeyBackspace)
	check("bcXd")
	tester.Key(ui.Cmd, ui.KeyA)
	tester.Type("BETA")
	check("BETA")
	tester.Key(ui.Cmd, ui.KeyZ)
	check("bcXd")
	tester.Key(ui.Cmd, ui.KeyY)
	check("BETA")
	tester.Key(ui.Cmd, ui.KeyA)
	tester.Key(ui.Cmd, ui.KeyC)
	if tester.Clipboard() != "BETA" {
		t.Fatal("ordinary copy did not use selected query")
	}
	tester.Key(ui.Cmd, ui.KeyX)
	check("")
	tester.Key(ui.Cmd, ui.KeyV)
	check("BETA")
	tester.Key(0, ui.KeyEnter)
	if a.openDocuments.open || a.Current() != second || !tester.Focused("文档编辑区") {
		t.Fatal("normal edited query did not activate destination")
	}
}

func TestOpenDocumentsSinglePointerSelect(t *testing.T) {
	a, _, second, tester := openDocumentsFixture(t)
	pickerOpen(t, a, tester)
	pickerClick(t, tester, pickerRowName(second))
	if a.openDocuments.open || a.Current() != second || !tester.Focused("文档编辑区") {
		t.Fatal("single click did not switch with destination focus")
	}
}

func TestOpenDocumentsMetadataUnicodeBoundsAndReuse(t *testing.T) {
	a, first, second, tester := openDocumentsFixture(t)
	first.Name, first.Doc.Path = "Résumé🙂.TXT", `C:\笔记\Résumé🙂.TXT`
	second.Name, second.Doc.Path, second.Doc.ReadOnly = first.Name, `D:\资料\Résumé🙂.TXT`, true
	third := a.appendTab(document.New("bytes"), "图像.PNG", "", true, false)
	third.BinaryPath, third.Doc.ReadOnly = `D:\资料\图像.PNG`, true
	fourth := a.appendTab(document.New("draft"), "未命名.txt", "", false, false)
	long := a.appendTab(document.New("long"), strings.Repeat("猫\n", 400), "", false, false)
	long.Doc.Path = "C:\\" + strings.Repeat("界🙂\t", 1000) + "needle-beyond-bound"
	pickerOpen(t, a, tester)
	p := &a.openDocuments
	if len(p.rows) != 5 || p.rows[0].id != first.ID || !p.rows[0].dirty || !p.rows[1].readOnly || p.rows[1].dirty || !p.rows[2].binary || p.rows[3].path != "" {
		t.Fatal("metadata/badges incorrect", p.rows)
	}
	if !tester.HasText("#1 · 当前 · 未保存") || !tester.HasText("#2 · 只读") || !tester.HasText("二进制") || !tester.HasText("尚未保存到文件") || !tester.HasText("仅搜索显示的部分") {
		t.Fatal("row distinction/badges/clipping missing", tester.Texts())
	}
	for _, row := range p.rows {
		if len(row.title) > 256 || len(row.path) > 4096 || len(row.foldedTitle) > 256 || len(row.foldedPath) > 4096 || !utf8.ValidString(row.title) || !utf8.ValidString(row.path) || strings.ContainsAny(row.title+row.path, "\n\t") {
			t.Fatal("unbounded/unsanitized metadata")
		}
	}
	ids := func() []int {
		var ids []int
		for _, row := range p.matches {
			ids = append(ids, row.id)
		}
		return ids
	}
	for _, tc := range []struct {
		query string
		want  []int
	}{{"rÉSUMÉ🙂", []int{first.ID, second.ID}}, {"d:\\资料", []int{second.ID, third.ID}}, {"PNG", []int{third.ID}}, {"未命名", []int{fourth.ID}}, {"needle-beyond-bound", nil}} {
		a.refreshOpenDocuments(tc.query)
		if !reflect.DeepEqual(ids(), tc.want) {
			t.Fatalf("substring %q: %v want%v", tc.query, ids(), tc.want)
		}
	}
	a.refreshOpenDocuments("")
	before := append([]openDocumentRow(nil), p.rows...)
	for range 5 {
		tester.Frame()
	}
	if allocs := testing.AllocsPerRun(25, func() { a.refreshOpenDocuments("") }); allocs != 0 {
		t.Fatalf("unchanged metadata allocates: %v", allocs)
	}
	for _, q := range []string{"résumé", "🙂", "none", ""} {
		a.refreshOpenDocuments(q)
	}
	for i, row := range p.rows {
		if unsafe.StringData(row.title) != unsafe.StringData(before[i].title) || unsafe.StringData(row.path) != unsafe.StringData(before[i].path) || unsafe.StringData(row.foldedTitle) != unsafe.StringData(before[i].foldedTitle) || unsafe.StringData(row.foldedPath) != unsafe.StringData(before[i].foldedPath) {
			t.Fatal("query-only work recloned metadata")
		}
	}
	first.Name = "changed.txt"
	a.refreshOpenDocuments("changed")
	if !reflect.DeepEqual(ids(), []int{first.ID}) {
		t.Fatal("rename cache stale")
	}
	a.Shutdown()
	if p.open || p.rows != nil || p.matches != nil || p.list.Key != nil || p.list.Label != nil || p.query != "" {
		t.Fatal("shutdown retained picker")
	}
}

func TestOpenDocumentsHundredVirtualRowsAndQueryNavigation(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	seedAdmissionTabs(a, session.MaxTabs)
	for i, tab := range a.Tabs {
		if i%5 == 1 {
			tab.Doc.ReadOnly = true
			tab.Editor.ReadOnly = true
		}
		if i%5 == 2 {
			tab.BinaryPath = fmt.Sprintf("C:\\images\\%d.png", i)
			tab.Doc.ReadOnly = true
			tab.Editor.ReadOnly = true
		}
	}
	first, last := a.Tabs[0], a.Tabs[len(a.Tabs)-1]
	tester := ui.NewTester(a.View, 760, 480)
	pickerClick(t, tester, "文档编辑区")
	pickerOpen(t, a, tester)
	start, end := a.openDocuments.list.Visible()
	if start != 0 || end > 10 || end < 1 {
		t.Fatalf("list not virtual: %d..%d", start, end)
	}
	for i := 0; i < session.MaxTabs-1; i++ {
		tester.Key(0, ui.KeyDown)
	}
	tester.Frame()
	start, end = a.openDocuments.list.Visible()
	lastRow, ok := tester.AccessByRole(ui.RoleButton, pickerRowName(last))
	if a.openDocuments.selectedID != last.ID || end != 99 || start < 85 || !ok || lastRow.Offscreen || lastRow.Bounds.Y < 0 || lastRow.Bounds.Y+lastRow.Bounds.H > 480 || !tester.Focused(openDocumentsQueryLabel) || a.Current() != first {
		t.Fatalf("final row not revealed without activating: %d..%d selected=%d row=%+v", start, end, a.openDocuments.selectedID, lastRow)
	}
	for _, tab := range a.Tabs {
		if tab.View != nil {
			t.Fatal("picker mounted an inactive view")
		}
	}
	tester.Type("no match")
	if len(a.openDocuments.matches) != 0 || a.openDocuments.selected != -1 || !tester.HasText("没有匹配") {
		t.Fatal("no-match state incorrect")
	}
	tester.Key(0, ui.KeyEnter)
	if !a.openDocuments.open || a.Current() != first {
		t.Fatal("empty Enter activated")
	}
	tester.Command("selectAll")
	tester.Type("tab-09")
	if len(a.openDocuments.matches) != 10 || a.openDocuments.selectedID != a.Tabs[90].ID {
		t.Fatal("query shrink selection incorrect")
	}
	tester.Key(0, ui.KeyDown)
	chosen := a.openDocuments.selectedID
	tester.Command("selectAll")
	tester.Type("tab-")
	if a.openDocuments.selectedID != chosen {
		t.Fatal("query expansion did not retain selected ID")
	}
	tester.Command("selectAll")
	tester.Type(strings.Repeat("🙂", 100))
	if text.UTF16Len(a.openDocuments.query) != 128 {
		t.Fatalf("query limit=%d", text.UTF16Len(a.openDocuments.query))
	}
	tester.Key(0, ui.KeyEscape)
	if a.openDocuments.open || len(a.Tabs) != session.MaxTabs || a.Current() != first || !tester.Focused("文档编辑区") {
		t.Fatal("navigation changed tabs/focus")
	}
}

func TestOpenDocumentsSourceAndUtilityCompositionBarriers(t *testing.T) {
	for _, kind := range []string{"source", "refused-first", "refused-ongoing", "find", "replacement", "goto"} {
		t.Run(kind, func(t *testing.T) {
			a, first, _, tester := openDocumentsFixture(t)
			label := "文档编辑区"
			switch kind {
			case "source":
				tester.Compose("未提交🙂", 4)
			case "refused-first", "refused-ongoing":
				b := text.NewWithLimit("a", 4)
				first.Doc.Buffer, first.Editor.Buffer = b, b
				first.Editor.Reveal(1, 1)
				tester.Frame()
				if kind == "refused-ongoing" {
					tester.Compose("ni", 2)
				}
				tester.Compose("🙂", 1)
			case "find", "replacement":
				tester.Key(ui.Cmd, ui.KeyF)
				label = "查找内容"
				if kind == "replacement" {
					label = "替换内容"
					pickerClick(t, tester, label)
				}
				tester.Compose("needle", 6)
			case "goto":
				tester.Key(ui.Cmd, ui.KeyG)
				label = "转到行号"
				tester.Compose("12", 2)
			}
			snapshot, history, selection := first.Doc.Buffer.Snapshot(), first.Doc.Buffer.History(), first.Doc.Buffer.Selection()
			mark, marked := first.Editor.MarkedRange()
			for _, button := range []bool{false, true} {
				if button {
					pickerClick(t, tester, "已打开文档")
				} else {
					tester.Key(ui.Cmd, ui.KeyP)
				}
				next, active := first.Editor.MarkedRange()
				if a.openDocuments.open || !tester.Focused(label) || a.Current() != first || !first.Doc.Buffer.EqualSnapshot(snapshot) || first.Doc.Buffer.History() != history || first.Doc.Buffer.Selection() != selection || next != mark || active != marked {
					t.Fatalf("%s composition admission moved input: %s", kind, a.Message)
				}
			}
			if strings.HasPrefix(kind, "refused") {
				if !errors.Is(first.Editor.Flush(), text.ErrSizeLimit) {
					t.Fatal("refused input bypassed")
				}
				tester.Key(0, ui.KeyEscape)
			} else {
				tester.Type("x")
			}
			pickerOpen(t, a, tester)
			tester.Key(0, ui.KeyEscape)
			if !tester.Focused(label) {
				t.Fatalf("utility/source opener not restored: %s", label)
			}
		})
	}
}

func TestOpenDocumentsAdmissionAndReentrantIdentity(t *testing.T) {
	for _, when := range []string{"open", "select", "same"} {
		for _, change := range []string{"fail", "remove", "reorder", "activate"} {
			t.Run(when+"/"+change, func(t *testing.T) {
				a, first, second, tester := openDocumentsFixture(t)
				third := a.appendTab(document.New("third"), "third.txt", "", false, false)
				v := &navigationFlushView{}
				first.View = v
				if when != "open" {
					pickerOpen(t, a, tester)
				}
				v.flush = func() error {
					switch change {
					case "fail":
						return errors.New("kept draft")
					case "remove":
						a.Tabs = a.Tabs[:1]
					case "reorder":
						a.Tabs[1], a.Tabs[2] = a.Tabs[2], a.Tabs[1]
					case "activate":
						a.Active = 2
					}
					return nil
				}
				if when == "open" {
					tester.Key(ui.Cmd, ui.KeyP)
					// A changed non-current tab does not invalidate opening; current
					// origin reentry/failure does. Activation will resolve fresh IDs.
					if change == "activate" || change == "fail" {
						if a.openDocuments.open {
							t.Fatal("opening crossed failed/reentrant origin")
						}
					} else {
						if !a.openDocuments.open {
							t.Fatal("valid origin could not open fresh list")
						}
					}
				} else {
					id := second.ID
					if when == "same" {
						id = first.ID
					}
					ok := a.activateOpenDocument(a.openDocuments.generation, id)
					want := when == "same" && (change == "remove" || change == "reorder")
					if ok != want {
						t.Fatalf("activation=%v want%v: %s", ok, want, a.Message)
					}
					tester.Frame()
					if a.openDocuments.open {
						t.Fatal("closed action retained dialog")
					}
				}
				if v.releases != 0 || first.Doc.Buffer.String() != "first untouched" || second.Doc.Buffer.String() != "second untouched" || third.Doc.Buffer.String() != "third" {
					t.Fatal("failed/no-op picker released or edited a document")
				}
				if change == "activate" {
					if a.Current() != third {
						t.Fatal("reentry selected wrong target")
					}
				} else if a.Current() != first {
					t.Fatal("refusal/no-op changed origin")
				}
			})
		}
	}
	for _, state := range []string{"restore", "modal", "shutdown"} {
		t.Run(state, func(t *testing.T) {
			a, _, _, _ := openDocumentsFixture(t)
			switch state {
			case "restore":
				a.restoring = true
			case "modal":
				a.modalDepth = 1
			case "shutdown":
				a.Shutdown()
			}
			if a.openDocumentPicker() {
				t.Fatal("unavailable app admitted picker")
			}
		})
	}
}

func TestOpenDocumentsAsyncSelectionInvalidatesRows(t *testing.T) {
	a, first, second, tester := openDocumentsFixture(t)
	pickerOpen(t, a, tester)
	generation := a.openDocuments.generation
	if !a.Select(1) {
		t.Fatal(a.Message)
	}
	tester.Frame()
	if a.openDocuments.open || a.Current() != second || !tester.Focused("文档编辑区") || a.activateOpenDocument(generation, first.ID) || a.Current() != second {
		t.Fatal("stale dialog restored/switched old tab")
	}
	tester.Type("!")
	if strings.Contains(first.Doc.Buffer.String(), "!") || !strings.Contains(second.Doc.Buffer.String(), "!") {
		t.Fatal("async invalidation focused wrong editor")
	}
}

func TestOpenDocumentsSameTargetMutationDuringFlush(t *testing.T) {
	for _, change := range []string{"remove", "replace", "move"} {
		t.Run(change, func(t *testing.T) {
			a, first, second, tester := openDocumentsFixture(t)
			third := a.appendTab(document.New("third"), "third.txt", "", false, false)
			v := &navigationFlushView{}
			first.View = v
			secondView := &failingView{}
			second.View = secondView
			pickerOpen(t, a, tester)
			original := []*Tab{first, second, third}
			versions := []uint64{first.Doc.Buffer.Version(), second.Doc.Buffer.Version(), third.Doc.Buffer.Version()}
			histories := []text.HistoryStatus{first.Doc.Buffer.History(), second.Doc.Buffer.History(), third.Doc.Buffer.History()}
			v.flush = func() error {
				switch change {
				case "remove":
					a.Tabs = []*Tab{second, third}
					a.Active = 0
				case "replace":
					a.Tabs = []*Tab{third, second}
					a.Active = 0
				case "move":
					a.Tabs[0], a.Tabs[1] = a.Tabs[1], a.Tabs[0]
					a.Active = 1
				}
				return nil
			}
			// Use the real same-row click; the captured slot is no longer owned
			// by that ID when the synchronous Flush callback returns.
			pickerClick(t, tester, pickerRowName(first))
			if a.openDocuments.open || !strings.Contains(a.Message, "文档列表已变化") || v.releases != 0 || secondView.releases != 0 {
				t.Fatal("same-target mutation was not visibly refused", a.Message)
			}
			for i, tab := range original {
				if tab.Doc.Buffer.Version() != versions[i] || tab.Doc.Buffer.History() != histories[i] {
					t.Fatal("refusal changed text/history")
				}
			}
			want := second
			if change == "replace" {
				want = third
			}
			if change == "move" {
				want = first
			}
			if a.Current() != want {
				t.Fatalf("refusal selected a different slot: %v", a.Current())
			}
		})
	}
}

func TestOpenDocumentsRepeatedOpenReusesPublicHandles(t *testing.T) {
	a, first, second, tester := openDocumentsFixture(t)
	pickerOpen(t, a, tester)
	queryHandle, listHandle := a.openDocuments.queryHandle, a.openDocuments.list.Handle
	if queryHandle == (ui.Handle{}) || listHandle == (ui.Handle{}) {
		t.Fatal("fixture handles not bound")
	}
	generation := a.openDocuments.generation
	for i := 0; i < 12; i++ {
		tester.Command("selectAll")
		tester.Type("BETA")
		if a.openDocuments.queryHandle != queryHandle || a.openDocuments.list.Handle != listHandle {
			t.Fatal("filter replaced control identity")
		}
		tester.Key(0, ui.KeyEscape)
		p := &a.openDocuments
		if p.open || p.rows != nil || p.matches != nil || p.query != "" || p.filteredQuery != "" || p.message != "" || p.pendingID != 0 || p.pendingClose || p.originID != 0 || p.selectedID != 0 || p.composing || p.list.Key != nil || p.list.Label != nil || p.list.Selected != nil || p.queryHandle != queryHandle || p.list.Handle != listHandle || p.generation <= generation || !tester.Focused("文档编辑区") {
			t.Fatal("close did not clear data/restore focus/reuse identities")
		}
		generation = p.generation
		if i%2 == 0 {
			pickerClick(t, tester, "已打开文档")
		} else {
			pickerOpen(t, a, tester)
		}
		if p.queryHandle != queryHandle || p.list.Handle != listHandle || !tester.Focused(openDocumentsQueryLabel) {
			t.Fatal("reopening replaced handles or missed query focus")
		}
	}
	pickerClick(t, tester, pickerRowName(second))
	if a.Current() != second || !tester.Focused("文档编辑区") || first.Doc.Buffer.String() != "first untouched" {
		t.Fatal("repeated open changed owner/focus")
	}
	a.Shutdown()
	if a.openDocuments.rows != nil || a.openDocuments.list.Key != nil || a.openDocuments.queryHandle != queryHandle || a.openDocuments.list.Handle != listHandle {
		t.Fatal("shutdown retained data or allocated identities")
	}
}

func TestOpenDocumentsRealMarkdownDraftBarriersAndFocus(t *testing.T) {
	for _, ending := range []string{"escape", "same", "failed-other", "failed-same"} {
		t.Run(ending, func(t *testing.T) {
			a := newTestApp(t, t.TempDir())
			origin := a.add(document.New("# 标题\n\n正文\n"), "note.md", "")
			other := a.appendTab(document.New("other"), "other.txt", "", false, false)
			tester := ui.NewTester(a.View, 760, 480)
			beginRealMarkdownComposition(t, origin, tester)
			view := origin.View
			before, version := origin.Doc.Buffer.String(), origin.Doc.Buffer.Version()
			tester.Key(ui.Cmd, ui.KeyP)
			if a.openDocuments.open || !strings.Contains(a.Message, "输入法正在组合") || !tester.Focused("文档编辑区") || origin.View != view || origin.Doc.Buffer.String() != before {
				t.Fatal("rich preedit opening barrier failed")
			}
			tester.Type("changed🙂")
			if origin.Doc.Buffer.Version() == version || !strings.Contains(origin.Doc.Buffer.String(), "changed🙂") {
				t.Fatal("real draft did not commit")
			}
			pickerOpen(t, a, tester)
			if strings.HasPrefix(ending, "failed") {
				// Opening has already flushed the real field. Inject a later
				// fallible Flusher result while retaining its actual drawing/input
				// identity; the next test covers genuine rich growth refusal.
				origin.View = &pickerRefusingView{View: view}
			}
			snapshot, history := origin.Doc.Buffer.Snapshot(), origin.Doc.Buffer.History()
			switch ending {
			case "escape":
				tester.Key(0, ui.KeyEscape)
			case "same", "failed-same":
				pickerClick(t, tester, pickerRowName(origin))
			case "failed-other":
				pickerClick(t, tester, pickerRowName(other))
			}
			tester.Frame()
			if a.openDocuments.open || a.Current() != origin || origin.Mode != "preview" || !tester.Focused("文档编辑区") || !origin.Doc.Buffer.EqualSnapshot(snapshot) || origin.Doc.Buffer.History() != history {
				t.Fatalf("rich opener/draft not preserved: ending=%s message=%s texts=%v", ending, a.Message, tester.Texts())
			}
			if strings.HasPrefix(ending, "failed") && !strings.Contains(a.Message, "未能同步格式编辑") {
				t.Fatalf("real flush failure not reported: %s", a.Message)
			}
			// Text targets the visible rich draft, never a hidden source buffer.
			tester.Type("!")
			if origin.Doc.Buffer.String() != "# changed🙂!\n\n正文\n" {
				t.Fatalf("did not restore exact rich editor: %q", origin.Doc.Buffer.String())
			}
		})
	}
}

type pickerRefusingView struct{ editor.View }

func (v *pickerRefusingView) Flush() error {
	return errors.New("later format flush refused; draft retained")
}

func TestOpenDocumentsRealRichGrowthRefusalKeepsEditableDraft(t *testing.T) {
	const source = "# 标题\n\n正文\n"
	a := newTestApp(t, t.TempDir())
	doc := document.New(source)
	doc.Buffer = text.NewWithLimit(source, len(source))
	origin := a.add(doc, "limited.md", "")
	a.appendTab(document.New("other"), "other.txt", "", false, false)
	tester := ui.NewTester(a.View, 760, 480)
	beginRealMarkdownComposition(t, origin, tester)
	tester.Type("中文🙂") // real private draft exceeds the host's byte ceiling
	view, history, selection := origin.View, doc.Buffer.History(), doc.Buffer.Selection()
	if err := editor.FlushView(view); !errors.Is(err, text.ErrSizeLimit) {
		t.Fatalf("fixture has no real growth refusal: %v", err)
	}
	for _, entry := range []string{"key", "button"} {
		if entry == "key" {
			tester.Key(ui.Cmd, ui.KeyP)
		} else {
			pickerClick(t, tester, "已打开文档")
		}
		if a.openDocuments.open || a.Current() != origin || origin.View != view || !tester.Focused("文档编辑区") || doc.Buffer.String() != source || doc.Buffer.History() != history || doc.Buffer.Selection() != selection || !strings.Contains(a.Message, "内容上限") {
			t.Fatal("real rejected draft lost focus/content", a.Message)
		}
	}
	tester.Command("selectAll")
	tester.Type("短")
	if doc.Buffer.String() != "# 短\n\n正文\n" {
		t.Fatalf("could not correct exact rich draft: %q", doc.Buffer.String())
	}
	pickerOpen(t, a, tester)
	tester.Key(0, ui.KeyEscape)
	tester.Type("字")
	if doc.Buffer.String() != "# 短字\n\n正文\n" {
		t.Fatal("valid retry did not restore rich field")
	}
}

func TestOpenDocumentsMinimumWindowCapture(t *testing.T) {
	dir := os.Getenv("LEVNOTE_OPEN_DOCUMENTS_CAPTURE_DIR")
	if dir == "" {
		t.Skip("set LEVNOTE_OPEN_DOCUMENTS_CAPTURE_DIR for one picker capture")
	}
	a := newTestApp(t, t.TempDir())
	for i, name := range []string{"今天的计划.md", "项目说明.md", "项目说明.md", "界面草图.png", "待办事项.txt", "会议记录.md", "数据导出.bin", "未命名.txt"} {
		tab := a.appendTab(document.New("本地草稿"), name, "", false, false)
		tab.Mode = "source"
		if i < 7 {
			tab.Doc.Path = fmt.Sprintf("C:\\笔记\\项目%d\\%s", i, name)
		}
		if i == 3 || i == 6 {
			tab.BinaryPath = tab.Doc.Path
			tab.Doc.ReadOnly = true
			tab.Editor.ReadOnly = true
		}
	}
	a.Active = 0
	tester := ui.NewTester(a.View, 760, 480)
	pickerClick(t, tester, "文档编辑区")
	pickerOpen(t, a, tester)
	dialog, ok := tester.AccessByRole(ui.RoleDialog, "已打开文档")
	if !ok || dialog.Bounds.X < 0 || dialog.Bounds.Y < 0 || dialog.Bounds.X+dialog.Bounds.W > 760 || dialog.Bounds.Y+dialog.Bounds.H > 480 {
		t.Fatalf("dialog does not fit: %+v", dialog)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "open-documents-760x480.png")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("refusing to overwrite capture", err)
	}
	captureAcceptanceTester(t, path, tester)
}
