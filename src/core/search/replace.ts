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
 *
 * ## 原因交出去的时候**不翻**（ADR 0047）
 *
 * `skipped` / `failed` 里的 `reason` 是 `LocalizedText`，不是句子 ——
 * 抛物自带的 message 当**参数**走（那是系统的原话，翻了就是撒谎），
 * 而「读不到 / 写不进」这个上下文是我们的，才进 catalog。
 * 这一层拼好的句子会被 `SearchPanel` 存进 state 并活过很多次渲染，
 * 切语言时它不会跟着翻 —— 与其每处记得清一遍，不如交出去的时候就不翻。
 * 翻的动作只有一处：`renderReplaceReport()`。
 */

import { replaceInText, type FindOptions } from './find.js';
import { msg, t, textOf, type LocalizedText } from '../i18n/index.js';
import { basename } from '../path.js';
import type { TextFile, WriteRequest } from '../document/types.js';

/** 替换需要的最小 IO 面。传入 provider 的两个方法即可（结构化类型，直接兼容）。 */
export interface ReplaceIo {
  readTextFile(path: string): Promise<TextFile>;
  /** 返回值在这里没有用途，声明成 unknown —— 只关心「写成功了没有」。 */
  writeTextFile(request: WriteRequest): Promise<unknown>;
}

export interface ReplaceSkip {
  readonly path: string;
  /** 还没翻的理由（ADR 0047）：catalog 里那一条 + 它的参数，翻译留给渲染那一刻。 */
  readonly reason: LocalizedText;
}

/** 一个文件的问题行：跳过与失败用的是同一张形状。 */
export interface ReplaceProblem {
  readonly path: string;
  readonly reason: LocalizedText;
}

/**
 * 一次跨文件替换的**结构化**汇报（ADR 0047）。
 *
 * 之前这里直接返回一句拼好的中文句子，它被 `SearchPanel` 存进 state
 * 就再也不翻了 —— 切到英文时那行还是中文。现在交出来的全是「要查哪一条」，
 * `renderReplaceReport()` 才是唯一把它翻成句子的地方。
 */
export interface ReplaceReport {
  /** 首行的片段：做了什么 + 跳过了几个 + 失败了几个。分隔符也来自 catalog。 */
  readonly headline: readonly LocalizedText[];
  /** 逐个文件的问题行，按文件名列出来。 */
  readonly problems: readonly ReplaceProblem[];
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
      skipped.push({ path, reason: msg('core.replace.skipDirty') });
      continue;
    }

    let file: TextFile;
    try {
      file = await io.readTextFile(path);
    } catch (thrown) {
      // 抛物自带的 message 是**原文**，不是译文 —— 它当参数走，不当文案翻。
      // 而「读不出来」这个上下文是我们知道的，所以一定要说：只给一句
      // `EACCES: permission denied` 用户得自己猜是哪一步坏了。
      failed.push({ path, reason: msg('core.replace.readFailed', { detail: describeThrown(thrown) }) });
      continue;
    }

    const { text: next, count } = replaceInText(file.text, options, replacement);
    if (count === 0) {
      // 检索结果已过期（文件被改过 / 换了工作区）：没有命中就没有写入，
      // 如实记一笔跳过，不冒充成功
      skipped.push({ path, reason: msg('core.replace.skipStale') });
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
      failed.push({ path, reason: msg('core.replace.writeFailed', { detail: describeThrown(thrown) }) });
      continue;
    }

    filesReplaced += 1;
    matchesReplaced += count;
  }

  return { filesReplaced, matchesReplaced, skipped, failed };
}

/**
 * 把结果收敛成**结构化**的汇报：做了什么、跳过了几个、失败了哪些文件。
 *
 * 「没有需要替换的内容」是一个整体句子而不是片段 —— 它独占一行，
 * 混进逗号串里会读成「，没有需要替换的内容」。
 */
export function describeReplaceOutcome(outcome: ReplaceOutcome): ReplaceReport {
  if (outcome.filesReplaced === 0 && outcome.skipped.length === 0 && outcome.failed.length === 0) {
    return { headline: [msg('core.replace.nothingToDo')], problems: [] };
  }
  const headline: LocalizedText[] = [
    msg('core.replace.doneSummary', { matches: outcome.matchesReplaced, files: outcome.filesReplaced }),
  ];
  if (outcome.skipped.length > 0) headline.push(msg('core.replace.skippedN', { n: outcome.skipped.length }));
  if (outcome.failed.length > 0) headline.push(msg('core.replace.failedN', { n: outcome.failed.length }));
  return { headline, problems: [...outcome.failed, ...outcome.skipped] };
}

/**
 * 唯一一处把汇报翻成句子的地方 —— **在渲染的那一刻调**（ADR 0047）。
 *
 * 连分隔符和「- 文件名：原因」的冒号都来自 catalog：中文用「，」，
 * 英文用 ", "；漏掉这一步就等于在英文界面里画一个中文逗号。
 */
export function renderReplaceReport(report: ReplaceReport): string {
  const headline = report.headline.map(textOf).join(t('core.replace.join'));
  if (report.problems.length === 0) return headline;
  const problems = report.problems.map((problem) =>
    t('core.replace.problemLine', { name: basename(problem.path), reason: textOf(problem.reason) }),
  );
  return `${headline}\n${problems.join('\n')}`;
}

/**
 * 把抛出来的东西压成一句**原文**。
 *
 * 它来自文件系统 / 内核，是系统自己的话；当成译文去「翻译」就是撒谎，
 * 所以只当参数插进我们 catalog 里那一句的 `{detail}`。
 */
function describeThrown(thrown: unknown): string {
  return thrown instanceof Error ? thrown.message : String(thrown);
}
