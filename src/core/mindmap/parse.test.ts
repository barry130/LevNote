import { describe, expect, it } from 'vitest';

import { parseMindmap } from './parse.js';
import { serializeMindmap } from './serialize.js';
import {
  MINDMAP_DEFAULT_LAYOUT,
  MINDMAP_FORMAT,
  MINDMAP_VERSION,
  type MindmapDocument,
  type MindmapIssue,
  type MindmapNode,
  type ParseMindmapResult,
} from './types.js';

/** 造一份 `.mindmap` 文本。`JSON.stringify(…, 2)` + 末尾换行 = **我们序列化出来的规范形式**。 */
const file = (body: unknown): string => `${JSON.stringify(body, null, 2)}\n`;

/** 最小合法文件。 */
const minimalBody = (): Record<string, unknown> => ({
  format: MINDMAP_FORMAT,
  version: MINDMAP_VERSION,
  layout: MINDMAP_DEFAULT_LAYOUT,
  root: { id: 'n1', topic: '根', children: [] },
});

function mustParse(text: string): ParseMindmapResult & { document: MindmapDocument } {
  const result = parseMindmap(text);
  if (result.document === null) {
    throw new Error(`期望解析成功，实际失败：${result.issues.map((issue) => issue.code).join('、')}`);
  }
  return { ...result, document: result.document };
}

const codes = (issues: MindmapIssue[]): string[] => issues.map((issue) => issue.code);
const node = (over: Partial<MindmapNode> = {}): MindmapNode => ({
  id: 'x',
  topic: 't',
  children: [],
  extra: {},
  ...over,
});

describe('parseMindmap · 正常路径', () => {
  it('最小文件解析成树', () => {
    const { document, issues } = mustParse(file(minimalBody()));
    expect(issues).toEqual([]);
    expect(document.root).toEqual(node({ id: 'n1', topic: '根' }));
    expect(document.layout).toBe(MINDMAP_DEFAULT_LAYOUT);
    expect(document.version).toBe(MINDMAP_VERSION);
    expect(document.extra).toEqual({});
  });

  it('我们自己写出来的文本读回来 `canonical` 为 true（自洽）', () => {
    const document: MindmapDocument = {
      root: node({ id: 'n1', topic: '根', children: [node({ id: 'n2', topic: '分支' })] }),
      layout: MINDMAP_DEFAULT_LAYOUT,
      extra: {},
      version: MINDMAP_VERSION,
    };
    const text = serializeMindmap(document);
    expect(parseMindmap(text).canonical).toBe(true);
  });

  it('缩进不同的文件 `canonical` 为 false，但树是一样的', () => {
    const text = `${JSON.stringify(minimalBody(), null, 4)}\n`;
    const result = mustParse(text);
    expect(result.canonical).toBe(false);
    expect(result.document.root.topic).toBe('根');
  });

  it('可选字段缺省时就不写出来 —— 不补默认值', () => {
    const text = serializeMindmap({
      root: node({ id: 'n1', topic: '根' }),
      layout: MINDMAP_DEFAULT_LAYOUT,
      extra: {},
      version: MINDMAP_VERSION,
    });
    expect(text).not.toContain('"note"');
    expect(text).not.toContain('"collapsed"');
    expect(text).not.toContain('"style"');
  });

  it('可选字段存在时原样往返', () => {
    const body = minimalBody();
    body.root = { id: 'n1', topic: '根', note: '备注', collapsed: true, style: { color: '#f00' }, children: [] };
    const { document } = mustParse(file(body));
    expect(document.root.note).toBe('备注');
    expect(document.root.collapsed).toBe(true);
    expect(document.root.style).toEqual({ color: '#f00' });
  });
});

describe('parseMindmap · 向前兼容（§5 的透传桶）', () => {
  it('顶层未识别的键进 extra，并给一条 UNKNOWN_KEY', () => {
    const body = { ...minimalBody(), futureThing: { a: 1 } };
    const { document, issues } = mustParse(file(body));
    expect(codes(issues)).toEqual(['UNKNOWN_KEY']);
    expect(document.extra).toEqual({ futureThing: { a: 1 } });
  });

  it('节点上未识别的键进该节点的 extra，且**往返后仍在**', () => {
    const body = minimalBody();
    body.root = { id: 'n1', topic: '根', children: [], tags: ['a', 'b'], weight: 3 };
    const first = mustParse(file(body));
    expect(codes(first.issues)).toEqual(['UNKNOWN_KEY']);
    expect(first.document.root.extra).toEqual({ tags: ['a', 'b'], weight: 3 });

    // 关键：**过一遍内核会丢的东西，必须先过一遍我们的信封**
    const again = mustParse(serializeMindmap(first.document));
    expect(again.document.root.extra).toEqual({ tags: ['a', 'b'], weight: 3 });
  });

  it('未识别键的**相对顺序**在往返中不变', () => {
    const text = `{"format":"${MINDMAP_FORMAT}","version":1,"layout":"${MINDMAP_DEFAULT_LAYOUT}","zeta":1,"alpha":2,"mid":3,"root":{"id":"n1","topic":"根","children":[]}}\n`;
    const { document } = mustParse(text);
    expect(Object.keys(document.extra)).toEqual(['zeta', 'alpha', 'mid']);
    const again = mustParse(serializeMindmap(document));
    expect(Object.keys(again.document.extra)).toEqual(['zeta', 'alpha', 'mid']);
  });

  it('版本比我们新 → FUTURE_VERSION，照读；**写回时版本号不变**', () => {
    const body = { ...minimalBody(), version: MINDMAP_VERSION + 1 };
    const { document, issues } = mustParse(file(body));
    expect(codes(issues)).toContain('FUTURE_VERSION');
    expect(document.version).toBe(MINDMAP_VERSION + 1);
    expect(serializeMindmap(document)).toContain(`"version": ${MINDMAP_VERSION + 1}`);
  });

  it('认不出的 layout → UNKNOWN_LAYOUT，且原样写回', () => {
    const body = { ...minimalBody(), layout: 'mindmapFromTheFuture' };
    const { document, issues } = mustParse(file(body));
    expect(codes(issues)).toEqual(['UNKNOWN_LAYOUT']);
    expect(document.layout).toBe('mindmapFromTheFuture');
    expect(serializeMindmap(document)).toContain('"layout": "mindmapFromTheFuture"');
  });

  it('version 不是数字 → BAD_VERSION，按当前版本处理', () => {
    const body = { ...minimalBody(), version: 'one' };
    const { document, issues } = mustParse(file(body));
    expect(codes(issues)).toEqual(['BAD_VERSION']);
    expect(document.version).toBe(MINDMAP_VERSION);
  });
});

describe('parseMindmap · 类型不对的字段一个都不丢', () => {
  const cases: { key: string; bad: unknown; normalizesTo: unknown }[] = [
    { key: 'note', bad: 42, normalizesTo: 42 },
    { key: 'collapsed', bad: 'yes', normalizesTo: 'yes' },
    { key: 'style', bad: [1, 2], normalizesTo: [1, 2] },
  ];

  for (const item of cases) {
    it(`${item.key} 类型不对 → BAD_FIELD，值转进 extra 并原样往返`, () => {
      const body = minimalBody();
      body.root = { id: 'n1', topic: '根', children: [], [item.key]: item.bad };
      const first = mustParse(file(body));
      expect(codes(first.issues)).toContain('BAD_FIELD');
      expect(first.document.root.extra[item.key]).toEqual(item.normalizesTo);

      const again = mustParse(serializeMindmap(first.document));
      expect(again.document.root.extra[item.key]).toEqual(item.normalizesTo);
    });
  }

  it('topic 是数字 → 转成字符串（比丢掉好）', () => {
    const body = minimalBody();
    body.root = { id: 'n1', topic: 2026, children: [] };
    const { document, issues } = mustParse(file(body));
    expect(codes(issues)).toEqual(['MISSING_TOPIC']);
    expect(document.root.topic).toBe('2026');
  });
});

describe('parseMindmap · 结构缺失（warning，补一个能用的值）', () => {
  it('缺 id → 补一个**确定性**的 id（否则每次打开都会产生 diff）', () => {
    const text = file({ ...minimalBody(), root: { topic: '根', children: [] } });
    const first = mustParse(text);
    const second = mustParse(text);
    expect(codes(first.issues)).toEqual(['MISSING_NODE_ID']);
    expect(first.document.root.id).not.toBe('');
    expect(second.document.root.id).toBe(first.document.root.id);
  });

  it('自动 id 不与文件里已有的 id 撞车', () => {
    const text = file({
      ...minimalBody(),
      root: { id: 'auto-0', topic: '根', children: [{ topic: '子', children: [] }] },
    });
    const { document, issues } = mustParse(text);
    expect(codes(issues)).toEqual(['MISSING_NODE_ID']);
    expect(document.root.children[0]?.id).not.toBe('auto-0');
  });

  it('id 重复 → DUPLICATE_NODE_ID（树本身没错，但按 id 索引的旁表会串）', () => {
    const text = file({
      ...minimalBody(),
      root: { id: 'same', topic: '根', children: [{ id: 'same', topic: '子', children: [] }] },
    });
    const { document, issues } = mustParse(text);
    expect(codes(issues)).toEqual(['DUPLICATE_NODE_ID']);
    expect(document.root.children[0]?.id).toBe('same');
  });

  it('缺 topic → 空串 + MISSING_TOPIC', () => {
    const text = file({ ...minimalBody(), root: { id: 'n1', children: [] } });
    const { document, issues } = mustParse(text);
    expect(codes(issues)).toEqual(['MISSING_TOPIC']);
    expect(document.root.topic).toBe('');
  });
});

describe('parseMindmap · 致命问题一律返回 null，绝不抛异常', () => {
  const fatal: { name: string; text: string; code: string }[] = [
    { name: '不是 JSON', text: '这不是 JSON', code: 'NOT_JSON' },
    { name: '顶层是数组', text: '[]\n', code: 'NOT_OBJECT' },
    { name: '顶层是 null', text: 'null\n', code: 'NOT_OBJECT' },
    { name: 'format 不对', text: file({ ...minimalBody(), format: 'someone.else' }), code: 'WRONG_FORMAT' },
    { name: '缺 format', text: file({ version: 1, layout: MINDMAP_DEFAULT_LAYOUT, root: { id: 'n1', topic: 't', children: [] } }), code: 'WRONG_FORMAT' },
    { name: '缺 root', text: file({ format: MINDMAP_FORMAT, version: 1, layout: MINDMAP_DEFAULT_LAYOUT }), code: 'MISSING_ROOT' },
    { name: 'root 是字符串', text: file({ ...minimalBody(), root: 'nope' }), code: 'MISSING_ROOT' },
  ];

  for (const item of fatal) {
    it(`${item.name} → ${item.code}`, () => {
      const run = (): ParseMindmapResult => parseMindmap(item.text);
      expect(run).not.toThrow();
      const result = run();
      expect(result.document).toBeNull();
      expect(codes(result.issues)).toEqual([item.code]);
    });
  }
});

describe('parseMindmap · 树内部的坏值（error 但不致命，剩下的仍然能用）', () => {
  it('children 不是数组 → BAD_CHILDREN（error），其余照常', () => {
    const text = file({ ...minimalBody(), root: { id: 'n1', topic: '根', children: 'oops' } });
    const { document, issues } = mustParse(text);
    expect(codes(issues)).toEqual(['BAD_CHILDREN']);
    expect(issues[0]?.severity).toBe('error');
    expect(document.root.topic).toBe('根');
    expect(document.root.children).toEqual([]);
  });

  it('children 里有非对象 → 跳过那一个 + BAD_NODE，兄弟节点不受影响', () => {
    const text = file({
      ...minimalBody(),
      root: {
        id: 'n1',
        topic: '根',
        children: [{ id: 'n2', topic: '好的', children: [] }, 42, null, { id: 'n3', topic: '也好', children: [] }],
      },
    });
    const { document, issues } = mustParse(text);
    expect(codes(issues)).toEqual(['BAD_NODE', 'BAD_NODE']);
    expect(document.root.children.map((child) => child.topic)).toEqual(['好的', '也好']);
  });
});

describe('parseMindmap · 行尾是文件属性', () => {
  it('CRLF 被识别出来', () => {
    const text = file(minimalBody()).replace(/\n/g, '\r\n');
    const result = mustParse(text);
    expect(result.lineEnding).toBe('\r\n');
  });

  it('CRLF 的文件往返之后还是 CRLF', () => {
    const text = file(minimalBody()).replace(/\n/g, '\r\n');
    const result = mustParse(text);
    const written = serializeMindmap(result.document, { lineEnding: result.lineEnding });
    expect(written).toContain('\r\n');
    expect(written.replace(/\r\n/g, '')).not.toContain('\n');
    expect(parseMindmap(written).canonical).toBe(true);
  });
});

describe('parseMindmap · 透传桶的安全性', () => {
  it('`__proto__` 作为未识别键：不污染原型，且原样往返', () => {
    const text = `{"format":"${MINDMAP_FORMAT}","version":1,"layout":"${MINDMAP_DEFAULT_LAYOUT}","__proto__":{"polluted":true},"root":{"id":"n1","topic":"根","children":[]}}\n`;
    const { document } = mustParse(text);

    expect(Object.getPrototypeOf(document.extra)).toBe(Object.prototype);
    expect(Object.prototype).not.toHaveProperty('polluted');
    expect(Object.prototype.hasOwnProperty.call(document.extra, '__proto__')).toBe(true);

    const again = mustParse(serializeMindmap(document));
    expect(Object.prototype.hasOwnProperty.call(again.document.extra, '__proto__')).toBe(true);
    expect(again.document.extra['__proto__']).toEqual({ polluted: true });
  });
});

describe('往返幂等', () => {
  it('parse → serialize → parse 得到的树与第一次完全相同', () => {
    const text = file({
      ...minimalBody(),
      root: {
        id: 'n1',
        topic: '根',
        collapsed: false,
        children: [{ id: 'n2', topic: '子', children: [], future: 1 }],
        alsoFuture: 'x',
      },
    });
    const first = mustParse(text).document;
    const second = mustParse(serializeMindmap(first)).document;
    expect(second).toEqual(first);
    const third = mustParse(serializeMindmap(second)).document;
    expect(third).toEqual(second);
  });
});
