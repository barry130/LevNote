package markdownview

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/ui/editor"
)

func alignmentButton(t *testing.T, tester *ui.Tester, column int, alignment markdown.TableAlignment) ui.AccessInfo {
	t.Helper()
	return rowButton(t, tester, tableAlignmentName(column, alignment))
}

func focusAlignment(t *testing.T, tester *ui.Tester, column int, alignment markdown.TableAlignment) {
	t.Helper()
	tester.Key(0, ui.KeyF6)
	if !rowButton(t, tester, "在下方插入行").Focused {
		t.Fatal("keyboard route must enter native row actions from the cell draft")
	}
	for step := 0; step < 9; step++ {
		if alignmentButton(t, tester, column, alignment).Focused {
			return
		}
		tester.Key(0, ui.KeyTab)
	}
	t.Fatal("Tab failed to reach selected-column alignment")
}

func TestTableAlignmentNativePointerKeyboardAndUndo(t *testing.T) {
	for _, mode := range []string{"pointer", "enter", "space"} {
		t.Run(mode, func(t *testing.T) {
			buffer, v, tester := rowView(t, rowActionSource, 760)
			changes := 0
			v.host.Changed = func() { changes++ }
			selection := text.Range{Anchor: buffer.LenUTF16(), Head: 2}
			_ = buffer.SetSelection(selection.Anchor, selection.Head)
			if mode == "pointer" {
				clickDraftControl(t, tester, "第1列居中对齐")
			} else {
				focusAlignment(t, tester, 0, markdown.TableAlignCenter)
				tester.Key(ui.Shift, ui.KeyTab)
				if !alignmentButton(t, tester, 0, markdown.TableAlignLeft).Focused {
					t.Fatal("ShiftTab did not traverse alignment controls")
				}
				tester.Key(0, ui.KeyTab)
				key := ui.KeyEnter
				if mode == "space" {
					key = ui.KeySpace
				}
				tester.KeyRepeat(0, key)
				if buffer.String() != rowActionSource || v.draft == nil {
					t.Fatal("repeated key performed alignment")
				}
				tester.Key(0, key)
			}
			want := strings.Replace(rowActionSource, "|---|---|", "|:---:|---|", 1)
			wantSelection := text.Range{Anchor: selection.Anchor + 2, Head: selection.Head}
			if buffer.String() != want || buffer.Selection() != wantSelection || changes != 1 ||
				buffer.History().UndoDepth != 1 || v.draft != nil || v.cellEdit != nil || v.tableActionError != nil {
				t.Fatalf("native alignment source=%q sel=%+v history=%+v err=%v", buffer.String(), buffer.Selection(), buffer.History(), v.tableActionError)
			}
			if !buffer.Undo() || buffer.String() != rowActionSource || buffer.Selection() != selection ||
				!buffer.Redo() || buffer.String() != want || buffer.Selection() != wantSelection {
				t.Fatal("alignment undo/redo did not restore exact source and oriented selection")
			}
		})
	}
}

func TestTableAlignmentNoopClosesPanelWithoutSecondEdit(t *testing.T) {
	for _, pending := range []bool{false, true} {
		t.Run(fmt.Sprint(pending), func(t *testing.T) {
			buffer, v, tester := rowView(t, rowActionSource, 760)
			changes := 0
			v.host.Changed = func() { changes++ }
			beforeVersion, beforeSelection := buffer.Version(), buffer.Selection()
			want, undo := rowActionSource, 0
			if pending {
				replaceDraftText(t, tester, "新🙂")
				want, undo = strings.Replace(rowActionSource, "| 甲 |", "| 新🙂 |", 1), 1
			}
			clickDraftControl(t, tester, "第1列默认对齐")
			if v.draft != nil || v.cellEdit != nil || buffer.String() != want || buffer.History().UndoDepth != undo || changes != undo {
				t.Fatalf("noop panel/history/source mismatch pending=%v changes=%d history=%+v", pending, changes, buffer.History())
			}
			if !pending && (buffer.Version() != beforeVersion || buffer.Selection() != beforeSelection) {
				t.Fatal("clean noop changed version or selection")
			}
			if pending && (!buffer.Undo() || buffer.String() != rowActionSource || buffer.History().UndoDepth != 0) {
				t.Fatal("noop added a hidden alignment history step")
			}
		})
	}
}

func TestTableAlignmentSelectedHeaderColumnReadonlyAndPriorOutsideEdit(t *testing.T) {
	buffer := text.New(rowActionSource)
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 760, 480)
	tester.Frame()
	openCellByPointer(t, tester, ui.RoleColumnHeader, "表头，第 1 行，第 2 列：B")
	draft := v.draft
	if !tester.HasText("表头 · 第2列 · 当前：默认") || !tester.HasText("F6 表格操作 · "+tableHeaderHint) {
		t.Fatal("header/current-column display is not literal")
	}
	v.host.ReadOnly = true
	tester.Frame()
	for _, name := range []string{"第2列默认对齐", "第2列左对齐", "第2列居中对齐", "第2列右对齐"} {
		button, ok := tester.AccessByRole(ui.RoleButton, name)
		if !ok || !button.Disabled || button.Focusable {
			t.Fatal("readonly alignment action is interactive", name, button)
		}
		tester.ClickAt(button.Bounds.X+4, button.Bounds.Y+4)
	}
	if buffer.String() != rowActionSource || buffer.History().UndoDepth != 0 || v.draft != draft {
		t.Fatal("readonly click changed source/draft")
	}
	v.host.ReadOnly = false
	_ = buffer.ReplaceUTF16(0, 0, "外🙂\r\n")
	tester.Frame()
	if !tester.HasText("表头 · 第2列 · 当前：待重验") {
		t.Fatal("stale descriptor advertised current alignment")
	}
	tester.Advance(time.Second)
	clickDraftControl(t, tester, "第2列右对齐")
	want := "外🙂\r\n" + strings.Replace(rowActionSource, "|---|---|", "|---|---:|", 1)
	if buffer.String() != want || v.draft != nil || buffer.History().UndoDepth != 2 || v.tableActionError != nil {
		t.Fatalf("outside edit rebase selected wrong column: %q %v", buffer.String(), v.tableActionError)
	}
}

func TestTableAlignmentCompositionGeometryAndSeparateUndos(t *testing.T) {
	buffer, v, tester := rowView(t, rowActionSource, 265)
	draft := v.draft
	replaceDraftText(t, tester, "新🙂")
	marked, composing := draft.MarkedRange()
	value, version, selection, history := draft.Buffer.String(), draft.Buffer.Version(), draft.Buffer.Selection(), draft.Buffer.History()
	tester.Key(0, ui.KeyF6)
	field, ok := tester.AccessByRole(ui.RoleTextField, v.cellEdit.label)
	afterMarked, afterComposing := draft.MarkedRange()
	if !composing || !ok || !field.Focused || !afterComposing || afterMarked != marked ||
		draft.Buffer.String() != value || draft.Buffer.Version() != version || draft.Buffer.Selection() != selection ||
		draft.Buffer.History() != history || !draft.Buffer.InGroup() || !tester.HasText(tableActionComposingHint) {
		t.Fatal("F6 changed the composing draft")
	}
	button := alignmentButton(t, tester, 0, markdown.TableAlignRight)
	x, y := button.Bounds.X+4, button.Bounds.Y+4
	tester.Press(x, y)
	current := alignmentButton(t, tester, 0, markdown.TableAlignRight)
	if current.Bounds != button.Bounds || !v.tableActionPress.active {
		t.Fatalf("pointer composition finish moved alignment: before=%+v after=%+v", button.Bounds, current.Bounds)
	}
	tester.Release(x, y)
	written := strings.Replace(rowActionSource, "| 甲 |", "| 新🙂 |", 1)
	want := strings.Replace(written, "|---|---|", "|---:|---|", 1)
	if buffer.String() != want || buffer.History().UndoDepth != 2 || v.draft != nil {
		t.Fatalf("composition alignment=%q history=%+v err=%v", buffer.String(), buffer.History(), v.tableActionError)
	}
	if !buffer.Undo() || buffer.String() != written || !buffer.Undo() || buffer.String() != rowActionSource ||
		!buffer.Redo() || buffer.String() != written || !buffer.Redo() || buffer.String() != want {
		t.Fatal("cell/alignment did not remain two exact undo/redo steps")
	}
}

func TestTableAlignmentNarrowBoundsAndRecovery(t *testing.T) {
	for _, width := range []float32{265, 760} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			buffer, v, tester := rowView(t, rowActionSource, width)
			if !tester.HasText("数据第2行 · 第1列 · 当前：默认") {
				t.Fatal("bounded column/alignment title missing", tester.Texts())
			}
			var rowY float32 = -1
			for _, alignment := range []markdown.TableAlignment{markdown.TableAlignDefault, markdown.TableAlignLeft, markdown.TableAlignCenter, markdown.TableAlignRight} {
				button := alignmentButton(t, tester, 0, alignment)
				if button.Bounds.X < 0 || button.Bounds.X+button.Bounds.W > width || button.Bounds.Y+button.Bounds.H > 451 || button.Bounds.W < 40 {
					t.Fatalf("alignment outside compact viewport: %+v", button)
				}
				if rowY >= 0 && button.Bounds.Y != rowY {
					t.Fatal("alignment buttons wrapped")
				}
				rowY = button.Bounds.Y
			}
			replaceDraftText(t, tester, "坏|值")
			draft, history := v.draft, buffer.History()
			clickDraftControl(t, tester, "第1列左对齐")
			if v.draft != draft || draft.Buffer.String() != "坏|值" || buffer.String() != rowActionSource || buffer.History() != history ||
				!errors.Is(v.draftError, errTableCellPipe) || v.tableActionError == nil || !strings.Contains(v.tableActionError.Error(), "对齐未修改") {
				t.Fatal("invalid cell alignment lost draft or truth", v.draftError, v.tableActionError)
			}
			for _, name := range []string{"完成编辑", "放弃草稿", "打开未同步副本", "在下方插入行", "删除此数据行", "第1列默认对齐", "第1列右对齐"} {
				button := rowButton(t, tester, name)
				if button.Bounds.X+button.Bounds.W > width || button.Bounds.Y+button.Bounds.H > 451 {
					t.Fatalf("refusal control below status or outside width: %s %+v", name, button.Bounds)
				}
			}
			recovered := ""
			v.host.RecoverDraft = func(_, value string) error { recovered = value; return nil }
			clickDraftControl(t, tester, "打开未同步副本")
			if recovered != "坏|值" || buffer.String() != rowActionSource {
				t.Fatal("recovery is not exact and reachable")
			}
		})
	}
}

func TestTableAlignmentPostCellLimitRefusal(t *testing.T) {
	source := "| H | B |\n|" + strings.Repeat("-", 4096) + "|---|\n| 甲 | 乙 |\n"
	buffer, v, tester := rowView(t, source, 760)
	replaceDraftText(t, tester, "新🙂")
	clickDraftControl(t, tester, "第1列居中对齐")
	want := strings.Replace(source, "| 甲 |", "| 新🙂 |", 1)
	if buffer.String() != want || buffer.History().UndoDepth != 1 || v.draft != nil || v.tableActionError == nil ||
		!strings.Contains(v.tableActionError.Error(), "单元格已写回，对齐未修改") || !strings.Contains(v.tableActionError.Error(), "上限") {
		t.Fatal("post-cell refusal did not preserve independent write", v.tableActionError)
	}
	if !buffer.Undo() || buffer.String() != source {
		t.Fatal("cell write could not be undone after alignment refusal")
	}
}

func TestTableAlignmentRapidContinuationAndHeldRepeat(t *testing.T) {
	t.Run("rapid-pointer", func(t *testing.T) {
		buffer := text.New(rowActionSource)
		v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
		tester := ui.NewTester(v.Draw, 760, 480)
		tester.Frame()
		initial := tableCellAccess(t, tester, ui.RoleCell, "数据，第 3 行，第 1 列：同")
		// The added column row moves alignment below the old top-quarter hit.
		// Keep the same cell's lower-quarter point and the physical overlap
		// assertion so continuation still targets the newly opened alignment.
		initialX, initialY := initial.Bounds.X+initial.Bounds.W/2, initial.Bounds.Y+initial.Bounds.H*3/4
		tester.ClickAt(initialX, initialY)
		tester.ClickAt(initialX, initialY)
		if v.draft == nil {
			t.Fatal("initial physical double click did not open cell")
		}
		focusAlignment(t, tester, 0, markdown.TableAlignDefault)
		tester.Key(0, ui.KeyEnter)
		if v.draft != nil || buffer.String() != rowActionSource || buffer.History().UndoDepth != 0 {
			t.Fatal("noop did not finish initial panel")
		}
		cell := tableCellAccess(t, tester, ui.RoleCell, "数据，第 3 行，第 1 列：同")
		x, y := cell.Bounds.X+cell.Bounds.W/2, cell.Bounds.Y+cell.Bounds.H*3/4
		tester.ClickAt(x, y) // no virtual-time advance: continuation reopens
		if v.draft == nil || buffer.String() != rowActionSource {
			t.Fatal("rapid continuation must actually reopen the selected cell")
		}
		button := alignmentButton(t, tester, 0, markdown.TableAlignRight)
		if x < button.Bounds.X || x > button.Bounds.X+button.Bounds.W || y < button.Bounds.Y || y > button.Bounds.Y+button.Bounds.H {
			t.Fatalf("rapid oracle must intersect alignment button: point=%v,%v bounds=%+v", x, y, button.Bounds)
		}
		draft, history, version := v.draft, buffer.History(), buffer.Version()
		tester.ClickAt(x, y)
		if buffer.String() != rowActionSource || buffer.History() != history || buffer.Version() != version || v.draft != draft || v.tableActionPress.active {
			t.Fatal("continuation clicked through the panel into alignment")
		}
		tester.Advance(time.Second)
		clickDraftControl(t, tester, "第1列右对齐")
		if buffer.String() != strings.Replace(rowActionSource, "|---|---|", "|---:|---|", 1) || buffer.History().UndoDepth != 1 || v.draft != nil {
			t.Fatal("fresh single click remained blocked")
		}
	})
	t.Run("held-key", func(t *testing.T) {
		buffer, v, tester := rowView(t, rowActionSource, 760)
		focusAlignment(t, tester, 0, markdown.TableAlignLeft)
		tester.KeyDown(ui.KeySpace)
		want := strings.Replace(rowActionSource, "|---|---|", "|:---|---|", 1)
		if buffer.String() != want || v.draft != nil || buffer.History().UndoDepth != 1 {
			t.Fatal("first key down failed")
		}
		tester.Advance(time.Second)
		openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
		draft := v.draft
		focusAlignment(t, tester, 0, markdown.TableAlignRight)
		tester.KeyRepeat(0, ui.KeySpace)
		tester.KeyUp(ui.KeySpace)
		if buffer.String() != want || v.draft != draft || buffer.History().UndoDepth != 1 {
			t.Fatal("held repeat retargeted a new cell panel")
		}
		tester.Key(0, ui.KeySpace)
		if buffer.String() != strings.Replace(rowActionSource, "|---|---|", "|---:|---|", 1) || v.draft != nil || buffer.History().UndoDepth != 2 {
			t.Fatal("new key press after release failed")
		}
	})
}

func TestTableAlignmentHeldSourceIdentityAndScroll(t *testing.T) {
	for _, mode := range []string{"source", "outside", "draft-selection", "replace-draft", "readonly", "foreign-group", "scroll", "cancel", "release"} {
		t.Run(mode, func(t *testing.T) {
			source := rowActionSource + strings.Repeat("\nparagraph\n", 80)
			buffer, v, tester := rowView(t, source, 760)
			button := alignmentButton(t, tester, 0, markdown.TableAlignLeft)
			x, y := button.Bounds.X+4, button.Bounds.Y+4
			tester.Press(x, y)
			if !v.tableActionPress.active {
				t.Fatal("real pointer did not capture alignment")
			}
			switch mode {
			case "source":
				start := text.UTF16Len(source[:strings.Index(source, "|---|")])
				_ = buffer.ReplaceUTF16(start+1, start+4, ":---")
			case "outside":
				_ = buffer.ReplaceUTF16(0, 0, "外部\n")
			case "draft-selection":
				previous := v.draft.Buffer.Selection()
				_ = v.draft.Buffer.SetSelection(0, 1)
				if v.draft.Buffer.Selection() == previous {
					t.Fatal("draft selection fixture did not change")
				}
			case "replace-draft":
				v.DiscardDraft()
				v.draft = editor.New(text.New("replacement"))
			case "readonly":
				v.host.ReadOnly = true
			case "foreign-group":
				_ = buffer.BeginGroup()
			case "scroll":
				before := v.scroll.Y
				tester.Scroll(400, 430, 0, 120)
				if v.scroll.Y <= before || v.tableActionPress.active {
					t.Fatal("alignment cancel swallowed actual preview scroll")
				}
			case "cancel":
				tester.CancelPress()
			case "release":
				v.Release()
			}
			value, version, history := buffer.String(), buffer.Version(), buffer.History()
			tester.Frame()
			tester.Release(x, y)
			if buffer.String() != value || buffer.Version() != version || buffer.History() != history || v.tableActionPress.active {
				t.Fatalf("held alignment followed changed target: %s %v", mode, v.tableActionError)
			}
			if mode == "foreign-group" && (!buffer.InGroup() || !buffer.CancelGroup()) {
				t.Fatal("foreign group was closed")
			}
		})
	}
}

func TestTableAlignmentChangedAndReportOwnership(t *testing.T) {
	for _, moment := range []string{"cell-changed", "alignment-changed", "report"} {
		for _, replacement := range []string{"release", "buffer", "draft"} {
			t.Run(moment+"/"+replacement, func(t *testing.T) {
				buffer, v, tester := rowView(t, rowActionSource, 760)
				replacementError := errors.New("replacement issue")
				var replacementDraft *editor.Editor
				var replacementBuffer *text.Buffer
				var releasedDraft *editor.Editor
				callbackRan, reports := false, ""
				replace := func() {
					callbackRan = true
					if replacement == "release" {
						v.Release()
						releasedDraft = v.draft // Release retains an invalid cell draft for recovery.
						return
					}
					v.clearDraft()
					if replacement == "buffer" {
						replacementBuffer = text.New("new source")
						v.host.Buffer = replacementBuffer
					} else {
						replacementDraft = editor.New(text.New("new draft"))
						v.draft = replacementDraft
					}
					v.tableActionError = replacementError
				}
				v.host.Report = func(err error) {
					reports += err.Error() + "\n"
					if moment == "report" && !callbackRan {
						replace()
					}
				}
				changes := 0
				v.host.Changed = func() {
					changes++
					if moment == "cell-changed" || moment == "alignment-changed" && changes == 2 {
						replace()
					}
				}
				value := "新🙂"
				if moment == "report" {
					value = "坏|值"
				}
				replaceDraftText(t, tester, value)
				clickDraftControl(t, tester, "第1列左对齐")
				want, depth := rowActionSource, 0
				if moment != "report" {
					want, depth = strings.Replace(want, "| 甲 |", "| 新🙂 |", 1), 1
				}
				if moment == "alignment-changed" {
					want, depth = strings.Replace(want, "|---|---|", "|:---|---|", 1), 2
				}
				if !callbackRan || buffer.String() != want || buffer.History().UndoDepth != depth {
					t.Fatalf("callback action state mismatch ran=%v depth=%+v err=%v", callbackRan, buffer.History(), v.tableActionError)
				}
				if replacement == "release" {
					if v.draft != releasedDraft || v.tableActionError != nil {
						t.Fatal("old action resurrected released state")
					}
				} else if v.tableActionError != replacementError || replacementDraft != nil && v.draft != replacementDraft ||
					replacementBuffer != nil && v.host.Buffer != replacementBuffer {
					t.Fatal("old action overwrote replacement state")
				}
				if moment == "cell-changed" && !strings.Contains(reports, "单元格已写回，对齐未修改") {
					t.Fatal("lost truthful host refusal", reports)
				}
			})
		}
	}
}

func TestTableAlignmentReportInPlaceChangesDoNotReceiveOldLocalError(t *testing.T) {
	for _, change := range []string{"draft-text", "draft-selection", "source-text", "source-selection"} {
		t.Run(change, func(t *testing.T) {
			buffer, v, tester := rowView(t, rowActionSource, 760)
			replaceDraftText(t, tester, "坏|值")
			draft, callbackRan := v.draft, false
			replacementError := errors.New("callback-local issue")
			v.host.Report = func(err error) {
				if callbackRan {
					return
				}
				callbackRan = true
				switch change {
				case "draft-text":
					_ = draft.Buffer.ReplaceUTF16(0, draft.Buffer.LenUTF16(), "回调保留")
				case "draft-selection":
					_ = draft.Buffer.SetSelection(0, 1)
				case "source-text":
					_ = buffer.ReplaceUTF16(0, 0, "回调\n")
				case "source-selection":
					_ = buffer.SetSelection(0, 1)
				}
				v.tableActionError = replacementError
			}
			clickDraftControl(t, tester, "第1列左对齐")
			if !callbackRan || v.draft != draft || v.tableActionError != replacementError {
				t.Fatal("stale refusal overwrote an in-place callback edit", v.tableActionError)
			}
			if change == "draft-text" && draft.Buffer.String() != "回调保留" ||
				change == "draft-selection" && draft.Buffer.Selection() != (text.Range{Anchor: 0, Head: 1}) ||
				change == "source-text" && buffer.String() != "回调\n"+rowActionSource ||
				change == "source-selection" && buffer.Selection() != (text.Range{Anchor: 0, Head: 1}) {
				t.Fatal("callback content or selection was restored")
			}
		})
	}
}

func TestTableAlignmentSelectionAndIdle(t *testing.T) {
	for _, kind := range []string{"before", "inside", "after", "reverse-astral"} {
		t.Run(kind, func(t *testing.T) {
			buffer, _, tester := rowView(t, rowActionSource, 760)
			start := text.UTF16Len(rowActionSource[:strings.Index(rowActionSource, "---")])
			selection, want := text.Range{Anchor: 0, Head: 2}, text.Range{Anchor: 0, Head: 2}
			switch kind {
			case "inside":
				selection, want = text.Range{Anchor: start + 1, Head: start + 2}, text.Range{Anchor: start, Head: start}
			case "after":
				selection = text.Range{Anchor: start + 3, Head: buffer.LenUTF16()}
				want = text.Range{Anchor: selection.Anchor + 2, Head: selection.Head + 2}
			case "reverse-astral":
				selection = text.Range{Anchor: text.UTF16Len(rowActionSource[:len(rowActionSource)-2]), Head: text.UTF16Len("# 前")}
				want = text.Range{Anchor: selection.Anchor + 2, Head: selection.Head}
			}
			if err := buffer.SetSelection(selection.Anchor, selection.Head); err != nil {
				t.Fatal(err)
			}
			clickDraftControl(t, tester, "第1列居中对齐")
			if buffer.Selection() != want {
				t.Fatalf("UTF16 endpoint mapping %s=%+v want=%+v", kind, buffer.Selection(), want)
			}
			if !buffer.Undo() || buffer.Selection() != selection {
				t.Fatal("undo did not restore legal oriented selection", buffer.Selection(), selection)
			}
		})
	}
	for _, state := range []string{"draft", "success", "noop", "refused"} {
		t.Run("idle/"+state, func(t *testing.T) {
			buffer, v, tester := rowView(t, rowActionSource, 760)
			switch state {
			case "success":
				clickDraftControl(t, tester, "第1列右对齐")
			case "noop":
				clickDraftControl(t, tester, "第1列默认对齐")
			case "refused":
				replaceDraftText(t, tester, "坏|值")
				clickDraftControl(t, tester, "第1列右对齐")
			}
			tester.Advance(time.Second)
			tester.Frame()
			block, row, draft := &v.doc.Blocks[0], &v.rows[0], v.draft
			version, history := buffer.Version(), buffer.History()
			for frame := 0; frame < 5; frame++ {
				tester.Frame()
				if tester.FramePending() || &v.doc.Blocks[0] != block || &v.rows[0] != row || v.draft != draft ||
					v.tableActionPress != (tableActionGesture{}) || buffer.Version() != version || buffer.History() != history {
					t.Fatal("idle alignment frame retained capture, invalidated or reparsed")
				}
			}
		})
	}
}
