import { describe, expect, it } from 'vitest';

import { t } from '../../core/i18n/index.js';
import { applyMarkerToggle, markerGroupOf, MINDMAP_MARKER_GROUPS, MINDMAP_MARKERS } from './markers.js';

describe('MINDMAP_MARKERS · 词表自洽', () => {
  it('字形不重复 —— 调色盘按字形作 React key，重复会静默串键', () => {
    const glyphs = MINDMAP_MARKERS.map((marker) => marker.glyph);
    expect(new Set(glyphs).size).toBe(glyphs.length);
  });

  it('每个条目都指向一个存在的组', () => {
    const groupIds = new Set(MINDMAP_MARKER_GROUPS.map((group) => group.id));
    for (const marker of MINDMAP_MARKERS) expect(groupIds.has(marker.group)).toBe(true);
  });

  it('词表里每个 labelKey 都是真的 i18n 键 —— 现查表，查得出文字（不烤空串）', () => {
    for (const marker of MINDMAP_MARKERS) {
      expect(t(marker.labelKey, marker.labelParams)).not.toBe('');
    }
    for (const group of MINDMAP_MARKER_GROUPS) {
      expect(t(group.labelKey)).not.toBe('');
    }
  });

  it('互斥组恰好是 priority 与 color，symbol 不互斥（语义住词表，见文件头）', () => {
    const exclusive = new Set(MINDMAP_MARKER_GROUPS.filter((group) => group.exclusive).map((group) => group.id));
    expect(exclusive).toEqual(new Set(['priority', 'color']));
  });
});

describe('markerGroupOf', () => {
  it('词表内的字形给回自己的组', () => {
    expect(markerGroupOf('1️⃣')).toBe('priority');
    expect(markerGroupOf('🟥')).toBe('color');
    expect(markerGroupOf('⭐')).toBe('symbol');
  });

  it('词表外的字形给 null —— 文件里可以有，调色盘里没有（§5）', () => {
    expect(markerGroupOf('🦆')).toBeNull();
    expect(markerGroupOf('')).toBeNull();
  });
});

describe('applyMarkerToggle · 点一下调色盘', () => {
  it('已有该字形 → 摘掉（任何组都一样：再点一次就是取消）', () => {
    expect(applyMarkerToggle(['1️⃣', '⭐'], '1️⃣')).toEqual(['⭐']);
    expect(applyMarkerToggle(['⭐'], '⭐')).toEqual([]);
  });

  it('互斥组：标优先级 2 摘掉优先级 1 —— 一个节点只有一个优先级', () => {
    expect(applyMarkerToggle(['1️⃣'], '2️⃣')).toEqual(['2️⃣']);
    expect(applyMarkerToggle(['1️⃣', '⭐'], '3️⃣')).toEqual(['⭐', '3️⃣']);
  });

  it('不互斥组：⭐ 与 ✅ 叠在一起是正常用法', () => {
    expect(applyMarkerToggle(['⭐'], '✅')).toEqual(['⭐', '✅']);
  });

  it('追加在末尾：「后标的排在后面」是稳定规则，文件往返不会乱序', () => {
    expect(applyMarkerToggle(['⭐'], '🟥')).toEqual(['⭐', '🟥']);
    expect(applyMarkerToggle(['🟥', '⭐'], '1️⃣')).toEqual(['🟥', '⭐', '1️⃣']);
  });

  it('词表外的字形按不互斥处理 —— 这个函数面对任意文件内容', () => {
    expect(applyMarkerToggle(['🦆'], '⭐')).toEqual(['🦆', '⭐']);
    // 两个「词表外的优先级」也各自保留：文件层从不说「不许」
    expect(applyMarkerToggle(['🦆'], '🐔')).toEqual(['🦆', '🐔']);
  });

  it('不改入参（返回新数组）—— React 状态的纪律', () => {
    const current = ['1️⃣'];
    applyMarkerToggle(current, '2️⃣');
    expect(current).toEqual(['1️⃣']);
  });
});
