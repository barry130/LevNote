// @vitest-environment jsdom
/**
 * 命令面文案的**求值时机**（#TASK-I18N-CMD-LIVE-01）。
 *
 * 这里钉的不是「字典里有译文」（那是 `i18n.test.ts` 的事），而是
 * **命令对象读出来的字符串真的跟着语言走**：
 *
 * 缺陷的形状是「`title: t(...)` 在模块加载时烤进模块级常量」——
 * 于是切语言后重注册完全无效，而单测全绿（没人测过「切完之后读到的是什么」）。
 * 所以这一条必须在**读的那一刻**取证，并且**两向都验**（切过去要变，切回来要变）。
 */

import { afterEach, describe, expect, it } from 'vitest';

import { setLocale, t } from '../i18n/index.js';
import { BUILT_IN_COMMANDS, buildRecentFileCommands } from './definitions.js';
import { CommandRegistry } from './registry.js';

function find(id: string) {
  const command = BUILT_IN_COMMANDS.find((item) => item.id === id);
  if (command === undefined) throw new Error(`没有这条内置命令：${id}`);
  return command;
}

afterEach(() => {
  // 语言是模块级状态，用例之间会互相影响 —— 复位成默认值
  setLocale('zh-CN');
});

describe('内置命令的文案随语言实时变化', () => {
  it('title / category 在切换语言后读到的是新语言；切回来也变', () => {
    const open = find('file.open');

    expect(open.title).toBe('打开文件');
    expect(open.category).toBe('文件');

    setLocale('en');
    expect(open.title).toBe('Open File…');
    expect(open.category).toBe('File');
    // 与目录现查的值一致 —— 证明它走的确实是查表，而不是另一份副本
    expect(open.title).toBe(t('command.file.open.title'));

    setLocale('zh-CN');
    expect(open.title).toBe('打开文件');
    expect(open.category).toBe('文件');
  });

  it('机械生成的命令（格式 / 段落 / 主题 / 编码）同样跟着走', () => {
    // 这几组是 `.map()` 出来的，最容易被「模块级常量」那条路径漏掉。
    // 编码命令的标题来源不同（`encodingLabel` 而不是目录 key）—— 它**刻意**
    // 不参与翻译（纯技术名 UTF-8 / GBK 换语言也不该变），所以另测。
    const mechanical = ['format.bold', 'paragraph.heading1', 'theme.ink'];
    const zhTitles = mechanical.map((id) => [id, find(id).title] as const);

    setLocale('en');
    for (const [id, zh] of zhTitles) {
      expect(find(id).title, id).not.toBe(zh);
    }

    setLocale('zh-CN');
    for (const [id, zh] of zhTitles) {
      expect(find(id).title, id).toBe(zh);
    }
  });

  it('所有内置命令的 title 都不是「烤死的」—— 切到 English 后没有一条停在中文', () => {
    setLocale('en');
    // 反例判据：把 title 改回注册时求值，这一条会**大批**失败
    const stuck = BUILT_IN_COMMANDS.filter((command) => /[\u4e00-\u9fff]/.test(command.title));
    expect(stuck.map((command) => `${command.id}=${command.title}`)).toEqual([]);
  });

  it('数据类命令（最近打开的文件）的标题来自数据，不参与翻译', () => {
    const recent = buildRecentFileCommands(['C:/w/轻笺示例.md'], () => undefined)[0];
    if (recent === undefined) throw new Error('最近文件命令没生成');
    expect(recent.title).toBe('轻笺示例.md — C:/w');
    setLocale('en');
    // 文件名是数据 —— 换成英文反而是错的
    expect(recent.title).toBe('轻笺示例.md — C:/w');
  });
});

describe('只改文案求值时机，命令的 id / 快捷键 / 可用性一个字不动', () => {
  it('id / 快捷键 / keywords / enabled 在两种语言下逐条同一，且是同一批对象', () => {
    const snapshot = () =>
      BUILT_IN_COMMANDS.map((command) => [
        command.id,
        command.keybinding,
        command.keywords,
        command.enabled,
        command.dynamic,
      ]);

    const zh = snapshot();
    const before = [...BUILT_IN_COMMANDS];
    setLocale('en');
    expect(snapshot()).toEqual(zh);

    // 逐条**引用同一**：切语言时谁要是把命令重新造一遍（例如又跑一次
    // `.map()`），这条立刻红。它也顺带覆盖了 `run` —— 同一个对象上
    // `run` 不可能被换掉，不必单独比函数。
    expect(BUILT_IN_COMMANDS.map((command, index) => command === before[index])).toEqual(
      BUILT_IN_COMMANDS.map(() => true),
    );
  });

  it('注册表解析按键的结果不变（谁认领 Ctrl+O 与语言无关）', () => {
    const registry = new CommandRegistry();
    for (const command of BUILT_IN_COMMANDS) registry.register(command);

    const event = new KeyboardEvent('keydown', { key: 'o', ctrlKey: true, bubbles: true, cancelable: true });
    const zh = registry.resolve(event, 'other', false)?.id;
    const zhBinding = registry.displayBinding(find('file.open'), 'other', (binding) => binding.key);

    setLocale('en');
    expect(registry.resolve(event, 'other', false)?.id).toBe(zh);
    expect(registry.displayBinding(find('file.open'), 'other', (binding) => binding.key)).toBe(zhBinding);
    expect(zh).toBe('file.open');
  });

});

describe('命令对象上不留构造用的 key', () => {
  it('titleKey / categoryKey 只进不出 —— 消费方看到的是普通命令', () => {
    const open = find('file.open');
    expect(Object.keys(open)).not.toContain('titleKey');
    expect(Object.keys(open)).not.toContain('categoryKey');
    // 展开 `{...command}`（外壳里真实存在这种写法）不该带出内部字段
    expect(Object.keys({ ...open }).sort()).toEqual(['category', 'id', 'keybinding', 'keywords', 'run', 'title']);
  });
});
