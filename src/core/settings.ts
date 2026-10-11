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

// ---------------------------------------------------------------------------
// 读侧：一次「从存储里读」的结论（ADR 0044）
// ---------------------------------------------------------------------------

/**
 * 读不出来的成因。**分法与 `PersistFailure` 是同一条**：`no-storage` 预期内、其余是坏了。
 *
 * - `'read-failed'`：有存储，`getItem` 自己抛了（隐私模式 / 企业策略禁读）。
 * - `'corrupt'`：读出来了，但**内容**与预期对不上 —— 不是 JSON、不是对象，
 *   或者里面的某几项用不了（那一档看 `lostCount` 就知道丢了没有）。
 */
export type LoadFailure = 'no-storage' | 'read-failed' | 'corrupt';

/**
 * 一次读取的结论。`PersistOutcome` 的**读侧对偶**。
 *
 * ## 为什么读侧也要有一枚 `ok`
 *
 * 0042 把写侧做到了，理由是「用户改了主题、界面立刻变了、于是他相信改成了，
 * 而下次启动回到默认值，全程没有一个字告诉他」。读侧是同一句话的另一半，
 * 而且丢得更彻底：写失败丢的是**一个改动**，读失败丢的是**整份上次的状态** ——
 * 主题、侧栏宽度、快捷键、上次开着的文件、**未保存的草稿**一起回到默认。
 *
 * 区别只有一处，而那一处很吃重：**读有默认值可依**（回落到默认，应用照常起来），
 * 所以这里仍然不抛、不白屏 —— 但「能起来」与「该说话」从来不是一件事。
 *
 * ## `lostKeys` 与 `lostCount` 为什么是两个字段
 *
 * `lostKeys` 是**点得出名**的那些键（设置的字段名、位置与最近打开的路径）。
 * 最近打开那一列是数组，被滤掉的垃圾点不出名（「第 3 项」对用户没有意义），
 * 所以那一部分只进 `lostCount`。两者分开，调用方才不会在「要报几个」上猜。
 */
export type LoadOutcome =
  /** 什么也没丢（含「存储里本来就是空的」—— 首次启动不是丢失）。 */
  | { readonly ok: true }
  | {
      readonly ok: false;
      readonly reason: LoadFailure;
      readonly lostKeys: readonly string[];
      readonly lostCount: number;
    };

/** 什么都没丢的那一份。常量，避免每次返回都造一个新对象（与 `PERSISTED` 对称）。 */
export const LOADED: LoadOutcome = { ok: true };

/** 一次读取的产物与它的结论。 */
export interface LoadResult<T> {
  /** 回落之后的那一份 —— 应用照它起来，绝不因为读失败而白屏。 */
  readonly value: T;
  readonly outcome: LoadOutcome;
}

/** 什么都没丢：`value` 就是读出来的那一份。 */
export function loaded<T>(value: T): LoadResult<T> {
  return { value, outcome: LOADED };
}

/**
 * 丢了东西的那一份。
 *
 * `lostCount` 默认等于 `lostKeys.length`，点不出名的那几类（最近打开里被滤掉的
 * 数组元素）显式传一个数进去即可。
 *
 * **整份读不出来时必须显式传一个非零的数**（各调用点一律传 1）：
 * 「原来有几条」已经无从得知，但「报 0」在用户那里读作「什么都没丢」——
 * 那等于把一次丢失又藏了回去。传 1 是在说「**至少有一份没了**」。
 */
export function loadLost<T>(
  value: T,
  reason: LoadFailure,
  lostKeys: readonly string[] = [],
  lostCount: number = lostKeys.length,
): LoadResult<T> {
  return { value, outcome: { ok: false, reason, lostKeys, lostCount } };
}

/** 「是一个有限数值」这一关单独拆出来，好让读侧能分辨「值不对」与「根本不是值」。 */
function finiteNumber(value: unknown): number | null {
  return typeof value === 'number' && Number.isFinite(value) ? value : null;
}

// 下面这组 `parse…` 统一返回 `T | null`，**`null` = 这一项用不了**。
// 与写侧那三个 `clamp…` 的区别只有返回值：写侧拿到的是「已经确认是数」的输入，
// 读侧必须先自己确认一遍，确认不了就只能承认「这一项没了」（见 `pickField`）。

function parseTheme(value: unknown): ThemePreference | null {
  return isThemePreference(value) ? value : null;
}

function parseSidebarWidth(value: unknown): number | null {
  const width = finiteNumber(value);
  if (width === null) return null;
  return Math.min(SIDEBAR_MAX_WIDTH, Math.max(SIDEBAR_MIN_WIDTH, Math.round(width)));
}

function parseBoolean(value: unknown): boolean | null {
  return typeof value === 'boolean' ? value : null;
}

function parseSidebarView(value: unknown): SidebarView | null {
  return isSidebarView(value) ? value : null;
}

function parseSplitRatio(value: unknown): number | null {
  const ratio = finiteNumber(value);
  if (ratio === null) return null;
  return Math.min(SPLIT_RATIO_MAX, Math.max(SPLIT_RATIO_MIN, ratio));
}

function parseEditorFontSize(value: unknown): number | null {
  const size = finiteNumber(value);
  return size === null ? null : clampEditorFontSize(size);
}

function parseEditorLineHeight(value: unknown): number | null {
  const height = finiteNumber(value);
  return height === null ? null : clampEditorLineHeight(height);
}

function parseEditorFontFamily(value: unknown): string | null {
  // **净化不算「用不了」**：剥掉声明语法字符之后，那份字体意图照样生效了
  return typeof value === 'string' ? sanitizeEditorFontFamily(value) : null;
}

/**
 * 取一个字段，并把「键在、值用不了」记进 `lost`（ADR 0044）。
 *
 * ## 为什么要分「键不在」与「键在但坏了」
 *
 * 版本升级会新增字段，老数据里**必然**没有它 —— 那是正常回落，不是丢失。
 * 反过来，键在、值却不认识（手改坏、类型变了、别的工具写进来的脏数据）才是真的丢了：
 * 用户明明设过，这一次没生效，而界面不会告诉他「上次那个设置没读出来」。
 *
 * ## 为什么夹取不算丢
 *
 * 越界的值被夹回合法区间之后，**那份意图仍然生效了一部分** —— 用户调的「更宽一点」
 * 确实更宽了，只是被上限挡住。这与「整个值用不了」是两回事，
 * 按后者去报，等于每次都因为「把 9999 记成 480」提醒用户一次。
 */
function pickField<T>(
  key: string,
  raw: unknown,
  parse: (value: unknown, lostKeys: string[]) => T | null,
  fallback: T,
  lost: string[],
): T {
  if (raw === undefined) return fallback;
  const parsed = parse(raw, lost);
  if (parsed === null) {
    lost.push(key);
    return fallback;
  }
  return parsed;
}

/**
 * 从存储里读偏好。
 *
 * 逐字段校验而不是整体信任：版本升级后旧数据里可能缺少新字段、
 * 或者某个字段的类型变了。逐字段回落能让「新增一个偏好项」不影响老用户的其他设置。
 *
 * ## 为什么不只交回一个 `ViewSettings`（ADR 0044）
 *
 * 回落**必须有**：`localStorage` 在某些环境下连读都会抛，不包的话这次抛会从
 * `App` 的 `useState` 初始化器一路穿出去 —— 那是**启动白屏**（0042 顺手补的那处 try）。
 *
 * 但「回落到默认值」本身是一次**静默降级**：用户设过主题，下次启动界面是另一个主题，
 * 全程没有一个字告诉他。0042 把**写侧**做到了（`PersistOutcome`），
 * 这一份是它的读侧对偶 —— 而且丢得更狠：写失败丢一个改动，读失败丢**整份上次的状态**。
 *
 * 所以这里把**值**与**这次读丢了什么**一起交出去。说什么、怎么说不是这一层能决定的
 * （它连 UI 都没有），那是**外壳**的事（ADR 0043：内核交数据、外壳才翻成话）。
 */
export function loadSettings(storage: StorageLike | null): LoadResult<ViewSettings> {
  if (!storage) return loadLost(DEFAULT_SETTINGS, 'no-storage');

  let raw: string | null;
  try {
    raw = storage.getItem(STORAGE_KEY);
  } catch {
    // 有存储却读不出来 = 之前存过的东西这一次全读不到了 = 至少丢了一份
    return loadLost(DEFAULT_SETTINGS, 'read-failed', [], 1);
  }
  // 首次启动 / 用户清过存储：**一个键都没有**不是「丢了什么」。安静回落。
  if (raw === null || raw === '') return loaded(DEFAULT_SETTINGS);

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    // 内容坏了：回落到默认值，不抛错也不清空（用户可能只是手改错了）。
    // 整份都用不了 = **至少丢了一份**（见 `loadLost` 为什么要显式传 1）。
    return loadLost(DEFAULT_SETTINGS, 'corrupt', [], 1);
  }
  // 数组也是「键全都不在」—— 逐字段回落会把它当成一份全新的空偏好，安静得连自己都骗得过
  if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
    return loadLost(DEFAULT_SETTINGS, 'corrupt', [], 1);
  }

  const record = parsed as Record<string, unknown>;
  /** 「键在、值用不了」的字段名。一次读只收集一次（见 `pickField`）。 */
  const lost: string[] = [];
  const pick = <K extends keyof ViewSettings>(
    key: K,
    parse: (value: unknown, lostKeys: string[]) => ViewSettings[K] | null,
  ): ViewSettings[K] => pickField(key, record[key], parse, DEFAULT_SETTINGS[key], lost);

  const value: ViewSettings = {
    theme: pick('theme', parseTheme),
    sidebarWidth: pick('sidebarWidth', parseSidebarWidth),
    sidebarVisible: pick('sidebarVisible', parseBoolean),
    sidebarView: pick('sidebarView', parseSidebarView),
    outlineVisible: pick('outlineVisible', parseBoolean),
    wordWrap: pick('wordWrap', parseBoolean),
    codeLineNumbers: pick('codeLineNumbers', parseBoolean),
    rememberPositions: pick('rememberPositions', parseBoolean),
    splitRatio: pick('splitRatio', parseSplitRatio),
    restoreSession: pick('restoreSession', parseBoolean),
    keybindings: pick('keybindings', parseKeybindings),
    editorFontSize: pick('editorFontSize', parseEditorFontSize),
    editorFontFamily: pick('editorFontFamily', parseEditorFontFamily),
    editorLineHeight: pick('editorLineHeight', parseEditorLineHeight),
    locale: pick('locale', parseLocale),
    focusMode: pick('focusMode', parseBoolean),
    typewriterMode: pick('typewriterMode', parseBoolean),
  };

  return lost.length === 0 ? loaded(value) : loadLost(value, 'corrupt', lost);
}

/** 语言字段的逐项校验：不在清单里的一律回落 zh-CN（与主题同一个写法）。 */
function isSupportedLocale(value: unknown): value is SupportedLocale {
  return typeof value === 'string' && (SUPPORTED_LOCALES as readonly string[]).includes(value);
}

function parseLocale(value: unknown): SupportedLocale | null {
  return isSupportedLocale(value) ? value : null;
}

/**
 * 快捷键覆盖的逐项校验。
 *
 * 值根本不是对象 → 整项用不了（记一条）。是对象但个别命令的值不是字符串 →
 * **只丢那几条**，其余键位继续生效（与位置的「整条丢」刻意相反：
 * 一个键位坏了不该让用户其余的键位一起失效 —— 那比坏掉本身更让人火大）。
 */
function parseKeybindings(value: unknown, lost: string[]): Readonly<Record<string, string>> | null {
  if (typeof value !== 'object' || value === null) return null;
  const out: Record<string, string> = {};
  for (const [command, entry] of Object.entries(value as Record<string, unknown>)) {
    if (typeof entry === 'string') out[command] = entry;
    else lost.push(`keybindings.${command}`);
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
