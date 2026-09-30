/**
 * 自包含 HTML 导出。
 *
 * 「自包含」在这里是有明确验收标准的，不是形容词：
 * - 把文件拷到任意一台机器、断网、双击打开，**观感与编辑器里一致**；
 * - 不需要我们的样式表、不需要字体文件、不能因为图片路径失效而出现破图；
 * - 不依赖任何 `file://` 之外的资源，不加载外部 JS。
 *
 * 它承担了「桌面端没有分享链接」之后「把成果发给别人」的角色，属于一等公民
 * （见 ADR 0008 与 `src-tauri/src/export/mod.rs` 的路线说明）。
 *
 * 实现上刻意**不重新遍历文档树**：结构由内核的 `toHtml()` 给出（与编辑器同源），
 * 这里只补内核不负责的两件事（高亮、图片内联，见 `inline-html.ts`），
 * 再把版式内联进去。三条来源各管一段，互不重叠。
 */

import { inlineAssets, renderDiagrams, renderMath, type InlineReport } from './inline-html.js';
import { buildMathStyles, type MathStyleIo } from './katex-css.js';

import documentStyles from './document-styles.css?raw';

export interface SelfContainedExportOptions {
  /** 文档标题，写进 `<title>`。 */
  title: string;
  /**
   * 内核产出的**语义 HTML 片段**（不含 `<html>` / `<head>`）。
   * `null` 是合法输入：表示这个内核没有富文本语义可导出（例如纯文本内核），
   * 此时如实退化为 `<pre>` 呈现原文，而不是伪造一份富文本。
   */
  bodyHtml: string | null;
  /** `bodyHtml` 为 null 时用来呈现的原文。 */
  fallbackText: string;
  /** 文档所在目录，用于把相对图片路径解析成可读路径。 */
  baseDir: string | null;
  /** 按字节读文件的能力。省略表示平台不支持，图片会标注为未内联。 */
  readBinaryFile?: (path: string) => Promise<ArrayBuffer>;
  /**
   * 取 KaTeX 的 CSS 与字体。省略表示用真实实现（打包产物里的那份）。
   *
   * 做成可注入的只有一个理由：**测试不该依赖打包产物**。真实实现走
   * `import.meta.glob` + `fetch`，那两样在单测环境里没有意义，
   * 而「内联后的 CSS 长什么样」这件事必须被钉住。
   */
  mathIo?: MathStyleIo;
}

export interface ExportReport extends InlineReport {
  /**
   * 内联进导出文件的公式字体文件数。0 表示文档里没有公式。
   *
   * 三个字段都不是 `readonly`：它们是在「确认文档里真的有公式」之后
   * 才被填上的，而那个判断要在 `inlineAssets` 跑完之后才做得了。
   */
  mathFonts: number;
  /** 没能内联的公式字体文件数。大于 0 时导出提示必须说出来。 */
  missingMathFonts: number;
  /** 内联进去的公式样式（含字体）有多少字节。用来如实告诉用户文件为什么变大了。 */
  mathStyleBytes: number;
}

export interface SelfContainedExport {
  html: string;
  report: ExportReport;
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

/**
 * 导出页面的外壳样式。
 *
 * 与 `document-styles.css` 分开：那一份是「文档长什么样」（编辑器也用），
 * 这一份是「一张纸长什么样」（只有导出的 HTML 才有纸）。混在一起会让编辑器
 * 莫名其妙多出页面内边距和投影。
 */
const PAGE_STYLES = `
*, *::before, *::after { box-sizing: border-box; }
html { -webkit-text-size-adjust: 100%; }
body {
  margin: 0;
  padding: 40px 16px 80px;
  background: #eef3f0;
  color: #16302a;
  font-family: 'Segoe UI', system-ui, -apple-system, 'Microsoft YaHei', 'PingFang SC', sans-serif;
  line-height: 1.75;
}
.qm-page {
  max-width: 820px;
  margin: 0 auto;
  padding: 48px 56px 64px;
  background: #ffffff;
  border: 1px solid #dfe6e2;
  border-radius: 10px;
  box-shadow: 0 1px 2px rgba(22, 48, 42, 0.06), 0 10px 28px rgba(22, 48, 42, 0.05);
}
/* 内核不支持富文本时的兜底呈现。保留原始换行与空白，不假装是排版好的文档。 */
.qm-plain {
  margin: 0;
  font-family: 'Cascadia Mono', 'JetBrains Mono', Consolas, 'Sarasa Mono SC', monospace;
  font-size: 13px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-word;
}
@media (max-width: 720px) {
  body { padding: 0; }
  .qm-page { max-width: none; border: 0; border-radius: 0; box-shadow: none; padding: 24px 18px 48px; }
}
/* 打印/另存为 PDF 时把「纸」去掉，交给打印管线分页 —— 否则会出现图片被截断的空白页 */
@media print {
  body { padding: 0; background: #fff; }
  .qm-page { max-width: none; border: 0; border-radius: 0; box-shadow: none; padding: 0; }
  .qm-doc { font-size: 12pt; }
  .qm-doc pre, .qm-doc blockquote, .qm-doc table, .qm-doc img { break-inside: avoid; }
  .qm-doc h1, .qm-doc h2, .qm-doc h3 { break-after: avoid; }
  @page { margin: 18mm 16mm; }
}
`.trim();

export async function buildSelfContainedHtml(options: SelfContainedExportOptions): Promise<SelfContainedExport> {
  const root = document.createElement('div');
  root.className = 'qm-doc';

  if (options.bodyHtml !== null) {
    root.innerHTML = options.bodyHtml;
  } else {
    const pre = document.createElement('pre');
    pre.className = 'qm-plain';
    // 用 textContent 而不是 innerHTML：原文里可能有 `<`，那也是内容的一部分
    pre.textContent = options.fallbackText;
    root.appendChild(pre);
  }

  const report: ExportReport = {
    ...(await inlineAssets(root, {
      baseDir: options.baseDir,
      readBinaryFile: options.readBinaryFile,
    })),
    mathFonts: 0,
    missingMathFonts: 0,
    mathStyleBytes: 0,
  };

  /**
   * 公式：渲染 + 内联字体。
   *
   * **先看有没有公式再决定要不要做**。带上字体的 KaTeX 样式有 300 KB 上下，
   * 一篇没有公式的文档一个字节都不该多出来 —— 所以这里用
   * `querySelector` 探一下，而不是无条件地拼一份样式出来。
   *
   * 探测放在**渲染之前**：`renderMath` 会把壳的内容换掉，但 `data-tex`
   * 属性仍在，所以先后都探得到；放在前面是为了「没有公式时连渲染都不做」。
   */
  const hasMath = root.querySelector('[data-tex]') !== null;
  let mathCss = '';

  if (hasMath) {
    await renderMath(root, report);

    const styles = await buildMathStyles(options.mathIo);
    mathCss = styles.css;
    report.mathFonts = styles.inlinedFonts;
    report.missingMathFonts = styles.missingFonts;
    report.mathStyleBytes = byteLength(styles.css);
  }

  /**
   * 图表（Mermaid）：画成 SVG 内联进去。
   *
   * ## 为什么明暗**写死 `'light'`**
   *
   * 导出产物是一张**纸**：上面 `PAGE_STYLES` 把底色写死成浅色，它没有深色模式。
   * 所以哪怕用户此刻用的是深色主题，图也必须按浅色画 —— 否则会出现
   * 「浅色的纸上一张深色底的图」。这与 `Lightbox.tsx` 那条「灯箱恒深」
   * 是同一类判断：**容器决定配色，而不是应用当前的主题决定配色**。
   *
   * ## 为什么不需要像公式那样先探一下
   *
   * 公式要探是因为「带上字体的 KaTeX 样式」有 300 KB，没有公式就不该多一个字节。
   * 图表没有这层代价：`renderDiagrams` 里那个筛选为空时，一次 `renderMermaid`
   * 都不会发生，**mermaid 那一堆 chunk 也就不会被下载**（它是动态加载的，
   * 见 `core/mermaid-render.ts`）。所以「有没有图表」这件事由它自己判断就够了。
   */
  await renderDiagrams(root, report, 'light');

  // innerHTML 会丢掉根元素的 class，所以取它的内容再套一层
  const body = root.innerHTML;
  const title = escapeHtml(options.title);

  /**
   * 公式样式单独一个 `<style>`，并且排在版式之前。
   *
   * 分开是为了让「这份导出为什么大」一眼可查 —— 版式那份只有几 KB，
   * 剩下三百 KB 全是 KaTeX 的字体。顺序上放在前面，与我们自己的
   * `.qm-doc .qm-math*` 规则形成「先给框架、再给版式」的层次。
   */
  const mathStyleTag =
    mathCss === ''
      ? ''
      : `
<style>
${mathCss}
</style>`;

  const html = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="generator" content="轻墨 (qingmo)">
<title>${title}</title>${mathStyleTag}
<style>
${PAGE_STYLES}
</style>
<style>
${documentStyles}
</style>
</head>
<body>
<main class="qm-page">
<article class="qm-doc">
${body}
</article>
</main>
</body>
</html>
`;

  return { html, report };
}

/**
 * 一段文本按 UTF-8 算有多少字节。
 *
 * 不能直接用 `text.length`：那是**码元**个数，中文一个字算 1，而它在文件里
 * 占 3 个字节。导出的公式样式里 ASCII 占绝大多数，但字体是 base64、
 * 说明文字是中文 —— 报一个偏小的数字，用户会觉得「提示里说 300 KB，
 * 文件怎么大了 400 KB」。`TextEncoder` 在所有目标环境里都有。
 */
function byteLength(text: string): number {
  return new TextEncoder().encode(text).length;
}
