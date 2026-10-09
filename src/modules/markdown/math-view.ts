/**
 * 公式的 NodeView：渲染、报错、**在原位改**。
 *
 * ## 为什么公式需要 NodeView
 *
 * 公式在 Markdown 里是一段 `$…$` 源码，在屏幕上应该是排版好的样子。
 * 「源码 → 排版」这一步只能由我们做：
 *
 * - `toDOM` 是**同步**的，而 KaTeX 是动态加载的（见 `core/katex.ts`）；
 * - 渲染可能失败（用户写了个不存在的命令），失败必须**看得见**；
 * - 用户得能改它 —— 而改的地方最好就是公式本身所在的位置。
 *
 * ## DOM 结构与导出侧**同构**
 *
 * 根元素是 `.qm-math`，里面直接放 KaTeX 的输出（`.katex` / `.katex-display`）。
 * 导出侧把 `toDOM` 生成的壳里的内容换成同一份 KaTeX 输出，于是
 * 「编辑器里的公式」与「导出文件里的公式」是同一棵 DOM —— 共用一份 CSS
 * （`markdown.css` 与 `document-styles.css` 里 `.qm-math` 那一段），
 * 观感才可能一致。多包一层 `.qm-math-body` 之类的东西就会让两边分叉。
 *
 * ## 原位编辑
 *
 * 点一下公式，它的位置换成一个输入框；确认后写回节点。整个交互**不经过外壳** ——
 * 不需要新的浮层、不需要新的回调契约，因为「公式该长什么样」这件事的内核
 * 完全知道。这与代码块的语言选择器（要问外壳「有哪些语言」）不同。
 *
 * ## 提交时必须自己声明「这是用户发起的编辑」
 *
 * 输入框里的按键被 `stopEvent` 挡住了，ProseMirror 的 DOM 观察者与
 * `markdownEmitPlugin` 的 `beforeinput` 监听**都收不到**（见
 * `prosemirror-view` 的 `eventBelongsToView`）。少了 `USER_EDIT_META`，
 * 改完公式文档不会变脏，用户按 Ctrl+S 写回磁盘的是**改之前**的公式，
 * 而屏幕上显示的是新的。这是本项目第六次遇到这条路径 —— 前五次见
 * `setup.ts` 的 `withUserEdit`。
 */

import type { Node as ProseNode } from '@milkdown/kit/prose/model';
import { Plugin, PluginKey } from '@milkdown/kit/prose/state';
import type { EditorView, NodeView } from '@milkdown/kit/prose/view';
import { $prose } from '@milkdown/kit/utils';

import { renderTex } from '../../core/katex.js';
import { texProblem, type MathKind } from '../../core/math.js';
import { MATH_BLOCK, MATH_INLINE, TEX_ATTR } from './math.js';
import { USER_EDIT_META } from './plugin.js';

const mathViewKey = new PluginKey('levnote-math-view');

/** `attrs` 是 `any`，取值一律收窄。 */
function readTex(attrs: ProseNode['attrs']): string {
  const raw: unknown = attrs[TEX_ATTR];
  return typeof raw === 'string' ? raw : '';
}

/** 公式在源文档里的写法，用来在出错/空公式时给用户看。 */
function sourceOf(tex: string, kind: MathKind): string {
  return kind === 'block' ? `$$\n${tex}\n$$` : `$${tex}$`;
}

/** 一个不可编辑的说明元素。它**不是文档内容**，所以必须 `contenteditable=false`。 */
function noteElement(className: string, text: string): HTMLElement {
  const el = document.createElement('span');
  el.className = className;
  el.contentEditable = 'false';
  el.textContent = text;
  return el;
}

/**
 * 公式的视图。
 *
 * 状态用根元素上的类名表示（CSS 负责画），同时**在字段里也记一份**：
 * 「现在是不是在编辑」这类问题必须能直接问，而不是从 `classList` 反推 ——
 * DOM 是渲染结果，不是真相。
 */
class MathView implements NodeView {
  readonly dom: HTMLElement;

  private node: ProseNode;
  private readonly kind: MathKind;
  private readonly getPos: () => number | undefined;
  private readonly view: EditorView;

  /** 已经渲染（或正在渲染）的那个值；`null` 表示还没渲染过。 */
  private painted: string | null = null;
  /**
   * 竞态令牌。用户可能连着改两次公式，而渲染是异步的 ——
   * 先发的请求后回来会把**过期的公式**盖到新的上面（与 `ImageView` 同一个坑）。
   */
  private token = 0;
  private alive = true;
  private editing = false;
  /** 编辑中的输入框；不在编辑时为 `null`。 */
  private input: HTMLInputElement | HTMLTextAreaElement | null = null;

  constructor(
    node: ProseNode,
    view: EditorView,
    getPos: () => number | undefined,
    kind: MathKind,
  ) {
    this.node = node;
    this.view = view;
    this.getPos = getPos;
    this.kind = kind;

    const block = kind === 'block';
    this.dom = document.createElement(block ? 'div' : 'span');
    this.dom.className = block ? 'qm-math qm-math-block' : 'qm-math qm-math-inline';

    /**
     * 点一下进入编辑。
     *
     * 挂在 `mousedown` 而不是 `click`：不拦下 `mousedown`，浏览器会先把焦点
     * 挪走、ProseMirror 会把光标放到公式旁边 —— 等 `click` 到达时位置已经变了。
     * 这与代码块的语言标签、浮层工具条上的按钮是同一个坑。
     *
     * 监听挂在我们**自己造的元素**上，而不是让外壳去编辑器容器上做事件委托：
     * 后者要求外壳认识 `qm-math` 这个类名，而外壳不许认识内核。
     */
    this.dom.addEventListener('mousedown', (event) => {
      if (this.editing) return;
      event.preventDefault();
      event.stopPropagation();
      this.enterEdit();
    });

    const tex = readTex(node.attrs);
    if (tex.trim() === '') {
      // 空公式没什么可渲染的。直接进入编辑 —— 这也是「斜杠菜单插入公式」
      // 之后用户看到的状态：一个等着他打字的输入框。
      //
      // 聚焦**推迟到这一轮之后**：这个构造函数是在 `view.dispatch` 里跑的，
      // 紧接着 ProseMirror 还会走一遍 `selectionToDOM` 把焦点抢回编辑器。
      // 同步聚焦会被那一步覆盖，用户看到的是一个没聚焦的输入框
      this.enterEdit();
      queueMicrotask(() => {
        if (this.alive && this.editing) this.focusInput();
      });
    } else {
      void this.render(tex);
    }
  }

  // -------------------------------------------------------------------------
  // 渲染
  // -------------------------------------------------------------------------

  private async render(tex: string): Promise<void> {
    this.painted = tex;
    const mine = (this.token += 1);
    this.dom.dataset['tex'] = tex;

    if (tex.trim() === '') {
      this.paint('empty');
      return;
    }

    this.paint('loading');

    let result;
    try {
      result = await renderTex(tex, this.kind === 'block');
    } catch (thrown) {
      // `renderTex` 内部已经把失败收敛成返回值，这里只是不让一个未捕获的
      // 拒绝变成没人处理的 Promise
      result = { ok: false as const, message: String(thrown) };
    }

    // 过期（值又变了）或已销毁：丢掉这次结果
    if (!this.alive || mine !== this.token) return;

    if (result.ok) {
      this.dom.classList.remove('is-loading', 'is-failed', 'is-empty');
      this.dom.classList.add('is-loaded');
      this.dom.replaceChildren();
      // KaTeX 的输出是我们自己生成的，且 `trust: false` 关掉了
      // `\href` / `\url` / `\htmlStyle` 这类能注入的写法（见 core/katex.ts）
      this.dom.innerHTML = result.html;
      return;
    }

    this.paint('failed', result.message);
  }

  /**
   * 画「非渲染结果」的那几种状态。
   *
   * 失败时**同时**给出错原因和公式源码：用户改错了一个命令，他需要知道
   * 错在哪（原因），也需要看见自己写的是什么（源码）。只给一个空白或一个
   * 红色图标，等于把「为什么」藏起来。
   */
  private paint(mode: 'loading' | 'empty' | 'failed', message?: string): void {
    this.dom.classList.remove('is-loading', 'is-loaded', 'is-failed', 'is-empty');
    this.dom.replaceChildren();

    if (mode === 'loading') {
      this.dom.classList.add('is-loading');
      this.dom.appendChild(noteElement('qm-math-note', '正在渲染公式…'));
      return;
    }

    const tex = this.painted ?? readTex(this.node.attrs);
    if (mode === 'empty') {
      this.dom.classList.add('is-empty');
      this.dom.appendChild(noteElement('qm-math-note', '空公式 —— 点一下输入'));
      return;
    }

    this.dom.classList.add('is-failed');
    this.dom.appendChild(noteElement('qm-math-src', sourceOf(tex, this.kind)));
    this.dom.appendChild(noteElement('qm-math-note', message ?? '公式渲染失败'));
  }

  // -------------------------------------------------------------------------
  // 原位编辑
  // -------------------------------------------------------------------------

  private enterEdit(): void {
    if (this.editing) return;
    this.editing = true;

    const block = this.kind === 'block';
    const input: HTMLInputElement | HTMLTextAreaElement = block
      ? document.createElement('textarea')
      : document.createElement('input');
    if (input instanceof HTMLInputElement) input.type = 'text';
    input.className = 'qm-math-input';
    input.value = readTex(this.node.attrs);
    input.spellcheck = false;
    input.setAttribute('aria-label', block ? '块级公式的 LaTeX 源码' : '行内公式的 LaTeX 源码');
    if (input instanceof HTMLTextAreaElement) {
      input.rows = Math.min(8, Math.max(1, input.value.split('\n').length));
    }

    /**
     * 键盘。
     *
     * `stopPropagation` 是必需的：`stopEvent` 只让 ProseMirror 忽略这些事件，
     * 而外壳的全局快捷键挂在 `window` 上，照样收得到 —— 在里面按 Esc
     * 会顺手把查找栏关掉（与选中工具条的链接输入框是同一个坑）。
     *
     * 确认键按形态区分：行内公式是单行，`Enter` 就是确认；块级公式的换行
     * 是公式的一部分（`aligned` 环境必须换行），所以确认要按 `Ctrl/Cmd+Enter`。
     */
    /**
     * 绑事件的收窄别名。
     *
     * `input` 的静态类型是 `HTMLInputElement | HTMLTextAreaElement`，在联合上调用
     * `addEventListener` 时 TypeScript 合不出一个具体的事件类型，会退化成
     * `EventListener`（回调参数变成 `Event`，`event.key` 直接报错）。两者共同的基类
     * `HTMLElement` 的 `keydown` 就是 `KeyboardEvent`，所以绑事件走这个别名。
     */
    const target: HTMLElement = input;

    target.addEventListener('keydown', (event: KeyboardEvent) => {
      event.stopPropagation();
      if (event.key === 'Escape') {
        event.preventDefault();
        this.exitEdit();
        return;
      }
      const commitKey = this.kind === 'block' ? (event.ctrlKey || event.metaKey) && event.key === 'Enter' : event.key === 'Enter';
      if (commitKey) {
        event.preventDefault();
        this.commit();
      }
    });

    // 失焦就确认：用户点到别处去了，他心里的动作已经做完了。
    // 不确认的话输入框会留在那儿，而用户已经看不到自己在编辑什么
    target.addEventListener('blur', () => {
      if (!this.editing) return;
      this.commit();
    });

    this.input = input;
    this.dom.classList.remove('is-loading', 'is-loaded', 'is-failed', 'is-empty');
    this.dom.classList.add('is-editing');
    this.dom.replaceChildren(input);

    const hint = document.createElement('span');
    hint.className = 'qm-math-hint';
    hint.contentEditable = 'false';
    hint.textContent = block ? 'Ctrl+Enter 确认 · Esc 取消' : 'Enter 确认 · Esc 取消';
    this.dom.appendChild(hint);

    this.focusInput();
  }

  /** 把光标交给输入框，落在**末尾**。 */
  private focusInput(): void {
    const input = this.input;
    if (input === null) return;
    input.focus();
    // 光标放到末尾而不是全选：多数时候用户是去改公式里的一两个字符，
    // 全选会让「点一下」变成「一不小心全删了」
    input.setSelectionRange(input.value.length, input.value.length);
  }

  private exitEdit(): void {
    if (!this.editing) return;
    // 先置位再摘元素：摘掉一个聚焦中的元素在部分浏览器上会派发 blur，
    // 那个回调必须看到「已经不在编辑了」
    this.editing = false;
    const input = this.input;
    this.input = null;
    input?.remove();
    this.dom.classList.remove('is-editing');
    // DOM 已经被输入框占过一轮了，所以**必须**重画 —— 把 `painted` 清掉，
    // 让重画不会被「值没变就跳过」的优化挡住
    this.painted = null;
    this.paintCurrent();
  }

  /** 按**当前节点**的值重画。空值画占位，其余交给 `render`。 */
  private paintCurrent(): void {
    const tex = readTex(this.node.attrs);
    if (tex.trim() === '') {
      this.painted = tex;
      this.dom.dataset['tex'] = tex;
      this.paint('empty');
      return;
    }
    void this.render(tex);
  }

  /**
   * 把输入框里的内容写回节点。
   *
   * **先校验再写**（`core/math.ts` 的 `texProblem`）：有些值写进 Markdown 之后
   * 读不回来，保存下去就是**静默丢内容**。拦下来并说明原因，比「存进去再说」好得多。
   */
  private commit(): void {
    const input = this.input;
    if (input === null) return;
    const tex = input.value;

    const problem = texProblem(tex, this.kind);
    if (problem !== null) {
      // 不退出编辑：让用户就着手改。原因写在输入框旁边
      this.dom.querySelector('.qm-math-hint')?.replaceWith(noteElement('qm-math-hint is-error', problem));
      input.focus();
      return;
    }

    const current = readTex(this.node.attrs);
    if (tex === current) {
      this.exitEdit();
      return;
    }

    const pos = this.getPos();
    if (pos === undefined) {
      // 节点已经不在文档里了（被删掉、或整篇被替换）。什么都不做 ——
      // 往一个猜出来的位置上写比不写更糟
      this.exitEdit();
      return;
    }

    // 先派发再退出编辑：`update()` 会在这次 dispatch 里同步跑，
    // 由它把视图切回渲染态并渲染**新**的值。反过来做会先闪一下旧公式
    this.view.dispatch(
      this.view.state.tr
        .setNodeMarkup(pos, undefined, { ...this.node.attrs, [TEX_ATTR]: tex })
        .setMeta(USER_EDIT_META, true),
    );
  }

  // -------------------------------------------------------------------------
  // ProseMirror 回调
  // -------------------------------------------------------------------------

  update(node: ProseNode): boolean {
    // 只处理公式节点。返回 true 说「我还能用」会把一个别的类型的节点塞给这个视图
    if (node.type.name !== (this.kind === 'block' ? MATH_BLOCK : MATH_INLINE)) return false;

    const next = readTex(node.attrs);
    this.node = node;

    /**
     * 正在编辑时文档里的值变了（撤销、整篇替换、另一处编辑）——
     * 丢弃草稿，以文档为准。留着草稿会让用户以为自己的改动还在。
     *
     * 退出编辑本身就会用**当前节点**的值重画，所以这条路径不需要再渲染一次。
     */
    if (this.editing) {
      this.exitEdit();
      return true;
    }

    // `update` 在**每一次文档变更**时都会被调用（包括在别处打字）。
    // 值没变就什么都不做 —— 否则每敲一个字都要重新渲染一次公式
    if (next !== this.painted) void this.render(next);
    return true;
  }

  /**
   * 这个 NodeView 里的所有 DOM 变更都不算文档变更。
   *
   * 公式是叶子节点，没有 `contentDOM`，所以「不在 contentDOM 里就不算」
   * 简化成了「全都不算」。少了它，KaTeX 写进去的那一大棵 DOM
   * （以及输入框里用户打的字）会被 ProseMirror 当成用户输入**读进文档** ——
   * 保存出来的 Markdown 里会多出一坨 KaTeX 的 HTML。
   */
  ignoreMutation(): boolean {
    return true;
  }

  /**
   * 这个 NodeView 里的事件不要让 ProseMirror 处理。
   *
   * 判据只有一条：**事件发生在 `this.dom` 里面**。这块 DOM 整个是我们的
   * （叶子节点，没有 `contentDOM`），所以「在里面」就等于「是我们的」。
   *
   * 不挡的话，在输入框里按退格会被 ProseMirror 当成「删掉公式节点」处理 ——
   * 用户想改一个字符，公式没了。这与代码块的语言标签是同一条理由。
   */
  stopEvent(event: Event): boolean {
    const target = event.target;
    return target instanceof Node && this.dom.contains(target);
  }

  destroy(): void {
    this.alive = false;
    this.editing = false;
    this.input = null;
  }
}

/**
 * 注册公式的 NodeView。
 *
 * 两个节点类型共用一个类：它们的差别只有「displayMode」和「确认键」两处，
 * 分成两个类会把状态机（渲染、竞态、编辑）抄两遍。
 */
export function mathViewPlugin() {
  return $prose(
    () =>
      new Plugin({
        key: mathViewKey,
        props: {
          nodeViews: {
            [MATH_INLINE]: (node, view, getPos) => new MathView(node, view, getPos, 'inline'),
            [MATH_BLOCK]: (node, view, getPos) => new MathView(node, view, getPos, 'block'),
          },
        },
      }),
  );
}
