import { describe, expect, it } from 'vitest';

import { MINDMAP_FORMAT, MINDMAP_VERSION } from '../mindmap/types.js';
import { extractMindmapOutline } from './mindmap-outline.js';

describe('导图大纲（ADR 0052）', () => {
  it('空文件解析不出大纲（面板据此说「暂时读不出」，而不是给假大纲）', () => {
    expect(extractMindmapOutline('')).toEqual([]);
  });

  it('坏 JSON 返回空数组（parseMindmap 绝不抛）', () => {
    expect(extractMindmapOutline('{"format":"levnote.mindmap","root":')).toEqual([]);
  });

  it('根是先序第一个，层级就是树深，line 是先序序号而不是行号', () => {
    const text = JSON.stringify({
      format: MINDMAP_FORMAT,
      version: MINDMAP_VERSION,
      root: {
        id: 'r',
        topic: '根主题',
        children: [
          { id: 'n1', topic: '分支甲', children: [{ id: 'n1-1', topic: '甲的子' }] },
          { id: 'n2', topic: '分支乙' },
        ],
      },
    });
    // 布局无关地摆在多行 JSON 里：line 若真按「行号」算，根在第 3 行 ——
    // 这里断言它是 1（先序序号），把两种定义钉死在测试里。
    expect(extractMindmapOutline(text)).toEqual([
      { level: 1, text: '根主题', line: 1, id: 'r' },
      { level: 2, text: '分支甲', line: 2, id: 'n1' },
      { level: 3, text: '甲的子', line: 3, id: 'n1-1' },
      { level: 2, text: '分支乙', line: 4, id: 'n2' },
    ]);
  });

  it('折叠的子树照样列出（导航要能去到现在看不见的地方）', () => {
    const text = JSON.stringify({
      format: MINDMAP_FORMAT,
      version: MINDMAP_VERSION,
      root: {
        id: 'r',
        topic: '根',
        collapsed: true,
        children: [{ id: 'a', topic: '折叠里的分支' }],
      },
    });
    expect(extractMindmapOutline(text).map((item) => item.id)).toEqual(['r', 'a']);
  });

  it('缺 id 的文件由解析器补 auto- 前缀 —— 与喂给内核的那套 id 同源（跳转全链路一致）', () => {
    const text = JSON.stringify({
      format: MINDMAP_FORMAT,
      version: MINDMAP_VERSION,
      root: { topic: '根', children: [{ topic: '分支' }] },
    });
    const items = extractMindmapOutline(text);
    expect(items.map((item) => item.id)).toEqual(['auto-0', 'auto-1']);
  });

  it('topic 是数字会被解析器转成字符串（大纲与内核看到的同一句话）', () => {
    const text = JSON.stringify({
      format: MINDMAP_FORMAT,
      version: MINDMAP_VERSION,
      root: { id: 'r', topic: 42 },
    });
    expect(extractMindmapOutline(text)).toEqual([{ level: 1, text: '42', line: 1, id: 'r' }]);
  });
});
