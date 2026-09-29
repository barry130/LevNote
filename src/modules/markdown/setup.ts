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
  defaultValueCtx,
  editorViewCtx,
  editorViewOptionsCtx,
  parserCtx,
  remarkPluginsCtx,
  rootCtx,
  serializerCtx,
} from '@milkdown/kit/core';
import { clipboard } from '@milkdown/kit/plugin/clipboard';
import { cursor } from '@milkdown/kit/plugin/cursor';
import { history } from '@milkdown/kit/plugin/history';
import { trailing } from '@milkdown/kit/plugin/trailing';
import type { Node as ProseNode } from '@milkdown/kit/prose/model';
import { DOMSerializer } from '@milkdown/kit/prose/model';
import { TextSelection } from '@milkdown/kit/prose/state';
import { commonmark } from '@milkdown/kit/preset/commonmark';
import { gfm } from '@milkdown/kit/preset/gfm';
import { $prose } from '@milkdown/kit/utils';

import type { CursorPosition, EditorSearch } from '../../core/editor/types.js';
import { createFindPattern, findWithPattern, type FindOptions } from '../../core/search/find.js';
import { normalizeMarkdownAst } from './normalize-ast.js';
import {
  codeHighlightPlugin,
  cursorReporterPlugin,
  markdownEmitPlugin,
  placeholderPlugin,
  SILENT_META,
  USER_EDIT_META,
} from './plugin.js';

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
}

export interface MarkdownEditorInstance {
  getMarkdown(): string;
  setMarkdown(markdown: string): void;
  focus(): void;
  /** 语义 HTML 片段，不含 `<html>` / `<head>`。 */
  toHtml(): string;
  readonly search: EditorSearch;
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

  const editor = await Editor.make()
    .config((ctx) => {
      ctx.set(rootCtx, options.root);
      ctx.set(defaultValueCtx, options.defaultValue);
      ctx.set(editorViewOptionsCtx, {
        editable: () => !options.readOnly,
        // 内容元素上挂共用版式类名：编辑器与导出用同一套 CSS，这是「所见即所得导出」的前提
        attributes: { class: DOC_CLASS },
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
    })
    .use(commonmark)
    .use(gfm)
    .use(history)
    .use(clipboard)
    .use(cursor)
    .use(trailing)
    .use(codeHighlightPlugin)
    .use(cursorReporterPlugin(() => options.onCursorChange))
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
   * 行尾空白与连续空行会被收敛、`~~~` 围栏变成 ```、`___` 分隔线变成 `***`、
   * 硬换行的两个行尾空格变成反斜杠。`*斜体*` 倒是原样保留。
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
