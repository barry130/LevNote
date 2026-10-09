/**
 * 把一段 Mermaid 源码渲染成 SVG —— 编辑器与导出**共用这一处**。
 *
 * ## 为什么必须共用
 *
 * 与 KaTeX（`core/katex.ts`）同一条理由：编辑器里看到的图和导出文件里的图
 * 必须是同一张。两边各写一次 `mermaid.render`，迟早会在某次调整里只改了其中
 * 一处（配置、容器、错误处理），于是「编辑器里是好的，导出的是空的」。
 *
 * ## 为什么 mermaid 是**动态**加载的
 *
 * `mermaid.core.mjs` 本身只有 53 KB，但它一被加载就会把几十个图种
 * （flowchart / sequence / class / gitGraph / …）和各自那套布局算法一起拉进来 ——
 * 实测 `node_modules/mermaid/dist` 里非 sourcemap 的部分是 12.7 MB / 271 个文件。
 * 静态引入会让**每一个**打开轻笺的人先付这笔钱，哪怕这篇文档一张图都没有。
 * 所以走 `import()`：只有文档里真的出现 mermaid 代码块时，这部分才会被取下来。
 *
 * 代价是渲染变成异步的。两处调用方本来就是异步的（NodeView 的 DOM 由我们自己管，
 * 导出的整条链路已经是 async），所以这个代价是零。
 *
 * ## 为什么必须自己准备容器
 *
 * `mermaid.render(id, text, container)` 的第三个参数**不能省**。省了的话
 * mermaid 走的是另一条分支（`mermaid.core.mjs` 里 `root = select("body")`），
 * 会把量尺寸用的临时 `<div>` / `<svg>` **直接挂到页面的 body 上**：
 *
 * - 在编辑器里，那些节点会真的出现在 DOM 里（可能抢焦点、可能让滚动条跳一下）；
 * - 在导出时更糟 —— 它污染的是**真实页面**，而我们只要它返回的那段字符串。
 *
 * 传了容器，mermaid 就只在容器里写（`root = select(svgContainingElement)`），
 * 并且渲染结束后自己把临时节点摘掉（`removeTempElements()`）。
 *
 * ## 为什么容器必须真的在文档里、真的参与布局
 *
 * mermaid 靠 `getBBox()` / `getComputedTextLength()` 量文字尺寸来决定节点框多大。
 * 这两个 API 只在元素**真的被渲染**时才给得出数字：`display: none` 或者游离
 * （detached）节点会得到全 0（Firefox 上直接抛错），画出来是一片挤在一起、
 * 互相重叠的框。所以容器用 `position: fixed` 挪到屏幕外 —— 它在布局里，但看不见。
 *
 * ## 为什么一次渲染一个容器
 *
 * mermaid 内部按 `id` 找它自己塞进去的临时节点（`select('#d' + id)`），
 * 而这个查找是在**整个 document** 上做的。两次渲染共用一个 id 会互相踩。
 * 这里给每次渲染一个自增 id，容器也是一次性的（`finally` 里摘掉），谁也不欠谁。
 */

import type { MermaidConfig, RenderResult } from 'mermaid';
import type { ResolvedTheme } from './theme/theme.js';
import { t } from './i18n/index.js';

/**
 * 我们真正用到的那部分 mermaid。
 *
 * 做成**结构类型**而不是 `typeof mermaid`：后者是完整的 `Mermaid` 接口
 * （`run` / `init` / `registerExternalDiagrams` / …十来个成员），测试里为了
 * 造一个假实现得把它们全凑齐，而那些成员我们一个都不用。
 * 真实的 `mermaid.default` **结构上满足**这个接口，所以不需要 `as` 断言。
 */
export interface MermaidLike {
  readonly initialize: (config: MermaidConfig) => void;
  readonly render: (id: string, text: string, container: Element) => Promise<RenderResult>;
}

/** 渲染成功：`svg` 是 mermaid 产出的完整 `<svg>` 片段，自带 `<style>`。 */
export interface MermaidRenderOk {
  readonly ok: true;
  readonly svg: string;
}

/**
 * 渲染失败：`message` 是**可以直接显示给用户**的一句话。
 *
 * 失败绝不吞掉 —— 与 KaTeX 那一侧同一个判据：用户写错了语法，他需要看到
 * 「哪里错了」，而不是一个空白。空白会被理解成「图表没画出来」或者
 * 「编辑器坏了」，而真正的原因（第几行多了个箭头）只有 mermaid 知道。
 */
export interface MermaidRenderFail {
  readonly ok: false;
  readonly message: string;
}

export type MermaidRender = MermaidRenderOk | MermaidRenderFail;

/**
 * 渲染需要的两件外部能力，做成注入的。
 *
 * 两个都必须能换掉：`load` 换掉之后测试不必真的去加载 12 MB 的 mermaid；
 * `withContainer` 换掉之后……其实它不需要换，但**它做的事必须和 load 分开** ——
 * 「加载」是一次性的（有缓存），「准备容器」是每次渲染都要做的，
 * 混在一起就没法单独钉住「渲染失败时容器有没有被摘掉」。
 */
export interface MermaidIo {
  readonly load: () => Promise<MermaidLike>;
  readonly withContainer: <T>(run: (container: HTMLElement) => Promise<T>) => Promise<T>;
}


// ---------------------------------------------------------------------------
// 加载
// ---------------------------------------------------------------------------

/**
 * 把「加载一次」这个语义单独做出来。
 *
 * 缓存 **Promise** 而不是模块本身：一篇文档里有三张图时会同时触发加载，
 * 缓存模块的话第二、第三个调用会在第一个 `await` 完成之前看到 `null`，
 * 于是把同一批 chunk 取三遍。缓存 Promise 天然去重。
 *
 * **失败不缓存**：一次加载失败（打包产物不完整、磁盘读不到）如果被记成
 * 「永远失败」，用户就算把文件修好了也得重启应用。清掉之后下一次渲染会重新
 * 尝试 —— 反正失败的路径本来就是罕见的。
 *
 * 抽成工厂而不是写死在 `loadMermaid` 里，是因为这条「缓存什么、什么时候清」
 * 的规则正是最容易写错的那一处，而直接用 `import('mermaid')` 就没法在测试里
 * 验证它（那会真的去加载 12 MB 的东西）。
 */
export function cachedLoader(load: () => Promise<MermaidLike>): () => Promise<MermaidLike> {
  let pending: Promise<MermaidLike> | null = null;
  return () => {
    pending ??= load().catch((thrown: unknown) => {
      pending = null;
      throw thrown;
    });
    return pending;
  };
}

const loadMermaid = cachedLoader(async () => (await import('mermaid')).default);

// ---------------------------------------------------------------------------
// 配置
// ---------------------------------------------------------------------------

/**
 * mermaid 的选项。
 *
 * - `startOnLoad: false` —— 我们只调 `render`，不让 mermaid 自己去扫页面上的
 *   `.mermaid` 元素。留着它等于在 `render` 之外又开了一个渲染入口，哪天页面上
 *   真出现一个 `.mermaid` 元素就会被画两次。
 * - `suppressErrorRendering: true` —— 语法错时不要画它自己那张「错误图」。
 *   反正那张图我们也不会用（错误路径下 mermaid 照样会 throw，返回的 svg
 *   被我们丢掉，显示的是源码 + 原因），画它只是白费一遍布局。
 * - `securityLevel: 'strict'` —— 与 KaTeX 的 `trust: false` 是同一条边界：
 *   图表源码来自文档，而文档可能来自别人。`strict` 下 mermaid 会用 DOMPurify
 *   清洗产出的 SVG，并且不执行图表里的 `click` 指令。**不要用 `loose`** ——
 *   那会连清洗都跳过。
 * - `theme` —— 跟随应用的明暗。代价记在 ADR 0020：这是 mermaid 的**内置配色**，
 *   不是轻笺的配色，两者只是明暗一致、色号不同。
 */
function configFor(theme: ResolvedTheme): MermaidConfig {
  return {
    startOnLoad: false,
    suppressErrorRendering: true,
    securityLevel: 'strict',
    theme: theme === 'dark' ? 'dark' : 'default',
  };
}

/**
 * `initialize` 是有副作用且不算便宜的（它会重算一遍 theme 变量、注册图种），
 * 所以只在**模块变了**或**主题变了**时才调。
 *
 * 记的是「上次配置给谁配的」而不是一个布尔量：加载失败过一次之后
 * `load()` 会返回一个**新的**模块对象，那时必须重新 initialize ——
 * 否则新模块带着默认配置上场，图表画出来是浅色的。
 */
let initializedFor: MermaidLike | null = null;
let initializedTheme: ResolvedTheme | null = null;

function ensureInitialized(mermaid: MermaidLike, theme: ResolvedTheme): void {
  if (initializedFor === mermaid && initializedTheme === theme) return;
  mermaid.initialize(configFor(theme));
  initializedFor = mermaid;
  initializedTheme = theme;
}

// ---------------------------------------------------------------------------
// 渲染
// ---------------------------------------------------------------------------

/** 每次渲染一个自增 id —— 理由见文件头「为什么一次渲染一个容器」。 */
let renderSeq = 0;

/**
 * 渲染一段 Mermaid 源码。
 *
 * `theme` 决定图表的明暗，由调用方从应用的当前主题推导（编辑器读 `data-theme`，
 * 导出读导出时的主题）。这里**不设默认值**：默认一个值就等于允许某处调用方
 * 「忘了传」而安静地画出相反明暗的图。
 */
export async function renderMermaid(
  text: string,
  theme: ResolvedTheme,
  io: MermaidIo = DEFAULT_IO,
): Promise<MermaidRender> {
  // 空源码不值得惊动 mermaid：它只会回一句 "No diagram type detected"，
  // 而那句话对「我还没开始写」这个事实没有任何信息量。更实际的是 ——
  // 不短路的话，一个空的代码块就会把 mermaid 那一堆 chunk 拉下来
  if (text.trim() === '') return { ok: false, message: t('core.render.mermaidEmpty') };

  let mermaid: MermaidLike;
  try {
    mermaid = await io.load();
  } catch (thrown) {
    // 取不到 chunk（打包产物不完整、文件被删）。这不是用户的错，
    // 但用户必须知道「图画不出来」而不是看到一片空白
    return { ok: false, message: t('core.render.mermaidLoadFailed', { reason: describe(thrown) }) };
  }

  ensureInitialized(mermaid, theme);

  renderSeq += 1;
  const id = `qm-mermaid-${renderSeq}`;

  try {
    const svg = await io.withContainer(async (container) => {
      const result = await mermaid.render(id, text, container);
      return result.svg;
    });
    return { ok: true, svg };
  } catch (thrown) {
    return { ok: false, message: describeMermaidError(thrown) };
  }
}

/**
 * 把 mermaid 抛出来的东西翻译成一句给用户看的话。
 *
 * mermaid 的语法错误长这样：
 *
 * ```
 * Parse error on line 2:
 * graph TB;  a-->|b|
 * ------------------^
 * Expecting 'SEMI', 'NEWLINE', ...
 * ```
 *
 * 那段 ASCII 示意图是**唯一能指出出错位置**的东西，改不得；整段英文既改不动、
 * 硬译又容易译错（`'SEMI'` 这类记号名译出来没人认得）。所以只在前面加一句
 * 中文引言，原文照实给。
 *
 * 认不出来的一律如实带上原文，不编一句「图表有误」—— 那句话没有信息量。
 */
export function describeMermaidError(thrown: unknown): string {
  const raw = thrown instanceof Error ? thrown.message : String(thrown);
  const trimmed = raw.trim();
  if (trimmed === '') return t('core.render.mermaidNoReason');
  if (/^parse error\b/i.test(trimmed)) {
    return t('core.render.mermaidSyntax') + trimmed;
  }
  return trimmed;
}

/** 兜底：`String(thrown)` 对非 Error 也能给出一句话，而不是 `[object Object]`。 */
function describe(thrown: unknown): string {
  return thrown instanceof Error ? thrown.message : String(thrown);
}

// ---------------------------------------------------------------------------
// 真实环境下的容器
// ---------------------------------------------------------------------------

/**
 * 「在布局里、但看不见」—— 不能加 `display: none`，理由见文件头。
 *
 * `position: fixed` 是刻意选的：它脱离文档流，所以不会影响页面的排布与滚动范围，
 * 而 `absolute` 会挂到某个祖先的定位上下文上，那个上下文我们控制不了。
 */
export const MEASURE_CONTAINER_STYLE = 'position:fixed;top:0;left:-99999px;pointer-events:none;';

/**
 * 造一个临时容器给 mermaid 量尺寸，用完摘掉。
 *
 * 导出也走这一条路：导出的目标虽然是一个字符串，但 mermaid 只在**真的页面**上
 * 画得出来，所以容器仍然挂在 `document.body` 上，只是挪到了屏幕外。
 */
export async function withMeasureContainer<T>(
  run: (container: HTMLElement) => Promise<T>,
): Promise<T> {
  const container = document.createElement('div');
  container.setAttribute('style', MEASURE_CONTAINER_STYLE);
  container.setAttribute('aria-hidden', 'true');
  document.body.appendChild(container);
  try {
    return await run(container);
  } finally {
    // mermaid 会摘掉它自己塞进容器的临时节点，但容器本身是我们造的，
    // 而且必须造在页面上 —— 所以也只能由我们摘掉。放在 `finally` 里：
    // 抛错的那条路同样不能把容器留在 body 上
    container.remove();
  }
}

const DEFAULT_IO: MermaidIo = {
  load: loadMermaid,
  withContainer: withMeasureContainer,
};
