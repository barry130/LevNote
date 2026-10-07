/**
 * 视图三档切换：富文本 / 源码 / 分屏。
 *
 * ## 为什么是一个「三选一」而不是两个开关
 *
 * 「源码」和「分屏」不是两个独立的维度：分屏的左栏**就是**源码视图，
 * 所以「源码 + 分屏同时为真」这个状态不存在。用两个布尔值表达它，
 * 就等于允许界面进入一个自己都说不清的状态 —— 而那种状态下
 * 两个按钮的亮灭组合看起来都是合理的，用户没法判断哪个对。
 *
 * 枚举（`ViewMode`）把这个不可能的状态从类型上就消掉了。
 *
 * ## 为什么放在状态栏
 *
 * 它描述的是「**当前这一份文档**现在怎么看」，与光标行列、字数、编码
 * 是同一类信息 —— 都属于「文档此刻的样子」，而不是「要对文档做什么」。
 * （曾经的顶部工具栏放的是后者 —— 打开 / 保存 / 导出；那排按钮在菜单栏
 * 落地后整排撤掉，「源码 / 富文本」两态开关并入了菜单的 view.toggleSource，
 * 与这里的三选一并存是刻意的：一个是在两边之间来回，一个是直接跳过去。）
 */

import type { ViewMode } from '../core/document/store.js';
import { t, type MessageKey } from '../core/i18n/index.js';

/**
 * 静态数据只存 key，渲染时才 `t()`：模块加载时求值会把语言**钉死在启动那一刻**，
 * 换语言后这三个按钮不会跟着变。同一条纪律适用于所有「常量清单 + 文案」的组件
 * （块手柄、斜杠菜单、表格工具条……）。
 */
const MODES: ReadonlyArray<{ value: ViewMode; labelKey: MessageKey; hintKey: MessageKey }> = [
  { value: 'rich', labelKey: 'ui.view.rich.label', hintKey: 'ui.view.rich.hint' },
  { value: 'source', labelKey: 'ui.view.source.label', hintKey: 'ui.view.source.hint' },
  { value: 'split', labelKey: 'ui.view.split.label', hintKey: 'ui.view.split.hint' },
];

export interface ViewSwitcherProps {
  value: ViewMode;
  onChange: (mode: ViewMode) => void;
}

export function ViewSwitcher({ value, onChange }: ViewSwitcherProps) {
  return (
    <span className="view-switcher" role="group" aria-label={t('ui.view.groupLabel')}>
      {MODES.map(({ value: mode, labelKey, hintKey }) => (
        <button
          key={mode}
          type="button"
          // 用 `aria-pressed` 而不是 `aria-current`：这三个是**开关组**，
          // 语义上「这一个开着」，不是「你在这里」。
          aria-pressed={value === mode}
          title={t(hintKey)}
          onClick={() => onChange(mode)}
        >
          {t(labelKey)}
        </button>
      ))}
    </span>
  );
}
