/**
 * 变更上报插件的判定规则。
 *
 * 这个插件决定了一件很要命的事：**什么时候算「用户改了内容」**。
 * 判错的方向有两种，代价都很实在：
 *
 * - 判成「用户改了」，实际是程序写入 → 打开文件没动过就显示「有未保存的改动」，
 *   用户被引着去点保存，文件里他没碰过的排版就被规范化改写。
 * - 判成「程序写入」，实际是用户改了 → 用户的改动不上报，外壳以为没变、
 *   不标记脏、不提示保存，**改动静默丢失**。
 *
 * 所以正反两个方向都要钉住。
 */

import { Schema } from '@milkdown/kit/prose/model';
import { EditorState, TextSelection } from '@milkdown/kit/prose/state';
import { describe, expect, it } from 'vitest';

import { markdownEmitPlugin, SILENT_META } from './plugin.js';

const schema = new Schema({
  nodes: {
    doc: { content: 'block+' },
    paragraph: {
      content: 'text*',
      group: 'block',
      toDOM: () => ['p', 0],
      parseDOM: [{ tag: 'p' }],
    },
    text: {},
  },
});

interface Harness {
  reported: string[];
  /** 当前文档的纯文本，用来确认编辑器状态确实在推进 */
  text(): string;
  /** 造一次真实输入事件的效果（参数被忽略，给空值即可）。 */
  typeInto(): void;
  /** 按一次 Ctrl+Z。 */
  pressUndo(): void;
  /** 在文档最前面插入一段文字并推进状态。`silent` 模拟 setMarkdown 写入。 */
  insert(text: string, options?: { silent?: boolean }): void;
  /** 只移动光标，不改文档。 */
  moveCursor(): void;
}

function setup(): Harness {
  const reported: string[] = [];
  const plugin = markdownEmitPlugin(
    (markdown) => reported.push(markdown),
    // 用 textContent 当序列化器：这个测试关心的是「报没报」，不是序列化得对不对
    (doc) => doc.textContent,
  );

  let state = EditorState.create({
    schema,
    doc: schema.node('doc', null, [schema.node('paragraph', null, [schema.text('原文')])]),
    plugins: [plugin],
  });

  return {
    reported,
    text: () => state.doc.textContent,
    typeInto: () => {
      // 用 .call(plugin, ...) 而不是 plugin.props.x(...)：ProseMirror 的处理器
      // 声明了 `this: Plugin`，直接从 props 上取出来调用会让 this 指向 props 本身。
      plugin.props.handleDOMEvents?.['input']?.call(plugin, undefined as never, undefined as never);
    },
    pressUndo: () => {
      plugin.props.handleKeyDown?.call(plugin, undefined as never, {
        ctrlKey: true,
        metaKey: false,
        key: 'z',
      } as KeyboardEvent);
    },
    insert: (text, options) => {
      // 注意每次都从**当前** state 派发。上一版这里写成了从初始 state 派发，
      // 结果后一次变更把前一次覆盖掉，还让「一次交互只上报一次」因为错误的原因通过。
      const tr = state.tr.insert(1, schema.text(text));
      if (options?.silent) tr.setMeta(SILENT_META, true);
      state = state.apply(tr);
    },
    moveCursor: () => {
      state = state.apply(state.tr.setSelection(TextSelection.create(state.doc, 1)));
    },
  };
}

describe('用户改动的判定', () => {
  it('输入事件之后的文档变更会上报，报的是当前完整文档', () => {
    const h = setup();

    h.typeInto();
    h.insert('新增');

    expect(h.reported).toEqual(['新增原文']);
    expect(h.text()).toBe('新增原文');
  });

  it('没有输入事件的文档变更不上报 —— 初始化、trailing 补段落都属于这类', () => {
    const h = setup();

    h.insert('程序写的');

    expect(h.reported).toEqual([]);
    expect(h.text()).toBe('程序写的原文');
  });

  it('撤销快捷键也算用户交互', () => {
    const h = setup();

    h.pressUndo();
    h.insert('撤销回来的');

    expect(h.reported).toEqual(['撤销回来的原文']);
  });

  it('只是移动光标不会消耗掉「已武装」状态', () => {
    const h = setup();

    h.typeInto();
    // 中间来一条不改文档的事务（比如点了一下别处）：它不该把这次交互的「名额」占掉
    h.moveCursor();
    h.insert('改');

    expect(h.reported).toEqual(['改原文']);
  });

  it('一次交互只上报一次，不会连续报', () => {
    const h = setup();

    h.typeInto();
    h.insert('一');
    // 第二条变更没有新的输入事件，属于程序行为
    h.insert('二');

    expect(h.reported).toEqual(['一原文']);
    // 但文档确实两边都改了 —— 证明这次不是靠「事务没生效」蒙过去的
    expect(h.text()).toBe('二一原文');
  });

  it('每次输入都会重新武装', () => {
    const h = setup();

    h.typeInto();
    h.insert('一');
    h.typeInto();
    h.insert('二');

    expect(h.reported).toEqual(['一原文', '二一原文']);
  });
});

describe('程序写入的判定', () => {
  it('带 SILENT_META 的事务不上报', () => {
    const h = setup();

    h.typeInto();
    h.insert('setMarkdown 写的', { silent: true });

    expect(h.reported).toEqual([]);
    expect(h.text()).toBe('setMarkdown 写的原文');
  });

  it('带 SILENT_META 的事务会消耗掉「已武装」状态', () => {
    const h = setup();

    h.typeInto();
    h.insert('一', { silent: true });
    // 这次交互已经被上面那条事务用掉了，紧接着的程序变更不该被误当成用户改动
    h.insert('二');

    expect(h.reported).toEqual([]);
  });

  it('用户先改、程序后写：只报用户那一次', () => {
    const h = setup();

    h.typeInto();
    h.insert('用户的');
    h.insert('程序的', { silent: true });

    expect(h.reported).toEqual(['用户的原文']);
    expect(h.text()).toBe('程序的用户的原文');
  });

  it('程序先写、用户后改：只报用户那一次，且报的是完整文档', () => {
    const h = setup();

    h.insert('程序的', { silent: true });
    h.typeInto();
    h.insert('用户的');

    expect(h.reported).toEqual(['用户的程序的原文']);
  });
});
