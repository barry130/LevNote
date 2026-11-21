package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
)

func TestCompletionPaletteAccessibleCandidatesAndKeyboardPointerUndo(t *testing.T) {
	const source = "package p; func f(val int) { _ = va }"
	for _, pointer := range []bool{false, true} {
		a, tab, q, path := languageFixture(t, ".go", source)
		tester := ui.NewTester(a.View, 760, 480)
		tester.Frame()
		at := strings.LastIndex(source, "va") + 2
		completionSelect(t, tab, source, at, at, false)
		before := languageState(tab)
		paletteOpen(t, a, tester)
		tester.Type("语言 源码补全")
		if a.commandPalette.selectedID != "language.complete" {
			t.Fatal("real command palette did not select completion", a.commandPalette.selectedID)
		}
		tester.Key(0, ui.KeyEnter)
		tester.Frame()
		if a.commandPalette.open || a.sourceLanguage.task == nil || a.sourceLanguage.intent.operation != code.ServiceCompletion {
			t.Fatal("completion command did not start B1 physical slot")
		}
		completionFinish(t, a, q)
		tester.Frame()
		for _, label := range []string{"源码补全候选", "确认补全", "取消补全"} {
			access, ok := tester.Access(label)
			if !ok || access.Bounds.W <= 0 || access.Bounds.H <= 0 || access.Bounds.X < 0 || access.Bounds.Y < 0 || access.Bounds.X+access.Bounds.W > 760 || access.Bounds.Y+access.Bounds.H > 480 {
				t.Fatalf("accessible completion control %q absent/clipped: %+v", label, access)
			}
		}
		rows := a.sourceLanguageRows()
		if len(rows) < 2 || !strings.Contains(rows[0].label, "本文件词") {
			t.Fatal("real word and keyword vocabulary missing", rows)
		}
		id := completionCandidate(t, a, "val")
		index := -1
		for i, row := range rows {
			if row.id == id {
				index = i
			}
		}
		if index < 0 {
			t.Fatal("real candidate not visible")
		}
		// The published list autofocus is the public handle, retained through
		// semantic resets. Real arrow input chooses only; it must not edit.
		handle := a.sourceLanguage.list.Handle
		for i := 0; i < index; i++ {
			tester.Key(0, ui.KeyDown)
		}
		if tab.Doc.Buffer.String() != source || tab.Doc.Buffer.Version() != before.version {
			t.Fatal("arrow selection edited source")
		}
		if pointer {
			if err := tester.Click("确认补全"); err != nil {
				t.Fatal(err)
			}
		} else {
			tester.Key(0, ui.KeyEnter)
		}
		tester.Frame()
		want := "package p; func f(val int) { _ = val }"
		if tab.Doc.Buffer.String() != want || a.sourceLanguage.visible || a.sourceLanguage.list.Handle != handle {
			t.Fatal("public confirmation did not commit/retire with retained Handle", tab.Doc.Buffer.String())
		}
		if !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != source || !reflect.DeepEqual(tab.Doc.Buffer.SelectionSet(), before.selections) || !tab.Doc.Buffer.Redo() || tab.Doc.Buffer.String() != want {
			t.Fatal("public completion whole Undo/Redo")
		}
		if string(mustReadFile(t, path)) != source || q.maximum.Load() != 1 {
			t.Fatal("completion saved or started another Analyzer")
		}
	}
}

func TestCompletionCancelAndSavedActionCannotConsumeNewPresentation(t *testing.T) {
	t.Run("click", func(t *testing.T) { checkCompletionCancelAndSavedAction(t, false) })
	t.Run("held-press", func(t *testing.T) { checkCompletionCancelAndSavedAction(t, true) })
}

func checkCompletionCancelAndSavedAction(t *testing.T, hold bool) {
	t.Helper()
	const source = "package p; func f(val int) { _ = va }"
	a, tab, q, _ := languageFixture(t, ".go", source)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	at := strings.LastIndex(source, "va") + 2
	completionSelect(t, tab, source, at, at, false)
	completionReady(t, a, q)
	tester.Frame()
	id := completionCandidate(t, a, "val")
	old := sourceLanguageAction{kind: "complete", rowID: id, request: a.sourceLanguage.request, presentation: a.sourceLanguage.presentation}
	handle := a.sourceLanguage.list.Handle
	before := languageState(tab)
	if hold {
		button, ok := tester.AccessByRole(ui.RoleButton, "取消补全")
		if !ok || button.Disabled || button.Bounds.W <= 0 || button.Bounds.H <= 0 {
			t.Fatalf("first completion frame lacks a pointer-reachable Cancel button: %+v", button)
		}
		x, y := button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2
		tester.Press(x, y)
		pressed, ok := tester.AccessByRole(ui.RoleButton, "取消补全")
		if !ok || pressed.Disabled || pressed.Bounds != button.Bounds {
			tester.CancelPress()
			t.Fatalf("Cancel moved or became unavailable while held: before=%+v held=%+v", button, pressed)
		}
		if !a.sourceLanguage.visible || a.sourceLanguage.request != old.request || a.sourceLanguage.presentation != old.presentation || a.sourceLanguage.action != nil {
			tester.CancelPress()
			t.Fatal("pointer down affected completion before release")
		}
		languageUnchanged(t, tab, before)
		tester.Release(x, y)
	} else if err := tester.Click("取消补全"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	languageUnchanged(t, tab, before)
	if a.sourceLanguage.visible || a.sourceLanguage.owner != nil || a.sourceLanguage.intent.completion != nil || len(a.sourceLanguage.result.Completion) != 0 || a.sourceLanguage.list.Handle != handle {
		t.Fatalf("cancel did not release bounded completion state: visible=%t owner=%t intent=%t results=%d handleEqual=%t",
			a.sourceLanguage.visible, a.sourceLanguage.owner != nil, a.sourceLanguage.intent.completion != nil, len(a.sourceLanguage.result.Completion), a.sourceLanguage.list.Handle == handle)
	}
	completionReady(t, a, q)
	tester.Frame()
	request, presentation := a.sourceLanguage.request, a.sourceLanguage.presentation
	a.sourceLanguage.action = &old
	a.processSourceLanguageActions()
	if a.sourceLanguage.request != request || a.sourceLanguage.presentation != presentation || a.sourceLanguage.intent.completion == nil || tab.Doc.Buffer.String() != source {
		t.Fatal("old action consumed newer presentation")
	}
	// Escape on the current list is a completion cancel, not a source edit.
	tester.Key(0, ui.KeyEscape)
	tester.Frame()
	if a.sourceLanguage.visible || tab.Doc.Buffer.String() != source {
		t.Fatal("completion list Escape did not cancel")
	}
}

func TestCompletionRejectsMultipleSelectionsWithoutCollapsingOrAnalysis(t *testing.T) {
	const source = "package p; func f(val int) { _ = va }"
	a, tab, q, _ := languageFixture(t, ".go", source)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	if err := tab.Doc.Buffer.SetSelectionSet(text.SelectionSet{Ranges: []text.Range{{Anchor: 1, Head: 0}, {Anchor: 5, Head: 5}}, Primary: 1}); err != nil {
		t.Fatal(err)
	}
	before := languageState(tab)
	paletteOpen(t, a, tester)
	tester.Type("语言 源码补全")
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	if !strings.Contains(a.Message, "仅支持一个选择") || a.sourceLanguage.task != nil || q.outstanding.Load() != 0 {
		t.Fatal("multi-selection refusal not visible or analyzed", a.Message)
	}
	languageUnchanged(t, tab, before)
}

func TestCompletionNativeAndForeignOwnerAdmissionRetainsOriginalInput(t *testing.T) {
	const source = "package p; func f(val int) { _ = va }"
	for _, kind := range []string{"native-marked", "markless-single", "markless-multi", "foreign-ordinary", "foreign-composition"} {
		t.Run(kind, func(t *testing.T) {
			a, tab, q, _ := languageFixture(t, ".go", source)
			if strings.HasPrefix(kind, "markless") {
				b := text.NewWithLimit(source, len(source))
				tab.Doc.Buffer = b
				tab.Editor.SetBuffer(b)
			}
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			at := strings.LastIndex(source, "va") + 2
			completionSelect(t, tab, source, at, at, false)
			completionReady(t, a, q)
			tester.Frame()
			id := completionCandidate(t, a, "val")
			// The multi case must first move focus from the completed candidate
			// list to the installed source input. Queue its old confirm only
			// after that focus frame, so the frame cannot apply it early.
			if kind != "markless-multi" {
				a.queueSourceLanguageAction("complete", id)
			}
			request, presentation, handle := a.sourceLanguage.request, a.sourceLanguage.presentation, a.sourceLanguage.list.Handle
			b := tab.Doc.Buffer
			var token text.CompositionToken
			var checkMultiInput func(bool)
			switch kind {
			case "foreign-ordinary":
				if b.BeginGroup() != nil {
					t.Fatal("foreign ordinary")
				}
			case "foreign-composition":
				var err error
				token, err = b.BeginCompositionGroup()
				if err != nil {
					t.Fatal(err)
				}
			case "markless-multi":
				// Focus is a public Editor operation on the existing Tester tree;
				// unlike a text click, it does not change the admitted selector.
				tab.Editor.Focus()
				tester.Frame()
				if !tester.Focused(tab.Editor.Label) {
					t.Fatal("multi refusal fixture did not focus installed source input")
				}
				if active, noIME := tester.TextInputActive(); !active || noIME {
					t.Fatal("multi refusal fixture has no active native text client")
				}
				if a.sourceLanguage.request != request || a.sourceLanguage.presentation != presentation || a.sourceLanguage.list.Handle != handle || b.String() != source {
					t.Fatal("focus frame changed the original completion target")
				}
				a.queueSourceLanguageAction("complete", id)
				if a.sourceLanguage.action == nil {
					t.Fatal("multi refusal fixture did not queue the original completion before native input")
				}
				// Independent literal UTF-16 oracle: both ranges are backward,
				// the second is primary and selects the two ASCII bytes "va".
				wantSelections := text.SelectionSet{Ranges: []text.Range{{Anchor: 1, Head: 0}, {Anchor: 35, Head: 33}}, Primary: 1}
				if at != 35 || len(source) != 37 || source[33:35] != "va" || len("🙂") != 4 || b.MaxBytes() != len(source) {
					t.Fatal("oversized multi-preedit literal/budget oracle is wrong")
				}
				if err := b.SetSelectionSet(wantSelections); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(b.SelectionSet(), wantSelections) || b.Selection() != (text.Range{Anchor: 35, Head: 33}) || tab.Editor.InputActionError() != nil {
					t.Fatal("literal full directed multi-selection was not installed while idle")
				}
				original := languageState(tab)
				queueMaximum := q.maximum.Load()
				checkMultiInput = func(pending bool) {
					t.Helper()
					if b.String() != source || b.Version() != original.version || b.History() != original.history || !reflect.DeepEqual(b.SelectionSet(), wantSelections) || b.Selection() != (text.Range{Anchor: 35, Head: 33}) {
						t.Fatal("native multi refusal/cancel changed bytes, version, history or full directed selections")
					}
					if _, marked := tab.Editor.MarkedRange(); marked {
						t.Fatal("first-refused multi input gained a native mark")
					}
					if !tester.Focused(tab.Editor.Label) || a.sourceLanguage.task != nil || q.outstanding.Load() != 0 || q.maximum.Load() != queueMaximum {
						t.Fatal("refused completion detached focus or started another Analyzer")
					}
					if pending {
						if !tab.Editor.MultiCompositionPending() || !tab.Editor.CompositionActive() || !b.InGroup() || !errors.Is(tab.Editor.InputActionError(), text.ErrCompositionPending) || b.SelectionGeneration() != original.selectionGeneration {
							t.Fatal("native transport did not retain the original markless multi owner/selection generation")
						}
					} else {
						if tab.Editor.MultiCompositionPending() || tab.Editor.CompositionActive() || b.InGroup() || tab.Editor.InputActionError() != nil || b.CanUndo() != original.canUndo || b.CanRedo() != original.canRedo {
							t.Fatal("public Escape did not clear the original multi owner while preserving history")
						}
					}
				}
				// Emoji is four bytes versus the two-byte primary and the source
				// is already at its ceiling. The actual installed native client
				// begins/retains multi ownership before refusing the first preedit.
				tester.Compose("🙂", 1)
				checkMultiInput(true)
			default:
				tab.Editor.SetMarkedText(nil, "🙂", ui.TextInputRange{})
			}
			if tab.Editor.InputActionError() == nil {
				t.Fatal("fixture lacks actual retained input/group")
			}
			if strings.HasPrefix(kind, "markless") {
				if _, marked := tab.Editor.MarkedRange(); marked {
					t.Fatal("markless fixture gained mark")
				}
			}
			before := languageState(tab)
			a.processSourceLanguageActions()
			if a.sourceLanguage.action != nil || a.sourceLanguage.request != request || a.sourceLanguage.presentation != presentation || a.sourceLanguage.list.Handle != handle || !a.sourceLanguage.visible {
				t.Fatal("refused old completion action retired input/presentation")
			}
			if err := a.requestSourceCompletion(); err == nil || a.sourceLanguage.task != nil {
				t.Fatal("request ignored pending native owner")
			}
			a.queueSourceLanguageAction("completion-cancel", "")
			a.processSourceLanguageActions()
			languageUnchanged(t, tab, before)
			if checkMultiInput != nil {
				checkMultiInput(true)
			}
			switch kind {
			case "foreign-ordinary":
				if b.ReplaceUTF16(0, 1, "P") != nil || !b.CancelGroup() || b.String() != source {
					t.Fatal("foreign group continuation lost")
				}
			case "foreign-composition":
				if !b.OwnsComposition(token) || b.ReplaceCompositionUTF16(token, 0, 1, "P") != nil || b.CancelCompositionGroup(token) != nil || b.String() != source {
					t.Fatal("foreign token continuation lost")
				}
			default:
				if kind == "markless-multi" {
					tester.Key(0, ui.KeyEscape)
					checkMultiInput(false)
				} else {
					tab.Editor.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape})
				}
				if tab.Editor.InputActionError() != nil || b.String() != source {
					t.Fatal("original native Escape lost")
				}
			}
			tester.Frame()
			if b.String() != source || a.sourceLanguage.intent.completion != nil {
				t.Fatal("refused action revived after native cancellation")
			}
		})
	}
}

func TestCompletionEqualCandidateStillRefusesReadonlyOrForeignInput(t *testing.T) {
	const source = "package p; func f(val int) { _ = val }"
	for _, readonly := range []bool{false, true} {
		a, tab, q, _ := languageFixture(t, ".go", source)
		at := strings.LastIndex(source, "val")
		completionSelect(t, tab, source, at, at+3, true)
		completionReady(t, a, q)
		id := completionCandidate(t, a, "val")
		if readonly {
			tab.Doc.ReadOnly = true
		} else if tab.Doc.Buffer.BeginGroup() != nil {
			t.Fatal("group fixture")
		}
		before := languageState(tab)
		a.queueSourceLanguageAction("complete", id)
		a.processSourceLanguageActions()
		languageUnchanged(t, tab, before)
		if !readonly && !tab.Doc.Buffer.CancelGroup() {
			t.Fatal("equal candidate stole group")
		}
	}
}

func TestCompletionOnChangeNewRequestPreservesCommittedBytesAndCallbackExit(t *testing.T) {
	const source = "package p; func f(val int) { _ = va }"
	for _, replaceTab := range []bool{false, true} {
		a, tab, q, _ := languageFixture(t, ".go", source)
		at := strings.LastIndex(source, "va") + 2
		completionSelect(t, tab, source, at, at, false)
		completionReady(t, a, q)
		id := completionCandidate(t, a, "val")
		original := tab.Editor.OnChange
		calls := 0
		var expectedCurrent *Tab
		var expectedRequest, expectedPresentation uint64
		var expectedTask *sourceLanguageTask
		var expectedOwner *sourceLanguageOwner
		tab.Editor.OnChange = func() {
			calls++
			if original != nil {
				original()
			}
			if replaceTab {
				a.NewDocument("text")
			} else {
				a.runSourceLanguageCommand("symbols")
			}
			a.Message = "callback exit stays"
			a.sourceLanguage.message = "new request exit stays"
			expectedCurrent = a.Current()
			expectedRequest = a.sourceLanguage.request
			expectedPresentation = a.sourceLanguage.presentation
			expectedTask = a.sourceLanguage.task
			expectedOwner = a.sourceLanguage.owner
		}
		a.queueSourceLanguageAction("complete", id)
		a.processSourceLanguageActions()
		want := "package p; func f(val int) { _ = val }"
		if calls != 1 || tab.Doc.Buffer.String() != want || a.Current() != expectedCurrent || a.sourceLanguage.request != expectedRequest || a.sourceLanguage.presentation != expectedPresentation || a.sourceLanguage.task != expectedTask || a.sourceLanguage.owner != expectedOwner || a.Message != "callback exit stays" || a.sourceLanguage.message != "new request exit stays" {
			t.Fatal("completion touched callback replacement exit state", calls, a.Message)
		}
		if expectedTask != nil {
			languageWait(t, expectedTask.done)
			languageReceive(t, q.calls)()
			if a.sourceLanguage.result.Status != code.ServiceComplete || len(a.sourceLanguage.result.Symbols) == 0 {
				t.Fatal("new real request was canceled/cleared")
			}
		}
	}
}

func TestCompletionStalePresentationCallbacksNeverApplyAfterNewRequest(t *testing.T) {
	const source = "package p; func f(val int) { _ = va }"
	a, tab, q, _ := languageFixture(t, ".go", source)
	at := strings.LastIndex(source, "va") + 2
	completionSelect(t, tab, source, at, at, false)
	completionReady(t, a, q)
	id := completionCandidate(t, a, "val")
	a.queueSourceLanguageAction("complete", id)
	old := *a.sourceLanguage.action
	if err := a.requestSourceLanguage(languageIntent(code.ServiceSymbols)); err != nil {
		t.Fatal(err)
	}
	task := a.sourceLanguage.task
	request := a.sourceLanguage.request
	a.sourceLanguage.action = &old
	a.processSourceLanguageActions()
	if tab.Doc.Buffer.String() != source || a.sourceLanguage.request != request || a.sourceLanguage.task != task {
		t.Fatal("saved completion action edited/retired successor")
	}
	languageWait(t, task.done)
	languageReceive(t, q.calls)()
	if len(a.sourceLanguage.result.Symbols) == 0 {
		t.Fatal("successor real symbols lost")
	}
}

func TestCompletionHostFailuresLimitsCancellationRetireApplicablePayload(t *testing.T) {
	const source = "package p; func f(val int) { _ = va }"
	for _, kind := range []string{"host-error", "malformed", "source-budget", "canceled", "invalid-dispatch"} {
		a, tab, q, _ := languageFixture(t, ".go", source)
		at := strings.LastIndex(source, "va") + 2
		completionSelect(t, tab, source, at, at, false)
		completionReady(t, a, q)
		switch kind {
		case "host-error":
			a.sourceLanguageHooks.fail = errors.New("real worker controlled failure")
		case "malformed":
			a.sourceLanguageHooks.malformed = true
		case "source-budget":
			large := text.New("package p\n// " + strings.Repeat("x", 2<<20))
			tab.Doc.Buffer = large
			tab.Editor.SetBuffer(large)
		case "invalid-dispatch":
			a.sourceLanguageHooks.dispatch = nil
			a.Window = nil
		case "canceled":
			started, release := languageHold(t, a)
			if err := a.requestSourceCompletion(); err != nil {
				t.Fatal(err)
			}
			task := a.sourceLanguage.task
			languageReceive(t, started)
			a.cancelSourceLanguage()
			release()
			languageWait(t, task.done)
			languageReceive(t, q.calls)()
		}
		if kind != "canceled" {
			err := a.requestSourceCompletion()
			if kind == "source-budget" || kind == "invalid-dispatch" {
				if err == nil {
					t.Fatal("admission failure not returned")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				completionFinish(t, a, q)
			}
		}
		if a.sourceLanguage.intent.completion != nil || len(a.sourceLanguage.result.Completion) != 0 || a.sourceLanguage.owner != nil {
			t.Fatal("failure retained applicable completion", kind)
		}
	}
}

func TestCompletionQueuedIntentCannotFollowLaterCaretBeforeLaunch(t *testing.T) {
	const source = "package p; func f(val int) { _ = va }"
	a, tab, q, _ := languageFixture(t, ".go", source)
	at := strings.LastIndex(source, "va") + 2
	completionSelect(t, tab, source, at, at, false)
	started, release := languageHold(t, a)
	first := languageRequest(t, a, languageIntent(code.ServiceDiagnostics))
	languageReceive(t, started)
	if err := a.requestSourceCompletion(); err != nil {
		t.Fatal(err)
	}
	pending := a.sourceLanguage.pending
	if pending == nil || pending.intent.completion == nil || pending.intent.completion.oldText != "va" {
		t.Fatal("missing bounded queued completion")
	}
	_ = tab.Doc.Buffer.SetSelection(0, 0)
	_ = tab.Doc.Buffer.SetSelection(completionUnits(source[:at]), completionUnits(source[:at]))
	before := languageState(tab)
	release()
	languageWait(t, first.done)
	languageReceive(t, q.calls)()
	languageUnchanged(t, tab, before)
	if a.sourceLanguage.task != nil || a.sourceLanguage.pending != nil || a.sourceLanguage.intent.completion != nil || a.sourceLanguage.owner != nil {
		t.Fatal("queued edit rebound new target after caret ABA")
	}
}

func TestCompletionPublicationDuringNativePreeditDoesNotStealInput(t *testing.T) {
	const source = "package p; func f(val int) { _ = va }"
	a, tab, q, _ := languageFixture(t, ".go", source)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	at := strings.LastIndex(source, "va") + 2
	completionSelect(t, tab, source, at, at, false)
	started, release := languageHold(t, a)
	if err := a.requestSourceCompletion(); err != nil {
		t.Fatal(err)
	}
	task := a.sourceLanguage.task
	languageReceive(t, started)
	tab.Editor.SetMarkedText(nil, "🙂", ui.TextInputRange{})
	before := languageState(tab)
	if tab.Editor.InputActionError() == nil {
		t.Fatal("fixture not actual native composition")
	}
	release()
	languageWait(t, task.done)
	languageReceive(t, q.calls)()
	languageUnchanged(t, tab, before)
	if a.sourceLanguage.intent.completion != nil || len(a.sourceLanguage.result.Completion) != 0 || a.sourceLanguage.owner != nil {
		t.Fatal("native priority lost at publication")
	}
	tab.Editor.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape})
	if tab.Editor.InputActionError() != nil || tab.Doc.Buffer.String() != source {
		t.Fatal("original native owner lost Escape")
	}
}

func TestCompletionRealNativeTransportAdmissionPrecedesFocusAndCancelEffects(t *testing.T) {
	const source = "package p; func f(val int) { _ = va }"
	for _, refused := range []bool{false, true} {
		a, tab, q, _ := languageFixture(t, ".go", source)
		if refused {
			b := text.NewWithLimit(source, len(source))
			tab.Doc.Buffer = b
			tab.Editor.SetBuffer(b)
		}
		tester := ui.NewTester(a.View, 760, 480)
		tester.Frame()
		if err := tester.Click(tab.Editor.Label); err != nil {
			t.Fatal(err)
		}
		at := strings.LastIndex(source, "va") + 2
		completionSelect(t, tab, source, at, at, false)
		tester.Compose("🙂", 1)
		if tab.Editor.InputActionError() == nil {
			t.Fatal("real native transport did not retain composition/refusal")
		}
		before := languageState(tab)
		request, presentation, handle := a.sourceLanguage.request, a.sourceLanguage.presentation, a.sourceLanguage.list.Handle
		a.runSourceCompletionCommand()
		if a.sourceLanguage.request != request || a.sourceLanguage.presentation != presentation || a.sourceLanguage.list.Handle != handle || a.sourceLanguage.task != nil || q.outstanding.Load() != 0 || !tester.Focused(tab.Editor.Label) {
			t.Fatal("completion command crossed native pure gate")
		}
		languageUnchanged(t, tab, before)
		tester.Key(0, ui.KeyEscape)
		if tab.Editor.InputActionError() != nil || tab.Doc.Buffer.String() != source || !tester.Focused(tab.Editor.Label) {
			t.Fatal("native Escape path was detached")
		}
	}
}

func TestCompletionLargeVocabularyLabelsAndVisibleRowBudgetRemainTruthful(t *testing.T) {
	var source strings.Builder
	source.WriteString(`{"dst":"old"`)
	for i := 0; i < 511; i++ {
		fmt.Fprintf(&source, `,"k%03d":%d`, i, i)
	}
	source.WriteByte('}')
	value := source.String()
	a, tab, q, path := languageFixture(t, ".json", value)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	at := strings.Index(value, `"old"`)
	completionSelect(t, tab, value, at, at+len(`"old"`), false)
	before := languageState(tab)
	completionReady(t, a, q)
	tester.Frame()
	rows := a.sourceLanguageRows()
	if len(a.sourceLanguage.result.Completion) != code.DefaultServiceLimits().Completion || len(rows) != code.DefaultServiceLimits().Completion {
		t.Fatal("real JSON completion rows were truncated", len(rows), len(a.sourceLanguage.result.Completion))
	}
	candidate := `"k510"`
	index := -1
	for i, row := range rows {
		if row.label != "" && strings.HasPrefix(row.label, candidate+" · ") {
			index = i
			break
		}
	}
	if index <= 255 {
		t.Fatal("post-256 JSON candidate was not in the bounded List", index)
	}
	for i := 0; i < index; i++ {
		tester.Key(0, ui.KeyDown)
	}
	if a.sourceLanguage.selected != index {
		t.Fatal("real List keyboard did not select post-256 candidate", a.sourceLanguage.selected, index)
	}
	calls := 0
	original := tab.Editor.OnChange
	tab.Editor.OnChange = func() {
		calls++
		if original != nil {
			original()
		}
	}
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	want := value[:at] + candidate + value[at+len(`"old"`):]
	if tab.Doc.Buffer.String() != want || !json.Valid([]byte(want)) || calls != 1 || a.sourceLanguage.visible || tab.Doc.Buffer.History().UndoDepth != before.history.UndoDepth+1 {
		t.Fatal("post-256 JSON candidate did not commit once", tab.Doc.Buffer.String(), calls)
	}
	caret := completionUnits(value[:at] + candidate)
	if !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != value || !reflect.DeepEqual(tab.Doc.Buffer.SelectionSet(), before.selections) || !tab.Doc.Buffer.Redo() || tab.Doc.Buffer.String() != want || tab.Doc.Buffer.Selection() != (text.Range{Anchor: caret, Head: caret}) {
		t.Fatal("post-256 JSON candidate Undo/Redo")
	}
	if string(mustReadFile(t, path)) != value {
		t.Fatal("post-256 JSON completion implicitly saved")
	}
	for _, row := range rows {
		if len(row.label) > sourceLanguageLabelBytes || row.kind != "completion" {
			t.Fatal("unbounded or fabricated candidate label")
		}
	}

	var over strings.Builder
	over.WriteString(`{"dst":"old"`)
	for i := 0; i < 512; i++ {
		fmt.Fprintf(&over, `,"q%03d":%d`, i, i)
	}
	over.WriteByte('}')
	a, tab, q, _ = languageFixture(t, ".json", over.String())
	at = strings.Index(over.String(), `"old"`)
	completionSelect(t, tab, over.String(), at, at+len(`"old"`), false)
	before = languageState(tab)
	if err := a.requestSourceCompletion(); err != nil {
		t.Fatal(err)
	}
	completionFinish(t, a, q)
	if a.sourceLanguage.status != string(code.ServiceLimit) || len(a.sourceLanguage.result.Completion) != 0 || a.sourceLanguage.intent.completion != nil || len(a.sourceLanguageRows()) != 0 {
		t.Fatal("over-512 service result exposed a partial payload", a.sourceLanguage.result)
	}
	languageUnchanged(t, tab, before)

	// A single >512-byte actual word still has a bounded label and exact raw
	// insertion text; a label is never used as the replacement source.
	long := strings.Repeat("a", 600)
	value = "package p; var " + long + " int; func f(){ _ = a }"
	a, tab, q, _ = languageFixture(t, ".go", value)
	at = strings.LastIndex(value, "a") + 1
	completionSelect(t, tab, value, at, at, false)
	completionReady(t, a, q)
	id := completionCandidate(t, a, long)
	for _, row := range a.sourceLanguageRows() {
		if row.id == id && len(row.label) > sourceLanguageLabelBytes {
			t.Fatal("long label limit")
		}
	}
	completionConfirm(t, a, long)
	want = value[:at-1] + long + value[at:]
	if tab.Doc.Buffer.String() != want || !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != value {
		t.Fatal("label truncation leaked into exact completion patch")
	}
}

func TestCompletionOtherNativeFieldsRefuseQueuedConfirmAndCancel(t *testing.T) {
	const source = "package p; func f(val int) { _ = va }"
	for _, field := range []string{"palette", "picker", "find", "replacement", "line"} {
		a, tab, q, _ := languageFixture(t, ".go", source)
		at := strings.LastIndex(source, "va") + 2
		completionSelect(t, tab, source, at, at, false)
		completionReady(t, a, q)
		id := completionCandidate(t, a, "val")
		a.queueSourceLanguageAction("complete", id)
		request, presentation := a.sourceLanguage.request, a.sourceLanguage.presentation
		switch field {
		case "palette":
			a.commandPalette.composing = true
		case "picker":
			a.openDocuments.composing = true
		case "find":
			a.FindQueryComposing = true
		case "replacement":
			a.FindReplacementComposing = true
		case "line":
			a.GoToLineComposing = true
		}
		before := languageState(tab)
		a.processSourceLanguageActions()
		a.queueSourceLanguageAction("completion-cancel", "")
		a.processSourceLanguageActions()
		if err := a.requestSourceCompletion(); err == nil {
			t.Fatal("completion admitted another native field owner", field)
		}
		languageUnchanged(t, tab, before)
		if a.sourceLanguage.request != request || a.sourceLanguage.presentation != presentation || a.sourceLanguage.action != nil || a.sourceLanguage.intent.completion == nil {
			t.Fatal("field owner refusal retired or queued completion", field)
		}
	}
}

func TestCompletionIdleFieldFocusIntentWinsDelayedAndPreBuildPublication(t *testing.T) {
	for _, phase := range []string{"before-publication", "after-publication-before-list-build"} {
		for _, field := range []string{"find", "line"} {
			t.Run(phase+"/"+field, func(t *testing.T) {
				const source = "package p; func f(val int) { _ = va }"
				a, tab, q, _ := languageFixture(t, ".go", source)
				tester := ui.NewTester(a.View, 760, 480)
				tester.Frame()
				at := strings.LastIndex(source, "va") + 2
				completionSelect(t, tab, source, at, at, false)
				before := languageState(tab)
				started, release := languageHold(t, a)
				released := false
				t.Cleanup(func() {
					if !released {
						release()
					}
				})
				if err := a.requestSourceCompletion(); err != nil {
					t.Fatal(err)
				}
				if err := a.openSourceLanguagePresentation(); err != nil {
					t.Fatal(err)
				}
				task := a.sourceLanguage.task
				languageReceive(t, started)
				tester.Frame()
				openIdleField := func() string {
					label, initialText := "查找内容", "x"
					if field == "find" {
						if !a.openFind() {
							t.Fatal("Find focus request refused")
						}
					} else {
						label, initialText = "转到行号", "1"
						if !a.openGoToLine() {
							t.Fatal("GoToLine focus request refused")
						}
					}
					tester.Frame()
					if !tester.Focused(label) {
						t.Fatalf("actual field focus was not established: %s", label)
					}
					tester.Type(initialText)
					tester.Frame()
					if !tester.Focused(label) {
						t.Fatalf("idle ASCII input lost field focus: %s", label)
					}
					return label
				}
				if phase == "before-publication" {
					openIdleField()
				}
				release()
				released = true
				languageWait(t, task.done)
				languageReceive(t, q.calls)()
				if a.sourceLanguage.result.Status != code.ServiceComplete || len(a.sourceLanguage.result.Completion) == 0 {
					t.Fatal("real completion result was lost after focus transfer")
				}
				label := "查找内容"
				if field == "line" {
					label = "转到行号"
				}
				if phase == "after-publication-before-list-build" {
					openIdleField()
				} else {
					tester.Frame()
				}
				tester.Frame()
				if !tester.Focused(label) {
					t.Fatalf("delayed completion reclaimed idle field focus: %s", label)
				}
				nextText, expected := "y", "xy"
				if field == "line" {
					nextText, expected = "2", "12"
				}
				tester.Type(nextText)
				tester.Frame()
				if !tester.Focused(label) {
					t.Fatalf("later ASCII input did not remain in the field: %s", label)
				}
				if field == "find" && a.Query != expected || field == "line" && a.GoToLineQuery != expected {
					t.Fatalf("input was not retained by its focused field: find=%q line=%q", a.Query, a.GoToLineQuery)
				}
				languageUnchanged(t, tab, before)
				if completionCandidate(t, a, "val") == "" {
					t.Fatal("completion candidate is unavailable for explicit selection")
				}
				if a.sourceLanguage.completionFocus {
					t.Fatal("stale List focus request remained pending")
				}
			})
		}
	}
}

func TestCompletionPointerDoubleClickAndStaleRepeatSubmitOnce(t *testing.T) {
	const source = "package p; func f(val int) { _ = va }"
	a, tab, q, path := languageFixture(t, ".go", source)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	at := strings.LastIndex(source, "va") + 2
	completionSelect(t, tab, source, at, at, false)
	before := languageState(tab)
	completionReady(t, a, q)
	tester.Frame()
	id := completionCandidate(t, a, "val")
	rows := a.sourceLanguageRows()
	index := -1
	var label string
	for i, row := range rows {
		if row.id == id {
			index, label = i, row.label
			break
		}
	}
	if index < 0 {
		t.Fatal("real candidate row not found")
	}
	for i := 0; i < index; i++ {
		tester.Key(0, ui.KeyDown)
	}
	tester.Frame()
	access, ok := tester.Access("补全候选：" + label)
	if !ok || access.Bounds.W <= 0 || access.Bounds.H <= 0 {
		t.Fatalf("actual candidate row is not pointer reachable: %+v", access)
	}
	changes := 0
	original := tab.Editor.OnChange
	tab.Editor.OnChange = func() {
		changes++
		if original != nil {
			original()
		}
	}
	x, y := access.Bounds.X+access.Bounds.W/2, access.Bounds.Y+access.Bounds.H/2
	tester.ClickAt(x, y)
	tester.ClickAt(x, y)
	tester.Frame()
	want := "package p; func f(val int) { _ = val }"
	if tab.Doc.Buffer.String() != want || changes != 1 || a.sourceLanguage.visible {
		t.Fatalf("double-click confirmation did not apply exactly once: source=%q changes=%d visible=%v", tab.Doc.Buffer.String(), changes, a.sourceLanguage.visible)
	}
	if !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != source || !reflect.DeepEqual(tab.Doc.Buffer.SelectionSet(), before.selections) || !tab.Doc.Buffer.Redo() || tab.Doc.Buffer.String() != want {
		t.Fatal("double-click did not remain one exact undoable patch")
	}
	// These repeated coordinates refer to the retired row/presentation.
	tester.ClickAt(x, y)
	tester.ClickAt(x, y)
	tester.Frame()
	if tab.Doc.Buffer.String() != want || changes != 1 {
		t.Fatalf("stale pointer repeats submitted another edit: source=%q changes=%d", tab.Doc.Buffer.String(), changes)
	}
	if string(mustReadFile(t, path)) != source {
		t.Fatal("pointer completion implicitly saved")
	}
}

// No successful payload hook is used anywhere: scheduling barriers only control
// timing; every candidate above comes from real AnalyzeService.
