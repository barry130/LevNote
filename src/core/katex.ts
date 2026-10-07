/**
 * 把一段 TeX 渲染成 HTML —— 编辑器与导出**共用这一处**。
 *
 * ## 为什么必须共用
 *
 * 编辑器里看到的公式和导出文件里的公式必须长得一样。两边各写一次 `renderToString`
 * 调用，迟早会在某次调整里只改了其中一处（选项、displayMode、错误处理），
 * 于是「编辑器里是好的，导出的文件里是红的」。这与代码高亮共用
 * `core/highlight/hljs.ts` 是同一条理由。
 *
 * ## 为什么 KaTeX 是**动态**加载的
 *
 * `katex.min.js` 未压缩 272 KB。项目主包已经 1.7 MB，再静态引进来会让**每一个**
 * 打开轻墨的人先下载 272 KB —— 哪怕他从来不写公式。所以这里用 `import()`：
 * 只有文档里真的出现公式时，这个 chunk 才会被取下来。
 *
 * 代价是渲染变成**异步**的。两处调用方本来就能接受异步（NodeView 的 DOM 由我们
 * 自己管；导出整条链路已经是 async），所以这个代价是零。
 *
 * ## 为什么不用 `render` 而用 `renderToString`
 *
 * `render` 会自己往一个 DOM 元素里写。导出那边根本没有 DOM 可写（产物是一段字符串），
 * 编辑器这边则希望**由自己决定什么时候把结果放进 DOM**（异步回来时那个元素可能
 * 已经被销毁了）。返回字符串，把「放进去」留给调用方，两边的需求都能满足。
 */

import type katex from 'katex';
import { t } from './i18n/index.js';
import type { KatexOptions } from 'katex';

/**
 * KaTeX 模块对象本身的类型。
 *
 * 只能这样取：`katex` 是**动态**加载的，所以不能写成 `import katex from 'katex'`
 * （那会把 272 KB 拉进主包，正是这个文件存在的理由）。而
 * `typeof import('katex').default` 这种内联 `import()` 类型被
 * `consistent-type-imports` 禁止，于是先做一次 **type-only** 的默认导入
 * （编译后会被完全擦除），再用 `typeof` 把它变成类型。
 */
type KatexModule = typeof katex;

/** 渲染成功：`html` 是 KaTeX 产出的完整片段（含给读屏软件用的 MathML）。 */
export interface MathRenderOk {
  readonly ok: true;
  readonly html: string;
}

/**
 * 渲染失败：`message` 是**可以直接显示给用户**的一句话。
 *
 * 失败绝不吞掉。用户写错了一个命令，他需要看到「哪里错了」，而不是一个空白 ——
 * 空白会被理解成「公式没渲染出来」或者「编辑器坏了」，而真正的原因
 * （比如 `\foo` 这个命令不存在）只有 KaTeX 知道。
 */
export interface MathRenderFail {
  readonly ok: false;
  readonly message: string;
}

export type MathRender = MathRenderOk | MathRenderFail;

/**
 * KaTeX 的加载。
 *
 * 缓存 **Promise** 而不是缓存模块本身：并发的多个公式会同时触发加载，
 * 缓存模块的话第二个调用会在第一个 `await` 完成之前看到 `null`，
 * 于是把同一个 chunk 取两遍。缓存 Promise 天然去重。
 *
 * **失败不缓存**：一次加载失败（磁盘读不到、打包产物不完整）如果被记成
 * 「永远失败」，用户就算把文件修好了也得重启应用。清掉之后下一次渲染
 * 会重新尝试 —— 反正失败的路径本来就是罕见的。
 */
let katexPromise: Promise<KatexModule> | null = null;

function loadKatex(): Promise<KatexModule> {
  katexPromise ??= import('katex')
    .then((module) => module.default)
    .catch((thrown: unknown) => {
      katexPromise = null;
      throw thrown;
    });
  return katexPromise;
}

/**
 * KaTeX 的选项。
 *
 * - `throwOnError: true` —— 让错误**抛出来**，由我们接住、翻译成人话再显示。
 *   默认值是 `false`，它会把错误信息用红字画进公式里：颜色是 KaTeX 定的、
 *   文案是英文的，而且和「公式本身长什么样」混在同一个位置上，用户分不清
 *   哪部分是公式、哪部分是报错。接住它，我们就能把「源码 + 原因」分开摆。
 * - `strict: false` —— 关掉 KaTeX 对「LaTeX 标准符合性」的警告。
 *   它**不改变渲染结果**，只决定要不要往控制台打日志；关掉是为了不让
 *   控制台被 `\text{中文}` 这类「非标准但完全可用」的写法刷满。
 * - `trust` 保持默认的 `false` —— 于是 `\href` / `\url` / `\includegraphics`
 *   一律不可用。这是**安全边界**：公式的内容来自文档，而文档可能来自别人。
 *   打开它等于允许一份 Markdown 在渲染时构造出链接或去取外部资源。
 * - `maxExpand` / `maxSize` 保持默认 —— 它们挡的是「宏展开到内存爆掉」
 *   这类恶意输入（`\def` 自嵌套）。
 */
const OPTIONS = {
  throwOnError: true,
  strict: false,
  trust: false,
} satisfies KatexOptions;

/**
 * 渲染一段 TeX。
 *
 * `displayMode` 为 true 时是块级公式（居中、上下留白、`\sum` 的下标跑到正下方），
 * false 时是行内公式。这个开关由节点的类型决定，不由公式内容决定。
 */
export async function renderTex(tex: string, displayMode: boolean): Promise<MathRender> {
  let katex: KatexModule;
  try {
    katex = await loadKatex();
  } catch (thrown) {
    // 取不到 chunk（打包产物不完整、文件被删）。这不是用户的错，
    // 但用户必须知道「公式显示不出来」而不是看到一片空白
    return { ok: false, message: t('core.render.katexLoadFailed', { reason: describe(thrown) }) };
  }

  try {
    return { ok: true, html: katex.renderToString(tex, { ...OPTIONS, displayMode }) };
  } catch (thrown) {
    return { ok: false, message: describeMathError(thrown) };
  }
}

/**
 * 把 KaTeX 抛出来的东西翻译成一句给用户看的话。
 *
 * KaTeX 的 `ParseError.message` 形如
 * `KaTeX parse error: Undefined control sequence: \foo at position 1: \̲f̲o̲o̲{`，
 * 里面混着「错误定位标记」（那个 `̲` 是组合下划线，用来指出出错位置）。
 * 直接显示会是一团乱码，所以：
 *
 * - 有 `rawMessage` 就用它（`Undefined control sequence: \foo`）——
 *   干净、说明了原因；
 * - 再加上 `position`，用户才知道是**第几个字符**出的问题。
 *
 * 认不出来的一律如实带上原文，不编一句「公式有误」—— 那句话没有任何信息量。
 */
function describeMathError(thrown: unknown): string {
  if (thrown instanceof Error) {
    const raw = (thrown as { rawMessage?: unknown }).rawMessage;
    const position = (thrown as { position?: unknown }).position;
    if (typeof raw === 'string' && raw !== '') {
      const at = typeof position === 'number' ? t('core.render.katexAt', { position: position + 1 }) : '';
      return `${raw}${at}`;
    }
    return thrown.message;
  }
  return String(thrown);
}

/** 兜底：`String(thrown)` 对非 Error 也能给出一句话，而不是 `[object Object]`。 */
function describe(thrown: unknown): string {
  return thrown instanceof Error ? thrown.message : String(thrown);
}
