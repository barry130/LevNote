// @vitest-environment jsdom
/**
 * 图纸内核（`.drawio`）的挂载测试（ADR 0048）。
 *
 * 这里测的不是「图画得对不对」（那是 mxGraph 的事，jsdom 也没有布局引擎），
 * 而是**这一层的契约**：
 *
 * - 「现在只能查看」这件事**在面板上写出来了**（硬约束 2：能力缺失要说出来，
 *   不能靠「没有那个按钮」让用户自己发现）；
 * - 渲染器给的失败原因**原样出现在面板上**，一个字都没被吞掉、没被翻译；
 * - 缺 stencil 库要说出是哪些库（图画出来了，但部分形状退化成矩形 ——
 *   不说的话就是静默地画错，ADR 0005 的 V1 实测结论 4）；
 * - 内核**从不调用 `onChange`**：文档不会因为「看一眼图纸」就变脏，
 *   而 `getValue()` 逐字节交回打开时的原文；
 * - 切语言时自己写的那几句话跟着变，而**渲染器的原话保持原样**（ADR 0047）。
 *
 * 与 `modules/markdown/docblock-block.test.ts` 同一条理由要 mock 掉渲染器：
 * 真查看器是 2.6 MB 的上游产物，jsdom 里画不出东西。
 * 能证明的是「我们与渲染器之间的那层」—— 那正是本轮新写的东西。
 */

import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { DrawioRender } from '../../core/drawio-render.js';
import type { EditorHandle, EditorMountProps } from '../../core/editor/types.js';
import { setLocale } from '../../core/i18n/index.js';
import DrawioEditor from './DrawioEditor.js';

declare global {
  var IS_REACT_ACT_ENVIRONMENT: boolean | undefined;
}

/**
 * `vi.hoisted` 是必需的：`vi.mock` 的工厂会被**提升**到文件最顶部执行，
 * 那时普通的模块级变量还处在 TDZ，工厂里碰它们会直接 ReferenceError。
 */
const spy: {
  calls: Array<{ xml: string; attached: boolean }>;
  result: DrawioRender;
} = vi.hoisted(() => ({
  calls: [],
  result: { ok: true, svg: '<svg id="d"><rect/></svg>' },
}));

vi.mock('../../core/drawio-render.js', () => ({
  renderDrawio: (xml: string, container: HTMLElement) => {
    spy.calls.push({ xml, attached: container.isConnected });
    if (spy.result.ok) {
      const host = document.createElement('div');
      host.className = 'mxgraph';
      host.innerHTML = spy.result.svg ?? '';
      container.appendChild(host);
    }
    return Promise.resolve(spy.result);
  },
}));

const XML = [
  '<mxfile><diagram name="页1"><mxGraphModel><root>',
  '<mxCell id="0" /><mxCell id="1" parent="0" />',
  '<mxCell id="2" value="标题" style="rounded=1;html=1;" vertex="1" parent="1">',
  '<mxGeometry x="120" y="120" width="200" height="80" as="geometry" />',
  '</mxCell>',
  '</root></mxGraphModel></diagram></mxfile>',
].join('');

let container: HTMLDivElement;
let root: Root;

beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  spy.calls = [];
  spy.result = { ok: true, svg: '<svg id="d"><rect/></svg>' };
  container = document.createElement('div');
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(async () => {
  setLocale('zh-CN');
  act(() => root.unmount());
  container.remove();
});

interface Mounted {
  handle: EditorHandle | null;
  changes: string[];
}

async function mount(over: Partial<EditorMountProps> = {}): Promise<Mounted> {
  const changes: string[] = [];
  const captured: { handle: EditorHandle | null } = { handle: null };

  const props: EditorMountProps = {
    initialValue: XML,
    path: '图.drawio',
    onChange: (value) => changes.push(value),
    onReady: (handle) => {
      captured.handle = handle;
    },
    ...over,
  };

  await act(async () => {
    root.render(<DrawioEditor {...props} />);
  });
  return { handle: captured.handle, changes };
}

/** 把未决的 Promise（渲染器的 `.then`）排干 —— 用宏任务，与 `definitions.test.ts` 同款。 */
async function settle(): Promise<void> {
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
}

const note = (): string => container.querySelector('.qm-drawio-note')?.textContent ?? '';
const blank = (): string => container.querySelector('.qm-drawio-blank')?.textContent ?? '';

describe('图纸内核 · 「只能查看」这件事写在面板上', () => {
  it('面板顶部一直挂着那条说明，且给出可走的下一步', async () => {
    await mount();
    await settle();

    // 这句话是本轮的核心承诺：不是「没做」，是「做了能看的这一半，另一半的入口在这儿」。
    expect(note()).toContain('只能查看');
    expect(note()).toContain('Ctrl+2');
  });

  it('图画出来了就不占位，画布留着查看器画的东西', async () => {
    await mount();
    await settle();

    expect(spy.calls.map((call) => call.xml)).toEqual([XML]);
    // 宿主**已经接在文档里**才被交出去 —— 否则查看器算不出尺寸，
    // 表现是「打开图纸是空白的，切一下窗口又好了」。
    expect(spy.calls[0]!.attached).toBe(true);
    expect(container.querySelector('.qm-drawio-blank')).toBeNull();
    expect(container.querySelector('.qm-drawio-canvas .mxgraph')).not.toBeNull();
  });
});

describe('图纸内核 · 三种占位各说各的话', () => {
  it('空文件：给的是「怎么让它有内容」，而且根本不去调渲染器', async () => {
    await mount({ initialValue: '   ' });
    await settle();

    // 渲染器会回一句「图表内容是空的」—— 那是一句准确的诊断，
    // 但对着一份还没开始写的文件毫无用处。要给的是下一步。
    expect(blank()).toContain('Ctrl+2');
    expect(spy.calls).toEqual([]);
  });

  it('渲染失败：渲染器的原话一个字都不改、不翻', async () => {
    const message = 'Invalid XML: Parser error on line 1 (不是合法的图纸)';
    spy.result = { ok: false, message };
    await mount();
    await settle();

    expect(container.querySelector('.qm-drawio-blank')?.className).toContain('is-failed');
    expect(blank()).toBe(message);
  });

  it('缺 stencil 库：说出来是哪些库，而不是悄悄把形状画成矩形', async () => {
    // 用一个**没内置**的库名（`bpmn` 与 `flowchart` 是随包内置的，见 `core/drawio.ts`）。
    // 判据是内核自己的 `missingStencilLibs`，这里只负责造一份会缺库的 XML。
    const xml = XML.replace('rounded=1;html=1;', 'shape=mxgraph.network.server;html=1;');
    await mount({ initialValue: xml });
    await settle();

    // 图**画出来了**，但那些形状退化成矩形 —— 渲染器既不报错也不提示。
    // 不说的话就是静默地画错（ADR 0005 实测结论 4）。
    expect(blank()).toContain('network');
    expect(container.querySelector('.qm-drawio-blank')?.className).not.toContain('is-failed');
  });
});

describe('图纸内核 · 与外壳之间的接缝', () => {
  it('内核从不调用 onChange —— 看一眼图纸不会把文档弄脏', async () => {
    const m = await mount();
    await settle();

    // 这个内核不能编辑，所以它**没有**「改了什么」可报。
    // 一旦这里报了，外壳就会把文档标脏、Ctrl+S 就把没变的文件重写一遍。
    expect(m.changes).toEqual([]);
  });

  it('getValue() 逐字节交回打开时的原文', async () => {
    const m = await mount();
    await settle();

    expect(m.handle?.getValue()).toBe(XML);
  });

  it('外壳推内容（重新加载 / 冲突解决）之后：重画，且交回的是推进来的那份', async () => {
    const m = await mount();
    await settle();

    const next = '<mxfile><diagram name="页2"><mxGraphModel><root/></mxGraphModel></diagram></mxfile>';
    await act(async () => {
      m.handle?.setValue(next);
    });
    await settle();

    expect(spy.calls.map((call) => call.xml)).toEqual([XML, next]);
    expect(m.handle?.getValue()).toBe(next);
  });

  it('setValue 把内容从空图纸换成有图纸时，图真的画出来了', async () => {
    // 这条钉的是一个只在空图纸上才复现的死角：canvas 若按「有空就渲染」
    // 条件挂载，那么 xml 从空变成非空的那一刻 effect 重跑，
    // 而那一刻 canvasRef 恰好是 null —— 什么都不画，且不会自己好。
    const m = await mount({ initialValue: '' });
    await settle();
    expect(container.querySelector('.qm-drawio-blank')).not.toBeNull();

    await act(async () => {
      m.handle?.setValue(XML);
    });
    await settle();

    expect(container.querySelector('.qm-drawio-blank')).toBeNull();
    expect(container.querySelector('.qm-drawio-canvas .mxgraph')).not.toBeNull();
  });

  it('toHtml() 返回 null —— 有外链图片的图纸不能塞进自包含导出（ADR 0001）', async () => {
    const m = await mount();
    await settle();

    expect(await m.handle?.toHtml()).toBeNull();
  });
});

describe('图纸内核 · 切语言（ADR 0047）', () => {
  it('自己写的那几句话跟着变，渲染器的原话保持原样', async () => {
    await mount({ initialValue: '' });
    await settle();
    expect(blank()).toContain('这份图纸还是空的');

    setLocale('en');
    await act(async () => {
      root.render(<DrawioEditor initialValue="" path="图.drawio" onChange={() => undefined} onReady={() => undefined} />);
    });
    await settle();

    expect(blank()).toContain('still empty');
    expect(note()).toContain('can only be viewed');
  });

  it('渲染器的失败原因不跟着语言变 —— 它是 mxGraph 说的，不该被我们替它翻译', async () => {
    spy.result = { ok: false, message: 'Parser error on line 3' };
    await mount();
    await settle();

    setLocale('en');
    await act(async () => {
      root.render(<DrawioEditor initialValue={XML} path="图.drawio" onChange={() => undefined} onReady={() => undefined} />);
    });
    await settle();

    expect(blank()).toBe('Parser error on line 3');
  });
});