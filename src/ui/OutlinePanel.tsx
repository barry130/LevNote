/**
 * 大纲面板。
 *
 * 长文档的导航刚需。它与 Markdown 内核**解耦**：大纲直接从 `doc.text`
 * 用纯函数算出来（`core/outline/outline.ts`），因此
 * 打开一份 10 万字的文档时，算大纲不会去碰编辑器内核，
 * 也不会因为内核还在异步初始化而拿不到内容。
 *
 * 跳转走 `EditorHandle.goToOutlineItem`。不支持跳转的内核（纯文本兜底）
 * 会把条目显示成不可点击并说明原因 —— 而不是给一个点了没反应的列表。
 */

import { useMemo, useSyncExternalStore } from 'react';

import type { DocumentStore } from '../core/document/store.js';
import { emptyStoreState } from '../core/document/store.js';
import { extractOutline, outlineDepths, type OutlineItem } from '../core/outline/outline.js';
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

  const items = useMemo(() => extractOutline(text), [text]);
  const depths = useMemo(() => outlineDepths(items), [items]);

  return (
    <aside className="outline">
      <header className="outline-head">
        <span className="outline-title">{t('ui.outline.title')}</span>
        <span className="grow" />
        <span className="hint">{t('ui.outline.count', { n: items.length })}</span>
        <button type="button" title={t('ui.outline.close')} onClick={onClose}>
          ×
        </button>
      </header>

      {items.length === 0 ? (
        <div className="outline-empty">
          {text === '' ? t('ui.outline.emptyDoc') : t('ui.outline.noHeadings')}
          <br />
          {t('ui.outline.howHint')}
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
              title={canJump ? t('ui.outline.lineN', { line: item.line }) : t('ui.outline.noJump')}
              onClick={() => onJump(item)}
            >
              {item.text === '' ? <span className="hint">{t('ui.outline.emptyHeading')}</span> : item.text}
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
