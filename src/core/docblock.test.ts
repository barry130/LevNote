import { describe, expect, it } from 'vitest';

import { parseDocBlock, referencedDiagramSrc } from './docblock.js';

describe('parseDocBlock · 属性区', () => {
  it('读出五个已知键', () => {
    const parsed = parseDocBlock('id: a1\nkind: diagram\nmode: reference\nsrc: fig.drawio\ntitle: 流程图\n---\n');
    expect(parsed.attrs).toMatchObject({
      id: 'a1',
      kind: 'diagram',
      mode: 'reference',
      src: 'fig.drawio',
      title: '流程图',
      extra: {},
    });
  });

  it('冒号后的空白剥掉，空值给空串', () => {
    const parsed = parseDocBlock('id: x\n---\n');
    expect(parsed.attrs.id).toBe('x');
    const empty = parseDocBlock('title:\n---\n');
    expect(empty.attrs.title).toBe('');
  });

  it('成对的双引号剥掉；只有一侧的不算引用', () => {
    expect(parseDocBlock('title: "带 空格 的标题"\n---\n').attrs.title).toBe('带 空格 的标题');
    // 值本身含引号字符的形态（只有开头一侧有引号）
    expect(parseDocBlock('title: "不对称\n---\n').attrs.title).toBe('"不对称');
  });

  it('不认识的键进 extra 原样保留，并报 UNKNOWN_KEY（向前兼容，绝不丢弃）', () => {
    const parsed = parseDocBlock('kind: diagram\nfuture-key: 保留我\n---\n');
    expect(parsed.attrs.extra).toEqual({ 'future-key': '保留我' });
    expect(parsed.errors).toContain('UNKNOWN_KEY');
    // 已知键重复出现时后者覆盖前者；extra 的重复同理
    const twice = parseDocBlock('src: a.drawio\nsrc: b.drawio\n---\n');
    expect(twice.attrs.src).toBe('b.drawio');
  });

  it('同一行的 `key: value` 里值可以含冒号', () => {
    expect(parseDocBlock('src: a:b.drawio\n---\n').attrs.src).toBe('a:b.drawio');
  });

  it('属性区里没有冒号的行跳过：一句注释不该让整个块渲染失败', () => {
    const parsed = parseDocBlock('kind: diagram\n这是随手写的注释\n---\npayload\n');
    expect(parsed.attrs.kind).toBe('diagram');
    expect(parsed.payload).toBe('payload\n');
    expect(parsed.errors).toEqual([]);
  });

  it('空行不算结束标记，`---` 前后夹空行照常解析', () => {
    const parsed = parseDocBlock('kind: diagram\n\n\n---\n\npayload\n');
    expect(parsed.attrs.kind).toBe('diagram');
    expect(parsed.payload).toBe('\npayload\n');
  });

  it('`---   `（带尾随空白）也是结束标记：按 trim 后整行判断', () => {
    const parsed = parseDocBlock('kind: diagram\n---   \npayload\n');
    expect(parsed.payload).toBe('payload\n');
    expect(parsed.errors).toEqual([]);
  });
});

describe('parseDocBlock · 载荷', () => {
  it('载荷逐字节原样，不做任何规范化', () => {
    const payload = '<mxfile>  spaced\n\n  lines  </mxfile>';
    const parsed = parseDocBlock(`kind: diagram\n---\n${payload}`);
    expect(parsed.payload).toBe(payload);
  });

  it('没有结束标记：payload 为 null，报 MISSING_TERMINATOR，属性照读', () => {
    const parsed = parseDocBlock('kind: diagram\nsrc: a.drawio');
    expect(parsed.payload).toBeNull();
    expect(parsed.errors).toContain('MISSING_TERMINATOR');
    expect(parsed.attrs.kind).toBe('diagram');
    // 只有 MISSING_TERMINATOR，别把 UNKNOWN_KEY 也凑出来
    expect(parsed.errors).toEqual(['MISSING_TERMINATOR']);
  });

  it('空内容：什么属性都没有，没有结束标记', () => {
    const parsed = parseDocBlock('');
    expect(parsed.attrs.extra).toEqual({});
    expect(parsed.errors).toEqual(['MISSING_TERMINATOR']);
  });

  it('结束标记之后还有属性行：那是载荷的一部分，不再当属性读', () => {
    const parsed = parseDocBlock('kind: diagram\n---\nsrc: 不是属性\n');
    expect(parsed.attrs.src).toBeNull();
    expect(parsed.payload).toBe('src: 不是属性\n');
  });
});

describe('referencedDiagramSrc', () => {
  it('kind: diagram + src 非空 → 返回 trim 过的 src', () => {
    const parsed = parseDocBlock('kind: diagram\nsrc:  ./figs/a.drawio \n---\n');
    expect(referencedDiagramSrc(parsed)).toBe('./figs/a.drawio');
  });

  it('mode 缺省按 reference（规范 §2.3 的默认语义）', () => {
    const parsed = parseDocBlock('kind: diagram\nsrc: a.drawio\n---\n');
    expect(referencedDiagramSrc(parsed)).toBe('a.drawio');
  });

  it('mode: inline 不是引用 —— 返回 null 让它按普通代码块显示', () => {
    const parsed = parseDocBlock('kind: diagram\nmode: inline\n---\n<mxfile/>');
    expect(referencedDiagramSrc(parsed)).toBeNull();
  });

  it('src 空串 / 纯空白 → null', () => {
    const parsed = parseDocBlock('kind: diagram\nsrc:\n---\n');
    expect(referencedDiagramSrc(parsed)).toBeNull();
    const blank = parseDocBlock('kind: diagram\nsrc:   \n---\n');
    expect(referencedDiagramSrc(parsed)).toBeNull();
    expect(referencedDiagramSrc(blank)).toBeNull();
  });

  it('其它 kind（code / markdown / …）都不渲染成图', () => {
    for (const kind of ['code', 'markdown', 'mindmap', 'image', 'text', 'diagrams']) {
      const parsed = parseDocBlock(`kind: ${kind}\nsrc: a.drawio\n---\n`);
      expect(referencedDiagramSrc(parsed)).toBeNull();
    }
  });

  it('没写 kind 也不是图', () => {
    const parsed = parseDocBlock('src: a.drawio\n---\n');
    expect(referencedDiagramSrc(parsed)).toBeNull();
  });
});
