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
  const report: InlineReport = { inlinedImages: 0, missingImages: 0, remoteImages: 0, highlightedBlocks: 0 };

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
