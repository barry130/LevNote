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
	"levnote.local/native/core/document"
)

const markdownTasksBody = "# 标题\r\n\r\n- [ ] 未完成\r\n- [x] 小写完成\r\n- [X] 大写完成\r\n- [ ] **粗体** 🙂\r\n- [ ] \r\n1. [ ] 有序任务\r\n2. [X] 有序完成\r\n- [ ] 末尾邻居\r\n"

func markdownTasksFixture() []byte {
	return append([]byte{0xef, 0xbb, 0xbf}, []byte(markdownTasksBody)...)
}

func taskAccessName(checked bool, label string) string {
	if label == "" {
		label = "（空任务）"
	}
	if checked {
		return "已完成任务：" + label
	}
	return "未完成任务：" + label
}

func boundedTaskAccessLabel(label string) string {
	const limit = 128
	if len(label) <= limit {
		return label
	}
	used := 0
	end := 0
	for end < len(label) {
		_, size := utf8.DecodeRuneInString(label[end:])
		if used+size > limit {
			break
		}
		used += size
		end += size
	}
	return label[:end] + "…"
}

func openRealMarkdownTasks(t *testing.T, content []byte, width, height int) (*App, *Tab, *ui.Tester, string) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "tasks.md")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "markdown" ||
		tab.Mode != "preview" || tab.Doc == nil || tab.Doc.Buffer == nil {
		t.Fatalf("real Markdown task fixture did not open in preview: %#v", tab)
	}
	tester := ui.NewTester(a.View, width, height)
	tester.Frame()
	return a, tab, tester, path
}

func requireTaskAccess(t *testing.T, tester *ui.Tester, checked bool, label string) ui.AccessInfo {
	t.Helper()
	name := taskAccessName(checked, label)
	info, ok := tester.AccessByRole(ui.RoleCheckBox, name)
	if !ok {
		t.Fatalf("checkbox %q is missing from the accessibility tree: %#v", name, tester.Texts())
	}
	if info.Name != name || info.Description != name || info.Checked != checked || info.Mixed {
		t.Fatalf("checkbox accessibility state for %q = %#v", name, info)
	}
	if info.Disabled {
		t.Fatalf("writable checkbox %q is disabled", name)
	}
	if !info.Focusable {
		t.Fatalf("writable checkbox %q is not keyboard focusable", name)
	}
	if info.Bounds.W <= 0 || info.Bounds.H <= 0 || info.Offscreen {
		t.Fatalf("checkbox %q has no visible hit target: %#v", name, info.Bounds)
	}
	return info
}

func requireOneMiddleMarkerByteDiff(t *testing.T, before, after []byte) int {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("task toggle changed file length: %d -> %d", len(before), len(after))
	}
	index, changed := -1, 0
	for i := range before {
		if before[i] == after[i] {
			continue
		}
		index, changed = i, changed+1
	}
	if changed != 1 {
		t.Fatalf("task toggle changed %d bytes; want exactly one marker byte", changed)
	}
	if index <= 0 || index+1 >= len(before) ||
		before[index-1] != '[' || before[index+1] != ']' ||
		after[index-1] != '[' || after[index+1] != ']' {
		t.Fatalf("changed byte %d is not the middle of one task marker", index)
	}
	if !((before[index] == ' ' && (after[index] == 'x' || after[index] == 'X')) ||
		((before[index] == 'x' || before[index] == 'X') && after[index] == ' ')) {
		t.Fatalf("task marker changed from %q to %q", before[index], after[index])
	}
	return index
}

func TestMarkdownTaskCheckboxRealAdapterPointerUndoRedoSaveAndViews(t *testing.T) {
	before := markdownTasksFixture()
	a, tab, tester, path := openRealMarkdownTasks(t, before, 1180, 780)
	if tab.Doc.Dirty() {
		t.Fatal("opening the task fixture made it dirty")
	}
	if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, before) {
		t.Fatalf("opening changed source bytes: len=%d err=%v", len(got), err)
	}
	if _, err := tab.View.Export("html"); err != nil {
		t.Fatalf("opening/exporting unchanged tasks failed: %v", err)
	}
	if tab.Doc.Dirty() {
		t.Fatal("exporting unchanged tasks made the document dirty")
	}
	if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, before) {
		t.Fatalf("exporting unchanged tasks changed source bytes: len=%d err=%v", len(got), err)
	}

	requireTaskAccess(t, tester, false, "未完成")
	requireTaskAccess(t, tester, true, "小写完成")
	requireTaskAccess(t, tester, true, "大写完成")
	requireTaskAccess(t, tester, false, "**粗体** 🙂")
	requireTaskAccess(t, tester, false, "")
	requireTaskAccess(t, tester, false, "有序任务")
	requireTaskAccess(t, tester, true, "有序完成")

	const unchecked = "未完成任务：未完成"
	buffer := tab.Doc.Buffer
	if err := buffer.SetSelection(buffer.LenUTF16()-2, 2); err != nil {
		t.Fatal(err)
	}
	selection := buffer.Selection()
	if selection.Anchor <= selection.Head {
		t.Fatalf("test selection is not reversed: %#v", selection)
	}
	history := buffer.History()
	if history.UndoDepth != 0 {
		t.Fatalf("fresh fixture unexpectedly has undo history: %#v", history)
	}
	if err := tester.Click(unchecked); err != nil {
		t.Fatalf("pointer activation of %q failed: %v", unchecked, err)
	}
	tester.Frame()
	checkedName := taskAccessName(true, "未完成")
	checkedInfo, ok := tester.AccessByRole(ui.RoleCheckBox, checkedName)
	if !ok || !checkedInfo.Checked || checkedInfo.Disabled {
		t.Fatalf("pointer activation did not expose a checked writable control: %#v", checkedInfo)
	}
	afterToggleSource := []byte(buffer.String())
	markerIndex := requireOneMiddleMarkerByteDiff(t, []byte(markdownTasksBody), afterToggleSource)
	if afterToggleSource[markerIndex] != 'x' {
		t.Fatalf("pointer activation did not write lowercase x")
	}
	afterToggleDisk := append([]byte{0xef, 0xbb, 0xbf}, afterToggleSource...)
	if got := mustReadFile(t, path); !bytes.Equal(got, before) {
		t.Fatal("toggling without saving unexpectedly changed the file on disk")
	}
	if buffer.Selection() != selection {
		t.Fatalf("pointer activation changed reversed source selection: %#v -> %#v", selection, buffer.Selection())
	}
	if history := buffer.History(); history.UndoDepth != 1 || history.RedoDepth != 0 {
		t.Fatalf("pointer activation was not one transaction: %#v", history)
	}

	tab.Editor.Command("undo")
	tester.Frame()
	if buffer.String() != markdownTasksBody {
		t.Fatal("one undo did not restore exact original source")
	}
	if got := mustReadFile(t, path); !bytes.Equal(got, before) {
		t.Fatal("undo changed the unsaved file on disk")
	}
	if buffer.Selection() != selection {
		t.Fatalf("undo did not restore reversed source selection: %#v", buffer.Selection())
	}
	if history := buffer.History(); history.UndoDepth != 0 || history.RedoDepth != 1 {
		t.Fatalf("undo transaction depth = %#v", history)
	}
	requireTaskAccess(t, tester, false, "未完成")

	tab.Editor.Command("redo")
	tester.Frame()
	if buffer.String() != string(afterToggleSource) {
		t.Fatal("one redo did not restore the checked marker")
	}
	if buffer.Selection() != selection {
		t.Fatalf("redo did not preserve reversed source selection: %#v", buffer.Selection())
	}
	if history := buffer.History(); history.UndoDepth != 1 || history.RedoDepth != 0 {
		t.Fatalf("redo transaction depth = %#v", history)
	}
	requireTaskAccess(t, tester, true, "未完成")

	if !a.Save(tab, false) {
		t.Fatalf("saving task toggle failed: %s", a.Message)
	}
	if tab.Mode != "preview" {
		t.Fatalf("saving task toggle changed mode to %q", tab.Mode)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(saved, afterToggleDisk) {
		t.Fatal("saved task file differs from the one-byte toggled source")
	}
	if tab.Doc.Dirty() {
		t.Fatal("saved task document remains dirty")
	}

	if !a.setMode(tab, "source") || tab.Mode != "source" {
		t.Fatalf("source mode failed: %s", a.Message)
	}
	tester.Frame()
	if _, ok := tester.AccessByRole(ui.RoleCheckBox, checkedName); ok {
		t.Fatal("source mode still exposed a task checkbox")
	}
	if !a.setMode(tab, "preview") || tab.Mode != "preview" {
		t.Fatalf("preview mode failed: %s", a.Message)
	}
	tester.Frame()
	requireTaskAccess(t, tester, true, "未完成")
	if !a.setMode(tab, "split") || tab.Mode != "split" {
		t.Fatalf("split mode failed: %s", a.Message)
	}
	tester.Frame()
	requireTaskAccess(t, tester, true, "未完成")
	const splitTarget = "未完成任务：有序任务"
	if err := tester.Click(splitTarget); err != nil {
		t.Fatalf("split-view checkbox activation failed: %v", err)
	}
	tester.Frame()
	requireTaskAccess(t, tester, true, "有序任务")
	if got := []byte(buffer.String()); len(got) != len(afterToggleSource) {
		t.Fatalf("split checkbox activation changed source length: %d", len(got))
	}
	tab.Editor.Command("undo")
	tester.Frame()
	if buffer.String() != string(afterToggleSource) {
		t.Fatal("split checkbox undo did not restore the preview-mode source")
	}
	requireTaskAccess(t, tester, false, "有序任务")
	if tab.Doc.Buffer.String() != string(afterToggleSource) {
		t.Fatal("source and split views diverged from the authoritative task source")
	}

	reopened := newTestApp(t, filepath.Join(t.TempDir(), "config"))
	reopened.Open(path)
	reopenedTab := reopened.Current()
	reopenedTester := ui.NewTester(reopened.View, 1180, 780)
	reopenedTester.Frame()
	if reopenedTab == nil || reopenedTab.Adapter == nil ||
		reopenedTab.Adapter.ID != "markdown" || reopenedTab.Mode != "preview" {
		t.Fatalf("saved task file did not reopen through Markdown preview: %#v", reopenedTab)
	}
	requireTaskAccess(t, reopenedTester, true, "未完成")
	if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, saved) {
		t.Fatalf("reopening changed saved task bytes: len=%d err=%v", len(got), err)
	}
}

func TestMarkdownTaskCheckboxRealAdapterSpaceAndReadonly(t *testing.T) {
	t.Run("space", func(t *testing.T) {
		before := markdownTasksFixture()
		_, tab, tester, path := openRealMarkdownTasks(t, before, 1180, 780)
		name := taskAccessName(true, "大写完成")
		info, ok := tester.AccessByRole(ui.RoleCheckBox, name)
		if !ok || !info.Focusable || info.Bounds.W <= 0 || info.Bounds.H <= 0 {
			t.Fatalf("keyboard target is unavailable: %#v", info)
		}
		tester.Press(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
		tester.CancelPress()
		tester.Frame()
		if !tester.Focused(name) {
			t.Fatalf("pointer press/cancel did not focus %q", name)
		}
		if !bytes.Equal(mustReadFile(t, path), before) {
			t.Fatal("focusing the checkbox changed source bytes")
		}
		tester.Key(0, ui.KeySpace)
		tester.Frame()
		requireTaskAccess(t, tester, false, "大写完成")
		afterSource := []byte(tab.Doc.Buffer.String())
		markerIndex := requireOneMiddleMarkerByteDiff(t, []byte(markdownTasksBody), afterSource)
		if afterSource[markerIndex] != ' ' {
			t.Fatalf("Space did not change uppercase X to ASCII space")
		}
		if got := mustReadFile(t, path); !bytes.Equal(got, before) {
			t.Fatal("Space without saving unexpectedly changed the file on disk")
		}
		if history := tab.Doc.Buffer.History(); history.UndoDepth != 1 || history.RedoDepth != 0 {
			t.Fatalf("Space activation was not one transaction: %#v", history)
		}
	})

	t.Run("readonly-then-save-as", func(t *testing.T) {
		a := newTestApp(t, t.TempDir())
		doc := document.New(markdownTasksBody)
		doc.ReadOnly = true
		tab := a.add(doc, "readonly.md", "")
		if tab == nil || tab.Mode != "preview" || tab.Adapter == nil || tab.Adapter.ID != "markdown" {
			t.Fatalf("readonly task document did not configure preview before mount: %#v", tab)
		}
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		const name = "未完成任务：未完成"
		info, ok := tester.AccessByRole(ui.RoleCheckBox, name)
		if !ok || !info.Disabled || info.Checked {
			t.Fatalf("readonly checkbox state = %#v", info)
		}
		if info.Bounds.W <= 0 || info.Bounds.H <= 0 || info.Offscreen {
			t.Fatalf("readonly checkbox has no visible target: %#v", info.Bounds)
		}
		version, selection, history := tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection(), tab.Doc.Buffer.History()
		tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
		tester.Frame()
		if tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.Selection() != selection ||
			tab.Doc.Buffer.History() != history || tab.Doc.Buffer.String() != markdownTasksBody {
			t.Fatal("pointer activation changed a readonly task document")
		}
		if still, ok := tester.AccessByRole(ui.RoleCheckBox, name); !ok || !still.Disabled || still.Checked {
			t.Fatalf("readonly checkbox changed after pointer attempt: %#v", still)
		}

		destination := filepath.Join(t.TempDir(), "writable.md")
		a.saveDialog = func(*Tab) (string, error) { return destination, nil }
		if !a.Save(tab, true) {
			t.Fatalf("readonly Save As failed: %s", a.Message)
		}
		if tab.Doc.Path != destination || tab.Doc.ReadOnly || tab.Editor.ReadOnly {
			t.Fatalf("Save As retained readonly state: path=%q doc=%v editor=%v",
				tab.Doc.Path, tab.Doc.ReadOnly, tab.Editor.ReadOnly)
		}
		tester.Frame()
		writable, ok := tester.AccessByRole(ui.RoleCheckBox, name)
		if !ok || writable.Disabled || !writable.Focusable {
			t.Fatalf("writable Save As copy did not expose an editable checkbox: %#v", writable)
		}
		if err := tester.Click(name); err != nil {
			t.Fatalf("writable copy checkbox click failed: %v", err)
		}
		tester.Frame()
		requireTaskAccess(t, tester, true, "未完成")
	})
}

func TestMarkdownTaskCheckboxRealAdapterFlushesOtherBlockCompose(t *testing.T) {
	before := markdownTasksFixture()
	_, tab, tester, _ := openRealMarkdownTasks(t, before, 1180, 780)
	beginRealMarkdownComposition(t, tab, tester)
	tester.Frame()
	if !strings.Contains(tab.Doc.Buffer.String(), "# 标题") ||
		strings.Contains(tab.Doc.Buffer.String(), "# 中文🙂") {
		t.Fatalf("composition was not pending in a distinct heading block: %q", tab.Doc.Buffer.String())
	}
	const name = "未完成任务：未完成"
	if _, ok := tester.AccessByRole(ui.RoleCheckBox, name); !ok {
		t.Fatalf("pending composition hid the task target: %#v", tester.Texts())
	}
	if err := tester.Click(name); err != nil {
		t.Fatalf("checkbox activation with pending composition failed: %v", err)
	}
	tester.Frame()
	sourceAfterFlush := tab.Doc.Buffer.String()
	if !strings.Contains(sourceAfterFlush, "# 中文🙂\r\n") ||
		!strings.Contains(sourceAfterFlush, "- [x] 未完成\r\n") {
		t.Fatalf("checkbox did not flush then toggle source: %q", sourceAfterFlush)
	}
	requireTaskAccess(t, tester, true, "未完成")

	tab.Editor.Command("undo")
	tester.Frame()
	if !strings.Contains(tab.Doc.Buffer.String(), "# 中文🙂\r\n") ||
		!strings.Contains(tab.Doc.Buffer.String(), "- [ ] 未完成\r\n") {
		t.Fatalf("one undo did not isolate the checkbox transaction after Flush: %q", tab.Doc.Buffer.String())
	}
	requireTaskAccess(t, tester, false, "未完成")
	tab.Editor.Command("redo")
	tester.Frame()
	if !strings.Contains(tab.Doc.Buffer.String(), "- [x] 未完成\r\n") {
		t.Fatalf("redo did not restore the checkbox transaction: %q", tab.Doc.Buffer.String())
	}
	requireTaskAccess(t, tester, true, "未完成")
}

func TestMarkdownTaskCheckboxRealAdapterFailedFlushRetainsDraftAndSource(t *testing.T) {
	before := markdownTasksFixture()
	a, tab, tester, path := openRealMarkdownTasks(t, before, 760, 480)
	beginRealMarkdownComposition(t, tab, tester)
	tester.Frame()

	// This is the same overlapping host edit used by the existing real-adapter
	// app integration test TestNarrowMarkdownConflictKeepsViewOutOfStatusRow.
	if err := tab.Doc.Buffer.ReplaceUTF16(2, 4, "外部"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	version, selection, history := tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection(), tab.Doc.Buffer.History()
	sourceBeforeClick := tab.Doc.Buffer.String()
	viewBeforeClick := tab.View
	const name = "未完成任务：未完成"
	if _, ok := tester.AccessByRole(ui.RoleCheckBox, name); !ok {
		t.Fatalf("conflicting draft hid the stale preview checkbox: %#v", tester.Texts())
	}
	if err := tester.Click(name); err != nil {
		t.Fatalf("real checkbox click during failed Flush failed at input delivery: %v", err)
	}
	tester.Frame()
	if tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.Selection() != selection ||
		tab.Doc.Buffer.History() != history || tab.Doc.Buffer.String() != sourceBeforeClick {
		t.Fatal("failed Flush checkbox activation changed source, selection or history")
	}
	if tab.View != viewBeforeClick || tab.Mode != "preview" || tab.Editor == nil {
		t.Fatalf("failed Flush replaced or closed the active draft view: mode=%q", tab.Mode)
	}
	if got := mustReadFile(t, path); !bytes.Equal(got, before) {
		t.Fatal("failed Flush checkbox activation changed the file on disk")
	}
	if !strings.Contains(a.Message, "草稿仍保留") {
		t.Fatalf("failed Flush refusal was not visible: %q", a.Message)
	}
	if !tester.HasText("打开未同步副本") || !tester.HasText("完成编辑") {
		t.Fatalf("failed Flush discarded the recoverable draft controls: %#v", tester.Texts())
	}
}

func TestMarkdownTaskCheckboxesNarrowScreenshot(t *testing.T) {
	out := os.Getenv("LEVNOTE_MARKDOWN_TASKS_SCREENSHOT_DIR")
	if out == "" {
		t.Skip("set LEVNOTE_MARKDOWN_TASKS_SCREENSHOT_DIR for task-checkbox visual acceptance")
	}
	longLabel := strings.Repeat("长任务🙂", 20)
	body := "# 任务清单\r\n\r\n- [ ] 待完成事项\r\n- [x] 已完成事项\r\n- [X] 大写已完成\r\n- [ ] " + longLabel + "\r\n"
	content := append([]byte{0xef, 0xbb, 0xbf}, []byte(body)...)
	a, tab, tester, _ := openRealMarkdownTasks(t, content, 760, 480)
	a.Dark = true
	tester.SetDark(true)
	tester.Frame()
	tester.Advance(time.Second)
	tester.Frame()

	const statusTop = float32(480 - 29)
	for _, label := range []string{"待完成事项", "已完成事项", "大写已完成"} {
		info := requireTaskAccess(t, tester, label == "已完成事项" || label == "大写已完成", label)
		if info.Bounds.X < 0 || info.Bounds.Y < 0 ||
			info.Bounds.X+info.Bounds.W > 760 || info.Bounds.Y+info.Bounds.H > statusTop {
			t.Fatalf("checkbox %q overlaps chrome or status row: %#v", info.Name, info.Bounds)
		}
	}
	bounded := boundedTaskAccessLabel(longLabel)
	longInfo, ok := tester.AccessByRole(ui.RoleCheckBox, taskAccessName(false, bounded))
	if !ok || longInfo.Bounds.W <= 0 || longInfo.Bounds.H <= 0 ||
		longInfo.Bounds.Y+longInfo.Bounds.H > statusTop || longInfo.Offscreen {
		t.Fatalf("bounded long task label is not visible: %#v", longInfo)
	}
	if _, ok := tester.AccessByRole(ui.RoleCheckBox, taskAccessName(false, longLabel)); ok {
		t.Fatal("long task accessibility label was not bounded")
	}
	for _, text := range []string{"tasks.md", "阅读 / 编辑", "源码", "分屏"} {
		rect, ok := tester.Find(text)
		if !ok || rect.W <= 0 || rect.H <= 0 || rect.Y+rect.H > statusTop {
			t.Fatalf("narrow app chrome text %q is clipped: %#v texts=%#v", text, rect, tester.Texts())
		}
	}
	status, ok := tester.Find(a.Message)
	if !ok || status.W <= 0 || status.H <= 0 || status.Y < statusTop ||
		status.X < 0 || status.X+status.W > 760 || status.Y+status.H > 480 {
		t.Fatalf("narrow status text is clipped or outside the status row: %#v", status)
	}
	captureAcceptanceTester(t, filepath.Join(out, "markdown-tasks-narrow-dark-760x480.png"), tester)
	if tab.Mode != "preview" {
		t.Fatalf("task screenshot changed mode to %q", tab.Mode)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	value, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
