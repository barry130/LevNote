// @vitest-environment jsdom

import { afterEach, describe, expect, it } from 'vitest';
import { t } from './i18n/index.js';
import type { RenderResult } from 'mermaid';
import {
  MEASURE_CONTAINER_STYLE,
  cachedLoader,
  describeMermaidError,
  renderMermaid,
  withMeasureContainer,
  type MermaidIo,
  type MermaidLike,
} from './mermaid-render.js';

/**
 * 这里的假实现**不加载真的 mermaid**。
 *
 * 不是因为慢，而是因为在 jsdom 里它根本画不出东西：mermaid 靠 `getBBox()`
 * 量文字尺寸，而 jsdom 没有布局引擎，一律返回 0。也就是说「渲染出来长什么样」
 * 这件事单测证明不了，只能靠人工看。单测能证明的是**我们这一层的契约**：
 * 容器给了没有、给错时摘没摘、错误有没有被吞掉、配置有没有跟上主题。
 */
interface Fake {
  readonly mermaid: MermaidLike;
  /** 每次 `initialize` 收到的配置。 */
  readonly configs: MermaidConfigLike[];
  /** 每次 `render` 收到的参数。 */
  readonly calls: RenderCall[];
}

type MermaidConfigLike = Parameters<MermaidLike['initialize']>[0];
interface RenderCall {
  readonly id: string;
  readonly text: string;
  readonly container: Element;
}

function fake(
  render?: (id: string, text: string, container: Element) => Promise<RenderResult>,
): Fake {
  const configs: MermaidConfigLike[] = [];
  const calls: RenderCall[] = [];
  const mermaid: MermaidLike = {
    initialize: (config) => {
      configs.push(config);
    },
    render: async (id, text, container) => {
      calls.push({ id, text, container });
      return render?.(id, text, container) ?? { svg: '<svg></svg>', diagramType: 'flowchart' };
    },
  };
  return { mermaid, configs, calls };
}

function io(mermaid: MermaidLike): MermaidIo {
  return { load: async () => mermaid, withContainer: withMeasureContainer };
}

function failingIo(error: Error): MermaidIo {
  return {
    load: () => Promise.reject(error),
    withContainer: withMeasureContainer,
  };
}

afterEach(() => {
  document.body.innerHTML = '';
});

describe('cachedLoader', () => {
  it('并发只加载一次', async () => {
    let calls = 0;
    const load = cachedLoader(async () => {
      calls += 1;
      return fake().mermaid;
    });

    await Promise.all([load(), load(), load()]);
    expect(calls).toBe(1);
  });

  it('成功后不再加载', async () => {
    let calls = 0;
    const load = cachedLoader(async () => {
      calls += 1;
      return fake().mermaid;
    });

    await load();
    await load();
    expect(calls).toBe(1);
  });

  it('失败不缓存 —— 否则用户修好了文件也得重启应用', async () => {
    let calls = 0;
    const load = cachedLoader(async () => {
      calls += 1;
      if (calls < 2) throw new Error('chunk 404');
      return fake().mermaid;
    });

    await expect(load()).rejects.toThrow('chunk 404');
    await expect(load()).resolves.toBeDefined();
    expect(calls).toBe(2);
  });
});

describe('renderMermaid', () => {
  it('成功时把 mermaid 的 svg 原样带出来', async () => {
    const { mermaid } = fake(async () => ({ svg: '<svg id="a">图</svg>', diagramType: 'flowchart' }));

    const result = await renderMermaid('graph LR\n a-->b', 'light', io(mermaid));

    expect(result).toEqual({ ok: true, svg: '<svg id="a">图</svg>' });
  });

  it('加载失败要说出「渲染器加载失败」，而不是留白', async () => {
    const result = await renderMermaid('graph LR', 'light', failingIo(new Error('chunk 404')));

    expect(result.ok).toBe(false);
    expect(result.ok === false && result.message).toContain('图表渲染器加载失败');
  });

  it('加载失败时也要带上原始原因', async () => {
    const result = await renderMermaid('graph LR', 'light', failingIo(new Error('没有这个 chunk')));

    expect(result.ok === false && result.message).toContain('没有这个 chunk');
  });

  it('空源码不去加载 mermaid —— 否则一个空块就会拉下一堆 chunk', async () => {
    let loaded = 0;
    const failing: MermaidIo = {
      load: () => {
        loaded += 1;
        return Promise.resolve(fake().mermaid);
      },
      withContainer: withMeasureContainer,
    };

    const result = await renderMermaid('   \n  ', 'light', failing);

    expect(loaded).toBe(0);
    expect(result).toEqual({ ok: false, message: t('core.render.mermaidEmpty') });
  });

  it('渲染抛错时把原因带出来，不吞掉', async () => {
    const { mermaid } = fake(async () => {
      throw new Error('No diagram type detected');
    });

    const result = await renderMermaid('这不是图', 'light', io(mermaid));

    expect(result).toEqual({ ok: false, message: 'No diagram type detected' });
  });

  it('传给 mermaid 的容器在文档里，且不是 body 本身', async () => {
    // 必须在**渲染进行中**看：渲染一结束容器就被摘掉了（见下面两条）
    let isBody: boolean | null = null;
    let inDocument: boolean | null = null;
    const { mermaid } = fake(async (_id, _text, container) => {
      isBody = container === document.body;
      inDocument = document.body.contains(container);
      return { svg: '<svg></svg>', diagramType: 'flowchart' };
    });

    await renderMermaid('graph LR', 'light', io(mermaid));

    expect(isBody).toBe(false);
    expect(inDocument).toBe(true);
  });

  it('容器用完就摘掉 —— 成功那条路', async () => {
    const { mermaid } = fake();
    const before = document.body.childElementCount;

    await renderMermaid('graph LR', 'light', io(mermaid));

    expect(document.body.childElementCount).toBe(before);
  });

  it('容器用完就摘掉 —— 抛错那条路也不能留在 body 上', async () => {
    const { mermaid } = fake(async () => {
      throw new Error('boom');
    });
    const before = document.body.childElementCount;

    await renderMermaid('graph LR', 'light', io(mermaid));

    expect(document.body.childElementCount).toBe(before);
  });

  it('每次渲染一个不同的 id —— mermaid 按 id 在 document 上找临时节点', async () => {
    const { mermaid, calls } = fake();

    await renderMermaid('graph LR', 'light', io(mermaid));
    await renderMermaid('graph LR', 'light', io(mermaid));

    expect(calls[0]!.id).not.toBe(calls[1]!.id);
  });

  it('第一次渲染前 initialize，且安全带为 strict', async () => {
    const { mermaid, configs } = fake();

    await renderMermaid('graph LR', 'light', io(mermaid));

    expect(configs).toHaveLength(1);
    expect(configs[0]!.securityLevel).toBe('strict');
    // 扫页面的那条路要关掉：我们只调 render
    expect(configs[0]!.startOnLoad).toBe(false);
  });

  it('同一个模块、同一个主题不重复 initialize', async () => {
    const { mermaid, configs } = fake();

    await renderMermaid('graph LR', 'light', io(mermaid));
    await renderMermaid('graph TD', 'light', io(mermaid));

    expect(configs).toHaveLength(1);
  });

  it('主题变了要重新 initialize —— 否则深色下画出浅色图', async () => {
    const { mermaid, configs } = fake();

    await renderMermaid('graph LR', 'light', io(mermaid));
    await renderMermaid('graph LR', 'dark', io(mermaid));

    expect(configs).toHaveLength(2);
    expect(configs[1]!.theme).toBe('dark');
  });

  it('换了一个模块（上一次加载失败过）也要重新 initialize', async () => {
    const first = fake();
    const second = fake();

    await renderMermaid('graph LR', 'light', io(first.mermaid));
    await renderMermaid('graph LR', 'light', io(second.mermaid));

    expect(second.configs).toHaveLength(1);
  });
});

describe('withMeasureContainer', () => {
  it('容器必须参与布局：不能用 display:none，否则 getBBox 全是 0', () => {
    expect(MEASURE_CONTAINER_STYLE).toContain('position:fixed');
    expect(MEASURE_CONTAINER_STYLE).not.toContain('display:none');
  });

  it('容器挪到屏幕外，且在文档里', async () => {
    let seen: HTMLElement | null = null;

    await withMeasureContainer(async (container) => {
      seen = container;
      expect(document.body.contains(container)).toBe(true);
      expect(container.getAttribute('style')).toBe(MEASURE_CONTAINER_STYLE);
    });

    expect(seen).not.toBeNull();
  });

  it('run 抛错时容器照样被摘掉', async () => {
    const before = document.body.childElementCount;

    await expect(
      withMeasureContainer(async () => {
        throw new Error('boom');
      }),
    ).rejects.toThrow('boom');

    expect(document.body.childElementCount).toBe(before);
  });
});

describe('describeMermaidError', () => {
  it('语法错误保留原文 —— 那段 ASCII 示意图是唯一能指出位置的东西', () => {
    const raw = "Parse error on line 2:\ngraph TB; a-->|b|\n------------------^\nExpecting 'SEMI'";

    const message = describeMermaidError(new Error(raw));

    expect(message).toContain('图表语法有误');
    expect(message).toContain(raw);
  });

  it('普通错误直接给原文', () => {
    expect(describeMermaidError(new Error('No diagram type detected'))).toBe(
      'No diagram type detected',
    );
  });

  it('空消息不能显示成空白 —— 那会被当成「编辑器坏了」', () => {
    expect(describeMermaidError(new Error('   '))).toContain('没有给出原因');
  });

  it('抛的不是 Error 也不能变成 [object Object]', () => {
    expect(describeMermaidError({ weird: true })).toBe('[object Object]');
    expect(describeMermaidError('炸了')).toBe('炸了');
  });
});
