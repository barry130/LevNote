// @vitest-environment jsdom
/**
 * 撤销 / 重做（菜单入口）—— 走真实内核，断言的是序列化出来的 Markdown。
 *
 * ## 这里钉住的是硬约束 7 的又一个变体
 *
 * 菜单点击**不产生** `beforeinput` / `input`，`markdownEmitPlugin` 的上膛窗口
 * 根本不会打开。`undo` / `redo` 必须自己打 `USER_EDIT_META`，否则：
 * 文档内容真的回退了，但 `onChange` 不会上报 → 外壳不变脏 →
 * Ctrl+S 写回**撤销前**的内容，界面还显示「已保存」—— 静默丢数据。
 *
 * 所以每个用例都同时断言「内容回退了」与「上报真的发生了」——
 * 前者能靠别的路径碰巧成立，后者不能（与 formatting.test.ts 同一条规矩）。
 *
 * 编辑的制造走 `formatting.toggleInline`（真实内核动作），不直接塞事务：
 * 塞事务既绕过上报链也绕过 history 插件，测出来的「撤销成功」是同义反复。
 */

import { afterEach, describe, expect, it } from 'vitest';

import { createMarkdownEditor, type MarkdownEditorInstance } from './setup.js';

const FIND_BASE = { caseSensitive: true, wholeWord: false, regex: false } as const;

interface Harness {
  editor: MarkdownEditorInstance;
  /** 内核报给外壳的内容变更。 */
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

function select(editor: MarkdownEditorInstance, needle: string): void {
  const count = editor.search.find({ ...FIND_BASE, query: needle });
  expect(count, `编辑器里找不到「${needle}」`).toBeGreaterThan(0);
  editor.search.reveal(0);
}

describe('撤销 / 重做（菜单入口）', () => {
  it('undo 回退上一次编辑，redo 再做回来', async () => {
    const { editor } = await open('正文 abc 结尾\n');
    select(editor, 'abc');

    editor.formatting.toggleInline('strong');
    expect(editor.getMarkdown()).toContain('**abc**');

    editor.undo();
    expect(editor.getMarkdown()).not.toContain('**abc**');
    expect(editor.getMarkdown()).toContain('正文 abc 结尾');

    editor.redo();
    expect(editor.getMarkdown()).toContain('**abc**');
  });

  it('undo / redo 本身也上报（否则撤销后文档不变脏，保存写回旧内容）', async () => {
    const { editor, reports } = await open('正文 abc 结尾\n');
    select(editor, 'abc');

    editor.formatting.toggleInline('strong');
    const afterEdit = reports.length;

    editor.undo();
    expect(reports.length).toBeGreaterThan(afterEdit);
    expect(reports[reports.length - 1]).not.toContain('**abc**');

    const afterUndo = reports.length;
    editor.redo();
    expect(reports.length).toBeGreaterThan(afterUndo);
    expect(reports[reports.length - 1]).toContain('**abc**');
  });

  it('setMarkdown 的整篇替换不进撤销栈 —— 重新加载后撤销不会把旧内容变回来', async () => {
    const { editor } = await open('正文 abc 结尾\n');
    select(editor, 'abc');
    editor.formatting.toggleInline('strong');

    // 与外壳「重新加载 / 冲突解决」同一条路径：setMarkdown 换掉整篇。
    // 它带 addToHistory:false —— prosemirror-history 对被排除的事务会把
    // 已经不再适用的旧步骤丢弃，否则撤销会「复活」一份早已被替换掉的内容
    editor.setMarkdown('全新的内容\n');
    expect(editor.getMarkdown()).toContain('全新的内容');

    // 撤销不能把旧文档变回来：用户刚看到「重新加载」，再撤销又跳回旧文，
    // 那比「撤销没反应」诡异得多
    editor.undo();
    expect(editor.getMarkdown()).toContain('全新的内容');

    editor.redo();
    expect(editor.getMarkdown()).toContain('全新的内容');
  });
});
