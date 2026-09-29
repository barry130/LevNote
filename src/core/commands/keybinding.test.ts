import { describe, expect, it } from 'vitest';

import {
  detectPlatform,
  formatKeybinding,
  hasModifier,
  isAllowedInEditable,
  isEditableTarget,
  matchesKeybinding,
  normalizeKey,
  parseKeybinding,
} from './keybinding.js';

/** 造一个只带这几个字段的假事件 —— 匹配逻辑只读这五个属性。 */
function key(init: Partial<KeyboardEvent> & { key: string }): KeyboardEvent {
  return {
    ctrlKey: false,
    metaKey: false,
    altKey: false,
    shiftKey: false,
    ...init,
  } as KeyboardEvent;
}

describe('parseKeybinding', () => {
  it('Mod 按平台展开成具体的键', () => {
    // 这条是整个模块的核心：展开发生在解析期，匹配期只做精确比对，
    // 于是 macOS 上 `Ctrl+S` 与 `Cmd+S` 不会撞车
    expect(parseKeybinding('Mod+S', 'other')).toMatchObject({ ctrl: true, meta: false, key: 's' });
    expect(parseKeybinding('Mod+S', 'mac')).toMatchObject({ ctrl: false, meta: true, key: 's' });
  });

  it('显式写 Ctrl / Cmd 时不看平台', () => {
    expect(parseKeybinding('Ctrl+S', 'mac')).toMatchObject({ ctrl: true, meta: false });
    expect(parseKeybinding('Cmd+S', 'other')).toMatchObject({ ctrl: false, meta: true });
  });

  it('修饰键组合', () => {
    expect(parseKeybinding('Ctrl+Shift+Alt+ArrowDown', 'other')).toMatchObject({
      ctrl: true,
      shift: true,
      alt: true,
      key: 'arrowdown',
    });
  });

  it('键名归一化：单字符小写、别名展开', () => {
    expect(parseKeybinding('Ctrl+P', 'other').key).toBe('p');
    expect(parseKeybinding('Escape', 'other').key).toBe('escape');
    expect(parseKeybinding('Esc', 'other').key).toBe('escape');
    expect(parseKeybinding('F3', 'other').key).toBe('f3');
    expect(parseKeybinding('Ctrl+Space', 'other').key).toBe(' ');
  });

  it('写错就抛错 —— 开发期错误该在注册那一刻炸出来', () => {
    expect(() => parseKeybinding('Ctrl+', 'other')).toThrow(/快捷键写法无法解析/);
    expect(() => parseKeybinding('Ctrl', 'other')).toThrow(/缺少键名/);
    expect(() => parseKeybinding('Ctrl+A+B', 'other')).toThrow(/两个键名/);
  });
});

describe('matchesKeybinding', () => {
  it('精确比对：Ctrl+S 不匹配 Ctrl+Shift+S', () => {
    const binding = parseKeybinding('Mod+S', 'other');
    expect(matchesKeybinding(binding, key({ key: 's', ctrlKey: true }))).toBe(true);
    // 宽松匹配会让「重做」同时命中「撤销」，是这类实现最经典的 bug
    expect(matchesKeybinding(binding, key({ key: 'S', ctrlKey: true, shiftKey: true }))).toBe(false);
  });

  it('按下 Shift 时 event.key 是大写，也要能匹配上', () => {
    // 不归一化的话，所有带 Shift 的快捷键会集体失效
    const binding = parseKeybinding('Mod+Shift+S', 'other');
    expect(matchesKeybinding(binding, key({ key: 'S', ctrlKey: true, shiftKey: true }))).toBe(true);
  });

  it('macOS 上 Cmd+S 不匹配 Ctrl+S', () => {
    const binding = parseKeybinding('Mod+S', 'mac');
    expect(matchesKeybinding(binding, key({ key: 's', metaKey: true }))).toBe(true);
    expect(matchesKeybinding(binding, key({ key: 's', ctrlKey: true }))).toBe(false);
  });

  it('多按了修饰键就不匹配', () => {
    const binding = parseKeybinding('Mod+O', 'other');
    expect(matchesKeybinding(binding, key({ key: 'o', ctrlKey: true, altKey: true }))).toBe(false);
  });
});

describe('formatKeybinding', () => {
  it('非 macOS 用 Ctrl+Shift+S 的写法', () => {
    expect(formatKeybinding(parseKeybinding('Mod+Shift+S', 'other'), 'other')).toBe('Ctrl+Shift+S');
    expect(formatKeybinding(parseKeybinding('Escape', 'other'), 'other')).toBe('Esc');
    expect(formatKeybinding(parseKeybinding('Ctrl+Tab', 'other'), 'other')).toBe('Ctrl+Tab');
  });

  it('macOS 用 Apple 惯例的符号且不留加号', () => {
    expect(formatKeybinding(parseKeybinding('Mod+Shift+S', 'mac'), 'mac')).toBe('⇧⌘S');
  });
});

describe('isAllowedInEditable —— 输入框里哪些快捷键还该生效', () => {
  it('带修饰键的放行', () => {
    expect(isAllowedInEditable(parseKeybinding('Mod+S', 'other'))).toBe(true);
    expect(hasModifier(parseKeybinding('Mod+S', 'other'))).toBe(true);
  });

  it('单字符键在输入框里让给输入行为', () => {
    // 不这样做，用户在大纲搜索框里打 `p`，命令面板就会弹出来
    expect(isAllowedInEditable(parseKeybinding('P', 'other'))).toBe(false);
    expect(isAllowedInEditable(parseKeybinding('Enter', 'other'))).toBe(false);
  });

  it('不产生文本的导航 / 功能键放行', () => {
    expect(isAllowedInEditable(parseKeybinding('Escape', 'other'))).toBe(true);
    expect(isAllowedInEditable(parseKeybinding('F3', 'other'))).toBe(true);
    expect(isAllowedInEditable(parseKeybinding('ArrowDown', 'other'))).toBe(true);
  });
});

describe('normalizeKey / detectPlatform / isEditableTarget', () => {
  it('归一化', () => {
    expect(normalizeKey('S')).toBe('s');
    expect(normalizeKey('Esc')).toBe('escape');
    expect(normalizeKey('ArrowUp')).toBe('arrowup');
  });

  it('平台探测', () => {
    expect(detectPlatform({ platform: 'MacIntel' })).toBe('mac');
    expect(detectPlatform({ userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0)' })).toBe('mac');
    expect(detectPlatform({ platform: 'Win32' })).toBe('other');
    expect(detectPlatform(null)).toBe('other');
  });

  it('识别可输入元素', () => {
    // 只带这几个属性的假元素即可 —— 判断逻辑只读 tagName / isContentEditable
    const element = (shape: object) => shape as unknown as EventTarget;
    expect(isEditableTarget(element({ tagName: 'INPUT' }))).toBe(true);
    expect(isEditableTarget(element({ tagName: 'TEXTAREA' }))).toBe(true);
    expect(isEditableTarget(element({ isContentEditable: true }))).toBe(true);
    expect(isEditableTarget(element({ tagName: 'DIV' }))).toBe(false);
    expect(isEditableTarget(null)).toBe(false);
  });
});
