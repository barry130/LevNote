package markdownview

import (
	"strings"
	"testing"
	"time"

	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

// The exact Task141 fixture supplies real source/Editor callback seams. Every
// pointer and queued Clicked below comes from Tester; callbacks never run a frame.
func TestMarkdownReferenceReleaseReentryClicked(t *testing.T) {
	f := newMarkdownActionFixture(t, false)
	old := f.openCell(t, "已写🙂")
	info := referenceAccess(t, f.tester, true)
	x, y := info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2
	f.tester.Press(x, y)
	pressed := f.view.referenceActivation
	list := f.view.list
	if pressed.buffer != f.buffer || pressed.rowID == 0 || f.view.draft != old {
		t.Fatal("actual pointer-down did not capture the visible reference beside the dirty cell")
	}
	version, selection, history := f.buffer.Version(), f.buffer.Selection(), f.buffer.History()
	sameOld := markdownActionBufferReceipt(old.Buffer)
	want := strings.Replace(markdownActionSource, "| 甲 |", "| 已写🙂 |", 1)
	end := text.UTF16Len(markdownActionSource[:strings.Index(markdownActionSource, "| 甲 |")+2]) + text.UTF16Len("已写🙂")
	committed := text.Range{Anchor: end, Head: end}
	var successor *editor.Editor
	var sameMain, sameOwner, sameSuccessor func(*testing.T)
	releaseReturned := false
	f.onChanged = func() {
		if len(f.reports) != 1 || f.reports[0] != errReferenceStale || f.changes != 1 || f.view.draft != nil ||
			f.buffer.String() != want || f.buffer.Version() != version+1 || f.buffer.Selection() != committed ||
			f.buffer.History().UndoDepth != history.UndoDepth+1 || f.buffer.History().RedoDepth != 0 || f.buffer.InGroup() {
			t.Fatal("Release Changed did not see exactly the completed old-cell commit after the stale report")
		}
		sameMain = markdownActionBufferReceipt(f.buffer)
		successor = f.installSuccessor(t)
		sameOwner = f.ownerReceipt(t, successor)
		sameSuccessor = markdownActionBufferReceipt(successor.Buffer)
		if successor == old || f.view.host.Buffer != pressed.buffer || f.view.list != list ||
			captureMarkdownAction(t, f.view, "reference").reference.rowID != pressed.rowID {
			t.Fatal("same-buffer callback did not preserve the old list and unchanged reference row identity")
		}
	}
	f.onReport = func(err error) {
		if err != errReferenceStale || f.view.referenceActivation.buffer != nil || f.view.referencePointerUpBuffer != nil ||
			!f.buffer.InGroup() || f.buffer.String() != markdownActionSource || f.buffer.Version() != version ||
			f.buffer.Selection() != selection || f.buffer.History() != history || f.changes != 0 {
			t.Fatal("actual stale PointerUp did not retire its gesture before Report or changed the foreign group")
		}
		if f.buffer.EndGroup() || f.buffer.InGroup() {
			t.Fatal("callback did not end the unchanged caller-owned group")
		}
		f.view.Release()
		releaseReturned = true
		if sameMain == nil || sameOwner == nil || sameSuccessor == nil {
			t.Fatal("Release did not synchronously reach the real Changed successor seam")
		}
		sameMain(t)
		sameOwner(t)
		sameSuccessor(t)
	}
	// BeginGroup after the real down makes referenceMapped refuse PointerUp.
	// It changes no bytes and leaves the independent table target mappable once
	// Report ends the group. No intervening Tester frame replaces the handler.
	if err := f.buffer.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	f.tester.Release(x, y) // MyGo delivers PointerUp, queues Clicked, then settles.
	if !releaseReturned || successor == nil || f.changes != 1 || len(f.opened) != 0 ||
		len(f.reports) != 1 || f.reports[0] != errReferenceStale {
		t.Fatalf("old queued Clicked duplicated/continued the stale release: changes=%d reports=%v opened=%v", f.changes, f.reports, f.opened)
	}
	// Public cell Flush returns nil here: its clear succeeded before Changed.
	// Thus the complete report inventory is one stale object, zero independent
	// Flush/supersession reports; action-receipt semantics are not involved.
	f.focusSuccessor(t) // an additional actual frame after the queued click
	sameMain(t)
	sameOwner(t)
	sameSuccessor(t)
	sameOld(t)
	if old.OnChange != nil || f.view.referenceActivation.buffer != nil || f.view.referencePointerUpBuffer != nil ||
		f.view.CompositionActive() || len(f.reports) != 1 || len(f.opened) != 0 || f.changes != 1 {
		t.Fatal("old Release/input tail retained an old editor/gesture or changed the successor")
	}
	if _, active := old.MarkedRange(); active {
		t.Fatal("retired old cell retained composition")
	}
	if _, active := successor.MarkedRange(); active {
		t.Fatal("old tail introduced successor composition")
	}
	if active, noIME := f.tester.TextInputActive(); !active || noIME {
		t.Fatal("successor lost its real native text/IME client after the old queued click")
	}
	if _, active := f.tester.TextCaret(); !active {
		t.Fatal("successor lost the native IME caret")
	}
	if !f.buffer.Undo() || f.buffer.String() != markdownActionSource || f.buffer.Selection() != selection ||
		f.buffer.History().UndoDepth != history.UndoDepth || f.buffer.History().RedoDepth != 1 {
		t.Fatal("main undo lost the exact old-cell bytes, directed selection or single history step")
	}
	sameSuccessor(t)
	if !f.buffer.Redo() || f.buffer.String() != want || f.buffer.Selection() != committed ||
		f.buffer.History().UndoDepth != history.UndoDepth+1 || f.buffer.History().RedoDepth != 0 {
		t.Fatal("main redo did not restore exactly the old-cell commit")
	}
	sameSuccessor(t)
	f.independentSuccessorHistory(t, successor)
	if f.changes != 1 || len(f.reports) != 1 || len(f.opened) != 0 {
		t.Fatal("independent successor history leaked an old action or report")
	}
	// Explicitly discard the independently verified successor before testing a
	// new gesture: its multiline recovery text is intentionally not a valid cell.
	f.view.DiscardDraft()
	f.tester.Frame()
	sameAfter := markdownActionBufferReceipt(f.buffer)
	f.tester.Advance(time.Second)
	clickReference(t, f.tester, referenceAccess(t, f.tester, true))
	if len(f.reports) != 1 || len(f.opened) != 1 || f.opened[0] != "target.txt" || f.changes != 1 {
		t.Fatal("old refusal receipt consumed a later successful actual gesture")
	}
	sameAfter(t)
}

func TestMarkdownReferenceReleaseClickedFullControls(t *testing.T) {
	for _, state := range []string{"no-draft", "changed-cell", "failed-cell-flush"} {
		t.Run(state, func(t *testing.T) {
			f := newMarkdownActionFixture(t, false)
			want, changes, reports := markdownActionSource, 0, 1
			if state == "changed-cell" {
				f.openCell(t, "已写🙂")
				want, changes = strings.Replace(want, "| 甲 |", "| 已写🙂 |", 1), 1
			} else if state == "failed-cell-flush" {
				f.openCell(t, "坏|值")
				reports = 2 // stale pointer + independently owned public Flush refusal
			}
			info := referenceAccess(t, f.tester, true)
			x, y := info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2
			f.tester.Press(x, y)
			pressed, list := f.view.referenceActivation, f.view.list
			if pressed.buffer != f.buffer || pressed.rowID == 0 {
				t.Fatal("control did not capture a real reference down")
			}
			version, history := f.buffer.Version(), f.buffer.History()
			released := false
			f.onReport = func(err error) {
				if err != errReferenceStale || !f.buffer.InGroup() || f.buffer.EndGroup() || f.buffer.InGroup() {
					t.Fatal("control callback did not retire its unchanged foreign group")
				}
				f.view.Release()
				released = true
				if f.view.list == list || len(f.view.rows) != 0 || f.view.loaded || f.view.docBuffer != nil {
					t.Fatal("normal full Release did not invalidate old list/row identities")
				}
			}
			if err := f.buffer.BeginGroup(); err != nil {
				t.Fatal(err)
			}
			f.tester.Release(x, y)
			f.tester.Frame()
			if !released || len(f.reports) != reports || f.reports[0] != errReferenceStale ||
				state == "failed-cell-flush" && f.reports[1] != errTableCellPipe ||
				f.changes != changes || len(f.opened) != 0 || f.buffer.String() != want ||
				f.buffer.Version() != version+uint64(changes) || f.buffer.InGroup() ||
				f.buffer.History().UndoDepth != history.UndoDepth+changes || f.buffer.History().RedoDepth != 0 ||
				captureMarkdownAction(t, f.view, "reference").reference.rowID == pressed.rowID {
				t.Fatalf("full Release kept an old click or changed Flush semantics: reports=%v changes=%d opened=%v", f.reports, f.changes, f.opened)
			}
			if state == "failed-cell-flush" {
				if f.view.draft == nil || f.view.cellEdit == nil || f.view.draft.Buffer.String() != "坏|值" || f.view.draftError != errTableCellPipe {
					t.Fatal("failed public Flush did not retain its exact recovery draft/error")
				}
				f.view.DiscardDraft()
				f.tester.Frame()
			} else if f.view.draft != nil {
				t.Fatal("successful full Release retained a draft")
			}
			sameMain := markdownActionBufferReceipt(f.buffer)
			f.tester.Advance(time.Second)
			clickReference(t, f.tester, referenceAccess(t, f.tester, true))
			if len(f.reports) != reports || f.changes != changes || len(f.opened) != 1 || f.opened[0] != "target.txt" {
				t.Fatal("full Release leaked a receipt into a fresh successful gesture")
			}
			sameMain(t)
		})
	}
}
