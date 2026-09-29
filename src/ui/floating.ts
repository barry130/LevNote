/**
 * 浮层的摆放计算 —— 一个**纯函数**。
 *
 * ## 为什么抽出来
 *
 * M3 的选中工具条与 M4 的斜杠菜单各自写了一份摆放逻辑（十几行 clamp + 翻边）。
 * 到 M5 的表格工具条就是第三份了。ADR 0016 立的规矩是
 * 「共用的部分是一组**机制**就抽，是一组**开关**就别抽」——
 * 「量出尺寸之后摆在哪」正是机制：三处的差别只有「优先在上还是在下」
 * 「居中还是左对齐」，那是两个参数，不是三份代码。
 *
 * ## 为什么它是纯函数，而不是一个 hook
 *
 * 这样它就能被**单元测试**。浮层几何一直挂着一句「只有人眼能验」，
 * 因为 jsdom 没有布局引擎、`getBoundingClientRect()` 全是 0。
 * 但那说的是**输入**测不了 —— 「拿到真实尺寸之后算得对不对」是纯算术，
 * 完全测得了。把算术抽出来，那句免责声明就只剩下一半。
 *
 * 视口尺寸从参数进来（而不是读 `window`）也是同一个理由：测试不需要伪造全局对象。
 */

import type { SelectionRect } from '../core/editor/types.js';

/** 浮层与视口边缘的最小距离。 */
export const FLOATING_MARGIN = 8;
/** 浮层与锚点之间的空隙。 */
export const FLOATING_GAP = 8;

export interface FloatingSize {
  readonly width: number;
  readonly height: number;
}

export interface FloatingPlacement {
  readonly left: number;
  readonly top: number;
}

export interface FloatingViewport {
  readonly width: number;
  readonly height: number;
}

export interface FloatingOptions {
  /** 优先放在锚点的哪一侧。那一侧放不下就翻到另一侧。 */
  readonly vertical: 'above' | 'below';
  /**
   * 水平方向：
   * - `center` —— 浮层中心对齐锚点中心（选中工具条）；
   * - `start` —— 浮层左边贴着锚点左边（斜杠菜单、表格工具条）；
   * - `end` —— 浮层**右边**贴着锚点右边（代码块的语言选择器）。
   *
   * `end` 是 M5 代码块半边加进来的，不是凑数：语言标签贴在代码块的**右上角**，
   * 而代码块占满整栏正文。按 `start` 摆的话面板会从标签往右长，
   * 直接伸到正文栏外面去 —— 再被视口夹回来，就离标签很远了，
   * 用户看不出那个面板是刚点出来的。向右贴住标签才是它该在的位置。
   */
  readonly horizontal: 'center' | 'start' | 'end';
  readonly margin?: number;
  readonly gap?: number;
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max);
}

/**
 * 算出浮层左上角的位置（视口坐标）。
 *
 * ## 两个不能省的 `Math.max`
 *
 * 上界写成 `Math.max(margin, 视口 - margin - 尺寸)` 而不是直接
 * `视口 - margin - 尺寸`：浮层比视口还大时后者是**负数**，
 * `clamp` 的上界小于下界，结果会随实现细节乱跳（把浮层推到屏幕外）。
 * 取 `margin` 作为兜底至少保证左上角可见 —— 那才是用户还能操作的唯一前提。
 *
 * ## 翻边的判据
 *
 * 只判「优先的那一侧放不放得下」，不去比较两侧哪个更宽裕。
 * 比较会带来一个反直觉的行为：浮层明明在上方能完整显示，却因为下方更空
 * 而跳到下方 —— 而浮层离锚点越近，用户越容易把两者联系起来。
 */
export function computeFloatingPlacement(
  anchor: SelectionRect,
  size: FloatingSize,
  options: FloatingOptions,
  viewport: FloatingViewport,
): FloatingPlacement {
  const margin = options.margin ?? FLOATING_MARGIN;
  const gap = options.gap ?? FLOATING_GAP;

  const aboveTop = anchor.top - gap - size.height;
  const belowTop = anchor.bottom + gap;

  const preferred =
    options.vertical === 'above'
      ? aboveTop >= margin
        ? aboveTop
        : belowTop
      : belowTop + size.height <= viewport.height - margin
        ? belowTop
        : aboveTop;

  const top = clamp(preferred, margin, Math.max(margin, viewport.height - margin - size.height));

  const rawLeft =
    options.horizontal === 'center'
      ? anchor.left + anchor.width / 2 - size.width / 2
      : options.horizontal === 'end'
        ? anchor.right - size.width
        : anchor.left;
  const left = clamp(rawLeft, margin, Math.max(margin, viewport.width - margin - size.width));

  return { left, top };
}
