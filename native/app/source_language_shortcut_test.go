package app

import (
	"context"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
	"levnote.local/native/ui/editor"
)

const shortcutCompletionSource = "package p\r\n// 🙂 unchanged\r\nfunc f(val int) { _ = va }\n"

func shortcutFocus(t *testing.T, tester *ui.Tester, tab *Tab) {
	t.Helper()
	tab.Editor.Focus()
	tester.Frame()
	if !tester.Focused(tab.Editor.Label) {
		t.Fatal("actual main source input did not receive focus")
	}
}

func shortcutKey(t *testing.T, tester *ui.Tester, action editor.SourceAction) {
	t.Helper()
	switch action {
	case editor.SourceActionComplete:
		tester.Key(ui.Ctrl, ui.KeySpace)
	case editor.SourceActionReferences:
		tester.Key(ui.Shift, ui.KeyF12)
	case editor.SourceActionRename:
		tester.Key(0, ui.KeyF2)
	default:
		t.Fatal("unknown test action")
	}
	tester.Frame()
}

func shortcutAccessible(t *testing.T, tester *ui.Tester, labels ...string) {
	t.Helper()
	for _, label := range labels {
		info, ok := tester.Access(label)
		if !ok || info.Disabled || info.Bounds.W <= 0 || info.Bounds.H <= 0 || info.Bounds.X < 0 || info.Bounds.Y < 0 || info.Bounds.X+info.Bounds.W > 760 || info.Bounds.Y+info.Bounds.H > 480 {
			t.Fatalf("control %q is absent, disabled or clipped: %+v", label, info)
		}
	}
}

func TestLanguageShortcutMainSourceCompletionAndExplicitFallback(t *testing.T) {
	for _, method := range []string{"shortcut-keyboard", "shortcut-button", "explicit"} {
		t.Run(method, func(t *testing.T) {
			a, tab, q, path := languageFixture(t, ".go", shortcutCompletionSource)
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			if err := tester.Click(tab.Editor.Label); err != nil {
				t.Fatal(err)
			}
			at := strings.LastIndex(shortcutCompletionSource, "va") + 2
			completionSelect(t, tab, shortcutCompletionSource, at, at, false)
			before := languageState(tab)
			changes := 0
			original := tab.Editor.OnChange
			tab.Editor.OnChange = func() { changes++; original() }
			if method == "explicit" {
				a.runSourceCompletionCommand()
			} else {
				shortcutKey(t, tester, editor.SourceActionComplete)
			}
			if a.sourceLanguage.task == nil || a.sourceLanguage.intent.operation != code.ServiceCompletion || !a.sourceLanguage.visible {
				t.Fatal("main source key did not admit the real completion command", a.Message)
			}
			languageUnchanged(t, tab, before)
			completionFinish(t, a, q)
			tester.Frame()
			shortcutAccessible(t, tester, "源码补全候选", "确认补全", "取消补全")
			id := completionCandidate(t, a, "val")
			rows, index := a.sourceLanguageRows(), -1
			for i, row := range rows {
				if row.id == id {
					index = i
				}
			}
			if index < 0 {
				t.Fatal("real val candidate missing")
			}
			a.sourceLanguage.list.Handle.Focus()
			tester.Frame()
			for i := 0; i < index; i++ {
				tester.Key(0, ui.KeyDown)
			}
			languageUnchanged(t, tab, before)
			if method == "shortcut-keyboard" {
				tester.Key(0, ui.KeyEnter)
			} else if err := tester.Click("确认补全"); err != nil {
				t.Fatal(err)
			}
			tester.Frame()
			want := "package p\r\n// 🙂 unchanged\r\nfunc f(val int) { _ = val }\n"
			if tab.Doc.Buffer.String() != want || changes != 1 || a.sourceLanguage.visible || string(mustReadFile(t, path)) != shortcutCompletionSource {
				t.Fatal("completion did not apply one exact unsaved edit", changes, tab.Doc.Buffer.String())
			}
			if !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != shortcutCompletionSource || !reflect.DeepEqual(tab.Doc.Buffer.SelectionSet(), before.selections) || !tab.Doc.Buffer.Redo() || tab.Doc.Buffer.String() != want || q.maximum.Load() != 1 {
				t.Fatal("completion history/selection or single Analyzer queue changed")
			}
		})
	}
}

func TestLanguageShortcutMainSourceReferencesAndReadonlyNavigation(t *testing.T) {
	for _, readonly := range []string{"writable", "document", "editor"} {
		t.Run(readonly, func(t *testing.T) {
			a, tab, q, path := languageFixture(t, ".go", languageGoSource)
			if readonly == "document" {
				tab.Doc.ReadOnly = true
			}
			if readonly == "editor" {
				tab.Editor.SetReadOnly(true)
			}
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			shortcutFocus(t, tester, tab)
			at := text.UTF16Len(languageGoSource[:strings.Index(languageGoSource, "变量 int")])
			if err := tab.Doc.Buffer.SetSelection(at, at); err != nil {
				t.Fatal(err)
			}
			before := languageState(tab)
			shortcutKey(t, tester, editor.SourceActionReferences)
			if a.sourceLanguage.task == nil || a.sourceLanguage.intent.operation != code.ServiceReferences {
				t.Fatal("references key did not start real Analyzer")
			}
			completionFinish(t, a, q)
			tester.Frame()
			if a.sourceLanguage.result.Status != code.ServiceComplete || len(a.sourceLanguage.result.References) != 1 {
				t.Fatalf("real references missing: %+v", a.sourceLanguage.result)
			}
			languageUnchanged(t, tab, before)
			shortcutAccessible(t, tester, "转到语言结果", "关闭语言结果")
			if err := tester.Click("转到语言结果"); err != nil {
				t.Fatal(err)
			}
			tester.Frame()
			want := text.UTF16Len(languageGoSource[:strings.LastIndex(languageGoSource, "变量")])
			if tab.Doc.Buffer.Selection() != (text.Range{Anchor: want, Head: want + 2}) || !tester.Focused(tab.Editor.Label) || tab.Doc.Buffer.String() != before.source || tab.Doc.Buffer.History() != before.history || tab.Doc.Buffer.Version() != before.version || string(mustReadFile(t, path)) != languageGoSource {
				t.Fatal("readonly references did not use checked physical UTF-16 navigation")
			}
		})
	}
}

func TestLanguageShortcutMainSourceRenameUsesSeparateFieldAndRealPreview(t *testing.T) {
	a, tab, q, path := languageFixture(t, ".go", renameLifecycleSource)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	shortcutFocus(t, tester, tab)
	renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
	before := languageState(tab)
	shortcutKey(t, tester, editor.SourceActionRename)
	d := a.sourceLanguage.rename
	if d == nil || a.sourceLanguage.task != nil || d.field.Buffer == tab.Doc.Buffer || d.buffer.MaxBytes() != 512 {
		t.Fatal("F2 did not open separate bounded name Editor")
	}
	shortcutAccessible(t, tester, "本文件符号新名称", "生成重命名预览", "取消重命名")
	if err := d.buffer.SetSelection(0, d.buffer.LenUTF16()); err != nil {
		t.Fatal(err)
	}
	d.field.Focus()
	tester.Frame()
	for _, action := range []editor.SourceAction{editor.SourceActionComplete, editor.SourceActionReferences, editor.SourceActionRename} {
		shortcutKey(t, tester, action)
		if a.sourceLanguage.rename != d || a.sourceLanguage.task != nil {
			t.Fatal("name Editor invoked main source shortcut")
		}
	}
	tester.Type("next")
	if d.buffer.String() != "next" {
		t.Fatal("ordinary native name typing changed")
	}
	languageUnchanged(t, tab, before)
	if err := tester.Click("生成重命名预览"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	renameFinish(t, a, q)
	tester.Frame()
	shortcutAccessible(t, tester, "重命名预览", "确认重命名", "取消重命名")
	if err := tester.Click("确认重命名"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	want := "package p; func f(next int){ _=next }"
	if tab.Doc.Buffer.String() != want || a.sourceLanguage.rename != nil || a.sourceLanguage.visible || string(mustReadFile(t, path)) != renameLifecycleSource {
		t.Fatal("real rename confirmation/save behavior changed")
	}
	if !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != renameLifecycleSource || !reflect.DeepEqual(tab.Doc.Buffer.SelectionSet(), before.selections) || !tab.Doc.Buffer.Redo() || tab.Doc.Buffer.String() != want {
		t.Fatal("rename is not one undoable source edit")
	}
}

// This is a format-view seam containing a real independent project Editor. The
// actual split-right/preview draw path receives it, never the main binding call.
type shortcutProjectView struct{ field *editor.Editor }

func (v *shortcutProjectView) Draw(c *ui.Context)          { v.field.View(c) }
func (v *shortcutProjectView) Release()                    { v.field.ReleaseView() }
func (*shortcutProjectView) Warnings() []string            { return nil }
func (*shortcutProjectView) Export(string) ([]byte, error) { return nil, nil }

func TestLanguageShortcutSplitLeftAndProjectPreviewFocusScope(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", languageGoSource)
	field := editor.New(text.New("project field"))
	field.Label, field.LineNumbers = "独立项目字段", false
	view := &shortcutProjectView{field: field}
	tab.Adapter = &editor.Adapter{ID: "test-project-view", Name: "test project view", Create: func(editor.Host) editor.View { return view }}
	tab.View, tab.Mode = view, "split"
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	shortcutFocus(t, tester, tab)
	renamePosition(t, tab, languageGoSource, strings.Index(languageGoSource, "变量 int"))
	shortcutKey(t, tester, editor.SourceActionReferences)
	completionFinish(t, a, q)
	tester.Frame()
	if len(a.sourceLanguage.result.References) != 1 {
		t.Fatal("split-left did not run accepted references")
	}
	for _, mode := range []string{"split", "preview"} {
		if mode == "preview" && !a.setMode(tab, mode) {
			t.Fatal("actual preview transition refused")
		}
		tester.Frame()
		field.Focus()
		tester.Frame()
		if !tester.Focused(field.Label) {
			t.Fatal("split-right/preview project input not focused")
		}
		before := languageState(tab)
		request, dialog := a.sourceLanguage.request, a.sourceLanguage.rename
		for _, action := range []editor.SourceAction{editor.SourceActionComplete, editor.SourceActionReferences, editor.SourceActionRename} {
			shortcutKey(t, tester, action)
		}
		if a.sourceLanguage.request != request || a.sourceLanguage.rename != dialog || a.sourceLanguage.task != nil || q.outstanding.Load() != 0 {
			t.Fatal("project/preview Editor invoked main source")
		}
		languageUnchanged(t, tab, before)
		tester.Type("!")
		if !strings.Contains(field.Buffer.String(), "!") {
			t.Fatal("project Editor ordinary native text stopped")
		}
	}
}

func TestLanguageShortcutFindUtilityAndOverlayFocusNegatives(t *testing.T) {
	for _, field := range []string{"查找内容", "替换内容", "palette", "picker", "goto", "settings", "plugins", "modal", "restoring", "window-blur"} {
		t.Run(field, func(t *testing.T) {
			a, tab, q, _ := languageFixture(t, ".go", languageGoSource)
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			shortcutFocus(t, tester, tab)
			switch field {
			case "查找内容", "替换内容":
				a.openFind()
				tester.Frame()
				if err := tester.Click(field); err != nil {
					t.Fatal(err)
				}
			case "palette":
				if !a.openCommandPalette() {
					t.Fatal("palette refused")
				}
				tester.Frame()
			case "picker":
				a.openDocumentPicker()
				tester.Frame()
			case "goto":
				a.openGoToLine()
				tester.Frame()
			case "settings":
				a.ShowSettings = true
			case "plugins":
				a.ShowPlugins = true
			case "modal":
				a.modalDepth++
			case "restoring":
				a.restoring = true
			case "window-blur":
				tester.SetFocused(false)
			}
			before := languageState(tab)
			request, presentation := a.sourceLanguage.request, a.sourceLanguage.presentation
			for _, action := range []editor.SourceAction{editor.SourceActionComplete, editor.SourceActionReferences, editor.SourceActionRename} {
				shortcutKey(t, tester, action)
			}
			if a.sourceLanguage.request != request || a.sourceLanguage.presentation != presentation || a.sourceLanguage.rename != nil || a.sourceLanguage.task != nil || q.outstanding.Load() != 0 {
				t.Fatal("non-source focus/disabled source invoked a language command", field)
			}
			languageUnchanged(t, tab, before)
			if field == "window-blur" {
				tester.SetFocused(true)
				shortcutFocus(t, tester, tab)
				shortcutKey(t, tester, editor.SourceActionRename)
				if a.sourceLanguage.rename == nil || a.sourceLanguage.request != request+1 || a.sourceLanguage.presentation != presentation+2 || a.sourceLanguage.task != nil || q.outstanding.Load() != 0 {
					t.Fatal("a fresh F2 after window refocus did not open exactly one rename")
				}
				languageUnchanged(t, tab, before)
			}
			if field == "查找内容" || field == "替换内容" {
				tester.Type("needle")
				if field == "查找内容" && a.Query != "needle" || field == "替换内容" && a.Replacement != "needle" {
					t.Fatal("utility text typing was suppressed")
				}
			}
			// Restore fixture shutdown admission after the pure negative checks.
			a.modalDepth, a.restoring = 0, false
		})
	}
}

func TestLanguageShortcutReadonlyEditsAreHandledVisibleRefusals(t *testing.T) {
	for _, readonly := range []string{"document", "editor"} {
		for _, action := range []editor.SourceAction{editor.SourceActionComplete, editor.SourceActionRename} {
			t.Run(readonly+sourceLanguageOperation(map[editor.SourceAction]code.ServiceOperation{editor.SourceActionComplete: code.ServiceCompletion, editor.SourceActionRename: code.ServiceRename}[action]), func(t *testing.T) {
				a, tab, q, _ := languageFixture(t, ".go", shortcutCompletionSource)
				if readonly == "document" {
					tab.Doc.ReadOnly = true
				} else {
					tab.Editor.SetReadOnly(true)
				}
				tester := ui.NewTester(a.View, 760, 480)
				tester.Frame()
				shortcutFocus(t, tester, tab)
				before := languageState(tab)
				shortcutKey(t, tester, action)
				want := "只读源码不能重命名"
				if action == editor.SourceActionComplete {
					want = editor.ErrSourceReadOnly.Error()
				}
				if a.sourceLanguage.task != nil || a.sourceLanguage.rename != nil || q.outstanding.Load() != 0 || a.Message != want {
					t.Fatal("readonly action was not visibly refused", a.Message)
				}
				languageUnchanged(t, tab, before)
			})
		}
	}
}

func TestLanguageShortcutExactModifiersRepeatConflictAndDistinctRetry(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", languageGoSource)
	conflicts := 0
	tester := ui.NewTester(func(c *ui.Context) {
		if c.Shortcut(ui.Ctrl, ui.KeySpace) {
			conflicts++
		}
		if c.Shortcut(ui.Shift, ui.KeyF12) {
			conflicts++
		}
		if c.Shortcut(0, ui.KeyF2) {
			conflicts++
		}
		a.View(c)
	}, 760, 480)
	tester.Frame()
	shortcutFocus(t, tester, tab)
	renamePosition(t, tab, languageGoSource, strings.Index(languageGoSource, "变量 int"))
	before := languageState(tab)
	for _, key := range []struct {
		mods ui.Modifiers
		key  ui.Key
	}{
		{ui.Ctrl | ui.Shift, ui.KeySpace}, {ui.Alt, ui.KeySpace}, {ui.Super, ui.KeySpace}, {ui.Ctrl | ui.Alt, ui.KeySpace},
		{ui.Ctrl, ui.KeyF2}, {ui.Shift, ui.KeyF2}, {ui.Ctrl, ui.KeyF12},
	} {
		tester.Key(key.mods, key.key)
	}
	// Plain F12 is Inspector-owned; the Editor side tests cover both Inspector
	// builds. Check Host dispatch through the actual raw public event as well.
	_ = tab.Editor.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyF12})
	if a.sourceLanguage.task != nil || a.sourceLanguage.rename != nil || conflicts != 0 {
		t.Fatal("extra modifiers or F12 invoked B6")
	}
	languageUnchanged(t, tab, before)
	started, release := languageHold(t, a)
	shortcutKey(t, tester, editor.SourceActionReferences)
	first := a.sourceLanguage.task
	if first == nil || conflicts != 0 {
		t.Fatal("source element did not beat conflicting window shortcut once")
	}
	languageReceive(t, started)
	request, presentation := a.sourceLanguage.request, a.sourceLanguage.presentation
	for i := 0; i < 3; i++ {
		tester.KeyRepeat(ui.Shift, ui.KeyF12)
		tester.KeyRepeat(ui.Ctrl, ui.KeySpace)
		tester.KeyRepeat(0, ui.KeyF2)
	}
	tester.Frame()
	if a.sourceLanguage.task != first || a.sourceLanguage.pending != nil || a.sourceLanguage.request != request || a.sourceLanguage.presentation != presentation || conflicts != 0 {
		t.Fatal("held key created/replaced a request/dialog")
	}
	shortcutKey(t, tester, editor.SourceActionReferences)
	if a.sourceLanguage.task != first || a.sourceLanguage.pending == nil || a.sourceLanguage.request == request {
		t.Fatal("later distinct source key could not explicitly queue a request")
	}
	a.sourceLanguageHooks.beforeAnalyze = nil
	release()
	languageWait(t, first.done)
	languageReceive(t, q.calls)()
	completionFinish(t, a, q)
	if a.sourceLanguage.result.Status != code.ServiceComplete || len(a.sourceLanguage.result.References) != 1 || conflicts != 0 || q.maximum.Load() != 1 {
		t.Fatal("real latest request or one physical slot failed")
	}
	languageUnchanged(t, tab, before)
}

func TestLanguageShortcutQueuedOriginalTargetAndIntentDoNotFollowCaret(t *testing.T) {
	for _, action := range []editor.SourceAction{editor.SourceActionReferences, editor.SourceActionComplete} {
		for _, change := range []string{"caret", "selection-ABA", "text-undo"} {
			t.Run(sourceLanguageOperation(map[editor.SourceAction]code.ServiceOperation{editor.SourceActionReferences: code.ServiceReferences, editor.SourceActionComplete: code.ServiceCompletion}[action])+change, func(t *testing.T) {
				source := languageGoSource
				if action == editor.SourceActionComplete {
					source = shortcutCompletionSource
				}
				a, tab, q, _ := languageFixture(t, ".go", source)
				tester := ui.NewTester(a.View, 760, 480)
				tester.Frame()
				shortcutFocus(t, tester, tab)
				at := strings.Index(source, "变量 int")
				if action == editor.SourceActionComplete {
					at = strings.LastIndex(source, "va") + 2
				}
				renamePosition(t, tab, source, at)
				started, release := languageHold(t, a)
				first := languageRequest(t, a, languageIntent(code.ServiceSymbols))
				languageReceive(t, started)
				original := a.sourceLanguage.owner.target
				shortcutKey(t, tester, action)
				pending := a.sourceLanguage.pending
				if pending == nil || !reflect.DeepEqual(pending.owner.target, original) || a.sourceLanguage.task != first || q.outstanding.Load() != 0 {
					t.Fatal("source key materialized/replaced original occupied-slot target")
				}
				if action == editor.SourceActionReferences && pending.intent.utf16Offset != text.UTF16Len(source[:at]) {
					t.Fatal("queued references lost original UTF-16 intent")
				}
				if action == editor.SourceActionComplete && (pending.intent.completion == nil || pending.intent.completion.oldText != "va") {
					t.Fatal("queued completion lost exact original edit descriptor")
				}
				set := tab.Doc.Buffer.SelectionSet()
				switch change {
				case "caret":
					_ = tab.Doc.Buffer.SetSelection(0, 0)
				case "selection-ABA":
					_ = tab.Doc.Buffer.SetSelection(0, 0)
					_ = tab.Doc.Buffer.SetSelectionSet(set)
				case "text-undo":
					if tab.Doc.Buffer.ReplaceUTF16(0, 0, " ") != nil || !tab.Doc.Buffer.Undo() {
						t.Fatal("text ABA setup")
					}
				}
				before := languageState(tab)
				a.sourceLanguageHooks.beforeAnalyze = nil
				release()
				languageWait(t, first.done)
				languageReceive(t, q.calls)()
				if action == editor.SourceActionReferences && change != "text-undo" {
					second := a.sourceLanguage.task
					if second == nil || !reflect.DeepEqual(a.sourceLanguage.owner.target, original) || a.sourceLanguage.intent.byteOffset != at {
						t.Fatal("read queue reinterpreted the later caret or recaptured original target")
					}
					completionFinish(t, a, q)
					if len(a.sourceLanguage.result.References) != 1 {
						t.Fatal("original reference intent did not run real Analyzer")
					}
				} else if a.sourceLanguage.task != nil || a.sourceLanguage.status != sourceLanguageStale {
					t.Fatal("stale editing/text target was silently repaired")
				}
				languageUnchanged(t, tab, before)
			})
		}
	}
}

func TestLanguageShortcutBusyRenameKeepsOriginalTargetAndOnePhysicalSlot(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	shortcutFocus(t, tester, tab)
	at := strings.Index(renameLifecycleSource, "x int")
	renamePosition(t, tab, renameLifecycleSource, at)
	before := languageState(tab)
	started, release := languageHold(t, a)
	hold := a.sourceLanguageHooks.beforeAnalyze
	var active, maximum, count atomic.Int32
	a.sourceLanguageHooks.beforeAnalyze = func(ctx context.Context, id uint64) {
		languageMax(&maximum, active.Add(1))
		count.Add(1)
		hold(ctx, id)
	}
	a.sourceLanguageHooks.afterAnalyze = func(uint64) { active.Add(-1) }
	first := languageRequest(t, a, languageIntent(code.ServiceSymbols))
	languageReceive(t, started)
	original := a.sourceLanguage.owner.target
	shortcutKey(t, tester, editor.SourceActionRename)
	d := a.sourceLanguage.rename
	if d == nil || !reflect.DeepEqual(d.original.target, original) || d.utf16Offset != text.UTF16Len(renameLifecycleSource[:at]) || d.byteOffset != at || d.buffer.String() != "" || a.sourceLanguage.task != first || a.sourceLanguage.pending != nil {
		t.Fatal("F2 on an occupied slot failed to reuse the whole original target")
	}
	for _, name := range []string{"one", "finalName"} {
		renameName(t, a, name)
		if err := a.requestSourceRename(); err != nil {
			t.Fatal(err)
		}
		pending := a.sourceLanguage.pending
		if pending == nil || !reflect.DeepEqual(pending.owner.target, original) || pending.intent.newName != name || a.sourceLanguage.task != first || count.Load() != 1 || q.outstanding.Load() != 0 {
			t.Fatal("busy rename changed the admitted target or started more work")
		}
	}
	release()
	languageWait(t, first.done)
	oldNotice := languageReceive(t, q.calls)
	oldNotice()
	second := a.sourceLanguage.task
	if second == nil || second == first || !reflect.DeepEqual(a.sourceLanguage.owner.target, original) {
		t.Fatal("missing latest rename with its original target")
	}
	languageReceive(t, started)
	oldNotice()
	renameFinish(t, a, q)
	if a.sourceLanguage.result.Status != code.ServiceComplete || a.sourceLanguage.intent.newName != "finalName" || len(d.patch) != 2 || count.Load() != 2 || maximum.Load() != 1 || active.Load() != 0 || q.maximum.Load() != 1 || q.outstanding.Load() != 0 {
		t.Fatal("rename preview did not preserve latest intent and the physical slot", a.sourceLanguage.status, a.sourceLanguage.reason)
	}
	languageUnchanged(t, tab, before)
}

func TestLanguageShortcutNativeAndForeignOwnersRemainAvailableForOwnEnd(t *testing.T) {
	for _, kind := range []string{"single", "multi", "markless-single", "markless-multi", "foreign-group", "foreign-composition"} {
		t.Run(kind, func(t *testing.T) {
			a, tab, q, _ := languageFixture(t, ".go", shortcutCompletionSource)
			if strings.HasPrefix(kind, "markless") {
				b := text.NewWithLimit(shortcutCompletionSource, len(shortcutCompletionSource))
				tab.Doc.Buffer = b
				tab.Editor.SetBuffer(b)
			}
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			shortcutFocus(t, tester, tab)
			at := text.UTF16Len(shortcutCompletionSource[:strings.LastIndex(shortcutCompletionSource, "va")+2])
			if err := tab.Doc.Buffer.SetSelection(at, at); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(kind, "multi") {
				if err := tab.Doc.Buffer.SetSelectionSet(text.SelectionSet{Ranges: []text.Range{{Anchor: 1, Head: 0}, {Anchor: at, Head: at - 2}}, Primary: 1}); err != nil {
					t.Fatal(err)
				}
			}
			b := tab.Doc.Buffer
			var token text.CompositionToken
			switch kind {
			case "foreign-group":
				if err := b.BeginGroup(); err != nil {
					t.Fatal(err)
				}
			case "foreign-composition":
				var err error
				token, err = b.BeginCompositionGroup()
				if err != nil {
					t.Fatal(err)
				}
			default:
				tester.Compose("🙂", 1)
			}
			if tab.Editor.InputActionError() == nil {
				t.Fatal("fixture did not retain original native/foreign input")
			}
			if strings.HasPrefix(kind, "markless") {
				if _, marked := tab.Editor.MarkedRange(); marked {
					t.Fatal("first refused input unexpectedly acquired a mark")
				}
			}
			before := languageState(tab)
			binding := a.sourceLanguage.shortcut
			request, presentation := a.sourceLanguage.request, a.sourceLanguage.presentation
			for _, action := range []editor.SourceAction{editor.SourceActionComplete, editor.SourceActionReferences, editor.SourceActionRename} {
				a.Message = ""
				issues := len(a.Issues)
				shortcutKey(t, tester, action)
				if a.Message == "" || len(a.Issues) != issues+1 {
					t.Fatal("recognized native or foreign ownership refusal was not reported once", action)
				}
				languageUnchanged(t, tab, before)
			}
			errorsBeforeRepeat := len(a.Issues)
			tester.KeyRepeat(ui.Ctrl, ui.KeySpace)
			tester.KeyRepeat(ui.Shift, ui.KeyF12)
			tester.KeyRepeat(0, ui.KeyF2)
			tester.Frame()
			if a.sourceLanguage.task != nil || a.sourceLanguage.pending != nil || a.sourceLanguage.rename != nil || a.sourceLanguage.request != request || a.sourceLanguage.presentation != presentation || a.sourceLanguage.shortcut != binding || q.outstanding.Load() != 0 || len(a.Issues) != errorsBeforeRepeat || !tester.Focused(tab.Editor.Label) {
				t.Fatal("refusal/repeat changed operation, binding, native focus or repeated errors")
			}
			languageUnchanged(t, tab, before)
			switch kind {
			case "foreign-group":
				b.EndGroup()
			case "foreign-composition":
				if err := b.CancelCompositionGroup(token); err != nil {
					t.Fatal(err)
				}
			default:
				tester.Key(0, ui.KeyEscape)
				if b.String() != shortcutCompletionSource {
					t.Fatal("native owner could not cancel its own exact bytes")
				}
			}
			if tab.Editor.InputActionError() != nil || !tester.Focused(tab.Editor.Label) {
				t.Fatal("owner's own end/cancel was detached by shortcut")
			}
		})
	}
}

func TestLanguageShortcutSavedBindingLifecycleWithoutRequestAndBetweenFrames(t *testing.T) {
	for _, kind := range []string{"buffer-ABA", "readonly-ABA", "view-release", "retire-no-request", "tab-ABA", "split-ABA", "preview-ABA", "configure", "session", "editor", "document", "buffer", "path", "name", "encoding", "bom", "doc-readonly", "profile", "route", "manager", "SaveAs", "SaveAs-same-path", "plugins", "restoring", "shutdown", "byte-inspector"} {
		t.Run(kind, func(t *testing.T) {
			a, tab, q, path := languageFixture(t, ".go", languageGoSource)
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			old := a.sourceLanguage.shortcut
			if old == nil || !a.sourceLanguageShortcutCurrent(old) || a.sourceLanguage.owner != nil || a.sourceLanguage.task != nil {
				t.Fatal("fixture lacks source binding before any request")
			}
			b := tab.Doc.Buffer
			switch kind {
			case "buffer-ABA":
				tab.Editor.SetBuffer(text.New(languageGoSource))
				tab.Editor.SetBuffer(b)
			case "readonly-ABA":
				tab.Editor.SetReadOnly(true)
				tab.Editor.SetReadOnly(false)
			case "view-release":
				tab.Editor.ReleaseView()
			case "retire-no-request":
				a.retireSourceLanguage(tab)
			case "tab-ABA":
				a.NewDocument("text")
				if !a.Select(0) {
					t.Fatal("tab A-B-A transition refused")
				}
			case "split-ABA", "preview-ABA":
				tab.Adapter = &editor.Adapter{ID: "test-view", Create: func(editor.Host) editor.View { return foldSplitTestView{} }}
				mode := "split"
				if kind == "preview-ABA" {
					mode = "preview"
				}
				if !a.setMode(tab, mode) || !a.setMode(tab, "source") {
					t.Fatal("actual presentation ABA refused")
				}
			case "configure":
				a.configureTab(tab)
			case "session":
				a.resetSourceLanguageSession()
			case "editor":
				tab.Editor = editor.New(b)
			case "document":
				d := document.New(languageGoSource)
				d.Buffer = b
				d.Path = path
				tab.Doc = d
			case "buffer":
				b = text.New(languageGoSource)
				tab.Doc.Buffer = b
				tab.Editor.SetBuffer(b)
			case "path":
				tab.Doc.Path = filepath.Join(filepath.Dir(path), "other.go")
			case "name":
				tab.Name = "other.go"
			case "encoding":
				tab.Doc.Encoding = document.UTF16LE
			case "bom":
				tab.Doc.BOM = !tab.Doc.BOM
			case "doc-readonly":
				tab.Doc.ReadOnly = true
			case "profile":
				tab.sourceLanguageProfile = code.StructureJSON
			case "route":
				tab.sourceLanguageRoute++
			case "manager":
				a.pluginManager = nil
			case "SaveAs", "SaveAs-same-path":
				destination := path
				if kind == "SaveAs" {
					destination = filepath.Join(filepath.Dir(path), "saved.go")
				}
				a.saveDialog = func(*Tab) (string, error) { return destination, nil }
				if !a.Save(tab, true) {
					t.Fatal("real same-extension SaveAs failed", a.Message)
				}
			case "plugins":
				a.ShowPlugins = true
			case "restoring":
				a.restoring = true
			case "shutdown":
				a.shuttingDown.Store(true)
			case "byte-inspector":
				tab.ByteInspector = true
			}
			// No frame intervenes between a lifecycle transition and invocation
			// of the actual saved Host closure from the previous input binding.
			before := languageState(tab)
			request, presentation, activation := a.sourceLanguage.request, a.sourceLanguage.presentation, a.sourceLanguage.activation
			binding, owner, pending, task, dialog := a.sourceLanguage.shortcut, a.sourceLanguage.owner, a.sourceLanguage.pending, a.sourceLanguage.task, a.sourceLanguage.rename
			a.Message, a.sourceLanguage.message = "successor host exit", "successor panel exit"
			for _, action := range []editor.SourceAction{editor.SourceActionComplete, editor.SourceActionReferences, editor.SourceActionRename} {
				old.handler(action)
			}
			if a.sourceLanguage.request != request || a.sourceLanguage.presentation != presentation || a.sourceLanguage.activation != activation || a.sourceLanguage.shortcut != binding || a.sourceLanguage.owner != owner || a.sourceLanguage.pending != pending || a.sourceLanguage.task != task || a.sourceLanguage.rename != dialog || a.Message != "successor host exit" || a.sourceLanguage.message != "successor panel exit" || q.outstanding.Load() != 0 {
				t.Fatal("saved binding wrote into a replacement lifecycle", kind)
			}
			languageUnchanged(t, tab, before)
			a.restoring = false
			a.shuttingDown.Store(false)
		})
	}
}

func TestLanguageShortcutRoutedF2DoesNotCrossSamePathSaveAs(t *testing.T) {
	a, tab, q, path := languageFixture(t, ".go", renameLifecycleSource)
	var tester *ui.Tester
	var postSave sourceLanguageState
	var postSaveDoc languageDocumentState
	var postSaveMessage string
	var postSaveIssues []string
	armed, saves := false, 0
	tester = ui.NewTester(func(c *ui.Context) {
		if armed {
			// MyGo has already routed the queued key using the preceding frame's
			// source element registration. Save As runs before the App draws it.
			armed = false
			if !a.Save(tab, true) {
				t.Fatal("real same-path Save As failed", a.Message)
			}
			if !tester.Focused(tab.Editor.Label) || a.sourceLanguage.shortcut != nil {
				t.Fatal("Save As must preserve actual source focus and retire the old binding before redraw")
			}
			postSave, postSaveDoc = a.sourceLanguage, languageState(tab)
			postSaveMessage = a.Message
			postSaveIssues = append([]string(nil), a.Issues...)
		}
		a.View(c)
	}, 760, 480)
	tester.Frame()
	shortcutFocus(t, tester, tab)
	renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
	old := a.sourceLanguage.shortcut
	if old == nil || !a.sourceLanguageShortcutCurrent(old) {
		t.Fatal("fixture lacks a current source shortcut binding")
	}
	a.saveDialog = func(*Tab) (string, error) {
		saves++
		return path, nil
	}

	armed = true
	tester.Key(0, ui.KeyF2)
	after := a.sourceLanguage
	fresh := after.shortcut
	if armed || saves != 1 || tab.Doc.Path != path || !tester.Focused(tab.Editor.Label) || fresh == nil || fresh == old ||
		postSave.shortcutSequence != old.generation+1 || fresh.generation != postSave.shortcutSequence+1 ||
		!a.sourceLanguageShortcutCurrent(fresh) || tab.Editor.SourceActionBindingCurrent(old.lease) {
		t.Fatal("queued F2 frame did not preserve source focus and advance exactly to the replacement binding")
	}
	// A fresh registration is expected. Every other language field and the
	// notices must retain their post-save values after consuming the stale F2.
	after.shortcut, after.shortcutSequence = postSave.shortcut, postSave.shortcutSequence
	if !reflect.DeepEqual(after, postSave) || a.Message != postSaveMessage || !slices.Equal(a.Issues, postSaveIssues) || q.outstanding.Load() != 0 {
		t.Fatal("stale registered F2 changed post-save language state, notices, or work queue")
	}
	languageUnchanged(t, tab, postSaveDoc)
	if _, ok := tester.Access("本文件符号新名称"); ok {
		t.Fatal("stale registered F2 rendered the rename field")
	}

	tester.Key(0, ui.KeyF2)
	s := &a.sourceLanguage
	if saves != 1 || s.rename == nil || !s.visible || s.intent.operation != code.ServiceRename || s.activation != postSave.activation+1 ||
		s.request != postSave.request+1 || s.presentation != postSave.presentation+2 || s.task != nil || s.pending != nil || q.outstanding.Load() != 0 {
		t.Fatal("a distinct F2 after re-registration did not open exactly one rename")
	}
	shortcutAccessible(t, tester, "本文件符号新名称", "生成重命名预览", "取消重命名")
	languageUnchanged(t, tab, postSaveDoc)
}

func TestLanguageShortcutLeaseAndHostGenerationStayStableAcrossOrdinaryFrames(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", languageGoSource)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	shortcutFocus(t, tester, tab)
	binding := a.sourceLanguage.shortcut
	for i := 0; i < 3; i++ {
		tester.Frame()
	}
	if binding == nil || a.sourceLanguage.shortcut != binding || !tab.Editor.SourceActionBindingCurrent(binding.lease) {
		t.Fatal("ordinary frames churned a binding/lease")
	}
	if err := tab.Doc.Buffer.SetSelection(0, 0); err != nil {
		t.Fatal(err)
	}
	tester.Type(" ")
	tester.Frame()
	if a.sourceLanguage.shortcut != binding || !a.sourceLanguageShortcutCurrent(binding) {
		t.Fatal("ordinary caret/text edit churned source binding")
	}
	if !a.Save(tab, false) {
		t.Fatal("normal same-metadata save failed", a.Message)
	}
	tester.Frame()
	if a.sourceLanguage.shortcut != binding {
		t.Fatal("same-metadata save revoked an unchanged source presentation")
	}
	// Setter ABA invalidates the lease without a source snapshot, then the
	// actual next source draw installs exactly one replacement generation.
	tab.Editor.SetReadOnly(true)
	tab.Editor.SetReadOnly(false)
	if a.sourceLanguageShortcutCurrent(binding) {
		t.Fatal("readonly setter ABA revived the old lease")
	}
	tester.Frame()
	fresh := a.sourceLanguage.shortcut
	if fresh == nil || fresh == binding || !a.sourceLanguageShortcutCurrent(fresh) {
		t.Fatal("next admitted source frame did not rebind after lifecycle change")
	}
	a.runSourceLanguageCommand("symbols")
	task, request, owner := a.sourceLanguage.task, a.sourceLanguage.request, a.sourceLanguage.owner
	binding.handler(editor.SourceActionRename)
	binding.handler(editor.SourceActionComplete)
	binding.handler(editor.SourceActionReferences)
	if a.sourceLanguage.task != task || a.sourceLanguage.request != request || a.sourceLanguage.owner != owner || a.sourceLanguage.shortcut != fresh || a.sourceLanguage.rename != nil {
		t.Fatal("saved binding canceled/overwrote a live successor")
	}
	completionFinish(t, a, q)
	if a.sourceLanguage.result.Status != code.ServiceComplete {
		t.Fatal("successor real Analyzer did not survive saved callback")
	}
}

func TestLanguageShortcutStrictJSONKeepsRawTokenAndRenameRefusal(t *testing.T) {
	const source = "{\"a\":1,\"\\u0061\":2,\"dst\":\"old\"}\r\n"
	a, tab, q, path := languageFixture(t, ".json", source)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	shortcutFocus(t, tester, tab)
	at := strings.Index(source, `"old"`)
	completionSelect(t, tab, source, at, at+len(`"old"`), true)
	before := languageState(tab)
	shortcutKey(t, tester, editor.SourceActionComplete)
	completionFinish(t, a, q)
	tester.Frame()
	if a.sourceLanguage.result.Status != code.ServiceComplete || a.sourceLanguage.intent.completion == nil || !a.sourceLanguage.intent.completion.proven {
		t.Fatal("JSON shortcut did not prove the original full raw token")
	}
	languageUnchanged(t, tab, before)
	completionConfirm(t, a, `"\u0061"`)
	want := "{\"a\":1,\"\\u0061\":2,\"dst\":\"\\u0061\"}\r\n"
	if tab.Doc.Buffer.String() != want || string(mustReadFile(t, path)) != source || !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != source || !reflect.DeepEqual(tab.Doc.Buffer.SelectionSet(), before.selections) {
		t.Fatal("JSON shortcut normalized raw escapes, saved or broke one-step undo")
	}
	shortcutFocus(t, tester, tab)
	before = languageState(tab)
	shortcutKey(t, tester, editor.SourceActionRename)
	if a.sourceLanguage.rename != nil || a.sourceLanguage.task != nil || !strings.Contains(a.Message, "仅支持 Go") {
		t.Fatal("JSON F2 fell back to a fabricated rename", a.Message)
	}
	languageUnchanged(t, tab, before)
}

func TestLanguageShortcutUnavailableRouteAndMultipleSelectionsAreTruthful(t *testing.T) {
	for _, kind := range []string{"unsupported-text", "actual-manager-disabled", "byte-inspector", "multi-selection", "invalid-go"} {
		t.Run(kind, func(t *testing.T) {
			suffix, source := ".go", shortcutCompletionSource
			if kind == "unsupported-text" {
				suffix = ".txt"
			}
			if kind == "invalid-go" {
				source = "package p; func broken("
			}
			a, tab, q, _ := languageFixture(t, suffix, source)
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			shortcutFocus(t, tester, tab)
			if kind == "actual-manager-disabled" {
				if err := a.pluginManager.SetEnabled("code", false); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "byte-inspector" {
				tab.ByteInspector = true
				a.configureTab(tab)
			}
			if kind == "multi-selection" {
				if err := tab.Doc.Buffer.SetSelectionSet(text.SelectionSet{Ranges: []text.Range{{Anchor: 1, Head: 0}, {Anchor: 5, Head: 5}}, Primary: 1}); err != nil {
					t.Fatal(err)
				}
			}
			before := languageState(tab)
			shortcutKey(t, tester, editor.SourceActionComplete)
			if kind == "invalid-go" {
				completionFinish(t, a, q)
				if a.sourceLanguage.result.Status != code.ServiceInvalid || len(a.sourceLanguage.result.Diagnostics) == 0 || len(a.sourceLanguage.result.Completion) != 0 {
					t.Fatal("invalid source became a successful empty vocabulary")
				}
			} else {
				if a.sourceLanguage.task != nil || a.sourceLanguage.rename != nil || q.outstanding.Load() != 0 {
					t.Fatal("unavailable source performed analysis")
				}
				if kind != "byte-inspector" && a.Message == "" {
					t.Fatal("recognized live refusal was silent")
				}
			}
			languageUnchanged(t, tab, before)
		})
	}
}

func TestLanguageShortcutKeyAndNativeTextRemainIndependent(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", languageGoSource)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	shortcutFocus(t, tester, tab)
	renamePosition(t, tab, languageGoSource, strings.Index(languageGoSource, "变量 int"))
	before := languageState(tab)
	shortcutKey(t, tester, editor.SourceActionReferences)
	languageUnchanged(t, tab, before)
	completionFinish(t, a, q)
	if err := tab.Doc.Buffer.SetSelection(0, 0); err != nil {
		t.Fatal(err)
	}
	// Type is a separate native TextInput event. No key flag may swallow it.
	tester.Type(" ")
	if tab.Doc.Buffer.String() != " "+languageGoSource || !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != languageGoSource {
		t.Fatal("shortcut swallowed a later independent native space or broke history")
	}
	shortcutFocus(t, tester, tab)
	if err := tab.Doc.Buffer.SetSelection(0, 0); err != nil {
		t.Fatal(err)
	}
	tester.Compose("拼", 1)
	request := a.sourceLanguage.request
	shortcutKey(t, tester, editor.SourceActionReferences)
	if a.sourceLanguage.request != request {
		t.Fatal("composing input crossed shortcut admission")
	}
	tester.Type("终")
	if tab.Doc.Buffer.String() != "终"+languageGoSource || tab.Editor.InputActionError() != nil || !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != languageGoSource {
		t.Fatal("retained native owner could not commit its own final text after shortcut refusal")
	}
}

func TestLanguageShortcutErrorCallbackReentryIsTerminal(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", shortcutCompletionSource)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	shortcutFocus(t, tester, tab)
	tester.Compose("拼", 1)
	if tab.Editor.InputActionError() == nil {
		t.Fatal("native refusal fixture missing")
	}
	calls := 0
	var request, presentation uint64
	var replacement *sourceLanguageTask
	var buffer *text.Buffer
	var owner *sourceLanguageOwner
	tab.Editor.OnError = func(err error) {
		if err == nil {
			t.Fatal("missing original admission error")
		}
		calls++
		// The callback owns this explicit cancellation and new operation.
		tab.Editor.UnmarkText()
		a.runSourceLanguageCommand("symbols")
		a.Message, a.sourceLanguage.message = "error callback exit", "new result exit"
		request, presentation, replacement, owner, buffer = a.sourceLanguage.request, a.sourceLanguage.presentation, a.sourceLanguage.task, a.sourceLanguage.owner, tab.Editor.Buffer
	}
	if !tab.Editor.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Mods: ui.Ctrl, Key: ui.KeySpace}) {
		t.Fatal("recognized refusal fell through")
	}
	if calls != 1 || replacement == nil || a.sourceLanguage.task != replacement || a.sourceLanguage.request != request || a.sourceLanguage.presentation != presentation || a.sourceLanguage.owner != owner || tab.Editor.Buffer != buffer || a.Message != "error callback exit" || a.sourceLanguage.message != "new result exit" {
		t.Fatal("old shortcut wrote after OnError replacement")
	}
	completionFinish(t, a, q)
}

func TestLanguageShortcutApplyCallbackPreservesReplacementExitState(t *testing.T) {
	for _, operation := range []string{"completion", "rename"} {
		for _, kind := range []string{"new-tab", "new-request", "new-dialog", "new-buffer"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				source := shortcutCompletionSource
				if operation == "rename" {
					source = renameLifecycleSource
				}
				a, tab, q, _ := languageFixture(t, ".go", source)
				tester := ui.NewTester(a.View, 760, 480)
				tester.Frame()
				shortcutFocus(t, tester, tab)
				at := strings.LastIndex(source, "va") + 2
				if operation == "rename" {
					at = strings.Index(source, "x int")
				}
				renamePosition(t, tab, source, at)
				if operation == "completion" {
					shortcutKey(t, tester, editor.SourceActionComplete)
					completionFinish(t, a, q)
					a.queueSourceLanguageAction("complete", completionCandidate(t, a, "val"))
				} else {
					shortcutKey(t, tester, editor.SourceActionRename)
					renameName(t, a, "next")
					if err := a.requestSourceRename(); err != nil {
						t.Fatal(err)
					}
					renameFinish(t, a, q)
					a.queueSourceLanguageAction("rename-confirm", "")
				}
				if a.sourceLanguage.action == nil {
					t.Fatal("real candidate/preview did not queue Apply")
				}
				original := tab.Editor.OnChange
				calls := 0
				var current *Tab
				var request, presentation uint64
				var task *sourceLanguageTask
				var dialog *sourceRenameDialog
				var buffer *text.Buffer
				var selections text.SelectionSet
				tab.Editor.OnChange = func() {
					calls++
					original()
					switch kind {
					case "new-tab":
						a.NewDocument("text")
					case "new-request":
						a.runSourceLanguageCommand("symbols")
					case "new-dialog":
						if err := a.openSourceRename(); err != nil {
							t.Fatal(err)
						}
					case "new-buffer":
						buffer = text.New("replacement callback bytes")
						tab.Doc.Buffer = buffer
						tab.Editor.SetBuffer(buffer)
					}
					a.Message, a.sourceLanguage.message = "apply callback exit", "replacement panel exit"
					current, request, presentation, task, dialog = a.Current(), a.sourceLanguage.request, a.sourceLanguage.presentation, a.sourceLanguage.task, a.sourceLanguage.rename
					buffer, selections = tab.Editor.Buffer, tab.Editor.Buffer.SelectionSet()
				}
				a.processSourceLanguageActions()
				if calls != 1 || a.Current() != current || a.sourceLanguage.request != request || a.sourceLanguage.presentation != presentation || a.sourceLanguage.task != task || a.sourceLanguage.rename != dialog || tab.Editor.Buffer != buffer || !reflect.DeepEqual(buffer.SelectionSet(), selections) || a.Message != "apply callback exit" || a.sourceLanguage.message != "replacement panel exit" {
					t.Fatal("old Apply wrote after final OnChange")
				}
				if kind != "new-buffer" {
					want := "package p; func f(next int){ _=next }"
					if operation == "completion" {
						want = "package p\r\n// 🙂 unchanged\r\nfunc f(val int) { _ = val }\n"
					}
					if tab.Doc.Buffer.String() != want {
						t.Fatal("callback reentry rolled back committed bytes")
					}
				}
				if task != nil {
					completionFinish(t, a, q)
				}
			})
		}
	}
}
