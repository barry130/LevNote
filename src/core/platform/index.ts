/**
 * 平台适配入口。整个应用只有这一处判断「我在哪个环境里跑」。
 *
 * 编辑器内核、UI 组件都不许直接引用 `@tauri-apps/*`，一律通过 DocumentProvider，
 * 否则将来要加一条新平台线索（比如把文档内核抽出去做 CLI）就得全局改。
 */

import type { DocumentProvider } from '../document/types.js';
import { t } from '../i18n/index.js';
import { UNSUPPORTED_SEARCHER, type WorkspaceSearcher } from '../search/workspace.js';
import { UNSUPPORTED_FILE_MANAGER, type WorkspaceFileManager } from '../workspace/fsmanage.js';
import { createBrowserProvider, toggleFullscreenBrowser } from './browser.js';
import {
  createDesktopFileManager,
  createDesktopProvider,
  createDesktopSearcher,
  onFilesChangedDesktop,
  openExternalDesktop,
  printHtmlDesktop,
  toggleFullscreenDesktop,
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

/**
 * 工作区文件管理的执行者（新建 / 重命名 / 删除）。
 *
 * 与检索同一条分工：能力依赖真实目录树，浏览器预览给**明确拒绝**的
 * `UNSUPPORTED_FILE_MANAGER` —— 侧栏据此禁用文件管理按钮，而不是点了没反应。
 */
export function createWorkspaceFileManager(): WorkspaceFileManager {
  return isDesktop() ? createDesktopFileManager() : UNSUPPORTED_FILE_MANAGER;
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

// ---- 打印 / 导出 PDF（P3 第 16 项，ADR 0030） ------------------------------
//
// 与检索同一条分工：浏览器预览没有「打印子窗口」这个能力，给**明确拒绝**
// 而不是点了没反应。命令面板里这个命令在浏览器里是禁用的，这里再拦一道
// 是因为动作本身可被别的入口调用。

export function printHtml(html: string, title: string): Promise<void> {
  if (!isDesktop()) {
    return Promise.reject(new Error(t('core.print.desktopOnly')));
  }
  return printHtmlDesktop(html, title);
}

// ---- 打开外部链接 / 文件（ADR 0040） ----------------------------------------
//
// 浏览器预览里「用系统默认程序打开」没有对应物：那里连真实文件系统都没有。
// 所以给的是**明确拒绝**而不是静默无事 —— 与上面检索 / 打印同一条分工。

/**
 * 用系统默认程序打开（网页 → 浏览器，文件 → 关联程序）。
 * 失败时 reject —— 调用方必须把原因说给用户听（硬约束 2）。
 */
export function openExternal(target: string): Promise<void> {
  if (!isDesktop()) {
    return Promise.reject(new Error(t('core.openExternal.desktopOnly')));
  }
  return openExternalDesktop(target);
}

// ---- 全屏 ------------------------------------------------------------------
//
// 两个平台都**能**做这件事，所以这里不需要拒绝分支 —— 差异只在实现源
// （桌面是 Tauri 的窗口级全屏，浏览器是 Fullscreen API 的视口级全屏）。

export function toggleFullscreen(): Promise<void> {
  return isDesktop() ? toggleFullscreenDesktop() : toggleFullscreenBrowser();
}
