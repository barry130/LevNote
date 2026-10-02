/**
 * Milkdown 实例的组装与命令面。
 *
 * 这里**不使用 `@milkdown/react`**：React 封装层的 API 在主版本之间变动较多，
 * 而核心 API（`Editor.make()...create()`）稳定得多。自己用 `useEffect` 管生命周期，
 * 少一层需要跟随上游升级的抽象。
 *
 * 主题也不使用 `@milkdown/theme-nord` —— 规范要求「导出的 HTML 在别的机器上观感一致」，
 * 主题必须是我们自己完全掌控的一份 CSS，编辑器与导出共用
 * （见 `core/export/document-styles.css` 与 `core/export/document.ts`）。
 *
 * 同理**不使用 `@milkdown/plugin-listener`**：它的 `markdownUpdated` 是防抖 200ms 才触发的，
 * 防抖会让「这次变更是程序写入的」判断失效，后果是「重新加载后一个字没改就显示成脏」。
 * 变更上报改为自研（见 `plugin.ts` 的 `markdownEmitPlugin`）。
 */

import {
  Editor,
  commandsCtx,
  defaultValueCtx,
  editorViewCtx,
  editorViewOptionsCtx,
  parserCtx,
  remarkPluginsCtx,
  remarkStringifyOptionsCtx,
  rootCtx,
  serializerCtx,
} from '@milkdown/kit/core';
import type { CmdKey } from '@milkdown/kit/core';
import { clipboard } from '@milkdown/kit/plugin/clipboard';
import { cursor } from '@milkdown/kit/plugin/cursor';
import { history } from '@milkdown/kit/plugin/history';
import { trailing } from '@milkdown/kit/plugin/trailing';
import { lift } from '@milkdown/kit/prose/commands';
import type { Node as ProseNode } from '@milkdown/kit/prose/model';
import { DOMSerializer } from '@milkdown/kit/prose/model';
import type { Command, Transaction } from '@milkdown/kit/prose/state';
import { TextSelection } from '@milkdown/kit/prose/state';
import { deleteColumn, deleteTable, findTable, isInTable, selectedRect, TableMap, tableNodeTypes } from '@milkdown/kit/prose/tables';
import type { EditorView } from '@milkdown/kit/prose/view';
import {
  createCodeBlockCommand,
  insertHrCommand,
  liftListItemCommand,
  toggleEmphasisCommand,
  toggleInlineCodeCommand,
  toggleLinkCommand,
  toggleStrongCommand,
  turnIntoTextCommand,
  updateLinkCommand,
  wrapInBlockquoteCommand,
  wrapInBulletListCommand,
  wrapInHeadingCommand,
  wrapInOrderedListCommand,
} from '@milkdown/kit/preset/commonmark';
import { commonmark } from '@milkdown/kit/preset/commonmark';
import {
  addColAfterCommand,
  addColBeforeCommand,
  addRowAfterCommand,
  addRowBeforeCommand,
  exitTable,
  gfm,
  insertTableCommand,
  toggleStrikethroughCommand,
} from '@milkdown/kit/preset/gfm';
import { $prose } from '@milkdown/kit/utils';
import type { Options as RemarkStringifyOptions } from 'remark-stringify';

import type {
  BlockFormat,
  CodeBlockState,
  CursorPosition,
  DiagramView,
  EditorCodeBlock,
  EditorFormatting,
  EditorPosition,
  EditorSearch,
  EditorTable,
  ImageInfo,
  ImagePasteRequest,
  SelectionSnapshot,
  SlashState,
  TableAlign,
  TableState,
} from '../../core/editor/types.js';
import { createFindPattern, findWithPattern, type FindOptions } from '../../core/search/find.js';
import { MERMAID_LANGUAGE, MERMAID_STARTER } from '../../core/mermaid.js';
import { CODE_BLOCK, codeBlockChromePlugin, type CodeBlockChromeHandlers } from './code-block-view.js';
import { IMAGE, imageViewPlugin } from './image-view.js';
import {
  mathBlockInputRule,
  mathBlockSchema,
  mathInlineInputRule,
  mathInlineSchema,
  mathRemarkPlugin,
} from './math.js';
import { mathViewPlugin } from './math-view.js';
import { normalizeMarkdownAst } from './normalize-ast.js';
import { applyScrollRatio, readScrollRatio } from '../../core/editor/position.js';
import {
  codeHighlightPlugin,
  cursorReporterPlugin,
  markdownEmitPlugin,
  placeholderPlugin,
  positionReporterPlugin,
  selectionReporterPlugin,
  SILENT_META,
  slashReporterPlugin,
  tableReporterPlugin,
  USER_EDIT_META,
} from './plugin.js';
import { activeLinkHref, describeSlash, resolveBlock } from './selection.js';

/**
 * 编辑器与导出共用的版式类名。
 * 编辑器里由 `editorViewOptionsCtx.attributes` 挂到 contenteditable 上，
 * 导出时由 `core/export/document.ts` 挂到导出容器上 —— 同一套选择器，两处生效。
 */
export const DOC_CLASS = 'qm-doc';

/**
 * 创建内核所需的输入。
 *
 * **回调用属性签名**，理由与 `EditorMountProps` 相同（见 `core/editor/types.ts`）：
 * 这些函数会被**取出来当值传**（例如下面 `cursorReporterPlugin(() => options.onCursorChange)`），
 * 方法简写声明的成员被认为「有 `this`」，一被取出就触发 `unbound-method`。
 *
 * 注意区分：**返回实例上的方法**（下面的 `MarkdownEditorInstance`）保持方法简写 ——
 * 它们只会以 `instance.foo()` 的形式调用，不会被拆下来单传。
 */
export interface CreateMarkdownEditorOptions {
  root: HTMLElement;
  defaultValue: string;
  readOnly?: boolean;
  placeholder?: string;
  /** 内容变化。**由 `setMarkdown` 触发的变化不会回调到这里**（避免回声）。 */
  onChange: (markdown: string) => void;
  onCursorChange?: (position: CursorPosition) => void;
  /** 光标或滚动位置变化（位置记忆）。高频，调用方别在里面做重活。 */
  onPositionChange?: (position: EditorPosition) => void;
  /** 选区变化。参数为 null 表示「现在不该有浮层工具条」（见契约注释）。 */
  onSelectionChange?: (selection: SelectionSnapshot | null) => void;
  /** 斜杠菜单状态。参数为 null 表示「现在不该有斜杠菜单」。 */
  onSlashChange?: (state: SlashState | null) => void;
  /** 表格状态。参数为 null 表示「光标不在表格里，不该有表格工具条」。 */
  onTableChange?: (state: TableState | null) => void;
  /** 语言选择器状态。参数为 null 表示「面板没开着」。 */
  onCodeBlockChange?: (state: CodeBlockState | null) => void;
  /**
   * 把文档里的图片 `src` 解析成能直接放进 `<img>` 的 URL。
   * 返回 null 表示「这张图拿不到」，内核画一个带路径的占位。
   */
  resolveImageUrl?: (src: string) => Promise<string | null>;
  /**
   * 粘贴了一张图片：交出字节，换回**写进文档的相对路径**。
   * 返回 null 表示放弃（格式不认识 / 文档没保存过 / 落盘失败）。
   */
  onImagePaste?: (request: ImagePasteRequest) => Promise<string | null>;
  /** 用户点了文档里的一张图片（灯箱）。 */
  onImageClick?: (image: ImageInfo) => void;
  /**
   * 用户要放大看一张 Mermaid 图表。
   *
   * 不实现时代码块上**不显示**放大按钮 —— 与「不实现 `onImageClick` 就不开灯箱」
   * 同一条规则：给一个点了没反应的按钮，不如不给。
   */
  onDiagramOpen?: (diagram: DiagramView) => void;
}

export interface MarkdownEditorInstance {
  getMarkdown(): string;
  setMarkdown(markdown: string): void;
  focus(): void;
  /** 当前光标与滚动位置（位置记忆）。 */
  getPosition(): EditorPosition;
  /** 恢复光标与滚动。`textLength` 对不上时只恢复滚动。 */
  setPosition(position: EditorPosition): void;
  /** 语义 HTML 片段，不含 `<html>` / `<head>`。 */
  toHtml(): string;
  readonly search: EditorSearch;
  readonly formatting: EditorFormatting;
  readonly table: EditorTable;
  readonly codeBlock: EditorCodeBlock;
  /** 大纲跳转：定位到指定层级的指定标题。 */
  goToHeading(level: number, text: string): boolean;
  destroy(): Promise<void>;
}

/**
 * 编辑器内的查找替换。
 *
 * ## 为什么在文档树而不是 Markdown 源码上搜
 *
 * 用户看到的是**渲染结果**。拿 Markdown 源码去搜会在 `**加粗**` 上匹配到
 * 一个用户根本看不见的位置（`**` 是语法标记），「下一个」跳到那里会让人以为跳错了。
 * 在文档树上搜，位置天然就是 ProseMirror 位置，选中与滚动都不需要坐标翻译。
 *
 * ## 代价（必须写清楚）
 *
 * 跨节点的匹配找不到。搜 `hello` 时 `he**llo**` 不会命中，因为 `he` 与 `llo`
 * 是两个文本节点。这是**有意的取舍**：支持跨节点需要把整个文档拍平再建立
 * 「扁平偏移 ↔ 文档位置」的双向映射，而那个映射要正确复现 ProseMirror
 * `textBetween` 的分隔符规则（块间 `\n`、硬换行 `\n`、图片无文本）——
 * 一旦不一致，替换就会写到错误的位置上，那是比「偶尔少找到一个」严重得多的错误。
 * **宁可少找，不可错改。**
 */
function createMarkdownSearch(editor: Editor): EditorSearch {
  interface PMMatch {
    from: number;
    to: number;
    text: string;
  }

  let matches: PMMatch[] = [];
  let options: FindOptions | null = null;
  let pattern: RegExp | null = null;

  const getView = () => editor.action((ctx) => ctx.get(editorViewCtx));

  const refind = (): number => {
    matches = [];
    if (options === null) return 0;
    pattern = createFindPattern(options);
    if (pattern === null) return 0;

    const view = getView();
    view.state.doc.descendants((node, pos) => {
      if (!node.isText) return true;
      const text = node.text ?? '';
      const current = pattern;
      if (current === null) return false;
      for (const hit of findWithPattern(text, current)) {
        matches.push({ from: pos + hit.from, to: pos + hit.to, text: hit.text });
      }
      return true;
    });

    return matches.length;
  };

  return {
    find: (next) => {
      options = next;
      return refind();
    },

    reveal: (index) => {
      const hit = matches[index];
      if (!hit) return;
      const view = getView();
      view.dispatch(
        view.state.tr.setSelection(TextSelection.create(view.state.doc, hit.from, hit.to)).scrollIntoView(),
      );
      view.focus();
    },

    replace: (index, replacement) => {
      const hit = matches[index];
      if (!hit || options === null) return matches.length;
      const insert = expand(hit.text, pattern, replacement, options.regex);
      const view = getView();
      view.dispatch(view.state.tr.insertText(insert, hit.from, hit.to).setMeta(USER_EDIT_META, true));
      return refind();
    },

    replaceAll: (replacement) => {
      if (options === null || matches.length === 0) return 0;
      const hits = [...matches];
      const asRegex = options.regex;
      const view = getView();
      let tr = view.state.tr;
      // 从后往前替换：前面的替换不会让后面的位置失效
      for (let i = hits.length - 1; i >= 0; i -= 1) {
        const hit = hits[i];
        if (!hit) continue;
        tr = tr.insertText(expand(hit.text, pattern, replacement, asRegex), hit.from, hit.to);
      }
      // 这条 meta 是必需的：没有它，替换结果不会报给外壳，
      // 用户按保存时写回磁盘的会是替换前的旧内容（详见 plugin.ts 的 USER_EDIT_META）
      view.dispatch(tr.setMeta(USER_EDIT_META, true));
      matches = [];
      return hits.length;
    },

    clear: () => {
      matches = [];
      pattern = null;
      options = null;
    },
  };
}

/** 正则模式下展开单条替换里的 `$1` / `$&`；字面模式下 `$` 是普通字符。 */
function expand(text: string, pattern: RegExp | null, replacement: string, asRegex: boolean): string {
  if (!asRegex || pattern === null) return replacement;
  return text.replace(new RegExp(pattern.source, pattern.flags.replace('g', '')), replacement);
}

/**
 * 把一次编辑标记成「用户发起的」。
 *
 * ## 少了这一步会**静默丢数据**
 *
 * `markdownEmitPlugin` 只在「刚发生过用户输入」的窗口内上报变更，而点工具条上的
 * 按钮**不产生任何 `input` / `paste` 事件** —— 没有这个标记，加粗之后文档不会变脏，
 * 用户按 Ctrl+S，写回磁盘的是加粗前的旧内容，而界面一直显示「已保存」。
 *
 * 这与 `USER_EDIT_META` 当初为「查找替换」加上的理由是**同一条**：
 * 由用户动作触发、但不经过键盘事件的编辑，必须自己声明身份。
 * 已经踩过三次 —— 查找替换、浮层格式化、斜杠菜单插入；表格操作是第四次。
 *
 * ## 为什么是临时换掉 `view.dispatch`
 *
 * 命令由 Milkdown 的 `CommandManager` 内部派发（`command(view.state, view.dispatch, view)`），
 * 中间没有任何可以插进去的钩子，也没有 `dispatchTransaction` 选项被用到。
 * 换掉再在 `finally` 里换回来，作用范围只有这一次同步调用 ——
 * 不是全局补丁，也不会漏到别的派发路径上。`dispatch` 是实例上的可写属性
 * （`prosemirror-view` 里声明为 `dispatch: (tr) => void`），所以这样做是合法的。
 *
 * 提到模块级是因为**表格操作也需要它**。做成两个各自独立的实现，
 * 迟早只给其中一处修 bug —— 而这类 bug 的形态是「数据静默丢失」。
 */
function withUserEdit<T>(editor: Editor, run: () => T): T {
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
function withUserEditOnView<T>(view: EditorView, run: () => T): T {
  const original = view.dispatch;
  view.dispatch = (tr) => original.call(view, tr.setMeta(USER_EDIT_META, true));
  try {
    return run();
  } finally {
    view.dispatch = original;
  }
}

/**
 * 浮层工具条要用的格式化命令面。
 *
 * ## 为什么几乎全部走 Milkdown 命令，而不是直接用 ProseMirror
 *
 * 命令注册在 `commandsCtx` 里，是 Milkdown 与它各预设之间的**唯一接口**。
 * 自己写 `setBlockType(schema.nodes.heading, {level})` 看着更直接，但会把
 * 预设内部的取舍（标题 id 同步、列表序号同步、块引用容器类型）绕过去 ——
 * 那些是 `withMeta` 之外的插件在 `appendTransaction` 里做的，绕过它们不会报错，
 * 只会在某个具体文档上悄悄少做一件事。
 *
 * 只有「从引用里脱身」是个例外：预设没给这个命令（它只有 `wrapInBlockquote`），
 * 所以退回 prosemirror-commands 的 `lift`。这是唯一一处。
 */
function createMarkdownFormatting(editor: Editor): EditorFormatting {
  /** 本文件内的短名字。实现见模块级的 `withUserEdit`。 */
  const asUserEdit = <T>(run: () => T): T => withUserEdit(editor, run);

  /** 调一条 Milkdown 命令。返回它是否真的执行了（false = 当前选区不适用）。 */
  const call = <T>(key: CmdKey<T>, payload?: T): boolean =>
    editor.action((ctx) => ctx.get(commandsCtx).call(key, payload));

  /** 直接跑一条 ProseMirror 命令。只给 `lift` 用，理由见上。 */
  const runRaw = (command: Command): boolean =>
    editor.action((ctx) => {
      const view = ctx.get(editorViewCtx);
      return command(view.state, view.dispatch, view);
    });

  const currentBlock = (): { block: BlockFormat; headingLevel: number | null } | null =>
    editor.action((ctx) => {
      const view = ctx.get(editorViewCtx);
      // 用 $from 而不是 describeSelection：后者在光标塌缩时返回 null，
      // 而「光标在标题里、没选中文字」也是合法的格式化场景
      return resolveBlock(view.state.selection.$from);
    });

  /**
   * 斜杠触发状态下的 `/query` 在文档里的范围；不在触发状态时返回 null。
   *
   * 这里**故意重新判断一次**（而不是让外壳把范围传进来）：文档坐标只有内核知道，
   * 而且从外壳决定「用户点了哪一项」到内核真的执行，中间隔着一次 React 渲染 ——
   * 那段时间里用户完全可能又敲了一个字。重新判断等于顺手做了一次乐观锁。
   */
  const slashRange = (): { from: number; to: number } | null =>
    editor.action((ctx) => {
      const view = ctx.get(editorViewCtx);
      if (describeSlash(view.state) === null) return null;
      const { $from } = view.state.selection;
      return { from: $from.start(), to: $from.pos };
    });

  /**
   * 在光标处放一个**空的块级公式**，然后让 NodeView 自己接管 ——
   * 空公式的 NodeView 会直接进入编辑态，用户看到的就是一个等着打字的输入框。
   *
   * ## 为什么不用 `setBlockType`
   *
   * `setBlockType` 换完之后选区会被映射到新节点**内部**，而公式块是原子、
   * 没有内容 —— 那个位置在 schema 里不成立。这里改成「替换 + 显式指定
   * 新选区」，位置是自己算的，不依赖映射的结果。
   *
   * ## 段落里有文字时**插在后面**，绝不替换
   *
   * 无条件 `replaceWith(段落前, 段落后, …)` 会把整段文字**静默吃掉**：
   * 用户在 `前文` 后面敲 `/公式` 选菜单，结果是 `前文` 不见了，而且没有任何
   * 报错。斜杠菜单的既有约定是「不在触发状态时一个字符都不删」
   * （见 `slash.test.ts`），其它块（表格、代码块、分隔线）都遵守它 ——
   * 公式块没有理由例外。
   *
   * 所以分两种情形：
   *
   * - 段落是**空的**（空行上敲 `/公式`、或 `insertBlock` 刚把 `/公式` 删掉）
   *   → 整块换成公式；
   * - 段落里有文字 → 公式插在段落**之后**，一个字都不动。
   *
   * 新选区取 `near(..., 1)`（向前偏）：公式块后面没有内容时它会落到
   * 下一个可放光标的位置上，而 `@milkdown/plugin-trailing` 会在文档以
   * 公式块结尾时补一个空段落 —— 两者合起来，光标总是有地方可去。
   */
  const insertMathBlock = (): void => {
    editor.action((ctx) => {
      const view = ctx.get(editorViewCtx);
      const { $from } = view.state.selection;
      // 不在文本块里（例如光标落在表格上）就什么都不做。
      // 猜一个位置去放公式，比不响应更糟
      if (!$from.parent.isTextblock) return;

      const node = mathBlockSchema.type(ctx).create();
      const start = $from.before();
      const end = $from.after();
      const empty = $from.parent.content.size === 0;

      const tr = empty
        ? view.state.tr.replaceWith(start, end, node)
        : // 插在段落之后：`replaceWith` 的起止相同就是「插入」
          view.state.tr.replaceWith(end, end, node);
      const after = empty ? start + node.nodeSize : end + node.nodeSize;
      tr.setSelection(TextSelection.near(tr.doc.resolve(Math.min(after, tr.doc.content.size)), 1));
      view.dispatch(tr);
    });
  };

  /**
   * 插入一个 **Mermaid 图表块**。
   *
   * 走的是与 `insertMathBlock` 同一条路径（空段落 → 整块替换；有文字 → 插在段后），
   * 但有三处不同，每一处都是被「图表不是公式」逼出来的：
   *
   * - **带起始内容**（`MERMAID_STARTER`）。空块会停在源码态（见
   *   `code-block-view.ts` 的 `wantDiagram`），用户只看到一个空框；
   * - **光标落在块里面**而不是块后面。图表块是要他**改那段源码**的，
   *   落到块外面等于让他先点一下才写得进去；
   * - **语言属性必须在建节点时就给**（`{ language: 'mermaid' }`）。
   *   先建一个无语言的代码块再 `setNodeAttribute` 也可以，但那是两次事务，
   *   中间那一帧内核已经把它当成「纯文本块」了。
   */
  const insertDiagramBlock = (): void => {
    editor.action((ctx) => {
      const view = ctx.get(editorViewCtx);
      const { $from } = view.state.selection;
      if (!$from.parent.isTextblock) return;

      const type = view.state.schema.nodes[CODE_BLOCK];
      if (type === undefined) return;

      const node = type.create({ language: MERMAID_LANGUAGE }, view.state.schema.text(MERMAID_STARTER));
      const start = $from.before();
      const end = $from.after();
      const empty = $from.parent.content.size === 0;

      const tr = empty
        ? view.state.tr.replaceWith(start, end, node)
        : view.state.tr.replaceWith(end, end, node);
      // 块内**内容末尾**：`+1` 跨过 code_block 自己的开标签，再走完内容长度
      const base = empty ? start : end;
      const at = Math.min(base + 1 + node.content.size, tr.doc.content.size);
      tr.setSelection(TextSelection.create(tr.doc, at));
      view.dispatch(tr);
    });
  };

  const applyBlock = (format: BlockFormat, level?: number): void => {
    const now = currentBlock();
    if (now === null) return;
    switch (format) {
      case 'paragraph': {
        // 不能只调 setBlockType(paragraph)：在列表 / 引用里，文本块**本来就已经是**
        // 段落，命令会「成功但什么都没改」，用户看到的是点了没反应。
        // 所以先把自己从容器里提出来。
        if (now.block === 'bulletList' || now.block === 'orderedList') {
          call(liftListItemCommand.key);
          return;
        }
        if (now.block === 'blockquote') {
          runRaw(lift);
          return;
        }
        call(turnIntoTextCommand.key);
        return;
      }

      case 'heading': {
        // 已经是这一级 → 取消（回到正文）。原地不动会让按钮看起来点坏了
        const wanted = level ?? 1;
        if (now.block === 'heading' && now.headingLevel === wanted) {
          call(turnIntoTextCommand.key);
          return;
        }
        call(wrapInHeadingCommand.key, wanted);
        return;
      }

      case 'blockquote':
        if (now.block === 'blockquote') {
          runRaw(lift);
          return;
        }
        call(wrapInBlockquoteCommand.key);
        return;

      case 'bulletList':
        if (now.block === 'bulletList') {
          call(liftListItemCommand.key);
          return;
        }
        call(wrapInBulletListCommand.key);
        return;

      case 'orderedList':
        if (now.block === 'orderedList') {
          call(liftListItemCommand.key);
          return;
        }
        call(wrapInOrderedListCommand.key);
        return;

      case 'codeBlock':
        // 代码块是终态，没有「取消」这一说 —— 退出的方式是再按一次「正文」
        if (now.block === 'codeBlock') return;
        call(createCodeBlockCommand.key);
        return;
    }
  };

  return {
    toggleInline: (format, value) => {
      asUserEdit(() => {
        switch (format) {
          case 'strong':
            call(toggleStrongCommand.key);
            return;
          case 'emphasis':
            call(toggleEmphasisCommand.key);
            return;
          case 'strikethrough':
            call(toggleStrikethroughCommand.key);
            return;
          case 'inlineCode':
            call(toggleInlineCodeCommand.key);
            return;

          case 'link': {
            const href = value?.trim() ?? '';
            const existing = editor.action((ctx) => activeLinkHref(ctx.get(editorViewCtx).state));

            // 没给地址：只处理「去掉」。不在链接里就什么都不做 —— 内核不替用户猜地址
            if (href === '') {
              if (existing !== null) call(toggleLinkCommand.key);
              return;
            }
            // 已经在链接里就改地址（`updateLinkCommand` 会保留原有的 title），
            // 否则直接加上。用 toggleMark 去改地址是不行的：它看到「已有该 mark」
            // 会执行**移除**，用户的链接会莫名其妙消失。
            if (existing !== null) call(updateLinkCommand.key, { href });
            else call(toggleLinkCommand.key, { href });
            return;
          }
        }
      });
    },

    toggleBlock: (format, level) => {
      asUserEdit(() => {
        applyBlock(format, level);
      });
    },

    insertBlock: (block) => {
      asUserEdit(() => {
        /**
         * 先删掉 `/query`，再套块格式。
         *
         * 两步是两次 dispatch，但**不必担心变成两步撤销**：`prosemirror-history`
         * 默认把 500ms 内的连续事务并成一个撤销组，这里两次之间隔着不到一帧。
         * 如果改成「自己拼一条事务」，就得复刻 Milkdown 各命令内部的事务构造 ——
         * 那正是「用命令而不是自己写 ProseMirror」要避免的事。
         */
        const range = slashRange();
        if (range !== null && range.to > range.from) {
          editor.action((ctx) => {
            const view = ctx.get(editorViewCtx);
            view.dispatch(view.state.tr.delete(range.from, range.to));
          });
        }

        switch (block) {
          case 'paragraph':
            applyBlock('paragraph');
            return;
          case 'heading1':
            applyBlock('heading', 1);
            return;
          case 'heading2':
            applyBlock('heading', 2);
            return;
          case 'heading3':
            applyBlock('heading', 3);
            return;
          case 'blockquote':
            applyBlock('blockquote');
            return;
          case 'bulletList':
            applyBlock('bulletList');
            return;
          case 'orderedList':
            applyBlock('orderedList');
            return;
          case 'codeBlock':
            applyBlock('codeBlock');
            return;
          case 'mathBlock':
            insertMathBlock();
            return;
          case 'diagram':
            insertDiagramBlock();
            return;
          case 'table':
            // 3 × 3（含表头行）。菜单项是「表格」而不是「3×3 表格」——
            // 尺寸是最容易改的东西（加行加列就在工具条上），不值得为它多开三个菜单项
            call(insertTableCommand.key, { row: 3, col: 3 });
            return;
          case 'hr':
            call(insertHrCommand.key);
            return;
        }
      });
    },
  };
}

/**
 * 表格命令面。
 *
 * ## 为什么与 `createMarkdownFormatting` 分成两个函数
 *
 * 表格能力**只有支持表格的内核才有**（`EditorHandle.table` 是可选的），
 * 而格式化能力是所有富文本内核都该有的。混在一起会让「这个内核支持表格吗」
 * 变成一个需要逐方法探测的问题。
 *
 * ## 加行 / 加列走 Milkdown 命令，删行自己重建，删列走 prosemirror-tables
 *
 * 三条路各走各的，每一条都是被实测逼出来的：
 *
 * - **加行 / 加列**：预设的 `AddRowAfter` 等不是直接转调 prosemirror-tables 的同名函数，
 *   它多做了 `addRowWithAlignment` —— **把相邻单元格的对齐方式抄到新行 / 新列上**。
 *   自己调底层函数不会报错，只会在「整列居中」的表格上悄悄多出一列左对齐的格子。
 * - **删行**：不能用 `deleteRow`。Milkdown 的 gfm 表格 schema 是
 *   `table_header_row table_row+`（见下），`deleteRow` 不知道这条约束，
 *   删掉表头行之后留下的表格 schema 仍然合法（表头行的 content 是 `(table_header)*`），
 *   但**序列化时那一行会被静默跳过**。实测：文档里 4 行、存下来只剩 2 行，
 *   而且用户完全看不出少了哪两行。所以这里改成**重建表格节点**，见 `deleteTableRow`。
 * - **删列**：`deleteColumn` 会按 `TableMap` 正确地处理 `colspan` 收缩，
 *   而重建做不到这一点（`colspan` 是跨列的，删一列要把它从 2 改成 1 而不是删掉格子）。
 *   行结构不会被它破坏 —— 每一行至少还剩一格，表头行也还在。
 *
 * ## 这条 schema 约束到底约束了什么
 *
 * `table_header_row table_row+` 的意思是：**一张表必须有表头行，且至少有一行正文**。
 * 于是：
 * - 表格最少 2 行 —— 「只剩 1 行」不是一种可表示的状态，删到 2 行就该停手；
 * - 表头行不能凭空消失 —— 删掉它就得让下一行顶上（`asHeaderRow`）。
 *
 * 另外 `tableSchema.toMarkdown` 是用**第一行**的单元格决定分隔行的对齐的
 * （`align.push(cell.attrs.alignment)`），所以「对齐」在 Markdown 里本质是**按列**的，
 * 而且以表头行为准 —— 这也是 `setAlign` 必须整列一起改的原因。
 */
function createMarkdownTable(editor: Editor): EditorTable {
  const call = <T>(key: CmdKey<T>, payload?: T): boolean =>
    editor.action((ctx) => ctx.get(commandsCtx).call(key, payload));

  /** 直接跑一条 ProseMirror 命令。只给删列 / 删表用，理由见上。 */
  const runRaw = (command: Command): boolean =>
    editor.action((ctx) => {
      const view = ctx.get(editorViewCtx);
      return command(view.state, view.dispatch, view);
    });

  return {
    isActive: () => editor.action((ctx) => isInTable(ctx.get(editorViewCtx).state)),

    run: (operation) => {
      const snap = snapshotTable(editor);
      if (snap === null) return;

      switch (operation) {
        case 'rowBefore':
          withUserEdit(editor, () => call(addRowBeforeCommand.key));
          return;
        case 'rowAfter':
          withUserEdit(editor, () => call(addRowAfterCommand.key));
          return;
        case 'colBefore':
        case 'colAfter': {
          const insertBefore = operation === 'colBefore';
          // 参考列的对齐要在**插入之前**记下来：插入之后那一列的位置就变了
          const align = headerAlignmentAt(snap);
          withUserEdit(editor, () => {
            call(insertBefore ? addColBeforeCommand.key : addColAfterCommand.key);
            // 预设只给**加行**抄对齐（`addRowWithAlignment`），加列没有这一步 ——
            // 不补上的话，一张整列居中的表格里新加的列会是左对齐的。
            // 行与列在这里应该一致，否则「居中的表格加一列」会得到一个斑驳的结果
            editor.action((ctx) => {
              const view = ctx.get(editorViewCtx);
              const after = snapshotFrom(view);
              if (after === null) return;
              writeColumnAlign(view, after, insertBefore ? snap.col : snap.col + 1, align);
            });
          });
          return;
        }
        case 'deleteRow':
          // 删完只剩 1 行 —— 那张表没法表示，按钮那边也已经是禁用的
          if (snap.rows < 3) return;
          deleteTableRow(editor, snap);
          return;
        case 'deleteColumn':
          if (snap.columns < 2) return;
          deleteTableColumn(editor, snap);
          return;
        case 'deleteTable':
          withUserEdit(editor, () => runRaw(deleteTable));
          return;
        case 'exit':
          withUserEdit(editor, () => call(exitTable.key));
          return;
      }
    },

    setAlign: (align: TableAlign) => {
      withUserEdit(editor, () => {
        editor.action((ctx) => {
          const view = ctx.get(editorViewCtx);
          const snap = snapshotFrom(view);
          if (snap === null) return;
          writeColumnAlign(view, snap, snap.col, align);
        });
      });
    },
  };
}

/** 光标所在表格的一次性快照。不在表格里（或表格结构损坏）时返回 null。 */
interface TableSnapshot {
  readonly table: ProseNode;
  /** 表格节点**之前**的位置 —— `replaceWith` 的区间起点。 */
  readonly from: number;
  /** 光标所在的行 / 列（0 基，表头行是第 0 行）。 */
  readonly row: number;
  readonly col: number;
  readonly rows: number;
  readonly columns: number;
}

/**
 * 取一次快照。
 *
 * 每个操作**开头取一次**、之后全部基于它算 —— 不在循环里反复读 `view.state`：
 * 事务是攒着一次性派发的，中间态不存在于任何地方，反复读只会读到同一个旧值。
 */
function snapshotTable(editor: Editor): TableSnapshot | null {
  return editor.action((ctx) => snapshotFrom(ctx.get(editorViewCtx)));
}

/** 快照的纯函数形态：同一个操作里第二次取快照时用得上（`view` 已经换了）。 */
function snapshotFrom(view: EditorView): TableSnapshot | null {
  const { state } = view;
  const found = findTable(state.selection.$from);
  if (found === null) return null;

  let rect;
  try {
    rect = selectedRect(state);
  } catch {
    // 表格结构损坏（单元格重叠 / 非矩形）时 `selectedRect` 会抛 RangeError。
    // 如实返回 null 而不是让一次按钮点击把编辑器带崩
    return null;
  }

  return {
    table: found.node,
    from: found.pos,
    row: rect.top,
    col: rect.left,
    rows: rect.map.height,
    columns: rect.map.width,
  };
}

/**
 * 把某一列**所有**单元格的对齐方式写成 `align`。
 *
 * 为什么是整列而不是光标那一格，见 `setAlign` 的注释。`col` 越界时什么都不做。
 *
 * 位置换算：`TableMap.positionAt` 给的是**相对表格内容起点**的偏移，
 * 而 `snap.from` 是表格**之前**的位置，所以单元格的绝对位置是 `from + 1 + rel`。
 */
function writeColumnAlign(view: EditorView, snap: TableSnapshot, col: number, align: string): void {
  if (col < 0 || col >= snap.columns) return;

  const tr = view.state.tr;
  const map = TableMap.get(snap.table);
  let changed = false;

  for (let row = 0; row < map.height; row += 1) {
    const rel = map.positionAt(row, col, snap.table);
    if (rel < 0) continue;
    const pos = snap.from + 1 + rel;
    const cell = view.state.doc.nodeAt(pos);
    if (cell === null || cell.attrs['alignment'] === align) continue;
    tr.setNodeMarkup(pos, undefined, { ...cell.attrs, alignment: align });
    changed = true;
  }

  if (changed) view.dispatch(tr);
}

/** 表头行里第 `snap.col` 列那个单元格的对齐方式（读不到就按空串）。 */
function headerAlignmentAt(snap: TableSnapshot): string {
  const header = snap.table.child(0);
  let col = 0;
  for (let i = 0; i < header.childCount; i += 1) {
    const cell = header.child(i);
    if (col === snap.col) return String(cell.attrs['alignment'] ?? '');
    col += Number(cell.attrs['colspan'] ?? 1);
  }
  return '';
}

/**
 * 跑一条「只能通过 `dispatch` 交回结果」的命令，把那个事务截下来自己派发。
 *
 * 命令没有派发（返回 false，也就是当前选区不适用）时返回 null。
 */
function takeTransaction(run: (dispatch: (tr: Transaction) => void) => boolean): Transaction | null {
  let captured: Transaction | null = null;
  if (!run((tr) => {
    captured = tr;
  })) {
    return null;
  }
  return captured;
}

/**
 * 删掉表格里的一行 —— **重建表格节点**，而不是调 prosemirror-tables 的 `deleteRow`。
 *
 * 重建一次把两件事做对：
 * - 删掉目标行；
 * - 若删的是表头行，把顶上来的那一行**转成表头行**。
 *
 * 单元格节点是**原样搬过去**的（内容、marks、`colspan` / `rowspan` 都保留），
 * 所以这不是「重新造一张表」，只是拿掉一行、必要时换一个行的类型。
 *
 * ## 表头行的类型为什么从表格自身取，而不是 `tableNodeTypes(...).row`
 *
 * `TableRole` 只有 `table` / `row` / `cell` / `header_cell` 四种 ——
 * **表头行与普通行共用 `row` 这个角色**，所以 `tableNodeTypes(schema).row`
 * 给的是「schema 里最后注册的那个 row」，可能是普通行也可能是表头行。
 * 表格的第 0 行按 schema（`table_header_row table_row+`）一定是表头行，
 * 从它身上取类型是唯一可靠的来源。
 */
function deleteTableRow(editor: Editor, snap: TableSnapshot): void {
  withUserEdit(editor, () => {
    editor.action((ctx) => {
      const view = ctx.get(editorViewCtx);
      const headerRowType = snap.table.child(0).type;
      const headerCellType = tableNodeTypes(view.state.schema).header_cell;

      const rows: ProseNode[] = [];
      snap.table.forEach((row, _offset, index) => {
        if (index === snap.row) return;

        // 删的是表头行 → 顶上来的那一行必须变成表头行。
        // 少了这一步，新的第 0 行会是普通行，序列化时整行被静默跳过
        if (snap.row === 0 && index === 1) {
          const cells: ProseNode[] = [];
          row.forEach((cell) => {
            cells.push(headerCellType.create(cell.attrs, cell.content, cell.marks));
          });
          rows.push(headerRowType.create(row.attrs, cells, row.marks));
          return;
        }

        rows.push(row);
      });

      const next = snap.table.type.create(snap.table.attrs, rows, snap.table.marks);
      const tr = view.state.tr.replaceWith(snap.from, snap.from + snap.table.nodeSize, next);
      keepCaretInTable(tr, next, snap);
      view.dispatch(tr);
    });
  });
}

/**
 * 删掉光标所在的那一列。
 *
 * 这里**用** prosemirror-tables 的 `deleteColumn`，但把它的出口截下来 ——
 * 因为它会把光标映射到**表格之外**（被删掉的正是光标所在的那一格），
 * 工具条会跟着一起消失，用户下一眼看到的就是「点了一下，条没了」。
 *
 * 为什么删列不自己重建：`deleteColumn` 会按 `TableMap` 正确地**收缩 `colspan`**
 * （跨两列的格子要变成跨一列，而不是被删掉）。重建做不到这件事，
 * 而做错的表现是**静默丢内容**。
 */
function deleteTableColumn(editor: Editor, snap: TableSnapshot): void {
  withUserEdit(editor, () => {
    editor.action((ctx) => {
      const view = ctx.get(editorViewCtx);
      const tr = takeTransaction((dispatch) => deleteColumn(view.state, dispatch));
      if (tr === null) return;

      const next = tr.doc.nodeAt(snap.from);
      if (next !== null && next.type === snap.table.type) keepCaretInTable(tr, next, snap);

      view.dispatch(tr);
    });
  });
}

/**
 * 把事务的选区放回表格里 —— 同一行、同一列，越界就夹到新的边界上。
 *
 * 删除操作会把光标映射到表格外面（它所在的那一格没了）。不修的话工具条会消失，
 * 而「删掉一行之后还想再删一行」是最自然的连续操作。
 */
function keepCaretInTable(tr: Transaction, table: ProseNode, snap: TableSnapshot): void {
  const map = TableMap.get(table);
  const row = Math.min(snap.row, map.height - 1);
  const col = Math.min(snap.col, map.width - 1);
  const rel = map.positionAt(row, col, table);
  // `+ 2` = 表格内容起点（+1）再进一格到单元格内容
  if (rel >= 0) tr.setSelection(TextSelection.near(tr.doc.resolve(snap.from + rel + 2), 1));
}

/**
 * 在文档树里找第一个「层级 + 标题文字」都匹配的标题，返回它的位置；找不到返回 `null`。
 *
 * **为什么单独拆成函数**（而不是在调用处就地写 `let found` + 回调里赋值）：
 * 「回调里给外部 `let` 赋值，回到外层再收窄」这个写法会让控制流分析失效。
 * 用 TS 6 的编译器 API 实测过同一个表达式：
 *
 * - 本函数这种「直接 `return found`」的写法，两个编译器都得到 `number | null`；
 * - 原来那种「回调里赋值 → 外层 `if (found === null) return false;` → 再 `found + 1`」
 *   的写法，**TS 6 会把 `found` 收窄成 `null`**（也就是完全没把回调里的那次赋值
 *   算进去），于是排除 `null` 之后变成 `never`；而 TS 7 能正确得到 `number`。
 *
 * 这正好是本项目「lint 用 TS 6、tsc 用 TS 7」双编译器并存（见 ADR 0013）
 * 会暴露出来的一类分歧：**lint 看到的类型不一定等于 tsc 看到的类型**。
 * 与其赌哪个版本的 CFA 更聪明，不如换成不依赖它的写法。
 */
function findHeadingPos(doc: ProseNode, level: number, text: string): number | null {
  let found: number | null = null;

  doc.descendants((node, pos) => {
    if (found !== null) return false;
    if (node.type.name !== 'heading') return true;
    if (node.attrs['level'] !== level) return true;
    if (node.textContent.trim() !== text) return true;
    found = pos;
    return false;
  });

  return found;
}

// ---------------------------------------------------------------------------
// 代码块（语言选择器）
// ---------------------------------------------------------------------------

/**
 * 语言选择器的状态。**两半共享同一个对象**：
 *
 * - `handlers` 交给 NodeView 调 —— 插件必须在编辑器创建**之前**装好；
 * - `commands` 交给外壳调 —— 它要写文档，得等编辑器创建出来。
 *
 * 用一个可变对象把两半连起来，是因为它们的创建时机天然错开。中间那段
 * 「编辑器正在创建」的时间里面板不可能被打开（还没有 DOM 可以点）。
 *
 * `target` 存的是**视图实例**（`token`）与一个**现算位置的函数**，而不是一个坐标数字：
 * 面板开着的时候用户完全可能在别处打字，记下的坐标就指向了别的节点。
 * 「面板还是不是属于同一个代码块」只有实例身份说得准，而「它在哪」得现问。
 */
interface CodeBlockPicker {
  target: { token: unknown; resolvePos: () => number | null } | null;
  readonly notify: (state: CodeBlockState | null) => void;
}

/**
 * 语言标签被点开 / 所在的代码块消失时要做的事。
 *
 * 与命令面分成两个函数，是因为创建时机不同（见 `CodeBlockPicker`）。
 */
function codeBlockPickerHandlers(
  picker: CodeBlockPicker,
  onDiagramOpen: ((diagram: DiagramView) => void) | undefined,
): CodeBlockChromeHandlers {
  return {
    openPicker: (request) => {
      picker.target = { token: request.token, resolvePos: request.resolvePos };
      picker.notify({ anchor: request.anchor, language: request.language });
    },
    /**
     * 放大看图表。
     *
     * 不在这里拼装任何东西：NodeView 交出来的就是「它此刻画在屏幕上的那一份」，
     * 而全屏查看器要的也正是那一份 —— 重新渲染一遍既慢，又可能画出不一样的图。
     */
    openViewer: (diagram) => onDiagramOpen?.(diagram),
    closePicker: (token) => {
      // 只有**面板正对着的那个**代码块消失时才关。删掉别的代码块不该把面板带走 ——
      // 用户可能正打算给手里这个换个语言
      if (picker.target === null || picker.target.token !== token) return;
      picker.target = null;
      picker.notify(null);
    },
  };
}

/**
 * 代码块命令面。
 *
 * ## 为什么 `setLanguage` 要自己拼一条事务，而不是调
 * `updateCodeBlockLanguageCommand`
 *
 * 预设确实给了这条命令（`{ pos, language }`），但它内部就是
 * `tr.setNodeAttribute(pos, 'language', language)`，而且**不做任何校验**。
 * 面板开着的时候用户完全可能在别处打字：位置会漂，甚至那个代码块已经被删掉了。
 * 位置对不上时 `setNodeAttribute` 会把语言写到**另一个**节点上，不报错也不生效 ——
 * 典型的静默错。所以这里自己写，好在写之前核一次「那个位置上还是一个代码块吗」。
 *
 * 位置是**现问的**（`target.resolvePos()`），不是面板打开时记下的那个数字。
 */
function createMarkdownCodeBlock(editor: Editor, picker: CodeBlockPicker): EditorCodeBlock {
  const close = (): void => {
    if (picker.target === null) return;
    picker.target = null;
    picker.notify(null);
  };

  return {
    setLanguage: (language) => {
      const target = picker.target;
      // 先关面板再改文档：面板一关，外壳那次渲染就已经不再指向旧状态了
      close();
      if (target === null) return;

      const pos = target.resolvePos();
      // 那个代码块已经不在了（被删掉、或整篇被替换）。什么都不做 ——
      // 写错地方比不写更糟
      if (pos === null) return;

      withUserEdit(editor, () => {
        editor.action((ctx) => {
          const view = ctx.get(editorViewCtx);
          const node = view.state.doc.nodeAt(pos);
          // 二次确认。`resolvePos` 给的位置按说是准的，但这里是最后一道闸：
          // 万一它因为任何原因对不上，宁可什么都不做
          if (node === null || node.type.name !== CODE_BLOCK) return;
          // 语言没变就什么都不做。`setNodeAttribute` **即使写的是同一个值**
          // 也会造出一个新节点、产生一次 docChanged —— 于是用户在面板上点了
          // 一下「当前就是这个」的那一项，文档就莫名其妙变脏了
          const current = node.attrs['language'];
          if ((typeof current === 'string' ? current : '') === language) return;
          view.dispatch(view.state.tr.setNodeAttribute(pos, 'language', language));
        });
      });
    },

    close,
  };
}

/**
 * 从剪贴板里挑出图片文件。
 *
 * **两处都要看。** 各平台 WebView 的填充方式不一致：
 * 有的把图片放进 `dataTransfer.files`，有的只放进 `dataTransfer.items`
 * 且 `files` 是空的。只看 `files` 的后果是「在某些机器上粘贴图片没反应」——
 * 而这种「只在别人的机器上坏」的 bug 最难查。
 *
 * 非图片（从文件管理器复制一个 `.txt`）会被过滤掉：那种粘贴应该走
 * ProseMirror 默认的文本/文件处理，而不是被我们截住。
 */
function imageFilesOf(data: DataTransfer | null): File[] {
  if (data === null) return [];

  const out: File[] = [];
  const consider = (file: File | null): void => {
    if (file !== null && file.type.startsWith('image/')) out.push(file);
  };

  for (const file of Array.from(data.files)) consider(file);
  if (out.length > 0) return out;

  for (const item of Array.from(data.items)) {
    if (item.kind === 'file') consider(item.getAsFile());
  }
  return out;
}

/**
 * 把一张图片插到光标处。
 *
 * 与 Milkdown 的 `insertImageCommand` 逐字一致（`replaceSelectionWith` +
 * `scrollIntoView`），只是这里不经过它的 command manager：`handlePaste` 在
 * `Editor.make()` 的 config 阶段就要交出去，那时还拿不到编辑器实例，
 * 而 `view` 是 ProseMirror 直接给我们的。为同一行代码绕一圈去取 `commandsCtx`
 * 不划算 —— 代价是这层「与上游一致」的约定需要一句注释守住。
 */
function insertImageNode(view: EditorView, src: string): void {
  const type = view.state.schema.nodes[IMAGE];
  if (type === undefined) return;
  const image = type.create({ src, alt: '', title: '' });
  view.dispatch(view.state.tr.replaceSelectionWith(image).scrollIntoView());
}

/**
 * 把剪贴板里的图片逐张交给外壳落盘，再把换回来的路径插进文档。
 *
 * ## 为什么必须自己声明「这是用户发起的编辑」
 *
 * 落盘是异步的（要读字节、写文件），等它回来时 `markdownEmitPlugin` 的
 * 「刚发生过用户输入」窗口早就过了 —— 与浮层按钮、查找替换是同一个问题。
 * 不声明的话文档不会变脏，用户按 `Ctrl+S` 写回磁盘的是**粘贴之前**的文本，
 * 而界面上那张图还在。这是本项目第四次遇到这条路径（见 `withUserEdit` 的注释）。
 *
 * ## 失败要一张一张地跳过，不能让整次粘贴崩掉
 *
 * 三张图里有一张格式不认识，另外两张仍然应该落盘 ——
 * 所以循环里逐张 `try`，且 `null` 只代表「这一张放弃」。
 */
async function insertPastedImages(
  view: EditorView,
  handler: (request: ImagePasteRequest) => Promise<string | null>,
  files: readonly File[],
): Promise<void> {
  const sources: string[] = [];

  for (const file of files) {
    let src: string | null;
    try {
      src = await handler({ bytes: await file.arrayBuffer(), mimeType: file.type });
    } catch {
      // 外壳理论上不抛（它内部已把失败收敛成 null），但这里不能假设：
      // 一个未捕获的拒绝会变成没人处理的 Promise，而用户只看到「粘贴没反应」
      src = null;
    }
    if (src !== null && src !== '') sources.push(src);
  }

  if (sources.length === 0) return;

  // 逐张插入而不是一次事务插全部：`replaceSelectionWith` 会把光标挪到
  // 刚插入的节点之后，所以连续调用是「追加」，而一次事务里插多个节点
  // 需要自己算位置 —— 没必要。
  for (const src of sources) {
    withUserEditOnView(view, () => insertImageNode(view, src));
  }
}

/**
 * 把序列化的符号选项对齐到最常见的 Markdown 写法。
 *
 * ## 为什么单独拆成函数、还要显式标返回类型
 *
 * `ctx.update` 的签名是 `update<T, N>(slice: SliceType<T, N> | N, updater: (prev: T) => T)` ——
 * `T` 在回调的**返回位置**上仍然是一个推断目标，上下文类型还没定下来。
 * 于是把 `'-'` 直接写进返回的对象字面量时，它会被**放宽成 `string`**，
 * 赋值检查报 TS2322（`string` 不可赋给 `'*' | '+' | '-' | null | undefined`）。
 *
 * 标上返回类型之后字面量按目标类型收窄，问题消失。**不要改回内联箭头函数** ——
 * 那会重新触发同一个报错，而且报错信息（「`string` 不可赋给 `'*' | '+' | '-'`」）
 * 指向的是 `ctx.update` 那一行，看不出真正的原因是「返回位置的字面量被放宽了」。
 *
 * ## 语义：见 `createMarkdownEditor` 里调用处的注释
 *
 * 一句话：`remark-stringify` 默认把无序列表写成 `* 项目`，而 `-` 才是绝大多数人的写法。
 * 不改这一条，「打开一份用 `-` 写的文档、改一个字、保存」会把**全篇**的 `-` 静默换成 `*`。
 */
function useDashSymbols(options: RemarkStringifyOptions): RemarkStringifyOptions {
  return { ...options, bullet: '-', rule: '-' };
}

export async function createMarkdownEditor(
  options: CreateMarkdownEditorOptions,
): Promise<MarkdownEditorInstance> {
  /**
   * 内核当前的 markdown 源文本（**规范化后**的写法）。
   *
   * 缓存而不是每次现序列化，是为了让 `getMarkdown()` 保持同步 ——
   * 外壳在窗口关闭、`beforeunload` 这类时机需要同步拿到内容，异步拿不到。
   */
  let current = options.defaultValue;

  /**
   * 语言选择器的共享状态。**必须在 `Editor.make()` 之前建好** ——
   * NodeView 插件是在 `.use()` 里装上去的，那时编辑器还不存在。
   */
  const picker: CodeBlockPicker = {
    target: null,
    notify: (state) => options.onCodeBlockChange?.(state),
  };

  /**
   * 建一次就固定下来。传给插件的是「取它的函数」——
   * 这样插件拿到的是同一个对象，而不是每次调用都新建一个。
   */
  const codeBlockHandlers = codeBlockPickerHandlers(picker, options.onDiagramOpen);

  const editor = await Editor.make()
    .config((ctx) => {
      ctx.set(rootCtx, options.root);
      ctx.set(defaultValueCtx, options.defaultValue);
      ctx.set(editorViewOptionsCtx, {
        editable: () => !options.readOnly,
        // 内容元素上挂共用版式类名：编辑器与导出用同一套 CSS，这是「所见即所得导出」的前提
        attributes: { class: DOC_CLASS },
        /**
         * 粘贴图片。
         *
         * 挂在 `editorViewOptionsCtx` 而不是自建一个 `$prose` 插件：这是
         * ProseMirror 为「我要先看一眼剪贴板」准备的官方钩子，返回 `true`
         * 表示已处理、不再走默认的 slice 解析。自建插件要用
         * `props.handlePaste` 复现同一个位置，多一层没有收益的间接。
         *
         * 这里**不检查 readOnly** —— 读源码确认过：`prosemirror-view` 注册
         * 事件监听时用的是 `view.editable || !(event.type in editHandlers)`，
         * 所以只读视图里 `paste` 这个监听**根本没装上**，`handlePaste` 不会被调用。
         *
         * 这条不能想当然，因为我们的处理顺序是「**先落盘、再插节点**」
         * （见 `insertPastedImages`）。要是只读视图里也会走到这里，
         * 就会在磁盘上留下一个文档里没有引用的孤儿图片 ——
         * 而「多写一个 readOnly 判断」在这里反而是有害的：
         * 它会让「ProseMirror 到底拦不拦」这件事变得没人知道。
         */
        handlePaste: (view, event) => {
          const handler = options.onImagePaste;
          if (handler === undefined) return false;

          const files = imageFilesOf(event.clipboardData);
          if (files.length === 0) return false;

          // 必须在这里就拦下：默认处理会把剪贴板解析成一个 slice 插进去，
          // 而对一张图片来说那要么是空的、要么是一段没意义的文本
          event.preventDefault();
          void insertPastedImages(view, handler, files);
          return true;
        },
      });
      /**
       * 解析前把 mdast 里「声明为字符串、实际是 null」的字段收敛成空字符串。
       *
       * 不加这一步，**没有标题的图片会被静默丢出文档** ——
       * 用户改一个字保存，插图就全没了。细节见 `normalize-ast.ts`。
       *
       * 注册必须走 config（早于 parser 创建），不能等编辑器建好再挂。
       */
      ctx.update(remarkPluginsCtx, (plugins) => [...plugins, { plugin: normalizeMarkdownAst, options: {} }]);

      /**
       * 序列化时用的符号：**对齐到最常见的 Markdown 写法**。
       *
       * ## 不改这一条会怎样
       *
       * `remark-stringify` 的默认是 `bullet: '*'` —— 无序列表一律写成 `* 项目`。
       * 而 `-` 才是绝大多数人的写法（Typora 的默认、GitHub 上最常见、也更短）。
       *
       * 于是「打开一份用 `-` 写的文档 → 改一个字 → 保存」会让**全篇的 `-` 静默变成 `*`**，
       * 而那些行用户根本没碰过。实测一份 23419 字符的文档里这类改写有 18 处，
       * 而它**连序列化器的「已知且已接受」清单都没进**（清单在
       * `markdown-core.test.ts`，那里压根没提列表符号）。
       *
       * 分隔线同理：`---` 被写成 `***`，而 `---` 是它最常见的写法。
       *
       * ## 这**不是**在解决问题，只是把受害面缩到最小
       *
       * 真正的解法是「只把改动写回它发生的位置」（horseMD 的卖点），
       * 那要换架构（源码优先 / Live Preview），不是配置能办到的。
       * 在换之前，这一条保证的是：**最常用的写法不再被改写**。
       * 用 `*` 写列表的人仍然会被改成 `-` —— 但那比反过来少见得多
       * （`*` 与强调标记视觉上容易混，很多人本来就避开它）。
       *
       * 表格的列宽重排改不掉（那是 remark 的表格式化），仍然是已知代价。
       */
      ctx.update(remarkStringifyOptionsCtx, useDashSymbols);
    })
    .use(commonmark)
    .use(gfm)
    .use(history)
    .use(clipboard)
    .use(cursor)
    .use(trailing)
    /**
     * 公式。五样东西缺一不可：
     *
     * - `mathRemarkPlugin` —— 让 `$…$` 在**解析阶段**就变成 mdast 的公式节点。
     *   少了它，schema 里的两个节点永远不会被用到，公式在源码视图里
     *   也只是一段普通文字；
     * - 两个 `$nodeSchema` —— 文档树里的节点类型（值存在 `value` 属性上）；
     * - 两条输入规则 —— 打字时的即时转换；
     * - `mathViewPlugin` —— 渲染与**原位编辑**。
     *
     * 顺序没有讲究（它们互不依赖），但**必须都在 `.create()` 之前** ——
     * `$nodeSchema` 是在 schema 构建阶段注册节点类型的。
     */
    .use(mathRemarkPlugin)
    .use(mathInlineSchema)
    .use(mathBlockSchema)
    .use(mathInlineInputRule)
    .use(mathBlockInputRule)
    .use(mathViewPlugin())
    .use(codeHighlightPlugin)
    .use(codeBlockChromePlugin(() => codeBlockHandlers))
    .use(imageViewPlugin(() => ({ resolve: options.resolveImageUrl, onClick: options.onImageClick })))
    .use(cursorReporterPlugin(() => options.onCursorChange))
    .use(positionReporterPlugin(options.root, () => options.onPositionChange))
    .use(selectionReporterPlugin(() => options.onSelectionChange))
    .use(slashReporterPlugin(() => options.onSlashChange))
    .use(tableReporterPlugin(() => options.onTableChange))
    .use($prose(() => placeholderPlugin(options.placeholder ?? '')))
    .use(
      $prose((ctx) => {
        // 这个工厂可能在 serializer 就绪之前就被调用，所以惰性取一次并缓存：
        // 真正用到它是在事务的 state.apply 里，那时序列化器一定已经就绪。
        let cached: ((doc: ProseNode) => string) | null = null;
        const serialize = (doc: ProseNode): string => {
          cached ??= ctx.get(serializerCtx);
          return cached(doc);
        };
        return markdownEmitPlugin((markdown) => {
          current = markdown;
          options.onChange(markdown);
        }, serialize);
      }),
    )
    .create();

  /**
   * 用**规范化后**的写法初始化 `current`，而不是入参原文。
   *
   * 规范化是真实存在的：实测 setext 标题（`标题\n====`）会变成 ATX（`# 标题`）、
   * 行尾空白与连续空行会被收敛、`~~~` 围栏变成 ```、`___` 分隔线变成 `---`、
   * 硬换行的两个行尾空格变成反斜杠、**表格按列宽重排**。`*斜体*` 倒是原样保留。
   * 让 `getMarkdown()` 一开始就说真话，好过等到第一次编辑才悄悄变样。
   *
   * 这不影响「打开后一个字没改就不该是脏的」：外壳那边的脏判定只认
   * `onChange` 报上来的值，而初始文档永远不上报（见 `markdownEmitPlugin`）。
   */
  current = editor.action((ctx) => ctx.get(serializerCtx)(ctx.get(editorViewCtx).state.doc));

  return {
    getMarkdown: () => current,

    setMarkdown: (markdown) => {
      editor.action((ctx) => {
        const view = ctx.get(editorViewCtx);
        const parse = ctx.get(parserCtx);
        const serialize = ctx.get(serializerCtx);
        const next = parse(markdown);
        const { state } = view;

        view.dispatch(
          state.tr
            // 只替换内容、复用现有 state：schema 与各插件状态都保留
            .replaceWith(0, state.doc.content.size, next.content)
            // 程序写入，别当成用户改动报给外壳 —— 否则「重新加载」会让文档立刻变脏
            .setMeta(SILENT_META, true)
            // 整篇替换不是一个自然的撤销步骤，不进撤销栈
            .setMeta('addToHistory', false),
        );

        /**
         * 记下**规范化后**的写法，而不是入参原文。
         *
         * 这是外壳去重的基准：外壳拿入参原文一比，就会发现和内核报的值不一样，
         * 以为内核又改了内容，于是再推一次 —— 来回成环。
         *
         * 注意 `@milkdown/plugin-trailing` 可能在这次 dispatch 里追加一个空段落
         * （文档以代码块 / 列表 / 表格结尾时），`view.dispatch` 是同步跑完
         * `appendTransaction` 的，所以这里取到的 `view.state.doc` 已经包含它。
         */
        current = serialize(view.state.doc);
      });
    },

    focus: () => {
      editor.action((ctx) => ctx.get(editorViewCtx).focus());
    },

    /**
     * 当前光标与滚动位置。
     *
     * `offset` 用的是 ProseMirror 的文档位置，`textLength` 用 `doc.content.size` ——
     * 两者在**同一个坐标系**里，且都是 O(1)。刻意不去换算成「第几个可见字符」：
     * 那要重新实现一遍 `textBetween` 在块之间插分隔符的规则，而抄一份的下场
     * 是两份实现在边界上悄悄分叉（见 `EditorPosition` 的注释）。
     *
     * `scrollRatio` 读的是 `root` —— `.qm-editor`，也就是内核自己那个滚动容器。
     */
    getPosition: () =>
      editor.action((ctx) => {
        const view = ctx.get(editorViewCtx);
        return {
          offset: view.state.selection.head,
          textLength: view.state.doc.content.size,
          scrollRatio: readScrollRatio(options.root),
        };
      }),

    /**
     * 恢复光标与滚动。
     *
     * ## 两条刻意的取舍
     *
     * 1. **长度对不上就只恢复滚动。** 长度变了说明这份文档被改过（外部程序改的、
     *    或者上一次是别的内核存的），那个偏移可能已经不指同一个地方了。
     *    硬套上去的结果是光标落在一段无关的文字里，比「回到开头」更让人困惑。
     * 2. **不调 `scrollIntoView()`。** 光标与滚动是**分别**恢复的，因为我们恢复的
     *    是「用户离开时的那一屏」：他可能正滚在别处读，光标停在屏幕外 ——
     *    那正是他离开时的样子。把光标拉进视野反而破坏了这个状态。
     *
     * 选择事务不进撤销栈（`addToHistory: false`）：恢复位置不是一次编辑，
     * 用户按 Ctrl+Z 时不该先撤销掉一次「光标跳转」。
     */
    setPosition: (position) => {
      editor.action((ctx) => {
        const view = ctx.get(editorViewCtx);
        const { doc } = view.state;

        if (position.textLength === doc.content.size) {
          // 夹到合法范围：位置记忆坏掉不该让文档打不开
          const inside = Math.min(Math.max(position.offset, 0), doc.content.size);
          view.dispatch(
            view.state.tr
              .setSelection(TextSelection.near(doc.resolve(inside)))
              .setMeta('addToHistory', false),
          );
        }

        applyScrollRatio(options.root, position.scrollRatio);
      });
    },

    /**
     * 序列化成**语义 HTML 片段**。
     *
     * 刻意走 `DOMSerializer` 而不是 `serializerCtx`：后者产出的是 markdown，
     * 这里要的是 HTML。DOMSerializer 直接按 schema 把文档树转成 DOM，
     * 与编辑器里看到的结构逐节点一致，不需要再走一次 markdown → HTML 的往返。
     *
     * 产物**不含任何行内样式**，样式由 `core/export/document.ts` 内联的
     * 版式 CSS 提供（与编辑器同一份来源）。这是导出保真的前提：
     * 换了机器、没有网络、没有我们的样式表，观感也一致。
     */
    toHtml: () =>
      editor.action((ctx) => {
        const view = ctx.get(editorViewCtx);
        const serializer = DOMSerializer.fromSchema(view.state.schema);
        const fragment = serializer.serializeFragment(view.state.doc.content);
        const container = document.createElement('div');
        container.appendChild(fragment);
        return container.innerHTML;
      }),

    search: createMarkdownSearch(editor),

    formatting: createMarkdownFormatting(editor),

    table: createMarkdownTable(editor),

    codeBlock: createMarkdownCodeBlock(editor, picker),

    /**
     * 大纲跳转：按「层级 + 标题文字」在文档树里找标题节点。
     *
     * 为什么不用行号：文档树里没有行。硬把源文档的行号映射到位置，
     * 需要在序列化/解析的往返里维持一份行号表，而那份表只要有一次不精确，
     * 跳转就会落到无关的位置上 —— 对导航功能来说，「跳错」比「不跳」更让人失去信任。
     * 按文字匹配没有这个问题：找不到就如实返回 false，外壳会把它显示成不可跳转。
     */
    goToHeading: (level, text) =>
      editor.action((ctx) => {
        const view = ctx.get(editorViewCtx);

        const found = findHeadingPos(view.state.doc, level, text.trim());
        if (found === null) return false;

        // +1 进到标题节点内部，避免把光标停在节点边界上
        const inside = Math.min(found + 1, view.state.doc.content.size);
        view.dispatch(view.state.tr.setSelection(TextSelection.near(view.state.doc.resolve(inside))).scrollIntoView());
        view.focus();
        return true;
      }),

    // Editor.destroy() 返回 Promise<Editor>，这里收窄成 Promise<void>，
    // 免得外壳被迫依赖 @milkdown/core 的类型
    destroy: async () => {
      await editor.destroy();
    },
  };
}
