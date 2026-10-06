// @vitest-environment jsdom
/**
 * 菜单栏的**外壳**行为（ADR 0037）。
 *
 * 菜单不带任何自己的语义 —— 显示什么、能不能点都来自 CommandRegistry，
 * 所以这里用假注册表只回答两件事（标题、快捷键），钉住的是 MenuBar 自己
 * 该负责的那层：下拉的展开收起、命令的上报、禁用的呈现、键盘与外点关闭。
 */

import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeAll, describe, expect, it } from 'vitest';

import type { Platform } from '../core/commands/keybinding.js';
import type { CommandRegistry } from '../core/commands/registry.js';
import { MENUS, MenuBar, type MenuDef } from './MenuBar.js';

declare global {
  var IS_REACT_ACT_ENVIRONMENT: boolean | undefined;
}

const TITLES: Readonly<Record<string, string>> = {
  'file.open': '打开文件',
  'file.save': '保存',
  'view.rich': '富文本视图',
  'format.bold': '加粗',
};

/** 只实现 MenuBar 用到的两个方法 —— 注册表的其余机制在别的测试里钉。 */
class FakeRegistry {
  find(id: string): { id: string; title: string } | undefined {
    const title = TITLES[id];
    return title === undefined ? undefined : { id, title };
  }

  displayBinding(
    command: { id: string },
    _platform: Platform,
    _format: (binding: never, platform: Platform) => string,
  ): string | null {
    return command.id === 'file.save' ? 'Ctrl+S' : null;
  }
}

const registry = new FakeRegistry() as unknown as CommandRegistry;
const PLATFORM: Platform = 'other';

/** 小菜单：覆盖分隔线、禁用、快捷键三种形态。 */
const SMALL_MENUS: readonly MenuDef[] = [
  { id: 'file', accel: 'F', items: ['file.open', 'separator', 'file.save'] },
  { id: 'view', accel: 'V', items: ['view.rich'] },
];

const roots: Root[] = [];
const containers: HTMLElement[] = [];

beforeAll(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
});

afterEach(() => {
  for (const root of roots.splice(0)) root.unmount();
  for (const container of containers.splice(0)) container.remove();
  document.body.innerHTML = '';
});

interface Mounted {
  container: HTMLElement;
  /** 点第几个顶层菜单（按下标）。 */
  openMenuAt: (index: number) => Promise<void>;
  dropdown: () => HTMLElement | null;
  menuButtons: () => HTMLElement[];
}

async function mount(
  options: {
    enabled?: (id: string) => boolean;
    onRun?: (id: string) => void;
    isChecked?: (id: string) => boolean | undefined;
  } = {},
): Promise<Mounted> {
  const container = document.createElement('div');
  document.body.appendChild(container);
  const root = createRoot(container);
  roots.push(root);
  containers.push(container);

  const draw = (): void => {
    root.render(
      <MenuBar
        menus={SMALL_MENUS}
        registry={registry}
        platform={PLATFORM}
        isEnabled={options.enabled ?? (() => true)}
        onRun={options.onRun ?? (() => undefined)}
        isChecked={options.isChecked}
      />,
    );
  };

  await act(async () => {
    draw();
  });

  return {
    container,
    openMenuAt: async (index) => {
      const button = container.querySelectorAll<HTMLButtonElement>('.menubar-button')[index];
      if (button === undefined) throw new Error(`没有第 ${index} 个菜单`);
      await act(async () => {
        button.click();
      });
    },
    dropdown: () => container.querySelector<HTMLElement>("[role='menu']"),
    menuButtons: () => [...container.querySelectorAll<HTMLElement>('.menubar-button')],
  };
}

describe('菜单栏（ADR 0037）', () => {
  it('顶层菜单带助记字母画出来 —— 平时没有任何下拉', async () => {
    const mounted = await mount();
    const buttons = mounted.menuButtons();
    expect(buttons).toHaveLength(2);
    expect(buttons[0]?.textContent).toBe('文件(F)');
    expect(buttons[1]?.textContent).toBe('视图(V)');
    expect(mounted.dropdown()).toBeNull();
  });

  it('点开菜单：标题与快捷键来自注册表，禁用项如实灰着，分隔线是分隔线', async () => {
    const mounted = await mount({ enabled: (id) => id !== 'file.open' });
    await mounted.openMenuAt(0);

    const dropdown = mounted.dropdown();
    expect(dropdown).not.toBeNull();
    const items = [...dropdown!.querySelectorAll<HTMLButtonElement>('button[role="menuitem"]')];
    expect(items.map((item) => item.textContent)).toEqual(['打开文件', '保存Ctrl+S']);
    // enabled 是命令的契约，菜单只是转述 —— 禁用是渲染结果
    expect(items[0]?.disabled).toBe(true);
    expect(items[1]?.disabled).toBe(false);
    expect(dropdown!.querySelector("[role='separator']")).not.toBeNull();
  });

  it('点命令项：上报命令 id 并收起', async () => {
    const ran: string[] = [];
    const mounted = await mount({ onRun: (id) => ran.push(id) });
    await mounted.openMenuAt(0);

    const save = [...mounted.dropdown()!.querySelectorAll<HTMLButtonElement>('button[role="menuitem"]')].find(
      (item) => item.textContent === '保存Ctrl+S',
    );
    await act(async () => {
      save!.click();
    });

    expect(ran).toEqual(['file.save']);
    expect(mounted.dropdown()).toBeNull();
  });

  it('勾选状态：可勾选项画 ✓ 或留空槽，不可勾选项不画勾选列', async () => {
    const mounted = await mount({
      isChecked: (id) => (id === 'view.rich' ? true : id === 'file.save' ? false : undefined),
    });
    await mounted.openMenuAt(0);

    const fileItems = [...mounted.dropdown()!.querySelectorAll<HTMLButtonElement>('button[role="menuitem"]')];
    // file.open 不可勾选：连勾选列都没有；file.save 可勾选但未选中：留空槽
    expect(fileItems[0]?.querySelector('.menubar-item-check')).toBeNull();
    const saveCheck = fileItems[1]?.querySelector<HTMLElement>('.menubar-item-check');
    expect(saveCheck).not.toBeNull();
    expect(saveCheck!.textContent).toBe('');
    expect(fileItems[1]?.getAttribute('aria-checked')).toBe('false');

    await mounted.openMenuAt(1);
    const rich = mounted.dropdown()!.querySelector<HTMLButtonElement>('button[role="menuitem"]');
    expect(rich?.querySelector<HTMLElement>('.menubar-item-check')?.textContent).toBe('✓');
    expect(rich?.getAttribute('aria-checked')).toBe('true');
  });

  it('Esc 收起菜单，并且不让事件继续传播（全局 ui.escape 不许被连带触发）', async () => {
    const mounted = await mount();
    await mounted.openMenuAt(0);
    expect(mounted.dropdown()).not.toBeNull();

    // 冒泡阶段的旁观者收不到被拦截的事件 —— 若捕获监听没拦下，
    // 这次 keydown 就会落到全局快捷键手里把查找栏之类顺手关掉
    const seen: string[] = [];
    const spy = (event: KeyboardEvent): void => {
      seen.push(event.key);
    };
    window.addEventListener('keydown', spy);
    try {
      await act(async () => {
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: false }));
      });
    } finally {
      window.removeEventListener('keydown', spy);
    }

    expect(seen).toEqual([]);
    expect(mounted.dropdown()).toBeNull();
  });

  it('点外面收起', async () => {
    const mounted = await mount();
    await mounted.openMenuAt(0);
    expect(mounted.dropdown()).not.toBeNull();

    await act(async () => {
      document.body.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }));
    });
    expect(mounted.dropdown()).toBeNull();
  });

  it('ArrowDown 把焦点挪到第一个**启用的**命令项上', async () => {
    const mounted = await mount({ enabled: (id) => id !== 'file.open' });
    await mounted.openMenuAt(0);

    await act(async () => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }));
    });
    expect(document.activeElement?.textContent).toBe('保存Ctrl+S');
  });

  it('MENUS（真实结构）覆盖七个菜单，项全部是命令 id —— 菜单里没有第二个真相来源', () => {
    expect(MENUS.map((menu) => menu.id)).toEqual([
      'file',
      'edit',
      'paragraph',
      'format',
      'view',
      'theme',
      'help',
    ]);
    for (const menu of MENUS) {
      for (const item of menu.items) {
        if (item !== 'separator') expect(item).toMatch(/^[a-z][a-zA-Z0-9.]+$/);
      }
    }
    expect(MENUS[0]?.items).toContain('file.save');
    expect(MENUS[5]?.items).toContain('theme.ink');
  });
});
