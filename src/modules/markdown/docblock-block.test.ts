// @vitest-environment jsdom
/**
 * `doc-block` 图表引用在**编辑器**里的行为（P5 第 25 项）。
 *
 * ## 为什么这里要 mock 掉渲染器
 *
 * 与 `mermaid-block.test.ts` 同一条理由 —— 真查看器是 2.6 MB 的上游产物，
 * jsdom 里画不出东西。单测能证明的是**这一层的契约**：
 * 引用的文件怎么读进来、读不到时失败原因有没有被说出来、
 * inline 载荷是不是原样交给渲染器、缺 stencil 库的警示有没有出现、
 * 「放大看」交出去的源是不是**渲染用的那份 XML**（引用模式下它与块文本不同）。
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { DiagramView } from '../../core/editor/types.js';
import type { DrawioRender } from '../../core/drawio-render.js';
import { typeInto } from '../../test/typing.js';
import { createMarkdownEditor, type MarkdownEditorInstance } from './setup.js';

/**
 * `vi.hoisted` 是必需的：`vi.mock` 的工厂会被**提升**到文件最顶部执行，
 * 那时普通的模块级变量还处在 TDZ，工厂里碰它们会直接 ReferenceError。
 */
const spy = vi.hoisted(() => {
  const state: {
    calls: Array<{ xml: string; attached: boolean }>;
    /** 下一次渲染的结果；ok 时用 `svg` 补进容器（模拟查看器把图画进宿主）。 */
    result: DrawioRender;
  } = {
    calls: [],
    result: { ok: true, svg: '<svg id="d"><rect/></svg>' },
  };
  return state;
});

vi.mock('../../core/drawio-render.js', () => ({
  renderDrawio: (xml: string, container: HTMLElement) => {
    spy.calls.push({ xml, attached: container.isConnected });
    if (spy.result.ok) {
      const host = document.createElement('div');
      host.className = 'mxgraph';
      host.setAttribute('data-mxgraph', JSON.stringify({ xml }));
      host.innerHTML = spy.result.svg ?? '';
      container.appendChild(host);
    }
    return Promise.resolve(spy.result);
  },
}));

let opened: MarkdownEditorInstance | null = null;

beforeEach(() => {
  spy.calls = [];
  spy.result = { ok: true, svg: '<svg id="d"><rect/></svg>' };
});

afterEach(async () => {
  await opened?.destroy();
  opened = null;
  document.body.innerHTML = '';
});

const JS_BLOCK = '```js\nlet a = 1;\n```\n';

/** 引用一个不存在的图文件（`resolveDocBlock` 会返回 null）。 */
const BROKEN_SRC = '```doc-block\nkind: diagram\nsrc: missing.drawio\n---\n```\n';

/** 文档开头必须先有一段正文 —— 理由见 `mermaid-block.test.ts` 的 `LEAD`。 */
const LEAD = '开头一段。\n\n';

interface OpenOptions {
  resolveDocBlock?: (src: string) => Promise<string | null>;
  onDiagramOpen?: (diagram: DiagramView) => void;
}

async function open(
  source: string,
  options: OpenOptions = {},
): Promise<{ editor: MarkdownEditorInstance; dom: HTMLElement }> {
  const root = document.createElement('div');
  document.body.appendChild(root);
  const editor = await createMarkdownEditor({
    root,
    defaultValue: LEAD + source,
    onChange: () => undefined,
    resolveDocBlock: options.resolveDocBlock,
    onDiagramOpen: options.onDiagramOpen,
  });
  opened = editor;

  const dom = root.querySelector('.ProseMirror');
  if (!(dom instanceof HTMLElement)) throw new Error('没有找到 ProseMirror 视图');
  return { editor, dom };
}

async function flush(): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, 0));
}

function query<T extends Element>(dom: HTMLElement, selector: string): T {
  const found = dom.querySelector<T>(selector);
  if (found === null) throw new Error(`找不到 ${selector}`);
  return found;
}

function mode(dom: HTMLElement): string {
  return query<HTMLElement>(dom, '.qm-cb').dataset['mode'] ?? '';
}

function diagramText(dom: HTMLElement): string {
  return query<HTMLElement>(dom, '.qm-cb-diagram').textContent ?? '';
}

function lastCall(): { xml: string; attached: boolean } | undefined {
  return spy.calls[spy.calls.length - 1];
}

/** 开头那段正文的落点（把光标挪出代码块用，与 mermaid 测试的 `codeCaret` 相对）。 */
function leadCaret(dom: HTMLElement): Text {
  const paragraph = query<HTMLParagraphElement>(dom, 'p');
  const walker = document.createTreeWalker(paragraph, NodeFilter.SHOW_TEXT);
  let last: Text | null = null;
  for (let node = walker.nextNode(); node !== null; node = walker.nextNode()) {
    last = node as Text;
  }
  if (last !== null) return last;
  const created = document.createTextNode('');
  paragraph.appendChild(created);
  return created;
}

// ---------------------------------------------------------------------------

describe('doc-block · 引用模式（src）', () => {
  it('声明的图文件读到了就画：交给渲染器的是**文件原文**，不是块文本', async () => {
    const { dom } = await open(BROKEN_SRC.replace('missing.drawio', 'fig.drawio'), {
      resolveDocBlock: async (src) => (src === 'fig.drawio' ? '<mxfile>图的内容</mxfile>' : null),
    });
    await flush();

    expect(mode(dom)).toBe('diagram');
    expect(spy.calls).toHaveLength(1);
    expect(lastCall()?.xml).toBe('<mxfile>图的内容</mxfile>');
    // 查看器宿主挂在图区里，图区此刻可见
    expect(query<HTMLElement>(dom, '.qm-cb-drawio')).not.toBeNull();
  });

  it('读不到引用的文件：失败占着图区、原因说得出来（绝不留白）', async () => {
    const { dom } = await open(BROKEN_SRC, { resolveDocBlock: async () => null });
    await flush();

    expect(mode(dom)).toBe('diagram');
    expect(spy.calls).toHaveLength(0);
    expect(diagramText(dom)).toContain('读不到引用的文件：missing.drawio');
    expect(query<HTMLElement>(dom, '.qm-cb-diagram').classList.contains('is-failed')).toBe(true);
  });

  it('外壳没实现 resolveDocBlock：同样是「读不到」，不抛错', async () => {
    const { dom } = await open(BROKEN_SRC);
    await flush();

    expect(mode(dom)).toBe('diagram');
    expect(diagramText(dom)).toContain('读不到引用的文件：missing.drawio');
  });

  it('没有 kind: diagram 的 doc-block 就是普通代码块（显示源码），也不去读文件', async () => {
    const resolver = vi.fn(async () => '<mxfile/>');
    const { dom } = await open('```doc-block\nkind: code\nsrc: a.js\n---\n```\n', {
      resolveDocBlock: resolver,
    });
    await flush();

    expect(mode(dom)).toBe('code');
    expect(resolver).not.toHaveBeenCalled();
    expect(spy.calls).toHaveLength(0);
  });

  it('声明了 kind 却既没有 src 也不是 inline：把怎么改写出来', async () => {
    const { dom } = await open('```doc-block\nkind: diagram\n---\n```\n', {
      resolveDocBlock: async () => '<mxfile/>',
    });
    await flush();

    expect(mode(dom)).toBe('diagram');
    expect(diagramText(dom)).toContain('没有 src 引用，也不是 inline 载荷');
  });

  it('属性区没有结束行：告诉用户那一行「---」不能省', async () => {
    const { dom } = await open('```doc-block\nkind: diagram\nsrc: fig.drawio\n```\n', {
      resolveDocBlock: async () => '<mxfile/>',
    });
    await flush();

    expect(mode(dom)).toBe('diagram');
    expect(spy.calls).toHaveLength(0);
    expect(diagramText(dom)).toContain('缺少结束行');
  });

  it('引用模式下「放大看」交出去的源是**渲染用的那份 XML**', async () => {
    const seen: DiagramView[] = [];
    const { dom } = await open(BROKEN_SRC.replace('missing.drawio', 'fig.drawio'), {
      resolveDocBlock: async () => '<mxfile>图的内容</mxfile>',
      onDiagramOpen: (diagram) => seen.push(diagram),
    });
    await flush();

    const zoom = query<HTMLButtonElement>(dom, '.qm-cb-zoom');
    expect(zoom.hidden).toBe(false);
    zoom.click();

    expect(seen).toHaveLength(1);
    expect(seen[0]?.svg).toContain('<rect');
    // 块文本只有属性区，图的 XML 在引用的文件里 —— 两者不是一回事
    expect(seen[0]?.source).toBe('<mxfile>图的内容</mxfile>');
  });

  it('引用了未内置的形状库：图画出来，但缺库这件事必须说出来', async () => {
    // `missingStencilLibs` 是**真的**在跑（没 mock）：aws4 不在内置清单里
    const { dom } = await open(BROKEN_SRC.replace('missing.drawio', 'fig.drawio'), {
      resolveDocBlock: async () =>
        '<mxGraphModel><root><mxCell style="shape=mxgraph.aws4.lambda_function;"/></root></mxGraphModel>',
    });
    await flush();

    expect(mode(dom)).toBe('diagram');
    expect(query<HTMLElement>(dom, '.qm-cb-diagram').classList.contains('is-ok')).toBe(true);
    expect(diagramText(dom)).toContain('未内置的形状库：aws4');
    expect(query<HTMLElement>(dom, '.qm-cb-note-warn')).not.toBeNull();
  });
});

describe('doc-block · inline 模式', () => {
  it('载荷就是图本身：原样交给渲染器，不需要读任何文件', async () => {
    const resolver = vi.fn(async () => {
      throw new Error('inline 模式不该去读文件');
    });
    const { dom } = await open(
      '```doc-block\nkind: diagram\nmode: inline\n---\n<mxfile>内联的图</mxfile>\n```\n',
      { resolveDocBlock: resolver },
    );
    await flush();

    expect(mode(dom)).toBe('diagram');
    expect(resolver).not.toHaveBeenCalled();
    expect(lastCall()?.xml).toBe('<mxfile>内联的图</mxfile>');
  });

  it('inline 载荷是空的：说清楚该把图写在哪', async () => {
    const { dom } = await open('```doc-block\nkind: diagram\nmode: inline\n---\n```\n');
    await flush();

    expect(mode(dom)).toBe('diagram');
    expect(diagramText(dom)).toContain('inline 载荷是空的');
  });
});

describe('斜杠菜单 / 段落菜单插入 draw.io 图表', () => {
  it('插进来就是一段 inline 的 doc-block，起始内容直接可渲染', async () => {
    const { editor, dom } = await open('');
    editor.formatting.insertBlock('drawio');
    await flush();

    const markdown = editor.getMarkdown();
    expect(markdown).toContain('```doc-block');
    expect(markdown).toContain('kind: diagram');
    expect(markdown).toContain('mode: inline');
    expect(markdown).toContain('<mxfile>');
    // 光标落进块内源码 —— 他要改的就是这段 XML
    expect(mode(dom)).toBe('code');
  });

  it('光标离开块之后图就画出来了：交给渲染器的是载荷 XML', async () => {
    const { editor, dom } = await open('');
    editor.formatting.insertBlock('drawio');
    await flush();
    // 还在块里 = 正在编辑源码，不渲染
    expect(spy.calls).toHaveLength(0);

    await typeInto(dom, '。', leadCaret);
    await flush();

    expect(spy.calls).toHaveLength(1);
    expect(mode(dom)).toBe('diagram');
    expect(lastCall()?.xml).toContain('<mxfile>');
    // 起始内容不引用 stencil 库 —— 缺库警示不该出现
    expect(diagramText(dom)).not.toContain('未内置的形状库');
  });
});

describe('doc-block · 与编辑的互动', () => {
  it('引用换了个文件：光标离开块之后才按**新内容**重画', async () => {
    const { editor, dom } = await open(BROKEN_SRC.replace('missing.drawio', 'fig.drawio'), {
      resolveDocBlock: async (src) =>
        src === 'fig.drawio' ? '<mxfile>第一版</mxfile>' : '<mxfile>第二版</mxfile>',
    });
    await flush();
    expect(spy.calls).toHaveLength(1);

    // 引用换成另一个文件（走「文档被替换」这条真实路径）。
    // setMarkdown 会把旧光标（在替换范围内）映射到新块的**里面** ——
    // 那正是「正在看源码」的状态，不重画是对的；在正文里敲一个字把光标挪出去
    editor.setMarkdown(LEAD + '```doc-block\nkind: diagram\nsrc: fig2.drawio\n---\n```\n');
    await flush();
    expect(spy.calls).toHaveLength(1);

    await typeInto(dom, '。', leadCaret);
    await flush();

    expect(spy.calls).toHaveLength(2);
    expect(lastCall()?.xml).toBe('<mxfile>第二版</mxfile>');
  });

  it('渲染失败（查看器加载不出来）：原因占着图区', async () => {
    spy.result = { ok: false, message: '查看器脚本加载失败' };
    const { dom } = await open(BROKEN_SRC.replace('missing.drawio', 'fig.drawio'), {
      resolveDocBlock: async () => '<mxfile/>',
    });
    await flush();

    expect(mode(dom)).toBe('diagram');
    expect(diagramText(dom)).toContain('查看器脚本加载失败');
    // 失败时没有图，放大按钮不该出现 —— 不做点了没反应的按钮
    expect(query<HTMLButtonElement>(dom, '.qm-cb-zoom').hidden).toBe(true);
  });

  it('别的语言的块永远不触发这条链路', async () => {
    const { dom } = await open(JS_BLOCK);
    await flush();

    expect(mode(dom)).toBe('code');
    expect(spy.calls).toHaveLength(0);
  });
});
