package app

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"levnote.local/native/core/document"
	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/core/engineruntime"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

type exportRunner interface {
	Export(context.Context, engineipc.EngineID, engineruntime.Source) (engineruntime.Result, error)
	Close() error
}

// Identity is captured on UI. Only the bounded result slot below crosses
// threads; no background operation reads App/Tab/Buffer fields.
type exportJob struct {
	generation       uint64
	engineGeneration uint64
	lane             *engineLane
	tab              *Tab
	doc              *document.Document
	buffer           *text.Buffer
	input            *editor.Editor
	version          uint64
	adapter          *editor.Adapter
	capability       editor.ExportEngine
	name             string
	mode             string
	view             editor.View
	page             editor.ExportPage
	token            editor.ExportToken
	cancel           context.CancelFunc
	done             chan struct{}
	delivering       bool // UI only; modal nested callbacks cannot deliver twice.
	mu               sync.Mutex
	result           engineruntime.Result
	err              error
	ready            bool
	invalid          string
}

type exportEngineStatus struct {
	checked        bool
	lastVerified   engineruntime.EngineInfo
	latestVerified bool
	outcome        string
}

func (j *exportJob) complete(result engineruntime.Result, err error) {
	j.mu.Lock()
	if j.invalid != "" {
		result.Payload = nil
		result.Warnings = nil
	}
	j.result, j.err, j.ready = result, err, true
	close(j.done)
	j.mu.Unlock()
}

func (j *exportJob) invalidate(reason string) {
	j.mu.Lock()
	if j.invalid == "" {
		j.invalid = reason
	}
	j.result.Payload = nil
	j.result.Warnings = nil
	j.mu.Unlock()
	j.cancel()
}

func (j *exportJob) clear() {
	j.mu.Lock()
	j.result = engineruntime.Result{}
	j.err = nil
	j.mu.Unlock()
}

func (a *App) startEngineExport(t *Tab) {
	if a.shuttingDown.Load() || t == nil || t != a.Current() || t.Doc == nil || t.Doc.Buffer == nil || t.Adapter == nil || t.Adapter.ExportEngine == nil {
		return
	}
	capability, adapter, doc, buffer, input := *t.Adapter.ExportEngine, t.Adapter, t.Doc, t.Doc.Buffer, t.Editor
	mode, view := t.Mode, t.View
	if a.refuseUnpreparedViewExport(t, capability) {
		return
	}
	name, generation := t.Name, a.exportGeneration
	descriptor, err := exportDescriptor(capability)
	if err != nil {
		a.report(err)
		return
	}
	if !a.flushTab(t, "导出") {
		return
	}
	// Flush may synchronously report, recover a draft or invoke a host callback.
	// Its own field/source commit may advance version; replacing an owner may
	// not retarget this export to another tab, buffer or advertised operation.
	if a.shuttingDown.Load() || a.Current() != t || a.exportJob != nil || a.exportGeneration != generation ||
		t.Doc != doc || t.Doc.Buffer != buffer || t.Editor != input || t.Adapter != adapter || t.Name != name ||
		t.Mode != mode || t.View != view ||
		t.Adapter.ExportEngine == nil || *t.Adapter.ExportEngine != capability {
		a.report(errors.New("导出前文档或格式已变化，请重新导出"))
		return
	}
	if buffer.InGroup() {
		a.report(errors.New("还有未完成的编辑事务，请完成后再导出"))
		return
	}
	if int64(buffer.LenBytes()) > descriptor.SourceMaxBytes {
		name := "Markdown 源码"
		switch descriptor.Engine {
		case enginecatalog.MindmapSVG:
			name = "思维导图源码"
		case enginecatalog.DrawioSVG:
			name = "图纸源码"
		}
		a.report(fmt.Errorf("%s超过 %d MiB 导出上限", name, descriptor.SourceMaxBytes>>20))
		return
	}
	version := buffer.Version()
	page := editor.ExportPage{}
	token := editor.ExportToken(0)
	var requestPage *engineipc.RequestPage
	if capability.RequiresView {
		preparer, ok := view.(editor.ExportRequestPreparer)
		if !ok {
			a.report(errors.New("当前图纸视图不支持页面级导出"))
			return
		}
		request, err := preparer.PrepareExportRequest()
		if err != nil {
			a.report(err)
			return
		}
		page, token = request.Page, request.Token
		// Preparation may synchronously report and trigger a host callback.
		// It must not silently retarget an already admitted document or view.
		if a.shuttingDown.Load() || a.Current() != t || a.exportJob != nil || a.exportGeneration != generation ||
			t.Doc != doc || t.Doc.Buffer != buffer || t.Editor != input || t.Adapter != adapter || t.Name != name ||
			t.Mode != mode || t.View != view || t.Adapter.ExportEngine == nil ||
			*t.Adapter.ExportEngine != capability || buffer.Version() != version || buffer.InGroup() ||
			!exportViewTokenMatches(view, token) {
			a.report(errors.New("导出准备期间文档或格式已变化，请重新导出"))
			return
		}
		requestPage = &engineipc.RequestPage{Index: page.Index, ID: page.ID}
	}
	if err := a.ensureEngineLane(); err != nil {
		a.report(err)
		return
	}
	a.exportGeneration++
	ctx, cancel := context.WithCancel(context.Background())
	j := &exportJob{generation: a.exportGeneration, tab: t, doc: doc, buffer: buffer, input: input,
		version: version, adapter: adapter, capability: capability, name: name,
		mode: mode, view: view, page: page, token: token,
		cancel: cancel, done: make(chan struct{}), lane: a.engineLane, engineGeneration: a.engineGeneration}
	source := engineruntime.Source{Size: int64(buffer.LenBytes()), Data: buffer.Snapshot(), Page: requestPage}
	a.exportJob = j
	a.Message = "正在导出…"
	task := &engineTask{ctx: ctx, cancel: cancel, id: engineipc.EngineID(j.capability.ID), source: source, export: true, complete: j.complete}
	if a.Window != nil {
		task.notify = a.Window.Invalidate
	}
	if err := j.lane.reserveExport(task); err != nil {
		a.exportJob = nil
		cancel()
		a.report(err)
		return
	}
	// Capture the headless test dispatcher on UI. Production invalidates the
	// window after completion; its next View drains the protected result slot.
	dispatch := a.uiDispatch
	if a.Window == nil && dispatch != nil {
		// A headless test dispatch seam is notified only after completion. It
		// cannot delay runner cleanup, lane exit, or the job's done barrier.
		go func() {
			<-j.done
			dispatch(func() {
				if !a.shuttingDown.Load() {
					a.deliverEngineExport(j)
				}
			})
		}()
	}
}

func (a *App) validExportIdentity(j *exportJob) bool {
	valid := !a.shuttingDown.Load() && a.exportJob == j && a.exportGeneration == j.generation && j.lane == a.engineLane && j.engineGeneration == a.engineGeneration &&
		a.Current() == j.tab && j.tab.Doc == j.doc && j.tab.Doc.Buffer == j.buffer && j.tab.Editor == j.input &&
		j.tab.Name == j.name && j.tab.Adapter == j.adapter &&
		j.tab.Adapter != nil && j.tab.Adapter.ExportEngine != nil &&
		*j.tab.Adapter.ExportEngine == j.capability && j.buffer.Version() == j.version
	if !valid || !j.capability.RequiresView {
		return valid
	}
	return j.tab.Mode == j.mode && j.tab.View == j.view &&
		exportViewModeValid(j.mode) && exportViewTokenMatches(j.view, j.token)
}

func exportViewModeValid(mode string) bool {
	return mode == "preview" || mode == "split"
}

func exportViewTokenMatches(view editor.View, token editor.ExportToken) bool {
	provider, ok := view.(editor.ExportRequestTokenProvider)
	return ok && view != nil && provider.ExportRequestToken() == token
}

func (a *App) refuseUnpreparedViewExport(t *Tab, capability editor.ExportEngine) bool {
	if !capability.RequiresView {
		return false
	}
	if t == nil || !exportViewModeValid(t.Mode) {
		a.report(errors.New("图纸导出需要阅读或分屏视图；请切换到阅读或分屏并选择页面"))
		return true
	}
	if t.View == nil {
		a.report(errors.New("图纸视图尚未准备好；请等待阅读或分屏视图显示后选择页面"))
		return true
	}
	if _, ok := t.View.(editor.ExportRequestPreparer); !ok {
		a.report(errors.New("当前图纸视图不支持页面级导出"))
		return true
	}
	if _, ok := t.View.(editor.ExportRequestTokenProvider); !ok {
		a.report(errors.New("当前图纸视图无法验证导出页面"))
		return true
	}
	return false
}

func (a *App) recheckExport(j *exportJob) bool {
	if !a.validExportIdentity(j) {
		a.report(errors.New("导出期间文档或格式已变化，请重新导出"))
		return false
	}
	if !a.flushTab(j.tab, "导出") {
		return false
	}
	if !a.validExportIdentity(j) || j.buffer.InGroup() {
		a.report(errors.New("导出期间有新的编辑，请完成输入后重新导出"))
		return false
	}
	return true
}

func (a *App) deliverEngineExport(j *exportJob) {
	if a.shuttingDown.Load() || a.exportJob != j || a.modalDepth > 0 || j.delivering {
		return
	}
	j.mu.Lock()
	ready, runErr, invalid, info := j.ready, j.err, j.invalid, j.result.Engine
	j.mu.Unlock()
	if !ready {
		return
	}
	j.delivering = true
	defer func() {
		j.cancel()
		j.clear()
		if a.exportJob == j {
			a.exportJob = nil
		}
		a.releaseExportReservation(j)
	}()
	status := a.exportStatuses[j.capability.ID]
	status.checked, status.latestVerified = true, info.ID != ""
	if info.ID != "" {
		status.lastVerified = info
	}
	status.outcome = "引擎处理完成；尚未发布文件"
	if runErr != nil {
		status.outcome = "失败：" + runErr.Error()
	}
	var runtimeFailure *engineruntime.Failure
	cleanupFailed := errors.As(runErr, &runtimeFailure) && runtimeFailure.Kind == engineruntime.FailureCleanup
	if invalid != "" && !cleanupFailed {
		status.outcome = "导出已取消"
	}
	if a.exportStatuses == nil {
		a.exportStatuses = make(map[string]exportEngineStatus)
	}
	a.exportStatuses[j.capability.ID] = status
	if invalid != "" {
		if cleanupFailed {
			a.report(fmt.Errorf("导出进程清理失败：%w", runErr))
		}
		return
	}
	if runErr != nil {
		a.report(fmt.Errorf("导出失败：%w", runErr))
		return
	}
	if !a.recheckExport(j) {
		a.exportOutcome(j, "文档或输入已变化，未发布文件")
		return
	}
	base := j.name
	if dot := strings.LastIndexByte(base, '.'); dot > 0 {
		base = base[:dot]
	}
	var path string
	var err error
	func() {
		defer a.deferExportDelivery()()
		path, err = a.chooseExportPath(mygo.SaveDialogOptions{Parent: a.Window, Title: "导出 " + strings.ToUpper(j.capability.Format), DefaultPath: base + "." + j.capability.Format})
	}()
	if err != nil {
		a.exportOutcome(j, "选择导出位置失败，未发布文件")
		a.report(err)
		return
	}
	if path == "" {
		a.exportOutcome(j, "导出已取消")
		a.Message = "已取消导出"
		return
	}
	if !a.recheckExport(j) {
		a.exportOutcome(j, "文档或输入已变化，未发布文件")
		return
	}
	j.mu.Lock()
	result, invalid := j.result, j.invalid
	j.result = engineruntime.Result{}
	j.mu.Unlock()
	if invalid != "" {
		a.exportOutcome(j, "导出已取消")
		return
	}
	// Publication is the existing synchronous exclusive atomic write. Once this
	// call starts it cannot be canceled; disk latency remains on UI in this slice.
	if err := writeExportFile(path, result.Payload); err != nil {
		a.exportOutcome(j, "文件写入失败")
		a.report(fmt.Errorf("导出失败：%w", err))
		return
	}
	for _, warning := range result.Warnings {
		a.report(fmt.Errorf("导出提示 [%s]：%s", warning.Code, warning.Message))
	}
	a.Message = "已导出 " + path
	a.exportOutcome(j, "文件已导出")
	if len(result.Warnings) > 0 {
		a.Message += fmt.Sprintf("（%d 项提示，见问题记录）", len(result.Warnings))
	}
}

func (a *App) exportOutcome(j *exportJob, outcome string) {
	status := a.exportStatuses[j.capability.ID]
	status.outcome = outcome
	a.exportStatuses[j.capability.ID] = status
}

func (a *App) cancelEngineExport(reason string) {
	if j := a.exportJob; j != nil {
		a.exportGeneration++
		j.invalidate(reason)
		a.Message = reason
	}
}

func (a *App) invalidateTabExport(t *Tab, reason string) {
	a.cancelPreview(t)
	if a.exportJob != nil && a.exportJob.tab == t {
		a.cancelEngineExport(reason)
	}
}

// Cleanup never needs UI dispatch. Close remains idempotent in the runner;
// waiting done joins the small result-slot handoff after runner cleanup.
func (a *App) closeEngineExport() error {
	packageErr := a.joinPackageWork()
	a.cancelPreview(nil)
	j := a.exportJob
	if j != nil {
		j.invalidate("已取消导出")
		a.exportGeneration++
	}
	var err error
	if a.engineLane != nil {
		err = a.engineLane.close()
	} else if a.exporter != nil {
		err = a.exporter.Close()
	}
	if j != nil {
		<-j.done
		if err != nil {
			j.mu.Lock()
			info := j.result.Engine
			j.mu.Unlock()
			status := a.exportStatuses[j.capability.ID]
			status.checked, status.latestVerified = true, info.ID != ""
			if info.ID != "" {
				status.lastVerified = info
			}
			status.outcome = "清理失败：" + err.Error()
			if a.exportStatuses == nil {
				a.exportStatuses = make(map[string]exportEngineStatus)
			}
			a.exportStatuses[j.capability.ID] = status
		}
		j.clear()
		if a.exportJob == j {
			a.exportJob = nil
		}
	}
	return errors.Join(packageErr, err, a.enginePoison)
}

// Native dialogs can pump nested UI callbacks. Deferral spans the full owning
// operation, so failed Save/Close keeps an unchanged job and resumes delivery
// only when every enclosing modal operation has unwound.
func (a *App) deferExportDelivery() func() {
	a.modalDepth++
	return func() {
		a.modalDepth--
		if a.modalDepth == 0 && a.exportJob != nil {
			a.deliverEngineExport(a.exportJob)
		}
		if a.modalDepth == 0 {
			a.deliverBinary()
			a.deliverPackageWork()
			a.deliverScene()
			a.scheduleAutosave()
		}
	}
}

func (a *App) chooseExportPath(options mygo.SaveDialogOptions) (string, error) {
	if a.exportDialog != nil {
		return a.exportDialog(options)
	}
	return mygo.Dialog.Save(options)
}

func (a *App) chooseOpenPaths(options mygo.OpenDialogOptions) ([]string, error) {
	if a.openDialog != nil {
		return a.openDialog(options)
	}
	return mygo.Dialog.Open(options)
}

func (a *App) exportStatusText(capability editor.ExportEngine) string {
	descriptor, err := exportDescriptor(capability)
	if err != nil {
		return "本地引擎导出不可用 · " + err.Error()
	}
	prefix := "本地引擎 " + strings.ToUpper(descriptor.Format) + " 导出 · "
	status := a.exportStatuses[capability.ID]
	if !status.checked {
		return prefix + "未检查，导出时验证；界面与编辑原生运行，只读导图场景由所选程序计算"
	}
	label := prefix + status.outcome
	if status.lastVerified.ID != "" {
		prefix := "本次已验证 "
		if !status.latestVerified {
			prefix = "历史已验证（本次未通过验证） "
		}
		hash := status.lastVerified.SHA256
		if len(hash) > 12 {
			hash = hash[:12]
		}
		source := "随包"
		if status.lastVerified.Source == engineruntime.PackageInstalled {
			source = "已安装 · 包 " + shortPackageDigest(status.lastVerified.PackageDigest)
		}
		label += "\n" + prefix + source + " · " + status.lastVerified.Version + " · SHA256 " + hash
	}
	return label + "\n界面与编辑原生运行；只读导图场景由所选程序计算"
}

func exportDescriptor(capability editor.ExportEngine) (enginecatalog.Descriptor, error) {
	descriptor, ok := enginecatalog.Lookup(enginecatalog.EngineID(capability.ID))
	if !ok || descriptor.Format != capability.Format {
		return enginecatalog.Descriptor{}, errors.New("导出引擎与格式声明不匹配")
	}
	if _, ok := enginecatalog.Match(descriptor.Engine, capability.Format, runtime.GOOS, runtime.GOARCH); !ok {
		return enginecatalog.Descriptor{}, errors.New("当前平台不支持此导出程序包")
	}
	return descriptor, nil
}
