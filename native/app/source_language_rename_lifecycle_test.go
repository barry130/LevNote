package app

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
	"levnote.local/native/ui/editor"
)

const renameLifecycleSource = "package p; func f(x int){ _=x }"

func TestRenameSourceIdentityABAAndMetadataNeverRepairOriginal(t *testing.T) {
	for _, kind := range []string{"selection", "selection-ABA", "secondary-ABA", "direction-ABA", "primary-ABA", "text-undo-journal", "buffer-ABA", "editor", "readonly-ABA", "document-readonly", "route", "mode-ABA", "session", "activation", "encoding", "bom", "path", "name", "new-tab"} {
		t.Run(kind, func(t *testing.T) {
			a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
			at := strings.Index(renameLifecycleSource, "x int")
			u := renameUnits(renameLifecycleSource[:at])
			renamePosition(t, tab, renameLifecycleSource, at)
			if kind == "direction-ABA" {
				if err := tab.Doc.Buffer.SetSelection(u+1, u); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "primary-ABA" {
				if err := tab.Doc.Buffer.SetSelectionSet(text.SelectionSet{Ranges: []text.Range{{Anchor: 0, Head: 0}, {Anchor: u, Head: u}}, Primary: 1}); err != nil {
					t.Fatal(err)
				}
			}
			renameReady(t, a, q, "next")
			a.queueSourceLanguageAction("rename-confirm", "")
			old := *a.sourceLanguage.action
			originalField := a.sourceLanguage.rename.field
			b := tab.Doc.Buffer
			set := b.SelectionSet()
			switch kind {
			case "selection":
				_ = b.SetSelection(0, 0)
			case "selection-ABA":
				_ = b.SetSelection(0, 0)
				_ = b.SetSelection(u, u)
			case "secondary-ABA":
				_ = b.SetSelectionSet(text.SelectionSet{Ranges: []text.Range{{Anchor: 0, Head: 0}, {Anchor: u, Head: u}}, Primary: 1})
				_ = b.SetSelection(u, u)
			case "direction-ABA":
				_ = b.SetSelection(u, u+1)
				_ = b.SetSelection(u+1, u)
			case "primary-ABA":
				other := set
				other.Primary = 0
				_ = b.SetSelectionSet(other)
				_ = b.SetSelectionSet(set)
			case "text-undo-journal":
				for i := 0; i < 40; i++ {
					if b.ReplaceUTF16(0, 0, " ") != nil || !b.Undo() {
						t.Fatal("journal ABA fixture")
					}
				}
			case "buffer-ABA":
				tab.Editor.SetBuffer(text.New(renameLifecycleSource))
				tab.Editor.SetBuffer(b)
			case "editor":
				tab.Editor = editor.New(b)
			case "readonly-ABA":
				tab.Editor.SetReadOnly(true)
				tab.Editor.SetReadOnly(false)
			case "document-readonly":
				tab.Doc.ReadOnly = true
			case "route":
				tab.sourceLanguageRoute++
			case "mode-ABA":
				// The .go fallback has no rich Adapter. Supply the same narrow
				// test view used by the lifecycle suite so split is supported.
				tab.Adapter = &editor.Adapter{ID: "test-split", Create: func(editor.Host) editor.View { return foldSplitTestView{} }}
				if !a.setMode(tab, "split") || tab.Mode != "split" {
					t.Fatal("mode-ABA fixture did not enter split")
				}
				if !a.setMode(tab, "source") || tab.Mode != "source" {
					t.Fatal("mode-ABA fixture did not return to source")
				}
			case "session":
				a.resetSourceLanguageSession()
			case "activation":
				a.retireSourceLanguage(tab)
			case "encoding":
				tab.Doc.Encoding = document.UTF16LE
			case "bom":
				tab.Doc.BOM = !tab.Doc.BOM
			case "path":
				tab.Doc.Path = filepath.Join(t.TempDir(), "other.go")
			case "name":
				tab.Name = "other.go"
			case "new-tab":
				a.NewDocument("text")
			}
			before := languageState(tab)
			a.sourceLanguage.action = &old
			a.processSourceLanguageActions()
			languageUnchanged(t, tab, before)
			if b.String() != renameLifecycleSource {
				t.Fatal("stale action changed original source")
			}
			if a.sourceLanguage.rename != nil {
				if err := a.requestSourceRename(); err == nil {
					t.Fatal("new preview silently repaired original source identity", kind)
				}
				if a.sourceLanguage.rename.field != originalField {
					t.Fatal("refusal replaced native field")
				}
			}
		})
	}
}

func TestRenameNameOpaqueIdentityWithoutCallbackOrFrameRejectsDelayedRealSuccess(t *testing.T) {
	for _, kind := range []string{"commit-ABA", "empty-preedit-end", "mark-cancel", "markless-refusal-cancel", "attachment-ABA", "readonly-ABA", "view-release", "field-replacement", "buffer-pointer", "caret-only"} {
		t.Run(kind, func(t *testing.T) {
			a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
			renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
			if err := a.openSourceRename(); err != nil {
				t.Fatal(err)
			}
			renameName(t, a, "next")
			d := a.sourceLanguage.rename
			field, b := d.field, d.buffer
			// Hold AFTER real AnalyzeService returned, before the task publishes.
			// Removing notifications proves the opaque target backstop, not an
			// OnChange boolean. No frame occurs in the mutation/action sequence.
			analyzed, release := make(chan struct{}, 1), make(chan struct{})
			var once sync.Once
			finish := func() { once.Do(func() { close(release) }) }
			t.Cleanup(finish)
			a.sourceLanguageHooks.afterAnalyze = func(uint64) { analyzed <- struct{}{}; <-release }
			if err := a.requestSourceRename(); err != nil {
				t.Fatal(err)
			}
			task := a.sourceLanguage.task
			languageReceive(t, analyzed)
			a.queueSourceLanguageAction("rename-confirm", "")
			if a.sourceLanguage.action != nil {
				t.Fatal("running result exposed an apply action")
			}
			saved := sourceLanguageAction{kind: "rename-confirm", request: a.sourceLanguage.request, presentation: a.sourceLanguage.presentation, renameDialog: d, renameEpoch: d.epoch}
			r := d.request
			original := d.original
			field.OnChange, field.OnError = nil, nil
			switch kind {
			case "commit-ABA":
				field.ReplaceText(&ui.TextInputRange{Start: 0, End: 4}, "other")
				field.ReplaceText(&ui.TextInputRange{Start: 0, End: 5}, "next")
			case "empty-preedit-end":
				field.SetMarkedText(&ui.TextInputRange{Start: 0, End: 0}, "", ui.TextInputRange{})
				field.UnmarkText()
			case "mark-cancel":
				field.SetMarkedText(&ui.TextInputRange{Start: 0, End: 0}, "中", ui.TextInputRange{})
				field.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape})
			case "markless-refusal-cancel":
				field.SetMarkedText(&ui.TextInputRange{Start: 0, End: 0}, strings.Repeat("🙂", 129), ui.TextInputRange{})
				if field.InputActionError() == nil {
					t.Fatal("missing first-refused input")
				}
				if _, marked := field.MarkedRange(); marked {
					t.Fatal("refused field has mark")
				}
				field.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape})
			case "attachment-ABA":
				field.SetBuffer(text.New("next"))
				field.SetBuffer(b)
			case "readonly-ABA":
				field.SetReadOnly(true)
				field.SetReadOnly(false)
			case "view-release":
				field.ReleaseView()
			case "field-replacement":
				d.field = editor.New(b)
			case "buffer-pointer":
				d.buffer = text.New("next")
			case "caret-only":
				if err := b.SetSelection(0, 0); err != nil {
					t.Fatal(err)
				}
			}
			before := languageState(tab)
			if kind == "caret-only" {
				if err := a.sourceRenameRequestValid(r); err != nil {
					t.Fatal("name caret-only invalidated preview", err)
				}
			} else if err := a.sourceRenameRequestValid(r); err == nil {
				t.Fatal("old name target survived mutation", kind)
			}
			finish()
			languageWait(t, task.done)
			callback := languageReceive(t, q.calls)
			callback()
			callback()
			if kind == "caret-only" {
				if a.sourceLanguage.result.Status != code.ServiceComplete || len(d.patch) != 2 {
					t.Fatal("caret-only lost genuine result")
				}
				a.sourceLanguage.action = &saved
				a.processSourceLanguageActions()
				if tab.Doc.Buffer.String() != "package p; func f(next int){ _=next }" {
					t.Fatal("caret-only confirmation")
				}
				return
			}
			a.sourceLanguage.action = &saved
			a.processSourceLanguageActions()
			languageUnchanged(t, tab, before)
			if a.sourceLanguage.rename != d || d.original != original || len(d.patch) != 0 || len(a.sourceLanguage.result.Edits) != 0 || d.request != nil {
				t.Fatal("old result revived or destroyed original dialog")
			}
			// Restore only test-owned field pointers; keep the original source
			// seal, and obtain a NEW name target from an explicit fresh preview.
			d.field, d.buffer = field, b
			a.sourceLanguageHooks.afterAnalyze = nil
			if field.InputActionError() != nil {
				t.Fatal("normal field cancellation route lost")
			}
			renamePreview(t, a, q, "next")
			if d.original != original || a.sourceLanguage.result.Status != code.ServiceComplete || len(d.patch) != 2 {
				t.Fatal("fresh preview could not use same still-valid original source")
			}
		})
	}
}

func TestRenameNameCallbacksRetireOnlyPreviewAndSavedActionBeforeFrame(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
	renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
	renameReady(t, a, q, "next")
	d := a.sourceLanguage.rename
	original, field, b := d.original, d.field, d.buffer
	a.queueSourceLanguageAction("rename-confirm", "")
	saved := *a.sourceLanguage.action
	renameName(t, a, "other")
	renameName(t, a, "next")
	if a.sourceLanguage.rename != d || d.original != original || d.field != field || d.buffer != b || d.request != nil || len(d.patch) != 0 || a.sourceLanguage.action != nil {
		t.Fatal("native committed-name ABA rebound dialog/source/field")
	}
	before := languageState(tab)
	a.sourceLanguage.action = &saved
	a.processSourceLanguageActions()
	languageUnchanged(t, tab, before)
	renamePreview(t, a, q, "next")
	if d.original != original || len(d.patch) != 2 {
		t.Fatal("new preview lost OPEN target")
	}
}

func TestRenameBusyLatestOnlyOneAnalyzerAndAdmittedDispatcher(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
	renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
	started, release := languageHold(t, a)
	var active, maximum, count atomic.Int32
	hold := a.sourceLanguageHooks.beforeAnalyze
	a.sourceLanguageHooks.beforeAnalyze = func(ctx context.Context, id uint64) {
		languageMax(&maximum, active.Add(1))
		count.Add(1)
		hold(ctx, id)
	}
	a.sourceLanguageHooks.afterAnalyze = func(uint64) { active.Add(-1) }
	first := languageRequest(t, a, languageIntent(code.ServiceDiagnostics))
	languageReceive(t, started)
	if err := a.openSourceRename(); err != nil {
		t.Fatal(err)
	}
	d := a.sourceLanguage.rename
	original := d.original
	for _, name := range []string{"one", "two", "finalName"} {
		renameName(t, a, name)
		if err := a.requestSourceRename(); err != nil {
			t.Fatal(err)
		}
		if a.sourceLanguage.task != first || a.sourceLanguage.pending == nil || count.Load() != 1 || q.outstanding.Load() != 0 || d.original != original {
			t.Fatal("busy rename started second source/task or lost OPEN target")
		}
	}
	latest := a.sourceLanguage.pending.owner.request
	a.sourceLanguageHooks.dispatch = func(func()) { t.Error("pending request rebound admitted dispatcher") }
	release()
	languageWait(t, first.done)
	oldNotice := languageReceive(t, q.calls)
	oldNotice()
	second := a.sourceLanguage.task
	if second == nil || second == first || second.id != latest {
		t.Fatal("latest bounded successor missing")
	}
	languageReceive(t, started)
	oldNotice()
	languageWait(t, second.done)
	languageReceive(t, q.calls)()
	if a.sourceLanguage.result.Status != code.ServiceComplete || a.sourceLanguage.intent.newName != "finalName" || len(d.patch) != 2 || count.Load() != 2 || maximum.Load() != 1 || active.Load() != 0 || q.maximum.Load() != 1 || q.outstanding.Load() != 0 {
		t.Fatal("single slot/latest real result bounds", a.sourceLanguage.status, a.sourceLanguage.reason, count.Load(), maximum.Load())
	}
}

func TestRenameBusySourceSelectionABARefusesOpenAndPendingDoesNotLaunch(t *testing.T) {
	for _, changeAfterOpen := range []bool{false, true} {
		a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
		renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
		started, release := languageHold(t, a)
		first := languageRequest(t, a, languageIntent(code.ServiceDiagnostics))
		languageReceive(t, started)
		b := tab.Doc.Buffer
		set := b.SelectionSet()
		if changeAfterOpen {
			if err := a.openSourceRename(); err != nil {
				t.Fatal(err)
			}
			renameName(t, a, "next")
			if err := a.requestSourceRename(); err != nil {
				t.Fatal(err)
			}
		}
		_ = b.SetSelection(0, 0)
		_ = b.SetSelectionSet(set)
		before := languageState(tab)
		if !changeAfterOpen {
			if err := a.openSourceRename(); err == nil {
				t.Fatal("busy OPEN repaired stale full selection target")
			}
		}
		release()
		languageWait(t, first.done)
		languageReceive(t, q.calls)()
		if a.sourceLanguage.task != nil || a.sourceLanguage.pending != nil || len(a.sourceLanguage.result.Edits) != 0 {
			t.Fatal("stale pending rename launched/revived")
		}
		languageUnchanged(t, tab, before)
	}
}

func TestRenamePendingNameNativeInputKeepsFieldAndOwnEscape(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
	renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
	started, release := languageHold(t, a)
	first := languageRequest(t, a, languageIntent(code.ServiceDiagnostics))
	languageReceive(t, started)
	if err := a.openSourceRename(); err != nil {
		t.Fatal(err)
	}
	renameName(t, a, "next")
	if err := a.requestSourceRename(); err != nil {
		t.Fatal(err)
	}
	d := a.sourceLanguage.rename
	field := d.field
	field.OnChange, field.OnError = nil, nil
	field.SetMarkedText(&ui.TextInputRange{Start: 0, End: 0}, "中", ui.TextInputRange{})
	if field.InputActionError() == nil {
		t.Fatal("name native preedit fixture")
	}
	before := languageState(tab)
	release()
	languageWait(t, first.done)
	languageReceive(t, q.calls)()
	if a.sourceLanguage.rename != d || d.field != field || a.sourceLanguage.task != nil || len(d.patch) != 0 || field.InputActionError() == nil {
		t.Fatal("pending refusal released or settled field native owner")
	}
	a.queueSourceLanguageAction("rename-cancel", "")
	a.processSourceLanguageActions()
	if a.sourceLanguage.rename != d || field.InputActionError() == nil {
		t.Fatal("refused cancel discarded native name")
	}
	field.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape})
	if field.InputActionError() != nil {
		t.Fatal("original field Escape lost")
	}
	languageUnchanged(t, tab, before)
}

func TestRenameSavedConfirmAfterRealSaveAsNeverApplies(t *testing.T) {
	for _, published := range []bool{false, true} {
		a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
		renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
		if err := a.openSourceRename(); err != nil {
			t.Fatal(err)
		}
		renameName(t, a, "next")
		if err := a.requestSourceRename(); err != nil {
			t.Fatal(err)
		}
		task := a.sourceLanguage.task
		languageWait(t, task.done)
		notice := languageReceive(t, q.calls)
		if published {
			notice()
		}
		d := a.sourceLanguage.rename
		saved := sourceLanguageAction{kind: "rename-confirm", request: a.sourceLanguage.request, presentation: a.sourceLanguage.presentation, renameDialog: d, renameEpoch: d.epoch}
		if published {
			a.queueSourceLanguageAction("rename-confirm", "")
			if a.sourceLanguage.action == nil {
				t.Fatal("published rename missing action")
			}
			saved = *a.sourceLanguage.action
		}
		destination := filepath.Join(t.TempDir(), "renamed.go")
		a.saveDialog = func(*Tab) (string, error) { return destination, nil }
		if !a.Save(tab, true) || tab.Doc.Path != destination || string(mustReadFile(t, destination)) != renameLifecycleSource {
			t.Fatal("real SaveAs fixture", a.Message)
		}
		before := languageState(tab)
		if !published {
			notice()
		}
		a.sourceLanguage.action = &saved
		a.processSourceLanguageActions()
		languageUnchanged(t, tab, before)
		if a.sourceLanguage.rename != nil && len(a.sourceLanguage.rename.patch) != 0 {
			t.Fatal("SaveAs retained applicable rename")
		}
	}
}

func TestRenameHostFailuresKeepNoUsablePrefix(t *testing.T) {
	for _, kind := range []string{"error", "panic", "malformed"} {
		a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
		renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
		if err := a.openSourceRename(); err != nil {
			t.Fatal(err)
		}
		switch kind {
		case "error":
			a.sourceLanguageHooks.fail = errors.New("actual negative hook")
		case "panic":
			a.sourceLanguageHooks.panicValue = "negative panic"
		case "malformed":
			a.sourceLanguageHooks.malformed = true
		}
		before := languageState(tab)
		renamePreview(t, a, q, "next")
		if a.sourceLanguage.status != sourceLanguageHostError || a.sourceLanguage.failure == nil || len(a.sourceLanguage.result.Edits) != 0 || a.sourceLanguage.rename == nil || len(a.sourceLanguage.rename.patch) != 0 {
			t.Fatal("host failure masqueraded as applicable rename", kind)
		}
		a.queueSourceLanguageAction("rename-confirm", "")
		a.processSourceLanguageActions()
		languageUnchanged(t, tab, before)
	}
}

func TestRenameTerminalOnChangeReplacementStateSurvives(t *testing.T) {
	for _, kind := range []string{"new-tab", "new-read-request", "new-rename-dialog", "new-buffer"} {
		a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
		renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
		renameReady(t, a, q, "next")
		original := tab.Editor.OnChange
		calls := 0
		var current *Tab
		var request, presentation uint64
		var task *sourceLanguageTask
		var dialog *sourceRenameDialog
		var input *editor.Editor
		var buffer *text.Buffer
		var selections text.SelectionSet
		tab.Editor.OnChange = func() {
			calls++
			if original != nil {
				original()
			}
			switch kind {
			case "new-tab":
				a.NewDocument("text")
			case "new-read-request":
				a.runSourceLanguageCommand("symbols")
			case "new-rename-dialog":
				if err := a.openSourceRename(); err != nil {
					t.Fatal(err)
				}
			case "new-buffer":
				b := text.New("callback buffer")
				tab.Doc.Buffer = b
				tab.Editor.SetBuffer(b)
			}
			a.Message = "callback exit"
			a.sourceLanguage.message = "replacement exit"
			current, request, presentation, task, dialog = a.Current(), a.sourceLanguage.request, a.sourceLanguage.presentation, a.sourceLanguage.task, a.sourceLanguage.rename
			input, buffer = tab.Editor, tab.Editor.Buffer
			selections = buffer.SelectionSet()
		}
		renameConfirm(t, a)
		if calls != 1 || a.Current() != current || a.sourceLanguage.request != request || a.sourceLanguage.presentation != presentation || a.sourceLanguage.task != task || a.sourceLanguage.rename != dialog || tab.Editor != input || tab.Editor.Buffer != buffer || !reflect.DeepEqual(buffer.SelectionSet(), selections) || a.Message != "callback exit" || a.sourceLanguage.message != "replacement exit" {
			t.Fatal("rename wrote after terminal OnChange", kind, calls, a.Message)
		}
		if kind != "new-buffer" && tab.Doc.Buffer.String() != "package p; func f(next int){ _=next }" {
			t.Fatal("commit was rolled back")
		}
		if task != nil {
			languageWait(t, task.done)
			languageReceive(t, q.calls)()
			if a.sourceLanguage.result.Status != code.ServiceComplete {
				t.Fatal("callback replacement request was discarded")
			}
		}
	}
}
