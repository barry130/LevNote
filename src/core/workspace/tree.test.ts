import { describe, expect, it } from 'vitest';

import type { DirEntry } from '../document/types.js';
import { ancestorsOf, compareEntries, flattenTree, isIgnoredName, visibleEntries } from './tree.js';

function dir(name: string, path = `/root/${name}`): DirEntry {
  return { name, path, isDir: true, size: 0, mtimeMs: 0 };
}
function file(name: string, path = `/root/${name}`): DirEntry {
  return { name, path, isDir: false, size: 0, mtimeMs: 0 };
}

describe('compareEntries', () => {
  it('目录排在文件前面', () => {
    expect(compareEntries(dir('z'), file('a'))).toBeLessThan(0);
  });

  it('数字按数值比较，`2` 在 `10` 前面', () => {
    // 字典序会把 10 排到 2 前面，这是文件列表里最刺眼的一类错误
    const sorted = [file('10.md'), file('2.md')].sort(compareEntries).map((item) => item.name);
    expect(sorted).toEqual(['2.md', '10.md']);
  });

  it('大小写不敏感的稳定排序', () => {
    const sorted = [file('B.md'), file('a.md')].sort(compareEntries).map((item) => item.name);
    expect(sorted).toEqual(['a.md', 'B.md']);
  });
});

describe('visibleEntries', () => {
  it('默认折叠重目录', () => {
    const entries = [file('a.md'), dir('node_modules'), dir('target'), dir('.git'), dir('src')];
    expect(visibleEntries(entries).map((item) => item.name)).toEqual(['src', 'a.md']);
  });

  it('showIgnored 时全部显示', () => {
    const entries = [dir('node_modules'), dir('src')];
    expect(visibleEntries(entries, true)).toHaveLength(2);
  });

  it('isIgnoredName 只按名字判断，不做递归扫描', () => {
    expect(isIgnoredName('node_modules')).toBe(true);
    expect(isIgnoredName('my-node_modules')).toBe(false);
    expect(isIgnoredName('src')).toBe(false);
  });
});

describe('flattenTree', () => {
  // 目录名刻意用 `empty` / `src`：字母序是 empty < src，
  // 因此「目录在前、按名字排」这条规则的输出是可预期的
  const dirs = {
    '/root': [dir('src', '/root/src'), file('a.md', '/root/a.md'), dir('empty', '/root/empty')],
    '/root/src': [file('b.ts', '/root/src/b.ts')],
    // 空数组表示「这个目录加载过，里面什么都没有」
    '/root/empty': [],
  };

  it('目录排在文件前面，目录之间按名字', () => {
    const rows = flattenTree(dirs, '/root', []);
    expect(rows.map((row) => row.name)).toEqual(['empty', 'src', 'a.md']);
    expect(rows.every((row) => row.depth === 0)).toBe(true);
  });

  it('展开后子项接在父项之后，缩进 +1', () => {
    const rows = flattenTree(dirs, '/root', ['/root/src']);
    expect(rows.map((row) => row.name)).toEqual(['empty', 'src', 'b.ts', 'a.md']);
    expect(rows[2]).toMatchObject({ name: 'b.ts', depth: 1 });
  });

  it('区分「空目录」与「还没加载」', () => {
    // 展开过（在 dirs 里，值是空数组）→ loaded 为 true，UI 就知道它是真的空
    const empty = flattenTree(dirs, '/root', ['/root/empty']).find((row) => row.name === 'empty');
    expect(empty?.loaded).toBe(true);

    // 标了展开但 dirs 里没有它 → 还在加载
    const notLoaded = flattenTree({ '/root': [dir('src', '/root/src')] }, '/root', ['/root/src']).find(
      (row) => row.name === 'src',
    );
    expect(notLoaded?.loaded).toBe(false);
  });

  it('根为 null 时没有任何行', () => {
    expect(flattenTree(dirs, null, [])).toEqual([]);
  });

  it('某一级没加载时照样输出这一行（只是没有子行）', () => {
    const rows = flattenTree({ '/root': [dir('src', '/root/src')] }, '/root', ['/root/src']);
    expect(rows.map((row) => row.name)).toEqual(['src']);
  });
});

describe('ancestorsOf', () => {
  it('给出从根到目标的祖先链（不含目标本身）', () => {
    expect(ancestorsOf('/root', '/root/a/b/c.md', '/')).toEqual(['/root/a', '/root/a/b']);
  });

  it('根的直接子项没有祖先', () => {
    expect(ancestorsOf('/root', '/root/a.md', '/')).toEqual([]);
  });

  it('不在根之内的路径返回空', () => {
    expect(ancestorsOf('/root', '/other/a.md', '/')).toEqual([]);
  });

  it('Windows 风格', () => {
    expect(ancestorsOf('C:\\w', 'C:\\w\\a\\b.md', '\\')).toEqual(['C:\\w\\a']);
  });

  it('根为 null', () => {
    expect(ancestorsOf(null, '/root/a.md', '/')).toEqual([]);
  });
});
