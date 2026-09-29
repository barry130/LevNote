/**
 * Markdown 内核的三个自研 ProseMirror 插件。
 *
 * 为什么自己写而不找现成包：Milkdown 生态里做代码高亮的包长期跟不上主版本
 * （`@milkdown/plugin-math` 已被官方标记为不再支持，同类插件也有版本错位问题）。
 * 这三个插件一共百来行，用标准 ProseMirror API 写出来比引入一个会随主版本漂移的依赖更稳。
 */

import { Plugin, PluginKey } from '@milkdown/kit/prose/state';
import type { Node as ProseNode } from '@milkdown/kit/prose/model';
import { Decoration, DecorationSet } from '@milkdown/kit/prose/view';
import { $prose } from '@milkdown/kit/utils';

import type { CursorPosition } from '../../core/editor/types.js';
import { highlightCode } from '../../core/highlight/hljs.js';

const highlightKey = new PluginKey<DecorationSet>('qingmo-code-highlight');
const cursorKey = new PluginKey('qingmo-cursor-reporter');
const emitKey = new PluginKey('qingmo-markdown-emit');
const placeholderKey = new PluginKey('qingmo-placeholder');

/** ProseMirror 的 code_block 节点名与语言属性名（对齐 preset-commonmark 的 schema）。 */
const CODE_BLOCK = 'code_block';

/**
 * 把每个 `code_block` 的内容按 highlight.js 的结果套上 inline 装饰。
 *
 * 只处理 code_block：行内代码不套色 —— 密集段落里行内代码套色会很吵，而且 Typora 也不做。
 *
 * 装饰是**纯视觉**的，不进入文档内容，因此不影响 markdownUpdated 的序列化结果。
 * 高亮只在「导出的 HTML」里才需要成为真实结构，那一步走的是另一条路（highlightCode → <span>）。
 */
function buildDecorations(doc: ProseNode): DecorationSet {
  const decorations: Decoration[] = [];

  doc.descendants((node, pos) => {
    if (node.type.name !== CODE_BLOCK) return;
    const text = node.textContent;
    if (text.trim() === '') return;

    const raw = node.attrs['language'];
    const language = typeof raw === 'string' && raw !== '' ? raw : undefined;
    const result = highlightCode(text, language);
    if (!result) return;

    // +1 跳过 code_block 自身的位置标记，进到它内部文本的坐标空间
    const base = pos + 1;
    for (const token of result.tokens) {
      // inclusiveEnd 保持默认的 false：装饰到 token 末尾即止。
      // 这里不用管「会不会吃掉换行」—— 装饰不参与文档内容，改不到换行。
      decorations.push(Decoration.inline(base + token.from, base + token.to, { class: token.className }));
    }
  });

  return DecorationSet.create(doc, decorations);
}

export const codeHighlightPlugin = $prose(
  () =>
    new Plugin<DecorationSet>({
      key: highlightKey,
      state: {
        init: (_config, state) => buildDecorations(state.doc),
        apply: (tr, value) => (tr.docChanged ? buildDecorations(tr.doc) : value.map(tr.mapping, tr.doc)),
      },
      props: {
        decorations: (state) => highlightKey.getState(state),
      },
    }),
);

/** 从 ProseMirror 文档位置换算出行列号（1 基）。 */
function positionOf(doc: ProseNode, pos: number): CursorPosition {
  const safe = Math.max(0, Math.min(pos, doc.content.size));
  const before = doc.textBetween(0, safe, '\n', '\n');
  const lines = before.split('\n');
  return { line: lines.length, column: (lines[lines.length - 1]?.length ?? 0) + 1 };
}

/**
 * 光标位置上报。
 *
 * 用「取回调的函数」而不是直接传回调：插件实例在编辑器创建时就固定了，
 * 而 React 那边每次渲染都会换一个新的回调函数。中间隔一层取值，插件才不会拿着过期的闭包。
 */
export function cursorReporterPlugin(getReporter: () => ((position: CursorPosition) => void) | undefined) {
  return $prose(
    () =>
      new Plugin({
        key: cursorKey,
        view: (editorView) => {
          const report = (view: typeof editorView) => {
            getReporter()?.(positionOf(view.state.doc, view.state.selection.head));
          };
          report(editorView);
          return {
            // 第二个参数是**上一个 state**，不是上一个 view —— 这一点容易写错。
            // 只有选择真正变了才上报，避免每次输入都触发一次无关的状态更新。
            update: (view, prevState) => {
              if (view.state.selection !== prevState.selection) report(view);
            },
          };
        },
      }),
  );
}

// ---------------------------------------------------------------------------
// 变更上报
// ---------------------------------------------------------------------------

/**
 * 标记「这次文档变更是程序写入的，别报给外壳」。
 * 由 `setup.ts` 的 `setMarkdown` 打在事务上。
 */
export const SILENT_META = 'qingmo-silent';

/**
 * 标记「这次变更是程序写入的，但**是用户发起的**，要报给外壳」。
 *
 * 为什么需要第三个状态：`SILENT_META` 与「用户输入」之外还有一类事务 ——
 * **由用户动作触发、但不是键盘输入产生的编辑**。最典型的是查找替换：
 * 用户点「全部替换」，内容确实变了、文档确实该变脏、该能保存，
 * 但它不经过任何 `input` / `paste` 事件，`armedByUser` 永远是 false。
 *
 * 少了这个标记的后果不是「少报一次变更」，而是**静默丢数据**：
 * 外壳的 `doc.text` 停在替换前的内容上，用户按 Ctrl+S 保存，
 * 写回磁盘的是替换前的旧文本 —— 用户看到「已保存」，
 * 而他刚做的替换在磁盘上根本不存在。
 */
export const USER_EDIT_META = 'qingmo-user-edit';

/**
 * 把文档变更序列化成 markdown，上报给外壳。
 *
 * ## 为什么不用 `@milkdown/plugin-listener`
 *
 * 它的 `markdownUpdated` 是**防抖 200ms** 才触发的（`debounce(..., 200)`）。
 * 防抖恰好让「这次变更是程序写入的」这个判断在最需要它的时刻失效：
 * 我们在 dispatch 时设的标志，等到 200ms 后回调才去读，早就清掉了。
 * 护不住的结果是 —— 从磁盘重新加载后，内核把自己序列化（规范化）过的 markdown
 * 再报一次，外壳以为用户改了内容，于是一个字都没动的文档立刻显示「有未保存的改动」。
 * 用户被这个假信号引着去点保存，保存就把文件里他没碰过的排版改掉了 ——
 * 实测会被改的有：setext 标题（`标题\n====`）改成 ATX（`# 标题`）、
 * 行尾空白与连续空行被收敛、`~~~` 围栏改成 ```、`___` 分隔线改成 `***`、
 * 硬换行的两个行尾空格改成反斜杠。都是语义等价的改写，但**字节确实变了**。
 *
 * ## 判定规则：只有**用户交互之后**的变更才算用户改动
 *
 * 光看 `tr.docChanged` 不够 —— `@milkdown/plugin-trailing` 会在文档以
 * 代码块 / 列表 / 表格 / 分隔线结尾时，用 `appendTransaction` 补一个空段落。
 * 那也是一次 docChanged，但用户什么都没做。若把它当成用户改动，
 * 「打开一个以代码块结尾的文件」就会被立刻判成脏。
 *
 * 所以这里监听 DOM 层的输入事件，只在「刚发生过用户输入」的窗口内上报：
 * 打字、输入法上屏、粘贴、剪切、拖放、以及撤销/重做快捷键。
 * 其余一切（初始化、`setMarkdown`、trailing 补段落、任何插件自己的事务）都不上报。
 *
 * 这个规则还有个附带好处：上报是**同步**的（`state.apply` 在 `dispatch` 里跑），
 * 外壳拿到内容的时间和用户按键是同一个事件循环，不会出现「输入完 200ms 才变脏」的迟滞。
 */
export function markdownEmitPlugin(
  report: (markdown: string) => void,
  serialize: (doc: ProseNode) => string,
) {
  /** 用户交互已经发生，下一次文档变更可以上报。 */
  let armedByUser = false;

  const arm = () => {
    armedByUser = true;
  };

  return new Plugin({
    key: emitKey,
    props: {
      // 全部返回 false = 只是旁听，不拦截事件，ProseMirror 自己的处理照常进行。
      handleDOMEvents: {
        input: () => {
          arm();
          return false;
        },
        // 输入法（中文等）在组合期间不发 input，以上屏事件为准
        compositionend: () => {
          arm();
          return false;
        },
        paste: () => {
          arm();
          return false;
        },
        cut: () => {
          arm();
          return false;
        },
        drop: () => {
          arm();
          return false;
        },
      },
      handleKeyDown: (_view, event) => {
        // 撤销/重做走快捷键命令，不产生 input 事件，需要单独认出来
        const mod = event.ctrlKey || event.metaKey;
        if (mod && ['z', 'Z', 'y', 'Y'].includes(event.key)) arm();
        return false;
      },
    },
    state: {
      // 初始文档永远不上报：这也是「打开文件后一个字没改就不该是脏的」的保证
      init: () => false,
      apply: (tr, _value, _oldState, newState) => {
        if (!tr.docChanged) return false;
        const silent = tr.getMeta(SILENT_META) === true;
        const byUser = (armedByUser || tr.getMeta(USER_EDIT_META) === true) && !silent;
        // 无论报不报，这次机会都要用掉：一条事务对应一次交互
        armedByUser = false;
        if (byUser) report(serialize(newState.doc));
        return byUser;
      },
    },
  });
}

// ---------------------------------------------------------------------------
// 占位提示
// ---------------------------------------------------------------------------

/**
 * 空文档时显示占位文案。
 *
 * ProseMirror 的空文档并不是「没有节点」，而是「一个空的 paragraph」——
 * 所以 CSS 的 `:empty` 永远不成立，必须在插件里把这个状态标出来。
 * 标在 contenteditable 自己身上（`is-empty` + `data-placeholder`），
 * 由 `markdown.css` 画出来。挂在宿主元素上而不是插入伪节点，是为了不污染文档内容。
 */
export function placeholderPlugin(text: string) {
  const isEmpty = (doc: ProseNode): boolean =>
    doc.childCount === 1 && doc.firstChild?.type.name === 'paragraph' && doc.firstChild.content.size === 0;

  return new Plugin({
    key: placeholderKey,
    props: {
      attributes: (state) => ({
        'data-placeholder': text,
        // 类名只在空文档时加，避免 CSS 用 :not() 反复判断
        class: isEmpty(state.doc) ? 'is-empty' : '',
      }),
    },
  });
}
