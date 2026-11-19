package graphview_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/modules/graphview"
	"levnote.local/native/ui/editor"
)

// This complete first oracle owns its literals and public drivers independently.
const buttonOwnerSource = "{\r\n" +
	`"format":"levnote.mindmap","version":1,"layout":"logicalStructure","root":{"id":"button-owner-root","topic":"按钮原主题🙂","children":[],"tags":[{"text":"既存🙂","future":1.20}],"futureNode":{"keep":"原样🙂"}},"futureTop":{"n":2.00}}` + "\r\n"

const buttonOwnerInput = "按钮草稿🙂\r\n原样"
const buttonOwnerTitle = "button-owner.mindmap-节点草稿.txt"
const buttonOwnerContent = "button-owner.mindmap\n未添加的标签输入\n节点 ID: button-owner-root\n字段: 新标签（尚未添加）\n\n按钮草稿🙂\r\n原样"
const buttonOwnerInputLabel = "新标签（单行，最多512字节）"

func buttonOwnerClick(t *testing.T, tester *ui.Tester, role ui.Role, name string) {
	t.Helper()
	info, found := tester.AccessByRole(role, name)
	if !found || info.Role != role || info.Name != name || info.Disabled || !info.Focusable || info.Offscreen ||
		info.Bounds.W <= 0 || info.Bounds.H <= 0 {
		t.Fatalf("GR65_CLICK_TARGET: role=%v name=%q found=%t access=%+v", role, name, found, info)
	}
	// Click validates the actual pointer target before sending one down/up pair.
	if err := tester.Click(name); err != nil {
		t.Fatalf("GR65_PUBLIC_CLICK: name=%q error=%v", name, err)
	}
	tester.Frame()
}

func buttonOwnerFocusedInput(t *testing.T, tester *ui.Tester) {
	t.Helper()
	input, found := tester.AccessByRole(ui.RoleTextField, buttonOwnerInputLabel)
	if !found || input.Role != ui.RoleTextField || input.Name != buttonOwnerInputLabel ||
		input.Disabled || !input.Focusable || !input.Focused || input.Offscreen {
		t.Fatalf("GR65_NATIVE_INPUT_FOCUS: found=%t access=%+v", found, input)
	}
	if active, noInputMethod := tester.TextInputActive(); !active || noInputMethod {
		t.Fatalf("GR65_NATIVE_INPUT_ENTRANCE: active=%t noInputMethod=%t", active, noInputMethod)
	}
	// The self-drawn editor does not bind AccValue. Exact bytes come only from
	// the independent Host envelope and public copy after the failed recovery.
}

func buttonOwnerSourceUnchanged(t *testing.T, buffer *text.Buffer, changed int) {
	t.Helper()
	if buffer.String() != buttonOwnerSource || buffer.Version() != 0 ||
		buffer.Selection() != (text.Range{Anchor: 23, Head: 7}) || buffer.InGroup() ||
		buffer.History() != (text.HistoryStatus{}) || changed != 0 {
		t.Fatalf("GR65_SOURCE_INVARIANTS: text=%q version=%d selection=%+v group=%t history=%+v changed=%d",
			buffer.String(), buffer.Version(), buffer.Selection(), buffer.InGroup(), buffer.History(), changed)
	}
}

func TestGraphRecoveryButtonReportsOriginalHostFailureOnce(t *testing.T) {
	buffer := text.New(buttonOwnerSource)
	if err := buffer.SetSelection(23, 7); err != nil {
		t.Fatalf("GR65_FIXTURE_SELECTION: %v", err)
	}
	original := errors.New("button-original-host-failure🙂")
	changed := 0
	var reports []error
	var envelopes []struct{ title, content string }
	view := graphview.MindmapAdapter().Create(editor.Host{
		Buffer:  buffer,
		Path:    "button-owner.mindmap",
		Changed: func() { changed++ },
		Report:  func(err error) { reports = append(reports, err) },
		RecoverDraft: func(title, content string) error {
			envelopes = append(envelopes, struct{ title, content string }{title, content})
			return original
		},
	})
	t.Cleanup(view.Release)
	composer, ok := view.(editor.Composer)
	if !ok {
		t.Fatal("GR65_PUBLIC_COMPOSITION_CAPABILITY: absent")
	}
	tester := ui.NewTester(view.Draw, 960, 900)
	initialWarnings := []string{
		`node "button-owner-root" has 1 preserved unknown field(s)`,
		"preserved 1 unknown top-level field(s)",
	}
	if got := view.Warnings(); !reflect.DeepEqual(got, initialWarnings) {
		t.Fatalf("GR65_INITIAL_COMPATIBILITY_WARNINGS: got=%#v want=%#v", got, initialWarnings)
	}
	buttonOwnerSourceUnchanged(t, buffer, changed)
	if len(reports) != 0 || len(envelopes) != 0 || composer.CompositionActive() {
		t.Fatalf("GR65_INITIAL_STATE: reports=%#v envelopes=%#v composition=%t", reports, envelopes, composer.CompositionActive())
	}
	t.Log("GR65_PRECONDITIONS_PASSED")

	buttonOwnerClick(t, tester, ui.RoleButton, "展开标签面板")
	buttonOwnerClick(t, tester, ui.RoleTextField, buttonOwnerInputLabel)
	buttonOwnerFocusedInput(t, tester)
	tester.Type(buttonOwnerInput)
	tester.Frame()
	buttonOwnerFocusedInput(t, tester)
	if composer.CompositionActive() || len(reports) != 0 || len(envelopes) != 0 {
		t.Fatalf("GR65_CONFIRMED_NATIVE_TYPE: composition=%t reports=%#v envelopes=%#v",
			composer.CompositionActive(), reports, envelopes)
	}
	buttonOwnerSourceUnchanged(t, buffer, changed)
	t.Log("GR65_NATIVE_INPUT_CONFIRMED")

	// This is the sole recovery click; no direct RecoverDraft call is used.
	buttonOwnerClick(t, tester, ui.RoleButton, "恢复标签及节点草稿")
	if len(envelopes) != 1 || envelopes[0].title != buttonOwnerTitle || envelopes[0].content != buttonOwnerContent {
		t.Fatalf("GR65_HOST_ENVELOPE: got=%#v wantTitle=%q wantContent=%q", envelopes, buttonOwnerTitle, buttonOwnerContent)
	}
	if composer.CompositionActive() {
		t.Fatal("GR65_RECOVERY_COMPOSITION: confirmed input became a composition")
	}
	t.Log("GR65_HOST_ENVELOPE_PASSED")

	buttonOwnerClick(t, tester, ui.RoleTextField, buttonOwnerInputLabel)
	buttonOwnerFocusedInput(t, tester)
	tester.Command("selectAll")
	tester.Command("copy")
	tester.Frame()
	if got := tester.Clipboard(); got != buttonOwnerInput {
		t.Fatalf("GR65_FAILED_INPUT_PRESERVATION: clipboard=%q want=%q", got, buttonOwnerInput)
	}
	buttonOwnerFocusedInput(t, tester)
	buttonOwnerSourceUnchanged(t, buffer, changed)
	if len(envelopes) != 1 || envelopes[0].title != buttonOwnerTitle || envelopes[0].content != buttonOwnerContent || composer.CompositionActive() {
		t.Fatalf("GR65_POST_COPY_HOST_STATE: envelopes=%#v composition=%t", envelopes, composer.CompositionActive())
	}
	t.Log("GR65_FAILED_INPUT_SOURCE_PRESERVED")

	// All diagnostic observations precede cleanup, which may attempt a Flush.
	originalReport1 := []error{original}
	warnings := view.Warnings()
	wantWarnings := []string{
		`node "button-owner-root" has 1 preserved unknown field(s)`,
		"preserved 1 unknown top-level field(s)",
		"节点草稿尚未写回：button-original-host-failure🙂",
	}
	t.Logf("GR65_DIAGNOSTICS_OBSERVED: reports=%d warnings=%#v original=%p", len(reports), warnings, original)
	for index, report := range reports {
		t.Logf("GR65_REPORT_OBJECT: index=%d object=%p originalObject=%t error=%v", index, report, report == original, report)
	}
	if len(reports) != len(originalReport1) || reports[0] != originalReport1[0] {
		t.Errorf("GR65_ONE_REPORT_ORIGINAL_OBJECT: got=%#v want=[]error{%p}", reports, original)
	}
	if !reflect.DeepEqual(warnings, wantWarnings) {
		t.Errorf("GR65_EXACT_WARNINGS: got=%#v want=%#v", warnings, wantWarnings)
	}
}
