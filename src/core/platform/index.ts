/**
 * 平台适配入口。整个应用只有这一处判断「我在哪个环境里跑」。
 *
 * 编辑器内核、UI 组件都不许直接引用 `@tauri-apps/*`，一律通过 DocumentProvider，
 * 否则将来要加一条新平台线索（比如把文档内核抽出去做 CLI）就得全局改。
 */

import type { DocumentProvider } from '../document/types.js';
import { createBrowserProvider } from './browser.js';
import { createDesktopProvider } from './desktop.js';

/** Tauri 2 在注入脚本里挂 `__TAURI_INTERNALS__`；用它判断比看 userAgent 可靠。 */
export function isDesktop(): boolean {
  return typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window;
}

export function createDocumentProvider(): DocumentProvider {
  return isDesktop() ? createDesktopProvider() : createBrowserProvider();
}

export { askConfirm } from './desktop.js';
