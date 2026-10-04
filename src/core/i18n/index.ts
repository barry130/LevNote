/**
 * i18n 取用（P3 第 17 项，ADR 0031）。
 *
 * 三个函数就是骨架的全部：`t`（查表 + 插值）、`setLocale` / `getLocale`
 * （语言状态）。回退链：**当前语言 → zh-CN（key 的定义处）→ key 本身**。
 * 最后一级返回 key 而不是空串是有意的：目录漏了 key，屏幕上出现的是一个
 * 可搜的标识（一眼定位到漏在哪），而不是一段无声的空白 —— 硬约束 2 的
 * i18n 版。
 *
 * ## 语言状态为什么是模块常量、不进设置页
 *
 * 只有 zh-CN 一份目录的今天，暴露一个「语言」偏好只会做出一个**切了没
 * 反应**的死设置 —— 那是另一种静默撒谎。设置页的接入点留给第二份目录
 * 落地的那次改动（到时 `registerBuiltInCommands` 之前先 `setLocale`，
 * 顺序在 App 启动 effect 里是现成的）。
 */

import { zhCN, type MessageKey, type SupportedLocale } from './messages.js';

/** 非默认语言允许缺 key（只翻了一部分的目录），缺的落回 zh-CN。 */
type PartialCatalog = Partial<Record<MessageKey, string>>;

const CATALOGS: Record<SupportedLocale, PartialCatalog> = {
  'zh-CN': zhCN,
};

let current: SupportedLocale = 'zh-CN';

export function getLocale(): SupportedLocale {
  return current;
}

export function setLocale(locale: SupportedLocale): void {
  current = locale;
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
