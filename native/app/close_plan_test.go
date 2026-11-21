package app

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levnote.local/native/core/document"
	"levnote.local/native/core/session"
	"levnote.local/native/core/text"
)

func closePlanFixture(t *testing.T) (*App, *Tab, *controlledExporter, string, []byte) {
	t.Helper()
	a, tab, runner, _, _ := workerApp(t)
	tab.Editor.ReplaceText(nil, "live dirty draft🙂")
	a.Persist()
	filename := filepath.Join(a.configDir, "session.json")
	before := mustReadFile(t, filename)
	a.closeDialog = func(*Tab) (closeDecision, error) { return closeDiscard, nil }
	return a, tab, runner, filename, before
}

func requireOldCloseRecovery(t *testing.T, a *App, filename string, before []byte) {
	t.Helper()
	if got := mustReadFile(t, filename); !bytes.Equal(got, before) {
		t.Fatalf("uncommitted close changed original recovery: %q", got)
	}
	if len(a.Tabs) != 1 {
		t.Fatal("failed close removed a live tab")
	}
	saved, issues := session.Load(filename)
	if len(issues) != 0 || len(saved.Tabs) != 1 {
		t.Fatalf("old recovery was not loadable: %#v %v", saved, issues)
	}
	restored, err := session.RestoreTab(saved.Tabs[0])
	if err != nil || restored.Document.Path != "" || !restored.Document.IsNew || !strings.Contains(restored.Document.Buffer.String(), "live dirty draft🙂") {
		t.Fatalf("old dirty draft could not actually restore: %#v %v", restored, err)
	}
}

func requireNoCloseTemps(t *testing.T, filename string) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(filepath.Dir(filename), ".levnote-save-*"))
	if err != nil || len(files) != 0 {
		t.Fatalf("close leaked a temporary file: %v %v", files, err)
	}
}

func TestClosePreparedSessionWriteFailureDoesNotCancelWork(t *testing.T) {
	a, tab, runner, filename, before := closePlanFixture(t)
	a.exportCurrent()
	call, job := awaitExportCall(t, runner), a.exportJob
	failure := errors.New("injected private-state stream failure")
	a.prepareSessionState = func(path string, _ session.Snapshot) (preparedSessionState, error) {
		return document.PreparePrivateStateTo(path, func(w io.Writer) error {
			_, _ = io.WriteString(w, "partial candidate")
			return failure
		})
	}
	if a.CanClose() {
		t.Fatal("failed temp write accepted close")
	}
	select {
	case <-call.ctx.Done():
		t.Fatal("failed temp write canceled the original job")
	default:
	}
	if a.exportJob != job || a.Current() != tab || !strings.Contains(a.Message, failure.Error()) || a.persistPending {
		t.Fatal("failed prepare changed job ownership or hid the error", a.Message)
	}
	requireOldCloseRecovery(t, a, filename, before)
	requireNoCloseTemps(t, filename)
}

func TestClosePreparedSessionJoinFailureKeepsExactRecoveryAndStickyCauses(t *testing.T) {
	a, _, runner, filename, before := closePlanFixture(t)
	packageFailure, runnerFailure := errors.New("package cleanup refused"), errors.New("runner cleanup refused")
	a.enginePoison = packageFailure
	runner.closeErr = runnerFailure
	if a.CanClose() || !errors.Is(a.enginePoison, packageFailure) || !errors.Is(a.enginePoison, runnerFailure) {
		t.Fatalf("cleanup failure lost causes or closed: %v %s", a.enginePoison, a.Message)
	}
	requireOldCloseRecovery(t, a, filename, before)
	requireNoCloseTemps(t, filename)
	runner.closeErr = nil // Its next idempotent Close no longer reports a new error.
	if a.CanClose() || !errors.Is(a.enginePoison, packageFailure) || !errors.Is(a.enginePoison, runnerFailure) {
		t.Fatal("second close forgot the failed cleanup")
	}
	requireOldCloseRecovery(t, a, filename, before)
}

func TestClosePreparedSessionOwnerChangesAbortBeforePublication(t *testing.T) {
	for _, change := range []string{"dialog-view", "dialog-buffer", "dialog-order", "join-version", "join-view", "join-path", "join-draft-notice", "capture-owner"} {
		t.Run(change, func(t *testing.T) {
			a, tab, runner, filename, before := closePlanFixture(t)
			view := &sessionDraftFixture{}
			tab.View = view
			switch change {
			case "dialog-view":
				a.closeDialog = func(*Tab) (closeDecision, error) { tab.View = &sessionDraftFixture{}; return closeDiscard, nil }
			case "dialog-buffer":
				a.closeDialog = func(*Tab) (closeDecision, error) {
					tab.Doc.Buffer = text.New("new buffer")
					tab.Editor.Buffer = tab.Doc.Buffer
					return closeDiscard, nil
				}
			case "dialog-order":
				a.closeDialog = func(*Tab) (closeDecision, error) { a.Tabs = append(a.Tabs, tab); return closeDiscard, nil }
			case "join-version":
				runner.onClose = func() { tab.Editor.ReplaceText(nil, "owner changed") }
			case "join-view":
				runner.onClose = func() { tab.View = &sessionDraftFixture{} }
			case "join-path":
				runner.onClose = func() { tab.Doc.Path = filepath.Join(filepath.Dir(filename), "changed.md") }
			case "join-draft-notice":
				runner.onClose = func() { a.editorHost(tab).DraftRecoveryChanged(view) }
			case "capture-owner":
				a.prepareSessionState = func(path string, snapshot session.Snapshot) (preparedSessionState, error) {
					prepared, err := session.PrepareSave(path, snapshot)
					tab.View = &sessionDraftFixture{}
					return prepared, err
				}
			}
			if a.CanClose() || !strings.Contains(a.Message, "已变化") {
				t.Fatalf("changed owner was accepted or undisclosed: %s", a.Message)
			}
			if got := mustReadFile(t, filename); !bytes.Equal(got, before) {
				t.Fatal("stale owner published an old discard candidate")
			}
			requireNoCloseTemps(t, filename)
			if change == "dialog-order" {
				a.Tabs = a.Tabs[:1] // Restore the deliberately aliased test tab for cleanup.
			}
		})
	}
}

type closeCommitFault struct {
	preparedSessionState
	commit func(func() error) error
}

func (p closeCommitFault) Commit(check func() error) error { return p.commit(check) }

func TestClosePreparedSessionCommitFailureKeepsPriorRecovery(t *testing.T) {
	a, _, runner, filename, before := closePlanFixture(t)
	failure := errors.New("injected commit refusal")
	a.prepareSessionState = func(path string, snapshot session.Snapshot) (preparedSessionState, error) {
		prepared, err := session.PrepareSave(path, snapshot)
		if err != nil {
			return nil, err
		}
		return closeCommitFault{preparedSessionState: prepared, commit: func(check func() error) error {
			if err := check(); err != nil {
				return err
			}
			return &document.StatePublicationError{Path: path, Err: failure}
		}}, nil
	}
	if a.CanClose() || !strings.Contains(a.Message, failure.Error()) || a.persistPending {
		t.Fatal("precommit refusal was accepted or hidden", a.Message)
	}
	select {
	case <-runner.closed:
	default:
		t.Fatal("commit preceded joining cleanup")
	}
	requireOldCloseRecovery(t, a, filename, before)
	requireNoCloseTemps(t, filename)
}

func TestClosePreparedSessionMissingTempFailsBeforePublication(t *testing.T) {
	a, _, runner, filename, before := closePlanFixture(t)
	runner.onClose = func() {
		files, err := filepath.Glob(filepath.Join(filepath.Dir(filename), ".levnote-save-*"))
		if err != nil || len(files) != 1 {
			t.Fatalf("prepared candidate not complete before join: %v %v", files, err)
		}
		// This is the one temp in the test-owned target directory.
		if err := os.Remove(files[0]); err != nil {
			t.Fatal(err)
		}
	}
	if a.CanClose() || !strings.Contains(a.Message, "未发布") {
		t.Fatal("missing temp accepted close or hid the precommit failure", a.Message)
	}
	requireOldCloseRecovery(t, a, filename, before)
	requireNoCloseTemps(t, filename)
}

func TestClosePreparedSessionPostcommitErrorCompletesCommittedClose(t *testing.T) {
	a, _, runner, filename, before := closePlanFixture(t)
	failure := errors.New("directory sync not confirmed")
	a.prepareSessionState = func(path string, snapshot session.Snapshot) (preparedSessionState, error) {
		prepared, err := session.PrepareSave(path, snapshot)
		if err != nil {
			return nil, err
		}
		return closeCommitFault{preparedSessionState: prepared, commit: func(check func() error) error {
			if err := prepared.Commit(check); err != nil {
				return err
			}
			return &document.StatePublicationError{Path: path, Committed: true, Err: failure}
		}}, nil
	}
	if !a.CanClose() || len(a.Tabs) != 0 || !strings.Contains(a.Message, "关闭已提交") || !strings.Contains(a.Message, failure.Error()) || a.persistPending {
		t.Fatal("published close was misrepresented as abort or kept live drafts", a.Message)
	}
	if got := mustReadFile(t, filename); bytes.Equal(got, before) {
		t.Fatal("postcommit fixture did not actually publish the discard")
	}
	select {
	case <-runner.closed:
	default:
		t.Fatal("postcommit preceded cleanup")
	}
	snapshot := readRecovery(t, a)
	if len(snapshot.Tabs) != 1 || snapshot.Tabs[0].Draft != nil {
		t.Fatalf("committed discard did not match actual recovery: %#v", snapshot)
	}
	requireNoCloseTemps(t, filename)
}

func TestCloseTabPrecommitRefusalKeepsExactRecoveryAndRestore(t *testing.T) {
	a, _, _, filename, before := closePlanFixture(t)
	liveBefore := a.Current().Doc.Buffer.String()
	failure := errors.New("single-tab owner rejected before publish")
	called := false
	a.prepareSessionState = func(path string, snapshot session.Snapshot) (preparedSessionState, error) {
		prepared, err := session.PrepareSave(path, snapshot)
		if err != nil {
			return nil, err
		}
		return closeCommitFault{preparedSessionState: prepared, commit: func(func() error) error {
			called = true
			return prepared.Commit(func() error { return failure })
		}}, nil
	}
	if a.CloseTab(0) || !called || !strings.Contains(a.Message, failure.Error()) {
		t.Fatalf("precommit refusal was accepted or the real Commit was bypassed: called=%v message=%q", called, a.Message)
	}
	requireOldCloseRecovery(t, a, filename, before)
	if len(a.Tabs) != 1 || a.Current() == nil || a.Current().Doc.Buffer.String() != liveBefore {
		t.Fatal("precommit refusal changed the live dirty tab")
	}
}

func TestCloseTabPostcommitErrorRemovesOnlyTargetAndPreservesMessage(t *testing.T) {
	a, target, _, filename, _ := closePlanFixture(t)
	a.NewDocument("text")
	kept := a.Current()
	kept.Editor.ReplaceText(nil, "remaining draft🙂")
	// Keep the persisted target valid while ensuring it cannot enter recent-close history.
	target.Doc.Path = filepath.Join(t.TempDir(), strings.Repeat("p", closedPathLimit+32))
	target.Doc.IsNew = false
	a.Persist()
	before := mustReadFile(t, filename)
	failure := errors.New("directory sync outcome unknown")
	called, published := false, false
	a.prepareSessionState = func(path string, snapshot session.Snapshot) (preparedSessionState, error) {
		prepared, err := session.PrepareSave(path, snapshot)
		if err != nil {
			return nil, err
		}
		return closeCommitFault{preparedSessionState: prepared, commit: func(check func() error) error {
			called = true
			if err := prepared.Commit(check); err != nil {
				return err
			}
			published = true
			return &document.StatePublicationError{Path: path, Committed: true, Err: failure}
		}}, nil
	}
	if !a.CloseTab(0) || !called || !published {
		t.Fatalf("real prepared publication was not completed: closed=%v prepared=%v published=%v message=%q", len(a.Tabs) == 1, called, published, a.Message)
	}
	if len(a.Tabs) != 1 || a.Tabs[0] != kept || a.Active != 0 || a.Current() != kept {
		t.Fatalf("committed close removed wrong tab or active mapping: %#v active=%d", a.Tabs, a.Active)
	}
	if !strings.Contains(a.Message, "关闭已提交") || !strings.Contains(a.Message, failure.Error()) || strings.Contains(a.Message, "未加入最近关闭") {
		t.Fatalf("postcommit warning was hidden, misreported, or overwritten: %q", a.Message)
	}
	if len(a.closedFiles) != 0 {
		t.Fatal("oversized closed path entered recent-close history")
	}
	if after := mustReadFile(t, filename); bytes.Equal(after, before) {
		t.Fatal("postcommit fixture did not publish changed recovery bytes")
	}
	snapshot, issues := session.Load(filename)
	if len(issues) != 0 || len(snapshot.Tabs) != 1 || snapshot.Tabs[0].Draft == nil || *snapshot.Tabs[0].Draft != "remaining draft🙂" || snapshot.Active != 0 {
		t.Fatalf("published close did not retain the remaining active draft: %#v %v", snapshot, issues)
	}
	restored := newTestApp(t, a.configDir)
	restored.Restore()
	if len(restored.Tabs) != 1 || restored.Active != 0 || restored.Current().Doc.Buffer.String() != "remaining draft🙂" {
		t.Fatalf("remaining draft/active tab did not actually restore: %#v active=%d", restored.Tabs, restored.Active)
	}
	requireNoCloseTemps(t, filename)
}
