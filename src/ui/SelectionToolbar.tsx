/**
 * 选中文字后浮出来的格式工具条。
 *
 * ## 它为什么住在**外壳**而不是内核里
 *
 * Milkdown 自带 `@milkdown/plugin-tooltip`，但它给出的是一段**由内核用原生 DOM
 * 拼出来的**元素。用它就意味着两件坏事：
 *
 * 1. 工具条的样式不再由 `ui/styles.css` 一处说了算 —— 主题色板、深色模式的变量
 *    全在那边，内核里另起一套必然对不齐；
 * 2. 内核开始持有「用户界面」，而本项目的红线是内核不认识外壳、外壳不认识内核。
 *
 * 所以这里走的是与「查找替换」同一条路：内核只上报**选区的格式**和**它在屏幕上的位置**
 * （`SelectionSnapshot`），浮层由外壳用 React 画。将来换掉 Milkdown，这个组件一行都不用改。
 *
 * ## 两个容易踩的坑
 *
 * - **按钮上要 `preventDefault` 掉 `mousedown`**。不拦的话浏览器会先把焦点从编辑器
 *   挪到按钮上，ProseMirror 的选区随之丢掉 —— 浮层会在 `click` 到达之前消失，
 *   表现成「按钮点不动」。链接输入框相反：它必须能拿到焦点。
 * - **上方放不下要翻到下方**。选区在第一行时，浮层会顶出视口外，用户根本点不到。
 */

import { useEffect, useLayoutEffect, useRef, useState } from 'react';

import type {
  BlockFormat,
  EditorFormatting,
  InlineFormat,
  SelectionSnapshot,
} from '../core/editor/types.js';
import { t, type MessageKey } from '../core/i18n/index.js';
import { computeFloatingPlacement } from './floating.js';

interface InlineButton {
  readonly format: Exclude<InlineFormat, 'link'>;
  /** label 多为字形（B / I / <>），不分语言；title 存 key。 */
  readonly label: string;
  readonly titleKey: MessageKey;
}

const INLINE_BUTTONS: readonly InlineButton[] = [
  { format: 'strong', label: 'B', titleKey: 'ui.selection.bold' },
  { format: 'emphasis', label: 'I', titleKey: 'ui.selection.italic' },
  { format: 'strikethrough', label: 'S', titleKey: 'ui.selection.strikethrough' },
  { format: 'inlineCode', label: '<>', titleKey: 'ui.selection.inlineCode' },
];

interface BlockButton {
  readonly format: BlockFormat;
  readonly level?: number;
  readonly labelKey: MessageKey;
  readonly titleKey: MessageKey;
}

/**
 * 块级格式。
 *
 * 只放到三级标题 —— 与 Typora 的快捷菜单一致。H4–H6 走命令面板，
 * 不是因为做不到，而是六个标题按钮会把浮层撑得比选区还宽。
 */
const BLOCK_BUTTONS: readonly BlockButton[] = [
  { format: 'paragraph', labelKey: 'ui.selection.paragraph', titleKey: 'ui.selection.paragraphTitle' },
  { format: 'heading', level: 1, labelKey: 'ui.selection.h1', titleKey: 'ui.selection.heading1' },
  { format: 'heading', level: 2, labelKey: 'ui.selection.h2', titleKey: 'ui.selection.heading2' },
  { format: 'heading', level: 3, labelKey: 'ui.selection.h3', titleKey: 'ui.selection.heading3' },
  { format: 'blockquote', labelKey: 'ui.selection.blockquote', titleKey: 'ui.selection.quote' },
  { format: 'bulletList', labelKey: 'ui.selection.bulletList', titleKey: 'ui.selection.bulletListTitle' },
  { format: 'orderedList', labelKey: 'ui.selection.orderedList', titleKey: 'ui.selection.orderedListTitle' },
  { format: 'codeBlock', labelKey: 'ui.selection.codeBlock', titleKey: 'ui.selection.codeBlockTitle' },
];

export interface SelectionToolbarProps {
  snapshot: SelectionSnapshot;
  formatting: EditorFormatting;
}

export function SelectionToolbar({ snapshot, formatting }: SelectionToolbarProps) {
  const { anchor, format } = snapshot;
  const rootRef = useRef<HTMLDivElement | null>(null);

  /**
   * 算好的摆放位置。
   *
   * 初值 `null` = 还没量过，先渲染成 `visibility: hidden`。
   * 用 `visibility` 而不是 `display: none`：后者量不出尺寸，永远算不出位置。
   */
  const [placement, setPlacement] = useState<{ left: number; top: number } | null>(null);

  /** 链接地址输入。null = 不在编辑链接。 */
  const [linkDraft, setLinkDraft] = useState<string | null>(null);

  useLayoutEffect(() => {
    const node = rootRef.current;
    if (node === null) return;

    const { width, height } = node.getBoundingClientRect();

    // 优先摆在选区**上方**：下方是用户正在读的内容。摆放算术见 `floating.ts`
    setPlacement(
      computeFloatingPlacement(
        anchor,
        { width, height },
        { vertical: 'above', horizontal: 'center' },
        { width: window.innerWidth, height: window.innerHeight },
      ),
    );
  }, [anchor]);

  /**
   * 选区换了就关掉链接输入框。
   *
   * 不关的话，用户选中一段文字、点开链接输入、又去选另一段文字 ——
   * 输入框还开着，回车会把地址加到**新的**选区上，而用户以为是在编辑旧的。
   *
   * 用 effect 而不是「渲染中 setState」：后者虽然 React 允许，
   * 但要让控制流正确就得额外加一层判断，而这里晚一帧关掉输入框没有任何可见代价。
   */
  const linkKey = `${anchor.top}:${anchor.left}:${format.linkHref ?? ''}`;
  useEffect(() => {
    setLinkDraft(null);
  }, [linkKey]);

  const applyLink = () => {
    if (linkDraft === null) return;
    formatting.toggleInline('link', linkDraft);
    setLinkDraft(null);
  };

  return (
    <div
      ref={rootRef}
      className="selection-toolbar"
      role="toolbar"
      aria-label={t('ui.selection.groupLabel')}
      style={{
        left: placement?.left ?? 0,
        top: placement?.top ?? 0,
        visibility: placement === null ? 'hidden' : 'visible',
      }}
    >
      <div className="st-row">
        {INLINE_BUTTONS.map((button) => (
          <button
            key={button.format}
            type="button"
            className={`st-btn st-${button.format} ${format.inline.has(button.format) ? 'active' : ''}`}
            title={t(button.titleKey)}
            aria-label={t(button.titleKey)}
            aria-pressed={format.inline.has(button.format)}
            // 不拦这一下，焦点会离开编辑器，选区丢失，click 永远不会到
            onMouseDown={(event) => event.preventDefault()}
            onClick={() => formatting.toggleInline(button.format)}
          >
            {button.label}
          </button>
        ))}

        {linkDraft === null ? (
          <button
            type="button"
            className={`st-btn st-link ${format.linkHref !== null ? 'active' : ''}`}
            title={format.linkHref === null ? t('ui.selection.insertLink') : t('ui.selection.editLink', { url: format.linkHref })}
            aria-label={t('ui.selection.link')}
            aria-pressed={format.linkHref !== null}
            onMouseDown={(event) => event.preventDefault()}
            // 已经在链接里就直接摘掉；不在才需要用户给地址。
            // 「点一下摘掉」比「弹个框让你确认要删」更符合工具条的直觉
            onClick={() => {
              if (format.linkHref !== null) formatting.toggleInline('link');
              else setLinkDraft('https://');
            }}
          >
            {t('ui.selection.link')}
          </button>
        ) : (
          <input
            className="st-link-input"
            autoFocus
            value={linkDraft}
            placeholder="https://"
            aria-label={t('ui.selection.linkUrl')}
            onChange={(event) => setLinkDraft(event.target.value)}
            onKeyDown={(event) => {
              // 不拦的话这些按键会冒到 window 上被全局快捷键接走（Esc 会顺手关掉查找栏）
              event.stopPropagation();
              if (event.key === 'Enter') applyLink();
              if (event.key === 'Escape') setLinkDraft(null);
            }}
            onBlur={applyLink}
          />
        )}
      </div>

      <div className="st-row">
        {BLOCK_BUTTONS.map((button) => {
          const active =
            format.block === button.format &&
            (button.format !== 'heading' || format.headingLevel === button.level);
          return (
            <button
              key={`${button.format}-${button.level ?? ''}`}
              type="button"
              className={`st-btn st-block ${active ? 'active' : ''}`}
              title={t(button.titleKey)}
              aria-label={t(button.titleKey)}
              aria-pressed={active}
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => formatting.toggleBlock(button.format, button.level)}
            >
              {t(button.labelKey)}
            </button>
          );
        })}
      </div>
    </div>
  );
}
