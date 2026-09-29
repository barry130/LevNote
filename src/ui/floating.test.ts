/**
 * 浮层摆放的算术。
 *
 * ## 它为什么值得单独测
 *
 * 「浮层摆在哪」一直挂着一句免责声明：**只有人眼能验**，因为 jsdom 没有布局引擎，
 * `getBoundingClientRect()` 与 `coordsAtPos()` 全是 0。
 *
 * 但那句话说的只是**输入**测不了。「拿到真实尺寸之后算得对不对」是纯算术 ——
 * 而翻边、夹进视口、浮层比视口还大这三种情况，恰恰是写错之后
 * **在真机上很难复现**（要刚好把窗口拖到那么小）的地方。
 *
 * 所以把算术抽成 `computeFloatingPlacement` 并在这里逐条钉住。
 * 剩下真正只能靠人眼的，就只有「内核量出来的坐标对不对」那一半了。
 */

import { describe, expect, it } from 'vitest';

import type { SelectionRect } from '../core/editor/types.js';
import { computeFloatingPlacement, FLOATING_GAP, FLOATING_MARGIN } from './floating.js';

const VIEWPORT = { width: 1000, height: 800 } as const;

function rect(top: number, left: number, width = 100, height = 20): SelectionRect {
  return { top, left, bottom: top + height, right: left + width, width, height };
}

describe('垂直方向', () => {
  it('优先在上：上方放得下就待在上方', () => {
    const anchor = rect(400, 400);
    const { top } = computeFloatingPlacement(
      anchor,
      { width: 200, height: 40 },
      { vertical: 'above', horizontal: 'center' },
      VIEWPORT,
    );

    expect(top).toBe(anchor.top - FLOATING_GAP - 40);
  });

  it('优先在上但上方放不下：翻到下方（选区在第一行时就是这样）', () => {
    // 锚点贴着视口顶端，上方只剩 2px
    const anchor = rect(2, 400);
    const { top } = computeFloatingPlacement(
      anchor,
      { width: 200, height: 40 },
      { vertical: 'above', horizontal: 'center' },
      VIEWPORT,
    );

    expect(top).toBe(anchor.bottom + FLOATING_GAP);
  });

  it('优先在下：下方放得下就待在下放', () => {
    const anchor = rect(400, 400);
    const { top } = computeFloatingPlacement(
      anchor,
      { width: 200, height: 40 },
      { vertical: 'below', horizontal: 'start' },
      VIEWPORT,
    );

    expect(top).toBe(anchor.bottom + FLOATING_GAP);
  });

  it('优先在下但下方放不下：翻到上方', () => {
    // 锚点贴着视口底端：下方只剩 10px，放不下 40px 的浮层
    const anchor = rect(770, 400);
    const { top } = computeFloatingPlacement(
      anchor,
      { width: 200, height: 40 },
      { vertical: 'below', horizontal: 'start' },
      VIEWPORT,
    );

    expect(top).toBe(anchor.top - FLOATING_GAP - 40);
  });

  it('两侧都放不下时仍然夹在视口内 —— 不能算出一个负数', () => {
    const anchor = rect(390, 400);
    const { top } = computeFloatingPlacement(
      anchor,
      // 比视口还高
      { width: 200, height: 900 },
      { vertical: 'above', horizontal: 'center' },
      VIEWPORT,
    );

    // 上界小于下界时兜底到 margin：左上角至少可见，用户还能操作
    expect(top).toBe(FLOATING_MARGIN);
  });
});

describe('水平方向', () => {
  it('居中：浮层中心对齐锚点中心', () => {
    const anchor = rect(400, 400, 100);
    const { left } = computeFloatingPlacement(
      anchor,
      { width: 200, height: 40 },
      { vertical: 'above', horizontal: 'center' },
      VIEWPORT,
    );

    // 锚点中心 450，浮层宽 200 → 左边 350
    expect(left).toBe(350);
  });

  it('左对齐：浮层左边贴着锚点左边', () => {
    const anchor = rect(400, 400, 100);
    const { left } = computeFloatingPlacement(
      anchor,
      { width: 200, height: 40 },
      { vertical: 'above', horizontal: 'start' },
      VIEWPORT,
    );

    expect(left).toBe(400);
  });

  it('右对齐：浮层右边贴着锚点右边 —— 面板从锚点往**左**长', () => {
    const anchor = rect(400, 400, 100);
    const { left } = computeFloatingPlacement(
      anchor,
      { width: 200, height: 40 },
      { vertical: 'below', horizontal: 'end' },
      VIEWPORT,
    );

    // 锚点右边 500，浮层宽 200 → 左边 300
    expect(left).toBe(300);
  });

  it('右对齐且锚点贴着左边缘时被夹回来', () => {
    const anchor = rect(400, 0, 40);
    const { left } = computeFloatingPlacement(
      anchor,
      { width: 200, height: 40 },
      { vertical: 'below', horizontal: 'end' },
      VIEWPORT,
    );

    expect(left).toBe(FLOATING_MARGIN);
  });

  it('贴着左边缘时被夹回来 —— 不能出现负的 left', () => {
    const anchor = rect(400, 0, 40);
    const { left } = computeFloatingPlacement(
      anchor,
      { width: 200, height: 40 },
      { vertical: 'above', horizontal: 'center' },
      VIEWPORT,
    );

    expect(left).toBe(FLOATING_MARGIN);
  });

  it('贴着右边缘时被夹回来 —— 不能整个推到视口外', () => {
    const anchor = rect(400, 990, 10);
    const { left } = computeFloatingPlacement(
      anchor,
      { width: 200, height: 40 },
      { vertical: 'above', horizontal: 'center' },
      VIEWPORT,
    );

    expect(left).toBe(VIEWPORT.width - FLOATING_MARGIN - 200);
  });
});

describe('参数', () => {
  it('margin 与 gap 可以覆盖，且默认值就是那两个常量', () => {
    const anchor = rect(400, 400);

    const custom = computeFloatingPlacement(
      anchor,
      { width: 200, height: 40 },
      { vertical: 'above', horizontal: 'start', gap: 2, margin: 20 },
      VIEWPORT,
    );
    expect(custom.top).toBe(anchor.top - 2 - 40);

    // 贴到 0 附近才能看出 margin 的作用：默认 8 → 夹到 8，改成 20 → 夹到 20
    const tight = rect(0, -50);
    expect(
      computeFloatingPlacement(
        tight,
        { width: 200, height: 40 },
        { vertical: 'above', horizontal: 'start' },
        VIEWPORT,
      ).left,
    ).toBe(FLOATING_MARGIN);
    expect(
      computeFloatingPlacement(
        tight,
        { width: 200, height: 40 },
        { vertical: 'above', horizontal: 'start', margin: 20 },
        VIEWPORT,
      ).left,
    ).toBe(20);
  });
});
