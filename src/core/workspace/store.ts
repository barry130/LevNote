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

import type { AppErrorShape, DirEntry, DocumentProvider } from '../document/types.js';
import { asAppError } from '../document/types.js';
import { DocumentStore, isDirty, type DocumentStatus } from '../document/store.js';
import { basename, dirname, joinPath, validateEntryName } from '../path.js';
import { ancestorsOf } from './tree.js';
import { UNSUPPORTED_FILE_MANAGER, type WorkspaceFileManager } from './fsmanage.js';

export interface TabSummary {
  readonly id: number;
  readonly path: string;
  readonly displayName: string;
  readonly dirty: boolean;
  readonly status: DocumentStatus;
  readonly readonly: boolean;
  /**
   * 这个标签页是从哪个标签页「就地编辑」钻过来的（P6 第 28 项 L2，ADR 0036）。
   * null = 不是钻取打开的。只用于导航（标签栏的返回 chip），不进会话恢复 ——
   * 标签 id 每次启动重新编号，跨会话保留只会指错。
   */
  readonly drilledFrom: number | null;
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
  readonly error: AppErrorShape | null;
}

export interface WorkspaceState {
  readonly tabs: readonly TabSummary[];
  readonly activeId: number | null;
  readonly tree: TreeState;
  /** 最近一次打开失败。UI 弹一次之后应调 `clearOpenError`。 */
  readonly openError: AppErrorShape | null;
}

/**
 * 会话恢复的一条输入。
 *
 * 刻意**只有路径与草稿**，没有 `at` —— 那是存储层用来决定淘汰顺序的，
 * 工作区不关心它。也刻意没有编码与行尾：那两项在 `open()` 读磁盘时重新探测，
 * 存一份等于造出第二个真相来源（见 `core/session.ts` 的已知限制）。
 */
export interface RestoreEntry {
  readonly path: string;
  /** 未保存的草稿。`null` = 从磁盘读。 */
  readonly draft: string | null;
}

export interface RestoreResult {
  /** 成功打开的路径数（含本来就已经开着的）。 */
  readonly restored: number;
  /**
   * 打不开的路径（文件被删 / 被移走 / 没权限）。
   *
   * **刻意不写进 `openError`**：那个字段是「一个错误横幅」的单槽位，
   * 而启动恢复可能一次失败好几个 —— 逐条塞进去的结果是用户只看到最后一条，
   * 前面几个文件**悄无声息地不见了**。所以这里把整份清单交回给外壳，
   * 由它用一条横幅把话说完（硬约束 2）。
   */
  readonly failed: readonly string[];
}

interface Tab {
  readonly id: number;
  readonly store: DocumentStore;
  readonly unsubscribe: () => void;
  /** 见 `TabSummary.drilledFrom`。不可变字段，更新走整对象替换。 */
  readonly drilledFrom: number | null;
}

const EMPTY_TREE: TreeState = {
  root: null,
  dirs: {},
  expanded: [],
  loading: [],
  showIgnored: false,
  error: null,
};

function summarize(tab: Tab): TabSummary | null {
  const { doc, status } = tab.store.getSnapshot();
  if (!doc) return null;
  return {
    id: tab.id,
    path: doc.path,
    displayName: doc.displayName,
    dirty: status === 'dirty',
    status,
    readonly: doc.readonly,
    drilledFrom: tab.drilledFrom,
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
      x.readonly !== y.readonly ||
      x.drilledFrom !== y.drilledFrom
    ) {
      return false;
    }
  }
  return true;
}

export class WorkspaceStore {
  private readonly provider: DocumentProvider;
  private readonly fileManager: WorkspaceFileManager;
  private tabs: Tab[] = [];
  private activeId: number | null = null;
  private nextId = 1;
  private listeners = new Set<() => void>();
  private tree: TreeState = EMPTY_TREE;
  private state: WorkspaceState;

  constructor(provider: DocumentProvider, fileManager: WorkspaceFileManager = UNSUPPORTED_FILE_MANAGER) {
    this.provider = provider;
    this.fileManager = fileManager;
    this.state = { tabs: [], activeId: null, tree: EMPTY_TREE, openError: null };
  }

  /** 浏览器预览为 false：外壳据此把侧栏的文件管理按钮画成禁用。 */
  get canManageFiles(): boolean {
    return this.fileManager.available;
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
      const summary = summarize(tab);
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
      // 普通打开命中已有标签页时，清掉旧的「来路」：用户显式打开这个文件，
      // 意味着之前从引用卡片钻过来的那次导航已经作废（chip 再挂着就是错的）
      if (existing.drilledFrom !== null) {
        this.tabs = this.tabs.map((tab) => (tab.id === existing.id ? { ...tab, drilledFrom: null } : tab));
      }
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
    this.tabs = [...this.tabs, { id, store, unsubscribe, drilledFrom: null }];
    this.activeId = id;
    this.syncTabs();

    // 打开文件时把它所在的目录链展开，否则用户看到的是一个和侧栏无关的文档
    if (this.tree.root !== null) await this.revealInTree(path);
    return true;
  }

  /**
   * 「就地编辑」（P6 第 28 项 L2，ADR 0036）：从 `fromId` 那个标签页出发，
   * 打开 `path` 并把来路记在目标标签页上（标签栏出现「↩ 返回」chip）。
   *
   * 编辑走的完全是正常文档路径 —— 这里只多做两件事：记来路、切过去。
   * 先走普通 `open()`（去重、激活、文件树展开都是它的），**再**把来路记上去，
   * 顺序不能反：`open()` 的去重路径会清 `drilledFrom`。
   */
  async openInPlace(fromId: number, path: string): Promise<boolean> {
    const from = this.tabOf(fromId);
    if (from === null) return this.open(path);

    const ok = await this.open(path);
    if (!ok) return false;

    const target = this.tabs.find((tab) => tab.store.getSnapshot().doc?.path === path);
    // 文档引用自己（path 与来源相同）时不记来路 —— 自己↩自己没有意义
    if (target === undefined || target.id === fromId) return true;

    this.tabs = this.tabs.map((tab) => (tab.id === target.id ? { ...tab, drilledFrom: fromId } : tab));
    this.syncTabs();
    return true;
  }

  /**
   * 沿「就地编辑」的来路返回。返回即清来路（chip 消失；想再过去，
   * 标签页还在标签栏里）。来路标签页已被关掉时把悬空来路清掉就好 ——
   * `closeTab` 会顺手清，这里是兜底。
   */
  goBackDrill(id: number): void {
    const tab = this.tabOf(id);
    if (tab === null || tab.drilledFrom === null) return;
    const from = tab.drilledFrom;
    this.tabs = this.tabs.map((item) => (item.id === id ? { ...item, drilledFrom: null } : item));
    if (this.tabOf(from) !== null) this.activate(from);
    else this.syncTabs();
  }

  async openViaDialog(): Promise<boolean> {
    const path = await this.provider.pickOpenPath();
    if (path === null) return false;
    return this.open(path);
  }

  /**
   * 会话恢复：一次性把上次开着的标签页重新打开。
   *
   * ## 为什么不循环调 `open()`
   *
   * `open()` 每打开一个就把那个标签页**激活**一次。循环调它意味着启动时
   * 有 N 次 `activeId` 变化、N 次外壳重渲染、N 次编辑器内核创建 ——
   * 而每次创建内核的代价是实打实的（实测：2 万字文档 700ms 上下）。
   * 十个标签页的会话会变成「启动要等好几秒，中间还闪十下」。
   *
   * 这里改成**先全部塞进去、最后一次性通知**：整个恢复过程只有一次渲染，
   * 也就只有一个内核被创建（用户最后看到的那个标签页）。
   *
   * ## 草稿为什么在这里就写进 store
   *
   * 因为**面板是照着 `doc.text` 挂载的**（`DocumentPane` 把 `doc.text` 交给
   * `initialValue`）。在标签页被激活之前把草稿写进去，内核一出生拿到的就是草稿；
   * 反过来（先激活、再 `updateText`）会让内核先用磁盘内容建一遍、再被整篇替换一次 ——
   * 白花一次创建，中间还会闪一下磁盘版本。
   *
   * 写草稿走的是 `updateText`，它只改 `text`、不动 `savedText`，
   * 所以这个标签页**天然是脏的**：用户看到的是「有未保存的改动」，
   * 按 Ctrl+S 就会把草稿写回磁盘，与崩溃前完全是同一个状态。
   *
   * ## 重复调用是安全的
   *
   * 已经开着的路径会被跳过（与 `open()` 的去重同一条理由）。这让调用方
   * 不必为「effect 会不会跑两遍」操心 —— React 的 StrictMode 在开发模式下
   * 确实会把启动 effect 跑两遍。
   */
  async restoreTabs(entries: readonly RestoreEntry[], activePath: string | null): Promise<RestoreResult> {
    const failed: string[] = [];
    let restored = 0;

    for (const entry of entries) {
      if (this.tabs.some((tab) => tab.store.getSnapshot().doc?.path === entry.path)) {
        restored += 1;
        continue;
      }

      const store = new DocumentStore(this.provider);
      const ok = await store.open(entry.path);
      if (!ok) {
        store.dispose();
        failed.push(entry.path);
        continue;
      }

      if (entry.draft !== null) {
        store.updateText(entry.draft);
        // `open()` 留了一句「已打开 X」。那是**磁盘读取**的结果，
        // 而屏幕上现在是草稿 —— 留着它等于让状态栏说一句半真的话。
        // 清掉之后状态栏由「有未保存的改动」那枚状态标签说话，那是准的。
        store.clearNotice();
      }

      const id = this.nextId;
      this.nextId += 1;
      // 订阅放在 `updateText` **之后**：否则上面那次写入会立刻触发一次
      // `syncTabs()`，而那时这个标签页还没进 `this.tabs`，通知的是半成品状态
      const unsubscribe = store.subscribe(() => this.syncTabs());
      this.tabs = [...this.tabs, { id, store, unsubscribe, drilledFrom: null }];
      restored += 1;
    }

    const target = activePath === null ? undefined : this.tabs.find((tab) => tab.store.getSnapshot().doc?.path === activePath);
    // 上次激活的那个文件打不开了（被删 / 被移走）时落到第一个 ——
    // 宁可打开一个别的标签页，也不要留一个「什么都没打开」的空窗口
    const chosen = target ?? this.tabs[0];
    if (chosen) this.activeId = chosen.id;

    // 整个恢复过程**只通知这一次**
    this.syncTabs();

    // 把当前标签页在文件树里展开。只展开它一个：十个标签页各展开一次
    // 意味着十倍的目录读取，而用户真正需要看到的是「我正在看的这份在哪」。
    const chosenPath = chosen?.store.getSnapshot().doc?.path;
    if (chosenPath !== undefined && this.tree.root !== null) await this.revealInTree(chosenPath);

    return { restored, failed };
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

    // 关掉的标签页可能是别人的「来路」—— `goBackDrill` 对悬空来路有兜底，
    // 但 chip 显示「↩ 一个不存在的标签页」本身就是错的，在这里清干净
    if (this.tabs.some((item) => item.drilledFrom === id)) {
      this.tabs = this.tabs.map((item) => (item.drilledFrom === id ? { ...item, drilledFrom: null } : item));
    }

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

  /**
   * 保存**所有**变脏的标签页（菜单的「保存全部」）。
   *
   * 逐个 await 而不是并发：每次保存都可能弹冲突对话框（磁盘文件被外部改过），
   * 并发的话两个对话框会叠在一起，用户根本答不完。干净的标签页直接跳过，
   * 只读文档由 DocumentStore.save 自己拒绝。
   *
   * 标签页的脏标记不用手动同步 —— 每份 store 的订阅回调就是 `syncTabs`。
   */
  async saveAll(): Promise<boolean> {
    let allSaved = true;
    for (const tab of this.tabs) {
      const snapshot = tab.store.getSnapshot();
      if (snapshot.doc === null || !isDirty(snapshot.doc)) continue;
      if (!(await tab.store.save())) allSaved = false;
    }
    return allSaved;
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

  /**
   * 文件监听推来的一批变更路径（P3 第 14 项，ADR 0029）。
   *
   * 只做**点名**：打开着的文档里只有路径命中事件的那几份需要重查 mtime
   * （弹不弹冲突仍由 `checkExternalChange` 的 mtime 比对裁决 —— 事件是
   * 通知不是判决，丢了、重了都不会造成错误的冲突提示）；目录树只重拉
   * 「已经加载过的」受影响目录，没展开的层级不在界面上，重拉纯属浪费。
   *
   * 窗口焦点时的全量比对（`checkExternalChanges`）保留作兜底 ——
   * 事件系统可能丢，焦点比对是那个「迟到最后但一定会到」的检查。
   */
  handleFsChanges(paths: readonly string[]): void {
    if (paths.length === 0) return;
    const changed = new Set(paths);

    for (const tab of this.tabs) {
      const path = tab.store.getSnapshot().doc?.path;
      if (path !== undefined && changed.has(path)) void tab.store.checkExternalChange();
    }

    const dirs = new Set<string>();
    for (const path of paths) {
      const parent = dirname(path);
      if (parent !== null && this.tree.dirs[parent] !== undefined) dirs.add(parent);
      // 变更路径本身就是一个已加载的目录（被删 / 改名 / 内容变化）：
      // 重拉它自己的那一层，让树把消失的子行摘掉
      if (this.tree.dirs[path] !== undefined) dirs.add(path);
    }
    for (const dir of dirs) void this.loadDir(dir);
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

  // ---- 文件管理（A2） -----------------------------------------------------
  //
  // 三条编排的共同点：**磁盘与打开着的标签页是同一份文件的两个视图**，
  // 只改一边就会让它们说不一样的话 —— 所以动磁盘之前先看清标签页这边。

  /** 命中某个路径（或其子树内）的打开标签页。两种分隔符都认。 */
  private tabsUnder(path: string): Tab[] {
    return this.tabs.filter((tab) => {
      const docPath = tab.store.getSnapshot().doc?.path;
      if (docPath === undefined) return false;
      return docPath === path || docPath.startsWith(`${path}\\`) || docPath.startsWith(`${path}/`);
    });
  }

  /** 文件管理动作失败的通报位。侧栏的错误槽位就在树上方，用户一定看得到。 */
  private failTreeAction(message: string, detail?: string): void {
    this.setTree({ error: { code: 'IO', message, detail: detail ?? null } });
  }

  private requireFileManager(): WorkspaceFileManager | null {
    if (!this.fileManager.available) {
      this.failTreeAction('当前环境不支持管理文件（浏览器预览没有真实文件系统，请用桌面版）');
      return null;
    }
    return this.fileManager;
  }

  /**
   * 在目录里新建文件 / 文件夹。成功返回新路径（文件还会被顺手打开），
   * 失败返回 null，原因在 `tree.error`。
   *
   * 新建的文件**要打开**：用户输入名字的下一秒就该看到它，而不是自己去树里找。
   * 打开失败不影响「文件已建成」这个事实 —— `open()` 自己会把错误放进
   * `openError` 横幅，这里原样返回路径即可。
   */
  async createEntry(dir: string, rawName: string, kind: 'file' | 'dir'): Promise<string | null> {
    const fileManager = this.requireFileManager();
    if (fileManager === null) return null;

    const name = rawName.trim();
    const problem = validateEntryName(name);
    if (problem !== null) {
      this.failTreeAction(problem);
      return null;
    }
    const target = joinPath(dir, name);
    if (target === null) {
      this.failTreeAction('无法确定目标路径');
      return null;
    }

    try {
      if (kind === 'file') await fileManager.createFile(target);
      else await fileManager.createDirectory(target);
    } catch (thrown) {
      const error = asAppError(thrown);
      this.failTreeAction(`新建失败：${error.message}`, error.detail ?? undefined);
      return null;
    }

    await this.loadDir(dir);
    if (kind === 'file') await this.open(target);
    return target;
  }

  /**
   * 重命名一个文件或目录。成功返回新路径，失败返回 null（原因在 `tree.error`）。
   *
   * 打开着的标签页**跟着换路径**（`DocumentStore.retargetPath`）而不是关掉重开：
   * 改名不碰内容，脏缓冲、撤销栈、光标原样保住 —— 这是对「脏标签页」最诚实的
   * 处理，改名对它们是安全的。目录改名时，子树内每个标签页的前缀一起替换。
   *
   * 树刷新走 `refreshTree()` 而不是只拉父目录：目录改名后它**自己的**子项列表
   * （键还是旧路径）也过期了，逐个算「哪些目录受影响」不如把已加载的目录
   * 全量重拉 —— 数量以十计，成本可忽略。
   */
  async renameEntry(oldPath: string, rawName: string): Promise<string | null> {
    const fileManager = this.requireFileManager();
    if (fileManager === null) return null;

    const name = rawName.trim();
    const problem = validateEntryName(name);
    if (problem !== null) {
      this.failTreeAction(problem);
      return null;
    }
    const parent = dirname(oldPath);
    if (parent === null) {
      this.failTreeAction(`「${basename(oldPath)}」没有所在目录信息，无法重命名`);
      return null;
    }
    const target = joinPath(parent, name);
    if (target === null) {
      this.failTreeAction('无法确定目标路径');
      return null;
    }
    if (target === oldPath) return target;

    try {
      await fileManager.renamePath(oldPath, target);
    } catch (thrown) {
      const error = asAppError(thrown);
      this.failTreeAction(`重命名失败：${error.message}`, error.detail ?? undefined);
      return null;
    }

    for (const tab of this.tabsUnder(oldPath)) {
      const docPath = tab.store.getSnapshot().doc?.path;
      if (docPath === undefined) continue;
      // 文件改名：docPath === oldPath，slice 得空串；目录改名：slice 得分隔符 + 余下路径
      tab.store.retargetPath(target + docPath.slice(oldPath.length));
    }
    await this.refreshTree();
    return target;
  }

  /**
   * 删除一个文件或目录（目录递归）。**不可撤销** —— 调用方（外壳）必须已经
   * 向用户确认过；这里守住的是另一条线：**打开着的脏标签页**。
   *
   * 磁盘删掉了而标签页还在的话，用户按 Ctrl+S 会把内容写回一个**新文件** ——
   * 「删除」就变成了「没删干净」。所以删除时要关掉受影响的标签页（干净的
   * 关了不丢东西）；而有未保存改动的标签页**不能替用户做主**：明确拒绝，
   * 让他先表态（保存或放弃），这与「绝不静默丢内容」是同一句话。
   */
  async deleteEntry(path: string): Promise<boolean> {
    const fileManager = this.requireFileManager();
    if (fileManager === null) return false;

    const affected = this.tabsUnder(path);
    const dirty = affected.find((tab) => tab.store.getSnapshot().status === 'dirty');
    if (dirty !== undefined) {
      const name = dirty.store.getSnapshot().doc?.displayName ?? basename(path);
      this.failTreeAction(`「${name}」有未保存的改动，请先保存或关闭对应标签页，再删除`);
      return false;
    }

    try {
      await fileManager.deletePath(path);
    } catch (thrown) {
      const error = asAppError(thrown);
      this.failTreeAction(`删除失败：${error.message}`, error.detail ?? undefined);
      return false;
    }

    for (const tab of affected) this.closeTab(tab.id, true);
    await this.refreshTree();
    return true;
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
