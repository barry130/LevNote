import { describe, expect, it } from 'vitest';

import { highlightCode, tokenizeHighlightedHtml } from './hljs.js';

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
