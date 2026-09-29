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

import { inlineAssets, type InlineReport } from './inline-html.js';

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
}

export interface SelfContainedExport {
  html: string;
  report: InlineReport;
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

  const report = await inlineAssets(root, {
    baseDir: options.baseDir,
    readBinaryFile: options.readBinaryFile,
  });

  // innerHTML 会丢掉根元素的 class，所以取它的内容再套一层
  const body = root.innerHTML;
  const title = escapeHtml(options.title);

  const html = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="generator" content="轻墨 (qingmo)">
<title>${title}</title>
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
