/**
 * 数学公式的纯逻辑：**识别**（给输入规则用）与**校验**（给编辑用）。
 *
 * 这个文件里没有 ProseMirror、没有 KaTeX、没有 DOM —— 只有「Markdown 里的公式
 * 长什么样」这一件事。放在 `core/` 是因为三处都要用它，而且三处必须**用同一份判断**：
 *
 * 1. 输入规则（`modules/markdown/math.ts`）：用户敲下收尾的 `$` 时，判断这一段
 *    到底是不是公式、以及它在文本里占几个字符；
 * 2. 编辑面板（`modules/markdown/math-view.ts`）：用户改完公式后，判断这个值
 *    能不能安全地存回文档；
 * 3. 测试：上面两处的行为都要能脱离编辑器被钉住。
 *
 * ## 为什么识别要**真的调一遍解析器**，而不是自己写正则
 *
 * 「`$...$` 是不是公式」这件事的判定规则比看起来复杂得多，而且**规则住在
 * `micromark-extension-math` 里**：
 *
 * - 开头的 `$` 前面不能再有 `$`（否则是 `$$` 围栏）；
 * - 内容不能为空；
 * - 收尾的 `$` 必须与开头的 `$` **个数相同**（`$$x$` 不是公式）；
 * - 内容首尾**同时**是空白时会被剥掉，只有一边是空白时不会被剥。
 *
 * 自己写一条正则去近似它，就等于在项目里维护**第二套语法规则**。两套规则
 * 一旦有一处不一致，后果是「打字时是这样、重新打开文件是那样」——
 * 而这种不一致用户根本无从理解，只会觉得「这个编辑器会吃我的字」。
 *
 * 所以这里直接复用 `remark-math`（也就是文档解析器自己用的那一套），
 * 再用 mdast 的 `position.offset` 反推源文本长度。**打字与重新打开必然一致**，
 * 因为它们跑的是同一段代码。
 *
 * ## 代价（必须写清楚）
 *
 * 这个模块在 `core/` 里持有**第二个** remark 处理器实例（内核那个在
 * `@milkdown/core` 的 `parserCtx` 里，拿不到 mdast，只有 ProseMirror 文档）。
 * 两边的差别只在于「有没有装上 normalize-ast 那个补丁」，而那个补丁只改
 * 「声明为字符串、实际是 null」的字段，不碰公式 —— 见 `math.test.ts` 里
 * 与真实解析器的对照用例。
 */

import { unified } from 'unified';
import { t } from './i18n/index.js';
import remarkParse from 'remark-parse';
import remarkMath from 'remark-math';

/**
 * 只在行内找公式的处理器。
 *
 * **刻意不装 `remark-stringify`**：这里只需要「读」，装上序列化器只会让
 * 这个实例多一份与本文件无关的行为。
 */
const inlineProcessor = unified().use(remarkParse).use(remarkMath);

/** 公式的两种形态。`block` 是独占一段的 `$$…$$`，`inline` 是行内的 `$…$`。 */
export type MathKind = 'inline' | 'block';

// ---------------------------------------------------------------------------
// DOM 契约：内核画出来的公式，导出侧要认得
// ---------------------------------------------------------------------------

/**
 * 公式节点的 DOM 契约。**这是内核与导出之间唯一的东西**，所以放在 core：
 *
 * - 内核的 `toDOM` 把 TeX 源码写在 `data-tex` 上，用 `data-type` 区分行内/块级；
 * - 导出侧（`core/export/inline-html.ts`）据此找到每一处公式，
 *   把壳里的内容换成 KaTeX 渲染的结果。
 *
 * 为什么不放在内核里让导出去 import：依赖方向只能是「模块 → core」。
 * 导出层不知道 Markdown 内核的存在，也不该知道 —— 将来思维导图、draw.io
 * 导出要嵌进同一份 HTML 时，它们照同一个契约写就行。
 */
export const TEX_DATA_ATTR = 'data-tex';

/** `data-type` 的两个取值。内核**故意**把 ProseMirror 节点名也取成同样的字符串，
 * 于是「节点名 ↔ DOM 契约值」的映射是恒等的，不需要一张翻译表。 */
export const MATH_INLINE = 'math_inline';
export const MATH_BLOCK = 'math_block';


/**
 * 「光标前刚刚闭合了一个行内公式」的识别结果。
 *
 * 两个字段缺一不可：
 * - `tex` 是要放进节点的**值**（已经过 micromark 的空白剥离规则）；
 * - `length` 是这段公式在**源文本**里占的字符数（含两边的 `$`），
 *   用来算输入规则该替换掉哪一段。
 *
 * 注意 `tex` 的长度与 `length` **不是**一回事：`$ x $` 的 `tex` 是 `x`（3 个字符
 * 被剥掉），而 `length` 是 5。少了 `length` 就只能靠正则去猜边界。
 */
export interface InlineMathSpan {
  readonly tex: string;
  readonly length: number;
}

/** mdast 节点的最小形状（只声明本文件用得到的字段，不去依赖 `@types/mdast`）。 */
interface MdastNode {
  readonly type: string;
  readonly value?: unknown;
  readonly children?: readonly MdastNode[];
  readonly position?: {
    readonly start?: { readonly offset?: number };
    readonly end?: { readonly offset?: number };
  };
}

/**
 * 判断 `text` 的**末尾**是不是刚刚闭合了一个行内公式；不是就返回 `null`。
 *
 * `text` 的语义是「光标所在文本块里、光标之前的全部文字」。调用方负责给对 ——
 * 传整段文档进来会得到没有意义的结果（`- $x$` 会被解析成列表）。
 *
 * ## 为什么要求「末尾」而不是「任意位置」
 *
 * 输入规则只在**刚打完一个字符**时触发。用户敲下收尾的 `$` 之后，公式必然贴着
 * 光标结束。允许「任意位置」的话，在 `$a$ 后面还有字` 的段落里敲任何键都会
 * 重新识别一遍，而那时公式早就闭合了 —— 结果是重复转换。
 *
 * 判定用的是 mdast 的**绝对偏移**：`position.end.offset === text.length`。
 * 这比「最后一个子节点是 inlineMath」更强 —— 后者在 `$a$ ` （末尾有空格）时
 * 也会成立，而那种情况公式并没有贴着光标。
 */
export function inlineMathAtEnd(text: string): InlineMathSpan | null {
  if (!text.endsWith('$')) return null;

  const tree = inlineProcessor.parse(text) as unknown as MdastNode;
  const last = lastChild(tree);
  if (last === null || last.type !== 'paragraph') return null;

  const node = lastChild(last);
  if (node === null || node.type !== 'inlineMath') return null;

  const start = node.position?.start?.offset;
  const end = node.position?.end?.offset;
  if (typeof start !== 'number' || typeof end !== 'number') return null;
  // 公式必须**贴着**末尾结束；末尾还有别的字符就说明它早就闭合了
  if (end !== text.length) return null;
  // `$$x$$` 那种写法会让 `position` 覆盖不到开头以外的位置，长度不可能是 0 或负
  if (end - start < 2) return null;

  return { tex: typeof node.value === 'string' ? node.value : '', length: end - start };
}

function lastChild(node: MdastNode): MdastNode | null {
  const children = node.children;
  if (children === undefined || children.length === 0) return null;
  return children[children.length - 1] ?? null;
}

/**
 * 这个公式值能不能安全地存回文档；能存返回 `null`，不能存返回**给用户看的原因**。
 *
 * ## 为什么必须校验，而不是「存进去再说」
 *
 * 公式的值最终要变成 Markdown 源文本里的一段 `$…$` / `$$…$$`。有些值写回去之后
 * **读不回来** —— 而那时文档已经被保存了，用户的公式只剩下一半。实测（见
 * `math.test.ts` 的往返用例）只有两种情况会坏：
 *
 * 1. **空值**：行内空公式序列化成 `$$`，重新解析出来是**块级**公式 ——
 *    一个行内位置凭空冒出一个独占一段的公式，把整段文字劈成两半；
 * 2. **行内公式里含空行**：`$a` + 空行 + `b$` 重新解析后**整段消失**，
 *    公式不见了、文字也没了。
 *
 * 块级公式**没有**这两个问题：`mdast-util-math` 遇到值里有 `$$` 时会自动
 * 换用更长的围栏（`$$$`），值里有空行也照样读得回来。所以这里对块级只挡空值。
 *
 * 反过来说：**除了这两条，其余一律放行**。值里有 `$`、有反斜杠、有 `#`、有
 * `---` 都试过，往返都正确（序列化器会自己加空格或换围栏）。多加一条无谓的
 * 限制，代价是用户写不出本来合法的公式 —— 而「拦住一个合法输入」和
 * 「放进去一个坏输入」一样糟。
 */
export function texProblem(tex: string, kind: MathKind): string | null {
  if (tex.trim() === '') return t('core.math.empty');

  if (kind === 'inline' && /\n[ \t]*\n/.test(tex)) {
    return t('core.math.inlineBlank');
  }

  return null;
}
