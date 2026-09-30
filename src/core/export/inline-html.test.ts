// @vitest-environment jsdom
/**
 * 自包含导出的三个关键动作：代码高亮、图片内联、公式渲染。
 *
 * 这三件事的失败模式都是**静默的**：导出的 HTML 看着好好的，拷到别人机器上
 * 才发现代码全是黑的、图片全是破图标、公式掉了字形。所以必须在单测里把它们
 * 钉住，不能只靠「我点了一下导出，看起来没问题」。
 */

import { describe, expect, it, vi } from 'vitest';

import { MATH_BLOCK, MATH_INLINE, TEX_DATA_ATTR } from '../math.js';
import { inlineAssets, renderDiagrams, renderMath, type InlineReport } from './inline-html.js';
import type { MathStyleIo } from './katex-css.js';
import { buildSelfContainedHtml } from './document.js';

/** 造一个 1×1 的 PNG 字节（文件头是真的，足以让 MIME 嗅探认出 image/png）。 */
function pngBytes(): Uint8Array {
  return new Uint8Array([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 1, 2, 3, 4]);
}

function host(html: string): HTMLElement {
  const root = document.createElement('div');
  root.className = 'qm-doc';
  root.innerHTML = html;
  return root;
}

/** 公式壳：内核的 `toDOM` 产出的就是它（见 `modules/markdown/math.ts`）。 */
function mathShell(tex: string, kind: 'inline' | 'block'): string {
  const tag = kind === 'block' ? 'div' : 'span';
  const type = kind === 'block' ? MATH_BLOCK : MATH_INLINE;
  const className = kind === 'block' ? 'qm-math qm-math-block' : 'qm-math qm-math-inline';
  const source = kind === 'block' ? `$$\n${tex}\n$$` : `$${tex}$`;
  return `<${tag} class="${className}" data-type="${type}" ${TEX_DATA_ATTR}="${tex}">${source}</${tag}>`;
}

/**
 * 一份**假的** KaTeX 样式。
 *
 * 用假数据而不是真实的 `katex.min.css`：这里要验的是「拿到 CSS 之后怎么拼进
 * 导出文件」，字体怎么拆、体积多大那件事由 `katex-css.test.ts` 对真实文件钉住。
 */
const FAKE_MATH_IO: MathStyleIo = {
  readCss: async () =>
    '@font-face{font-family:KaTeX_Main;src:url(fonts/KaTeX_Main-Regular.woff2) format("woff2")}.katex{font:normal 1.21em KaTeX_Main,serif}',
  loadFont: async (file) => `data:font/woff2;base64,${file}`,
};

describe('代码高亮', () => {
  it('把 <pre data-language> 里的代码变成带 hljs 类的 span', async () => {
    const root = host('<pre data-language="js"><code>const a = 1;</code></pre>');

    const report = await inlineAssets(root, { baseDir: null });

    expect(report.highlightedBlocks).toBe(1);
    const code = root.querySelector('code');
    expect(code?.innerHTML).toContain('hljs-keyword');
    // 高亮后文本内容必须一字不差 —— 丢字比不着色严重得多
    expect(code?.textContent).toBe('const a = 1;');
  });

  it('没写语言时自动识别，并把识别结果写回 data-language', async () => {
    // 以前面没有 data-language 的 pre，角标就不该出现
    const root = host('<pre><code>function hello() { return 1; }</code></pre>');

    const report = await inlineAssets(root, { baseDir: null });

    expect(report.highlightedBlocks).toBe(1);
    expect(root.querySelector('pre')?.getAttribute('data-language')).toBeTruthy();
  });

  it('已有语言时不覆盖用户写明的语言', async () => {
    const root = host('<pre data-language="python"><code>def f(): pass</code></pre>');
    await inlineAssets(root, { baseDir: null });
    expect(root.querySelector('pre')?.getAttribute('data-language')).toBe('python');
  });

  it('空代码块跳过，不产生空 span', async () => {
    const root = host('<pre data-language="js"><code>   </code></pre>');
    const report = await inlineAssets(root, { baseDir: null });
    expect(report.highlightedBlocks).toBe(0);
  });

  it('HTML 实体在往返后保持正确（&lt; 不能变成真的标签）', async () => {
    const root = host('<pre data-language="html"><code>&lt;div&gt;a &amp; b&lt;/div&gt;</code></pre>');
    await inlineAssets(root, { baseDir: null });
    const code = root.querySelector('code');
    expect(code?.textContent).toBe('<div>a & b</div>');
  });
});

describe('图片内联', () => {
  it('相对路径按 baseDir 解析，并写成 data URI', async () => {
    const root = host('<p><img src="img/logo.png" alt="logo"></p>');
    const readBinaryFile = vi.fn(async () => pngBytes().buffer as ArrayBuffer);

    const report = await inlineAssets(root, { baseDir: 'C:\\docs\\note', readBinaryFile });

    expect(readBinaryFile).toHaveBeenCalledWith('C:\\docs\\note\\img\\logo.png');
    expect(report.inlinedImages).toBe(1);
    expect(report.missingImages).toBe(0);
    expect(root.querySelector('img')?.getAttribute('src')).toMatch(/^data:image\/png;base64,/);
    // 内联不是「加个属性」，原来的 src 必须被替换掉
    expect(root.querySelector('img')?.getAttribute('src')).not.toContain('logo.png');
  });

  it('URL 编码的路径会被解码', async () => {
    const root = host('<img src="my%20pics/a.png">');
    const readBinaryFile = vi.fn(async () => pngBytes().buffer as ArrayBuffer);

    await inlineAssets(root, { baseDir: '/docs', readBinaryFile });

    expect(readBinaryFile).toHaveBeenCalledWith('/docs/my pics/a.png');
  });

  it('已经是 data URI 的图片原样保留', async () => {
    const root = host('<img src="data:image/gif;base64,R0lGOD">');
    const readBinaryFile = vi.fn(async () => pngBytes().buffer as ArrayBuffer);

    const report = await inlineAssets(root, { baseDir: '/docs', readBinaryFile });

    expect(readBinaryFile).not.toHaveBeenCalled();
    expect(report.inlinedImages).toBe(1);
    expect(root.querySelector('img')?.getAttribute('src')).toBe('data:image/gif;base64,R0lGOD');
  });

  it('网络图片不内联，但被计入并把情况报出去', async () => {
    const root = host('<img src="https://example.com/a.png">');
    const readBinaryFile = vi.fn(async () => pngBytes().buffer as ArrayBuffer);

    const report = await inlineAssets(root, { baseDir: '/docs', readBinaryFile });

    expect(readBinaryFile).not.toHaveBeenCalled();
    expect(report.remoteImages).toBe(1);
    expect(report.inlinedImages).toBe(0);
    // 不内联也不能给它打「读不到」的标记 —— 它是能显示的，只是需要联网
    expect(root.querySelector('img')?.hasAttribute('data-qm-missing')).toBe(false);
  });

  it('读不到字节时保留原链接并打上标记，而不是丢掉这张图', async () => {
    const root = host('<img src="missing.png" alt="丢了">');
    const readBinaryFile = vi.fn(async () => {
      throw new Error('ENOENT');
    });

    const report = await inlineAssets(root, { baseDir: '/docs', readBinaryFile });

    expect(report.missingImages).toBe(1);
    const img = root.querySelector('img');
    expect(img?.getAttribute('src')).toBe('missing.png');
    expect(img?.hasAttribute('data-qm-missing')).toBe(true);
    // alt 不能被顺手清掉：图片加载不出来时它是唯一的说明
    expect(img?.getAttribute('alt')).toBe('丢了');
  });

  it('一张图失败不影响其余图片与高亮', async () => {
    const root = host(
      '<img src="bad.png"><pre data-language="js"><code>const a = 1;</code></pre><img src="good.png">',
    );
    const readBinaryFile = vi.fn(async (path: string) => {
      if (path.endsWith('bad.png')) throw new Error('ENOENT');
      return pngBytes().buffer as ArrayBuffer;
    });

    const report = await inlineAssets(root, { baseDir: '/docs', readBinaryFile });

    expect(report.missingImages).toBe(1);
    expect(report.inlinedImages).toBe(1);
    expect(report.highlightedBlocks).toBe(1);
  });

  it('没有二进制读取能力时全部标注为未内联，但高亮照常', async () => {
    const root = host('<img src="a.png"><pre data-language="js"><code>const a = 1;</code></pre>');

    const report = await inlineAssets(root, { baseDir: '/docs' });

    expect(report.missingImages).toBe(1);
    expect(report.highlightedBlocks).toBe(1);
  });
});

describe('buildSelfContainedHtml', () => {
  it('产出的是完整文档，版式内联，不引用任何外部资源', async () => {
    const { html, report } = await buildSelfContainedHtml({
      title: '我的笔记',
      bodyHtml: '<h1>标题</h1><pre data-language="js"><code>const a = 1;</code></pre>',
      fallbackText: '',
      baseDir: null,
    });

    expect(html.startsWith('<!DOCTYPE html>')).toBe(true);
    expect(html).toContain('<meta charset="utf-8">');
    expect(html).toContain('<title>我的笔记</title>');
    // 版式必须在 <style> 里，而不是 <link>
    // `--qm-ink` 只存在于 document-styles.css，用它来确认 ?raw 内联真的发生了，
    // 而不是被 article 上的 class 名骗过
    expect(html).toContain('--qm-ink');
    expect(html).toContain('.qm-doc');
    expect(html).not.toContain('<link');
    expect(html).not.toMatch(/src\s*=\s*["']https?:/);
    expect(report.highlightedBlocks).toBe(1);
  });

  it('标题里的尖括号被转义，不会撕坏 <title>', async () => {
    const { html } = await buildSelfContainedHtml({
      title: 'a<b>&"c"',
      bodyHtml: '<p>x</p>',
      fallbackText: '',
      baseDir: null,
    });
    expect(html).toContain('<title>a&lt;b&gt;&amp;&quot;c&quot;</title>');
  });

  it('内核给不出结构时退化为 <pre>，并且原文里的尖括号被转义', async () => {
    const { html } = await buildSelfContainedHtml({
      title: '纯文本',
      bodyHtml: null,
      fallbackText: 'if (a < b) { return a & b; }',
      baseDir: null,
    });

    expect(html).toContain('qm-plain');
    expect(html).toContain('if (a &lt; b) { return a &amp; b; }');
    // 不能被当成真的标签渲染出来
    expect(html).not.toContain('<b)');
  });

  it('body 内容被包进 .qm-doc 容器 —— 版式选择器都挂在它下面', async () => {
    const { html } = await buildSelfContainedHtml({
      title: 't',
      bodyHtml: '<p>正文</p>',
      fallbackText: '',
      baseDir: null,
    });
    expect(html).toContain('<article class="qm-doc">');
    expect(html).toContain('<p>正文</p>');
  });
});

// ---------------------------------------------------------------------------
// 公式
// ---------------------------------------------------------------------------

/** 一份空的报告，用来单独喂给 `renderMath`。 */
function emptyReport(): InlineReport {
  return {
    inlinedImages: 0,
    missingImages: 0,
    remoteImages: 0,
    highlightedBlocks: 0,
    renderedMath: 0,
    failedMath: 0,
    firstMathError: null,
    renderedDiagrams: 0,
    failedDiagrams: 0,
    firstDiagramError: null,
  };
}

describe('公式渲染', () => {
  /**
   * 内核写的是「带 `data-tex` 的壳 + 源码」，导出侧把壳里的内容换成 KaTeX 的结果。
   * **壳本身必须留着** —— 复制一段导出内容粘回轻墨时，`parseDOM` 从 `data-tex`
   * 上把 TeX 取回来；壳没了就只能粘回来一堆 KaTeX 的 HTML。
   */
  it('把壳里的源码换成 KaTeX 的结果，`data-tex` 留着', async () => {
    const root = host(`<p>前 ${mathShell('x^2', 'inline')} 后</p>`);
    const report = emptyReport();

    await renderMath(root, report);

    expect(report.renderedMath).toBe(1);
    expect(report.failedMath).toBe(0);

    const box = root.querySelector('.qm-math-inline') as HTMLElement;
    expect(box.className).toContain('is-loaded');
    expect(box.innerHTML).toContain('katex');
    expect(box.innerHTML).toContain('<math');
    expect(box.getAttribute(TEX_DATA_ATTR)).toBe('x^2');
    // 源码不该还留在可见内容里
    expect(box.textContent).not.toContain('$x^2$');
  });

  it('块级公式走 displayMode', async () => {
    const root = host(mathShell('\\frac{1}{2}', 'block'));
    await renderMath(root, emptyReport());

    expect((root.querySelector('.qm-math-block') as HTMLElement).innerHTML).toContain('katex-display');
  });

  /**
   * 一份导出文件是**发出去**的。渲染失败时留一个空壳等于「这段内容凭空消失」，
   * 留下 `$x^2$` 至少让人看得出原本写了什么、也看得出是哪里写错了。
   */
  it('渲染失败时留下源码与原因，而不是空白', async () => {
    const root = host(`<p>${mathShell('\\foo{', 'inline')}</p>`);
    const report = emptyReport();

    await renderMath(root, report);

    expect(report.renderedMath).toBe(0);
    expect(report.failedMath).toBe(1);
    expect(report.firstMathError).toContain('Undefined control sequence');

    const box = root.querySelector('.qm-math') as HTMLElement;
    expect(box.className).toContain('is-failed');
    expect(box.querySelector('.qm-math-src')?.textContent).toBe('$\\foo{$');
    expect(box.querySelector('.qm-math-note')?.textContent).toContain('Undefined control sequence');
  });

  it('一处失败不中断其余公式', async () => {
    const root = host(
      `<p>${mathShell('x', 'inline')} ${mathShell('\\foo{', 'inline')} ${mathShell('y', 'inline')}</p>`,
    );
    const report = emptyReport();

    await renderMath(root, report);

    expect(report.renderedMath).toBe(2);
    expect(report.failedMath).toBe(1);
  });

  /**
   * 只留**第一条**错误：一份文档里十个公式都写错，提示里堆十句话没人看。
   * 用户看到其中一条就知道自己写错了什么，回去改就是了。
   */
  it('只记第一条错误', async () => {
    const root = host(`<p>${mathShell('\\aaa{', 'inline')} ${mathShell('\\bbb{', 'inline')}</p>`);
    const report = emptyReport();

    await renderMath(root, report);

    expect(report.failedMath).toBe(2);
    expect(report.firstMathError).toContain('\\aaa');
  });

  it('没有公式时什么都不做', async () => {
    const root = host('<p>正文</p>');
    const report = emptyReport();

    await renderMath(root, report);

    expect(report.renderedMath).toBe(0);
    expect(report.failedMath).toBe(0);
    expect(report.firstMathError).toBeNull();
  });
});

describe('buildSelfContainedHtml：公式', () => {
  it('把 KaTeX 的样式与字体拼进去，并如实报告体积', async () => {
    const { html, report } = await buildSelfContainedHtml({
      title: 't',
      bodyHtml: `<p>${mathShell('x^2', 'inline')}</p>`,
      fallbackText: '',
      baseDir: null,
      mathIo: FAKE_MATH_IO,
    });

    expect(report.renderedMath).toBe(1);
    expect(report.mathFonts).toBe(1);
    expect(report.missingMathFonts).toBe(0);
    // 体积要按 **UTF-8 字节**算：`text.length` 会少报中文与 base64 之外的部分，
    // 用户会觉得「提示说 300 KB，文件怎么大了 400 KB」
    expect(report.mathStyleBytes).toBeGreaterThan(0);

    expect(html).toContain('@font-face');
    expect(html).toContain('data:font/woff2;base64,KaTeX_Main-Regular.woff2');
    expect(html).toContain('.katex');
    // 渲染结果也在里面
    expect(html).toContain('katex-mathml');
    // 依然是自包含的
    expect(html).not.toContain('<link');
  });

  /**
   * **没有公式的文档一个字节都不该多出来。**
   *
   * 带字体的 KaTeX 样式有 300 KB 上下。所以这里不只是断言「没有 KaTeX 的 CSS」，
   * 而是用一个**一调用就抛**的 io 证明：没有公式时**连 CSS 都不去读**。
   */
  it('没有公式时不读 KaTeX 的样式', async () => {
    const neverIo: MathStyleIo = {
      readCss: async () => {
        throw new Error('没有公式时不该读 KaTeX 的 CSS');
      },
      loadFont: async () => {
        throw new Error('没有公式时不该取字体');
      },
    };

    const { html, report } = await buildSelfContainedHtml({
      title: 't',
      bodyHtml: '<p>正文</p>',
      fallbackText: '',
      baseDir: null,
      mathIo: neverIo,
    });

    expect(report.mathFonts).toBe(0);
    expect(report.missingMathFonts).toBe(0);
    expect(report.mathStyleBytes).toBe(0);
    expect(html).not.toContain('@font-face');
    expect(html).not.toContain('katex-mathml');
  });

  /**
   * 公式样式单独一个 `<style>`，排在版式**之前**。
   * 分开是为了让「这份导出为什么大」一眼可查 —— 版式只有几 KB，
   * 剩下三百 KB 全是字体。
   */
  it('公式样式排在版式之前', async () => {
    const { html } = await buildSelfContainedHtml({
      title: 't',
      bodyHtml: `<p>${mathShell('x', 'inline')}</p>`,
      fallbackText: '',
      baseDir: null,
      mathIo: FAKE_MATH_IO,
    });

    // `--qm-ink` 只存在于 document-styles.css，用它当版式那一份的指纹
    expect(html.indexOf('@font-face')).toBeLessThan(html.indexOf('--qm-ink'));
  });

  /**
   * 字体取不到时**如实上报**，并且不留下指向 `fonts/…` 的规则 ——
   * 那个路径在单文件里不存在，留着只会让打开时多出 20 个 404。
   */
  it('字体取不到时计入 missingMathFonts，且不留坏引用', async () => {
    const io: MathStyleIo = { readCss: FAKE_MATH_IO.readCss, loadFont: async () => null };

    const { html, report } = await buildSelfContainedHtml({
      title: 't',
      bodyHtml: `<p>${mathShell('x', 'inline')}</p>`,
      fallbackText: '',
      baseDir: null,
      mathIo: io,
    });

    expect(report.mathFonts).toBe(0);
    expect(report.missingMathFonts).toBe(1);
    expect(html).not.toContain('url(fonts/');
    // 版式规则仍在（公式的间距还得靠它），只是字体退化
    expect(html).toContain('.katex');
  });

  it('公式渲染失败时导出文件里留下源码，而不是空白', async () => {
    const { html, report } = await buildSelfContainedHtml({
      title: 't',
      bodyHtml: `<p>${mathShell('\\foo{', 'inline')}</p>`,
      fallbackText: '',
      baseDir: null,
      mathIo: FAKE_MATH_IO,
    });

    expect(report.failedMath).toBe(1);
    expect(report.firstMathError).toContain('Undefined control sequence');
    expect(html).toContain('qm-math-src');
    expect(html).toContain('\\foo{');
  });
});

// ---------------------------------------------------------------------------
// 图表（Mermaid）
// ---------------------------------------------------------------------------

/**
 * 这里把渲染器换成假的。
 *
 * mermaid 在 jsdom 里画不出东西（`getBBox()` 一律返回 0，见
 * `core/mermaid-render.ts` 的文件头），真去加载它既慢又只会得到一张塌掉的图。
 * 单测能证明的是**导出这一层的契约**：哪一块该被换掉、换成了什么、
 * 失败时留下了什么。
 */
const diagramSpy = vi.hoisted(() => {
  const state: {
    calls: Array<{ source: string; theme: string }>;
    result: { ok: true; svg: string } | { ok: false; message: string };
  } = {
    calls: [],
    result: { ok: true, svg: '<svg id="g"><rect/></svg>' },
  };
  return state;
});

vi.mock('../mermaid-render.js', () => ({
  renderMermaid: (source: string, theme: string) => {
    diagramSpy.calls.push({ source, theme });
    return Promise.resolve(diagramSpy.result);
  },
}));

/** 内核产出的 mermaid 代码块：`pre[data-language]` 里套一个 `code`。 */
function mermaidShell(source: string): string {
  return `<pre data-language="mermaid"><code>${source}</code></pre>`;
}

describe('图表渲染', () => {
  it('把 mermaid 代码块换成画好的 SVG', async () => {
    diagramSpy.calls = [];
    diagramSpy.result = { ok: true, svg: '<svg id="g"><rect/></svg>' };
    const root = host(`<p>前</p>${mermaidShell('graph LR\n  A-->B')}`);
    const report = emptyReport();

    await renderDiagrams(root, report, 'light');

    expect(report.renderedDiagrams).toBe(1);
    expect(report.failedDiagrams).toBe(0);
    const figure = root.querySelector('.qm-diagram') as HTMLElement;
    expect(figure.innerHTML).toContain('<svg');
    // 源码块必须消失：导出产物里没有「另一种看法」这回事
    expect(root.querySelector('pre')).toBeNull();
  });

  it('画不出来时留下源码，并在后面说明原因', async () => {
    diagramSpy.calls = [];
    diagramSpy.result = { ok: false, message: '图表语法有误' };
    const root = host(mermaidShell('graph LR\n  A-->B'));
    const report = emptyReport();

    await renderDiagrams(root, report, 'light');

    expect(report.failedDiagrams).toBe(1);
    expect(report.firstDiagramError).toContain('图表语法有误');
    // 源码还在（发出去的文件里，一段代码好过一个空白）
    expect(root.querySelector('pre')?.textContent).toContain('graph LR');
    expect(root.querySelector('.qm-diagram-error')?.textContent).toContain('图表语法有误');
  });

  it('不是 mermaid 的代码块一个都不动，渲染器一次都不叫', async () => {
    diagramSpy.calls = [];
    const root = host('<pre data-language="js"><code>let a = 1;</code></pre>');
    const report = emptyReport();

    await renderDiagrams(root, report, 'light');

    expect(diagramSpy.calls).toHaveLength(0);
    expect(root.querySelector('pre')).not.toBeNull();
    expect(report.renderedDiagrams).toBe(0);
  });

  it('空的 mermaid 块不算数：既不画，也不报错', async () => {
    diagramSpy.calls = [];
    const root = host(mermaidShell('   '));
    const report = emptyReport();

    await renderDiagrams(root, report, 'light');

    expect(diagramSpy.calls).toHaveLength(0);
    expect(report.renderedDiagrams).toBe(0);
    expect(report.failedDiagrams).toBe(0);
  });

  it('大小写不规范的围栏也认（`Mermaid` 与 `mermaid` 是一回事）', async () => {
    diagramSpy.calls = [];
    diagramSpy.result = { ok: true, svg: '<svg id="g"></svg>' };
    const root = host('<pre data-language="Mermaid"><code>graph LR</code></pre>');

    await renderDiagrams(root, emptyReport(), 'light');

    expect(diagramSpy.calls).toHaveLength(1);
  });

  it('导出是一张纸：图永远按浅色画，哪怕应用此刻是深色', async () => {
    diagramSpy.calls = [];
    diagramSpy.result = { ok: true, svg: '<svg id="g"></svg>' };
    const root = host(mermaidShell('graph LR'));

    await renderDiagrams(root, emptyReport(), 'light');

    expect(diagramSpy.calls[0]?.theme).toBe('light');
  });
});
