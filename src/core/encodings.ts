/**
 * 文档编码的候选清单 —— **只有这一份**。
 *
 * 状态栏的下拉与菜单栏的「编码」菜单都从它生成（同一份选项、同一批标签、
 * 同一种键）。抄成两份的下场与 Markdown 扩展名名单一样：菜单里能选的
 * 下拉里没有，或者两边写着不同的名字。
 */

import type { BomKind } from './document/types.js';
import { t } from './i18n/index.js';
import type { MessageKey } from './i18n/messages.js';

export interface EncodingOption {
  /** 界面上显示的名字（状态栏与菜单共用）。 */
  readonly label: string;
  /**
   * 需要翻译的条目给 key、渲染时现取（静态清单存 key 是 i18n 的既定纪律，
   * 模块加载时求值会把语言钉死在启动那一刻）。纯技术名（UTF-8、GBK …）
   * 两种语言长得一样，不给 key。
   */
  readonly labelKey?: MessageKey;
  /** 传给读写层的编码名（Rust 侧按它解码 / 编码）。 */
  readonly encoding: string;
  readonly bom: BomKind;
}

export const ENCODINGS: readonly EncodingOption[] = [
  { label: 'UTF-8', encoding: 'UTF-8', bom: 'none' },
  { label: 'UTF-8（含 BOM）', labelKey: 'core.encoding.utf8Bom', encoding: 'UTF-8', bom: 'utf-8' },
  { label: 'GBK', encoding: 'GBK', bom: 'none' },
  { label: 'GB18030', encoding: 'GB18030', bom: 'none' },
  { label: 'Big5', encoding: 'Big5', bom: 'none' },
  { label: 'UTF-16 LE', encoding: 'UTF-16LE', bom: 'utf-16le' },
  { label: 'UTF-16 BE', encoding: 'UTF-16BE', bom: 'utf-16be' },
  { label: 'windows-1252', encoding: 'windows-1252', bom: 'none' },
];

/** 界面上该显示的名字：有 key 的条目按当前语言现取，其余用技术名。 */
export function encodingLabel(option: EncodingOption): string {
  return option.labelKey !== undefined ? t(option.labelKey) : option.label;
}

/**
 * 一份文档的「编码 + BOM」在下拉 / 菜单里的稳定键。
 *
 * 磁盘上实际探测出来的编码可能**不在这个候选清单里**（chardetng 会给
 * 一长串罕见编码）：这种情况回 `other|…`，界面显示原样而不是空白。
 */
export function encodingKey(encoding: string, bom: BomKind): string {
  const hit = ENCODINGS.find(
    (item) => item.encoding.toUpperCase() === encoding.toUpperCase() && item.bom === bom,
  );
  if (hit) return `${hit.encoding}|${hit.bom}`;
  return `other|${encoding}|${bom}`;
}

/** 编码命令 id 的稳定片段：`encoding.set.<slug>`。菜单、命令、勾选判断共用。 */
export function encodingSlug(encoding: string, bom: BomKind): string {
  return `${encoding}-${bom}`.toLowerCase().replace(/[^a-z0-9]+/g, '-');
}
