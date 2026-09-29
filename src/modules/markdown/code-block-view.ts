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
 * （与导出共用 `core/highlight/hljs.ts` 这一份实现）。这里只加两个
 * **不进入文档**的元素：行号槽与语言标签。装饰与 NodeView 互不干扰 ——
 * 装饰落在 `contentDOM` 里，两个新元素都在 `contentDOM` 之外。
 */

import type { Node as ProseNode } from '@milkdown/kit/prose/model';
import { Plugin, PluginKey } from '@milkdown/kit/prose/state';
import type { EditorView, NodeView, ViewMutationRecord } from '@milkdown/kit/prose/view';
import { $prose } from '@milkdown/kit/utils';

import type { SelectionRect } from '../../core/editor/types.js';

/**
 * ProseMirror 的代码块节点名（对齐 preset-commonmark 的 schema）。
 *
 * 这个字符串在本模块目录里出现三处：这里（NodeView）、`plugin.ts`（高亮装饰）、
 * `selection.ts` 的 `NODE_FORMATS`（块格式识别）。三处都**静默**依赖它 ——
 * 上游改名不会有任何报错，只会让行号、高亮、块识别同时悄悄失效。
 * 改动时三个地方一起改。
 */
export const CODE_BLOCK = 'code_block';

/** 没有指定语言时，标签上显示的文字。 */
export const PLAIN_TEXT_LABEL = '纯文本';

const chromeKey = new PluginKey('qingmo-code-block-chrome');

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
  private readonly getPos: () => number | undefined;
  private readonly getHandlers: () => CodeBlockChromeHandlers | undefined;
  private readonly gutter: HTMLElement;
  private readonly badge: HTMLButtonElement;
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

  constructor(
    node: ProseNode,
    _view: EditorView,
    getPos: () => number | undefined,
    getHandlers: () => CodeBlockChromeHandlers | undefined,
  ) {
    this.node = node;
    this.getPos = getPos;
    this.getHandlers = getHandlers;

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

    const badge = document.createElement('button');
    badge.type = 'button';
    badge.className = 'qm-cb-lang';
    badge.contentEditable = 'false';
    // 不拦 mousedown 的话，焦点会先离开编辑器、选区塌掉，再轮到 click ——
    // 与浮层工具条同一个坑。这里更糟：代码块会因为焦点变化而重绘，
    // 标签的位置在 click 到达之前就变了
    badge.addEventListener('mousedown', (event) => event.preventDefault());
    badge.addEventListener('click', () => this.requestPicker());

    dom.append(gutter, pre, badge);

    this.dom = dom;
    this.contentDOM = code;
    this.gutter = gutter;
    this.badge = badge;

    this.renderChrome();
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

  /**
   * ProseMirror 把变更同步进 `contentDOM` 时会调它。
   *
   * 返回 `true` 表示「我能就地更新」—— 于是 `contentDOM` 里的文本由
   * ProseMirror 自己换掉，我们只负责把行号与标签重画一遍。
   * 返回 `false` 会让 ProseMirror 把整个视图销毁重建，那样每敲一个字
   * 语言标签都会闪一下。
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
   * 标签上的事件不要让 ProseMirror 处理。
   *
   * 只挡标签：行号槽和代码本体都要保持编辑器的默认行为
   * （点行号槽应该能把光标放进代码块，而不是被吞掉）。
   */
  stopEvent(event: Event): boolean {
    const target = event.target;
    return target instanceof Node && this.badge.contains(target);
  }

  destroy(): void {
    // 先置 false 再上报：上报出去之后内核可能立刻来问位置，
    // 而那时这个视图已经不在文档里了
    this.alive = false;
    this.getHandlers()?.closePicker(this);
  }
}

/**
 * 装上代码块的 NodeView。
 *
 * 回调用「取回调的函数」而不是直接传：插件实例在编辑器创建时就固定了，
 * 而 React 那边每次渲染都会换一个新的回调函数（与 `cursorReporterPlugin` 同一条理由）。
 */
export function codeBlockChromePlugin(
  getHandlers: () => CodeBlockChromeHandlers | undefined,
) {
  return $prose(
    () =>
      new Plugin({
        key: chromeKey,
        props: {
          nodeViews: {
            [CODE_BLOCK]: (node, view, getPos) => new CodeBlockView(node, view, getPos, getHandlers),
          },
        },
      }),
  );
}
