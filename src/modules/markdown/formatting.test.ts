// @vitest-environment jsdom
/**
 * 浮层工具条背后的格式化命令 —— 走真实内核，断言的是**序列化出来的 Markdown**。
 *
 * ## 为什么断言 markdown 而不是断言 DOM
 *
 * 加粗在 DOM 上显示成 `<strong>` 并不说明什么：用户最终存进磁盘的是 markdown，
 * 而 `**加粗**` 与 `<strong>加粗</strong>` 之间隔着一次序列化。断在 markdown 上，
 * 「点一下按钮」到「文件里真的多了两个星号」这条链才算被验证过。
 *
 * ## 这里还顺手钉住了一条**会静默丢数据**的规矩
 *
 * `markdownEmitPlugin` 只在「刚发生过用户输入」的窗口内上报变更，而点按钮不产生
 * 任何 input 事件。所以格式化必须自己打 `USER_EDIT_META`（见 `setup.ts` 的
 * `asUserEdit`）。少了它，`getMarkdown()` 会停在旧内容上、文档不会变脏、
 * 用户按 Ctrl+S 写回磁盘的是**格式化之前**的文本。
 *
 * 所以每个用例除了断言 markdown，还会断言 `onChange` **真的被调用过** ——
 * 前者能靠别的路径碰巧成立，后者不能。
 */

import { afterEach, describe, expect, it } from 'vitest';

import { createMarkdownEditor, type MarkdownEditorInstance } from './setup.js';

const FIND_BASE = { caseSensitive: true, wholeWord: false, regex: false } as const;

interface Harness {
  editor: MarkdownEditorInstance;
  /** 内核报给外壳的内容变更。空数组 = 这次编辑没告诉外壳（就是丢数据）。 */
  reports: string[];
}

let opened: MarkdownEditorInstance | null = null;

afterEach(async () => {
  await opened?.destroy();
  opened = null;
  document.body.innerHTML = '';
});

async function open(source: string): Promise<Harness> {
  const root = document.createElement('div');
  document.body.appendChild(root);
  const reports: string[] = [];
  const editor = await createMarkdownEditor({
    root,
    defaultValue: source,
    onChange: (markdown) => reports.push(markdown),
  });
  opened = editor;
  return { editor, reports };
}

/** 选中编辑器里第一处出现的 `needle`。走查找能力，不碰 ProseMirror 内部。 */
function select(editor: MarkdownEditorInstance, needle: string): void {
  const count = editor.search.find({ ...FIND_BASE, query: needle });
  expect(count, `编辑器里找不到「${needle}」`).toBeGreaterThan(0);
  editor.search.reveal(0);
}

describe('行内格式', () => {
  it('加粗：真的写进 markdown，而且真的报给了外壳', async () => {
    const { editor, reports } = await open('正文 abc 结尾\n');
    select(editor, 'abc');

    editor.formatting.toggleInline('strong');

    expect(editor.getMarkdown()).toContain('**abc**');
    // 这一条才是「不会静默丢数据」的保证：少了 USER_EDIT_META 时它必然为空
    expect(reports).not.toHaveLength(0);
    expect(reports[reports.length - 1]).toContain('**abc**');
  });

  it('斜体 / 删除线 / 行内代码各自写对语法', async () => {
    for (const [format, expected] of [
      ['emphasis', '*abc*'],
      ['strikethrough', '~~abc~~'],
      ['inlineCode', '`abc`'],
    ] as const) {
      const { editor } = await open('正文 abc 结尾\n');
      select(editor, 'abc');
      editor.formatting.toggleInline(format);
      expect(editor.getMarkdown(), format).toContain(expected);
      await editor.destroy();
      opened = null;
    }
  });

  it('再点一次同一个格式就取消（toggle 语义）', async () => {
    const { editor } = await open('正文 abc 结尾\n');
    select(editor, 'abc');

    editor.formatting.toggleInline('strong');
    expect(editor.getMarkdown()).toContain('**abc**');

    // 取消之后选区还在原地，所以这里能直接再点一次
    editor.formatting.toggleInline('strong');
    expect(editor.getMarkdown()).not.toContain('**abc**');
    expect(editor.getMarkdown()).toContain('正文 abc 结尾');
  });
});

describe('链接', () => {
  it('给了地址就加上链接', async () => {
    const { editor } = await open('正文 abc 结尾\n');
    select(editor, 'abc');

    editor.formatting.toggleInline('link', 'https://example.com');

    expect(editor.getMarkdown()).toContain('[abc](https://example.com)');
  });

  it('不给地址时，在链接里就摘掉', async () => {
    const { editor } = await open('[abc](https://example.com) 结尾\n');
    select(editor, 'abc');

    editor.formatting.toggleInline('link');

    expect(editor.getMarkdown()).not.toContain('https://example.com');
    expect(editor.getMarkdown()).toContain('abc 结尾');
  });

  it('不在链接里又不给地址时什么都不做 —— 内核不替用户猜地址', async () => {
    const { editor, reports } = await open('正文 abc 结尾\n');
    select(editor, 'abc');

    editor.formatting.toggleInline('link');

    expect(editor.getMarkdown()).toContain('正文 abc 结尾');
    expect(reports).toHaveLength(0);
  });

  /**
   * 这条钉的是一个很容易写错的分支：改地址必须用 `updateLinkCommand`。
   *
   * 用 `toggleLinkCommand` 改会**把链接删掉** —— 它的底层是 `toggleMark`，
   * 看到选区已经带着 link mark 就执行移除。用户想改个域名，结果链接没了。
   */
  it('已有链接时给新地址是**改地址**，不是把链接删掉', async () => {
    const { editor } = await open('[abc](https://old.example) 结尾\n');
    select(editor, 'abc');

    editor.formatting.toggleInline('link', 'https://new.example');

    expect(editor.getMarkdown()).toContain('https://new.example');
    expect(editor.getMarkdown()).not.toContain('old.example');
    // 文字还在、还是链接 —— 只是指向变了
    expect(editor.getMarkdown()).toContain('[abc]');
  });
});

describe('块级格式', () => {
  it('标题：设成二级，再点一次回到正文', async () => {
    const { editor } = await open('正文 abc 结尾\n');
    select(editor, 'abc');

    editor.formatting.toggleBlock('heading', 2);
    expect(editor.getMarkdown()).toContain('## 正文 abc 结尾');

    // 同一个层级再点一次 = 取消。原地不动会让按钮看起来点坏了
    editor.formatting.toggleBlock('heading', 2);
    expect(editor.getMarkdown()).toContain('正文 abc 结尾');
    expect(editor.getMarkdown()).not.toContain('##');
  });

  it('换一个层级是换标题，不是取消', async () => {
    const { editor } = await open('正文 abc 结尾\n');
    select(editor, 'abc');

    editor.formatting.toggleBlock('heading', 1);
    editor.formatting.toggleBlock('heading', 3);

    expect(editor.getMarkdown()).toContain('### 正文 abc 结尾');
  });

  it('引用 / 无序列表 / 有序列表 / 代码块', async () => {
    const cases: Array<[Parameters<MarkdownEditorInstance['formatting']['toggleBlock']>[0], RegExp]> = [
      ['blockquote', /^> 正文 abc 结尾$/m],
      // 序列化器把无序列表写成 `* ` 而不是 `- `（两者等价，remark 选了前者）。
      // 断言写成 `- ` 会失败 —— 但那不是功能坏了，是我们记错了序列化器的口味
      ['bulletList', /^[*+-] 正文 abc 结尾$/m],
      ['orderedList', /^1\. 正文 abc 结尾$/m],
      ['codeBlock', /^```/m],
    ];

    for (const [format, expected] of cases) {
      const { editor } = await open('正文 abc 结尾\n');
      select(editor, 'abc');
      editor.formatting.toggleBlock(format);
      expect(editor.getMarkdown(), format).toMatch(expected);
      await editor.destroy();
      opened = null;
    }
  });

  /**
   * 从列表 / 引用里退出来必须**先出容器**，不能只调 `setBlockType(paragraph)`。
   *
   * 列表里的文本块本来就已经是段落，`setBlockType(paragraph)` 会「成功但什么都没改」，
   * 用户看到的是点了「正文」没反应 —— 这是本功能里最容易漏掉的一条分支。
   */
  it('从无序列表退回正文：列表标记要真的消失', async () => {
    const { editor } = await open('- 正文 abc 结尾\n');
    select(editor, 'abc');

    editor.formatting.toggleBlock('paragraph');

    // 用正则而不是 `toContain('- ')`：序列化器会把 `- ` 写成 `* `，
    // 只查 `- ` 的话这条断言在「什么都没做」时也会通过（假绿）
    expect(editor.getMarkdown()).not.toMatch(/^[*+-] /m);
    expect(editor.getMarkdown()).toContain('正文 abc 结尾');
  });

  it('从引用退回正文：引用标记要真的消失', async () => {
    const { editor } = await open('> 正文 abc 结尾\n');
    select(editor, 'abc');

    editor.formatting.toggleBlock('paragraph');

    expect(editor.getMarkdown()).not.toMatch(/^>/m);
    expect(editor.getMarkdown()).toContain('正文 abc 结尾');
  });

  it('再点一次列表按钮会退出列表', async () => {
    const { editor } = await open('- 正文 abc 结尾\n');
    select(editor, 'abc');

    editor.formatting.toggleBlock('bulletList');

    expect(editor.getMarkdown()).not.toMatch(/^[*+-] /m);
  });
});
