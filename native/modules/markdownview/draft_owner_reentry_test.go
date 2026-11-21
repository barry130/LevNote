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

const (
	draftOwnerSource      = "\uFEFF<!-- opaque: key=🙂 -->\r\n\r\n# 小节🙂\r\n\r\n| A | B |\r\n|---|---|\r\n| 甲 | 乙 |\r\n\r\n末行"
	draftOwnerFirstCell   = "数据，第 2 行，第 1 列：甲"
	draftOwnerSecondCell  = "数据，第 2 行，第 2 列：乙"
	draftOwnerFirstField  = "编辑单元格 · 数据 · 第 2 行 · 第 1 列"
	draftOwnerSecondField = "编辑单元格 · 数据 · 第 2 行 · 第 2 列"
	draftOwnerOldText     = "原🙂草稿"
	draftOwnerNewText     = "新\r\n留🙂"
)

var draftOwnerNewSelection = text.Range{Anchor: 6, Head: 4}

type draftOwnerFixture struct {
	buffer      *text.Buffer
	view        *View
	tester      *ui.Tester
	notices     int
	changes     int
	reports     []error
	copies      []struct{ title, value string }
	onceNotice  func()
	onceChanged func()
	onceReport  func(error)
	recoverCopy func() error
}

func newDraftOwnerFixture(t *testing.T) *draftOwnerFixture {
	return newDraftOwnerFixtureWithBuffer(t, text.New(draftOwnerSource))
}

func newDraftOwnerFixtureWithBuffer(t *testing.T, buffer *text.Buffer) *draftOwnerFixture {
	t.Helper()
	f := &draftOwnerFixture{buffer: buffer}
	if err := f.buffer.SetSelection(f.buffer.LenUTF16(), 2); err != nil {
		t.Fatal(err)
	}
	f.view = Adapter().Create(editor.Host{
		Buffer: f.buffer,
		Path:   "owner.md",
		Changed: func() {
			f.changes++
			if call := f.onceChanged; call != nil {
				f.onceChanged = nil
				call()
			}
		},
		Report: func(err error) {
			f.reports = append(f.reports, err)
			if call := f.onceReport; call != nil {
				f.onceReport = nil
				call(err)
			}
		},
		DraftRecoveryChanged: func(owner editor.View) {
			if owner != f.view {
				t.Fatalf("recovery notification owner = %T %p", owner, owner)
			}
			f.notices++
			if call := f.onceNotice; call != nil {
				f.onceNotice = nil
				call()
			}
		},
		RecoverDraft: func(title, value string) error {
			f.copies = append(f.copies, struct{ title, value string }{title, value})
			if call := f.recoverCopy; call != nil {
				f.recoverCopy = nil
				return call()
			}
			return nil
		},
	}).(*View)
	f.tester = ui.NewTester(f.view.Draw, 760, 760)
	f.tester.Frame()
	return f
}

func (f *draftOwnerFixture) open(t *testing.T, cell, field string) *editor.Editor {
	t.Helper()
	f.tester.Frame()
	f.tester.Advance(time.Second)
	openCellByPointer(t, f.tester, ui.RoleCell, cell)
	f.requireFocus(t, field)
	if f.view.draft == nil || f.view.cellEdit == nil {
		t.Fatal("fixture did not create an independent cell editor")
	}
	return f.view.draft
}

func (f *draftOwnerFixture) openBlock(t *testing.T, label string) *editor.Editor {
	t.Helper()
	f.tester.Frame()
	rect, ok := f.tester.Find(label)
	if !ok {
		t.Fatalf("ordinary block not rendered: %q %#v", label, f.tester.Texts())
	}
	f.tester.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	f.tester.Frame()
	rect, ok = f.tester.Find(label)
	if !ok {
		t.Fatalf("ordinary block disappeared after first click: %q %#v", label, f.tester.Texts())
	}
	f.tester.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	f.tester.Frame()
	if f.view.draft == nil || f.view.cellEdit != nil || f.view.draft.Buffer.String() != label {
		t.Fatalf("actual double-click did not open the ordinary block: draft=%p cell=%#v", f.view.draft, f.view.cellEdit)
	}
	f.requireAnyTextFieldFocus(t)
	return f.view.draft
}

func (f *draftOwnerFixture) requireAnyTextFieldFocus(t *testing.T) {
	t.Helper()
	info, ok := f.tester.AccessByRole(ui.RoleTextField, "")
	if !ok || !info.Focused || info.Disabled || info.Offscreen {
		t.Fatalf("ordinary draft is not an accessible focused text field: %#v texts=%q", info, f.tester.Texts())
	}
}

func (f *draftOwnerFixture) requireFocus(t *testing.T, field string) {
	t.Helper()
	info, ok := f.tester.AccessByRole(ui.RoleTextField, field)
	if !ok || !info.Focused || info.Disabled || info.Offscreen {
		t.Fatalf("editable focused field %q missing: %#v texts=%q", field, info, f.tester.Texts())
	}
}

func (f *draftOwnerFixture) replaceText(value string) {
	f.tester.Command("selectAll")
	f.tester.Type(value)
	f.tester.Frame()
}

func (f *draftOwnerFixture) capture(t *testing.T, value string, selection text.Range) {
	t.Helper()
	got, ok, err := f.view.CaptureSessionDraft(maxConfirmedCellDraftBytes)
	if err != nil || !ok || got.Title != "owner.md-表格单元格草稿.txt" || got.Content != value ||
		got.Anchor != selection.Anchor || got.Head != selection.Head {
		t.Fatalf("session draft = %#v ok=%v err=%v, want %q selection=%+v", got, ok, err, value, selection)
	}
}

func (f *draftOwnerFixture) noCapture(t *testing.T) {
	t.Helper()
	got, ok, err := f.view.CaptureSessionDraft(maxConfirmedCellDraftBytes)
	if err != nil || ok || got != (editor.SessionDraft{}) {
		t.Fatalf("cleared draft still recoverable: %#v ok=%v err=%v", got, ok, err)
	}
}

func (f *draftOwnerFixture) installSuccessor(t *testing.T) *editor.Editor {
	t.Helper()
	// Reenter the public discard before driving another real cell. That is
	// also how the callback releases the old editor on the buggy baseline.
	f.view.DiscardDraft()
	f.noCapture(t)
	newDraft := f.open(t, draftOwnerSecondCell, draftOwnerSecondField)
	f.replaceText(draftOwnerNewText)
	f.tester.Key(ui.Shift, ui.KeyLeft)
	f.tester.Frame()
	f.capture(t, draftOwnerNewText, draftOwnerNewSelection)
	return newDraft
}

func (f *draftOwnerFixture) requireSuccessor(t *testing.T, old, successor *editor.Editor) {
	t.Helper()
	if successor == nil || old == successor || f.view.draft != successor || f.view.recoveryDraft != successor {
		t.Fatalf("outer operation lost the successor: old=%p successor=%p draft=%p recovery=%p", old, successor, f.view.draft, f.view.recoveryDraft)
	}
	// Check before another frame could rebuild a preview reset by a stale tail.
	if !f.view.loaded || f.view.docBuffer != f.buffer || len(f.view.rows) == 0 ||
		f.view.cellEdit == nil || f.view.cellEdit.label != draftOwnerSecondField || f.view.draftError != nil {
		t.Fatalf("outer operation overwrote successor preview, panel or issue: loaded=%v rows=%d cell=%#v issue=%v", f.view.loaded, len(f.view.rows), f.view.cellEdit, f.view.draftError)
	}
	if successor.Buffer.String() != draftOwnerNewText || successor.Buffer.Selection() != draftOwnerNewSelection {
		t.Fatalf("successor editor lost its text or reverse selection: %q %+v", successor.Buffer.String(), successor.Buffer.Selection())
	}
	f.capture(t, draftOwnerNewText, draftOwnerNewSelection)
	f.tester.Frame()
	f.requireFocus(t, draftOwnerSecondField)
	if _, exists := f.tester.AccessByRole(ui.RoleTextField, draftOwnerFirstField); exists {
		t.Fatal("old cell editor remained mounted")
	}
	f.capture(t, draftOwnerNewText, draftOwnerNewSelection)
}

func draftOwnerBufferReceipt(buffer *text.Buffer) func(*testing.T) {
	value, version, history, selection := buffer.String(), buffer.Version(), buffer.History(), buffer.Selection()
	return func(t *testing.T) {
		t.Helper()
		if buffer.String() != value || buffer.Version() != version || buffer.History() != history || buffer.Selection() != selection {
			t.Fatalf("main buffer changed: source=%q version=%d history=%+v selection=%+v", buffer.String(), buffer.Version(), buffer.History(), buffer.Selection())
		}
	}
}

func draftOwnerHeaderTarget(t *testing.T, v *View) tableCellTarget {
	t.Helper()
	preview := previewSnapshot{v.doc, v.rows, v.docBuffer, v.version}
	for _, row := range preview.rows {
		if row.tableRow != 0 || row.block < 0 || row.block >= len(preview.doc.Blocks) {
			continue
		}
		if target, ok := tableCellTargetOf(preview, row, preview.doc.Blocks[row.block], 0); ok {
			if captured, ok := captureTableCellTarget(preview, target); ok {
				return captured
			}
		}
	}
	t.Fatal("fixture header cell target missing")
	return tableCellTarget{}
}

func TestMarkdownDraftOwnerDiscardReentry(t *testing.T) {
	f := newDraftOwnerFixture(t)
	sameMain := draftOwnerBufferReceipt(f.buffer)
	old := f.open(t, draftOwnerFirstCell, draftOwnerFirstField)
	f.replaceText(draftOwnerOldText)
	f.capture(t, draftOwnerOldText, text.Range{Anchor: 5, Head: 5})
	f.tester.Compose("临🙂", 2)
	f.tester.Frame()
	if _, active := old.MarkedRange(); !active {
		t.Fatal("fixture needs a composing old editor")
	}
	var successor *editor.Editor
	f.onceNotice = func() { successor = f.installSuccessor(t) }
	f.view.DiscardDraft()
	f.requireSuccessor(t, old, successor)
	if _, active := old.MarkedRange(); active {
		t.Fatal("original editor was detached without retiring its composition")
	}
	sameMain(t)
	if f.changes != 0 || len(f.reports) != 0 {
		t.Fatalf("discard dispatched a write or error: changes=%d reports=%v", f.changes, f.reports)
	}
}

func TestMarkdownDraftOwnerFinishAndCallerReentry(t *testing.T) {
	for _, action := range []string{"finish", "open-cell", "open-block", "add-paragraph", "release", "outline"} {
		t.Run(action, func(t *testing.T) {
			f := newDraftOwnerFixture(t)
			sameMain := draftOwnerBufferReceipt(f.buffer)
			old := f.open(t, draftOwnerFirstCell, draftOwnerFirstField)
			want, wantChanges := draftOwnerSource, 0
			if action != "open-cell" {
				f.replaceText("已写🙂")
				want, wantChanges = strings.Replace(want, "| 甲 |", "| 已写🙂 |", 1), 1
			}
			list := f.view.list
			var successor *editor.Editor
			var sameAfterClose func(*testing.T)
			f.onceNotice = func() {
				if f.buffer.String() != want || f.buffer.History().UndoDepth != wantChanges {
					t.Fatalf("unexpected main write at close: source=%q history=%+v", f.buffer.String(), f.buffer.History())
				}
				sameAfterClose = draftOwnerBufferReceipt(f.buffer)
				successor = f.installSuccessor(t)
			}
			// Invoke the same finish/open handlers outside a Tester build. The
			// callback itself opens and types through the public Tester controls.
			var actionErr error
			switch action {
			case "finish":
				f.view.commit()
			case "open-cell":
				actionErr = f.view.activateTableCell(draftOwnerHeaderTarget(t, f.view))
			case "open-block":
				f.view.begin(0, f.view.doc.Blocks[0])
			case "add-paragraph":
				f.view.addParagraph()
			case "release":
				f.view.Release()
			case "outline":
				outline := Adapter().Outline(editor.Host{Buffer: f.buffer})
				item, err := outline.OutlineSnapshot()
				if err != nil || len(item.Items) == 0 {
					t.Fatalf("fixture outline missing: %+v %v", item, err)
				}
				actionErr = f.view.NavigateOutlineItem(item.Items[0].ID)
			}
			f.requireSuccessor(t, old, successor)
			sameAfterClose(t)
			if f.view.list != list || f.changes != wantChanges || len(f.reports) != 1 || !strings.Contains(f.reports[0].Error(), "草稿状态已在回调中改变") {
				t.Fatalf("stale result or missing refusal: sameList=%v changes=%d reports=%v", f.view.list == list, f.changes, f.reports)
			}
			if (action == "open-cell" || action == "outline") && (actionErr != f.reports[0] || f.view.outlineCurrent != "") {
				t.Fatalf("interrupted action continued: item=%q err=%v", f.view.outlineCurrent, actionErr)
			}
			if wantChanges == 0 {
				sameMain(t)
			} else {
				if !strings.Contains(f.reports[0].Error(), "原单元格已写回") || !f.buffer.Undo() || f.buffer.String() != draftOwnerSource {
					t.Fatalf("partial finish did not report the exact undoable write: source=%q reports=%v", f.buffer.String(), f.reports)
				}
			}
		})
	}
}

func TestMarkdownDraftOwnerRecoverReentry(t *testing.T) {
	for _, mode := range []string{"success-clear", "error-replacement", "error-retains"} {
		t.Run(mode, func(t *testing.T) {
			f := newDraftOwnerFixture(t)
			sameMain := draftOwnerBufferReceipt(f.buffer)
			old := f.open(t, draftOwnerFirstCell, draftOwnerFirstField)
			f.replaceText(draftOwnerOldText)
			before := f.notices
			var successor *editor.Editor
			var failure error
			if mode == "success-clear" {
				f.onceNotice = func() { successor = f.installSuccessor(t) }
			} else {
				failure = errors.New("recovery copy refused")
				f.recoverCopy = func() error {
					if mode == "error-replacement" {
						successor = f.installSuccessor(t)
					}
					return failure
				}
			}
			if err := f.view.RecoverDraft(); err != failure {
				t.Fatalf("recovery result = %v, want %v", err, failure)
			}
			if len(f.copies) != 1 || f.copies[0].title != "owner.md-块草稿.txt" || f.copies[0].value != draftOwnerOldText {
				t.Fatalf("host copied the wrong payload: %+v", f.copies)
			}
			if failure != nil && (len(f.reports) != 1 || f.reports[0] != failure) || failure == nil && len(f.reports) != 0 {
				t.Fatalf("unexpected recovery reporting: %v", f.reports)
			}
			if mode == "error-retains" {
				if f.view.draft != old || f.view.draftError != failure || f.notices != before {
					t.Fatal("failed copy cleared the original draft, local issue or recovery notice")
				}
				f.capture(t, draftOwnerOldText, text.Range{Anchor: 5, Head: 5})
			} else {
				f.requireSuccessor(t, old, successor)
			}
			sameMain(t)
			if f.changes != 0 {
				t.Fatalf("recovery wrote the main document: changes=%d", f.changes)
			}
		})
	}
}

func TestMarkdownDraftOwnerClearNotification(t *testing.T) {
	for _, mode := range []string{"ordinary", "recursive-discard", "recursive-release"} {
		t.Run(mode, func(t *testing.T) {
			f := newDraftOwnerFixture(t)
			sameMain := draftOwnerBufferReceipt(f.buffer)
			f.view.DiscardDraft()
			if f.notices != 0 {
				t.Fatal("no-draft discard notified the host")
			}
			f.open(t, draftOwnerFirstCell, draftOwnerFirstField)
			f.replaceText(draftOwnerOldText)
			before := f.notices
			f.onceNotice = func() {
				f.noCapture(t)
				if mode == "recursive-discard" {
					f.view.DiscardDraft()
				} else if mode == "recursive-release" {
					f.view.Release()
				}
			}
			f.view.DiscardDraft()
			f.view.DiscardDraft()
			f.noCapture(t)
			if f.notices != before+1 || f.view.draft != nil || f.view.recoveryDraft != nil || f.view.recoverySnapshotSet || f.changes != 0 || len(f.reports) != 0 {
				t.Fatalf("clear was not quiet after its one notice: notices=%d before=%d changes=%d reports=%v", f.notices, before, f.changes, f.reports)
			}
			sameMain(t)
		})
	}
}

func TestMarkdownDraftOwnerFailedReleaseReportReentry(t *testing.T) {
	f := newDraftOwnerFixture(t)
	sameMain := draftOwnerBufferReceipt(f.buffer)
	old := f.open(t, draftOwnerFirstCell, draftOwnerFirstField)
	f.replaceText("坏|值")
	var successor *editor.Editor
	f.onceReport = func(err error) {
		if !errors.Is(err, errTableCellPipe) {
			t.Fatalf("expected original write refusal, got %v", err)
		}
		successor = f.installSuccessor(t)
	}
	f.view.Release()
	f.requireSuccessor(t, old, successor)
	if len(f.reports) != 1 || !errors.Is(f.reports[0], errTableCellPipe) || f.changes != 0 {
		t.Fatalf("failed Release reported or wrote incorrectly: changes=%d reports=%v", f.changes, f.reports)
	}
	sameMain(t)
}

func TestMarkdownDraftOwnerCleanRebaseReentry(t *testing.T) {
	f := newDraftOwnerFixture(t)
	old := f.open(t, draftOwnerFirstCell, draftOwnerFirstField)
	start := text.UTF16Len(draftOwnerSource[:strings.Index(draftOwnerSource, "| 甲 |")+2])
	if err := f.buffer.ReplaceUTF16(start, start+1, "外部🙂"); err != nil {
		t.Fatal(err)
	}
	sameMain := draftOwnerBufferReceipt(f.buffer)
	var successor *editor.Editor
	f.onceNotice = func() { successor = f.installSuccessor(t) }
	_ = f.view.Warnings() // refresh lets the clean old draft yield to source
	f.requireSuccessor(t, old, successor)
	sameMain(t)
	if f.changes != 0 || len(f.reports) != 0 {
		t.Fatalf("clean rebase dispatched a write or error: changes=%d reports=%v", f.changes, f.reports)
	}
}

func TestMarkdownDraftOwnerRetainedEditorReleaseReportReentry(t *testing.T) {
	f := newDraftOwnerFixture(t)
	sameMain := draftOwnerBufferReceipt(f.buffer)
	old := f.open(t, draftOwnerFirstCell, draftOwnerFirstField)
	f.replaceText(draftOwnerOldText)
	// Use a valid directed set and installed native input to put the retained
	// editor into a multi-selection composition. Its ReleaseView reports it.
	if err := old.Buffer.SetSelectionSet(text.SelectionSet{
		Ranges:  []text.Range{{Anchor: 1, Head: 0}, {Anchor: 3, Head: 4}},
		Primary: 0,
	}); err != nil {
		t.Fatal(err)
	}
	f.tester.Compose("临🙂", 2)
	f.tester.Frame()
	if _, active := old.MarkedRange(); !active || !old.Buffer.InGroup() || len(f.reports) != 0 {
		t.Fatalf("fixture needs live accepted multi-selection input: active=%v reports=%v", active, f.reports)
	}
	if err := old.Flush(); !errors.Is(err, text.ErrCompositionPending) || len(f.reports) != 0 {
		t.Fatalf("retained editor did not enter pending multi-selection input: %v reports=%v", err, f.reports)
	}
	beforeNotices := f.notices
	var successor *editor.Editor
	f.onceReport = func(err error) {
		if !errors.Is(err, text.ErrCompositionPending) || f.view.draft != nil || f.view.cellEdit != nil || f.notices != beforeNotices+1 {
			t.Fatalf("release error arrived before the old draft detached and notified: err=%v notices=%d", err, f.notices)
		}
		f.noCapture(t)
		successor = f.installSuccessor(t)
	}
	// Discard does not flush or validate a cell. This report must come from
	// the captured editor's ReleaseView, after the recovery notice.
	f.view.DiscardDraft()
	f.requireSuccessor(t, old, successor)
	if len(f.reports) != 1 || !errors.Is(f.reports[0], text.ErrCompositionPending) || f.changes != 0 {
		t.Fatalf("releasing the old editor reported or wrote incorrectly: changes=%d reports=%v", f.changes, f.reports)
	}
	sameMain(t)
}
