import { describe, expect, it } from 'vitest';

import { buildSampleDocument, SAMPLE_DOCUMENT_NAME } from './sample-document.js';

describe('示例文档（B1）', () => {
  it('文件名与内容的基本形状', () => {
    expect(SAMPLE_DOCUMENT_NAME).toBe('轻笺示例.md');
    const content = buildSampleDocument();
    // 以一级标题开头：大纲面板与文档标题都有着落
    expect(content.startsWith('# ')).toBe(true);
  });

  it('演示四类能力里的可视块：代码块、数学块、Mermaid、draw.io doc-block', () => {
    const content = buildSampleDocument();
    expect(content).toContain('```ts');
    expect(content).toContain('```mermaid');
    expect(content).toContain('$$');
    // draw.io 走 inline 模式：XML 就在文档里，不依赖第二个文件
    expect(content).toContain('kind: diagram');
    expect(content).toContain('mode: inline');
    expect((content.match(/```doc-block/g) ?? []).length).toBe(1);
  });

  it('列表符号只用 "-" —— 与写回盘的 useDashSymbols 同一口径', () => {
    const content = buildSampleDocument();
    // remark-stringify 的默认是 '*'；示例文档若混进 '*' 起头的列表，
    // 「打开 → 什么都没改 → 保存」就会把它重排一遍
    const bulletLines = content.split('\n').filter((line) => /^[+*-] /.test(line));
    expect(bulletLines.length).toBeGreaterThan(0);
    for (const line of bulletLines) {
      expect(line.startsWith('- ')).toBe(true);
    }
  });

  it('doc-block 的 id 在文档内唯一（全文只有这一个 doc-block）', () => {
    const content = buildSampleDocument();
    const ids = [...content.matchAll(/^id: (.+)$/gm)].map((match) => match[1]);
    expect(new Set(ids).size).toBe(ids.length);
  });
});
