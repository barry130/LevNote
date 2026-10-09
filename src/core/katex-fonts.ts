/**
 * KaTeX 的 `@font-face` 声明：拆分、改写 `src`、收敛成只留 woff2。
 *
 * ## 为什么这个文件在 `core/`
 *
 * 它被两个地方用，而这两个地方的运行环境完全不同：
 *
 * 1. **构建期**（`vite.config.ts` 里的 PostCSS 插件）：编辑器那份 KaTeX CSS
 *    走 Vite 的 CSS 管线，这里把每个 `@font-face` 的 `src` 收敛成只留 woff2 ——
 *    否则 Vite 会把 woff 与 ttf 也当成资源产出，那是**约 800 KB 永远不会被取用**
 *    的死重（见 `woff2OnlyFontFaces`）；
 * 2. **导出期**（`core/export/katex-css.ts`）：把 `src` 换成内联的 data URI，
 *    让导出的单文件在断网机器上也有字形。
 *
 * 它必须是**零运行时依赖**：Vite 的配置文件在应用模块图之外被加载，这里不能有
 * `import.meta.glob`、不能有 DOM、不能有副作用 —— 否则 `vite.config.ts`
 * 一 import 就崩。唯一的外部引用是 `import type { Root }`，类型，编译后消失。
 *
 * ## 为什么可以放心用正则切
 *
 * `katex.min.css` 的结构是**扁平**的：整份文件里 `{` 的最大嵌套深度是 1，
 * 唯一的 at-rule 就是 `@font-face`（没有 `@media`、没有 `@supports`），
 * 每条 `@font-face` 体内也没有嵌套的 `{}`。这三点都有测试钉着
 * （`katex-fonts.test.ts` 直接对**真实的** katex.min.css 跑一遍）。
 *
 * 换成通用 CSS 解析器当然更「正确」，但代价是引一个依赖来处理一份
 * 我们自己看得见的、格式固定的文件 —— 而那份依赖一旦哪天解析出别的形状，
 * 出错的方式是**公式悄悄掉字形**，比正则更难发现。
 */

import type { Root } from 'postcss';

/** 一条 `@font-face` 规则里我们关心的部分。 */
export interface FontFace {
  /** 规则原文（含 `@font-face{` 与结尾的 `}`）。 */
  readonly text: string;
  /** 字体族名，如 `KaTeX_Main`。 */
  readonly family: string;
  /** woff2 的文件名，如 `KaTeX_Main-Regular.woff2`；规则里没有 woff2 时为 null。 */
  readonly woff2: string | null;
}

export interface SplitFontFaces {
  /** 按出现顺序排列的字体声明。 */
  readonly faces: readonly FontFace[];
  /** 去掉所有 `@font-face` 之后剩下的 CSS（版式规则）。 */
  readonly rest: string;
}

/** 一条 `src` 取值里 woff2 那一段。 */
export interface Woff2Ref {
  /** 那一段里的 url 路径原文，如 `fonts/KaTeX_Main-Regular.woff2`。 */
  readonly url: string;
  /** 整段原文，含 `format("woff2")`，如 `url(fonts/KaTeX_Main-Regular.woff2) format("woff2")`。 */
  readonly segment: string;
}

/**
 * 从一条 `src` 取值里挑出 woff2 那一段。
 *
 * `src` 是个逗号分隔的候选列表（woff2 / woff / ttf 各一段），这里只取 woff2
 * 那一段，且**连 `format("woff2")` 一起原样返回** —— 构建期要把这段放回 CSS，
 * 里面的相对路径得原封不动地交给 Vite 去解析（理由见 `woff2OnlyFontFaces`）。
 *
 * 无 `g` 标记，所以这个正则无状态、可以安全地放在模块级复用。
 */
export function woff2Ref(srcValue: string): Woff2Ref | null {
  const match = /url\(\s*([^)]*?\.woff2)\s*\)(?:\s*format\([^)]*\))?/.exec(srcValue);
  if (match === null) return null;
  return { url: match[1] ?? '', segment: match[0].trim() };
}

/**
 * KaTeX 的字体文件所在目录（相对项目根）。
 *
 * **只给导出侧当 `import.meta.glob` 的键前缀用**（`core/export/katex-css.ts`），
 * 绝不能拿去写进 CSS 的 `url()` —— 那在构建产物里是个 public 路径，
 * Vite 不解析、不产出，字体 404，而构建照样成功。
 */
export const KATEX_FONT_DIR = '/node_modules/katex/dist/fonts/';

export function splitFontFaces(css: string): SplitFontFaces {
  const faces: FontFace[] = [];
  // 每次调用都新建正则：带 `g` 的正则是有状态的（`lastIndex`），
  // 复用模块级的那个会让第二次调用从上次的位置接着找
  const pattern = /@font-face\{([^}]*)\}/g;

  let match: RegExpExecArray | null;
  while ((match = pattern.exec(css)) !== null) {
    const body = match[1] ?? '';
    const ref = woff2Ref(body);
    faces.push({
      text: match[0],
      family: fontFamilyOf(body),
      woff2: ref === null ? null : basename(ref.url),
    });
  }

  return { faces, rest: css.replace(pattern, '') };
}

/** 路径里最后一段。`fonts/KaTeX_Main-Regular.woff2` → `KaTeX_Main-Regular.woff2`。 */
function basename(path: string): string {
  const parts = path.split(/[\\/]/);
  return parts[parts.length - 1] ?? path;
}

/**
 * 取 `font-family` 的取值。
 *
 * 3 条 `KaTeX_SansSerif` 的族名**带引号**（`font-family:"KaTeX_SansSerif"`），
 * 其余不带 —— 正则少写一个 `"?` 就会让那三条取到空字符串。族名只用于报错与
 * 调试，取不到不会报错，于是**静默**。有测试钉着。
 */
function fontFamilyOf(text: string): string {
  return /font-family:\s*"?([^;"]+)"?/.exec(text)?.[1]?.trim() ?? '';
}

/**
 * 换掉一条 `@font-face` 的 `src`。
 *
 * **只替换 `src:` 这一段**，而不是重建整条规则：`font-display` / `font-weight` /
 * `font-style` 是 KaTeX 定的，重建就等于把它们的取值抄一份到我们这边，
 * 抄漏一个（比如 `font-style:italic`）就会让所有斜体字母变成正体。
 */
export function fontFaceWithSrc(face: FontFace, src: string): string {
  const body = face.text.slice('@font-face{'.length, -1);
  return `@font-face{${body.replace(/src:[^;}]*/, `src:${src}`)}}`;
}

/**
 * 把一棵 CSS AST 里的每条 `@font-face` 收敛成「只声明 woff2」，**就地改**。
 *
 * ## 为什么要做这件事
 *
 * `katex.min.css` 的每条 `@font-face` 都指向三个文件（woff2 / woff / ttf）。
 * 编辑器那份 CSS 一旦 `@import` 它，Vite 就会把**三个都**当成资源产出：
 * 实测 60 个文件、1.05 MB，而 woff 与 ttf 那 798 KB **永远不会被取用** ——
 * 目标环境只有 WebView2（Chromium）与 WKWebView（WebKit），两者从 2016 年起
 * 就支持 woff2。它们会一直躺在安装包里。
 *
 * woff / ttf 是给不认 woff2 的老浏览器准备的，而轻笺没有那种目标环境。
 *
 * ## 为什么必须**就地改**，而不是把 CSS 重新 parse 一遍
 *
 * 这是踩过的坑，值得写下来。Vite 解析 `url()` 时用的 importer 是
 * **每个声明自己的** `declaration.source.input.file`
 * （`vite-url-rewrite` 插件里那句 `const importer = declaration.source?.input.file`）。
 * 所以 `@import` 进来的 katex.min.css，它那些 `url(fonts/X.woff2)` 的相对基准
 * 是 **katex 目录**，Vite 才能解析出带哈希的 `/assets/…woff2`。
 *
 * 一开始的实现是「`root.toString()` → 文本变换 → `postcss.parse` → 换掉整棵
 * 节点树」。结果新节点的来源全变成 `markdown.css`，相对路径解析不到 ——
 * 构建日志里是 `fonts/KaTeX_Main-BoldItalic.woff2 referenced in … didn't resolve
 * at build time`，**构建照样成功**，产物里字体 404、公式全部掉字形。
 *
 * 所以这里只改 `decl.value`，一个节点都不重建。
 *
 * ## 为什么让它**响亮地失败**
 *
 * 这条变换靠正则匹配 KaTeX 的 CSS 格式。哪天 KaTeX 换了写法（比如给
 * `@font-face` 套一层 `@media`），就会一条都匹配不到 —— 而「匹配不到」的表现是
 * **所有公式字体静默退化**，构建照样成功。所以这里宁可让构建失败。
 *
 * 这与导出侧的取舍不同：那边有 `report` 可以如实上报「有几个字体没内联」，
 * 编辑器这边没有汇报渠道，唯一的出口就是让构建停下来。
 */
export function woff2OnlyFontFaces(root: Root): void {
  let faceCount = 0;
  const dropped: string[] = [];

  root.walkAtRules('font-face', (at) => {
    faceCount += 1;
    let kept = false;

    at.walkDecls('src', (decl) => {
      const ref = woff2Ref(decl.value);
      if (ref === null) return;
      decl.value = ref.segment;
      kept = true;
    });

    if (!kept) {
      const family = fontFamilyOf(at.toString());
      dropped.push(family === '' ? '(族名取不到)' : family);
    }
  });

  if (faceCount === 0) {
    throw new Error(
      '从 KaTeX 的 CSS 里一条 @font-face 都没解析出来。正则可能已经与上游格式不匹配 —— ' +
        '继续构建会让所有公式静默退化成系统字体。请检查 katex.min.css 的结构（见 katex-fonts.ts 的说明）。',
    );
  }

  if (dropped.length > 0) {
    throw new Error(
      `KaTeX 的 CSS 里有 ${dropped.length} 条 @font-face 没有 woff2（${dropped.join('、')}）。` +
        '编辑器没有「上报缺字体」的渠道，所以这里让构建失败而不是静默丢字形。',
    );
  }
}
