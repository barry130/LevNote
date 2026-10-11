// @vitest-environment jsdom
/**
 * 链接的**内核半边**：点击拦截与锚点定位。
 *
 * ## 这一组测试真正在钉的三条不变量
 *
 * 1. **编辑器里的链接绝不能让 WebView 导航走，一次都不能。**
 *    默认行为是「整个应用变成一个网页」，且**回不来**。所以 `preventDefault`
 *    必须是无条件的 —— 包括没按修饰键的那一次点击（它只是移动光标）。
 *    漏掉它的表现不是「打开失败」，而是应用消失。
 * 2. **上报的必须是文档里的原始 `href`，不是 DOM 上的那个。**
 *    DOM 上的是消毒后的值（`javascript:` → 空串）。用它的后果是用户点了一个
 *    明明白白写着地址的链接，却只能得到一句与事实无关的话，或者干脆没反应
 *    （硬约束 2）。
 * 3. **普通点击（不按 Ctrl / Cmd）不上报。**
 *    编辑器里的左键点击是**移动光标**：想点进链接中间改一个字是最正常的操作，
 *    被劫持成「打开网页」等于这个功能没法用。
 *
 * ## 为什么命中走 DOM 元素而不是坐标
 *
 * 与 `block.test.ts` 同一条理由：jsdom 没有 `elementFromPoint` / `caretRangeFromPoint`，
 * 走坐标必抛。走 `event.target` → `posAtDOM` 反而就是它的真实语义。
 */

import { Schema } from '@milkdown/kit/prose/model';
import { afterEach, describe, expect, it } from 'vitest';

import type { LinkActivation } from '../../core/editor/types.js';
import { createMarkdownEditor, type MarkdownEditorInstance } from './setup.js';
import { findAnchorPos } from './link.js';

// ---------------------------------------------------------------------------
// 纯函数那一半：一张最小 schema 就够
// ---------------------------------------------------------------------------

/**
 * 只有 heading 带 `id`。
 *
 * 刻意**不复用内核的 schema**：`findAnchorPos` 只吃「节点的 `attrs.id`」一样东西，
 * 自己造一张小表能让「哪个节点有哪个 id」一眼看清 —— 而这里要验的正是
 * 「它按 id 找，不是按节点类型找」。
 */
const schema = new Schema({
  nodes: {
    doc: { content: 'block+' },
    // 段落也带 `id`：为的是证明函数认的是「有没有这个 id」而非「是不是标题」
    paragraph: { content: 'text*', group: 'block', attrs: { id: { default: '' } } },
    heading: { content: 'text*', group: 'block', attrs: { level: { default: 1 }, id: { default: '' } } },
    text: {},
  },
});

/**
 * 上面那张表的位置算术（节点 size = 开标签 1 + 文本 + 闭标签 1）：
 * `paragraph`(正文) 占 4 → 下一个块从 4 开始；`heading`(第一节) 占 5 → 再下一个从 9 开始。
 */
describe('findAnchorPos', () => {
  const doc = schema.node('doc', null, [
    schema.node('paragraph', null, [schema.text('正文')]),
    schema.node('heading', { level: 2, id: 'sec-a' }, [schema.text('第一节')]),
    schema.node('heading', { level: 2, id: 'sec-b' }, [schema.text('第二节')]),
  ]);

  it('按 id 找到那个节点的位置（位置是节点**之前**的边界）', () => {
    expect(findAnchorPos(doc, 'sec-b')).toBe(9);
  });

  it('第一个匹配项优先，且不会继续往下找', () => {
    expect(findAnchorPos(doc, 'sec-a')).toBe(4);
  });

  it('没有这个 id 时返回 null —— 由外壳把它说成一句话', () => {
    expect(findAnchorPos(doc, 'nope')).toBeNull();
  });

  /**
   * 空串是 heading 的 `id` 默认值（preset-commonmark 的 schema 里它就是 `''`），
   * 而不是「没有 id」。锚点为空时外壳压根不会来问（见 `classifyLink` 的 `empty`），
   * 但这里仍要钉住：拿空串去问不能命中一堆没写过 id 的标题。
   */
  it('空串 id 不匹配没有 id 的节点', () => {
    expect(findAnchorPos(doc, '')).toBeNull();
  });

  /**
   * **刻意不限定为标题**：判别依据只有「有没有这个 id」。
   * 今天只有标题带 id，但把规则写成「只看 heading」等于替明天的 schema 做决定。
   */
  it('非标题节点上的 id 也认 —— 判据是 id 本身，不是节点类型', () => {
    // 给段落也声明 `id`，才能证明函数不认识「heading」这个名字也照样命中
    const tagged = schema.node('doc', null, [
      schema.node('paragraph', null, [schema.text('正文')]),
      schema.node('paragraph', { id: 'note-1' }, [schema.text('锚点段')]),
    ]);
    expect(findAnchorPos(tagged, 'note-1')).toBe(4);
  });
});

// ---------------------------------------------------------------------------
// 点击拦截：真内核
// ---------------------------------------------------------------------------

let opened: MarkdownEditorInstance | null = null;

afterEach(async () => {
  await opened?.destroy();
  opened = null;
  document.body.innerHTML = '';
});

interface Harness {
  editor: MarkdownEditorInstance;
  dom: HTMLElement;
  reported: LinkActivation[];
}

interface OpenOptions {
  /** 上报器在**点击那一刻**才被取用，所以可以中途换掉（见「ref 镜像」那条用例）。 */
  onLinkClick?: (link: LinkActivation) => void;
}

async function open(source: string, options: OpenOptions = {}): Promise<Harness> {
  const root = document.createElement('div');
  document.body.appendChild(root);
  const reported: LinkActivation[] = [];

  const editor = await createMarkdownEditor({
    root,
    defaultValue: source,
    onChange: () => undefined,
    onLinkClick: options.onLinkClick ?? ((link) => reported.push(link)),
  });
  opened = editor;

  const dom = root.querySelector('.ProseMirror');
  if (!(dom instanceof HTMLElement)) throw new Error('没有找到 ProseMirror 视图');
  return { editor, dom, reported };
}

/** 等一帧（插件初始化 / 事务结算）。 */
async function settle(): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, 0));
  await new Promise((resolve) => setTimeout(resolve, 0));
}

/**
 * 在链接上点一下。
 *
 * `cancelable: true` 是必须的：不可取消的事件上 `preventDefault` 是空操作，
 * 而「有没有掐掉默认行为」正是这一组用例的一号断言。
 */
function clickLink(h: Harness, options: { ctrlKey?: boolean; metaKey?: boolean } = {}): MouseEvent {
  const anchor = h.dom.querySelector('a');
  if (!(anchor instanceof HTMLElement)) throw new Error('文档里没有 <a>');
  const event = new MouseEvent('click', {
    bubbles: true,
    cancelable: true,
    ctrlKey: options.ctrlKey ?? false,
    metaKey: options.metaKey ?? false,
  });
  anchor.dispatchEvent(event);
  return event;
}

describe('点击拦截：默认导航必须无条件掐掉', () => {
  /**
   * 一号不变量：按了修饰键的那次点击。
   * 少了 `preventDefault`，WebView 会把整个应用导航到那个地址。
   */
  it('Ctrl+点击：掐掉默认行为并上报', async () => {
    const h = await open('[链接](./a.md)\n');
    await settle();

    const event = clickLink(h, { ctrlKey: true });

    expect(event.defaultPrevented, '不掐掉的话 WebView 会把整个应用导航走').toBe(true);
    expect(h.reported).toEqual([{ href: './a.md', text: '链接' }]);
  });

  /**
   * 二号不变量：**没按**修饰键的那次点击也必须掐掉。
   * 它只是移动光标，但导航照样会发生 —— 这才是最容易漏的一条。
   */
  it('普通点击：不上报，但默认行为**照样**掐掉', async () => {
    const h = await open('[链接](./a.md)\n');
    await settle();

    const event = clickLink(h);

    expect(event.defaultPrevented, '普通点击同样会让 WebView 导航走').toBe(true);
    expect(h.reported, '普通点击是移动光标，不是打开链接').toEqual([]);
  });

  it('Cmd+点击与 Ctrl 等价（macOS 的手势）', async () => {
    const h = await open('[链接](https://example.com)\n');
    await settle();

    clickLink(h, { metaKey: true });

    expect(h.reported).toEqual([{ href: 'https://example.com', text: '链接' }]);
  });

  /**
   * 这一条是「不读 DOM 的 href」这条设计的**回归测试**。
   * Milkdown 渲染时会把 `javascript:` 消毒成空串；DOM 上看上去是
   * 「这个链接没有地址」，而文档里明明写着地址。
   * 读 DOM 的后果：用户点了只能得到一句与事实无关的话，或者干脆没反应。
   */
  it('上报的是文档原文 —— 消毒后的 DOM href 不算数', async () => {
    const h = await open('[坏链接](javascript:alert(1))\n');
    await settle();

    const anchor = h.dom.querySelector('a');
    expect(anchor?.getAttribute('href'), '渲染层确实把它消毒掉了').toBe('');

    clickLink(h, { ctrlKey: true });

    expect(h.reported, '外壳要靠原文才能如实说清「这个协议打不开」').toEqual([
      { href: 'javascript:alert(1)', text: '坏链接' },
    ]);
  });

  it('点在链接**外面**（按了 Ctrl 也一样）不上报', async () => {
    const h = await open('[链接](./a.md)\n');
    await settle();

    const event = new MouseEvent('click', { bubbles: true, cancelable: true, ctrlKey: true });
    h.dom.querySelector('p')?.dispatchEvent(event);

    expect(h.reported).toEqual([]);
  });

  /**
   * 上报器是**点击那一刻**才取的（`getReporter()`），不是创建时捕获的闭包。
   * 否则外壳改一次回调就得重建整个编辑器 —— 而编辑器重建会丢掉光标与撤销栈。
   */
  it('上报器在点击那一刻才取 —— 外壳换回调不需要重建编辑器', async () => {
    const root = document.createElement('div');
    document.body.appendChild(root);
    const first: LinkActivation[] = [];
    const second: LinkActivation[] = [];
    let current: (link: LinkActivation) => void = (link) => first.push(link);

    const editor = await createMarkdownEditor({
      root,
      defaultValue: '[链接](./a.md)\n',
      onChange: () => undefined,
      onLinkClick: (link) => current(link),
    });
    opened = editor;
    const dom = root.querySelector('.ProseMirror');
    if (!(dom instanceof HTMLElement)) throw new Error('没有找到 ProseMirror 视图');

    await settle();
    dom.querySelector('a')?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, ctrlKey: true }));
    expect(first).toHaveLength(1);

    current = (link) => second.push(link);
    dom.querySelector('a')?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, ctrlKey: true }));

    expect(second, '换了回调之后要打到新的那个上').toHaveLength(1);
    expect(first).toHaveLength(1);
  });
});

describe('锚点定位', () => {
  /**
   * 标题的 `id` 由 preset-commonmark 的 `syncHeadingIdPlugin` 生成
   * （`textContent.toLowerCase().trim().replace(/\s+/g, '-')`），
   * 所以「第二节」这个标题的 id 就是 `第二节`。
   */
  it('跳到存在的锚点：返回 true，光标进了那一节', async () => {
    const h = await open('## 第一节\n\n正文\n\n## 第二节\n\n正文二\n');
    await settle();

    expect(h.editor.goToAnchor('第二节')).toBe(true);
  });

  it('没有这个锚点时返回 false —— 由外壳说一句话，而不是静默', async () => {
    const h = await open('## 第一节\n\n正文\n');
    await settle();

    expect(h.editor.goToAnchor('不存在')).toBe(false);
  });

  it('空锚点返回 false（空串不是「任何一节」）', async () => {
    const h = await open('## 第一节\n\n正文\n');
    await settle();

    expect(h.editor.goToAnchor('')).toBe(false);
  });
});
