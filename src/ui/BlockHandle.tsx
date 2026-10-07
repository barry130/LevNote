/**
 * 块手柄：贴在某一块左侧的握把 + 点开后的块级菜单 + 拖动重排。
 *
 * ## 分工与另外四个浮层一致
 *
 * 内核只报**状态与几何**（哪一块、它的包围盒、能不能上移），
 * 外壳画界面、管交互、管开关时机。手柄本身是外壳画的，所以它才能自己挂
 * `mouseenter` —— 这一点是必要的，不是顺手：手柄在**编辑器外面**，
 * 用户把鼠标从正文移到手柄上的那一刻，内核一定会报「指针离开编辑器」（`null`）。
 *
 * ## 手柄的显隐要合三个条件，不能只看内核报的那个值
 *
 * 1. **指针停在某一块上**（内核报的 `hover`）；
 * 2. **指针停在这个手柄自己身上** —— 从 1 走到 2 的那一瞬间 1 已经不成立了，
 *    只看 1 的话手柄会在用户点到它之前先消失（`click` 永远发不出来）；
 * 3. **菜单开着 / 正在拖动** —— 这时指针可能在菜单上，两个条件都不成立。
 *
 * 这三条里 2 与 3 只有外壳知道，所以**显隐的最终决定权必须在外壳**。
 * 内核那边对应的保证是：它记着的目标块不在鼠标离开时清（见 `EditorBlockHandle`）。
 *
 * ## 菜单为什么用 `window` 捕获阶段收键盘
 *
 * 与斜杠菜单同一条理由（硬约束 9）：点手柄时焦点大概率还在 contenteditable 里
 * （`pointerdown` 上做了 `preventDefault`，焦点根本不会移到按钮上），
 * 于是 Esc / 方向键会先在 `view.dom` 的冒泡路径上被 ProseMirror 处理掉 ——
 * 按 Esc 变成了「移动光标」，而不是「收起菜单」。
 *
 * ## 拖动为什么不用 HTML5 拖放
 *
 * 原生 DnD 的落点反馈只能靠 `dragover` + `preventDefault`，画不出一条跟着走的
 * 指示线（要自己叠一层，还要处理 `dragend` 之后残留的 ghost）。
 * 用指针事件自己实现，落点线是**内核算出来的**（`dropIndicatorAt`），
 * 外壳只负责画 —— 文档坐标一个都没有离开内核。
 *
 * ## 「点开菜单」为什么不挂在 `click` 上
 *
 * `pointerdown` 上必须 `preventDefault()`（不拦这一下，拖动会顺手把正文选中一片）。
 * 而**被取消的 `pointerdown` 还会不会产生 `click`，各浏览器的处理并不一致** ——
 * 押在 `click` 上等于押在一个没写进规范的行为上。
 *
 * 所以指针那条路在 `pointerup` 里当场判（没位移就是点击），
 * `click` 只留给键盘激活，并用 `event.detail === 0` 把它们分开
 * （键盘产生的 `click` 没有点击次数）。
 */

import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import type { MouseEvent as ReactMouseEvent, PointerEvent as ReactPointerEvent } from 'react';

import type { BlockFormat, BlockHandleState, EditorBlockHandle, SelectionRect } from '../core/editor/types.js';
import { t, type MessageKey } from '../core/i18n/index.js';
import { computeFloatingPlacement, FLOATING_MARGIN } from './floating.js';
import { IconGrip } from './icons.js';

/** 手柄的边长（CSS 里也是这个值）。菜单要按它算锚点，所以必须是常量。 */
const GRIP_SIZE = 22;
/** 手柄与块左边缘的距离。 */
const GRIP_OFFSET = 26;
/** 位移小于它算「点击」，大于它算「拖动」。 */
const DRAG_THRESHOLD = 4;

/** 「转为」那一组。顺序即菜单顺序。 */
interface TurnIntoItem {
  readonly id: string;
  /** 存 key 不存文案：静态清单在模块加载时求值，那时把语言钉死就再也换不回来了。 */
  readonly labelKey: MessageKey;
  /** 右侧的写法提示。用 Markdown 里的写法，不用图标。 */
  readonly hint: string;
  readonly block: BlockFormat;
  readonly level?: number;
}

const TURN_INTO: readonly TurnIntoItem[] = [
  { id: 'paragraph', labelKey: 'ui.blockhandle.turnParagraph', hint: '¶', block: 'paragraph' },
  { id: 'heading1', labelKey: 'ui.blockhandle.turnHeading1', hint: '#', block: 'heading', level: 1 },
  { id: 'heading2', labelKey: 'ui.blockhandle.turnHeading2', hint: '##', block: 'heading', level: 2 },
  { id: 'heading3', labelKey: 'ui.blockhandle.turnHeading3', hint: '###', block: 'heading', level: 3 },
  { id: 'blockquote', labelKey: 'ui.blockhandle.turnQuote', hint: '>', block: 'blockquote' },
  { id: 'bulletList', labelKey: 'ui.blockhandle.turnBulletList', hint: '- ', block: 'bulletList' },
  { id: 'orderedList', labelKey: 'ui.blockhandle.turnOrderedList', hint: '1. ', block: 'orderedList' },
  { id: 'codeBlock', labelKey: 'ui.blockhandle.turnCodeBlock', hint: '```', block: 'codeBlock' },
];

export interface BlockHandleProps {
  /**
   * 内核报的悬停状态。`null` 表示「指针不在编辑器里」——
   * **它不等于「该把手柄藏起来」**，理由见文件头。
   */
  hover: BlockHandleState | null;
  blockHandle: EditorBlockHandle;
  /** 菜单开着吗。开关时机由外壳定（Esc / 点外面 / 滚动 / 操作完）。 */
  menuOpen: boolean;
  onOpenMenu: () => void;
  onCloseMenu: () => void;
}

export function BlockHandle({ hover, blockHandle, menuOpen, onOpenMenu, onCloseMenu }: BlockHandleProps) {
  const [overGrip, setOverGrip] = useState(false);
  const [dragging, setDragging] = useState(false);
  const [dropLine, setDropLine] = useState<SelectionRect | null>(null);
  /** 拖动中挂在 `window` 上的两个监听，卸载时要收掉。 */
  const teardownRef = useRef<(() => void) | null>(null);

  /**
   * 最后一次**非 null** 的悬停状态。
   *
   * 指针移到手柄上之后内核就报 `null` 了，但菜单还要画、位置还要摆 ——
   * 所以不能直接拿 `hover` 当渲染输入。写 ref 而不是 state：
   * 它只在 `hover` 变化时被读到，而 `hover` 变化本身就会触发重渲染。
   */
  const lastStateRef = useRef<BlockHandleState | null>(null);
  if (hover !== null) lastStateRef.current = hover;
  const state = hover ?? lastStateRef.current;

  const visible = state !== null && (hover !== null || overGrip || menuOpen || dragging);

  // 卸载时把可能还挂着的拖动监听收掉（拖到一半切标签页）
  useEffect(
    () => () => {
      teardownRef.current?.();
      teardownRef.current = null;
    },
    [],
  );

  /**
   * 手柄「彻底不可交互」时，通知内核把目标块清掉。
   *
   * ## 判据是 `visible === false`，不是「菜单关了」
   *
   * 菜单关掉的那一刻指针**还在手柄上**（用户刚点完一项，或者刚拖完），
   * 他随时会再点一次。这时就把目标清掉的话，第二次打开的菜单里
   * 每一个动作都会作用在「没有目标」上 —— 表现为**点了没反应**，
   * 而且完全看不出原因（菜单画得好好的）。
   *
   * 所以清得晚一点：等指针离开手柄、又不在任何一块上时才清。
   * 这正是「内核记目标块、外壳决定什么时候不要它」这条分工的落点。
   *
   * 组件卸载也要清（切标签页 / 文档变成只读）。
   */
  useEffect(() => {
    if (!visible) {
      blockHandle.close();
      return;
    }
    return () => blockHandle.close();
  }, [visible, blockHandle]);

  const gripAt = useMemo(() => {
    if (state === null) return null;
    const { anchor } = state;
    const maxTop = Math.max(FLOATING_MARGIN, window.innerHeight - GRIP_SIZE - FLOATING_MARGIN);
    return {
      // 贴在手柄**右侧**离块左边缘一小段 —— 正文栏的 padding 就是留给它的
      left: Math.max(FLOATING_MARGIN, anchor.left - GRIP_OFFSET),
      // 与块的**首行**对齐。块很高时（比如一张长表格）手柄留在首行，
      // 跟着块中心跑会让它在滚动中漂走
      top: Math.min(Math.max(anchor.top, FLOATING_MARGIN), maxTop),
    };
  }, [state]);

  const onGripPointerDown = useCallback(
    (event: ReactPointerEvent<HTMLButtonElement>) => {
      if (state === null) return;
      // 不拦这一下，拖动会顺手把正文选中一片（浏览器默认行为）
      event.preventDefault();

      const startX = event.clientX;
      const startY = event.clientY;
      let moved = false;

      const onMove = (move: PointerEvent) => {
        if (!moved) {
          const far = Math.abs(move.clientX - startX) >= DRAG_THRESHOLD || Math.abs(move.clientY - startY) >= DRAG_THRESHOLD;
          if (!far) return;
          moved = true;
          setDragging(true);
        }
        // 落点由内核算 —— 外壳不知道文档坐标，也算不出来
        setDropLine(blockHandle.dropIndicatorAt(move.clientX, move.clientY));
      };

      const stop = () => {
        window.removeEventListener('pointermove', onMove);
        window.removeEventListener('pointerup', onUp);
        window.removeEventListener('pointercancel', onCancel);
        teardownRef.current = null;
      };

      const onUp = (up: PointerEvent) => {
        stop();
        if (!moved) {
          // 没动过 = 一次点击。在这里当场判，不等 `click`（理由见文件头）
          if (menuOpen) onCloseMenu();
          else onOpenMenu();
          return;
        }
        blockHandle.dropAt(up.clientX, up.clientY);
        setDragging(false);
        setDropLine(null);
        // 操作完就收起菜单（与 Typora 一致）。不关的话菜单会停在一个
        // 已经不指着任何东西的位置上，用户下一次点击会作用在别处
        onCloseMenu();
      };

      const onCancel = () => {
        stop();
        setDragging(false);
        setDropLine(null);
      };

      window.addEventListener('pointermove', onMove);
      window.addEventListener('pointerup', onUp);
      window.addEventListener('pointercancel', onCancel);
      teardownRef.current = stop;
    },
    [state, blockHandle, onCloseMenu, onOpenMenu, menuOpen],
  );

  /**
   * 键盘激活（Enter / Space）也走 `click`，而它**没有** `pointerdown` ——
   * 指针那条路已经在 `pointerup` 里处理过了，这里用 `detail` 把它们分开：
   * 键盘产生的 `click` 点击次数是 0。
   */
  const onGripClick = useCallback(
    (event: ReactMouseEvent<HTMLButtonElement>) => {
      if (event.detail !== 0) return;
      if (menuOpen) onCloseMenu();
      else onOpenMenu();
    },
    [menuOpen, onOpenMenu, onCloseMenu],
  );

  if (!visible || gripAt === null || state === null) return null;

  const gripRect: SelectionRect = {
    top: gripAt.top,
    left: gripAt.left,
    bottom: gripAt.top + GRIP_SIZE,
    right: gripAt.left + GRIP_SIZE,
    width: GRIP_SIZE,
    height: GRIP_SIZE,
  };

  return (
    <>
      <button
        type="button"
        className={`bh-grip${menuOpen ? ' open' : ''}${dragging ? ' dragging' : ''}`}
        style={{ left: gripAt.left, top: gripAt.top, width: GRIP_SIZE, height: GRIP_SIZE }}
        aria-haspopup="menu"
        aria-expanded={menuOpen}
        onPointerDown={onGripPointerDown}
        onMouseEnter={() => setOverGrip(true)}
        onMouseLeave={() => setOverGrip(false)}
        onClick={onGripClick}
      >
        <IconGrip />
        {/* 可访问名 + 测试按文字找得到（与活动栏同一条约定） */}
        <span className="sr-only">{t('ui.blockhandle.grip')}</span>
      </button>

      {dropLine !== null && (
        <div
          className="bh-drop-line"
          style={{ left: dropLine.left, width: dropLine.width, top: dropLine.top }}
          aria-hidden="true"
        />
      )}

      {menuOpen && <BlockMenu state={state} blockHandle={blockHandle} anchor={gripRect} onClose={onCloseMenu} />}
    </>
  );
}

/** 菜单里的一项。 */
interface MenuEntry {
  readonly id: string;
  /** 分组序号。相邻且相同的不画分隔线。 */
  readonly group: number;
  /** 渲染时才 `t()` —— `entries` 是 memo，把译文存进去会在换语言后过期。 */
  readonly labelKey: MessageKey;
  readonly hint: string;
  readonly disabled: boolean;
  readonly run: () => void;
}

interface BlockMenuProps {
  state: BlockHandleState;
  blockHandle: EditorBlockHandle;
  anchor: SelectionRect;
  onClose: () => void;
}

function BlockMenu({ state, blockHandle, anchor, onClose }: BlockMenuProps) {
  const rootRef = useRef<HTMLDivElement | null>(null);
  const [placement, setPlacement] = useState<{ left: number; top: number } | null>(null);
  const [highlighted, setHighlighted] = useState(0);

  const entries = useMemo<MenuEntry[]>(() => {
    const list: MenuEntry[] = [];

    if (state.canTurnInto) {
      for (const item of TURN_INTO) {
        // 「已经是这个格式」显示成禁用，而不是点了没反应。
        // 内核那边也会挡一次（`toggleBlock` 对同格式是**取消**语义）
        const current =
          state.block === item.block && (item.block !== 'heading' || state.headingLevel === (item.level ?? 1));
        list.push({
          id: `turn-${item.id}`,
          group: 0,
          labelKey: item.labelKey,
          hint: item.hint,
          disabled: current,
          run: () => blockHandle.turnInto(item.block, item.level),
        });
      }
    }

    list.push({
      id: 'moveUp',
      group: 1,
      labelKey: 'ui.blockhandle.moveUp',
      hint: '',
      disabled: !state.canMoveUp,
      run: () => blockHandle.run('moveUp'),
    });
    list.push({
      id: 'moveDown',
      group: 1,
      labelKey: 'ui.blockhandle.moveDown',
      hint: '',
      disabled: !state.canMoveDown,
      run: () => blockHandle.run('moveDown'),
    });
    list.push({
      id: 'duplicate',
      group: 1,
      labelKey: 'ui.blockhandle.duplicate',
      hint: '',
      disabled: false,
      run: () => blockHandle.run('duplicate'),
    });
    list.push({
      id: 'delete',
      group: 1,
      labelKey: 'ui.blockhandle.delete',
      hint: '',
      disabled: !state.canDelete,
      run: () => blockHandle.run('delete'),
    });

    return list;
  }, [state, blockHandle]);

  useLayoutEffect(() => {
    const node = rootRef.current;
    if (node === null) return;
    const { width, height } = node.getBoundingClientRect();
    // 摆在手柄**右侧偏下**：手柄贴在正文栏左边距里，菜单往右长才落在正文上，
    // 往左会长到窗口外面去
    setPlacement(
      computeFloatingPlacement(
        anchor,
        { width, height },
        { vertical: 'below', horizontal: 'start' },
        { width: window.innerWidth, height: window.innerHeight },
      ),
    );
  }, [anchor]);

  /** 从 `at` 出发找下一个可用的项。全都禁用时停在原地。 */
  const stepEnabled = useCallback(
    (at: number, step: number): number => {
      const count = entries.length;
      if (count === 0) return at;
      for (let i = 1; i <= count; i += 1) {
        // 取模两次：`step` 为负时 JS 的 `%` 会给出负数，直接拿去当下标是 undefined
        const next = (((at + step * i) % count) + count) % count;
        if (entries[next]?.disabled === false) return next;
      }
      return at;
    },
    [entries],
  );

  const apply = useCallback(
    (entry: MenuEntry) => {
      if (entry.disabled) return;
      entry.run();
      // 操作完就收。不收的话菜单会停在一个已经不指着任何东西的位置上
      onClose();
    },
    [onClose],
  );

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      // 输入法组合期间不抢键
      if (event.isComposing) return;

      if (event.key === 'Escape') {
        event.preventDefault();
        event.stopPropagation();
        onClose();
        return;
      }

      if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
        event.preventDefault();
        event.stopPropagation();
        setHighlighted((at) => stepEnabled(at, event.key === 'ArrowDown' ? 1 : -1));
        return;
      }

      if (event.key === 'Enter') {
        const entry = entries[highlighted];
        if (entry === undefined || entry.disabled) return;
        event.preventDefault();
        event.stopPropagation();
        apply(entry);
      }
    };

    // `true` = 捕获阶段，理由见文件头
    window.addEventListener('keydown', onKeyDown, true);
    return () => window.removeEventListener('keydown', onKeyDown, true);
  }, [entries, highlighted, apply, onClose, stepEnabled]);

  useEffect(() => {
    const onPointerDown = (event: MouseEvent) => {
      const node = rootRef.current;
      if (node === null || !(event.target instanceof Node) || node.contains(event.target)) return;
      onClose();
    };
    document.addEventListener('mousedown', onPointerDown);
    return () => document.removeEventListener('mousedown', onPointerDown);
  }, [onClose]);

  useEffect(() => {
    const onScroll = (event: Event) => {
      const node = rootRef.current;
      // 菜单自己也会滚（项多时）。捕获阶段收得到它，得排掉
      if (node !== null && event.target instanceof Node && node.contains(event.target)) return;
      onClose();
    };
    // 捕获阶段：`scroll` 不冒泡，挂在 `window` 上只有捕获才收得到
    window.addEventListener('scroll', onScroll, true);
    return () => window.removeEventListener('scroll', onScroll, true);
  }, [onClose]);

  return (
    <div
      ref={rootRef}
      className="bh-menu"
      role="menu"
      aria-label={t('ui.blockhandle.grip')}
      style={{
        left: placement?.left ?? 0,
        top: placement?.top ?? 0,
        visibility: placement === null ? 'hidden' : 'visible',
      }}
    >
      {entries.map((entry, index) => {
        const first = index === 0;
        const newGroup = !first && entries[index - 1]?.group !== entry.group;
        return (
          <button
            key={entry.id}
            type="button"
            role="menuitem"
            disabled={entry.disabled}
            className={`bh-item${index === highlighted ? ' highlighted' : ''}${newGroup ? ' separated' : ''}`}
            // 不拦这一下，焦点会离开编辑器、接着触发「点菜单外面」的监听 ——
            // 菜单会在 click 到达按钮之前被卸载（与斜杠菜单同一个坑）
            onMouseDown={(event) => event.preventDefault()}
            onMouseEnter={() => setHighlighted(index)}
            onClick={() => apply(entry)}
          >
            <span className="bh-label">{t(entry.labelKey)}</span>
            {entry.hint === '' ? null : <span className="bh-hint">{entry.hint}</span>}
          </button>
        );
      })}
    </div>
  );
}
