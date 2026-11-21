package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/mindmap"
	"levnote.local/native/ui/editor"
)

// These layout regressions always run; capture environment variables are not
// prerequisites for checking the real minimum-window geometry.
func requireRecoveryLayoutRect(t *testing.T, label string, rect ui.Rect, bottom float32) {
	t.Helper()
	if rect.W <= 0 || rect.H <= 0 || rect.X < 0 || rect.Y < 0 ||
		rect.X+rect.W > 760 || rect.Y+rect.H > bottom {
		t.Fatalf("%s is clipped at 760x480: %#v (bottom %v)", label, rect, bottom)
	}
}

func TestLayoutRecoveryStackedTableBodyAboveAddParagraph(t *testing.T) {
	for _, dark := range []bool{false, true} {
		name := map[bool]string{false: "light", true: "dark"}[dark]
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "table-view.md")
			if err := os.WriteFile(path, []byte(markdownTablesScreenshotBody), 0600); err != nil {
				t.Fatal(err)
			}
			a := newTestApp(t, filepath.Join(root, "config"))
			a.Root = root
			a.refreshTreeNow()
			a.Open(path)
			tab := a.Current()
			if tab == nil || !a.setMode(tab, "split") {
				t.Fatal("could not open Markdown split fixture")
			}
			a.Dark = dark
			buffer := tab.Doc.Buffer
			version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
			tester := ui.NewTester(a.View, 760, 480)
			tester.SetDark(dark)
			tester.Frame()
			tester.Frame()
			tester.Frame()
			if !tester.HasText("表头 · 第 1 行") || !tester.HasText("数据 · 第 2 行") {
				t.Fatal("narrow table lost its labeled stacked records")
			}
			add, ok := tester.Find("添加段落")
			if !ok {
				t.Fatal("Add Paragraph action disappeared")
			}
			requireRecoveryLayoutRect(t, "添加段落", add, 451)
			for _, name := range []string{markdownCellBodyName, "数据，第 2 行，第 2 列：**字面**"} {
				cell := requireMarkdownCellAccess(t, tester, ui.RoleCell, name)
				requireRecoveryLayoutRect(t, name, cell.Bounds, add.Y)
			}
			literal, ok := tester.Find("**字面**")
			if !ok {
				t.Fatal("literal body text disappeared")
			}
			requireRecoveryLayoutRect(t, "literal body text", literal, add.Y)
			if buffer.String() != markdownTablesScreenshotBody || buffer.Version() != version ||
				buffer.Selection() != selection || buffer.History() != history || tab.Doc.Dirty() {
				t.Fatal("layout settling changed source, selection, history or dirty state")
			}
		})
	}
}

func TestLayoutRecoveryMindmapConflictKeepsActiveNoteAndFullWarnings(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "fields.mindmap")
	if err := os.WriteFile(path, []byte(mindmapFieldsFixture), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	clickFieldControl(t, tester, "字段：纯文本备注")
	if err := tester.Click("节点备注（纯文本）"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	draft := "窄屏保留草稿🙂\r\n第二行"
	tester.Compose(draft, utf8.RuneCountInString(draft))
	tester.Frame()
	raw := tab.Doc.Buffer.String()
	external := strings.Replace(raw, `"note": "原备注"`, `"note": "外部改动"`, 1)
	if external == raw {
		t.Fatal("note conflict fixture did not change")
	}
	if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), external); err != nil {
		t.Fatal(err)
	}
	if a.flushTab(tab, "布局回归") {
		t.Fatal("conflicting note draft unexpectedly flushed")
	}
	tester.Frame()
	for _, label := range []string{"字段：纯文本备注", "节点备注（纯文本）"} {
		rect, _ := fieldControl(t, tester, label)
		requireRecoveryLayoutRect(t, label, rect, 451)
	}
	button, ok := tester.AccessByRole(ui.RoleButton, "查看图形提示详情")
	if !ok || !button.Focusable || button.Offscreen {
		t.Fatalf("warning details are not keyboard accessible: %#v", button)
	}
	requireRecoveryLayoutRect(t, button.Name, button.Bounds, 451)
	for _, warning := range tab.View.Warnings() {
		if !strings.Contains(button.Description, warning) {
			t.Fatalf("inline warning details omitted full text: %q", warning)
		}
	}
	version, selection, history := tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection(), tab.Doc.Buffer.History()
	if err := tester.Click(button.Name); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if !tester.HasText("图形视图提示详情") {
		t.Fatal("warning details did not open")
	}
	for _, warning := range tab.View.Warnings() {
		if !tester.HasText(warning) {
			t.Fatalf("warning dialog omitted complete text: %q", warning)
		}
	}
	if tab.Doc.Buffer.String() != external || tab.Doc.Buffer.Version() != version ||
		tab.Doc.Buffer.Selection() != selection || tab.Doc.Buffer.History() != history {
		t.Fatal("warning presentation committed or replaced the conflicted draft")
	}
}

func TestLayoutRecoverySingleHostWarningHasAccessibleFullDetails(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.NewDocument("markdown")
	tab := a.Current()
	tab.Mode = "preview"
	warning := strings.Repeat("完整警告不能只放在悬停提示中。", 12)
	tab.View = &failingView{warnings: []string{warning}}
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	button, ok := tester.AccessByRole(ui.RoleButton, "查看详情")
	if !ok || !button.Focusable || button.Offscreen || button.Description != warning {
		t.Fatalf("single warning lost accessible full details: %#v", button)
	}
	requireRecoveryLayoutRect(t, "查看详情", button.Bounds, 451)
	for attempt := 0; attempt < 120 && !tester.Focused(button.Name); attempt++ {
		tester.Key(0, ui.KeyTab)
		tester.Frame()
	}
	if !tester.Focused(button.Name) {
		t.Fatal("warning Details could not receive keyboard focus")
	}
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	if !a.ShowIssues || !tester.HasText(warning) || len(tab.ViewWarnings) != 1 {
		t.Fatal("single full warning was not available in the issues panel")
	}
}

func TestLayoutRecoveryHealthyGraphInputKeepsNativeOwnerAcrossDetails(t *testing.T) {
	for _, kind := range []string{"note", "field-search"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "healthy-warning.mindmap")
			if err := os.WriteFile(path, []byte(mindmapFieldsFixture), 0600); err != nil {
				t.Fatal(err)
			}
			a := newTestApp(t, filepath.Join(root, "config"))
			a.Open(path)
			tab := a.Current()
			if tab == nil || tab.Editor == nil || tab.View != nil {
				t.Fatal("fixture needs the real source callback before the lazy adapter is mounted")
			}
			changed := 0
			onChange := tab.Editor.OnChange
			tab.Editor.OnChange = func() {
				changed++
				if onChange != nil {
					onChange()
				}
			}
			// App creates the rich adapter lazily while drawing the first frame.
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			if tab == nil || a.Current() != tab || tab.View == nil {
				t.Fatal("real graph adapter did not open")
			}
			view, buffer := tab.View, tab.Doc.Buffer
			composer, ok := view.(editor.Composer)
			if !ok {
				t.Fatal("real graph adapter does not expose composition state")
			}
			label := "节点备注（纯文本）"
			if kind == "note" {
				clickFieldControl(t, tester, "字段：纯文本备注")
			} else {
				if err := tester.Click("打开或收起字段查找替换"); err != nil {
					t.Fatal(err)
				}
				label = "字段查找文字"
			}
			if err := tester.Click(label); err != nil {
				t.Fatal(err)
			}
			tester.Command("selectAll")
			preedit := "健康输入尚未确认🙂"
			tester.Compose(preedit, utf8.RuneCountInString(preedit))
			tester.Frame()
			// Editor text is custom-drawn and is not a Tester label node.
			// Exact preedit bytes/mark/history are checked by the graph package;
			// App checks the real accessible input and native attachment here.
			inputAccess, ok := tester.AccessByRole(ui.RoleTextField, label)
			if !ok || inputAccess.Disabled || !inputAccess.Focusable || inputAccess.Offscreen ||
				inputAccess.Bounds.W <= 0 || inputAccess.Bounds.H <= 0 || !composer.CompositionActive() || !tester.Focused(label) {
				t.Fatalf("fixture did not establish healthy focused graph input: %#v", inputAccess)
			}
			var native ui.TesterTextInputCapture
			if err := tester.TextInputNoFrame(func(n *ui.TesterTextInput) { native = n.Capture() }); err != nil {
				t.Fatal(err)
			}
			warnings := append([]string(nil), view.Warnings()...)
			if len(warnings) < 2 {
				t.Fatalf("fixture lost complete compatibility warnings: %#v", warnings)
			}
			version, stamp, selection, history := buffer.Version(), buffer.ChangeStamp(), buffer.Selection(), buffer.History()
			dirty, grouped := tab.Doc.Dirty(), buffer.InGroup()
			button, ok := tester.AccessByRole(ui.RoleButton, "查看图形提示详情")
			if !ok || button.Disabled || !button.Focusable || button.Offscreen {
				t.Fatalf("Details is not reachable during preedit: %#v", button)
			}
			requireRecoveryLayoutRect(t, button.Name, button.Bounds, 451)
			for _, warning := range warnings {
				if !strings.Contains(button.Description, warning) {
					t.Fatalf("accessible Details omitted full warning %q", warning)
				}
			}
			if err := tester.Click(button.Name); err != nil {
				t.Fatal(err)
			}
			tester.Frame()
			var after ui.TesterTextInputCapture
			if err := tester.TextInputNoFrame(func(n *ui.TesterTextInput) { after = n.Capture() }); err != nil {
				t.Fatal(err)
			}
			inputAfter, ok := tester.AccessByRole(ui.RoleTextField, label)
			if !ok || inputAfter.Disabled || !inputAfter.Focusable || inputAfter.Offscreen ||
				inputAfter.Bounds.W <= 0 || inputAfter.Bounds.H <= 0 {
				t.Fatalf("Details attempt hid the accessible graph input: %#v", inputAfter)
			}
			if after != native || !composer.CompositionActive() || !tester.Focused(label) ||
				a.Current() != tab || tab.View != view || tab.Doc.Buffer != buffer || tester.HasText("图形视图提示详情") {
				t.Fatal("Details attempt moved focus, retired the native input or changed the real App owner")
			}
			if !tester.HasText("请先完成或取消当前输入，再查看详情") {
				t.Fatal("blocked Details has no visible completion guidance")
			}
			if buffer.String() != mindmapFieldsFixture || buffer.Version() != version || buffer.ChangeStamp() != stamp ||
				buffer.Selection() != selection || buffer.History() != history || buffer.InGroup() != grouped || tab.Doc.Dirty() != dirty || changed != 0 {
				t.Fatal("Details attempt mutated real App host source, selection, stamp, history or dirty state")
			}
			// A legal native final writes the note through Host.Changed once.
			// Warning inspection is measured against that committed state.
			wantSource := mindmapFieldsFixture
			if kind == "note" {
				document := parseMindmapBytes(t, []byte(wantSource))
				if err := document.SetNote("root", preedit); err != nil {
					t.Fatal(err)
				}
				serialized, err := document.Serialize(mindmap.SerializeOptions{})
				if err != nil {
					t.Fatal(err)
				}
				wantSource = string(serialized)
				tester.Type(preedit)
			} else {
				tester.Key(0, ui.KeyEscape)
			}
			tester.Frame()
			if composer.CompositionActive() || !tester.Focused(label) {
				t.Fatal("explicit user commit/cancel did not complete the focused input")
			}
			if a.Current() != tab || tab.View != view || tab.Doc.Buffer != buffer {
				t.Fatal("explicit user completion replaced the real App owner")
			}
			completedInput, ok := tester.AccessByRole(ui.RoleTextField, label)
			if !ok || completedInput.Disabled || !completedInput.Focusable || completedInput.Offscreen ||
				completedInput.Bounds.W <= 0 || completedInput.Bounds.H <= 0 {
				t.Fatalf("explicit completion hid the real accessible input: %#v", completedInput)
			}
			if kind == "note" {
				note, present, err := parseMindmapBytes(t, []byte(buffer.String())).Field("root", mindmap.FieldNote)
				committedStamp, committedHistory := buffer.ChangeStamp(), buffer.History()
				end := text.UTF16Len(wantSource)
				if err != nil || !present || note != preedit || buffer.String() != wantSource ||
					buffer.Version() != version+1 || committedStamp.BaselineID != stamp.BaselineID ||
					committedStamp.ConfirmedChangeID == stamp.ConfirmedChangeID ||
					committedHistory.UndoDepth != history.UndoDepth+1 || committedHistory.RedoDepth != 0 ||
					committedHistory.Truncated != history.Truncated || buffer.InGroup() ||
					buffer.Selection() != (text.Range{Anchor: end, Head: end}) || !tab.Doc.Dirty() || changed != 1 {
					t.Fatalf("legal native final did not commit the exact root note once: note=%q present=%t changed=%d err=%v", note, present, changed, err)
				}
			} else if buffer.String() != mindmapFieldsFixture || buffer.Version() != version || buffer.ChangeStamp() != stamp ||
				buffer.Selection() != selection || buffer.History() != history || buffer.InGroup() != grouped || tab.Doc.Dirty() != dirty || changed != 0 {
				t.Fatal("field-search cancellation changed the document host")
			}
			version, stamp, selection, history = buffer.Version(), buffer.ChangeStamp(), buffer.Selection(), buffer.History()
			dirty, grouped = tab.Doc.Dirty(), buffer.InGroup()
			settledChanged, message, issueCount, dropped := changed, a.Message, len(a.Issues), a.IssuesDropped
			for attempt := 0; attempt < 120 && !tester.Focused(button.Name); attempt++ {
				tester.Key(0, ui.KeyTab)
				tester.Frame()
			}
			if !tester.Focused(button.Name) {
				t.Fatal("Details could not receive keyboard focus after explicit user completion")
			}
			tester.Key(0, ui.KeyEnter)
			tester.Frame()
			if !tester.HasText("图形视图提示详情") {
				t.Fatal("keyboard Details did not open after explicit user completion")
			}
			for _, warning := range warnings {
				if !tester.HasText(warning) {
					t.Fatalf("keyboard Details omitted complete warning %q", warning)
				}
			}
			if buffer.String() != wantSource || buffer.Version() != version || buffer.ChangeStamp() != stamp ||
				buffer.Selection() != selection || buffer.History() != history || buffer.InGroup() != grouped || tab.Doc.Dirty() != dirty ||
				changed != settledChanged || a.Current() != tab || tab.View != view || tab.Doc.Buffer != buffer ||
				a.Message != message || len(a.Issues) != issueCount || a.IssuesDropped != dropped {
				t.Fatal("keyboard Details added a host write, callback, report or owner change after explicit completion")
			}
			if err := tester.Click("关闭详情"); err != nil {
				t.Fatal(err)
			}
			tester.Frame()
			if tester.HasText("图形视图提示详情") {
				t.Fatal("Details did not close through its accessible action")
			}
		})
	}
}
