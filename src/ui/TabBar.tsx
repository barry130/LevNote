/**
 * 标签栏。
 *
 * 这是「单窗口装下多个文档」这个产品主张的**可见形态**（与 horseMD 的核心差异点）。
 * 因此它的每一条交互都必须经得起高频使用：中键关闭、拖拽重排、
 * 关闭时接管邻居 —— 这些不是锦上添花，是「一天点几百次」的东西。
 */

import { useState } from 'react';

import { t } from '../core/i18n/index.js';
import type { TabSummary } from '../core/workspace/store.js';

export interface TabBarProps {
  tabs: readonly TabSummary[];
  activeId: number | null;
  /**
   * 「就地编辑」的返回目标（P6 第 28 项 L2，ADR 0036）：当前活动标签页是
   * 从哪个标签页钻过来的。null = 没有钻取上下文，不画 chip。
   */
  backTo: { id: number; name: string } | null;
  onBack: () => void;
  onActivate: (id: number) => void;
  onClose: (id: number) => void;
  onReorder: (from: number, to: number) => void;
}

export function TabBar({ tabs, activeId, backTo, onBack, onActivate, onClose, onReorder }: TabBarProps) {
  /** 正在被拖的标签页下标。用 state 而不是 dataTransfer：拖拽过程中要实时画落点。 */
  const [dragging, setDragging] = useState<number | null>(null);
  const [overAt, setOverAt] = useState<number | null>(null);

  if (tabs.length === 0) return null;

  const drop = (to: number): void => {
    if (dragging !== null && dragging !== to) onReorder(dragging, to);
    setDragging(null);
    setOverAt(null);
  };

  return (
    <div className="tabbar" role="tablist">
      {backTo !== null && (
        /*
         * 「↩ 返回」chip（ADR 0036）：点卡片上的「就地编辑」钻进来之后，
         * 一键回来源 —— 面板常驻保证回去时撤销栈 / 光标 / 滚动原样。
         * 放在标签页列表的**最左边**：它是当前标签页的导航上下文，
         * 不是一枚可点开的标签页。
         */
        <button type="button" className="tab-back" title={t('ui.tabs.backTo', { name: backTo.name })} onClick={onBack}>
          ↩ {backTo.name}
        </button>
      )}
      {tabs.map((tab, index) => (
        <div
          key={tab.id}
          role="tab"
          aria-selected={tab.id === activeId}
          tabIndex={-1}
          className={[
            'tab',
            tab.id === activeId ? 'active' : '',
            dragging === index ? 'dragging' : '',
            overAt === index && dragging !== index ? 'drop-target' : '',
          ]
            .filter(Boolean)
            .join(' ')}
          title={tab.path}
          onClick={() => onActivate(tab.id)}
          // 中键关闭：浏览器里关标签页的习惯，用户不需要先瞄准那个小叉
          onAuxClick={(event) => {
            if (event.button === 1) {
              event.preventDefault();
              onClose(tab.id);
            }
          }}
          draggable
          onDragStart={(event) => {
            setDragging(index);
            // Firefox 要求设置 dataTransfer 才会真正开始拖拽
            event.dataTransfer.setData('text/plain', tab.path);
            event.dataTransfer.effectAllowed = 'move';
          }}
          onDragOver={(event) => {
            event.preventDefault();
            event.dataTransfer.dropEffect = 'move';
            setOverAt(index);
          }}
          onDragLeave={() => setOverAt((at) => (at === index ? null : at))}
          onDrop={(event) => {
            event.preventDefault();
            drop(index);
          }}
          onDragEnd={() => {
            setDragging(null);
            setOverAt(null);
          }}
        >
          {tab.dirty && <span className="dot" title={t('ui.tabs.dirty')} />}
          <span className="tab-name">{tab.displayName}</span>
          <button
            type="button"
            className="tab-close"
            aria-label={t('ui.tabs.close', { name: tab.displayName })}
            title={t('ui.tabs.closeShortcut')}
            onClick={(event) => {
              // 不加这一句，点关闭会先冒泡到标签页的 onClick 把它激活一次
              event.stopPropagation();
              onClose(tab.id);
            }}
          >
            ×
          </button>
        </div>
      ))}
    </div>
  );
}
