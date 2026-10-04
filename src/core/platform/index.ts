/**
 * 平台适配入口。整个应用只有这一处判断「我在哪个环境里跑」。
 *
 * 编辑器内核、UI 组件都不许直接引用 `@tauri-apps/*`，一律通过 DocumentProvider，
 * 否则将来要加一条新平台线索（比如把文档内核抽出去做 CLI）就得全局改。
 */

import type { DocumentProvider } from '../document/types.js';
import { UNSUPPORTED_SEARCHER, type WorkspaceSearcher } from '../search/workspace.js';
import { createBrowserProvider } from './browser.js';
import {
  createDesktopProvider,
  createDesktopSearcher,
  onFilesChangedDesktop,
  watchFilesDesktop,
  type FsChangeBatch,
} from './desktop.js';

/** Tauri 2 在注入脚本里挂 `__TAURI_INTERNALS__`；用它判断比看 userAgent 可靠。 */
export function isDesktop(): boolean {
  return typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window;
}

export function createDocumentProvider(): DocumentProvider {
  return isDesktop() ? createDesktopProvider() : createBrowserProvider();
}

/**
 * 工作区检索的执行者。
 *
 * 浏览器预览拿不到它 —— 内存虚拟盘里没有目录树，硬造一个「能搜」的假象
 * 只会让「预览里搜不到」变成一个查不出来的谜。所以那里给的是
 * **明确拒绝**的执行者，面板据此显示一句说明，而不是一个点了没反应的面板。
 */
export function createWorkspaceSearcher(): WorkspaceSearcher {
  return isDesktop() ? createDesktopSearcher() : UNSUPPORTED_SEARCHER;
}

export { askConfirm } from './desktop.js';

// ---- 文件监听（P3 第 14 项，ADR 0029） -------------------------------------
//
// 与 createDocumentProvider 同一条分工：环境差异收口在这里。
// 浏览器预览的虚拟盘不会有外部改动，两个入口都给「无害」的空实现 ——
// `watchFiles` 立刻成功（返回「全部监视成功」）、`onFilesChanged` 给一个
// 永远不会触发的订阅。调用方不需要判断平台。

export type { FsChangeBatch } from './desktop.js';

/**
 * 把监视集合整体替换成 `paths`（工作区根目录 + 根外的打开文件）。
 * 返回没能监视上的路径（桌面端通常为空，浏览器端恒为空）。
 */
export function watchFiles(paths: readonly string[]): Promise<readonly string[]> {
  return isDesktop() ? watchFilesDesktop(paths) : Promise.resolve([]);
}

/** 订阅文件系统变更批次。返回解绑函数；浏览器预览里它永远不会被触发。 */
export function onFilesChanged(handler: (batch: FsChangeBatch) => void): Promise<() => void> {
  return isDesktop() ? onFilesChangedDesktop(handler) : Promise.resolve(() => undefined);
}
