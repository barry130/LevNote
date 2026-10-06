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

import type { StorageLike } from './settings.js';

export const RECENT_FILES_KEY = 'qingmo.recentFiles';

/** 持久化的上界。菜单里只展示更少的一条（见 definitions.ts 的菜单槽数）。 */
export const RECENT_FILES_MAX = 15;

/**
 * 读出最近打开列表。损坏的存储按「没有记录」处理 ——
 * 这份数据随时可以重建，不值得为它报错。没有存储（`null`，隐私模式等）同理。
 */
export function loadRecentFiles(storage: StorageLike | null): readonly string[] {
  const raw = storage?.getItem(RECENT_FILES_KEY) ?? null;
  if (raw === null) return [];
  try {
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((item): item is string => typeof item === 'string' && item !== '');
  } catch {
    return [];
  }
}

/** 把列表写回存储。没有存储时安静跳过 —— 记不住，但不是错误。 */
export function saveRecentFiles(storage: StorageLike | null, paths: readonly string[]): void {
  storage?.setItem(RECENT_FILES_KEY, JSON.stringify([...paths]));
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

/** 清空（设置或菜单的「清空最近打开」）。 */
export function clearRecentFiles(storage: StorageLike | null): void {
  storage?.setItem(RECENT_FILES_KEY, '[]');
}
