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
import { AppError } from '../document/types.js';
import { parseMindmap } from '../mindmap/parse.js';
import { newFileContent } from '../new-file-content.js';
import { BUILT_IN_COMMANDS, buildRecentFileCommands } from './definitions.js';
import { CommandRegistry } from './registry.js';
import type { CommandContext } from './types.js';

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

/**
 * 「起一份新内容再打开」这条编排的行为（ADR 0048）。
 *
 * 三个入口共用它，所以这里一次钉住**共有的那几条语义**——
 * 已存在绝不覆盖、写失败说出理由、失败后不继续打开、用户取消就什么都不做 ——
 * 再各自钉一条**差异**（示例文档允许已存在就直接打开）。
 * 抄开之后再逐条对，就是下一次「给 `file.new` 补了、示例文档忘了」的开始。
 */
describe('新建文件的三个入口共用一条编排', () => {
  interface Harness {
    readonly ctx: CommandContext;
    readonly written: { path: string; text: string }[];
    readonly opened: string[];
    readonly notices: string[];
    readonly picked: string[];
  }

  function harness(options: {
    readonly pick: string | null;
    readonly existing?: readonly string[];
    readonly failWrite?: boolean;
    readonly root?: string | null;
  }): Harness {
    const written: { path: string; text: string }[] = [];
    const opened: string[] = [];
    const notices: string[] = [];
    const picked: string[] = [];
    const existing = new Set(options.existing ?? []);
    const ctx = {
      provider: {
        pickSavePath: async (name: string) => {
          picked.push(name);
          return options.pick;
        },
        statFile: async (path: string) => {
          if (!existing.has(path)) throw new AppError('NOT_FOUND', '没有这个文件');
          return { path, size: 0, mtimeMs: 0, readonly: false };
        },
        writeTextFile: async (request: { path: string; text: string }) => {
          if (options.failWrite === true) throw new AppError('IO', 'EACCES: 没有写入权限');
          written.push({ path: request.path, text: request.text });
        },
      },
      workspace: {
        getSnapshot: () => ({ tree: { root: options.root ?? 'C:\\w' } }),
        open: (path: string) => {
          opened.push(path);
          return Promise.resolve();
        },
      },
      ui: {
        notify: async (message: string) => {
          notices.push(message);
        },
      },
    } as unknown as CommandContext;
    return { ctx, written, opened, notices, picked };
  }

  /**
   * 命令的 `run` 是即发即弃的（`void (async () => …)()`），这里等它走到终态。
   *
   * 等一个**宏任务**而不是轮询：整条链上的 promise 都已兑现，全排在微任务队列里，
   * 一个宏任务就是它们的终点；轮询在「本该什么都不发生」（用户取消）那个用例上会超时。
   */
  async function settle(id: string, ctx: CommandContext): Promise<void> {
    find(id).run(ctx);
    await new Promise((resolve) => setTimeout(resolve, 0));
  }

  it('★ 新建思维导图：落一张真的导图，然后打开它', async () => {
    const h = harness({ pick: 'C:\\notes\\会议记录.mindmap' });

    await settle('file.newMindmap', h.ctx);

    expect(h.written).toHaveLength(1);
    expect(h.written[0]?.path).toBe('C:\\notes\\会议记录.mindmap');
    const parsed = parseMindmap(h.written[0]?.text ?? '');
    expect(parsed.issues).toEqual([]);
    expect(parsed.document?.root.topic).toBe('会议记录');
    expect(h.opened).toEqual(['C:\\notes\\会议记录.mindmap']);
    expect(h.notices).toEqual([]);
  });

  it('另存为对话框默认给的是 `.mindmap` 后缀（不给默认名，用户无从知道后缀）', async () => {
    const h = harness({ pick: 'C:\\notes\\会议记录.mindmap' });

    await settle('file.newMindmap', h.ctx);

    expect(h.picked).toEqual(['未命名.mindmap']);
  });

  it('★ 已存在：说明并停手 —— 不写、不打开', async () => {
    const h = harness({ pick: 'C:\\notes\\会议记录.mindmap', existing: ['C:\\notes\\会议记录.mindmap'] });

    await settle('file.newMindmap', h.ctx);

    expect(h.written).toEqual([]);
    expect(h.opened).toEqual([]);
    expect(h.notices).toHaveLength(1);
    expect(h.notices[0]).toContain('没有新建');
  });

  it('★ 写失败：说出原因，且不继续打开', async () => {
    const h = harness({ pick: 'C:\\notes\\会议记录.mindmap', failWrite: true });

    await settle('file.newMindmap', h.ctx);

    expect(h.opened).toEqual([]);
    expect(h.notices).toHaveLength(1);
    expect(h.notices[0]).toContain('EACCES');
  });

  it('用户在对话框里取消：什么都不发生', async () => {
    const h = harness({ pick: null });

    await settle('file.newMindmap', h.ctx);

    expect(h.written).toEqual([]);
    expect(h.opened).toEqual([]);
    expect(h.notices).toEqual([]);
  });

  it('★ 普通「新建文件」选了 `.mindmap` 也落同一份模板 —— 格式判据只有一条出口', async () => {
    const h = harness({ pick: 'C:\\notes\\随手一张.mindmap' });

    await settle('file.new', h.ctx);

    expect(h.written[0]?.text).toBe(newFileContent('C:\\notes\\随手一张.mindmap'));
    expect(h.picked).toEqual(['未命名.md']);
  });

  it('普通「新建文件」写失败的说法里也有原因（降级要说出理由）', async () => {
    const h = harness({ pick: 'C:\\notes\\a.md', failWrite: true });

    await settle('file.new', h.ctx);

    expect(h.notices).toHaveLength(1);
    expect(h.notices[0]).toContain('EACCES');
  });

  it('示例文档的例外语义没被共用编排改掉：已存在时不打扰，直接打开', async () => {
    const h = harness({ pick: 'C:\\w\\轻笺示例.md', existing: ['C:\\w\\轻笺示例.md'], root: 'C:\\w' });

    await settle('help.sampleDocument', h.ctx);

    expect(h.written).toEqual([]);
    expect(h.opened).toEqual(['C:\\w\\轻笺示例.md']);
    expect(h.notices).toEqual([]);
  });
});
