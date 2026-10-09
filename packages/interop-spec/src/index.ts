/**
 * @levnote/interop-spec —— 块级互操作规范 v0。
 *
 * 这个包**不能**依赖 React / Tauri / DOM：编辑器内核、CLI、未来的服务端校验
 * 都要能直接复用它。因此：
 * - 只导出纯函数与类型；
 * - 解析器接收字符串、返回字符串与结构，不碰文件系统；
 * - 用 `.js` 后缀写相对导入（NodeNext 解析），交给打包器处理。
 */

export * from './types.js';
export { parseDocBlocks, validateDocBlocks, escapesRoot, splitLines } from './parse.js';
export {
  createDocBlock,
  detectLineEnding,
  formatAttrs,
  generateBlockId,
  neededFenceLength,
  serializeDocBlock,
  spliceBlocks,
} from './serialize.js';
export type { LineEnding, SerializeOptions } from './serialize.js';
