package markdownview

import (
	"errors"
	"strings"
	"testing"

	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

func TestMarkdownDraftOwnerChangedCommitReceipt(t *testing.T) {
	for _, action := range []string{"outline", "public-flush", "clean-outline"} {
		t.Run(action, func(t *testing.T) {
			f := newDraftOwnerFixture(t)
			old := f.open(t, draftOwnerFirstCell, draftOwnerFirstField)
			outline := Adapter().Outline(editor.Host{Buffer: f.buffer})
			snapshot, err := outline.OutlineSnapshot()
			if err != nil || len(snapshot.Items) == 0 {
				t.Fatalf("fixture outline missing: %+v %v", snapshot, err)
			}
			id := snapshot.Items[0].ID
			if action == "clean-outline" {
				sameMain := draftOwnerBufferReceipt(f.buffer)
				if err := f.view.NavigateOutlineItem(id); err != nil || f.view.outlineCurrent != id {
					t.Fatalf("clean cell should close and navigate without a write: item=%q err=%v", f.view.outlineCurrent, err)
				}
				f.noCapture(t)
				sameMain(t)
				if f.view.draft != nil || f.changes != 0 || len(f.reports) != 0 {
					t.Fatalf("clean navigation left a draft or a write receipt: changes=%d reports=%v", f.changes, f.reports)
				}
				return
			}

			const oldValue = "已写🙂"
			want := strings.Replace(draftOwnerSource, "| 甲 |", "| 已写🙂 |", 1)
			revision, oldSelection := f.buffer.Version(), f.buffer.Selection()
			list, outlineCaret := f.view.list, f.view.outlineCaret
			f.replaceText(oldValue)
			var successor *editor.Editor
			var sameCommittedMain, sameSuccessor func(*testing.T)
			var committedSelection text.Range
			f.onceChanged = func() {
				if f.view.draft != nil || f.view.cellEdit != nil || f.buffer.String() != want || f.buffer.Version() != revision+1 ||
					f.buffer.History().UndoDepth != 1 || f.buffer.History().RedoDepth != 0 || f.buffer.InGroup() {
					t.Fatalf("Changed did not see the exact committed old cell: source=%q history=%+v", f.buffer.String(), f.buffer.History())
				}
				committedSelection = f.buffer.Selection()
				sameCommittedMain = draftOwnerBufferReceipt(f.buffer)
				successor = f.installSuccessor(t)
				sameSuccessor = draftOwnerBufferReceipt(successor.Buffer)
			}
			if action == "outline" {
				err = f.view.NavigateOutlineItem(id)
			} else {
				err = f.view.Flush()
			}
			f.requireSuccessor(t, old, successor)
			sameCommittedMain(t)
			if f.changes != 1 || f.view.list != list || f.view.outlineCurrent != "" || f.view.outlineCaret != outlineCaret {
				t.Fatalf("old action advanced after Changed: changes=%d outline=%q caret=%d", f.changes, f.view.outlineCurrent, f.view.outlineCaret)
			}
			if action == "outline" {
				if !errors.Is(err, errDraftSuperseded) || err.Error() != "原单元格已写回："+errDraftSuperseded.Error() ||
					len(f.reports) != 1 || f.reports[0] != err {
					t.Fatalf("interrupted outline lost the committed write receipt: err=%v reports=%v", err, f.reports)
				}
			} else if err != nil || len(f.reports) != 0 {
				t.Fatalf("public Flush must leave completion to its caller: err=%v reports=%v", err, f.reports)
			}

			if !f.buffer.Undo() || f.buffer.String() != draftOwnerSource || f.buffer.Selection() != oldSelection ||
				f.buffer.History().UndoDepth != 0 || f.buffer.History().RedoDepth != 1 {
				t.Fatalf("one undo must restore the old source and selection: source=%q history=%+v", f.buffer.String(), f.buffer.History())
			}
			sameSuccessor(t)
			if !f.buffer.Redo() || f.buffer.String() != want || f.buffer.Selection() != committedSelection ||
				f.buffer.History().UndoDepth != 1 || f.buffer.History().RedoDepth != 0 {
				t.Fatalf("one redo must restore only the original cell write: source=%q history=%+v", f.buffer.String(), f.buffer.History())
			}
			sameSuccessor(t)
			f.capture(t, draftOwnerNewText, draftOwnerNewSelection)
		})
	}
}

func TestMarkdownDraftOwnerOrdinaryBlockChangedCommitReceipt(t *testing.T) {
	for _, action := range []string{"outline", "public-flush"} {
		t.Run(action, func(t *testing.T) {
			f := newDraftOwnerFixture(t)
			old := f.openBlock(t, "末行")
			if f.view.recoveryDraft != nil {
				t.Fatal("ordinary block unexpectedly owns cell recovery state")
			}
			outline := Adapter().Outline(editor.Host{Buffer: f.buffer})
			snapshot, err := outline.OutlineSnapshot()
			if err != nil || len(snapshot.Items) == 0 {
				t.Fatalf("fixture outline missing: %+v %v", snapshot, err)
			}
			id := snapshot.Items[0].ID
			list, outlineCaret := f.view.list, f.view.outlineCaret
			outlineCurrent := f.view.outlineCurrent
			revision, oldSelection, oldHistory := f.buffer.Version(), f.buffer.Selection(), f.buffer.History()
			sameOriginal := draftOwnerBufferReceipt(f.buffer)

			const committedValue = "已写🙂"
			want := strings.Replace(draftOwnerSource, "\r\n\r\n末行", "\r\n\r\n"+committedValue, 1)
			f.tester.Command("selectAll")
			f.tester.Compose(committedValue, 3)
			f.tester.Frame()
			if _, active := old.MarkedRange(); !active || old.Buffer.String() != committedValue {
				t.Fatalf("fixture needs a real single-selection composition: active=%v value=%q", active, old.Buffer.String())
			}
			sameOriginal(t)
			if f.changes != 0 || len(f.reports) != 0 {
				t.Fatalf("preedit wrote or reported before Flush: changes=%d reports=%v", f.changes, f.reports)
			}

			var successor *editor.Editor
			var sameCommittedMain, sameSuccessor func(*testing.T)
			var committedSelection text.Range
			f.onceChanged = func() {
				if f.view.draft != old || f.view.cellEdit != nil {
					t.Fatalf("Changed did not run while the ordinary draft still owned the editor: draft=%p cell=%#v", f.view.draft, f.view.cellEdit)
				}
				if f.buffer.String() != want || f.buffer.Version() != revision+1 ||
					f.buffer.History().UndoDepth != oldHistory.UndoDepth+1 || f.buffer.History().RedoDepth != 0 || f.buffer.InGroup() {
					t.Fatalf("Changed did not see one committed ordinary-block write: source=%q version=%d history=%+v", f.buffer.String(), f.buffer.Version(), f.buffer.History())
				}
				committedSelection = f.buffer.Selection()
				sameCommittedMain = draftOwnerBufferReceipt(f.buffer)
				successor = f.installSuccessor(t)
				sameSuccessor = draftOwnerBufferReceipt(successor.Buffer)
			}

			var actionErr error
			if action == "outline" {
				actionErr = f.view.NavigateOutlineItem(id)
			} else {
				actionErr = f.view.Flush()
			}
			f.requireSuccessor(t, old, successor)
			if sameCommittedMain == nil || sameSuccessor == nil {
				t.Fatal("Changed did not synchronously install and record the successor")
			}
			successorHistory := successor.Buffer.History()
			if successorHistory.UndoDepth == 0 || successorHistory.RedoDepth != 0 {
				t.Fatalf("successor did not keep its independent edit history: %+v", successorHistory)
			}
			sameCommittedMain(t)
			if f.changes != 1 || f.view.list != list || f.view.outlineCurrent != outlineCurrent ||
				f.view.outlineCaret != outlineCaret {
				t.Fatalf("old action advanced after Changed: changes=%d outline=%q caret=%d", f.changes, f.view.outlineCurrent, f.view.outlineCaret)
			}
			if action == "outline" {
				wantError := "原块已写回：" + errDraftSuperseded.Error()
				if !errors.Is(actionErr, errDraftSuperseded) || actionErr.Error() != wantError ||
					len(f.reports) != 1 || f.reports[0] != actionErr {
					t.Fatalf("outer outline omitted the committed block write: err=%v reports=%v", actionErr, f.reports)
				}
			} else if actionErr != errDraftSuperseded || len(f.reports) != 1 || f.reports[0] != errDraftSuperseded {
				t.Fatalf("public Flush result shape changed: err=%v reports=%v", actionErr, f.reports)
			}

			if !f.buffer.Undo() || f.buffer.String() != draftOwnerSource || f.buffer.Selection() != oldSelection ||
				f.buffer.History().UndoDepth != oldHistory.UndoDepth || f.buffer.History().RedoDepth != 1 {
				t.Fatalf("one undo did not restore original source/selection/history: source=%q history=%+v", f.buffer.String(), f.buffer.History())
			}
			sameSuccessor(t)
			f.tester.Frame()
			f.requireFocus(t, draftOwnerSecondField)
			if !f.buffer.Redo() || f.buffer.String() != want || f.buffer.Selection() != committedSelection ||
				f.buffer.History().UndoDepth != oldHistory.UndoDepth+1 || f.buffer.History().RedoDepth != 0 {
				t.Fatalf("one redo did not restore only the old write: source=%q history=%+v", f.buffer.String(), f.buffer.History())
			}
			sameSuccessor(t)
			f.tester.Frame()
			f.requireFocus(t, draftOwnerSecondField)
			f.capture(t, draftOwnerNewText, draftOwnerNewSelection)
			if f.changes != 1 {
				t.Fatalf("undo/redo emitted extra Changed callbacks: %d", f.changes)
			}
		})
	}
}

func TestMarkdownDraftOwnerOrdinaryBlockNoFalseCommitReceipt(t *testing.T) {
	t.Run("clean", func(t *testing.T) {
		f := newDraftOwnerFixture(t)
		f.openBlock(t, "末行")
		sameMain := draftOwnerBufferReceipt(f.buffer)
		receipt, err := f.view.flushDraft(true)
		if err != nil || receipt.blockCommitted || receipt.cellCommitted {
			t.Fatalf("clean ordinary draft produced a write receipt: receipt=%+v err=%v", receipt, err)
		}
		outline := Adapter().Outline(editor.Host{Buffer: f.buffer})
		snapshot, err := outline.OutlineSnapshot()
		if err != nil || len(snapshot.Items) == 0 {
			t.Fatalf("fixture outline missing: %+v %v", snapshot, err)
		}
		if err := f.view.NavigateOutlineItem(snapshot.Items[0].ID); err != nil || f.view.outlineCurrent != snapshot.Items[0].ID {
			t.Fatalf("clean draft did not permit navigation: current=%q err=%v", f.view.outlineCurrent, err)
		}
		sameMain(t)
		if f.view.draft != nil || f.changes != 0 || len(f.reports) != 0 {
			t.Fatalf("clean ordinary navigation wrote or reported: changes=%d reports=%v", f.changes, f.reports)
		}
	})

	t.Run("rejected-replace", func(t *testing.T) {
		buffer := text.NewWithLimit(draftOwnerSource, len(draftOwnerSource))
		f := newDraftOwnerFixtureWithBuffer(t, buffer)
		old := f.openBlock(t, "末行")
		outline := Adapter().Outline(editor.Host{Buffer: f.buffer})
		snapshot, err := outline.OutlineSnapshot()
		if err != nil || len(snapshot.Items) == 0 {
			t.Fatalf("fixture outline missing: %+v %v", snapshot, err)
		}
		id := snapshot.Items[0].ID
		sameMain := draftOwnerBufferReceipt(f.buffer)
		f.tester.Command("selectAll")
		f.tester.Compose("受限🙂", 3)
		f.tester.Frame()
		if _, active := old.MarkedRange(); !active || old.Buffer.String() != "受限🙂" {
			t.Fatalf("fixture needs a real composition before the rejected write: active=%v value=%q", active, old.Buffer.String())
		}
		sameMain(t)

		var successor *editor.Editor
		f.onceReport = func(err error) {
			if !errors.Is(err, text.ErrSizeLimit) || strings.Contains(err.Error(), "原块已写回") {
				t.Fatalf("failed ReplaceUTF16 claimed a committed write: %v", err)
			}
			successor = f.installSuccessor(t)
		}
		actionErr := f.view.NavigateOutlineItem(id)
		f.requireSuccessor(t, old, successor)
		sameMain(t)
		if len(f.reports) != 1 || !errors.Is(actionErr, text.ErrSizeLimit) || actionErr != f.reports[0] ||
			!errors.Is(f.reports[0], text.ErrSizeLimit) {
			t.Fatalf("refused write fabricated a receipt or advanced the action: err=%v reports=%v", actionErr, f.reports)
		}
		for _, report := range f.reports {
			if strings.Contains(report.Error(), "原块已写回") {
				t.Fatalf("rejected write reported a commit receipt: %v", report)
			}
		}
		if f.changes != 0 || f.view.outlineCurrent != "" {
			t.Fatalf("rejected write changed the document or advanced outline: changes=%d outline=%q", f.changes, f.view.outlineCurrent)
		}
	})
}
