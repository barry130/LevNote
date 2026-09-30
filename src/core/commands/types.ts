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

import type { DocumentProvider } from '../document/types.js';
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
  toggleOutline(): void;
  openPalette(): void;
  closePalette(): void;
  openFind(): void;
  closeFind(): void;
  openSettings(): void;
  closeSettings(): void;
  setTheme(preference: ThemePreference): void;
  cycleTheme(): void;
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
  run(ctx: CommandContext): void | Promise<void>;
}
