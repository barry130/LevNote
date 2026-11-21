package app

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"

	"levnote.local/native/core/document"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/core/engineruntime"
	"levnote.local/native/core/session"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
)

const languageSavedGoSource = "package p\r\n// 🙂 更新🙂\r\nfunc Solve(变量 int) int {\r\n  next := 变量\n  return next\r\n}\n"

func languageDirtySaveFixture(t *testing.T) (*App, *Tab, *languageQueue, string) {
	t.Helper()
	a, tab, queue, path := languageFixture(t, ".go", languageGoSource)
	languageMakeDirty(t, tab)
	return a, tab, queue, path
}

func languageMakeDirty(t *testing.T, tab *Tab) {
	t.Helper()
	b := tab.Doc.Buffer
	at := len(utf16.Encode([]rune("package p\r\n// 🙂 ")))
	if err := b.ReplaceUTF16(at, at+len("original"), "更新🙂"); err != nil {
		t.Fatal(err)
	}
	// Retain both a committed edit and a redo entry beyond that edit.
	if err := b.ReplaceUTF16(0, 0, " "); err != nil {
		t.Fatal(err)
	}
	if !b.Undo() || !b.CanUndo() || !b.CanRedo() || b.String() != languageSavedGoSource || !tab.Doc.Dirty() {
		t.Fatal("fixture lacks the expected dirty committed source and Undo/Redo state")
	}
	languageCorrectionSelections(t, tab)
}

func languageCheckSavedBytes(t *testing.T, tab *Tab, path string, old document.DiskToken, want []byte) {
	t.Helper()
	if got := mustReadFile(t, path); !bytes.Equal(got, want) {
		t.Fatalf("saved disk bytes = %q, want %q", got, want)
	}
	if next := tab.Doc.DiskToken(); !old.Valid() || !next.Valid() || tab.Doc.SameBaseline(old) || tab.Doc.Dirty() || tab.Doc.IsNew {
		t.Fatal("dirty save did not advance the verified baseline and become clean")
	} else if check, err := next.CheckDisk(context.Background()); err != nil || check != (document.DiskCheckResult{}) {
		t.Fatalf("saved baseline does not identify the written bytes: %+v %v", check, err)
	}
}

func languageCheckSaveOwner(t *testing.T, a *App, owner *sourceLanguageOwner, task *sourceLanguageTask, pending *sourceLanguagePending) {
	t.Helper()
	s := &a.sourceLanguage
	if s.owner != owner || s.task != task || s.pending != pending || a.sourceLanguageOwnerValid(owner, true) != nil {
		t.Fatalf("ordinary save retired still-valid language work: owner=%v task=%v pending=%v status=%s reason=%s",
			s.owner == owner, s.task == task, s.pending == pending, s.status, s.reason)
	}
}

func TestLanguageSavePreservesRunningAndCompleted(t *testing.T) {
	for _, name := range []string{"running", "completed"} {
		t.Run(name, func(t *testing.T) {
			a, tab, q, path := languageDirtySaveFixture(t)
			var task *sourceLanguageTask
			var ctx context.Context
			var release func()
			if name == "running" {
				started, finish := languageHold(t, a)
				release = finish
				task = languageRequest(t, a, languageIntent(code.ServiceSymbols))
				ctx = languageReceive(t, started)
			} else {
				languageComplete(t, a, q, languageIntent(code.ServiceSymbols))
				languageCheckSolveSymbol(t, a)
			}
			owner, result := a.sourceLanguage.owner, a.sourceLanguage.result
			before, old := languageState(tab), tab.Doc.DiskToken()
			if owner == nil || !before.dirty || !a.Save(tab, false) {
				t.Fatalf("real ordinary dirty Save failed: %s", a.Message)
			}
			languageCheckSavedBytes(t, tab, path, old, []byte(languageSavedGoSource))
			before.dirty = false
			languageUnchanged(t, tab, before)
			languageCheckSaveOwner(t, a, owner, task, nil)
			if task != nil {
				if ctx.Err() != nil || q.outstanding.Load() != 0 {
					t.Fatal("ordinary Save canceled or completed the held Analyzer")
				}
				release()
				languageWait(t, task.done)
				languageReceive(t, q.calls)()
			} else if !reflect.DeepEqual(a.sourceLanguage.result, result) {
				t.Fatal("ordinary Save replaced the completed payload")
			}
			languageCheckSolveSymbol(t, a)
			languageCheckSaveOwner(t, a, owner, nil, nil)
			languageUnchanged(t, tab, before)
		})
	}
}

func languageCheckSavedRecovery(t *testing.T, path, sourcePath string) []byte {
	t.Helper()
	raw := mustReadFile(t, path)
	snapshot, issues := session.Load(path)
	if len(issues) != 0 || len(snapshot.Tabs) != 1 || snapshot.Active != 0 {
		t.Fatalf("post-Save recovery could not be loaded: %+v %v", snapshot, issues)
	}
	entry := snapshot.Tabs[0]
	if entry.Path != sourcePath || entry.Title != filepath.Base(sourcePath) || entry.Draft != nil {
		t.Fatalf("post-Save recovery did not record the clean source: %+v", entry)
	}
	restored, err := session.RestoreTab(entry)
	if err != nil || restored.Document.Dirty() || restored.Document.Buffer.String() != languageSavedGoSource {
		t.Fatalf("post-Save recovery did not restore the newly saved bytes: %v", err)
	}
	return raw
}

func languageCloseEngine(t *testing.T, a *App, runner *controlledExporter) (*engineLane, *exportCall, <-chan struct{}) {
	t.Helper()
	lane := newEngineLane(runner)
	a.exporter, a.engineLane = runner, lane
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan struct{})
	if err := lane.submitPreview(&engineTask{ctx: ctx, cancel: cancel, id: engineipc.MarkdownHTML,
		source: engineruntime.Source{}, complete: func(engineruntime.Result, error) { close(done) }}); err != nil {
		t.Fatal(err)
	}
	return lane, awaitExportCall(t, runner), done
}

func TestLanguageSavePreservesWorkWhenCloseRefused(t *testing.T) {
	for _, tc := range []struct {
		name, stage                string
		window, pending, completed bool
	}{
		{name: "tab-prepare", stage: "prepare"},
		{name: "tab-uncommitted", stage: "publish"},
		{name: "window-prepare-completed", stage: "prepare", window: true, completed: true},
		{name: "window-uncommitted-latest", stage: "publish", window: true, pending: true},
		{name: "window-engine-cleanup", stage: "engine", window: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, tab, q, path := languageDirtySaveFixture(t)
			a.Persist()
			recovery := filepath.Join(a.configDir, "session.json")
			dirtyRecovery := mustReadFile(t, recovery)
			dirtySnapshot, issues := session.Load(recovery)
			if len(issues) != 0 || len(dirtySnapshot.Tabs) != 1 || dirtySnapshot.Tabs[0].Draft == nil || *dirtySnapshot.Tabs[0].Draft != languageSavedGoSource {
				t.Fatal("expected a persisted dirty draft before asking to close")
			}
			var task *sourceLanguageTask
			var started <-chan context.Context
			var ctx context.Context
			var release func()
			if tc.completed {
				languageComplete(t, a, q, languageIntent(code.ServiceSymbols))
				languageCheckSolveSymbol(t, a)
			} else {
				started, release = languageHold(t, a)
				operation := code.ServiceSymbols
				if tc.pending {
					operation = code.ServiceDiagnostics
				}
				task = languageRequest(t, a, languageIntent(operation))
				ctx = languageReceive(t, started)
				if tc.pending && languageRequest(t, a, languageIntent(code.ServiceSymbols)) != task {
					t.Fatal("latest operation started a second physical task")
				}
			}
			owner, pending, result := a.sourceLanguage.owner, a.sourceLanguage.pending, a.sourceLanguage.result
			before, old := languageState(tab), tab.Doc.DiskToken()
			if owner == nil || (pending != nil) != tc.pending {
				t.Fatal("missing expected language owner/descriptor")
			}
			decisionCalls, commitCalls := 0, 0
			a.closeDialog = func(candidate *Tab) (closeDecision, error) {
				if candidate != tab || !candidate.Doc.Dirty() {
					t.Fatal("close did not offer to save the actual dirty tab")
				}
				decisionCalls++
				return closeSave, nil
			}
			failure := errors.New("controlled close " + tc.stage + " refusal")
			var savedRecovery []byte
			a.prepareSessionState = func(destination string, snapshot session.Snapshot) (preparedSessionState, error) {
				if destination != recovery || savedRecovery != nil || (len(snapshot.Tabs) == 1) != tc.window {
					t.Fatal("unexpected close candidate")
				}
				languageCheckSavedBytes(t, tab, path, old, []byte(languageSavedGoSource))
				savedRecovery = languageCheckSavedRecovery(t, recovery, path)
				if tc.stage == "prepare" {
					return document.PreparePrivateStateTo(destination, func(w io.Writer) error {
						_, _ = io.WriteString(w, "partial candidate")
						return failure
					})
				}
				prepared, err := session.PrepareSave(destination, snapshot)
				if err != nil {
					return nil, err
				}
				return closeCommitFault{preparedSessionState: prepared, commit: func(check func() error) error {
					commitCalls++
					return prepared.Commit(func() error {
						if err := check(); err != nil {
							return err
						}
						return failure
					})
				}}, nil
			}
			var cleanup chan error
			var engineDone <-chan struct{}
			var lane *engineLane
			if tc.stage == "engine" {
				runner := newControlledExporter()
				runner.closeErr = failure
				cleanup = make(chan error, 1)
				runner.onClose = func() { cleanup <- ctx.Err() }
				lane, _, engineDone = languageCloseEngine(t, a, runner)
			}
			closed := false
			if tc.window {
				closed = a.CanClose()
			} else {
				closed = a.CloseTab(0)
			}
			if closed || decisionCalls != 1 || savedRecovery == nil || !strings.Contains(a.Message, failure.Error()) ||
				(commitCalls == 1) != (tc.stage == "publish") || a.shuttingDown.Load() || a.Current() != tab || len(a.Tabs) != 1 {
				t.Fatalf("close refusal lost its stage or live tab: closed=%v decision=%d commit=%d message=%s", closed, decisionCalls, commitCalls, a.Message)
			}
			if bytes.Equal(savedRecovery, dirtyRecovery) || !bytes.Equal(mustReadFile(t, recovery), savedRecovery) {
				t.Fatal("refused close did not preserve the actual post-Save recovery bytes")
			}
			requireNoCloseTemps(t, recovery)
			languageCheckSavedBytes(t, tab, path, old, []byte(languageSavedGoSource))
			before.dirty = false
			languageUnchanged(t, tab, before)
			languageCheckSaveOwner(t, a, owner, task, pending)
			if cleanup != nil {
				if err := languageReceive(t, cleanup); err != nil || !errors.Is(a.enginePoison, failure) {
					t.Fatalf("native cleanup canceled language or lost its own failure: %v %v", err, a.enginePoison)
				}
				languageWait(t, lane.closed)
				languageWait(t, engineDone)
			}
			if task != nil {
				if (ctx.Err() != nil) != tc.pending || q.outstanding.Load() != 0 {
					t.Fatal("refused close changed cancellation or queued an early notice")
				}
				release()
				languageWait(t, task.done)
				languageReceive(t, q.calls)()
				if tc.pending {
					latest := a.sourceLanguage.task
					if latest == nil || latest == task || latest.id != owner.request {
						t.Fatal("retirement of the old task lost the pending symbols request")
					}
					languageReceive(t, started)
					languageWait(t, latest.done)
					languageReceive(t, q.calls)()
				}
			} else if !reflect.DeepEqual(a.sourceLanguage.result, result) {
				t.Fatal("refused close changed the completed language payload")
			}
			languageCheckSolveSymbol(t, a)
			languageCheckSaveOwner(t, a, owner, nil, nil)
			languageUnchanged(t, tab, before)
		})
	}
}

func TestLanguageSaveMetadataRetiresInvalidOwners(t *testing.T) {
	for _, name := range []string{"different-path-save-as", "same-path-save-as", "same-path-adopted-failure", "encoding", "bom", "readonly", "name-rebound", "stale-name-restored", "case-only-path", "text-aba", "attachment-aba"} {
		t.Run(name, func(t *testing.T) {
			source := languageGoSource
			if name == "encoding" {
				// Keep BOM unchanged while changing only the encoding to UTF-16LE.
				source = "\xef\xbb\xbf" + source
			}
			a, tab, q, path := languageFixture(t, ".go", source)
			languageMakeDirty(t, tab)
			casePath := filepath.Join(filepath.Dir(path), "Source.go")
			if name == "name-rebound" {
				tab.Name = "scratch.go"
			} else if name == "case-only-path" {
				// Keep the name stable so only the exact path spelling invalidates the seal.
				tab.Name = filepath.Base(casePath)
			}
			started, release := languageHold(t, a)
			task := languageRequest(t, a, languageIntent(code.ServiceSymbols))
			ctx := languageReceive(t, started)
			owner := a.sourceLanguage.owner
			saveAs := name == "different-path-save-as" || name == "same-path-save-as"
			destination, dialogCalls := path, 0
			if name == "different-path-save-as" {
				destination = filepath.Join(filepath.Dir(path), "copy.go")
			}
			a.saveDialog = func(*Tab) (string, error) { dialogCalls++; return destination, nil }
			switch name {
			case "same-path-adopted-failure":
				a.saveDocument = func(doc *document.Document, target string) error {
					if target != "" || doc.Path != path {
						t.Fatal("same-path adoption fixture lost the ordinary Save branch")
					}
					if err := os.WriteFile(path, []byte(languageSavedGoSource), 0600); err != nil {
						t.Fatal(err)
					}
					return &document.PublicationError{Path: path, MetadataAdopted: true, Cause: document.ErrConflict}
				}
			case "encoding":
				tab.Doc.Encoding = document.UTF16LE
			case "bom":
				tab.Doc.BOM = true
			case "readonly", "case-only-path":
				a.saveDocument = func(doc *document.Document, target string) error {
					if target != "" {
						t.Fatal("metadata transition unexpectedly went through SaveAs")
					}
					if err := doc.Save(); err != nil {
						return err
					}
					if name == "readonly" {
						if err := os.Chmod(path, 0400); err != nil {
							return err
						}
						doc.ReadOnly = true
					} else {
						if err := os.Rename(path, casePath); err != nil {
							return err
						}
						doc.Path = casePath
					}
					return nil
				}
			case "stale-name-restored":
				tab.Name = "stale.go"
			case "text-aba":
				if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, " "); err != nil || !tab.Doc.Buffer.Undo() {
					t.Fatalf("could not change and undo source text: %v", err)
				}
			case "attachment-aba":
				tab.Editor.SetBuffer(text.New("other attachment"))
				tab.Editor.SetBuffer(tab.Doc.Buffer)
			}
			adopted := name == "same-path-adopted-failure"
			if ok := a.Save(tab, saveAs); ok == adopted || (dialogCalls == 1) != saveAs {
				t.Fatalf("unexpected Save outcome: saveAs=%v outcome=%v dialog=%d message=%s", saveAs, ok, dialogCalls, a.Message)
			}
			if adopted && (!tab.Doc.Dirty() || !tab.Conflict || !strings.Contains(a.Message, "元数据已采用目标")) {
				t.Fatalf("adopted publication failure lost its draft or warning: %s", a.Message)
			}
			if saveAs && (tab.Doc.Path != destination || tab.Doc.Dirty() || string(mustReadFile(t, destination)) != languageSavedGoSource) {
				t.Fatal("requested SaveAs did not publish the real selected path")
			}
			if name == "readonly" && (!tab.Doc.ReadOnly || !tab.Editor.ReadOnly) || name == "case-only-path" && tab.Doc.Path != casePath || tab.Name != filepath.Base(tab.Doc.Path) {
				t.Fatal("language retirement bypassed metadata reconciliation")
			}
			if a.sourceLanguage.owner != nil || a.sourceLanguage.pending != nil || a.sourceLanguage.task != task || ctx.Err() == nil ||
				a.sourceLanguage.status != sourceLanguageStale || a.sourceLanguageOwnerValid(owner, false) == nil {
				t.Fatal("Save failed to retire the previous language seal")
			}
			release()
			languageWait(t, task.done)
			languageReceive(t, q.calls)()
			if a.sourceLanguage.task != nil || a.sourceLanguage.owner != nil || !reflect.DeepEqual(a.sourceLanguage.result, code.ServiceResult{}) {
				t.Fatal("old language result published after save invalidation")
			}
		})
	}
}

func TestLanguageSaveCanceledAndNonAdoptedFailurePreserveWork(t *testing.T) {
	for _, name := range []string{"dialog-canceled", "dialog-error", "non-adopted-publication", "disk-conflict"} {
		t.Run(name, func(t *testing.T) {
			a, tab, q, path := languageDirtySaveFixture(t)
			started, release := languageHold(t, a)
			task := languageRequest(t, a, languageIntent(code.ServiceSymbols))
			ctx := languageReceive(t, started)
			owner := a.sourceLanguage.owner
			before, baseline := languageState(tab), tab.Doc.DiskToken()
			as := name != "disk-conflict"
			saveCalls := 0
			failure := errors.New("controlled save failure")
			if as {
				a.saveDialog = func(*Tab) (string, error) {
					switch name {
					case "dialog-canceled":
						return "", nil
					case "dialog-error":
						return "", failure
					default:
						return path, nil
					}
				}
				a.saveDocument = func(_ *document.Document, target string) error {
					saveCalls++
					return &document.PublicationError{Path: target, MetadataAdopted: false, Cause: failure}
				}
			} else if err := os.WriteFile(path, []byte("outside edit"), 0600); err != nil {
				t.Fatal(err)
			}
			if a.Save(tab, as) || (saveCalls == 1) != (name == "non-adopted-publication") || !tab.Doc.SameBaseline(baseline) || ctx.Err() != nil {
				t.Fatalf("failed save altered the baseline or canceled valid work: %s", a.Message)
			}
			if name == "disk-conflict" && !tab.Conflict || name == "non-adopted-publication" && !strings.Contains(a.Message, "文档元数据保持原状态") {
				t.Fatalf("save failure was not reported as expected: %s", a.Message)
			}
			languageCheckSaveOwner(t, a, owner, task, nil)
			languageUnchanged(t, tab, before)
			release()
			languageWait(t, task.done)
			languageReceive(t, q.calls)()
			languageCheckSolveSymbol(t, a)
			languageCheckSaveOwner(t, a, owner, nil, nil)
		})
	}
}

func TestLanguageSaveDirtyBOMRoundTripsPreserveCompletedResult(t *testing.T) {
	for _, encoding := range []document.Encoding{document.UTF8, document.UTF16LE} {
		t.Run(string(encoding), func(t *testing.T) {
			// Encode independent literal source oracles with the standard library.
			// The mixed LF/CRLF, emoji surrogate pair and non-ASCII identifiers must survive.
			encode := func(source string) []byte {
				if encoding == document.UTF8 {
					return append([]byte{0xef, 0xbb, 0xbf}, source...)
				}
				dst := []byte{0xff, 0xfe}
				for _, unit := range utf16.Encode([]rune(source)) {
					dst = binary.LittleEndian.AppendUint16(dst, unit)
				}
				return dst
			}
			original, want := encode(languageGoSource), encode(languageSavedGoSource)
			root := t.TempDir()
			path := filepath.Join(root, "encoded.go")
			if err := os.WriteFile(path, original, 0600); err != nil {
				t.Fatal(err)
			}
			a := newTestApp(t, filepath.Join(root, "config"))
			a.Open(path)
			tab := a.Current()
			if tab == nil || tab.Doc.Encoding != encoding || !tab.Doc.BOM || tab.Doc.Buffer.String() != languageGoSource {
				t.Fatal("encoded Go fixture did not open with its original metadata and source")
			}
			q := newLanguageQueue()
			a.sourceLanguageHooks = &sourceLanguageHooks{dispatch: q.dispatch}
			languageMakeDirty(t, tab)
			languageComplete(t, a, q, languageIntent(code.ServiceSymbols))
			languageCheckSolveSymbol(t, a)
			owner, result := a.sourceLanguage.owner, a.sourceLanguage.result
			before, old := languageState(tab), tab.Doc.DiskToken()
			if !before.dirty || bytes.Equal(want, original) || !bytes.Equal(mustReadFile(t, path), original) {
				t.Fatal("fixture did not reach a dirty Save with untouched original bytes")
			}
			if !a.Save(tab, false) {
				t.Fatalf("real dirty encoded Save failed: %s", a.Message)
			}
			languageCheckSavedBytes(t, tab, path, old, want)
			before.dirty = false
			languageUnchanged(t, tab, before)
			languageCheckSaveOwner(t, a, owner, nil, nil)
			if !reflect.DeepEqual(a.sourceLanguage.result, result) {
				t.Fatal("successful dirty encoded save replaced the analysis payload")
			}
			reopened, err := document.Open(path)
			if err != nil || reopened.Encoding != encoding || !reopened.BOM || reopened.Buffer.String() != languageSavedGoSource || reopened.Dirty() {
				t.Fatalf("encoded bytes did not round-trip in the original encoding: %v", err)
			}
		})
	}
}

func TestLanguageSaveAcceptedNativeCloseAndDirectShutdownOrdering(t *testing.T) {
	for _, name := range []string{"native-close", "direct-shutdown"} {
		t.Run(name, func(t *testing.T) {
			a, tab, q, path := languageDirtySaveFixture(t)
			started, release := languageHold(t, a)
			task := languageRequest(t, a, languageIntent(code.ServiceSymbols))
			ctx := languageReceive(t, started)
			runner := newControlledExporter()
			type observation struct{ canceled, finished, shutdown bool }
			observed := make(chan observation, 1)
			runner.onClose = func() {
				o := observation{canceled: ctx.Err() != nil, shutdown: a.shuttingDown.Load()}
				select {
				case <-task.done:
					o.finished = true
				default:
				}
				observed <- o
			}
			lane, call, engineDone := languageCloseEngine(t, a, runner)
			if name == "native-close" {
				owner := a.sourceLanguage.owner
				a.closeDialog = func(*Tab) (closeDecision, error) { return closeSave, nil }
				committed := false
				a.prepareSessionState = func(filename string, candidate session.Snapshot) (preparedSessionState, error) {
					prepared, err := session.PrepareSave(filename, candidate)
					if err != nil {
						return nil, err
					}
					return closeCommitFault{preparedSessionState: prepared, commit: func(check func() error) error {
						select {
						case <-lane.closed:
						default:
							t.Error("native recovery commit preceded engine cleanup")
						}
						if ctx.Err() != nil || a.sourceLanguage.owner != owner || a.Current() != tab {
							t.Error("native precommit cleanup did not preserve the still-open language owner")
						}
						err := prepared.Commit(check)
						committed = err == nil
						return err
					}}, nil
				}
				// This is the App sequence called by native OnClose and OnClosed.
				if !a.CanClose() || !committed || a.shuttingDown.Load() || len(a.Tabs) != 0 ||
					ctx.Err() == nil || a.sourceLanguage.owner != nil || a.sourceLanguage.task != task {
					t.Fatalf("native close did not retire at tab removal after the commit: %s", a.Message)
				}
				if got := languageReceive(t, observed); got != (observation{}) {
					t.Errorf("native precommit cleanup observation = %+v", got)
				}
				languageCheckSavedRecovery(t, filepath.Join(a.configDir, "session.json"), path)
			}
			joined := make(chan struct{})
			go func() { a.Shutdown(); close(joined) }()
			if name == "direct-shutdown" {
				if got := languageReceive(t, observed); got != (observation{canceled: true, shutdown: true}) {
					t.Errorf("direct Shutdown cleanup observation = %+v", got)
				}
			}
			languageWait(t, a.shutdownDone)
			select {
			case <-joined:
				t.Error("Shutdown returned while the Analyzer was still held")
			default:
			}
			release()
			languageWait(t, joined)
			languageWait(t, task.done)
			languageWait(t, lane.closed)
			languageWait(t, engineDone)
			if call.ctx.Err() == nil || a.sourceLanguage.task != nil || a.sourceLanguage.owner != nil || a.sourceLanguage.pending != nil || q.outstanding.Load() != 1 {
				t.Fatal("shutdown did not cancel and join both owned tasks while their UI notice was queued")
			}
			languageReceive(t, q.calls)()
			if a.sourceLanguage.status != string(code.ServiceCanceled) || !reflect.DeepEqual(a.sourceLanguage.result, code.ServiceResult{}) {
				t.Fatal("late notice published after shutdown")
			}
		})
	}
}
