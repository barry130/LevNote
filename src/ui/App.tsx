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
import { buildDocBlockPreview } from '../core/document/block-preview.js';
import { registerBuiltInCommands } from '../core/commands/definitions.js';
import { detectPlatform, isEditableTarget, type Platform } from '../core/commands/keybinding.js';
import { CommandRegistry } from '../core/commands/registry.js';
import type { CommandContext, UiActions } from '../core/commands/types.js';
import type { DocBlockPreview, EditorHandle, EditorPosition } from '../core/editor/types.js';
import type { OutlineItem } from '../core/outline/outline.js';
import { basename, isMarkdownPath, resolveRelative } from '../core/path.js';
import type { DocumentStore } from '../core/document/store.js';
import { askConfirm, createWorkspaceSearcher, isDesktop, onFilesChanged, toggleFullscreen, watchFiles } from '../core/platform/index.js';
import { isInside } from '../core/path.js';
import { EMPTY_POSITIONS, loadPositions, rememberPosition, savePositions, type PositionMap } from '../core/positions.js';
import { WorkspaceSearchStore, type WorkspaceMatch, type WorkspaceSearcher } from '../core/search/workspace.js';
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
import type { TabSummary, WorkspaceStore } from '../core/workspace/store.js';
import {
  editorRegistrationFailures,
  registerBuiltInEditors,
  type KernelLoadFailure,
} from '../modules/index.js';
import { ActivityBar } from './ActivityBar.js';
import { Mark } from './brand.js';
import { CommandPalette } from './CommandPalette.js';
import { DocumentPane, type PaneRevealRequest, type PaneViewInfo } from './DocumentPane.js';
import { FindBar } from './FindBar.js';
import { MenuBar } from './MenuBar.js';
import { OutlinePanel } from './OutlinePanel.js';
import { SearchPanel } from './SearchPanel.js';
import { Sidebar } from './Sidebar.js';
import { SettingsDialog } from './SettingsDialog.js';
import { TabBar } from './TabBar.js';

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
  /**
   * 工作区检索的执行者。
   *
   * 与 `provider` 一样由宿主注入，而不是在这里调平台 —— 理由不是「方便测试」，
   * 而是**入口只有一个**：`main.tsx` 造 provider 与 workspace，检索执行者
   * 属于同一类东西。默认值只是给嵌入场景留的便利。
   */
  searcher?: WorkspaceSearcher;
}

/**
 * 一个常驻面板的槽位：把 `DocumentPane` 包起来，负责三件「每个面板各自」的事。
 *
 * 1. **隐藏**：非活动面板整槽 `display: none`。用行内样式而不是类，
 *    因为外壳样式表里 `.pane-slot` 是 `display: contents`（让面板的横幅与
 *    编辑区直接成为 `workspace-main` 的 flex 成员，DOM 层级与单面板时代一致），
 *    行内样式压过它只改这一处，不新开一个「谁赢了看打包顺序」的战场（硬约束 13）。
 * 2. **按面板分桶的汇报**：`onHandle` / `onViewInfoChange` 必须是**按槽位稳定**
 *    的引用 —— `DocumentPane` 把它们挂在 effect 依赖里（卸载清理靠它们上报 null），
 *    引用一变就会反复触发「上报 null」。所以在这里用 `useCallback` 固定，
 *    带上 `tab.id` 让外壳能分清是哪个面板在说话。
 * 3. **位置记忆逐面板取**：`initialPosition` 是这份文档自己的位置，
 *    不是「当前激活文档」的 —— 挂载发生在首次激活那一刻，两者天然一致。
 */
interface PaneSlotProps {
  tab: TabSummary;
  active: boolean;
  store: DocumentStore;
  wordWrap: boolean;
  codeLineNumbers: boolean;
  onToggleCodeLineNumbers: (next: boolean) => void;
  splitRatio: number;
  onSplitRatioChange: (next: number) => void;
  initialPosition: EditorPosition | null;
  revealRequest: PaneRevealRequest | null;
  onRevealResult: (tick: number, applied: boolean) => void;
  onDocBlockOpen: (ref: { src: string; kind: string; title: string | null }) => void;
  onDocBlockEditInPlace: (ref: { src: string; kind: string; title: string | null }) => void;
  resolveDocBlockPreview: (ref: { src: string; kind: string }) => Promise<DocBlockPreview | null>;
  reportHandle: (paneId: number, handle: EditorHandle | null) => void;
  reportViewInfo: (paneId: number, info: PaneViewInfo | null) => void;
  rememberPositionAt: (path: string, position: EditorPosition) => void;
}

function PaneSlot({
  tab,
  active,
  store,
  wordWrap,
  codeLineNumbers,
  onToggleCodeLineNumbers,
  splitRatio,
  onSplitRatioChange,
  initialPosition,
  revealRequest,
  onRevealResult,
  onDocBlockOpen,
  onDocBlockEditInPlace,
  resolveDocBlockPreview,
  reportHandle,
  reportViewInfo,
  rememberPositionAt,
}: PaneSlotProps) {
  const onHandle = useCallback((handle: EditorHandle | null) => reportHandle(tab.id, handle), [reportHandle, tab.id]);
  const onViewInfoChange = useCallback(
    (info: PaneViewInfo | null) => reportViewInfo(tab.id, info),
    [reportViewInfo, tab.id],
  );
  const onPositionChange = useCallback(
    (position: EditorPosition) => rememberPositionAt(tab.path, position),
    [rememberPositionAt, tab.path],
  );
  return (
    <div className="pane-slot" style={active ? undefined : { display: 'none' }}>
      <DocumentPane
        store={store}
        active={active}
        wordWrap={wordWrap}
        codeLineNumbers={codeLineNumbers}
        onToggleCodeLineNumbers={onToggleCodeLineNumbers}
        splitRatio={splitRatio}
        onSplitRatioChange={onSplitRatioChange}
        onHandle={onHandle}
        onViewInfoChange={onViewInfoChange}
        initialPosition={initialPosition}
        onPositionChange={onPositionChange}
        revealRequest={revealRequest}
        onRevealResult={onRevealResult}
        onDocBlockOpen={onDocBlockOpen}
        onDocBlockEditInPlace={onDocBlockEditInPlace}
        resolveDocBlockPreview={resolveDocBlockPreview}
      />
    </div>
  );
}

export function App({ workspace, provider, initialSettings, searcher: injectedSearcher }: AppProps) {
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

  const activeStore = activeId === null ? null : workspace.storeOf(activeId);
  const activeTab = tabs.find((tab) => tab.id === activeId) ?? null;
  const activePath = activeTab?.path ?? null;
  /**
   * 「就地编辑」的返回目标（P6 第 28 项 L2，ADR 0036）：当前标签页是从哪个
   * 标签页钻过来的。来路已被关掉时不显示 chip（store 会在 closeTab 时清理，
   * 这里再兜一层）。
   */
  const drillBackTo = (() => {
    const fromId = activeTab?.drilledFrom ?? null;
    if (fromId === null) return null;
    const from = tabs.find((tab) => tab.id === fromId);
    return from === undefined ? null : { id: fromId, name: from.displayName };
  })();

  // ---- 面板常驻（P3 第 19 项）-------------------------------------------

  /**
   * 每个面板的句柄与视图信息，按标签 id 分桶。
   *
   * 面板常驻之后外壳里同时挂着 N 份内核，而命令面（保存 / 查找 / 格式化）
   * 与「能不能切源码」只应该属于**活动的那一份**。每个面板都经
   * `reportHandle` / `reportViewInfo` 汇报：先进桶，再只有活动面板那一份
   * 进入外壳状态。少了这道分发，「关闭一个后台标签」的卸载清理
   * （面板卸载时上报 null）会把活动面板的句柄一起清掉 —— 查找栏当场失灵。
   */
  const handlesRef = useRef(new Map<number, EditorHandle>());
  const infosRef = useRef(new Map<number, PaneViewInfo>());
  /** 汇报回调是稳定引用，读「现在谁活动」走 ref（与 `revealRequestRef` 同一条理由）。 */
  const activeIdRef = useRef(activeId);
  activeIdRef.current = activeId;

  /**
   * 已经访问过的标签 —— 这些标签的面板**常驻**（P3 第 19 项）。
   *
   * 面板常驻 = 切走的标签不再卸载，内核（撤销栈、光标、滚动）原样保留，
   * 切回来就是离开时的样子。为什么不全部挂载：会话恢复一次带出十个标签的话，
   * 启动就要付十份内核创建的成本；「访问过才常驻」让启动只付活动标签那一份，
   * 而撤销栈本来就只可能在访问过的标签里积累。
   *
   * 内存实测（2026-10-05，见 ADR 0034）：空闲内核每实例 Markdown ≈ 0.27 MB、
   * CodeMirror ≈ 1.9 MB（V8 堆），几十个标签在可接受范围 —— 这是「敢常驻」的依据。
   */
  const [visitedIds, setVisitedIds] = useState<ReadonlySet<number>>(() => new Set());
  if (activeId !== null && !visitedIds.has(activeId)) {
    // 渲染期更新（React 官方认可的「从 props/state 派生」写法）：
    // 必须与 activeId 在同一次提交里生效，否则「切到一个没去过的标签」的
    // 第一帧会因为没有面板可渲染而闪一下空状态，面板挂载也被推迟一帧。
    setVisitedIds((previous) => {
      if (previous.has(activeId)) return previous;
      const next = new Set(previous);
      next.add(activeId);
      return next;
    });
  }

  const reportHandle = useCallback((paneId: number, next: EditorHandle | null) => {
    if (next === null) handlesRef.current.delete(paneId);
    else handlesRef.current.set(paneId, next);
    if (paneId !== activeIdRef.current) return;
    handleRef.current = next;
    setHandle(next);
  }, []);

  const reportViewInfo = useCallback((paneId: number, info: PaneViewInfo | null) => {
    if (info === null) infosRef.current.delete(paneId);
    else infosRef.current.set(paneId, info);
    if (paneId !== activeIdRef.current) return;
    setPaneInfo(info);
  }, []);

  // 激活标签变化时，把命令面与视图信息换成新活动面板桶里那一份。
  // 常驻面板被重新激活时内核没有重建、**不会重新汇报** —— 必须主动来取。
  useEffect(() => {
    const next = activeId === null ? null : (handlesRef.current.get(activeId) ?? null);
    handleRef.current = next;
    setHandle(next);
    setPaneInfo(activeId === null ? null : (infosRef.current.get(activeId) ?? null));
  }, [activeId]);

  // ---- 工作区检索 -------------------------------------------------------

  /**
   * 检索会话。
   *
   * 它自己管三件时序事（防抖、取消、丢弃过期批次），外壳只负责把根目录喂给它、
   * 把点击结果翻译成「打开文件 + 定位」。理由见 `core/search/workspace.ts`。
   *
   * 浏览器预览拿到的是**明确拒绝**的执行者 —— 那里没有真实目录，
   * 造一个「能搜」的假象只会让「预览里搜不到」变成查不出来的谜。
   *
   * 「能不能检索」问的是**执行者自己**（`searcher.available`），不是平台。
   * 另问一次平台就是同一个问题两份答案，将来加一个平台就会得到
   * 「按钮亮着、点下去被拒绝」。
   */
  const fallbackSearcher = useMemo(() => createWorkspaceSearcher(), []);
  const searcher = injectedSearcher ?? fallbackSearcher;
  const searchStore = useMemo(() => new WorkspaceSearchStore(searcher), [searcher]);
  const canSearchFolder = searcher.available;

  // 根目录跟着工作区走。**必须跟着**：换了文件夹还留着上一次的结果，
  // 用户会点进去、然后打开一个跟他要找的东西无关的文件
  useEffect(() => {
    searchStore.setRoot(tree.root);
  }, [searchStore, tree.root]);

  useEffect(() => () => searchStore.dispose(), [searchStore]);

  /**
   * 内核加载失败是**降级**，降级不许静默（硬约束 2）：少了哪个内核、为什么，
   * 要在界面上说出来。注册由 main.tsx 在首帧前 await，这里再问一次拿到的是
   * 同一个已完成的 Promise，所以这个 effect 实际是「启动后读一次结果」。
   * 只在有失败时 setState —— 空数组的渲染结果相同，不值得惊动 React。
   */
  useEffect(() => {
    void registerBuiltInEditors().then(() => {
      const failed = editorRegistrationFailures();
      if (failed.length > 0) setKernelFailures(failed);
    });
  }, []);

  /**
   * 待定位的检索结果，连同它的编号。
   *
   * ## 为什么它进 state、并且交给面板在内核就绪时落地
   *
   * 点一条结果要做三件事：打开文件 → 等内核就绪 → 定位。中间隔着一次
   * **异步的**打开与至少一次 React 渲染，所以这个意图必须能活过渲染 ——
   * 这决定了它必须住在面板**之上**（state），而不是一次性调用。
   *
   * 而「落地」必须发生在**内核就绪那一刻** —— 这与 `initialPosition`
   * （位置记忆）是同一个时刻。内核是异步创建的，「外壳握着句柄戳一次」
   * 的写法赶不上「内核刚就绪」与「面板重建」谁先谁后：StrictMode 的
   * 模拟重挂（开发模式）实测踩中过 —— 定位刚落地、面板随后重挂，
   * 选区与状态栏的光标读数一起被销毁，而外壳这边早已把待办清掉，
   * 再也没有人补那一次定位。
   * 把请求作为 prop 传给面板（`revealRequest`）、由它在 `handleEditorReady`
   * 里应用，**任何一次内核重建都会重新落地**，这条竞态从设计上消失。
   *
   * ## 请求为什么在 `open()` 成功**之后**才立起来
   *
   * 那一刻 `activePath` 已经是目标文件，「请求是给谁的」与「现在看着谁」
   * 天然一致，下面那个对账 effect 不需要区分「正在赶往目标」与「用户去了别处」。
   * 打开失败（文件被删了）时不立请求，也不会留下一个永远落不了地的待办。
   */
  const [revealRequest, setRevealRequest] = useState<PaneRevealRequest | null>(null);
  const revealTickRef = useRef(0);
  /**
   * 已经为哪些请求**切过视图**。
   *
   * 防死循环：富文本内核（Markdown / 导图）没有源码行列的概念，
   * 只能回落到源码视图；回落会重建内核，不加这个标记的话，
   * 一个定位不到的结果会让视图在富文本与源码之间来回弹。
   */
  const switchedRevealTicksRef = useRef(new Set<number>());
  /** `handleRevealResult` 里读当前请求用的镜像，与位置记忆的 `positionsRef` 同一条理由。 */
  const revealRequestRef = useRef<PaneRevealRequest | null>(null);
  revealRequestRef.current = revealRequest;

  const openSearchMatch = useCallback(
    (path: string, match: WorkspaceMatch) => {
      const tick = (revealTickRef.current += 1);
      void (async () => {
        const opened = await workspace.open(path);
        if (opened) {
          setRevealRequest({
            path,
            location: { line: match.line, column: match.column, length: match.length },
            tick,
          });
        }
      })();
    },
    [workspace],
  );

  /**
   * 把引用卡片交来的 `src` 换算成**绝对路径**（P6 第 28 项）。
   *
   * 「打开」与「就地编辑」共用这一步。点击只会发生在**活动面板**上
   * （隐藏面板收不到点击），所以用 `workspace.activeStore` 现读文档路径，
   * 不存在过期闭包的问题。`resolveRelative` 把 src 的正斜杠统一成
   * 文档路径的分隔符风格（硬约束 12）—— 写入时用正斜杠、打开时按字符串
   * 换算，两份字符串必须能对上。
   *
   * 光秃秃的文件名（没有目录信息）拼不出被引用文件的路径 —— 与
   * `resolveDocBlock` 同一条纪律：不能猜，但要说出来，不能点了没反应。
   */
  const resolveDocBlockSrc = useCallback(
    (ref: { src: string }): string | null => {
      const docPath = workspace.activeStore?.getSnapshot().doc?.path ?? null;
      const absolute = docPath === null ? null : resolveRelative(docPath, ref.src);
      if (absolute === null) {
        void provider.notify(`无法确定 ${ref.src} 的位置 —— 这份文档没有可用的目录信息。`, '打开引用');
      }
      return absolute;
    },
    [provider, workspace],
  );

  /** 打开一张引用卡片指向的文档（L1：新标签页）。 */
  const openDocBlockReference = useCallback(
    (ref: { src: string; kind: string; title: string | null }) => {
      const absolute = resolveDocBlockSrc(ref);
      if (absolute === null) return;
      void workspace.open(absolute);
    },
    [resolveDocBlockSrc, workspace],
  );

  /**
   * 就地编辑（L2，ADR 0036）：同面板钻取 —— 被引用文档走**正常文档路径**
   * 打开（自己的 store、脏标记、保存、文件监听），只是把来路（当前标签页）
   * 记在目标标签页上，标签栏出现「↩ 返回」chip；返回时 P3-19 的面板常驻
   * 保证光标 / 滚动 / 撤销栈原样。
   */
  const editDocBlockInPlace = useCallback(
    (ref: { src: string; kind: string; title: string | null }) => {
      const fromId = workspace.getSnapshot().activeId;
      const absolute = resolveDocBlockSrc(ref);
      if (absolute === null || fromId === null) return;
      void workspace.openInPlace(fromId, absolute);
    },
    [resolveDocBlockSrc, workspace],
  );

  /**
   * 卡片「预览」展开要的结构摘要（ADR 0036 决策 3）。
   *
   * 读文件归外壳（provider 可能抛错，这里统一兜成 null）；计算复用
   * `block-preview.ts`（依赖注入可单测）。宿主文档路径与 `resolveDocBlockSrc`
   * 同一条理由现读 —— 摘要请求只会从活动面板的卡片上发起。
   */
  const resolveDocBlockPreview = useCallback(
    (ref: { src: string; kind: string }) => {
      const docPath = workspace.activeStore?.getSnapshot().doc?.path ?? null;
      if (docPath === null) {
        return Promise.resolve({ kind: 'error' as const, reason: '无法确定这份文档的位置' });
      }
      const readTextFile = async (path: string) => {
        try {
          return await provider.readTextFile(path);
        } catch {
          return null;
        }
      };
      return buildDocBlockPreview({ readTextFile }, docPath, ref);
    },
    [provider, workspace],
  );

  /** 沿「就地编辑」的来路返回（chip 点击）。 */
  const goBackFromDrill = useCallback(() => {
    const current = workspace.getSnapshot().activeId;
    if (current !== null) workspace.goBackDrill(current);
  }, [workspace]);

  /**
   * 面板回报「这次定位落没落地」。
   *
   * ## 回落链
   *
   * ① 当前内核实现了 `revealSourceLocation` → 面板直接落地（applied = true）；
   * ② 没有但这份文档有源码视图 → 切到源码视图，下一个内核就绪时面板会再试（只切一次）；
   * ③ 都没有 → 放弃。**不能留着这个待办**：留着的下一次点击会先消费掉它，
   *    表现是「点 A 没反应，点 B 跳到了 A」。
   */
  const handleRevealResult = useCallback(
    (tick: number, applied: boolean) => {
      if (applied) return;
      const pending = revealRequestRef.current;
      if (pending === null || pending.tick !== tick) return;
      if (!switchedRevealTicksRef.current.has(tick) && paneInfo?.adapter?.sourceView !== undefined) {
        switchedRevealTicksRef.current.add(tick);
        workspace.activeStore?.setViewMode('source');
        return;
      }
      setRevealRequest(null);
    },
    [paneInfo, workspace],
  );

  // 用户把标签页切走时收掉待办：它属于「跳到那个文件的那一行」这件事，
  // 用户已经去了别处，留着只会在他切回来时跳一次 —— 那不是他要的。
  // 请求立起的那一刻 activePath 已经是目标文件（见 openSearchMatch），
  // 所以这里不需要区分「正在赶往目标」的窗口期。
  useEffect(() => {
    if (revealRequest !== null && activePath !== null && activePath !== revealRequest.path) {
      setRevealRequest(null);
    }
  }, [activePath, revealRequest]);

  const [paletteOpen, setPaletteOpen] = useState(false);
  const [findOpen, setFindOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  /** 启动时没能加载的内核（P3 第 18 项）。空数组 = 全部就位，横幅不出现。 */
  const [kernelFailures, setKernelFailures] = useState<readonly KernelLoadFailure[]>([]);
  const [dragging, setDragging] = useState(false);
  const [liveSidebarWidth, setLiveSidebarWidth] = useState<number | null>(null);

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

  const rememberPositions = settings.rememberPositions;

  const flushPositions = useCallback(() => {
    if (positionsTimerRef.current !== null) {
      clearTimeout(positionsTimerRef.current);
      positionsTimerRef.current = null;
    }
    savePositions(storage, positionsRef.current ?? EMPTY_POSITIONS);
  }, [storage]);

  /**
   * 把一份位置记到**指定的路径**上。
   *
   * 路径由调用方（面板槽位）给出，而不是在这里读「当前激活的文档」：
   * 面板常驻之后（P3 第 19 项），隐藏面板在极端情况下也会上报位置
   * （比如外部改写触发内核重建选区），按「谁上报记谁」才不会把
   * 隐藏文档的位置写到活动文档头上。
   */
  const rememberPositionAt = useCallback(
    (path: string, position: EditorPosition) => {
      if (!rememberPositions) return;

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
   * 「上次读到哪儿」现在由 `PaneSlot` 逐面板去取（`positionsRef.current?.[tab.path]`），
   * 不再有一个全局的 `initialPosition`：面板常驻之后每份文档都有自己的挂载时刻，
   * 挂载那一刻 `activePath` 就是它自己，逐面板取值与「当时激活的是谁」天然一致。
   */

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
  // 面板的汇报入口已改成按面板分桶的 `reportHandle` / `reportViewInfo`
  // （见「面板常驻」一节），这里不再有单一面板的回调。

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
      /**
       * 活动栏的「文件浏览器」。
       *
       * 与 `toggleSidebar` 分开，是因为它们要的是两种行为：
       * `Ctrl+B` 是「把侧栏收起来 / 放出来」（**保留**用户正在看的那一栏），
       * 而这个按钮是「我要看文件树」—— 侧栏正显示检索时，它应当**换过去**
       * 而不是把整个侧栏关掉。
       */
      openFilesView: () => {
        const showing = settings.sidebarVisible && settings.sidebarView === 'files';
        update(showing ? { sidebarVisible: false } : { sidebarVisible: true, sidebarView: 'files' });
      },
      openSearchFolder: () => {
        const showing = settings.sidebarVisible && settings.sidebarView === 'search';
        update(showing ? { sidebarVisible: false } : { sidebarVisible: true, sidebarView: 'search' });
      },
      closeSearchFolder: () => update({ sidebarVisible: false }),
      toggleOutline: () => update({ outlineVisible: !settings.outlineVisible }),
      openPalette: () => setPaletteOpen(true),
      closePalette: () => setPaletteOpen(false),
      openFind: () => setFindOpen(true),
      closeFind: () => setFindOpen(false),
      openSettings: () => setSettingsOpen(true),
      closeSettings: () => setSettingsOpen(false),
      setTheme: (preference: ThemePreference) => update({ theme: preference }),
      cycleTheme: () => update({ theme: cycleTheme(settings.theme) }),
      // 这两个「开关」改的是设置层字段（与设置页同一个真相），不是临时视图状态；
      // 代码内核与代码块的显隐由各面板的 effect 消费这些字段
      toggleWordWrap: () => update({ wordWrap: !settings.wordWrap }),
      toggleCodeLineNumbers: () => update({ codeLineNumbers: !settings.codeLineNumbers }),
      // 全屏是平台桥的动作（桌面 = Tauri 窗口级，浏览器 = Fullscreen API），
      // 失败就让它 reject 落进未处理 Promise —— 两端实现都不该在正常路径上失败
      toggleFullscreen: () => {
        void toggleFullscreen();
      },
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
    [
      closeActiveTab,
      paneInfo,
      provider,
      settings.codeLineNumbers,
      settings.outlineVisible,
      settings.sidebarView,
      settings.sidebarVisible,
      settings.theme,
      settings.wordWrap,
      update,
      workspace,
    ],
  );

  const context = useMemo<CommandContext>(() => ({ workspace, provider, ui }), [workspace, provider, ui]);

  const registry = useMemo(() => {
    const created = new CommandRegistry();
    registerBuiltInCommands(created);
    return created;
  }, []);

  // 快捷键覆盖（P6 第 31 项）：整表推给注册表。settings 里改了快捷键（设置页）
  // 会产生新的 keybindings 对象，这里重推一次 —— setKeybindingOverrides 是幂等的。
  useEffect(() => {
    registry.setKeybindingOverrides(settings.keybindings);
  }, [registry, settings.keybindings]);

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

  /**
   * 菜单里可勾选项的状态。**只画状态，不产生语义** —— 真相来源是设置快照
   * 与活动标签页的视图模式，菜单的 ✓ 与状态栏、设置页读的是同一份字段。
   * 返回 `undefined` 的项没有开关语义，菜单不画勾选列。
   */
  const menuChecked = useCallback(
    (id: string): boolean | undefined => {
      switch (id) {
        case 'view.toggleSidebar':
          return settings.sidebarVisible;
        case 'view.filesView':
          return settings.sidebarVisible && settings.sidebarView === 'files';
        case 'view.toggleOutline':
          return settings.outlineVisible;
        case 'view.toggleWordWrap':
          return settings.wordWrap;
        case 'view.toggleCodeLineNumbers':
          return settings.codeLineNumbers;
        case 'view.rich':
        case 'view.source':
        case 'view.split': {
          const mode = paneInfo?.viewMode ?? 'rich';
          return id === `view.${mode}`;
        }
        case 'theme.system':
          return settings.theme === 'system';
        case 'theme.paper':
          return settings.theme === 'paper';
        case 'theme.ink':
          return settings.theme === 'ink';
        default:
          return undefined;
      }
    },
    [
      paneInfo,
      settings.codeLineNumbers,
      settings.outlineVisible,
      settings.sidebarView,
      settings.sidebarVisible,
      settings.theme,
      settings.wordWrap,
    ],
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

  // ---- 外部改动：窗口重新获得焦点时比对 mtime（兜底） --------------------
  //
  // 主通道是文件监听（下面那个 effect）：事件丢了、漏了、监听没起来，
  // 焦点回归时这次全量比对仍然能把「文件被外部改了」找回来。

  useEffect(() => {
    const onFocus = () => {
      void workspace.checkExternalChanges();
    };
    window.addEventListener('focus', onFocus);
    return () => window.removeEventListener('focus', onFocus);
  }, [workspace]);

  // ---- 外部改动：文件监听（P3 第 14 项，ADR 0029） -----------------------

  useEffect(() => {
    if (!isDesktop()) return;

    let disposed = false;
    let cleanup: (() => void) | null = null;
    void onFilesChanged((batch) => {
      if (disposed) return;
      if (batch.error !== undefined) {
        // 监听后端自身出了问题（比如被监视的目录被删）。说一句原因，
        // 但不弹横幅 —— 下一批事件与焦点兜底会继续工作
        console.warn('文件监听出错：', batch.error);
        return;
      }
      workspace.handleFsChanges(batch.paths);
    }).then((unlisten) => {
      if (disposed) unlisten();
      else cleanup = unlisten;
    });

    return () => {
      disposed = true;
      cleanup?.();
    };
  }, [workspace]);

  /**
   * 把「想监视什么」推给 Rust 侧（整表替换，幂等）。
   *
   * 集合 = 工作区根目录（递归）+ 不在根目录下的打开文件（非递归）。
   * 根目录下的文件已被递归覆盖，单独再 watch 一遍 notify 会报错。
   * 依赖是 `watchPathsKey`（字符串）而不是对象 —— 集合内容没变就不发。
   */
  const watchPathsKey = useMemo(() => {
    const paths: string[] = [];
    const root = tree?.root ?? null;
    if (root !== null) paths.push(root);
    for (const tab of tabs) {
      if (root !== null && isInside(tab.path, root)) continue;
      paths.push(tab.path);
    }
    return JSON.stringify(paths);
  }, [tree, tabs]);

  useEffect(() => {
    if (!isDesktop()) return;
    // 全关了也要把集合清空 —— 否则 Rust 侧会一直守着一个已经关闭的目录
    const paths: string[] = JSON.parse(watchPathsKey) as string[];
    void watchFiles(paths).then((failed) => {
      if (failed.length > 0) console.warn('这些路径没能监视（多半已不存在，下次刷新会重试）：', failed);
    });
  }, [watchPathsKey]);

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

  /**
   * 常驻面板 = 访问过、还开着的标签。`storeOf` 类型上允许 null（store 与 tab
   * 同生共死，实际不会），滤掉以防万一 —— 面板组件要求非空。
   */
  const mountedPanes = useMemo(
    () =>
      tabs
        .filter((tab) => visitedIds.has(tab.id))
        .flatMap((tab) => {
          const store = workspace.storeOf(tab.id);
          return store === null ? [] : [{ tab, store }];
        }),
    [tabs, visitedIds, workspace],
  );

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

      {/*
        菜单栏（ADR 0037）：命令的另一个入口 —— 能点什么、什么时候灰着，
        与工具栏/命令面板同问 CommandRegistry.enabled，这里没有第二个真相来源。
      */}
      <MenuBar registry={registry} platform={platform} isEnabled={isEnabled} onRun={runCommand} isChecked={menuChecked} />

      <TabBar
        tabs={tabs}
        activeId={activeId}
        backTo={drillBackTo}
        onBack={goBackFromDrill}
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

      {kernelFailures.length > 0 && (
        <div className="banner error">
          <span className="grow">
            <strong>有 {kernelFailures.length} 个编辑内核没能加载</strong>
            （{kernelFailures.map((f) => `${f.id}：${f.reason}`).join('；')}）。
            相关文档会以纯文本打开 —— 其余功能不受影响。重启应用通常会解决。
          </span>
        </div>
      )}

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
          filesView={settings.sidebarView === 'files'}
          searchView={settings.sidebarView === 'search'}
          outlineVisible={settings.outlineVisible}
          canOutline={outlineEnabled}
          canFind={isEnabled('find.open')}
          canSearchFolder={canSearchFolder && isEnabled('search.inFolder')}
          onToggleSidebar={() => runCommand('view.filesView')}
          onSearchFolder={() => runCommand('search.inFolder')}
          onToggleOutline={() => runCommand('view.toggleOutline')}
          onFind={() => runCommand('find.open')}
          onPalette={() => runCommand('palette.open')}
          onSettings={() => runCommand('app.settings')}
        />

        {settings.sidebarVisible && (
          <>
            <div className="sidebar-slot" style={{ width: sidebarWidth }}>
              {settings.sidebarView === 'search' ? (
                <SearchPanel
                  store={searchStore}
                  root={tree.root}
                  canSearch={canSearchFolder}
                  onOpenMatch={openSearchMatch}
                  onClose={() => update({ sidebarVisible: false })}
                />
              ) : (
                <Sidebar
                  tree={tree}
                  activePath={activePath}
                  onOpenFolder={() => void workspace.openFolder()}
                  onToggleDir={(path) => void workspace.toggleDir(path)}
                  onOpenFile={(path) => void workspace.open(path)}
                  onRefresh={() => void workspace.refreshTree()}
                  onToggleIgnored={() => workspace.setShowIgnored(!tree.showIgnored)}
                />
              )}
            </div>
            <div className="sidebar-resizer" onPointerDown={beginResize} role="separator" aria-orientation="vertical" />
          </>
        )}

        <div className="workspace-main">
          {mountedPanes.length > 0 ? (
            /*
              面板常驻（P3 第 19 项）：每个访问过的标签一个槽位，非活动的整槽隐藏。
              切标签页不再销毁内核 —— 撤销栈、光标、滚动原样保留（这正是本项的目标），
              代价是常驻内存，实测数字与「为什么不全部挂载」见 `visitedIds` 的注释与 ADR 0034。
            */
            mountedPanes.map(({ tab, store }) => (
              <PaneSlot
                key={tab.id}
                tab={tab}
                active={tab.id === activeId}
                store={store}
                wordWrap={settings.wordWrap}
                codeLineNumbers={settings.codeLineNumbers}
                onToggleCodeLineNumbers={(next) => update({ codeLineNumbers: next })}
                splitRatio={settings.splitRatio}
                onSplitRatioChange={(next) => update({ splitRatio: next })}
                initialPosition={rememberPositions ? (positionsRef.current?.[tab.path] ?? null) : null}
                revealRequest={revealRequest}
                onRevealResult={handleRevealResult}
                onDocBlockOpen={openDocBlockReference}
                onDocBlockEditInPlace={editDocBlockInPlace}
                resolveDocBlockPreview={resolveDocBlockPreview}
                reportHandle={reportHandle}
                reportViewInfo={reportViewInfo}
                rememberPositionAt={rememberPositionAt}
              />
            ))
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
