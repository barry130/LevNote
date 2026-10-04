/**
 * 打印 / 导出 PDF 的动作（P3 第 16 项，ADR 0030）。
 *
 * 与 `exportHtml` 的关系是「**共用渲染，分道出口**」：两者都走
 * `buildSelfContainedHtml` 生成同一份自包含 HTML（连 `@media print`
 * 的纸张样式都是那一份），区别只在出口 —— HTML 导出落盘成文件，
 * 这里交给打印窗口弹系统打印对话框（「另存为 PDF」是里面的目的地）。
 * 任何一方改了渲染，另一方自动跟上，不会出现「导出的 HTML 是排好版的、
 * 打印出来却不一样」。
 *
 * `printHtml` 做成注入能力而不是直接 import platform：动作保持与平台无关
 * （与 provider 的注入同一条理由），单测给个假实现就能钉住整条链路。
 */

import type { DocumentProvider } from '../document/types.js';
import { dirname } from '../path.js';
import { buildSelfContainedHtml } from './document.js';

export interface PrintDocumentRequest {
  /** 源文档的完整路径 */
  readonly path: string;
  /** 文档标题（不带扩展名），用在打印窗口标题上 */
  readonly title: string;
  /** 内核给出的语义 HTML 片段；null 表示内核不支持富文本 */
  readonly bodyHtml: string | null;
  /** `bodyHtml` 为 null 时用来呈现的原文 */
  readonly fallbackText: string;
  readonly provider: DocumentProvider;
  /** 打开打印窗口的能力（桌面端走 Rust 命令；测试注入假实现） */
  readonly printHtml: (html: string, title: string) => Promise<void>;
}

export type PrintDocumentResult =
  | { readonly status: 'opened'; readonly message: string }
  | { readonly status: 'failed'; readonly message: string };

function describe(thrown: unknown): string {
  return thrown instanceof Error ? thrown.message : String(thrown);
}

export async function printDocument(request: PrintDocumentRequest): Promise<PrintDocumentResult> {
  let html: string;
  let missingImages: number;
  let remoteImages: number;
  let failedMath: number;
  try {
    const built = await buildSelfContainedHtml({
      title: request.title,
      bodyHtml: request.bodyHtml,
      fallbackText: request.fallbackText,
      baseDir: dirname(request.path),
      readBinaryFile: (path) => request.provider.readBinaryFile(path),
    });
    html = built.html;
    missingImages = built.report.missingImages;
    remoteImages = built.report.remoteImages;
    failedMath = built.report.failedMath;
  } catch (thrown) {
    return { status: 'failed', message: `生成打印内容失败：${describe(thrown)}` };
  }

  try {
    await request.printHtml(html, request.title);
  } catch (thrown) {
    return { status: 'failed', message: `打不开打印窗口：${describe(thrown)}` };
  }

  // 与 HTML 导出同一条「如实通报」：打印里没能内联的东西要说出来（硬约束 2）
  const caveats: string[] = [];
  if (missingImages > 0) caveats.push(`${missingImages} 张图片读不到，打印里只保留原链接`);
  if (remoteImages > 0) caveats.push(`${remoteImages} 张图片是网络链接，离线打印时不会显示`);
  if (failedMath > 0) caveats.push(`${failedMath} 条公式渲染失败，打印里只保留源码`);
  const caveat = caveats.length > 0 ? `（${caveats.join('；')}）` : '';
  return {
    status: 'opened',
    message: `打印预览已打开 —— 在打印对话框里选「另存为 PDF」即可导出${caveat}。`,
  };
}
