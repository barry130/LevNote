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
import type { EditorView } from '@milkdown/kit/prose/view';

import type { BlockFormat, InlineFormat, SelectionFormat, SelectionRect } from '../../core/editor/types.js';

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
 * 选区在视口里的包围盒。
 *
 * 取首尾两端各自的 `coordsAtPos` 再求并集，而不是只量起点：
 * 跨行的选区只量起点会让浮层压在文字上（起点在一行中间，浮层却按「整块」居中）。
 *
 * 量不出来时返回 `null`（`coordsAtPos` 在位置落在未渲染区域时会抛错）。
 * 这是**有意的降级**：外壳会把浮层藏起来。把浮层定位到一个瞎猜的坐标上，
 * 比不显示更糟 —— 用户会以为工具条坏了。
 */
export function anchorOf(view: EditorView): SelectionRect | null {
  const { selection } = view.state;
  if (selection.empty) return null;

  try {
    const start = view.coordsAtPos(selection.from);
    // `-1` 表示「取这个位置**之前**那一侧」：选到行首时该落在上一行行尾，
    // 不传的话会落到下一行开头，包围盒凭空多出一行高
    const end = view.coordsAtPos(selection.to, -1);

    const left = Math.min(start.left, end.left);
    const right = Math.max(start.right, end.right);
    const top = Math.min(start.top, end.top);
    const bottom = Math.max(start.bottom, end.bottom);

    return { top, left, bottom, right, width: right - left, height: bottom - top };
  } catch {
    return null;
  }
}
