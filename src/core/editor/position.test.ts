import { describe, expect, it } from 'vitest';

import { applyScrollRatio, readScrollRatio, typewriterScrollTop, type ScrollMetrics } from './position.js';

/**
 * 这两个函数只有四行，但它们是**三个内核共用的除零边界**。
 *
 * 用普通对象而不是真实 DOM：jsdom 没有布局引擎，`scrollHeight` 恒等于
 * `clientHeight`，拿真实元素测的话「有可滚动空间」这条分支永远走不到 ——
 * 而那正是唯一会出错的路径。
 */

function metrics(scrollTop: number, scrollHeight: number, clientHeight: number): ScrollMetrics {
  return { scrollTop, scrollHeight, clientHeight };
}

describe('readScrollRatio', () => {
  it('滚到顶是 0，滚到底是 1', () => {
    expect(readScrollRatio(metrics(0, 1000, 200))).toBe(0);
    expect(readScrollRatio(metrics(800, 1000, 200))).toBe(1);
  });

  it('中间按比例算', () => {
    expect(readScrollRatio(metrics(400, 1000, 200))).toBe(0.5);
  });

  it('内容比视口短时返回 0，不是 NaN', () => {
    // 这条是重点：`NaN` 写进存储会被 JSON 变成 `null`，
    // 恢复时再乘一次得到 `NaN` —— 表现是「滚动位置随机」
    const ratio = readScrollRatio(metrics(0, 200, 200));
    expect(ratio).toBe(0);
    expect(Number.isNaN(ratio)).toBe(false);
  });

  it('容器高度为 0（还没布局）时也不炸', () => {
    expect(readScrollRatio(metrics(0, 0, 0))).toBe(0);
  });

  it('比例被夹在 0–1（浏览器可能报出略超界的 scrollTop）', () => {
    expect(readScrollRatio(metrics(-5, 1000, 200))).toBe(0);
    expect(readScrollRatio(metrics(900, 1000, 200))).toBe(1);
  });
});

describe('applyScrollRatio', () => {
  it('按比例还原 scrollTop', () => {
    const element = metrics(0, 1000, 200);
    applyScrollRatio(element, 0.5);
    expect(element.scrollTop).toBe(400);
  });

  it('没有可滚动空间时**什么都不做**，而不是写 0', () => {
    // 那可能是「内容还没渲染完」的中间态，硬写 0 会抹掉用户原来的位置
    const element = metrics(123, 200, 200);
    applyScrollRatio(element, 0.5);
    expect(element.scrollTop).toBe(123);
  });

  it('与 readScrollRatio 往返一致', () => {
    const source = metrics(350, 1000, 200);
    const target = metrics(0, 1000, 200);
    applyScrollRatio(target, readScrollRatio(source));
    expect(target.scrollTop).toBeCloseTo(350, 6);
  });

  it('越界的比例被夹住', () => {
    const element = metrics(0, 1000, 200);
    applyScrollRatio(element, 5);
    expect(element.scrollTop).toBe(800);
    applyScrollRatio(element, -1);
    expect(element.scrollTop).toBe(0);
  });

  it('比例是 NaN 时退化成 0，不把 scrollTop 写成 NaN', () => {
    const element = metrics(100, 1000, 200);
    applyScrollRatio(element, Number.NaN);
    expect(element.scrollTop).toBe(0);
  });
});

describe('typewriterScrollTop', () => {
  // 与打字机模式的关系：内核取到光标的视口坐标与滚动容器的几何后喂进来，
  // 这里钉住的是「该不该动、往哪动、边界不越界」这三件事。

  it('光标已经在正中 → null（不动，免得每个字都跟滚动合成器打架）', () => {
    // 光标在视口 400 = 容器顶 100 + 半高 300，scrollTop 随便给，内容位置恒居中
    expect(typewriterScrollTop(400, 100, 0, 600)).toBeNull();
    expect(typewriterScrollTop(400, 100, 500, 600)).toBeNull();
  });

  it('差值不足 1px 也算居中 → null', () => {
    expect(typewriterScrollTop(399.5, 100, 0, 600)).toBeNull();
  });

  it('光标偏下 → 向下滚，把光标抬回中线', () => {
    // 光标视口 600，容器顶 100：内容位置 500，中线应是 scrollTop 200
    expect(typewriterScrollTop(600, 100, 0, 600)).toBe(200);
  });

  it('光标偏上 → 向上滚（负差值）', () => {
    // 光标视口 150，容器顶 100：内容位置 50，中线应是 scrollTop -250，
    // 也就是「当前已经在比中线更靠上的位置」→ 目标比当前小
    expect(typewriterScrollTop(150, 100, 400, 600)).toBe(150);
  });

  it('目标夹到 0：文档开头无法居中，也不许滚成负数', () => {
    // 光标就在视口上部、内容位置 50 → 理论中线 -250 → 夹成 0，
    // 而当前 scrollTop 若已是 0，差值为 0 → null
    expect(typewriterScrollTop(150, 100, 0, 600)).toBeNull();
    expect(typewriterScrollTop(150, 100, 40, 600)).toBe(0);
  });

  it('容器还没布局（高度 0）或几何不是数字 → null', () => {
    expect(typewriterScrollTop(300, 100, 0, 0)).toBeNull();
    expect(typewriterScrollTop(Number.NaN, 100, 0, 600)).toBeNull();
    expect(typewriterScrollTop(300, Number.NaN, 0, 600)).toBeNull();
    expect(typewriterScrollTop(300, 100, Number.NaN, 600)).toBeNull();
  });
});
