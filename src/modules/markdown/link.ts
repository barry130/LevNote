/**
 * 链接：点击事件的上报与锚点定位（Markdown 内核）。
 *
 * ## 为什么内核要管「点击」这件事
 *
 * WebView 里点一个 `<a href="https://…">`，默认行为是**整个应用导航走** ——
 * 界面变成一个网页，回不来。这是当前存在的真实缺陷（之前没有任何一处拦截）：
 * 内核画的链接由内核负责拦截，外壳既看不到那个事件，也不该去 `querySelector`
 * 内核的 DOM（硬约束 11）。
 *
 * ## 为什么 `preventDefault` 与「上报」是两件事
 *
 * - **`preventDefault` 是无条件的**：任何 `<a>` 的默认行为都得掐掉，无论这一次
 *   上不上报、无论用户有没有按修饰键。它是安全网，不是打开动作的副作用。
 * - **上报只在「按住 Ctrl / Cmd」时发生**：编辑器里的普通左键点击是**移动光标**，
 *   想点进链接中间改一个字是最正常的操作，不该被劫持成「打开网页」。
 *   与 VSCode / Typora 的手势一致。
 *
 * ## 为什么只能用 `click`，不能用 `handleClickOn`
 *
 * `handleClickOn` 看起来就是为这件事准备的（它直接给出被点击的节点），**但它
 * 是在 `mouseup` 里被调用的**（prosemirror-view 的 `LeftMouseDown.up`）：
 * 在 mouseup 上 `preventDefault()` 掐不掉 `click` 的默认行为，链接照样会导航走。
 * 所以它用不了，只有挂在真实 `click` 上的 handler 才掐得住。
 *
 * ## 为什么读文档里的原始 `href`，不读 DOM 里的 `href` 属性
 *
 * DOM 上的 `href` 是 **Milkdown 消毒之后**的值（`sanitizeLinkHref`）：不安全的
 * 协议会被换成空串。用它有三个问题：
 *
 * 1. 那份消毒规则是 preset-commonmark 的私有实现，会随主版本漂移；
 * 2. 用户文档里明明写着 `javascript:…`，我们却只能看到一个空串 —— 就没法如实
 *    告诉他「这个协议打不开」（硬约束 2），点击变成一次静默 no-op；
 * 3. 相对地址虽然会原样留下，但我们依赖的是「哪些能留下」这个偶然事实。
 *
 * 所以本协议走文档模型取 `attrs.href`（= 用户写的原文），分类（`core/link.ts`
 * 的 `classifyLink`）与「相对谁」都由外壳负责 —— 内核连自己正在编辑哪个文件都
 * 不该知道。
 */

import type { Mark, Node as ProseNode } from '@milkdown/kit/prose/model';
import type { EditorView } from '@milkdown/kit/prose/view';
import { Plugin, PluginKey } from '@milkdown/kit/prose/state';
import { $prose } from '@milkdown/kit/utils';

import type { LinkActivation } from '../../core/editor/types.js';

/** 链接 mark 的名字（对齐 preset-commonmark 的 schema）。 */
const LINK = 'link';
/** `<a>` 的标签名。只用它划定「点在链接上」，不读它的 `href`（理由见文件头）。 */
const ANCHOR_TAG = 'A';

const linkKey = new PluginKey('levnote-link-activation');

/** 「打开链接」的手势：Windows / Linux 用 Ctrl，macOS 用 Cmd。 */
function isActivation(event: MouseEvent): boolean {
  return event.ctrlKey || event.metaKey;
}

/**
 * 从事件目标往上找到**这个编辑器里的**那个 `<a>`。
 *
 * 走到 `view.dom` 就停：整块编辑器被一个外层的 `<a>` 包住时（本项目的外壳里
 * 不会发生，但 NodeView 里难说），不该回答「点在链接上」。
 */
function anchorElementOf(view: EditorView, target: EventTarget | null): Element | null {
  if (!(target instanceof Element)) return null;

  let current: Element | null = target;
  while (current !== null && current !== view.dom) {
    if (current.tagName === ANCHOR_TAG) return current;
    current = current.parentElement;
  }
  return null;
}

/**
 * 某个 DOM 元素位置上的**链接 mark**。取不到返回 null。
 *
 * 向上试错的写法照抄 `block.ts` 的 `blockIndexFromElement`：NodeView 内部的装饰性
 * 元素不在 docView 树里，`posAtDOM` 会直接抛，没有异常就没法处理。
 *
 * 这里**只认某一个祖先**：位置解出来之后若那里没有链接，就到此为止 —— 继续往上
 * 只会在「点击明明不在链接上」时误判成「在链接上」（链接 mark 挂在 text 节点上，
 * 第一个能解开位置的祖先已经是最精确的那个答案了）。
 */
function linkMarkAt(view: EditorView, element: Element): Mark | null {
  let current: Element | null = element;
  while (current !== null && current !== view.dom) {
    try {
      return view.state.doc.nodeAt(view.posAtDOM(current, 0))?.marks.find((m) => m.type.name === LINK) ?? null;
    } catch {
      // 这个元素不在 docView 树里（NodeView 内部的装饰性元素）
      current = current.parentElement;
      continue;
    }
  }
  return null;
}

/**
 * 某个文档节点上的 `id` 属性值（锚点判定用），没有或不是字符串时 null。
 */
function idOf(node: { attrs: Readonly<Record<string, unknown>> }): string | null {
  const id = node.attrs['id'];
  return typeof id === 'string' ? id : null;
}

/**
 * 找到文档里 `id` 等于 `anchor` 的那个节点的位置。
 *
 * **刻意不限定为标题**：判别依据是「有没有这个 `id`」本身，与浏览器 hash 跳转
 * 的口径一致。今天只有标题带 `id`（preset-commonmark 的 `syncHeadingIdPlugin`
 * 生成，重名时加 `-#2` / `-#3` 后缀），但把规则写成「只看 heading」等于替明天
 * 的 schema 做决定 —— 哪天代码块也带上了 id，那份判定就得跟着改一遍。
 *
 * **空串不算锚点**：没有 `id` 的节点（比如一条还没被 `syncHeadingIdPlugin` 写到
 * 的空标题）其 `id` 正是空串，拿空串去问会命中它 —— 那是「没写锚点」而不是
 * 「锚点指向这里」。空串由外壳判成 `empty` 并说一句话，不该走到这里。
 */
export function findAnchorPos(doc: ProseNode, anchor: string): number | null {
  if (anchor === '') return null;

  let found: number | null = null;

  doc.descendants((node, pos) => {
    if (found !== null) return false;
    if (idOf(node) === anchor) {
      found = pos;
      return false;
    }
    return true;
  });

  return found;
}

/**
 * 链接的**点击**上报。
 *
 * `onLinkClick` 没挂时（比如详情页面）行为与「掐掉默认导航」一致：链接可编辑，
 * 但点了不会打开任何东西 —— 不做任何假装支持的补偿。
 */
export function linkClickPlugin(getReporter: () => ((link: LinkActivation) => void) | undefined) {
  return $prose(
    () =>
      new Plugin({
        key: linkKey,
        props: {
          handleDOMEvents: {
            click: (view, event) => {
              if (!(event instanceof MouseEvent)) return false;
              const anchor = anchorElementOf(view, event.target);
              if (anchor === null) return false;

              // 编辑器里没有任何链接该让 WebView 导航走 —— 无条件掐掉
              event.preventDefault();

              // 没按修饰键 = 用户要的是把光标放到这里，不是打开链接
              if (!isActivation(event)) return false;

              const mark = linkMarkAt(view, anchor);
              if (mark === null) return false;

              const href = mark.attrs['href'];
              if (typeof href !== 'string') return false;

              getReporter()?.({ href, text: anchor.textContent ?? '' });
              return true;
            },
          },
        },
      }),
  );
}
