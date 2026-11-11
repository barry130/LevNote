package app

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/core/engineruntime"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

// Independent wire literal: the app tests exercise decoder admission rather
// than constructing their only expected result with the production encoder.
const previewLiteral = `{"schema":1,"layout":"logicalStructureRight","nodeCount":2,"edgeCount":1,"bounds":{"x":0,"y":0,"width":312,"height":96},"nodes":[{"id":"root","label":"预览🙂","x":28,"y":28,"width":92,"height":40,"shape":"rounded","fill":"","stroke":"","text":"","root":true,"collapsed":false},{"id":"folded","label":"折叠","x":192,"y":28,"width":92,"height":40,"shape":"rounded","fill":"#ff0000ff","stroke":"","text":"","root":false,"collapsed":true}],"edges":[{"from":"root","to":"folded","fromPoint":{"x":120,"y":48},"toPoint":{"x":192,"y":48},"stroke":"#123456ff"}]}`

func previewSuccess() exportOutcome {
	return exportOutcome{result: engineruntime.Result{Payload: []byte(previewLiteral), Warnings: []engineipc.Warning{},
		Engine: engineruntime.EngineInfo{ID: engineipc.MindmapPreview, Version: "preview-test", SHA256: strings.Repeat("c", 64)}}}
}

func previewApp(t *testing.T) (*App, *Tab, *controlledExporter, chan func(), string) {
	t.Helper()
	a, tab, runner, queue, root := mindmapWorkerApp(t)
	tab.Doc.ReadOnly, tab.Editor.ReadOnly = true, true
	return a, tab, runner, queue, root
}

// A controlled join can deliberately ignore cancellation while remaining
// bounded on test cleanup, including Fatal before the test sends its outcome.
type joiningPreviewRunner struct {
	*controlledExporter
	active, peak atomic.Int32
}

func (f *joiningPreviewRunner) Export(ctx context.Context, id engineipc.EngineID, source engineruntime.Source) (engineruntime.Result, error) {
	active := f.active.Add(1)
	defer f.active.Add(-1)
	for peak := f.peak.Load(); active > peak; peak = f.peak.Load() {
		if f.peak.CompareAndSwap(peak, active) {
			break
		}
	}
	call := &exportCall{ctx: ctx, id: id, source: source, response: make(chan exportOutcome, 1)}
	f.calls <- call
	select {
	case outcome := <-call.response:
		return outcome.result, outcome.err
	case <-f.closed:
		return engineruntime.Result{}, context.Canceled
	}
}

func controlledPreviewJoin(a *App, f *controlledExporter) *joiningPreviewRunner {
	runner := &joiningPreviewRunner{controlledExporter: f}
	a.exporter = runner
	return runner
}

func mountPreview(t *testing.T, a *App, f *controlledExporter, width, height int) (*ui.Tester, *exportCall, *previewJob) {
	t.Helper()
	tester := ui.NewTester(a.View, width, height)
	tester.Frame()
	call := awaitExportCall(t, f)
	if call.id != engineipc.MindmapPreview || a.previewJob == nil {
		t.Fatal("readonly adapter did not request scene", call.id)
	}
	return tester, call, a.previewJob
}

func awaitPreviewDone(t *testing.T, j *previewJob) {
	t.Helper()
	select {
	case <-j.done:
	case <-time.After(5 * time.Second):
		t.Fatal("preview result did not join")
	}
}

func noEngineCall(t *testing.T, runner *controlledExporter) {
	t.Helper()
	select {
	case call := <-runner.calls:
		t.Fatalf("unexpected call %s", call.id)
	default:
	}
}

func TestScenePreviewRegisteredReadonlySnapshotAndNoSourceFallback(t *testing.T) {
	a, tab, f, _, _ := previewApp(t)
	// This invalid parser input must still reach the fake worker. A view-side
	// concrete Parse fallback would reject it instead of showing literal DTO.
	if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), "not mindmap JSON 🙂"); err != nil {
		t.Fatal(err)
	}
	before, version, history, selection := tab.Doc.Buffer.String(), tab.Doc.Buffer.Version(), tab.Doc.Buffer.History(), tab.Doc.Buffer.Selection()
	tester, call, job := mountPreview(t, a, f, 760, 480)
	var snapshot bytes.Buffer
	n, err := call.source.Data.WriteTo(&snapshot)
	if err != nil || n != call.source.Size || snapshot.String() != before {
		t.Fatal("snapshot", err)
	}
	call.response <- previewSuccess()
	awaitPreviewDone(t, job)
	tester.Frame()
	if a.previewJob != nil || !tester.HasText("只读场景") || !tester.HasText("此视图只读；节点属性、标签与图片引用请在源码中查看。") {
		t.Fatal("scene/source limitation missing", tester.Texts())
	}
	if tab.Doc.Buffer.String() != before || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.History() != history || tab.Doc.Buffer.Selection() != selection {
		t.Fatal("readonly preview mutated source")
	}
	job.mu.Lock()
	retained := len(job.result.Scene.Nodes)
	callback := job.complete
	job.mu.Unlock()
	if retained != 0 || callback != nil {
		t.Fatal("delivered result/callback retained")
	}
	a.Dark = true
	tester.Frame()
	tester.Frame()
	noEngineCall(t, f)
	if err := tester.Click("查看只读文档源码"); err != nil || tab.Mode != "source" {
		t.Fatal("source action", err, tab.Mode)
	}
	if !a.setMode(tab, "preview") {
		t.Fatal(a.Message)
	}
	tester.Frame()
	next := awaitExportCall(t, f)
	if next.id != engineipc.MindmapPreview {
		t.Fatal("released view did not reactivate")
	}
}

func TestScenePreviewExportPriorityJoinReservationAndResume(t *testing.T) {
	a, tab, f, queue, root := previewApp(t)
	runner := controlledPreviewJoin(a, f)
	tester, first, preview := mountPreview(t, a, f, 900, 650)
	dialogEntered := false
	a.exportDialog = func(mygo.SaveDialogOptions) (string, error) {
		dialogEntered = true
		noEngineCall(t, f) // reservation lasts through the nested dialog
		return filepath.Join(root, "priority.svg"), nil
	}
	a.exportCurrent() // must return before the fake ignored cancellation joins
	export := a.exportJob
	if export == nil {
		t.Fatal(a.Message)
	}
	select {
	case <-first.ctx.Done():
	default:
		t.Fatal("export did not cancel active preview")
	}
	noEngineCall(t, f)
	first.response <- exportOutcome{err: context.Canceled}
	exportCall := awaitExportCall(t, f)
	if exportCall.id != engineipc.MindmapSVG {
		t.Fatal("preview ran before priority export", exportCall.id)
	}
	select {
	case <-preview.done:
		t.Fatal("preempted generation was terminally canceled")
	default:
	}
	exportCall.response <- mindmapWorkerOutcome()
	awaitExportDone(t, export)
	noEngineCall(t, f)
	drainOneExportCallback(t, queue)
	if !dialogEntered || a.exportJob != nil {
		t.Fatal("publication not completed")
	}
	resumed := awaitExportCall(t, f)
	if resumed.id != engineipc.MindmapPreview || a.previewJob != preview {
		t.Fatal("same valid generation not resumed")
	}
	resumed.response <- previewSuccess()
	awaitPreviewDone(t, preview)
	tester.Frame()
	if len(tab.View.Warnings()) != 0 || a.previewJob != nil {
		t.Fatal("resumed preview stranded", tab.View.Warnings())
	}
	if string(mustReadFile(t, filepath.Join(root, "priority.svg"))) != mindmapWorkerSVG {
		t.Fatal("export payload changed")
	}
	if runner.peak.Load() != 1 || runner.active.Load() != 0 {
		t.Fatal("lane concurrency/join", runner.peak.Load(), runner.active.Load())
	}
}

func TestScenePreviewOwnerReleaseAndCanceledNewestPending(t *testing.T) {
	for _, action := range []string{"source", "tab", "dispose"} {
		t.Run(action, func(t *testing.T) {
			a, tab, f, _, _ := previewApp(t)
			controlledPreviewJoin(a, f)
			tester, running, first := mountPreview(t, a, f, 900, 650)
			owner := tab.View
			if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), "pending newest"); err != nil {
				t.Fatal(err)
			}
			tester.Frame()
			pending := a.previewJob
			if pending == nil || pending == first {
				t.Fatal("pending generation not established")
			}
			a.engineLane.mu.Lock()
			isPending := a.engineLane.pending == pending.task
			a.engineLane.mu.Unlock()
			if !isPending {
				t.Fatal("fixture did not occupy pending slot")
			}
			switch action {
			case "source":
				if !a.setMode(tab, "source") {
					t.Fatal(a.Message)
				}
			case "tab":
				a.NewDocument("text")
			case "dispose":
				a.disposePresentation(tab)
			}
			awaitPreviewDone(t, pending)
			a.engineLane.mu.Lock()
			empty := a.engineLane.pending == nil
			a.engineLane.mu.Unlock()
			pending.mu.Lock()
			dropped := pending.complete == nil && pending.result.Scene.Nodes == nil
			pending.mu.Unlock()
			if !empty || !dropped || pending.task.source.Data != nil || pending.owner != nil || pending.buffer != nil || a.previewJob != nil {
				t.Fatal("canceled pending snapshot/callback/owner retained")
			}
			running.response <- previewSuccess()
			awaitPreviewDone(t, first)
			a.deliverScene()
			noEngineCall(t, f)
			if first.owner != nil || first.buffer != nil || first.result.Scene.Nodes != nil {
				t.Fatal("late completion retained released ownership")
			}
			if action == "dispose" && tab.View != nil {
				t.Fatal("late completion resurrected disposed view")
			}
			if action == "tab" && a.Current() == tab {
				t.Fatal("late completion reselected old tab")
			}
			if action == "source" && tab.Mode != "source" {
				t.Fatal("late completion changed source mode")
			}
			// Reentry is explicit, and may reuse the released owner or create a
			// new one after disposal. Either must receive a fresh generation.
			if action == "tab" && !a.Select(0) {
				t.Fatal(a.Message)
			}
			if action == "source" && !a.setMode(tab, "preview") {
				t.Fatal(a.Message)
			}
			tester.Frame()
			next := awaitExportCall(t, f)
			if a.previewJob == nil || a.previewJob == pending || action == "dispose" && tab.View == owner {
				t.Fatal("reentry ownership not refreshed")
			}
			job := a.previewJob
			next.response <- previewSuccess()
			awaitPreviewDone(t, job)
			tester.Frame()
			if a.previewJob != nil || len(tab.View.Warnings()) != 0 {
				t.Fatal("fresh scene did not install")
			}
		})
	}
}

func TestScenePreviewNewestPendingSnapshotAndStaleSuppression(t *testing.T) {
	a, tab, f, _, _ := previewApp(t)
	controlledPreviewJoin(a, f)
	tester, first, firstJob := mountPreview(t, a, f, 900, 650)
	for _, source := range []string{"second", "third🙂"} {
		if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), source); err != nil {
			t.Fatal(err)
		}
		tester.Frame()
	}
	latest := a.previewJob
	if latest == nil || latest == firstJob {
		t.Fatal("new generation missing")
	}
	var original bytes.Buffer
	_, _ = first.source.Data.WriteTo(&original)
	if original.String() != mindmapWorkerBody {
		t.Fatal("snapshot followed live edits")
	}
	first.response <- previewSuccess()
	awaitPreviewDone(t, firstJob)
	call := awaitExportCall(t, f)
	var snapshot bytes.Buffer
	_, _ = call.source.Data.WriteTo(&snapshot)
	if snapshot.String() != "third🙂" {
		t.Fatal("did not coalesce newest", snapshot.String())
	}
	noEngineCall(t, f)
	call.response <- previewSuccess()
	awaitPreviewDone(t, latest)
	tester.Frame()
	if len(tab.View.Warnings()) != 0 {
		t.Fatal(tab.View.Warnings())
	}
	firstJob.mu.Lock()
	nodes, callback := len(firstJob.result.Scene.Nodes), firstJob.complete
	firstJob.mu.Unlock()
	if nodes != 0 || callback != nil {
		t.Fatal("invalidated old result retained")
	}
}

func TestScenePreviewNestedModalAndInvalidatedCleanupRemainVisible(t *testing.T) {
	for _, cleanup := range []bool{false, true} {
		t.Run(stringBool(cleanup), func(t *testing.T) {
			a, tab, f, _, _ := previewApp(t)
			_, call, job := mountPreview(t, a, f, 900, 650)
			outer, inner := a.deferExportDelivery(), a.deferExportDelivery()
			if cleanup {
				// Preserve an active exact request while its callback is removed.
				call.response <- exportOutcome{err: &engineruntime.Failure{Kind: engineruntime.FailureCleanup, Detail: "owned cleanup sentinel"}}
				awaitPreviewDone(t, job)
				a.cancelPreview(tab)
			} else {
				call.response <- previewSuccess()
				awaitPreviewDone(t, job)
			}
			inner()
			if !cleanup && a.previewJob != job {
				t.Fatal("nested modal delivered early")
			}
			outer()
			if cleanup {
				if !strings.Contains(a.Message, "owned cleanup sentinel") || a.engineLane.failure() == nil {
					t.Fatal("lost invalidated cleanup failure", a.Message)
				}
				if err := a.closeEngineExport(); err == nil || a.closeEngineExport() == nil {
					t.Fatal("poisoned close forgot failure")
				}
			} else if a.previewJob != nil || len(tab.View.Warnings()) != 0 {
				t.Fatal("modal unwind failed to deliver")
			}
		})
	}
}

func stringBool(value bool) string {
	if value {
		return "cleanup"
	}
	return "success"
}

func TestScenePreviewMalformedAndUnavailableDoNotRetry(t *testing.T) {
	for _, scenario := range []string{"unavailable", "malformed"} {
		t.Run(scenario, func(t *testing.T) {
			a, tab, f, _, _ := previewApp(t)
			tester, call, job := mountPreview(t, a, f, 760, 480)
			if scenario == "malformed" {
				out := previewSuccess()
				out.result.Payload = []byte(`{"schema":1}`)
				call.response <- out
			} else {
				call.response <- exportOutcome{err: &engineruntime.Failure{Kind: engineruntime.FailureUnavailable, Detail: "missing preview worker"}}
			}
			awaitPreviewDone(t, job)
			tester.Frame()
			tester.Frame()
			tester.Frame()
			if len(tab.View.Warnings()) == 0 || !strings.Contains(tab.View.Warnings()[0], "不可用") {
				t.Fatal("failure invisible", tab.View.Warnings())
			}
			noEngineCall(t, f)
		})
	}
}

func TestScenePreviewShutdownAndCanceledQueuedExportNoDispatch(t *testing.T) {
	a, _, f, _, _ := previewApp(t)
	controlledPreviewJoin(a, f)
	_, call, preview := mountPreview(t, a, f, 900, 650)
	a.exportCurrent()
	export := a.exportJob
	a.cancelEngineExport("queued cancellation")
	_ = call
	a.Shutdown()
	awaitExportDone(t, export)
	awaitPreviewDone(t, preview)
	noEngineCall(t, f)
	if a.previewJob != nil || a.exportJob != nil {
		t.Fatal("shutdown retained jobs")
	}
}

func TestScenePreviewPreflightAndDisabledAdapter(t *testing.T) {
	a, tab, f, _, _ := previewApp(t)
	capability := *tab.Adapter.SceneEngine
	view := &failingView{}
	tab.View = view
	for _, mutate := range []func(){
		func() { tab.Doc.ReadOnly = false },
		func() { tab.Doc.ReadOnly = true; tab.Mode = "source" },
		func() { tab.Mode = "preview"; tab.Doc.Buffer = text.New(strings.Repeat("x", (16<<20)+1)) },
	} {
		mutate()
		if cancel, err := a.beginScene(tab, view, editor.SceneRequest{Capability: capability}, func(editor.SceneResult, error) { t.Error("preflight callback") }); err == nil || cancel != nil {
			t.Fatal("invalid preview admitted")
		}
	}
	noEngineCall(t, f)
	if err := a.pluginManager.SetEnabled("mindmap", false); err != nil {
		t.Fatal(err)
	}
	a.configureTab(tab)
	if tab.Adapter != nil || tab.Mode != "source" {
		t.Fatal("disabled format retained scene adapter")
	}
}
