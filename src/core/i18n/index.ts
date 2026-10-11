/**
 * i18n 取用（P3 第 17 项，ADR 0031）。
 *
 * 三个函数就是骨架的全部：`t`（查表 + 插值）、`setLocale` / `getLocale`
 * （语言状态）。回退链：**当前语言 → zh-CN（key 的定义处）→ key 本身**。
 * 最后一级返回 key 而不是空串是有意的：目录漏了 key，屏幕上出现的是一个
 * 可搜的标识（一眼定位到漏在哪），而不是一段无声的空白 —— 硬约束 2 的
 * i18n 版。
 *
 * ## 语言状态与设置页（2026-10-07 起接入）
 *
 * `settings.locale` 是真相来源；App 的 effect 在它变化时调 `setLocale`，
 * 再 bump 一次自己的 `localeTick` 让整棵外壳重渲染一遍。
 * 只有 zh-CN 一份目录的年代，暴露「语言」偏好只会做出一个切了没反应的
 * 死设置 —— 那正是当初不接设置页的原因。
 *
 * ## 订阅者与「重注册」都已不需要（2026-10-09）
 *
 * `setLocale` 仍会通知 `subscribeLocale` 的订阅者，但**目前的消费方一个都不靠它**：
 * 命令面的 `title` / `category` 现在是**读的那一刻才查表**的 getter
 * （`core/commands/definitions.ts` 的 `builtInCommand`），刷新文案只需要重渲染。
 * 在此之前这里挂着一个反例 —— 注释说「App 收到通知后要重新注册一遍内置命令」，
 * 而命令对象是模块级常量、`title` 早被 `t()` 烤死，重注册从未生效过
 * （#TASK-I18N-CMD-LIVE-01）。
 *
 * ## 一条「还没翻的文案」：ADR 0047
 *
 * `LocalizedText` / `msg()` / `textOf()` 是 ADR 0047 的全部内容，理由是
 * **这一族残留的病根只有一个形状**：`t()` 的返回值被烤进某个字符串，
 * 而那个字符串会**跨一次渲染活下来**（进 state、存进 store 字段、传进回调）——
 * 于是切语言时它在原语言里不动。
 *
 * 与 ADR 0031 复评记录三的命令面（读的时候才查表的 getter）、`Notice`
 * 族的结构化联合、`localizedError` 的 getter 是**同一族修法的三种落法**，
 * 差别只在「消费方要不要改字段类型」。ADR 0047 把这一族挑明并钉住形状：
 *
 * - `msg(key, params)` 造的**不是句子**，是「**将来要翻哪一条**」。
 *   在造它的那一刻调 `t()` 就等于把语言烤死（ADR 0031 踩过：`BUILT_IN_COMMANDS`）。
 * - `textOf(line)` **只在渲染的那一刻调**。看到 `textOf` 出现在 `useState`
 *   的初始值、effect 依赖、事件处理器里，就是同一个 bug 换了个地方。
 * - 与 `Notice` 的 key 变体**同形**（字段名也用 `params`）——
 *   `renderNotice` 能直接吃它，于是「`{ text: 已翻好 }`」那道品牌门
 *   （`core/document/store.ts` 的 `legacyText()`）在 images.ts 改回 key 后
 *   整个删掉了。
 */

import { en, zhCN, type MessageKey, type SupportedLocale } from './messages.js';

// 调用方（MenuBar 之类的渲染时取文案的组件）需要 key 类型
export type { MessageKey, SupportedLocale };

/** 非默认语言允许缺 key（只翻了一部分的目录），缺的落回 zh-CN。 */
type PartialCatalog = Partial<Record<MessageKey, string>>;

const CATALOGS: Record<SupportedLocale, PartialCatalog> = {
  'zh-CN': zhCN,
  // en 目录的完整对应由类型钉住（Record<MessageKey, string>），运行时不再校验
  en,
};

let current: SupportedLocale = 'zh-CN';
const listeners = new Set<() => void>();

export function getLocale(): SupportedLocale {
  return current;
}

/** 切换语言并通知订阅者。重复设置同一个值时安静返回（不惊动界面）。 */
export function setLocale(locale: SupportedLocale): void {
  if (current === locale) return;
  current = locale;
  for (const listener of listeners) listener();
}

/** 订阅语言变化（返回解绑函数）。App 在解绑时不需要做什么特殊清理。 */
export function subscribeLocale(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

/**
 * 模板插值：`{name}` 占位符换成参数，**没有对应参数的占位符原样保留**。
 *
 * 保留而不是删掉：漏传参数是调用点的 bug，一段写着 `{name}` 的文案
 * 立刻可见、可搜；悄悄删掉就没人知道少了东西。
 */
export function interpolate(template: string, params: Readonly<Record<string, string | number>>): string {
  return template.replace(/\{(\w+)\}/g, (match, name: string) => {
    const value = params[name];
    return value === undefined ? match : String(value);
  });
}

/** 查文案。key 类型是 `MessageKey`，写错 key 编译期就会报 —— 不要用 as 绕过去。 */
export function t(key: MessageKey, params?: Readonly<Record<string, string | number>>): string {
  const template = CATALOGS[current][key] ?? zhCN[key] ?? key;
  return params === undefined ? template : interpolate(template, params);
}

/**
 * 一条**还没翻成句子**的文案（ADR 0047）。
 *
 * 与 `Notice` 的 key 变体同形（字段名都是 `params`），所以 `renderNotice`
 * 与 `msg()` 的结果可以互换 —— 有意如此，两条同族的东西不许长得不一样。
 * 与 ADR 0046 的 `HealthLine` 同样是 `{ key, params }`，那边已改成这里的别名。
 *
 * **什么时候该用它**：你算出来的这句话**不是当场显示的**，它会被存起来
 * （进 state / 存进 store / 传给回调），而那种字符串会**跨一次渲染活下来**。
 * 当场显示的一律直接 `t()`。
 */
export interface LocalizedText {
  readonly key: MessageKey;
  readonly params: Readonly<Record<string, string | number>>;
}

/**
 * 造一条「还没翻的文案」。`t()` 的孪生：那个查表，这个只记下**要查哪一条**。
 *
 * `params` 留空引用而不是新造一个对象 —— 这条数据会进 state、会触发相等比较，
 * 而 `{}` 每次都是新引用（`resolveIssue` 对「内容一字未改」就靠引用相等，
 * 见 ADR 0046）。
 */
export function msg(key: MessageKey, params: Readonly<Record<string, string | number>> = {}): LocalizedText {
  return { key, params };
}

/**
 * 把一条「还没翻的文案」翻成句子 —— **只在渲染的那一刻调**。
 *
 * 判据只有一句：**这句译文会不会跨过下一次 `setLocale`？**
 * 会（存进了 state / store）就用 `msg()` + `textOf()`；不会就直接 `t()`。
 */
export function textOf(line: LocalizedText): string {
  return t(line.key, line.params);
}
