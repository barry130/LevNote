// @vitest-environment jsdom
/**
 * 块手柄的**内核半边**：指针压在哪一块、四个块操作、转为、落点。
 *
 * ## 这一组测试真正在钉的两条不变量
 *
 * 1. **手柄作用的那一块，就是指针压着的那一块。**
 *    悬停命中（`blockIndexFromElement`）与命令目标（`target.index`）必须是同一个
 *    下标 —— 错位的表现是「点了『删除』，删掉的是另一块」，而界面上完全看不出来：
 *    手柄还老老实实贴在原来那一块旁边，菜单也画得好好的。
 * 2. **块操作必须让文档变脏。**
 *    这四个动作都不产生 `input` / `paste`，漏了 `USER_EDIT_META` 就是**静默丢数据**：
 *    用户点「下移」，界面显示成功，按 Ctrl+S 写回磁盘的却是**下移之前**的文本
 *    （硬约束 7，已经踩过六次）。
 *
 * ## 悬停那条链路在 jsdom 里跑得通，是刻意选的实现方式换来的
 *
 * 命中走的是 DOM 元素（`event.target` → `posAtDOM`），不是坐标 ——
 * 而 jsdom **没有** `document.elementFromPoint` / `caretRangeFromPoint`
 * （实测都是 `undefined`），`posAtCoords` 必抛。若走坐标，这个文件里
 * 「块操作」那一整节就一条都写不出来，而它们恰恰是最需要测的部分。
 *
 * 代价是**几何量取**（包围盒、落点线）在 jsdom 里量不出来（所有矩形全 0），
 * 那部分只有真机能验。所以这里不去断言坐标，只断言「命中了第几块」。
 */

import { Schema } from '@milkdown/kit/prose/model';
import { afterEach, describe, expect, it } from 'vitest';

import type { BlockHandleState } from '../../core/editor/types.js';
import { settleTyping } from '../../test/typing.js';
import { childStart, describeBlock, dropIndexFor, isNoopMove, withinGutter } from './block.js';
import { createMarkdownEditor, type MarkdownEditorInstance } from './setup.js';

// ---------------------------------------------------------------------------
// 纯算术那一半：一张最小 schema 就够
// ---------------------------------------------------------------------------

/**
 * 只含本文件要用到的节点类型。
 *
 * 刻意**不复用内核的 schema**：那些纯函数只吃「节点的名字、`nodeSize`、`attrs.level`」
 * 三样东西，自己造一张小表能让每个用例的文档一眼看清有几个块、各是多大。
 */
const schema = new Schema({
  nodes: {
    doc: { content: 'block+' },
    paragraph: { content: 'text*', group: 'block' },
    heading: { content: 'text*', group: 'block', attrs: { level: { default: 1 } } },
    horizontal_rule: { group: 'block' },
    text: {},
  },
});

/** 三个块：一段正文、一个二级标题、一条分隔线。 */
const doc = schema.node('doc', null, [
  schema.node('paragraph', null, [schema.text('一')]),
  schema.node('heading', { level: 2 }, [schema.text('标题')]),
  schema.node('horizontal_rule'),
]);

describe('describeBlock：一块能不能上移 / 能不能删 / 是什么格式', () => {
  it('第一块不能上移、能下移', () => {
    expect(describeBlock(doc, 0)).toMatchObject({ canMoveUp: false, canMoveDown: true });
  });

  it('最后一块能上移、不能下移', () => {
    expect(describeBlock(doc, 2)).toMatchObject({ canMoveUp: true, canMoveDown: false });
  });

  it('标题带上层级 —— 菜单据此把「二级标题」那一项显示成「已经是它」', () => {
    expect(describeBlock(doc, 1)).toMatchObject({ block: 'heading', headingLevel: 2, canTurnInto: true });
  });

  it('非标题的 headingLevel 是 null，而不是假装 1', () => {
    expect(describeBlock(doc, 0)?.headingLevel).toBeNull();
  });

  /**
   * 分隔线 / 表格 / 公式是**原子块**：它们不是「文本块的另一种形式」。
   * 假装成段落的话，菜单会点亮「正文」，而用户点下去什么都发生不了 ——
   * 更糟的是「转为正文」真做起来就是**把一张表格变成一段空文字**（硬约束 2）。
   */
  it('原子块如实给 null，并且不能「转为」', () => {
    expect(describeBlock(doc, 2)).toMatchObject({ block: null, headingLevel: null, canTurnInto: false });
  });

  it('只剩一块且有内容时仍可删（删掉等于清空它）', () => {
    const single = schema.node('doc', null, [schema.node('paragraph', null, [schema.text('一')])]);
    expect(describeBlock(single, 0)).toMatchObject({ canDelete: true, canMoveUp: false, canMoveDown: false });
  });

  it('只剩一块而且是空的时候不能删 —— 那等于什么都不做', () => {
    const empty = schema.node('doc', null, [schema.node('paragraph')]);
    expect(describeBlock(empty, 0)?.canDelete).toBe(false);
  });

  /**
   * 下标是内核从**指针位置**推出来的，而指针位置与文档之间隔着一次布局 ——
   * 理论上存在「算完之后文档又变了」的窗口。抛错会让一次鼠标移动把编辑器带崩，
   * 返回 null 只是手柄不显示。
   */
  it('下标越界 / 不是整数时返回 null，而不是抛错', () => {
    expect(describeBlock(doc, -1)).toBeNull();
    expect(describeBlock(doc, 3)).toBeNull();
    expect(describeBlock(doc, 1.5)).toBeNull();
    expect(describeBlock(doc, Number.NaN)).toBeNull();
  });
});

describe('childStart：第 index 块之前的文档位置', () => {
  it('累加各块的 nodeSize', () => {
    // 段落「一」= 1 文字 + 2 个边界 = 3；标题「标题」= 2 + 2 = 4
    expect(childStart(doc, 0)).toBe(0);
    expect(childStart(doc, 1)).toBe(3);
    expect(childStart(doc, 2)).toBe(7);
  });

  it('下标等于块数时给出文档内容的长度', () => {
    expect(childStart(doc, doc.childCount)).toBe(doc.content.size);
  });
});

describe('dropIndexFor：指针在块的下半就插到它后面', () => {
  it('上半 → 插到这一块之前', () => {
    expect(dropIndexFor(2, false, 5)).toBe(2);
  });

  it('下半 → 插到这一块之后', () => {
    expect(dropIndexFor(2, true, 5)).toBe(3);
  });

  it('夹在 0 … count 之间 —— 落点下标必须是合法插入位置', () => {
    expect(dropIndexFor(0, false, 3)).toBe(0);
    expect(dropIndexFor(2, true, 3)).toBe(3);
    expect(dropIndexFor(-5, false, 3)).toBe(0);
    expect(dropIndexFor(99, true, 3)).toBe(3);
  });
});

describe('isNoopMove：原地不动不该进撤销栈', () => {
  it('插到自己前面、插到自己后面都是原地不动', () => {
    expect(isNoopMove(2, 2)).toBe(true);
    expect(isNoopMove(2, 3)).toBe(true);
  });

  it('其它位置都是真移动', () => {
    expect(isNoopMove(2, 1)).toBe(false);
    expect(isNoopMove(2, 4)).toBe(false);
    expect(isNoopMove(0, 2)).toBe(false);
  });
});

describe('withinGutter：指针走到块左边那条留白里还算指着它', () => {
  const rect = { top: 100, left: 200, bottom: 140, right: 600, width: 400, height: 40 };

  it('块自己身上算', () => {
    expect(withinGutter(rect, 300, 120)).toBe(true);
  });

  it('左边那条留白里算 —— 手柄就摆在那儿', () => {
    expect(withinGutter(rect, 180, 120)).toBe(true);
  });

  it('留白之外不算（再往左就到别的栏里去了）', () => {
    expect(withinGutter(rect, 100, 120)).toBe(false);
  });

  it('块的上下之外不算 —— 否则手柄会赖在块上不走', () => {
    expect(withinGutter(rect, 300, 90)).toBe(false);
    expect(withinGutter(rect, 300, 150)).toBe(false);
  });

  it('右边之外不算', () => {
    expect(withinGutter(rect, 700, 120)).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// 悬停命中与块操作：真内核
// ---------------------------------------------------------------------------

let opened: MarkdownEditorInstance | null = null;

afterEach(async () => {
  await opened?.destroy();
  opened = null;
  document.body.innerHTML = '';
});

interface Harness {
  editor: MarkdownEditorInstance;
  /** 内核报给外壳的悬停状态序列。`null` = 指针不在任何一块上。 */
  hover: Array<BlockHandleState | null>;
  /** 内核报给外壳的内容变更。条数不变 = 这次操作没让文档变脏。 */
  reports: string[];
  /** ProseMirror 的 contenteditable。它的**直接子元素就是顶层块**。 */
  dom: HTMLElement;
}

async function open(source: string): Promise<Harness> {
  const root = document.createElement('div');
  document.body.appendChild(root);
  const hover: Array<BlockHandleState | null> = [];
  const reports: string[] = [];

  const editor = await createMarkdownEditor({
    root,
    defaultValue: source,
    onChange: (markdown) => reports.push(markdown),
    onBlockHandleChange: (state) => hover.push(state),
  });
  opened = editor;

  const dom = root.querySelector('.ProseMirror');
  if (!(dom instanceof HTMLElement)) throw new Error('没有找到 ProseMirror 视图');
  return { editor, hover, reports, dom };
}

/** 最近一次上报。 */
function latest(h: Harness): BlockHandleState | null {
  return h.hover[h.hover.length - 1] ?? null;
}

/**
 * 把指针「放」到第 index 块上。
 *
 * 直接派发 `mousemove` 到那一块的元素上，而不是模拟一串坐标 ——
 * 悬停命中的输入就是「指针下面那个元素」，这是它的真实语义。
 * `buttons` 用来模拟「按住鼠标」（拖动中）。
 */
async function hoverBlock(h: Harness, index: number, buttons = 0): Promise<void> {
  const block = h.dom.children[index];
  if (!(block instanceof HTMLElement)) throw new Error(`没有第 ${index} 块（共 ${h.dom.children.length} 块）`);
  block.dispatchEvent(new MouseEvent('mousemove', { bubbles: true, clientX: 300, clientY: 120, buttons }));
  await settleTyping();
}

/** 把指针放到编辑器自己的内边距上（不在任何一块上）。 */
async function hoverPadding(h: Harness, clientX = 300, clientY = 120): Promise<void> {
  h.dom.dispatchEvent(new MouseEvent('mousemove', { bubbles: true, clientX, clientY, buttons: 0 }));
  await settleTyping();
}

/** 把 markdown 拆成顶层块，用来断言顺序（比整串比对耐得住尾换行）。 */
function blocks(markdown: string): string[] {
  return markdown
    .split(/\n{2,}/)
    .map((part) => part.trim())
    .filter((part) => part !== '');
}

describe('悬停命中：手柄指着的那一块就是指针压着的那一块', () => {
  it('指针压在第 3 块上 → 报第 3 块，并且它「能上移」', async () => {
    const h = await open('一\n\n二\n\n三\n');
    await hoverBlock(h, 2);

    expect(latest(h)).toMatchObject({ block: 'paragraph', canMoveUp: true, canMoveDown: false });
  });

  it('换一块就报新的一块 —— 报的必须是**指针下**那一块，不是上一次那一块', async () => {
    const h = await open('一\n\n二\n\n三\n');
    await hoverBlock(h, 0);
    expect(latest(h)?.canMoveUp).toBe(false);

    await hoverBlock(h, 1);
    expect(latest(h)).toMatchObject({ canMoveUp: true, canMoveDown: true });
  });

  it('指针停在编辑器自己的内边距上 → 报 null（手柄要消失）', async () => {
    const h = await open('一\n\n二\n\n三\n');
    await hoverBlock(h, 1);
    expect(latest(h)).not.toBeNull();

    await hoverPadding(h);
    expect(latest(h)).toBeNull();
  });

  /**
   * ## 按住鼠标时必须冻住目标块
   *
   * 拖动重排期间 `mousemove` 照常发。不挡的话指针扫过哪一块、目标块就变成哪一块 ——
   * 用户拖的是第 1 块，松手却把第 3 块搬走了。判据是 `event.buttons !== 0`。
   */
  it('按住鼠标移动不换目标块 —— 否则拖动会搬走错误的那一块', async () => {
    const h = await open('一\n\n二\n\n三\n');
    await hoverBlock(h, 0);
    const before = h.hover.length;

    await hoverBlock(h, 2, 1);

    expect(h.hover.length, '按住期间一次都不该上报').toBe(before);
    expect(latest(h)?.canMoveUp, '目标块仍是第 1 块（第 1 块不能上移）').toBe(false);
  });

  /**
   * ## 指针走到左边那条留白里，目标块不能丢
   *
   * 手柄贴在块左边缘往左 26px，而那一段是编辑器自己的内边距 ——
   * 用户从正文移向手柄的中途必然穿过它。那时「指针压着哪个块」的答案是**没有块**，
   * 内核一报 null，外壳就把手柄卸载了，用户永远点不到它（`mouseenter` 都不会再发）。
   *
   * **这条用例只能验「兜底分支走通了」**：jsdom 没有布局引擎，包围盒全是 0，
   * 所以带子的实际宽度验不了（那要真机）。真正钉住宽度的是 `withinGutter` 那组单测。
   */
  it('指针落在左侧留白里时保住目标块（jsdom 里包围盒全 0，只验分支可达）', async () => {
    const h = await open('一\n\n二\n\n三\n');
    await hoverBlock(h, 1);
    expect(latest(h)?.canMoveUp).toBe(true);

    // 全 0 的包围盒往左扩 44px 之后是 [-44, 0] × [0, 0]
    await hoverPadding(h, -20, 0);

    expect(latest(h), '还指着原来那一块').toMatchObject({ canMoveUp: true, canMoveDown: true });
  });

  it('同一块上反复移动不重复上报（每次上报都会让外壳重渲染一遍）', async () => {
    const h = await open('一\n\n二\n');
    await hoverBlock(h, 0);
    const before = h.hover.length;

    await hoverBlock(h, 0);
    await hoverBlock(h, 0);

    expect(h.hover.length).toBe(before);
  });

  it('鼠标离开编辑器 → 报 null', async () => {
    const h = await open('一\n\n二\n');
    await hoverBlock(h, 0);

    h.dom.dispatchEvent(new MouseEvent('mouseleave', { bubbles: false }));
    await settleTyping();

    expect(latest(h)).toBeNull();
  });
});

describe('块操作：上移 / 下移 / 复制 / 删除', () => {
  it('下移把这一块搬到下一块后面', async () => {
    const h = await open('一\n\n二\n\n三\n');
    await hoverBlock(h, 0);

    h.editor.blockHandle.run('moveDown');
    await settleTyping();

    expect(blocks(h.editor.getMarkdown())).toEqual(['二', '一', '三']);
  });

  it('上移把这一块搬到上一块前面', async () => {
    const h = await open('一\n\n二\n\n三\n');
    await hoverBlock(h, 2);

    h.editor.blockHandle.run('moveUp');
    await settleTyping();

    expect(blocks(h.editor.getMarkdown())).toEqual(['一', '三', '二']);
  });

  /**
   * 往后搬时的位置算术最容易错：`tr.delete` 之后，删除点**之后**的位置整体左移了
   * 一个块高。不减这一下，被搬的块会一路落到文档末尾 —— 因为原来的 `insertAt`
   * 在新的坐标系里已经是文档长度了。
   */
  it('往后搬的位置算术是对的（不是少一格也不是多一格）', async () => {
    const h = await open('一\n\n二\n\n三\n\n四\n');
    await hoverBlock(h, 0);

    h.editor.blockHandle.run('moveDown');
    await settleTyping();
    expect(blocks(h.editor.getMarkdown())).toEqual(['二', '一', '三', '四']);

    // 「一」现在在第 2 块上。再往下搬一次，落点与删除点之间隔着「三」
    await hoverBlock(h, 1);
    h.editor.blockHandle.run('moveDown');
    await settleTyping();

    expect(blocks(h.editor.getMarkdown())).toEqual(['二', '三', '一', '四']);
  });

  it('复制一份插在它自己后面', async () => {
    const h = await open('一\n\n二\n');
    await hoverBlock(h, 0);

    h.editor.blockHandle.run('duplicate');
    await settleTyping();

    expect(blocks(h.editor.getMarkdown())).toEqual(['一', '一', '二']);
  });

  it('删除把这一块整块拿掉', async () => {
    const h = await open('一\n\n二\n\n三\n');
    await hoverBlock(h, 1);

    h.editor.blockHandle.run('delete');
    await settleTyping();

    expect(blocks(h.editor.getMarkdown())).toEqual(['一', '三']);
  });

  /**
   * 只剩一块时**清空**它而不是真删 —— schema 要求文档至少有一个块，
   * 真删掉的话这条事务会被 schema 拒绝（抛 `RangeError`），
   * 用户看到的是「点了没反应」。
   */
  it('只剩一块时「删除」是清空它，而不是抛错', async () => {
    const h = await open('一\n');
    await hoverBlock(h, 0);

    expect(() => h.editor.blockHandle.run('delete')).not.toThrow();
    await settleTyping();

    expect(h.editor.getMarkdown().trim()).toBe('');
  });

  it('第一块上移 / 最后一块下移是空操作，而且**不进变更上报**', async () => {
    const h = await open('一\n\n二\n');

    await hoverBlock(h, 0);
    const atFirst = h.reports.length;
    h.editor.blockHandle.run('moveUp');
    await settleTyping();
    expect(h.reports.length, '原地不动不该被当成一次改动').toBe(atFirst);

    await hoverBlock(h, 1);
    const atLast = h.reports.length;
    h.editor.blockHandle.run('moveDown');
    await settleTyping();
    expect(h.reports.length).toBe(atLast);
  });

  /**
   * ## 硬约束 7：这几个动作都不产生 `input` / `paste`
   *
   * 漏了 `USER_EDIT_META` 就是**静默丢数据**：用户点「下移」，界面显示成功，
   * 按 Ctrl+S 写回磁盘的是下移**之前**的文本，而界面一直显示「已保存」。
   */
  it('四个动作都会让文档变脏（否则 Ctrl+S 写回的是操作前的旧内容）', async () => {
    for (const [operation, source] of [
      ['moveDown', '一\n\n二\n\n三\n'],
      ['moveUp', '一\n\n二\n\n三\n'],
      ['duplicate', '一\n\n二\n'],
      ['delete', '一\n\n二\n'],
    ] as const) {
      const h = await open(source);
      await hoverBlock(h, 1);
      const before = h.reports.length;

      h.editor.blockHandle.run(operation);
      await settleTyping();

      expect(h.reports.length, `${operation} 没有上报变更`).toBeGreaterThan(before);
      await h.editor.destroy();
      document.body.innerHTML = '';
    }
  });

  it('没有目标块时安静忽略（手柄还没出现就点了菜单）', async () => {
    const h = await open('一\n\n二\n');
    const before = h.reports.length;

    h.editor.blockHandle.run('moveDown');
    h.editor.blockHandle.run('delete');
    await settleTyping();

    expect(h.reports.length).toBe(before);
    expect(blocks(h.editor.getMarkdown())).toEqual(['一', '二']);
  });
});

describe('转为：换一种块级格式', () => {
  it('正文 → 二级标题', async () => {
    const h = await open('一\n\n二\n');
    await hoverBlock(h, 0);

    h.editor.blockHandle.turnInto('heading', 2);
    await settleTyping();

    expect(blocks(h.editor.getMarkdown())).toEqual(['## 一', '二']);
  });

  it('二级标题 → 正文', async () => {
    const h = await open('## 一\n\n二\n');
    await hoverBlock(h, 0);

    h.editor.blockHandle.turnInto('paragraph');
    await settleTyping();

    expect(blocks(h.editor.getMarkdown())).toEqual(['一', '二']);
  });

  it('正文 → 无序列表', async () => {
    const h = await open('一\n\n二\n');
    await hoverBlock(h, 0);

    h.editor.blockHandle.turnInto('bulletList');
    await settleTyping();

    expect(h.editor.getMarkdown()).toContain('- 一');
  });

  /**
   * ## 「转为」不能是「取消」
   *
   * `toggleBlock` 对同格式是**取消**语义（标题降回正文、列表脱出容器），
   * 而「转为」的语义是「设成这个」。不挡这一下，用户点「转为 → 二级标题」
   * 会把一个已经是 H2 的标题变成正文 —— 菜单上明明写着「转为二级标题」，
   * 结果二级标题没了。外壳会把当前那一项显示成禁用，这里是第二道。
   */
  it('已经是这个格式时什么都不做（不能变成取消）', async () => {
    const h = await open('## 一\n\n二\n');
    await hoverBlock(h, 0);
    const before = h.reports.length;

    h.editor.blockHandle.turnInto('heading', 2);
    await settleTyping();

    expect(blocks(h.editor.getMarkdown())).toEqual(['## 一', '二']);
    expect(h.reports.length).toBe(before);
  });

  it('层级不同就是不同的格式：H2 转 H3 照做', async () => {
    const h = await open('## 一\n\n二\n');
    await hoverBlock(h, 0);

    h.editor.blockHandle.turnInto('heading', 3);
    await settleTyping();

    expect(blocks(h.editor.getMarkdown())).toEqual(['### 一', '二']);
  });

  it('原子块（分隔线）不能转 —— 菜单上不会出现这一组', async () => {
    const h = await open('---\n\n二\n');
    await hoverBlock(h, 0);

    expect(latest(h)).toMatchObject({ block: null, canTurnInto: false });

    const before = h.reports.length;
    h.editor.blockHandle.turnInto('paragraph');
    await settleTyping();

    expect(h.reports.length, '被挡住了，一次事务都不该派发').toBe(before);
    expect(h.editor.getMarkdown()).toContain('---');
  });
});

describe('落点（拖动）：jsdom 里量不出来，安静给 null', () => {
  /**
   * 落点用的是坐标命中（`posAtCoords`），而它内部要 `elementFromPoint` ——
   * jsdom 没有。**这里断言的是「环境缺失时安静降级」，不是「落点算得对」**：
   * 算得对只有真机能验。写下来是为了让下一个人别以为这一段在 jsdom 里被覆盖了。
   */
  it('`dropIndicatorAt` 返回 null，而不是抛错', async () => {
    const h = await open('一\n\n二\n');
    await hoverBlock(h, 0);

    expect(() => h.editor.blockHandle.dropIndicatorAt(300, 120)).not.toThrow();
    expect(h.editor.blockHandle.dropIndicatorAt(300, 120)).toBeNull();
  });

  it('`dropAt` 量不出落点时安静不动文档', async () => {
    const h = await open('一\n\n二\n');
    await hoverBlock(h, 0);
    const before = h.reports.length;

    h.editor.blockHandle.dropAt(300, 400);
    await settleTyping();

    expect(h.reports.length).toBe(before);
    expect(blocks(h.editor.getMarkdown())).toEqual(['一', '二']);
  });
});

describe('close：清掉目标块之后手柄就不该再作用', () => {
  it('`close()` 之后四个动作都安静忽略', async () => {
    const h = await open('一\n\n二\n');
    await hoverBlock(h, 0);

    h.editor.blockHandle.close();
    const before = h.reports.length;

    h.editor.blockHandle.run('moveDown');
    h.editor.blockHandle.run('duplicate');
    h.editor.blockHandle.run('delete');
    h.editor.blockHandle.turnInto('heading', 1);
    await settleTyping();

    expect(h.reports.length).toBe(before);
    expect(blocks(h.editor.getMarkdown())).toEqual(['一', '二']);
  });
});
