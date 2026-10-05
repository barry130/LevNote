/**
 * 往返保真自检（P6 第 29 项，ADR 0033）的**比对与报告**。
 *
 * 背景：Markdown 内核「打开 → 不改 → 保存」不保证逐字节还原 ——
 * 解析成文档树再序列化回去，会做一些规范化（实测：setext 标题变 ATX、
 * 表格按列宽重排、行尾空白收敛……见 `modules/markdown/setup.ts` 的注释）。
 * 那些改写**故意保留**的（它们让写回的文件更规整），但用户有权**显式地
 * 看到它们**，而不是某天保存后才发现文件被动了。
 *
 * 这个模块只负责「逐行比对 + 生成人读的报告」；「走一遍内核自己的
 * 解析 → 序列化」是内核的能力（`EditorHandle.roundTrip`），外壳拿两者比。
 *
 * ## 为什么逐行而不是逐字节
 *
 * 用户关心的是「哪些**行**被动了」，一行一个可读单元；逐字节的字符级
 * diff（Myers / 词法）实现贵、输出还读不懂。行级 LCS 的 O(n·m) 在
 * 千行级文档上是几十万格，毫秒级；超过上限时退化为多重集计数
 * （只报数量、不给样例，并如实标注「未逐行比对」）。
 */

/** 逐行 LCS 的格数上限：约 2000 × 2000 行。超过就走近似比对。 */
const LCS_CELL_LIMIT = 4_000_000;

export interface RoundTripReport {
  /** 往返后逐字节一致 */
  readonly same: boolean;
  /** 内容有变化的行对数（旧行还在、内容不同） */
  readonly changed: number;
  /** 往返后多出来的行数 */
  readonly added: number;
  /** 往返后消失的行数 */
  readonly removed: number;
  /** 原文行数 */
  readonly totalLines: number;
  /** 人读样例（最多三条）。近似比对时为空。 */
  readonly samples: readonly string[];
  /**
   * 超过逐行比对上限、退化为多重集计数时为 true ——
   * 数量仍如实，但没有逐行依据，报告里必须说出来。
   */
  readonly approximate: boolean;
}

function truncate(line: string): string {
  return line.length > 40 ? `${line.slice(0, 40)}…` : line;
}

/**
 * 按人的直觉分行：结尾的换行符是「这一行结束了」的标记，不产生空行。
 * （`'a\n'.split('\n')` 会多出一个空元素，行数差一。）
 */
function splitLines(text: string): string[] {
  if (text === '') return [];
  const body = text.endsWith('\n') ? text.slice(0, -1) : text;
  return body.split('\n');
}

/** 逐行 LCS。`dp[i][j]` = a 从第 i 行、b 从第 j 行开始的公共行数。 */
function lcsTable(a: readonly string[], b: readonly string[]): Int32Array {
  const rows = a.length + 1;
  const cols = b.length + 1;
  const dp = new Int32Array(rows * cols);
  for (let i = a.length - 1; i >= 0; i -= 1) {
    for (let j = b.length - 1; j >= 0; j -= 1) {
      const left = a[i] ?? '';
      const right = b[j] ?? '';
      dp[i * cols + j] =
        left === right
          ? (dp[(i + 1) * cols + j + 1] ?? 0) + 1
          : Math.max(dp[(i + 1) * cols + j] ?? 0, dp[i * cols + j + 1] ?? 0);
    }
  }
  return dp;
}

export function describeRoundTrip(original: string, roundTripped: string): RoundTripReport {
  if (original === roundTripped) {
    return { same: true, changed: 0, added: 0, removed: 0, totalLines: splitLines(original).length, samples: [], approximate: false };
  }

  const a = splitLines(original);
  const b = splitLines(roundTripped);
  const samples: string[] = [];

  // 超上限：不跑 LCS，用多重集计数估计规模（每个行文本出现次数取交集）
  if (a.length * b.length > LCS_CELL_LIMIT) {
    const counts = new Map<string, number>();
    for (const line of a) counts.set(line, (counts.get(line) ?? 0) + 1);
    let common = 0;
    for (const line of b) {
      const left = counts.get(line) ?? 0;
      if (left > 0) {
        common += 1;
        counts.set(line, left - 1);
      }
    }
    let removed = a.length - common;
    let added = b.length - common;
    const changed = Math.min(removed, added);
    removed -= changed;
    added -= changed;
    return { same: false, changed, added, removed, totalLines: a.length, samples, approximate: true };
  }

  const cols = b.length + 1;
  const dp = lcsTable(a, b);
  let changed = 0;
  let added = 0;
  let removed = 0;
  let i = 0;
  let j = 0;

  while (i < a.length && j < b.length) {
    if (a[i] === b[j]) {
      i += 1;
      j += 1;
      continue;
    }
    const del = dp[(i + 1) * cols + j] ?? 0;
    const ins = dp[i * cols + j + 1] ?? 0;
    // a[i] ≠ b[j] 时「当成替换走」与「删+增」的 LCS 代价相同；
    // 优先替换：一行一对，报告读起来是「这行被改了」而不是「删了一行又加了一行」
    const sub = dp[(i + 1) * cols + j + 1] ?? 0;
    if (sub >= del && sub >= ins) {
      changed += 1;
      if (samples.length < 3) samples.push(`第 ${i + 1} 行：「${truncate(a[i] ?? '')}」→「${truncate(b[j] ?? '')}」`);
      i += 1;
      j += 1;
    } else if (del >= ins) {
      removed += 1;
      if (samples.length < 3) samples.push(`第 ${i + 1} 行：「${truncate(a[i] ?? '')}」被删掉`);
      i += 1;
    } else {
      added += 1;
      if (samples.length < 3) samples.push(`第 ${i + 1} 行前新增：「${truncate(b[j] ?? '')}」`);
      j += 1;
    }
  }
  while (i < a.length) {
    removed += 1;
    if (samples.length < 3) samples.push(`第 ${i + 1} 行：「${truncate(a[i] ?? '')}」被删掉`);
    i += 1;
  }
  while (j < b.length) {
    added += 1;
    if (samples.length < 3) samples.push(`末尾新增：「${truncate(b[j] ?? '')}」`);
    j += 1;
  }

  return { same: false, changed, added, removed, totalLines: a.length, samples, approximate: false };
}

/** 报告 → 用户可见的一句话（或一段话）。如实是底线：改了什么、没比对什么，都要说。 */
export function formatRoundTripReport(report: RoundTripReport): string {
  if (report.same) {
    return `往返保真：原文 ${report.totalLines} 行，「打开 → 不改 → 保存」不会改写任何内容。`;
  }
  const rewritten = report.changed + report.added + report.removed;
  const head =
    `往返保真：原文 ${report.totalLines} 行，不改一个字保存会改写 ${rewritten} 行` +
    `（修改 ${report.changed} / 新增 ${report.added} / 删除 ${report.removed}）` +
    (report.approximate ? '。文档太大，未逐行比对，数量是估计值。' : '。');
  if (report.samples.length === 0) return head;
  return `${head}\n${report.samples.join('\n')}`;
}
