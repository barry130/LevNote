/**
 * 代码内核的 **Markdown 源码级**格式化（`EditorFormatting` 的第二个实现）。
 *
 * ## 为什么代码内核也要这个能力
 *
 * `EditorFormatting` 此前只有 Markdown 富文本内核实现，于是一份 .md
 * **切到源码视图或分屏**（可编辑面是 CodeMirror）时，「段落 / 格式」菜单
 * 整体灰着 —— 用户看着一份 Markdown 却不能把它变成标题 / 列表，只会觉得坏了。
 *
 * 源码内核做格式化有天然的路：**改的就是 Markdown 文本本身**。标题是 `#`、
 * 引用是 `>`、加粗是 `**` —— 行级前缀与选区包裹，全是一次普通的文本编辑，
 * 与用户手敲没有区别（撤销栈、上报链都走 CodeMirror 的既有机制，
 * 不需要 `USER_EDIT_META` 那套上膛 —— 见 `setup.ts` 的 `undo` 注释）。
 *
 * ## 与富文本实现的语义对齐
 *
 * - **toggle 语义**：同一个动作点第二下回到原样（行内格式与引用 / 列表）；
 * - **代码块是终态**：已经在对围栏里时不再动作（与富文本实现同一条约定）；
 * - **链接**：给地址就加 / 改，不给地址只去掉 —— 内核不替用户猜一个地址。
 *
 * 每个动作都是**一次 dispatch**：撤销栈里一步、上报里一次，不产生中间态。
 */

import type { EditorView } from '@codemirror/view';

import type { BlockFormat, EditorFormatting, InlineFormat, InsertableBlock } from '../../core/editor/types.js';
import { drawioDocBlockStarter } from '../../core/drawio.js';
import { MERMAID_STARTER } from '../../core/mermaid.js';

/** 行内格式的源码标记。`link` 不在这里 —— 它的规则比「包一层」多。 */
const INLINE_MARKERS: Record<Exclude<InlineFormat, 'link'>, string> = {
  strong: '**',
  emphasis: '*',
  strikethrough: '~~',
  inlineCode: '`',
};

/** 已有块前缀。`paragraph` 要剥掉它们，列表互转时要先摘掉对方的记号。 */
const HEADING_RE = /^#{1,6}[ \t]/;
const QUOTE_RE = /^>[ \t]?/;
const BULLET_RE = /^[-*+][ \t]/;
const ORDERED_RE = /^\d+[.)][ \t]/;
const FENCE_RE = /^(```|~~~)/;

function escapeRegExp(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/** 选区覆盖到的**整行**范围（块级格式按行生效，不管选区只圈了半行）。 */
function lineSpan(view: EditorView): { from: number; to: number } {
  const { state } = view;
  const selection = state.selection.main;
  return {
    from: state.doc.lineAt(selection.from).from,
    to: state.doc.lineAt(selection.to).to,
  };
}

function selectedLines(view: EditorView): string[] {
  const { from, to } = lineSpan(view);
  return view.state.doc.sliceString(from, to).split('\n');
}

/** 剥掉一行的一种块前缀；没有该前缀时原样返回。 */
function stripPrefix(line: string, pattern: RegExp): string {
  return line.replace(pattern, '');
}

/** 光标（或选区终点）所在行。插入类动作都发生在「当前行的后面」。 */
function cursorLine(view: EditorView) {
  return view.state.doc.lineAt(view.state.selection.main.head);
}

// ---------------------------------------------------------------------------
// 行内格式
// ---------------------------------------------------------------------------

function toggleInlineMarker(view: EditorView, format: Exclude<InlineFormat, 'link'>): void {
  const marker = INLINE_MARKERS[format];
  const { state } = view;
  const selection = state.selection.main;
  const selected = state.doc.sliceString(selection.from, selection.to);

  // 选区自己就是「标记 + 内容 + 标记」→ 拆掉
  const wrapped = new RegExp(`^${escapeRegExp(marker)}(.+)${escapeRegExp(marker)}$`, 's');
  const inner = wrapped.exec(selected);
  if (inner !== null) {
    view.dispatch({
      changes: { from: selection.from, to: selection.to, insert: inner[1] ?? '' },
      selection: { anchor: selection.from, head: selection.from + (inner[1]?.length ?? 0) },
    });
    return;
  }

  // 选区两侧各有一个标记（可能选中了一部分）→ 删掉两侧
  const before = state.doc.sliceString(Math.max(0, selection.from - marker.length), selection.from);
  const after = state.doc.sliceString(selection.to, Math.min(state.doc.length, selection.to + marker.length));
  if (before === marker && after === marker) {
    view.dispatch({
      changes: [
        { from: selection.from - marker.length, to: selection.from },
        { from: selection.to, to: selection.to + marker.length },
      ],
      selection: { anchor: selection.from - marker.length, head: selection.to - marker.length },
    });
    return;
  }

  // 包上（空选区 → 光标落到中间等着打字）
  const insert = marker + selected + marker;
  view.dispatch({
    changes: { from: selection.from, to: selection.to, insert },
    selection:
      selected === ''
        ? { anchor: selection.from + marker.length }
        : { anchor: selection.from + marker.length, head: selection.from + marker.length + selected.length },
  });
}

function toggleLink(view: EditorView, value?: string): void {
  const { state } = view;
  const selection = state.selection.main;
  const selected = state.doc.sliceString(selection.from, selection.to);
  const href = value?.trim() ?? '';
  // 只认完整的 `[文字](地址)`；光标在链接中间但没有选全文时不硬猜范围
  const hit = /^\[(.+)\]\(([^)]*)\)$/s.exec(selected);

  if (href === '') {
    // 没给地址：只处理「去掉」。不在链接里就什么都不做 —— 不替用户猜地址
    if (hit !== null) {
      const text = hit[1] ?? '';
      view.dispatch({
        changes: { from: selection.from, to: selection.to, insert: text },
        selection: { anchor: selection.from, head: selection.from + text.length },
      });
    }
    return;
  }

  if (hit !== null) {
    // 已经是链接 → 只改地址（文字与选区保留）
    const next = `[${hit[1]}](${href})`;
    view.dispatch({ changes: { from: selection.from, to: selection.to, insert: next } });
    return;
  }

  // 加链接：有选中文字就用它，空选区放一个「链接」占位并选中，方便直接改
  const text = selected === '' ? '链接' : selected;
  view.dispatch({
    changes: { from: selection.from, to: selection.to, insert: `[${text}](${href})` },
    selection: { anchor: selection.from + 1, head: selection.from + 1 + text.length },
  });
}

// ---------------------------------------------------------------------------
// 块级格式（按行前缀）
// ---------------------------------------------------------------------------

function setLines(view: EditorView, lines: readonly string[]): void {
  const { from, to } = lineSpan(view);
  view.dispatch({
    changes: { from, to, insert: lines.join('\n') },
    // 光标保持在原选区的相对位置意义不大（行内容变了），收成一个塌缩点即可
    selection: { anchor: from },
  });
}

/** 整个选区（按行）是不是每一行都带着这个前缀。空行跳过 —— 引用块里允许空行。 */
function allMatch(lines: readonly string[], pattern: RegExp): boolean {
  return lines.every((line) => line.trim() === '' || pattern.test(line));
}

function toggleBlockPrefix(view: EditorView, format: Exclude<BlockFormat, 'heading' | 'codeBlock'>): void {
  const lines = selectedLines(view);

  switch (format) {
    case 'paragraph': {
      // 「变为正文」= 剥掉一层块记号（标题 / 引用 / 列表），与富文本的 lift 同义
      setLines(
        view,
        lines.map((line) =>
          stripPrefix(stripPrefix(stripPrefix(line, HEADING_RE), QUOTE_RE), BULLET_RE).replace(ORDERED_RE, ''),
        ),
      );
      return;
    }

    case 'blockquote': {
      if (allMatch(lines, QUOTE_RE)) setLines(view, lines.map((line) => stripPrefix(line, QUOTE_RE)));
      else setLines(view, lines.map((line) => `> ${stripPrefix(line, BULLET_RE).replace(ORDERED_RE, '')}`));
      return;
    }

    case 'bulletList': {
      if (allMatch(lines, BULLET_RE)) setLines(view, lines.map((line) => stripPrefix(line, BULLET_RE)));
      else {
        setLines(
          view,
          lines.map((line) => `- ${stripPrefix(stripPrefix(line, BULLET_RE), QUOTE_RE).replace(ORDERED_RE, '')}`),
        );
      }
      return;
    }

    case 'orderedList': {
      if (allMatch(lines, ORDERED_RE)) setLines(view, lines.map((line) => line.replace(ORDERED_RE, '')));
      else {
        let index = 0;
        setLines(
          view,
          lines.map((line) => {
            if (line.trim() === '') return line;
            index += 1;
            return `${index}. ${stripPrefix(stripPrefix(line, BULLET_RE), QUOTE_RE).replace(ORDERED_RE, '')}`;
          }),
        );
      }
      return;
    }
  }
}

function toggleHeading(view: EditorView, level?: number): void {
  const wanted = Math.min(Math.max(level ?? 1, 1), 6);
  const lines = selectedLines(view);
  const exact = new RegExp(`^#{${wanted}}[ \t]`);

  if (allMatch(lines, exact)) {
    // 已经全是这一级 → 取消（回到正文）。「原地不动」会让按钮看起来点坏了
    setLines(view, lines.map((line) => stripPrefix(line, HEADING_RE)));
    return;
  }
  setLines(
    view,
    lines.map((line) => `${'#'.repeat(wanted)} ${stripPrefix(line, HEADING_RE)}`),
  );
}

// ---------------------------------------------------------------------------
// 代码块围栏（终态语义：已经在围栏里就不再动作）
// ---------------------------------------------------------------------------

/** 光标是否在某对围栏里，在的话给出上下两条围栏线。 */
function enclosingFence(view: EditorView): { open: number; close: number } | null {
  const { doc } = view.state;
  const cursorAt = doc.lineAt(view.state.selection.main.head).number;

  let open = 0;
  for (let number = cursorAt - 1; number >= 1; number -= 1) {
    if (FENCE_RE.test(doc.line(number).text)) {
      open = number;
      break;
    }
  }
  if (open === 0) return null;

  for (let number = cursorAt; number <= doc.lines; number += 1) {
    if (number > open && FENCE_RE.test(doc.line(number).text)) {
      return { open, close: number };
    }
  }
  return null;
}

function toggleCodeBlock(view: EditorView): void {
  const fence = enclosingFence(view);
  // 与富文本实现同一条约定：代码块是终态，没有「取消」——
  // 退出的方式是选中围栏删掉，或把内容转为正文
  if (fence !== null) return;

  const { from, to } = lineSpan(view);
  view.dispatch({
    changes: [
      { from, to: from, insert: '```\n' },
      { from: to, to, insert: '\n```' },
    ],
    selection: { anchor: to + 4 },
  });
}

// ---------------------------------------------------------------------------
// 插入块（斜杠菜单那几项的源码等价物）
// ---------------------------------------------------------------------------

/** 在当前行之后另起一块的插入点（行尾 + 空行分隔，保证插入的东西按块解析）。 */
function insertionPoint(view: EditorView): { at: number; prefix: string } {
  const line = cursorLine(view);
  const empty = line.text.trim() === '';
  return empty ? { at: line.from, prefix: '' } : { at: line.to, prefix: '\n\n' };
}

function insertRawBlock(view: EditorView, body: string, cursorInside?: number): void {
  const { at, prefix } = insertionPoint(view);
  const insert = `${prefix}${body}`;
  view.dispatch({
    changes: { from: at, to: at, insert },
    selection: { anchor: at + prefix.length + (cursorInside ?? body.length) },
  });
}

// ---------------------------------------------------------------------------

/**
 * 出口。`view` 是**现取的**（由调用方传工厂）—— CodeMirror 实例与 view
 * 同生共死，这里只需要拿到当前那份。
 */
export function createMarkdownSourceFormatting(getView: () => EditorView): EditorFormatting {
  return {
    toggleInline: (format, value) => {
      const view = getView();
      if (format === 'link') toggleLink(view, value);
      else toggleInlineMarker(view, format);
    },

    toggleBlock: (format, level) => {
      const view = getView();
      if (format === 'heading') toggleHeading(view, level);
      else if (format === 'codeBlock') toggleCodeBlock(view);
      else toggleBlockPrefix(view, format);
    },

    insertBlock: (block: InsertableBlock) => {
      const view = getView();
      // 没有「/query 要先删」的问题（斜杠菜单只在富文本内核触发），
      // 块类直接走与 toggleBlock 同一条路，插入类给源码片段
      switch (block) {
        case 'paragraph':
          toggleBlockPrefix(view, 'paragraph');
          return;
        case 'heading1':
          toggleHeading(view, 1);
          return;
        case 'heading2':
          toggleHeading(view, 2);
          return;
        case 'heading3':
          toggleHeading(view, 3);
          return;
        case 'blockquote':
          toggleBlockPrefix(view, 'blockquote');
          return;
        case 'bulletList':
          toggleBlockPrefix(view, 'bulletList');
          return;
        case 'orderedList':
          toggleBlockPrefix(view, 'orderedList');
          return;
        case 'codeBlock':
          toggleCodeBlock(view);
          return;
        case 'mathBlock':
          insertRawBlock(view, '$$\n\n$$', 3);
          return;
        case 'diagram':
          insertRawBlock(view, '```mermaid\n' + MERMAID_STARTER + '\n```', '```mermaid\n'.length + MERMAID_STARTER.length);
          return;
        case 'drawio':
          // 起始内容与富文本内核共用同一份（core/drawio.ts）—— inline 模式，
          // 插进来就渲染；参考模式在这里只会立刻「读不到引用的文件」
          insertRawBlock(view, drawioDocBlockStarter());
          return;
        case 'table':
          // 表头带占位文字 —— 空表头的表格在序列化时会整行丢失（Milkdown 的
          // schema 约束，见 AGENTS 坑 12），占位文字让它至少能活过一次保存
          insertRawBlock(view, '| 列1 | 列2 | 列3 |\n| --- | --- | --- |\n|  |  |  |\n');
          return;
        case 'hr':
          insertRawBlock(view, '---\n');
          return;
      }
    },
  };
}
