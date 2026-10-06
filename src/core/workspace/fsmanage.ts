/**
 * 工作区文件管理（新建 / 重命名 / 删除）的**能力接口**。
 *
 * 与检索（`core/search/workspace.ts` 的 `WorkspaceSearcher`）同一条分工：
 * 这组能力**依赖真实的目录树**，而浏览器预览的内存虚拟盘是扁平的 ——
 * 硬造一个「预览里也能建文件」的假象，只会把「点了没反应」变成查不出来的谜。
 * 所以它是独立的小接口：桌面端给真实现（`createDesktopFileManager`），
 * 浏览器端给**明确拒绝**的 `UNSUPPORTED_FILE_MANAGER`，侧栏据此禁用按钮并说明。
 *
 * 三条语义约束（在 Rust 命令与 store 编排层各守一遍，这里写总纲）：
 * 1. **绝不覆盖**：create / rename 的目标已存在时明确拒绝（CONFLICT），
 *    由调用方把原因说给用户；
 * 2. **删除不可撤销**：调用方必须先向用户确认（硬约束 2 —— 不静默），
 *    接口层不做确认 —— 「问不问用户」是界面的事，「删的是什么」是这里的事；
 * 3. **不猜路径**：所有路径都是完整绝对路径，由调用方用 `core/path.ts` 拼 ——
 *    这个接口不做任何字符串解析。
 */

import { AppError } from '../document/types.js';

export interface WorkspaceFileManager {
  /** 浏览器预览为 false：UI 应把文件管理入口显示为禁用并说明原因。 */
  readonly available: boolean;
  /** 创建空文件。已存在时拒绝 —— 绝不静默覆盖。 */
  createFile(path: string): Promise<void>;
  /** 创建目录（单层，父目录必须已存在）。已存在时拒绝。 */
  createDirectory(path: string): Promise<void>;
  /** 重命名 / 移动。目标已存在时拒绝 —— 绝不覆盖。 */
  renamePath(from: string, to: string): Promise<void>;
  /**
   * 删除文件或目录（目录递归删除其中全部内容）。**不可撤销**。
   * 调用方必须先向用户确认过；接口层不确认、不进回收站。
   */
  deletePath(path: string): Promise<void>;
}

function notSupported(): AppError {
  return new AppError(
    'NOT_SUPPORTED',
    '当前环境不支持管理文件（浏览器预览没有真实文件系统，请用桌面版）',
  );
}

export const UNSUPPORTED_FILE_MANAGER: WorkspaceFileManager = {
  available: false,
  createFile() {
    return Promise.reject(notSupported());
  },
  createDirectory() {
    return Promise.reject(notSupported());
  },
  renamePath() {
    return Promise.reject(notSupported());
  },
  deletePath() {
    return Promise.reject(notSupported());
  },
};
