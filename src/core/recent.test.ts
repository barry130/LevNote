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
    expect([...loadRecentFiles(storage)]).toEqual(['C:\\w\\a.md', 'C:\\w\\b.md']);
  });

  it('没有记录、坏 JSON、非字符串数组：都按「没有记录」处理', () => {
    expect([...loadRecentFiles(memoryStorage())]).toEqual([]);
    expect([...loadRecentFiles(memoryStorage('{oops'))]).toEqual([]);
    expect([...loadRecentFiles(memoryStorage('{"a":1}'))]).toEqual([]);
    expect([...loadRecentFiles(memoryStorage('[1,"a",null,""]'))]).toEqual(['a']);
  });

  it('清空', () => {
    const storage = memoryStorage();
    saveRecentFiles(storage, ['a.md']);
    clearRecentFiles(storage);
    expect([...loadRecentFiles(storage)]).toEqual([]);
  });
});
