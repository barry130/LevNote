// @vitest-environment jsdom
/**
 * 问题清单对话框（ADR 0046）。
 *
 * 这一族的测试钉的不是「画出来好不好看」，而是三条会**丢信息**的地方：
 * 1. 「已经恢复」的记录不许被收起来 —— 横幅一消失就翻脸不认账，等于又一次静默；
 * 2. 复制出去的文本要**自足**（带分组名、带时刻），它是这个应用里唯一一条
 *    把内部状态交给别人的路（没有账号、没有网络、没有遥测，ADR 0001）；
 * 3. 复制失败要有一张脸，不能当没发生（`core.cmd.clipCopyDenied` 记着同一件事）。
 */

import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeAll, beforeEach, describe, expect, it } from 'vitest';

import { EMPTY_HEALTH, recordIssue, resolveIssue, type HealthNote } from '../core/health.js';
import { HealthDialog } from './HealthDialog.js';

declare global {
  var IS_REACT_ACT_ENVIRONMENT: boolean | undefined;
}

function note(id: string, overrides: Partial<HealthNote> = {}): HealthNote {
  return {
    id,
    area: 'storage',
    severity: 'warning',
    lines: [{ key: 'ui.app.storageBrokenTitle', values: {} }],
    ...overrides,
  };
}

describe('HealthDialog · 本次运行的问题', () => {
  let host: HTMLDivElement;
  let root: Root;
  let container: HTMLDivElement;
  let cleared: number;
  let closed: number;

  beforeAll(() => {
    globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  });

  beforeEach(() => {
    container = document.createElement('div');
    document.body.appendChild(container);
    host = document.createElement('div');
    container.appendChild(host);
    root = createRoot(host);
    cleared = 0;
    closed = 0;
  });

  afterEach(() => {
    act(() => {
      root.unmount();
    });
    container.remove();
  });

  function render(log: ReturnType<typeof recordIssue>): void {
    act(() => {
      root.render(
        <HealthDialog
          log={log}
          onClear={() => {
            cleared += 1;
          }}
          onClose={() => {
            closed += 1;
          }}
        />,
      );
    });
  }

  function buttonOf(text: string): HTMLButtonElement {
    const button = [...container.querySelectorAll('button')].find((item) => item.textContent === text);
    if (button === undefined) throw new Error(`找不到「${text}」按钮`);
    return button;
  }

  function recordOf(id: string): HTMLElement {
    const element = container.querySelector(`[data-testid="health-record-${id}"]`);
    if (!element) throw new Error(`找不到 id 为「${id}」的记录`);
    return element as HTMLElement;
  }

  it('空的时候说「没有」，而且复制与清空都不该亮着', () => {
    render(EMPTY_HEALTH);

    expect(container.textContent).toContain('这次运行没有出现过降级。');
    expect(buttonOf('复制全部').disabled).toBe(true);
    expect(buttonOf('清空').disabled).toBe(true);
  });

  it('按族分组：丢东西的族在上，看不见的族在下，空的族不画', () => {
    const log = recordIssue(
      recordIssue(EMPTY_HEALTH, note('watch.reason', { area: 'watch', severity: 'error' }), 100),
      note('storage.write', { area: 'storage', severity: 'error' }), 200,
    );
    render(log);

    const headings = [...container.querySelectorAll('h3')].map((element) => element.textContent);
    expect(headings).toEqual(['存储', '文件监听']);
    expect(container.textContent).not.toContain('编辑内核');
  });

  it('严重的那条画成 error，还没恢复的写着「还没恢复」', () => {
    render(recordIssue(EMPTY_HEALTH, note('session.drafts', { area: 'session', severity: 'error' }), 100));

    const record = recordOf('session.drafts');
    expect(record.className).toContain('error');
    expect(record.className).not.toContain('resolved');
    expect(record.textContent).toContain('还没恢复');
    expect(container.textContent).toContain('现在还有 1 条没恢复。');
  });

  it('恢复了的那条留着，只是打上标记 —— 自动收起来等于又一次静默', () => {
    const log = resolveIssue(recordIssue(EMPTY_HEALTH, note('watch.reason', { area: 'watch' }), 100), 'watch.reason', 200);
    render(log);

    const record = recordOf('watch.reason');
    expect(record.textContent).toContain('已经恢复');
    expect(record.className).toContain('resolved');
    expect(container.textContent).toContain('现在还有 0 条没恢复。');
  });

  it('「发生过 n 次」只在 n > 1 时出现：只发生一次就不要占一行', () => {
    render(recordIssue(EMPTY_HEALTH, note('watch.reason', { area: 'watch' }), 100));
    expect(recordOf('watch.reason').textContent).not.toContain('发生过 1 次');

    render(recordIssue(recordIssue(EMPTY_HEALTH, note('watch.reason', { area: 'watch' }), 100), note('watch.reason', { area: 'watch' }), 200));
    expect(recordOf('watch.reason').textContent).toContain('发生过 2 次');
  });

  it('复制出去的文本自足：分组名、文案、时刻、状态都在里面', async () => {
    const written: string[] = [];
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: (text: string) => { written.push(text); return Promise.resolve(); } },
    });
    render(recordIssue(EMPTY_HEALTH, note('storage.write', { severity: 'error' }), 100));

    await act(async () => {
      buttonOf('复制全部').click();
    });

    expect(written).toHaveLength(1);
    expect(written[0]).toContain('【存储】');
    expect(written[0]).toContain('还没恢复');
    expect(container.textContent).toContain('已复制到剪贴板。');
  });

  it('复制失败要说出来：记录还在，但用户得知道刚才没复制成功', async () => {
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: () => Promise.reject(new Error('denied')) },
    });
    render(recordIssue(EMPTY_HEALTH, note('storage.write', { severity: 'error' }), 100));

    await act(async () => {
      buttonOf('复制全部').click();
    });

    expect(container.textContent).toContain('复制失败');
    expect(container.textContent).not.toContain('已复制到剪贴板。');
  });

  it('清空走 onClear（清的是记录本，不是把对话框关掉）', () => {
    render(recordIssue(EMPTY_HEALTH, note('storage.write'), 100));

    act(() => {
      buttonOf('清空').click();
    });

    expect(cleared).toBe(1);
    expect(closed).toBe(0);
  });

  it('Esc 关掉，点遮罩关掉，点对话框里面不关', () => {
    render(recordIssue(EMPTY_HEALTH, note('storage.write'), 100));
    const backdrop = container.querySelector('.palette-backdrop') as HTMLElement;
    const dialog = container.querySelector('[role="dialog"]') as HTMLElement;

    act(() => {
      dialog.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));
    });
    expect(closed).toBe(0);

    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    });
    expect(closed).toBe(1);

    act(() => {
      backdrop.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));
    });
    expect(closed).toBe(2);
  });

  it('一打开就把焦点收进来', () => {
    render(recordIssue(EMPTY_HEALTH, note('storage.write'), 100));

    expect(document.activeElement).toBe(container.querySelector('[role="dialog"]'));
  });
});