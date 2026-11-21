package graphview

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/mindmap"
	"levnote.local/native/ui/editor"
)

const warningInputMindmap = `{
  "format": "levnote.mindmap",
  "version": 1,
  "layout": "logicalStructure",
  "futureTop": {"kept": 1},
  "root": {
    "id": "root",
    "topic": "中心",
    "note": "原备注",
    "futureNode": {"kept": 2},
    "children": []
  }
}`

type warningInputState struct {
	value      string
	version    uint64
	selection  text.Range
	history    text.HistoryStatus
	group      bool
	mark       ui.TextInputRange
	marked     bool
	active     bool
	inputError error
	stamp      text.ChangeStamp
	delivery   editor.InputDelivery
	delivered  bool
}

func warningInputSnapshot(input *editor.Editor) warningInputState {
	mark, marked := input.MarkedRange()
	delivery, delivered := input.LastInputDelivery()
	return warningInputState{
		value: input.Buffer.String(), version: input.Buffer.Version(),
		selection: input.Buffer.Selection(), history: input.Buffer.History(),
		group: input.Buffer.InGroup(), mark: mark, marked: marked,
		active: input.CompositionActive(), inputError: input.InputActionError(),
		stamp: input.Buffer.ChangeStamp(), delivery: delivery, delivered: delivered,
	}
}

func warningInputClick(t *testing.T, tester *ui.Tester, label string) {
	t.Helper()
	if err := tester.Click(label); err != nil {
		t.Fatalf("click %q: %v", label, err)
	}
}

// Capture compares the actual installed native adapter, cancellation lease and
// transport session; this read performs no text event or frame settlement.
func warningNativeCapture(t *testing.T, tester *ui.Tester) ui.TesterTextInputCapture {
	t.Helper()
	var capture ui.TesterTextInputCapture
	if err := tester.TextInputNoFrame(func(n *ui.TesterTextInput) { capture = n.Capture() }); err != nil {
		t.Fatal(err)
	}
	return capture
}

func warningDetailsKeyboard(t *testing.T, tester *ui.Tester) {
	t.Helper()
	for attempt := 0; attempt < 120 && !tester.Focused("查看图形提示详情"); attempt++ {
		tester.Key(0, ui.KeyTab)
		tester.Frame()
	}
	if !tester.Focused("查看图形提示详情") {
		t.Fatal("Details could not receive keyboard focus after explicit completion")
	}
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
}

func warningInputNoteSource(t *testing.T, source, value string) string {
	t.Helper()
	document, _, err := mindmap.Parse([]byte(source), mindmap.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := document.SetNote("root", value); err != nil {
		t.Fatal(err)
	}
	serialized, err := document.Serialize(mindmap.SerializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return string(serialized)
}

func TestGraphWarningDetailsRetainsMarklessRefusalUntilEscape(t *testing.T) {
	buffer := text.New(warningInputMindmap)
	changed := 0
	v := newView(kindMindmap, editor.Host{Buffer: buffer, Changed: func() { changed++ }})
	t.Cleanup(v.Release)
	tester := ui.NewTester(v.Draw, 960, 720)
	tester.Frame()
	warningInputClick(t, tester, "打开或收起字段查找替换")
	input := v.search.query
	input.Buffer = text.NewWithLimit("a", 1)
	if err := input.Buffer.ReplaceUTF16(0, 1, "b"); err != nil || !input.Buffer.Undo() {
		t.Fatal("could not prepare retained-input redo branch", err)
	}
	warningInputClick(t, tester, "字段查找文字")
	tester.Command("selectAll")
	tester.Compose("🙂", 1)
	tester.Frame()
	before := warningInputSnapshot(input)
	if before.inputError == nil || before.marked || before.active || before.group || !input.Buffer.CanRedo() {
		t.Fatalf("fixture did not establish actual markless refused native input: %+v", before)
	}
	native := warningNativeCapture(t, tester)
	owner, stamp, selection := v.fieldSearchOwner(), buffer.ChangeStamp(), buffer.Selection()
	warningInputClick(t, tester, "查看图形提示详情")
	tester.Frame()
	if !reflect.DeepEqual(warningInputSnapshot(input), before) || warningNativeCapture(t, tester) != native ||
		!tester.Focused("字段查找文字") || v.fieldSearchOwner() != owner || tester.HasText("图形视图提示详情") ||
		buffer.ChangeStamp() != stamp || buffer.Selection() != selection || buffer.String() != warningInputMindmap || changed != 0 {
		t.Fatal("Details changed retained markless input, native client, owner or host source")
	}
	if !tester.HasText("请先完成或取消当前输入，再查看详情") {
		t.Fatal("markless refusal has no visible completion guidance")
	}
	tester.Key(0, ui.KeyEscape)
	tester.Frame()
	if input.InputActionError() != nil || !input.Buffer.CanRedo() {
		t.Fatal("explicit Escape did not clear retained input while preserving redo")
	}
	warnings := v.Warnings()
	warningDetailsKeyboard(t, tester)
	if !tester.HasText("图形视图提示详情") {
		t.Fatal("Details did not open after retained input was cancelled")
	}
	for _, warning := range warnings {
		if !tester.HasText(warning) {
			t.Fatalf("Details omitted warning after Escape: %q", warning)
		}
	}
	if buffer.String() != warningInputMindmap || buffer.ChangeStamp() != stamp || buffer.Selection() != selection ||
		buffer.History() != (text.HistoryStatus{}) || changed != 0 {
		t.Fatal("Details after Escape changed host source or history")
	}
}

func TestGraphWarningDetailsKeepsHealthyMarkedInputUntilCompletion(t *testing.T) {
	for _, kind := range []string{"note", "field-search"} {
		t.Run(kind, func(t *testing.T) {
			buffer := text.New(warningInputMindmap)
			changed, reports := 0, 0
			v := newView(kindMindmap, editor.Host{
				Buffer: buffer, Changed: func() { changed++ }, Report: func(error) { reports++ },
			})
			t.Cleanup(v.Release)
			tester := ui.NewTester(v.Draw, 960, 720)
			tester.Frame()
			if warnings := v.Warnings(); len(warnings) < 2 {
				t.Fatalf("compatibility fixture has no full warning set: %#v", warnings)
			}

			var input *editor.Editor
			focusLabel := ""
			switch kind {
			case "note":
				warningInputClick(t, tester, "字段：纯文本备注")
				warningInputClick(t, tester, "节点备注（纯文本）")
				input, focusLabel = v.draft.input, "节点备注（纯文本）"
			case "field-search":
				warningInputClick(t, tester, "打开或收起字段查找替换")
				input, focusLabel = v.search.query, "字段查找文字"
			default:
				t.Fatalf("unknown input fixture %q", kind)
			}
			if input == nil || !tester.Focused(focusLabel) {
				t.Fatalf("actual graph input was not focused: %q", focusLabel)
			}
			tester.Command("selectAll")
			preedit := "尚未完成的中文🙂"
			tester.Compose(preedit, len([]rune(preedit)))
			tester.Frame()

			before := warningInputSnapshot(input)
			if !before.marked || !before.active || before.inputError == nil || !tester.Focused(focusLabel) {
				t.Fatalf("fixture did not establish healthy native preedit: state=%+v focused=%t", before, tester.Focused(focusLabel))
			}
			fieldOwner, searchOwner := v.fieldIdentity(), v.fieldSearchOwner()
			hostStamp, hostSelection := buffer.ChangeStamp(), buffer.Selection()
			nativeBefore := warningNativeCapture(t, tester)
			initialWarnings := v.Warnings()
			if len(initialWarnings) < 2 {
				t.Fatalf("compatibility warnings disappeared before Details: %#v", initialWarnings)
			}
			warningGroup, ok := tester.Access("图形视图提示")
			if !ok || warningGroup.Role != ui.RoleGroup || warningGroup.Offscreen || warningGroup.Bounds.H <= 0 || warningGroup.Bounds.H > 28.1 {
				t.Fatalf("warning row must retain its labeled compact group: %#v", warningGroup)
			}
			for _, warning := range initialWarnings {
				if !strings.Contains(warningGroup.Description, warning) {
					t.Fatalf("warning must remain accessible in full: %q group=%#v", warning, warningGroup)
				}
			}
			button, ok := tester.AccessByRole(ui.RoleButton, "查看图形提示详情")
			if !ok || button.Disabled || !button.Focusable || button.Offscreen {
				t.Fatalf("Details lost access while input is pending: %#v", button)
			}
			for _, warning := range initialWarnings {
				if !strings.Contains(button.Description, warning) {
					t.Fatalf("accessible warning details omitted %q: %q", warning, button.Description)
				}
			}

			// This actual pointer gesture is the regression trigger: it must not
			// release the active native text-input client before admission.
			warningInputClick(t, tester, "查看图形提示详情")
			tester.Frame()
			after := warningInputSnapshot(input)
			if !reflect.DeepEqual(after, before) || !tester.Focused(focusLabel) {
				t.Fatalf("refused Details settled or moved native input: before=%+v after=%+v focused=%t", before, after, tester.Focused(focusLabel))
			}
			if warningNativeCapture(t, tester) != nativeBefore || v.fieldIdentity() != fieldOwner || v.fieldSearchOwner() != searchOwner ||
				buffer.ChangeStamp() != hostStamp || buffer.Selection() != hostSelection || v.draftError != nil || v.search.err != nil {
				t.Fatal("refused Details replaced the installed native client/session, draft owner or host stamp")
			}
			if active, noIME := tester.TextInputActive(); !active || noIME {
				t.Fatalf("refused Details retired native text input: active=%t noIME=%t", active, noIME)
			}
			if tester.HasText("图形视图提示详情") {
				t.Fatal("modal opened while marked input was still active")
			}
			if !tester.HasText("请先完成或取消当前输入，再查看详情") {
				t.Fatalf("input refusal was not visible: %#v", tester.Texts())
			}
			if !reflect.DeepEqual(v.Warnings(), initialWarnings) || buffer.String() != warningInputMindmap ||
				buffer.Version() != 0 || buffer.History() != (text.HistoryStatus{}) || buffer.InGroup() || changed != 0 || reports != 0 {
				t.Fatal("refused Details changed compatibility warnings or host source")
			}

			// Exercise both explicit user routes. A legal note final commits once;
			// search cancellation remains independent of the document host.
			wantSource := warningInputMindmap
			finalVersion, finalHistory, finalChanged := buffer.Version(), buffer.History(), changed
			if kind == "note" {
				wantSource = warningInputNoteSource(t, buffer.String(), preedit)
				tester.Type(preedit)
			} else {
				tester.Key(0, ui.KeyEscape)
			}
			tester.Frame()
			if input.CompositionActive() || input.Buffer.InGroup() || tester.Focused(focusLabel) == false {
				t.Fatalf("explicit completion did not settle only the input: composing=%t group=%t focused=%t", input.CompositionActive(), input.Buffer.InGroup(), tester.Focused(focusLabel))
			}
			if kind == "note" {
				note, present, err := parseMind(t, buffer).Field("root", mindmap.FieldNote)
				committedStamp, committedHistory := buffer.ChangeStamp(), buffer.History()
				end := text.UTF16Len(wantSource)
				if err != nil || !present || note != preedit || input.Buffer.String() != preedit ||
					v.draft == nil || v.draft.input != input || v.pendingDraft() || v.draftError != nil ||
					buffer.String() != wantSource || buffer.Version() != finalVersion+1 ||
					committedStamp.BaselineID != hostStamp.BaselineID || committedStamp.ConfirmedChangeID == hostStamp.ConfirmedChangeID ||
					committedHistory.UndoDepth != finalHistory.UndoDepth+1 || committedHistory.RedoDepth != 0 ||
					committedHistory.Truncated != finalHistory.Truncated || buffer.InGroup() ||
					buffer.Selection() != (text.Range{Anchor: end, Head: end}) || changed != finalChanged+1 || reports != 0 {
					t.Fatalf("legal native final did not commit the exact root note once: note=%q present=%t changed=%d err=%v", note, present, changed, err)
				}
			} else if buffer.String() != warningInputMindmap || buffer.Version() != finalVersion ||
				buffer.ChangeStamp() != hostStamp || buffer.Selection() != hostSelection || buffer.History() != finalHistory ||
				buffer.InGroup() || changed != finalChanged || reports != 0 {
				t.Fatal("field-search cancellation changed the document host")
			}
			settled := warningInputSnapshot(input)
			if settled.inputError != nil || settled.marked {
				t.Fatalf("user completion left input inadmissible: %+v", settled)
			}
			hostVersion, hostHistory := buffer.Version(), buffer.History()
			hostStamp, hostSelection = buffer.ChangeStamp(), buffer.Selection()
			settledChanged, settledReports := changed, reports
			fieldOwner, searchOwner = v.fieldIdentity(), v.fieldSearchOwner()
			warningDetailsKeyboard(t, tester)
			if !tester.HasText("图形视图提示详情") {
				t.Fatal("Details stayed unavailable after user completed input")
			}
			for _, warning := range initialWarnings {
				if !tester.HasText(warning) {
					t.Fatalf("Details modal omitted full warning %q", warning)
				}
			}
			if input.Buffer.String() != settled.value || input.Buffer.ChangeStamp() != settled.stamp ||
				input.Buffer.Selection() != settled.selection || input.Buffer.History() != settled.history {
				t.Fatal("keyboard Details mutated the completed local draft")
			}
			if buffer.String() != wantSource || buffer.Version() != hostVersion || buffer.ChangeStamp() != hostStamp ||
				buffer.Selection() != hostSelection || buffer.History() != hostHistory || buffer.InGroup() ||
				changed != settledChanged || reports != settledReports || v.fieldIdentity() != fieldOwner || v.fieldSearchOwner() != searchOwner {
				t.Fatal("opening Details added a host write, callback, report or owner change after explicit completion")
			}
			if _, ok := tester.AccessByRole(ui.RoleButton, "关闭详情"); !ok {
				t.Fatal("modal close action is not accessible")
			}
			warningInputClick(t, tester, "关闭详情")
			tester.Frame()
			if tester.HasText("图形视图提示详情") {
				t.Fatal("accessible modal close did not close Details")
			}
		})
	}
}

func TestGraphWarningDetailsRetainsSourceGroupRefusedNote(t *testing.T) {
	buffer := text.New(warningInputMindmap)
	changed, reports := 0, 0
	v := newView(kindMindmap, editor.Host{
		Buffer: buffer, Changed: func() { changed++ }, Report: func(error) { reports++ },
	})
	t.Cleanup(v.Release)
	tester := ui.NewTester(v.Draw, 960, 720)
	tester.Frame()
	warningInputClick(t, tester, "字段：纯文本备注")
	warningInputClick(t, tester, "节点备注（纯文本）")
	input := v.draft.input
	version, stamp, selection, history := buffer.Version(), buffer.ChangeStamp(), buffer.Selection(), buffer.History()
	if err := buffer.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if buffer.InGroup() {
			buffer.CancelGroup()
		}
	})
	tester.Command("selectAll")
	const note = "真实拒写后保留的备注🙂"
	tester.Type(note)
	tester.Frame()
	settled := warningInputSnapshot(input)
	if settled.value != note || settled.marked || settled.active || settled.group || settled.inputError != nil ||
		!v.pendingDraft() || !errors.Is(v.draftError, text.ErrGroup) || changed != 0 || reports != 1 {
		t.Fatalf("native final did not retain a real source-group-refused note: state=%+v pending=%t error=%v changed=%d reports=%d", settled, v.pendingDraft(), v.draftError, changed, reports)
	}
	if !buffer.InGroup() || buffer.String() != warningInputMindmap || buffer.Version() != version || buffer.ChangeStamp() != stamp ||
		buffer.Selection() != selection || buffer.History() != history {
		t.Fatal("rejected final changed the source transaction")
	}
	if buffer.EndGroup() {
		t.Fatal("fixture source group unexpectedly contained an edit")
	}
	if buffer.InGroup() || buffer.String() != warningInputMindmap || buffer.Version() != version || buffer.ChangeStamp() != stamp ||
		buffer.Selection() != selection || buffer.History() != history || changed != 0 || reports != 1 {
		t.Fatal("ending the empty source group changed the document host")
	}
	// With the genuine rejection retained and the source group restored, Details
	// must inspect the warning without retrying the ordinary staged note.
	settled = warningInputSnapshot(input)
	owner, draftError := v.fieldIdentity(), v.draftError
	warnings := append([]string(nil), v.Warnings()...)
	warningDetailsKeyboard(t, tester)
	if !tester.HasText("图形视图提示详情") {
		t.Fatal("Details did not accept an ordinary noncomposing retained note")
	}
	for _, warning := range warnings {
		if !tester.HasText(warning) {
			t.Fatalf("Details omitted the actual retained-draft warning: %q", warning)
		}
	}
	if !reflect.DeepEqual(warningInputSnapshot(input), settled) || v.fieldIdentity() != owner ||
		!v.pendingDraft() || v.draftError != draftError || !errors.Is(v.draftError, text.ErrGroup) || !reflect.DeepEqual(v.Warnings(), warnings) ||
		buffer.String() != warningInputMindmap || buffer.Version() != version || buffer.ChangeStamp() != stamp ||
		buffer.Selection() != selection || buffer.History() != history || buffer.InGroup() || changed != 0 || reports != 1 {
		t.Fatal("Details retried, reported, discarded or mutated the genuinely refused note")
	}
	warningInputClick(t, tester, "关闭详情")
	tester.Frame()
	if tester.HasText("图形视图提示详情") || !v.pendingDraft() || v.draftError != draftError ||
		buffer.String() != warningInputMindmap || buffer.ChangeStamp() != stamp || buffer.History() != history || changed != 0 || reports != 1 {
		t.Fatal("closing Details changed the retained refusal")
	}
	wantSource := warningInputNoteSource(t, warningInputMindmap, note)
	warningInputClick(t, tester, "重试写回")
	tester.Frame()
	if buffer.String() != wantSource || buffer.Version() != version+1 || buffer.History().UndoDepth != history.UndoDepth+1 ||
		buffer.History().RedoDepth != 0 || buffer.InGroup() || v.pendingDraft() || v.draftError != nil || changed != 1 || reports != 1 {
		t.Fatal("explicit retry did not recover the retained note exactly once")
	}
}
