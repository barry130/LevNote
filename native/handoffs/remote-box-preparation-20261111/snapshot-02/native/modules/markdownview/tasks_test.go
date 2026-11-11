package markdownview

import (
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/ui/editor"
)

func taskAccess(t *testing.T, tester *ui.Tester, name string) ui.AccessInfo {
	t.Helper()
	info, ok := tester.AccessByRole(ui.RoleCheckBox, name)
	if !ok || info.Bounds.W <= 0 || info.Bounds.H <= 0 {
		t.Fatalf("native task not reachable: %q %+v texts=%q", name, info, tester.Texts())
	}
	return info
}

func captureTask(t *testing.T, v *View, index int) taskTarget {
	t.Helper()
	v.refresh()
	p := previewSnapshot{v.doc, v.rows, v.docBuffer, v.version}
	for _, row := range p.rows {
		if row.item >= 0 {
			item := p.doc.Blocks[row.block].Items[row.item]
			if item.Task != nil {
				if index == 0 {
					return p.taskTarget(row, item)
				}
				index--
			}
		}
	}
	t.Fatal("task fixture missing")
	return taskTarget{}
}

func TestTaskNativePointerSpaceRapidClicksPreserveBytesHistoryAndFocus(t *testing.T) {
	original := "前文🙂\r\n\r\n- [X] **完成**🙂  \r\n- [ ] 第二项\t\r\n\r\n0003) [ ] 有序\r\n"
	buffer := text.New(original)
	selection := text.Range{Anchor: buffer.LenUTF16() - 2, Head: 2}
	if err := buffer.SetSelection(selection.Anchor, selection.Head); err != nil {
		t.Fatal(err)
	}
	changes := 0
	v := Adapter().Create(editor.Host{Buffer: buffer, Changed: func() { changes++ }}).(*View)
	tester := ui.NewTester(v.Draw, 720, 480)
	info := taskAccess(t, tester, "已完成任务：**完成**🙂")
	if !info.Checked || !info.Focusable || info.Disabled || info.Bounds.W > 24 {
		t.Fatal("checkbox state/box-only hit target", info)
	}
	target := captureTask(t, v, 0)
	tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	want := strings.Replace(original, "[X]", "[ ]", 1)
	if buffer.String() != want || buffer.Selection() != selection || changes != 1 || buffer.History().UndoDepth != 1 || v.draft != nil {
		t.Fatal("native pointer did not make exact single edit", buffer.String(), buffer.Selection(), changes)
	}
	info = taskAccess(t, tester, "未完成任务：**完成**🙂")
	if !info.Focused || info.Checked || captureTask(t, v, 0).rowID != target.rowID {
		t.Fatal("own toggle replaced focused row", info)
	}
	if !buffer.Undo() || buffer.String() != original || buffer.Selection() != selection || !buffer.Redo() || buffer.String() != want || buffer.Selection() != selection {
		t.Fatal("one undo/redo must restore X spelling and reversed selection")
	}
	// Undo/redo is an external overlapping edit; establish a fresh actual focus
	// before verifying repeated native keyboard toggles of our own row.
	tester.Frame()
	info = taskAccess(t, tester, "未完成任务：**完成**🙂")
	tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	for i := 0; i < 6; i++ {
		tester.Key(0, ui.KeySpace)
		name := "未完成任务：**完成**🙂"
		if i%2 == 1 {
			name = "已完成任务：**完成**🙂"
		}
		if !taskAccess(t, tester, name).Focused || v.draft != nil {
			t.Fatal("Space lost focus or opened list draft", i)
		}
	}
	info = taskAccess(t, tester, "已完成任务：**完成**🙂")
	for i := 0; i < 6; i++ {
		tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
		if v.draft != nil {
			t.Fatal("rapid checkbox clicks bubbled into row draft", i)
		}
	}
	if changes != 14 || buffer.String() != strings.Replace(original, "[X]", "[x]", 1) || buffer.Selection() != selection {
		t.Fatal("repeat toggles lost source authority", changes, buffer.String())
	}
	if tester.FramePending() {
		t.Fatal("task controls failed to settle")
	}
	if _, ok := tester.Find("3."); !ok {
		t.Fatal("ordered task lost visible numbering")
	}
}

func TestTaskEmptyClippedNamesReadonlyAndLiveReadonlyRefusal(t *testing.T) {
	longLabel := strings.Repeat("中🙂", 40)
	for _, readonly := range []bool{false, true} {
		buffer := text.New("- [ ]\t \r\n- [x] " + longLabel + "\r\n")
		var reported error
		v := Adapter().Create(editor.Host{Buffer: buffer, ReadOnly: readonly, Report: func(err error) { reported = err }}).(*View)
		tester := ui.NewTester(v.Draw, 265, 480)
		info := taskAccess(t, tester, "未完成任务：（空任务）")
		if info.Disabled != readonly || info.Checked {
			t.Fatal("empty task state", info)
		}
		name := taskName(v.doc.Blocks[0].Items[1])
		info2 := taskAccess(t, tester, name)
		if !strings.HasSuffix(name, "…") || !utf8.ValidString(name) || len(name) > len("已完成任务：")+131 || info2.Description != name {
			t.Fatal("label/tooltip not bounded", name, info2)
		}
		source := buffer.String()
		if !readonly {
			// Last frame was enabled; the command must recheck live host state
			// before any source edit, even before a disabled rebuild exists.
			v.host.ReadOnly = true
		}
		tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
		tester.Key(0, ui.KeySpace)
		if source != buffer.String() || buffer.CanUndo() || v.draft != nil {
			t.Fatal("readonly task edited source")
		}
		if !readonly && (reported == nil || !strings.Contains(reported.Error(), "只读")) {
			t.Fatal("live readonly refusal was silent", reported)
		}
		if !taskAccess(t, tester, "未完成任务：（空任务）").Disabled {
			t.Fatal("readonly control not visibly disabled")
		}
	}
}

func TestTaskCapturedRangeMappingsAndContext(t *testing.T) {
	for _, change := range []string{"before", "duplicate-before", "closed-fence", "open-fence", "target-label", "target-marker", "delete", "restore", "expired", "buffer", "foreign-group", "partial", "oversize"} {
		t.Run(change, func(t *testing.T) {
			original := "前文🙂\n\n- [ ] 相同\n- [ ] 相同\n\n尾部\n"
			buffer := text.New(original)
			changes := 0
			v := Adapter().Create(editor.Host{Buffer: buffer, Changed: func() { changes++ }}).(*View)
			target := captureTask(t, v, 1)
			prefix := ""
			success := false
			switch change {
			case "before":
				prefix, success = "新增🙂\n\n", true
			case "duplicate-before":
				prefix, success = "- [ ] 相同\n\n", true
			case "closed-fence":
				prefix, success = "```\ncode\n```\n\n", true
			case "open-fence":
				prefix = "```\n"
			case "target-label":
				_ = buffer.ReplaceUTF16(target.marker.End()+1, target.marker.End()+3, "更改")
			case "target-marker":
				_ = buffer.ReplaceUTF16(target.marker.Start()+1, target.marker.Start()+2, "x")
			case "delete":
				_ = buffer.ReplaceUTF16(target.item.Start(), target.item.End(), "")
			case "restore":
				buffer.Restore(buffer.Snapshot())
			case "expired":
				for i := 0; i < 65; i++ {
					_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), "x")
				}
			case "buffer":
				v.host.Buffer = text.New(original)
			case "foreign-group":
				_ = buffer.BeginGroup()
			case "partial":
				_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), "\n\n"+strings.Repeat("- x\n", 4200))
			case "oversize":
				_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), strings.Repeat("x", markdown.MaxParseBytes))
			}
			if prefix != "" {
				_ = buffer.ReplaceUTF16(0, 0, prefix)
			}
			authority := v.host.Buffer.String()
			selection := v.host.Buffer.Selection()
			depth := v.host.Buffer.History().UndoDepth
			err := v.toggleTask(target)
			if success {
				want := prefix + strings.Replace(original, "- [ ] 相同\n\n尾部", "- [x] 相同\n\n尾部", 1)
				if err != nil || buffer.String() != want || buffer.Selection() != selection || changes != 1 || buffer.History().UndoDepth != depth+1 {
					t.Fatal("mapped exact duplicate task failed", err, buffer.String())
				}
			} else if err == nil || v.host.Buffer.String() != authority || v.host.Buffer.Selection() != selection || changes != 0 || v.host.Buffer.History().UndoDepth != depth {
				t.Fatal("stale/contextual task accepted or changed state", err)
			}
			if change == "foreign-group" && !buffer.InGroup() {
				t.Fatal("foreign transaction stolen")
			}
		})
	}
}

func TestTaskNativeCompositionFlushAndConflict(t *testing.T) {
	for _, mode := range []string{"other-block", "same-list", "failed-flush"} {
		t.Run(mode, func(t *testing.T) {
			original := "# 标题\r\n\r\n- [ ] task\r\n- [ ] second\r\n\r\n尾部\r\n"
			buffer := text.New(original)
			var reports []error
			changes := 0
			v := Adapter().Create(editor.Host{Buffer: buffer, Changed: func() { changes++ }, Report: func(err error) { reports = append(reports, err) }}).(*View)
			tester := ui.NewTester(v.Draw, 720, 640)
			if mode == "same-list" {
				beginDraft(t, v, tester, "task")
				tester.Command("selectAll")
				tester.Compose("- [ ] changed\r\n- [ ] second\r\n", 10)
			} else {
				beginDraft(t, v, tester, "标题")
				tester.Command("selectAll")
				tester.Compose("新标题🙂", 4)
			}
			if mode == "failed-flush" {
				_ = buffer.ReplaceUTF16(2, 4, "外部")
			}
			authority := buffer.String()
			info := taskAccess(t, tester, "未完成任务：task")
			tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
			switch mode {
			case "other-block":
				want := strings.Replace(strings.Replace(original, "标题", "新标题🙂", 1), "[ ] task", "[x] task", 1)
				if buffer.String() != want || changes != 2 || len(reports) != 0 {
					t.Fatal("Flush/toggle did not preserve separate history", buffer.String(), changes, reports)
				}
				draft := v.draft
				if !buffer.Undo() || buffer.String() != strings.Replace(original, "标题", "新标题🙂", 1) {
					t.Fatal("toggle undo reverted the heading composition")
				}
				tester.Frame()
				if taskAccess(t, tester, "未完成任务：task").Checked || v.draft != draft || v.editing != -1 {
					t.Fatal("clean independent draft froze undone task preview or rebound its index")
				}
				if !buffer.Redo() || buffer.String() != want {
					t.Fatal("toggle redo failed")
				}
				tester.Frame()
				if !taskAccess(t, tester, "已完成任务：task").Checked || v.draft != draft {
					t.Fatal("clean independent draft froze redone task preview")
				}
				if !buffer.Undo() || !buffer.Undo() || buffer.String() != original {
					t.Fatal("composition/toggle undo did not restore original")
				}
			case "same-list":
				if !strings.Contains(buffer.String(), "[ ] changed") || strings.Contains(buffer.String(), "[x]") || changes != 1 || len(reports) == 0 {
					t.Fatal("same-list replacement silently retargeted", buffer.String(), reports)
				}
			case "failed-flush":
				if buffer.String() != authority || v.draft == nil || v.draft.Buffer.String() != "新标题🙂" || !errors.Is(v.draftError, errDraftConflict) || len(reports) == 0 {
					t.Fatal("failed flush toggled or lost recoverable composition", buffer.String(), reports)
				}
			}
		})
	}
}

func TestTaskVirtualFarRowsAndImmutableBuild(t *testing.T) {
	var fixture strings.Builder
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&fixture, "- [ ] Task %04d\n", i)
	}
	buffer := text.New(fixture.String())
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 265, 480)
	v.list.ScrollTo(900, ui.Start)
	tester.Frame()
	info := taskAccess(t, tester, "未完成任务：Task 0900")
	id := v.rows[900].id
	tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tester.Key(0, ui.KeySpace)
	if buffer.String() != fixture.String() || v.rows[900].id != id || v.frameRows > 60 || tester.FramePending() {
		t.Fatal("far task focus/virtualization/settling failed", v.frameRows)
	}
	measuredViewWork(t, "1000 tasks five unchanged native frames", func() {
		for i := 0; i < 5; i++ {
			tester.Frame()
		}
	})
	t.Logf("built rows=%d pending-frame=%v", v.frameRows, tester.FramePending())
	// A host notification is allowed to request warnings immediately. This
	// reparses during Changed while the current virtual List still has rows.
	v.host.Changed = func() { _ = v.Warnings() }
	info = taskAccess(t, tester, "未完成任务：Task 0900")
	tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	if !strings.Contains(buffer.String(), "- [x] Task 0900\n") || v.frameRows > 60 {
		t.Fatal("reentrant preview invalidation corrupted current list build")
	}
}

func TestTaskNativeDuplicateAfterInsertionAndRepeatedSpace(t *testing.T) {
	prefix := "- [ ] duplicate\n\n" + headingFixture(120) + "\n"
	original := prefix + "- [ ] duplicate\n\nend\n"
	buffer := text.New(original)
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 720, 480)
	v.list.ScrollTo(121, ui.Start)
	tester.Frame()
	info := taskAccess(t, tester, "未完成任务：duplicate")
	tester.Press(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	_ = buffer.ReplaceUTF16(0, 0, "- [ ] duplicate\n\n")
	tester.Frame()
	info = taskAccess(t, tester, "未完成任务：duplicate")
	tester.Release(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	want := "- [ ] duplicate\n\n" + prefix + "- [x] duplicate\n\nend\n"
	if buffer.String() != want {
		t.Fatal("actual held click retargeted duplicate after prefix insertion", buffer.String())
	}
	if !taskAccess(t, tester, "已完成任务：duplicate").Focused {
		t.Fatal("mapped duplicate target lost native focus")
	}
	tester.Key(0, ui.KeySpace)
	if buffer.String() != "- [ ] duplicate\n\n"+original || v.draft != nil {
		t.Fatal("mapped duplicate Space lost target")
	}
}

func TestTaskControlDoesNotEditLabelAndLargeLabelIsBounded(t *testing.T) {
	buffer := text.New("- [ ] **bold** task\n- [ ] " + strings.Repeat("🙂", maxPreviewBlockBytes) + "\n")
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 265, 480)
	source := buffer.String()
	label, ok := tester.Find("bold task")
	if !ok {
		t.Fatal("formatted task label missing", tester.Texts())
	}
	tester.ClickAt(label.X+label.W/2, label.Y+label.H/2)
	if buffer.String() != source || v.draft != nil {
		t.Fatal("single label selection toggled or edited task")
	}
	for _, displayed := range tester.Texts() {
		if len(displayed) > 512 {
			t.Fatal("large task label reached shaping unbounded", len(displayed))
		}
	}
	if !tester.HasText("较大列表项") {
		t.Fatal("large task missing full-source guidance")
	}
	info := taskAccess(t, tester, "未完成任务：**bold** task")
	tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	if buffer.String() != strings.Replace(source, "[ ]", "[x]", 1) {
		t.Fatal("formatted task native toggle changed label bytes")
	}
}

func TestTaskPartialHasNoActiveControlsAndRefusedFrameRestoresState(t *testing.T) {
	buffer := text.New("- [ ] task\n")
	var reported error
	v := Adapter().Create(editor.Host{Buffer: buffer, Report: func(err error) { reported = err }}).(*View)
	tester := ui.NewTester(v.Draw, 265, 480)
	_ = buffer.BeginGroup()
	info := taskAccess(t, tester, "未完成任务：task")
	tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	if !errors.Is(reported, text.ErrGroup) || buffer.String() != "- [ ] task\n" || !buffer.InGroup() || taskAccess(t, tester, "未完成任务：task").Checked {
		t.Fatal("failed actual toggle left false checked state or took group", reported)
	}
	buffer.EndGroup()
	_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), strings.Repeat("- x\n", 4200))
	tester.Frame()
	if _, found := tester.AccessByRole(ui.RoleCheckBox, ""); found || !tester.HasText("源码") || !v.doc.Partial {
		t.Fatal("partial document exposed active task controls")
	}
}

func TestTaskNarrowLightDarkReadonlyGeometry(t *testing.T) {
	for _, dark := range []bool{false, true} {
		for _, readonly := range []bool{false, true} {
			v := Adapter().Create(editor.Host{Buffer: text.New("# 任务清单\n\n- [ ] 整理 **本地** 笔记\n- [X] 已核对🙂\n- [ ]\n\n3) [ ] 下一步\n"), ReadOnly: readonly}).(*View)
			tester := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Width(265).Height(360).Children(func() {
					ui.Box(c).Height(330).Shrink(0).Children(func() { v.Draw(c) })
					ui.Text(c, "STATUS").Height(30).Shrink(0)
				})
			}, 265, 360)
			tester.SetDark(dark)
			if readonly && (!tester.HasText("只读预览") || tester.HasText("双击内容编辑")) {
				t.Fatal("readonly preview advertised editing")
			}
			for _, name := range []string{"未完成任务：整理 **本地** 笔记", "已完成任务：已核对🙂", "未完成任务：（空任务）", "未完成任务：下一步"} {
				info := taskAccess(t, tester, name)
				if info.Disabled != readonly || info.Bounds.X < 0 || info.Bounds.X+info.Bounds.W > 265 || info.Bounds.Y < 0 || info.Bounds.Y+info.Bounds.H > 330 {
					t.Fatal("narrow task exceeds bounded content", info)
				}
			}
			if dir := os.Getenv("LEVNOTE_MARKDOWN_SCREENSHOT_DIR"); dir != "" {
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				file, err := os.Create(filepath.Join(dir, fmt.Sprintf("markdown-tasks-narrow-dark-%v-readonly-%v.png", dark, readonly)))
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(file, tester.Image())
				_ = file.Close()
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
