package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/egoist/mygo/ui"
	"golang.org/x/text/encoding/simplifiedchinese"
	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
)

func TestLanguageResultsControlsMinimumWindowAndInputGate(t *testing.T) {
	a, tab, queue, _ := languageFixture(t, ".go", languageGoSource)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	languageComplete(t, a, queue, languageIntent(code.ServiceSymbols))
	if err := a.openSourceLanguagePresentation(); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	for _, label := range []string{"语言服务结果", "语言服务状态", "关闭语言结果", "重新分析语言结果", "取消语言分析", "转到语言结果"} {
		info, ok := tester.Access(label)
		if !ok || info.Bounds.W <= 0 || info.Bounds.H <= 0 || info.Bounds.X < 0 || info.Bounds.Y < 0 || info.Bounds.X+info.Bounds.W > 760 || info.Bounds.Y+info.Bounds.H > 480 {
			t.Fatalf("minimum-window result control %q absent/clipped: %#v", label, info)
		}
	}
	if len(a.sourceLanguageRows()) == 0 {
		t.Fatal("real symbols result has no actionable rows")
	}
	owner, request, presentation, handle := a.sourceLanguage.owner, a.sourceLanguage.request, a.sourceLanguage.presentation, a.sourceLanguage.list.Handle
	if err := tester.Click(tab.Editor.Label); err != nil {
		t.Fatal(err)
	}
	a.queueSourceLanguageAction("close", "") // captures a legitimate idle action
	if a.sourceLanguage.action == nil {
		t.Fatal("result control did not capture its idle action")
	}
	buffer := tab.Doc.Buffer
	tester.Compose("候选", 2)
	if !tab.Editor.CompositionActive() || !buffer.InGroup() {
		t.Fatal("fixture lacks a real active native source composition")
	}
	a.processSourceLanguageActions()
	tester.Frame()
	closeButton, ok := tester.Access("关闭语言结果")
	if !ok || !closeButton.Disabled || closeButton.Focusable {
		t.Fatalf("close control must be inaccessible while native input is active: %+v", closeButton)
	}
	if a.sourceLanguage.action != nil || !a.sourceLanguage.visible || a.sourceLanguage.presentation != presentation ||
		a.sourceLanguage.request != request || a.sourceLanguage.owner != owner || a.sourceLanguage.list.Handle != handle ||
		!buffer.InGroup() || !tab.Editor.CompositionActive() || a.sourceLanguage.task != nil {
		t.Fatal("deferred close action stole input or retired the current result/list")
	}
	if err := tab.Editor.InputActionError(); !errors.Is(err, text.ErrComposition) {
		t.Fatalf("result control admission did not preserve its owner error: %v", err)
	}
	tester.Key(0, ui.KeyEscape)
	if tab.Editor.CompositionActive() || buffer.InGroup() {
		t.Fatal("original native Escape could not recover after refused result action")
	}

	t.Run("stale-attachment-retires-hidden-result-list", func(t *testing.T) {
		a, tab, queue, _ := languageFixture(t, ".go", languageGoSource)
		tester := ui.NewTester(a.View, 760, 480)
		tester.Frame()
		languageComplete(t, a, queue, languageIntent(code.ServiceSymbols))
		if err := a.openSourceLanguagePresentation(); err != nil {
			t.Fatal(err)
		}
		tester.Frame()
		handle := a.sourceLanguage.list.Handle
		if !a.sourceLanguage.visible || a.sourceLanguage.owner == nil || len(a.sourceLanguageRows()) == 0 {
			t.Fatal("real results/presentation missing before stale-owner transition")
		}
		newBuffer := text.New(languageGoSource)
		tab.Editor.SetBuffer(newBuffer)
		tab.Doc.Buffer = newBuffer
		tester.Frame()
		if a.sourceLanguage.visible || a.sourceLanguage.owner != nil || a.sourceLanguage.pending != nil || a.sourceLanguage.task != nil ||
			len(a.sourceLanguage.result.Symbols) != 0 || a.sourceLanguage.list.Handle != handle || a.sourceLanguage.list.Key != nil || a.sourceLanguage.list.Label != nil || a.sourceLanguage.list.Selected != nil {
			t.Fatalf("stale result/list lifecycle was not retired: state=%+v list=%+v", a.sourceLanguage, a.sourceLanguage.list)
		}
	})
}

func TestLanguagePalettePendingClosePreservesForeignGroups(t *testing.T) {
	for _, composition := range []bool{false, true} {
		name := "ordinary-group"
		if composition {
			name = "composition-group"
		}
		t.Run(name, func(t *testing.T) {
			a, tab, queue, _ := languageFixture(t, ".go", languageGoSource)
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			paletteOpen(t, a, tester)
			buffer := tab.Doc.Buffer
			var token text.CompositionToken
			var err error
			if composition {
				token, err = buffer.BeginCompositionGroup()
			} else {
				err = buffer.BeginGroup()
			}
			if err != nil {
				t.Fatal(err)
			}
			before := languageState(tab)
			a.commandPalette.pendingClose = true
			a.processCommandPalette()
			if !a.commandPalette.open || a.commandPalette.pendingClose || !tester.Focused(commandPaletteQueryLabel) || !buffer.InGroup() ||
				a.sourceLanguage.task != nil || queue.outstanding.Load() != 0 {
				t.Fatal("palette close settled or stole a foreign edit group")
			}
			if got := languageState(tab); got.source != before.source || got.version != before.version || got.history != before.history || !reflect.DeepEqual(got.selections, before.selections) {
				t.Fatal("refused palette close changed foreign-owner source/history/selection")
			}
			if composition {
				if err := buffer.CancelCompositionGroup(token); err != nil {
					t.Fatal(err)
				}
			} else if !buffer.CancelGroup() {
				t.Fatal("ordinary foreign group lost its own cancellation route")
			}
		})
	}
}

func TestLanguagePalettePendingCloseAndExecuteRefuseMarklessInput(t *testing.T) {
	for _, kind := range []string{"close", "execute"} {
		t.Run(kind, func(t *testing.T) {
			a, tab, queue, _ := languageFixture(t, ".go", languageGoSource)
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			paletteOpen(t, a, tester)
			p := &a.commandPalette
			generation, listHandle, selected, query := p.generation, p.list.Handle, p.selectedID, p.query
			matches := append([]int(nil), p.matches...)
			if kind == "close" {
				p.pendingClose = true
			} else {
				p.pendingID = "language.analyze"
			}
			buffer := text.NewWithLimit("a", 1)
			tab.Doc.Buffer = buffer
			tab.Editor.SetBuffer(buffer)
			if err := buffer.SetSelection(0, 1); err != nil {
				t.Fatal(err)
			}
			tab.Editor.SetMarkedText(nil, "🙂", ui.TextInputRange{})
			if !errors.Is(tab.Editor.InputActionError(), text.ErrSizeLimit) || tab.Editor.CompositionActive() || buffer.InGroup() {
				t.Fatal("fixture lacks a retained single refusal with no mark/group")
			}
			before := languageState(tab)
			a.processCommandPalette()
			if !p.open || p.generation != generation || p.list.Handle != listHandle || p.selectedID != selected || p.query != query ||
				!reflect.DeepEqual(p.matches, matches) || !tester.Focused(commandPaletteQueryLabel) {
				t.Fatal("pending close/execute changed palette, focus, or list identity on refusal")
			}
			if p.pendingClose || p.pendingID != "" {
				t.Fatal("refused action remained queued to fire after input settles")
			}
			if a.sourceLanguage.task != nil || queue.outstanding.Load() != 0 {
				t.Fatal("refused pending language command started analysis")
			}
			if !errors.Is(tab.Editor.InputActionError(), text.ErrSizeLimit) || buffer.InGroup() {
				t.Fatal("palette refusal settled or detached the native input owner")
			}
			if got := languageState(tab); got.source != before.source || got.version != before.version || got.history != before.history ||
				!reflect.DeepEqual(got.selections, before.selections) || got.canRedo != before.canRedo || got.group != before.group {
				t.Fatal("refused action changed source/selection/history/redo/group")
			}
			tab.Editor.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape})
			if tab.Editor.InputActionError() != nil || buffer.InGroup() || !p.open {
				t.Fatal("original Escape recovery did not work while keeping the palette open")
			}
		})
	}
}

func TestLanguageReadonlyLargeSourcePalettePassesThenReportsSourceLimit(t *testing.T) {
	const maxBytes = 2 << 20
	source := "package p\n" + strings.Repeat("x", maxBytes+1)
	a, tab, queue, _ := languageFixture(t, ".go", languageGoSource)
	large := text.New(source)
	tab.Doc.Buffer = large
	tab.Editor.SetBuffer(large)
	tab.Doc.ReadOnly = true
	tab.Editor.SetReadOnly(true)
	if large.LenBytes() <= maxBytes || tab.sourceLanguageProfile != code.StructureGo || tab.ByteInspector {
		t.Fatal("fixture must be a supported readonly source above only the language request budget")
	}
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	if err := tab.Editor.InputActionError(); err != nil {
		t.Fatalf("idle readonly/large input should pass input-only admission: %v", err)
	}
	paletteOpen(t, a, tester)
	tester.Type("语言 分析")
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	if a.commandPalette.open || a.sourceLanguage.task != nil || queue.outstanding.Load() != 0 || a.sourceLanguage.result.Status != code.ServiceLimit ||
		a.sourceLanguage.result.Resource != code.StructureBytes || a.sourceLanguage.result.Limit != maxBytes {
		t.Fatalf("source limit refusal not distinct/truthful: palette=%t task=%v outstanding=%d result=%+v", a.commandPalette.open, a.sourceLanguage.task, queue.outstanding.Load(), a.sourceLanguage.result)
	}
	if !a.sourceLanguage.visible || a.sourceLanguage.status != string(code.ServiceLimit) || !tester.HasText("已达分析上限 · bytes · 2097152") {
		t.Fatalf("actual source-byte limit not shown after palette opened: %v", tester.Texts())
	}
}

func TestLanguagePaletteSixReadCommandsReachRealAnalyzer(t *testing.T) {
	cases := []struct {
		query, command string
		operation      code.ServiceOperation
		profile        string
		resultCheck    func(*testing.T, code.ServiceResult)
	}{
		{"语言 分析", "language.analyze", code.ServiceDiagnostics, ".go", func(t *testing.T, result code.ServiceResult) {
			if result.Status != code.ServiceComplete {
				t.Fatalf("diagnostics did not report complete analysis: %+v", result)
			}
		}},
		{"语言 符号", "language.symbols", code.ServiceSymbols, ".go", func(t *testing.T, result code.ServiceResult) {
			if result.Status != code.ServiceComplete || len(result.Symbols) == 0 {
				t.Fatalf("symbols result missing: %+v", result)
			}
		}},
		{"语言 定义", "language.definition", code.ServiceDefinition, ".go", func(t *testing.T, result code.ServiceResult) {
			if result.Status != code.ServiceComplete || result.Definition == nil || result.Definition.Name != "Solve" {
				t.Fatalf("definition result missing: %+v", result)
			}
		}},
		{"语言 引用", "language.references", code.ServiceReferences, ".go", func(t *testing.T, result code.ServiceResult) {
			if result.Status != code.ServiceComplete || len(result.References) == 0 {
				t.Fatalf("references result missing: %+v", result)
			}
		}},
		{"语言 信息", "language.info", code.ServiceInfo, ".go", func(t *testing.T, result code.ServiceResult) {
			if result.Status != code.ServiceComplete || result.Info == nil || result.Info.Symbol.Name != "Solve" {
				t.Fatalf("symbol info result missing: %+v", result)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			a, tab, queue, _ := languageFixture(t, tc.profile, languageGoSource)
			if tc.operation == code.ServiceDefinition || tc.operation == code.ServiceReferences || tc.operation == code.ServiceInfo {
				byteAt := strings.Index(languageGoSource, "Solve")
				if tc.operation == code.ServiceReferences {
					byteAt = strings.LastIndex(languageGoSource, "变量")
				}
				if err := tab.Doc.Buffer.SetSelection(text.UTF16Len(languageGoSource[:byteAt]), text.UTF16Len(languageGoSource[:byteAt])); err != nil {
					t.Fatal(err)
				}
			}
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			paletteOpen(t, a, tester)
			tester.Type(tc.query)
			if a.commandPalette.selectedID != tc.command {
				t.Fatalf("query selected %q, want %q", a.commandPalette.selectedID, tc.command)
			}
			tester.Key(0, ui.KeyEnter)
			tester.Frame()
			task := a.sourceLanguage.task
			if task == nil || a.commandPalette.open || a.sourceLanguage.intent.operation != tc.operation {
				t.Fatalf("palette command did not execute real B1 operation: task=%v open=%t intent=%q", task, a.commandPalette.open, a.sourceLanguage.intent.operation)
			}
			languageWait(t, task.done)
			languageReceive(t, queue.calls)()
			tester.Frame()
			tc.resultCheck(t, a.sourceLanguage.result)
			if queue.maximum.Load() != 1 || queue.outstanding.Load() != 0 {
				t.Fatalf("unexpected physical worker/notification count: %d/%d", queue.maximum.Load(), queue.outstanding.Load())
			}
		})
	}

	a, _, _, _ := languageFixture(t, ".json", `{"x":1}`)
	pal := ui.NewTester(a.View, 760, 480)
	pal.Frame()
	paletteOpen(t, a, pal)
	pal.Type("语言 定义")
	access, ok := pal.AccessByRole(ui.RoleButton, "执行命令：语言：转到本文件定义")
	if !ok || !access.Disabled {
		t.Fatalf("Go-only semantic command was not disabled for strict JSON: %+v", access)
	}
}

func TestLanguagePendingSemanticIntentKeepsAdmittedUTF16CoordinateAndOneSlot(t *testing.T) {
	a, tab, queue, _ := languageFixture(t, ".go", languageGoSource)
	caretByte := strings.LastIndex(languageGoSource, "变量")
	caret16 := text.UTF16Len(languageGoSource[:caretByte])
	if err := tab.Doc.Buffer.SetSelection(caret16, caret16); err != nil {
		t.Fatal(err)
	}
	before := languageState(tab)
	started, releaseFirstTask := languageHold(t, a)
	secondStarted := make(chan uint64, 1)
	releaseSecond := make(chan struct{})
	var secondReleaseOnce sync.Once
	releaseSecondTask := func() { secondReleaseOnce.Do(func() { close(releaseSecond) }) }
	defer releaseSecondTask()
	if err := a.requestSourceLanguage(languageIntent(code.ServiceSymbols)); err != nil {
		t.Fatal(err)
	}
	first := a.sourceLanguage.task
	if first == nil {
		t.Fatal("initial physical task missing")
	}
	languageReceive(t, started)
	originalTarget := a.sourceLanguage.owner.target
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	paletteOpen(t, a, tester)
	tester.Type("语言 引用")
	if a.commandPalette.selectedID != "language.references" {
		t.Fatalf("pending command identity = %q", a.commandPalette.selectedID)
	}
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	pending := a.sourceLanguage.pending
	if pending == nil || pending.owner == nil || pending.owner.request != a.sourceLanguage.request || pending.intent.operation != code.ServiceReferences ||
		pending.intent.utf16Offset != caret16 || pending.intent.byteOffset != 0 || !reflect.DeepEqual(pending.owner.target, originalTarget) || a.sourceLanguage.task != first ||
		queue.outstanding.Load() != 0 {
		releaseFirstTask()
		t.Fatalf("pending semantic intent materialized a target/source or escaped one-slot bound: %+v queue=%d", pending, queue.outstanding.Load())
	}
	if pending.intent.utf16Offset != text.UTF16Len(languageGoSource[:caretByte]) {
		t.Fatal("the admitted caret was not retained as its original bounded UTF-16 scalar")
	}
	if a.sourceLanguageOwnerValid(pending.owner, false) != nil || tab.Doc.Buffer.String() != before.source || tab.Doc.Buffer.Version() != before.version || tab.Doc.Buffer.History() != before.history ||
		!reflect.DeepEqual(tab.Doc.Buffer.SelectionSet(), before.selections) || tab.Doc.Buffer.CanRedo() != before.canRedo {
		releaseFirstTask()
		releaseSecondTask()
		t.Fatal("pending request changed the original source owner/document/history")
	}
	a.sourceLanguageHooks.beforeAnalyze = func(_ context.Context, id uint64) {
		secondStarted <- id
		<-releaseSecond
	}
	releaseFirstTask()
	languageWait(t, first.done)
	if queue.outstanding.Load() != 1 {
		t.Fatalf("first worker must queue a single retirement notice: %d", queue.outstanding.Load())
	}
	languageReceive(t, queue.calls)()
	second := a.sourceLanguage.task
	if second == nil || second.id != pending.owner.request || a.sourceLanguage.pending != nil {
		t.Fatalf("latest pending semantic intent did not run after physical slot retirement: task=%v pending=%v", second, a.sourceLanguage.pending)
	}
	if languageReceive(t, secondStarted) != second.id {
		t.Fatal("physical pending invocation started with the wrong request id")
	}
	// The UI inspects its admitted descriptor; the worker hook carries only ID.
	resolvedIntent := a.sourceLanguage.intent
	byteOffset := resolvedIntent.byteOffset
	if byteOffset != caretByte || byteOffset < 0 || byteOffset >= len(languageGoSource) || languageGoSource[byteOffset:byteOffset+len("变量")] != "变量" {
		releaseSecondTask()
		t.Fatalf("pending UTF-16 caret resolved to wrong real semantic byte offset: %d want %d", byteOffset, caretByte)
	}
	releaseSecondTask()
	languageWait(t, second.done)
	if queue.outstanding.Load() != 1 {
		t.Fatalf("second real Analyzer must queue exactly one completion notice: %d", queue.outstanding.Load())
	}
	languageReceive(t, queue.calls)()
	if a.sourceLanguage.status != string(code.ServiceComplete) || len(a.sourceLanguage.result.References) != 1 || queue.maximum.Load() != 1 || queue.outstanding.Load() != 0 {
		t.Fatalf("latest semantic result/slot limits failed: state=%+v queue=%d/%d", a.sourceLanguage, queue.maximum.Load(), queue.outstanding.Load())
	}
	languageUnchanged(t, tab, before)
}

func TestLanguageCancelCommandCancelsRealHeldAnalyzer(t *testing.T) {
	a, _, queue, _ := languageFixture(t, ".go", languageGoSource)
	started, release := languageHold(t, a)
	defer release()
	if err := a.requestSourceLanguage(languageIntent(code.ServiceSymbols)); err != nil {
		t.Fatal(err)
	}
	task := a.sourceLanguage.task
	if task == nil {
		t.Fatal("real initial Analyzer task did not start")
	}
	languageReceive(t, started)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	paletteOpen(t, a, tester)
	tester.Type("语言 取消")
	if a.commandPalette.selectedID != "language.cancel" {
		t.Fatalf("cancel command selection = %q", a.commandPalette.selectedID)
	}
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	if a.commandPalette.open || a.sourceLanguage.task != task || a.sourceLanguage.pending != nil || a.sourceLanguage.owner != nil ||
		a.sourceLanguage.status != string(code.ServiceCanceled) || a.sourceLanguage.reason != "canceled-by-host" {
		t.Fatalf("real cancel action failed or released physical slot early: palette=%t taskSame=%t state=%+v", a.commandPalette.open, a.sourceLanguage.task == task, a.sourceLanguage)
	}
	// The real parser is still held; its one notice can only be queued after the
	// physical worker returns, regardless of the host cancellation request.
	if queue.outstanding.Load() != 0 {
		t.Fatalf("held canceled worker published before physical return: %d", queue.outstanding.Load())
	}
	release()
	languageWait(t, task.done)
	if queue.outstanding.Load() != 1 {
		t.Fatalf("real canceled worker must enqueue exactly one late notice: %d", queue.outstanding.Load())
	}
	languageReceive(t, queue.calls)()
	if a.sourceLanguage.status != string(code.ServiceCanceled) || a.sourceLanguage.task != nil || queue.outstanding.Load() != 0 {
		t.Fatalf("late real completion overwrote explicit cancellation: state=%+v queued=%d", a.sourceLanguage, queue.outstanding.Load())
	}
}

func TestLanguageResultsVirtualListIsBoundedAndReusesHandle(t *testing.T) {
	var source strings.Builder
	source.WriteString("package p\n")
	for i := 0; i < sourceLanguageDisplayRows+44; i++ {
		source.WriteString("var Symbol")
		source.WriteString(fmt.Sprintf("%03d", i))
		source.WriteString(" = 1\n")
	}
	a, tab, queue, _ := languageFixture(t, ".go", source.String())
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	if err := a.openSourceLanguagePresentation(); err != nil {
		t.Fatal(err)
	}
	languageComplete(t, a, queue, languageIntent(code.ServiceSymbols))
	tester.Frame()
	rows := a.sourceLanguageRows()
	if len(rows) != sourceLanguageDisplayRows || a.sourceLanguage.result.Status != code.ServiceComplete || len(a.sourceLanguage.result.Symbols) <= len(rows) {
		t.Fatalf("virtual list did not apply the explicit row bound: rows=%d symbols=%d result=%+v", len(rows), len(a.sourceLanguage.result.Symbols), a.sourceLanguage.result.Status)
	}
	handle := a.sourceLanguage.list.Handle
	if handle == (ui.Handle{}) {
		t.Fatal("result ListState did not bind a public list handle")
	}
	visible := 0
	for _, row := range rows {
		if _, ok := tester.Find("语言结果：" + row.label); ok {
			visible++
		}
	}
	if visible == 0 || visible >= len(rows) {
		t.Fatalf("result list was not virtualized; visible=%d bounded=%d", visible, len(rows))
	}
	a.closeSourceLanguagePresentation()
	if a.sourceLanguage.list.Handle != handle {
		t.Fatal("closing language presentation replaced the list's public Handle")
	}
	tester.Frame()
	if a.sourceLanguage.list.Key != nil || a.sourceLanguage.list.Label != nil || a.sourceLanguage.list.Selected != nil {
		t.Fatal("safe frame boundary did not drain old list callbacks")
	}
	if err := a.openSourceLanguagePresentation(); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if a.sourceLanguage.list.Handle != handle || a.sourceLanguage.selected != 0 {
		t.Fatalf("reopened presentation did not reuse the public Handle and select its first real row: oldHandle=%v newHandle=%v selected=%d", handle, a.sourceLanguage.list.Handle, a.sourceLanguage.selected)
	}
	a.sourceLanguage.list.Handle.Focus()
	tester.Frame()
	tester.Key(0, ui.KeyEnd)
	tester.Frame()
	if a.sourceLanguage.selected != sourceLanguageDisplayRows-1 {
		t.Fatalf("public list keyboard End did not select its last bounded row: %d", a.sourceLanguage.selected)
	}
	last := a.sourceLanguageRows()[sourceLanguageDisplayRows-1]
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	if a.sourceLanguage.message == "" || tab.Doc.Buffer.Selection().Anchor != last.start || tab.Doc.Buffer.Selection().Head != last.end || tab.Doc.Buffer.String() != source.String() {
		t.Fatalf("public list Enter did not reveal the selected real virtual row: selection=%+v row=%+v message=%q", tab.Doc.Buffer.Selection(), last, a.sourceLanguage.message)
	}
}

func TestLanguageResultsInvalidAndStaleStatesAreDistinct(t *testing.T) {
	t.Run("invalid-source-diagnostics", func(t *testing.T) {
		a, _, queue, _ := languageFixture(t, ".go", "package p\nfunc broken(\n")
		tester := ui.NewTester(a.View, 760, 480)
		tester.Frame()
		if err := a.openSourceLanguagePresentation(); err != nil {
			t.Fatal(err)
		}
		languageComplete(t, a, queue, languageIntent(code.ServiceDiagnostics))
		tester.Frame()
		if a.sourceLanguage.result.Status != code.ServiceInvalid || len(a.sourceLanguage.result.Diagnostics) == 0 || !tester.HasText("源码无效") || !tester.HasText("语法诊断") {
			t.Fatalf("invalid source did not display actual diagnostics/status: %+v / %v", a.sourceLanguage.result, tester.Texts())
		}
	})
	t.Run("unsupported-operation", func(t *testing.T) {
		a, _, queue, _ := languageFixture(t, ".json", `{"x":1}`)
		tester := ui.NewTester(a.View, 760, 480)
		tester.Frame()
		if err := a.openSourceLanguagePresentation(); err != nil {
			t.Fatal(err)
		}
		intent := languageIntent(code.ServiceDefinition)
		if err := a.requestSourceLanguage(intent); err != nil {
			t.Fatal(err)
		}
		task := a.sourceLanguage.task
		languageWait(t, task.done)
		languageReceive(t, queue.calls)()
		tester.Frame()
		if a.sourceLanguage.result.Status != code.ServiceUnsupported || a.sourceLanguage.result.Reason != "operation-for-profile" || !tester.HasText("不支持") {
			t.Fatalf("unsupported result not distinct in host: %+v / %v", a.sourceLanguage.result, tester.Texts())
		}
	})
	t.Run("stale-after-owner-change", func(t *testing.T) {
		a, tab, queue, _ := languageFixture(t, ".go", languageGoSource)
		tester := ui.NewTester(a.View, 760, 480)
		tester.Frame()
		started, release := languageHold(t, a)
		defer release()
		if err := a.openSourceLanguagePresentation(); err != nil {
			t.Fatal(err)
		}
		if err := a.requestSourceLanguage(languageIntent(code.ServiceSymbols)); err != nil {
			t.Fatal(err)
		}
		task := a.sourceLanguage.task
		languageReceive(t, started)
		newBuffer := text.New(languageGoSource)
		tab.Editor.SetBuffer(newBuffer)
		tab.Doc.Buffer = newBuffer
		release()
		languageWait(t, task.done)
		languageReceive(t, queue.calls)()
		tester.Frame()
		if a.sourceLanguage.status != sourceLanguageStale || a.sourceLanguage.owner != nil || len(a.sourceLanguage.result.Symbols) != 0 ||
			!strings.Contains(a.Message, "语言结果已过期") {
			t.Fatalf("stale owner was presented as live/successful or its retirement was silent: state=%+v message=%q texts=%v", a.sourceLanguage, a.Message, tester.Texts())
		}
	})
}

func TestLanguageSavedRowActionCannotNavigateAfterNewRequest(t *testing.T) {
	a, tab, queue, _ := languageFixture(t, ".go", languageGoSource)
	if err := tab.Doc.Buffer.SetSelection(text.UTF16Len(languageGoSource[:strings.Index(languageGoSource, "Solve")]), text.UTF16Len(languageGoSource[:strings.Index(languageGoSource, "Solve")])); err != nil {
		t.Fatal(err)
	}
	if err := a.openSourceLanguagePresentation(); err != nil {
		t.Fatal(err)
	}
	intent := languageIntent(code.ServiceDefinition)
	intent.byteOffset = strings.Index(languageGoSource, "Solve")
	languageComplete(t, a, queue, intent)
	rows := a.sourceLanguageRows()
	if len(rows) != 1 {
		t.Fatalf("initial definition row missing: %+v", rows)
	}
	oldRequest, oldPresentation := a.sourceLanguage.request, a.sourceLanguage.presentation
	oldRow := rows[0]
	a.queueSourceLanguageAction("navigate", oldRow.id)
	if a.sourceLanguage.action == nil {
		t.Fatal("real current row action was not captured")
	}
	if err := tab.Doc.Buffer.SetSelection(text.UTF16Len(languageGoSource[:strings.LastIndex(languageGoSource, "变量")]), text.UTF16Len(languageGoSource[:strings.LastIndex(languageGoSource, "变量")])); err != nil {
		t.Fatal(err)
	}
	newTask := a.requestSourceLanguage(languageIntent(code.ServiceSymbols))
	if newTask != nil {
		t.Fatalf("new request failed unexpectedly: %v", newTask)
	}
	if a.sourceLanguage.request == oldRequest || a.sourceLanguage.presentation != oldPresentation {
		t.Fatal("fresh request did not advance only request ownership")
	}
	beforeSelection := tab.Doc.Buffer.SelectionSet()
	a.processSourceLanguageActions()
	if a.sourceLanguage.action != nil || a.sourceLanguage.request == oldRequest || a.sourceLanguage.owner == nil ||
		!reflect.DeepEqual(tab.Doc.Buffer.SelectionSet(), beforeSelection) {
		t.Fatal("saved row callback navigated after a newer real request replaced its result")
	}
	task := a.sourceLanguage.task
	languageWait(t, task.done)
	languageReceive(t, queue.calls)()
	if a.sourceLanguage.status != string(code.ServiceComplete) || len(a.sourceLanguage.result.Symbols) == 0 {
		t.Fatalf("fresh successor result was damaged by stale action: %+v", a.sourceLanguage)
	}
}

func TestLanguageGBKReadAnalysisAndExplicitSavePreserveExactBytes(t *testing.T) {
	const original = "package p\r\n// 注释\r\nfunc Solve(变量 int) int {\n return 变量\r\n}\n"
	encoded, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(original))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	path := filepath.Join(root, "legacy.go")
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	inspector := a.Current()
	if inspector == nil || !inspector.ByteInspector || !a.reopenWithEncoding(inspector, document.GBK) {
		t.Fatalf("explicit strict-roundtrip GBK open failed: %v", a.Message)
	}
	tab := a.Current()
	if tab == nil || tab.Doc.Encoding != document.GBK || tab.Doc.BOM || tab.Doc.Buffer.String() != original || !languageGoCommandAvailable(a) {
		t.Fatalf("GBK Go source route differs after explicit decoding: encoding=%s BOM=%t text=%q", tab.Doc.Encoding, tab.Doc.BOM, tab.Doc.Buffer.String())
	}
	queue := newLanguageQueue()
	a.sourceLanguageHooks = &sourceLanguageHooks{dispatch: queue.dispatch}
	if err := tab.Doc.Buffer.ReplaceUTF16(tab.Doc.Buffer.LenUTF16()-1, tab.Doc.Buffer.LenUTF16()-1, " "); err != nil {
		t.Fatal(err)
	}
	before := tab.Doc.Buffer.String()
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	if err := a.openSourceLanguagePresentation(); err != nil {
		t.Fatal(err)
	}
	languageComplete(t, a, queue, languageIntent(code.ServiceDiagnostics))
	if a.sourceLanguage.result.Status != code.ServiceComplete || tab.Doc.Buffer.String() != before {
		t.Fatalf("GBK language read altered or rejected source: %+v", a.sourceLanguage.result)
	}
	if !bytes.Equal(mustReadFile(t, path), encoded) {
		t.Fatal("language analysis changed original GBK disk bytes before the user's save")
	}
	if !a.Save(tab, false) {
		t.Fatalf("explicit user save failed after GBK analysis: %s", a.Message)
	}
	expected, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(before))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(mustReadFile(t, path), expected) || tab.Doc.Encoding != document.GBK || tab.Doc.BOM || tab.Doc.Dirty() {
		t.Fatal("explicit later GBK save did not preserve exact encoding/EOL/content")
	}
}

func TestLanguageResultsRevealPreservesCompleteDirectedSelectionOnFailedAction(t *testing.T) {
	a, tab, queue, _ := languageFixture(t, ".go", languageGoSource)
	if err := tab.Doc.Buffer.SetSelectionSet(text.SelectionSet{Ranges: []text.Range{{Anchor: 8, Head: 2}, {Anchor: 12, Head: 12}}, Primary: 0}); err != nil {
		t.Fatal(err)
	}
	before := languageState(tab)
	if err := a.openSourceLanguagePresentation(); err != nil {
		t.Fatal(err)
	}
	languageComplete(t, a, queue, languageIntent(code.ServiceSymbols))
	bad := code.StructureSpan{UTF16: code.StructureUTF16Range{Start: -1, End: -1}}
	row := sourceLanguageRow{id: sourceLanguageRowID("negative", 0, -1, -1), label: "invalid", kind: "symbol", start: bad.UTF16.Start, end: bad.UTF16.End}
	a.sourceLanguage.rows = []sourceLanguageRow{row}
	a.sourceLanguage.rowsReady = true
	a.sourceLanguage.rowRequest, a.sourceLanguage.rowPresentation = a.sourceLanguage.request, a.sourceLanguage.presentation
	a.queueSourceLanguageAction("navigate", row.id)
	a.processSourceLanguageActions()
	if a.sourceLanguage.message == "" || tab.Doc.Buffer.String() != before.source || tab.Doc.Buffer.Version() != before.version || tab.Doc.Buffer.History() != before.history ||
		!reflect.DeepEqual(tab.Doc.Buffer.SelectionSet(), before.selections) {
		t.Fatalf("invalid target did not preserve full directed selection/source on refusal: message=%q selection=%+v", a.sourceLanguage.message, tab.Doc.Buffer.SelectionSet())
	}
}

func TestLanguageCommandOpenPreflightDoesNotFlushForReadonlyIdleLargeSource(t *testing.T) {
	const maxBytes = 2 << 20
	source := "package p\n" + strings.Repeat("x\n", maxBytes/2+1)
	a, tab, queue, _ := languageFixture(t, ".go", languageGoSource)
	buffer := text.New(source)
	tab.Doc.Buffer = buffer
	tab.Editor.SetBuffer(buffer)
	tab.Doc.ReadOnly, tab.Editor.ReadOnly = true, true
	checks := 0
	original := tab.Editor.OnError
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame() // let normal source layout emit any one-time line-budget warning first
	tab.Editor.OnError = func(error) { checks++ }
	defer func() { tab.Editor.OnError = original }()
	if err := a.openSourceLanguagePresentation(); err != nil || !a.sourceLanguage.visible || checks != 0 {
		t.Fatalf("idle readonly input gate blocked palette/result open or invoked callback: visible=%t checks=%d err=%v", a.sourceLanguage.visible, checks, err)
	}
	if !a.openCommandPalette() || !a.commandPalette.open || queue.outstanding.Load() != 0 {
		t.Fatal("idle readonly >2MiB source did not pass palette's input-only gate")
	}
	a.closeCommandPalette()
	a.runSourceLanguageCommand("analyze")
	if a.sourceLanguage.status != string(code.ServiceLimit) || a.sourceLanguage.result.Resource != code.StructureBytes || a.sourceLanguage.result.Limit != maxBytes || checks != 0 {
		t.Fatalf("operation must pass neutral input admission then report the actual byte budget: status=%s result=%+v callbacks=%d", a.sourceLanguage.status, a.sourceLanguage.result, checks)
	}
	languageCheckNonMutating(t, tab, buffer, source, checks)
}

func languageCheckNonMutating(t *testing.T, tab *Tab, b *text.Buffer, source string, errorCallbacks int) {
	t.Helper()
	if b.String() != source || b.Version() != 0 || b.InGroup() || b.History() != (text.HistoryStatus{}) || errorCallbacks != 0 {
		t.Fatal("readonly input-only host preflight changed source, transaction state, or invoked callbacks")
	}
}

const task98Source = "package p\nvar x = 1\n"

// All successful payloads below come from the real App/B1/Analyzer chain.
func task98CompletedDisplay(t *testing.T, operation code.ServiceOperation, limited bool) (*App, *Tab, *languageQueue, *ui.Tester) {
	t.Helper()
	a, tab, queue, _ := languageFixture(t, ".go", task98Source)
	if limited {
		buffer := text.NewWithLimit(task98Source, len(task98Source))
		tab.Doc.Buffer = buffer
		tab.Editor.SetBuffer(buffer)
	}
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	if err := a.openSourceLanguagePresentation(); err != nil {
		t.Fatal(err)
	}
	languageComplete(t, a, queue, languageIntent(operation))
	tester.Frame() // publish/cache the completed result before the mutation
	if a.sourceLanguage.status != string(code.ServiceComplete) || a.sourceLanguage.result.Status != code.ServiceComplete ||
		a.sourceLanguage.owner == nil || !a.sourceLanguage.rowsReady || a.sourceLanguage.task != nil || queue.outstanding.Load() != 0 {
		t.Fatalf("real completed/cached display fixture failed: %+v", a.sourceLanguage)
	}
	if operation == code.ServiceSymbols && len(a.sourceLanguageRows()) != 1 {
		t.Fatalf("expected real x symbol, got %+v", a.sourceLanguageRows())
	}
	return a, tab, queue, tester
}

func task98RequireVisibleStale(t *testing.T, a *App, queue *languageQueue, tester *ui.Tester) {
	t.Helper()
	s := &a.sourceLanguage
	if !s.visible || s.status != sourceLanguageStale || s.owner != nil || s.pending != nil || s.task != nil ||
		len(s.rows) != 0 || len(s.cachedPositions) != 0 || !reflect.DeepEqual(s.result, code.ServiceResult{}) ||
		!tester.HasText("结果已过期") || tester.HasText("未发现语法诊断") || queue.outstanding.Load() != 0 {
		t.Fatalf("stale display retained successful payloads, silently hid status, or started work: state=%+v texts=%v", s, tester.Texts())
	}
}

func TestLanguageCompletedCachedSymbolsSameBufferEditIsVisiblyStale(t *testing.T) {
	a, tab, queue, tester := task98CompletedDisplay(t, code.ServiceSymbols, false)
	oldRow := a.sourceLanguageRows()[0]
	oldAction := sourceLanguageAction{kind: "navigate", rowID: oldRow.id, request: a.sourceLanguage.request, presentation: a.sourceLanguage.presentation}
	buffer := tab.Doc.Buffer
	if err := tester.Click(tab.Editor.Label); err != nil {
		t.Fatal(err)
	}
	if err := buffer.SetSelection(0, 0); err != nil {
		t.Fatal(err)
	}
	tester.Type("(") // real installed native route; Tab/Doc/Editor/Buffer stay attached
	if tab.Doc.Buffer != buffer || buffer.String() != "("+task98Source {
		t.Fatal("fixture did not perform a same-Buffer source edit")
	}
	before := languageState(tab)
	tester.Frame()
	task98RequireVisibleStale(t, a, queue, tester)
	if _, ok := tester.Find("语言结果：" + oldRow.label); ok {
		t.Fatal("old cached symbol row is still exposed after its source changed")
	}
	a.navigateSourceLanguageRow(oldAction)
	languageUnchanged(t, tab, before)
	if !tester.Focused(tab.Editor.Label) {
		t.Fatal("stale result retirement or rejected old navigation changed native focus")
	}
}

func TestLanguageCompletedDiagnosticsEditCannotKeepZeroDiagnosticsClaim(t *testing.T) {
	a, tab, queue, tester := task98CompletedDisplay(t, code.ServiceDiagnostics, false)
	if !tester.HasText("未发现语法诊断") {
		t.Fatal("real complete diagnostics fixture did not advertise its initial result")
	}
	if err := tester.Click(tab.Editor.Label); err != nil {
		t.Fatal(err)
	}
	if err := tab.Doc.Buffer.SetSelection(0, 0); err != nil {
		t.Fatal(err)
	}
	tester.Type("(")
	before := languageState(tab)
	tester.Frame()
	task98RequireVisibleStale(t, a, queue, tester)
	languageUnchanged(t, tab, before)
	a.runSourceLanguageCommand("analyze") // a fresh explicit request, never automatic
	task := a.sourceLanguage.task
	if task == nil {
		t.Fatalf("explicit analysis against the edited source did not start: %s", a.sourceLanguage.message)
	}
	languageWait(t, task.done)
	languageReceive(t, queue.calls)()
	tester.Frame()
	if a.sourceLanguage.status != string(code.ServiceInvalid) || len(a.sourceLanguage.result.Diagnostics) == 0 ||
		!a.sourceLanguage.rowsReady || len(a.sourceLanguage.rows) == 0 || a.sourceLanguage.rows[0].kind != "diagnostic" ||
		!tester.HasText("源码无效") || tester.HasText("未发现语法诊断") || a.sourceLanguage.analysisVersion != tab.Doc.Buffer.Version() {
		t.Fatalf("fresh explicit analysis did not replace the stale zero-diagnostics result: %+v texts=%v", a.sourceLanguage, tester.Texts())
	}
	languageUnchanged(t, tab, before)
}

func TestLanguageDirectStaleRowNavigationDisclosesWithoutMutatingSource(t *testing.T) {
	a, tab, queue, tester := task98CompletedDisplay(t, code.ServiceSymbols, false)
	row := a.sourceLanguageRows()[0]
	action := sourceLanguageAction{kind: "navigate", rowID: row.id, request: a.sourceLanguage.request, presentation: a.sourceLanguage.presentation}
	if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, "("); err != nil {
		t.Fatal(err)
	}
	if err := tab.Doc.Buffer.SetSelectionSet(text.SelectionSet{Ranges: []text.Range{{Anchor: 3, Head: 1}, {Anchor: 10, Head: 12}}, Primary: 1}); err != nil {
		t.Fatal(err)
	}
	before := languageState(tab)
	// No frame has rechecked the result: direct navigation itself must disclose
	// the stale owner rather than silently returning from the SourceTarget gate.
	a.navigateSourceLanguageRow(action)
	if a.sourceLanguage.status != sourceLanguageStale || a.sourceLanguage.owner != nil || len(a.sourceLanguage.rows) != 0 ||
		!strings.Contains(a.sourceLanguage.message, "已过期") {
		t.Fatal("direct stale row navigation silently retained its old result")
	}
	languageUnchanged(t, tab, before)
	tester.Frame()
	task98RequireVisibleStale(t, a, queue, tester)
	languageUnchanged(t, tab, before)
}

func TestLanguageEditUndoDoesNotReviveCachedResultAndExplicitRequestRefreshes(t *testing.T) {
	a, tab, queue, tester := task98CompletedDisplay(t, code.ServiceSymbols, false)
	oldRequest := a.sourceLanguage.request
	oldRow := a.sourceLanguageRows()[0]
	oldAction := sourceLanguageAction{kind: "navigate", rowID: oldRow.id, request: oldRequest, presentation: a.sourceLanguage.presentation}
	buffer := tab.Doc.Buffer
	if err := buffer.ReplaceUTF16(0, 0, "("); err != nil || !buffer.Undo() || buffer.String() != task98Source {
		t.Fatal("edit/Undo fixture did not restore equal bytes with a changed original identity", err)
	}
	before := languageState(tab)
	tester.Frame()
	task98RequireVisibleStale(t, a, queue, tester)
	a.navigateSourceLanguageRow(oldAction)
	languageUnchanged(t, tab, before)
	// A later explicit command may analyze current source and publish new rows.
	a.runSourceLanguageCommand("symbols")
	newRequest, newOwner, task := a.sourceLanguage.request, a.sourceLanguage.owner, a.sourceLanguage.task
	if newRequest == oldRequest || task == nil || newOwner == nil {
		t.Fatal("explicit refresh rebound an old request instead of creating a fresh one")
	}
	languageWait(t, task.done)
	languageReceive(t, queue.calls)()
	tester.Frame()
	if a.sourceLanguage.status != string(code.ServiceComplete) || a.sourceLanguage.owner != newOwner ||
		a.sourceLanguage.analysisVersion != buffer.Version() || len(a.sourceLanguageRows()) != 1 {
		t.Fatalf("fresh request did not cache a new real result: %+v", a.sourceLanguage)
	}
	newRows := append([]sourceLanguageRow(nil), a.sourceLanguage.rows...)
	a.navigateSourceLanguageRow(oldAction)
	if a.sourceLanguage.request != newRequest || a.sourceLanguage.owner != newOwner || !reflect.DeepEqual(a.sourceLanguage.rows, newRows) {
		t.Fatal("old saved row action retired or navigated the explicit successor result")
	}
	languageUnchanged(t, tab, before)
	if err := buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), "var y = 2\n"); err != nil {
		t.Fatal(err)
	}
	edited := languageState(tab)
	tester.Frame()
	task98RequireVisibleStale(t, a, queue, tester)
	a.runSourceLanguageCommand("symbols")
	freshTask := a.sourceLanguage.task
	if freshTask == nil {
		t.Fatal("explicit symbols refresh did not start against the new text")
	}
	languageWait(t, freshTask.done)
	languageReceive(t, queue.calls)()
	tester.Frame()
	if a.sourceLanguage.status != string(code.ServiceComplete) || !a.sourceLanguage.rowsReady || len(a.sourceLanguage.rows) != 2 ||
		len(a.sourceLanguage.result.Symbols) != 2 || a.sourceLanguage.analysisVersion != buffer.Version() {
		t.Fatalf("explicit analysis did not publish new text's two real symbols: %+v", a.sourceLanguage)
	}
	foundY := false
	for _, symbol := range a.sourceLanguage.result.Symbols {
		if symbol.Name == "y" {
			foundY = true
		}
	}
	if !foundY {
		t.Fatal("new explicit symbol result contains no new y declaration")
	}
	languageUnchanged(t, tab, edited)
}

func TestLanguageCompletedDisplayForeignGroupRefusalIsTransient(t *testing.T) {
	for _, composition := range []bool{false, true} {
		name := "ordinary"
		if composition {
			name = "composition"
		}
		t.Run(name, func(t *testing.T) {
			a, tab, queue, tester := task98CompletedDisplay(t, code.ServiceSymbols, false)
			s := &a.sourceLanguage
			owner, request, presentation := s.owner, s.request, s.presentation
			rows := append([]sourceLanguageRow(nil), s.rows...)
			action := sourceLanguageAction{kind: "navigate", rowID: rows[0].id, request: request, presentation: presentation}
			buffer := tab.Doc.Buffer
			var token text.CompositionToken
			if composition {
				var err error
				token, err = buffer.BeginCompositionGroup()
				if err != nil {
					t.Fatal(err)
				}
			} else if err := buffer.BeginGroup(); err != nil {
				t.Fatal(err)
			}
			before := languageState(tab)
			for i := 0; i < 3; i++ {
				tester.Frame()
				a.navigateSourceLanguageRow(action)
				if s.owner != owner || s.request != request || s.presentation != presentation || s.status != string(code.ServiceComplete) ||
					!reflect.DeepEqual(s.rows, rows) || !buffer.InGroup() || !errors.Is(tab.Editor.InputActionError(), text.ErrGroup) {
					t.Fatal("temporary foreign group was settled or permanently classified as stale")
				}
				languageUnchanged(t, tab, before)
			}
			if composition {
				if !buffer.OwnsComposition(token) || buffer.CancelCompositionGroup(token) != nil {
					t.Fatal("foreign composition owner lost cancellation")
				}
			} else if !buffer.CancelGroup() {
				t.Fatal("foreign ordinary owner lost cancellation")
			}
			tester.Frame()
			if tab.Editor.InputActionError() != nil || s.owner != owner || s.status != string(code.ServiceComplete) || !reflect.DeepEqual(s.rows, rows) || queue.outstanding.Load() != 0 {
				t.Fatal("unchanged source stayed stale after temporary foreign ownership ended")
			}
		})
	}
}

func TestLanguagePendingNativeInputRetainsDisplayUntilOwnFinalOrEscape(t *testing.T) {
	for _, limited := range []bool{false, true} {
		name := "accepted-preedit"
		if limited {
			name = "first-refused-markless"
		}
		t.Run(name, func(t *testing.T) {
			a, tab, queue, tester := task98CompletedDisplay(t, code.ServiceSymbols, limited)
			s := &a.sourceLanguage
			owner, request, presentation := s.owner, s.request, s.presentation
			rows := append([]sourceLanguageRow(nil), s.rows...)
			action := sourceLanguageAction{kind: "navigate", rowID: rows[0].id, request: request, presentation: presentation}
			if err := tester.Click(tab.Editor.Label); err != nil {
				t.Fatal(err)
			}
			if err := tab.Doc.Buffer.SetSelection(0, 0); err != nil {
				t.Fatal(err)
			}
			if limited {
				tester.Compose("🙂", 1)
			} else {
				tester.Compose("(", 1)
			}
			before := languageState(tab)
			if limited {
				if !errors.Is(tab.Editor.InputActionError(), text.ErrSizeLimit) || tab.Editor.CompositionActive() || tab.Doc.Buffer.InGroup() {
					t.Fatal("markless refusal fixture missing")
				}
			} else if !tab.Editor.CompositionActive() || !tab.Doc.Buffer.InGroup() {
				t.Fatal("accepted native preedit fixture missing")
			}
			for i := 0; i < 3; i++ {
				tester.Frame()
				a.navigateSourceLanguageRow(action)
				if s.owner != owner || s.request != request || s.presentation != presentation || s.status != string(code.ServiceComplete) ||
					!reflect.DeepEqual(s.rows, rows) || !tester.Focused(tab.Editor.Label) || s.task != nil || queue.outstanding.Load() != 0 {
					t.Fatal("pending native owner/result/cache/focus was retired by a display probe")
				}
				languageUnchanged(t, tab, before)
			}
			if limited {
				tester.Key(0, ui.KeyEscape)
				if tab.Editor.InputActionError() != nil {
					t.Fatal("retained owner's native Escape route became unavailable")
				}
				if err := tab.Doc.Buffer.SetSelection(0, 1); err != nil {
					t.Fatal(err)
				}
				tester.Type("q") // same-size valid final input remains deliverable through the original native route
				if tab.Doc.Buffer.String() != "q"+task98Source[1:] {
					t.Fatal("native input did not remain usable after refused display actions")
				}
			} else {
				tester.Type("(") // real final text settles only its original native owner
			}
			if tab.Editor.InputActionError() != nil || tab.Doc.Buffer.InGroup() {
				t.Fatal("owned final input failed to settle")
			}
			afterFinal := languageState(tab)
			tester.Frame()
			task98RequireVisibleStale(t, a, queue, tester)
			languageUnchanged(t, tab, afterFinal)
		})
	}
}

func TestLanguageCompletedDisplaySameBufferPathChangeRetiresResult(t *testing.T) {
	a, tab, queue, tester := task98CompletedDisplay(t, code.ServiceSymbols, false)
	buffer := tab.Doc.Buffer
	tab.Doc.Path = filepath.Join(t.TempDir(), "renamed.go")
	before := languageState(tab)
	tester.Frame()
	if tab.Doc.Buffer != buffer {
		t.Fatal("path fixture changed the Buffer identity")
	}
	task98RequireVisibleStale(t, a, queue, tester)
	languageUnchanged(t, tab, before)
}

func TestLanguageStaleRevealCallbackPreservesExplicitSuccessor(t *testing.T) {
	a, tab, queue, tester := task98CompletedDisplay(t, code.ServiceSymbols, false)
	row := a.sourceLanguageRows()[0]
	action := sourceLanguageAction{kind: "navigate", rowID: row.id, request: a.sourceLanguage.request, presentation: a.sourceLanguage.presentation}
	previousHighlight := tab.Editor.Highlight
	if previousHighlight == nil {
		t.Fatal("real Go highlighter fixture missing")
	}
	calls := 0
	var successor *sourceLanguageOwner
	var successorRequest uint64
	var callbackExit languageDocumentState
	var callbackMessage string
	tab.Editor.Highlight = func(line string) []ui.Span {
		if calls == 0 {
			calls++
			if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, "("); err != nil {
				t.Fatal(err)
			}
			a.runSourceLanguageCommand("analyze")
			successor, successorRequest = a.sourceLanguage.owner, a.sourceLanguage.request
			callbackExit, callbackMessage = languageState(tab), a.sourceLanguage.message
		}
		return previousHighlight(line)
	}
	tab.Editor.SetFontSize(tab.Editor.FontSize + 1) // real callback must run during checked, cold-geometry reveal
	a.navigateSourceLanguageRow(action)
	if calls != 1 || successor == nil || a.sourceLanguage.owner != successor || a.sourceLanguage.request != successorRequest ||
		a.sourceLanguage.status != sourceLanguageRunning || a.sourceLanguage.message != callbackMessage {
		t.Fatal("old stale navigation retired or overwrote the callback's explicit successor")
	}
	languageUnchanged(t, tab, callbackExit)
	task := a.sourceLanguage.task
	if task == nil {
		t.Fatal("callback's real explicit Analyzer task disappeared")
	}
	languageWait(t, task.done)
	languageReceive(t, queue.calls)()
	tester.Frame()
	if a.sourceLanguage.status != string(code.ServiceInvalid) || len(a.sourceLanguage.result.Diagnostics) == 0 || a.sourceLanguage.owner != successor {
		t.Fatalf("callback successor could not finish against its new source: %+v", a.sourceLanguage)
	}
	languageUnchanged(t, tab, callbackExit)
}
