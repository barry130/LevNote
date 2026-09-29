import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { EditorAdapter } from './types.js';

/**
 * 这里守的是整个应用最硬的一条不变量：**任何文件都必须能被打开**。
 * 打不开文件是比「用错内核」严重得多的问题 —— 前者用户无路可走，
 * 后者至少内容还在、还能看到、还能另存。
 *
 * 注册表是模块级单例，用例之间必须互不影响，所以每个用例都动态导入一份新的
 * （`vi.resetModules()` 把模块缓存清掉）。
 */

type Registry = typeof import('./registry.js');

async function freshRegistry(): Promise<Registry> {
  vi.resetModules();
  return import('./registry.js');
}

function stub(
  id: EditorAdapter['id'],
  matches: (path: string) => boolean,
  canHandle?: EditorAdapter['canHandle'],
): EditorAdapter {
  return {
    id,
    label: id,
    matches,
    ...(canHandle ? { canHandle } : {}),
    Component: () => null,
  };
}

/** 兜底内核：永不匹配任何路径，由调用方显式传入。 */
const fallback = stub('plaintext', () => false);

/** 用来在断言失败时看清「被选中」的是哪个内核。 */
const idOf = (adapter: EditorAdapter): string => adapter.id;

describe('适配器注册表', () => {
  beforeEach(() => {
    vi.resetModules();
  });

  it('没有任何候选时回落到兜底内核', async () => {
    const { resolveAdapter } = await freshRegistry();
    expect(idOf(resolveAdapter('/tmp/a.unknown', fallback))).toBe('plaintext');
  });

  it('按扩展名匹配到专门化内核', async () => {
    const { registerAdapter, resolveAdapter } = await freshRegistry();
    registerAdapter(stub('markdown', (path) => path.endsWith('.md')));
    expect(idOf(resolveAdapter('/tmp/a.md', fallback))).toBe('markdown');
  });

  it('canHandle 否决后继续找下一个候选，而不是直接跳到兜底', async () => {
    const { registerAdapter, resolveAdapter } = await freshRegistry();
    // 后注册的优先。第一个 .md 内核否决内容，于是第二个（code）应该被选中
    registerAdapter(stub('markdown', (path) => path.endsWith('.md'), () => true));
    registerAdapter(stub('code', (path) => path.endsWith('.md'), () => false));

    expect(idOf(resolveAdapter('/tmp/a.md', fallback, '内容'))).toBe('markdown');
  });

  it('所有候选都否决时落到兜底内核 —— 文件仍然打开了', async () => {
    const { registerAdapter, resolveAdapter } = await freshRegistry();
    registerAdapter(stub('markdown', (path) => path.endsWith('.md'), () => false));
    expect(idOf(resolveAdapter('/tmp/a.md', fallback, '内容'))).toBe('plaintext');
  });

  it('不传内容时跳过 canHandle（此时还没有内容可判断）', async () => {
    const { registerAdapter, resolveAdapter } = await freshRegistry();
    const canHandle = vi.fn(() => false);
    registerAdapter(stub('markdown', (path) => path.endsWith('.md'), canHandle));

    expect(idOf(resolveAdapter('/tmp/a.md', fallback))).toBe('markdown');
    expect(canHandle).not.toHaveBeenCalled();
  });

  it('canHandle 抛错时不把整个打开流程带崩 —— 退到兜底内核', async () => {
    const { registerAdapter, resolveAdapter } = await freshRegistry();
    // 这条路径会刻意打一条 warn，测试里不想让它把输出刷满
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
    registerAdapter(
      stub('markdown', (path) => path.endsWith('.md'), () => {
        throw new Error('判定逻辑写挂了');
      }),
    );

    expect(() => resolveAdapter('/tmp/a.md', fallback, '内容')).not.toThrow();
    expect(idOf(resolveAdapter('/tmp/a.md', fallback, '内容'))).toBe('plaintext');
    expect(warn).toHaveBeenCalled();
    warn.mockRestore();
  });

  it('重复注册同一个 id 会顶到最前，不会出现两份', async () => {
    const { listAdapters, registerAdapter } = await freshRegistry();
    registerAdapter(stub('markdown', () => true));
    registerAdapter(stub('code', () => true));
    registerAdapter(stub('markdown', () => false));

    expect(listAdapters().map(idOf)).toEqual(['markdown', 'code']);
  });

  it('兜底内核不出现在注册表里 —— 它不可能被遮蔽', async () => {
    const { listAdapters, registerAdapter } = await freshRegistry();
    registerAdapter(stub('markdown', () => true));
    // plaintext 从未被注册，所以注册表里只有 markdown
    expect(listAdapters().map(idOf)).not.toContain('plaintext');
  });
});
