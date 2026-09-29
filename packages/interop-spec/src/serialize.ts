/**
 * doc-block 序列化器。
 *
 * 行尾策略：默认输出 `\n`，但支持显式指定目标行尾。
 * 为什么不干脆「永远只输出 \n，交给写入层统一」：那样在 CRLF 文档里改动一个块，
 * 就会出现「这个块 LF、其余 CRLF」的混合行尾，下一轮打开又被整篇重写，diff 全是噪点。
 * 行尾必须由**知道目标文件属性的一方**（文档模型）显式传入，序列化器只忠实执行。
 */

import {
  DOC_BLOCK_FENCE_MIN,
  DOC_BLOCK_INFO,
  DOC_BLOCK_TERMINATOR,
  ID_PATTERN,
  type CreateDocBlockInput,
  type DocBlock,
  type DocBlockAttrs,
  type DocKind,
} from './types.js';

export type LineEnding = '\n' | '\r\n' | '\r';

export interface SerializeOptions {
  /** 目标行尾，默认 `\n`。 */
  lineEnding?: LineEnding;
}

/** 从一段文本推断主行尾。混排时取多数，空文本视为 `LF`。 */
export function detectLineEnding(text: string): LineEnding {
  const crlf = (text.match(/\r\n/g) ?? []).length;
  const totalLf = (text.match(/\n/g) ?? []).length;
  const totalCr = (text.match(/\r/g) ?? []).length;
  const loneLf = totalLf - crlf;
  const loneCr = totalCr - crlf;

  if (crlf === 0 && loneCr === 0) return '\n';
  if (crlf > 0 && loneLf === 0 && loneCr === 0) return '\r\n';
  if (loneCr > 0 && loneLf === 0 && crlf === 0) return '\r';

  if (crlf >= loneLf && crlf >= loneCr) return '\r\n';
  return loneCr > loneLf ? '\r' : '\n';
}

function toLf(text: string): string {
  return text.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
}

/** 载荷里出现「整行都是反引号」时必须加长围栏，否则会被误判成结束围栏。 */
export function neededFenceLength(payload: string, preferred = DOC_BLOCK_FENCE_MIN): number {
  let max = DOC_BLOCK_FENCE_MIN;
  for (const line of payload.split(/\r\n|\n|\r/)) {
    const trimmed = line.trim();
    if (trimmed.length >= DOC_BLOCK_FENCE_MIN && /^`+$/.test(trimmed)) {
      max = Math.max(max, trimmed.length + 1);
    }
  }
  return Math.max(max, preferred, DOC_BLOCK_FENCE_MIN);
}

/** 值含首尾空白或引号时加引号，避免解析后被 trim 掉。 */
function formatValue(value: string): string {
  const needsQuote =
    value !== value.trim() ||
    value.startsWith('"') ||
    value.startsWith("'") ||
    value.includes('\n') ||
    value.includes('\r');
  if (!needsQuote) return value;
  return `"${value.replace(/"/g, "'")}"`;
}

/** 属性区文本（不含围栏与分隔行）。导出出来便于测试与预览。 */
export function formatAttrs(attrs: DocBlockAttrs): string {
  const lines: string[] = [
    `id: ${formatValue(attrs.id)}`,
    `kind: ${attrs.kind}`,
    `mode: ${attrs.mode}`,
  ];
  if (attrs.src !== undefined && attrs.src !== '') lines.push(`src: ${formatValue(attrs.src)}`);
  if (attrs.title !== undefined && attrs.title !== '') lines.push(`title: ${formatValue(attrs.title)}`);

  // extra 已排序输出，保证同样的输入永远产出同样的文本（diff 友好）
  for (const key of Object.keys(attrs.extra).sort()) {
    const value = attrs.extra[key];
    if (value === undefined) continue;
    lines.push(`${key}: ${formatValue(value)}`);
  }
  return lines.join('\n');
}

export function serializeDocBlock(block: DocBlock, options: SerializeOptions = {}): string {
  const ending = options.lineEnding ?? '\n';

  // 先把载荷压成 LF，最后统一替换成目标行尾 ——
  // 否则载荷里已有的 \r\n 会在替换时被二次转义成 \r\r\n。
  const normalized: DocBlock = ending === '\n'
    ? block
    : { ...block, payload: toLf(block.payload) };

  const result = buildBlockWithLf(normalized);
  return ending === '\n' ? result : result.replace(/\n/g, ending);
}

/** 以 LF 组装块文本。内部函数：对外一律走 `serializeDocBlock`。 */
function buildBlockWithLf(block: DocBlock): string {
  const fence = '`'.repeat(neededFenceLength(block.payload, block.fenceLength));
  const head = `${fence}${DOC_BLOCK_INFO}\n${formatAttrs(block.attrs)}\n${DOC_BLOCK_TERMINATOR}\n`;

  if (block.payload === '') return `${head}${fence}`;

  // 载荷原样写入；结尾若没有换行，补一个让结束围栏独占一行
  const body = block.payload.endsWith('\n') ? block.payload : `${block.payload}\n`;
  return `${head}${body}${fence}`;
}

let idCounter = 0;

/** 生成稳定可读的默认 id。带时间戳 + 计数器，避免同一毫秒内新建多个块时撞号。 */
export function generateBlockId(kind: DocKind): string {
  idCounter = (idCounter + 1) % 100000;
  const stamp = Date.now().toString(36).slice(-4);
  const seq = idCounter.toString(36).padStart(2, '0');
  return `${kind}-${stamp}${seq}`;
}

export function createDocBlock(input: CreateDocBlockInput): DocBlock {
  const requestedId = input.id ?? '';
  const id = ID_PATTERN.test(requestedId) ? requestedId : generateBlockId(input.kind);

  const attrs: DocBlockAttrs = {
    id,
    kind: input.kind,
    mode: input.mode ?? 'reference',
    extra: { ...(input.extra ?? {}) },
  };
  if (input.src !== undefined) attrs.src = input.src;
  if (input.title !== undefined) attrs.title = input.title;

  const block: DocBlock = {
    attrs,
    payload: input.payload ?? '',
    range: { start: 0, end: 0 },
    raw: '',
    fenceLength: DOC_BLOCK_FENCE_MIN,
  };

  // raw 与 range 由序列化结果反填，保证新建的块也能走「未改动直接回填」的快路径
  block.raw = serializeDocBlock(block);
  block.range = { start: 0, end: block.raw.length };
  return block;
}

/**
 * 把解析出的块按位移写回原文。用于编辑器里替换单个块而不重排全文。
 * `blocks` 必须按 range.start 升序且互不重叠。
 */
export function spliceBlocks(text: string, blocks: readonly DocBlock[]): string {
  let out = '';
  let cursor = 0;
  for (const block of blocks) {
    if (block.range.start < cursor || block.range.end > text.length || block.range.start > block.range.end) {
      throw new Error(`块 ${block.attrs.id} 的 range 非法或与他块重叠，拒绝写回以避免破坏文档`);
    }
    out += text.slice(cursor, block.range.start);
    out += block.raw;
    cursor = block.range.end;
  }
  out += text.slice(cursor);
  return out;
}
