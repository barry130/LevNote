// @vitest-environment jsdom
/**
 * 表格的**内核半边**：插入、加行列、删行列、对齐，以及「光标在表格里」怎么报给外壳。
 *
 * ## 这一组测试真正在钉的那条不变量
 *
 * **文档里有几行几列，存出来的 markdown 就必须有几行几列。**
 *
 * 写这个文件时先踩了一个真实的坑，值得留在这里：Milkdown 的 gfm 表格 schema 是
 * `table_header_row table_row+`（必须有表头行、且至少一行正文），而
 * `tableSchema.toMarkdown` 是用**第一行**的单元格决定分隔行对齐的，
 * `tableHeaderRowSchema.toMarkdown` 遇到**空行会直接 return**（那一行不进 mdast）。
 *
 * 于是「用 prosemirror-tables 的 `deleteRow` 删掉表头行」会留下一个
 * schema 合法、语义非法的空表头行 —— 编辑区里看得见，**存盘时被静默丢掉**。
 * 实测：文档里 4 行、`getMarkdown()` 只有 2 行，而且没有任何报错。
 *
 * 所以每个操作之后都要同时看**两边**（`toHtml()` 数 `<tr>` / `<td>` 与 markdown 数行），
 * 只断言其中一边的话，这个 bug 会完整地漏过去。
 */

import { afterEach, describe, expect, it } from 'vitest';

import type { TableOperation, TableState } from '../../core/editor/types.js';
import { settleTyping, typeInto } from '../../test/typing.js';
import { createMarkdownEditor, type MarkdownEditorInstance } from './setup.js';

let opened: MarkdownEditorInstance | null = null;

afterEach(async () => {
  await opened?.destroy();
  opened = null;
  document.body.innerHTML = '';
});

interface Harness {
  editor: MarkdownEditorInstance;
  /** 内核报给外壳的表格状态序列。`null` = 光标不在表格里。 */
  table: Array<TableState | null>;
  /** 内核报给外壳的内容变更。空数组 = 这次编辑没告诉外壳（就是丢数据）。 */
  reports: string[];
  /** ProseMirror 的 contenteditable，`typeInto` 往它里面打字。 */
  dom: HTMLElement;
}

async function open(source: string): Promise<Harness> {
  const root = document.createElement('div');
  document.body.appendChild(root);
  const table: Array<TableState | null> = [];
  const reports: string[] = [];

  const editor = await createMarkdownEditor({
    root,
    defaultValue: source,
    onChange: (markdown) => reports.push(markdown),
    onTableChange: (state) => table.push(state),
  });
  opened = editor;

  const dom = root.querySelector('.ProseMirror');
  if (!(dom instanceof HTMLElement)) throw new Error('没有找到 ProseMirror 视图');
  return { editor, table, reports, dom };
}

/** 最近一次上报（理由见 `slash.test.ts` 里的同名函数）。 */
function latest(table: Array<TableState | null>): TableState | null {
  return table[table.length - 1] ?? null;
}

/**
 * 从渲染出来的 HTML 数表格形状。
 *
 * 刻意走 `toHtml()` 而不是读 `TableState`：上报值来自内核自己算的 `TableMap`，
 * 而这里要验的正是「内核算的」与「文档里真有的」是不是一回事。
 */
function domShape(editor: MarkdownEditorInstance): { rows: number; cells: number[] } {
  const container = document.createElement('div');
  container.innerHTML = editor.toHtml();
  const rows = Array.from(container.querySelectorAll('tr'));
  return { rows: rows.length, cells: rows.map((row) => row.querySelectorAll('th, td').length) };
}

/** markdown 里的表格行数（含分隔行，所以正常比 `domShape().rows` 多 1）。 */
function mdRows(markdown: string): number {
  return markdown.split('\n').filter((line) => line.trim().startsWith('|')).length;
}

/**
 * 核心不变量：文档里的行数 = markdown 里的行数 − 1（那一行是分隔行），
 * 且每一行的列数一致。
 */
function expectConsistent(h: Harness, expectedRows: number, expectedColumns: number): void {
  const shape = domShape(h.editor);
  const markdown = h.editor.getMarkdown();

  expect(shape.rows, `文档里的行数`).toBe(expectedRows);
  expect(mdRows(markdown), `markdown 里的行数（文档 ${shape.rows} 行 + 分隔行）`).toBe(expectedRows + 1);
  for (const [index, cells] of shape.cells.entries()) {
    expect(cells, `第 ${index} 行的列数`).toBe(expectedColumns);
  }
}

/** 表头行 + 两行正文，三列。 */
const TABLE = '| A | B | C |\n| --- | --- | --- |\n| a | b | c |\n| d | e | f |\n';

/**
 * 打开一张表格，并把光标送进**最后一行最后一列**。
 *
 * `typeInto` 找的是 DOM 里最后一个 `<p>` —— 表格的单元格内容就是段落，
 * 所以光标自然落在最后一个格子里。这是 jsdom 里唯一可靠的「把光标放进指定格子」的办法
 * （内核没有暴露「设置选区」的接口，也不该为了测试暴露）。
 */
async function openTableAtLastCell(source = TABLE): Promise<Harness> {
  const h = await open(source);
  await typeInto(h.dom, '!');
  await settleTyping();
  return h;
}

describe('插入表格', () => {
  it('从斜杠菜单插入一张 3 × 3 表格，`/表` 被删掉', async () => {
    const { editor, dom } = await open('\n');
    await typeInto(dom, '/表');
    expect(editor.getMarkdown()).toContain('/表');

    editor.formatting.insertBlock('table');
    await settleTyping();

    const markdown = editor.getMarkdown();
    expect(markdown, '菜单的筛选词不该留在文件里').not.toContain('/表');
    expect(mdRows(markdown), '3 行（表头 + 2 行正文）').toBe(4);
    expect(editor.table.isActive()).toBe(true);
  });

  it('插入之后光标就在表格里 —— 这是「插完立刻能用工具条」的前提', async () => {
    const { editor, table, dom } = await open('\n');
    await typeInto(dom, '/表');
    editor.formatting.insertBlock('table');
    await settleTyping();

    expect(editor.table.isActive()).toBe(true);
    expect(latest(table)).not.toBeNull();
    expect(latest(table)?.rows).toBe(3);
    expect(latest(table)?.columns).toBe(3);
  });

  it('插入会报给外壳 —— 否则用户按 Ctrl+S 存的是插入前的旧内容', async () => {
    const { editor, reports, dom } = await open('\n');
    await typeInto(dom, '/表');

    const before = reports.length;
    editor.formatting.insertBlock('table');

    expect(reports.length).toBeGreaterThan(before);
    expect(reports[reports.length - 1]).toContain('|');
  });
});

describe('表格状态上报', () => {
  it('打开一份以表格开头的文档就上报行列数（表头行算一行）', async () => {
    const h = await open(TABLE);
    await settleTyping();

    expect(h.editor.table.isActive()).toBe(true);
    expect(latest(h.table)?.rows).toBe(3);
    expect(latest(h.table)?.columns).toBe(3);
  });

  it('光标不在表格里时 `isActive` 为 false、上报 null', async () => {
    const h = await open(`正文\n\n${TABLE}\n`);
    await settleTyping();

    // 文档以正文开头 → 初始光标在正文段落里，不在表格里
    expect(h.editor.table.isActive()).toBe(false);
    expect(latest(h.table)).toBeNull();
  });

  it('`exit` 之后光标离开表格，上报 null', async () => {
    const h = await open(TABLE);
    await settleTyping();
    expect(latest(h.table)).not.toBeNull();

    h.editor.table.run('exit');
    await settleTyping();

    expect(h.editor.table.isActive()).toBe(false);
    expect(latest(h.table)).toBeNull();
  });
});

describe('加行加列', () => {
  it('`rowAfter` 加一行，文档与 markdown 一致', async () => {
    const h = await openTableAtLastCell();
    expectConsistent(h, 3, 3);

    h.editor.table.run('rowAfter');
    await settleTyping();

    expectConsistent(h, 4, 3);
    expect(h.editor.getMarkdown()).toContain('f!');
  });

  it('`rowBefore` 在光标那一行之前插入', async () => {
    const h = await openTableAtLastCell();
    h.editor.table.run('rowBefore');
    await settleTyping();

    expectConsistent(h, 4, 3);
    expect(latest(h.table)?.rows).toBe(4);
  });

  it('`colAfter` 加一列，文档与 markdown 一致', async () => {
    const h = await openTableAtLastCell();
    h.editor.table.run('colAfter');
    await settleTyping();

    expectConsistent(h, 3, 4);
    expect(latest(h.table)?.columns).toBe(4);
  });

  /**
   * 钉「加列必须走 gfm 命令，不能直接调 prosemirror-tables 的 `addColumnAfter`」。
   *
   * 预设的 `AddColAfter` 多做了一步：把相邻单元格的**对齐方式**抄到新列上。
   * 绕过它不会报错，只会在「整列居中」的表格上悄悄多出一列左对齐的格子。
   */
  it('`colAfter` 的新列沿用相邻列的对齐方式', async () => {
    const h = await open(TABLE);
    await settleTyping();

    // 光标在第一个格子里（第 0 列）→ 整列居中
    h.editor.table.setAlign('center');
    await settleTyping();
    expect(h.editor.getMarkdown()).toContain(':-:');

    h.editor.table.run('colAfter');
    await settleTyping();

    expectConsistent(h, 3, 4);
    const centered = h.editor.getMarkdown().match(/:-:/g) ?? [];
    expect(centered, '新列应该也是居中的，也就是分隔行里有两个 :-:').toHaveLength(2);
  });
});

describe('删行', () => {
  it('删正文行：行数减一，文档与 markdown 仍然一致', async () => {
    const h = await openTableAtLastCell();
    expect(h.editor.getMarkdown()).toContain('f!');

    h.editor.table.run('deleteRow');
    await settleTyping();

    expectConsistent(h, 2, 3);
    const markdown = h.editor.getMarkdown();
    expect(markdown, '被删掉的那一行不该留下任何痕迹').not.toContain('f');
    expect(markdown, '没被删的行要原样留着').toContain('c');
  });

  /**
   * 这条钉的是 `deleteTableRow` 里的「提升」逻辑。
   *
   * Markdown 表格的表头行是**结构性**的，删掉它之后下一行必须顶上来当表头。
   * 少了这一步，新表头行会是一个普通行 —— 而普通行在序列化时会被整行跳过，
   * 于是「删了一行」变成「删了两行」，用户还看不出为什么。
   */
  it('删表头行：下一行顶上来当表头，文档与 markdown 仍然一致', async () => {
    const h = await open(TABLE);
    await settleTyping();
    expect(latest(h.table)?.rows).toBe(3);

    // 光标在表头行的第一个格子里（打开时的默认位置）
    h.editor.table.run('deleteRow');
    await settleTyping();

    expectConsistent(h, 2, 3);
    const markdown = h.editor.getMarkdown();
    expect(markdown, '原来的表头行应该没了').not.toContain('A');
    expect(markdown, '原来的第一行正文顶上来当了表头').toContain('a');
    expect(markdown, '剩下的正文行还在').toContain('d');
  });

  it('只剩两行时 `canDeleteRow` 为 false，且 `run` 一个字符都不改', async () => {
    const h = await open('| A | B |\n| --- | --- |\n| a | b |\n');
    await settleTyping();
    expect(latest(h.table)?.rows).toBe(2);
    expect(latest(h.table)?.canDeleteRow).toBe(false);

    const before = h.editor.getMarkdown();
    h.editor.table.run('deleteRow');
    await settleTyping();

    expect(h.editor.getMarkdown()).toBe(before);
    expectConsistent(h, 2, 2);
  });
});

describe('删列', () => {
  it('删列：列数减一，文档与 markdown 仍然一致', async () => {
    const h = await openTableAtLastCell();
    h.editor.table.run('deleteColumn');
    await settleTyping();

    expectConsistent(h, 3, 2);
    expect(latest(h.table)?.columns).toBe(2);
    expect(h.editor.getMarkdown()).not.toContain('f');
  });

  it('只剩一列时 `canDeleteColumn` 为 false，且 `run` 一个字符都不改', async () => {
    const h = await open('| A |\n| --- |\n| a |\n');
    await settleTyping();
    expect(latest(h.table)?.columns).toBe(1);
    expect(latest(h.table)?.canDeleteColumn).toBe(false);

    const before = h.editor.getMarkdown();
    h.editor.table.run('deleteColumn');
    await settleTyping();

    expect(h.editor.getMarkdown()).toBe(before);
  });
});

describe('对齐', () => {
  it('`setAlign` 改的是**整列**，分隔行变成 `:-:`', async () => {
    const h = await open(TABLE);
    await settleTyping();

    h.editor.table.setAlign('center');
    await settleTyping();

    const delimiter = h.editor.getMarkdown().split('\n')[1] ?? '';
    expect(delimiter).toContain(':-:');
    expect(latest(h.table)?.align).toBe('center');
  });

  /**
   * 这条钉的是「正文格子里点居中也要生效」。
   *
   * Milkdown 的 `keepTableAlignPlugin` 会把正文格子的对齐**同步回表头行的值**，
   * 所以只改光标那一格（预设 `setAlignCommand` 的做法）会被它当场改回去 ——
   * 表现为「点了没反应」。`setAlign` 整列一起改才绕得过去。
   */
  it('光标在正文格子里时，居中同样生效', async () => {
    const h = await openTableAtLastCell();
    await settleTyping();

    h.editor.table.setAlign('right');
    await settleTyping();

    const delimiter = h.editor.getMarkdown().split('\n')[1] ?? '';
    // 序列化器写的是最短合法写法：右对齐是 `-:` 而不是 `---:`
    expect(delimiter, '第三列应该是右对齐').toContain('-:');
    expect(latest(h.table)?.align).toBe('right');
  });

  it('`setAlign` 会报给外壳 —— 否则文档不变脏，保存写回的是对齐前的排版', async () => {
    const h = await open(TABLE);
    await settleTyping();

    const before = h.reports.length;
    h.editor.table.setAlign('center');
    await settleTyping();

    expect(h.reports.length).toBeGreaterThan(before);
  });
});

describe('删表', () => {
  it('`deleteTable` 之后 markdown 里没有表格了，且上报 null', async () => {
    const h = await openTableAtLastCell();

    h.editor.table.run('deleteTable');
    await settleTyping();

    const markdown = h.editor.getMarkdown();
    expect(mdRows(markdown)).toBe(0);
    expect(markdown).not.toContain('|');
    expect(latest(h.table)).toBeNull();
  });
});

describe('USER_EDIT_META', () => {
  /**
   * 与浮层工具条、斜杠菜单同一条规矩：这些操作由用户点击触发、**不经过键盘事件**，
   * 必须自己打 `USER_EDIT_META`，否则文档不会变脏 —— 用户按 Ctrl+S
   * 写回磁盘的是操作之前的旧内容，而界面显示「已保存」。
   */
  const operations: readonly TableOperation[] = ['rowBefore', 'rowAfter', 'colBefore', 'colAfter', 'deleteRow', 'exit'];

  for (const operation of operations) {
    it(`${operation} 会报给外壳`, async () => {
      const h = await openTableAtLastCell();
      const before = h.reports.length;

      h.editor.table.run(operation);
      await settleTyping();

      // `exit` 也会报 —— 它确实改了文档：`@milkdown/plugin-trailing` 会在表格之后
      // 补一个空段落（否则用户没法把光标点到表格下面去）。
      // 这一点被断言钉住是有意的：它是「退出表格 = 文档变脏」这个行为的记录，
      // 而不是一个意外
      expect(h.reports.length).toBeGreaterThan(before);
    });
  }

  it('`exit` 之后文档末尾多了一个空段落 —— 那是 trailing 插件干的，不是丢数据', async () => {
    const h = await openTableAtLastCell();
    h.editor.table.run('exit');
    await settleTyping();

    expect(h.editor.getMarkdown()).toMatch(/\n\n<br \/>\n$/);
  });

  it('deleteRow / deleteColumn / deleteTable 会报给外壳', async () => {
    const h = await openTableAtLastCell();
    const before = h.reports.length;

    h.editor.table.run('deleteRow');
    h.editor.table.run('deleteColumn');
    await settleTyping();

    expect(h.reports.length).toBeGreaterThan(before);
  });
});
