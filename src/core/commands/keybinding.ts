/**
 * 快捷键的解析与匹配。
 *
 * 为什么自己写而不引第三方库（`tinykeys` / `mousetrap` / `hotkeys-js`）：
 * 我们需要的只有「解析一串声明式写法 + 和 KeyboardEvent 精确比对」这两件事，
 * 一共百来行。第三方库要么把 `Mod` 的平台语义写死成 `metaKey || ctrlKey`
 * （于是 macOS 上 `Ctrl+S` 和 `Cmd+S` 会撞车，两个命令同时触发），
 * 要么要求你在一处集中注册、与我们「命令自带快捷键」的注册表模型冲突。
 *
 * ## 平台语义：解析期就定死，匹配期只做精确比对
 *
 * `Mod` 在 macOS 上是 Cmd、在其余平台是 Ctrl。这个映射在**解析时**就按目标平台
 * 展开成具体的 ctrl/meta 布尔值，而不是留到匹配时再判断。好处是匹配函数退化成
 * 逐字段相等比较 —— 没有分支，没有「macOS 上 Ctrl 算不算 Cmd」这类边界争议，
 * 也就不会出现两个命令同时命中。
 *
 * ## 精确比对是刻意的
 *
 * `Ctrl+S` 不匹配 `Ctrl+Shift+S`。宽松匹配（只看按下了什么、不管多按了什么）
 * 会让 `Ctrl+Shift+Z`（重做）同时命中 `Ctrl+Z`（撤销），是这类实现最常见的 bug。
 */

import { t } from '../i18n/index.js';

export type Platform = 'mac' | 'other';

export interface Keybinding {
  /** Control 键被要求按下（或要求不按下） */
  readonly ctrl: boolean;
  /** Command（macOS）/ Windows 键 */
  readonly meta: boolean;
  readonly alt: boolean;
  readonly shift: boolean;
  /** 归一化后的键名，与 `KeyboardEvent.key` 的可比较形式一致 */
  readonly key: string;
}

/** 键名别名：用户和文档里习惯写短的，`KeyboardEvent.key` 给的是长的。 */
const KEY_ALIASES: Record<string, string> = {
  esc: 'escape',
  escape: 'escape',
  del: 'delete',
  delete: 'delete',
  ins: 'insert',
  space: ' ',
  spacebar: ' ',
  up: 'arrowup',
  down: 'arrowdown',
  left: 'arrowleft',
  right: 'arrowright',
  enter: 'enter',
  return: 'enter',
  tab: 'tab',
  backspace: 'backspace',
  home: 'home',
  end: 'end',
  pageup: 'pageup',
  pagedown: 'pagedown',
  comma: ',',
  period: '.',
  slash: '/',
  backslash: '\\',
  plus: '+',
  minus: '-',
  equal: '=',
};

/** 把 `event.key` 归一化成与解析结果同构的形式。 */
export function normalizeKey(key: string): string {
  if (key.length === 1) return key.toLowerCase();
  const lower = key.toLowerCase();
  return KEY_ALIASES[lower] ?? lower;
}

/** 只按下了修饰键本身（还在等真正的键名）时，`specFromEvent` 返回 null。 */
const MODIFIER_KEYS = new Set(['shift', 'control', 'alt', 'meta', 'capslock', 'fn']);

/**
 * 把「用户刚按下的这一次键」变成规范的快捷键写法（P6 第 31 项）。
 *
 * 返回 null 的两种情况：
 * - **只按了修饰键**（或 CapsLock / Fn 这类不该被录制的键）—— 录制还在进行中；
 * - 键名归一化后为空（理论上不会发生，防御一下）。
 *
 * 修饰键写法按平台归一：macOS 上 Cmd → `Mod`、Ctrl → `Ctrl`；
 * 其余平台反过来。这样存进偏好里的 spec 与内置定义同一套语言，
 * `parseKeybinding` 可以原样吃回去 —— 换平台迁移偏好时语义也对得上。
 */
export function specFromEvent(event: KeyboardEvent, platform: Platform): string | null {
  const normalized = normalizeKey(event.key);
  if (normalized === '' || MODIFIER_KEYS.has(normalized)) return null;
  // 单字符键按惯例存大写（与内置定义的 `Mod+S` 一致）；多字符键名保持小写
  const key = normalized.length === 1 ? normalized.toUpperCase() : normalized;

  const parts: string[] = [];
  if (platform === 'mac') {
    if (event.metaKey) parts.push('Mod');
    if (event.ctrlKey) parts.push('Ctrl');
  } else {
    if (event.ctrlKey) parts.push('Mod');
    if (event.metaKey) parts.push('Win');
  }
  if (event.altKey) parts.push('Alt');
  if (event.shiftKey) parts.push('Shift');
  parts.push(key);
  return parts.join('+');
}

/** 这个写法在全局监听里是**安全**的 —— 不会抢走用户的打字。 */
export function isSafeGlobalSpec(spec: string, platform: Platform): boolean {
  try {
    return isAllowedInEditable(parseKeybinding(spec, platform));
  } catch {
    return false;
  }
}

export class KeybindingSyntaxError extends Error {
  constructor(spec: string, reason: string) {
    super(t('core.keys.unparseable', { spec: JSON.stringify(spec), reason }));
    this.name = 'KeybindingSyntaxError';
  }
}

/**
 * 解析 `Mod+Shift+S` 这类写法。
 *
 * 支持的修饰键：`Mod`（主修饰键）、`Ctrl`/`Control`、`Cmd`/`Meta`/`Super`、`Alt`/`Option`、`Shift`。
 * 键名部分可以是单字符、`F1`–`F24`，或 `KEY_ALIASES` 里的别名。
 *
 * 无法解析时**抛错**而不是返回 null：快捷键写错属于开发期错误，
 * 应该在注册那一刻就炸出来，而不是变成「这个快捷键莫名没反应」。
 */
export function parseKeybinding(spec: string, platform: Platform): Keybinding {
  const parts = spec.split('+').map((part) => part.trim());
  if (parts.some((part) => part === '')) throw new KeybindingSyntaxError(spec, t('core.keys.emptyPart'));

  let ctrl = false;
  let meta = false;
  let alt = false;
  let shift = false;
  let key: string | null = null;

  for (const part of parts) {
    const lower = part.toLowerCase();
    switch (lower) {
      case 'mod':
        // 主修饰键：解析期就展开成具体平台上的那个键
        if (platform === 'mac') meta = true;
        else ctrl = true;
        continue;
      case 'ctrl':
      case 'control':
        ctrl = true;
        continue;
      case 'cmd':
      case 'command':
      case 'meta':
      case 'super':
        meta = true;
        continue;
      case 'alt':
      case 'option':
        alt = true;
        continue;
      case 'shift':
        shift = true;
        continue;
      default:
        if (key !== null) throw new KeybindingSyntaxError(spec, t('core.keys.twoKeys', { first: key, second: part }));
        key = normalizeKey(part);
    }
  }

  if (key === null) throw new KeybindingSyntaxError(spec, t('core.keys.noKey'));
  if (key === '') throw new KeybindingSyntaxError(spec, t('core.keys.emptyKey'));

  return { ctrl, meta, alt, shift, key };
}

/**
 * 精确比对。
 *
 * 注意 `event.key` 也要归一化：按下 Shift+S 时 `event.key` 是 `'S'`（大写），
 * 不归一化就会和解析出来的 `'s'` 对不上，表现为「带 Shift 的快捷键全部失效」。
 */
export function matchesKeybinding(binding: Keybinding, event: KeyboardEvent): boolean {
  return (
    event.ctrlKey === binding.ctrl &&
    event.metaKey === binding.meta &&
    event.altKey === binding.alt &&
    event.shiftKey === binding.shift &&
    normalizeKey(event.key) === binding.key
  );
}

/**
 * 生成给人看的写法。
 *
 * macOS 上按 Apple 的惯例用符号且不留 `+`（`⌘⇧S`）；其余平台用 `Ctrl+Shift+S`。
 * 中文界面里 `Cmd` 比 `⌘` 更好认，但 `⌘⇧S` 是 macOS 用户一眼就懂的写法，
 * 所以两处都按各自平台的惯例来，而不是强行统一。
 */
export function formatKeybinding(binding: Keybinding, platform: Platform): string {
  const key = displayKey(binding.key);
  if (platform === 'mac') {
    let out = '';
    if (binding.ctrl) out += '⌃';
    if (binding.alt) out += '⌥';
    if (binding.shift) out += '⇧';
    if (binding.meta) out += '⌘';
    return out + key;
  }
  const parts: string[] = [];
  if (binding.ctrl) parts.push('Ctrl');
  if (binding.meta) parts.push('Win');
  if (binding.alt) parts.push('Alt');
  if (binding.shift) parts.push('Shift');
  parts.push(key);
  return parts.join('+');
}

const DISPLAY_KEYS: Record<string, string> = {
  arrowup: '↑',
  arrowdown: '↓',
  arrowleft: '←',
  arrowright: '→',
  escape: 'Esc',
  ' ': 'Space',
  enter: 'Enter',
  backspace: 'Backspace',
  delete: 'Delete',
  pageup: 'PgUp',
  pagedown: 'PgDn',
};

function displayKey(key: string): string {
  const hit = DISPLAY_KEYS[key];
  if (hit) return hit;
  if (key.length === 1) return key.toUpperCase();
  return key.charAt(0).toUpperCase() + key.slice(1);
}

/** 从运行环境猜平台。拿不到信息时按非 macOS 处理（Windows 是首要目标平台）。 */
export function detectPlatform(source?: { platform?: string; userAgent?: string } | null): Platform {
  const probe = source ?? (typeof navigator === 'undefined' ? null : navigator);
  if (!probe) return 'other';
  const text = `${probe.platform ?? ''} ${probe.userAgent ?? ''}`.toLowerCase();
  return /mac|iphone|ipad|ipod/.test(text) ? 'mac' : 'other';
}

/**
 * 事件是否来自可输入元素。
 *
 * 规则：焦点在输入框里时，**只有带修饰键的快捷键生效**。
 * 否则用户在大纲搜索框里打 `p`，命令面板就会弹出来 —— 这是这类功能最经典的坑。
 * 反过来 `Ctrl+P` 在输入框里仍然应该工作（用户想从搜索框直接跳到命令面板）。
 */
export function isEditableTarget(target: EventTarget | null): boolean {
  if (target === null || typeof target !== 'object') return false;
  const element = target as { tagName?: unknown; isContentEditable?: unknown };
  if (element.isContentEditable === true) return true;
  const tag = typeof element.tagName === 'string' ? element.tagName.toLowerCase() : '';
  return tag === 'input' || tag === 'textarea' || tag === 'select';
}

/** 快捷键声明里是否带修饰键。 */
export function hasModifier(binding: Keybinding): boolean {
  return binding.ctrl || binding.meta || binding.alt;
}

/**
 * 焦点在输入框 / contenteditable 里时，这个快捷键还该不该触发。
 *
 * 规则是「**能打字就说明用户想打字**」：单字符键、回车、Tab、退格、删除、空格
 * 在输入框里一律让给输入行为本身。否则用户在大纲搜索框里打 `p`，命令面板就会弹出来。
 *
 * 但两类键必须放行：
 * - **带修饰键的**（`Ctrl+S`、`Ctrl+P`）—— 用户按修饰键就是明确要执行命令；
 * - **不产生文本的导航/功能键**（`Esc`、`F3`、方向键、`PageUp/Down`、`Home/End`）——
 *   它们在输入框里本来也没有「输入」含义，`Esc` 关掉查找栏是刚需。
 *
 * 白名单而不是黑名单：将来新增一个 `Ctrl+Enter` 之类的绑定，
 * 漏进黑名单的后果是「在输入框里误触发命令」，而漏进白名单的后果只是
 * 「某个键在输入框里不生效」—— 后者的危害小得多。
 */
export function isAllowedInEditable(binding: Keybinding): boolean {
  if (hasModifier(binding)) return true;
  const key = binding.key;
  if (/^f([1-9]|1[0-9]|2[0-4])$/.test(key)) return true;
  return (
    key === 'escape' ||
    key === 'arrowup' ||
    key === 'arrowdown' ||
    key === 'arrowleft' ||
    key === 'arrowright' ||
    key === 'pageup' ||
    key === 'pagedown' ||
    key === 'home' ||
    key === 'end' ||
    key === 'insert'
  );
}
