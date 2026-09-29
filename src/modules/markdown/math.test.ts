// @vitest-environment jsdom
/**
 * 公式的**内核半边**：节点往返、两条输入规则、NodeView 的渲染与原位编辑。
 *
 * ## 这一组测试真正在钉的四条不变量
 *
 * 1. **打字时的转换与重新打开文件时的解析是同一次判断。** 输入规则只认「光标前
 *    刚打了一个 `$`」，到底是不是公式交给 `core/math.ts`（也就是文档解析器）。
 *    两者一旦分叉，表现是「打字时是公式、保存后变成文字」。
 * 2. **NodeView 画出来的 DOM 绝不能进文档。** 我们往 `.qm-math` 里塞的是 KaTeX
 *    的一整棵 DOM，而文档里该留的只有 TeX 源码。少了 `ignoreMutation`，
 *    存出来的 Markdown 里会多出一坨 KaTeX 的 HTML —— 而界面上一切正常。
 * 3. **原位编辑必须让文档变脏。** 输入框里的按键被 `stopEvent` 挡住了，
 *    ProseMirror 的 DOM 观察者与 `beforeinput` 监听都收不到。不显式声明
 *    `USER_EDIT_META`，用户按 Ctrl+S 写回磁盘的是**改之前**的公式。
 * 4. **存不住的值必须被拦住。** 空的行内公式序列化之后会变成块级公式
 *    （把整段文字劈成两半）—— 拦下来并说明原因，比「存进去再说」好得多。
 */

import { afterEach, describe, expect, it } from 'vitest';

import { MATH_BLOCK, MATH_INLINE, TEX_DATA_ATTR } from '../../core/math.js';
import { settleTyping, typeInto } from '../../test/typing.js';
import { MATH_BLOCK as MODULE_BLOCK, MATH_INLINE as MODULE_INLINE } from './math.js';
import { createMarkdownEditor, type MarkdownEditorInstance } from './setup.js';

let opened: MarkdownEditorInstance | null = null;

afterEach(async () => {
  await opened?.destroy();
  opened = null;
  document.body.innerHTML = '';
});

interface Harness {
  editor: MarkdownEditorInstance;
  root: HTMLDivElement;
  /** ProseMirror 的 contenteditable，`typeInto` 往它里面打字。 */
  dom: HTMLElement;
  /** 内核报给外壳的内容变更。空数组 = 这次编辑没告诉外壳（就是丢数据）。 */
  reports: string[];
}

async function open(source: string): Promise<Harness> {
  const root = document.createElement('div');
  document.body.appendChild(root);
  const reports: string[] = [];

  const editor = await createMarkdownEditor({
    root,
    defaultValue: source,
    onChange: (markdown) => reports.push(markdown),
  });
  opened = editor;

  const dom = root.querySelector('.ProseMirror');
  if (!(dom instanceof HTMLElement)) throw new Error('没有找到 ProseMirror 视图');
  return { editor, root, dom, reports };
}

/**
 * 等一个条件成立。
 *
 * 公式的渲染是**异步**的（KaTeX 是动态加载的，见 `core/katex.ts`），
 * 而 jsdom 里没有「渲染完成」这个事件可等。所以这里轮询 DOM。
 * 不用固定次数的 `setTimeout`：那要么不够（偶发失败）、要么每次都白等。
 */
async function waitFor(condition: () => boolean, label: string): Promise<void> {
  for (let attempt = 0; attempt < 100; attempt += 1) {
    if (condition()) return;
    await new Promise((resolve) => setTimeout(resolve, 5));
  }
  throw new Error(`等不到：${label}`);
}

/** 等所有公式渲染出结果（成功或失败）。 */
async function waitForMath(harness: Harness): Promise<void> {
  await waitFor(
    () =>
      harness.dom.querySelectorAll('.qm-math').length > 0 &&
      harness.dom.querySelectorAll('.qm-math.is-loading').length === 0,
    '公式渲染完成',
  );
}

function element(harness: Harness, selector: string): HTMLElement {
  const found = harness.dom.querySelector(selector);
  if (!(found instanceof HTMLElement)) throw new Error(`找不到 ${selector}`);
  return found;
}

/** 点一下公式（NodeView 挂在 `mousedown` 上）。 */
function clickMath(harness: Harness, selector: string): void {
  element(harness, selector).dispatchEvent(
    new MouseEvent('mousedown', { bubbles: true, cancelable: true }),
  );
}

function pressKey(target: HTMLElement, key: string, mods: { ctrlKey?: boolean } = {}): void {
  target.dispatchEvent(
    new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true, ...mods }),
  );
}

/** 在编辑中的输入框里改值并确认。 */
function editTo(harness: Harness, tex: string, key: 'Enter' | 'Escape' = 'Enter'): void {
  const input = element(harness, '.qm-math-input') as HTMLInputElement | HTMLTextAreaElement;
  input.value = tex;
  pressKey(input, key, tex.includes('\n') ? { ctrlKey: true } : {});
}

// ---------------------------------------------------------------------------
// 节点名与 DOM 契约
// ---------------------------------------------------------------------------

describe('节点名就是 DOM 契约值', () => {
  /**
   * 内核**故意**把 ProseMirror 节点名取成与 `data-type` 一样的字符串，
   * 于是「节点名 ↔ 契约值」是恒等映射，不需要一张可能写错的翻译表。
   *
   * 导出侧（`core/export/inline-html.ts`）靠 `[data-tex]` 找公式、
   * 靠 `data-type` 判断行内还是块级。改名不会有任何报错 —— 只会让导出文件里
   * 的公式变成一坨 `$…$` 源码。所以这里同时钉住重导出与真实 DOM。
   */
  it('模块重导出的两个常量与 core 的一致', () => {
    expect(MODULE_INLINE).toBe(MATH_INLINE);
    expect(MODULE_BLOCK).toBe(MATH_BLOCK);
  });

  it('渲染出来的 DOM 上带着契约属性', async () => {
    const harness = await open('$x^2$\n\n$$\ny\n$$\n');
    const html = harness.editor.toHtml();

    expect(html).toContain(`data-type="${MATH_INLINE}"`);
    expect(html).toContain(`data-type="${MATH_BLOCK}"`);
    // 值在 `data-tex` 上，导出侧据此渲染
    expect(html).toContain(`${TEX_DATA_ATTR}="x^2"`);
    expect(html).toContain(`${TEX_DATA_ATTR}="y"`);
  });
});

// ---------------------------------------------------------------------------
// 往返
// ---------------------------------------------------------------------------

describe('打开再存回，公式不变', () => {
  it.each([
    ['$x^2$\n'],
    ['前文 $E=mc^2$ 后文\n'],
    ['$$\n\\frac{1}{2}\n$$\n'],
    ['a $x$ b\n\n$$\ny\n$$\n'],
    ['$\\sum_{i=1}^{n} i$\n'],
  ])('%j 原样存回', async (source) => {
    const harness = await open(source);
    expect(harness.editor.getMarkdown()).toBe(source);
  });

  it('转义过的美元符号是字面量，不会变成公式', async () => {
    const harness = await open('价格 \\$5 到 \\$10\n');

    expect(harness.dom.querySelector('.qm-math')).toBeNull();
    expect(harness.editor.getMarkdown()).toContain('\\$');
  });
});

// ---------------------------------------------------------------------------
// 输入规则：行内
// ---------------------------------------------------------------------------

describe('行内公式的输入规则', () => {
  it('敲下收尾的 `$` 就变成公式节点', async () => {
    const harness = await open('\n');
    await typeInto(harness.dom, 'a $x^2$');
    await settleTyping();

    // 反斜杠转义：`$x^2$` 不能被多留一个 `$` 在文档里
    expect(harness.editor.getMarkdown()).toBe('a $x^2$\n');
    expect(element(harness, '.qm-math-inline').dataset['tex']).toBe('x^2');
  });

  it('转换会让文档变脏 —— 否则按 Ctrl+S 存的是转换前的文字', async () => {
    const harness = await open('\n');
    const before = harness.reports.length;

    await typeInto(harness.dom, 'a $x^2$');
    await settleTyping();

    expect(harness.reports.length).toBeGreaterThan(before);
  });

  /**
   * **`$ $` 不转换。**
   *
   * 解析器认它（值是一个空格），但空格值的行内公式**存不住**：序列化之后
   * 变成块级公式，把整段文字劈成两半。所以输入规则在「解析器认了」之后
   * 还要过一道 `texProblem` —— 代价是 `$ $` 保持字面量，而那正是用户想要的
   * （他打的是两个美元符号，不是一个空公式）。
   */
  it('`$ $` 保持字面量 —— 空格值的行内公式存不住', async () => {
    const harness = await open('\n');
    await typeInto(harness.dom, '$ $');
    await settleTyping();

    expect(harness.dom.querySelector('.qm-math')).toBeNull();
    /**
     * 两个美元符号被**转义**了。这不是多余的动作：不转义的话重新打开
     * 这份文件时它又会被读成公式 —— 也就是我们刚刚拒绝的那件事。
     * 「字面量」在 Markdown 里必须写成 `\$`。
     */
    expect(harness.editor.getMarkdown()).toBe('\\$ \\$\n');
  });

  it('还没闭合就不转换', async () => {
    const harness = await open('\n');
    await typeInto(harness.dom, '价格 $5 到');
    await settleTyping();

    expect(harness.dom.querySelector('.qm-math')).toBeNull();
  });

  /**
   * **已知代价，钉住它。**
   *
   * `singleDollarTextMath` 保持默认的 `true`（关掉它行内公式就没了），
   * 代价是「两个美元符号夹一段文字」会被当成公式，连尾随空格都进值。
   * 这是上游规则，不在我们这边加判断 —— 加了就与「重新打开文件」不一致。
   */
  it('`$5 到 $` 会被当成公式 —— 与重新打开文件一致', async () => {
    const harness = await open('\n');
    await typeInto(harness.dom, '价格 $5 到 $');
    await settleTyping();

    expect(element(harness, '.qm-math-inline').dataset['tex']).toBe('5 到 ');
    // 存回之后重新打开，还是同一个公式（值不变）
    expect(harness.editor.getMarkdown()).toBe('价格 $5 到 $\n');
  });
});

// ---------------------------------------------------------------------------
// 输入规则：块级
// ---------------------------------------------------------------------------

describe('块级公式的输入规则', () => {
  it('在空段落开头敲 `$$ ` 变成块级公式', async () => {
    const harness = await open('\n');
    await typeInto(harness.dom, '$$ ');
    await settleTyping();

    expect(harness.dom.querySelector('.qm-math-block')).not.toBeNull();
    expect(harness.editor.getMarkdown()).toBe('$$\n$$\n');
  });

  /**
   * `^` 匹配的是 `textBefore` 的开头，而 `textBefore` 会被截断到光标前
   * 500 个字符。所以「恰好出现在开头」必须再核一次，否则一段长正文里
   * 第 501 个字符处的 `$$ ` 会把正文凭空换成一个公式。
   */
  it('出现在段落中间时**不**触发', async () => {
    const harness = await open('\n');
    await typeInto(harness.dom, '文字$$ ');
    await settleTyping();

    expect(harness.dom.querySelector('.qm-math-block')).toBeNull();
    expect(harness.editor.getMarkdown()).toBe('文字$$ \n');
  });

  it('只打 `$$` 不触发 —— 后面必须有空白', async () => {
    const harness = await open('\n');
    await typeInto(harness.dom, '$$');
    await settleTyping();

    expect(harness.dom.querySelector('.qm-math-block')).toBeNull();
  });

  /**
   * 这条钉的是一个**踩过的坑**：这条规则原来用 `setBlockType`，而它只接受
   * textblock（`math_block` 是块级原子节点），传进去抛
   * `RangeError: Type given to setBlockType should be a textblock`。
   *
   * 那是一条**异常**，会顺着 `handleTextInput` 冒到 ProseMirror 的事件处理里：
   * 输入无声无息地失败，用户只看到 `$$ ` 变成了字面量。
   * 现在走 `replaceWith($pos.before(), $pos.after(), …)`，把整个段落换掉。
   *
   * 所以这条用例要**同时**看两边：文档里有块级公式，**而且**没有未捕获异常。
   */
  it('整个段落被换成公式块（不是 `setBlockType`，那个会抛异常）', async () => {
    const harness = await open('\n');
    await typeInto(harness.dom, '$$ ');
    await settleTyping();

    const html = harness.editor.toHtml();
    expect(html).toContain(`data-type="${MATH_BLOCK}"`);
    // 那个空段落应该没了 —— 只换类型的话会留下一个空段落
    expect(html.match(/<p><\/p>/g) ?? [], '空段落被替换掉了，后面那个是 trailing 补的').toHaveLength(1);
  });
});

// ---------------------------------------------------------------------------
// 斜杠菜单
// ---------------------------------------------------------------------------

describe('斜杠菜单插入公式', () => {
  it('插入一个空的块级公式，筛选词被删掉', async () => {
    const harness = await open('\n');
    await typeInto(harness.dom, '/公');
    expect(harness.editor.getMarkdown()).toContain('/公');

    harness.editor.formatting.insertBlock('mathBlock');
    await settleTyping();

    expect(harness.editor.getMarkdown(), '菜单的筛选词不该留在文件里').not.toContain('/公');
    expect(harness.dom.querySelector('.qm-math-block')).not.toBeNull();
  });

  /**
   * 与浮层工具条、斜杠菜单、表格工具条同一条规矩：这些操作由用户点击触发、
   * **不经过键盘事件**，必须自己打 `USER_EDIT_META`。少了它用户按 Ctrl+S
   * 存的是插入前的旧内容。
   */
  it('插入会报给外壳', async () => {
    const harness = await open('前文\n');
    const before = harness.reports.length;

    harness.editor.formatting.insertBlock('mathBlock');
    await settleTyping();

    expect(harness.reports.length).toBeGreaterThan(before);
    expect(harness.reports[harness.reports.length - 1]).toContain('$$');
  });

  it('插入之后立刻进入编辑 —— 用户不用再点一下', async () => {
    const harness = await open('\n');

    harness.editor.formatting.insertBlock('mathBlock');
    await settleTyping();

    // 空公式没什么可渲染的，直接给一个等着打字的输入框
    expect(harness.dom.querySelector('.qm-math-input')).not.toBeNull();
  });

  /**
   * **插入公式块不能吃掉段落里的文字。**
   *
   * 这条钉的是一个真实踩过的坑：`insertMathBlock` 原来无条件把**整个段落**
   * 换成公式块，于是「在 `前文` 后面敲 `/公式` 选菜单」的结果是 `前文` 不见了 ——
   * 而且没有任何报错。
   *
   * 斜杠菜单的既有约定是「不在触发状态时一个字符都不删」（见 `slash.test.ts`），
   * 表格、代码块、分隔线都遵守它。公式块没有理由例外：段落是空的才整块替换，
   * 有文字就插在**后面**。
   */
  it('段落里有文字时公式插在后面，一个字都不动', async () => {
    const harness = await open('前文\n');
    await typeInto(harness.dom, '/公');
    await settleTyping();

    harness.editor.formatting.insertBlock('mathBlock');
    await settleTyping();

    const markdown = harness.editor.getMarkdown();
    expect(markdown, '原来的文字必须还在').toContain('前文');
    expect(markdown).toMatch(/^\$\$/m);
  });
});

// ---------------------------------------------------------------------------
// NodeView：渲染
// ---------------------------------------------------------------------------

describe('公式的渲染', () => {
  it('行内公式渲染出 KaTeX 的 HTML', async () => {
    const harness = await open('$x^2$\n');
    await waitForMath(harness);

    const box = element(harness, '.qm-math-inline');
    expect(box.className).toContain('is-loaded');
    expect(box.querySelector('.katex')).not.toBeNull();
    // MathML 是「真的渲染了」的证据，而不是把源码原样吐回来
    expect(box.innerHTML).toContain('<math');
  });

  it('块级公式带上 `katex-display`', async () => {
    const harness = await open('$$\n\\frac{1}{2}\n$$\n');
    await waitForMath(harness);

    expect(element(harness, '.qm-math-block').innerHTML).toContain('katex-display');
  });

  /**
   * **渲染结果绝不进文档。**
   *
   * 少了 `ignoreMutation`，KaTeX 写进去的那一大棵 DOM 会被 ProseMirror
   * 当成用户输入读进文档 —— 存出来的 Markdown 里多出一坨 KaTeX 的 HTML。
   */
  it('渲染出来的 DOM 不会跑进文档', async () => {
    const harness = await open('$x^2$\n');
    await waitForMath(harness);

    expect(element(harness, '.qm-math-inline').innerHTML).toContain('katex');
    expect(harness.editor.getMarkdown()).toBe('$x^2$\n');
    expect(harness.editor.getMarkdown()).not.toContain('katex');
    expect(harness.editor.getMarkdown()).not.toContain('<span');
  });

  it('渲染过程中文档不会变脏', async () => {
    const harness = await open('$x^2$\n');
    await waitForMath(harness);

    // 打开一份已有公式的文档，一个字都没改 —— 不该有任何上报
    expect(harness.reports).toEqual([]);
  });

  /**
   * 渲染失败必须**看得见**：给出错原因（用户需要知道改哪儿）**和**公式源码
   * （用户需要看见自己写的是什么）。只给一个空白或一个红点，等于把「为什么」藏起来。
   */
  it('渲染失败时画出源码与原因，而不是留一个空白', async () => {
    const harness = await open('$\\foo{$\n');
    await waitForMath(harness);

    const box = element(harness, '.qm-math-inline');
    expect(box.className).toContain('is-failed');
    expect(box.querySelector('.qm-math-src')?.textContent).toContain('\\foo{');
    const note = box.querySelector('.qm-math-note')?.textContent ?? '';
    expect(note).toContain('Undefined control sequence');
    // KaTeX 的定位标记（组合下划线）不能显示给用户
    expect(note).not.toContain('\u0332');
  });

  it('渲染失败不影响文档', async () => {
    const harness = await open('$\\foo{$\n');
    await waitForMath(harness);

    expect(harness.editor.getMarkdown()).toBe('$\\foo{$\n');
  });

  it('空公式不进渲染，直接给一个等着打字的输入框', async () => {
    const harness = await open('$$\n$$\n');
    await settleTyping();

    expect(harness.dom.querySelector('.qm-math-input')).not.toBeNull();
    expect(element(harness, '.qm-math-block').className).not.toContain('is-failed');
  });
});

// ---------------------------------------------------------------------------
// NodeView：原位编辑
// ---------------------------------------------------------------------------

describe('公式的原位编辑', () => {
  it('点一下公式就出现输入框，值是 TeX 源码', async () => {
    const harness = await open('$x^2$\n');
    await waitForMath(harness);
    expect(harness.dom.querySelector('.qm-math-input')).toBeNull();

    clickMath(harness, '.qm-math-inline');

    const input = element(harness, '.qm-math-input') as HTMLInputElement;
    expect(input.value).toBe('x^2');
    expect(element(harness, '.qm-math-inline').className).toContain('is-editing');
  });

  it('改完按 Enter 写回文档，并让文档变脏', async () => {
    const harness = await open('$x^2$\n');
    await waitForMath(harness);
    const before = harness.reports.length;

    clickMath(harness, '.qm-math-inline');
    editTo(harness, 'y^3');
    await settleTyping();

    expect(harness.editor.getMarkdown()).toBe('$y^3$\n');
    /**
     * 这一条才是这个功能的重点：输入框里的按键被 `stopEvent` 挡住了，
     * ProseMirror 的 DOM 观察者与 `beforeinput` 监听都收不到 ——
     * 不显式声明 `USER_EDIT_META`，用户按 Ctrl+S 存的是改之前的公式。
     */
    expect(harness.reports.length).toBeGreaterThan(before);
    expect(harness.reports[harness.reports.length - 1]).toBe('$y^3$\n');
  });

  it('确认之后回到渲染态，渲染的是**新**的值', async () => {
    const harness = await open('$x^2$\n');
    await waitForMath(harness);

    clickMath(harness, '.qm-math-inline');
    editTo(harness, 'y^3');
    await waitForMath(harness);

    expect(harness.dom.querySelector('.qm-math-input')).toBeNull();
    expect(element(harness, '.qm-math-inline').dataset['tex']).toBe('y^3');
    expect(element(harness, '.qm-math-inline').className).toContain('is-loaded');
  });

  it('值没变时确认不产生变更（也不该变脏）', async () => {
    const harness = await open('$x^2$\n');
    await waitForMath(harness);
    const before = harness.reports.length;

    clickMath(harness, '.qm-math-inline');
    editTo(harness, 'x^2');
    await settleTyping();

    expect(harness.editor.getMarkdown()).toBe('$x^2$\n');
    expect(harness.reports.length).toBe(before);
  });

  it('Esc 取消：一个字都不改', async () => {
    const harness = await open('$x^2$\n');
    await waitForMath(harness);
    const before = harness.reports.length;

    clickMath(harness, '.qm-math-inline');
    editTo(harness, '完全不一样的东西', 'Escape');
    await settleTyping();

    expect(harness.editor.getMarkdown()).toBe('$x^2$\n');
    expect(harness.reports.length).toBe(before);
    expect(harness.dom.querySelector('.qm-math-input')).toBeNull();
  });

  /**
   * **存不住的值拦下来，并且不退出编辑。**
   *
   * 空的行内公式序列化之后会变成**块级**公式，把整段文字劈成两半。
   * 所以 `commit` 里先校验；拦住时留在编辑态、把原因写在输入框旁边，
   * 让用户就着手改 —— 而不是写进去再告诉他「你的公式没了」。
   */
  it('空值被拦住：留在编辑态、说明原因、文档不变', async () => {
    const harness = await open('前文 $x^2$ 后文\n');
    await waitForMath(harness);

    clickMath(harness, '.qm-math-inline');
    editTo(harness, '');
    await settleTyping();

    expect(harness.editor.getMarkdown()).toBe('前文 $x^2$ 后文\n');
    expect(harness.dom.querySelector('.qm-math-input'), '还留在编辑态').not.toBeNull();
    const hint = element(harness, '.qm-math-hint');
    expect(hint.className).toContain('is-error');
    expect(hint.textContent).toContain('空的');
  });

  /**
   * 「行内公式含空行」这条拦截**在界面上走不到** —— 这一条把原因钉下来。
   *
   * 行内公式用单行 `<input>`，而 `<input>` 会把 `value` 里的换行剥掉，
   * 所以用户不可能在界面上造出含空行的值。`texProblem` 里那条拦截防的是
   * **程序写入**（将来的源码模式、整篇替换时带进来的值）——
   * 它是防御性的，不是日常路径。`core/math.test.ts` 里有它真的会丢内容的证据。
   */
  it('行内输入框装不下换行 —— 那条拦截只防程序写入', async () => {
    const harness = await open('$x$\n');
    await waitForMath(harness);

    clickMath(harness, '.qm-math-inline');
    const input = element(harness, '.qm-math-input') as HTMLInputElement;
    expect(input.tagName, '行内用单行 input').toBe('INPUT');

    input.value = 'a\n\nb';
    expect(input.value, '换行被 `<input>` 剥掉了').toBe('ab');
  });

  /**
   * 块级公式用 `<textarea>`，**能**装下换行 —— 而且空行是合法的
   * （`mdast-util-math` 会自己换成更长的围栏 `$$$`）。所以块级只挡空值。
   */
  it('块级公式可以含空行', async () => {
    const harness = await open('$$\nx\n$$\n');
    await waitForMath(harness);

    clickMath(harness, '.qm-math-block');
    const input = element(harness, '.qm-math-input') as HTMLTextAreaElement;
    expect(input.tagName, '块级用 textarea').toBe('TEXTAREA');

    input.value = 'a\n\nb';
    pressKey(input, 'Enter', { ctrlKey: true });
    await settleTyping();

    expect(harness.editor.getMarkdown()).toBe('$$\na\n\nb\n$$\n');
  });

  it('块级公式用 Ctrl+Enter 确认（Enter 是公式里的换行）', async () => {
    const harness = await open('$$\nx\n$$\n');
    await waitForMath(harness);

    clickMath(harness, '.qm-math-block');
    const input = element(harness, '.qm-math-input') as HTMLTextAreaElement;
    input.value = 'a\nb';
    pressKey(input, 'Enter');
    await settleTyping();
    // 单个 Enter 不是确认键 —— 文档不变，还在编辑
    expect(harness.editor.getMarkdown()).toBe('$$\nx\n$$\n');
    expect(harness.dom.querySelector('.qm-math-input')).not.toBeNull();

    pressKey(element(harness, '.qm-math-input'), 'Enter', { ctrlKey: true });
    await settleTyping();
    expect(harness.editor.getMarkdown()).toBe('$$\na\nb\n$$\n');
  });

  /**
   * 文档在编辑期间被换掉（撤销、整篇替换）—— 以**文档**为准，丢掉草稿。
   * 留着草稿会让用户以为自己的改动还在。
   */
  it('编辑期间文档被替换时，草稿被丢掉、以文档为准', async () => {
    const harness = await open('$x$\n');
    await waitForMath(harness);

    clickMath(harness, '.qm-math-inline');
    expect(harness.dom.querySelector('.qm-math-input')).not.toBeNull();

    harness.editor.setMarkdown('$z$\n');
    await waitForMath(harness);

    expect(harness.dom.querySelector('.qm-math-input')).toBeNull();
    expect(element(harness, '.qm-math-inline').dataset['tex']).toBe('z');
  });
});
