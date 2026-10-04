/**
 * 工作区检索的前端核心。
 *
 * 这个文件里**大部分篇幅在测时序**（防抖、取消、过期批次），而不是测渲染 ——
 * 那正是把这一层从组件里抽出来的原因：三件时序事每一条写错了都不报错，
 * 表现是「结果里混着上一次的」「面板一直转」「点了没反应」，
 * 而它们用假执行者可以在几毫秒内精确复现。
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  DEFAULT_SEARCH_OPTIONS,
  EMPTY_PROGRESS,
  WorkspaceSearchStore,
  describeSkips,
  describeSummary,
  flattenResults,
  formatBytes,
  progressOf,
  splitPreview,
  type WorkspaceBatch,
  type WorkspaceFileMatches,
  type WorkspaceSearchRequest,
  type WorkspaceSearcher,
  type WorkspaceSummary,
} from './workspace.js';

// ---------------------------------------------------------------- 纯函数

function batchOf(files: WorkspaceFileMatches[], overrides: Partial<WorkspaceBatch> = {}): WorkspaceBatch {
  return {
    files,
    matchedFiles: files.length,
    totalMatches: files.reduce((sum, file) => sum + file.matches.length, 0),
    scannedFiles: 10,
    scannedBytes: 2048,
    skippedBinary: 0,
    skippedLarge: 0,
    skippedUnreadable: 0,
    ...overrides,
  };
}

describe('flattenResults', () => {
  it('把文件行与命中行拍平成一个数组', () => {
    const rows = flattenResults([
      {
        path: 'a.md',
        matches: [
          { line: 2, column: 0, length: 3, text: 'abc', textOffset: 0, tailCut: false },
          { line: 5, column: 1, length: 3, text: 'abc', textOffset: 0, tailCut: false },
        ],
      },
      { path: 'b.md', matches: [{ line: 1, column: 0, length: 3, text: 'abc', textOffset: 0, tailCut: false }] },
    ]);

    expect(rows.map((row) => row.kind)).toEqual(['file', 'match', 'match', 'file', 'match']);
    expect(rows[0]).toMatchObject({ kind: 'file', path: 'a.md', count: 2 });
    // a.md 占 0/1/2，所以 b.md 的文件行在 3、命中行在 4
    expect(rows[3]).toMatchObject({ kind: 'file', path: 'b.md', count: 1 });
    expect(rows[4]).toMatchObject({ kind: 'match', path: 'b.md' });
  });

  it('每一行的 key 都不重复', () => {
    const rows = flattenResults([
      {
        path: 'a.md',
        matches: [
          { line: 1, column: 0, length: 1, text: 'x', textOffset: 0, tailCut: false },
          { line: 1, column: 4, length: 1, text: 'x', textOffset: 0, tailCut: false },
          { line: 2, column: 0, length: 1, text: 'x', textOffset: 0, tailCut: false },
        ],
      },
    ]);
    expect(new Set(rows.map((row) => row.key)).size).toBe(rows.length);
  });
});

describe('splitPreview', () => {
  const base = { line: 1, column: 0, length: 0, text: '', textOffset: 0, tailCut: false };

  it('按列与长度切出命中段', () => {
    const parts = splitPreview({ ...base, column: 4, length: 3, text: 'say hit here' });
    expect(parts).toEqual({ before: 'say ', matched: 'hit', after: ' here' });
  });

  it('窗口被截断时按窗口起点换算', () => {
    const parts = splitPreview({ ...base, column: 100, length: 3, text: 'xxhit', textOffset: 98 });
    expect(parts).toEqual({ before: 'xx', matched: 'hit', after: '' });
  });

  it('零宽匹配切出空的命中段', () => {
    const parts = splitPreview({ ...base, column: 2, length: 0, text: 'abcd' });
    expect(parts).toEqual({ before: 'ab', matched: '', after: 'cd' });
  });

  it('命中比窗口还长时不切出重复内容', () => {
    // 一行几万字符被一次匹配掉：`length` 远大于 `text.length`，
    // 不夹取的话 `after` 会变成一段负长度的怪东西
    const parts = splitPreview({ ...base, column: 0, length: 50_000, text: 'yyy', tailCut: true });
    expect(parts.before + parts.matched + parts.after).toBe('yyy');
    expect(parts.matched).toBe('yyy');
    expect(parts.after).toBe('');
  });

  it('列号落在窗口之外时不会切出越界内容', () => {
    const parts = splitPreview({ ...base, column: 0, length: 2, text: 'abc', textOffset: 5 });
    expect(parts.before + parts.matched + parts.after).toBe('abc');
  });
});

describe('统计文案', () => {
  const summary: WorkspaceSummary = {
    reason: 'completed',
    matchedFiles: 3,
    totalMatches: 7,
    scannedFiles: 120,
    scannedBytes: 2 * 1024 * 1024,
    skippedBinary: 0,
    skippedLarge: 0,
    skippedUnreadable: 0,
    elapsedMs: 1500,
  };

  it('正常结束时报命中数与扫描量', () => {
    expect(describeSummary(summary)).toBe('共 7 处，3 个文件 · 扫了 120 个文件 · 2.0 MB · 1.5s');
  });

  it('被取消时说的是「已停止」而不是命中数', () => {
    expect(describeSummary({ ...summary, reason: 'cancelled' })).toContain('已停止');
  });

  it('到上限时必须说出来，否则用户会以为这就是全部', () => {
    expect(describeSummary({ ...summary, reason: 'truncated' })).toContain('结果可能不全');
  });

  it('跳过的东西一定要报出来', () => {
    const text = describeSummary({ ...summary, skippedBinary: 2, skippedLarge: 1, skippedUnreadable: 3 });
    expect(text).toContain('跳过 2 个二进制、1 个超大文件、3 个读不到');
  });

  it('一个都没跳过时不显示噪音', () => {
    expect(describeSkips(summary)).toBeNull();
  });

  it('字节格式化', () => {
    expect(formatBytes(512)).toBe('512 B');
    expect(formatBytes(2048)).toBe('2 KB');
    expect(formatBytes(3 * 1024 * 1024)).toBe('3.0 MB');
  });

  it('progressOf 只取进度字段（丢掉这一批的 files）', () => {
    const progress = progressOf(batchOf([], { scannedFiles: 3, scannedBytes: 99 }));
    expect(progress).toEqual({ ...EMPTY_PROGRESS, scannedFiles: 3, scannedBytes: 99 });
    // 批次里那一批文件不该混进进度对象 —— 面板拿它去渲染「扫了多少」时，
    // 多带一个 `files` 只会让两个来源看起来都能代表「结果」
    expect('files' in progress).toBe(false);
  });
});

// ---------------------------------------------------------------- 会话

interface PendingRun {
  readonly request: WorkspaceSearchRequest;
  readonly onBatch: (batch: WorkspaceBatch) => void;
  readonly resolve: (summary: WorkspaceSummary) => void;
  readonly reject: (error: unknown) => void;
}

function createFakeSearcher() {
  const runs: PendingRun[] = [];
  const cancels: number[] = [];
  const searcher: WorkspaceSearcher = {
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
  };
  return { searcher, runs, cancels };
}

function summaryOf(reason: WorkspaceSummary['reason'] = 'completed'): WorkspaceSummary {
  return { ...EMPTY_PROGRESS, reason, elapsedMs: 5 };
}

describe('WorkspaceSearchStore', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  function makeStore() {
    const fake = createFakeSearcher();
    const store = new WorkspaceSearchStore(fake.searcher, 250);
    store.setRoot('/root');
    return { ...fake, store };
  }

  it('没有根目录时不发起检索', () => {
    const fake = createFakeSearcher();
    const store = new WorkspaceSearchStore(fake.searcher, 250);
    store.setQuery('abc');
    vi.advanceTimersByTime(500);
    expect(fake.runs).toHaveLength(0);
    expect(store.getSnapshot().files).toEqual([]);
  });

  it('空查询不发起检索，也不报错', () => {
    const { runs, store } = makeStore();
    store.setQuery('');
    vi.advanceTimersByTime(500);
    expect(runs).toHaveLength(0);
    expect(store.getSnapshot().error).toBeNull();
  });

  it('连续打字只发起一次检索（防抖）', () => {
    const { runs, store } = makeStore();
    store.setQuery('配');
    vi.advanceTimersByTime(100);
    store.setQuery('配置');
    vi.advanceTimersByTime(100);
    store.setQuery('配置项');
    expect(runs).toHaveLength(0);
    vi.advanceTimersByTime(250);
    expect(runs).toHaveLength(1);
    expect(runs[0]?.request.query).toBe('配置项');
  });

  it('回车（searchNow）不等防抖', () => {
    const { runs, store } = makeStore();
    store.setQuery('abc');
    store.searchNow();
    expect(runs).toHaveLength(1);
    // 防抖那次不该再补发一次
    vi.advanceTimersByTime(1000);
    expect(runs).toHaveLength(1);
  });

  it('发起新的检索会取消上一次', () => {
    const { runs, cancels, store } = makeStore();
    store.setQuery('abc');
    vi.advanceTimersByTime(250);
    const firstId = runs[0]?.request.searchId ?? 0;

    store.setQuery('abcd');
    vi.advanceTimersByTime(250);

    expect(cancels).toEqual([firstId]);
    expect(runs).toHaveLength(2);
  });

  it('★ 过期批次必须被丢掉', async () => {
    const { runs, store } = makeStore();
    store.setQuery('abc');
    vi.advanceTimersByTime(250);
    const stale = runs[0];

    store.setQuery('abcd');
    vi.advanceTimersByTime(250);
    const fresh = runs[1];

    // 上一次的批次姗姗来迟 —— 放进来的话，用户会在「abcd」的结果里
    // 看到「abc」的命中，而且看起来完全正常
    stale?.onBatch(
      batchOf([
        { path: 'stale.md', matches: [{ line: 1, column: 0, length: 3, text: 'abc', textOffset: 0, tailCut: false }] },
      ]),
    );
    fresh?.onBatch(
      batchOf([
        { path: 'fresh.md', matches: [{ line: 9, column: 0, length: 4, text: 'abcd', textOffset: 0, tailCut: false }] },
      ]),
    );

    expect(store.getSnapshot().files.map((file) => file.path)).toEqual(['fresh.md']);
  });

  it('批次是追加的，进度是替换的', () => {
    const { runs, store } = makeStore();
    store.setQuery('abc');
    vi.advanceTimersByTime(250);
    const run = runs[0];

    run?.onBatch(batchOf([{ path: 'a.md', matches: [] }], { scannedFiles: 10 }));
    run?.onBatch(batchOf([{ path: 'b.md', matches: [] }], { scannedFiles: 25 }));

    const state = store.getSnapshot();
    expect(state.files.map((file) => file.path)).toEqual(['a.md', 'b.md']);
    // 进度是累计值，累加会变成 35
    expect(state.progress.scannedFiles).toBe(25);
  });

  it('结局之后到来的批次不许让进度回退', async () => {
    const { runs, store } = makeStore();
    store.setQuery('abc');
    vi.advanceTimersByTime(250);
    const run = runs[0];

    run?.onBatch(batchOf([], { scannedFiles: 10 }));
    run?.resolve({ ...summaryOf(), scannedFiles: 120, totalMatches: 2, matchedFiles: 2 });
    await Promise.resolve();

    // Rust 侧发完最后一批就返回了，所以「结局先到、最后一批后到」是可能的
    run?.onBatch(batchOf([{ path: 'late.md', matches: [] }], { scannedFiles: 118 }));

    const state = store.getSnapshot();
    // 那一批的 files 该收 —— 它确实是结果的一部分
    expect(state.files.map((file) => file.path)).toEqual(['late.md']);
    // 但计数不能被它拉回去
    expect(state.progress.scannedFiles).toBe(120);
  });

  it('失败时 running 必须放掉，并留下错误', async () => {
    const { runs, store } = makeStore();
    store.setQuery('abc');
    vi.advanceTimersByTime(250);
    expect(store.getSnapshot().running).toBe(true);

    runs[0]?.reject({ code: 'INVALID_PATH', message: '正则表达式无效' });
    await Promise.resolve();
    await Promise.resolve();

    const state = store.getSnapshot();
    expect(state.running).toBe(false);
    expect(state.error?.message).toBe('正则表达式无效');
  });

  it('取消会立刻放掉 running，但保留已有结果', async () => {
    const { runs, cancels, store } = makeStore();
    store.setQuery('abc');
    vi.advanceTimersByTime(250);
    runs[0]?.onBatch(batchOf([{ path: 'a.md', matches: [] }]));

    store.cancel();
    expect(store.getSnapshot().running).toBe(false);
    expect(store.getSnapshot().files.map((file) => file.path)).toEqual(['a.md']);
    expect(cancels).toHaveLength(1);
  });

  it('换根目录会清掉上一次的结果', () => {
    const { runs, store } = makeStore();
    store.setQuery('abc');
    vi.advanceTimersByTime(250);
    runs[0]?.onBatch(batchOf([{ path: 'a.md', matches: [] }]));
    expect(store.getSnapshot().files).toHaveLength(1);

    store.setRoot('/other');
    expect(store.getSnapshot().files).toEqual([]);
    expect(store.getSnapshot().summary).toBeNull();
  });

  it('setRoot 传同一个值时不打断进行中的检索', () => {
    const { runs, cancels, store } = makeStore();
    store.setQuery('abc');
    vi.advanceTimersByTime(250);
    expect(runs).toHaveLength(1);

    store.setRoot('/root');
    expect(cancels).toHaveLength(0);
    expect(store.getSnapshot().running).toBe(true);
  });

  it('改开关会立刻重查（用户是主动点的，不该再等防抖）', () => {
    const { runs, store } = makeStore();
    store.setQuery('abc');
    vi.advanceTimersByTime(250);
    expect(runs).toHaveLength(1);

    store.setOption('regex', true);
    expect(runs).toHaveLength(2);
    expect(runs[1]?.request.regex).toBe(true);
  });

  it('关掉面板（dispose）之后不再有回调', () => {
    const { runs, store } = makeStore();
    const seen: number[] = [];
    store.subscribe(() => seen.push(1));
    store.setQuery('abc');
    vi.advanceTimersByTime(250);

    store.dispose();
    const before = seen.length;
    runs[0]?.onBatch(batchOf([{ path: 'a.md', matches: [] }]));
    expect(seen.length).toBe(before);
  });

  it('clear 清空结果但保留根目录与开关', () => {
    const { runs, store } = makeStore();
    store.setOption('caseSensitive', true);
    store.setQuery('abc');
    vi.advanceTimersByTime(250);
    runs[0]?.onBatch(batchOf([{ path: 'a.md', matches: [] }]));

    store.clear();
    const state = store.getSnapshot();
    expect(state.files).toEqual([]);
    expect(state.query).toBe('');
    expect(state.root).toBe('/root');
    expect(state.options.caseSensitive).toBe(true);
  });

  it('默认开关全关', () => {
    expect(DEFAULT_SEARCH_OPTIONS).toEqual({
      caseSensitive: false,
      wholeWord: false,
      regex: false,
      includeIgnored: false,
    });
  });
});
