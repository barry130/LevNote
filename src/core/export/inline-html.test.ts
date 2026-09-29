// @vitest-environment jsdom
/**
 * 自包含导出的两个关键动作：代码高亮、图片内联。
 *
 * 这两件事的失败模式都是**静默的**：导出的 HTML 看着好好的，拷到别人机器上
 * 才发现代码全是黑的、图片全是破图标。所以必须在单测里把它们钉住，
 * 不能只靠「我点了一下导出，看起来没问题」。
 */

import { describe, expect, it, vi } from 'vitest';

import { inlineAssets } from './inline-html.js';
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
