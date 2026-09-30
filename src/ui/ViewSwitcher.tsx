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
 * 工具栏上那些按钮全是后者（打开 / 保存 / 导出）。
 *
 * 与工具栏上那个「源码 / 富文本」开关**并存**是刻意的：
 * 那个是**两态开关**（按一下换一边，符合肌肉记忆），这里是**三选一**
 * （一次点到分屏）。两者都不替代对方。
 */

import type { ViewMode } from '../core/document/store.js';

const MODES: ReadonlyArray<{ value: ViewMode; label: string; hint: string }> = [
  { value: 'rich', label: '富文本', hint: '富文本视图（Ctrl+1）' },
  { value: 'source', label: '源码', hint: '源码视图（Ctrl+2）' },
  { value: 'split', label: '分屏', hint: '源码与预览并排（Ctrl+3）' },
];

export interface ViewSwitcherProps {
  value: ViewMode;
  onChange: (mode: ViewMode) => void;
}

export function ViewSwitcher({ value, onChange }: ViewSwitcherProps) {
  return (
    <span className="view-switcher" role="group" aria-label="视图模式">
      {MODES.map(({ value: mode, label, hint }) => (
        <button
          key={mode}
          type="button"
          // 用 `aria-pressed` 而不是 `aria-current`：这三个是**开关组**，
          // 语义上「这一个开着」，不是「你在这里」。
          aria-pressed={value === mode}
          title={hint}
          onClick={() => onChange(mode)}
        >
          {label}
        </button>
      ))}
    </span>
  );
}
