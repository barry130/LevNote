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
   * 可选：表格命令面。
   *
   * 与 `formatting` 并列而不是塞进去，理由见 `EditorTable` 的注释。
   * 不实现时外壳不显示表格工具条，也不注册表格相关的命令。
   */
  readonly table?: EditorTable;
  /**
   * 可选：代码块命令面。
   *
   * 与 `table` 并列。不实现时外壳不显示语言选择器面板 ——
   * 代码块本身仍然照常渲染，只是没法从富文本视图里改它的语言。
   */
  readonly codeBlock?: EditorCodeBlock;
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
  | 'mathBlock'
  | 'table'
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
 * 光标所在表格的状态 —— 表格浮层工具条的输入。
 *
 * 只在「光标落在表格里」时非 null。与 `SlashState` 同理：触发条件收窄之后，
 * 外壳不必每次移动光标都去判断要不要收浮层。
 */
export interface TableState {
  /**
   * **整张表格**在视口里的包围盒，浮层据此摆放。
   *
   * 取整张表而不是光标所在的单元格：用户心里的操作对象是表格，
   * 浮层跟着单元格跑会在表格里跳来跳去。
   */
  readonly anchor: SelectionRect;
  readonly rows: number;
  readonly columns: number;
  /** 光标所在单元格的对齐方式，用来点亮对应的对齐按钮。 */
  readonly align: TableAlign;
  /**
   * 还能不能再删一行 / 一列。
   *
   * 表格至少要有一行（表头行），只剩一行时删行是无意义的。
   * 把它报上来是为了让外壳**把按钮显示成禁用**，而不是给一个点了没反应的按钮 ——
   * 后者是本项目一直在避免的形态。
   */
  readonly canDeleteRow: boolean;
  readonly canDeleteColumn: boolean;
}

export type TableAlign = 'left' | 'center' | 'right';

/**
 * 表格操作。
 *
 * 用一个判别字符串而不是九个方法：外壳只说「在下面插一行」，
 * 至于那一步在 ProseMirror 里叫什么、要不要先改选区，内核自己决定。
 * 与 `InsertableBlock` 同理 —— 这些是**动作的身份**，不是参数。
 */
export type TableOperation =
  | 'rowBefore'
  | 'rowAfter'
  | 'colBefore'
  | 'colAfter'
  | 'deleteRow'
  | 'deleteColumn'
  | 'deleteTable'
  | 'exit';

/**
 * 内核交回给外壳的**表格命令面**。
 *
 * 与 `EditorFormatting` 分开而不是并进去：表格能力**只有支持表格的内核才有**，
 * 而格式化能力是「所有富文本内核都该有」。混在一起会让「这个内核支持表格吗」
 * 变成一个需要逐方法探测的问题。
 */
export interface EditorTable {
  /** 光标在不在表格里。工具栏的显隐与命令的 `enabled` 都用它。 */
  isActive(): boolean;
  run(operation: TableOperation): void;
  /** 设置光标所在单元格的对齐方式。 */
  setAlign(align: TableAlign): void;
}

/**
 * 代码块语言选择器的状态 —— 那个面板的输入。
 *
 * ## 为什么它是**点击触发**，而不是像表格那样「光标一进去就出现」
 *
 * 语言标签（代码块右上角那个 `js`）**常驻**在每一个代码块上，不管光标在哪 ——
 * 那才是它的正常形态（Typora 也是这样）。既然标签一直都在，触发就是
 * 「用户点了它」，而不是「光标在代码块里」。所以这个状态只在面板真的开着时非 null，
 * 生命周期与 `TableState`（跟着光标走）不同。
 *
 * ## 为什么带 `language` 而不是让外壳自己去 DOM 里读
 *
 * 外壳拿不到 `code_block` 的 `attrs.language` —— 那是 ProseMirror 的文档模型，
 * 外壳不认识它。而标签上显示的文字（`js` / `纯文本`）是**展示口径**，
 * 不一定等于属性值（空串显示成「纯文本」）。两者都由内核给出，外壳只负责画。
 */
export interface CodeBlockState {
  /** 面板该出现在哪儿（视口坐标），取的是语言标签的矩形。 */
  readonly anchor: SelectionRect;
  /** 代码块当前的语言标识；**空串表示未指定**（面板里显示成「纯文本」）。 */
  readonly language: string;
}

/**
 * 内核交回给外壳的**代码块命令面**。
 *
 * 与 `EditorTable` 并列而不是并进去，理由相同：只有支持代码块的内核才有它。
 *
 * 刻意**不暴露「第几个代码块」**：面板是内核打开的，它自己记着对应哪一个节点。
 * 一旦把位置交给外壳，外壳就得替内核保管一个文档坐标 —— 而那个坐标会随着
 * 用户在前面打字而失效，外壳既不知道什么时候失效，也没法自己修。
 */
export interface EditorCodeBlock {
  /**
   * 设置语言并关闭面板。
   *
   * **空串 = 清除语言**（回到 ```` ``` ```` 这种没有 info string 的围栏），
   * 而不是「什么都不做」—— 面板上的「纯文本」那一项就是它。
   */
  setLanguage(language: string): void;
  /** 关闭面板（Esc / 点面板外面 / 滚动）。 */
  close(): void;
}

/**
 * 内核要把一张图片落盘时的请求。
 *
 * ## 为什么是「内核交出字节、外壳负责落盘」
 *
 * 落盘要决定三件事：**存到哪**（文档旁边的资源目录）、**叫什么名**（不撞名）、
 * **怎么写进文档**（相对路径）。三件事全都依赖「文档在哪」，而那是外壳的知识 ——
 * 内核连自己正在编辑哪个文件都不该知道（它拿到的 `path` 只用于推断语法）。
 *
 * 所以分工是：内核做它唯一能做的那部分（从剪贴板里把字节取出来），
 * 外壳做其余全部。
 */
export interface ImagePasteRequest {
  /** 图片的原始字节。 */
  readonly bytes: ArrayBuffer;
  /**
   * 剪贴板报的 MIME，如 `image/png`。
   *
   * **可能是不认识的类型** —— 由外壳判断并如实拒绝，内核不做白名单：
   * 「支持哪些图片格式」这件事在 `core/image.ts` 里有唯一的一份答案，
   * 内核再抄一份只会多一个会过期的地方。
   */
  readonly mimeType: string;
}

/**
 * 用户点开的一张图片 —— 灯箱要显示的全部信息。
 *
 * 三个字段各自有用途，缺一个都会让灯箱说少一句话：
 * `url` 决定画什么，`src` 决定**这是哪个文件**（用户下一步要去找它），
 * `alt` / `title` 是作者给它的说明（有就显示，没有就不显示）。
 */
export interface ImageInfo {
  /**
   * 已经解析好、能直接放进 `<img>` 的 URL（`data:` URI）。
   *
   * 内核给的是**它已经加载出来的那一个**，而不是让外壳拿着 `src` 再解析一次 ——
   * 再解析一次意味着把同一个文件读第二遍（甚至第三遍），
   * 而这份字节此刻就在 DOM 上，白扔了。
   */
  readonly url: string;
  /** 文档里写的那一串（`./a.assets/x.png`）。用来在灯箱上标出这是哪个文件。 */
  readonly src: string;
  readonly alt: string;
  readonly title: string;
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
  /**
   * 表格状态变化。可选 —— 不实现的内核，外壳就不显示表格工具条。
   *
   * 与 `onSelectionChange` 分开，理由相同：表格工具条在**光标塌缩**时也要出现
   * （用户点进某个单元格），而选中工具条只在有选中文字时出现。
   */
  onTableChange?: (state: TableState | null) => void;
  /**
   * 代码块语言选择器的状态变化。可选 —— 不实现的内核，外壳就不显示这个面板。
   *
   * 与 `onTableChange` 分开：那个跟着光标走，这个由**点击语言标签**触发
   * （标签常驻在每个代码块上，与光标在不在里面无关，理由见 `CodeBlockState`）。
   */
  onCodeBlockChange?: (state: CodeBlockState | null) => void;
  /**
   * 把文档里的图片 `src` 解析成**能直接放进 `<img>` 的 URL**。
   *
   * 内核拿到的 `src` 是文档里写的那一串（`./a.assets/x.png`），而 WebView 的
   * base URL 是应用自身 —— 直接塞进 `<img>` 会去请求应用自己的资源目录，404。
   * 所以「显示本地图片」这件事**只能由外壳提供**：只有它知道文档在哪、
   * 也只有它有读文件的能力（内核不许碰文件系统）。
   *
   * 返回 `null` 表示「这张图拿不到」，内核据此画一个带路径的占位 ——
   * 与静默留一个破图相比，用户至少知道发生了什么、缺的是哪一张。
   *
   * 不实现时内核不显示任何图片，一律画占位。
   */
  resolveImageUrl?: (src: string) => Promise<string | null>;
  /**
   * 粘贴了一张图片：内核把字节交出来，外壳负责落盘并**返回写进文档的相对路径**。
   *
   * 返回 `null` 表示放弃（格式不认识、文档没保存过、落盘失败）——
   * 此时内核**什么都不做**：不插入节点，也不报错。理由与浮层按钮一致：
   * 拒绝的原因只有外壳知道，由它去说；内核再报一次会变成两条消息。
   *
   * 不实现时内核不处理图片粘贴，剪贴板里的图会被安静忽略（这正是纯文本内核的期望）。
   */
  onImagePaste?: (request: ImagePasteRequest) => Promise<string | null>;
  /**
   * 用户点了文档里的一张图片（灯箱）。
   *
   * ## 为什么由内核报、而不是外壳在编辑器容器上挂一个 click
   *
   * 外壳「从 DOM 事件反推出点了哪个图片节点」需要认识内核画的类名，
   * 而**外壳不许 import 任何具体内核**（依赖方向：`modules/` → `core/editor/types.ts`
   * ← `ui/`）。把类名挪到 `core/` 只是把耦合换个地方藏起来 —— 那份类名
   * 本质上是内核的渲染细节。让内核在它自己构造的那个元素上挂监听，
   * 是唯一不需要外壳知道任何 DOM 细节的做法。
   *
   * 不实现时点图片什么都不发生（纯文本内核没有图片，自然也走不到这里）。
   */
  onImageClick?: (image: ImageInfo) => void;
  /** 编辑器报错（内核初始化失败等），交给外壳呈现。 */
  onError?: (error: Error) => void;
  readOnly?: boolean;
  placeholder?: string;
  /** 视图偏好：软换行。不适用的内核忽略即可。 */
  wordWrap?: boolean;
  /**
   * 视图偏好：代码块是否显示行号。不适用的内核忽略即可。
   *
   * 与 `wordWrap` 同属「视图偏好」，但**用法不同**：软换行要内核真的换一种编辑行为，
   * 而行号纯粹是画出来的东西。所以内核读它只为了**选一个类名**
   * （`MarkdownEditor` 给 `.qm-editor` 挂上 `qm-no-line-numbers`），
   * 行号槽本身照建不误、由 CSS 负责显隐。
   *
   * 这样做换来的是「改这个偏好不需要重建内核」：重建会丢掉光标、滚动位置与撤销栈，
   * 而用户只是在面板上勾了一个框。代价是关掉行号时那一列 DOM 仍然存在 ——
   * 每个行号是一个空 `div`，实测在几千行的代码块上仍是可忽略的开销；
   * 若将来它真的成为瓶颈，重审条件是「把它挪进内核并强制重绘 NodeView」。
   */
  codeLineNumbers?: boolean;
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
