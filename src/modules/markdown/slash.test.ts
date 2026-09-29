// @vitest-environment jsdom
/**
 * 斜杠菜单的**内核半边** —— 触发判定，以及「先删掉 `/query` 再套块格式」。
 *
 * ## 为什么必须真的打字
 *
 * 斜杠触发是一个**逐键推进**的状态机：`/` 之后每多一个字，筛选词就长一个字，
 * 中间还有「带空格就不触发」这种转折。直接构造一个 `/表` 的文档去测，
 * 会把「用户敲到第几个字时菜单出现/消失」这一整段行为全部跳过 ——
 * 而那正是这个功能唯一容易写错的地方。
 *
 * 所以这里用 `typeInto` 走真实输入链路（改 DOM → MutationObserver →
 * `input` 事件 → ProseMirror 读变更），理由与做法见 `src/test/typing.ts`。
 *
 * ## 断言分两层
 *
 * - **上报**：内核告诉外壳「现在该不该有菜单、筛选词是什么」；
 * - **产物**：`insertBlock` 之后序列化出来的 Markdown。
 *
 * 第二层里有一条不能省：`/表` 必须**真的消失**。少了它，`insertBlock` 会变成
 * 「在 `/表` 后面插了一个标题」—— 菜单看着是能用的，文件里却多出一行垃圾。
 */

import { afterEach, describe, expect, it } from 'vitest';

import type { InsertableBlock, SlashState } from '../../core/editor/types.js';
import { settleTyping, typeInto } from '../../test/typing.js';
import { createMarkdownEditor, type MarkdownEditorInstance } from './setup.js';

const FIND_BASE = { caseSensitive: true, wholeWord: false, regex: false } as const;

let opened: MarkdownEditorInstance | null = null;

afterEach(async () => {
  await opened?.destroy();
  opened = null;
  document.body.innerHTML = '';
});

interface Harness {
  editor: MarkdownEditorInstance;
  /** 内核报给外壳的斜杠状态序列。`null` = 菜单该收起来。 */
  slash: Array<string | null>;
  /** 内核报给外壳的内容变更。空数组 = 这次编辑没告诉外壳（就是丢数据）。 */
  reports: string[];
  /** ProseMirror 的 contenteditable，`typeInto` 往它里面打字。 */
  dom: HTMLElement;
}

async function open(source: string): Promise<Harness> {
  const root = document.createElement('div');
  document.body.appendChild(root);
  const slash: Array<string | null> = [];
  const reports: string[] = [];

  const editor = await createMarkdownEditor({
    root,
    defaultValue: source,
    onChange: (markdown) => reports.push(markdown),
    onSlashChange: (state: SlashState | null) => slash.push(state === null ? null : state.query),
  });
  opened = editor;

  const dom = root.querySelector('.ProseMirror');
  if (!(dom instanceof HTMLElement)) throw new Error('没有找到 ProseMirror 视图');
  return { editor, slash, reports, dom };
}

/**
 * 最近一次上报。
 *
 * 用「最近一次」而不是整条序列：内核创建时会先报一次 `null`（还没有选区），
 * 之后每敲一个字各报一次。整条序列的中间值会被 rAF 合并的时机影响，
 * 钉死它只会让测试变脆；这里只钉**每个动作之后的状态**。
 */
function latest(slash: Array<string | null>): string | null {
  return slash[slash.length - 1] ?? null;
}

describe('斜杠触发条件', () => {
  it('空段落里打 `/` 就上报，筛选词跟着打字一个字一个字地长', async () => {
    const { editor, slash, dom } = await open('\n');

    await typeInto(dom, '/');
    // 刚打斜杠：筛选词是空串，菜单应该显示全部项
    expect(latest(slash)).toBe('');

    await typeInto(dom, '表');
    expect(latest(slash)).toBe('表');

    await typeInto(dom, '题');
    expect(latest(slash)).toBe('表题');

    // 顺带确认这一段确实进了文档 —— 不然上面只是在断言一个空状态
    expect(editor.getMarkdown()).toContain('/表题');
  });

  it('段落里已经有正文时，斜杠只是普通字符', async () => {
    const { editor, slash, dom } = await open('正文 abc 结尾\n');

    await typeInto(dom, '/表');

    expect(latest(slash)).toBeNull();
    expect(editor.getMarkdown()).toContain('正文 abc 结尾/表');
  });

  /**
   * 这条钉的是 `SLASH_TRIGGER` 里那个 `\S*`。
   *
   * 允许筛选词带空格的话，用户写「一半 / 另一半」这种正文时会莫名弹出菜单。
   * 菜单在打字途中自己冒出来，比少一个功能烦人得多。
   */
  it('筛选词里出现空格就不再触发', async () => {
    const { slash, dom } = await open('\n');

    await typeInto(dom, '/一半');
    expect(latest(slash)).toBe('一半');

    await typeInto(dom, ' ');
    expect(latest(slash)).toBeNull();

    await typeInto(dom, '另一半');
    expect(latest(slash)).toBeNull();
  });

  /**
   * 同一条 `/表` 打头的内容，**光标在段首**时不触发、**挪到末尾**后触发。
   *
   * 这样写才真的验到「光标必须在文本末尾」那一条：只测前一半的话，
   * 「打开 `/表` 没有菜单」这个结果在「压根没实现上报」时也成立（假绿）。
   */
  it('光标不在文本末尾时不触发', async () => {
    const { slash, dom } = await open('/表\n');

    // 刚打开时光标在段首
    expect(latest(slash)).toBeNull();

    // 往后打一个字，光标自然落在末尾 —— 这时才该触发
    await typeInto(dom, '甲');
    expect(latest(slash)).toBe('表甲');
  });

  it('选中文字时不触发 —— 用户是在选东西，不是在打命令', async () => {
    const { editor, slash, dom } = await open('\n');

    await typeInto(dom, '/表');
    expect(latest(slash)).toBe('表');

    // 用查找能力造出一个真实的选中范围（不是塌缩光标）
    expect(editor.search.find({ ...FIND_BASE, query: '表' })).toBe(1);
    editor.search.reveal(0);
    // 上报是**下一帧**的事（合并到 rAF 是为了拖选时不逐次强制布局），
    // 不等这一帧就会断言到上一个状态 —— 那是假绿
    await settleTyping();

    expect(latest(slash)).toBeNull();
  });

  it('标题里打斜杠不触发 —— 菜单只属于段落', async () => {
    const { slash, dom } = await open('\n');

    // 先用同一套能力把空段落变成标题，光标留在里面
    opened?.formatting.insertBlock('heading1');
    await typeInto(dom, '/表');

    expect(latest(slash)).toBeNull();
  });
});

describe('插入块', () => {
  /**
   * 九种块各插一次。
   *
   * 期望值写的是**实际序列化产物**而不是「应该长什么样」。两个地方与直觉不同，
   * 都是实测出来的，不是猜的：
   *
   * - 分隔线写成 `***`（remark 的写法，不是 `---`）；
   * - 空段落套进引用 / 列表之后会多出一个 `<br />`。那是 Milkdown
   *   `remarkPreserveEmptyLinePlugin` 的**空行占位符**：mdast 里空段落序列化出来
   *   什么都没有，没有占位符的话「一个空引用」存盘再打开就消失了。
   *   它只影响「插入了但一个字没打就保存」，一旦打字（`> 甲`）占位符就没了。
   */
  const cases: ReadonlyArray<[InsertableBlock, RegExp]> = [
    ['paragraph', /^\s*$/],
    ['heading1', /^#\s*$/m],
    ['heading2', /^##\s*$/m],
    ['heading3', /^###\s*$/m],
    ['blockquote', /^>/m],
    // 序列化器把无序列表写成 `* ` 而不是 `- `（两者等价，remark 选了前者）
    ['bulletList', /^[*+-] /m],
    ['orderedList', /^1\. /m],
    ['codeBlock', /^```/m],
    ['hr', /^(?:\*{3,}|-{3,}|_{3,})\s*$/m],
  ];

  for (const [block, expected] of cases) {
    it(`${block}：` + '`/query` 被删掉，并且真的套上了块格式', async () => {
      const { editor, dom } = await open('\n');
      await typeInto(dom, '/表');
      expect(editor.getMarkdown()).toContain('/表');

      editor.formatting.insertBlock(block);

      const markdown = editor.getMarkdown();
      // 这一条是最要紧的：菜单的筛选词不该留在文件里
      expect(markdown, `${block} 没删掉 /表`).not.toContain('/表');
      expect(markdown, `${block}`).toMatch(expected);
    });
  }

  it('插入之后菜单自己收起来', async () => {
    const { editor, slash, dom } = await open('\n');
    await typeInto(dom, '/表');
    expect(latest(slash)).toBe('表');

    editor.formatting.insertBlock('heading2');
    await settleTyping();

    // `/表` 没了 → 触发条件不再成立 → 内核报 null → 外壳收起菜单。
    // 少了这一步，用户选完菜单项之后它会一直挂在屏幕上
    expect(latest(slash)).toBeNull();
  });

  /**
   * 这条钉的是 `slashRange()` 返回 `null` 的那条分支。
   *
   * 不在斜杠触发状态时（这里靠选中一段文字），`insertBlock` 必须**什么都不删** ——
   * 它退化成 `toggleBlock`。写错的话，用户选中一段文字、从命令面板插入一个块，
   * 选区里的文字会凭空少掉几个字，而且不会有任何报错。
   */
  it('不在触发状态时一个字符都不删，等价于 toggleBlock', async () => {
    const { editor } = await open('正文 abc 结尾\n');
    expect(editor.search.find({ ...FIND_BASE, query: 'abc' })).toBe(1);
    editor.search.reveal(0);

    editor.formatting.insertBlock('heading1');

    const markdown = editor.getMarkdown();
    expect(markdown).toContain('正文 abc 结尾');
    expect(markdown).toMatch(/^# /m);
  });

  /**
   * 与浮层工具条同一条规矩：`insertBlock` 由用户动作触发、不经过键盘事件，
   * 必须自己打 `USER_EDIT_META`，否则文档不会变脏，保存写回的是插入**之前**的内容。
   */
  it('插入块会报给外壳 —— 否则用户按 Ctrl+S 存的是插入前的旧内容', async () => {
    const { editor, reports, dom } = await open('\n');
    await typeInto(dom, '/表');

    const before = reports.length;
    editor.formatting.insertBlock('blockquote');

    expect(reports.length).toBeGreaterThan(before);
    expect(reports[reports.length - 1]).toContain('>');
  });
});
