/**
 * 单文档面板：告示条 + 编辑器 + 状态栏。
 *
 * ## 它为什么自己订阅 store，而不是从外壳收一堆 props
 *
 * 这一层要的东西全部来自**一份文档**的状态（脏、冲突、编码、行尾、光标、统计），
 * 把它们提到外壳再传下来，等于让外壳同时持有「工作区状态」和「活动文档状态」
 * 两套东西，而后者在切标签页时还要跟着换 —— 那是外壳最容易写错的地方。
 *
 * 反过来，让面板自己订阅「我这份文档」之后：
 * - 外壳完全不需要知道活动文档的任何字段；
 * - 切标签页就是换一个 `store`，其余一切自动正确；
 * - 面板的每一行代码都只依赖一个 store，不依赖它现在是第几个标签页。
 *
 * 唯一的对外输出是 `onHandle`：外壳持有编辑器命令面来执行导出、查找这类命令。
 */

import { useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore } from 'react';
import type { PointerEvent as ReactPointerEvent } from 'react';

import type { LineEndingName } from '../core/document/types.js';
import type { ConflictChoice, DocumentStore, ViewMode } from '../core/document/store.js';
import type {
  BlockHandleState,
  CodeBlockState,
  CursorPosition,
  DiagramView,
  EditorAdapter,
  DocBlockPreview,
  EditorBlockHandle,
  EditorCodeBlock,
  EditorFormatting,
  EditorHandle,
  EditorPosition,
  EditorTable,
  ImageInfo,
  ImagePasteRequest,
  SelectionSnapshot,
  SlashState,
  TableState,
} from '../core/editor/types.js';
import { resolveAdapter } from '../core/editor/registry.js';
import { ENCODINGS, encodingKey as encodingKeyCore } from '../core/encodings.js';
import { SPLIT_RATIO_MAX, SPLIT_RATIO_MIN } from '../core/settings.js';
import { analyze, formatStatistics } from '../core/statistics.js';
import { EDITOR_FALLBACK } from '../modules/index.js';
import { BlockHandle } from './BlockHandle.js';
import { CodeBlockPicker } from './CodeBlockPicker.js';
import { DiagramViewer } from './DiagramViewer.js';
import { Lightbox } from './Lightbox.js';
import { SelectionToolbar } from './SelectionToolbar.js';
import { SlashMenu } from './SlashMenu.js';
import { TableToolbar } from './TableToolbar.js';
import { ViewSwitcher } from './ViewSwitcher.js';

// 编码候选清单与它的稳定键住在 `core/encodings.ts` —— 菜单栏的「编码」菜单
// 与这里的下拉从**同一份**生成，抄成两份迟早会各改各的（硬约束 5 的近亲）
const LINE_ENDINGS: Array<{ value: LineEndingName; label: string }> = [
  { value: 'lf', label: 'LF' },
  { value: 'crlf', label: 'CRLF' },
  { value: 'cr', label: 'CR' },
];

const STATUS_TEXT = {
  clean: '已保存',
  dirty: '有未保存的改动',
  saving: '处理中…',
  conflict: '磁盘上有更新的版本',
  error: '出错',
} as const;

/**
 * 分屏时「左栏改完 → 右栏重画」的等待窗口（毫秒）。
 *
 * 取 160ms 的依据是**两端**的代价：
 *
 * - 太短（比如逐字推、或者 16ms 一帧）等于把打字成本乘以文档长度 ——
 *   右栏每收到一次新文本都要**整篇重新解析**（`setValue` → 一次完整 parse
 *   ＋ 重新渲染全部节点）。这正是 `open-decisions.md` 的 V3 要测的那类退化，
 *   而分屏恰恰是最容易把它引出来的场景：用户在左栏打字，右栏全程陪跑。
 * - 太长（比如 500ms）用户会明确感到「预览慢半拍」，开始怀疑是不是卡了。
 *
 * 160ms 落在「连续击键的间隔（约 80–150ms）」与「人开始察觉延迟（约 200ms）」
 * 之间：连续打字时它一直在重置、一次都不推（等用户停手），
 * 而用户停手后到预览更新只隔一次眨眼。
 *
 * 这是个**手感参数**，不是正确性参数 —— 调它不会让任何测试变红，
 * 只有人眼能判断好坏。
 */
const PREVIEW_SYNC_DELAY_MS = 160;

/**
 * 面板向上汇报的「视图信息」。
 *
 * 外壳为什么需要它：工具栏上那个「源码 / 富文本」按钮既要判断**能不能切**，
 * 又要显示**现在在哪一边**。这两件事都住在文档状态里，而外壳**刻意不订阅
 * 文档状态** —— 它只订阅工作区，否则用户每敲一个字，整个外壳（侧栏、标签栏、
 * 工具栏）都要跟着重渲染。所以由面板把外壳真正需要的两个字段报上来，
 * 而不是让外壳去偷看某一份文档的内部状态。
 *
 * `adapter` 给的是**首选内核**（不考虑源码视图），这是刻意的：
 * 切到源码模式之后，当前内核变成了代码内核，而代码内核自己没有 `sourceView` ——
 * 若把「当前内核」报上去，按钮会在切过去的那一刻自己变成禁用，用户再也切不回来。
 */
export interface PaneViewInfo {
  /** 首选内核。null 表示这份文档没有解析出内核（正常情况下不会发生）。 */
  readonly adapter: EditorAdapter | null;
  readonly viewMode: ViewMode;
}

/**
 * 一次检索定位请求（工作区检索 → 「跳到那个文件的那一行」）。
 *
 * 由外壳立起、面板落地。`tick` 是外壳侧自增的编号：同一条结果被再次点击时
 * 请求的内容可能一模一样，面板靠编号分辨「这是新的一次请求」。
 *
 * 为什么由面板落地而不是外壳拿句柄自己定位：见 `revealRequest` prop 的注释。
 */
export interface PaneRevealRequest {
  readonly path: string;
  readonly location: {
    /** 1 基行号。 */
    readonly line: number;
    /** 0 基列，UTF-16 码元（与 `WorkspaceMatch` 一致）。 */
    readonly column: number;
    readonly length: number;
  };
  readonly tick: number;
}

export interface DocumentPaneProps {
  store: DocumentStore;
  /**
   * 这个面板现在是不是活动标签的那一个。
   *
   * 面板常驻（P3 第 19 项）之后外壳里同时挂着 N 份面板，非活动的整槽隐藏。
   * 隐藏要连带两件事：**收起一切浮层与全屏视图**（它们的键盘监听挂在 window
   * 的捕获阶段，见下方 effect 的注释），以及**不再收「浮层类」上报**
   * （隐藏的内核不会有人输入，但外部改写之类的路径可能让它重报一次，
   * 收下来就是一份位置已经失效的浮层状态）。
   * 内容与位置的上报**不在此列**：内容永远是真相，位置按「谁上报记谁」。
   */
  active: boolean;
  wordWrap: boolean;
  /** 代码块是否显示行号。视图偏好，不写进文档。 */
  codeLineNumbers: boolean;
  onToggleCodeLineNumbers: (next: boolean) => void;
  /**
   * 分屏时左栏占的宽度比例（0.2–0.8）。视图偏好，不写进文档。
   *
   * 与 `wordWrap` 一样由外壳持有并持久化：面板只是把它**用**在布局上。
   * 面板自己存的话，切标签页会重置成 50%，而用户刚把分隔条拖好。
   */
  splitRatio: number;
  /** 分隔条拖完（松手）时提交新比例。**拖动过程中不要调它** —— 见 `beginSplitResize`。 */
  onSplitRatioChange: (next: number) => void;
  /** 编辑器就绪时交出命令面；卸载或换文档时交回 null。 */
  onHandle: (handle: EditorHandle | null) => void;
  /** 首选内核或视图模式变化时通知外壳；卸载时以 null 撤回。 */
  onViewInfoChange?: (info: PaneViewInfo | null) => void;
  /**
   * 这份文档上次离开时的位置。外壳从位置记忆里取，null 表示没有记忆。
   *
   * 由外壳传进来而不是面板自己去读存储：位置记忆是**跨文档**的
   * （「最近打开过哪些文件」），那是外壳的知识；面板只该知道自己这一份。
   * 面板自己去读的话，它还得知道存储键、淘汰策略、开关状态 —— 三样都不属于它。
   */
  initialPosition?: EditorPosition | null;
  /**
   * 位置变化（光标或滚动）。**高频**：光标每动一次、容器每滚一下都会调用。
   *
   * 外壳必须把它当成高频信号处理（存 ref + 防抖落盘），
   * 而不是 setState —— 否则用户每敲一个字都要重渲染整条链路。
   */
  onPositionChange?: (position: EditorPosition) => void;
  /**
   * 用户点击了一个 doc-block 引用卡片（P6 第 28 项）——「打开被引用的文档」。
   *
   * 由内核上报（监听挂在卡片元素上，硬约束 11），面板原样转发。
   * 「src 相对谁」的换算与打开动作都在外壳：只有外壳知道文档路径。
   */
  onDocBlockOpen?: (ref: { src: string; kind: string; title: string | null }) => void;
  /**
   * 用户点了卡片上的「就地编辑」（P6 第 28 项 L2，ADR 0036）。
   * 与 `onDocBlockOpen` 同一条转发纪律：内核上报，面板原样转发。
   */
  onDocBlockEditInPlace?: (ref: { src: string; kind: string; title: string | null }) => void;
  /**
   * 卡片「预览」展开要的结构摘要。摘要的计算在 App（读文件归外壳），
   * 面板只转发。不传时内核不画预览按钮。
   */
  resolveDocBlockPreview?: (ref: { src: string; kind: string }) => Promise<DocBlockPreview | null>;
  /**
   * 待落地的检索定位请求；null 表示没有。
   *
   * ## 为什么「落地」发生在面板里、而且发生在内核就绪那一刻
   *
   * 这与 `initialPosition`（位置记忆）是**同一个时刻、同一条理由**：
   * 内核是异步创建的，外壳握着句柄「戳一次」的写法赶不上
   * 「内核刚就绪」与「面板重建」谁先谁后 —— StrictMode 的模拟重挂
   * （开发模式）实测踩中过：定位刚落了地，面板随后重挂，
   * 选区与状态栏的光标读数一起被销毁。
   * 把请求作为 prop 传下来、在 `handleEditorReady` 里应用，
   * **任何一次内核重建都会重新落地**，竞态从设计上消失。
   *
   * 请求立起时外壳已经把目标文件打开并激活了，所以这里的对账只有一条：
   * `request.path` 是否就是这份文档。
   */
  revealRequest?: PaneRevealRequest | null;
  /**
   * 一次定位请求落地（或确认落不了地）时回报。
   *
   * `applied` 为 false 说明当前内核**不认源码行列**（富文本内核没有这个概念）
   * 或行号越界 —— 怎么回落（切源码视图 / 放弃）是外壳的决策，面板只如实上报。
   * 同一次内核重建里成功过的请求不会重复回报（见 `appliedRevealTicksRef`）。
   */
  onRevealResult?: (tick: number, applied: boolean) => void;
}

export function DocumentPane({
  store,
  active,
  wordWrap,
  codeLineNumbers,
  onToggleCodeLineNumbers,
  splitRatio,
  onSplitRatioChange,
  onHandle,
  onViewInfoChange,
  initialPosition,
  onPositionChange,
  revealRequest,
  onRevealResult,
  onDocBlockOpen,
  onDocBlockEditInPlace,
  resolveDocBlockPreview,
}: DocumentPaneProps) {
  const state = useSyncExternalStore(store.subscribe, store.getSnapshot);
  const { doc, status, error, conflict, busy, notice, viewMode } = state;

  const handleRef = useRef<EditorHandle | null>(null);
  /**
   * `handleRef` 里这个句柄属于**哪一次内核实例**，形如 `会话号:视图模式`。
   *
   * 必须有它：换文件、切视图都会重建内核，而新的内核是**异步**创建的，
   * 同步的 effect 会先跑。只凭 `handleRef.current` 存在就往上写，
   * 会写到一个已经被销毁的旧内核上。
   *
   * 把「视图模式」也算进这个标识，是因为切视图时**会话号不变** ——
   * 只用会话号的话，那一段异步窗口里 `handleRef` 会被误认为「有效」。
   */
  const handleKeyRef = useRef<string | null>(null);
  const lastReportedRef = useRef<string | null>(null);
  const lastPushedRef = useRef<string | null>(null);

  /**
   * 分屏右栏（只读预览）的句柄与同步基准。
   *
   * 与左栏那三个 ref 一一对应，但**刻意不复用**：两个内核是两个独立实例，
   * 「我属于哪一次实例」「我最后推给它的是什么」这两件事各自都要有一份。
   * 共用一套的后果很具体 —— 左栏就绪时会把右栏的基准一起改掉，
   * 于是分屏刚打开的那一帧右栏会闪一下旧内容。
   *
   * 声明在这个位置（而不是 `previewKey` 旁边）是因为下面的
   * `handlePositionChange` 要用到 `previewRef`。
   */
  const previewRef = useRef<EditorHandle | null>(null);
  const previewKeyRef = useRef<string | null>(null);
  const previewPushedRef = useRef<string | null>(null);

  /**
   * 待恢复的位置，以及上报用的回调。
   *
   * **都用 ref 而不是直接进依赖**，理由不同但都必要：
   *
   * - `initialPosition` 只在**内核就绪那一刻**读一次。放进 `handleEditorReady` 的
   *   依赖表会让它的标识随位置变化而变，而 `handleEditorReady` 是传给内核的
   *   `onReady` —— 纯文本兜底内核把它写在 effect 依赖里，一变就重建内核。
   * - `onPositionChange` 是**高频**回调（光标每动一次都调）。它一变就重渲染
   *   整条链路的话，用户每敲一个字都要走一遍 React 的提交。
   */
  const initialPositionRef = useRef(initialPosition);
  initialPositionRef.current = initialPosition;
  const onPositionChangeRef = useRef(onPositionChange);
  onPositionChangeRef.current = onPositionChange;

  /**
   * 左栏上报的位置。
   *
   * 除了转给外壳（位置记忆），分屏时还要**顺手把滚动比例同步给右栏**。
   *
   * 这一步能成立，靠的是 ADR 0021 定下的那条约定：`offset` 是**内核空间的不透明值**，
   * 而 `textLength` 是一致性凭据。左栏报的是源码字符数、右栏算的是
   * ProseMirror 的 `doc.content.size`，两者**必然对不上** ——
   * 于是右栏的 `setPosition` 会走「只恢复滚动、不动光标」那条路。
   * 这正是分屏想要的效果：两栏一起滚，但预览栏不抢焦点。
   *
   * 换句话说：**跨内核传位置时「对不上」不是缺陷，而是这里唯一正确的行为。**
   * 若哪天有人「修好」了 textLength 让两边相等，右栏就会开始跟着左栏跳光标，
   * 而那是一个比不联动更糟的结果。
   */
  const handlePositionChange = useCallback((position: EditorPosition) => {
    onPositionChangeRef.current?.(position);
    previewRef.current?.setPosition?.(position);
  }, []);

  const [cursor, setCursor] = useState<CursorPosition | null>(null);
  const [editorError, setEditorError] = useState<string | null>(null);

  /** 内核启动失败的那一次实例标识。用标识而不是布尔值，换文件 / 切视图后自然重新尝试。 */
  const [failedKey, setFailedKey] = useState<string | null>(null);
  /**
   * **预览栏**创建失败的那一次实例标识。与 `failedKey` 分开是必须的。
   *
   * 共用 `failedKey` 会造成一个很难查的连锁反应：右栏起不来 → 标记成
   * 「这一次内核失败」→ 左栏被判定失败 → 整个文档退化成白底 textarea。
   * 而实际情况是「预览渲染不出来，源码完全正常」—— 用户会以为文件坏了。
   */
  const [previewFailedKey, setPreviewFailedKey] = useState<string | null>(null);

  /**
   * 浮层工具条的两份输入：当前选区，和内核交出来的格式化命令面。
   *
   * 都放在 state 里而不是只留 ref：它们直接决定**画什么**。
   * 分开两个 state 是因为它们的生命周期不同 ——
   * 选区每次移动都换，命令面一个内核实例只有一个。
   */
  const [selection, setSelection] = useState<SelectionSnapshot | null>(null);
  const [formatting, setFormatting] = useState<EditorFormatting | null>(null);
  /**
   * 斜杠菜单的当前状态。
   *
   * 与 `selection` 并列而不是合成一个：两者的触发条件几乎互斥
   * （斜杠要求光标塌缩，工具条要求有选中文字），合成一个反而要写分支。
   */
  const [slash, setSlash] = useState<SlashState | null>(null);
  /**
   * 表格工具条的两份输入，与浮层工具条同构：当前表格状态 + 内核交出来的表格命令面。
   *
   * `table` 与 `formatting` 分成两个 state（而不是合成一个「命令面」对象），
   * 是因为它们的**可用性不同**：表格能力只有支持表格的内核才有，
   * 合成一个会让「这个内核支持表格吗」变成一个需要逐字段探测的问题。
   */
  const [table, setTable] = useState<TableState | null>(null);
  const [tableOps, setTableOps] = useState<EditorTable | null>(null);
  /**
   * 代码块语言选择器的两份输入，与上面几组同构。
   *
   * 与前几组不同的是它的**触发方式**：表格工具条跟着光标走，而这个面板是
   * 用户点代码块右上角的语言标签才打开的（那个标签常驻在每个代码块上）。
   * 所以 `codeBlock` 非 null 的含义是「面板开着」，不是「光标在代码块里」。
   */
  const [codeBlock, setCodeBlock] = useState<CodeBlockState | null>(null);
  const [codeBlockOps, setCodeBlockOps] = useState<EditorCodeBlock | null>(null);
  /**
   * 块手柄的两份输入，与上面几组同构。
   *
   * `blockHandle` 非 null 的含义就是字面的「鼠标停在这一块上」，
   * 指针离开编辑器它会变 null —— **但那不等于「手柄该藏起来」**：
   * 手柄在编辑器外面，用户要点它就必须先离开编辑器。
   * 所以这里只负责把内核报的值存下来，**显隐由 `BlockHandle` 自己合三个条件**
   * （见那个组件的文件头）。
   */
  const [blockHover, setBlockHover] = useState<BlockHandleState | null>(null);
  const [blockHandleOps, setBlockHandleOps] = useState<EditorBlockHandle | null>(null);
  /**
   * 块菜单开着吗。
   *
   * **由外壳决定开关时机**，内核只提供「操作哪一块」。四个关闭时机：
   * Esc / 点菜单外面 / 滚动（菜单会跟内容分家）/ 执行完一个动作。
   * 内容变了也关 —— 那条走 `handleEditorChange`。
   */
  const [blockMenuOpen, setBlockMenuOpen] = useState(false);
  /**
   * 灯箱里正在看的那张图。null 表示灯箱关着。
   *
   * 只存**已经解析好的那一个**（内核给什么就是什么），不拿 `src` 再解析一次 ——
   * 那意味着把同一个文件读第二遍，而字节此刻就在 DOM 上。
   */
  const [lightbox, setLightbox] = useState<ImageInfo | null>(null);
  /** 全屏正在看的那张图表。null 表示查看器关着。 */
  const [diagram, setDiagram] = useState<DiagramView | null>(null);
  /**
   * 拖动分隔条过程中的**临时**比例。null 表示「没在拖」。
   *
   * 与 `sidebarWidth` 的处理完全同构：拖动期间只改这个本地值，
   * 松手才把它提交给外壳去落盘。逐次 `pointermove` 都写一次 localStorage
   * 会让拖拽明显发涩 —— 而拖拽恰恰是最需要跟手的一个交互。
   */
  const [liveSplitRatio, setLiveSplitRatio] = useState<number | null>(null);

  const session = doc?.session ?? 0;
  /**
   * 这次内核实例的标识。
   *
   * 加上视图模式换来一个「自然重试」：一次「富文本内核起不来」不该把用户
   * 永久锁在兜底内核上 —— 他至少该能切到源码视图，用代码内核把内容看全。
   */
  const kernelKey = `${session}:${viewMode}`;
  const kernelFailed = failedKey !== null && failedKey === kernelKey;

  /**
   * 右栏预览的内核标识。
   *
   * 刻意**不含 `viewMode`**：右栏只在分屏下存在，所以它一旦存在就是
   * 「这一次会话的只读渲染」。用会话号而不是复用 `kernelKey`，
   * 是为了避免「左栏的 key 变了右栏跟着重建」这种没必要的耦合。
   */
  const previewKey = `${session}:preview`;
  const previewFailed = previewFailedKey !== null && previewFailedKey === previewKey;

  /**
   * 检索定位请求的镜像、以及「哪些请求已经在这块面板里落地过」的台账。
   *
   * **台账刻意是面板实例自己的 ref**：它要挡的是「同一次内核重建里重复落地」，
   * 而它**挡不住** StrictMode 的模拟重挂 —— 那次重挂会连这个 ref 一起销毁，
   * 新面板实例拿着同一个请求重新落地。这正是要的：选区与光标读数因此活过重挂。
   * 「用户切走标签页后不该再跳」由外壳收掉请求来保证（App 里的对账 effect）。
   */
  const revealRequestRef = useRef<PaneRevealRequest | null>(revealRequest ?? null);
  revealRequestRef.current = revealRequest ?? null;
  const appliedRevealTicksRef = useRef<Set<number>>(new Set());

  /**
   * 把当前请求落到现在这块面板上。落不了地时如实回报，由外壳决定回落。
   *
   * 放在 ref 里而不是 `useCallback`：`handleEditorReady` 会被**纯文本兜底内核
   * 写进 effect 依赖**，它的标识一变内核就重建 —— 所以它依赖的每一件东西
   * 都必须稳定，读「最新值」一律走 ref（与 `initialPositionRef` 同一条理由）。
   */
  const applyRevealRef = useRef<() => void>(() => undefined);
  applyRevealRef.current = () => {
    const request = revealRequestRef.current;
    if (request === null || request.path !== doc?.path) return;
    if (appliedRevealTicksRef.current.has(request.tick)) return;
    const handle = handleRef.current;
    // 句柄必须属于当前这次内核实例 —— 异步创建窗口里握着的可能是旧的（见 handleKeyRef）
    if (handle === null || handleKeyRef.current !== kernelKey) return;
    const applied = handle.revealSourceLocation?.(request.location) === true;
    if (applied) appliedRevealTicksRef.current.add(request.tick);
    onRevealResult?.(request.tick, applied);
  };

  // 请求在内核已经就绪时到达（点同一条结果、或同一文件里的另一条）：
  // 没有内核创建事件可搭，就在 prop 变化的这次渲染之后直接落地。
  useEffect(() => {
    applyRevealRef.current();
  }, [revealRequest]);

  /**
   * 选内核（**首选内核**，与视图模式无关）。
   *
   * 依赖刻意只写 `session`：`canHandle` 关心的是文件开头（front matter 之类），
   * 与用户后续的编辑无关。若依赖写成 `doc.text`，每敲一个字都会重跑一次
   * 「注册表匹配 + front matter 探测」，没有任何意义。
   *
   * 之所以把「首选内核」单独算出来而不是直接算「用哪个」：外壳需要首选内核
   * 才能回答「能不能切源码」，而切过去之后当前内核已经不是它了。
   */
  const { primary, downgraded } = useMemo(() => {
    const current = store.getSnapshot().doc;
    if (!current) return { primary: null, downgraded: false };
    const byContent = resolveAdapter(current.path, EDITOR_FALLBACK, current.text);
    const byName = resolveAdapter(current.path, EDITOR_FALLBACK);
    // 扩展名认得、内容却让首选内核退位 —— 这是一次有理由的降级，必须告诉用户
    return { primary: byContent, downgraded: byContent.id !== byName.id };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session, store]);

  /**
   * 实际渲染用的内核。
   *
   * 优先级：**内核起不来 → 兜底** ＞ 源码模式 → 首选内核的 `sourceView` ＞ 首选内核。
   * 「起不来」排在最前，因为一个创建失败的实例渲染不出任何东西，
   * 而兜底内核至少保证内容可读可改 —— **打开文件绝不能失败**。
   */
  const adapter = useMemo(() => {
    if (primary === null) return null;
    if (kernelFailed) return EDITOR_FALLBACK;
    // `source` 与 `split` 的**左栏**都是源码视图；内核没声明 `sourceView` 时
    // 退回首选内核（与源码模式同样的降级：宁可显示富文本，也不要空着）
    if (viewMode !== 'rich') return primary.sourceView ?? primary;
    return primary;
  }, [primary, viewMode, kernelFailed]);

  /**
   * 分屏右栏的只读预览内核。null 表示「这次不分屏」。
   *
   * 三个条件缺一不可：
   * - **首选内核起得来**。它起不来时左栏已经是兜底内核（白底 textarea）了，
   *   再挂一个「同一份文档的只读渲染」没有意义；
   * - **首选内核真的声明了 `sourceView`**。少了这一条，左右两栏会是同一个内核 ——
   *   那不是分屏，是并排画两遍同样的东西；
   * - 当前确实是 `split`。
   *
   * 预览用**首选内核本身**加 `readOnly`，而不是另造一条 Markdown → HTML 的渲染管线。
   * 后者会得到「预览和富文本视图长得不一样」的结果，而用户一定会认为其中一个是错的 ——
   * 而且两条渲染管线要各自维护表格、公式、图表、图片的相对路径解析。
   */
  const previewAdapter = useMemo(() => {
    if (primary === null || kernelFailed) return null;
    if (viewMode !== 'split') return null;
    if (primary.sourceView === undefined) return null;
    return primary;
  }, [primary, viewMode, kernelFailed]);

  useEffect(() => {
    onViewInfoChange?.({ adapter: primary, viewMode });
  }, [primary, viewMode, onViewInfoChange]);

  /**
   * 卸载时把视图信息撤回。
   *
   * 少了这一步，切标签页的那一帧工具栏会拿着**上一个标签页**的内核信息去决定
   * 「能不能切源码」—— 表现为按钮闪一下（比如从 `.md` 切到 `.py` 时，
   * 按钮先亮着再变灰）。React 会把这次清理与下一个面板的首次上报放在同一批里，
   * 所以用户看不到中间态。
   */
  useEffect(() => () => onViewInfoChange?.(null), [onViewInfoChange]);

  // 卸载（换标签页 / 关标签页）时把命令面交回去，避免外壳握着一个已销毁的内核
  useEffect(() => () => onHandle(null), [onHandle]);

  const EditorComponent = adapter?.Component ?? null;
  /** 分屏右栏要用的内核。null 表示这次不分屏（见 `previewAdapter`）。 */
  const PreviewComponent = previewAdapter?.Component ?? null;

  // ---- 内核 ↔ 外壳 的双向同步 -------------------------------------------

  const handleEditorChange = useCallback(
    (value: string) => {
      lastReportedRef.current = value;
      store.updateText(value);
      /**
       * 内容变了就收起块菜单。
       *
       * 菜单里记着的那一块可能已经被推到了别处（比如上一行插进了一段字），
       * 而菜单还停在原来的屏幕位置上 —— 用户点「删除」删掉的会是**另一块**。
       * 内核那边也会重新算一次目标块，但两边的判断合起来才安全：
       * 这里关掉菜单，就不存在「菜单与目标对不上」的窗口了。
       */
      setBlockMenuOpen(false);
    },
    [store],
  );

  const handleEditorReady = useCallback(
    (handle: EditorHandle) => {
      handleRef.current = handle;
      handleKeyRef.current = kernelKey;
      // 错误属于**上一个创建失败的实例**。新实例就绪说明界面已经能用了，
      // 把过期错误留在横幅上，用户会以为「现在还是坏的」。
      setEditorError(null);
      // 内核刚才是用 `doc.text` 挂载的，所以两边已经一致，不需要再推一次。
      // 同步这两个基准，是为了让下面那个 effect 不会在刚挂载时就误判成外部变更。
      const text = store.getSnapshot().doc?.text ?? '';
      lastReportedRef.current = text;
      lastPushedRef.current = text;
      // 只在内核真的交出了格式化能力时才留下它。纯文本兜底内核没有这个字段 ——
      // 那时浮层根本不出现，而不是出现一个点了没反应的
      setFormatting(handle.formatting ?? null);
      // 同理：只有支持表格的内核才交出 `table`。外壳据此决定要不要画表格工具条，
      // 而不是「画出来再禁用按钮」—— 后者会让纯文本内核里也浮出一条死工具条
      setTableOps(handle.table ?? null);
      // 同理：只有支持代码块语言的内核才交出 `codeBlock`
      setCodeBlockOps(handle.codeBlock ?? null);
      // 同理：只有富文本内核才有「块」这个概念，源码 / 纯文本内核不交 `blockHandle`
      setBlockHandleOps(handle.blockHandle ?? null);

      /**
       * 恢复上次离开时的光标与滚动。
       *
       * 放在这里（内核就绪之后）而不是更早：`setPosition` 要往文档里派发一个
       * 选择事务，而文档树是内核建出来的 —— 内核没就绪时没有地方可派发。
       *
       * 优先用**切视图时**从上一个内核接过来的位置（A4）—— 它比位置记忆里的
       * 「打开文档时」新；用完即清，别让它漏到下一次不相关的内核重建里。
       *
       * **只在有位置、且这个内核支持位置读写时才调。** 不支持的内核
       * （将来可能的思维导图 / draw.io）没有这个方法，文档照常打开，
       * 只是从头开始 —— 而不是抛一个空引用错误把打开流程整个打断。
       */
      const remembered = viewSwitchPositionRef.current ?? initialPositionRef.current;
      viewSwitchPositionRef.current = null;
      if (remembered && handle.setPosition) handle.setPosition(remembered);

      // 检索定位放在位置恢复**之后**：用户刚点的那条结果比「上次离开时的位置」新，
      // 两个都想动光标时应该听用户的。
      applyRevealRef.current();

      onHandle(handle);
    },
    [kernelKey, store, onHandle],
  );

  /**
   * 右栏预览就绪。
   *
   * 它**刻意不做** `handleEditorReady` 做的那些事，逐条对应：
   *
   * - **不调 `onHandle`**。命令面（保存、查找、格式化、导出）属于
   *   **用户正在编辑的那一栏**，也就是左栏。右栏交出句柄会让 `Ctrl+S`
   *   保存到预览上去 —— 而两者其实是同一份文档，用户不会立刻发现，
   *   等到发现时已经晚了。
   * - **不恢复位置**。左栏恢复完之后会通过 `handlePositionChange`
   *   把滚动同步过来。右栏自己再恢复一次是重复劳动，而且两份位置可能不同。
   * - **不设 `formatting` / `table` / `codeBlock`**。那些是浮层工具条的输入，
   *   而预览栏里不该弹工具条：在预览里选中一段文字、浮出一个「加粗」、
   *   点下去却作用在左栏 —— 没有任何用户能猜到这个行为。
   * - **不碰 `failedKey`**。见 `previewFailedKey` 的注释。
   */
  const handlePreviewReady = useCallback(
    (handle: EditorHandle) => {
      previewRef.current = handle;
      previewKeyRef.current = previewKey;
      // 右栏刚才是用 `doc.text` 挂载的，两边已经一致 —— 记下基准，
      // 免得下面那个同步 effect 立刻又推一遍
      previewPushedRef.current = store.getSnapshot().doc?.text ?? '';
    },
    [previewKey, store],
  );

  /**
   * 右栏创建失败。
   *
   * 只标右栏，不动左栏 —— 源码照常可编辑，界面上会如实说明「预览没起来」。
   * 这是「绝不静默」在分屏这一处的具体形态：**降级要说出来，而且只降该降的那一半**。
   */
  const handlePreviewError = useCallback(() => {
    previewRef.current = null;
    previewKeyRef.current = null;
    setPreviewFailedKey(previewKey);
  }, [previewKey]);

  /**
   * 左栏 → 右栏的内容同步。
   *
   * **必须防抖**：`doc.text` 每敲一个字就变一次，而右栏拿到新文本要
   * **整篇重新解析**（`setMarkdown` 是一次完整 parse + 重新渲染）。
   * 逐字推过去等于把打字成本乘以文档长度 —— 长文档上这会直接拖垮手感，
   * 而那正是这条技术路线上最该避免的一类退化（见 `open-decisions.md` 的 V3）。
   *
   * 窗口取 160ms：短于人的「停一下」感知，长于连续击键的间隔。
   * 代价是预览会滞后一点点 —— 分屏预览本来就不是逐帧同步的东西。
   *
   * 它**不读 `handleRef`**：那条链路（下面那个 effect）是「外壳 → 左栏」的
   * 外部变更推送，方向相反。混在一起会让「谁该覆盖谁」变成一个需要仔细推理的问题，
   * 而这类问题一旦推理错就是静默丢数据。
   */
  useEffect(() => {
    if (previewAdapter === null || previewFailed) return;
    if (previewKeyRef.current !== previewKey) return;
    const text = doc?.text;
    if (text === undefined || text === previewPushedRef.current) return;

    const timer = setTimeout(() => {
      const handle = previewRef.current;
      // 定时器到期时世界可能已经变了：换了文件、切了视图、内核重建过。
      // 这三件事都会让 `previewKeyRef` 与 `previewKey` 不再相等
      if (!handle || previewKeyRef.current !== previewKey) return;
      handle.setValue(text);
      previewPushedRef.current = text;
    }, PREVIEW_SYNC_DELAY_MS);
    return () => clearTimeout(timer);
  }, [doc, previewAdapter, previewFailed, previewKey]);

  useEffect(() => {
    if (!doc) return;
    // 新内核还没就绪（或者启动失败了），此时 handleRef 里要么是空的、
    // 要么指向上一份文档 / 上一次视图那个已经被销毁的实例，都不能碰
    if (handleKeyRef.current !== kernelKey) return;
    const handle = handleRef.current;
    if (!handle) return;
    // 内核自己报的：外壳这边已经把内容记下了，不要再推回去（回声）
    if (doc.text === lastReportedRef.current) return;
    // 外壳自己刚推过的：不要推第二遍
    if (doc.text === lastPushedRef.current) return;
    // 走到这里说明 doc.text 是被**外部**改的：重新加载、冲突解决、或换文件
    handle.setValue(doc.text);
    lastPushedRef.current = doc.text;
  }, [doc, session, kernelKey]);

  // 光标位置随文档切换重置，避免新文档还显示上一份的行列
  useEffect(() => {
    setCursor(null);
  }, [session]);

  /**
   * 换一次内核实例就清掉失败标记 —— 也就是**每次显式切视图都重试目标内核**。
   *
   * 少了这一步会留下一个永久失败标记：用户切到源码视图、再切回富文本，
   * 看到的仍然是兜底内核，而且**没有任何解释**（错误横幅属于上一个实例，
   * 早被撤掉了）。那是这个功能最糟的失败形态 —— 界面不对，还不说为什么。
   * 重试一次的成本是一次内核创建，而用户已经明确表达了「我要看那一边」。
   *
   * ## 预览栏的失败标记也在这里清
   *
   * 同样的道理，而且它更容易踩：`previewKey` 只含会话号（不含视图模式），
   * 所以「分屏 → 富文本 → 分屏」不会换出一个新的 `previewKey`。
   * 不在这里清的话，右栏失败一次就**永久**失败 —— 用户来回切几次都还是
   * 「预览没能渲染出来」，而他能做的只有换标签页。
   *
   * 这个 effect 在**离开分屏**时（`kernelKey` 从 `…:split` 变成 `…:rich`）
   * 就已经把它清掉了，所以再进分屏一定是一次全新的尝试。
   */
  useEffect(() => {
    setFailedKey(null);
    setPreviewFailedKey(null);
    // 换内核实例 = 换了一份文档或换了一边视图，上一个实例报上来的选区与命令面
    // 都已经失效。不清的话，切标签页的那一帧会拿旧选区画出一个位置正确的浮层 ——
    // 位置对了反而更糟，用户会以为它属于新文档
    setSelection(null);
    setFormatting(null);
    setSlash(null);
    setTable(null);
    setTableOps(null);
    setCodeBlock(null);
    setCodeBlockOps(null);
    setBlockHover(null);
    setBlockHandleOps(null);
    setBlockMenuOpen(false);
    // 灯箱也要关：换标签页之后还留着一张上一个文档的图，用户会以为它属于新文档
    setLightbox(null);
  }, [kernelKey]);

  /**
   * 把内核交出来的粘贴请求转给文档模型。
   *
   * 内核说的是「字节 + MIME」，文档模型说的是「写进文档的相对路径」——
   * 中间那一层（生成文件名、写到哪、绝不覆盖）全在 `DocumentStore.pasteImage`
   * 与 `core/document/images.ts` 里。这里只做一次形状转换，
   * 不让内核的契约类型漏进 `core/document`（那是两个不同层次的抽象）。
   */
  const handleImagePaste = useCallback(
    (request: ImagePasteRequest) => store.pasteImage(request.bytes, request.mimeType),
    [store],
  );

  /**
   * 关灯箱。
   *
   * **必须是稳定引用**：`Lightbox` 把它放进 `useEffect` 的依赖里（Esc 的监听），
   * 每渲染一次换一个新函数就等于每次打字都把那个 window 监听拆掉再装上。
   * 功能上不会坏，但那是「每次按键做一次多余的 DOM 操作」。
   */
  const closeLightbox = useCallback(() => setLightbox(null), []);
  /** 与 `closeLightbox` 同理：必须是稳定引用（`DiagramViewer` 的 Esc 监听依赖它）。 */
  const closeDiagram = useCallback(() => setDiagram(null), []);

  /**
   * 开关块菜单。
   *
   * ## 关闭时**不**通知内核清目标块
   *
   * 清目标块的时机在 `BlockHandle` 里，判据是「手柄彻底不可交互了」
   * （指针离开手柄、又不在任何一块上）—— 不是「菜单关了」。
   * 菜单关掉的那一刻指针还在手柄上，用户随时会再点一次；
   * 那时清掉目标，第二次打开的菜单里每个动作都会「点了没反应」。
   *
   * 同理**不要**在这里清 `blockHover` —— 那是内核报上来的事实，
   * 外壳自己清会让两边不一致（按 Esc 时指针还停在这一块上，
   * 手柄本该继续显示，清了它就闪一下再回来）。
   */
  const openBlockMenu = useCallback(() => setBlockMenuOpen(true), []);
  const closeBlockMenu = useCallback(() => setBlockMenuOpen(false), []);

  /**
   * 切换视图三态。
   *
   * 直接调 `store.setViewMode` 而不走命令注册表：命令是给**键盘与命令面板**
   * 用的入口（它们需要 `enabled` 判断、需要出现在快捷键表里），而这个按钮
   * 就在面板自己身上、状态也是面板自己读的。绕一圈反而要在这里再判一次
   * 「有没有活动文档」——而 `setViewMode` 自己已经静默忽略空文档了。
   */
  /**
   * 切视图前从**要退场的内核**那里取一次位置（A4）。
   *
   * 新内核就绪后（`handleEditorReady`）会把它应用上去 —— 恢复的语义完全走
   * 既有契约：两个视图的 `offset` 是不同坐标系（文档树尺寸 vs 源码字符数），
   * `textLength` 通常对不上，内核的 `setPosition` 于时**只恢复滚动比例、
   * 不动光标**（ADR 0021 的设计原本就为这一刻留了口子）。
   *
   * 刻意取「比例」而不是像素：比例与视口高度、字体无关，富文本与源码的
   * 内容高度本来就不同，像素毫无对应可言。
   */
  const viewSwitchPositionRef = useRef<EditorPosition | null>(null);

  const switchView = useCallback(
    (mode: ViewMode) => {
      if (mode === store.getSnapshot().viewMode) return;
      viewSwitchPositionRef.current = handleRef.current?.getPosition?.() ?? null;
      store.setViewMode(mode);
    },
    [store],
  );

  /**
   * 拖动分屏分隔条。
   *
   * ## 为什么在 `pointerdown` 那一刻就把容器的矩形量下来
   *
   * 拖动过程中容器的宽度**会变**（分隔条一动，两侧的宽度就变），
   * 但容器的**外框**是不变的。在每次 `pointermove` 里重量一次也得到同样的值，
   * 而那是每次移动都强制一次布局 —— 拖拽立刻变涩。量一次、记住，就够了。
   *
   * ## 为什么矩形宽度为 0 时直接返回
   *
   * jsdom 没有布局引擎，`getBoundingClientRect()` 返回全 0。不挡这一下会算出
   * `NaN`（0/0），而 `NaN` 会一路走进偏好里被持久化 —— 之后每次打开都是
   * 「比例是 NaN」，布局塌成一条线，看起来像设置页坏了。
   */
  const beginSplitResize = useCallback(
    (event: ReactPointerEvent<HTMLDivElement>) => {
      const container = event.currentTarget.parentElement;
      if (container === null) return;
      const rect = container.getBoundingClientRect();
      if (rect.width <= 0) return;
      event.preventDefault();

      const onMove = (move: PointerEvent) => {
        const ratio = (move.clientX - rect.left) / rect.width;
        setLiveSplitRatio(Math.min(SPLIT_RATIO_MAX, Math.max(SPLIT_RATIO_MIN, ratio)));
      };
      const onUp = () => {
        window.removeEventListener('pointermove', onMove);
        window.removeEventListener('pointerup', onUp);
        setLiveSplitRatio((ratio) => {
          if (ratio !== null) onSplitRatioChange(ratio);
          return null;
        });
      };
      window.addEventListener('pointermove', onMove);
      window.addEventListener('pointerup', onUp);
    },
    [onSplitRatioChange],
  );

  const handleEditorError = useCallback(
    (thrown: Error) => {
      setEditorError(thrown.message);
      // 记的是**这一次实例**的标识，而不是「当下 store 里的 session」——
      // 后者在切视图之后会指向另一个内核，把失败标记盖到它头上。
      // （已经卸载的实例不会报错：`MarkdownEditor` 的 effect 里有 `disposed` 检查。）
      setFailedKey(kernelKey);
    },
    [kernelKey],
  );

  // 只依赖文本本身：光标移动、脏标记翻转之类不该触发重新统计
  const docText = doc?.text;
  const stats = useMemo(() => (docText === undefined ? null : analyze(docText)), [docText]);

  const activeEncodingKey = doc ? encodingKeyCore(doc.encoding, doc.bom) : '';

  /**
   * 隐藏面板不收「浮层类」上报。
   *
   * 正常情况下隐藏的内核不会有人输入，但外部改写 → 内容重推这类路径
   * 会让内核重报一次选区 —— 收下来的浮层状态带着**面板隐藏时算出的坐标**，
   * 等用户切回来，浮层就画在错误的位置上。判定走 ref：包装器标识必须稳定
   * （内核把回调存进 ref），真正的开关是「此刻活动吗」。
   */
  const activeRef = useRef(active);
  activeRef.current = active;

  const reportSelection = useCallback((value: SelectionSnapshot | null) => {
    if (activeRef.current) setSelection(value);
  }, []);
  const reportSlash = useCallback((value: SlashState | null) => {
    if (activeRef.current) setSlash(value);
  }, []);
  const reportTable = useCallback((value: TableState | null) => {
    if (activeRef.current) setTable(value);
  }, []);
  const reportCodeBlock = useCallback((value: CodeBlockState | null) => {
    if (activeRef.current) setCodeBlock(value);
  }, []);
  const reportBlockHover = useCallback((value: BlockHandleState | null) => {
    if (activeRef.current) setBlockHover(value);
  }, []);

  /**
   * 面板被隐藏（切到别的标签）时收起一切浮层与全屏视图。
   *
   * 浮层的键盘监听挂在 window 的捕获阶段（硬约束 9）—— 面板藏起来
   * **并不会摘掉那些监听**，不收起的话隐藏面板的菜单会截走活动面板的按键：
   * 回车先把段落拆开、Esc 把别人的浮层关掉。灯箱与图表全屏同理，
   * 而且它们本来就是「此刻屏幕上的这一份」，人已经不在这份文档上了。
   *
   * 能力标记（`formatting` / `tableOps` / `codeBlockOps` / `blockHandleOps`）
   * **不清**：那是内核就绪时交出的能力清单，不是浮层状态，
   * 清了就要等下一次内核重建才恢复。
   */
  useEffect(() => {
    if (active) return;
    setSelection(null);
    setSlash(null);
    setTable(null);
    setCodeBlock(null);
    setBlockHover(null);
    setBlockMenuOpen(false);
    setLightbox(null);
    setDiagram(null);
  }, [active]);

  /**
   * 左栏（不分屏时是唯一一栏）的编辑器元素。
   *
   * 抽成变量而不是写两遍 JSX：分屏与不分屏的**左栏完全相同**，
   * 写两遍的话将来加一个 prop 就得记得改两处 ——
   * 而漏改的那一处表现是「分屏下某个功能不工作」，很难联想到是 props 少传了。
   */
  const editorNode =
    doc && EditorComponent ? (
      <EditorComponent
        // 换文件、换视图都要重建内核：撤销栈、光标、滚动位置都该跟着换。
        // 「另存为」换了路径但 session 与视图都不变，所以不会被重建 ——
        // 否则用户刚改完按 Ctrl+Shift+S，光标会跳回开头。
        key={kernelKey}
        initialValue={doc.text}
        path={doc.path}
        readOnly={doc.readonly}
        wordWrap={wordWrap}
        codeLineNumbers={codeLineNumbers}
        placeholder="在这里写。支持 Markdown、代码 / 文本、思维导图（.mindmap）与 draw.io 图表（.drawio）四类文档。"
        onChange={handleEditorChange}
        onReady={handleEditorReady}
        onCursorChange={setCursor}
        onPositionChange={handlePositionChange}
        onSelectionChange={reportSelection}
        onSlashChange={reportSlash}
        onTableChange={reportTable}
        onCodeBlockChange={reportCodeBlock}
        // 只读文档不给手柄：手柄上全是「改文档」的动作，而只读文档里
        // 那些动作会被 schema / 只读视图挡下 —— 用户看到的是一个能点但什么都不发生的菜单。
        // 直接不给（`undefined`）比「给了再禁用每一项」更简单，也更诚实
        onBlockHandleChange={doc.readonly ? undefined : reportBlockHover}
        resolveImageUrl={store.resolveImageUrl}
        resolveDocBlock={store.resolveDocBlock}
        onImagePaste={handleImagePaste}
        onImageClick={setLightbox}
        onDiagramOpen={setDiagram}
        onDocBlockOpen={onDocBlockOpen}
        onDocBlockEditInPlace={onDocBlockEditInPlace}
        resolveDocBlockPreview={resolveDocBlockPreview}
        onError={handleEditorError}
      />
    ) : null;

  /**
   * 现在是不是真的要并排显示两栏。
   *
   * 判据是 `previewAdapter` **加上**「右栏这一次没起崩」——
   * 右栏起崩之后 `previewAdapter` 仍然非 null（它是「该不该分屏」的判断，
   * 与运行时成败无关），所以少了后半个条件会让一个已经报错的预览栏
   * 继续占着屏幕右半边。
   */
  const splitActive = previewAdapter !== null && PreviewComponent !== null && !previewFailed;

  /** 拖动中就用临时值，否则用偏好里那个。拖动结束会把临时值提交上去并清空它。 */
  const effectiveSplitRatio = liveSplitRatio ?? splitRatio;

  /**
   * 右栏报上来的内容变更。
   *
   * **它不该被调用**，而且这里刻意什么都不做。两个理由：
   *
   * 1. 右栏是 `readOnly` 的（Markdown 内核靠 ProseMirror 的
   *    `editable: () => false` 实现真正的只读），用户改不动它；
   * 2. 就算它报了，**也不能往 `store` 里写**：左栏才是用户正在编辑的那一栏，
   *    同一份 `doc.text` 有第二个写入方就会变成竞态 —— 而竞态的两种结果
   *    （丢用户的字 / 把预览的旧文本写回去）都属于静默丢数据。
   *
   * 所以这里不是「忘了处理」，是**明确地拒绝第二个写入方**。
   * 之所以还要留一个函数而不是传 `() => {}`：每次渲染新建一个函数会让
   * 内核那一侧的 props 比较失效（它把 `onChange` 存进 ref，但仍有别的实现
   * 会把它放进依赖里），而这是个稳定引用，零成本。
   */
  const ignoreChange = useCallback(() => undefined, []);

  return (
    <>
      {conflict && (
        <div className="banner warn">
          <strong>磁盘上的文件已被其他程序修改。</strong>
          <span className="grow">{conflict.message}</span>
          <button type="button" onClick={() => void store.resolveConflict('reload')} disabled={busy}>
            用磁盘版本覆盖我的
          </button>
          <button type="button" onClick={() => void store.resolveConflict('save-copy')} disabled={busy}>
            另存为副本
          </button>
          <button type="button" onClick={() => void store.resolveConflict('overwrite')} disabled={busy}>
            用我的覆盖磁盘
          </button>
        </div>
      )}

      {doc?.lossy && (
        <div className="banner info">
          <span className="grow">
            这个文件的编码是自动猜出来的，可能猜错。若下方出现乱码，请在状态栏手动指定编码。
          </span>
        </div>
      )}

      {downgraded && (
        <div className="banner info">
          <span className="grow">
            这个文件的开头是 YAML front matter。用富文本内核打开会把 <code>---</code> 当成分隔线、
            把 <code>键: 值</code> 当成正文，保存时元数据就没了。
            所以这个文件<strong>以「{primary?.label ?? '纯文本'}」打开</strong>，内容不会有任何改动。
          </span>
        </div>
      )}

      {editorError && (
        <div className="banner error">
          <span className="grow">
            <strong>编辑器内核没能启动。</strong> {editorError}
            {kernelFailed ? '（已退回纯文本内核，内容不会丢）' : ''}
          </span>
          <button type="button" onClick={() => setEditorError(null)}>
            知道了
          </button>
        </div>
      )}

      {error && (
        <div className="banner error">
          <span className="grow">
            <strong>{error.code}</strong> · {error.message}
            {error.detail ? `（${error.detail}）` : ''}
          </span>
          <button type="button" onClick={store.clearError}>
            知道了
          </button>
        </div>
      )}

      {previewFailed && (
        <div className="banner warn">
          <span className="grow">
            <strong>预览没能渲染出来。</strong>
            源码那一栏照常可以编辑，保存也不受影响 —— 只是右边看不到渲染结果。
          </span>
          <button type="button" onClick={() => store.setViewMode('source')}>
            只显示源码
          </button>
        </div>
      )}

      <main className="editor-area">
        {editorNode !== null ? (
          splitActive ? (
            <div className="editor-split">
              {/* 左栏 = 源码（可编辑），右栏 = 只读渲染。
                  两栏的 flex-grow 用内联样式给：它是**每次拖动都在变**的值，
                  写进样式表就得靠自定义属性 + 一次字符串拼装，
                  而这里直接给数字更直白、也没有转义问题。
                  `flex-basis: 0` 在样式表里（见 styles.css）—— 那一半是不变的。 */}
              <div className="editor-pane" style={{ flexGrow: effectiveSplitRatio }}>
                {editorNode}
              </div>
              <div
                className="editor-divider"
                role="separator"
                aria-orientation="vertical"
                aria-label="拖动调整源码与预览的宽度"
                onPointerDown={beginSplitResize}
              />
              <div className="editor-pane" aria-label="只读预览" style={{ flexGrow: 1 - effectiveSplitRatio }}>
                <PreviewComponent
                  // 右栏的内核标识不含视图模式（见 `previewKey`）：它只在分屏下存在，
                  // 一旦存在就是「这一次会话的只读渲染」，与左栏是源码还是富文本无关
                  key={previewKey}
                  initialValue={doc?.text ?? ''}
                  path={doc?.path ?? ''}
                  // 只读。Markdown 内核靠 ProseMirror 的 `editable: () => !readOnly`
                  // 实现真正的只读 —— 不是「拦下按键」那种假只读
                  readOnly
                  wordWrap={wordWrap}
                  codeLineNumbers={codeLineNumbers}
                  onChange={ignoreChange}
                  onReady={handlePreviewReady}
                  // 图片与图表在预览栏里**照常可点**：那两件事只读不改文档，
                  // 而且「点一下放大看」正是用户想在预览里做的事
                  resolveImageUrl={store.resolveImageUrl}
                  resolveDocBlock={store.resolveDocBlock}
                  onImageClick={setLightbox}
                  onDiagramOpen={setDiagram}
                  onError={handlePreviewError}
                />
              </div>
            </div>
          ) : (
            editorNode
          )
        ) : doc ? (
          <div className="empty">
            <h2>找不到能打开这个文档的内核</h2>
            <p>这不应该发生 —— 纯文本兜底内核会处理任何文件。请把这份文件路径反馈给开发者。</p>
          </div>
        ) : null}

        {/* 浮层是编辑器区域的兄弟节点，不是子节点 —— 内核挂载时会替换掉自己那层容器的内容，
            放进同一层会被它抹掉。`position: fixed` 让它脱离 flex 布局，不会撑开高度 */}
        {selection !== null && formatting !== null && (
          <SelectionToolbar snapshot={selection} formatting={formatting} />
        )}

        {/* 斜杠菜单与工具条同属浮层，理由相同。两者不会同时出现：
            斜杠只在光标塌缩时触发，而工具条只在有选中文字时出现 */}
        {slash !== null && formatting !== null && <SlashMenu state={slash} formatting={formatting} />}

        {/* 表格工具条与前两者不同的一点：它**会**和斜杠菜单同时出现
            （光标在表格的某个格子里打 `/`）。这没关系 —— 两者在 CSS 里
            差一层 z-index（29 对 30），斜杠菜单压在上面，不会被工具条遮住 */}
        {table !== null && tableOps !== null && <TableToolbar state={table} table={tableOps} />}

        {/* 语言选择器也是浮层。它会和表格工具条同时出现的情况只有一种：
            表格的某个格子里写了代码块 —— 那不可能，代码块不能进表格。
            所以这里不需要像表格工具条那样安排 z-index 的高低 */}
        {codeBlock !== null && codeBlockOps !== null && (
          <CodeBlockPicker
            state={codeBlock}
            codeBlock={codeBlockOps}
            lineNumbers={codeLineNumbers}
            onToggleLineNumbers={onToggleCodeLineNumbers}
          />
        )}

        {/* 块手柄。它与上面几个浮层的**不同**在于：手柄常驻在悬停的那一块旁边，
            点开才出菜单。所以「要不要画」由组件自己合三个条件判断
            （有悬停 / 指针在手柄上 / 菜单开着），这里只做能力判断 */}
        {blockHandleOps !== null && !doc?.readonly && (
          <BlockHandle
            hover={blockHover}
            blockHandle={blockHandleOps}
            menuOpen={blockMenuOpen}
            onOpenMenu={openBlockMenu}
            onCloseMenu={closeBlockMenu}
          />
        )}

        {/* 灯箱是**模态**的：它铺满整个视口，所以放在最后（同层里最后画的在最上面），
            并且不像上面几个浮层那样有「什么时候该出现」的条件判断 ——
            它只由用户点图片决定 */}
        {lightbox !== null && <Lightbox image={lightbox} onClose={closeLightbox} />}

        {/* 图表查看器也是模态的，与灯箱同一层、排在它后面（同层最后画的在最上面）。
            两者不会同时出现：一个由点图片触发，一个由点「放大」按钮触发 */}
        {diagram !== null && <DiagramViewer diagram={diagram} onClose={closeDiagram} />}
      </main>

      <footer className="statusbar">
        <span className={`status-pill ${status}`}>{STATUS_TEXT[status]}</span>

        {doc && (
          <>
            <span className="sep">│</span>
            <span className="kernel" title="当前使用的编辑器内核">
              {adapter?.label ?? '无内核'}
              {/* 源码模式下说明「这是谁的源码」：否则一份 .md 的状态栏写着
                  「代码 / 文本」，用户会以为打开的文件不对 */}
              {viewMode === 'source' && primary?.sourceView ? ` · ${primary.label} 源码` : ''}
            </span>

            <span className="sep">│</span>
            <label>
              编码{' '}
              <select
                value={activeEncodingKey}
                onChange={(event) => {
                  const value = event.target.value;
                  if (value.startsWith('other|')) return;
                  const hit = ENCODINGS.find((item) => `${item.encoding}|${item.bom}` === value);
                  if (!hit) return;
                  // 编码与 BOM 必须一起改，否则会出现「GBK + UTF-8 BOM」这类不可能的组合
                  store.setFormat(hit.encoding, hit.bom);
                }}
              >
                {activeEncodingKey.startsWith('other|') && (
                  <option value={activeEncodingKey}>
                    {doc.encoding}
                    {doc.bom === 'none' ? '' : ' + BOM'}
                  </option>
                )}
                {ENCODINGS.map((item) => (
                  <option key={`${item.encoding}|${item.bom}`} value={`${item.encoding}|${item.bom}`}>
                    {item.label}
                  </option>
                ))}
              </select>
            </label>

            <label>
              行尾{' '}
              <select value={doc.lineEnding} onChange={(event) => store.setLineEnding(event.target.value as LineEndingName)}>
                {doc.lineEnding === 'mixed' && <option value="mixed">混合</option>}
                {LINE_ENDINGS.map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </select>
            </label>

            {/* 内核不报光标位置时整段隐藏，而不是显示一个假的 1:1 */}
            {cursor && (
              <>
                <span className="sep">│</span>
                <span>
                  第 {cursor.line} 行，第 {cursor.column} 列
                </span>
              </>
            )}
            <span className="sep">│</span>
            <span
              className="statistics"
              title="字数按 Microsoft Word 的口径：中日韩文字与标点各算一字，拉丁文按词计"
            >
              {stats ? formatStatistics(stats) : ''}
            </span>
            {doc.readonly && (
              <>
                <span className="sep">│</span>
                <span>只读文件</span>
              </>
            )}
          </>
        )}

        <span className="grow" />
        {notice && <span className="hint">{notice}</span>}
        <span className="sep">│</span>
        <span className="hint">Ctrl+S 保存 · Ctrl+P 命令面板</span>
        {/* 视图三档放在最右：VSCode 的状态栏最右侧就是「这份文档是什么、
            现在怎么看」这一类信息（语言模式、编码）。它**只在有得选的时候出现** ——
            代码 / 纯文本文档本身就是源码，「切到源码」对它们是个不存在的动作，
            给一个恒亮的单选项比不给更让人困惑。 */}
        {primary?.sourceView !== undefined && (
          <>
            <span className="sep">│</span>
            <ViewSwitcher value={viewMode} onChange={switchView} />
          </>
        )}
      </footer>
    </>
  );
}

export type { ConflictChoice };
