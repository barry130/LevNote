// @vitest-environment jsdom
/**
 * 文件树**外壳半边**的文件管理交互（A2）。
 *
 * 这里钉的是输入行的三条纪律与入口的可见性 —— 业务编排（名字校验、
 * 磁盘操作、标签页跟随）在 `core/workspace/store.test.ts`：
 *
 * 1. 管理能力不可用时按钮**整体不画**（画出来点不了是说谎）；
 * 2. Enter 提交、Esc 取消、空值提交按取消算；
 * 3. 提交失败时输入框留在原地 —— 清掉等于让用户重打一遍刚打错的名字；
 * 4. 行上的按钮点击不会顺带触发行自己的打开 / 折叠。
 */

import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import type { TreeState } from '../core/workspace/store.js';
import { Sidebar } from './Sidebar.js';

declare global {
  var IS_REACT_ACT_ENVIRONMENT: boolean | undefined;
}

const TREE: TreeState = {
  root: 'C:\\w',
  dirs: {
    'C:\\w': [
      { name: 'src', path: 'C:\\w\\src', isDir: true, size: 0, mtimeMs: 0 },
      { name: 'a.md', path: 'C:\\w\\a.md', isDir: false, size: 0, mtimeMs: 0 },
    ],
    'C:\\w\\src': [{ name: 'b.ts', path: 'C:\\w\\src\\b.ts', isDir: false, size: 0, mtimeMs: 0 }],
  },
  expanded: ['C:\\w', 'C:\\w\\src'],
  loading: [],
  showIgnored: false,
  error: null,
};

interface Props {
  tree: TreeState;
  canManageFiles: boolean;
  createResult: boolean;
  renameResult: boolean;
}

interface Calls {
  created: Array<{ dir: string; name: string; kind: 'file' | 'dir' }>;
  renamed: Array<{ path: string; name: string }>;
  deleted: Array<{ path: string; isDir: boolean }>;
  opened: string[];
  toggled: string[];
}

interface Mounted {
  container: HTMLElement;
  calls: Calls;
  setProps: (next: Partial<Props>) => Promise<void>;
  unmount: () => Promise<void>;
}

let roots: Root[] = [];
let containers: HTMLElement[] = [];

beforeEach(() => {
  roots = [];
  containers = [];
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
});

afterEach(async () => {
  for (const root of roots) {
    await act(async () => {
      root.unmount();
    });
  }
});

async function mountSidebar(initial: Partial<Props> = {}): Promise<Mounted> {
  const container = document.createElement('div');
  document.body.appendChild(container);
  const root = createRoot(container);
  roots.push(root);
  containers.push(container);

  const calls: Calls = { created: [], renamed: [], deleted: [], opened: [], toggled: [] };
  const props: Required<Props> = {
    tree: TREE,
    canManageFiles: true,
    createResult: true,
    renameResult: true,
    ...initial,
  };

  const draw = async (): Promise<void> => {
    await act(async () => {
      root.render(
        <Sidebar
          tree={props.tree}
          activePath={null}
          canManageFiles={props.canManageFiles}
          onOpenFolder={() => undefined}
          onToggleDir={(path) => calls.toggled.push(path)}
          onOpenFile={(path) => calls.opened.push(path)}
          onRefresh={() => undefined}
          onToggleIgnored={() => undefined}
          onCreateEntry={(dir, name, kind) => {
            calls.created.push({ dir, name, kind });
            return Promise.resolve(props.createResult);
          }}
          onRenameEntry={(path, name) => {
            calls.renamed.push({ path, name });
            return Promise.resolve(props.renameResult);
          }}
          onDeleteEntry={(path, isDir) => {
            calls.deleted.push({ path, isDir });
            return Promise.resolve(true);
          }}
        />,
      );
    });
  };

  await draw();
  return {
    container,
    calls,
    setProps: async (next) => {
      Object.assign(props, next);
      await draw();
    },
    unmount: async () => {
      await act(async () => {
        root.unmount();
      });
    },
  };
}

// ---------------------------------------------------------------- 查询与事件

function headerButtonsOf(container: HTMLElement): HTMLButtonElement[] {
  return Array.from(container.querySelectorAll('.sidebar-head button')).filter(
    (el): el is HTMLButtonElement => el instanceof HTMLButtonElement,
  );
}

function actionButtonsOf(row: HTMLElement): HTMLButtonElement[] {
  return Array.from(row.querySelectorAll('.tree-actions button')).filter(
    (el): el is HTMLButtonElement => el instanceof HTMLButtonElement,
  );
}

function rowByName(container: HTMLElement, name: string): HTMLElement | null {
  const rows = Array.from(container.querySelectorAll('.tree-row'));
  const found = rows.find((el) => el.querySelector('.tree-name')?.textContent === name);
  return found instanceof HTMLElement ? found : null;
}

function inputOf(container: HTMLElement): HTMLInputElement | null {
  const el = container.querySelector('.tree-input-row input');
  return el instanceof HTMLInputElement ? el : null;
}

function press(input: HTMLInputElement, key: string): void {
  input.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }));
}

/** 往受控输入框里「打字」：React 认 `input` 事件，且要用原生 setter 改值。 */
async function type(input: HTMLInputElement, value: string): Promise<void> {
  await act(async () => {
    const setNativeValue = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set?.bind(input);
    setNativeValue?.(value);
    input.dispatchEvent(new Event('input', { bubbles: true }));
  });
}

/** 让提交 promise 落地并回到 act 环境。 */
async function flush(): Promise<void> {
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
}

// ---------------------------------------------------------------------- 用例

describe('Sidebar 文件管理（A2）', () => {
  it('管理能力不可用时：文件管理按钮整体不画，其余入口照常', async () => {
    const mounted = await mountSidebar({ canManageFiles: false });

    const header = headerButtonsOf(mounted.container).map((el) => el.title);
    expect(header).not.toContain('在根目录新建文件');
    expect(header).toContain('刷新目录');

    const row = rowByName(mounted.container, 'a.md');
    expect(row).not.toBeNull();
    expect(row?.querySelector('.tree-actions')).toBeNull();
  });

  it('头部「新建文件」→ 输入行出现在树顶，Enter 提交后输入行消失', async () => {
    const mounted = await mountSidebar();

    const create = headerButtonsOf(mounted.container).find((el) => el.title === '在根目录新建文件');
    expect(create).toBeDefined();
    await act(async () => {
      create!.click();
    });

    const input = inputOf(mounted.container);
    expect(input).not.toBeNull();

    await type(input!, '笔记.md');
    await act(async () => {
      press(input!, 'Enter');
    });
    await flush();

    expect(mounted.calls.created).toEqual([{ dir: 'C:\\w', name: '笔记.md', kind: 'file' }]);
    expect(inputOf(mounted.container)).toBeNull();
  });

  it('提交失败：输入框留在原地，改名再试成功后收起', async () => {
    const mounted = await mountSidebar({ createResult: false });

    await act(async () => {
      headerButtonsOf(mounted.container).find((el) => el.title === '在根目录新建文件')!.click();
    });
    let input = inputOf(mounted.container);
    expect(input).not.toBeNull();

    await type(input!, 'a/b.md');
    await act(async () => {
      press(input!, 'Enter');
    });
    await flush();

    // 失败：onCreateEntry 已经收到名字，但输入行还在
    expect(mounted.calls.created).toEqual([{ dir: 'C:\\w', name: 'a/b.md', kind: 'file' }]);
    input = inputOf(mounted.container);
    expect(input).not.toBeNull();

    await mounted.setProps({ createResult: true });
    await type(input!, 'ok.md');
    await act(async () => {
      press(input!, 'Enter');
    });
    await flush();

    expect(mounted.calls.created).toHaveLength(2);
    expect(inputOf(mounted.container)).toBeNull();
  });

  it('目录行上的「新建文件夹」：提交目标是那个目录，行点击不被触发', async () => {
    const mounted = await mountSidebar();

    const row = rowByName(mounted.container, 'src');
    expect(row).not.toBeNull();
    const buttons = actionButtonsOf(row!);
    expect(buttons.map((el) => el.title)).toEqual(['在这新建文件', '在这新建文件夹', '重命名', '删除']);

    await act(async () => {
      buttons[1]!.click();
    });
    const input = inputOf(mounted.container);
    expect(input).not.toBeNull();

    await type(input!, 'docs');
    await act(async () => {
      press(input!, 'Enter');
    });
    await flush();

    expect(mounted.calls.created).toEqual([{ dir: 'C:\\w\\src', name: 'docs', kind: 'dir' }]);
    // stopPropagation：点按钮不该把目录折叠 / 展开一下
    expect(mounted.calls.toggled).toEqual([]);
  });

  it('重命名：输入行预填旧名，Enter 交出新名字', async () => {
    const mounted = await mountSidebar();

    const row = rowByName(mounted.container, 'a.md');
    await act(async () => {
      actionButtonsOf(row!).find((el) => el.title === '重命名')!.click();
    });

    const input = inputOf(mounted.container);
    expect(input?.value).toBe('a.md');

    await type(input!, 'b.md');
    await act(async () => {
      press(input!, 'Enter');
    });
    await flush();

    expect(mounted.calls.renamed).toEqual([{ path: 'C:\\w\\a.md', name: 'b.md' }]);
    expect(inputOf(mounted.container)).toBeNull();
  });

  it('Esc 取消，空值提交按取消算 —— 都不惊动业务层', async () => {
    const mounted = await mountSidebar();

    await act(async () => {
      headerButtonsOf(mounted.container).find((el) => el.title === '在根目录新建文件')!.click();
    });
    let input = inputOf(mounted.container);
    await act(async () => {
      press(input!, 'Escape');
    });
    expect(inputOf(mounted.container)).toBeNull();
    expect(mounted.calls.created).toEqual([]);

    await act(async () => {
      headerButtonsOf(mounted.container).find((el) => el.title === '在根目录新建文件夹')!.click();
    });
    input = inputOf(mounted.container);
    // 直接回车（空名字）：输入行消失，store 一次都没被调
    await act(async () => {
      press(input!, 'Enter');
    });
    await flush();
    expect(inputOf(mounted.container)).toBeNull();
    expect(mounted.calls.created).toEqual([]);
  });

  it('删除按钮把「删的是什么」原样交出去，也不触发行点击', async () => {
    const mounted = await mountSidebar();

    const fileRow = rowByName(mounted.container, 'a.md');
    await act(async () => {
      actionButtonsOf(fileRow!).find((el) => el.title === '删除')!.click();
    });
    const dirRow = rowByName(mounted.container, 'src');
    await act(async () => {
      actionButtonsOf(dirRow!).find((el) => el.title === '删除')!.click();
    });

    expect(mounted.calls.deleted).toEqual([
      { path: 'C:\\w\\a.md', isDir: false },
      { path: 'C:\\w\\src', isDir: true },
    ]);
    expect(mounted.calls.opened).toEqual([]);
    expect(mounted.calls.toggled).toEqual([]);
  });
});
