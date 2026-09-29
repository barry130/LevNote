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
}

export const DEFAULT_SETTINGS: ViewSettings = {
  theme: DEFAULT_THEME,
  sidebarWidth: 240,
  sidebarVisible: true,
  outlineVisible: false,
  wordWrap: false,
  codeLineNumbers: true,
};

export const SIDEBAR_MIN_WIDTH = 160;
export const SIDEBAR_MAX_WIDTH = 480;

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
    outlineVisible: asBoolean(record['outlineVisible'], DEFAULT_SETTINGS.outlineVisible),
    wordWrap: asBoolean(record['wordWrap'], DEFAULT_SETTINGS.wordWrap),
    codeLineNumbers: asBoolean(record['codeLineNumbers'], DEFAULT_SETTINGS.codeLineNumbers),
  };
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
