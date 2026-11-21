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
	markdownActionSource        = "\uFEFF<!-- opaque: key=🙂 -->\r\n\r\n# 小节🙂\r\n\r\n| A | B |\r\n|---|---|\r\n| 甲 | 乙 |\r\n\r\n- [ ] 待办\r\n\r\n[目标](target.md) ![图](picture.png)\r\n\r\n```doc-block\r\nid: action\r\nkind: text\r\nmode: reference\r\nsrc: target.txt\r\n---\r\n载荷\r\n```\r\n\r\n末行"
	markdownActionFirstCell     = "数据，第 2 行，第 1 列：甲"
	markdownActionSecondField   = "编辑单元格 · 数据 · 第 2 行 · 第 2 列"
	markdownActionSuccessorText = "新\r\n留🙂"
	markdownActionSuperseded    = "草稿状态已在回调中改变，请确认当前草稿后重试"
)

var markdownActionReverse = text.Range{Anchor: 6, Head: 4}

type markdownActionFixture struct {
	buffer    *text.Buffer
	view      *View
	tester    *ui.Tester
	changes   int
	reports   []error
	opened    []string
	onChanged func()
	onReport  func(error)
	onOpen    func()
}

func newMarkdownActionFixture(t *testing.T, limited bool) *markdownActionFixture {
	t.Helper()
	buffer := text.New(markdownActionSource)
	if limited {
		buffer = text.NewWithLimit(markdownActionSource, len(markdownActionSource))
	}
	if err := buffer.SetSelection(buffer.LenUTF16(), 2); err != nil {
		t.Fatal(err)
	}
	f := &markdownActionFixture{buffer: buffer}
	f.view = Adapter().Create(editor.Host{
		Buffer: buffer, Path: "actions.md",
		Changed: func() {
			f.changes++
			if call := f.onChanged; call != nil {
				f.onChanged = nil
				call()
			}
		},
		Report: func(err error) {
			f.reports = append(f.reports, err)
			if call := f.onReport; call != nil {
				f.onReport = nil
				call(err)
			}
		},
		OpenReference: func(value string) {
			f.opened = append(f.opened, value)
			if call := f.onOpen; call != nil {
				f.onOpen = nil
				call()
			}
		},
	}).(*View)
	f.tester = ui.NewTester(f.view.Draw, 1180, 1200)
	f.tester.Frame()
	return f
}

type markdownActionCommand struct {
	kind      string
	task      taskTarget
	link      linkTarget
	reference referenceTarget
}

func captureMarkdownAction(t *testing.T, v *View, kind string) markdownActionCommand {
	t.Helper()
	command := markdownActionCommand{kind: kind}
	switch kind {
	case "task":
		command.task = captureTask(t, v, 0)
	case "link":
		command.link = captureLink(t, v, 0)
	case "image":
		command.link = captureImageReference(t, v, 0)
	case "reference":
		v.refresh()
		preview := previewSnapshot{v.doc, v.rows, v.docBuffer, v.version}
		for _, row := range preview.rows {
			block := preview.doc.Blocks[row.block]
			if target, ok := referenceTargetOf(preview, row, block); ok {
				command.reference = target
				return command
			}
		}
		t.Fatal("fixture has no doc-block reference target")
	default:
		t.Fatalf("unknown action %q", kind)
	}
	return command
}

func (command markdownActionCommand) run(v *View) error {
	switch command.kind {
	case "task":
		return v.toggleTask(command.task)
	case "link", "image":
		return v.activateLink(command.link)
	default:
		return v.activateReference(command.reference)
	}
}

func (command markdownActionCommand) click(t *testing.T, f *markdownActionFixture) {
	t.Helper()
	// These are the real checkbox/inline-link/reference-button UI wrappers.
	// Callback code below never invokes this Tester or builds another frame.
	switch command.kind {
	case "task":
		info := taskAccess(t, f.tester, "未完成任务：待办")
		f.tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	case "link":
		clickLink(t, f.tester, "目标")
	case "image":
		clickImageReference(t, f.tester, "图")
	case "reference":
		info := referenceAccess(t, f.tester, true)
		f.tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	}
}

func (f *markdownActionFixture) openCell(t *testing.T, value string) *editor.Editor {
	t.Helper()
	openCellByPointer(t, f.tester, ui.RoleCell, markdownActionFirstCell)
	old := f.view.draft
	if old == nil || f.view.cellEdit == nil || old.Buffer.String() != "甲" {
		t.Fatal("real pointer did not open the original cell")
	}
	if value != "甲" {
		f.tester.Command("selectAll")
		f.tester.Type(value)
		f.tester.Frame()
	}
	if old.Buffer.String() != value || f.buffer.String() != markdownActionSource || f.changes != 0 || len(f.reports) != 0 {
		t.Fatalf("cell input was not held independently: draft=%q main=%q changes=%d reports=%v", old.Buffer.String(), f.buffer.String(), f.changes, f.reports)
	}
	return old
}

func (f *markdownActionFixture) openOrdinary(t *testing.T, compose bool) *editor.Editor {
	t.Helper()
	beginDraft(t, f.view, f.tester, "末行")
	old := f.view.draft
	if old == nil || f.view.cellEdit != nil || old.Buffer.String() != "末行" {
		t.Fatal("real double-click did not open the ordinary draft")
	}
	if compose {
		f.tester.Command("selectAll")
		f.tester.Compose("已写🙂", 3)
		f.tester.Frame()
		if _, active := old.MarkedRange(); !active || old.Buffer.String() != "已写🙂" {
			t.Fatal("ordinary receipt fixture needs accepted native preedit")
		}
	}
	if f.buffer.String() != markdownActionSource || f.changes != 0 || len(f.reports) != 0 {
		t.Fatal("ordinary preedit wrote or reported before the action")
	}
	return old
}

func (f *markdownActionFixture) capture(t *testing.T, value string, selection text.Range) {
	t.Helper()
	draft, ok, err := f.view.CaptureSessionDraft(maxConfirmedCellDraftBytes)
	if err != nil || !ok || draft.Title != "actions.md-表格单元格草稿.txt" || draft.Content != value ||
		draft.Anchor != selection.Anchor || draft.Head != selection.Head {
		t.Fatalf("recovery capture = %#v ok=%v err=%v, want %q %+v", draft, ok, err, value, selection)
	}
}

func markdownActionBufferReceipt(buffer *text.Buffer) func(*testing.T) {
	value, version, history, selection := buffer.String(), buffer.Version(), buffer.History(), buffer.Selection()
	return func(t *testing.T) {
		t.Helper()
		if buffer.String() != value || buffer.Version() != version || buffer.History() != history || buffer.Selection() != selection || buffer.InGroup() {
			t.Fatalf("buffer receipt changed: source=%q version=%d history=%+v selection=%+v", buffer.String(), buffer.Version(), buffer.History(), buffer.Selection())
		}
	}
}

func (f *markdownActionFixture) installSuccessor(t *testing.T) *editor.Editor {
	t.Helper()
	// This is the first safe callback seam: existing Markdown source/Editor
	// APIs only. In particular, do not call Tester.Frame/openCellByPointer here.
	f.view.DiscardDraft()
	f.view.refresh()
	preview := previewSnapshot{f.view.doc, f.view.rows, f.view.docBuffer, f.view.version}
	var target tableCellTarget
	found := false
	for _, row := range preview.rows {
		if row.tableRow != 1 {
			continue
		}
		candidate, ok := tableCellTargetOf(preview, row, preview.doc.Blocks[row.block], 1)
		if ok {
			target, found = captureTableCellTarget(preview, candidate)
			if found {
				break
			}
		}
	}
	if !found {
		t.Fatal("callback could not capture the current second cell")
	}
	context, err := f.view.currentTableCellContext(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.view.openTableCellDraft(context, target); err != nil {
		t.Fatal(err)
	}
	successor := f.view.draft
	if successor == nil || successor.Buffer.String() != "乙" {
		t.Fatal("callback did not open the real successor cell")
	}
	if err := successor.Buffer.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	if err := successor.Buffer.ReplaceUTF16(0, successor.Buffer.LenUTF16(), markdownActionSuccessorText); err != nil {
		t.Fatal(err)
	}
	if err := successor.Buffer.SetSelection(markdownActionReverse.Anchor, markdownActionReverse.Head); err != nil {
		t.Fatal(err)
	}
	if !successor.Buffer.EndGroup() {
		t.Fatal("successor edit did not create independent history")
	}
	f.view.confirmCellDraft(successor)
	f.capture(t, markdownActionSuccessorText, markdownActionReverse)
	return successor
}

func (f *markdownActionFixture) ownerReceipt(t *testing.T, successor *editor.Editor) func(*testing.T) {
	t.Helper()
	v := f.view
	if successor == nil || len(v.rows) == 0 {
		t.Fatal("no successor/preview to pin")
	}
	list, rows, cell := v.list, &v.rows[0], v.cellEdit
	version, epoch, released := v.version, v.tableActionEpoch, v.tableActionReleased
	raw := v.doc.Raw()
	return func(t *testing.T) {
		t.Helper()
		if v.draft != successor || v.recoveryDraft != successor || v.cellEdit != cell || cell == nil || cell.label != markdownActionSecondField ||
			!v.loaded || v.docBuffer != f.buffer || v.version != version || len(v.rows) == 0 || &v.rows[0] != rows || v.list != list || v.doc.Raw() != raw ||
			v.tableActionEpoch != epoch || v.tableActionReleased != released || v.draftError != nil ||
			successor.Buffer.String() != markdownActionSuccessorText || successor.Buffer.Selection() != markdownActionReverse {
			t.Fatalf("old action changed successor/preview: draft=%p successor=%p loaded=%v version=%d epoch=%d issue=%v", v.draft, successor, v.loaded, v.version, v.tableActionEpoch, v.draftError)
		}
		f.capture(t, markdownActionSuccessorText, markdownActionReverse)
	}
}

func (f *markdownActionFixture) focusSuccessor(t *testing.T) {
	t.Helper()
	f.tester.Frame()
	info, ok := f.tester.AccessByRole(ui.RoleTextField, markdownActionSecondField)
	if !ok || !info.Focused || info.Disabled || info.Offscreen {
		t.Fatalf("successor lost native focus after frame: %#v texts=%q", info, f.tester.Texts())
	}
}

func (f *markdownActionFixture) independentSuccessorHistory(t *testing.T, successor *editor.Editor) {
	t.Helper()
	sameMain := markdownActionBufferReceipt(f.buffer)
	before := successor.Buffer.History()
	if before.UndoDepth != 1 || before.RedoDepth != 0 {
		t.Fatalf("successor initial history = %+v", before)
	}
	if err := successor.Buffer.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	if err := successor.Buffer.ReplaceUTF16(4, 6, "改🙂"); err != nil {
		t.Fatal(err)
	}
	selection := text.Range{Anchor: 7, Head: 4}
	if err := successor.Buffer.SetSelection(selection.Anchor, selection.Head); err != nil {
		t.Fatal(err)
	}
	if !successor.Buffer.EndGroup() {
		t.Fatal("independent follow-up edit was not undoable")
	}
	f.view.confirmCellDraft(successor)
	f.capture(t, "新\r\n留改🙂", selection)
	if successor.Buffer.History().UndoDepth != 2 || !successor.Buffer.Undo() ||
		successor.Buffer.String() != markdownActionSuccessorText || successor.Buffer.Selection() != markdownActionReverse ||
		successor.Buffer.History().UndoDepth != 1 || successor.Buffer.History().RedoDepth != 1 {
		t.Fatal("successor undo lost original reverse selection/history")
	}
	f.view.confirmCellDraft(successor)
	f.capture(t, markdownActionSuccessorText, markdownActionReverse)
	if !successor.Buffer.Redo() || successor.Buffer.String() != "新\r\n留改🙂" || successor.Buffer.Selection() != selection ||
		successor.Buffer.History().UndoDepth != 2 || successor.Buffer.History().RedoDepth != 0 {
		t.Fatal("successor redo lost directed selection/history")
	}
	f.view.confirmCellDraft(successor)
	f.capture(t, "新\r\n留改🙂", selection)
	sameMain(t)
}

func TestMarkdownDraftActionCellChangedStops(t *testing.T) {
	for _, kind := range []string{"task", "link", "image", "reference"} {
		for _, route := range []string{"direct", "ui"} {
			t.Run(kind+"/"+route, func(t *testing.T) {
				f := newMarkdownActionFixture(t, false)
				command := captureMarkdownAction(t, f.view, kind)
				old := f.openCell(t, "已写🙂")
				version, beforeSelection := f.buffer.Version(), f.buffer.Selection()
				want := strings.Replace(markdownActionSource, "| 甲 |", "| 已写🙂 |", 1)
				end := text.UTF16Len(markdownActionSource[:strings.Index(markdownActionSource, "| 甲 |")+2]) + text.UTF16Len("已写🙂")
				committedSelection := text.Range{Anchor: end, Head: end}
				var successor *editor.Editor
				var sameMain, sameOwner, sameSuccessor func(*testing.T)
				f.onChanged = func() {
					if f.buffer.String() != want || f.buffer.Version() != version+1 || f.buffer.Selection() != committedSelection ||
						f.buffer.History().UndoDepth != 1 || f.buffer.History().RedoDepth != 0 || f.buffer.InGroup() || f.view.draft != nil {
						t.Fatal("Changed did not see the exact single completed old-cell write")
					}
					sameMain = markdownActionBufferReceipt(f.buffer)
					successor = f.installSuccessor(t)
					sameOwner = f.ownerReceipt(t, successor)
					sameSuccessor = markdownActionBufferReceipt(successor.Buffer)
				}
				var returned error
				if route == "direct" {
					returned = command.run(f.view)
				} else {
					command.click(t, f)
				}
				if successor == nil || successor == old || sameMain == nil || sameOwner == nil || sameSuccessor == nil {
					t.Fatal("Changed did not install/record an independent successor")
				}
				// Inspect immediately after the action, before requesting a frame.
				sameMain(t)
				sameOwner(t)
				sameSuccessor(t)
				if f.changes != 1 || len(f.opened) != 0 || len(f.reports) != 1 ||
					f.reports[0].Error() != "原单元格已写回："+markdownActionSuperseded {
					t.Fatalf("old action continued or misreported: changes=%d opened=%v reports=%v", f.changes, f.opened, f.reports)
				}
				if route == "direct" && returned != f.reports[0] {
					t.Fatalf("direct return/report identity changed: returned=%v report=%v", returned, f.reports[0])
				}
				f.focusSuccessor(t)
				sameOwner(t)
				sameMain(t)
				if !f.buffer.Undo() || f.buffer.String() != markdownActionSource || f.buffer.Selection() != beforeSelection ||
					f.buffer.History().UndoDepth != 0 || f.buffer.History().RedoDepth != 1 {
					t.Fatal("one main undo did not restore exact bytes/reverse selection")
				}
				sameSuccessor(t)
				if !f.buffer.Redo() || f.buffer.String() != want || f.buffer.Selection() != committedSelection ||
					f.buffer.History().UndoDepth != 1 || f.buffer.History().RedoDepth != 0 {
					t.Fatal("one main redo did not restore only the original cell write")
				}
				sameSuccessor(t)
				f.independentSuccessorHistory(t, successor)
				if f.changes != 1 || len(f.reports) != 1 || len(f.opened) != 0 {
					t.Fatal("independent history emitted an extra host event")
				}
			})
		}
	}
}

func TestMarkdownDraftActionOrdinaryChangedReceipt(t *testing.T) {
	for _, kind := range []string{"task", "link", "image", "reference"} {
		t.Run(kind, func(t *testing.T) {
			f := newMarkdownActionFixture(t, false)
			command := captureMarkdownAction(t, f.view, kind)
			old := f.openOrdinary(t, true)
			version := f.buffer.Version()
			want := strings.TrimSuffix(markdownActionSource, "末行") + "已写🙂"
			var successor *editor.Editor
			var sameMain, sameOwner func(*testing.T)
			f.onChanged = func() {
				if f.view.draft != old || f.buffer.String() != want || f.buffer.Version() != version+1 || f.buffer.History().UndoDepth != 1 {
					t.Fatal("ordinary receipt callback did not see a real old-block write")
				}
				sameMain = markdownActionBufferReceipt(f.buffer)
				successor = f.installSuccessor(t)
				sameOwner = f.ownerReceipt(t, successor)
			}
			returned := command.run(f.view)
			if sameMain == nil || sameOwner == nil || successor == old {
				t.Fatal("ordinary Changed did not install its successor")
			}
			sameMain(t)
			sameOwner(t)
			if returned == nil || returned.Error() != "原块已写回："+markdownActionSuperseded ||
				len(f.reports) != 1 || returned != f.reports[0] || f.changes != 1 || len(f.opened) != 0 {
				t.Fatalf("ordinary committed receipt/identity = %v reports=%v changes=%d opened=%v", returned, f.reports, f.changes, f.opened)
			}
			f.focusSuccessor(t)
			sameOwner(t)
			sameMain(t)
		})
	}
}

func TestMarkdownDraftActionSuccessfulControls(t *testing.T) {
	for _, kind := range []string{"task", "link", "image", "reference"} {
		for _, state := range []string{"no-draft", "clean-cell", "dirty-cell", "clean-ordinary", "dirty-ordinary"} {
			t.Run(kind+"/"+state, func(t *testing.T) {
				f := newMarkdownActionFixture(t, false)
				command := captureMarkdownAction(t, f.view, kind)
				want, changes := markdownActionSource, 0
				switch state {
				case "clean-cell":
					f.openCell(t, "甲")
				case "dirty-cell":
					f.openCell(t, "已写🙂")
					want, changes = strings.Replace(want, "| 甲 |", "| 已写🙂 |", 1), 1
				case "clean-ordinary", "dirty-ordinary":
					f.openOrdinary(t, state == "dirty-ordinary")
					if state == "dirty-ordinary" {
						want, changes = strings.TrimSuffix(want, "末行")+"已写🙂", 1
					}
				}
				if err := command.run(f.view); err != nil {
					t.Fatalf("valid action disabled by owner guard: %v", err)
				}
				if kind == "task" {
					want, changes = strings.Replace(want, "[ ] 待办", "[x] 待办", 1), changes+1
					if len(f.opened) != 0 {
						t.Fatal("task unexpectedly opened a reference")
					}
				} else {
					destination := map[string]string{"link": "target.md", "image": "picture.png", "reference": "target.txt"}[kind]
					if len(f.opened) != 1 || f.opened[0] != destination {
						t.Fatalf("ordinary successful destination = %v", f.opened)
					}
				}
				if f.buffer.String() != want || f.changes != changes || f.buffer.History().UndoDepth != changes ||
					f.buffer.History().RedoDepth != 0 || f.buffer.InGroup() || len(f.reports) != 0 {
					t.Fatalf("control bytes/history/events changed: source=%q changes=%d history=%+v reports=%v", f.buffer.String(), f.changes, f.buffer.History(), f.reports)
				}
			})
		}
	}
}

func TestMarkdownDraftActionNativeRefusalReporting(t *testing.T) {
	for _, kind := range []string{"task", "link", "image", "reference"} {
		for _, route := range []string{"direct", "ui"} {
			for _, refusal := range []string{"cell-pipe", "cell-readonly", "ordinary-size"} {
				if kind == "task" && route == "ui" && refusal == "cell-readonly" {
					// Native readonly checkboxes are disabled before this flush seam.
					// The existing live-readonly UI test covers that earlier route.
					continue
				}
				t.Run(kind+"/"+route+"/"+refusal, func(t *testing.T) {
					f := newMarkdownActionFixture(t, refusal == "ordinary-size")
					command := captureMarkdownAction(t, f.view, kind)
					var old *editor.Editor
					wantError := errTableCellPipe
					switch refusal {
					case "cell-pipe":
						old = f.openCell(t, "坏|值")
					case "cell-readonly":
						old = f.openCell(t, "已写🙂")
						f.view.host.ReadOnly = true
						wantError = errTableCellReadOnly
					case "ordinary-size":
						old = f.openOrdinary(t, true)
						wantError = text.ErrSizeLimit
					}
					sameMain := markdownActionBufferReceipt(f.buffer)
					var successor *editor.Editor
					var sameOwner func(*testing.T)
					f.onReport = func(err error) {
						if !errors.Is(err, wantError) || strings.Contains(err.Error(), "已写回") {
							t.Fatalf("refusal fabricated a commit or changed cause: %v", err)
						}
						f.view.host.ReadOnly = false
						successor = f.installSuccessor(t)
						sameOwner = f.ownerReceipt(t, successor)
					}
					var returned error
					if route == "direct" {
						returned = command.run(f.view)
					} else {
						command.click(t, f)
					}
					if successor == nil || successor == old || sameOwner == nil {
						t.Fatal("terminal Report did not replace the owner")
					}
					sameMain(t)
					sameOwner(t)
					if len(f.reports) != 1 || !errors.Is(f.reports[0], wantError) || len(f.opened) != 0 || f.changes != 0 {
						t.Fatalf("UI/nested flush did not report exactly once: reports=%v opened=%v changes=%d", f.reports, f.opened, f.changes)
					}
					if route == "direct" && returned != f.reports[0] {
						t.Fatal("direct nested failure did not return the identical Report object")
					}
					f.focusSuccessor(t)
					sameOwner(t)
					sameMain(t)
				})
			}
		}
	}
}

func TestMarkdownDraftActionReportingIsActionLocal(t *testing.T) {
	for _, kind := range []string{"task", "link", "image", "reference"} {
		t.Run(kind, func(t *testing.T) {
			f := newMarkdownActionFixture(t, false)
			command := captureMarkdownAction(t, f.view, kind)
			sameMain := markdownActionBufferReceipt(f.buffer)
			if kind == "reference" {
				// Pointer-up has its own earlier source refusal while grouped.
				// Native Enter reaches the action's transaction error route.
				focusReference(t, f.tester, referenceAccess(t, f.tester, true))
			}
			if err := f.buffer.BeginGroup(); err != nil {
				t.Fatal(err)
			}
			for attempt := 0; attempt < 2; attempt++ {
				f.tester.Advance(time.Second) // a distinct image gesture, not repeat suppression
				if kind == "reference" {
					f.tester.Key(0, ui.KeyEnter)
				} else {
					command.click(t, f)
				}
				if len(f.reports) != attempt+1 || !errors.Is(f.reports[attempt], text.ErrGroup) || !f.buffer.InGroup() {
					t.Fatalf("an unrelated action report was suppressed/repeated: attempt=%d reports=%v", attempt, f.reports)
				}
			}
			f.buffer.EndGroup()
			sameMain(t)
			if f.changes != 0 || len(f.opened) != 0 {
				t.Fatal("foreign-group refusal wrote or opened a reference")
			}
		})
	}
}

func TestMarkdownDraftActionTargetAndReadonlyControls(t *testing.T) {
	for _, kind := range []string{"task", "link", "image", "reference"} {
		for _, state := range []string{"stale", "readonly"} {
			t.Run(kind+"/"+state, func(t *testing.T) {
				f := newMarkdownActionFixture(t, false)
				command := captureMarkdownAction(t, f.view, kind)
				if state == "stale" {
					old := map[string]string{"task": "待办", "link": "target.md", "image": "picture.png", "reference": "src: target.txt"}[kind]
					replaceReferenceBufferText(t, f.buffer, old, "changed")
				} else {
					f.view.host.ReadOnly = true
				}
				sameMain := markdownActionBufferReceipt(f.buffer)
				err := command.run(f.view)
				if state == "readonly" && kind != "task" {
					if err != nil || len(f.opened) != 1 {
						t.Fatalf("readonly navigation refused: err=%v opened=%v", err, f.opened)
					}
				} else {
					want := map[string]error{"task": errTaskStale, "link": errLinkStale, "image": errLinkStale, "reference": errReferenceStale}[kind]
					if state == "readonly" {
						want = errTaskReadOnly
					}
					if !errors.Is(err, want) || len(f.opened) != 0 {
						t.Fatalf("target/readonly contract changed: err=%v want=%v opened=%v", err, want, f.opened)
					}
				}
				sameMain(t)
				if f.changes != 0 || len(f.reports) != 0 {
					t.Fatal("direct target-only failure changed its reporting contract")
				}
			})
		}
	}
}

func TestMarkdownDraftActionPublicFlushControls(t *testing.T) {
	for _, state := range []string{"no-draft", "clean-cell", "cell", "ordinary"} {
		t.Run(state, func(t *testing.T) {
			f := newMarkdownActionFixture(t, false)
			if state == "no-draft" || state == "clean-cell" {
				if state == "clean-cell" {
					f.openCell(t, "甲")
				}
				sameMain := markdownActionBufferReceipt(f.buffer)
				if err := f.view.Flush(); err != nil || f.view.draft != nil || f.changes != 0 || len(f.reports) != 0 {
					t.Fatalf("public clean Flush changed behavior: err=%v changes=%d reports=%v", err, f.changes, f.reports)
				}
				sameMain(t)
				return
			}
			want := strings.Replace(markdownActionSource, "| 甲 |", "| 已写🙂 |", 1)
			if state == "cell" {
				f.openCell(t, "已写🙂")
			} else {
				f.openOrdinary(t, true)
				want = strings.TrimSuffix(markdownActionSource, "末行") + "已写🙂"
			}
			version := f.buffer.Version()
			var sameMain, sameOwner func(*testing.T)
			f.onChanged = func() {
				if f.buffer.String() != want || f.buffer.Version() != version+1 || f.buffer.History().UndoDepth != 1 {
					t.Fatal("public Flush did not complete exactly one write")
				}
				sameMain = markdownActionBufferReceipt(f.buffer)
				sameOwner = f.ownerReceipt(t, f.installSuccessor(t))
			}
			err := f.view.Flush()
			if sameMain == nil || sameOwner == nil || f.changes != 1 {
				t.Fatal("public Flush did not reach Changed once")
			}
			if state == "cell" {
				if err != nil || len(f.reports) != 0 {
					t.Fatalf("public cell Flush must keep nil completion: err=%v reports=%v", err, f.reports)
				}
			} else if err == nil || err.Error() != markdownActionSuperseded || len(f.reports) != 1 || err != f.reports[0] {
				t.Fatalf("public ordinary Flush must keep bare refusal identity: err=%v reports=%v", err, f.reports)
			}
			sameMain(t)
			sameOwner(t)
			f.focusSuccessor(t)
			sameMain(t)
			sameOwner(t)
		})
	}
}

func TestMarkdownDraftActionOpenCallbackOwner(t *testing.T) {
	for _, kind := range []string{"link", "image", "reference"} {
		t.Run(kind, func(t *testing.T) {
			f := newMarkdownActionFixture(t, false)
			command := captureMarkdownAction(t, f.view, kind)
			sameMain := markdownActionBufferReceipt(f.buffer)
			var sameOwner func(*testing.T)
			f.onOpen = func() { sameOwner = f.ownerReceipt(t, f.installSuccessor(t)) }
			command.click(t, f)
			if sameOwner == nil || len(f.opened) != 1 || len(f.reports) != 0 || f.changes != 0 {
				t.Fatal("successful UI OpenReference did not remain terminal")
			}
			sameOwner(t)
			sameMain(t)
			f.focusSuccessor(t)
			sameOwner(t)
			sameMain(t)
		})
	}
}
