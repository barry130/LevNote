package app

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"

	"levnote.local/native/core/document"
	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineruntime"
	"levnote.local/native/core/enginescene"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

type previewJob struct {
	tab              *Tab
	tabID            int
	document         *document.Document
	buffer           *text.Buffer
	version          uint64
	name, path, mode string
	adapter          *editor.Adapter
	owner            editor.View
	request          editor.SceneRequest
	task             *engineTask
	lane             *engineLane
	engineGeneration uint64
	done             chan struct{}
	mu               sync.Mutex
	complete         func(editor.SceneResult, error)
	result           editor.SceneResult
	err              error
	ready, canceled  bool
}

func (j *previewJob) finish(result engineruntime.Result, err error) {
	// No source/app/view reads. Invalidated slots do not decode or retain a
	// large scene. The second check covers cancellation during Decode.
	j.mu.Lock()
	canceled := j.canceled
	j.mu.Unlock()
	var decoded editor.SceneResult
	if !canceled && err == nil {
		decoded.Scene, err = enginescene.Decode(result.Payload)
		if err != nil {
			err = fmt.Errorf("预览场景无效：%w", err)
		}
		if err == nil {
			decoded.Engine = editor.SceneEngineInfo{ID: string(result.Engine.ID), Version: result.Engine.Version, SHA256: result.Engine.SHA256,
				Source: string(result.Engine.Source), PackageDigest: result.Engine.PackageDigest}
			decoded.Warnings = make([]editor.SceneWarning, len(result.Warnings))
			for i, warning := range result.Warnings {
				decoded.Warnings[i] = editor.SceneWarning{Code: warning.Code, Message: warning.Message, Start: warning.Start, End: warning.End}
			}
		}
	}
	result.Payload = nil
	j.mu.Lock()
	if !j.canceled {
		j.result, j.err = decoded, err
	}
	j.ready = true
	close(j.done)
	j.mu.Unlock()
}

func (j *previewJob) invalidate() {
	j.mu.Lock()
	j.canceled = true
	j.complete = nil
	j.result = editor.SceneResult{}
	j.err = nil
	j.mu.Unlock()
	// Identity is UI-owned. A canceled execution needs only its immutable
	// source and completion slot until join, not the released tab/view graph.
	j.tab, j.document, j.buffer, j.adapter, j.owner = nil, nil, nil, nil, nil
	j.name, j.path = "", ""
}

func (a *App) ensureEngineLane() error {
	if a.enginePoison != nil {
		return a.enginePoison
	}
	if a.engineSwitching || a.shuttingDown.Load() {
		return errors.New("引擎正在切换或关闭，请稍候")
	}
	if a.engineLane != nil {
		return a.engineLane.failure()
	}
	if a.exporter == nil {
		if a.packageInitErr != nil {
			return a.packageInitErr
		}
		var runner exportRunner
		var err error
		if a.managedFactory != nil {
			runner, err = a.managedFactory(a.packageManager)
		} else {
			runner, err = engineruntime.NewManaged(a.packageManager)
		}
		if err != nil {
			return err
		}
		a.exporter = runner
	}
	a.engineLane = newEngineLane(a.exporter)
	return nil
}

func (a *App) beginScene(t *Tab, owner editor.View, request editor.SceneRequest, callback func(editor.SceneResult, error)) (func(), error) {
	if a.shuttingDown.Load() || t == nil || a.Current() != t || !a.tabIsOpen(t) || t.View != owner || owner == nil || callback == nil ||
		t.Doc == nil || t.Doc.Buffer == nil || !t.Doc.ReadOnly || t.Mode != "preview" && t.Mode != "split" ||
		t.Adapter == nil || t.Adapter.SceneEngine == nil || *t.Adapter.SceneEngine != request.Capability {
		return nil, errors.New("预览所属文档或格式已变化，请重新进入阅读视图")
	}
	descriptor, ok := enginecatalog.Match(enginecatalog.EngineID(request.Capability.ID), request.Capability.Format, runtime.GOOS, runtime.GOARCH)
	if !ok || descriptor.Engine != enginecatalog.MindmapPreview {
		return nil, errors.New("此格式没有可用的原生场景引擎")
	}
	buffer := t.Doc.Buffer
	if buffer.InGroup() {
		return nil, errors.New("预览等待当前编辑事务完成，请重新进入阅读视图")
	}
	if int64(buffer.LenBytes()) > descriptor.SourceMaxBytes {
		return nil, errors.New("思维导图源码超过 16 MiB 预览上限，请查看源码")
	}
	if err := a.ensureEngineLane(); err != nil {
		return nil, err
	}
	a.cancelPreview(nil)
	ctx, cancel := context.WithCancel(context.Background())
	j := &previewJob{tab: t, tabID: t.ID, document: t.Doc, buffer: buffer, version: buffer.Version(),
		name: t.Name, path: t.Doc.Path, mode: t.Mode, adapter: t.Adapter, owner: owner, request: request,
		complete: callback, done: make(chan struct{}), lane: a.engineLane, engineGeneration: a.engineGeneration}
	j.task = &engineTask{ctx: ctx, cancel: cancel, id: descriptor.Engine,
		source: engineruntime.Source{Size: int64(buffer.LenBytes()), Data: buffer.Snapshot()}, complete: j.finish}
	if a.Window != nil {
		j.task.notify = a.Window.Invalidate
	}
	a.previewJob = j
	if err := j.lane.submitPreview(j.task); err != nil {
		a.previewJob = nil
		j.invalidate()
		cancel()
		return nil, err
	}
	return func() {
		j.invalidate()
		j.lane.cancelTask(j.task)
		if a.previewJob == j {
			a.previewJob = nil
		}
	}, nil
}

func (a *App) validPreview(j *previewJob) bool {
	return j != nil && !a.shuttingDown.Load() && j.lane == a.engineLane && j.engineGeneration == a.engineGeneration && a.previewJob == j && a.Current() == j.tab && a.tabIsOpen(j.tab) &&
		j.tab.ID == j.tabID && j.tab.Doc == j.document && j.tab.Doc.Buffer == j.buffer && j.buffer.Version() == j.version &&
		j.tab.Doc.ReadOnly && !j.buffer.InGroup() && j.tab.Name == j.name && j.tab.Doc.Path == j.path && j.tab.Mode == j.mode &&
		j.tab.View == j.owner && j.tab.Adapter == j.adapter && j.adapter.SceneEngine != nil && *j.adapter.SceneEngine == j.request.Capability
}

func (a *App) cancelPreview(t *Tab) {
	if j := a.previewJob; j != nil && (t == nil || j.tab == t) {
		j.invalidate()
		a.previewJob = nil
		if j.lane != nil {
			j.lane.cancelTask(j.task)
		}
	}
}

func (a *App) deliverScene() {
	if a.shuttingDown.Load() || a.modalDepth > 0 {
		return
	}
	a.reportEngineCleanup()
	j := a.previewJob
	if j == nil {
		return
	}
	if !a.validPreview(j) {
		a.cancelPreview(j.tab)
		return
	}
	j.mu.Lock()
	if !j.ready || j.canceled {
		j.mu.Unlock()
		return
	}
	result, err, callback := j.result, j.err, j.complete
	j.result, j.err, j.complete = editor.SceneResult{}, nil, nil
	j.canceled = true // a callback may synchronously release/reconfigure.
	j.mu.Unlock()
	a.previewJob = nil
	if callback != nil {
		callback(result, err)
	}
}

func (a *App) reportEngineCleanup() {
	if a.engineCleanupReported || a.engineLane == nil {
		return
	}
	if err := a.engineLane.failure(); err != nil {
		a.engineCleanupReported = true
		a.report(fmt.Errorf("引擎进程清理失败：%w", err))
	}
}

// Called on UI before an export reservation releases. No pending snapshot is
// allowed to start after publication without an ownership/version recheck.
func (a *App) releaseExportReservation(j *exportJob) {
	if j == nil || j.lane == nil {
		return
	}
	if a.engineLane == j.lane && a.engineGeneration == j.engineGeneration && a.previewJob != nil && !a.validPreview(a.previewJob) {
		a.cancelPreview(nil)
	}
	j.lane.releaseExport()
}
