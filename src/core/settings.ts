/**
 * 视图偏好的持久化。
 *
 * 为什么用 `localStorage` 而不是 Tauri 的 store 插件或自己写一个 JSON 文件：
 * - WebView 的 localStorage 落在应用数据目录里，本来就是持久化的，不需要额外依赖；
 * - 偏好项都是「丢了也不心疼」的东西（主题、侧栏宽度），不值得为它引入
 *   一个带 IPC 往返、需要处理并发写入的存储层；
 * - 浏览器预览模式（`vite dev`）下它是唯一能用的持久化手段，两边行为一致。
 *
 * **所有读写都包在 try/catch 里**：localStorage 在某些环境下会直接抛
 * （隐私模式、存储配额用尽、企业策略禁用）。偏好读不出来不该让应用起不来 ——
 * 回落到默认值即可。这是这一层唯一需要小心的失败模式。
 */

import { DEFAULT_THEME, isThemePreference, type ThemePreference } from './theme/theme.js';

export interface ViewSettings {
  readonly theme: ThemePreference;
  /** 侧栏宽度（像素） */
  readonly sidebarWidth: number;
  readonly sidebarVisible: boolean;
  /**
   * 侧栏里现在显示哪一栏。
   *
   * ## 为什么与 `sidebarVisible` 分开，而不是合成一个三态枚举
   *
   * 「看不看侧栏」与「侧栏里看什么」是两个独立的问题。合成 `'hidden' | 'files' | 'search'`
   * 之后，「关掉侧栏再打开」会**丢掉用户上次看的是哪一栏** ——
   * 而那恰恰是他最可能想要回来的东西（正在搜到一半、先关掉看一眼代码）。
   * 分开之后，关掉再打开必然回到原来那一栏。
   */
  readonly sidebarView: SidebarView;
  readonly outlineVisible: boolean;
  /** 代码 / 纯文本内核是否软换行。Markdown 内核不适用。 */
  readonly wordWrap: boolean;
  /**
   * Markdown 的代码块是否显示行号。
   *
   * 默认**开**。这是一个取舍：Typora 默认关（代码块里多一列数字，
   * 短代码片段看着会重一些）。选开的理由是它更容易被发现 ——
   * 关掉是「看不见但可以打开」，打开是「看得见但可以关掉」，
   * 而后者对第一次用的人友好得多。开关就在代码块右上角那个语言标签点开的面板里，
   * 关掉只要一次点击。
   *
   * 重审条件：如果收到「代码块太吵」这类反馈明显多过「行号好用」，
   * 就把默认值翻过来。
   */
  readonly codeLineNumbers: boolean;
  /**
   * 是否记住每份文档的光标与滚动位置（位置记忆）。
   *
   * 默认**开**。它是一个开关而不是永远开着，是因为它**写盘**：每次光标停顿都会
   * 往 localStorage 里写一份。虽然只是几十 KB、失败也无害，但「会不会偷偷写东西」
   * 是用户有权知道的事 —— 尤其对一个把「纯本地、无网络」写进定位的产品来说，
   * 可见的开关比不可见的默认值更符合它的性格。
   *
   * 关掉之后**已有的记忆不会被清掉**（用户可能只是想临时关一下），
   * 只是不再读也不再写。要真的清空，设置页里有独立的「清除位置记忆」。
   */
  readonly rememberPositions: boolean;
  /**
   * 分屏时左栏（源码）占的宽度比例，0.2–0.8。
   *
   * 与 `sidebarWidth` 一样是**持久化**的：分屏比例是「我这台机器、这个屏」的
   * 一次性调好的东西，每次打开都要重调一遍会很烦。
   *
   * 只存**比例**不存像素：比例与窗口大小无关，换个窗口宽度打开还是同一个观感；
   * 存像素的话在宽屏上调好的 800px，到了窄窗口就变成「预览只剩一条缝」。
   * 这也是位置记忆里滚动用比例的同一条理由（见 `EditorPosition.scrollRatio`）。
   */
  readonly splitRatio: number;
  /**
   * 启动时是否恢复上次的标签页与**未保存的草稿**。
   *
   * 默认**开**。和 `rememberPositions` 一样做成可见的开关，但理由更硬：
   * 它会在启动时**自动打开文件**，那是一次侵入性行为 —— 用户有权拒绝。
   *
   * 关掉之后**不读也不写**会话存储。代价必须说清楚：草稿持久化挂在这一条线上，
   * 关掉它意味着**异常退出时未保存的改动会丢**。设置页的提示要如实写出来，
   * 而不是只说「不恢复标签页」—— 那会让用户以为关掉它只是少了个便利。
   */
  readonly restoreSession: boolean;
  /**
   * 快捷键覆盖（P6 第 31 项，ADR 0032）：命令 id → 快捷键写法。
   *
   * **空字符串 = 显式清除**这条命令的快捷键，键不存在 = 用内置默认 ——
   * 三态缺一不可：被新快捷键顶掉的那条命令必须停在「无快捷键」，
   * 而不是悄悄回到默认值继续响应（那会让两个命令抢同一个键）。
   *
   * 值是 `Mod+J` 这类规范写法（`specFromEvent` 的产物），
   * 显示交给 `formatKeybinding`，不在这里存排版好的字符串。
   */
  readonly keybindings: Readonly<Record<string, string>>;
}

export const DEFAULT_SETTINGS: ViewSettings = {
  theme: DEFAULT_THEME,
  sidebarWidth: 240,
  sidebarVisible: true,
  sidebarView: 'files',
  outlineVisible: false,
  wordWrap: false,
  codeLineNumbers: true,
  rememberPositions: true,
  splitRatio: 0.5,
  restoreSession: true,
  keybindings: {},
};

/** 侧栏的两栏。活动栏上的两个按钮各对应一个。 */
export type SidebarView = 'files' | 'search';

function isSidebarView(value: unknown): value is SidebarView {
  return value === 'files' || value === 'search';
}

export const SIDEBAR_MIN_WIDTH = 160;
export const SIDEBAR_MAX_WIDTH = 480;

/**
 * 分屏比例的上下限。
 *
 * **刻意不是 0 和 1**：0 或 1 会让另一栏宽度归零，而它仍然占着 DOM ——
 * 用户会看到「分屏开着但只有一栏」，既不知道为什么、也拖不回来
 * （分隔条本身也只剩一个像素）。要只看一栏请用「富文本 / 源码」那两档，
 * 它们是真正的「只显示一栏」。0.2 / 0.8 保证两栏各自至少有一屏的 1/5，
 * 任何一边都还看得见、还能被拖。
 */
export const SPLIT_RATIO_MIN = 0.2;
export const SPLIT_RATIO_MAX = 0.8;

const STORAGE_KEY = 'qingmo.settings.v1';

/** 只接受 `Storage` 形状，方便测试传入一个内存实现。 */
export type StorageLike = Pick<Storage, 'getItem' | 'setItem'>;

function clampSidebar(width: unknown): number {
  if (typeof width !== 'number' || !Number.isFinite(width)) return DEFAULT_SETTINGS.sidebarWidth;
  return Math.min(SIDEBAR_MAX_WIDTH, Math.max(SIDEBAR_MIN_WIDTH, Math.round(width)));
}

function asBoolean(value: unknown, fallback: boolean): boolean {
  return typeof value === 'boolean' ? value : fallback;
}

function clampSplitRatio(value: unknown): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) return DEFAULT_SETTINGS.splitRatio;
  return Math.min(SPLIT_RATIO_MAX, Math.max(SPLIT_RATIO_MIN, value));
}

/**
 * 从存储里读偏好。
 *
 * 逐字段校验而不是整体信任：版本升级后旧数据里可能缺少新字段、
 * 或者某个字段的类型变了。逐字段回落能让「新增一个偏好项」不影响老用户的其他设置。
 */
export function loadSettings(storage: StorageLike | null): ViewSettings {
  if (!storage) return DEFAULT_SETTINGS;

  let raw: string | null;
  try {
    raw = storage.getItem(STORAGE_KEY);
  } catch {
    return DEFAULT_SETTINGS;
  }
  if (raw === null || raw === '') return DEFAULT_SETTINGS;

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    // 存储里的内容坏了：回落到默认值，不抛错也不清空（用户可能只是手改错了）
    return DEFAULT_SETTINGS;
  }
  if (typeof parsed !== 'object' || parsed === null) return DEFAULT_SETTINGS;

  const record = parsed as Record<string, unknown>;
  return {
    theme: isThemePreference(record['theme']) ? record['theme'] : DEFAULT_SETTINGS.theme,
    sidebarWidth: clampSidebar(record['sidebarWidth']),
    sidebarVisible: asBoolean(record['sidebarVisible'], DEFAULT_SETTINGS.sidebarVisible),
    sidebarView: isSidebarView(record['sidebarView']) ? record['sidebarView'] : DEFAULT_SETTINGS.sidebarView,
    outlineVisible: asBoolean(record['outlineVisible'], DEFAULT_SETTINGS.outlineVisible),
    wordWrap: asBoolean(record['wordWrap'], DEFAULT_SETTINGS.wordWrap),
    codeLineNumbers: asBoolean(record['codeLineNumbers'], DEFAULT_SETTINGS.codeLineNumbers),
    rememberPositions: asBoolean(record['rememberPositions'], DEFAULT_SETTINGS.rememberPositions),
    splitRatio: clampSplitRatio(record['splitRatio']),
    restoreSession: asBoolean(record['restoreSession'], DEFAULT_SETTINGS.restoreSession),
    keybindings: asKeybindingOverrides(record['keybindings']),
  };
}

/** 快捷键覆盖的逐项校验：只要「对象且值全是字符串」，其余整个回落为空。 */
function asKeybindingOverrides(value: unknown): Readonly<Record<string, string>> {
  if (typeof value !== 'object' || value === null) return DEFAULT_SETTINGS.keybindings;
  const out: Record<string, string> = {};
  for (const [key, entry] of Object.entries(value as Record<string, unknown>)) {
    if (typeof entry === 'string') out[key] = entry;
  }
  return out;
}

export function saveSettings(storage: StorageLike | null, settings: ViewSettings): void {
  if (!storage) return;
  try {
    storage.setItem(STORAGE_KEY, JSON.stringify(settings));
  } catch {
    // 配额用尽 / 被禁用：静默放弃。偏好没存上不是需要打断用户的事。
  }
}

/** 拿到浏览器的 localStorage，拿不到时返回 null（由调用方走默认值）。 */
export function defaultStorage(): StorageLike | null {
  try {
    if (typeof localStorage === 'undefined') return null;
    return localStorage;
  } catch {
    return null;
  }
}
