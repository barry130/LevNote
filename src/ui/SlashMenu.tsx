/**
 * 在空段落里打 `/` 时弹出的「插入什么块」菜单。
 *
 * ## 它与选中工具条是同一套分工
 *
 * 内核只上报**两件事**：`/` 后面已经输入的文字（用来筛选）和菜单该出现在哪儿
 * （段落内容起点的视口坐标）。菜单长什么样、有哪些项、选中哪一项、回车做什么，
 * 全部由外壳决定。理由与 `SelectionToolbar` 完全一致 ——
 * 内核不认识外壳的组件，外壳不认识 ProseMirror。
 *
 * ## 两个不那么显然的决定
 *
 * ### 1. 键盘必须挂在 `window` 的**捕获阶段**
 *
 * 打开菜单时焦点始终在 contenteditable 里（用户正在打字），所以 keydown 的
 * 事件目标是编辑器。ProseMirror 把 `handleKeyDown` 挂在 `view.dom` 上，
 * 也就是说**冒泡路径上它先到**：如果在 `window` 上用默认（冒泡）方式监听，
 * 等到我们收到 Enter 时，ProseMirror 已经把段落拆开了 —— 菜单会「插入了块，
 * 但也多出一个空行」。
 *
 * 捕获阶段从 `window` 往下走，先于目标元素，于是我们能先一步
 * `preventDefault()` + `stopPropagation()` 把这一下拿走。
 * `stopPropagation` 还顺带拦住了外壳自己的全局快捷键（`App.tsx` 也是
 * 在 `window` 上冒泡监听），否则 Esc 会顺手关掉查找栏。
 *
 * ### 2. Esc 只**收起**菜单，不删用户打下的字
 *
 * 不删的话有个坑：`/表` 还在文档里，内核下一帧仍会认为「处于斜杠触发状态」，
 * 菜单立刻又弹回来 —— 按 Esc 等于没反应。所以这里记住「哪一段筛选词被收起过」，
 * 只要筛选词没变就不再显示。用户接着打字（筛选词变了）菜单自然回来，
 * 而他按 Esc 想保留的那几个字一个都没丢。
 *
 * 反过来，删掉 `/表` 也能解决重弹问题，而且 Typora 就是这么做的。但那意味着
 * **一次 Esc 会改文档内容** —— 用户只是想「算了不插了」，不该付出这个代价。
 */

import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';

import { rankCommands } from '../core/commands/fuzzy.js';
import type { EditorFormatting, InsertableBlock, SlashState } from '../core/editor/types.js';
import { computeFloatingPlacement } from './floating.js';

/** 菜单与光标所在行之间的空隙。比工具条小一点 —— 菜单是「接着这一行」长出来的。 */
const GAP = 6;

interface SlashItem {
  readonly block: InsertableBlock;
  /** 左侧图标位。刻意用字形而不是图标组件 —— 不引入图标库。 */
  readonly abbr: string;
  readonly label: string;
  /** 右侧的补充说明，同时也是搜索命中面的一部分（不进 title）。 */
  readonly hint: string;
  /** 拼音 / 英文别名。中文标题没有「词首」概念，靠这个才能用 `biaoti` 搜到「标题」。 */
  readonly keywords: readonly string[];
}

/**
 * 菜单项清单。
 *
 * 顺序就是**默认展示顺序**（筛选词为空时）。按「用户最常插入的排前面」排：
 * 正文 / 标题 / 列表 / 引用 / 代码块 / 分隔线。
 *
 * 只到三级标题，与选中工具条一致：H4–H6 走命令面板。
 */
const SLASH_ITEMS: readonly SlashItem[] = [
  { block: 'paragraph', abbr: '¶', label: '正文', hint: '普通段落', keywords: ['zhengwen', 'duanluo', 'paragraph', 'text', 'p'] },
  { block: 'heading1', abbr: 'H1', label: '一级标题', hint: '#', keywords: ['yijibiaoti', 'biaoti', 'heading', 'h1'] },
  { block: 'heading2', abbr: 'H2', label: '二级标题', hint: '##', keywords: ['erjibiaoti', 'biaoti', 'heading', 'h2'] },
  { block: 'heading3', abbr: 'H3', label: '三级标题', hint: '###', keywords: ['sanjibiaoti', 'biaoti', 'heading', 'h3'] },
  { block: 'bulletList', abbr: '•', label: '无序列表', hint: '- ', keywords: ['wuxuliebiao', 'liebiao', 'bullet', 'ul', 'list'] },
  { block: 'orderedList', abbr: '1.', label: '有序列表', hint: '1. ', keywords: ['youxuliebiao', 'liebiao', 'ordered', 'ol', 'list'] },
  { block: 'blockquote', abbr: '❝', label: '引用', hint: '> ', keywords: ['yinyong', 'quote', 'blockquote'] },
  { block: 'codeBlock', abbr: '</>', label: '代码块', hint: '```', keywords: ['daimakuai', 'daima', 'code', 'fence'] },
  { block: 'mathBlock', abbr: '∑', label: '数学公式', hint: '$$', keywords: ['shuxue', 'gongshi', 'math', 'latex', 'tex', 'katex', 'formula', 'equation'] },
  { block: 'diagram', abbr: '◇', label: '图表', hint: 'mermaid', keywords: ['tubiao', 'tu', 'liuchengtu', 'diagram', 'mermaid', 'graph', 'chart', 'flowchart'] },
  { block: 'table', abbr: '▦', label: '表格', hint: '3 × 3', keywords: ['biaoge', 'biao', 'table', 'grid'] },
  { block: 'hr', abbr: '—', label: '分隔线', hint: '---', keywords: ['fengexian', 'fenge', 'hr', 'divider', 'rule'] },
];

export interface SlashMenuProps {
  state: SlashState;
  formatting: EditorFormatting;
}

export function SlashMenu({ state, formatting }: SlashMenuProps) {
  const { query, anchor } = state;
  const rootRef = useRef<HTMLDivElement | null>(null);

  /** 算好的摆放位置。`null` = 还没量过，先渲染成 `visibility: hidden`。 */
  const [placement, setPlacement] = useState<{ left: number; top: number } | null>(null);
  const [highlighted, setHighlighted] = useState(0);
  /** 被 Esc 收起的那一段筛选词。等于当前筛选词时不显示菜单。 */
  const [dismissed, setDismissed] = useState<string | null>(null);

  const rows = useMemo(
    () =>
      rankCommands(query, SLASH_ITEMS, (item) => ({
        title: item.label,
        keywords: item.keywords,
        id: item.block,
      })).map((hit) => hit.item),
    [query],
  );

  const visible = query !== dismissed;

  // 筛选词一变就回到第一项：不回的话，用户先翻了五下再改搜索词，
  // 高亮会停在一条和当前搜索毫无关系的项上
  useEffect(() => {
    setHighlighted(0);
  }, [query]);

  // 列表变短时把高亮收回范围内
  useEffect(() => {
    setHighlighted((at) => Math.min(at, Math.max(0, rows.length - 1)));
  }, [rows.length]);

  useLayoutEffect(() => {
    if (!visible) return;
    const node = rootRef.current;
    if (node === null) return;

    const { width, height } = node.getBoundingClientRect();

    // 优先摆在光标那一行的**下方**（斜杠菜单的常规位置），放不下才翻上去。
    // 与选中工具条相反 —— 那个默认在上方。摆放算术见 `floating.ts`
    setPlacement(
      computeFloatingPlacement(
        anchor,
        { width, height },
        { vertical: 'below', horizontal: 'start', gap: GAP },
        { width: window.innerWidth, height: window.innerHeight },
      ),
    );
  }, [visible, anchor]);

  /**
   * 选中一项就交给内核去插。
   *
   * 包一层 `useCallback` 不是为了省一次渲染，而是因为它要进下面那个键盘 effect
   * 的依赖表：不包的话每次渲染都是新函数，effect 就会每帧重装一次 `window`
   * 监听。依赖只有 `formatting`，而它一个内核实例只有一个，所以是稳定的。
   */
  const apply = useCallback(
    (block: InsertableBlock): void => {
      // 删 `/query` 与套块格式都在内核里一次做完 —— 文档坐标只有它知道
      formatting.insertBlock(block);
    },
    [formatting],
  );

  useEffect(() => {
    if (!visible) return;

    const onKeyDown = (event: KeyboardEvent) => {
      // 输入法组合期间不要抢键：中文输入时 Enter 是「上屏」，
      // 我们截走它会让用户永远打不出中文
      if (event.isComposing) return;

      if (event.key === 'Escape') {
        event.preventDefault();
        event.stopPropagation();
        setDismissed(query);
        return;
      }

      if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
        event.preventDefault();
        event.stopPropagation();
        if (rows.length === 0) return;
        const step = event.key === 'ArrowDown' ? 1 : -1;
        // 循环而不是停在两端：菜单项很少，撞到底再按一下没反应会显得卡住了
        setHighlighted((at) => (at + step + rows.length) % rows.length);
        return;
      }

      if (event.key === 'Enter' || event.key === 'Tab') {
        const row = rows[highlighted];
        // 一个匹配项都没有时**不拦**：用户多半只是想换行（`/zzz` 就是他想打的字），
        // 把回车吞掉会变成「按了没反应」。放过之后 ProseMirror 拆开段落，
        // 新段落是空的 → 触发条件不再成立 → 菜单自己收起来
        if (row === undefined) return;
        event.preventDefault();
        event.stopPropagation();
        apply(row.block);
      }
    };

    // `true` = 捕获阶段，理由见文件头
    window.addEventListener('keydown', onKeyDown, true);
    return () => window.removeEventListener('keydown', onKeyDown, true);
  }, [visible, query, rows, highlighted, apply]);

  if (!visible) return null;

  return (
    <div
      ref={rootRef}
      className="slash-menu"
      role="listbox"
      aria-label="插入块"
      style={{
        left: placement?.left ?? 0,
        top: placement?.top ?? 0,
        visibility: placement === null ? 'hidden' : 'visible',
      }}
    >
      {rows.length === 0 ? (
        <div className="slash-empty">没有匹配的块</div>
      ) : (
        rows.map((item, index) => (
          <button
            key={item.block}
            type="button"
            role="option"
            aria-selected={index === highlighted}
            className={`slash-item${index === highlighted ? ' highlighted' : ''}`}
            // 不拦这一下，焦点会离开编辑器、ProseMirror 的选区丢失，
            // 菜单会在 click 到达之前被卸载（与选中工具条同一个坑）
            onMouseDown={(event) => event.preventDefault()}
            onMouseEnter={() => setHighlighted(index)}
            onClick={() => apply(item.block)}
          >
            <span className="slash-abbr">{item.abbr}</span>
            <span className="slash-label">{item.label}</span>
            <span className="slash-hint">{item.hint}</span>
          </button>
        ))
      )}
    </div>
  );
}
