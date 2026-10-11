/**
 * 拖入多个文件时的打开与汇总（ADR 0041）。
 *
 * ## 起因
 *
 * 拖放分支原先只取 `paths[0]`：
 *
 * ```ts
 * const first = event.payload.paths?.[0];
 * if (first) void workspace.open(first);
 * ```
 *
 * 拖三个文件进来只开一个，另外两个**连一句交代都没有** —— 用户看到的是
 * 「我拖进去了，它只认一个」，而真相是「它压根没去试另外两个」。
 * 这正是硬约束 2 说的「静默丢东西」：不是打开失败了没说，是**根本没尝试**。
 *
 * ## 为什么逐个打开而不批量
 *
 * `WorkspaceStore.open()` 一次只接一个路径（它在成功时还要展开文件树、
 * 激活标签页，批量会让「开到第三个失败了」这件事没有落点）。逐个调的代价
 * 只是多几次 await，而收益是**每一个都有结果**。
 *
 * ## 成功为什么安静
 *
 * 与 ADR 0040 的取舍同一条：打开成功这件事，屏幕上的新标签页自己会说，
 * 再弹一句「已打开 3 个」是噪音。所以这里只在**有失败**时开口，
 * 且一句话说完 —— 用户可以照着名单自己再试一次。
 */

import { basename } from '../path.js';
import { t } from '../i18n/index.js';
import type { WorkspaceStore } from './store.js';

export interface DropOutcome {
  /** 真正打开（或本来就已经开着被激活）的数量。 */
  readonly opened: number;
  /** 没能打开的路径，顺序与传入一致。 */
  readonly failed: readonly string[];
}

/**
 * 逐个打开拖进来的路径。
 *
 * 单个失败**不中断**后面的 —— 拖进来的头一个坏了，不该连累第二、第三个。
 * 每个失败的原因落在 `WorkspaceStore.openError` 上（一次只留最近一个），
 * 所以**名单**必须在这里收集：那是唯一一份完整的记录。
 */
export async function openDroppedPaths(paths: readonly string[], workspace: WorkspaceStore): Promise<DropOutcome> {
  const failed: string[] = [];
  let opened = 0;

  for (const path of paths) {
    try {
      const ok = await workspace.open(path);
      if (ok) opened += 1;
      else failed.push(path);
    } catch {
      // `open` 在正常情况下把失败收进 openError 并返回 false；
      // 走到这里说明它抛了（provider 直接 throw 之类），同样只记名字 ——
      // 提示要说的是「哪几个没开成」，不是逐个陈列异常
      failed.push(path);
    }
  }

  return { opened, failed };
}

/**
 * 把结果变成一句提示。**没有失败就返回 null**（成功安静）。
 *
 * 「一个都没开成」与「开了几个、剩下几个没开成」分开两句：
 * 前者是这次拖放整个失败了，用户需要立刻知道；后者是他的多数目的已经达成，
 * 提示只是补一句名单。混成一句会让「全失败」听起来像是小问题。
 */
export function describeDropOutcome(outcome: DropOutcome): { readonly title: string; readonly body: string } | null {
  const { opened, failed } = outcome;
  if (failed.length === 0) return null;

  const names = failed.map((path) => basename(path)).join('、');
  const n = failed.length;
  return {
    title: t('core.cmd.dropTitle'),
    body: opened === 0 ? t('core.cmd.dropAllFailed', { n, names }) : t('core.cmd.dropSomeFailed', { n, names }),
  };
}
