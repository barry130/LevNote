package app

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/session"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

func TestMarkdownCellDraftRecoveryQuietPeriodCompositionAndDiscard(t *testing.T) {
	a, tab, tester, originalPath := openRealMarkdownTables(t, 1180, 780)
	h := installAutosaveClock(a)
	body := requireMarkdownCellAccess(t, tester, ui.RoleCell, markdownCellBodyName)
	openMarkdownCellPointer(t, tester, body, markdownCellBodyEditor)
	buffer := tab.Doc.Buffer
	version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
	replaceMarkdownCellDraft(t, tester, markdownCellBodyEditor, "待提交🙂")
	tester.Type("待提交🙂")
	tester.Frame()
	timer := h.latest(t)
	if timer.delay != 900*time.Millisecond {
		t.Fatalf("session quiet period = %s", timer.delay)
	}
	composer, ok := tab.View.(editor.Composer)
	if !ok || composer.CompositionActive() {
		t.Fatal("confirmed cell input did not end composition")
	}
	tester.Compose("预编辑", 3)
	tester.Frame()
	if !composer.CompositionActive() {
		t.Fatal("cell preedit did not start")
	}
	if h.latest(t) != timer {
		t.Fatal("unconfirmed preedit replaced the confirmed edit timer")
	}
	h.now = h.now.Add(autosaveDelay - time.Millisecond)
	timer.fn()
	h.drain()
	requireNoRecovery(t, a)
	h.now = h.now.Add(time.Millisecond)
	h.latest(t).fn()
	requireNoRecovery(t, a) // Dispatch posts before the UI owns capture.
	h.drain()
	if !composer.CompositionActive() || !requireMarkdownCellEditor(t, tester, markdownCellBodyEditor).Focused {
		t.Fatal("session capture finished composition or changed cell focus")
	}
	if buffer.String() != markdownTablesBody || tab.Doc.Dirty() || buffer.Version() != version ||
		buffer.Selection() != selection || buffer.History() != history {
		t.Fatal("session capture edited the main document or its selection/history")
	}
	if got := mustReadFile(t, originalPath); !bytes.Equal(got, markdownTablesFixture()) {
		t.Fatalf("session capture rewrote the original: %q", got)
	}
	snapshot := readRecovery(t, a)
	wantSelection := text.Range{Anchor: 5, Head: 5}
	if len(snapshot.Tabs) != 2 || snapshot.Active != 0 || snapshot.Tabs[0].Draft != nil ||
		snapshot.Tabs[0].Path != originalPath || !snapshot.Tabs[0].BOM || snapshot.Tabs[0].Encoding != document.UTF8 {
		t.Fatalf("original document or session order changed: %#v", snapshot)
	}
	copy := snapshot.Tabs[1]
	if copy.Path != "" || copy.OriginalPath != originalPath || copy.Draft == nil || *copy.Draft != "待提交🙂" || copy.Selection != wantSelection ||
		copy.Encoding != document.UTF8 || copy.BOM || copy.View != "source" || !strings.HasSuffix(copy.Title, ".txt") {
		t.Fatalf("independent cell record = %#v", copy)
	}

	reopened := newTestApp(t, a.configDir)
	reopened.Restore()
	if len(reopened.Tabs) != 2 || reopened.Active != 0 || reopened.Current().Doc.Path != originalPath || reopened.Current().Doc.Dirty() {
		t.Fatalf("restart lost the original active document: tabs=%d active=%d", len(reopened.Tabs), reopened.Active)
	}
	recovered := reopened.Tabs[1]
	if recovered.Doc.Path != "" || recovered.OriginalPath != originalPath || !recovered.Doc.IsNew || recovered.Mode != "source" ||
		recovered.Doc.Buffer.String() != "待提交🙂" || recovered.Doc.Buffer.Selection() != wantSelection {
		t.Fatalf("restart lost the independent source copy: %#v", recovered)
	}

	// Confirm the next input, then clear the field and persist a real empty draft.
	tester.Type("预编辑")
	tester.Frame()
	tester.Command("selectAll")
	tester.Key(0, ui.KeyBackspace)
	tester.Frame()
	h.now = h.now.Add(autosaveDelay)
	h.latest(t).fn()
	h.drain()
	snapshot = readRecovery(t, a)
	if len(snapshot.Tabs) != 2 || snapshot.Tabs[1].Draft == nil || *snapshot.Tabs[1].Draft != "" {
		t.Fatalf("confirmed empty draft disappeared: %#v", snapshot)
	}

	if err := tester.Click(markdownCellDiscardLabel); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	h.now = h.now.Add(autosaveDelay)
	h.latest(t).fn()
	h.drain()
	if snapshot := readRecovery(t, a); len(snapshot.Tabs) != 1 || snapshot.Tabs[0].Path != originalPath || snapshot.Tabs[0].Draft != nil {
		t.Fatalf("discarded cell survived the next quiet capture: %#v", snapshot)
	}
}

type sessionDraftFixture struct {
	failingView
	draft    editor.SessionDraft
	dirty    bool
	err      error
	captures int
	budget   int
	capture  func()
}

func (v *sessionDraftFixture) CaptureSessionDraft(maxBytes int) (editor.SessionDraft, bool, error) {
	v.captures++
	v.budget = maxBytes
	if v.capture != nil {
		v.capture()
	}
	return v.draft, v.dirty, v.err
}

func TestSessionDraftNotificationIsBoundToLiveViewAndKeepsActivePrimary(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	first := a.appendTab(document.New("first"), "first.txt", "", false, false)
	second := a.appendTab(document.New("second"), "second.txt", "", false, false)
	a.Active = 1
	h := installAutosaveClock(a)
	owner := &sessionDraftFixture{draft: editor.SessionDraft{Title: `C:\example\cell.md`, Content: "草稿🙂", Anchor: 4, Head: 2}, dirty: true}
	first.View = owner
	notify := a.editorHost(first).DraftRecoveryChanged
	notify(nil)
	notify(&sessionDraftFixture{})
	if len(h.timers) != 0 || a.persistPending || !a.lastEdit.IsZero() {
		t.Fatal("nil or stale view scheduled session recovery")
	}
	notify(owner)
	h.now = h.now.Add(autosaveDelay)
	h.latest(t).fn()
	if owner.captures != 0 {
		t.Fatal("timer captured outside the UI callback")
	}
	h.drain()
	snapshot := readRecovery(t, a)
	if owner.captures != 1 || owner.budget != session.MaxDraftBytes || owner.flushes != 0 || owner.releases != 0 || a.Current() != second {
		t.Fatal("capture changed owners, focus, view lifecycle, or the fixed byte budget")
	}
	if len(snapshot.Tabs) != 3 || snapshot.Active != 2 || snapshot.Tabs[1].Title != "恢复-cell.md.txt" ||
		snapshot.Tabs[1].OriginalPath != "" || snapshot.Tabs[1].Path != "" || snapshot.Tabs[1].Selection != (text.Range{Anchor: 4, Head: 2}) {
		t.Fatalf("independent record displaced the active primary or kept a writable path: %#v", snapshot)
	}
	first.View = &sessionDraftFixture{}
	notify(owner)
	a.Tabs = a.Tabs[1:]
	a.Active = 0
	notify(first.View)
	if len(h.timers) != 1 || a.persistPending || owner.captures != 1 {
		t.Fatal("replaced or closed owner rearmed session capture")
	}
}

func TestSessionDraftCaptureFailuresKeepPreviousSnapshot(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		err     error
		maxTabs bool
	}{
		{name: "capture-error", err: errors.New("independent capture failed")},
		{name: "draft-bytes", content: strings.Repeat("a", session.MaxDraftBytes+1)},
		{name: "escaped-JSON", content: strings.Repeat("<", session.MaxDraftBytes)},
		{name: "tab-limit", content: "additional draft", maxTabs: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := newTestApp(t, t.TempDir())
			tab := a.appendTab(document.New("main draft"), "main.txt", "", false, false)
			a.Active = 0
			if tc.maxTabs {
				seedAdmissionTabs(a, session.MaxTabs-1)
			}
			a.Persist()
			filename := filepath.Join(a.configDir, "session.json")
			before := mustReadFile(t, filename)
			h := installAutosaveClock(a)
			owner := &sessionDraftFixture{draft: editor.SessionDraft{Content: tc.content}, dirty: true, err: tc.err}
			tab.View = owner
			a.editorHost(tab).DraftRecoveryChanged(owner)
			h.now = h.now.Add(autosaveDelay)
			h.latest(t).fn()
			h.drain()
			if got := mustReadFile(t, filename); !bytes.Equal(got, before) {
				t.Fatal("failed capture replaced the previous recovery snapshot")
			}
			if owner.captures != 1 || a.persistPending || len(h.timers) != 1 || !strings.Contains(a.lastPersistError, "现有恢复快照保持不变") {
				t.Fatalf("failed capture retried or was not disclosed: captures=%d pending=%v timers=%d error=%q", owner.captures, a.persistPending, len(h.timers), a.lastPersistError)
			}
		})
	}
}

func TestSessionDraftCaptureRejectsReentrantOwnerAndInvalidSelection(t *testing.T) {
	for _, change := range []string{"view", "document", "buffer", "selection"} {
		t.Run(change, func(t *testing.T) {
			a := newTestApp(t, t.TempDir())
			tab := a.appendTab(document.New("main text"), "note.md", "", false, false)
			a.Active = 0
			a.Persist()
			filename := filepath.Join(a.configDir, "session.json")
			before := mustReadFile(t, filename)
			owner := &sessionDraftFixture{draft: editor.SessionDraft{Content: "confirmed🙂"}, dirty: true}
			doc, buffer := tab.Doc, tab.Doc.Buffer
			defer func() { tab.Doc = doc; doc.Buffer = buffer; tab.Editor.Buffer = buffer }()
			tab.View = owner
			switch change {
			case "view":
				owner.capture = func() { tab.View = &sessionDraftFixture{} }
			case "document":
				owner.capture = func() { tab.Doc = nil }
			case "buffer":
				owner.capture = func() { tab.Doc.Buffer = text.New("new buffer") }
			case "selection":
				owner.draft.Anchor = text.UTF16Len(owner.draft.Content) + 1
			}
			if err := a.persistTabs(a.Tabs, a.Active, "捕获归属测试"); err == nil {
				t.Fatal("stale owner or invalid independent selection was persisted")
			}
			if got := mustReadFile(t, filename); !bytes.Equal(got, before) {
				t.Fatal("refused capture replaced the previous recovery file")
			}
		})
	}
}

func TestSessionDraftQuitDiscardDoesNotCaptureOrLeaveIndependentCopy(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := a.appendTab(document.New("new Markdown draft"), "new.md", "", false, false)
	a.Active = 0
	owner := &sessionDraftFixture{draft: editor.SessionDraft{Content: "independent draft"}, dirty: true}
	tab.View = owner
	a.Persist()
	if snapshot := readRecovery(t, a); len(snapshot.Tabs) != 2 {
		t.Fatalf("quit fixture did not capture both records: %#v", snapshot)
	}
	a.closeDialog = func(*Tab) (closeDecision, error) { return closeDiscard, nil }
	if !a.CanClose() || len(a.Tabs) != 0 || owner.captures != 1 {
		t.Fatal("accepted quit recaptured or retained the discarded drafts")
	}
	if snapshot := readRecovery(t, a); len(snapshot.Tabs) != 0 {
		t.Fatalf("quit did not publish discard: %#v", snapshot)
	}
}
