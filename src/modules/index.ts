/**
 * 编辑器模块的汇总入口 —— 依赖注入发生在这里。
 *
 * 依赖方向必须保持单向：
 *   模块（`src/modules/*`） → 契约（`src/core/editor/types.ts`） ← 外壳（`src/ui/*`）
 *
 * 也就是说外壳**不 import 任何具体内核**，它只 import 这个汇总入口拿到
 * 「兜底内核」和「注册动作」。将来接 CodeMirror / 别的导图内核 / 别的图纸渲染器，
 * 只需在这里多注册一个适配器，外壳一行都不用改。
 *
 * ## 三个内核走**动态 import**（P3 第 18 项）
 *
 * Milkdown（连带 ProseMirror）、CodeMirror（连带十份 Lezer 语法表）、
 * mind-elixir 加起来占了主包的大头，而外壳首屏（活动栏、标签栏、设置页）
 * 一行内核代码都用不到。所以专门化内核改成 `import()` —— 它们各自落进
 * 独立的 chunk，与外壳并行加载；兜底内核（plaintext）刻意**保持静态**：
 * 「打开任何文件都不能失败」这条承诺不能自己也是异步的。
 *
 * **注册顺序与加载成功与否是两件独立的事**：await 按固定次序逐个进行，
 * 谁先谁后与模块解析、网络时序无关；某个内核加载失败只影响它自己
 * （注册表里没有它，备胎链自动降级），不影响其余内核与启动本身。
 * 失败的原因存在 `editorRegistrationFailures()` 里，外壳负责说出来 ——
 * 「少了一个内核」是降级，降级不许静默（硬约束 2）。
 */

import type { EditorAdapter } from '../core/editor/types.js';
import { registerAdapter } from '../core/editor/registry.js';
import { plainTextAdapter } from './plaintext/index.js';

/**
 * 兜底内核。**不注册进注册表**，而是作为常量直接交给外壳 ——
 * 这样它不可能被某个内核的错误配置遮蔽掉，「打开任何文件都没反应」也就不可能发生。
 */
export const EDITOR_FALLBACK: EditorAdapter = plainTextAdapter;

/** 一个内核没能加载出来的原因。外壳要在界面上说出来。 */
export interface KernelLoadFailure {
  readonly id: string;
  readonly reason: string;
}

let failures: KernelLoadFailure[] = [];

/** 这次启动里没能加载的内核（空数组 = 全部就位）。幂等，重复读无副作用。 */
export function editorRegistrationFailures(): readonly KernelLoadFailure[] {
  return failures;
}

/** 把非 Error 的抛出物也变成一句可读的话。 */
function describe(thrown: unknown): string {
  return thrown instanceof Error ? thrown.message : String(thrown);
}

let registration: Promise<void> | null = null;

/**
 * 注册所有专门化内核。幂等 —— 并发调用共享同一个 Promise
 * （React 严格模式会重复执行副作用）。
 *
 * **注册顺序即优先级**（见 `registry.ts`）。这里的顺序由 await 的先后钉死，
 * 与动态 import 的解析时序无关：
 * 1. `markdown` 在前 —— `.md` 归富文本内核；
 * 2. `mindmap` 居中 —— `.mindmap` 归导图内核；
 * 3. `drawio` 在其后 —— `.drawio` 归图纸内核（ADR 0048）；
 * 4. `code` 在**最后** —— 它是**备胎**，而且认得前三个认得的所有扩展名
 *    （`.md` / `.mindmap` / `.drawio`）。所以它必须排在它们后面：备胎抢在正主前面，
 *    就会把所有文件都吃掉。
 *
 * 这条顺序决定了两件具体的事，而它们光看模块文件是推不出来的：
 * - 「带 YAML front matter 的 `.md` 用什么打开」—— Markdown 内核的 `canHandle`
 *   否决之后，注册表继续找，`code` 接住（一份有着色的 Markdown 源码）；
 * - 「坏掉的 `.mindmap` 用什么打开」—— 同理，导图内核否决之后由 `code` 接住
 *   （一份有着色的 JSON 源码），而不是掉到纯文本。
 *
 * **`.drawio` 是这条顺序的第三道考验**：代码内核的 `CODE_EXTENSIONS` 里
 * **必须**留着 `.drawio`（`drawio` 内核的 `sourceView` 就指着它），
 * 所以「图纸内核能不能认领」与「源码视图还能不能打开」是两个必须同时成立的事 ——
 * 少任何一个，`.drawio` 都会掉回「一份 XML 源码」。改动任一侧时，
 * 先看一眼 `src/modules/wiring.test.ts` 里那几条断言。
 *
 * **「打开任何文件都不能失败」的具体形态就是这条链**：
 * 正主 → 备胎 → 纯文本，一级比一级少，但一级都不会打不开。
 */
export function registerBuiltInEditors(): Promise<void> {
  registration ??= (async () => {
    failures = [];

    const markdown = await import('./markdown/index.js').then(
      (module) => module,
      (thrown: unknown) => {
        failures.push({ id: 'markdown', reason: describe(thrown) });
        return null;
      },
    );
    if (markdown !== null) registerAdapter(markdown.markdownAdapter);

    const mindmap = await import('./mindmap/index.js').then(
      (module) => module,
      (thrown: unknown) => {
        failures.push({ id: 'mindmap', reason: describe(thrown) });
        return null;
      },
    );
    if (mindmap !== null) registerAdapter(mindmap.mindmapAdapter);

    const drawio = await import('./drawio/index.js').then(
      (module) => module,
      (thrown: unknown) => {
        failures.push({ id: 'drawio', reason: describe(thrown) });
        return null;
      },
    );
    if (drawio !== null) registerAdapter(drawio.drawioAdapter);

    const code = await import('./code/index.js').then(
      (module) => module,
      (thrown: unknown) => {
        failures.push({ id: 'code', reason: describe(thrown) });
        return null;
      },
    );
    if (code !== null) registerAdapter(code.codeAdapter);
  })();
  return registration;
}
