/**
 * 数学公式的**节点定义与输入规则**。
 *
 * ## 为什么自己写节点，不用 `@milkdown/plugin-math`
 *
 * 三条理由，每一条单独看都够用：
 *
 * 1. **它已经被官方废弃了。** npm 上的 `@milkdown/plugin-math` 带着
 *    「This package has been deprecated. Package no longer supported.」的标记；
 *    最后一版 7.5.9 停在 2024-12-16，而 `@milkdown/kit` 已经是 7.22.2。
 *    引一个不再维护的包，等于把「上游 API 漂移」这件事永远留给自己。
 * 2. **它的 API 已经和现在的 `@milkdown/utils` 对不上了。** 它用的是
 *    `$nodeSchema` / `$remark` 的老调用方式，依赖里还钉着 `katex@^0.16`；
 *    真装进来会同时存在两份 `@milkdown/utils`（7.5.9 与 7.22.2）。
 * 3. **它在 `toDOM` 里同步调 `katex.render`。** `toDOM` 是同步的，所以 KaTeX
 *    只能**静态**引入 —— 272 KB 会进主包，每个打开轻墨的人都要先下载它。
 *    而且 `toDOM` 里抛异常会直接把编辑器带崩（非法公式必然抛）。
 *
 * 自己写这一百多行，换来的是：KaTeX 动态加载、渲染失败可以如实显示、
 * 公式可以在原位编辑。这与 M5 拒用 Vue 版官方组件（ADR 0017）、
 * M6 自己画图片 NodeView（ADR 0018）是同一条路线。
 *
 * ## 值存在 `value` 属性里，两种公式**一致**
 *
 * 上游把行内公式的 TeX 存在**文本内容**里（`content: 'text*'` + `atom: true`），
 * 块级公式却存在属性里 —— 同一种东西两种存法，序列化与解析各写一套。
 * 这里统一存 `value` 属性：
 *
 * - 节点是**真叶子**（没有 content），`atom: true` 的含义变得直白；
 * - 解析、序列化、NodeView、原位编辑四处都只读同一个字段；
 * - 不用担心「用户在原子节点里打字」这种状态存在。
 *
 * ## 输入规则的判定**复用文档解析器**
 *
 * 见 `core/math.ts` 的 `inlineMathAtEnd` —— 打字时的识别与重新打开文件时的
 * 解析必须用同一套规则，否则会出现「打字时是公式、保存后变成文字」这种
 * 用户完全无法理解的差异。
 */

import type { Node as ProseNode } from '@milkdown/kit/prose/model';
import { InputRule } from '@milkdown/kit/prose/inputrules';
import { $inputRule, $nodeSchema, $remark } from '@milkdown/kit/utils';
import remarkMath from 'remark-math';

import { inlineMathAtEnd, MATH_BLOCK, MATH_INLINE, TEX_DATA_ATTR, texProblem } from '../../core/math.js';

/**
 * 两个节点名。它们**静默地**出现在 schema、NodeView 注册、`parseDOM` 的
 * `data-type` 与导出侧的 `[data-tex]` 扫描里 —— 上游改名不会有任何报错，
 * 只会让公式变成一坨源码。改动时四处一起改。
 *
 * 取值来自 `core/math.ts`：那里定义的是**内核与导出之间的 DOM 契约**，
 * 节点名故意取成同样的字符串，于是「节点名 ↔ 契约值」是恒等映射，
 * 不需要一张可能写错的翻译表（`math.test.ts` 里有用例钉着这一点）。
 */
export { MATH_BLOCK, MATH_INLINE };

/** TeX 源码所在的属性名。**这是节点自己的属性**，不属于 DOM 契约。 */
export const TEX_ATTR = 'value';

/**
 * 装上 `remark-math`，让 `$…$` / `$$…$$` 在解析阶段就变成 mdast 的
 * `inlineMath` / `math` 节点。
 *
 * **不传 options**：`singleDollarTextMath` 保持默认的 `true`。关掉它意味着
 * 只认 `$$…$$`，行内公式就没了 —— 那是更大的损失。代价是 `$5 到 $10`
 * 这种「两个美元符号夹一段文字」会被当成公式（micromark 只在内容**首尾同时**
 * 是空白时才剥离，不会因为「收尾的 `$` 前面是空格」而拒绝）。
 * 这是上游规则的既有行为，不在这里私自加判断 —— 加了就与「重新打开文件」不一致。
 * 用户可以用 `\$` 转义。详见 ADR 0019 的「已知代价」。
 */
export const mathRemarkPlugin = $remark('remarkMath', () => remarkMath);

/** `attrs` 是 `any`，取值一律收窄，不把未知当字符串用。 */
function readTex(attrs: ProseNode['attrs']): string {
  const raw: unknown = attrs[TEX_ATTR];
  return typeof raw === 'string' ? raw : '';
}

/** 从 DOM 元素的 `data-tex` 上取回 TeX。 */
function texFromDom(dom: HTMLElement): string {
  return dom.getAttribute(TEX_DATA_ATTR) ?? '';
}

/**
 * 生成用于**序列化**（导出、剪贴板）的 DOM。
 *
 * 它**不渲染公式**，只写一个带 `data-tex` 的壳，里面放公式的源码：
 *
 * - 导出时由 `core/export/inline-html.ts` 把壳的内容换成 KaTeX 的结果；
 * - 万一那一步没做到（例如某个平台上 KaTeX 加载失败），用户看到的至少是
 *   `$x^2$` 这样一段**能读懂的源码**，而不是一个空白；
 * - 复制公式再粘回轻墨时，`parseDOM` 从 `data-tex` 上把它原样取回来。
 *
 * `toDOM` 必须保持同步且廉价：它在**每一次序列化**时都会跑，包括复制一小段
 * 文字。这里去做 KaTeX 渲染的话，KaTeX 就只能静态引入（见文件头）。
 */
function mathToDom(node: ProseNode, kind: 'inline' | 'block'): HTMLElement {
  const tex = readTex(node.attrs);
  const block = kind === 'block';

  const dom = document.createElement(block ? 'div' : 'span');
  dom.className = block ? 'qm-math qm-math-block' : 'qm-math qm-math-inline';
  dom.dataset['type'] = block ? MATH_BLOCK : MATH_INLINE;
  dom.setAttribute(TEX_DATA_ATTR, tex);
  dom.textContent = block ? `$$\n${tex}\n$$` : `$${tex}$`;
  return dom;
}

/**
 * 行内公式：`$E=mc^2$`。
 *
 * `selectable: false` 是刻意的：可选中意味着点一下会变成 `NodeSelection`，
 * 而我们要的是「点一下进入编辑」。`draggable` 同理保持关闭 —— 拖动一个公式
 * 的收益远小于「不小心把它拖丢了」的风险。
 */
export const mathInlineSchema = $nodeSchema(MATH_INLINE, () => ({
  inline: true,
  group: 'inline',
  atom: true,
  marks: '',
  selectable: false,
  attrs: { [TEX_ATTR]: { default: '' } },
  parseDOM: [
    {
      tag: `span[data-type="${MATH_INLINE}"]`,
      getAttrs: (dom) => (dom instanceof HTMLElement ? { [TEX_ATTR]: texFromDom(dom) } : false),
    },
  ],
  toDOM: (node) => mathToDom(node, 'inline'),
  parseMarkdown: {
    match: (node) => node.type === 'inlineMath',
    runner: (state, node, type) => {
      state.addNode(type, { [TEX_ATTR]: typeof node.value === 'string' ? node.value : '' });
    },
  },
  toMarkdown: {
    match: (node) => node.type.name === MATH_INLINE,
    runner: (state, node) => {
      state.addNode('inlineMath', undefined, readTex(node.attrs));
    },
  },
}));

/**
 * 块级公式：独占一段的 `$$…$$`。
 *
 * `isolating: true` 让它在编辑上把自己与相邻块隔开 —— 不然在公式旁边按退格
 * 可能把两段文字接起来，而用户以为自己只删了一个字符。
 */
export const mathBlockSchema = $nodeSchema(MATH_BLOCK, () => ({
  group: 'block',
  atom: true,
  marks: '',
  selectable: false,
  isolating: true,
  attrs: { [TEX_ATTR]: { default: '' } },
  parseDOM: [
    {
      tag: `div[data-type="${MATH_BLOCK}"]`,
      getAttrs: (dom) => (dom instanceof HTMLElement ? { [TEX_ATTR]: texFromDom(dom) } : false),
    },
  ],
  toDOM: (node) => mathToDom(node, 'block'),
  parseMarkdown: {
    match: (node) => node.type === 'math',
    runner: (state, node, type) => {
      state.addNode(type, { [TEX_ATTR]: typeof node.value === 'string' ? node.value : '' });
    },
  },
  toMarkdown: {
    match: (node) => node.type.name === MATH_BLOCK,
    runner: (state, node) => {
      state.addNode('math', undefined, readTex(node.attrs));
    },
  },
}));

/**
 * 敲下收尾的 `$` 时把 `$…$` 变成行内公式节点。
 *
 * ## 坐标是怎么算的
 *
 * 输入规则触发时，**那个 `$` 还没有进入文档** —— 处理函数返回事务就表示
 * 「这次输入由我负责」，ProseMirror 会 `preventDefault` 掉浏览器的默认插入。
 * 所以 `end`（= 光标位置）正好是「公式在源文本里结束」的位置，而公式占
 * `span.length` 个字符，起点就是 `end + 1 - span.length`。
 *
 * 反过来说：**绝不能**用 `end` 当公式末尾去删 —— 那样每转换一次就会在文档里
 * 多留一个 `$`，用户看到的是 `$x^2` 后面跟着一个孤零零的美元符号。
 *
 * ## 为什么正则只有 `/\$$/`
 *
 * 真正「这一段是不是公式」的判断交给 `inlineMathAtEnd`（也就是文档解析器）。
 * 正则放宽到只认「光标前刚打了一个 `$`」，判定只有一处、且与重新打开文件
 * 完全一致。正则写严的版本（比如 `/\$[^$]+\$$/`）会在 `$$x$$` 上匹配到
 * 错的位置，还会与解析器的规则悄悄分叉。
 *
 * ## 为什么还要再过一道 `texProblem`
 *
 * **输入规则绝不能产出「存不住」的值。** 解析器认可的写法不一定能存回来：
 * `$ $` 解析出来的值是**一个空格**，而空格值的行内公式序列化之后会变成
 * 块级公式（见 `core/math.ts` 的 `texProblem`）—— 用户敲一个 `$ $`，
 * 整段文字就被一个独占一段的公式劈成两半。
 *
 * 所以「解析器认了」之后还要问一次 `texProblem`。两道判断都指向同一份
 * 规则（都在 `core/math.ts`），不会分叉。代价是 `$ $` 保持字面量 ——
 * 那正是用户想要的（他打的是两个美元符号，不是一个空公式）。
 */
export const mathInlineInputRule = $inputRule(
  () =>
    new InputRule(/\$$/, (state, _match, _start, end) => {
      const $end = state.doc.resolve(end);
      const parent = $end.parent;
      if (!parent.isTextblock) return null;

      // 取**整块**文本而不是输入规则给的 textBefore：后者在长段落里会被
      // 截断到 500 个字符，那样算出来的位置会整体偏移
      const before = parent.textBetween(0, $end.parentOffset);
      const span = inlineMathAtEnd(`${before}$`);
      if (span === null) return null;
      // 解析器认可、但存不回来的值（空值 / 只含空白）不转换，保持字面量
      if (texProblem(span.tex, 'inline') !== null) return null;

      const from = end + 1 - span.length;
      if (from < $end.start()) return null;

      const type = state.schema.nodes[MATH_INLINE];
      if (type === undefined) return null;

      return state.tr.replaceWith(from, end, type.create({ [TEX_ATTR]: span.tex }));
    }),
);

/**
 * 在空段落开头敲 `$$ ` 时把它变成一个空的块级公式。
 *
 * ## 为什么必须额外判断「真的在块首」
 *
 * 正则的 `^` 匹配的是 `textBefore` 的开头，而 `textBefore` 会被截断到光标前
 * 500 个字符。于是一段 600 字的段落里，第 501 个字符处的 `$$ ` 也会命中 ——
 * 用户会看到自己的正文被凭空换成一个公式。所以再用 `start` 与块的起点
 * 比对一次，把「恰好出现在开头」收紧成「真的是开头」。
 *
 * ## 为什么是这个触发方式
 *
 * `$$` 后面必须是**空白**才触发。只打 `$$` 不触发：那样任何以 `$$` 开头、
 * 后面接着写字的行都会被吃掉，而 `$$` 在正文里完全可能是「两个美元符号」。
 */
export const mathBlockInputRule = $inputRule(
  (ctx) =>
    new InputRule(/^\$\$\s$/, (state, _match, start, end) => {
      const $end = state.doc.resolve(end);
      const parent = $end.parent;
      if (!parent.isTextblock) return null;
      // 命中区间必须真的从块首开始（见上面的注释）
      if (start !== $end.start()) return null;

      const type = mathBlockSchema.type(ctx);

      /**
       * **不能用 `setBlockType`。**
       *
       * 它只接受 textblock（它的作用是把一个文本块换成另一种文本块），而
       * `math_block` 是块级原子节点 —— 传进去会抛
       * `RangeError: Type given to setBlockType should be a textblock`。
       * 那是一条**异常**，不是返回值：它会顺着 `handleTextInput` 冒到
       * ProseMirror 的事件处理里，整次输入无声无息地失败，用户只看到
       * `$$ ` 变成了字面量。
       *
       * 改成把**整个段落**换掉：`before()` / `after()` 给出段落前后两个
       * 绝对位置（`depth` 默认就是当前深度，也就是那个段落），于是
       * 「段落 → 公式块」是一次普通的 `replaceWith`。
       */
      const from = $end.before();
      const to = $end.after();
      return state.tr.replaceWith(from, to, type.create());
    }),
);
