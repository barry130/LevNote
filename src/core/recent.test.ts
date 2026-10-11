// @vitest-environment node
import { describe, expect, it } from 'vitest';

import {
  clearRecentFiles,
  loadRecentFiles,
  RECENT_FILES_MAX,
  recordRecentList,
  saveRecentFiles,
} from './recent.js';
import type { StorageLike } from './settings.js';

function memoryStorage(seed?: string): StorageLike & { value: string | null } {
  const box = {
    value: seed ?? null,
    getItem(): string | null {
      return box.value;
    },
    setItem(_key: string, value: string): void {
      box.value = value;
    },
  };
  return box;
}

describe('recordRecentList', () => {
  it('新路径排到最前', () => {
    expect([...recordRecentList(['b.md', 'a.md'], 'c.md')]).toEqual(['c.md', 'b.md', 'a.md']);
  });

  it('已存在的路径移到最前，不重复', () => {
    expect([...recordRecentList(['b.md', 'a.md', 'c.md'], 'a.md')]).toEqual(['a.md', 'b.md', 'c.md']);
  });

  it('记录已在最前时返回同一个引用 —— 调用方据此免掉一次落盘', () => {
    const list = ['a.md', 'b.md'];
    expect(recordRecentList(list, 'a.md')).toBe(list);
  });

  it(`超出上界（${RECENT_FILES_MAX}）时最久的让位`, () => {
    const seed = Array.from({ length: RECENT_FILES_MAX }, (_, i) => `f${i}.md`);
    const next = recordRecentList(seed, 'new.md');
    expect(next).toHaveLength(RECENT_FILES_MAX);
    expect(next[0]).toBe('new.md');
    expect([...next]).not.toContain(`f${RECENT_FILES_MAX - 1}.md`);
  });
});

describe('loadRecentFiles / saveRecentFiles', () => {
  it('往返一致', () => {
    const storage = memoryStorage();
    saveRecentFiles(storage, ['C:\\w\\a.md', 'C:\\w\\b.md']);
    const read = loadRecentFiles(storage);
    expect([...read.value]).toEqual(['C:\\w\\a.md', 'C:\\w\\b.md']);
    // 自己写进去的最近列表自己读得回来 —— 若这里报了丢失，告警系统本身就是坏的
    expect(read.outcome).toEqual({ ok: true });
  });

  it('没有记录时给空列表，且什么都不报', () => {
    // 从没打开过文档是常态，不是故障
    const read = loadRecentFiles(memoryStorage());
    expect([...read.value]).toEqual([]);
    expect(read.outcome).toEqual({ ok: true });
  });

  it('坏 JSON / 非数组：值仍按「没有记录」给，但要说出来', () => {
    for (const raw of ['{oops', '{"a":1}', '"a"', '42']) {
      const read = loadRecentFiles(memoryStorage(raw));
      expect([...read.value]).toEqual([]);
      expect(read.outcome).toEqual({ ok: false, reason: 'corrupt', lostKeys: [], lostCount: 1 });
    }
  });

  it('数组里的非字符串逐条丢掉，好条目保留，只报总数', () => {
    const read = loadRecentFiles(memoryStorage('[1,"a",null,""]'));
    expect([...read.value]).toEqual(['a']);
    // 这几项本身没有可读的名字（是数字不是路径），点不出名 —— 只报数，总数不是 0
    expect(read.outcome).toEqual({ ok: false, reason: 'corrupt', lostKeys: [], lostCount: 3 });
  });

  it('清空', () => {
    const storage = memoryStorage();
    saveRecentFiles(storage, ['a.md']);
    clearRecentFiles(storage);
    const read = loadRecentFiles(storage);
    expect([...read.value]).toEqual([]);
    // 清空是我们自己写的空列表，不是丢东西
    expect(read.outcome).toEqual({ ok: true });
  });

  it('写成功时返回 ok', () => {
    expect(saveRecentFiles(memoryStorage(), ['a.md'])).toEqual({ ok: true });
    expect(clearRecentFiles(memoryStorage())).toEqual({ ok: true });
  });

  it('写被拒时返回 write-failed —— 「下次启动菜单里没有它」必须有个来源（ADR 0042）', () => {
    // 之前 `clearRecentFiles` 连 try/catch 都没有：点了清空而菜单里几条还在
    const hostile: StorageLike = {
      getItem: () => null,
      setItem() {
        throw new Error('QuotaExceededError');
      },
    };
    expect(saveRecentFiles(hostile, ['a.md'])).toEqual({ ok: false, reason: 'write-failed' });
    expect(clearRecentFiles(hostile)).toEqual({ ok: false, reason: 'write-failed' });
  });

  it('没有存储时返回 no-storage：那是预期内的降级，不该说话', () => {
    expect(saveRecentFiles(null, ['a.md'])).toEqual({ ok: false, reason: 'no-storage' });
    expect(clearRecentFiles(null)).toEqual({ ok: false, reason: 'no-storage' });
  });

  it('读抛错时回落到「没有记录」，并说出「读不到」—— 不加回落层，读抛会从 App.tsx 的 useState 初始化器穿出（启动白屏）', () => {
    const hostile: StorageLike = {
      getItem() {
        throw new Error('SecurityError');
      },
      setItem: () => undefined,
    };
    const read = loadRecentFiles(hostile);
    expect([...read.value]).toEqual([]);
    expect(read.outcome).toEqual({ ok: false, reason: 'read-failed', lostKeys: [], lostCount: 1 });
  });

  it('没有存储时读侧也给 no-storage —— 与写侧同一套话术', () => {
    const read = loadRecentFiles(null);
    expect([...read.value]).toEqual([]);
    expect(read.outcome).toEqual({ ok: false, reason: 'no-storage', lostKeys: [], lostCount: 0 });
  });
});
