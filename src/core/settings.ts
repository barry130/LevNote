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
import { SUPPORTED_LOCALES, type SupportedLocale } from './i18n/messages.js';

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
  /**
   * 编辑器字号（px，12–28，默认 15）。
   *
   * **一个旋钮管三个内核**：Markdown 正文直接用它；代码内核与纯文本
   * 各自按比例折算（13.5px / 14px ≈ 15 × 0.9 / 0.93），默认档下三个内核
   * 的观感与引入这设置之前**逐像素一致** —— 老用户不会察觉任何变化，
   * 而调大调小时三个内核一起动。导出的「纸」不受它影响（见
   * `document-styles.css` 里的 var 回退）。
   */
  readonly editorFontSize: number;
  /**
   * 编辑器正文字体（font-family 的值）。**空串 = 内置默认**。
   *
   * 只作用于比例字体文本（Markdown 正文、纯文本）；代码与代码块**始终
   * 等宽** —— 把比例字体塞进代码编辑器是破坏可用性的，不做。
   *
   * 值会进 CSS 变量，所以**只接受白名单字符**（字母数字空格逗号引号
   * 连字符与 CJK）：`url(` / 分号 / 花括号这类声明语法一律剥掉 ——
   * 这是个纯本地应用，字体值不该有能力发起任何外部引用。
   */
  readonly editorFontFamily: string;
  /** 编辑器行高（1.3–2.4，默认 1.75）。与字号同一条「三个内核一起动」的口径。 */
  readonly editorLineHeight: number;
  /**
   * 界面语言（默认 zh-CN）。真相来源在设置，App 的 effect 把它推给 i18n。
   * 命令系统、全部界面文案与 core 层消息均已双语（2026-10-07，ADR 0031）。
   */
  readonly locale: SupportedLocale;
  /**
   * 专注模式（默认关）：光标不在其中的段落调淡。
   *
   * 只对实现了它的内核生效（Markdown 富文本是主要场景）—— 代码 / 纯文本 /
   * 导图内核忽略它，不打断用户。做成可见的开关而非永远可用的能力，
   * 理由与 `wordWrap` 相同：它是「我此刻想怎么读」的偏好，不是能力开关。
   */
  readonly focusMode: boolean;
  /**
   * 打字机模式（默认关）：每次光标移动后把光标行滚到编辑区垂直正中。
   *
   * Markdown 与代码内核实现；对导图没有意义（没有线性光标），忽略。
   * 它**改写滚动位置**，与位置记忆同走一条路 —— 打字机开着时位置记忆
   * 记下的就是居中后的位置，这是符合直觉的（下次打开还在同样的观感）。
   */
  readonly typewriterMode: boolean;
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
  editorFontSize: 15,
  editorFontFamily: '',
  editorLineHeight: 1.75,
  locale: 'zh-CN',
  focusMode: false,
  typewriterMode: false,
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

export const EDITOR_FONT_SIZE_MIN = 12;
export const EDITOR_FONT_SIZE_MAX = 28;
export const EDITOR_LINE_HEIGHT_MIN = 1.3;
export const EDITOR_LINE_HEIGHT_MAX = 2.4;

/** 设置页实时套用的也是这份夹取 —— 输入中途的值同样不该以越界形态生效。 */
export function clampEditorFontSize(value: unknown): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) return DEFAULT_SETTINGS.editorFontSize;
  return Math.min(EDITOR_FONT_SIZE_MAX, Math.max(EDITOR_FONT_SIZE_MIN, Math.round(value)));
}

export function clampEditorLineHeight(value: unknown): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) return DEFAULT_SETTINGS.editorLineHeight;
  const clamped = Math.min(EDITOR_LINE_HEIGHT_MAX, Math.max(EDITOR_LINE_HEIGHT_MIN, value));
  // 两位小数足够了：行高的第三位小数没人看得出来，留着只会存进 1.7500000001 这类脏值
  return Math.round(clamped * 100) / 100;
}

/**
 * 字体族白名单：字母、数字、空格、逗号、引号、连字符、下划线与 CJK。
 * 自定义值会被塞进 CSS 变量，凡是能让它构成一条声明语法的东西
 * （分号、花括号、括号）一律剥掉。
 */
export function sanitizeEditorFontFamily(value: unknown): string {
  if (typeof value !== 'string') return DEFAULT_SETTINGS.editorFontFamily;
  return value
    .replace(/[^a-zA-Z0-9 ,'"_\-\u4e00-\u9fa5\u3040-\u30ff\uac00-\ud7af]/g, '')
    .trim()
    .slice(0, 120);
}

const STORAGE_KEY = 'levnote.settings.v1';

/** 只接受 `Storage` 形状，方便测试传入一个内存实现。 */
export type StorageLike = Pick<Storage, 'getItem' | 'setItem'>;

/**
 * 一次「写到偏好存储」的结果。
 *
 * ## 为什么不返回 `void`
 *
 * `localStorage` 在三种情况下会**直接抛**：隐私模式、配额用尽、企业策略禁用。
 * 第一版的 `saveSettings` / `savePositions` 把这三种都当成「静默放弃」——
 * 注释写的是「偏好没存上不是需要打断用户的事」。那句话对**打断**是对的
 * （不该弹模态框拦住用户），对**不说话**是错的：用户改了主题、关掉应用、再打开，
 * 主题回到了默认值，而全程没有任何一个字告诉他「上次那个改动没记住」。
 * 硬约束 2 说的正是这一类：不静默降级。
 *
 * 所以返回值里带一个 `ok`。**调用方**拿着它决定怎么说 ——
 * 存储层自己不知道该弹横幅还是该进 notice（它连 UI 都没有）。
 *
 * ## 为什么 `ok: false` 也分「没有存储」与「写失败」两种
 *
 * 见 `PersistFailure`：两者对用户意味着完全不同的事。
 * `null` 存储是**已知且无害**的（浏览器预览、隐私模式，本来就没打算持久化），
 * 为它弹一句「没能记住」是纯粹的噪音；写失败才是真的坏了。
 *
 * 与 `core/session.ts` 的 `SaveOutcome` 是同一条纪律的两个实例 ——
 * 那份先做到，这份是照着它推广过来的（ADR 0042）。
 */
export type PersistOutcome =
  /** 写进去了（或本来就无事可做）。 */
  | { readonly ok: true }
  /** 没能写进去。 */
  | { readonly ok: false; readonly reason: PersistFailure };

/**
 * 写不进去的两种成因。
 *
 * - `'no-storage'`：根本没有存储可用（隐私模式、`defaultStorage()` 给了 null）。
 *   这是**预期内**的降级，应用在浏览器预览里就是这么跑的 —— 不说。
 * - `'write-failed'`：有存储，但 `setItem` 抛了（配额用尽 / 被禁用）。
 *   这是**降级**，要说。
 */
export type PersistFailure = 'no-storage' | 'write-failed';

/** 成功的那一份。用常量避免每次返回都造一个新对象。 */
export const PERSISTED: PersistOutcome = { ok: true };

/** 写失败的那一份。同理，常量。 */
export function persistFailed(reason: PersistFailure): PersistOutcome {
  return reason === 'no-storage' ? { ok: false, reason: 'no-storage' } : { ok: false, reason: 'write-failed' };
}

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
    editorFontSize: clampEditorFontSize(record['editorFontSize']),
    editorFontFamily: sanitizeEditorFontFamily(record['editorFontFamily']),
    editorLineHeight: clampEditorLineHeight(record['editorLineHeight']),
    locale: isSupportedLocale(record['locale']) ? record['locale'] : DEFAULT_SETTINGS.locale,
    focusMode: asBoolean(record['focusMode'], DEFAULT_SETTINGS.focusMode),
    typewriterMode: asBoolean(record['typewriterMode'], DEFAULT_SETTINGS.typewriterMode),
  };
}

/** 语言字段的逐项校验：不在清单里的一律回落 zh-CN（与主题同一个写法）。 */
function isSupportedLocale(value: unknown): value is SupportedLocale {
  return typeof value === 'string' && (SUPPORTED_LOCALES as readonly string[]).includes(value);
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

/**
 * 把偏好写回存储。
 *
 * 不抛。写不进去时返回 `ok: false`（ADR 0042）—— 偏好没存上**不该打断用户**
 * （不弹模态框、不拦住他下一步操作），但也不能装作存上了：
 * 用户在设置页里改了主题，下次启动回到默认值，那是一次**静默降级**。
 * 说什么、怎么说由调用方决定（见 `PersistOutcome` 的注释）。
 */
export function saveSettings(storage: StorageLike | null, settings: ViewSettings): PersistOutcome {
  if (!storage) return persistFailed('no-storage');
  try {
    storage.setItem(STORAGE_KEY, JSON.stringify(settings));
    return PERSISTED;
  } catch {
    return persistFailed('write-failed');
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
