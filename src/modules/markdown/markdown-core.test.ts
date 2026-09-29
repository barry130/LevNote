// @vitest-environment jsdom
/**
 * Markdown 内核的保真度测试。
 *
 * ## 为什么这个文件比别的测试都重要
 *
 * 富文本 Markdown 编辑器最危险的行为不是「排版难看」，而是**静默改坏文件**：
 * 用户打开笔记、随手改一个字、保存，某个他不认识的结构就消失了。
 * 这类损失不可逆，而且只有当用户事后回看才发现。
 *
 * 真实案例：`![图](a.png)`（没有标题的图片）在 Milkdown 里会抛
 * `RangeError`，**整张图片被丢出文档**。若不是这里放了一张没有标题的图片，
 * 这个 bug 会一直躺在代码里，直到毁掉某个用户的插图笔记。
 *
 * 所以这里的做法是**广谱采样**：把常见 Markdown 构件逐个过一遍真实内核，
 * 断言「结构还在」。没被采到的构件就是未验证的 —— 这份清单本身也是文档。
 */

import { describe, expect, it } from 'vitest';

import { createMarkdownEditor } from './setup.js';

interface Harness {
  html: string;
  markdown: string;
  destroy(): Promise<void>;
}

async function mount(source: string): Promise<Harness> {
  const root = document.createElement('div');
  document.body.appendChild(root);
  const editor = await createMarkdownEditor({
    root,
    defaultValue: source,
    onChange: () => undefined,
  });
  return {
    html: editor.toHtml(),
    // 初始化即规范化，所以这里拿到的是「内核认为的当前内容」，
    // 而不是我们塞进去的原文 —— 后者会把解析损失掩盖掉
    markdown: editor.getMarkdown(),
    destroy: () => editor.destroy(),
  };
}

/** 跑一遍真实内核，返回导出用的 HTML 与规范化后的 markdown。 */
async function roundTrip(source: string): Promise<Harness> {
  const harness = await mount(source);
  await harness.destroy();
  return harness;
}

describe('图片（曾经会整张丢失）', () => {
  it('没有标题的图片必须活下来', async () => {
    const { html, markdown } = await roundTrip('正文\n\n![图](img/a.png)\n');

    expect(html).toContain('<img');
    expect(html).toContain('src="img/a.png"');
    expect(markdown).toContain('![图](img/a.png)');
  });

  it('带标题的图片照常', async () => {
    const { html } = await roundTrip('![图](img/a.png "标题")\n');
    expect(html).toContain('src="img/a.png"');
    expect(html).toContain('title="标题"');
  });

  it('路径里带空格与括号的图片', async () => {
    const { html } = await roundTrip('![图](<my pics/a (1).png>)\n');
    expect(html).toContain('<img');
    expect(html).toContain('my pics/a (1).png');
  });

  it('同一段里的多张图片都活着', async () => {
    const { html } = await roundTrip('![一](1.png) 与 ![二](2.png) 与 ![三](3.png)\n');
    expect(html.match(/<img/g)).toHaveLength(3);
  });
});

describe('常见块级构件', () => {
  it('标题层级与正文', async () => {
    const { html } = await roundTrip('# 一级\n\n## 二级\n\n### 三级\n\n段落\n');
    expect(html).toContain('<h1');
    expect(html).toContain('<h2');
    expect(html).toContain('<h3');
    expect(html).toContain('<p>段落</p>');
  });

  it('无序 / 有序 / 嵌套列表', async () => {
    const { html } = await roundTrip('- 一\n- 二\n  - 二点一\n\n1. 甲\n2. 乙\n');
    expect(html).toContain('<ul');
    expect(html).toContain('<ol');
    // 嵌套列表必须真的嵌套，而不是被拍平
    expect(html).toMatch(/<li[^>]*>[\s\S]*<ul/);
  });

  it('任务列表保留勾选状态', async () => {
    const { html } = await roundTrip('- [ ] 未完成\n- [x] 已完成\n');
    expect(html).toContain('data-checked="false"');
    expect(html).toContain('data-checked="true"');
  });

  it('引用（含嵌套）', async () => {
    const { html } = await roundTrip('> 外层\n>\n> > 内层\n');
    expect(html.match(/<blockquote/g)?.length).toBeGreaterThanOrEqual(2);
  });

  it('代码块保留语言', async () => {
    const { html } = await roundTrip('```python\nprint(1)\n```\n');
    expect(html).toContain('data-language="python"');
  });

  it('没有语言的代码块也被保留', async () => {
    const { html } = await roundTrip('```\n裸代码\n```\n');
    expect(html).toContain('<pre');
    expect(html).toContain('裸代码');
  });

  it('围栏可以用波浪线', async () => {
    const { html } = await roundTrip('~~~js\nlet a = 1;\n~~~\n');
    expect(html).toContain('<pre');
    expect(html).toContain('let a = 1;');
  });

  it('分隔线', async () => {
    const { html } = await roundTrip('上\n\n---\n\n下\n');
    expect(html).toContain('<hr');
  });

  it('表格保留对齐方式', async () => {
    const { html } = await roundTrip('| 左 | 中 | 右 |\n| :-- | :-: | --: |\n| a | b | c |\n');
    expect(html).toContain('<table');
    expect(html).toContain('<th');
    expect(html).toContain('text-align: center');
    expect(html).toContain('text-align: right');
  });
});

describe('行内构件', () => {
  it('加粗 / 斜体 / 删除线 / 行内代码', async () => {
    const { html } = await roundTrip('**粗** *斜* ~~删~~ `码`\n');
    expect(html).toContain('<strong>');
    expect(html).toContain('<em>');
    expect(html).toContain('<del>');
    expect(html).toContain('<code>');
  });

  it('链接（无标题也不会被丢）', async () => {
    const { html } = await roundTrip('[文字](https://example.com)\n');
    expect(html).toContain('href="https://example.com"');
    expect(html).toContain('>文字</a>');
  });

  it('尖括号自动链接', async () => {
    const { html } = await roundTrip('<https://example.com>\n');
    expect(html).toContain('https://example.com');
  });

  it('硬换行', async () => {
    const { html } = await roundTrip('第一行  \n第二行\n');
    expect(html).toContain('<br');
  });

  it('HTML 实体与转义字符不被误解', async () => {
    const { html } = await roundTrip('AT&T 与 5 < 6 与 \\*不是斜体\\*\n');
    expect(html).toContain('AT&amp;T');
    expect(html).toContain('*不是斜体*');
  });

  it('原始 HTML 块被保留而不是吃掉', async () => {
    const { html } = await roundTrip('<details>\n<summary>折叠</summary>\n内容\n</details>\n');
    expect(html).toContain('details');
    expect(html).toContain('折叠');
  });

  it('中文标点与全角字符原样', async () => {
    const source = '他说：“标点——不能变。”\n';
    const { html } = await roundTrip(source);
    expect(html).toContain('他说：“标点——不能变。”');
  });
});

describe('内部不做处理但也不该丢的结构', () => {
  // 这些构件 P2 不渲染（ADR 0008 明确推迟），但**必须原样留在文本里**。
  // 一个渲染不了但不能解析的结构如果被丢掉，用户看到的是「内容凭空少了」。

  it('数学公式保持为文本', async () => {
    const { markdown } = await roundTrip('行内 $a^2+b^2$ 与块级：\n\n$$\n\\int_0^1 x dx\n$$\n');
    expect(markdown).toContain('$a^2+b^2$');
    expect(markdown).toContain('\\int_0^1 x dx');
  });

  it('Mermaid 代码块保持为代码块', async () => {
    const { html, markdown } = await roundTrip('```mermaid\ngraph TD;\n  A-->B;\n```\n');
    expect(html).toContain('data-language="mermaid"');
    expect(markdown).toContain('A-->B;');
  });

  it('脚注引用 / 定义保持为文本', async () => {
    const { markdown } = await roundTrip('正文[^1]\n\n[^1]: 说明\n');
    expect(markdown).toContain('[^1]');
    expect(markdown).toContain('说明');
  });

  it('引用式链接的链接与文字都还在', async () => {
    const { markdown } = await roundTrip('[文字][ref]\n\n[ref]: https://example.com\n');
    expect(markdown).toContain('文字');
    expect(markdown).toContain('https://example.com');
  });

  it('setext 风格标题变成 ATX，但标题与层级不变', async () => {
    const { html } = await roundTrip('标题\n====\n\n小节\n----\n');
    // 语法形式会变（这是规范化的正常行为），语义不能变
    expect(html).toContain('<h1');
    expect(html).toContain('<h2');
    expect(html).toContain('标题');
    expect(html).toContain('小节');
  });
});

describe('变更上报的语义', () => {
  /**
   * 这两条守的是「打开文件后一个字没改，却被显示成有未保存的改动」。
   * 它不是体验问题，而是会引导用户去点保存 —— 一保存就按内核的规范化改写文件，
   * 用户从没碰过的排版（`*a*` → `_a_`）就这么变了。
   */

  async function mountCounting(source: string): Promise<{ changes: string[]; destroy(): Promise<void> }> {
    const root = document.createElement('div');
    document.body.appendChild(root);
    const changes: string[] = [];
    const editor = await createMarkdownEditor({
      root,
      defaultValue: source,
      onChange: (markdown) => changes.push(markdown),
    });
    return { changes, destroy: () => editor.destroy() };
  }

  it('创建编辑器不上报任何变更', async () => {
    const { changes, destroy } = await mountCounting('# 标题\n\n正文\n');
    // 给 trailing 之类会 dispatch 的插件留出机会：它们是同步跑的，
    // 但即使异步执行（防抖），这条断言也应该继续成立
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(changes).toEqual([]);
    await destroy();
  });

  it('文档以代码块结尾时也不上报 —— trailing 会补一个空段落，那不算用户改动', async () => {
    const { changes, destroy } = await mountCounting('```js\nlet a = 1;\n```\n');
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(changes).toEqual([]);
    await destroy();
  });

  it('文档以表格结尾时也不上报', async () => {
    const { changes, destroy } = await mountCounting('| a | b |\n| - | - |\n| 1 | 2 |\n');
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(changes).toEqual([]);
    await destroy();
  });

  it('setMarkdown 不上报 —— 否则「从磁盘重新加载」会让文档立刻变脏', async () => {
    const root = document.createElement('div');
    document.body.appendChild(root);
    const changes: string[] = [];
    const editor = await createMarkdownEditor({
      root,
      defaultValue: '原文',
      onChange: (markdown) => changes.push(markdown),
    });

    editor.setMarkdown('# 新内容\n\n段落\n');
    await new Promise((resolve) => setTimeout(resolve, 250));

    expect(changes).toEqual([]);
    // 但内容确实进去了
    expect(editor.getMarkdown()).toContain('# 新内容');
    await editor.destroy();
  });

  it('setMarkdown 之后 getMarkdown 返回的是规范化写法，不是入参原文', async () => {
    const root = document.createElement('div');
    document.body.appendChild(root);
    const editor = await createMarkdownEditor({ root, defaultValue: '', onChange: () => undefined });

    editor.setMarkdown('标题\n====\n');
    // setext 标题被改写成 ATX —— 外壳拿入参原文当去重基准就会来回成环
    expect(editor.getMarkdown()).toBe('# 标题\n');

    await editor.destroy();
  });

  it('初始就给出规范化后的写法（不是原样回放入参）', async () => {
    const root = document.createElement('div');
    document.body.appendChild(root);
    const editor = await createMarkdownEditor({ root, defaultValue: '标题\n----\n', onChange: () => undefined });

    expect(editor.getMarkdown()).toBe('## 标题\n');
    await editor.destroy();
  });
});

/**
 * 这一组是**可执行的说明书**：内核会把什么样的写法改成什么样。
 *
 * 为什么必须把这些钉下来：用户打开文件改一个字再保存，这些改动就会落盘。
 * 它们语义等价（Markdown 层面无差别），但**字节确实变了** ——
 * 如果哪天有人给文件做 diff、或者用别的工具链处理，会看到一片无关的变更。
 * 这份清单是「我们的保存会改什么」的准确答案，也是将来判断
 * 「值不值得引入更保守的序列化器」时的依据。
 *
 * 反过来说：**不在这个清单里的东西就不该被改写**。这一组同时也是回归保护 ——
 * 哪天升级 Milkdown 后发现 `*斜体*` 变成了 `_斜体_`，这里会红。
 */
describe('序列化器的规范化行为（已知且已接受）', () => {
  it('setext 标题改写成 ATX', async () => {
    expect((await roundTrip('标题\n====\n')).markdown).toBe('# 标题\n');
    expect((await roundTrip('标题\n----\n')).markdown).toBe('## 标题\n');
  });

  it('行尾空白被去掉、连续空行收敛成一个', async () => {
    expect((await roundTrip('行尾有空格   \n\n\n\n下一段\n')).markdown).toBe('行尾有空格\n\n下一段\n');
  });

  it('波浪线围栏改写成反引号围栏', async () => {
    expect((await roundTrip('~~~js\nlet a = 1;\n~~~\n')).markdown).toBe('```js\nlet a = 1;\n```\n');
  });

  it('下划线分隔线改写成星号', async () => {
    expect((await roundTrip('___\n')).markdown).toBe('***\n');
  });

  it('两个行尾空格的硬换行改写成反斜杠', async () => {
    expect((await roundTrip('上行  \n下行\n')).markdown).toBe('上行\\\n下行\n');
  });

  it('有序列表的 1) 改写成 1.', async () => {
    expect((await roundTrip('1) 项目\n')).markdown).toBe('1. 项目\n');
  });

  it('表格对齐语法被重排（补空白、缩短破折号）', async () => {
    const { markdown } = await roundTrip('| 左 | 右 |\n| :-- | --: |\n| a | b |\n');
    // 对齐语义保留，写法变了
    expect(markdown).toContain(':-');
    expect(markdown).toContain('-:');
  });

  it('但强调标记是**保留**的 —— 不在这里的东西不该被改写', async () => {
    // 这条曾经被我误以为会变，实测不会。写下来免得以后又有人想当然
    expect((await roundTrip('这是 *斜体* 与 _也是斜体_ 与 **加粗**\n')).markdown).toBe(
      '这是 *斜体* 与 _也是斜体_ 与 **加粗**\n',
    );
  });

  it('原始 HTML 与自动链接原样保留', async () => {
    expect((await roundTrip('<div>\n块\n</div>\n')).markdown).toBe('<div>\n块\n</div>\n');
    expect((await roundTrip('<https://example.com>\n')).markdown).toBe('<https://example.com>\n');
  });
});

describe('边界情况', () => {
  it('空文档：markdown 是空的，HTML 里只有一个占位空段落', async () => {
    const { html, markdown } = await roundTrip('');
    // ProseMirror 的文档至少有一个块级节点，空文档就是「一个空 paragraph」。
    // 这不是缺陷，是它的文档模型要求；导出时表现为一页空白。
    expect(html).toBe('<p></p>');
    // 但序列化回 markdown 必须是真正的空 —— 否则「打开空文件、保存」会写进一个换行
    expect(markdown.trim()).toBe('');
  });

  it('只有空白字符不炸', async () => {
    await expect(roundTrip('   \n\n  \n')).resolves.toBeDefined();
  });

  it('很长的单行不炸', async () => {
    await expect(roundTrip(`${'很长的一段文字。'.repeat(2000)}\n`)).resolves.toBeDefined();
  });

  it('未闭合的代码围栏不炸（拿它当代码块收尾）', async () => {
    await expect(roundTrip('```js\nlet a = 1;\n')).resolves.toBeDefined();
  });

  it('未闭合的行内标记不炸', async () => {
    await expect(roundTrip('这里有 **没关掉的加粗\n')).resolves.toBeDefined();
  });

  it('只有分隔线不炸', async () => {
    await expect(roundTrip('---\n')).resolves.toBeDefined();
  });
});
