/**
 * 会话恢复 —— 「上次开着什么，这次还开着什么」，以及**未保存的改动不丢**。
 *
 * ## 为什么这是数据安全，不是体验项
 *
 * 位置记忆（`positions.ts`）坏了只是「打开文档回到开头」，用户笑一下就过去了。
 * 这一份坏了是**用户刚写的字没了**。所以两者的校验策略刻意不同：
 * 位置是「任何一项不合法就整条丢」（一条记录要么整体可信、要么没有意义 ——
 * 拿一个错的 `offset` 去恢复会把光标悄悄移到别处）；这里是**逐条丢**，
 * 一个标签页的记录坏了不该连累其余九个。
 *
 * ## 为什么草稿必须有上限，且上限必须**说出来**
 *
 * 草稿是**整篇文档**，不是几十字节的位置记录。配额用尽时 `setItem` 会直接抛，
 * 而抛的那一刻**所有草稿都写不进去** —— 那比「丢掉最旧的一条」糟得多。
 * 所以按「最后编辑时间」从旧到新裁剪，直到装得下。
 *
 * 但裁剪就是**丢内容**，而「绝不静默」是硬约束 2。所以被裁掉几条要记进
 * `droppedDrafts` 存下去，由外壳在**下一次启动**时如实告诉用户 ——
 * 他至少知道自己丢了什么、下次该先保存。
 *
 * ## 为什么和 `settings` / `positions` 各用一个键
 *
 * 三者的生命周期与损坏代价都不同：设置是用户显式改的、位置是机器记的小数据、
 * 会话是机器记的**大数据**。合成一个键之后，任何一次写草稿都会把设置整份重写一遍，
 * 而且存储坏了会一起没。分开之后坏哪份丢哪份。
 *
 * ## 已知限制（写下来而不是留着让人猜）
 *
 * **一、只恢复磁盘上还在的文件。** 草稿跟着路径走，文件被删 / 被移走之后没有落脚点 ——
 * 要救那种情况需要「无路径的虚拟标签页」，那是另一套东西（保存时得先问存哪儿，
 * 而「问存哪儿」的对话框在启动恢复这条路径上会连着弹好几个）。
 * 当前取舍是：**宁可明确地不恢复，也不要造一个用户看不懂的标签页**。
 *
 * **二、只恢复了「文本」，没恢复用户手动指定的编码与行尾。** 那两项在 `open()`
 * 读磁盘时重新探测，所以「改了行尾但还没保存就崩了」这一种改动会丢。
 * 为什么不一起存：那两项在正常路径上是**磁盘的属性**，存一份等于造出第二个
 * 真相来源 —— 文件在应用关闭期间被别的工具转码之后，恢复出来的记录会以**旧编码**
 * 写回去，而那是一次静默转码（硬约束 2）。宁可丢一次「手动改了行尾」，
 * 也不要冒一次静默转码的风险。
 */

import type { StorageLike } from './settings.js';

/** 一个标签页的会话记录。 */
export interface SessionTab {
  readonly path: string;
  /**
   * 未保存的草稿。`null` = 这个标签页没有未保存改动（内容从磁盘读）。
   *
   * 刻意**不存编码与行尾**：那些在 `open()` 读磁盘时就拿到了，存一份等于
   * 造出第二个真相来源 —— 文件在应用关闭期间被别的工具转码之后，
   * 恢复的草稿会以**旧编码**写回去，而那是一次静默转码（硬约束 2）。
   */
  readonly draft: string | null;
  /** 这一条最后被改动的时间戳（毫秒）。只用于容量裁剪的顺序。 */
  readonly at: number;
}

export interface SessionState {
  readonly tabs: readonly SessionTab[];
  /** 上次激活的标签页路径。它已经不在 `tabs` 里时会被忽略。 */
  readonly activePath: string | null;
  /** 上次打开的工作区根目录。`null` = 没打开文件夹。 */
  readonly root: string | null;
  /**
   * 上一次写盘时，因为超出容量（或存储损坏）而**没能存下**的草稿数。
   *
   * 它必须被存下来：裁剪发生在写的时候，而告知发生在**下一次启动**，
   * 中间隔着一次进程结束。没有这个字段，用户就永远不知道自己的草稿被丢了。
   */
  readonly droppedDrafts: number;
}

export const EMPTY_SESSION: SessionState = {
  tabs: [],
  activePath: null,
  root: null,
  droppedDrafts: 0,
};

/**
 * 存储键。
 *
 * `v1` 与设置、位置那两个键**各自独立**：改了会话的格式只让会话失效，
 * 用户的主题与光标记忆照旧。
 */
const STORAGE_KEY = 'qingmo.session.v1';

/** 最多恢复多少个标签页。超出的按「最后编辑时间」从旧到新丢。 */
export const MAX_SESSION_TABS = 50;

/**
 * 草稿总量的上限（按 UTF-16 码元数，也就是 `String#length`）。
 *
 * 取 150 万：WebView 的 localStorage 配额一般是 500 万码元，留三倍余量给设置、
 * 位置与将来别的东西。按中文算约 150 万字，够放下十几份长文档的草稿 ——
 * 而真正会「改完没保存就崩」的，通常是用户正在写的那一两份。
 */
export const MAX_DRAFT_CHARS = 1_500_000;

function isFiniteNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value);
}

/**
 * 解析一条标签页记录。
 *
 * 返回 `null` = **整条不可用**（连路径都不合法），调用方丢掉这一条。
 * 只有草稿不合法时**保留路径、丢掉草稿**并计入 `droppedDrafts` ——
 * 路径还在就还能把文件打开，比整条丢掉有用得多。
 */
function parseTab(value: unknown): { tab: SessionTab; droppedDraft: boolean } | null {
  if (typeof value !== 'object' || value === null) return null;
  const record = value as Record<string, unknown>;

  const path = record['path'];
  if (typeof path !== 'string' || path === '') return null;

  const rawDraft = record['draft'];
  const draft = typeof rawDraft === 'string' ? rawDraft : null;
  // `null` 是合法值（表示没有草稿），只有「既不是 null 也不是字符串」才算坏
  const droppedDraft = rawDraft !== null && typeof rawDraft !== 'string';

  const at = record['at'];
  return { tab: { path, draft, at: isFiniteNumber(at) ? at : 0 }, droppedDraft };
}

/** 从存储里读会话。任何异常（读不出、不是 JSON、形状不对）都回落到空会话。 */
export function loadSession(storage: StorageLike | null): SessionState {
  if (!storage) return EMPTY_SESSION;

  let raw: string | null;
  try {
    raw = storage.getItem(STORAGE_KEY);
  } catch {
    return EMPTY_SESSION;
  }
  if (raw === null || raw === '') return EMPTY_SESSION;

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    // 内容坏了：当成没有会话，**不清空**（用户可能只是手改错了，
    // 下一次写就会覆盖掉）。清空反而多一次写操作、多一次失败机会。
    return EMPTY_SESSION;
  }
  if (typeof parsed !== 'object' || parsed === null) return EMPTY_SESSION;

  const record = parsed as Record<string, unknown>;
  const rawTabs = record['tabs'];
  if (!Array.isArray(rawTabs)) return EMPTY_SESSION;

  const tabs: SessionTab[] = [];
  const seen = new Set<string>();
  let droppedDrafts = 0;

  for (const item of rawTabs) {
    const parsedTab = parseTab(item);
    // 连路径都不合法的整条丢掉。**不计入 `droppedDrafts`** —— 那个字段的语义是
    // 「有多少份草稿没能恢复」，而这种是存储被手改坏了的异常情形，路径都没了，
    // 告诉用户也做不了任何事。
    if (parsedTab === null) continue;
    // 同一个文件不该有两个标签页（与 `WorkspaceStore.open` 的去重同一条理由：
    // 两处各改一半，保存时的乐观锁冲突对用户完全无法理解）
    if (seen.has(parsedTab.tab.path)) continue;
    seen.add(parsedTab.tab.path);
    tabs.push(parsedTab.tab);
    if (parsedTab.droppedDraft) droppedDrafts += 1;
  }

  const rawActive = record['activePath'];
  const activePath = typeof rawActive === 'string' && seen.has(rawActive) ? rawActive : null;

  const rawRoot = record['root'];
  const root = typeof rawRoot === 'string' && rawRoot !== '' ? rawRoot : null;

  // 上一次写盘时丢掉的还没被告知过 → 累加进来（而不是覆盖），
  // 否则「先丢 2 条、再丢 3 条」最后只提示 3 条
  const storedDropped = record['droppedDrafts'];
  if (isFiniteNumber(storedDropped) && storedDropped > 0) droppedDrafts += storedDropped;

  return { tabs, activePath, root, droppedDrafts };
}

/**
 * 按容量上限裁剪。**纯函数**，不改入参。
 *
 * 两步，顺序不能反：
 *
 * 1. 先按**标签页数**裁 —— 超出的整条丢掉（含草稿）；
 * 2. 再按**草稿总量**裁 —— 只把草稿置 `null`，**路径留着**。
 *
 * 第二步刻意不丢整条：用户至少还能看到「这个文件当时开着」，点一下就打开了。
 * 丢路径换来的那点空间，远不如这个有用。
 */
export function fitSession(state: SessionState): { state: SessionState; droppedDrafts: number } {
  let tabs = [...state.tabs];
  let droppedDrafts = 0;

  if (tabs.length > MAX_SESSION_TABS) {
    // 按最后编辑时间从新到旧留
    const kept = [...tabs].sort((a, b) => b.at - a.at).slice(0, MAX_SESSION_TABS);
    const keptPaths = new Set(kept.map((tab) => tab.path));
    for (const tab of tabs) {
      if (!keptPaths.has(tab.path) && tab.draft !== null) droppedDrafts += 1;
    }
    tabs = kept;
  }

  let total = 0;
  for (const tab of tabs) total += tab.draft === null ? 0 : tab.draft.length;

  const drop = new Set<string>();
  if (total > MAX_DRAFT_CHARS) {
    for (const tab of [...tabs].sort((a, b) => a.at - b.at)) {
      if (total <= MAX_DRAFT_CHARS) break;
      if (tab.draft === null) continue;
      total -= tab.draft.length;
      drop.add(tab.path);
    }
  }

  if (drop.size === 0) return { state: { ...state, tabs }, droppedDrafts };

  return {
    state: { ...state, tabs: tabs.map((tab) => (drop.has(tab.path) ? { ...tab, draft: null } : tab)) },
    droppedDrafts: droppedDrafts + drop.size,
  };
}

/**
 * 两份会话在**要落盘的内容**上是否相同。
 *
 * 存在的理由只有一个：`saveSession` 每次都要把整份草稿 `JSON.stringify` 一遍，
 * 而它的调用方是被「每次按键后的防抖」和「切标签页」两处高频事件驱动的 ——
 * 绝大多数触发时文本根本没变（切视图、状态栏提示变化、脏标记来回翻）。
 * 没有这道闸，用户每次切标签页都要把几百 KB 的草稿重写一遍。
 *
 * ## 为什么 `droppedDrafts` 也算在内
 *
 * 它读出来之后就该被清零（外壳已经把它显示给用户了）。算进来的效果是：
 * 启动后的第一次写盘会因为「旧账目 ≠ 0」而必然发生，正好把那份账目清掉。
 * 不算的话，那次写盘可能被跳过，用户**下一次启动又会看到同一条提示**。
 *
 * 比较是**按顺序逐条**的。顺序必须参与比较：`restoreTabs` 就是按这份顺序重建
 * 标签页的，不比的话用户拖出来的顺序在下次启动时会悄悄回到旧样子。
 */
export function sameSession(a: SessionState, b: SessionState): boolean {
  if (a === b) return true;
  if (a.activePath !== b.activePath) return false;
  if (a.root !== b.root) return false;
  if (a.droppedDrafts !== b.droppedDrafts) return false;
  if (a.tabs.length !== b.tabs.length) return false;

  for (let i = 0; i < b.tabs.length; i += 1) {
    const x = a.tabs[i];
    const y = b.tabs[i];
    if (x === undefined || y === undefined) return false;
    if (x.path !== y.path) return false;
    // `draft` 用 `!==` 比：两边是同一个字符串对象时是 O(1) 的指针比较，
    // 而「文本没变」恰好就是这个情形（同一个 `doc.text` 引用）。
    if (x.draft !== y.draft) return false;
    if (x.at !== y.at) return false;
  }
  return true;
}

export interface SaveOutcome {
  /**
   * 实际写下去的那一份（`droppedDrafts` 已累加）。
   *
   * 调用方要拿它当「下次比较的基准」，否则会被「我传了完整的一份、
   * 存下去的却少了几条草稿」搞乱 —— 下一轮比较又会认为有变化，反复重写。
   */
  readonly state: SessionState;
  /**
   * 写成功为 `true`。
   *
   * `false` 时外壳**应当提示用户手动保存**：存储被禁用或写不进去，
   * 「自动备份」这条线断了，而用户完全无从察觉（硬约束 2）。
   */
  readonly ok: boolean;
}

/**
 * 写会话。**内部先裁剪**，所以调用方给什么都装得下。
 *
 * 不会抛：存储在某些环境下会直接抛（隐私模式、企业策略禁用），
 * **会话记不住不该让应用起不来**，但也不能装作没事 —— 见 `ok`。
 */
export function saveSession(storage: StorageLike | null, state: SessionState): SaveOutcome {
  const fitted = fitSession(state);
  const next: SessionState = {
    ...fitted.state,
    // 上一次丢的还没被告知过 → 累加而不是覆盖
    droppedDrafts: state.droppedDrafts + fitted.droppedDrafts,
  };

  if (!storage) return { state: next, ok: false };
  try {
    storage.setItem(STORAGE_KEY, JSON.stringify(next));
    return { state: next, ok: true };
  } catch {
    return { state: next, ok: false };
  }
}

/**
 * 忘掉整个会话。
 *
 * 目前只有测试与「用户明确要求不恢复」时用得上。刻意不提供「忘掉某一个标签页」——
 * 那是「关闭标签页」，走 `WorkspaceStore.closeTab` 就行。
 */
export function clearSession(storage: StorageLike | null): void {
  if (!storage) return;
  try {
    storage.setItem(STORAGE_KEY, JSON.stringify(EMPTY_SESSION));
  } catch {
    // 同 `saveSession`：写不进去不该让调用方崩，这里也确实无事可做
  }
}
