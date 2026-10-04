// @vitest-environment jsdom
/**
 * 设置页快捷键录制（P6 第 31 项）的测试。
 *
 * 录制链条的三个关键点各钉一条：
 * 1. 按键 → 规范写法 → 写进偏好（ happy path）；
 * 2. 冲突：录到的组合被别的命令占着 —— 提示要说出来，保存时**显式顶掉**；
 * 3. 录制中按 Esc 是「取消录制」，不是「关闭设置」—— 两个监听同为
 *    window 捕获，这条钉住的是它们之间的让路约定（见 SettingsDialog 的注释）。
 */

import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import { CommandRegistry } from '../core/commands/registry.js';
import type { Command } from '../core/commands/types.js';
import type { ViewSettings } from '../core/settings.js';
import { DEFAULT_SETTINGS } from '../core/settings.js';
import { SettingsDialog } from './SettingsDialog.js';

function command(id: string, title: string, keybinding?: string): Command {
  return { id, title, category: '测试', keybinding, run: () => undefined };
}

function makeRegistry(): CommandRegistry {
  const registry = new CommandRegistry();
  registry.register(command('a', '命令甲', 'Mod+A'));
  registry.register(command('b', '命令乙', 'Mod+B'));
  return registry;
}

const BASE_SETTINGS: ViewSettings = DEFAULT_SETTINGS;

describe('SettingsDialog 快捷键录制', () => {
  let host: HTMLDivElement;
  let root: Root;
  let container: HTMLDivElement;
  let onChangeCalls: Array<Partial<ViewSettings>>;
  let closed: number;

  beforeEach(() => {
    container = document.createElement('div');
    document.body.appendChild(container);
    host = document.createElement('div');
    container.appendChild(host);
    root = createRoot(host);
    onChangeCalls = [];
    closed = 0;
  });

  afterEach(() => {
    act(() => {
      root.unmount();
    });
    container.remove();
  });

  function render(settings?: Partial<ViewSettings>): CommandRegistry {
    const registry = makeRegistry();
    // App 里由一个 effect 把 settings.keybindings 推给注册表（displayBinding
    // 显示的是注册表此刻生效的写法）；直接渲染对话框就要自己做这一步
    const merged = { ...BASE_SETTINGS, ...settings };
    registry.setKeybindingOverrides(merged.keybindings);
    act(() => {
      root.render(
        <SettingsDialog
          settings={merged}
          onChange={(patch) => onChangeCalls.push(patch)}
          registry={registry}
          platform="other"
          onClearPositions={() => undefined}
          onClose={() => {
            closed += 1;
          }}
        />,
      );
    });
    return registry;
  }

  function rowOf(title: string): HTMLDivElement {
    const row = [...container.querySelectorAll('.settings-keys-row')].find(
      (element) => element.textContent?.includes(title),
    );
    if (!row) throw new Error(`找不到「${title}」所在的行`);
    return row as HTMLDivElement;
  }

  function buttonIn(row: HTMLDivElement, text: string): HTMLButtonElement {
    const button = [...row.querySelectorAll('button')].find((item) => item.textContent === text);
    if (!button) throw new Error(`行里找不到「${text}」按钮`);
    return button;
  }

  function pressKey(init: KeyboardEventInit): void {
    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { cancelable: true, bubbles: true, ...init }));
    });
  }

  it('录制并保存：写进偏好的是规范写法', () => {
    render();
    const row = rowOf('命令甲');
    act(() => {
      buttonIn(row, '改').click();
    });

    // 只按修饰键不算录完
    pressKey({ key: 'Control' });
    expect(row.textContent).toContain('按下新的组合键');

    pressKey({ key: 'j', ctrlKey: true });
    expect(row.textContent).toContain('Mod+J');

    act(() => {
      buttonIn(row, '保存').click();
    });

    expect(onChangeCalls).toEqual([{ keybindings: { a: 'Mod+J' } }]);
  });

  it('冲突被说出来，保存时把原主人显式顶掉（置空，不是回到默认）', () => {
    render();
    const row = rowOf('命令甲');
    act(() => {
      buttonIn(row, '改').click();
    });

    // 命令乙此刻占着 Mod+B
    pressKey({ key: 'b', ctrlKey: true });
    expect(container.textContent).toContain('命令乙');
    expect(container.textContent).toContain('未设置');

    act(() => {
      buttonIn(row, '保存').click();
    });

    expect(onChangeCalls).toEqual([{ keybindings: { a: 'Mod+B', b: '' } }]);
  });

  it('录制中按 Esc 是取消录制，设置页不关', () => {
    render();
    const row = rowOf('命令甲');
    act(() => {
      buttonIn(row, '改').click();
    });

    pressKey({ key: 'Escape' });

    expect(closed).toBe(0);
    // 退回展示态：显示的还是默认快捷键，没有写入任何偏好
    expect(row.textContent).toContain('Ctrl+A');
    expect(onChangeCalls).toEqual([]);
  });

  it('改过的行出现「恢复默认」，点击后覆盖被摘掉', () => {
    render({ keybindings: { a: 'Mod+J' } });

    const row = rowOf('命令甲');
    expect(row.textContent).toContain('Ctrl+J');
    act(() => {
      buttonIn(row, '恢复默认').click();
    });

    expect(onChangeCalls).toEqual([{ keybindings: {} }]);
  });
});
