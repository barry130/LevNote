import { describe, expect, it } from 'vitest';

import { fuzzyMatch, rankCommands } from './fuzzy.js';

describe('fuzzyMatch', () => {
  it('子序列命中', () => {
    expect(fuzzyMatch('sv', 'save')).not.toBeNull();
    expect(fuzzyMatch('xyz', 'save')).toBeNull();
  });

  it('大小写不敏感', () => {
    expect(fuzzyMatch('SAVE', 'save')).not.toBeNull();
    expect(fuzzyMatch('save', 'SAVE')).not.toBeNull();
  });

  it('空查询匹配一切，分数为 0', () => {
    expect(fuzzyMatch('', 'anything')).toEqual({ score: 0, positions: [] });
  });

  it('空文本不匹配', () => {
    expect(fuzzyMatch('a', '')).toBeNull();
  });

  it('连续命中优于分散命中', () => {
    const tight = fuzzyMatch('save', '保存 save');
    const loose = fuzzyMatch('save', 's a v e');
    expect(tight).not.toBeNull();
    expect(loose).not.toBeNull();
    expect(tight!.score).toBeGreaterThan(loose!.score);
  });

  it('给出命中位置，供 UI 高亮', () => {
    expect(fuzzyMatch('sv', 'save')?.positions).toEqual([0, 2]);
  });
});

describe('rankCommands', () => {
  const items = [
    { id: 'file.save', title: '保存', keywords: ['save', 'baocun'] },
    { id: 'file.saveAs', title: '另存为', keywords: ['save as'] },
    { id: 'view.cycleTheme', title: '切换主题', keywords: ['theme'] },
  ];
  const fields = (item: (typeof items)[number]) => ({ title: item.title, keywords: item.keywords, id: item.id });

  it('空查询按原顺序返回全部', () => {
    expect(rankCommands('', items, fields).map((row) => row.item.id)).toEqual(items.map((item) => item.id));
  });

  it('标题命中排在关键词命中之前', () => {
    // 用户打 `保存` 时，「保存」这条必须在「另存为」（关键词里有 save as）前面
    const ranked = rankCommands('保存', items, fields);
    expect(ranked[0]?.item.id).toBe('file.save');
  });

  it('拼音关键词能命中中文标题', () => {
    const ranked = rankCommands('baocun', items, fields);
    expect(ranked[0]?.item.id).toBe('file.save');
    expect(ranked[0]?.field).toBe('keywords');
  });

  it('全部不命中时返回空数组，而不是退化成显示全部', () => {
    // 退化成「显示全部」会让用户以为搜索没生效
    expect(rankCommands('zzzzz', items, fields)).toEqual([]);
  });

  it('首字母缩写能命中', () => {
    const ranked = rankCommands('fsa', items, fields);
    expect(ranked.map((row) => row.item.id)).toContain('file.saveAs');
  });
});
