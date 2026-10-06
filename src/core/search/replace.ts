/**
 * 跨文件替换（A3）的编排层。
 *
 * ## 为什么替换在 TS 而不是 Rust 侧再写一遍
 *
 * 匹配与替换的语义（大小写 / 全词 / 正则 / `$1` 展开）已经有**唯一一份**实现
 * —— `core/search/find.ts`，文档内查找替换用的就是它。Rust 的检索匹配器
 * （`search/matcher.rs`）只负责「找出位置」，不负责「展开替换」；
 * 如果在这边为了贴近检索再实现一套替换语义，`$1`、字面 `$`、全词边界
 * 就会有两种行为 —— 「检索到的东西替换成了别的东西」正是这种漂移的典型症状。
 *
 * ## 读写同一条链路
 *
 * 文件的读与写都走 `DocumentProvider`：读回来的 `TextFile` 带着**探测出的**
 * 编码、BOM 与行尾，写回时原样带上 —— GBK 文件替换后还是 GBK，
 * 带 BOM 的还带 BOM，混排行尾**不传目标行尾**（保存路径的同一条纪律：
 * 不该由一次替换擅自统一掉）。
 *
 * ## 跳过与失败要说清楚
 *
 * 调用方把**打开且有未保存改动**的路径传进来，这里原样跳过；
 * 读不到 / 写不进（只读、权限）的文件逐个记进 `failed`。全部如实回给调用方
 * （硬约束 2）—— 「替换了 7 处」后面必须跟着「3 个文件跳过了」，
 * 否则用户分不清「没替换」和「替换了但看不见」。
 */

import { replaceInText, type FindOptions } from './find.js';
import type { TextFile, WriteRequest } from '../document/types.js';

/** 替换需要的最小 IO 面。传入 provider 的两个方法即可（结构化类型，直接兼容）。 */
export interface ReplaceIo {
  readTextFile(path: string): Promise<TextFile>;
  /** 返回值在这里没有用途，声明成 unknown —— 只关心「写成功了没有」。 */
  writeTextFile(request: WriteRequest): Promise<unknown>;
}

export interface ReplaceSkip {
  readonly path: string;
  readonly reason: string;
}

export interface ReplaceOutcome {
  /** 成功写入的文件数。 */
  readonly filesReplaced: number;
  /** 实际替换掉的匹配处数。 */
  readonly matchesReplaced: number;
  /** 有意跳过的文件（打开且脏 / 这次扫描已经没有命中）。 */
  readonly skipped: readonly ReplaceSkip[];
  /** 想替换但没成功的文件（读不到 / 写不进），逐个带原因。 */
  readonly failed: readonly ReplaceSkip[];
}

export const EMPTY_REPLACE_OUTCOME: ReplaceOutcome = {
  filesReplaced: 0,
  matchesReplaced: 0,
  skipped: [],
  failed: [],
};

/**
 * 对 `paths` 里的文件逐个做「读 → 替换 → 写回」。
 *
 * 逐个 await 而不是并发：几十个文件并发写盘对机械硬盘不友好，
 * 而且中途失败时串行的版本更容易知道停在哪。文件数是检索结果的文件数
 * （几十的量级），串行的总耗时可接受。
 *
 * `dirtyPaths` 里的文件**一个字节都不会碰** —— 那个标签页的缓冲区还是旧的，
 * 磁盘改了它也不知道，用户一按 Ctrl+S 就把替换结果整个盖回去了。
 */
export async function replaceAcrossFiles(
  paths: readonly string[],
  options: FindOptions,
  replacement: string,
  dirtyPaths: ReadonlySet<string>,
  io: ReplaceIo,
): Promise<ReplaceOutcome> {
  const skipped: ReplaceSkip[] = [];
  const failed: ReplaceSkip[] = [];
  let filesReplaced = 0;
  let matchesReplaced = 0;

  for (const path of paths) {
    if (dirtyPaths.has(path)) {
      skipped.push({ path, reason: '在打开的标签页里有未保存的改动' });
      continue;
    }

    let file: TextFile;
    try {
      file = await io.readTextFile(path);
    } catch (thrown) {
      failed.push({ path, reason: thrown instanceof Error ? thrown.message : '读不出来' });
      continue;
    }

    const { text: next, count } = replaceInText(file.text, options, replacement);
    if (count === 0) {
      // 检索结果已过期（文件被改过 / 换了工作区）：没有命中就没有写入，
      // 如实记一笔跳过，不冒充成功
      skipped.push({ path, reason: '重新读取后没有命中（结果可能已过期）' });
      continue;
    }

    try {
      await io.writeTextFile({
        path,
        text: next,
        // 读到什么编码 / BOM 就按什么写回去；混排行尾不传目标行尾（原样保留）
        encoding: file.encoding,
        bom: file.bom,
        create: true,
      });
    } catch (thrown) {
      const reason = thrown instanceof Error ? thrown.message : '写不进去';
      failed.push({ path, reason });
      continue;
    }

    filesReplaced += 1;
    matchesReplaced += count;
  }

  return { filesReplaced, matchesReplaced, skipped, failed };
}

/** 把结果收敛成一句给用户看的话。跳过与失败必须说出来。 */
export function describeReplaceOutcome(outcome: ReplaceOutcome): string {
  const parts: string[] = [];
  if (outcome.filesReplaced === 0 && outcome.skipped.length === 0 && outcome.failed.length === 0) {
    return '没有需要替换的内容。';
  }
  parts.push(`替换了 ${String(outcome.matchesReplaced)} 处（${String(outcome.filesReplaced)} 个文件）`);
  if (outcome.skipped.length > 0) parts.push(`跳过 ${String(outcome.skipped.length)} 个`);
  if (outcome.failed.length > 0) parts.push(`失败 ${String(outcome.failed.length)} 个`);
  return parts.join('，');
}
