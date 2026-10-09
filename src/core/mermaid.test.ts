import { describe, expect, it } from 'vitest';
import { isMermaidLanguage, MERMAID_LANGUAGE } from './mermaid.js';

describe('isMermaidLanguage', () => {
  it('认 mermaid', () => {
    expect(isMermaidLanguage(MERMAID_LANGUAGE)).toBe(true);
  });

  it('大小写与前后空白都不影响 —— 围栏后的文本是原样存进 language 的', () => {
    expect(isMermaidLanguage('  Mermaid ')).toBe(true);
  });

  it('不认别的别名 —— 认了会出现「轻笺画成图、别人打开只是代码」', () => {
    expect(isMermaidLanguage('mmd')).toBe(false);
    expect(isMermaidLanguage('mermaidjs')).toBe(false);
    expect(isMermaidLanguage('')).toBe(false);
  });
});
