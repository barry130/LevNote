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
  readonly Component: EditorComponent;
}
