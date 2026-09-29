import { describe, expect, it } from 'vitest';

import { extractOutline, outlineDepths, stripInlineMarkup } from './outline.js';

describe('ATX 标题', () => {
  it('按 # 的个数定层级，并给出 1 基行号', () => {
    const items = extractOutline('# 一\n\n正文\n\n## 二\n\n### 三\n');
    expect(items).toEqual([
      { level: 1, text: '一', line: 1 },
      { level: 2, text: '二', line: 5 },
      { level: 3, text: '三', line: 7 },
    ]);
  });

  it('结尾的 # 是装饰性的', () => {
    expect(extractOutline('## 标题 ##\n')[0]?.text).toBe('标题');
  });

  it('缩进 4 空格是缩进代码块，不是标题', () => {
    expect(extractOutline('    # 这是代码\n')).toEqual([]);
    // 3 空格以内仍然是标题
    expect(extractOutline('   # 这是标题\n')).toHaveLength(1);
  });

  it('`#` 后面没有空格不算标题', () => {
    expect(extractOutline('#hashtag\n')).toEqual([]);
  });

  it('空标题也保留（用户可能正在打）', () => {
    expect(extractOutline('#\n')[0]).toEqual({ level: 1, text: '', line: 1 });
  });
});

describe('setext 标题', () => {
  it('=== 是一级，--- 是二级，行号取标题那一行', () => {
    expect(extractOutline('标题\n====\n\n小标题\n----\n')).toEqual([
      { level: 1, text: '标题', line: 1 },
      { level: 2, text: '小标题', line: 4 },
    ]);
  });

  it('上一行是列表项时不当作 setext（那更可能是分隔线 / 列表延续）', () => {
    expect(extractOutline('- 一\n---\n')).toEqual([]);
  });

  it('上一行是空行时不成立', () => {
    expect(extractOutline('\n---\n')).toEqual([]);
  });
});

describe('必须跳过的「假标题」', () => {
  it('围栏代码块内部的一律不算', () => {
    const source = ['# 真标题', '', '```js', '# 这是注释，不是标题', '```', '', '## 又一个真标题', ''].join('\n');
    expect(extractOutline(source).map((item) => item.text)).toEqual(['真标题', '又一个真标题']);
  });

  it('~~~ 围栏同样跳过', () => {
    expect(extractOutline('~~~\n# 注释\n~~~\n')).toEqual([]);
  });

  it('围栏长度不同不会提前结束', () => {
    const source = ['````', '```', '# 还在代码里', '```', '````', '# 出来了', ''].join('\n');
    expect(extractOutline(source).map((item) => item.text)).toEqual(['出来了']);
  });

  it('未闭合的围栏一直吞到文末', () => {
    expect(extractOutline('# 标题\n```\n# 后面全在代码里\n')).toEqual([{ level: 1, text: '标题', line: 1 }]);
  });

  it('YAML front matter 里的注释不算标题', () => {
    const source = ['---', 'title: 笔记', '# 这是 YAML 注释', '---', '', '# 正文标题', ''].join('\n');
    expect(extractOutline(source).map((item) => item.text)).toEqual(['正文标题']);
  });

  it('只有开头那一段 `---` 算 front matter，正文中间的 `---` 是分隔线', () => {
    const source = ['# 一', '', '正文', '', '---', '', '# 二', ''].join('\n');
    expect(extractOutline(source).map((item) => item.text)).toEqual(['一', '二']);
  });

  it('front matter 没有结束标记时不当它是 front matter（否则整篇都进不了大纲）', () => {
    const source = ['---', 'title: 笔记', '', '# 标题', ''].join('\n');
    expect(extractOutline(source).map((item) => item.text)).toEqual(['标题']);
  });
});

describe('stripInlineMarkup', () => {
  it('剥掉常见的行内标记', () => {
    expect(stripInlineMarkup('**加粗** 与 *斜体*')).toBe('加粗 与 斜体');
    expect(stripInlineMarkup('`code`')).toBe('code');
    expect(stripInlineMarkup('[文字](http://x)')).toBe('文字');
    expect(stripInlineMarkup('![图](a.png)')).toBe('图');
    expect(stripInlineMarkup('~~删除~~')).toBe('删除');
  });

  it('图片在链接之前处理 —— 顺序反了会剩下一个 `!`', () => {
    expect(stripInlineMarkup('![替代文字](img/a.png)')).toBe('替代文字');
  });
});

describe('outlineDepths', () => {
  it('把跳级的层级压成连续的缩进深度', () => {
    // `#` 之后直接 `###`：按原始 level 缩进会留下空档
    const items = extractOutline('# 一\n\n### 三\n\n## 二\n\n### 三之二\n');
    expect(outlineDepths(items)).toEqual([0, 1, 1, 2]);
  });

  it('空数组', () => {
    expect(outlineDepths([])).toEqual([]);
  });
});
