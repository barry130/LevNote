//go:build windows

package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineruntime"
)

func awaitPackageDone(t *testing.T, job *packageJob) {
	t.Helper()
	timer := time.NewTimer(enginePackageTestWait)
	defer timer.Stop()
	select {
	case <-job.done:
	case <-timer.C:
		t.Fatalf("package job %s did not join within %s", job.action, enginePackageTestWait)
	}
}

func runPackageLifecycleJob(t *testing.T, a *App, action packageAction, engine enginecatalog.EngineID, value string) *packageJob {
	t.Helper()
	a.startPackageWork(action, engine, value)
	job := a.packageJob
	if job == nil {
		t.Fatalf("%s did not start: %q", action, a.Message)
	}
	awaitPackageDone(t, job)
	a.deliverPackageWork()
	return job
}

func TestEnginePackageTransitionJoinsOldLaneAndPreservesWritableDraft(t *testing.T) {
	a, tab, f, _, _ := workerApp(t)
	call := func() *exportCall {
		a.exportCurrent()
		if a.exportJob == nil {
			t.Fatalf("export did not start: %q", a.Message)
		}
		return awaitExportCall(t, f)
	}
	_, oldJob := call(), a.exportJob
	if oldJob.lane != a.engineLane || a.engineLane.runner != f {
		t.Fatal("first export did not capture the controlled runner")
	}
	if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, "pending "); err != nil {
		t.Fatal(err)
	}
	if tab.Doc.Buffer.History().UndoDepth == 0 {
		t.Fatal("writable draft has no undo history")
	}
	dirty := tab.Doc.Dirty()
	f.closeErr = errors.New("owned cleanup failure")

	runPackageLifecycleJob(t, a, packageResetAction, enginecatalog.MarkdownHTML, "")
	awaitExportDone(t, oldJob)
	if oldJob.lane == a.engineLane || oldJob.engineGeneration == a.engineGeneration {
		t.Fatal("old export retained lane or generation ownership")
	}
	if a.exportJob == oldJob {
		t.Fatal("old export remained the current publication owner")
	}
	if len(f.calls) != 0 {
		call := <-f.calls
		t.Fatalf("new lane used captured runner: %s", call.id)
	}
	if a.exporter == f || a.engineLane != nil {
		t.Fatalf("transition did not clear lazy engine ownership: exporter=%v lane=%v", a.exporter, a.engineLane)
	}
	if tab.Doc.Dirty() != dirty || tab.Doc.Buffer.History().UndoDepth == 0 {
		t.Fatal("transition rewrote the writable draft or undo history")
	}
	if !tab.Doc.Buffer.Undo() {
		t.Fatal("pending edit lost undo")
	}
	if a.enginePoison == nil || !strings.Contains(a.enginePoison.Error(), "owned cleanup failure") {
		t.Fatalf("old cleanup failure was not retained: %v", a.enginePoison)
	}
}

func TestEnginePackageTransitionPreservesRealPendingNativeDraft(t *testing.T) {
	a, tab, _, _, _ := mindmapWorkerApp(t)
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	openTagsAppPanel(t, tester, 1180, 780)
	typeTagAppInput(t, tester, 1180, 780, "未提交引擎切换🙂", false)
	before := tab.Doc.Buffer.String()
	history := tab.Doc.Buffer.History()
	selection := tab.Doc.Buffer.Selection()
	dirty := tab.Doc.Dirty()
	view := tab.View
	if before != mindmapWorkerBody || history.UndoDepth != 0 || tab.Doc.Buffer.InGroup() {
		t.Fatalf("pending tag unexpectedly wrote source: before=%q history=%+v", before, history)
	}

	runPackageLifecycleJob(t, a, packageResetAction, enginecatalog.MindmapSVG, "")
	tester.Frame()
	if tab.View != view || tab.Doc.Buffer.String() != before || tab.Doc.Buffer.History() != history || tab.Doc.Buffer.Selection() != selection || tab.Doc.Dirty() != dirty {
		t.Fatal("package transition rewrote or discarded the pending native draft")
	}
	requireTagAppInputText(t, tester, 1180, 780, mindmapTagsInput, "未提交引擎切换🙂")
}

func TestEnginePackageTransitionDropsStaleReadyExportWithoutReleasingNewReservation(t *testing.T) {
	a, _, f, queue, _ := workerApp(t)
	dialogs := 0
	a.exportDialog = func(mygo.SaveDialogOptions) (string, error) {
		dialogs++
		return filepath.Join(t.TempDir(), "stale.html"), nil
	}
	a.exportCurrent()
	oldExport := a.exportJob
	oldLane := a.engineLane
	oldCall := awaitExportCall(t, f)
	oldCall.response <- exportSuccess()
	awaitExportDone(t, oldExport)
	if !oldLane.reserved {
		t.Fatal("ready export lost its lane reservation")
	}

	runPackageLifecycleJob(t, a, packageResetAction, enginecatalog.MarkdownHTML, "")
	freshRunner := newControlledExporter()
	a.managedFactory = func(*engineruntime.PackageManager) (exportRunner, error) { return freshRunner, nil }
	a.exportCurrent()
	newExport := a.exportJob
	newLane := a.engineLane
	newCall := awaitExportCall(t, freshRunner)
	newLane.mu.Lock()
	reserved := newLane.reserved
	newLane.mu.Unlock()
	if newExport == nil || newLane == oldLane || !reserved {
		t.Fatal("new controlled export lane was not reserved")
	}

	drainOneExportCallback(t, queue)
	if dialogs != 0 || a.exportJob != newExport || oldExport.result.Payload != nil {
		t.Fatal("stale ready export published or replaced the current owner")
	}
	newLane.mu.Lock()
	reserved = newLane.reserved
	newLane.mu.Unlock()
	if !reserved {
		t.Fatal("stale export release cleared the new lane reservation")
	}
	newCall.response <- exportSuccess()
	awaitExportDone(t, newExport)
	drainOneExportCallback(t, queue)
	if dialogs != 1 {
		t.Fatalf("current export did not publish once: dialogs=%d", dialogs)
	}
}

func TestEnginePackageTransitionRefreshesReadonlySceneWithoutRewritingDraft(t *testing.T) {
	a, tab, f, _, _ := previewApp(t)
	writable := a.appendTab(document.New(mindmapWorkerBody), "draft.mindmap", "", false, false)
	if !a.setMode(tab, "split") {
		t.Fatal(a.Message)
	}
	a.Active = 0
	tester, call, preview := mountPreview(t, a, f, 900, 650)
	call.response <- previewSuccess()
	awaitPreviewDone(t, preview)
	tester.Frame()
	if err := writable.Doc.Buffer.ReplaceUTF16(0, 0, "draft "); err != nil {
		t.Fatal(err)
	}
	writableDirty, writableUndo := writable.Doc.Dirty(), writable.Doc.Buffer.History().UndoDepth
	freshRunner := newControlledExporter()
	a.managedFactory = func(*engineruntime.PackageManager) (exportRunner, error) {
		return freshRunner, nil
	}

	runPackageLifecycleJob(t, a, packageResetAction, enginecatalog.MindmapPreview, "")
	tester.Frame()
	fresh := awaitExportCall(t, freshRunner)
	if a.previewJob == nil || a.previewJob == preview {
		t.Fatal("transition did not refresh the readonly scene")
	}
	fresh.response <- previewSuccess()
	awaitPreviewDone(t, a.previewJob)
	tester.Frame()
	if writable.Doc.Dirty() != writableDirty || writable.Doc.Buffer.History().UndoDepth != writableUndo {
		t.Fatal("readonly refresh rewrote the writable draft or undo history")
	}
	if len(tab.View.Warnings()) != 0 {
		t.Fatal(tab.View.Warnings())
	}
	if a.enginePoison != nil {
		t.Fatal("successful transition retained poison")
	}
}

func TestEnginePackageCleanupPoisonRemainsVisibleAndBlocksNextTransition(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	poison := errors.New("owned cleanup failure")
	a.enginePoison = poison
	a.startPackageWork(packageActivateAction, enginecatalog.MarkdownHTML, "digest")
	if a.packageJob != nil || !strings.Contains(a.Message, "无法切换引擎") {
		t.Fatalf("poisoned activation was admitted: job=%v message=%q", a.packageJob, a.Message)
	}
	a.startPackageWork(packageListAction, enginecatalog.MarkdownHTML, "")
	if a.packageJob == nil {
		t.Fatal("poison incorrectly blocked diagnostic list")
	}
	awaitPackageDone(t, a.packageJob)
	a.deliverPackageWork()
	list := a.packageLists[enginecatalog.MarkdownHTML]
	if a.packageJob != nil || !errors.Is(a.enginePoison, poison) || !list.checked || list.err != nil || len(list.receipts) != 0 {
		t.Fatalf("list delivery did not retain poison truthfully: job=%v poison=%v list=%#v", a.packageJob, a.enginePoison, list)
	}
	a.startPackageWork(packageActivateAction, enginecatalog.MarkdownHTML, "digest")
	if a.packageJob != nil || !strings.Contains(a.Message, "无法切换引擎") {
		t.Fatalf("poisoned activation was admitted after list: job=%v message=%q", a.packageJob, a.Message)
	}
}

func TestEnginePackageStaleModalAndGenerationDoNotReplaceCurrentWork(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	fixture := enginePackageFixture(t, enginecatalog.MarkdownHTML, "app.stale.pending")
	a.modalDepth++
	a.startPackageWork(packageImportAction, enginecatalog.MarkdownHTML, fixture)
	job := a.packageJob
	if job == nil {
		t.Fatal("package import did not start")
	}
	awaitPackageDone(t, job)
	a.deliverPackageWork()
	if a.packageJob != job {
		t.Fatal("modal depth did not defer package delivery")
	}
	a.packageGeneration++
	a.modalDepth--
	a.deliverPackageWork()
	if a.packageJob != nil {
		t.Fatalf("stale generation did not clear the discarded package work: job=%v", a.packageJob)
	}
	if strings.Contains(a.packageOutcome, "已导入") || len(a.packageLists[enginecatalog.MarkdownHTML].receipts) != 0 {
		t.Fatalf("stale generation delivered package data: outcome=%q list=%#v", a.packageOutcome, a.packageLists[enginecatalog.MarkdownHTML])
	}
	a.startPackageWork(packageListAction, enginecatalog.MarkdownHTML, "")
	if a.packageJob == nil {
		t.Fatal("package list did not start")
	}
	awaitPackageDone(t, a.packageJob)
	a.deliverPackageWork()
	list := a.packageLists[enginecatalog.MarkdownHTML]
	if list.err != nil {
		t.Fatalf("list failed after import: %v", list.err)
	}
	if len(list.receipts) == 0 {
		t.Fatal("delivered package work did not retain its receipt")
	}
}

func TestEnginePackageShutdownJoinsWithoutUIDispatch(t *testing.T) {
	a, _, f, _, _ := workerApp(t)
	a.exportCurrent()
	if a.exportJob == nil {
		t.Fatal("export did not start")
	}
	exportJob := a.exportJob
	_ = awaitExportCall(t, f)
	fixture := enginePackageFixture(t, enginecatalog.MarkdownHTML, "app.shutdown.pending")
	a.startPackageWork(packageImportAction, enginecatalog.MarkdownHTML, fixture)
	job := a.packageJob
	if job == nil {
		t.Fatal("package import did not start")
	}
	a.Shutdown()
	awaitExportDone(t, exportJob)
	awaitPackageDone(t, job)
	select {
	case call := <-f.calls:
		t.Fatalf("shutdown submitted another engine call: %s", call.id)
	default:
	}
	if a.packageJob != nil || a.engineSwitching {
		t.Fatal("shutdown retained package state")
	}
	if len(a.packageLists[enginecatalog.MarkdownHTML].receipts) != 0 {
		t.Fatal("shutdown delivered package completion")
	}
	if a.packageInitErr != nil {
		t.Fatalf("shutdown changed package init error: %v", a.packageInitErr)
	}
}

func TestEnginePackageCloseRefusalLeavesPackageWorkAlive(t *testing.T) {
	a, _, f, _, _ := workerApp(t)
	a.exportCurrent()
	if a.exportJob == nil {
		t.Fatal("export did not start")
	}
	exportJob := a.exportJob
	call := awaitExportCall(t, f)
	fixture := enginePackageFixture(t, enginecatalog.MarkdownHTML, "app.close.pending")
	a.startPackageWork(packageImportAction, enginecatalog.MarkdownHTML, fixture)
	job := a.packageJob
	if job == nil {
		t.Fatal("package import did not start")
	}
	blocked := filepath.Join(t.TempDir(), "blocked-config")
	if err := os.WriteFile(blocked, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	a.configDir = blocked
	a.closeDialog = func(*Tab) (closeDecision, error) { return closeDiscard, nil }
	a.modalDepth++
	if a.CanClose() {
		t.Fatal("persistence refusal accepted close")
	}
	if a.packageJob != job {
		t.Fatal("close refusal canceled or replaced package work")
	}
	a.modalDepth--
	a.deliverPackageWork()
	call.response <- exportSuccess()
	awaitExportDone(t, exportJob)
	awaitPackageDone(t, job)
}
