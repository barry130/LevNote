/**
 * 代码块的 NodeView：行号槽 + 语言标签。
 *
 * ## 为什么这里必须是 NodeView，而不是像表格那样用「浮层 + 装饰」
 *
 * 表格工具条是**跟着光标**的浮层：光标在哪个格子里，工具条就浮在整张表旁边。
 * 代码块的两样东西都不是这个形态：
 *
 * - **行号**要和代码的每一行对齐。代码块内部是一个带 `\n` 的文本节点，
 *   DOM 里并没有「一行一个元素」这种结构，所以只能由我们**自己造出逐行的 DOM**。
 *   装饰做不到这件事 —— `Decoration.inline` 造不出块边界，而空行（`from === to`）
 *   根本没法被 inline 装饰命中，行号会从空行那里开始错位。
 * - **语言标签**要常驻在每一个代码块的右上角（Typora 就是这样），
 *   与光标在不在里面无关。做成浮层的话，代码块一多就要为每个块跟踪位置，
 *   滚动时还得逐个重算 —— 而它本来就该是块自己的一部分。
 *
 * 两件事都要求「块的内部有我们自己的 DOM」，那就是 NodeView 的定义。
 *
 * ## 为什么不用 `@milkdown/kit/component/code-block`
 *
 * 它确实提供了语言选择器（可搜索）和 CodeMirror 高亮，但它**是 Vue 组件**：
 * `@milkdown/components` 的依赖里有 `vue@^3.5`，另外还带
 * `lodash-es` / `dompurify` / `nanoid` / `clsx` / `@floating-ui/dom`。
 * 在一个 React 19 工程里为了一个代码块把整套 Vue 运行时和五个工具库拖进来，
 * 代价远大于自己写这一百多行。这与 M5 表格半边不用 `table-block` 是同一条理由，
 * 取舍写在 ADR 0017。
 *
 * ## 它**不**负责高亮
 *
 * 语法着色仍然由 `plugin.ts` 的 `codeHighlightPlugin` 用 inline 装饰做
 * （与导出共用 `core/highlight/hljs.ts` 这一份实现）。这里只加几个
 * **不进入文档**的元素：行号槽、语言标签、图表区。装饰与 NodeView 互不干扰 ——
 * 装饰落在 `contentDOM` 里，这几个新元素都在 `contentDOM` 之外。
 *
 * ## Mermaid：图与源码是**同一个块的两种看法**
 *
 * 语言是 `mermaid` 时，块有两种形态（`dom.dataset.mode`）：
 *
 * - `diagram` —— 画好的图（或「正在渲染」/「语法错在哪」的说明）；
 * - `code` —— 源码，可编辑。
 *
 * 光标**在块里**就一定是 `code`：用户正在改的东西必须是他正在打的字，
 * 而不是一张每敲一个键就重画的图。光标离开后回到 `diagram`。
 *
 * 判据是「光标在不在里面」，而不是「有没有点过」：后者要用户自己记得切回来，
 * 结果是文档里留着一堆忘了切回图形的块。
 *
 * ## 为什么失败也要占着 `diagram` 那一格
 *
 * 渲染失败时退回 `code` 会把**失败原因藏起来**（图区被 CSS 隐藏了），
 * 用户只看到一段源码，不知道哪里不对。所以失败时仍然显示图区，
 * 里面是「原因 + 点一下改源码」。点它就进 `code`。
 *
 * ## 为什么「光标在不在里面」要插件推给视图
 *
 * NodeView 的 `update()` 只在**节点本身**变了时才被 ProseMirror 调用，
 * 光标挪动不会触发它（选区变化不是文档变化）。所以插件自己持有一份
 * 活跃视图表，在 `PluginView.update()` 里逐个同步 —— 那是唯一一个
 * 「每次状态变化都会被调用」的钩子。
 */

import type { Node as ProseNode } from '@milkdown/kit/prose/model';
import type { EditorState, Selection } from '@milkdown/kit/prose/state';
import { Plugin, PluginKey, TextSelection } from '@milkdown/kit/prose/state';
import type { EditorView, NodeView, ViewMutationRecord } from '@milkdown/kit/prose/view';
import { $prose } from '@milkdown/kit/utils';

import type { DocBlockPreview, SelectionRect } from '../../core/editor/types.js';
import { referencedDiagramSrc, parseDocBlock } from '../../core/docblock.js';
import { missingStencilLibs } from '../../core/drawio.js';
import { renderDrawio } from '../../core/drawio-render.js';
import { isMermaidLanguage } from '../../core/mermaid.js';
import { renderMermaid } from '../../core/mermaid-render.js';
import type { ResolvedTheme } from '../../core/theme/theme.js';
import { currentTheme } from '../../core/theme/theme.js';

/**
 * ProseMirror 的代码块节点名（对齐 preset-commonmark 的 schema）。
 *
 * 这个字符串在本模块目录里出现三处：这里（NodeView）、`plugin.ts`（高亮装饰）、
 * `selection.ts` 的 `NODE_FORMATS`（块格式识别）。三处都**静默**依赖它 ——
 * 上游改名不会有任何报错，只会让行号、高亮、块识别同时悄悄失效。
 * 改动时三个地方一起改。
 */
export const CODE_BLOCK = 'code_block';

/**
 * 引用卡片（`mode: reference`）能覆盖的 kind。
 *
 * `diagram` 有自己的渲染路径；`code` 的引用就该显示为代码；
 * `image` 与不认识的 kind 照旧显示原文 —— 展示层不知道它们是什么，
 * 把原文藏起来等于替用户做决定。
 */
const CARD_KINDS = new Set(['markdown', 'mindmap', 'text']);

/** 没有指定语言时，标签上显示的文字。 */
export const PLAIN_TEXT_LABEL = '纯文本';

const chromeKey = new PluginKey('levnote-code-block-chrome');

/** 用户点了语言标签。 */
export interface CodeBlockChromeRequest {
  /**
   * 发起请求的那个视图实例。
   *
   * 内核内部用它判断「面板对应的还是不是同一个代码块」。**不是文档坐标** ——
   * 坐标会随着用户在别处打字而失效，而实例身份不会。
   */
  readonly token: unknown;
  /**
   * **现在**去问这个代码块在文档里的位置；它已经不在了就返回 `null`。
   *
   * 刻意给一个函数而不是一个 `pos` 数字。面板开着的时候用户完全可能在别处打字 ——
   * 一旦那样，之前记下的那个数字就指向了别的节点，而 `setNodeAttribute`
   * **不检查节点类型**：它会安静地把语言写到那个不相干的节点上。
   *
   * 给函数就没有这个窗口：写的那一刻现问，拿到的一定是当下的位置。
   */
  readonly resolvePos: () => number | null;
  /** 当前语言；空串表示未指定。 */
  readonly language: string;
  /** 语言标签在视口里的矩形，外壳据此摆放面板。 */
  readonly anchor: SelectionRect;
}

export interface CodeBlockChromeHandlers {
  readonly openPicker: (request: CodeBlockChromeRequest) => void;
  /**
   * 把 `doc-block` 引用的 `src` 读成那份图表文件的原文（P5 第 25 项）。
   *
   * 与图片的 `resolve` 同一条分工：`src` 相对**当前文档目录**，只有外壳知道
   * 文档在哪、也只有它能读文件。返回 `null` 表示读不到 —— NodeView 据此
   * 在图区画带路径的说明，绝不留白。
   *
   * 不实现时 `doc-block` 的图表引用一律走「读不到」的说明路径。
   */
  readonly resolveDiagram?: (src: string) => Promise<string | null>;
  /**
   * 用户要**放大看**这个块的图。
   *
   * 交出去的是「此刻画在屏幕上的那一份」而不是让外壳再去渲染一遍 ——
   * 重新渲染既慢，又可能因为时序画出不一样的图。
   */
  readonly openViewer: (diagram: { svg: string; source: string }) => void;
  /**
   * 用户点击了一个 doc-block **引用卡片**（P6 第 28 项）——
   * 「去打开被引用的那份文档」。交出的是解析好的属性，
   * 「src 相对谁」由外壳用当前文档目录换算（与 `resolveDiagram` 同一条分工）。
   */
  readonly openDocBlock?: (ref: { src: string; kind: string; title: string | null }) => void;
  /**
   * 用户点了卡片上的「就地编辑」（P6 第 28 项 L2，ADR 0036）——
   * 「把被引用的文档在同一面板里打开来改」。与 `openDocBlock` 的差别只在
   * 导航方式（钻取 + 返回栈），编辑走的都是被引用文档自己的真实路径。
   */
  readonly editDocBlockInPlace?: (ref: { src: string; kind: string; title: string | null }) => void;
  /**
   * 卡片「预览」展开要的**结构摘要**（ADR 0036 决策 3）。
   *
   * 与 `resolveDiagram` 同一条分工：文件是外壳的领域，内核只画。
   * 读不到、解析失败以 `kind: 'error'` 作为**数据**回来，照样要画出来。
   */
  readonly resolveDocBlockPreview?: (ref: { src: string; kind: string }) => Promise<DocBlockPreview | null>;
  /**
   * 语言标签所在的代码块从文档里消失了（被删掉、或整篇被替换）。
   *
   * 必须上报：面板此时正对着一个已经不存在的代码块，而它自己不知道。
   * 用户看到的会是一个点下去没有任何反应的面板。
   */
  readonly closePicker: (token: unknown) => void;
}

/**
 * 这段代码文本占**几行**。
 *
 * 末尾的那个换行**不算一行**。理由：`<pre>` 里的行尾换行在渲染上不产生
 * 一个可见的行框（HTML 规范里 `pre` 元素的行尾换行本来就要被忽略，
 * 而我们这里靠 `white-space: pre` 才让它有意义）。多算一行的后果是
 * 行号槽比代码高一截，整个代码块跟着长高 —— 比少一个号显眼得多。
 *
 * 宁可少算也不能多算，所以这里**从少**：`"a\n"` 算 1 行，不是 2 行。
 * 空文本算 1 行（`<pre>` 里总有一个空行框）。
 */
export function lineCount(text: string): number {
  if (text === '') return 1;
  const segments = text.split('\n');
  return text.endsWith('\n') ? Math.max(1, segments.length - 1) : segments.length;
}

/** 把 `1..n` 铺成行号槽的内容。 */
function fillGutter(gutter: HTMLElement, lines: number): void {
  const next = document.createDocumentFragment();
  for (let index = 1; index <= lines; index += 1) {
    const item = document.createElement('div');
    item.className = 'qm-cb-num';
    item.textContent = String(index);
    next.appendChild(item);
  }
  gutter.replaceChildren(next);
}

class CodeBlockView implements NodeView {
  readonly dom: HTMLElement;
  readonly contentDOM: HTMLElement;

  private node: ProseNode;
  private readonly view: EditorView;
  private readonly getPos: () => number | undefined;
  private readonly getHandlers: () => CodeBlockChromeHandlers | undefined;
  /** 插件持有的活跃视图表 —— 「光标在不在里面」由插件推过来，见文件头。 */
  private readonly live: Set<CodeBlockView>;
  private readonly gutter: HTMLElement;
  private readonly badge: HTMLButtonElement;
  /** 图表区。非 mermaid 的块里它一直是空的，由 CSS 藏起来。 */
  private readonly diagram: HTMLElement;
  /** 「放大看」按钮。只在**真的画出了图**的时候才显示。 */
  private readonly zoom: HTMLButtonElement;
  /** 最近一次渲染成功的 SVG。`null` = 现在没有可放大的东西。 */
  private svg: string | null = null;
  /** 行号槽当前画了几行；-1 表示还没画过。 */
  private drawnLines = -1;
  /** 标签上当前显示的文字；null 表示还没画过。 */
  private drawnLabel: string | null = null;
  /**
   * 这个视图还在不在文档里。
   *
   * `destroy()` 之后 `getPos()` 就不可信了（ProseMirror 已经把它从视图树上摘掉），
   * 所以先把它置成 false 再上报，`currentPos()` 据此拒绝调用 `getPos()`。
   */
  private alive = true;

  /** 当前形态。与 `dom.dataset.mode` **同步维护** —— DOM 是渲染结果，不是真相。 */
  private mode: 'code' | 'diagram' = 'code';
  /** 光标是不是在这个块里。 */
  private editing = false;
  /**
   * 图区正在画（或已经画完）的那一份「源码 + 主题」。
   *
   * 两个都要记：前者决定「源码改了要不要重画」，后者决定「换了明暗要不要重画」
   * （只有 mermaid 在乎主题 —— drawio 的颜色写在图自己的 XML 里，见
   * `renderDrawio` 的注释）。`null` 表示图区是空的。
   */
  private painted:
    | { readonly kind: 'mermaid'; readonly source: string; readonly theme: ResolvedTheme }
    | { readonly kind: 'docblock'; readonly source: string }
    | { readonly kind: 'card'; readonly source: string }
    | null = null;
  /**
   * 交给「放大看」的那份**源**。mermaid 是块文本本身；doc-block 是被渲染的
   * 那份 XML（引用模式下与块文本不是一回事）。`null` = 现在没有图。
   */
  private diagramSource: string | null = null;
  /** 竞态令牌 —— 与 ImageView / MathView 同一个坑：先发的请求后回来会盖住新的。 */
  private token = 0;
  /** 引用卡片的摘要展开了吗。重画卡片（源码变过）就收起 —— 摘要对应的是那一刻。 */
  private previewOpen = false;
  /** 摘要请求的竞态令牌（与 `token` 分开：互不覆盖对方的在途请求）。 */
  private previewToken = 0;
  /** 当前挂着的摘要元素。`null` = 没展开。 */
  private previewEl: HTMLElement | null = null;

  constructor(
    node: ProseNode,
    view: EditorView,
    getPos: () => number | undefined,
    getHandlers: () => CodeBlockChromeHandlers | undefined,
    live: Set<CodeBlockView>,
  ) {
    this.node = node;
    this.view = view;
    this.getPos = getPos;
    this.getHandlers = getHandlers;
    this.live = live;

    const dom = document.createElement('div');
    dom.className = 'qm-cb';

    // 行号槽。`contenteditable=false` 让光标进不去，`aria-hidden` 让读屏软件
    // 跳过它 —— 行号是给眼睛看的，念出来只会打断正文
    const gutter = document.createElement('div');
    gutter.className = 'qm-cb-gutter';
    gutter.contentEditable = 'false';
    gutter.setAttribute('aria-hidden', 'true');

    // 代码本体。这一层是 contentDOM —— ProseMirror 把文本与装饰都渲染进这里。
    // 外面仍然套一个 `<pre>`：`white-space: pre` 与等宽字体是代码块的定义，
    // 换掉它会让 `.qm-doc pre code` 那套共用样式（导出也在用）落空
    const pre = document.createElement('pre');
    pre.className = 'qm-cb-pre';
    const code = document.createElement('code');
    code.className = 'qm-cb-code';
    pre.appendChild(code);

    // 图表区。`contenteditable=false` 让光标进不去：`contentDOM` 是那个
    // `<code>`，这里的一切都不是文档内容，只是「同一段源码的另一种看法」
    const diagram = document.createElement('div');
    diagram.className = 'qm-cb-diagram';
    diagram.contentEditable = 'false';
    /**
     * 点图 = 我要改源码。
     *
     * 挂 `mousedown` 而不是 `click`：不拦下 `mousedown`，浏览器会先把焦点挪走、
     * ProseMirror 会把光标放到块旁边 —— 等 `click` 到达时位置已经变了
     * （与公式、语言标签、浮层按钮同一个坑）。
     */
    diagram.addEventListener('mousedown', (event) => {
      event.preventDefault();
      event.stopPropagation();
      this.enterCode();
    });

    const zoom = document.createElement('button');
    zoom.type = 'button';
    zoom.className = 'qm-cb-zoom';
    zoom.contentEditable = 'false';
    zoom.textContent = '⤢';
    zoom.title = '放大看这张图';
    zoom.setAttribute('aria-label', '放大看这张图');
    zoom.hidden = true;
    // 同语言标签：不拦 mousedown，焦点会先离开编辑器、图会被重画一遍
    zoom.addEventListener('mousedown', (event) => event.preventDefault());
    zoom.addEventListener('click', () => this.requestViewer());

    const badge = document.createElement('button');
    badge.type = 'button';
    badge.className = 'qm-cb-lang';
    badge.contentEditable = 'false';
    // 不拦 mousedown 的话，焦点会先离开编辑器、选区塌掉，再轮到 click ——
    // 与浮层工具条同一个坑。这里更糟：代码块会因为焦点变化而重绘，
    // 标签的位置在 click 到达之前就变了
    badge.addEventListener('mousedown', (event) => event.preventDefault());
    badge.addEventListener('click', () => this.requestPicker());

    dom.append(gutter, pre, diagram, zoom, badge);

    this.dom = dom;
    this.contentDOM = code;
    this.gutter = gutter;
    this.diagram = diagram;
    this.zoom = zoom;
    this.badge = badge;
    this.dom.dataset['mode'] = 'code';

    this.renderChrome();

    // 自己先同步一次：插件的 PluginView 可能在**这个视图之前**就已经初始化完了，
    // 那时表里还没有它，谁也不会来通知它「光标其实不在里面」。
    // 顺序不能反 —— 光标本来就在里面时，先问一次才不会白白先画一张看不见的图
    live.add(this);
    this.syncSelection(view.state);
    this.ensureDiagram();
  }

  /** 这个块的语言是不是 Mermaid。 */
  private isMermaid(): boolean {
    return isMermaidLanguage(this.language());
  }

  /**
   * 这个块按**哪一种预览**画；`null` 表示它按普通代码块显示。
   *
   * - `mermaid` —— 源码即图（ADR 0020）；
   * - `docblock` —— 属性区声明了 `kind: diagram` 的 doc-block（ADR 0002）。
   *   只有声明了 kind 的才进图区：`kind: code` 的引用块就该显示为代码，
   *   硬画成图反而把用户写的东西藏起来了。
   * - `card` —— `mode: reference` 的 markdown / mindmap / text 引用
   *   （P6 第 28 项）。内容在**另一个文件**里，这里放一张说明卡片，
   *   点击打开那份文档；`mode: inline` 的内容就在块里，照旧显示为代码。
   *   `kind: code` 的引用也照旧显示为代码 —— 那是「这段代码在这里」的
   *   原样呈现，藏进卡片后面没有意义。
   */
  private diagramKind(): 'mermaid' | 'docblock' | 'card' | null {
    if (this.isMermaid()) return 'mermaid';
    if (this.language() === 'doc-block') {
      const parsed = parseDocBlock(this.node.textContent);
      if (parsed.attrs.kind === 'diagram') return 'docblock';
      if (parsed.attrs.kind === 'code') return null;
      if (parsed.attrs.mode === 'inline') return null;
      if (parsed.attrs.kind !== null && CARD_KINDS.has(parsed.attrs.kind)) return 'card';
      // 规范 §2.3 说「不认识的 kind 按 text 处理」，但那是**格式层**的兜底；
      // 展示层不知道未知的 kind 会不会真的是别的什么东西，宁可照旧显示原文
      return null;
    }
    return null;
  }

  /** 当前语言标识；空串表示未指定。 */
  private language(): string {
    const raw = this.node.attrs['language'];
    return typeof raw === 'string' ? raw : '';
  }

  /**
   * 行号与标签是**纯展示**，两边都从当前节点重算。
   *
   * ## 为什么两个 `if` 不能省
   *
   * 这个方法在**每一次文档变更**时都会被调用（`update` 里）。不判就重画的话，
   * 在一个一千行的代码块里敲一个字就要新建一千个行号元素 —— 而它们的内容
   * 一个都没变。加上判断之后，在代码块里打字产生的 DOM 变更是**零**，
   * MutationObserver 也就完全没有东西要处理。
   */
  private renderChrome(): void {
    const lines = lineCount(this.node.textContent);
    if (lines !== this.drawnLines) {
      this.drawnLines = lines;
      fillGutter(this.gutter, lines);
    }

    const language = this.language();
    const label = language === '' ? PLAIN_TEXT_LABEL : language;
    if (label === this.drawnLabel) return;
    this.drawnLabel = label;

    this.badge.textContent = label;
    this.badge.title = `代码语言：${label}（点击选择）`;
    this.badge.setAttribute('aria-label', `代码语言：${label}，点击选择`);
  }

  /**
   * 这个代码块**此刻**在文档里的位置；它已经不在了就返回 `null`。
   *
   * 用 `getPos()` 而不是自己记一个数字：`getPos` 是 ProseMirror 提供的，
   * 它按 DOM 节点现算位置，文档怎么变它都跟得上。
   */
  private currentPos(): number | null {
    if (!this.alive) return null;
    return this.getPos() ?? null;
  }

  private requestPicker(): void {
    const handlers = this.getHandlers();
    if (!handlers) return;

    const rect = this.badge.getBoundingClientRect();
    handlers.openPicker({
      token: this,
      resolvePos: () => this.currentPos(),
      language: this.language(),
      anchor: {
        top: rect.top,
        left: rect.left,
        bottom: rect.bottom,
        right: rect.right,
        width: rect.width,
        height: rect.height,
      },
    });
  }

  // -------------------------------------------------------------------------
  // 图表（Mermaid）
  // -------------------------------------------------------------------------

  /**
   * 让图区与「当前该显示什么」对齐；需要重画时就发起一次异步渲染。
   *
   * 这个方法在**每一次文档变更**时都会被调用（在别处打字也算），所以
   * 开头的判断是必需的：不判的话，用户在正文里每敲一个字，所有图表块
   * 都要重新渲染一遍。
   */
  ensureDiagram(): void {
    const source = this.node.textContent;
    const theme = currentTheme();
    const kind = this.diagramKind();

    if (kind === null || source.trim() === '') {
      // 不是图表块（或还没写内容）：把图区清空。
      // 空的图区由 CSS 藏起来，所以「清空」只是别留下上一次的图
      this.clearDiagram();
      this.syncMode();
      return;
    }

    /**
     * 正在改源码的时候**不画**。
     *
     * 用户每敲一个键都让整张图重新布局一遍纯属浪费 —— 而那张图他此刻根本
     * 看不见（光标在块里，显示的是源码）。更糟的是渲染是异步的：连着敲
     * 十个字符会排起十次渲染，而用户停下时看到的是**某一次中间的**结果。
     *
     * 光标离开的那一刻 `syncSelection` 会再叫一次，那时才画 ——
     * 也就是「只在要显示图的时候才为它付钱」。
     */
    if (this.editing) {
      this.syncMode();
      return;
    }

    const painted = this.painted;
    const paintedFresh =
      painted !== null &&
      painted.kind === kind &&
      painted.source === source &&
      // 只有 mermaid 在乎主题（颜色是渲染时定死的）；drawio 的颜色在 XML 里
      (painted.kind !== 'mermaid' || painted.theme === theme);
    if (paintedFresh) {
      this.syncMode();
      return;
    }

    if (kind === 'mermaid') void this.renderDiagram(source, theme);
    else if (kind === 'card') this.renderRefCard(source);
    else void this.renderDocBlock(source);
  }

  private clearDiagram(): void {
    if (this.painted === null && this.diagram.childElementCount === 0) return;
    this.painted = null;
    this.svg = null;
    this.diagramSource = null;
    this.diagram.classList.remove('is-loading', 'is-failed', 'is-ok');
    this.diagram.replaceChildren();
    this.syncZoom();
  }

  /**
   * 「放大」按钮只在**真的有一张图**时出现。
   *
   * 一直显示而点了没反应（正在渲染 / 语法错 / 不是图表块）是典型的
   * 「假按钮」—— 用户会以为软件坏了。
   */
  private syncZoom(): void {
    this.zoom.hidden = this.svg === null;
  }

  private requestViewer(): void {
    const svg = this.svg;
    if (svg === null) return;
    this.getHandlers()?.openViewer({ svg, source: this.diagramSource ?? this.node.textContent });
  }

  private async renderDiagram(source: string, theme: ResolvedTheme): Promise<void> {
    // 同步记下「正在画的是这一份」：于是并发的第二次 `ensureDiagram`
    // 不会重复发起，而 `syncMode()` 也知道现在该显示图区（哪怕是「正在渲染」）
    this.painted = { kind: 'mermaid', source, theme };
    this.svg = null;
    this.diagramSource = source;
    this.paintDiagramNote('正在渲染图表…', 'is-loading');
    this.syncZoom();
    this.syncMode();

    const mine = (this.token += 1);
    const result = await renderMermaid(source, theme).catch((thrown: unknown) => ({
      ok: false as const,
      message: thrown instanceof Error ? thrown.message : String(thrown),
    }));

    // 过期（源码又变了 / 主题又变了）或已销毁：丢掉这次结果
    if (!this.alive || mine !== this.token) return;

    if (result.ok) {
      this.diagram.classList.remove('is-loading', 'is-failed');
      this.diagram.classList.add('is-ok');
      // mermaid 在 `securityLevel: 'strict'` 下会自己用 DOMPurify 清洗产出
      // （见 `core/mermaid-render.ts`）；`innerHTML` 还有一个好处 —— 插进来的
      // `<script>` 不会执行，而 `appendChild` 一个解析好的节点会
      this.diagram.innerHTML = result.svg;
      this.svg = result.svg;
    } else {
      this.paintDiagramNote(result.message, 'is-failed');
    }
    this.syncZoom();
    this.syncMode();
  }

  /**
   * 画一个 `doc-block` 图表引用（P5 第 25 项）。
   *
   * 与 mermaid 那条路的差别只有**「源」从哪来**：
   *
   * - `mode: reference` —— 去外部文件读（经 `resolveDiagram` 契约，只有外壳
   *   知道路径与怎么读）；
   * - `mode: inline` —— 载荷本身就是文档（规范 §2.2），不需要读任何东西。
   *
   * 解析失败、读不到、渲染失败，全部落成图区里**带原因的一句话** ——
   * 这个块的降级形态是「显示源码」，而失败时退回源码会把原因藏起来
   * （图区被 CSS 隐藏了），与 mermaid 失败时占着图区的理由相同。
   */
  private async renderDocBlock(source: string): Promise<void> {
    this.painted = { kind: 'docblock', source };
    this.svg = null;
    this.syncZoom();
    this.syncMode();

    const mine = (this.token += 1);
    const parsed = parseDocBlock(source);
    const src = referencedDiagramSrc(parsed);
    const inline = parsed.attrs.mode === 'inline';

    let xml: string | null = null;
    let failed: string | null = null;

    if (parsed.errors.includes('MISSING_TERMINATOR')) {
      failed = 'doc-block 缺少结束行 —— 属性区和载荷之间必须有一整行「---」。';
    } else if (src !== null) {
      this.paintDiagramNote(`正在读取 ${src} …`, 'is-loading');
      xml = (await this.getHandlers()?.resolveDiagram?.(src)) ?? null;
      if (xml === null) failed = `读不到引用的文件：${src}`;
    } else if (inline) {
      // 载荷全是空白与「没有载荷」是同一件事 —— 都要把图该写在哪儿说出来。
      // 不留给 renderDrawio 的空内容短路：那句「内容是空的」没有这句可操作
      if (parsed.payload !== null && parsed.payload.trim() !== '') {
        xml = parsed.payload;
      } else {
        failed = 'inline 载荷是空的 —— 图的内容应该写在「---」之后。';
      }
    } else {
      failed = '声明了 kind: diagram，但没有 src 引用，也不是 inline 载荷。';
    }

    if (!this.alive || mine !== this.token) return;

    if (failed !== null || xml === null) {
      this.paintDiagramNote(failed ?? '读不到引用的文件。', 'is-failed');
      this.syncZoom();
      this.syncMode();
      return;
    }

    // 「放大看」交出去的源是**这份 XML**（引用模式下与块文本不是一回事）
    this.diagramSource = xml;

    const host = document.createElement('div');
    host.className = 'qm-cb-drawio';
    // 在渲染**之前**算缺库：这是纯文本分析，与渲染并发与否无关
    const missing = missingStencilLibs(xml);
    // host 必须先挂进文档再渲染：mxGraph 量容器尺寸来决定画布大小，
    // 游离节点量出来是 0 —— 与 mermaid 的「容器必须参与布局」是同一类问题
    this.diagram.replaceChildren(host);

    const result = await renderDrawio(xml, host);
    if (!this.alive || mine !== this.token) return;

    if (!result.ok) {
      this.paintDiagramNote(result.message, 'is-failed');
      this.syncZoom();
      this.syncMode();
      return;
    }

    this.diagram.classList.remove('is-loading', 'is-failed');
    this.diagram.classList.add('is-ok');
    if (missing.length > 0) {
      // stencil 缺失是**静默降级**（mxGraph 画成矩形、标签照旧）——
      // 所以必须由我们把这件事说出来。硬约束 2 在查看器上的落点
      const note = document.createElement('div');
      note.className = 'qm-cb-note qm-cb-note-warn';
      note.textContent = `未内置的形状库：${missing.join('、')} —— 相关形状已按矩形显示。`;
      this.diagram.appendChild(note);
    }
    this.svg = result.svg;
    this.syncZoom();
    this.syncMode();
  }

  /** 画「渲染失败 / 正在渲染」这类说明，而不是一张图。 */
  private paintDiagramNote(text: string, state: 'is-loading' | 'is-failed'): void {
    this.diagram.classList.remove('is-loading', 'is-failed', 'is-ok');
    this.diagram.classList.add(state);

    const note = document.createElement('div');
    note.className = 'qm-cb-note';
    note.textContent = text;

    const children: Node[] = [note];
    if (state === 'is-failed') {
      const hint = document.createElement('div');
      hint.className = 'qm-cb-note-hint';
      hint.textContent = '点一下改源码';
      children.push(hint);
    }
    this.diagram.replaceChildren(...children);
  }

  /**
   * 画一张 doc-block **引用卡片**（P6 第 28 项）。
   *
   * 与 `renderDocBlock` 的差别：那里要等异步的文件读取与渲染，这里**同步**
   * 就画得完 —— 卡片只描述「这份内容在哪个文件里」，不替用户打开它。
   * 所以竞态令牌照记不误（形状与另外两条渲染路径一致），但成功路径上
   * 用不到它。
   *
   * 属性区有问题时如实画成带原因的说明，**绝不装作正常**：
   * 缺 `src` 的引用点开了也是空白，那是「静默降级」。
   *
   * 卡片是 `div` 而不是整块可点的 `button`（第二版起）：动作到三个之后
   * button 里不能再嵌 button —— 主区（点 = 打开）+ 动作行（预览 / 就地编辑）
   * 分开画（ADR 0036 决策 4）。动作按钮**只在对应能力真的挂着时才画**：
   * 一个点下去没有反应的按钮是「假按钮」，比不画更糟。
   */
  private renderRefCard(source: string): void {
    this.painted = { kind: 'card', source };
    this.svg = null;
    this.syncZoom();
    // 重画 = 源码变过。展开着的摘要对应的是上一份属性，收起（想看再展开，
    // 摘要本来就是展开即读的）
    this.previewOpen = false;
    this.previewToken += 1;

    const parsed = parseDocBlock(source);
    const attrs = parsed.attrs;

    if (parsed.errors.includes('MISSING_TERMINATOR')) {
      this.paintDiagramNote('doc-block 缺少结束行 —— 属性区和载荷之间必须有一整行「---」。', 'is-failed');
      this.syncMode();
      return;
    }
    if (attrs.src === null || attrs.src === '') {
      this.paintDiagramNote('引用缺少 src —— 这里应该写被引用文件的相对路径。', 'is-failed');
      this.syncMode();
      return;
    }

    this.diagram.classList.remove('is-loading', 'is-failed');
    this.diagram.classList.add('is-ok');

    const ref = { src: attrs.src, kind: attrs.kind ?? 'text', title: attrs.title };
    const handlers = this.getHandlers();

    const card = document.createElement('div');
    card.className = 'qm-docref-card';

    // 主区：整块可点 = 打开被引用的文档（与第一版行为一致）。
    // 硬约束 11 的正路：监听挂在内核自己造的元素上，外壳经契约收事件；
    // 点源码区仍然是进源码编辑，互不干扰。
    const main = document.createElement('button');
    main.type = 'button';
    main.className = 'qm-docref-main';

    const badge = document.createElement('span');
    badge.className = 'qm-docref-kind';
    badge.textContent = ref.kind;
    main.appendChild(badge);

    const title = document.createElement('span');
    title.className = 'qm-docref-title';
    title.textContent = ref.title ?? ref.src;
    main.appendChild(title);

    const from = document.createElement('span');
    from.className = 'qm-docref-src';
    from.textContent = `来自 ${ref.src}`;
    main.appendChild(from);

    main.addEventListener('click', (event) => {
      event.preventDefault();
      // 卡片在图区里，图区只有「光标不在块里」时显示；点击不会抢走编辑焦点。
      // 派发时**现问** getHandlers（与第一版一致）：渲染时的快照只用来决定
      // 画不画按钮，不拿来派发
      this.getHandlers()?.openDocBlock?.(ref);
    });
    card.appendChild(main);

    const actions = document.createElement('div');
    actions.className = 'qm-docref-actions';

    if (handlers?.resolveDocBlockPreview !== undefined) {
      const preview = document.createElement('button');
      preview.type = 'button';
      preview.className = 'qm-docref-action';
      preview.textContent = '预览';
      // 坑 9 的配套：不拦掉 mousedown，焦点先挪到按钮上，ProseMirror 的选区
      // 随之丢掉 —— 卡片虽然在「非编辑」形态，同一条纪律不豁免
      preview.addEventListener('mousedown', (event) => event.preventDefault());
      preview.addEventListener('click', (event) => {
        event.preventDefault();
        // 摘要只要 src + kind（title 是卡片展示用的）—— 契约的字段面就给它多少
        this.togglePreview({ src: ref.src, kind: ref.kind });
      });
      actions.appendChild(preview);
    }

    if (handlers?.editDocBlockInPlace !== undefined) {
      const edit = document.createElement('button');
      edit.type = 'button';
      edit.className = 'qm-docref-action';
      edit.textContent = '就地编辑';
      edit.addEventListener('mousedown', (event) => event.preventDefault());
      edit.addEventListener('click', (event) => {
        event.preventDefault();
        this.getHandlers()?.editDocBlockInPlace?.(ref);
      });
      actions.appendChild(edit);
    }

    if (actions.childElementCount > 0) card.appendChild(actions);

    this.diagram.replaceChildren(card);
    this.syncMode();
  }

  /**
   * 展开 / 收起引用摘要。
   *
   * 摘要是异步的（外壳要读文件），所以竞态令牌照上 —— 收起再展开、
   * 或者摘要没回来卡片就重画了，先发的旧响应都不能盖住新状态。
   * 等待期间先画「正在读」：点了立刻有反应，比转圈半天什么都看不见强。
   */
  private togglePreview(ref: { src: string; kind: string }): void {
    const handlers = this.getHandlers();
    if (handlers === undefined) return;

    if (this.previewOpen) {
      this.previewOpen = false;
      this.previewToken += 1;
      this.paintPreview();
      return;
    }
    if (handlers.resolveDocBlockPreview === undefined) return;

    this.previewOpen = true;
    const mine = (this.previewToken += 1);
    this.paintPreview();

    void handlers
      .resolveDocBlockPreview(ref)
      .catch((thrown: unknown): DocBlockPreview => ({
        kind: 'error',
        reason: thrown instanceof Error ? thrown.message : String(thrown),
      }))
      .then((preview) => {
        if (!this.alive || mine !== this.previewToken) return;
        // null = 外壳此刻给不出摘要（按钮画了之后能力被摘掉）。收起比
        // 画一个空的框诚实
        if (preview === null) {
          this.previewOpen = false;
          this.paintPreview();
          return;
        }
        this.paintPreview(preview);
      });
  }

  /**
   * 按当前 `previewOpen` 画（或摘掉）摘要区。
   *
   * 摘要挂在卡片**下面**（同一个 `diagram` 容器里），收起就是把它摘掉 ——
   * 卡片本身不动，展开收起不重画整张卡片。
   */
  private paintPreview(preview?: DocBlockPreview): void {
    if (this.previewEl !== null) {
      this.previewEl.remove();
      this.previewEl = null;
    }
    if (!this.previewOpen) return;

    const el = document.createElement('div');
    el.className = 'qm-docref-preview';

    if (preview === undefined) {
      el.classList.add('is-loading');
      el.textContent = '正在读摘要……';
    } else if (preview.kind === 'error') {
      el.classList.add('is-failed');
      el.textContent = preview.reason;
    } else if (preview.kind === 'markdown') {
      const meta = document.createElement('div');
      meta.className = 'qm-docref-preview-meta';
      meta.textContent = `约 ${preview.wordCount} 字`;
      el.appendChild(meta);
      if (preview.headings.length === 0) {
        const none = document.createElement('div');
        none.className = 'qm-docref-preview-empty';
        none.textContent = '没有标题';
        el.appendChild(none);
      } else {
        const list = document.createElement('ul');
        list.className = 'qm-docref-preview-outline';
        for (const item of preview.headings) {
          const li = document.createElement('li');
          // 层级缩进直接写在行内：六级标题不值得为它写六条 CSS 规则
          li.style.paddingLeft = `${(item.level - 1) * 14}px`;
          li.textContent = item.text;
          list.appendChild(li);
        }
        el.appendChild(list);
      }
    } else if (preview.kind === 'text') {
      const meta = document.createElement('div');
      meta.className = 'qm-docref-preview-meta';
      meta.textContent = `共 ${preview.lineCount} 行`;
      el.appendChild(meta);
      const snippet = document.createElement('pre');
      snippet.className = 'qm-docref-preview-snippet';
      snippet.textContent = preview.snippet;
      el.appendChild(snippet);
    } else {
      const title = document.createElement('div');
      title.className = 'qm-docref-preview-meta';
      title.textContent = preview.rootTitle;
      el.appendChild(title);
      const meta = document.createElement('div');
      meta.className = 'qm-docref-preview-empty';
      meta.textContent = `${preview.nodeCount} 个节点`;
      el.appendChild(meta);
    }

    this.diagram.appendChild(el);
    this.previewEl = el;
  }

  /** 当前该显示图还是源码。 */
  private wantDiagram(): boolean {
    if (this.editing) return false;
    const source = this.node.textContent;
    if (source.trim() === '') return false;
    const painted = this.painted;
    if (painted === null || painted.kind !== this.diagramKind() || painted.source !== source) {
      return false;
    }
    if (painted.kind === 'mermaid') return painted.theme === currentTheme();
    return true;
  }

  private syncMode(): void {
    const next = this.wantDiagram() ? 'diagram' : 'code';
    if (next === this.mode) return;
    this.mode = next;
    this.dom.dataset['mode'] = next;
  }

  /** 点图之后：把光标放进源码里（`syncMode` 会因此切到 `code`）。 */
  private enterCode(): void {
    const pos = this.currentPos();
    if (pos === null) return;

    const { state } = this.view;
    // 落在块**内部的第一个合法位置**：`pos + 1` 是 code_block 内容的起点。
    // 用 `TextSelection.near` 而不是直接 `create`：块可能是空的，
    // 那时 `pos + 1` 不是一个能放文本光标的点
    const selection = TextSelection.near(state.doc.resolve(pos + 1));
    // 只改选区、不改文档 —— 所以不需要 `USER_EDIT_META`（那条是给「改了文档
    // 但没经过键盘事件」的操作用的）。这一步没有产生任何输入事件，
    // 但它也不产生文档变更，本来就不该让文档变脏
    this.view.dispatch(state.tr.setSelection(selection));
    this.view.focus();
  }

  /**
   * 插件在**每次状态变化**时推过来的一句「状态变了」。
   *
   * 公开给插件而不是自己订阅，是因为 ProseMirror 只给 NodeView 两个钩子
   * （`update` / `destroy`），都不是「状态变了」。
   *
   * ## 为什么图区的对齐也放在这里，而不是 `update()` 里
   *
   * 一个事务里 ProseMirror **先**跑 `nodeView.update()`、**后**跑
   * `PluginView.update()`。所以「在块里打第一个字」时，`update()` 看到的
   * `this.editing` 还是**上一个状态**的（光标在块外面）—— 在那里判断
   * 「要不要画图」会因为这一帧的滞后而多画一张没人看的图。
   *
   * 放在这里就没有滞后：插件推过来的就是**刚刚应用完**的那个状态。
   */
  syncSelection(state: EditorState): void {
    if (!this.alive) return;
    this.editing = this.containsSelection(state.selection);
    // `ensureDiagram` 内部会调 `syncMode`，所以不需要再调一次
    this.ensureDiagram();
  }

  /**
   * 选区是不是**完全在**这个块里面。
   *
   * 两端都取严格内部（`> pos` 且 `< pos + nodeSize`）：贴在块边界上的光标
   * 属于「块外面」—— 那时按回车是新建段落，而不是往代码里加一行。
   */
  private containsSelection(selection: Selection): boolean {
    const pos = this.currentPos();
    if (pos === null) return false;
    return selection.from > pos && selection.to < pos + this.node.nodeSize;
  }

  // -------------------------------------------------------------------------
  // ProseMirror 回调
  // -------------------------------------------------------------------------

  /**
   * ProseMirror 把变更同步进 `contentDOM` 时会调它。
   *
   * 返回 `true` 表示「我能就地更新」—— 于是 `contentDOM` 里的文本由
   * ProseMirror 自己换掉，我们只负责把行号与标签重画一遍。
   * 返回 `false` 会让 ProseMirror 把整个视图销毁重建，那样每敲一个字
   * 语言标签都会闪一下。
   *
   * 这里**不**调 `ensureDiagram` —— 图区的对齐统一在 `syncSelection` 里做，
   * 因为 `update` 跑在插件视图之前，那时 `editing` 还是上一帧的。
   */
  update(node: ProseNode): boolean {
    if (node.type.name !== CODE_BLOCK) return false;
    this.node = node;
    this.renderChrome();
    return true;
  }

  /**
   * 哪些 DOM 变更**不算**文档变更。
   *
   * 我们自己往行号槽与标签里写的每一个字符都会触发 MutationObserver。
   * 不声明忽略的话，ProseMirror 会把它当成用户在输入，试图把行号
   * **读进文档里** —— 保存出来的 Markdown 会多出一列数字。
   *
   * 判据只有一条：**不在 `contentDOM` 里的都不算**。属性变更也一律忽略：
   * 我们改的都是 `class` / `title` / `aria-*` 这类纯展示属性。
   */
  ignoreMutation(mutation: ViewMutationRecord): boolean {
    if (mutation.type === 'attributes') return true;
    return !this.contentDOM.contains(mutation.target);
  }

  /**
   * 标签、放大按钮与图表区上的事件不要让 ProseMirror 处理。
   *
   * 只挡这三处：行号槽和代码本体都要保持编辑器的默认行为
   * （点行号槽应该能把光标放进代码块，而不是被吞掉）。
   */
  stopEvent(event: Event): boolean {
    const target = event.target;
    if (!(target instanceof Node)) return false;
    return (
      this.badge.contains(target) || this.zoom.contains(target) || this.diagram.contains(target)
    );
  }

  destroy(): void {
    // 先置 false 再上报：上报出去之后内核可能立刻来问位置，
    // 而那时这个视图已经不在文档里了
    this.alive = false;
    this.live.delete(this);
    this.getHandlers()?.closePicker(this);
  }
}

/**
 * 装上代码块的 NodeView。
 *
 * 回调用「取回调的函数」而不是直接传：插件实例在编辑器创建时就固定了，
 * 而 React 那边每次渲染都会换一个新的回调函数（与 `cursorReporterPlugin` 同一条理由）。
 *
 * ## 插件为什么还要持有一份活跃视图表
 *
 * ProseMirror 给 NodeView 的只有「节点变了」和「要销毁了」两个钩子，
 * 都没有「选区变了」。而图/源码的切换完全由选区决定，所以插件在
 * `PluginView.update()`（唯一每帧都跑的钩子）里逐个推过去。
 *
 * ## 为什么要盯 `data-theme`
 *
 * 图的配色是**渲染时**定死的（mermaid 把颜色写进了 SVG），换个明暗必须重画。
 * 而「换主题」这件事不会经过编辑器 —— 没人派发事务，插件也就收不到通知。
 * 所以直接盯根元素上 `applyTheme` 写的那个属性（`core/theme/theme.ts`）：
 * 那是**唯一**一个一定会变的地方。
 */
export function codeBlockChromePlugin(
  getHandlers: () => CodeBlockChromeHandlers | undefined,
) {
  return $prose(() => {
    /** 活着的视图。`destroy()` 里摘掉 —— 不摘就会把整棵视图树都留在内存里。 */
    const live = new Set<CodeBlockView>();

    const themeWatcher =
      typeof MutationObserver === 'undefined'
        ? null
        : new MutationObserver(() => {
            for (const view of live) view.ensureDiagram();
          });

    return new Plugin({
      key: chromeKey,
      props: {
        nodeViews: {
          [CODE_BLOCK]: (node, view, getPos) =>
            new CodeBlockView(node, view, getPos, getHandlers, live),
        },
      },
      view: () => {
        themeWatcher?.observe(document.documentElement, {
          attributes: true,
          attributeFilter: ['data-theme'],
        });

        return {
          update: (current) => {
            for (const view of live) view.syncSelection(current.state);
          },
          destroy: () => {
            themeWatcher?.disconnect();
            live.clear();
          },
        };
      },
    });
  });
}
