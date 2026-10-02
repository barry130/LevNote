// @vitest-environment jsdom
/**
 * 块手柄的**外壳半边**：显隐、菜单交互、拖动。
 *
 * ## 这里钉的是「手柄什么时候该在、什么时候该走」
 *
 * 手柄的显隐合**三个**条件，而只有第一个来自内核：
 *
 * 1. 指针停在某一块上（内核报的 `hover`）；
 * 2. 指针停在这个手柄自己身上；
 * 3. 菜单开着 / 正在拖动。
 *
 * 第 2 条是最容易漏的一条，而且漏了以后现象很怪：从正文移向手柄的中途
 * `hover` 变成 null，手柄被卸载，`mouseenter` 再也不会发 —— **用户永远点不到它**，
 * 但看起来「手柄就在那儿闪了一下」。
 *
 * ## 还有一条：什么时候告诉内核「目标块不要了」
 *
 * 判据是「手柄彻底不可交互」，**不是「菜单关了」**。菜单关掉时指针还在手柄上，
 * 用户随时会再点一次；那时清掉目标，第二次打开的菜单里每个动作都会
 * 「点了没反应」，而菜单画得好好的，完全看不出原因。
 *
 * ## 用假的内核命令面
 *
 * 手柄的全部输入就是 `hover` + 一个 `EditorBlockHandle`，所以给一个记录调用的假实现
 * 就能把外壳的行为测干净。真内核那一半在 `modules/markdown/block.test.ts`。
 */

import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import type {
  BlockFormat,
  BlockHandleState,
  BlockOperation,
  EditorBlockHandle,
  SelectionRect,
} from '../core/editor/types.js';
import { BlockHandle } from './BlockHandle.js';

declare global {
  // React 要求测试环境显式声明这一点，否则 act() 会警告并跳过批处理
  var IS_REACT_ACT_ENVIRONMENT: boolean | undefined;
}

// ---------------------------------------------------------------------------
// 假的命令面
// ---------------------------------------------------------------------------

interface Calls {
  run: BlockOperation[];
  turnInto: Array<{ block: BlockFormat; level: number | undefined }>;
  dropAt: Array<{ x: number; y: number }>;
  dropIndicatorAt: Array<{ x: number; y: number }>;
  close: number;
}

/** 落点线的位置。外壳只负责把它画出来，值本身没有意义。 */
const INDICATOR: SelectionRect = { top: 300, left: 100, bottom: 300, right: 700, width: 600, height: 0 };

const RECT: SelectionRect = { top: 100, left: 200, bottom: 140, right: 600, width: 400, height: 40 };

function makeHandle(): { handle: EditorBlockHandle; calls: Calls } {
  const calls: Calls = { run: [], turnInto: [], dropAt: [], dropIndicatorAt: [], close: 0 };
  return {
    calls,
    handle: {
      run: (operation) => calls.run.push(operation),
      turnInto: (block, level) => calls.turnInto.push({ block, level }),
      dropIndicatorAt: (x, y) => {
        calls.dropIndicatorAt.push({ x, y });
        return INDICATOR;
      },
      dropAt: (x, y) => calls.dropAt.push({ x, y }),
      close: () => {
        calls.close += 1;
      },
    },
  };
}

function hoverState(overrides: Partial<BlockHandleState> = {}): BlockHandleState {
  return {
    anchor: RECT,
    block: 'paragraph',
    headingLevel: null,
    canMoveUp: true,
    canMoveDown: true,
    canDelete: true,
    canTurnInto: true,
    ...overrides,
  };
}

// ---------------------------------------------------------------------------
// 渲染
// ---------------------------------------------------------------------------

interface Props {
  hover: BlockHandleState | null;
  blockHandle: EditorBlockHandle;
  menuOpen: boolean;
}

interface Mounted {
  container: HTMLElement;
  /** `onOpenMenu` 的调用记录。 */
  open: string[];
  /** `onCloseMenu` 的调用记录。 */
  close: string[];
  setProps: (next: Partial<Props>) => Promise<void>;
  unmount: () => Promise<void>;
}

let roots: Root[] = [];
let containers: HTMLElement[] = [];

/**
 * 挂一个手柄。
 *
 * 回调写成**每次渲染新造的函数**（真实外壳也是这么传的）——
 * 复用同一个引用会让「依赖数组写漏了」这类 bug 在测试里消失。
 */
async function mountHandle(initial: Props): Promise<Mounted> {
  const container = document.createElement('div');
  document.body.appendChild(container);
  const root = createRoot(container);
  roots.push(root);
  containers.push(container);

  const open: string[] = [];
  const close: string[] = [];
  let props = initial;

  const draw = async (): Promise<void> => {
    await act(async () => {
      root.render(
        <BlockHandle
          hover={props.hover}
          blockHandle={props.blockHandle}
          menuOpen={props.menuOpen}
          onOpenMenu={() => open.push('open')}
          onCloseMenu={() => close.push('close')}
        />,
      );
    });
  };

  await draw();
  return {
    container,
    open,
    close,
    setProps: async (next) => {
      props = { ...props, ...next };
      await draw();
    },
    unmount: async () => {
      await act(async () => {
        root.unmount();
      });
    },
  };
}

// ---------------------------------------------------------------------------
// 查询与事件
// ---------------------------------------------------------------------------

function gripOf(container: HTMLElement): HTMLButtonElement | null {
  const el = container.querySelector('.bh-grip');
  return el instanceof HTMLButtonElement ? el : null;
}

function menuOf(container: HTMLElement): HTMLElement | null {
  const el = container.querySelector('.bh-menu');
  return el instanceof HTMLElement ? el : null;
}

/** 按标签找菜单项。 */
function itemOf(container: HTMLElement, label: string): HTMLButtonElement {
  const items = Array.from(container.querySelectorAll('.bh-item'));
  const found = items.find((el) => el.textContent?.includes(label) ?? false);
  if (!(found instanceof HTMLButtonElement)) throw new Error(`菜单里没有「${label}」`);
  return found;
}

/** 高亮（键盘/鼠标当前指着的）那一项。 */
function highlightedOf(container: HTMLElement): string {
  const el = container.querySelector('.bh-item.highlighted');
  return el?.textContent ?? '';
}

async function pointer(
  target: EventTarget,
  type: string,
  options: { x?: number; y?: number } = {},
): Promise<void> {
  await act(async () => {
    target.dispatchEvent(
      new PointerEvent(type, { bubbles: true, clientX: options.x ?? 0, clientY: options.y ?? 0 }),
    );
  });
}

async function mouse(
  target: EventTarget,
  type: string,
  options: { detail?: number } = {},
): Promise<void> {
  await act(async () => {
    target.dispatchEvent(new MouseEvent(type, { bubbles: true, detail: options.detail ?? 0 }));
  });
}

async function key(key: string): Promise<void> {
  await act(async () => {
    window.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }));
  });
}

/**
 * 指针进入 / 离开一个元素。
 *
 * **不能直接派发 `mouseenter` / `mouseleave`**：React 的 `onMouseEnter` /
 * `onMouseLeave` 是从 `mouseover` / `mouseout` 合成出来的（它挂在根容器上，
 * 按 `relatedTarget` 算进出），真正的 `mouseenter` 它一个都不监听。
 * 派发前者会得到「事件发了、回调没跑」的假绿。
 */
async function pointerEnter(target: EventTarget): Promise<void> {
  await act(async () => {
    target.dispatchEvent(new MouseEvent('mouseover', { bubbles: true, relatedTarget: null }));
  });
}

async function pointerLeave(target: EventTarget): Promise<void> {
  await act(async () => {
    target.dispatchEvent(new MouseEvent('mouseout', { bubbles: true, relatedTarget: null }));
  });
}

beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
});

afterEach(async () => {
  for (const root of roots) {
    await act(async () => {
      root.unmount();
    });
  }
  roots = [];
  for (const container of containers) container.remove();
  containers = [];
});

// ---------------------------------------------------------------------------

describe('显隐：三个条件合起来才算', () => {
  it('指针不在编辑器里、也没碰手柄 → 什么都不渲染', async () => {
    const { handle } = makeHandle();
    const m = await mountHandle({ hover: null, blockHandle: handle, menuOpen: false });

    expect(gripOf(m.container)).toBeNull();
    expect(menuOf(m.container)).toBeNull();
  });

  it('指针停在一块上 → 手柄出现', async () => {
    const { handle } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: false });

    expect(gripOf(m.container)).not.toBeNull();
    expect(menuOf(m.container), '菜单要等点开').toBeNull();
  });

  /**
   * 这一条是文件头说的「第 2 个条件」。少了它，用户从正文移向手柄的中途
   * 手柄就被卸载了 —— 而且因为卸载得太早，`mouseenter` 根本不会发，
   * 用户怎么点都点不到。
   */
  it('指针移到手柄自己身上之后，内核报 null 也不该把柄收掉', async () => {
    const { handle } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: false });
    const grip = gripOf(m.container);
    expect(grip).not.toBeNull();
    if (grip === null) return;

    await pointerEnter(grip);
    await m.setProps({ hover: null });

    expect(gripOf(m.container), '指针还在柄上，柄就该在').not.toBeNull();

    // 指针真的走开了才收
    const still = gripOf(m.container);
    if (still !== null) await pointerLeave(still);
    expect(gripOf(m.container)).toBeNull();
  });

  it('菜单开着时内核报 null 也照样画（指针这会儿在菜单上）', async () => {
    const { handle } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: true });
    expect(menuOf(m.container)).not.toBeNull();

    await m.setProps({ hover: null });

    expect(menuOf(m.container), '菜单不能被指针位置收掉').not.toBeNull();
  });
});

describe('什么时候告诉内核「目标块不要了」', () => {
  it('手柄一直可见时不清目标块', async () => {
    const { handle, calls } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: false });
    const before = calls.close;

    await m.setProps({ menuOpen: true });
    await m.setProps({ menuOpen: false });

    expect(calls.close, '开关菜单不该清目标块 —— 用户随时会再点一次').toBe(before);
  });

  it('手柄彻底不可交互时才清', async () => {
    const { handle, calls } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: false });
    const before = calls.close;

    await m.setProps({ hover: null });

    expect(calls.close).toBeGreaterThan(before);
  });

  it('组件卸载时也清（切标签页 / 文档变成只读）', async () => {
    const { handle, calls } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: false });
    const before = calls.close;

    await m.unmount();

    expect(calls.close).toBeGreaterThan(before);
  });
});

describe('点开菜单：指针那条路与键盘那条路', () => {
  /**
   * 指针那条路在 `pointerup` 里当场判，不等 `click` ——
   * `pointerdown` 上做了 `preventDefault`（不拦的话拖动会顺手选中一片正文），
   * 而「被取消的 pointerdown 还会不会产生 click」各浏览器并不一致。
   */
  it('按下再松开、没有位移 → 打开菜单', async () => {
    const { handle } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: false });
    const grip = gripOf(m.container);
    if (grip === null) throw new Error('没有手柄');

    await pointer(grip, 'pointerdown', { x: 100, y: 100 });
    await pointer(window, 'pointerup', { x: 100, y: 100 });

    expect(m.open).toHaveLength(1);
    expect(m.close).toHaveLength(0);
  });

  it('指针点击产生的那次 `click` 不会把菜单再关一遍', async () => {
    const { handle } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: false });
    const grip = gripOf(m.container);
    if (grip === null) throw new Error('没有手柄');

    await pointer(grip, 'pointerdown', { x: 100, y: 100 });
    await pointer(window, 'pointerup', { x: 100, y: 100 });
    // 有些浏览器在取消过 pointerdown 之后仍会补一次 click
    await mouse(grip, 'click', { detail: 1 });

    expect(m.open, '开一次就够了，不能被 click 再翻一次').toHaveLength(1);
    expect(m.close).toHaveLength(0);
  });

  it('键盘激活（click 的 detail 为 0）照样能打开 —— 指针那条路管不到它', async () => {
    const { handle } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: false });
    const grip = gripOf(m.container);
    if (grip === null) throw new Error('没有手柄');

    await mouse(grip, 'click', { detail: 0 });

    expect(m.open).toHaveLength(1);
  });

  it('菜单开着时再点一次是关掉（切换语义）', async () => {
    const { handle } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: true });
    const grip = gripOf(m.container);
    if (grip === null) throw new Error('没有手柄');

    await pointer(grip, 'pointerdown', { x: 100, y: 100 });
    await pointer(window, 'pointerup', { x: 100, y: 100 });

    expect(m.close).toHaveLength(1);
    expect(m.open).toHaveLength(0);
  });
});

describe('拖动：位移超过阈值才算拖，落点交给内核算', () => {
  it('位移没到阈值就当点击', async () => {
    const { handle, calls } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: false });
    const grip = gripOf(m.container);
    if (grip === null) throw new Error('没有手柄');

    await pointer(grip, 'pointerdown', { x: 100, y: 100 });
    await pointer(window, 'pointermove', { x: 102, y: 101 });
    await pointer(window, 'pointerup', { x: 102, y: 101 });

    expect(calls.dropAt, '这点位移不该被当成拖动').toHaveLength(0);
    expect(m.open, '应当按点击处理').toHaveLength(1);
  });

  it('超过阈值 → 落点线由内核算，松手时把落点交给内核', async () => {
    const { handle, calls } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: false });
    const grip = gripOf(m.container);
    if (grip === null) throw new Error('没有手柄');

    await pointer(grip, 'pointerdown', { x: 100, y: 100 });
    await pointer(window, 'pointermove', { x: 140, y: 260 });

    expect(calls.dropIndicatorAt, '落点必须问内核要（外壳算不出来）').toEqual([{ x: 140, y: 260 }]);
    expect(m.container.querySelector('.bh-drop-line'), '指示线要画出来').not.toBeNull();

    await pointer(window, 'pointerup', { x: 140, y: 260 });

    expect(calls.dropAt).toEqual([{ x: 140, y: 260 }]);
    expect(m.open, '拖完不该顺手把菜单打开').toHaveLength(0);
    expect(m.close, '拖完要收菜单').toHaveLength(1);
    expect(m.container.querySelector('.bh-drop-line'), '松手后指示线要消失').toBeNull();
  });

  it('拖动中手柄不吃指针事件 —— 否则落点永远算在它自己身上', async () => {
    const { handle } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: false });
    const grip = gripOf(m.container);
    if (grip === null) throw new Error('没有手柄');

    await pointer(grip, 'pointerdown', { x: 100, y: 100 });
    await pointer(window, 'pointermove', { x: 140, y: 260 });

    const dragging = gripOf(m.container);
    expect(dragging?.classList.contains('dragging')).toBe(true);
  });

  it('指针事件被取消（系统抢走了指针）时不留下拖动状态', async () => {
    const { handle, calls } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: false });
    const grip = gripOf(m.container);
    if (grip === null) throw new Error('没有手柄');

    await pointer(grip, 'pointerdown', { x: 100, y: 100 });
    await pointer(window, 'pointermove', { x: 140, y: 260 });
    await pointer(window, 'pointercancel');

    expect(calls.dropAt, '取消不是松手，不该搬动任何块').toHaveLength(0);
    expect(gripOf(m.container)?.classList.contains('dragging')).toBe(false);
    expect(m.container.querySelector('.bh-drop-line')).toBeNull();
  });
});

describe('菜单：条目、禁用态、键盘导航', () => {
  it('「已经是这个格式」那一项显示成禁用，而不是点了没反应', async () => {
    const { handle } = makeHandle();
    const m = await mountHandle({ hover: hoverState({ block: 'heading', headingLevel: 2 }), blockHandle: handle, menuOpen: true });

    expect(itemOf(m.container, '二级标题').disabled).toBe(true);
    expect(itemOf(m.container, '一级标题').disabled).toBe(false);
    expect(itemOf(m.container, '正文').disabled).toBe(false);
  });

  it('不能上移 / 不能下移 / 不能删时对应的项是禁用的', async () => {
    const { handle } = makeHandle();
    const m = await mountHandle({
      hover: hoverState({ canMoveUp: false, canMoveDown: false, canDelete: false }),
      blockHandle: handle,
      menuOpen: true,
    });

    expect(itemOf(m.container, '上移').disabled).toBe(true);
    expect(itemOf(m.container, '下移').disabled).toBe(true);
    expect(itemOf(m.container, '删除').disabled).toBe(true);
    expect(itemOf(m.container, '复制一份').disabled, '复制永远可用').toBe(false);
  });

  it('原子块不显示「转为」那一组', async () => {
    const { handle } = makeHandle();
    const m = await mountHandle({ hover: hoverState({ block: null, canTurnInto: false }), blockHandle: handle, menuOpen: true });

    expect(() => itemOf(m.container, '正文')).toThrow();
    expect(itemOf(m.container, '上移')).toBeDefined();
  });

  it('点一项就执行它，然后收起菜单', async () => {
    const { handle, calls } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: true });

    await mouse(itemOf(m.container, '上移'), 'click');

    expect(calls.run).toEqual(['moveUp']);
    expect(m.close).toHaveLength(1);
  });

  it('点「转为」带上层级 —— 一级标题与二级标题是两个不同的条目', async () => {
    const { handle, calls } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: true });

    await mouse(itemOf(m.container, '二级标题'), 'click');

    expect(calls.turnInto).toEqual([{ block: 'heading', level: 2 }]);
  });

  it('禁用项点了什么都不做', async () => {
    const { handle, calls } = makeHandle();
    const m = await mountHandle({
      hover: hoverState({ canMoveUp: false }),
      blockHandle: handle,
      menuOpen: true,
    });

    await mouse(itemOf(m.container, '上移'), 'click');

    expect(calls.run).toHaveLength(0);
    expect(m.close, '点了没反应也不该收菜单').toHaveLength(0);
  });

  it('Esc 收起菜单', async () => {
    const { handle } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: true });

    await key('Escape');

    expect(m.close).toHaveLength(1);
  });

  /**
   * 方向键要**跳过禁用项** —— 停在禁用项上的话，用户按回车会「什么都没发生」，
   * 而高亮明明在动，看起来像卡住了。
   */
  it('方向键跳过禁用项', async () => {
    const { handle } = makeHandle();
    // 「正文」是当前格式 → 禁用；所以第一项就是禁用的
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: true });

    await key('ArrowDown');
    expect(highlightedOf(m.container), '要跳过禁用的「正文」').toContain('一级标题');

    await key('ArrowUp');
    expect(highlightedOf(m.container), '往上也要跳过它').toContain('删除');
  });

  it('回车执行高亮的那一项', async () => {
    const { handle, calls } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: true });

    await key('ArrowDown');
    await key('Enter');

    expect(calls.turnInto).toEqual([{ block: 'heading', level: 1 }]);
    expect(m.close).toHaveLength(1);
  });

  it('菜单里滚动不收菜单（菜单自己会滚）', async () => {
    const { handle } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: true });
    const menu = menuOf(m.container);
    if (menu === null) throw new Error('没有菜单');

    await act(async () => {
      menu.dispatchEvent(new Event('scroll', { bubbles: false }));
    });

    expect(m.close).toHaveLength(0);
  });

  it('外面滚动收菜单（捕获阶段 —— `scroll` 不冒泡，挂在 window 上只有捕获才收得到）', async () => {
    const { handle } = makeHandle();
    const m = await mountHandle({ hover: hoverState(), blockHandle: handle, menuOpen: true });

    await act(async () => {
      document.body.dispatchEvent(new Event('scroll', { bubbles: false }));
    });

    expect(m.close.length).toBeGreaterThan(0);
  });
});
