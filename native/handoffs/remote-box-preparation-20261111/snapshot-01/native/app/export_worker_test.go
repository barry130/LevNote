package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/core/engineruntime"
	"levnote.local/native/core/session"
	"levnote.local/native/core/text"
)

type exportCall struct {
	ctx      context.Context
	id       engineipc.EngineID
	source   engineruntime.Source
	response chan exportOutcome
}
type exportOutcome struct {
	result engineruntime.Result
	err    error
}

// ControlledExporter implements the app seam only. Real process ownership was
// separately tested in engineruntime; no fake process/integrity claim is made.
type controlledExporter struct {
	calls              chan *exportCall
	closed             chan struct{}
	once               sync.Once
	closeErr           error
	ignoreCancellation bool
	onClose            func()
}

func newControlledExporter() *controlledExporter {
	return &controlledExporter{calls: make(chan *exportCall, 8), closed: make(chan struct{})}
}
func (f *controlledExporter) Export(ctx context.Context, id engineipc.EngineID, source engineruntime.Source) (engineruntime.Result, error) {
	call := &exportCall{ctx: ctx, id: id, source: source, response: make(chan exportOutcome, 1)}
	f.calls <- call
	if f.ignoreCancellation {
		outcome := <-call.response
		return outcome.result, outcome.err
	}
	select {
	case outcome := <-call.response:
		return outcome.result, outcome.err
	case <-ctx.Done():
		return engineruntime.Result{}, ctx.Err()
	case <-f.closed:
		return engineruntime.Result{}, context.Canceled
	}
}
func (f *controlledExporter) Close() error {
	f.once.Do(func() {
		close(f.closed)
		if f.onClose != nil {
			f.onClose()
		}
	})
	return f.closeErr
}

func awaitExportCall(t *testing.T, f *controlledExporter) *exportCall {
	t.Helper()
	select {
	case call := <-f.calls:
		return call
	case <-time.After(5 * time.Second):
		t.Fatal("export did not reach runner")
		return nil
	}
}
func awaitExportDone(t *testing.T, j *exportJob) {
	t.Helper()
	select {
	case <-j.done:
	case <-time.After(5 * time.Second):
		t.Fatal("export never joined")
	}
}
func drainOneExportCallback(t *testing.T, queue <-chan func()) {
	t.Helper()
	select {
	case callback := <-queue:
		callback()
	case <-time.After(5 * time.Second):
		t.Fatal("export completion not dispatched")
	}
}
func exportSuccess() exportOutcome {
	return exportOutcome{result: engineruntime.Result{Payload: []byte("<!doctype html>\n<p>独立结果 &amp;</p>\n"), Warnings: []engineipc.Warning{}, Engine: engineruntime.EngineInfo{ID: engineipc.MarkdownHTML, Version: "test-v1", SHA256: strings.Repeat("a", 64)}}}
}

func workerApp(t *testing.T) (*App, *Tab, *controlledExporter, chan func(), string) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "note.md")
	if err := os.WriteFile(path, []byte("# 标题\r\n\r\n正文🙂\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ExportEngine == nil || tab.Adapter.ExportEngine.ID != string(engineipc.MarkdownHTML) {
		t.Fatal("registered Markdown descriptor not connected")
	}
	f := newControlledExporter()
	a.exporter = f
	queue := make(chan func(), 16)
	a.uiDispatch = func(fn func()) { queue <- fn }
	return a, tab, f, queue, root
}

type exportForbiddenView struct {
	failingView
	exports int
}

func (v *exportForbiddenView) Export(string) ([]byte, error) {
	v.exports++
	return []byte("forbidden fallback"), nil
}

func TestWorkerExportActualDescriptorAllModesAndPreservesSource(t *testing.T) {
	for _, mode := range []string{"source", "preview", "split"} {
		t.Run(mode, func(t *testing.T) {
			a, tab, f, queue, root := workerApp(t)
			if !a.setMode(tab, mode) {
				t.Fatal(a.Message)
			}
			if mode != "source" {
				tester := ui.NewTester(a.View, 1180, 780)
				tester.Frame()
			} else if tab.View != nil {
				t.Fatal("source route must work without View")
			}
			if err := tab.Doc.Buffer.SetSelection(4, 1); err != nil {
				t.Fatal(err)
			}
			value, version, selection, history, dirty, baseline := tab.Doc.Buffer.String(), tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection(), tab.Doc.Buffer.History(), tab.Doc.Dirty(), tab.Doc.DiskToken()
			path := filepath.Join(root, "out.html")
			dialogs := 0
			a.exportDialog = func(options mygo.SaveDialogOptions) (string, error) {
				dialogs++
				if options.DefaultPath != "note.html" {
					t.Errorf("name=%q", options.DefaultPath)
				}
				return path, nil
			}
			a.exportCurrent()
			job := a.exportJob
			call := awaitExportCall(t, f)
			var snapshot bytes.Buffer
			written, err := call.source.Data.WriteTo(&snapshot)
			if err != nil || written != call.source.Size || snapshot.String() != value || call.id != engineipc.MarkdownHTML {
				t.Fatalf("snapshot/id mismatch: %v", err)
			}
			if dialogs != 0 {
				t.Fatal("SaveDialog opened before joined worker")
			}
			if call.source.Page != nil {
				t.Fatalf("legacy Markdown request carried page %+v", call.source.Page)
			}
			call.response <- exportSuccess()
			awaitExportDone(t, job)
			if dialogs != 0 {
				t.Fatal("worker touched UI without queued delivery")
			}
			drainOneExportCallback(t, queue)
			data, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(data, exportSuccess().result.Payload) || dialogs != 1 || a.exportJob != nil {
				t.Fatalf("publish: %q %v dialogs%d", data, err, dialogs)
			}
			if tab.Doc.Buffer.String() != value || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.Selection() != selection || tab.Doc.Buffer.History() != history || tab.Doc.Dirty() != dirty || !tab.Doc.SameBaseline(baseline) {
				t.Fatal("export changed document state")
			}
			job.mu.Lock()
			retained := len(job.result.Payload)
			job.mu.Unlock()
			if retained != 0 {
				t.Fatal("completed job retained HTML")
			}
		})
	}
}

func TestWorkerExportRefusesWithoutFallbackAndCachesHistoricalVerification(t *testing.T) {
	a, tab, f, queue, _ := workerApp(t)
	view := &exportForbiddenView{}
	tab.View = view
	capability := *tab.Adapter.ExportEngine
	if !strings.Contains(a.exportStatusText(capability), "未检查") {
		t.Fatal("startup claims inspection")
	}
	if len(f.calls) != 0 {
		t.Fatal("status started worker")
	}
	dialogs := 0
	a.exportDialog = func(mygo.SaveDialogOptions) (string, error) { dialogs++; return "", nil }
	a.exportCurrent()
	call := awaitExportCall(t, f)
	first := exportSuccess()
	first.result.Payload = nil
	first.err = &engineruntime.Failure{Kind: engineruntime.FailureEngine, Code: "partial-document", Detail: "内容不完整"}
	call.response <- first
	awaitExportDone(t, a.exportJob)
	drainOneExportCallback(t, queue)
	if dialogs != 0 || view.exports != 0 || !strings.Contains(a.Message, "内容不完整") {
		t.Fatal("worker error fell back or opened dialog")
	}
	if !strings.Contains(a.exportStatusText(capability), "test-v1") {
		t.Fatal("verified failed-run info missing")
	}
	installed := engineruntime.EngineInfo{
		ID: engineipc.EngineID(capability.ID), Version: "installed-v1", SHA256: strings.Repeat("d", 64),
		Source: engineruntime.PackageInstalled, PackageDigest: strings.Repeat("e", 64),
	}
	a.exportStatuses[capability.ID] = exportEngineStatus{checked: true, lastVerified: installed, outcome: "已启用本地包"}
	installedStatus := a.exportStatusText(capability)
	if !strings.Contains(installedStatus, "本地引擎 HTML 导出") || !strings.Contains(installedStatus, "已安装 · 包 "+shortPackageDigest(installed.PackageDigest)) {
		t.Fatalf("installed selection was labeled bundled: %s", installedStatus)
	}
	a.exportCurrent()
	call = awaitExportCall(t, f)
	call.response <- exportOutcome{err: &engineruntime.Failure{Kind: engineruntime.FailureUnavailable, Detail: "缺少随包程序"}}
	awaitExportDone(t, a.exportJob)
	drainOneExportCallback(t, queue)
	label := a.exportStatusText(capability)
	if !strings.Contains(label, "历史已验证") || !strings.Contains(label, "缺少随包程序") || view.exports != 0 {
		t.Fatalf("historical status: %s", label)
	}
}

func TestWorkerExportPreflightAndExclusivePublishing(t *testing.T) {
	for _, kind := range []string{"foreign-group", "oversize", "flush"} {
		t.Run(kind, func(t *testing.T) {
			a, tab, f, _, _ := workerApp(t)
			switch kind {
			case "foreign-group":
				if err := tab.Doc.Buffer.BeginGroup(); err != nil {
					t.Fatal(err)
				}
				defer tab.Doc.Buffer.CancelGroup()
			case "oversize":
				tab.Doc.Buffer = text.New(strings.Repeat("x", engineipc.SourceMaxBytes+1))
				tab.Editor.Buffer = tab.Doc.Buffer
			case "flush":
				tab.View = &failingView{flushErr: errors.New("draft remains")}
			}
			a.exportCurrent()
			if a.exportJob != nil || len(f.calls) != 0 || a.Message == "" {
				t.Fatalf("preflight %s reached worker", kind)
			}
		})
	}
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprint("readonly-existing-", existing), func(t *testing.T) {
			a, tab, f, queue, root := workerApp(t)
			tab.Doc.ReadOnly = true
			tab.Editor.ReadOnly = true
			tab.Conflict = true
			path := filepath.Join(root, "output.html")
			if existing {
				if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			a.exportDialog = func(mygo.SaveDialogOptions) (string, error) { return path, nil }
			a.exportCurrent()
			call := awaitExportCall(t, f)
			outcome := exportSuccess()
			outcome.result.Warnings = []engineipc.Warning{{Code: "unsafe-link", Message: "外部链接未激活", Start: 0, End: 1}}
			call.response <- outcome
			awaitExportDone(t, a.exportJob)
			drainOneExportCallback(t, queue)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if existing {
				if string(data) != "keep" || !strings.Contains(a.Message, "导出失败") {
					t.Fatal("overwrote existing target")
				}
			} else if !bytes.Equal(data, outcome.result.Payload) || !strings.Contains(a.Message, "1 项提示") || !strings.Contains(strings.Join(a.Issues, "\n"), "unsafe-link") {
				t.Fatal("readonly export/warnings missing")
			}
			if !tab.Conflict || !tab.Doc.ReadOnly {
				t.Fatal("export changed readonly/conflict state")
			}
		})
	}
}

func TestWorkerExportLateEditsUndoAndDialogGuards(t *testing.T) {
	for _, action := range []string{"edit", "edit-undo", "pending-before-dialog", "flush-failure", "dialog-edit", "dialog-undo", "dialog-pending-draft", "dialog-cancel-job", "dialog-replace", "dialog-error"} {
		t.Run(action, func(t *testing.T) {
			a, tab, f, queue, root := workerApp(t)
			path := filepath.Join(root, "late.html")
			dialogs := 0
			a.exportDialog = func(mygo.SaveDialogOptions) (string, error) {
				dialogs++
				switch action {
				case "dialog-edit", "dialog-undo":
					_ = tab.Doc.Buffer.ReplaceUTF16(0, 0, "new")
					if action == "dialog-undo" {
						tab.Doc.Buffer.Undo()
					}
				case "dialog-cancel-job":
					a.cancelEngineExport("已取消导出")
				case "dialog-pending-draft":
					tab.View = &mutatingFlushView{buffer: tab.Doc.Buffer}
				case "dialog-replace":
					tab.Doc.Buffer = text.New("replacement")
					tab.Editor.Buffer = tab.Doc.Buffer
				case "dialog-error":
					return "", errors.New("dialog unavailable")
				}
				return path, nil
			}
			a.exportCurrent()
			call := awaitExportCall(t, f)
			switch action {
			case "edit", "edit-undo":
				_ = tab.Doc.Buffer.ReplaceUTF16(0, 0, "new")
				if action == "edit-undo" {
					tab.Doc.Buffer.Undo()
				}
			case "pending-before-dialog":
				tab.View = &mutatingFlushView{buffer: tab.Doc.Buffer}
			case "flush-failure":
				tab.View = &failingView{flushErr: errors.New("draft recoverable")}
			}
			call.response <- exportSuccess()
			awaitExportDone(t, a.exportJob)
			drainOneExportCallback(t, queue)
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("stale export published: %v", err)
			}
			if strings.HasPrefix(action, "dialog-") {
				if dialogs != 1 {
					t.Fatal("expected one post-worker dialog")
				}
			} else if dialogs != 0 {
				t.Fatal("stale draft reached dialog")
			}
			if a.exportJob != nil {
				t.Fatal("finished stale job retained")
			}
		})
	}
}

type mutatingFlushView struct {
	failingView
	buffer *text.Buffer
}

func (v *mutatingFlushView) Flush() error { return v.buffer.ReplaceUTF16(0, 0, "pending") }

func TestWorkerExportLifecycleSuccessAndFailure(t *testing.T) {
	for _, action := range []string{"select", "new", "recover", "reload", "save-as", "plugin-disable", "close-tab", "mode", "failed-select", "failed-close", "failed-save"} {
		t.Run(action, func(t *testing.T) {
			a, tab, f, queue, root := workerApp(t)
			a.appendTab(document.New("other"), "other.txt", "", false, false)
			dialogs := 0
			a.exportDialog = func(mygo.SaveDialogOptions) (string, error) { dialogs++; return "", nil }
			a.exportCurrent()
			job := a.exportJob
			call := awaitExportCall(t, f)
			survives := strings.HasPrefix(action, "failed-") || action == "mode"
			switch action {
			case "select":
				if !a.Select(1) {
					t.Fatal(a.Message)
				}
			case "new":
				a.NewDocument("text")
			case "recover":
				if err := a.recoverDraft("draft.md", "draft"); err != nil {
					t.Fatal(err)
				}
			case "reload":
				a.Reload()
			case "save-as":
				a.saveDialog = func(*Tab) (string, error) { return filepath.Join(root, "copy.md"), nil }
				if !a.Save(tab, true) {
					t.Fatal(a.Message)
				}
			case "plugin-disable":
				a.setPluginEnabled("markdown", false)
			case "close-tab":
				if !a.CloseTab(0) {
					t.Fatal(a.Message)
				}
			case "mode":
				if !a.setMode(tab, "source") {
					t.Fatal(a.Message)
				}
			case "failed-select":
				tab.View = &failingView{flushErr: errors.New("keep")}
				if a.Select(1) {
					t.Fatal("failed Flush switched")
				}
				tab.View = nil
			case "failed-close":
				a.closeDialog = func(*Tab) (closeDecision, error) { return closeCancel, nil }
				tab.Doc.IsNew = true
				if a.CloseTab(0) {
					t.Fatal("cancel closed")
				}
			case "failed-save":
				a.saveDialog = func(*Tab) (string, error) { return "", errors.New("cancel save") }
				if a.Save(tab, true) {
					t.Fatal("failed save succeeded")
				}
			}
			if survives {
				select {
				case <-call.ctx.Done():
					t.Fatal("failed operation/same-buffer mode canceled job")
				default:
				}
				call.response <- exportSuccess()
			} else {
				select {
				case <-call.ctx.Done():
				case <-time.After(5 * time.Second):
					t.Fatal("successful mutation did not cancel")
				}
			}
			awaitExportDone(t, job)
			drainOneExportCallback(t, queue)
			if dialogs != btoi(survives) {
				t.Fatalf("dialogs=%d survives=%v", dialogs, survives)
			}
		})
	}
}
func btoi(v bool) int {
	if v {
		return 1
	}
	return 0
}

func TestWorkerExportNestedModalDeferral(t *testing.T) {
	for _, action := range []string{"close-tab-save-cancel", "can-close-save-cancel", "open-cancel", "folder-cancel", "plugin-install-cancel", "save-error"} {
		t.Run(action, func(t *testing.T) {
			a, tab, f, queue, _ := workerApp(t)
			tab.Doc.IsNew = true
			inside := false
			dialogs := 0
			a.exportDialog = func(mygo.SaveDialogOptions) (string, error) {
				dialogs++
				if inside || a.modalDepth != 1 {
					t.Errorf("export dialog nested inside other modal: depth%d", a.modalDepth)
				}
				return "", nil
			}
			a.exportCurrent()
			job := a.exportJob
			call := awaitExportCall(t, f)
			finishInside := func() {
				call.response <- exportSuccess()
				awaitExportDone(t, job)
				drainOneExportCallback(t, queue)
				if dialogs != 0 {
					t.Error("completion opened a nested dialog")
				}
			}
			a.saveDialog = func(*Tab) (string, error) {
				inside = true
				defer func() { inside = false }()
				finishInside()
				return "", errors.New("save canceled")
			}
			a.closeDialog = func(*Tab) (closeDecision, error) { return closeSave, nil }
			a.openDialog = func(mygo.OpenDialogOptions) ([]string, error) {
				inside = true
				defer func() { inside = false }()
				finishInside()
				return nil, nil
			}
			switch action {
			case "close-tab-save-cancel":
				if a.CloseTab(0) {
					t.Fatal("failed save closed")
				}
			case "can-close-save-cancel":
				if a.CanClose() {
					t.Fatal("failed save closed window")
				}
			case "open-cancel":
				a.ChooseOpen()
			case "folder-cancel":
				a.ChooseFolder()
			case "plugin-install-cancel":
				a.installPlugin()
			case "save-error":
				if a.Save(tab, true) {
					t.Fatal("save failure accepted")
				}
			}
			if dialogs != 1 || a.modalDepth != 0 || a.exportJob != nil || a.Current() != tab {
				t.Fatalf("deferred completion dialogs%d depth%d retained%v", dialogs, a.modalDepth, a.exportJob != nil)
			}
		})
	}
}

func TestWorkerExportShutdownAndAcceptedCloseJoinWithoutDispatch(t *testing.T) {
	for _, action := range []string{"shutdown-running", "shutdown-ready", "accepted-close", "failed-persistence", "cleanup-error"} {
		t.Run(action, func(t *testing.T) {
			a, tab, f, queue, root := workerApp(t)
			dialogs := 0
			a.exportDialog = func(mygo.SaveDialogOptions) (string, error) { dialogs++; return filepath.Join(root, "never.html"), nil }
			a.exportCurrent()
			job := a.exportJob
			call := awaitExportCall(t, f)
			if action == "shutdown-ready" {
				call.response <- exportSuccess()
				awaitExportDone(t, job)
			}
			switch action {
			case "shutdown-running", "shutdown-ready":
				a.Shutdown()
				a.Shutdown()
			case "accepted-close":
				if !a.CanClose() {
					t.Fatal(a.Message)
				}
				if len(a.Tabs) != 0 {
					t.Fatal("accepted close retained tabs")
				}
			case "failed-persistence":
				a.configDir = filepath.Join(root, "config-file")
				if err := os.WriteFile(a.configDir, []byte("not directory"), 0600); err != nil {
					t.Fatal(err)
				}
				if a.CanClose() {
					t.Fatal("persistence failure closed")
				}
				select {
				case <-call.ctx.Done():
					t.Fatal("failed persistence canceled job")
				default:
				}
				a.cancelEngineExport("test cleanup")
			case "cleanup-error":
				f.closeErr = errors.New("owned cleanup failure")
				if a.CanClose() {
					t.Fatal("cleanup failure closed")
				}
				if len(a.Tabs) != 1 || a.Current() != tab || !strings.Contains(a.Message, "清理失败") {
					t.Fatal("cleanup failure lost tabs/error")
				}
				if a.CanClose() {
					t.Fatal("stored cleanup failure forgotten")
				}
			}
			awaitExportDone(t, job)
			drainOneExportCallback(t, queue)
			if dialogs != 0 {
				t.Fatal("closed/canceled completion opened dialog")
			}
			job.mu.Lock()
			retained := len(job.result.Payload)
			job.mu.Unlock()
			if retained != 0 {
				t.Fatal("shutdown retained queued HTML")
			}
		})
	}
}

func TestWorkerExportNativeProgressCancelAndStatusImages(t *testing.T) {
	output := os.Getenv("LEVNOTE_WORKER_APP_SCREENSHOT_DIR")
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprint(dark), func(t *testing.T) {
			a, tab, f, queue, _ := workerApp(t)
			a.Dark = dark
			tester := ui.NewTester(a.View, 760, 480)
			tester.SetDark(dark)
			tester.Frame()
			a.exportCurrent()
			job := a.exportJob
			_ = awaitExportCall(t, f)
			tester.Frame()
			button, ok := tester.AccessByRole(ui.RoleButton, "取消导出")
			if !ok || button.Bounds.W < 20 || button.Bounds.H < 24 || button.Bounds.X < 0 || button.Bounds.Y < 0 || button.Bounds.X+button.Bounds.W > 760 || button.Bounds.Y+button.Bounds.H > 451 || !tester.HasText("正在导出…") {
				t.Fatalf("cancel progress bounds: %+v found%v", button, ok)
			}
			if output != "" {
				if err := os.MkdirAll(output, 0755); err != nil {
					t.Fatal(err)
				}
				captureAcceptanceTester(t, filepath.Join(output, fmt.Sprintf("worker-progress-%v-760x480.png", dark)), tester)
			}
			tester.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
			tester.Frame()
			awaitExportDone(t, job)
			drainOneExportCallback(t, queue)
			tester.Frame()
			if a.exportJob != nil || !strings.Contains(a.Message, "取消") {
				t.Fatal("native cancel did not stop job")
			}
			a.ShowPlugins = true
			tester.Frame()
			if !strings.Contains(strings.Join(tester.Texts(), "\n"), "本地引擎 HTML 导出") {
				t.Fatal("cached engine status missing from native plugin panel")
			}
			// A deterministic verified-failure snapshot displays historical metadata;
			// rendering itself must never request another export/inspection.
			a.exportStatuses[tab.Adapter.ExportEngine.ID] = exportEngineStatus{checked: true, lastVerified: exportSuccess().result.Engine, outcome: "失败：随包程序缺失"}
			tester.Frame()
			if len(f.calls) != 0 {
				t.Fatal("plugin frame started worker")
			}
			if output != "" {
				captureAcceptanceTester(t, filepath.Join(output, fmt.Sprintf("worker-status-%v-760x480.png", dark)), tester)
			}
		})
	}
}

func TestWorkerExportCanceledCleanupFailureIsNotHidden(t *testing.T) {
	for _, switchTab := range []bool{false, true} {
		t.Run(fmt.Sprint(switchTab), func(t *testing.T) {
			a, tab, f, queue, _ := workerApp(t)
			f.ignoreCancellation = true
			a.appendTab(document.New("other"), "other.txt", "", false, false)
			dialogs := 0
			a.exportDialog = func(mygo.SaveDialogOptions) (string, error) { dialogs++; return "", nil }
			a.exportCurrent()
			job := a.exportJob
			call := awaitExportCall(t, f)
			if switchTab {
				if !a.Select(1) {
					t.Fatal(a.Message)
				}
			} else {
				a.cancelEngineExport("已取消导出")
			}
			failed := exportSuccess()
			failed.result.Payload = nil
			failed.err = &engineruntime.Failure{Kind: engineruntime.FailureCleanup, Detail: "owned handles could not close"}
			call.response <- failed
			awaitExportDone(t, job)
			drainOneExportCallback(t, queue)
			if dialogs != 0 || !strings.Contains(a.Message, "清理失败") || !strings.Contains(strings.Join(a.Issues, "\n"), "owned handles") || !strings.Contains(a.exportStatusText(*tab.Adapter.ExportEngine), "owned handles") {
				t.Fatal("canceled cleanup failure was hidden")
			}
		})
	}
}

func TestWorkerExportFailedCloseRestoresDiscardedRecovery(t *testing.T) {
	for _, secondaryFailure := range []bool{false, true} {
		t.Run(fmt.Sprint(secondaryFailure), func(t *testing.T) {
			a, existing, f, queue, root := workerApp(t)
			if err := existing.Doc.Buffer.ReplaceUTF16(0, 0, "existing dirty\n"); err != nil {
				t.Fatal(err)
			}
			newTab := a.appendTab(document.New("new dirty🙂"), "new.md", "", false, false)
			a.Persist()
			originalConfig := a.configDir
			f.closeErr = errors.New("cleanup still active")
			if secondaryFailure {
				f.onClose = func() {
					blocked := filepath.Join(root, "blocked-state")
					if err := os.WriteFile(blocked, []byte("file"), 0600); err != nil {
						t.Error(err)
					}
					a.configDir = blocked
				}
			}
			a.closeDialog = func(*Tab) (closeDecision, error) { return closeDiscard, nil }
			a.exportCurrent()
			job := a.exportJob
			_ = awaitExportCall(t, f)
			if a.CanClose() {
				t.Fatal("cleanup failure accepted close")
			}
			awaitExportDone(t, job)
			drainOneExportCallback(t, queue)
			if len(a.Tabs) != 2 || !existing.Doc.Dirty() || newTab.Doc.Buffer.String() != "new dirty🙂" {
				t.Fatal("refused close disposed live drafts")
			}
			if secondaryFailure {
				if !a.persistPending || !strings.Contains(strings.Join(a.Issues, "\n"), "恢复当前草稿失败") || !strings.Contains(strings.Join(a.Issues, "\n"), "清理失败") {
					t.Fatal("secondary recovery failure not explicit/retryable")
				}
				a.configDir = originalConfig
				a.Persist()
			}
			saved, issues := session.Load(filepath.Join(originalConfig, "session.json"))
			if len(issues) != 0 || len(saved.Tabs) != 2 {
				t.Fatalf("restored snapshot: tabs%d issues%v", len(saved.Tabs), issues)
			}
			for index, want := range []string{existing.Doc.Buffer.String(), newTab.Doc.Buffer.String()} {
				if saved.Tabs[index].Draft == nil || *saved.Tabs[index].Draft != want {
					t.Fatal("discard decision survived refused close")
				}
				restored, err := session.RestoreTab(saved.Tabs[index])
				if err != nil || restored.Document.Buffer.String() != want || !restored.Document.IsNew {
					t.Fatalf("next session lost draft: %v", err)
				}
			}
		})
	}
}

func TestWorkerExportFlushesRealPendingCompositionBeforeSnapshot(t *testing.T) {
	a, tab, f, queue, _ := workerApp(t)
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	beginRealMarkdownComposition(t, tab, tester)
	a.exportDialog = func(mygo.SaveDialogOptions) (string, error) { return "", nil }
	a.exportCurrent()
	job := a.exportJob
	call := awaitExportCall(t, f)
	var snapshot bytes.Buffer
	if _, err := call.source.Data.WriteTo(&snapshot); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(snapshot.String(), "# 中文🙂") || snapshot.String() != tab.Doc.Buffer.String() {
		t.Fatalf("pending composition not flushed into snapshot: %q", snapshot.String())
	}
	version, selection, history := tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection(), tab.Doc.Buffer.History()
	call.response <- exportSuccess()
	awaitExportDone(t, job)
	drainOneExportCallback(t, queue)
	if tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.Selection() != selection || tab.Doc.Buffer.History() != history {
		t.Fatal("completion changed post-Flush state")
	}
}

func TestWorkerExportBusyReuseAndSnapshotSurvivesLiveEdit(t *testing.T) {
	a, tab, f, queue, _ := workerApp(t)
	a.exportDialog = func(mygo.SaveDialogOptions) (string, error) { return "", nil }
	before := tab.Doc.Buffer.String()
	a.exportCurrent()
	first := a.exportJob
	call := awaitExportCall(t, f)
	a.exportCurrent()
	if a.exportJob != first || len(f.calls) != 0 || !strings.Contains(a.Message, "已有导出") {
		t.Fatal("busy call replaced admitted job")
	}
	if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, "edited later\n"); err != nil {
		t.Fatal(err)
	}
	var snapshot bytes.Buffer
	if n, err := call.source.Data.WriteTo(&snapshot); err != nil || n != int64(len(before)) || snapshot.String() != before {
		t.Fatal("captured snapshot changed with live edit")
	}
	call.response <- exportSuccess()
	awaitExportDone(t, first)
	drainOneExportCallback(t, queue)
	if a.exportJob != nil {
		t.Fatal("stale first job retained admission")
	}
	a.exportCurrent()
	second := a.exportJob
	call = awaitExportCall(t, f)
	if second == first || second.version != tab.Doc.Buffer.Version() {
		t.Fatal("second job reused stale identity")
	}
	call.response <- exportSuccess()
	awaitExportDone(t, second)
	drainOneExportCallback(t, queue)
	if a.exportJob != nil {
		t.Fatal("next job never completed")
	}
}
