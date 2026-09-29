/**
 * 应用外壳 —— 工作区形态。
 *
 * ## 它与 P2 那版外壳的关系
 *
 * P2 的外壳管「一份文档的完整生命周期」。现在文档生命周期被移进 `DocumentPane`
 * （每个标签页一份，各自订阅自己的 store），外壳只剩三件与「几份文档」有关的事：
 *
 * 1. **工作区状态**：标签页集合、活动标签页、文件树、主题、视图偏好；
 * 2. **命令与快捷键**：把键盘事件翻译成命令（见 `core/commands/`）；
 * 3. **浮层编排**：命令面板、查找栏、大纲的开关与互斥。
 *
 * 这个切分让「加一个标签页」不会碰到任何文档语义的代码，
 * 也让「加一个命令」不会碰到任何布局代码。
 *
 * ## 外壳不认识任何编辑器内核
 *
 * 依赖方向仍然是 `模块 → 契约 ← 外壳`：外壳只从 `DocumentPane` 拿到一个
 * `EditorHandle`，然后用它执行「导出」「查找」这类与内核无关的动作。
 * 想知道当前用的是 Milkdown 还是 CodeMirror，外壳得去问 `adapter.label`。
 */

import { useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore } from 'react';

import type { DocumentProvider } from '../core/document/types.js';
import { registerBuiltInCommands } from '../core/commands/definitions.js';
import { detectPlatform, isEditableTarget, type Platform } from '../core/commands/keybinding.js';
import { CommandRegistry } from '../core/commands/registry.js';
import type { CommandContext, UiActions } from '../core/commands/types.js';
import type { EditorHandle } from '../core/editor/types.js';
import type { OutlineItem } from '../core/outline/outline.js';
import { isMarkdownPath } from '../core/path.js';
import { askConfirm, isDesktop } from '../core/platform/index.js';
import {
  defaultStorage,
  loadSettings,
  saveSettings,
  SIDEBAR_MAX_WIDTH,
  SIDEBAR_MIN_WIDTH,
  type ViewSettings,
} from '../core/settings.js';
import { applyTheme, cycleTheme, resolveTheme, systemPrefersDark, type ThemePreference } from '../core/theme/theme.js';
import type { WorkspaceStore } from '../core/workspace/store.js';
import { registerBuiltInEditors } from '../modules/index.js';
import { Mark } from './brand.js';
import { CommandPalette } from './CommandPalette.js';
import { DocumentPane, type PaneViewInfo } from './DocumentPane.js';
import { FindBar } from './FindBar.js';
import { OutlinePanel } from './OutlinePanel.js';
import { Sidebar } from './Sidebar.js';
import { TabBar } from './TabBar.js';

// 注册表要在任何一次 resolveAdapter 之前填好。放在模块顶层是刻意的：
// 只要这个模块被加载（外壳必然如此），注册就一定发生过。
registerBuiltInEditors();

/**
 * 工具栏按钮。
 *
 * **是精选子集，不是命令面板的镜像**：工具栏只放高频且需要「一眼看到」的动作，
 * 其余交给 Ctrl+P。但每个按钮的**行为**都走命令 ——
 * 于是不可能出现「按钮和命令面板里的同一个功能行为不一致」。
 */
const TOOLBAR: ReadonlyArray<{ id: string; label: string }> = [
  { id: 'file.open', label: '打开' },
  { id: 'file.save', label: '保存' },
  { id: 'file.saveAs', label: '另存为' },
  { id: 'file.exportHtml', label: '导出 HTML' },
  { id: 'file.reload', label: '重新加载' },
  // 视图动作，不是文件动作 —— 但它是最常被点的开关之一，值得占一个位置。
  // 文字由下面的渲染按当前视图模式决定（它是开关，不是按钮）。
  { id: 'view.toggleSource', label: '源码' },
];

export interface AppProps {
  workspace: WorkspaceStore;
  provider: DocumentProvider;
  /** 测试与嵌入场景可以注入初始偏好，跳过 localStorage */
  initialSettings?: ViewSettings;
}

export function App({ workspace, provider, initialSettings }: AppProps) {
  const state = useSyncExternalStore(workspace.subscribe, workspace.getSnapshot);
  const { tabs, activeId, tree, openError } = state;

  const platform = useMemo<Platform>(() => detectPlatform(), []);
  const storage = useMemo(() => defaultStorage(), []);
  const [settings, setSettings] = useState<ViewSettings>(() => initialSettings ?? loadSettings(storage));
  const [systemDark, setSystemDark] = useState(() => systemPrefersDark());

  const [handle, setHandle] = useState<EditorHandle | null>(null);
  const handleRef = useRef<EditorHandle | null>(null);

  /**
   * 活动面板报上来的视图信息。
   *
   * 外壳**刻意不订阅文档 store**（那会让每次按键都重渲染整个外壳），
   * 所以「能不能切源码」「现在在哪一边」这两件事由面板报上来。
   * 没有活动面板时是 null。
   */
  const [paneInfo, setPaneInfo] = useState<PaneViewInfo | null>(null);

  const [paletteOpen, setPaletteOpen] = useState(false);
  const [findOpen, setFindOpen] = useState(false);
  const [dragging, setDragging] = useState(false);
  const [liveSidebarWidth, setLiveSidebarWidth] = useState<number | null>(null);

  const activeStore = activeId === null ? null : workspace.storeOf(activeId);
  const activeTab = tabs.find((tab) => tab.id === activeId) ?? null;
  const activePath = activeTab?.path ?? null;

  // ---- 偏好 ------------------------------------------------------------

  const update = useCallback(
    (patch: Partial<ViewSettings>) => {
      setSettings((previous) => {
        const next = { ...previous, ...patch };
        saveSettings(storage, next);
        return next;
      });
    },
    [storage],
  );

  useEffect(() => {
    // jsdom 没有实现 matchMedia，这里必须容得下它的缺席（测试环境就是这么跑的）
    const probe = typeof window === 'undefined' ? null : (window.matchMedia?.bind(window) ?? null);
    if (!probe) return;
    const query = probe('(prefers-color-scheme: dark)');
    const onChange = (event: MediaQueryListEvent) => setSystemDark(event.matches);
    query.addEventListener('change', onChange);
    return () => query.removeEventListener('change', onChange);
  }, []);

  useEffect(() => {
    applyTheme(resolveTheme(settings.theme, systemDark), document.documentElement);
  }, [settings.theme, systemDark]);

  // ---- 编辑器命令面 -----------------------------------------------------

  const onHandle = useCallback((next: EditorHandle | null) => {
    handleRef.current = next;
    setHandle(next);
  }, []);

  const onViewInfoChange = useCallback((info: PaneViewInfo | null) => {
    setPaneInfo(info);
  }, []);

  const closeActiveTab = useCallback(() => {
    void (async () => {
      const current = workspace.activeTab;
      if (current === null) return;
      if (current.dirty) {
        const leave = await askConfirm(`${current.displayName} 有未保存的改动，仍要关闭吗？`);
        if (!leave) return;
      }
      workspace.closeActive(true);
    })();
  }, [workspace]);

  // ---- 命令上下文 -------------------------------------------------------

  const ui = useMemo<UiActions>(
    () => ({
      toggleSidebar: () => update({ sidebarVisible: !settings.sidebarVisible }),
      toggleOutline: () => update({ outlineVisible: !settings.outlineVisible }),
      openPalette: () => setPaletteOpen(true),
      closePalette: () => setPaletteOpen(false),
      openFind: () => setFindOpen(true),
      closeFind: () => setFindOpen(false),
      setTheme: (preference: ThemePreference) => update({ theme: preference }),
      cycleTheme: () => update({ theme: cycleTheme(settings.theme) }),
      editor: () => handleRef.current,
      // 「能不能切」问的是**首选内核**有没有声明源码视图，而不是路径后缀 ——
      // 后缀判断会漏掉「富文本内核起不来、已退回兜底内核」这种情况
      canToggleSourceView: () => paneInfo?.adapter?.sourceView !== undefined,
      toggleSourceView: () => {
        const store = workspace.activeStore;
        if (store === null) return;
        const snapshot = store.getSnapshot();
        if (snapshot.doc === null) return;
        // 只翻模式，不碰内容：两种视图共用 `doc.text` 这一份权威副本，
        // 所以来回切是**无损**的 —— 这也是「外壳不认识内核」带来的好处之一
        store.setViewMode(snapshot.viewMode === 'source' ? 'rich' : 'source');
      },
      notify: (message, title) => provider.notify(message, title),
      closeActiveTab,
    }),
    [closeActiveTab, paneInfo, provider, settings.outlineVisible, settings.sidebarVisible, settings.theme, update, workspace],
  );

  const context = useMemo<CommandContext>(() => ({ workspace, provider, ui }), [workspace, provider, ui]);

  const registry = useMemo(() => {
    const created = new CommandRegistry();
    registerBuiltInCommands(created);
    return created;
  }, []);

  const runCommand = useCallback(
    (id: string) => {
      const command = registry.find(id);
      if (!command) return;
      if (command.enabled && !command.enabled(context)) return;
      void Promise.resolve(command.run(context)).catch((thrown: unknown) => {
        console.warn(`命令 ${id} 执行失败`, thrown);
      });
    },
    [registry, context],
  );

  // ---- 全局快捷键 -------------------------------------------------------

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const command = registry.resolve(event, platform, isEditableTarget(event.target));
      if (!command) return;
      // 可用性判断在这里也执行一次：`enabled` 是命令的契约，
      // 不能只在命令面板里生效，否则「保存」在只读文件上仍会被 Ctrl+S 触发
      if (command.enabled && !command.enabled(context)) return;
      event.preventDefault();
      void Promise.resolve(command.run(context)).catch((thrown: unknown) => {
        console.warn(`命令 ${command.id} 执行失败`, thrown);
      });
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [registry, platform, context]);

  // ---- 外部改动：窗口重新获得焦点时比对 mtime ---------------------------

  useEffect(() => {
    const onFocus = () => {
      void workspace.checkExternalChanges();
    };
    window.addEventListener('focus', onFocus);
    return () => window.removeEventListener('focus', onFocus);
  }, [workspace]);

  // ---- 拖入文件 / 目录打开，有未保存改动时拦下关闭 ----------------------

  useEffect(() => {
    if (!isDesktop()) return;

    let disposed = false;
    const cleanups: Array<() => void> = [];

    void (async () => {
      const [{ getCurrentWebview }, { getCurrentWindow }] = await Promise.all([
        import('@tauri-apps/api/webview'),
        import('@tauri-apps/api/window'),
      ]);
      if (disposed) return;

      cleanups.push(
        await getCurrentWebview().onDragDropEvent((event) => {
          if (event.payload.type === 'over' || event.payload.type === 'enter') {
            setDragging(true);
          } else if (event.payload.type === 'drop') {
            setDragging(false);
            const first = event.payload.paths?.[0];
            if (first) void workspace.open(first);
          } else {
            setDragging(false);
          }
        }),
      );

      cleanups.push(
        await getCurrentWindow().onCloseRequested(async (event) => {
          const dirty = workspace.getSnapshot().tabs.filter((tab) => tab.dirty);
          if (dirty.length === 0) return;
          const names = dirty.map((tab) => tab.displayName).join('、');
          const leave = await askConfirm(`${names} 有未保存的改动，仍要退出吗？`);
          if (!leave) event.preventDefault();
        }),
      );
    })();

    return () => {
      disposed = true;
      for (const cleanup of cleanups) cleanup();
    };
  }, [workspace]);

  // ---- 窗口标题 ---------------------------------------------------------

  useEffect(() => {
    const title = activeTab ? `${activeTab.dirty ? '• ' : ''}${activeTab.displayName} — 轻墨` : '轻墨';
    document.title = title;
    if (!isDesktop()) return;
    void import('@tauri-apps/api/window')
      .then(({ getCurrentWindow }) => getCurrentWindow().setTitle(title))
      .catch(() => undefined);
  }, [activeTab]);

  // ---- 侧栏宽度拖拽 -----------------------------------------------------

  const beginResize = useCallback(
    (event: React.PointerEvent<HTMLDivElement>) => {
      event.preventDefault();
      const startX = event.clientX;
      const startWidth = settings.sidebarWidth;

      const onMove = (move: PointerEvent) => {
        const next = Math.min(SIDEBAR_MAX_WIDTH, Math.max(SIDEBAR_MIN_WIDTH, startWidth + (move.clientX - startX)));
        setLiveSidebarWidth(next);
      };
      const onUp = () => {
        window.removeEventListener('pointermove', onMove);
        window.removeEventListener('pointerup', onUp);
        // 拖动过程中只改本地状态，松手才落盘 ——
        // 每次 pointermove 都写一次 localStorage 会拖慢拖拽手感
        setLiveSidebarWidth((width) => {
          if (width !== null) update({ sidebarWidth: width });
          return null;
        });
      };
      window.addEventListener('pointermove', onMove);
      window.addEventListener('pointerup', onUp);
    },
    [settings.sidebarWidth, update],
  );

  // ---- 大纲跳转 ---------------------------------------------------------

  const jumpToOutline = useCallback((item: OutlineItem) => {
    const current = handleRef.current;
    if (!current?.goToOutlineItem) return;
    current.goToOutlineItem({ level: item.level, text: item.text });
  }, []);

  /**
   * 只有 Markdown 家族的文档才显示大纲。
   *
   * 判断放在外壳而不是内核：这是**产品决策**（哪些文件值得有大纲），
   * 不是内核能力。内核只需要实现 `goToOutlineItem`。
   *
   * 名单来自 `core/path.ts` —— 外壳**可以**知道「哪些文件是 Markdown」，
   * 因为那是产品范围；它**不可以**知道「Markdown 用哪个内核渲染」，
   * 那才是内核知识。两者很容易被混为一谈。
   */
  const outlineEnabled = useMemo(() => activePath !== null && isMarkdownPath(activePath), [activePath]);

  const sidebarWidth = liveSidebarWidth ?? settings.sidebarWidth;

  return (
    <div className="shell">
      <header className="titlebar">
        <span className="brand">
          <Mark size={18} />
          轻墨
        </span>

        <span className="workspace-name" title={tree.root ?? ''}>
          {tree.root === null ? '未打开文件夹' : tree.root}
        </span>

        <span className="spacer" />

        <span
          className="badge"
          title={provider.kind === 'desktop' ? '桌面版：直接读写磁盘' : '浏览器预览：内存虚拟盘，保存会触发下载'}
        >
          {provider.kind === 'desktop' ? '桌面版' : '浏览器预览'}
        </span>

        <span className="toolbar">
          {TOOLBAR.map(({ id, label }) => {
            const command = registry.find(id);
            const enabled = command !== undefined && (command.enabled ? command.enabled(context) : true);
            // 「源码」是一个**开关**，文字必须反映当前在哪一边 ——
            // 固定写「源码」的话，切过去之后用户会以为再点一次还是切到源码
            const text = id === 'view.toggleSource' && paneInfo?.viewMode === 'source' ? '富文本' : label;
            return (
              <button
                key={id}
                type="button"
                className={id === 'file.save' ? 'primary' : ''}
                disabled={!enabled}
                title={command?.title ?? label}
                onClick={() => runCommand(id)}
              >
                {text}
              </button>
            );
          })}
          <button type="button" onClick={() => setPaletteOpen(true)} title="命令面板（Ctrl+P）">
            命令
          </button>
        </span>
      </header>

      <TabBar
        tabs={tabs}
        activeId={activeId}
        onActivate={(id) => workspace.activate(id)}
        onClose={(id) => {
          const tab = tabs.find((item) => item.id === id);
          if (tab === undefined) return;
          if (!tab.dirty) {
            workspace.closeTab(id, true);
            return;
          }
          void (async () => {
            const leave = await askConfirm(`${tab.displayName} 有未保存的改动，仍要关闭吗？`);
            if (leave) workspace.closeTab(id, true);
          })();
        }}
        onReorder={(from, to) => workspace.moveTab(from, to)}
      />

      {openError && (
        <div className="banner error">
          <span className="grow">
            <strong>{openError.code}</strong> · {openError.message}
            {openError.detail ? `（${openError.detail}）` : ''}
          </span>
          <button type="button" onClick={workspace.clearOpenError}>
            知道了
          </button>
        </div>
      )}

      <div className="body">
        {settings.sidebarVisible && (
          <>
            <div className="sidebar-slot" style={{ width: sidebarWidth }}>
              <Sidebar
                tree={tree}
                activePath={activePath}
                onOpenFolder={() => void workspace.openFolder()}
                onToggleDir={(path) => void workspace.toggleDir(path)}
                onOpenFile={(path) => void workspace.open(path)}
                onRefresh={() => void workspace.refreshTree()}
                onToggleIgnored={() => workspace.setShowIgnored(!tree.showIgnored)}
              />
            </div>
            <div className="sidebar-resizer" onPointerDown={beginResize} role="separator" aria-orientation="vertical" />
          </>
        )}

        <div className="workspace-main">
          {activeStore !== null ? (
            <DocumentPane
              // 换标签页就换一个面板实例：内核与文档是一对一的，
              // 让它们跟着标签页走比在同一个面板里换内核安全得多
              key={activeId}
              store={activeStore}
              wordWrap={settings.wordWrap}
              onHandle={onHandle}
              onViewInfoChange={onViewInfoChange}
            />
          ) : (
            <main className="editor-area">
              <div className="empty">
                <Mark size={56} />
                <h2>还没有打开文档</h2>
                <p>
                  按 <kbd>Ctrl</kbd> + <kbd>O</kbd> 打开文件，按 <kbd>Ctrl</kbd> + <kbd>Shift</kbd> + <kbd>O</kbd>{' '}
                  打开文件夹，或把文件直接拖进这个窗口。
                </p>
                <p className="hint">
                  轻墨支持 Markdown、代码 / 文本、思维导图（.mindmap）与 draw.io 图表（.drawio）四类文档，它们可以互相引用。
                </p>
              </div>
            </main>
          )}

          {findOpen && activeStore !== null && (
            <FindBar key={`find-${activeId}`} handle={handle} onClose={() => setFindOpen(false)} />
          )}
        </div>

        {settings.outlineVisible && outlineEnabled && (
          <OutlinePanel
            store={activeStore}
            canJump={typeof handle?.goToOutlineItem === 'function'}
            onJump={jumpToOutline}
            onClose={() => update({ outlineVisible: false })}
          />
        )}
      </div>

      {paletteOpen && (
        <CommandPalette registry={registry} context={context} platform={platform} onClose={() => setPaletteOpen(false)} />
      )}

      {dragging && <div className="drop-overlay">松手即可打开</div>}
    </div>
  );
}
