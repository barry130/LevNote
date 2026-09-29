// @vitest-environment jsdom
/**
 * 图片的**内核半边**：显示（NodeView）与粘贴（handlePaste）。
 *
 * ## 这一组测试真正在钉的三条不变量
 *
 * 1. **NodeView 画的 DOM 绝不能进文档。** 我们往 `<img>` 里写的是解析后的
 *    data URI（几千个字符），而文档里该留的是 `./a.assets/x.png`。
 *    少了 `ignoreMutation`，存出来的 Markdown 里会多出一串 base64 ——
 *    而界面上一切正常，只有打开那个文件才会发现。
 * 2. **粘贴必须让文档变脏。** 落盘是异步的，等它回来时「刚发生过用户输入」
 *    的窗口早就过了。不显式声明的话，用户按 Ctrl+S 写回磁盘的是粘贴**之前**
 *    的文本，而界面上那张图还在 —— 静默丢数据。
 * 3. **过期的解析结果不能盖到新的图上。** 用户改了路径，而读盘是异步的：
 *    先发的请求后回来就会把旧图贴上去。
 */

import { afterEach, describe, expect, it } from 'vitest';

import type { ImagePasteRequest } from '../../core/editor/types.js';
import { createMarkdownEditor, type MarkdownEditorInstance } from './setup.js';

let opened: MarkdownEditorInstance | null = null;

afterEach(async () => {
  await opened?.destroy();
  opened = null;
  document.body.innerHTML = '';
});

interface Harness {
  editor: MarkdownEditorInstance;
  root: HTMLDivElement;
  dom: HTMLElement;
  /** 内核报给外壳的内容变更。空数组 = 这次编辑没告诉外壳（就是丢数据）。 */
  reports: string[];
  /** 内核向外壳要过的 src，按顺序。 */
  asked: string[];
}

interface OpenOptions {
  resolveImageUrl?: (src: string) => Promise<string | null>;
  onImagePaste?: (request: ImagePasteRequest) => Promise<string | null>;
}

async function open(source: string, options: OpenOptions = {}): Promise<Harness> {
  const root = document.createElement('div');
  document.body.appendChild(root);
  const reports: string[] = [];
  const asked: string[] = [];

  const editor = await createMarkdownEditor({
    root,
    defaultValue: source,
    onChange: (markdown) => reports.push(markdown),
    resolveImageUrl: options.resolveImageUrl
      ? (src) => {
          asked.push(src);
          return options.resolveImageUrl!(src);
        }
      : undefined,
    onImagePaste: options.onImagePaste,
  });
  opened = editor;

  const dom = root.querySelector('.ProseMirror');
  if (!(dom instanceof HTMLElement)) throw new Error('没有找到 ProseMirror 视图');
  return { editor, root, dom, reports, asked };
}

/** 等异步解析 + 一帧（NodeView 的加载是 Promise，不经过 rAF，但要等微任务）。 */
async function settle(): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, 0));
  await new Promise((resolve) => setTimeout(resolve, 0));
}

/**
 * 等到 ProseMirror 的 `capturePaste` 兜底路径跑完 —— **断言「我们没拦」的用例必须用它**。
 *
 * ## 为什么不能只 `settle()`
 *
 * 我们的 `handlePaste` 返回 `false` 时（不是图片 / 外壳没提供能力），
 * `prosemirror-view` 的 `editHandlers.paste` 不会走默认解析，而是把 DOM 塞进一个
 * 隐藏元素，**50ms 之后**再把它的内容读回来重新 `doPaste` 一次（`capturePaste`）。
 *
 * 那第二次 `doPaste` 会挨个问所有 `handlePaste`，其中包含
 * `@milkdown/plugin-clipboard` 的那一个 —— 它的**第一行**就是 `ctx.get(parserCtx)`。
 * 编辑器此时已经销毁，于是抛 `MilkdownError: Context "parser" not found`。
 *
 * vitest 会把它单独列成「Unhandled Errors」并提示 *This might cause false positive
 * tests* —— 一条未捕获异常会掩盖真正的失败，所以不能放着不管。
 *
 * 等到定时器跑完，那第二次 `doPaste` 就在编辑器**还活着**的时候发生：
 * 我们返回 `false`、clipboard 插件也返回 `false`（假剪贴板没有文本），
 * 于是它什么都不做 —— 这正是这两条用例想钉的「没被拦下来」。
 */
async function settleFallbackPaste(): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, 80));
}

/** 真正的图片元素。ProseMirror 会在叶子节点两侧插 `<img class="ProseMirror-separator">`。 */
function imageEl(harness: Harness): HTMLImageElement | null {
  return harness.dom.querySelector<HTMLImageElement>('img.qm-img-el');
}

/** 在编辑器上派发一次带图片的粘贴。 */
function pasteFiles(dom: HTMLElement, files: readonly File[]): void {
  const data = new DataTransfer();
  for (const file of files) data.items.add(file);
  dom.dispatchEvent(
    new ClipboardEvent('paste', { clipboardData: data, bubbles: true, cancelable: true }),
  );
}

function pngFile(name = 'shot.png'): File {
  return new File([new Uint8Array([0x89, 0x50, 0x4e, 0x47])], name, { type: 'image/png' });
}

describe('图片的显示', () => {
  it('把文档里的 src 换成外壳给的 URL', async () => {
    const harness = await open('![说明](./a.assets/x.png)\n', {
      resolveImageUrl: async (src) => `data:image/png;base64,${src.length}`,
    });
    await settle();

    expect(harness.asked).toEqual(['./a.assets/x.png']);
    // 外壳返回什么就显示什么（这里用长度当指纹，避免把 base64 硬编码进断言）
    expect(imageEl(harness)?.getAttribute('src')).toBe('data:image/png;base64,16');
    expect(harness.dom.querySelector('.qm-img')?.className).toContain('is-loaded');
  });

  it('解析不出来时画一个**带路径**的占位，而不是留一个破图', async () => {
    const harness = await open('![说明](./missing.png)\n', {
      resolveImageUrl: async () => null,
    });
    await settle();

    const box = harness.dom.querySelector('.qm-img');
    expect(box?.className).toContain('is-failed');
    expect(box?.querySelector('.qm-img-note')?.textContent).toContain('./missing.png');
    expect(imageEl(harness)?.hasAttribute('src')).toBe(false);
  });

  it('外壳没有这个能力时也画占位，而不是抛错', async () => {
    const harness = await open('![说明](./x.png)\n');
    await settle();

    expect(harness.dom.querySelector('.qm-img')?.className).toContain('is-failed');
  });

  it('src 为空时不去问外壳', async () => {
    const harness = await open('![]()\n', { resolveImageUrl: async () => 'data:x' });
    await settle();

    expect(harness.asked).toEqual([]);
    expect(harness.dom.querySelector('.qm-img')?.className).toContain('is-failed');
  });

  it('alt 与 title 是属性，直接反映在 DOM 上', async () => {
    const harness = await open('![替代](./x.png "标题")\n', { resolveImageUrl: async () => 'data:x' });
    await settle();

    expect(imageEl(harness)?.getAttribute('alt')).toBe('替代');
    expect(imageEl(harness)?.getAttribute('title')).toBe('标题');
  });

  it('**解析出来的 data URI 绝不进文档** —— 存出来的还是相对路径', async () => {
    const harness = await open('![说明](./a.assets/x.png)\n', {
      resolveImageUrl: async () => 'data:image/png;base64,AAAAAAAAAAAAAAAA',
    });
    await settle();

    // 界面上是 data URI
    expect(imageEl(harness)?.getAttribute('src')).toContain('data:');
    // 文档里必须还是原来那一串
    expect(harness.editor.getMarkdown()).toBe('![说明](./a.assets/x.png)\n');
    expect(harness.editor.getMarkdown()).not.toContain('base64');
  });

  it('改路径会重新解析，且**过期的结果不会盖上来**', async () => {
    const pending: Array<{ src: string; resolve: (url: string | null) => void }> = [];
    const harness = await open('![a](./a.png)\n', {
      resolveImageUrl: (src) =>
        new Promise<string | null>((resolve) => {
          pending.push({ src, resolve });
        }),
    });

    expect(pending.map((p) => p.src)).toEqual(['./a.png']);

    // 换成另一张图：NodeView 要么被 update（同一个视图）、要么被重建，
    // 两条路都必须丢掉旧请求的结果
    harness.editor.setMarkdown('![a](./b.png)\n');
    await settle();

    // 先让**旧**请求返回，再让新的返回 —— 顺序反过来才对得起来
    pending[0]?.resolve('data:old');
    await settle();
    pending[1]?.resolve('data:new');
    await settle();

    expect(imageEl(harness)?.getAttribute('src')).toBe('data:new');
  });
});

describe('粘贴图片', () => {
  it('交出字节、拿回路径、插进文档', async () => {
    const seen: ImagePasteRequest[] = [];
    const harness = await open('前文\n', {
      onImagePaste: async (request) => {
        seen.push(request);
        return './a.assets/image-1.png';
      },
    });

    pasteFiles(harness.dom, [pngFile()]);
    await settle();

    expect(seen).toHaveLength(1);
    expect(seen[0]?.mimeType).toBe('image/png');
    expect(seen[0]?.bytes.byteLength).toBe(4);
    expect(harness.editor.getMarkdown()).toContain('![](./a.assets/image-1.png)');
  });

  it('**粘贴要让文档变脏** —— 否则按 Ctrl+S 会丢掉刚粘的图', async () => {
    const harness = await open('前文\n', {
      onImagePaste: async () => './a.assets/x.png',
    });

    pasteFiles(harness.dom, [pngFile()]);
    await settle();

    // 这条断言才是这个功能的重点：异步落盘之后，变更必须被上报
    expect(harness.reports).toHaveLength(1);
    expect(harness.reports[0]).toContain('./a.assets/x.png');
  });

  it('外壳放弃时什么都不插、也不报错', async () => {
    const harness = await open('前文\n', { onImagePaste: async () => null });

    pasteFiles(harness.dom, [pngFile()]);
    await settle();

    expect(harness.editor.getMarkdown()).toBe('前文\n');
    expect(harness.reports).toEqual([]);
  });

  it('一张失败不影响另一张', async () => {
    let call = 0;
    const harness = await open('前文\n', {
      onImagePaste: async () => {
        call += 1;
        return call === 1 ? null : './a.assets/ok.png';
      },
    });

    pasteFiles(harness.dom, [pngFile('a.png'), pngFile('b.png')]);
    await settle();

    expect(call).toBe(2);
    expect(harness.editor.getMarkdown()).toContain('./a.assets/ok.png');
  });

  it('外壳抛错时跳过这一张，不让整次粘贴崩掉', async () => {
    const harness = await open('前文\n', {
      onImagePaste: async () => {
        throw new Error('磁盘满了');
      },
    });

    pasteFiles(harness.dom, [pngFile()]);
    await settle();

    expect(harness.editor.getMarkdown()).toBe('前文\n');
  });

  it('多张图按顺序都插进去', async () => {
    const harness = await open('前文\n', {
      onImagePaste: async (request) =>
        request.mimeType === 'image/png' ? './a.assets/1.png' : './a.assets/2.jpg',
    });

    const jpg = new File([new Uint8Array([0xff, 0xd8])], 'b.jpg', { type: 'image/jpeg' });
    pasteFiles(harness.dom, [pngFile(), jpg]);
    await settle();

    const markdown = harness.editor.getMarkdown();
    expect(markdown.indexOf('./a.assets/1.png')).toBeGreaterThanOrEqual(0);
    expect(markdown.indexOf('./a.assets/2.jpg')).toBeGreaterThan(markdown.indexOf('./a.assets/1.png'));
  });

  it('不是图片的粘贴不拦 —— 普通文本要走默认处理', async () => {
    let called = 0;
    const harness = await open('前文\n', {
      onImagePaste: async () => {
        called += 1;
        return './a.assets/x.png';
      },
    });

    const txt = new File([new Uint8Array([0x61])], 'a.txt', { type: 'text/plain' });
    pasteFiles(harness.dom, [txt]);
    await settleFallbackPaste();

    expect(called).toBe(0);
    expect(harness.editor.getMarkdown()).toBe('前文\n');
  });

  it('外壳没提供这个能力时不拦', async () => {
    const harness = await open('前文\n');

    pasteFiles(harness.dom, [pngFile()]);
    await settleFallbackPaste();

    expect(harness.editor.getMarkdown()).toBe('前文\n');
    expect(harness.reports).toEqual([]);
  });

  it('粘贴的图不带 alt（内核不替用户编一个）', async () => {
    const harness = await open('前文\n', { onImagePaste: async () => './a.assets/x.png' });

    pasteFiles(harness.dom, [pngFile()]);
    await settle();

    expect(harness.editor.getMarkdown()).toContain('![](./a.assets/x.png)');
  });
});
