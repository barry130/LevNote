/**
 * 引用卡片「预览」展开要的**结构摘要**（P6 第 28 项 L2，ADR 0036 决策 3）。
 *
 * 分工与 `block-ops.ts` 同一条：内核只画（卡片是内核的 DOM），
 * 文件是外壳的领域 —— 这里只认**注入的依赖**，不碰真实 provider，
 * 编排逻辑在单测里跑。
 *
 * 刻意**不渲染全文**：markdown 显示成与编辑器不一样的「半个渲染结果」
 * 比不显示更糟（保真陷阱），结构摘要是如实的话 —— 标题树 + 字数足够
 * 回答用户在展开前真正要问的那句：「这是哪份内容，值得点开吗？」
 */

import type { DocBlockPreview } from '../editor/types.js';
import { t } from '../i18n/index.js';
import { resolveRelative } from '../path.js';
import { extractOutline } from '../outline/outline.js';
import { analyze } from '../statistics.js';
import { parseMindmap } from '../mindmap/parse.js';
import type { MindmapNode } from '../mindmap/types.js';

/** 摘要里标题树的上限。再多就是整份大纲了 —— 摘要只负责「认出文件」。 */
const MAX_HEADINGS = 8;
/** text 摘要片段的行数上限。 */
const SNIPPET_LINES = 40;

export interface BlockPreviewDeps {
  /** 读一个文本文件；读不到（不存在 / 无权限）返回 null，**不抛** —— 与 `BlockOpsDeps` 同一条约定。 */
  readonly readTextFile: (path: string) => Promise<{ text: string } | null>;
}

/**
 * 把一份引用算成结构摘要。
 *
 * 任何失败都以 `kind: 'error'` **作为数据**返回，不抛出 ——
 * 摘要就是要画在卡片里的东西，失败的原因也是摘要的一部分（硬约束 2）。
 */
export async function buildDocBlockPreview(
  deps: BlockPreviewDeps,
  docPath: string,
  ref: { src: string; kind: string },
): Promise<DocBlockPreview> {
  const absolute = resolveRelative(docPath, ref.src);
  if (absolute === null) {
    return { kind: 'error', reason: `解析不了相对路径「${ref.src}」—— 当前文档没有目录信息` };
  }

  const file = await deps.readTextFile(absolute);
  if (file === null) {
    return { kind: 'error', reason: `读不到 ${ref.src}` };
  }
  const text = file.text;

  // kind 收敛到卡片认识的三类。空值按 text 兜底（与卡片徽章同一约定）；
  // 不认识的值也按 text —— 卡片能画出来本身就意味着 kind 已经过那一道门。
  if (ref.kind === 'markdown') {
    return {
      kind: 'markdown',
      wordCount: analyze(text).words,
      headings: extractOutline(text)
        .slice(0, MAX_HEADINGS)
        .map((item) => ({ level: item.level, text: item.text })),
    };
  }
  if (ref.kind === 'mindmap') {
    const parsed = parseMindmap(text);
    if (parsed.document === null) {
      return { kind: 'error', reason: `${ref.src} 解析不了 —— 不是一份完整的 .mindmap` };
    }
    return {
      kind: 'mindmap',
      rootTitle: parsed.document.root.topic,
      nodeCount: countNodes(parsed.document.root),
    };
  }
  return {
    kind: 'text',
    lineCount: analyze(text).lines,
    snippet: firstLines(text, SNIPPET_LINES),
  };
}

function countNodes(node: MindmapNode): number {
  let total = 1;
  for (const child of node.children) {
    total += countNodes(child);
  }
  return total;
}

function firstLines(text: string, limit: number): string {
  const lines = text.split('\n');
  const head = lines.slice(0, limit);
  // 截断要如实：尾巴上补一行说明，用户才知道下面还有内容
  return lines.length > limit ? [...head, t('core.bprev.truncatedN', { n: lines.length })].join('\n') : head.join('\n');
}
