package app

import (
	"context"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"unicode/utf16"
	"unsafe"

	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
)

var languageCorrectionOperations = [...]code.ServiceOperation{
	code.ServiceDiagnostics, code.ServiceCompletion, code.ServiceDefinition,
	code.ServiceReferences, code.ServiceInfo, code.ServiceRename, code.ServiceSymbols,
}

func languageOperationView(operation code.ServiceOperation) (string, code.ServiceOperation) {
	// Each call allocates a large, independent backing string. A legal operation
	// is a short interior view, while an unknown operation can be the whole value.
	const padding = 1 << 19
	var backing strings.Builder
	backing.Grow(2*padding + len(operation))
	backing.WriteString(strings.Repeat("<", padding))
	backing.WriteString(string(operation))
	backing.WriteString(strings.Repeat(">", padding))
	caller := backing.String()
	if operation == "" {
		return caller, code.ServiceOperation(caller)
	}
	return caller, code.ServiceOperation(caller[padding : padding+len(operation)])
}

func languageOperationBackedBy(operation code.ServiceOperation, caller string) bool {
	if operation == "" {
		return false
	}
	// Read-only address comparison. Keep both strings live through the check.
	// No pointer is reconstituted or dereferenced and no GC behavior is assumed.
	start := uintptr(unsafe.Pointer(unsafe.StringData(caller)))
	address := uintptr(unsafe.Pointer(unsafe.StringData(string(operation))))
	inside := address >= start && address-start < uintptr(len(caller))
	runtime.KeepAlive(operation)
	runtime.KeepAlive(caller)
	return inside
}

func languageCheckOperationStorage(t *testing.T, retained, input, want code.ServiceOperation, caller string) {
	t.Helper()
	if !languageOperationBackedBy(input, caller) {
		t.Fatal("operation fixture does not point into its own backing string")
	}
	if languageOperationBackedBy(retained, caller) {
		t.Errorf("retained operation (%d bytes) points into %d-byte caller storage", len(retained), len(caller))
	}
	if retained != want {
		t.Errorf("retained operation has %d bytes, want canonical %q", len(retained), want)
	}
}

func languageCorrectionSelections(t *testing.T, tab *Tab) {
	t.Helper()
	if err := tab.Doc.Buffer.SetSelectionSet(text.SelectionSet{
		Ranges: []text.Range{{Anchor: 5, Head: 1}, {Anchor: 8, Head: 8}}, Primary: 1,
	}); err != nil {
		t.Fatal(err)
	}
}

func languageCheckUnsupportedOperation(t *testing.T, a *App) {
	t.Helper()
	s := &a.sourceLanguage
	want := code.ServiceResult{Status: code.ServiceUnsupported, Reason: "operation"}
	if s.status != string(want.Status) || s.reason != want.Reason || s.failure != nil || s.owner != nil ||
		!reflect.DeepEqual(s.result, want) {
		t.Fatalf("real unsupported operation outcome = %+v", s)
	}
}

func TestLanguageIntentCanonicalIdleAdmissionStorage(t *testing.T) {
	for _, operation := range append([]code.ServiceOperation{""}, languageCorrectionOperations[:]...) {
		name := string(operation)
		if name == "" {
			name = "unknown"
		}
		t.Run(name, func(t *testing.T) {
			a, tab, q, _ := languageFixture(t, ".go", languageGoSource)
			before := languageState(tab)
			caller, input := languageOperationView(operation)
			started, release := languageHold(t, a)
			launches := 0
			// Observe the actual admitted idle intent before the worker takes its
			// request. Forward to the normal launcher and real Analyzer.
			if err := a.admitSourceLanguageRequest(languageIntent(input), func(owner *sourceLanguageOwner, intent sourceLanguageIntent, source string, dispatch func(func())) {
				launches++
				languageCheckOperationStorage(t, intent.operation, input, operation, caller)
				a.launchSourceLanguage(owner, intent, source, dispatch)
			}); err != nil {
				t.Fatal(err)
			}
			task := a.sourceLanguage.task
			if launches != 1 || task == nil || a.sourceLanguage.pending != nil {
				t.Fatal("idle admission did not launch one task")
			}
			languageReceive(t, started)
			release()
			languageWait(t, task.done)
			languageReceive(t, q.calls)()
			if operation == "" {
				languageCheckUnsupportedOperation(t, a)
			}
			languageUnchanged(t, tab, before)
		})
	}
}

func TestLanguageIntentCanonicalIdleResults(t *testing.T) {
	for _, operation := range append([]code.ServiceOperation{""}, languageCorrectionOperations[:]...) {
		name := string(operation)
		if name == "" {
			name = "unknown"
		}
		t.Run(name, func(t *testing.T) {
			source := languageGoSource
			if operation == code.ServiceDiagnostics {
				source += "func broken("
			}
			a, tab, q, _ := languageFixture(t, ".go", source)
			languageCorrectionSelections(t, tab)
			before := languageState(tab)
			caller, input := languageOperationView(operation)
			intent := languageIntent(input)
			intent.byteOffset = strings.LastIndex(source, "变量")
			intent.prefix, intent.newName = "ne", "value"
			if operation == code.ServiceInfo {
				intent.byteOffset = strings.Index(source, "Solve")
			}
			languageComplete(t, a, q, intent)
			runtime.KeepAlive(caller)
			if q.maximum.Load() != 1 || q.outstanding.Load() != 0 || a.sourceLanguage.task != nil || a.sourceLanguage.pending != nil {
				t.Fatal("idle invocation did not consume one physical task and one queued notice")
			}
			languageUnchanged(t, tab, before)
			if operation == "" {
				languageCheckUnsupportedOperation(t, a)
				return
			}
			result := a.sourceLanguage.result
			if operation == code.ServiceDiagnostics {
				if result.Status != code.ServiceInvalid || result.Reason != "source" || len(result.Diagnostics) == 0 || validateSourceLanguageResult(result) != nil {
					t.Fatalf("malformed Go diagnostics = %+v", result)
				}
				return
			}
			if result.Status != code.ServiceComplete || a.sourceLanguageOwnerValid(a.sourceLanguage.owner, true) != nil {
				t.Fatalf("idle %s did not publish a complete owned result: %+v", operation, result)
			}
			declaration := strings.Index(source, "变量")
			use := strings.LastIndex(source, "变量")
			checkSpan := func(span code.StructureSpan, at int) {
				t.Helper()
				end := at + len("变量")
				if span.Bytes.Start != at || span.Bytes.End != end ||
					span.UTF16.Start != len(utf16.Encode([]rune(source[:at]))) || span.UTF16.End != len(utf16.Encode([]rune(source[:end]))) {
					t.Fatalf("symbol span = %+v, want bytes %d:%d and their UTF-16 endpoints", span, at, end)
				}
			}
			switch operation {
			case code.ServiceSymbols:
				languageCheckSolveSymbol(t, a)
			case code.ServiceCompletion:
				if want := []code.ServiceCompletionItem{{Text: "next", Kind: "file-word"}}; !reflect.DeepEqual(result.Completion, want) {
					t.Fatalf("literal-prefix completion = %+v, want %+v", result.Completion, want)
				}
			case code.ServiceDefinition:
				if result.Definition == nil || result.Definition.Name != "变量" || result.Definition.Kind != "parameter" {
					t.Fatalf("definition = %+v", result.Definition)
				}
				checkSpan(result.Definition.Span, declaration)
			case code.ServiceReferences:
				if len(result.References) != 1 {
					t.Fatalf("use-site references = %+v", result.References)
				}
				checkSpan(result.References[0], use)
			case code.ServiceInfo:
				if result.Info == nil || result.Info.Symbol.Name != "Solve" || result.Info.Signature != "func Solve(变量 int) int" || result.Info.Documentation != "// 🙂 original" {
					t.Fatalf("symbol info = %+v", result.Info)
				}
			case code.ServiceRename:
				if result.NoOp || len(result.Edits) != 2 {
					t.Fatalf("rename plan = %+v", result)
				}
				for i, at := range []int{declaration, use} {
					edit := result.Edits[i]
					checkSpan(edit.Span, at)
					if edit.OldText != "变量" || edit.NewText != "value" {
						t.Fatalf("rename edit %d = %+v", i, edit)
					}
				}
			}
		})
	}
}

func languageCheckSolveSymbol(t *testing.T, a *App) {
	t.Helper()
	if a.sourceLanguage.status == string(code.ServiceComplete) {
		for _, symbol := range a.sourceLanguage.result.Symbols {
			if symbol.Name == "Solve" && symbol.Kind == "func" {
				return
			}
		}
	}
	t.Fatalf("real Solve symbol missing: status=%s reason=%s result=%+v", a.sourceLanguage.status, a.sourceLanguage.reason, a.sourceLanguage.result)
}

func TestLanguageIntentCanonicalPendingStorageAndReplacement(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", languageGoSource)
	if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, " "); err != nil {
		t.Fatal(err)
	}
	if !tab.Doc.Buffer.Undo() || !tab.Doc.Buffer.CanRedo() {
		t.Fatal("missing real redo entry")
	}
	languageCorrectionSelections(t, tab)
	before := languageState(tab)
	starts, released := make(chan uint64, 16), make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(released) }) }
	t.Cleanup(release)
	var count, active, maximum atomic.Int32
	a.sourceLanguageHooks.beforeAnalyze = func(_ context.Context, id uint64) {
		count.Add(1)
		languageMax(&maximum, active.Add(1))
		starts <- id
		<-released
	}
	a.sourceLanguageHooks.afterAnalyze = func(uint64) { active.Add(-1) }
	first := languageRequest(t, a, languageIntent(code.ServiceDiagnostics))
	if languageReceive(t, starts) != first.id {
		t.Fatal("wrong initial Analyzer invocation")
	}
	target := a.sourceLanguage.owner.target
	var prior *sourceLanguagePending
	admit := func(t *testing.T, operation code.ServiceOperation) {
		t.Helper()
		caller, input := languageOperationView(operation)
		if err := a.requestSourceLanguage(languageIntent(input)); err != nil {
			t.Fatal(err)
		}
		pending := a.sourceLanguage.pending
		if pending == nil || pending == prior || pending.owner != a.sourceLanguage.owner ||
			!reflect.DeepEqual(pending.owner.target, target) || a.sourceLanguage.task != first || count.Load() != 1 || q.outstanding.Load() != 0 {
			t.Fatal("pending admission did not replace exactly one descriptor on the existing target and slot")
		}
		first.mu.Lock()
		requestCleared := first.request == (code.ServiceRequest{})
		first.mu.Unlock()
		if !requestCleared {
			t.Fatal("a request snapshot was retained in the occupied task slot")
		}
		languageCheckOperationStorage(t, pending.intent.operation, input, operation, caller)
		prior = pending
	}
	t.Run("huge-unknown", func(t *testing.T) { admit(t, "") })
	for _, operation := range languageCorrectionOperations {
		t.Run(string(operation), func(t *testing.T) { admit(t, operation) })
	}
	if prior == nil || prior.intent.operation != code.ServiceSymbols {
		t.Fatal("latest pending is not symbols")
	}
	release()
	languageWait(t, first.done)
	if a.sourceLanguage.pending != prior || a.sourceLanguage.status != sourceLanguageRunning || q.outstanding.Load() != 1 {
		t.Fatal("retiring Analyzer changed UI state before its queued callback")
	}
	oldNotice := languageReceive(t, q.calls)
	oldNotice()
	second := a.sourceLanguage.task
	if second == nil || second.id != prior.owner.request || second == first || a.sourceLanguage.pending != nil || languageReceive(t, starts) != second.id {
		t.Fatal("latest symbols did not take the one retired physical slot")
	}
	oldNotice()
	if a.sourceLanguage.task != second || a.sourceLanguage.status != sourceLanguageRunning {
		t.Fatal("old callback altered the running successor")
	}
	languageWait(t, second.done)
	languageReceive(t, q.calls)()
	oldNotice()
	languageCheckSolveSymbol(t, a)
	if a.sourceLanguage.owner != prior.owner || a.sourceLanguage.task != nil || count.Load() != 2 || maximum.Load() != 1 || active.Load() != 0 ||
		q.maximum.Load() != 1 || q.outstanding.Load() != 0 {
		t.Fatal("latest outcome or physical execution/notice bound was lost")
	}
	languageUnchanged(t, tab, before)
}

func TestLanguageIntentCanonicalPendingUnsupportedOrRetired(t *testing.T) {
	for _, retire := range []bool{false, true} {
		name := "execute-unsupported"
		if retire {
			name = "retire-unsupported"
		}
		t.Run(name, func(t *testing.T) {
			a, tab, q, _ := languageFixture(t, ".go", languageGoSource)
			before := languageState(tab)
			started, release := languageHold(t, a)
			first := languageRequest(t, a, languageIntent(code.ServiceSymbols))
			languageReceive(t, started)
			caller, input := languageOperationView("")
			if err := a.requestSourceLanguage(languageIntent(input)); err != nil {
				t.Fatal(err)
			}
			pending := a.sourceLanguage.pending
			if pending == nil || a.sourceLanguage.task != first {
				t.Fatal("unknown operation is not a bounded pending intent")
			}
			languageCheckOperationStorage(t, pending.intent.operation, input, "", caller)
			if retire {
				a.cancelSourceLanguage()
				if a.sourceLanguage.pending != nil || a.sourceLanguage.owner != nil || a.sourceLanguage.task != first {
					t.Fatal("retirement retained an intent or released a still-occupied slot")
				}
			}
			release()
			languageWait(t, first.done)
			oldNotice := languageReceive(t, q.calls)
			oldNotice()
			if retire {
				if a.sourceLanguage.status != string(code.ServiceCanceled) {
					t.Fatal("abandoned request published after retirement")
				}
			} else {
				task := a.sourceLanguage.task
				if task == nil || task.id != pending.owner.request {
					t.Fatal("pending unsupported intent bypassed the real worker")
				}
				languageReceive(t, started)
				languageWait(t, task.done)
				languageReceive(t, q.calls)()
				oldNotice()
				languageCheckUnsupportedOperation(t, a)
			}
			if a.sourceLanguage.pending != nil || a.sourceLanguage.task != nil || q.maximum.Load() != 1 || q.outstanding.Load() != 0 {
				t.Fatal("retirement leaked a descriptor/task or an extra callback")
			}
			languageUnchanged(t, tab, before)
		})
	}
}
