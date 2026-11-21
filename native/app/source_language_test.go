package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
)

func TestLanguageCommandPaletteRunsRealAnalyzerAndRendersStatus(t *testing.T) {
	a, tab, queue, _ := languageFixture(t, ".go", languageGoSource)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	before := languageState(tab)
	paletteOpen(t, a, tester)
	tester.Type("语言 分析")
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	task := a.sourceLanguage.task
	if task == nil || a.sourceLanguage.pending != nil || !a.sourceLanguage.visible || a.commandPalette.open {
		t.Fatalf("language command did not dispatch one real analysis request: palette=%t visible=%t task=%v", a.commandPalette.open, a.sourceLanguage.visible, task)
	}
	languageWait(t, task.done)
	languageReceive(t, queue.calls)()
	tester.Frame()
	if a.sourceLanguage.status != string(code.ServiceComplete) || a.sourceLanguage.result.Status != code.ServiceComplete ||
		a.sourceLanguage.intent.operation != code.ServiceDiagnostics || queue.maximum.Load() != 1 || queue.outstanding.Load() != 0 {
		t.Fatalf("real Analyzer outcome not published: status=%q result=%+v queue=%d/%d", a.sourceLanguage.status, a.sourceLanguage.result, queue.maximum.Load(), queue.outstanding.Load())
	}
	for _, label := range []string{"语言服务结果", "语言服务状态", "语法诊断", "关闭语言结果", "取消语言分析", "重新分析语言结果"} {
		if _, ok := tester.Find(label); !ok {
			t.Fatalf("accessible language result control missing: %q; texts=%v", label, tester.Texts())
		}
	}
	languageUnchanged(t, tab, before)
}

func TestLanguageIntentUTF16CaretResolutionPhysicalSource(t *testing.T) {
	const source = "🙂\r\n变量 + café"
	cases := []struct {
		name      string
		offset    int
		selection bool
		want      int
	}{
		{name: "right-unicode-identifier", offset: ui.UTF16Len("🙂\r\n"), want: len("🙂\r\n")},
		{name: "identifier-end-uses-previous-rune", offset: ui.UTF16Len("🙂\r\n变量"), want: len("🙂\r\n") + len("变")},
		{name: "selected-range-keeps-exact-start", offset: ui.UTF16Len("🙂\r\n变"), selection: true, want: len("🙂\r\n") + len("变")},
		{name: "physical-eof", offset: ui.UTF16Len(source), want: len(source) - len("é")},
		{name: "exact-selected-eof", offset: ui.UTF16Len(source), selection: true, want: len(source)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sourceLanguageByteOffset(source, tc.offset, tc.selection)
			if err != nil || got != tc.want {
				t.Fatalf("UTF-16 offset %d resolved to byte %d (%v), want %d", tc.offset, got, err, tc.want)
			}
		})
	}
	if _, err := sourceLanguageByteOffset(source, 1, false); err == nil {
		t.Fatal("UTF-16 offset inside an emoji surrogate pair was accepted")
	}
	const goSource = "// 🙂\r\npackage p\r\nvar café = 1\nvar x = café"
	byteOffset, err := sourceLanguageByteOffset(goSource, text.UTF16Len(goSource), false)
	if err != nil || byteOffset != len(goSource)-len("é") {
		t.Fatalf("identifier EOF did not select its preceding complete rune: offset=%d err=%v", byteOffset, err)
	}
	result := code.AnalyzeService(context.Background(), code.ServiceRequest{Source: goSource, Profile: code.StructureGo,
		Operation: code.ServiceDefinition, ByteOffset: byteOffset, Limits: code.DefaultServiceLimits()})
	if result.Status != code.ServiceComplete || result.Definition == nil || result.Definition.Name != "café" ||
		result.Definition.Span.Bytes.Start != strings.Index(goSource, "café") {
		t.Fatalf("EOF coordinate did not reach the actual preceding identifier binding: %+v", result)
	}
}

func TestLanguageValidatorNestingLimitIsTruthfulInHost(t *testing.T) {
	const prefix = "package p\nfunc f() {\n f()\n}\nvar x = "
	source := prefix + strings.Repeat("!", 100_005) + "true"
	a, tab, queue, _ := languageFixture(t, ".go", source)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	before := languageState(tab)
	if err := a.openSourceLanguagePresentation(); err != nil {
		t.Fatal(err)
	}
	languageComplete(t, a, queue, languageIntent(code.ServiceDiagnostics))
	tester.Frame()
	result := a.sourceLanguage.result
	if result.Status != code.ServiceLimit || result.Reason != "go-validator-nesting" || result.Resource != "" || result.Limit != 0 {
		t.Fatalf("real validator limit shape changed: %+v", result)
	}
	texts := strings.Join(tester.Texts(), "\n")
	if !strings.Contains(texts, "Go 语法校验嵌套过深") || !strings.Contains(texts, "数值阈值未提供") || strings.Contains(texts, "上限 ·  · 0") {
		t.Fatalf("validator exception was not rendered with its actual reason and missing threshold: %s", texts)
	}
	languageUnchanged(t, tab, before)
}

func TestLanguageOpenAndActionRefuseRetainedNativeInputWithoutSettling(t *testing.T) {
	const source = "package p\nfunc f() {}\n"
	a, tab, queue, _ := languageFixture(t, ".go", source)
	buffer := text.NewWithLimit(source, len(source))
	tab.Doc.Buffer = buffer
	tab.Editor.SetBuffer(buffer)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	if err := tester.Click(tab.Editor.Label); err != nil {
		t.Fatal(err)
	}
	tester.Compose("🙂", 1)
	if !errors.Is(tab.Editor.InputActionError(), text.ErrSizeLimit) || tab.Editor.CompositionActive() {
		t.Fatal("fixture is not a first-refused single composition with its actual retained size error")
	}
	if _, marked := tab.Editor.MarkedRange(); marked || buffer.InGroup() {
		t.Fatal("first refusal unexpectedly created a mark or buffer group")
	}
	before := languageState(tab)
	beforePaletteGeneration, beforePaletteHandle := a.commandPalette.generation, a.commandPalette.list.Handle
	tester.Key(ui.Cmd|ui.Shift, ui.KeyP)
	if a.commandPalette.open || a.commandPalette.generation != beforePaletteGeneration || a.commandPalette.list.Handle != beforePaletteHandle || !tester.Focused(tab.Editor.Label) {
		t.Fatal("palette opening crossed retained input or changed focus/list identity")
	}
	if err := a.requestSourceLanguage(languageIntent(code.ServiceDiagnostics)); err == nil || a.sourceLanguage.task != nil || queue.outstanding.Load() != 0 {
		t.Fatal("direct language command crossed retained native input")
	}
	languageUnchanged(t, tab, before)
	if !errors.Is(tab.Editor.InputActionError(), text.ErrSizeLimit) || !tester.Focused(tab.Editor.Label) {
		t.Fatal("refusal did not preserve the retained native input owner")
	}
	tester.Key(0, ui.KeyEscape)
	if tab.Editor.InputActionError() != nil || buffer.InGroup() || !tester.Focused(tab.Editor.Label) {
		t.Fatal("original native Escape route failed after refused host actions")
	}
	if err := buffer.SetSelection(0, 1); err != nil {
		t.Fatal(err)
	}
	tester.Compose("x", 1)
	if !tab.Editor.CompositionActive() || !buffer.InGroup() {
		t.Fatal("the original native input route could not begin valid text after Escape")
	}
	tester.Type("x")
	if tab.Editor.CompositionActive() || buffer.InGroup() || !strings.Contains(buffer.String(), "xackage") {
		t.Fatal("valid native final text could not settle after admission refusal")
	}
}
