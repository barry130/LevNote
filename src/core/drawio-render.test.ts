// @vitest-environment jsdom
/**
 * `renderDrawio` 的契约：**这一层**做对的事，不是 mxGraph 画得好不好。
 *
 * 与 `mermaid-render.test.ts` 同一条纪律 —— 真查看器是 2.6 MB 的上游产物，
 * jsdom 里画不出也画不好，所以注入假的 `DrawIoIo`（与 `MermaidIo` 的注入点
 * 同理），钉住我们这一层的事：错误被说出来（绝不吞）、SVG 快照、超时兜底。
 *
 * 唯一用**真实默认 io** 的一组用例钉的是「无网络」那条线（ADR 0001）：
 * 脚本注入**之前**十条资源路径必须全部占住、加载去重、失败不缓存。
 * jsdom 不会真的去取脚本（resource loading 默认关闭），所以脚本加载成功与否
 * 由测试手动触发 `onload` 模拟 —— 恰好把「加载」这件事完全攥在测试手里。
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { t } from './i18n/index.js';

import {
  type DrawIoIo,
  type DrawioViewerLike,
  renderDrawio,
} from './drawio-render.js';

interface Harness {
  io: DrawIoIo;
  /** `loadViewer` 的调用次数。 */
  loads: number;
  /** 让下一次 `loadViewer` 失败一次；之后恢复成功。 */
  failNextLoad: (error: Error) => void;
  /** 让下一次 `createViewerForElement` 抛出。 */
  failNextCreate: (error: Error) => void;
  /** 每次创建查看器时往宿主里补的 svg（null = 不补）。 */
  svgToPaint: string | null;
}

const XML = '<mxfile><diagram name="页1"><mxGraphModel/></diagram></mxfile>';

function makeHarness(): Harness {
  const h: Harness = {
    loads: 0,
    svgToPaint: null,
    io: null as unknown as DrawIoIo,
    failNextLoad: () => undefined,
    failNextCreate: () => undefined,
  };

  let loadError: Error | null = null;
  let createError: Error | null = null;

  const viewer: DrawioViewerLike = {
    createViewerForElement: (element, onReady) => {
      if (createError !== null) {
        const error = createError;
        createError = null;
        throw error;
      }
      if (h.svgToPaint !== null) element.innerHTML = h.svgToPaint;
      onReady({});
    },
  };

  h.io = {
    loadViewer: () => {
      h.loads += 1;
      if (loadError !== null) {
        const error = loadError;
        loadError = null;
        return Promise.reject(error);
      }
      return Promise.resolve(viewer);
    },
    settle: () => Promise.resolve(),
  };

  h.failNextLoad = (error) => {
    loadError = error;
  };
  h.failNextCreate = (error) => {
    createError = error;
  };

  return h;
}

describe('renderDrawio · 注入的 io', () => {
  let host: HTMLElement;

  beforeEach(() => {
    host = document.createElement('div');
    document.body.appendChild(host);
  });

  afterEach(() => {
    document.body.innerHTML = '';
    vi.useRealTimers();
  });

  it('空内容短路：不加载查看器，理由直接说出来', async () => {
    const h = makeHarness();
    const result = await renderDrawio('   ', host, h.io);

    expect(result).toEqual({ ok: false, message: t('core.render.drawioEmpty') });
    expect(h.loads).toBe(0);
    expect(host.childElementCount).toBe(0);
  });

  it('成功：创建查看器元素、返回当前页的 SVG 快照', async () => {
    const h = makeHarness();
    h.svgToPaint = '<svg id="page1"><rect/></svg>';
    const result = await renderDrawio(XML, host, h.io);

    expect(result.ok).toBe(true);
    if (result.ok) {
      // jsdom 解析 innerHTML 后 outerHTML 会有归一化（属性顺序 / 自闭合），断言实质
      expect(result.svg).toContain('id="page1"');
      expect(result.svg).toContain('<rect');
    }
    // 宿主里的查看器 DOM 留着（多页切换、缩放的交互面），不是渲染完就清掉
    expect(host.querySelector('.mxgraph')).not.toBeNull();
    // XML 经 data-mxgraph 交给查看器 —— 这就是「配置层」的全部接口
    const mxgraph = host.querySelector('.mxgraph');
    expect(JSON.parse(mxgraph?.getAttribute('data-mxgraph') ?? '{}')).toMatchObject({
      nav: true,
      resize: true,
      xml: XML,
    });
  });

  it('画出来的页没有 svg（空页）：成功但快照为 null', async () => {
    const h = makeHarness();
    const result = await renderDrawio(XML, host, h.io);
    expect(result).toEqual({ ok: true, svg: null });
  });

  it('loadViewer 失败：给出一句话，容器不留残骸', async () => {
    const h = makeHarness();
    h.failNextLoad(new Error('脚本加载失败'));
    const result = await renderDrawio(XML, host, h.io);

    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain('脚本加载失败');
    expect(host.childElementCount).toBe(0);
  });

  it('createViewerForElement 抛出：错误说出来而不是吞掉', async () => {
    const h = makeHarness();
    h.failNextCreate(new Error('不是合法的 XML'));
    const result = await renderDrawio(XML, host, h.io);

    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain('不是合法的 XML');
  });

  it('查看器在元素里写了错误说明时，把它并进失败消息（留着比清空好）', async () => {
    const h = makeHarness();
    h.io = {
      loadViewer: async () => ({
        createViewerForElement: (element: HTMLElement, _onReady: (viewer: unknown) => void) => {
          element.textContent = ' Cannot parse XML ';
          throw new Error('boom');
        },
      }),
      settle: () => Promise.resolve(),
    };
    const result = await renderDrawio(XML, host, h.io);

    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toBe('boom Cannot parse XML');
  });

  it('回调一直不来：超时兜底把失败说出来，不吊死渲染链路', async () => {
    vi.useFakeTimers();
    const h = makeHarness();
    h.io = {
      loadViewer: () =>
        Promise.resolve({ createViewerForElement: () => undefined }), // 永不回调
      settle: () => Promise.resolve(),
    };

    const pending = renderDrawio(XML, host, h.io);
    const assertion = expect(pending).resolves.toMatchObject({
      ok: false,
      message: expect.stringContaining('渲染没有在预期时间内完成'),
    });
    await vi.advanceTimersByTimeAsync(16_000);
    await assertion;
  });
});

describe('renderDrawio · 默认 io 的加载（「无网络」那条线）', () => {
  // `pendingViewer` 是模块级状态 —— 每个用例重置模块、改用**重导的那份**，
  // 顶部静态 import 的实例永远指向第一次（它的状态会跨用例存活）
  let freshRender: typeof renderDrawio;

  beforeEach(async () => {
    vi.resetModules();
    freshRender = (await import('./drawio-render.js')).renderDrawio;
    // jsdom 没有 rAF；真实 settle 等两帧 —— 用 setTimeout 顶上即可
    globalThis.requestAnimationFrame = ((cb: FrameRequestCallback) =>
      setTimeout(() => cb(performance.now()), 0)) as unknown as typeof requestAnimationFrame;
  });

  afterEach(() => {
    document.body.innerHTML = '';
    document.head.innerHTML = '';
    delete (globalThis as { requestAnimationFrame?: unknown }).requestAnimationFrame;
    delete (window as { GraphViewer?: unknown }).GraphViewer;
  });

  /**
   * 这些测试里没有别人往 head 里放脚本，head 里的 script 就是注入的查看器。
   * （不去 spy `createElement`：vitest 环境的 document 包了一层代理，spy 出来会递归。）
   */
  function headScripts(): HTMLScriptElement[] {
    return Array.from(document.head.querySelectorAll('script'));
  }

  it('加载之前先把十条资源路径全部占住 —— 漏一条就是一次静默联网', async () => {
    const w = window as unknown as Record<string, unknown>;
    (w as { GraphViewer: unknown }).GraphViewer = {
      createViewerForElement: (_el: HTMLElement, onReady: (viewer: unknown) => void) => onReady({}),
    };

    const pending = freshRender(XML, document.createElement('div'));
    // 脚本已注入但 onload 还没触发：此刻路径就必须已经就位
    await new Promise((resolve) => setTimeout(resolve, 0));
    const scripts = headScripts();
    expect(scripts).toHaveLength(1);
    expect(scripts[0]!.src).toContain('drawio/viewer.min.js');

    const base = w['SHAPES_PATH'] as string;
    expect(typeof base).toBe('string');
    expect(w['STENCIL_PATH']).toBe(`${base?.replace('/shapes', '')}/stencils`);
    expect(w['STYLE_PATH']).toContain('/styles');
    expect(w['PROXY_URL']).toBe(''); // 不走任何代理
    expect(w['DRAW_MATH_URL']).toBe('');
    expect(w['GRAPH_IMAGE_PATH']).toContain('/img');
    expect(w['RESOURCES_PATH']).toContain('/resources');
    expect(w['RESOURCE_BASE']).toContain('/resources/grapheditor');
    expect(w['mxImageBasePath']).toContain('/mxgraph/images');
    expect(w['mxBasePath']).toContain('/mxgraph');
    expect(w['DRAWIO_BASE_URL']).toBe(''); // 新式遥测/代理入口：空串走「没配置」分支
    expect(w['DRAWIO_LOG_URL']).toBe('');
    expect(typeof w['onDrawioViewerLoad']).toBe('function'); // 拦住脚本末尾的自动渲染

    scripts[0]!.onload?.(new Event('load'));
    await expect(pending).resolves.toMatchObject({ ok: true });
  });

  it('连续渲染共用一次加载：脚本只注入一次', async () => {
    (window as { GraphViewer?: unknown }).GraphViewer = {
      createViewerForElement: (_el: HTMLElement, onReady: (viewer: unknown) => void) => onReady({}),
    };

    // jsdom 不会真的去取脚本，onload 由测试手动触发
    const first = freshRender(XML, document.createElement('div'));
    headScripts()[0]!.onload?.(new Event('load'));
    await first;
    await freshRender(XML, document.createElement('div'));

    expect(headScripts()).toHaveLength(1);
  });

  it('失败不缓存：GraphViewer 没出现时给出原因，下一次渲染重试', async () => {
    const first = freshRender(XML, document.createElement('div'));
    // 脚本「加载成功」但全局入口没出现 —— 这是一次失败
    headScripts()[0]!.onload?.(new Event('load'));
    const firstResult = await first;
    expect(firstResult.ok).toBe(false);
    if (!firstResult.ok) expect(firstResult.message).toContain('没有找到 GraphViewer 入口');

    // 修好（模拟资源补齐）：下一次渲染要**重新注入**，而不是复用失败的 Promise
    (window as { GraphViewer?: unknown }).GraphViewer = {
      createViewerForElement: (_el: HTMLElement, onReady: (viewer: unknown) => void) => onReady({}),
    };
    const second = freshRender(XML, document.createElement('div'));
    headScripts()[1]!.onload?.(new Event('load'));
    const secondResult = await second;

    expect(headScripts()).toHaveLength(2);
    expect(secondResult).toMatchObject({ ok: true });
  });
});
