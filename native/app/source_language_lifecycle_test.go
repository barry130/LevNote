package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/core/engineruntime"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
	"levnote.local/native/ui/editor"
)

const languageGoSource = "package p\r\n// 🙂 original\r\nfunc Solve(变量 int) int {\r\n  next := 变量\n  return next\r\n}\n"

func languageReceive[T any](t *testing.T, channel <-chan T) T {
	t.Helper()
	select {
	case value := <-channel:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("language test deadlock: expected channel handoff")
		var zero T
		return zero
	}
}

func languageWait(t *testing.T, done <-chan struct{}) {
	t.Helper()
	languageReceive(t, done)
}

type languageQueue struct {
	calls       chan func()
	outstanding atomic.Int32
	maximum     atomic.Int32
}

func newLanguageQueue() *languageQueue { return &languageQueue{calls: make(chan func(), 8)} }

func languageMax(value *atomic.Int32, n int32) {
	for old := value.Load(); old < n; old = value.Load() {
		if value.CompareAndSwap(old, n) {
			return
		}
	}
}

func (q *languageQueue) dispatch(callback func()) {
	languageMax(&q.maximum, q.outstanding.Add(1))
	var once sync.Once
	q.calls <- func() {
		once.Do(func() { q.outstanding.Add(-1) })
		callback()
	}
}

func languageFixture(t *testing.T, suffix, source string) (*App, *Tab, *languageQueue, string) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "source"+suffix)
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Doc == nil || tab.Editor == nil {
		t.Fatalf("real App Open failed: %s", a.Message)
	}
	q := newLanguageQueue()
	a.sourceLanguageHooks = &sourceLanguageHooks{dispatch: q.dispatch}
	return a, tab, q, path
}

func languageIntent(operation code.ServiceOperation) sourceLanguageIntent {
	return sourceLanguageIntent{operation: operation, limits: code.DefaultServiceLimits()}
}

func languageRequest(t *testing.T, a *App, intent sourceLanguageIntent) *sourceLanguageTask {
	t.Helper()
	if err := a.requestSourceLanguage(intent); err != nil {
		t.Fatal(err)
	}
	if a.sourceLanguage.task == nil {
		t.Fatal("admitted request has no physical task")
	}
	return a.sourceLanguage.task
}

func languageComplete(t *testing.T, a *App, q *languageQueue, intent sourceLanguageIntent) {
	t.Helper()
	task := languageRequest(t, a, intent)
	languageWait(t, task.done)
	languageReceive(t, q.calls)()
}

func languageHold(t *testing.T, a *App) (<-chan context.Context, func()) {
	t.Helper()
	started, release := make(chan context.Context, 1), make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(release) }) }
	t.Cleanup(finish)
	a.sourceLanguageHooks.beforeAnalyze = func(ctx context.Context, _ uint64) { started <- ctx; <-release }
	return started, finish
}

type languageDocumentState struct {
	doc                                                                                              *document.Document
	input                                                                                            *editor.Editor
	buffer                                                                                           *text.Buffer
	source, path, name, mode                                                                         string
	version, selectionGeneration                                                                     uint64
	selections                                                                                       text.SelectionSet
	history                                                                                          text.HistoryStatus
	encoding                                                                                         document.Encoding
	bom, readOnly, editorReadOnly, dirty, isNew, canUndo, canRedo, group, pending, composing, marked bool
	mark                                                                                             ui.TextInputRange
}

func languageState(tab *Tab) languageDocumentState {
	b := tab.Doc.Buffer
	mark, marked := tab.Editor.MarkedRange()
	return languageDocumentState{doc: tab.Doc, input: tab.Editor, buffer: b, source: b.String(), path: tab.Doc.Path, name: tab.Name, mode: tab.Mode,
		version: b.Version(), selectionGeneration: b.SelectionGeneration(), selections: b.SelectionSet(), history: b.History(),
		encoding: tab.Doc.Encoding, bom: tab.Doc.BOM, readOnly: tab.Doc.ReadOnly, editorReadOnly: tab.Editor.ReadOnly, dirty: tab.Doc.Dirty(), isNew: tab.Doc.IsNew,
		canUndo: b.CanUndo(), canRedo: b.CanRedo(), group: b.InGroup(), pending: tab.Editor.MultiCompositionPending(), composing: tab.Editor.CompositionActive(), mark: mark, marked: marked}
}

func languageUnchanged(t *testing.T, tab *Tab, before languageDocumentState) {
	t.Helper()
	if after := languageState(tab); !reflect.DeepEqual(before, after) {
		t.Fatalf("language read/refusal changed document\nbefore=%+v\nafter=%+v", before, after)
	}
}

func TestLanguageHostSingleSlotCoalescesLatestIntent(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", languageGoSource)
	starts, release := make(chan uint64, 4), make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(release) }) }
	t.Cleanup(finish)
	var active, maximum, count atomic.Int32
	a.sourceLanguageHooks.beforeAnalyze = func(_ context.Context, id uint64) {
		languageMax(&maximum, active.Add(1))
		count.Add(1)
		starts <- id
		<-release
	}
	a.sourceLanguageHooks.afterAnalyze = func(uint64) { active.Add(-1) }
	first := languageRequest(t, a, languageIntent(code.ServiceDiagnostics))
	if id := languageReceive(t, starts); id != first.id {
		t.Fatal("wrong first request ID")
	}
	first.mu.Lock()
	retained := first.request.Source
	first.mu.Unlock()
	if retained != "" {
		t.Fatal("task retained a second source after handing it to the analyzer")
	}
	for i := 0; i < 100; i++ {
		intent := languageIntent(code.ServiceDiagnostics)
		if i == 99 {
			intent.operation = code.ServiceSymbols
		}
		if err := tab.Doc.Buffer.SetSelection(i%8, i%8); err != nil {
			t.Fatal(err)
		}
		if err := a.requestSourceLanguage(intent); err != nil {
			t.Fatal(err)
		}
		if a.sourceLanguage.task != first || a.sourceLanguage.pending == nil || count.Load() != 1 || q.outstanding.Load() != 0 {
			t.Fatal("pending request started a second analyzer, source or notification")
		}
	}
	latest := a.sourceLanguage.pending.owner.request
	before := languageState(tab)
	finish()
	languageWait(t, first.done)
	if a.sourceLanguage.status != sourceLanguageRunning || len(a.sourceLanguage.result.Symbols) != 0 {
		t.Fatal("worker wrote UI state before dispatch")
	}
	oldNotice := languageReceive(t, q.calls)
	oldNotice()
	second := a.sourceLanguage.task
	if second == nil || second.id != latest || second == first {
		t.Fatal("latest intent did not replace the retired physical slot")
	}
	if id := languageReceive(t, starts); id != latest {
		t.Fatal("an intermediate intent executed")
	}
	oldNotice()
	if a.sourceLanguage.task != second || a.sourceLanguage.status != sourceLanguageRunning {
		t.Fatal("duplicate old notice released the replacement job")
	}
	languageWait(t, second.done)
	languageReceive(t, q.calls)()
	if count.Load() != 2 || maximum.Load() != 1 || active.Load() != 0 || q.maximum.Load() != 1 || q.outstanding.Load() != 0 {
		t.Fatalf("slot bounds: starts=%d active=%d maximum=%d notices=%d", count.Load(), active.Load(), maximum.Load(), q.maximum.Load())
	}
	if a.sourceLanguage.status != string(code.ServiceComplete) || len(a.sourceLanguage.result.Symbols) < 2 {
		t.Fatalf("latest real symbols missing: %+v", a.sourceLanguage)
	}
	found := false
	for _, symbol := range a.sourceLanguage.result.Symbols {
		if symbol.Name == "Solve" && symbol.Kind == "func" {
			found = true
		}
	}
	if !found {
		t.Fatalf("real Go function symbol missing: %+v", a.sourceLanguage.result.Symbols)
	}
	languageUnchanged(t, tab, before)
}

func TestLanguageHostOldPublicationCannotOverwriteLatest(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", languageGoSource)
	first := languageRequest(t, a, languageIntent(code.ServiceSymbols))
	languageWait(t, first.done)
	old := languageReceive(t, q.calls)
	a.cancelSourceLanguage()
	if a.sourceLanguage.owner != nil || a.sourceLanguage.pending != nil || len(a.sourceLanguage.result.Symbols) != 0 {
		t.Fatal("cancel retained owners or results")
	}
	old()
	if a.sourceLanguage.status != string(code.ServiceCanceled) || a.sourceLanguage.task != nil {
		t.Fatal("old completion overwrote cancellation")
	}
	second := languageRequest(t, a, languageIntent(code.ServiceDiagnostics))
	old()
	if a.sourceLanguage.task != second || a.sourceLanguage.status != sourceLanguageRunning {
		t.Fatal("duplicate old callback cleared current task")
	}
	before := languageState(tab)
	languageWait(t, second.done)
	languageReceive(t, q.calls)()
	if a.sourceLanguage.status != string(code.ServiceComplete) || len(a.sourceLanguage.result.Symbols) != 0 {
		t.Fatal("old symbols leaked into newer diagnostics")
	}
	first.mu.Lock()
	empty := first.request.Source == "" && reflect.DeepEqual(first.outcome, sourceLanguageOutcome{}) && first.cancel == nil
	first.mu.Unlock()
	if !empty {
		t.Fatal("abandoned task retained source/outcome/cancel")
	}
	languageUnchanged(t, tab, before)
}

func TestLanguageHostUiOwnershipAndNoInlineFallback(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", languageGoSource)
	before := languageState(tab)
	task := languageRequest(t, a, languageIntent(code.ServiceSymbols))
	owner := a.sourceLanguage.owner
	languageWait(t, task.done)
	if a.sourceLanguage.owner != owner || a.sourceLanguage.status != sourceLanguageRunning || len(a.sourceLanguage.result.Symbols) != 0 {
		t.Fatal("background publication mutated UI state")
	}
	languageReceive(t, q.calls)()
	if a.sourceLanguage.status != string(code.ServiceComplete) {
		t.Fatal("queued UI completion failed")
	}
	languageUnchanged(t, tab, before)
	a.sourceLanguageHooks = nil
	inline := 0
	a.uiDispatch = func(callback func()) { inline++; callback() }
	if err := a.requestSourceLanguage(languageIntent(code.ServiceSymbols)); err == nil || a.sourceLanguage.status != sourceLanguageHostError ||
		a.sourceLanguage.reason != "ui-dispatch-unavailable" || a.sourceLanguage.task != nil || inline != 0 {
		t.Fatal("language used App.update's inline/test fallback")
	}
	languageUnchanged(t, tab, before)
}

func TestLanguageHostOwnerABAMatrix(t *testing.T) {
	for _, name := range []string{"selection-read", "readonly-ABA", "buffer-ABA", "text-undo", "journal-over-64", "editor", "document", "profile-ABA", "mode-ABA", "activation-ABA", "recover-draft", "restore", "close-reopen", "reload", "reopen-encoding"} {
		t.Run(name, func(t *testing.T) {
			a, tab, q, path := languageFixture(t, ".go", languageGoSource)
			if name == "restore" {
				a.Persist()
			}
			started, release := languageHold(t, a)
			task := languageRequest(t, a, languageIntent(code.ServiceSymbols))
			languageReceive(t, started)
			original := a.sourceLanguage.owner
			b := tab.Doc.Buffer
			switch name {
			case "selection-read":
				if err := b.SetSelectionSet(text.SelectionSet{Ranges: []text.Range{{Anchor: 5, Head: 1}, {Anchor: 8, Head: 8}}, Primary: 1}); err != nil {
					t.Fatal(err)
				}
			case "readonly-ABA":
				tab.Editor.SetReadOnly(true)
				tab.Editor.SetReadOnly(false)
			case "buffer-ABA":
				tab.Editor.SetBuffer(text.New(languageGoSource))
				tab.Editor.SetBuffer(b)
			case "text-undo", "journal-over-64":
				n := 1
				if name == "journal-over-64" {
					n = 70
				}
				for i := 0; i < n; i++ {
					if err := b.ReplaceUTF16(0, 0, " "); err != nil {
						t.Fatal(err)
					}
				}
				for i := 0; i < n; i++ {
					b.Undo()
				}
				if b.String() != languageGoSource {
					t.Fatal("Undo fixture did not restore identical source")
				}
			case "editor":
				tab.Editor = editor.New(b)
			case "document":
				copy := *tab.Doc
				tab.Doc = &copy
			case "profile-ABA":
				a.configureTab(tab)
				a.configureTab(tab)
			case "mode-ABA":
				tab.Adapter = &editor.Adapter{ID: "test-split", Create: func(editor.Host) editor.View { return foldSplitTestView{} }}
				a.setMode(tab, "split")
				a.setMode(tab, "source")
			case "activation-ABA":
				a.NewDocument("text")
				if !a.Select(0) {
					t.Fatal("select failed")
				}
			case "recover-draft":
				if err := a.recoverDraft("kept.go", "package draft\n"); err != nil {
					t.Fatal(err)
				}
				if !a.Select(0) {
					t.Fatal("return from draft failed")
				}
			case "restore":
				a.Restore()
			case "close-reopen":
				if !a.CloseTab(0) {
					t.Fatal(a.Message)
				}
				a.Open(path)
				if a.Current() == tab {
					t.Fatal("close/reopen reused Tab")
				}
			case "reload":
				if err := os.WriteFile(path, []byte(languageGoSource), 0600); err != nil {
					t.Fatal(err)
				}
				a.Reload()
			case "reopen-encoding":
				tab.ByteInspector = true
				if !a.reopenWithEncoding(tab, document.UTF8) {
					t.Fatal(a.Message)
				}
			}
			before := languageState(tab)
			release()
			languageWait(t, task.done)
			languageReceive(t, q.calls)()
			if name == "selection-read" {
				if a.sourceLanguage.status != string(code.ServiceComplete) || a.sourceLanguage.owner != original {
					t.Fatalf("ordinary selection movement invalidated read: %+v", a.sourceLanguage)
				}
				if err := original.input.CheckSourceTarget(original.target, true); err == nil {
					t.Fatal("selection-sensitive target unexpectedly survived")
				}
			} else if a.sourceLanguage.status == string(code.ServiceComplete) || a.sourceLanguage.owner != nil || len(a.sourceLanguage.result.Symbols) != 0 {
				t.Fatalf("old owner revived through %s: %+v", name, a.sourceLanguage)
			}
			languageUnchanged(t, tab, before)
		})
	}
}

func TestLanguageHostPendingChangedTargetIsExplicitStale(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", languageGoSource)
	started, release := languageHold(t, a)
	task := languageRequest(t, a, languageIntent(code.ServiceDiagnostics))
	languageReceive(t, started)
	if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, " "); err != nil {
		t.Fatal(err)
	}
	before := languageState(tab)
	if err := a.requestSourceLanguage(languageIntent(code.ServiceSymbols)); err == nil || a.sourceLanguage.status != sourceLanguageStale || a.sourceLanguage.pending != nil || a.sourceLanguage.owner != nil {
		t.Fatal("changed text acquired a second pending source/target")
	}
	release()
	languageWait(t, task.done)
	languageReceive(t, q.calls)()
	languageUnchanged(t, tab, before)
	a.sourceLanguageHooks.beforeAnalyze = nil
	languageComplete(t, a, q, languageIntent(code.ServiceSymbols))
	if a.sourceLanguage.status != string(code.ServiceComplete) {
		t.Fatal("explicit retry after physical retirement failed")
	}
}

func TestLanguageHostShutdownUsesEngineCleanupBeforeLanguageJoin(t *testing.T) {
	a, _, q, _ := languageFixture(t, ".go", languageGoSource)
	started, release := languageHold(t, a)
	task := languageRequest(t, a, languageIntent(code.ServiceSymbols))
	languageCtx := languageReceive(t, started)
	f := newControlledExporter()
	lane := newEngineLane(f)
	a.exporter, a.engineLane = f, lane
	engineCtx, cancel := context.WithCancel(context.Background())
	engineDone := make(chan struct{})
	if err := lane.submitPreview(&engineTask{ctx: engineCtx, cancel: cancel, id: engineipc.MarkdownHTML,
		source: engineruntime.Source{}, complete: func(engineruntime.Result, error) { close(engineDone) }}); err != nil {
		t.Fatal(err)
	}
	awaitExportCall(t, f)
	cleanup := make(chan error, 1)
	f.onClose = func() {
		var err error
		if languageCtx.Err() == nil {
			err = errors.New("engine cleanup ran before language cancellation")
		}
		select {
		case <-task.done:
			err = errors.New("language joined before engine cleanup handoff")
		default:
		}
		cleanup <- err
		release() // Language can return only after the actual runner Close handoff.
	}
	shutdown := make(chan struct{})
	go func() { a.Shutdown(); close(shutdown) }()
	if err := languageReceive(t, cleanup); err != nil {
		t.Fatal(err)
	}
	languageWait(t, shutdown)
	languageWait(t, engineDone)
	languageWait(t, lane.closed)
	languageWait(t, task.done)
	if a.sourceLanguage.owner != nil || a.sourceLanguage.pending != nil || a.sourceLanguage.task != nil || len(a.sourceLanguage.result.Symbols) != 0 {
		t.Fatal("shutdown retained language ownership")
	}
	languageReceive(t, q.calls)()
	if a.sourceLanguage.status != string(code.ServiceCanceled) || a.sourceLanguage.task != nil {
		t.Fatal("queued completion published after shutdown")
	}
	a.Shutdown() // Existing idempotent join remains usable.
}

func TestLanguageHostNativeInputAndForeignGroupsRemainOwned(t *testing.T) {
	for _, name := range []string{"native-compose", "markless-pending", "foreign-group", "foreign-composition"} {
		t.Run(name, func(t *testing.T) {
			a, tab, q, _ := languageFixture(t, ".go", languageGoSource)
			if name == "markless-pending" {
				b := text.NewWithLimit(languageGoSource, len(languageGoSource)+1)
				tab.Doc.Buffer = b
				tab.Editor.SetBuffer(b)
			}
			tester := ui.NewTester(a.View, 900, 620)
			tester.Frame()
			if err := tester.Click(tab.Editor.Label); err != nil {
				t.Fatal(err)
			}
			b := tab.Doc.Buffer
			var token text.CompositionToken
			switch name {
			case "native-compose":
				tester.Compose("中", 1)
			case "markless-pending":
				task40SetSelections(t, tab, []text.Range{{Anchor: 1, Head: 1}, {Anchor: 4, Head: 4}}, 0)
				tester.Compose("拼🙂", 2)
				if _, marked := tab.Editor.MarkedRange(); marked || !tab.Editor.MultiCompositionPending() {
					t.Fatal("first refused preedit was not markless/pending")
				}
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
			}
			before := languageState(tab)
			changes := 0
			tab.Editor.OnChange = func() { changes++ }
			if err := a.requestSourceLanguage(languageIntent(code.ServiceDiagnostics)); err == nil || a.sourceLanguage.task != nil || q.outstanding.Load() != 0 {
				t.Fatal("language crossed provisional/foreign input")
			}
			languageUnchanged(t, tab, before)
			if changes != 0 || !tester.Focused(tab.Editor.Label) {
				t.Fatal("language refusal emitted edit or stole focus")
			}
			switch name {
			case "native-compose", "markless-pending":
				tester.Key(0, ui.KeyEscape)
				if tab.Editor.CompositionActive() || b.InGroup() {
					t.Fatal("original native owner could not cancel")
				}
			case "foreign-group":
				if !b.CancelGroup() {
					t.Fatal("foreign ordinary group lost its owner")
				}
			case "foreign-composition":
				if err := b.CancelCompositionGroup(token); err != nil {
					t.Fatal(err)
				}
			}
			languageComplete(t, a, q, languageIntent(code.ServiceDiagnostics))
			if a.sourceLanguage.status != string(code.ServiceComplete) {
				t.Fatalf("retry after original owner cancellation: %+v", a.sourceLanguage)
			}
		})
	}
}

func TestLanguageHostRefusedCloseAndPersistenceStayRefused(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", languageGoSource)
	if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, " "); err != nil {
		t.Fatal(err)
	}
	a.Persist()
	recovery := mustReadFile(t, filepath.Join(a.configDir, "session.json"))
	started, release := languageHold(t, a)
	task := languageRequest(t, a, languageIntent(code.ServiceSymbols))
	languageReceive(t, started)
	owner := a.sourceLanguage.owner
	a.closeDialog = func(*Tab) (closeDecision, error) { return closeCancel, nil }
	before := languageState(tab)
	if a.CloseTab(0) || a.CanClose() || a.shuttingDown.Load() || a.sourceLanguage.owner != owner {
		t.Fatal("refused close retired live work or became successful shutdown")
	}
	languageUnchanged(t, tab, before)
	if string(mustReadFile(t, filepath.Join(a.configDir, "session.json"))) != string(recovery) {
		t.Fatal("refused close changed recovery bytes")
	}
	badConfig := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(badConfig, []byte("barrier"), 0600); err != nil {
		t.Fatal(err)
	}
	originalConfig := a.configDir
	a.configDir = badConfig
	a.closeDialog = func(*Tab) (closeDecision, error) { return closeDiscard, nil }
	if a.CanClose() || a.shuttingDown.Load() || a.sourceLanguage.owner != owner {
		t.Fatal("persistence refusal became shutdown")
	}
	a.configDir = originalConfig
	languageUnchanged(t, tab, before)
	release()
	languageWait(t, task.done)
	languageReceive(t, q.calls)()
	if a.sourceLanguage.status != string(code.ServiceComplete) {
		t.Fatal("refused close destroyed still-valid language result")
	}
}

func TestLanguageHostCancellationAndFailureShape(t *testing.T) {
	for _, name := range []string{"invalid-go", "invalid-json", "invalid-limits", "limit", "unsupported", "unsupported-json-operation", "ambiguous", "canceled", "service-canceled", "host-error", "panic", "malformed"} {
		t.Run(name, func(t *testing.T) {
			suffix, source := ".go", languageGoSource
			if name == "invalid-go" {
				source = "package p\nfunc ("
			}
			if name == "invalid-json" {
				suffix, source = ".json", "{ // comment\n}"
			}
			if name == "unsupported-json-operation" {
				suffix, source = ".json", `{"x":1}`
			}
			if name == "ambiguous" {
				source = "package p\nfunc f(){ var x int; var x int; _ = x }\n"
			}
			a, tab, q, _ := languageFixture(t, suffix, source)
			intent := languageIntent(code.ServiceDiagnostics)
			want := code.ServiceInvalid
			switch name {
			case "invalid-limits":
				intent.limits.Structure.Tokens = 0
			case "unsupported-json-operation":
				intent.operation = code.ServiceDefinition
				want = code.ServiceUnsupported
			case "limit":
				intent.limits.Structure.Tokens = 1
				want = code.ServiceLimit
			case "unsupported":
				intent.operation = "not-an-operation"
				want = code.ServiceUnsupported
			case "ambiguous":
				intent.operation = code.ServiceDefinition
				intent.byteOffset = strings.LastIndex(source, "x")
				want = code.ServiceAmbiguous
			case "host-error":
				a.sourceLanguageHooks.fail = errors.New("controlled analyzer host error")
			case "panic":
				a.sourceLanguageHooks.panicValue = "controlled analyzer panic"
			case "malformed":
				a.sourceLanguageHooks.malformed = true
			}
			before := languageState(tab)
			if name == "canceled" || name == "service-canceled" {
				started, release := languageHold(t, a)
				task := languageRequest(t, a, intent)
				languageReceive(t, started)
				if name == "service-canceled" {
					task.mu.Lock()
					cancel := task.cancel
					task.mu.Unlock()
					cancel()
				} else {
					a.cancelSourceLanguage()
				}
				release()
				languageWait(t, task.done)
				languageReceive(t, q.calls)()
				want = code.ServiceCanceled
			} else {
				languageComplete(t, a, q, intent)
			}
			if name == "host-error" || name == "panic" || name == "malformed" {
				if a.sourceLanguage.status != sourceLanguageHostError || a.sourceLanguage.failure == nil {
					t.Fatalf("host failure hidden: %+v", a.sourceLanguage)
				}
			} else if a.sourceLanguage.status != string(want) {
				t.Fatalf("status=%q want %q reason=%q", a.sourceLanguage.status, want, a.sourceLanguage.reason)
			}
			if name == "limit" && (a.sourceLanguage.result.Resource == "" || a.sourceLanguage.result.Limit != 1) {
				t.Fatal("lost limit resource/amount")
			}
			if (name == "invalid-go" || name == "invalid-json") && len(a.sourceLanguage.result.Diagnostics) == 0 {
				t.Fatal("real invalid diagnostics lost")
			}
			if a.sourceLanguage.reason == "" || len(a.sourceLanguage.result.Edits) != 0 || len(a.sourceLanguage.result.Symbols) != 0 || a.sourceLanguage.result.NoOp {
				t.Fatal("failure lost reason or retained a usable success payload")
			}
			languageUnchanged(t, tab, before)
		})
	}
	bad := code.ServiceResult{Status: code.ServiceCanceled, Edits: []code.ServiceEdit{{OldText: "negative-control"}}}
	if validateSourceLanguageResult(bad) == nil {
		t.Fatal("failed service result with edits was accepted")
	}
}

func TestLanguageHostPendingCapturesDispatcherAndRetiresStaleIntent(t *testing.T) {
	for _, changed := range []bool{false, true} {
		t.Run(fmt.Sprintf("changed-%v", changed), func(t *testing.T) {
			a, tab, q, _ := languageFixture(t, ".go", languageGoSource)
			started, release := languageHold(t, a)
			first := languageRequest(t, a, languageIntent(code.ServiceDiagnostics))
			languageReceive(t, started)
			if err := a.requestSourceLanguage(languageIntent(code.ServiceSymbols)); err != nil {
				t.Fatal(err)
			}
			pending := a.sourceLanguage.pending
			otherQueue := newLanguageQueue()
			a.sourceLanguageHooks.dispatch = otherQueue.dispatch
			if changed {
				if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, " "); err != nil {
					t.Fatal(err)
				}
				tab.Doc.Buffer.Undo()
			}
			before := languageState(tab)
			release()
			languageWait(t, first.done)
			notice := languageReceive(t, q.calls)
			notice()
			if changed {
				if a.sourceLanguage.task != nil || a.sourceLanguage.pending != nil || a.sourceLanguage.owner != nil || a.sourceLanguage.status != sourceLanguageStale {
					t.Fatal("stale pending intent started or retained an owner")
				}
			} else {
				second := a.sourceLanguage.task
				if second == nil || second.id != pending.owner.request {
					t.Fatal("latest pending intent was not admitted")
				}
				languageReceive(t, started)
				languageWait(t, second.done)
				if otherQueue.outstanding.Load() != 0 || q.outstanding.Load() != 1 {
					t.Fatal("pending worker reread a later dispatcher")
				}
				languageReceive(t, q.calls)()
				if a.sourceLanguage.status != string(code.ServiceComplete) || len(a.sourceLanguage.result.Symbols) < 2 {
					t.Fatal("pending real symbols did not publish")
				}
			}
			notice()
			languageUnchanged(t, tab, before)
		})
	}
}

func TestLanguageHostAdmissionBudgetRefusalsRetainFullState(t *testing.T) {
	for _, name := range []string{"bytes", "prefix", "new-name", "invalid-byte-budget"} {
		for _, busy := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s-busy-%v", name, busy), func(t *testing.T) {
				a, tab, q, _ := languageFixture(t, ".go", languageGoSource)
				var task *sourceLanguageTask
				var release func()
				if busy {
					started, finish := languageHold(t, a)
					release = finish
					task = languageRequest(t, a, languageIntent(code.ServiceDiagnostics))
					languageReceive(t, started)
				}
				intent := languageIntent(code.ServiceSymbols)
				want := code.ServiceLimit
				switch name {
				case "bytes":
					intent.limits.Structure.Bytes = len(languageGoSource) - 1
				case "prefix":
					intent.prefix = strings.Repeat("a", sourceLanguageFieldBytes+1)
				case "new-name":
					intent.newName = strings.Repeat("a", sourceLanguageFieldBytes+1)
				case "invalid-byte-budget":
					intent.limits.Structure.Bytes = 0
					want = code.ServiceInvalid
				}
				before := languageState(tab)
				if err := a.requestSourceLanguage(intent); err == nil || a.sourceLanguage.status != string(want) || a.sourceLanguage.pending != nil || a.sourceLanguage.owner != nil || q.outstanding.Load() != 0 {
					t.Fatal("budget refusal started work or lost exact failure state")
				}
				if busy {
					release()
					languageWait(t, task.done)
					languageReceive(t, q.calls)()
				}
				if a.sourceLanguage.status != string(want) {
					t.Fatal("abandoned completion overwrote admission refusal")
				}
				languageUnchanged(t, tab, before)
			})
		}
	}
}
