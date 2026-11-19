package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
)

func TestCoalescedTreeWorkerKeepsOnePendingRequestAndRejectsStaleApply(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.treeWorker.close()
	started := make(chan treeScanTask, 4)
	release := make(chan struct{})
	a.treeWorker = newCoalescedWorker(func(ctx context.Context, task treeScanTask) {
		started <- task
		select {
		case <-release:
		case <-ctx.Done():
			return
		}
		entry := Entry{Path: fmt.Sprintf("%s-%d", task.root, task.generation), Name: "result"}
		a.update(func() {
			a.applyTreeScan(task, treeScanResult{entries: []Entry{entry}})
		})
	})
	dispatches := make(chan func(), 4)
	a.uiDispatch = func(fn func()) { dispatches <- fn }
	defer a.Shutdown()

	a.Root = "root-a"
	a.refreshTree()
	first := <-started
	// A blocked disk worker must not block ordinary editing or tab creation.
	edited := make(chan struct{})
	go func() {
		a.NewDocument("text")
		close(edited)
	}()
	select {
	case <-edited:
	case <-time.After(time.Second):
		t.Fatal("editing blocked behind the disk worker")
	}

	a.Root = "root-b"
	a.refreshTree()
	a.Root = "root-c"
	a.refreshTree()
	release <- struct{}{}
	second := <-started
	if second.root != "root-c" || second.generation <= first.generation {
		t.Fatalf("pending request was not replaced by the newest root: first=%#v second=%#v", first, second)
	}

	firstApply := <-dispatches
	firstApply()
	if len(a.Entries) != 0 {
		t.Fatalf("stale tree result replaced the UI after newer requests: %#v", a.Entries)
	}
	release <- struct{}{}
	secondApply := <-dispatches
	secondApply()
	if len(a.Entries) != 1 || a.Entries[0].Path != "root-c-3" {
		t.Fatalf("newest tree result was not applied: %#v", a.Entries)
	}
}

func TestTreeWorkerShutdownTerminatesActiveAndPendingWork(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.treeWorker.close()
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	a.treeWorker = newCoalescedWorker(func(ctx context.Context, _ treeScanTask) {
		started <- struct{}{}
		select {
		case <-release:
		case <-ctx.Done():
		}
	})
	a.Root = "root"
	a.refreshTree()
	<-started
	a.refreshTree()

	done := make(chan struct{})
	go func() {
		a.Shutdown()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("shutdown retained a blocked tree worker")
	}
}

func TestOpenQueueTimesOutIfAsyncUIDispatchNeverRuns(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "open.txt")
	if err := os.WriteFile(path, []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	defer a.Shutdown()
	a.updateWaitTimeout = 40 * time.Millisecond
	queued := make(chan func(), 2)
	a.uiDispatch = func(fn func()) { queued <- fn }

	done := a.openRequests([]openRequest{{path: path}})
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("open queue waited forever for an unexecuted UI callback")
	}
	if len(a.Tabs) != 0 {
		t.Fatalf("timed-out open mutated tabs: %d", len(a.Tabs))
	}

	// The abandoned mutation and its failure report both go through dispatch.
	// Execute them in queue order as the real UI thread eventually would.
	lateMutation := <-queued
	lateMutation()
	lateFailure := <-queued
	lateFailure()
	if len(a.Tabs) != 0 {
		t.Fatalf("abandoned UI callback opened a tab after timeout: %#v", a.Tabs)
	}
	if !strings.Contains(a.Message, "界面未能响应") {
		t.Fatalf("dispatch failure was not surfaced: %q", a.Message)
	}
}

func TestQueuedUpdateRechecksShutdownAtExecution(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	queued := make(chan func(), 1)
	a.uiDispatch = func(fn func()) { queued <- fn }
	ran := false
	a.update(func() { ran = true })
	a.Shutdown()

	callback := <-queued
	callback()
	if ran {
		t.Fatal("queued worker callback mutated App after shutdown")
	}
}

func TestUpdateWaitIsReleasedAndLateCallbackSuppressedByShutdown(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.updateWaitTimeout = 5 * time.Second
	queued := make(chan func(), 1)
	a.uiDispatch = func(fn func()) { queued <- fn }
	ran := false
	result := make(chan bool, 1)
	go func() {
		result <- a.updateWait(func() { ran = true })
	}()
	callback := <-queued

	started := time.Now()
	a.Shutdown()
	select {
	case ok := <-result:
		if ok {
			t.Fatal("worker wait reported acknowledgement after shutdown")
		}
	case <-time.After(time.Second):
		t.Fatal("worker wait did not unblock immediately on shutdown")
	}
	if time.Since(started) >= time.Second {
		t.Fatal("worker wait relied on its five-second timeout")
	}
	callback()
	if ran {
		t.Fatal("late acknowledged callback ran after shutdown")
	}
}

func TestUpdateWaitTimeoutDoesNotAbandonStartedCallback(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.updateWaitTimeout = 40 * time.Millisecond
	queued := make(chan func(), 1)
	a.uiDispatch = func(fn func()) { queued <- fn }
	started := make(chan struct{})
	release := make(chan struct{})
	applied := false
	result := make(chan bool, 1)
	go func() {
		result <- a.updateWait(func() {
			close(started)
			<-release
			applied = true
		})
	}()

	callback := <-queued
	callbackDone := make(chan struct{})
	go func() {
		callback()
		close(callbackDone)
	}()
	<-started

	select {
	case <-result:
		t.Fatal("updateWait reported before the started UI callback completed")
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	if ok := <-result; !ok || !applied {
		t.Fatalf("started callback outcome was not acknowledged: ok=%v applied=%v", ok, applied)
	}
	select {
	case <-callbackDone:
	case <-time.After(time.Second):
		t.Fatal("UI callback did not finish")
	}

	// The callback must not be queued or applied twice after acknowledgement.
	select {
	case fn := <-queued:
		fn()
		t.Fatal("updateWait queued a duplicate UI callback")
	case <-time.After(20 * time.Millisecond):
	}
}

func TestDiskCheckWorkerIncludesReadonlyTextAndExcludesByteInspector(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "readonly.txt")
	if err := os.WriteFile(path, []byte("disk text"), 0600); err != nil {
		t.Fatal(err)
	}
	doc, err := document.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	doc.ReadOnly = true
	a := newTestApp(t, filepath.Join(root, "config"))
	readonly := a.appendTab(doc, "readonly.txt", "", false, false)
	readonly.Doc.ReadOnly = true
	inspectorDoc := document.New("inspector bytes")
	inspectorDoc.Path = path
	inspectorDoc.IsNew = false
	inspectorDoc.ReadOnly = true
	inspector := a.appendTab(inspectorDoc, "inspector", "", true, false)

	a.treeWorker.close()
	a.diskWorker.close()
	started := make(chan diskCheckTask, 4)
	release := make(chan struct{})
	a.diskWorker = newCoalescedWorker(func(ctx context.Context, task diskCheckTask) {
		started <- task
		select {
		case <-release:
		case <-ctx.Done():
			return
		}
		results := make([]diskCheckResult, 0, len(task.targets))
		for _, target := range task.targets {
			results = append(results, diskCheckResult{
				tabID: target.tabID, path: target.path, token: target.token, changed: true,
			})
		}
		a.update(func() { a.applyDiskCheck(task, results) })
	})
	dispatches := make(chan func(), 4)
	a.uiDispatch = func(fn func()) { dispatches <- fn }

	a.CheckExternal()
	first := <-started
	if len(first.targets) != 1 || first.targets[0].tabID != readonly.ID {
		t.Fatalf("disk targets = %#v, want only readonly tab %d (inspector %d)", first.targets, readonly.ID, inspector.ID)
	}
	a.CheckExternal()
	release <- struct{}{}
	second := <-started
	firstApply := <-dispatches
	firstApply()
	if readonly.Conflict {
		t.Fatal("stale disk result marked the readonly tab before the newest generation")
	}
	release <- struct{}{}
	secondApply := <-dispatches
	secondApply()
	if !readonly.Conflict || second.generation <= first.generation {
		t.Fatalf("newest readonly result was not applied: conflict=%v first=%d second=%d",
			readonly.Conflict, first.generation, second.generation)
	}
	if inspector.Conflict {
		t.Fatal("byte inspector received an external-change conflict")
	}
}

func TestDiskCheckRejectsResultAfterSaveChangesBaseline(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "saved.txt")
	if err := os.WriteFile(path, []byte("old disk"), 0600); err != nil {
		t.Fatal(err)
	}
	doc, err := document.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	tab := a.appendTab(doc, "saved.txt", "", false, false)

	a.diskWorker.close()
	started := make(chan diskCheckTask, 1)
	release := make(chan struct{})
	a.diskWorker = newCoalescedWorker(func(ctx context.Context, task diskCheckTask) {
		started <- task
		select {
		case <-release:
		case <-ctx.Done():
			return
		}
		result := diskCheckResult{
			tabID: task.targets[0].tabID, path: task.targets[0].path,
			token: task.targets[0].token, changed: true,
		}
		a.update(func() { a.applyDiskCheck(task, []diskCheckResult{result}) })
	})
	dispatches := make(chan func(), 1)
	a.uiDispatch = func(fn func()) { dispatches <- fn }

	a.CheckExternal()
	<-started
	tab.Editor.ReplaceText(
		&ui.TextInputRange{Start: 0, End: tab.Doc.Buffer.LenUTF16()},
		"new saved baseline",
	)
	if !a.Save(tab, false) {
		t.Fatal("save failed while preparing stale disk-check regression")
	}
	release <- struct{}{}
	apply := <-dispatches
	apply()
	if tab.Conflict {
		t.Fatal("pre-save disk result marked the newly saved baseline as conflicted")
	}
}

func TestDiskCheckRejectsOldTokenAfterUnchangedSaveOrRevert(t *testing.T) {
	for _, tc := range []struct {
		name   string
		action func(t *testing.T, a *App, tab *Tab)
	}{
		{
			name: "save",
			action: func(t *testing.T, a *App, tab *Tab) {
				t.Helper()
				if !a.Save(tab, false) {
					t.Fatal("unchanged save failed")
				}
			},
		},
		{
			name: "revert",
			action: func(t *testing.T, a *App, _ *Tab) {
				t.Helper()
				a.Active = 0
				a.Reload()
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "stable.txt")
			if err := os.WriteFile(path, []byte("same bytes"), 0600); err != nil {
				t.Fatal(err)
			}
			doc, err := document.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			a := newTestApp(t, filepath.Join(root, "config"))
			tab := a.appendTab(doc, "stable.txt", "", false, false)
			a.diskWorker.close()
			started := make(chan diskCheckTask, 1)
			release := make(chan struct{})
			a.diskWorker = newCoalescedWorker(func(ctx context.Context, task diskCheckTask) {
				started <- task
				select {
				case <-release:
				case <-ctx.Done():
					return
				}
				result := diskCheckResult{
					tabID: task.targets[0].tabID, path: task.targets[0].path,
					token: task.targets[0].token, changed: true,
				}
				a.update(func() { a.applyDiskCheck(task, []diskCheckResult{result}) })
			})
			dispatches := make(chan func(), 1)
			a.uiDispatch = func(fn func()) { dispatches <- fn }

			a.CheckExternal()
			task := <-started
			tc.action(t, a, tab)
			if task.generation == a.diskGeneration {
				t.Fatalf("%s did not invalidate the pending disk generation", tc.name)
			}
			release <- struct{}{}
			apply := <-dispatches
			apply()
			if tab.Conflict {
				t.Fatalf("%s accepted a pre-operation disk result", tc.name)
			}
		})
	}
}

func TestDiskCheckWorkerFindsSameSizeExternalReplacement(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "watched.txt")
	if err := os.WriteFile(path, []byte("before"), 0600); err != nil {
		t.Fatal(err)
	}
	doc, err := document.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	doc.ReadOnly = true
	a := newTestApp(t, filepath.Join(root, "config"))
	tab := a.appendTab(doc, "watched.txt", "", false, false)
	a.diskWorker.close()
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	a.diskWorker = newCoalescedWorker(func(ctx context.Context, task diskCheckTask) {
		started <- struct{}{}
		select {
		case <-release:
		case <-ctx.Done():
			return
		}
		result := checkDiskTask(ctx, task)
		if ctx.Err() == nil {
			a.update(func() { a.applyDiskCheck(task, result) })
		}
	})
	a.CheckExternal()
	<-started
	if err := os.WriteFile(path, []byte("after!"), 0600); err != nil {
		t.Fatal(err)
	}
	release <- struct{}{}

	deadline := time.Now().Add(2 * time.Second)
	for !tab.Conflict && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !tab.Conflict {
		t.Fatal("same-size external replacement was not detected")
	}
	if !strings.Contains(a.Message, "已在外部改变") {
		t.Fatalf("external replacement was not visible to the user: %q", a.Message)
	}
}

func TestDiskWorkerShutdownTerminatesActiveCheck(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "shutdown.txt")
	if err := os.WriteFile(path, []byte("disk"), 0600); err != nil {
		t.Fatal(err)
	}
	doc, err := document.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.appendTab(doc, "shutdown.txt", "", false, false)
	a.diskWorker.close()
	started := make(chan struct{}, 1)
	a.diskWorker = newCoalescedWorker(func(ctx context.Context, _ diskCheckTask) {
		started <- struct{}{}
		<-ctx.Done()
	})
	a.CheckExternal()
	<-started

	done := make(chan struct{})
	go func() {
		a.Shutdown()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown retained a blocked disk-check worker")
	}
}
