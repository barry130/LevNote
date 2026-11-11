package app

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	OriginalPath             string
	View                     editor.View
	Conflict                 bool
	ByteInspector            bool
	CanReinterpret           bool
}
type Entry struct {
	Path, Name string
	Directory  bool
	Depth      int
}
type App struct {
	Window                                          *mygo.Window
	uiDispatch                                      func(func())
	saveDialog                                      func(*Tab) (string, error)
	closeDialog                                     func(*Tab) (closeDecision, error)
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
	exportStatuses                                  map[string]exportEngineStatus
	modalDepth                                      int
	openDocuments                                   openDocumentPicker
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
		FontSize: 16, SidebarWidth: 228, LineNumbers: true,
		Expanded: map[string]bool{}, configDir: configDir,
		adapters: modules.Builtins(), pluginManager: manager,
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
		a.releasePresentation(old)
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
	return a.addDocumentChecked(d, name, warning, byteInspector, canReinterpret, nil)
}

// beforeAdd is used only by route-aware creation to revalidate after Flush,
// while the originating presentation is still alive on a refusal.
func (a *App) addDocumentChecked(d *document.Document, name, warning string, byteInspector, canReinterpret bool, beforeAdd func() error) *Tab {
	if err := a.admitNewTab(); err != nil {
		a.report(err)
		return nil
	}
	old := a.Current()
	if old != nil {
		if !a.flushTab(old, "打开新标签") {
			return nil
		}
		if beforeAdd != nil {
			if err := beforeAdd(); err != nil {
				a.report(err)
				return nil
			}
		}
	} else if beforeAdd != nil {
		if err := beforeAdd(); err != nil {
			a.report(err)
			return nil
		}
	}
	// Flush can synchronously create another tab through a rich-view callback.
	if err := a.admitNewTab(); err != nil {
		a.report(err)
		return nil
	}
	if old != nil {
		a.invalidateTabExport(old, "打开新标签，已取消导出")
		a.releasePresentation(old)
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
	ed.OnError = a.report
	ed.OnChange = a.noteEdit
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
	a.configureTab(t)
	a.Tabs = append(a.Tabs, t)
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
	tab := a.addDocumentChecked(document.New(adapter.Template()), name, "", false, false, func() error {
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
	})
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
		a.invalidateTabExport(old, "恢复草稿，已取消导出")
		// Recovery can activate a new tab without the ordinary selection path.
		// A readonly scene must be reactivatable when the user returns to it.
		if old.isBinary() {
			a.releasePresentation(old)
		}
		if old.Doc != nil && old.Doc.ReadOnly && old.Adapter != nil && old.Adapter.SceneEngine != nil && old.View != nil {
			old.View.Release()
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
	a.disposePresentation(t)
	t.Doc = d
	a.invalidateDiskChecks()
	t.Editor.Buffer = d.Buffer
	t.Editor.ReadOnly = d.ReadOnly
	t.ByteInspector = false
	t.CanReinterpret = false
	t.Warning = fmt.Sprintf("已按明确选择的 %s 编码打开。", encoding)
	t.Mode = "source"
	a.configureTab(t)
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
				allowed = true
				existing = a.selectOpenDuplicate(path, request)
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
			if candidate, kind, notice := probeImage(path); candidate {
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
				if a.selectOpenDuplicate(path, request) {
					return
				}
				if issueDetail != "" {
					a.report(errors.New(issueDetail))
				}
				beforeAdd := a.reopenAdoptionCheck(request.reopen)
				if t := a.addDocumentChecked(d, filepath.Base(path), warning, inspector, canReinterpret, beforeAdd); t != nil {
					a.acknowledgeReopen(request.reopen)
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
	a.releasePresentation(t)
	t.Mode = "source"
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
	a.TreeList = ui.ListState{}
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
	var err error
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
		err = t.Doc.SaveAs(path)
	} else {
		err = t.Doc.Save()
	}
	if err != nil {
		if errors.Is(err, document.ErrConflict) {
			t.Conflict = true
		}
		a.report(err)
		return false
	}
	t.Name = filepath.Base(t.Doc.Path)
	t.Conflict = false
	t.Editor.ReadOnly = t.Doc.ReadOnly
	a.invalidateDiskChecks()
	if !strings.EqualFold(oldPath, t.Doc.Path) {
		a.invalidateTabExport(t, "文档路径或格式已变化，已取消导出")
		if cancel := a.watches[oldPath]; cancel != nil {
			cancel()
			delete(a.watches, oldPath)
		}
		a.disposePresentation(t)
		t.Adapter = nil
		a.configureTab(t)
		if (oldMode == "preview" || oldMode == "split") && t.Adapter != nil {
			t.Mode = oldMode
		} else {
			t.Mode = "source"
		}
		a.startWatch(t.Doc.Path, false)
	}
	a.Message = "已保存 " + t.Name
	a.Persist()
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
	if !a.commitClose(remaining, active, func() {
		if index < len(a.Tabs) && a.Tabs[index] == t {
			a.removeTab(index)
			removed = !a.tabIsOpen(t)
		}
	}) || !removed {
		return false
	}
	if path != "" {
		a.recordClosedFile(path)
	} else if omission != "" {
		a.Message = omission
	}
	return true
}
func (a *App) prepareClose(t *Tab) (discarded bool, ok bool) {
	if !a.flushTab(t, "关闭标签") {
		return false, false
	}
	if t.isBinary() {
		return false, true
	}
	if t.Doc.Dirty() || t.Doc.IsNew {
		decision, err := a.chooseCloseAction(t)
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
func (a *App) releasePresentation(t *Tab) {
	a.releaseTabPresentation(t, false)
}

func (a *App) releaseTabPresentation(t *Tab, keepBinarySnapshot bool) {
	if t == nil {
		return
	}
	a.cancelPreview(t)
	if keepBinarySnapshot && t.isBinary() {
		t.BinaryGeneration++
		a.cancelBinaryPreview(t)
	} else {
		a.deactivateBinary(t)
	}
	a.releaseOutline()
	t.Editor.UnmarkText()
	t.Editor.ReleaseView()
	if t.View != nil {
		t.View.Release()
	}
}
func (a *App) disposePresentation(t *Tab) {
	if t == nil {
		return
	}
	a.releasePresentation(t)
	t.View = nil
}
func (a *App) removeTab(index int) {
	if index < 0 || index >= len(a.Tabs) {
		return
	}
	a.closeGoToLine()
	a.invalidateFindCache()
	t := a.Tabs[index]
	a.invalidateTabExport(t, "标签已关闭，已取消导出")
	a.disposePresentation(t)
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
	discarded := make([]bool, len(a.Tabs))
	for i, t := range a.Tabs {
		var ok bool
		discarded[i], ok = a.prepareClose(t)
		if !ok {
			return false
		}
	}
	if err := a.persistQuitTabs(a.Tabs, a.Active, discarded); err != nil {
		return false
	}
	if err := a.closeEngineExport(); err != nil {
		a.report(fmt.Errorf("关闭未完成：导出进程清理失败：%w", err))
		// Quit persistence may have recorded explicit discard decisions. Since
		// cleanup refused closing, restore recovery for the still-open live tabs.
		// A second persistence failure is independently reported and retryable.
		if a.persistTabs(a.Tabs, a.Active, "关闭已取消：恢复当前草稿失败") != nil {
			a.markPersistPending()
		}
		return false
	}
	func() {
		a.cancelReopen()
		for len(a.Tabs) > 0 {
			a.removeTab(len(a.Tabs) - 1)
		}
	}()
	return true
}
func (a *App) commitClose(remaining []*Tab, active int, remove func()) bool {
	if err := a.persistTabs(remaining, active, "关闭未完成：无法持久化恢复状态"); err != nil {
		return false
	}
	remove()
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
		if !a.flushTab(t, "重新载入") {
			return
		}
		if t.ByteInspector {
			a.report(errors.New("字节视图不能重新载入为可编辑文本；请重新打开并明确选择编码"))
			return
		}
		if err := t.Doc.Reload(); err != nil {
			a.report(err)
		} else {
			a.invalidateTabExport(t, "文档已重新载入，已取消导出")
			a.invalidateDiskChecks()
			t.Editor.Buffer = t.Doc.Buffer
			a.disposePresentation(t)
			t.Mode = "source"
			a.configureTab(t)
			a.Message = "已重新载入"
		}
	}
}

func (a *App) flushTab(t *Tab, action string) bool {
	if t == nil {
		return true
	}
	if err := t.Editor.Flush(); err != nil {
		a.report(fmt.Errorf("%s前未能完成文本输入：%w", action, err))
		return false
	}
	if t.View == nil {
		return true
	}
	if err := editor.FlushView(t.View); err != nil {
		a.report(fmt.Errorf("%s前未能同步格式编辑：%w", action, err))
		return false
	}
	return true
}
