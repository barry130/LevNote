// @vitest-environment jsdom
/**
 * 代码内核的撤销 / 重做（菜单入口）。
 *
 * 与 Markdown 内核不同，这里**没有**「上膛」机制：CodeMirror 的上报走
 * `updateListener`，任何来源的事务都如实触发 onChange（见 `setup.ts`
 * 的 `undo` 注释）。所以这里的用例钉的是另外两件事：
 *
 * 1. `undo` / `redo` 真的动得了撤销栈（命令与 historyKeymap 共用同一份状态）；
 * 2. **setMarkdown 对应的 setValue 整篇替换不进撤销栈** —— 打开文件后
 *    第一下 Ctrl+Z 不该把整篇文档清空。
 */

import { afterEach, describe, expect, it } from 'vitest';

import { createCodeEditor, type CodeEditorInstance } from './setup.js';

let opened: CodeEditorInstance | null = null;

afterEach(() => {
  opened?.destroy();
  opened = null;
  document.body.innerHTML = '';
});

function open(source: string): { editor: CodeEditorInstance; reports: string[] } {
  const root = document.createElement('div');
  document.body.appendChild(root);
  const reports: string[] = [];
  const editor = createCodeEditor({
    root,
    defaultValue: source,
    language: 'plaintext',
    onChange: (value) => reports.push(value),
  });
  opened = editor;
  return { editor, reports };
}

/** 用查找替换制造一次真实编辑（会进撤销栈，也会上报）。 */
function replaceFirst(editor: CodeEditorInstance, query: string, replacement: string): void {
  const count = editor.search.find({ caseSensitive: true, wholeWord: false, regex: false, query });
  expect(count, `文档里找不到「${query}」`).toBeGreaterThan(0);
  editor.search.replace(0, replacement);
}

describe('代码内核的撤销 / 重做（菜单入口）', () => {
  it('undo 回退上一次编辑，redo 再做回来；onChange 如实上报', () => {
    const { editor, reports } = open('hello world\n');

    replaceFirst(editor, 'world', 'there');
    expect(editor.getValue()).toContain('there');
    expect(reports[reports.length - 1]).toContain('there');

    editor.undo();
    expect(editor.getValue()).not.toContain('there');
    // 上报不分来源：撤销产生的事务同样走 updateListener
    expect(reports[reports.length - 1]).not.toContain('there');

    editor.redo();
    expect(editor.getValue()).toContain('there');
  });

  it('setValue 的整篇替换不进撤销栈 —— 打开文件后第一下撤销不清空文档', () => {
    const { editor, reports } = open('hello\n');
    reports.length = 0;

    // 与外壳「重新加载 / 冲突解决」同一条路径：setValue 换掉整篇
    editor.setValue('全新的内容\n');
    expect(editor.getValue()).toContain('全新的内容');
    // 程序写入不上报（SILENT），这条由 store.test.ts 钉过，这里只是确认前提

    // 撤销栈是空的：整篇替换被挡在外面
    editor.undo();
    expect(editor.getValue()).toContain('全新的内容');
  });

  it('撤销到头再撤是空操作，redo 亦然 —— 不抛错', () => {
    const { editor } = open('hello\n');

    expect(() => editor.undo()).not.toThrow();
    expect(editor.getValue()).toContain('hello');

    expect(() => editor.redo()).not.toThrow();
    expect(editor.getValue()).toContain('hello');
  });
});
