/**
 * 导出一份**自包含**的 KaTeX 样式：CSS + 内联成 data URI 的 woff2 字体。
 *
 * 「拆分 `@font-face`、改写 `src`」那些纯文本动作住在 `core/katex-fonts.ts`
 * （构建期的编辑器 CSS 也要用同一套，而 Vite 的配置文件不能 import 有
 * `import.meta.glob` 的模块）。这个文件只负责**导出这一侧**的两件事：
 * 用 data URI 内联字体、把结果拼成一份可直接放进 `<style>` 的 CSS。
 *
 * ## 为什么必须内联字体
 *
 * `katex.min.css` 里有 20 条 `@font-face`，每条指向三个文件（woff2 / woff / ttf）。
 * 公式的**字形与间距**全部依赖这些字体 —— 不带上它们，导出的文件在别人机器上
 * 会用系统字体顶替：根号、求和号、花体字母要么缺失、要么间距全乱。
 * 「拷到任何一台机器、断网、双击打开，观感与编辑器一致」是导出的验收标准
 * （见 `core/export/document.ts` 的文件头），所以字体必须跟着走。
 *
 * ## 只留 woff2
 *
 * woff / ttf 是给不认 woff2 的老浏览器准备的。导出文件是给**现在**的人看的，
 * 留三份会让体积变成三倍（219.9 KB → 660 KB）。丢掉之后每个 `@font-face`
 * 只剩一个 `src`。
 *
 * ## 为什么不按「用到的字体族」裁剪
 *
 * 试过，并且放弃了。裁掉没用的族能省一半体积（典型公式只需要
 * KaTeX_Main + KaTeX_Math，约 118 KB 而非 220 KB），但**判断哪些族被用到
 * 需要一份脆弱的启发式**：KaTeX 的基础字体来自 `.katex{font:…KaTeX_Main…}`
 * 这条 `font` **简写**（不是 `font-family`），还会通过继承生效 ——
 * 按选择器匹配会漏掉它，于是「任何公式都需要的正文字体」被判成不需要，
 * 导出结果悄悄退化。少 100 KB 换一个「可能悄悄渲染错」的风险，不划算。
 * 重审条件写在 ADR 0019 里。
 *
 * ## 字体字节怎么拿到
 *
 * 走 `?url` 的 glob + `fetch`。三条路都试过：
 *
 * - `?inline`：**dev 模式不生效**。Vite 在 serve 阶段对资源一律返回 URL，
 *   只有 build 才真的内联。于是 `npm run dev` 里导出会得到一堆指向不存在路径的
 *   字体引用 —— 一个只在开发期出现、上线就消失的 bug；
 * - 直接读 `node_modules`：打包之后那个目录在用户机器上不存在；
 * - `?url` + fetch：dev 与 build 行为一致，且与 `katex.min.css` 引用的
 *   **是同一份资源文件**（不会让字体在产物里存两份）。
 *
 * 这个 fetch 取的是**应用自己的资源**（Tauri 下是同源的自定义协议，
 * 开发时是本地 dev server），不是网络请求 —— CSP 里的
 * `connect-src 'self'` 已经允许。取不到时**如实上报**，由导出提示告诉用户
 * 「字体没能内联」，而不是安静地导出一份会掉字形的文件。
 *
 * ## 字体必须作为独立文件产出
 *
 * 走 `?url` 拿到的 URL 之后要 `fetch` 它。而 Vite 默认把小于 4 KB 的资源
 * 内联成 `data:` URI —— `KaTeX_Size3-Regular.woff2` 正好是 3624 字节，
 * 会踩中那条线，于是 `fetch('data:…')` 被 CSP 的 `connect-src 'self'` 拦下来，
 * 那个字体**静默缺失**。`vite.config.ts` 的 `assetsInlineLimit` 为此
 * 明确要求 woff2 永不内联。
 */

import { bytesToBase64 } from '../image.js';
import { fontFaceWithSrc, KATEX_FONT_DIR, splitFontFaces, type FontFace } from '../katex-fonts.js';

/**
 * 把一条 `@font-face` 的 `src` 换成内联的 data URI，并丢掉 woff / ttf 那两条。
 *
 * 只替换 `src:` 这一段而不是重建整条规则 —— 理由见 `fontFaceWithSrc`。
 */
export function fontFaceWithDataUri(face: FontFace, dataUri: string): string {
  return fontFaceWithSrc(face, `url(${dataUri}) format("woff2")`);
}

/** 取字节的能力。做成注入的，测试才能不依赖打包产物。 */
export interface MathStyleIo {
  /** 读 KaTeX 的原始 CSS。 */
  readonly readCss: () => Promise<string>;
  /** 取一个字体文件的 data URI；拿不到返回 `null`（由调用方如实上报）。 */
  readonly loadFont: (file: string) => Promise<string | null>;
}

export interface MathStyles {
  /** 可以直接放进 `<style>` 的完整 CSS。 */
  readonly css: string;
  /** 成功内联的字体文件数。 */
  readonly inlinedFonts: number;
  /** 没能内联的字体文件数。大于 0 时导出提示必须说出来。 */
  readonly missingFonts: number;
}

export async function buildMathStyles(io: MathStyleIo = DEFAULT_IO): Promise<MathStyles> {
  const css = await io.readCss();
  const { faces, rest } = splitFontFaces(css);

  const inlined: string[] = [];
  let missingFonts = 0;

  for (const face of faces) {
    if (face.woff2 === null) {
      missingFonts += 1;
      continue;
    }
    const dataUri = await io.loadFont(face.woff2);
    if (dataUri === null) {
      // 不保留原来那条指向 `fonts/…` 的规则：那个路径在导出的单文件里不存在，
      // 留着只会让打开文件时多出 20 个 404。丢掉它、上报它，行为是可解释的
      missingFonts += 1;
      continue;
    }
    inlined.push(fontFaceWithDataUri(face, dataUri));
  }

  return { css: `${inlined.join('\n')}\n${rest}`, inlinedFonts: inlined.length, missingFonts };
}

// ---------------------------------------------------------------------------
// 真实环境下的取字节实现
// ---------------------------------------------------------------------------

/**
 * 字体文件的懒加载表。
 *
 * `?url` 让 Vite 把每个 woff2 变成资源文件并返回它的 URL。**懒加载**
 * （没有 `eager`）是必需的：20 个 URL 虽然小，但没有理由让每一份文档
 * 都在启动时去建立这张表。真正的字节只在导出**带公式**的文档时才取。
 *
 * 路径写成以 `/` 开头（相对项目根）而不是相对本文件：glob 的模式必须是
 * 字面路径，不能写包名，而 `node_modules` 在项目根下。
 */
const fontModules = import.meta.glob('/node_modules/katex/dist/fonts/*.woff2', {
  query: '?url',
  import: 'default',
});

async function fetchFontDataUri(file: string): Promise<string | null> {
  // `noUncheckedIndexedAccess` 让这次取值天然带 `| undefined` ——
  // KaTeX 将来多出一个字体而我们没更新时，这里必须能如实说「没有这个文件」
  const load = fontModules[KATEX_FONT_DIR + file];
  if (load === undefined) return null;
  try {
    const url = await load();
    const response = await fetch(url);
    if (!response.ok) return null;
    return `data:font/woff2;base64,${bytesToBase64(await response.arrayBuffer())}`;
  } catch {
    return null;
  }
}

const DEFAULT_IO: MathStyleIo = {
  // 动态引入：这份 CSS 有 25 KB，只有「导出带公式的文档」时才用得上，
  // 没理由让每个打开轻笺的人都先下载它
  readCss: async () => (await import('katex/dist/katex.min.css?raw')).default,
  loadFont: fetchFontDataUri,
};
