package markdownview

import (
	"errors"
	"fmt"
	"image/png"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/ui/editor"
)

const columnBefore = "在此列前插入列"
const columnAfter = "在此列后插入列"
const columnDelete = "删除此列"

const columnBeforeSource = "# 前🙂\r\n\r\n|| H | B |\r\n|---|---|---|\r\n|| 甲 | 乙 |\r\n|| 同 | 值 |\r\n\r\n尾🙂\r\n"
const columnAfterSource = "# 前🙂\r\n\r\n| H || B |\r\n|---|---|---|\r\n| 甲 || 乙 |\r\n| 同 || 值 |\r\n\r\n尾🙂\r\n"
const columnDeleteSource = "# 前🙂\r\n\r\n| B |\r\n|---|\r\n| 乙 |\r\n| 值 |\r\n\r\n尾🙂\r\n"

func focusColumn(t *testing.T, tester *ui.Tester, name string) {
	t.Helper()
	tester.Key(0, ui.KeyF6)
	if !rowButton(t, tester, "在下方插入行").Focused {
		t.Fatal("F6 no longer enters the original first row control")
	}
	for step := 0; step < 5; step++ {
		if rowButton(t, tester, name).Focused {
			return
		}
		tester.Key(0, ui.KeyTab)
	}
	t.Fatal("Tab did not reach the requested column control", name)
}

func TestTableColumnsNativePointerKeyboardOneUndo(t *testing.T) {
	for _, action := range []struct{ name, want string }{
		{columnBefore, columnBeforeSource}, {columnAfter, columnAfterSource}, {columnDelete, columnDeleteSource},
	} {
		for _, mode := range []string{"pointer", "enter", "space"} {
			t.Run(action.name+"/"+mode, func(t *testing.T) {
				buffer, v, tester := rowView(t, rowActionSource, 760)
				selection := text.Range{Anchor: buffer.LenUTF16(), Head: 2}
				_ = buffer.SetSelection(selection.Anchor, selection.Head)
				changes := 0
				v.host.Changed = func() {
					changes++
					if buffer.InGroup() || buffer.String() != action.want {
						t.Fatal("callback observed a partial column transaction")
					}
				}
				if mode == "pointer" {
					clickDraftControl(t, tester, action.name)
				} else {
					focusColumn(t, tester, action.name)
					key := ui.KeyEnter
					if mode == "space" {
						key = ui.KeySpace
					}
					tester.KeyRepeat(0, key)
					if buffer.String() != rowActionSource || v.draft == nil {
						t.Fatal("repeated key activated a column action")
					}
					tester.Key(0, key)
				}
				wantSelection := text.Range{Anchor: text.UTF16Len(action.want), Head: 2}
				if buffer.String() != action.want || buffer.Selection() != wantSelection || buffer.History().UndoDepth != 1 ||
					changes != 1 || v.draft != nil || v.cellEdit != nil || v.tableActionError != nil {
					t.Fatalf("column result=%q selection=%+v history=%+v changes=%d err=%v", buffer.String(), buffer.Selection(), buffer.History(), changes, v.tableActionError)
				}
				if !buffer.Undo() || buffer.String() != rowActionSource || buffer.Selection() != selection ||
					!buffer.Redo() || buffer.String() != action.want || buffer.Selection() != wantSelection {
					t.Fatal("column edits were not one exact undo/redo with oriented selection")
				}
			})
		}
	}
}

func TestTableColumnsNarrowGeometryAndFocusOrder(t *testing.T) {
	for _, width := range []float32{265, 300, 760} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			buffer, _, tester := rowView(t, rowActionSource, width)
			if path := os.Getenv("LEVNOTE_MARKDOWN_COLUMNS_SCREENSHOT"); path != "" && width == 300 {
				file, err := os.Create(path)
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(file, tester.Image())
				closeErr := file.Close()
				if err != nil || closeErr != nil {
					t.Fatal(err, closeErr)
				}
			}
			names := []string{"在下方插入行", "删除此数据行", columnBefore, columnAfter, columnDelete,
				"第1列默认对齐", "第1列左对齐", "第1列居中对齐", "第1列右对齐"}
			tester.Key(0, ui.KeyF6)
			var columnY float32
			for i, name := range names {
				button := rowButton(t, tester, name)
				if !button.Focused || button.Bounds.X < 0 || button.Bounds.X+button.Bounds.W > width || button.Bounds.Y+button.Bounds.H > 451 {
					t.Fatalf("ordered control %q is unfocused/outside viewport: %+v", name, button)
				}
				if i == 2 {
					columnY = button.Bounds.Y
					if columnY <= rowButton(t, tester, names[0]).Bounds.Y {
						t.Fatal("column controls did not follow row controls")
					}
				} else if i == 3 || i == 4 {
					if button.Bounds.Y != columnY {
						t.Fatal("short column controls failed to fit the established narrow pane")
					}
				} else if i == 5 && button.Bounds.Y <= columnY {
					t.Fatal("alignment controls did not follow column controls")
				}
				if i < len(names)-1 {
					tester.Key(0, ui.KeyTab)
				}
			}
			for i := len(names) - 2; i >= 0; i-- {
				tester.Key(ui.Shift, ui.KeyTab)
				if !rowButton(t, tester, names[i]).Focused {
					t.Fatal("ShiftTab order changed", names[i])
				}
			}
			if buffer.String() != rowActionSource || buffer.History().UndoDepth != 0 {
				t.Fatal("focus traversal changed source")
			}
		})
	}
}

func TestTableColumnsBoundaryDisabledAndReadonly(t *testing.T) {
	for _, count := range []int{1, 8} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			source := "| H " + strings.Repeat("| X ", count-1) + "|\n|" + strings.Repeat("---|", count) + "\n"
			buffer := text.New(source)
			v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
			tester := ui.NewTester(v.Draw, 760, 480)
			tester.Frame()
			openCellByPointer(t, tester, ui.RoleColumnHeader, "表头，第 1 行，第 1 列：H")
			for _, name := range []string{columnBefore, columnAfter, columnDelete} {
				button, ok := tester.AccessByRole(ui.RoleButton, name)
				disabled := count == 1 && name == columnDelete || count == 8 && name != columnDelete
				if !ok || button.Disabled != disabled || button.Focusable == disabled {
					t.Fatalf("total-column boundary %d %s: %+v", count, name, button)
				}
			}
			tester.Key(0, ui.KeyF6)
			if !rowButton(t, tester, "在下方插入行").Focused {
				t.Fatal("F6 must start from the still-focused cell editor")
			}
			tester.Key(0, ui.KeyTab) // header row deletion and disabled column actions are skipped
			first := columnBefore
			if count == 8 {
				first = columnDelete
			}
			if !rowButton(t, tester, first).Focused {
				t.Fatal("Tab did not skip disabled controls")
			}
			tester.Key(ui.Shift, ui.KeyTab)
			if !rowButton(t, tester, "在下方插入行").Focused {
				t.Fatal("ShiftTab did not skip disabled controls")
			}
			for _, name := range []string{columnBefore, columnAfter, columnDelete} {
				button, _ := tester.AccessByRole(ui.RoleButton, name)
				if button.Disabled {
					tester.ClickAt(button.Bounds.X+4, button.Bounds.Y+4)
				}
			}
			v.host.ReadOnly = true
			tester.Frame()
			for _, name := range []string{columnBefore, columnAfter, columnDelete} {
				button, ok := tester.AccessByRole(ui.RoleButton, name)
				if !ok || !button.Disabled || button.Focusable {
					t.Fatal("readonly column control remained active", name)
				}
				tester.ClickAt(button.Bounds.X+4, button.Bounds.Y+4)
			}
			if buffer.String() != source || buffer.History().UndoDepth != 0 || v.draft == nil {
				t.Fatal("disabled controls altered source/history/draft")
			}
		})
	}
}

func TestTableColumnsCompositionGeometryAndTwoUndos(t *testing.T) {
	buffer, v, tester := rowView(t, rowActionSource, 265)
	_ = buffer.SetSelection(2, 0)
	beforeSelection := buffer.Selection()
	replaceDraftText(t, tester, "新🙂")
	draft := v.draft
	marked, composing := draft.MarkedRange()
	value, version, selection, history := draft.Buffer.String(), draft.Buffer.Version(), draft.Buffer.Selection(), draft.Buffer.History()
	tester.Key(0, ui.KeyF6)
	field, ok := tester.AccessByRole(ui.RoleTextField, v.cellEdit.label)
	afterMarked, afterComposing := draft.MarkedRange()
	if !composing || !afterComposing || !ok || !field.Focused || marked != afterMarked ||
		draft.Buffer.String() != value || draft.Buffer.Version() != version || draft.Buffer.Selection() != selection || draft.Buffer.History() != history ||
		!draft.Buffer.InGroup() || !tester.HasText(tableActionComposingHint) {
		t.Fatal("F6 changed live composition")
	}
	changes := 0
	v.host.Changed = func() {
		changes++
		if buffer.InGroup() {
			t.Fatal("callback inside group")
		}
	}
	button := rowButton(t, tester, columnBefore)
	x, y := button.Bounds.X+4, button.Bounds.Y+4
	tester.Press(x, y)
	if rowButton(t, tester, columnBefore).Bounds != button.Bounds || !v.tableActionPress.active {
		t.Fatal("normal composition confirmation moved the pressed column button")
	}
	tester.Release(x, y)
	written := strings.Replace(rowActionSource, "| 甲 |", "| 新🙂 |", 1)
	want := strings.Replace(columnBeforeSource, "| 甲 |", "| 新🙂 |", 1)
	caret := text.UTF16Len(written[:strings.Index(written, "新🙂")]) + 3
	wantCaret := text.UTF16Len(want[:strings.Index(want, "新🙂")]) + 3
	if buffer.String() != want || buffer.Selection() != (text.Range{Anchor: wantCaret, Head: wantCaret}) ||
		buffer.History().UndoDepth != 2 || changes != 2 || v.draft != nil || v.tableActionError != nil {
		t.Fatalf("column/cell transactions=%q sel=%+v history=%+v changes=%d err=%v", buffer.String(), buffer.Selection(), buffer.History(), changes, v.tableActionError)
	}
	if !buffer.Undo() || buffer.String() != written || buffer.Selection() != (text.Range{Anchor: caret, Head: caret}) ||
		!buffer.Undo() || buffer.String() != rowActionSource || buffer.Selection() != beforeSelection ||
		!buffer.Redo() || buffer.String() != written || !buffer.Redo() || buffer.String() != want {
		t.Fatal("cell and column did not preserve independent undo/redo")
	}
}

func TestTableColumnsSelectionAcrossEditsAndInsertionAffinity(t *testing.T) {
	for _, mode := range []string{"retained-reverse", "deleted-interior", "insert-boundary"} {
		t.Run(mode, func(t *testing.T) {
			buffer, v, _ := rowView(t, rowActionSource, 760)
			point := func(s, token string) int { return text.UTF16Len(s[:strings.Index(s, token)]) }
			action, want := markdown.TableColumnDelete, columnDeleteSource
			selection := text.Range{Anchor: point(rowActionSource, "值"), Head: point(rowActionSource, "B")}
			mapped := text.Range{Anchor: point(want, "值"), Head: point(want, "B")}
			if mode == "deleted-interior" {
				selection = text.Range{Anchor: point(rowActionSource, "甲"), Head: point(rowActionSource, "H")}
				mapped = text.Range{Anchor: point(want, "| 乙") + 1, Head: point(want, "| B") + 1}
			} else if mode == "insert-boundary" {
				action, want = markdown.TableColumnInsertBefore, columnBeforeSource
				selection = text.Range{Anchor: point(rowActionSource, "| 同") + 1, Head: point(rowActionSource, "| H") + 1}
				mapped = text.Range{Anchor: point(want, "|| 同") + 2, Head: point(want, "|| H") + 2}
			}
			_ = buffer.SetSelection(selection.Anchor, selection.Head)
			g := v.captureTableAction(tableAction{kind: tableActionColumn, column: action}, v.cellEdit, v.draft)
			if err := v.activateTableAction(g); err != nil || buffer.String() != want || buffer.Selection() != mapped {
				t.Fatalf("original offsets/affinity failed: %v source=%q selection=%+v want=%+v", err, buffer.String(), buffer.Selection(), mapped)
			}
			if !buffer.Undo() || buffer.String() != rowActionSource || buffer.Selection() != selection {
				t.Fatal("mapped selection undo lost endpoints")
			}
		})
	}
}

func TestTableColumnsInvalidDraftConflictAndPostFlushRefusal(t *testing.T) {
	for _, mode := range []string{"invalid", "conflict", "post-flush-context"} {
		t.Run(mode, func(t *testing.T) {
			source, label := rowActionSource, columnBefore
			if mode == "post-flush-context" {
				source, label = "H | B\r\n---|---\r\n甲 | 乙\\\r\n", columnAfter
			}
			buffer, v, tester := rowView(t, source, 300)
			if mode == "post-flush-context" {
				v.DiscardDraft()
				tester.Frame()
				tester.Advance(time.Second)
				openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 2 列：乙\\")
			}
			value := "草稿🙂"
			if mode == "invalid" {
				value = "坏|值"
			}
			if mode == "post-flush-context" {
				value = "新🙂\\"
			}
			replaceDraftText(t, tester, value)
			if mode == "conflict" {
				replaceTableBufferText(t, buffer, "乙", "外")
			}
			draft, before, history := v.draft, buffer.String(), buffer.History()
			clickDraftControl(t, tester, label)
			if mode == "post-flush-context" {
				want := strings.Replace(source, "乙\\", "新🙂\\", 1)
				if buffer.String() != want || buffer.History().UndoDepth != 1 || v.draft != nil || v.tableActionError == nil ||
					!strings.Contains(v.tableActionError.Error(), "单元格已写回，列未修改") || !strings.Contains(v.tableActionError.Error(), "原始单元格") {
					t.Fatalf("post-cell refusal lost committed cell/truth: source=%q err=%v", buffer.String(), v.tableActionError)
				}
				if !buffer.Undo() || buffer.String() != source {
					t.Fatal("refused column lost cell undo")
				}
				return
			}
			if buffer.String() != before || buffer.History() != history || v.draft != draft || draft.Buffer.String() != value ||
				v.tableActionError == nil || !strings.Contains(v.tableActionError.Error(), "列未修改") {
				t.Fatal("refusal changed source or lost reachable draft", v.tableActionError)
			}
			if mode == "invalid" && !errors.Is(v.draftError, errTableCellPipe) {
				t.Fatal("wrong draft refusal", v.draftError)
			}
			// A stale capture refuses before normal Flush; the ordinary completion
			// action exposes the existing conflict recovery path without a new one.
			if mode == "conflict" {
				clickDraftControl(t, tester, "完成编辑")
			}
			recovered := ""
			v.host.RecoverDraft = func(_, value string) error { recovered = value; return nil }
			clickDraftControl(t, tester, "打开未同步副本")
			if recovered != value || buffer.String() != before {
				t.Fatal("refused draft recovery was not exact")
			}
		})
	}
}

func TestTableColumnsHeldGestureInvalidation(t *testing.T) {
	for _, mode := range []string{"outside", "selection", "draft", "readonly", "foreign-group", "scroll", "cancel", "release"} {
		t.Run(mode, func(t *testing.T) {
			source := rowActionSource + strings.Repeat("\nparagraph\n", 80)
			buffer, v, tester := rowView(t, source, 760)
			button := rowButton(t, tester, columnDelete)
			x, y := button.Bounds.X+4, button.Bounds.Y+4
			tester.Press(x, y)
			if !v.tableActionPress.active {
				t.Fatal("pointer did not capture column action")
			}
			switch mode {
			case "outside":
				_ = buffer.ReplaceUTF16(0, 0, "外部\n")
			case "selection":
				_ = buffer.SetSelection(buffer.LenUTF16(), 0)
			case "draft":
				_ = v.draft.Buffer.ReplaceUTF16(0, 1, "改")
			case "readonly":
				v.host.ReadOnly = true
			case "foreign-group":
				_ = buffer.BeginGroup()
			case "scroll":
				before := v.scroll.Y
				tester.Scroll(400, 430, 0, 120)
				if v.scroll.Y <= before || v.tableActionPress.active {
					t.Fatal("column cancel swallowed native preview scroll")
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
				t.Fatal("stale held column gesture committed", mode, v.tableActionError)
			}
			if mode == "foreign-group" && (!buffer.InGroup() || !buffer.CancelGroup()) {
				t.Fatal("foreign group was consumed")
			}
		})
	}
}

func TestTableColumnsHeldKeyCannotRetargetReplacementPanel(t *testing.T) {
	buffer, v, tester := rowView(t, rowActionSource, 760)
	focusColumn(t, tester, columnAfter)
	tester.KeyDown(ui.KeySpace)
	if buffer.String() != columnAfterSource || buffer.History().UndoDepth != 1 || v.draft != nil {
		t.Fatal("initial held key failed")
	}
	tester.Advance(time.Second)
	openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 2 列：（空单元格）")
	draft := v.draft
	focusColumn(t, tester, columnDelete)
	tester.KeyRepeat(0, ui.KeySpace)
	tester.KeyUp(ui.KeySpace)
	if buffer.String() != columnAfterSource || buffer.History().UndoDepth != 1 || v.draft != draft {
		t.Fatal("held key retargeted replacement panel")
	}
	tester.Key(0, ui.KeySpace)
	if buffer.String() != rowActionSource || buffer.History().UndoDepth != 2 || v.draft != nil {
		t.Fatal("new key after release did not delete inserted column")
	}
}

func TestTableColumnsRapidPointerCannotDeleteThroughNewPanel(t *testing.T) {
	buffer := text.New(rowActionSource)
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 760, 480)
	tester.Frame()
	initial := tableCellAccess(t, tester, ui.RoleCell, "数据，第 3 行，第 1 列：同")
	x, y := initial.Bounds.X+initial.Bounds.W/2, initial.Bounds.Y+initial.Bounds.H/4
	tester.ClickAt(x, y)
	tester.ClickAt(x, y)
	if v.draft == nil {
		t.Fatal("initial double click did not open the selected cell")
	}
	focusAlignment(t, tester, 0, markdown.TableAlignDefault)
	tester.Key(0, ui.KeyEnter) // close the initial panel without a source change
	if v.draft != nil || buffer.String() != rowActionSource || buffer.History().UndoDepth != 0 {
		t.Fatal("clean no-op did not close the initial panel")
	}
	tester.ClickAt(x, y) // no virtual clock advance: continue the opening click sequence
	if v.draft == nil {
		t.Fatal("rapid continuation did not reopen the cell")
	}
	button := rowButton(t, tester, columnDelete)
	if x < button.Bounds.X || x > button.Bounds.X+button.Bounds.W || y < button.Bounds.Y || y > button.Bounds.Y+button.Bounds.H {
		t.Fatalf("continued click must physically overlap the column-delete control: point=%v,%v bounds=%+v", x, y, button.Bounds)
	}
	draft, version, history := v.draft, buffer.Version(), buffer.History()
	tester.ClickAt(x, y)
	if buffer.String() != rowActionSource || buffer.Version() != version || buffer.History() != history || v.draft != draft || v.tableActionPress.active {
		t.Fatal("rapid continuation deleted an entire column through the new panel")
	}
	tester.Advance(time.Second)
	clickDraftControl(t, tester, columnDelete)
	if buffer.String() != columnDeleteSource || buffer.History().UndoDepth != 1 || v.draft != nil {
		t.Fatal("fresh deliberate column click remained blocked")
	}
}

func TestTableColumnsCallbackReplacementOwnership(t *testing.T) {
	for _, moment := range []string{"cell-changed", "column-changed", "report"} {
		for _, replacement := range []string{"release", "buffer", "draft"} {
			t.Run(moment+"/"+replacement, func(t *testing.T) {
				buffer, v, tester := rowView(t, rowActionSource, 760)
				replacementError := errors.New("replacement issue")
				var replacementDraft *editor.Editor
				var replacementBuffer *text.Buffer
				callbackRan, reports, changes := false, "", 0
				replace := func() {
					if buffer.InGroup() {
						t.Fatal("callback ran in column group")
					}
					callbackRan = true
					if replacement == "release" {
						v.Release()
						replacementDraft = v.draft
						return
					}
					v.clearDraft()
					if replacement == "buffer" {
						replacementBuffer = text.New("replacement")
						v.host.Buffer = replacementBuffer
					} else {
						replacementDraft = editor.New(text.New("replacement draft"))
						v.draft = replacementDraft
					}
					v.tableActionError = replacementError
				}
				v.host.Changed = func() {
					changes++
					if moment == "cell-changed" || moment == "column-changed" && changes == 2 {
						replace()
					}
				}
				v.host.Report = func(err error) {
					reports += err.Error()
					if moment == "report" && !callbackRan {
						replace()
					}
				}
				value := "新🙂"
				if moment == "report" {
					value = "坏|值"
				}
				replaceDraftText(t, tester, value)
				clickDraftControl(t, tester, columnBefore)
				want, depth := rowActionSource, 0
				if moment != "report" {
					want, depth = strings.Replace(rowActionSource, "| 甲 |", "| 新🙂 |", 1), 1
				}
				if moment == "column-changed" {
					want, depth = strings.Replace(columnBeforeSource, "| 甲 |", "| 新🙂 |", 1), 2
				}
				if !callbackRan || buffer.String() != want || buffer.History().UndoDepth != depth || changes != depth {
					t.Fatalf("callback commit ownership failed source=%q history=%+v changes=%d", buffer.String(), buffer.History(), changes)
				}
				if replacement == "release" {
					if v.draft != replacementDraft || v.tableActionError != nil {
						t.Fatal("released UI resurrected")
					}
				} else if v.tableActionError != replacementError || replacementDraft != nil && v.draft != replacementDraft ||
					replacementBuffer != nil && v.host.Buffer != replacementBuffer {
					t.Fatal("old action overwrote replacement")
				}
				if moment == "cell-changed" && !strings.Contains(reports, "单元格已写回，列未修改") {
					t.Fatal("missing truthful post-Flush refusal", reports)
				}
			})
		}
	}
}

func TestTableColumnsOriginalRangeValidationAndOwnedRollback(t *testing.T) {
	t.Run("original-byte-validation", func(t *testing.T) {
		for _, edits := range [][]tableActionEdit{
			nil,
			{{rng: markdown.SourceRange{Start: -1, End: 0}}},
			{{rng: markdown.SourceRange{Start: 2, End: 2}}}, // middle of the UTF-8 emoji
			{{rng: markdown.SourceRange{Start: 1, End: 3}}},
			{{rng: markdown.SourceRange{Start: 0, End: 99}}},
			{{rng: markdown.SourceRange{Start: 5, End: 4}}},
			{{rng: markdown.SourceRange{Start: 0, End: 1}, value: string([]byte{0xff})}},
			{{rng: markdown.SourceRange{Start: 0, End: 5}}, {rng: markdown.SourceRange{Start: 1, End: 5}}},
			{{rng: markdown.SourceRange{Start: 5, End: 5}}, {rng: markdown.SourceRange{Start: 0, End: 0}}},
			{{rng: markdown.SourceRange{Start: 1, End: 1}}, {rng: markdown.SourceRange{Start: 1, End: 1}}},
		} {
			selection := text.Range{Anchor: 4, Head: 0}
			if got, mapped, err := prepareTableActionEdits("A🙂B", tableActionPatch{edits: edits}, selection); err == nil || got != nil || mapped != selection {
				t.Fatal("invalid original edit accepted", edits, got, mapped, err)
			}
		}
	})
	for _, mode := range []string{"replace-error", "selection-error", "foreign-group"} {
		t.Run(mode, func(t *testing.T) {
			buffer := text.New("A🙂B|C")
			_ = buffer.ReplaceUTF16(6, 6, "old")
			buffer.Undo() // preserve an existing redo chain across our canceled group
			selection := text.Range{Anchor: 6, Head: 1}
			_ = buffer.SetSelection(selection.Anchor, selection.Head)
			version, history := buffer.Version(), buffer.History()
			edits := []tableActionUTF16Edit{{start: 2, end: 2, value: "bad"}, {start: 6, end: 6, value: "!"}}
			mapped := selection
			if mode == "selection-error" {
				edits, mapped = edits[1:], text.Range{Anchor: 99, Head: 0}
			}
			if mode == "foreign-group" {
				_ = buffer.BeginGroup()
			}
			if err := applyTableActionEdits(buffer, edits, mapped); err == nil || buffer.String() != "A🙂B|C" || buffer.Selection() != selection || buffer.History() != history {
				t.Fatal("failed transaction did not restore exact state", err, buffer.String(), buffer.Selection(), buffer.History())
			}
			if mode == "foreign-group" {
				if buffer.Version() != version || !buffer.InGroup() || !buffer.CancelGroup() {
					t.Fatal("foreign group consumed")
				}
			} else if buffer.InGroup() || buffer.Version() <= version {
				t.Fatal("rollback must finish own group and may advance revision")
			}
			if !buffer.Redo() || buffer.String() != "A🙂B|Cold" {
				t.Fatal("rollback destroyed preexisting redo")
			}
		})
	}
}
