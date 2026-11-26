package app

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineruntime"
	"levnote.local/native/core/plugins"
	"levnote.local/native/core/search"
	"levnote.local/native/core/session"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
	"levnote.local/native/modules"
	"levnote.local/native/ui/editor"
)

type Tab struct {
	ID     int
	Name   string
	Doc    *document.Document
	Binary *document.BinaryFile
	// BinaryPath/Size are metadata for oversized byte inspection only; a
	// bounded BinaryFile, when present, is always the file authority.
	BinaryPath              string
	BinarySize              int64
	BinaryKind              document.BinaryKind
	BinaryGeneration        uint64
	BinaryNotice            string
	ImageFormat             string
	ImageWidth, ImageHeight int
	Editor                  *editor.Editor
	Mode                    string
	Warning                 string
	FormatWarning           string
	SyntaxNotice            string
	// SourceCapabilityBadge/Detail are tiny strings captured when the tab is
	// configured. Frame rendering must not ask the plugin manager to clone its
	// syntax metadata on every source-editor frame.
	SourceCapabilityBadge  string
	SourceCapabilityDetail string
	ViewWarnings           []string
	Saving                 bool
	Adapter                *editor.Adapter
	// OutlineFactory is the source-only outline composition seam. Built-in
	// adapters continue to expose their factory through Adapter.Outline; an
	// installed schema2 source format keeps Adapter nil and uses this factory.
	OutlineFactory           func(editor.Host) editor.OutlineSource
	OutlineFactoryGeneration uint64
	MatchedPlugin            plugins.Manifest
	sourceLanguageRoute      uint64
	sourceLanguageProfile    code.StructureProfile
	OriginalPath             string
	View                     editor.View
	Conflict                 bool
	ByteInspector            bool
	CanReinterpret           bool
	// The disclosure latch belongs to this tab's actual text buffer, not its ID.
	historyNoticeBuffer *text.Buffer
	seenTruncated       bool
	inputBinding        *sourceInputBinding
	inputEpoch          uint64
	formatHostEpoch     uint64
	formatHostActive    bool
}
type Entry struct {
	Path, Name string
	Directory  bool
	Depth      int
}
type App struct {
	Window     *mygo.Window
	uiDispatch func(func())
	saveDialog func(*Tab) (string, error)
	// Optional save-outcome seam; production always calls Document.Save/SaveAs.
	saveDocument                                    func(*document.Document, string) error
	closeDialog                                     func(*Tab) (closeDecision, error)
	reloadDialog                                    func(mygo.MessageOptions) (mygo.MessageResult, error)
	openDialog                                      func(mygo.OpenDialogOptions) ([]string, error)
	exportDialog                                    func(mygo.SaveDialogOptions) (string, error)
	exporter                                        exportRunner
	engineLane                                      *engineLane
	engineGeneration                                uint64
	engineSwitching                                 bool
	enginePoison                                    error
	packageManager                                  *engineruntime.PackageManager
	packageInitErr                                  error
	packageStatuses                                 []engineruntime.PackageStatus
	packageLists                                    map[enginecatalog.EngineID]packageList
	packageJob                                      *packageJob
	packageGeneration                               uint64
	packageEngine                                   enginecatalog.EngineID
	packageOutcome                                  string
	showEnginePackages                              bool
	managedFactory                                  func(*engineruntime.PackageManager) (exportRunner, error)
	previewJob                                      *previewJob
	engineCleanupReported                           bool
	exportJob                                       *exportJob
	exportGeneration                                uint64
	draftRecoveryGeneration                         uint64
	prepareSessionState                             func(string, session.Snapshot) (preparedSessionState, error)
	exportStatuses                                  map[string]exportEngineStatus
	modalDepth                                      int
	openDocuments                                   openDocumentPicker
	commandPalette                                  commandPaletteState
	sourceLanguage                                  sourceLanguageState
	sourceLanguageHooks                             *sourceLanguageHooks
	sourceAutoHooks                                 *sourceAutoHooks
	Tabs                                            []*Tab
	Active                                          int
	revealedTabID                                   int
	Root                                            string
	Entries                                         []Entry
	Expanded                                        map[string]bool
	Message                                         string
	Issues                                          []string
	IssuesDropped                                   int
	Dark                                            bool
	FontSize                                        float32
	SoftWrap, LineNumbers                           bool
	SidebarWidth                                    float32
	ShowFind, ShowSettings, ShowIssues, ShowPlugins bool
	Query, Replacement                              string
	FindQueryComposing, FindReplacementComposing    bool
	findGeneration                                  uint64
	findPanel, findQueryHandle                      ui.Handle
	findReplacementHandle                           ui.Handle
	findFocus                                       findFocusTarget
	findFocusGeneration                             uint64
	findFocusIntentEpoch                            uint64 // survives one-shot focus resolution and clearing
	findFocusTab                                    *Tab
	findFocusDoc                                    *document.Document
	findFocusEditor                                 *editor.Editor
	findFocusBuffer                                 *text.Buffer
	ShowGoToLine                                    bool
	GoToLineQuery, GoToLineError                    string
	GoToLineComposing                               bool
	goToLineGeneration                              uint64
	goToLineTab                                     *Tab
	goToLineDoc                                     *document.Document
	goToLineEditor                                  *editor.Editor
	goToLineBuffer                                  *text.Buffer
	CaseSensitive, Regex, WholeWord                 bool
	FindIndex                                       int
	findNavigation                                  findNavigationCache
	nextID                                          int
	configDir                                       string
	statusTitle                                     string
	lastEdit                                        time.Time
	persistPending                                  bool
	lastPersistError                                string
	autosave                                        autosaveState
	autosaveHooks                                   *autosaveHooks
	adapters                                        []editor.Adapter
	pluginManager                                   *plugins.Manager
	watches                                         map[string]context.CancelFunc
	searchCancel                                    context.CancelFunc
	searchID                                        int
	searchRoot                                      string
	SearchResults                                   []search.FileMatch
	SearchIssues                                    []search.FileIssue
	SearchIssuesDropped                             int
	SearchSummary                                   string
	Searching, ShowWorkspaceSearch                  bool
	TreeTruncated                                   bool
	TreeIssues                                      []string
	TreeList                                        ui.ListState
	SearchList                                      ui.ListState
	OutlineList                                     ui.ListState
	treeListLifetime                                appListLifetime
	searchListLifetime                              appListLifetime
	outlineListLifetime                             appListLifetime
	TreeSelected                                    int
	SearchSelected                                  int
	OutlineSelected                                 int
	ShowOutline                                     bool
	OutlineSource                                   editor.OutlineSource
	OutlineTabID                                    int
	OutlineBuffer                                   *text.Buffer
	OutlineItems                                    []editor.OutlineItem
	OutlineSnapshotVersion                          uint64
	OutlineCheckedVersion                           uint64
	outlineGeneration                               uint64
	outlineFactorySequence                          uint64
	OutlineFactoryGeneration                        uint64
	OutlineTruncated, OutlineLoaded                 bool
	OutlineError                                    string
	treeRowBuilt                                    func(int)
	searchRowBuilt                                  func(int)
	outlineRowBuilt                                 func(int)
	treeWorker                                      *coalescedWorker[treeScanTask]
	diskWorker                                      *coalescedWorker[diskCheckTask]
	treeRoot                                        string
	treeGeneration                                  uint64
	diskGeneration                                  uint64
	updateWaitTimeout                               time.Duration
	shuttingDown                                    atomic.Bool
	shutdownDone                                    chan struct{}
	shutdownJoined                                  chan struct{}
	openMu                                          sync.Mutex
	closedFiles                                     []closedFile
	nextClosedID                                    uint64
	reopening                                       *reopenReceipt
	binaryLane                                      *binaryLane
	binaryJob                                       *binaryJob
	binaryPreview                                   *binaryJob
	binarySnapshot                                  *document.BinarySnapshot
	binarySnapshotTab                               *Tab
	binaryDecode                                    func(context.Context, io.ReaderAt, int64) (editor.ImageResult, error)
	restoring                                       bool
	flushRefusalTokens                              [7]*flushRefusalToken
}

const (
	flushRefusalAdd           = 0
	flushRefusalBinaryOpen    = 1
	flushRefusalOpenDocuments = 2
	flushRefusalOutline       = 3
	flushRefusalWorkspace     = 4
	flushRefusalExport        = 5
	flushRefusalReplace       = 6
)

type flushRefusalLifecycle struct {
	shuttingDown  bool
	restoring     bool
	configDir     string
	sourceSession uint64
}

type flushRefusalToken struct{ marker byte }
type flushRefusalScope struct {
	token      *flushRefusalToken
	terminated bool
}

type flushLoss uint32

const (
	flushLossLifecycle flushLoss = 1 << iota
	flushLossSession
	flushLossConfig
	flushLossActivity
	flushLossSlot
	flushLossTabIdentity
	flushLossDocIdentity
	flushLossEditorIdentity
	flushLossBuffer
	flushLossMetadata
	flushLossReadOnly
	flushLossRoute
	flushLossFactory
	flushLossAdapter
	flushLossView
	flushLossOpaqueLease
	flushLossEpoch
	flushLossFormatHost
	flushLossBinaryFullToken
	flushLossActivation
	flushLossRestoreState
)

type flushKind uint8

const (
	flushCommitted flushKind = iota
	flushSeamErrorReported
	flushAuthorityLost
)

type flushResult struct {
	kind flushKind
	loss flushLoss
}

func (a *App) beginFlushRefusal(family int) *flushRefusalScope {
	token := &flushRefusalToken{marker: 1}
	a.flushRefusalTokens[family] = token
	return &flushRefusalScope{token: token}
}

func (a *App) currentFlushRefusal(family int, scope *flushRefusalScope) bool {
	return scope != nil && !scope.terminated && scope.token != nil && a.flushRefusalTokens[family] == scope.token
}

func (a *App) claimFlushRefusal(family int, scope *flushRefusalScope) bool {
	if !a.currentFlushRefusal(family, scope) {
		return false
	}
	scope.terminated = true
	return true
}

func (a *App) captureFlushRefusalLifecycle() flushRefusalLifecycle {
	return flushRefusalLifecycle{
		shuttingDown:  a.shuttingDown.Load(),
		restoring:     a.restoring,
		configDir:     a.configDir,
		sourceSession: a.sourceLanguage.session,
	}
}

func (a *App) currentFlushRefusalLifecycle(family int, scope *flushRefusalScope, captured flushRefusalLifecycle) bool {
	if captured.shuttingDown || !a.currentFlushRefusal(family, scope) || a.shuttingDown.Load() ||
		a.restoring != captured.restoring || a.configDir != captured.configDir ||
		a.sourceLanguage.session != captured.sourceSession {
		return false
	}
	return true
}

type reopenObservation struct {
	receipt *reopenReceipt
	entryID uint64
	origin  *Tab
}

func newReopenObservation(receipt *reopenReceipt, origin *Tab) *reopenObservation {
	if receipt == nil {
		return nil
	}
	return &reopenObservation{receipt: receipt, entryID: receipt.entryID, origin: origin}
}

func (a *App) observeReopenRefusal(observation *reopenObservation) error {
	if observation == nil {
		return nil
	}
	receipt := observation.receipt
	if receipt == nil {
		return nil
	}
	if a.reopening != receipt || receipt.outcome.Load() != reopenPending || a.closedFileIndex(observation.entryID) < 0 {
		return errors.New("最近关闭记录已变化；本次重新打开已取消，请重新选择")
	}
	if err := a.reopenBarrier(); err != nil {
		return err
	}
	if a.Current() != observation.origin {
		return errors.New("同步输入期间当前标签已变化；未重新打开文件")
	}
	return nil
}

type workspaceSearchObservation struct {
	index  int
	id     int
	root   string
	result search.FileMatch
	origin *Tab
}

type addAttachmentObservation struct {
	document  *document.Document
	inspector bool
	buffer    *text.Buffer
}

type pluginRouteObservation struct {
	id, name, adapterID string
	route               plugins.NativeRoute
	origin              *Tab
	manager             *plugins.Manager
	owner               *document.Document
	buffer              *text.Buffer
	input               *editor.Editor
	view                editor.View
	factoryGeneration   uint64
}

type addRequestObservation struct {
	document   *document.Document
	inspector  bool
	workspace  *workspaceSearchObservation
	attachment *addAttachmentObservation
	reopen     *reopenObservation
	route      *pluginRouteObservation
}

func (a *App) observeAddRefusal(observation *addRequestObservation) error {
	if err := a.admitNewTab(); err != nil {
		return err
	}
	if observation == nil {
		return errors.New("打开新标签前原文档或格式已变化；未收纳，请重新打开")
	}
	if observation.route != nil {
		if err := a.observePluginRouteRefusal(observation.route); err != nil {
			return err
		}
	}
	if observation.workspace != nil {
		if err := a.observeWorkspaceAttachmentRefusal(observation.workspace, observation.attachment); err != nil {
			return err
		}
	}
	if err := a.observeReopenRefusal(observation.reopen); err != nil {
		return err
	}
	return errors.New("打开新标签前原文档或格式已变化；未收纳，请重新打开")
}

func (a *App) observeWorkspaceAttachmentRefusal(workspace *workspaceSearchObservation, attachment *addAttachmentObservation) error {
	if workspace == nil {
		return nil
	}
	if a.shuttingDown.Load() || !a.ShowWorkspaceSearch || a.ShowPlugins ||
		a.searchID != workspace.id || a.searchRoot != workspace.root || a.Root != workspace.root ||
		workspace.index < 0 || workspace.index >= len(a.SearchResults) ||
		a.SearchResults[workspace.index] != workspace.result {
		return fmt.Errorf("工作区搜索结果已变化；未跳转")
	}
	if a.Current() != workspace.origin {
		return fmt.Errorf("工作区搜索目标已变化；未跳转")
	}
	if attachment == nil || attachment.document == nil || attachment.inspector || attachment.buffer == nil ||
		attachment.document.Buffer != attachment.buffer ||
		!workspace.result.MatchesBufferWithoutSnapshot(attachment.buffer) {
		return fmt.Errorf("工作区搜索结果对应的内容已变化；请重新搜索，未跳转")
	}
	return nil
}

func (a *App) observePluginRouteRefusal(route *pluginRouteObservation) error {
	if route == nil {
		return nil
	}
	if a.Current() != route.origin || a.pluginManager != route.manager || (route.origin != nil &&
		(route.origin.Doc != route.owner || route.owner != nil && route.owner.Buffer != route.buffer ||
			route.origin.Editor != route.input || route.origin.View != route.view ||
			route.origin.OutlineFactoryGeneration != route.factoryGeneration)) {
		return errors.New("当前文档或格式配置已变化，请重新新建")
	}
	if a.pluginManager == nil {
		return errors.New("格式配置管理器不可用")
	}
	manifest, err := a.pluginManager.Manifest(route.id)
	if err != nil || !manifest.Declarative || !manifest.Enabled || !manifest.Active || len(manifest.Extensions) == 0 {
		return errors.New("格式配置未启用或无法载入，请先检查插件列表")
	}
	syntax, ok := a.pluginManager.Syntax(route.id)
	if !ok || syntax.Schema != 3 {
		return errors.New("此配置没有内置格式模板")
	}
	adapter, ok := a.nativeRouteAdapter(syntax.NativeRoute)
	if !ok || adapter.Template == nil {
		return errors.New("配置指定的原生格式模板不可用")
	}
	name := "未命名" + manifest.Extensions[0]
	match, err := a.pluginManager.Match(name)
	if err != nil || match.ID != route.id {
		return fmt.Errorf("首个扩展名 %s 已被其它格式占用；请调整扩展名或启用状态后重试", manifest.Extensions[0])
	}
	if name != route.name || syntax.NativeRoute != route.route || adapter.ID != route.adapterID {
		return errors.New("格式配置已变化，请重新新建")
	}
	return nil
}

type searchRowKey struct {
	Path       string
	Start, End int
}

type revealTarget struct {
	Start, End int
}

type openRequest struct {
	path   string
	reveal *revealTarget
	reopen *reopenReceipt
	search *workspaceSearchTarget
}

type closeDecision uint8

const (
	closeCancel closeDecision = iota
	closeSave
	closeDiscard
)

const (
	treeEntryLimit          = 3000
	treeDirectoryEntryLimit = 256
	treeDepthLimit          = 12
	issueHistoryLimit       = 200
	searchIssueLimit        = 200
)

func New(configDir string) *App {
	pluginRoot := filepath.Join(configDir, "plugins")
	manager, err := plugins.NewManager(configDir, pluginRoot)
	a := &App{
		Active: -1, TreeSelected: -1, SearchSelected: -1, OutlineSelected: -1,
		FindIndex: -1, findNavigation: findNavigationCache{lastIndex: -1},
		Dark: true, FontSize: 16, SidebarWidth: 228, LineNumbers: true,
		Expanded: map[string]bool{}, configDir: configDir,
		adapters: modules.Builtins(), pluginManager: manager,
		sourceLanguage:    sourceLanguageState{session: 1, status: sourceLanguageIdle},
		watches:           map[string]context.CancelFunc{},
		updateWaitTimeout: 5 * time.Second,
		shutdownDone:      make(chan struct{}),
		shutdownJoined:    make(chan struct{}),
	}
	a.treeWorker = newCoalescedWorker(func(ctx context.Context, task treeScanTask) {
		result := scanTree(ctx, task.root, task.expanded)
		if ctx.Err() != nil {
			return
		}
		a.update(func() {
			a.applyTreeScan(task, result)
		})
	})
	a.diskWorker = newCoalescedWorker(func(ctx context.Context, task diskCheckTask) {
		result := checkDiskTask(ctx, task)
		if ctx.Err() != nil {
			return
		}
		a.update(func() {
			a.applyDiskCheck(task, result)
		})
	})
	if err := os.MkdirAll(pluginRoot, 0o755); err != nil {
		a.report(fmt.Errorf("创建格式插件目录：%w", err))
	}
	if err != nil {
		a.report(fmt.Errorf("启动格式插件：%w", err))
	} else {
		for _, issue := range manager.LoadErrors() {
			a.report(fmt.Errorf("插件 %s：%s", issue.Path, issue.Message))
		}
	}
	a.initializeEnginePackages()
	return a
}
func (a *App) Current() *Tab {
	if a.Active < 0 || a.Active >= len(a.Tabs) {
		return nil
	}
	return a.Tabs[a.Active]
}
func (a *App) update(fn func()) bool {
	if a.shuttingDown.Load() {
		return false
	}
	guarded := func() {
		if !a.shuttingDown.Load() {
			fn()
		}
	}
	if a.Window != nil {
		a.Window.Update(guarded)
	} else if a.uiDispatch != nil {
		a.uiDispatch(guarded)
	} else {
		guarded()
	}
	return true
}

func (a *App) updateWait(fn func()) bool {
	var mu sync.Mutex
	started := false
	abandoned := false
	done := make(chan struct{})
	if !a.update(func() {
		mu.Lock()
		if abandoned {
			mu.Unlock()
			return
		}
		started = true
		mu.Unlock()
		fn()
		mu.Lock()
		close(done)
		mu.Unlock()
	}) {
		return false
	}
	timeout := a.updateWaitTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	abandonPending := func() bool {
		mu.Lock()
		defer mu.Unlock()
		if started {
			return false
		}
		abandoned = true
		return true
	}
	for {
		select {
		case <-done:
			return true
		case <-a.shutdownDone:
			if abandonPending() {
				return false
			}
			// Once a UI callback starts, its mutation is authoritative. Wait for
			// its bounded completion instead of reporting that it never ran.
			<-done
			return true
		case <-timer.C:
			if abandonPending() {
				return false
			}
			<-done
			return true
		}
	}
}
func (a *App) report(err error) {
	if err != nil {
		a.Message = err.Error()
		a.Issues = append(a.Issues, err.Error())
		if overflow := len(a.Issues) - issueHistoryLimit; overflow > 0 {
			a.Issues = append([]string(nil), a.Issues[overflow:]...)
			a.IssuesDropped += overflow
		}
	}
}

// reportAsync is for disk/queue workers. UI-owned state must only be changed
// by the acknowledged dispatcher; if dispatch cannot be queued, the caller
// still gets a bounded failure instead of mutating App from a worker.
func (a *App) reportAsync(err error) {
	if err == nil {
		return
	}
	a.update(func() { a.report(err) })
}
func (a *App) Select(index int) bool {
	return a.selectChecked(index, nil)
}

func (a *App) selectChecked(index int, beforeSelect func() error) bool {
	if index < 0 || index >= len(a.Tabs) {
		return false
	}
	if index == a.Active {
		if beforeSelect != nil {
			if err := beforeSelect(); err != nil {
				a.report(err)
				return false
			}
		}
		a.Current().Editor.Focus()
		a.closeGoToLine()
		return true
	}
	if old := a.Current(); old != nil {
		if !a.flushTab(old, "切换标签") {
			return false
		}
		if beforeSelect != nil {
			if err := beforeSelect(); err != nil {
				a.report(err)
				return false
			}
		}
		a.invalidateTabExport(old, "切换标签，已取消导出")
		if !a.releasePresentation(old) {
			return false
		}
	} else if beforeSelect != nil {
		if err := beforeSelect(); err != nil {
			a.report(err)
			return false
		}
	}
	a.Active = index
	a.invalidateFindCache()
	a.closeGoToLine()
	a.Current().Editor.Focus()
	if a.Current().isBinary() {
		a.closeFind()
		a.setOutlineVisible(false)
	}
	a.markPersistPending()
	return true
}
func (a *App) selectAdjacentTab(forward bool) bool {
	// Select on the current tab still changes focus and closes GoToLine.
	if len(a.Tabs) < 2 {
		return false
	}
	// MyGo's utility fields have no public commit API. Moving their focus
	// during a shortcut frame can clear preedit before the host releases it.
	if !a.ShowPlugins && (a.ShowFind && a.findCompositionActive() || a.ShowGoToLine && a.GoToLineComposing) {
		a.Message = "输入法正在组合文本；请先完成组合后再切换标签"
		return false
	}
	delta := 1
	if !forward {
		delta = -1
	}
	return a.Select((a.Active + delta + len(a.Tabs)) % len(a.Tabs))
}
func (a *App) add(d *document.Document, name, warning string) *Tab {
	return a.addDocument(d, name, warning, false, false)
}

func (a *App) admitNewTab() error {
	if len(a.Tabs) < session.MaxTabs {
		return nil
	}
	return fmt.Errorf("标签已达到 %d 个上限；请先关闭一个标签，再打开新文件。已有标签仍可直接切换", session.MaxTabs)
}

func (a *App) addDocument(d *document.Document, name, warning string, byteInspector, canReinterpret bool) *Tab {
	return a.addDocumentChecked(d, name, warning, byteInspector, canReinterpret, nil, nil)
}

// beforeAdd is used only by route-aware creation to revalidate after Flush,
// while the originating presentation is still alive on a refusal.
func (a *App) addDocumentChecked(d *document.Document, name, warning string, byteInspector, canReinterpret bool, beforeAdd func() error, observation *addRequestObservation) *Tab {
	scope := a.beginFlushRefusal(flushRefusalAdd)
	lifecycle := a.captureFlushRefusalLifecycle()
	if err := a.admitNewTab(); err != nil {
		if a.currentFlushRefusalLifecycle(flushRefusalAdd, scope, lifecycle) && a.claimFlushRefusal(flushRefusalAdd, scope) {
			a.report(err)
		}
		return nil
	}
	old := a.Current()
	if old != nil {
		result := a.flushTabResult(old, "打开新标签")
		if result.kind == flushSeamErrorReported {
			return nil
		}
		if result.kind == flushAuthorityLost {
			if !a.currentFlushRefusalLifecycle(flushRefusalAdd, scope, lifecycle) {
				return nil
			}
			reason := a.observeAddRefusal(observation)
			if reason == nil {
				reason = errors.New("打开新标签前原文档或格式已变化；未收纳，请重新打开")
			}
			if !a.claimFlushRefusal(flushRefusalAdd, scope) {
				return nil
			}
			a.report(reason)
			return nil
		}
		if beforeAdd != nil {
			if err := beforeAdd(); err != nil {
				if a.currentFlushRefusalLifecycle(flushRefusalAdd, scope, lifecycle) && a.claimFlushRefusal(flushRefusalAdd, scope) {
					a.report(err)
				}
				return nil
			}
		}
	} else if beforeAdd != nil {
		if err := beforeAdd(); err != nil {
			if a.currentFlushRefusalLifecycle(flushRefusalAdd, scope, lifecycle) && a.claimFlushRefusal(flushRefusalAdd, scope) {
				a.report(err)
			}
			return nil
		}
	}
	// Flush can synchronously create another tab through a rich-view callback.
	if err := a.admitNewTab(); err != nil {
		if a.currentFlushRefusalLifecycle(flushRefusalAdd, scope, lifecycle) && a.claimFlushRefusal(flushRefusalAdd, scope) {
			a.report(err)
		}
		return nil
	}
	if old != nil {
		a.invalidateTabExport(old, "打开新标签，已取消导出")
		if !a.releasePresentation(old) {
			return nil
		}
	}
	t := a.appendTab(d, name, warning, byteInspector, canReinterpret)
	a.Active = len(a.Tabs) - 1
	a.closeGoToLine()
	t.Editor.Focus()
	a.startWatch(d.Path, false)
	a.markPersistPending()
	return t
}
func (a *App) appendTab(d *document.Document, name, warning string, byteInspector, canReinterpret bool) *Tab {
	a.invalidateDiskChecks()
	a.nextID++
	ed := editor.New(d.Buffer)
	ed.ReadOnly = d.ReadOnly
	ed.FontSize = a.FontSize
	ed.SoftWrap = a.SoftWrap
	ed.LineNumbers = a.LineNumbers
	if a.Window != nil {
		ed.Copy = func(s string) error { return mygo.Clipboard.Write(transfer.TextData(s)) }
		ed.Paste = func() (string, error) {
			data, err := mygo.Clipboard.ReadFormat(transfer.Text)
			return string(data), err
		}
	}
	t := &Tab{
		ID: a.nextID, Name: name, Doc: d, Editor: ed, Mode: "source",
		Warning: warning, ByteInspector: byteInspector, CanReinterpret: canReinterpret,
	}
	if !a.configureTab(t) {
		return nil
	}
	a.Tabs = append(a.Tabs, t)
	a.bindSourceInput(t)
	return t
}
func (a *App) NewDocument(kind string) {
	if kind == "image" {
		a.Message = "图片阅读器不提供空白图片模板"
		return
	}
	if err := a.admitNewTab(); err != nil {
		a.report(err)
		return
	}
	name, value := "未命名.txt", ""
	switch kind {
	case "markdown", "mindmap", "drawio":
		name = map[string]string{"markdown": "未命名.md", "mindmap": "未命名.mindmap", "drawio": "未命名.drawio"}[kind]
		if adapter, ok := a.adapterByID(kind); ok && adapter.Template != nil {
			value = adapter.Template()
		}
	}
	a.add(document.New(value), name, "")
}

// NewFromPlugin creates from a shipped template under the currently enabled
// local configuration. The first normalized suffix must actually match it.
func (a *App) NewFromPlugin(id string) bool {
	if err := a.admitNewTab(); err != nil {
		a.report(err)
		return false
	}
	name, adapter, route, err := a.pluginDocumentTemplate(id)
	if err != nil {
		a.report(err)
		return false
	}
	origin, manager := a.Current(), a.pluginManager
	var owner *document.Document
	var buffer *text.Buffer
	var input *editor.Editor
	var view editor.View
	var generation uint64
	if origin != nil {
		owner, generation = origin.Doc, origin.OutlineFactoryGeneration
		if owner != nil {
			buffer = owner.Buffer
		}
		input, view = origin.Editor, origin.View
	}
	newDocument := document.New(adapter.Template())
	routeObservation := &pluginRouteObservation{
		id: id, name: name, adapterID: adapter.ID, route: route,
		origin: origin, manager: manager, owner: owner, buffer: buffer,
		input: input, view: view, factoryGeneration: generation,
	}
	tab := a.addDocumentChecked(newDocument, name, "", false, false, func() error {
		if a.Current() != origin || a.pluginManager != manager || (origin != nil &&
			(origin.Doc != owner || owner != nil && owner.Buffer != buffer || origin.Editor != input || origin.View != view || origin.OutlineFactoryGeneration != generation)) {
			return errors.New("当前文档或格式配置已变化，请重新新建")
		}
		freshName, freshAdapter, freshRoute, err := a.pluginDocumentTemplate(id)
		if err != nil {
			return err
		}
		if freshName != name || freshRoute != route || freshAdapter.ID != adapter.ID {
			return errors.New("格式配置已变化，请重新新建")
		}
		return nil
	}, &addRequestObservation{document: newDocument, route: routeObservation})
	if tab == nil {
		return false
	}
	a.ShowPlugins = false
	a.Message = "已使用 " + nativeRouteLabel(route) + " 新建 " + tab.Name
	return true
}

func (a *App) pluginDocumentTemplate(id string) (string, editor.Adapter, plugins.NativeRoute, error) {
	refuse := func(err error) (string, editor.Adapter, plugins.NativeRoute, error) {
		return "", editor.Adapter{}, "", err
	}
	if a.pluginManager == nil {
		return refuse(errors.New("格式配置管理器不可用"))
	}
	manifest, err := a.pluginManager.Manifest(id)
	if err != nil || !manifest.Declarative || !manifest.Enabled || !manifest.Active || len(manifest.Extensions) == 0 {
		return refuse(errors.New("格式配置未启用或无法载入，请先检查插件列表"))
	}
	syntax, ok := a.pluginManager.Syntax(id)
	if !ok || syntax.Schema != 3 {
		return refuse(errors.New("此配置没有内置格式模板"))
	}
	adapter, ok := a.nativeRouteAdapter(syntax.NativeRoute)
	if !ok || adapter.Template == nil {
		return refuse(errors.New("配置指定的原生格式模板不可用"))
	}
	name := "未命名" + manifest.Extensions[0]
	match, err := a.pluginManager.Match(name)
	if err != nil || match.ID != id {
		return refuse(fmt.Errorf("首个扩展名 %s 已被其它格式占用；请调整扩展名或启用状态后重试", manifest.Extensions[0]))
	}
	return name, adapter, syntax.NativeRoute, nil
}

func (a *App) recoverDraft(title, content string) error {
	if err := a.admitNewTab(); err != nil {
		return err
	}
	if len(content) > session.MaxDraftBytes {
		return fmt.Errorf("恢复草稿超过 %d 字节上限", session.MaxDraftBytes)
	}
	if !utf8.ValidString(content) {
		return errors.New("恢复草稿不是有效 UTF-8")
	}
	name := strings.TrimSpace(filepath.Base(title))
	if name == "" || name == "." {
		name = "恢复草稿.txt"
	} else if filepath.Ext(name) == "" {
		name += ".txt"
	}
	if old := a.Current(); old != nil {
		if !a.retireSourceLanguage(old) {
			return errSourceLanguageSuperseded
		}
		a.invalidateTabExport(old, "恢复草稿，已取消导出")
		// Recovery can activate a new tab without the ordinary selection path.
		// A readonly scene must be reactivatable when the user returns to it.
		if old.isBinary() {
			if !a.releasePresentation(old) {
				return errSourceLanguageSuperseded
			}
		}
		if old.Doc != nil && old.Doc.ReadOnly && old.Adapter != nil && old.Adapter.SceneEngine != nil && old.View != nil {
			old.formatHostActive = false
			current := a.sourceLanguageContinuation()
			old.View.Release()
			if !current() {
				return errSourceLanguageSuperseded
			}
		}
	}
	t := a.appendTab(document.New(content), "恢复-"+name,
		"这是从冲突状态恢复的未保存副本；保存时请选择新位置。", false, false)
	a.Active = len(a.Tabs) - 1
	a.closeGoToLine()
	t.Editor.Focus()
	a.noteEdit()
	a.Message = "已把冲突草稿恢复为新文档"
	return nil
}

func (a *App) reopenWithEncoding(t *Tab, encoding document.Encoding) bool {
	if t == nil || t.Doc == nil || t.isBinary() || !t.ByteInspector {
		return false
	}
	d, err := document.OpenWithEncoding(t.Doc.Path, encoding)
	if err != nil {
		a.report(fmt.Errorf("按 %s 重新打开：%w", encoding, err))
		return false
	}
	a.invalidateTabExport(t, "重新解读文档，已取消导出")
	if !a.disposePresentation(t) {
		return false
	}
	t.Doc = d
	a.invalidateDiskChecks()
	t.Editor.SetBuffer(d.Buffer)
	t.Editor.SetReadOnly(d.ReadOnly)
	t.ByteInspector = false
	t.CanReinterpret = false
	t.Warning = fmt.Sprintf("已按明确选择的 %s 编码打开。", encoding)
	t.Mode = "source"
	if !a.configureTab(t) {
		return false
	}
	a.startWatch(d.Path, false)
	a.markPersistPending()
	a.Message = fmt.Sprintf("已按 %s 重新打开 %s", encoding, t.Name)
	return true
}
func canonical(path string) string {
	p, err := filepath.Abs(path)
	if err == nil {
		path = p
	}
	if p, err := filepath.EvalSymlinks(path); err == nil {
		path = p
	}
	return filepath.Clean(path)
}
func (a *App) Open(path string) {
	a.OpenMany([]string{path})
}
func (a *App) OpenAt(path string, start, end int) {
	target := &revealTarget{Start: start, End: end}
	a.openRequests([]openRequest{{path: path, reveal: target}})
}
func (a *App) OpenMany(paths []string) {
	requests := make([]openRequest, 0, len(paths))
	for _, path := range paths {
		requests = append(requests, openRequest{path: path})
	}
	a.openRequests(requests)
}
func (a *App) openRequests(requests []openRequest) <-chan struct{} {
	done := make(chan struct{})
	if len(requests) == 0 {
		close(done)
		return done
	}
	pending := append([]openRequest(nil), requests...)
	work := func() {
		defer func() {
			for _, request := range pending {
				if r := request.reopen; r != nil {
					r.outcome.CompareAndSwap(reopenPending, reopenRefused)
					r.finished.Store(true)
				}
			}
		}()
		a.openMu.Lock()
		defer a.openMu.Unlock()
		for _, request := range pending {
			path := canonical(request.path)
			var existing, allowed bool
			var admissionErr error
			if !a.updateWait(func() {
				if err := a.checkReopen(request.reopen); err != nil {
					a.report(err)
					return
				}
				if err := a.checkWorkspaceSearch(request.search); err != nil {
					a.report(err)
					return
				}
				allowed = true
				existing = a.selectRequestedDuplicate(path, request)
				if !existing {
					if admissionErr = a.admitNewTab(); admissionErr != nil {
						a.report(fmt.Errorf("打开 %s：%w", filepath.Base(path), admissionErr))
					}
				}
			}) {
				a.reportAsync(errors.New("打开文件时界面未能响应；已停止排队，原文件未被修改"))
				return
			}
			if !allowed {
				return
			}
			if existing {
				continue
			}
			if admissionErr != nil {
				continue
			}
			if candidate, kind, notice := probeImage(path); candidate && request.search == nil {
				if !a.openBinaryRequest(path, kind, notice, request.reopen) {
					return
				}
				continue
			}
			d, warning, issueDetail, inspector, canReinterpret, err := loadForOpen(path)
			if !a.updateWait(func() {
				if err := a.checkReopen(request.reopen); err != nil {
					a.report(err)
					return
				}
				if err != nil {
					a.report(err)
					return
				}
				if err := a.checkWorkspaceSearch(request.search); err != nil {
					a.report(err)
					return
				}
				if a.selectRequestedDuplicate(path, request) {
					return
				}
				capturedReopen := newReopenObservation(request.reopen, a.Current())
				var observation *addRequestObservation
				if request.search != nil {
					searchTarget := request.search
					observation = &addRequestObservation{
						document:  d,
						inspector: inspector,
						workspace: &workspaceSearchObservation{
							index:  searchTarget.index,
							id:     searchTarget.id,
							root:   searchTarget.root,
							result: searchTarget.result,
							origin: searchTarget.origin,
						},
						attachment: &addAttachmentObservation{
							document:  d,
							inspector: inspector,
							buffer:    d.Buffer,
						},
						reopen: capturedReopen,
					}
				} else if request.reopen != nil {
					observation = &addRequestObservation{reopen: capturedReopen}
				}
				beforeAdd := a.workspaceSearchAdoptionCheck(request.search, d, inspector, a.reopenAdoptionCheck(request.reopen))
				if request.search != nil && beforeAdd != nil {
					if err := beforeAdd(); err != nil {
						a.report(err)
						return
					}
				}
				if issueDetail != "" {
					a.report(errors.New(issueDetail))
				}
				if t := a.addDocumentChecked(d, filepath.Base(path), warning, inspector, canReinterpret, beforeAdd, observation); t != nil {
					a.acknowledgeReopen(request.reopen)
					if request.search != nil {
						request.search.origin = t
						if err := a.checkWorkspaceSearch(request.search); err != nil {
							a.report(err)
							return
						}
						a.Message = "已打开 " + filepath.Base(path)
						a.activateSearchResult(request.search.index)
						return
					}
					if request.reveal != nil && !a.revealTab(t, *request.reveal) {
						return
					}
					a.Message = "已打开 " + filepath.Base(path)
					if request.reopen != nil {
						a.Message = "已按当前磁盘内容重新打开 " + filepath.Base(path)
					}
				}
			}) {
				a.reportAsync(errors.New("打开文件后界面未能接收结果；原文件未被修改"))
				return
			}
		}
	}
	if a.Window == nil && a.uiDispatch == nil {
		work()
		close(done)
	} else {
		go func() {
			defer close(done)
			work()
		}()
	}
	return done
}
func loadForOpen(path string) (*document.Document, string, string, bool, bool, error) {
	d, err := document.Open(path)
	if err == nil {
		return d, "", "", false, false, nil
	}
	if !errors.Is(err, document.ErrBinary) &&
		!errors.Is(err, document.ErrUnsupportedEncoding) &&
		!errors.Is(err, document.ErrTooLarge) {
		return nil, "", "", false, false, err
	}
	original := err
	inspection, inspectErr := document.Inspect(path, 64<<10)
	if inspectErr != nil {
		return nil, "", "", false, false, fmt.Errorf("%w；字节视图读取失败：%v", original, inspectErr)
	}
	d = document.New(hex.Dump(inspection.Bytes))
	d.Path = path
	d.ReadOnly = true
	d.IsNew = false
	warning := fmt.Sprintf("检测到二进制内容，文件以只读字节视图打开。文件共 %d 字节；显示前 %d 字节，字节视图不会写回原文件。", inspection.Size, len(inspection.Bytes))
	switch {
	case errors.Is(original, document.ErrUnsupportedEncoding):
		warning = fmt.Sprintf("原始编码无法识别，文件以只读字节视图打开。文件共 %d 字节；显示前 %d 字节。请明确选择编码重新解读；未选择时不会转码或覆盖原文件。", inspection.Size, len(inspection.Bytes))
	case errors.Is(original, document.ErrTooLarge):
		warning = fmt.Sprintf("文件超过可编辑大小上限，以只读字节视图打开。文件共 %d 字节；显示前 %d 字节，字节视图不会写回原文件。", inspection.Size, len(inspection.Bytes))
	}
	issueDetail := fmt.Sprintf("打开 %s 时自动解码失败，已改用只读字节视图；原始错误：%v", path, original)
	return d, warning, issueDetail, true, errors.Is(original, document.ErrUnsupportedEncoding), nil
}
func (a *App) revealTab(t *Tab, reveal revealTarget) bool {
	if t == nil {
		return false
	}
	if !a.flushTab(t, "跳转搜索结果") {
		return false
	}
	return a.revealSource(t, reveal)
}

func (a *App) revealSource(t *Tab, reveal revealTarget) bool {
	if !t.textCapable() {
		a.Message = "图片只提供阅读和只读字节视图，没有文本位置"
		return false
	}
	if reveal.Start < 0 || reveal.End < reveal.Start || reveal.End > t.Doc.Buffer.LenUTF16() {
		a.report(errors.New("搜索结果位置已失效；文件已打开，但未跳转"))
		return false
	}
	if !a.releasePresentation(t) {
		return false
	}
	t.Mode = "source"
	a.bindSourceInput(t)
	t.Editor.Reveal(reveal.Start, reveal.End)
	a.markPersistPending()
	return true
}
func (a *App) ChooseOpen() {
	finishDialog := a.deferExportDelivery()
	paths, err := a.chooseOpenPaths(mygo.OpenDialogOptions{Parent: a.Window, Title: "打开文件", Multiple: true})
	finishDialog()
	if err != nil {
		a.report(err)
		return
	}
	a.OpenMany(paths)
}
func (a *App) ChooseFolder() {
	defer a.deferExportDelivery()()
	paths, err := a.chooseOpenPaths(mygo.OpenDialogOptions{Parent: a.Window, Title: "打开工作区", Directory: true})
	if err != nil {
		a.report(err)
		return
	}
	if len(paths) > 0 {
		if cancel := a.watches[a.Root]; cancel != nil {
			cancel()
			delete(a.watches, a.Root)
		}
		a.Root = paths[0]
		a.refreshTree()
		a.startWatch(a.Root, true)
		a.Persist()
	}
}
func (a *App) refreshTree() {
	if a.shuttingDown.Load() {
		return
	}
	a.treeGeneration++
	root := a.Root
	a.resetTreeIdentity(root)
	if root == "" {
		a.Entries = nil
		a.TreeTruncated = false
		a.TreeIssues = nil
		return
	}
	expanded := make(map[string]bool, len(a.Expanded))
	for path, open := range a.Expanded {
		if open {
			expanded[path] = true
		}
	}
	a.treeWorker.submit(treeScanTask{
		root: root, expanded: expanded, generation: a.treeGeneration,
	})
}

func (a *App) resetTreeIdentity(root string) {
	if a.treeRoot == root {
		return
	}
	a.treeRoot = root
	a.TreeSelected = -1
	a.treeListLifetime.reset(&a.TreeList)
	// An asynchronous scan must not leave the previous root's rows actionable
	// under the new root label while the replacement is still pending.
	a.Entries = nil
	a.TreeTruncated = false
	a.TreeIssues = nil
}
func (a *App) Save(t *Tab, as bool) bool {
	defer a.deferExportDelivery()()
	if t == nil || t.Saving {
		return false
	}
	if t.isBinary() {
		return a.saveBinary(t, as)
	}
	t.Saving = true
	defer func() { t.Saving = false }()
	if t.ByteInspector {
		a.report(errors.New("字节视图只用于检查原文件，不能保存为文档"))
		return false
	}
	owner, buffer, input, view, adapter := t.Doc, t.Doc.Buffer, t.Editor, t.View, t.Adapter
	active, generation, name, mode, sourcePath := a.Current(), t.OutlineFactoryGeneration, t.Name, t.Mode, t.Doc.Path
	stillOwned := func() bool {
		return !a.shuttingDown.Load() && a.tabIsOpen(t) && a.Current() == active &&
			t.Doc == owner && t.Doc.Buffer == buffer && t.Editor == input && t.View == view &&
			t.Adapter == adapter && t.OutlineFactoryGeneration == generation &&
			t.Name == name && t.Mode == mode && t.Doc.Path == sourcePath
	}
	if !a.flushTab(t, "保存") {
		return false
	}
	if !stillOwned() {
		a.report(errors.New("保存前文档或格式已变化，请重新保存"))
		return false
	}
	oldPath := t.Doc.Path
	if oldPath != "" {
		oldPath = canonical(oldPath)
	}
	oldMode := t.Mode
	if t.Doc.ReadOnly {
		as = true
	}
	save := a.saveDocument
	if save == nil {
		save = func(doc *document.Document, path string) error {
			if path != "" {
				return doc.SaveAs(path)
			}
			return doc.Save()
		}
	}
	var destination string
	if as || t.Doc.IsNew || t.Doc.Path == "" {
		path, dialogErr := a.chooseSavePath(t)
		if dialogErr != nil {
			a.report(dialogErr)
			return false
		}
		if path == "" {
			return false
		}
		if !stillOwned() {
			a.report(errors.New("选择保存位置期间文档或格式已变化，请重新保存"))
			return false
		}
		if !a.flushTab(t, "另存为") {
			return false
		}
		if !stillOwned() {
			a.report(errors.New("另存为前文档或格式已变化，请重新保存"))
			return false
		}
		destination = path
	}
	if err := save(t.Doc, destination); err != nil {
		if errors.Is(err, document.ErrConflict) {
			t.Conflict = true
		}
		var publication *document.PublicationError
		if errors.As(err, &publication) {
			state := "文档元数据保持原状态"
			if publication.MetadataAdopted {
				// The flush/ownership barriers above succeeded. Rebind only the
				// presentation metadata; the original saved/disk baseline and
				// current source-buffer draft remain owned by the document.
				if !a.reconcileSavedMetadata(t, oldPath, oldMode, true) {
					return false
				}
				a.markPersistPending()
				state = "文档路径/新建/只读元数据已采用目标，保存基线未确认"
			}
			a.report(fmt.Errorf("保存未完成：字节可能已写入目标 %s；%s；草稿仍未保存；重试前请检查目标或另存新副本：%w", publication.Path, state, err))
		} else {
			a.report(err)
		}
		return false
	}
	t.Conflict = false
	if !a.reconcileSavedMetadata(t, oldPath, oldMode, destination != "") {
		return false
	}
	a.Message = "已保存 " + t.Name
	a.Persist()
	return true
}

// reconcileSavedMetadata also serves SaveAs failures after target adoption.
// It is not a success transition: it never clears conflict, marks the document
// clean, persists a clean receipt, or reports completion.
func (a *App) reconcileSavedMetadata(t *Tab, oldPath, oldMode string, forceLanguageRetirement bool) bool {
	doc, input, buffer, view, adapter := t.Doc, t.Editor, t.Doc.Buffer, t.View, t.Adapter
	path, name, mode, epoch := doc.Path, t.Name, t.Mode, t.inputEpoch
	stamp, active, generation := buffer.ChangeStamp(), a.Current(), t.OutlineFactoryGeneration
	current := func() bool {
		return a.tabIsOpen(t) && t.Doc == doc && t.Editor == input && doc.Buffer == buffer && input.Buffer == buffer &&
			t.View == view && t.Adapter == adapter && doc.Path == path && t.Name == name && t.Mode == mode &&
			t.inputEpoch == epoch && buffer.ChangeStamp() == stamp && a.Current() == active && t.OutlineFactoryGeneration == generation
	}
	savedName := filepath.Base(t.Doc.Path)
	owner := a.sourceLanguage.owner
	binding := a.sourceLanguage.shortcut
	// Check the language seals before changing the presentation metadata.
	metadataChanged := t.Name != savedName || t.Editor.ReadOnly != t.Doc.ReadOnly
	bindingChanged := binding != nil && binding.tab == t && (metadataChanged || !a.sourceLanguageShortcutCurrent(binding))
	ownerChanged := owner != nil && owner.tab == t && (metadataChanged || a.sourceLanguageOwnerValid(owner, false) != nil)
	inputBinding := t.inputBinding
	inputChanged := inputBinding != nil && inputBinding.tab == t &&
		(forceLanguageRetirement || metadataChanged || !a.sourceInputCurrent(inputBinding))
	if forceLanguageRetirement || bindingChanged || ownerChanged || inputChanged {
		if !a.retireSourceLanguage(t) {
			return false
		}
		if !current() || a.sourceLanguage.owner != nil && a.sourceLanguage.owner.tab == t {
			return false
		}
	}
	if inputChanged {
		a.revokeSourceInput(t)
	}
	t.Name = savedName
	t.Editor.SetReadOnly(t.Doc.ReadOnly)
	if inputChanged && strings.EqualFold(oldPath, t.Doc.Path) {
		a.bindSourceInput(t)
	}
	a.invalidateDiskChecks()
	if !strings.EqualFold(oldPath, t.Doc.Path) {
		a.invalidateTabExport(t, "文档路径或格式已变化，已取消导出")
		if cancel := a.watches[oldPath]; cancel != nil {
			cancel()
			delete(a.watches, oldPath)
		}
		if !a.disposePresentation(t) {
			return false
		}
		t.Adapter = nil
		if !a.configureTab(t) {
			return false
		}
		if (oldMode == "preview" || oldMode == "split") && t.Adapter != nil {
			t.Mode = oldMode
		} else {
			t.Mode = "source"
		}
		a.startWatch(t.Doc.Path, false)
	}
	a.bindSourceInput(t)
	return true
}

func (a *App) chooseSavePath(t *Tab) (string, error) {
	if a.saveDialog != nil {
		return a.saveDialog(t)
	}
	return mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: a.Window, Title: "另存为", DefaultPath: t.Name})
}

func (a *App) CloseTab(index int) bool {
	defer a.deferExportDelivery()()
	if index < 0 || index >= len(a.Tabs) {
		return true
	}
	t := a.Tabs[index]
	if _, ok := a.prepareClose(t); !ok {
		return false
	}
	// A close/SaveAs dialog may run a nested event loop. Never remove the tab
	// that happens to occupy the old slot after it returns.
	if a.shuttingDown.Load() || index >= len(a.Tabs) || a.Tabs[index] != t {
		a.report(errors.New("关闭期间标签已变化；未关闭其他标签"))
		return false
	}
	path, omission := closedFilePath(t)
	remaining, active := tabsAfterRemoval(a.Tabs, a.Active, index)
	removed := false
	committedWithWarning := false
	if !a.commitCloseWithOutcome(remaining, active, func() {
		if index < len(a.Tabs) && a.Tabs[index] == t {
			a.removeTab(index)
			removed = !a.tabIsOpen(t)
		}
	}, &committedWithWarning) || !removed {
		return false
	}
	if path != "" {
		a.recordClosedFile(path)
	} else if omission != "" && !committedWithWarning {
		a.Message = omission
	}
	return true
}
func (a *App) prepareClose(t *Tab) (discarded bool, ok bool) {
	owner := sealSessionTab(t, false)
	active, config := a.Active, a.configDir
	if !a.flushTab(t, "关闭标签") {
		return false, false
	}
	if !owner() || a.Active != active || a.configDir != config || a.shuttingDown.Load() {
		a.report(errCloseOwnerChanged)
		return false, false
	}
	if t.isBinary() {
		return false, true
	}
	if t.Doc.Dirty() || t.Doc.IsNew {
		valid := a.sessionSeal(true)
		decision, err := a.chooseCloseAction(t)
		if !valid() {
			a.report(errCloseOwnerChanged)
			return false, false
		}
		if err != nil {
			a.report(err)
			return false, false
		}
		switch decision {
		case closeSave:
			if !a.Save(t, false) {
				return false, false
			}
		case closeDiscard:
			return true, true
		default:
			return false, false
		}
	}
	return false, true
}

func (a *App) chooseCloseAction(t *Tab) (closeDecision, error) {
	if a.closeDialog != nil {
		return a.closeDialog(t)
	}
	r, err := mygo.Dialog.Message(mygo.MessageOptions{Parent: a.Window, Type: mygo.MessageQuestion, Title: "保存修改", Message: t.Name + " 有未保存的修改", Buttons: []string{"保存", "取消", "不保存"}, DefaultButton: 0, CancelButton: 1, DestructiveButtons: []int{2}})
	if err != nil {
		return closeCancel, err
	}
	switch r.Button {
	case 0:
		return closeSave, nil
	case 2:
		return closeDiscard, nil
	default:
		return closeCancel, nil
	}
}

type editorTerminalSeal struct {
	editor           *editor.Editor
	buffer           reflect.Value
	readOnly         reflect.Value
	viewGeneration   reflect.Value
	observedBuffer   reflect.Value
	observedReadonly reflect.Value
	observedView     reflect.Value
	bufferEpoch      reflect.Value
	readonlyEpoch    reflect.Value
	viewEpoch        reflect.Value
	registration     reflect.Value
}

func editorTerminalSealFromValue(input *editor.Editor, value reflect.Value) (editorTerminalSeal, bool) {
	field := func(name string, kind reflect.Kind) (reflect.Value, bool) {
		field := value.FieldByName(name)
		return field, field.IsValid() && field.Kind() == kind
	}
	seal := editorTerminalSeal{editor: input}
	var ok bool
	if seal.buffer, ok = field("Buffer", reflect.Pointer); !ok {
		return editorTerminalSeal{}, false
	}
	if seal.readOnly, ok = field("ReadOnly", reflect.Bool); !ok {
		return editorTerminalSeal{}, false
	}
	if seal.viewGeneration, ok = field("viewGeneration", reflect.Uint64); !ok {
		return editorTerminalSeal{}, false
	}
	if seal.observedBuffer, ok = field("terminalObservedBuffer", reflect.Pointer); !ok {
		return editorTerminalSeal{}, false
	}
	if seal.observedReadonly, ok = field("terminalObservedReadonly", reflect.Bool); !ok {
		return editorTerminalSeal{}, false
	}
	if seal.observedView, ok = field("terminalObservedView", reflect.Uint64); !ok {
		return editorTerminalSeal{}, false
	}
	if seal.bufferEpoch, ok = field("terminalBufferEpoch", reflect.Pointer); !ok {
		return editorTerminalSeal{}, false
	}
	if seal.readonlyEpoch, ok = field("terminalReadonlyEpoch", reflect.Pointer); !ok {
		return editorTerminalSeal{}, false
	}
	if seal.viewEpoch, ok = field("terminalViewEpoch", reflect.Pointer); !ok {
		return editorTerminalSeal{}, false
	}
	if seal.registration, ok = field("terminalRegistration", reflect.Pointer); !ok {
		return editorTerminalSeal{}, false
	}
	return seal, true
}

func captureEditorTerminalSeal(input *editor.Editor) (editorTerminalSeal, bool) {
	if input == nil {
		return editorTerminalSeal{}, false
	}
	readOnly := input.ReadOnly
	input.SetReadOnly(readOnly)
	return editorTerminalSealFromValue(input, reflect.ValueOf(*input))
}

func (seal editorTerminalSeal) current(input *editor.Editor) bool {
	if input == nil || input != seal.editor {
		return false
	}
	current, ok := editorTerminalSealFromValue(input, reflect.ValueOf(*input))
	if !ok {
		return false
	}
	return seal.buffer.Equal(current.buffer) && seal.readOnly.Equal(current.readOnly) &&
		seal.viewGeneration.Equal(current.viewGeneration) && seal.observedBuffer.Equal(current.observedBuffer) &&
		seal.observedReadonly.Equal(current.observedReadonly) && seal.observedView.Equal(current.observedView) &&
		seal.bufferEpoch.Equal(current.bufferEpoch) && seal.readonlyEpoch.Equal(current.readonlyEpoch) &&
		seal.viewEpoch.Equal(current.viewEpoch) && seal.registration.Equal(current.registration)
}

func (seal editorTerminalSeal) equalExceptBuffer(current editorTerminalSeal) bool {
	return seal.readOnly.Equal(current.readOnly) && seal.viewGeneration.Equal(current.viewGeneration) &&
		seal.observedReadonly.Equal(current.observedReadonly) && seal.observedView.Equal(current.observedView) &&
		seal.readonlyEpoch.Equal(current.readonlyEpoch) && seal.viewEpoch.Equal(current.viewEpoch) &&
		seal.registration.Equal(current.registration)
}

func (seal editorTerminalSeal) replaceBuffer(input *editor.Editor, buffer *text.Buffer) (editorTerminalSeal, bool) {
	if input == nil || buffer == nil || input != seal.editor || input.Buffer != buffer {
		return editorTerminalSeal{}, false
	}
	current, ok := editorTerminalSealFromValue(input, reflect.ValueOf(*input))
	if !ok || !seal.equalExceptBuffer(current) || seal.buffer.Equal(current.buffer) ||
		!current.buffer.Equal(reflect.ValueOf(buffer)) || !current.observedBuffer.Equal(reflect.ValueOf(buffer)) ||
		current.bufferEpoch.IsNil() || current.bufferEpoch.Equal(seal.bufferEpoch) {
		return editorTerminalSeal{}, false
	}
	return current, true
}

func (seal editorTerminalSeal) equalExceptRegistration(current editorTerminalSeal) bool {
	return seal.buffer.Equal(current.buffer) && seal.readOnly.Equal(current.readOnly) &&
		seal.viewGeneration.Equal(current.viewGeneration) && seal.observedBuffer.Equal(current.observedBuffer) &&
		seal.observedReadonly.Equal(current.observedReadonly) && seal.observedView.Equal(current.observedView) &&
		seal.bufferEpoch.Equal(current.bufferEpoch) && seal.readonlyEpoch.Equal(current.readonlyEpoch) &&
		seal.viewEpoch.Equal(current.viewEpoch)
}

func (seal editorTerminalSeal) replaceRegistration(input *editor.Editor) (editorTerminalSeal, bool) {
	if input == nil || input != seal.editor {
		return editorTerminalSeal{}, false
	}
	current, ok := editorTerminalSealFromValue(input, reflect.ValueOf(*input))
	if !ok || !seal.equalExceptRegistration(current) || seal.registration.Equal(current.registration) ||
		current.registration.IsNil() {
		return editorTerminalSeal{}, false
	}
	return current, true
}

func (seal editorTerminalSeal) retireView(input *editor.Editor) (editorTerminalSeal, bool) {
	if input == nil || input != seal.editor {
		return editorTerminalSeal{}, false
	}
	current, ok := editorTerminalSealFromValue(input, reflect.ValueOf(*input))
	if !ok || !seal.equalExceptView(current) {
		return editorTerminalSeal{}, false
	}
	view := seal.viewGeneration.Uint() + 1
	if current.viewGeneration.Uint() != view || current.observedView.Uint() != view ||
		current.viewEpoch.IsNil() || current.viewEpoch.Equal(seal.viewEpoch) {
		return editorTerminalSeal{}, false
	}
	return current, true
}

func (seal editorTerminalSeal) equalExceptView(current editorTerminalSeal) bool {
	return seal.buffer.Equal(current.buffer) && seal.readOnly.Equal(current.readOnly) &&
		seal.observedBuffer.Equal(current.observedBuffer) && seal.observedReadonly.Equal(current.observedReadonly) &&
		seal.bufferEpoch.Equal(current.bufferEpoch) && seal.readonlyEpoch.Equal(current.readonlyEpoch) &&
		seal.registration.Equal(current.registration)
}

func (a *App) releasePresentation(t *Tab) bool {
	return a.releaseTabPresentation(t, false)
}

func (a *App) releaseTabPresentation(t *Tab, keepBinarySnapshot bool) bool {
	if t == nil {
		return true
	}
	doc0, input0, view0, adapter0 := t.Doc, t.Editor, t.View, t.Adapter
	binary0 := t.Binary
	var binaryToken0 document.BinaryToken
	if binary0 != nil {
		binaryToken0 = binary0.Token()
	}
	binaryPath0, binaryKind0, binarySize0, binaryGeneration0 := t.BinaryPath, t.BinaryKind, t.BinarySize, t.BinaryGeneration
	active, activeIndex, route, mode, generation := a.Current(), a.Active, t.sourceLanguageRoute, t.Mode, t.OutlineFactoryGeneration
	manager, profile, name, inspector := a.pluginManager, t.sourceLanguageProfile, t.Name, t.ByteInspector
	sourceSession, formatHostEpoch0 := a.sourceLanguage.session, t.formatHostEpoch
	configDir0, tabID0 := a.configDir, t.ID
	restoring0, shuttingDown0 := a.restoring, a.shuttingDown.Load()
	var docBuffer0, inputBuffer0 *text.Buffer
	var path string
	var encoding document.Encoding
	var bom, docReadOnly, readOnly bool
	if doc0 != nil {
		docBuffer0, path, encoding, bom, docReadOnly = doc0.Buffer, doc0.Path, doc0.Encoding, doc0.BOM, doc0.ReadOnly
	}
	if input0 != nil {
		inputBuffer0, readOnly = input0.Buffer, input0.ReadOnly
	}
	t.formatHostActive = false // revoke format callbacks before any release seam
	a.revokeSourceInput(t)     // no terminal observer during the release continuation
	epoch := t.inputEpoch
	expectedInputBuffer, expectedGeneration, expectedEpoch := inputBuffer0, binaryGeneration0, epoch
	expectedBinding := (*sourceInputBinding)(nil)
	terminalSeal, terminalSealed := captureEditorTerminalSeal(input0)
	current := func() bool {
		if a.shuttingDown.Load() != shuttingDown0 || a.restoring != restoring0 || !a.tabIsOpen(t) ||
			t.ID != tabID0 || a.configDir != configDir0 || a.Active != activeIndex ||
			t.Doc != doc0 || t.Editor != input0 || t.View != view0 || t.Adapter != adapter0 ||
			t.inputEpoch != expectedEpoch || t.sourceLanguageRoute != route || t.Mode != mode ||
			t.OutlineFactoryGeneration != generation || a.Current() != active || a.pluginManager != manager ||
			a.sourceLanguage.session != sourceSession ||
			t.sourceLanguageProfile != profile || t.Name != name || t.ByteInspector != inspector ||
			t.formatHostEpoch != formatHostEpoch0 || t.formatHostActive {
			return false
		}
		if doc0 != nil && (doc0.Buffer != docBuffer0 || doc0.Path != path || doc0.Encoding != encoding ||
			doc0.BOM != bom || doc0.ReadOnly != docReadOnly) {
			return false
		}
		if input0 != nil && (input0.Buffer != expectedInputBuffer || input0.ReadOnly != readOnly) {
			return false
		}
		if (input0 != nil && !terminalSealed) || t.inputBinding != expectedBinding ||
			(input0 != nil && !terminalSeal.current(input0)) {
			return false
		}
		if t.Binary != binary0 || t.BinaryPath != binaryPath0 || t.BinaryKind != binaryKind0 ||
			t.BinarySize != binarySize0 || t.BinaryGeneration != expectedGeneration {
			return false
		}
		if binary0 != nil && !binary0.SameBaseline(binaryToken0) {
			return false
		}
		return true
	}
	if !a.retireSourceLanguage(t) || !current() {
		return false
	}
	a.cancelPreview(t)
	if !current() {
		return false
	}
	if keepBinarySnapshot && t.isBinary() {
		t.BinaryGeneration++
		a.cancelBinaryPreview(t)
		expectedGeneration = binaryGeneration0 + 1
	} else {
		a.deactivateBinary(t)
		if t.isBinary() {
			expectedGeneration = binaryGeneration0 + 1
			if binary0 != nil {
				expectedInputBuffer = input0.Buffer
				if expectedInputBuffer == nil || expectedInputBuffer == inputBuffer0 || expectedInputBuffer.LenBytes() != 0 {
					return false
				}
				nextTerminalSeal, replaced := terminalSeal.replaceBuffer(input0, expectedInputBuffer)
				if !replaced {
					return false
				}
				terminalSeal = nextTerminalSeal
			}
		}
	}
	if !current() {
		return false
	}
	a.releaseOutline()
	if !current() {
		return false
	}
	var releaseErr error
	if input0 != nil {
		originalOnError := input0.OnError
		collecting := true
		input0.OnError = func(err error) {
			if collecting {
				if releaseErr == nil {
					releaseErr = err
				}
				return
			}
			if originalOnError != nil {
				originalOnError(err)
			}
		}
		input0.ReleaseView()
		collecting = false
		nextTerminalSeal, retired := terminalSeal.retireView(input0)
		if !retired {
			return false
		}
		terminalSeal = nextTerminalSeal
		if !current() {
			return false
		}
		input0.OnError = originalOnError
	}
	// Reprime this still-live Editor for recovery, including inactive tabs.
	if doc0 != nil && (!current() || t.inputBinding != nil || t.inputEpoch != expectedEpoch) {
		return false
	}
	a.bindSourceInput(t)
	b := expectedBinding
	if doc0 != nil {
		bR := t.inputBinding
		expectedEpoch = epoch + 1
		if bR == nil || bR.epoch != expectedEpoch || !a.sourceInputConsumerCurrent(bR) {
			return false
		}
		nextTerminalSeal, replaced := terminalSeal.replaceRegistration(input0)
		if !replaced {
			return false
		}
		terminalSeal = nextTerminalSeal
		expectedBinding = bR
		b = bR
		if !current() {
			return false
		}
	} else if b != nil {
		return false
	}
	if releaseErr != nil {
		a.report(releaseErr)
		return false
	}
	if view0 != nil {
		view0.Release()
	}
	if !current() {
		return false
	}
	if doc0 != nil && !a.sourceInputConsumerCurrent(b) {
		return false
	}
	if doc0 == nil && t.inputBinding != nil {
		return false
	}
	return true
}
func (a *App) disposePresentation(t *Tab) bool {
	if t == nil {
		return true
	}
	view := t.View
	if !a.releasePresentation(t) || t.View != view {
		return false
	}
	t.formatHostEpoch++
	t.View = nil
	a.bindSourceInput(t)
	return true
}
func (a *App) removeTab(index int) {
	if index < 0 || index >= len(a.Tabs) {
		return
	}
	a.closeGoToLine()
	a.invalidateFindCache()
	t := a.Tabs[index]
	a.invalidateTabExport(t, "标签已关闭，已取消导出")
	if !a.disposePresentation(t) {
		return
	}
	a.revokeSourceInput(t)
	a.cancelBinaryForClose(t)
	if cancel := a.watches[t.path()]; cancel != nil {
		cancel()
		delete(a.watches, t.path())
	}
	a.invalidateDiskChecks()
	copy(a.Tabs[index:], a.Tabs[index+1:])
	a.Tabs[len(a.Tabs)-1] = nil
	a.Tabs = a.Tabs[:len(a.Tabs)-1]
	if len(a.Tabs) == 0 {
		a.Active = -1
	} else {
		if a.Active > index {
			a.Active--
		}
		a.Active = max(0, min(a.Active, len(a.Tabs)-1))
		a.Current().Editor.Focus()
	}
}
func (a *App) CanClose() bool {
	if a.modalDepth > 0 {
		a.Message = "请先结束当前对话框再关闭窗口"
		return false
	}
	if !a.prepareBinaryWindowClose() {
		return false
	}
	defer a.deferExportDelivery()()
	tabs := append([]*Tab(nil), a.Tabs...)
	active, config := a.Active, a.configDir
	discarded := make([]bool, len(tabs))
	processed := make([]func() bool, 0, len(tabs))
	for i, t := range tabs {
		var ok bool
		discarded[i], ok = a.prepareClose(t)
		if !ok {
			return false
		}
		if len(a.Tabs) != len(tabs) || a.Active != active || a.configDir != config {
			a.report(errCloseOwnerChanged)
			return false
		}
		for j, expected := range tabs {
			if a.Tabs[j] != expected {
				a.report(errCloseOwnerChanged)
				return false
			}
		}
		for _, unchanged := range processed {
			if !unchanged() {
				a.report(errCloseOwnerChanged)
				return false
			}
		}
		processed = append(processed, sealSessionTab(t, true))
	}
	return a.closePreparedSession(discarded)
}
func (a *App) commitClose(remaining []*Tab, active int, remove func()) bool {
	return a.commitCloseWithOutcome(remaining, active, remove, nil)
}

func (a *App) commitCloseWithOutcome(remaining []*Tab, active int, remove func(), committedWarning *bool) bool {
	valid := a.sessionSeal(true)
	builder, err := a.captureSessionTabs(remaining, active, nil, "关闭标签")
	if err != nil {
		return false
	}
	if !valid() {
		a.report(errCloseOwnerChanged)
		return false
	}
	snapshot, err := builder.Snapshot()
	if err != nil {
		a.reportPersistFailure(err)
		return false
	}
	path := filepath.Join(a.configDir, "session.json")
	prepared, err := a.prepareCloseSnapshot(path, snapshot)
	if err != nil {
		a.reportPersistFailure(fmt.Errorf("关闭未完成：无法准备恢复状态；现有恢复快照保持不变：%w", err))
		return false
	}
	if !valid() {
		err = closePublicationResult(path, errCloseOwnerChanged, prepared.Abort())
		a.reportPersistFailure(err)
		return false
	}
	check := func() error {
		if !valid() {
			return errCloseOwnerChanged
		}
		return nil
	}
	err = closePublicationResult(path, prepared.Commit(check), prepared.Abort())
	var publication *document.StatePublicationError
	if err != nil && (!errors.As(err, &publication) || !publication.Committed) {
		a.reportPersistFailure(fmt.Errorf("关闭未完成：恢复状态未发布；现有恢复快照保持不变：%w", err))
		return false
	}
	a.lastPersistError = ""
	remove()
	a.cancelAutosave()
	a.persistPending = false
	if err != nil {
		a.report(fmt.Errorf("关闭已提交，恢复文件已发布：%w", err))
		if committedWarning != nil {
			*committedWarning = true
		}
	}
	return true
}
func tabsAfterRemoval(tabs []*Tab, active, index int) ([]*Tab, int) {
	remaining := make([]*Tab, 0, max(0, len(tabs)-1))
	for i, t := range tabs {
		if i != index {
			remaining = append(remaining, t)
		}
	}
	if len(remaining) == 0 {
		return remaining, -1
	}
	if active > index {
		active--
	}
	active = max(0, min(active, len(remaining)-1))
	return remaining, active
}
func (a *App) Reload() {
	if t := a.Current(); t != nil {
		if t.isBinary() {
			a.reloadBinary(t)
			return
		}
		stillOwned := a.reloadOwnerCheck(t)
		if !a.flushTab(t, "重新载入") {
			return
		}
		if !stillOwned() {
			a.report(errors.New("重新载入前文档或格式已变化，请重新操作"))
			return
		}
		if t.ByteInspector {
			a.report(errors.New("字节视图不能重新载入为可编辑文本；请重新打开并明确选择编码"))
			return
		}
		if err := t.Doc.Reload(); err != nil {
			a.report(err)
		} else {
			a.finishTextReload(t, "已重新载入")
		}
	}
}

// reloadOwnerCheck pins the target across format Flush and native modal callbacks.
func (a *App) reloadOwnerCheck(t *Tab) func() bool {
	owner, buffer, input, view, adapter := t.Doc, t.Doc.Buffer, t.Editor, t.View, t.Adapter
	active, generation, name, mode, path := a.Current(), t.OutlineFactoryGeneration, t.Name, t.Mode, t.Doc.Path
	inspector := t.ByteInspector
	return func() bool {
		return !a.shuttingDown.Load() && a.tabIsOpen(t) && a.Current() == active &&
			t.Doc == owner && t.Doc.Buffer == buffer && t.Editor == input && t.View == view &&
			t.Adapter == adapter && t.OutlineFactoryGeneration == generation &&
			t.Name == name && t.Mode == mode && t.Doc.Path == path && t.ByteInspector == inspector
	}
}

// finishTextReload records an already adopted disk version before reporting success.
// An explicit attempt consumes the request, including on failure, so unchanged
// state is not retried indefinitely by the recovery timer.
func (a *App) finishTextReload(t *Tab, message string) {
	a.invalidateTabExport(t, "文档已重新载入，已取消导出")
	a.invalidateDiskChecks()
	if !a.disposePresentation(t) {
		return
	}
	t.Editor.SetBuffer(t.Doc.Buffer)
	t.Editor.SetReadOnly(t.Doc.ReadOnly)
	t.Conflict = false
	t.Mode = "source"
	if !a.configureTab(t) {
		return
	}
	if err := a.persistTabs(a.Tabs, a.Active, "已采用磁盘版本；恢复快照更新失败，旧恢复快照仍保留"); err != nil {
		a.persistPending = false
		a.reportPersistFailure(err)
		// Issue deduplication must not hide the current failure behind a later status.
		a.Message = err.Error()
		return
	}
	a.Message = message
}

func (a *App) flushTab(t *Tab, action string) bool {
	return a.flushTabResult(t, action).kind == flushCommitted
}

func (a *App) flushTabResult(t *Tab, action string) flushResult {
	if t == nil {
		return flushResult{kind: flushCommitted}
	}
	target := t
	id := target.ID
	b0, doc0, input0, view0, adapter0 := target.inputBinding, target.Doc, target.Editor, target.View, target.Adapter
	active, activeIndex, activation, sourceSession := a.Current(), a.Active, a.sourceLanguage.activation, a.sourceLanguage.session
	generation, epoch := target.OutlineFactoryGeneration, target.inputEpoch
	manager, route, profile := a.pluginManager, target.sourceLanguageRoute, target.sourceLanguageProfile
	mode, name, inspector := target.Mode, target.Name, target.ByteInspector
	restoring0, shuttingDown0 := a.restoring, a.shuttingDown.Load()
	formatHostEpoch0, formatHostActive0 := target.formatHostEpoch, target.formatHostActive
	configDir0 := a.configDir
	isBinaryTarget := target.isBinary()
	var terminalSeal editorTerminalSeal
	var terminalSealed bool
	var docBuffer0, editorBuffer0 *text.Buffer
	var binary0 *document.BinaryFile
	var binaryToken0 document.BinaryToken
	var binaryPath0 string
	var binaryKind0 document.BinaryKind
	var binarySize0 int64
	var binaryGeneration0 uint64
	var b *sourceInputBinding
	if input0 != nil {
		if doc0 != nil {
			docBuffer0, editorBuffer0 = doc0.Buffer, input0.Buffer
		} else {
			editorBuffer0 = input0.Buffer
			binary0 = target.Binary
			if binary0 != nil {
				binaryToken0 = binary0.Token()
			}
			binaryPath0, binaryKind0, binarySize0, binaryGeneration0 = target.BinaryPath, target.BinaryKind, target.BinarySize, target.BinaryGeneration
		}
	}
	loss := func() flushLoss {
		var loss flushLoss
		if input0 == nil {
			return flushLossEditorIdentity
		}
		if a.shuttingDown.Load() {
			loss |= flushLossLifecycle
		}
		if a.restoring != restoring0 {
			loss |= flushLossRestoreState
		}
		if !a.tabIsOpen(target) {
			loss |= flushLossSlot
		}
		if target.ID != id {
			loss |= flushLossTabIdentity
		}
		if a.Current() != active || a.Active != activeIndex {
			loss |= flushLossActivity
		}
		if a.sourceLanguage.activation != activation {
			loss |= flushLossActivation
		}
		if a.sourceLanguage.session != sourceSession {
			loss |= flushLossSession
		}
		if a.pluginManager != manager || a.configDir != configDir0 {
			loss |= flushLossConfig
		}
		if target.sourceLanguageRoute != route || target.sourceLanguageProfile != profile {
			loss |= flushLossRoute
		}
		if target.Mode != mode || target.Name != name || target.ByteInspector != inspector {
			loss |= flushLossMetadata
		}
		if target.OutlineFactoryGeneration != generation {
			loss |= flushLossFactory
		}
		if target.formatHostEpoch != formatHostEpoch0 || target.formatHostActive != formatHostActive0 {
			loss |= flushLossFormatHost
		}
		if target.View != view0 {
			loss |= flushLossView
		}
		if target.Adapter != adapter0 {
			loss |= flushLossAdapter
		}
		if target.Editor != input0 {
			loss |= flushLossEditorIdentity
		}
		if doc0 != nil {
			binding := b
			if binding == nil {
				binding = b0
			}
			if target.Doc != doc0 {
				loss |= flushLossDocIdentity
			}
			if doc0.Buffer != docBuffer0 || input0.Buffer != editorBuffer0 {
				loss |= flushLossBuffer
			}
			if binding != nil {
				if target.inputBinding != binding || binding.tab != target || binding.doc != doc0 ||
					binding.input != input0 || binding.buffer != docBuffer0 || !input0.InputBindingCurrent(binding.lease) {
					loss |= flushLossOpaqueLease
				}
				if target.inputEpoch != binding.epoch {
					loss |= flushLossEpoch
				}
				if binding.manager != manager {
					loss |= flushLossConfig
				}
				if binding.route != route || binding.profile != profile {
					loss |= flushLossRoute
				}
				if binding.mode != mode || binding.inspector != inspector {
					loss |= flushLossMetadata
				}
				if binding.viewGeneration != generation {
					loss |= flushLossFactory
				}
				if binding.view != view0 {
					loss |= flushLossView
				}
				if binding.adapter != adapter0 {
					loss |= flushLossAdapter
				}
				if target.Doc == doc0 {
					if binding.path != doc0.Path || binding.name != name || binding.encoding != doc0.Encoding || binding.bom != doc0.BOM {
						loss |= flushLossMetadata
					}
					if binding.docReadOnly != doc0.ReadOnly || binding.readOnly != input0.ReadOnly {
						loss |= flushLossReadOnly
					}
				}
			}
		} else {
			if target.Doc != doc0 {
				loss |= flushLossDocIdentity
			}
			if target.Doc == nil && !target.isBinary() {
				loss |= flushLossDocIdentity
			}
			if target.inputBinding != nil {
				loss |= flushLossOpaqueLease
			}
			if target.inputEpoch != epoch {
				loss |= flushLossEpoch
			}
			if target.Binary != binary0 {
				loss |= flushLossDocIdentity
			}
			if binary0 != nil && !binary0.SameBaseline(binaryToken0) {
				loss |= flushLossBinaryFullToken
			}
			if target.BinaryPath != binaryPath0 || target.BinaryKind != binaryKind0 || target.BinarySize != binarySize0 || target.BinaryGeneration != binaryGeneration0 {
				loss |= flushLossMetadata
			}
			if !input0.ReadOnly {
				loss |= flushLossReadOnly
			}
			if input0.Buffer != editorBuffer0 {
				loss |= flushLossBuffer
			}
			if terminalSealed && !terminalSeal.current(input0) {
				loss |= flushLossEditorIdentity
			}
		}
		return loss
	}
	if shuttingDown0 || !a.tabIsOpen(target) || input0 == nil {
		return flushResult{kind: flushAuthorityLost, loss: loss()}
	}
	if doc0 != nil {
		if docBuffer0 == nil || editorBuffer0 == nil || docBuffer0 != editorBuffer0 {
			return flushResult{kind: flushAuthorityLost, loss: flushLossBuffer}
		}
	} else {
		var loss flushLoss
		if !isBinaryTarget {
			loss |= flushLossDocIdentity
		}
		if !input0.ReadOnly {
			loss |= flushLossReadOnly
		}
		if loss != 0 {
			return flushResult{kind: flushAuthorityLost, loss: loss}
		}
		editorBuffer0 = input0.Buffer
		if editorBuffer0 == nil {
			return flushResult{kind: flushAuthorityLost, loss: flushLossBuffer}
		}
		if b0 != nil {
			return flushResult{kind: flushAuthorityLost, loss: flushLossOpaqueLease}
		}
		terminalSealed = false
		terminalSeal, terminalSealed = captureEditorTerminalSeal(input0)
		if !terminalSealed {
			return flushResult{kind: flushAuthorityLost, loss: flushLossEditorIdentity}
		}
	}
	a.bindSourceInput(target)
	if doc0 != nil {
		switch target.inputEpoch {
		case epoch:
			b = target.inputBinding
			if b == nil || b != b0 {
				return flushResult{kind: flushAuthorityLost, loss: flushLossOpaqueLease}
			}
		case epoch + 1:
			b = target.inputBinding
			if b == nil || b.epoch != epoch+1 {
				return flushResult{kind: flushAuthorityLost, loss: flushLossEpoch}
			}
		default:
			return flushResult{kind: flushAuthorityLost, loss: flushLossEpoch}
		}
	} else {
		if target.inputEpoch != epoch || target.inputBinding != nil {
			return flushResult{kind: flushAuthorityLost, loss: flushLossEpoch | flushLossOpaqueLease}
		}
	}
	current := func() bool {
		if a.shuttingDown.Load() || a.restoring != restoring0 || !a.tabIsOpen(target) || target.ID != id ||
			a.Current() != active || a.Active != activeIndex || a.sourceLanguage.activation != activation || a.sourceLanguage.session != sourceSession ||
			a.pluginManager != manager || a.configDir != configDir0 ||
			target.sourceLanguageRoute != route || target.sourceLanguageProfile != profile ||
			target.Mode != mode || target.Name != name || target.ByteInspector != inspector ||
			target.OutlineFactoryGeneration != generation ||
			target.formatHostEpoch != formatHostEpoch0 || target.formatHostActive != formatHostActive0 ||
			target.View != view0 || target.Adapter != adapter0 || target.Editor != input0 {
			return false
		}
		if doc0 != nil {
			return target.Doc == doc0 && doc0.Buffer == docBuffer0 && input0.Buffer == editorBuffer0 && a.sourceInputConsumerCurrent(b)
		}
		return target.Doc == nil && target.isBinary() && target.inputBinding == nil && target.inputEpoch == epoch &&
			target.Binary == binary0 && (binary0 == nil || binary0.SameBaseline(binaryToken0)) &&
			target.BinaryPath == binaryPath0 && target.BinaryKind == binaryKind0 &&
			target.BinarySize == binarySize0 && target.BinaryGeneration == binaryGeneration0 &&
			input0.ReadOnly && input0.Buffer == editorBuffer0 && terminalSeal.current(input0)
	}
	if !current() {
		return flushResult{kind: flushAuthorityLost, loss: loss()}
	}
	err := input0.Flush()
	if !current() {
		return flushResult{kind: flushAuthorityLost, loss: loss()}
	} // never report through a callback's successor
	if err != nil {
		a.report(fmt.Errorf("%s前未能完成文本输入：%w", action, err))
		return flushResult{kind: flushSeamErrorReported}
	}
	if view0 == nil {
		return flushResult{kind: flushCommitted}
	}
	err = editor.FlushView(view0)
	if !current() {
		return flushResult{kind: flushAuthorityLost, loss: loss()}
	}
	if err != nil {
		a.report(fmt.Errorf("%s前未能同步格式编辑：%w", action, err))
		return flushResult{kind: flushSeamErrorReported}
	}
	return flushResult{kind: flushCommitted}
}
