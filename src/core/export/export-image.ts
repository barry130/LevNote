/**
 * 把内核画的图存成文件（ADR 0050）。
 *
 * 与 `export-document.ts` 是**并列**的两条导出路径，不是它的分支：
 * 那边走文本管线（内核给语义 HTML，外壳拼自包含网页），这边走二进制管线
 * （内核把**自己的视图**画成 PNG / SVG，外壳只管问存到哪、然后写盘）。
 * 画法长在内核里，外壳既不该也不懂怎么复现 —— 所以这里从头到尾
 * 没有出现任何导图专有的东西，将来 diagram / drawio 内核接上就能用同一条路。
 *
 * 编排顺序是「**先问存到哪，再画**」：画一张大图是要花时间的（PNG 还要
 * 位图编码），用户在对话框上点取消的话，这笔时间不该白花。
 *
 * 保持「如实通报」（ADR 0001）：三分支返回值与 HTML 导出同形 ——
 * `saved` 带上内核的实话（`warnings`），`cancelled` 不弹假成功，
 * `failed` 把原因原样带出去，一个都不吞。
 */

import type { DocumentProvider } from '../document/types.js';
import type { EditorImageExport } from '../editor/types.js';
import { t, textOf } from '../i18n/index.js';
import { basename } from '../path.js';

export type ImageExportFormat = 'png' | 'svg';

export interface ExportImageRequest {
  /** 文档标题（不带扩展名），用来拼建议文件名。 */
  readonly title: string;
  readonly format: ImageExportFormat;
  /** 内核画图。可能抛；返回 null 表示这一次没能画出来（按失败如实报）。 */
  readonly draw: () => Promise<EditorImageExport | null>;
  readonly provider: DocumentProvider;
}

export type ExportImageResult =
  | { readonly status: 'saved'; readonly target: string; readonly message: string }
  | { readonly status: 'cancelled' }
  | { readonly status: 'failed'; readonly message: string };

export async function exportImage(request: ExportImageRequest): Promise<ExportImageResult> {
  const { provider } = request;

  let target: string | null;
  try {
    target = await provider.pickSavePath(`${request.title}.${request.format}`);
  } catch (thrown) {
    return { status: 'failed', message: t('core.export.dialogFailed', { reason: describe(thrown) }) };
  }
  // 用户取消：什么都不画、不写，也不弹假成功
  if (target === null) return { status: 'cancelled' };

  let image: EditorImageExport | null;
  try {
    image = await request.draw();
  } catch (thrown) {
    return { status: 'failed', message: t('core.export.imageBuildFailed', { reason: describe(thrown) }) };
  }
  // 内核说画不出来 —— 这是失败，不是「导出了一张空图」
  if (image === null) return { status: 'failed', message: t('core.export.imageDrawFailed') };

  const bytes = await image.blob.arrayBuffer();
  let outcome: Awaited<ReturnType<DocumentProvider['writeBinaryFile']>>;
  try {
    outcome = await provider.writeBinaryFile(target, bytes);
  } catch (thrown) {
    // 导出家族统一**不覆盖已有文件**：writeBinaryFile 是 create_new 语义，
    // 目标已存在会以 CONFLICT 原文失败 —— 与 HTML 导出同一条纪律，如实转达即可。
    return { status: 'failed', message: t('core.export.writeFailed', { reason: describe(thrown) }) };
  }
  // 半成品必须说破（与 `core/document/images.ts` 的写盘纪律同一条）：
  // 少写一个字节，图就是坏的，静默交「成功」等于把坏文件发给用户
  if (outcome.size !== bytes.byteLength) {
    return {
      status: 'failed',
      message: t('core.images.partialWrite', { size: outcome.size, total: bytes.byteLength, path: outcome.path }),
    };
  }

  const parts = [t('core.export.exported', { name: basename(target) })];
  // 内核的实话一条不少地拼在后面 —— 吞掉任何一条都是「静默」
  for (const warning of image.warnings) parts.push(textOf(warning));
  return { status: 'saved', target, message: parts.join('；') };
}

function describe(thrown: unknown): string {
  return thrown instanceof Error ? thrown.message : String(thrown);
}
