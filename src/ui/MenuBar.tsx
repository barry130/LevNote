/**
 * 菜单栏 —— Typora 式的文字菜单（文件(F) 编辑(E) …），下拉项是命令。
 *
 * ## 为什么是自绘的 web 菜单，而不是 Tauri 的原生菜单
 *
 * 每个菜单项的可用性来自 `CommandRegistry.enabled`（保存要判只读、
 * 检索要判有没有文件夹），那是**前端的实时状态** —— 原生菜单要么做成静态
 * （点了禁用的项），要么每次状态变化都跨 Rust 桥整表重建。而命令的标题、
 * 快捷键、i18n 也全在前端这一层。自绘之后这条链路一厘米都不用跨（ADR 0037）。
 *
 * ## 菜单与命令的关系
 *
 * 菜单**只是命令的另一个入口**，不带任何自己的语义：能点什么、什么时候
 * 灰着，全部由命令的 `enabled` 契约回答 —— 与工具栏、活动栏、命令面板
 * 同一条约定（「答案必须来自同一个地方」）。菜单定义（`MENUS`）只是一份
 * 「哪个菜单里放哪些命令 id」的数据。
 *
 * ## 键盘
 *
 * 菜单开着时焦点还在编辑器里（用户可能正在打字），所以键盘监听挂在
 * **window 的捕获阶段**、处理完就拦下 —— 与浮层菜单同一条纪律（坑 9），
 * 不然 Esc 会先落进全局快捷键把查找栏顺手关掉。
 */

import { useEffect, useRef, useState } from 'react';

import { formatKeybinding, type Platform } from '../core/commands/keybinding.js';
import { ENCODING_COMMAND_IDS } from '../core/commands/definitions.js';
import type { CommandRegistry } from '../core/commands/registry.js';
import { t, type MessageKey } from '../core/i18n/index.js';

export interface MenuDef {
  /** 菜单的稳定标识（测试与 key 用），如 'file'。 */
  readonly id: string;
  /** 快捷助记字母，显示成「文件(F)」。 */
  readonly accel: string;
  readonly items: readonly (string | 'separator')[];
}

/**
 * 菜单结构。项就是**命令 id**（字符串）；显示文案与可用性都从
 * CommandRegistry 现查，这份定义里没有第二个真相来源。
 *
 * 分组与顺序参考 Typora（不要求逐项一致）——每一项都得**接上真实能力**：
 * 没有对应命令的项不进菜单，而不是画一个点了没反应的占位。
 */
export const MENUS: readonly MenuDef[] = [
  {
    id: 'file',
    accel: 'F',
    items: [
      'file.new',
      'file.newMindmap',
      'file.open',
      'file.openFolder',
      'separator',
      'file.save',
      'file.saveAs',
      'file.saveAll',
      'file.reload',
      'separator',
      'file.exportHtml',
      'file.exportPdf',
      'file.exportPng',
      'file.exportSvg',
      'separator',
      'app.settings',
      'file.closeTab',
    ],
  },
  {
    id: 'edit',
    accel: 'E',
    items: [
      'edit.undo',
      'edit.redo',
      'separator',
      'find.open',
      'search.inFolder',
      'edit.copyHtml',
      'separator',
      'edit.extractReference',
      'edit.inlineReference',
      'separator',
      'palette.open',
    ],
  },
  {
    id: 'paragraph',
    accel: 'P',
    items: [
      'paragraph.heading1',
      'paragraph.heading2',
      'paragraph.heading3',
      'paragraph.heading4',
      'paragraph.heading5',
      'paragraph.heading6',
      'paragraph.plainText',
      'separator',
      'paragraph.blockquote',
      'paragraph.bulletList',
      'paragraph.orderedList',
      'separator',
      'paragraph.codeBlock',
      'separator',
      'paragraph.table',
      'paragraph.mathBlock',
      'paragraph.diagram',
      'paragraph.drawio',
      'paragraph.hr',
    ],
  },
  {
    id: 'format',
    accel: 'O',
    items: ['format.bold', 'format.italic', 'format.strikethrough', 'format.inlineCode'],
  },
  {
    // Notepad++ 的「搜索」菜单位置：查找 / 跨文件检索 / 命令面板是同一类
    // 「去找点什么」的动作，与编辑动作分开一条菜单更好找
    id: 'search',
    accel: 'S',
    items: ['find.open', 'search.inFolder', 'separator', 'palette.open'],
  },
  {
    id: 'view',
    accel: 'V',
    items: [
      'view.toggleSidebar',
      'view.filesView',
      'view.toggleOutline',
      'separator',
      'view.rich',
      'view.source',
      'view.split',
      // 两态开关（Ctrl+/）：与上面三档并存而不是被取代 —— 它是「在两边之间
      // 来回」的肌肉记忆动作，三档是「直接跳过去」。原顶部工具栏的「源码」
      // 按钮并入这里（工具栏整排已撤）
      'view.toggleSource',
      'separator',
      'view.toggleWordWrap',
      'view.toggleCodeLineNumbers',
      // 写作模式（Typora 的两个）：偏好存设置层，内核按能力消费
      'view.toggleFocusMode',
      'view.toggleTypewriterMode',
      'separator',
      'view.toggleFullscreen',
      'separator',
      'tab.next',
      'tab.prev',
    ],
  },
  {
    // Notepad++ 的「编码」菜单：每一项是「保存时按这个编码写盘」。
    // 选项清单只有 core/encodings.ts 一份，命令 id 从 definitions.ts 导出
    id: 'encoding',
    accel: 'C',
    items: ENCODING_COMMAND_IDS,
  },
  {
    id: 'theme',
    accel: 'T',
    items: ['theme.system', 'theme.paper', 'theme.ink', 'separator', 'view.cycleTheme'],
  },
  {
    id: 'help',
    accel: 'H',
    items: [
      'help.sampleDocument',
      'separator',
      'diagnostics.issues',
      'diagnostics.roundTrip',
      'separator',
      'app.about',
    ],
  },
];

/**
 * 菜单名在**渲染时**经 `t()` 取（不再是模块常量）：语言切换后菜单栏要跟着变，
 * 而语言状态是模块级的，烤进模块常量里的文案永远不会更新。
 */
const MENU_LABEL_KEYS: Readonly<Record<string, MessageKey>> = {
  file: 'menu.file',
  edit: 'menu.edit',
  paragraph: 'menu.paragraph',
  format: 'menu.format',
  search: 'menu.search',
  view: 'menu.view',
  encoding: 'menu.encoding',
  theme: 'menu.theme',
  help: 'menu.help',
};

function menuLabel(id: string): string {
  const key = MENU_LABEL_KEYS[id];
  return key === undefined ? id : t(key);
}

export interface MenuBarProps {
  /** 菜单结构；默认 `MENUS`（测试可以喂一份小的）。 */
  menus?: readonly MenuDef[];
  registry: CommandRegistry;
  platform: Platform;
  isEnabled: (id: string) => boolean;
  onRun: (id: string) => void;
  /**
   * 勾选状态（三态）：`true` / `false` = 可勾选（画 ✓ 或留空槽），
   * `undefined` = 这条项没有「开关」语义，不画勾选列。
   * 状态的真相来源在调用方（设置快照、视图模式），菜单只负责画。
   */
  isChecked?: (id: string) => boolean | undefined;
}

export function MenuBar({ menus = MENUS, registry, platform, isEnabled, onRun, isChecked }: MenuBarProps) {
  /** 当前展开的菜单 id；null = 全部收起。 */
  const [openId, setOpenId] = useState<string | null>(null);
  const rootRef = useRef<HTMLElement | null>(null);
  /** 顶层按钮的引用，左右方向键切换菜单后把焦点跟过去。 */
  const buttonRefs = useRef(new Map<string, HTMLButtonElement>());

  // 只在「有菜单开着」时挂：收起状态下键盘不属于菜单栏（编辑器正在打字），
  // 一个常驻的 keydown 监听等于给每敲一个字加一道无关的关卡
  useEffect(() => {
    if (openId === null) return;

    const onPointerDown = (event: PointerEvent) => {
      if (
        rootRef.current !== null &&
        event.target instanceof Node &&
        !rootRef.current.contains(event.target)
      ) {
        setOpenId(null);
      }
    };

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        // 拦下并吃掉：全局 ui.escape 不该被菜单开着的这次 Esc 连带触发
        event.preventDefault();
        event.stopPropagation();
        setOpenId(null);
        return;
      }
      if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
        const menu = rootRef.current?.querySelector<HTMLElement>(`[data-menu-id="${openId}"] [role='menu']`);
        const items = menu === null || menu === undefined ? [] : ([...menu.querySelectorAll<HTMLButtonElement>('button[role="menuitem"]:not(:disabled)')]);
        if (items.length === 0) return;
        event.preventDefault();
        const current = items.indexOf(document.activeElement as HTMLButtonElement);
        const next =
          current < 0
            ? 0
            : event.key === 'ArrowDown'
              ? (current + 1) % items.length
              : (current - 1 + items.length) % items.length;
        items[next]?.focus();
        return;
      }
      if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') {
        // 左右在顶层菜单之间切换（保持展开状态），焦点跟过去 —— 与桌面应用一致
        const at = menus.findIndex((menu) => menu.id === openId);
        if (at < 0) return;
        const step = event.key === 'ArrowRight' ? 1 : -1;
        const next = menus[(at + step + menus.length) % menus.length];
        if (next === undefined) return;
        event.preventDefault();
        setOpenId(next.id);
        buttonRefs.current.get(next.id)?.focus();
        return;
      }
      if (event.key === 'Tab') {
        // Tab 离开菜单栏：收起，让焦点循环回到正常顺序
        setOpenId(null);
      }
    };

    document.addEventListener('pointerdown', onPointerDown, true);
    window.addEventListener('keydown', onKeyDown, true);
    return () => {
      document.removeEventListener('pointerdown', onPointerDown, true);
      window.removeEventListener('keydown', onKeyDown, true);
    };
  }, [openId, menus]);

  const runAndClose = (command: string): void => {
    setOpenId(null);
    onRun(command);
  };

  return (
    <nav className="menubar" role="menubar" ref={rootRef}>
      {menus.map((menu) => {
        const open = openId === menu.id;
        return (
          <div key={menu.id} className="menubar-menu" data-menu-id={menu.id}>
            <button
              type="button"
              role="menuitem"
              aria-haspopup="menu"
              aria-expanded={open}
              className={open ? 'menubar-button open' : 'menubar-button'}
              ref={(node) => {
                if (node === null) buttonRefs.current.delete(menu.id);
                else buttonRefs.current.set(menu.id, node);
              }}
              onClick={() => setOpenId(open ? null : menu.id)}
              // 有菜单开着时扫过去就切换（桌面应用的习惯）；全收起时悬停不展开
              onMouseEnter={openId !== null ? () => setOpenId(menu.id) : undefined}
            >
              {menuLabel(menu.id)}
              <span className="menubar-accel">({menu.accel})</span>
            </button>

            {open && (
              <div className="menubar-dropdown" role="menu">
                {menu.items.map((item, index) =>
                  item === 'separator' ? (
                    <div key={`sep-${index}`} className="menubar-separator" role="separator" />
                  ) : (
                    <MenuItem
                      key={item}
                      command={item}
                      registry={registry}
                      platform={platform}
                      enabled={isEnabled(item)}
                      checked={isChecked?.(item)}
                      // 勾选列按**整条菜单**保留：菜单里只要有一项可勾选，
                      // 所有项（含不可勾选的）都留出那个槽位 —— 否则同一菜单里
                      // 标签的起点会随「有没有勾选列」左右横跳
                      reserveCheck={menuHasChecks(menu, isChecked)}
                      onRun={runAndClose}
                    />
                  ),
                )}
              </div>
            )}
          </div>
        );
      })}
    </nav>
  );
}

/** 这条菜单里有没有可勾选的项（决定要不要给整条菜单保留勾选列）。 */
function menuHasChecks(menu: MenuDef, isChecked: ((id: string) => boolean | undefined) | undefined): boolean {
  if (isChecked === undefined) return false;
  return menu.items.some((item) => item !== 'separator' && isChecked(item) !== undefined);
}

interface MenuItemProps {
  command: string;
  registry: CommandRegistry;
  platform: Platform;
  enabled: boolean;
  /** `undefined` = 这一项没有开关语义；`true/false` = 可勾选（画 ✓ / 留空槽）。 */
  checked?: boolean;
  /** 本菜单里存在可勾选项时，不可勾选的项也要留出勾选列，让标签起点对齐。 */
  reserveCheck?: boolean;
  onRun: (id: string) => void;
}

function MenuItem({ command, registry, platform, enabled, checked, reserveCheck, onRun }: MenuItemProps) {
  const definition = registry.find(command);
  // 定义缺失不该让整个菜单挂掉 —— 画成禁用项，name 显示 id（开发期就该发现）
  const title = definition?.title ?? command;
  const shortcut = definition === undefined ? null : registry.displayBinding(definition, platform, formatKeybinding);
  const showCheck = checked !== undefined || reserveCheck === true;
  return (
    <button
      type="button"
      role="menuitem"
      className="menubar-item"
      disabled={!enabled}
      title={title}
      // 可勾选的项把状态也交给无障碍树 —— 与视觉上的 ✓ 同一个真相
      aria-checked={checked}
      onClick={() => onRun(command)}
    >
      {showCheck && (
        <span className="menubar-item-check" aria-hidden="true">
          {checked ? '✓' : ''}
        </span>
      )}
      <span className="menubar-item-label">{title}</span>
      {shortcut !== null && <span className="menubar-item-key">{shortcut}</span>}
    </button>
  );
}
