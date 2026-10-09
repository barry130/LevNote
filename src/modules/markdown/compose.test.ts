/**
 * 专注 / 打字机插件的判定规则。
 *
 * 打字机那半边依赖真实布局（`coordsAtPos` / `getBoundingClientRect`），
 * jsdom 里 `typewriterScrollTop` 拿到的全是 0，天然是空操作 —— 所以这里
 * 只能钉住**插件自己的契约**：开关状态怎么进、装饰按什么规则给（哪块亮、
 * 哪块淡）、关掉之后装饰消失。滚动算术本身在
 * `core/editor/position.ts` 有自己的单测。
 */

import { Schema, type Node as ProseNode } from '@milkdown/kit/prose/model';
import { EditorState, TextSelection, type Transaction } from '@milkdown/kit/prose/state';
import { type Decoration, DecorationSet } from '@milkdown/kit/prose/view';
import { describe, expect, it } from 'vitest';

import { composeKey, composePlugin, DIMMED_CLASS } from './plugin.js';

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

function threeParagraphs(): ProseNode {
  return schema.node('doc', null, [
    schema.node('paragraph', null, [schema.text('第一段')]),
    schema.node('paragraph', null, [schema.text('第二段')]),
    schema.node('paragraph', null, [schema.text('第三段')]),
  ]);
}

/**
 * 取一个装饰的类名。
 *
 * 装饰的**公开类型**只有 from / to / spec，没有 .type —— 类名实际落在内部的
 * type.attrs 上（Decoration.node / Decoration.inline 传的 attrs 原样存在那里）。
 * 这里不 as 也不 any，而是把内部字段读成 unknown 再逐层收窄：
 * 读不到就是 undefined，断言照常失败，不会把「类名变了」静默放过。
 */
function classNameOf(decoration: Decoration): string | undefined {
  const internals: unknown = Reflect.get(decoration, 'type');
  if (typeof internals !== 'object' || internals === null) return undefined;
  const attrs: unknown = Reflect.get(internals, 'attrs');
  if (typeof attrs !== 'object' || attrs === null) return undefined;
  const className: unknown = Reflect.get(attrs, 'class');
  return typeof className === 'string' ? className : undefined;
}

function setup(initial = { focus: false, typewriter: false }) {
  const plugin = composePlugin(
    // jsdom 没有布局，滚动容器不会被真的碰到；给个假元素满足类型即可
    () => ({}) as HTMLElement,
    initial,
  );
  let state = EditorState.create({
    schema,
    doc: threeParagraphs(),
    plugins: [plugin],
  });

  const dispatch = (fn: (tr: Transaction) => void) => {
    const tr = state.tr;
    fn(tr);
    state = state.apply(tr);
  };
  const setCursorIn = (text: string) => {
    dispatch((tr) => {
      let pos = 0;
      state.doc.forEach((node, offset) => {
        if (node.textContent === text) pos = offset + 1;
      });
      tr.setSelection(TextSelection.create(state.doc, pos));
    });
  };
  const decorations = (): DecorationSet => {
    const decorationsProp = plugin.props?.decorations;
    const source = decorationsProp?.call(plugin, state);
    return source instanceof DecorationSet ? source : DecorationSet.empty;
  };
  const dimmedTos = (): number[] =>
    decorations()
      .find()
      .filter((d) => classNameOf(d) === DIMMED_CLASS)
      .map((d) => d.to);

  return {
    get state() {
      return state;
    },
    setCursorIn,
    dispatch,
    dimmedTos,
    mode: () => composeKey.getState(state),
    setMode: (mode: { focus: boolean; typewriter: boolean }) => {
      dispatch((tr) => tr.setMeta(composeKey, mode));
    },
  };
}

describe('composePlugin · 专注', () => {
  it('开关默认关：没有装饰', () => {
    const h = setup();
    expect(h.mode()).toEqual({ focus: false, typewriter: false });
    expect(h.dimmedTos()).toEqual([]);
  });

  it('打开后光标所在块亮，其余变淡', () => {
    const h = setup();
    h.setCursorIn('第二段');
    h.setMode({ focus: true, typewriter: false });
    // 三段各占 5 个位置（1 开 + 3 字 + 1 合）：块边界 0/5/10/15。
    // 光标在第二段 → 第一段（to=5）与第三段（to=15）变淡
    expect(h.dimmedTos()).toEqual([5, 15]);
  });

  it('光标移走后亮块跟着走（装饰每次现算）', () => {
    const h = setup();
    h.setCursorIn('第一段');
    h.setMode({ focus: true, typewriter: false });
    // 亮的是第一段 → 第二段（to=10）与第三段（to=15）淡
    expect(h.dimmedTos()).toEqual([10, 15]);
    h.setCursorIn('第三段');
    // 亮的是第三段 → 第一段（to=5）与第二段（to=10）淡
    expect(h.dimmedTos()).toEqual([5, 10]);
  });

  it('非空选区跨两块时两块都亮', () => {
    const h = setup();
    // 从第一段开头选到第二段中间
    h.dispatch((tr) => tr.setSelection(TextSelection.create(h.state.doc, 1, 8)));
    h.setMode({ focus: true, typewriter: false });
    expect(h.dimmedTos()).toEqual([15]);
  });

  it('关掉之后装饰清空', () => {
    const h = setup();
    h.setMode({ focus: true, typewriter: false });
    expect(h.dimmedTos()).toHaveLength(2);
    h.setMode({ focus: false, typewriter: false });
    expect(h.dimmedTos()).toEqual([]);
  });

  it('初始状态可以带开（外壳在内核创建那一刻就可能已是开着的）', () => {
    const h = setup({ focus: true, typewriter: false });
    expect(h.dimmedTos()).toHaveLength(2);
  });
});

describe('composePlugin · 打字机（状态侧）', () => {
  it('开关进得了插件状态（滚动本身依赖布局，jsdom 里是空操作）', () => {
    const h = setup();
    h.setMode({ focus: false, typewriter: true });
    expect(h.mode()).toEqual({ focus: false, typewriter: true });
    // 开着也不会改文档
    expect(h.state.doc.textContent).toBe('第一段第二段第三段');
  });
});
