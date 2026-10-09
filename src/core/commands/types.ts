/**
 * 命令模型的类型。
 *
 * 命令是「一个可以在任何地方触发的动作」的抽象。有了它，
 * 快捷键、命令面板、工具栏按钮、右键菜单全部变成**同一个东西的不同入口** ——
 * 加一个功能只需要注册一次，三个入口同时具备，不会出现
 * 「工具栏上有但命令面板里搜不到」这类不一致。
 *
 * 这是本项目相对「每个按钮写一个 onClick」的路线最主要的工程收益：
 * **能力的清单只有一份**。命令面板直接由注册表生成，因此它永远不会过期。
 */

import type { ViewMode } from '../document/store.js';
import type { BomKind, DocumentProvider } from '../document/types.js';
import type { EditorHandle } from '../editor/types.js';
import type { ThemePreference } from '../theme/theme.js';
import type { WorkspaceStore } from '../workspace/store.js';

/**
 * 命令需要外壳提供的那部分能力。
 *
 * 刻意保持「动作式」而不是「状态式」：命令不该读 UI 的 state 去决定怎么做，
 * 它只说「把侧栏切一下」，具体切到什么状态由外壳决定。
 * 这样命令的实现里不会出现 `if (sidebarOpen)` 这类与 UI 结构耦合的分支。
 */
export interface UiActions {
  toggleSidebar(): void;
  /**
   * 活动栏的「文件浏览器」：侧栏没开就开，开着检索就换过去，开着文件树就收起来。
   *
   * 与 `toggleSidebar` 分开的理由：`Ctrl+B` 是「收起 / 放出侧栏」，
   * 它应当**保留**用户正在看的那一栏；而这个按钮的语义是「我要看文件树」。
   */
  openFilesView(): void;
  toggleOutline(): void;
  openPalette(): void;
  closePalette(): void;
  openFind(): void;
  closeFind(): void;
  /**
   * 打开跨文件检索面板（侧栏的那一栏）。
   *
   * 与 `openFind` 分开：那个是「在**当前文档**里找」（一个浮在编辑器下沿的栏），
   * 这个是「在**整个文件夹**里找」（侧栏里的一栏）。两者可以同时开着，
   * 而且用户会同时用到 —— 先用文件夹检索定位到文件，再在文件里找下一处。
   */
  openSearchFolder(): void;
  closeSearchFolder(): void;
  openSettings(): void;
  closeSettings(): void;
  setTheme(preference: ThemePreference): void;
  cycleTheme(): void;
  /** 切换代码内核的自动换行偏好（设置层，与设置页同一个字段）。 */
  toggleWordWrap(): void;
  /** 切换代码块 / 代码内核的行号显示偏好（设置层，同上）。 */
  toggleCodeLineNumbers(): void;
  /** 切换专注模式（设置层字段，内核按能力消费）。 */
  toggleFocusMode(): void;
  /** 切换打字机模式（同上）。 */
  toggleTypewriterMode(): void;
  /** 切换窗口全屏（平台桥决定用 Tauri 的窗口 API 还是 Fullscreen API）。 */
  toggleFullscreen(): void;
  /**
   * 换当前文档的保存编码（菜单「编码」的入口；与状态栏下拉同一个动作）。
   *
   * 只改元数据：屏幕上的文字不变，**保存时**才按新编码写盘 ——
   * 与「用错误编码打开」的重新探测是两回事。
   */
  setEncoding(encoding: string, bom: BomKind): void;
  /** 当前聚焦的编辑器命令面。没有内核就绪时为 null。 */
  editor(): EditorHandle | null;
  /**
   * 当前文档能不能切到源码视图。
   *
   * 判断依据是**当前文档的首选内核有没有声明 `sourceView`**
   * （见 `EditorAdapter.sourceView`），而不是路径后缀 ——
   * 后缀判断会漏掉「内核起不来、退回了兜底内核」这种情况，
   * 那时点一下「源码模式」只会让界面闪一下，什么都不变。
   *
   * 外壳给不出答案时（没有活动文档）返回 false，命令随之显示为禁用。
   */
  canToggleSourceView(): boolean;
  /** 在当前文档的富文本视图与源码视图之间切换。 */
  toggleSourceView(): void;
  /**
   * 切到指定视图（富文本 / 源码 / 源码+预览并排）。
   *
   * 与 `toggleSourceView` 并存而不是取代它：三档视图需要一个「直接跳过去」的
   * 入口（`Ctrl+1/2/3`），而 `Ctrl+/` 是「在两档之间来回」的习惯动作。
   * 两者都只改 `store.viewMode`，语义不重叠。
   *
   * 外壳**不检查**目标视图在这个内核上是否可用 —— 调用方（命令）已经通过
   * `enabled` 做过这件事；store 那边也刻意不做判断（见 `DocumentStore.setViewMode`）。
   */
  setViewMode(mode: ViewMode): void;
  /** 弹一次提示（走平台对话框）。 */
  notify(message: string, title?: string): Promise<void>;
  /** 关闭当前标签页（内部会处理未保存确认）。 */
  closeActiveTab(): void;
}

export interface CommandContext {
  readonly workspace: WorkspaceStore;
  readonly provider: DocumentProvider;
  readonly ui: UiActions;
}

export interface Command {
  /** 稳定标识，形如 `file.save`。用于查找、埋点与将来的快捷键自定义。 */
  readonly id: string;
  /** 面板与菜单里显示的名字 */
  readonly title: string;
  /** 分组名，命令面板里用来分隔 */
  readonly category: string;
  /**
   * 快捷键写法，见 `keybinding.ts`。省略表示没有默认快捷键。
   *
   * 快捷键**写在命令自己身上**而不是维护一张外部映射表：
   * 那张表一定会漂 —— 命令改名或删掉后，表里的条目变成幽灵绑定，
   * 表现为「按了某个键什么都不发生」，且没人知道该去删哪一行。
   */
  readonly keybinding?: string;
  /** 额外的搜索关键词（拼音、英文名），让命令面板能按用户脑子里的词搜到 */
  readonly keywords?: readonly string[];
  /**
   * 当前是否可用。返回 false 的命令在面板里显示为禁用（而不是隐藏）——
   * 隐藏会让用户以为功能不存在，禁用则明确告诉他「有，但此刻不能用」。
   */
  readonly enabled?: (ctx: CommandContext) => boolean;
  /**
   * 数据驱动的动态命令（如「最近打开的文件」）。
   *
   * 它们有真实的 `run`，但**不是能力清单的一员** —— 是一份随用户与文件系统
   * 变化的数据。命令面板与快捷键表都按 registry 派生，不过滤的话这十来条
   * 「打开某某.md」会淹掉真正的能力条目；所以带这个标记的命令：
   * - 进菜单（MenuBar 按 id 现查，与其他项同一条路）；
   * - 不进命令面板、不进设置页的快捷键表。
   *
   * 标题是数据（文件名之类），与编码命令用 `option.label` 当标题同一条先例。
   */
  readonly dynamic?: boolean;
  run(ctx: CommandContext): void | Promise<void>;
}
