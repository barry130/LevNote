/**
 * 块手柄的全部逻辑：把「文档 + 一个顶层下标」算成外壳要的那几个布尔值，
 * 把「指针在哪」算成「指着哪一块」与「松手会插到哪儿」。
 *
 * ## 为什么抽出来
 *
 * 与 `table.ts` 的 `describeTable` 同一条理由：手柄真正难的部分不是画界面，
 * 而是「这一块能不能上移」「松手会插到哪儿」这些判断。
 * 其中**纯算术**那部分（`describeBlock` / `dropIndexFor` / `isNoopMove`）
 * 在 jsdom 里就能跑，于是「手柄的行为对不对」这件事测得了一半；
 * 剩下的一半是几何量取（`posAtCoords` / `nodeDOM` / `coordsAtPos`），
 * 那才真的只有真机能验。
 *
 * ## 依赖方向
 *
 * `plugin.ts → block.ts → selection.ts`，单向。几何量取放在这里而不是
 * `selection.ts`，正是为了不让 `selection.ts` 反过来 import `childStart`
 * 而形成环。
 *
 * ## 「指针指着哪一块」有两条路，用途不同
 *
 * | 用途 | 走哪条 | 为什么 |
 * |---|---|---|
 * | 悬停出手柄 | `blockIndexFromElement`（DOM 元素反推） | 要的是「指针真的压在这一块上」；而且它不依赖 `elementFromPoint`，jsdom 里也跑得通 |
 * | 拖动算落点 | `blockIndexFromPoint`（`posAtCoords`） | 拖动时指针下面压着的是**手柄自己**，元素命中会命中手柄；坐标命中问的是「这个位置在文档的哪里」，与压着谁无关 |
 */

import type { Node as ProseNode } from '@milkdown/kit/prose/model';
import type { EditorView } from '@milkdown/kit/prose/view';

import type { BlockHandleState, SelectionRect } from '../../core/editor/types.js';
import { blockFormatOf, rectOfRange } from './selection.js';

/**
 * 能互相转换的顶层块类型。
 *
 * 原子块（表格 / 公式 / 分隔线）**不在其中**：它们不是「文本块的另一种形式」，
 * 转过去转不回来。把它们也列上，用户点「转为 → 正文」就会把一张表格
 * 变成一段空文字 —— 而那是**静默丢内容**，硬约束 2 的红线。
 */
const CONVERTIBLE: ReadonlySet<string> = new Set([
  'paragraph',
  'heading',
  'blockquote',
  'bullet_list',
  'ordered_list',
  'code_block',
]);

/** 第 index 个顶层子节点**之前**的位置（也就是它的节点位置）。 */
export function childStart(doc: ProseNode, index: number): number {
  let start = 0;
  for (let i = 0; i < index; i += 1) start += doc.child(i).nodeSize;
  return start;
}

/**
 * 描述第 index 个顶层块 —— `BlockHandleState` 去掉 `anchor`（那部分要量几何）。
 *
 * 下标越界时返回 `null` 而不是抛错：这个下标是内核从**指针位置**算出来的，
 * 而指针位置与文档之间隔着一次布局，理论上存在「算完之后文档又变了」的窗口。
 * 抛错会让一次鼠标移动把编辑器带崩，返回 null 只是手柄不显示。
 */
export function describeBlock(doc: ProseNode, index: number): Omit<BlockHandleState, 'anchor'> | null {
  if (!Number.isInteger(index) || index < 0 || index >= doc.childCount) return null;

  const node = doc.child(index);
  const level = node.attrs['level'];
  const name = node.type.name;

  return {
    // 原子块如实给 null，不假装是段落 —— 假装的话菜单会点亮「正文」，
    // 而用户点下去什么都不会发生（见 `BlockHandleState.block`）
    block: blockFormatOf(name),
    headingLevel: name === 'heading' && typeof level === 'number' ? level : null,
    canMoveUp: index > 0,
    canMoveDown: index < doc.childCount - 1,
    // 只剩一块、而且它是空的 → 删除等于什么都不做
    canDelete: doc.childCount > 1 || node.content.size > 0,
    canTurnInto: CONVERTIBLE.has(name),
  };
}

/**
 * 落点下标（0 … childCount）。
 *
 * 判据是「指针落在这一块的**下半**吗」—— 与所有拖拽排序的惯例一致：
 * 指针在块的上半 → 插到它前面，下半 → 插到它后面。
 * 只用「指针在第几块里」是不够的：那样永远只能插到块的边界上，
 * 而用户想插到两块之间时指针必然落在其中一块的上半或下半，
 * 没法表达「这两块之间」。
 */
export function dropIndexFor(index: number, lowerHalf: boolean, count: number): number {
  const at = lowerHalf ? index + 1 : index;
  return Math.min(Math.max(at, 0), count);
}

/**
 * 把第 `index` 块移到 `target` 位置是不是**原地不动**。
 *
 * `target === index`（放在自己前面）与 `target === index + 1`（放在自己后面）
 * 都是原地不动 —— 少了后者，用户把一块往上拖一点点就会看到「删掉再插回来」，
 * 视觉上闪一下，而且会往撤销栈里塞一条无意义的记录。
 */
export function isNoopMove(index: number, target: number): boolean {
  return target === index || target === index + 1;
}

// ---------------------------------------------------------------------------
// 几何量取
// ---------------------------------------------------------------------------

/**
 * 第 index 个顶层块在视口里的包围盒。
 *
 * ## 为什么先试 `nodeDOM` 再退回 `coordsAtPos`
 *
 * `nodeDOM` 拿到的是**块自己那个 DOM 元素**，`getBoundingClientRect()` 给的是
 * 它的真实外框 —— 包含块自己的 padding / 边框 / 列表的缩进，那正是「这一块」
 * 在屏幕上占的地方。`coordsAtPos` 只能给出行盒（文字那一行的左右边界），
 * 用来贴手柄会偏到文字边缘上，而手柄该贴在**块**的边缘。
 *
 * 退回 `coordsAtPos` 是给「块还没被渲染成元素」的情形兜底。
 * 两条路都量不出来时返回 null —— 手柄不显示，好过把它定位到一个瞎猜的坐标上。
 */
export function blockAnchorOf(view: EditorView, index: number): SelectionRect | null {
  const { doc } = view.state;
  if (index < 0 || index >= doc.childCount) return null;

  const start = childStart(doc, index);
  const dom = view.nodeDOM(start);
  if (dom instanceof HTMLElement) {
    const rect = dom.getBoundingClientRect();
    // jsdom 里所有矩形都是 0 —— 那种「量到了但全是 0」的读数不能当有效结果用，
    // 否则单测里手柄会定位到 (0,0)，而真机上是好的（同一个函数两套行为）
    if (rect.height > 0 || rect.width > 0) {
      return {
        top: rect.top,
        left: rect.left,
        bottom: rect.bottom,
        right: rect.right,
        width: rect.width,
        height: rect.height,
      };
    }
  }

  const node = doc.child(index);
  return rectOfRange(view, start, start + node.nodeSize);
}

/** 指针落在哪一块上：下标 + 那一块的完整描述（含包围盒）。 */
export function readBlockAt(view: EditorView, index: number): BlockHandleState | null {
  const described = describeBlock(view.state.doc, index);
  if (described === null) return null;

  const anchor = blockAnchorOf(view, index);
  if (anchor === null) return null;

  return { ...described, anchor };
}

/**
 * 「指针正下方是第几块」——**从 DOM 元素反推**，不走坐标。
 *
 * ## 为什么不用 `posAtCoords`
 *
 * 两条独立的原因，任一条都足够：
 *
 * 1. **它依赖 `document.elementFromPoint` / `caretRangeFromPoint`，而 jsdom
 *    两个都没有**（实测：都是 `undefined`）。于是 `posAtCoords` 必抛，
 *    整个悬停链路在测试里永远走不通 —— 而这条链路上挂着「上移 / 删除 /
 *    转为」全部命令，等于手柄的核心行为一条都测不了。
 * 2. **它算的是「最近的一个文本位置」，不是「指针压着哪个元素」。** 指针停在
 *    块的**内边距**上时它会给出一个看似合理的答案，而我们要的恰恰是
 *    「指针真的压在这一块上吗」。
 *
 * ## 为什么可以用 `posAtDOM`
 *
 * 顶层块**就是 `view.dom` 的直接子元素**（实测确认：`p.parentElement === view.dom`），
 * 而 `posAtDOM` 只查 docView 树、不需要布局。往上走到第一个 `posAtDOM` 认得的
 * 元素为止，再 `resolve(...).index(0)` 就得到顶层下标 —— 对 `<li>` 这种中间层
 * 也对（`index(0)` 取的是**最外层**那一块，整张列表算一块）。
 *
 * 走到 `view.dom` 就停：`posAtDOM(view.dom, 0)` 是 0，会答出「第 0 块」——
 * 指针在编辑器的内边距上时不该报任何块。
 */
export function blockIndexFromElement(view: EditorView, el: EventTarget | null): number | null {
  if (!(el instanceof Element)) return null;

  let current: Element | null = el;
  while (current !== null && current !== view.dom) {
    try {
      return view.state.doc.resolve(view.posAtDOM(current, 0)).index(0);
    } catch {
      // 这个元素不在 docView 树里（NodeView 内部的装饰性元素）
      current = current.parentElement;
    }
  }
  return null;
}

/**
 * 指针是不是停在**左侧那条留白**里 —— 手柄就摆在那儿。
 *
 * ## 为什么必须有这一条
 *
 * 手柄贴在块左边缘往左 26px 处，而块左边缘**左边**是编辑器自己的内边距。
 * 用户要把鼠标从正文移到手柄上，中途必然穿过那条留白。纯按元素命中算的话，
 * 那几像素里「指针压着哪个块」的答案是**没有块** → 内核报 null →
 * 外壳把整个手柄卸载掉 → 用户永远点不到它（`mouseenter` 都不会再发）。
 *
 * 所以留一条几何兜底：指针在**当前这一块**的包围盒往左扩 `GUTTER` 的带子里，
 * 就继续算作「还指着这一块」。
 *
 * 只对**当前那一块**判，不遍历全文档：一次鼠标移动扫一遍所有块的矩形是
 * 不必要的开销，而指针要走到留白里，起点必然是紧邻的那一块。
 */
export function withinGutter(anchor: SelectionRect, x: number, y: number): boolean {
  return x >= anchor.left - GUTTER && x <= anchor.right && y >= anchor.top && y <= anchor.bottom;
}

/** 左侧留白的宽度。比 `BlockHandle.tsx` 的 `GRIP_OFFSET`（26）宽一截，留点余量。 */
const GUTTER = 44;

/** 松手会插到第几个块之前（0 … childCount）；量不出来时 null。 */
export function resolveDropIndex(view: EditorView, x: number, y: number): number | null {
  const index = blockIndexFromPoint(view, x, y);
  if (index === null) return null;

  const anchor = blockAnchorOf(view, index);
  if (anchor === null) return null;

  return dropIndexFor(index, y > (anchor.top + anchor.bottom) / 2, view.state.doc.childCount);
}

/**
 * 视口坐标落在第几个顶层块里 —— **拖动落点用这一条**。
 *
 * 悬停那条路刻意不走坐标（理由见 `blockIndexFromElement`），但拖动不行：
 * 拖动时指针下面压着的是**手柄自己**（它跟着块的位置，而指针从它身上出发），
 * 元素命中会命中手柄而不是正文。坐标命中问的是「这个位置在文档的哪里」，
 * 与指针下面压着谁无关，正是拖动需要的语义。
 *
 * 手柄在拖动期间会被加上 `pointer-events: none`（见 `BlockHandle.tsx`），
 * 那是第二道保险：`posAtCoords` 内部也走 `elementFromPoint`。
 */
function blockIndexFromPoint(view: EditorView, x: number, y: number): number | null {
  try {
    const hit = view.posAtCoords({ left: x, top: y });
    if (hit === null) return null;
    // `index(0)` = 「这个位置落在 doc 的第几个子节点里」
    return view.state.doc.resolve(hit.pos).index(0);
  } catch {
    // jsdom 没有 `elementFromPoint` / `caretRangeFromPoint`，`posAtCoords` 会抛
    return null;
  }
}

/**
 * 落点指示线在视口里的位置（一条 `height` 为 0 的横线）。
 *
 * 横线的 x 范围取**编辑器自己那个 DOM 的宽度**（`view.dom`）——
 * 那是内核自己的元素，不是外壳按类名去认的东西。
 * 只给 `y` 不给 x 会让外壳自己去猜「线该多长」，而它猜不出来。
 */
export function dropLineAt(view: EditorView, target: number): SelectionRect | null {
  const { doc } = view.state;
  if (!Number.isInteger(target) || target < 0 || target > doc.childCount) return null;

  try {
    const coords = view.coordsAtPos(Math.min(childStart(doc, target), doc.content.size));
    const box = view.dom.getBoundingClientRect();
    return {
      top: coords.top,
      bottom: coords.top,
      height: 0,
      left: box.left,
      right: box.right,
      width: box.width,
    };
  } catch {
    return null;
  }
}
