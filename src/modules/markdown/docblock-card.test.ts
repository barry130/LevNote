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

import type { DocBlockPreview } from '../../core/editor/types.js';
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
  onDocBlockEditInPlace?: (ref: { src: string; kind: string; title: string | null }) => void;
  resolveDocBlockPreview?: (ref: { src: string; kind: string }) => Promise<DocBlockPreview | null>;
}

async function open(source: string, options: OpenOptions = {}): Promise<{ editor: MarkdownEditorInstance; dom: HTMLElement }> {
  // 一个用例里开两台编辑器时（对比「没挂能力 / 挂了能力」的卡片形态），
  // 上一台先销毁 —— 不留挂在同一个 document 上的旧实例
  if (opened !== null) {
    await opened.destroy();
    opened = null;
  }

  const root = document.createElement('div');
  document.body.appendChild(root);
  const editor = await createMarkdownEditor({
    root,
    defaultValue: LEAD + source,
    onChange: () => undefined,
    onDocBlockOpen: options.onDocBlockOpen,
    onDocBlockEditInPlace: options.onDocBlockEditInPlace,
    resolveDocBlockPreview: options.resolveDocBlockPreview,
  });
  opened = editor;

  const dom = root.querySelector('.ProseMirror');
  if (!(dom instanceof HTMLElement)) throw new Error('没有找到 ProseMirror 视图');
  return { editor, dom };
}

function card(dom: HTMLElement): HTMLElement | null {
  return dom.querySelector<HTMLElement>('.qm-docref-card');
}

/** 卡片主区（点 = 打开被引用文档）。 */
function cardMain(dom: HTMLElement): HTMLButtonElement | null {
  return dom.querySelector<HTMLButtonElement>('.qm-docref-main');
}

/** 动作按钮，按文字找。 */
function cardAction(dom: HTMLElement, label: string): HTMLButtonElement | null {
  const buttons = Array.from(dom.querySelectorAll<HTMLButtonElement>('.qm-docref-action'));
  return buttons.find((button) => button.textContent === label) ?? null;
}

/** 展开的摘要区（null = 没展开）。 */
function previewOf(dom: HTMLElement): HTMLElement | null {
  return dom.querySelector<HTMLElement>('.qm-docref-preview');
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

    cardMain(dom)!.click();
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

  it('「就地编辑」按钮只在能力挂着时画，点击交出解析好的属性（ADR 0036）', async () => {
    const onDocBlockEditInPlace = vi.fn();
    const source = '```doc-block\nid: a\nkind: markdown\nmode: reference\nsrc: notes/x.md\ntitle: 部署手册\n---\n```\n';

    // 能力没挂：不画按钮 —— 点了没反应的按钮比没有更糟
    const without = await open(source);
    expect(cardAction(without.dom, '就地编辑')).toBeNull();

    const withCapability = await open(source, { onDocBlockEditInPlace });
    const button = cardAction(withCapability.dom, '就地编辑');
    expect(button).not.toBeNull();
    button!.click();
    expect(onDocBlockEditInPlace).toHaveBeenCalledTimes(1);
    expect(onDocBlockEditInPlace).toHaveBeenCalledWith({ src: 'notes/x.md', kind: 'markdown', title: '部署手册' });
  });

  it('「预览」展开：等待期间先有话，数据到了换摘要；再点收起（ADR 0036）', async () => {
    const resolveDocBlockPreview = vi.fn(async (): Promise<DocBlockPreview> => ({
      kind: 'markdown',
      wordCount: 128,
      headings: [
        { level: 1, text: '部署手册' },
        { level: 2, text: '前置要求' },
      ],
    }));
    const { dom } = await open(
      '```doc-block\nid: a\nkind: markdown\nmode: reference\nsrc: notes/x.md\n---\n```\n',
      { resolveDocBlockPreview },
    );

    expect(previewOf(dom)).toBeNull();
    cardAction(dom, '预览')!.click();
    // 异步摘要还没回来就要有东西看 —— 点了立刻有反应
    expect(previewOf(dom)?.textContent).toContain('正在读');
    await vi.waitFor(() => {
      expect(previewOf(dom)?.textContent).toContain('128');
      expect(previewOf(dom)?.textContent).toContain('部署手册');
      expect(previewOf(dom)?.textContent).toContain('前置要求');
    });
    expect(resolveDocBlockPreview).toHaveBeenCalledWith({ src: 'notes/x.md', kind: 'markdown' });

    cardAction(dom, '预览')!.click();
    expect(previewOf(dom)).toBeNull();
  });

  it('摘要读不到时把原因画出来 —— 失败是数据，不是静默', async () => {
    const resolveDocBlockPreview = vi.fn(async (): Promise<DocBlockPreview> => ({
      kind: 'error',
      reason: '读不到 notes/x.md',
    }));
    const { dom } = await open('```doc-block\nid: a\nkind: text\nmode: reference\nsrc: notes/x.md\n---\n```\n', {
      resolveDocBlockPreview,
    });

    cardAction(dom, '预览')!.click();
    await vi.waitFor(() => {
      expect(previewOf(dom)?.classList.contains('is-failed')).toBe(true);
      expect(previewOf(dom)?.textContent).toContain('读不到 notes/x.md');
    });
  });

  it('text 摘要画成片段，mindmap 摘要画根主题与节点数', async () => {
    let call = 0;
    const resolveDocBlockPreview = vi.fn(async (): Promise<DocBlockPreview> => {
      call += 1;
      return call === 1
        ? { kind: 'text', lineCount: 3, snippet: '第一行\n第二行\n第三行' }
        : { kind: 'mindmap', nodeCount: 5, rootTitle: '根' };
    });
    const { dom } = await open('```doc-block\nid: a\nkind: text\nmode: reference\nsrc: a.txt\n---\n```\n', {
      resolveDocBlockPreview,
    });

    cardAction(dom, '预览')!.click();
    await vi.waitFor(() => {
      expect(previewOf(dom)?.querySelector('pre')?.textContent).toBe('第一行\n第二行\n第三行');
      expect(previewOf(dom)?.textContent).toContain('共 3 行');
    });

    // 收起再展开走的是同一条开关路径；第二次摘要是另一份数据（内核按当前数据画）
    cardAction(dom, '预览')!.click();
    expect(previewOf(dom)).toBeNull();
    cardAction(dom, '预览')!.click();
    await vi.waitFor(() => {
      expect(previewOf(dom)?.textContent).toContain('5 个节点');
      expect(previewOf(dom)?.textContent).toContain('根');
    });
  });
});
