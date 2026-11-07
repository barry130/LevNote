import { describe, expect, it } from 'vitest';

import {
  activeBranchIndex,
  activeFontSize,
  activeSwatchIndex,
  BRANCH_COLORS,
  isBold,
  STYLE_FILL_COLORS,
  STYLE_FONT_SIZES,
  STYLE_TEXT_COLORS,
  type StyleKey,
} from './node-style.js';

/**
 * 词表恰好这四个键，与内核**导出侧**认得的范围逐字对齐
 * （`LayoutSsr.js:45-47` 只映射这四样）。
 *
 * 这行同时是一道**编译期**的钉子：谁给 `StyleKey` 加了第五个成员，
 * 这个对象字面量就少一个属性、`tsc` 当场报错。运行时那半句
 * （`length === 4`）只是把同一条纪律写成读得懂的断言。
 */
const KEYS: Record<StyleKey, true> = { color: true, background: true, fontSize: true, fontWeight: true };

/** 每通道的取值范围（0-255）。 */
const channels = (hex: string): number[] => [1, 3, 5].map((at) => Number.parseInt(hex.slice(at, at + 2), 16));

describe('样式词表 · 自洽', () => {
  it('词表恰好四个键 —— 多一个就会在编辑器里画得出、导出的图里没有', () => {
    expect(Object.keys(KEYS)).toHaveLength(4);
  });

  it('色值不重复 —— 色块按色值作 React key，重复会静默串键', () => {
    expect(new Set(STYLE_TEXT_COLORS).size).toBe(STYLE_TEXT_COLORS.length);
    expect(new Set(STYLE_FILL_COLORS).size).toBe(STYLE_FILL_COLORS.length);
  });

  it('字号不重复且递增 —— 弹层按顺序画，乱序会让「大一号」看起来更小', () => {
    expect(new Set(STYLE_FONT_SIZES).size).toBe(STYLE_FONT_SIZES.length);
    expect([...STYLE_FONT_SIZES].sort((a, b) => a - b)).toEqual([...STYLE_FONT_SIZES]);
  });

  it('字号全是数字 —— 内核导出按 `${v}px` 拼，字符串会被拼成非法值', () => {
    for (const size of STYLE_FONT_SIZES) expect(typeof size).toBe('number');
  });

  it('色值一律是具体的小写十六进制 —— 不能是 `var(...)`（独立 SVG 里解析不出来，ADR 0050）', () => {
    for (const hex of [...STYLE_TEXT_COLORS, ...STYLE_FILL_COLORS]) expect(hex).toMatch(/^#[0-9a-f]{6}$/);
  });

  it('连线词表同一条十六进制纪律（ADR 0060）—— 与默认轮转同源，摘掉即回到轮转', () => {
    for (const hex of BRANCH_COLORS) expect(hex).toMatch(/^#[0-9a-f]{6}$/);
    expect(new Set(BRANCH_COLORS).size).toBe(BRANCH_COLORS.length);
  });

  it('文字色够暗、底色够淡 —— 「淡彩底 + 墨色字」那句注释的可测代理', () => {
    // 浅色纸面上的对比度是个连续量，这里只钉住两端：文字色每个通道都不超过
    // 0xa0，底色每个通道都不低于 0xc0。词表改色时越界就当场看得见，
    // 而不是等哪天真有人在浅底上写浅字。
    for (const hex of STYLE_TEXT_COLORS) expect(Math.max(...channels(hex))).toBeLessThanOrEqual(0xa0);
    for (const hex of STYLE_FILL_COLORS) expect(Math.min(...channels(hex))).toBeGreaterThanOrEqual(0xc0);
  });
});

describe('activeSwatchIndex · 亮哪一块', () => {
  const colors = STYLE_TEXT_COLORS;

  it('词表内的值：给回它的位置（弹层据此点亮）', () => {
    expect(activeSwatchIndex(colors, { color: colors[2] }, 'color')).toBe(2);
    expect(activeSwatchIndex(colors, { color: colors[0] }, 'color')).toBe(0);
  });

  it('没设过（style 缺失 / 键缺失）：-1，一块都不亮', () => {
    expect(activeSwatchIndex(colors, undefined, 'color')).toBe(-1);
    expect(activeSwatchIndex(colors, {}, 'color')).toBe(-1);
  });

  it('词表外的值不亮 —— 界面摘不掉的东西不假装认识（§5 原样保留）', () => {
    expect(activeSwatchIndex(colors, { color: '#123456' }, 'color')).toBe(-1);
    expect(activeSwatchIndex(colors, { color: '#333333' }, 'background')).toBe(-1);
  });

  it('形状不对的值也不亮，且不抛 —— 这个函数面对任意文件内容', () => {
    expect(activeSwatchIndex(colors, { color: 42 }, 'color')).toBe(-1);
    expect(activeSwatchIndex(colors, { color: null }, 'color')).toBe(-1);
    expect(activeSwatchIndex(colors, { color: undefined }, 'color')).toBe(-1);
  });

  it('底色与文字色各查各的键：同一个值挂在另一个键上不算亮', () => {
    const shared = '#fdf3d8';
    expect(activeSwatchIndex(STYLE_FILL_COLORS, { background: shared }, 'background')).toBe(0);
    expect(activeSwatchIndex(STYLE_FILL_COLORS, { color: shared }, 'background')).toBe(-1);
  });
});

describe('activeFontSize · 字号亮哪一档', () => {
  it('数字给回数字 —— 词表认识的形状只有它', () => {
    expect(activeFontSize({ fontSize: 20 })).toBe(20);
    expect(activeFontSize({ fontSize: 14 })).toBe(14);
  });

  it('带单位的字符串不亮：别的工具写的 `20px` 渲染得对，但不是词表那一档', () => {
    expect(activeFontSize({ fontSize: '20px' })).toBeNull();
    expect(activeFontSize({ fontSize: '20' })).toBeNull();
  });

  it('没设过 / 形状不对：null，且不抛', () => {
    expect(activeFontSize(undefined)).toBeNull();
    expect(activeFontSize({})).toBeNull();
    expect(activeFontSize({ fontSize: Number.NaN })).toBeNull();
    expect(activeFontSize({ fontSize: Number.POSITIVE_INFINITY })).toBeNull();
    expect(activeFontSize({ fontSize: null })).toBeNull();
  });
});

describe('activeBranchIndex · 连线亮哪一格（ADR 0060）', () => {
  it('词表内的值：给回它的位置（弹层据此点亮）', () => {
    expect(activeBranchIndex(BRANCH_COLORS, BRANCH_COLORS[3])).toBe(3);
    expect(activeBranchIndex(BRANCH_COLORS, BRANCH_COLORS[0])).toBe(0);
  });

  it('词表外的色值不亮 —— 没有格子代表它，也不假装摘得掉（§5 原样保留）', () => {
    expect(activeBranchIndex(BRANCH_COLORS, '#123456')).toBe(-1);
  });

  it('没设过 / 形状不对 / 空串：-1，且不抛 —— 这个函数面对任意文件内容', () => {
    expect(activeBranchIndex(BRANCH_COLORS, undefined)).toBe(-1);
    expect(activeBranchIndex(BRANCH_COLORS, null)).toBe(-1);
    expect(activeBranchIndex(BRANCH_COLORS, 42)).toBe(-1);
    expect(activeBranchIndex(BRANCH_COLORS, '')).toBe(-1);
  });
});

describe('isBold · 加粗亮不亮', () => {
  it('只有 `bold` 算亮 —— 点它写进去的也正是它', () => {
    expect(isBold({ fontWeight: 'bold' })).toBe(true);
  });

  it('词表外的字重不亮也不假装摘得掉（`600` 一类）', () => {
    expect(isBold({ fontWeight: '600' })).toBe(false);
    expect(isBold({ fontWeight: 700 })).toBe(false);
  });

  it('没设过 / 形状不对：false，且不抛', () => {
    expect(isBold(undefined)).toBe(false);
    expect(isBold({})).toBe(false);
    expect(isBold({ fontWeight: null })).toBe(false);
  });
});
