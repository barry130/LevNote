import type { Editor } from '@milkdown/kit/core';
import type { EditorView } from '@milkdown/kit/prose/view';
import { editorViewCtx } from '@milkdown/kit/core';

import { USER_EDIT_META } from './plugin.js';

/**
 * 把一次编辑标记成「用户发起的」。
 *
 * ## 少了这一步会**静默丢数据**（硬约束 7）
 *
 * `markdownEmitPlugin` 只在「刚发生过用户输入」的窗口内上报变更，而点工具条上的
 * 按钮、执行查找替换、做块级结构变换**都不产生任何 `input` / `paste` 事件** ——
 * 没有这个标记，改完之后文档不会变脏，用户按 Ctrl+S，写回磁盘的是改动前的
 * 旧内容，而界面一直显示「已保存」。已经踩过六次（查找替换、浮层格式化、
 * 斜杠菜单、表格的每一个操作、代码块改语言、图片插入）。
 *
 * 这与 `USER_EDIT_META` 当初为「查找替换」加上的理由是**同一条**：
 * 由用户动作触发、但不经过键盘事件的编辑，必须自己声明身份。
 *
 * ## 为什么是临时换掉 `view.dispatch`
 *
 * 命令由 Milkdown 的 `CommandManager` 内部派发（`command(view.state, view.dispatch, view)`），
 * 中间没有任何可以插进去的钩子，也没有 `dispatchTransaction` 选项被用到。
 * 换掉再在 `finally` 里换回来，作用范围只有这一次同步调用 ——
 * 不是全局补丁，也不会漏到别的派发路径上。`dispatch` 是实例上的可写属性
 * （`prosemirror-view` 里声明为 `dispatch: (tr) => void`），所以这样做是合法的。
 *
 * 这个实现原来长在 `setup.ts` 里；拆成本模块是因为**块级结构变换**
 * （`transform.ts`）需要同一份实现 —— 这类代码写成两份的后果是其中一份
 * 迟早漏打标记，而漏打的形态就是上面那句话。
 */
export function withUserEdit<T>(editor: Editor, run: () => T): T {
  return editor.action((ctx) => withUserEditOnView(ctx.get(editorViewCtx), run));
}

/**
 * 同上，但直接作用在一个 `view` 上。
 *
 * 拆出这一层是因为**粘贴图片**的入口拿不到 `editor`：它挂在
 * `editorViewOptionsCtx` 的 `handlePaste` 上，那个对象在 `Editor.make()` 的
 * `config` 阶段就要交出去，那时编辑器实例还不存在。而 ProseMirror 会把
 * `view` 作为 `handlePaste` 的第一个参数给我们 —— 所以真正需要的只是它。
 */
export function withUserEditOnView<T>(view: EditorView, run: () => T): T {
  const original = view.dispatch;
  view.dispatch = (tr) => original.call(view, tr.setMeta(USER_EDIT_META, true));
  try {
    return run();
  } finally {
    view.dispatch = original;
  }
}
