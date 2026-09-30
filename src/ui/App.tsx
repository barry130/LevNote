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
import type { EditorHandle, EditorPosition } from '../core/editor/types.js';
import type { OutlineItem } from '../core/outline/outline.js';
import { basename, isMarkdownPath } from '../core/path.js';
import { askConfirm, isDesktop } from '../core/platform/index.js';
import { EMPTY_POSITIONS, loadPositions, rememberPosition, savePositions, type PositionMap } from '../core/positions.js';
import {
  clearSession,
  EMPTY_SESSION,
  loadSession,
  sameSession,
  saveSession,
  type SessionState,
  type SessionTab,
} from '../core/session.js';
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
import { ActivityBar } from './ActivityBar.js';
import { Mark } from './brand.js';
import { CommandPalette } from './CommandPalette.js';
import { DocumentPane, type PaneViewInfo } from './DocumentPane.js';
import { FindBar } from './FindBar.js';
import { OutlinePanel } from './OutlinePanel.js';
import { Sidebar } from './Sidebar.js';
import { SettingsDialog } from './SettingsDialog.js';
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
 *
 * ## 「设置」与「命令面板」不在这里
 *
 * 它们搬去了活动栏（见 `ActivityBar.tsx`）。理由不是省地方，而是**分工**：
 * 工具栏放「对文档做什么」（打开 / 保存 / 导出），活动栏放「看哪个面板」。
 * 一个动作有两个常驻入口时，用户会开始猜「它们是不是不一样」。
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

/**
 * 位置记忆写盘的防抖时长。
 *
 * 600ms 的取法：光标停下来半秒多才落盘，人感觉不到延迟；而连续打字时
 * 一次都不写 —— 逐次写就是「每敲一个字一次同步 localStorage 写」，
 * 那会实打实地拖慢打字。
 */
const POSITION_FLUSH_DELAY_MS = 600;

/**
 * 会话写盘的防抖时长。
 *
 * 比位置记忆那 600ms 更长，因为要写的东西重得多：位置一次几十 KB，
 * 会话要写**整篇草稿**（一份两万字的中文文档约 60 KB，utf-16 算 120 KB）。
 * 与位置同一个理由 —— 连续打字时它一直在重置、一次都不写，
 * 只在用户停手之后落盘。
 */
const SESSION_FLUSH_DELAY_MS = 1000;

/**
 * 会话层面「有东西丢了 / 存不下」的提示。
 *
 * 三个字段都是**坏消息**，成功恢复的草稿不在这里 —— 那件事已经有脏标记
 * 与状态栏那枚「有未保存的改动」在说，再来一条横幅就变成每次启动弹一次的噪音。
 */
interface SessionWarning {
  /** 上一次写盘时因超出容量没能存下的草稿数。 */
  readonly droppedDrafts: number;
  /** 启动时打不开的标签页路径（被删 / 被移走 / 没权限）。 */
  readonly failed: readonly string[];
  /** 存储被禁用或写不进去 —— 自动备份这条线断了。 */
  readonly writeFailed: boolean;
}

const NO_SESSION_WARNING: SessionWarning = { droppedDrafts: 0, failed: [], writeFailed: false };

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

  // ---- 会话恢复 ---------------------------------------------------------

  /**
   * 上次的会话（开着哪些标签页、哪一份有未保存的草稿）。
   *
   * 与 `positionsRef` 同一条理由放在 ref 里：它每次落盘都会变，
   * 而屏幕上看不出任何区别 —— 进 state 只会换来一堆无意义的重渲染。
   * `??=` 也是同一条：`useRef(x)` 的参数每次渲染都会求值。
   */
  const sessionRef = useRef<SessionState | null>(null);
  sessionRef.current ??= loadSession(storage);
  const sessionTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  /**
   * 恢复是否已经结束。
   *
   * 它是一道**闸门**，挡的不是「写得太早」而是「写得比读还早」：
   * 启动时 `workspace` 还是空的，而落盘是被防抖定时器驱动的 ——
   * 恢复读完 N 个文件之前那一次写就会把一份**空会话**覆盖上去，
   * 用户的草稿在他自己看到之前就没了。
   */
  const sessionReadyRef = useRef(false);
  /** 恢复中。只用来把空状态换成一句「正在恢复…」，否则启动会闪一下「还没有打开文档」。 */
  const [restoring, setRestoring] = useState(false);

  /**
   * 会话层面丢了什么 / 存不下什么。
   *
   * `droppedDrafts` 在初始化时就取出来：它是**上一次写盘**留下的账目，
   * 而那一次写盘的见证者只有存储里那个数字 —— 读出来之后马上就会
   * 被下一次写盘清零（见 `sameSession` 为什么把 `droppedDrafts` 算进比较）。
   */
  const [sessionWarning, setSessionWarning] = useState<SessionWarning>(() => ({
    ...NO_SESSION_WARNING,
    droppedDrafts: settings.restoreSession ? (sessionRef.current?.droppedDrafts ?? 0) : 0,
  }));
  const sessionWarningVisible =
    sessionWarning.droppedDrafts > 0 || sessionWarning.failed.length > 0 || sessionWarning.writeFailed;

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
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [dragging, setDragging] = useState(false);
  const [liveSidebarWidth, setLiveSidebarWidth] = useState<number | null>(null);

  const activeStore = activeId === null ? null : workspace.storeOf(activeId);
  const activeTab = tabs.find((tab) => tab.id === activeId) ?? null;
  const activePath = activeTab?.path ?? null;

  // ---- 位置记忆 ---------------------------------------------------------

  /**
   * 路径 → 位置。
   *
   * 存在 ref 里而不是 state：它每次光标停顿都会变，进 state 就等于
   * **用户每敲一个字都重渲染整个外壳**（侧栏、标签栏、工具栏全跟着走）。
   * 落盘单独防抖，见 `onPositionChange`。
   *
   * 用 `??=` 而不是 `useRef(loadPositions(storage))`：后者看着只跑一次，
   * 实际上 `useRef` 的参数**每次渲染都会求值**（只是第一次的结果被保留下来），
   * 那会变成每帧一次同步的 localStorage 读取。
   */
  const positionsRef = useRef<PositionMap | null>(null);
  positionsRef.current ??= loadPositions(storage);
  const positionsTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  /**
   * 位置上报时要往哪份文档上记。
   *
   * 用 ref 而不是把它塞进 `onPositionChange` 的依赖：后者一变，
   * 面板那边虽然用 ref 转发了，但「切标签页」本来就会重建面板 ——
   * 没必要再多一次回调标识变化。
   */
  const activePathRef = useRef(activePath);
  activePathRef.current = activePath;

  const rememberPositions = settings.rememberPositions;

  const flushPositions = useCallback(() => {
    if (positionsTimerRef.current !== null) {
      clearTimeout(positionsTimerRef.current);
      positionsTimerRef.current = null;
    }
    savePositions(storage, positionsRef.current ?? EMPTY_POSITIONS);
  }, [storage]);

  const onPositionChange = useCallback(
    (position: EditorPosition) => {
      if (!rememberPositions) return;
      const path = activePathRef.current;
      if (path === null) return;

      positionsRef.current = rememberPosition(positionsRef.current ?? EMPTY_POSITIONS, path, position);

      if (positionsTimerRef.current !== null) clearTimeout(positionsTimerRef.current);
      positionsTimerRef.current = setTimeout(() => {
        positionsTimerRef.current = null;
        savePositions(storage, positionsRef.current ?? EMPTY_POSITIONS);
      }, POSITION_FLUSH_DELAY_MS);
    },
    [rememberPositions, storage],
  );

  /**
   * 关窗口 / 卸载时把还没落盘的那一份补上。
   *
   * 少了这一步，最后 600ms 内的移动会丢 —— 而「改完最后一行就关掉窗口」
   * 恰好是最常见的一种用法。
   */
  useEffect(() => {
    const onBeforeUnload = () => flushPositions();
    window.addEventListener('beforeunload', onBeforeUnload);
    return () => {
      window.removeEventListener('beforeunload', onBeforeUnload);
      flushPositions();
    };
  }, [flushPositions]);

  /**
   * 切文档时立刻落盘，不等防抖。
   *
   * 防抖是给「连续打字」用的；切文档是一个**明确的边界**，边界上就该结算一次。
   * 少了它，用户改完 A 就切到 B、然后进程被杀，A 的位置就没了 ——
   * 而「切走」恰恰是最常发生的一种离开方式，比正常关窗口还常见。
   *
   * 副作用是首屏也会写一次（内容与刚读出来的一样）。那是幂等的，
   * 换来的是这条链路上没有任何「靠定时器兜底」的时序假设。
   */
  useEffect(() => {
    flushPositions();
  }, [activePath, flushPositions]);

  /**
   * 这份文档上次离开时的位置。
   *
   * 依赖只写 `activePath`：`positionsRef` 是 ref，它的变化不会触发重算，
   * 但**也不需要** —— 面板是按 `activeId` 挂载的，切回来时这个 useMemo
   * 会跟着 `activePath` 重新求值，读到的一定是最新的那份。
   */
  const initialPosition = useMemo(
    () => (rememberPositions && activePath !== null ? (positionsRef.current?.[activePath] ?? null) : null),
    [activePath, rememberPositions],
  );

  const clearPositions = useCallback(() => {
    positionsRef.current = EMPTY_POSITIONS;
    savePositions(storage, EMPTY_POSITIONS);
  }, [storage]);

  // ---- 会话（上次开着什么 + 未保存的草稿） ------------------------------

  /**
   * 把当前工作区写进会话存储。
   *
   * 三条硬要求，每一条都对应一种「用户的东西没了」：
   *
   * 1. **恢复完成之前不许写**（`sessionReadyRef`）。见那道闸门的注释。
   * 2. **内容没变就不写**（`sameSession`）。这个函数被「每次按键后的防抖」与
   *    「切标签页」两处驱动，而绝大多数触发时文本并没有变 ——
   *    没有这道比较，用户每切一次标签页都要把几百 KB 的草稿重写一遍。
   * 3. **写不进去要说出来**（`outcome.ok`）。存储被禁用或写满时，
   *    「自动备份」这条线就断了，而用户完全无从察觉 —— 直到他崩一次。
   */
  const flushSession = useCallback(() => {
    if (sessionTimerRef.current !== null) {
      clearTimeout(sessionTimerRef.current);
      sessionTimerRef.current = null;
    }
    if (!sessionReadyRef.current || !settings.restoreSession) return;

    const snapshot = workspace.getSnapshot();
    const previous = sessionRef.current;
    const previousTabs = new Map<string, SessionTab>();
    for (const item of previous?.tabs ?? []) previousTabs.set(item.path, item);

    const now = Date.now();
    const tabs: SessionTab[] = [];
    for (const tab of snapshot.tabs) {
      const doc = workspace.storeOf(tab.id)?.getSnapshot().doc ?? null;
      // 只给**脏**的标签页存草稿：干净标签页的内容磁盘上就有，
      // 存一份等于把整份文件复制进 localStorage，而它一点用都没有
      const draft = tab.dirty && doc !== null ? doc.text : null;
      const before = previousTabs.get(tab.path);
      // `at` 只在草稿真的变了时才推进。每次落盘都刷新一遍的话，
      // 容量裁剪的顺序会退化成「标签页顺序」，最该保住的那一份反而可能先被丢
      const at = before !== undefined && before.draft === draft ? before.at : now;
      tabs.push({ path: tab.path, draft, at });
    }

    const next: SessionState = {
      tabs,
      activePath: snapshot.tabs.find((tab) => tab.id === snapshot.activeId)?.path ?? null,
      root: snapshot.tree.root,
      // 旧账目已经在 `sessionWarning` 里了，这里交 0 让 `saveSession` 把它清掉
      droppedDrafts: 0,
    };
    if (previous !== null && sameSession(previous, next)) return;

    const outcome = saveSession(storage, next);
    // 存回去的是**实际写下去的那一份**（可能被裁剪过），否则下一轮比较会
    // 认为又有变化、反复重写
    sessionRef.current = outcome.state;
    if (!outcome.ok) setSessionWarning((warn) => (warn.writeFailed ? warn : { ...warn, writeFailed: true }));
  }, [settings.restoreSession, storage, workspace]);

  const flushSessionRef = useRef(flushSession);
  flushSessionRef.current = flushSession;

  /**
   * 排一次防抖落盘。
   *
   * 高频事件（每一次按键）走这里，所以它必须**稳定引用**、且只做
   * 「重置一个定时器」这一件事 —— 任何进 state 的动作都会让打字重新变成
   * 「每敲一个字重渲染一次外壳」。
   */
  const scheduleSessionFlush = useCallback(() => {
    if (sessionTimerRef.current !== null) clearTimeout(sessionTimerRef.current);
    sessionTimerRef.current = setTimeout(() => {
      sessionTimerRef.current = null;
      flushSessionRef.current();
    }, SESSION_FLUSH_DELAY_MS);
  }, []);

  /**
   * 启动恢复。**只跑一次。**
   *
   * `restoredRef` 是给 StrictMode 的：开发模式下 effect 会「建立 → 清理 → 再建立」，
   * 少了这道闸会话会被恢复两遍。`restoreTabs` 自己也是幂等的（已开着的路径跳过），
   * 这里多一道只是省掉一轮无意义的异步操作。
   */
  const restoredRef = useRef(false);
  useEffect(() => {
    if (restoredRef.current) return;
    restoredRef.current = true;

    const session = sessionRef.current;
    if (session === null || !settings.restoreSession || (session.tabs.length === 0 && session.root === null)) {
      // 没有会话可恢复（或者用户关掉了恢复）：直接放行落盘
      sessionReadyRef.current = true;
      flushSessionRef.current();
      return;
    }

    void (async () => {
      setRestoring(true);
      try {
        // 先开文件夹再开标签页：`restoreTabs` 最后要把当前文件在树里展开，
        // 顺序反过来的话根目录还没加载，展开会落空
        if (session.root !== null) await workspace.openFolder(session.root);
        const result = await workspace.restoreTabs(session.tabs, session.activePath);
        if (result.failed.length > 0) setSessionWarning((warn) => ({ ...warn, failed: result.failed }));
      } finally {
        setRestoring(false);
        // 闸门打开，然后**立刻**写一次：把 `droppedDrafts` 那笔旧账清掉，
        // 免得用户下一次启动又看到同一条提示
        sessionReadyRef.current = true;
        flushSessionRef.current();
      }
    })();
  }, [settings.restoreSession, workspace]);

  /**
   * 结构变化（开 / 关 / 切 / 拖标签页、脏标记翻转）→ 排一次落盘。
   *
   * `tabs` 是 `useSyncExternalStore` 给的那个数组，它**只在摘要真的变了**时才
   * 换新对象（见 `WorkspaceStore.syncTabs`），所以这个 effect 不会被「每次按键」
   * 惊动 —— 那件事由下面那个文档订阅负责。
   */
  useEffect(() => {
    if (!settings.restoreSession) return;
    scheduleSessionFlush();
  }, [tabs, activePath, settings.restoreSession, scheduleSessionFlush]);

  /**
   * 活动文档的**每一次**状态变化（含每一次按键）→ 排一次落盘。
   *
   * ## 为什么这里可以直接订阅文档 store
   *
   * 「外壳刻意不订阅文档 store」那条规则针对的是 `useSyncExternalStore` ——
   * 那会让每次按键都重渲染整个外壳（侧栏、标签栏、工具栏全跟着走）。
   * 这里用的是**普通订阅**，回调里只重置一个定时器、不碰任何 state，
   * 所以没有任何渲染成本。用普通订阅换来的是「不必给面板再加一个回调 prop」，
   * 而且顺手覆盖了编码 / 行尾这类不改文本的改动。
   *
   * 只订阅**活动**标签页：非活动标签页用户碰不到，改不了内容；
   * 它少数几种会变的情形（重新加载、冲突解决）都会翻转 `dirty`，
   * 已经由上面那个 effect 覆盖了。
   */
  useEffect(() => {
    if (activeStore === null || !settings.restoreSession) return;
    return activeStore.subscribe(scheduleSessionFlush);
  }, [activeStore, settings.restoreSession, scheduleSessionFlush]);

  /**
   * 关窗口 / 卸载时把还没落盘的草稿补上。
   *
   * 与位置记忆那条同一个理由，但代价高得多：位置丢了只是「回到开头」，
   * 草稿丢了是**用户刚写的字没了**。所以这条链路上还有第二道保险 ——
   * 桌面版在 `onCloseRequested` 里也会写一次（见下面那个 effect），
   * 因为 `beforeunload` 在 WebView 里并不保证每次都发得出来。
   */
  useEffect(() => {
    const onBeforeUnload = () => flushSessionRef.current();
    window.addEventListener('beforeunload', onBeforeUnload);
    return () => {
      window.removeEventListener('beforeunload', onBeforeUnload);
      flushSessionRef.current();
    };
  }, []);

  /**
   * 关掉「恢复上次的标签页」= **忘掉**已有的会话记录。
   *
   * 只做「不再写」是不够的：用户关掉它、用一阵子、再打开，会突然恢复出
   * **几天前**那一批标签页与草稿 —— 那不是「恢复上次」，那是从坟里刨东西。
   */
  useEffect(() => {
    if (settings.restoreSession) return;
    clearSession(storage);
    sessionRef.current = EMPTY_SESSION;
  }, [settings.restoreSession, storage]);

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
      openSettings: () => setSettingsOpen(true),
      closeSettings: () => setSettingsOpen(false),
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
        // 只翻模式，不碰内容：三种视图共用 `doc.text` 这一份权威副本，
        // 所以来回切是**无损**的 —— 这也是「外壳不认识内核」带来的好处之一。
        // 判据写成「是不是 rich」而不是「是不是 source」：分屏也属于源码那一侧，
        // 从它切回去的落点应该是富文本，与源码模式同一个出口
        store.setViewMode(snapshot.viewMode === 'rich' ? 'source' : 'rich');
      },
      setViewMode: (mode) => {
        // 没有活动文档时 `DocumentStore.setViewMode` 自己会静默忽略
        // （切视图是纯视图动作，不该在空标签页上报错），这里不必再判一次
        workspace.activeStore?.setViewMode(mode);
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

  /**
   * 某个命令此刻可不可用。
   *
   * 抽出来是因为工具栏与活动栏都要问同一个问题，而**答案必须来自同一个地方**：
   * 命令的 `enabled` 是它的契约（保存要判只读、查找要判有没有文档），
   * 两处各判一次迟早会分叉 —— 而分叉的表现是「工具栏灰着但活动栏能点」。
   */
  const isEnabled = useCallback(
    (id: string) => {
      const command = registry.find(id);
      return command !== undefined && (command.enabled ? command.enabled(context) : true);
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
          // 先落盘再问用户。用户完全可能选「仍要退出」，而那时窗口马上就要销毁，
          // `beforeunload` 不保证发得出来 —— 而这次写盘是草稿的最后一次机会。
          flushSessionRef.current();
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
            const enabled = isEnabled(id);
            // 「源码」是一个**开关**，文字必须反映点了之后去哪 ——
            // 固定写「源码」的话，切过去之后用户会以为再点一次还是切到源码。
            // 分屏算「源码那一侧」，所以它显示的也是「富文本」（同一个出口）
            const text =
              id === 'view.toggleSource' && paneInfo !== null && paneInfo.viewMode !== 'rich' ? '富文本' : label;
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

      {/*
        会话层面的坏消息。**只在真的丢了东西时才出现** ——
        成功恢复的草稿有脏标记与状态栏那枚「有未保存的改动」在说，
        再来一条横幅就变成每次启动都弹一次的噪音，用户很快就不看了。
      */}
      {sessionWarningVisible && (
        <div className={`banner ${sessionWarning.failed.length > 0 || sessionWarning.writeFailed ? 'error' : 'warn'}`}>
          <span className="grow">
            {sessionWarning.failed.length > 0 && (
              <>
                <strong>有 {sessionWarning.failed.length} 个上次开着的文件打不开了</strong>
                （可能已被删除或移走）：{sessionWarning.failed.map((path) => basename(path)).join('、')}。
              </>
            )}
            {sessionWarning.droppedDrafts > 0 && (
              <>
                <strong>有 {sessionWarning.droppedDrafts} 份未保存的草稿没能存下来</strong>
                （会话记录超出了容量上限）。如果那正是你在写的东西，请在退出前手动保存。
              </>
            )}
            {sessionWarning.writeFailed && (
              <>
                <strong>会话记录写不进去</strong>
                （存储被禁用或已写满）。自动备份未保存改动的能力已经失效，请手动保存。
              </>
            )}
          </span>
          <button type="button" onClick={() => setSessionWarning(NO_SESSION_WARNING)}>
            知道了
          </button>
        </div>
      )}

      <div className="body">
        <ActivityBar
          sidebarVisible={settings.sidebarVisible}
          outlineVisible={settings.outlineVisible}
          canOutline={outlineEnabled}
          canFind={isEnabled('find.open')}
          onToggleSidebar={() => runCommand('view.toggleSidebar')}
          onToggleOutline={() => runCommand('view.toggleOutline')}
          onFind={() => runCommand('find.open')}
          onPalette={() => runCommand('palette.open')}
          onSettings={() => runCommand('app.settings')}
        />

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
              codeLineNumbers={settings.codeLineNumbers}
              onToggleCodeLineNumbers={(next) => update({ codeLineNumbers: next })}
              splitRatio={settings.splitRatio}
              onSplitRatioChange={(next) => update({ splitRatio: next })}
              onHandle={onHandle}
              onViewInfoChange={onViewInfoChange}
              initialPosition={initialPosition}
              onPositionChange={onPositionChange}
            />
          ) : restoring ? (
            /*
              恢复中。**不能省**：恢复要逐个读磁盘上的文件，那段时间 `activeId`
              还是 null，不挡住的话用户会先看到一句「还没有打开文档」再看到
              自己的标签页冒出来 —— 像是文件被关掉了又打开。
            */
            <main className="editor-area">
              <div className="empty">
                <Mark size={56} />
                <h2>正在恢复上次的标签页…</h2>
                <p>正在把上次没保存的改动一起读回来。</p>
              </div>
            </main>
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

      {/* 设置页与命令面板同层、排在它之后（同层里最后画的在最上面）。
          两者不会同时开：打开设置要走命令或工具栏，而命令面板打开时会先关掉自己 */}
      {settingsOpen && (
        <SettingsDialog
          settings={settings}
          onChange={update}
          registry={registry}
          platform={platform}
          onClearPositions={clearPositions}
          onClose={() => setSettingsOpen(false)}
        />
      )}

      {dragging && <div className="drop-overlay">松手即可打开</div>}
    </div>
  );
}
