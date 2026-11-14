package markdownview

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

// The preview surface owns a public document-history entry. Toggling a task
// leaves the keyboard focus on the row; Ctrl+Z from there must undo the
// document edit itself. The draft editor, when open, keeps its own history
// and stays the innermost handler (covered separately).
func TestPreviewHistoryKeyboardUndoRevertsDocumentEdit(t *testing.T) {
	original := "前文🙂\r\n\r\n- [X] **完成**🙂  \r\n- [ ] 第二项\t\r\n"
	buffer := text.New(original)
	changes := 0
	v := Adapter().Create(editor.Host{Buffer: buffer, Changed: func() { changes++ }}).(*View)
	tester := ui.NewTester(v.Draw, 720, 480)
	info := taskAccess(t, tester, "已完成任务：**完成**🙂")
	tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	want := strings.Replace(original, "[X]", "[ ]", 1)
	if buffer.String() != want || changes != 1 || buffer.History().UndoDepth != 1 {
		t.Fatal("fixture toggle did not make one history entry", buffer.String(), changes)
	}
	if !taskAccess(t, tester, "未完成任务：**完成**🙂").Focused {
		t.Fatal("fixture lost focus on the toggled row")
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != original || !buffer.CanRedo() || changes != 2 {
		t.Fatal("Ctrl+Z did not undo the document edit", buffer.String(), changes)
	}
	info = taskAccess(t, tester, "已完成任务：**完成**🙂")
	if !info.Checked || info.Disabled {
		t.Fatal("preview did not re-render the restored task", info)
	}
	if tester.FramePending() {
		t.Fatal("history entry did not settle")
	}
}

// Redo is a first-class entry: Ctrl+Y and Ctrl+Shift+Z both replay the undone
// document edit through the same focus-scoped preview seam.
func TestPreviewHistoryKeyboardRedoReplaysUndoneEdit(t *testing.T) {
	original := "前文🙂\r\n\r\n- [ ] **完成**🙂  \r\n"
	buffer := text.New(original)
	changes := 0
	v := Adapter().Create(editor.Host{Buffer: buffer, Changed: func() { changes++ }}).(*View)
	tester := ui.NewTester(v.Draw, 720, 480)
	info := taskAccess(t, tester, "未完成任务：**完成**🙂")
	tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	want := strings.Replace(original, "[ ]", "[x]", 1)
	if buffer.String() != want || changes != 1 {
		t.Fatal("fixture toggle did not make one history entry", buffer.String(), changes)
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != original || changes != 2 {
		t.Fatal("Ctrl+Z did not undo", buffer.String(), changes)
	}
	tester.Key(ui.Cmd, ui.KeyY)
	if buffer.String() != want || changes != 3 || !buffer.CanUndo() {
		t.Fatal("Ctrl+Y did not redo", buffer.String(), changes)
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != original || changes != 4 {
		t.Fatal("second undo failed", buffer.String(), changes)
	}
	tester.Key(ui.Cmd|ui.Shift, ui.KeyZ)
	if buffer.String() != want || changes != 5 {
		t.Fatal("Ctrl+Shift+Z did not redo", buffer.String(), changes)
	}
	if !taskAccess(t, tester, "已完成任务：**完成**🙂").Checked {
		t.Fatal("preview did not re-render the redone task")
	}
	if tester.FramePending() {
		t.Fatal("history entry did not settle")
	}
}
