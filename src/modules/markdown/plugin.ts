/**
 * Markdown 内核的三个自研 ProseMirror 插件。
 *
 * 为什么自己写而不找现成包：Milkdown 生态里做代码高亮的包长期跟不上主版本
 * （`@milkdown/plugin-math` 已被官方标记为不再支持，同类插件也有版本错位问题）。
 * 这三个插件一共百来行，用标准 ProseMirror API 写出来比引入一个会随主版本漂移的依赖更稳。
 */

import { Plugin, PluginKey } from '@milkdown/kit/prose/state';
import type { Node as ProseNode } from '@milkdown/kit/prose/model';
import { Decoration, DecorationSet } from '@milkdown/kit/prose/view';
import { $prose } from '@milkdown/kit/utils';

import type {
  BlockHandleState,
  CursorPosition,
  EditorPosition,
  SelectionSnapshot,
  SlashState,
  TableState,
} from '../../core/editor/types.js';
import type { EditorView } from '@milkdown/kit/prose/view';
import { highlightCode } from '../../core/highlight/hljs.js';
import { readScrollRatio, typewriterScrollTop } from '../../core/editor/position.js';
import { blockIndexFromElement, readBlockAt, withinGutter } from './block.js';
import { anchorOf, caretAnchorOf, describeSelection, describeSlash, describeTable } from './selection.js';

const highlightKey = new PluginKey<DecorationSet>('levnote-code-highlight');
const cursorKey = new PluginKey('levnote-cursor-reporter');
const positionKey = new PluginKey('levnote-position-reporter');
const selectionKey = new PluginKey('levnote-selection-reporter');
const slashKey = new PluginKey('levnote-slash-reporter');
const tableKey = new PluginKey('levnote-table-reporter');
const blockHoverKey = new PluginKey('levnote-block-hover');
const emitKey = new PluginKey('levnote-markdown-emit');
const placeholderKey = new PluginKey('levnote-placeholder');

/** ProseMirror 的 code_block 节点名与语言属性名（对齐 preset-commonmark 的 schema）。 */
const CODE_BLOCK = 'code_block';

/**
 * 把每个 `code_block` 的内容按 highlight.js 的结果套上 inline 装饰。
 *
 * 只处理 code_block：行内代码不套色 —— 密集段落里行内代码套色会很吵，而且 Typora 也不做。
 *
 * 装饰是**纯视觉**的，不进入文档内容，因此不影响 markdownUpdated 的序列化结果。
 * 高亮只在「导出的 HTML」里才需要成为真实结构，那一步走的是另一条路（highlightCode → <span>）。
 */
function buildDecorations(doc: ProseNode): DecorationSet {
  const decorations: Decoration[] = [];

  doc.descendants((node, pos) => {
    if (node.type.name !== CODE_BLOCK) return;
    const text = node.textContent;
    if (text.trim() === '') return;

    const raw = node.attrs['language'];
    const language = typeof raw === 'string' && raw !== '' ? raw : undefined;
    const result = highlightCode(text, language);
    if (!result) return;

    // +1 跳过 code_block 自身的位置标记，进到它内部文本的坐标空间
    const base = pos + 1;
    for (const token of result.tokens) {
      // inclusiveEnd 保持默认的 false：装饰到 token 末尾即止。
      // 这里不用管「会不会吃掉换行」—— 装饰不参与文档内容，改不到换行。
      decorations.push(Decoration.inline(base + token.from, base + token.to, { class: token.className }));
    }
  });

  return DecorationSet.create(doc, decorations);
}

export const codeHighlightPlugin = $prose(
  () =>
    new Plugin<DecorationSet>({
      key: highlightKey,
      state: {
        init: (_config, state) => buildDecorations(state.doc),
        apply: (tr, value) => (tr.docChanged ? buildDecorations(tr.doc) : value.map(tr.mapping, tr.doc)),
      },
      props: {
        decorations: (state) => highlightKey.getState(state),
      },
    }),
);

/** 从 ProseMirror 文档位置换算出行列号（1 基）。 */
function positionOf(doc: ProseNode, pos: number): CursorPosition {
  const safe = Math.max(0, Math.min(pos, doc.content.size));
  const before = doc.textBetween(0, safe, '\n', '\n');
  const lines = before.split('\n');
  return { line: lines.length, column: (lines[lines.length - 1]?.length ?? 0) + 1 };
}

/**
 * 光标位置上报。
 *
 * 用「取回调的函数」而不是直接传回调：插件实例在编辑器创建时就固定了，
 * 而 React 那边每次渲染都会换一个新的回调函数。中间隔一层取值，插件才不会拿着过期的闭包。
 */
export function cursorReporterPlugin(getReporter: () => ((position: CursorPosition) => void) | undefined) {
  return $prose(
    () =>
      new Plugin({
        key: cursorKey,
        view: (editorView) => {
          const report = (view: typeof editorView) => {
            getReporter()?.(positionOf(view.state.doc, view.state.selection.head));
          };
          report(editorView);
          return {
            // 第二个参数是**上一个 state**，不是上一个 view —— 这一点容易写错。
            // 只有选择真正变了才上报，避免每次输入都触发一次无关的状态更新。
            update: (view, prevState) => {
              if (view.state.selection !== prevState.selection) report(view);
            },
          };
        },
      }),
  );
}

// ---------------------------------------------------------------------------
// 位置记忆上报
// ---------------------------------------------------------------------------

/**
 * 位置（光标 + 滚动）上报。
 *
 * ## 为什么滚动要在这里挂，而不是外壳去挂
 *
 * 滚动容器是 `.qm-editor` —— **内核自己造的那个根节点**（`options.root`）。
 * 外壳要自己去监听它，就得按类名认内核画的 DOM，那是硬约束 11 禁止的事。
 * 内核手里正好有这个引用，所以由它挂。
 *
 * 用 `passive: true`：滚动监听里没有任何 `preventDefault`，声明被动能让浏览器
 * 不必等回调跑完就能继续滚 —— 对滚动流畅度是实打实的差别。
 *
 * ## 上报频率刻意不设防抖
 *
 * 一次上报只是读三个 O(1) 的字段（`selection.head` / `doc.content.size` /
 * `scrollTop`）然后调一个写 ref 的回调，代价可以忽略。防抖放在内核里反而有害：
 * 内核的定时器在销毁时不会自动清，得额外维护生命周期。落盘那一侧的防抖在外壳，
 * 那里才知道「写一次 localStorage」到底有多贵。
 */
export function positionReporterPlugin(
  root: HTMLElement,
  getReporter: () => ((position: EditorPosition) => void) | undefined,
) {
  return $prose(
    () =>
      new Plugin({
        key: positionKey,
        view: (editorView) => {
          const report = (): void => {
            const reporter = getReporter();
            if (!reporter) return;
            reporter({
              offset: editorView.state.selection.head,
              // O(1)：只读一个算好的尺寸，不遍历文档。
              // 位置每次光标移动都要读一次，O(n) 的写法在大文档上会拖慢打字。
              textLength: editorView.state.doc.content.size,
              scrollRatio: readScrollRatio(root),
            });
          };

          root.addEventListener('scroll', report, { passive: true });
          return {
            update: (view, prevState) => {
              if (view.state.selection !== prevState.selection) report();
            },
            destroy: () => root.removeEventListener('scroll', report),
          };
        },
      }),
  );
}

// ---------------------------------------------------------------------------
// 浮层上报（选中工具条 / 斜杠菜单）
// ---------------------------------------------------------------------------

/**
 * 把「量一次位置」推迟到这一帧画完。
 *
 * 为什么必须推迟：`coordsAtPos` 会强制浏览器立刻结算布局。它被调用的位置是
 * ProseMirror 的 `view.update`，也就是**事务派发的过程中** —— 此时布局是脏的，
 * 在这里读一次就会触发一次同步重排，而拖选时每个 mousemove 都要走一遍，
 * 结果是拖选手感直接垮掉。
 *
 * 顺带把一帧内的多次请求合并成一次：拖选时 ProseMirror 可能在一次事件里
 * 派发多条事务（输入规则、追加事务），逐条去量纯属浪费。
 *
 * jsdom 不保证提供 `requestAnimationFrame`，所以有一条同步兜底 ——
 * 与 `core/theme/theme.ts` 对 `matchMedia` 的处理是同一条规矩：
 * 环境缺什么就在缺的地方兜住，而不是让整个模块挂掉。
 */
function createFrameScheduler(run: () => void): { schedule: () => void; cancel: () => void } {
  if (typeof requestAnimationFrame !== 'function') {
    return { schedule: run, cancel: () => undefined };
  }

  let frame = 0;
  return {
    schedule: () => {
      if (frame !== 0) return;
      frame = requestAnimationFrame(() => {
        frame = 0;
        run();
      });
    },
    cancel: () => {
      if (frame !== 0) cancelAnimationFrame(frame);
      frame = 0;
    },
  };
}

/**
 * 「读一次浮层状态、报给外壳」这套骨架。
 *
 * 选中工具条与斜杠菜单的**机制完全一样**，只有「读什么」不同：
 * 都用「取回调的函数」拿回调（插件实例在编辑器创建时就固定了，而 React
 * 每次渲染都换一个新函数）、都要合并到帧、都要跟滚动。
 * 所以只有 `read` 是参数，其余共用 —— 免得两处各写一遍之后，
 * 某天只给其中一处修了 bug。
 *
 * ## 为什么还要监听滚动
 *
 * 滚动**不改变选区**，但会改变选区在屏幕上的位置。只在 selection 变化时上报的话，
 * 用户选中一段文字再滚动，浮层会留在原地和文字分家 —— 看起来像浮层飘走了。
 *
 * ## 为什么不在失焦时上报 null
 *
 * 用户点浮层上的按钮，浏览器会先把焦点从编辑器挪走。一失焦就上报 null，
 * 浮层会在 `click` 事件到达按钮之前被卸载 —— 表现为「按钮点不动」。
 * 所以只在**选区塌缩**（用户点掉了选中）时上报 null。
 */
function floatingReporterPlugin<T>(
  key: PluginKey,
  read: (view: EditorView) => T | null,
  getReporter: () => ((value: T | null) => void) | undefined,
): Plugin {
  return new Plugin({
    key,
    view: (editorView) => {
      const scheduler = createFrameScheduler(() => {
        const report = getReporter();
        if (!report) return;
        // 读不出状态、或位置量不出来，都报 null（把浮层藏起来）。
        // 「有没有内容」和「有没有位置」对外壳来说是同一个问题的两面 ——
        // 一个定位在瞎猜坐标上的浮层比不显示更糟
        report(read(editorView));
      });

      // 滚动容器是 contenteditable 的父节点（`.qm-editor`，见 markdown.css）。
      // 用 parentElement 而不是写死类名：类名改了这里不该跟着坏。
      const scroller = editorView.dom.parentElement;
      scroller?.addEventListener('scroll', scheduler.schedule, { passive: true });

      scheduler.schedule();

      return {
        update: (view, prevState) => {
          // 第二个参数是**上一个 state**，不是上一个 view —— 这一点容易写错。
          //
          // 三件事都会让浮层的内容过时，缺一不可：
          // - 选区变了：位置和「选中了什么」都变了；
          // - `storedMarks` 变了：切换格式后还没打字时，格式存在这里；
          // - **文档变了**：点工具条上的「加粗」**不会移动选区**，
          //   只是给选区里的文字加了一个 mark —— 只看选区的话，
          //   按钮会在点完之后一直停在旧的高亮上，用户以为没生效。
          //   这里比的是对象标识（O(1)），不是 `doc.eq()`（O(文档长度)）。
          const stale =
            view.state.selection !== prevState.selection ||
            view.state.storedMarks !== prevState.storedMarks ||
            view.state.doc !== prevState.doc;
          if (stale) scheduler.schedule();
        },
        destroy: () => {
          scheduler.cancel();
          scroller?.removeEventListener('scroll', scheduler.schedule);
        },
      };
    },
  });
}

/** 选中工具条的上报：选区格式 + 选区包围盒。 */
export function selectionReporterPlugin(
  getReporter: () => ((selection: SelectionSnapshot | null) => void) | undefined,
) {
  return $prose(() =>
    floatingReporterPlugin<SelectionSnapshot>(
      selectionKey,
      (view) => {
        const format = describeSelection(view.state);
        if (format === null) return null;
        const anchor = anchorOf(view);
        return anchor === null ? null : { format, anchor };
      },
      getReporter,
    ),
  );
}

/** 斜杠菜单的上报：`/` 后面已经输入的文字 + 段落内容起点的位置。 */
export function slashReporterPlugin(getReporter: () => ((state: SlashState | null) => void) | undefined) {
  return $prose(() =>
    floatingReporterPlugin<SlashState>(
      slashKey,
      (view) => {
        const query = describeSlash(view.state);
        if (query === null) return null;
        // 锚在**段落内容起点**而不是光标处：光标会随着筛选词变长一直往右跑，
        // 菜单跟着飘会很难受
        const anchor = caretAnchorOf(view, view.state.selection.$from.start());
        return anchor === null ? null : { query, anchor };
      },
      getReporter,
    ),
  );
}

/**
 * 表格工具条的上报：整张表格的包围盒 + 行列数 + 对齐方式。
 *
 * 与选中工具条共用同一套骨架（`floatingReporterPlugin`），所以这里只有一行 ——
 * 三处浮层的差别全在 `read` 里。这正是把骨架抽出来的收益。
 */
export function tableReporterPlugin(getReporter: () => ((state: TableState | null) => void) | undefined) {
  return $prose(() => floatingReporterPlugin<TableState>(tableKey, describeTable, getReporter));
}

// ---------------------------------------------------------------------------
// 块手柄
// ---------------------------------------------------------------------------

/**
 * 「指针停在哪一块上」的上报 —— 块手柄的输入。
 *
 * ## 它与另外四个浮层的上报骨架**不能共用**
 *
 * `floatingReporterPlugin` 的触发源是「选区 / 文档 / storedMarks 变了」加滚动，
 * 而这一个的触发源是**鼠标移动** —— 鼠标移动完全不碰 state，
 * 所以那个骨架一辈子都不会被唤醒。共用只能是硬塞。
 *
 * ## 滚动必须重算，而且要用**上一次的指针位置**
 *
 * 滚动不产生 `mousemove`，所以指针位置不变而它底下的块变了 ——
 * 不重算的话，用户滚一下轮子，手柄就指着原来的那一块（现在已经不是指针下的那一块了）。
 *
 * ## 「指针指着哪一块」走 DOM 元素，不走坐标
 *
 * 两条独立的原因（完整版在 `block.ts` 的 `blockIndexFromElement`）：
 * 坐标命中依赖 `elementFromPoint`，而 **jsdom 没有它** —— 走坐标等于
 * 把「上移 / 删除 / 转为」全部命令都挡在测试之外；而且坐标命中答的是
 * 「最近的一个文本位置」，不是「指针真的压在这一块上吗」。
 *
 * 拖动算落点**反过来**必须用坐标：那时指针下面压着的是手柄自己。
 *
 * ## 目标块（`target.index`）与上报值是**两件事**
 *
 * 上报值（这里交给 `getReporter`）是「指针现在在哪一块上」，指针离开编辑器就是 null；
 * 而 `target.index` 是「手柄命令该作用在哪一块」，**只在 `close()` 时清**。
 * 必须分开：用户要把鼠标从正文移到手柄上才能点它，而手柄在编辑器外面 ——
 * 一离开就清目标，菜单上的每一个动作都会作用在「没有目标」上，
 * 表现为**点了没反应**。
 *
 * ## 三处容易做错的地方
 *
 * 1. **按住鼠标时不许换目标块。** 拖动重排期间 `mousemove` 照常发，
 *    不挡的话指针扫过哪一块、`target.index` 就变成哪一块 ——
 *    用户拖的是第 3 块，松手却把第 7 块搬走了。
 *    判据用 `event.buttons !== 0`（一次契约都不用加）。
 * 2. **指针走到左边那条留白里要保住目标块。** 手柄摆在那儿，
 *    纯按元素命中算会在中途报 null，手柄被卸载 → 永远点不到。
 *    兜底在 `withinGutter`。
 * 3. **滚动时指针坐标不变、底下的块变了。** 用上一次的指针位置重算 ——
 *    坐标在这一版里只用来做留白兜底，所以存下来就够。
 */
export function blockHoverPlugin(
  getReporter: () => ((state: BlockHandleState | null) => void) | undefined,
  target: { index: number | null },
) {
  return $prose(() => {
    /**
     * 最后一次已知的指针位置（视口坐标）。只用来做留白兜底。
     *
     * 放在工厂作用域而不是插件实例上：`handleDOMEvents` 与 `view` 是**两个
     * 不同的 props 字段**，它们都要读写同一份状态，只有闭包能共享。
     */
    let pointer: { x: number; y: number } | null = null;
    /**
     * 这一次重算该用哪个下标。
     *
     * - `undefined` = 「不是指针移动触发的」（滚动 / 文档变化）→ 沿用 `target.index`
     * - `null` = 「指针移动了，但没压在任何一块上」
     * - 数字 = 指针移动后命中的那一块
     *
     * 元素命中在 `mousemove` 里**当场**算掉，而不是推到帧里 —— 推到帧里就得
     * 额外存一个 `EventTarget` 加一个布尔标记，而 `posAtDOM` 只走 docView 树、
     * 不触发布局，当场的代价可以忽略。
     */
    let pending: number | null | undefined;
    /** 最近一次上报出去的那一份（滚动重算时要拿它的 `anchor` 做留白判断）。 */
    let current: BlockHandleState | null = null;
    /** `view` 回调跑之前为 null（`props` 会先被读到）。 */
    let scheduler: ReturnType<typeof createFrameScheduler> | null = null;

    return new Plugin({
      key: blockHoverKey,
      props: {
        // 全部返回 false = 只旁听，不拦截事件
        handleDOMEvents: {
          mousemove: (view, event) => {
            // 按下鼠标 = 正在拖动 / 正在选文字：这一段里目标块必须冻住
            if (event.buttons !== 0) return false;
            pointer = { x: event.clientX, y: event.clientY };
            pending = blockIndexFromElement(view, event.target);
            scheduler?.schedule();
            return false;
          },
          mouseleave: () => {
            pointer = null;
            pending = undefined;
            scheduler?.schedule();
            return false;
          },
        },
      },
      view: (editorView) => {
        const run = createFrameScheduler(() => {
          const report = getReporter();
          if (!report) return;

          let index: number | null = null;
          if (pointer !== null) {
            index = pending !== undefined ? pending : target.index;
            // 元素命中不到（指针在块左边的留白里 —— 手柄就摆在那儿）时，
            // 只要还在当前这一块的横向带子里，就继续算作指着它。
            // 少了这一条，用户从正文移向手柄的中途手柄就被卸载了 ——
            // 那几像素里「压着哪个块」的答案是没有块，而手柄恰好在其中
            if (index === null && target.index !== null && current !== null && withinGutter(current.anchor, pointer.x, pointer.y)) {
              index = target.index;
            }
          }

          const next = index === null ? null : readBlockAt(editorView, index);
          if (next === null) {
            target.index = null;
            current = null;
            report(null);
            return;
          }

          target.index = index;
          // 同一块、同一几何就不重复上报：外壳那边每次上报都会 setState，
          // 而鼠标一动就是一次，不挡的话每次移动都要重渲染一遍整个面板
          if (current !== null && sameHover(current, next)) return;
          current = next;
          report(next);
        });
        scheduler = run;

        // 滚动容器是 contenteditable 的父节点（`.qm-editor`）。
        // 用 `parentElement` 而不是写死类名：类名改了这里不该跟着坏
        const scroller = editorView.dom.parentElement;
        const onScroll = () => {
          // 坐标没变、只是几何变了：沿用当前那一块，重算它的位置
          pending = undefined;
          run.schedule();
        };
        scroller?.addEventListener('scroll', onScroll, { passive: true });

        return {
          update: (view, prevState) => {
            if (view.state.doc === prevState.doc) return;
            // 文档变了：下标可能已经指到别的地方去了，重新算一次。
            // 指针不在编辑器里就**清掉**目标 —— 算不出来还留着，
            // 菜单上的「删除」会删掉一块用户根本没在看的块
            if (pointer === null) {
              target.index = null;
              current = null;
            }
            // 同滚动：指针没动，沿用 `target.index` 重算几何
            pending = undefined;
            run.schedule();
          },
          destroy: () => {
            run.cancel();
            scroller?.removeEventListener('scroll', onScroll);
            scheduler = null;
            current = null;
          },
        };
      },
    });
  });
}

/** 两次悬停状态是不是同一块、同一位置。 */
function sameHover(a: BlockHandleState, b: BlockHandleState): boolean {
  return (
    a.anchor.top === b.anchor.top &&
    a.anchor.left === b.anchor.left &&
    a.block === b.block &&
    a.headingLevel === b.headingLevel &&
    a.canMoveUp === b.canMoveUp &&
    a.canMoveDown === b.canMoveDown &&
    a.canDelete === b.canDelete &&
    a.canTurnInto === b.canTurnInto
  );
}

// ---------------------------------------------------------------------------
// 变更上报
// ---------------------------------------------------------------------------

/**
 * 标记「这次文档变更是程序写入的，别报给外壳」。
 * 由 `setup.ts` 的 `setMarkdown` 打在事务上。
 */
export const SILENT_META = 'levnote-silent';

/**
 * 标记「这次变更是程序写入的，但**是用户发起的**，要报给外壳」。
 *
 * 为什么需要第三个状态：`SILENT_META` 与「用户输入」之外还有一类事务 ——
 * **由用户动作触发、但不是键盘输入产生的编辑**。最典型的是查找替换：
 * 用户点「全部替换」，内容确实变了、文档确实该变脏、该能保存，
 * 但它不经过任何 `input` / `paste` 事件，`armedByUser` 永远是 false。
 *
 * 少了这个标记的后果不是「少报一次变更」，而是**静默丢数据**：
 * 外壳的 `doc.text` 停在替换前的内容上，用户按 Ctrl+S 保存，
 * 写回磁盘的是替换前的旧文本 —— 用户看到「已保存」，
 * 而他刚做的替换在磁盘上根本不存在。
 */
export const USER_EDIT_META = 'levnote-user-edit';

/**
 * 把文档变更序列化成 markdown，上报给外壳。
 *
 * ## 为什么不用 `@milkdown/plugin-listener`
 *
 * 它的 `markdownUpdated` 是**防抖 200ms** 才触发的（`debounce(..., 200)`）。
 * 防抖恰好让「这次变更是程序写入的」这个判断在最需要它的时刻失效：
 * 我们在 dispatch 时设的标志，等到 200ms 后回调才去读，早就清掉了。
 * 护不住的结果是 —— 从磁盘重新加载后，内核把自己序列化（规范化）过的 markdown
 * 再报一次，外壳以为用户改了内容，于是一个字都没动的文档立刻显示「有未保存的改动」。
 * 用户被这个假信号引着去点保存，保存就把文件里他没碰过的排版改掉了 ——
 * 实测会被改的有：setext 标题（`标题\n====`）改成 ATX（`# 标题`）、
 * 行尾空白与连续空行被收敛、`~~~` 围栏改成 ```、`___` 分隔线改成 `---`、
 * 硬换行的两个行尾空格改成反斜杠、**表格按列宽重排**。都是语义等价的改写，
 * 但**字节确实变了**。
 *
 * 符号类（列表符号 / 分隔线 / 围栏）已经统一到**最常见的写法**（见 `setup.ts` 里
 * `remarkStringifyOptionsCtx` 那段注释）—— 用户用 `-` 写的列表不会被改成 `*` 了。
 * 但表格重排改不掉：那是 `remark-stringify` 的表格式化，没有开关。
 * **真正的解法是「只写回改动发生的位置」**（horseMD 的卖点），那要换架构。
 *
 * ## 判定规则：只有**用户交互之后**的变更才算用户改动
 *
 * 光看 `tr.docChanged` 不够 —— `@milkdown/plugin-trailing` 会在文档以
 * 代码块 / 列表 / 表格 / 分隔线结尾时，用 `appendTransaction` 补一个空段落。
 * 那也是一次 docChanged，但用户什么都没做。若把它当成用户改动，
 * 「打开一个以代码块结尾的文件」就会被立刻判成脏。
 *
 * 所以这里监听 DOM 层的输入事件，只在「刚发生过用户输入」的窗口内上报：
 * 打字、输入法上屏、粘贴、剪切、拖放、以及撤销/重做快捷键。
 * 其余一切（初始化、`setMarkdown`、trailing 补段落、任何插件自己的事务）都不上报。
 *
 * ## `beforeinput` 不能省 —— 它是唯一能保证「早于事务」的那个事件
 *
 * 一开始这里只监听 `input`，结果**用户敲的第一个字存不进磁盘**：界面显示「已保存」，
 * 文件里却没有那个字。原因是 ProseMirror **并不监听 `input` 事件** ——
 * 它是靠 `MutationObserver` 读 DOM 变更、自己派发事务的。
 * `input` 到达时那次事务可能早就派发完了（实测在 jsdom 里必然如此：
 * MutationObserver 的回调是微任务，先于任何事件派发跑完），
 * 于是 `armedByUser` 还是 false，这次输入被安静地丢掉。
 *
 * 而且它是**攒着的**：第一次丢掉之后 `arm()` 才把标志置上，
 * 于是从第二次输入开始一切正常 —— 这种「只错第一次」的 bug 极难在人工测试里发现。
 *
 * `beforeinput` 按规范在**修改 DOM 之前**派发，所以它一定早于那次事务。
 * `input` 保留着是因为它是补充（不是所有输入路径都发 `beforeinput`），
 * 而且置一个布尔值是幂等的。
 *
 * 这个规则还有个附带好处：上报是**同步**的（`state.apply` 在 `dispatch` 里跑），
 * 外壳拿到内容的时间和用户按键是同一个事件循环，不会出现「输入完 200ms 才变脏」的迟滞。
 */
export function markdownEmitPlugin(
  report: (markdown: string) => void,
  serialize: (doc: ProseNode) => string,
) {
  /** 用户交互已经发生，下一次文档变更可以上报。 */
  let armedByUser = false;

  const arm = () => {
    armedByUser = true;
  };

  return new Plugin({
    key: emitKey,
    props: {
      // 全部返回 false = 只是旁听，不拦截事件，ProseMirror 自己的处理照常进行。
      handleDOMEvents: {
        beforeinput: () => {
          arm();
          return false;
        },
        input: () => {
          arm();
          return false;
        },
        // 输入法（中文等）在组合期间不发 input，以上屏事件为准
        compositionend: () => {
          arm();
          return false;
        },
        paste: () => {
          arm();
          return false;
        },
        cut: () => {
          arm();
          return false;
        },
        drop: () => {
          arm();
          return false;
        },
      },
      handleKeyDown: (_view, event) => {
        // 撤销/重做走快捷键命令，不产生 input 事件，需要单独认出来
        const mod = event.ctrlKey || event.metaKey;
        if (mod && ['z', 'Z', 'y', 'Y'].includes(event.key)) arm();
        return false;
      },
    },
    state: {
      // 初始文档永远不上报：这也是「打开文件后一个字没改就不该是脏的」的保证
      init: () => false,
      apply: (tr, _value, _oldState, newState) => {
        if (!tr.docChanged) return false;
        const silent = tr.getMeta(SILENT_META) === true;
        const byUser = (armedByUser || tr.getMeta(USER_EDIT_META) === true) && !silent;
        // 无论报不报，这次机会都要用掉：一条事务对应一次交互
        armedByUser = false;
        if (byUser) report(serialize(newState.doc));
        return byUser;
      },
    },
  });
}

// ---------------------------------------------------------------------------
// 占位提示
// ---------------------------------------------------------------------------

/**
 * 空文档时显示占位文案。
 *
 * ProseMirror 的空文档并不是「没有节点」，而是「一个空的 paragraph」——
 * 所以 CSS 的 `:empty` 永远不成立，必须在插件里把这个状态标出来。
 * 标在 contenteditable 自己身上（`is-empty` + `data-placeholder`），
 * 由 `markdown.css` 画出来。挂在宿主元素上而不是插入伪节点，是为了不污染文档内容。
 */
export function placeholderPlugin(text: string) {
  const isEmpty = (doc: ProseNode): boolean =>
    doc.childCount === 1 && doc.firstChild?.type.name === 'paragraph' && doc.firstChild.content.size === 0;

  return new Plugin({
    key: placeholderKey,
    props: {
      attributes: (state) => ({
        'data-placeholder': text,
        // 类名只在空文档时加，避免 CSS 用 :not() 反复判断
        class: isEmpty(state.doc) ? 'is-empty' : '',
      }),
    },
  });
}

// ---------------------------------------------------------------------------
// 专注 / 打字机（视图偏好，P3.5 之后的补课项）
// ---------------------------------------------------------------------------

export interface ComposeMode {
  readonly focus: boolean;
  readonly typewriter: boolean;
}

export const composeKey = new PluginKey<ComposeMode>('levnote-compose');

/** 光标所在顶层块调淡其余块的类名。浓淡归 `markdown.css`（内核自己的样式表）。 */
export const DIMMED_CLASS = 'qm-dimmed';

/**
 * 把光标行滚到编辑区垂直正中。
 *
 * 滚动容器是内核的（`.qm-editor`，即 `rootCtx` 那个元素 —— 与位置记忆读的
 * 是同一个容器，硬约束 13 的滚动条款）。算术在
 * `core/editor/position.ts` 的 `typewriterScrollTop`，这里只负责取几何。
 * jsdom 没有布局（高度恒 0），`typewriterScrollTop` 会返回 null，测试里天然是空操作。
 */
function centerCaret(view: EditorView, scroller: HTMLElement): void {
  const coords = view.coordsAtPos(view.state.selection.head);
  if (!coords) return;
  const rect = scroller.getBoundingClientRect();
  const target = typewriterScrollTop(coords.top, rect.top, scroller.scrollTop, scroller.clientHeight);
  if (target !== null) scroller.scrollTop = target;
}

/**
 * 专注 / 打字机共用的插件。两个开关同住一个状态对象，因为它们
 * **由同一处外壳偏好推来、用同一个 meta 事务切换**，拆成两个插件
 * 只会得到两次 dispatch 与两份要同步的真相。
 *
 * - **专注**：非活跃顶层块挂 `qm-dimmed`（node 装饰，纯视觉，不进文档）。
 *   「活跃」= 块与选区相交；空选区即光标所在的那一块。
 * - **打字机**：选区一动（或开关刚打开）就把光标行居中。只读 `scrollTop`、
 *   不 dispatch 事务 —— 否则会「滚动 → 上报 → 再滚动」转不停。
 *
 * 装饰每次现算（顶层 `doc.forEach`，长文档也就几百个孩子），不走
 * `DecorationSet.map` 增量 —— 顶层块的生死本来就少，增量簿记反而更贵。
 */
export function composePlugin(getScroller: () => HTMLElement, initial: ComposeMode): Plugin<ComposeMode> {
  return new Plugin<ComposeMode>({
    key: composeKey,
    state: {
      init: () => initial,
      apply(tr, value) {
        const next = tr.getMeta(composeKey) as ComposeMode | undefined;
        return next !== undefined ? next : value;
      },
    },
    props: {
      decorations(state) {
        const mode = composeKey.getState(state) ?? initial;
        if (!mode.focus) return DecorationSet.empty;
        const { doc, selection } = state;
        const decorations: Decoration[] = [];
        let pos = 0;
        doc.forEach((node) => {
          const from = pos;
          const to = pos + node.nodeSize;
          pos = to;
          // 空选区 = 光标所在块；非空选区 = 与 [from, to) 区间相交的块
          const active = selection.empty
            ? selection.head >= from && selection.head < to
            : selection.from < to && selection.to > from;
          if (!active) decorations.push(Decoration.node(from, to, { class: DIMMED_CLASS }));
        });
        return DecorationSet.create(doc, decorations);
      },
    },
    view: () => ({
      update(updated, prevState) {
        const mode = composeKey.getState(updated.state) ?? initial;
        if (!mode.typewriter) return;
        const prev = composeKey.getState(prevState) ?? initial;
        const turnedOn = mode.typewriter !== prev.typewriter;
        const moved =
          !updated.state.selection.eq(prevState.selection) || updated.state.doc !== prevState.doc;
        if (turnedOn || moved) centerCaret(updated, getScroller());
      },
    }),
  });
}
