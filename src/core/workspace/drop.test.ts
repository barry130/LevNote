// @vitest-environment node
/**
 * 拖入多个文件的打开与汇总（ADR 0041）。
 *
 * 钉的是两件事：**每一个都被试过**（不是只开第一个），以及**失败要说清是哪几个**
 * （而不是「拖了三个进来它只认一个」）。
 *
 * 汇总文案本身是纯函数，所以这里直接断言字符串 —— 它进的是 `provider.notify`，
 * 而外壳那一层（把名单交给对话框）由 `App.test.tsx` 的拖放用例覆盖。
 */

import { describe, expect, it } from 'vitest';

import { describeDropOutcome, openDroppedPaths, type DropOutcome } from './drop.js';
import { WorkspaceStore } from './store.js';

// 复用工作区测试里那份假来源：这里只需要「哪些文件存在」这一个维度
class FakeProvider {
  readonly kind = 'desktop' as const;
  readonly files = new Map<string, string>();
  readonly opened: string[] = [];

  put(path: string, text: string): void {
    this.files.set(path, text);
  }

  async readTextFile(path: string) {
    const text = this.files.get(path);
    if (text === undefined) throw { code: 'NOT_FOUND', message: `没有这个文件：${path}`, detail: null };
    return {
      path,
      text,
      encoding: 'UTF-8',
      bom: 'none' as const,
      lineEnding: 'lf' as const,
      size: text.length,
      mtimeMs: 1,
      readonly: false,
      lossy: false,
    };
  }

  async statFile(path: string) {
    const text = this.files.get(path);
    if (text === undefined) throw { code: 'NOT_FOUND', message: `没有这个文件：${path}`, detail: null };
    return { path, size: text.length, mtimeMs: 1, readonly: false, isDir: false };
  }

  async sniffEncoding(): Promise<string> {
    return 'UTF-8';
  }

  async listDir(): Promise<[]> {
    return [];
  }

  async writeTextFile(): Promise<never> {
    throw new Error('本套不测写盘');
  }

  async readBinaryFile(): Promise<ArrayBuffer> {
    return new ArrayBuffer(0);
  }

  async writeBinaryFile(): Promise<never> {
    throw new Error('本套不测写盘');
  }

  async runtimeInfo() {
    return { name: 'levnote', version: 'test', os: 'win32', arch: 'x64', nowMs: 1 };
  }

  async pickOpenPath(): Promise<string | null> {
    return null;
  }

  async pickSavePath(): Promise<string | null> {
    return null;
  }

  async pickDirectoryPath(): Promise<string | null> {
    return null;
  }

  async notify(): Promise<void> {
    /* 不需要弹窗 */
  }
}

function seeded(paths: readonly string[]): { provider: FakeProvider; workspace: WorkspaceStore } {
  const provider = new FakeProvider();
  for (const path of paths) provider.put(path, 'x');
  return { provider, workspace: new WorkspaceStore(provider) };
}

describe('拖入多个文件', () => {
  it('★ 三个都打开 —— 原先只有第一个会被试', async () => {
    const { workspace } = seeded(['C:\\w\\a.md', 'C:\\w\\b.md', 'C:\\w\\c.md']);

    const outcome = await openDroppedPaths(['C:\\w\\a.md', 'C:\\w\\b.md', 'C:\\w\\c.md'], workspace);

    expect(outcome).toEqual({ opened: 3, failed: [] });
    expect(workspace.getSnapshot().tabs).toHaveLength(3);
    // 全部成功 = 一句话都不说（新标签页自己会说话）
    expect(describeDropOutcome(outcome)).toBeNull();
  });

  it('★ 头一个坏了不连累后面：逐个试，失败收进名单', async () => {
    const { workspace } = seeded(['C:\\w\\b.md', 'C:\\w\\c.md']);

    const outcome = await openDroppedPaths(['C:\\w\\missing.md', 'C:\\w\\b.md', 'C:\\w\\c.md'], workspace);

    expect(outcome.opened).toBe(2);
    expect(outcome.failed).toEqual(['C:\\w\\missing.md']);
    expect(workspace.getSnapshot().tabs).toHaveLength(2);
  });

  it('★ 已经开着的那个算成功（去重是激活，不是失败）', async () => {
    const { workspace } = seeded(['C:\\w\\a.md', 'C:\\w\\b.md']);
    await workspace.open('C:\\w\\a.md');

    const outcome = await openDroppedPaths(['C:\\w\\a.md', 'C:\\w\\b.md'], workspace);

    expect(outcome).toEqual({ opened: 2, failed: [] });
    expect(workspace.getSnapshot().tabs).toHaveLength(2);
  });

  it('★ 部分失败：名单里是文件名，一句说完', async () => {
    const { workspace } = seeded(['C:\\w\\b.md']);

    const outcome = await openDroppedPaths(['C:\\w\\gone.md', 'C:\\w\\b.md', 'C:\\w\\lost.md'], workspace);
    const message = describeDropOutcome(outcome);

    expect(message).not.toBeNull();
    expect(message?.body).toContain('gone.md');
    expect(message?.body).toContain('lost.md');
    // 成功的那个不该出现在名单里
    expect(message?.body).not.toContain('b.md');
  });

  it('★ 全失败：说的是「都没能打开」，不是「有几个没打开」', async () => {
    const { workspace } = seeded([]);

    const outcome = await openDroppedPaths(['C:\\w\\x.md', 'C:\\w\\y.md'], workspace);
    const message = describeDropOutcome(outcome);

    expect(outcome.opened).toBe(0);
    expect(message?.body).toContain('都没能打开');
    // 与「部分失败」那句不同（那句是「这 N 个没能打开：…」），
    // 「整个拖放都失败了」不该听起来像小问题
    expect(message?.body).not.toContain('个没能打开');
  });

  it('★ 空名单什么都不做（拖空 / payload 没有 paths）', async () => {
    const { workspace } = seeded([]);

    const outcome = await openDroppedPaths([], workspace);

    expect(outcome).toEqual({ opened: 0, failed: [] });
    expect(describeDropOutcome(outcome)).toBeNull();
  });

  it('★ `open` 直接抛（不是返回 false）也算失败，不中断后面的', async () => {
    const provider = new FakeProvider();
    provider.put('C:\\w\\b.md', 'x');
    const workspace = new WorkspaceStore(provider);
    // 让第二个路径的读取抛出来，模拟 provider 不按约定收口
    const original = provider.readTextFile.bind(provider);
    provider.readTextFile = async (path: string) => {
      if (path === 'C:\\w\\boom.md') throw new Error('权限炸了');
      return original(path);
    };

    const outcome = await openDroppedPaths(['C:\\w\\boom.md', 'C:\\w\\b.md'], workspace);

    expect(outcome.failed).toEqual(['C:\\w\\boom.md']);
    expect(outcome.opened).toBe(1);
  });

  it('汇总函数不碰工作区：给定结果就能说出话（纯函数）', () => {
    const outcome: DropOutcome = { opened: 1, failed: ['C:\\w\\a.md', 'C:\\w\\b.md'] };
    const message = describeDropOutcome(outcome);
    expect(message?.body).toBe('这 2 个没能打开：a.md、b.md');
  });
});
