// @vitest-environment node
/**
 * 命令执行兜底层的测试（ADR 0041）。
 *
 * 钉的是「命令的 `run` 抛了以后，用户那侧会发生什么」—— 原先四个入口
 * 各自 `console.warn` 一句，用户看到的是「点了没反应」。这里把它变成可断言的：
 * 提示里要出现**哪个命令**、以及**为什么**。
 *
 * 另外一个容易被忽略的分支也钉住了：**提示自己失败**时不能再抛 ——
 * 否则调用方的 catch 会第二次进来，表现是「点了没反应」外加一条未处理 Promise。
 */

import { describe, expect, it, vi } from 'vitest';

import { reportCommandFailure, runCommandSafely } from './runner.js';
import type { Command, CommandContext } from './types.js';

interface Recorder {
  readonly notices: string[];
  readonly titles: string[];
}

function fakeContext(notify?: (message: string, title?: string) => Promise<void>): { ctx: CommandContext; rec: Recorder } {
  const notices: string[] = [];
  const titles: string[] = [];
  const ctx = {
    workspace: {},
    provider: {},
    ui: {
      notify: async (message: string, title?: string) => {
        notices.push(message);
        if (title !== undefined) titles.push(title);
        await notify?.(message, title);
      },
    },
  } as unknown as CommandContext;
  return { ctx, rec: { notices, titles } };
}

function command(id: string, title: string, run: Command['run']): Command {
  return { id, title, category: '测试', run };
}

describe('命令失败的兜底', () => {
  it('★ run 抛了：说出是哪个命令、为什么', async () => {
    const { ctx, rec } = fakeContext();
    const cmd = command('view.toggleFullscreen', '切换全屏', () => {
      throw new Error('没有用户手势');
    });

    await reportCommandFailure(cmd, new Error('没有用户手势'), ctx);

    expect(rec.notices).toHaveLength(1);
    expect(rec.notices[0]).toContain('切换全屏');
    expect(rec.notices[0]).toContain('没有用户手势');
  });

  it('★ 标题取的是当前语言下的命令名（惰性求值，不是烤死的）', async () => {
    const { ctx, rec } = fakeContext();
    const cmd = command('file.save', '保存', () => undefined);

    await reportCommandFailure(cmd, '坏了', ctx);

    expect(rec.notices[0]).toContain('保存');
  });

  it('★ 扔的不是 Error（裸字符串 / 对象）也要说成人话', async () => {
    const { ctx, rec } = fakeContext();
    const cmd = command('file.open', '打开文件', () => undefined);

    await reportCommandFailure(cmd, '磁盘没了', ctx);
    await reportCommandFailure(cmd, { code: 'NOT_FOUND', message: '没有这个文件', detail: null }, ctx);

    expect(rec.notices[0]).toContain('磁盘没了');
    expect(rec.notices[1]).toContain('没有这个文件');
  });

  it('★ 提示自己失败：不再抛（否则调用方的 catch 会二次进来）', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
    const { ctx, rec } = fakeContext(async () => {
      throw new Error('对话框也炸了');
    });
    const cmd = command('app.about', '关于', () => undefined);

    await expect(reportCommandFailure(cmd, '原始失败', ctx)).resolves.toBeUndefined();
    // 提示没送出去，但至少控制台留了一条 —— 兜底层自己要有兜底
    expect(rec.notices).toHaveLength(1);
    expect(warn).toHaveBeenCalled();
    warn.mockRestore();
  });

  it('★ runCommandSafely：同步抛与 reject 走同一条路', async () => {
    const { ctx, rec } = fakeContext();
    const sync = command('a.sync', '同步命令', () => {
      throw new Error('同步炸了');
    });
    const async_ = command('a.async', '异步命令', () => Promise.reject(new Error('异步炸了')));

    runCommandSafely(sync, ctx);
    runCommandSafely(async_, ctx);
    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(rec.notices).toHaveLength(2);
    expect(rec.notices[0]).toContain('同步炸了');
    expect(rec.notices[1]).toContain('异步炸了');
  });

  it('★ runCommandSafely：成功的命令一句话都不说', async () => {
    const { ctx, rec } = fakeContext();
    let ran = 0;
    const cmd = command('a.ok', '好命令', () => {
      ran += 1;
    });

    runCommandSafely(cmd, ctx);
    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(ran).toBe(1);
    expect(rec.notices).toHaveLength(0);
  });
});
