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
import { t, type MessageKey } from '../i18n/index.js';
import { resolveRelative } from '../path.js';
import {
  asAppError,
  type AppErrorShape,
  type BomKind,
  type ErrorCode,
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

/**
 * 状态栏那句话（P3 第 17 项 · 快照型文案第二批，#TASK-I18N-SNAPSHOT-01）。
 *
 * ## 为什么存 key 而不是存已经查好表的字符串
 *
 * `t()` 的结果是**那一刻**的译文。把它存进 state 就成了快照：用户在设置页
 * 把语言切成 English，屏幕上那条「已打开 a.md」不会跟着变 —— 而它此刻明明
 * 还挂在状态栏上。留一个**key + 参数**在 state 里、到**渲染时**才查表，
 * 这一条 notice 就与整个界面共用同一个语言。
 *
 * 与 ADR 0031 复评记录二为组件立下的「静态清单存 key、渲染时才 `t()`」
 * 是同一条纪律 —— 那时漏掉的正是命令面与这两处快照。
 *
 * ## `text` 变体只服务一个来源，且**有编译期拦截**
 *
 * `images.ts` 的 `PasteImageFail.reason` 已经是查好表的一句话，而那个文件
 * **不在本轮改动范围内**（它把 6 条译文烤成了字符串）。所以这一支保留原样。
 *
 * **它曾经「只有注释约束、无编译期拦截」**（门下实测：在白名单文件里插一句
 * `{ text: '硬编码译文' }` → `tsc` / 全量测试**全绿**）—— 注释拦不住任何人。
 * 现在 `text` 变体带一个**品牌符号**（`LegacyTextNotice` 上的
 * `[LEGACY_TEXT]`），且 `LegacyTextNotice` 与 `legacyText()` 都收成
 * **模块私有**：只有本文件能造出它，直接写字面量 `{ text: ... }` 是
 * **编译错误**（缺品牌字段）。与命令面的 `BuiltInCommandSpec` 不接受
 * 字面量 `title` 是同一条纪律 —— 该拦的形状交给类型系统，不靠人记得。
 *
 * **重审条件**：哪天 `images.ts` 交回 key（那时 `text` 变体整个删除），
 * 品牌符号随之一并删掉。
 *
 * ## 本轮的已知残留（不是「唯一」—— 详见 ADR 0031 复评记录四）
 *
 * - `core/document/images.ts` 的 `PasteImageFail.reason`（译文烤成字符串，
 *   本文件**模块私有**的 `legacyText()` 就是给它留的出口）；
 * - `core/search/replace.ts` 的跨文件替换汇总（`describeReplaceOutcome` /
 *   `skipped[].reason` 全是 `t()` 拼好的句子）→ 经 `App.tsx` 的
 *   `handleReplaceAll` 返回值 → `SearchPanel.tsx:81` 存进 state；
 * - `core/workspace/store.ts` 的 `tree.error` 一族 —— 有**两个**生产者：
 *   ① `failTreeAction` 的 **10 处调用**（其中 **8 处**直传 `t()` 结果：
 *   `fsManageUnsupported` / `cannotDetermineTarget` ×2 / `createFailed` /
 *   `renameNoDir` / `renameFailed` / `deleteDirty` / `deleteFailed`），
 *   以及 `toggleDir` 里直接写 `tree.error` 的 `cannotReadDir` 一处；
 *   ② `core/path.ts` 的 `validateEntryName()` —— 它**逐条 `return t()`**
 *   （`core.path.*` 七条），经 `failTreeAction(problem)` 落到 `tree.error.message`
 *   再上屏，是**同一族的独立生产者**（同一族的两个写入路径各传一个来源）；
 *   扫字段要先扫它**全部**的写入点，只看一个必然漏。
 * - `core/document/block-preview.ts:44/49/67` 的 `reason: t(...)` ——
 *   经契约交给内核画，内核不重问就不会重算；
 * - `SettingsDialog.tsx:156` `setError(t('ui.settings.keyUnsafe'))`。
 *
 * 上面这几处**都不在本单白名单内**，逐处判据见回奏的 AC6 扫描清单。
 * `error` / `conflict` 两个 store 字段与 notice 同型，已在同一批里改齐
 * （见 `localizedError` / `localizedConflict`）。
 *
 * 同一族还有**一处白名单内**的，判的是「不改 + 登记」：
 * `App.tsx` 的 `resolveDocBlockPreview` 回调里
 * `reason: t('ui.app.refNoBaseReason')`（宿主文档没有路径信息时）。它与
 * `block-preview.ts:44/49/67` 是**同一个契约字段**（`DocBlockPreview.reason`，
 * 内核拿到就画、不重问），单侧改不动：① 修它要动 `core/editor/types.ts`
 * 的契约（白名单外）；② 该分支生产不可达 —— 标签页只在 `store.open()`
 * 成功后入 `tabs`，活动面板的 `doc` 恒非 null，唯一置 null 的 `close()`
 * 在生产代码里无人调用。**白名单是「允许动的边界」，不是「必须改的清单」**
 * —— 判据落在「`t()` 结果是否跨渲染存活 + 可达性」上，不落在文件归属上。
 *
 * **注意 `DocumentPane.tsx` 的 `editorError` 不在上面** —— 它存的是
 * `thrown.message`（内核抛出的原始错误），不是 `t()` 的返回值，
 * 因此不属这一族。这是扫描时最容易误列的一处，判据要落到**来源**上。
 */

/** `text` 变体的品牌符号。**只有这个模块能造出带它的对象。** */
const LEGACY_TEXT: unique symbol = Symbol('levnote.legacyTextNotice');

/**
 * 已经查好表的一句话（**旧形态**）。
 *
 * 带上品牌符号之后，`{ text: '...' }` 这种字面量**不再是** `Notice` ——
 * 编译器会挡住它。**它不对外导出**：想造一个只能在本文件里走 `legacyText()`，
 * 而那个函数的文档字符串里写着「只为 `images.ts` 那一个来源保留」。
 * 收成私有是刻意的 —— 全仓对它的引用**只在 `store.ts` 内部**（写这一句时
 * 已 grep 核实），导出它等于给任何文件留一道绕开门（刑部封驳实测：
 * `export` 状态下 `legacyText('硬编码译文')` 与
 * `{ ...legacyText(''), text: '…' }` 两种写法 `tsc` 都是 rc 0）。
 */
interface LegacyTextNotice {
  readonly text: string;
  readonly [LEGACY_TEXT]: true;
}

/**
 * 把一句已经查好表的译文包成 `Notice`。**模块私有**（理由见上）。
 *
 * 只为 `core/document/images.ts` 的 `PasteImageFail.reason` 保留（那个文件
 * 不在本单白名单内，把 6 条译文烤成了字符串）。**新写的 notice 一律走 `key`
 * 变体** —— 走这条出口会让那句话停在写入时的语言。
 *
 * **已知边界：品牌门挡的是「字面量」，不是「绕道」。** 先可信地调一次
 * `legacyText('x')`、再 `{ ...legacyText('x'), text: '硬编码译文' }` 仍能
 * 过 `tsc`（实测，仅第 1 行字面量报错）—— 只是那要求调用方**先碰一次**这个
 * 出口，与「随便写个 `{ text: ... }` 就过」不是一回事。
 * **收成模块私有之后，能碰到它的只剩本文件**：将来要破门**必须先在本文件
 * 加一个导出**，那一步一定看得见。所以准确说法是「**可疑动作只能落在本文件
 * 内**」，而不是「`tsc` 挡得住一切绕道」。
 */
function legacyText(text: string): LegacyTextNotice {
  return { text, [LEGACY_TEXT]: true };
}

export type Notice =
  | { readonly key: MessageKey; readonly params?: Readonly<Record<string, string | number>> }
  | LegacyTextNotice;

/**
 * 把 `Notice` 渲染成一句话。
 *
 * 查表只发生在这里（渲染的那一刻）—— 别在写入 state 的地方先 `t()` 一遍，
 * 那正是这份类型要消除的写法。两个消费方（状态栏、查找栏）共用它，
 * 免得各写一份联合类型的分支。
 */
export function renderNotice(notice: Notice): string {
  return 'text' in notice ? notice.text : t(notice.key, notice.params);
}

/**
 * 错误横幅 / 冲突横幅里那句话（同批 · #TASK-I18N-SNAPSHOT-01）。
 *
 * ## 形态同族，修法不同：这里用 getter，不是结构化联合
 *
 * `error` / `conflict` 与 `notice` 是**同一种快照形态**（`t()` 的结果在动作
 * 那一刻被烤进 store 字段，之后切语言不再重算），但字段类型没法像 `notice`
 * 那样改成联合 —— `error` 的契约是 `AppErrorShape`，它**在别处被消费**
 * （`core/workspace/store.ts` 的 `openError`、`App.tsx` 的打开失败横幅），
 * 而那个文件不在本单白名单内。改字段类型会连着它们一起动。
 *
 * 而「读取时才查表」这件事**不需要改类型** —— 一个 getter 就够：
 * 消费者写的仍是 `error.message`，只是那一下会现查表。这正是 ADR 0031
 * 复评记录三为命令面立过的形状（`title` / `category` 是读的那一刻才查表的
 * getter），理由也相同：**让「消费方零改动」与「求值时机正确」同时成立**。
 *
 * ## 这不是「随便存译文也行」
 *
 * 能存译文的只有**最终渲染出来的那一份**。这里 getter 给出的仍是
 * 「**读的那一刻**」的译文，不是「写入那一刻」的 —— 与 `notice` 存 key
 * 是同一条纪律的两种落地。
 *
 * **重审条件**：若哪天 `AppErrorShape` 的消费方都收进了白名单，
 * 可以把它统一成 `Notice` 那样的结构化联合 —— 那时 getter 就该退场。
 */
function localizedError(
  code: ErrorCode,
  key: MessageKey,
  params?: Readonly<Record<string, string | number>>,
): AppErrorShape {
  return {
    code,
    detail: null,
    // getter：查表推迟到读取（渲染）那一刻，运行中切语言也跟着变
    get message(): string {
      return t(key, params);
    },
  };
}

/** 冲突横幅那句话。与 `localizedError` 同一形状，服务 `ConflictInfo`。 */
function localizedConflict(
  key: MessageKey,
  params?: Readonly<Record<string, string | number>>,
): ConflictInfo {
  return {
    detectedAt: Date.now(),
    get message(): string {
      return t(key, params);
    },
  };
}

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
  /** 状态栏的一句话反馈。**存 key 不存译文** —— 见 `Notice`。 */
  notice: Notice | null;
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
    notice: null,
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
  clearNotice = (): void => this.commit({ notice: null });

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
   * `resolveRelative`（分隔符统一成**文档路径**的风格，硬约束 12），读取走
   * `readTextFile` —— 与打开文档同一个解码器，`.drawio` 是 XML 文本。
   *
   * **不缓存**（与 `resolveImageUrl` 同一条理由）且**不抛**（调用链尽头是
   * NodeView 的一次渲染，未捕获的拒绝只会变成一个没人处理的 Promise）；
   * 读不到返回 `null`，内核据此画带路径的说明 —— 失败要说出来，不能留白。
   */
  resolveDocBlock = async (src: string): Promise<string | null> => {
    const doc = this.state.doc;
    if (!doc) return null;
    const absolute = resolveRelative(doc.path, src);
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
      this.commit({ notice: legacyText(result.reason) });
      return null;
    }

    // 说清楚「图片已经在磁盘上了，但文档还没保存」—— 这是用户真正会担心的事
    this.commit({ notice: { key: 'core.store.imageSaved', params: { name: basename(result.path) } } });
    return result.src;
  };

  /** 打开一个路径。失败时把错误放进 state，由 UI 决定怎么呈现。 */
  async open(path: string): Promise<boolean> {
    if (this.state.doc && isDirty(this.state.doc)) {
      // 调用方（UI）负责先问用户是否丢弃改动；这里只做兜底
      this.commit({ error: localizedError('INTERNAL', 'core.store.dirtyBeforeFormat') });
      return false;
    }

    this.commit({ busy: true, error: null, conflict: null, viewMode: 'rich' });
    try {
      const file = await this.provider.readTextFile(path);
      const doc = toDocument(file, (this.sessionCounter += 1));
      this.commit({
        doc,
        busy: false,
        notice: doc.lossy
          ? { key: 'core.store.openedLossy', params: { name: doc.displayName } }
          : { key: 'core.store.opened', params: { name: doc.displayName } },
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
   * 文件在磁盘上被重命名 / 移动了（文件树管理，A2）：只改路径，**内容原样保留**。
   *
   * 与 `saveAs` 同一个道理 —— 换路径不换内容：`session` 不变，编辑器实例不重建，
   * 撤销栈、光标、视图模式全都保住。脏改动也**原样保留**：改名不碰字节，
   * `text` 对 `savedText` 的差异在改名之后依然成立，下一次 Ctrl+S 会把这份
   * 差异写到**新路径**上（rename 不改 mtime，乐观锁仍然有效）。
   *
   * 这正是「改名」与「删除」的分工差异：删除碰到了内容归属（脏缓冲会变成
   * 「凭空写回一个新文件」），所以必须拦下脏标签页；改名没有这个问题。
   */
  retargetPath(newPath: string): void {
    const doc = this.state.doc;
    if (!doc || doc.path === newPath) return;
    this.commit({
      doc: { ...doc, path: newPath, displayName: basename(newPath) },
      notice: { key: 'core.store.renamed', params: { name: basename(newPath) } },
    });
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
        error: localizedError('PERMISSION_DENIED', 'core.store.readonlySave', { name: doc.displayName }),
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
        notice: { key: 'core.store.saved', params: { name: doc.displayName } },
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
      this.commit({
        doc: next,
        busy: false,
        conflict: null,
        notice: { key: 'core.store.savedAs', params: { name: next.displayName } },
      });
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
      this.commit({
        doc: toDocument(file, doc.session),
        busy: false,
        conflict: null,
        notice: { key: 'core.store.reloaded', params: { name: doc.displayName } },
      });
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
            notice: { key: 'core.store.overwritten', params: { name: doc.displayName } },
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
      this.commit({ conflict: localizedConflict('core.store.conflictDetected') });
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

    this.commit({ doc: null, conflict: null, error: null, busy: false, notice: null });
    return true;
  }

  dispose(): void {
    this.listeners.clear();
  }
}
