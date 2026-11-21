package markdownview

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

// The unchanged Task140 draft_action_reentry_test.go is a pinned common input.
// Every production symbol here also exists in the Task140 staged baseline;
// the whole file must compile there, independently of the eventual -run filter.
const markdownUISecondCell = "数据，第 2 行，第 2 列：乙"

func captureMarkdownUISecondCell(t *testing.T, v *View) tableCellTarget {
	t.Helper()
	v.refresh()
	preview := previewSnapshot{v.doc, v.rows, v.docBuffer, v.version}
	for _, row := range preview.rows {
		if row.tableRow != 1 {
			continue
		}
		candidate, ok := tableCellTargetOf(preview, row, preview.doc.Blocks[row.block], 1)
		if ok {
			if target, captured := captureTableCellTarget(preview, candidate); captured {
				return target
			}
		}
	}
	t.Fatal("fixture has no capturable second cell")
	return tableCellTarget{}
}

func markdownUICellFocusBufferReceipt(buffer *text.Buffer) func(*testing.T) {
	value, version, history, selection, group := buffer.String(), buffer.Version(), buffer.History(), buffer.Selection(), buffer.InGroup()
	return func(t *testing.T) {
		t.Helper()
		if buffer.String() != value || buffer.Version() != version || buffer.History() != history ||
			buffer.Selection() != selection || buffer.InGroup() != group {
			t.Fatal("cell focus preparation changed buffer bytes, version, selection, history or group ownership")
		}
	}
}

func runMarkdownUICell(t *testing.T, f *markdownActionFixture, target tableCellTarget, route string) error {
	t.Helper()
	if route == "direct" {
		return f.view.activateTableCell(target)
	}
	f.tester.Advance(time.Second)
	info := tableCellAccess(t, f.tester, ui.RoleCell, markdownUISecondCell)
	if route == "enter" {
		sameMain := markdownUICellFocusBufferReceipt(f.buffer)
		draft, cell, recovery := f.view.draft, f.view.cellEdit, f.view.recoveryDraft
		changes, reports, opened := f.changes, len(f.reports), len(f.opened)
		var draftBuffer *text.Buffer
		var sameDraft func(*testing.T)
		if draft != nil {
			draftBuffer = draft.Buffer
			sameDraft = markdownUICellFocusBufferReceipt(draftBuffer)
		}
		sameBeforeEnter := func() {
			sameMain(t)
			if sameDraft != nil {
				sameDraft(t)
			}
			if f.view.draft != draft || f.view.cellEdit != cell || f.view.recoveryDraft != recovery ||
				draft != nil && draft.Buffer != draftBuffer || f.view.tableCellActivation.buffer != nil ||
				f.changes != changes || len(f.reports) != reports || len(f.opened) != opened {
				t.Fatal("cell focus preparation activated a target or changed draft ownership/events before Enter")
			}
		}
		sameBeforeEnter()
		// A single press leaves the draft and selects the cell's text child.
		// That child is absent from focusOrder. Reverse Tab starts at the last
		// stop and reaches the preview cell before the earlier draft editor,
		// whose own Tab handler edits indentation. No cell is activated here.
		f.tester.Press(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
		f.tester.CancelPress()
		sameBeforeEnter()
		for attempt := 0; attempt < 64; attempt++ {
			info = tableCellAccess(t, f.tester, ui.RoleCell, markdownUISecondCell)
			if info.Focused {
				break
			}
			f.tester.Key(ui.Shift, ui.KeyTab)
			f.tester.Frame()
			sameBeforeEnter()
		}
		info = tableCellAccess(t, f.tester, ui.RoleCell, markdownUISecondCell)
		if !info.Focused {
			t.Fatalf("second cell did not receive exact native Shift+Tab focus: %#v", info)
		}
		if !f.tester.Focused(markdownUISecondCell) {
			t.Fatal("second cell did not receive native keyboard focus")
		}
		sameBeforeEnter()
		f.tester.Key(0, ui.KeyEnter)
	} else {
		doubleClickAccess(t, f.tester, info)
	}
	return nil
}

func TestMarkdownUIReportingCellChangedSuccessor(t *testing.T) {
	for _, route := range []string{"direct", "enter", "pointer"} {
		t.Run(route, func(t *testing.T) {
			f := newMarkdownActionFixture(t, false)
			target := captureMarkdownUISecondCell(t, f.view)
			old := f.openCell(t, "已写🙂")
			version, selection := f.buffer.Version(), f.buffer.Selection()
			want := strings.Replace(markdownActionSource, "| 甲 |", "| 已写🙂 |", 1)
			end := text.UTF16Len(markdownActionSource[:strings.Index(markdownActionSource, "| 甲 |")+2]) + text.UTF16Len("已写🙂")
			committed := text.Range{Anchor: end, Head: end}
			var successor *editor.Editor
			var sameMain, sameOwner, sameSuccessor func(*testing.T)
			f.onChanged = func() {
				if f.view.tableCellActivation.buffer != nil || f.view.draft != nil ||
					f.buffer.String() != want || f.buffer.Version() != version+1 || f.buffer.Selection() != committed ||
					f.buffer.History().UndoDepth != 1 || f.buffer.History().RedoDepth != 0 || f.buffer.InGroup() {
					t.Fatal("Changed did not see one completed old-cell write with activation retired")
				}
				sameMain = markdownActionBufferReceipt(f.buffer)
				successor = f.installSuccessor(t)
				sameOwner = f.ownerReceipt(t, successor)
				sameSuccessor = markdownActionBufferReceipt(successor.Buffer)
			}
			returned := runMarkdownUICell(t, f, target, route)
			if successor == nil || successor == old || sameMain == nil || sameOwner == nil || sameSuccessor == nil {
				t.Fatal("cell action did not reach the real successor callback")
			}
			sameMain(t)
			sameOwner(t)
			sameSuccessor(t)
			if f.changes != 1 || len(f.opened) != 0 || len(f.reports) != 1 ||
				f.reports[0].Error() != "原单元格已写回："+markdownActionSuperseded {
				t.Fatalf("cell wrapper repeated/continued the action: changes=%d reports=%v opened=%v", f.changes, f.reports, f.opened)
			}
			if route == "direct" && returned != f.reports[0] {
				t.Fatal("direct activation changed the original returned/reported error object")
			}
			f.focusSuccessor(t)
			sameOwner(t)
			sameMain(t)
			if !f.buffer.Undo() || f.buffer.String() != markdownActionSource || f.buffer.Selection() != selection ||
				f.buffer.History().UndoDepth != 0 || f.buffer.History().RedoDepth != 1 {
				t.Fatal("main undo did not restore exact bytes and directed selection")
			}
			sameSuccessor(t)
			if !f.buffer.Redo() || f.buffer.String() != want || f.buffer.Selection() != committed ||
				f.buffer.History().UndoDepth != 1 || f.buffer.History().RedoDepth != 0 {
				t.Fatal("main redo did not restore exactly the old-cell commit")
			}
			sameSuccessor(t)
			f.independentSuccessorHistory(t, successor)
			if f.changes != 1 || len(f.reports) != 1 || len(f.opened) != 0 {
				t.Fatal("successor history leaked into an old action")
			}
		})
	}
}

func TestMarkdownUIReportingCellRefusalSuccessor(t *testing.T) {
	for _, route := range []string{"direct", "enter", "pointer"} {
		for _, refusal := range []string{"pipe", "readonly", "size"} {
			t.Run(route+"/"+refusal, func(t *testing.T) {
				f := newMarkdownActionFixture(t, refusal == "size")
				target := captureMarkdownUISecondCell(t, f.view)
				value, wantError := "坏|值", errTableCellPipe
				if refusal != "pipe" {
					value = "已写🙂"
				}
				old := f.openCell(t, value)
				if refusal == "readonly" {
					f.view.host.ReadOnly, wantError = true, errTableCellReadOnly
				} else if refusal == "size" {
					wantError = text.ErrSizeLimit
				}
				sameMain := markdownActionBufferReceipt(f.buffer)
				var successor *editor.Editor
				var sameOwner, sameSuccessor func(*testing.T)
				f.onReport = func(err error) {
					if f.view.tableCellActivation.buffer != nil || !errors.Is(err, wantError) || strings.Contains(err.Error(), "已写回") {
						t.Fatalf("refused write lost cause/activation or invented a commit: %v", err)
					}
					f.view.host.ReadOnly = false
					successor = f.installSuccessor(t)
					sameOwner = f.ownerReceipt(t, successor)
					sameSuccessor = markdownActionBufferReceipt(successor.Buffer)
				}
				returned := runMarkdownUICell(t, f, target, route)
				if successor == nil || successor == old || sameOwner == nil || sameSuccessor == nil {
					t.Fatal("terminal cell Report did not install its successor")
				}
				sameMain(t)
				sameOwner(t)
				sameSuccessor(t)
				if f.changes != 0 || len(f.opened) != 0 || len(f.reports) != 1 || !errors.Is(f.reports[0], wantError) {
					t.Fatalf("cell refused write was repeated or continued: reports=%v changes=%d opened=%v", f.reports, f.changes, f.opened)
				}
				if route == "direct" && returned != f.reports[0] {
					t.Fatal("direct refusal did not keep returned/reported object identity")
				}
				f.focusSuccessor(t)
				sameMain(t)
				sameOwner(t)
				sameSuccessor(t)
				f.independentSuccessorHistory(t, successor)
				if f.changes != 0 || len(f.reports) != 1 || len(f.opened) != 0 {
					t.Fatal("refused cell action changed later independent history/events")
				}
			})
		}
	}
}

func TestMarkdownUIReportingCellPostFlushTarget(t *testing.T) {
	for _, route := range []string{"direct", "enter", "pointer"} {
		for _, state := range []string{"clean-cell", "changed-table", "readonly-after-commit"} {
			t.Run(route+"/"+state, func(t *testing.T) {
				f := newMarkdownActionFixture(t, false)
				target := captureMarkdownUISecondCell(t, f.view)
				value := "已写🙂"
				if state == "clean-cell" {
					value = "甲"
				}
				f.openCell(t, value)
				version, selection := f.buffer.Version(), f.buffer.Selection()
				if state == "readonly-after-commit" {
					f.onChanged = func() { f.view.host.ReadOnly = true }
				}
				returned := runMarkdownUICell(t, f, target, route)
				if state == "clean-cell" {
					if returned != nil || len(f.reports) != 0 || f.changes != 0 || len(f.opened) != 0 ||
						f.buffer.String() != markdownActionSource || f.buffer.Version() != version || f.buffer.Selection() != selection ||
						f.buffer.History().UndoDepth != 0 || f.buffer.History().RedoDepth != 0 || f.buffer.InGroup() ||
						f.view.draft == nil || f.view.cellEdit == nil || f.view.cellEdit.label != markdownActionSecondField || f.view.draft.Buffer.String() != "乙" {
						t.Fatal("clean cell flush did not open the new cell without reporting/writing")
					}
					return
				}
				want := strings.Replace(markdownActionSource, "| 甲 |", "| 已写🙂 |", 1)
				end := text.UTF16Len(markdownActionSource[:strings.Index(markdownActionSource, "| 甲 |")+2]) + text.UTF16Len("已写🙂")
				committed := text.Range{Anchor: end, Head: end}
				wantError := errTableCellAfterFlush
				if state == "readonly-after-commit" {
					wantError = errTableCellReadOnly
				}
				if f.buffer.String() != want || f.buffer.Version() != version+1 || f.buffer.Selection() != committed ||
					f.buffer.History().UndoDepth != 1 || f.buffer.History().RedoDepth != 0 || f.buffer.InGroup() ||
					f.changes != 1 || len(f.opened) != 0 || f.view.draft != nil {
					t.Fatal("post-flush target refusal lost or repeated the committed old-cell write")
				}
				if route == "direct" {
					if returned != wantError || len(f.reports) != 0 {
						t.Fatalf("direct post-flush refusal changed caller reporting: %v reports=%v", returned, f.reports)
					}
				} else if len(f.reports) != 1 || f.reports[0] != wantError {
					t.Fatalf("successful flush hid a later context/readonly refusal: %v", f.reports)
				}
				if !f.buffer.Undo() || f.buffer.String() != markdownActionSource || f.buffer.Selection() != selection ||
					f.buffer.History().UndoDepth != 0 || f.buffer.History().RedoDepth != 1 || !f.buffer.Redo() ||
					f.buffer.String() != want || f.buffer.Selection() != committed ||
					f.buffer.History().UndoDepth != 1 || f.buffer.History().RedoDepth != 0 {
					t.Fatal("post-flush refusal did not retain exact main undo/redo ownership")
				}
			})
		}
	}
}

func TestMarkdownUIReportingCellTargetControls(t *testing.T) {
	for _, route := range []string{"direct", "enter", "pointer"} {
		for _, state := range []string{"success", "readonly", "foreign-group"} {
			t.Run(route+"/"+state, func(t *testing.T) {
				f := newMarkdownActionFixture(t, false)
				target := captureMarkdownUISecondCell(t, f.view)
				sameMain := markdownActionBufferReceipt(f.buffer)
				want := errTableCellReadOnly
				if state == "readonly" {
					f.view.host.ReadOnly = true
				} else if state == "foreign-group" {
					want = errTableCellStale
					if err := f.buffer.BeginGroup(); err != nil {
						t.Fatal(err)
					}
				}
				returned := runMarkdownUICell(t, f, target, route)
				if state == "foreign-group" {
					if !f.buffer.InGroup() || f.buffer.EndGroup() {
						t.Fatal("cell action stole or wrote in the foreign group")
					}
				}
				sameMain(t)
				if f.changes != 0 || len(f.opened) != 0 {
					t.Fatal("target-only cell action wrote/opened a reference")
				}
				if state == "success" {
					if returned != nil || len(f.reports) != 0 || f.view.draft == nil || f.view.cellEdit == nil || f.view.draft.Buffer.String() != "乙" {
						t.Fatalf("valid cell open = %v reports=%v", returned, f.reports)
					}
					return
				}
				if f.view.draft != nil {
					t.Fatal("refused target opened a cell")
				}
				if route == "direct" {
					if returned != want || len(f.reports) != 0 {
						t.Fatalf("direct target failure changed caller reporting contract: %v %v", returned, f.reports)
					}
				} else if len(f.reports) != 1 || f.reports[0] != want {
					t.Fatalf("UI lost target/readonly refusal: %v", f.reports)
				}
			})
		}
	}
	t.Run("invalid-target-and-open-validation", func(t *testing.T) {
		f := newMarkdownActionFixture(t, false)
		sameMain := markdownActionBufferReceipt(f.buffer)
		if err := f.view.activateTableCell(tableCellTarget{}); err != errTableCellStale {
			t.Fatalf("pre-flush target error changed: %v", err)
		}
		target := captureMarkdownUISecondCell(t, f.view)
		if err := f.view.openTableCellDraft(tableCellContext{}, tableCellTarget{}); err != errTableCellStale {
			t.Fatalf("open validation error changed: %v", err)
		}
		sameMain(t)
		replaceReferenceBufferText(t, f.buffer, "| 甲 |", "| 改 |")
		staleMain := markdownActionBufferReceipt(f.buffer)
		if err := f.view.activateTableCell(target); err != errTableCellStale || len(f.reports) != 0 || f.view.draft != nil {
			t.Fatalf("direct stale target contract changed: %v reports=%v", err, f.reports)
		}
		staleMain(t)
		// Version/history change only through the explicit external edit and undo.
		if !f.buffer.Undo() || f.buffer.String() != markdownActionSource ||
			f.buffer.Selection() != (text.Range{Anchor: f.buffer.LenUTF16(), Head: 2}) ||
			f.buffer.History().UndoDepth != 0 || f.buffer.History().RedoDepth != 1 {
			t.Fatal("stale control did not preserve the external edit and original directed selection")
		}
	})
}
