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

  const parts = [`已导出 ${basename(target)}`];
  if (report.highlightedBlocks > 0) parts.push(`${report.highlightedBlocks} 个代码块已着色`);
  if (report.inlinedImages > 0) parts.push(`${report.inlinedImages} 张图片已内联`);
  if (report.missingImages > 0) parts.push(`${report.missingImages} 张图片读不到，只保留了原链接`);
  if (report.remoteImages > 0) parts.push(`${report.remoteImages} 张图片来自网络，打开时需要联网`);
  if (request.bodyHtml === null) parts.push('当前内核没有富文本结构，已按纯文本呈现');

  return { status: 'saved', target, message: parts.join('；') };
}

/** 导出建议的文件名（供 UI 预览、也供测试断言）。 */
export function suggestedExportName(path: string): string {
  return `${titleOf(path)}.html`;
}

function describe(thrown: unknown): string {
  return thrown instanceof Error ? thrown.message : String(thrown);
}
