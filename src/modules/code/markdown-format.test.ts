// @vitest-environment jsdom
/**
 * 代码内核的 Markdown 源码级格式化（`markdown-format.ts`）。
 *
 * 分两层：
 *
 * 1. **语义层** —— 对一个最小 EditorView 直接调 `createMarkdownSourceFormatting`，
 *    钉每个动作的源码变换（toggle 语义、链接三态、终态代码块、插入点规则）；
 * 2. **接线层** —— 经 `createCodeEditor` 确认 markdown 语言才有 formatting、
 *    其它语言没有，且动作经真实内核的 `updateListener` 如实上报
 *    （这里没有 `USER_EDIT_META` 那套上膛 —— CodeMirror 不需要，见 `setup.ts`）。
 */

import { afterEach, describe, expect, it } from 'vitest';
import { EditorState } from '@codemirror/state';
import { EditorView } from '@codemirror/view';

import { MERMAID_STARTER } from '../../core/mermaid.js';
import { createMarkdownSourceFormatting } from './markdown-format.js';
import { createCodeEditor, type CodeEditorInstance } from './setup.js';

const opened: Array<{ destroy(): void }> = [];

afterEach(() => {
  for (const instance of opened) instance.destroy();
  opened.length = 0;
  document.body.innerHTML = '';
});

function makeView(doc: string, anchor?: number, head?: number): { view: EditorView; reports: string[] } {
  const root = document.createElement('div');
  document.body.appendChild(root);
  const reports: string[] = [];
  const view = new EditorView({
    state: EditorState.create({
      doc,
      extensions: [
        EditorView.updateListener.of((update) => {
          if (update.docChanged) reports.push(update.state.doc.toString());
        }),
      ],
    }),
    parent: root,
  });
  if (anchor !== undefined) view.dispatch({ selection: { anchor, head: head ?? anchor } });
  opened.push(view);
  return { view, reports };
}

function formatting(view: EditorView) {
  return createMarkdownSourceFormatting(() => view);
}

/** 选中 doc 里 `needle` 的第一次出现，返回这次选区的起点。 */
function select(view: EditorView, needle: string): number {
  const from = view.state.doc.toString().indexOf(needle);
  expect(from, `文档里找不到「${needle}」`).toBeGreaterThanOrEqual(0);
  view.dispatch({ selection: { anchor: from, head: from + needle.length } });
  return from;
}

// ---------------------------------------------------------------------------
// 行内格式
// ---------------------------------------------------------------------------

describe('源码级行内格式', () => {
  it('选中文字包上标记，一个动作一次 dispatch', () => {
    const { view, reports } = makeView('hello world');
    select(view, 'world');

    formatting(view).toggleInline('strong');
    expect(view.state.doc.toString()).toBe('hello **world**');
    expect(reports).toHaveLength(1);
  });

  it('选区本身是「标记 + 内容 + 标记」→ 拆掉', () => {
    const { view } = makeView('hello **world**');
    select(view, '**world**');

    formatting(view).toggleInline('strong');
    expect(view.state.doc.toString()).toBe('hello world');
  });

  it('选区两侧各有一个标记 → 删掉两侧（只圈了内容也能取消）', () => {
    const { view } = makeView('hello **world**');
    select(view, 'world');

    formatting(view).toggleInline('strong');
    expect(view.state.doc.toString()).toBe('hello world');
  });

  it('空选区包上标记、光标落在中间等着打字', () => {
    const { view } = makeView('x', 1);

    formatting(view).toggleInline('emphasis');
    expect(view.state.doc.toString()).toBe('x**');
    expect(view.state.selection.main.anchor).toBe(2);
  });

  it('inlineCode 用反引号', () => {
    const { view } = makeView('run npm install');
    select(view, 'npm install');

    formatting(view).toggleInline('inlineCode');
    expect(view.state.doc.toString()).toBe('run `npm install`');
  });
});

describe('源码级链接', () => {
  it('给地址 → 加链接（空选区放占位并选中文字）', () => {
    const { view } = makeView('看这个');
    select(view, '这个');

    formatting(view).toggleInline('link', 'https://example.dev');
    expect(view.state.doc.toString()).toBe('看[这个](https://example.dev)');
  });

  it('空选区给地址 → 放「链接」占位并选中', () => {
    const { view } = makeView('x', 1);

    formatting(view).toggleInline('link', 'https://example.dev');
    expect(view.state.doc.toString()).toBe('x[链接](https://example.dev)');
    expect(view.state.selection.main.anchor).toBe(2);
    expect(view.state.selection.main.head).toBe(4);
  });

  it('选区是完整链接 → 只改地址', () => {
    const { view } = makeView('看[这个](https://old.dev)');
    select(view, '[这个](https://old.dev)');

    formatting(view).toggleInline('link', 'https://new.dev');
    expect(view.state.doc.toString()).toBe('看[这个](https://new.dev)');
  });

  it('不给地址 → 只去掉；不在链接里就什么都不做', () => {
    const linked = makeView('看[这个](https://old.dev)');
    select(linked.view, '[这个](https://old.dev)');

    formatting(linked.view).toggleInline('link');
    expect(linked.view.state.doc.toString()).toBe('看这个');

    const plain = makeView('就一句话');
    select(plain.view, '一句');
    formatting(plain.view).toggleInline('link');
    // 内核不替用户猜地址
    expect(plain.view.state.doc.toString()).toBe('就一句话');
    expect(plain.reports).toHaveLength(0);
  });
});

// ---------------------------------------------------------------------------
// 块级格式
// ---------------------------------------------------------------------------

describe('源码级块格式', () => {
  /** 块级格式按选区覆盖的行生效 —— 多行的用例先全选。 */
  function selectAll(view: EditorView): void {
    view.dispatch({ selection: { anchor: 0, head: view.state.doc.length } });
  }

  it('标题：加、换级、点同一下取消', () => {
    const { view } = makeView('a\nb');

    selectAll(view);
    formatting(view).toggleBlock('heading', 2);
    expect(view.state.doc.toString()).toBe('## a\n## b');

    selectAll(view);
    formatting(view).toggleBlock('heading', 3);
    expect(view.state.doc.toString()).toBe('### a\n### b');

    selectAll(view);
    formatting(view).toggleBlock('heading', 3);
    expect(view.state.doc.toString()).toBe('a\nb');
  });

  it('塌缩的光标只动所在行 —— 与「段落」的直觉一致', () => {
    const { view } = makeView('a\nb');

    formatting(view).toggleBlock('heading', 1);
    expect(view.state.doc.toString()).toBe('# a\nb');
  });

  it('引用：加与取消（空行也带前缀，整段是一个引用；取消剥净）', () => {
    const { view } = makeView('a\n\nb');

    selectAll(view);
    formatting(view).toggleBlock('blockquote');
    expect(view.state.doc.toString()).toBe('> a\n> \n> b');

    selectAll(view);
    formatting(view).toggleBlock('blockquote');
    expect(view.state.doc.toString()).toBe('a\n\nb');
  });

  it('无序列表：`-` `*` `+` 都算已加，取消一律剥净', () => {
    const { view } = makeView('- a\n* b\n+ c');

    selectAll(view);
    formatting(view).toggleBlock('bulletList');
    expect(view.state.doc.toString()).toBe('a\nb\nc');
  });

  it('列表互转：转有序时编号从 1 重排、先摘掉对方记号', () => {
    const { view } = makeView('- a\n- b');

    selectAll(view);
    formatting(view).toggleBlock('orderedList');
    expect(view.state.doc.toString()).toBe('1. a\n2. b');

    selectAll(view);
    formatting(view).toggleBlock('bulletList');
    expect(view.state.doc.toString()).toBe('- a\n- b');
  });

  it('变为正文：剥掉一层块记号（标题 / 引用 / 列表各算一层）', () => {
    const cases: Array<[string, string]> = [
      ['## 标题', '标题'],
      ['> 引用', '引用'],
      ['- 列表', '列表'],
      ['1. 有序', '有序'],
      ['> - 1. 混合', '混合'],
    ];
    for (const [before, after] of cases) {
      const { view } = makeView(before);
      formatting(view).toggleBlock('paragraph');
      expect(view.state.doc.toString(), `「${before}」变为正文`).toBe(after);
    }
  });

  it('代码块：包上围栏；已在围栏里是终态，再点不动文档', () => {
    const { view, reports } = makeView('x\ny');

    selectAll(view);
    formatting(view).toggleBlock('codeBlock');
    expect(view.state.doc.toString()).toBe('```\nx\ny\n```');
    expect(reports).toHaveLength(1);

    // 光标已在围栏里 —— 不再动作，也不产生新事务
    formatting(view).toggleBlock('codeBlock');
    expect(view.state.doc.toString()).toBe('```\nx\ny\n```');
    expect(reports).toHaveLength(1);
  });
});

// ---------------------------------------------------------------------------
// 插入块
// ---------------------------------------------------------------------------

describe('源码级插入块', () => {
  it('空行上插入：直接落在这行，不另起空行', () => {
    const { view } = makeView('');

    formatting(view).insertBlock('table');
    expect(view.state.doc.toString()).toBe('| 列1 | 列2 | 列3 |\n| --- | --- | --- |\n|  |  |  |\n');
  });

  it('有文字的行后插入：另起一块，原来的字一个不动', () => {
    const { view } = makeView('正文');

    formatting(view).insertBlock('hr');
    expect(view.state.doc.toString()).toBe('正文\n\n---\n');

    const other = makeView('正文');
    formatting(other.view).insertBlock('table');
    expect(other.view.state.doc.toString()).toBe(
      '正文\n\n| 列1 | 列2 | 列3 |\n| --- | --- | --- |\n|  |  |  |\n',
    );
  });

  it('公式块：光标落在空位上等输入', () => {
    const { view } = makeView('');

    formatting(view).insertBlock('mathBlock');
    expect(view.state.doc.toString()).toBe('$$\n\n$$');
    expect(view.state.selection.main.anchor).toBe(3);
  });

  it('图表块：带 Mermaid 起始内容（空块会停在源码态）', () => {
    const { view } = makeView('hi');

    formatting(view).insertBlock('diagram');
    expect(view.state.doc.toString()).toBe('hi\n\n```mermaid\n' + MERMAID_STARTER + '\n```');
  });
});

// ---------------------------------------------------------------------------
// 接线层：createCodeEditor 的 formatting
// ---------------------------------------------------------------------------

describe('接线层：createCodeEditor 的 formatting', () => {
  it('markdown 语言才有 formatting，其它语言没有；动作经真实内核如实上报', () => {
    const root = document.createElement('div');
    document.body.appendChild(root);
    const reports: string[] = [];
    const editor: CodeEditorInstance = createCodeEditor({
      root,
      defaultValue: 'hello\n',
      language: 'markdown',
      onChange: (value) => reports.push(value),
    });
    opened.push(editor);

    expect(editor.formatting).toBeDefined();
    editor.formatting?.toggleBlock('heading', 2);
    expect(editor.getValue()).toContain('## hello');
    // updateListener 不分来源：程序触发的格式化同样上报（无需打标记）
    expect(reports[reports.length - 1]).toContain('## hello');

    const plainRoot = document.createElement('div');
    document.body.appendChild(plainRoot);
    const plain: CodeEditorInstance = createCodeEditor({
      root: plainRoot,
      defaultValue: 'print(1)\n',
      language: 'python',
      onChange: () => {},
    });
    opened.push(plain);
    // 把 main.py 变成二级标题是个不存在的问题 —— 没有能力就报没有
    expect(plain.formatting).toBeUndefined();
  });
});
