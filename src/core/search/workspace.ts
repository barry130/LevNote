/**
 * 工作区检索（跨文件）的前端核心。
 *
 * ## 它解决的是「谁来管这一次检索」
 *
 * 检索与「在当前文档里查找」是两件完全不同的事：
 *
 * | | 文档内查找（`find.ts`） | 工作区检索（这里） |
 * |---|---|---|
 * | 范围 | 一份文档，内存里 | 一整个目录，磁盘上 |
 * | 耗时 | 微秒级 | 秒级 |
 * | 形态 | 同步返回全部匹配 | **流式**，分批回来 |
 * | 谁在算 | 内核自己 | Rust 侧的阻塞线程 |
 *
 * 所以这一层不是「把 `find.ts` 搬过来」，而是**一个会话**：
 * 有防抖、有取消、有「这一批结果属于哪一次检索」的对号。
 *
 * ## 三个必须由这一层管住的东西
 *
 * 1. **防抖**。用户打「配置」两个字，中间那一次「配」不该发起检索 ——
 *    每次按键扫一遍目录，机器风扇替用户记账。
 * 2. **取消**。新的一次检索发起时，上一次必须真的停下来（Rust 侧置位），
 *    而不是「结果丢掉、活照干」。
 * 3. **过期批次**。Rust 侧是流式的，被取消的那一次很可能还有几批在路上。
 *    它们**必须被丢掉**，否则用户会看到上一次检索的结果混进这一次里 ——
 *    而且混进来的那几条看起来完全正常，没有任何报错。
 *    对号的依据是 `searchId`。
 *
 * ## 为什么是一个 store 而不是写在组件里
 *
 * 上面三条每一条都是「写错了不报错」的类型（尤其是第 3 条：结果是错的、
 * 但长得和真的一样）。放在组件里就只能靠渲染测试去够它，而这三条全是
 * **时序**问题 —— 组件测试里的时序是最难写稳的东西。抽成一个纯 JS 的 store
 * 之后，它们可以用假 searcher 精确地测（见 `workspace.test.ts`）。
 */

import type { AppErrorShape } from '../document/types.js';
import { t } from '../i18n/index.js';
import { asAppError } from '../document/types.js';

/** 对应 Rust `search::SearchReason`。 */
export type SearchReason = 'completed' | 'cancelled' | 'truncated';

/**
 * 一处命中。
 *
 * `column` / `length` 的单位是 **UTF-16 码元**（= JS 字符串下标），
 * 与 `EditorHandle.revealSourceLocation` 的口径一致。
 * 理由见 `src-tauri/src/search/mod.rs` 的模块注释 —— 简言之：
 * 下游是 JS 与 CodeMirror，用别的单位都要在某一侧做一次转换，
 * 而那次转换正是「跳到命中右边一格」的来源。
 */
export interface WorkspaceMatch {
  /** 1 基行号。 */
  readonly line: number;
  /** 命中在整行里的起始列（0 基，UTF-16 码元）。 */
  readonly column: number;
  /** 命中长度（UTF-16 码元）。零宽匹配是 0。 */
  readonly length: number;
  /** 给用户看的整行（可能被截断）。 */
  readonly text: string;
  /** `text` 从整行的第几个码元开始；大于 0 表示前面被截掉了。 */
  readonly textOffset: number;
  /** `text` 后面还有没显示出来的内容。 */
  readonly tailCut: boolean;
}

/** 一个文件里的全部命中。 */
export interface WorkspaceFileMatches {
  readonly path: string;
  readonly matches: readonly WorkspaceMatch[];
}

/** 截至此刻的进度。 */
export interface WorkspaceProgress {
  readonly matchedFiles: number;
  readonly totalMatches: number;
  readonly scannedFiles: number;
  readonly scannedBytes: number;
  /** 前 8KB 含 NUL，当成二进制跳过。 */
  readonly skippedBinary: number;
  /** 超过单文件上限。 */
  readonly skippedLarge: number;
  /** 读不出来（权限、断链、解码失败）。 */
  readonly skippedUnreadable: number;
}

/** 收尾统计。 */
export interface WorkspaceSummary extends WorkspaceProgress {
  readonly reason: SearchReason;
  readonly elapsedMs: number;
}

/** 一批结果 + 截至此刻的进度。 */
export interface WorkspaceBatch extends WorkspaceProgress {
  readonly files: readonly WorkspaceFileMatches[];
}

/** 检索开关。与文档内查找的 `FindOptions` 前三项同名同义。 */
export interface WorkspaceSearchOptions {
  readonly caseSensitive: boolean;
  readonly wholeWord: boolean;
  readonly regex: boolean;
  /**
   * 把被 `.gitignore` 之类规则忽略的文件也搜进来。
   *
   * **它关掉的只是「规则」，隐藏项（以 `.` 开头）仍然跳过** ——
   * 否则 `.git/` 会被整棵搜一遍。Rust 侧也是这么做的（`walk.rs`）。
   */
  readonly includeIgnored: boolean;
}

export const DEFAULT_SEARCH_OPTIONS: WorkspaceSearchOptions = {
  caseSensitive: false,
  wholeWord: false,
  regex: false,
  includeIgnored: false,
};

export const EMPTY_PROGRESS: WorkspaceProgress = {
  matchedFiles: 0,
  totalMatches: 0,
  scannedFiles: 0,
  scannedBytes: 0,
  skippedBinary: 0,
  skippedLarge: 0,
  skippedUnreadable: 0,
};

/** 一次检索的请求。`searchId` 由前端生成（见 `WorkspaceSearchStore`）。 */
export interface WorkspaceSearchRequest {
  readonly searchId: number;
  readonly root: string;
  readonly query: string;
  readonly caseSensitive: boolean;
  readonly wholeWord: boolean;
  readonly regex: boolean;
  readonly includeIgnored: boolean;
}

/**
 * 平台侧的执行者。
 *
 * 与 `DocumentProvider` 分开而不是并进去：它们是**两条独立的链路** ——
 * 浏览器预览有内存虚拟盘（`DocumentProvider` 能用），但它没有目录遍历，
 * 检索在那里只能明确地不可用。混在一个接口里会让「能不能检索」
 * 变成一个需要逐个方法探测的问题。
 */
export interface WorkspaceSearcher {
  /**
   * 这个平台能不能做工作区检索。
   *
   * ## 为什么是执行者自己说，而不是让外壳问 `isDesktop()`
   *
   * 「能不能检索」与「拿到哪个执行者」是**同一个判断**。让外壳另问一次平台，
   * 就是同一个问题有两份答案，而两份之间没有任何东西保证它们一致 ——
   * 表现是**按钮亮着、点下去被明确拒绝**，且两边代码都看不出错。
   * 放在这里之后只有一处：执行者自己是那个「明确拒绝」的实现，它最清楚。
   * （`core/platform/index.ts` 的 `createWorkspaceSearcher()` 因此也只需要一处判断。）
   */
  readonly available: boolean;
  /**
   * 跑一次检索。返回的 promise 表示这次检索已经结束
   * （正常结束 / 被取消 / 到上限**都算结束**，都是 resolve）。
   *
   * 开工前的错误（没有文件夹、检索词为空、正则写错）走 reject。
   */
  run(
    request: WorkspaceSearchRequest,
    onBatch: (batch: WorkspaceBatch) => void,
  ): Promise<WorkspaceSummary>;
  /** 请求取消。返回的 promise 只表示「取消请求已送达」，不表示检索已经停了。 */
  cancel(searchId: number): Promise<void>;
}

/** 平台不支持检索时的执行者（浏览器预览）。 */
export const UNSUPPORTED_SEARCHER: WorkspaceSearcher = {
  available: false,
  run() {
    return Promise.reject(
      new Error(t('core.search.unsupported')),
    );
  },
  cancel() {
    return Promise.resolve();
  },
};

// ---------------------------------------------------------------- 纯函数

/** 从一批结果里取出进度部分。 */
export function progressOf(batch: WorkspaceBatch): WorkspaceProgress {
  return {
    matchedFiles: batch.matchedFiles,
    totalMatches: batch.totalMatches,
    scannedFiles: batch.scannedFiles,
    scannedBytes: batch.scannedBytes,
    skippedBinary: batch.skippedBinary,
    skippedLarge: batch.skippedLarge,
    skippedUnreadable: batch.skippedUnreadable,
  };
}

/**
 * 结果列表里的一行。
 *
 * 文件行与命中行拍平成一个数组（而不是嵌套渲染）：面板是**长列表**，
 * 用户要能一路滚下去、还要能被「跳转到第 N 条」这类动作定位。
 * 嵌套结构下每一条命中都要靠父级重新挂载，滚动位置与选中态都会跟着抖。
 */
export type ResultRow =
  | {
      readonly kind: 'file';
      readonly key: string;
      readonly path: string;
      /** 这个文件里有多少处命中。 */
      readonly count: number;
    }
  | {
      readonly kind: 'match';
      readonly key: string;
      readonly path: string;
      readonly match: WorkspaceMatch;
      /** 这个文件里的第几处（0 基），用来做 key 与「跳转到第几处」。 */
      readonly index: number;
    };

export function flattenResults(files: readonly WorkspaceFileMatches[]): ResultRow[] {
  const rows: ResultRow[] = [];
  for (const file of files) {
    rows.push({ kind: 'file', key: `f:${file.path}`, path: file.path, count: file.matches.length });
    file.matches.forEach((match, index) => {
      rows.push({ kind: 'match', key: `m:${file.path}:${String(match.line)}:${String(match.column)}`, path: file.path, match, index });
    });
  }
  return rows;
}

/** 结果里一共有多少处命中（用于「共 N 处」）。 */
export function totalMatchesOf(files: readonly WorkspaceFileMatches[]): number {
  let total = 0;
  for (const file of files) total += file.matches.length;
  return total;
}

/**
 * 收尾那句话。
 *
 * **必须把「跳过」说出来**：用户看不到某个文件时，至少要能分辨
 * 「它被跳过了」和「它里面真的没有这个词」—— 这正是硬约束 2
 * 在检索这条链路上的落点。
 */
export function describeSummary(summary: WorkspaceSummary): string {
  const parts: string[] = [];
  if (summary.reason === 'cancelled') parts.push(t('core.search.stopped'));
  else if (summary.reason === 'truncated') parts.push(t('core.search.truncated', { n: summary.totalMatches }));
  else parts.push(t('core.search.summary', { matches: summary.totalMatches, files: summary.matchedFiles }));

  parts.push(t('core.search.scanned', { n: summary.scannedFiles }));
  parts.push(`${formatBytes(summary.scannedBytes)}`);
  parts.push(`${(summary.elapsedMs / 1000).toFixed(1)}s`);

  const skipped = describeSkips(summary);
  if (skipped !== null) parts.push(skipped);

  return parts.join(' · ');
}

/** 跳过统计。一个都没跳过时返回 null（不要显示「跳过了 0 个」这种噪音）。 */
export function describeSkips(summary: WorkspaceProgress): string | null {
  const items: string[] = [];
  if (summary.skippedBinary > 0) items.push(t('core.search.skipBinary', { n: summary.skippedBinary }));
  if (summary.skippedLarge > 0) items.push(t('core.search.skipLarge', { n: summary.skippedLarge }));
  if (summary.skippedUnreadable > 0) items.push(t('core.search.skipUnreadable', { n: summary.skippedUnreadable }));
  return items.length === 0 ? null : t('core.search.skippedAll', { items: items.join('、') });
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${String(bytes)} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

/**
 * 把预览切成「命中前 / 命中 / 命中后」三段，供高亮渲染。
 *
 * 为什么不用 `indexOf` 在预览里找检索词：命中**未必等于检索词** ——
 * 勾了「区分大小写」时 `ABC` 命中的是 `ABC`，而正则模式下命中的东西
 * 跟检索词可以毫无字面关系（`\d+`）。Rust 侧已经把**精确的列与长度**算好了，
 * 在 JS 里再猜一次只会猜错。所以这里只做切片，不做匹配。
 *
 * 两处夹取都不能省：`text` 是截断过的窗口，命中极长时（一行几万字符被
 * 一次匹配掉）会被 `PREVIEW_MAX` 卡住 —— 不夹的话 `matched` 会切出
 * 比 `text` 还长的串，渲染出一段重复的文字。
 */
export function splitPreview(match: WorkspaceMatch): {
  readonly before: string;
  readonly matched: string;
  readonly after: string;
} {
  const raw = match.column - match.textOffset;
  const start = Math.max(0, Math.min(raw, match.text.length));
  const end = Math.max(start, Math.min(start + match.length, match.text.length));
  return {
    before: match.text.slice(0, start),
    matched: match.text.slice(start, end),
    after: match.text.slice(end),
  };
}

// ---------------------------------------------------------------- 会话

/**
 * 面板要渲染的全部状态。
 *
 * `files` 是**追加**的（批次一条条来），`progress` 是**替换**的
 * （Rust 侧报的是累计值）—— 这两种更新方式必须分清：
 * 把 `files` 也替换掉，第一批之后的结果就全没了；
 * 把 `progress` 累加，数字会翻好几倍。
 */
export interface WorkspaceSearchState {
  readonly query: string;
  readonly options: WorkspaceSearchOptions;
  readonly root: string | null;
  readonly files: readonly WorkspaceFileMatches[];
  readonly progress: WorkspaceProgress;
  readonly summary: WorkspaceSummary | null;
  readonly error: AppErrorShape | null;
  readonly running: boolean;
  /** 当前这次检索的编号；0 表示还没有发起过。 */
  readonly searchId: number;
}

export const EMPTY_SEARCH_STATE: WorkspaceSearchState = {
  query: '',
  options: DEFAULT_SEARCH_OPTIONS,
  root: null,
  files: [],
  progress: EMPTY_PROGRESS,
  summary: null,
  error: null,
  running: false,
  searchId: 0,
};

/**
 * 输入停手多久才真的发起检索。
 *
 * 250ms：比位置记忆的 600ms 短（那是写盘，越少越好；这是查，越早越好），
 * 但足够吞掉连续打字。回车键不受它限制（见 `searchNow`）——
 * 「我打完了」是一个明确的信号，不该再等。
 */
export const SEARCH_DEBOUNCE_MS = 250;

export class WorkspaceSearchStore {
  private readonly searcher: WorkspaceSearcher;
  private readonly debounceMs: number;
  private state: WorkspaceSearchState = EMPTY_SEARCH_STATE;
  private listeners = new Set<() => void>();
  private timer: ReturnType<typeof setTimeout> | null = null;
  private nextId = 1;

  constructor(searcher: WorkspaceSearcher, debounceMs: number = SEARCH_DEBOUNCE_MS) {
    this.searcher = searcher;
    this.debounceMs = debounceMs;
  }

  getSnapshot = (): WorkspaceSearchState => this.state;

  subscribe = (listener: () => void): (() => void) => {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  };

  private set(patch: Partial<WorkspaceSearchState>): void {
    this.state = { ...this.state, ...patch };
    for (const listener of this.listeners) listener();
  }

  /**
   * 工作区根目录变了（打开 / 关闭文件夹）。
   *
   * **必须把上一次的结果清掉**：那是另一个目录的结果，留着它比留一片空白更糟 ——
   * 用户会点进去，然后打开一个跟他要找的东西无关的文件。
   */
  setRoot(root: string | null): void {
    if (root === this.state.root) return;
    this.cancelTimer();
    void this.cancelRunning();
    this.state = { ...EMPTY_SEARCH_STATE, query: this.state.query, options: this.state.options, root };
    for (const listener of this.listeners) listener();
  }

  /** 改检索词。会排一次防抖。 */
  setQuery(query: string): void {
    if (query === this.state.query) return;
    this.set({ query });
    this.schedule();
  }

  setOption<K extends keyof WorkspaceSearchOptions>(key: K, value: WorkspaceSearchOptions[K]): void {
    if (this.state.options[key] === value) return;
    this.set({ options: { ...this.state.options, [key]: value } });
    // 开关变了要**立刻**重查：用户是主动点了一个开关，不是还在打字
    this.searchNow();
  }

  /** 立刻发起（回车 / 点开关）。 */
  searchNow(): void {
    this.cancelTimer();
    void this.cancelRunning();
    this.start();
  }

  /** 停止当前检索。结果**保留**（用户可能只是想让它停下来）。 */
  cancel(): void {
    this.cancelTimer();
    void this.cancelRunning();
  }

  /** 清空（关面板 / 清空输入）。 */
  clear(): void {
    this.cancelTimer();
    void this.cancelRunning();
    this.state = { ...EMPTY_SEARCH_STATE, root: this.state.root, options: this.state.options };
    for (const listener of this.listeners) listener();
  }

  dispose(): void {
    this.cancelTimer();
    void this.cancelRunning();
    this.listeners.clear();
  }

  private cancelTimer(): void {
    if (this.timer !== null) {
      clearTimeout(this.timer);
      this.timer = null;
    }
  }

  private schedule(): void {
    this.cancelTimer();
    this.timer = setTimeout(() => {
      this.timer = null;
      void this.cancelRunning();
      this.start();
    }, this.debounceMs);
  }

  /**
   * 让当前这次检索停下来。
   *
   * **刻意不 await**：调用方（`setQuery` / `schedule`）在热路径上，
   * 而取消是一次「告诉 Rust 别再算了」的通知，它的往返结果不改变本地的任何状态。
   * 真正的隔离靠 `searchId` —— 就算取消通知晚到、旧的那一次跑完了，
   * 它的批次也会因为编号对不上被丢掉。
   */
  private cancelRunning(): Promise<void> {
    const { running, searchId } = this.state;
    if (!running || searchId === 0) return Promise.resolve();
    // 立刻把 running 放掉，但**不清结果** —— 清掉会让面板在用户打字时闪白
    this.set({ running: false, summary: null });
    return this.searcher.cancel(searchId).catch(() => undefined);
  }

  private start(): void {
    const { query, root, options } = this.state;
    if (root === null || query === '') {
      // 没有可查的东西：把上一次的结果收掉，但不要报错 ——
      // 「还没打开文件夹」不是一个错误，面板自己会画一句说明
      this.state = { ...this.state, files: [], progress: EMPTY_PROGRESS, summary: null, error: null, running: false };
      for (const listener of this.listeners) listener();
      return;
    }

    const searchId = this.nextId++;
    this.set({
      searchId,
      running: true,
      files: [],
      progress: EMPTY_PROGRESS,
      summary: null,
      error: null,
    });

    const request: WorkspaceSearchRequest = {
      searchId,
      root,
      query,
      caseSensitive: options.caseSensitive,
      wholeWord: options.wholeWord,
      regex: options.regex,
      includeIgnored: options.includeIgnored,
    };

    void this.searcher
      .run(request, (batch) => {
        // ★ 过期批次：被取消 / 被顶替的那一次还会送来几批。
        // 放进来就会变成「上一次检索的结果混进这一次」，而且看起来完全正常。
        if (this.state.searchId !== searchId) return;
        this.set({
          files: [...this.state.files, ...batch.files],
          // ★ 收尾之后不再让进度**回退**。
          // Rust 侧发完最后一批就返回了，所以「结局先到、最后一批后到」是可能的 ——
          // 那一批的 `files` 该收（它确实是结果的一部分），
          // 但它的计数是**收尾之前**的快照，用它覆盖会把「扫了 1200 个文件」
          // 变成「扫了 1187 个文件」。
          progress: this.state.summary === null ? progressOf(batch) : this.state.progress,
        });
      })
      .then((summary) => {
        if (this.state.searchId !== searchId) return;
        this.set({
          running: false,
          summary,
          progress: {
            matchedFiles: summary.matchedFiles,
            totalMatches: summary.totalMatches,
            scannedFiles: summary.scannedFiles,
            scannedBytes: summary.scannedBytes,
            skippedBinary: summary.skippedBinary,
            skippedLarge: summary.skippedLarge,
            skippedUnreadable: summary.skippedUnreadable,
          },
        });
      })
      .catch((thrown: unknown) => {
        if (this.state.searchId !== searchId) return;
        // 失败也要把 running 放掉，否则面板会永远停在「检索中」，
        // 而唯一的出路是关掉面板
        this.set({ running: false, error: asAppError(thrown) });
      });
  }
}
