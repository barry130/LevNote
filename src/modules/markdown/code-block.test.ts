// @vitest-environment jsdom
/**
 * 代码块的**内核半边**：行号槽、语言标签、语言选择器，以及「点了标签」怎么报给外壳。
 *
 * ## 这一组测试真正在钉的两条不变量
 *
 * 1. **行号槽与语言标签是纯展示，绝不能进文档。**
 *    它们是我们自己往 NodeView 里塞的 DOM。少了 `ignoreMutation` 的声明，
 *    ProseMirror 会把行号当成用户输入读进文档 —— 存出来的 Markdown 里
 *    会凭空多出一列数字，而且不会有任何报错。
 *    所以「行号槽里有 1 2 3」和「存出来的 markdown 里没有 1 2 3」必须同时断言。
 *
 * 2. **高亮装饰必须仍然落在 `contentDOM` 里。**
 *    换成 NodeView 之后，代码块内部的 DOM 不再由 ProseMirror 直接渲染。
 *    装饰（`codeHighlightPlugin` 的 inline 装饰）还能不能进去，是一个
 *    只能靠实测回答的问题 —— 而它一旦坏掉，编辑器里就是一片黑白，
 *    导出却仍然有颜色（导出走的是另一条路），两边不一致且没人会发现。
 */

import { afterEach, describe, expect, it } from 'vitest';

import type { CodeBlockState } from '../../core/editor/types.js';
import { getLocale, setLocale } from '../../core/i18n/index.js';
import { typeInto } from '../../test/typing.js';
import { lineCount } from './code-block-view.js';
import { createMarkdownEditor, type MarkdownEditorInstance } from './setup.js';

let opened: MarkdownEditorInstance | null = null;

afterEach(async () => {
  await opened?.destroy();
  opened = null;
  document.body.innerHTML = '';
});

interface Harness {
  editor: MarkdownEditorInstance;
  /** 内核报给外壳的语言选择器状态序列。`null` = 面板没开着。 */
  picker: Array<CodeBlockState | null>;
  /** 内核报给外壳的内容变更。空数组 = 这次编辑没告诉外壳（就是丢数据）。 */
  reports: string[];
  /** ProseMirror 的 contenteditable。 */
  dom: HTMLElement;
}

async function open(source: string): Promise<Harness> {
  const root = document.createElement('div');
  document.body.appendChild(root);
  const picker: Array<CodeBlockState | null> = [];
  const reports: string[] = [];

  const editor = await createMarkdownEditor({
    root,
    defaultValue: source,
    onChange: (markdown) => reports.push(markdown),
    onCodeBlockChange: (state) => picker.push(state),
  });
  opened = editor;

  const dom = root.querySelector('.ProseMirror');
  if (!(dom instanceof HTMLElement)) throw new Error('没有找到 ProseMirror 视图');
  return { editor, picker, reports, dom };
}

/** 最近一次上报（理由见 `slash.test.ts` 里的同名函数）。 */
function latest(picker: Array<CodeBlockState | null>): CodeBlockState | null {
  return picker[picker.length - 1] ?? null;
}

function query<T extends Element>(root: HTMLElement, selector: string): T {
  const found = root.querySelector<T>(selector);
  if (found === null) throw new Error(`找不到 ${selector}`);
  return found;
}

/** 行号槽里画出来的号码，按 DOM 顺序。 */
function gutterNumbers(dom: HTMLElement): string[] {
  return [...dom.querySelectorAll('.qm-cb-num')].map((item) => item.textContent ?? '');
}

/** 语言标签上的文字。 */
function badgeLabel(dom: HTMLElement): string {
  return query<HTMLButtonElement>(dom, '.qm-cb-lang').textContent ?? '';
}

/** 点一下语言标签。 */
function clickBadge(dom: HTMLElement): void {
  query<HTMLButtonElement>(dom, '.qm-cb-lang').click();
}

/**
 * 深度优先找 `root` 里最后一个文本节点；一个都没有就在末尾造一个。
 *
 * 与 `test/typing.ts` 的 `caretTarget` 对段落做的事一样，但落点是**代码块**。
 * 不能直接拿 `code.firstChild` —— 高亮装饰会把代码切成一堆
 * `<span class="hljs-…">`，第一个子节点通常是个 span 而不是文本节点，
 * 拿它当落点会把字插到代码最前面（这个坑真踩过一次）。
 */
function lastTextNode(root: Element): Text {
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  let last: Text | null = null;
  for (let node = walker.nextNode(); node !== null; node = walker.nextNode()) {
    last = node as Text;
  }
  if (last !== null) return last;

  const created = document.createTextNode('');
  root.appendChild(created);
  return created;
}

/** 代码块末尾的落点。 */
function codeCaret(editable: HTMLElement): Text {
  return lastTextNode(query(editable, '.qm-cb-code'));
}

/** 第一个段落末尾的落点（用来在代码块**前面**打字）。 */
function firstParagraphCaret(editable: HTMLElement): Text {
  return lastTextNode(query(editable, 'p'));
}

const JS_BLOCK = '```js\nlet a = 1;\nlet b = 2;\nlet c = 3;\n```\n';

// ---------------------------------------------------------------------------
// lineCount —— 纯函数，单独钉住
// ---------------------------------------------------------------------------

describe('lineCount', () => {
  /*
   * 这段算术直接决定行号与代码对不对得上，所以逐种边界都写下来。
   * 判据只有一条：**宁可少算，不能多算**。
   * 多算一行的后果是行号槽比代码高一截，整个代码块跟着长高；
   * 少算一行只是最后那个空行没有号码，肉眼几乎看不出来。
   */
  it.each([
    ['', 1],
    ['a', 1],
    ['a\nb', 2],
    ['a\nb\nc', 3],
    // 末尾换行不算一行
    ['a\n', 1],
    ['a\nb\n', 2],
    // 中间的空行算一行
    ['a\n\nb', 3],
    // 全是换行：n 个换行渲染出 n 个空行（最后一个行尾换行同样被忽略）
    ['\n', 1],
    ['\n\n', 2],
    ['\n\n\n', 3],
  ])('%j → %i 行', (text, expected) => {
    expect(lineCount(text)).toBe(expected);
  });

  /*
   * 真实输入里**没有**尾部换行 —— Milkdown 解析围栏时把内容两端的空白去掉了
   * （实测 ```` ```js\nlet a = 1;\n``` ```` 解析出来的文本就是 `let a = 1;`）。
   * 上面那些带尾换行的用例是**防御性**的：`toMarkdown` 走的是
   * `node.content.firstChild?.text`，将来上游若改了这条路径，这里必须先兜住。
   */
  it('真实解析出来的代码块文本没有尾部换行', async () => {
    const h = await open('```js\nlet a = 1;\nlet b = 2;\n```\n');
    const code = h.dom.querySelector('.qm-cb-code');
    expect(code?.textContent).toBe('let a = 1;\nlet b = 2;');
    expect(gutterNumbers(h.dom)).toEqual(['1', '2']);
  });
});

// ---------------------------------------------------------------------------
// NodeView 的结构
// ---------------------------------------------------------------------------

describe('代码块的 NodeView', () => {
  it('渲染出行号槽、代码本体与语言标签', async () => {
    const h = await open(JS_BLOCK);

    expect(h.dom.querySelector('.qm-cb')).not.toBeNull();
    expect(h.dom.querySelector('.qm-cb-gutter')).not.toBeNull();
    // 代码本体必须仍然是 `<pre><code>` —— `.qm-doc pre code` 那套共用排版
    // 与导出都指望这个结构
    expect(h.dom.querySelector('.qm-cb-pre > code.qm-cb-code')).not.toBeNull();
  });

  it('行号数量等于代码的行数', async () => {
    const h = await open(JS_BLOCK);
    expect(gutterNumbers(h.dom)).toEqual(['1', '2', '3']);
  });

  it('没有语言的代码块也画行号，标签显示「纯文本」', async () => {
    const h = await open('```\n裸代码\n```\n');

    expect(gutterNumbers(h.dom)).toEqual(['1']);
    expect(badgeLabel(h.dom)).toBe('纯文本');
  });

  it('语言标签显示围栏里写的语言', async () => {
    const h = await open('```python\nprint(1)\n```\n');
    expect(badgeLabel(h.dom)).toBe('python');
  });

  it('语言标签的提示跟着语言走（ADR 0057）', async () => {
    const before = getLocale();
    const h = await open('```\n裸代码\n```\n');

    const badge = () => query<HTMLButtonElement>(h.dom, '.qm-cb-lang');
    expect(badge().title).toBe('代码语言：纯文本（点击选择）');

    setLocale('en');
    // 换语言不重建编辑器：title/aria 是我们自己画的 chrome，
    // 订阅者来了就重画（ADR 0049 同款纪律）
    expect(badge().title).toBe('Code language: Plain text (click to choose)');
    expect(badge().getAttribute('aria-label')).toBe('Code language: Plain text, click to choose');

    // 收尾必须把语言放回去，不污染后面的用例
    setLocale(before);
  });

  it('行号槽不进文档 —— 存出来的 markdown 里没有那一列数字', async () => {
    const h = await open(JS_BLOCK);

    // 前提：行号确实画出来了，否则下面那条断言是空的
    expect(gutterNumbers(h.dom)).toEqual(['1', '2', '3']);
    expect(h.editor.getMarkdown()).toBe('```js\nlet a = 1;\nlet b = 2;\nlet c = 3;\n```\n');
  });

  it('高亮装饰仍然落在代码本体里', async () => {
    /*
     * 这是换成 NodeView 之后最需要盯住的一条：装饰是 ProseMirror 往
     * `contentDOM` 的子节点上套的，而 `contentDOM` 现在是我们自己造的
     * `<code>`。装饰进不去的话，编辑器里就是一片黑白，而且不会报任何错。
     */
    const h = await open('```js\nconst a = 1;\n```\n');
    expect(h.dom.querySelector('.qm-cb-code .hljs-keyword')).not.toBeNull();
  });

  it('在代码块里打字照常进文档（NodeView 没有把输入链路截断）', async () => {
    const h = await open('```js\nlet a = 1;\n```\n');

    // `typeInto` 默认找最后一个 `<p>`，而代码块的文本直接挂在 `<code>` 上，
    // 所以这里自己给一个落点
    await typeInto(h.dom, '!', codeCaret);

    expect(h.editor.getMarkdown()).toContain('let a = 1;!');
    // 行号跟着重算，且没有把数字写进文档
    expect(gutterNumbers(h.dom)).toEqual(['1']);
  });

  it('在代码块里换行，行号跟着长出来', async () => {
    const h = await open('```js\nlet a = 1;\n```\n');
    expect(gutterNumbers(h.dom)).toEqual(['1']);

    await typeInto(h.dom, '\nlet b = 2;', codeCaret);

    expect(h.editor.getMarkdown()).toContain('let a = 1;\nlet b = 2;');
    expect(gutterNumbers(h.dom)).toEqual(['1', '2']);
  });
});

// ---------------------------------------------------------------------------
// 语言选择器
// ---------------------------------------------------------------------------

describe('语言选择器', () => {
  it('点语言标签就把状态报给外壳', async () => {
    const h = await open(JS_BLOCK);
    expect(h.picker).toEqual([]);

    clickBadge(h.dom);

    expect(latest(h.picker)).toEqual({ anchor: expect.any(Object), language: 'js' });
  });

  it('点标签不会把文档弄脏', async () => {
    const h = await open(JS_BLOCK);

    clickBadge(h.dom);

    expect(h.picker.length).toBe(1);
    expect(h.reports).toEqual([]);
  });

  it('设语言：文档真的变了、变脏了，标签也跟着更新', async () => {
    const h = await open('```\n裸代码\n```\n');

    clickBadge(h.dom);
    h.editor.codeBlock.setLanguage('rust');

    expect(h.editor.getMarkdown()).toBe('```rust\n裸代码\n```\n');
    // 不报给外壳的话，按 Ctrl+S 存回磁盘的还是改之前那份
    expect(h.reports.length).toBeGreaterThan(0);
    expect(badgeLabel(h.dom)).toBe('rust');
    // 改完就关面板，不用用户再点一下
    expect(latest(h.picker)).toBeNull();
  });

  it('设成空串 = 清除语言，围栏退回没有 info string 的样子', async () => {
    const h = await open(JS_BLOCK);

    clickBadge(h.dom);
    h.editor.codeBlock.setLanguage('');

    expect(h.editor.getMarkdown()).toBe('```\nlet a = 1;\nlet b = 2;\nlet c = 3;\n```\n');
    expect(badgeLabel(h.dom)).toBe('纯文本');
  });

  it('设成当前就已经是的语言：什么都不做，也不把文档弄脏', async () => {
    /*
     * 面板上「当前就是这个」的那一项是可以点的（```` ```js ```` 补出来的那一行
     * 点下去就等于原样保留）。`setNodeAttribute` 即使写的是同一个值也会
     * 造出一个新节点、产生一次 docChanged —— 于是用户什么都没改，
     * 文档却显示「有未保存的改动」，而按 Ctrl+S 存下去的还是原文件。
     */
    const h = await open(JS_BLOCK);

    clickBadge(h.dom);
    h.editor.codeBlock.setLanguage('js');

    expect(h.reports).toEqual([]);
    expect(h.editor.getMarkdown()).toBe(JS_BLOCK);
  });

  it('close() 收起面板并上报 null，且不动文档', async () => {
    const h = await open(JS_BLOCK);

    clickBadge(h.dom);
    expect(latest(h.picker)).not.toBeNull();

    h.editor.codeBlock.close();

    expect(latest(h.picker)).toBeNull();
    expect(h.reports).toEqual([]);
  });

  it('面板对应的代码块被删掉时上报 null —— 否则面板会点着没反应', async () => {
    const h = await open(JS_BLOCK);

    clickBadge(h.dom);
    expect(latest(h.picker)).not.toBeNull();

    // 整篇换成没有代码块的内容，那个 NodeView 会被销毁
    h.editor.setMarkdown('只是普通正文\n');

    expect(latest(h.picker)).toBeNull();
    expect(h.dom.querySelector('.qm-cb')).toBeNull();
  });

  it('删掉**别的**代码块不会把面板带走', async () => {
    const h = await open('```js\nlet a = 1;\n```\n\n```py\nb = 2\n```\n');

    // 点第一个代码块的标签
    const first = h.dom.querySelector('.qm-cb-lang');
    if (!(first instanceof HTMLButtonElement)) throw new Error('找不到第一个语言标签');
    first.click();
    expect(latest(h.picker)).toEqual({ anchor: expect.any(Object), language: 'js' });

    // 把第二个代码块删掉（整篇替换成只剩第一个）
    h.editor.setMarkdown('```js\nlet a = 1;\n```\n');

    // 面板还在：用户可能正打算给手里这个换语言
    expect(latest(h.picker)).toEqual({ anchor: expect.any(Object), language: 'js' });
  });

  it('面板开着的时候在前面打字，语言仍然写到原来那个代码块上', async () => {
    /*
     * 这条打的是「文档坐标会漂」这个坑：面板记住的是位置，而用户在别处
     * 打字会让位置整体后移。写之前不现问一次的话，`setNodeAttribute`
     * 会把语言写到**另一个**节点上 —— 它不检查节点类型，位置对不上就写错地方，
     * 还不报错。
     */
    const h = await open('正文\n\n```\n裸代码\n```\n');

    clickBadge(h.dom);
    expect(latest(h.picker)).toEqual({ anchor: expect.any(Object), language: '' });

    // 在代码块**前面**那段里打字：代码块的位置整体后移
    await typeInto(h.dom, '新', firstParagraphCaret);

    h.editor.codeBlock.setLanguage('go');

    expect(h.editor.getMarkdown()).toBe('正文新\n\n```go\n裸代码\n```\n');
  });

  it('代码块被删掉之后再设语言：什么都不写，也绝不写到别的节点上', async () => {
    const h = await open('正文\n\n```\n裸代码\n```\n');

    clickBadge(h.dom);

    // 整篇换成没有代码块的内容。那个 NodeView 会被销毁，
    // 面板随之关上（上面那条用例已经钉过），这里要钉的是**写不出去**
    h.editor.setMarkdown('只剩正文\n');
    h.editor.codeBlock.setLanguage('go');

    expect(h.editor.getMarkdown()).toBe('只剩正文\n');
  });
});
