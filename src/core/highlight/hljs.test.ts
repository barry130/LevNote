import { describe, expect, it, vi } from 'vitest';

import { highlightCode, hljs, tokenizeHighlightedHtml } from './hljs.js';

describe('tokenizeHighlightedHtml', () => {
  it('returns ranges that map back onto the plain source text', () => {
    const tokens = tokenizeHighlightedHtml('<span class="hljs-keyword">const</span> a = 1;');
    expect(tokens).toEqual([{ from: 0, to: 5, className: 'hljs-keyword' }]);

    // 关键性质：区间切片必须等于原文的对应片段
    expect('const a = 1;'.slice(tokens[0]!.from, tokens[0]!.to)).toBe('const');
  });

  it('decodes entities so offsets stay aligned with the source', () => {
    const tokens = tokenizeHighlightedHtml('<span class="hljs-string">&lt;div&gt;</span> &amp; more');
    expect(tokens[0]).toEqual({ from: 0, to: 5, className: 'hljs-string' });

    // 实体解码后，区间切出来的必须是解码后的原文，而不是带 &lt; 的形式
    expect('<div> & more'.slice(tokens[0]!.from, tokens[0]!.to)).toBe('<div>');
  });

  it('handles nested spans and reports outer before inner', () => {
    const tokens = tokenizeHighlightedHtml(
      '<span class="hljs-title"><span class="hljs-keyword">if</span> main</span>',
    );
    expect(tokens).toHaveLength(2);
    expect(tokens[0]).toEqual({ from: 0, to: 7, className: 'hljs-title' });
    expect(tokens[1]).toEqual({ from: 0, to: 2, className: 'hljs-keyword' });
  });

  it('ignores markup that carries no class and drops empty ranges', () => {
    expect(tokenizeHighlightedHtml('plain text')).toEqual([]);
    expect(tokenizeHighlightedHtml('<span>no class</span>')).toEqual([]);
  });

  it('is tolerant of unbalanced markup instead of throwing', () => {
    const tokens = tokenizeHighlightedHtml('<span class="hljs-keyword">const');
    expect(tokens).toEqual([{ from: 0, to: 5, className: 'hljs-keyword' }]);
  });
});

describe('highlightCode', () => {
  it('highlights a known language and reports which one it used', () => {
    const result = highlightCode('const a = 1;', 'js');
    expect(result).not.toBeNull();
    expect(result!.language).toBe('js');
    expect(result!.html).toContain('hljs-');
  });

  it('falls back to auto-detection for an unknown language instead of giving up', () => {
    const result = highlightCode('<html><body>hi</body></html>', 'definitely-not-a-language');
    expect(result).not.toBeNull();
    expect(result!.html).toContain('hljs-');
  });

  it('returns null for blank code so the caller can render a plain block', () => {
    expect(highlightCode('')).toBeNull();
    expect(highlightCode('   \n  ')).toBeNull();
  });

  it('never throws on hostile input', () => {
    const nasty = ['\u0000\u0001', '<script>alert(1)</script>', '```', '<?php echo 1; ?>'];
    for (const sample of nasty) {
      expect(() => highlightCode(sample, 'js')).not.toThrow();
    }
  });
});

/**
 * 缓存的存在理由是性能（见 `hljs.ts` 头部），但它同时也是一条**行为**：
 * 第二次调用返回的是同一个对象，调用方拿到的东西没有变。
 *
 * 这里用 `vi.spyOn` 数底层调用次数 —— 那是唯一能证明「真的没重算」的办法。
 * 光测「两次结果相等」是不够的：那在**没有缓存**时也成立。
 */
describe('highlightCode 的缓存', () => {
  it('同一段代码第二次不再调用 highlight.js，且返回同一个对象', () => {
    // 唯一内容，避免与别的用例共享缓存条目（缓存是模块级的，跨用例存活）
    const code = `const cached_${String(Date.now())} = 1;`;
    const spy = vi.spyOn(hljs, 'highlight');

    const first = highlightCode(code, 'js');
    const callsAfterFirst = spy.mock.calls.length;
    const second = highlightCode(code, 'js');

    expect(callsAfterFirst).toBe(1);
    expect(spy.mock.calls.length).toBe(1);
    // 共享同一个对象 —— 调用方都只读，这一点写在 hljs.ts 的注释里
    expect(second).toBe(first);

    spy.mockRestore();
  });

  /**
   * 这条对应**实测里最贵的那个场景**：`mermaid` 不是 hljs 认识的语言，
   * 于是每次高亮都会退回 `highlightAuto()` —— 把全部约 35 种语言各试一遍。
   * 一份文档里 5 个 mermaid 围栏共 605 字符，每次按键重算要 20.8ms。
   */
  it('不认识的语言（mermaid）第二次也不再走 highlightAuto', () => {
    const code = `graph LR\n  A_${String(Date.now())}-->B`;
    const spy = vi.spyOn(hljs, 'highlightAuto');

    highlightCode(code, 'mermaid');
    const callsAfterFirst = spy.mock.calls.length;
    highlightCode(code, 'mermaid');

    expect(callsAfterFirst).toBe(1);
    expect(spy.mock.calls.length).toBe(1);

    spy.mockRestore();
  });

  it('同样的代码在不同语言下是两个条目，不会互相污染', () => {
    const code = `x = ${String(Date.now())}`;
    const asJs = highlightCode(code, 'js');
    const asPython = highlightCode(code, 'python');

    expect(asJs).not.toBeNull();
    expect(asPython).not.toBeNull();
    expect(asJs).not.toBe(asPython);
  });
});
