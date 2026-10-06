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
 * `settings.locale` 是真相来源；App 的 effect 在它变化时调 `setLocale`。
 * 只有 zh-CN 一份目录的年代，暴露「语言」偏好只会做出一个切了没反应的
 * 死设置 —— 那正是当初不接设置页的原因。英文目录落地后这一页翻过去了：
 * `setLocale` 现在会**通知订阅者**，因为命令标题在注册时经 `t()` 求值、
 * 已经烤进命令对象里，App 收到通知后要重新注册一遍内置命令。
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
