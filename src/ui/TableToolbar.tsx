/**
 * 光标落进表格时浮出来的表格工具条。
 *
 * ## 为什么不做成表格四周的把手
 *
 * Typora / Notion 的做法是：鼠标悬停时在表格左侧 / 顶部浮出小把手，
 * 拖拽移动整行整列、点击选中。那需要**知道鼠标在表格的哪一行哪一列上**，
 * 也就是要监听 `mousemove` 并把屏幕坐标反解成行列 —— 那是布局引擎的活儿，
 * 而这个组件所在的外壳拿不到（内核给的是「光标在哪个格子」）。
 *
 * 所以这里走的是另一条路：**按光标所在的位置操作**（在上面插一行、删掉这一列），
 * 按钮上的文字直接说明会发生什么。少一点花哨，但没有一条操作依赖猜坐标。
 * 等真有人抱怨「想拖拽移动行」时再说 —— 那需要内核先能报出鼠标命中的格子。
 *
 * ## 一个不那么直觉的行为：删表头行
 *
 * Markdown 的表格**必须**有表头行（`table_header_row table_row+`），所以「删掉表头行」
 * 不是一种可表示的状态。内核的处理是：删掉它，并把下一行**顶上来当表头**。
 * 工具条不额外解释这件事（`行✕` 的 `title` 已经够长了），但按钮的禁用条件是
 * **行数**（≥ 3 才可删），与光标在哪一行无关 —— 这一点与内核的守卫完全一致。
 *
 * ## 为什么按钮这么挤
 *
 * 十一个操作排成两行，用的是短标签（`行↑` / `列✕` / `删表`）。
 * 完整的说法放在 `title` 与 `aria-label` 里 —— 这是刻意的：
 * 表格工具条盖在表格上，越宽挡得越多，而用户此刻正盯着表格看。
 */

import { useLayoutEffect, useRef, useState } from 'react';

import type { EditorTable, TableAlign, TableOperation, TableState } from '../core/editor/types.js';
import { t, type MessageKey } from '../core/i18n/index.js';
import { computeFloatingPlacement } from './floating.js';

interface OpButton {
  readonly op: TableOperation;
  /** 按钮上的紧凑可见文案也随语言走（行↑ / Row ↑），存 key 渲染时现取。 */
  readonly labelKey: MessageKey;
  /** 完整说法（title / aria），比短标签多说一句「这是在干嘛」。 */
  readonly titleKey: MessageKey;
}

const ROW_OPS: readonly OpButton[] = [
  { op: 'rowBefore', labelKey: 'ui.table.rowBeforeLabel', titleKey: 'ui.table.rowBefore' },
  { op: 'rowAfter', labelKey: 'ui.table.rowAfterLabel', titleKey: 'ui.table.rowAfter' },
  { op: 'deleteRow', labelKey: 'ui.table.deleteRowLabel', titleKey: 'ui.table.deleteRow' },
];

const COL_OPS: readonly OpButton[] = [
  { op: 'colBefore', labelKey: 'ui.table.colBeforeLabel', titleKey: 'ui.table.colBefore' },
  { op: 'colAfter', labelKey: 'ui.table.colAfterLabel', titleKey: 'ui.table.colAfter' },
  { op: 'deleteColumn', labelKey: 'ui.table.deleteColumnLabel', titleKey: 'ui.table.deleteColumn' },
];

const TABLE_OPS: readonly OpButton[] = [
  { op: 'exit', labelKey: 'ui.table.exitLabel', titleKey: 'ui.table.exit' },
  { op: 'deleteTable', labelKey: 'ui.table.deleteTableLabel', titleKey: 'ui.table.deleteTable' },
];

const ALIGN_OPS: ReadonlyArray<{ align: TableAlign; labelKey: MessageKey; titleKey: MessageKey }> = [
  { align: 'left', labelKey: 'ui.table.alignLeftLabel', titleKey: 'ui.table.alignLeft' },
  { align: 'center', labelKey: 'ui.table.alignCenterLabel', titleKey: 'ui.table.alignCenter' },
  { align: 'right', labelKey: 'ui.table.alignRightLabel', titleKey: 'ui.table.alignRight' },
];

export interface TableToolbarProps {
  state: TableState;
  table: EditorTable;
}

export function TableToolbar({ state, table }: TableToolbarProps) {
  const { anchor } = state;
  const rootRef = useRef<HTMLDivElement | null>(null);
  const [placement, setPlacement] = useState<{ left: number; top: number } | null>(null);

  useLayoutEffect(() => {
    const node = rootRef.current;
    if (node === null) return;

    const { width, height } = node.getBoundingClientRect();

    // 优先摆在表格**下方**：表格上方通常是刚才在写的那一段。
    // 摆不下的情况（表格很长 / 贴着视口底）会翻到上方或被夹回视口内
    setPlacement(
      computeFloatingPlacement(
        anchor,
        { width, height },
        { vertical: 'below', horizontal: 'start' },
        { width: window.innerWidth, height: window.innerHeight },
      ),
    );
  }, [anchor]);

  const renderOp = (button: OpButton, disabled: boolean, disabledTitle?: string) => (
    <button
      key={button.op}
      type="button"
      className="tt-btn"
      // 禁用时说明**为什么**不能点 —— 一个灰着的按钮不说理由，
      // 用户只会觉得它坏了
      title={disabled && disabledTitle !== undefined ? disabledTitle : t(button.titleKey)}
      aria-label={t(button.titleKey)}
      disabled={disabled}
      // 不拦这一下，焦点会离开编辑器，光标从表格里消失，工具条跟着一起没了
      onMouseDown={(event) => event.preventDefault()}
      onClick={() => table.run(button.op)}
    >
      {t(button.labelKey)}
    </button>
  );

  return (
    <div
      ref={rootRef}
      className="table-toolbar"
      role="toolbar"
      aria-label={t('ui.table.groupLabel')}
      style={{
        left: placement?.left ?? 0,
        top: placement?.top ?? 0,
        visibility: placement === null ? 'hidden' : 'visible',
      }}
    >
      <div className="tt-row">
        {ROW_OPS.map((button) =>
          renderOp(
            button,
            button.op === 'deleteRow' && !state.canDeleteRow,
            t('ui.table.minRows'),
          ),
        )}
        <span className="tt-sep" />
        {COL_OPS.map((button) =>
          renderOp(button, button.op === 'deleteColumn' && !state.canDeleteColumn, t('ui.table.minCols')),
        )}
        <span className="tt-sep" />
        <span className="tt-size" title={t('ui.table.size')}>
          {state.columns} × {state.rows}
        </span>
      </div>

      <div className="tt-row">
        {ALIGN_OPS.map((button) => (
          <button
            key={button.align}
            type="button"
            className={`tt-btn tt-align ${state.align === button.align ? 'active' : ''}`}
            title={t(button.titleKey)}
            aria-label={t(button.titleKey)}
            aria-pressed={state.align === button.align}
            onMouseDown={(event) => event.preventDefault()}
            onClick={() => table.setAlign(button.align)}
          >
            {t(button.labelKey)}
          </button>
        ))}
        <span className="tt-sep" />
        {TABLE_OPS.map((button) => renderOp(button, false))}
      </div>
    </div>
  );
}
