/**
 * KaTeX 的 `@font-face` 声明：拆分、改写 `src`、收敛成只留 woff2。
 *
 * ## 为什么直接对**真实的** `katex.min.css` 跑
 *
 * 这里的正则成立的前提是三条**关于那份文件的事实**：结构扁平
 * （`{` 最大嵌套深度是 1）、唯一的 at-rule 就是 `@font-face`、
 * 每条 `@font-face` 体内没有嵌套的 `{}`。
 *
 * 这三条不是我们对 CSS 的假设，是 KaTeX 那份文件的现状 —— 所以证据只能是
 * 「拿那份文件跑一遍」。哪天 KaTeX 加了 `@media`，这里会先红，
 * 而不是等到用户发现公式掉字形。
 *
 * ## `woff2OnlyFontFaces` 为什么要「响亮地失败」
 *
 * 它被 `vite.config.ts` 里的 PostCSS 插件在构建期调用，用来把编辑器那份
 * KaTeX CSS 里的 woff / ttf 引用去掉（那约 800 KB 永远不会被取用）。
 * 它一旦静默失效，后果是**所有公式字体退化成系统字体**，而构建照样成功 ——
 * 所以下面有两条用例专门钉「格式变了就让构建停下来」。
 */

import { describe, expect, it } from 'vitest';
import postcss from 'postcss';
import katexCss from 'katex/dist/katex.min.css?raw';

import {
  fontFaceWithSrc,
  KATEX_FONT_DIR,
  splitFontFaces,
  woff2OnlyFontFaces,
  woff2Ref,
  type FontFace,
} from './katex-fonts.js';

/** CSS 里 `{` 的最大嵌套深度。`@font-face` 扁平化是正则切法的前提。 */
function maxBraceDepth(css: string): number {
  let depth = 0;
  let max = 0;
  for (const char of css) {
    if (char === '{') {
      depth += 1;
      if (depth > max) max = depth;
    } else if (char === '}') {
      depth -= 1;
    }
  }
  return max;
}

function facesOf(css = katexCss): readonly FontFace[] {
  return splitFontFaces(css).faces;
}

describe('真实的 katex.min.css 具备正则切法的前提', () => {
  it('`{` 的最大嵌套深度是 1 —— 没有嵌套规则', () => {
    expect(maxBraceDepth(katexCss)).toBe(1);
  });

  it('唯一的 at-rule 就是 `@font-face`', () => {
    const atRules = [...katexCss.matchAll(/@[a-zA-Z-]+/g)].map((match) => match[0]);
    expect(new Set(atRules)).toEqual(new Set(['@font-face']));
  });

  it('有 20 条字体声明，每条都能取到族名与 woff2', () => {
    const faces = facesOf();

    expect(faces).toHaveLength(20);
    for (const face of faces) {
      expect(face.family, `族名取不到：${face.text.slice(0, 60)}`).not.toBe('');
      expect(face.woff2, `woff2 取不到：${face.text.slice(0, 60)}`).not.toBeNull();
    }
  });

  /**
   * 当初踩的坑：3 条 `KaTeX_SansSerif` 的 `font-family` **带引号**
   * （`font-family:"KaTeX_SansSerif"`），而其余不带。
   * 正则只写 `font-family:\s*(KaTeX_…)` 的话，那三条的族名会是空字符串 ——
   * 而族名只被用来记录与调试，取不到也不会报错，于是**静默**。
   */
  it('带引号的族名也能取到（KaTeX_SansSerif 那三条）', () => {
    const sansSerif = facesOf().filter((face) => face.family === 'KaTeX_SansSerif');

    expect(sansSerif).toHaveLength(3);
    expect(facesOf().some((face) => face.family === '')).toBe(false);
  });

  it('woff2 只留文件名，不带目录', () => {
    for (const face of facesOf()) {
      expect(face.woff2).not.toContain('/');
      expect(face.woff2).toMatch(/\.woff2$/);
    }
  });

  it('去掉字体声明之后剩下的版式规则里没有任何 `url(`', () => {
    const { rest } = splitFontFaces(katexCss);

    expect(rest).not.toContain('@font-face');
    // 这是「丢掉 woff/ttf 之后不会留下坏引用」的依据：剩下的规则本来就不引用资源
    expect(rest).not.toContain('url(');
    // 版式规则必须还在（不然公式的间距全乱）
    expect(rest).toContain('.katex');
  });

  it('连调两次结果一致 —— 带 `g` 的正则不能复用', () => {
    // `RegExp.prototype.exec` 在带 `g` 时是有状态的（`lastIndex`）。
    // 复用模块级的那个正则，第二次调用会从上次停下的位置接着找 ——
    // 表现是「第二次调用少一半结果」，而且只在同一进程里连续调用时出现
    const first = splitFontFaces(katexCss);
    const second = splitFontFaces(katexCss);

    expect(second.faces).toHaveLength(first.faces.length);
    expect(second.rest).toBe(first.rest);
  });

  it('空 CSS 不抛错', () => {
    expect(splitFontFaces('').faces).toEqual([]);
    expect(splitFontFaces('').rest).toBe('');
  });
});

describe('fontFaceWithSrc', () => {
  /** 找一条带 `font-style: italic` 的真实声明（斜体字形全靠它）。 */
  function italicFace(): FontFace {
    const face = facesOf().find((candidate) => candidate.text.includes('font-style:italic'));
    if (face === undefined) throw new Error('katex.min.css 里找不到斜体声明');
    return face;
  }

  it('`font-style` / `font-weight` / `font-display` 原样保留', () => {
    // 只替换 `src:` 而不是重建整条规则，就是为了不把 KaTeX 定的这些取值
    // 抄一份到我们这边 —— 抄漏一个（比如 `font-style:italic`）
    // 就会让所有斜体字母变成正体
    const output = fontFaceWithSrc(italicFace(), 'url(x.woff2) format("woff2")');

    expect(output).toContain('font-style:italic');
    expect(output).toContain('font-weight:');
    expect(output).toContain('font-display:');
  });

  it('只剩一个 `src`，woff / ttf 两条被丢掉', () => {
    const output = fontFaceWithSrc(italicFace(), 'url(x.woff2) format("woff2")');

    expect(output.match(/src:/g)).toHaveLength(1);
    expect(output).not.toContain('format("woff")');
    expect(output).not.toContain('truetype');
  });

  it('产出的仍然是一条合法的 `@font-face`', () => {
    const output = fontFaceWithSrc(italicFace(), 'url(x.woff2) format("woff2")');

    expect(output.startsWith('@font-face{')).toBe(true);
    expect(output.endsWith('}')).toBe(true);
    // 不能多出或少掉大括号
    expect(output.match(/\{/g)).toHaveLength(1);
    expect(output.match(/\}/g)).toHaveLength(1);
    expect(maxBraceDepth(output)).toBe(1);
  });
});

describe('woff2Ref', () => {
  it('从 `src` 的候选列表里挑出 woff2 那一段，连 `format` 一起原样返回', () => {
    const value =
      'url(fonts/X.woff2) format("woff2"),url(fonts/X.woff) format("woff"),url(fonts/X.ttf) format("truetype")';

    expect(woff2Ref(value)).toEqual({
      url: 'fonts/X.woff2',
      // 整段原样返回是**构建期能跑通的前提**：Vite 要按这段里的相对路径去解析
      segment: 'url(fonts/X.woff2) format("woff2")',
    });
  });

  it('上游没写 `format` 时也能取到', () => {
    expect(woff2Ref('url(a.woff2)')).toEqual({ url: 'a.woff2', segment: 'url(a.woff2)' });
  });

  it('只有 woff / ttf 时返回 null', () => {
    expect(woff2Ref('url(a.woff) format("woff")')).toBeNull();
    expect(woff2Ref('url(a.ttf) format("truetype")')).toBeNull();
  });

  it('真实文件里每一条声明都取得到', () => {
    for (const face of facesOf()) {
      expect(woff2Ref(face.text), `取不到：${face.family}`).not.toBeNull();
    }
  });
});

describe('woff2OnlyFontFaces', () => {
  /** 对真实（或给定）的 CSS 跑一遍就地变换，返回变换后的文本。 */
  function transformed(css = katexCss): string {
    const root = postcss.parse(css, { from: 'katex.min.css' });
    woff2OnlyFontFaces(root);
    return root.toString();
  }

  it('20 条声明全留下，每条只声明 woff2', () => {
    const faces = facesOf(transformed());

    expect(faces).toHaveLength(20);
    for (const face of faces) {
      expect(face.woff2, `woff2 没了：${face.family}`).not.toBeNull();
    }
  });

  /**
   * 这条当初写反过：初版把 `src` 换成了 `/node_modules/katex/dist/fonts/…`，
   * 理由是「Vite 才解析得到」。dev server 下确实能跑（`/node_modules/` 是
   * 它能服务的路径），但**构建产物里那是个 public 路径** —— Vite 不解析、
   * 不产出、字体 404，而构建照样成功。
   *
   * 所以这里钉的是「原样保留相对路径」，并且**禁止**那个前缀出现在 CSS 里。
   */
  it('`src` 原样保留相对路径，不换成绝对路径', () => {
    const css = transformed();

    expect(css).toContain('url(fonts/KaTeX_Main-Regular.woff2) format("woff2")');
    // `KATEX_FONT_DIR` 只是导出侧 `import.meta.glob` 的键前缀，不能进 CSS
    expect(css).not.toContain(KATEX_FONT_DIR);
  });

  it('woff / ttf 的引用一个都不剩', () => {
    const css = transformed();

    expect(css).not.toContain('.woff)');
    expect(css).not.toContain('.ttf');
    expect(css).not.toContain('truetype');
  });

  it('版式规则一个字节都没动', () => {
    const strip = (text: string): string => text.replace(/@font-face\{[^}]*\}/g, '');
    const css = transformed();

    expect(css).toContain('.katex');
    expect(css).toContain('katex-display');
    expect(strip(css)).toBe(strip(katexCss));
  });

  /**
   * **「字体 404」那个 bug 的回归。**
   *
   * Vite 解析 `url()` 时用的 importer 是每个声明自己的
   * `declaration.source.input.file`。曾经的实现是「文本变换 + `postcss.parse`
   * 换掉整棵树」，新节点的来源全变成 `markdown.css`，于是
   * `url(fonts/X.woff2)` 的基准从 katex 目录变成了 markdown.css 所在目录 ——
   * 构建成功、产物里字体 404、公式全部掉字形。
   *
   * 所以这条钉的是「节点与它的来源信息都不换」。
   */
  it('就地改取值：节点对象与来源信息都不换', () => {
    const root = postcss.parse(katexCss, { from: 'katex.min.css' });
    const nodesBefore = [...root.nodes];
    const sourcesBefore = root.nodes.map((node) => node.source?.input.file);

    woff2OnlyFontFaces(root);

    expect([...root.nodes]).toEqual(nodesBefore);
    expect(root.nodes.map((node) => node.source?.input.file)).toEqual(sourcesBefore);
    // 来源必须还指着 katex 那份文件 —— Vite 就是靠它把 `url(fonts/…)` 解析到
    // katex 的 dist 目录；换成 `markdown.css` 就解析不到了
    for (const file of sourcesBefore) {
      expect(file).toMatch(/katex\.min\.css$/);
    }
  });

  it('幂等：对自己的产物再跑一遍结果不变', () => {
    const once = transformed();
    expect(transformed(once)).toBe(once);
  });

  /**
   * 这条与下一条是**构建期的保险**。
   *
   * 正则一旦与上游格式不匹配（KaTeX 换了写法），变换会一条都匹配不到 ——
   * 而「匹配不到」的表现是所有公式字体**静默**退化成系统字体，
   * 构建照样成功。编辑器没有「上报缺字体」的渠道（导出侧有 `report`），
   * 所以唯一的出口就是让构建停下来。
   */
  it('一条 `@font-face` 都没有时抛错，而不是静默产出一份没有字体的 CSS', () => {
    expect(() => transformed('.katex{font:normal 1em serif}')).toThrow(/@font-face/);
  });

  it('有声明缺 woff2 时也抛错 —— 那意味着某个字族会静默掉字形', () => {
    const css = '@font-face{font-family:X;src:url(fonts/X.ttf) format("truetype")}.katex{color:red}';

    expect(() => transformed(css)).toThrow(/X/);
  });
});
