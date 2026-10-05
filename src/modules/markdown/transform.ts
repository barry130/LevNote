import type { Editor } from '@milkdown/kit/core';
import { parserCtx, serializerCtx } from '@milkdown/kit/core';
import type { Node as ProseNode } from '@milkdown/kit/prose/model';
import { TextSelection } from '@milkdown/kit/prose/state';
import type { EditorView } from '@milkdown/kit/prose/view';
import { editorViewCtx } from '@milkdown/kit/core';

import type { EditorBlockTransform, EditorFenceInfo } from '../../core/editor/types.js';
import { CODE_BLOCK } from './code-block-view.js';
import { withUserEdit } from './user-edit.js';

/**
 * 块级结构变换命令面（P6 第 28 项的 L3）：提取为引用、拉平引用的**内核一半**。
 *
 * 分工与图片粘贴是同构的：**文档坐标的事归内核，文件系统的事归外壳**。
 * 「哪一段被选中」「光标落在哪个围栏块里」「替换之后选区去哪」都是文档模型
 * 的知识；外壳拿到的是 Markdown 源（与 `roundTrip` 同一条理由——
 * 翻译由内核自己做），换算成文件与引用后原样交回来替换。
 *
 * 两个写方法都走 `withUserEdit`：它们由命令触发、不经过键盘事件，
 * 不打标记就是「文档不变脏、Ctrl+S 写回旧内容」的静默丢数据（硬约束 7）。
 */
export function createBlockTransform(editor: Editor): EditorBlockTransform {
  /** 光标所在（或选中范围覆盖）的围栏块及其文档位置。找不到返回 null。 */
  const enclosingFenceAt = (view: EditorView): { node: ProseNode; from: number; to: number } | null => {
    const { $from } = view.state.selection;
    for (let depth = $from.depth; depth >= 1; depth -= 1) {
      const node = $from.node(depth);
      if (node.type.name === CODE_BLOCK) {
        return { node, from: $from.before(depth), to: $from.after(depth) };
      }
    }
    return null;
  };

  return {
    selectedMarkdown(): string | null {
      return editor.action((ctx) => {
        const view = ctx.get(editorViewCtx);
        const { from, to, empty } = view.state.selection;
        if (empty) return null;
        // `cut` 给出的是**同类型**的新节点（doc），跨块选区的两端若切在段落中间，
        // 序列化出来的就是那半段 —— 提取成独立文件后它自成一篇，正是想要的。
        return ctx.get(serializerCtx)(view.state.doc.cut(from, to));
      });
    },

    replaceSelectionWith(markdown: string): void {
      withUserEdit(editor, () => {
        editor.action((ctx) => {
          const view = ctx.get(editorViewCtx);
          const { from, to } = view.state.selection;
          const tr =
            markdown === ''
              ? view.state.tr.delete(from, to)
              : view.state.tr.replaceWith(from, to, ctx.get(parserCtx)(markdown).content);
          // 光标落到替换内容之后：用户提取完接着写正文，而不是跳回文首
          tr.setSelection(TextSelection.near(tr.doc.resolve(tr.mapping.map(to)), -1));
          tr.scrollIntoView();
          view.dispatch(tr);
        });
      });
    },

    enclosingFence(): EditorFenceInfo | null {
      return editor.action((ctx) => {
        const fence = enclosingFenceAt(ctx.get(editorViewCtx));
        if (fence === null) return null;
        const info = fence.node.attrs['language'];
        return {
          info: typeof info === 'string' ? info : '',
          // 围栏内的原文：属性区与载荷都在里面（与 `parseDocBlock` 的输入一致）
          content: fence.node.textContent,
        };
      });
    },

    replaceEnclosingFenceWith(markdown: string): void {
      withUserEdit(editor, () => {
        editor.action((ctx) => {
          const view = ctx.get(editorViewCtx);
          const fence = enclosingFenceAt(view);
          if (fence === null) return;
          const tr = view.state.tr.replaceWith(
            fence.from,
            fence.to,
            ctx.get(parserCtx)(markdown).content,
          );
          tr.setSelection(TextSelection.near(tr.doc.resolve(tr.mapping.map(fence.to)), -1));
          tr.scrollIntoView();
          view.dispatch(tr);
        });
      });
    },
  };
}
