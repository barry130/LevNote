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
 * 2. `code` 在后 —— 它认得 `.md`，但只作为备胎：Markdown 内核的 `canHandle`
 *    否决（front matter 文件）之后，注册表会继续找下一个候选，这时它接住。
 *
 * 把这条顺序写下来，是因为它决定了「front matter 文件用什么打开」这个具体行为，
 * 而那个行为光看两个模块的文件是推不出来的。
 */
export function registerBuiltInEditors(): void {
  if (registered) return;
  registered = true;
  registerAdapter(markdownAdapter);
  registerAdapter(codeAdapter);
}
