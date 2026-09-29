/**
 * 把 ProseMirror 的选区翻译成**外壳的语言**。
 *
 * 这一层存在的唯一理由是那条边界：外壳不认识 ProseMirror，所以它不能自己去
 * 读 `state.selection`、不能拿 `mark.type.name` 去比字符串。翻译必须发生在内核这一侧。
 *
 * 反向的诱惑是「把 mark 名直接当格式名上报」。它当下能跑，但会在两个地方坏掉：
 * 一是 gfm 的删除线叫 `strike_through` 而不是 `strikethrough`（见下表），
 * 二是换内核之后名字全变。外壳一旦开始认这些名字，就再也没法换内核了。
 */

import type { ResolvedPos } from '@milkdown/kit/prose/model';
import type { EditorState } from '@milkdown/kit/prose/state';
import { findTable, selectedRect, tableNodeTypes } from '@milkdown/kit/prose/tables';
import type { EditorView } from '@milkdown/kit/prose/view';

import type {
  BlockFormat,
  InlineFormat,
  SelectionFormat,
  SelectionRect,
  TableAlign,
  TableState,
} from '../../core/editor/types.js';

/**
 * ProseMirror 的 mark 名 → 外壳的行内格式。
 *
 * `strike_through` 这个下划线写法是实测出来的，不是猜的：gfm 预设里
 * `strikethroughSchema` 的泛型参数写的就是 `"strike_through"`。
 * 写成 `strikethrough` 的后果是**删除线按钮永远不亮、点了也没反应**，
 * 而且不会报任何错 —— 这类「静默不生效」的错最难查。
 */
const MARK_FORMATS: ReadonlyMap<string, InlineFormat> = new Map([
  ['strong', 'strong'],
  ['emphasis', 'emphasis'],
  ['strike_through', 'strikethrough'],
  ['inlineCode', 'inlineCode'],
  ['link', 'link'],
]);

const NODE_FORMATS: ReadonlyMap<string, BlockFormat> = new Map([
  ['paragraph', 'paragraph'],
  ['heading', 'heading'],
  ['blockquote', 'blockquote'],
  ['bullet_list', 'bulletList'],
  ['ordered_list', 'orderedList'],
  ['code_block', 'codeBlock'],
]);

/**
 * 会被当作「块级格式」上报的容器，**由内向外**第一个命中者胜出。
 *
 * 顺序（内 → 外）很重要：段落套在「引用 → 列表」里时，用户心里的当前格式
 * 是列表，不是引用 —— 因为取消列表是他下一步最可能做的事。
 */
const CONTAINER_ORDER: readonly string[] = ['bullet_list', 'ordered_list', 'blockquote'];

/** 单个 mark 名 → 行内格式；不是已知格式时返回 null（内核可能挂了额外插件）。 */
export function inlineFormatOf(markName: string): InlineFormat | null {
  return MARK_FORMATS.get(markName) ?? null;
}

/** 单个节点名 → 块级格式；未知节点返回 null。 */
export function blockFormatOf(nodeName: string): BlockFormat | null {
  return NODE_FORMATS.get(nodeName) ?? null;
}

/**
 * 选区起点所在的块。
 *
 * 先由内向外找容器，再退回文本块本身。**不能反过来**：若先看文本块，
 * 一个「引用里的段落」会报成 `paragraph`，于是工具条上「引用」不亮、
 * 而用户明明在引用里 —— 他再点一下「引用」就会套第二层引用。
 */
export function resolveBlock($from: ResolvedPos): { block: BlockFormat; headingLevel: number | null } {
  for (let depth = $from.depth - 1; depth >= 1; depth -= 1) {
    const name = $from.node(depth).type.name;
    if (!CONTAINER_ORDER.includes(name)) continue;
    return { block: blockFormatOf(name) ?? 'paragraph', headingLevel: null };
  }

  const textblock = $from.parent;
  if (textblock.type.name === 'heading') {
    const level = textblock.attrs['level'];
    // 层级不是数字时如实给 null，而不是假装 1 —— 外壳会据此不点亮任何标题按钮
    return { block: 'heading', headingLevel: typeof level === 'number' ? level : null };
  }
  return { block: blockFormatOf(textblock.type.name) ?? 'paragraph', headingLevel: null };
}

/**
 * 选区当前生效的行内格式。
 *
 * 分两种情况，这不是啰嗦而是必须：
 * - **有选中文字**：取整个选区**共有**的 mark（`marksAcross`），
 *   于是选中 `普通**加粗**` 时加粗按钮不亮 —— 亮着会让人以为点一下是取消，
 *   实际上会变成整段加粗。
 * - **光标塌缩**：看 `storedMarks`，没有就退回光标所在位置的 marks。
 *   输入法上屏、切换格式之后还没打字时，格式就存在 `storedMarks` 里。
 */
export function activeInlineFormats(state: EditorState): ReadonlySet<InlineFormat> {
  const { selection, storedMarks } = state;
  const marks = selection.empty
    ? (storedMarks ?? selection.$from.marks())
    : (selection.$from.marksAcross(selection.$to) ?? []);

  const formats = new Set<InlineFormat>();
  for (const mark of marks) {
    const format = inlineFormatOf(mark.type.name);
    if (format !== null) formats.add(format);
  }
  return formats;
}

/** 选区所在链接的地址；不在链接里时返回 null。 */
export function activeLinkHref(state: EditorState): string | null {
  const { selection } = state;
  const marks = selection.empty
    ? (state.storedMarks ?? selection.$from.marks())
    : (selection.$from.marksAcross(selection.$to) ?? []);

  for (const mark of marks) {
    if (mark.type.name !== 'link') continue;
    const href = mark.attrs['href'];
    return typeof href === 'string' ? href : '';
  }
  return null;
}

/**
 * 描述当前选区。**光标塌缩（没选中文字）时返回 `null`** ——
 * 浮层工具条只在有选中内容时出现，这是它的定义，不是实现细节。
 */
export function describeSelection(state: EditorState): SelectionFormat | null {
  const { selection } = state;
  if (selection.empty) return null;

  return {
    inline: activeInlineFormats(state),
    linkHref: activeLinkHref(state),
    ...resolveBlock(selection.$from),
  };
}

/**
 * 一段文档区间在视口里的包围盒。
 *
 * 取首尾两端各自的 `coordsAtPos` 再求并集，而不是只量起点：
 * 跨行的选区只量起点会让浮层压在文字上（起点在一行中间，浮层却按「整块」居中）。
 *
 * 量不出来时返回 `null`（`coordsAtPos` 在位置落在未渲染区域时会抛错）。
 * 这是**有意的降级**：外壳会把浮层藏起来。把浮层定位到一个瞎猜的坐标上，
 * 比不显示更糟 —— 用户会以为工具条坏了。
 */
export function rectOfRange(view: EditorView, from: number, to: number): SelectionRect | null {
  try {
    const start = view.coordsAtPos(from);
    // `-1` 表示「取这个位置**之前**那一侧」：选到行首时该落在上一行行尾，
    // 不传的话会落到下一行开头，包围盒凭空多出一行高
    const end = view.coordsAtPos(to, -1);

    const left = Math.min(start.left, end.left);
    const right = Math.max(start.right, end.right);
    const top = Math.min(start.top, end.top);
    const bottom = Math.max(start.bottom, end.bottom);

    return { top, left, bottom, right, width: right - left, height: bottom - top };
  } catch {
    return null;
  }
}

/** 选区在视口里的包围盒；没有选中文字、或量不出来时返回 `null`。 */
export function anchorOf(view: EditorView): SelectionRect | null {
  const { selection } = view.state;
  if (selection.empty) return null;
  return rectOfRange(view, selection.from, selection.to);
}

/**
 * 某个**光标位置**在视口里的矩形（塌缩，宽度通常为 0）。
 *
 * 与 `anchorOf` 分开是因为语义不同：那个是「选中了哪一块」，这个是「这个点在哪」。
 * 斜杠菜单要的是后者 —— 它锚在段落内容起点上，那个位置根本没有选区。
 */
export function caretAnchorOf(view: EditorView, pos: number): SelectionRect | null {
  try {
    const rect = view.coordsAtPos(pos);
    return {
      top: rect.top,
      left: rect.left,
      bottom: rect.bottom,
      right: rect.right,
      width: rect.right - rect.left,
      height: rect.bottom - rect.top,
    };
  } catch {
    return null;
  }
}

// ---------------------------------------------------------------------------
// 斜杠菜单
// ---------------------------------------------------------------------------

/**
 * 斜杠触发：整段只有 `/` 加上一段**不含空白**的筛选词。
 *
 * 不允许筛选词里有空格是有意的：菜单项的筛选词都是单字（`标题`、`表格`…），
 * 而一旦允许空格，用户写「一半 / 另一半」这种正文时就会莫名其妙弹出菜单 ——
 * 那种「你以为在打字，它以为在选命令」的干扰比少一个功能更烦人。
 */
const SLASH_TRIGGER = /^\/(\S*)$/;

/**
 * 判断当前是不是在斜杠触发状态。是则返回 `/` 后面已输入的文字（可能为空串），
 * 否则返回 `null`。
 *
 * 四个条件缺一不可：
 * - 光标**塌缩**（有选中文字时用户是在选东西，不是在打命令）；
 * - 所在文本块是**段落**（标题 / 代码块里不该弹菜单）；
 * - 光标停在文本**末尾**（不然用户把光标移回中间时菜单还在，看着像卡住了）；
 * - 整段匹配 `SLASH_TRIGGER`。
 */
export function describeSlash(state: EditorState): string | null {
  const { selection } = state;
  if (!selection.empty) return null;

  const { $from } = selection;
  if ($from.parent.type.name !== 'paragraph') return null;

  const text = $from.parent.textContent;
  if ($from.parentOffset !== text.length) return null;

  const hit = SLASH_TRIGGER.exec(text);
  return hit === null ? null : (hit[1] ?? '');
}

// ---------------------------------------------------------------------------
// 表格
// ---------------------------------------------------------------------------

/** 读出光标所在单元格的对齐方式；不在单元格里时按 `left`。 */
function cellAlign(state: EditorState): TableAlign {
  const types = tableNodeTypes(state.schema);
  const { $from } = state.selection;

  for (let depth = $from.depth; depth > 0; depth -= 1) {
    const node = $from.node(depth);
    if (node.type !== types.cell && node.type !== types.header_cell) continue;
    const value = node.attrs['alignment'];
    return value === 'center' || value === 'right' ? value : 'left';
  }
  return 'left';
}

/**
 * 光标所在表格的状态；不在表格里时返回 `null`。
 *
 * ## 为什么用 `findTable` + `selectedRect` 而不是自己走 `$from` 的祖先链
 *
 * 「光标在哪张表、第几行第几列」在**合并单元格**（`colspan` / `rowspan`）下
 * 不是一次简单的祖先查找 —— 一个单元格可能横跨多列，行列号要靠 `TableMap`
 * 的映射算。prosemirror-tables 的 `selectedRect` 就是干这个的，
 * 自己重写一遍只会把合并单元格算错，而且错得很隐蔽（普通表格上完全看不出来）。
 *
 * ## 两个返回 null 的地方
 *
 * - `findTable` 找不到：光标不在表格里，正常情况；
 * - `selectedRect` 抛错：表格结构损坏（单元格重叠、非矩形）时它会抛
 *   `RangeError`。如实返回 null 而不是让一次光标移动把编辑器带崩 ——
 *   「打开任何文件都不能失败」这条红线也适用于**改坏了的**表格。
 */
export function describeTable(view: EditorView): TableState | null {
  const { state } = view;

  const found = findTable(state.selection.$from);
  if (found === null) return null;

  let rect;
  try {
    rect = selectedRect(state);
  } catch {
    return null;
  }

  const map = rect.map;
  // 表格节点的区间：`found.pos` 是节点**之前**的位置
  const anchor = rectOfRange(view, found.pos, found.pos + found.node.nodeSize);
  if (anchor === null) return null;

  return {
    anchor,
    rows: map.height,
    columns: map.width,
    align: cellAlign(state),
    /**
     * 只剩两行就删不动了 —— Milkdown 的 gfm 表格 schema 是
     * `table_header_row table_row+`，**表头行 + 至少一行正文**是硬要求。
     * 报上去让外壳把按钮显示成禁用，而不是点了没反应。
     *
     * 注意这里**不区分光标在哪一行**：删表头行也是允许的（会由内核把下一行顶上来当表头），
     * 所以判据只有行数。
     */
    canDeleteRow: map.height >= 3,
    canDeleteColumn: map.width >= 2,
  };
}
