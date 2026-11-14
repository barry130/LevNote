package markdownview

import (
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

func TestPreviewHistoryPendingCellDoesNotFlushOrUndo(t *testing.T) {
	source := "- [ ] 任务\r\n\r\n" + tableViewSource + "\r\n[链接](other.md)\r\n"
	buffer := text.New(source)
	if err := buffer.ReplaceUTF16(3, 4, "x"); err != nil {
		t.Fatal(err)
	}
	changes := 0
	v := Adapter().Create(editor.Host{Buffer: buffer, OpenReference: func(string) {}, Changed: func() { changes++ }}).(*View)
	tester := ui.NewTester(v.Draw, 760, 1200)
	openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
	tester.Command("selectAll")
	tester.Type("乙🙂")
	tester.Key(0, ui.KeyF6)
	focusAccessByTab(t, tester, ui.RoleLink, linkName("链接"))
	want, version, history, selection := buffer.String(), buffer.Version(), buffer.History(), buffer.Selection()
	draft, value := v.draft, v.draft.Buffer.String()
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != want || buffer.Version() != version || buffer.History() != history || buffer.Selection() != selection || changes != 0 || v.draft != draft || draft.Buffer.String() != value {
		t.Fatal("pending draft was flushed or document history changed", buffer.String(), changes)
	}
	if !tester.HasText("草稿尚未写回；请先完成或放弃草稿，再撤销或重做文档") {
		t.Fatal("pending-draft history refusal not visible", tester.Texts())
	}
}

func TestPreviewHistoryChangedReleaseDoesNotStealNewFocus(t *testing.T) {
	buffer := text.New("- [ ] 任务\r\n")
	if err := buffer.ReplaceUTF16(3, 4, "x"); err != nil {
		t.Fatal(err)
	}
	var v *View
	var other ui.Handle
	v = Adapter().Create(editor.Host{Buffer: buffer, Changed: func() { v.Release(); other.Focus() }}).(*View)
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.Button(c, "新视图焦点").Bind(&other)
			v.Draw(c)
		})
	}, 720, 480)
	focusAccessByTab(t, tester, ui.RoleCheckBox, "已完成任务：任务")
	tester.Key(ui.Cmd, ui.KeyZ)
	button, ok := tester.AccessByRole(ui.RoleButton, "新视图焦点")
	if !ok || !button.Focused || buffer.String() != "- [ ] 任务\r\n" {
		t.Fatal("released history callback stole the new view focus", button, tester.Texts())
	}
}

func TestPreviewHistoryRetainedHandleSurvivesRelease(t *testing.T) {
	buffer := text.New("[链接](other.md)\r\n\r\n- [ ] 任务\r\n")
	var v *View
	v = Adapter().Create(editor.Host{Buffer: buffer, OpenReference: func(string) { v.Release() }}).(*View)
	var retained ui.Handle
	focused := false
	tester := ui.NewTester(func(c *ui.Context) {
		v.Draw(c)
		focused = retained.Focused(c)
	}, 720, 480)
	tester.Frame()
	retained = v.list.Handle
	focusAccessByTab(t, tester, ui.RoleLink, linkName("链接"))
	tester.Key(0, ui.KeyEnter)
	retained.Focus()
	tester.Frame()
	if !focused || tester.FramePending() {
		t.Fatal("public preview handle abandoned across synchronous Release")
	}
}

func TestPreviewHistoryDraftEditorOwnsContinuousUndoRedo(t *testing.T) {
	source := "- [ ] 任务\r\n\r\n" + tableViewSource
	buffer := text.New(source)
	if err := buffer.ReplaceUTF16(3, 4, "x"); err != nil {
		t.Fatal(err)
	}
	changes := 0
	v := Adapter().Create(editor.Host{Buffer: buffer, Changed: func() { changes++ }}).(*View)
	tester := ui.NewTester(v.Draw, 760, 700)
	openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
	draft := v.draft
	want, version, history := buffer.String(), buffer.Version(), buffer.History()
	tester.Command("selectAll")
	tester.Type("乙🙂")
	for _, step := range []struct {
		mods ui.Modifiers
		key  ui.Key
		want string
	}{{ui.Cmd, ui.KeyZ, "甲"}, {ui.Cmd, ui.KeyY, "乙🙂"}, {ui.Cmd, ui.KeyZ, "甲"}, {ui.Cmd | ui.Shift, ui.KeyZ, "乙🙂"}} {
		tester.Key(step.mods, step.key)
		field, ok := tester.AccessByRole(ui.RoleTextField, v.cellEdit.label)
		if !ok || !field.Focused || draft.Buffer.String() != step.want || buffer.String() != want || buffer.Version() != version || buffer.History() != history || changes != 0 || v.historyError != nil {
			t.Fatal("draft history escaped to the document or lost focus", step, draft.Buffer.String(), tester.Texts())
		}
	}
}

func TestPreviewHistoryButtonsReachPlainAndEmptyDocument(t *testing.T) {
	for _, source := range []string{"普通正文🙂\r\n", ""} {
		t.Run(source, func(t *testing.T) {
			buffer := text.New(source)
			if err := buffer.ReplaceUTF16(0, buffer.LenUTF16(), "后来正文\r\n"); err != nil {
				t.Fatal(err)
			}
			changes := 0
			v := Adapter().Create(editor.Host{Buffer: buffer, Changed: func() { changes++ }}).(*View)
			tester := ui.NewTester(v.Draw, 720, 480)
			clickDraftControl(t, tester, "撤销文档")
			if buffer.String() != source || changes != 1 {
				t.Fatal("ordinary document undo entry unreachable", tester.Texts())
			}
			clickDraftControl(t, tester, "重做文档")
			if buffer.String() != "后来正文\r\n" || changes != 2 {
				t.Fatal("empty/plain document redo entry unreachable", tester.Texts())
			}
		})
	}
}

func TestPreviewHistoryIMERefusalStaysInDraftAndFailedFlushStopsHistory(t *testing.T) {
	buffer := text.New("- [ ] 任务\r\n\r\n" + tableViewSource + "\r\n[链接](other.md)\r\n")
	if err := buffer.ReplaceUTF16(3, 4, "x"); err != nil {
		t.Fatal(err)
	}
	var reports []error
	changes := 0
	v := Adapter().Create(editor.Host{Buffer: buffer, Changed: func() { changes++ }, OpenReference: func(string) {}, Report: func(err error) { reports = append(reports, err) }}).(*View)
	tester := ui.NewTester(v.Draw, 760, 1200)
	openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
	draft := v.draft
	// Configure the actual independent Buffer's public limit before native
	// composition. The refused input has no marked bytes, so pendingDraft is
	// false: history must still respect the editor's failing Flush receipt.
	draft.Buffer = text.NewWithLimit("甲", len("甲"))
	tester.Frame()
	tester.Command("selectAll")
	tester.Compose("拒绝🙂", 3)
	want, version, history, selection := buffer.String(), buffer.Version(), buffer.History(), buffer.Selection()
	for _, step := range []struct {
		mods ui.Modifiers
		key  ui.Key
	}{{ui.Cmd, ui.KeyZ}, {ui.Cmd, ui.KeyY}, {ui.Cmd | ui.Shift, ui.KeyZ}} {
		tester.Key(step.mods, step.key)
		field, ok := tester.AccessByRole(ui.RoleTextField, v.cellEdit.label)
		if !ok || !field.Focused || v.historyError != nil || v.draft != draft || draft.Buffer.String() != "甲" || buffer.String() != want || buffer.Version() != version || buffer.History() != history || buffer.Selection() != selection || changes != 0 {
			t.Fatal("IME refusal escaped draft history or lost input", tester.Texts())
		}
	}
	link, ok := tester.AccessByRole(ui.RoleLink, linkName("链接"))
	if !ok || link.Offscreen {
		t.Fatal("preview link unavailable for pointer focus")
	}
	tester.ClickAtWith(ui.Cmd, link.Bounds.X+4, link.Bounds.Y+link.Bounds.H/2)
	tester.Key(ui.Cmd, ui.KeyZ)
	if !errors.Is(v.historyError, text.ErrSizeLimit) || len(reports) == 0 || !errors.Is(reports[len(reports)-1], text.ErrSizeLimit) || !tester.HasText(v.historyError.Error()) || buffer.String() != want || buffer.Version() != version || buffer.History() != history || changes != 0 || v.draft != draft {
		t.Fatal("failed draft Flush did not refuse document history", v.historyError, reports, tester.Texts())
	}
}

func TestPreviewHistoryChangedIdentityReceiptDoesNotStealFocus(t *testing.T) {
	for _, change := range []string{"buffer", "version", "selection", "readonly"} {
		t.Run(change, func(t *testing.T) {
			buffer := text.New("- [ ] 任务\r\n")
			if err := buffer.ReplaceUTF16(3, 4, "x"); err != nil {
				t.Fatal(err)
			}
			var v *View
			var other ui.Handle
			v = Adapter().Create(editor.Host{Buffer: buffer, Changed: func() {
				switch change {
				case "buffer":
					v.host.Buffer = text.New("新文档\r\n")
				case "version":
					_ = buffer.ReplaceUTF16(0, 0, "新的写入\r\n")
				case "selection":
					_ = buffer.SetSelection(1, 1)
				case "readonly":
					v.host.ReadOnly = true
				}
				other.Focus()
			}}).(*View)
			tester := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					ui.Button(c, "回调焦点").Bind(&other)
					v.Draw(c)
				})
			}, 720, 480)
			focusAccessByTab(t, tester, ui.RoleCheckBox, "已完成任务：任务")
			tester.Key(ui.Cmd, ui.KeyZ)
			button, ok := tester.AccessByRole(ui.RoleButton, "回调焦点")
			if !ok || !button.Focused || buffer.History().RedoDepth != map[bool]int{true: 0, false: 1}[change == "version"] {
				t.Fatal("stale post-history receipt stole host focus or changed history", button, buffer.History())
			}
		})
	}
}

func TestPreviewHistoryNativeCommandsFromPreview(t *testing.T) {
	buffer := text.New("- [ ] 任务\r\n")
	if err := buffer.ReplaceUTF16(3, 4, "x"); err != nil {
		t.Fatal(err)
	}
	changes := 0
	v := Adapter().Create(editor.Host{Buffer: buffer, Changed: func() { changes++ }}).(*View)
	tester := ui.NewTester(v.Draw, 720, 480)
	focusAccessByTab(t, tester, ui.RoleCheckBox, "已完成任务：任务")
	tester.Command("undo")
	if buffer.String() != "- [ ] 任务\r\n" || changes != 1 {
		t.Fatal("native undo command from preview unavailable", tester.Texts())
	}
	tester.Command("redo")
	if buffer.String() != "- [x] 任务\r\n" || changes != 2 {
		t.Fatal("native redo command lost continuous preview focus", tester.Texts())
	}
}

func TestPreviewHistoryDeferredShortcutDoesNotRetargetChangedOwner(t *testing.T) {
	buffer := text.New("- [ ] 任务\r\n")
	if err := buffer.ReplaceUTF16(3, 4, "x"); err != nil {
		t.Fatal(err)
	}
	other := text.New("新文档\r\n")
	if err := other.ReplaceUTF16(0, 0, "后来\r\n"); err != nil {
		t.Fatal(err)
	}
	changes := 0
	v := Adapter().Create(editor.Host{Buffer: buffer, Changed: func() { changes++ }}).(*View)
	armed := false
	tester := ui.NewTester(func(c *ui.Context) {
		v.Draw(c)
		if armed {
			armed = false
			v.host.Buffer = other
		}
	}, 720, 480)
	focusAccessByTab(t, tester, ui.RoleCheckBox, "已完成任务：任务")
	want, history := other.String(), other.History()
	armed = true
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != "- [x] 任务\r\n" || other.String() != want || other.History() != history || changes != 0 {
		t.Fatal("deferred history shortcut retargeted a new owner")
	}
}

func TestPreviewHistoryControlsKeepContinuousKeysOnPartialDocument(t *testing.T) {
	buffer := text.New(strings.Repeat("x", 8<<20+1))
	if err := buffer.ReplaceUTF16(0, 0, "后"); err != nil {
		t.Fatal(err)
	}
	changes := 0
	v := Adapter().Create(editor.Host{Buffer: buffer, Changed: func() { changes++ }}).(*View)
	tester := ui.NewTester(v.Draw, 720, 480)
	if !v.doc.Partial {
		t.Fatal("partial fixture did not reach refusal preview")
	}
	focusAccessByTab(t, tester, ui.RoleButton, "撤销文档")
	tester.Key(ui.Cmd, ui.KeyZ)
	if changes != 1 || buffer.LenBytes() != 8<<20+1 {
		t.Fatal("partial visible control did not route history shortcut")
	}
	tester.Key(ui.Cmd, ui.KeyY)
	if changes != 2 || buffer.LenBytes() != 8<<20+4 {
		t.Fatal("partial continuous redo lost public focus")
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	tester.Key(ui.Cmd|ui.Shift, ui.KeyZ)
	if changes != 4 || buffer.LenBytes() != 8<<20+4 || tester.FramePending() {
		t.Fatal("partial continuous ShiftZ redo lost focus or settle frame")
	}
}

func TestPreviewHistoryReadonlyRefusesDocumentUndo(t *testing.T) {
	original := "- [ ] 任务\r\n\r\n[链接](other.md)\r\n"
	buffer := text.New(original)
	if err := buffer.ReplaceUTF16(3, 4, "x"); err != nil {
		t.Fatal(err)
	}
	want, version, history := buffer.String(), buffer.Version(), buffer.History()
	var reports []error
	changes := 0
	v := Adapter().Create(editor.Host{Buffer: buffer, ReadOnly: true, OpenReference: func(string) {}, Changed: func() { changes++ }, Report: func(err error) { reports = append(reports, err) }}).(*View)
	tester := ui.NewTester(v.Draw, 720, 480)
	focusAccessByTab(t, tester, ui.RoleLink, linkName("链接"))
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != want || buffer.Version() != version || buffer.History() != history || changes != 0 {
		t.Fatal("readonly history mutated document", buffer.String(), changes)
	}
	if len(reports) == 0 || !strings.Contains(reports[len(reports)-1].Error(), "只读") || !tester.HasText(reports[len(reports)-1].Error()) {
		t.Fatal("readonly history refusal was not visible", reports, tester.Texts())
	}
}
