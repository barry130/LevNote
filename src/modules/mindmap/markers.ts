/**
 * 标记调色盘的词表（ADR 0053 —— 对标 XMind 的优先级 / 旗帜标记）。
 *
 * ## 词表住在这里，而格式层一个字都不认识它
 *
 * `.mindmap` 文件里存的是**显示字形本身**（`MindmapNode.icons`，理由见
 * `core/mindmap/types.ts` 的字段注释）。这张表只回答**界面**的三个问题：
 * 调色盘里有哪些字形、按什么分组、组内是否互斥。格式层（`parse.ts` /
 * `serialize.ts`）只认「字符串数组」，对词表一无所知 —— 于是文件里出现
 * 词表之外的字形照样原样保留（§5），换个内核也照样能显示。
 *
 * ## 组内互斥是什么意思
 *
 * 「优先级」与「色标」是**互斥组**：给节点标优先级 2，就同时摘掉优先级 1
 * —— 一个节点只有一个优先级，两种颜色叠在一起读不出任何意思。
 * 「符号」组不互斥：⭐ 与 ✅ 放在一个节点上是正常用法。
 * 这条语义也只住在这里（`applyMarkerToggle`）：文件层从不说「不许」，
 * 手写两个优先级进文件，内核照样两个都显示（诚实呈现，不由词表做裁判）。
 *
 * ## XMind 的标记 id 与这张表的关系
 *
 * XMind 文件里存的是语义 id（`priority-1` 一类）加它自己的图标资产。
 * 我们存字形。将来做 XMind 互导，id ↔ 字形的对照表住在**互导那一层**，
 * 不反向渗回格式层 —— 那正是「字形即内容」换来的自由。
 */

import type { MessageKey } from '../../core/i18n/messages.js';

export type MarkerGroupId = 'priority' | 'color' | 'symbol';

/** 一个调色盘条目：字形、所属组、悬停文案。 */
export interface MindmapMarker {
  /** 显示字形本身 —— 也就是写进文件的那个字符串。 */
  readonly glyph: string;
  readonly group: MarkerGroupId;
  /** 悬停文案的 i18n key（强类型的 `MessageKey` —— 词表里写错键名是编译错误，不是运行时空串）。 */
  readonly labelKey: MessageKey;
  /** `labelKey` 的插值参数（可缺省）。 */
  readonly labelParams?: Readonly<Record<string, string | number>>;
}

/** 组：显示顺序、互斥性、组头文案。 */
export interface MindmapMarkerGroup {
  readonly id: MarkerGroupId;
  readonly labelKey: MessageKey;
  /** 组内一次只能有一个字形（见文件头）。 */
  readonly exclusive: boolean;
}

export const MINDMAP_MARKER_GROUPS: readonly MindmapMarkerGroup[] = [
  { id: 'priority', labelKey: 'ui.mindmap.markerGroupPriority', exclusive: true },
  { id: 'color', labelKey: 'ui.mindmap.markerGroupColor', exclusive: true },
  { id: 'symbol', labelKey: 'ui.mindmap.markerGroupSymbol', exclusive: false },
];

export const MINDMAP_MARKERS: readonly MindmapMarker[] = [
  // 优先级 1–9：XMind 的第一屏核心能力，数字键盘字形一看就懂。
  { glyph: '1️⃣', group: 'priority', labelKey: 'ui.mindmap.markerPriority', labelParams: { n: '1' } },
  { glyph: '2️⃣', group: 'priority', labelKey: 'ui.mindmap.markerPriority', labelParams: { n: '2' } },
  { glyph: '3️⃣', group: 'priority', labelKey: 'ui.mindmap.markerPriority', labelParams: { n: '3' } },
  { glyph: '4️⃣', group: 'priority', labelKey: 'ui.mindmap.markerPriority', labelParams: { n: '4' } },
  { glyph: '5️⃣', group: 'priority', labelKey: 'ui.mindmap.markerPriority', labelParams: { n: '5' } },
  { glyph: '6️⃣', group: 'priority', labelKey: 'ui.mindmap.markerPriority', labelParams: { n: '6' } },
  { glyph: '7️⃣', group: 'priority', labelKey: 'ui.mindmap.markerPriority', labelParams: { n: '7' } },
  { glyph: '8️⃣', group: 'priority', labelKey: 'ui.mindmap.markerPriority', labelParams: { n: '8' } },
  { glyph: '9️⃣', group: 'priority', labelKey: 'ui.mindmap.markerPriority', labelParams: { n: '9' } },
  // 色标六色：给主题做视觉分类。emoji 里没有「彩色旗帜」，用色块如实呈现。
  { glyph: '🟥', group: 'color', labelKey: 'ui.mindmap.markerColorRed' },
  { glyph: '🟧', group: 'color', labelKey: 'ui.mindmap.markerColorOrange' },
  { glyph: '🟨', group: 'color', labelKey: 'ui.mindmap.markerColorYellow' },
  { glyph: '🟩', group: 'color', labelKey: 'ui.mindmap.markerColorGreen' },
  { glyph: '🟦', group: 'color', labelKey: 'ui.mindmap.markerColorBlue' },
  { glyph: '🟪', group: 'color', labelKey: 'ui.mindmap.markerColorPurple' },
  // 符号：不互斥，可叠加。
  { glyph: '⭐', group: 'symbol', labelKey: 'ui.mindmap.markerSymbolStar' },
  { glyph: '✅', group: 'symbol', labelKey: 'ui.mindmap.markerSymbolDone' },
  { glyph: '❗', group: 'symbol', labelKey: 'ui.mindmap.markerSymbolImportant' },
  { glyph: '❓', group: 'symbol', labelKey: 'ui.mindmap.markerSymbolQuestion' },
];

/** 字形 → 所属组。不在词表里的字形交 `null`（文件里可以有，调色盘里没有）。 */
export function markerGroupOf(glyph: string): MarkerGroupId | null {
  return MINDMAP_MARKERS.find((marker) => marker.glyph === glyph)?.group ?? null;
}

/**
 * 把一次「点了调色盘里的某个字形」折进当前标记序列，返回**新数组**（不改入参）。
 *
 * - 已有该字形 → 摘掉（任何组都一样：再点一次就是取消）；
 * - 没有、且属于互斥组 → 先摘掉同组其他字形，再把它**追加到末尾**
 *   （末尾追加让「后标的排在后面」成为稳定规则，文件往返不会乱序）；
 * - 没有、不互斥 → 追加。
 *
 * 词表之外的字形（`markerGroupOf` 交 `null`）按「不互斥」处理 ——
 * 这个函数面对的是**任意文件内容**，不是只有调色盘产出的内容。
 */
export function applyMarkerToggle(current: readonly string[], glyph: string): string[] {
  if (current.includes(glyph)) return current.filter((entry) => entry !== glyph);

  const group = markerGroupOf(glyph);
  const exclusive = group !== null && (MINDMAP_MARKER_GROUPS.find((entry) => entry.id === group)?.exclusive ?? false);
  const next = exclusive ? current.filter((entry) => markerGroupOf(entry) !== group) : [...current];
  next.push(glyph);
  return next;
}
