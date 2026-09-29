/**
 * 导出前的内容加工：语法高亮 + 图片内联。
 *
 * 这一步在**已生成的语义 HTML**（内核 `toHtml()` 的产物）上原地做，而不是重新遍历文档树：
 * 内核已经按 schema 把结构转对了，这里只补两件内核不负责的事，
 * 于是「编辑器所见」与「导出所得」的结构来自同一处，不会漂移。
 *
 * 为什么这两件事内核不负责：
 * - **高亮**：ProseMirror 的高亮是*装饰*（Decoration），装饰只存在于视图层，
 *   `DOMSerializer` 序列化文档树时不会带上它们。所以导出侧必须自己再走一遍，
 *   用的是同一份 `highlightCode`（见 `core/highlight/hljs.ts`）—— 同源，观感才一致。
 * - **图片**：编辑时 `<img src="./a.png">` 由 WebView 自己解析；
 *   导出成单文件后没人给这个相对路径做解析，必须把字节读进来变成 data URI。
 */

import { highlightCode } from '../highlight/hljs.js';
import { bytesToDataUrl, sniffImageMime } from '../image.js';
import { renderTex } from '../katex.js';
import { MATH_BLOCK, TEX_DATA_ATTR } from '../math.js';

export interface InlineIo {
  /** 文档所在目录。null = 拿不到（例如浏览器预览的虚拟盘），此时只处理绝对路径的图片。 */
  baseDir: string | null;
  /**
   * 按字节读文件。失败时抛错，由这里降级处理。
   * 省略表示这个平台没有二进制读取能力 —— 图片会全部标注为「未内联」，
   * 但代码高亮照常进行（它不需要读盘）。
   */
  readBinaryFile?: (path: string) => Promise<ArrayBuffer>;
}

export interface InlineReport {
  /** 成功内联的图片数 */
  inlinedImages: number;
  /** 读不到字节、保留原链接并标注的图片数 */
  missingImages: number;
  /** 指向 http(s)、无法保证离线的图片数 */
  remoteImages: number;
  /** 实际做了高亮的代码块数 */
  highlightedBlocks: number;
  /** 实际渲染成 KaTeX 的公式数 */
  renderedMath: number;
  /** 渲染失败、只留下源码的公式数 */
  failedMath: number;
  /**
   * 第一条公式错误的原因，供导出提示直接显示。
   *
   * 只留第一条：一份文档里十个公式都错，提示里堆十句话没人看。用户看到
   * 其中一条就知道自己写错了什么，回去改就是了。
   */
  firstMathError: string | null;
}

/**
 * 拼接目录与相对路径，保留原路径的分隔符风格（Windows 上是反斜杠）。
 *
 * **它与 `core/path.ts` 的 `joinPath` 不是同一个东西，所以名字也不同。**
 * 两者都吃掉 `./` 与开头的分隔符，但这一份**还多收敛一层重复分隔符**
 * （`a//b.png` → `a/b.png`），而 `path.ts` 那份只做拼接、一个字符都不多改 ——
 * 它服务的是 UI（显示、拼接），不该悄悄改写用户写在文档里的路径。
 *
 * 导出这条路径上做规范化是合适的：拼出来的字符串马上要交给文件系统去读。
 * 但**不要**把两份合并成一个 —— 合并意味着其中一边要接受自己不需要的行为。
 */
function normalizeJoin(baseDir: string, relative: string): string {
  const separator = baseDir.includes('\\') ? '\\' : '/';
  const trimmed = baseDir.replace(/[\\/]+$/, '');
  const parts = relative.replace(/^\.?[\\/]+/, '').split(/[\\/]/).filter((part) => part !== '');
  return [trimmed, ...parts].join(separator);
}

function isAbsolutePath(src: string): boolean {
  return /^[A-Za-z]:[\\/]/.test(src) || src.startsWith('/') || src.startsWith('\\\\');
}

/**
 * 把 <img src> 变成 data URI，给代码块上色。
 *
 * 原地修改 `root` 的子树。任何一张图片失败都**不中断导出** ——
 * 一张图读不到不该让整篇文档导不出来，标注出来让用户自己决定。
 */
export async function inlineAssets(root: HTMLElement, io: InlineIo): Promise<InlineReport> {
  const report: InlineReport = {
    inlinedImages: 0,
    missingImages: 0,
    remoteImages: 0,
    highlightedBlocks: 0,
    renderedMath: 0,
    failedMath: 0,
    firstMathError: null,
  };

  // ---- 代码高亮 ----------------------------------------------------------

  for (const pre of root.querySelectorAll('pre')) {
    const code = pre.querySelector('code');
    if (!code) continue;
    const source = code.textContent ?? '';
    if (source.trim() === '') continue;

    // 语言挂在 pre 的 data-language 上（preset-commonmark 的 toDOM 就是这么写的）
    const declared = pre.getAttribute('data-language') ?? '';
    const result = highlightCode(source, declared);
    if (!result) continue;

    code.innerHTML = result.html;
    report.highlightedBlocks += 1;

    // 自动识别出的语言写回去，右上角的角标才不会跟实际着色不一致
    if (result.language !== '' && declared === '') pre.setAttribute('data-language', result.language);
  }

  // ---- 图片内联 ----------------------------------------------------------

  for (const img of root.querySelectorAll('img')) {
    const src = img.getAttribute('src') ?? '';
    if (src === '') continue;

    if (src.startsWith('data:')) {
      report.inlinedImages += 1;
      continue;
    }
    if (/^[a-z][a-z0-9+.-]*:/i.test(src)) {
      // http(s)、file: 等带协议的引用：内联需要联网或读盘越界，
      // 保持原样但记一笔，好让导出提示如实说明「这份文件不是完全自包含的」
      report.remoteImages += 1;
      continue;
    }
    if (src.startsWith('#')) continue;

    let resolved: string | null = null;
    if (isAbsolutePath(src)) {
      resolved = src;
    } else if (io.baseDir !== null) {
      resolved = normalizeJoin(io.baseDir, decodeURIComponent(src));
    }

    if (resolved === null || !io.readBinaryFile) {
      img.setAttribute('data-qm-missing', '1');
      report.missingImages += 1;
      continue;
    }

    try {
      const readBinaryFile = io.readBinaryFile;
      const bytes = await readBinaryFile(resolved);
      const mime = sniffImageMime(new Uint8Array(bytes), resolved);
      /**
       * SVG 也用 base64 而不是 `data:image/svg+xml,<svg…>`：
       * SVG 里可以带 `<script>`，放进 `<img>` 里虽然不会执行，
       * 但以明文形式躺在 HTML 里会让人以为它执行了。base64 让「这是一份不透明的资源」
       * 一目了然，也避免把 `#` `&` 这些字符带进 HTML 属性。
       */
      img.setAttribute('src', bytesToDataUrl(bytes, mime));
      report.inlinedImages += 1;
    } catch {
      // 读不到：保留原链接并打标记，由 CSS 画出可见的占位（见 document-styles.css）
      img.setAttribute('data-qm-missing', '1');
      report.missingImages += 1;
    }
  }

  return report;
}

/**
 * 把文档里的公式渲染成 KaTeX 的 HTML，原地替换掉 `toDOM` 留下的壳。
 *
 * ## 为什么不能由内核的 `toDOM` 直接渲染
 *
 * `toDOM` 是**同步**的，而 KaTeX 是动态加载的（见 `core/katex.ts`）。
 * 让 `toDOM` 能渲染，KaTeX 就只能静态引入 —— 272 KB 进主包，
 * 每个打开轻墨的人都要先下载它。而且 `toDOM` 里抛异常会把编辑器带崩，
 * 而非法公式**必然**抛。
 *
 * 于是分工是：内核写一个带 `data-tex` 的壳（内容是公式源码），
 * 导出侧在这里把它换成渲染结果。壳里的源码不是废料 —— 渲染失败时
 * 它就是用户唯一能看见的东西（见下）。
 *
 * ## 渲染失败要留下源码
 *
 * 一份导出文件是**发出去**的。某个公式渲染失败时，留一个空壳等于
 * 「这段内容凭空消失」，而留下 `$x^2$` 至少让人看得出原本写了什么、
 * 也看得出是哪里写错了。同时 `report` 里记一笔，导出提示会说出来。
 *
 * 与图片一样：**任何一处失败都不中断导出**。一个写错的公式不该让整篇文档
 * 导不出来。
 */
export async function renderMath(root: HTMLElement, report: InlineReport): Promise<void> {
  const targets = Array.from(root.querySelectorAll<HTMLElement>(`[${TEX_DATA_ATTR}]`));

  for (const element of targets) {
    const tex = element.getAttribute(TEX_DATA_ATTR) ?? '';
    const displayMode = element.dataset['type'] === MATH_BLOCK;
    const result = await renderTex(tex, displayMode);

    if (result.ok) {
      element.classList.remove('is-loading', 'is-empty', 'is-failed');
      element.classList.add('is-loaded');
      element.innerHTML = result.html;
      report.renderedMath += 1;
      continue;
    }

    // 失败：把壳里那段源码**显式**变成两个元素，样式与编辑器里一致
    element.classList.remove('is-loading', 'is-empty', 'is-loaded');
    element.classList.add('is-failed');
    element.replaceChildren(
      elementWithClass('qm-math-src', displayMode ? `$$\n${tex}\n$$` : `$${tex}$`),
      elementWithClass('qm-math-note', result.message),
    );
    report.failedMath += 1;
    report.firstMathError ??= result.message;
  }
}

function elementWithClass(className: string, text: string): HTMLElement {
  const element = document.createElement('span');
  element.className = className;
  element.textContent = text;
  return element;
}
