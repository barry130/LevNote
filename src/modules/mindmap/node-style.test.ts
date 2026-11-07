import { describe, expect, it } from 'vitest';

import {
  activeBranchIndex,
  activeFontSize,
  activeSwatchIndex,
  BRANCH_COLORS,
  countDashedBorders,
  isBold,
  STYLE_BORDER_COLORS,
  STYLE_BORDER_LINE_STYLES,
  STYLE_BORDER_WIDTHS,
  STYLE_FILL_COLORS,
  STYLE_FONT_SIZES,
  STYLE_TEXT_COLORS,
  type StyleKey,
} from './node-style.js';
import type { MindmapNode } from '../../core/mindmap/types.js';

/**
 * 词表的键恰好这七个（ADR 0062 之前是四个）：**文本四键**受内核导出侧的
 * 白名单约束（`LayoutSsr.js` 的 foreignObject 只拼四样），**边框三键**
 * 走导出侧的专门通道（节点矩形从 computedStyle 现读 borderColor /
 * borderWidth / borderRadius 到 SVG rect）—— 两半合起来才是「编辑器画得出、
 * 导出也带得走」的完整范围，理由见 node-style.ts 文件头第二条。
 *
 * 这行同时是一道**编译期**的钉子：谁给 `StyleKey` 加了第八个成员，
 * 这个对象字面量就少一个属性、`tsc` 当场报错。运行时那半句
 * （`length === 7`）只是把同一条纪律写成读得懂的断言。
 */
const KEYS: Record<StyleKey, true> = {
  color: true,
  background: true,
  fontSize: true,
  fontWeight: true,
  borderColor: true,
  borderWidth: true,
  borderStyle: true,
};

/** 深拷贝无所谓 —— countDashedBorders 只读，fixture 复用足矣。 */
const node = (style: Record<string, unknown> | undefined, children: MindmapNode[] = []): MindmapNode =>
  ({ id: 'x', topic: 't', children, ...(style === undefined ? {} : { style }) }) as MindmapNode;

/** 每通道的取值范围（0-255）。 */
const channels = (hex: string): number[] => [1, 3, 5].map((at) => Number.parseInt(hex.slice(at, at + 2), 16));

describe('样式词表 · 自洽', () => {
  it('词表恰好七个键 —— 文本四键（导出白名单）+ 边框三键（导出专门通道），第八个键两边都带不走', () => {
    expect(Object.keys(KEYS)).toHaveLength(7);
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

describe('边框词表 · ADR 0062', () => {
  it('粗细档位全是带单位字符串 —— 塞 CSSOM 必须有单位（同字号补 px 的教训反过来用）', () => {
    for (const width of STYLE_BORDER_WIDTHS) expect(width).toMatch(/^\d+(\.\d+)?px$/);
    expect(new Set(STYLE_BORDER_WIDTHS).size).toBe(STYLE_BORDER_WIDTHS.length);
  });

  it('线型档位恰好 solid / dashed 两档 —— 词表就是这两样的全部', () => {
    expect(STYLE_BORDER_LINE_STYLES).toEqual(['solid', 'dashed']);
  });

  it('边框色与文字色是同一张表（名单只有一份），词表外的值照 activeSwatchIndex 纪律不亮', () => {
    expect(STYLE_BORDER_COLORS).toBe(STYLE_TEXT_COLORS);
    expect(activeSwatchIndex(STYLE_BORDER_COLORS, { borderColor: '#8f3b2c' }, 'borderColor')).toBe(1);
    expect(activeSwatchIndex(STYLE_BORDER_COLORS, { borderColor: '#123456' }, 'borderColor')).toBe(-1);
    expect(activeSwatchIndex(STYLE_BORDER_COLORS, { borderWidth: '2px' }, 'borderWidth')).toBe(-1);
  });

  it('粗细 / 线型各查各的键：串了键 = 亮错行（0053 双写教训的家族）', () => {
    expect(activeSwatchIndex(STYLE_BORDER_WIDTHS, { borderWidth: '2px' }, 'borderWidth')).toBe(1);
    expect(activeSwatchIndex(STYLE_BORDER_LINE_STYLES, { borderStyle: 'dashed' }, 'borderStyle')).toBe(1);
    expect(activeSwatchIndex(STYLE_BORDER_LINE_STYLES, { borderColor: 'dashed' }, 'borderStyle')).toBe(-1);
  });
});

describe('countDashedBorders · 虚线导出落差点名（ADR 0062）', () => {
  it('solid 与没设过都不数 —— 它们在导出图里如实', () => {
    expect(countDashedBorders(node(undefined))).toBe(0);
    expect(countDashedBorders(node({ borderStyle: 'solid' }))).toBe(0);
    expect(countDashedBorders(node({ color: '#333333' }))).toBe(0);
  });

  it('dashed 数上 —— 整棵树递归，不是只看根', () => {
    const tree = node(undefined, [
      node({ borderStyle: 'dashed' }, [node({ borderStyle: 'dashed' })]),
      node({ borderStyle: 'dotted' }),
      node(undefined),
    ]);
    expect(countDashedBorders(tree)).toBe(3);
  });

  it('词表外的线型（dotted / double）照数 —— 判据是「导出会画成实线」，不是「是不是词表写的」', () => {
    expect(countDashedBorders(node({ borderStyle: 'dotted' }))).toBe(1);
    expect(countDashedBorders(node({ borderStyle: 'double' }))).toBe(1);
  });

  it('形状不对不数也不抛 —— 这个函数面对任意文件内容', () => {
    expect(countDashedBorders(node({ borderStyle: 42 }))).toBe(0);
    expect(countDashedBorders(node({ borderStyle: null }))).toBe(0);
  });
});
