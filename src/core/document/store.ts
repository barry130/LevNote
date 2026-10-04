/**
 * 单文档模型（P1 只开一个文档；多标签在 P2 把它变成数组即可）。
 *
 * 核心不变量：
 * - **一份文件同时只有一个权威编辑实例**。外部程序改了它，我们不会一边报错一边写，
 *   而是停下来让用户选「重新加载 / 覆盖 / 另存为」。
 * - **脏判定只认内容与格式**，不看时间戳。`text !== savedText` 或行尾被改过，
 *   才算脏。否则「打开又什么都没改」也会被当成待保存，用户会不信任这个标记。
 * - 保存前把 `mtimeMs` 作为乐观锁传给 Rust；不符即失败，绝不静默覆盖。
 */

import { randomSalt, readImageAsUrl, writePastedImage } from './images.js';
import { dirname, joinPath } from '../path.js';
import {
  asAppError,
  type AppErrorShape,
  type BomKind,
  type DocumentProvider,
  type LineEndingName,
  type ProviderKind,
  type TextFile,
} from './types.js';

export type DocumentStatus = 'clean' | 'dirty' | 'saving' | 'conflict' | 'error';

export interface OpenDocument {
  path: string;
  displayName: string;
  /**
   * 这次「打开」的会话编号。**只有真正换文件才会变**。
   *
   * 编辑器组件用它当 key：换文件要重建内核（撤销栈、光标、折叠状态都该重置），
   * 而「另存为」换了路径却还是同一份内容，不该重建 —— 否则用户刚改完按 Ctrl+Shift+S，
   * 光标会跳回文档开头。`reload()` 也不变，走 `setValue` 原地替换。
   */
  readonly session: number;
  /** 编辑器里的当前文本，权威副本 */
  text: string;
  /** 上次成功保存（或刚打开）时的文本 */
  savedText: string;
  encoding: string;
  bom: BomKind;
  /** 当前文档的行尾风格；`mixed` 表示混排 */
  lineEnding: LineEndingName;
  savedLineEnding: LineEndingName;
  /** 磁盘上的修改时间，保存时作为乐观锁 */
  mtimeMs: number;
  readonly: boolean;
  /** 打开时解码出现非法字节，编码可能是猜错的 */
  lossy: boolean;
}

export interface ConflictInfo {
  message: string;
  detectedAt: number;
}

/**
 * 视图模式。
 *
 * - `rich`：渲染后的视图（Markdown 内核）。新打开的文档默认在这里。
 * - `source`：源码视图（由适配器的 `sourceView` 内核渲染）。
 * - `split`：**源码与预览并排**。左栏是源码（可编辑，就是 `source` 那一个内核），
 *   右栏是同一个文档的只读渲染（首选内核 + `readOnly`）。见 ADR 0022。
 *
 * `split` 刻意**不是**一个独立的开关位（比如 `split: boolean`）：那样会多出
 * 「rich + split 同时为真」这种不存在的状态要回答，而三档是**互斥**的 ——
 * 用户在任何时刻只看其中一种排布。互斥的取值就该是一个枚举。
 *
 * ## 为什么它**不放在 `OpenDocument` 里**
 *
 * `OpenDocument` 描述的是**文件**（路径、编码、行尾、mtime），
 * 而视图模式描述的是**这次编辑会话**。混在一起有两个具体后果：
 * 一是 `reload()` 会走 `toDocument()` 重建文档对象，稍不注意就把视图模式
 * 一起重置回富文本；二是 `isDirty` 的语义会被迫回答「视图模式算不算脏」，
 * 而这个问题本身就不该被提出。
 *
 * 放进 `StoreState` 之后，它天然是**每标签页**的（一个标签页一个 store），
 * 且 `reload` / `saveAs` 这类只碰文档对象的操作碰不到它。
 */
export type ViewMode = 'rich' | 'source' | 'split';

export interface StoreState {
  doc: OpenDocument | null;
  status: DocumentStatus;
  /** 最近一次失败，供 UI 弹提示；处理后应清空 */
  error: AppErrorShape | null;
  /** 非空表示有未决冲突，UI 必须让用户选一个出路 */
  conflict: ConflictInfo | null;
  providerKind: ProviderKind;
  /** 有待进行的磁盘操作 */
  busy: boolean;
  /** 状态栏的一句话反馈 */
  notice: string;
  /** 当前看的是渲染后的视图还是源码。见 `ViewMode`。 */
  viewMode: ViewMode;
}

export function isDirty(doc: OpenDocument): boolean {
  return doc.text !== doc.savedText || doc.lineEnding !== doc.savedLineEnding;
}

/**
 * 「没有文档」时的状态快照。
 *
 * 导出它是为了让那些「store 可能为 null」的 UI（大纲面板）有一份**同源**的兜底，
 * 而不是自己手抄一份字段表。手抄的那份在给 `StoreState` 加字段时不会报错，
 * 只会在类型层面悄悄不同步 —— 实测：加 `viewMode` 时正是 `OutlinePanel` 里
 * 那份手抄的常量先炸，说明「不同源」这件事是真会发生的，不只是理论问题。
 */
export function emptyStoreState(providerKind: ProviderKind = 'desktop'): StoreState {
  return {
    doc: null,
    status: 'clean',
    error: null,
    conflict: null,
    providerKind,
    busy: false,
    notice: '',
    viewMode: 'rich',
  };
}

export function basename(path: string): string {
  const parts = path.split(/[\\/]/);
  return parts[parts.length - 1] || path;
}

function toDocument(file: TextFile, session: number): OpenDocument {
  return {
    path: file.path,
    displayName: basename(file.path),
    session,
    text: file.text,
    savedText: file.text,
    encoding: file.encoding,
    bom: file.bom,
    lineEnding: file.lineEnding,
    savedLineEnding: file.lineEnding,
    mtimeMs: file.mtimeMs,
    readonly: file.readonly,
    lossy: file.lossy,
  };
}

function computeStatus(state: Omit<StoreState, 'status'>): DocumentStatus {
  if (state.conflict) return 'conflict';
  if (state.error && !state.doc) return 'error';
  if (state.busy) return 'saving';
  if (!state.doc) return 'clean';
  return isDirty(state.doc) ? 'dirty' : 'clean';
}

export type ConflictChoice = 'reload' | 'overwrite' | 'save-copy';

export class DocumentStore {
  private readonly provider: DocumentProvider;
  private listeners = new Set<() => void>();
  private state: StoreState;
  /** 每次 `open()` 自增，用于区分「换文件」与「同一文件被重新读取」。 */
  private sessionCounter = 0;

  constructor(provider: DocumentProvider) {
    this.provider = provider;
    this.state = emptyStoreState(provider.kind);
  }

  // React 的 useSyncExternalStore 需要 getSnapshot 返回稳定引用：
  // 只在真正变化时才替换对象，避免无限重渲染。
  getSnapshot = (): StoreState => this.state;

  subscribe = (listener: () => void): (() => void) => {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  };

  private commit(patch: Partial<Omit<StoreState, 'status'>>): void {
    const merged = { ...this.state, ...patch };
    this.state = { ...merged, status: computeStatus(merged) };
    for (const listener of this.listeners) listener();
  }

  clearError = (): void => this.commit({ error: null });
  clearNotice = (): void => this.commit({ notice: '' });

  /**
   * 把文档里的图片 `src` 解析成能直接放进 `<img>` 的 URL。拿不到时返回 `null`。
   *
   * ## 为什么是箭头属性而不是方法
   *
   * 它会被**拆下来单传**（`DocumentPane` → `MarkdownEditor` → 内核的 NodeView），
   * 内核拿到的是函数本身。原型上的方法被拆下来之后 `this` 就丢了，
   * 一调用就抛 `Cannot read properties of undefined`。箭头属性把它绑在实例上，
   * 拆到哪儿都能用 —— 与 `getSnapshot` / `subscribe` 是同一条理由。
   *
   * ## 为什么**不缓存**
   *
   * 缓存看起来划算（切一次视图模式就要把整篇的图重读一遍），但缓存的对象是
   * `data:` URI —— 一张 1MB 的图会变成约 1.4MB 的字符串，而它是**常驻**的。
   * 一份有三十张截图的文档就能吃掉几十 MB，且没有任何上界。
   *
   * 不缓存的代价是有限的：NodeView 只在**节点创建时**解析一次，
   * 重渲染不会重读（`update()` 只在 `src` 变了才重新加载）。
   * 真正会重读的只有「切换视图模式」，而那是用户主动做的一次动作。
   *
   * **重审条件**：如果实测切视图出现可感知的卡顿，再加一个有上界的缓存
   * （按字节总量而不是条目数限制），而不是先加一个没有上界的。
   */
  resolveImageUrl = async (src: string): Promise<string | null> => {
    const doc = this.state.doc;
    if (!doc) return null;
    return readImageAsUrl(this.provider, doc.path, src);
  };

  /**
   * 把 `doc-block` 引用的 `src`（相对当前文档目录）读成**图表文件原文**。
   *
   * 与 `resolveImageUrl` 同一条链路的另一半（P5 第 25 项）：路径归并走
   * `joinPath`（分隔符统一成文档路径的风格，硬约束 12），读取走
   * `readTextFile` —— 与打开文档同一个解码器，`.drawio` 是 XML 文本。
   *
   * **不缓存**（与 `resolveImageUrl` 同一条理由）且**不抛**（调用链尽头是
   * NodeView 的一次渲染，未捕获的拒绝只会变成一个没人处理的 Promise）；
   * 读不到返回 `null`，内核据此画带路径的说明 —— 失败要说出来，不能留白。
   */
  resolveDocBlock = async (src: string): Promise<string | null> => {
    const doc = this.state.doc;
    if (!doc) return null;
    const absolute = joinPath(dirname(doc.path), src);
    if (absolute === null) return null;
    try {
      const file = await this.provider.readTextFile(absolute);
      return file.text;
    } catch {
      return null;
    }
  };

  /**
   * 粘贴进来的一张图片：落盘，返回**写进文档的相对路径**；失败返回 `null`。
   *
   * 失败原因写进 `notice`（状态栏）而不是 `error`（错误横幅）：
   * 「这份文档还没保存过」「这个格式不认识」都是**正常的用户操作结果**，
   * 不是程序异常。用错误横幅会让人以为软件坏了，而状态栏那句话
   * 恰好能在他下一次动作之前被读到。
   *
   * 无论成败都**不抛**：调用链尽头是内核的一次异步插入，
   * 抛出去只会变成一个没人处理的 Promise。
   */
  pasteImage = async (bytes: ArrayBuffer, mimeType: string): Promise<string | null> => {
    const doc = this.state.doc;
    if (!doc) return null;

    const result = await writePastedImage(this.provider, doc.path, {
      bytes,
      mimeType,
      at: new Date(),
      salt: randomSalt,
    });

    if (!result.ok) {
      this.commit({ notice: result.reason });
      return null;
    }

    // 说清楚「图片已经在磁盘上了，但文档还没保存」—— 这是用户真正会担心的事
    this.commit({ notice: `图片已存到 ${basename(result.path)}（文档还需保存）` });
    return result.src;
  };

  /** 打开一个路径。失败时把错误放进 state，由 UI 决定怎么呈现。 */
  async open(path: string): Promise<boolean> {
    if (this.state.doc && isDirty(this.state.doc)) {
      // 调用方（UI）负责先问用户是否丢弃改动；这里只做兜底
      this.commit({ error: { code: 'INTERNAL', message: '当前文档有未保存的改动，请先处理', detail: null } });
      return false;
    }

    this.commit({ busy: true, error: null, conflict: null, viewMode: 'rich' });
    try {
      const file = await this.provider.readTextFile(path);
      const doc = toDocument(file, (this.sessionCounter += 1));
      this.commit({
        doc,
        busy: false,
        notice: doc.lossy ? `已打开 ${doc.displayName}（编码可能是猜错的，建议手动确认）` : `已打开 ${doc.displayName}`,
      });
      return true;
    } catch (thrown) {
      this.commit({ busy: false, error: asAppError(thrown) });
      return false;
    }
  }

  /** 通过系统对话框打开。用户取消时安静返回 false。 */
  async openViaDialog(): Promise<boolean> {
    const path = await this.provider.pickOpenPath();
    if (path === null) return false;
    return this.open(path);
  }

  updateText(text: string): void {
    const doc = this.state.doc;
    if (!doc || doc.text === text) return;
    this.commit({ doc: { ...doc, text } });
  }

  /**
   * 切换视图模式（富文本 / 源码）。
   *
   * 刻意**不做任何「这个内核支持吗」的判断**：那是外壳的事，而且外壳能从
   * `EditorAdapter.sourceView` 得到权威答案。store 只管记住用户选了哪一种 ——
   * 让它去猜内核能力，等于把内核知识漏进了文档模型。
   *
   * 没有文档时静默忽略：切视图是纯视图动作，不该在空标签页上报错。
   */
  setViewMode(viewMode: ViewMode): void {
    if (!this.state.doc) return;
    if (this.state.viewMode === viewMode) return;
    this.commit({ viewMode });
  }

  setEncoding(encoding: string): void {
    const doc = this.state.doc;
    if (!doc) return;
    // 编码变更会让待写字节变化，但不改内存文本，所以不算「内容脏」，
    // 由「是否与最近一次保存的编码一致」另行体现。
    this.commit({ doc: { ...doc, encoding } });
  }

  /**
   * 同时设置编码与 BOM。
   * UI 上「UTF-8」与「UTF-8（含 BOM）」是两个选项，实际影响两个字段，
   * 所以必须一起改 —— 分开设会出现「GBK + UTF-8 BOM」这种不可能的组合。
   */
  setFormat(encoding: string, bom: BomKind): void {
    const doc = this.state.doc;
    if (!doc) return;
    this.commit({ doc: { ...doc, encoding, bom } });
  }

  setLineEnding(lineEnding: LineEndingName): void {
    const doc = this.state.doc;
    if (!doc) return;
    this.commit({ doc: { ...doc, lineEnding } });
  }

  /**
   * 保存。返回是否成功。
   * 冲突时不抛错而是把状态切到 `conflict`，因为这是**需要用户决策**的正常分支，
   * 不是程序异常。
   */
  async save(): Promise<boolean> {
    const doc = this.state.doc;
    if (!doc) return false;

    if (doc.readonly) {
      this.commit({
        error: { code: 'PERMISSION_DENIED', message: `${doc.displayName} 是只读文件，请用「另存为」`, detail: null },
      });
      return false;
    }

    this.commit({ busy: true, error: null });
    try {
      const outcome = await this.provider.writeTextFile({
        path: doc.path,
        text: doc.text,
        encoding: doc.encoding,
        bom: doc.bom,
        // mixed 不传目标行尾：混排是文件既有事实，不该由一次保存擅自统一掉
        lineEnding: doc.lineEnding === 'mixed' ? undefined : (doc.lineEnding),
        expectedMtimeMs: doc.mtimeMs,
        create: true,
      });

      this.commit({
        busy: false,
        conflict: null,
        notice: `已保存 ${doc.displayName}`,
        doc: {
          ...doc,
          savedText: doc.text,
          savedLineEnding: doc.lineEnding,
          mtimeMs: outcome.mtimeMs,
          encoding: outcome.encoding,
          bom: outcome.bom,
        },
      });
      return true;
    } catch (thrown) {
      const error = asAppError(thrown);
      if (error.code === 'CONFLICT') {
        this.commit({ busy: false, conflict: { message: error.message, detectedAt: Date.now() } });
      } else {
        this.commit({ busy: false, error });
      }
      return false;
    }
  }

  /** 另存为。目标由对话框给出，因此不做 mtime 冲突检测（原生对话框已确认过覆盖）。 */
  async saveAs(): Promise<boolean> {
    const doc = this.state.doc;
    if (!doc) return false;

    const target = await this.provider.pickSavePath(doc.displayName);
    if (target === null) return false;

    this.commit({ busy: true, error: null });
    try {
      const outcome = await this.provider.writeTextFile({
        path: target,
        text: doc.text,
        encoding: doc.encoding,
        bom: doc.bom,
        lineEnding: doc.lineEnding === 'mixed' ? undefined : (doc.lineEnding),
        create: true,
      });

      const next: OpenDocument = {
        ...doc,
        path: outcome.path,
        displayName: basename(outcome.path),
        savedText: doc.text,
        savedLineEnding: doc.lineEnding,
        mtimeMs: outcome.mtimeMs,
        encoding: outcome.encoding,
        bom: outcome.bom,
        readonly: false,
      };
      this.commit({ doc: next, busy: false, conflict: null, notice: `已另存为 ${next.displayName}` });
      return true;
    } catch (thrown) {
      this.commit({ busy: false, error: asAppError(thrown) });
      return false;
    }
  }

  /** 丢弃内存改动、从磁盘重新读取。 */
  async reload(): Promise<boolean> {
    const doc = this.state.doc;
    if (!doc) return false;

    this.commit({ busy: true, error: null });
    try {
      const file = await this.provider.readTextFile(doc.path, doc.encoding);
      // 沿用同一个 session：这是同一份文档被重新读取，编辑器实例不该重建
      this.commit({ doc: toDocument(file, doc.session), busy: false, conflict: null, notice: `已从磁盘重新加载 ${doc.displayName}` });
      return true;
    } catch (thrown) {
      this.commit({ busy: false, error: asAppError(thrown) });
      return false;
    }
  }

  async resolveConflict(choice: ConflictChoice): Promise<boolean> {
    switch (choice) {
      case 'reload':
        return this.reload();
      case 'overwrite': {
        const doc = this.state.doc;
        if (!doc) return false;
        this.commit({ busy: true, error: null, conflict: null });
        try {
          const outcome = await this.provider.writeTextFile({
            path: doc.path,
            text: doc.text,
            encoding: doc.encoding,
            bom: doc.bom,
            lineEnding: doc.lineEnding === 'mixed' ? undefined : (doc.lineEnding),
            create: true,
            // 明确不传 expectedMtimeMs：用户已确认要用当前内容覆盖磁盘
          });
          this.commit({
            busy: false,
            notice: `已覆盖磁盘上的 ${doc.displayName}`,
            doc: { ...doc, savedText: doc.text, savedLineEnding: doc.lineEnding, mtimeMs: outcome.mtimeMs },
          });
          return true;
        } catch (thrown) {
          this.commit({ busy: false, error: asAppError(thrown) });
          return false;
        }
      }
      case 'save-copy': {
        const ok = await this.saveAs();
        if (ok) this.commit({ conflict: null });
        return ok;
      }
    }
  }

  /**
   * 检查磁盘是否被外部改动。
   *
   * P1 没有文件监听（P3 补），改用「窗口重新获得焦点时比对 mtime」。
   * 这条路径与将来的监听回调共用同一套判定，不会白写。
   */
  async checkExternalChange(): Promise<void> {
    const doc = this.state.doc;
    if (!doc || this.state.busy) return;

    let stat;
    try {
      stat = await this.provider.statFile(doc.path);
    } catch {
      // 文件被删或被移走：交给用户下一步动作，不主动弹错
      return;
    }
    if (stat.mtimeMs === doc.mtimeMs) return;

    if (isDirty(doc)) {
      this.commit({ conflict: { message: '磁盘上的文件已被其他程序修改，而你本地也有改动', detectedAt: Date.now() } });
    } else {
      // 本地没有改动，直接跟上磁盘，不打扰用户
      await this.reload();
    }
  }

  /** 关闭。`force` 为 false 且有脏改动时返回 false，由 UI 先确认。 */
  close(force = true): boolean {
    const doc = this.state.doc;
    if (!doc) return true;
    if (!force && isDirty(doc)) return false;

    this.commit({ doc: null, conflict: null, error: null, busy: false, notice: '' });
    return true;
  }

  dispose(): void {
    this.listeners.clear();
  }
}
