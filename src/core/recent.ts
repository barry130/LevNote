/**
 * 最近打开的文件（A5）。
 *
 * 与位置记忆（`core/positions.ts`）同一条存储纪律：独立键、注入式
 * `StorageLike`、有上界（LRU —— 最久没动的让位）。它记的是**路径字符串**，
 * 不记文档内容：文件被删了就打不开（打开失败有横幅，由既有的 open 链路说）。
 *
 * 为什么不是 WorkspaceStore 的一部分：它要跨会话持久化，而 store 的职责是
 * 「这一刻开着什么」；「最近开过什么」是外壳层的偏好数据，与设置同层。
 * 外壳在活动标签页变化时记一笔（打开、恢复会话、切标签都会走到那里），
 * store 不需要知道这件事。
 */

import { PERSISTED, persistFailed } from './settings.js';
import type { PersistOutcome, StorageLike } from './settings.js';

export const RECENT_FILES_KEY = 'levnote.recentFiles';

/** 持久化的上界。菜单里只展示更少的一条（见 definitions.ts 的菜单槽数）。 */
export const RECENT_FILES_MAX = 15;

/**
 * 读出最近打开列表。损坏的存储按「没有记录」处理 ——
 * 这份数据随时可以重建，不值得为它报错。没有存储（`null`，隐私模式等）同理。
 *
 * `getItem` 自己也包了 try/catch：`localStorage` 在某些环境下**读**都会抛
 * （隐私模式 / 企业策略）。不包的话，这次抛会从 `App` 的 `useState` 初始化器
 * 里一路穿出来 —— 那就是**启动即白屏**，比「记不住」严重得多（ADR 0042）。
 * 读失败有默认值可依（`[]`：这份数据随时能重建），所以这里安静回落，
 * 与写失败要出声是两回事。
 */
export function loadRecentFiles(storage: StorageLike | null): readonly string[] {
  let raw: string | null;
  try {
    raw = storage?.getItem(RECENT_FILES_KEY) ?? null;
  } catch {
    return [];
  }
  if (raw === null) return [];
  try {
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((item): item is string => typeof item === 'string' && item !== '');
  } catch {
    return [];
  }
}

/**
 * 把列表写回存储。不抛：写不进去返回 `ok: false`（ADR 0042）。
 *
 * 与 `saveSettings` / `savePositions` 同一条纪律。记不住最近打开**不值得打断**
 * 用户，但也不是「没有错误」—— 菜单里的「最近打开下一次空了」应当有解释。
 */
export function saveRecentFiles(storage: StorageLike | null, paths: readonly string[]): PersistOutcome {
  if (!storage) return persistFailed('no-storage');
  try {
    storage.setItem(RECENT_FILES_KEY, JSON.stringify([...paths]));
    return PERSISTED;
  } catch {
    return persistFailed('write-failed');
  }
}

/**
 * 记一笔「刚打开过」。纯函数：**没有变化时返回同一个数组引用**，
 * 调用方据它决定要不要 setState / 落盘 —— 不变就是不变，不多写一次。
 */
export function recordRecentList(paths: readonly string[], path: string): readonly string[] {
  const next = [path, ...paths.filter((item) => item !== path)];
  if (next.length > RECENT_FILES_MAX) next.length = RECENT_FILES_MAX;
  return next[0] === paths[0] && next.length === paths.length ? paths : next;
}

/**
 * 清空（设置或菜单的「清空最近打开」）。
 *
 * 它是一次**用户动作**，不是后台记账，所以写不进去必须让用户知道 ——
 * 点了「清空」而菜单里那几条还在，那是「点了没反应」。返回 `PersistOutcome`
 * 让调用方决定怎么说（ADR 0042）。
 */
export function clearRecentFiles(storage: StorageLike | null): PersistOutcome {
  if (!storage) return persistFailed('no-storage');
  try {
    storage.setItem(RECENT_FILES_KEY, '[]');
    return PERSISTED;
  } catch {
    return persistFailed('write-failed');
  }
}
