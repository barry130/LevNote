/**
 * 代码 / 文本内核（CodeMirror 6）。
 *
 * ## 为什么是 CodeMirror 6 而不是自己用 textarea 凑合
 *
 * 产品定位里这一类要对标 Notepad++。textarea 做不到的三件事恰好是「写代码」的核心：
 * 语法着色、行号、缩进/括号/折叠这些结构性编辑命令。
 * 而 CodeMirror 6 的**架构**正好与本项目的适配层契约同构：
 * 它是「不可变 state + 显式 transaction」，没有隐藏的内部状态，
 * 所以 `setValue` 与「用户输入」在模型层是同一件事的不同标注 ——
 * 这正是我们在 Milkdown 那边踩过坑的地方（回声、假脏），这里从架构上就不成立。
 *
 * ## 与 Markdown 内核的取舍差异
 *
 * Markdown 那边刻意**不用** `@milkdown/plugin-listener`（防抖会让「程序写入」的判定失效），
 * 这里同样刻意**不用** `@codemirror/search` 的内建面板：
 * 它的查找 UI 是一套自己的浮层，样式、快捷键、i18n 都不受我们控制，
 * 而外壳已经有一个统一的查找栏。我们只用 CM 的 `SearchCursor` 语义
 * （这里用 `core/search/find.ts` 的纯函数实现同一件事），
 * 把 UI 留给自己 —— **内核负责模型，外壳负责界面**，这条边界不能破。
 *
 * ## 主题
 *
 * 版式与配色全部走 CSS 变量（`--qm-*` / `--syn-*`），而不是把色值写进 JS。
 * 于是「深色主题下代码块没换色」永远是一个 CSS 问题，不用回来翻这个文件；
 * 也让编辑器与导出的 HTML 共用同一套色板（见 `document-styles.css`）。
 */

import { Annotation, Compartment, EditorSelection, EditorState } from '@codemirror/state';
import { EditorView, keymap, placeholder as cmPlaceholder } from '@codemirror/view';
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands';
import { highlightSelectionMatches, searchKeymap } from '@codemirror/search';
import { bracketMatching, foldGutter, foldKeymap, HighlightStyle, indentOnInput, syntaxHighlighting } from '@codemirror/language';
import { closeBrackets, closeBracketsKeymap, completionKeymap } from '@codemirror/autocomplete';
import { javascript } from '@codemirror/lang-javascript';
import { json } from '@codemirror/lang-json';
import { css } from '@codemirror/lang-css';
import { html } from '@codemirror/lang-html';
import { markdown } from '@codemirror/lang-markdown';
import { python } from '@codemirror/lang-python';
import { rust } from '@codemirror/lang-rust';
import { xml } from '@codemirror/lang-xml';
import { yaml } from '@codemirror/lang-yaml';
import { sql } from '@codemirror/lang-sql';
import type { Extension } from '@codemirror/state';
import { tags as t } from '@lezer/highlight';

import type { CursorPosition, EditorPosition, EditorSearch } from '../../core/editor/types.js';
import { applyScrollRatio, readScrollRatio } from '../../core/editor/position.js';
import { stripInlineMarkup } from '../../core/outline/outline.js';
import { isMarkdownPath } from '../../core/path.js';
import { createFindPattern, findInText, replaceInText, type FindOptions, type TextMatch } from '../../core/search/find.js';
import { extensionOf } from '../../core/path.js';

/** 标记「这次变更是程序写入的」。与 Markdown 内核的 `SILENT_META` 同一个用途。 */
const SILENT = Annotation.define<boolean>();

export interface CreateCodeEditorOptions {
  root: HTMLElement;
  defaultValue: string;
  language: string;
  readOnly?: boolean;
  wordWrap?: boolean;
  placeholder?: string;
  onChange(value: string): void;
  onCursorChange?(position: CursorPosition): void;
  /** 光标或滚动位置变化（位置记忆）。高频，调用方别在里面做重活。 */
  onPositionChange?(position: EditorPosition): void;
}

export interface CodeEditorInstance {
  getValue(): string;
  setValue(value: string): void;
  focus(): void;
  /** 当前光标与滚动位置（位置记忆）。 */
  getPosition(): EditorPosition;
  /** 恢复光标与滚动。`textLength` 对不上时只恢复滚动。 */
  setPosition(position: EditorPosition): void;
  /** `<pre data-language>` 片段，交给导出层上色（与 Markdown 的代码块走同一条路）。 */
  toHtml(): string;
  setWordWrap(on: boolean): void;
  readonly search: EditorSearch;
  goToHeading(level: number, text: string): boolean;
  destroy(): void;
}

// ---------------------------------------------------------------------------
// 语言
// ---------------------------------------------------------------------------

/**
 * 扩展名 → 语言。
 *
 * 名单**刻意比 `TEXT_EXTENSIONS` 短**：装了什么语言就写什么。
 * 没装的语言（Go / Java / C++ / Shell）仍然能被这个内核打开 ——
 * 只是没有语法着色。**「打开」与「着色」是两件事**，不能因为缺一个语言包
 * 就让用户打不开文件。这也与产品的「轻」一致：不为长尾语言背几十个语法包的体积。
 */
export function languageForPath(path: string): string {
  /**
   * Markdown 家族走 `core/path.ts` 的名单，而不是在这里逐个 `case`。
   *
   * 这条路径有两个入口：打开一个 `.md` 源码文件，以及**在 Markdown 文档里
   * 切到源码视图**（见 `EditorAdapter.sourceView`）。后者尤其不能漏 ——
   * 漏了 `.mdown` 的表现是「同一个文档，切到源码视图突然没有语法着色」，
   * 用户会以为源码视图坏了。
   */
  if (isMarkdownPath(path)) return 'markdown';

  const ext = extensionOf(path);
  switch (ext) {
    case '.js':
    case '.mjs':
    case '.cjs':
      return 'javascript';
    case '.jsx':
      return 'jsx';
    case '.ts':
    case '.mts':
    case '.cts':
      return 'typescript';
    case '.tsx':
      return 'tsx';
    // 导图文件（`.mindmap`）就是 JSON。这条同时服务两个入口：
    // 打开一份**被导图内核否决**的 `.mindmap`，以及给一份正常的 `.mindmap`
    // 切到源码视图（`sourceView`）。后者尤其不能漏 ——
    // 漏了的表现是「同一个文件，切到源码视图突然没有着色」。
    case '.json':
    case '.jsonc':
    case '.mindmap':
      return 'json';
    case '.css':
      return 'css';
    case '.html':
    case '.htm':
      return 'html';
    case '.xml':
    case '.svg':
    case '.drawio':
      return 'xml';
    case '.py':
    case '.pyw':
      return 'python';
    case '.rs':
      return 'rust';
    case '.yml':
    case '.yaml':
      return 'yaml';
    case '.sql':
      return 'sql';
    default:
      return '';
  }
}

function languageExtension(language: string): Extension {
  switch (language) {
    case 'javascript':
      return javascript();
    case 'jsx':
      return javascript({ jsx: true });
    case 'typescript':
      return javascript({ typescript: true });
    case 'tsx':
      return javascript({ typescript: true, jsx: true });
    case 'json':
      return json();
    case 'css':
      return css();
    case 'html':
      return html();
    case 'xml':
      return xml();
    case 'markdown':
      return markdown();
    case 'python':
      return python();
    case 'rust':
      return rust();
    case 'yaml':
      return yaml();
    case 'sql':
      return sql();
    default:
      return [];
  }
}

// ---------------------------------------------------------------------------
// 外观
// ---------------------------------------------------------------------------

/**
 * 着色规则。
 *
 * 用 CSS 变量而不是字面色值，是为了让「深色主题」只改 CSS 一处。
 * 变量的名字与 `document-styles.css` 里 hljs 那组一一对应 ——
 * 同一种语义（关键字、字符串、注释…）在编辑器与导出文件里必须是同一个颜色，
 * 否则用户会以为导出把代码改坏了。
 */
const highlight = HighlightStyle.define([
  { tag: [t.comment, t.lineComment, t.blockComment], color: 'var(--syn-comment)', fontStyle: 'italic' },
  { tag: [t.keyword, t.controlKeyword, t.moduleKeyword, t.operatorKeyword], color: 'var(--syn-keyword)' },
  { tag: [t.string, t.special(t.string), t.regexp], color: 'var(--syn-string)' },
  { tag: [t.number, t.bool, t.null, t.atom], color: 'var(--syn-number)' },
  { tag: [t.typeName, t.className, t.namespace], color: 'var(--syn-type)' },
  { tag: [t.function(t.variableName), t.function(t.propertyName)], color: 'var(--syn-function)' },
  { tag: [t.definition(t.variableName), t.definition(t.propertyName)], color: 'var(--syn-def)' },
  { tag: [t.propertyName, t.attributeName], color: 'var(--syn-property)' },
  { tag: [t.tagName], color: 'var(--syn-tag)' },
  { tag: [t.operator, t.punctuation, t.separator, t.bracket], color: 'var(--syn-punct)' },
  { tag: [t.meta, t.processingInstruction], color: 'var(--syn-comment)' },
  { tag: [t.heading], color: 'var(--syn-def)', fontWeight: '650' },
  { tag: [t.strong], fontWeight: '680' },
  { tag: [t.emphasis], fontStyle: 'italic' },
  { tag: [t.link, t.url], color: 'var(--syn-link)', textDecoration: 'underline' },
  { tag: [t.invalid], color: 'var(--danger)' },
]);

const editorTheme = EditorView.theme({
  '&': {
    height: '100%',
    fontSize: '13.5px',
    color: 'var(--ink)',
    backgroundColor: 'transparent',
  },
  '.cm-scroller': {
    fontFamily: 'var(--mono)',
    lineHeight: '1.7',
    overflow: 'auto',
  },
  '.cm-content': { caretColor: 'var(--vermilion)', padding: '10px 0' },
  '.cm-gutters': {
    backgroundColor: 'transparent',
    color: 'var(--line-strong)',
    border: 'none',
    borderRight: '1px solid var(--line)',
    userSelect: 'none',
  },
  '.cm-activeLineGutter': { backgroundColor: 'transparent', color: 'var(--slate)' },
  '.cm-activeLine': { backgroundColor: 'var(--surface-sunken)' },
  '.cm-cursor, .cm-dropCursor': { borderLeftColor: 'var(--vermilion)', borderLeftWidth: '2px' },
  '&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground, .cm-selectionBackground':
    { backgroundColor: 'var(--selection)' },
  '.cm-content ::selection': { backgroundColor: 'var(--selection)' },
  '.cm-selectionMatch': { backgroundColor: 'var(--selection-match)' },
  '.cm-matchingBracket, &.cm-focused .cm-matchingBracket': {
    backgroundColor: 'var(--selection-match)',
    outline: '1px solid var(--line-strong)',
  },
  '.cm-foldPlaceholder': {
    backgroundColor: 'var(--surface-sunken)',
    border: '1px solid var(--line)',
    color: 'var(--slate)',
    padding: '0 4px',
    borderRadius: '4px',
  },
  '.cm-tooltip': {
    backgroundColor: 'var(--surface)',
    border: '1px solid var(--line)',
    color: 'var(--ink)',
    borderRadius: '6px',
    boxShadow: '0 6px 20px rgba(0,0,0,0.12)',
  },
  '.cm-tooltip-autocomplete ul li[aria-selected]': {
    backgroundColor: 'var(--bamboo)',
    color: '#fff',
  },
  // 占位文案：CM 默认给的是半透明文字，深色主题下几乎看不见
  '.cm-placeholder': { color: 'var(--slate)', fontStyle: 'normal' },
});

// ---------------------------------------------------------------------------
// 创建
// ---------------------------------------------------------------------------

export function createCodeEditor(options: CreateCodeEditorOptions): CodeEditorInstance {
  const wrapCompartment = new Compartment();
  const readOnlyCompartment = new Compartment();

  const wrapExtension = (on: boolean): Extension => (on ? EditorView.lineWrapping : []);

  const state = EditorState.create({
    doc: options.defaultValue,
    extensions: [
      // 顺序有意义：`defaultKeymap` 在前，自定义键位在后才能覆盖它
      keymap.of([...closeBracketsKeymap, ...defaultKeymap, ...searchKeymap, ...historyKeymap, ...foldKeymap, ...completionKeymap, indentWithTab]),
      history(),
      closeBrackets(),
      bracketMatching(),
      indentOnInput(),
      foldGutter(),
      highlightSelectionMatches(),
      EditorState.allowMultipleSelections.of(true),
      EditorView.clickAddsSelectionRange.of((event) => event.altKey),
      syntaxHighlighting(highlight),
      editorTheme,
      languageExtension(options.language),
      wrapCompartment.of(wrapExtension(options.wordWrap ?? false)),
      readOnlyCompartment.of(EditorState.readOnly.of(options.readOnly ?? false)),
      options.placeholder ? cmPlaceholder(options.placeholder) : [],
      EditorView.updateListener.of((update) => {
        if (update.docChanged && !isSilent(update)) {
          options.onChange(update.state.doc.toString());
        }
        if (update.selectionSet || update.docChanged) {
          const head = update.state.selection.main.head;
          const line = update.state.doc.lineAt(head);
          options.onCursorChange?.({ line: line.number, column: head - line.from + 1 });
          // 与上面共用同一个判断：位置记忆关心的是同一批事件（光标动了、内容变了）
          options.onPositionChange?.({
            offset: head,
            // O(1)：CodeMirror 的文档长度是算好的字段，不是遍历出来的
            textLength: update.state.doc.length,
            // 用 `update.view` 而不是外面那个 `view` 变量：`EditorView` 的构造
            // 过程里就可能派发一次更新，那时 `const view =` 还没赋值（TDZ），
            // 引用它会抛 `Cannot access 'view' before initialization`。
            // `ViewUpdate` 自带 `view`，拿它永远安全。
            scrollRatio: readScrollRatio(update.view.scrollDOM),
          });
        }
      }),
    ],
  });

  const view = new EditorView({ state, parent: options.root });

  /**
   * 滚动单独监听，而不是塞进 `updateListener`。
   *
   * CodeMirror 的滚动**不产生 transaction** —— 它只改 DOM 的 `scrollTop`，
   * 所以 `updateListener` 根本收不到。滚动容器是 `view.scrollDOM`
   * （也就是 `.cm-scroller`），这是 CodeMirror 公开的 API，不是我们猜的类名。
   *
   * `passive: true`：回调里没有 `preventDefault`，声明被动能让浏览器不等回调
   * 就继续滚。
   */
  const onScroll = (): void => {
    options.onPositionChange?.({
      offset: view.state.selection.main.head,
      textLength: view.state.doc.length,
      scrollRatio: readScrollRatio(view.scrollDOM),
    });
  };
  view.scrollDOM.addEventListener('scroll', onScroll, { passive: true });

  return {
    getValue: () => view.state.doc.toString(),

    setValue: (value) => {
      const current = view.state.doc.toString();
      if (current === value) return;
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: value },
        // 程序写入，别当成用户改动报给外壳 —— 否则「重新加载」会让文档立刻变脏
        annotations: SILENT.of(true),
        // 整篇替换不是一个自然的撤销步骤，不进撤销栈
        // （与 Markdown 内核的 setMarkdown 保持一致）
      });
    },

    focus: () => view.focus(),

    /**
     * 当前光标与滚动位置。
     *
     * 这里 `offset` 就是**文档的字符偏移**（CodeMirror 的位置天然是这个），
     * 与 Markdown 内核的 ProseMirror 位置是两套坐标系 —— 这正是
     * `EditorPosition` 让外壳只做往返、不做解释的原因。两者靠 `textLength`
     * 区分：文档树的内容尺寸与源码字符数通常对不上，所以切视图后不会拿
     * 一套坐标去另一套里定位。
     */
    getPosition: () => ({
      offset: view.state.selection.main.head,
      textLength: view.state.doc.length,
      scrollRatio: readScrollRatio(view.scrollDOM),
    }),

    /**
     * 恢复光标与滚动。`textLength` 对不上时只恢复滚动 —— 理由与 Markdown 内核
     * 完全一致（见 `EditorHandle.setPosition`）。
     *
     * 不调 `EditorView.scrollIntoView`：滚动是单独按比例恢复的，
     * 光标与滚动各自回到用户离开时的样子，而 `scrollIntoView` 会把光标拉进视野、
     * 破坏「用户当时正滚在别处读」这个状态。
     */
    setPosition: (position) => {
      const doc = view.state.doc;
      if (position.textLength === doc.length) {
        const inside = Math.min(Math.max(position.offset, 0), doc.length);
        view.dispatch({ selection: { anchor: inside } });
      }
      applyScrollRatio(view.scrollDOM, position.scrollRatio);
    },

    /**
     * `<pre data-language>` 而不是带行内样式的 HTML。
     *
     * 导出层已经会遍历 `<pre><code>` 并调用同一个 highlight.js 上色
     * （见 `core/export/inline-html.ts`）。这里只要把**结构**给对，
     * 着色与版式就自动与 Markdown 文档里的代码块一致 ——
     * 不需要在导出层为「代码文件」再开一条分支。
     */
    toHtml: () => {
      const text = view.state.doc.toString();
      const language = options.language;
      const attr = language === '' ? '' : ` data-language="${language}"`;
      return `<pre${attr}><code>${escapeHtml(text)}</code></pre>`;
    },

    setWordWrap: (on) => {
      view.dispatch({ effects: wrapCompartment.reconfigure(wrapExtension(on)) });
    },

    search: createCodeSearch(view),

    /**
     * 大纲跳转。
     *
     * 代码内核这里**有**行号，所以可以精确定位。实现上仍按「层级 + 文字」匹配
     * 而不是收一个行号参数 —— 契约要同时服务两个内核，
     * 而 Markdown 内核给不出行号（见 `EditorHandle.goToOutlineItem` 的说明）。
     */
    goToHeading: (level, text) => {
      const doc = view.state.doc;
      const target = stripInlineMarkup(text);
      for (let number = 1; number <= doc.lines; number += 1) {
        const line = doc.line(number);
        const match = /^ {0,3}(#{1,6})(?:[ \t]+(.*?))?[ \t]*$/.exec(line.text);
        if (!match) continue;
        if ((match[1] ?? '').length !== level) continue;
        if (stripInlineMarkup((match[2] ?? '').replace(/[ \t]+#+[ \t]*$/, '')) !== target) continue;
        view.dispatch({
          selection: { anchor: line.from },
          effects: EditorView.scrollIntoView(line.from, { y: 'center' }),
        });
        view.focus();
        return true;
      }
      return false;
    },

    destroy: () => {
      // 先摘监听再销毁视图：`view.destroy()` 之后 `scrollDOM` 已经脱离文档，
      // 留在它上面的监听会跟着一起被回收，但显式摘掉更清楚，也免得
      // 将来有人把 `destroy` 改成「只是隐藏」时留下一个野监听
      view.scrollDOM.removeEventListener('scroll', onScroll);
      view.destroy();
    },
  };
}

function isSilent(update: { transactions: readonly { annotation(type: unknown): unknown }[] }): boolean {
  return update.transactions.some((tr) => tr.annotation(SILENT) === true);
}

function escapeHtml(text: string): string {
  return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

// ---------------------------------------------------------------------------
// 查找
// ---------------------------------------------------------------------------

/**
 * CodeMirror 位置就是 `doc.toString()` 的字符偏移。
 *
 * 这一点值得写下来：CM 的文档模型是「行数组」，位置是**行的 from + 行内偏移**。
 * 只要 `lineBreak` 是默认的 `\n`（我们不改它），`doc.toString()` 的字符偏移
 * 与 CM 位置就一一对应，于是纯函数算出来的匹配可以直接当 selection 用，
 * 不需要任何坐标翻译。改 `lineBreak` 会静默破坏这个前提 —— 所以不要改。
 */
function createCodeSearch(view: EditorView): EditorSearch {
  let matches: TextMatch[] = [];
  let pattern: RegExp | null = null;
  let options: FindOptions | null = null;

  const refind = (): number => {
    if (options === null) return 0;
    matches = findInText(view.state.doc.toString(), options);
    pattern = createFindPattern(options);
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
      view.dispatch({
        selection: EditorSelection.range(hit.from, hit.to),
        effects: EditorView.scrollIntoView(hit.from, { y: 'center' }),
      });
      view.focus();
    },

    replace: (index, replacement) => {
      const hit = matches[index];
      if (!hit || options === null) return matches.length;
      const insert = expandFor(hit, replacement, pattern, options.regex);
      view.dispatch({ changes: { from: hit.from, to: hit.to, insert } });
      return refind();
    },

    replaceAll: (replacement) => {
      if (options === null || matches.length === 0) return 0;
      const before = view.state.doc.toString();

      if (options.regex) {
        // 正则模式下替换文本里的 `$1` 要展开，而捕获组信息在逐条替换时拿不到 ——
        // 直接对整篇文本做一次替换，语义最准。代价是撤销栈里只有一步，
        // 但「全部替换」本来就是一个原子操作，这个代价是合理的。
        const { text, count } = replaceInText(before, options, replacement);
        if (count > 0) {
          view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } });
        }
        matches = [];
        return count;
      }

      const count = matches.length;
      // 逐条替换：偏移从后往前，前面匹配的位置才不会因为长度变化而失效。
      // （CodeMirror 自己也会重映射，但显式倒序能让「意图」在代码里看得见。）
      const changes = [...matches].reverse().map((hit) => ({ from: hit.from, to: hit.to, insert: replacement }));
      view.dispatch({ changes });
      matches = [];
      return count;
    },

    clear: () => {
      matches = [];
      pattern = null;
      options = null;
    },
  };
}

/** 正则模式下展开单条替换里的 `$1` / `$&`；字面模式下 `$` 是普通字符。 */
function expandFor(hit: TextMatch, replacement: string, pattern: RegExp | null, asRegex: boolean): string {
  if (!asRegex || pattern === null) return replacement;
  const single = new RegExp(pattern.source, pattern.flags.replace('g', ''));
  return hit.text.replace(single, replacement);
}
