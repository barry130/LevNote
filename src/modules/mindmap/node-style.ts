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
 * ## 词表的范围（0059 的「恰好四键」在 0062 修过适用面）
 *
 * 文本类键（`color` / `background` / `fontSize` / `fontWeight`）受**导出白名单**
 * 约束：`LayoutSsr.js` 的 `foreignObject` 只拼 `font-family / font-size /
 * font-weight / color` 四样（:2607），多一个键就是「编辑器画得出、导出的图里
 * 没有」（ADR 0050 要拦的落差）。
 *
 * **边框类键不受这个约束**（ADR 0062 探针）：导出侧对节点矩形是
 * **专门通道** —— `Xo` / `se`（`MindElixir.js:2611-2635`）把 `getComputedStyle`
 * 的 `borderColor → stroke`、`borderWidth → stroke-width`、
 * `borderRadius → rx/ry` **现读**到 SVG `rect` 上，**没有白名单**，DOM 上
 * 长什么样（无论谁写上去的）导出就长什么样。唯一的例外是
 * `borderStyle`：导出的 rect **没有** `stroke-dasharray` 映射 —— 编辑器里
 * 画的虚线，导出的图里是实线。这个落差不静默：`countDashedBorders`
 * 数出来交给 `exportImage` 的 `warnings`（0050 的管道）。
 *
 * ## `fontSize` 为什么是数字（`borderWidth` 为什么是带单位字符串）
 *
 * 内核导出时按 `font-size: ${v}px` 拼接 —— 值必须是数字才拼得对；
 * 而内核渲染侧把 `style` 每个键原样塞给 CSSOM（`MindElixir.js:344-347`），
 * CSSOM 对无单位的非零长度按**无效值忽略**（规范行为），数字字号在
 * 编辑器里反而画不出来 —— 这个缺口由 `MindmapEditor` 的 `linkDiv`
 * 兜底同步补一次 `n + 'px'`（同 0056 徽章的同步时机）。
 * `borderWidth` 走的是另一条路：导出**现读** computedStyle（不拼字符串），
 * 带不带单位导出都吃得下，而渲染侧塞 CSSOM **必须要单位** —— 所以
 * 词表存 `'2px'` 这样的带单位字符串，两边都成立，不需要任何同步器。
 * 带单位的字符串字号（别的工具写的 `'20px'`）渲染侧没问题、导出侧会被
 * 拼成非法值 —— 已知边界，如实保留，不替别人改写。
 */

import type { MindmapNode } from '../../core/mindmap/types.js';

/** 弹层可编辑的键 —— 文本类四键见文件头（导出白名单），边框类三键见同上（导出专门通道）。 */
export type StyleKey =
  | 'color'
  | 'background'
  | 'fontSize'
  | 'fontWeight'
  | 'borderColor'
  | 'borderWidth'
  | 'borderStyle';

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

/**
 * 边框颜色（ADR 0062）。**复用文字色那张表、不另抄一份**（名单只有一份的纪律）：
 * 八个偏墨色同样适合描边 —— 边框要压得住浅底，与文字「在浅色纸面上读得清」
 * 是同一条判据。弹层里「边框」行的格子画的是**空心框**（`border` 即色值，
 * 见 `mindmap.css` 的 `.mm-style-swatch-border`），与「背景」的实心格、
 * 「文字」的 A 字、「连线」的横线天然可辨（0059 的四行可辨性问题五）。
 */
export const STYLE_BORDER_COLORS: readonly string[] = STYLE_TEXT_COLORS;

/**
 * 边框粗细档位。存**带单位字符串**：渲染侧 `Se` 原样塞 CSSOM，无单位长度
 * 被规范忽略（同 `fontSize` 的教训），而导出侧现读 computedStyle、带不带单位
 * 都吃得下 —— 带单位是两边都成立的唯一形状，**不需要** `syncStyleFontSizes`
 * 那样的 linkDiv 兜底（见文件头第三条）。
 */
export const STYLE_BORDER_WIDTHS: readonly string[] = ['1px', '2px', '3px'];

/**
 * 边框线型档位。`solid` 在表里、再点一次 = 摘键（不是「设成默认」—— 文件里
 * 留下显式的 `borderStyle: 'solid'` 与「没设过」是两个状态，摘掉才是干净）。
 * 虚线的导出边界见 `countDashedBorders`。
 */
export const STYLE_BORDER_LINE_STYLES: readonly string[] = ['solid', 'dashed'];

/**
 * 数「导出图里虚线会画成实线」的节点（ADR 0062）。导出的节点矩形
 * （`Xo` / `se`，`MindElixir.js:2611-2635`）只映射 `borderColor / borderWidth /
 * borderRadius` 三个计算样式，**没有 `stroke-dasharray`** —— `borderStyle`
 * 在编辑器里画得出来、导出图里静默变成实线。这个落差不静默（硬约束 2）：
 * `exportImage` 调它数个数，交给 `warnings` 说一句（0050 的管道）。
 *
 * **词表外的值照数**（`'dotted'`、`'double'` 一类别的工具写的）：导出丢线型
 * 对它们同样成立，按「实际会画成实线」的判据数，不按「是不是词表写的」数。
 */
export function countDashedBorders(root: MindmapNode): number {
  let count = 0;
  const visit = (node: MindmapNode): void => {
    // 只数「编辑器里真的画成了虚线」的节点：非字符串 / 空串在 CSSOM 赋值那一步
    // 就被忽略（规范行为），导出图里本来就没有那条线，数上就是虚报。
    const borderStyle = node.style?.borderStyle;
    if (typeof borderStyle === 'string' && borderStyle !== 'solid' && borderStyle !== '') count += 1;
    for (const child of node.children) visit(child);
  };
  visit(root);
  return count;
}

/**
 * 分支连线颜色（ADR 0060）。**与主题的主分支取色环是同一张表、且名单只有
 * 一份**：`MindmapEditor` 的 `LEVNOTE_THEME.palette` 从这里取（不抄第二份 ——
 * 名单出现两份的下场是「弹层给的颜色」与「摘掉后回到的默认轮转色」对不上）。
 *
 * 语义因此自洽：再点一次 = 摘掉 = **回到默认轮转**，而默认轮转恰好就是
 * 这张表。这八个品牌色本就是画连线用的（中等饱和、细线在深浅画布上都可见），
 * 不存在 0059 那样的「导出侧词表」约束 —— 导出克隆的是已渲染的连线
 * SVG，任何色值都带得走（ADR 0060 判据 2）。
 */
export const BRANCH_COLORS: readonly string[] = [
  '#2e7d6a',
  '#3f6ea8',
  '#8a5a86',
  '#e2603f',
  '#7a8690',
  '#b4d4c6',
  '#16302a',
  '#209fb5',
];

/** 分支连线亮哪一格：只有**非空字符串且在词表内**才亮 ——
 * 词表外的色值（别的工具写的自定义色）没有格子代表它，不亮、也不假装摘得掉
 * （摘除照样能摘，走「清除」的显式一刀，ADR 0060 取舍 ④）。 */
export function activeBranchIndex(colors: readonly string[], branchColor: unknown): number {
  if (typeof branchColor !== 'string' || branchColor === '') return -1;
  return colors.indexOf(branchColor);
}

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
