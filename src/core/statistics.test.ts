import { describe, expect, it } from 'vitest';

import { analyze, formatStatistics } from './statistics.js';

/**
 * 这些用例钉的是**口径**，不是实现细节。
 *
 * 「字数」在中文写作工具之间并不统一，所以每个断言背后都对应
 * `statistics.ts` 文件头那张表里的一行 —— 改口径时这些用例会红，
 * 那是**故意的**：口径变了必须有人知道。
 */

describe('analyze · 行数', () => {
  it('空文本算 1 行', () => {
    // 与状态栏「第 1 行，第 1 列」保持一致 —— 空文档里光标确实在第 1 行
    expect(analyze('').lines).toBe(1);
  });

  it('按 \\n 计数，结尾换行也算一行', () => {
    expect(analyze('a\nb\nc').lines).toBe(3);
    expect(analyze('a\n').lines).toBe(2);
    expect(analyze('\n').lines).toBe(2);
  });

  it('CRLF 只算一次换行', () => {
    expect(analyze('a\r\nb').lines).toBe(2);
  });
});

describe('analyze · 字数（Word 口径）', () => {
  it('中日韩文字每个算一个字', () => {
    expect(analyze('你好世界').words).toBe(4);
  });

  it('中日韩标点也算一个字', () => {
    // 「你好，世界。」= 4 字 + 2 标点 = 6，与 Microsoft Word 的「字数」一致
    expect(analyze('你好，世界。').words).toBe(6);
  });

  it('拉丁文按词计，不按字母计', () => {
    expect(analyze('hello world').words).toBe(2);
  });

  it('拉丁词内部的撇号 / 连字符 / 下划线不断词', () => {
    expect(analyze("don't well-known snake_case 42").words).toBe(4);
  });

  it('中英混排分别计后相加', () => {
    // 「轻笺 LevNote 编辑器」= 5 个汉字 + 1 个拉丁词
    expect(analyze('轻笺 LevNote 编辑器').words).toBe(6);
  });

  it('半角标点不计入字数', () => {
    // 逗号与句号是半角，按口径不计；两个词仍算 2
    expect(analyze('hello, world.').words).toBe(2);
  });
});

describe('analyze · 字符数', () => {
  it('characters 不含空白，charactersWithSpaces 含', () => {
    const stats = analyze('hello world');
    expect(stats.characters).toBe(10);
    expect(stats.charactersWithSpaces).toBe(11);
  });

  it('全角空格（U+3000）按空白处理，不算字符也不算字', () => {
    const stats = analyze('\u3000');
    expect(stats.characters).toBe(0);
    expect(stats.charactersWithSpaces).toBe(1);
    expect(stats.words).toBe(0);
  });

  it('emoji 按码点算 1 个字符 —— 这正是不能用 text.length 的原因', () => {
    const stats = analyze('😀');
    expect(stats.characters).toBe(1);
    // 如果用 text.length，这里会是 2（UTF-16 代理对）
    expect('😀'.length).toBe(2);
  });

  it('零宽连接符不单独算字符（组合 emoji 不虚高）', () => {
    // 👨‍👩‍👧 = 3 个 emoji + 2 个 ZWJ，码点 5 个，但零宽连接符不计
    expect(analyze('👨‍👩‍👧').characters).toBe(3);
    expect([...'👨‍👩‍👧'].length).toBe(5);
  });
});

describe('analyze · 阅读时长', () => {
  it('空文本为 0', () => {
    expect(analyze('').readingMinutes).toBe(0);
    expect(analyze('   \n  ').readingMinutes).toBe(0);
  });

  it('有内容时至少 1 分钟', () => {
    expect(analyze('你好').readingMinutes).toBe(1);
  });

  it('中文按 300 字/分钟', () => {
    expect(analyze('字'.repeat(300)).readingMinutes).toBe(1);
    expect(analyze('字'.repeat(600)).readingMinutes).toBe(2);
  });

  it('拉丁按 200 词/分钟', () => {
    expect(analyze(Array.from({ length: 200 }, () => 'word').join(' ')).readingMinutes).toBe(1);
    expect(analyze(Array.from({ length: 400 }, () => 'word').join(' ')).readingMinutes).toBe(2);
  });
});

describe('formatStatistics', () => {
  it('空文档不显示「约 0 分钟」', () => {
    // 「约 0 分钟」不是信息，是噪声
    expect(formatStatistics(analyze(''))).toBe('1 行 · 0 字 · 0 字符');
  });

  it('有内容时带上阅读时长', () => {
    expect(formatStatistics(analyze('你好'))).toBe('1 行 · 2 字 · 2 字符 · 约 1 分钟');
  });

  it('多行文档', () => {
    expect(formatStatistics(analyze('# 标题\n\n正文\n'))).toBe('4 行 · 4 字 · 5 字符 · 约 1 分钟');
  });
});
