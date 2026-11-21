package app

import (
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
)

func TestCompletionAlreadyOpenFindFocusIntentSurvivesResolution(t *testing.T) {
	const source = "package p; func f(val int) { _ = va }"
	const queryLabel, replacementLabel = "查找内容", "替换内容"
	const candidateLabel = "补全候选：val · 本文件词"
	for _, tc := range []struct {
		name, intent string
		afterPublish bool
	}{
		{"before-publication/query", "query", false},
		{"after-publication/query", "query", true},
		{"before-publication/replacement", "replacement", false},
		{"after-publication/replacement", "replacement", true},
		{"no-request", "", false},
		{"stale-generation", "stale-generation", false},
		{"stale-target", "stale-target", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, tab, q, path := languageFixture(t, ".go", source)
			tester := ui.NewTester(a.View, 1180, 780)
			tester.Frame()
			tester.Key(ui.Cmd, ui.KeyF)
			if !a.ShowFind || !tester.Focused(queryLabel) || a.findFocus != findFocusNone {
				t.Fatal("initial Ctrl+F did not resolve real query focus")
			}
			if tc.intent == "replacement" {
				// Enter must exercise the actual replacement key path without a
				// match, navigation, or a source edit that would stale completion.
				tester.Type("zz_no_such_token")
				if a.Query != "zz_no_such_token" || strings.Contains(source, a.Query) {
					t.Fatalf("no-match replacement query was not entered: %q", a.Query)
				}
			}
			generation, originalQuery := a.findGeneration, a.Query
			b := tab.Doc.Buffer
			start := strings.LastIndex(source, "va")
			completionSelect(t, tab, source, start, start+2, true)
			tab.Editor.Focus() // no pointer caret change, no Find close
			tester.Frame()
			if !a.ShowFind || a.findGeneration != generation || !tester.Focused(tab.Editor.Label) {
				t.Fatal("source did not recapture focus while Find stayed open")
			}
			if active, noIME := tester.TextInputActive(); !active || noIME || tab.Editor.InputActionError() != nil {
				t.Fatal("admission fixture lacks idle native source input")
			}
			before := languageState(tab)
			changes := 0
			onChange := tab.Editor.OnChange
			tab.Editor.OnChange = func() {
				changes++
				if onChange != nil {
					onChange()
				}
			}
			unchanged := func() {
				t.Helper()
				languageUnchanged(t, tab, before)
				if changes != 0 || string(mustReadFile(t, path)) != source || tab.Editor.InputActionError() != nil || a.findCompositionActive() {
					t.Fatal("focus/typing changed source OnChange, disk, or idle IME state")
				}
			}

			started, release := languageHold(t, a)
			if err := a.requestSourceCompletion(); err != nil {
				t.Fatal(err)
			}
			task := a.sourceLanguage.task
			if task == nil {
				t.Fatal("missing physical analyzer task")
			}
			languageReceive(t, started)
			publish := func() {
				t.Helper()
				release()
				languageWait(t, task.done)
				languageReceive(t, q.calls)()
				if a.sourceLanguage.result.Status != code.ServiceComplete || len(a.sourceLanguage.result.Completion) == 0 {
					t.Fatalf("real completion did not publish: status=%s reason=%s failure=%v", a.sourceLanguage.status, a.sourceLanguage.reason, a.sourceLanguage.failure)
				}
			}
			if tc.afterPublish {
				publish()
				if !a.sourceLanguage.completionFocus {
					t.Fatal("publication without a later intent did not retain initial List autofocus")
				}
			}

			// Mount the completion presentation only after the following actual
			// Find frames: in the after-publication case this isolates the second
			// gate and avoids a still-pending Find request masking the ABA.
			field := ""
			switch tc.intent {
			case "query":
				tester.Key(ui.Cmd, ui.KeyF)
				field = queryLabel
			case "replacement":
				if err := tester.Click(replacementLabel); err != nil {
					t.Fatal(err)
				}
				tester.Key(0, ui.KeyEnter)
				field = replacementLabel
			case "stale-generation":
				a.requestFindFocusAt(generation-1, findFocusQuery, tab, tab.Doc, tab.Editor, b)
			case "stale-target":
				a.requestFindFocusAt(generation, findFocusReplacement, tab, tab.Doc, tab.Editor, nil)
			}
			tester.Frame()
			if !a.ShowFind || a.findGeneration != generation || a.findFocus != findFocusNone {
				t.Fatal("fixture did not leave the original Find generation fully resolved")
			}
			if field != "" && !tester.Focused(field) || field == "" && !tester.Focused(tab.Editor.Label) {
				t.Fatalf("unexpected actual focus before first completion List build: field=%q", field)
			}
			unchanged()
			if !tc.afterPublish {
				publish()
			}
			publicationWantsFocus := a.sourceLanguage.completionFocus
			if err := a.openSourceLanguagePresentation(); err != nil {
				t.Fatal(err)
			}
			tester.Frame()

			if field != "" {
				if !tester.Focused(field) {
					t.Fatalf("completion List stole the resolved same-generation Find focus from %q", field)
				}
				if active, noIME := tester.TextInputActive(); !active || noIME {
					t.Fatal("focused Find field lost native text input")
				}
				tester.Type("typed")
				tester.Frame()
				wantQuery, wantReplacement := originalQuery, ""
				if field == queryLabel {
					wantQuery = "typed"
				} else {
					wantReplacement = "typed"
				}
				if !tester.Focused(field) || a.Query != wantQuery || a.Replacement != wantReplacement {
					t.Fatalf("typed input did not stay in %q: query=%q replacement=%q", field, a.Query, a.Replacement)
				}
				unchanged()
				if !tc.afterPublish && publicationWantsFocus {
					t.Fatal("publication granted autofocus despite the resolved Find intent")
				}
				// Explicitly select the retained, non-default analyzer row through
				// real pointer input, then use the existing confirmation control.
				if err := tester.Click(candidateLabel); err != nil {
					t.Fatal(err)
				}
				unchanged()
				if err := tester.Click("确认补全"); err != nil {
					t.Fatal(err)
				}
			} else {
				// In the no-intent and rejected-intent controls, neither input
				// field can handle Down/Enter as a completion. Real List keyboard
				// navigation must reach the non-default val row and apply it.
				if tester.Focused(tab.Editor.Label) || tester.Focused(queryLabel) || tester.Focused(replacementLabel) {
					t.Fatal("without a newer valid Find intent the completion List did not autofocus")
				}
				tester.Key(0, ui.KeyDown)
				unchanged()
				tester.Key(0, ui.KeyEnter)
			}
			tester.Frame()
			want := source[:start] + "val" + source[start+2:]
			caret := completionUnits(source[:start] + "val")
			if b.String() != want || changes != 1 || !a.ShowFind || a.sourceLanguage.visible || b.History().UndoDepth != before.history.UndoDepth+1 ||
				!reflect.DeepEqual(b.SelectionSet(), text.SelectionSet{Ranges: []text.Range{{Anchor: caret, Head: caret}}, Primary: 0}) {
				t.Fatalf("explicit choice of retained val row did not commit once: source=%q changes=%d", b.String(), changes)
			}
			if !b.Undo() || b.String() != source || !reflect.DeepEqual(b.SelectionSet(), before.selections) || string(mustReadFile(t, path)) != source {
				t.Fatal("explicit completion lost exact source/selection Undo or saved to disk")
			}
		})
	}
}
