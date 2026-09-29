/**
 * 导出自包含的 KaTeX 样式：**把字体换成 data URI**，拼成一份可用的 CSS。
 *
 * 「拆分 `@font-face`、改写 `src`」那些纯文本动作住在 `core/katex-fonts.ts`，
 * 由 `katex-fonts.test.ts` 对**真实的** katex.min.css 钉住。这里只验导出侧
 * 多出来的那一层：内联、以及「取不到时怎么如实上报」。
 *
 * ## 为什么字体字节要注入
 *
 * 真实实现走 `?url` glob + `fetch`，而 `fetch` 需要**同源**环境：vitest 的
 * node 环境里那个 URL 是 `/node_modules/…`，没有 base，`fetch` 会直接抛
 * `TypeError: Failed to parse URL`。所以真实实现只能靠**构建产物**验证
 * （见 `ROADMAP.md` 的验收项），这里把取字节这件事注入进来，
 * 验的是「拿到字节之后怎么拼」。
 */

import { describe, expect, it } from 'vitest';
import katexCss from 'katex/dist/katex.min.css?raw';

import { splitFontFaces } from '../katex-fonts.js';
import { buildMathStyles, fontFaceWithDataUri, type MathStyleIo } from './katex-css.js';

describe('fontFaceWithDataUri', () => {
  it('把 `src` 换成 data URI，并声明格式是 woff2', () => {
    const face = splitFontFaces(katexCss).faces.find((candidate) =>
      candidate.text.includes('font-style:italic'),
    );
    if (face === undefined) throw new Error('katex.min.css 里找不到斜体声明');

    const output = fontFaceWithDataUri(face, 'data:font/woff2;base64,AAAA');

    expect(output).toContain('url(data:font/woff2;base64,AAAA)');
    expect(output).toContain('format("woff2")');
    expect(output.match(/src:/g)).toHaveLength(1);
    expect(output).not.toContain('truetype');
    // 斜体字形靠它，不能被顺手丢掉
    expect(output).toContain('font-style:italic');
  });
});

describe('buildMathStyles', () => {
  /** 一个永远成功的取字节实现：data URI 里放文件名当指纹。 */
  const okIo: MathStyleIo = {
    readCss: async () => katexCss,
    loadFont: async (file) => `data:font/woff2;base64,${file}`,
  };

  it('20 条全部内联，一条都不缺', async () => {
    const styles = await buildMathStyles(okIo);

    expect(styles.inlinedFonts).toBe(20);
    expect(styles.missingFonts).toBe(0);
    expect(styles.css.match(/data:font\/woff2;base64,/g)).toHaveLength(20);
    // 版式规则也必须在（只有字体没有版式的 CSS 等于没有公式排版）
    expect(styles.css).toContain('.katex');
  });

  it('字体声明排在版式规则**前面** —— 先声明再用', async () => {
    const styles = await buildMathStyles(okIo);

    expect(styles.css.indexOf('@font-face')).toBeLessThan(styles.css.indexOf('.katex'));
  });

  /**
   * 取不到的字体**如实上报**，并且**不留下**指向 `fonts/…` 的规则。
   *
   * 那个路径在导出的单文件里不存在，留着只会让打开文件时多出 20 个 404。
   * 丢掉它、把数字报给导出提示，行为才是可解释的。
   */
  it('取不到的字体计入 missingFonts，且不留下坏引用', async () => {
    const io: MathStyleIo = {
      readCss: async () => katexCss,
      loadFont: async (file) => (file.includes('Size3') ? null : `data:font/woff2;base64,${file}`),
    };

    const styles = await buildMathStyles(io);

    expect(styles.inlinedFonts).toBe(19);
    expect(styles.missingFonts).toBe(1);

    const faces = [...styles.css.matchAll(/@font-face\{[^}]*\}/g)].map((match) => match[0]);
    expect(faces).toHaveLength(19);
    expect(faces.some((face) => face.includes('Size3'))).toBe(false);
    expect(styles.css).not.toContain('url(fonts/');
    // 其余 19 条不受影响
    expect(styles.css).toContain('KaTeX_Main-Regular');

    /**
     * **注意 `KaTeX_Size3` 这个名字仍然在** —— 版式规则里有一条
     * `.katex .delimsizing.size3{font-family:KaTeX_Size3}`，它指向一个
     * 已经不存在 `@font-face` 的族。
     *
     * 这就是「缺一个字体」的真实后果：不报错、不留 404，只是浏览器退回默认字体，
     * 于是 `\left(` 之类的大号定界符**悄悄换了字形**。所以 `missingFonts`
     * 必须被报给用户（见 `export-document.ts` 的 `describeExport`），
     * 而不是当成一个内部计数器。
     */
    expect(styles.css, '版式规则指向一个已不存在的族').toContain('font-family:KaTeX_Size3');
  });

  it('全部取不到时仍然产出可用的版式 CSS，只是没有字体', async () => {
    const styles = await buildMathStyles({
      readCss: async () => katexCss,
      loadFont: async () => null,
    });

    expect(styles.inlinedFonts).toBe(0);
    expect(styles.missingFonts).toBe(20);
    expect(styles.css).not.toContain('@font-face');
    expect(styles.css).toContain('.katex');
  });

  /**
   * 声明里没有 woff2 时也算「缺一个」—— 不能因为「没有 woff2 可丢」就
   * 假装它没问题。真实文件里 20 条都有 woff2，所以这里用一段合成的 CSS。
   */
  it('声明里没有 woff2 时计入 missingFonts，且不去取字节', async () => {
    const synthetic = '@font-face{font-family:X;src:url(fonts/X.ttf) format("truetype")}';
    const asked: string[] = [];

    const styles = await buildMathStyles({
      readCss: async () => synthetic,
      loadFont: async (file) => {
        asked.push(file);
        return 'data:x';
      },
    });

    expect(styles.inlinedFonts).toBe(0);
    expect(styles.missingFonts).toBe(1);
    expect(asked, '没有 woff2 就没有文件可取').toEqual([]);
    expect(styles.css).not.toContain('url(fonts/');
  });
});
