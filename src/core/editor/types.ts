/**
 * 编辑器适配层契约。
 *
 * 四类文档（Markdown / 代码 / 思维导图 / draw.io）的内核差别极大，
 * 但对**外壳**来说它们只有四件事不一样：怎么挂载、怎么给值取值、怎么序列化成 HTML、什么时候通知内容变了。
 * 这个接口就是把这四件事抽出来 —— 外壳不认识 Milkdown、CodeMirror 或 mind-elixir。
 *
 * 设计约束：
 * - **不引入任何内核的类型**。这里只允许出现 React 与项目自己的类型。
 *   一旦签名里出现 ProseMirror 的 `EditorView`，四类内核就无法共用这一个接口了。
 * - **`toHtml()` 允许返回 null**。纯文本内核序列化成 HTML 没有意义，
 *   自包含导出会据此退化为 `<pre>` 包裹，而不是假装成功。
 */

import type { ComponentType } from 'react';

import type { FindOptions } from '../search/find.js';

export type EditorModuleId = 'markdown' | 'code' | 'mindmap' | 'diagram' | 'plaintext';

/**
 * 查找能力。
 *
 * 由内核**自己**实现，而不是外壳拿 `getValue()` 的字符串去找：
 * 编辑器的可见文本与它保存的源文本常常不是一回事 ——
 * Markdown 内核的 `getValue()` 给的是 Markdown 源码，而用户看到的是渲染结果，
 * 拿源码去搜「加粗」会在 `**加粗**` 上匹配到一个用户看不见的位置。
 * 只有内核知道「用户看到的文本」与「文档模型坐标」之间的映射。
 *
 * ## 为什么是「有状态的索引」而不是「返回偏移列表」
 *
 * 返回偏移需要调用方理解内核的坐标系（ProseMirror 位置 / CodeMirror 位置 /
 * textarea 字符偏移），而这三个坐标系互不相同，外壳就得为每个内核写一份翻译。
 * 改成「内核自己记着上次的匹配列表，外壳只说第几个」之后，
 * 外壳只依赖 `find` → `reveal` → `replace` 这个动作序列，与内核无关。
 *
 * `find` 返回匹配总数。**顺序即导航顺序**（文档顺序，从头到尾）。
 * `reveal` / `replace` 的 `index` 是 0 基下标；越界时安静忽略而不是抛错 ——
 * 用户连按「下一个」到末尾时越界是正常操作，不是异常。
 */
export interface EditorSearch {
  find(options: FindOptions): number;
  /** 选中第 index 个匹配并滚动到可见处。 */
  reveal(index: number): void;
  /** 替换第 index 个匹配，然后重新查找。返回替换后的匹配总数。 */
  replace(index: number, replacement: string): number;
  /** 全部替换，返回替换掉的个数。 */
  replaceAll(replacement: string): number;
  /** 清空匹配状态（关闭查找栏、改查询时）。 */
  clear(): void;
}

/**
 * 内核交回给外壳的命令面。
 * 外壳持有它来取值、重置、聚焦、导出；内核在就绪时通过 `onReady` 交出。
 */
export interface EditorHandle {
  /** 编辑器里的当前内容（对 Markdown 内核即 Markdown 源文本）。 */
  getValue(): string;
  /**
   * 用外部内容覆盖编辑器。用于「重新加载」与冲突解决。
   *
   * 实现必须**避免回声**：由 setValue 触发的内容变化不应再回调 onChange，
   * 否则「外壳设值 → 内核报变更 → 外壳设值」会成环。
   */
  setValue(text: string): void;
  focus(): void;
  /**
   * 序列化成**语义 HTML 片段**（不含 `<html>` / `<head>`），供自包含导出使用。
   *
   * 返回 `null` 表示该内核不支持导出 —— 这是**合法结果**，不是错误。
   * 自包含导出会退回用 `<pre>` 呈现原文，并如实说明，而不是伪造一份富文本。
   */
  toHtml(): Promise<string | null>;
  /**
   * 可选：编辑器内的查找替换。
   *
   * 不实现时外壳会明确告诉用户「当前内核不支持在编辑器内查找」，
   * 而不是给一个点了没反应的按钮。与 `onCursorChange` 同属「可选能力」，
   * 缺了它功能降级，但整个内核仍然可用。
   */
  readonly search?: EditorSearch;
  /**
   * 可选：格式化命令面（浮层工具条用）。
   *
   * 不实现时外壳**不显示**浮层工具条 —— 与 `sourceView` 的处理不同：
   * 那个是「有但这份文档用不上」（禁用），这个是「这个内核没有这个能力」。
   * 一个纯文本内核弹出加粗按钮，用户点下去只会什么都没发生。
   */
  readonly formatting?: EditorFormatting;
  /**
   * 可选：把光标移到某个标题处（大纲跳转）。
   *
   * 参数是「层级 + 标题文字」而不是「源文档行号」：行号只对
   * 「源码即文档」的内核（代码 / 纯文本）有意义，Markdown 内核的文档树里
   * 没有「行」这个概念，硬要它把行号翻译成位置只能靠猜 ——
   * 猜错的后果是跳到文档里一个无关的地方，比不跳更糟。
   *
   * 返回 false 表示找不到或这个内核不支持。外壳据此决定大纲项是否可点击，
   * 而不是给一个点了没反应的项。
   */
  goToOutlineItem?(item: { level: number; text: string }): boolean;
}

export interface CursorPosition {
  line: number;
  column: number;
}

// ---------------------------------------------------------------------------
// 选区格式（浮层工具条的输入）
// ---------------------------------------------------------------------------

/**
 * 选区在**视口坐标系**里的矩形 —— 与 `getBoundingClientRect()` 同一套坐标。
 *
 * 为什么是「视口坐标」而不是「相对编辑器容器的坐标」：内核能拿到的只有
 * `coordsAtPos()`，它给的就是视口坐标。若要换算成容器坐标，内核就得知道
 * 容器是谁、它滚动到哪里 —— 那是外壳的布局知识，内核不该有。
 * 外壳用 `position: fixed` 摆放浮层，视口坐标可以直接用，一次换算都不需要。
 *
 * 刻意是**快照**（普通数字字段）而不是 `DOMRect`：`DOMRect` 的字段是活取值，
 * 拿到之后再被布局改动就会变；而且它没法在测试里当普通对象断言。
 */
export interface SelectionRect {
  readonly top: number;
  readonly left: number;
  readonly bottom: number;
  readonly right: number;
  readonly width: number;
  readonly height: number;
}

/**
 * 行内格式。
 *
 * 这里是**外壳的语言**，不是任何内核的 mark 名。Milkdown 把删除线叫
 * `strike_through`、把行内代码叫 `inlineCode`，将来换成别的内核名字还会变 ——
 * 那层翻译住在内核自己的映射表里（`modules/markdown/selection.ts`），
 * 外壳只认这五个词。
 */
export type InlineFormat = 'strong' | 'emphasis' | 'strikethrough' | 'inlineCode' | 'link';

/** 块级格式。`heading` 需要配一个 1–6 的层级。 */
export type BlockFormat = 'paragraph' | 'heading' | 'blockquote' | 'bulletList' | 'orderedList' | 'codeBlock';

/** 选区**当前**的格式。工具条据此把对应的按钮显示成「已生效」。 */
export interface SelectionFormat {
  /**
   * 选区内**每一个字符都带有**的行内格式。
   *
   * 用「全部都有」而不是「起点有」：选中 `普通**加粗**` 时按钮不该亮 ——
   * 亮着会让人以为点一下是「取消加粗」，而实际上会变成整段加粗。
   */
  readonly inline: ReadonlySet<InlineFormat>;
  /** 选区起点所在的块。列表 / 引用里返回**容器**，不是容器里的段落。 */
  readonly block: BlockFormat;
  /** `block === 'heading'` 时的层级；其余情况为 null。 */
  readonly headingLevel: number | null;
  /**
   * 选区所在链接的地址；不在链接里时为 null。
   *
   * 单独给一个字段而不是让外壳去猜：工具条上的链接按钮要**回填**现有地址，
   * 否则用户想把 `http://` 改成 `https://` 就只能先删掉再重加。
   */
  readonly linkHref: string | null;
}

/** 内核上报的一次选区快照。`format` 与 `anchor` 必须同时有效，所以打成一包。 */
export interface SelectionSnapshot {
  readonly format: SelectionFormat;
  readonly anchor: SelectionRect;
}

/**
 * 斜杠菜单能插入的块。
 *
 * 刻意是**扁平的字符串**而不是 `{ block, level }` 那种结构：菜单项本身就要
 * 一个稳定的 id（用来做 key、做筛选、写测试），而「标题几级」是菜单项的身份
 * 而不是它的参数 —— `heading1` 和 `heading2` 是两个不同的条目，不是同一个条目
 * 加不同参数。
 */
export type InsertableBlock =
  | 'paragraph'
  | 'heading1'
  | 'heading2'
  | 'heading3'
  | 'blockquote'
  | 'bulletList'
  | 'orderedList'
  | 'codeBlock'
  | 'hr';

/**
 * 斜杠菜单的当前状态。
 *
 * 只在「一个空段落里、光标停在 `/` 打头的文字末尾」时非 null —— 这个触发条件
 * 刻意收得很紧：松一点（比如允许段落里已经有别的文字）就会在用户正常打字时
 * 反复弹出菜单，而那种「你以为在打字，它以为在选命令」的干扰比功能本身更烦人。
 */
export interface SlashState {
  /** `/` 之后已经输入的文字，用来筛选菜单。可以为空串。 */
  readonly query: string;
  /** 菜单该出现在哪儿（视口坐标），取的是**段落内容起点**而不是光标位置。 */
  readonly anchor: SelectionRect;
}

/**
 * 内核交回给外壳的**格式化命令面**。
 *
 * 与 `EditorSearch` 同一个套路：外壳只说「加粗」/「变成二级标题」，
 * 至于这一步在 ProseMirror 里是 `setBlockType` 还是别的什么，外壳不知道也不该知道。
 *
 * 两个方法都是 **toggle** 语义 —— 用户点第二下就该回到原样。
 * 「已经是这个格式了」时再调用一次，结果必须是取消它，而不是原地不动：
 * 原地不动会让按钮看起来点坏了。
 */
export interface EditorFormatting {
  /**
   * 切换行内格式。
   *
   * `value` 只有 `link` 用得上，链接的规则比其余四个多一层「指向哪」：
   *
   * - 给了非空 `value`：已经在链接里就**改地址**，不在就**加上**；
   * - 没给（或给了空串）：在链接里就**去掉**；不在就**什么都不做** ——
   *   地址得由用户给，内核不替他猜一个。
   *
   * 其余四个格式忽略 `value`。
   */
  toggleInline(format: InlineFormat, value?: string): void;
  /** 切换块级格式。`level` 只有 `heading` 用得上（1–6，越界按 1 处理）。 */
  toggleBlock(format: BlockFormat, level?: number): void;
  /**
   * 插入一个块 —— 斜杠菜单的入口。
   *
   * 与 `toggleBlock` 分开是因为它多一件事要做：**先把 `/query` 那一段删掉**。
   * 删文字和套格式必须由内核在一次调用里完成，外壳不该（也不能）自己去算
   * 「要删哪几个字符」——那是文档坐标，只有内核知道。
   *
   * 当前不在斜杠触发状态时，它就等价于 `toggleBlock`（不会去删任何东西）。
   */
  insertBlock(block: InsertableBlock): void;
}

/**
 * 挂载一个内核所需的全部输入。
 *
 * **回调一律用属性签名**（`onChange: (v: string) => void`），不用方法简写
 * （`onChange(v: string): void`）。两个原因，第二个更重要：
 *
 * 1. 组件会把 props 解构（`function Editor({ onChange }: Props)`）。方法简写声明的
 *    成员被当作「有 `this`」，解构出来就触发 `@typescript-eslint/unbound-method`
 *    —— 而 React 组件根本不是类实例，这个 `this` 不存在。属性签名没有这个问题。
 * 2. 方法参数是**双变**的，属性参数是**逆变**的（在 `strictFunctionTypes` 下）。
 *    用属性签名意味着「回调参数类型必须真的对得上」，而不是「长得差不多就行」。
 */
export interface EditorMountProps {
  /** 初始内容。**只在挂载时读取一次**；之后的外部变更一律走 `handle.setValue`。 */
  initialValue: string;
  /**
   * 当前文档路径。
   *
   * 只用于**推断**（代码内核据此选语法包），不是给内核去读文件的 ——
   * 内核永远不碰文件系统，那是 `DocumentProvider` 的职责。
   */
  path: string;
  /** 用户在编辑器里改动内容时回调。 */
  onChange: (value: string) => void;
  /** 内核就绪，交出命令面。只调用一次。 */
  onReady: (handle: EditorHandle) => void;
  /**
   * 光标位置变化。可选 —— 不想实现的内核不实现即可，
   * 外壳会自动隐藏状态栏里的行列显示，而不是显示一个假的 1:1。
   */
  onCursorChange?: (position: CursorPosition) => void;
  /**
   * 选区变化。可选 —— 不实现的内核，外壳就不显示浮层工具条。
   *
   * 参数为 `null` 表示「现在不该有浮层」，两种情况都归到它：
   * 一是没有选中文字（光标塌缩），二是内核量不出选区在屏幕上的位置。
   * 后者是**有意的降级**：藏起来好过把浮层定位到一个瞎猜的坐标上。
   *
   * 内核**不要**在编辑器失焦时上报 null。用户点工具条上的按钮必然先失焦，
   * 一失焦就藏，按钮会在 `click` 之前消失 —— 这是浮层菜单最经典的坑。
   */
  onSelectionChange?: (selection: SelectionSnapshot | null) => void;
  /**
   * 斜杠菜单的状态变化。可选 —— 不实现的内核，外壳就不显示斜杠菜单。
   *
   * 与 `onSelectionChange` 分开而不是合成一个回调，是因为两者的**生命周期不同**：
   * 选区每次移动都变，而斜杠菜单只在「空段落里打了 `/`」这个很窄的条件下出现。
   * 合成一个的话，外壳每次移动光标都要重新判断「要不要收菜单」。
   */
  onSlashChange?: (state: SlashState | null) => void;
  /** 编辑器报错（内核初始化失败等），交给外壳呈现。 */
  onError?: (error: Error) => void;
  readOnly?: boolean;
  placeholder?: string;
  /** 视图偏好：软换行。不适用的内核忽略即可。 */
  wordWrap?: boolean;
}

export type EditorComponent = ComponentType<EditorMountProps>;

export interface EditorAdapter {
  readonly id: EditorModuleId;
  /** 状态栏里显示的名字，如「Markdown」。 */
  readonly label: string;
  /**
   * 能否处理这个路径。按扩展名判断即可，不要读文件内容 ——
   * 适配器选择发生在打开文件**之前**，此时还没有内容。
   */
  matches(path: string): boolean;
  /**
   * 可选：拿到内容之后的**第二道确认**。
   *
   * 有些风险光看文件名判断不出来。典型例子是带 YAML front matter 的 Markdown：
   * 扩展名完全是 `.md`，但富文本内核会把开头的 `---` 当成水平分隔线、
   * 把 `title: xxx` 当成普通段落，**一保存就把元数据毁了**。
   *
   * 返回 `false` 表示「我处理不了这个内容」，由注册表继续找下一个候选，
   * 最终至少落到纯文本兜底 —— 内容不会被改坏，只是没有富文本视图。
   * 判断必须是纯函数、不能有副作用：它可能对同一个文件被调用多次。
   */
  canHandle?(path: string, text: string): boolean;
  /**
   * 可选：本内核的「**源码视图**」用哪个内核渲染。
   *
   * Markdown 内核靠它把「富文本 / 源码」的切换交给外壳：外壳只决定
   * **现在要看哪一个**，不需要知道源码视图底下坐着谁（Milkdown？CodeMirror？
   * 还是将来某个自研的）。这与 `EditorHandle` 的存在理由是同一条 ——
   * 外壳不认识具体内核。
   *
   * **不实现 = 没有源码视图**。代码内核与纯文本内核本身就是源码，
   * 对它们来说「切到源码视图」是一个没有意义的动作。外壳据此把
   * 「切换源码模式」显示为**禁用**（而不是隐藏）—— 隐藏会让用户以为
   * 功能不存在，禁用则明确告诉他「有，但这份文档用不上」。
   *
   * ## 它**不注册进注册表**，这是刻意的
   *
   * 注册表按路径挑内核，而同一个路径只能有一个答案。源码视图是
   * 「同一个路径的第二种看法」，不是「另一个能处理这个路径的内核」——
   * 把它塞进注册表，`resolveAdapter('a.md')` 就有了两个合法答案，
   * 而哪一个先被匹配到取决于注册顺序，那是没法推理的。
   * 所以它只挂在适配器对象上，由外壳在 `viewMode === 'source'` 时显式取用。
   */
  readonly sourceView?: EditorAdapter;
  readonly Component: EditorComponent;
}
