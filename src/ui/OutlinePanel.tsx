/**
 * 大纲面板。
 *
 * 长文档的导航刚需。它与编辑器内核**解耦**：大纲直接从 `doc.text`
 * 用纯函数算出来 —— Markdown 走 `core/outline/outline.ts`，
 * 导图走 `core/outline/mindmap-outline.ts`（ADR 0052，复用 `parseMindmap`，
 * 与喂给内核的数据同一套语义）。因此打开一份 10 万字的文档时，
 * 算大纲不会去碰编辑器内核，也不会因为内核还在异步初始化而拿不到内容。
 *
 * 跳转走 `EditorHandle.goToOutlineItem`。不支持跳转的内核（纯文本兜底）
 * 会把条目显示成不可点击并说明原因 —— 而不是给一个点了没反应的列表。
 */

import { useMemo, useSyncExternalStore } from 'react';

import type { DocumentStore } from '../core/document/store.js';
import { emptyStoreState } from '../core/document/store.js';
import { extractOutline, outlineDepths, type OutlineItem } from '../core/outline/outline.js';
import { extractMindmapOutline } from '../core/outline/mindmap-outline.js';
import { isMindmapPath } from '../core/path.js';
import { t } from '../core/i18n/index.js';

export interface OutlinePanelProps {
  store: DocumentStore | null;
  /** 当前内核是否支持按标题跳转 */
  canJump: boolean;
  onJump: (item: OutlineItem) => void;
  onClose: () => void;
}

export function OutlinePanel({ store, canJump, onJump, onClose }: OutlinePanelProps) {
  const state = useSyncExternalStore(store?.subscribe ?? NOOP_SUBSCRIBE, store?.getSnapshot ?? EMPTY_SNAPSHOT);
  const text = state.doc?.text ?? '';
  /** 提取器按**产品范围**分派（哪些文件有大纲是产品决策，见 App 的 `outlineEnabled`）。 */
  const mindmap = isMindmapPath(state.doc?.path ?? '');

  const items = useMemo(
    () => (mindmap ? extractMindmapOutline(text) : extractOutline(text)),
    [mindmap, text],
  );
  const depths = useMemo(() => outlineDepths(items), [items]);

  return (
    <aside className="outline">
      <header className="outline-head">
        <span className="outline-title">{t('ui.outline.title')}</span>
        <span className="grow" />
        <span className="hint">{mindmap ? t('ui.outline.countTopics', { n: items.length }) : t('ui.outline.count', { n: items.length })}</span>
        <button type="button" title={t('ui.outline.close')} onClick={onClose}>
          ×
        </button>
      </header>

      {items.length === 0 ? (
        <div className="outline-empty">
          {text === ''
            ? t('ui.outline.emptyDoc')
            : mindmap
              ? t('ui.outline.mmUnreadable')
              : t('ui.outline.noHeadings')}
          <br />
          {/* 「用 # 开头」只对 Markdown 有意义；导图解析不出来时如实说读不出，不教用户改行 */}
          {!mindmap && t('ui.outline.howHint')}
        </div>
      ) : (
        <div className="outline-list" role="tree">
          {items.map((item, index) => (
            <button
              key={`${item.line}-${index}`}
              type="button"
              role="treeitem"
              className={`outline-item level-${item.level}`}
              style={{ paddingLeft: 10 + (depths[index] ?? 0) * 12 }}
              disabled={!canJump}
              title={
                canJump
                  ? mindmap
                    ? t('ui.outline.jumpToTopic')
                    : t('ui.outline.lineN', { line: item.line })
                  : t('ui.outline.noJump')
              }
              onClick={() => onJump(item)}
            >
              {/* 标记字形排在文字前（ADR 0053，XMind 同款视觉）。原样 join，不做词表翻译 ——
                  大纲只负责「显示它有」，认字形是人的事。`aria-hidden`：字形已在前面的
                  title 文字里，读屏不必逐个念 emoji。Markdown 条目没有 markers，一段都不渲染。 */}
              {item.markers !== undefined && item.markers.length > 0 ? (
                <span className="outline-markers" aria-hidden={true}>
                  {item.markers.join('')}
                </span>
              ) : null}
              {item.text === '' ? (
                <span className="hint">{mindmap ? t('ui.outline.emptyTopic') : t('ui.outline.emptyHeading')}</span>
              ) : (
                item.text
              )}
              {/* 备注指示（ADR 0056）：📝 排在文字后 —— 标记是「这条的性质」（排前），备注是「这条还藏着话」（排后）。
                  悬停 `title` 给备注原文（面板只负责「显示它有」，不截断不翻译）；读屏念「备注」两个字
                  （`aria-label`），不逐个念 emoji。Markdown 条目没有 note，一段都不渲染。 */}
              {item.note !== undefined ? (
                <span className="outline-note" role="note" aria-label={t('ui.mindmap.noteBadgeTitle')} title={item.note}>
                  📝
                </span>
              ) : null}
            </button>
          ))}
        </div>
      )}
    </aside>
  );
}

// 没有活动文档时也要能稳定地订阅：这两个常量必须是**引用稳定**的，
// 否则 `useSyncExternalStore` 每次渲染都会重新订阅并判定「有变化」，直接死循环。
const NOOP_SUBSCRIBE = (): (() => void) => () => undefined;

// 兜底快照从 store 同源取，不在这里手抄字段表（理由见 `emptyStoreState` 的注释）
const EMPTY_SNAPSHOT_STATE = emptyStoreState();
const EMPTY_SNAPSHOT = () => EMPTY_SNAPSHOT_STATE;
