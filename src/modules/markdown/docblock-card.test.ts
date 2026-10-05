// @vitest-environment jsdom
/**
 * `doc-block` **引用卡片**在编辑器里的行为（P6 第 28 项）。
 *
 * 卡片是同步画的（不需要读文件、不需要渲染器），所以这里没有 mock 对象 ——
 * 单测钉住的是**展示层契约**：哪些 kind / mode 显示成卡片、哪些照旧显示为
 * 代码、属性区有问题时说明原因而不是装作正常、点击卡片交出的是
 * 解析好的属性（外壳不需要再解析一遍）。
 */

import { afterEach, describe, expect, it, vi } from 'vitest';

import { createMarkdownEditor, type MarkdownEditorInstance } from './setup.js';

let opened: MarkdownEditorInstance | null = null;

afterEach(async () => {
  await opened?.destroy();
  opened = null;
  document.body.innerHTML = '';
});

/** 文档开头必须先有一段正文 —— 光标要有地方待（不在块里，块才会画预览）。 */
const LEAD = '开头一段。\n\n';

interface OpenOptions {
  onDocBlockOpen?: (ref: { src: string; kind: string; title: string | null }) => void;
}

async function open(source: string, options: OpenOptions = {}): Promise<{ editor: MarkdownEditorInstance; dom: HTMLElement }> {
  const root = document.createElement('div');
  document.body.appendChild(root);
  const editor = await createMarkdownEditor({
    root,
    defaultValue: LEAD + source,
    onChange: () => undefined,
    onDocBlockOpen: options.onDocBlockOpen,
  });
  opened = editor;

  const dom = root.querySelector('.ProseMirror');
  if (!(dom instanceof HTMLElement)) throw new Error('没有找到 ProseMirror 视图');
  return { editor, dom };
}

function card(dom: HTMLElement): HTMLButtonElement | null {
  return dom.querySelector<HTMLButtonElement>('.qm-docref-card');
}

function failureNote(dom: HTMLElement): string | null {
  const diagram = dom.querySelector<HTMLElement>('.qm-cb-diagram.is-failed');
  return diagram?.textContent ?? null;
}

describe('doc-block 引用卡片（P6 第 28 项）', () => {
  it('markdown / mindmap / text 的 reference 引用显示为卡片，点击交出解析好的属性', async () => {
    const onDocBlockOpen = vi.fn();
    const { dom } = await open(
      '```doc-block\nid: a\nkind: markdown\nmode: reference\nsrc: notes/x.md\ntitle: 部署手册\n---\n```\n',
      { onDocBlockOpen },
    );

    const button = card(dom);
    expect(button).not.toBeNull();
    expect(button!.textContent).toContain('部署手册');
    expect(button!.textContent).toContain('notes/x.md');
    expect(button!.textContent).toContain('markdown');

    button!.click();
    expect(onDocBlockOpen).toHaveBeenCalledTimes(1);
    expect(onDocBlockOpen).toHaveBeenCalledWith({ src: 'notes/x.md', kind: 'markdown', title: '部署手册' });
  });

  it('没有标题时卡片显示 src —— 卡片上不能出现空白', async () => {
    const { dom } = await open('```doc-block\nid: a\nkind: mindmap\nmode: reference\nsrc: map.mindmap\n---\n```\n');
    const button = card(dom);
    expect(button).not.toBeNull();
    expect(button!.textContent).toContain('map.mindmap');
    expect(button!.textContent).not.toContain('null');
    expect(button!.textContent).not.toContain('undefined');
  });

  it('inline 模式不画卡片 —— 内容就在块里，照旧显示为代码', async () => {
    const { dom } = await open('```doc-block\nid: a\nkind: markdown\nmode: inline\n---\n正文就在这里\n```\n');
    expect(card(dom)).toBeNull();
    expect(dom.querySelector('.qm-cb-note')).toBeNull();
  });

  it('kind: code 的引用不画卡片 —— 「这段代码在这里」应该原样呈现', async () => {
    const { dom } = await open('```doc-block\nid: a\nkind: code\nmode: reference\nsrc: s.py\n---\n```\n');
    expect(card(dom)).toBeNull();
  });

  it('reference 缺少 src 时如实说明 —— 点开了也是空白不如当场说清楚', async () => {
    const { dom } = await open('```doc-block\nid: a\nkind: markdown\nmode: reference\n---\n```\n');
    expect(card(dom)).toBeNull();
    expect(failureNote(dom)).toContain('src');
  });

  it('属性区没有结束行时如实说明', async () => {
    // 故意不写「---」：解析器会报 MISSING_TERMINATOR
    const { dom } = await open('```doc-block\nid: a\nkind: markdown\nmode: reference\nsrc: x.md\n```\n');
    expect(card(dom)).toBeNull();
    expect(failureNote(dom)).toContain('---');
  });

  it('不认识的 kind 照旧显示原文 —— 展示层不替用户猜它是什么', async () => {
    const { dom } = await open('```doc-block\nid: a\nkind: spreadsheet\nmode: reference\nsrc: t.csv\n---\n```\n');
    expect(card(dom)).toBeNull();
  });
});
