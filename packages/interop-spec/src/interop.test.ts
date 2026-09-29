import { describe, expect, it } from 'vitest';

import {
  createDocBlock,
  detectLineEnding,
  escapesRoot,
  neededFenceLength,
  parseDocBlocks,
  serializeDocBlock,
  spliceBlocks,
  validateDocBlocks,
  type DocBlock,
} from './index.js';

/** 规范原文样本：一个 reference 块 + 一个 inline 块 + 正文。 */
const SAMPLE = [
  '# 设计说明',
  '',
  '架构图见下：',
  '',
  '```doc-block',
  'id: arch',
  'kind: diagram',
  'mode: reference',
  'src: diagrams/arch.drawio',
  'title: 整体架构',
  '---',
  '```',
  '',
  '代码块内联：',
  '',
  '````doc-block',
  'id: snippet-1',
  'kind: markdown',
  'mode: inline',
  '---',
  'a',
  '```',
  'b',
  '````',
  '',
  '结尾正文。',
  '',
].join('\n');

describe('parseDocBlocks', () => {
  it('parses reference and inline blocks with correct attributes', () => {
    const { blocks, issues } = parseDocBlocks(SAMPLE);
    expect(blocks).toHaveLength(2);

    const [arch, snippet] = blocks as [DocBlock, DocBlock];
    expect(arch.attrs).toEqual({
      id: 'arch',
      kind: 'diagram',
      mode: 'reference',
      src: 'diagrams/arch.drawio',
      title: '整体架构',
      extra: {},
    });
    expect(arch.payload).toBe('');

    expect(snippet.attrs.id).toBe('snippet-1');
    expect(snippet.attrs.mode).toBe('inline');
    expect(snippet.payload).toBe('a\n```\nb\n');
    expect(snippet.fenceLength).toBe(4);

    // 样本本身完全合规，不该有任何问题
    expect(issues).toEqual([]);
    expect(validateDocBlocks(blocks)).toEqual([]);
  });

  it('keeps every block byte-identical through parse -> serialize', () => {
    const { blocks } = parseDocBlocks(SAMPLE);
    for (const block of blocks) {
      expect(serializeDocBlock(block, { lineEnding: detectLineEnding(block.raw) })).toBe(block.raw);
    }
  });

  it('spliceBlocks with untouched blocks returns the original text exactly', () => {
    const { blocks } = parseDocBlocks(SAMPLE);
    expect(spliceBlocks(SAMPLE, blocks)).toBe(SAMPLE);
  });

  it('records ranges that map back onto the source text', () => {
    const { blocks } = parseDocBlocks(SAMPLE);
    for (const block of blocks) {
      expect(SAMPLE.slice(block.range.start, block.range.end)).toBe(block.raw);
    }
  });

  it('does not absorb the newline after the closing fence into the block range', () => {
    const text = ['```doc-block', 'id: a', 'kind: code', 'mode: inline', '---', 'x', '```', '', 'after'].join('\n');
    const { blocks } = parseDocBlocks(text);
    expect(blocks[0]!.raw.endsWith('```')).toBe(true);
    // 块与块之间的空行留给调用方，不属于任何块
    expect(text.slice(blocks[0]!.range.end)).toBe('\n\nafter');
    expect(spliceBlocks(text, blocks)).toBe(text);
  });

  it('ignores fences whose info string is not exactly doc-block', () => {
    const text = ['```json', '{ "a": 1 }', '```', '', '```doc-blocks', 'id: x', '---', '```'].join('\n');
    expect(parseDocBlocks(text).blocks).toEqual([]);
  });

  it('handles CRLF documents without losing the line ending inside payloads', () => {
    const text = ['```doc-block', 'id: win', 'kind: code', 'mode: inline', '---', 'let a = 1;', '```'].join('\r\n');
    const { blocks } = parseDocBlocks(text);
    expect(blocks).toHaveLength(1);
    // 载荷用原文的行尾，不加不减
    expect(blocks[0]!.payload).toBe('let a = 1;\r\n');
    expect(serializeDocBlock(blocks[0]!, { lineEnding: '\r\n' })).toBe(text);
  });

  it('does not double-escape CRLF when the payload already carries CRLF', () => {
    const text = ['```doc-block', 'id: win2', 'kind: code', 'mode: inline', '---', 'a', 'b', '```'].join('\r\n');
    const { blocks } = parseDocBlocks(text);
    const out = serializeDocBlock(blocks[0]!, { lineEnding: '\r\n' });
    expect(out).toBe(text);
    expect(out).not.toContain('\r\r');
  });

  it('reports a missing terminator instead of guessing where attributes end', () => {
    const text = ['```doc-block', 'id: bad', 'kind: code', '```'].join('\n');
    const { issues } = parseDocBlocks(text);
    expect(issues.map((i) => i.code)).toContain('MISSING_TERMINATOR');
  });

  it('reports an unterminated fence', () => {
    const text = ['```doc-block', 'id: bad', 'kind: code', '---', 'trailing text that looks like body'].join('\n');
    const { issues } = parseDocBlocks(text);
    expect(issues.map((i) => i.code)).toContain('UNTERMINATED_FENCE');
  });

  it('preserves unknown keys so a newer version can round-trip through an older one', () => {
    const text = ['```doc-block', 'id: f1', 'kind: text', 'mode: inline', 'collapsed: true', '---', 'x', '```'].join('\n');
    const { blocks, issues } = parseDocBlocks(text);
    expect(blocks[0]!.attrs.extra).toEqual({ collapsed: 'true' });
    expect(issues.map((i) => i.code)).toContain('UNKNOWN_KEY');
    // 关键性质：老版本读进来写回去，新版本的键还在
    expect(serializeDocBlock(blocks[0]!)).toBe(text);
  });

  it('warns about malformed header lines and keeps going', () => {
    const text = ['```doc-block', 'id: f2', 'kind: text', 'mode: inline', 'this line has no colon', '---', 'x', '```'].join('\n');
    const { blocks, issues } = parseDocBlocks(text);
    expect(blocks).toHaveLength(1);
    expect(issues.map((i) => i.code)).toContain('HEADER_LINE_IGNORED');
  });

  it('accepts an inline block that itself contains a doc-block (longer fence wins)', () => {
    const inner = ['```doc-block', 'id: inner', 'kind: code', 'mode: inline', '---', 'z', '```'].join('\n');
    const outer = ['````doc-block', 'id: outer', 'kind: markdown', 'mode: inline', '---', inner, '````'].join('\n');
    const { blocks } = parseDocBlocks(outer);
    expect(blocks).toHaveLength(1);
    expect(blocks[0]!.attrs.id).toBe('outer');
    expect(blocks[0]!.payload.trimEnd()).toBe(inner);
    expect(serializeDocBlock(blocks[0]!)).toBe(outer);
  });

  it('treats a block kind it does not know as plain text rather than dropping it', () => {
    const text = ['```doc-block', 'id: fut', 'kind: spreadsheet', 'mode: inline', '---', 'data', '```'].join('\n');
    const { blocks, issues } = parseDocBlocks(text);
    expect(blocks[0]!.attrs.kind).toBe('text');
    expect(issues.map((i) => i.code)).toContain('UNKNOWN_KIND');
    expect(blocks[0]!.payload.trimEnd()).toBe('data');
  });
});

describe('validateDocBlocks', () => {
  const blockOf = (header: string, payload = ''): DocBlock => {
    const text = ['```doc-block', header, '---', payload, '```'].join('\n');
    const { blocks } = parseDocBlocks(text);
    return blocks[0]!;
  };

  it('flags a missing id and a malformed id', () => {
    const codes = (b: DocBlock) => validateDocBlocks([b]).map((i) => i.code);
    expect(codes(blockOf('kind: text\nmode: inline'))).toContain('MISSING_ID');
    expect(codes(blockOf('id: has space\nkind: text\nmode: inline'))).toContain('BAD_ID');
    expect(codes(blockOf('id: ok-id_1\nkind: text\nmode: inline'))).not.toContain('BAD_ID');
  });

  it('flags duplicate ids across the document', () => {
    const a = blockOf('id: dup\nkind: text\nmode: inline', 'x');
    const b = blockOf('id: dup\nkind: text\nmode: inline', 'y');
    expect(validateDocBlocks([a, b]).map((i) => i.code)).toContain('DUPLICATE_ID');
  });

  it('requires src in reference mode and rejects absolute or escaping paths', () => {
    const codes = (b: DocBlock) => validateDocBlocks([b]).map((i) => i.code);
    expect(codes(blockOf('id: a\nkind: diagram\nmode: reference'))).toContain('MISSING_SRC');
    expect(codes(blockOf('id: a\nkind: diagram\nmode: reference\nsrc: C:/x/y.drawio'))).toContain('ABSOLUTE_SRC');
    expect(codes(blockOf('id: a\nkind: diagram\nmode: reference\nsrc: /abs/y.drawio'))).toContain('ABSOLUTE_SRC');
    expect(codes(blockOf('id: a\nkind: diagram\nmode: reference\nsrc: a\\b.drawio'))).toContain('BACKSLASH_IN_SRC');
    expect(codes(blockOf('id: a\nkind: diagram\nmode: reference\nsrc: ../../y.drawio'))).toContain('PATH_ESCAPES_ROOT');
  });

  it('warns when a reference block also carries a payload', () => {
    const b = blockOf('id: a\nkind: diagram\nmode: reference\nsrc: a.drawio', 'stray');
    expect(validateDocBlocks([b]).map((i) => i.code)).toContain('PAYLOAD_IN_REFERENCE');
  });

  it('warns about inlining an image', () => {
    const b = blockOf('id: a\nkind: image\nmode: inline', 'binary-ish');
    expect(validateDocBlocks([b]).map((i) => i.code)).toContain('IMAGE_MUST_REFERENCE');
  });
});

describe('createDocBlock / serialize', () => {
  it('produces a block that parses back to the same attributes', () => {
    const created = createDocBlock({
      kind: 'mindmap',
      id: 'mm-1',
      src: 'maps/plan.mindmap',
      title: '排期',
    });
    const { blocks, issues } = parseDocBlocks(created.raw);
    expect(issues).toEqual([]);
    expect(blocks).toHaveLength(1);
    expect(blocks[0]!.attrs).toEqual(created.attrs);
    expect(blocks[0]!.attrs.mode).toBe('reference');
  });

  it('replaces an invalid explicit id with a generated one', () => {
    const created = createDocBlock({ kind: 'code', id: 'bad id!' });
    expect(created.attrs.id).toMatch(/^code-/);
  });

  it('generates unique ids', () => {
    const ids = new Set(Array.from({ length: 200 }, () => createDocBlock({ kind: 'code' }).attrs.id));
    expect(ids.size).toBe(200);
  });

  it('quotes values with leading or trailing whitespace so they survive a round-trip', () => {
    const created = createDocBlock({ kind: 'text', id: 'n1', title: '  padded  ' });
    const { blocks } = parseDocBlocks(created.raw);
    expect(blocks[0]!.attrs.title).toBe('  padded  ');
  });

  it('grows the fence when the payload contains a bare fence line', () => {
    expect(neededFenceLength('a\n```\nb')).toBe(4);
    expect(neededFenceLength('a\n`````\nb')).toBe(6);
    expect(neededFenceLength('a\n```json\nb')).toBe(3);
  });

  it('does not introduce CRLF or trailing newline churn', () => {
    const created = createDocBlock({ kind: 'code', mode: 'inline', payload: 'x\n' });
    expect(created.raw).not.toContain('\r');
    expect(created.raw.endsWith('```')).toBe(true);
  });
});

describe('escapesRoot', () => {
  it('walks relative segments correctly', () => {
    expect(escapesRoot('a/b.drawio')).toBe(false);
    expect(escapesRoot('./a/b.drawio')).toBe(false);
    expect(escapesRoot('a/../b.drawio')).toBe(false);
    expect(escapesRoot('../a.drawio')).toBe(true);
    expect(escapesRoot('a/../../b.drawio')).toBe(true);
  });
});

describe('detectLineEnding', () => {
  it('picks the dominant ending and is stable for edge cases', () => {
    expect(detectLineEnding('')).toBe('\n');
    expect(detectLineEnding('no newline at all')).toBe('\n');
    expect(detectLineEnding('a\nb\nc')).toBe('\n');
    expect(detectLineEnding('a\r\nb\r\nc')).toBe('\r\n');
    expect(detectLineEnding('a\rb\rc')).toBe('\r');
    // 混排取多数
    expect(detectLineEnding('a\r\nb\r\nc\nd')).toBe('\r\n');
    expect(detectLineEnding('a\r\nb\nc\nd')).toBe('\n');
  });
});
