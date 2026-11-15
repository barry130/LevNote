package graphview_test

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/modules/graphview"
	"levnote.local/native/ui/editor"
)

// These controls reuse only the first oracle's public drivers and independent
// literals. They never call RecoverDraft directly or inspect private view state.
func buttonControlEmptyInput(t *testing.T, tester *ui.Tester, label string) {
	t.Helper()
	if info, found := tester.AccessByRole(ui.RoleButton, "恢复标签及节点草稿"); found {
		t.Fatalf("%s_RECOVERY_BUTTON_REMAINED: access=%+v", label, info)
	}
	buttonOwnerClick(t, tester, ui.RoleTextField, buttonOwnerInputLabel)
	buttonOwnerFocusedInput(t, tester)
	const sentinel = "GR65_empty_clipboard_probe🙂"
	// Empty selection copies nothing. A distinct public clipboard sentinel
	// distinguishes an empty, focused input from retained draft bytes.
	tester.SetClipboard(sentinel)
	tester.Command("selectAll")
	tester.Command("copy")
	tester.Frame()
	if got := tester.Clipboard(); got != sentinel {
		t.Fatalf("%s_RECOVERED_INPUT_NOT_EMPTY: clipboard=%q wantUnchanged=%q", label, got, sentinel)
	}
	buttonOwnerFocusedInput(t, tester)
	if info, found := tester.AccessByRole(ui.RoleButton, "恢复标签及节点草稿"); found {
		t.Fatalf("%s_RECOVERY_BUTTON_REAPPEARED: access=%+v", label, info)
	}
}

func TestGraphRecoveryButtonSuccessClearsPriorTagError(t *testing.T) {
	buffer := text.New(buttonOwnerSource)
	if err := buffer.SetSelection(23, 7); err != nil {
		t.Fatalf("GR65_CLEAR_FIXTURE_SELECTION: %v", err)
	}
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
			return nil
		},
	})
	t.Cleanup(view.Release)
	composer, ok := view.(editor.Composer)
	if !ok {
		t.Fatal("GR65_CLEAR_PUBLIC_COMPOSITION_CAPABILITY: absent")
	}
	tester := ui.NewTester(view.Draw, 960, 900)
	compatibility := []string{
		`node "button-owner-root" has 1 preserved unknown field(s)`,
		"preserved 1 unknown top-level field(s)",
	}
	if got := view.Warnings(); !reflect.DeepEqual(got, compatibility) {
		t.Fatalf("GR65_CLEAR_INITIAL_WARNINGS: got=%#v want=%#v", got, compatibility)
	}
	buttonOwnerSourceUnchanged(t, buffer, changed)
	if len(reports) != 0 || len(envelopes) != 0 || composer.CompositionActive() {
		t.Fatalf("GR65_CLEAR_INITIAL_STATE: reports=%#v envelopes=%#v composition=%t", reports, envelopes, composer.CompositionActive())
	}
	t.Log("GR65_CLEAR_PRECONDITIONS_PASSED")

	buttonOwnerClick(t, tester, ui.RoleButton, "展开标签面板")
	buttonOwnerClick(t, tester, ui.RoleTextField, buttonOwnerInputLabel)
	buttonOwnerFocusedInput(t, tester)
	tester.Type(buttonOwnerInput)
	tester.Frame()
	buttonOwnerFocusedInput(t, tester)
	if composer.CompositionActive() || len(reports) != 0 || len(envelopes) != 0 {
		t.Fatalf("GR65_CLEAR_NATIVE_TYPE: composition=%t reports=%#v envelopes=%#v", composer.CompositionActive(), reports, envelopes)
	}
	buttonOwnerSourceUnchanged(t, buffer, changed)
	t.Log("GR65_CLEAR_NATIVE_INPUT_CONFIRMED")

	// CRLF is valid confirmed input to preserve but not a valid single-line tag.
	buttonOwnerClick(t, tester, ui.RoleButton, "添加标签")
	const validation = "新标签须为非空单行UTF-8文字，最多512字节；输入仍保留"
	priorWarnings := []string{
		`node "button-owner-root" has 1 preserved unknown field(s)`,
		"preserved 1 unknown top-level field(s)",
		"标签操作：新标签须为非空单行UTF-8文字，最多512字节；输入仍保留",
	}
	if len(reports) != 1 || reports[0] == nil || reports[0].Error() != validation || len(envelopes) != 0 ||
		!reflect.DeepEqual(view.Warnings(), priorWarnings) {
		t.Fatalf("GR65_CLEAR_PRIOR_VALIDATION: reports=%#v envelopes=%#v warnings=%#v want=%#v", reports, envelopes, view.Warnings(), priorWarnings)
	}
	prior := reports[0]
	buttonOwnerClick(t, tester, ui.RoleTextField, buttonOwnerInputLabel)
	buttonOwnerFocusedInput(t, tester)
	tester.Command("selectAll")
	tester.Command("copy")
	tester.Frame()
	if got := tester.Clipboard(); got != buttonOwnerInput {
		t.Fatalf("GR65_CLEAR_PRIOR_INPUT_PRESERVATION: clipboard=%q want=%q", got, buttonOwnerInput)
	}
	buttonOwnerSourceUnchanged(t, buffer, changed)
	t.Log("GR65_CLEAR_PRIOR_TAG_ERROR_CONFIRMED")

	buttonOwnerClick(t, tester, ui.RoleButton, "恢复标签及节点草稿")
	if len(envelopes) != 1 || envelopes[0].title != buttonOwnerTitle || envelopes[0].content != buttonOwnerContent {
		t.Fatalf("GR65_CLEAR_SUCCESS_ENVELOPE: got=%#v wantTitle=%q wantContent=%q", envelopes, buttonOwnerTitle, buttonOwnerContent)
	}
	if len(reports) != 1 || reports[0] != prior || composer.CompositionActive() {
		t.Fatalf("GR65_CLEAR_SUCCESS_REPORTS: reports=%#v wantOriginal=%p composition=%t", reports, prior, composer.CompositionActive())
	}
	t.Log("GR65_CLEAR_SUCCESS_ENVELOPE_PASSED")

	buttonControlEmptyInput(t, tester, "GR65_CLEAR")
	buttonOwnerSourceUnchanged(t, buffer, changed)
	if len(envelopes) != 1 || envelopes[0].title != buttonOwnerTitle || envelopes[0].content != buttonOwnerContent ||
		len(reports) != 1 || reports[0] != prior || composer.CompositionActive() {
		t.Fatalf("GR65_CLEAR_FINAL_PUBLIC_STATE: reports=%#v envelopes=%#v composition=%t", reports, envelopes, composer.CompositionActive())
	}
	t.Log("GR65_CLEAR_INPUT_AND_SOURCE_PRESERVED")

	// The validation Report remains an event; successful recovery clears its
	// active warning and generates no second Report, all observed before cleanup.
	warnings := view.Warnings()
	t.Logf("GR65_CLEAR_DIAGNOSTICS_OBSERVED: reports=%d warnings=%#v original=%p", len(reports), warnings, prior)
	for index, report := range reports {
		t.Logf("GR65_CLEAR_REPORT_OBJECT: index=%d object=%p originalObject=%t error=%v", index, report, report == prior, report)
	}
	if !reflect.DeepEqual(warnings, compatibility) {
		t.Errorf("GR65_CLEAR_EXACT_WARNINGS: got=%#v want=%#v", warnings, compatibility)
	}
}

type buttonControlHostError struct{ message string }

func (e *buttonControlHostError) Error() string { return e.message }

func TestGraphRecoveryButtonRetryReportsOriginalFailureOnce(t *testing.T) {
	buffer := text.New(buttonOwnerSource)
	if err := buffer.SetSelection(23, 7); err != nil {
		t.Fatalf("GR65_RETRY_FIXTURE_SELECTION: %v", err)
	}
	original := &buttonControlHostError{message: "button-original-host-failure🙂"}
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
			if len(envelopes) == 1 {
				return original
			}
			return nil
		},
	})
	t.Cleanup(view.Release)
	composer, ok := view.(editor.Composer)
	if !ok {
		t.Fatal("GR65_RETRY_PUBLIC_COMPOSITION_CAPABILITY: absent")
	}
	tester := ui.NewTester(view.Draw, 960, 900)
	compatibility := []string{
		`node "button-owner-root" has 1 preserved unknown field(s)`,
		"preserved 1 unknown top-level field(s)",
	}
	if got := view.Warnings(); !reflect.DeepEqual(got, compatibility) {
		t.Fatalf("GR65_RETRY_INITIAL_WARNINGS: got=%#v want=%#v", got, compatibility)
	}
	buttonOwnerSourceUnchanged(t, buffer, changed)
	if len(reports) != 0 || len(envelopes) != 0 || composer.CompositionActive() {
		t.Fatalf("GR65_RETRY_INITIAL_STATE: reports=%#v envelopes=%#v composition=%t", reports, envelopes, composer.CompositionActive())
	}
	t.Log("GR65_RETRY_PRECONDITIONS_PASSED")

	buttonOwnerClick(t, tester, ui.RoleButton, "展开标签面板")
	buttonOwnerClick(t, tester, ui.RoleTextField, buttonOwnerInputLabel)
	buttonOwnerFocusedInput(t, tester)
	tester.Type(buttonOwnerInput)
	tester.Frame()
	buttonOwnerFocusedInput(t, tester)
	if composer.CompositionActive() || len(reports) != 0 || len(envelopes) != 0 {
		t.Fatalf("GR65_RETRY_NATIVE_TYPE: composition=%t reports=%#v envelopes=%#v", composer.CompositionActive(), reports, envelopes)
	}
	buttonOwnerSourceUnchanged(t, buffer, changed)
	t.Log("GR65_RETRY_NATIVE_INPUT_CONFIRMED")

	buttonOwnerClick(t, tester, ui.RoleButton, "恢复标签及节点草稿")
	failureWarnings := []string{
		`node "button-owner-root" has 1 preserved unknown field(s)`,
		"preserved 1 unknown top-level field(s)",
		"节点草稿尚未写回：button-original-host-failure🙂",
	}
	if len(envelopes) != 1 || envelopes[0].title != buttonOwnerTitle || envelopes[0].content != buttonOwnerContent ||
		len(reports) != 1 || reports[0] != original || !reflect.DeepEqual(view.Warnings(), failureWarnings) || composer.CompositionActive() {
		t.Fatalf("GR65_RETRY_FIRST_FAILURE: envelopes=%#v reports=%#v warnings=%#v composition=%t",
			envelopes, reports, view.Warnings(), composer.CompositionActive())
	}
	t.Log("GR65_RETRY_FIRST_FAILURE_PASSED")

	// Re-focus and copy between the two real recovery clicks, proving that
	// retry consumes the unchanged confirmed input instead of a recreated draft.
	buttonOwnerClick(t, tester, ui.RoleTextField, buttonOwnerInputLabel)
	buttonOwnerFocusedInput(t, tester)
	tester.Command("selectAll")
	tester.Command("copy")
	tester.Frame()
	if got := tester.Clipboard(); got != buttonOwnerInput {
		t.Fatalf("GR65_RETRY_FAILED_INPUT_COPY: clipboard=%q want=%q", got, buttonOwnerInput)
	}
	buttonOwnerFocusedInput(t, tester)
	buttonOwnerSourceUnchanged(t, buffer, changed)
	if len(envelopes) != 1 || len(reports) != 1 || reports[0] != original || composer.CompositionActive() {
		t.Fatalf("GR65_RETRY_INTERMEDIATE_STATE: reports=%#v envelopes=%#v composition=%t", reports, envelopes, composer.CompositionActive())
	}
	t.Log("GR65_RETRY_FAILED_INPUT_PRESERVED")

	buttonOwnerClick(t, tester, ui.RoleButton, "恢复标签及节点草稿")
	if len(envelopes) != 2 || len(reports) != 1 || reports[0] != original || composer.CompositionActive() {
		t.Fatalf("GR65_RETRY_SECOND_SUCCESS_STATE: reports=%#v envelopes=%#v composition=%t", reports, envelopes, composer.CompositionActive())
	}
	for index, envelope := range envelopes {
		if envelope.title != buttonOwnerTitle || envelope.content != buttonOwnerContent {
			t.Fatalf("GR65_RETRY_TWO_EXACT_ENVELOPES: index=%d got=%#v wantTitle=%q wantContent=%q", index, envelope, buttonOwnerTitle, buttonOwnerContent)
		}
	}
	t.Log("GR65_RETRY_SECOND_SUCCESS_PASSED")

	buttonControlEmptyInput(t, tester, "GR65_RETRY")
	buttonOwnerSourceUnchanged(t, buffer, changed)
	if len(envelopes) != 2 || len(reports) != 1 || reports[0] != original || composer.CompositionActive() {
		t.Fatalf("GR65_RETRY_FINAL_PUBLIC_STATE: reports=%#v envelopes=%#v composition=%t", reports, envelopes, composer.CompositionActive())
	}
	t.Log("GR65_RETRY_INPUT_AND_SOURCE_PRESERVED")

	warnings := view.Warnings()
	t.Logf("GR65_RETRY_DIAGNOSTICS_OBSERVED: reports=%d warnings=%#v original=%p", len(reports), warnings, original)
	for index, report := range reports {
		t.Logf("GR65_RETRY_REPORT_OBJECT: index=%d object=%p originalObject=%t error=%v", index, report, report == original, report)
	}
	if !reflect.DeepEqual(warnings, compatibility) {
		t.Errorf("GR65_RETRY_EXACT_WARNINGS: got=%#v want=%#v", warnings, compatibility)
	}
}
