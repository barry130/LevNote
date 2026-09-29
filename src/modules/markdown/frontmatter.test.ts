import { describe, expect, it } from 'vitest';

import { hasFrontMatter } from './frontmatter.js';

/**
 * 这些用例的价值不在「函数算得对」，而在**两个方向的代价不对称**：
 *
 * - 漏判 → 用户打开笔记，改一个字，元数据被换成 `---` 分隔线，不可逆地毁掉。
 * - 误判 → 这个文件用纯文本打开，内容安全，还有一行提示说明原因。
 *
 * 所以下面的「应当判为 front matter」用例要覆盖各种真实写法，
 * 而「排除项」只挑明显不可能是 front matter 的形状。
 */
describe('hasFrontMatter', () => {
  it('识别最常见的 Obsidian / Hugo 写法', () => {
    const text = ['---', 'title: 我的笔记', 'tags:', '  - 随笔', '---', '', '# 正文'].join('\n');
    expect(hasFrontMatter(text)).toBe(true);
  });

  it('收尾用 ... 也算（YAML 文档结束标记）', () => {
    const text = ['---', 'title: x', '...', '正文'].join('\n');
    expect(hasFrontMatter(text)).toBe(true);
  });

  it('CRLF 行尾也要认出来', () => {
    const text = '---\r\ntitle: x\r\n---\r\n\r\n正文\r\n';
    expect(hasFrontMatter(text)).toBe(true);
  });

  it('收尾标记在 50 行以内就认，不要求紧贴开头', () => {
    const middle = Array.from({ length: 40 }, (_, i) => `key${i}: ${i}`);
    const text = ['---', 'title: x', ...middle, '---', '', '正文'].join('\n');
    expect(hasFrontMatter(text)).toBe(true);
  });

  it('没有收尾标记时不算 —— 那是普通的分隔线', () => {
    const text = ['---', 'title: x', '正文继续'].join('\n');
    expect(hasFrontMatter(text)).toBe(false);
  });

  it('第一行不是 --- 时不算', () => {
    const text = ['', '---', 'title: x', '---'].join('\n');
    expect(hasFrontMatter(text)).toBe(false);
  });

  it('分隔线之间没有 键: 值 时不算', () => {
    // 一篇拿 --- 当章节分隔的普通文章，删掉元数据判定的第三条件就能通过
    const text = ['---', '这是一段普通正文，没有冒号键值对。', '---', '后面还有内容'].join('\n');
    expect(hasFrontMatter(text)).toBe(false);
  });

  it('收尾标记超出了前 50 行就不算（避免把长文的两个分隔线连起来）', () => {
    const middle = Array.from({ length: 60 }, (_, i) => `key${i}: ${i}`);
    const text = ['---', 'title: x', ...middle, '---', '', '正文'].join('\n');
    expect(hasFrontMatter(text)).toBe(false);
  });

  it('空文档与极短文件不炸', () => {
    expect(hasFrontMatter('')).toBe(false);
    expect(hasFrontMatter('---')).toBe(false);
    expect(hasFrontMatter('---\n---')).toBe(false);
  });

  it('普通 Markdown 不受影响', () => {
    const text = ['# 标题', '', '正文里有冒号：这不算键值对。', '', '- 列表项'].join('\n');
    expect(hasFrontMatter(text)).toBe(false);
  });

  it('行内 URL 里的冒号不会误导判定', () => {
    const text = ['见 https://example.com/a:b', '', '## 小节'].join('\n');
    expect(hasFrontMatter(text)).toBe(false);
  });
});
