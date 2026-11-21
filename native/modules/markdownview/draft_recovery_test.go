package markdownview

import (
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

func TestTableCellSessionDraftCapturePreservesConfirmedTextAndHost(t *testing.T) {
	buffer := text.New(tableViewSource)
	notices := 0
	v := Adapter().Create(editor.Host{
		Buffer:               buffer,
		DraftRecoveryChanged: func(editor.View) { notices++ },
	}).(*View)
	tester := ui.NewTester(v.Draw, 760, 480)
	tester.Frame()
	openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
	before := buffer.String()
	if draft, ok, err := v.CaptureSessionDraft(4 << 20); err != nil || ok || draft.Content != "" {
		t.Fatalf("clean cell capture=%#v ok=%v err=%v", draft, ok, err)
	}

	draft := v.draft
	draft.ReplaceText(&ui.TextInputRange{Start: 0, End: 1}, "乙")
	draft.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "临", ui.TextInputRange{Start: 1, End: 1})
	marked, markedActive := draft.MarkedRange()
	compositionState := draft.Buffer.Snapshot()
	compositionVersion, compositionHistory := draft.Buffer.Version(), draft.Buffer.History()
	captured, ok, err := v.CaptureSessionDraft(4 << 20)
	if err != nil || !ok || captured.Content != "乙" || captured.Anchor != 1 || captured.Head != 1 {
		t.Fatalf("composition capture=%#v ok=%v err=%v", captured, ok, err)
	}
	if buffer.String() != before || notices != 1 || !draft.Buffer.EqualSnapshot(compositionState) || draft.Buffer.Version() != compositionVersion || draft.Buffer.History() != compositionHistory {
		t.Fatalf("composition capture changed editor/host: hostChanged=%v notices=%d marked=%+v active=%v", buffer.String() != before, notices, marked, markedActive)
	}
	draft.UnmarkText()
	tester.Frame()
	captured, ok, err = v.CaptureSessionDraft(4 << 20)
	if err != nil || !ok || captured.Content != "临" {
		t.Fatalf("confirmed capture=%#v ok=%v err=%v", captured, ok, err)
	}
	if notices != 2 {
		t.Fatalf("composition end did not reschedule recovery: %d", notices)
	}
	v.DiscardDraft()
	if _, ok, err := v.CaptureSessionDraft(4 << 20); err != nil || ok {
		t.Fatalf("discarded cell remained recoverable: ok=%v err=%v", ok, err)
	}
}

func TestTableCellSessionDraftCaptureRejectsBudgetAndInvalidLimit(t *testing.T) {
	buffer := text.New(tableViewSource)
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 760, 480)
	tester.Frame()
	openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
	if err := v.draft.Buffer.ReplaceUTF16(0, v.draft.Buffer.LenUTF16(), "过长"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := v.CaptureSessionDraft(1); !errors.Is(err, errCellDraftRecoveryLimit) || ok {
		t.Fatalf("budget refusal ok=%v err=%v", ok, err)
	}
	if _, ok, err := v.CaptureSessionDraft(0); !errors.Is(err, errCellDraftRecoveryLimit) || ok {
		t.Fatalf("zero budget refusal ok=%v err=%v", ok, err)
	}
}

func TestTableCellSessionDraftCaptureSnapshotBudgetAndEmptyValue(t *testing.T) {
	buffer := text.New(tableViewSource)
	notices := 0
	v := Adapter().Create(editor.Host{Buffer: buffer, DraftRecoveryChanged: func(editor.View) { notices++ }}).(*View)
	tester := ui.NewTester(v.Draw, 760, 480)
	tester.Frame()
	openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
	draft := v.draft
	if err := draft.Buffer.ReplaceUTF16(0, 1, "乙"); err != nil {
		t.Fatal(err)
	}
	v.observeCellDraftConfirmation(draft)
	captured, ok, err := v.CaptureSessionDraft(maxConfirmedCellDraftBytes)
	if err != nil || !ok || captured.Content != "乙" {
		t.Fatalf("bounded confirmed value=%#v ok=%v err=%v", captured, ok, err)
	}
	confirmed := v.recoverySnapshot
	if err := draft.Buffer.ReplaceUTF16(0, draft.Buffer.LenUTF16(), strings.Repeat("x", maxConfirmedCellDraftBytes+1)); err != nil {
		t.Fatal(err)
	}
	v.observeCellDraftConfirmation(draft)
	if _, ok, err := v.CaptureSessionDraft(maxConfirmedCellDraftBytes); !errors.Is(err, errCellDraftRecoveryLimit) || ok {
		t.Fatalf("overflow capture ok=%v err=%v", ok, err)
	}
	if !v.recoveryOverflow || v.recoverySnapshot.String() != confirmed.String() || notices != 2 {
		t.Fatalf("overflow did not preserve prior snapshot: overflow=%v notices=%d", v.recoveryOverflow, notices)
	}
	if err := draft.Buffer.ReplaceUTF16(0, draft.Buffer.LenUTF16(), ""); err != nil {
		t.Fatal(err)
	}
	captured, ok, err = v.CaptureSessionDraft(maxConfirmedCellDraftBytes)
	if err != nil || !ok || captured.Content != "" {
		t.Fatalf("empty edited draft was lost: %#v ok=%v err=%v", captured, ok, err)
	}
}

func TestTableCellSessionDraftCaptureAcceptsExactFourMiBBoundary(t *testing.T) {
	buffer := text.New(tableViewSource)
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 760, 480)
	tester.Frame()
	openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
	value := strings.Repeat("x", maxConfirmedCellDraftBytes)
	if err := v.draft.Buffer.ReplaceUTF16(0, v.draft.Buffer.LenUTF16(), value); err != nil {
		t.Fatal(err)
	}
	v.observeCellDraftConfirmation(v.draft)
	captured, ok, err := v.CaptureSessionDraft(maxConfirmedCellDraftBytes)
	if err != nil || !ok || len(captured.Content) != maxConfirmedCellDraftBytes {
		t.Fatalf("exact 4 MiB capture length=%d ok=%v err=%v", len(captured.Content), ok, err)
	}
}

func TestTableCellSessionDraftCaptureSeparatesObservedAndConfirmedSelection(t *testing.T) {
	buffer := text.New(tableViewSource)
	notices := 0
	v := Adapter().Create(editor.Host{
		Buffer:               buffer,
		DraftRecoveryChanged: func(editor.View) { notices++ },
	}).(*View)
	tester := ui.NewTester(v.Draw, 760, 480)
	tester.Frame()
	openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
	draft := v.draft
	if err := draft.Buffer.ReplaceUTF16(0, 1, "待提交🙂"); err != nil {
		t.Fatal(err)
	}
	if err := draft.Buffer.SetSelection(5, 0); err != nil {
		t.Fatal(err)
	}
	v.observeCellDraftConfirmation(draft)
	confirmedNotices := notices
	if v.recoveryConfirmedSelection != (text.Range{Anchor: 5, Head: 0}) {
		t.Fatalf("reverse selection was not confirmed: %+v", v.recoveryConfirmedSelection)
	}
	draft.SetMarkedText(&ui.TextInputRange{Start: 1, End: 3}, "预", ui.TextInputRange{Start: 1, End: 1})
	liveText, liveSelection := draft.Buffer.Snapshot(), draft.Buffer.Selection()
	liveVersion, liveHistory := draft.Buffer.Version(), draft.Buffer.History()
	captured, ok, err := v.CaptureSessionDraft(maxConfirmedCellDraftBytes)
	if err != nil || !ok || captured.Content != "待提交🙂" || captured.Anchor != 5 || captured.Head != 0 {
		t.Fatalf("IME capture lost confirmed text/selection: %#v ok=%v err=%v", captured, ok, err)
	}
	if notices != confirmedNotices || !draft.Buffer.EqualSnapshot(liveText) || draft.Buffer.Selection() != liveSelection ||
		draft.Buffer.Version() != liveVersion || draft.Buffer.History() != liveHistory || buffer.String() != tableViewSource {
		t.Fatalf("readonly capture changed state or recursively notified host: notices=%d want=%d", notices, confirmedNotices)
	}
	draft.UnmarkText()
	tester.Frame()
	captured, ok, err = v.CaptureSessionDraft(maxConfirmedCellDraftBytes)
	if err != nil || !ok || captured.Content != "待预🙂" || captured.Anchor != 2 || captured.Head != 2 {
		t.Fatalf("post-IME confirmation did not update matching selection: %#v ok=%v err=%v", captured, ok, err)
	}
}

func TestTableCellSessionDraftCaptureCompositionBaselineAndUnmark(t *testing.T) {
	buffer := text.New(tableViewSource)
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 760, 480)
	tester.Frame()
	openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
	draft := v.draft
	if err := draft.Buffer.ReplaceUTF16(0, 1, "乙"); err != nil {
		t.Fatal(err)
	}
	v.observeCellDraftConfirmation(draft)
	draft.SetMarkedText(nil, "预", ui.TextInputRange{Start: 1, End: 1})
	captured, ok, err := v.CaptureSessionDraft(maxConfirmedCellDraftBytes)
	if err != nil || !ok || captured.Content != "乙" {
		t.Fatalf("composition should retain confirmed baseline: %#v ok=%v err=%v", captured, ok, err)
	}
	draft.UnmarkText()
	captured, ok, err = v.CaptureSessionDraft(maxConfirmedCellDraftBytes)
	if err != nil || !ok || captured.Content != "乙预" {
		t.Fatalf("Unmark without OnChange was not captured: %#v ok=%v err=%v", captured, ok, err)
	}
}
