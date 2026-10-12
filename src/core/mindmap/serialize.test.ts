import { describe, expect, it } from 'vitest';

import { serializeMindmap } from './serialize.js';
import {
  MINDMAP_DEFAULT_LAYOUT,
  MINDMAP_FORMAT,
  MINDMAP_VERSION,
  type MindmapDocument,
  type MindmapNode,
} from './types.js';

const node = (over: Partial<MindmapNode> = {}): MindmapNode => ({
  id: 'x',
  topic: 't',
  children: [],
  extra: {},
  ...over,
});

const doc = (over: Partial<MindmapDocument> = {}): MindmapDocument => ({
  root: node({ id: 'n1', topic: '根' }),
  layout: MINDMAP_DEFAULT_LAYOUT,
  extra: {},
  version: MINDMAP_VERSION,
  ...over,
});

describe('serializeMindmap · 确定性', () => {
  it('顶层键顺序固定：format → version → layout → 未识别键 → root', () => {
    const text = serializeMindmap(doc({ extra: { zzz: 1, aaa: 2 } }));
    const keys = Object.keys(JSON.parse(text) as Record<string, unknown>);
    expect(keys).toEqual(['format', 'version', 'layout', 'zzz', 'aaa', 'root']);
  });

  it('节点键顺序固定：id → topic → note → collapsed → style → icons → 未识别键 → children', () => {
    const text = serializeMindmap(
      doc({
        root: node({
          id: 'n1',
          topic: '根',
          note: 'n',
          collapsed: true,
          style: {},
          icons: ['1️⃣'],
          extra: { future: 1 },
          children: [node({ id: 'n2', topic: '子' })],
        }),
      }),
    );
    const root = (JSON.parse(text) as { root: Record<string, unknown> }).root;
    expect(Object.keys(root)).toEqual(['id', 'topic', 'note', 'collapsed', 'style', 'icons', 'future', 'children']);
  });

  it('icons：undefined 不写、空数组照写 ——「全清过」是一句话，不是没有（ADR 0053）', () => {
    const withoutIcons = serializeMindmap(doc({ root: node({ id: 'n1', topic: '根' }) }));
    expect(withoutIcons).not.toContain('"icons"');

    const cleared = serializeMindmap(doc({ root: node({ id: 'n1', topic: '根', icons: [] }) }));
    expect(cleared).toContain('"icons": []');
  });

  it('icons 照写显示字形本身 —— 词表外的字形一个字符不动（§5）', () => {
    const text = serializeMindmap(doc({ root: node({ id: 'n1', topic: '根', icons: ['1️⃣', '🦆'] }) }));
    const root = (JSON.parse(text) as { root: { icons?: string[] } }).root;
    expect(root.icons).toEqual(['1️⃣', '🦆']);
  });

  it('同一份文档序列化两次得到完全一样的文本', () => {
    const document = doc({ root: node({ id: 'n1', topic: '根', children: [node({ id: 'n2', topic: '子' })] }) });
    expect(serializeMindmap(document)).toBe(serializeMindmap(document));
  });

  it('末尾恰好一个换行', () => {
    const text = serializeMindmap(doc());
    expect(text.endsWith('\n')).toBe(true);
    expect(text.endsWith('\n\n')).toBe(false);
  });
});

describe('serializeMindmap · 缩进与行尾', () => {
  it('默认缩进 2 空格', () => {
    expect(serializeMindmap(doc())).toContain('\n  "version"');
  });

  it('缩进可配', () => {
    expect(serializeMindmap(doc(), { indent: 4 })).toContain('\n    "version"');
  });

  it('行尾可配 —— 行尾是文件属性，序列化器不自己猜（§4）', () => {
    const text = serializeMindmap(doc(), { lineEnding: '\r\n' });
    expect(text).toContain('\r\n');
    expect(text.replace(/\r\n/g, '')).not.toContain('\n');
  });

  it('内容里的换行不会被行尾替换碰到', () => {
    const text = serializeMindmap(doc({ root: node({ id: 'n1', topic: '第一行\n第二行' }) }), {
      lineEnding: '\r\n',
    });
    // 字符串内部的换行被 JSON 转义成两个字符，所以整篇里不该出现「裸的 \n」
    expect(text.replace(/\r\n/g, '')).not.toContain('\n');
    expect(text).toContain('第一行\\n第二行');
  });
});

describe('serializeMindmap · 绝不静默丢数据', () => {
  it('extra 里的值如果是 undefined → 抛错，而不是让 JSON.stringify 悄悄略掉它', () => {
    const document = doc({ root: node({ id: 'n1', topic: '根', extra: { future: undefined } }) });
    expect(() => serializeMindmap(document)).toThrow(/undefined/);
  });

  it('顶层 extra 里的 undefined 同样抛错', () => {
    const document = doc({ extra: { future: undefined } });
    expect(() => serializeMindmap(document)).toThrow(/undefined/);
  });

  it('节点 extra 里混进 `id` / `topic` / `children` → 抛错（它们永远会被写出来）', () => {
    for (const key of ['id', 'topic', 'children']) {
      const document = doc({ root: node({ id: 'n1', topic: '根', extra: { [key]: '别的' } }) });
      expect(() => serializeMindmap(document)).toThrow(/两个来源/);
    }
  });

  it('`note` 只在**两边都有**时才抛错 —— 这正是类型不对的值唯一的落脚点', () => {
    // 一边有：可以，就是 parse 把 `"note": 42` 转进 extra 之后的样子
    const onlyExtra = doc({ root: node({ id: 'n1', topic: '根', extra: { note: 42 } }) });
    expect(serializeMindmap(onlyExtra)).toContain('"note": 42');

    // 两边都有：重复键，必须炸
    const both = doc({ root: node({ id: 'n1', topic: '根', note: '正常', extra: { note: 42 } }) });
    expect(() => serializeMindmap(both)).toThrow(/两个来源/);
  });

  it('顶层 extra 里混进已知键 → 抛错', () => {
    const document = doc({ extra: { format: 'someone.else' } });
    expect(() => serializeMindmap(document)).toThrow(/已知键/);
  });

  it('null 是合法的「空」，不抛错', () => {
    const document = doc({ root: node({ id: 'n1', topic: '根', extra: { future: null } }) });
    expect(serializeMindmap(document)).toContain('"future": null');
  });
});

describe('serializeMindmap · 规范示例', () => {
  it('§7.1 的骨架能被写出来（键与值都对得上）', () => {
    const document: MindmapDocument = {
      root: node({
        id: 'n1',
        topic: '根主题',
        note: '',
        collapsed: false,
        style: {},
        children: [node({ id: 'n2', topic: '分支' })],
      }),
      layout: MINDMAP_DEFAULT_LAYOUT,
      extra: {},
      version: MINDMAP_VERSION,
    };
    const parsed = JSON.parse(serializeMindmap(document)) as Record<string, unknown>;
    expect(parsed.format).toBe(MINDMAP_FORMAT);
    expect(parsed.version).toBe(1);
    expect(parsed.layout).toBe('logicalStructure');
    expect(parsed.root).toEqual({
      id: 'n1',
      topic: '根主题',
      note: '',
      collapsed: false,
      style: {},
      children: [{ id: 'n2', topic: '分支', children: [] }],
    });
  });
});
