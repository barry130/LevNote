/**
 * 从 `.mindmap` 文本里提取主题大纲（ADR 0052 —— 对标 XMind 的大纲视图）。
 *
 * 与 Markdown 那份（`outline.ts`）同一条纪律：**不经过编辑器内核**的纯函数，
 * 这样它对一个还没挂载内核的文档也能用，也才好测。
 *
 * ## 为什么复用 `parseMindmap`，而不是自己 `JSON.parse` 一遍
 *
 * 大纲必须和「喂给内核的那份数据」是**同一套语义**：`topic` 的类型纠正
 * （数字 / 布尔 → 字符串）、缺 id 时补的 `auto-` 前缀、`children` 的形状 ——
 * 全在解析器里。在这里再写一遍「什么算一个节点」，两边迟早漂移：
 * 最典型的后果是大纲里的 id 和内核 DOM 里的 `data-nodeid` 对不上，
 * 跳转悄悄指错地方。判定只有一份，就是解析器。
 *
 * 代价也如实写：`parseMindmap` 会顺手做一次完整校验（含序列化比对），
 * 比裸 `JSON.parse` 贵。大纲每次文本变化重算一次，对真实的导图体量
 * （几百个节点）这是毫秒级的事，与 Markdown 大纲逐行扫描同量级。
 *
 * ## 解析不出来的时候
 *
 * 返回空数组 —— 面板据此说「暂时读不出导图结构」，而不是给它一个假大纲
 * （硬约束 2）。真实场景：源码视图里手改 JSON 的中间态。
 */

import { parseMindmap } from '../mindmap/parse.js';
import type { MindmapNode } from '../mindmap/types.js';
import type { OutlineItem } from './outline.js';

/** 导图大纲项：多一个**必有**的文件层节点 id（跳转靠它，不靠文字）。 */
export interface MindmapOutlineItem extends OutlineItem {
  readonly id: string;
}

/**
 * 提取导图大纲。
 *
 * - **顺序**是先序遍历（根 → 子 → 孙），与查找导航的「文档顺序」同一条
 *   （`EditorSearch` 契约）—— 大纲也是导航，顺序必须一致。
 * - **层级**就是树深（根 = 1）。导图没有 Markdown 那种「跳级」
 *   （`###` 直接跟在 `#` 后面），树深天然连续，`outlineDepths` 算出来
 *   就是 `level - 1`，缩进直接复用。
 * - **折叠的子树照样列出来**（XMind 同款）：大纲是导航，导航要能去到
 *   「现在看不见」的地方 —— 内核跳转时会把折叠的祖先展开（见
 *   `MindmapEditor.goToOutlineItem`）。
 * - `line` 字段在导图里**不是行号**，是先序序号：面板拿它当 key 用，
 *   显示用 `text`，跳转用 `id`。诚实的理由：导图文档没有「行」这个概念
 *   （`goToOutlineItem` 契约注释里同一条）。
 */
export function extractMindmapOutline(text: string): MindmapOutlineItem[] {
  const parsed = parseMindmap(text);
  const document = parsed.document;
  if (document === null) return [];

  const items: MindmapOutlineItem[] = [];
  let ordinal = 0;
  const walk = (node: MindmapNode, level: number): void => {
    ordinal += 1;
    items.push({ level, text: node.topic, line: ordinal, id: node.id });
    for (const child of node.children) walk(child, level + 1);
  };
  walk(document.root, 1);
  return items;
}
