package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"levnote.local/native/core/document"
)

func (a *App) startBinaryExplicit(j *binaryJob, request binaryRequest) error {
	if old := a.binaryJob; old != nil && old.abandoned.Load() {
		a.discardAbandonedBinary(old)
	}
	if a.shuttingDown.Load() || a.binaryJob != nil {
		return errors.New("已有图片读取或复制正在完成，请稍候")
	}
	j.done = make(chan struct{})
	j.task = newBinaryTask(request, a.binaryNotify())
	if err := a.ensureBinaryLane().submit(j.task, true); err != nil {
		return err
	}
	a.binaryJob = j
	return nil
}

func (a *App) finishHeadlessBinary(j *binaryJob) {
	if a.Window == nil && a.uiDispatch == nil {
		<-j.task.done
		// Public Save wraps a modal scope; its deferred delivery owns adoption.
		if a.modalDepth == 0 {
			a.deliverBinary()
		}
	}
}

func (a *App) openBinaryRequest(path string, kind document.BinaryKind, notice string, reopen *reopenReceipt) bool {
	j := &binaryJob{op: "open", name: filepath.Base(path), notice: notice, reopen: reopen}
	var notify func()
	var selected bool
	if !a.updateWait(func() {
		notify = a.binaryNotify()
		if j.err = a.checkReopen(reopen); j.err != nil {
			return
		}
		if selected = a.selectOpenDuplicate(canonical(path), openRequest{reopen: reopen}); selected {
			return
		}
		if j.err = a.admitNewTab(); j.err != nil {
			j.err = fmt.Errorf("打开 %s：%w", filepath.Base(path), j.err)
			return
		}
		j.err = a.startBinaryExplicit(j, binaryRequest{op: "open", path: path, kind: kind})
	}) {
		return false
	}
	if selected {
		return true
	}
	if j.err != nil {
		a.reportAsync(j.err)
		return true
	}
	select {
	case <-j.task.done:
	case <-a.shutdownDone:
		return false
	}
	// The queued closure carries no image/source data. Shutdown/cancellation can
	// clear the owned slot before this callback is eventually dispatched.
	if !a.updateWait(a.deliverBinary) {
		j.abandoned.Store(true)
		j.task.abandon()
		if notify != nil {
			notify()
		}
		return false
	}
	select {
	case <-j.done:
		return true
	case <-a.shutdownDone:
		return false
	}
}

// A timed-out dispatch leaves only a small UI record until the next frame or
// command. Reclaim its reservation on UI; abandoned payload cannot open later.
func (a *App) discardAbandonedBinary(j *binaryJob) {
	if a.binaryJob != j {
		return
	}
	a.binaryJob = nil
	j.task.abandon()
	j.tab, j.owner, j.view, j.adapter, j.complete = nil, nil, nil, nil, nil
	close(j.done)
	a.binaryLane.release(j.task, true)
	a.report(errors.New("图片读取结果未能交回界面；已取消打开并释放任务，请重试"))
}

// Refuse the first close without joining IO on UI. The outcome is displayed
// while the window exists; the user may close again after seeing it.
func (a *App) prepareBinaryWindowClose() bool {
	j := a.binaryJob
	if j == nil {
		return true
	}
	if j.abandoned.Load() {
		a.discardAbandonedBinary(j)
		return false
	}
	if out, ready := j.task.take(); ready {
		a.completeBinaryExplicit(j, out)
		a.Message += "；请确认结果后再次关闭窗口"
		return false
	}
	j.task.cancel()
	a.Message = "正在取消图片任务并确认副本是否已写入；完成后请再次关闭窗口"
	return false
}

func (a *App) saveBinary(t *Tab, as bool) bool {
	if !as {
		a.Message = "图片为只读；请使用另存副本保存原始字节"
		return false
	}
	if t.Binary == nil {
		a.report(errors.New("图片超过 16 MiB 上限，此字节视图不能另存副本"))
		return false
	}
	if a.modalDepth > 1 {
		a.Message = "请先结束当前对话框再另存图片副本"
		return false
	}
	if t != a.Current() || !a.tabIsOpen(t) || a.binaryJob != nil {
		a.report(errors.New("图片标签已变化或任务正在完成"))
		return false
	}
	owner, token, generation, mode, view, adapter := t.Binary, t.Binary.Token(), t.BinaryGeneration, t.Mode, t.View, t.Adapter
	snapshot := a.heldBinary(t)
	path, err := a.chooseSavePath(t)
	if err != nil {
		a.report(err)
		return false
	}
	if path == "" {
		return false
	}
	if a.Current() != t || !a.tabIsOpen(t) || t.Binary != owner || !owner.SameBaseline(token) ||
		t.BinaryGeneration != generation || t.Mode != mode || t.View != view || t.Adapter != adapter || a.shuttingDown.Load() {
		a.report(errors.New("选择副本位置期间图片已变化；未复制"))
		return false
	}
	j := &binaryJob{op: "copy", tab: t, owner: owner, token: token, generation: generation, mode: mode}
	if err := a.startBinaryExplicit(j, binaryRequest{op: "copy", path: path, token: token, snapshot: snapshot}); err != nil {
		a.report(err)
		return false
	}
	t.Saving = true
	a.Message = "正在另存原始图片副本…"
	a.finishHeadlessBinary(j)
	return true
}

func (a *App) reloadBinary(t *Tab) {
	if t.Binary == nil || t != a.Current() {
		return
	}
	j := &binaryJob{op: "reload", tab: t, owner: t.Binary, token: t.Binary.Token(), generation: t.BinaryGeneration, mode: t.Mode}
	if err := a.startBinaryExplicit(j, binaryRequest{op: "reload", path: t.path(), kind: t.BinaryKind}); err != nil {
		a.report(err)
		return
	}
	a.Message = "正在重新载入图片…"
	a.finishHeadlessBinary(j)
}

func (a *App) completeBinaryExplicit(j *binaryJob, out binaryOutput) {
	a.binaryJob = nil
	defer func() {
		if j.tab != nil {
			j.tab.Saving = false
		}
		j.tab, j.owner, j.view, j.adapter, j.complete = nil, nil, nil, nil, nil
		j.err = out.err
		close(j.done)
		a.binaryLane.release(j.task, true)
	}()
	if j.op == "copy" {
		a.completeBinaryCopy(j, out)
		return
	}
	if out.err != nil {
		a.report(fmt.Errorf("图片读取失败：%w", out.err))
		return
	}
	if j.op == "reload" {
		if !a.validBinaryJob(j) {
			a.Message = "图片标签已变化，已丢弃重新载入结果"
			return
		}
		t := j.tab
		a.disposePresentation(t)
		t.Binary, t.BinaryKind, t.Conflict, t.Warning = out.file, out.file.Kind(), false, ""
		a.configureTab(t)
		if j.mode == "source" {
			t.Mode = "source"
		}
		a.attachBinarySnapshot(t, out.snapshot)
		a.invalidateDiskChecks()
		a.markPersistPending()
		a.Message = "已重新载入 " + t.Name
		return
	}
	if err := a.checkReopen(j.reopen); err != nil {
		a.report(err)
		return
	}
	path := ""
	if out.file != nil {
		path = out.file.Path()
	} else if out.inspection != nil {
		path = out.inspection.Path
	}
	for index, existing := range a.Tabs {
		if sameNativeFilePath(canonical(existing.path()), canonical(path)) {
			if j.reopen != nil {
				a.selectOpenDuplicate(canonical(path), openRequest{reopen: j.reopen})
				return
			}
			if a.Select(index) {
				a.Message = "文件已在标签中打开：" + existing.Name
			}
			return
		}
	}
	if err := a.admitNewTab(); err != nil {
		out.err = fmt.Errorf("打开 %s：%w", j.name, err)
		a.report(out.err)
		return
	}
	check := a.reopenAdoptionCheck(j.reopen)
	if old := a.Current(); old != nil {
		if !a.flushTab(old, "打开图片") {
			return
		}
		if check != nil {
			if err := check(); err != nil {
				a.report(err)
				return
			}
		}
		if err := a.admitNewTab(); err != nil {
			out.err = fmt.Errorf("打开 %s：%w", j.name, err)
			a.report(out.err)
			return
		}
		a.invalidateTabExport(old, "打开图片，已取消导出")
		a.releasePresentation(old)
	}
	t := a.appendBinary(out, j.name, j.notice)
	a.Active = len(a.Tabs) - 1
	a.closeFind()
	a.setOutlineVisible(false)
	a.closeGoToLine()
	a.attachBinarySnapshot(t, out.snapshot)
	a.startWatch(t.path(), false)
	a.markPersistPending()
	a.acknowledgeReopen(j.reopen)
	a.Message = "已打开 " + t.Name
	if j.reopen != nil {
		a.Message = "已按当前磁盘内容重新打开 " + t.Name
	}
}

func (a *App) completeBinaryCopy(j *binaryJob, out binaryOutput) {
	if out.err != nil {
		if out.copy.Committed {
			a.report(fmt.Errorf("副本已写入 %s，但未完成验证，当前图片未改动：%w", out.receipt.Path(), out.err))
		} else {
			a.report(fmt.Errorf("图片副本未写入：%w", out.err))
		}
		return
	}
	if !a.validBinaryJob(j) || j.tab.Mode != j.mode {
		if out.copy.Committed {
			a.report(fmt.Errorf("副本已完成：%s；原标签已关闭或变化，未更改当前标签", out.receipt.Path()))
		} else {
			a.Message = "原路径验证完成；原标签已变化，未更改当前标签"
		}
		return
	}
	t := j.tab
	oldPath := t.path()
	result, rebound, err := t.Binary.AdoptCopy(out.receipt, out.snapshot)
	if err != nil {
		a.report(fmt.Errorf("副本结果 committed=%t verified=%t：%s；未采纳：%w", result.Committed, result.Verified, out.receipt.Path(), err))
		return
	}
	if result.Adopted {
		a.disposePresentation(t)
		t.Name, t.Conflict = filepath.Base(t.path()), false
		a.configureTab(t)
		if j.mode == "source" {
			t.Mode = "source"
		}
		if cancel := a.watches[oldPath]; cancel != nil {
			cancel()
			delete(a.watches, oldPath)
		}
		a.startWatch(t.path(), false)
	}
	a.attachBinarySnapshot(t, rebound)
	a.invalidateDiskChecks()
	a.Message = "已另存原始图片副本：" + t.path()
	if result.NoRewrite {
		a.Message = "原路径字节已验证，没有重写文件"
	}
	a.Persist()
}

func (a *App) cancelBinaryForClose(t *Tab) {
	if j := a.binaryJob; j != nil && j.tab == t {
		// Copy result ownership survives the tab. A committed result must still
		// be reported after cancellation; the slot retains no tab or callback.
		j.tab, j.owner, j.view, j.adapter, j.complete = nil, nil, nil, nil, nil
		j.task.cancel()
	}
}

func (a *App) shutdownBinary() {
	if a.binaryLane == nil {
		return
	}
	a.cancelBinaryPreview(nil)
	a.binaryLane.close()
	if j := a.binaryJob; j != nil {
		if out, ready := j.task.take(); ready {
			if j.op == "copy" {
				if out.copy.Committed {
					a.report(fmt.Errorf("关闭时副本已写入 %s（验证完成：%t）；未切换文档", out.receipt.Path(), out.copy.Verified))
				} else if out.err != nil && !errors.Is(out.err, context.Canceled) {
					a.report(out.err)
				}
			}
		}
		if j.tab != nil {
			j.tab.Saving = false
		}
		j.tab, j.owner, j.complete, j.view, j.adapter = nil, nil, nil, nil, nil
		close(j.done)
		a.binaryJob = nil
	}
	for _, t := range a.Tabs {
		if t.isBinary() {
			a.disposePresentation(t)
		}
	}
	a.binarySnapshot, a.binarySnapshotTab = nil, nil
}
