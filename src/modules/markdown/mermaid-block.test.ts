// @vitest-environment jsdom
/**
 * Mermaid 代码块在**编辑器**里的行为：图 / 源码怎么切。
 *
 * ## 为什么这里要 mock 掉渲染器
 *
 * `renderMermaid` 在 jsdom 里画不出东西 —— mermaid 靠 `getBBox()` 量文字尺寸，
 * 而 jsdom 没有布局引擎，一律返回 0。真去加载它既慢（12 MB 的模块图）又只会
 * 得到一张塌掉的图，什么也证明不了。
 *
 * 单测能证明的是**这一层的契约**：图/源码什么时候切、失败有没有被藏起来、
 * SVG 有没有被读进文档、主题有没有传给渲染器。
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { DiagramView } from '../../core/editor/types.js';
import type { MermaidRender } from '../../core/mermaid-render.js';
import { typeInto } from '../../test/typing.js';
import { createMarkdownEditor, type MarkdownEditorInstance } from './setup.js';

/**
 * `vi.hoisted` 是必需的：`vi.mock` 的工厂会被**提升**到文件最顶部执行，
 * 那时普通的模块级变量还处在 TDZ，工厂里碰它们会直接 ReferenceError。
 */
const spy = vi.hoisted(() => {
  const state: {
    calls: Array<{ source: string; theme: string }>;
    result: MermaidRender;
  } = {
    calls: [],
    result: { ok: true, svg: '<svg id="g"><rect/></svg>' },
  };
  return state;
});

vi.mock('../../core/mermaid-render.js', () => ({
  renderMermaid: (source: string, theme: string) => {
    spy.calls.push({ source, theme });
    return Promise.resolve(spy.result);
  },
}));

let opened: MarkdownEditorInstance | null = null;

beforeEach(() => {
  spy.calls = [];
  spy.result = { ok: true, svg: '<svg id="g"><rect/></svg>' };
  document.documentElement.dataset['theme'] = 'light';
});

afterEach(async () => {
  await opened?.destroy();
  opened = null;
  document.body.innerHTML = '';
});

const MERMAID_BLOCK = '```mermaid\ngraph LR\n  A-->B\n```\n';
const JS_BLOCK = '```js\nlet a = 1;\n```\n';

/**
 * 文档开头**必须**先有一段正文。
 *
 * 编辑器的初始光标落在文档最开头，而那个位置对「第一个块就是代码块」的文档
 * 来说**是在代码块里面**（`doc` 的第 0 位不是合法文本位置，第一个合法位置是
 * 块内的第 1 位）。于是块会老老实实地停在源码态 —— 那是对的，
 * 但这些用例要看的是「光标不在里面」时的样子，所以先垫一段正文把光标顶出去。
 */
const LEAD = '开头一段。\n\n';

async function open(
  source: string,
  options: { onDiagramOpen?: (diagram: DiagramView) => void } = {},
): Promise<{ editor: MarkdownEditorInstance; dom: HTMLElement }> {
  const root = document.createElement('div');
  document.body.appendChild(root);
  const editor = await createMarkdownEditor({
    root,
    defaultValue: LEAD + source,
    onChange: () => undefined,
    onDiagramOpen: options.onDiagramOpen,
  });
  opened = editor;

  const dom = root.querySelector('.ProseMirror');
  if (!(dom instanceof HTMLElement)) throw new Error('没有找到 ProseMirror 视图');
  return { editor, dom };
}

/**
 * 把挂起的微任务跑完。
 *
 * 渲染是一次 `await`（即使 mock 出来的 Promise 已经 resolve），
 * 所以断言之前必须让出一次宏任务。
 */
async function flush(): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, 0));
}

function query<T extends Element>(dom: HTMLElement, selector: string): T {
  const found = dom.querySelector<T>(selector);
  if (found === null) throw new Error(`找不到 ${selector}`);
  return found;
}

/** 最后一次发起的渲染（`Array.prototype.at` 不在 es2020 的 lib 里）。 */
function lastCall(): { source: string; theme: string } | undefined {
  return spy.calls[spy.calls.length - 1];
}

/** 代码块根元素上的 `data-mode`。 */
function mode(dom: HTMLElement): string {
  return query<HTMLElement>(dom, '.qm-cb').dataset['mode'] ?? '';
}

function diagramText(dom: HTMLElement): string {
  return query<HTMLElement>(dom, '.qm-cb-diagram').textContent ?? '';
}

/** 代码块末尾的落点（与 `code-block.test.ts` 里那个同源）。 */
function codeCaret(dom: HTMLElement): Text {
  const root = query(dom, '.qm-cb-code');
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

/** 在代码块里点一下图区。 */
function clickDiagram(dom: HTMLElement): void {
  query<HTMLElement>(dom, '.qm-cb-diagram').dispatchEvent(
    new MouseEvent('mousedown', { bubbles: true, cancelable: true }),
  );
}

// ---------------------------------------------------------------------------

describe('Mermaid 代码块的图 / 源码双态', () => {
  it('画得出来时就显示图，不显示源码', async () => {
    const { dom } = await open(MERMAID_BLOCK);
    await flush();

    expect(mode(dom)).toBe('diagram');
    expect(query<HTMLElement>(dom, '.qm-cb-diagram').innerHTML).toContain('<svg');
  });

  it('不是 mermaid 的块永远停在源码态，也不去渲染', async () => {
    const { dom } = await open(JS_BLOCK);
    await flush();

    expect(mode(dom)).toBe('code');
    expect(spy.calls).toHaveLength(0);
  });

  it('空图表停在源码态 —— 否则用户看不到该往哪儿打字', async () => {
    const { dom } = await open('```mermaid\n```\n');
    await flush();

    expect(mode(dom)).toBe('code');
    expect(spy.calls).toHaveLength(0);
  });

  it('渲染失败时仍然占着图区：原因必须看得见', async () => {
    spy.result = { ok: false, message: 'No diagram type detected' };
    const { dom } = await open(MERMAID_BLOCK);
    await flush();

    expect(mode(dom)).toBe('diagram');
    expect(diagramText(dom)).toContain('No diagram type detected');
    // 还得告诉他怎么回去改
    expect(diagramText(dom)).toContain('点一下改源码');
  });

  it('光标进到块里就切回源码', async () => {
    const { dom } = await open(MERMAID_BLOCK);
    await flush();
    expect(mode(dom)).toBe('diagram');

    await typeInto(dom, 'x', codeCaret);

    expect(mode(dom)).toBe('code');
  });

  it('点一下图就回到源码 —— 并且不把文档弄脏', async () => {
    const { dom } = await open(MERMAID_BLOCK);
    await flush();
    expect(mode(dom)).toBe('diagram');

    clickDiagram(dom);

    expect(mode(dom)).toBe('code');
  });

  it('图里的 SVG 不会被读进文档', async () => {
    const { editor, dom } = await open(MERMAID_BLOCK);
    await flush();

    expect(query<HTMLElement>(dom, '.qm-cb-diagram').innerHTML).toContain('<svg');
    expect(editor.getMarkdown()).not.toContain('svg');
    expect(editor.getMarkdown()).toContain('graph LR');
  });

  it('主题传给渲染器', async () => {
    document.documentElement.dataset['theme'] = 'dark';
    const { dom } = await open(MERMAID_BLOCK);
    await flush();

    expect(mode(dom)).toBe('diagram');
    expect(lastCall()?.theme).toBe('dark');
  });

  it('换主题要重画 —— 图的配色是渲染时定死的', async () => {
    await open(MERMAID_BLOCK);
    await flush();
    expect(spy.calls).toHaveLength(1);

    document.documentElement.dataset['theme'] = 'dark';
    // MutationObserver 的回调是微任务，渲染又要一次 await
    await flush();
    await flush();

    expect(spy.calls).toHaveLength(2);
    expect(lastCall()?.theme).toBe('dark');
  });

  it('在块里打字不渲染：每敲一个键重排整张图纯属浪费', async () => {
    const { dom } = await open(MERMAID_BLOCK);
    await flush();
    expect(spy.calls).toHaveLength(1);

    await typeInto(dom, '  B-->C', codeCaret);
    await flush();

    expect(spy.calls).toHaveLength(1);
  });

  it('光标离开块之后才补画，画的是**最新**的源码', async () => {
    const { dom } = await open(MERMAID_BLOCK);
    await flush();

    await typeInto(dom, '  B-->C', codeCaret);
    // 往正文里打一个字 = 把光标挪出代码块
    await typeInto(dom, '。');
    await flush();

    expect(spy.calls).toHaveLength(2);
    expect(lastCall()?.source).toContain('B-->C');
  });

  it('放大按钮只在真的画出图时出现 —— 不做假按钮', async () => {
    spy.result = { ok: false, message: '画不出来' };
    const { dom } = await open(MERMAID_BLOCK);
    await flush();

    expect(query<HTMLButtonElement>(dom, '.qm-cb-zoom').hidden).toBe(true);
  });

  it('点放大按钮：把**此刻屏幕上的那一份**交给外壳，并带上源码', async () => {
    spy.result = { ok: true, svg: '<svg id="real"><circle/></svg>' };
    const seen: DiagramView[] = [];
    const { dom } = await open(MERMAID_BLOCK, { onDiagramOpen: (diagram) => seen.push(diagram) });
    await flush();

    const zoom = query<HTMLButtonElement>(dom, '.qm-cb-zoom');
    expect(zoom.hidden).toBe(false);
    zoom.click();

    expect(seen).toHaveLength(1);
    expect(seen[0]?.svg).toContain('<circle');
    expect(seen[0]?.source).toContain('graph LR');
  });

  it('不实现 onDiagramOpen 时点放大按钮什么都不发生（也不报错）', async () => {
    const { dom } = await open(MERMAID_BLOCK);
    await flush();

    expect(() => query<HTMLButtonElement>(dom, '.qm-cb-zoom').click()).not.toThrow();
  });

  it('语言从 mermaid 改成别的：图区清空并回到源码态', async () => {
    const { editor, dom } = await open(MERMAID_BLOCK);
    await flush();
    expect(mode(dom)).toBe('diagram');

    // 整篇换成别的语言的块（走的是「文档被替换」这条真实路径）
    editor.setMarkdown(JS_BLOCK);
    await flush();

    expect(mode(dom)).toBe('code');
    expect(query<HTMLElement>(dom, '.qm-cb-diagram').innerHTML).toBe('');
  });
});

describe('斜杠菜单插入图表', () => {
  it('插进来的是**带起始内容**的 mermaid 块，不是空块', async () => {
    const { editor, dom } = await open('');

    editor.formatting.insertBlock('diagram');
    await flush();

    expect(editor.getMarkdown()).toContain('```mermaid');
    // 空块会停在源码态、用户不知道该写什么，所以必须带一段能直接画出来的例子
    expect(editor.getMarkdown()).toContain('A[开始]-->B[结束]');
    // 光标落在块**里面**：他要改的就是这段源码
    expect(mode(dom)).toBe('code');
    // 光标在块里 = 还没到要画图的时候
    expect(spy.calls).toHaveLength(0);
  });

  it('光标离开之后图就画出来了', async () => {
    const { editor, dom } = await open('');

    editor.formatting.insertBlock('diagram');
    await typeInto(dom, '。');
    await flush();

    expect(spy.calls).toHaveLength(1);
    expect(mode(dom)).toBe('diagram');
  });
});
