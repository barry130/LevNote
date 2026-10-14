/**
 * 样式弹层的词表与折算（ADR 0059 —— 对标 XMind 的主题样式面板第一屏）。
 *
 * ## 词表住在这里，而格式层一个字都不认识它
 *
 * `.mindmap` 文件里存的是内核的 `style` 对象本身（`MindmapNode.style`，
 * §7.1 骨架键，**不透明**：`Record<string, unknown>`，格式层照 §5 原样保留）。
 * 这张表只回答**界面**的三个问题：弹层里有哪些键、每个键给哪些取值、
 * 哪个取值算「当前挂着」。文件里出现词表之外的键（别的工具写的
 * `borderColor` 一类）照样原样往返 —— 词表是界面知识，不是格式语义，
 * 与 `markers.ts` 同一条纪律。
 *
 * ## 为什么恰好是这四个键
 *
 * `color` / `background` / `fontSize` / `fontWeight` —— 这是内核**导出侧**
 * 认得的全部（`LayoutSsr.js:45-47` 只把这四样拼进独立 SVG 的内联样式，
 * 其余键导出时静默丢）。词表若超出这个范围，编辑器里画得出、导出的
 * 图里没有，正是 ADR 0050「导出画出来的样子」要拦的落差。
 *
 * ## `fontSize` 为什么是数字
 *
 * 内核导出时按 `font-size: ${v}px` 拼接 —— 值必须是数字才拼得对；
 * 而内核渲染侧把 `style` 每个键原样塞给 CSSOM（`MindElixir.js:344-347`），
 * CSSOM 对无单位的非零长度按**无效值忽略**（规范行为），数字字号在
 * 编辑器里反而画不出来 —— 这个缺口由 `MindmapEditor` 的 `linkDiv`
 * 兜底同步补一次 `n + 'px'`（同 0056 徽章的同步时机）。
 * 带单位的字符串（别的工具写的 `'20px'`）渲染侧没问题、导出侧会被
 * 拼成非法值 —— 已知边界，如实保留，不替别人改写。
 */

/** 弹层可编辑的键 —— 与内核导出侧的映射范围逐字对齐（见文件头第二条）。 */
export type StyleKey = 'color' | 'background' | 'fontSize' | 'fontWeight';

/** 样式对象的形状：文件层是不透明的 `Record<string, unknown>`，这里同名同形。 */
export type StyleValues = Readonly<Record<string, unknown>>;

/**
 * 文字颜色。**具体色值，不走主题变量**：样式是文件内容，要跟着文件走
 * （换台机器、换个主题都长一个样）；而 `var()` 进了独立 SVG 解析不出来
 * （ADR 0050 给 `--bgcolor` 踩的同一条）。取值偏墨、压低饱和，
 * 在浅色纸面上都读得清。
 */
export const STYLE_TEXT_COLORS: readonly string[] = [
  '#333333',
  '#8f3b2c',
  '#a0651e',
  '#3f7a52',
  '#2f5f8f',
  '#6d4a8e',
  '#0f766e',
  '#5c6873',
];

/**
 * 背景色。淡彩底：默认墨色文字压上去仍然读得清。深色主题下浅底块
 * 与浅色文字的对比度**不保证** —— 样式是用户亲手刷的漆，跟着文件走
 * 不跟着主题翻（已知边界，重审条件见 ADR 0059）。
 */
export const STYLE_FILL_COLORS: readonly string[] = [
  '#fdf3d8',
  '#f9e7d1',
  '#f4e3dd',
  '#e6efe4',
  '#e3ecf3',
  '#e9e1f1',
  '#efeadd',
  '#ececec',
];

/** 字号档位。内核默认正文是 16px（`.map-container` 的字体大小），16 在表里、不亮就是「没设过」。 */
export const STYLE_FONT_SIZES: readonly number[] = [14, 16, 20, 26];

/** 「当前挂着这个值吗」—— 词表外的值（自定义色、`'20px'` 字符串字号）不亮：
 * 亮的是界面能一键摘掉的状态，摘不掉的不假装认识。 */
export function activeSwatchIndex(
  colors: readonly string[],
  style: StyleValues | undefined,
  key: StyleKey,
): number {
  const current = style?.[key];
  if (typeof current !== 'string') return -1;
  return colors.indexOf(current);
}

/** 字号亮哪一档：只有**数字**是词表认识的形状（见文件头第三条），带单位的字符串不亮。 */
export function activeFontSize(style: StyleValues | undefined): number | null {
  const current = style?.fontSize;
  return typeof current === 'number' && Number.isFinite(current) ? current : null;
}

/** 加粗亮不亮：词表只认 `'bold'`（点它写入的也是它）。`'600'` 一类词表外的值不亮、也不假装摘得掉。 */
export function isBold(style: StyleValues | undefined): boolean {
  return style?.fontWeight === 'bold';
}
