package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/session"
)

type fakeAutosaveTimer struct {
	delay   time.Duration
	fn      func()
	stopped bool
}

func (f *fakeAutosaveTimer) Stop() bool {
	wasActive := !f.stopped
	f.stopped = true
	return wasActive
}

type autosaveClock struct {
	now    time.Time
	timers []*fakeAutosaveTimer
	mu     sync.Mutex
	queue  []func()
}

func installAutosaveClock(a *App) *autosaveClock {
	h := &autosaveClock{now: time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)}
	a.autosaveHooks = &autosaveHooks{
		now: func() time.Time { return h.now },
		afterFunc: func(delay time.Duration, fn func()) autosaveTimer {
			timer := &fakeAutosaveTimer{delay: delay, fn: fn}
			h.timers = append(h.timers, timer)
			return timer
		},
		dispatch: func(fn func()) {
			h.mu.Lock()
			h.queue = append(h.queue, fn)
			h.mu.Unlock()
		},
	}
	return h
}

func (h *autosaveClock) latest(t *testing.T) *fakeAutosaveTimer {
	t.Helper()
	if len(h.timers) == 0 {
		t.Fatal("no autosave timer was armed")
	}
	return h.timers[len(h.timers)-1]
}

func (h *autosaveClock) drain() {
	h.mu.Lock()
	queue := h.queue
	h.queue = nil
	h.mu.Unlock()
	for _, fn := range queue {
		fn()
	}
}

func requireNoRecovery(t *testing.T, a *App) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(a.configDir, "session.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected recovery file: %v", err)
	}
}

func readRecovery(t *testing.T, a *App) session.Snapshot {
	t.Helper()
	path := filepath.Join(a.configDir, "session.json")
	if _, err := os.ReadFile(path); err != nil {
		t.Fatal(err)
	}
	snapshot, issues := session.Load(path)
	if len(issues) != 0 {
		t.Fatalf("invalid recovery: %#v", issues)
	}
	return snapshot
}

func TestRecoveryAutosaveWithoutViewUsesQueuedUIAndDoesNotFlush(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	h := installAutosaveClock(a)
	a.NewDocument("text")
	a.Current().Editor.ReplaceText(nil, "committed draft")
	view := &failingView{flushErr: errors.New("uncommitted format-local draft")}
	a.Current().View = view
	h.now = h.now.Add(autosaveDelay)
	timer := h.latest(t)
	posted := make(chan struct{})
	go func() { timer.fn(); close(posted) }()
	<-posted
	// Posting from the timer never captures documents or writes files. The
	// following drain is the only simulated main-thread work; no View runs.
	requireNoRecovery(t, a)
	h.drain()
	snapshot := readRecovery(t, a)
	if len(snapshot.Tabs) != 1 || snapshot.Tabs[0].Draft == nil || *snapshot.Tabs[0].Draft != "committed draft" || a.persistPending {
		t.Fatalf("committed draft not recovered: %#v", snapshot)
	}
	if view.flushes != 0 || view.releases != 0 {
		t.Fatal("automatic persistence touched a format-local draft")
	}
	// Source preedit already lives in its buffer, but automatic capture must
	// leave the editor's grouping/marked state untouched rather than commit it.
	a.Current().Editor.SetMarkedText(nil, "预编辑", ui.TextInputRange{Start: 3, End: 3})
	marked, composing := a.Current().Editor.MarkedRange()
	if !composing {
		t.Fatal("composition fixture did not start")
	}
	h.now = h.now.Add(autosaveDelay)
	h.latest(t).fn()
	h.drain()
	if got, active := a.Current().Editor.MarkedRange(); !active || got != marked || view.flushes != 0 {
		t.Fatal("autosave committed live composition")
	}
}

func TestRecoveryAutosaveDebounceAndNonEditCoalescing(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.NewDocument("text")
	a.NewDocument("text")
	h := installAutosaveClock(a)
	a.Select(0)
	a.Current().Editor.ReplaceText(nil, "first")
	old := h.latest(t)
	h.now = h.now.Add(autosaveDelay)
	old.fn() // already queued when the next edit arrives
	a.Current().Editor.ReplaceText(nil, " later")
	newest := h.latest(t)
	h.drain()
	requireNoRecovery(t, a)
	if !old.stopped || newest.delay != autosaveDelay {
		t.Fatal("later edit did not replace the old deadline")
	}
	lastEdit, timerCount := a.lastEdit, len(h.timers)
	h.now = h.now.Add(100 * time.Millisecond)
	a.Select(1)
	if a.lastEdit != lastEdit || len(h.timers) != timerCount {
		t.Fatal("tab selection reset the edit deadline or multiplied timers")
	}
	// A timer delivered early must recheck the deadline on UI and rearm only
	// its remainder; selecting a tab is not permission to save early.
	newest.fn()
	h.drain()
	requireNoRecovery(t, a)
	if h.latest(t).delay != 800*time.Millisecond {
		t.Fatalf("early callback remainder = %s", h.latest(t).delay)
	}
	h.now = h.now.Add(800 * time.Millisecond)
	h.latest(t).fn()
	h.drain()
	snapshot := readRecovery(t, a)
	if len(snapshot.Tabs) != 2 || snapshot.Active != 1 || snapshot.Tabs[0].Draft == nil || *snapshot.Tabs[0].Draft != "first later" {
		t.Fatalf("latest edit/selection not recovered: %#v", snapshot)
	}
}

func TestRecoveryAutosaveModalExitResumesWithoutView(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	h := installAutosaveClock(a)
	a.NewDocument("text")
	a.Current().Editor.ReplaceText(nil, "keep after cancelled close")
	h.now = h.now.Add(autosaveDelay)
	h.latest(t).fn()
	a.closeDialog = func(*Tab) (closeDecision, error) {
		finishNested := a.deferExportDelivery()
		h.drain()
		requireNoRecovery(t, a)
		count := len(h.timers)
		finishNested()
		if len(h.timers) != count || !a.persistPending {
			t.Fatal("nested modal exit scheduled too early or lost pending state")
		}
		return closeCancel, nil
	}
	if a.CloseTab(0) {
		t.Fatal("close cancellation was ignored")
	}
	if h.latest(t).delay != 0 || a.modalDepth != 0 {
		t.Fatal("outer modal exit did not rearm overdue recovery")
	}
	h.latest(t).fn()
	h.drain()
	snapshot := readRecovery(t, a)
	if len(snapshot.Tabs) != 1 || snapshot.Tabs[0].Draft == nil || *snapshot.Tabs[0].Draft != "keep after cancelled close" {
		t.Fatalf("cancelled close lost its draft: %#v", snapshot)
	}
}

func TestRecoveryAutosaveRestoreAndShutdownInvalidateLateCallbacks(t *testing.T) {
	t.Run("restore transaction", func(t *testing.T) {
		a := newTestApp(t, t.TempDir())
		path := filepath.Join(a.configDir, "session.json")
		first, second := "first restored draft", "second restored draft"
		if err := session.Save(path, session.Snapshot{Active: 1, Tabs: []session.Tab{
			{Title: "one.txt", Draft: &first}, {Title: "two.txt", Draft: &second},
		}}); err != nil {
			t.Fatal(err)
		}
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		h := installAutosaveClock(a)
		a.markPersistPending()
		queued := h.latest(t)
		queued.fn()
		count := len(h.timers)
		a.Restore()
		if len(a.Tabs) != 2 || a.Active != 1 || a.Current().Doc.Buffer.String() != second || a.persistPending || a.restoring {
			t.Fatal("restore transaction/final decision changed")
		}
		if len(h.timers) != count || !queued.stopped {
			t.Fatal("restore armed partial-state autosave or retained old timer")
		}
		h.drain()
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("late callback overwrote the recovery being restored", err)
		}
	})
	t.Run("shutdown", func(t *testing.T) {
		a := newTestApp(t, t.TempDir())
		h := installAutosaveClock(a)
		a.NewDocument("text")
		a.Current().Editor.ReplaceText(nil, "must not save after shutdown")
		timer := h.latest(t)
		h.now = h.now.Add(autosaveDelay)
		timer.fn()
		a.Shutdown()
		if !timer.stopped {
			t.Fatal("shutdown did not stop timer")
		}
		// Include the Stop race: a callback may still post after cancellation.
		timer.fn()
		h.drain()
		requireNoRecovery(t, a)
	})
}

func TestRecoveryAutosaveExplicitCloseSupersedesQueuedCapture(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	h := installAutosaveClock(a)
	a.NewDocument("text")
	a.Current().Editor.ReplaceText(nil, "discarded draft")
	h.now = h.now.Add(autosaveDelay)
	h.latest(t).fn()
	a.closeDialog = func(*Tab) (closeDecision, error) { return closeDiscard, nil }
	if !a.CloseTab(0) || len(readRecovery(t, a).Tabs) != 0 {
		t.Fatal("explicit close did not publish the discard")
	}
	a.NewDocument("text")
	a.Current().Editor.ReplaceText(nil, "new draft")
	h.drain()
	if len(readRecovery(t, a).Tabs) != 0 || !a.persistPending {
		t.Fatal("old callback captured the next state before its deadline")
	}
	h.now = h.now.Add(autosaveDelay)
	h.latest(t).fn()
	h.drain()
	snapshot := readRecovery(t, a)
	if len(snapshot.Tabs) != 1 || snapshot.Tabs[0].Draft == nil || *snapshot.Tabs[0].Draft != "new draft" {
		t.Fatal("new state did not recover after explicit close")
	}
}

func TestRecoveryAutosaveFailureWaitsForMutationOrExplicitRetry(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		name := "automatic"
		if explicit {
			name = "failed close"
		}
		t.Run(name, func(t *testing.T) {
			a := newTestApp(t, t.TempDir())
			h := installAutosaveClock(a)
			a.NewDocument("text")
			a.Current().Editor.ReplaceText(nil, "draft")
			path := filepath.Join(a.configDir, "session.json")
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
			h.now = h.now.Add(autosaveDelay)
			timer := h.latest(t)
			timer.fn()
			if explicit {
				a.closeDialog = func(*Tab) (closeDecision, error) { return closeDiscard, nil }
				if a.CloseTab(0) {
					t.Fatal("failed close discarded the live tab")
				}
			} else {
				h.drain()
			}
			if a.persistPending || len(a.Issues) != 1 {
				t.Fatal("failed attempt remained pending or did not report once")
			}
			count := len(h.timers)
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			timer.fn()
			h.drain()
			requireNoRecovery(t, a)
			if len(h.timers) != count || len(a.Issues) != 1 {
				t.Fatal("unchanged failed state was retried")
			}
			if explicit {
				if !a.CloseTab(0) || len(readRecovery(t, a).Tabs) != 0 {
					t.Fatal("explicit persistence could not retry")
				}
			} else {
				a.Current().Editor.ReplaceText(nil, " retry")
				h.now = h.now.Add(autosaveDelay)
				h.latest(t).fn()
				h.drain()
				snapshot := readRecovery(t, a)
				if len(snapshot.Tabs) != 1 || snapshot.Tabs[0].Draft == nil || *snapshot.Tabs[0].Draft != "draft retry" {
					t.Fatal("new edit did not request fresh recovery")
				}
			}
		})
	}
}

func TestRecoveryAutosaveBinaryNonEditStoresOnlyMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "image.png")
	imageFixture(t, path, "PNG")
	a := newTestApp(t, filepath.Join(dir, "config"))
	h := installAutosaveClock(a)
	a.Open(path)
	if a.Current() == nil || !a.Current().isBinary() || !a.lastEdit.IsZero() {
		t.Fatal("binary open fixture failed or changed edit history")
	}
	count := len(h.timers)
	if !a.setMode(a.Current(), "source") || len(h.timers) != count || h.latest(t).delay != 0 {
		t.Fatal("binary presentation change did not coalesce its immediate request")
	}
	h.latest(t).fn()
	h.drain()
	snapshot := readRecovery(t, a)
	if len(snapshot.Tabs) != 1 {
		t.Fatal("binary record was omitted")
	}
	record, err := session.RestoreBinary(snapshot.Tabs[0])
	if err != nil || record.Path != path || record.Kind != document.BinaryKindPNG || record.View != "source" {
		t.Fatalf("binary recovery metadata = %#v, %v", record, err)
	}
	data, err := os.ReadFile(filepath.Join(a.configDir, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	var wire struct{ Tabs []map[string]json.RawMessage }
	if err := json.Unmarshal(data, &wire); err != nil || len(wire.Tabs) != 1 {
		t.Fatal("invalid binary wire record", err)
	}
	allowed := map[string]bool{"record": true, "kind": true, "path": true, "title": true, "pluginId": true, "view": true}
	for key := range wire.Tabs[0] {
		if !allowed[key] {
			t.Fatalf("binary autosave included non-metadata field %q", key)
		}
	}
}

func TestRecoveryAutosaveHeadlessInlineDispatcherNeverArmsRealTimer(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	calls := 0
	a.uiDispatch = func(fn func()) { calls++; fn() }
	a.NewDocument("text")
	a.Current().Editor.ReplaceText(nil, "headless draft")
	if a.autosave.timer != nil || calls != 0 || !a.persistPending {
		t.Fatal("an inline headless dispatcher was treated as a safe main queue")
	}
	a.Persist()
	if len(readRecovery(t, a).Tabs) != 1 {
		t.Fatal("headless explicit persistence was disabled")
	}
}
