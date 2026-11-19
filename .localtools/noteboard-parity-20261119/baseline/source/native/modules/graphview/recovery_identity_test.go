package graphview_test

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/modules/graphview"
	"levnote.local/native/ui/editor"
)

// BEGIN FIRST GRAPH RECOVERY ORACLE
const graphRecoverySource = "{\r\n" +
	`"format":"levnote.mindmap","version":1,"layout":"logicalStructure","root":{"id":"recovery-root","topic":"原主题🙂","children":[{"id":"recovery-child","topic":"子主题🙂","children":[]}],"tags":[{"text":"既存🙂","future":1.20}],"futureNode":{"keep":"原样🙂"}},"futureTop":{"n":2.00}}` + "\r\n"

const graphRecoveryTagLabel = "新标签（单行，最多512字节）"

type graphRecoveryView interface {
	RecoverDraft() error
	CompositionActive() bool
	Warnings() []string
}

func graphRecoveryClick(t *testing.T, tester *ui.Tester, label string) {
	t.Helper()
	if err := tester.Click(label); err != nil {
		t.Fatalf("public fixture click %q: %v; visible=%q", label, err, tester.Texts())
	}
}

func graphRecoveryInput(t *testing.T, tester *ui.Tester, label, want string) {
	t.Helper()
	input, ok := tester.AccessByRole(ui.RoleTextField, label)
	// The native editor is a self-drawn text client; AccessInfo.Value is not
	// bound to its text. This observes only the real public input entrance.
	// Exact pending bytes are checked by the literal Host recovery envelopes.
	if !ok || !input.Focusable || input.Disabled {
		t.Fatalf("public input %q: found=%t focusable=%t disabled=%t intended=%q", label, ok, input.Focusable, input.Disabled, want)
	}
}

func graphRecoverySourceUnchanged(t *testing.T, buffer *text.Buffer, changed int) {
	t.Helper()
	if buffer.String() != graphRecoverySource || buffer.Version() != 0 ||
		buffer.Selection() != (text.Range{Anchor: 23, Head: 7}) || buffer.InGroup() ||
		buffer.History().UndoDepth != 0 || buffer.History().RedoDepth != 0 || changed != 0 {
		t.Fatalf("recovery touched authoritative source: text=%q version=%d selection=%+v history=%+v group=%t changed=%d",
			buffer.String(), buffer.Version(), buffer.Selection(), buffer.History(), buffer.InGroup(), changed)
	}
}

func TestGraphRecoveryIdentityTagSameTextNewComposition(t *testing.T) {
	buffer := text.New(graphRecoverySource)
	if err := buffer.SetSelection(23, 7); err != nil {
		t.Fatal(err)
	}
	changed, calls := 0, 0
	var reports []error
	var tester *ui.Tester
	var recovery graphRecoveryView
	view := graphview.MindmapAdapter().Create(editor.Host{
		Buffer:  buffer,
		Path:    "recovery.mindmap",
		Changed: func() { changed++ },
		Report:  func(err error) { reports = append(reports, err) },
		RecoverDraft: func(title, content string) error {
			calls++
			if title != "recovery.mindmap-节点草稿.txt" ||
				content != "recovery.mindmap\n未添加的标签输入\n节点 ID: recovery-root\n字段: 新标签（尚未添加）\n\n标签🙂\r\n原样" {
				t.Fatalf("initiating tag-only literal changed: title=%q content=%q", title, content)
			}
			if calls == 1 {
				graphRecoveryInput(t, tester, graphRecoveryTagLabel, "标签🙂\r\n原样")
				if recovery.CompositionActive() {
					t.Fatal("fixture must start with a confirmed tag, not an old composition")
				}
				// Genuine selectAll then same-text composition replaces the full
				// literal without changing its bytes. Its rune caret 7 is UTF-16
				// offset 8, the same final selection as the initiating Type.
				// The public editor/text no-op contract keeps the input version;
				// private input Buffer/version/selection are deliberately not read.
				tester.Command("selectAll")
				tester.Compose("标签🙂\r\n原样", 7)
				graphRecoveryInput(t, tester, graphRecoveryTagLabel, "标签🙂\r\n原样")
				if !recovery.CompositionActive() {
					t.Fatal("public callback did not accept the new same-text composition")
				}
				t.Log("callback accepted a new composition with the exact initiating tag literal")
			}
			return nil
		},
	})
	t.Cleanup(view.Release)
	var ok bool
	recovery, ok = view.(graphRecoveryView)
	if !ok {
		t.Fatal("public mindmap view lacks the existing recovery/composition/warnings capabilities")
	}
	tester = ui.NewTester(view.Draw, 960, 900)
	graphRecoveryClick(t, tester, "展开标签面板")
	graphRecoveryClick(t, tester, graphRecoveryTagLabel)
	if active, noInputMethod := tester.TextInputActive(); !active || noInputMethod {
		t.Fatal("public tag click did not focus the native input method")
	}
	tester.Type("标签🙂\r\n原样")
	graphRecoveryInput(t, tester, graphRecoveryTagLabel, "标签🙂\r\n原样")
	graphRecoverySourceUnchanged(t, buffer, changed)
	if recovery.CompositionActive() || len(reports) != 0 {
		t.Fatal("confirmed initiating tag fixture changed composition/report state", reports)
	}
	if err := recovery.RecoverDraft(); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || !recovery.CompositionActive() {
		t.Fatalf("RECOVERY_REPLACED_NEW_TAG_COMPOSITION: calls=%d composition=%t", calls, recovery.CompositionActive())
	}
	tester.Frame()
	graphRecoveryInput(t, tester, graphRecoveryTagLabel, "标签🙂\r\n原样")
	graphRecoverySourceUnchanged(t, buffer, changed)
	if len(reports) != 0 {
		t.Fatal("successful old recovery reported against the surviving input", reports)
	}
	// An unchanged second recovery must close that surviving composition.
	// This rules out blanket preservation of every active composition.
	if err := recovery.RecoverDraft(); err != nil || calls != 2 || recovery.CompositionActive() {
		t.Fatalf("unchanged successor recovery: err=%v calls=%d composition=%t", err, calls, recovery.CompositionActive())
	}
	tester.Frame()
	graphRecoveryInput(t, tester, graphRecoveryTagLabel, "")
	graphRecoverySourceUnchanged(t, buffer, changed)
	warnings := recovery.Warnings()
	if len(reports) != 0 || len(warnings) != 2 ||
		warnings[0] != "node \"recovery-root\" has 1 preserved unknown field(s)" ||
		warnings[1] != "preserved 1 unknown top-level field(s)" {
		t.Fatal("ordinary successor recovery added an error or lost original compatibility warnings", reports, warnings)
	}
}

// END FIRST GRAPH RECOVERY ORACLE

// This fixture owns only public objects and callback observations.
// No private graph/editor draft or input Buffer is exposed to the tests.
type graphRecoveryHarness struct {
	view     editor.View
	recovery graphRecoveryView
	tester   *ui.Tester
	buffer   *text.Buffer
	changed  int
	calls    int
	reports  []error
}

func graphRecoveryMount(t *testing.T, callback func(string, string) error) *graphRecoveryHarness {
	t.Helper()
	h := &graphRecoveryHarness{buffer: text.New(graphRecoverySource)}
	if err := h.buffer.SetSelection(23, 7); err != nil {
		t.Fatal(err)
	}
	host := editor.Host{
		Buffer:  h.buffer,
		Path:    "recovery.mindmap",
		Changed: func() { h.changed++ },
		Report:  func(err error) { h.reports = append(h.reports, err) },
	}
	if callback != nil {
		host.RecoverDraft = func(title, content string) error {
			h.calls++
			return callback(title, content)
		}
	}
	h.view = graphview.MindmapAdapter().Create(host)
	t.Cleanup(h.view.Release)
	var ok bool
	h.recovery, ok = h.view.(graphRecoveryView)
	if !ok {
		t.Fatal("public graph recovery capabilities missing")
	}
	h.tester = ui.NewTester(h.view.Draw, 960, 900)
	return h
}

func graphRecoveryQuiet(t *testing.T, h *graphRecoveryHarness) {
	t.Helper()
	warnings := h.recovery.Warnings()
	if len(warnings) != 2 || warnings[0] != "node \"recovery-root\" has 1 preserved unknown field(s)" ||
		warnings[1] != "preserved 1 unknown top-level field(s)" {
		t.Fatalf("recovery changed original compatibility warnings: %q", warnings)
	}
}

func graphRecoveryFocusTag(t *testing.T, h *graphRecoveryHarness) {
	t.Helper()
	graphRecoveryClick(t, h.tester, "展开标签面板")
	graphRecoveryClick(t, h.tester, graphRecoveryTagLabel)
	if active, noInputMethod := h.tester.TextInputActive(); !active || noInputMethod {
		t.Fatal("genuine new-tag click did not focus the native text client")
	}
}

func TestGraphRecoveryIdentityNoPendingIsQuiet(t *testing.T) {
	h := graphRecoveryMount(t, func(string, string) error {
		t.Fatal("a clean graph must not invoke Host.RecoverDraft")
		return nil
	})
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 0 || h.recovery.CompositionActive() {
		t.Fatalf("clean recovery changed state: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	h.tester.Frame()
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
	if len(h.reports) != 0 {
		t.Fatal("clean recovery reported an error", h.reports)
	}
}

func TestGraphRecoveryIdentityNoHostPreservesComposition(t *testing.T) {
	h := graphRecoveryMount(t, nil)
	graphRecoveryFocusTag(t, h)
	h.tester.Compose("宿主缺失🙂\r\n原样", 9)
	if !h.recovery.CompositionActive() {
		t.Fatal("no-host fixture did not compose")
	}
	err := h.recovery.RecoverDraft()
	if err == nil || err.Error() != "当前工作区不能创建草稿副本；节点草稿仍保留" ||
		len(h.reports) != 1 || h.reports[0] != err || !h.recovery.CompositionActive() {
		t.Fatalf("no-host recovery lost the initiating owner/error: err=%v reports=%v composing=%t", err, h.reports, h.recovery.CompositionActive())
	}
	if next := h.recovery.RecoverDraft(); next == nil || next.Error() != err.Error() || len(h.reports) != 1 || !h.recovery.CompositionActive() {
		t.Fatalf("same no-host retry changed the retained draft/error: err=%v reports=%v", next, h.reports)
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	h.tester.Type("宿主缺失🙂\r\n原样")
	h.tester.Command("selectAll")
	h.tester.Command("copy")
	if h.tester.Clipboard() != "宿主缺失🙂\r\n原样" || h.recovery.CompositionActive() {
		t.Fatalf("no-host draft was not recoverable through real copy: %q", h.tester.Clipboard())
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
}

func TestGraphRecoveryIdentityOriginalErrorRetry(t *testing.T) {
	original := errors.New("graph recovery host fault🙂")
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		if title != "recovery.mindmap-节点草稿.txt" ||
			content != "recovery.mindmap\n未添加的标签输入\n节点 ID: recovery-root\n字段: 新标签（尚未添加）\n\n失败重试🙂\r\n原样" {
			t.Fatalf("unchanged retry lost initiating literal: title=%q content=%q", title, content)
		}
		if h.calls < 3 {
			return original
		}
		return nil
	})
	graphRecoveryFocusTag(t, h)
	h.tester.Compose("失败重试🙂\r\n原样", 9)
	for attempt := 1; attempt <= 2; attempt++ {
		if err := h.recovery.RecoverDraft(); err != original || h.calls != attempt ||
			len(h.reports) != 1 || h.reports[0] != original || !h.recovery.CompositionActive() {
			t.Fatalf("original failure/retry %d: err=%v calls=%d reports=%v composing=%t", attempt, err, h.calls, h.reports, h.recovery.CompositionActive())
		}
		warnings := h.recovery.Warnings()
		if len(warnings) != 3 || warnings[2] != "节点草稿尚未写回：graph recovery host fault🙂" {
			t.Fatalf("initiating owner did not retain its original error: %q", warnings)
		}
		graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	}
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 3 || h.recovery.CompositionActive() {
		t.Fatalf("unchanged successful retry did not settle: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	h.tester.Frame()
	graphRecoveryQuiet(t, h)
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	if len(h.reports) != 1 {
		t.Fatal("successful retry reported a new error", h.reports)
	}
}

func TestGraphRecoveryIdentityFieldSameTextNewComposition(t *testing.T) {
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		if title != "recovery.mindmap-节点草稿.txt" ||
			content != "recovery.mindmap\n节点 ID: recovery-root\n页面 ID: \n字段: 节点文本\n原字段存在: true\n\n原主题🙂" {
			t.Fatalf("field-only literal envelope changed: title=%q content=%q", title, content)
		}
		if h.calls == 1 {
			h.tester.Compose("原主题🙂", 4)
			if !h.recovery.CompositionActive() {
				t.Fatal("callback did not accept the same-literal new field composition")
			}
			t.Log("callback accepted a new field composition with identical text and caret")
		}
		return nil
	})
	graphRecoveryClick(t, h.tester, "节点文本")
	h.tester.Command("selectAll")
	h.tester.Compose("原主题🙂", 4)
	if !h.recovery.CompositionActive() || len(h.reports) != 0 {
		t.Fatal("initial field composition was not accepted", h.reports)
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 1 || !h.recovery.CompositionActive() {
		t.Fatalf("RECOVERY_REPLACED_NEW_FIELD_COMPOSITION: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	h.tester.Frame()
	graphRecoveryInput(t, h.tester, "节点文本", "原主题🙂")
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 || h.recovery.CompositionActive() {
		t.Fatalf("unchanged field composition did not settle: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	h.tester.Frame()
	graphRecoveryQuiet(t, h)
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	if len(h.reports) != 0 {
		t.Fatal("field recovery reported an unrelated error", h.reports)
	}
}

func TestGraphRecoveryIdentityTagNewCompositionFailure(t *testing.T) {
	original := errors.New("old tag recovery failure🙂")
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		if title != "recovery.mindmap-节点草稿.txt" ||
			content != "recovery.mindmap\n未添加的标签输入\n节点 ID: recovery-root\n字段: 新标签（尚未添加）\n\n新所有者🙂\r\n原样" {
			t.Fatalf("failed recovery changed initiating literal: title=%q content=%q", title, content)
		}
		if h.calls == 1 {
			h.tester.Command("selectAll")
			h.tester.Compose("新所有者🙂\r\n原样", 9)
			if !h.recovery.CompositionActive() {
				t.Fatal("failed host callback did not install the new composition")
			}
			t.Log("failed host accepted a same-text successor tag composition")
			return original
		}
		return nil
	})
	graphRecoveryFocusTag(t, h)
	h.tester.Type("新所有者🙂\r\n原样")
	if err := h.recovery.RecoverDraft(); err != original || h.calls != 1 ||
		len(h.reports) != 1 || h.reports[0] != original || !h.recovery.CompositionActive() {
		t.Fatalf("old recovery failure lost the original error/new input: err=%v calls=%d reports=%v composing=%t", err, h.calls, h.reports, h.recovery.CompositionActive())
	}
	warnings := h.recovery.Warnings()
	if len(warnings) != 2 || warnings[0] != "node \"recovery-root\" has 1 preserved unknown field(s)" ||
		warnings[1] != "preserved 1 unknown top-level field(s)" {
		t.Fatalf("RECOVERY_ATTACHED_OLD_FAILURE_TO_NEW_TAG: %q", warnings)
	}
	h.tester.Frame()
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 || h.recovery.CompositionActive() {
		t.Fatalf("successor tag recovery failed: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	h.tester.Frame()
	graphRecoveryQuiet(t, h)
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	if len(h.reports) != 1 {
		t.Fatal("successor recovery replayed the old host error", h.reports)
	}
}

func TestGraphRecoveryIdentityTagSameLiteralConfirmation(t *testing.T) {
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		if title != "recovery.mindmap-节点草稿.txt" ||
			content != "recovery.mindmap\n未添加的标签输入\n节点 ID: recovery-root\n字段: 新标签（尚未添加）\n\n确认🙂\r\n原样" {
			t.Fatalf("accepted confirmation lost literal: title=%q content=%q", title, content)
		}
		if h.calls == 1 {
			h.tester.Type("确认🙂\r\n原样")
			if h.recovery.CompositionActive() {
				t.Fatal("genuine same-literal Type did not confirm the initiating composition")
			}
		}
		return nil
	})
	graphRecoveryFocusTag(t, h)
	h.tester.Compose("确认🙂\r\n原样", 7)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 1 || h.recovery.CompositionActive() {
		t.Fatalf("same-literal confirmation recovery: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	h.tester.Command("selectAll")
	h.tester.Command("copy")
	if h.tester.Clipboard() != "确认🙂\r\n原样" {
		t.Fatalf("RECOVERY_REPLACED_ACCEPTED_TAG_CONFIRMATION: %q", h.tester.Clipboard())
	}
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 {
		t.Fatalf("unchanged confirmed successor was not recoverable: err=%v calls=%d", err, h.calls)
	}
	h.tester.Frame()
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 {
		t.Fatalf("settled tag unexpectedly remained pending: err=%v calls=%d", err, h.calls)
	}
	graphRecoveryQuiet(t, h)
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	if len(h.reports) != 0 {
		t.Fatal("confirmation control reported an error", h.reports)
	}
}

func TestGraphRecoveryIdentityTagSameEmptyComposition(t *testing.T) {
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		if title != "recovery.mindmap-节点草稿.txt" ||
			content != "recovery.mindmap\n未添加的标签输入\n节点 ID: recovery-root\n字段: 新标签（尚未添加）\n\n" {
			t.Fatalf("empty tag composition lost exact envelope: title=%q content=%q", title, content)
		}
		if h.calls == 1 {
			h.tester.Compose("", 0)
			if !h.recovery.CompositionActive() {
				t.Fatal("native text client did not accept the successor empty composition")
			}
		}
		return nil
	})
	graphRecoveryFocusTag(t, h)
	h.tester.Compose("", 0)
	if !h.recovery.CompositionActive() {
		t.Fatal("native new-tag text client did not accept the initiating empty composition")
	}
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 1 || !h.recovery.CompositionActive() {
		t.Fatalf("RECOVERY_REPLACED_NEW_EMPTY_TAG_COMPOSITION: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	h.tester.Frame()
	graphRecoveryQuiet(t, h)
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 || h.recovery.CompositionActive() {
		t.Fatalf("unchanged empty composition was blanket protected: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	h.tester.Frame()
	graphRecoveryQuiet(t, h)
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	if len(h.reports) != 0 {
		t.Fatal("empty composition control reported an error", h.reports)
	}
}

func TestGraphRecoveryIdentityAbsentFieldEmptyComposition(t *testing.T) {
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		if title != "recovery.mindmap-节点草稿.txt" ||
			content != "recovery.mindmap\n节点 ID: recovery-root\n页面 ID: \n字段: 节点备注（纯文本）\n原字段存在: false\n\n" {
			t.Fatalf("absent field empty literal envelope changed: title=%q content=%q", title, content)
		}
		if h.calls == 1 {
			h.tester.Compose("", 0)
			if !h.recovery.CompositionActive() {
				t.Fatal("callback did not accept a same-empty field composition")
			}
		}
		return nil
	})
	graphRecoveryClick(t, h.tester, "字段：纯文本备注")
	graphRecoveryClick(t, h.tester, "节点备注（纯文本）")
	h.tester.Compose("", 0)
	if !h.recovery.CompositionActive() {
		t.Fatal("empty absent-field composition was not accepted")
	}
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 1 || !h.recovery.CompositionActive() {
		t.Fatalf("RECOVERY_REPLACED_NEW_EMPTY_FIELD_COMPOSITION: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 || h.recovery.CompositionActive() {
		t.Fatalf("unchanged empty field was blanket protected: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	h.tester.Frame()
	graphRecoveryQuiet(t, h)
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	if len(h.reports) != 0 {
		t.Fatal("empty absent-field recovery reported an error", h.reports)
	}
}

func TestGraphRecoveryIdentityFieldNewCompositionFailure(t *testing.T) {
	original := errors.New("old field recovery failure🙂")
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		if title != "recovery.mindmap-节点草稿.txt" ||
			content != "recovery.mindmap\n节点 ID: recovery-root\n页面 ID: \n字段: 节点文本\n原字段存在: true\n\n原主题🙂" {
			t.Fatalf("field failure lost initiating literal: title=%q content=%q", title, content)
		}
		if h.calls == 1 {
			h.tester.Compose("原主题🙂", 4)
			if !h.recovery.CompositionActive() {
				t.Fatal("field failure callback did not accept its successor composition")
			}
			return original
		}
		return nil
	})
	graphRecoveryClick(t, h.tester, "节点文本")
	h.tester.Command("selectAll")
	h.tester.Compose("原主题🙂", 4)
	if err := h.recovery.RecoverDraft(); err != original || h.calls != 1 ||
		len(h.reports) != 1 || h.reports[0] != original || !h.recovery.CompositionActive() {
		t.Fatalf("original field failure/successor state: err=%v calls=%d reports=%v composing=%t", err, h.calls, h.reports, h.recovery.CompositionActive())
	}
	graphRecoveryQuiet(t, h)
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 || h.recovery.CompositionActive() {
		t.Fatalf("unchanged successor field recovery: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	h.tester.Frame()
	graphRecoveryQuiet(t, h)
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	if len(h.reports) != 1 {
		t.Fatal("field successor replayed original host failure", h.reports)
	}
}

func TestGraphRecoveryIdentityTagSelectionOnly(t *testing.T) {
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		if title != "recovery.mindmap-节点草稿.txt" ||
			content != "recovery.mindmap\n未添加的标签输入\n节点 ID: recovery-root\n字段: 新标签（尚未添加）\n\n选区🙂\r\n原样" {
			t.Fatalf("selection-only initiating envelope: title=%q content=%q", title, content)
		}
		if h.calls == 1 {
			h.tester.Key(ui.Shift, ui.KeyLeft)
			h.tester.Command("copy")
			if h.tester.Clipboard() != "样" {
				t.Fatalf("native reverse selection did not select the final character: %q", h.tester.Clipboard())
			}
		}
		return nil
	})
	graphRecoveryFocusTag(t, h)
	h.tester.Type("选区🙂\r\n原样")
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 1 || h.recovery.CompositionActive() {
		t.Fatalf("selection-only recovery: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	h.tester.Command("copy")
	if h.tester.Clipboard() != "样" {
		t.Fatal("recovery lost the callback reverse selection", h.tester.Clipboard())
	}
	h.tester.Command("selectAll")
	h.tester.Command("copy")
	if h.tester.Clipboard() != "选区🙂\r\n原样" {
		t.Fatalf("RECOVERY_REPLACED_CHANGED_TAG_SELECTION: %q", h.tester.Clipboard())
	}
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 {
		t.Fatalf("unchanged selection successor recovery: err=%v calls=%d", err, h.calls)
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
	if len(h.reports) != 0 {
		t.Fatal("selection-only recovery reported an error", h.reports)
	}
}

func TestGraphRecoveryIdentityTagDifferentTextVersion(t *testing.T) {
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		want := "recovery.mindmap\n未添加的标签输入\n节点 ID: recovery-root\n字段: 新标签（尚未添加）\n\n最初🙂\r\n原样"
		if h.calls > 1 {
			want = "recovery.mindmap\n未添加的标签输入\n节点 ID: recovery-root\n字段: 新标签（尚未添加）\n\n新文字🙂\r\n后继"
		}
		if title != "recovery.mindmap-节点草稿.txt" || content != want {
			t.Fatalf("different-text literal envelope: title=%q content=%q", title, content)
		}
		if h.calls == 1 {
			h.tester.Command("selectAll")
			h.tester.Type("新文字🙂\r\n后继")
		}
		return nil
	})
	graphRecoveryFocusTag(t, h)
	h.tester.Type("最初🙂\r\n原样")
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 1 {
		t.Fatalf("different-text recovery: err=%v calls=%d", err, h.calls)
	}
	h.tester.Command("selectAll")
	h.tester.Command("copy")
	if h.tester.Clipboard() != "新文字🙂\r\n后继" {
		t.Fatalf("RECOVERY_REPLACED_CHANGED_TAG_TEXT_VERSION: %q", h.tester.Clipboard())
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 {
		t.Fatalf("unchanged different-text successor: err=%v calls=%d", err, h.calls)
	}
	h.tester.Frame()
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
	if len(h.reports) != 0 {
		t.Fatal("different-text recovery reported an error", h.reports)
	}
}

func TestGraphRecoveryIdentityFieldDifferentTextComposition(t *testing.T) {
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		want := "recovery.mindmap\n节点 ID: recovery-root\n页面 ID: \n字段: 节点文本\n原字段存在: true\n\n第一字段🙂\r\n原样"
		if h.calls > 1 {
			want = "recovery.mindmap\n节点 ID: recovery-root\n页面 ID: \n字段: 节点文本\n原字段存在: true\n\n后继字段🙂\r\n原样"
		}
		if title != "recovery.mindmap-节点草稿.txt" || content != want {
			t.Fatalf("different field composition envelope: title=%q content=%q", title, content)
		}
		if h.calls == 1 {
			h.tester.Compose("后继字段🙂\r\n原样", 10)
		}
		return nil
	})
	graphRecoveryClick(t, h.tester, "节点文本")
	h.tester.Command("selectAll")
	h.tester.Compose("第一字段🙂\r\n原样", 10)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 1 || !h.recovery.CompositionActive() {
		t.Fatalf("RECOVERY_REPLACED_CHANGED_FIELD_TEXT_VERSION: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 || h.recovery.CompositionActive() {
		t.Fatalf("unchanged changed-text field recovery: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	h.tester.Frame()
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
	if len(h.reports) != 0 {
		t.Fatal("different field composition recovery reported an error", h.reports)
	}
}

func TestGraphRecoveryIdentityFieldSurvivesTagCloses(t *testing.T) {
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		want := "recovery.mindmap\n节点 ID: recovery-root\n页面 ID: \n字段: 节点文本\n原字段存在: true\n\n待字段🙂\r\n原样\n\n--- 未添加的标签输入 ---\n节点 ID: recovery-root\n字段: 新标签（尚未添加）\n\n待标签🙂\r\n原样"
		if h.calls == 2 {
			want = "recovery.mindmap\n节点 ID: recovery-root\n页面 ID: \n字段: 节点文本\n原字段存在: true\n\n待字段🙂\r\n原样"
		}
		if title != "recovery.mindmap-节点草稿.txt" || content != want {
			t.Fatalf("field-survives independent envelope: title=%q content=%q", title, content)
		}
		if h.calls == 1 {
			graphRecoveryClick(t, h.tester, "节点文本")
			h.tester.Command("selectAll")
			h.tester.Compose("待字段🙂\r\n原样", 9)
		}
		return nil
	})
	graphRecoveryFocusTag(t, h)
	graphRecoveryClick(t, h.tester, "节点文本")
	h.tester.Command("selectAll")
	h.tester.Compose("待字段🙂\r\n原样", 9)
	graphRecoveryClick(t, h.tester, graphRecoveryTagLabel)
	h.tester.Type("待标签🙂\r\n原样")
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 1 || !h.recovery.CompositionActive() {
		t.Fatalf("field successor/old tag recovery: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 || h.recovery.CompositionActive() {
		t.Fatalf("field-only successor did not close: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 {
		t.Fatalf("closed tag/field remained pending: err=%v calls=%d", err, h.calls)
	}
	h.tester.Frame()
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
	if len(h.reports) != 0 {
		t.Fatal("independent field/tag clearing reported an error", h.reports)
	}
}

func TestGraphRecoveryIdentityTagSurvivesFieldCloses(t *testing.T) {
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		want := "recovery.mindmap\n节点 ID: recovery-root\n页面 ID: \n字段: 节点文本\n原字段存在: true\n\n双待字段🙂\r\n原样\n\n--- 未添加的标签输入 ---\n节点 ID: recovery-root\n字段: 新标签（尚未添加）\n\n双待标签🙂\r\n原样"
		if h.calls == 2 {
			want = "recovery.mindmap\n未添加的标签输入\n节点 ID: recovery-root\n字段: 新标签（尚未添加）\n\n双待标签🙂\r\n原样"
		}
		if title != "recovery.mindmap-节点草稿.txt" || content != want {
			t.Fatalf("tag-survives independent envelope: title=%q content=%q", title, content)
		}
		if h.calls == 1 {
			h.tester.Command("selectAll")
			h.tester.Compose("双待标签🙂\r\n原样", 10)
		}
		return nil
	})
	graphRecoveryFocusTag(t, h)
	graphRecoveryClick(t, h.tester, "节点文本")
	h.tester.Command("selectAll")
	h.tester.Compose("双待字段🙂\r\n原样", 10)
	graphRecoveryClick(t, h.tester, graphRecoveryTagLabel)
	h.tester.Type("双待标签🙂\r\n原样")
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 1 || !h.recovery.CompositionActive() {
		t.Fatalf("tag successor/old field recovery: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 || h.recovery.CompositionActive() {
		t.Fatalf("tag-only successor did not close: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 {
		t.Fatalf("closed field/tag remained pending: err=%v calls=%d", err, h.calls)
	}
	h.tester.Frame()
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
	if len(h.reports) != 0 {
		t.Fatal("independent tag/field clearing reported an error", h.reports)
	}
}

func TestGraphRecoveryIdentityFailedAdmissionDoesNotOwnTag(t *testing.T) {
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		if title != "recovery.mindmap-节点草稿.txt" ||
			content != "recovery.mindmap\n未添加的标签输入\n节点 ID: recovery-root\n字段: 新标签（尚未添加）\n\n有效🙂\r\n原样" {
			t.Fatalf("failed admission initiating envelope: title=%q content=%q", title, content)
		}
		h.tester.Type("\xff")
		if len(h.reports) != 1 || h.reports[0] != text.ErrUTF8 {
			t.Fatalf("native input admission did not report its original UTF8 error: %v", h.reports)
		}
		return nil
	})
	graphRecoveryFocusTag(t, h)
	h.tester.Type("有效🙂\r\n原样")
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 1 || h.recovery.CompositionActive() {
		t.Fatalf("valid initiating draft recovery: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 1 {
		t.Fatalf("FAILED_ADMISSION_BLANKET_PROTECTED_OLD_TAG: err=%v calls=%d", err, h.calls)
	}
	h.tester.Frame()
	graphRecoveryClick(t, h.tester, graphRecoveryTagLabel)
	h.tester.Command("selectAll")
	h.tester.Command("copy")
	if h.tester.Clipboard() != "" {
		t.Fatalf("old valid tag was retained after failed admission: %q", h.tester.Clipboard())
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	warnings := h.recovery.Warnings()
	if len(warnings) != 3 || warnings[0] != "node \"recovery-root\" has 1 preserved unknown field(s)" ||
		warnings[1] != "preserved 1 unknown top-level field(s)" || warnings[2] != "标签操作：text is not valid UTF-8" ||
		len(h.reports) != 1 || h.reports[0] != text.ErrUTF8 {
		t.Fatalf("real admission failure was suppressed or relabeled: reports=%v warnings=%q", h.reports, warnings)
	}
}

func TestGraphRecoveryIdentityFieldSameLiteralConfirmation(t *testing.T) {
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		if title != "recovery.mindmap-节点草稿.txt" ||
			content != "recovery.mindmap\n节点 ID: recovery-root\n页面 ID: \n字段: 节点文本\n原字段存在: true\n\n原主题🙂" {
			t.Fatalf("property confirmation literal: title=%q content=%q", title, content)
		}
		if h.calls != 1 {
			t.Fatal("a confirmed unchanged property must become clean", h.calls)
		}
		h.tester.Type("原主题🙂")
		return nil
	})
	graphRecoveryClick(t, h.tester, "节点文本")
	h.tester.Command("selectAll")
	h.tester.Compose("原主题🙂", 4)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 1 || h.recovery.CompositionActive() {
		t.Fatalf("accepted property confirmation: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	h.tester.Command("selectAll")
	h.tester.Command("copy")
	if h.tester.Clipboard() != "原主题🙂" {
		t.Fatalf("RECOVERY_RELEASED_ACCEPTED_PROPERTY_CONFIRMATION: %q", h.tester.Clipboard())
	}
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 1 {
		t.Fatalf("confirmed property was not clean: err=%v calls=%d", err, h.calls)
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
	if len(h.reports) != 0 {
		t.Fatal("property confirmation changed reports", h.reports)
	}
}

func TestGraphRecoveryIdentityReplacementFieldFailure(t *testing.T) {
	original := errors.New("old property recovery failure🙂")
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		want := "recovery.mindmap\n节点 ID: recovery-root\n页面 ID: \n字段: 节点文本\n原字段存在: true\n\n原主题🙂"
		if h.calls > 1 {
			want = "recovery.mindmap\n节点 ID: recovery-root\n页面 ID: \n字段: 节点备注（纯文本）\n原字段存在: false\n\n"
		}
		if title != "recovery.mindmap-节点草稿.txt" || content != want {
			t.Fatalf("replacement-field recovery envelope: title=%q content=%q", title, content)
		}
		if h.calls == 1 {
			// The ordinary selector confirms the old source-equal composition
			// through Flush, then creates the different, absent property input.
			graphRecoveryClick(t, h.tester, "字段：纯文本备注")
			graphRecoveryClick(t, h.tester, "节点备注（纯文本）")
			h.tester.Compose("", 0)
			if !h.recovery.CompositionActive() {
				t.Fatal("public property selector did not install its new input")
			}
			return original
		}
		return nil
	})
	graphRecoveryClick(t, h.tester, "节点文本")
	h.tester.Command("selectAll")
	h.tester.Compose("原主题🙂", 4)
	if err := h.recovery.RecoverDraft(); err != original || h.calls != 1 || !h.recovery.CompositionActive() ||
		len(h.reports) != 1 || h.reports[0] != original {
		t.Fatalf("replacement property owner/error: err=%v calls=%d reports=%v composing=%t", err, h.calls, h.reports, h.recovery.CompositionActive())
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 || h.recovery.CompositionActive() {
		t.Fatalf("replacement property's own recovery: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	h.tester.Frame()
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
	if len(h.reports) != 1 {
		t.Fatal("replacement property replayed the old host failure", h.reports)
	}
}

func TestGraphRecoveryIdentityNestedFieldInputRestored(t *testing.T) {
	original := errors.New("nested field recovery failure🙂")
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		if title != "recovery.mindmap-节点草稿.txt" ||
			content != "recovery.mindmap\n节点 ID: recovery-root\n页面 ID: \n字段: 节点文本\n原字段存在: true\n\n原主题🙂" {
			t.Fatalf("nested field literal: title=%q content=%q", title, content)
		}
		switch h.calls {
		case 1:
			if err := h.recovery.RecoverDraft(); err != original || h.calls != 2 || !h.recovery.CompositionActive() {
				t.Fatalf("nested error lost its accepted input: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
			}
			// Inner defer must restore the still-active outer observer before
			// accepting this exact-text composition on the same editor.
			h.tester.Compose("原主题🙂", 4)
		case 2:
			h.tester.Compose("原主题🙂", 4)
			return original
		default:
			t.Fatal("confirmed nested property must be clean", h.calls)
		}
		return nil
	})
	graphRecoveryClick(t, h.tester, "节点文本")
	h.tester.Command("selectAll")
	h.tester.Compose("原主题🙂", 4)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 || !h.recovery.CompositionActive() ||
		len(h.reports) != 1 || h.reports[0] != original {
		t.Fatalf("NESTED_RECOVERY_REPLACED_FIELD_INPUT: err=%v calls=%d reports=%v composing=%t", err, h.calls, h.reports, h.recovery.CompositionActive())
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
	// The outer defer must restore the original property handler. F6 emits
	// only transient focus guidance; genuine confirmation clears that guidance
	// through the original OnChange -> Flush path without source mutation.
	h.tester.Key(0, ui.KeyF6)
	warnings := h.recovery.Warnings()
	if len(warnings) != 4 || warnings[2] != "节点草稿尚未写回：请先确认正在输入的文字，再移动到属性操作" ||
		warnings[3] != "标签操作：请先确认正在输入的文字，再移动到属性操作" || len(h.reports) != 1 {
		t.Fatalf("public property focus guidance: reports=%v warnings=%q", h.reports, warnings)
	}
	h.tester.Type("原主题🙂")
	if h.recovery.CompositionActive() {
		t.Fatal("genuine confirmation did not settle the retained composition")
	}
	graphRecoveryQuiet(t, h)
	h.tester.Command("selectAll")
	h.tester.Command("copy")
	if h.tester.Clipboard() != "原主题🙂" {
		t.Fatalf("restored property handler lost native copy: %q", h.tester.Clipboard())
	}
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 || len(h.reports) != 1 {
		t.Fatalf("nested field became spuriously pending: err=%v calls=%d reports=%v", err, h.calls, h.reports)
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
}

func TestGraphRecoveryIdentityNestedTagInputRestored(t *testing.T) {
	original := errors.New("nested tag recovery failure🙂")
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		if title != "recovery.mindmap-节点草稿.txt" ||
			content != "recovery.mindmap\n未添加的标签输入\n节点 ID: recovery-root\n字段: 新标签（尚未添加）\n\n嵌套标签🙂\r\n原样" {
			t.Fatalf("nested tag literal: title=%q content=%q", title, content)
		}
		switch h.calls {
		case 1:
			if err := h.recovery.RecoverDraft(); err != original || h.calls != 2 || !h.recovery.CompositionActive() {
				t.Fatalf("nested tag error lost its accepted input: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
			}
			h.tester.Compose("嵌套标签🙂\r\n原样", 10)
		case 2:
			h.tester.Compose("嵌套标签🙂\r\n原样", 10)
			return original
		case 3:
			// Unchanged recovery after both defers must close only this input.
		default:
			t.Fatal("recovered nested tag must be clean", h.calls)
		}
		return nil
	})
	graphRecoveryFocusTag(t, h)
	h.tester.Compose("嵌套标签🙂\r\n原样", 10)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 || !h.recovery.CompositionActive() ||
		len(h.reports) != 1 || h.reports[0] != original {
		t.Fatalf("NESTED_RECOVERY_REPLACED_TAG_INPUT: err=%v calls=%d reports=%v composing=%t", err, h.calls, h.reports, h.recovery.CompositionActive())
	}
	graphRecoveryQuiet(t, h)
	h.tester.Key(0, ui.KeyF6)
	warnings := h.recovery.Warnings()
	if len(warnings) != 3 || warnings[2] != "标签操作：请先确认正在输入的文字，再移动到属性操作" || len(h.reports) != 1 {
		t.Fatalf("public tag focus guidance: reports=%v warnings=%q", h.reports, warnings)
	}
	// Confirmation after both returns must still execute the original tag
	// OnChange, clearing only transient focus guidance before genuine copy.
	h.tester.Type("嵌套标签🙂\r\n原样")
	if h.recovery.CompositionActive() {
		t.Fatal("tag confirmation left its native composition active")
	}
	graphRecoveryQuiet(t, h)
	h.tester.Command("selectAll")
	h.tester.Command("copy")
	if h.tester.Clipboard() != "嵌套标签🙂\r\n原样" {
		t.Fatalf("restored tag handler lost native copy: %q", h.tester.Clipboard())
	}
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 3 {
		t.Fatalf("nested tag ordinary recovery: err=%v calls=%d", err, h.calls)
	}
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 3 || len(h.reports) != 1 {
		t.Fatalf("nested tag remained pending: err=%v calls=%d reports=%v", err, h.calls, h.reports)
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
}

func TestGraphRecoveryIdentityReplacementTagNodeFailure(t *testing.T) {
	original := errors.New("old root tag recovery failure🙂")
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		want := "recovery.mindmap\n未添加的标签输入\n节点 ID: recovery-root\n字段: 新标签（尚未添加）\n\n跨节点🙂\r\n原样"
		if h.calls > 1 {
			want = "recovery.mindmap\n未添加的标签输入\n节点 ID: recovery-child\n字段: 新标签（尚未添加）\n\n跨节点🙂\r\n原样"
		}
		if title != "recovery.mindmap-节点草稿.txt" || content != want {
			t.Fatalf("replacement-node tag envelope: title=%q content=%q", title, content)
		}
		if h.calls == 1 {
			// Discard is the real explicit local-input action, not recovery or
			// a private draft edit. Existing public outline navigation then
			// changes the node owner without changing document bytes/history.
			graphRecoveryClick(t, h.tester, "放弃标签输入")
			navigator, ok := h.view.(editor.OutlineNavigator)
			if !ok {
				t.Fatal("public outline navigation capability missing")
			}
			if err := navigator.NavigateOutlineItem("recovery-child"); err != nil {
				t.Fatal("public navigation after explicit discard", err)
			}
			h.tester.Frame()
			graphRecoveryClick(t, h.tester, graphRecoveryTagLabel)
			h.tester.Compose("跨节点🙂\r\n原样", 9)
			if !h.recovery.CompositionActive() {
				t.Fatal("replacement child tag input did not accept composition")
			}
			return original
		}
		return nil
	})
	graphRecoveryFocusTag(t, h)
	h.tester.Type("跨节点🙂\r\n原样")
	if err := h.recovery.RecoverDraft(); err != original || h.calls != 1 || !h.recovery.CompositionActive() ||
		len(h.reports) != 1 || h.reports[0] != original {
		t.Fatalf("RECOVERY_REPLACED_NEW_NODE_TAG_OWNER: err=%v calls=%d reports=%v composing=%t", err, h.calls, h.reports, h.recovery.CompositionActive())
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 || h.recovery.CompositionActive() {
		t.Fatalf("replacement child tag recovery: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 || len(h.reports) != 1 {
		t.Fatalf("replacement child tag remained pending: err=%v calls=%d reports=%v", err, h.calls, h.reports)
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
}

func TestGraphRecoveryIdentityReleaseEpochFailure(t *testing.T) {
	original := errors.New("pre-release graph recovery failure🙂")
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		if title != "recovery.mindmap-节点草稿.txt" ||
			content != "recovery.mindmap\n未添加的标签输入\n节点 ID: recovery-root\n字段: 新标签（尚未添加）\n\n释放后🙂\r\n原样" {
			t.Fatalf("released tag recovery literal: title=%q content=%q", title, content)
		}
		if h.calls == 1 {
			// No new text or composition is accepted here. Public Release
			// preserves the pending tag, while its defensive Flush emits the
			// existing field/tag pending guidance. The old Host error must be
			// reported separately instead of overwriting that released state.
			h.view.Release()
			return original
		}
		return nil
	})
	graphRecoveryFocusTag(t, h)
	h.tester.Type("释放后🙂\r\n原样")
	if err := h.recovery.RecoverDraft(); err != original || h.calls != 1 || h.recovery.CompositionActive() ||
		len(h.reports) != 3 || h.reports[0].Error() != "新标签尚未添加；请添加或放弃输入" ||
		h.reports[1].Error() != "新标签尚未添加；请添加或放弃输入" || h.reports[2] != original {
		t.Fatalf("released recovery error/report: err=%v calls=%d reports=%v composing=%t", err, h.calls, h.reports, h.recovery.CompositionActive())
	}
	warnings := h.recovery.Warnings()
	if len(warnings) != 4 || warnings[0] != "node \"recovery-root\" has 1 preserved unknown field(s)" ||
		warnings[1] != "preserved 1 unknown top-level field(s)" ||
		warnings[2] != "节点草稿尚未写回：新标签尚未添加；请添加或放弃输入" ||
		warnings[3] != "标签操作：新标签尚未添加；请添加或放弃输入" {
		t.Fatalf("RECOVERY_ATTACHED_PRE_RELEASE_FAILURE: %q", warnings)
	}
	h.tester.Frame()
	graphRecoveryClick(t, h.tester, graphRecoveryTagLabel)
	h.tester.Command("selectAll")
	h.tester.Command("copy")
	if h.tester.Clipboard() != "释放后🙂\r\n原样" {
		t.Fatalf("released view lost pending tag copy: %q", h.tester.Clipboard())
	}
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 || h.recovery.CompositionActive() {
		t.Fatalf("released tag retry: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 || len(h.reports) != 3 {
		t.Fatalf("released tag remained pending: err=%v calls=%d reports=%v", err, h.calls, h.reports)
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	warnings = h.recovery.Warnings()
	if len(warnings) != 3 || warnings[0] != "node \"recovery-root\" has 1 preserved unknown field(s)" ||
		warnings[1] != "preserved 1 unknown top-level field(s)" ||
		warnings[2] != "标签操作：新标签尚未添加；请添加或放弃输入" {
		t.Fatalf("released tag original pending guidance changed: %q", warnings)
	}
}

func TestGraphRecoveryIdentityTagSameLiteralTypeWithoutComposition(t *testing.T) {
	var h *graphRecoveryHarness
	h = graphRecoveryMount(t, func(title, content string) error {
		if title != "recovery.mindmap-节点草稿.txt" ||
			content != "recovery.mindmap\n未添加的标签输入\n节点 ID: recovery-root\n字段: 新标签（尚未添加）\n\n同字Type🙂\r\n原样" {
			t.Fatalf("noncomposing same-literal tag recovery: title=%q content=%q", title, content)
		}
		if h.calls == 1 {
			if h.recovery.CompositionActive() {
				t.Fatal("plain Type callback unexpectedly began with a composition")
			}
			h.tester.Command("selectAll")
			h.tester.Type("同字Type🙂\r\n原样")
			// With no composition, equal replacement still accepts the input
			// and collapses the full selection to the end. A reverse one-rune
			// copy proves that public postcondition independently of OnChange;
			// an ignored Type would leave the old full selection instead.
			h.tester.Key(ui.Shift, ui.KeyLeft)
			h.tester.Command("copy")
			if h.tester.Clipboard() != "样" || h.recovery.CompositionActive() || len(h.reports) != 0 {
				t.Fatalf("plain same-literal Type not accepted: copy=%q composing=%t reports=%v", h.tester.Clipboard(), h.recovery.CompositionActive(), h.reports)
			}
			h.tester.Key(0, ui.KeyRight)
			t.Log("callback accepted noncomposing same-literal Type and restored the initiating end selection")
		}
		return nil
	})
	graphRecoveryFocusTag(t, h)
	h.tester.Type("同字Type🙂\r\n原样")
	if h.recovery.CompositionActive() {
		t.Fatal("initial plain Type unexpectedly created a composition")
	}
	h.tester.Command("selectAll")
	h.tester.Command("copy")
	if h.tester.Clipboard() != "同字Type🙂\r\n原样" {
		t.Fatalf("initial plain Type literal copy: %q", h.tester.Clipboard())
	}
	h.tester.Key(0, ui.KeyRight)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 1 || h.recovery.CompositionActive() || len(h.reports) != 0 {
		t.Fatalf("plain Type first recovery: err=%v calls=%d composing=%t reports=%v", err, h.calls, h.recovery.CompositionActive(), h.reports)
	}
	h.tester.Frame()
	graphRecoveryClick(t, h.tester, graphRecoveryTagLabel)
	h.tester.Command("selectAll")
	h.tester.Command("copy")
	if h.tester.Clipboard() != "同字Type🙂\r\n原样" {
		t.Fatalf("RECOVERY_REPLACED_NONCOMPOSING_SAME_LITERAL_TYPE: copy=%q", h.tester.Clipboard())
	}
	h.tester.Key(0, ui.KeyRight)
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 || h.recovery.CompositionActive() {
		t.Fatalf("RECOVERY_DROPPED_NONCOMPOSING_TYPE_OWNER: err=%v calls=%d composing=%t", err, h.calls, h.recovery.CompositionActive())
	}
	if err := h.recovery.RecoverDraft(); err != nil || h.calls != 2 || len(h.reports) != 0 {
		t.Fatalf("unchanged plain Type successor did not settle: err=%v calls=%d reports=%v", err, h.calls, h.reports)
	}
	graphRecoverySourceUnchanged(t, h.buffer, h.changed)
	graphRecoveryQuiet(t, h)
}
