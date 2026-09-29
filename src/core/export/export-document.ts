/**
 * 导出动作本身。
 *
 * 这段逻辑原先长在 `App.tsx` 的 `exportHtml` 里，与 React 状态纠缠在一起。
 * 抽出来有两个实在的好处：
 * - 命令面板、工具栏按钮、将来的「导出全部标签页」共用同一条路径，
 *   不会出现「按钮上的导出」和「命令面板里的导出」行为不一致；
 * - 它可以被单测直接调用，不必先渲染一个外壳。
 *
 * 保持「**如实通报**」这条原则（见 ADR 0001）：返回值里带着做了什么、有什么没做到，
 * 由调用方原样呈现，而不是在这里编一句「导出成功」。
 */

import type { DocumentProvider } from '../document/types.js';
import { basename, dirname, titleOf } from '../path.js';
import type { ExportReport } from './document.js';
import { buildSelfContainedHtml } from './document.js';

export interface ExportHtmlRequest {
  /** 源文档的完整路径 */
  readonly path: string;
  /** 文档标题（不带扩展名） */
  readonly title: string;
  /** 内核给出的语义 HTML 片段；null 表示内核不支持富文本 */
  readonly bodyHtml: string | null;
  /** `bodyHtml` 为 null 时用来呈现的原文 */
  readonly fallbackText: string;
  readonly provider: DocumentProvider;
}

export type ExportHtmlResult =
  | { readonly status: 'saved'; readonly target: string; readonly message: string }
  | { readonly status: 'cancelled' }
  | { readonly status: 'failed'; readonly message: string };

export async function exportHtml(request: ExportHtmlRequest): Promise<ExportHtmlResult> {
  const { provider } = request;

  let html: string;
  let report: Awaited<ReturnType<typeof buildSelfContainedHtml>>['report'];
  try {
    const built = await buildSelfContainedHtml({
      title: request.title,
      bodyHtml: request.bodyHtml,
      fallbackText: request.fallbackText,
      baseDir: dirname(request.path),
      readBinaryFile: (path) => provider.readBinaryFile(path),
    });
    html = built.html;
    report = built.report;
  } catch (thrown) {
    return { status: 'failed', message: `生成 HTML 失败：${describe(thrown)}` };
  }

  const suggested = `${request.title}.html`;
  let target: string | null;
  try {
    target = await provider.pickSavePath(suggested);
  } catch (thrown) {
    return { status: 'failed', message: `无法弹出保存对话框：${describe(thrown)}` };
  }
  // 用户取消：什么都不写，也不弹假成功
  if (target === null) return { status: 'cancelled' };

  try {
    await provider.writeTextFile({
      path: target,
      text: html,
      // 导出文件固定 UTF-8 无 BOM + LF：这是一个要发给别人、可能在任意系统打开的网页，
      // 跟随源文档的 GBK / CRLF 只会让它在别人机器上变成乱码
      encoding: 'UTF-8',
      bom: 'none',
      lineEnding: 'lf',
      create: true,
    });
  } catch (thrown) {
    return { status: 'failed', message: `写入失败：${describe(thrown)}` };
  }

  const message = describeExport({
    target,
    report,
    plainFallback: request.bodyHtml === null,
  });

  return { status: 'saved', target, message };
}

export interface ExportSummaryInput {
  /** 导出到的完整路径；提示里只用它的文件名。 */
  readonly target: string;
  readonly report: ExportReport;
  /** 内核没有富文本结构，已按纯文本呈现。 */
  readonly plainFallback: boolean;
}

/**
 * 把一次导出的结果拼成给用户看的一句话。
 *
 * **单独成函数**是因为它有一堆分支，而每条分支都对应一件用户需要知道的事。
 * 混在 `exportHtml` 里就只能靠「真的导出一次带公式、带图、带代码的文档」
 * 去覆盖 —— 而那种测试同时也在测读盘、写盘、弹对话框，失败时不知道是谁的错。
 * 抽出来之后，这里是一个纯函数：给一份报告，给一句话。
 *
 * 顺序是按「用户最关心什么」排的：先说自己要的那件事成了（导出到了哪），
 * 再说内容里有什么被处理过，最后说有什么**没做到**。
 */
export function describeExport(input: ExportSummaryInput): string {
  const { report } = input;
  const parts = [`已导出 ${basename(input.target)}`];

  if (report.highlightedBlocks > 0) parts.push(`${report.highlightedBlocks} 个代码块已着色`);
  if (report.inlinedImages > 0) parts.push(`${report.inlinedImages} 张图片已内联`);
  if (report.missingImages > 0) parts.push(`${report.missingImages} 张图片读不到，只保留了原链接`);
  if (report.remoteImages > 0) parts.push(`${report.remoteImages} 张图片来自网络，打开时需要联网`);

  /**
   * 公式。四条消息各有各的用处：
   *
   * - 渲染成功数 —— 让用户确认「我的公式都进去了」；
   * - 字体大小 —— 一份带公式的文档会比同样长度的文档大 300 KB 上下，
   *   不说清楚的话用户只会觉得「这个导出是不是有 bug」。这是他该知道的
   *   **代价**，而不是我们该藏的细节；
   * - 失败与缺失 —— 这两条是**必须**说出来的（本项目「绝不静默」的红线）：
   *   字体没内联意味着发出去的文件在别人机器上会掉字形，
   *   公式没渲染出来意味着那段内容只剩源码。
   */
  if (report.renderedMath > 0) parts.push(`${report.renderedMath} 个公式已渲染`);
  if (report.failedMath > 0) {
    const why = report.firstMathError === null ? '' : `（${report.firstMathError}）`;
    parts.push(`${report.failedMath} 个公式渲染失败，只保留了源码${why}`);
  }
  if (report.mathFonts > 0) {
    const kb = Math.round(report.mathStyleBytes / 1024);
    parts.push(`公式字体已内联（${report.mathFonts} 个文件，约 ${kb} KB）`);
  }
  if (report.missingMathFonts > 0) {
    parts.push(`${report.missingMathFonts} 个公式字体没能内联，导出的公式可能用系统字体显示`);
  }

  if (input.plainFallback) parts.push('当前内核没有富文本结构，已按纯文本呈现');

  return parts.join('；');
}

/** 导出建议的文件名（供 UI 预览、也供测试断言）。 */
export function suggestedExportName(path: string): string {
  return `${titleOf(path)}.html`;
}

function describe(thrown: unknown): string {
  return thrown instanceof Error ? thrown.message : String(thrown);
}
