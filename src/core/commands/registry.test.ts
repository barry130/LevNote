import { describe, expect, it } from 'vitest';

import { CommandRegistry } from './registry.js';
import type { Command } from './types.js';

function key(init: Partial<KeyboardEvent> & { key: string }): KeyboardEvent {
  return { ctrlKey: false, metaKey: false, altKey: false, shiftKey: false, ...init } as KeyboardEvent;
}

function command(id: string, keybinding?: string): Command {
  return { id, title: id, category: '测试', keybinding, run: () => undefined };
}

describe('注册与查找', () => {
  it('按注册顺序返回 —— 面板的排列因此不会漂移', () => {
    const registry = new CommandRegistry();
    registry.register(command('a'));
    registry.register(command('b'));
    expect(registry.list().map((item) => item.id)).toEqual(['a', 'b']);
  });

  it('重复 id 覆盖内容但保留原位置', () => {
    const registry = new CommandRegistry();
    registry.register(command('a'));
    registry.register(command('b'));
    registry.register({ ...command('a'), title: '改过的' });

    expect(registry.list().map((item) => item.id)).toEqual(['a', 'b']);
    expect(registry.find('a')?.title).toBe('改过的');
  });

  it('注销', () => {
    const registry = new CommandRegistry();
    registry.register(command('a'));
    registry.unregister('a');
    expect(registry.find('a')).toBeUndefined();
    expect(registry.list()).toEqual([]);
  });
});

describe('快捷键解析', () => {
  it('匹配到对应的命令', () => {
    const registry = new CommandRegistry();
    registry.register(command('file.save', 'Mod+S'));
    const found = registry.resolve(key({ key: 's', ctrlKey: true }), 'other', false);
    expect(found?.id).toBe('file.save');
  });

  it('没有绑定的按键返回 null，让事件继续传播', () => {
    // 返回一个假命令会让用户在某些键上打不出字
    const registry = new CommandRegistry();
    registry.register(command('file.save', 'Mod+S'));
    expect(registry.resolve(key({ key: 'a' }), 'other', false)).toBeNull();
  });

  it('焦点在输入框里时，单字符快捷键不触发', () => {
    const registry = new CommandRegistry();
    registry.register(command('mystery', 'P'));
    expect(registry.resolve(key({ key: 'p' }), 'other', true)).toBeNull();
    expect(registry.resolve(key({ key: 'p' }), 'other', false)?.id).toBe('mystery');
  });

  it('焦点在输入框里时，带修饰键的快捷键照常触发', () => {
    const registry = new CommandRegistry();
    registry.register(command('palette.open', 'Mod+P'));
    expect(registry.resolve(key({ key: 'p', ctrlKey: true }), 'other', true)?.id).toBe('palette.open');
  });

  it('先注册的优先 —— 两个命令绑同一个键时，注册顺序决定谁赢', () => {
    const registry = new CommandRegistry();
    registry.register(command('first', 'Mod+K'));
    registry.register(command('second', 'Mod+K'));
    expect(registry.resolve(key({ key: 'k', ctrlKey: true }), 'other', false)?.id).toBe('first');
  });

  it('快捷键写法错误时只忽略这一条，不抛错', () => {
    // 一条命令的快捷键写错不该让整个应用（或整个测试套件）挂掉
    const registry = new CommandRegistry();
    registry.register(command('broken', 'Ctrl+'));
    registry.register(command('fine', 'Mod+S'));
    expect(registry.resolve(key({ key: 's', ctrlKey: true }), 'other', false)?.id).toBe('fine');
  });

  it('平台差异在解析期生效', () => {
    const registry = new CommandRegistry();
    registry.register(command('file.save', 'Mod+S'));
    expect(registry.resolve(key({ key: 's', metaKey: true }), 'mac', false)?.id).toBe('file.save');
    expect(registry.resolve(key({ key: 's', metaKey: true }), 'other', false)).toBeNull();
  });
});

describe('displayBinding', () => {
  it('给出可显示的写法', () => {
    const registry = new CommandRegistry();
    registry.register(command('file.save', 'Mod+Shift+S'));
    const text = registry.displayBinding(registry.find('file.save')!, 'other', (binding) =>
      [binding.ctrl && 'Ctrl', binding.shift && 'Shift', binding.key.toUpperCase()].filter(Boolean).join('+'),
    );
    expect(text).toBe('Ctrl+Shift+S');
  });

  it('没有快捷键时返回 null', () => {
    const registry = new CommandRegistry();
    registry.register(command('plain'));
    expect(registry.displayBinding(registry.find('plain')!, 'other', () => 'x')).toBeNull();
  });
});
