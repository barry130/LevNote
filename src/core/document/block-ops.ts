/**
 * 块级互操作（P6 第 28 项）里**属于外壳**的那一半：文件命名、引用围栏的
 * 组装、以及「提取 / 拉平」两条动作的编排。
 *
 * 分工（与图片粘贴同构）：内核做文档模型的事（哪一段被选中、光标在哪个
 * 围栏块里、怎么替换），外壳做文件系统的事（新建文件、读引用、命名、
 * 绝不覆盖）。这里的函数只认**注入的依赖**，不碰真实的 provider ——
 * 编排逻辑要在单测里跑，文件系统在哪都行。
 */

import type { EditorFenceInfo } from '../editor/types.js';
import { t } from '../i18n/index.js';
import { resolveRelative } from '../path.js';
import { createDocBlock } from '@qingmo/interop-spec';
import { parseDocBlock } from '../docblock.js';

/**
 * 从选区内容里取一个**可读的**标题：第一个 ATX 标题的文字。
 * 没有标题时返回 null（调用方回退到文件名）。
 */
export function extractTitle(selectedMarkdown: string): string | null {
  for (const line of selectedMarkdown.split('\n')) {
    const match = line.match(/^#{1,6}\s+(.*\S)\s*$/);
    if (match !== null) return match[1] ?? null;
  }
  return null;
}

/**
 * 从选区内容里推断新文件的名字（不含目录）。
 *
 * 第一优先是**第一个标题**：用户写文档时，标题本来就是这一节的名字，
 * 提取出来的文件叫这个最自然。标题里不适合做文件名的字符全部替换掉
 * （Windows 保留字符与路径分隔符），空白压成连字符，太长的截断 ——
 * 截断后**再**去重，否则两个长标题可能截出同一个名字。
 */
export function extractNameCandidates(selectedMarkdown: string): { first: string; next(): string } {
  const title = extractTitle(selectedMarkdown);
  const base = sanitizeFileName(title ?? 'excerpt');
  let previous: string | null = null;
  let counter = 1;
  return {
    first: `${base}.md`,
    next(): string {
      previous = previous ?? `${base}.md`;
      counter += 1;
      return `${base}-${counter}.md`;
    },
  };
}

function sanitizeFileName(raw: string): string {
  const cleaned = raw
    // Windows 文件名不接受的字符 + 路径分隔符：全换成连字符
    .replace(/[\\/:*?"<>|]/g, '-')
    .replace(/\s+/g, '-')
    .replace(/^-+|-+$/g, '');
  const trimmed = cleaned === '' ? 'excerpt' : cleaned;
  return trimmed.length > 32 ? trimmed.slice(0, 32).replace(/-+$/, '') : trimmed;
}

/**
 * 组装一个 `mode: reference` 的 doc-block 围栏（规范 §2 / §4）。
 *
 * 序列化走 `@qingmo/interop-spec` —— 属性顺序固定、载荷为空、
 * 反引号长度自算，都是规范里钉死的行为，不在这里重抄一份。
 * `src` 按规范**一律正斜杠**（相对当前文档目录）。
 */
export function buildReferenceFence(src: string, title: string | null): string {
  return createDocBlock({
    kind: 'markdown',
    mode: 'reference',
    // 规范 §2.3：src 一律正斜杠（这个项目的 TS lib 没有 replaceAll）
    src: src.split('\\').join('/'),
    ...(title !== null && title !== '' ? { title } : {}),
  }).raw;
}

/** 读不到 / 写不进时依赖方怎么把话说出来，由命令层注入。 */
export interface BlockOpsNotify {
  (message: string, title?: string): Promise<void>;
}

export interface BlockOpsDeps {
  /** 当前文档的路径 —— 引用的 src 相对它的目录。 */
  docPath: string;
  /** 内核命令面（`EditorHandle.blockTransform`）。 */
  selectedMarkdown(): string | null;
  replaceSelectionWith(markdown: string): void;
  enclosingFence(): EditorFenceInfo | null;
  replaceEnclosingFenceWith(markdown: string): void;
  /** 读一个文本文件；读不到（不存在 / 无权限）返回 null，**不抛**。 */
  readTextFile(path: string): Promise<{ text: string } | null>;
  /** 写一个文本文件（新建）。失败时抛错，由这里统一兜住。 */
  writeTextFile(path: string, text: string): Promise<void>;
  notify: BlockOpsNotify;
}

/**
 * 「提取为引用」：把选区写成一个新 `.md` 文件，选区原地替换成指向它的
 * `doc-block` 引用（L3 的正向）。
 *
 * - **绝不覆盖**：候选名被占用就加序号重试（与图片命名同一条纪律），
 *   试到上限就放弃并说明 —— 用户可以自己改名再试，而不是被覆盖掉别的东西。
 * - 新文件**立即**写入磁盘；宿主文档变脏等正常保存。两件事哪个先发生都
 *   不产生引用悬空：引用在文档里，文件已在盘上。
 * - 提取范围限定在**同目录**：跨目录的相对路径要处理 `..` 折叠，
 *   而 `joinPath` 刻意不做这件事（硬约束 12）。
 */
export async function extractToReference(deps: BlockOpsDeps): Promise<void> {
  const selected = deps.selectedMarkdown();
  if (selected === null || selected.trim() === '') {
    await deps.notify(t('core.blops.selectFirst'), t('core.blops.extractTitle'));
    return;
  }

  const candidates = extractNameCandidates(selected);
  let name = candidates.first;
  // 上限是个保险丝：理论上撞到 100 个的概率为零，但「绝不确定地覆盖」
  // 意味着循环必须有个说清楚理由的出口
  let free = false;
  for (let attempt = 0; attempt < 100; attempt += 1) {
    const existing = await deps.readTextFile(resolveRelative(deps.docPath, name) ?? '');
    if (existing === null) {
      free = true;
      break;
    }
    name = candidates.next();
  }
  if (!free) {
    await deps.notify(t('core.blops.nameExhausted'), t('core.blops.extractTitle'));
    return;
  }

  const title = extractTitle(selected);
  const fence = buildReferenceFence(name, title);

  try {
    await deps.writeTextFile(resolveRelative(deps.docPath, name) ?? '', selected);
  } catch (thrown) {
    await deps.notify(t('core.blops.writeFailed', { name, reason: thrown instanceof Error ? thrown.message : String(thrown) }), t('core.blops.extractTitle'));
    return;
  }

  deps.replaceSelectionWith(fence);
  await deps.notify(
    t('core.blops.extracted', { name }),
    t('core.blops.extractTitle'),
  );
}

/**
 * 「拉平引用」：把光标所在的 doc-block 引用替换成它指向的内容（L3 的反向）。
 *
 * - 只有**文本形**的 kind（markdown / text / code）可以拉平 —— 图表与导图
 *   的内容塞进 Markdown 正文里只会得到一堆没人读得了的源码，那是降级不是转换；
 * - `mode: inline` 的载荷就在块里，直接展开；`mode: reference` 的去读文件，
 *   读不到要说清楚；
 * - 被引用的文件**不删除** —— 拉平是一次性的文档操作，删文件是另一个决定，
 *   该由用户自己做。
 */
export async function flattenReference(deps: BlockOpsDeps): Promise<void> {
  const fence = deps.enclosingFence();
  if (fence === null || fence.info !== 'doc-block') {
    await deps.notify(t('core.blops.cursorNotInRef'), t('core.blops.flattenTitle'));
    return;
  }

  const parsed = parseDocBlock(fence.content);
  if (parsed.errors.includes('MISSING_TERMINATOR')) {
    await deps.notify(t('core.blops.missingEnd'), t('core.blops.flattenTitle'));
    return;
  }

  const kind = parsed.attrs.kind;
  if (kind !== 'markdown' && kind !== 'text' && kind !== 'code') {
    await deps.notify(t('core.blops.kindNotFlattenable'), t('core.blops.flattenTitle'));
    return;
  }

  let content: string;
  if (parsed.attrs.mode === 'inline') {
    content = parsed.payload ?? '';
  } else {
    const src = parsed.attrs.src;
    if (src === null || src === '') {
      await deps.notify(t('core.blops.noSrc'), t('core.blops.flattenTitle'));
      return;
    }
    const absolute = resolveRelative(deps.docPath, src);
    const file = absolute === null ? null : await deps.readTextFile(absolute);
    if (file === null) {
      await deps.notify(t('core.blops.srcUnreadable', { src }), t('core.blops.flattenTitle'));
      return;
    }
    content = file.text;
  }

  if (content.trim() === '') {
    await deps.notify(t('core.blops.emptyRef'), t('core.blops.flattenTitle'));
    return;
  }

  deps.replaceEnclosingFenceWith(content);
  await deps.notify(t('core.blops.flattened'), t('core.blops.flattenTitle'));
}
