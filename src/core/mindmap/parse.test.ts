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
    expect(text).not.toContain('"icons"');
    expect(text).not.toContain('"image"');
  });

  it('可选字段存在时原样往返', () => {
    const body = minimalBody();
    body.root = {
      id: 'n1',
      topic: '根',
      note: '备注',
      collapsed: true,
      style: { color: '#f00' },
      icons: ['1️⃣', '⭐'],
      children: [],
    };
    const { document } = mustParse(file(body));
    expect(document.root.note).toBe('备注');
    expect(document.root.collapsed).toBe(true);
    expect(document.root.style).toEqual({ color: '#f00' });
    expect(document.root.icons).toEqual(['1️⃣', '⭐']);
  });

  it('icons 空数组是「全清过」的如实记录 —— 照读照写，canonical 不破（ADR 0053）', () => {
    const body = minimalBody();
    body.root = { id: 'n1', topic: '根', icons: [], children: [] };
    const first = mustParse(file(body));
    expect(first.issues).toEqual([]);
    expect(first.document.root.icons).toEqual([]);

    const text = serializeMindmap(first.document);
    expect(text).toContain('"icons": []');
    // 空数组折成 undefined 就是「保存后悄悄少一句话」—— 绝不静默（硬约束 2）
    const again = mustParse(text);
    expect(again.document.root.icons).toEqual([]);
  });

  it('image 升格为正字段（ADR 0054）：四键齐全读进类型，不认识的子键原样跟着走', () => {
    const body = minimalBody();
    body.root = {
      id: 'n1',
      topic: '根',
      // caption 不在格式层词表里 —— 整对象照收，只认四键的「干净拷贝」会把它静默丢掉
      image: { url: 'assets/shot.png', width: 640, height: 480, fit: 'contain', caption: '截图' },
      children: [],
    };
    const first = mustParse(file(body));
    expect(first.issues).toEqual([]);
    expect(first.document.root.image).toEqual({
      url: 'assets/shot.png',
      width: 640,
      height: 480,
      fit: 'contain',
      caption: '截图',
    });

    const again = mustParse(serializeMindmap(first.document));
    expect(again.document.root.image).toEqual({
      url: 'assets/shot.png',
      width: 640,
      height: 480,
      fit: 'contain',
      caption: '截图',
    });
  });

  it('image 的 fit 可省略；url/width/height 是内核画图的底线（ADR 0054）', () => {
    const body = minimalBody();
    body.root = { id: 'n1', topic: '根', image: { url: 'a.png', width: 10, height: 20 }, children: [] };
    const first = mustParse(file(body));
    expect(first.issues).toEqual([]);
    expect(first.document.root.image).toEqual({ url: 'a.png', width: 10, height: 20 });
  });

  it('hyperLink 与 tags 升格为正字段（ADR 0055）：读进类型，标签对象不认识的子键原样跟着走', () => {
    const body = minimalBody();
    body.root = {
      id: 'n1',
      topic: '根',
      hyperLink: 'https://example.com/a',
      // riskLevel 不在词表里 —— 整对象照收，只认三键的「干净拷贝」会把它静默丢掉
      tags: ['重要', { text: '风险', className: 'mm-x', style: { color: '#f00' }, riskLevel: '高' }],
      children: [],
    };
    const first = mustParse(file(body));
    expect(first.issues).toEqual([]);
    expect(first.document.root.hyperLink).toBe('https://example.com/a');
    expect(first.document.root.tags).toEqual([
      '重要',
      { text: '风险', className: 'mm-x', style: { color: '#f00' }, riskLevel: '高' },
    ]);

    const again = mustParse(serializeMindmap(first.document));
    expect(again.document.root.hyperLink).toBe('https://example.com/a');
    expect(again.document.root.tags).toEqual([
      '重要',
      { text: '风险', className: 'mm-x', style: { color: '#f00' }, riskLevel: '高' },
    ]);
  });

  it('tags 空数组是「全清过」的如实记录 —— 照读照写（ADR 0055）', () => {
    const body = minimalBody();
    body.root = { id: 'n1', topic: '根', tags: [], children: [] };
    const first = mustParse(file(body));
    expect(first.issues).toEqual([]);
    expect(first.document.root.tags).toEqual([]);

    const text = serializeMindmap(first.document);
    expect(text).toContain('"tags": []');
    const again = mustParse(text);
    expect(again.document.root.tags).toEqual([]);
  });

  it('branchColor 升格为正字段（ADR 0060）：非空字符串读进类型，键序在 tags 之后、children 之前', () => {
    const body = minimalBody();
    body.root = { id: 'n1', topic: '根', tags: ['A'], branchColor: '#e2603f', children: [] };
    const first = mustParse(file(body));
    expect(first.issues).toEqual([]);
    expect(first.document.root.branchColor).toBe('#e2603f');
    expect(first.document.root.extra).toEqual({});

    // 词表外的色值（别的工具写的自定义色）不在格式层的解释范围 —— 照收照写
    const text = serializeMindmap(first.document);
    expect(text.indexOf('"tags"')).toBeLessThan(text.indexOf('"branchColor"'));
    expect(text.indexOf('"branchColor"')).toBeLessThan(text.indexOf('"children"'));
    const again = mustParse(text);
    expect(again.document.root.branchColor).toBe('#e2603f');
    expect(again.document.root.extra).toEqual({});
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
    // `tags` 曾是这里的样本 —— ADR 0055 升格为正字段后它不再进 extra，换 `labels`
    const body = minimalBody();
    body.root = { id: 'n1', topic: '根', children: [], labels: ['a', 'b'], weight: 3 };
    const first = mustParse(file(body));
    expect(codes(first.issues)).toEqual(['UNKNOWN_KEY']);
    expect(first.document.root.extra).toEqual({ labels: ['a', 'b'], weight: 3 });

    // 关键：**过一遍内核会丢的东西，必须先过一遍我们的信封**
    const again = mustParse(serializeMindmap(first.document));
    expect(again.document.root.extra).toEqual({ labels: ['a', 'b'], weight: 3 });
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
    // icons 不做逐项过滤：半好半坏的数组整值转 extra，防「悄悄只留下好的一半」
    { key: 'icons', bad: 42, normalizesTo: 42 },
    { key: 'icons', bad: ['1️⃣', 2], normalizesTo: ['1️⃣', 2] },
    // image 同理整值降级：内核 url/width/height 缺一即不渲染，
    // 「半张图」装作读进来了就是静默丢内容的另一种写法（ADR 0054）
    { key: 'image', bad: 42, normalizesTo: 42 },
    { key: 'image', bad: { url: 'a.png' }, normalizesTo: { url: 'a.png' } },
    { key: 'image', bad: { url: 'a.png', width: 0, height: 10 }, normalizesTo: { url: 'a.png', width: 0, height: 10 } },
    { key: 'image', bad: { url: 'a.png', width: 10, height: 10, fit: 'bogus' }, normalizesTo: { url: 'a.png', width: 10, height: 10, fit: 'bogus' } },
    // hyperLink 只认非空串：内核对假值不渲染，`""` 是画不出来的链接（ADR 0055）
    { key: 'hyperLink', bad: 42, normalizesTo: 42 },
    { key: 'hyperLink', bad: '', normalizesTo: '' },
    // branchColor 同一条纪律：内核对假值 `||` 回落轮转色（`MindElixir.js:1347`），
    // `""` 是画不出来的颜色（ADR 0060）
    { key: 'branchColor', bad: 42, normalizesTo: 42 },
    { key: 'branchColor', bad: '', normalizesTo: '' },
    // tags 不逐项过滤：混进一个坏形状整条转 extra，防「悄悄只留好的一半」（ADR 0055）
    { key: 'tags', bad: '重要', normalizesTo: '重要' },
    { key: 'tags', bad: ['重要', 2], normalizesTo: ['重要', 2] },
    { key: 'tags', bad: [{ className: 'mm-x' }], normalizesTo: [{ className: 'mm-x' }] },
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
