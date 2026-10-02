/**
 * 编辑器模块的汇总入口 —— 依赖注入发生在这里。
 *
 * 依赖方向必须保持单向：
 *   模块（`src/modules/*`） → 契约（`src/core/editor/types.ts`） ← 外壳（`src/ui/*`）
 *
 * 也就是说外壳**不 import 任何具体内核**，它只 import 这个汇总入口拿到
 * 「兜底内核」和「注册动作」。将来接 CodeMirror / mind-elixir / draw.io，
 * 只需在这里多注册一个适配器，外壳一行都不用改。
 */

import type { EditorAdapter } from '../core/editor/types.js';
import { registerAdapter } from '../core/editor/registry.js';
import { codeAdapter } from './code/index.js';
import { markdownAdapter } from './markdown/index.js';
import { mindmapAdapter } from './mindmap/index.js';
import { plainTextAdapter } from './plaintext/index.js';

/**
 * 兜底内核。**不注册进注册表**，而是作为常量直接交给外壳 ——
 * 这样它不可能被某个内核的错误配置遮蔽掉，「打开任何文件都没反应」也就不可能发生。
 */
export const EDITOR_FALLBACK: EditorAdapter = plainTextAdapter;

let registered = false;

/**
 * 注册所有专门化内核。幂等，可以放心多次调用（React 严格模式会重复执行副作用）。
 *
 * **注册顺序即优先级**（见 `registry.ts`）。这里的顺序有实际含义：
 * 1. `markdown` 在前 —— `.md` 归富文本内核；
 * 2. `mindmap` 在中间 —— `.mindmap` 归导图内核；
 * 3. `code` 在**最后** —— 它是**备胎**，而且认得前两个认得的所有扩展名
 *    （`.md` 与 `.mindmap`）。所以它必须排在它们后面：备胎抢在正主前面，
 *    就会把所有文件都吃掉。
 *
 * 这条顺序决定了两件具体的事，而它们光看模块文件是推不出来的：
 * - 「带 YAML front matter 的 `.md` 用什么打开」—— Markdown 内核的 `canHandle`
 *   否决之后，注册表继续找，`code` 接住（一份有着色的 Markdown 源码）；
 * - 「坏掉的 `.mindmap` 用什么打开」—— 同理，导图内核否决之后由 `code` 接住
 *   （一份有着色的 JSON 源码），而不是掉到纯文本。
 *
 * **「打开任何文件都不能失败」的具体形态就是这条链**：
 * 正主 → 备胎 → 纯文本，一级比一级少，但一级都不会打不开。
 */
export function registerBuiltInEditors(): void {
  if (registered) return;
  registered = true;
  registerAdapter(markdownAdapter);
  registerAdapter(mindmapAdapter);
  registerAdapter(codeAdapter);
}
