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

const rowActionSource = "# 前🙂\r\n\r\n| H | B |\r\n|---|---|\r\n| 甲 | 乙 |\r\n| 同 | 值 |\r\n\r\n尾🙂\r\n"

func rowView(t *testing.T, source string, width float32) (*text.Buffer, *View, *ui.Tester) {
	t.Helper()
	buffer := text.New(source)
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, int(width), 480)
	tester.Frame()
	openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
	return buffer, v, tester
}

func rowButton(t *testing.T, tester *ui.Tester, label string) ui.AccessInfo {
	t.Helper()
	button, ok := tester.AccessByRole(ui.RoleButton, label)
	if !ok || button.Disabled || !button.Focusable || button.Offscreen || button.Bounds.W <= 0 || button.Bounds.H <= 0 {
		t.Fatalf("row button %q unavailable: %+v texts=%q", label, button, tester.Texts())
	}
	return button
}

func TestTableRowsNativePointerEnterSpaceAndUndo(t *testing.T) {
	for _, mode := range []string{"pointer", "enter", "space", "key-repeat"} {
		t.Run(mode, func(t *testing.T) {
			buffer, v, tester := rowView(t, rowActionSource, 760)
			changes := 0
			v.host.Changed = func() { changes++ }
			selection := text.Range{Anchor: buffer.LenUTF16(), Head: 2}
			_ = buffer.SetSelection(selection.Anchor, selection.Head)
			button := rowButton(t, tester, "在下方插入行")
			if mode == "pointer" {
				tester.ClickAt(button.Bounds.X+4, button.Bounds.Y+4)
			} else {
				field, ok := tester.AccessByRole(ui.RoleTextField, v.cellEdit.label)
				if !ok || !field.Focused {
					t.Fatal("keyboard route must start inside the cell draft", field)
				}
				tester.Key(0, ui.KeyF6)
				if !rowButton(t, tester, "在下方插入行").Focused || v.draft.Buffer.String() != "甲" {
					t.Fatal("F6 did not move focus without editing the draft")
				}
				tester.Key(0, ui.KeyTab)
				if !rowButton(t, tester, "删除此数据行").Focused {
					t.Fatal("Tab did not reach the second row control")
				}
				tester.Key(ui.Shift, ui.KeyTab)
				if !rowButton(t, tester, "在下方插入行").Focused {
					t.Fatal("Shift+Tab did not return to the first row control")
				}
				key := ui.KeyEnter
				if mode == "space" {
					key = ui.KeySpace
				}
				if mode == "key-repeat" {
					tester.KeyRepeat(0, key)
					if buffer.String() != rowActionSource || v.draft == nil {
						t.Fatal("repeated key activated a row action")
					}
				}
				tester.Key(0, key)
			}
			tester.Frame()
			want := strings.Replace(rowActionSource, "| 甲 | 乙 |\r\n", "| 甲 | 乙 |\r\n| | |\r\n", 1)
			wantSelection := text.Range{Anchor: selection.Anchor + 7, Head: selection.Head}
			if buffer.String() != want || buffer.Selection() != wantSelection || changes != 1 ||
				buffer.History().UndoDepth != 1 || v.draft != nil || v.tableActionError != nil {
				t.Fatalf("native once source=%q sel=%+v changes=%d undo=%+v error=%v", buffer.String(), buffer.Selection(), changes, buffer.History(), v.tableActionError)
			}
			if !buffer.Undo() || buffer.String() != rowActionSource || buffer.Selection() != selection ||
				!buffer.Redo() || buffer.String() != want || buffer.Selection() != wantSelection {
				t.Fatal("single row transaction did not restore text and reversed selection")
			}
		})
	}
}

func TestTableRowsHeaderAndNarrowControls(t *testing.T) {
	for _, width := range []float32{760, 265} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			buffer := text.New("| H | B |\r\n|---|---|")
			v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
			tester := ui.NewTester(v.Draw, int(width), 480)
			tester.Frame()
			openCellByPointer(t, tester, ui.RoleColumnHeader, "表头，第 1 行，第 1 列：H")
			button := rowButton(t, tester, "在下方插入行")
			if button.Bounds.X < 0 || button.Bounds.X+button.Bounds.W > width || button.Bounds.Y+button.Bounds.H > 480 {
				t.Fatal("insert outside narrow viewport", button)
			}
			remove, ok := tester.AccessByRole(ui.RoleButton, "删除此数据行")
			if !ok || !remove.Disabled || !tester.HasText("F6 表格操作 · "+tableHeaderHint) {
				t.Fatal("header deletion is not visibly disabled")
			}
			clickDraftControl(t, tester, "在下方插入行")
			if buffer.String() != "| H | B |\r\n|---|---|\r\n| | |" || buffer.History().UndoDepth != 1 {
				t.Fatalf("header insert at EOF=%q", buffer.String())
			}
		})
	}
}

func TestTableRowsPendingCompositionFlushAndTwoUndos(t *testing.T) {
	for _, action := range []string{"在下方插入行", "删除此数据行"} {
		t.Run(action, func(t *testing.T) {
			buffer, v, tester := rowView(t, rowActionSource, 760)
			_ = buffer.SetSelection(2, 0)
			beforeSelection := buffer.Selection()
			changes := 0
			v.host.Changed = func() { changes++ }
			draft := v.draft
			draft.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "新🙂", ui.TextInputRange{Start: 3, End: 3})
			if _, composing := draft.MarkedRange(); !composing || buffer.String() != rowActionSource {
				t.Fatal("composition fixture already committed")
			}
			tester.Frame()
			button := rowButton(t, tester, action)
			x, y := button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2
			tester.Press(x, y)
			if current := rowButton(t, tester, action); current.Bounds != button.Bounds || !v.tableActionPress.active {
				t.Fatalf("composition confirmation moved the pressed control: before=%+v after=%+v active=%v", button.Bounds, current.Bounds, v.tableActionPress.active)
			}
			tester.Release(x, y)
			tester.Frame()
			cellOnly := strings.Replace(rowActionSource, "| 甲 |", "| 新🙂 |", 1)
			want := strings.Replace(cellOnly, "| 新🙂 | 乙 |\r\n", "| 新🙂 | 乙 |\r\n| | |\r\n", 1)
			caret := text.UTF16Len(cellOnly[:strings.Index(cellOnly, "新🙂")]) + 3
			wantCaret := caret
			if action == "删除此数据行" {
				want = strings.Replace(cellOnly, "| 新🙂 | 乙 |\r\n", "", 1)
				wantCaret = text.UTF16Len(cellOnly[:strings.Index(cellOnly, "| 新🙂 |")])
			}
			if buffer.String() != want || buffer.Selection() != (text.Range{Anchor: wantCaret, Head: wantCaret}) ||
				buffer.History().UndoDepth != 2 || changes != 2 || v.draft != nil || v.tableActionError != nil {
				t.Fatalf("composition row source=%q sel=%+v history=%+v changes=%d error=%v", buffer.String(), buffer.Selection(), buffer.History(), changes, v.tableActionError)
			}
			if !buffer.Undo() || buffer.String() != cellOnly || buffer.Selection() != (text.Range{Anchor: caret, Head: caret}) {
				t.Fatal("first undo must restore row and post-cell selection")
			}
			if !buffer.Undo() || buffer.String() != rowActionSource || buffer.Selection() != beforeSelection {
				t.Fatal("second undo must restore original cell and selection")
			}
		})
	}
}

func TestTableRowsF6PreservesCompositionAndHeldKeys(t *testing.T) {
	t.Run("composition-focus-only", func(t *testing.T) {
		buffer, v, tester := rowView(t, rowActionSource, 760)
		draft := v.draft
		draft.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "新🙂", ui.TextInputRange{Start: 3, End: 3})
		tester.Frame()
		marked, composing := draft.MarkedRange()
		value, version, selection, history := draft.Buffer.String(), draft.Buffer.Version(), draft.Buffer.Selection(), draft.Buffer.History()
		if !composing || !draft.Buffer.InGroup() {
			t.Fatal("expected a live composition group before F6")
		}
		tester.Key(0, ui.KeyF6)
		currentMarked, currentComposing := draft.MarkedRange()
		field, ok := tester.AccessByRole(ui.RoleTextField, v.cellEdit.label)
		if !ok || !field.Focused || !tester.HasText(tableActionComposingHint) || v.draft != draft ||
			draft.Buffer.String() != value || draft.Buffer.Version() != version || draft.Buffer.Selection() != selection ||
			draft.Buffer.History() != history || !draft.Buffer.InGroup() || currentMarked != marked || !currentComposing ||
			buffer.String() != rowActionSource || buffer.History().UndoDepth != 0 || v.tableActionPress.active {
			t.Fatal("F6 did not keep the live composition and focus intact")
		}
		draft.UnmarkText() // normal native-client confirmation, no host Flush
		tester.Frame()
		tester.Key(0, ui.KeyF6)
		if !rowButton(t, tester, "在下方插入行").Focused || draft.Buffer.String() != value ||
			draft.Buffer.Version() != version || draft.Buffer.Selection() != selection ||
			buffer.String() != rowActionSource || buffer.History().UndoDepth != 0 || v.tableActionPress.active {
			t.Fatal("F6 after composition confirmation changed source or draft")
		}
	})
	for _, key := range []ui.Key{ui.KeyEnter, ui.KeySpace} {
		t.Run(fmt.Sprint(key), func(t *testing.T) {
			buffer, v, tester := rowView(t, rowActionSource, 760)
			tester.Key(0, ui.KeyF6)
			tester.KeyDown(key) // MyGo activates buttons on key down.
			want := strings.Replace(rowActionSource, "| 甲 | 乙 |\r\n", "| 甲 | 乙 |\r\n| | |\r\n", 1)
			if buffer.String() != want || v.draft != nil || v.tableActionPress.active {
				t.Fatal("first native key down did not finish exactly one row action")
			}
			// Start a distinct physical double-click. The prior opening's click
			// count otherwise lets the first press reopen, so the second press
			// can legitimately hit the compact panel now occupying that point.
			tester.Advance(time.Second)
			openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
			draft := v.draft
			if draft == nil || buffer.String() != want || buffer.History().UndoDepth != 1 {
				t.Fatalf("replacement draft fixture changed source: %q history=%+v draft=%v", buffer.String(), buffer.History(), draft != nil)
			}
			tester.Key(0, ui.KeyF6)
			if !rowButton(t, tester, "在下方插入行").Focused {
				t.Fatalf("replacement draft did not focus row control: texts=%q", tester.Texts())
			}
			tester.KeyRepeat(0, key)
			tester.KeyUp(key)
			if buffer.String() != want || v.draft != draft || buffer.History().UndoDepth != 1 || v.tableActionPress.active {
				t.Fatalf("held key repeated or followed the replacement draft: source=%q history=%+v sameDraft=%v press=%+v", buffer.String(), buffer.History(), v.draft == draft, v.tableActionPress)
			}
			tester.Key(0, key)
			want = strings.Replace(want, "| 甲 | 乙 |\r\n", "| 甲 | 乙 |\r\n| | |\r\n", 1)
			if buffer.String() != want || buffer.History().UndoDepth != 2 || v.draft != nil || v.tableActionPress.active {
				t.Fatal("fresh key after release did not activate once")
			}
		})
	}
}

func TestTableRowsRapidClickContinuationCannotDeleteReopenedCell(t *testing.T) {
	buffer := text.New(rowActionSource)
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 760, 480)
	tester.Frame()
	initial := tableCellAccess(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
	initialX, initialY := initial.Bounds.X+initial.Bounds.W/4, initial.Bounds.Y+initial.Bounds.H/2
	tester.ClickAt(initialX, initialY)
	tester.ClickAt(initialX, initialY)
	if v.draft == nil {
		t.Fatal("initial physical double click did not open cell")
	}
	tester.Key(0, ui.KeyF6)
	tester.Key(0, ui.KeyEnter)
	want := strings.Replace(rowActionSource, "| 甲 | 乙 |\r\n", "| 甲 | 乙 |\r\n| | |\r\n", 1)
	if buffer.String() != want || buffer.History().UndoDepth != 1 || v.draft != nil {
		t.Fatal("rapid-click fixture did not complete its initial row insertion")
	}
	cell := tableCellAccess(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
	// The compact face puts delete in the left quarter of this cell's old
	// rectangle. The intersection assertion below keeps the oracle physical.
	x, y := cell.Bounds.X+cell.Bounds.W/4, cell.Bounds.Y+cell.Bounds.H/2
	// No virtual time advance: the native click counter continues the earlier
	// opening. This first press already opens the compact panel at this point.
	tester.ClickAt(x, y)
	if v.draft == nil || buffer.String() != want || buffer.History().UndoDepth != 1 {
		t.Fatal("rapid continuation did not reopen without changing the source")
	}
	draft, version, history := v.draft, buffer.Version(), buffer.History()
	button := rowButton(t, tester, "删除此数据行")
	if x < button.Bounds.X || x > button.Bounds.X+button.Bounds.W || y < button.Bounds.Y || y > button.Bounds.Y+button.Bounds.H {
		t.Fatalf("regression must hit the replacement delete button: point=%v,%v button=%+v", x, y, button.Bounds)
	}
	tester.ClickAt(x, y)
	if buffer.String() != want || buffer.Version() != version || buffer.History() != history || v.draft != draft || v.tableActionPress.active {
		t.Fatalf("multi-click continuation deleted a row through the new panel: source=%q history=%+v", buffer.String(), buffer.History())
	}
	// Only the later deliberate single-click advances the virtual clock. It
	// proves the guard does not leave the live row action permanently blocked.
	tester.Advance(time.Second)
	button = rowButton(t, tester, "删除此数据行")
	tester.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
	if buffer.String() != strings.Replace(want, "| 甲 | 乙 |\r\n", "", 1) || buffer.History().UndoDepth != 2 || v.draft != nil {
		t.Fatal("fresh single-click was blocked after the canceled continuation")
	}
}

func TestTableRowsSelectionMappingAndExactDuplicateRow(t *testing.T) {
	source := "前🙂\n\n| H |\n|---|\n| 同🙂 |\n| 同🙂 |\n\n后🙂\n"
	for _, selectionKind := range []string{"before", "after", "inside", "reverse-astral", "insert-boundary"} {
		t.Run(selectionKind, func(t *testing.T) {
			buffer := text.New(source)
			v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
			v.refresh()
			doc := markdown.Parse(source)
			table := doc.Blocks[1]
			row := v.previewTableRow(1, 2)
			preview := previewSnapshot{v.doc, v.rows, buffer, buffer.Version()}
			target, ok := tableCellTargetOf(preview, row, table, 0)
			if !ok {
				t.Fatal("second equal row identity unavailable")
			}
			target, ok = captureTableCellTarget(preview, target)
			if !ok || v.activateTableCell(target) != nil {
				t.Fatal("cannot open second equal row")
			}
			start := text.UTF16Len(source[:strings.LastIndex(source, "| 同🙂 |")])
			end := start + text.UTF16Len("| 同🙂 |\n")
			selection, want := text.Range{Anchor: 0, Head: 3}, text.Range{Anchor: 0, Head: 3}
			action := markdown.TableRowDelete
			switch selectionKind {
			case "after":
				selection = text.Range{Anchor: buffer.LenUTF16(), Head: end}
				want = text.Range{Anchor: selection.Anchor - (end - start), Head: start}
			case "inside":
				selection = text.Range{Anchor: start + 2, Head: start + 5}
				want = text.Range{Anchor: start, Head: start}
			case "reverse-astral":
				selection = text.Range{Anchor: buffer.LenUTF16() - 1, Head: 3}
				want = text.Range{Anchor: selection.Anchor - (end - start), Head: 3}
			case "insert-boundary":
				action = markdown.TableRowInsertAfter
				selection = text.Range{Anchor: end, Head: end}
				want = text.Range{Anchor: end + 4, Head: end + 4}
			}
			if err := buffer.SetSelection(selection.Anchor, selection.Head); err != nil {
				t.Fatal(err)
			}
			g := v.captureTableRow(action, v.cellEdit, v.draft)
			if err := v.activateTableRow(g); err != nil {
				t.Fatal(err)
			}
			wantSource := "前🙂\n\n| H |\n|---|\n| 同🙂 |\n\n后🙂\n"
			if action == markdown.TableRowInsertAfter {
				wantSource = "前🙂\n\n| H |\n|---|\n| 同🙂 |\n| 同🙂 |\n| |\n\n后🙂\n"
			}
			if buffer.String() != wantSource || buffer.Selection() != want || buffer.History().UndoDepth != 1 {
				t.Fatalf("exact target or selection lost: %q %+v want%+v", buffer.String(), buffer.Selection(), want)
			}
			if !buffer.Undo() || buffer.String() != source || buffer.Selection() != selection {
				t.Fatal("undo failed oriented selection")
			}
		})
	}
}

func TestTableRowsInvalidDraftRecoveryAndWrittenButRefused(t *testing.T) {
	t.Run("invalid-retained", func(t *testing.T) {
		buffer, v, tester := rowView(t, rowActionSource, 265)
		recovered := ""
		v.host.RecoverDraft = func(_, value string) error { recovered = value; return nil }
		replaceDraftText(t, tester, "坏|值")
		draft, sel, history := v.draft, v.draft.Buffer.Selection(), buffer.History()
		clickDraftControl(t, tester, "删除此数据行")
		if v.draft != draft || draft.Buffer.String() != "坏|值" || draft.Buffer.Selection() != sel ||
			buffer.String() != rowActionSource || buffer.History() != history || !errors.Is(v.draftError, errTableCellPipe) || v.tableActionError == nil {
			t.Fatal("invalid row request lost pending draft or changed host", v.draftError, v.tableActionError)
		}
		clickDraftControl(t, tester, "打开未同步副本")
		if recovered != "坏|值" {
			t.Fatal("recovery did not retain literal invalid draft", recovered)
		}
	})
	t.Run("cell-written-row-limit", func(t *testing.T) {
		source := "| H | B |\n|---|---|\n| 甲 | 乙 |\n" + strings.Repeat("| x | y |\n", 255)
		buffer, v, tester := rowView(t, source, 760)
		replaceDraftText(t, tester, "新🙂")
		clickDraftControl(t, tester, "在下方插入行")
		want := strings.Replace(source, "| 甲 |", "| 新🙂 |", 1)
		if buffer.String() != want || buffer.History().UndoDepth != 1 || v.draft != nil || v.tableActionError == nil ||
			!strings.Contains(v.tableActionError.Error(), "单元格已写回，行未修改") || !strings.Contains(v.tableActionError.Error(), "上限") {
			t.Fatal("post-Flush refusal misreported or changed row", v.tableActionError)
		}
		if !buffer.Undo() || buffer.String() != source {
			t.Fatal("successful cell write not separately undoable")
		}
	})
}

func TestTableRowsHeldGestureSourceDraftSelectionAndCancel(t *testing.T) {
	for _, change := range []string{"source-same-table", "outside-during-press", "host-selection", "draft-text", "draft-selection", "replacement", "discard-new-draft", "readonly", "foreign-group", "cancel", "scroll", "release"} {
		t.Run(change, func(t *testing.T) {
			buffer, v, tester := rowView(t, rowActionSource, 760)
			button := rowButton(t, tester, "删除此数据行")
			x, y := button.Bounds.X+4, button.Bounds.Y+4
			tester.Press(x, y)
			if !v.tableActionPress.active {
				t.Fatal("native pointer down did not capture")
			}
			switch change {
			case "source-same-table":
				replaceTableBufferText(t, buffer, "乙", "外")
			case "outside-during-press":
				_ = buffer.ReplaceUTF16(0, 0, "外\n")
			case "host-selection":
				_ = buffer.SetSelection(1, 1)
			case "draft-text":
				_ = v.draft.Buffer.ReplaceUTF16(0, 1, "改")
			case "draft-selection":
				before := v.draft.Buffer.Selection()
				point := 1
				if before == (text.Range{Anchor: 1, Head: 1}) {
					point = 0
				}
				if err := v.draft.Buffer.SetSelection(point, point); err != nil || v.draft.Buffer.Selection() == before {
					t.Fatal("held draft selection fixture did not change", err)
				}
			case "replacement":
				v.host.Buffer = text.New(rowActionSource)
			case "discard-new-draft":
				v.DiscardDraft()
				v.refresh()
			case "readonly":
				v.host.ReadOnly = true
			case "foreign-group":
				_ = buffer.BeginGroup()
			case "cancel":
				tester.CancelPress()
			case "scroll":
				tester.Scroll(x, y, 0, 80)
			case "release":
				v.Release()
			}
			value, version, history := buffer.String(), buffer.Version(), buffer.History()
			tester.Frame()
			tester.Release(x, y)
			tester.Frame()
			if buffer.String() != value || buffer.Version() != version || buffer.History() != history || v.tableActionPress.active {
				t.Fatalf("held action followed changed target: %q %v", buffer.String(), v.tableActionError)
			}
			if change == "foreign-group" {
				if !buffer.InGroup() || !buffer.CancelGroup() {
					t.Fatal("row action closed foreign group")
				}
			}
		})
	}
}

func TestTableRowsOwnFlushCallbackInvalidation(t *testing.T) {
	for _, mutation := range []string{"selection", "outside-text", "same-text-undo", "release", "replacement", "new-draft", "readonly", "foreign-group"} {
		t.Run(mutation, func(t *testing.T) {
			buffer, v, tester := rowView(t, rowActionSource, 760)
			replaceDraftText(t, tester, "新🙂")
			callbackValue := ""
			callbackSelection := text.Range{}
			var reported []string
			v.host.Report = func(err error) { reported = append(reported, err.Error()) }
			replacementError := errors.New("replacement UI keeps its own issue")
			var replacementDraft *editor.Editor
			v.host.Changed = func() {
				v.host.Changed = nil
				switch mutation {
				case "selection":
					_ = buffer.SetSelection(0, 0)
				case "outside-text":
					_ = buffer.ReplaceUTF16(0, 0, "回调\n")
				case "same-text-undo":
					_ = buffer.ReplaceUTF16(0, 0, "临时")
					buffer.Undo()
				case "release":
					v.Release()
				case "replacement":
					v.host.Buffer = text.New(buffer.String())
					v.tableActionError = replacementError
				case "new-draft":
					v.draft = editor.New(text.New("另一个草稿"))
					replacementDraft = v.draft
					v.tableActionError = replacementError
				case "readonly":
					v.host.ReadOnly = true
				case "foreign-group":
					_ = buffer.BeginGroup()
				}
				callbackValue, callbackSelection = buffer.String(), buffer.Selection()
			}
			clickDraftControl(t, tester, "删除此数据行")
			if buffer.String() != callbackValue || buffer.Selection() != callbackSelection ||
				!strings.Contains(strings.Join(reported, "\n"), "单元格已写回，行未修改") {
				t.Fatalf("row committed after callback invalidation: %v", v.tableActionError)
			}
			switch mutation {
			case "release":
				if v.tableActionError != nil || v.draft != nil {
					t.Fatal("refusal resurrected released UI")
				}
			case "replacement", "new-draft":
				if v.tableActionError != replacementError || replacementDraft != nil && v.draft != replacementDraft {
					t.Fatal("old refusal overwrote replacement UI")
				}
			case "selection", "outside-text", "same-text-undo":
				if v.tableActionError != nil {
					t.Fatal("old refusal was attached to a callback-modified source")
				}
			default:
				if v.tableActionError == nil || !strings.Contains(v.tableActionError.Error(), "单元格已写回，行未修改") {
					t.Fatal("current owner lost visible refusal")
				}
			}
			if mutation == "foreign-group" {
				if !buffer.InGroup() || !buffer.CancelGroup() {
					t.Fatal("callback's group was closed")
				}
			}
		})
	}
}

func TestTableRowsOutsideEditBeforeGestureAndHistoryExpiry(t *testing.T) {
	for _, expired := range []bool{false, true} {
		t.Run(fmt.Sprint(expired), func(t *testing.T) {
			buffer, v, _ := rowView(t, rowActionSource, 760)
			count := 1
			if expired {
				count = 65
			}
			for i := 0; i < count; i++ {
				_ = buffer.ReplaceUTF16(0, 0, "外\n")
			}
			before, history := buffer.String(), buffer.History()
			g := v.captureTableRow(markdown.TableRowDelete, v.cellEdit, v.draft)
			err := v.activateTableRow(g)
			if expired {
				if err == nil || buffer.String() != before || buffer.History() != history || v.draft == nil {
					t.Fatal("expired mapping modified source or lost draft")
				}
			} else if err != nil || buffer.String() != strings.Replace(before, "| 甲 | 乙 |\r\n", "", 1) || buffer.History().UndoDepth != history.UndoDepth+1 {
				t.Fatal("exact outside-source rebase refused", err)
			}
		})
	}
}

func TestTableRowsSettledFramesKeepPreviewAndReleaseCapture(t *testing.T) {
	for _, state := range []string{"draft", "refused", "success", "canceled"} {
		t.Run(state, func(t *testing.T) {
			buffer, v, tester := rowView(t, rowActionSource, 760)
			switch state {
			case "refused":
				replaceDraftText(t, tester, "坏|值")
				clickDraftControl(t, tester, "删除此数据行")
			case "success":
				clickDraftControl(t, tester, "在下方插入行")
			case "canceled":
				button := rowButton(t, tester, "删除此数据行")
				tester.Press(button.Bounds.X+4, button.Bounds.Y+4)
				tester.CancelPress()
			}
			tester.Frame()
			if len(v.doc.Blocks) == 0 || len(v.rows) == 0 {
				t.Fatal("idle fixture has no parsed preview")
			}
			block, row, draft := &v.doc.Blocks[0], &v.rows[0], v.draft
			version, history := buffer.Version(), buffer.History()
			for frame := 0; frame < 5; frame++ {
				tester.Frame()
				if tester.FramePending() || v.tableActionPress != (tableActionGesture{}) ||
					&v.doc.Blocks[0] != block || &v.rows[0] != row || v.draft != draft ||
					buffer.Version() != version || buffer.History() != history {
					t.Fatalf("unchanged frame %d scheduled work, retained capture or rebuilt preview", frame)
				}
			}
		})
	}
}

func TestTableRowsNativeScrollCancelsWithoutTakingPreviewScroll(t *testing.T) {
	source := rowActionSource + strings.Repeat("\n# 后续内容\n", 80)
	buffer, v, tester := rowView(t, source, 760)
	button := rowButton(t, tester, "删除此数据行")
	x, y := button.Bounds.X+4, button.Bounds.Y+4
	tester.Press(x, y)
	if !v.tableActionPress.active || v.scroll.MaxY <= 0 {
		t.Fatal("expected held action and scrollable preview")
	}
	before := v.scroll.Y
	tester.Scroll(400, 390, 0, 120)
	if v.tableActionPress.active || v.scroll.Y <= before {
		t.Fatalf("scroll was swallowed or capture remained: before=%v after=%v active=%v", before, v.scroll.Y, v.tableActionPress.active)
	}
	tester.Release(x, y)
	if buffer.String() != source || buffer.History().UndoDepth != 0 || v.draft == nil || v.tableActionPress.active {
		t.Fatal("canceled press modified source or lost cell draft")
	}
}
