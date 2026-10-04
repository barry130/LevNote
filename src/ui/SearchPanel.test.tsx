// @vitest-environment jsdom
/**
 * 检索面板的**外壳半边**：什么时候能搜、结果怎么画、状态怎么读。
 *
 * ## 这里钉的是什么
 *
 * 时序（防抖 / 取消 / 过期批次）已经在 `core/search/workspace.test.ts` 里
 * 用假执行者钉死了 —— 那三层是**看不见的**，错了只会表现成
 * 「结果里混着上一次的」「面板一直转」。这里不重复测它们，只测**用户能看见的那一层**：
 *
 * 1. **禁用态必须说出原因。** 面板有三种「现在搜不了」：平台不支持、没打开文件夹、
 *    没输入内容。前两种如果只是把输入框灰掉，用户会以为软件坏了 ——
 *    所以每一种都配一句说明，这是硬约束 2（绝不静默）在面板上的落点。
 * 2. **零宽匹配要看得见。** `^` / `\b` / `a*` 命中时没有可高亮的字符。
 *    什么都不画的话，「搜到了」就成了一句用户无法验证的话。
 * 3. **跳过必须显示。** 二进制 / 超大 / 读不到三个数字，与命中数分开显示 ——
 *    它们回答的是另一个问题：「我明明知道它在里面」。
 * 4. **空结果不是一句话，是一段可行动的说明。** 提示那个 `⊙` 逃生口，
 *    且勾上之后那句提示要消失（否则用户会去点一个已经打开的开关）。
 *
 * ## 用真 store + 假执行者
 *
 * 面板**没有自己的状态**，全部在 `WorkspaceSearchStore` 里。所以这里挂真 store、
 * 只把平台那一段换成假的 —— 接线（`useSyncExternalStore` 的订阅、防抖到点后
 * 重新渲染）也就一并测到了。用一个手搓的假 store 反而会把接线漏掉。
 */

import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  WorkspaceSearchStore,
  type WorkspaceBatch,
  type WorkspaceMatch,
  type WorkspaceSearchRequest,
  type WorkspaceSearcher,
  type WorkspaceSummary,
} from '../core/search/workspace.js';
import { SearchPanel } from './SearchPanel.js';

declare global {
  // React 要求测试环境显式声明这一点，否则 act() 会警告并跳过批处理
  var IS_REACT_ACT_ENVIRONMENT: boolean | undefined;
}

// ---------------------------------------------------------------------------
// 假执行者
// ---------------------------------------------------------------------------

interface FakeRun {
  readonly request: WorkspaceSearchRequest;
  readonly onBatch: (batch: WorkspaceBatch) => void;
  readonly resolve: (summary: WorkspaceSummary) => void;
  readonly reject: (error: unknown) => void;
}

interface FakeSearcher {
  readonly searcher: WorkspaceSearcher;
  readonly runs: FakeRun[];
  readonly cancels: number[];
}

function createFakeSearcher(): FakeSearcher {
  const runs: FakeRun[] = [];
  const cancels: number[] = [];
  return {
    runs,
    cancels,
    searcher: {
      available: true,
      run(request, onBatch) {
        return new Promise<WorkspaceSummary>((resolve, reject) => {
          runs.push({ request, onBatch, resolve, reject });
        });
      },
      cancel(searchId) {
        cancels.push(searchId);
        return Promise.resolve();
      },
    },
  };
}

function match(overrides: Partial<WorkspaceMatch> = {}): WorkspaceMatch {
  return { line: 1, column: 0, length: 3, text: 'abc', textOffset: 0, tailCut: false, ...overrides };
}

function batch(files: WorkspaceBatch['files'], overrides: Partial<WorkspaceBatch> = {}): WorkspaceBatch {
  return {
    files,
    matchedFiles: files.length,
    totalMatches: files.reduce((sum, file) => sum + file.matches.length, 0),
    scannedFiles: 4,
    scannedBytes: 1024,
    skippedBinary: 0,
    skippedLarge: 0,
    skippedUnreadable: 0,
    ...overrides,
  };
}

function summaryOf(overrides: Partial<WorkspaceSummary> = {}): WorkspaceSummary {
  return {
    reason: 'completed',
    matchedFiles: 0,
    totalMatches: 0,
    scannedFiles: 4,
    scannedBytes: 1024,
    skippedBinary: 0,
    skippedLarge: 0,
    skippedUnreadable: 0,
    elapsedMs: 20,
    ...overrides,
  };
}

// ---------------------------------------------------------------------------
// 渲染
// ---------------------------------------------------------------------------

interface Props {
  root: string | null;
  canSearch: boolean;
}

interface Mounted {
  readonly container: HTMLElement;
  readonly store: WorkspaceSearchStore;
  readonly runs: FakeRun[];
  readonly cancels: number[];
  /** `onOpenMatch` 的调用记录。 */
  readonly opened: Array<{ path: string; match: WorkspaceMatch }>;
  readonly closeCount: () => number;
  readonly setProps: (next: Partial<Props>) => Promise<void>;
  readonly unmount: () => Promise<void>;
}

let roots: Root[] = [];
let containers: HTMLElement[] = [];

async function mountPanel(initial: Props = { root: '/root', canSearch: true }): Promise<Mounted> {
  const fake = createFakeSearcher();
  const store = new WorkspaceSearchStore(fake.searcher, 250);
  store.setRoot(initial.root);

  const container = document.createElement('div');
  document.body.appendChild(container);
  const root = createRoot(container);
  roots.push(root);
  containers.push(container);

  const opened: Array<{ path: string; match: WorkspaceMatch }> = [];
  let closes = 0;
  let props = initial;

  const draw = async (): Promise<void> => {
    await act(async () => {
      root.render(
        <SearchPanel
          store={store}
          root={props.root}
          canSearch={props.canSearch}
          onOpenMatch={(path, m) => opened.push({ path, match: m })}
          onClose={() => {
            closes += 1;
          }}
        />,
      );
    });
  };

  await draw();

  return {
    container,
    store,
    runs: fake.runs,
    cancels: fake.cancels,
    opened,
    closeCount: () => closes,
    async setProps(next) {
      props = { ...props, ...next };
      // 先取成局部量：窄化在回调里会失效（TS 不能证明 `next.root` 没被改过）
      const nextRoot = next.root;
      // `setRoot` 会通知订阅者 → 触发重渲染，必须包在 act 里
      if (nextRoot !== undefined) {
        await act(async () => {
          store.setRoot(nextRoot);
        });
      }
      await draw();
    },
    async unmount() {
      await act(async () => {
        root.unmount();
      });
      container.remove();
    },
  };
}

/** 往受控输入框里「打字」：React 认 `input` 事件，所以要先改值再派发。 */
async function type(input: HTMLInputElement, value: string): Promise<void> {
  await act(async () => {
    // 先 bind 再用（unbound-method）：与 App.test.tsx 里 native setter 的写法一致
    const setNativeValue = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set?.bind(input);
    setNativeValue?.(value);
    input.dispatchEvent(new Event('input', { bubbles: true }));
  });
}

async function click(element: Element): Promise<void> {
  await act(async () => {
    element.dispatchEvent(new MouseEvent('click', { bubbles: true }));
  });
}

async function pressEnter(element: Element): Promise<void> {
  await act(async () => {
    element.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
  });
}

async function advance(ms: number): Promise<void> {
  await act(async () => {
    vi.advanceTimersByTime(ms);
  });
}

/** 让 store 的 promise 链结算（`invoke` 返回后的 set 都在微任务里）。 */
async function settle(): Promise<void> {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  });
}

/** 「输入 → 防抖到点」的常用两步。 */
async function searchFor(m: Mounted, query: string): Promise<void> {
  await type(inputOf(m.container), query);
  await advance(250);
}

function inputOf(container: HTMLElement): HTMLInputElement {
  const input = container.querySelector<HTMLInputElement>('.search-input');
  if (input === null) throw new Error('没有输入框');
  return input;
}

function toggleOf(container: HTMLElement, label: string): HTMLButtonElement {
  const found = [...container.querySelectorAll<HTMLButtonElement>('.search-toggle')].find((button) =>
    (button.textContent ?? '').includes(label),
  );
  if (found === undefined) throw new Error(`没有开关：${label}`);
  return found;
}

function hitsOf(container: HTMLElement): HTMLButtonElement[] {
  return [...container.querySelectorAll<HTMLButtonElement>('.search-hit')];
}

function runButton(container: HTMLElement): HTMLButtonElement {
  const button = container.querySelector<HTMLButtonElement>('.search-row button');
  if (button === null) throw new Error('没有检索按钮');
  return button;
}

/** 让第 n 次（从 0 数）检索收尾。 */
async function finish(m: Mounted, index: number, summary: Partial<WorkspaceSummary> = {}): Promise<void> {
  await act(async () => {
    m.runs[index]?.resolve(summaryOf(summary));
  });
  await settle();
}

beforeEach(() => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
});

afterEach(async () => {
  for (const root of roots) {
    await act(async () => {
      root.unmount();
    });
  }
  for (const container of containers) container.remove();
  roots = [];
  containers = [];
  vi.useRealTimers();
});

// ---------------------------------------------------------------------------

describe('SearchPanel · 什么时候能搜', () => {
  it('没打开文件夹时输入框禁用，并且说出原因', async () => {
    const m = await mountPanel({ root: null, canSearch: true });

    expect(inputOf(m.container).disabled).toBe(true);
    expect(m.container.querySelector('.search-note')?.textContent).toContain('还没有打开文件夹');
  });

  it('平台不支持时说清楚「只在桌面版可用」，而不是画一个灰输入框', async () => {
    const m = await mountPanel({ root: '/root', canSearch: false });

    expect(inputOf(m.container).disabled).toBe(true);
    expect(m.container.querySelector('.search-note')?.textContent).toContain('只在桌面版可用');
    // 不支持时不该画「已扫 0 个文件」这种假进度
    expect(m.container.querySelector('.search-status')).toBeNull();
  });

  it('输入框为空时按钮禁用（没什么可搜的）', async () => {
    const m = await mountPanel();
    expect(runButton(m.container).disabled).toBe(true);
  });

  it('没检索过时给一句「输入要检索的内容」，不留一片空白', async () => {
    const m = await mountPanel();
    expect(m.container.querySelector('.search-status')?.textContent).toContain('输入要检索的内容');
  });

  it('打开文件夹（root 由 null 变成路径）之后就能搜了', async () => {
    const m = await mountPanel({ root: null, canSearch: true });
    expect(inputOf(m.container).disabled).toBe(true);

    await m.setProps({ root: '/root' });
    expect(inputOf(m.container).disabled).toBe(false);
    expect(m.container.querySelector('.search-note')).toBeNull();
  });
});

describe('SearchPanel · 发起与停止', () => {
  it('打字之后防抖到点才发起', async () => {
    const m = await mountPanel();
    await type(inputOf(m.container), '配置');

    expect(m.runs).toHaveLength(0);
    await advance(250);
    expect(m.runs).toHaveLength(1);
    expect(m.runs[0]?.request.query).toBe('配置');
  });

  it('回车立刻发起，不等防抖', async () => {
    const m = await mountPanel();
    await type(inputOf(m.container), '配置');
    await pressEnter(inputOf(m.container));

    expect(m.runs).toHaveLength(1);
    // 防抖那次不该再补发
    await advance(1000);
    expect(m.runs).toHaveLength(1);
  });

  it('进行中显示「已扫 N 个文件 · 命中 M 处」，按钮变「停止」', async () => {
    const m = await mountPanel();
    await searchFor(m, 'abc');

    await act(async () => {
      m.runs[0]?.onBatch(batch([], { scannedFiles: 42, totalMatches: 7 }));
    });

    const text = m.container.querySelector('.search-progress')?.textContent ?? '';
    expect(text).toContain('已扫 42 个文件');
    expect(text).toContain('命中 7 处');
    expect(runButton(m.container).textContent).toContain('停止');
  });

  it('点「停止」把取消通知发给平台', async () => {
    const m = await mountPanel();
    await searchFor(m, 'abc');
    const id = m.runs[0]?.request.searchId;

    await click(runButton(m.container));

    expect(m.cancels).toEqual([id]);
  });

  it('收尾之后按钮回到「检索」', async () => {
    const m = await mountPanel();
    await searchFor(m, 'abc');
    await finish(m, 0, { totalMatches: 2, matchedFiles: 1 });

    expect(runButton(m.container).textContent).toContain('检索');
  });

  it('错误显示在面板里，不是只写进控制台', async () => {
    const m = await mountPanel();
    await searchFor(m, '(');
    await act(async () => {
      m.runs[0]?.reject({ code: 'INVALID_PATH', message: '正则表达式无效：unclosed group' });
    });
    await settle();

    expect(m.container.querySelector('.sidebar-error')?.textContent).toContain('正则表达式无效');
    // 失败之后不能一直转
    expect(runButton(m.container).textContent).toContain('检索');
  });
});

describe('SearchPanel · 结果怎么画', () => {
  it('文件行与命中行都画出来，命中段高亮', async () => {
    const m = await mountPanel();
    await searchFor(m, 'abc');
    await act(async () => {
      m.runs[0]?.onBatch(
        batch([
          {
            path: 'C:\\w\\notes\\a.md',
            matches: [match({ line: 12, column: 4, length: 3, text: 'say abc here' })],
          },
        ]),
      );
    });

    expect(m.container.querySelector('.search-file-name')?.textContent).toBe('a.md');
    expect(m.container.querySelector('.search-file-count')?.textContent).toBe('1');
    expect(hitsOf(m.container)).toHaveLength(1);
    expect(hitsOf(m.container)[0]?.querySelector('.search-mark')?.textContent).toBe('abc');
    expect(m.container.querySelector('.search-line-no')?.textContent).toBe('12');
  });

  it('零宽匹配画一根细竖线 —— 什么都不画就等于「搜到了但看不见」', async () => {
    const m = await mountPanel();
    await searchFor(m, '^');
    await act(async () => {
      m.runs[0]?.onBatch(batch([{ path: 'a.md', matches: [match({ column: 2, length: 0, text: 'abcd' })] }]));
    });

    const hit = hitsOf(m.container)[0];
    expect(hit?.querySelector('.search-caret')).not.toBeNull();
    expect(hit?.querySelector('.search-mark')).toBeNull();
    // 零宽匹配没有可高亮的字符，但上下文要在
    expect(hit?.textContent).toContain('abcd');
  });

  it('窗口被截断时两端画省略号', async () => {
    const m = await mountPanel();
    await searchFor(m, 'abc');
    await act(async () => {
      m.runs[0]?.onBatch(
        batch([
          {
            path: 'a.md',
            matches: [match({ column: 100, length: 3, text: 'xxabc yy', textOffset: 98, tailCut: true })],
          },
        ]),
      );
    });

    const hit = hitsOf(m.container)[0];
    expect(hit?.querySelectorAll('.search-ellipsis')).toHaveLength(2);
    expect(hit?.querySelector('.search-mark')?.textContent).toBe('abc');
  });

  it('点命中行把路径与命中位置交出去', async () => {
    const m = await mountPanel();
    await searchFor(m, 'abc');
    const target = match({ line: 3, column: 8 });
    await act(async () => {
      m.runs[0]?.onBatch(batch([{ path: 'C:\\w\\a.md', matches: [target] }]));
    });

    const row = hitsOf(m.container)[0];
    if (row === undefined) throw new Error('没有命中行');
    await click(row);

    expect(m.opened).toEqual([{ path: 'C:\\w\\a.md', match: target }]);
  });

  it('批次是追加的：两条不同批次的命中都要留着', async () => {
    const m = await mountPanel();
    await searchFor(m, 'abc');
    await act(async () => {
      m.runs[0]?.onBatch(batch([{ path: 'a.md', matches: [match()] }]));
    });
    await act(async () => {
      m.runs[0]?.onBatch(batch([{ path: 'b.md', matches: [match()] }]));
    });

    expect(hitsOf(m.container)).toHaveLength(2);
    expect(m.container.querySelectorAll('.search-file-name')).toHaveLength(2);
  });
});

describe('SearchPanel · 收尾与空结果', () => {
  it('收尾那行报命中数与扫描量', async () => {
    const m = await mountPanel();
    await searchFor(m, 'abc');
    await finish(m, 0, { totalMatches: 9, matchedFiles: 3, scannedFiles: 120, elapsedMs: 1500 });

    const text = m.container.querySelector('.search-status')?.textContent ?? '';
    expect(text).toContain('共 9 处，3 个文件');
    expect(text).toContain('扫了 120 个文件');
  });

  it('跳过的东西与命中数分开报出来', async () => {
    const m = await mountPanel();
    await searchFor(m, 'abc');
    await finish(m, 0, { skippedBinary: 2, skippedLarge: 1 });

    expect(m.container.querySelector('.search-skipped')?.textContent).toBe('跳过 2 个二进制、1 个超大文件');
  });

  it('被截断时必须说出来，否则用户会以为这就是全部', async () => {
    const m = await mountPanel();
    await searchFor(m, 'abc');
    await finish(m, 0, { reason: 'truncated', totalMatches: 2000 });

    expect(m.container.querySelector('.search-status')?.textContent).toContain('结果可能不全');
  });

  it('空结果给一段可行动的说明，而不是一句「没有命中」', async () => {
    const m = await mountPanel();
    await searchFor(m, '找不到的词');
    await finish(m, 0);

    expect(hitsOf(m.container)).toHaveLength(0);
    expect(m.container.querySelector('.search-note')?.textContent).toContain('⊙');
  });

  it('已经勾了「包含被忽略的文件」之后，那句提示要消失', async () => {
    const m = await mountPanel();
    await searchFor(m, '找不到的词');
    await finish(m, 0);

    // 点 ⊙ 会立刻重查（用户是主动点的），所以先点、再让新的那次收尾
    await click(toggleOf(m.container, '包含被忽略的文件'));
    await finish(m, 1);

    const note = m.container.querySelector('.search-note');
    expect(note?.textContent).toContain('没有命中');
    // 「试试打开上面那个 ⊙」在开关已经打开时是一句错话
    expect(note?.textContent).not.toContain('⊙');
  });

  it('⊙ 的说明文案跟着开关走', async () => {
    const m = await mountPanel();
    expect(m.container.querySelector('.search-hint')?.textContent).toContain('按 .gitignore');

    await click(toggleOf(m.container, '包含被忽略的文件'));
    expect(m.container.querySelector('.search-hint')?.textContent).toContain('已关掉 .gitignore');
  });
});

describe('SearchPanel · 其它', () => {
  it('关闭按钮把请求交出去', async () => {
    const m = await mountPanel();
    const close = m.container.querySelector<HTMLButtonElement>('.sidebar-head button');
    if (close === null) throw new Error('没有关闭按钮');
    await click(close);

    expect(m.closeCount()).toBe(1);
  });

  it('四个开关都带可访问名（纯符号按钮在测试与读屏里都得找得到）', async () => {
    const m = await mountPanel();
    const toggles = [...m.container.querySelectorAll('.search-toggle')];

    expect(toggles).toHaveLength(4);
    for (const toggle of toggles) {
      expect(toggle.querySelector('.sr-only')?.textContent).toBeTruthy();
      expect(toggle.getAttribute('aria-pressed')).toBe('false');
    }
  });

  it('切走再切回来（重新挂载）不会把查询丢掉 —— 状态在 store 里', async () => {
    const m = await mountPanel();
    await searchFor(m, '配置');

    await m.unmount();
    // 再挂一次，用同一个 store：面板自己没有状态，所以查询还在
    const container = document.createElement('div');
    document.body.appendChild(container);
    const root = createRoot(container);
    roots.push(root);
    containers.push(container);
    await act(async () => {
      root.render(
        <SearchPanel
          store={m.store}
          root="/root"
          canSearch
          onOpenMatch={() => undefined}
          onClose={() => undefined}
        />,
      );
    });

    expect(inputOf(container).value).toBe('配置');
  });
});
