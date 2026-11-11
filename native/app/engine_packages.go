package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/egoist/mygo"
	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineruntime"
	"levnote.local/native/ui/editor"
)

type packageAction string

const (
	packageListAction     packageAction = "list"
	packageImportAction   packageAction = "import"
	packageActivateAction packageAction = "activate"
	packageResetAction    packageAction = "reset"
	packageRepairAction   packageAction = "repair"
)

type packageList struct {
	checked  bool
	receipts []engineruntime.PackageReceipt
	err      error
}

// Exactly one bounded package result slot; no worker reads App/Tab/View state.
// done closes without UI dispatch, including cancellation and cleanup failure.
type packageJob struct {
	manager         *engineruntime.PackageManager
	engine          enginecatalog.EngineID
	action          packageAction
	generation      uint64
	transition      bool
	cancel          context.CancelFunc
	done            chan struct{}
	oldLane         *engineLane
	oldRunner       exportRunner
	oldExport       *exportJob
	oldPreview      *previewJob
	mu              sync.Mutex
	ready           bool
	receipts        []engineruntime.PackageReceipt
	receipt         engineruntime.PackageReceipt
	err, cleanupErr error
}

func (a *App) initializeEnginePackages() {
	a.packageEngine = enginecatalog.MarkdownHTML
	a.packageLists = make(map[enginecatalog.EngineID]packageList)
	executable, err := os.Executable()
	if err == nil {
		executable, err = filepath.Abs(executable)
	}
	if err == nil {
		a.packageManager, err = engineruntime.NewPackageManager(filepath.Dir(executable), a.configDir)
	}
	a.packageInitErr = err
	if err != nil {
		a.report(fmt.Errorf("初始化本地引擎包：%w", err))
		return
	}
	a.packageStatuses = a.packageManager.Snapshot()
	if err := a.packageManager.LoadError(); err != nil {
		a.report(fmt.Errorf("引擎包选择状态不可用：%w", err))
	}
}

func packageTransition(action packageAction) bool {
	return action == packageActivateAction || action == packageResetAction || action == packageRepairAction
}

func (a *App) chooseEngineImport() {
	if a.shuttingDown.Load() || a.packageJob != nil || a.packageManager == nil {
		return
	}
	manager, engine, generation := a.packageManager, a.packageEngine, a.packageGeneration
	defer a.deferExportDelivery()()
	paths, err := a.chooseOpenPaths(mygo.OpenDialogOptions{Parent: a.Window, Title: "导入本地引擎包", Message: "选择可信发布文件夹。仅复制所选引擎及许可证；导入不会启用它。", Directory: true})
	if err != nil {
		a.report(err)
		return
	}
	if len(paths) == 0 {
		return
	}
	if a.shuttingDown.Load() || a.packageManager != manager || a.packageEngine != engine || a.packageGeneration != generation || a.packageJob != nil {
		a.report(errors.New("选择目录期间引擎或操作已变化，未导入"))
		return
	}
	a.startPackageWork(packageImportAction, engine, paths[0])
}

func (a *App) startPackageWork(action packageAction, engine enginecatalog.EngineID, value string) {
	if a.shuttingDown.Load() || a.packageJob != nil {
		return
	}
	if a.packageInitErr != nil || a.packageManager == nil {
		a.report(errors.New("本地引擎包仓库不可用"))
		return
	}
	if _, ok := enginecatalog.Lookup(engine); !ok {
		a.report(errors.New("未知的固定引擎"))
		return
	}
	if action != packageListAction && action != packageImportAction && !packageTransition(action) {
		a.report(errors.New("未知的引擎包操作"))
		return
	}
	// A transition must join the old runner before persisting a new selection.
	// Once that cleanup has poisoned the service, starting another transition
	// would only enqueue a guaranteed failure; list/import remain independent
	// store operations and stay available for diagnosis.
	if packageTransition(action) && a.enginePoison != nil {
		a.report(fmt.Errorf("引擎服务清理失败，无法切换引擎；请重启应用后重试：%w", a.enginePoison))
		return
	}
	a.packageGeneration++
	ctx, cancel := context.WithCancel(context.Background())
	j := &packageJob{manager: a.packageManager, engine: engine, action: action, generation: a.packageGeneration, transition: packageTransition(action), cancel: cancel, done: make(chan struct{})}
	a.packageJob = j
	a.packageOutcome = "正在" + packageActionLabel(action) + "…"
	if j.transition {
		a.engineSwitching = true
		a.engineGeneration++
		j.oldLane, j.oldRunner, j.oldExport, j.oldPreview = a.engineLane, a.exporter, a.exportJob, a.previewJob
		if j.oldExport != nil {
			j.oldExport.invalidate("引擎正在切换，已取消导出")
			a.exportGeneration++
			a.exportJob = nil
		}
		if j.oldPreview != nil {
			j.oldPreview.invalidate()
			j.oldPreview.lane.cancelTask(j.oldPreview.task)
			a.previewJob = nil
		}
		// New requests are blocked until this exact old owner is joined. The
		// background closure sees only captured handles, never dynamic app fields.
		a.engineLane, a.exporter = nil, nil
	}
	var notify func()
	if a.Window != nil {
		notify = a.Window.Invalidate
	}
	poison := a.enginePoison
	go j.run(ctx, value, poison, notify)
}

func (j *packageJob) run(ctx context.Context, value string, poison error, notify func()) {
	var err, cleanupErr error
	var receipts []engineruntime.PackageReceipt
	var receipt engineruntime.PackageReceipt
	if j.transition {
		cleanupErr = poison
		if j.oldLane != nil {
			cleanupErr = errors.Join(cleanupErr, j.oldLane.close())
		} else if j.oldRunner != nil {
			cleanupErr = errors.Join(cleanupErr, j.oldRunner.Close())
		}
		if j.oldExport != nil {
			<-j.oldExport.done
			j.oldExport.clear()
		}
		if j.oldPreview != nil {
			<-j.oldPreview.done
		}
	}
	if cleanupErr != nil {
		err = fmt.Errorf("旧引擎清理失败，选择未切换：%w", cleanupErr)
	} else if ctx.Err() != nil {
		err = ctx.Err()
	} else {
		switch j.action {
		case packageListAction:
			receipts, err = j.manager.Installed(ctx, j.engine)
		case packageImportAction:
			receipt, err = j.manager.ImportFromReleaseDir(ctx, value, j.engine)
		case packageActivateAction:
			err = j.manager.Activate(ctx, j.engine, value)
		case packageResetAction:
			err = j.manager.Reset(ctx, j.engine)
		case packageRepairAction:
			err = j.manager.RepairResetAll(ctx)
		}
		if cleanupFailure(err) {
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	j.cancel()
	j.mu.Lock()
	j.receipts, j.receipt, j.err, j.cleanupErr, j.ready = receipts, receipt, err, cleanupErr, true
	close(j.done)
	j.mu.Unlock()
	if notify != nil {
		notify()
	}
}

func (a *App) deliverPackageWork() {
	if a.shuttingDown.Load() || a.modalDepth > 0 {
		return
	}
	j := a.packageJob
	if j == nil {
		return
	}
	j.mu.Lock()
	if !j.ready {
		j.mu.Unlock()
		return
	}
	receipts, receipt, err, cleanupErr := j.receipts, j.receipt, j.err, j.cleanupErr
	j.receipts, j.receipt = nil, engineruntime.PackageReceipt{}
	j.mu.Unlock()
	if cleanupErr != nil {
		a.enginePoison = errors.Join(a.enginePoison, cleanupErr)
		a.report(fmt.Errorf("引擎服务清理失败，已停止新任务：%w", cleanupErr))
	}
	// Cleanup poison is retained even when UI ownership changed. Other data is
	// applied only to the manager/generation that owns this result slot.
	valid := a.packageManager == j.manager && a.packageGeneration == j.generation
	a.packageJob = nil
	if j.transition {
		a.engineSwitching = false
	}
	if !valid {
		return
	}
	a.packageStatuses = j.manager.Snapshot()
	if j.action == packageListAction {
		a.packageLists[j.engine] = packageList{checked: true, receipts: receipts, err: err}
	} else if j.action == packageImportAction && receipt.Digest != "" {
		list := a.packageLists[j.engine]
		found := false
		for i, old := range list.receipts {
			if old.Digest == receipt.Digest {
				list.receipts[i] = receipt
				found = true
			}
		}
		if !found {
			list.receipts = append(list.receipts, receipt)
		}
		a.packageLists[j.engine] = list
	}
	if err != nil {
		a.packageOutcome = packageActionLabel(j.action) + "未完整完成：" + err.Error() + "；当前选择以面板显示为准"
		a.report(errors.New(a.packageOutcome))
	} else {
		a.packageOutcome = packageActionLabel(j.action) + "完成"
		if j.action == packageImportAction {
			a.packageOutcome = "已导入 " + receipt.Version + "，尚未启用"
		}
	}
	if j.transition {
		// No configureTab/Flush/Release: writable drafts and undo are untouched.
		for _, tab := range a.Tabs {
			if tab.Doc != nil && tab.Doc.ReadOnly && tab.Adapter != nil && tab.Adapter.SceneEngine != nil {
				if refresher, ok := tab.View.(editor.SceneRefresher); ok {
					refresher.RefreshScene()
				}
			}
		}
	}
}

// UI invalidation occurs before joining; no worker needs a UI callback to close
// its done barrier. Called only after CanClose's save/persistence decisions, or
// after Shutdown has marked the app closed.
func (a *App) joinPackageWork() error {
	j := a.packageJob
	if j == nil {
		return a.enginePoison
	}
	j.cancel()
	<-j.done
	j.mu.Lock()
	cleanupErr := j.cleanupErr
	j.receipts, j.receipt = nil, engineruntime.PackageReceipt{}
	j.mu.Unlock()
	if cleanupErr != nil {
		a.enginePoison = errors.Join(a.enginePoison, cleanupErr)
	}
	if a.packageManager == j.manager {
		a.packageStatuses = j.manager.Snapshot()
	}
	a.packageJob = nil
	a.engineSwitching = false
	return a.enginePoison
}

func packageActionLabel(action packageAction) string {
	switch action {
	case packageListAction:
		return "检查已安装包"
	case packageImportAction:
		return "导入引擎包"
	case packageActivateAction:
		return "启用引擎包"
	case packageResetAction:
		return "恢复随包引擎"
	case packageRepairAction:
		return "重置全部引擎选择"
	}
	return "引擎包操作"
}

func packageEngineLabel(engine enginecatalog.EngineID) string {
	switch engine {
	case enginecatalog.MarkdownHTML:
		return "Markdown HTML 导出"
	case enginecatalog.MindmapSVG:
		return "思维导图 SVG 导出"
	case enginecatalog.MindmapPreview:
		return "只读思维导图预览"
	case enginecatalog.DrawioSVG:
		return "图纸 SVG 导出"
	}
	return string(engine)
}

func shortPackageDigest(digest string) string {
	if len(digest) > 12 {
		return digest[:12]
	}
	return digest
}
