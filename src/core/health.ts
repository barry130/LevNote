/**
 * 降级记录本。
 *
 * ## 它补的是「说出来了，然后呢」
 *
 * ADR 0042–0045 一路把静默降级一条条搬到了横幅上：偏好写不进去、存储里的东西读不出来、
 * 文件监听没起来。到这里「绝不静默」这件事做完了 —— **前提是用户看见了横幅**，
 * 而横幅有一个前提它自己管不了：**应用还开着**。
 *
 * 一次监听停掉、一次存储读坏、一次内核加载失败，用户完全可能整段会话都在别的窗口里，
 * 关掉应用时横幅从来没进入过视野。红线的另一半是「**发现**」：说过一次的话，
 * 至少要有一个地方留得下来，让他事后能回来查 —— 而不是「当时弹过了，就算了」。
 *
 * ## 为什么记的是「key + 参数」而不是「已经翻好的句子」
 *
 * 和 ADR 0043 同一条纪律：内核交数据、外壳翻成话。记录本里如果存的是成品句子，
 * 运行中切语言时老记录就永远停在启动那一刻的语言上（同一类毛病，`definitions.ts`
 * 的命令标题踩过一次）。存 key + 参数，文案永远由渲染那一刻的 `t()` 现查。
 *
 * ## 三个刻意的选择
 *
 * 1. **只认 id，不认内容**。同一个 `id` 再次发生是**累加次数**而不是新增一条 ——
 *    一个每分钟抖一次的监听错误不该把清单刷成一百行。时间跨度与次数都在同一条上。
 * 2. **恢复要显式**。`resolveIssue` 只打一个时刻，**不删记录**：用户回头查的是
 *    「发生过什么」，「后来好了」本身就是答案的一部分（自动删掉 = 又一次静默）。
 * 3. **纯函数，不自己取时间**。`at` 一律由调用方传进来 —— 记录本自己调时钟，
 *    单测就得跟真实时间赛跑，而这类测试一旦写成 `toBeTruthy` 就等于没写。
 *
 * **不落盘**是刻意的（见 ADR 0046）：跨运行保留是另一种产品决定，
 * 真要做，它该走的正是 ADR 0044 建好的 `LoadOutcome` 那条路。
 */

import type { MessageKey } from './i18n/messages.js';

/**
 * 问题属于哪一族。清单里按这个分组，顺序也是它决定的 ——
 * 分组名本身要经 `t()` 取，所以这里是标识，不是文案。
 */
export type HealthArea = 'storage' | 'session' | 'watch' | 'kernel';

/**
 * 严重度决定清单里的排序与配色。
 *
 * `error` = 现在还在坏 / 可能已经丢了东西；`warning` = 已经过去了，或者只影响一部分。
 * 只分两档：三档以上会让「要不要现在担心」这件事失去分辨力。
 */
export type HealthSeverity = 'warning' | 'error';

/** 一行文案：key + 参数，现查现翻。 */
export interface HealthLine {
  readonly key: MessageKey;
  readonly values: Readonly<Record<string, string | number>>;
}

/** 一条问题。同一个 id 永远只有一条，重复发生累加在 `count` 上。 */
export interface HealthRecord {
  /**
   * 幂等键，跨运行稳定。
   *
   * 它要能区分「同一族的两个不同实例」—— 丢的是哪几个文档必须分得开，
   * 而丢了几条偏好这种只有一个实例的，写一个固定的就行。
   * **不要**把时刻、随机数、数组下标塞进来：那样每次都是「新的一条」。
   */
  readonly id: string;
  readonly area: HealthArea;
  readonly severity: HealthSeverity;
  readonly lines: readonly HealthLine[];
  /** 发生过几次（含首次）。 */
  readonly count: number;
  readonly firstSeenAt: number;
  readonly lastSeenAt: number;
  /** 后来又好了的时刻；`null` = 此刻还没恢复。 */
  readonly resolvedAt: number | null;
}

export const EMPTY_HEALTH: readonly HealthRecord[] = [];

/** 新记一条时的入参：不带计数与时刻，那些由记录本自己管。 */
export interface HealthNote {
  readonly id: string;
  readonly area: HealthArea;
  readonly severity: HealthSeverity;
  readonly lines: readonly HealthLine[];
}

const MAX_RECORDS = 60;

/**
 * 记一次（幂等）。
 *
 * 已有的 id：`count + 1`、刷新 `lastSeenAt`、把 `resolvedAt` 清回 `null`
 * —— **又发生了就是又发生了**，不能因为上次标记过「已恢复」就当没发生过。
 * 同时更新 `lines`：同一族的问题第二次丢的可能是别的几个文档，
 * 记录本要留着最新的说法（但**只留最新的**，否则清单会越滚越长）。
 *
 * 新的 id 排在最前面（用户先看最近发生的），并受 `MAX_RECORDS` 限制 ——
 * 记录本是给人看的，不是无限追加的文件；超过上限丢掉最老的一条，
 * 且**只在这里丢**，不静默丢在别处。
 */
export function recordIssue(
  log: readonly HealthRecord[],
  note: HealthNote,
  at: number,
): readonly HealthRecord[] {
  const index = log.findIndex((record) => record.id === note.id);
  if (index >= 0) {
    const previous = log[index] as HealthRecord;
    const next: HealthRecord = {
      ...previous,
      severity: note.severity,
      lines: note.lines,
      count: previous.count + 1,
      lastSeenAt: at,
      resolvedAt: null,
    };
    const rest = [...log.slice(0, index), ...log.slice(index + 1)];
    return [next, ...rest];
  }
  return [{ ...note, count: 1, firstSeenAt: at, lastSeenAt: at, resolvedAt: null }, ...log].slice(0, MAX_RECORDS);
}

/**
 * 标记「这一条已经好了」。
 *
 * **保留记录**（打一个时刻），不删：清单回答的是「发生过什么」，
 * 删掉就等于让一次已经发生过的降级从用户的视野里消失第二次。
 * 未知 id 原样返回 —— 没有的问题不该凭空多出一条「已恢复」。
 */
export function resolveIssue(
  log: readonly HealthRecord[],
  id: string,
  at: number,
): readonly HealthRecord[] {
  let changed = false;
  const next = log.map((record) => {
    if (record.id !== id || record.resolvedAt !== null) return record;
    changed = true;
    return { ...record, resolvedAt: at };
  });
  return changed ? next : log;
}

/** 清空（「清空」按钮与「清完重新开始」共用一条路径）。 */
export function clearHealth(log: readonly HealthRecord[]): readonly HealthRecord[] {
  return log.length === 0 ? log : EMPTY_HEALTH;
}

/** 此刻还没恢复的那些（横幅关心的是「现在」，清单关心的是「全部」）。 */
export function activeHealth(log: readonly HealthRecord[]): readonly HealthRecord[] {
  return log.filter((record) => record.resolvedAt === null);
}

/** 还没恢复的条数 —— 给菜单标题、按钮标签之类要写「(n)」的地方用。 */
export function activeHealthCount(log: readonly HealthRecord[]): number {
  let total = 0;
  for (const record of log) if (record.resolvedAt === null) total += 1;
  return total;
}

/**
 * 两组文案是不是同一组。
 *
 * `values` 的键按字母序比一遍：`HealthNote` 是数据不是结构，
 * 两个参数对象里键的插入顺序不同不该让同一件事被认成两件。
 */
function sameLines(a: readonly HealthLine[], b: readonly HealthLine[]): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i += 1) {
    const left = a[i] as HealthLine;
    const right = b[i] as HealthLine;
    if (left.key !== right.key) return false;
    const leftKeys = Object.keys(left.values).sort();
    const rightKeys = Object.keys(right.values).sort();
    if (leftKeys.length !== rightKeys.length) return false;
    for (let j = 0; j < leftKeys.length; j += 1) {
      const name = leftKeys[j] as string;
      if (name !== rightKeys[j]) return false;
      if (left.values[name] !== right.values[name]) return false;
    }
  }
  return true;
}

/**
 * 把一族的「此刻状况」同步进记录本。**横幅那一族用这个。**
 *
 * 横幅状态回答的就是「此刻还有什么问题」，所以两件事都归它：
 * **有的记上，消失的标成已恢复**（恢复要显式，而这里正好知道它恢复了）。
 *
 * 一次性发生的事不走这里 —— 内核加载失败、一次保存写不进去，它们没有
 * 「恢复」这个状态，硬套只会让记录变成另一种含糊。
 *
 * ## 为什么「重复的那次」不算又发生一次
 *
 * 上报点是**渲染里**（横幅状态每帧都读得到），而 StrictMode 会把渲染重跑一遍 ——
 * 那不是又出了一次问题，是同一次问题的第二次读。判据是「**内容有没有变**」：
 * 同一条、同样的文案、又还没恢复 → 什么都不做。这样双挂不翻倍，
 * 清单上「发生过 n 次」那句话也就不会说谎。内容变了才算又发生一次
 * （丢的文档换了、监听的报错换了）。
 */
export function syncFamily(
  log: readonly HealthRecord[],
  notes: readonly HealthNote[],
  ids: readonly string[],
  at: number,
): readonly HealthRecord[] {
  let next = log;
  const live = new Set<string>();
  for (const note of notes) {
    live.add(note.id);
    const previous = next.find((record) => record.id === note.id);
    if (previous !== undefined && previous.resolvedAt === null && sameLines(previous.lines, note.lines)) continue;
    next = recordIssue(next, note, at);
  }
  for (const id of ids) {
    if (live.has(id)) continue;
    next = resolveIssue(next, id, at);
  }
  return next;
}