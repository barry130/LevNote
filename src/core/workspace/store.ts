/**
 * 工作区模型：一个窗口里装下多个文档。
 *
 * ## 为什么不是「把 DocumentStore 改成数组」
 *
 * `DocumentStore` 已经把「一份文档的生命周期」做对了（脏判定只认内容、mtime 乐观锁、
 * 冲突要用户决策、编码与行尾是文件属性）。把它改成数组意味着**重写**这套语义，
 * 而它的每一条都是踩过坑才定下来的（见 ADR 0003 / 0009 与 `store.ts` 的注释）。
 *
 * 这里选择**组合**：一个标签页 = 一个独立的 `DocumentStore` 实例。
 * 于是「未保存改动」天然是**每标签页**的，不会出现「A 文档的改动把 B 文档也标脏」；
 * 每个标签页各自持有自己的 mtime 基准，保存时的乐观锁不会串台。
 * 代价是标签页之间不能共享撤销栈 —— 而它们本来就不该共享。
 *
 * ## 与 horseMD 的差别（这是刻意保留的差别）
 *
 * horseMD 用 `path → tab` 的映射做去重，同一个文件只会有一个标签页。
 * 这里同样按路径去重（`open` 命中已有标签页就激活它），因为「同一个文件开两遍、
 * 两边各改一半」在桌面端几乎总是误操作。**但没有做**「文件被外部程序删除后
 * 自动关闭标签页」这类自作主张的行为 —— 标签页只有在用户明确关闭时才消失。
 */

import type { AppError, DirEntry, DocumentProvider } from '../document/types.js';
import { DocumentStore, type DocumentStatus } from '../document/store.js';
import { basename } from '../path.js';
import { ancestorsOf } from './tree.js';

export interface TabSummary {
  readonly id: number;
  readonly path: string;
  readonly displayName: string;
  readonly dirty: boolean;
  readonly status: DocumentStatus;
  readonly readonly: boolean;
}

export interface TreeState {
  /** 工作区根目录。null = 没打开文件夹。 */
  readonly root: string | null;
  /** 已加载的目录 → 其直接子项。懒加载：只有展开过的目录才在这里。 */
  readonly dirs: Readonly<Record<string, readonly DirEntry[]>>;
  readonly expanded: readonly string[];
  readonly loading: readonly string[];
  /** 是否显示 node_modules / target 这类默认折叠项 */
  readonly showIgnored: boolean;
  readonly error: AppError | null;
}

export interface WorkspaceState {
  readonly tabs: readonly TabSummary[];
  readonly activeId: number | null;
  readonly tree: TreeState;
  /** 最近一次打开失败。UI 弹一次之后应调 `clearOpenError`。 */
  readonly openError: AppError | null;
}

interface Tab {
  readonly id: number;
  readonly store: DocumentStore;
  readonly unsubscribe: () => void;
}

const EMPTY_TREE: TreeState = {
  root: null,
  dirs: {},
  expanded: [],
  loading: [],
  showIgnored: false,
  error: null,
};

function summarize(id: number, store: DocumentStore): TabSummary | null {
  const { doc, status } = store.getSnapshot();
  if (!doc) return null;
  return {
    id,
    path: doc.path,
    displayName: doc.displayName,
    dirty: status === 'dirty',
    status,
    readonly: doc.readonly,
  };
}

function sameSummaries(a: readonly TabSummary[], b: readonly TabSummary[]): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i += 1) {
    const x = a[i];
    const y = b[i];
    if (x === undefined || y === undefined) return false;
    if (
      x.id !== y.id ||
      x.path !== y.path ||
      x.displayName !== y.displayName ||
      x.dirty !== y.dirty ||
      x.status !== y.status ||
      x.readonly !== y.readonly
    ) {
      return false;
    }
  }
  return true;
}

export class WorkspaceStore {
  private readonly provider: DocumentProvider;
  private tabs: Tab[] = [];
  private activeId: number | null = null;
  private nextId = 1;
  private listeners = new Set<() => void>();
  private tree: TreeState = EMPTY_TREE;
  private state: WorkspaceState;

  constructor(provider: DocumentProvider) {
    this.provider = provider;
    this.state = { tabs: [], activeId: null, tree: EMPTY_TREE, openError: null };
  }

  getSnapshot = (): WorkspaceState => this.state;

  subscribe = (listener: () => void): (() => void) => {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  };

  private notify(): void {
    for (const listener of this.listeners) listener();
  }

  /**
   * 按子 store 的最新状态重建标签页摘要。
   *
   * 每个标签页的子 store 每敲一个字都会通知一次，所以这里必须**先比对再替换** ——
   * 摘要没变却换了 state 对象，会让 `useSyncExternalStore` 认为有变化而整棵树重渲染，
   * 结果是「打字时侧栏和标签栏跟着闪」。
   */
  private syncTabs(): void {
    const tabs: TabSummary[] = [];
    for (const tab of this.tabs) {
      const summary = summarize(tab.id, tab.store);
      if (summary !== null) tabs.push(summary);
    }
    if (sameSummaries(tabs, this.state.tabs) && this.state.activeId === this.activeId) return;
    this.state = { ...this.state, tabs, activeId: this.activeId };
    this.notify();
  }

  private setTree(patch: Partial<TreeState>): void {
    this.tree = { ...this.tree, ...patch };
    this.state = { ...this.state, tree: this.tree };
    this.notify();
  }

  private tabOf(id: number | null): Tab | null {
    if (id === null) return null;
    return this.tabs.find((tab) => tab.id === id) ?? null;
  }

  /** 当前活动标签页的文档 store。没有打开任何文档时为 null。 */
  get activeStore(): DocumentStore | null {
    return this.tabOf(this.activeId)?.store ?? null;
  }

  get activeTab(): TabSummary | null {
    return this.state.tabs.find((tab) => tab.id === this.activeId) ?? null;
  }

  storeOf(id: number): DocumentStore | null {
    return this.tabOf(id)?.store ?? null;
  }

  clearOpenError = (): void => {
    if (this.state.openError === null) return;
    this.state = { ...this.state, openError: null };
    this.notify();
  };

  // ---- 标签页 -----------------------------------------------------------

  /**
   * 打开一个路径。
   *
   * 已经在标签页里的路径**不重复打开**，只把它激活 —— 这是刻意的：
   * 同一个文件在两个标签页里各改一半，会让 mtime 乐观锁在保存时抛冲突，
   * 而那个冲突对用户来说完全无法理解（「我明明只改了一个地方」）。
   *
   * 打开失败时**不留下一个空标签页**：失败的 store 直接丢弃，
   * 错误放进 `openError` 由外壳呈现。留一个空标签页只会让用户困惑
   * 「这个标签页是什么」。
   */
  async open(path: string): Promise<boolean> {
    const existing = this.tabs.find((tab) => tab.store.getSnapshot().doc?.path === path);
    if (existing) {
      this.activate(existing.id);
      return true;
    }

    const store = new DocumentStore(this.provider);
    const ok = await store.open(path);
    if (!ok) {
      const error = store.getSnapshot().error;
      store.dispose();
      this.state = { ...this.state, openError: error };
      this.notify();
      return false;
    }

    const id = this.nextId;
    this.nextId += 1;
    const unsubscribe = store.subscribe(() => this.syncTabs());
    this.tabs = [...this.tabs, { id, store, unsubscribe }];
    this.activeId = id;
    this.syncTabs();

    // 打开文件时把它所在的目录链展开，否则用户看到的是一个和侧栏无关的文档
    if (this.tree.root !== null) await this.revealInTree(path);
    return true;
  }

  async openViaDialog(): Promise<boolean> {
    const path = await this.provider.pickOpenPath();
    if (path === null) return false;
    return this.open(path);
  }

  activate(id: number): void {
    if (this.activeId === id) return;
    if (this.tabOf(id) === null) return;
    this.activeId = id;
    this.syncTabs();
  }

  activateAt(index: number): void {
    const tab = this.tabs[index];
    if (tab) this.activate(tab.id);
  }

  /** 循环切换标签页。`delta` 为 1 是下一个，-1 是上一个。 */
  cycle(delta: number): void {
    if (this.tabs.length === 0) return;
    const at = this.tabs.findIndex((tab) => tab.id === this.activeId);
    const from = at < 0 ? 0 : at;
    const size = this.tabs.length;
    const next = (((from + delta) % size) + size) % size;
    const tab = this.tabs[next];
    if (tab) this.activate(tab.id);
  }

  /**
   * 关闭标签页。
   *
   * `force` 为 false 且有未保存改动时返回 false，**由 UI 先去问用户**。
   * 这条与 `DocumentStore.close` 的语义一致：需要用户决策的事不在 store 里替他决定。
   */
  closeTab(id: number, force = false): boolean {
    const at = this.tabs.findIndex((tab) => tab.id === id);
    if (at < 0) return true;
    const tab = this.tabs[at];
    if (!tab) return true;

    if (!force && tab.store.getSnapshot().status === 'dirty') return false;

    tab.unsubscribe();
    tab.store.dispose();
    this.tabs = this.tabs.filter((item) => item.id !== id);

    if (this.activeId === id) {
      // 优先接管右边的标签页，没有就接管左边 —— 与主流编辑器的行为一致
      const next = this.tabs[at] ?? this.tabs[at - 1] ?? null;
      this.activeId = next?.id ?? null;
    }
    this.syncTabs();
    return true;
  }

  closeActive(force = false): boolean {
    if (this.activeId === null) return true;
    return this.closeTab(this.activeId, force);
  }

  closeOthers(keep: number): void {
    for (const tab of [...this.tabs]) {
      if (tab.id !== keep) this.closeTab(tab.id, true);
    }
  }

  /** 拖拽重排。`to` 是目标下标（重排后的位置）。 */
  moveTab(from: number, to: number): void {
    if (from === to) return;
    if (from < 0 || from >= this.tabs.length) return;
    const clamped = Math.max(0, Math.min(this.tabs.length - 1, to));
    const next = [...this.tabs];
    const [moved] = next.splice(from, 1);
    if (!moved) return;
    next.splice(clamped, 0, moved);
    this.tabs = next;
    this.syncTabs();
  }

  /** 保存当前标签页。 */
  async save(): Promise<boolean> {
    return (await this.activeStore?.save()) ?? false;
  }

  async saveAs(): Promise<boolean> {
    const ok = (await this.activeStore?.saveAs()) ?? false;
    if (ok) this.syncTabs();
    return ok;
  }

  async reload(): Promise<boolean> {
    return (await this.activeStore?.reload()) ?? false;
  }

  /**
   * 检查**所有**标签页的磁盘改动。
   *
   * 为什么不是只查当前标签页：用户切到别的窗口改了一堆文件再切回来，
   * 只查当前那个的话，其他标签页会在用户切过去的那一刻才突然报冲突 ——
   * 而那时他已经在心里认为「这个文件是干净的」了。逐个 stat 的成本对
   * 桌面端的标签页数量来说可以忽略。
   */
  async checkExternalChanges(): Promise<void> {
    for (const tab of this.tabs) {
      await tab.store.checkExternalChange();
    }
  }

  // ---- 文件树 -----------------------------------------------------------

  /** 打开文件夹作为工作区。传 `path` 则跳过对话框（拖入目录、测试用）。 */
  async openFolder(path?: string): Promise<boolean> {
    const root = path ?? (await this.provider.pickDirectoryPath());
    if (root === null || root === '') return false;

    this.tree = { ...EMPTY_TREE, root, expanded: [root] };
    this.state = { ...this.state, tree: this.tree };
    this.notify();

    await this.loadDir(root);
    return true;
  }

  closeFolder(): void {
    this.setTree({ root: null, dirs: {}, expanded: [], loading: [], error: null });
  }

  async loadDir(path: string): Promise<void> {
    if (this.tree.loading.includes(path)) return;
    this.setTree({ loading: [...this.tree.loading, path] });
    try {
      const entries = await this.provider.listDir(path);
      this.setTree({
        dirs: { ...this.tree.dirs, [path]: entries },
        loading: this.tree.loading.filter((item) => item !== path),
        error: null,
      });
    } catch (thrown) {
      this.setTree({
        loading: this.tree.loading.filter((item) => item !== path),
        error: {
          code: 'IO',
          message: `读不出目录 ${basename(path)}`,
          detail: thrown instanceof Error ? thrown.message : String(thrown),
        },
      });
    }
  }

  async toggleDir(path: string): Promise<void> {
    if (this.tree.expanded.includes(path)) {
      // 折叠时把子孙目录一并收起，否则重新展开时它们的子行会凭空出现
      const descendants = this.tree.expanded.filter((item) => item === path || item.startsWith(path));
      this.setTree({ expanded: this.tree.expanded.filter((item) => !descendants.includes(item)) });
      return;
    }
    this.setTree({ expanded: [...this.tree.expanded, path] });
    if (this.tree.dirs[path] === undefined) await this.loadDir(path);
  }

  /** 重新读取所有已加载的目录（外部新建 / 删除文件后）。 */
  async refreshTree(): Promise<void> {
    const paths = Object.keys(this.tree.dirs);
    if (paths.length === 0) return;
    const results = await Promise.all(
      paths.map(async (path) => {
        try {
          return [path, await this.provider.listDir(path)] as const;
        } catch {
          return [path, null] as const;
        }
      }),
    );
    const dirs: Record<string, readonly DirEntry[]> = {};
    for (const [path, entries] of results) {
      if (entries !== null) dirs[path] = entries;
    }
    this.setTree({ dirs, error: null });
  }

  setShowIgnored(show: boolean): void {
    this.setTree({ showIgnored: show });
  }

  /** 把某个路径的祖先目录链展开并加载，让它在树里可见。 */
  async revealInTree(path: string): Promise<void> {
    const root = this.tree.root;
    if (root === null) return;
    if (!path.startsWith(root)) return;

    const separator = root.includes('\\') ? '\\' : '/';
    const chain = ancestorsOf(root, path, separator);
    if (chain.length === 0) return;

    const expanded = [...new Set([...this.tree.expanded, ...chain])];
    this.setTree({ expanded });
    for (const dir of chain) {
      if (this.tree.dirs[dir] === undefined) await this.loadDir(dir);
    }
  }

  dispose(): void {
    for (const tab of this.tabs) {
      tab.unsubscribe();
      tab.store.dispose();
    }
    this.tabs = [];
    this.activeId = null;
    this.tree = EMPTY_TREE;
    // 快照也要一起清掉。只清 tabs 不清 state 的话，dispose 之后
    // `getSnapshot()` 会继续返回一份「指向已销毁子 store」的摘要 ——
    // 那是最难查的一类问题：外壳还在渲染一些已经不存在的标签页。
    this.state = { tabs: [], activeId: null, tree: EMPTY_TREE, openError: null };
    this.listeners.clear();
  }
}
