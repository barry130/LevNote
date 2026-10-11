/**
 * 文档位置记忆 —— 「上次读到哪儿」。
 *
 * ## 为什么和 `settings.ts` 分成两个键，而不是塞进同一个对象
 *
 * 两者看着都是「偏好」，但**生命周期完全不同**：
 *
 * - 设置是用户**显式**改的、条目固定（十几个字段）、每次改动都值得写盘；
 * - 位置是**机器**记的、条目随打开过的文件数增长、每次光标停顿都会写。
 *
 * 合成一个键之后，任何一次写位置都会把设置整份重写一遍（反之亦然），
 * 而**存储里的内容坏掉时的代价也不一样**：位置坏了丢掉重记就行，
 * 设置坏了会让用户的三套主题偏好一起没。分开之后坏哪份丢哪份。
 *
 * ## 上限是必须的
 *
 * 条目数只增不减，而存储配额是有限的（WebView 一般 5–10 MB，但配额用尽时
 * `setItem` 会直接抛）。所以按**最后访问时间**淘汰，只留最近
 * `MAX_REMEMBERED_POSITIONS` 条 —— 用得上的永远只有最近打开的那几份。
 *
 * ## 所有读写都在 try/catch 里
 *
 * 与 `settings.ts` 同一条理由：localStorage 在某些环境下会直接抛
 * （隐私模式、配额用尽、企业策略禁用）。**位置记不住不该让应用起不来**，
 * 也不该打断用户 —— 它坏了只是「打开文档回到开头」。
 */

import type { EditorPosition } from './editor/types.js';
import { PERSISTED, loadLost, loaded, persistFailed } from './settings.js';
import type { LoadResult, PersistOutcome, StorageLike } from './settings.js';

/**
 * 一条位置记录。
 *
 * 比 `EditorPosition` 多一个 `at`：那是**淘汰顺序**的依据，属于存储层的事，
 * 不该混进内核契约（内核不关心自己这条记录什么时候被写下的）。
 */
export interface StoredPosition {
  readonly offset: number;
  readonly textLength: number;
  readonly scrollRatio: number;
  /** 最后一次更新的时间戳（毫秒）。只用于淘汰。 */
  readonly at: number;
}

/** 路径 → 位置。只读，改动一律返回新对象。 */
export type PositionMap = Readonly<Record<string, StoredPosition>>;

export const EMPTY_POSITIONS: PositionMap = {};

/**
 * 最多记住多少份文档的位置。
 *
 * 取 200 是「远大于任何人一天打开的文档数」与「存储里别攒垃圾」之间的折中。
 * 单条记录约 80 字节，200 条约 16 KB —— 写一次的开销可以忽略。
 */
export const MAX_REMEMBERED_POSITIONS = 200;

/**
 * 存储键。
 *
 * `v1` 与设置那个键是**各自独立**的版本：改了位置的格式只让位置失效，
 * 用户的主题与侧栏宽度照旧。两个键共用一个版本号的话，改一个就把另一个也废了。
 */
const STORAGE_KEY = 'levnote.positions.v1';

function isFiniteNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value);
}

/**
 * 校验一条记录。**任何一项不合法就整条丢掉**，不做逐字段回落。
 *
 * 与设置那边的策略刻意相反：设置里一个字段坏了，其余字段仍然该保住
 * （用户只是手改错了一个值）；而位置是一条**要么整体可信、要么没有意义**的记录 ——
 * 拿一个 `offset: 0` 配上正确的 `textLength` 去恢复，等于把光标悄悄移到文档开头，
 * 用户会以为文件被改了。丢掉它只是「回到顶部」，那是个诚实的失败。
 */
function parseEntry(value: unknown): StoredPosition | null {
  if (typeof value !== 'object' || value === null) return null;
  const record = value as Record<string, unknown>;

  const offset = record['offset'];
  const textLength = record['textLength'];
  const scrollRatio = record['scrollRatio'];
  if (!isFiniteNumber(offset) || offset < 0) return null;
  if (!isFiniteNumber(textLength) || textLength < 0) return null;
  if (!isFiniteNumber(scrollRatio)) return null;

  const at = record['at'];
  return {
    offset,
    textLength,
    scrollRatio: Math.min(1, Math.max(0, scrollRatio)),
    at: isFiniteNumber(at) ? at : 0,
  };
}

/**
 * 从存储里读全部位置。
 *
 * 形状不对、JSON 坏了、某几条记录不对 —— **一律回落到剩下的那些**，
 * 应用照常起来（ADR 0042：读有默认值可依，绝不白屏）。
 *
 * ## 但「回落了」这件事本身要说出去（ADR 0044）
 *
 * 位置丢了的代价比看起来大：文档打开后回到顶部，用户以为**文件被改了**
 * （第 71–77 行的 `parseEntry` 宁可整条丢也不给半条，就是怕这个）。
 * 而「我明明读到第三章中间」这句话，从此没有出处。
 *
 * 逐条点名比报一个总数有用 —— 丢的是**哪一份文档**的位置，用户是知道的。
 * 空键名（`''`）点不出名，只进计数。
 */
export function loadPositions(storage: StorageLike | null): LoadResult<PositionMap> {
  if (!storage) return loadLost(EMPTY_POSITIONS, 'no-storage');

  let raw: string | null;
  try {
    raw = storage.getItem(STORAGE_KEY);
  } catch {
    // 有存储却读不出来 = 之前记过的那 200 条这一次全读不到了（见 `loadLost`）
    return loadLost(EMPTY_POSITIONS, 'read-failed', [], 1);
  }
  // 从没记过位置（首次启动 / 关掉了「记住位置」）不是丢失。安静回落。
  if (raw === null || raw === '') return loaded(EMPTY_POSITIONS);

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    // 内容坏了：当成没有记忆，**不清空**（用户可能只是手改错了，
    // 下一次写就会覆盖掉）。清空反而多一次写操作、多一次失败机会。
    return loadLost(EMPTY_POSITIONS, 'corrupt', [], 1);
  }
  // 数组会把下标当成路径键（'0'、'1'……），报出来的是没人看得懂的名字 —— 当整份报废算
  if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
    return loadLost(EMPTY_POSITIONS, 'corrupt', [], 1);
  }

  const out: Record<string, StoredPosition> = {};
  const lostKeys: string[] = [];
  let lostCount = 0;
  for (const [path, value] of Object.entries(parsed as Record<string, unknown>)) {
    if (path === '') {
      // 键本身是空的：点不出名，只记个数。
      lostCount += 1;
      continue;
    }
    const entry = parseEntry(value);
    if (entry !== null) out[path] = entry;
    else {
      lostKeys.push(path);
      lostCount += 1;
    }
  }
  return lostCount === 0 ? loaded(out) : loadLost(out, 'corrupt', lostKeys, lostCount);
}

/**
 * 把整张位置表写回存储。
 *
 * 不抛：写不进去返回 `ok: false`，由调用方决定怎么说（ADR 0042）。
 * 与 `settings.ts` 的 `saveSettings` 是同一条纪律 —— 位置没存上确实不值得
 * **打断**用户（不该弹模态框），但不该**不说**：下次打开文档回到开头，
 * 而用户全程不知道「上次读到哪儿」是被丢掉的而不是他记错了。
 */
export function savePositions(storage: StorageLike | null, positions: PositionMap): PersistOutcome {
  if (!storage) return persistFailed('no-storage');
  try {
    storage.setItem(STORAGE_KEY, JSON.stringify(positions));
    return PERSISTED;
  } catch {
    return persistFailed('write-failed');
  }
}

/**
 * 记下一条位置，并按上限淘汰。
 *
 * 返回新对象而不是就地改：调用方（外壳）把它存在 ref 里，
 * 就地改会让「有没有变」变得看不出来，而将来的撤销/调试都需要看出这一点。
 *
 * `at` 可以注入，测试才不用去 mock `Date.now()`。
 */
export function rememberPosition(
  positions: PositionMap,
  path: string,
  position: EditorPosition,
  at: number = Date.now(),
): PositionMap {
  if (path === '') return positions;

  const next: Record<string, StoredPosition> = {
    ...positions,
    [path]: {
      offset: position.offset,
      textLength: position.textLength,
      scrollRatio: Math.min(1, Math.max(0, position.scrollRatio)),
      at,
    },
  };

  const paths = Object.keys(next);
  if (paths.length <= MAX_REMEMBERED_POSITIONS) return next;

  // 淘汰最旧的若干条。按 `at` 升序，丢掉前面的。
  // 排序而不是找最小值循环删除：条目数有上限，一次 O(n log n) 比 n 次 O(n) 便宜，
  // 而且能一次丢够（并发写入或旧版本留下的超大表都能收敛回上限）。
  const oldest = paths
    .map((key) => ({ key, at: next[key]?.at ?? 0 }))
    .sort((a, b) => a.at - b.at)
    .slice(0, paths.length - MAX_REMEMBERED_POSITIONS);

  for (const { key } of oldest) delete next[key];
  return next;
}

/**
 * 忘掉一份文档的位置。
 *
 * 用户主动要求「不要记住这个文件」时用 —— 目前还没有入口，
 * 但存储层的契约先定下来，免得将来为了加这个功能去改数据格式。
 */
export function forgetPosition(positions: PositionMap, path: string): PositionMap {
  if (!(path in positions)) return positions;
  const next = { ...positions };
  delete next[path];
  return next;
}
